package agent

import (
	"context"
	"crypto/sha256"
	"encoding/hex"
	"fmt"
	"io"
	"net/http"
	"os"
	"path/filepath"
	"sync"
	"time"

	"github.com/kaovilai/tailbond/agent/internal/config"
	"github.com/kaovilai/tailbond/agent/internal/update"
)

// Downloads the phone page ("Get tailbond") offers besides the APK.
//
// The macOS bundle (tailbond-macos-arm64.tar.gz, docs/macos.md) is built
// on a Mac, so the Linux packages don't carry it. The server serves, in
// order: macos_bundle / DefaultMacOSBundle when an admin put it there; a
// copy cached in <state_dir>/downloads/; else, on the first request, the
// one from this version's GitHub release (or the latest, for a build that
// isn't a release), checked against the release's SHA256SUMS like an
// update, and cached. With update.policy "off" it never asks GitHub.

const macBundleName = "tailbond-macos-arm64.tar.gz"

// macFetchRetry: after a failed fetch, the next try waits this long (a
// page view or a click can't make the server hammer GitHub).
const macFetchRetry = 10 * time.Minute

type dlState struct {
	mu       sync.Mutex
	sums     map[string]fileSum // path -> checksum, until the file changes
	fetching chan struct{}      // closed when the running fetch ends
	failedAt time.Time
	lastErr  string
}

type fileSum struct{ key, sum string }

// sumOf is a file's SHA-256 (hex), cached until its size or mtime changes;
// "" when it can't be read.
func (a *Agent) sumOf(path string, st os.FileInfo) string {
	key := fmt.Sprintf("%d|%d", st.Size(), st.ModTime().UnixNano())
	a.dl.mu.Lock()
	if c, ok := a.dl.sums[path]; ok && c.key == key {
		a.dl.mu.Unlock()
		return c.sum
	}
	a.dl.mu.Unlock()
	f, err := os.Open(path)
	if err != nil {
		return ""
	}
	defer f.Close()
	h := sha256.New()
	if _, err := io.Copy(h, f); err != nil {
		return ""
	}
	sum := hex.EncodeToString(h.Sum(nil))
	a.dl.mu.Lock()
	if a.dl.sums == nil {
		a.dl.sums = map[string]fileSum{}
	}
	a.dl.sums[path] = fileSum{key, sum}
	a.dl.mu.Unlock()
	return sum
}

func (a *Agent) macCachePath() string {
	return filepath.Join(a.store.Get().StateDir, "downloads", macBundleName)
}

// macBundle: the bundle to serve right now and its checksum ("" path when
// there is none yet).
func (a *Agent) macBundle() (path, sum string, size int64) {
	for _, p := range []string{a.store.Get().MacOSBundlePath(), a.macCachePath()} {
		if st, err := os.Stat(p); err == nil && !st.IsDir() && st.Size() > 0 {
			return p, a.sumOf(p, st), st.Size()
		}
	}
	return "", "", 0
}

// macFetchAllowed: GitHub may be asked (not with updates off, not right
// after a failure).
func (a *Agent) macFetchAllowed() bool {
	if a.store.Get().UpdateOrDefault().Policy == config.UpdateOff {
		return false
	}
	a.dl.mu.Lock()
	defer a.dl.mu.Unlock()
	return a.dl.failedAt.IsZero() || time.Since(a.dl.failedAt) > macFetchRetry
}

// fetchMacBundle downloads the release's bundle into the cache; concurrent
// callers wait for the one fetch.
func (a *Agent) fetchMacBundle(ctx context.Context) error {
	a.dl.mu.Lock()
	if ch := a.dl.fetching; ch != nil {
		a.dl.mu.Unlock()
		select {
		case <-ch:
		case <-ctx.Done():
			return ctx.Err()
		}
		if p, _, _ := a.macBundle(); p != "" {
			return nil
		}
		a.dl.mu.Lock()
		defer a.dl.mu.Unlock()
		return fmt.Errorf("%s", a.dl.lastErr)
	}
	ch := make(chan struct{})
	a.dl.fetching = ch
	a.dl.mu.Unlock()

	err := a.fetchMacBundleOnce(ctx)
	a.dl.mu.Lock()
	a.dl.fetching = nil
	if err != nil {
		a.dl.failedAt, a.dl.lastErr = time.Now(), err.Error()
	} else {
		a.dl.failedAt, a.dl.lastErr = time.Time{}, ""
	}
	a.dl.mu.Unlock()
	close(ch)
	return err
}

func (a *Agent) fetchMacBundleOnce(ctx context.Context) error {
	ctx, cancel := context.WithTimeout(ctx, 2*time.Minute)
	defer cancel()
	c, repo := a.updateClient(), a.updateRepo()
	var rel *update.Release
	var err error
	if isRelease(a.opts.Version) {
		// this server's own version: the Mac matches it
		rel, err = c.Tag(ctx, repo, "v"+a.opts.Version)
	}
	if rel == nil {
		rel, err = c.Latest(ctx, repo)
	}
	if err != nil {
		return err
	}
	if err := c.Download(ctx, rel, macBundleName, a.macCachePath()); err != nil {
		return err
	}
	a.Logf("phone page: cached %s %s from GitHub for Macs", macBundleName, rel.Version)
	return nil
}

// prefetchMacBundle starts a fetch in the background when there is no
// bundle yet, so it's ready (with its checksum on the page) by the time a
// Mac user taps Download.
func (a *Agent) prefetchMacBundle() {
	if p, _, _ := a.macBundle(); p != "" || !a.macFetchAllowed() {
		return
	}
	go func() {
		if err := a.fetchMacBundle(context.Background()); err != nil {
			a.Logf("phone page: macOS bundle not available: %v", err)
		}
	}()
}

// hMacBundle: GET /tailbond-macos-arm64.tar.gz.
func (a *Agent) hMacBundle(w http.ResponseWriter, r *http.Request) {
	p, _, _ := a.macBundle()
	if p == "" && a.macFetchAllowed() {
		_ = a.fetchMacBundle(r.Context())
		p, _, _ = a.macBundle()
	}
	if p == "" {
		http.Error(w, "The Mac download isn't on this server yet: get tailbond-macos-arm64.tar.gz from the project's GitHub releases.", http.StatusNotFound)
		return
	}
	serveDownload(w, r, p, macBundleName, "application/gzip")
}
