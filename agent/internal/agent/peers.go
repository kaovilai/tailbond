package agent

import (
	"context"
	"crypto/hmac"
	"crypto/sha256"
	"crypto/subtle"
	"encoding/hex"
	"errors"
	"fmt"
	"io"
	"net"
	"net/http"
	"net/netip"
	"os"
	"strings"
	"time"

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
	if a.role == config.RoleClient {
		a.hPeersViaServer(w, r)
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

// hPeersViaServer: on the client role (travel router), GET /api/peers is
// the server's list, fetched through the tunnel with this router's client
// token (what LAN devices can reach). Cached for 20 s.
func (a *Agent) hPeersViaServer(w http.ResponseWriter, r *http.Request) {
	a.mu.Lock()
	p := a.client.payload
	ready := a.rt.ready
	cached, at := a.client.peers, a.client.peersAt
	a.mu.Unlock()
	if p == nil {
		httpErr(w, http.StatusConflict, errors.New("not paired"))
		return
	}
	if cached != nil && time.Since(at) < 20*time.Second {
		w.Header().Set("Content-Type", "application/json")
		_, _ = w.Write(cached)
		return
	}
	if !ready {
		httpErr(w, http.StatusServiceUnavailable, errors.New("not connected"))
		return
	}
	// the active server's tunnel address (its DNS IP when it has one)
	host := p.TunnelSubnet().Addr().Next().String()
	if len(p.DNS) > 0 {
		host = p.DNS[0]
	}
	ctx, cancel := context.WithTimeout(r.Context(), 5*time.Second)
	defer cancel()
	req, _ := http.NewRequestWithContext(ctx, http.MethodGet, "http://"+net.JoinHostPort(host, a.apiPort)+"/api/peers", nil)
	req.Header.Set("Authorization", "Bearer "+APIToken(p.Key))
	resp, err := httpClient.Do(req)
	if err != nil {
		httpErr(w, http.StatusBadGateway, errors.New("home server didn't answer"))
		return
	}
	defer resp.Body.Close()
	b, err := io.ReadAll(io.LimitReader(resp.Body, 1<<20))
	if err != nil || resp.StatusCode != http.StatusOK {
		httpErr(w, http.StatusBadGateway, fmt.Errorf("home server: HTTP %d", resp.StatusCode))
		return
	}
	a.mu.Lock()
	a.client.peers, a.client.peersAt = b, time.Now()
	a.mu.Unlock()
	w.Header().Set("Content-Type", "application/json")
	_, _ = w.Write(b)
}
