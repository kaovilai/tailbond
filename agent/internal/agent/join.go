package agent

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"net"
	"net/http"
	"net/netip"
	"os"
	"strings"
	"sync"
	"time"

	"github.com/kaovilai/tailbond/agent/internal/config"
	"github.com/kaovilai/tailbond/agent/internal/pairing"
	"github.com/kaovilai/tailbond/agent/internal/tailscale"
)

// Client role: find tailbond servers among the router's tailnet peers and
// ask one to pair (docs/architecture.md, "Pairing over Tailscale").

type helloInfo struct {
	Tailbond bool   `json:"tailbond"`
	Name     string `json:"name"`
	Version  string `json:"version"`
	Role     string `json:"role"`
	Protocol int    `json:"protocol,omitempty"`
}

// foundServer is one entry of GET /api/discover.
type foundServer struct {
	Name    string `json:"name"`
	DNSName string `json:"dns_name,omitempty"`
	IP      string `json:"ip"`
	OS      string `json:"os,omitempty"`
	Version string `json:"version,omitempty"`
}

type discoveryResult struct {
	TailscaleInstalled bool          `json:"tailscale_installed"`
	TailscaleRunning   bool          `json:"tailscale_running"`
	Self               string        `json:"self,omitempty"`
	Tailnet            string        `json:"tailnet,omitempty"`
	Servers            []foundServer `json:"servers"`
	Checked            int           `json:"checked"`
	At                 time.Time     `json:"at"`
}

type discoveryCache struct {
	mu  sync.Mutex
	res *discoveryResult
}

const (
	discoverTTL     = 20 * time.Second
	discoverMax     = 64 // peers probed per scan
	joinGiveUpAfter = 15 * time.Minute
)

var joinPollEvery = 3 * time.Second

var httpClient = &http.Client{Timeout: 4 * time.Second, Transport: &http.Transport{Proxy: nil, DisableKeepAlives: true}}

func (a *Agent) fetchHello(ctx context.Context, addr string) (*helloInfo, error) {
	ctx, cancel := context.WithTimeout(ctx, 2*time.Second)
	defer cancel()
	req, _ := http.NewRequestWithContext(ctx, http.MethodGet, "http://"+addr+"/api/hello", nil)
	resp, err := httpClient.Do(req)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()
	if resp.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("hello: HTTP %d", resp.StatusCode)
	}
	var h helloInfo
	if err := json.NewDecoder(io.LimitReader(resp.Body, 4096)).Decode(&h); err != nil {
		return nil, err
	}
	if !h.Tailbond {
		return nil, errors.New("not a tailbond agent")
	}
	return &h, nil
}

// discover probes every online tailnet peer's :8077/api/hello.
func (a *Agent) discover(ctx context.Context, refresh bool) *discoveryResult {
	a.disc.mu.Lock()
	defer a.disc.mu.Unlock()
	if !refresh && a.disc.res != nil && time.Since(a.disc.res.At) < discoverTTL {
		return a.disc.res
	}
	if refresh {
		a.ts.Invalidate()
	}
	ti := a.ts.Get(ctx)
	res := &discoveryResult{TailscaleInstalled: ti.Installed, TailscaleRunning: ti.Running, Self: ti.Self, Tailnet: ti.Tailnet, Servers: []foundServer{}, At: time.Now()}
	if ti.Running {
		var wg sync.WaitGroup
		var mu sync.Mutex
		for _, p := range ti.PeerList {
			if !p.Online || res.Checked >= discoverMax {
				continue
			}
			ip := firstIPv4(p.TailscaleIPs)
			if ip == "" {
				continue
			}
			res.Checked++
			wg.Add(1)
			go func(p tailscale.Node, ip string) {
				defer wg.Done()
				h, err := a.hello(ctx, net.JoinHostPort(ip, a.apiPort))
				if err != nil || h.Role != config.RoleServer {
					return
				}
				name := h.Name
				if name == "" {
					name = p.HostName
				}
				mu.Lock()
				res.Servers = append(res.Servers, foundServer{Name: name, DNSName: p.DNSName, IP: ip, OS: p.OS, Version: h.Version})
				mu.Unlock()
			}(p, ip)
		}
		wg.Wait()
		sortServers(res.Servers)
	}
	a.disc.res = res
	return res
}

func sortServers(s []foundServer) {
	for i := 1; i < len(s); i++ {
		for j := i; j > 0 && strings.ToLower(s[j].Name) < strings.ToLower(s[j-1].Name); j-- {
			s[j], s[j-1] = s[j-1], s[j]
		}
	}
}

func firstIPv4(ips []string) string {
	for _, s := range ips {
		if ip, err := netip.ParseAddr(s); err == nil && ip.Is4() {
			return s
		}
	}
	return ""
}

