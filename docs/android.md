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
(`~/.android/debug.keystore`, created on first build; not committed). It
can't update a release install, and a release APK can't update it: see
[Updates](#updates) for `build-release-apk.sh` and the release key.
Only `arm64-v8a` is built.

Individual steps:

```sh
android/scripts/gradle.sh :app:testDebugUnitTest   # pairing parser, mode table, path gate, server list, failover, website bypass, updates, backup
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
  *Disconnected* ("Stopped. …"). **Hand the VPN back to Tailscale** is the last row of the main screen.
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
connect). **Hand the VPN back to Tailscale** (at the bottom of the main screen) stops tailbond, remembers that choice
(`handedToTailscale`; cleared the next time tailbond starts) and opens
`com.tailscale.ipn` (declared in `<queries>`), or its store page.

**Settings**: servers (order, pair again / remove, add another, when to
switch; see [Servers and failover](#servers-and-failover)), internet traffic,
apps and websites that skip tailbond ([Skipping the tunnel](#skipping-the-tunnel)),
switch back to Tailscale, Always-on VPN guidance (turn Tailscale's always-on
off first).

**Quick settings tile** ("tailbond"): subtitle is the same wording (*Wi-Fi +
mobile data*, *Connecting…*, *Disconnected*, *Can't connect*); tap to
connect/disconnect. It opens the app when VPN consent or pairing is missing.

**Notification**: "Connected through <server>", the failover note (when
there is one, e.g. "homebox is unreachable", and on a backup "Still
protected, nothing to do"), the connections sentence and `↓ rate ↑ rate`,
with a Disconnect action.

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

## Updates

Like a typical open-source app: each release is a GitHub release with a
release-signed `tailbond-<version>.apk` (and `SHA256SUMS`;
[architecture.md](architecture.md#updates), [updates.md](updates.md)).
Two ways to stay current:

- **Recommended: [Obtainium](https://obtainium.imranr.dev/).** It tracks
  GitHub releases and installs updates for you. In Obtainium tap **Add App**,
  paste `https://github.com/kaovilai/tailbond`, keep the source "GitHub", and
  set the APK filter to `^tailbond-[0-9.]+\.apk$` (leaves out `-debug` and
  pre-release APKs). It works only with the release APK (same key as the
  one installed), not a debug build.
- **In the app**: Settings → **Updates**. About once a day (when the app
  comes to the front or the page opens, and the last check is older than
  24 h; **Check now** any time) it asks
  `GET https://api.github.com/repos/<repo>/releases/latest`. `<repo>` is
  built in: `BuildConfig.UPDATE_REPO`, from the Gradle property or env
  `TB_UPDATE_REPO` (default `kaovilai/tailbond`). A newer `tag_name` than
  the app's `versionName` shows **Update available: vX.Y.Z** with **What's
  new** (the release page) and **Download update**, which opens the APK in
  the browser; open the download and tap Update, and Android installs it over
  the app because it's signed with the same key. Policy **Tell me**
  (default) or **Off** (no checks by itself). A build whose version isn't a
  numbered release (`dev`, `-dirty`) shows the latest release but never
  says "Update available". A debug build says it can't take release APKs.

The app's requests to its server (`GET /api/peers` and the others with the
client token) carry `X-Tailbond-Version` and `X-Tailbond-Kind: android`, so
the server's dashboard can show which phones are out of date.

Installing an update ends the app's process, so the tunnel drops. If it was
on (tailbond remembers whether you turned it on or off), the
`MY_PACKAGE_REPLACED` broadcast reconnects it, as long as Android still lets
tailbond be the VPN (`VpnService.prepare()` needs no dialog) and a server is
paired.

