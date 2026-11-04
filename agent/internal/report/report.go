// SPDX-License-Identifier: Apache-2.0

// Package report builds "Report a problem" reports (docs/reporting.md):
// the redactor, the report text and the prefilled GitHub issue URL.
//
// The Android app (dev.tailbond.app.report) and tailbond Share
// (TailbondShareKit/Report.swift) port this file; testdata/vectors.json
// pins all three to the same output. Change the rules here, regenerate the
// vectors (go test ./internal/report -update), and port the change.
//
// Nothing here sends anything: the user opens the URL in their browser,
// edits the issue there and submits it themselves.
package report

import (
	"regexp"
	"strings"
)

// Repo is the GitHub repository reports go to.
const Repo = "kaovilai/tailbond"

// MaxURL is the longest issue URL built (GitHub and browsers cut longer
// ones); the logs are shortened from the oldest line to fit.
const MaxURL = 7000

// Notice is shown above every report preview, in every app.
const Notice = "Nothing is sent automatically: tailbond has no telemetry and makes no network call for this. " +
	"The report opens in your browser as a GitHub issue that you can still edit, and once you submit it, it is public. " +
	"Keys, pairing links, public IP addresses, email addresses, your tailnet's name and similar details were replaced with [brackets]; " +
	"read it and remove anything else you don't want public (Wi-Fi names, device names)."

// Confirm is the checkbox the user must tick before the issue opens.
const Confirm = "I've read this report and it has nothing I don't want public."

// SecurityNote points security problems away from public issues.
const SecurityNote = "A security problem? Don't open a public issue: report it privately on GitHub (Security → Report a vulnerability)."

// SecurityURL is GitHub's private vulnerability report form.
const SecurityURL = "https://github.com/" + Repo + "/security/advisories/new"

type rule struct {
	re   *regexp.Regexp
	repl string              // with $1, $2 (written ${1} for Go)
	fn   func(string) string // instead of repl: the whole match in, its replacement out
}

// The rules, in order. Patterns use only syntax RE2, Java and ICU agree on:
// ASCII classes ([0-9], no \d or \w), \b, (?i), no lookarounds.
var rules = []rule{
	// pairing and share links: keep the kind, drop the payload
	{re: regexp.MustCompile(`(tailbond://[A-Za-z0-9._-]+)\?[^\s"'<>]+`), repl: "${1}?[redacted]"},
	// Tailscale auth keys
	{re: regexp.MustCompile(`tskey-[A-Za-z0-9-]+`), repl: "[tailscale key]"},
	// HTTP authorization values
	{re: regexp.MustCompile(`(?i)\b(Basic|Bearer)\s+[A-Za-z0-9+/=._-]+`), repl: "${1} [redacted]"},
	// command-line flags with a secret after a space: --password hunter2
	{re: regexp.MustCompile(`(?i)(--?(?:password|passwd|pass|token|secret|psk|authkey|auth-key|key))[ \t]+[^\s-][^\s]*`), repl: "${1} [redacted]"},
	// key: value, key=value, "key":"value" for secret-ish names
	{re: regexp.MustCompile(`(?i)\b(share_key|private_key|client_key|api_key|key|token|secret|password|passwd|pass|psk|auth|cookie)("?[ \t]*[:=][ \t]*"?)[^\s",;&}\]]+`), repl: "${1}${2}[redacted]"},
	// MAC addresses (before the IPv6 rule)
	{re: regexp.MustCompile(`\b[0-9A-Fa-f]{2}(?:[:-][0-9A-Fa-f]{2}){5}\b`), repl: "[mac]"},
	// long hex (keys, hashes of secrets) and long base64url runs
	{re: regexp.MustCompile(`\b[0-9A-Fa-f]{32,}\b`), repl: "[hex]"},
	{re: regexp.MustCompile(`[A-Za-z0-9+/_-]{40,}={0,2}`), fn: func(m string) string {
		if randomish(m) {
			return "[secret]"
		}
		return m
	}},
	// email addresses
	{re: regexp.MustCompile(`[A-Za-z0-9._%+-]+@[A-Za-z0-9.-]+\.[A-Za-z]{2,}`), repl: "[email]"},
	// the tailnet's name in MagicDNS names: homebox.tail1234.ts.net
	{re: regexp.MustCompile(`(?i)\b([A-Za-z0-9-]+)\.[A-Za-z0-9-]+\.ts\.net\b`), repl: "${1}.[tailnet].ts.net"},
	// the user name in home directories
	{re: regexp.MustCompile(`(/Users|/home)/[^/\s"']+`), repl: "${1}/[user]"},
	// public IPv4 (private, CGNAT/tailnet, loopback, link-local stay)
	// (a version like a kernel's 6.18.33.2-microsoft stays: the suffix comes along)
	{re: regexp.MustCompile(`\b[0-9]{1,3}\.[0-9]{1,3}\.[0-9]{1,3}\.[0-9]{1,3}\b(?:-[A-Za-z][A-Za-z0-9]*)?`), fn: func(m string) string {
		if !strings.Contains(m, "-") && publicIP(m) {
			return "[public IPv4]"
		}
		return m
	}},
	// public IPv6 (global unicast 2000::/3); tailnet fd7a:…, ULA, link-local stay
	// (a match that starts with one stray colon, "peer:2001:db8::1", is
	// checked without it)
	{re: regexp.MustCompile(`[0-9A-Fa-f]{0,4}(?::[0-9A-Fa-f]{0,4}){2,7}`), fn: func(m string) string {
		if publicIP(m) {
			return "[public IPv6]"
		}
		if strings.HasPrefix(m, ":") && !strings.HasPrefix(m, "::") && publicIP(m[1:]) {
			return ":[public IPv6]"
		}
		return m
	}},
}

