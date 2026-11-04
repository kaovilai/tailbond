package agent

import (
	"context"
	"path/filepath"
	"testing"

	"github.com/kaovilai/tailbond/agent/internal/config"
	"github.com/kaovilai/tailbond/agent/internal/tailscale"
)

func serverAgent(t *testing.T, disableTS bool) *Agent {
	t.Helper()
	cfg := config.Default()
	cfg.Role = config.RoleServer
	cfg.StateDir = t.TempDir()
	cfg.DisableTailscale = disableTS
	cfg.ApplyDefaults()
	return New(config.NewStore(filepath.Join(t.TempDir(), "tailbond.json"), cfg), Options{})
}

// --no-tailscale (disable_tailscale): the server hooks leave Tailscale
// alone, route advertisement included (PR 19 review).
func TestServerHookEnvTailscale(t *testing.T) {
	on := serverAgent(t, false).hookEnvLocked()
	if on["TB_TAILSCALE"] != "1" || on["TB_ADVERTISE_ROUTE"] != "1" {
		t.Fatalf("tailscale on: %v", on)
	}
	off := serverAgent(t, true).hookEnvLocked()
	if off["TB_TAILSCALE"] != "0" || off["TB_ADVERTISE_ROUTE"] != "0" {
		t.Fatalf("disable_tailscale: TB_TAILSCALE=%q TB_ADVERTISE_ROUTE=%q", off["TB_TAILSCALE"], off["TB_ADVERTISE_ROUTE"])
	}
}

// tailscaleSig changes when Tailscale comes up after the server (the
// macOS watcher then runs server-up again).
func TestTailscaleSig(t *testing.T) {
	ctx := context.Background()
	if s := serverAgent(t, true).tailscaleSig(ctx); s != "off" {
		t.Fatalf("disabled: %q", s)
	}
	a := serverAgent(t, false)
	a.ts = tailscale.NewFake(&tailscale.Info{Installed: true, Running: false})
	before := a.tailscaleSig(ctx)
	if before != "none" {
		t.Fatalf("not running: %q", before)
	}
	a.ts = tailscale.NewFake(&tailscale.Info{Installed: true, Running: true, IPs: []string{"fd7a:115c:a1e0::1", "100.101.0.1"}})
	after := a.tailscaleSig(ctx)
	if after == before || after[:len("100.101.0.1@")] != "100.101.0.1@" {
		t.Fatalf("running: %q (before %q)", after, before)
	}
}
