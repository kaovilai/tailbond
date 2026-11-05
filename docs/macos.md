# tailbond on a Mac

Your MacBook as the **hub**: tailbond bonds the Mac's Wi-Fi, its Ethernet or
USB-C adapter, an **iPhone plugged in over USB**, and the mobile data that
**nearby phones share** ([Pair & Share](pair-and-share.md)) into one tunnel
to your home server. It's the same client role as the travel router and a
Linux laptop: the same dashboard, the same settings, the same tunnel.

A Mac that stays at home can also **be** the home server instead: see
[Mac as home server](#mac-as-home-server).

- Apple silicon (arm64) Macs. Intel Macs: not built (see [Limits](#limits-and-what-isnt-tested)).
- No paid Apple account: the binaries are unsigned (ad-hoc signed by the
  linker, which is all Apple silicon needs to run them), and you install
  them yourself. Gatekeeper only gets in the way of software that's
  *downloaded* and then double-clicked; the installer below takes care of
  that ([Gatekeeper](#gatekeeper-and-quarantine)).
- Runs as a LaunchDaemon (root): mqvpn opens a `utun` tunnel device and sets
  the routes and DNS servers, like any VPN.

## Install

**Download:** [tailbond-macos-arm64.tar.gz](https://github.com/kaovilai/tailbond/releases/latest/download/tailbond-macos-arm64.tar.gz)
(the bundle) or the Homebrew formula
[tailbond.rb](https://github.com/kaovilai/tailbond/releases/latest/download/tailbond.rb), from the latest release
([all versions](https://github.com/kaovilai/tailbond/releases)). Releases appear once the first release is
published; until then, [build it from source](#build-from-source).

### From your home server's page (easiest)

With Tailscale on, open **homebox/** (your server's Tailscale name, with the
slash) in Safari on the Mac. The **Get tailbond** page opens on its **Mac**
tab (or go to `http://homebox/#mac`):

1. Copy the Terminal line under **From homebox** and run it: it downloads
   the bundle from the server (SHA-256 shown on the page), unpacks it in
   Downloads and runs `install.sh`. The server serves the bundle when its
   admin put one at `/usr/share/tailbond/tailbond-macos-arm64.tar.gz`, else
   it fetches it once from the GitHub release (checked against
   `SHA256SUMS`) and keeps it. The page also has a download button and the
   Homebrew commands below.
2. Tap **Open tailbond and ask homebox**: tailbond's page on this Mac opens
   (`http://localhost:8077/?ask=homebox…`) with **Ask your home server**
   filled in. Tap **Ask homebox**; whoever runs homebox taps **Approve** on
   its page.

On another device (a phone, the PC you're reading this on), the page's Mac
tab shows the same steps to do on the Mac.

### The release bundle

You need a tailbond home server and either Tailscale on the Mac (to ask the
server from the dashboard's **Setup** tab) or a pairing link (the server's
dashboard → **Devices** → **Add a device**).

1. In Terminal, download the bundle and check it against the release's
   `SHA256SUMS`:

   ```sh
   cd ~/Downloads
   curl -fLO https://github.com/kaovilai/tailbond/releases/latest/download/tailbond-macos-arm64.tar.gz
   curl -fLO https://github.com/kaovilai/tailbond/releases/latest/download/SHA256SUMS
   grep ' tailbond-macos-arm64.tar.gz$' SHA256SUMS | shasum -a 256 -c   # tailbond-macos-arm64.tar.gz: OK
   ```

2. Unpack it and install:

   ```sh
   tar -xzf tailbond-macos-arm64.tar.gz
   sudo tailbond-macos-arm64/install.sh --pair 'tailbond://pair?d=…'
   ```

   (Leave out `--pair` to pair later in the dashboard's **Setup** tab:
   **Ask your home server**, or paste the link.)
3. Open **http://127.0.0.1:8077**. The dashboard is on this Mac only.

`install.sh` puts `tailbondd` and `mqvpn` in `/usr/local/bin`, the hooks in
`/usr/local/lib/tailbond/hooks`, the settings in
`/usr/local/etc/tailbond/tailbond.json`, the pairing and data usage in
`/usr/local/var/lib/tailbond`, the log in `/usr/local/var/log/tailbond.log`,
and the LaunchDaemon `/Library/LaunchDaemons/com.tailbond.tailbondd.plist`
(starts at boot, restarts if it stops). Running it again keeps your
settings and pairing.

### Homebrew

Each release also has a formula with its checksum. Homebrew only installs
formulae from a tap, so put it in a local one:

```sh
brew tap-new --no-git local/tailbond
curl -fsSLo "$(brew --repository local/tailbond)/Formula/tailbond.rb" \
  https://github.com/kaovilai/tailbond/releases/latest/download/tailbond.rb
brew install local/tailbond/tailbond
sudo brew services start tailbond
```

The formula's `sha256` pins that release's tarball. Settings live in
`$(brew --prefix)/etc/tailbond/tailbond.json`, the log in
`$(brew --prefix)/var/log/tailbond.log`.

### Build from source

For contributors, on an Apple silicon Mac with the Xcode command line tools
and Homebrew:

```sh
brew install libevent gpatch librsvg go
git clone --recurse-submodules https://github.com/kaovilai/tailbond
cd tailbond
scripts/build-mqvpn-macos.sh && agent/build.sh darwin/arm64 && packaging/macos/build-bundle.sh
tar -C out/macos -xzf out/macos/tailbond-macos-arm64.tar.gz
sudo out/macos/tailbond-macos-arm64/install.sh
```

`build-bundle.sh` makes `out/macos/tailbond-macos-arm64.tar.gz` and the
formula `out/macos/tailbond.rb`; the app icon is rendered from
`assets/brand/tailbond-icon.svg` during the build (rsvg-convert, iconutil),
not kept in git. Or let Homebrew build it: copy
`packaging/macos/tailbond.rb` into the tap as above and
`brew install --HEAD local/tailbond/tailbond` (a few minutes).

## Menu bar

The bundle installs **tailbond.app** in /Applications: an icon in the menu
bar (no Dock icon) that opens a small window over tailbond's own API
(127.0.0.1:8077):

- **State**: Connected (to which server, how many links), Connecting,
  Off, Not paired, or *tailbond isn't running*; the speeds; an on/off
  switch.
- **Mode**: Fastest, Smooth calls and video, Most reliable, Save mobile data.
- **Connections**: each link (Wi-Fi, Ethernet, an iPhone over USB, a
  nearby phone's or computer's share) with its delay and speed, standby, or
  what's wrong.
- **Share this Mac's connection** ([below](#sharing-this-macs-connection)):
  what it shares and with which hub, a switch, and **Sharing settings…**
  (the dashboard's choices).
- **Next to other VPNs** ([below](#using-tailbond-with-netbird-or-another-work-vpn)):
  NetBird or another VPN that's on, whether its traffic goes **Direct** or
  **Bonded** (a switch), work names on its DNS, the Tailscale app, and any
  warning. Hidden when there's nothing to say.
- **Updates**: the version, *up to date* or *Update available*, the
  policy (**Right away**, Automatic (night), Tell me, Off) as a menu, and
  **Check now**.
- After a crash, **Report a problem…** ([reporting.md](reporting.md));
  **Open dashboard**; **Open at login** (on after the first launch;
  System Settings → General → Login Items lists it); **Quit** (tailbond
  itself keeps running: it's the LaunchDaemon).

It's ad-hoc signed (no Apple Developer ID), like the binaries, and
install.sh clears the quarantine flag, so it opens without a Gatekeeper
prompt. It reads and changes only what the dashboard does; with a dashboard
password set, it asks you to sign in on the dashboard instead.
`/Applications/tailbond.app/Contents/MacOS/tailbond --status` prints the
same summary in Terminal. Homebrew installs keep it in
`$(brew --prefix)/opt/tailbond/tailbond.app` (the formula's caveats say how
to link it). Source: `macos/TailbondMenu` (SwiftUI `MenuBarExtra`, macOS
13+), built by `packaging/macos/build-app.sh`.

## Bond your connections

The **Connections** tab lists everything the Mac can use, by the name macOS
gives it (`networksetup -listallhardwareports`):

| Hardware port | Shows as | Kind |
|---|---|---|
| Wi-Fi | Wi-Fi | Wi-Fi |
| Ethernet, Thunderbolt Ethernet, USB 10/100/1000 LAN, a USB-C adapter | its port name | Ethernet |
| iPhone USB | iPhone (USB) | USB |
| Bluetooth PAN | Bluetooth tethering | USB |

Each gets the usual **Use it / Backup only / Off** and a monthly data cap.
**Use this Mac's connections** at the top switches all of them off at once:
then only the phones' shared data is bonded (and with no phone sharing,
tailbond connects over the Mac's default route).

### An iPhone over USB

1. On the iPhone: **Settings → Personal Hotspot → Allow Others to Join** on.
2. Plug it into the Mac with a cable; tap **Trust** if it asks. tailbond
   must be on (the switch at the top of its page).
3. **iPhone (USB)** appears under Connections within a few seconds and is
   bonded with the Wi-Fi. It's charged meanwhile, and it works where the
   hotel Wi-Fi keeps devices apart. To save its data for when the Wi-Fi
   fails, set the **iPhone (USB)** card's dropdown to **Backup**.
4. To stop: unplug it, or turn Personal Hotspot off. The Wi-Fi carries on
   alone.

This is the best way to use an iPhone with the Mac: the Mac bonds it
itself, so the iPhone needs no app. Your hotspot's Wi-Fi password still
protects it while "Allow Others to Join" is on; change it if you've shared
it.

### Phones sharing their mobile data

Phones on the **same Wi-Fi as the Mac** can lend their 5G/LTE
([Pair & Share](pair-and-share.md)):

- Android: tailbond → Settings → **Share my mobile data**.
- iPhone: **tailbond Share** → **Share mobile data** (keep it open on the
  screen, [ios.md](ios.md)).

They show up under **Shared by nearby devices**, each as a connection
labelled e.g. *Pixel's 5G (shared)* or *Mobile data from sam-pixel*, kind
`shared-cellular`. Another Mac or a Linux laptop sharing its connection
shows up there too, e.g. *Ethernet from tiger-mbp*. The Mac finds them with
Bonjour (mDNS, `_tailbond-share._udp`) and through the home server's list.
A share that stops answering for 3.5 s is dropped from the bond at once and
comes back when it does. The switch next to **Shared by nearby devices**
turns them all off.

### Sharing this Mac's connection

The Mac can be the one that lends ([Pair & Share](pair-and-share.md#computers-can-share-too)).
At a hotel: the Mac is on the travel router's Wi-Fi and the room has an
Ethernet cable. Plug it in, then on tailbond's page → **Share this Mac's
connection** → pick *USB 10/100/1000 LAN (en5)* (or whatever the adapter
is called) → **Start sharing**. The travel router bonds it as *Ethernet from
<this Mac's name>*. The menu bar shows who uses it and how much, and its
switch turns sharing off and on.

- Pick a connection the hub isn't on: sharing the Wi-Fi you share with the
  travel router only loops its traffic back through it.
- tailbond on the Mac doesn't have to be on. If it is, the Mac keeps using
  the Ethernet itself too; set it to **Off** under Connections to leave it
  to the router.
- The relay's traffic is pinned to that adapter (`IP_BOUND_IF`) and goes
  only to your home server. tailbond listens on UDP 47077 on the Mac's
  other networks; the kill switch blocks this, as it blocks phones' shares.

### Using a phone's bonded connection (phone hub)

The other way round: a phone already bonding its 5G with the hotel Wi-Fi
can let the Mac use that whole bonded connection as a proxy
([phone-hub.md](phone-hub.md)). Join the phone's hotspot or plug it in
over USB, then **Connections → Use a phone's connection → Use**. The
card says *Using Pixel's bonded connection (proxy)*, and the top of the
page says *via Pixel (proxy)*. The Mac's own tunnel can stay off. tailbondd points
the Mac's SOCKS proxy at its own forwarder (127.0.0.1:1889), which relays
to the phone with a login derived from the pass your home server gave this Mac for that phone.

- **Stop using it** puts your previous proxy setting back exactly. So does
  quitting tailbondd. The Mac may still be on the phone's hotspot
  afterwards: rejoin your usual Wi-Fi, and turn the hotspot off on the
  phone.
- USB tethering from an Android phone works on a Mac when the phone uses
  NCM (Pixel 6 and newer, most phones from 2021 on). Otherwise use the
  hotspot. After a crash or a restart, tailbondd puts it back
  when it starts.
- Only apps that follow the system proxy go through the phone, and only
  TCP traffic does: browsers yes, video calls and games no.

## Using tailbond with NetBird (or another work VPN)

tailbond and a work VPN can both be on: NetBird, the Tailscale app, or any
other VPN with its own utun. tailbond never touches their addresses, routes
or DNS. The dashboard says what it found, for example: *NetBird detected
(work VPN): its addresses and DNS are left alone · Work VPN traffic:
Direct*. It warns if something in the routing table steps on another VPN.

How tailbond stays out of the way:

- **Addresses.** NetBird's peers live in 100.64.0.0/10, like Tailscale's
  (often a /16 such as 100.92.0.0/16).
  - tailbond reaches your tailnet machines by host routes to their actual
    addresses: a /32 or /128 per machine from your home server's machine
    list, refreshed as it changes. It also routes its own tunnel subnet.
  - It never adds a route that equals or covers a route another VPN's
    interface owns, and never one to another VPN's own address.
  - A tailnet machine whose address falls inside NetBird's range still
    works: only that one address goes to tailbond.
  - tailbond's catch-all routes (`0/1`, `128/1`) are less specific than any
    VPN's own routes, so those keep winning.
- **The Tailscale app.** If it is running on the Mac, tailbond routes no
  tailnet machines at all: Tailscale reaches them itself. The dashboard
  says so.
- **DNS.** With another VPN present, tailbond never sets the Mac's global
  DNS servers. Instead it uses split DNS:
  - it writes `/etc/resolver/ts.net` (and one for your tailnet's own suffix
    when that isn't under ts.net), pointing at the tunnel's DNS;
  - every other name keeps the Mac's DNS, NetBird's match domains included;
  - the files are removed on disconnect, and after a crash at the next
    start (from a restore record);
  - a resolver file tailbond didn't write is never touched.
- **Work VPN traffic** (NetBird's own WireGuard UDP to its peers, relays,
  signal and management servers). Choose on the dashboard's *Other VPNs*
  card, or with `POST /api/coexist {"work_vpn": "direct"|"bonded"}`:
  - **Direct** (default): tailbond adds host routes for those endpoints via
    the physical uplink (en0 …), from `netbird status --json`, resolved and
    refreshed. This is the most compatible choice, but work traffic isn't
    bonded: it uses whatever connection the Mac is on.
  - **Bonded**: NetBird's traffic rides tailbond's bonded tunnel. For that,
    tailbond's tunnel MTU must be at least NetBird's interface MTU + 80, so
    tailbond raises its own MTU (and reconnects) when the uplink leaves room:
    the uplink's MTU minus tailbond's 100 bytes of overhead. If it can't,
    tailbond uses Direct and says why, as information next to the choice
    (`bonded_blocked` in the coexist status), not a warning: Direct works.
- **Kill switch.** With `client.kill_switch` on, tailbond allows NetBird's
  interface and (in Direct mode) its endpoints. This is the pf anchor
  `com.apple/249.tailbond`, evaluated before tailbond's own block.
- **Bypass rules.** IP-range and domain rules with the target *direct*
  always go straight out via the uplink, like the travel router's bypass
  feature: corporate ranges, work domains. Domains are resolved and kept as
  host routes.

Detection uses `netbird status --json` when the NetBird CLI is installed (a
logged-out NetBird is fine: it's simply not in the way yet). Without the
CLI, tailbond looks at which utun interfaces own which routes.
`"client": {"coexist": "off"}` turns all of this off; `"split_dns": true`
uses split DNS even with no other VPN.

**Not tested with a real NetBird login.** The CI uses a NetBird stand-in
(a wireguard-go utun on 100.92.0.0/16, a resolver for `corp.example`, a
`netbird status --json` stub) and the real NetBird package logged out.
Also not tested: NetBird's routed networks (its "networks"), its DNS route
management, or a real kill-switch session next to it.

## Hotel Wi-Fi that keeps devices apart

Many hotel and café networks block devices from talking to each other
(*client isolation*). Then phones on that Wi-Fi can't reach the Mac, and
tailbond says so: *This Wi-Fi blocks devices from talking to each other;
sharing isn't available here.* What works instead:

- **iPhone over USB** (above): no Wi-Fi involved.
- **A travel router** (GL.iNet, [glinet.md](glinet.md)): the router joins
  the hotel Wi-Fi, and the Mac and the phones join the router's own Wi-Fi,
  where nothing is isolated. The router can be the hub itself.
- **A phone's hotspot**: the Mac and the other phones join one phone's
  Personal Hotspot. Its data reaches tailbond as the Mac's Wi-Fi, and the
  other phones share theirs over that hotspot's network, where phone
  hotspots usually let devices reach each other.
- **Internet Sharing on the Mac** (System Settings → General → Sharing →
  Internet Sharing, from Ethernet or iPhone USB *to* Wi-Fi): the phones join
  the Mac's own Wi-Fi network. The Mac's Wi-Fi radio then can't also be a
  hotel uplink.

## Gatekeeper and quarantine

A browser marks downloads with the `com.apple.quarantine` attribute, and
Gatekeeper refuses to *open* unsigned quarantined programs from Finder.
`install.sh` removes the attribute from the bundle it installs, so nothing
else is needed. If you run the binaries by hand from a download, clear it
yourself:

```sh
xattr -dr com.apple.quarantine tailbond-macos-arm64
```

No Developer ID, notarization or paid Apple account is involved: Apple
silicon requires a code signature, and the Go and clang linkers add an
ad-hoc one. A Homebrew install isn't quarantined.

## Updates

**Updates & backup → When a new version is out**:

- A new bundle install updates **Right away**: it checks GitHub every hour
  and installs a new release as soon as you're not on a call or in a big
  download (at most 30 minutes later; security releases at once).
- **Automatic (night)** waits for 03:00–05:00.
- **Tell me** only shows it.

A Mac installed before this keeps its policy. The menu-bar app (coming)
shows the same choice. With the bundle install, **Update now** downloads
`tailbond-macos-arm64.tar.gz`, checks it against the release's `SHA256SUMS`
and runs its `install.sh --update`: the old binaries are kept as `.prev` and
put back if the new version isn't running within 60 s. By hand:

```sh
sudo /usr/local/lib/tailbond/install.sh --update
```

With Homebrew, tailbond can't replace its own files (Homebrew owns them),
so every policy means "tell me": the dashboard shows the command to run. `sudo brew services`
makes the installed files root-owned (Homebrew does that for every root
service), so give them back to yourself before `brew` changes them:

```sh
sudo brew services stop tailbond
sudo chown -R "$USER" "$(brew --prefix)/Cellar/tailbond"
curl -fsSLo "$(brew --repository local/tailbond)/Formula/tailbond.rb" \
  https://github.com/kaovilai/tailbond/releases/latest/download/tailbond.rb
brew reinstall local/tailbond/tailbond
sudo brew services start tailbond
```

## Settings

`tailbond.json` is the same file as on Linux ([architecture.md](architecture.md#agent-config-file));
on a Mac the defaults are `/usr/local` paths and `"listen":
"127.0.0.1:8077"`. Mac-relevant keys:

- `client.manage_routes` (default `true`): mqvpn routes all traffic through
  the tunnel (`0/1` and `128/1` via `utunN`, the server pinned to its
  gateway) and puts it back when it stops.
- `client.no_dns`: keep the Mac's DNS servers (by default mqvpn sets the
  tunnel's with `networksetup` and restores them).
- `client.kill_switch` (default off): nothing leaves outside the tunnel
  while tailbond runs (mqvpn's pf anchor). It blocks the LAN too, so
  phones' shares don't work with it on.
- `client.use_local` (default on): "Use this Mac's connections".
- `client.paths[].iface` may name any interface, also one tailbond doesn't
  list by itself (it's then used as named).

The hooks (`client-up`, `client-down` in `/usr/local/lib/tailbond/hooks`)
only flush the DNS cache: on macOS mqvpn sets routes and DNS itself.
Executables in `client-up.d/` etc. run too ([hooks](architecture.md#platform-hooks-keep-the-agent-platform-neutral)).

## Report a problem

**Report a problem** at the bottom of the dashboard or in its Logs tab (and a banner after a crash): a redacted report you check, then GitHub in your browser; the Mac sends nothing. [reporting.md](reporting.md).

## Uninstall

```sh
sudo /usr/local/lib/tailbond/uninstall.sh            # keeps settings and pairing
sudo /usr/local/lib/tailbond/uninstall.sh --purge    # removes them too
```

Homebrew:

```sh
sudo brew services stop tailbond
sudo chown -R "$USER" "$(brew --prefix)/Cellar/tailbond"   # the root service owned it
brew uninstall tailbond
```

Stopping tailbond stops mqvpn, which removes its routes, DNS servers and
kill-switch rules.

## Mac as home server

A Mac that stays at home (a Mac mini, an iMac, a MacBook that stays plugged
in with its lid open) can be the server your hubs bond to, instead of a
GL.iNet router, a Fedora box or a VPS. It does the same job as the
[Linux server](vps.md): it ends the bonded tunnel, sends the traffic on to
the internet and to your tailnet, and serves the dashboard, the pairing
over Tailscale and the "Get tailbond" page.

### Install the server

You need Tailscale on the Mac (for pairing over Tailscale, the dashboard
from your other devices, and the tailnet for bonded devices) and either a
public IPv4 address at home or a DNS name for it, with a UDP port forwarded
to the Mac on your router. **Start Tailscale before installing**: the
installer uses it only if `tailscale status` works then (see below).

Download the bundle and check it against `SHA256SUMS` as in
[The release bundle](#the-release-bundle) (step 1), or
[build it from source](#build-from-source), then, in the folder it's in:

```sh
tar -xzf tailbond-macos-arm64.tar.gz
sudo tailbond-macos-arm64/install.sh --role server --public-addr home.example.net
```

- `--public-addr`: what devices dial from away: your home's public IP or a
  DNS name for it. Without it, pairing links carry the Mac's home-network
  address, which only works at home.
- `--port N`: the tunnel's UDP port. Default: the first free of 443, 8443,
  4443, 10443, 50443 (an existing server keeps its own). Forward this port
  to the Mac on your router.
- `--subnet CIDR`: the tunnel's IPv4 subnet. Default: the first /24 from
  `10.78.0.0` that nothing on the Mac uses. `--subnet6` sets an IPv6 one.
- `--no-tailscale`: don't use Tailscale even if it's running (then no
  tailnet for bonded devices and no dashboard from other devices).

The dashboard is `http://127.0.0.1:8077` on the Mac and
`http://<mac's tailscale name>:8077` from your tailnet; add devices on its
**Devices** tab, or let them ask over Tailscale and approve them there. The
"Get tailbond" page is `http://<mac's tailscale name>/`. Same files as the
hub (`/usr/local/...`, the LaunchDaemon `com.tailbond.tailbondd`); updates
(**Right away** by default, `install.sh --update`), backups (dashboard
**Updates & backup**, or `tailbondd backup`/`restore`) and the uninstaller
work as for the hub. Homebrew installs the hub only: use the bundle for a
server.

### What it changes on the Mac

While the server runs, the `server-up` hook
(`/usr/local/lib/tailbond/hooks/server-up`):

- turns on IP forwarding (`net.inet.ip.forwarding`, and
  `net.inet6.ip6.forwarding` with an IPv6 subnet);
- loads the pf anchor `com.apple/251.tailbond.server` (macOS's own
  `/etc/pf.conf` evaluates `com.apple/*`; pf's main ruleset is never
  replaced: if another firewall loaded one without Apple's anchors, the
  hook adds nothing and says so in the log) and enables pf with a reference
  token: bonded devices' traffic is NATed out the default route's interface
  (and the Mac's other Ethernet and Wi-Fi ports, so a switch between them
  keeps working), and out Tailscale's interface toward `100.64.0.0/10` and
  `fd7a:115c:a1e0::/48`; tailnet machines reach bonded devices through the
  advertised route without NAT; the dashboard (tcp/8077) and the phone page
  (tcp/80), on the addresses tailbond serves them on, answer only loopback,
  bonded devices on the tunnel and tailnet addresses (a web server of your
  own on another address keeps its port 80). tailbondd itself also refuses
  other sources on those addresses, so this holds while the server is
  stopped too;
- with the macOS firewall on (System Settings → Network → Firewall), allows
  incoming connections for `mqvpn` and `tailbondd`. "Block all incoming
  connections" would stop phones and routers from connecting: the hook warns
  about it;
- holds a power assertion (`caffeinate -i -s`) so the Mac doesn't idle-sleep;
- (not the hook: mqvpn itself) TCP from bonded devices travels in mqvpn's
  TCP lane in the default modes, and the server opens the outgoing TCP
  connection from the Mac's own address; UDP, ICMP and the rest are
  forwarded and NATed by the pf rules above;
- advertises the tunnel subnet in Tailscale (`tailscale set
  --advertise-routes`, keeping the routes already advertised). Approve it
  once in the admin console if your tailnet machines should reach bonded
  devices; bonded devices reach the tailnet without it.

Tailscale is used only if it was running when you installed (as on Linux:
the server's certificate, which devices pin, depends on it). Installed
without it? Start Tailscale, then run `install.sh --role server` again; it
switches Tailscale on, and devices paired before then pair again (the
certificate changes). With Tailscale on, when it starts after tailbond
(after a restart, the apps start at login, tailbond at boot) or its
interface changes, tailbondd notices within about 20 s and runs the hook
again for the pf rules and the route. With `--no-tailscale`
(`"disable_tailscale": true`) the hook leaves Tailscale alone: no tailnet NAT,
no route advertised.

Everything except the route and the firewall entries is written first to
`/usr/local/var/lib/tailbond/server-net.state`. `server-down` (the server
stops) puts it all back from there: the anchor emptied, pf's reference
released (pf goes back off if nothing else uses it), the forwarding values
as they were before, the assertion dropped. After a crash (tailbondd killed),
the next start first restores from the record, and `uninstall.sh` does too.

### Keep it awake

A sleeping Mac answers nobody: bonded devices lose the server until it
wakes.

- tailbond holds off **idle** sleep while the server runs. The display can
  still turn off. **Setup health** on the dashboard says whether the Mac is
  being kept awake.
- **A MacBook sleeps when its lid closes**, whatever tailbond does, unless
  it's in closed-display mode (power adapter, external display and keyboard
  connected). Keep the lid open and the charger in. On battery, Setup
  health flags it.
- A desktop Mac also benefits from System Settings → Energy → **Prevent
  automatic sleeping when the display is off** and **Start up automatically
  after a power failure** (`sudo pmset -a autorestart 1`); tailbond starts
  at boot by itself (LaunchDaemon).

### Tailscale on a Mac server

Tailscale's docs say all three macOS variants (the Standalone app, the App
Store app and the open-source `tailscaled`) can be a subnet router
([Subnet routers](https://tailscale.com/docs/features/subnet-routers),
[the three variants](https://tailscale.com/docs/concepts/macos-variants)).
tailbond calls the `tailscale` CLI on `PATH`, else the one inside
`/Applications/Tailscale.app`, as root from its LaunchDaemon. What that
means per variant:

| Variant | Runs before login | tailbond as a server |
|---|---|---|
| Standalone app (Tailscale's `.pkg`, recommended by Tailscale) | no | should work while you're logged in; **not tested** |
| Mac App Store app | no (sandboxed) | should work while you're logged in; `tailscale cert` may not reach the sandbox, so tailbond falls back to a self-signed (pinned) certificate; **not tested** |
| Open-source `tailscaled` (a LaunchDaemon) | yes | the best fit for a server that reboots unattended; **not tested** |

With either app, a Mac that restarts and sits at the login screen has no
Tailscale until someone logs in: tailbond runs, bonded devices still reach
the internet, but not the tailnet, and the dashboard and pairing over
Tailscale wait. CI uses a stand-in for Tailscale (a fake CLI and a utun
with a tailnet address behind it), not any real variant.

### What CI checks, and what it doesn't

CI (job `macos-server`, GitHub's `macos-latest` arm64 runner) runs
`scripts/e2e-macos-server.sh`: the Mac is the server (mqvpn's server mode
on a utun and its TCP egress,
[patch 0006](../patches/mqvpn/0006-darwin-server-mode.patch), with the hooks
above), and a Linux client in a QEMU VM bonds two network
interfaces to it. It checks that:

- Tailscale starting after the server: the tailnet NAT and the route
  follow it;
- a pair request from a tailnet address is accepted (whois), approved and
  delivered; the client connects and the server sees two active paths;
- the client reaches the internet over TCP (through the server's TCP
  egress) and UDP (through the Mac's NAT: pf's state table shows the
  translation), a tailnet machine through the tunnel (it sees the
  Mac's Tailscale address), and the server's DNS forwarder; a tailnet
  machine reaches the client through the advertised route; the subnet is
  advertised;
- the dashboard answers the tailnet and, through the tunnel, the client;
  the phone page (on the Tailscale address only) answers the tailnet; both
  refuse other sources, also after the server is stopped; with the macOS
  firewall on, mqvpn is allowed in; another firewall's pf anchor, its pf
  reference and pf's main ruleset survive tailbond's start, stop and
  crashes; with Tailscale disabled the hook adds no tailnet NAT and
  advertises nothing;
- a power assertion is held and Setup health says so; the server's
  settings back up and restore;
- pf, both forwarding sysctls and the power assertion are back to how they
  were after a stop, after `kill -9` and a restart, and after `kill -9` and
  the uninstaller's restore.

`scripts/test-macos-install-helpers.sh` checks the installer's port and
subnet choices on canned macOS output (a browser's QUIC to some server's
443 doesn't hold udp/443; a `10/8` route rules out every `10.N.0.0/24`).
`scripts/test-macos-install.sh` (job `macos`) installs with `--role server`
under launchd, checks the anchor and forwarding, updates, and uninstalls.

Not tested:

- **A real Mac behind a real home router** (port forwarding, a public
  address, real phones and travel routers connecting from away).
- **Any real Tailscale variant** on the Mac (above).
- **IPv6** through the Mac (the hooks NAT an IPv6 subnet; CI's is IPv4 only).
- **Sleep** itself: CI checks that the assertion is held, not that a real
  Mac stays awake through a night, nor clamshell mode.

## Limits and what isn't tested

CI (job `macos`, GitHub's `macos-latest` arm64 runner) builds the patched
mqvpn and runs its Darwin unit tests, runs tailbondd's Go tests on darwin,
and runs `scripts/e2e-macos.sh`: the hub on the runner bonds an
interface path and two phones' shares (the reference relay, on a
fake-Ethernet LAN) to a tailbond server in a small Linux VM under QEMU. The
server sees the Mac with 3 paths, traffic crosses all of them, a phone that
stops drops out within ~4 s while the tunnel keeps working, and the client
isolation note appears. A second tailbondd on the runner, with its own
tunnel off, shares `lo0` with the hub: the hub finds it over Bonjour and
the server sees it as one more path that carries traffic. It also installs
the bundle (`install.sh`, `--update`, `uninstall.sh`), runs the update path
end to end against a fake GitHub release (`scripts/e2e-macos-update.sh`:
the update API that the dashboard's **Update now** calls, with the detached
`install.sh --update` restarting the LaunchDaemon, the rollback of a
release that doesn't start, `install.sh --update` alone, a refused bad
`SHA256SUMS`), and the Homebrew formula.

What that can't cover:

- **The dashboard UI itself.** The update test drives the API behind **Update now** (what the button calls), not the button, its progress text or the restart nag in a browser.
- **The interface path in CI is `lo0`.** The runner has one network
  interface, and the only way it reaches the server VM is QEMU's port
  forwarding on 127.0.0.1, so the hub's IP_BOUND_IF-pinned path is pinned to
  the loopback interface. Wi-Fi, Ethernet adapters and an iPhone over USB
  as separate paths, and pulling one out, are not tested on a real Mac yet
  (mqvpn's Darwin layer is the same as upstream's there). The same goes for
  sharing: CI shares `lo0`, not an Ethernet adapter, and the Mac's own
  tunnel is off while it does (one mqvpn per Mac).
- **`manage_routes` and DNS** are off in the CI run: routing the runner's own
  traffic through the tunnel would cut it off from GitHub. mqvpn's Darwin
  routing, DNS and kill switch are covered by its unit tests only.
- **Real phones** sharing on real Wi-Fi with a Mac hub; Bonjour across a
  real access point; macOS's firewall and Local Network privacy with a root
  LaunchDaemon (daemons aren't subject to the Local Network prompt; not
  verified).
- **Intel Macs**: no x86_64 build (upstream mqvpn doesn't build or test one
  either). A universal binary would need libevent built for both
  architectures.
- **Updates** are tested with the same bundle (`--update --force`); the
  first real release with a macOS asset is the first end-to-end check of
  "Update now" on a Mac.
