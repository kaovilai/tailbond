package bleproto

import (
	"bytes"
	"encoding/hex"
	"encoding/json"
	"flag"
	"os"
	"reflect"
	"testing"
	"time"

	"github.com/kaovilai/tailbond/agent/internal/proxy"
	"github.com/kaovilai/tailbond/agent/internal/share"
)

var update = flag.Bool("update", false, "rewrite testdata/vectors.json")

// vector: one login, the session after it and a sealed command each way.
type vector struct {
	ShareKey   string `json:"share_key"` // the phone's own key
	Hub        string `json:"hub"`
	Exp        uint64 `json:"exp"`
	PassKey    string `json:"pass_key"` // hex: what the server gives the Mac
	User       string `json:"user"`
	Secret     string `json:"secret"`
	ProxyPass  string `json:"proxy_pass"` // differs from secret
	CN         string `json:"cn"`         // base64, as sent
	SN         string `json:"sn"`
	AuthMAC    string `json:"auth_mac"`
	PhoneProof string `json:"phone_proof"`
	SessionKey string `json:"session_key"` // hex
	ToPhone    string `json:"to_phone"`    // the plain message, sealed at counter 0
	ToPhoneHex string `json:"to_phone_sealed"`
	ToMac      string `json:"to_mac"` // sealed at counter 1
	ToMacHex   string `json:"to_mac_sealed"`
}

type chunkVector struct {
	Message string   `json:"message"`
	MTU     int      `json:"mtu"`
	Chunks  []string `json:"chunks"` // hex
}

type vectors struct {
	Service string        `json:"service"`
	RX      string        `json:"rx"`
	TX      string        `json:"tx"`
	Logins  []vector      `json:"logins"`
	Chunks  []chunkVector `json:"chunks"`
}

func build(t *testing.T) vectors {
	in := []struct {
		key, hub string
		exp      uint64
		cn, sn   string
	}{
		{"0123456789abcdef0123456789abcdef", "mac", 1793836800, "AAECAwQFBgcICQoLDA0ODw==", "8PHy8/T19vf4+fr7/P3+/w=="},
		{"a much longer share key, 64 bytes or so, as the server makes it.", "dana-macbook", 4102444800, "MTIzNDU2Nzg5MGFiY2RlZg==", "ZmVkY2JhMDk4NzY1NDMyMQ=="},
	}
	v := vectors{Service: Service, RX: RX, TX: TX}
	for _, x := range in {
		pk := share.PassKey([]byte(x.key), x.hub, x.exp)
		u, s := Credentials(pk, x.hub, x.exp)
		_, pp := proxy.Credentials(pk, x.hub, x.exp)
		sk := SessionKey(s, x.cn, x.sn, u)
		toPhone := `{"t":"cmd","id":1,"cmd":"hotspot.on"}`
		toMac := `{"t":"res","id":1,"ok":true,"ssid":"Pixel 9","pass":"correct horse"}`
		a, err := Seal(sk, ToPhone, 0, []byte(toPhone))
		if err != nil {
			t.Fatal(err)
		}
		b, _ := Seal(sk, ToMac, 1, []byte(toMac))
		v.Logins = append(v.Logins, vector{x.key, x.hub, x.exp, hex.EncodeToString(pk), u, s, pp, x.cn, x.sn,
			AuthMAC(s, x.cn, x.sn, u), PhoneProof(s, x.cn, x.sn, u), hex.EncodeToString(sk), toPhone, hex.EncodeToString(a), toMac, hex.EncodeToString(b)})
	}
	for _, c := range []struct {
		msg string
		mtu int
	}{{`{"t":"hello","v":1,"cn":"AAECAwQFBgcICQoLDA0ODw=="}`, 23}, {`{"t":"cmd","id":2,"cmd":"status"}`, 185}} {
		f, _ := Frame([]byte(c.msg))
		var hs []string
		for _, p := range Chunks(f, c.mtu) {
			hs = append(hs, hex.EncodeToString(p))
		}
		v.Chunks = append(v.Chunks, chunkVector{c.msg, c.mtu, hs})
	}
	return v
}

