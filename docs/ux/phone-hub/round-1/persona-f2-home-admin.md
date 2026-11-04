# Round 1: persona F2, home admin

Who I am: I set up homebox for the family. I know what a SOCKS proxy, a
hotspot and a kill switch are. Before I route my Mac through a phone I want
to know who else can use that proxy, what leaks around it, and how much
battery it costs. I have a MacBook, a Pixel 9 and an iPhone.

## Task 1: at a hotel, put the Mac on the Pixel's bonded connection

**Steps**

1. On the Pixel I open tailbond and go to Settings (`share-01-settings-row.png`). I'm looking for "Phone hub", "Hotspot" or "Share bonded connection". None of those words is on the screen. Scrolling, I find "Lend your mobile data → Share my mobile data · Not sharing". I tap it, but I'm not sure it's the right place: I want to lend my *bonded* connection, not my mobile data.
2. The screen is called "Share my mobile data" (`hub-01-off.png`). The first big switch is "Share my mobile data", and I nearly flip it. The second card is "Share bonded connection", described as "A Mac paired with your home server can use this phone's bonded connection… through a proxy". That's the one, so I flip it.
3. The status changes to "Ready for your laptop: On the Mac: join this phone's hotspot or plug it in over USB, then tailbond → Use a phone's bonded connection" (`hub-02-ready.png`). The app has no button for the hotspot, so I leave the app and turn it on in Android Settings → Network → Hotspot.
4. I check the Pixel is still on hotel Wi-Fi, because if the hotspot drops it, the "bonded" connection is just 5G. The Share screen doesn't show the phone's own paths, so I'd have to go back to the main tab to check.
5. On the Mac I join the Pixel's hotspot and open the dashboard at localhost:8077 → Connections. The "Use a phone's bonded connection" card shows "Pixel 9 · found on this network" with a **Use** button (`web-08-mac-phone-hub-found.png`).
6. I don't know whether to tick "Keep local and tailnet addresses direct (proxy auto-config)". As an admin I'd guess yes, so homebox's LAN and tailnet don't hairpin through the phone. Nothing on screen explains it. I tick it and click **Use**.
7. The card reads "Using Pixel 9's bonded connection (proxy). The Mac's proxy setting (Wi-Fi) points at it. 46.0 MB up, 870 MB down" (`web-08-mac-phone-hub-using.png`). But the header above still says **Disconnected**, 0 bps, and "No connections yet: Waiting for mqvpn to bring up its paths". I'm tempted to flip the top "Disconnected" switch on, and I don't know whether that helps or breaks things.
8. On the Pixel the bonded card turns teal: "Sharing bonded connection with 1 device: 960.2 MB", with the device shown as "192.168.43.112" (`hub-03-sharing.png`).
9. To undo: **Stop using it** on the Mac, which per docs/macos.md restores my earlier proxy setting exactly. Then I turn off "Share bonded connection" and the hotspot on the Pixel, and rejoin hotel Wi-Fi on the Mac. If the phone leaves, the Mac drops the hub by itself and says so (`web-08-mac-phone-hub-lost.png`).

**Where I hesitate**

- Settings: no row names the phone hub. I'd only find it by guessing that "Share my mobile data" contains it.
- Choosing between the two switches on the Share screen.
- Whether turning on the hotspot drops the Pixel's hotel Wi-Fi.
- The "proxy auto-config" checkbox.
- The Mac header saying "Disconnected" while the card says "Using".
- Whether "192.168.43.112" is my Mac or a stranger on my hotspot.

**Findings**

