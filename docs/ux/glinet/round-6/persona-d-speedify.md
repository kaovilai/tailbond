# Round 6, persona D: long-time Speedify-on-GL.iNet user

Background: I've run Speedify on GL routers for years. Speedify sits in GL's left
menu under VPN, uses the GL admin login, has Speed / Redundant / Streaming modes,
per-connection Always / Secondary / Backup / Never, monthly caps, live
per-connection graphs, "Using 2 connections". On a hotel captive portal it keeps
the VPN up on my other links, notifies me, and the notification opens the
hotel's sign-in page; after that the Wi-Fi joins the bond.

Inputs: task-script.md, screens.md, round-6 PNGs (all desktop variants, phone
variants of 05 checked).

First impression: this feels like Speedify on GL. **VPN → tailbond** sits in the
same place under VPN as Speedify (02), behind the same GL login. The Slate 7 page
has the same shape: big on/off switch, "Connected to homebox", "Using Repeater
(Wi-Fi) + Tethering (phone)", per-link cards with data used and a cap, and a mode
picker. The main differences are install (via LuCI) and captive portals (two
different flows).

---

## Task 1: Install on the Slate 7 without SSH

**Clicks:** on the home side, "Download the router package" (Flint 11–13/18, or
Fedora 15 "Add a GL.iNet travel router (recommended)"). On the Slate 7: SYSTEM →
Advanced Settings → "Go To LuCI" (01a; "Use the same admin password to log in").
Then in LuCI: System → Software → "Upload Package…", pick
`tailbond_0.1.0-r42_aarch64_cortex-a53_neon-vfpv4.ipk (6.1 MB)`, and in the
"Manually install package / Really install the uploaded package?" dialog click
**Install** (01b).

**Friction:**
- Speedify on GL is a one-click install from GL's own UI. Here I go through
  LuCI, which I've never needed for Speedify. The 3-step instructions on the home
  page match the screens 01a/01b exactly, so I wouldn't get stuck. It's just more
  steps than I'm used to. (low)
- The dialog has an unchecked "Allow overwriting conflicting package files". The
  home page says "leave the other options as they are", so I'd leave it. I only
  noticed that line because I re-read the steps. (low)

**Expected:** a GL plugin entry, or at least the GL UI's own upload. It works
without SSH, which is what the task asked for.

## Task 2: Pair with the home server

