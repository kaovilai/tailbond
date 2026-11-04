// SPDX-License-Identifier: Apache-2.0

package agent

// Report a problem (docs/reporting.md): crash capture and the report the
// dashboard (and the GL panel) previews. Nothing is sent from here: the
// browser opens the GitHub issue URL the user approved.

import (
	"bufio"
	"errors"
	"fmt"
	"net/http"
	"net/netip"
	"os"
	"os/exec"
	"path/filepath"
	"regexp"
	goruntime "runtime"
	"runtime/debug"
	"strings"
	"sync"
	"syscall"
	"time"

	"github.com/kaovilai/tailbond/agent/internal/config"
	"github.com/kaovilai/tailbond/agent/internal/report"
)

const (
	crashFile = "crash.log"      // this run's fatal error output (runtime/debug.SetCrashOutput)
	crashLast = "crash-last.log" // the last crash, until the user reports or dismisses it
)

type crashState struct {
	mu sync.Mutex
	at time.Time // of crash-last.log; zero = none
}

func (a *Agent) crashPath(name string) string { return filepath.Join(a.store.Get().StateDir, name) }

// setupCrash keeps the previous run's crash output (a Go fatal error or
// panic that killed tailbondd) as crash-last.log, and sends this run's to
// crash.log.
func (a *Agent) setupCrash() {
	cur, last := a.crashPath(crashFile), a.crashPath(crashLast)
	if fi, err := os.Stat(cur); err == nil && fi.Size() > 0 {
		b, _ := os.ReadFile(cur)
		head := fmt.Sprintf("tailbondd stopped with a fatal error at %s (found when %s started)\n",
			fi.ModTime().UTC().Format(time.RFC3339), a.opts.Version)
		if err := os.WriteFile(last, append([]byte(head), b...), 0o600); err == nil {
			a.Logf("tailbondd crashed last time (%s): Report a problem has the details", fi.ModTime().UTC().Format(time.RFC3339))
		}
	}
	f, err := os.OpenFile(cur, os.O_CREATE|os.O_TRUNC|os.O_WRONLY, 0o600)
	if err != nil {
		a.Logf("crash output: %v", err)
	} else {
		if err := debug.SetCrashOutput(f, debug.CrashOptions{}); err != nil {
			a.Logf("crash output: %v", err)
		}
		f.Close() // SetCrashOutput keeps its own copy
	}
	a.loadCrash()
}

func (a *Agent) loadCrash() {
	if a.crash == nil {
		a.crash = &crashState{}
	}
	a.crash.mu.Lock()
	defer a.crash.mu.Unlock()
	a.crash.at = time.Time{}
	if fi, err := os.Stat(a.crashPath(crashLast)); err == nil && fi.Size() > 0 {
		a.crash.at = fi.ModTime()
	}
}

// crashAt: when the kept crash happened (RFC 3339), "" when none.
func (a *Agent) crashAt() string {
	if a.crash == nil {
		return ""
	}
	a.crash.mu.Lock()
	defer a.crash.mu.Unlock()
	if a.crash.at.IsZero() {
		return ""
	}
	return a.crash.at.UTC().Format(time.RFC3339)
}

// noteChildCrash records mqvpn dying on a crash signal, with its last lines.
func (a *Agent) noteChildCrash(err error) {
	var ee *exec.ExitError
	if !errors.As(err, &ee) {
		return
	}
	ws, ok := ee.Sys().(syscall.WaitStatus)
	if !ok || !ws.Signaled() {
		return
	}
	switch ws.Signal() {
	case syscall.SIGSEGV, syscall.SIGABRT, syscall.SIGBUS, syscall.SIGILL, syscall.SIGFPE, syscall.SIGTRAP:
	default:
		return
	}
	var b strings.Builder
	fmt.Fprintf(&b, "mqvpn stopped with signal %s at %s (tailbondd %s)\nits last lines:\n", ws.Signal(), time.Now().UTC().Format(time.RFC3339), a.opts.Version)
	var lines []string
	for _, e := range a.logs.Since(0, 0) {
		if e.Src == "mqvpn" {
			lines = append(lines, e.Line)
		}
	}
	if len(lines) > 40 {
		lines = lines[len(lines)-40:]
	}
	b.WriteString(strings.Join(lines, "\n"))
	if err := os.WriteFile(a.crashPath(crashLast), []byte(b.String()+"\n"), 0o600); err == nil {
		a.Logf("mqvpn crashed (signal %s): Report a problem has the details", ws.Signal())
		a.loadCrash()
	}
}

// component: the bug_report.yml "Where does it happen?" option for this install.
func (a *Agent) component() string {
	switch {
	case isOpenWrt() && a.role == config.RoleServer:
		return "GL.iNet / OpenWrt router (home server)"
	case isOpenWrt():
		return "GL.iNet / OpenWrt router (travel client)"
	case goruntime.GOOS == "darwin":
		return "Mac"
	case a.role == config.RoleClient:
		return "Linux laptop (client)"
	case a.method() == methodDnf:
		return "Fedora home server (RPM)"
	}
	return "VPS / Linux install.sh"
}

