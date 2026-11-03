package agent

import (
	"archive/tar"
	"bytes"
	"compress/gzip"
	"io"
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func untarMap(t *testing.T, b []byte) map[string]string {
	t.Helper()
	gz, err := gzip.NewReader(bytes.NewReader(b))
	if err != nil {
		t.Fatal(err)
	}
	tr := tar.NewReader(gz)
	out := map[string]string{}
	for {
		h, err := tr.Next()
		if err == io.EOF {
			return out
		}
		if err != nil {
			t.Fatal(err)
		}
		c, _ := io.ReadAll(tr)
		out[h.Name] = string(c)
	}
}

func TestInstalledIPK(t *testing.T) {
	root, info := t.TempDir(), t.TempDir()
	oldRoot, oldInfo := pkgRoot, opkgInfoDir
	pkgRoot, opkgInfoDir = root, info
	t.Cleanup(func() { pkgRoot, opkgInfoDir = oldRoot, oldInfo })

	write := func(base, p, s string) {
		full := filepath.Join(base, p)
		os.MkdirAll(filepath.Dir(full), 0o755)
		if err := os.WriteFile(full, []byte(s), 0o755); err != nil {
			t.Fatal(err)
		}
	}
	if _, ok, _ := installedIPK(); ok {
		t.Fatal("ok without opkg records")
	}
	write(info, "tailbond.control", "Package: tailbond\nVersion: 0.1.0-r42\nArchitecture: aarch64_cortex-a53_neon-vfpv4\nStatus: install user installed\nInstalled-Time: 1793500000\n")
	write(info, "tailbond.postinst", "#!/bin/sh\nexit 0\n")
	write(info, "tailbond.list", "/usr/bin/tailbondd\n/etc/config/tailbond\n/etc/uci-defaults/90-tailbond\n/etc/tailbond/secret\n/usr/share/tailbond/pristine/etc/config/tailbond\n")
	write(root, "usr/bin/tailbondd", "BINARY")
	write(root, "etc/config/tailbond", "option role 'server'\noption public_addr 'home.example.net'\n")
	write(root, "etc/tailbond/secret", "KEY")
	write(root, "usr/share/tailbond/pristine/etc/config/tailbond", "option role 'client'\n")
	write(root, "usr/share/tailbond/pristine/etc/uci-defaults/90-tailbond", "#!/bin/sh\n")

	b, ok, err := installedIPK()
	if !ok || err != nil {
		t.Fatalf("ok=%v err=%v", ok, err)
	}
	outer := untarMap(t, b)
	if outer["./debian-binary"] != "2.0\n" {
		t.Fatalf("outer = %v", outer)
	}
	ctl := untarMap(t, []byte(outer["./control.tar.gz"]))
	if c := ctl["./control"]; !strings.Contains(c, "Version: 0.1.0-r42") || strings.Contains(c, "Status:") || strings.Contains(c, "Installed-Time") {
		t.Fatalf("control = %q", c)
	}
	if ctl["./postinst"] == "" {
		t.Fatal("postinst missing")
	}
	data := untarMap(t, []byte(outer["./data.tar.gz"]))
	if data["./usr/bin/tailbondd"] != "BINARY" {
		t.Fatal("binary missing")
	}
	if data["./etc/config/tailbond"] != "option role 'client'\n" {
		t.Fatalf("config not pristine: %q", data["./etc/config/tailbond"])
	}
	if data["./etc/uci-defaults/90-tailbond"] == "" {
		t.Fatal("removed uci-defaults not restored from pristine copy")
	}
	if _, leaked := data["./etc/tailbond/secret"]; leaked {
		t.Fatal("state leaked into the package")
	}
}
