// Command tb-sharesim is a stand-alone Pair & Share sharer for tests: the
// reference relay (internal/share), its mDNS advert and its registration
// with the home server, without a tailbondd or an mqvpn of its own. It
// stands in for a phone sharing its mobile data where a test can't run a
// whole sharer per "phone" (scripts/e2e-macos.sh: two of them on one Mac).
//
//	tb-sharesim -pairing 'tailbond://pair?d=…' -listen 192.168.77.2:47077 \
//	    -label "Pixel's 5G" [-mdns feth1] [-register http://127.0.0.1:18077]
//
// The relay sends upstream to the pairing's server address over the
// normal route (a phone would bind it to cellular). -register-only posts
// one registry entry and exits (a sharer the hub can't reach: the client
// isolation check).
//
// -proxy ADDR:PORT runs the phone hub's proxy instead (docs/phone-hub.md):
// internal/proxy's SOCKS5 + HTTP CONNECT server with the credentials derived
// from the pairing's share key, advertised as _tailbond-proxy._tcp and
// registered with kind "proxy". Its upstream sockets use the normal route,
// so on a Mac running tailbondd they go into the tunnel, as a phone's do.
package main

import (
	"bytes"
	"context"
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"flag"
	"fmt"
	"log"
	"net"
	"net/http"
	"os"
	"os/signal"
	"strings"
	"syscall"
	"time"

	"github.com/kaovilai/tailbond/agent/internal/agent"
	"github.com/kaovilai/tailbond/agent/internal/pairing"
	"github.com/kaovilai/tailbond/agent/internal/proxy"
	"github.com/kaovilai/tailbond/agent/internal/share"
)

func main() {
	uri := flag.String("pairing", "", "pairing URI of a device paired with the home server (its share key, server and token)")
	listen := flag.String("listen", "", "LAN address:port the relay listens on (the address hubs reach)")
	label := flag.String("label", "Test phone", "what hubs show")
	id := flag.String("id", "", "sharer id (default: from the label and address)")
	mdns := flag.String("mdns", "", "comma-separated interfaces to answer mDNS on (default: all LAN interfaces; \"-\" = none)")
	register := flag.String("register", "", "home server API base URL to register with, e.g. http://127.0.0.1:8077 (default: don't)")
	registerOnly := flag.Bool("register-only", false, "register once and exit, without relaying")
	proxyAddr := flag.String("proxy", "", "run the phone hub's proxy on this address:port instead of a relay")
	upstream := flag.String("upstream", "", "relay to this server address instead of the pairing's (the Android emulator e2e: the pairing says 10.0.2.2, the host's 127.0.0.1)")
	flag.Parse()
	if *proxyAddr != "" {
		*listen = *proxyAddr
	}
	if *uri == "" || *listen == "" {
		flag.Usage()
		os.Exit(2)
	}
	p, err := pairing.Decode(*uri)
	if err != nil {
		log.Fatal(err)
	}
	if p.ShareKey == "" {
		log.Fatal("the pairing carries no share key")
	}
	key := []byte(p.ShareKey)
	network := "udp4"
	if *proxyAddr != "" {
		network = "tcp4"
	}
	la, err := resolve(network, *listen)
	if err != nil {
		log.Fatal(err)
	}
	if *id == "" {
		h := sha256.Sum256([]byte(*label + "/" + *listen))
		*id = "sim-" + hex.EncodeToString(h[:5])
	}
	ctx, stop := signal.NotifyContext(context.Background(), os.Interrupt, syscall.SIGTERM)
	defer stop()
	logf := func(format string, args ...any) { log.Printf(*id+": "+format, args...) }

	kind := "sim"
	if *proxyAddr != "" {
		kind = "proxy"
	}
	reg := func() error {
		body, _ := json.Marshal(map[string]any{"id": *id, "label": *label, "key_id": share.KeyID(key),
			"lan": []string{la.IP.String()}, "port": la.Port, "kind": kind})
		req, _ := http.NewRequestWithContext(ctx, http.MethodPost, strings.TrimRight(*register, "/")+"/api/shares", bytes.NewReader(body))
		req.Header.Set("Content-Type", "application/json")
		req.Header.Set("Authorization", "Bearer "+agent.APIToken(p.Key))
		resp, err := http.DefaultClient.Do(req)
		if err != nil {
			return err
		}
		resp.Body.Close()
		if resp.StatusCode != http.StatusOK {
			return fmt.Errorf("register: HTTP %d", resp.StatusCode)
		}
		return nil
	}
	if *registerOnly {
		if err := reg(); err != nil {
			log.Fatal(err)
		}
		logf("registered %s:%d with %s", la.IP, la.Port, *register)
		return
	}

	if *mdns != "-" {
		go func() {
			adv := share.Advert{ID: *id, Label: *label, KeyID: share.KeyID(key), Port: la.Port, IP: la.IP}
			if *proxyAddr != "" {
				adv.Service = proxy.Service
				adv.Bonded = "1" // a stand-in for an Android phone's bonded tunnel
			}
			for ctx.Err() == nil {
				if err := share.Advertise(ctx, ifaces(*mdns), adv); err != nil {
					logf("mDNS: %v", err)
				}
				select {
				case <-ctx.Done():
				case <-time.After(5 * time.Second):
				}
			}
		}()
	}
	if *register != "" {
		go func() {
			for ctx.Err() == nil {
				if err := reg(); err != nil {
					logf("%v", err)
				}
				select {
				case <-ctx.Done():
				case <-time.After(20 * time.Second):
				}
			}
		}()
	}
	if *proxyAddr != "" {
		runProxy(ctx, la, key, logf)
		return
	}

	server := p.Server
	if *upstream != "" {
		server = *upstream
	}
	sh, err := share.NewSharer(share.SharerConfig{
		ID: *id, Label: *label, Key: key, Server: server,
		Dial: func(ctx context.Context, server string) (net.Conn, error) {
			var d net.Dialer
			return d.DialContext(ctx, "udp", server)
		},
		Logf: logf,
	})
	if err != nil {
		log.Fatal(err)
	}
	conn, err := net.ListenPacket("udp4", la.String())
	if err != nil {
		log.Fatal(err)
	}
	go func() { _ = sh.Serve(ctx, conn) }()
	logf("sharing as %q on %s, upstream %s", *label, la, server)
	t := time.NewTicker(5 * time.Second)
	defer t.Stop()
	for {
		select {
		case <-ctx.Done():
			sh.Stop(share.ReasonPaused)
			return
		case <-t.C:
			for _, s := range sh.Sessions() {
				logf("hub %s at %s: up %d down %d", s.HubID, s.Addr, s.Up, s.Down)
			}
		}
	}
}

