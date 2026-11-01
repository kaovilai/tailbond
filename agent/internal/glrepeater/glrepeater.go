// Package glrepeater reads GL.iNet firmware's own repeater captive-portal
// state, so tailbond follows GL instead of probing the repeater itself.
// This is the only place that knows GL internals (docs/glinet.md,
// "Captive portals and GL's Login Mode"); fix it here if a firmware differs.
//
// What it relies on, from GL 4.9.1 firmware (BE14000 dump; Slate 7 / Flint 3
// not checked on hardware):
//   - gl-repeater registers the ubus object "repeater"; `ubus call repeater
//     status` returns a flat object with "portal" (portal detected),
//     "portal_detecting", "portal_url", "bare_mode" (Login Mode for Public
//     Hotspots is on), "auto_portal" (enter it automatically) and "state";
//   - the repeater uplink is the network "wwan";
//   - GL probes captive.apple.com / msftconnecttest.com bound to the wwan
//     device with wwan's DNS, and leaves Login Mode by itself after a
//     successful sign-in.
package glrepeater

import (
	"context"
	"encoding/json"
	"os/exec"
	"time"
)

// Network is the OpenWrt network GL's repeater feature owns.
const Network = "wwan"

// State is GL's view of the repeater uplink.
type State struct {
	Present   bool   // the repeater ubus object answered with portal fields
	Enabled   bool   // GL's portal handling is active for this uplink
	Portal    bool   // a captive portal was detected and not passed yet
	Detecting bool   // GL is checking right now
	PortalURL string // where GL would send the user
	LoginMode bool   // "Login Mode for Public Hotspots" (bare mode) is on
}

// Run executes a command (tests replace it).
var Run = func(ctx context.Context, name string, args ...string) ([]byte, error) {
	ctx, cancel := context.WithTimeout(ctx, 3*time.Second)
	defer cancel()
	return exec.CommandContext(ctx, name, args...).Output()
}

type status struct {
	Portal          *bool  `json:"portal"`
	PortalDetecting *bool  `json:"portal_detecting"`
	PortalURL       string `json:"portal_url"`
	BareMode        *bool  `json:"bare_mode"`
	AutoPortal      *bool  `json:"auto_portal"`
	State           any    `json:"state"`
}

func isTrue(b *bool) bool { return b != nil && *b }

// Read asks GL. Present is false when GL's repeater object or its portal
// fields aren't there (non-GL OpenWrt, older firmware): use tailbond's probe.
func Read(ctx context.Context) State {
	out, err := Run(ctx, "ubus", "call", "repeater", "status")
	if err != nil {
		return State{}
	}
	return Parse(out)
}

// Parse decodes `ubus call repeater status`.
func Parse(out []byte) State {
	var s status
	if json.Unmarshal(out, &s) != nil || s.Portal == nil {
		return State{}
	}
	st := State{
		Present:   true,
		Portal:    isTrue(s.Portal),
		Detecting: isTrue(s.PortalDetecting),
		PortalURL: s.PortalURL,
		LoginMode: isTrue(s.BareMode),
	}
	// GL detects on interface-up when auto_portal is set; a portal flag,
	// a running check or Login Mode also show the feature at work.
	st.Enabled = isTrue(s.AutoPortal) || st.Portal || st.Detecting || st.LoginMode
	return st
}
