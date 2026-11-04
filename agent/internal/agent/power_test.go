package agent

import (
	"testing"

	"github.com/kaovilai/tailbond/agent/internal/config"
)

func TestParsePmset(t *testing.T) {
	// `pmset -g assertions` (macOS 15) with caffeinate -i -s running
	held := `2026-11-04 10:00:00 -0500
Assertion status system-wide:
   BackgroundTask                 0
   ApplePushServiceTask           0
   UserIsActive                   1
   PreventUserIdleDisplaySleep    0
   PreventSystemSleep             1
   ExternalMedia                  0
   PreventUserIdleSystemSleep     1
   NetworkClientActive            0
Listed by owning process:
   pid 4242(caffeinate): [0x0001] 00:00:05 PreventUserIdleSystemSleep named: "caffeinate command-line tool"
`
	if !parsePmsetAssertions(held) {
		t.Fatal("caffeinate's assertion not seen")
	}
	none := "Assertion status system-wide:\n   PreventSystemSleep             0\n   PreventUserIdleSystemSleep     0\n"
	if parsePmsetAssertions(none) {
		t.Fatal("no assertion held, but parsed as awake")
	}
	laptop, batt := parsePmsetBatt("Now drawing from 'Battery Power'\n -InternalBattery-0 (id=1)\t80%; discharging")
	if !laptop || !batt {
		t.Fatalf("battery: laptop=%v onBattery=%v", laptop, batt)
	}
	laptop, batt = parsePmsetBatt("Now drawing from 'AC Power'\n")
	if laptop || batt {
		t.Fatalf("desktop: laptop=%v onBattery=%v", laptop, batt)
	}
}

func TestHealthSleep(t *testing.T) {
	get := func(p *powerInfo) *healthCheck {
		v := buildHealth(healthInputs{Role: config.RoleServer, Power: p})
		for i := range v.Checks {
			if v.Checks[i].ID == "sleep" {
				return &v.Checks[i]
			}
		}
		return nil
	}
	if c := get(nil); c != nil {
		t.Fatalf("no power info (Linux) but a sleep check: %+v", c)
	}
	for _, tc := range []struct {
		p    powerInfo
		want string
	}{
		{powerInfo{Awake: true}, hOK},
		{powerInfo{Awake: true, Laptop: true}, hOK},
		{powerInfo{Awake: true, Laptop: true, OnBattery: true}, hBad},
		{powerInfo{}, hBad},
	} {
		c := get(&tc.p)
		if c == nil || c.Status != tc.want {
			t.Errorf("%+v: got %+v, want %s", tc.p, c, tc.want)
		}
	}
	if c := get(&powerInfo{Awake: true, Laptop: true}); c.Detail == "" {
		t.Error("a laptop gets no lid note")
	}
	// the client role never shows it
	if v := buildHealth(healthInputs{Role: config.RoleClient, Power: &powerInfo{}}); len(v.Checks) > 1 {
		for _, c := range v.Checks {
			if c.ID == "sleep" {
				t.Fatal("sleep check on a client")
			}
		}
	}
}