// addr is a listen address as IP and port (UDP or TCP alike).
type addr struct {
	IP   net.IP
	Port int
}

func (a addr) String() string { return net.JoinHostPort(a.IP.String(), fmt.Sprint(a.Port)) }

func resolve(network, s string) (addr, error) {
	if network == "tcp4" {
		t, err := net.ResolveTCPAddr(network, s)
		if err != nil {
			return addr{}, err
		}
		return addr{t.IP, t.Port}, nil
	}
	u, err := net.ResolveUDPAddr(network, s)
	if err != nil {
		return addr{}, err
	}
	return addr{u.IP, u.Port}, nil
}

// runProxy is the phone hub's proxy until ctx ends, logging each client's
// byte counts (scripts/e2e-macos.sh reads them: the hub carried the traffic).
func runProxy(ctx context.Context, la addr, key []byte, logf func(string, ...any)) {
	user, pass := proxy.Credentials(key)
	ln, err := net.Listen("tcp4", la.String())
	if err != nil {
		log.Fatal(err)
	}
	s := &proxy.Server{User: user, Pass: pass, Logf: logf}
	go func() { _ = s.Serve(ln) }()
	logf("proxy on %s (user %s)", la, user)
	t := time.NewTicker(2 * time.Second)
	defer t.Stop()
	last := ""
	for {
		select {
		case <-ctx.Done():
			ln.Close()
			return
		case <-t.C:
			up, down, clients := s.Stats()
			line := fmt.Sprintf("proxy: up %d down %d", up, down)
			for _, c := range clients {
				line += fmt.Sprintf("; client %s conns %d up %d down %d", c.Addr, c.Conns, c.Up, c.Down)
			}
			if line != last {
				logf("%s", line)
				last = line
			}
		}
	}
}

func ifaces(list string) []net.Interface {
	if list == "" {
		return share.MulticastIfaces()
	}
	var out []net.Interface
	for _, n := range strings.Split(list, ",") {
		if ifi, err := net.InterfaceByName(strings.TrimSpace(n)); err == nil {
			out = append(out, *ifi)
		}
	}
	return out
}
