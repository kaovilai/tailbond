package agent

import (
	"context"
	"errors"
	"fmt"
	"net"
	"os"
	"path/filepath"
	"strconv"
	"strings"
	"time"

	"github.com/tailbond/tailbond/agent/internal/config"
	"github.com/tailbond/tailbond/agent/internal/dnsfwd"
	"github.com/tailbond/tailbond/agent/internal/mqconf"
	"github.com/tailbond/tailbond/agent/internal/mqctl"
	"github.com/tailbond/tailbond/agent/internal/pairing"
	"github.com/tailbond/tailbond/agent/internal/status"
	"github.com/tailbond/tailbond/agent/internal/supervisor"
	"github.com/tailbond/tailbond/agent/internal/tlsutil"
)

func (a *Agent) startServer(ctx context.Context) {
	cfg := a.store.Get()
	if !cfg.Server.DisableDNS {
		ip := cfg.ServerTunnelIP()
		f := &dnsfwd.Forwarder{
			Addr: net.JoinHostPort(ip, "53"),
			Suffixes: func() []string {
				ti := a.ts.Get(context.Background())
				if ti.MagicDNSSuffix != "" {
					return []string{ti.MagicDNSSuffix}
				}
				return nil
			},
			TailscaleUp: func() bool { return a.ts.Get(context.Background()).Running },
			Log:         func(s string) { a.Logf("%s", s) },
		}
		go f.Run(ctx)
	}
	go a.certRenewer(ctx)
	a.sup.Start()
}

// certRenewer refreshes the tailscale cert daily; a changed cert restarts mqvpn.
func (a *Agent) certRenewer(ctx context.Context) {
	t := time.NewTicker(24 * time.Hour)
	defer t.Stop()
	for {
		select {
		case <-ctx.Done():
			return
		case <-t.C:
		}
		a.mu.Lock()
		cur := a.tlsInfo
		a.mu.Unlock()
		if cur.source != "tailscale" {
			continue
		}
		next, err := a.prepareTLS(ctx, true)
		if err == nil && next.sha256 != cur.sha256 && a.sup.Active() {
			a.Logf("tailscale certificate renewed, restarting mqvpn")
			a.sup.Restart()
		}
	}
}

// prepareTLS picks the tailscale cert when available, else a self-signed one.
func (a *Agent) prepareTLS(ctx context.Context, forceRefresh bool) (tlsInfo, error) {
	cfg := a.store.Get()
	var info tlsInfo
	if cfg.Server.UseTailscaleCert && !cfg.DisableTailscale {
		ti := a.ts.Get(ctx)
		if ti.Running && ti.DNSName != "" {
			certFile := filepath.Join(cfg.StateDir, "ts-cert.pem")
			keyFile := filepath.Join(cfg.StateDir, "ts-key.pem")
			fresh := false
			if fi, err := os.Stat(certFile); err == nil && time.Since(fi.ModTime()) < 24*time.Hour && !forceRefresh {
				fresh = true
			}
			var err error
			if !fresh {
				err = a.ts.Cert(ctx, ti.DNSName, certFile, keyFile)
				if err != nil {
					a.Logf("tailscale cert for %s unavailable (%v); using self-signed certificate", ti.DNSName, err)
				}
			}
			if err == nil {
				if sha, ferr := tlsutil.FingerprintFile(certFile); ferr == nil {
					info = tlsInfo{certFile: certFile, keyFile: keyFile, sni: ti.DNSName, sha256: sha, source: "tailscale"}
				}
			}
		}
	}
	if info.certFile == "" {
		certFile := filepath.Join(cfg.StateDir, "server.crt")
		keyFile := filepath.Join(cfg.StateDir, "server.key")
		host, _ := os.Hostname()
		names := []string{host}
		if h := hostOnly(cfg.Server.PublicAddr); h != "" {
			names = append(names, h)
		}
		if err := tlsutil.EnsureSelfSigned(certFile, keyFile, names); err != nil {
			return info, fmt.Errorf("self-signed cert: %w", err)
		}
		sha, err := tlsutil.FingerprintFile(certFile)
		if err != nil {
			return info, err
		}
		info = tlsInfo{certFile: certFile, keyFile: keyFile, sni: host, insecure: true, sha256: sha, source: "self-signed"}
	}
	a.mu.Lock()
	a.tlsInfo = info
	a.mu.Unlock()
	return info, nil
}