// deviceInfo: model and OS, as the issue form asks.
func deviceInfo() string {
	arch := goruntime.GOARCH
	if goruntime.GOOS == "darwin" {
		v, _ := exec.Command("sw_vers", "-productVersion").Output()
		m, _ := exec.Command("sysctl", "-n", "hw.model").Output()
		s := "macOS " + strings.TrimSpace(string(v))
		if mm := strings.TrimSpace(string(m)); mm != "" {
			s += " (" + mm + ")"
		}
		return s + ", " + arch
	}
	var parts []string
	if b, err := os.ReadFile("/tmp/sysinfo/model"); err == nil {
		parts = append(parts, strings.TrimSpace(string(b)))
	}
	if b, err := os.ReadFile("/etc/glversion"); err == nil {
		parts = append(parts, "GL "+strings.TrimSpace(string(b)))
	}
	if v := osRelease("/etc/openwrt_release", "DISTRIB_DESCRIPTION"); v != "" {
		parts = append(parts, v)
	} else if v := osRelease("/etc/os-release", "PRETTY_NAME"); v != "" {
		parts = append(parts, v)
	}
	parts = append(parts, arch)
	return strings.Join(parts, ", ")
}

func osRelease(path, key string) string {
	f, err := os.Open(path)
	if err != nil {
		return ""
	}
	defer f.Close()
	sc := bufio.NewScanner(f)
	for sc.Scan() {
		if v, ok := strings.CutPrefix(sc.Text(), key+"="); ok {
			return strings.Trim(v, `"'`)
		}
	}
	return ""
}

// reportLogs: the newest log lines, as the Logs tab shows them.
func (a *Agent) reportLogs(n int) string {
	var b strings.Builder
	for _, e := range a.logs.Since(0, n) {
		b.WriteString(time.UnixMilli(e.Time).UTC().Format("15:04:05") + " [" + e.Src + "] " + e.Line + "\n")
	}
	return b.String()
}

var reAddrToken = regexp.MustCompile(`\b[0-9]{1,3}(?:\.[0-9]{1,3}){3}(?:/[0-9]{1,2})?\b|[0-9A-Fa-f]{0,4}(?::[0-9A-Fa-f]{0,4}){2,7}(?:/[0-9]{1,3})?`)

// redactLocal: what only this machine knows is private, on top of
// report.Redact (docs/macos.md "Using tailbond with NetBird"): the work
// VPN's addresses and the networks it routes (NetBird's range, the company
// subnets it pushes) as [work address] / [work network], and its internal
// names (the split-DNS domains) as [work name].
func (a *Agent) redactLocal(s string) string {
	st := a.Snapshot()
	if st == nil || st.Coexist == nil {
		return s
	}
	var work []netip.Prefix
	for _, f := range st.Coexist.Foreign {
		if f.Kind == "tailscale" {
			continue // the tailnet's own addresses stay, as everywhere
		}
		for _, p := range append(append([]string{}, f.Prefixes...), f.Addrs...) {
			if pf, err := netip.ParsePrefix(p); err == nil {
				work = append(work, pf.Masked())
			}
		}
	}
	if len(work) > 0 {
		s = reAddrToken.ReplaceAllStringFunc(s, func(m string) string {
			addr, isNet := m, false
			if i := strings.IndexByte(m, '/'); i >= 0 {
				addr, isNet = m[:i], true
			}
			ip, err := netip.ParseAddr(addr)
			if err != nil {
				return m
			}
			for _, p := range work {
				if p.Contains(ip.Unmap()) {
					if isNet {
						return "[work network]"
					}
					return "[work address]"
				}
			}
			return m
		})
	}
	for _, d := range st.Coexist.DNSDomains {
		d = strings.Trim(strings.ToLower(strings.TrimSpace(d)), ".")
		if !strings.Contains(d, ".") {
			continue
		}
		re := regexp.MustCompile(`(?i)\b(?:[a-z0-9-]+\.)*` + regexp.QuoteMeta(d) + `\b`)
		s = re.ReplaceAllString(s, "[work name]")
	}
	return s
}

func (a *Agent) redactReport(r report.Report) report.Report {
	f := a.redactLocal
	return report.Report{Component: r.Component, Version: f(r.Version), Device: f(r.Device), What: f(r.What),
		Steps: f(r.Steps), Logs: f(r.Logs), Crash: f(r.Crash)}
}

// GET /api/report: the report to preview, redacted.
func (a *Agent) hReportGet(w http.ResponseWriter, r *http.Request) {
	crash := ""
	if b, err := os.ReadFile(a.crashPath(crashLast)); err == nil {
		crash = string(b)
	}
	rep := a.redactReport(report.Report{
		Component: a.component(),
		Version:   a.opts.Version,
		Device:    deviceInfo(),
		Logs:      a.reportLogs(300),
		Crash:     crash,
	})
	writeJSON(w, http.StatusOK, map[string]any{
		"component":     rep.Component,
		"version":       report.Redact(rep.Version),
		"device":        report.Redact(rep.Device),
		"logs":          report.Redact(rep.Logs),
		"crash":         report.Redact(rep.Crash),
		"crash_at":      a.crashAt(),
		"notice":        report.Notice,
		"confirm":       report.Confirm,
		"security_note": report.SecurityNote,
		"security_url":  report.SecurityURL,
		"account_note":  report.AccountNote,
	})
}

// POST /api/report: the user's edited report -> the issue URL (redacted
// again: they may have pasted something), the full text and a file name.
func (a *Agent) hReportPost(w http.ResponseWriter, r *http.Request) {
	var rep report.Report
	if err := readJSON(r, &rep); err != nil {
		httpErr(w, http.StatusBadRequest, err)
		return
	}
	rep = a.redactReport(rep)
	writeJSON(w, http.StatusOK, map[string]any{
		"url":       rep.IssueURL(),
		"full":      rep.Full(),
		"file_name": report.FileName(time.Now().UTC().Format(time.RFC3339)),
	})
}

// POST /api/report/crash/dismiss: forget the kept crash.
func (a *Agent) hReportDismiss(w http.ResponseWriter, r *http.Request) {
	_ = os.Remove(a.crashPath(crashLast))
	a.loadCrash()
	writeJSON(w, http.StatusOK, map[string]any{"ok": true})
}
