# tailbond Android app

`android/` builds `dev.tailbond.app` ("tailbond"), the phone side of tailbond:
it bonds Wi-Fi + cellular (+ Ethernet) to your home server's mqvpn with
Multipath QUIC, and through that server reaches the internet and your
tailnet. The phone runs only this VPN app, not Tailscale (Android allows one
VPN at a time).

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

First launch shows one screen for Tailscale users: tailbond replaces the
Tailscale VPN on this phone (Android runs one VPN at a time), and the
tailnet stays reachable through the home server. Then, in order of ease:

**Getting the app** (no store listing): with Tailscale on, type the home
server's name (`homebox`) into the phone's browser. The server's **phone
page** (`http://homebox/`, port 80 on its Tailscale IP; also
`http://homebox:8077/get`) offers `tailbond.apk`, shows what Android's
install prompts look like (Chrome's "File might be harmful", *Install
unknown apps → Allow from this source*, *Install*, Play Protect's scan),
and then **Open tailbond and ask homebox**: an Android intent link to
`tailbond://ask?server=homebox.<tailnet>.ts.net` that falls back to the
page's install step when the app isn't installed yet. The app skips its intro
and asks the server right away (step 0 below), unless it's already paired.
The admin's dashboard and the GL panel show the page's QR code under **Add
an Android phone**. Those prompts are Android's own and can't be removed
for an app outside a store.

0. **Ask your home server** (no link, no code): while the Tailscale app still
   runs the VPN, type the home server's Tailscale name and tap **Ask to
   connect**. The app resolves it through Tailscale (MagicDNS or a 100.x
   address; anything that isn't a Tailscale address is refused), checks
   `GET /api/hello`, sends `POST /api/pair-requests` and polls until someone
   approves it on the server; the usual confirm dialog then shows the
   pairing ([contract](architecture.md#pairing-over-tailscale)). The request
   is plain HTTP inside Tailscale's encrypted tunnel, over a raw socket
   (`data/PairRequest.kt`), because the app's network security config allows
   cleartext only to the tunnel address.
1. **On this phone** (no QR code): while Tailscale is still on, open the home
   server's page (`http://<server>:8077`, the app's *Open home server page*
   button does it from the server's Tailscale name). On a phone the page shows
   only **Add this phone**: step 1 downloads the app if the server offers
   `GET /tailbond.apk`, step 2 creates the client and opens the
   `tailbond://pair` deep link, so the app comes up with the pairing.
2. **From another screen**: on the server's dashboard **Devices → Add a
   device** shows a QR code; in the app tap **Scan code from your home
   server** (CameraX + ZXing, no Google Play services).
3. **Paste** a `tailbond://pair?d=<base64url(JSON)>` link
   ([payload](architecture.md#pairing-payload)).

The confirm dialog says which home server and phone name in plain words;
address, TLS name, mode, DNS, routes and the pin are under *Technical details*.

The pairing (it holds the PSK) is stored AES-256-GCM-encrypted with a
non-exportable Android Keystore key (`data/SecureBox.kt`); settings, the
cellular counter and the last machine list are plain SharedPreferences.
Backups are disabled.

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

Laid out like the official Tailscale app (`ui/HomeScreen.kt`):

- **Top bar**: the home server's name and "via home server · <tailnet>", gear → Settings.
- **Status card**: one big switch; *Connected* / *Connecting…* / *Reconnecting…* /
  *Disconnected* ("Stopped. …"). Below it, **Switch back to Tailscale**.
- **Errors** say what happened and offer the fix: home server unreachable →
  check it's on / the router forwards UDP 443 → *Try again*; certificate
  mismatch → *Try again* or *Pair again*; key rejected → *Pair again*.
- **Connections**: one sentence ("Using both Wi-Fi and mobile data", "Using Wi-Fi ·
  saving mobile data (backup only)", "Wi-Fi isn't working · using mobile data") and a
  card per link (*In use*, *Not in use · saving data*, *Not working right now*, …);
  a **Save mobile data** switch (mode `failover`); tap for the Connections page:
  rates and graph, per link *Use for tailbond* and role (Main / Extra / Backup
  only = primary / secondary / backup), the four modes with one-line
  explanations, and the monthly mobile data limit.
- **Internet traffic** (like Tailscale's Exit node row): through the home server
  (full tunnel) or only Tailscale machines.
- **This device**: the phone's tunnel address (tap to copy) and a note that its
  traffic reaches the tailnet from the home server.
- **Machines**: the home server's `GET /api/peers` (see the contract), refreshed
  every 30 s while connected and cached for offline display: online dot, name,
  MagicDNS name, 100.x address, OS, search, copy icon (IPv4); tap a machine for
  its page where every address is tap-to-copy. The token is
  HMAC-SHA256(pairing key, "tailbond-api"); cleartext HTTP is allowed only to
  `10.77.0.1` (`res/xml/network_security_config.xml`).

**VPN handoff**: before Android's own VPN dialog the app explains that turning
tailbond on turns off Tailscale (shown when another VPN is up, or on the first
connect). **Switch back to Tailscale** stops tailbond, remembers that choice
(`handedToTailscale`; cleared the next time tailbond starts) and opens
`com.tailscale.ipn` (declared in `<queries>`), or its store page.

**Settings**: home server (pair again / remove), internet traffic, apps that
skip tailbond (per-app exclude, searchable), switch back to Tailscale,
Always-on VPN guidance (turn Tailscale's always-on off first).

**Quick settings tile** ("tailbond"): subtitle is the same wording (*Wi-Fi +
mobile data*, *Connecting…*, *Disconnected*, *Can't connect*); tap to
connect/disconnect. It opens the app when VPN consent or pairing is missing.

**Notification**: "Connected through <server>", the connections sentence and
`↓ rate ↑ rate`, with a Disconnect action.

**Always-on VPN**: supported (`SUPPORTS_ALWAYS_ON`); when the system starts the
service it uses the stored pairing and settings.

## Screen renders

Every screen and state renders to PNGs with Robolectric + Roborazzi from fake
state (`src/test/kotlin/dev/tailbond/app/ux/`), 1080x2400, light and dark:

```sh
android/scripts/gradle.sh :app:testDebugUnitTest --tests 'dev.tailbond.app.ux.*' -Ptailbond.uxOut=$PWD/out/ux/after
scripts/ux/web-screens.sh   # the phone page + the dashboard's "Add this phone" flow → out/ux/web
```

Without `-Ptailbond.uxOut` the render tests are skipped. The simulated user
tests that used them are in [ux/summary.md](ux/summary.md) and
[ux/final-summary.md](ux/final-summary.md).

## Updates (future option)

Today a new version is installed the same way (the phone page offers the
APK the server has; Android installs it over the old one, same signing
key). An Obtainium- or F-Droid-style update channel (a release feed the
phone polls) would make updates automatic; nothing is published for it yet.
Release builds would also need a stable signing key instead of the debug
key.

## Modes

Mapped exactly per the contract's mode table (`model/Mode.kt`, tested in
`ModeMappingTest`):

| Mode | Scheduler | Other mqvpn keys |
|---|---|---|
| Speed (default; app: Fastest) | `wlb` | Hybrid on, Reorder on with a UDP/443 rule `Profile = fiber_lte`, `Reinjection = deadline`, `RequestServerScheduler = true` |
| Streaming (Smooth calls and video) | `wlb` | `Redundant = true` (all UDP), `Reinjection = idle`, `CC = bbr`, Hybrid on |
| Redundant (Most reliable) | `minrtt` | `Redundant = true`, `Reinjection = idle`, `CC = bbr`, Hybrid on, `RequestServerScheduler = true` |
| Failover (Save mobile data) | `minrtt` (or `backup_fec` if the experimental switch is on) | Reinjection off; non-primary links held in standby by the app |

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

## Wi-Fi sign-in pages (captive portals)

mqvpn's network monitor (`sdk-network`) only offers networks Android has
validated (`NET_CAPABILITY_VALIDATED`), so a hotel Wi-Fi behind a sign-in
page is never bonded; mobile data keeps the tunnel up. tailbond watches
Wi-Fi networks itself (`service/CaptiveWatcher.kt`): one with
`NET_CAPABILITY_CAPTIVE_PORTAL`, or still unvalidated after 10 s, shows
**Wi-Fi needs sign-in** on the home screen and the Wi-Fi card. **Sign in to
Wi-Fi** opens Android's internet panel, where the network offers "Sign in to
network". Android's captive portal login app binds itself to that Wi-Fi
network (not the default route), so tailbond's VPN doesn't intercept it and
you don't turn tailbond off. Once Android validates the Wi-Fi, the monitor
offers it and it joins the bond by itself. The prompt can be turned off in
Settings (**Tell me when Wi-Fi needs a sign-in**); bonding never uses an
unvalidated Wi-Fi either way. With "Block connections without VPN" on,
Android's sign-in can't load, so leave it off ([setup tips](setup-tips.md)).

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
- **The redesigned UI is verified by renders, not on a phone**: the
  handoff detection (another VPN network present), launching Tailscale, the
  browser round trip through *Add this phone* and the machine list over the
  tunnel are untested on a device. `GET /api/peers` itself is covered by the
  agent unit tests and `scripts/e2e-netns.sh`.
- The phone is not a tailnet machine: tailnet peers see its traffic as coming
  from the home server, and reach it at its tunnel IP only if the server's
  `10.77.0.0/24` route is approved.
- Only `arm64-v8a`; `minSdk` 26 (Android 8).
- The QR scanner needs a back camera; without one, paste the URI.
