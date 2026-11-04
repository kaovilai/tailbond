# tailbond: architecture and cross-component contract

tailbond = Speedify-style WAN bonding to **your own home server**, with your tailnet reachable through it.
Data plane: [mqvpn](https://github.com/mp0rta/mqvpn) (MASQUE CONNECT-IP over Multipath QUIC), pinned as `third_party/mqvpn` (v0.19.0).
tailbond adds the management agent, web UI, Tailscale integration, pairing, packaging, and the Android app.

Every component below MUST follow this document. Change it here first.

## Components

| Component | Path | Runs on | Role |
|---|---|---|---|
| `tailbondd` agent + web UI | `agent/` (Go, UI embedded) | Fedora (x86_64/aarch64), GL.iNet (aarch64 musl), macOS (arm64) | supervises mqvpn, serves dashboard + API, Tailscale glue |
| mqvpn patches | `patches/mqvpn/*.patch` | all | client-side control API, relay paths, their macOS wiring (see below) |
| Fedora package | `packaging/fedora/` | Fedora 43+ | RPM: `tailbond-server` (mqvpn + tailbondd + units + firewalld service) |
| OpenWrt / GL.iNet package | `packaging/openwrt/` | Flint 3 (GL-BE9300), Slate 7 (GL-BE3600), any aarch64_cortex-a53 OpenWrt | `.ipk` with static musl mqvpn + tailbondd, init.d, UCI, GL admin panel (VPN → tailbond: oui view + `/rpc` Lua object, [glinet.md](glinet.md)) |
| Android app | `android/` | Android 8+ arm64 | fork of mqvpn's Android app, rebranded `dev.tailbond.app` |
| macOS client | `packaging/macos/` | Apple silicon Macs | the hub on a laptop: LaunchDaemon, `install.sh`, Homebrew formula, darwin hooks ([macos.md](macos.md)) |
| Generic Linux server bundle | `packaging/linux/` (+ Ansible role `packaging/ansible/roles/tailbond/`) | Debian/Ubuntu, Alma/RHEL/Rocky 9–10, Fedora; x86_64 | static mqvpn + tailbondd, `install.sh` (ports, subnet, firewall, systemd memory limits), for a VPS server ([vps.md](vps.md)) |

Roles of `tailbondd` (`--role` / config `role`):
- `server`: home endpoint (Fedora box or Flint 3). Runs `mqvpn --mode server`.
- `client`: travel router (Slate 7), Linux laptop or Mac. Runs `mqvpn --mode client` over all WANs; routes LAN through the tunnel (router) or the machine's own traffic (laptop, Mac: mqvpn's `manage_routes`).

## Network layout (defaults)

- mqvpn listen: UDP `443` on the home server (port-forward on the home router if the server is behind it).
- Tunnel subnet: `10.77.0.0/24` (server = `10.77.0.1`), IPv6 `fd77:77::/112`. TUN name `tb0`.
- Tailnet access (server role):
  - `tailscale set --advertise-routes=10.77.0.0/24` so tailnet peers reach bonded clients by tunnel IP (requires route approval in the admin console, once).
  - Bonded clients reach the tailnet: route `100.64.0.0/10` and `fd7a:115c:a1e0::/48` from `tb0` to `tailscale0` with masquerade. The server is one tailnet device; clients don't appear in the admin console.
  - Optional `--advertise-exit-node`; not required, since bonded clients exit via the server's internet NAT (mqvpn-server-nat.sh).
- DNS for clients: `tailbondd` runs a forwarder on `10.77.0.1:53`. `*.ts.net` and the tailnet's MagicDNS suffix go to `100.100.100.100`, everything else to the system resolver. mqvpn's server cannot push DNS, so the pairing payload carries `dns: ["10.77.0.1"]` and the client sets `DNS = 10.77.0.1`.
- Hybrid TCP lane: the server's egress ACL denies private ranges by default, including CGNAT. The server config always has `EgressAllow` for `100.64.0.0/10` and `fd7a:115c:a1e0::/48` so tailnet TCP works.
- TLS: if `tailscale cert <machine>.<tailnet>.ts.net` works, use it as mqvpn `Cert`/`Key`. Clients connect to the public address with `tls_server_name = <machine>.<tailnet>.ts.net` and verify normally. Fallback: self-signed + `insecure: true` plus the cert SHA-256 pinned in the pairing payload (enforced by the agent/app, not by mqvpn).

## Modes (user-facing → mqvpn config)

The wire names below are the contract. Apps show plain words instead
(Android: Fastest / Smooth calls and video / Most reliable / Save mobile data;
priorities Main / Extra / Backup only).

| Mode | Purpose | Scheduler | Other keys |
|---|---|---|---|
| `speed` (default) | aggregate bandwidth | `wlb` | Hybrid on, Reorder on (`Profile = fiber_lte`), `Reinjection = deadline`, `RequestServerScheduler = true` |
| `streaming` | calls/streams stay up during link trouble | `wlb` | `Redundant = true` (UDP), `Reinjection = idle`, `CC = bbr`, Hybrid on |
| `redundant` | maximum reliability, every UDP packet on every path | `minrtt` | `Redundant = true`, `Reinjection = idle`, `CC = bbr`, Hybrid on, `RequestServerScheduler = true` |
| `failover` | one primary link, others only on failure | `minrtt` | Reinjection off; agent/app keep non-primary paths in standby. Experimental: `backup_fec` |

Per-path user settings (client side): `enabled`, `priority` (`primary`/`secondary`/`backup`), `data_cap_mb` (backup paths drop out after the cap).

## Pairing payload

A server issues one credential per client (mqvpn `users`, via control API `add_user`, persisted to config).
It's shared as a QR code or URI:

```
tailbond://pair?d=<base64url(JSON)>
```

```json
{
  "v": 1,
  "name": "pixel",
  "server": "203.0.113.7:443",
  "sni": "homebox.tail1234.ts.net",
  "user": "pixel",
  "key": "<psk>",
  "insecure": false,
  "cert_sha256": "<hex, only when insecure>",
  "dns": ["10.77.0.1"],
  "mode": "speed",
  "tailnet_routes": ["100.64.0.0/10", "fd7a:115c:a1e0::/48"]
}
```

`server` may be a DDNS hostname. The Android app and client-role agent both import this.

Additive keys (servers since multi-server support always send them; importers
must accept payloads without them):
- `server_name`: the server's own short name (`homebox`, `vps`), used as the
  server's name in a client's [server list](#multiple-servers-and-failover).
  Missing: the first label of `sni`, else the host of `server`.
- `subnet`: the server's IPv4 tunnel subnet (`server.subnet`, e.g.
  `10.77.0.0/24`). Missing: the /24 around `dns[0]`, else `10.77.0.0/24`.
- `share_key`: the server's [Pair & Share](#pair--share) key. Missing: the
  client fetches it from `GET /api/shares` once connected.

## mqvpn client control API (patch)

Upstream mqvpn only serves the control API in server mode. `patches/mqvpn/0001-client-control-api.patch` adds it to client mode on Linux: `[Control] Listen` / `--control-port` in client mode serve `get_status` and `get_stats` with the **same JSON shape as server mode**, as one client object whose path entries additionally carry `"iface"` (the bound interface name). Keep the patch upstreamable (small, confined to `src/platform/linux/` + `src/platform/posix/`).

`patches/mqvpn/0003-darwin-client-control-and-relay-paths.patch` (on top of 0001 and 0002) does the same on macOS: the Darwin CLI compiles the same `control_socket.c` (plain POSIX + libevent; it stays in `src/platform/linux/`, a move to `posix/` being a pure rename) and serves the client API (no RX offload counters there), and a relay path (`NAME@HOST:PORT`) is wired into `platform_darwin.c`: its socket is not `IP_BOUND_IF`-pinned and is routed normally, and the route monitor, the reactivate re-pin and the scoped server pins (`#F1`) skip it, since its name is not an interface. Interface paths are unchanged. `test_control_socket` joins mqvpn's Darwin ctest set, which already runs the relay bind tests (`test_bind_posix_socket`).

## tailbondd HTTP API (served on `:8077`, UI at `/`)

Auth: if `ui_password` is set, a session cookie from `POST /api/login {"password"}`. Without one (server role), requests from tailnet addresses get the admin API only when `tailscale whois` puts the device under the server's owner (the login of `tailscale status` Self's user; none for a tagged machine) or a login in `approvers`; others get 403 `{"ok":false,"error":"owner only","owner":"<allowed logins>"}`. Loopback and other sources, the phone page and the pair-request endpoints are unaffected; a tagged server with no `approvers` keeps the old rule (anyone who can reach it). `POST /api/settings` also takes `approvers` (list of logins); `GET` reports `owner` and `approvers`. Pair-request replies (to the requester) additionally carry `device`, `user`, `ip`: how the approval card shows it. Server role binds `127.0.0.1` plus the Tailscale IP by default (or the configured `listen` host), and always also its tunnel IP (`10.77.0.1:8077`, bound once `tb0` has it) unless `listen` is a wildcard. Client role binds the LAN.