**The release key.** Android installs an update only if it's signed with
the same key as the installed app, so every release APK is signed with one
keystore, forever. Debug builds use the debug key, so a release APK can't
update a debug install (or the other way round): save a backup, uninstall,
install the release APK, restore. `scripts/release/gen-android-keystore.sh`
makes the keystore once and stores it as the GitHub secrets
`TB_RELEASE_KEYSTORE_B64`, `TB_RELEASE_KEYSTORE_PASSWORD`,
`TB_RELEASE_KEY_ALIAS` and `TB_RELEASE_KEY_PASSWORD`, plus an encrypted
backup outside the repo. CI builds the APK with
`android/scripts/build-release-apk.sh`, which stops with a clear message
when the secrets are missing. Locally, from a decrypted copy:

```sh
TB_RELEASE_KEYSTORE=/path/to/tailbond-release.p12 \
TB_RELEASE_KEYSTORE_PASSWORD=… TB_RELEASE_KEY_ALIAS=tailbond TB_VERSION=0.2.0 \
    android/scripts/build-release-apk.sh         # → out/android/tailbond-0.2.0.apk, prints the signer
```

> **Losing the release keystore** means every user uninstalls and
> reinstalls tailbond once (Android refuses an update signed with another
> key, in the app and in Obtainium alike), then restores a `.tbbackup` (see
> below). Keep the encrypted backup and its passphrase safe.

**Versions.** `versionName` is `TB_VERSION` (leading `v` dropped), else
`tailbond.version` in `android/gradle.properties`. `versionCode` must grow
with every release: `MAJOR*1000000 + MINOR*10000 + PATCH*100` (0.2.0 →
20000); a pre-release ending in a number N is that − 100 + N (1.2.3-beta.4 →
1020204). MINOR and PATCH stay ≤ 99.

## Backup and restore

Settings → **Back up and restore** (servers and routers: [backup-restore.md](backup-restore.md)):

