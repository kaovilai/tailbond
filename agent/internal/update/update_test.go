package update

import (
	"context"
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"errors"
	"fmt"
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func TestVersionOrder(t *testing.T) {
	order := []string{"0.1.0-beta.1", "0.1.0-beta.2", "0.1.0-beta.11", "0.1.0-rc.1", "v0.1.0", "0.1.1", "0.2.0", "1.0.0"}
	for i := 1; i < len(order); i++ {
		if !Newer(order[i], order[i-1]) || Newer(order[i-1], order[i]) {
			t.Errorf("%s should be newer than %s", order[i], order[i-1])
		}
	}
	if !Same("1.2.3-beta.1-r1", "1.2.3-beta.1") || !Same("0.2.0-r1", "0.2.0") {
		t.Error("ipk revision after a pre-release")
	}
	if !Same("0.1.0-r96", "v0.1.0") || !Same("0.1.0", "0.1.0+build.5") {
		t.Error("revision / build metadata should be ignored")
	}
	for _, bad := range []string{"dev", "abc1234", "0.1", "0.1.0-dirty", "v0.1.0-5-gabc123-dirty", "0.2.0-3-gabc1234", "1.x.0", ""} {
		if _, ok := Parse(bad); ok {
			t.Errorf("%q parsed", bad)
		}
	}
	if v, _ := Parse("v1.2.3-beta.4"); v.String() != "1.2.3-beta.4" {
		t.Error(v.String())
	}
}

func TestRepoPath(t *testing.T) {
	for in, want := range map[string]string{
		"":                                DefaultRepo,
		"kaovilai/tailbond":               "kaovilai/tailbond",
		"https://github.com/me/fork":      "me/fork",
		"https://github.com/me/fork.git/": "me/fork",
		" https://github.com/a_b/c-d.e ":  "a_b/c-d.e",
	} {
		if got, err := RepoPath(in); err != nil || got != want {
			t.Errorf("%q: %q %v", in, got, err)
		}
	}
	for _, bad := range []string{"https://gitlab.com/a/b", "a", "a/b/c", "a/../b x", "http://github.com/a/b/../../x/y"} {
		if got, err := RepoPath(bad); err == nil {
			t.Errorf("%q accepted as %q", bad, got)
		}
	}
}

func releaseJSON(tag string, extra map[string]any, assets ...[2]string) []byte {
	r := map[string]any{"tag_name": tag, "html_url": "https://github.com/kaovilai/tailbond/releases/tag/" + tag}
	var as []map[string]any
	for _, a := range assets {
		as = append(as, map[string]any{"name": a[0], "browser_download_url": a[1], "size": 1})
	}
	r["assets"] = as
	for k, v := range extra {
		r[k] = v
	}
	b, _ := json.Marshal(r)
	return b
}

func TestParseRelease(t *testing.T) {
	r, err := ParseRelease(releaseJSON("v0.2.0", nil, [2]string{"tailbond-0.2.0.apk", "https://x/a"}, [2]string{"../evil", "https://x/e"}, [2]string{"SHA256SUMS", "https://x/s"}))
	if err != nil || r.Version != "0.2.0" || r.Tag != "v0.2.0" || !strings.HasSuffix(r.NotesURL, "/v0.2.0") || len(r.Assets) != 2 {
		t.Fatalf("%v %+v", err, r)
	}
	if a, ok := r.FindAsset(func(n string) bool { return strings.HasSuffix(n, ".apk") }); !ok || a.URL != "https://x/a" {
		t.Fatal("find asset")
	}
	if _, ok := r.Asset("../evil"); ok {
		t.Fatal("path in an asset name kept")
	}
	for _, bad := range [][]byte{
		releaseJSON("nightly", nil),
		releaseJSON("v0.3.0", map[string]any{"prerelease": true}),
		releaseJSON("v0.3.0", map[string]any{"draft": true}),
		[]byte("not json"),
	} {
		if _, err := ParseRelease(bad); err == nil {
			t.Errorf("accepted %s", bad)
		}
	}
}

func TestParseSums(t *testing.T) {
	h := strings.Repeat("ab", 32)
	m := ParseSums([]byte(h + "  tailbond-linux-x86_64.tar.gz\n" + strings.ToUpper(h) + " *tailbond.ipk\njunk line\n" + "zz" + h[2:] + "  bad\n"))
	if len(m) != 2 || m["tailbond-linux-x86_64.tar.gz"] != h || m["tailbond.ipk"] != h {
		t.Fatalf("%v", m)
	}
}

// fakeGitHub serves /repos/<repo>/releases/latest and the assets.
func fakeGitHub(t *testing.T, files map[string][]byte, sums string) *httptest.Server {
	t.Helper()
	var srv *httptest.Server
	srv = httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		switch {
		case r.URL.Path == "/repos/kaovilai/tailbond/releases/latest":
			var as [][2]string
			for n := range files {
				as = append(as, [2]string{n, srv.URL + "/dl/" + n})
			}
			if sums != "" {
				as = append(as, [2]string{SumsFile, srv.URL + "/dl/" + SumsFile})
			}
			w.Write(releaseJSON("v0.2.0", nil, as...))
		case r.URL.Path == "/dl/"+SumsFile:
			w.Write([]byte(sums))
		case strings.HasPrefix(r.URL.Path, "/dl/"):
			if b, ok := files[strings.TrimPrefix(r.URL.Path, "/dl/")]; ok {
				w.Write(b)
				return
			}
			http.NotFound(w, r)
		default:
			http.NotFound(w, r)
		}
	}))
	t.Cleanup(srv.Close)
	return srv
}

