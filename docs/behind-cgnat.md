# Behind carrier-grade NAT

Bonding is a direct connection: every path of a phone or travel router (Wi-Fi,
cellular, …) sends UDP to the home server's tunnel port (443 by default).
If your provider puts the home line behind **carrier-grade NAT** (CGNAT: the
router's own internet address is in `100.64.0.0/10` or is private while your
public address is different), nothing from outside can reach the server, and
a port forward on your router doesn't help. Tailscale's relays can't stand in:
they carry one path, so they'd turn the bond into a single, slower link.

Setup health shows this as **Your internet line uses carrier-grade NAT**.
The ways out, simplest first:

| | Way | Needs | Status |
|---|---|---|---|
| 1 | A public IPv4 from your provider | A call or a small fee | Works, nothing to set up |
| 2 | Public IPv6 | IPv6 on the home line | Works; detected by itself |
| 3 | The server on a VPS | A small VPS | Works with what's there |
| 4 | Home server, tunnel port relayed through a VPS | A small VPS and WireGuard | **Recommended for CGNAT today**; design below, `server.relayed`, works without tailbond changes |
| 5 | Hole punching between the devices and the home server | Changes to mqvpn | Not implemented |
| 6 | One `tailscaled` per internet connection, each tun an mqvpn path | Network namespaces on the router, N+1 tailnet devices | Researched, not built; [design note](design-tailscaled-per-connection.md) — deferred in favour of 4 |

**Which way.** Ask the provider for a public address first (1): it is free or
cheap and changes nothing else. If that fails, IPv6 (2) needs no work when the
home line has it. Only then choose between a server on a VPS (3) and the relay
(4): the relay keeps the server, its pairings and its state where they are,
and needs no tailbond changes, so it is the default answer. The per-connection
Tailscale design (6) solves the same problem without a VPS but adds a
namespace, a device and a relayed-path problem per WAN; the research is in the
[design note](design-tailscaled-per-connection.md), and it is deferred.

## 1 and 2: a public address

Ask your provider for a public IPv4 ("for reaching my home network from
outside"; often free or a small fee), or use the line's public IPv6.

With no `server.public_addr` set, the server finds its address by itself,
through the public address Tailscale sees it at (`tailscale netcheck`), with
the tunnel port. It picks:

1. **IPv4**, when there is a public one and the router's own address isn't
   carrier-grade NAT. Forward the tunnel port (UDP 443) to the server on
   your router.
2. **IPv6**, when there is no usable public IPv4. Nothing to forward; allow
   the tunnel port in the router's IPv6 firewall (GL.iNet's `wan` zone rule
   from the `server-up` hook already covers both families).
3. Otherwise a guess from the default route, which is a home-network address:
   the panel then asks you for one.

Setting `server.public_addr` (a name, an IP, or `host:port`) always wins.
Clients pick up a changed address by themselves, over Tailscale
([architecture.md](architecture.md), "If the public address changed").

A device whose paths have no IPv6 (many hotel Wi-Fi networks) can't use an
IPv6 server address on those paths, so it bonds fewer links; mobile networks
usually have IPv6.

## 3: the server on a VPS

Run the tailbond server on a small VPS ([vps.md](vps.md)) and pair phones and
travel routers with it. The VPS has a public address, so nothing here depends
on your home line. The VPS is on your tailnet, so bonded devices still reach
your Tailscale machines (including one at home) through it.

- Internet traffic leaves from the **VPS's** address, not your home's.
- Bonded devices reach tailnet machines, not a home LAN behind a Tailscale
  subnet router: tailbond routes the tailnet's own ranges only.
- Making the VPS send its internet traffic out through your home machine (a
  Tailscale exit node on the VPS) is possible in principle. tailbond doesn't
  set it up, and it isn't tested here.
