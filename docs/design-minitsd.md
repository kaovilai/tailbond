# Design note: deconstructing tailscaled (one process, N engines)

**Status: researched, measured, not implemented — and the measurements say
don't.** The memory premise is gone: a stock `tailscaled` is ~9 MB resident
today, ~4.5 MB with `GOMEMLIMIT`, so running one per WAN costs less than the
deconstruction would save. This note exists so the option isn't re-litigated,
so the numbers below are not rediscovered, and so the maintenance machinery —
the part "we have CI" would have to carry — is designed for whoever revisits
it. Follow-up to [one tailscaled per internet
connection](design-tailscaled-per-connection.md), which is itself parked
behind the [VPS relay](behind-cgnat.md).

The question it answers: `tailscaled` is open source (BSD-3) and the per-WAN
design runs one instance per internet connection. Could the instances be
*deconstructed* — one process holding N identities, N engines, one Go runtime
— so only "the part that needs splicing across WANs" is kept and memory is
minimal?

## Verdict

**The deconstruction is buildable, CI can carry its maintenance, and it is
not worth building: at three WANs it saves nothing — the shared-runtime
shape measured *worse* than separate stock processes.** Measured 2026-11-08
against a local headscale, tailscale `v1.104.5` (tag commit `78838476328d`),
Linux 6.18, idle registered nodes, kernel tun:

| shape | resident memory |
|---|---|
| stock `tailscaled`, one process per node | **9.3 MB per instance** (3 instances: 9.1–9.3) |
| stock + `GOMEMLIMIT=6MiB` | **4.5 MB per instance** |
| one process, N embedded nodes (tsnet, userspace netstack) | 18.9 MB baseline + **~5.8 MB per node** → 34.7 MB at 3 nodes |

Three WANs cost **28 MB as three stock processes**, or **~14 MB tuned with
`GOMEMLIMIT`** — no fork, no treadmill. The one-process form starts at 18.9 MB
before the first node exists. A kernel-tun deconstruction would land between
the two (shared control + magicsock per engine, no netstack), but it would
have to beat ~3 MB/node marginal to matter, and it inherits the maintenance
treadmill below to get there.

**Architecture is not the explanation for bigger numbers elsewhere.** The
same one-node probe run natively (amd64) and under qemu-aarch64 (guest-truth
Go runtime stats, unpolluted by the emulator) came out at Sys 22.9 MB vs
19.1 MB, heap 6.2 vs 6.3 MB — parity. What differs is *configuration and
load*: the parent note's router measurements (~34 MB idle, ~107 MB for a
loaded service instance) are a fuller setup — real tailnet netmap, DNS,
routes — and an older tailscale. The minimal per-WAN instance this design
needs is the cheap row of the table, not the loaded one; a fork cannot fix
load-driven memory anyway, because per-path buffers scale with traffic in
either shape.