- **Save a backup file**: choose a passphrase (at least 10 characters,
  typed twice), then a place through Android's file picker (Downloads, Drive,
  …). The file is a `.tbbackup` ([architecture.md](architecture.md#backup-and-restore)):
  scrypt (N=32768, r=8, p=1, 16-byte salt) and AES-256-GCM (12-byte nonce,
  the header line as AAD), with `"role":"client"` and `"platform":"android"`.
  The plaintext is `{"format":1,"config":{"servers":[…],"settings":{…}}}`:
  every server with its pairing (keys included), and every setting, including
  apps and websites that skip tailbond. There is no unencrypted export.
- **Restore from a file**: pick the file and type its passphrase. A preview
  lists what changes ("Servers: homebox → homebox and vps", "Mobile data
  limit: no limit → 5 GB a month", …) and warns about excluded apps that
  aren't installed. Nothing changes until you tap **Restore**. A backup from a
  server or a router is refused. A wrong passphrase and a damaged file give the
  same message ("Wrong passphrase or damaged file"). Before restoring, the
  current setup is kept on the phone, sealed with the app's Android Keystore
  key like the pairing itself, so **Undo restore** works once. A running
  tunnel picks up the restored servers and settings by itself.
- **Send a backup to homebox** (while connected): the same encrypted file
  (≤ 1 MiB) goes to `POST /api/backups/push` on the server in use, with the
  client token. The server keeps the last 3 per client.

scrypt is implemented in Kotlin from RFC 7914 (`backup/Scrypt.kt`, tested
with the RFC's PBKDF2, Salsa20/8 and scrypt vectors), because Android has no
scrypt in the platform. A backup takes about a second to encrypt or open
and needs 32 MiB.

Interop vectors for other implementations (the Go agent) are in
`android/app/src/test/resources/interop/`: `android.tbbackup` with its
plaintext, and `vectors.json` (passphrase, salt, nonce, derived key,
hashes). `InteropVectorsTest` rebuilds them byte for byte.

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

## Servers and failover

The app keeps an **ordered list of servers** (each one a pairing, all of
them sealed together with the Keystore key, `data/Repository.kt`); the first
is the main server. An app that had one pairing moves it into a one-server
list on first start. Contract: [Multiple servers and
failover](architecture.md#multiple-servers-and-failover).

- **Names and subnets**: a server is called by its pairing's `server_name`,
  else the first label of `sni`, else the host of `server`. Its tunnel subnet
  is the pairing's `subnet`, else the /24 around its DNS address, else
  `10.77.0.0/24`. Its DNS and machine list come from that subnet's `.1`.
- **Add another server** (Settings): the same three ways as the first pairing
  (ask over Tailscale, the server's page, a code or link), but the result is
  added as a backup. The page offers the tailnet machines that answer
  `GET /api/hello` as a tailbond server (asked through the tunnel, 1.5 s
  each); otherwise type the name. Asking sends `avoid_subnets` (the subnets
  the phone already has), so a new server without devices moves off them by
  itself. The app refuses a server that is already in the list, or whose
  subnet overlaps one in the list. For a clash it says which server clashes,
  shows the steps for someone who runs the new server (`install.sh --subnet`
  with a free /24, or the GL panel), then offers a message to copy for
  whoever does. A `tailbond://pair` link for a server
  already in the list renews that server's pairing; any other link adds a
  backup.
- **Settings list**: each server with its state (*In use*, *Ready as
  backup*, *Unreachable*, *Connecting…*), up/down buttons to reorder, *Use
  this server now*, *Pair again* and *Remove*. **When to switch**: "Prefer
  homebox, use vps if homebox is down" (default) or "Always use <server>".
- **How it switches** (`failover/FailoverEngine.kt`, pure, tested with a fake
  clock in `FailoverEngineTest`): the server in use works while mqvpn says
  connected and at least one path is active. After 15 s without that, the
  app switches to the next server in order, skipping one whose last probe
  failed if another answered. Standby servers get one UDP packet every 10 s,
  outside the tunnel (`service/ServerProber.kt`: a 60-byte QUIC short-header
  packet from a `protect()`ed socket bound to each validated non-VPN network
  in turn; any reply within 2 s counts). Once an earlier server has answered
  every probe for 2 minutes, the app goes back to it (one missed probe starts
  the count again); a failback that doesn't come up doubles that server's
  wait, up to 16 times, until it has worked for 10 minutes. *Use this server
  now* switches at once and stays there while that server works. The 15 s and
  2 minutes are `AppSettings.switchAfterSec` / `failbackAfterSec`.
- **A switch** ends the mqvpn session and starts one to the next server with
  its own address, TLS name, key, DNS and routes; app and website bypass stay.
  A server that fails before connecting (no answer, wrong certificate, key
  refused) is retried every 3 s until the 15 s are up; when every server has
  failed that way in a row, tailbond stops and says so, like with one server.
- **Staying protected**: during a switch (a few seconds) and when no server
  answers, Android sends apps over the phone's normal internet unless its
  *Block connections without VPN* is on. With backup servers Settings
  recommends turning it on (with an *Open Android VPN settings* button), and
  the switching note links there. It also blocks hotel Wi-Fi sign-in pages
  and handing the VPN back to Tailscale, so turn it off for those.
- **Recent switches**: the app keeps the last 20 (time, from, to, why) and
  lists them under the servers in Settings. After *Use this server now*,
  "When to switch" shows "Paused: vps picked by hand" with *Go back to automatic*.
- **Wording**: the home card says "Using vps since 14:02 (homebox is
  unreachable)", "Back to homebox in 1:20" while counting down (with **Stay
  on vps until you pick homebox or vps stops working**), "Using vps since
  15:10 (picked by hand)" (with **Go back to
  homebox**), and "Back on homebox since 14:20" for a few minutes after going
  back; a **Servers** section lists each server's state. *Use this server
  now* isn't offered for a server that doesn't answer. With backups, the
  internet traffic setting reads "Through your server" (the one in use).
  With one server nothing of this shows.

## Skipping the tunnel

Two ways to let traffic use the phone's own connection instead of tailbond:

- **Apps that skip tailbond** (every Android version): the chosen apps are
  left out of the VPN (`addDisallowedApplication`). The surest way: it covers
  everything the app does.
- **Websites that skip tailbond** (**Android 13 or newer** only): the app
  looks up each website and its usual sub-names (`www`, `login`, `secure`,
  `online`, `auth`, `account(s)`, `my`, `m`, `app`, `api`) through the tunnel's DNS when the
  tunnel comes up and every 10 minutes (`data/DnsLookup.kt`), and leaves those
  addresses out of the VPN with `VpnService.Builder.excludeRoute`. Addresses
  in the server's subnet, its DNS and the tailnet ranges stay in the tunnel.
  New addresses mean building the VPN interface again
  (`MqvpnVpnService.rebuildTun`, a short break, no new QUIC connection): at
  most once a minute, while addresses seen in the last hour stay excluded.
  Changing the list applies at once. On Android 12 and older the page says
  "Needs Android 13 or newer" and points to *Apps that skip tailbond*:
  older Android can't exclude routes from a VPN.
  Presets: **Banking** (type your bank's site), **Streaming** (adds
  `netflix.com`, `nflxvideo.net`; for a streaming app, skip the app instead),
  **Wi-Fi calling** (nothing to add: the phone's own Wi-Fi calling already
  goes straight over Wi-Fi; calling apps are apps).
  A rule covers exactly those names (Android can only leave out addresses it
  looked up itself, with no DNS interception inside the VPN, so it can't learn
  other sub-names); add other names as their own rows, or put the bank's app
  in *Apps that skip tailbond*. A tailbond travel router covers every name
  under a website. On a tailbond travel router's Wi-Fi the phone's own rules
  apply first; the router's rules then treat the phone like any other device. The
  lookups are answered by the server in use, so a site may pick its servers
  for where that server is rather than where the phone is.
  The limit, as the page says it: it goes by addresses, so parts of a site
  loaded from other names still use tailbond, other sites on the same
  addresses (big hosting networks) skip it too, and apps with their own
  private DNS may not follow it. With *Internet traffic: Only Tailscale
  machines*, websites already skip tailbond.

## Routes and DNS

- **Full tunnel** (default): `0.0.0.0/0` and `::/0` (IPv6 is routed into the
  tunnel even when the server assigns no IPv6 address, so it can't leak around it).
- **Tailnet + home LAN only**: the server's tunnel subnet (`10.77.0.0/24` by default) plus the pairing's
  `tailnet_routes` (e.g. `100.64.0.0/10`, `fd7a:115c:a1e0::/48`); DNS servers
  outside those prefixes get a host route.
- **Per-app exclude**: `addDisallowedApplication` for each chosen app.
- **Per-website exclude** (Android 13+): `excludeRoute` for each website's
  addresses ([Skipping the tunnel](#skipping-the-tunnel)).
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
Android's sign-in can't load: turn it off while you sign in, then on again
(it's recommended with backup servers, see [Servers and failover](#servers-and-failover);
[setup tips](setup-tips.md)).

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
- **Failover and website bypass are untested on a phone.** The switching
  rules, the probe packet, the DNS parsing and the address lists are unit
  tested; the probe reply from a real server, the switch itself, and
  `excludeRoute` plus the in-place TUN rebuild on Android 13+ are not.
- **The redesigned UI is verified by renders, not on a phone**: the
  handoff detection (another VPN network present), launching Tailscale, the
  browser round trip through *Add this phone* and the machine list over the
  tunnel are untested on a device. `GET /api/peers` itself is covered by the
  agent unit tests and `scripts/e2e-netns.sh`.
- **The update check is untested against the live GitHub API on a phone**:
  parsing, version comparison, the 24 h rule and the words are unit tested
  with recorded answers; the browser download and Android installing it
  over the app are not.
- The phone is not a tailnet machine: tailnet peers see its traffic as coming
  from the home server, and reach it at its tunnel IP only if the server's
  `10.77.0.0/24` route is approved.
- Only `arm64-v8a`; `minSdk` 26 (Android 8).
- The QR scanner needs a back camera; without one, paste the URI.
