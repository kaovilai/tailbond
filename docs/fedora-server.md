# tailbond on a Fedora home server

This guide sets up the **home end** of tailbond on Fedora 43 or newer. Your phone or travel router bonds its Wi-Fi, 5G and Ethernet links to this server. Its traffic then leaves through your home internet connection, and your Tailscale tailnet is reachable through it.

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
| `tailbond.service` | systemd unit |
| firewalld `tailbond` service, zone and policies | UDP 443, `tb0`, forwarding and NAT |

Check that it runs:

```sh
systemctl status tailbond
journalctl -u tailbond -f
```

To open the dashboard from the server itself, browse to <http://127.0.0.1:8077/>. From another tailnet device use `http://<server>.<tailnet>.ts.net:8077/`, since the agent also listens on the server's Tailscale IPs. Set `"ui_password"` in `tailbond.json` to require a login.

## 2. Make the server reachable: forward UDP 443

Bonding needs each of the device's links to reach the server **directly over UDP**. It does not go through Tailscale's relays.

1. On your home router, forward **UDP port 443** to this server's LAN address. Use a DHCP reservation so that address doesn't change.
2. Tell tailbond your public address, either a static IP or a DDNS name such as `home.example.net`. Use the banner in the dashboard's **Clients** tab, or edit the config:

   ```json
   "server": { "public_addr": "home.example.net", ... }
   ```

   Then run `sudo systemctl restart tailbond`. If you leave it empty, pairing links use the server's LAN address, which only works at home.
3. firewalld: the `server-up` hook opens the `tailbond` service (UDP 443) in your default zone. You don't need to run any `firewall-cmd` commands yourself.

If your ISP puts you behind CGNAT (no public IPv4), use a public IPv6 address for `public_addr` (for example `[2001:db8::10]:443`) and allow UDP 443 in the router's IPv6 firewall.

## 3. Tailscale (optional, recommended)

tailbond works without Tailscale. With Tailscale running on the server (`sudo tailscale up`), you also get:

- **Tailnet access for bonded devices.** Traffic from the tunnel (`10.77.0.0/24`) to `100.64.0.0/10` and `fd7a:115c:a1e0::/48` leaves through `tailscale0` with masquerade. To the tailnet it looks like it comes from the server, so bonded devices don't show up as extra machines.
- **MagicDNS for bonded devices.** The agent answers DNS on `10.77.0.1`: `*.ts.net` and your tailnet's MagicDNS names go to `100.100.100.100`, everything else to the server's normal resolver.
- **Reaching bonded devices from the tailnet.** The `server-up` hook runs `tailscale set --advertise-routes=10.77.0.0/24` and keeps any routes you already advertise. **Approve the route once:** open the [admin console](https://login.tailscale.com/admin/machines), find this machine, choose **Edit route settings**, and enable `10.77.0.0/24`. The dashboard's **Tailscale** tab shows whether the route is advertised and approved.
- **A real TLS certificate.** If HTTPS certificates are enabled for your tailnet (admin console → DNS → HTTPS Certificates), the agent runs `tailscale cert` for `<server>.<tailnet>.ts.net`. mqvpn then uses that certificate, and devices check it by name even though they connect to your public address. Without it, the agent creates a self-signed certificate, and its SHA-256 fingerprint goes into every pairing link.

If you switch between the self-signed and the Tailscale certificate (by turning tailnet HTTPS on or off), pair your devices again: the old links pin the old certificate.

## 4. Add a device and pair it with a QR code

1. In the dashboard, open **Clients**, type a name (for example `pixel` or `slate7`), and press **Add**.
2. A QR code and a `tailbond://pair?d=…` link appear. Each device gets its own key, so you can revoke one without touching the others.
   - **Phone:** scan the QR code with the tailbond app.
   - **Travel router (tailbondd in client role):** open its dashboard, go to **Setup**, and paste the link or scan the QR code. Then press **Pair & connect**.
3. The device appears under **Connections** with one card per link (path), showing live rate, latency and loss.

To show the QR code again later, use the QR button next to the device. **Revoke** (the trash button) deletes the key and disconnects the device.

The same steps from a shell on the server:

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

The tunnel route stays advertised in Tailscale. Remove it with `tailscale set --advertise-routes=` (keeping your other routes).