The parent note's cost line — "tens of MB per Go tailscaled" — is therefore
config-dependent rather than universally true; it is corrected there to
point here. The parent note's *other* reasons for parking stand unchanged
(DERP is liveness not capacity, double wrapping, one tailnet device per
WAN, phones can't).

## What tailscaled is made of, and what the splice needs

| part | needed per WAN? | note |
|---|---|---|
| control client (TS2021 noise login, netmap, key rotation) | yes | what makes the instance a distinct tailnet node with its own `100.x`; control is per-node, so N identities means N control connections — this cannot be collapsed |
| magicsock (STUN, disco, hole punching, DERP client) | yes | the entire point; deeply coupled to control |
| WireGuard + tun | yes | wireguard-go does the crypto in userspace even with a kernel tun (the tun is just the interface — the parent note's AEAD numbers are wireguard-go's); the tun is what mqvpn's paths bind to |
| routes/ACL application, netfilter, MagicDNS, localapi, SSH, health/metrics, userspace netstack | no | roughly the difference between `tailscaled` and a splice |

The kept parts are the heart of `tailscaled`; the discarded parts are the
trimmings. There is no clean seam that leaves the crown jewel.

## The architecture that would work

Same topology as the parent note — one network namespace per WAN, veths back
to the single mqvpn — with the N daemons replaced by one process:

```
 main namespace                                  per-WAN namespace N
 ┌───────────────────────────────┐              ┌────────────────────────────┐
 │ mqvpn  --path tbv0 --path tbv1│  veth pair   │  (created by minitsd)      │
 │  socket bound to tbvN         │ tbvN ────────│ tailscale0 = 100.x/32      │
 └───────────────┬───────────────┘ 10.99.N.1/.2 │ SNAT veth -> 100.x on tun  │
                 │                              └─────────────┬──────────────┘
 ┌───────────────┴──────────────────────────┐                 │ hole punch / DERP
 │ minitsd (one process, one Go runtime)    │           WAN N │
 │  engine 0..N-1: controlclient + magicsock│                 ▼
 │   + wgengine + tstun  (per WAN)          │   server: one tailscaled + mqvpn
 │  per-engine socket steering (see below)  │           on 0.0.0.0:443
 └──────────────────────────────────────────┘
```

The hard part is not the engines, it is **steering each engine's sockets
out the right WAN**. Go creates sockets on arbitrary threads, and a Go
control hook (`net.Dialer.Control`) runs *after* socket creation, so a
namespace cannot be chosen there — sockets belong to the namespace they
were created in. `tailscaled` itself does not setns at all: verified at
v1.104.5, `tailscale.com/net/netns`'s Linux control functions set
`SO_MARK` (or `SO_BINDTODEVICE`) — `setsockopt` calls, valid at that
point. A deconstruction gives each engine its own control function and
routes accordingly:

- **Per-engine `SO_BINDTODEVICE`** to WAN N's interface: the same knob
  mqvpn uses for its paths; no mark space to collide in, but every WAN
  must be an interface in the main namespace.
- **Per-engine `SO_MARK` + policy routing**: the marks must not collide
  with tailbond's own fwmark-keyed per-WAN tables
  (`packaging/openwrt/hooks/bypass-apply`); reserve a range.

Both keep all engines in one namespace; the parent note's per-WAN
namespace isolation is *not* available to a single process, which is
itself a reason to stay with separate stock daemons.

Either way the parent note's blackhole-default per-WAN tables stay: a table
that loses its default must drop, not leak onto another WAN (two paths on one
uplink, silently).

## The maintenance treadmill (what "we have CI" has to carry)

`tailscaled`'s stable channel ships exactly the `vX.Y.Z` tags in the OSS repo
(one release train a month, patch tags on the series; at the time of writing
stable is `v1.104.5` → commit `78838476328d`, and a running node reports the
same mapping in `tailscale version`). The pin discipline: vendor at the newest
**stable tag** — never `main`, never `-pre` — and re-vendor on a cadence
Tailscale chooses: the coordination server ages clients out, so "pin one tag
forever" is not available.

- **Re-vendor PR**: scheduled weekly job bumps the pin, builds `minitsd`,
  runs the harness below, opens the PR. Compile breakage — the most common
  treadmill failure — shows up here.
- **The harness**: a local [headscale](https://github.com/juanfont/headscale)
  control server inside the CI job (the same double this note's numbers were
  measured against): `minitsd` joins, gets its `100.x`, punches to a peer
  across namespaces. `scripts/e2e-cgnat-relay.sh` models an inbound-filtering
  carrier (MASQUERADE plus drop-unsolicited), not endpoint-dependent
  mapping — plain MASQUERADE preserves the source port across
  destinations, so it is not a symmetric NAT. Punch assertions under
  symmetric rules are new harness work: `SNAT --random-fully` per flow
  gives the destination-dependent ports.
- **The canary**: headscale tracks the client protocol but is not Tailscale's
  production coordination server. A small periodic job logs one real node
  into the real tailnet with a `TS_AUTHKEY` secret — same style as the
  real-GitHub-redirect test — to catch protocol drift on their server side.
- **CVEs**: Dependabot on the vendored module; the re-vendor cadence bounds
  exposure.

## Breakage shapes

Each row: how it breaks, what catches it, what doesn't.

| # | shape | caught by | not caught by |
|---|---|---|---|
| 1 | **Compile drift**: packages move/rename between stable tags | re-vendor PR fails to build | — |
| 2 | **Behavioral drift behind a stable API**: punch heuristics, disco, netmap field semantics change | harness assertions (join, address, punch under synthetic NAT) | punch *success rate* on real cellular; DERP fallback character |
| 3 | **Control-protocol drift** (Tailscale's server side): production coordination requires newer protocol while headscale still accepts ours | the canary job | the harness (headscale is not production) |
| 4 | **Headscale-vs-production divergence**: green harness, broken login | the canary (if it overlaps) | anything short of a real login |
| 5 | **Socket steering regressions**: a `net/netns` or Go runtime change breaks per-engine `SO_BINDTODEVICE`/`SO_MARK` steering — an engine's traffic leaves the wrong WAN, two paths on one uplink, silently | blackhole defaults turn it into a dropped path; harness's per-path source-address assertions | the mis-steered failure when the wrong WAN still routes |
| 6 | **fwmark collision**: per-engine marks overlap tailbond's per-WAN tables | harness routing assertions; explicit reserved range | nothing, if unreserved |
| 7 | **GC coupling**: one engine's pressure throttles all engines sharing the runtime; `GOMEMLIMIT` tuning becomes a whole-process compromise | load tests | idle measurements like this note's |
| 8 | **tun/netns lifecycle races** on engine restart inside one process | restart-under-load in the harness | rare interleavings |
| 9 | **CVE accumulation** in the vendored tree | Dependabot | — |

Rows 5 and 7 are the ones that argue against the shared process even at
parity: separate daemons cannot break each other's routing or GC; one process
can.

## What not to build

- **tsnet as the engine**: today's tsnet doc even advertises "multiple
  independent Tailscale nodes within a single binary" — but it is gVisor
  userspace netstack with no kernel tun, and mqvpn's paths need a real tun
  to bind. (The parent note's 92.9% burst loss was measured on mqvpn over
  a SOCKS5 UDP-ASSOCIATE relay, not on tsnet's data path — evidence that
  userspace-relayed datagrams hurt that shape, not a tsnet measurement;
  the tun requirement is the reason, not the number.)
- **Collapsing the N control connections into one**: separate node identities
  are the feature (one `100.x` source per path); control is per-node.
- **A general-purpose "mini-tailscale"**: the splice should carry only what
  the per-WAN paths need, or it grows its own maintenance surface on top of
  the treadmill.

## If it is ever un-parked

The gate in front of everything is still the parent note's hour-long
experiment (`tailscale ping` the server from each WAN; DERP-only cellular
kills the flagship case regardless of memory). Then, in order:

1. `minitsd`: one engine, kernel tun (fork tsnet's construction path, swap
   netstack for `tstun`), joins headscale in CI. Success: registered,
   addressed, punching.
2. N engines in one process with per-engine steering; measure marginal
   MB/node against this note's 9.3/4.5 MB stock numbers — the fork has to
   beat the *tuned* stock number to justify existing.
3. mqvpn path integration against the relay-e2e NAT lab.

Revisit triggers: a tailscale release whose resident size regresses past
~30 MB again; hardware smaller than ~256 MB; more than ~6 bonded paths.
