# tailbond on a Mac

Your MacBook as the **hub**: tailbond bonds the Mac's Wi-Fi, its Ethernet or
USB-C adapter, an **iPhone plugged in over USB**, and the mobile data that
**nearby phones share** ([Pair & Share](pair-and-share.md)) into one tunnel
to your home server. It's the same client role as the travel router and a
Linux laptop: the same dashboard, the same settings, the same tunnel.

- Apple silicon (arm64) Macs. Intel Macs: not built (see [Limits](#limits-and-what-isnt-tested)).
- No paid Apple account: the binaries are unsigned (ad-hoc signed by the
  linker, which is all Apple silicon needs to run them), and you install
  them yourself. Gatekeeper only gets in the way of software that's
  *downloaded* and then double-clicked; the installer below takes care of
  that ([Gatekeeper](#gatekeeper-and-quarantine)).
- Runs as a LaunchDaemon (root): mqvpn opens a `utun` tunnel device and sets
  the routes and DNS servers, like any VPN.

## Install

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

1. Download `tailbond-macos-arm64.tar.gz` from the
   [latest release](https://github.com/kaovilai/tailbond/releases/latest).
2. In Terminal:

   ```sh
   cd ~/Downloads && tar -xzf tailbond-macos-arm64.tar.gz
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

The copy in this repository (`packaging/macos/tailbond.rb`) builds from
source instead: copy it into the tap the same way and
`brew install --HEAD local/tailbond/tailbond` (Xcode command line tools, a
few minutes). Settings live in
`$(brew --prefix)/etc/tailbond/tailbond.json`, the log in
`$(brew --prefix)/var/log/tailbond.log`.

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
2. Plug it into the Mac with a cable; tap **Trust** if it asks.
3. **iPhone (USB)** appears under Connections within a few seconds and is
   bonded with the Wi-Fi. It's charged meanwhile, and it works where the
   hotel Wi-Fi keeps devices apart. To save its data for when the Wi-Fi
   fails, set the **iPhone (USB)** card's dropdown to **Backup only**.
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

They show up under **Shared by nearby phones**, each as a connection
labelled e.g. *Pixel's 5G (shared)*, kind `shared-cellular`. The Mac finds
them with Bonjour (mDNS, `_tailbond-share._udp`) and through the home
server's list. A phone that stops answering for 3.5 s is dropped from the
bond at once and comes back when it does. The switch next to **Shared by
nearby phones** turns them all off.

### Using a phone's bonded connection (phone hub)

The other way round: a phone already bonding its 5G with the hotel Wi-Fi
can let the Mac use that whole bonded connection as a proxy
([phone-hub.md](phone-hub.md)). Join the phone's hotspot or plug it in
over USB, then **Connections → Use a phone's connection → Use**. The
card says *Using Pixel's bonded connection (proxy)*, and the top of the
page says *via Pixel (proxy)*. The Mac's own tunnel can stay off. tailbondd points
the Mac's SOCKS proxy at its own forwarder (127.0.0.1:1889), which relays
to the phone with a password derived from your home server's share key.

- **Stop using it** puts your previous proxy setting back exactly. So does
  quitting tailbondd. After a crash or a restart, tailbondd puts it back
  when it starts.
- Only apps that follow the system proxy go through the phone, and only
  TCP traffic does: browsers yes, video calls and games no.

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

The dashboard checks GitHub for new releases once a day (**Updates &
backup**). With the bundle install, **Update now** downloads
`tailbond-macos-arm64.tar.gz`, checks it against the release's `SHA256SUMS`
and runs its `install.sh --update`: the old binaries are kept as `.prev` and
put back if the new version isn't running within 60 s. By hand:

```sh
sudo /usr/local/lib/tailbond/install.sh --update
```

With Homebrew, the dashboard shows the command to run. `sudo brew services`
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

## Limits and what isn't tested

CI (job `macos`, GitHub's `macos-latest` arm64 runner) builds the patched
mqvpn and runs its Darwin unit tests, runs tailbondd's Go tests on darwin,
and runs `scripts/e2e-macos.sh`: the hub on the runner bonds an
interface path and two phones' shares (the reference relay, on a
fake-Ethernet LAN) to a tailbond server in a small Linux VM under QEMU. The
server sees the Mac with 3 paths, traffic crosses all of them, a phone that
stops drops out within ~4 s while the tunnel keeps working, and the client
isolation note appears. It also installs the bundle (`install.sh`,
`--update`, `uninstall.sh`) and the Homebrew formula.

What that can't cover:

- **The interface path in CI is `lo0`.** The runner has one network
  interface, and the only way it reaches the server VM is QEMU's port
  forwarding on 127.0.0.1, so the hub's IP_BOUND_IF-pinned path is pinned to
  the loopback interface. Wi-Fi, Ethernet adapters and an iPhone over USB
  as separate paths, and pulling one out, are not tested on a real Mac yet
  (mqvpn's Darwin layer is the same as upstream's there).
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
