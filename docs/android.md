# tailbond Android app

`android/` builds `dev.tailbond.app` ("tailbond"), the phone side of tailbond:
it bonds Wi-Fi + cellular (+ Ethernet) to your home server's mqvpn with
Multipath QUIC, and through that server reaches the internet and your
tailnet. The phone runs only this VPN app, not Tailscale (Android allows one
VPN at a time; see [findings.md](findings.md)).

It is built on mqvpn v0.19.0's Android SDK (`third_party/mqvpn/android`):

| Module | Source |
|---|---|
| `sdk-runtime`, `sdk-network` | used as-is from the submodule (Gradle `projectDir`) |
| `sdk-native`, `sdk-core` | forked into `android/mqvpn-sdk/` with small marked patches, see [android/mqvpn-sdk/README.md](../android/mqvpn-sdk/README.md) |
| `app` | tailbond's own UI and VPN service (`android/app`) |

## Build

On Fedora (tested on Fedora 44 under WSL2):

```sh
android/scripts/setup-sdk.sh           # JDK 21, SDK 37, build-tools 37, NDK 28.2.13676358, CMake 3.22.1 → ~/Android/Sdk
git submodule update --init --recursive
android/scripts/build-native.sh        # libmqvpn + xquic + BoringSSL for arm64-v8a (mqvpn's build_android.sh, -j4)
android/scripts/build-apk.sh           # unit tests + lint + assembleDebug → out/android/tailbond-debug.apk
```

`build-apk.sh` runs `scripts/gradle.sh`, which points the Gradle wrapper at
the JDK and SDK above. Gradle is tuned for a small shared host
(`android/gradle.properties`: 2 GB heap, 2 workers, in-process Kotlin).
The debug APK is signed with the standard Android debug key
(`~/.android/debug.keystore`, created on first build; not committed).
Only `arm64-v8a` is built.

Individual steps:

```sh
android/scripts/gradle.sh :app:testDebugUnitTest   # pairing parser, mode table, path gate
android/scripts/gradle.sh :app:lintDebug
android/scripts/gradle.sh :app:assembleDebug
```

## Install

```sh
adb install -r out/android/tailbond-debug.apk
```

From WSL2, either use the Windows `adb.exe` (`/mnt/c/.../platform-tools/adb.exe install -r
'\\wsl$\...\tailbond-debug.apk'`) or pair over Wi-Fi with `adb pair` / `adb connect` from
the WSL `adb` in `~/Android/Sdk/platform-tools`.

## Pairing

1. On the server's tailbond dashboard: **Clients → Add client**. It shows a QR code
   and a `tailbond://pair?d=<base64url(JSON)>` URI (payload in
   [architecture.md](architecture.md#pairing-payload)).
2. In the app, either
   - tap **Scan QR code** (CameraX + ZXing, no Google Play services),
   - paste the URI (**Paste** / **Import**), or
   - open the URI on the phone (the app handles the `tailbond://pair` deep link).
3. Confirm the server, TLS name, mode, DNS, routes and certificate mode in the dialog.

The pairing (it holds the PSK) is stored AES-256-GCM-encrypted with a
non-exportable Android Keystore key (`data/SecureBox.kt`); settings and the
cellular counter are plain SharedPreferences. Backups are disabled.

TLS:
- `sni` → mqvpn `tls_server_name`; the certificate is verified against the system
  trust store for that name (the normal `tailscale cert` case).
- `insecure: true` with `cert_sha256` → the app turns mqvpn's `insecure` off and
  installs a pin check in libmqvpn's certificate-verifier hook: the TLS handshake
  fails unless the leaf certificate's SHA-256 matches. This is enforced in the
  handshake (sdk-native fork, `setCertPin`), not after it.
- `insecure: true` without a pin → no verification at all; the confirm dialog
  warns about it.

## Using it

**Dashboard**
- Big connect switch; total download/upload rate with a 60-second graph; data
  totals, tunnel RTT, loss and uptime.
- Mode selector: Speed / Streaming / Redundant / Failover (below).
- One card per link kind (Wi-Fi / Cellular / Ethernet): state chip
  (active / pending / degraded / standby / failed / off), live rate and
  sparkline, RTT, bytes, an enable switch and a priority
  (Primary / Secondary / Backup).
- Cellular data used this calendar month, against the optional cap.

**Settings**: routing (full tunnel or tailnet + home LAN only), per-app exclude,
cellular cap, the experimental `backup_fec` failover scheduler, a shortcut to
system VPN settings (always-on), re-pair / forget.

**Quick settings tile** ("tailbond"): shows state and combined rate; tap to
connect/disconnect. It opens the app when VPN consent or pairing is missing.

**Notification**: persistent while connected, shows `↓ rate ↑ rate` and the
active links, with a Disconnect action.

**Always-on VPN**: supported (`SUPPORTS_ALWAYS_ON`); when the system starts the
service it uses the stored pairing and settings.

## Modes

Mapped exactly per the contract's mode table (`model/Mode.kt`, tested in
`ModeMappingTest`):

| Mode | Scheduler | Other mqvpn keys |
|---|---|---|
| Speed (default) | `wlb` | Hybrid on, Reorder on with a UDP/443 rule `Profile = fiber_lte`, `Reinjection = deadline`, `RequestServerScheduler = true` |
| Streaming | `wlb` | `Redundant = true` (all UDP), `Reinjection = idle`, `CC = bbr`, Hybrid on |
| Redundant | `minrtt` | `Redundant = true`, `Reinjection = idle`, `CC = bbr`, Hybrid on, `RequestServerScheduler = true` |
| Failover | `minrtt` (or `backup_fec` if the experimental switch is on) | Reinjection off; non-primary links held in standby by the app |

Keys a row doesn't mention are reset to mqvpn defaults (BBR2, no reorder,
no redundancy, no hybrid). The pairing's `mode` is the default; picking a mode
in the app overrides it. A mode change restarts the session (mqvpn config is
fixed per connection).

