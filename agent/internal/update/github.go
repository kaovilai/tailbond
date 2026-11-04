package update

import (
	"bufio"
	"bytes"
	"context"
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"os"
	"path/filepath"
	"regexp"
	"strings"
	"time"
)

// SumsFile is the release asset with every asset's SHA-256 (sha256sum format).
const SumsFile = "SHA256SUMS"

var repoRE = regexp.MustCompile(`^[A-Za-z0-9_.-]+/[A-Za-z0-9_.-]+$`)

// RepoPath reads "owner/name" or "https://github.com/owner/name[.git][/]".
func RepoPath(s string) (string, error) {
	s = strings.TrimSpace(s)
	if s == "" {
		s = BuildRepo
	}
	if s == "" {
		s = DefaultRepo
	}
	if strings.Contains(s, "://") {
		u, err := url.Parse(s)
		if err != nil || u.Host != "github.com" {
			return "", fmt.Errorf("repo %q: owner/name or https://github.com/owner/name", s)
		}
		s = strings.TrimSuffix(strings.Trim(u.Path, "/"), ".git")
	}
	if !repoRE.MatchString(s) {
		return "", fmt.Errorf("repo %q: owner/name or https://github.com/owner/name", s)
	}
	return s, nil
}

// Release is what the update check needs from a GitHub release.
type Release struct {
	Version  string // tag without the leading v
	Tag      string
	NotesURL string // the release page (changelog)
	Assets   []Asset
	// Security: the release notes carry SecurityMarker (the release job
	// adds it when a commit has a "Security: yes" trailer; a maintainer can
	// add it by editing the notes). "Right away" installs those without
	// waiting for a quiet moment.
	Security bool
}

// SecurityMarker in a release's name or notes flags a security release.
const SecurityMarker = "[security]"

type Asset struct {
	Name string
	URL  string
	Size int64
}

// ParseRelease reads a GitHub API release object. Drafts, pre-releases and
// tags that aren't semver are refused.
func ParseRelease(b []byte) (*Release, error) {
	var r struct {
		TagName    string `json:"tag_name"`
		Name       string `json:"name"`
		Body       string `json:"body"`
		HTMLURL    string `json:"html_url"`
		Draft      bool   `json:"draft"`
		Prerelease bool   `json:"prerelease"`
		Assets     []struct {
			Name string `json:"name"`
			URL  string `json:"browser_download_url"`
			Size int64  `json:"size"`
		} `json:"assets"`
	}
	if err := json.Unmarshal(b, &r); err != nil {
		return nil, fmt.Errorf("release: %w", err)
	}
	if r.Draft || r.Prerelease {
		return nil, errors.New("the latest release is a draft or a pre-release")
	}
	v, ok := Parse(r.TagName)
	if !ok {
		return nil, fmt.Errorf("release tag %q isn't a version", r.TagName)
	}
	out := &Release{Version: v.String(), Tag: r.TagName, NotesURL: r.HTMLURL,
		Security: strings.Contains(strings.ToLower(r.Name+"\n"+r.Body), SecurityMarker)}
	for _, a := range r.Assets {
		if a.Name == "" || strings.ContainsAny(a.Name, "/\\") || a.URL == "" {
			continue
		}
		out.Assets = append(out.Assets, Asset{Name: a.Name, URL: a.URL, Size: a.Size})
	}
	return out, nil
}

// Asset finds an asset by exact name.
func (r *Release) Asset(name string) (Asset, bool) {
	for _, a := range r.Assets {
		if a.Name == name {
			return a, true
		}
	}
	return Asset{}, false
}

// FindAsset returns the first asset whose name matches.
func (r *Release) FindAsset(match func(string) bool) (Asset, bool) {
	for _, a := range r.Assets {
		if match(a.Name) {
			return a, true
		}
	}
	return Asset{}, false
}

// ParseSums reads sha256sum output ("<hex>  <name>" or "<hex> *<name>").
func ParseSums(b []byte) map[string]string {
	out := map[string]string{}
	sc := bufio.NewScanner(bytes.NewReader(b))
	for sc.Scan() {
		f := strings.Fields(sc.Text())
		if len(f) != 2 || len(f[0]) != 64 {
			continue
		}
		if _, err := hex.DecodeString(f[0]); err != nil {
			continue
		}
		out[strings.TrimPrefix(f[1], "*")] = strings.ToLower(f[0])
	}
	return out
}

// Client talks to the GitHub API and downloads release assets.
type Client struct {
	API   string // default https://api.github.com
	HTTP  *http.Client
	Token string // optional (rate limit)
}

func (c Client) api() string {
	if c.API == "" {
		return "https://api.github.com"
	}
	return strings.TrimRight(c.API, "/")
}

// allowHTTP: plain http only when the API itself is http (tests, a local
// mirror of the API); github.com is always https.
func (c Client) allowHTTP() bool { return strings.HasPrefix(c.api(), "http://") }

func (c Client) http() *http.Client {
	if c.HTTP != nil {
		return c.HTTP
	}
	return &http.Client{Timeout: 10 * time.Minute}
}

// errNotModified: a conditional API request (If-None-Match) found the
// resource unchanged.
var errNotModified = errors.New("not modified")

func (c Client) get(ctx context.Context, u string, api bool, w io.Writer, max int64) error {
	_, err := c.getETag(ctx, u, api, "", w, max)
	return err
}

