package glrepeater

import (
	"context"
	"errors"
	"testing"
)

func TestParse(t *testing.T) {
	// shape of GL 4.9.1's repeater status (the stock UI's defaults + fields)
	s := Parse([]byte(`{"ssid":"Hotel-Guest","state":2,"state_s":"connected","portal":true,"portal_detecting":false,
	  "portal_url":"http://10.20.0.1/login?dst=x","bare_mode":false,"auto_portal":true,"ipv4":{"ip":"10.20.3.7","dns":["10.20.0.1"]}}`))
	if !s.Present || !s.Enabled || !s.Portal || s.LoginMode || s.PortalURL != "http://10.20.0.1/login?dst=x" {
		t.Fatalf("%+v", s)
	}
	s = Parse([]byte(`{"portal":true,"bare_mode":true,"auto_portal":false}`))
	if !s.Enabled || !s.LoginMode {
		t.Fatalf("login mode: %+v", s)
	}
	// feature present but off, nothing detected: not authoritative
	if s := Parse([]byte(`{"portal":false,"portal_detecting":false,"bare_mode":false,"auto_portal":false}`)); !s.Present || s.Enabled {
		t.Fatalf("off: %+v", s)
	}
	// older firmware without portal fields, or daemon still starting
	for _, out := range []string{`{"state":"initializing"}`, `{"ssid":"x","state":2}`, `not json`} {
		if s := Parse([]byte(out)); s.Present {
			t.Fatalf("%s: %+v", out, s)
		}
	}
}

func TestReadNoUbus(t *testing.T) {
	old := Run
	defer func() { Run = old }()
	Run = func(context.Context, string, ...string) ([]byte, error) { return nil, errors.New("not found") }
	if Read(context.Background()).Present {
		t.Fatal("present without ubus")
	}
}
