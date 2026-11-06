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
2. From a node on the Wi-Fi/Ethernet side, and from a node on cellular (a phone
   with Tailscale, or a laptop tethered to one): `tailscale ping <server>` three
   to five times each per WAN type.
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
passes for that device. Masquerade each veth subnet out its WAN — tailbond
already builds per-WAN tables for bypass rules.

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
carries is 1252 bytes; WireGuard's worst case adds 80. That leaves an mqvpn
inner of 1182 (1174 with the reorder header) — **below mqvpn's 1280 floor**, so
mqvpn refuses, and IPv6 is impossible (`IPv6 requires an inner MTU of at least
1280`). Three shapes exist:

1. **IPv4-only inner at 1182** — set `PathMTU=1280` on both ends, leave `MTU`
   unset so mqvpn's auto value follows it, drop `Subnet6` for these
   deployments. Clean, no fragmentation, about 14.5% less payload than a
   1500-uplink bond. The recommended shape if this is ever built.
2. **IPv6 with fragmentation** — today's config (no `PathMTU`): mqvpn
   negotiates ~1382-byte QUIC packets and `tailscale0` fragments every
   full-size one, with GSO falling back. Works, costs throughput and CPU.
3. **Raise `tailscale0` to ~1386** so a 1280 inner fits — needs `TS_DEBUG_MTU`
   or `ip link set`, puts ~1466-byte WireGuard packets on the underlay, and
   exceeds Tailscale's tested safe wire MTU of 1360.

`PathMTU` is connection-wide, so a bond mixing a 1500-underlay direct path with
1280 tailnet paths is sized for the smaller. Tailbond writes no `PathMTU`
anywhere today; adding it to `mqconf.Client` and `mqconf.Server` is required
either way.

## Costs

- **Memory.** ~11–80 MB private per connected instance (34 MB RSS idle, 107 MB
  for a loaded service instance with a 365 MB lifetime GC peak). Three fit a
  1 GB router; four to five risk the OOM killer. Both the Slate 7 and the
  Brume 2 have 1 GB; a 512 MB model would cap at two.
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
