package agent

import "strings"

// A Mac that sleeps can't serve (docs/macos.md "Mac as home server"). The
// server-up hook holds a sleep assertion (caffeinate) while the server
// runs; Setup health says whether one is held and what else is needed.
type powerInfo struct {
	Awake     bool // some process holds a "don't sleep" assertion
	Laptop    bool // has an internal battery (a lid that sleeps it)
	OnBattery bool
}

// parsePmsetAssertions reads `pmset -g assertions`: whether the system is
// kept from (idle) sleep right now.
func parsePmsetAssertions(out string) bool {
	for _, l := range strings.Split(out, "\n") {
		f := strings.Fields(l)
		if len(f) == 2 && (f[0] == "PreventUserIdleSystemSleep" || f[0] == "PreventSystemSleep") && f[1] == "1" {
			return true
		}
	}
	return false
}

// parsePmsetBatt reads `pmset -g batt`.
func parsePmsetBatt(out string) (laptop, onBattery bool) {
	return strings.Contains(out, "InternalBattery"), strings.Contains(out, "'Battery Power'")
}

func powerCheck(p *powerInfo) healthCheck {
	const lid = "A MacBook still sleeps when its lid is closed, unless it's connected to power, an external display and a keyboard (closed-display mode)."
	switch {
	case !p.Awake:
		return healthCheck{ID: "sleep", Status: hBad, Title: "This Mac can go to sleep, and then nothing can connect",
			Detail: "tailbond keeps the Mac awake while its server runs; right now nothing does.",
			Fix:    "Turn tailbond's server on, or in System Settings → Energy turn on \"Prevent automatic sleeping when the display is off\"."}
	case p.Laptop && p.OnBattery:
		return healthCheck{ID: "sleep", Status: hBad, Title: "This MacBook is on battery: it sleeps when the battery runs low or the lid closes",
			Detail: lid, Fix: "Keep it plugged in with the lid open."}
	case p.Laptop:
		return healthCheck{ID: "sleep", Status: hOK, Title: "This Mac stays awake while it serves", Detail: lid}
	}
	return healthCheck{ID: "sleep", Status: hOK, Title: "This Mac stays awake while it serves",
		Detail: "tailbond holds a sleep assertion while the server runs; the display can still turn off."}
}
