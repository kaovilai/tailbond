package agent

import (
	"context"
	"crypto/rand"
	"crypto/subtle"
	"errors"
	"fmt"
	"github.com/tailbond/tailbond/agent/internal/update"
	"net/http"
	"net/netip"
	"os"
	"regexp"
	"sort"
	"strings"
	"time"

	"github.com/tailbond/tailbond/agent/internal/config"
	"github.com/tailbond/tailbond/agent/internal/mqconf"
	"github.com/tailbond/tailbond/agent/internal/tailscale"
)

// Pair requests: a device on the tailnet asks the server to pair; the admin
// approves or declines it in the dashboard; the device polls for the result
// (docs/architecture.md, "Pairing over Tailscale"). Kept in memory only:
// a restart drops pending requests and the device simply asks again.
const (
	pairReqTTL        = 10 * time.Minute // pending requests expire
	pairReqPickupTTL  = 10 * time.Minute // approved: the device must fetch within this
	pairReqKeepDone   = time.Hour        // decided requests stay listed this long
	pairReqPerNode    = 3                // new requests per device per pairReqWindow
	pairReqWindow     = 10 * time.Minute
	pairReqMaxPending = 16
)

type pairRequest struct {
	ID       string    `json:"id"`
	Name     string    `json:"name"` // suggested client name
	Kind     string    `json:"kind,omitempty"`
	Device   string    `json:"device"` // tailnet machine name
	DNSName  string    `json:"dns_name,omitempty"`
	Hostname string    `json:"hostname,omitempty"`
	OS       string    `json:"os,omitempty"`
	User     string    `json:"user"`
	UserName string    `json:"user_name,omitempty"`
	IP       string    `json:"ip"`
	Status   string    `json:"status"` // pending | approved | declined | expired
	Created  time.Time `json:"created"`
	Expires  time.Time `json:"expires"`
	Client   string    `json:"client,omitempty"` // client name once approved
	Repair   bool      `json:"repair,omitempty"` // this device is already a client

	nodeID    string
	token     string
	delivered bool
	avoid     []netip.Prefix // the asker's other servers' subnets
}

const (
	reqPending  = "pending"
	reqApproved = "approved"
	reqDeclined = "declined"
	reqExpired  = "expired"
)

// pairRequests is guarded by Agent.mu.
type pairRequests struct {
	byID    map[string]*pairRequest
	created map[string][]time.Time // nodeID -> creation times (rate limit)
}

func newPairRequests() *pairRequests {
	return &pairRequests{byID: map[string]*pairRequest{}, created: map[string][]time.Time{}}
}

// gc expires and forgets old requests. Caller holds Agent.mu.
func (p *pairRequests) gc(now time.Time) {
	for id, r := range p.byID {
		switch {
		case r.Status == reqPending && now.After(r.Expires):
			r.Status = reqExpired
			r.Expires = now.Add(pairReqKeepDone)
		case r.Status == reqApproved && now.After(r.Expires):
			// unclaimed or claimed: the payload is no longer served
			r.Status, r.delivered = reqExpired, true
			r.Expires = now.Add(pairReqKeepDone)
		case r.Status != reqPending && r.Status != reqApproved && now.After(r.Expires):
			delete(p.byID, id)
		}
	}
	for n, ts := range p.created {
		keep := ts[:0]
		for _, t := range ts {
			if now.Sub(t) < pairReqWindow {
				keep = append(keep, t)
			}
		}
		if len(keep) == 0 {
			delete(p.created, n)
		} else {
			p.created[n] = keep
		}
	}
}

func (p *pairRequests) list() []pairRequest {
	out := make([]pairRequest, 0, len(p.byID))
	for _, r := range p.byID {
		out = append(out, *r)
	}
	sort.Slice(out, func(i, j int) bool { return out[i].Created.After(out[j].Created) })
	return out
}

