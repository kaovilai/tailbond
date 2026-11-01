package update

import (
	"fmt"
	"strconv"
	"strings"
	"time"
)

const (
	PolicyAuto   = "auto"
	PolicyNotify = "notify"
	PolicyOff    = "off"

	ActionNone    = "none"
	ActionInstall = "install"
	ActionWait    = "wait"
	ActionBlocked = "blocked"
)

// Policy is the update part of the config, defaults filled in.
type Policy struct {
	Mode      string // auto | notify | off
	Pin       string
	Window    string // "03:00-05:00", local time; "" = any time (servers only)
	DeferKbps int
}

// Situation is everything the decision looks at.
type Situation struct {
	Now      time.Time
	Server   bool
	Current  string
	Manifest *Manifest
	Failed   []string
	// client: the active server's protocol (0 = unknown, taken as 1)
	ServerProtocol int
	// client: tunnel traffic, average over the last minute
	TrafficKbps int
	// the user pressed "Install now" (ignores policy, window and deferral,
	// not compatibility or a pin)
	Requested bool
}

type Decision struct {
	Action  string // none | install | wait | blocked
	Version string
	Reason  string
	// Code, for wait/blocked: busy | window | server | notify
	Code string
}

// Decide is the whole update policy (docs/architecture.md, "Updates").
func Decide(p Policy, s Situation) Decision {
	if p.Mode == PolicyOff && !s.Requested {
		return Decision{Action: ActionNone, Reason: "updates are off"}
	}
	m := s.Manifest
	if m == nil {
		return Decision{Action: ActionNone, Reason: "no manifest"}
	}
	if _, ok := Parse(s.Current); !ok {
		if !s.Requested {
			return Decision{Action: ActionNone, Version: m.Version, Reason: "this build isn't a release (" + s.Current + ")"}
		}
	}
	target := m.Version
	if p.Pin != "" {
		if Same(p.Pin, s.Current) {
			return Decision{Action: ActionNone, Reason: "pinned to " + p.Pin}
		}
		if !Same(p.Pin, m.Version) {
			return Decision{Action: ActionNone, Reason: fmt.Sprintf("pinned to %s; the channel offers %s", p.Pin, m.Version)}
		}
	} else if _, ok := Parse(s.Current); ok && !Newer(target, s.Current) {
		return Decision{Action: ActionNone, Version: target, Reason: "up to date"}
	}
	for _, f := range s.Failed {
		if Same(f, target) && !s.Requested {
			return Decision{Action: ActionNone, Version: target, Reason: target + " failed its health check here before"}
		}
	}
	if !s.Server {
		have := s.ServerProtocol
		if have == 0 {
			have = 1
		}
		if m.RequiresServerProtocol > have {
			return Decision{Action: ActionBlocked, Version: target, Reason: "waiting for the server to update first", Code: "server"}
		}
	}
	if s.Requested {
		return Decision{Action: ActionInstall, Version: target, Reason: "requested"}
	}
	if p.Mode == PolicyNotify {
		return Decision{Action: ActionWait, Version: target, Reason: "update available", Code: "notify"}
	}
	if s.Server {
		if ok, next := InWindow(p.Window, s.Now); !ok {
			return Decision{Action: ActionWait, Version: target, Reason: "waiting for " + next, Code: "window"}
		}
	} else if p.DeferKbps > 0 && s.TrafficKbps > p.DeferKbps && !m.Security {
		return Decision{Action: ActionWait, Version: target, Reason: "busy: a call or a stream is using the tunnel", Code: "busy"}
	}
	return Decision{Action: ActionInstall, Version: target}
}

// InWindow reports whether now (local time) is inside "HH:MM-HH:MM"
// (may wrap past midnight; "" or equal ends = always) and, if not, the
// start time.
func InWindow(window string, now time.Time) (bool, string) {
	window = strings.TrimSpace(window)
	if window == "" {
		return true, ""
	}
	a, b, ok := strings.Cut(window, "-")
	start, ok1 := clock(a)
	end, ok2 := clock(b)
	if !ok || !ok1 || !ok2 {
		return true, ""
	}
	if start == end { // "00:00-00:00": any time
		return true, ""
	}
	cur := now.Hour()*60 + now.Minute()
	in := false
	if start <= end {
		in = cur >= start && cur < end
	} else {
		in = cur >= start || cur < end
	}
	return in, strings.TrimSpace(a)
}

// ValidWindow checks "HH:MM-HH:MM" ("" is valid).
func ValidWindow(w string) bool {
	if strings.TrimSpace(w) == "" {
		return true
	}
	a, b, ok := strings.Cut(w, "-")
	_, ok1 := clock(a)
	_, ok2 := clock(b)
	return ok && ok1 && ok2
}

func clock(s string) (int, bool) {
	h, m, ok := strings.Cut(strings.TrimSpace(s), ":")
	if !ok {
		return 0, false
	}
	hh, err1 := strconv.Atoi(h)
	mm, err2 := strconv.Atoi(m)
	if err1 != nil || err2 != nil || hh < 0 || hh > 23 || mm < 0 || mm > 59 {
		return 0, false
	}
	return hh*60 + mm, true
}
