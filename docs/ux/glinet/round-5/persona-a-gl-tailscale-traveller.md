# Round 5: Persona A (GL Tailscale traveller), raw report

Persona: frequent traveller, not a network engineer. I already use GL.iNet travel
routers and know GL's own pages: APPLICATIONS → Tailscale (one Enable toggle) and
the repeater's "Login Mode for Public Hotspots" for hotel sign-in pages. I expect
new GL features to work like those.

Inputs: task-script.md, screens.md, round-5 PNGs (every phone screen, plus desktop
for 05, 11, 13, 15, 16, 16b, 17, 17b, 18, 18b, 18c).

## Task 1: Install on the Slate 7 without SSH

- Start: the Flint 3 panel (11/13/18*) "Add a GL.iNet travel router" card, or the
  Fedora page (15) card "Add a GL.iNet travel router (recommended)". I click
  "Download the router package". Step 2 says: "On the travel router's admin
  page: SYSTEM → Advanced Settings → Go To LuCI (log in with the same password),
  then System → Software → Upload Package…, pick the file and click Install
  (leave the other options as they are). The router's internet keeps working."
- 01a: SYSTEM / Advanced Settings shows "Go To LuCI" and "Use the same admin
  password to log in." This matches the instructions word for word.
- 01b: LuCI dialog "Manually install package / Really install the uploaded
  package?" with `tailbond_0.1.0-r42_aarch64_cortex-a53_neon-vfpv4.ipk (6.1 MB)`,
  an unticked "Allow overwriting conflicting package files", and Install. The
  instructions told me to leave the other options alone, so I leave the box
  unticked and click Install.
- Where I hesitate:
  - I've never been in LuCI. I'd rather install from GL's own plug-ins page.
    Because the steps name each menu exactly, I get through it. (low)
  - The file name `aarch64_cortex-a53_neon-vfpv4` means nothing to me, and the
    download button doesn't say it's the right file for a Slate 7 (GL-BE3600).
    I'd wonder whether I need a different file for my model. (low)
  - The Fedora version of step 2 (15) is shorter than the Flint's: it leaves
    out "(log in with the same password)" and "leave the other options as they
    are". (low)
  - Afterwards I'd look for tailbond under APPLICATIONS, next to Tailscale. Step
    3 says VPN → tailbond, and 02 confirms it's there. Fine once I'm told. (low)

## Task 2: Pair with the home server

- 02: "Looking for your home server on your tailnet…". 03: "Found your home
  server: homebox", "homebox.tail1234.ts.net · 100.101.1.1", and a big "Ask to
  connect" button. Very easy. It feels like GL: Tailscale is already on, and
  it just finds the server.