func sum(b []byte) string { h := sha256.Sum256(b); return hex.EncodeToString(h[:]) }

func TestLatestAndDownload(t *testing.T) {
	ctx := context.Background()
	good, bad := []byte("the bundle"), []byte("tampered")
	srv := fakeGitHub(t, map[string][]byte{"a.tar.gz": good, "b.ipk": bad, "c.apk": good},
		fmt.Sprintf("%s  a.tar.gz\n%s  b.ipk\n", sum(good), sum(good)))
	c := Client{API: srv.URL}
	r, err := c.Latest(ctx, "https://github.com/kaovilai/tailbond")
	if err != nil || r.Version != "0.2.0" {
		t.Fatalf("latest: %v %+v", err, r)
	}
	dir := t.TempDir()
	if err := c.Download(ctx, r, "a.tar.gz", filepath.Join(dir, "a")); err != nil {
		t.Fatal(err)
	}
	if b, _ := os.ReadFile(filepath.Join(dir, "a")); string(b) != string(good) {
		t.Fatal("content")
	}
	// wrong checksum, no line in SHA256SUMS, unknown asset: nothing written
	for name, want := range map[string]string{"b.ipk": "doesn't match", "c.apk": "no line", "d": "has no d"} {
		err := c.Download(ctx, r, name, filepath.Join(dir, name))
		if err == nil || !strings.Contains(err.Error(), want) {
			t.Errorf("%s: %v", name, err)
		}
		if _, serr := os.Stat(filepath.Join(dir, name)); serr == nil {
			t.Errorf("%s written", name)
		}
	}
	// a release without SHA256SUMS installs nothing
	nosums := fakeGitHub(t, map[string][]byte{"a.tar.gz": good}, "")
	r2, _ := Client{API: nosums.URL}.Latest(ctx, "")
	if err := (Client{API: nosums.URL}).Download(ctx, r2, "a.tar.gz", filepath.Join(dir, "x")); err == nil || !strings.Contains(err.Error(), SumsFile) {
		t.Fatalf("no sums: %v", err)
	}
	// the real API is https: plain http asset URLs are refused
	if err := (Client{}).Download(ctx, r, "a.tar.gz", filepath.Join(dir, "y")); err == nil || !strings.Contains(err.Error(), "not https") {
		t.Fatalf("http with the real API: %v", err)
	}
	// no release (404), rate limit (403)
	empty := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) { w.WriteHeader(http.StatusForbidden) }))
	defer empty.Close()
	if _, err := (Client{API: empty.URL}).Latest(ctx, ""); err == nil || !strings.Contains(err.Error(), "rate limit") {
		t.Fatalf("403: %v", err)
	}
	if _, err := (Client{API: srv.URL}).Latest(ctx, "other/repo"); err == nil || !strings.Contains(err.Error(), "no release") {
		t.Fatalf("404: %v", err)
	}
}

// GitHub answers 404 on releases/latest both for "no release yet" and for a
// repository that doesn't exist (or is private): only the first is
// ErrNoRelease.
func TestNoReleaseVsNoRepo(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path == "/repos/kaovilai/tailbond" {
			w.Write([]byte(`{"full_name":"kaovilai/tailbond"}`))
			return
		}
		http.NotFound(w, r)
	}))
	t.Cleanup(srv.Close)
	c := Client{API: srv.URL, HTTP: srv.Client()}
	if _, _, _, err := c.LatestIfChanged(context.Background(), "kaovilai/tailbond", ""); !errors.Is(err, ErrNoRelease) {
		t.Fatalf("repo with no release: %v", err)
	}
	_, _, _, err := c.LatestIfChanged(context.Background(), "kaovilai/tailbnod", "")
	if err == nil || errors.Is(err, ErrNoRelease) || !strings.Contains(err.Error(), "not found") {
		t.Fatalf("misspelled repo: %v", err)
	}
}
