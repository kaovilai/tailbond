package update

import (
	"archive/tar"
	"compress/gzip"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"os"
	"path"
	"path/filepath"
	"slices"
	"strings"
	"time"
)

// Linux self-update (docs/architecture.md, "Applying an update"): new
// versions unpack to <state_dir>/update/versions/<v>/; the packaged
// tailbondd (the base) execs the active one at start, and rolls back a
// version that doesn't pass its trial.

const (
	TrialTimeout = 90 * time.Second
	MaxTrialBoot = 3
	// ExitRestart is the exit status asking systemd to start us again
	// (Restart=on-failure).
	ExitRestart = 75
	// EnvExeced marks a process started by the base's exec.
	EnvExeced = "TB_UPDATE_EXECED"
)

type Trial struct {
	Version       string    `json:"version"`
	Started       time.Time `json:"started"`
	Deadline      time.Time `json:"deadline"`
	Boots         int       `json:"boots"`
	ClientsBefore int       `json:"clients_before,omitempty"` // server: connected clients
	WasConnected  bool      `json:"was_connected,omitempty"`  // client: tunnel up
}

type Result struct {
	Result  string `json:"result"` // ok | rolled_back | failed
	Version string `json:"version,omitempty"`
	From    string `json:"from,omitempty"`
	At      string `json:"at"`
	Detail  string `json:"detail,omitempty"`
}

type State struct {
	Active string   `json:"active,omitempty"` // "" = the base binary
	Prev   string   `json:"prev,omitempty"`
	Trial  *Trial   `json:"trial,omitempty"`
	Failed []string `json:"failed,omitempty"`
	Last   *Result  `json:"last,omitempty"`
}

// Store is <state_dir>/update.
type Store struct{ Dir string }

func (s Store) statePath() string            { return filepath.Join(s.Dir, "state.json") }
func (s Store) VersionDir(v string) string   { return filepath.Join(s.Dir, "versions", v) }
func (s Store) Binary(v, name string) string { return filepath.Join(s.VersionDir(v), name) }
func (s Store) DownloadDir() string          { return filepath.Join(s.Dir, "download") }

func (s Store) Load() State {
	var st State
	if b, err := os.ReadFile(s.statePath()); err == nil {
		_ = json.Unmarshal(b, &st)
	}
	return st
}

func (s Store) Save(st State) error {
	if err := os.MkdirAll(s.Dir, 0o700); err != nil {
		return err
	}
	b, _ := json.MarshalIndent(st, "", "  ")
	tmp := s.statePath() + ".tmp"
	if err := os.WriteFile(tmp, b, 0o600); err != nil {
		return err
	}
	return os.Rename(tmp, s.statePath())
}

func now() string { return time.Now().UTC().Format(time.RFC3339) }

// rollback marks the active version failed and goes back to the previous.
func (st *State) rollback(detail string) {
	bad := st.Active
	if bad != "" && !slices.Contains(st.Failed, bad) {
		st.Failed = append(st.Failed, bad)
	}
	st.Last = &Result{Result: "rolled_back", Version: bad, From: st.Prev, At: now(), Detail: detail}
	st.Active, st.Prev, st.Trial = st.Prev, "", nil
}

// Boot runs in the base binary at every start: it counts trial boots,
// rolls back a version whose trial ran out, and returns the binary to
// exec ("" = run the base itself).
func (s Store) Boot(base string, t time.Time) string {
	st := s.Load()
	changed := false
	if st.Trial != nil {
		st.Trial.Boots++
		changed = true
		switch {
		case st.Trial.Boots > MaxTrialBoot:
			st.rollback(fmt.Sprintf("%s didn't start properly (%d tries)", st.Trial.Version, MaxTrialBoot))
		case t.After(st.Trial.Deadline):
			st.rollback(st.Trial.Version + " didn't pass its health check in time")
		}
	}
	exec := ""
	if st.Active != "" {
		bin := s.Binary(st.Active, "tailbondd")
		if fi, err := os.Stat(bin); err == nil && fi.Mode()&0o111 != 0 && Newer(st.Active, base) {
			exec = bin
		}
	}
	if changed {
		_ = s.Save(st)
	}
	return exec
}

// Install records an unpacked version as active, on trial.
func (s Store) Install(version string, tr Trial) error {
	st := s.Load()
	tr.Version, tr.Started = version, time.Now().UTC()
	if tr.Deadline.IsZero() {
		tr.Deadline = tr.Started.Add(TrialTimeout)
	}
	if st.Active != version {
		st.Prev = st.Active
	}
	st.Active, st.Trial = version, &tr
	if err := s.Save(st); err != nil {
		return err
	}
	s.Prune(st)
	return nil
}

// Commit ends a trial: the version stays.
func (s Store) Commit(detail string) error {
	st := s.Load()
	if st.Trial == nil {
		return nil
	}
	st.Last = &Result{Result: "ok", Version: st.Trial.Version, From: st.Prev, At: now(), Detail: detail}
	st.Trial = nil
	return s.Save(st)
}

// Rollback ends a trial the other way.
func (s Store) Rollback(detail string) error {
	st := s.Load()
	st.rollback(detail)
	return s.Save(st)
}

// Prune removes unpacked versions other than the active and previous.
func (s Store) Prune(st State) {
	ents, _ := os.ReadDir(filepath.Join(s.Dir, "versions"))
	for _, e := range ents {
		if e.Name() != st.Active && e.Name() != st.Prev {
			_ = os.RemoveAll(filepath.Join(s.Dir, "versions", e.Name()))
		}
	}
}

// MaxUnpack caps what a tarball may expand to.
const MaxUnpack = 256 << 20

// Unpack extracts tailbondd, mqvpn and hooks/* from a linux tarball (found
// anywhere in it) into versions/<v>/.
func (s Store) Unpack(tarball, version string) error {
	f, err := os.Open(tarball)
	if err != nil {
		return err
	}
	defer f.Close()
	gz, err := gzip.NewReader(f)
	if err != nil {
		return err
	}
	dst := s.VersionDir(version)
	tmp := dst + ".tmp"
	_ = os.RemoveAll(tmp)
	if err := os.MkdirAll(filepath.Join(tmp, "hooks"), 0o755); err != nil {
		return err
	}
	tr := tar.NewReader(gz)
	var total int64
	got := map[string]bool{}
	for {
		h, err := tr.Next()
		if errors.Is(err, io.EOF) {
			break
		}
		if err != nil {
			return err
		}
		if h.Typeflag != tar.TypeReg {
			continue
		}
		name := path.Clean(h.Name)
		base := path.Base(name)
		var out string
		switch {
		case base == "tailbondd" || base == "mqvpn":
			out = filepath.Join(tmp, base)
		case path.Base(path.Dir(name)) == "hooks" && !strings.HasPrefix(base, "."):
			out = filepath.Join(tmp, "hooks", base)
		default:
			continue
		}
		total += h.Size
		if total > MaxUnpack {
			return errors.New("update archive too large")
		}
		w, err := os.OpenFile(out, os.O_CREATE|os.O_TRUNC|os.O_WRONLY, 0o755)
		if err != nil {
			return err
		}
		_, err = io.Copy(w, io.LimitReader(tr, h.Size))
		if cerr := w.Close(); err == nil {
			err = cerr
		}
		if err != nil {
			return err
		}
		got[base] = true
	}
	if !got["tailbondd"] {
		_ = os.RemoveAll(tmp)
		return errors.New("update archive has no tailbondd")
	}
	_ = os.RemoveAll(dst)
	return os.Rename(tmp, dst)
}
