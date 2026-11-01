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
| OpenWrt / GL.iNet package | `packaging/openwrt/` | Flint 3 (GL-BE9300), Slate 7 (GL-BE3600), any aarch64_cortex-a53 OpenWrt | `.ipk` with static musl mqvpn + tailbondd, init.d, UCI |
| Android app | `android/` | Android 8+ arm64 | fork of mqvpn's Android app, rebranded `dev.tailbond.app` |

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

## mqvpn client control API (patch)

Upstream mqvpn only serves the control API in server mode. `patches/mqvpn/0001-client-control-api.patch` adds it to client mode on Linux: `[Control] Listen` / `--control-port` in client mode serve `get_status` and `get_stats` with the **same JSON shape as server mode**, as one client object whose path entries additionally carry `"iface"` (the bound interface name). Keep the patch upstreamable (small, confined to `src/platform/linux/` + `src/platform/posix/`).

## tailbondd HTTP API (served on `:8077`, UI at `/`)

Auth: if `ui_password` is set, a session cookie from `POST /api/login {"password"}`. Server role binds `127.0.0.1` plus the Tailscale IP by default (or the configured `listen` host), and always also its tunnel IP (`10.77.0.1:8077`, bound once `tb0` has it) unless `listen` is a wildcard. Client role binds the LAN.

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
| POST | `/api/pair` | client: `{"uri"}` to import a pairing |
| GET | `/api/tailscale` | `tailscale status --json` summary + route approval state |
| GET | `/api/peers` | server, additive: the tailnet machines bonded clients reach (below). Auth: client API token, or an admin session |
| GET | `/api/clients/{name}` | server, additive: re-issue `{"uri","qr_svg"}` for an existing client |
| GET | `/api/logs?since=<seq>&limit=<n>` | additive: `{"entries":[{"seq","t","src":"agent\|mqvpn\|hook","line"}]}` ring buffer |
| GET/POST | `/api/settings` | additive: read role/TLS/pairing info; set `public_addr`, `ui_password` |
| POST | `/api/logout` | additive |

### Client API token and `GET /api/peers`

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
none. With Tailscale disabled or down, `peers` is `[]` and
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
- server role, viewed on a phone (or opened as `#add-phone`): an **Add this phone** card on top. One tap
  creates the client (`POST /api/clients`, name guessed from the phone model) and navigates to the
  `tailbond://pair` deep link, so the app opens with the pairing; no QR code on the same device.
  Re-adding an existing name reuses that client's key (`GET /api/clients/{name}`) after a confirm;
- client role: a Setup tab (paste/scan pairing URI).
- Works on phone screens; dark/light follows the system.

## Build outputs

- `scripts/build-mqvpn-native.sh` → `third_party/mqvpn/build/mqvpn` (Fedora).
- `scripts/build-mqvpn-aarch64-musl.sh` → static `out/aarch64/mqvpn` via `podman` + `docker.io/library/alpine` (linux/arm64, qemu-user-static).
- `agent`: `CGO_ENABLED=0 go build` for linux/amd64 and linux/arm64.
- RPM: `packaging/fedora/build-rpm.sh` → `out/rpm/`.
- IPK: `packaging/openwrt/build-ipk.sh` → `out/ipk/tailbond_<ver>_aarch64_cortex-a53.ipk` (plain ar/tar ipk; no OpenWrt SDK needed since binaries are static).
- APK: `android/` Gradle → `out/android/tailbond-debug.apk`.

## Platform hooks (keep the agent platform-neutral)

`tailbondd` never edits firewall, routing or Tailscale settings directly. It runs executable hooks from `hooks_dir` (default `/usr/lib/tailbond/hooks`) and logs their output:

| Hook | When | Args / env |
|---|---|---|
| `server-up` | after mqvpn server starts | env `TB_TUN`, `TB_SUBNET`, `TB_SUBNET6`, `TB_PORT`, `TB_DNS_IP` |
| `server-down` | after mqvpn server stops | same |
| `client-up` | tunnel connected | env `TB_TUN`, `TB_TUNNEL_IP`, `TB_SERVER_IP`, `TB_TAILNET_ROUTES` |
| `client-down` | tunnel down | same |

Every hook also gets `TB_ROLE`, `TB_MODE`, `TB_STATE_DIR` and `TB_CONFIG` (path of the generated mqvpn INI). Besides `<hooks_dir>/<hook>`, executables in `<hooks_dir>/<hook>.d/` run in lexical order.

- Fedora server hooks (`packaging/fedora/hooks/`): firewalld (open UDP port, `tb0` in a zone with masquerade toward the internet and `tailscale0`), `tailscale set --advertise-routes`. Without a running firewalld they fall back to mqvpn's NAT script plus iptables masquerade toward `tailscale0`.
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
- `client.manage_routes` (default `true`; set it to `false` on OpenWrt), `client.no_dns` (don't let mqvpn change the system resolver), `client.disconnected` (persisted connect toggle), `client.paths[].kind` (override the kind heuristic);
- `disable_tailscale`: skip every `tailscale` CLI call; `control_addr`; `log_level`.

Mode table details: `Reorder on (Profile = fiber_lte)` is rendered as `[Reorder] Enabled = true` plus a `[ReorderRule]` for UDP 443 with `fiber_lte` and one for UDP 53 with `default_udp`. The server always enables `[Hybrid]`, `[Reorder]` and `AllowClientScheduler`, so each client's mode takes effect. Failover keeps only the best usable priority tier in the client's `Path` list. The bonding modes use primary and secondary paths, and add backup paths only when no primary or secondary path is usable. Paths over their monthly `data_cap_mb` drop out.
