# Design note: one tailscaled per internet connection

**Status: researched, not implemented. Not recommended as the default** — the
VPS relay in [behind-cgnat.md](behind-cgnat.md#4-keep-the-server-at-home-relay-its-port-through-a-vps)
reaches the same goal for a fraction of the work. This note exists so the
option isn't re-litigated from scratch, and so the numbers below are not
rediscovered. Follow-up to [issue #44](https://github.com/kaovilai/tailbond/issues/44).

The problem it would solve: a home server behind carrier-grade NAT with no
public IPv6 can't be a bond endpoint, because every path needs to reach the
server's tunnel port. Tailscale alone can't stand in — one `tailscaled` uses
one path per peer and doesn't add paths up.

The idea: run **one `tailscaled` per internet connection** on the router, make
each instance's tun one mqvpn path, and let Tailscale's own NAT traversal
(WireGuard hole punching, DERP as fallback) carry each path to the server's
tailnet address. The server then needs no public address or port forward.

## Verdict

**Viable-with-costs, and one experiment decides it.** The mechanism works — it
was measured on a GL-BE9300 (GL.iNet OpenWrt 23.05, kernel 5.4.213): network
namespaces, `veth` and tun all load, a kernel-mode `tailscaled` runs inside a
namespace, a socket bound to `tailscale0` gets that instance's own `100.x`
source address, `tailscale0` passes mqvpn's `SO_BINDTODEVICE` pin and
FIB-match gate, and mqvpn carried two validated active paths that differed
only in source address — exactly what one node per connection produces.

The risk is not the mechanism, it's **what Tailscale can't take direct**:

| | DERP fallback | Direct |
|---|---|---|
| Transport | TCP + TLS on 443 | WireGuard UDP |
| Measured throughput | **~2.2 Mbit/s** sustained on one transoceanic path | underlay-limited |
| RTT (CGNAT case study) | 441–478 ms | tens of ms |

A relayed path is liveness, not capacity, and it must stay out of the WLB
datagram set: mqvpn's own benchmarks show 10 ms vs 30 ms paths already
producing 10–30 ms of jitter, so a 300 ms path would wreck the aggregate.
Cellular — the path that makes bonding worth having — is the most likely to be
symmetric-CGNAT and therefore relayed. The design can end up adding capacity
only where it is needed least.

## The experiment that decides it (about an hour, no code)

1. On the server: `tailscale netcheck --json`, record `MappingVariesByDestIP`.
2. From a node on the Wi-Fi/Ethernet side, and from a node on cellular,
   `tailscale ping <server>` three to five times each. Phones can't run the CLI
   — the Tailscale app has no shell — so either use the app's own connection
   details for that device, or tether a laptop to the phone and ping from the
   laptop. Tethering is a proxy, not the same test: Android adds its own NAT
   hop, which can flip the result either way, so confirm on the router itself
   before rejecting the design.
3. Record direct vs `via DERP` and the RTT for each.

If the cellular paths are DERP-only, the flagship case is gone; stop there and
use the relay or a server on a VPS. Put the server behind a phone hotspot for
an hour to create a real carrier CGNAT on the server side if there isn't one.

## The architecture that works

One **network namespace per WAN**, a kernel `tailscaled` inside each, and a
veth bridge back to the single mqvpn process in the main namespace. Not policy
routing, not one flat namespace — see "What not to build".

```
 main namespace                                  per-WAN namespace N
 ┌───────────────────────────────┐              ┌──────────────────────────────┐
 │ mqvpn  --path tbv0 --path tbv1│  veth pair   │ tailscaled (own statedir,    │
 │  socket bound to tbvN         │ tbvN ────────│ socket, port, tun)           │
 │  route 100.y/32 via 10.99.N.2 │ 10.99.N.1/.2 │  tailscale0 = 100.x/32       │
 │  metric route per path        │              │  default route via veth      │
 │ masquerade tbvN out WAN N     │              │  SNAT veth -> 100.x on tun   │
 └───────────────────────────────┘              └──────────────┬───────────────┘
                                                               │ hole punch / DERP
                                                         WAN N │
                                                               ▼
                              server: one tailscaled + mqvpn listening on 0.0.0.0:443
                              clients connect to 100.y; no port forward needed
```

Per WAN: `ip netns add tbpath-N`, a veth pair, one end moved in; the namespace's
only route is the veth, so **everything** that instance does — WireGuard, STUN,
control, DERP — leaves through WAN N. Run
`tailscaled --statedir=/etc/tailscale/path-N --socket=/run/tailscale/path-N.sock --port=<unique> --tun=tailscale0`,
then `tailscale --socket=… up --authkey=<reusable tagged key> --accept-dns=false --hostname=tb-path-N`;
`--accept-dns=false` is required because `/etc/resolv.conf` is not namespaced,
and mqvpn reaches the server by literal `100.y`, so MagicDNS isn't needed.

In the main namespace, mqvpn runs once with the pairing's server address set to
the server's `100.y`, one path per veth, and per-path metric routes
(`ip route add 100.y/32 via 10.99.N.2 dev tbvN metric N`) so the FIB-match gate
passes for that device.

Getting each veth's traffic out of the **right** WAN is explicit work, not
something the existing tables do for you: tailbond's per-WAN tables are keyed by
fwmark (`packaging/openwrt/hooks/bypass-apply`), and a packet arriving from a
netns carries no mark. Per WAN, add a source-based rule and table, plus SNAT
bound to that interface:

```sh
ip rule add from 10.99.N.0/30 lookup tbwanN priority 1000+N
ip route add default via <WAN N gateway> dev <WAN N device> table tbwanN
iptables -t nat -A POSTROUTING -s 10.99.N.0/30 -o <WAN N device> -j MASQUERADE
```

The veth end in the main namespace needs a route back into the namespace
(`ip route add 10.99.N.0/30 dev tbvN`), and the WAN device must not be a member
of a zone that drops unmanaged forwarding.

**Keep the paths from falling back onto each other.** A `lookup` rule whose
table has no matching route continues to the next rule and ends at `main`, so a
per-WAN table that loses its default (link down, DHCP lease gone, uplink
switched to a different WAN) quietly sends that path out another WAN — the bond
then has two paths on one uplink and none on the intended one, and nothing in
the UI says so. Give every per-WAN table a blackhole default with a worse
metric, so a missing route drops instead of leaking:

```sh
ip route add blackhole default table tbwanN metric 1000
```

Then a path whose WAN is down fails to send, which is what a path is supposed
to do. The same rule applies to the namespace side: its only route is the veth,
so it cannot pick another uplink by accident. When a WAN is DHCP or a repeater
uplink, its gateway and device change: the rules must be rewritten from the
same events that reconfigure that uplink, which is where most of this design's
maintenance cost sits.

**The SNAT inside each namespace is not cosmetic.** WireGuard's cryptokey
routing drops a packet whose source isn't in the peer's AllowedIPs, so a packet
injected into the instance's tun with the veth's private source is discarded by
the server. Masquerading to the instance's own `100.x` satisfies that check and
gives each path the distinct source address the multipath needs; conntrack
un-NATs the replies, so mqvpn sees the server's `100.y` as the peer.

Server side: the existing `tailscaled`, an ACL letting the path nodes reach
UDP 443, a host firewall accept for UDP 443 on `tailscale0` (the Fedora hook
opens only the WAN zone), and an MTU/PathMTU setting.

## MTU: the concrete engineering blocker

`tailscale0` is 1280 by default. With DF set the largest IPv4 UDP payload it
carries is 1252 bytes; WireGuard's worst case adds 80. mqvpn's own defaults are
larger than that — `PathMTU` unset means 1400-byte QUIC packets, which yield a
~1382-byte tunnel MTU — so on a tailnet path the outer packets are what need
shrinking, in one of three shapes:

1. **IPv4-only, sized to fit** — set `PathMTU = 1280` on both ends. mqvpn then
   emits 1216-byte QUIC packets and derives an inner MTU near 1198, under the
   path's 1252, so nothing fragments. mqvpn accepts a configured `PathMTU` of
   1280 (its validator takes 0 or 1280–9000), and an **IPv4-only** tunnel may
   run below 1280 — that is documented behaviour, not a workaround. Drop
   `Subnet6` for these deployments. The cost is roughly 14.5% less payload than
   a 1500-uplink bond. This is the shape to build.
2. **IPv4-only with `PMTUD = true` on both ends** — mqvpn starts at the size
   that gives a 1280 inner, steps down to QUIC's 1200-byte minimum if the path
   is smaller, and searches upward. Same outcome without hard-coding a number,
   and it adapts when a hotel's path is worse; it needs both ends and it probes.
3. **IPv6 inside the tunnel** — requires room for a complete 1280-byte packet,
   so `PathMTU` must be at least **1378**, or **1386** with reorder enabled.
   `tailscale0` at 1280 cannot carry that, and the failure is not one clean
   rejection: a *configured* `PathMTU` below 1378 is refused at server startup,
   while at runtime the path drops the oversized packets and the client closes
   with *"IPv6 requires an inner MTU of at least 1280"*. Raising `tailscale0` (via `TS_DEBUG_MTU` or `ip link set`) puts
   ~1466-byte WireGuard packets on the underlay and exceeds Tailscale's tested
   safe wire MTU of 1360, so **treat IPv6 as out of scope for this design**.

Two corrections worth keeping in mind, because both were wrong in the first
draft of this note: mqvpn's 1280 floor applies to the *configured* `MTU` and
`PathMTU`, not to what a connection negotiates (IPv4-only can end up below it);
and the server's TUN MTU does **not** follow `PathMTU` — the server sets its TUN
once at startup (1382 by default, or its configured `MTU`) and answers oversized
packets with ICMP "packet too big" so the inner senders shrink.

`PathMTU` is connection-wide, so a bond mixing a 1500-underlay direct path with
1280 tailnet paths is sized for the smaller. Tailbond writes no `PathMTU` and no
`PMTUD` anywhere today, and its client `MTU` (default 1280) is only a cap that
cannot go below the validator's floor; adding `PathMTU`/`PMTUD` to
`mqconf.Client` and `mqconf.Server` is required for shape 1 or 2.

## Costs

- **Memory.** Config-dependent: ~11–80 MB private per *loaded* connected
  instance (34 MB RSS idle on a full router setup, 107 MB for a loaded
  service instance with a 365 MB lifetime GC peak). But a *minimal* per-WAN
  instance — no DNS, no routes, the only thing this design needs — measured
  9.3 MB at tailscale v1.104.5 (4.5 MB with `GOMEMLIMIT=6MiB`), with amd64
  and arm64 at parity in Go runtime terms; see
  [design-minitsd.md](design-minitsd.md). With that number three paths cost
  ~14–28 MB, well within a 512 MB router.
- **CPU is not the limit.** The shipped `x/crypto` AEAD does ~0.77–0.87 Gbit/s
  per measured core at 1280-byte packets; roughly 15% of a core per 100 Mbit/s
  of path traffic, ~27% more on a Slate 7's slower core.
- **Tailnet devices.** N+1 per router (the pairing/control node plus one per
  WAN), each a separate tailnet node: reusable tagged auth keys (tags disable
  key expiry), ACLs, device clutter, 50 tagged resources included.
- **Namespaces are not on stock OpenWrt.** `CONFIG_NAMESPACES` is unset in the
  generic 23.05/24.10/main configs, and even x86 lacks `NET_NS`. They are
  present and working on GL.iNet firmware (measured); support would have to be
  per-model, probed at install.
- **Lifecycle.** An init that recreates namespaces, veths and routes at boot and
  after a firmware upgrade; a supervisor that waits for the veths (mqvpn aborts
  if a pinned interface is missing); re-auth after a factory reset.

## What not to build

- **Not the flat-namespace, policy-routing variant.** Every kernel instance's
  startup runs `router.CleanUp`, which deletes the shared `ts-input`/
  `ts-forward`/`ts-postrouting` chains (measured 13 → 0 rules within a second,
  by exact name); a second instance on the default tun name strips the live
  instance's addresses off `tailscale0` before failing `device or resource
  busy`; all instances share policy priorities 5210–5270, bypass mark
  `0x80000/0xff0000` and table 52, where `RouteReplace` gives the destination to
  whichever instance synced last; and control/DERP TCP follows the default
  route, so a dead WAN leaves a path "alive" over another uplink. Upstream
  calls multi-instance per namespace a hack it won't support
  ([tailscale#183](https://github.com/tailscale/tailscale/issues/183)).
  cgroup marking works on this firmware but fixes neither isolation nor IPv6
  source selection.
- **Not macvlan per WAN.** Creation fails `EBUSY` on bridge members; ipvlan
  isn't built.
- **Not a custom kernel for namespaces on stock OpenWrt.** Use GL.iNet firmware
  or a VPS.
- **Not DERP as bonded capacity.** Relayed paths are liveness/backup only; the
  issue's "slower, but it still reaches the server" is true as connectivity and
  false as throughput.
- **Not the phone share relay carrying a router's tailscaled.** It is UDP-only
  and forwards only to its own home server; widening it to WireGuard, STUN and
  DERP makes it a general relay, and it still can't reach a CGNAT'd server.
- **Not SOCKS5 userspace as a drop-in.** `tailscaled --tun=userspace-networking`
  creates no tun (so no mqvpn interface path); its SOCKS5 UDP ASSOCIATE relay
  port is ephemeral per TCP association, userspace networking is beta, and the
  relay is not built for packet rates (8 KiB buffers, 92.9% loss on a
  3000-datagram burst, measured).
- **Not hole punching.** Unimplemented, needs mqvpn changes, and measures ~40%
  against symmetric NATs — the case this exists for.

## If it is ever built

1. The kill test above.
2. **Mechanism proof on one WAN**: two namespaces over the same physical WAN,
   one `tailscaled` each, mqvpn with two paths to the server's `100.y`.
   Bandwidth won't add up (one uplink), but it proves the chain: coexisting
   nodes, per-path binding, the veth/SNAT bridge, distinct sources, server-side
   multipath.
3. **Binding proof with a second WAN**: one namespace per WAN, verify each
   instance's public address differs and each server-side path maps to the
   right node, then compare aggregate throughput with one path.
4. **Measure**: per-WAN direct-vs-DERP and RTT; validated/active path counts and
   sources; inner MTU and `/proc/net/snmp` fragment counters; CPU per 100 Mbit/s
   per process; `VmRSS`/`VmHWM`; reconnect time after a WAN flap and after a
   `tailscaled` restart; WLB loss/jitter with a relayed path in and out.

## Open questions

Whether the Brume 2 and Slate 7 firmwares ship namespaces and survive an
upgrade (only the GL-BE9300 was measured); end-to-end nested throughput
(QUIC-in-WireGuard) on a weak ARM core; DERP capacity and QoS per connection;
whether `tailscaled` tolerates its tun being moved into another namespace
(the fallback path if the veth bridge misbehaves); hostname collisions when
several instances register from one router.