**Clicks:** VPN → tailbond (02 shows "Looking for your home server on your
tailnet…"), then 03 "Found your home server: homebox · homebox.tail1234.ts.net ·
100.101.1.1", then **Ask to connect**. 04 shows "Waiting for approval on
homebox… On homebox, tap **Approve** next to **slate7**. This router connects by
itself once approved; you can close this page."

**Friction:**
- Speedify needs an account login. Here it finds the server by itself, which is
  nicer. The fallbacks (03b "Turn on Tailscale first" with "Open Tailscale
  settings", 03c "No home server found… Checked 4 online machines", and typing
  the name) are clear.
- 04 calls this router **slate7**, while the GL header and the home-side request
  say "GL-BE3600". On the home side "slate7 wants to connect … GL-BE3600" puts
  the two names together, so I'd match them. (low)

## Task 3: Hotel Wi-Fi (Repeater) + phone USB tether, confirm both used

**Clicks:** none. Look at 05: header "Connected to homebox / Using Repeater
(Wi-Fi) + Tethering (phone)", "↓ 59 Mbps ↑ 7.1 Mbps Up 2 h 13 min". The Repeater
and Tethering cards both say "In use" with their own speed and latency
("↓ 38 Mbps ↑ 4.2 Mbps · 24 ms", "↓ 21 Mbps ↑ 2.9 Mbps · 48 ms").

**Friction:**
- This is the equivalent of Speedify's "Using 2 connections", and it's clear.
  Speedify also shows live per-connection graphs; here I get numbers only. The
  Fedora page (14) has graphs and per-path stats, but the travel router doesn't.
  Fine for confirming, but I'd miss the graphs when troubleshooting. (low)
- 06 (phone trouble): the Tethering card says "Not working right now. It's
  connected, but can't reach homebox. tailbond keeps trying; the other
  connections carry your traffic." That's clearer than Speedify's red bar.

## Task 4: Phone as backup only

**Clicks:** Tethering card → "Use it" → **Backup only** (07). The status changes
to "Backup · saving data. Used only if your other connections stop working.
Until then it isn't used at all, so it uses no data." The header now says
"Tethering (phone) on standby". Optionally "Change limit" → "5 GB a month" →
Save (07b: "At the limit, tailbond stops using this connection until the count
restarts on the 1st of each month. 0 = no limit.").

**Friction:**
- Speedify has four levels (Always / Secondary / Backup / Never). Here there are
  three: "Always / Backup only / Off". I'd miss "Secondary" (used only when the
  main link is slow, not just when it's dead), but for this task "Backup only" is
  exactly the right button. (low)
- The mode picker maps well: "Fastest" = Speed, "Smooth calls and video" =
  Streaming, "Most reliable" = Redundant. The hint under it ("Saving your phone's
  data? Set Tethering or Cellular to 'Backup only' under Internet connections")
  sends me to the right control. Good.
- The limit resets on the 1st of each month, and I can't choose a billing day
  like I can in Speedify. My carrier cycle starts on the 14th. (low)

## Task 5: Reach the NAS by MagicDNS name

**Clicks:** "Tailnet machines" (05): "nas / nas.tail1234.ts.net / 100.101.2.2 /
linux · Online · direct", then **Copy name**. The Home server card says "Tailnet
names: Work on every device connected to this router while tailbond is on. Use
the full name, like nas.tail1234.ts.net."

**Friction:**
- Speedify has nothing like this, so it's a plus. On my own Tailscale devices I
  normally type just `nas`. The page says to use the full name, so I would, but
  I'd first try `nas` and wonder why it fails. The rule is shown twice, so
  that's on me. (low)

## Task 6: Turn tailbond off

**Clicks:** the big switch at the top right (05). 09 shows "tailbond is off.
Devices on this router use the internet directly, without your home server. Turn
on to bond your connections again." Unpair is a separate red button with a
confirm (09b: "This router stops using tailbond and forgets the home server…").

**Friction:**
- Same as Speedify's switch, so no trouble. One worry: when it's off, the
  Tethering card still shows "Backup only" but says "Ready · Used when tailbond
  is on." The page doesn't tell me whether GL's own multi-WAN might now send
  traffic over my phone. In Speedify, turning it off means it's GL's problem,
  and that's the same here, but I'd like one line saying so. (low)

## Task 7: Approve on the home side

**Clicks (Flint 3):** VPN → tailbond, then the "Waiting for your approval · 1 new"
card: "slate7 wants to connect · Travel router on your tailnet · signed in as
alex@example.com · GL-BE3600 · linux · 100.101.9.9 · asked just now", then
**Approve** (11). 12 then lists "slate7 · Connected just now · 2 connections
bonded".
**Clicks (Fedora):** http://homebox:8077, then the top card "slate7 wants to
connect", then **Approve** (14).

**Friction:**
- Easy on both sides. The note "Who can approve: anyone … on your Tailscale
  account who opens http://homebox:8077" surprised me (Speedify has nothing open
  like that), but it tells me where to set a password. (low)
- On the Fedora page, the header switch on the *server* is labelled
  "Connected". Connected to what? On the Flint it says "Home server is on". (low)

## Task 8: Hotel Wi-Fi with a sign-in page (GL's repeater detected it)

**Clicks:** 16 shows the header "Using Tethering (phone) · Repeater (Wi-Fi) needs
sign-in" and a blue banner: "Repeater needs you to sign in. GL's Repeater page
handles this sign-in. Open it, click 'Login Mode for Public Hotspots' (or 'Go to
Login') and accept the hotel's terms." I'd click **Open GL's Repeater page**,
then start Login Mode. Back on tailbond (16b), the banner says "Signing in to
Repeater through GL's Login Mode … It ends by itself once you've signed in (or
click 'Quit Login Mode' on GL's Repeater page); tailbond then adds Repeater back
to bonding."

**Friction:**
- This is the closest match to Speedify's behaviour: the bond stays up on the
  phone ("Your other connections keep working meanwhile"), and it rejoins
  Repeater by itself. But it's a hop to a different page (GL's Repeater) rather
  than "tap the notification and the hotel page opens". It's doable because the
  button names the exact GL button to press. (low)
- **During sign-in, everything bypasses the tunnel.** 16b: "all traffic from
  devices here goes out the hotel Wi-Fi directly (not just the sign-in page), not
  through your home server". Yet the big title above it still says "Connected to
  homebox" in green. Speedify only lets the portal page out; everything else
  stays on the VPN. I'd read the green title first and assume I'm protected. The
  sentence underneath does say otherwise, and I'd catch it, but I'd stop
  background syncs until done. It's worded honestly, but it doesn't match the
  green title. (low; I'd make it medium if the subtitle weren't right under the
  title)

## Task 9: Hotel Ethernet with a sign-in page

**Clicks:** 17 shows "Ethernet needs you to sign in. GL's Login Mode covers the
repeater Wi-Fi only; tailbond handles sign-in pages on other connections like
this one." Click **Open sign-in page**. The hotel page opens in a new tab. In 17b
the banner says "Signing in to Ethernet… For up to 9 more minutes (until 04:18 on
this router's clock)…", with "Open the sign-in page again", "I've signed in,
check now" and "Stop sign-in mode".

**Friction:**
- This is the Speedify experience I wanted: one button, the hotel page opens,
  then it rejoins the bond. "I've signed in, check now" is better than Speedify,
  which makes me wait.
- Wi-Fi and Ethernet use **two different sign-in flows** (GL's Login Mode vs
  tailbond's own button). The 17 banner explains why, but as a Speedify user I'd
  expect the same "Open sign-in page" button for both. (low)
- "until 04:18 on this router's clock": the router's clock may not match mine
  when I'm in a different timezone. The countdown "9 more minutes" is what I'd
  actually use. (low)
- Same full-bypass caveat as task 8 ("all traffic from devices here goes out
  Ethernet directly"), stated clearly. (covered in task 8 finding)

## Task 10: Home side set up for best speed

**Flint 3:** the "Setup health" card.
- 18c: "All good · Everything is set up for best speed", with checks for Tailscale
  direct, UPnP, route approved, public address, "UDP 443 reached from the
  internet". Done.
- 18: "2 to improve … (tailbond works meanwhile, just slower)". "On this router:
  NETWORK → UPnP, turn it on." That's a concrete action.
- 13: "2 setup steps left": Step 1 approve the route in the Tailscale admin
  console ("Open admin console"), Step 2 Dynamic DNS name. Clear.
- 18b: "Home server is on · 1 setup step left" and "Step 1: Your internet line uses
  carrier-grade NAT … travel routers can't reach UDP 443 here. Ask your provider
  for a public IPv4 address".

**Fedora:** 14 banner "Setup health: 2 things to improve for best speed. Show". 15
Devices tab lists them with "Forward UDP port 41641 … or turn on UPnP … How to
fix".

**Friction:**
- **18b contradicts itself.** The CGNAT step says "travel routers can't reach UDP
  443 here", but the Devices card right below shows "pixel-8 · Connected 3 h 0 min
  · 2 connections bonded". So does it work or not? Is my Slate 7 going to connect
  from the hotel? 18b also has no Setup health card (18/18c have one), so I can't
  check the other speed items. Speedify never made me think about my home line,
  so this is where I'd stall. If it's working over some slower fallback, say so
  ("works, but slower via …"). (medium)
- The Flint's UPnP fix and the Fedora "Forward UDP port 41641" fix are understandable.
  "Show this card to whoever set up your internet" is a nice touch. (none)

---

## Overall vs Speedify on GL

Better than Speedify: auto-discovery and approval instead of an account, plain
status messages ("Not working right now … the other connections carry your
traffic"), tailnet names, "I've signed in, check now". Worse: install via LuCI,
no live graphs on the travel router, no "Secondary" priority, fixed reset day for
the data cap, two different captive-portal flows, and during sign-in all traffic
bypasses the tunnel under a still-green "Connected" title.

## Findings

| # | task | finding | severity |
|---|---|---|---|
| 1 | 1 | Install needs LuCI (System → Software → Upload Package…) instead of a GL plugin one-click like Speedify; steps on the home page match screens exactly | low |
| 2 | 1 | LuCI dialog's "Allow overwriting conflicting package files" checkbox; only "leave the other options as they are" tells me to ignore it | low |
| 3 | 2 | Router called "slate7" in 04 but "GL-BE3600" in GL header; matched only on home side | low |
| 4 | 3 | No live per-connection graphs on the travel router (numbers only); Speedify has graphs | low |
| 5 | 4 | Only Always / Backup only / Off; no Speedify-style "Secondary" | low |
| 6 | 4 | Data cap resets "on the 1st of each month"; no billing-day choice | low |
| 7 | 5 | Short name `nas` won't work; must use nas.tail1234.ts.net (stated twice on the page) | low |
| 8 | 6 | When off, Tethering shows "Backup only" + "Used when tailbond is on", but doesn't say whether GL's own failover may now use phone data | low |
| 9 | 7 | Fedora server page header switch labelled "Connected" (vs Flint "Home server is on") | low |
| 10 | 7 | "Anyone on your Tailscale account who opens http://homebox:8077" can approve; surprising default, though it shows where to set a password | low |
| 11 | 8 | Wi-Fi sign-in hops to GL's Repeater page and its Login Mode, rather than Speedify-style "open the hotel page" | low |
| 12 | 8/9 | During sign-in all traffic bypasses the tunnel, while title still says green "Connected to homebox"; explained in subtitle | low |
| 13 | 9 | Two different sign-in flows (GL Login Mode for Repeater, tailbond button for Ethernet) | low |
| 14 | 9 | "until 04:18 on this router's clock" may confuse across timezones; countdown is enough | low |
| 15 | 10 | 18b: CGNAT step says "travel routers can't reach UDP 443 here" while pixel-8 shows "Connected · 2 connections bonded"; no Setup health card on that screen, so I can't tell whether it works or what speed to expect | medium |