func (p *pairRequests) pending() []pairRequest {
	out := []pairRequest{}
	for _, r := range p.list() {
		if r.Status == reqPending {
			out = append(out, r)
		}
	}
	return out
}

func remoteAddr(r *http.Request) netip.Addr {
	ap, err := netip.ParseAddrPort(r.RemoteAddr)
	if err != nil {
		return netip.Addr{}
	}
	return ap.Addr().Unmap()
}

var reNameJunk = regexp.MustCompile(`[^a-z0-9._-]+`)

// clientName turns a device name into a valid mqvpn user name.
func clientName(cands ...string) string {
	for _, c := range cands {
		n := strings.Trim(reNameJunk.ReplaceAllString(strings.ToLower(strings.TrimSpace(c)), "-"), "-.")
		if len(n) > 63 {
			n = strings.Trim(n[:63], "-.")
		}
		if mqconf.ValidUserName(n) {
			return n
		}
	}
	return "device"
}

func (a *Agent) serverName() string {
	if n := a.ts.Get(context.Background()).Self; n != "" {
		return n
	}
	h, _ := os.Hostname()
	return h
}

// hHello lets routers and phones on the tailnet find tailbond servers.
// Unauthenticated and minimal on purpose.
func (a *Agent) hHello(w http.ResponseWriter, r *http.Request) {
	writeJSON(w, http.StatusOK, map[string]any{"tailbond": true, "name": a.serverName(), "version": a.opts.Version, "role": a.role, "protocol": update.Protocol})
}

// hPairRequestCreate: POST /api/pair-requests from a tailnet device.
func (a *Agent) hPairRequestCreate(w http.ResponseWriter, r *http.Request) {
	if a.role != config.RoleServer {
		httpErr(w, http.StatusBadRequest, errServerOnly)
		return
	}
	ip := remoteAddr(r)
	if !ip.IsValid() || !tailscale.IsTailnetIP(ip) || a.viaTunnel(r) {
		httpErr(w, http.StatusForbidden, errors.New("pairing requests are accepted only over Tailscale"))
		return
	}
	var body struct {
		Name  string   `json:"name"`
		Kind  string   `json:"kind"`
		Avoid []string `json:"avoid_subnets"`
	}
	if err := readJSON(r, &body); err != nil {
		httpErr(w, http.StatusBadRequest, err)
		return
	}
	who, err := a.whois(r.Context(), ip.String())
	if err != nil {
		a.Logf("pair request from %s: whois failed: %v", ip, err)
		httpErr(w, http.StatusForbidden, errors.New("this device isn't known to Tailscale on the server"))
		return
	}
	kind := ""
	switch body.Kind {
	case "router", "phone", "computer":
		kind = body.Kind
	}
	now := time.Now()
	a.mu.Lock()
	defer a.mu.Unlock()
	p := a.pairReqs
	p.gc(now)
	// one live request per device: asking again returns the same one
	for _, x := range p.byID {
		if x.nodeID == who.NodeID && (x.Status == reqPending || (x.Status == reqApproved && !x.delivered)) {
			x.IP = ip.String()
			x.avoid = parseAvoid(body.Avoid)
			writeJSON(w, http.StatusAccepted, a.pairReqReply(x, true))
			return
		}
	}
	if len(p.created[who.NodeID]) >= pairReqPerNode {
		w.Header().Set("Retry-After", fmt.Sprint(int(pairReqWindow.Seconds())))
		httpErr(w, http.StatusTooManyRequests, errors.New("too many requests from this device; try again in a few minutes"))
		return
	}
	if len(p.pending()) >= pairReqMaxPending {
		httpErr(w, http.StatusTooManyRequests, errors.New("too many pending requests on the server"))
		return
	}
	req := &pairRequest{
		ID: randHex(8), token: randHex(24), Kind: kind,
		Name:   clientName(body.Name, who.Node, who.Hostname),
		Device: who.Node, DNSName: who.DNSName, Hostname: who.Hostname, OS: who.OS,
		User: who.User, UserName: who.UserName, IP: ip.String(), nodeID: who.NodeID,
		Status: reqPending, Created: now, Expires: now.Add(pairReqTTL),
		avoid: parseAvoid(body.Avoid),
	}
	for _, u := range a.store.Get().Server.Users {
		if u.TSNode != "" && u.TSNode == who.NodeID {
			req.Repair, req.Name = true, u.Name
		}
	}
	p.byID[req.ID] = req
	p.created[who.NodeID] = append(p.created[who.NodeID], now)
	a.logs.Add("agent", fmt.Sprintf("pair request %s from %s (%s, %s)", req.ID, req.Device, req.User, req.IP))
	writeJSON(w, http.StatusAccepted, a.pairReqReply(req, true))
}