// The Kotlin and Swift sides derive, seal and chunk the same (vectors.json).
func TestVectors(t *testing.T) {
	want := build(t)
	path := "testdata/vectors.json"
	if *update {
		os.MkdirAll("testdata", 0o755)
		b, _ := json.MarshalIndent(want, "", "  ")
		os.WriteFile(path, append(b, '\n'), 0o644)
	}
	b, err := os.ReadFile(path)
	if err != nil {
		t.Fatal(err)
	}
	var got vectors
	if err := json.Unmarshal(b, &got); err != nil {
		t.Fatal(err)
	}
	if !reflect.DeepEqual(got, want) {
		t.Fatalf("vectors.json is stale: go test ./internal/bleproto -update")
	}
	for _, v := range want.Logins {
		if v.Secret == v.ProxyPass {
			t.Errorf("%s: the Bluetooth secret equals the proxy password", v.Hub)
		}
		sk, _ := hex.DecodeString(v.SessionKey)
		sealed, _ := hex.DecodeString(v.ToMacHex)
		plain, err := Open(sk, ToMac, 1, sealed)
		if err != nil || string(plain) != v.ToMac {
			t.Errorf("%s: open: %q %v", v.Hub, plain, err)
		}
	}
}

func TestVerifier(t *testing.T) {
	key := []byte("home-share-key")
	exp := uint64(time.Now().Add(time.Hour).Unix())
	u, s := Credentials(share.PassKey(key, "mac", exp), "mac", exp)
	now := time.Now
	cn, sn := "Y24=", "c24="
	if got, ok := Verifier(key, now, nil)(u, cn, sn, AuthMAC(s, cn, sn, u)); !ok || got != s {
		t.Fatal("the phone refuses its own derivation")
	}
	cases := map[string]func() bool{
		"wrong mac":     func() bool { _, ok := Verifier(key, now, nil)(u, cn, sn, AuthMAC("nope", cn, sn, u)); return ok },
		"other nonce":   func() bool { _, ok := Verifier(key, now, nil)(u, cn, "eA==", AuthMAC(s, cn, sn, u)); return ok },
		"another phone": func() bool { _, ok := Verifier([]byte("other"), now, nil)(u, cn, sn, AuthMAC(s, cn, sn, u)); return ok },
		"hub not allowed": func() bool {
			_, ok := Verifier(key, now, func(string) bool { return false })(u, cn, sn, AuthMAC(s, cn, sn, u))
			return ok
		},
		"expired": func() bool {
			_, ok := Verifier(key, func() time.Time { return time.Unix(int64(exp)+1, 0) }, nil)(u, cn, sn, AuthMAC(s, cn, sn, u))
			return ok
		},
		"no expiry": func() bool { _, ok := Verifier(key, now, nil)("mac", cn, sn, AuthMAC(s, cn, sn, "mac")); return ok },
	}
	for name, f := range cases {
		if f() {
			t.Errorf("%s: accepted", name)
		}
	}
}

func TestSealOpen(t *testing.T) {
	k := SessionKey("abcdefghijklmnop", "a", "b", "mac/1")
	s, _ := Seal(k, ToPhone, 7, []byte("hi"))
	if _, err := Open(k, ToPhone, 8, s); err == nil {
		t.Error("opened at another counter (replay)")
	}
	if _, err := Open(k, ToMac, 7, s); err == nil {
		t.Error("opened in the other direction (reflection)")
	}
	s[0] ^= 1
	if _, err := Open(k, ToPhone, 7, s); err == nil {
		t.Error("opened a changed message")
	}
}

func TestFraming(t *testing.T) {
	msgs := [][]byte{[]byte(`{"t":"hello"}`), bytes.Repeat([]byte("x"), 700), {}}
	var stream []byte
	for _, m := range msgs {
		f, err := Frame(m)
		if err != nil {
			t.Fatal(err)
		}
		stream = append(stream, f...)
	}
	var r Reassembler
	var got [][]byte
	for _, p := range Chunks(stream, 23) {
		if len(p) > 20 {
			t.Fatalf("chunk of %d bytes for MTU 23", len(p))
		}
		m, err := r.Add(p)
		if err != nil {
			t.Fatal(err)
		}
		got = append(got, m...)
	}
	if len(got) != 3 || !bytes.Equal(got[1], msgs[1]) || len(got[2]) != 0 {
		t.Fatalf("reassembled %d messages", len(got))
	}
	if _, err := Frame(make([]byte, MaxMessage+1)); err != ErrTooLong {
		t.Error("framed a message over MaxMessage")
	}
	if _, err := r.Add([]byte{0xff, 0xff}); err != ErrTooLong {
		t.Error("accepted a length over MaxMessage")
	}
	if len(Chunks([]byte("abc"), 0)) != 1 {
		t.Error("tiny MTU")
	}
}