// randomish: a long run that looks like a key (upper and lower case
// letters and digits, not a path), not a long file or package name.
func randomish(s string) bool {
	if strings.HasPrefix(s, "/") {
		return false
	}
	var up, lo, dg bool
	for i := 0; i < len(s); i++ {
		c := s[i]
		up = up || 'A' <= c && c <= 'Z'
		lo = lo || 'a' <= c && c <= 'z'
		dg = dg || '0' <= c && c <= '9'
	}
	return up && lo && dg
}

// publicIP: a valid IPv4 outside the private, CGNAT (tailnet), loopback,
// link-local, multicast and reserved ranges, or an IPv6 in 2000::/3.
// Hand-written parsers (no netip) so the ports parse exactly alike.
func publicIP(s string) bool {
	if b, ok := parseIPv4(s); ok {
		switch {
		case b[0] == 0, b[0] == 10, b[0] == 127, b[0] >= 224:
			return false
		case b[0] == 100 && b[1]&0xc0 == 64: // 100.64/10
			return false
		case b[0] == 169 && b[1] == 254, b[0] == 172 && b[1]&0xf0 == 16, b[0] == 192 && b[1] == 168:
			return false
		}
		return true
	}
	first, ok := parseIPv6First(s)
	return ok && first >= 0x2000 && first <= 0x3fff
}

// parseIPv4: four decimal octets, 0-255, no leading zeros.
func parseIPv4(s string) ([4]int, bool) {
	var b [4]int
	parts := strings.Split(s, ".")
	if len(parts) != 4 {
		return b, false
	}
	for i, p := range parts {
		if p == "" || len(p) > 3 || (len(p) > 1 && p[0] == '0') {
			return b, false
		}
		n := 0
		for j := 0; j < len(p); j++ {
			if p[j] < '0' || p[j] > '9' {
				return b, false
			}
			n = n*10 + int(p[j]-'0')
		}
		if n > 255 {
			return b, false
		}
		b[i] = n
	}
	return b, true
}

// parseIPv6First: whether s is an IPv6 address in hex groups (1-4 digits
// each, one "::" for one or more zero groups, no embedded IPv4), and its
// first 16-bit group.
func parseIPv6First(s string) (int, bool) {
	groups := func(p string) ([]int, bool) {
		if p == "" {
			return nil, true
		}
		var out []int
		for _, g := range strings.Split(p, ":") {
			if g == "" || len(g) > 4 {
				return nil, false
			}
			n := 0
			for j := 0; j < len(g); j++ {
				c := g[j]
				switch {
				case '0' <= c && c <= '9':
					n = n*16 + int(c-'0')
				case 'a' <= c && c <= 'f':
					n = n*16 + int(c-'a'+10)
				case 'A' <= c && c <= 'F':
					n = n*16 + int(c-'A'+10)
				default:
					return nil, false
				}
			}
			out = append(out, n)
		}
		return out, true
	}
	if i := strings.Index(s, "::"); i >= 0 {
		if strings.Contains(s[i+2:], "::") {
			return 0, false
		}
		l, ok1 := groups(s[:i])
		r, ok2 := groups(s[i+2:])
		if !ok1 || !ok2 || len(l)+len(r) > 7 {
			return 0, false
		}
		if len(l) > 0 {
			return l[0], true
		}
		return 0, true
	}
	g, ok := groups(s)
	if !ok || len(g) != 8 {
		return 0, false
	}
	return g[0], true
}

// Redact replaces secrets and identifying details in s (the rules above).
func Redact(s string) string {
	for _, r := range rules {
		if r.fn != nil {
			s = r.re.ReplaceAllStringFunc(s, r.fn)
		} else {
			s = r.re.ReplaceAllString(s, r.repl)
		}
	}
	return s
}

// Report is one problem report. Every field is redacted when used.
type Report struct {
	Component string `json:"component"` // an option of bug_report.yml's "Where does it happen?"
	Version   string `json:"version"`
	Device    string `json:"device"`
	What      string `json:"what"`
	Steps     string `json:"steps"`
	Logs      string `json:"logs"`
	Crash     string `json:"crash"` // a crash's stack/trace, when there was one
}