- 04: "Waiting for approval on homebox… On homebox, tap Approve next to slate7.
  This router connects by itself once approved; you can close this page." It
  also says where to approve ("http://homebox:8077 … At home with a Flint 3, its
  admin panel → VPN → tailbond"). Clear.
- 03b: "Turn on Tailscale first" with "Open Tailscale settings". That's my GL
  Tailscale page. Good.
- 03c: "No home server found on your tailnet / Checked 4 online machines…" plus
  "Search again" and a box to type the name. Clear.
- Small point: while I'm waiting in 04, the top card still says "Not connected
  to a home server / Not set up yet." That reads as if nothing happened, though
  the card below explains it. (low)

## Task 3: Hotel Wi-Fi plus phone tethering, confirm both are used

- 05: The top card says "Connected to homebox / Using Repeater (Wi-Fi) +
  Tethering (phone)" with "59 Mbps / 7.1 Mbps". Repeater and Tethering each show
  a green "In use" and their own speed. Confirmed in a second.
- 06: Tethering shows "Not working right now / It's connected, but can't reach
  homebox. tailbond keeps trying; the other connections carry your traffic."
  Reassuring and clear.
- No issues.

## Task 4: Phone as backup only

- On the Tethering card I click "Use it: Backup only" (07). The badge changes to
  "Backup · saving data" and the text reads "Used only if your other
  connections stop working. Until then it isn't used at all, so it uses no
  data." The hint under Speed or reliability ("Saving your phone's data? Set
  Tethering or Cellular to 'Backup only'") also points here. Easy.
- 07b, "Change limit": "5 GB a month", Save/Cancel, "Counts everything tailbond
  sends and receives over it… 0 = no limit." Understandable.
- Small point: "Counts everything tailbond sends and receives" made me wonder
  whether data used while tailbond is off counts toward the 5 GB. It doesn't
  affect the task. (low)

## Task 5: Reach the NAS by its MagicDNS name

- 05, Tailnet machines: "nas / nas.tail1234.ts.net / 100.101.2.2 / linux · Online
  · direct" with "Copy name". The intro says: "Laptops and phones on this
  router's Wi-Fi or LAN reach these by their full name, e.g. nas.tail1234.ts.net,
  or by their 100.x address." The Home server card adds "Tailnet names: Work on
  every device connected to this router while tailbond is on. Use the full
  name". I'd use nas.tail1234.ts.net.
- When tailbond is off or can't connect (08, 09), the list reads "The list
  appears once tailbond is connected." That's fine, though I can't look up the
  name then. (low, not counted)
- No issues.

## Task 6: Turn tailbond off

- The blue toggle on the status card (05) is just like GL's Tailscale Enable
  toggle. 09: "tailbond is off / Devices on this router use the internet
  directly, without your home server. Turn on to bond your connections again."
  The Home server badge shows "Off". Clear.
- 09b: Unpair is separate and says plainly "This router stops using tailbond
  and forgets the home server… (it will need to approve)." I wouldn't mix it up
  with Off.
- 08: if home can't be reached, "Meanwhile, devices on this router have no
  internet… To use this internet directly for now, turn tailbond off." with a
  "Turn tailbond off" button. Good. It warns me before I panic.
- Small point: in 09, with tailbond off, the Home server card still shows "Ask
  to connect again". I'm not sure whether that turns tailbond back on or asks
  for approval again. (low)
- 10: "tailbond isn't running on this router… Starting it is safe" with "Start
  tailbond". Clear.

## Task 7: Approve on the home side

- Flint 3 (11): The "Waiting for your approval" card is at the top: "slate7 wants
  to connect / Travel router on your tailnet · signed in as alex@example.com /
  GL-BE3600 · linux · 100.101.9.9 · asked just now" with Decline / Approve. I
  click Approve. 12 shows slate7 under Devices, "Connected just now · 2
  connections bonded". Easy.
- Fedora (14 phone, 15 desktop): the same card, "slate7 wants to connect" and
  Approve, at the top. On the phone, a large "Add this phone" card follows. I
  ignore it because Approve is above it. (no issue)
- Confusing detail on 11: Setup health says "UDP 443 reached from the internet —
  slate7 connected from 198.51.100.9, 2 h ago", but the same page says slate7
  "asked just now" and is still waiting for approval. How did it connect 2
  hours ago? (low)
- On Fedora, setting a password means editing `"ui_password" in
  /etc/tailbond/tailbond.json`. I couldn't do that, but it isn't part of this
  task. (low)

## Task 8: Hotel Wi-Fi sign-in page (GL repeater)

- 16: "Repeater needs you to sign in / GL's Repeater page handles this sign-in.
  Open it, click 'Login Mode for Public Hotspots' (or 'Go to Login') and accept
  the hotel's terms." Button "Open GL's Repeater page". This is exactly what I
  already know. It also says "Your other connections keep working meanwhile",
  and Tethering shows "In use". Good.
- 16b, after starting Login Mode: "GL's Login Mode is on: devices here use the
  hotel Wi-Fi directly, not through your home server, so its sign-in page loads.
  It ends by itself once you've signed in (or click 'Quit Login Mode' on GL's
  Repeater page); tailbond then adds Repeater back to bonding." Repeater badge:
  "Signing in…". I understand it gets bonded again by itself.
- Small points (low):
  - In 16b the banner title still says "Repeater needs you to sign in" and still
    starts with "Open it, click 'Login Mode…'", but I've already done that.
  - The top card says "Repeater (Wi-Fi) needs sign-in" while the Repeater card
    says "Signing in…".
  - It doesn't say where the hotel's page will appear (my laptop's browser?). I
    know from GL's Login Mode, but a new user might not.

## Task 9: Sign-in page on the hotel Ethernet cable

- 17: "Ethernet needs you to sign in / It shows a sign-in page before it lets
  anyone online. Until then tailbond keeps Ethernet out of bonding…" Button
  "Open sign-in page". The small print says devices reach Ethernet directly "for
  up to 10 minutes".
- 17b: "Signing in to Ethernet… Sign in on the page that opened (accept the
  terms, enter the room number…). For up to 9 more minutes (until 04:13)…
  tailbond adds it back by itself once you're signed in." Buttons: "Open the
  sign-in page again", "I've signed in, check now", "Stop sign-in mode". Clear;
  I'd finish it.
- Small points (low):
  - For Wi-Fi I'm sent to GL's Login Mode, but for the cable tailbond has its own
    button. I'd wonder why it works differently. The banner sends me the right
    way each time, though.
  - "until 04:13": I'm not sure which clock that is (router time? my laptop?).
    "9 more minutes" is enough.

## Task 10: Home set up for best speed

- Flint 3, all good (18c): "Setup health · All good / ✓ Everything is set up for
  best speed", "Tailscale: 2 machines direct", "Network is friendly to direct
  connections (port mapping: UPnP)", "Route 10.77.0.0/24 approved", "Public
  internet address 203.0.113.7", "UDP 443 reached from the internet". It's in
  the Flint panel itself, so I'm done.
