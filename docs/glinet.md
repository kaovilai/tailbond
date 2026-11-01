# tailbond on GL.iNet routers

One package, two roles:

| Router | Role | What it does |
|---|---|---|
| Slate 7 (GL-BE3600), travel | `client` | Bonds its WANs (ethernet, repeater Wi-Fi, USB tethering, cellular) into one tunnel to home. All LAN devices go through it. |
| Flint 3 (GL-BE9300), home | `server` | Terminates the bonded tunnels and gives clients your home LAN and your tailnet (through GL's built-in Tailscale). Use this **or** the Fedora server, not both. |

Setup tips for best speed (direct Tailscale, ports, the route):
[setup-tips.md](setup-tips.md).

## Hardware facts (from GL.iNet firmware listings, not verified on a device)

| | Flint 3 | Slate 7 |
|---|---|---|
| SoC | Qualcomm IPQ5332 | Qualcomm IPQ5312 |
| Firmware | GL 4.x, QSDK 12.5 / OpenWrt 23.05-SNAPSHOT, kernel 5.4 | same |
| opkg arch | `aarch64_cortex-a53_neon-vfpv4` | same |
| Firewall | fw3 (iptables) or fw4 (nftables) depending on build; the hooks handle both through `/etc/config/firewall` | same |

Stock GL firmware only accepts the `aarch64_cortex-a53_neon-vfpv4` package.
Plain OpenWrt 23.05/24.10 aarch64_cortex-a53 builds take
`tailbond_<ver>_aarch64_cortex-a53.ipk`. Both contain the same static
binaries. The ipk is a gzipped ustar tar (GL's opkg reportedly crashes on
ar-format ipks: [Wattline notes][wl]).

## Build

```sh
scripts/build-mqvpn-aarch64-musl.sh      # static mqvpn → out/aarch64/mqvpn
packaging/openwrt/build-ipk.sh           # tailbondd (arm64) + both .ipk files → out/ipk/
scripts/test-openwrt-ipk.sh              # emulated OpenWrt 23.05 arm64 install test (podman)
```

## Install without SSH

GL's **APPLICATIONS → Plug-ins** page installs from GL's repository only; it
has no upload ([GL docs: Plug-ins][gl-plugins]). A local package goes
through LuCI, which GL 4.x ships ([GL docs: Advanced Settings][gl-adv]):

1. Get the package (the file ending in `_neon-vfpv4.ipk`). A home server
   offers it: the Fedora page's **Devices → Add a GL.iNet travel router**
   (`/tailbond-glinet.ipk`, from `/usr/share/tailbond/tailbond-glinet.ipk`),
   or a Flint 3's panel, which rebuilds the package it runs (below).
2. On the travel router: **SYSTEM → Advanced Settings → Go To LuCI** (same
   admin password), then **System → Software → Upload Package…**, pick the
   file, **Install**. Leave "Allow overwriting" unticked.
3. In GL's admin, open **VPN → tailbond**. On a Slate 7 tap **Ask to connect**
   (below). A Flint 3 (`GL-BE9300` in `/tmp/sysinfo/model`) starts as the home
   server; the panel switches roles either way.

The package installs **enabled**: the agent starts right away (the panel
needs it) and the tunnel connects once paired. The LAN keeps its internet
until then (the kill switch arms only when paired and on). One command, if
you do have SSH: `opkg install /tmp/tailbond_*_neon-vfpv4.ipk`.

A firmware upgrade that doesn't keep packages removes tailbond (Speedify
documents the same for its GL package); `/etc/tailbond/` (keys, pairing,
users) is kept (`/lib/upgrade/keep.d/tailbond`), so reinstalling picks up
where it left off.

A Flint 3 serves the package without having the file: `GET
/tailbond-glinet.ipk` rebuilds it from opkg's records
(`/usr/lib/opkg/info/tailbond.{control,list,postinst,…}`), using pristine
copies of the files that change after install
(`/usr/share/tailbond/pristine/…`) and never including `/etc/tailbond/`.

