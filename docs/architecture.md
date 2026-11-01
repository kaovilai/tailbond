# tailbond: architecture and cross-component contract

tailbond = Speedify-style WAN bonding to **your own home server**, with your tailnet reachable through it.
Data plane: [mqvpn](https://github.com/mp0rta/mqvpn) (MASQUE CONNECT-IP over Multipath QUIC), pinned as `third_party/mqvpn` (v0.19.0).
tailbond adds the management agent, web UI, Tailscale integration, pairing, packaging, and the Android app.

Every component below MUST follow this document. Change it here first.

## Components

| Component | Path | Runs on | Role |
|---|---|---|---|
| `tailbondd` agent + web UI | `agent/` (Go, UI embedded) | Fedora (x86_64/aarch64), GL.iNet (aarch64 musl) | supervises mqvpn, serves dashboard + API, Tailscale glue |
| mqvpn patches | `patches/mqvpn/*.patch` | all | client-side control API (see below) |
| Fedora package | `packaging/fedora/` | Fedora 43+ | RPM: `tailbond-server` (mqvpn + tailbondd + units + firewalld service) |
| OpenWrt / GL.iNet package | `packaging/openwrt/` | Flint 3 (GL-BE9300), Slate 7 (GL-BE3600), any aarch64_cortex-a53 OpenWrt | `.ipk` with static musl mqvpn + tailbondd, init.d, UCI, GL admin panel (VPN → tailbond: oui view + `/rpc` Lua object, [glinet.md](glinet.md)) |
| Android app | `android/` | Android 8+ arm64 | fork of mqvpn's Android app, rebranded `dev.tailbond.app` |
| Generic Linux server bundle | `packaging/linux/` (+ Ansible role `packaging/ansible/roles/tailbond/`) | Debian/Ubuntu, Alma/RHEL/Rocky 9–10, Fedora; x86_64 | static mqvpn + tailbondd, `install.sh` (ports, subnet, firewall, systemd memory limits), for a VPS server ([vps.md](vps.md)) |

Roles of `tailbondd` (`--role` / config `role`):
- `server`: home endpoint (Fedora box or Flint 3). Runs `mqvpn --mode server`.
- `client`: travel router (Slate 7). Runs `mqvpn --mode client` over all WANs; routes LAN through the tunnel.

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

## mqvpn client control API (patch)

Upstream mqvpn only serves the control API in server mode. `patches/mqvpn/0001-client-control-api.patch` adds it to client mode on Linux: `[Control] Listen` / `--control-port` in client mode serve `get_status` and `get_stats` with the **same JSON shape as server mode**, as one client object whose path entries additionally carry `"iface"` (the bound interface name). Keep the patch upstreamable (small, confined to `src/platform/linux/` + `src/platform/posix/`).

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
| GET/POST/DELETE | `/api/join` | client, additive: ask a server to pair (`{"server":"homebox"\|"homebox.tail…ts.net"\|"100.x"}`), its state `{"state":"idle\|asking\|waiting\|approved\|declined\|expired\|error","server","server_name","id","name","error"}`, cancel. Additive: `"add": true` adds the approved server to the [server list](#multiple-servers-and-failover) as a backup instead of replacing the primary; `error` may then be `subnet_in_use` or `exists`, with `conflict` (the server already in the list) and `subnet` |
| DELETE | `/api/pair` | client, additive: forget the pairing, every server in the list (tunnel stops, LAN back on the WANs) |
| GET | `/api/peers` | client, additive: the server's list (below), fetched through the tunnel with this client's token, cached 20 s |
| POST/DELETE | `/api/captive/signin` | client, additive: `{"name": <network or device>}` sign-in mode on a WAN behind a portal (409 otherwise) / off ([captive portals](#captive-portals)) |
| POST | `/api/captive/check` | client, additive: `{"name"}` re-check now (`""` = all) |
| GET | `/api/health` | additive: [setup health](#setup-health) |
| GET | `/tailbond-glinet.ipk` | additive: the GL router package, from `glinet_ipk` (default `/usr/share/tailbond/tailbond-glinet.ipk`) or rebuilt from opkg's records of the installed package |
| GET | `/get` | server, additive, **unauthenticated**: the [phone page](#phone-page) (HTML), also served on port 80 |
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

Server role: a page for phones that run Tailscale but not tailbond yet,
served on **port 80 of the Tailscale IPs** (config `landing_port`, default
80, `-1` = off; `landing_public_port` is the port tailnet devices use when a
firewall redirect sits in between: OpenWrt binds 8078 and redirects tailnet
tcp/80 to it). Typing the server's MagicDNS short name in a phone's browser
opens it. It serves exactly `GET /` (HTML: the APK download with what
Android's install prompts look like, and a link that opens the app asking
this server), `GET /tailbond.apk` and `GET /icon.svg`; every other path
redirects to `/`, whatever `ui_password` says. Binding failures (port in
use) are logged once and the page stays at `/get` on the dashboard port.

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
- `agent`: `CGO_ENABLED=0 go build` for linux/amd64 and linux/arm64.
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
- `client.manage_routes` (default `true`; set it to `false` on OpenWrt), `client.no_dns` (don't let mqvpn change the system resolver), `client.disconnected` (persisted connect toggle), `client.paths[].kind` (override the kind heuristic);
- `disable_tailscale`: skip every `tailscale` CLI call; `control_addr`; `log_level`;
- `android_apk`: APK offered as `GET /tailbond.apk` (default `/usr/share/tailbond/tailbond.apk`; nothing is offered when the file is missing);
- `glinet_ipk`: router package offered as `GET /tailbond-glinet.ipk`;
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

Every tailbond install can update itself from a **signed manifest**. A home
server also acts as an **update mirror**, so routers and phones don't need
to reach GitHub (hotel Wi-Fi, captive portals, no GitHub remote at all).
Details for users: [updates.md](updates.md).

### Manifest and signature

A channel (`stable`, `beta`) is one file `<channel>.json` plus
`<channel>.json.sig` under the update source (`update.source`, a base URL;
default set at build time, e.g.
`https://github.com/OWNER/REPO/releases/download/channels`; CI re-uploads
both files to that rolling release on every tag). Format 1:

```json
{
  "format": 1,
  "channel": "stable",
  "version": "0.2.0",
  "released": "2026-11-02T10:00:00Z",
  "protocol": 2,
  "requires_server_protocol": 1,
  "security": false,
  "notes_url": "https://github.com/OWNER/REPO/releases/tag/v0.2.0",
  "artifacts": [
    {"name": "tailbond-linux-x86_64.tar.gz", "kind": "linux", "arch": "x86_64",
     "url": "https://…/v0.2.0/tailbond-linux-x86_64.tar.gz", "sha256": "…", "size": 9123456},
    {"name": "tailbond_0.2.0-r1_aarch64_cortex-a53.ipk", "kind": "ipk", "arch": "aarch64_cortex-a53", "url": "…", "sha256": "…", "size": 0},
    {"name": "tailbond_0.2.0-r1_aarch64_cortex-a53_neon-vfpv4.ipk", "kind": "ipk", "arch": "aarch64_cortex-a53_neon-vfpv4", "url": "…", "sha256": "…", "size": 0},
    {"name": "tailbond-server-0.2.0-1.fc44.x86_64.rpm", "kind": "rpm", "arch": "x86_64", "url": "…", "sha256": "…", "size": 0},
    {"name": "tailbond-0.2.0.apk", "kind": "apk", "arch": "arm64-v8a", "url": "…", "sha256": "…", "size": 0,
     "version_code": 20000}
  ]
}
```

- `version`: semver (`MAJOR.MINOR.PATCH[-pre]`); a leading `v` and an
  OpenWrt `-rN` revision are ignored for ordering. A build whose own version
  isn't semver (`dev`, a git hash, `-dirty`) never installs anything by itself.
- `protocol`: the agent protocol this release speaks (`update.Protocol`;
  `GET /api/hello` and `/api/status` report it as `protocol`).
  `requires_server_protocol`: a **client** (router, phone) installs this
  release only if its active server reports at least this protocol. Servers
  install first; clients follow.
- `security: true`: install even while traffic would otherwise defer it
  (the maintenance window still applies to servers).
- `linux` tarballs contain `tailbondd`, `mqvpn` and `hooks/`, all static.

`<channel>.json.sig` is JSON: `{"key_id": "<hex of the first 8 bytes of
SHA-256(public key)>", "sig": "<base64 ed25519 signature of the exact bytes
of <channel>.json>"}`. Plain ed25519 (RFC 8032), no prehash. Trusted public
keys (raw 32 bytes, base64) are compiled in at build time
(`-X …/update.BuildKeys=<key>[,<key>]`, CI variable `TB_UPDATE_PUBKEY`;
Android `BuildConfig.UPDATE_KEYS`) plus `update.trusted_keys` in the config
(self-hosted builds, tests). With no trusted key, updates are off
("this build has no update key"). Keys: `scripts/release/gen-update-key.sh`.
Every artifact is checked against `sha256` and `size` before use.

### Config (`update`, additive, all optional)

```json
"update": {"policy": "auto", "channel": "stable", "source": "", "window": "03:00-05:00",
           "check_every_hours": 6, "pin": "", "defer_kbps": 300,
           "mirror": true, "mirror_max_mb": 600, "trusted_keys": []}
```

- `policy`: `auto` (default; servers install inside `window`, local time;
  clients any time unless deferred), `notify` (check and show, install only
  on request), `off` (no checks). `pin`: install exactly this version (from
  the manifest or the mirror cache) and nothing newer, for IaC (Ansible
  `tailbond_update_policy: pinned`).
- `defer_kbps`: clients don't install while the tunnel carried more than this
  (tx+rx) on average over the last minute (a call or a stream), unless the
  release is `security`.
- `mirror`, `mirror_max_mb` (server role): mirror the channel, keep the
  current and the previous version (keep-last-2) within the cap.

Status (`/api/status`, additive): `protocol`, and

```json
"update": {"current": "0.1.0", "available": "0.2.0", "policy": "auto", "channel": "stable",
           "state": "idle", "source": "mirror", "last_check": "…", "last_result": "…",
           "notes_url": "…", "security": false, "deferred": "", "window": "03:00-05:00",
           "outdated": [{"name": "slate7", "kind": "router", "version": "0.1.0"}]}
```

- `state`: `idle` | `checking` | `downloading` | `waiting` (window or
  deferral; `deferred` says why) | `installing` | `trial` (new version
  running, health check pending) | `failed` | `rolled_back` | `off`.
- `source`: `mirror` (the home server) | `public` (update.source) | `upload` (an offline upload, server) | `cache` (server: the last good manifest, the public source being unreachable; `error` says why).
- `outdated` (server role): registered clients whose last reported version
  is older than this server's.

Clients report their version to their server: every request a paired
client makes with its API token (`GET /api/peers`, `/updates/…`,
`/api/pairing`) carries `X-Tailbond-Version` and `X-Tailbond-Kind`
(`router` | `android`); the server records them per client
(`clients[].version`, `clients[].kind`, additive).

### HTTP (additive)

| Method | Path | Body / result |
|---|---|---|
| GET | `/api/update` | the `update` status object plus `manifest` (summary of the verified manifest) |
| POST | `/api/update/check` | check now |
| POST | `/api/update/install` | install `available` now (ignores the window and deferral, not compatibility); `{"version"}` optional (a cached version, e.g. to go back) |
| POST | `/api/update/policy` | `{"policy","channel","window","source","pin"}` (any subset) |
| POST | `/api/update/upload?name=<file>` | server, admin: `Content-Type: application/octet-stream`, one file (a manifest, its `.sig`, or an artifact) into the mirror cache, for offline setups; the manifest is verified once both it and its `.sig` are there, artifacts against it |
| GET | `/updates/<channel>.json`, `/updates/<channel>.json.sig` | server mirror: the cached, verified manifest and signature (byte-identical) |
| GET | `/updates/files/<name>` | server mirror: a cached artifact |
| GET | `/api/pairing` | server, client token: this client's current pairing URI (same credential), so a client learns a new public address after the server moved |

`/updates/…` and `GET /api/pairing` are served on the admin listener and the
tunnel address; they need a client API token, an admin session, or a
tailnet / loopback source (the mirror holds only public, signed files).
Clients try the mirror first (`http://<active server tunnel IP>:8077`, then
its Tailscale name) and fall back to `update.source`.

### Applying an update

**Linux (Fedora RPM, generic bundle)**: package-owned files are never
replaced (`rpm -V` stays clean, `ProtectSystem=full` is fine).
- Updates unpack to `<state_dir>/update/versions/<v>/{tailbondd,mqvpn,hooks/}`.
- `<state_dir>/update/state.json` records `{active, prev, trial, failed}`.
- The package's `tailbondd` (the *base*) checks that file first at every
  start. If `active` is newer than itself, it `exec`s that version's binary;
  the running agent then uses the same version's `mqvpn`.
- **Install**: download → verify → unpack → `trial = {version, deadline
  (now + 90 s), boots: 0, clients_before}` → the agent exits with status 75;
  systemd's `Restart=on-failure` (`RestartSec` short) starts the base again.
- **Health check** (the new version, within 90 s): the API serves, mqvpn
  runs (server: control API answers; client: tunnel connected if it was
  before), and a server that had connected clients sees one again. Pass →
  `active` stays and `trial` clears. Fail → the version is added to `failed`,
  `active = prev`, exit 75.
- **The base**: at each start during a trial it increments `boots`. If
  `boots > 3` or the deadline has passed, it rolls back before exec'ing.
  So a version that crashes or hangs on start is rolled back by the base.
  A `failed` version is never installed again automatically.
- An RPM or bundle upgrade whose base is newer than `active` wins: the base
  runs itself.

**OpenWrt / GL.iNet**: the agent picks the ipk whose `arch` is in
`opkg print-architecture`, verifies it, keeps it in `/etc/tailbond/pkg/`
(kept across sysupgrade by keep.d; current + previous), and runs
`/usr/lib/tailbond/update-ipk <new.ipk> <prev.ipk>` detached:
`opkg install`, restart, wait up to 90 s for the API and the tunnel/role,
else reinstall the previous ipk. After a firmware upgrade a kept
`/etc/tailbond/pkg/reinstall`, run from a line in the kept `/etc/rc.local`,
reinstalls the package from that cache (or the home mirror) on first boot
([glinet.md](glinet.md), [updates.md](updates.md)).

**Android**: the app reads the manifest (mirror first), downloads and
verifies the APK, installs it with a `PackageInstaller` session
(`REQUEST_INSTALL_PACKAGES`; Android 12+ `setRequireUserAction(
USER_ACTION_NOT_REQUIRED)`, silent once the app installed itself the first
time). Release APKs are signed with the release key; debug and release
builds can't update each other.

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
  keeps the newest 3 per client in `<state_dir>/client-backups/<name>/`.

| Method | Path | Body / result |
|---|---|---|
| POST | `/api/backup` | `{"passphrase"}` (≥ 10 characters) → the `.tbbackup` file (`application/octet-stream`, attachment) |
| GET | `/api/backup/summary` | what a backup would contain, no secrets |
| POST | `/api/restore/preview` | `{"file": "<base64>", "passphrase"}` → `{"role","host","created","agent_version","changes":[{"what","from","to"}],"warnings":[…]}` (role mismatch is an error) |
| POST | `/api/restore` | same body plus `"confirm": true` → restores, keeps the rollback copy, restarts |
| GET | `/api/backups` | admin: the local scheduled backups and the clients' pushed ones `[{"name","client","size","created"}]` |
| GET | `/api/backups/{name}` | admin: download one (still encrypted) |
| POST | `/api/backups/push` | client token: a client's own encrypted backup |
| POST | `/api/backup/settings` | `{"passphrase","every_hours","keep","push"}` |

Request bodies for these endpoints may be up to 2 MiB.

CLI:
- `tailbondd backup --out FILE [--passphrase-file F]`: the passphrase comes from the file, from `TB_BACKUP_PASSPHRASE`, or a prompt.
- `tailbondd backup --plain` prints the summary.
- `tailbondd restore FILE [--yes] [--passphrase-file F]`: run it with the service stopped, or restart the service right after (the running agent would otherwise keep its settings in memory).

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
