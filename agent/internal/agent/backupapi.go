package agent

import (
	"bytes"
	"context"
	"encoding/base64"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"net"
	"net/http"
	"net/netip"
	"os"
	"os/exec"
	"path/filepath"
	"regexp"
	"sort"
	"strings"
	"time"

	"github.com/tailbond/tailbond/agent/internal/backup"
	"github.com/tailbond/tailbond/agent/internal/config"
	"github.com/tailbond/tailbond/agent/internal/status"
	"github.com/tailbond/tailbond/agent/internal/tailscale"
)

// Encrypted backups and restore (docs/architecture.md, "Backup and restore").

const maxBackupBody = 4 << 20 // base64 of a 2 MiB backup, with room

func isTailnet(ip netip.Addr) bool { return tailscale.IsTailnetIP(ip) }

func hostPort(host, port string) string { return net.JoinHostPort(host, port) }

func jsonDecode(r io.Reader, out any) error {
	return json.NewDecoder(io.LimitReader(r, 64<<10)).Decode(out)
}

func readJSONMax(r *http.Request, v any, max int64) error {
	if err := json.NewDecoder(io.LimitReader(r.Body, max)).Decode(v); err != nil {
		return fmt.Errorf("bad JSON body: %w", err)
	}
	return nil
}

func (a *Agent) backupDir() string { return filepath.Join(a.store.Get().StateDir, "backups") }

func (a *Agent) clientBackupDir() string {
	return filepath.Join(a.store.Get().StateDir, "client-backups")
}

func (a *Agent) uciExport() string {
	if !isOpenWrt() {
		return ""
	}
	out, err := exec.Command("uci", "export", "tailbond").Output()
	if err != nil {
		return ""
	}
	return string(out)
}

func (a *Agent) backupHeader() backup.Header {
	host, _ := os.Hostname()
	if n := a.serverName(); a.role == config.RoleServer && n != "" {
		host = n
	}
	plat := "linux"
	if isOpenWrt() {
		plat = "openwrt"
	} else {
		if _, err := os.Stat("/etc/fedora-release"); err == nil {
			plat = "fedora"
		}
	}
	return backup.Header{Role: a.role, Platform: plat, Host: host, AgentVersion: a.opts.Version}
}

// MakeBackup seals this machine's backup (also used by the CLI).
func MakeBackup(cfgPath, stateDir, uci, pass string, h backup.Header, cur *config.Config) ([]byte, error) {
	p, err := backup.Collect(cfgPath, stateDir, uci, cur)
	if err != nil {
		return nil, err
	}
	pt, err := p.Encode()
	if err != nil {
		return nil, err
	}
	return backup.Seal(pt, pass, h)
}

func (a *Agent) makeBackup(pass string) ([]byte, error) {
	return MakeBackup(a.store.Path(), a.store.Get().StateDir, a.uciExport(), pass, a.backupHeader(), a.store.Get())
}

func (a *Agent) hBackup(w http.ResponseWriter, r *http.Request) {
	var body struct {
		Passphrase string `json:"passphrase"`
	}
	if err := readJSON(r, &body); err != nil {
		httpErr(w, http.StatusBadRequest, err)
		return
	}
	data, err := a.makeBackup(body.Passphrase)
	if err != nil {
		httpErr(w, http.StatusBadRequest, err)
		return
	}
	h := a.backupHeader()
	name := fmt.Sprintf("tailbond-%s-%s.tbbackup", safeName(h.Host), time.Now().Format("20060102-1504"))
	w.Header().Set("Content-Type", "application/octet-stream")
	w.Header().Set("Content-Disposition", `attachment; filename="`+name+`"`)
	_, _ = w.Write(data)
}

var reSafe = regexp.MustCompile(`[^A-Za-z0-9._-]+`)

func safeName(s string) string {
	s = reSafe.ReplaceAllString(s, "-")
	if s == "" {
		return "tailbond"
	}
	return s
}

