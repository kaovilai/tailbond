package update

import (
	"bytes"
	"context"
	"crypto/ed25519"
	"errors"
	"fmt"
	"io"
	"net/http"
	"os"
	"path/filepath"
	"sort"
	"strings"
	"time"
)

// Mirror is the server's cache of a channel: <dir>/<channel>.json(.sig),
// <dir>/<channel>.prev.json (the previous version, keep-last-2) and
// <dir>/files/<artifact>.
type Mirror struct {
	Dir      string
	MaxBytes int64
}

func (m Mirror) FilesDir() string { return filepath.Join(m.Dir, "files") }

// FilePath is the cached artifact name ("" if the name isn't a plain file name).
func (m Mirror) FilePath(name string) string {
	if name == "" || strings.ContainsAny(name, "/\\") || strings.HasPrefix(name, ".") {
		return ""
	}
	return filepath.Join(m.FilesDir(), name)
}

func (m Mirror) manifestPath(channel string) string {
	return filepath.Join(m.Dir, channel+".json")
}

// Manifest returns the cached raw manifest and signature of a channel.
func (m Mirror) Manifest(channel string) (manifest, sig []byte, ok bool) {
	mb, err1 := os.ReadFile(m.manifestPath(channel))
	sb, err2 := os.ReadFile(m.manifestPath(channel) + ".sig")
	return mb, sb, err1 == nil && err2 == nil
}

// Store keeps a verified manifest; the one it replaces becomes .prev.
func (m Mirror) Store(channel string, manifest, sig []byte) error {
	if err := os.MkdirAll(m.FilesDir(), 0o755); err != nil {
		return err
	}
	if old, oldSig, ok := m.Manifest(channel); ok && string(old) != string(manifest) {
		_ = os.WriteFile(filepath.Join(m.Dir, channel+".prev.json"), old, 0o644)
		_ = os.WriteFile(filepath.Join(m.Dir, channel+".prev.json.sig"), oldSig, 0o644)
	}
	if err := writeAtomic(m.manifestPath(channel)+".sig", sig); err != nil {
		return err
	}
	return writeAtomic(m.manifestPath(channel), manifest)
}

// SetOrigin records where the cached manifest came from ("public",
// "upload"); Origin reads it.
func (m Mirror) SetOrigin(channel, origin string) {
	_ = os.WriteFile(filepath.Join(m.Dir, channel+".origin"), []byte(origin), 0o644)
}

func (m Mirror) Origin(channel string) string {
	b, _ := os.ReadFile(filepath.Join(m.Dir, channel+".origin"))
	return strings.TrimSpace(string(b))
}

// Prune keeps the artifacts of the given manifests (current, previous),
// drops the rest, and then drops the previous version's artifacts too if
// the cache is over MaxBytes.
func (m Mirror) Prune(current, previous []*Manifest) {
	keep := map[string]int{} // name -> 0 current, 1 previous
	for _, mf := range previous {
		if mf != nil {
			for _, a := range mf.Artifacts {
				keep[a.Name] = 1
			}
		}
	}
	for _, mf := range current {
		if mf != nil {
			for _, a := range mf.Artifacts {
				keep[a.Name] = 0
			}
		}
	}
	ents, _ := os.ReadDir(m.FilesDir())
	type file struct {
		name string
		size int64
		prev bool
	}
	var files []file
	var total int64
	for _, e := range ents {
		k, ok := keep[e.Name()]
		if !ok {
			_ = os.Remove(filepath.Join(m.FilesDir(), e.Name()))
			continue
		}
		info, err := e.Info()
		if err != nil {
			continue
		}
		files = append(files, file{e.Name(), info.Size(), k == 1})
		total += info.Size()
	}
	if m.MaxBytes <= 0 || total <= m.MaxBytes {
		return
	}
	sort.Slice(files, func(i, j int) bool { return files[i].prev && !files[j].prev })
	for _, f := range files {
		if total <= m.MaxBytes {
			break
		}
		_ = os.Remove(filepath.Join(m.FilesDir(), f.name))
		total -= f.size
	}
}

// Size is the cache's total artifact size.
func (m Mirror) Size() int64 {
	var total int64
	ents, _ := os.ReadDir(m.FilesDir())
	for _, e := range ents {
		if info, err := e.Info(); err == nil {
			total += info.Size()
		}
	}
	return total
}

func writeAtomic(p string, b []byte) error {
	tmp := p + ".tmp"
	if err := os.WriteFile(tmp, b, 0o644); err != nil {
		return err
	}
	return os.Rename(tmp, p)
}