func (r Report) redacted() Report {
	return Report{
		Component: Redact(strings.TrimSpace(r.Component)),
		Version:   Redact(strings.TrimSpace(r.Version)),
		Device:    Redact(strings.TrimSpace(r.Device)),
		What:      Redact(strings.TrimSpace(r.What)),
		Steps:     Redact(strings.TrimSpace(r.Steps)),
		Logs:      Redact(strings.TrimRight(r.Logs, "\n")),
		Crash:     Redact(strings.TrimRight(r.Crash, "\n")),
	}
}

// logsField is what goes in the issue's Logs box: the crash first.
func (r Report) logsField() string {
	if r.Crash == "" {
		return r.Logs
	}
	if r.Logs == "" {
		return "--- crash ---\n" + r.Crash
	}
	return "--- crash ---\n" + r.Crash + "\n--- logs ---\n" + r.Logs
}

// Full is the whole report as text ("Copy full report", "Save report
// file"): the same sections as the issue form, never shortened.
func (r Report) Full() string {
	r = r.redacted()
	var b strings.Builder
	sec := func(h, v string) {
		if v == "" {
			v = "(not given)"
		}
		b.WriteString("### " + h + "\n\n" + v + "\n\n")
	}
	sec("Where does it happen?", r.Component)
	sec("What happened, and what did you expect?", r.What)
	sec("Steps to reproduce", r.Steps)
	sec("tailbond version", r.Version)
	sec("Device and firmware / OS", r.Device)
	b.WriteString("### Logs\n\n```text\n" + r.logsField() + "\n```\n")
	return b.String()
}

// IssueURL is the prefilled bug_report.yml form for r, at most MaxURL
// long: the oldest log lines go first (the crash stays), with a note.
func (r Report) IssueURL() string {
	r = r.redacted()
	lines := []string{}
	if r.Logs != "" {
		lines = strings.Split(r.Logs, "\n")
	}
	dropped := 0
	for {
		rr := r
		rr.Logs = strings.Join(lines, "\n")
		if dropped > 0 {
			rr.Logs = "(" + itoa(dropped) + " earlier lines left out: the full report has them)\n" + rr.Logs
		}
		u := rr.url()
		if len(u) <= MaxURL || len(lines) == 0 {
			if len(u) > MaxURL { // the crash alone is too long: cut it from the end
				rr.Crash = ""
				rr.Logs = "(the crash report is too long for a link: paste the full report here)"
				return rr.url()
			}
			return u
		}
		// drop about a tenth at a time, at least one line
		n := len(lines) / 10
		if n < 1 {
			n = 1
		}
		lines = lines[n:]
		dropped += n
	}
}

func (r Report) url() string {
	title := "bug: " + firstLine(r.What, 60)
	q := []string{
		"template=bug_report.yml",
		"title=" + Escape(title),
		"component=" + Escape(r.Component),
		"what=" + Escape(r.What),
		"steps=" + Escape(r.Steps),
		"version=" + Escape(r.Version),
		"device=" + Escape(r.Device),
		"logs=" + Escape(r.logsField()),
	}
	return "https://github.com/" + Repo + "/issues/new?" + strings.Join(q, "&")
}

// firstLine: the first line of s, at most n runes.
func firstLine(s string, n int) string {
	if i := strings.IndexByte(s, '\n'); i >= 0 {
		s = s[:i]
	}
	rs := []rune(strings.TrimSpace(s))
	if len(rs) > n {
		return string(rs[:n-1]) + "…"
	}
	return string(rs)
}

// Escape percent-encodes s for a URL query value: everything but
// A-Z a-z 0-9 - . _ ~ as %XX of its UTF-8 bytes (upper case), a space as
// %20. Spelled out so the three ports agree byte for byte.
func Escape(s string) string {
	const hex = "0123456789ABCDEF"
	var b strings.Builder
	for i := 0; i < len(s); i++ {
		c := s[i]
		switch {
		case 'A' <= c && c <= 'Z', 'a' <= c && c <= 'z', '0' <= c && c <= '9', c == '-', c == '.', c == '_', c == '~':
			b.WriteByte(c)
		default:
			b.WriteByte('%')
			b.WriteByte(hex[c>>4])
			b.WriteByte(hex[c&15])
		}
	}
	return b.String()
}

func itoa(n int) string {
	if n == 0 {
		return "0"
	}
	var d []byte
	for ; n > 0; n /= 10 {
		d = append([]byte{byte('0' + n%10)}, d...)
	}
	return string(d)
}

// FileName is the name "Save report file" suggests.
func FileName(at string) string {
	at = strings.NewReplacer(":", "", "-", "", "T", "-", "Z", "").Replace(at)
	if at == "" {
		return "tailbond-report.txt"
	}
	return "tailbond-report-" + at + ".txt"
}