func (a *Agent) hBackupSummary(w http.ResponseWriter, r *http.Request) {
	p, err := backup.Collect(a.store.Path(), a.store.Get().StateDir, a.uciExport(), a.store.Get())
	if err != nil {
		httpErr(w, http.StatusInternalServerError, err)
		return
	}
	writeJSON(w, http.StatusOK, backup.Summary(a.store.Get(), p))
}

type restoreBody struct {
	File       string `json:"file"` // base64 of the .tbbackup
	Passphrase string `json:"passphrase"`
	Confirm    bool   `json:"confirm"`
}

// openRestore decrypts and checks a backup for this machine.
func (a *Agent) openRestore(b restoreBody) (backup.Header, *backup.Payload, *config.Config, error) {
	data, err := base64.StdEncoding.DecodeString(strings.TrimSpace(b.File))
	if err != nil {
		data = []byte(b.File)
	}
	h, pt, err := backup.Open(data, b.Passphrase)
	if err != nil {
		return h, nil, nil, err
	}
	p, cfg, err := backup.Decode(pt)
	if err != nil {
		return h, nil, nil, err
	}
	if h.Role != a.role || cfg.Role != a.role {
		return h, nil, nil, fmt.Errorf("this is a %s backup; this machine is a %s", h.Role, a.role)
	}
	return h, p, cfg, nil
}

func (a *Agent) restoreWarnings(h backup.Header, next *config.Config) []string {
	var out []string
	cur := a.store.Get()
	host, _ := os.Hostname()
	if a.role == config.RoleServer {
		if h.Host != "" && h.Host != a.serverName() && h.Host != host {
			out = append(out, fmt.Sprintf("The backup is from %s. Devices keep their pairing (same keys); if this machine's public address differs, set it after restoring: devices then pick up the new address from the server's Tailscale name by themselves, others need their link again.", h.Host))
		}
		if next.Server.PublicAddr != "" && next.Server.PublicAddr != cur.Server.PublicAddr {
			out = append(out, "Public address in the backup: "+next.Server.PublicAddr+". Change it under Settings if this machine is reached differently.")
		}
	}
	if h.Platform != "" && h.Platform != a.backupHeader().Platform {
		out = append(out, fmt.Sprintf("Made on %s, restoring on %s: paths that belong to this machine (programs, hooks, state directory) are kept.", h.Platform, a.backupHeader().Platform))
	}
	return out
}

func (a *Agent) hRestorePreview(w http.ResponseWriter, r *http.Request) {
	var b restoreBody
	if err := readJSONMax(r, &b, maxBackupBody); err != nil {
		httpErr(w, http.StatusBadRequest, err)
		return
	}
	h, _, cfg, err := a.openRestore(b)
	if err != nil {
		httpErr(w, http.StatusBadRequest, err)
		return
	}
	writeJSON(w, http.StatusOK, map[string]any{
		"role": h.Role, "host": h.Host, "platform": h.Platform, "created": h.Created, "agent_version": h.AgentVersion,
		"changes": backup.Changes(a.store.Get(), cfg), "warnings": a.restoreWarnings(h, cfg),
	})
}

func (a *Agent) hRestore(w http.ResponseWriter, r *http.Request) {
	var b restoreBody
	if err := readJSONMax(r, &b, maxBackupBody); err != nil {
		httpErr(w, http.StatusBadRequest, err)
		return
	}
	if !b.Confirm {
		httpErr(w, http.StatusBadRequest, errors.New(`send "confirm": true to restore`))
		return
	}
	_, p, cfg, err := a.openRestore(b)
	if err != nil {
		httpErr(w, http.StatusBadRequest, err)
		return
	}
	rollback, err := RestoreBackup(a.store, p, cfg, b.Passphrase, a.uciExport(), a.backupHeader(), isOpenWrt())
	if err != nil {
		httpErr(w, http.StatusInternalServerError, err)
		return
	}
	a.Logf("restored a backup; the previous settings are in %s; restarting", rollback)
	writeJSON(w, http.StatusOK, map[string]any{"ok": true, "rollback": filepath.Base(rollback), "restarting": true})
	go func() {
		time.Sleep(500 * time.Millisecond)
		a.requestRestart()
	}()
}

