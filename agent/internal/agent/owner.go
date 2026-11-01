package agent

import (
	"context"
	"net/http"
	"sync"
	"time"

	"github.com/tailbond/tailbond/agent/internal/config"
	"github.com/tailbond/tailbond/agent/internal/tailscale"
)

// Without a dashboard password, a home server's admin API answers tailnet
// devices only when they belong to the Tailscale user who owns the server
// (people it's shared with, and other users on the tailnet, can still ask
// to connect, but not approve themselves). Loopback and other sources are
// unaffected; a tagged server (no owner) keeps the old rule (anyone who can
// reach it); a password replaces the rule.

type whoisCache struct {
	mu sync.Mutex
	m  map[string]whoisEntry
}

type whoisEntry struct {
	user string
	at   time.Time
}

const whoisTTL = time.Minute

// requesterUser is the Tailscale login of the device at ip ("" if unknown).
func (a *Agent) requesterUser(ctx context.Context, ip string) string {
	a.wc.mu.Lock()
	if e, ok := a.wc.m[ip]; ok && time.Since(e.at) < whoisTTL {
		a.wc.mu.Unlock()
		return e.user
	}
	a.wc.mu.Unlock()
	user := ""
	if w, err := a.whois(ctx, ip); err == nil {
		user = w.User
	}
	a.wc.mu.Lock()
	if a.wc.m == nil {
		a.wc.m = map[string]whoisEntry{}
	}
	a.wc.m[ip] = whoisEntry{user: user, at: time.Now()}
	a.wc.mu.Unlock()
	return user
}

// ownerOnly reports the owner who alone may use the admin API from r's
// source, or "" when r may (see above).
func (a *Agent) ownerOnly(r *http.Request) string {
	cfg := a.store.Get()
	if cfg.Role != config.RoleServer || cfg.UIPassword != "" {
		return ""
	}
	ip := remoteAddr(r)
	if !ip.IsValid() || !tailscale.IsTailnetIP(ip) {
		return ""
	}
	owner := a.ts.Get(r.Context()).Owner
	if owner == "" || a.requesterUser(r.Context(), ip.String()) == owner {
		return ""
	}
	return owner
}
