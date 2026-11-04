# Coming from Tailscale

tailbond doesn't replace your tailnet: your **server** keeps running
Tailscale, and bonded devices reach the tailnet through it. What changes
depends on the device:

| Device | With the Tailscale app | Shows in the Tailscale admin console as |
|---|---|---|
| Home server (GL.iNet Flint 3, Fedora) or VPS | **Run both.** Keep Tailscale on; tailbond uses it | The same one machine as before, now advertising tailbond's tunnel subnet as a route |
| Android phone | **One or the other at a time.** Android runs one VPN; tailbond takes over from Tailscale while it's connected | Its existing entry, offline while tailbond is on. Nothing new is added |
| GL.iNet travel router (Slate 7) | **Run both.** GL's built-in Tailscale stays on | Its existing entry, if it has one. Nothing new is added |
| iPhone (tailbond Share) | **Run both.** The Share app isn't a VPN | Unchanged |
| Mac (Apple silicon) | **Run both.** tailbond keeps out of the Tailscale app's and a work VPN's (NetBird's) routes and DNS. Next to NetBird: CI-tested. Next to the Tailscale app: designed for, not tested yet | Its existing entry, if the Tailscale app is on it. Nothing new is added |

Bonded devices never appear as machines of their own. To the rest of your
tailnet their traffic comes **from the server** (it's masqueraded), so
Tailscale ACLs, `tailscale whois` and "last seen" all see the server, not the
phone or router behind it.

Use one home server: a Flint 3 **or** a Fedora box, not both. A VPS can be a
second, failover server.

## Exit nodes: tailbond already is one

By default a bonded device sends **all** its internet traffic through
tailbond to your server, which is what a Tailscale exit node does. You don't
need an exit node on top, and on a Mac or travel router the two would fight
over the same routes.

What you get, the same as with an exit node at home:

- **Safe on hotel or café Wi-Fi.** Everything leaves the device inside one
  encrypted tunnel (QUIC, which uses TLS 1.3) to your server. The Wi-Fi's
  owner sees only encrypted UDP to your server's address: not the sites you
  visit, not your DNS lookups. The device checks the server's certificate
  (from `tailscale cert`, or the one pinned when you paired), so a fake
  hotspot can't pretend to be your server.
- **You appear to be at home.** Websites see your server's public IP: your
  home's, or the VPS's while a VPS is in use. Banking, streaming and
  region-locked sites behave as they do at home.
- **DNS goes home too.** Lookups go to your server's DNS, not the hotel's.
  The exception is a Mac that also runs the Tailscale app or a work VPN:
  there tailbond only takes `ts.net` names and leaves every other lookup on
  the Mac's existing DNS settings. If those point at the hotel's DNS server,
  the hotel can see the names you look up (not what you send). To avoid
  that, set the Mac's DNS yourself (for example the other VPN's DNS, or
  an encrypted DNS profile), or turn coexistence off with
  `"client": {"coexist": "off"}` so tailbond takes over DNS, at the cost of
  the other VPN's names ([details](macos.md#using-tailbond-with-netbird-or-another-work-vpn)).
- **Your tailnet still works**: tailnet machines are reached through the
  server.

What's different from an exit node:

- **It's on by default.** Tailscale's exit node is off until you pick one.
  On Android you can change this: **Settings → Internet traffic → Only
  Tailscale machines** sends just tailnet traffic through tailbond, and
  everything else goes out directly.
- **You can't choose another exit.** Traffic leaves from your tailbond
  server (or the VPS it fails over to), not from an exit node elsewhere on
  your tailnet. To browse from another place, use that exit node in the
  Tailscale app with tailbond off, or set up tailbond on a VPS there.
- **If the tunnel drops:**
  - **Travel router:** its kill switch (on by default) holds traffic back
    rather than letting it out on the hotel Wi-Fi.
  - **Mac:** the kill switch is off by default (`client.kill_switch`; it
    also blocks phones' shares), so traffic can go out directly until
    tailbond reconnects.
  - **Android:** turn on **Always-on VPN** and **Block connections without
    VPN** for tailbond in Android's VPN settings if you want the same.
- **Exceptions you set:** bypass rules (a banking site, an app, a device) go
  out directly by design.

Don't turn on a Tailscale exit node while tailbond is connected:

- **Android:** it can't happen. One VPN runs at a time, so turning on
  Tailscale (and its exit node) turns tailbond off.
- **Mac:** both would claim the Mac's default route, and which one wins
  isn't defined. Your traffic could leave through the exit node instead of
  the bond, or go into one tunnel and then the other. Use one at a time.
- **Travel router:** don't set an exit node in GL's Tailscale settings while
  tailbond is on, for the same reason.

## Home server or VPS: run both

1. Turn Tailscale on first: GL.iNet **APPLICATIONS → Tailscale**; Fedora or a
   VPS `sudo tailscale up`.
2. Install tailbond ([GL.iNet](glinet.md), [Fedora](fedora-server.md),
   [VPS](vps.md)). It finds Tailscale and advertises the tunnel subnet next to
   any routes you already advertise.
3. In the [admin console](https://login.tailscale.com/admin/machines),
   approve the tunnel subnet route on the server, once (`10.77.0.0/24` on
   a Flint 3 or Fedora; a VPS picks a free one, shown on its dashboard)
   ([subnet routers](https://tailscale.com/docs/features/subnet-routers)).
   Until then, bonded devices can still reach the tailnet, but tailnet
   machines can't reach the bonded devices.
4. Forward UDP 443 to the server, plus UDP 41641 for direct Tailscale
   connections ([setup tips](setup-tips.md)).

Don't remove the server's Tailscale later: bonded devices lose the tailnet,
and new devices lose the "Ask to connect" pairing.

## Android phone: one VPN at a time

Do these in order. Tailscale stays installed; you switch between the two.

1. **With Tailscale on**, type your server's name with a slash
   (`homebox/`) into the phone's browser and install tailbond from that page.
2. In tailbond, **Ask to connect** to your server, again with Tailscale on:
   the request travels over Tailscale. Approve it on the server.
3. Tap **Connect** in tailbond. Android hands the VPN over: the Tailscale app
   shows it's disconnected. Your tailnet machines still answer, through the
   server.
4. In Android's VPN settings (**Settings → Network & internet → VPN**, the
   gear next to Tailscale), turn **Always-on VPN** and **Block connections
   without VPN** off, so Tailscale doesn't take
   the VPN back.
5. Leave the phone's entry in the admin console; don't remove it. It's how
   you go back, and how you re-pair.

**Going back to Tailscale:** Disconnect in tailbond, then Connect in the
Tailscale app.

Sharing this phone's data with a hub (**Share my mobile data**) needs no VPN,
so it works with tailbond connected or off. With the Tailscale VPN on
instead, sharing hasn't been tested yet.

## GL.iNet travel router (Slate 7): run both

1. Turn on GL's Tailscale (**APPLICATIONS → Tailscale**) and join the same
   tailnet as the server, so the router can find it.
2. Install tailbond and open **VPN → tailbond**: **Found your home server →
   Ask to connect**, then approve it on the server ([GL.iNet
   guide](glinet.md#pairing-over-tailscale)).
3. Keep Tailscale on. Devices on the router's Wi-Fi reach the tailnet through
   the bonded tunnel; the router itself keeps using its own Tailscale.

Without Tailscale on the router, paste a pairing link instead (**Other ways
to connect**). GL's hotel **Login Mode** pauses GL's Tailscale while you sign
in; tailbond steps aside too and comes back after.

## iPhone (tailbond Share): run both

The Share app lends mobile data to a hub over Wi-Fi; it isn't a VPN, so the
Tailscale app can stay on. See [ios.md](ios.md).

## Mac

tailbond runs as a background service with a menu bar icon
([macos.md](macos.md)). It's built and tested on macOS runners in CI
against a Linux server; **it hasn't run on a real Mac yet**.

**Order:**

1. Optional: keep the Tailscale app on. With it, the Mac can ask your server
   to connect over Tailscale. Without it, pair with a link from the
   server's dashboard (**Devices → Add a device**) instead.
2. Install tailbond (the [bundle or Homebrew](macos.md#install), or the
   server's *Get tailbond* page, `homebox/`, **Mac** tab).
3. On `http://127.0.0.1:8077`, **Setup → Ask your home server** (or paste
   the link), then approve it on the server. tailbond connects and the menu
   bar icon shows the links it bonds.

**Next to the Tailscale app.** Leave both on:

- tailbond routes no tailnet machines while the Tailscale app runs: the app
  reaches them itself, as before, and the dashboard says so. Without the
  app, tailbond reaches them through your server.
- tailbond never adds a route that equals or covers one the Tailscale app
  owns, and uses split DNS (only `ts.net` names go to the tunnel's DNS), so
  MagicDNS keeps working through the app.
- **Exit node:** you don't need one: tailbond already sends the Mac's
  internet traffic out through your server, encrypted
  ([what that gives you](#exit-nodes-tailbond-already-is-one)). Don't turn
  one on while tailbond is connected: both would claim the default route.

Unit tests cover the Tailscale-app rules; the macOS CI runs without the
Tailscale app, so **running both is untested**, and so is an exit node.

**Next to NetBird (or another work VPN).** Leave both on. NetBird keeps its
addresses, routes and match-domain DNS. Its own traffic goes **Direct**
(the Mac's own connection, the default) or **Bonded**, your choice in the
menu bar or on the dashboard ([details](macos.md#using-tailbond-with-netbird-or-another-work-vpn)).
CI checks this on a macOS runner against a NetBird stand-in (a WireGuard
interface on 100.92.0.0/16 with a `corp.example` resolver) and the real
NetBird app logged out: tailbond's routes stay clear of NetBird's network,
work names keep NetBird's DNS, the global DNS is untouched, NetBird's
endpoints go Direct, and Bonded raises the MTU or falls back to Direct.
**A real NetBird login hasn't been tested.**

**Switching back:** turn tailbond off with the menu bar switch (or
**Disconnect** on the dashboard). mqvpn puts the Mac's routes and DNS back,
and tailbond removes its resolver files: CI checks both, also after the
agent is killed. To remove it: `sudo /usr/local/lib/tailbond/uninstall.sh`.
The Tailscale app and NetBird need nothing changed either way.