// RestoreBackup replaces the config and state files with a backup's,
// keeping this machine's own paths, after saving an encrypted copy of the
// current settings. Returns that copy's path. Used by the API and the CLI.
func RestoreBackup(store *config.Store, p *backup.Payload, next *config.Config, pass, curUCI string, h backup.Header, openwrt bool) (string, error) {
	cur := store.Get()
	dir := filepath.Join(cur.StateDir, "backups")
	if err := os.MkdirAll(dir, 0o700); err != nil {
		return "", err
	}
	old, err := MakeBackup(store.Path(), cur.StateDir, curUCI, pass, h, cur)
	if err != nil {
		return "", fmt.Errorf("saving the current settings first: %w", err)
	}
	rollback := filepath.Join(dir, "before-restore-"+time.Now().Format("20060102-150405")+".tbbackup")
	if err := os.WriteFile(rollback, old, 0o600); err != nil {
		return "", err
	}
	if _, err := store.Update(func(c *config.Config) error {
		keep := *c
		*c = *next
		// this machine's own paths and binding
		c.MqvpnBin, c.StateDir, c.HooksDir, c.Listen, c.ControlAddr = keep.MqvpnBin, keep.StateDir, keep.HooksDir, keep.Listen, keep.ControlAddr
		c.AndroidAPK, c.GLiNetIPK = keep.AndroidAPK, keep.GLiNetIPK
		return nil
	}); err != nil {
		return "", err
	}
	if err := backup.Apply(p, cur.StateDir); err != nil {
		return "", err
	}
	if openwrt && p.UCI != "" {
		cmd := exec.Command("uci", "import", "tailbond")
		cmd.Stdin = strings.NewReader(p.UCI)
		if out, err := cmd.CombinedOutput(); err != nil {
			return rollback, fmt.Errorf("uci import: %v: %s", err, out)
		}
		_ = exec.Command("uci", "commit", "tailbond").Run()
	}
	return rollback, nil
}

type backupEntry struct {
	Name    string `json:"name"`
	Client  string `json:"client,omitempty"`
	Size    int64  `json:"size"`
	Created string `json:"created"`
}

func listBackups(dir, client string) []backupEntry {
	var out []backupEntry
	ents, _ := os.ReadDir(dir)
	for _, e := range ents {
		if !strings.HasSuffix(e.Name(), ".tbbackup") {
			continue
		}
		info, err := e.Info()
		if err != nil {
			continue
		}
		out = append(out, backupEntry{Name: e.Name(), Client: client, Size: info.Size(), Created: info.ModTime().UTC().Format(time.RFC3339)})
	}
	return out
}

func (a *Agent) hBackups(w http.ResponseWriter, r *http.Request) {
	out := listBackups(a.backupDir(), "")
	if ents, err := os.ReadDir(a.clientBackupDir()); err == nil {
		for _, e := range ents {
			if e.IsDir() {
				out = append(out, listBackups(filepath.Join(a.clientBackupDir(), e.Name()), e.Name())...)
			}
		}
	}
	if out == nil {
		out = []backupEntry{}
	}
	sort.Slice(out, func(i, j int) bool { return out[i].Created > out[j].Created })
	writeJSON(w, http.StatusOK, out)
}

func (a *Agent) hBackupDownload(w http.ResponseWriter, r *http.Request) {
	name := r.PathValue("name")
	client := r.URL.Query().Get("client")
	dir := a.backupDir()
	if client != "" {
		if safeName(client) != client {
			http.NotFound(w, r)
			return
		}
		dir = filepath.Join(a.clientBackupDir(), client)
	}
	if safeName(name) != name || !strings.HasSuffix(name, ".tbbackup") {
		http.NotFound(w, r)
		return
	}
	serveDownload(w, r, filepath.Join(dir, name), name, "application/octet-stream")
}