## Links, priority and standby

`sdk-network` holds a `requestNetwork` for every enabled transport, which is
what keeps cellular up while Wi-Fi is connected (verified in
`NetworkMonitor.start`: one `requestNetwork(TRANSPORT_CELLULAR + INTERNET)` hold
per allowed transport, plus a listener on validated INTERNET networks), and
binds one UDP socket per `Network` (`PathBinder`, `protect` + `bindSocket`).

On top of that the app gates which networks carry a path (`service/PathGate.kt`):
- **Speed / Streaming / Redundant**: primary and secondary links are used;
  backup links are attached only while no primary/secondary link is healthy.
- **Failover**: only primary links; secondary and backup are standby.
- A link is unhealthy when mqvpn reports it degraded/failed, or it has been
  pending for 10 s. Standby links detach again after the primaries have been
  healthy for 10 s.
- Disabling a link, changing priority and the cellular cap apply live; enabling
  a transport that was off when the session started restarts the session.

Defaults: Wi-Fi and Ethernet primary, cellular secondary.

## Routes and DNS

- **Full tunnel** (default): `0.0.0.0/0` and `::/0` (IPv6 is routed into the
  tunnel even when the server assigns no IPv6 address, so it can't leak around it).
- **Tailnet + home LAN only**: `10.77.0.0/24` plus the pairing's
  `tailnet_routes` (e.g. `100.64.0.0/10`, `fd7a:115c:a1e0::/48`); DNS servers
  outside those prefixes get a host route.
- **Per-app exclude**: `addDisallowedApplication` for each chosen app.
- **DNS**: the pairing's `dns` list (default `10.77.0.1`, the server's
  forwarder that resolves MagicDNS names).

## Known limitations

- **Untested on a device.** No phone or emulator was available while building
  (WSL2 has no KVM, so no emulator). Tested: unit tests, lint, a successful
  debug build. Untested: the tunnel itself, QR scanning, the tile, always-on,
  the cert pin against a live server.
- **Per-path loss**: mqvpn's client API (`mqvpn_client_get_paths`) only returns
  status, SRTT and byte counters per path. The cards show the tunnel-wide
  datagram loss (`dgram_lost / dgram_sent`, last second) instead of per-path
  loss. Per-path loss needs an mqvpn change (`xqc_path_metrics_t.path_lost_count`
  is available inside `mqvpn_client.c`).
- **Priority is an app-side policy.** mqvpn has no per-path priority; the app
  attaches/detaches paths. A standby link therefore needs a fresh QUIC path
  validation (around one RTT plus the bind) when it comes up, rather than being
  pre-validated. `backup_fec` (experimental) keeps validated standby paths in
  mqvpn itself.
- **Cellular counter** counts bytes on tunnel paths over cellular (mqvpn path
  byte counters), not traffic that bypasses the VPN (excluded apps, other
  profiles). It resets on the 1st of each month or manually.
- **Mode, route and app-exclude changes reconnect** the tunnel (a short drop).
- Only `arm64-v8a`; `minSdk` 26 (Android 8).
- The QR scanner needs a back camera; without one, paste the URI.
