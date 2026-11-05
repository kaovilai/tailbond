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
| 4 | Home server, tunnel port relayed through a VPS | A small VPS and WireGuard | Works in principle; recipe below, `server.relayed` |
| 5 | Hole punching between the devices and the home server | Changes to mqvpn | Not implemented |

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
Devices connect to the VPS's address; the home server answers. Internet
traffic still leaves from home.

This is a recipe, not a tested feature: tailbond doesn't install or manage the
relay. Example addresses: `10.200.0.1` (VPS) and `10.200.0.2` (home) on a
`wg0` between them; replace `eth0` with the VPS's internet interface and
`203.0.113.10` with its public address.

**On the VPS** (WireGuard set up as usual, `PersistentKeepalive` on the home
side; allow UDP 443 in the provider's firewall too):

```sh
sysctl -w net.ipv4.ip_forward=1
iptables -t nat -A PREROUTING  -i eth0 -p udp --dport 443 -j DNAT --to-destination 10.200.0.2:443
iptables -t nat -A POSTROUTING -o wg0 -p udp -d 10.200.0.2 --dport 443 -j MASQUERADE
iptables -A FORWARD -i eth0 -o wg0 -p udp -d 10.200.0.2 --dport 443 -j ACCEPT
iptables -A FORWARD -i wg0 -o eth0 -m conntrack --ctstate ESTABLISHED,RELATED -j ACCEPT
```

The `MASQUERADE` rule makes replies come back through `wg0` with no policy
routing at home. The cost: the home server sees every device as the VPS, so
its logs don't show real addresses.

**At home:** allow UDP 443 **in** on the WireGuard interface (tailbond's
firewall rules cover the `wan` zone or the default route's zone only: add the
interface to a zone that allows UDP 443, or a rule of your own). Then, in
`/etc/tailbond/tailbond.json` `server` (GL.iNet UCI: `config server`):

```json
"public_addr": "203.0.113.10:443",
"relayed": true
```

```
option public_addr '203.0.113.10:443'
option relayed '1'
```

`relayed` makes Setup health stop checking the home line's own address. Devices
already paired fetch the new address over Tailscale by themselves.

Know before you rely on it: every packet takes the detour through the VPS,
so latency and the usable bandwidth are bounded by the VPS and by home's
upload. If large packets stall, lower the WireGuard `MTU`.

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
against symmetric ones. Use 1 to 4.