// hBackupPush: a paired client's own encrypted backup (opaque here).
func (a *Agent) hBackupPush(w http.ResponseWriter, r *http.Request) {
	if a.role != config.RoleServer {
		httpErr(w, http.StatusBadRequest, errServerOnly)
		return
	}
	client := a.clientForToken(r)
	if client == "" {
		httpErr(w, http.StatusUnauthorized, errors.New("client token required"))
		return
	}
	a.noteClientVersion(r, client)
	data, err := io.ReadAll(io.LimitReader(r.Body, 1<<20+1))
	if err != nil || len(data) > 1<<20 {
		httpErr(w, http.StatusRequestEntityTooLarge, errors.New("at most 1 MiB"))
		return
	}
	if _, _, _, err := backup.ReadHeader(data); err != nil {
		httpErr(w, http.StatusBadRequest, err)
		return
	}
	dir := filepath.Join(a.clientBackupDir(), safeName(client))
	if err := os.MkdirAll(dir, 0o700); err != nil {
		httpErr(w, http.StatusInternalServerError, err)
		return
	}
	name := time.Now().UTC().Format("20060102-150405") + ".tbbackup"
	if err := os.WriteFile(filepath.Join(dir, name), data, 0o600); err != nil {
		httpErr(w, http.StatusInternalServerError, err)
		return
	}
	pruneBackups(dir, "", 3)
	writeJSON(w, http.StatusOK, map[string]any{"ok": true, "name": name})
}

// pruneBackups keeps the newest keep files starting with prefix.
func pruneBackups(dir, prefix string, keep int) {
	ents, _ := os.ReadDir(dir)
	var names []string
	for _, e := range ents {
		if strings.HasPrefix(e.Name(), prefix) && strings.HasSuffix(e.Name(), ".tbbackup") && !strings.HasPrefix(e.Name(), "before-restore-") {
			names = append(names, e.Name())
		}
	}
	sort.Strings(names) // timestamps sort
	for len(names) > keep {
		_ = os.Remove(filepath.Join(dir, names[0]))
		names = names[1:]
	}
}

func (a *Agent) hBackupSettings(w http.ResponseWriter, r *http.Request) {
	var body struct {
		Passphrase *string `json:"passphrase"`
		EveryHours *int    `json:"every_hours"`
		Keep       *int    `json:"keep"`
		Push       *bool   `json:"push"`
	}
	if err := readJSON(r, &body); err != nil {
		httpErr(w, http.StatusBadRequest, err)
		return
	}
	if body.Passphrase != nil && *body.Passphrase != "" && len([]rune(*body.Passphrase)) < backup.MinPassphrase {
		httpErr(w, http.StatusBadRequest, backup.ErrShort)
		return
	}
	if _, err := a.store.Update(func(c *config.Config) error {
		b := config.BackupSettings{}
		if c.Backup != nil {
			b = *c.Backup
		}
		if body.Passphrase != nil {
			b.Passphrase = *body.Passphrase
		}
		if body.EveryHours != nil {
			b.EveryHours = *body.EveryHours
		}
		if body.Keep != nil {
			b.Keep = *body.Keep
		}
		if body.Push != nil {
			b.Push = body.Push
		}
		c.Backup = &b
		return nil
	}); err != nil {
		httpErr(w, http.StatusBadRequest, err)
		return
	}
	b := a.store.Get().BackupOrDefault()
	writeJSON(w, http.StatusOK, map[string]any{"ok": true, "scheduled": b.Passphrase != "", "every_hours": b.EveryHours, "keep": b.Keep, "push": *b.Push})
}

// backupLoop makes the scheduled backups (when a passphrase is set) and,
// on a client, sends the newest to its server.
func (a *Agent) backupLoop(ctx context.Context) {
	t := time.NewTicker(10 * time.Minute)
	defer t.Stop()
	if a.client != nil {
		go a.refreshPairings(ctx.Done())
	}
	for {
		select {
		case <-ctx.Done():
			return
		case <-t.C:
		}
		a.scheduledBackup(ctx)
	}
}

