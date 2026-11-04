# Round 2: persona F2, home admin

I set up homebox for the family. I know what a SOCKS proxy, a hotspot and
CGNAT are. Before I turn anything on I want to know who else can reach it,
what it costs in battery and data, and how to put everything back.

## Task 1: hotel, Mac on the Pixel's bonded connection

**Steps**

1. Pixel: tailbond → Settings, scroll past Always-on VPN to **Sharing** →
   the "Sharing" row ("Share this phone's bonded connection with your
   laptop, or lend its mobile data") (share-01).
2. Sharing screen: flip **Share bonded connection** at the top (hub-01).
   It changes to "Ready for your laptop" with the two steps (hub-02).
3. On the Pixel: Android Settings → Network → Hotspot & tethering → Wi-Fi
   hotspot on. I check that the phone stays on the hotel Wi-Fi.
4. Mac: join the Pixel's hotspot, then open the tailbond dashboard
   (localhost:8077). It opens on Connections. Under **Use a phone's
   connection** I see "pixel-8 · bonded connection · found nearby · reached
   over en0" (web-08-found).
5. I tick "Keep home, local and Tailscale addresses on this Mac's own
   network", because I want to reach homebox and the NAS directly, then
   click **Use**.
6. Check: the badge at the top says "Via Pixel-8 (Proxy)" and the card says
   "Using pixel-8's bonded connection (proxy) … 46.0 MB up, 870 MB down"
   (web-08-using). The phone says "Sharing bonded connection with 1 device:
   960.2 MB, Device at 192.168.43.112" (hub-03).
7. Undo: on the Mac, **Stop using it**. On the phone, turn **Share bonded
   connection** off. Then (my own step, not on screen) turn off the hotspot
   and rejoin the hotel Wi-Fi on the Mac.

**Where I hesitate**

- Step 3: "Most phones stay on the hotel Wi-Fi meanwhile". Which phones
  don't? If my Pixel drops the hotel Wi-Fi when the hotspot comes on, the
  "bond" is just 5G. Nothing on the Sharing screen shows which links the
  bond is using right now.
- Step 4: the header says **Disconnected**, with an off switch, next to
  "Bonded to homebox". Before I click Use, I'm tempted to turn that switch
  on. Only after Use does the page explain "This machine's own tunnel is
  off. That's fine." I'm also unsure whether turning it on would bond the
  hotspot link a second time.
- Step 5: the checkbox says "this Mac's own network". At that point the
  Mac's own network *is* the phone's hotspot, so "direct" means my phone's
  mobile data, unbonded and outside the tunnel. If tailscaled isn't
  running on the Mac, it's also unclear how `.ts.net` names would resolve
  "directly" at all.
- Step 6: the big numbers at the top read 0 bps, DATA 0 B, flat graph,
  while the card below says 870 MB went down. At first glance it looks
  broken.
- Security: the phone lists the client as "Device at 192.168.43.112". I
  can't tell whether that's my MacBook or my kid's laptop. There's no name
  and no way to kick it off.
- Is the phone's proxy only on the hotspot interface, or also on the hotel
  Wi-Fi where other guests are? The screen says "only devices paired with
  homebox … get its password from homebox". That's good, but it doesn't say
  where it listens, and it gives no way to change or revoke that password.
- Battery: the phone screen states the battery and monthly limits "are for
  lending mobile data, not for this". So the hotspot, the bond and the
  proxy run on battery with no cut-off. That's the worst battery case,
  and it's the one without a guard.
- Undo: the Mac's undo text ("Stop using it here, then turn off 'Share
  bonded connection' on the phone") stops there. The Mac stays on the
  phone's hotspot, so video calls and everything else keep spending my
  mobile data unbonded.

**Findings**

- **High** (web-08-mac-phone-hub-using.png): the undo leaves the Mac joined
  to the phone's hotspot and the hotspot on, so non-proxied traffic keeps
  using mobile data outside the tunnel. *Suggestion:* make it a 3-step
  undo: "Stop using it → turn off the hotspot on the phone → rejoin your
  Wi-Fi on this Mac". Show the same text after Stop, and on the phone when
  Share bonded connection goes off with the hotspot still on.
- **Medium** (hub-02-ready.png / hub-03-sharing.png): no battery cut-off or
  auto-off for the bonded share. The text says the limits don't apply to
  it. *Suggestion:* give it its own "stop when battery below X% unless
  charging" control. At least show the battery level and a "plug in the
  phone" hint while sharing.
- **Medium** (hub-03-sharing.png): the client appears only as an IP, with
  no device name and no way to disconnect it. "Who can use your mobile
  data" covers only lending. *Suggestion:* show the homebox device name if
  the proxy auth can carry it, add a per-device "Disconnect", and let the
  allow list cover the bonded share too.
- **Medium** (hub-02-ready.png): it doesn't say which interfaces the proxy
  listens on or how to rotate the derived password. *Suggestion:* one line:
  "Listens on the hotspot/USB only; password comes from homebox's share
  key. To revoke, re-key on homebox." Better still, really bind only to
  tether interfaces.
- **Medium** (web-08-mac-phone-hub-found.png): the "Disconnected" header
  switch above a usable hub invites the wrong click. *Suggestion:* in the
  found state, show the same "own tunnel can stay off" note as the using
  state, next to the Use button.
- **Medium** (web-08-mac-phone-hub-found.png): the PAC checkbox's "this
  Mac's own network" means the phone's hotspot, unbonded. It doesn't say
  where homebox/`.ts.net` traffic goes if Tailscale isn't up on the Mac.
  *Suggestion:* "Keep home, local and Tailscale addresses off the proxy
  (they go over the hotspot directly, or through Tailscale if it's on)".
- **Medium** (hub-02-ready.png): "or USB tethering with a cable" to a Mac.
  Android USB tethering to macOS has historically needed extra drivers,
  and nothing here says it works. *Suggestion:* say whether a Mac supports
  it without extras, or recommend the Wi-Fi hotspot for Macs.
- **Medium** (hub-03-sharing.png): 960.2 MB total doesn't tell me how much
  was mobile data and how much was hotel Wi-Fi, and that split is what my
  plan bills. *Suggestion:* split the total into "via 5G / via Wi-Fi /
  via sam-pixel".
- **Low** (web-08-mac-phone-hub-using.png): the hero stats (0 bps, 0 B)
  contradict the card's 46 MB / 870 MB. *Suggestion:* drive the hero
  graph from the proxy counters while a hub is in use, or replace it with
  "Traffic goes through pixel-8".
- **Low** (web-08-mac-phone-hub-using.png): "Via Pixel-8 (Proxy)" vs
  "pixel-8" everywhere else. *Suggestion:* use the device name verbatim.
- **Low** (hub-02-ready.png): "1. … 2. …" is run together as one
  paragraph, with a long "Settings → Network → Hotspot & tethering" path
  and no button. *Suggestion:* a real numbered list, plus an "Open hotspot
  settings" button.
- **Low** (docs/phone-hub.md "Mac: Use phone hub"): the doc names the
  checkbox *"Keep local and tailnet addresses direct"*, but the UI says
  "Keep home, local and Tailscale addresses on this Mac's own network".
  *Suggestion:* match the UI string.
- **Low** (docs/phone-hub.md): the doc explains crash-safe proxy restore
  well (restore record, LaunchDaemon), but the UI never tells me my
  previous proxy settings come back. *Suggestion:* add "Your previous proxy
  setting comes back" under Stop using it.

**Verdict:** I'd get it working in a few minutes and trust the fail-closed
behaviour ("Nothing from the laptop goes out until the tunnel is up"), but
the undo is incomplete and the battery and who-is-connected controls are
missing for exactly this mode.

## Task 2: iPhone as the hub over USB

**Steps**

1. Mac dashboard → Connections. The "Use a phone's connection" card says
   "iPhone: no hub needed. Plug it in over USB with Personal Hotspot on:
   this Mac bonds it itself, as iPhone (USB) with the connections above"
   (web-08-none).
2. iPhone: Settings → Personal Hotspot → Allow Others to Join on (from
   docs/macos.md "An iPhone over USB").
3. Plug in the cable and tap Trust.
4. Look for an "iPhone (USB)" card next to en0. What appears is **"en7 ·
   USB Tether · Secondary"** (web-08-iphone-usb).
5. Notice that the header is now "Connected" with a tunnel IP. In Task 1
   the tunnel was off, so I flip the top switch on.
6. Optional: set the en7 card's dropdown to "Backup only" to save data, as
   the doc says.
7. Undo: unplug, or turn Personal Hotspot off.
8. I skip tailbond Share's "Share mobile data with a Mac": its own footer
   says it isn't bonded and stops when the phone locks.

**Where I hesitate**

- Step 4: everything I was told says "iPhone (USB)", but the card says
  "en7 / USB Tether". I can work out that en7 is the iPhone. A family
  member couldn't, and with two cables plugged in (USB-C Ethernet plus the
  iPhone) I couldn't either.
- Step 5: nothing in the dashboard hint or in macos.md's 4 steps says the
  Mac's own tunnel has to be **on** for the iPhone to be bonded. Task 1
  just taught me "the own tunnel can stay off". If I leave it off, I'd
  expect the iPhone to sit there unbonded.
- Step 6: the doc says to set the dropdown to **Backup only**, but the
  dropdown shows Primary/Secondary. There's also a separate "Save mobile
  data · Backup only on failure" mode at the top. I don't know whether
  these are the same thing, or whether the mode applies to every link.
- Security: "Allow Others to Join" turns on a Wi-Fi hotspot that anyone
  nearby could try, even though I only want USB.
- On the iPhone: tailbond Share has both "Share mobile data" (lend to
  nearby phones) and "Share mobile data with a Mac" (proxy). The Mac's
  "Shared by nearby phones" hint says "tailbond Share → Share mobile data".
  Two near-identical names for different things.
- web-08-iphone-found lists **dana-iphone**, my partner's phone, with a
  **Use** button on my Mac. Any family device paired with homebox can ride
  any other's mobile data, and there's no allow list for this proxy.

**Findings**

- **Medium** (web-08-mac-phone-hub-iphone-usb.png): the card is labelled
  "en7 · USB Tether", while the hint, macos.md and phone-hub.md all
  promise "iPhone (USB)". *Suggestion:* label the card "iPhone (USB)",
  with en7 in small print, as the docs say.
- **Medium** (web-08-mac-phone-hub-none.png / docs/macos.md "An iPhone over
  USB"): it never says the Mac's own tunnel must be on, and in Task 1 the
  UI said off was fine. *Suggestion:* add "turn on tailbond at the top" as
  a step. Better: when an iPhone USB link appears while the tunnel is off,
  show "iPhone plugged in. Turn on tailbond to bond it".
- **Medium** (web-08-mac-phone-hub-iphone-found.png): a family member's
  iPhone (dana-iphone) is usable by my Mac with no consent step or allow
  list on the sharing side. *Suggestion:* by default, offer the hub only
  to the owner's own devices, or show "used by <device>" on the sharer
  with a Stop.
- **Medium** (web-08-mac-phone-hub-iphone-found.png): docs/phone-hub.md
  says that for an unbonded iPhone the Mac "says how to bond an iPhone
  over USB instead". The render shows only "mobile data only, not bonded",
  and the USB hint is gone in this state. *Suggestion:* put "Better: plug
  it in over USB with Personal Hotspot on" under that row.
- **Low** (docs/macos.md "An iPhone over USB"): "set the dropdown to
  Backup only" doesn't match the Primary/Secondary dropdown shown.
  *Suggestion:* use the real option names, and explain how this relates to
  the "Save mobile data" mode.
- **Low** (docs/macos.md "An iPhone over USB"): Allow Others to Join also
  opens Wi-Fi. *Suggestion:* note that with the iPhone's Wi-Fi turned off,
  the hotspot is USB/Bluetooth only, which is what a cautious user wants
  here.
- **Low** (docs/ios.md "For a Mac (experimental)"): "Share mobile data"
  and "Share mobile data with a Mac" are confusable. The footer is honest
  and helpful ("Better: plug the iPhone into the Mac with a cable…").
  *Suggestion:* rename the Mac one "Lend mobile data to a Mac (proxy, not
  bonded)".

**Verdict:** I'd get it working, because USB plus Personal Hotspot is the
simplest path and the dashboard points to it. But the "iPhone (USB)" label
promised everywhere never appears, and nothing says the Mac's tunnel must
be on.