// getETag is get with a conditional request: etag ("" = none) goes out as
// If-None-Match, and the response's ETag comes back. A 304 is
// errNotModified (GitHub doesn't count those against the rate limit).
func (c Client) getETag(ctx context.Context, u string, api bool, etag string, w io.Writer, max int64) (string, error) {
	if !strings.HasPrefix(u, "https://") && !(c.allowHTTP() && strings.HasPrefix(u, "http://")) {
		return "", fmt.Errorf("refusing %s: not https", u)
	}
	ctx, cancel := context.WithTimeout(ctx, 10*time.Minute)
	defer cancel()
	req, err := http.NewRequestWithContext(ctx, http.MethodGet, u, nil)
	if err != nil {
		return "", err
	}
	if etag != "" {
		req.Header.Set("If-None-Match", etag)
	}
	if api {
		req.Header.Set("Accept", "application/vnd.github+json")
		req.Header.Set("X-GitHub-Api-Version", "2022-11-28")
		if c.Token != "" {
			req.Header.Set("Authorization", "Bearer "+c.Token)
		}
	}
	resp, err := c.http().Do(req)
	if err != nil {
		return "", err
	}
	defer resp.Body.Close()
	switch {
	case resp.StatusCode == http.StatusNotModified && etag != "":
		return etag, errNotModified
	case resp.StatusCode == http.StatusNotFound && api:
		return "", errors.New("no release yet")
	case resp.StatusCode == http.StatusForbidden || resp.StatusCode == http.StatusTooManyRequests:
		return "", fmt.Errorf("GitHub said HTTP %d (rate limit?)", resp.StatusCode)
	case resp.StatusCode != http.StatusOK:
		return "", fmt.Errorf("%s: HTTP %d", u, resp.StatusCode)
	}
	n, err := io.Copy(w, io.LimitReader(resp.Body, max+1))
	if err == nil && n > max {
		err = errors.New("download too large")
	}
	return resp.Header.Get("ETag"), err
}

// Latest asks GET /repos/<repo>/releases/latest.
func (c Client) Latest(ctx context.Context, repo string) (*Release, error) {
	repo, err := RepoPath(repo)
	if err != nil {
		return nil, err
	}
	var b bytes.Buffer
	if err := c.get(ctx, c.api()+"/repos/"+repo+"/releases/latest", true, &b, 4<<20); err != nil {
		return nil, err
	}
	return ParseRelease(b.Bytes())
}

// LatestIfChanged is Latest as a conditional request: etag is the ETag of
// the last answer ("" = none). Unchanged since: (nil, etag, true, nil),
// and the caller keeps the release it has.
func (c Client) LatestIfChanged(ctx context.Context, repo, etag string) (rel *Release, newETag string, unchanged bool, err error) {
	repo, err = RepoPath(repo)
	if err != nil {
		return nil, "", false, err
	}
	var b bytes.Buffer
	newETag, err = c.getETag(ctx, c.api()+"/repos/"+repo+"/releases/latest", true, etag, &b, 4<<20)
	if errors.Is(err, errNotModified) {
		return nil, etag, true, nil
	}
	if err != nil {
		return nil, "", false, err
	}
	rel, err = ParseRelease(b.Bytes())
	return rel, newETag, false, err
}

// Tag asks GET /repos/<repo>/releases/tags/<tag>.
func (c Client) Tag(ctx context.Context, repo, tag string) (*Release, error) {
	repo, err := RepoPath(repo)
	if err != nil {
		return nil, err
	}
	var b bytes.Buffer
	if err := c.get(ctx, c.api()+"/repos/"+repo+"/releases/tags/"+url.PathEscape(tag), true, &b, 4<<20); err != nil {
		return nil, err
	}
	return ParseRelease(b.Bytes())
}

// Download fetches one asset into dst and checks it against the release's
// SHA256SUMS; dst only appears when it matches.
func (c Client) Download(ctx context.Context, r *Release, name, dst string) error {
	a, ok := r.Asset(name)
	if !ok {
		return fmt.Errorf("release %s has no %s", r.Version, name)
	}
	s, ok := r.Asset(SumsFile)
	if !ok {
		return fmt.Errorf("release %s has no %s: not installing anything unchecked", r.Version, SumsFile)
	}
	var sb bytes.Buffer
	if err := c.get(ctx, s.URL, false, &sb, 1<<20); err != nil {
		return err
	}
	want, ok := ParseSums(sb.Bytes())[name]
	if !ok {
		return fmt.Errorf("%s has no line for %s", SumsFile, name)
	}
	if err := os.MkdirAll(filepath.Dir(dst), 0o700); err != nil {
		return err
	}
	tmp := dst + ".part"
	f, err := os.Create(tmp)
	if err != nil {
		return err
	}
	h := sha256.New()
	err = c.get(ctx, a.URL, false, io.MultiWriter(f, h), 512<<20)
	if cerr := f.Close(); err == nil {
		err = cerr
	}
	if err == nil && hex.EncodeToString(h.Sum(nil)) != want {
		err = fmt.Errorf("%s: SHA-256 doesn't match %s", name, SumsFile)
	}
	if err != nil {
		os.Remove(tmp)
		return err
	}
	return os.Rename(tmp, dst)
}
