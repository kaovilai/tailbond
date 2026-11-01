# tailbond on GL.iNet routers

One package, two roles:

| Router | Role | What it does |
|---|---|---|
| Slate 7 (GL-BE3600), travel | `client` | Bonds its WANs (ethernet WAN, repeater Wi-Fi, USB tethering, cellular dongle) into one tunnel to home. All LAN devices go through it. |
| Flint 3 (GL-BE9300), home | `server` | Terminates the bonded tunnels and gives clients your home LAN and your tailnet (through GL's built-in Tailscale). Use this **or** the Fedora server, not both. |

## Hardware facts (from GL.iNet firmware listings, not verified on a device)

| | Flint 3 | Slate 7 |
|---|---|---|
| SoC | Qualcomm IPQ5332 | Qualcomm IPQ5312 |
| Firmware | GL 4.x, QSDK 12.5 / OpenWrt 23.05-SNAPSHOT, kernel 5.4 | same |
| opkg arch | `aarch64_cortex-a53_neon-vfpv4` | same |
| Firewall | fw3 (iptables) or fw4 (nftables) depending on build; the hooks handle both through `/etc/config/firewall` | same |

Stock GL firmware only accepts the `aarch64_cortex-a53_neon-vfpv4` package. Plain OpenWrt 23.05/24.10 aarch64_cortex-a53 builds take `tailbond_<ver>_aarch64_cortex-a53.ipk`. Both packages contain the same static binaries.

## Build

```sh
scripts/build-mqvpn-aarch64-musl.sh      # static mqvpn → out/aarch64/mqvpn
packaging/openwrt/build-ipk.sh           # builds tailbondd (arm64) and both .ipk files → out/ipk/
```

## Install (either router)

```sh
scp -O out/ipk/tailbond_*_aarch64_cortex-a53_neon-vfpv4.ipk root@192.168.8.1:/tmp/
ssh root@192.168.8.1
opkg update && opkg install /tmp/tailbond_*_neon-vfpv4.ipk
```

Dependencies (`kmod-tun`, `ip-full`, `jshn`, `jsonfilter`) come from GL's feed during `opkg update`.

The admin panel gets a **Applications → tailbond** entry that links to the tailbond web UI at `http://<router>:8077`. Plain OpenWrt gets a LuCI entry instead.

## Travel router (Slate 7, client)

1. On your home server's dashboard, add a client named after the router (e.g. `slate7`). Copy its `tailbond://pair?...` link.
2. On the Slate 7, open `http://192.168.8.1:8077` → **Setup** → paste the link. Or set it in UCI:
   ```sh
   uci set tailbond.client.pairing_uri='tailbond://pair?d=...'
   uci set tailbond.main.role=client tailbond.main.enabled=1
   uci commit tailbond && /etc/init.d/tailbond restart
   ```
3. Bring up the WANs you want to bond the normal GL way: cable, repeater, tethering, cellular. Each `config path` in `/etc/config/tailbond` names one:

   | WAN on the Slate 7 | OpenWrt network | Typical device |
   |---|---|---|
   | Ethernet WAN port | `wan` | `eth0` |
   | Repeater (hotel Wi-Fi) | `wwan` | `phy0-sta0` / `wlan-sta` |
   | USB tethering (phone) | `tethering` | `usb0` / `eth1` |
   | Cellular USB dongle | `modem_*` or `wwan_*` | `wwan0` / `usb0` |

   Names are resolved to devices when the config renders, so the network name is usually all you need. Check yours with `ifstatus wan | jsonfilter -e @.l3_device`.

What `client-up` does:
- adds a `tailbond` network on `tb0` and a firewall zone with masquerade, plus `lan → tailbond` forwarding;
- routes LAN traffic through the tunnel with policy routing (table 77). The router's own traffic, including mqvpn's per-WAN connections to your server, stays on the WANs.

Options:
- `kill_switch` (default on): LAN traffic is blocked rather than leaked while the tunnel is down.
- `tunnel_dns` (default on): points dnsmasq at `10.77.0.1`, so MagicDNS names like `nas.<tailnet>.ts.net` resolve on every LAN device.

## Home router (Flint 3, server)

1. In the GL admin panel, turn on **Tailscale** and log in. Under Tailscale settings, allow the Flint 3 to advertise routes.
2. Configure and start tailbond:
   ```sh
   uci set tailbond.main.role=server tailbond.main.enabled=1
   uci set tailbond.server.public_addr='home.example.com'   # public IP or DDNS name
   uci commit tailbond && /etc/init.d/tailbond restart
   ```
3. Approve the advertised route `10.77.0.0/24` once in the Tailscale admin console (**Machines → Flint 3 → Edit route settings**).
4. Add clients on `http://192.168.8.1:8077` (reachable from LAN, or over Tailscale at the Flint 3's 100.x address) → **Clients → Add**. Scan the QR on Android, or paste the link into the Slate 7.

`server-up` opens UDP `443` on the `wan` zone and adds a `tailbond` zone for `tb0` with masquerade to `wan`, forwarding to `tailscale0` (tailnet) and, if `allow_lan` is set, to `lan`. If the Flint 3 isn't your edge router, forward UDP 443 to it on the upstream router. Carrier-grade NAT on the home line won't work; you need a public IPv4 or IPv6.

## Logs and troubleshooting

```sh
logread -e tailbond          # agent + hook output
/etc/init.d/tailbond status
curl -s http://127.0.0.1:8077/api/status
```

Config lives in `/etc/config/tailbond` (rendered to `/etc/tailbond/tailbond.json`). Paired credentials and server users are kept in `/etc/tailbond/state`, which survives reboots and sysupgrade (`/lib/upgrade/keep.d/tailbond`).

## Tested so far

- In emulated OpenWrt 23.05 arm64 (podman): the package installs, the config renders from UCI, `tailbondd` starts and serves the UI and `/api/status` with the WANs listed, and the package uninstalls cleanly.
- The static `mqvpn` binary passed the netns two-path smoke test under qemu.
- **Not yet tested on real Flint 3 / Slate 7 hardware**: firewall/routing hooks, the GL admin panel entry, cellular dongle naming.