## Admin panel (VPN → tailbond)

A native GL 4.x "oui" view, under the **VPN** section like GL's own VPN
client and Speedify's GL package (both `"parent": "vpn"` in
`/usr/share/oui/menu.d/`; Speedify's view is an iframe to its own web app,
ours renders natively). It uses GL's login; there is no second password.

| Piece | File | How it works |
|---|---|---|
| Menu | `/usr/share/oui/menu.d/tailbond.json` | `parent: vpn`, `disables: ["portal"]` (GL's first-boot setup redirect, as GL's VPN client and Speedify declare) |
| View | `/www/views/gl-sdk4-ui-tailbond.common.js.gz` | GL's app fetches it and `eval()`s it into a Vue 2 component (runtime-only Vue: render functions). Self-contained styles |
| Backend | `/usr/lib/oui-httpd/rpc/tailbond` (Lua) | The view calls `window.$rpcRequest("call", ["sid","tailbond",method,{…}])` → `POST /rpc`. oui-httpd checks GL's session (admin = `aclgroup root`), then calls the module, which talks HTTP to tailbondd on 127.0.0.1:8077 (ngx cosocket; busybox `nc` outside nginx). Methods: `status`, `api{method,path,body}` (only `/api/…` paths, no login), `info`, `service{action}`, `set_role{role}` |
| Validator | `/usr/share/gl-validator.d/tailbond.lua` | oui-httpd rejects string args outside `^[%w%.%s%-_:#/]-$` unless a validator allows them; `api` carries JSON bodies, so it's `true` |
| Fallback | `/etc/nginx/gl-conf.d/tailbond.conf` | Same-origin `/tailbond-api/` proxy (and `/tailbond-glinet.ipk`), checking the `Admin-Token` cookie with `ubus call gl-session session`. The view switches to it only if `/rpc` refuses the object |
| Agent | `tailbondd` | Client role listens on 127.0.0.1:8077 only (`lan_dashboard=1` adds the LAN page as an advanced/debug view); server role on 127.0.0.1 + its Tailscale IP |

Sources for GL's internals: GL 4.9.1 firmware files as published in
[wickedyoda/gl.inet_firmware][dump] (BE14000 image), the oui-httpd Lua
copied in [MudiModem's reference/][mm-ref] (`oui-rpc.lua`, `oui-lib-rpc.lua`,
`gl.conf`) and its [notes][mm], and the [Wattline GL panel notes][wl].

