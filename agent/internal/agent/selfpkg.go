package agent

import (
	"archive/tar"
	"bufio"
	"bytes"
	"compress/gzip"
	"io"
	"net/http"
	"os"
	"path/filepath"
	"strings"
	"time"
)

// The GL.iNet router package, served by a home server that runs on a GL
// router (Flint 3) itself: tailbond is installed there, so the agent can
// rebuild the .ipk from opkg's record of it (control files + file list).
// Files that change after install (the UCI config, the uci-defaults script
// that deletes itself) come from pristine copies the package ships under
// /usr/share/tailbond/pristine/. Anything under /etc/tailbond/ (keys,
// pairing) is never included.

var (
	opkgInfoDir = "/usr/lib/opkg/info"
	pkgRoot     = "/"
)

const pristineDir = "usr/share/tailbond/pristine"

// installedIPK rebuilds the installed tailbond package; ok is false when
// it isn't installed through opkg (Fedora, tests without fixtures).
func installedIPK() ([]byte, bool, error) {
	info := filepath.Join(opkgInfoDir, "tailbond")
	list, err := os.ReadFile(info + ".list")
	if err != nil {
		return nil, false, nil
	}
	ctl, err := os.ReadFile(info + ".control")
	if err != nil {
		return nil, false, nil
	}
	control := stripInstallFields(ctl)

	var ctlTar, dataTar bytes.Buffer
	if err := writeTarGz(&ctlTar, func(tw *tar.Writer) error {
		if err := addBytes(tw, "./control", control, 0o644); err != nil {
			return err
		}
		for _, f := range []string{"conffiles", "postinst", "prerm", "postrm", "preinst"} {
			if b, err := os.ReadFile(info + "." + f); err == nil {
				mode := int64(0o755)
				if f == "conffiles" {
					mode = 0o644
				}
				if err := addBytes(tw, "./"+f, b, mode); err != nil {
					return err
				}
			}
		}
		return nil
	}); err != nil {
		return nil, true, err
	}
	if err := writeTarGz(&dataTar, func(tw *tar.Writer) error {
		sc := bufio.NewScanner(bytes.NewReader(list))
		for sc.Scan() {
			p := strings.TrimSpace(strings.SplitN(sc.Text(), "\t", 2)[0])
			if p == "" || !strings.HasPrefix(p, "/") || strings.HasPrefix(p, "/etc/tailbond/") || strings.Contains(p, "..") {
				continue
			}
			rel := strings.TrimPrefix(p, "/")
			src := filepath.Join(pkgRoot, pristineDir, rel)
			if _, err := os.Stat(src); err != nil {
				src = filepath.Join(pkgRoot, rel)
			}
			st, err := os.Stat(src)
			if err != nil || st.IsDir() {
				continue // removed since install and no pristine copy
			}
			b, err := os.ReadFile(src)
			if err != nil {
				return err
			}
			if err := addBytes(tw, "./"+rel, b, int64(st.Mode().Perm())); err != nil {
				return err
			}
		}
		return sc.Err()
	}); err != nil {
		return nil, true, err
	}
	var out bytes.Buffer
	err = writeTarGz(&out, func(tw *tar.Writer) error {
		if err := addBytes(tw, "./debian-binary", []byte("2.0\n"), 0o644); err != nil {
			return err
		}
		if err := addBytes(tw, "./data.tar.gz", dataTar.Bytes(), 0o644); err != nil {
			return err
		}
		return addBytes(tw, "./control.tar.gz", ctlTar.Bytes(), 0o644)
	})
	return out.Bytes(), true, err
}

// stripInstallFields drops what opkg adds to an installed package's control.
func stripInstallFields(ctl []byte) []byte {
	var out bytes.Buffer
	for _, line := range strings.Split(strings.TrimRight(string(ctl), "\n"), "\n") {
		k, _, _ := strings.Cut(line, ":")
		switch k {
		case "Status", "Installed-Time", "Auto-Installed", "Conffiles":
			continue
		}
		out.WriteString(line + "\n")
	}
	return out.Bytes()
}

func writeTarGz(w io.Writer, fill func(*tar.Writer) error) error {
	gz := gzip.NewWriter(w)
	tw := tar.NewWriter(gz)
	if err := fill(tw); err != nil {
		return err
	}
	if err := tw.Close(); err != nil {
		return err
	}
	return gz.Close()
}

func addBytes(tw *tar.Writer, name string, b []byte, mode int64) error {
	if err := tw.WriteHeader(&tar.Header{Name: name, Mode: mode, Size: int64(len(b)), ModTime: time.Unix(0, 0), Format: tar.FormatUSTAR, Typeflag: tar.TypeReg}); err != nil {
		return err
	}
	_, err := tw.Write(b)
	return err
}

// glinetIPKAvailable: a package file is configured, or this is an opkg install.
func (a *Agent) glinetIPKAvailable() bool {
	if apkAvailable(a.store.Get().GLiNetIPKPath()) {
		return true
	}
	_, err := os.Stat(filepath.Join(opkgInfoDir, "tailbond.list"))
	return err == nil
}

// hIPK offers the GL.iNet router package (additive), for installing on a
// travel router from the computer that opened the home server's page.
func (a *Agent) hIPK(w http.ResponseWriter, r *http.Request) {
	const name = "tailbond_aarch64_cortex-a53_neon-vfpv4.ipk"
	if path := a.store.Get().GLiNetIPKPath(); apkAvailable(path) {
		serveDownload(w, r, path, name, "application/octet-stream")
		return
	}
	b, ok, err := installedIPK()
	if !ok {
		http.NotFound(w, r)
		return
	}
	if err != nil {
		httpErr(w, http.StatusInternalServerError, err)
		return
	}
	w.Header().Set("Content-Type", "application/octet-stream")
	w.Header().Set("Content-Disposition", `attachment; filename="`+name+`"`)
	http.ServeContent(w, r, name, time.Time{}, bytes.NewReader(b))
}
