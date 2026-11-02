# tailbond on a Fedora home server

This guide sets up the **home end** of tailbond on Fedora 43 or newer. Your phone or travel router bonds its Wi-Fi, 5G and Ethernet links to this server. Its traffic then leaves through your home internet connection, and your Tailscale tailnet is reachable through it.

For a VPS as a second (backup) server, or another distribution (Debian, Ubuntu, AlmaLinux/RHEL), see [vps.md](vps.md).

## 1. Install

```sh
sudo dnf install ./tailbond-server-*.rpm
sudo systemctl enable --now tailbond
```

The package contains:

| Path | What |
|---|---|
| `/usr/bin/mqvpn` | the data plane (mqvpn 0.19.0, Multipath QUIC) |
| `/usr/bin/tailbondd` | the agent: supervises mqvpn, serves the dashboard on port 8077 |
| `/etc/tailbond/tailbond.json` | configuration (kept on upgrade) |
| `/var/lib/tailbond/` | generated `server.conf`, certificates, data usage |
| `/usr/lib/tailbond/hooks/` | firewall and Tailscale glue (`server-up`, `server-down`) |
| `/usr/share/tailbond/tailbond.apk` | the Android app, offered to phones on the phone page (when the package was built with it) |
| `tailbond.service` | systemd unit |
| firewalld `tailbond` service, zone and policies | UDP 443, `tb0`, forwarding and NAT |

Check that it runs:

```sh
systemctl status tailbond
journalctl -u tailbond -f
```

To open the dashboard from the server itself, browse to <http://127.0.0.1:8077/>. From another of your devices on Tailscale use `http://<server>.<tailnet>.ts.net:8077/` (or `http://<server>:8077`), since the agent also listens on the server's Tailscale IPs. Without a password, the dashboard answers only devices of the Tailscale account this server belongs to (and the server itself); see [who can approve](#who-can-approve-and-what-an-approved-device-reaches).

The agent also serves a **phone page** on port 80 of its Tailscale IPs: typing the server's name with a slash (`homebox/`; without the slash, phone browsers search for the word instead) into a phone's browser shows *Get tailbond* with the Android app and three short steps. It offers only the app and the steps (no dashboard, no API, nothing that changes the server). If something else already uses port 80 on those addresses, the page is at `http://<server>:8077/get` instead and the dashboard points there. Turn it off with `"landing_port": -1`.

Optional HTTPS (`https://homebox.<tailnet>.ts.net/`): `sudo tailscale serve --bg http://100.x.y.z:80`, using this server's Tailscale IPv4, proxies only the phone page. It isn't needed (Tailscale already encrypts the connection, and phones type the plain name), and **don't** point `tailscale serve` at port 8077: every request would then arrive from tailscaled on this machine, so the dashboard would see neither the device's Tailscale identity nor its address, and pair requests would be refused.

What the package changes in firewalld (the `server-up` hook, every start, both the running and the saved configuration; `dnf remove` takes the ports and rules back out). There's nothing to do here yourself; this list is for checking:

- UDP 443 (the `tailbond` service) in the zone of the interface that has the default route, the one your router's port forward arrives on (else the default zone). Check with `sudo firewall-cmd --get-active-zones` and `sudo firewall-cmd --zone=<zone> --list-services`.
- tcp 8077 and 80 from tailnet addresses only (`100.64.0.0/10`, `fd7a:115c:a1e0::/48`), in the zone `tailscale0` is in (else the default zone): `sudo firewall-cmd --zone=<zone> --list-rich-rules`. Tailscale drops those source addresses on any other interface, so this doesn't open the ports to your LAN.
- UDP 41641 (Tailscale's own port) in the same zone as UDP 443 when Tailscale is installed, so a forward on your router is all direct Tailscale connections need.
- `tb0` in the `tailbond` zone, and the `tailbond-out` policy: bonded devices may use this machine's services on its tunnel address (`10.77.0.1`, e.g. the DNS forwarder; the agent answers them there only with the machine list, never the dashboard), and reach the internet, your home LAN and your tailnet through it. To keep them off your home LAN, add a rule of your own; the hook only re-adds its masquerade rules and leaves yours alone:

  ```sh
  sudo firewall-cmd --permanent --policy=tailbond-out --add-rich-rule='rule family=ipv4 destination address=192.168.1.0/24 reject'
  sudo firewall-cmd --reload
  ```

UDP 443 is also what HTTP/3 uses: if this machine runs a web server with HTTP/3, set another port (`"server": {"port": 8443}`) and forward that one instead. ([vps.md](vps.md#2-ports-on-a-box-that-already-runs-a-web-server) has more on sharing a box with a web server.) SELinux needs no changes: the service runs unconfined as root, like other VPN servers.

## 2. Make the server reachable: forward UDP 443

Bonding needs each of the device's links to reach the server **directly over UDP**. It does not go through Tailscale's relays. Until this is done, devices only connect at home; the dashboard's Devices tab shows it as a setup step at the top (and a line on every tab).

1. On your home router's admin page, find **Port forwarding** (sometimes called *Virtual server* or *NAT*) and add: protocol **UDP**, port **443**, to this server's LAN address (`ip -4 addr` shows it, e.g. `192.168.1.2`). Give the server a fixed address on the same router (**DHCP reservation**, sometimes *static lease*), so the forward keeps pointing at it.
2. Find your home's public address: the router's status page shows it (*WAN* or *Internet* IP). If your router has a **Dynamic DNS** page, a DDNS name (such as `home.example.net`) is better, since it keeps working when the address changes.
3. Tell tailbond: type that address into the setup step at the top of the dashboard's **Devices** tab and click **Save** (the port is added for you). Or edit the config and restart:

   ```json
   "server": { "public_addr": "home.example.net", ... }
   ```

   `sudo systemctl restart tailbond`.
4. Once a phone is paired (section 4), test the forward from outside: on that phone, turn Wi-Fi off; once tailbond connects over mobile data, the dashboard's *UDP 443* health item turns green.

**Carrier-grade NAT**: if the router's own WAN address is in `100.64.0.0`–`100.127.255.255`, or is private (`10.x`, `192.168.x`, `172.16–31.x`) while your public address is different, your provider shares one address between customers and a port forward can't work. Ask the provider for a public IPv4 address, or use a public IPv6 address for `public_addr` (for example `[2001:db8::10]:443`) and allow UDP 443 in the router's IPv6 firewall. The dashboard's setup health detects this.

## 3. Tailscale (optional, recommended)

tailbond works without Tailscale. With Tailscale running on the server (`sudo tailscale up`), you also get:

- **Tailnet access for bonded devices.** Traffic from the tunnel (`10.77.0.0/24`) to `100.64.0.0/10` and `fd7a:115c:a1e0::/48` leaves through `tailscale0` with masquerade. To the tailnet it looks like it comes from the server, so bonded devices don't show up as extra machines.
- **MagicDNS for bonded devices.** The agent answers DNS on `10.77.0.1`: `*.ts.net` and your tailnet's MagicDNS names go to `100.100.100.100`, everything else to the server's normal resolver.
- **Reaching bonded devices from the tailnet (optional).** The `server-up` hook runs `tailscale set --advertise-routes=10.77.0.0/24` and keeps any routes you already advertise. Bonded devices reach your tailnet without it; approve it only if your Tailscale machines should reach the bonded devices (it exposes them to your tailnet, under your ACLs). An `autoApprovers` rule in your policy approves it without asking. To not advertise it at all, set `"no_advertise_route": true` under `"server"` in `tailbond.json`. **To approve it:** open the [admin console](https://login.tailscale.com/admin/machines), find this machine, choose **Edit route settings**, and enable `10.77.0.0/24`. The dashboard's **Tailscale** tab shows whether the route is advertised and approved.
- **A real TLS certificate.** If HTTPS certificates are enabled for your tailnet (admin console → DNS → HTTPS Certificates), the agent runs `tailscale cert` for `<server>.<tailnet>.ts.net`. mqvpn then uses that certificate, and devices check it by name even though they connect to your public address. Without it, the agent creates a self-signed certificate, and its SHA-256 fingerprint goes into every pairing link.

If you switch between the self-signed and the Tailscale certificate (by turning tailnet HTTPS on or off), pair your devices again: the old links pin the old certificate.

For best speed (direct Tailscale connections, the port, the route), see
[setup tips](setup-tips.md); the dashboard's **Devices** tab shows a
**Setup health** card that checks them. What each item means:

| Health item | Means | Fix |
|---|---|---|
| Carrier-grade NAT (red: stops devices connecting from away) | your provider shares your internet address | section 2, *Carrier-grade NAT* |
| UDP 443 not confirmed from outside yet (?) | nothing has connected from away yet | none: the first device connecting from away confirms it |
| Route 10.77.0.0/24 isn't approved yet (optional) | a choice, not a speed problem: approving exposes bonded devices to your tailnet (under your ACLs); devices already reach your machines without it | admin console → Machines → this machine → **Edit route settings** → tick it. Or approve it automatically with an ACL [`autoApprovers`](https://tailscale.com/docs/reference/syntax/policy-file#autoapprovers) entry |
| *N* machines through a relay (slower) | Tailscale traffic between this machine and those machines goes through Tailscale's relays (DERP); only traffic to *those* machines is slower, bonding itself isn't affected | on your router, forward **UDP 41641** to this machine (this machine's firewall already allows it; see section 1). UPnP on the router also works but lets any device on your LAN open ports. Check with `tailscale ping <machine>`: "via DERP(…)" is relayed, "via <ip>:<port>" is direct. A relay can also come from the other machine's network, which you may not control |
| Anyone on your Tailscale network can approve devices | the server is tagged (no owner account) and has no approvers or password | name approvers or set a password (Devices → Who can approve) |
| This network makes direct connections hard ("hard NAT") | the router in front of this machine picks a new outside port for every destination | same fix as the relay item (one forward fixes both) |

## 4. Add a device

**A GL.iNet travel router (no SSH).** Open **Devices → Add a GL.iNet travel
router**: download the router package (served as `/tailbond-glinet.ipk` when
`/usr/share/tailbond/tailbond-glinet.ipk` exists, or `glinet_ipk` is set),
install it on the router through LuCI, then on the router open **VPN →
tailbond → Ask to connect**. The request shows up at the top of this page
with the router's Tailscale identity; tap **Approve**.

See [who can approve](#who-can-approve-and-what-an-approved-device-reaches) below.

**An Android phone that runs Tailscale.** The phone must be on your
Tailscale network: simplest, invite the person (admin console → Users →
Invite); sharing just this machine with their account works too, but once
you approve their phone it reaches what this server reaches either way. It
must also be allowed to
reach this machine on tcp 80 and 8077 (Tailscale's default policy allows
it). On the phone, with Tailscale on, open the browser and type the
server's name with a slash (`homebox/`; without it the browser searches), or scan the QR code
under **Devices → Add an Android phone**. The phone page offers the app
(with pictures of each Android install prompt) and an **Open tailbond and
ask homebox** button: the app opens, asks this server, and the request shows
up at the top of the dashboard with the phone's Tailscale identity. Tap
**Approve**; the phone shows up under the devices, connected, a few seconds
after its owner turns tailbond on. For family, send them the link from that
card, or tell them to type `homebox/`.

The page offers the app when `/usr/share/tailbond/tailbond.apk` exists (the
RPM ships it when it was built with one; or set `android_apk` in
`tailbond.json`). There is no store listing, so Android asks the person to
allow installs from their browser once; the page shows those prompts.

### Who can approve, and what an approved device reaches

- **No password (default)**: the dashboard (and so approving, adding and
  removing devices) answers the Tailscale account this server belongs to, from
  any device signed in to it (including a router or phone you lent to someone
  while it's signed in to your account: sign those in with the borrower's
  account, or tag them), and anyone with a login on this machine
  (`http://127.0.0.1:8077`): any local user or program can use it without
  a password, so set one if this machine runs things you don't trust. Everyone else on your tailnet, including people
  you share this machine with, gets a short "this page is only for …" note;
  they can still **ask to connect** (that's how their phone or router gets
  added) and open the phone page. A *tagged* server has no owning account:
  then anyone on the tailnet who can reach it can use the dashboard, so set a
  password.
- **More approvers, no password**: add their Tailscale logins under
  **Devices → Who can approve and add devices** (or `"approvers":
  ["sam@example.com"]` in `tailbond.json`). On a tagged server, the approvers
  are the only ones.
- **Password** (same card, or `"ui_password"` in `tailbond.json`): replaces
  the rules above. Anyone with the password, from any tailnet device, and
  everyone (you and this machine too) needs it; it protects the whole
  dashboard API, including `/api/clients`. The curl examples below then need a login first:
  `curl -c jar -H 'Content-Type: application/json' -d '{"password":"…"}' http://127.0.0.1:8077/api/login`, then `-b jar` on each call.
- **What an approved device reaches**: the internet through your home
  connection, your home LAN, and your tailnet. To the tailnet its traffic
  comes from this server, so **your Tailscale access rules for this server
  apply to every bonded device**. Approve only devices you'd trust with this
  server's access. To limit it, tag the server and give the tag only what
  bonded devices should reach, for example:

  ```json
  "tagOwners": { "tag:tailbond": ["autogroup:admin"] },
  "grants": [
    { "src": ["tag:tailbond"], "dst": ["tag:nas"], "ip": ["*"] },
    { "src": ["autogroup:member"], "dst": ["tag:tailbond"], "ip": ["tcp:80", "tcp:8077"] }
  ]
  ```

  Keep tcp 80 and 8077 to the server open for the people whose devices should
  ask to connect (the phone page and pair requests use them). A tagged server
  has no owner: add the approvers (or a password) **before** tagging it,
  so there's no moment when anyone can approve; setup health warns while a
  tagged server has neither. The `grants` above only narrow anything once
  your policy's default allow-all rule is removed.

**With a QR code** (phone and computer, or a travel router):

1. In the dashboard, open **Devices**, type a name (for example `pixel` or `slate7`), and press **Add**.
2. A QR code and a `tailbond://pair?d=…` link appear. Each device gets its own key, so you can revoke one without touching the others.
   - **Phone:** scan the QR code with the tailbond app.
   - **Travel router (tailbondd in client role):** open its dashboard, go to **Setup**, and paste the link or scan the QR code. Then press **Pair & connect**.
3. The device appears under **Connections** with one card per link (path), showing live rate, latency and loss.

To show the QR code again later, use the QR button next to the device. **Revoke** (the trash button) deletes the key and disconnects the device.

The same steps from a shell on the server (with a password set, log in first as shown above):

```sh
curl -s -X POST -H 'Content-Type: application/json' -d '{"name":"pixel"}' http://127.0.0.1:8077/api/clients | jq -r .uri
curl -s http://127.0.0.1:8077/api/status | jq '.clients'
curl -s -X DELETE http://127.0.0.1:8077/api/clients/pixel
```

## Modes

Pick a mode in the dashboard. Changing it rewrites mqvpn's config and restarts it, which takes a few seconds.

| Mode | Use it for |
|---|---|
| Speed | the most bandwidth: every link carries traffic |
| Streaming | calls and streams that must survive a link dropping (UDP is duplicated) |
| Redundant | the most reliability: every UDP packet goes over every link |
| Failover | one primary link, with the others on standby |

The server's mode sets its default for downloads. Each device's own mode decides its connection.

## Updates

The server updates itself in its maintenance window (default policy `auto`;
**Settings → Updates**) and serves each release to your routers and phones
(the mirror). Updates go to `/var/lib/tailbond/update/`, never over the
RPM's files, so `rpm -V tailbond-server` stays clean; a `dnf upgrade` to a
newer package simply takes over. How it all works: [updates.md](updates.md).

## Troubleshooting

- **Device stays "connecting":** check the port forward from outside your network (for example from the phone on 5G) and check `public_addr`. Look at `journalctl -u tailbond` for `client authenticated` lines.
- **Tailnet unreachable from a device:** run `tailscale status` on the server, then check the **Tailscale** tab (route approved?) and `sudo firewall-cmd --info-policy=tailbond-out`.
- **No firewalld:** the hooks fall back to mqvpn's own NAT script (`/usr/lib/tailbond/mqvpn/mqvpn-server-nat.sh`) plus iptables rules for `tailscale0`.
- **Logs:** see the dashboard's **Logs** tab or `journalctl -u tailbond`. Both show the agent, mqvpn and hook output.

## Uninstall

```sh
sudo systemctl disable --now tailbond
sudo dnf remove tailbond-server
```

`dnf remove` closes UDP 443 and 41641, removes the tailnet rules for tcp 80
and 8077, and the `tailbond` zone and policies. The tunnel route stays
advertised in Tailscale: `tailscale set --advertise-routes=<your other routes,
comma separated>` keeps the others (an empty value withdraws all of them).
