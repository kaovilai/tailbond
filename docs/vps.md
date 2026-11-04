# tailbond on a VPS (backup server)

A small VPS makes a good **second server**: when your home connection is
down, your travel router or phone switches to it and keeps going
([multiple servers and failover](architecture.md#multiple-servers-and-failover)).
It also works as the only server, if you'd rather exit from a datacenter
than from home.

- **Fedora**: use the RPM, see [fedora-server.md](fedora-server.md).
- **Debian 12, Ubuntu 22.04 / 24.04, AlmaLinux / RHEL / Rocky 9 and 10**: the
  generic bundle below (static binaries, any glibc, x86_64 or aarch64).
- **Ansible**: the role in `packaging/ansible/roles/tailbond` does the same
  ([below](#ansible)).

## 1. Install

```sh
tar xzf tailbond-linux-x86_64.tar.gz
cd tailbond-linux-x86_64
sudo ./install.sh --dry-run     # shows what it would do, changes nothing
sudo ./install.sh
```

The bundle comes from `packaging/linux/build-bundle.sh` (CI artifact
`linux-bundle`). It installs:

| Path | What |
|---|---|
| `/usr/local/bin/mqvpn`, `/usr/local/bin/tailbondd` | the data plane and the agent, both static |
| `/usr/local/lib/tailbond/hooks/` | `server-up`, `server-down`: firewall, NAT, Tailscale route |
| `/etc/tailbond/tailbond.json` | configuration, mode 0600 (kept on re-runs) |
| `/var/lib/tailbond/` | generated `server.conf`, certificates |
| `/etc/systemd/system/tailbond.service` | one unit: tailbondd runs mqvpn as its child |
| `/etc/sysctl.d/90-tailbond.conf` | IP forwarding |
| `/etc/firewalld/{services,zones,policies}/tailbond*.xml` | only when firewalld is installed |
| `/usr/local/share/doc/tailbond/` | this file, mqvpn's license |

`--prefix /usr` puts the binaries in `/usr/bin` and the hooks in
`/usr/lib/tailbond/hooks` instead. Running it again upgrades the binaries
and keeps the registered devices, the subnet, the port, the public address
and the Tailscale choice; options you pass are applied on top (it needs `jq`
or `python3` to merge them into an existing config). `sudo ./uninstall.sh`
removes everything but `/etc/tailbond` and `/var/lib/tailbond`
(`--purge` removes those too).

The installer prints every decision. Override any of them:

| Option | Default |
|---|---|
| `--port N` | 443 if free, else 8443, 4443, 10443, 50443 |
| `--public-addr HOST` | the source address of the default route |
| `--subnet CIDR`, `--subnet6 CIDR` | a random free /24 in 10.78–10.99, a random `fd77:xxxx::/112` |
| `--no-tailscale` | Tailscale is used when `tailscale status` works |
| `--landing-port N` | 80, or 8078 when something listens on `*:80`; `-1` = off |
| `--memory-high`, `--memory-max` | `96M`, `160M` |
| `--update-policy P` | `notify`: [Updates](#updates) |
| `--update [--version vX.Y.Z]` | update the binaries only (below) |
| `--no-firewall`, `--no-start`, `--mqvpn`, `--tailbondd`, `--bundle-dir`, `--config` | |

### A Linux laptop instead (client role)

The same bundle installs the **client** on a laptop that bonds its own
Wi-Fi, Ethernet and a tethered phone to your home server (the home
server's "Get tailbond" page, **Linux** tab, has this line to copy):

```sh
curl -fsSL https://github.com/kaovilai/tailbond/releases/latest/download/tailbond-linux-$(uname -m).tar.gz | tar -xz
sudo ./tailbond-linux-$(uname -m)/install.sh --client     # --pair 'tailbond://pair?d=…' to pair at once
```

It installs the binaries, a client config (`/etc/tailbond/tailbond.json`,
dashboard on `127.0.0.1:8077` only) and a plain systemd unit; no firewall,
NAT, sysctl or phone page. mqvpn sets the routes and DNS itself. Then open
`http://localhost:8077` → **Setup** → **Ask your home server** (Tailscale
on). `install.sh --update` and `--uninstall` work the same; the server
installer refuses to run over a client's config and the other way round.

## 2. Ports on a box that already runs a web server

Nothing that's already listening is touched; the installer looks at
`ss -lnu` / `ss -lnt` first.

- **Tunnel, UDP.** Caddy, nginx-quic and other HTTP/3 servers hold udp/443,
  so the tunnel takes **udp/8443** (TCP 8443 being in use doesn't matter).
  That's the port pairing links carry. Allow it in your provider's firewall
  or security group too, if there is one.
- **Phone page.** The agent serves it on port 80 of the Tailscale address.
  A web server on `*:80` makes that impossible, so the page moves to
  **8078** (`landing_port`/`landing_public_port`) and phones open
  `http://vps:8078/`. To keep plain `vps/`, let the web server pass the
  tailnet name through (the installer prints this with your names; it never
  edits the web server's config):

  ```
  http://vps, http://vps.tail1234.ts.net {
      reverse_proxy 100.x.y.z:8078
  }
  ```

  then set `"landing_public_port": 80` and restart tailbond.
- **Dashboard, tcp/8077** on 127.0.0.1 and the Tailscale address (routers
  find servers there). It must be free; the installer stops if it isn't.
- **DNS for bonded devices** on the tunnel address only (`10.x.y.1:53`), so
  systemd-resolved, podman's aardvark-dns (`10.89.0.1:53`) and the like don't
  clash. Only a DNS server on `*:53` does; then DNS for clients is turned off
  (`server.disable_dns`) and they keep their own.

## 3. Tunnel subnet

Every server a client uses needs its own subnet. Home servers default to
`10.77.0.0/24`; the installer picks a random free `/24` between
`10.78.0.0` and `10.99.255.0` that overlaps no route (all tables, so
routes accepted from your tailnet count) and no address on the VPS, e.g.
`10.89.0.0/24` of a podman network. Re-runs keep it.

## 4. With or without Tailscale

**With Tailscale** (`tailscale up` before installing): mqvpn uses a
`tailscale cert` certificate when HTTPS is on for your tailnet, bonded
devices reach your tailnet through the VPS, and the `server-up` hook
advertises the tunnel subnet (approve it in the admin console only if tailnet
machines should reach bonded devices). Pairing works over Tailscale, below.

**Without Tailscale** (`--no-tailscale`, or it isn't installed): a
self-signed certificate pinned in each pairing link, internet exit only, and
`public_addr` is required (detected; a private address means the VPS is
behind NAT, as on AWS, GCP or Oracle, so pass `--public-addr`). Pair with a
link or QR code from the VPS:

```sh
curl -s -X POST -H 'Content-Type: application/json' -d '{"name":"slate7"}' http://127.0.0.1:8077/api/clients
```

The reply has `uri` (`tailbond://pair?d=…`, paste it into the router's
panel or the app) and `qr_svg` (an SVG QR code; save it and open it in a
browser, or open the dashboard through an SSH tunnel:
`ssh -L 8077:127.0.0.1:8077 vps`, then <http://127.0.0.1:8077/>).

## 5. Pair it as a backup server

- **Travel router**: its panel → **Add another server**. Over Tailscale,
  pick the VPS from the list; the request shows up at the top of the VPS's
  dashboard (`http://vps:8077/` from your tailnet), approve it. Or paste a
  link from section 4.
- **Android app**: the server list → **Add another server**, same choices.

The new server goes to the end of the list; the first one (home) stays
primary and the client falls back to it when it's back. If the client
says the subnet is in use, the two servers share a subnet: on the VPS, re-run
`sudo ./install.sh --subnet 10.83.1.0/24` (any free /24) and pair again.

## 6. Firewall

The installer and the hooks use what's there, in this order:

| Found | What happens |
|---|---|
| firewalld running | like the Fedora RPM: the `tailbond` service (the tunnel port) in the zone of the default-route interface, `tb0` in the `tailbond` zone, policy `tailbond-out` masquerades bonded traffic to the internet and `tailscale0`, tcp 8077 and the phone page only from tailnet addresses |
| ufw active | persistent rules (comment `tailbond`): the tunnel port, everything in on `tb0`, 8077 and the phone page in on `tailscale0`; NAT and forwarding from the hook's nftables table |
| nftables | the hook loads its own `table inet tailbond` (forward `tb0`, masquerade the tunnel subnets) at every start and deletes it at stop. Another table with an input `policy drop` still drops: allow the port there |
| iptables only | rules with comment `tailbond` |
| none | a warning: bonded devices then reach only the VPS |

The hooks check again at every start, so a firewalld that's sometimes
stopped is fine. When the iptables `FORWARD` policy is `DROP` (Docker, ufw),
the hook also adds `ACCEPT` rules for `tb0` there.

firewalld's first reload (to load the zone) can drop container port
forwards on old podman/netavark; netavark 1.9+ restores them itself,
otherwise run `podman network reload --all`.

## 7. SELinux (Alma, RHEL, Rocky, Fedora)

Nothing to configure. The binaries in `/usr/local/bin` get `bin_t` (the
installer runs `restorecon`; a binary moved in from your home directory
would keep `user_home_t` and systemd couldn't run it), and systemd starts
them as `unconfined_service_t`, like any locally installed service, so the
TUN device, the UDP port and the hooks need no policy. The unit doesn't set
`NoNewPrivileges=`, which would stop the hooks' `iptables`/`nft`/`firewall-cmd`
from entering their own SELinux domains.

If the service fails while `setenforce 0` makes it work:

```sh
sudo ausearch -m avc -ts recent | grep -E 'tailbond|mqvpn'
ls -Z /usr/local/bin/mqvpn /usr/local/bin/tailbondd    # want bin_t
sudo restorecon -RFv /usr/local/bin /usr/local/lib/tailbond /etc/tailbond /var/lib/tailbond /etc/systemd/system/tailbond.service
```

Then `sudo systemctl restart tailbond`. A denial that remains after
`restorecon` is a bug; please report it with the `ausearch` output.

## 8. Memory and CPU

Measured with `scripts/measure-server-mem.sh` (network namespaces on a
desktop Ryzen: tailbondd in server role running the static mqvpn from the
bundle; 2 clients × 2 paths, speed mode; each client sends 300 MB and
receives 300 MB at the same time). Run it yourself with
`sudo scripts/measure-server-mem.sh` (`NETEM_A`/`NETEM_B` shape the paths):

| | mqvpn | tailbondd | together |
|---|---|---|---|
| idle, no clients | 32 MB | 13 MB | 45 MB |
| 2 clients connected, idle | 33 MB | 15 MB | 48 MB |
| 2 clients busy, paths shaped like fiber + LTE (~170 Mbit/s each) | 48 MB | 17 MB | 65 MB |
| 2 clients busy, unshaped (1 Gbit/s total) | 98 MB | 16 MB | 114 MB |

The static (musl) mqvpn starts at ~32 MB; a glibc build of the same code
idles at ~8 MB and peaked at 26 MB in the shaped run. Add tailscaled
(30–50 MB) if it isn't running already.

| Sizing | |
|---|---|
| RAM for tailbond | ~50 MB idle, ~65 MB with two busy clients; the unit allows `MemoryHigh=96M` (throttled above), `MemoryMax=160M` (killed and restarted above) |
| per extra client | ~1–2 MB connected, ~8 MB while it moves data at full speed |
| smallest VPS | 512 MB RAM if it only runs tailbond and Tailscale; 1 GB with other services |
| CPU | 3–14% of one desktop core per 100 Mbit/s through the server (2.7–11 CPU seconds per GB; lossy, shaped paths cost the most; on a real home link we've seen ~1 s per GB); plan for 0.3 vCPU per 100 Mbit/s on a small VPS |

The unit's other limits: `OOMScoreAdjust=-100` (the kernel's OOM killer
picks a bigger process, such as a 500 MB n8n, first; tailbond isn't exempt,
and `MemoryMax` caps it anyway), `GOMEMLIMIT=40MiB` for tailbondd's garbage
collector, default `CPUWeight`. Change them with `--memory-high` /
`--memory-max` or `sudo systemctl edit tailbond`.

**Transfer.** Everything a bonded device does crosses the VPS twice: a
download comes in from the internet and goes out to the device through the
tunnel; an upload comes in through the tunnel and goes out to the internet.
So 10 GB of downloads on the road is about 10 GB in and 10 GB out on the
VPS, plus a few percent of tunnel overhead, and more in the streaming and
redundant modes (UDP is sent on several paths). Check how your provider
counts the monthly allowance (outbound only, or both directions), and set
data caps on the client's paths if it matters.

## Ansible

`packaging/ansible/roles/tailbond` installs the same bundle with the
installer's decisions written down as variables (port, subnet, phone page
port, memory limits, Tailscale on/off, firewall). Before changing anything
it runs the bundle's `install.sh --dry-run` with those values, which fails
if the port is taken by something else or the subnet overlaps a route. It
merges only its own keys into `tailbond.json` (registered devices stay) and
doesn't restart the service when nothing changed. It doesn't install
Tailscale; bring that up first. Copy the role into your playbook repository
and add it to a play:

```sh
cp -r packaging/ansible/roles/tailbond <your repo>/ansible/roles/
```

```yaml
# site.yml
- hosts: vps
  become: true
  roles:
    - role: tailbond
      vars:
        tailbond_bundle: files/tailbond-linux-x86_64.tar.gz
        tailbond_port: 8443          # Caddy has udp/443
        tailbond_subnet: 10.83.142.0/24
```

`tailbond_subnet` has no default (pick a free /24, e.g. from the
installer's dry run); `tailbond_landing_port` defaults to 8078 for boxes
with a web server on `*:80`. See `defaults/main.yml` for every variable.
Collections: `ansible.posix` (firewalld, sysctl), `community.general` (only
for `tailbond_firewall: ufw`).

## Updates

The dashboard checks GitHub's latest release about once a day and shows it.
**Update now** (or policy **Automatic**) runs the new bundle's
`install.sh --update`; by hand:

```sh
sudo /usr/local/lib/tailbond/install.sh --update                   # the latest release
sudo /usr/local/lib/tailbond/install.sh --update --version v0.2.0  # a given one (also to go back)
```

It downloads `tailbond-linux-<arch>.tar.gz` and `SHA256SUMS` from the GitHub
release, checks the SHA-256, replaces only the binaries and hooks (keeping
`tailbondd.prev` / `mqvpn.prev`), restarts tailbond and puts the old
binaries back if it isn't up on the new version within 60 seconds. Run from
an unpacked bundle, `sudo ./install.sh --update` installs that bundle.
`--update-policy notify|auto|off` sets the dashboard's policy. More:
[updates.md](updates.md).

**Ansible**: `tailbond_version` is the version; bump it and run the playbook
(the role downloads that release's bundle and checks it against
`SHA256SUMS`). `tailbond_update_policy` (default `notify`) is the dashboard's
policy; with the role in charge keep it `notify` or `off`, since the next
playbook run puts `tailbond_version` back.

```yaml
    - role: tailbond
      vars:
        tailbond_version: "0.2.0"
        tailbond_update_policy: notify
```

## Troubleshooting

- `journalctl -u tailbond -n 50`: agent, mqvpn and hook output.
- `sudo ./install.sh --dry-run` on the VPS shows what the installer sees
  (listeners, routes, firewall) without changing anything.
- Devices stay "connecting": the provider's firewall doesn't let the UDP
  port in, or `public_addr` is wrong (`curl -s http://127.0.0.1:8077/api/settings`).
- Memory: `systemctl status tailbond` shows the unit's current memory and its
  peak.