- Freshly set up (13): "Home server is on · 2 setup steps left". Step 1: "In
  the Tailscale admin console, open homebox, choose Edit route settings, tick
  the one route listed (10.77.0.0/24) and save", with "Open admin console". I've
  never used the Tailscale admin console because GL's page is a single toggle,
  but the steps are exact. Step 2 links "APPLICATIONS → Dynamic DNS", a GL page
  I know. Nice. (low: the admin console is unfamiliar)
- CGNAT (18b): "Step 1: Your internet line uses carrier-grade NAT… Ask your
  provider for a public IPv4 address (or use public IPv6). How to fix. Not
  sure? Show this to whoever set up your internet." I'd call my ISP. Clear.
  Small point: the Port row on the same card still says "UDP 443, opened on this
  router automatically." That seems to contradict "travel routers can't reach
  UDP 443 here". (low)
- **Relayed (18) and Fedora (15): this is where I get stuck.** The panel shows
  "Setup done" (green) and, just below, "Setup health · 2 to improve". The
  fixes:
  - "Usually fixed by forwarding UDP port 41641 to this machine, or by turning on
    UPnP on the router in front of it."
  - "Hard NAT…; no port mapping on the router. Forward UDP port 41641 to this
    machine, or turn on UPnP on the router in front of it."

  On the Flint 3, "this machine" is the Flint. Which "router in front of it"? A
  few lines up, the Port row says "Is your internet cable plugged straight into
  this router? Then nothing else to do." My Flint is plugged straight in, so is
  there nothing to do, or should I turn on UPnP on the Flint itself (GL NETWORK →
  UPnP) or on my ISP modem? Step 2 links the Dynamic DNS page, but there's no
  GL menu path here. On Fedora (15), "the router in front of it" is my Flint,
  but nothing tells me where port forwarding or UPnP is on a Flint. "How to fix"
  links exist, and "Show this card to whoever set up your internet" helps, but
  that person is me. Both items also repeat the same fix, so it looks like two
  problems when it's one job. (medium)
- "To check one: tailscale ping pixel-8" is a command I can't run without SSH.
  (low)

## Findings

| # | task | finding | severity |
|---|---|---|---|
| 1 | 1 | Installing goes through LuCI (01a/01b), not a GL plug-ins page. Unfamiliar, but the exact menu path gets me through | low |
| 2 | 1 | "Download the router package" doesn't say it fits my model. The file name `aarch64_cortex-a53_neon-vfpv4.ipk` means nothing to me | low |
| 3 | 1 | Fedora's step 2 (15) leaves out "(log in with the same password)" and "leave the other options as they are", which the Flint version has | low |
| 4 | 1 | I'd look for tailbond under APPLICATIONS next to Tailscale; it's under VPN (step 3 says so) | low |
| 5 | 2 | While waiting (04), the top card still says "Not connected to a home server / Not set up yet" | low |
| 6 | 4 | "Counts everything tailbond sends and receives over it": unclear whether data used with tailbond off counts toward the limit | low |
| 7 | 6 | With tailbond off (09), "Ask to connect again" still shows, and it's unclear what it does compared with the toggle | low |
| 8 | 7 | 11: Setup health says "slate7 connected from 198.51.100.9, 2 h ago" while slate7 "asked just now" and is still pending | low |
| 9 | 7 | Fedora password needs editing `/etc/tailbond/tailbond.json` | low |
| 10 | 8 | 16b: banner title and first line still say "Repeater needs you to sign in… click 'Login Mode…'" after Login Mode is on; top card says "needs sign-in" while the card says "Signing in…" | low |
| 11 | 8 | Doesn't say where the hotel page appears (open a browser on the laptop) | low |
| 12 | 9 | Wi-Fi sign-in uses GL's Login Mode, but cable sign-in uses tailbond's own "Open sign-in page". Two methods for the same thing | low |
| 13 | 9 | 17b "until 04:13": unclear which clock that is | low |
| 14 | 10 | 13 Step 1 sends me to the Tailscale admin console (Edit route settings), which I've never used; steps are exact | low |
| 15 | 10 | 18b: Port row says "opened on this router automatically" next to the CGNAT warning that travel routers can't reach UDP 443 | low |
| 16 | 10 | 18/15: the fix "Forward UDP port 41641 to this machine, or turn on UPnP on the router in front of it" is unclear on a Flint, where "this machine" is the router and the Port row says "plugged straight into this router? Then nothing else to do". No GL menu path (NETWORK → UPnP / port forwarding). The same fix appears twice, and it sits under a green "Setup done" | medium |
| 17 | 10 | "To check one: tailscale ping pixel-8" is a command I can't run without SSH | low |
