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

Stock GL firmware only accepts the `aarch64_cortex-a53_neon-vfpv4` package;
plain OpenWrt 23.05/24.10 aarch64_cortex-a53 builds take the
`aarch64_cortex-a53` one. Both contain the same static binaries. The ipk is
a gzipped ustar tar (GL's opkg reportedly crashes on ar-format ipks:
[Wattline notes][wl]). OpenWrt 25.12 and newer replaced opkg with apk and
installs a `.apk` package instead ([below](#openwrt-2512-and-apk)).

## Install without SSH

**Download** (latest release; [all versions](https://github.com/kaovilai/tailbond/releases)):

- GL.iNet stock firmware (Flint 3, Slate 7):
  [tailbond-glinet-aarch64_cortex-a53_neon-vfpv4.ipk](https://github.com/kaovilai/tailbond/releases/latest/download/tailbond-glinet-aarch64_cortex-a53_neon-vfpv4.ipk)
- Plain OpenWrt 23.05/24.10 aarch64:
  [tailbond-openwrt-aarch64_cortex-a53.ipk](https://github.com/kaovilai/tailbond/releases/latest/download/tailbond-openwrt-aarch64_cortex-a53.ipk)
- Plain OpenWrt 25.12+ aarch64 (apk):
  [tailbond-openwrt-aarch64_cortex-a53.apk](https://github.com/kaovilai/tailbond/releases/latest/download/tailbond-openwrt-aarch64_cortex-a53.apk)

To check a download against the release's `SHA256SUMS` (on a computer):

```sh
curl -fLO https://github.com/kaovilai/tailbond/releases/latest/download/tailbond-glinet-aarch64_cortex-a53_neon-vfpv4.ipk
curl -fLO https://github.com/kaovilai/tailbond/releases/latest/download/SHA256SUMS
sha256sum --ignore-missing -c SHA256SUMS
```

Releases appear once the first release is published; until then,
[build it from source](#build-from-source).


GL's **APPLICATIONS → Plug-ins** page has no upload ([GL docs:
Plug-ins][gl-plugins]); it installs from software sources, and tailbond's
signed feed can be one ([Install and update from GL's Plug-ins
page](#install-and-update-from-gls-plug-ins-page)), once the router trusts
its key. A local package goes through LuCI, which GL 4.x ships ([GL docs:
Advanced Settings][gl-adv]):

1. Get the package: the GL.iNet download above (the file ending in
   `_neon-vfpv4.ipk`). A home server offers it too: the Fedora page's
   **Devices → Add a GL.iNet travel router** (`/tailbond-glinet.ipk`, from
   `/usr/share/tailbond/tailbond-glinet.ipk`), or a Flint 3's panel, which
   rebuilds the package it runs (below).
2. On the travel router: **SYSTEM → Advanced Settings → Go To LuCI** (same
   admin password), then **System → Software → Upload Package…**, pick the
   file, **Install**. Leave "Allow overwriting" unticked.
3. In GL's admin, open **VPN → tailbond**. On a Slate 7 tap **Ask to connect**
   (below). A Flint 3 (`GL-BE9300` in `/tmp/sysinfo/model`) starts as the home
   server; the panel switches roles either way.

The package installs **enabled**: the agent starts right away (the panel
needs it) and the tunnel connects once paired. The LAN keeps its internet
until then (the kill switch arms only when paired and on). One command, if
you do have SSH: copy the file to `/tmp/` and
`opkg install /tmp/tailbond-glinet-aarch64_cortex-a53_neon-vfpv4.ipk`.

Updates (tailbond panel → Updates):

- **Right away**, the default on a new install, checks every hour. It
  installs a new release with `opkg` as soon as nobody is on a call or
  pulling a big download through the router: at most 30 minutes later, and
  at once for a security release.
- **Automatic (night)** installs between 03:00 and 05:00.
- **Tell me** only shows it until you press **Update now**.

Each install is checked against the release's `SHA256SUMS`, and the
previous package comes back if the new one doesn't come up. A router
installed before this keeps its policy ([updates.md](updates.md)).

A firmware upgrade removes every package you installed, tailbond too
(Speedify documents the same for its GL package). With **Keep Settings**
ticked, tailbond reinstalls itself on the first boot and keeps your pairing
and settings ([Firmware upgrades](#firmware-upgrades)). Without it, install
the package again and pair again.

A Flint 3 serves the package without having the file: `GET
/tailbond-glinet.ipk` rebuilds it from opkg's records
(`/usr/lib/opkg/info/tailbond.{control,list,postinst,…}`), using pristine
copies of the files that change after install
(`/usr/share/tailbond/pristine/…`) and never including `/etc/tailbond/`.

## Install and update from GL's Plug-ins page

Each release is also a signed opkg feed. GL's Plug-ins page can use it as
a custom software source, then installs and upgrades tailbond like any
other plug-in. GL firmware checks feed signatures (`option check_signature`
in `/etc/opkg.conf`) and refuses a feed signed by a key that isn't in
`/etc/opkg/keys/`, so the router has to trust tailbond's key first.

1. **The key**, one of:
   - Install tailbond once by hand ([above](#install-without-ssh)). The
     package installs its key as `/etc/opkg/keys/<fingerprint>`.
   - With SSH, before the first install:

     ```sh
     wget -O /tmp/tailbond-opkg.pub https://github.com/kaovilai/tailbond/releases/latest/download/tailbond-opkg.pub && opkg-key add /tmp/tailbond-opkg.pub
     ```

     `usign -F -p /tmp/tailbond-opkg.pub` prints the key's fingerprint.
     It must match the file name in
     [`packaging/openwrt/files/etc/opkg/keys/`](../packaging/openwrt/files/etc/opkg/keys/)
     in this repository.
2. **APPLICATIONS → Plug-ins → Manage Sources → Add Custom Software
   Source**:

   | Field | Value |
   |---|---|
   | Name | `tailbond` |
   | URL | `https://github.com/kaovilai/tailbond/releases/latest/download` |

   GL writes `src/gz tailbond https://github.com/kaovilai/tailbond/releases/latest/download`
   to `/etc/opkg/customfeeds.conf`. tailbond never adds this line itself.
3. Update the plug-in list, find **tailbond**, and **Install** it (or
   **Upgrade** it when a new release is out). Then open **VPN → tailbond**
   as above.

The feed is three files next to each release's packages: `Packages`,
`Packages.gz` and `Packages.sig`. It lists both
`tailbond_<version>-r1_<arch>.ipk` packages, and opkg picks the one for
the router's architecture. `releases/latest/download` always points at
the newest release. A release that comes out between a list update and an
install moves the files: update the list again. Releases from before the
feed key existed have no feed.

**To update, the tailbond panel is still better.** A Plug-ins upgrade is a
plain `opkg upgrade`. opkg checks the file against the signed list's
SHA-256, but nothing checks that the new agent comes up, and nothing puts
the old package back if it doesn't. The panel's updates (**Update now**,
**Right away**, **Automatic**) wait for a quiet moment and roll back a
package whose agent doesn't come up ([updates.md](updates.md)). The two
work side by side: after either one installs a version, the other sees it
as current. A firmware upgrade still reinstalls a version the Plug-ins page
installed ([Firmware upgrades](#firmware-upgrades)).

If the feed key ever changes, releases ship the old and the new key
together for a while first, so a router that keeps updating never notices.
A router that missed those releases gets "signature check failed" for the
source: add the key again (step 1). Removing tailbond also removes its key. The source stays until you delete
it under **Manage Sources**. Until then, GL's list update reports a
failed signature check for it.

## OpenWrt 25.12 and apk

OpenWrt 25.12 replaced opkg with **apk** (the Alpine package keeper) —
24.10 and 23.05, and GL.iNet's stock firmware, keep opkg and the `.ipk`.
The same package ships as an `.apk` for 25.12+ routers
(`aarch64_cortex-a53`, plain OpenWrt on this hardware):

```sh
wget -O /tmp/tailbond.apk https://github.com/kaovilai/tailbond/releases/latest/download/tailbond-openwrt-aarch64_cortex-a53.apk
apk add --allow-untrusted /tmp/tailbond.apk
# apk pins the file's content hash in /etc/apk/world when installing a
# file, which would block every later upgrade; name the package plainly:
sed -i 's/^tailbond[><=].*/tailbond/' /etc/apk/world
```

Then configure it as on GL firmware (**Services → tailbond** in LuCI; the
GL panel is GL firmware's). Everything the package ships is the same as
the ipk's, including the config surviving upgrades (`/etc/config/tailbond`
is a conffile through `/lib/apk/packages/tailbond.conffiles`: a changed
config is kept, the new default lands next to it as `.apk-new`), the
boot-time reinstall after a firmware upgrade, and the agent's updates with
rollback. One gap: on an apk system the agent can't rebuild its own
package for the rollback copy (`GET /tailbond-glinet.ipk` reads opkg's
file list), so a hand-installed router updates without a rollback copy the
first time; every update after that keeps one.

**An apk feed can't live on GitHub Releases**: apk does not follow
redirects, and every Releases URL redirects
([openwrt/openwrt#17180](https://github.com/openwrt/openwrt/issues/17180)).
`scripts/release/apk-feed.sh` builds a signed feed (an EC P-256 key,
`scripts/release/gen-apk-key.sh`; the key the packages ship lets a router
that installed one trust it) for a server that answers directly — GitHub
Pages, a VPS. Until that is hosted, install by hand and update from the
panel or the release.

Both the per-push CI (pinned image) and a weekly run against the newest
25.12 rootfs image ([openwrt-latest.yml]; the README's *OpenWrt latest*
badge) install the apk in a container and run the same checks as the 23.05
test: panel files, the agent, the /rpc backend, bypass rules, render-config,
updates with rollback, firmware-upgrade reinstall, the half-installed
repair, and the signed feed with its rejections.

[openwrt-latest.yml]: https://github.com/kaovilai/tailbond/actions/workflows/openwrt-latest.yml

## Build from source

For contributors (podman; the router needs no SDK):

```sh
scripts/build-mqvpn-aarch64-musl.sh      # static mqvpn → out/aarch64/mqvpn
packaging/openwrt/build-ipk.sh           # tailbondd (arm64) + both .ipk files → out/ipk/
                                         # (carries out/android/tailbond-debug.apk or TAILBOND_APK when present)
scripts/test-openwrt-ipk.sh              # emulated OpenWrt 23.05 arm64 install test (podman)
packaging/openwrt/build-apk.sh           # the same package as an .apk (OpenWrt 25.12+) → out/apk/
scripts/test-openwrt-apk.sh              # emulated OpenWrt 25.12 arm64 (apk) install test (podman)
```

`out/ipk/` gets `tailbond_<version>_aarch64_cortex-a53_neon-vfpv4.ipk` (GL
stock firmware) and `tailbond_<version>_aarch64_cortex-a53.ipk`; install
one the same way (LuCI upload). A release has both under these versioned
names too: the router's package cache and its reinstall after a firmware
upgrade read the version from the name.

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
| Fallback | `/etc/nginx/gl-conf.d/tailbond.conf` (shipped as `/usr/share/tailbond/nginx-tailbond.conf`) | Same-origin `/tailbond-api/` proxy (and `/tailbond-glinet.ipk`), checking the `Admin-Token` cookie with `ubus call gl-session session`. **Needs nginx's Lua module, which GL firmware 4.10 (Flint 3) lacks**: postinst enables it only if `nginx -t` still passes, otherwise it is skipped and logged, because a config nginx can't parse takes GL's admin UI down at the next reload. Without it, panel features that move files (backup download/restore, package download) aren't available from the panel |
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

### Servers and failover

The travel router can keep more than one server, for example `homebox` at
home and a `vps` as a backup. One tunnel runs at a time; if the server in
use stops answering for 15 s, the next one in the list takes over, and the
first one is used again once it has answered for 2 minutes.

In the panel (**Home server** card, which becomes **Servers** with two or more):
- **Add a backup server**: ask a server found on your tailnet (it approves you,
  like the first one), or paste a pairing link from it. It's added as a backup.
- The list: first choice and backups, each with its state (Connected,
  Standby, Can't reach, Connecting…, Checking…, Off) and when it last
  answered. Arrows change the order, **Use now** switches at once (it stays
  there while that server works), **Remove** forgets one server.
- **When to switch**: "Prefer homebox, use vps if homebox is down" (default)
  or "Always use vps" (never switches by itself).
- The top card says what's happening: "Using vps (homebox is unreachable)",
  then "Back to homebox in 1:20" while it counts down; **Recent switches**
  lists the last ones.
- **Kill switch** (UCI `client.kill_switch`, on by default): while no server
  can be reached, or during a switch, devices here wait instead of using the
  WAN unprotected. The switch in the card takes effect the next time the
  tunnel reconnects.

Each server needs its own tunnel addresses. A VPS set up with
`packaging/linux/install.sh` picks a free range by itself; a new server with
no devices yet moves out of the way when it approves the request. Otherwise
the panel says which server clashes: change `server.subnet` on the new one
(for example `10.78.0.0/24`), restart tailbond there and add it again.
Contract: [architecture.md](architecture.md#multiple-servers-and-failover).

### Bypass tailbond

Rules that let some traffic skip the tunnel (or never skip it):
- **A website** (and everything under it): a bank that blocks VPNs, streaming
  that should see where you are;
- **A device** on the router's LAN, picked from the list (by MAC, and by its
  DHCP name, so a new private Wi-Fi address keeps the rule): a TV stick
  that should use the hotel internet, or a work laptop that must **always use
  tailbond** (that beats website rules);
- **An address or port** (under *More*): Wi-Fi calling is UDP 500 and 4500.

Each rule either uses the internet directly (GL's own multi-WAN picks the
connection), uses only one connection (Repeater, Tethering…), or always uses
tailbond. Quick picks: Banking, Streaming location, Wi-Fi calling, Work
laptop. Each rule shows how many connections it has matched. Rules that
skip tailbond keep working while the tunnel is down (kill switch or not).

Website rules need **dnsmasq-full** (dnsmasq fills an nftables or ipset set
with each address it looks up for the website). GL 4.10 on the Flint 3 and
Slate 7 ships dnsmasq-full with nftset support and firewall4 (verified from
the firmware images), so website rules work out of the box there. Other
firmwares may need `opkg install dnsmasq-full`; until then the panel shows
website rules as waiting and device, address and port rules still work.
Devices with their own secure DNS (Private DNS on Android, Secure DNS in
Chrome) aren't matched by website rules; use a device rule for those.
Contract: [architecture.md](architecture.md#bypass-rules-client-role).

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

**Phones**: the panel's **Add an Android phone** card shows a QR code of
`http://homebox.<tailnet>.ts.net/`; typing `homebox` into a phone's browser
(Tailscale on) opens the same phone page: the Android app (the package
carries it, about 27 MB, at `/usr/share/tailbond/tailbond.apk`), pictures of
Android's install prompts, and **Open tailbond and ask homebox**. The page
offers only that: no API, no settings. GL's nginx owns port 80, so the
agent serves the page on `main.landing_port` (8078) of its Tailscale IPs and
`server-up` redirects tailnet tcp/80 there (UCI `firewall.tailbond_landing`;
GL's own admin isn't reachable from the tailnet zone by default, so nothing
it served there is hidden). `landing_port 0` turns it off.

`server-up` opens UDP 443 on the `wan` zone and TCP 8077 from the tailnet
zone (discovery and pair requests), adds a `tailbond` zone for `tb0` with
masquerade to `wan`, forwarding to `tailscale0` (tailnet) and, if
`allow_lan` is set, to `lan`. If the Flint 3 isn't your edge router, forward
UDP 443 to it on the upstream router. Who can approve: anyone logged in to
GL's admin, and anyone on the tailnet who opens `http://homebox:8077` unless
the panel sets a password for that page.

## Brume 2 (GL-MT2500), as a home server

Installed and run as a server on one real Brume 2; its quirks, so you can
skip the digging. Everything under [Home router](#home-router-flint-3-server)
applies.

| | Brume 2 |
|---|---|
| SoC | MediaTek MT7981 |
| Firmware | GL 4.x on OpenWrt 21.02, kernel 5.4, fw3 (iptables) |
| opkg arch | `aarch64_cortex-a53` (**no** `_neon-vfpv4`) |

- **Package:** use the plain OpenWrt ipk
  (`tailbond-openwrt-aarch64_cortex-a53.ipk`), not the GL.iNet one: check
  with `opkg print-architecture`. 21.02 is older than the 23.05 the docs
  list; it installs and runs, the rest of this list is what to expect.
- **Role:** a fresh install starts as `client`. Open **VPN → tailbond** and
  switch to **server** (or `uci set tailbond.main.role=server && uci commit
  tailbond && /etc/init.d/tailbond restart`).
- **"tailbond not running" while `tailbondd` runs:** Go listens with MPTCP
  by default, and this firmware's kernel has an MPTCP stack that accepts
  the listening socket but never completes a connection to it, loopback
  included, so the panel's `/rpc` call to `127.0.0.1:8077` times out.
  The agent now checks each new listener with a plain TCP connection and
  falls back to plain TCP when it doesn't connect (the log says `an MPTCP
  listener on … didn't accept a connection; using plain TCP`). To force it,
  set `option tcp_mode 'tcp'` (or `'mptcp'`; default `'auto'`) in
  `/etc/config/tailbond` and restart, or set `TAILBOND_TCP_MODE` for the
  agent. Builds from before this check: add
  `procd_set_param env GODEBUG=multipathtcp=0` to `/etc/init.d/tailbond`
  after the `procd_set_param command` line. Check: `curl -s -m5 -o /dev/null
  -w '%{http_code}\n' http://127.0.0.1:8077/api/status` prints `200`.
- **The package cache can come up empty** if the agent was hung when the
  package installed (nothing to cache from). Run
  `/usr/lib/tailbond/pkg-cache seed` once the agent answers, so a firmware
  upgrade with **Keep Settings** can reinstall it
  ([Firmware upgrades](#firmware-upgrades)).
- **Log lines that are expected:** `nginx can't load the admin panel
  fallback (no Lua module); using /rpc only` (the panel works over `/rpc`;
  the same firmware family as the Flint 3), and `Command failed: Not found`
  from `opkg` during install and `restart` (the package still configures
  and starts).
- **Admin over WAN is blocked** by GL's firewall, as on every GL router,
  and SSH too. A Brume behind another router is reachable from its LAN
  side, or over Tailscale once GL's Tailscale is on.
- **Behind carrier-grade NAT:** see [behind-cgnat.md](behind-cgnat.md).
- **Behind another router:** the Brume's WAN gets a private address, so
  forward UDP 443 to it on the upstream router, or keep phones on
  Tailscale. `server-up` already opens UDP 443 on the Brume's own `wan`
  zone.
- **Not tested:** pairing a phone and bonding over this unit's data path.

## Pair & Share: phones lend their mobile data

Phones on the Slate 7's Wi-Fi can lend their 5G/LTE to it
([pair-and-share.md](pair-and-share.md)): on Android tailbond → Settings →
**Share my mobile data**; on iPhone the **tailbond Share** app ([ios.md](ios.md)).
A Mac or Linux laptop on its Wi-Fi can lend its other connection, e.g. the
hotel's Ethernet ([Computers can share too](pair-and-share.md#computers-can-share-too)).
The panel's **Shared by nearby devices** card lists each one (sharing,
battery, why it stopped) with **Use connections that nearby devices share**;
each also appears under **Internet connections** as *Mobile data from <phone>* (or *Ethernet from <computer>*) with
Use it / Backup only / Off and a data limit, defaulting to the mobile tier
(kept in reserve by **Save mobile data**).

The router finds phones over mDNS on `br-lan` (alongside GL's own `umdns`)
and through your home server; it talks to them on UDP 47077 from the LAN
side, which GL's default lan-zone rules allow. Phones on the router's own
Wi-Fi can always reach it; on a hotel network that isolates devices, the
panel says *This Wi-Fi blocks devices from talking to each other; sharing
isn't available here*.

A phone plugged in by USB with tethering on is a WAN like any other (no app
needed); an iPhone shows as **iPhone (USB)**.

## Firmware upgrades

What GL.iNet 4.x keeps on a firmware upgrade with **Keep Settings**, read
from the GL-BE3600 (Slate 7) 4.10.1 image: `/etc/config/`, every path in
`/lib/upgrade/keep.d/*` (tailbond's keeps `/etc/tailbond/`), changed
conffiles of every package, and `/etc/rc.local`, which runs at every boot.
Not kept: the packages themselves. GL's "Keep Installed Packages" (sent
together with Keep Settings) only reinstalls packages from GL's own feeds,
which don't have tailbond.

So the package leaves itself a way back:

| File (kept) | What it does |
|---|---|
| `/etc/tailbond/pkg/tailbond_<version>_<arch>.ipk`, `current` | the installed package (and the previous one, for rollbacks) |
| `/etc/tailbond/pkg/reinstall` | installs that package if tailbond is missing **or half-installed** (it checks a file, not only opkg's record); else downloads that version from the GitHub release (checked against `SHA256SUMS`); retries for 20 minutes; at most once per boot |
| one line in `/etc/rc.local` (`# tailbond-reinstall`) | runs it in the background at boot |
| `/lib/upgrade/tailbond.sh` (not kept, runs before the flash) | saves the running package if the cache is empty and adds these files to the keep list |

One GL.iNet 4.9.0 beta was seen keeping opkg's record and
`/usr/bin/tailbondd` while dropping `/usr/lib/tailbond/` and
`/usr/share/tailbond/` — tailbond half-installed, and `opkg install
--force-reinstall` could not even extract the files back, because that
opkg does not create the package's directories ([issue #49]). The boot
hook now notices the missing files and force-reinstalls from the cache,
the package's `preinst` creates its directories before extraction, and
`/etc/init.d/tailbond start` repairs instead of only failing. Fixing one
by hand:

```sh
opkg files tailbond | grep '^/' | while read f; do mkdir -p "$(dirname "$f")"; done
opkg install --force-reinstall /etc/tailbond/pkg/tailbond_<version>_<arch>.ipk
```

[issue #49]: https://github.com/kaovilai/tailbond/issues/49

After the upgrade, `logread -e tailbond-reinstall` shows what happened, and
the panel says "reinstalled after a firmware upgrade". Removing the package
(`opkg remove tailbond`) removes the rc.local line and the cached packages;
`/etc/tailbond/` stays. See also [updates.md](updates.md#glinet-firmware-upgrades).

## Report a problem

**Report a problem** at the bottom of the tailbond panel (and an alert after a crash): a redacted report you check, then GitHub in your browser; the router sends nothing. [reporting.md](reporting.md).

## Logs and troubleshooting

```sh
logread -e tailbond          # agent + hook output (tailbond-update: updates)
cat /etc/tailbond/state/update-result.json   # the last update or reinstall
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
- Emulated OpenWrt 25.12 arm64 (`scripts/test-openwrt-apk.sh`; per push on
  the pinned 25.12.6 image, weekly on the rolling tag): the apk package
  installs, conffiles survive upgrades (`.apk-new`), the world pin a file
  install leaves is normalized, updates with rollback, the firmware-upgrade
  reinstall, the half-installed repair, the GitHub-release fallback, and
  the signed apk feed with its rejections.
- `scripts/e2e-netns.sh`: pairing over Tailscale with fake `tailscale`
  CLIs, captive portals (own probe, mocked GL repeater with Login Mode, the
  real portal hooks against a stub uci).
- `scripts/e2e-share.sh`: Pair & Share with a Linux sharer and hub in
  namespaces (2 paths on the server, sharer gone → clean again within
  seconds, client-isolation note).
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
