# Round 1: persona A, traveller (Mac + Pixel + iPhone)

I use Tailscale every day, I travel a lot, and I'm fine with apps. I don't
know what SOCKS, PAC or TCP mean. I only had the screenshots and the doc
excerpts listed in screens.md.

## Task 1: at a hotel, put the Mac on the Pixel's bonded connection

**Steps**

1. On the Mac I open the tailbond dashboard, Connections tab
   (web-08-mac-phone-hub-none.png). The bottom card, "Use a phone's bonded
   connection", says "No phone hub found. On the phone: tailbond → Share
   bonded connection. Then join its hotspot, or plug it in over USB".
2. On the Pixel I open tailbond and look for "Share bonded connection".
   The hint doesn't say where it is, so I go into Settings
   (share-01-settings-row.png). No row is called that. The nearest is
   "Lend your mobile data → Share my mobile data · Not sharing", so I tap
   it.
3. The screen is titled "Share my mobile data" (hub-01-off.png). The first
   big switch is also "Share my mobile data". I scroll past it and turn on
   the second switch, "Share bonded connection".
4. It says "Ready for your laptop: On the Mac: join this phone's hotspot or
   plug it in over USB, then tailbond → Use a phone's bonded connection"
   (hub-02-ready.png). I turn on the Pixel's hotspot in Android quick
   settings and join it from the Mac's Wi-Fi menu. Can the Pixel stay on
   the hotel Wi-Fi with its hotspot on? Nothing tells me.
5. On the Mac the card now shows "Pixel 9 · found on this network" with a
   **Use** button and a checkbox, "Keep local and tailnet addresses direct
   (proxy auto-config)" (web-08-mac-phone-hub-found.png). I leave it
   unticked because I don't know what it means, and click **Use**.
6. The card turns green: "Using Pixel 9's bonded connection (proxy) …
   46.0 MB up, 870 MB down" (web-08-mac-phone-hub-using.png). On the Pixel
   the card turns teal and shows 1 device, 960.2 MB, listed by IP address
   (hub-03-sharing.png).
7. To undo it I click **Stop using it** on the Mac, turn off "Share bonded
   connection" on the Pixel, turn off the hotspot, and rejoin the hotel
   Wi-Fi on the Mac. No screen lists the last two steps; I worked them out
   myself.
8. If the phone goes away, the Mac shows a clear orange banner, "Lost the
   phone hub, so the proxy is off again…", with **Look again**
   (web-08-mac-phone-hub-lost.png).

**Where I hesitate**

- Finding the switch on the Pixel. The Mac says "tailbond → Share bonded
  connection", but it's under Settings → Lend your mobile data → Share my
  mobile data.
- Two very similar switches on one screen. "Share my mobile data" comes
  first and is bigger. In the dark render it sits in its own card, so it
  looks even more like the main control (hub-04-sharing-dark.png).
- Do I need the hotspot at all? If the Mac is on the Pixel's hotspot, isn't
  it already using the phone? Nothing explains that the hotspot alone gives
  only mobile data, not the bonded connection, so **Use** is still needed.
- The checkbox: tick it or not? Will Tailscale (my NAS, MagicDNS names)
  still work if I leave it unticked?
- While it works, the Mac dashboard says "Disconnected" twice, plus "No
  connections yet: Waiting for mqvpn to bring up its paths". Is it working
  or not? Should I flip the Disconnected switch on?
- Battery and monthly-limit chips: do they protect my 5G plan from the Mac,
  or only the other feature?

**Findings**

- **High** (android/hub-01-off.png, share-01-settings-row.png): the
  feature lives inside a screen titled "Share my mobile data", under a row
  "Share my mobile data · Not sharing", under the heading "Lend your mobile
  data". Its own switch comes second, below a bigger look-alike. I'd likely
  turn on the wrong switch first and then wonder why the Mac finds nothing.
  The Settings row also still says "Not sharing" when bonded sharing is on.
  *Suggestion:* give the Settings row a name that covers both, e.g.
  "Share with a laptop or other phones", with a status that reflects either
  switch. Put "Share bonded connection" first on the screen, or split the
  two into clearly separate sections with headings like "For your Mac" and
  "For a travel router".
- **Medium** (web/web-08-mac-phone-hub-none.png): the Mac's hint for
  Android, "tailbond → Share bonded connection", skips the Settings → Share
  my mobile data steps. The line just above it, for nearby phones, does
  give the full path. *Suggestion:* "On Android: tailbond → Settings →
  Share my mobile data → Share bonded connection".