- Keep the VPS first in the server list ([the panel's Servers
  card](glinet.md#admin-panel-vpn--tailbond)); a second server can back it up.

## 4: keep the server at home, relay its port through a VPS

The home server opens a WireGuard tunnel **out** to a VPS (outbound traffic
passes carrier-grade NAT), and the VPS forwards the tunnel port into it.
Devices connect to the VPS's address; the home server answers. Internet traffic
still leaves from home, through the bond, with your home IP.

This is the recommended way out of CGNAT short of getting a public address. It
needs no tailbond code: the relay is a WireGuard tunnel plus forwarding rules
on a box you rent. tailbond doesn't install or manage it. Example addresses:
`10.200.0.1` (VPS) and `10.200.0.2` (home) on a `wg0` between them; `eth0` is
the VPS's internet interface and `203.0.113.10` its public address.

**On the VPS.** WireGuard between the two ends as usual, with
`PersistentKeepalive = 25` on the home side (the home end starts the tunnel, so
the mapping stays open). Then forward the tunnel port, and allow UDP 443 in the
provider's own firewall as well — a cloud security group that drops it looks
exactly like a broken relay:

```sh
sysctl -w net.ipv4.ip_forward=1
iptables -t nat -A PREROUTING  -i eth0 -p udp --dport 443 -j DNAT --to-destination 10.200.0.2:443
iptables -t nat -A POSTROUTING -o wg0 -p udp -d 10.200.0.2 --dport 443 -j MASQUERADE
iptables -A FORWARD -i eth0 -o wg0 -p udp -d 10.200.0.2 --dport 443 -j ACCEPT
iptables -A FORWARD -i wg0 -o eth0 -m conntrack --ctstate ESTABLISHED,RELATED -j ACCEPT
```

The `MASQUERADE` rule makes replies come back through `wg0` with no policy
routing at home.

**At home.** Allow UDP 443 **in** on the WireGuard interface: tailbond's own
firewall rules cover the `wan` zone, or the default route's zone, only. Either
put `wg0` in a zone that allows UDP 443 or add the rule yourself. Then point
tailbond at the relay in `/etc/tailbond/tailbond.json` `server` (GL.iNet UCI:
`config server`):

```json
"public_addr": "203.0.113.10:443",
"relayed": true
```

```
option public_addr '203.0.113.10:443'
option relayed '1'
```

`relayed` makes Setup health stop checking the home line's own address, which
can never pass behind CGNAT. Devices already paired fetch the new address over
Tailscale by themselves; a new device pairs as usual.

**Verify it.** From outside the home network, with a phone on mobile data
(Wi-Fi off), open the dashboard's Setup tab: it should show the tunnel port
reachable, and a paired device should connect with the VPS as its peer. On the
VPS, `conntrack -L -p udp --dport 443` shows the flow, and
`tcpdump -ni wg0 udp port 443` shows it arriving at the home end.

**Sizing and MTU.** Every packet takes the detour, so throughput is bounded by
the VPS's bandwidth (its egress is billed on most providers) and by the home
line's upload — the bond can't exceed either. WireGuard's overhead sits on top
of the tunnel's own: if large packets stall or the dashboard shows reconnects,
lower `MTU` in the mqvpn config on both ends (1382 → 1340 is the first thing to
try for a `wg0` with MTU 1420), and lower the WireGuard `MTU` on the VPS side
if the underlay is PPPoE.

**What it costs you.** The home server sees every device as the VPS, so its
logs, its per-device data accounting and its "who is connected" view lose the
real addresses (pairing and approval still work — they run over Tailscale,
which is unaffected). The VPS is a single point of failure and a monthly cost
(a few dollars; more if the traffic is heavy). And it is a detour: a direct
public address, if you can get one, is always better.

**What it does not fix.** The relay only carries the tunnel port. Nothing else
on the home server becomes reachable from the internet, and the bond's paths
still all end at the VPS — the relay is one address, so a path that can't
reach the VPS is still a path that's down.

## 5: hole punching

The server could open NAT mappings toward each device path before the device
connects, coordinated over Tailscale. It isn't built: mqvpn would have to send
from the server's listening socket toward the device's observed addresses,
and tailbond would have to carry those addresses between the two. Even then
it only works when the NATs keep one mapping per local port; a NAT that picks
a new mapping per destination (common on mobile networks) defeats it.
[RFC 5128](https://www.rfc-editor.org/rfc/rfc5128.html) describes the
limits, and a [measurement of IPFS hole punching](https://arxiv.org/abs/2604.12484)
saw about 83% success against port-restricted cone NATs and about 40%
against symmetric ones. Use 1 to 4; option 6 is researched but deferred.