Requests that arrive on a tunnel address (the server's `subnet`/`subnet6`) come from bonded clients. There the agent serves **only** `GET /api/peers` with a client API token (below); everything else is 404, with or without `ui_password`.

| Method | Path | Body / result |
|---|---|---|
| GET | `/api/status` | normalized status (below) |
| GET | `/api/events` | SSE stream of the same object, once a second |
| POST | `/api/mode` | `{"mode":"speed"\|"streaming"\|"redundant"\|"failover"}`; the agent rewrites config and restarts mqvpn |
| GET/POST | `/api/paths` | client: list interfaces with settings / update `{name, enabled, priority, data_cap_mb}` |
| POST | `/api/connect`, `/api/disconnect` | client only |
| GET | `/api/clients` | server: registered clients and live sessions |
| POST | `/api/clients` | server: `{"name"}`, returns `{"uri","qr_svg"}` |
| DELETE | `/api/clients/{name}` | server: revoke |
| POST | `/api/pair` | client: `{"uri"}` to import a pairing (additive: it replaces the primary server; `"add": true` = `POST /api/servers`) |
| GET | `/api/tailscale` | `tailscale status --json` summary + route approval state |
| GET | `/tailbond.apk` | additive: the Android app, if the file at `android_apk` exists (else 404); `/api/settings` reports `android_apk: true/false` |
| GET | `/api/peers` | server, additive: the tailnet machines bonded clients reach (below). Auth: client API token, or an admin session |
| GET | `/api/clients/{name}` | server, additive: re-issue `{"uri","qr_svg"}` for an existing client |
| GET | `/api/logs?since=<seq>&limit=<n>` | additive: `{"entries":[{"seq","t","src":"agent\|mqvpn\|hook","line"}]}` ring buffer |
| GET/POST | `/api/settings` | additive: read role/TLS/pairing info; set `public_addr`, `ui_password`, `captive_portal_detect` (client) |
| POST | `/api/logout` | additive |
| GET | `/api/hello` | additive, **unauthenticated**: `{"tailbond":true,"name","version","role"}` ([pairing over Tailscale](#pairing-over-tailscale)); additive: `protocol` ([updates](#updates)) |
| POST | `/api/pair-requests` | server, additive, **tailnet sources only**: `{"name","kind":"router"\|"phone"}` → 202 `{"id","token","status":"pending","name","expires"}` |
| GET | `/api/pair-requests/{id}` | server, additive: the requester polls with `Authorization: Bearer <token>` from the same address → `{"id","status","name","expires"}`, plus `"uri"` once approved |
| GET | `/api/pair-requests` | server, additive (admin): every live request `{"id","name","kind","device","dns_name","hostname","os","user","user_name","ip","status","created","expires","client","repair"}` |
| POST | `/api/pair-requests/{id}/approve`, `/decline` | server, additive (admin): body `{"name"}` optional. Refused (403) from the requesting address |
| GET | `/api/discover?refresh=1` | client, additive: tailbond servers among online tailnet peers `{"tailscale_installed","tailscale_running","self","tailnet","servers":[{"name","dns_name","ip","os","version"}],"checked","at"}` |
| GET/POST/DELETE | `/api/join` | client, additive (the dashboard's Setup → **Ask your home server**; `/?ask=<server>` fills it in): ask a server to pair (`{"server":"homebox"\|"homebox.tail…ts.net"\|"100.x"}`), its state `{"state":"idle\|asking\|waiting\|approved\|declined\|expired\|error","server","server_name","id","name","error"}`, cancel. Additive: `"add": true` adds the approved server to the [server list](#multiple-servers-and-failover) as a backup instead of replacing the primary; `error` may then be `subnet_in_use` or `exists`, with `conflict` (the server already in the list) and `subnet` |
| DELETE | `/api/pair` | client, additive: forget the pairing, every server in the list (tunnel stops, LAN back on the WANs) |
| GET | `/api/peers` | client, additive: the server's list (below), fetched through the tunnel with this client's token, cached 20 s |
| POST/DELETE | `/api/captive/signin` | client, additive: `{"name": <network or device>}` sign-in mode on a WAN behind a portal (409 otherwise) / off ([captive portals](#captive-portals)) |
| POST | `/api/captive/check` | client, additive: `{"name"}` re-check now (`""` = all) |
| GET | `/api/health` | additive: [setup health](#setup-health) |
| GET | `/tailbond-glinet.ipk` | additive: the GL router package, from `glinet_ipk` (default `/usr/share/tailbond/tailbond-glinet.ipk`) or rebuilt from opkg's records of the installed package |
| GET | `/get` | server, additive, **unauthenticated**: the [phone page](#phone-page) (HTML), also served on port 80 |
| GET | `/tailbond-macos-arm64.tar.gz` | server, additive, **unauthenticated**: the macOS bundle for the phone page's Mac section (`macos_bundle`, else a copy fetched once from the GitHub release and cached; 404 when there is none), also on port 80 |
| GET | `/api/servers` | client, additive: `{"servers":[…],"active_server","failover":{…}}`, the same objects as in `/api/status` ([multiple servers](#multiple-servers-and-failover)) |
| POST | `/api/servers` | client, additive: `{"uri", "name"?}` adds a server at the end of the list (a backup). 409 `{"error":"subnet_in_use","conflict":"<server>","subnet"}` when its tunnel subnet overlaps another server's; 409 `{"error":"exists","conflict"}` when it's the same server |
| DELETE | `/api/servers/{name}` | client, additive: forget one server (the next one takes over if it was active) |
| POST | `/api/servers/active` | client, additive: `{"name"}` "Use this server now" (switches immediately; no automatic failback until it fails) |
| POST | `/api/servers/order` | client, additive: `{"names":[…]}` the full list in the new order; the first is the primary |
| POST | `/api/servers/policy` | client, additive: `{"policy":"prefer"\|"always","server"?,"switch_after_sec"?,"failback_after_sec"?}` |
| GET | `/api/bypass` | client, additive: `{"rules":[…],"support":{…},"wans":[{"name","label","up","default"}],"counters":{"<id>":{"connections"}}}` (connections a rule sent past the tunnel since it was applied) ([bypass rules](#bypass-rules-client-role)) |
| POST | `/api/bypass` | client, additive: one rule; without `id` (or an unknown one) it is added, else replaced. Returns the stored rule (with its `id`) |
| DELETE | `/api/bypass/{id}` | client, additive |
| GET | `/api/lan-clients` | client, additive: devices on the LAN `[{"mac","ip","name","online"}]` from the DHCP leases (`client.leases_file`, default `/tmp/dhcp.leases`) and the neighbour table, online first, then by name |

`GET /api/settings` (server role) additionally reports `landing_url` (where
phones get tailbond: `http://<full MagicDNS name>/` while the phone page
listens, else `http://<name>:8077/get`), `landing_typed` (what to type in a
phone's browser, e.g. `homebox`) and `landing_qr_svg` (QR code of
`landing_url`).

#### Phone page

Server role: "Get tailbond", a page for devices that run Tailscale but not
tailbond yet, served on **port 80 of the Tailscale IPs** (config
`landing_port`, default 80, `-1` = off; `landing_public_port` is the port
tailnet devices use when a firewall redirect sits in between: OpenWrt binds
8078 and redirects tailnet tcp/80 to it). Typing the server's MagicDNS short
name in a browser opens it. It serves exactly `GET /` (HTML), `GET
/tailbond.apk`, `GET /tailbond-macos-arm64.tar.gz`, `GET /landing.js` and
`GET /icon.svg`; every other path redirects to `/`, whatever `ui_password`
says. Binding failures (port in use) are logged once and the page stays at
`/get` on the dashboard port.

The page has a section per platform behind a switcher: **Android** (the
APK with what Android's install prompts look like, Obtainium as the
alternative, a link that opens the app asking this server, and on other
devices a QR code of `<page>#android` to scan with the phone), **Mac** (a
one-line Terminal install from this server, the download button with its
SHA-256, Homebrew through a local tap; then `http://localhost:8077/?ask=<full
name>`, which opens the Mac's own dashboard with **Ask your home server**
filled in), **iPhone** (bonded with no app over the router's Wi-Fi or USB;
tailbond Share and docs/ios.md) and **Linux** (`install.sh --client` for a
laptop, the RPM and `install.sh` for a server). `#android`, `#mac`,
`#iphone`, `#linux` pick a section (CSS `:target`, no script needed);
without one, the server's guess from `User-Agent` / `Sec-CH-UA-Platform`
shows first (Windows: a note and the Android steps), refined in the browser
by `landing.js` (an iPad that asks for the desktop site). `landing.js` also
runs the copy buttons (with an `execCommand` fallback: `http://homebox/`
isn't a secure context). No external assets.

The Mac download: `macos_bundle` (default
`/usr/share/tailbond/tailbond-macos-arm64.tar.gz`) when present, else
`<state_dir>/downloads/tailbond-macos-arm64.tar.gz`, else the asset of this
version's GitHub release (the latest for a non-release build) downloaded
on the first request (or in the background when a Mac opens the page),
checked against the release's `SHA256SUMS` like an update, and cached. One
fetch at a time; after a failure none for 10 minutes; none at all with
`update.policy` `off`.

The app link is `intent://ask?server=<full MagicDNS name>#Intent;scheme=tailbond;package=dev.tailbond.app;S.browser_fallback_url=<page>#get-app;end`,
i.e. **`tailbond://ask?server=<name>`**: the app asks that server to pair
([pairing over Tailscale](#pairing-over-tailscale)) as if the name had been
typed. Only plain host names or addresses are accepted.

## Client API token and `GET /api/peers`

A paired client authenticates to the server's agent with

```
Authorization: Bearer <lowercase hex HMAC-SHA256(key = the pairing payload's "key", message = "tailbond-api")>
```

The agent recomputes it for every registered user (constant-time compare);
revoking a client revokes its token. The token opens `GET /api/peers` only;
it is not an admin session. Test vector: key `test-key-123` →
`68c8131634495b2e11cfcb57f604aa091793b004cfac7f5ae50ae49d553c1747`.

`GET /api/peers` summarises `tailscale status --json` (online first, then by name):

```json
{
  "server": "homebox", "tailnet": "alex@example.com", "magic_dns_suffix": "tail1234.ts.net",
  "tailscale_running": true, "client": "pixel",
  "self": {"hostname": "homebox", "dns_name": "homebox.tail1234.ts.net", "tailscale_ips": ["100.101.1.1"], "os": "linux", "online": true},
  "peers": [
    {"hostname": "nas", "dns_name": "nas.tail1234.ts.net", "tailscale_ips": ["100.101.2.2", "fd7a:115c:a1e0::2"],
     "os": "linux", "online": true},
    {"hostname": "work-laptop", "dns_name": "work-laptop.tail1234.ts.net", "tailscale_ips": ["100.101.3.3"],
     "os": "macOS", "online": false, "last_seen": "2026-10-30T08:00:00Z"}
  ]
}
```

`dns_name` has no trailing dot; `last_seen` is omitted when Tailscale reports
none. Additive per machine: `conn` (`"direct"` when Tailscale has a
`CurAddr`, `"relay"` when only a DERP `Relay`), `relay` (the DERP region). With Tailscale disabled or down, `peers` is `[]` and
`tailscale_running` is `false`. The Android app polls
`http://10.77.0.1:8077/api/peers` every 30 s while connected (cleartext is
allowed for that address only; the request travels inside the tunnel).

Mutating requests must send `Content-Type: application/json` (CSRF guard). `POST /api/paths` also accepts `kind` and `reset_usage`. On the server role, `/api/connect` and `/api/disconnect` start/stop mqvpn.

Normalized status:

```json
{
  "role": "client", "mode": "speed", "state": "connected|connecting|disconnected|error",
  "server": "homebox.tail1234.ts.net", "tunnel_ip": "10.77.0.2", "uptime_sec": 123,
  "totals": {"tx_bps": 0, "rx_bps": 0, "tx_bytes": 0, "rx_bytes": 0},
  "paths": [
    {"id": 0, "iface": "wwan0", "kind": "wifi|cellular|ethernet|usb|unknown", "state": "active|standby|failed|pending",
     "srtt_ms": 31, "min_rtt_ms": 18, "loss_pct": 0.2, "tx_bps": 0, "rx_bps": 0, "tx_bytes": 0, "rx_bytes": 0,
     "priority": "primary", "enabled": true}
  ],
  "clients": [],
  "tailscale": {"running": true, "self": "homebox", "routes_approved": true}
}
```

Rates (`*_bps`) are bits per second, derived by the agent from successive byte counters.

Additive fields (consumers must ignore unknown keys):
- top level: `error` (last mqvpn failure), `control_api` (live per-path stats available), `paired` (client), `version`;
- path: `data_cap_mb`, `used_bytes` (this month), `cap_reached`;
- `tailscale`: `installed`, `dns_name`, `ips`;
- `pair_requests` (server): the pending requests, same shape as `GET /api/pair-requests`;
- `join` (client): the `GET /api/join` object while not idle;
- path (client): `network` and `label` (OpenWrt: `wan`/"Ethernet", `wwan`/"Repeater", `tethering`/"Tethering", `modem_*`/"Cellular"), `captive` (`open`\|`portal`\|`offline`), `portal_url`, `signin_until`, `captive_source` (`tailbond`\|`glinet`); a WAN held back behind a portal has `state: standby`, not `failed`;
- `servers[]`, `active_server`, `failover` (client): see [multiple servers](#multiple-servers-and-failover);
- `clients[]` (server role), one per registered or connected client:
  `{"name","registered","connected","endpoint","scheduler","connected_sec","tx_bps","rx_bps","tx_bytes","rx_bytes","paths":[path…]}`.
  Server-side paths have no `iface` (mqvpn's server doesn't know it) and `kind` is `unknown`.

Client role without the control API patch: `state` comes from mqvpn's log (`state: … → ESTABLISHED`), totals from `tb0`'s kernel counters, per-path bytes from each bound interface's counters, RTT from the 30 s `[STATUS]` log lines, and `control_api` is `false`.

## Web UI

One single-page app (vanilla JS + SVG, no build step, embedded with `go:embed`), Speedify-like:
- a header with the total download/upload rate and a 60 s sparkline, and a connected/disconnected toggle;
- a mode selector (Speed / Streaming / Redundant / Failover);
- one card per connection: icon by kind, live rate sparkline, latency, loss, state chip, enable toggle, priority dropdown, data used vs cap;
- server role: a Devices tab (add a device → QR code, revoke, per-client paths) and a Tailscale panel;
- server role, Devices tab: **Add an Android phone** with a QR code of the [phone page](#phone-page) and its link;
- server role, viewed on a phone (or opened as `#add-phone`): an **Add this phone** card on top. One tap
  creates the client (`POST /api/clients`, name guessed from the phone model) and navigates to the
  `tailbond://pair` deep link, so the app opens with the pairing; no QR code on the same device.
  Re-adding an existing name reuses that client's key (`GET /api/clients/{name}`) after a confirm;
- client role: a Setup tab (paste/scan pairing URI).
- Works on phone screens; dark/light follows the system.

## Build outputs

- `scripts/build-mqvpn-native.sh` → `third_party/mqvpn/build/mqvpn` (Fedora).
- `scripts/build-mqvpn-musl.sh <aarch64|x86_64>` → static `out/<arch>/mqvpn` via `podman` + `docker.io/library/alpine` (native when the host has that arch, else qemu-user-static); `scripts/build-mqvpn-aarch64-musl.sh` is the aarch64 shorthand.
- `packaging/linux/build-bundle.sh` → `out/linux/tailbond-linux-<arch>.tar.gz` (installer, static binaries, hooks, unit).
- `agent`: `CGO_ENABLED=0 go build` for linux/amd64 and linux/arm64; `agent/build.sh darwin/arm64` for macOS.
- `scripts/build-mqvpn-macos.sh` (on a Mac) → `out/macos/mqvpn`, arm64, libevent from Homebrew linked statically; `packaging/macos/build-bundle.sh` → `out/macos/tailbond-macos-arm64.tar.gz` and the formula `out/macos/tailbond.rb`.
- RPM: `packaging/fedora/build-rpm.sh` → `out/rpm/`.
- IPK: `packaging/openwrt/build-ipk.sh` → `out/ipk/tailbond_<ver>_aarch64_cortex-a53.ipk` (plain ar/tar ipk; no OpenWrt SDK needed since binaries are static).
- APK: `android/` Gradle → `out/android/tailbond-debug.apk`. The RPM and the IPK carry it at `/usr/share/tailbond/tailbond.apk` when it exists at build time (or `TAILBOND_APK`).

## Platform hooks (keep the agent platform-neutral)

`tailbondd` never edits firewall, routing or Tailscale settings directly. It runs executable hooks from `hooks_dir` (default `/usr/lib/tailbond/hooks`) and logs their output:

| Hook | When | Args / env |
|---|---|---|
| `server-up` | after mqvpn server starts | env `TB_TUN`, `TB_SUBNET`, `TB_SUBNET6`, `TB_PORT`, `TB_DNS_IP`, `TB_WEB_PORTS` (additive: the tcp ports the agent serves to the tailnet, dashboard then phone page, e.g. `8077 80`), `TB_ADVERTISE_ROUTE` (additive: `0` when `server.no_advertise_route` is set) |
| `server-down` | after mqvpn server stops | same |
| `client-up` | tunnel connected | env `TB_TUN`, `TB_TUNNEL_IP`, `TB_SERVER_IP`, `TB_TAILNET_ROUTES`; additive: `TB_SERVER_NAME`, `TB_SUBNET` (the active server's tunnel subnet), `TB_DNS_IP` (its tunnel DNS, `""` when it has none) |
| `client-down` | tunnel down | the env `client-up` got for that tunnel (a server switch tears down the old server's routes, then brings up the new one's); `TB_RELEASE=1` when the user turned tailbond off or unpaired (the LAN goes back to the WANs whatever the kill switch says); additive: `TB_SWITCH=1` when another server is about to take over |
| `bypass-apply` | client, additive: start, rule changes, WAN up/down (SIGHUP), every 5 minutes, stop | env `TB_BYPASS_FILE` (JSON `{"rules":[enabled rules]}`; next to it the agent renders every variant: `bypass.nft`, `bypass.ipset`, `bypass.iptables`, `bypass.ip6tables`, `bypass-nftset.conf`, `bypass-ipset.conf`, `bypass-wans` (`<network> <mark> <table>` per line), `bypass-marks` (`<mask> <direct> <tunnel>`), `bypass-domains`), `TB_BYPASS_STATUS` (the hook writes `{"domains":"nftset"\|"ipset"\|"none","firewall":"nft"\|"iptables"\|"none","reason"}` there), `TB_BYPASS_ACTION` = `apply` \| `counters` (write `{"counters":{"<id>":{"connections"}}}` to `TB_BYPASS_STATUS`) \| `clear` (agent stopping: remove everything) |
| `portal-open` | sign-in mode on for one WAN (client) | env `TB_PORTAL_DEV`, `TB_PORTAL_NET`, `TB_PORTAL_DNS` (the WAN's DHCP DNS, comma separated), `TB_PORTAL_URL`, `TB_PORTAL_SOURCE` (`tailbond`\|`glinet`) |
| `portal-close` | sign-in mode off (open, timeout, stopped, shutdown) | same |

Every hook also gets `TB_ROLE`, `TB_MODE`, `TB_STATE_DIR` and `TB_CONFIG` (path of the generated mqvpn INI). Besides `<hooks_dir>/<hook>`, executables in `<hooks_dir>/<hook>.d/` run in lexical order.

`bypass-apply` exists only on OpenWrt; Fedora and the generic Linux installer have no client role, so they ship none (a missing hook is not an error).

- Fedora server hooks (`packaging/fedora/hooks/`): firewalld (open UDP port, `tb0` in a zone with masquerade toward the internet and `tailscale0`, `TB_WEB_PORTS` from tailnet source ranges only), `tailscale set --advertise-routes`. Without a running firewalld they fall back to mqvpn's NAT script plus iptables masquerade toward `tailscale0`.
- Generic Linux hooks (`packaging/linux/hooks/`, installed to `/usr/local/lib/tailbond/hooks` and set as `hooks_dir`): firewalld like Fedora, else an nftables `table inet tailbond`, else iptables (plus FORWARD accepts when that policy is DROP); same Tailscale route handling.
- OpenWrt hooks (`packaging/openwrt/hooks/`): UCI network/firewall zone `tailbond` for `tb0`, forwarding `lan → tailbond`, policy routing of LAN through `tb0`, server-role equivalents for Flint 3.
- macOS hooks (`packaging/macos/hooks/`): `client-up` / `client-down` only flush the DNS cache; mqvpn's Darwin layer sets the routes (`manage_routes`), the DNS servers (`networksetup`) and the optional pf kill switch itself, and undoes them when it stops.
- The agent runs mqvpn with `manage_routes = false` on OpenWrt and lets hooks route. On Fedora (server only), internet NAT comes from the firewalld `tailbond-out` policy, or from mqvpn's own NAT script when firewalld isn't running.

## Agent config file

`/etc/tailbond/tailbond.json`:

```json
{
  "role": "server",
  "mqvpn_bin": "/usr/bin/mqvpn",
  "state_dir": "/var/lib/tailbond",
  "hooks_dir": "/usr/lib/tailbond/hooks",
  "listen": ":8077",
  "ui_password": "",
  "mode": "speed",
  "server": {"port": 443, "public_addr": "", "subnet": "10.77.0.0/24", "subnet6": "fd77:77::/112", "use_tailscale_cert": true},
  "client": {"pairing_uri": "", "paths": [{"iface": "wan", "enabled": true, "priority": "primary", "data_cap_mb": 0}]}
}
```

The agent generates the mqvpn INI under `state_dir` (`server.conf` / `client.conf`) from this file plus the mode table, and enables mqvpn's control API on `127.0.0.1:9077`.

Additive keys (all optional):
- `server.users`: `[{"name","key","created"}]`, the registered clients (written by `POST /api/clients`; the file is mode 0600);
- `server.disable_dns`: no DNS forwarder (the pairing payload then has no `dns`);
- `server.no_advertise_route`: don't advertise the tunnel subnet to the tailnet (Fedora hook; OpenWrt uses UCI `server.advertise_routes`);
- `approvers`: Tailscale logins who may use the dashboard from the tailnet without a password, besides the server's owner (on a tagged server, the only ones);
- `client.manage_routes` (default `true`; set it to `false` on OpenWrt), `client.no_dns` (don't let mqvpn change the system resolver), `client.disconnected` (persisted connect toggle), `client.paths[].kind` (override the kind heuristic), `client.mtu` (tunnel MTU, default 1280: mqvpn's auto value assumes a 1500-byte uplink and loses large packets on smaller ones);
- `disable_tailscale`: skip every `tailscale` CLI call; `control_addr`; `log_level`;
- `android_apk`: APK offered as `GET /tailbond.apk` (default `/usr/share/tailbond/tailbond.apk`; nothing is offered when the file is missing);
- `glinet_ipk`: router package offered as `GET /tailbond-glinet.ipk`;
- `macos_bundle`: the macOS bundle the phone page offers (default `/usr/share/tailbond/tailbond-macos-arm64.tar.gz`; without it, fetched from the GitHub release and cached);
- `landing_port` (server: phone page bind port, default 80, `-1` = off), `landing_public_port` (the port tailnet devices use, default the same);
- `server.users[].ts_node`: the Tailscale stable node ID that asked to pair (re-asking re-issues this user's key);
- `client.paths[].iface` may be an OpenWrt network name (`wan`, `wwan`, `tethering`, `modem_0001`); unconfigured networks in the firewall's wan zone are used as primary (Ethernet, Wi-Fi) or secondary (tethering, cellular) paths;
- `client.captive_portal_detect` (default `true`), `client.captive_probe_urls` (default: Google's `generate_204`, Firefox's `success.txt`).
- `client.servers`: `[{"name","pairing_uri","enabled"}]`, the ordered server list (first = primary); `client.failover`: `{"policy":"prefer"|"always","server","switch_after_sec":15,"failback_after_sec":120,"probe_every_sec":10}` ([multiple servers](#multiple-servers-and-failover));
- `update`: auto-updates and the server's update mirror ([updates](#updates)); `backup`: scheduled encrypted backups ([backup and restore](#backup-and-restore));
- `client.bypass`: `[{"id","kind","match","ip","host","target","wan","enabled","note"}]` ([bypass rules](#bypass-rules-client-role)); `client.leases_file` (default `/tmp/dhcp.leases`); `client.bypass_mark_mask` (default `"0x07000000"`), `client.bypass_local_dns` (default `true`).

**Migration.** A config with `client.pairing_uri` and no `client.servers`
loads as a one-server list (`[{"name": <payload server_name>, "pairing_uri"}]`)
and is saved that way on the next change. `client.pairing_uri` stays
written, always equal to the primary's URI, so older tools and
`/etc/init.d/tailbond` keep working. A `client.pairing_uri` that isn't in the
list (set by hand or through UCI) replaces the primary.

## Pairing over Tailscale

Both ends are on one tailnet; no link to copy.

1. **Discovery.** Every tailbond agent answers `GET /api/hello`
   (unauthenticated, minimal; servers bind 127.0.0.1 + their Tailscale IP, and
   the OpenWrt server hook opens tcp/8077 from the tailnet zone). A client
   agent (`GET /api/discover`) lists online peers from `tailscale status --json`
   (at most 64) and probes `http://<peer 100.x>:8077/api/hello`; servers
   answer `"role":"server"`. Phones skip discovery: the user types the name,
   the phone resolves it through Tailscale (MagicDNS or 100.x) and refuses
   anything that isn't a Tailscale address.
2. **Request.** `POST /api/pair-requests` is accepted only from a tailnet
   source address (`100.64.0.0/10`, `fd7a:115c:a1e0::/48`), not over the
   bonded tunnel, and only if `tailscale whois --json <ip>` knows the device
   (a spoofed 100.x on the LAN fails here). The request records the device's
   tailnet name, OS hostname and login. One live request per device (asking
   again returns the same one), at most 3 new requests per device per
   10 minutes and 16 pending in total (429). Requests are in memory: a
   restart drops them and the device simply asks again.
   Additive: the body may carry `"avoid_subnets": ["10.77.0.0/24"]` (the
   asking client's other servers), see [distinct subnets](#multiple-servers-and-failover).
3. **Approval.** The admin sees pending requests in the dashboard, the GL
   panel and `pair_requests` in `/api/status`, and approves or declines with
   one tap (not from the requesting address). Approval creates a client
   (`POST /api/clients` semantics) under a free name, or, if this Tailscale
   node was paired before (`users[].ts_node`), re-issues its credential.
   Who can approve is whoever has admin access to the dashboard (no
   `ui_password`: anyone who can reach it, i.e. the tailnet, subject to your
   ACLs).
4. **Pick-up.** The requester polls `GET /api/pair-requests/{id}` every 3 s
   with the token from step 2 and from the same address; once approved the
   reply carries the usual `tailbond://pair` URI, which the client imports
   (Android: the normal confirm dialog). Pending requests expire after 10
   minutes, approved ones are served for 10 minutes, declined/expired ones
   stay listed for an hour.

## Multiple servers and failover

Client role (router agent and Android app). The client keeps an **ordered
list of servers**, e.g. `homebox` (primary, at home) and `vps` (a backup VPS).
Each comes from its own pairing (link, QR, or asking over Tailscale with
`POST /api/join {"server", "add": true}`) and has its own tunnel subnet,
DNS IP and tailnet routes. One tunnel runs at a time, to the **active**
server; the others are on standby.

**Distinct subnets.** Two servers must not share a tunnel subnet (both would
advertise it to the tailnet, and a server switch would keep stale routes).
- A server's subnet is `server.subnet` (default `10.77.0.0/24`). The generic
  Linux installer (`packaging/linux/install.sh`, a VPS) picks a random free
  `/24` in `10.78.0.0`–`10.99.255.0` that overlaps no local route.
- Asking to pair sends `{"avoid_subnets": [the client's other servers' subnets]}`
  with `POST /api/pair-requests`. If the server's subnet is in that list and
  the server has **no registered clients yet**, approving moves the server
  to the first free `/24` from `10.78.0.0/24` up (not in `avoid_subnets`, not
  overlapping a local route), saves it, logs it and restarts mqvpn (the
  hooks re-run with the new `TB_SUBNET`). A server that already has clients
  keeps its subnet.
- The client refuses a server whose subnet overlaps one already in its list
  (`409 subnet_in_use`); the UI says which server clashes and how to change
  `server.subnet` on the new one.

**Health.** The active server is working while mqvpn reports the
connection established and, with the control API, at least one path is
`active` and the server still answers: the mqvpn client sends keep-alive
PINGs, so on a live path `pkt_recv` moves whenever `pkt_sent` does. When
every path has had sends go unanswered for 5 s, the tunnel counts as down
(mqvpn itself would only give up after its 120 s idle timeout). Without the
control API (unpatched mqvpn) only the connection state counts. Standby servers are checked every `probe_every_sec` (10 s) with
one UDP datagram to their mqvpn port, sent outside the tunnel (router: bound
to a WAN; Android: on the underlying network): a 60-byte QUIC short-header
packet with a random connection ID, which any QUIC server answers with a
Stateless Reset (RFC 9000 §10.3; xquic does, without creating state). Any
reply within 2 s = reachable. This is not a tunnel pinger: the active
tunnel's health comes only from mqvpn.

**Policy** (`client.failover.policy`):
- `prefer` (default, "Prefer <primary>, use <backup> if <primary> is down"):
  - **failover**: when the active server has not been working for
    `switch_after_sec` (default 15 s; counted from the later of the last
    working moment and the switch to it), switch to the next server in list
    order (wrapping), skipping ones whose last probe failed when another one
    answered;
  - **failback**: when a server earlier in the list than the active one has
    answered every probe for `failback_after_sec` (default 120 s), switch
    back to it. One failed probe restarts the count (hysteresis). If a
    failback doesn't come up within `switch_after_sec`, the hold for that
    server doubles for the next try (up to 16×, reset once it stays working
    for 10 minutes), so a server that answers probes but can't carry a
    tunnel doesn't make the client flap.
- `always` ("Always use <server>"): only `client.failover.server`; no automatic
  switching.
- **Manual** ("Use this server now", `POST /api/servers/active`): switch at
  once; no failback while that server works; if it fails, `prefer` resumes.

A switch stops mqvpn, runs `client-down` with the old server's env
(`TB_SWITCH=1`; with the kill switch on the LAN is blocked for the few
seconds until the new tunnel is up), writes the new server's config and
starts mqvpn; `client-up` then gets the new server's `TB_SERVER_IP`,
`TB_SUBNET`, `TB_DNS_IP`, `TB_TAILNET_ROUTES`. Bypass rules don't depend on
the server and stay in place.

Status (`/api/status`, `/api/servers`), additive:

```json
"active_server": "vps",
"servers": [
  {"name": "homebox", "role": "primary", "state": "unreachable", "active": false,
   "last_ok": "2026-11-01T08:00:00Z", "subnet": "10.77.0.0/24", "server": "home.example.net:443"},
  {"name": "vps", "role": "backup", "state": "active", "active": true,
   "last_ok": "2026-11-01T08:03:10Z", "subnet": "10.83.142.0/24", "server": "203.0.113.9:8443"}
],
"failover": {"policy": "prefer", "server": "", "switch_after_sec": 15, "failback_after_sec": 120,
             "reason": "primary_down", "since": "2026-11-01T08:01:02Z", "failback_in_sec": 80}
```

- `state`: `active` (the active server, tunnel working), `connecting` (active, not working yet), `standby` (answered its last probe), `unreachable` (didn't), `unknown` (not probed yet), `off` (`enabled: false`).
- `role`: `primary` for the first in the list, else `backup`.
- `last_ok`: when the tunnel last worked (active) or the last probe was answered (others); omitted if never.
- `failover.reason`: why the active server isn't the primary: `""` (it is), `primary_down`, `manual`, `policy` (`always`).
- `failback_in_sec`: present while counting down to a failback.
- `history`: the last 20 switches since the agent started, newest first:
  `[{"at","from","to","reason":"failover"|"failback"|"manual"|"policy"|"list","detail"}]`
  (`detail`, for failovers: why the old server stopped working, e.g. "no answer on any path for 5 s").

In `GET /api/bypass`, `wans[].default` marks the WAN carrying the router's
default route now, i.e. where "Use the internet directly" goes.

`status.server` and `tunnel_ip` describe the active server. Apps word it as
"Using vps (homebox is unreachable)" / "Back to homebox in 1:20".

The optional split "VPS for internet, home for tailnet" (two tunnels at once)
is not implemented: mqvpn runs one client per TUN and the hooks route the
LAN to one table. It would need a second mqvpn client, a second table and
per-destination rules; noted as future work.

## Bypass rules (client role)

On the travel router, LAN traffic can skip the tunnel by rule
(`client.bypass`, like OpenMPTCProuter's OMR-ByPass):

```json
{"id": "b3f9", "kind": "domain", "match": "bank.example", "target": "direct", "enabled": true, "note": "Bank"}
{"id": "c1d2", "kind": "device", "match": "aa:bb:cc:dd:ee:ff", "ip": "192.168.8.23", "host": "work-laptop", "target": "tunnel", "enabled": true, "note": "Work laptop"}
{"id": "e5f6", "kind": "ip", "match": "203.0.113.0/24", "target": "wan", "wan": "wwan", "enabled": true}
{"id": "a7b8", "kind": "port", "match": "udp/4500", "target": "direct", "enabled": true, "note": "Wi-Fi calling"}
```

- `kind`/`match`:
  - `domain`: a domain and all its subdomains (`netflix.com`; a leading `*.` is dropped);
  - `device`: a LAN device's MAC (from `GET /api/lan-clients`), with `ip` used only when `match` is empty. Optional `host`, the device's DHCP name: the rule then also covers every MAC the router currently leases under that name (checked every 30 s), so a laptop that comes back with a new private Wi-Fi address still matches. A MAC another device rule names is never taken over;
  - `ip`: an IPv4/IPv6 address or CIDR;
  - `port`: `tcp/<port>`, `udp/<port>` or a range `udp/5060-5061`.
- `target`:
  - `direct`: the main table, i.e. GL's normal multi-WAN ("Use the internet directly");
  - `wan` + `wan` (an OpenWrt network: `wan`, `wwan`, `tethering`, `modem_…`): only that WAN ("Use only <WAN>");
  - `tunnel`: through tailbond even if another rule says otherwise ("Always use tailbond").
- `id`: 1–16 of `[a-z0-9]`, assigned by the agent; `note` ≤ 64 characters. At most 64 rules.

**How it's applied** (OpenWrt `bypass-apply` hook):
- An nftables table `inet tailbond` (fw4) or iptables `mangle` chain `tailbond_bypass` plus ipsets (fw3).
- It marks LAN-originated packets in prerouting with an fwmark under `client.bypass_mark_mask` (default `0x07000000`; Tailscale uses `0x00ff0000`, mwan3 `0x3f00`).
- The mark is saved to conntrack, so a flow keeps its route even after a DNS answer expires.
- Device rules are evaluated first, then port, IP and domain rules, each in list order. The first match wins. A device rule with `target: tunnel` therefore beats a domain bypass.
- Domains: dnsmasq adds every address it resolves for the domain to the rule's set, whichever upstream answered:
  - `nftset=/d/4#inet#tailbond#bp_<id>_4,6#…` on fw4;
  - `ipset=/d/tb_<id>_4,tb_<id>_6` on fw3;
  - written to dnsmasq's conf-dir.
  For `direct`/`wan` rules it also sends that domain's queries to the WAN's own DNS servers (`server=/d/<ip>`), so the domain resolves while the tunnel is down and CDNs answer for where the router is.
- Both directives need **dnsmasq-full**. Without it (`support.domains = "none"`), domain rules are kept but inactive, the panel says why, and device/IP/port rules still work.
- Limits:
  - matching is on the addresses the router's dnsmasq hands out;
  - a device using its own DNS-over-HTTPS/DoT resolver, or hard-coded IPs, isn't matched;
  - ECH hides nothing here (matching happens at DNS, not SNI) but means there's no SNI fallback;
  - CDNs share IPs between sites, so a bypassed domain can take other sites on the same addresses with it.

**Rule priorities** (`ip rule`, base `P = client.rule_priority`, default 1077; v4 and v6):

| Priority | Rule | Purpose |
|---|---|---|
| P-7 (1070) | `fwmark T/mask lookup main suppress_prefixlength 0`, same for each `W<i>` | marked traffic (and its marked replies) still reaches the LAN and other local routes |
| P-6 (1071) | `fwmark T/mask lookup 77` | "Always use tailbond": the tunnel, or the kill switch's blackhole while it's down |
| P-5 (1072) | `fwmark D/mask lookup main` | "Use the internet directly" |
| P-4 (1073) | `fwmark W<i>/mask lookup 80+i` | "Use only <WAN>": that WAN's default route (table refreshed on WAN changes) |
| P-3 (1074) | `iif <lan> lookup 78` | captive-portal sign-in mode |
| P-2 (1075) | `to <server> lookup main` | the active server's address never enters the tunnel |
| P-1 (1076) | `iif <lan> lookup main suppress_prefixlength 0` | LAN-to-LAN and other non-default routes |
| P (1077) | `iif <lan> lookup 77` | everything else: the tunnel (or the kill switch's blackhole) |

Marks: `T` = `0x02000000`, `D` = `0x01000000`, `W<i>` = `0x03000000 + i<<24` (up to 5 WANs), each within the mask.

Replies carry their connection's mark too (restored from conntrack on every
interface), and the hook turns on `net.ipv4.conf.all.src_valid_mark` (as
wg-quick does; the previous value comes back on `clear`). The kernel checks
a reply's source by looking up routes as if it came from the LAN device; that
lookup would otherwise land in table 77, which is a blackhole while the kill
switch is armed, so with `rp_filter` on every reply to a bypassed device would
be dropped exactly when bypassing matters.

Interplay:
- **Kill switch**: bypassed traffic is intentional, so `direct`/`wan` rules keep working while the tunnel is down. Everything else stays blocked, and so does `tunnel`-target traffic.
- **Sign-in mode** (captive portals) applies after the bypass rules: a "tunnel" device keeps using the tunnel during sign-in, while every other LAN device reaches the portal WAN.
- **GL Login Mode** is the same sign-in mode.
- **Tailscale on the router**: its rules (priority 5210+) come after these.
- **Idempotence**: the hook rebuilds the nft table / chain and the dnsmasq file from the rule file on every run. It restarts dnsmasq only when the file changed, and replaces ip rules instead of adding them. `TB_BYPASS_ACTION=clear` removes all of it.

**UCI mirror.** `config bypass` sections in `/etc/config/tailbond`
(`option kind/match/ip/target/wan/note`, `option enabled`) replace
`client.bypass` when present (like `config path`); otherwise the rules saved
from the panel are kept across re-renders. UCI `client.bypass_mark_mask` and
`client.bypass_local_dns` (default `1`: the per-domain WAN DNS above) render
to the JSON keys of the same name.

**Counters.** Each rule has an nft named counter (`c_<id>`) or an iptables
rule counter (comment `tb:<id>`). It counts the first packet of every
connection the rule matched; later packets follow the connection's saved
mark. The panel shows "N connections".

**Android**: per-app exclusion already exists (`addDisallowedApplication`).
Per-website bypass, on Android 13+ (API 33):
- the app resolves each website through the tunnel's DNS when the tunnel comes up, and every 10 minutes while it's up;
- it adds the resulting addresses to the VPN with `VpnService.Builder.excludeRoute`, which needs a tunnel rebuild (a short reconnect) when the addresses change;
- older Android versions show the limit instead.
See [android.md](android.md#skipping-the-tunnel).

## Updates

Like most open-source projects: tagged **GitHub releases** with every
artifact and a `SHA256SUMS` file (plus GitHub build-provenance attestations,
which anyone can check with `gh attestation verify`; the agent doesn't need
them). Each install checks the GitHub "latest release" API about once a day,
shows "Update available", and updates with the platform's normal mechanism.
Users' guide: [updates.md](updates.md).

### Release assets (CI job `release`, on `v*` tags)

`tailbond-linux-x86_64.tar.gz`, `tailbond-linux-aarch64.tar.gz` (static
bundle with `install.sh`), `tailbond_<v>-r1_aarch64_cortex-a53.ipk`,
`tailbond_<v>-r1_aarch64_cortex-a53_neon-vfpv4.ipk`,
`tailbond-server-<v>-1.fc44.x86_64.rpm`, `tailbond-<v>.apk` (release-signed),
`tailbond-macos-arm64.tar.gz` (Mac bundle with `install.sh`), `tailbond.rb`
(Homebrew formula with that tarball's checksum),
`tailbondd-<tag>-linux-<arch>.tar.gz`, and `SHA256SUMS` (`sha256sum` format
over all of them). A `-pre` tag (`v1.2.3-beta.1`) is a GitHub pre-release,
which the "latest release" API never returns.

### The check

`GET <api_url>/repos/<repo>/releases/latest` (`api_url` default
`https://api.github.com`; `repo` default `kaovilai/tailbond`, also accepted as
`https://github.com/<owner>/<name>`). From the answer: `tag_name` (version:
semver, leading `v` dropped), `html_url` (changelog), `assets[].name` /
`browser_download_url`. A newer version than the running one is `available`.
A build whose own version isn't semver (`dev`, `-dirty`, a git hash) shows
what's available but never installs by itself.

**Verification**: the asset and `SHA256SUMS` are downloaded over HTTPS (http
only when `api_url` itself is http, for tests); the asset's SHA-256 must match
its line in `SHA256SUMS`, else nothing is installed.

### Install methods (`update.method` in the status)

| method | detected by | one-click / automatic install |
|---|---|---|
| `static` | `/etc/tailbond/install.env` (written by `install.sh`) | yes: download `tailbond-linux-<arch>.tar.gz` + check, unpack under `<state_dir>/update/`, run the new bundle's `install.sh --update --bundle-dir <dir>` detached (`systemd-run --unit tailbond-update`): it keeps `<bin>.prev`, replaces the binaries and hooks, restarts the service, and if the service isn't up on the new version within 60 s puts `.prev` back and restarts again |
| `opkg` | OpenWrt (`/etc/openwrt_release` + `opkg`) | yes: download the ipk whose arch `opkg print-architecture` lists + check, keep it in `/etc/tailbond/pkg/`, run `/usr/lib/tailbond/update-ipk` detached (`opkg install`, health check, previous ipk back on failure) |
| `dnf` | `tailbondd` owned by an RPM | no: the status says so and the UI shows `sudo dnf upgrade tailbond-server` (COPR, `dnf-automatic`) |
| `static` (macOS) | `/usr/local/etc/tailbond/install.env` (written by `packaging/macos/install.sh`) | yes: `tailbond-macos-<arch>.tar.gz`, its `install.sh --update` run in a new session (it restarts the LaunchDaemon), same `.prev` rollback |
| `brew` | macOS, `tailbondd` resolves into Homebrew's Cellar | no: the UI shows the commands (the release's `tailbond.rb` into a local tap, `brew reinstall`, `sudo brew services start tailbond`) |
| `none` | anything else (a dev build, a copied binary) | no: shown only |

Both installers write `<state_dir>/update-result.json`
(`{"result": "ok"|"rolled_back"|"failed", "version", "from", "at", "detail"}`;
OpenWrt `state_dir` is `/etc/tailbond/state`), which the agent reports after
its restart. `detail` `reinstalled_after_firmware_upgrade`: GL firmware
upgrade, see below.

**GL firmware upgrades**: the installed ipk is kept in `/etc/tailbond/pkg/`
(kept by `/lib/upgrade/keep.d/tailbond`); a line in the kept `/etc/rc.local`
runs `/etc/tailbond/pkg/reinstall` at boot, which reinstalls it when the
firmware upgrade removed the package (else downloads that version's ipk from
the GitHub release, checked against its `SHA256SUMS`).

**Android**: the app makes the same check (repo built in) and shows "Update
available", which opens the release APK in the browser; Android installs it
over the app (same signing key). Automatic updates: Obtainium.

### Config (`update`, all optional)

```json
"update": {"policy": "notify", "repo": "kaovilai/tailbond", "check_every_hours": 24, "api_url": ""}
```

- `policy`: `notify` (default: check and show; install on request), `auto`
  (install when found, between 03:00 and 05:00 local time, methods `static`
  and `opkg`; a version that rolled back isn't installed automatically
  again), `off` (no checks).
- Older keys (`channel`, `window`, `pin`, `source`, `mirror`, …) are ignored.

### Status (`/api/status`, additive)

```json
"update": {"current": "0.1.0", "available": "0.2.0",
           "notes_url": "https://github.com/kaovilai/tailbond/releases/tag/v0.2.0",
           "policy": "notify", "repo": "kaovilai/tailbond", "method": "static",
           "state": "idle", "last_check": "…", "last_result": "ok",
           "last_result_version": "0.1.0", "last_result_at": "…", "detail": "",
           "error": "", "outdated": [{"name": "slate7", "kind": "router", "version": "0.1.0"}]}
```

- `state`: `idle` | `checking` | `downloading` | `installing` | `failed` |
  `rolled_back` | `off`.
- `auto_window`: when `auto` installs (`"03:00-05:00"`); `dnf_automatic`
  (method `dnf`): `install` | `download` (a dnf-automatic timer with Fedora's
  default `apply_updates = no`) | `off`; it covers every package.
- `outdated` (server): registered clients whose last reported version is
  older than this server's. Clients send `X-Tailbond-Version` and
  `X-Tailbond-Kind` (`router` | `android`) with their API requests
  (`GET /api/peers`, `/api/pairing`); the server shows them as
  `clients[].version` / `clients[].kind`.

### HTTP

| Method | Path | Body / result |
|---|---|---|
| GET | `/api/update` | the `update` status object |
| POST | `/api/update/check` | check now |
| POST | `/api/update/install` | install `available` now (methods `static`, `opkg`; else 400 with what to do) |
| POST | `/api/update/policy` | `{"policy", "repo"}` (any subset) |
| GET | `/api/pairing` | server, client token: this client's current pairing URI (same credential), so a client learns a new public address after the server moved |


## Backup and restore

An encrypted backup file `.tbbackup` (format 1):

```
TBBACKUP1\n
{"v":1,"kdf":"scrypt","n":32768,"r":8,"p":1,"salt":"<b64, 16 bytes>","nonce":"<b64, 12 bytes>",
 "role":"server","platform":"fedora|openwrt|linux|android","host":"homebox","agent_version":"0.2.0",
 "created":"2026-11-02T10:00:00Z"}\n
<base64 of AES-256-GCM(key = scrypt(passphrase, salt, n, r, p, 32), nonce, plaintext, aad = the header line bytes)>\n
```

- AES-GCM, not XChaCha20-Poly1305: Android has no XChaCha in the platform,
  and the per-file random salt gives every file its own key, so nonce reuse
  can't happen. The header is authenticated (AAD).
- A wrong passphrase and a corrupted file both fail the GCM check
  ("wrong passphrase or damaged file").

Plaintext: JSON `{"format":1,"config":<tailbond.json>,"files":{"<name>":"<b64>"},"uci":"<uci export tailbond>"}`.
- `config` is the whole config: users and keys, pairings and servers,
  approvers, bypass rules, modes, paths, update policy, UI password.
- `files`: the state files that matter: `server.crt`, `server.key`
  (self-signed TLS, so clients' pins keep working) and `global.key`.
  `ts-cert.pem`/`ts-key.pem` are left out; Tailscale reissues them.
- `uci`: `/etc/config/tailbond` on OpenWrt.
- Android's backup is the app's own JSON (profiles/servers, settings,
  excluded apps, websites), same container, `"platform":"android"`.

There is never an unencrypted export of secrets. `tailbondd backup --plain`
/ `GET /api/backup/summary` give a JSON summary without keys or passwords,
for inspection.

**Restore**:
- A backup is restored onto the same role. Another machine is fine: a new
  VPS, a new home server.
- Restore first saves the current config as an encrypted rollback copy
  (`<state_dir>/backups/before-restore-<time>.tbbackup`, same passphrase),
  then replaces the config and the files and restarts the agent.
- Clients keep working: same users and keys, and the same pinned certificate
  when it was self-signed.
- If the public address changed: set `server.public_addr`. Clients then fetch
  their refreshed pairing (`GET /api/pairing`, same credential) from the
  server's Tailscale name, every 6 h, and right away when every server has
  been unreachable for 2 minutes. Phones re-ask over Tailscale (the server
  re-issues the same credential, `ts_node`), or scan the client's link again.

**Scheduled backups** (`backup`, additive, server and router):
`{"passphrase": "", "every_hours": 24, "keep": 7, "push": true}`.
- With a passphrase set, the agent writes `<state_dir>/backups/<host>-<time>.tbbackup`
  and keeps the newest `keep`.
- A client with `push` sends its newest backup to its server
  (`POST /api/backups/push`, client token, ≤ 1 MiB, opaque). The server
  keeps the newest 3 per client in `<state_dir>/client-backups/<name>/`;
  a server's own backup carries them (`files["client-backups/<name>/<file>"]`,
  still encrypted with the client's passphrase).

| Method | Path | Body / result |
|---|---|---|
| POST | `/api/backup` | `{"passphrase"}` (≥ 10 characters) → the `.tbbackup` file (`application/octet-stream`, attachment) |
| GET | `/api/backup/summary` | what a backup would contain, no secrets |
| POST | `/api/restore/preview` | `{"file": "<base64>", "passphrase"}` → `{"role","host","created","agent_version","changes":[{"what","from","to"}],"warnings":[…]}` (role mismatch is an error) |
| POST | `/api/restore` | same body plus `"confirm": true` → restores, keeps the rollback copy, restarts |
| GET | `/api/backups` | admin: the local scheduled backups and the clients' pushed ones `[{"name","client","size","created"}]` |
| GET | `/api/backups/{name}[?client=<client>]` | admin: download one (still encrypted); `client` for a client's pushed backup (`name` is a plain file name in the list) |
| POST | `/api/backups/push` | client token: a client's own encrypted backup |
| POST | `/api/backup/settings` | `{"passphrase","every_hours","keep","push"}`: an omitted passphrase keeps the current one, `""` turns scheduled backups off. `GET /api/settings` reports `backup: {"passphrase_set","every_hours","keep","push","last","last_push","last_error"}` |

Request bodies for these endpoints may be up to 2 MiB.

CLI:
- `tailbondd backup --out FILE [--passphrase-file F]`: the passphrase comes from the file, from `TB_BACKUP_PASSPHRASE`, or a prompt.
- `tailbondd backup --plain` prints the summary.
- `tailbondd restore FILE [--yes] [--passphrase-file F]`: run it with the service stopped, or restart the service right after (the running agent would otherwise keep its settings in memory).

## macOS (darwin) notes

The client role on a Mac ([macos.md](macos.md)) is the same agent with a
few platform pieces behind build tags:

- **Interfaces** (`internal/netif`, `darwin.go`): candidates are the
  interfaces `networksetup -listallhardwareports` names, kind and label from
  the hardware port (Wi-Fi; Ethernet, Thunderbolt Ethernet and USB-C/USB LAN
  adapters by their port name; "iPhone USB" → kind `usb`, label
  *iPhone (USB)*; Bluetooth PAN → *Bluetooth tethering*); the Thunderbolt
  bridge, `utun`, `awdl`, `llw`, `bridge`, `ap`, `feth` and the like never
  are. Link state from `ifconfig` (`status: active`), byte counters from
  `NET_RT_IFLIST2` (64-bit). A `client.paths[].iface` naming an interface
  discovery skips is used as named (Linux too).
- **Tunnel**: mqvpn asks for `utun` and the kernel picks the unit; the agent
  takes the real name from mqvpn's `TUN utunN configured` line (`TB_TUN`).
  Routes and DNS are mqvpn's (`manage_routes`, default on; `no_dns`), the
  kill switch is mqvpn's pf anchor (`client.kill_switch`, off by default; it
  blocks the LAN, so shares don't work with it on).
- **Sockets pinned to a WAN** (captive probes, a sharer's upstream):
  `internal/sockbind`, `SO_BINDTODEVICE` on Linux, `IP_BOUND_IF` /
  `IPV6_BOUND_IF` on macOS.
- **Shares**: the same hub logic (mDNS browse by legacy unicast, the
  server's list, keepalive a second, 3.5 s to drop). The mDNS responder
  (a Mac sharing, or the test sharer) shares port 5353 with mDNSResponder
  through `SO_REUSEPORT`. Point-to-point links are never share LANs.
- **No parent-death signal**: the supervisor records mqvpn's pid in
  `<state_dir>/mqvpn.pid` and stops one a previous agent left running
  before starting another (mqvpn holds a single-instance lock).
- **Defaults**: config `/usr/local/etc/tailbond/tailbond.json`, state
  `/usr/local/var/lib/tailbond`, hooks `/usr/local/lib/tailbond/hooks`,
  mqvpn `/usr/local/bin/mqvpn`, `listen` `127.0.0.1:8077` (this Mac only).
- **Service and updates**: LaunchDaemon `com.tailbond.tailbondd` (root);
  update method `static` with `tailbond-macos-arm64.tar.gz`
  (`/usr/local/etc/tailbond/install.env`), or `brew` for the Homebrew
  formula (shown, not installed by the agent).

## Pair & Share

User guide and design: [pair-and-share.md](pair-and-share.md). A **sharer**
(Android app, iOS tailbond Share, or tailbondd with `client.share`) lends its
cellular uplink to a **hub** (tailbondd client role) on its LAN. Reference
implementation: `agent/internal/share` (Go); Android `share/ShareProtocol.kt`
and iOS `TailbondShareKit` follow it byte for byte
(`agent/internal/share/testdata/vectors.json`, checked by all three).

**Share key.** The server's `share.key` (state dir, random, created once).
Every pairing carries it as `share_key` (additive). Older pairings get it
from `GET /api/shares` with the client token. Holding it is what "paired
with the same server" means. `key_id` = hex of the first 4 bytes of
HMAC-SHA256(key, `"tbs1-keyid"`): public, used to skip other servers' sharers.

**Relay protocol** (UDP, sharer port 47077; integers big-endian):

| Frame | Bytes |
|---|---|
| HELLO (hub →) | `01 01 hub_nonce[16] len hub_id` |
| CHALLENGE (← sharer) | `02 01 hub_nonce[16] sharer_nonce[16] len sharer_id len label` |
| AUTH (hub →) | `03 01 hub_nonce[16] sharer_nonce[16] mac[32]`, mac = HMAC(key, `"tbs1-auth"` ‖ hub_nonce ‖ sharer_nonce ‖ hub_id) |
| WELCOME (← sharer) | `04 01 hub_nonce[16] session[8] cap_left[8] mac[32]`, mac = HMAC(key, `"tbs1-welcome"` ‖ hub_nonce ‖ sharer_nonce ‖ session ‖ cap_left) |
| REFUSE (← sharer) | `05 01 hub_nonce[16] reason mac[32]`, mac = HMAC(key, `"tbs1-refuse"` ‖ hub_nonce ‖ sharer_nonce ‖ reason) |
| DATA (both ways) | `10 session[8] payload` (one mqvpn/QUIC datagram) |
| KEEPALIVE (hub →) | `11 session[8] seq[4]` |
| KA_ACK (← sharer) | `12 session[8] seq[4] up[8] down[8] cap_left[8] battery flags` (battery 255 = unknown; flags bit 0 = charging) |
| BYE (both ways) | `13 session[8] reason` |

Reasons: 1 not allowed, 2 data cap, 3 battery, 4 paused, 5 no cellular,
6 replaced (a newer session from the same hub), 7 idle. `cap_left` is bytes,
all ones = no cap. Rules: a wrong MAC, an unknown handshake or a stranger gets
no answer at all; REFUSE only after a valid AUTH. A session is bound to the
hub's address; DATA/KEEPALIVE from any other address are dropped. One session
per hub id (a new one replaces it); at most 8 hubs and 64 pending handshakes;
handshakes expire after 10 s, sessions after 30 s without traffic. The sharer
forwards only to its own pairing's server (`server`), over cellular only. The
hub keeps a keepalive a second and treats 3.5 s without KA_ACK as lost; it
retries a REFUSE after 30 s. Counters (`up`/`down`) are payload bytes; the
sharer's cap counts both directions.

**Discovery.** mDNS `_tailbond-share._udp.local`: instance = label, SRV port,
TXT `v=1`, `id=<sharer id>`, `k=<key_id>`, `n=<label>`. The hub browses with a
legacy-unicast query on each LAN interface (not `tb0`, not the shared
uplink). Server registry, client token required (also over the tunnel):

- `POST /api/shares` `{"id","label","key_id","lan":["192.168.8.23"],"port":47077,"kind":"android|ios|linux"}`
  → `{"ok":true}`. Entries expire after 3 minutes; sharers refresh every
  minute while connected. Private/link-local IPv4 only, up to 8 addresses,
  64 sharers.
- `GET /api/shares` → `{"key","key_id","sharers":[{"id","label","key_id","client","lan","port","kind","seen"}]}`.
  On the client role it returns this hub's view: `{"shares","note","sharing"}`.
- `POST /api/share` (client role): `{"use_shares": bool}`, `{"use_local": bool}` and/or
  `{"share": {"enabled","iface","port","label","cap_mb","allowed_hubs"}}`.

**Hub paths.** Each sharer becomes path `sh` + hex(sha256(id))[:6], kind
`shared-cellular`, label `"<label> (shared)"` (phones already send `"Mobile data from <name>"`, kept as is); default priority secondary
(like cellular), settings and data caps keyed by that name, usage from the
link's counters. It is in the path set only while its session is open. mqvpn
runs it as a relay path, `Path = shNNNNNN@127.0.0.1:PORT`
(`patches/mqvpn/0002-relay-paths.patch`): the slot isn't pinned to an
interface, its datagrams go to the hub's link on that port, and replies from
there are delivered as the server's.

**Status (client role, additive):** `shares[]`
`{id,label,path,addr,via,state,reason,up_bytes,down_bytes,cap_left_bytes,battery,charging}`
(state: connecting, ready, refused, unreachable, lost), `use_shares`,
`use_local` (the machine's own connections are bonded; "Use this Mac's
connections"), `platform` (`linux` | `darwin`), `share_note` (client isolation: *"This Wi-Fi blocks devices from talking to
each other; sharing isn't available here."*), and `sharing`
`{enabled,iface,label,port,state,error,used_bytes,cap_mb,hubs[]}` for this
machine's own share.

## Phone hub

User guide and design: [phone-hub.md](phone-hub.md). A phone running
tailbond lends its whole bonded tunnel to a laptop through a proxy.

- **The proxy:** SOCKS5 + HTTP CONNECT on TCP 1888, with the reference
  implementation in `agent/internal/proxy`, ported to Kotlin
  (`ProxyServer.kt`) and Swift (`TailbondShareKit`).
  - User `tailbond`. The password is the first 16 characters of
    `base32(HMAC-SHA256(share_key, "tbp1-proxy"))`, lower case
    (`testdata/vectors.json`).
  - Only LAN-arriving connections from private or link-local sources are
    served.
  - Upstream sockets are on the phone's VPN network (Android), so the
    traffic goes into the tunnel.
- **Discovery:** mDNS `_tailbond-proxy._tcp`, and the server's sharer
  registry with `kind: "proxy"`.
- **tailbondd (client role):**
  - `GET/POST /api/phone-hub` and `GET /proxy.pac`, with
    `phone_hub` in `/api/status`.
  - A loopback forwarder on 127.0.0.1:1889 relays to the hub with the
    password.
  - On macOS the system SOCKS proxy (or the PAC URL) of the default route's
    network service points at it: `internal/sysproxy`, with a restore record
    in `state_dir/phone-hub-proxy.json` put back on stop and at the next
    start.
  - `tailbondd restore-proxy` does the same for `uninstall.sh`.
- **Android as a Pair & Share hub:** relay paths through
  `mqvpn_android_reactor_add_relay_path` (patch 0004). The Android native
  libraries are now built from the patched copy of mqvpn
  (`android/scripts/build-native.sh`).

## Captive portals

Client role (travel router). Each usable WAN is checked for a sign-in page;
one behind a portal is kept out of mqvpn's paths and out of failover's
usable tiers until it's open, so it neither stalls bonding nor makes
failover flap; the other WANs keep the tunnel up.

- **Check** (`internal/captive`): an HTTP GET bound to the WAN's device
  (`SO_BINDTODEVICE`), resolved through the WAN's own DHCP DNS (bound too).
  204 / the expected body → `open`; a redirect, another 200 or 511 →
  `portal` (`portal_url` from `Location`, meta refresh or JS, else the probe
  URL); no answer → `offline` (not held back). On link up, hotplug (SIGHUP),
  a failed mqvpn path, every 60 s while `portal`/`offline`, every 5 s in
  sign-in mode, every 10 minutes otherwise.
- **GL repeater** (`internal/glrepeater`, one-file adapter): when `ubus call
  repeater status` reports GL's portal handling on, GL is the source of
  truth for the `wwan` network (no probe of ours there; `captive_source:
  glinet`), and its Login Mode (`bare_mode`) opens our sign-in mode for wwan,
  even with `captive_portal_detect` off. See [glinet.md](glinet.md#captive-portals-and-gls-login-mode).
- **Sign-in mode**: `POST /api/captive/signin` (or GL's Login Mode) runs the
  `portal-open` hook: LAN devices reach that WAN directly, past the tunnel
  and the kill switch, with DNS through the WAN's own servers. It ends with
  `portal-close` when the WAN checks open (or GL leaves Login Mode), after
  10 minutes, on `DELETE /api/captive/signin`, or when the agent stops; the
  WAN then rejoins bonding by itself.
- **Android**: mqvpn's network monitor only uses validated networks; the
  app shows "Wi-Fi needs sign-in" for a Wi-Fi with
  `NET_CAPABILITY_CAPTIVE_PORTAL` (or unvalidated for 10 s) and opens
  Android's own sign-in ([android.md](android.md#wi-fi-sign-in-pages-captive-portals)).

## Setup health

`GET /api/health` (both roles, admin):

```json
{"all_good": false, "direct": 2, "relayed": 1, "relayed_peers": ["pixel-8"],
 "netcheck": {"udp": true, "ipv4": true, "ipv6": false, "global_v4": "203.0.113.7:41641",
              "mapping_varies_by_dest_ip": true, "upnp": false, "pmp": false, "pcp": false, "preferred_derp": 4},
 "checks": [{"id": "tailscale_direct", "status": "warn", "title": "Tailscale: 2 machines direct, 1 through a relay (slower)",
             "detail": "...", "fix": "...", "fix_url": "https://tailscale.com/docs/reference/connection-types"}]}
```

`status` is `ok` | `warn` | `bad` | `unknown`; `all_good` means no `warn`/`bad`.
Check ids: `tailscale` (off), `tailscale_direct` (online peers with a
`CurAddr` are direct, with only a `Relay` relayed; the client role counts
only its home server), `netcheck` (`tailscale netcheck --format=json`,
cached 10 minutes: UDP blocked, hard NAT = `MappingVariesByDestIP`, UPnP /
NAT-PMP / PCP), and on the server `route` (the tunnel subnet approved),
`cgnat` (this machine's internet-facing address in `100.64.0.0/10` =
carrier-grade NAT; private = behind a router, with the public address from
netcheck), `port` (UDP `server.port` confirmed by a client last seen from a
public address; `unknown` until then, since it can't be checked from
inside). Fix links point to Tailscale's docs ([setup tips](setup-tips.md)).

Mode table details: `Reorder on (Profile = fiber_lte)` is rendered as `[Reorder] Enabled = true` plus a `[ReorderRule]` for UDP 443 with `fiber_lte` and one for UDP 53 with `default_udp`. The server always enables `[Hybrid]`, `[Reorder]` and `AllowClientScheduler`, so each client's mode takes effect. Failover keeps only the best usable priority tier in the client's `Path` list. The bonding modes use primary and secondary paths, and add backup paths only when no primary or secondary path is usable. Paths over their monthly `data_cap_mb` drop out.