- **High** (web-08-mac-phone-hub-using.png): while the hub is in use, the dashboard header still says "Disconnected · 0 bps · Uptime —" and the link area says "No connections yet: Waiting for mqvpn to bring up its paths". The page tells me two opposite things. I'd likely flip the top switch on, and nothing says whether running Mac tailbond on top of the phone hub is supported, harmful or pointless. *Suggestion:* while a hub is in use, change the header to "Via Pixel 9 (phone hub)" with the hub's traffic, and replace "No connections yet" with "This Mac's links aren't needed while you use Pixel 9's bonded connection". If the top switch must stay off, say so.
- **Medium** (share-01-settings-row.png, web-08-mac-phone-hub-none.png): the Mac's hint says "On the phone: tailbond → Share bonded connection". The real path is tailbond → Settings → Lend your mobile data → Share my mobile data → Share bonded connection. The Settings row's subtitle is only about mobile data ("Not sharing"). *Suggestion:* rename the row (e.g. "Share with other devices") and make its subtitle cover both switches ("Mobile data off · Bonded connection on"), or add a separate "Share bonded connection (phone hub)" row. Make the Mac's hint give the exact path.
- **Medium** (hub-03-sharing.png): the device using my proxy is shown only as "192.168.43.112". As an admin I can't tell whether that's my MacBook or someone who got onto my hotspot. The screen also has no way to kick a device off. *Suggestion:* show the paired device name (the Mac's homebox name), mark unknown clients clearly, and add a per-device "Stop sharing with this device" action.
- **Medium** (hub-01..05, Who can use it): the security model isn't in the phone UI. "Who can use it: Any device paired with homebox" sits at the bottom of a screen with three features, so I can't tell which feature it controls. docs/phone-hub.md says the proxy requires a password derived from the share key. That's what I want to know, and only the docs say it. *Suggestion:* under "Share bonded connection" add one line: "Password-protected: only devices paired with homebox can use it." State which features "Who can use it" covers.
- **Medium** (hub-03-sharing.png): "Who's using it: Turn sharing on to see who uses it" appears directly under a card saying I'm sharing with 1 device. "Monthly limit: 0 B shared this month" also appears while 960.2 MB has gone through the bonded share. Either the limit and the battery cut-off don't apply to the bonded share, or the counters are wrong. The UI doesn't say which. *Suggestion:* tie "Who's using it", "Stop when the battery is low" and "Monthly limit" to a named feature. Either show bonded-share traffic in them or add a note like "Applies to Share my mobile data only". The battery cut-off matters most here: hotspot plus bonding plus proxy is the heaviest drain this phone will see.
- **Medium** (hub-02-ready.png): "join this phone's hotspot or plug it in over USB" leaves out the Android steps: turning on the hotspot, or turning on USB tethering, which the Mac may not support with every phone. It also doesn't warn whether the hotspot keeps the hotel Wi-Fi up, and that's the whole point of bonding. *Suggestion:* add an "Open hotspot settings" button. Show a live line such as "Bonding now: 5G + hotel Wi-Fi" so I can see the hotspot didn't drop Wi-Fi, and warn if it did.
- **Medium** (web-08-mac-phone-hub-found.png): "found on this network" doesn't say *which* network. If the Mac and Pixel are both on hotel Wi-Fi, can the Mac find and use the Pixel over the hotel LAN, sending my proxy traffic across a network I don't trust? *Suggestion:* show the link, e.g. "via Pixel 9's hotspot (en0)" or "via USB", and warn or refuse if it's a shared Wi-Fi.
- **Medium** (web-08-mac-phone-hub-using.png): "Others, and anything that isn't TCP… still use this Mac's own connection." When I'm on the Pixel's hotspot, "this Mac's own connection" *is* the Pixel's raw hotspot uplink, unbonded and not through homebox. The text doesn't say where that traffic actually goes, or whether DNS is resolved through the proxy or leaks. *Suggestion:* add a "What bypasses this" line: "UDP, video calls and apps that ignore proxies go out through the phone's hotspot directly, not through homebox. DNS: …".
- **Medium** (web-08-mac-phone-hub-found.png): "Keep local and tailnet addresses direct (proxy auto-config)" is unexplained and off by default. I can't tell what happens if I leave it off: does homebox/tailnet traffic go through the phone? *Suggestion:* one plain sentence under it, e.g. "Home network, .local and Tailscale (100.x/.ts.net) addresses skip the phone". Consider making it on by default.
- **Low** (web-08-mac-phone-hub-lost.png): the Mac fails open: when the phone goes away it quietly goes back to hotel Wi-Fi. The banner is clear, and that's right for keeping the Mac online. But there's no fail-closed option, and the setting isn't remembered after sleep or reboot, so I could be off the hub without noticing. *Suggestion:* add a macOS notification on "Lost the phone hub", and an optional "Block instead of falling back" for cautious users.
- **Low** (share-01-settings-row.png vs web-08-mac-phone-hub-found.png): the phone calls itself "pixel-8" in Settings, but the Mac lists "Pixel 9". With mismatched names I can't confirm it's the same device. *Suggestion:* use the same name, the homebox device name, in both places.
- **Low** (hub-05-no-tunnel.png): "Turn tailbond on. Nothing from the laptop goes out until the tunnel is up." That's reassuring and fail-closed, which I like, but there's no button to turn it on. *Suggestion:* add a "Turn tailbond on" button.
- **Low** (hub-04-sharing-dark.png): in dark mode the "Share my mobile data" card is drawn as a raised, tinted card even though it's off. In light mode it's flat. That makes it look active. *Suggestion:* use the same off style in both themes.
- **Low** (hub-01..05): "Use nearby phones' shared data" pulls data *into* this phone, yet it sits between two "share out" features. It makes the screen harder to scan. *Suggestion:* move it to its own section or screen ("Use other phones").
- **Low** (web-08-*): the sticky header overlaps the mode cards and half-covers the tab labels. "Connections" and "Setup" are cut off in `found`, and the mode cards are hidden in `using`. Jargon also shows up in the UI: "mqvpn", "mqvpn control API unavailable", "proxy auto-config". *Suggestion:* fix the header overlap and use plain wording.

**Verdict:** I'd get there in about 10 minutes and the undo is clean, but the Mac page contradicts itself ("Disconnected" vs "Using") and the phone won't tell me who is on my proxy, so I wouldn't fully trust it.

## Task 2: use the iPhone as the hub over USB

**Steps**

1. On the Mac dashboard, the phone-hub card (`web-08-mac-phone-hub-none.png`) says "No phone hub found. On the phone: tailbond → Share bonded connection. Then join its hotspot, or plug it in over USB (iPhone: Personal Hotspot on)." I read that as: the iPhone can be a phone hub over USB.
2. So on the iPhone I open tailbond Share and look for "Share bonded connection". It doesn't exist. Per docs/ios.md there's "Phone hub (experimental) → Share as a proxy", and its footer says it's **not bonded**, mobile data only. That's when I realize the iPhone "hub" isn't what the Mac card promised.
3. I turn on Settings → Personal Hotspot → Allow Others to Join, plug in the cable and tap Trust (docs/macos.md "An iPhone over USB").
4. Now there are two possible paths:
   - (a) Leave the iPhone app out entirely. "Use this Mac's connections" says "an iPhone plugged in over USB (Personal Hotspot on) are bonded", so the Mac bonds "iPhone (USB)" with hotel Wi-Fi itself.
   - (b) Turn on "Share as a proxy" on the iPhone and click **Use** on the Mac.

   After reading docs/phone-hub.md ("Recommended for an iPhone: USB… needs no app") I'd pick (a). Without the docs, I'd follow the Mac card and pick (b).
5. With (a), I'd expect an "iPhone (USB)" link card next to en0 under Connections. There's no render of that, so I can't confirm what I'd see. I'd set it to "Backup only" with the per-link dropdown if I want to save the iPhone's data.
6. To undo (a): unplug the cable or turn off Personal Hotspot. To undo (b): **Stop using it** on the Mac, then turn off "Share as a proxy" on the iPhone.

**Where I hesitate**

- The Mac's phone-hub card tells iPhone users to use the phone hub, but the iPhone has no bonded share, and the docs recommend *not* using the hub for an iPhone.
- The Mac screen offers three iPhone paths, and I have to work out which one I want:
  - "Use this Mac's connections" (USB link)
  - "Shared by nearby phones" (tailbond Share → Share mobile data)
  - "Use a phone's bonded connection" (Share as a proxy)
- Whether the iPhone proxy keeps working once the screen locks.

**Findings**

- **High** (web-08-mac-phone-hub-none.png, web-08-mac-phone-hub-lost.png): the hint in "Use a phone's bonded connection" ends with "or plug it in over USB (iPhone: Personal Hotspot on)". That sends iPhone owners to the phone hub. But on an iPhone the hub is experimental and not bonded, and docs/phone-hub.md says USB bonding on the Mac is better. Following the UI gives me a worse result than doing nothing. *Suggestion:* change the hint to "iPhone? Just plug it in with Personal Hotspot on. This Mac bonds it with the Wi-Fi itself (see the card above)." Keep the phone hub for Android.
- **High** (docs/ios.md "Phone hub (experimental)", web-08-mac-phone-hub-using.png): this card's title is "Use a phone's bonded connection", and the only in-use text I've seen is "Using <phone>'s bonded connection (proxy)". Nothing I've seen shows the Mac labeling an iPhone hub "not bonded". I'd believe I was bonding while running on cellular alone. *Suggestion:* when the hub is an iPhone (or anything advertising unbonded), say "Using iPhone's mobile data (proxy, not bonded)" and suggest "Plug in over USB to bond instead".
- **Medium** (docs/phone-hub.md "iPhone (experimental)", docs/ios.md): iOS pauses tailbond Share off-screen, so the iPhone proxy dies when the screen locks. After about 45 s the Mac then shows "Lost the phone hub". Keeping the screen on for hours costs battery, and the proxy switch footer doesn't mention it. *Suggestion:* put "Keep this screen open; iOS stops the proxy when it's in the background" in the "Share as a proxy" footer itself. Or hide the switch from non-developers until the bonded tunnel exists.
- **Medium** (web-08-mac-phone-hub-none.png): one Connections page has three iPhone-related instructions in three cards: USB link, "On iPhone: tailbond Share → Share mobile data", and the phone-hub hint. Nothing says which to choose. *Suggestion:* add one short "With an iPhone, plug it in (best). Sharing over Wi-Fi works too." line, or a "Which should I use?" link to the "When to use which" section of docs/phone-hub.md.
- **Medium** (screens.md): there is no render of the Mac with "iPhone (USB)" connected, or of the iPhone's "Share as a proxy" screen. I can't check that the recommended path looks right: link name, data counter, "Backup only" option. *Suggestion:* add a web render with an iPhone (USB) link card and a still of the iOS Phone hub section.
- **Low** (docs/macos.md "An iPhone over USB"): "Set **Backup only** to save its data". The mode cards include "Save mobile data: Backup only on failure", which is global, and each link card has a "Primary" dropdown. I'm not sure which one is meant. *Suggestion:* say "in the iPhone (USB) card, set the dropdown to Backup only".
- **Low** (docs/macos.md): USB tethering needs "Allow Others to Join", which also opens the Wi-Fi hotspot to anyone with its password. As an admin I'd note it, but it's Apple's design. *Suggestion:* one line: "Your hotspot's Wi-Fi password still protects it; change it if you've shared it."

**Verdict:** the right answer (plug in and let the Mac bond it) is there and simple, but the phone-hub card points iPhone users to the experimental, unbonded proxy. Without the docs I'd have picked the worse option.
