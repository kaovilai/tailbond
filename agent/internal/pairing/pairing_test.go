package pairing

import (
	"strings"
	"testing"
)

func sample() *Payload {
	return &Payload{
		Name:          "pixel",
		Server:        "203.0.113.7:443",
		SNI:           "homebox.tail1234.ts.net",
		User:          "pixel",
		Key:           "c2VjcmV0LWtleS0xMjM0NTY3ODkwYWJjZGVmZ2hpams=",
		DNS:           []string{"10.77.0.1"},
		Mode:          "speed",
		TailnetRoutes: []string{"100.64.0.0/10", "fd7a:115c:a1e0::/48"},
	}
}

func TestRoundTrip(t *testing.T) {
	in := sample()
	uri, err := Encode(in)
	if err != nil {
		t.Fatal(err)
	}
	if !strings.HasPrefix(uri, "tailbond://pair?d=") {
		t.Fatalf("bad prefix: %s", uri)
	}
	if strings.ContainsAny(uri[len("tailbond://pair?d="):], "+/=") {
		t.Fatalf("payload is not unpadded base64url: %s", uri)
	}
	out, err := Decode(uri)
	if err != nil {
		t.Fatal(err)
	}
	if out.V != 1 || out.Name != in.Name || out.Server != in.Server || out.SNI != in.SNI ||
		out.Key != in.Key || out.Mode != in.Mode || len(out.TailnetRoutes) != 2 || out.DNS[0] != "10.77.0.1" {
		t.Fatalf("round trip mismatch: %+v", out)
	}
}

func TestRoundTripInsecurePinned(t *testing.T) {
	in := sample()
	in.Insecure = true
	in.CertSHA256 = strings.Repeat("ab", 32)
	uri, err := Encode(in)
	if err != nil {
		t.Fatal(err)
	}
	out, err := Decode("  " + uri + "\n")
	if err != nil {
		t.Fatal(err)
	}
	if !out.Insecure || out.CertSHA256 != in.CertSHA256 {
		t.Fatalf("pin lost: %+v", out)
	}
}

func TestDecodePadded(t *testing.T) {
	uri, _ := Encode(sample())
	if _, err := Decode(uri + "=="); err != nil {
		t.Fatalf("padded payload rejected: %v", err)
	}
}

func TestRejects(t *testing.T) {
	bad := []string{
		"",
		"https://pair?d=e30",
		"tailbond://pair",
		"tailbond://pair?d=!!!",
		"tailbond://pair?d=e30", // {} -> version 0
	}
	for _, u := range bad {
		if _, err := Decode(u); err == nil {
			t.Errorf("Decode(%q) accepted", u)
		}
	}
	p := sample()
	p.Insecure = true
	if _, err := Encode(p); err == nil {
		t.Error("insecure payload without pin accepted")
	}
	p = sample()
	p.Server = "no-port"
	if _, err := Encode(p); err == nil {
		t.Error("server without port accepted")
	}
}

func TestQRSVG(t *testing.T) {
	uri, _ := Encode(sample())
	svg, err := QRSVG(uri)
	if err != nil {
		t.Fatal(err)
	}
	if !strings.HasPrefix(svg, "<svg") || !strings.Contains(svg, "<path") || !strings.HasSuffix(svg, "</svg>") {
		t.Fatalf("unexpected svg: %.80s", svg)
	}
}
