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
| Mac | In progress: not released yet | |

Bonded devices never appear as machines of their own. To the rest of your
tailnet their traffic comes **from the server** (it's masqueraded), so
Tailscale ACLs, `tailscale whois` and "last seen" all see the server, not the
phone or router behind it.

Use one home server: a Flint 3 **or** a Fedora box, not both. A VPS can be a
second, failover server.

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

The Mac hub isn't released yet. Its guide will say how it runs next to the
Tailscale app.