func (a *Agent) hDiscover(w http.ResponseWriter, r *http.Request) {
	if a.role != config.RoleClient {
		httpErr(w, http.StatusBadRequest, errClientOnly)
		return
	}
	writeJSON(w, http.StatusOK, a.discover(r.Context(), r.URL.Query().Get("refresh") == "1"))
}

// joinState is the client's request to a server (guarded by Agent.mu).
type joinState struct {
	State      string    `json:"state"` // idle | asking | waiting | approved | declined | expired | error
	Server     string    `json:"server,omitempty"`
	ServerName string    `json:"server_name,omitempty"`
	ID         string    `json:"id,omitempty"`
	Name       string    `json:"name,omitempty"` // how the server will list us
	Error      string    `json:"error,omitempty"`
	Since      time.Time `json:"since,omitzero"`

	Add   bool   `json:"add,omitempty"` // add as another server
	Added string `json:"added,omitempty"`
	// error subnet_in_use / exists: the server already in the list
	Conflict string `json:"conflict,omitempty"`
	Subnet   string `json:"subnet,omitempty"`

	token  string
	cancel context.CancelFunc
}

const (
	joinIdle     = "idle"
	joinAsking   = "asking"
	joinWaiting  = "waiting"
	joinApproved = "approved"
	joinDeclined = "declined"
	joinExpired  = "expired"
	joinError    = "error"
)

func (a *Agent) joinView() *joinState {
	a.mu.Lock()
	defer a.mu.Unlock()
	j := *a.join
	return &j
}

func (a *Agent) hJoinGet(w http.ResponseWriter, r *http.Request) {
	if a.role != config.RoleClient {
		httpErr(w, http.StatusBadRequest, errClientOnly)
		return
	}
	writeJSON(w, http.StatusOK, a.joinView())
}

func (a *Agent) hJoinCancel(w http.ResponseWriter, r *http.Request) {
	if a.role != config.RoleClient {
		httpErr(w, http.StatusBadRequest, errClientOnly)
		return
	}
	a.mu.Lock()
	if a.join.cancel != nil {
		a.join.cancel()
	}
	a.join = &joinState{State: joinIdle}
	a.mu.Unlock()
	writeJSON(w, http.StatusOK, map[string]any{"ok": true})
}

// hJoin: POST /api/join {"server": "homebox" | "homebox.tailnet.ts.net" | "100.x.y.z"}.
func (a *Agent) hJoin(w http.ResponseWriter, r *http.Request) {
	if a.role != config.RoleClient {
		httpErr(w, http.StatusBadRequest, errClientOnly)
		return
	}
	var body struct {
		Server string `json:"server"`
		Name   string `json:"name"`
		Add    bool   `json:"add"`
	}
	if err := readJSON(r, &body); err != nil {
		httpErr(w, http.StatusBadRequest, err)
		return
	}
	target := strings.TrimSpace(body.Server)
	if target == "" || strings.ContainsAny(target, " /\\@?#") {
		httpErr(w, http.StatusBadRequest, errors.New("enter the home server's Tailscale name"))
		return
	}
	addr, label := a.resolveServer(r.Context(), target)
	name := body.Name
	if name == "" {
		name = a.selfName(r.Context())
	}
	ctx, cancel := context.WithCancel(context.Background())
	a.mu.Lock()
	if a.join.cancel != nil {
		a.join.cancel()
	}
	add := body.Add && a.client.payload != nil
	a.join = &joinState{State: joinAsking, Server: addr, ServerName: label, Name: clientName(name), Since: time.Now(), Add: add, cancel: cancel}
	a.mu.Unlock()
	go a.runJoin(ctx, addr, clientName(name), add)
	writeJSON(w, http.StatusAccepted, a.joinView())
}

// resolveServer maps a typed name to host:port using the tailnet peer list
// (short name or MagicDNS name), else uses it as given.
func (a *Agent) resolveServer(ctx context.Context, target string) (addr, label string) {
	host, port, err := net.SplitHostPort(target)
	if err != nil {
		host, port = target, a.apiPort
	}
	low := strings.ToLower(strings.TrimSuffix(host, "."))
	for _, p := range a.ts.Get(ctx).PeerList {
		short, _, _ := strings.Cut(p.DNSName, ".")
		if low == strings.ToLower(p.HostName) || low == strings.ToLower(p.DNSName) || low == strings.ToLower(short) {
			if ip := firstIPv4(p.TailscaleIPs); ip != "" {
				label = short
				if label == "" {
					label = p.HostName
				}
				return net.JoinHostPort(ip, port), label
			}
		}
	}
	short, _, _ := strings.Cut(host, ".")
	return net.JoinHostPort(host, port), short
}

func (a *Agent) selfName(ctx context.Context) string {
	if n := a.ts.Get(ctx).Self; n != "" {
		return n
	}
	h, _ := os.Hostname()
	return h
}

func (a *Agent) setJoin(fn func(j *joinState)) {
	a.mu.Lock()
	fn(a.join)
	a.mu.Unlock()
}