// parseAvoid reads avoid_subnets: up to 16 IPv4 prefixes.
func parseAvoid(in []string) []netip.Prefix {
	var out []netip.Prefix
	for _, s := range in {
		if p, err := netip.ParsePrefix(strings.TrimSpace(s)); err == nil && p.Addr().Is4() && len(out) < 16 {
			out = append(out, p.Masked())
		}
	}
	return out
}

// pairReqReply is what the requesting device sees. Caller holds a.mu.
func (a *Agent) pairReqReply(x *pairRequest, withToken bool) map[string]any {
	out := map[string]any{"id": x.ID, "status": x.Status, "name": x.Name, "expires": x.Expires.UTC()}
	if withToken {
		out["token"] = x.token
		// how the server's approval card shows the requester, so the two
		// can be matched (it is the requester's own identity)
		out["device"], out["user"], out["ip"] = x.Device, x.User, x.IP
	}
	return out
}

// hPairRequestPoll: GET /api/pair-requests/{id}, the requester waiting for
// the admin. Needs the token from the create reply and the same source IP.
func (a *Agent) hPairRequestPoll(w http.ResponseWriter, r *http.Request) {
	if a.role != config.RoleServer {
		httpErr(w, http.StatusBadRequest, errServerOnly)
		return
	}
	tok, _ := strings.CutPrefix(r.Header.Get("Authorization"), "Bearer ")
	ip := remoteAddr(r)
	now := time.Now()
	a.mu.Lock()
	a.pairReqs.gc(now)
	x, ok := a.pairReqs.byID[r.PathValue("id")]
	if !ok || tok == "" || subtle.ConstantTimeCompare([]byte(strings.TrimSpace(tok)), []byte(x.token)) != 1 || x.IP != ip.String() {
		a.mu.Unlock()
		httpErr(w, http.StatusNotFound, errNotFound)
		return
	}
	reply := a.pairReqReply(x, false)
	var client config.User
	approved := x.Status == reqApproved
	if approved {
		for _, u := range a.store.Get().Server.Users {
			if u.Name == x.Client {
				client = u
			}
		}
		x.delivered = true
	}
	a.mu.Unlock()
	if approved {
		if client.Name == "" {
			httpErr(w, http.StatusGone, errors.New("the server removed this device again"))
			return
		}
		res, err := a.pairingFor(client)
		if err != nil {
			httpErr(w, http.StatusInternalServerError, err)
			return
		}
		reply["uri"], reply["client"] = res.URI, client.Name
	}
	writeJSON(w, http.StatusOK, reply)
}

// hPairRequestList: GET /api/pair-requests (admin).
func (a *Agent) hPairRequestList(w http.ResponseWriter, r *http.Request) {
	if a.role != config.RoleServer {
		httpErr(w, http.StatusBadRequest, errServerOnly)
		return
	}
	a.mu.Lock()
	a.pairReqs.gc(time.Now())
	out := a.pairReqs.list()
	a.mu.Unlock()
	writeJSON(w, http.StatusOK, out)
}