func (a *Agent) scheduledBackup(ctx context.Context) {
	b := a.store.Get().BackupOrDefault()
	if b.Passphrase == "" {
		return
	}
	dir := a.backupDir()
	prefix := safeName(a.backupHeader().Host) + "-"
	newest := time.Time{}
	for _, e := range listBackups(dir, "") {
		if strings.HasPrefix(e.Name, prefix) {
			if t, err := time.Parse(time.RFC3339, e.Created); err == nil && t.After(newest) {
				newest = t
			}
		}
	}
	if time.Since(newest) < time.Duration(b.EveryHours)*time.Hour {
		return
	}
	data, err := a.makeBackup(b.Passphrase)
	if err != nil {
		a.Logf("scheduled backup: %v", err)
		a.noteBackup(func(s *backupStatus) { s.LastError = err.Error() })
		return
	}
	_ = os.MkdirAll(dir, 0o700)
	name := prefix + time.Now().UTC().Format("20060102-150405") + ".tbbackup"
	if err := os.WriteFile(filepath.Join(dir, name), data, 0o600); err != nil {
		a.Logf("scheduled backup: %v", err)
		return
	}
	pruneBackups(dir, prefix, b.Keep)
	a.Logf("scheduled backup %s", name)
	a.noteBackup(func(s *backupStatus) { s.Last, s.LastError = time.Now().UTC().Format(time.RFC3339), "" })
	if a.client != nil && *b.Push && a.Snapshot().State == status.StateConnected {
		if err := a.pushBackup(ctx, data); err != nil {
			a.Logf("sending the backup to the server: %v", err)
			a.noteBackup(func(s *backupStatus) { s.LastError = "sending to the server: " + err.Error() })
		} else {
			a.noteBackup(func(s *backupStatus) { s.LastPush = time.Now().UTC().Format(time.RFC3339) })
		}
	}
}

func (a *Agent) pushBackup(ctx context.Context, data []byte) error {
	base := a.serverAPIBase()
	if base == "" {
		return errors.New("not connected")
	}
	url := base + "/api/backups/push"
	req, err := http.NewRequestWithContext(ctx, http.MethodPost, url, bytes.NewReader(data))
	if err != nil {
		return err
	}
	req.Header = a.clientHeaders()
	req.Header.Set("Content-Type", "application/octet-stream")
	resp, err := httpClient.Do(req)
	if err != nil {
		return err
	}
	defer resp.Body.Close()
	if resp.StatusCode != http.StatusOK {
		return fmt.Errorf("HTTP %d", resp.StatusCode)
	}
	return nil
}

// backupStatus is what GET /api/settings reports as "backup".
type backupStatus struct {
	PassphraseSet bool   `json:"passphrase_set"`
	EveryHours    int    `json:"every_hours"`
	Keep          int    `json:"keep"`
	Push          bool   `json:"push"`
	Last          string `json:"last,omitempty"`
	LastPush      string `json:"last_push,omitempty"`
	LastError     string `json:"last_error,omitempty"`
}

func (a *Agent) noteBackup(fn func(s *backupStatus)) {
	a.upd.mu.Lock()
	fn(&a.upd.backup)
	a.upd.mu.Unlock()
}

func (a *Agent) backupView() *backupStatus {
	b := a.store.Get().BackupOrDefault()
	a.upd.mu.Lock()
	v := a.upd.backup
	a.upd.mu.Unlock()
	v.PassphraseSet, v.EveryHours, v.Keep, v.Push = b.Passphrase != "", b.EveryHours, b.Keep, *b.Push
	if v.Last == "" {
		for _, e := range listBackups(a.backupDir(), "") {
			if !strings.HasPrefix(e.Name, "before-restore-") && e.Created > v.Last {
				v.Last = e.Created
			}
		}
	}
	return &v
}
