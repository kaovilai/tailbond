# Round 2: persona A, traveller (Mac + Android + iPhone)

I use Tailscale every day, travel a lot, and carry a MacBook (Apple silicon),
an Android phone (Pixel) and an iPhone. I'm fine with apps. I don't know what
SOCKS, PAC or `en0` mean and don't want to learn. I only saw the renders and
doc sections listed in screens.md.

## Task 1: hotel, Mac onto the Pixel's bonded connection

**Steps**

1. On the Pixel, open tailbond, go to Settings, scroll to **Sharing** and tap
   the **Sharing** row ("Share this phone's bonded connection with your
   laptop, or lend its mobile data") (share-01). That was easy to find.
2. On the Sharing screen, turn on **Share bonded connection**, the top card
   (hub-01). It changes to "Ready for your laptop" with two steps written as
   one paragraph (hub-02).
3. Step 1 on the phone: Settings → Network → Hotspot & tethering, turn on
   Hotspot. (On my Pixel the menu is "Network & internet", but I'd find it.)
4. Step 2 on the Mac: in the Wi-Fi menu, join the Pixel's hotspot.
5. On the Mac, open tailbond's dashboard and go to **Connections** →
   **Use a phone's connection**. Click **Use** next to "pixel-8 · bonded
   connection · found nearby · reached over en0" (web-08-found). I'd leave
   the checkbox alone, though I'm unsure about it (see below).
6. Check that it worked. The badge at the top says "Via Pixel-8 (Proxy)".
   The card says "Using pixel-8's bonded connection (proxy) … 46.0 MB up,
   870 MB down" (web-08-using). On the phone the card turns green: "Sharing
   bonded connection with 1 device", "Device at 192.168.43.112" (hub-03).
7. Undo: click **Stop using it** on the Mac, then turn off **Share bonded
   connection** on the phone (the undo text in web-08-using says this). I
   would also have to remember to turn off the hotspot myself and rejoin the
   hotel Wi-Fi on the Mac, but nothing tells me to.

**Where I hesitate**

- At step 4. Once the Mac is on the Pixel's hotspot, I'd think I'm done:
  "I'm on my phone's connection now, aren't I?" Nothing tells me *before*
  I join that the hotspot on its own isn't the bonded connection. I only
  learn that from the small print in the "using" state.
- At step 5, the top of the dashboard says **Disconnected** with a switch
  that's off (web-08-found). I'd almost certainly turn that on first, which
  bonds the Mac's own links instead. Only after I click Use does the page
  say "This machine's own tunnel is off. That's fine" (web-08-using).
- The checkbox "Keep home, local and Tailscale addresses on this Mac's own
  network". As a Tailscale user I'm scared of breaking Tailscale. Should I
  tick it? There's no recommendation, and "Off: everything a browser opens
  goes through the phone" doesn't say whether my 100.x machines still work.
- At step 6, the big graph and the DATA counter at the top show **0 bps / 0 B**
  while the card below says 870 MB down. For a moment I'd think it isn't
  working.
- On the phone, "Device at 192.168.43.112" doesn't tell me it's *my Mac*.
- At step 7, does "Stop using it" leave me on the phone's hotspot, using
  phone data that isn't bonded? Probably yes, and I wouldn't notice.

**Findings**

- **High** (web-08-mac-phone-hub-using.png, hub-02-ready.png): the undo is
  incomplete. "To undo: Stop using it here, then turn off 'Share bonded
  connection' on the phone" never says to turn off the phone's hotspot or
  rejoin the hotel Wi-Fi. After following it exactly, my Mac is still on the
  Pixel's hotspot, using its mobile data, not bonded, with no proxy, and the
  dashboard looks "normal". On a capped travel plan, that's the outcome I care
  most about avoiding. *Suggestion:* make the undo a three-step list: Stop
  using it → turn off Share bonded connection → turn off Hotspot / rejoin
  your Wi-Fi. Better still, after Stop show "This Mac is still on pixel-8's
  hotspot (not bonded)" when the default route is the phone.
- **Medium** (web-08-mac-phone-hub-found.png): when a hub is found, the top
  switch still shows "Disconnected" and looks like the thing to fix. The
  reassurance ("own tunnel can stay off") only appears after Use. *Suggestion:*
  put "You don't need to turn on the switch at the top for this" in the
  found row, or change the header label to something neutral ("Own tunnel
  off") before Use, not only after.
- **Medium** (hub-02-ready.png): steps 1 and 2 run together in one dense
  paragraph, and the most important idea, that the hotspot alone isn't
  bonded and you must press Use on the Mac, is buried. "Most phones stay on
  the hotel Wi-Fi meanwhile" is hedged. What if mine doesn't? *Suggestion:*
  show the steps as separate numbered lines, with a bold step 3 "On the Mac:
  Use. Joining the hotspot alone isn't bonded." Tell me what to look for if
  the phone dropped the hotel Wi-Fi.
- **Medium** (web-08-mac-phone-hub-found.png): the checkbox wording worries a
  Tailscale user and has no default advice. Docs (phone-hub.md) call it
  "Keep local and tailnet addresses direct", but the UI says something
  different. *Suggestion:* recommend one setting ("Leave on if you use
  Tailscale or home devices"), say plainly what breaks either way, and use
  the same label in UI and docs.
- **Medium** (web-08-mac-phone-hub-using.png): the hero graph and DATA show
  0 while the proxy has moved 900 MB. That contradicts the card under it.
  *Suggestion:* when using a hub, feed the hero numbers from the proxy
  counters, or grey them out with "Your own tunnel is off; see 'Use a
  phone's connection' below".
- **Medium** (hub-03-sharing.png): the phone shows "Device at
  192.168.43.112: 960.2 MB", an IP address, not a name. I can't tell
  whether it's my Mac or someone else on the hotspot. *Suggestion:* show the
  paired device name ("MacBook (tiger-mbp)"), fall back to the IP, and add a
  per-device "Stop" option.
- **Low** (web-08-mac-phone-hub-using.png): the card "Shared by nearby
  phones: No phone is sharing" sits right under "Using pixel-8" and
  contradicts it at a glance. Two different "phone" cards with different
  instructions is confusing. *Suggestion:* when a hub is in use, collapse
  the nearby-phones card, or explain the difference in one line ("lend
  mobile data" vs "use a phone's bonded connection").
- **Low** (web-08-mac-phone-hub-found.png): "reached over en0" is jargon.
  *Suggestion:* say "over Wi-Fi" or "over the phone's hotspot".
- **Low** (hub-02-ready.png): "Settings → Network → Hotspot & tethering"
  doesn't match the Pixel's "Network & internet". *Suggestion:* use the
  real Pixel label or add an "Open hotspot settings" button.
- **Low** (web-08-mac-phone-hub-lost.png): "Lost the phone hub, so the proxy
  is off again" is clear and reassuring. But "phone hub" and "proxy" appear
  nowhere else in the words I clicked, and it doesn't say what to try
  (rejoin the hotspot? wake the phone?). *Suggestion:* add "Rejoin pixel-8's
  hotspot and press Look again".
- **Low** (all web-08): the header says "Bonded to homebox.tail1234.ts.net"
  even while "Disconnected". *Suggestion:* say "Home server:
  homebox…" instead.
- **Low** (hub-05-no-tunnel.png): it's good that "Turn tailbond on" is right
  there. But the Mac's view of this state isn't shown, so I don't know
  whether the Mac's browsing just stops ("Nothing from the laptop goes out
  until the tunnel is up"). *Suggestion:* make the Mac card say "pixel-8 is
  waiting for its own tailbond to connect".

**Verdict:** I'd get the Mac onto the Pixel's bonded connection after one
wrong turn (the top switch). I'd probably *undo* it only halfway and keep
burning phone data on an unbonded hotspot.

## Task 2: iPhone as the hub, over USB

**Steps**

1. My first instinct is to open **tailbond Share** on the iPhone, since the
   Mac's "Shared by nearby phones" card literally says "On iPhone: tailbond
   Share → Share mobile data". There I'd see "Share mobile data with a Mac"
   under For a Mac (experimental). The footer ("Better: plug the iPhone into
   the Mac with a cable and turn on Personal Hotspot … you don't need this")
   would turn me around, but only if I read it.
2. iPhone: Settings → Personal Hotspot → **Allow Others to Join** on
   (macos.md "An iPhone over USB").
3. Plug the iPhone into the Mac and tap **Trust**.
4. On the Mac, open the dashboard → Connections. I'd expect an **iPhone (USB)**
   card. The doc promises one, and the "Use a phone's connection" card says
   "this Mac bonds it itself, as iPhone (USB) with the connections above".
   What I actually see is a card called **en7 · USB Tether · Secondary**
   (web-08-iphone-usb).
5. The top switch says Connected in that render. The USB doc section never
   tells me to turn it on, so I'd assume it happens automatically, or flip
   it if it's off.
6. Optionally, set the card's dropdown to **Backup only** to save iPhone
   data (the doc's advice). The dropdown in the render reads "Secondary".
7. Undo: unplug or turn off Personal Hotspot. That part is clear and simple.

**Where I hesitate**

- At step 1, two routes (app switch vs USB) are offered in different
  places, and the Mac's nearby-phones card points me at the weaker one.
- At step 4, nothing on screen says "iPhone". "en7 / USB Tether" could be
  anything. I'd replug the cable and wait, thinking it hadn't been detected.
- The "Use a phone's connection" card still shows the generic "iPhone: no
  hub needed. Plug it in…" text and a **Look again** button while the
  iPhone is already bonded. That suggests it still isn't found.
- At step 6, the doc says "Backup only", the dropdown shows
  "Primary/Secondary", and the mode tile says "Save mobile data: Backup only
  on failure". Which one do I change?
- If I did flip the iPhone app switch (web-08-iphone-found): "dana-iphone ·
  mobile data only, not bonded" with a big green **Use** button. Nothing on
  that screen suggests USB instead, though phone-hub.md says the Mac "says
  how to bond an iPhone over USB instead" there.

**Findings**

- **High** (web-08-mac-phone-hub-iphone-usb.png, macos.md "An iPhone over
  USB"): the docs and the dashboard's own text promise a card named
  **iPhone (USB)**, but the render shows **en7 · USB Tether**. That name is
  the only confirmation I'm told to look for, so I'd conclude it failed.
  *Suggestion:* label the card "iPhone (USB)" (keep en7 as small secondary
  text if needed), or update the docs and the hint text to match.
- **Medium** (web-08-mac-phone-hub-iphone-usb.png): "Use a phone's
  connection" keeps its "nothing found" instructions and **Look again** while
  the iPhone is bonded. *Suggestion:* when an iPhone USB link is active,
  replace that text with "Your iPhone is bonded over USB (above). No hub
  needed."
- **Medium** (all web-08, "Shared by nearby phones"): "On iPhone: tailbond
  Share → Share mobile data" steers iPhone owners to the not-bonded,
  stops-when-locked route that the iPhone app itself calls second best. The
  label also doesn't match the iOS switch ("Share mobile data with a Mac").
  *Suggestion:* lead with "iPhone: plug it in over USB with Personal Hotspot
  on (best)" and use the exact iOS switch name for the fallback.
- **Medium** (web-08-mac-phone-hub-iphone-found.png): an iPhone hub is
  offered with the same **Use** button as the bonded Pixel. The "not bonded"
  warning is grey small print, and the USB advice that phone-hub.md says is
  shown there isn't on screen. *Suggestion:* add an inline line under the row,
  "Better: plug this iPhone in over USB with Personal Hotspot on; the Mac
  bonds it", and visually de-emphasise Use for unbonded hubs.
- **Medium** (macos.md "An iPhone over USB" vs web-08-iphone-usb): the doc says
  to set the dropdown to **Backup only**, but the card's dropdown shows
  Primary/Secondary, and "Backup only" is also the name of a mode tile. I
  can't tell whether that's a per-link or whole-Mac setting. *Suggestion:*
  quote the exact dropdown option in the doc and keep per-link and mode
  names distinct.
- **Medium** (macos.md "An iPhone over USB"): the steps never mention the
  dashboard's top switch, which must be Connected for bonding (it's
  Connected only in this render). In Task 1 I was told the switch can stay
  off, so I'd now wrongly assume the same. *Suggestion:* add step 0 "Turn
  tailbond on (top switch)", or auto-prompt when an iPhone is plugged in and
  the tunnel is off.
- **Low** (ios.md "For a Mac (experimental)", phone-hub.md "iPhone
  (experimental)"): user-facing text mentions "SOCKS5 / HTTP CONNECT, port
  1888, Bonjour", and the phone-hub section mixes setup with developer
  internals (DirectDialer, lwIP). *Suggestion:* split the user steps from the
  "how it works / not built yet" part.
- **Low** (iOS footer, from screens.md): the footer is clear and honest ("only
  lends mobile data, not bonded … stops when you leave this screen or lock
  the iPhone"). Keep it. Consider showing it *above* the switch, not under
  it.

**Verdict:** the USB route itself is simple and the undo is trivial. But the
card says "en7 · USB Tether" where everything promised "iPhone (USB)", and
the hub card still says "not found". So I'd doubt it worked, and might fall
back to the weaker in-app switch the Mac itself suggested.
