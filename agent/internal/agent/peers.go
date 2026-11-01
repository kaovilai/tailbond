package agent

import (
	"crypto/hmac"
	"crypto/sha256"
	"crypto/subtle"
	"encoding/hex"
	"net"
	"net/http"
	"net/netip"
	"os"
	"strings"

	"github.com/tailbond/tailbond/agent/internal/config"
	"github.com/tailbond/tailbond/agent/internal/tailscale"
)

// apiTokenLabel is the HMAC message for the per-client API token
// (docs/architecture.md, "Client API token").
const apiTokenLabel = "tailbond-api"

// APIToken is the bearer token a paired client derives from its pairing key:
// lowercase hex of HMAC-SHA256(key = the client's key, message "tailbond-api").
func APIToken(userKey string) string {
	m := hmac.New(sha256.New, []byte(userKey))
	m.Write([]byte(apiTokenLabel))
	return hex.EncodeToString(m.Sum(nil))
}

// clientForToken returns the registered client whose derived token is in
// the request's Authorization header, or "".
func (a *Agent) clientForToken(r *http.Request) string {
	auth := r.Header.Get("Authorization")
	tok, ok := strings.CutPrefix(auth, "Bearer ")
	if !ok || tok == "" {
		return ""
	}
	tok = strings.TrimSpace(tok)
	found := ""
	for _, u := range a.store.Get().Server.Users {
		if u.Key == "" {
			continue
		}
		// compare against every user so timing doesn't reveal the match position
		if subtle.ConstantTimeCompare([]byte(tok), []byte(APIToken(u.Key))) == 1 && found == "" {
			found = u.Name
		}
	}
	return found
}

// viaTunnel reports whether r arrived on a tunnel address of the server
// (it came from a bonded client). Such requests only get GET /api/peers.
func (a *Agent) viaTunnel(r *http.Request) bool {
	if a.role != config.RoleServer {
		return false
	}
	la, ok := r.Context().Value(http.LocalAddrContextKey).(net.Addr)
	if !ok {
		return false
	}
	ap, err := netip.ParseAddrPort(la.String())
	if err != nil {
		return false
	}
	ip := ap.Addr().Unmap()
	cfg := a.store.Get()
	for _, s := range []string{cfg.Server.Subnet, cfg.Server.Subnet6} {
		if p, err := netip.ParsePrefix(s); err == nil && p.Contains(ip) {
			return true
		}
	}
	return false
}

type peersView struct {
	Server           string           `json:"server"`
	Tailnet          string           `json:"tailnet,omitempty"`
	MagicDNSSuffix   string           `json:"magic_dns_suffix,omitempty"`
	TailscaleRunning bool             `json:"tailscale_running"`
	Self             *tailscale.Node  `json:"self,omitempty"`
	Peers            []tailscale.Node `json:"peers"`
	Client           string           `json:"client,omitempty"`
}

// hPeers lists the tailnet machines bonded clients reach through this server
// (a summary of `tailscale status --json`).
func (a *Agent) hPeers(w http.ResponseWriter, r *http.Request) {
	if a.role != config.RoleServer {
		httpErr(w, http.StatusBadRequest, errServerOnly)
		return
	}
	ti := a.ts.Get(r.Context())
	v := peersView{
		Server:           ti.Self,
		Tailnet:          ti.Tailnet,
		MagicDNSSuffix:   ti.MagicDNSSuffix,
		TailscaleRunning: ti.Running,
		Self:             ti.SelfNode,
		Peers:            ti.PeerList,
		Client:           a.clientForToken(r),
	}
	if v.Server == "" {
		v.Server, _ = os.Hostname()
	}
	if v.Peers == nil {
		v.Peers = []tailscale.Node{}
	}
	writeJSON(w, http.StatusOK, v)
}
