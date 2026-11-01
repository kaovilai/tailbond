package backup

import (
	"bytes"
	"encoding/base64"
	"encoding/json"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/tailbond/tailbond/agent/internal/config"
)

const pass = "correct horse battery"

func TestSealOpen(t *testing.T) {
	plain := []byte(`{"format":1,"config":{}}`)
	data, err := Seal(plain, pass, Header{Role: "server", Platform: "fedora", Host: "homebox", AgentVersion: "0.2.0"})
	if err != nil {
		t.Fatal(err)
	}
	if !bytes.HasPrefix(data, []byte(Magic+"\n")) || bytes.Contains(data, plain) {
		t.Fatal("not sealed")
	}
	h, got, err := Open(data, pass)
	if err != nil || !bytes.Equal(got, plain) || h.Role != "server" || h.Host != "homebox" || h.N != ScryptN {
		t.Fatalf("round trip: %v %s %+v", err, got, h)
	}
	if _, _, err := Open(data, "wrong passphrase!"); err != ErrPassphrase {
		t.Fatalf("wrong passphrase: %v", err)
	}
	// a flipped byte in the ciphertext
	lines := bytes.SplitN(data, []byte("\n"), 4)
	ct, _ := base64.StdEncoding.DecodeString(string(lines[2]))
	ct[len(ct)/2] ^= 1
	bad := bytes.Join([][]byte{lines[0], lines[1], []byte(base64.StdEncoding.EncodeToString(ct)), nil}, []byte("\n"))
	if _, _, err := Open(bad, pass); err != ErrPassphrase {
		t.Fatalf("damaged payload: %v", err)
	}
	// an edited header (role) is authenticated
	edited := bytes.Replace(data, []byte(`"role":"server"`), []byte(`"role":"client"`), 1)
	if _, _, err := Open(edited, pass); err != ErrPassphrase {
		t.Fatalf("edited header: %v", err)
	}
	// silly scrypt parameters are refused before any work
	weak := bytes.Replace(data, []byte(`"n":32768`), []byte(`"n":16`), 1)
	if _, _, err := Open(weak, pass); err == nil || err == ErrPassphrase {
		t.Fatalf("weak kdf: %v", err)
	}
	for _, junk := range [][]byte{nil, []byte("hello"), []byte(Magic + "\n{}\nAAAA\n"), bytes.Repeat([]byte("x"), 10)} {
		if _, _, err := Open(junk, pass); err == nil {
			t.Fatalf("junk %q opened", junk)
		}
	}
	if _, err := Seal(plain, "short", Header{}); err != ErrShort {
		t.Fatalf("short passphrase: %v", err)
	}
	// two backups of the same data differ (fresh salt and nonce)
	d2, _ := Seal(plain, pass, Header{Role: "server"})
	if bytes.Equal(data, d2) {
		t.Fatal("deterministic output")
	}
}

func TestCollectDecodeApply(t *testing.T) {
	dir := t.TempDir()
	state := filepath.Join(dir, "state")
	os.MkdirAll(state, 0o700)
	cfg := config.Default()
	cfg.StateDir = state
	cfg.UIPassword = "s3cret"
	cfg.Server.PublicAddr = "home.example.net"
	cfg.Server.Users = []config.User{{Name: "slate7", Key: "k1"}, {Name: "pixel", Key: "k2"}}
	cfgPath := filepath.Join(dir, "tailbond.json")
	if err := config.WriteFile(cfgPath, cfg); err != nil {
		t.Fatal(err)
	}
	os.WriteFile(filepath.Join(state, "server.crt"), []byte("CERT"), 0o600)
	os.WriteFile(filepath.Join(state, "server.key"), []byte("KEY"), 0o600)
	os.WriteFile(filepath.Join(state, "usage.json"), []byte("{}"), 0o600)
	p, err := Collect(cfgPath, state, "config tailbond 'main'\n", nil)
	if err != nil {
		t.Fatal(err)
	}
	if len(p.Files) != 2 || p.UCI == "" {
		t.Fatalf("files %v", p.Files)
	}
	// round trip through the encrypted container
	pt, _ := encodeJSON(p)
	data, _ := Seal(pt, pass, Header{Role: "server"})
	_, pt2, err := Open(data, pass)
	if err != nil {
		t.Fatal(err)
	}
	p2, cfg2, err := Decode(pt2)
	if err != nil || len(cfg2.Server.Users) != 2 || cfg2.UIPassword != "s3cret" {
		t.Fatalf("decode: %v %+v", err, cfg2)
	}
	// restore onto a new machine
	state2 := filepath.Join(t.TempDir(), "state")
	if err := Apply(p2, state2); err != nil {
		t.Fatal(err)
	}
	if b, _ := os.ReadFile(filepath.Join(state2, "server.key")); string(b) != "KEY" {
		t.Fatal("key not restored")
	}
	if fi, _ := os.Stat(filepath.Join(state2, "server.key")); fi.Mode().Perm() != 0o600 {
		t.Fatalf("mode %v", fi.Mode())
	}
	// preview of what changes, without secrets
	cur := config.Default()
	cur.Server.PublicAddr = "new-vps.example.net"
	ch := Changes(cur, cfg2)
	joined := ""
	for _, c := range ch {
		joined += c.What + ":" + c.From + "->" + c.To + ";"
	}
	if !strings.Contains(joined, "Devices (clients):(none)->slate7, pixel") || !strings.Contains(joined, "Public address:new-vps.example.net->home.example.net") || strings.Contains(joined, "s3cret") || strings.Contains(joined, "k1") {
		t.Fatalf("changes %s", joined)
	}
	sum := Summary(cfg2, p2)
	if s := encodeAny(sum); strings.Contains(s, "s3cret") || strings.Contains(s, "k1") || !strings.Contains(s, "slate7") {
		t.Fatalf("summary leaks or misses: %s", s)
	}
	// a payload with a file it shouldn't carry
	bad := strings.Replace(string(pt2), `"files":{`, `"files":{"../../etc/passwd":"eA==",`, 1)
	if _, _, err := Decode([]byte(bad)); err == nil {
		t.Fatal("unexpected file accepted")
	}
}

func encodeJSON(p *Payload) ([]byte, error) { return json.Marshal(p) }

func encodeAny(v any) string {
	b, _ := json.Marshal(v)
	return string(b)
}
