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
- DNS for clients: `tailbondd` runs a forwarder on `10.77.0.1:53`. `*.ts.net` and the tailnet's MagicDNS suffix go to `100.100.100.100`, everything else to the system resolver. mqvpn pushes `DNS = 10.77.0.1`.
- TLS: if `tailscale cert <machine>.<tailnet>.ts.net` works, use it as mqvpn `Cert`/`Key`. Clients connect to the public address with `tls_server_name = <machine>.<tailnet>.ts.net` and verify normally. Fallback: self-signed + `insecure: true` plus the cert SHA-256 pinned in the pairing payload (enforced by the agent/app, not by mqvpn).

## Modes (user-facing → mqvpn config)

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

Auth: if `ui_password` is set, a session cookie from `POST /api/login {"password"}`. Server role binds `127.0.0.1` plus the Tailscale IP by default. Client role binds the LAN.

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

Rates (`*_bps`) are derived by the agent from successive byte counters.

## Web UI

One single-page app (vanilla JS + SVG, no build step, embedded with `go:embed`), Speedify-like:
- a header with the total download/upload rate and a 60 s sparkline, and a connected/disconnected toggle;
- a mode selector (Speed / Streaming / Redundant / Failover);
- one card per connection: icon by kind, live rate sparkline, latency, loss, state chip, enable toggle, priority dropdown, data used vs cap;
- server role: a Clients tab (add client → QR code, revoke, per-client paths) and a Tailscale panel;
- client role: a Setup tab (paste/scan pairing URI).
- Works on phone screens; dark/light follows the system.

## Build outputs

- `scripts/build-mqvpn-native.sh` → `third_party/mqvpn/build/mqvpn` (Fedora).
- `scripts/build-mqvpn-aarch64-musl.sh` → static `out/aarch64/mqvpn` via `podman` + `docker.io/library/alpine` (linux/arm64, qemu-user-static).
- `agent`: `CGO_ENABLED=0 go build` for linux/amd64 and linux/arm64.
- RPM: `packaging/fedora/build-rpm.sh` → `out/rpm/`.
- IPK: `packaging/openwrt/build-ipk.sh` → `out/ipk/tailbond_<ver>_aarch64_cortex-a53.ipk` (plain ar/tar ipk; no OpenWrt SDK needed since binaries are static).
- APK: `android/` Gradle → `out/android/tailbond-debug.apk`.
