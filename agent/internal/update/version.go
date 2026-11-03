// Package update implements tailbond's update check (docs/architecture.md,
// "Updates"): version ordering, the GitHub "latest release" API, and
// downloads checked against the release's SHA256SUMS.
package update

import (
	"regexp"
	"strconv"
	"strings"
)

// Protocol is the agent protocol this build speaks (GET /api/hello).
const Protocol = 1

// DefaultRepo is where releases come from; BuildRepo (-ldflags -X, CI:
// github.repository) overrides it, update.repo in the config overrides both.
const DefaultRepo = "kaovilai/tailbond"

var BuildRepo string

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
	// an OpenWrt package revision at the end: "0.2.0-r1", "1.2.3-beta.1-r1"
	if m := reRevision.FindStringIndex(s); m != nil {
		s = s[:m[0]]
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
		// "-<n>-g<hash>" (git describe) and "-dirty" contain a "-" here
		// and fail below: such builds never update themselves
		ids := strings.Split(pre, ".")
		for _, id := range ids {
			if id == "" || id == "dirty" || strings.Contains(id, "-") {
				return Version{}, false
			}
		}
		v.Pre = ids
	}
	return v, true
}

var reRevision = regexp.MustCompile(`-r[0-9]+$`)

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