// joinFinal records a final state unless the join was replaced or cancelled.
func (a *Agent) joinFinal(ctx context.Context, state, msg string) {
	a.mu.Lock()
	defer a.mu.Unlock()
	if ctx.Err() != nil {
		return
	}
	a.join.State, a.join.Error, a.join.cancel = state, msg, nil
}

func (a *Agent) runJoin(ctx context.Context, addr, name string, add bool) {
	req := map[string]any{"name": name, "kind": "router"}
	// the subnets this client's servers use, so a fresh server picks another
	var avoid []string
	for _, e := range a.store.Get().Client.Servers {
		if p, err := pairing.Decode(e.PairingURI); err == nil {
			avoid = append(avoid, p.TunnelSubnet().String())
		}
	}
	if len(avoid) > 0 {
		req["avoid_subnets"] = avoid
	}
	body, _ := json.Marshal(req)
	var created struct {
		ID     string `json:"id"`
		Token  string `json:"token"`
		Status string `json:"status"`
		Name   string `json:"name"`
		Error  string `json:"error"`
	}
	code, err := a.joinCall(ctx, http.MethodPost, "http://"+addr+"/api/pair-requests", "", body, &created)
	switch {
	case err != nil:
		a.Logf("ask %s to pair: %v", addr, err)
		a.joinFinal(ctx, joinError, "unreachable")
		return
	case code == http.StatusForbidden:
		a.joinFinal(ctx, joinError, "not_tailnet")
		return
	case code == http.StatusTooManyRequests:
		a.joinFinal(ctx, joinError, "too_many")
		return
	case code == http.StatusNotFound || code == http.StatusBadRequest:
		a.joinFinal(ctx, joinError, "not_server")
		return
	case code != http.StatusAccepted || created.ID == "":
		a.joinFinal(ctx, joinError, "server_error")
		return
	}
	a.setJoin(func(j *joinState) {
		if ctx.Err() == nil {
			j.State, j.ID, j.token, j.Name = joinWaiting, created.ID, created.Token, created.Name
		}
	})
	a.Logf("asked %s to pair as %s (request %s); waiting for approval", addr, created.Name, created.ID)
	deadline := time.Now().Add(joinGiveUpAfter)
	t := time.NewTicker(joinPollEvery)
	defer t.Stop()
	fails := 0
	for {
		select {
		case <-ctx.Done():
			return
		case <-t.C:
		}
		if time.Now().After(deadline) {
			a.joinFinal(ctx, joinExpired, "")
			return
		}
		var st struct {
			Status string `json:"status"`
			URI    string `json:"uri"`
		}
		code, err := a.joinCall(ctx, http.MethodGet, "http://"+addr+"/api/pair-requests/"+created.ID, created.Token, nil, &st)
		if err != nil || code >= 500 {
			if fails++; fails > 20 {
				a.joinFinal(ctx, joinError, "unreachable")
				return
			}
			continue
		}
		fails = 0
		switch {
		case code == http.StatusNotFound:
			// server restarted (requests are in memory) or the request is gone
			a.joinFinal(ctx, joinExpired, "")
			return
		case st.Status == reqDeclined:
			a.joinFinal(ctx, joinDeclined, "")
			return
		case st.Status == reqExpired:
			a.joinFinal(ctx, joinExpired, "")
			return
		case st.Status == reqApproved && st.URI != "":
			var err error
			added := ""
			if add {
				added, err = a.addServer(st.URI, "")
			} else {
				_, err = a.importPairing(st.URI)
			}
			var ae *addServerError
			if errors.As(err, &ae) {
				a.Logf("pairing from %s: %v", addr, err)
				a.setJoin(func(j *joinState) { j.Conflict, j.Subnet = ae.conflict, ae.subnet })
				a.joinFinal(ctx, joinError, ae.code)
				return
			}
			if err != nil {
				a.Logf("pairing from %s: %v", addr, err)
				a.joinFinal(ctx, joinError, "bad_pairing")
				return
			}
			a.setJoin(func(j *joinState) { j.Added = added })
			a.joinFinal(ctx, joinApproved, "")
			return
		}
	}
}

func (a *Agent) joinCall(ctx context.Context, method, url, token string, body []byte, out any) (int, error) {
	var rd io.Reader
	if body != nil {
		rd = bytes.NewReader(body)
	}
	req, err := http.NewRequestWithContext(ctx, method, url, rd)
	if err != nil {
		return 0, err
	}
	if body != nil {
		req.Header.Set("Content-Type", "application/json")
	}
	if token != "" {
		req.Header.Set("Authorization", "Bearer "+token)
	}
	resp, err := httpClient.Do(req)
	if err != nil {
		return 0, err
	}
	defer resp.Body.Close()
	_ = json.NewDecoder(io.LimitReader(resp.Body, 64<<10)).Decode(out)
	return resp.StatusCode, nil
}