// hPairRequestDecide: POST /api/pair-requests/{id}/approve|decline (admin).
func (a *Agent) hPairRequestDecide(w http.ResponseWriter, r *http.Request) {
	if a.role != config.RoleServer {
		httpErr(w, http.StatusBadRequest, errServerOnly)
		return
	}
	var body struct {
		Name string `json:"name"`
	}
	if r.ContentLength != 0 {
		if err := readJSON(r, &body); err != nil {
			httpErr(w, http.StatusBadRequest, err)
			return
		}
	}
	approve := r.PathValue("action") == "approve"
	if !approve && r.PathValue("action") != "decline" {
		httpErr(w, http.StatusNotFound, errNotFound)
		return
	}
	a.mu.Lock()
	a.pairReqs.gc(time.Now())
	x, ok := a.pairReqs.byID[r.PathValue("id")]
	if !ok {
		a.mu.Unlock()
		httpErr(w, http.StatusNotFound, errNotFound)
		return
	}
	if x.Status != reqPending {
		st := x.Status
		a.mu.Unlock()
		httpErr(w, http.StatusConflict, fmt.Errorf("this request is already %s", st))
		return
	}
	// the asking device must not approve itself through the dashboard
	if src := remoteAddr(r); src.IsValid() && src.String() == x.IP {
		a.mu.Unlock()
		httpErr(w, http.StatusForbidden, errors.New("approve this request from another device"))
		return
	}
	if !approve {
		x.Status, x.Expires = reqDeclined, time.Now().Add(pairReqKeepDone)
		a.mu.Unlock()
		a.Logf("pair request %s from %s declined", x.ID, x.Device)
		writeJSON(w, http.StatusOK, map[string]any{"ok": true, "status": reqDeclined})
		return
	}
	name, nodeID, avoid := x.Name, x.nodeID, x.avoid
	if body.Name != "" {
		name = clientName(body.Name)
	}
	a.mu.Unlock()

	// a client that already uses this server's subnet for another server:
	// a server without clients moves to a free one
	if len(avoid) > 0 && len(a.store.Get().Server.Users) == 0 {
		if _, err := a.renumberSubnet(avoid); err != nil && !errors.Is(err, errHasClients) {
			a.Logf("pair request %s: can't move the tunnel subnet: %v", x.ID, err)
		}
	}

	client, err := a.clientForRequest(r.Context(), name, nodeID)
	if err != nil {
		httpErr(w, http.StatusBadRequest, err)
		return
	}
	a.mu.Lock()
	if x.Status == reqPending {
		x.Status, x.Client, x.Expires = reqApproved, client, time.Now().Add(pairReqPickupTTL)
	}
	a.mu.Unlock()
	a.Logf("pair request %s from %s approved as %s", x.ID, x.Device, client)
	a.poll(r.Context())
	writeJSON(w, http.StatusOK, map[string]any{"ok": true, "status": reqApproved, "client": client})
}

// clientForRequest returns the client a tailnet node pairs as: its existing
// credential, or a new one under a free name.
func (a *Agent) clientForRequest(ctx context.Context, name, nodeID string) (string, error) {
	users := a.store.Get().Server.Users
	for _, u := range users {
		if nodeID != "" && u.TSNode == nodeID {
			return u.Name, nil
		}
	}
	taken := map[string]bool{}
	for _, u := range users {
		taken[u.Name] = true
	}
	base, n := name, 2
	for taken[name] {
		suffix := fmt.Sprintf("-%d", n)
		if len(base)+len(suffix) > 63 {
			base = base[:63-len(suffix)]
		}
		name, n = base+suffix, n+1
	}
	if _, err := a.addClientNode(ctx, name, nodeID); err != nil {
		return "", err
	}
	return name, nil
}

func randHex(n int) string {
	b := make([]byte, n)
	_, _ = rand.Read(b)
	return fmt.Sprintf("%x", b)
}