**Not verified on a Slate 7 / Flint 3** (those firmwares weren't available):
`$rpcRequest`'s name and error shape, the validator path, GL 4.7+'s
"method-level validation at the /rpc layer" (could refuse third-party
objects: hence the nginx fallback), the panel routes the view links to
(`#/tailscaleview`, `#/internet`, `#/ddns`, `#/acceleration`), and nginx's
`gl-conf.d` include. If `/rpc` refuses the object *and* the nginx include is
missing, the advanced page on :8077 (`uci set tailbond.main.lan_dashboard=1`)
still works.

## Travel router (Slate 7, client)

### Pairing over Tailscale

Both GL routers have Tailscale built in (APPLICATIONS → Tailscale). With the
Slate 7 and the home server on the same tailnet, the panel lists online
peers (`tailscale status --json`), probes their `:8077/api/hello`, and
shows **Found your home server: homebox → Ask to connect**. The home server
lists the request with the router's Tailscale identity (`tailscale whois`);
someone taps **Approve** (Flint 3 panel, or `http://homebox:8077` from any
device on the tailnet) and the Slate 7 connects by itself. Contract:
[architecture.md](architecture.md#pairing-over-tailscale). Without Tailscale
on the router, paste a pairing link (**Other ways to connect**).

### WANs: detected, not configured

Every network in the firewall's `wan` zone (`ubus call network.interface
dump` + `uci show firewall`; GL's names `wan`, `secondwan`, `usbwan`, `wwan` =
repeater, `tethering`, `modem_*` = cellular, from GL's `kmwan.sh`) is used
without configuration: Ethernet and Wi-Fi as main links, tethering and
cellular as secondary (so **Save mobile data** keeps them on standby). The
panel's per-WAN choice is **Always / Backup only / Off** plus a monthly
data limit; settings are keyed by network name, so a different phone on
`tethering` keeps the same choice. `/etc/hotplug.d/iface/90-tailbond` sends
SIGHUP on ifup/ifdown so the agent re-checks immediately (it also polls
every 5 s). IPv6 twins (`wan6`, …) and a modem's `_4` child are folded in.

What `client-up` does:
- adds a `tailbond` network on `tb0` and a firewall zone with masquerade, plus `lan → tailbond` forwarding;
- routes LAN traffic through the tunnel with policy routing (table 77, rules
  at priority 1076/1077). The router's own traffic, including mqvpn's
  per-WAN connections and GL's own checks, stays on the WANs.

Options (`/etc/config/tailbond`, `client` section):
- `kill_switch` (default on): LAN traffic is blocked rather than leaked while the tunnel drops. Turning tailbond off, or unpairing, always releases the LAN.
- `tunnel_dns` (default on): dnsmasq forwards to `10.77.0.1`, so MagicDNS names like `nas.<tailnet>.ts.net` resolve on every LAN device.
- `captive_portal_detect` (default on), `captive_probe_url` (list), `portal_table` (78): below.

GL's multi-WAN (kmwan, a reworked mwan3: [GL 4.5.0 notes][kmwan]) can stay
on. Speedify's GL package turns it off; tailbond doesn't need to, because its
LAN rules (priority 1074–1077) come before mwan3-style policy rules (1001+
per-interface, 2001+ by fwmark, assumed the same for kmwan, **unverified**)
only for LAN input, and mqvpn binds each path to its device.

## Captive portals and GL's Login Mode

Hotel and airport networks often show a sign-in page first. tailbond keeps
such a WAN out of bonding (the other WANs keep the tunnel up; it doesn't
count as a failed path or for failover), offers a sign-in, and adds the WAN
back by itself once it's open. Speedify does the same from its own app
([Speedify: captive portals][sfy-portal]); on GL routers the repeater
already has its own portal handling, which tailbond follows instead of
competing with.

### GL's own handling (GL 4.6+)

From [GL's docs][gl-portal-faq] and [repeater docs][gl-repeater], and GL
4.9.1's files ([dump][dump]):

- **Detection**: the `gl-repeater` daemon (`/etc/init.d/repeater` →
  `/usr/sbin/gl-repeater`, logic in `/usr/local/lib/lua/5.4/gl/repeater-portal.lua`)
  probes `http://captive.apple.com/hotspot-detect.html` and
  `http://www.msftconnecttest.com/connecttest.txt` bound to the `wwan` device
  with wwan's DNS servers, on interface-up when `auto_portal` is set; the
  portal URL comes from `Location`, `meta refresh` or JS.
- **State**: the native ubus object `repeater`; `ubus call repeater status`
  returns a flat object with `portal`, `portal_detecting`, `portal_url`,
  `bare_mode`, `auto_portal`, `state` (GL's own UI reads exactly these:
  `needPortal()` in `gl-sdk4-ui-internet.common.js`). No UCI flag or state
  file marks the mode. `repeater.status` is a websocket notification
  (`gl-session notify`), not a ubus event, so it's polled.
- **Prompt**: GL's Repeater page shows "Portal Authentication Required" with
  **Login Mode for Public Hotspots** / **Go to Login**; with
  **Auto-Enable Login Mode** it enters the mode by itself and leaves it after
  a successful login.
- **Login Mode ("bare mode")** suspends AdGuard Home, custom DNS (rebind
  protection off), ZeroTier, **GL's Tailscale**, Tor, parental control, kmwan
  probing, and bypasses GL's VPN client with
  `iptables -t mangle -I PREROUTING -i br-+ … -j MARK --set-mark 0x8000/0xf000`
  (GL's `nonevpn_mark`, `/lib/functions/vpn_func/route_policy_func.sh`).
  That mark only matters to GL's own route_policy VPN; there is no plug-in
  hook, so a third-party tunnel such as tailbond (or Speedify) is **not**
  suspended and must step aside itself.
- **Camouflage / MAC clone** (Repeater → Modify → MAC Mode): the router's
  WAN MAC; tailbond doesn't touch MACs, so both keep working.

### What tailbond does

| Situation | Source of truth | tailbond |
|---|---|---|
| Repeater (`wwan`), GL's portal handling present and on (`ubus call repeater status` has `portal`; on = `auto_portal`, or a portal/check/Login Mode seen) | **GL** | No probe of its own on wwan (GL already probes it; no double probing). `portal: true` → wwan out of bonding, `captive_source: glinet`; the panel says "Finish sign-in in GL's Repeater page" with a link, no second flow. `bare_mode: true` → tailbond's scoped bypass (below) follows it, also with tailbond's detection turned off. Login Mode over / `portal: false` → bypass reverted, wwan back in bonding, no action in either UI |
| Ethernet, tethering, cellular, or no GL repeater | **tailbond's probe** | `GET http://connectivitycheck.gstatic.com/generate_204`, then `http://detectportal.firefox.com/success.txt` (configurable), bound to the WAN's device (`SO_BINDTODEVICE`) and resolved through the WAN's own DHCP DNS (also bound to it). 204/expected body = open; redirect, other 200, 511 = portal (URL from `Location`, meta refresh or JS); no answer = offline (not held back). On link up, hotplug, a failed mqvpn path, every 60 s while captive, every 5 s while signing in, every 10 min otherwise. The panel's **Open sign-in page** starts sign-in mode and opens the portal URL (or `http://neverssl.com/`) |

**Sign-in mode** (hooks `portal-open` / `portal-close`, idempotent): LAN
traffic goes straight out that WAN (table 78, rule `iif br-lan lookup 78` at
priority 1074, ahead of the tunnel and kill-switch rules), dnsmasq uses the
WAN's DNS servers bound to it (`server=IP@dev`) instead of the tunnel DNS,
and DNS rebind protection is off; all restored on close. It ends when the
WAN checks open (or GL leaves Login Mode), after 10 minutes, or with **Stop
sign-in mode**. The portal sees the router's MAC (NAT), so signing in on a
laptop authorises the router. The rules match LAN input only, so the
router's own traffic, including GL's portal checks (bound to wwan with a
fwmark), never goes through `tb0` or table 78; the e2e test checks a
device-bound request during sign-in.

Assumptions about GL (adapter: `agent/internal/glrepeater/glrepeater.go`,
one file, our probe as the fallback when it reads nothing): the field names
above, `wwan` as the repeater network, and that Login Mode exits by itself
after a pass. Read from GL 4.9.1 for the BE14000 (compiled Lua read as
strings, plus GL's plain-text web UI), not from a Slate 7.

## Network Acceleration

GL's **NETWORK → Network Acceleration** ([GL docs][gl-accel]) is one of:
MediaTek hardware NAT (`mtkhnat.global.enable`), Qualcomm ECM/SFE
(`/etc/config/ecm` `acceleration_engine`, the Flint 3 / Slate 7 SoCs), or
generic `firewall.@defaults[0].flow_offloading` (from GL 4.9.1's
`/usr/lib/oui-httpd/rpc/network`). Speedify turns `mtkhnat` off at start
because its transparent proxy (TPROXY) conflicts with it
([Speedify: GL connection issues][sfy-gl]). tailbond has no transparent
proxy: LAN traffic goes to `tb0`, a TUN device that offload engines can't
fast-path (every packet must reach mqvpn in user space), and mqvpn's own UDP
sockets are local traffic. So tailbond leaves acceleration alone. The only
effect: connections that were already offloaded before tailbond connected
can keep going around the tunnel (and the kill switch) until they end. The
panel notes when acceleration is on and links the page; turn it off if that
matters to you. **Unverified on hardware**: ECM's behaviour with policy
routing on IPQ53xx.

## Home router (Flint 3, server)

1. In GL's admin, turn on **Tailscale** and log in (APPLICATIONS → Tailscale).
2. **VPN → tailbond** shows the remaining setup steps: approve the
   `10.77.0.0/24` route once in the Tailscale admin console, and give travel
   routers your home's address (GL's **Dynamic DNS** name, or a public IP).
3. Approve travel routers and phones as they ask; **Setup health** checks
   direct vs relayed Tailscale connections, carrier-grade NAT, the route and
   whether UDP 443 was reached from outside ([setup tips](setup-tips.md)).

`server-up` opens UDP 443 on the `wan` zone and TCP 8077 from the tailnet
zone (discovery and pair requests), adds a `tailbond` zone for `tb0` with
masquerade to `wan`, forwarding to `tailscale0` (tailnet) and, if
`allow_lan` is set, to `lan`. If the Flint 3 isn't your edge router, forward
UDP 443 to it on the upstream router. Who can approve: anyone logged in to
GL's admin, and anyone on the tailnet who opens `http://homebox:8077` unless
the panel sets a password for that page.

## Logs and troubleshooting

```sh
logread -e tailbond          # agent + hook output
/etc/init.d/tailbond status
curl -s http://127.0.0.1:8077/api/status
lua -e 'print(require("cjson").encode(dofile("/usr/lib/oui-httpd/rpc/tailbond").status()))'
ubus call repeater status    # GL's portal state, as tailbond reads it
```

Config: `/etc/config/tailbond`, rendered to `/etc/tailbond/tailbond.json`.

## Tested so far

- Emulated OpenWrt 23.05 arm64 (`scripts/test-openwrt-ipk.sh`): install,
  enabled by default, role by model, loopback listen, no sample WANs, panel
  files present, the `/rpc` module (real `lua` + `lua-cjson` on the image)
  answers `status`/`api`/`info` against the running agent and refuses
  `/api/login` and path escapes, the agent rebuilds its own package, clean
  uninstall.
- `scripts/e2e-netns.sh`: pairing over Tailscale with fake `tailscale`
  CLIs, captive portals (own probe, mocked GL repeater with Login Mode, the
  real portal hooks against a stub uci).
- **Not tested on real Flint 3 / Slate 7 hardware**: GL's `/rpc` and nginx
  behaviour, firewall/routing hooks, GL's repeater status fields, Network
  Acceleration interaction, cellular naming.

[dump]: https://github.com/wickedyoda/gl.inet_firmware/tree/main/firmware/be14000/v4.9.1_release2-1053-0722-1784729880/filesystem
[mm]: https://github.com/kevinherzig/MudiModem/blob/main/CLAUDE.md
[mm-ref]: https://github.com/kevinherzig/MudiModem/tree/main/reference
[wl]: https://github.com/keithah/openwrt-wattline/blob/main/docs/gl-panel-integration.md
[gl-plugins]: https://docs.gl-inet.com/router/en/4/interface_guide/plugins/
[gl-adv]: https://docs.gl-inet.com/router/en/4/interface_guide/advanced_settings/
[gl-portal-faq]: https://docs.gl-inet.com/router/en/4/faq/connect_to_a_hotspot_with_captive_portal/
[gl-repeater]: https://docs.gl-inet.com/router/en/4/interface_guide/internet_repeater/
[gl-accel]: https://docs.gl-inet.com/router/en/4/interface_guide/network_acceleration/
[kmwan]: https://dl.gl-inet.com/release/router/release/mt3000/4.5.0
[sfy-portal]: https://support.speedify.com/article/877-captive-portal
[sfy-gl]: https://support.speedify.com/article/918-openwrt