- **Medium** (android/hub-02-ready.png, web-08-mac-phone-hub-none.png): it
  never explains why the hotspot is needed, or that joining it is not
  enough by itself. I might join the hotspot and stop there, thinking I'm
  done, while really using only the Pixel's 5G. Nobody says whether the
  Pixel can stay on hotel Wi-Fi with its hotspot on. *Suggestion:* add one
  line on the phone: "Your Mac needs a way to reach this phone: its hotspot
  (it stays on the hotel Wi-Fi) or a USB cable. Then press Use on the Mac."
  Add a button that opens Android's hotspot settings.
- **Medium** (web/web-08-mac-phone-hub-using.png): while the proxy works,
  the page says "Disconnected" (header switch and status pill), "No
  connections yet", "Waiting for mqvpn to bring up its paths", and "mqvpn
  control API unavailable". That reads as broken. The en0 Wi-Fi card from
  the other renders is also gone. *Suggestion:* when a phone hub is in
  use, show it in the header ("Via Pixel 9's bonded connection") and hide
  or soften the Mac's own tunnel status. Never show "mqvpn" to users.
- **Medium** (web/web-08-mac-phone-hub-using.png): "Turned off when tailbond
  stops" is unclear because the header switch already reads "Disconnected".
  Does flipping it stop the proxy, or start something that competes with
  it? *Suggestion:* say what turns it off in UI terms ("…when you quit
  tailbond or the phone goes away"). Say whether the Mac's own tailbond
  switch should stay off.
- **Medium** (web/web-08-mac-phone-hub-found.png): "Keep local and tailnet
  addresses direct (proxy auto-config)" is the one choice I have to make,
  and it's in jargon. I can't tell which is safe for Tailscale. *Suggestion:*
  "Keep Tailscale and local devices (printer, NAS) on this Mac's own
  connection (recommended)". Tick it by default, or explain what happens if
  it's left unticked.
- **Medium** (android/hub-02-ready.png, web-08-mac-phone-hub-using.png):
  "Only apps that follow the laptop's proxy setting, and only TCP" means
  nothing to me. The Mac's version ("browsers, most apps … not video calls,
  games") is better. *Suggestion:* drop "proxy setting" and "TCP" on both
  screens: "Works for web browsing and most apps. Video calls and games
  keep using the Mac's own connection."
- **Medium** (android/hub-03-sharing.png): "Who's using it: Turn sharing on
  to see who uses it" appears while sharing is on and a Mac is using it.
  "0 B shared this month" sits next to 960.2 MB. The battery and monthly-cap
  chips sit below both features, and I can't tell if they cap the Mac.
  *Suggestion:* scope those sections ("For mobile-data sharing") or make
  them apply to both. Count the bonded traffic in the monthly total.
- **Medium** (all four web-08 renders, docs/macos.md "Using a phone's bonded
  connection"): undoing it only says **Stop using it**. Nothing reminds me
  to turn off sharing on the Pixel, turn off its hotspot, or put the Mac
  back on the hotel Wi-Fi. *Suggestion:* after Stop, show "Done. You can
  turn off the Pixel's hotspot and rejoin your Wi-Fi."
- **Low** (android/hub-05-no-tunnel.png): "Waiting for tailbond to connect.
  Turn tailbond on." has no button. *Suggestion:* add a "Turn on tailbond"
  button.
- **Low** (android/hub-03-sharing.png): the Mac is listed as
  "192.168.43.112". *Suggestion:* show the device name ("Tiger's MacBook
  Air") and keep the IP as small text under it.
- **Low** (android/share-01-settings-row.png and web-08-mac-phone-hub-found.png):
  the phone calls itself "pixel-8"; the Mac lists "Pixel 9". If that's
  real naming, not test data, I'd wonder whether it's the same phone.
  *Suggestion:* use the same name on both sides.
- **Low** (all web-08 renders): the sticky header (logo, "Bonded to" with
  nothing after it, Disconnected switch) floats over the mode cards and
  tabs and hides "Connections / Setup / …". This may be a screenshot
  artifact, but it looks broken. *Suggestion:* check the full-page capture,
  and fill in or hide the empty "Bonded to".
- **Low** (web/web-08-mac-phone-hub-none.png): "A VPN with a kill switch can
  block this; allow local network access in it." Is Tailscale such a VPN?
  *Suggestion:* name the common case ("Tailscale exit node with 'Allow LAN
  access' off…") or link to a short help note.
- **Low** (web/web-08-mac-phone-hub-lost.png): the lost-phone banner is
  clear and calm. Keep it.

**Verdict:** I'd get there, but only after opening the wrong screen, maybe
turning on the wrong switch, and doubting it worked because the Mac says
"Disconnected" the whole time.

## Task 2: use the iPhone as the hub over USB

**Steps**

1. I look at the same Mac card, "Use a phone's bonded connection". It says
   "plug it in over USB (iPhone: Personal Hotspot on)" and "On the phone:
   tailbond → Share bonded connection" (web-08-mac-phone-hub-none.png).
   So I assume the iPhone works like the Pixel.
2. On the iPhone I open tailbond Share. There's no "Share bonded
   connection", only "Phone hub (experimental) → Share as a proxy", whose
   footer says it's "not bonded" (docs/ios.md). I'm unsure, but I turn it
   on.
3. I turn on Personal Hotspot (Settings → Personal Hotspot → Allow Others
   to Join), plug the iPhone in, and tap **Trust**.
4. On the Mac I'd expect the iPhone to appear in the hub card with **Use**,
   and I'd click it. Per the docs, that gives me only the iPhone's mobile
   data, not bonding.
5. Only from the docs (docs/phone-hub.md "iPhone (experimental)",
   docs/macos.md "An iPhone over USB") do I learn the better way: skip the
   phone hub. Plug in with Personal Hotspot on, and the Mac bonds "iPhone
   (USB)" with the hotel Wi-Fi itself. The only on-screen trace of this is
   the small print on "Use this Mac's connections": "…and an iPhone plugged
   in over USB (Personal Hotspot on) are bonded".
6. To undo it, I'd guess I unplug the cable and turn off Personal Hotspot.
   Nothing says so.

**Where I hesitate**

- Which path is "the hub" for an iPhone? The Mac card invites me to use the
  proxy. The docs say not to; plain USB is better.
- The names don't match. The Mac says "tailbond → Share bonded connection";
  the iPhone app is "tailbond Share" and the switch is "Share as a proxy".
- If I do both (plugged in with hotspot on, and **Use** on the proxy), which
  one is my browser using? Did I just downgrade from bonded to mobile data
  only?
- Does the proxy stop when I lock the iPhone? (The docs say iOS pauses the
  app off-screen.)

**Findings**

- **High** (web/web-08-mac-phone-hub-none.png, docs/ios.md "Phone hub
  (experimental)"): the Mac's phone-hub card steers iPhone users into the
  experimental, unbonded proxy ("iPhone: Personal Hotspot on" right next to
  "Share bonded connection"). The recommended route is plain USB, bonded by
  the Mac, and the card never says so. I would choose the worse option and
  not know it. *Suggestion:* on the Mac card, say "iPhone? Just plug it in
  with Personal Hotspot on. tailbond bonds it with your Wi-Fi above, so no
  hub is needed." When an iPhone is already bonded over USB and also offers
  a proxy, warn before **Use** that it would bypass the bond.
- **Medium** (docs/ios.md "Phone hub (experimental)"): the iPhone footer is
  written for engineers ("SOCKS5 / HTTP CONNECT, port 1888, Bonjour
  `_tailbond-proxy._tcp`"). Only "It is not bonded … plug the iPhone into
  the Mac over USB" helps me, and it's buried. *Suggestion:* lead with the
  advice: "For the best connection, plug this iPhone into your Mac instead
  (Personal Hotspot on). This switch shares only mobile data, and only
  while tailbond Share is open." Move the ports and protocols to the docs.
- **Medium** (web/web-08-mac-phone-hub-none.png): the hint names Android's
  menu ("tailbond → Share bonded connection") even for iPhone. On the
  iPhone the app and the switch have different names. *Suggestion:* give
  separate lines, "Android: tailbond → Settings → Share my mobile data →
  Share bonded connection" and "iPhone: tailbond Share → Share as a proxy
  (mobile data only)", or name the switch the same on both.
- **Medium** (docs/phone-hub.md "iPhone (experimental)"): iOS pausing the
  app off-screen means the proxy dies when I pocket the phone. Nothing I
  can see warns me about this before I rely on it. *Suggestion:* say "Keep
  tailbond Share open on screen" next to the switch, or don't offer the
  proxy over USB at all.
- **Medium** (no render): no screenshot shows the success state for the
  recommended path ("iPhone (USB)" card under Connections). I can't confirm
  what I'd see, or how it differs from a phone hub. *Suggestion:* render
  web-08 with an iPhone (USB) card and use it in the docs.
- **Low** (docs/macos.md "An iPhone over USB"): "Set **Backup only** to save
  its data" doesn't say where. The per-link dropdown shows "Primary", and
  "Save mobile data / Backup only on failure" is a mode for everything.
  *Suggestion:* "In the iPhone (USB) card, change Primary to Backup".
- **Low** (docs/macos.md "An iPhone over USB"): no undo step. *Suggestion:*
  "Unplug, or turn off Personal Hotspot; the Mac goes back to Wi-Fi alone."

**Verdict:** I'd end up with something that works, but probably the
experimental unbonded proxy rather than the better plain-USB bonding,
because the Mac's phone-hub card points iPhone users the wrong way.
