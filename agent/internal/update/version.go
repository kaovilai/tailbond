// Package update implements tailbond's signed auto-updates
// (docs/architecture.md, "Updates"): version ordering, the signed channel
// manifest, the install decision, the server's mirror cache and the Linux
// self-update with trial boots and rollback.
package update

import (
	"strconv"
	"strings"
)

// Protocol is the agent protocol this build speaks. Bump it when a client
// needs a newer server to work (manifest requires_server_protocol).
const Protocol = 1

// Build-time settings (-ldflags -X):
//   - BuildKeys: trusted ed25519 public keys, base64 of the raw 32 bytes,
//     comma separated (CI variable TB_UPDATE_PUBKEY);
//   - DefaultSource: the update source base URL when the config has none.
var (
	BuildKeys     string
	DefaultSource string
)

// Version is a parsed semver: MAJOR.MINOR.PATCH[-pre]. A leading "v" and an
// OpenWrt package revision ("-r96") are ignored.
type Version struct {
	Major, Minor, Patch int
	Pre                 []string
}

// Parse reads a version; ok is false for anything that isn't semver
// ("dev", a git hash, "-dirty" builds).
func Parse(s string) (Version, bool) {
	s = strings.TrimSpace(strings.TrimPrefix(strings.TrimSpace(s), "v"))
	if i := strings.IndexByte(s, '+'); i >= 0 {
		s = s[:i]
	}
	core, pre, _ := strings.Cut(s, "-")
	parts := strings.Split(core, ".")
	if len(parts) != 3 {
		return Version{}, false
	}
	var v Version
	for i, p := range parts {
		n, err := strconv.Atoi(p)
		if err != nil || n < 0 || p == "" {
			return Version{}, false
		}
		switch i {
		case 0:
			v.Major = n
		case 1:
			v.Minor = n
		default:
			v.Patch = n
		}
	}
	if pre != "" {
		ids := strings.Split(pre, ".")
		// an OpenWrt revision: "0.1.0-r96" (optionally after a pre-release)
		if last := ids[len(ids)-1]; isRevision(last) {
			ids = ids[:len(ids)-1]
		}
		for _, id := range ids {
			if id == "" || id == "dirty" || strings.Contains(id, "-") {
				return Version{}, false
			}
		}
		v.Pre = ids
	}
	return v, true
}

func isRevision(s string) bool {
	if len(s) < 2 || s[0] != 'r' {
		return false
	}
	_, err := strconv.Atoi(s[1:])
	return err == nil
}

// Compare orders two versions like semver (-1, 0, 1). A version with a
// pre-release sorts before the same version without one.
func Compare(a, b Version) int {
	for _, d := range [][2]int{{a.Major, b.Major}, {a.Minor, b.Minor}, {a.Patch, b.Patch}} {
		if d[0] != d[1] {
			if d[0] < d[1] {
				return -1
			}
			return 1
		}
	}
	switch {
	case len(a.Pre) == 0 && len(b.Pre) == 0:
		return 0
	case len(a.Pre) == 0:
		return 1
	case len(b.Pre) == 0:
		return -1
	}
	for i := 0; i < len(a.Pre) && i < len(b.Pre); i++ {
		x, y := a.Pre[i], b.Pre[i]
		xn, xerr := strconv.Atoi(x)
		yn, yerr := strconv.Atoi(y)
		switch {
		case xerr == nil && yerr == nil:
			if xn != yn {
				if xn < yn {
					return -1
				}
				return 1
			}
		case xerr == nil:
			return -1
		case yerr == nil:
			return 1
		case x != y:
			if x < y {
				return -1
			}
			return 1
		}
	}
	switch {
	case len(a.Pre) < len(b.Pre):
		return -1
	case len(a.Pre) > len(b.Pre):
		return 1
	}
	return 0
}

// Newer reports whether version string a is newer than b (false if either
// isn't semver).
func Newer(a, b string) bool {
	va, ok1 := Parse(a)
	vb, ok2 := Parse(b)
	return ok1 && ok2 && Compare(va, vb) > 0
}

// Same reports whether two version strings name the same release.
func Same(a, b string) bool {
	va, ok1 := Parse(a)
	vb, ok2 := Parse(b)
	return ok1 && ok2 && Compare(va, vb) == 0
}

func (v Version) String() string {
	s := strconv.Itoa(v.Major) + "." + strconv.Itoa(v.Minor) + "." + strconv.Itoa(v.Patch)
	if len(v.Pre) > 0 {
		s += "-" + strings.Join(v.Pre, ".")
	}
	return s
}
