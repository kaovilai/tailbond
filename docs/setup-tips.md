# Setup tips: best speed

A short checklist. Each point links to Tailscale's own docs instead of
repeating them. The **Setup health** card (GL panel on a Flint 3, the
Fedora page's Devices tab, `GET /api/health`) checks most of these for you.

- [ ] **Tailscale connects directly to the home server, not through a relay.**
  Forward UDP 41641 to the home server, or turn on UPnP / NAT-PMP on the home
  router; avoid double NAT; public IPv6 helps.
  Check with `tailscale ping <peer>`: "via `<ip>:<port>`" is direct,
  "via DERP(xxx)" is relayed; `tailscale netcheck` shows UDP, hard NAT and port
  mapping.
  [Connection types](https://tailscale.com/docs/reference/connection-types) ·
  [Firewall ports](https://tailscale.com/docs/reference/faq/firewall-ports) ·
  [Using Tailscale with your firewall](https://tailscale.com/docs/integrations/firewalls) ·
  [Device connectivity](https://tailscale.com/docs/reference/device-connectivity) ·
  [DERP servers](https://tailscale.com/docs/reference/derp-servers) ·
  [CLI](https://tailscale.com/docs/reference/tailscale-cli)
- [ ] **tailbond's own port: UDP 443 reaches the home server from the internet.**
  Forward it on any router in front of the server (a Flint 3 opens it on
  itself). Carrier-grade NAT on the home line (the router's WAN address is
  in 100.64.0.0/10 or private while your public IP differs) can't work: ask
  the provider for a public IPv4, or use public IPv6.
  [Device connectivity](https://tailscale.com/docs/reference/device-connectivity)
- [ ] **Approve the `10.77.0.0/24` route once** in the Tailscale admin console,
  so tailnet machines reach bonded devices.
  [Subnet routers](https://tailscale.com/docs/features/subnet-routers)
- [ ] **GL.iNet: Network Acceleration** can stay on; tailbond's tunnel traffic
  can't be offloaded anyway. If something looks off right after turning
  tailbond on, connections opened before it are still offloaded; they end by
  themselves, or turn acceleration off. See [glinet.md](glinet.md#network-acceleration).
- [ ] **GL.iNet: hotel sign-in pages.** GL's repeater "Login Mode for Public
  Hotspots" works with tailbond on; tailbond follows it. Camouflage / MAC
  clone keep working. See [glinet.md](glinet.md#captive-portals-and-gls-login-mode).
- [ ] **DNS rebinding protection vs MagicDNS and portals.** On OpenWrt/GL,
  dnsmasq's rebind protection drops answers in private ranges; MagicDNS
  answers are 100.x (not private), but a hotel portal's may be 10.x. tailbond's
  sign-in mode turns it off only while signing in.
  [Troubleshooting](https://tailscale.com/docs/reference/troubleshooting)
- [ ] **Android: leave the Tailscale app's always-on VPN off** (and "Block
  connections without VPN" off) on a phone that uses tailbond; Android runs
  one VPN at a time. See [android.md](android.md#known-limitations).
