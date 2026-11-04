package agent

import (
	"net"
	"net/http"
	"net/netip"

	"github.com/kaovilai/tailbond/agent/internal/config"
	"github.com/kaovilai/tailbond/agent/internal/tailscale"
)

// sourceGuard: on a server, a listener bound to a Tailscale address or the
// tunnel address answers only loopback, tailnet and tunnel-subnet sources,
// whatever the firewall does. A host that routes the server's Tailscale
// address through its LAN address (most OSes accept that: the weak host
// model) is refused here, also while mqvpn and its firewall hooks are
// stopped and the listeners keep running. Listeners on other addresses (an
// explicit listen host, a wildcard) are left as configured.
func (a *Agent) sourceGuard(addr string, h http.Handler) http.Handler {
	host, _, err := net.SplitHostPort(addr)
	if err != nil || a.role != config.RoleServer {
		return h
	}
	ip, err := netip.ParseAddr(host)
	if err != nil || !(tailscale.IsTailnetIP(ip) || ip.String() == a.store.Get().ServerTunnelIP()) {
		return h
	}
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if !a.allowedSource(remoteAddr(r)) {
			http.Error(w, "only reachable from your tailnet", http.StatusForbidden)
			return
		}
		h.ServeHTTP(w, r)
	})
}

// allowedSource: loopback, tailnet, or this server's tunnel subnets.
func (a *Agent) allowedSource(ip netip.Addr) bool {
	if !ip.IsValid() {
		return false
	}
	if ip.IsLoopback() || tailscale.IsTailnetIP(ip) {
		return true
	}
	cfg := a.store.Get()
	for _, s := range []string{cfg.Server.Subnet, cfg.Server.Subnet6} {
		if p, err := netip.ParsePrefix(s); err == nil && p.Contains(ip) {
			return true
		}
	}
	return false
}