// Source is where a client looks for the manifest and artifacts: the home
// server's mirror (Mirror true, files under /updates/files/) or the public
// update source (artifact URLs from the manifest).
type Source struct {
	Base   string // ".../updates" for a mirror, the update.source URL otherwise
	Mirror bool
	Header http.Header
}

// Fetcher downloads with a bounded client.
type Fetcher struct {
	Client  *http.Client
	MaxSize int64 // per download; 0 = 512 MiB
}

func (f *Fetcher) client() *http.Client {
	if f.Client != nil {
		return f.Client
	}
	return &http.Client{Timeout: 10 * time.Minute}
}

func (f *Fetcher) get(ctx context.Context, url string, h http.Header, w io.Writer, max int64) error {
	ctx, cancel := context.WithTimeout(ctx, 10*time.Minute)
	defer cancel()
	req, err := http.NewRequestWithContext(ctx, http.MethodGet, url, nil)
	if err != nil {
		return err
	}
	for k, v := range h {
		req.Header[k] = v
	}
	resp, err := f.client().Do(req)
	if err != nil {
		return err
	}
	defer resp.Body.Close()
	if resp.StatusCode != http.StatusOK {
		return fmt.Errorf("%s: HTTP %d", url, resp.StatusCode)
	}
	n, err := io.Copy(w, io.LimitReader(resp.Body, max+1))
	if err == nil && n > max {
		err = errors.New("download too large")
	}
	return err
}

func (f *Fetcher) getBytes(ctx context.Context, url string, h http.Header, max int64) ([]byte, error) {
	var b bytes.Buffer
	if err := f.get(ctx, url, h, &b, max); err != nil {
		return nil, err
	}
	return b.Bytes(), nil
}

// Fetched is a verified manifest and where it came from.
type Fetched struct {
	Manifest *Manifest
	Raw, Sig []byte
	From     Source
}

// Manifest tries the sources in order and returns the first manifest that
// verifies (errors from all of them otherwise).
func (f *Fetcher) Manifest(ctx context.Context, sources []Source, channel string, keys []ed25519.PublicKey) (*Fetched, error) {
	var errs []string
	for _, s := range sources {
		if s.Base == "" {
			continue
		}
		base := strings.TrimRight(s.Base, "/") + "/" + channel + ".json"
		raw, err := f.getBytes(ctx, base, s.Header, 1<<20)
		if err == nil {
			var sig []byte
			if sig, err = f.getBytes(ctx, base+".sig", s.Header, 4096); err == nil {
				var m *Manifest
				if m, err = Verify(raw, sig, keys); err == nil {
					return &Fetched{Manifest: m, Raw: raw, Sig: sig, From: s}, nil
				}
			}
		}
		errs = append(errs, err.Error())
	}
	if len(errs) == 0 {
		return nil, errors.New("no update source configured")
	}
	return nil, errors.New(strings.Join(errs, "; "))
}

// Download fetches an artifact (mirror first) into dst and verifies it.
func (f *Fetcher) Download(ctx context.Context, sources []Source, a Artifact, dst string) (Source, error) {
	max := f.MaxSize
	if max <= 0 {
		max = 512 << 20
	}
	if a.Size > 0 && a.Size < max {
		max = a.Size
	}
	if err := os.MkdirAll(filepath.Dir(dst), 0o700); err != nil {
		return Source{}, err
	}
	var errs []string
	try := func(s Source, url string) bool {
		tmp := dst + ".part"
		w, err := os.Create(tmp)
		if err != nil {
			errs = append(errs, err.Error())
			return false
		}
		err = f.get(ctx, url, s.Header, w, max)
		if cerr := w.Close(); err == nil {
			err = cerr
		}
		if err == nil {
			err = CheckFile(tmp, a)
		}
		if err == nil {
			err = os.Rename(tmp, dst)
		}
		if err != nil {
			_ = os.Remove(tmp)
			errs = append(errs, err.Error())
			return false
		}
		return true
	}
	for _, s := range sources {
		switch {
		case s.Mirror && s.Base != "":
			if try(s, strings.TrimRight(s.Base, "/")+"/files/"+a.Name) {
				return s, nil
			}
		case !s.Mirror && a.URL != "":
			if try(s, a.URL) {
				return s, nil
			}
		}
	}
	return Source{}, fmt.Errorf("download %s: %s", a.Name, strings.Join(errs, "; "))
}