func hostOnly(addr string) string {
	if addr == "" {
		return ""
	}
	if h, _, err := net.SplitHostPort(addr); err == nil {
		return h
	}
	return strings.Trim(addr, "[]")
}

func (a *Agent) serverPreStart(ctx context.Context) error {
	cfg := a.store.Get()
	tls, err := a.prepareTLS(ctx, false)
	if err != nil {
		return err
	}
	users := make([]mqconf.User, 0, len(cfg.Server.Users))
	for _, u := range cfg.Server.Users {
		users = append(users, mqconf.User{Name: u.Name, Key: u.Key})
	}
	global := ""
	if len(users) == 0 {
		// mqvpn refuses to start without a credential; this one is never shared.
		global, err = a.stateSecret("global.key")
		if err != nil {
			return err
		}
	}
	ini, err := mqconf.Server(mqconf.ServerParams{
		Port: cfg.Server.Port, Subnet: cfg.Server.Subnet, Subnet6: cfg.Server.Subnet6,
		TunName: config.DefaultTunName, CertFile: tls.certFile, KeyFile: tls.keyFile,
		Users: users, GlobalKey: global, Mode: cfg.Mode, ControlAddr: cfg.ControlAddr,
		LogLevel: cfg.LogLevel, EgressAllow: config.TailnetRoutes,
	})
	if err != nil {
		return err
	}
	return writeFileAtomic(a.confPath(), []byte(ini), 0o600)
}

func (a *Agent) stateSecret(name string) (string, error) {
	p := filepath.Join(a.store.Get().StateDir, name)
	if b, err := os.ReadFile(p); err == nil && len(strings.TrimSpace(string(b))) > 0 {
		return strings.TrimSpace(string(b)), nil
	}
	k, err := genKey()
	if err != nil {
		return "", err
	}
	return k, writeFileAtomic(p, []byte(k+"\n"), 0o600)
}

func (a *Agent) serverStatus(ctx context.Context) *status.Status {
	cfg := a.store.Get()
	a.mu.Lock()
	ready := a.rt.ready
	tls := a.tlsInfo
	a.mu.Unlock()

	st := &status.Status{
		Role: cfg.Role, Mode: cfg.Mode, Paths: []status.Path{}, TunnelIP: cfg.ServerTunnelIP(),
	}
	var mst *mqctl.Status
	var mstats *mqctl.Stats
	if a.sup.Info().State == supervisor.Running {
		var err error
		if mst, err = a.ctl.Status(ctx); err == nil {
			ready = true
			st.ControlAPI = true
			mstats, _ = a.ctl.Stats(ctx)
		} else {
			mst = nil
		}
	}
	st.State, st.Error = a.supState(ready)
	names := make([]string, 0, len(cfg.Server.Users))
	for _, u := range cfg.Server.Users {
		names = append(names, u.Name)
	}
	a.rater.Begin()
	st.Clients, st.Totals, st.UptimeSec = status.Server(a.rater, time.Now(), mst, mstats, names)
	a.rater.End()
	st.Tailscale = a.tsStatus(ctx)
	st.Server = a.publicAddr()
	if tls.source == "tailscale" {
		st.Server = tls.sni
	}
	return st
}

