// SPDX-License-Identifier: Apache-2.0

package report

import (
	"encoding/json"
	"flag"
	"os"
	"strings"
	"testing"
)

var update = flag.Bool("update", false, "rewrite testdata/vectors.json's outputs")

// vectors.json: the Android and iOS ports run the same cases.
type vectors struct {
	Comment string `json:"comment"`
	Redact  []struct {
		Name string `json:"name"`
		In   string `json:"in"`
		Out  string `json:"out"`
	} `json:"redact"`
	Escape []struct {
		In  string `json:"in"`
		Out string `json:"out"`
	} `json:"escape"`
	Texts   map[string]string `json:"texts"`
	Reports []struct {
		Name   string `json:"name"`
		Report Report `json:"report"`
		Full   string `json:"full"`
		URL    string `json:"url"`
	} `json:"reports"`
}

func load(t *testing.T) (vectors, []byte) {
	t.Helper()
	b, err := os.ReadFile("testdata/vectors.json")
	if err != nil {
		t.Fatal(err)
	}
	var v vectors
	if err := json.Unmarshal(b, &v); err != nil {
		t.Fatal(err)
	}
	return v, b
}

func TestVectors(t *testing.T) {
	v, _ := load(t)
	if len(v.Redact) < 20 || len(v.Reports) < 3 {
		t.Fatalf("too few vectors: %d redact, %d reports", len(v.Redact), len(v.Reports))
	}
	for i := range v.Redact {
		c := &v.Redact[i]
		got := Redact(c.In)
		if *update {
			c.Out = got
		} else if got != c.Out {
			t.Errorf("%s: Redact(%q)\n got %q\nwant %q", c.Name, c.In, got, c.Out)
		}
	}
	for i := range v.Escape {
		c := &v.Escape[i]
		if *update {
			c.Out = Escape(c.In)
		} else if got := Escape(c.In); got != c.Out {
			t.Errorf("Escape(%q) = %q, want %q", c.In, got, c.Out)
		}
	}
	for i := range v.Reports {
		c := &v.Reports[i]
		full, u := c.Report.Full(), c.Report.IssueURL()
		if *update {
			c.Full, c.URL = full, u
			continue
		}
		if full != c.Full {
			t.Errorf("%s: Full\n got %q\nwant %q", c.Name, full, c.Full)
		}
		if u != c.URL {
			t.Errorf("%s: IssueURL\n got %q\nwant %q", c.Name, u, c.URL)
		}
		if len(u) > MaxURL {
			t.Errorf("%s: URL %d long", c.Name, len(u))
		}
	}
	texts := map[string]string{"notice": Notice, "confirm": Confirm, "security_note": SecurityNote, "security_url": SecurityURL, "account_note": AccountNote, "repo": Repo, "max_url": itoa(MaxURL)}
	for k, want := range texts {
		if !*update && v.Texts[k] != want {
			t.Errorf("texts.%s = %q, want %q", k, v.Texts[k], want)
		}
	}
	if *update {
		v.Texts = texts
		b, _ := json.MarshalIndent(v, "", "  ")
		if err := os.WriteFile("testdata/vectors.json", append(b, '\n'), 0o644); err != nil {
			t.Fatal(err)
		}
	}
}

// What the vectors must show, independent of their recorded outputs.
func TestNothingSecretSurvives(t *testing.T) {
	v, _ := load(t)
	for _, c := range v.Redact {
		got := Redact(c.In)
		for _, bad := range []string{"tskey-auth", "d=eyJ", "203.0.113.", "2001:db8:1", "sam@example.com", "hunter2",
			"tail1234", "/home/tiger", "/Users/tiger", "aa:bb:cc:dd:ee:ff", "0123456789abcdef0123456789abcdef", "s3cr3tT0kenVALUEthatIsLongEnough1234567890"} {
			if strings.Contains(got, bad) {
				t.Errorf("%s: %q survives in %q", c.Name, bad, got)
			}
		}
	}
	// what must stay
	keep := "10.77.0.2 192.168.8.1 100.101.1.1 fd7a:115c:a1e0::1 fe80::1 127.0.0.1 mqvpn 0.19.0 tailbond-glinet-aarch64_cortex-a53_neon-vfpv4.ipk /usr/local/var/lib/tailbond/update 12:34:56 key id 3f2a"
	if got := Redact(keep); got != keep {
		t.Errorf("changed what should stay:\n got %q\nwant %q", got, keep)
	}
}

func TestURLStaysShort(t *testing.T) {
	var logs []string
	for i := 0; i < 3000; i++ {
		logs = append(logs, "2026-11-04 06:51:24 path wifi-13 srtt 14ms loss 0.1% line "+itoa(i))
	}
	r := Report{Component: "Mac", Version: "0.2.0", Device: "macOS 26.1 (Mac15,3)", What: "slow", Logs: strings.Join(logs, "\n")}
	u := r.IssueURL()
	if len(u) > MaxURL || len(u) < MaxURL-600 {
		t.Fatalf("URL is %d long", len(u))
	}
	if !strings.Contains(u, Escape("line 2999")) || strings.Contains(u, Escape("line 0\n")) || !strings.Contains(u, "earlier%20lines%20left%20out") {
		t.Error("didn't keep the newest lines with a note")
	}
	if !strings.Contains(r.Full(), "line 0\n") {
		t.Error("the full report lost lines")
	}
	// a crash longer than a link: a note instead
	huge := Report{Component: "Mac", What: "crash", Crash: strings.Repeat("goroutine 1 [running]:\n", 600)}
	if u := huge.IssueURL(); len(u) > MaxURL || !strings.Contains(u, "too%20long%20for%20a%20link") {
		t.Errorf("huge crash: %d %q", len(u), u[:200])
	}
}

// The issue form's dropdown takes a prefilled value only when it's one of
// its options, word for word: what the agent and the apps send must be.
func TestComponentsAreFormOptions(t *testing.T) {
	b, err := os.ReadFile("../../../.github/ISSUE_TEMPLATE/bug_report.yml")
	if err != nil {
		t.Fatal(err)
	}
	for _, c := range []string{
		// agent (reportapi.go component())
		"GL.iNet / OpenWrt router (home server)", "GL.iNet / OpenWrt router (travel client)", "Mac",
		"Linux laptop (client)", "Fedora home server (RPM)", "VPS / Linux install.sh",
		// Android (ReportScreen.kt), iPhone (ReportView.swift)
		"Android app", "iPhone Share app",
	} {
		if !strings.Contains(string(b), "\n        - "+c+"\n") {
			t.Errorf("bug_report.yml has no option %q", c)
		}
	}
	for _, id := range []string{"id: component", "id: what", "id: steps", "id: version", "id: device", "id: logs"} {
		if !strings.Contains(string(b), id) {
			t.Errorf("bug_report.yml lost field %q (IssueURL prefills it)", id)
		}
	}
}