// publicAddr is what clients dial: server.public_addr, else a guess (the
// source address of the default route), which is usually a LAN address.
func (a *Agent) publicAddr() string {
	cfg := a.store.Get()
	port := strconv.Itoa(cfg.Server.Port)
	if pa := strings.TrimSpace(cfg.Server.PublicAddr); pa != "" {
		if _, _, err := net.SplitHostPort(pa); err == nil {
			return pa
		}
		return net.JoinHostPort(strings.Trim(pa, "[]"), port)
	}
	if ip := outboundIP(); ip != "" {
		return net.JoinHostPort(ip, port)
	}
	return ""
}

func outboundIP() string {
	c, err := net.Dial("udp", "192.0.2.1:9")
	if err != nil {
		return ""
	}
	defer c.Close()
	return c.LocalAddr().(*net.UDPAddr).IP.String()
}

type pairResult struct {
	Name  string `json:"name"`
	URI   string `json:"uri"`
	QRSVG string `json:"qr_svg"`
}

func (a *Agent) pairingFor(u config.User) (*pairResult, error) {
	cfg := a.store.Get()
	a.mu.Lock()
	tls := a.tlsInfo
	a.mu.Unlock()
	if tls.certFile == "" {
		var err error
		if tls, err = a.prepareTLS(context.Background(), false); err != nil {
			return nil, err
		}
	}
	server := a.publicAddr()
	if server == "" {
		return nil, errors.New("no public address: set server.public_addr")
	}
	p := &pairing.Payload{
		V: pairing.Version, Name: u.Name, Server: server, User: u.Name, Key: u.Key,
		Mode: cfg.Mode, TailnetRoutes: config.TailnetRoutes, SNI: tls.sni,
	}
	if tls.insecure {
		p.Insecure, p.CertSHA256 = true, tls.sha256
	}
	if !cfg.Server.DisableDNS {
		p.DNS = []string{cfg.ServerTunnelIP()}
	}
	uri, err := pairing.Encode(p)
	if err != nil {
		return nil, err
	}
	svg, err := pairing.QRSVG(uri)
	if err != nil {
		return nil, err
	}
	return &pairResult{Name: u.Name, URI: uri, QRSVG: svg}, nil
}

func (a *Agent) addClient(ctx context.Context, name string) (*pairResult, error) {
	if !mqconf.ValidUserName(name) {
		return nil, errors.New("name must be 1-63 characters: letters, digits, '.', '_' or '-'")
	}
	key, err := genKey()
	if err != nil {
		return nil, err
	}
	u := config.User{Name: name, Key: key, Created: time.Now().Unix()}
	if _, err := a.store.Update(func(c *config.Config) error {
		for _, x := range c.Server.Users {
			if x.Name == name {
				return fmt.Errorf("client %q already exists", name)
			}
		}
		c.Server.Users = append(c.Server.Users, u)
		return nil
	}); err != nil {
		return nil, err
	}
	if err := a.ctl.AddUser(ctx, name, key); err != nil {
		if !errors.Is(err, mqctl.ErrUnavailable) {
			a.Logf("add_user %s: %v", name, err)
		}
	} else {
		a.Logf("client %s added", name)
	}
	return a.pairingFor(u)
}

func (a *Agent) removeClient(ctx context.Context, name string) error {
	found := false
	if _, err := a.store.Update(func(c *config.Config) error {
		out := c.Server.Users[:0]
		for _, x := range c.Server.Users {
			if x.Name == name {
				found = true
				continue
			}
			out = append(out, x)
		}
		c.Server.Users = out
		return nil
	}); err != nil {
		return err
	}
	if !found {
		return errNotFound
	}
	if err := a.ctl.RemoveUser(ctx, name); err != nil && !errors.Is(err, mqctl.ErrUnavailable) {
		a.Logf("remove_user %s: %v", name, err)
	}
	a.Logf("client %s revoked", name)
	// With no users left mqvpn falls back to the private global key, which
	// needs a fresh config.
	if len(a.store.Get().Server.Users) == 0 && a.sup.Active() {
		a.sup.Restart()
	}
	return nil
}

var errNotFound = errors.New("not found")
