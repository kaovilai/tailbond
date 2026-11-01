# Round 7 — GL persona C (family member)

Persona: a less-technical family member. Someone else set up the home server; I was handed a
GL.iNet Slate 7 travel router and asked to get it working. I know Wi-Fi and phone hotspots, not
routes, ports or NAT.

Materials: only `task-script.md`, `screens.md` and the PNGs in `packet-gl/` (desktop versions of
every screen used; phone versions of 01b, 05 and 16 checked for layout).

Overall: I could finish all 10 tasks. The router screens mostly talk my language ("Backup only",
"Used only if your other connections stop working", "Copy name"), and the home side has a
"show this to whoever set up your internet" option when things get technical. The two places I
really got confused were the hotel Wi-Fi sign-in (where do I actually sign in?) and one home
screen that says two opposite things about whether the travel router works.

---

## Task 1 — Install tailbond on the Slate 7 without SSH

**Where I click**
- On the home server's page (15, Fedora) under "Add a GL.iNet travel router (recommended)": I
  already clicked "Download the router package". Step 2 says: "On the travel router's admin page:
  SYSTEM → Advanced Settings → Go To LuCI, then System → Software → Upload Package…, pick the
  file, Install."
- 01a: SYSTEM → Advanced Settings, button "Go To LuCI". The page says "Use the same admin password
  to log in." Good, that tells me the password.
- 01b: LuCI → Software → "Upload Package…", picked
  `tailbond_0.1.0-r42_aarch64_cortex-a53_neon-vfpv4.ipk (6.1 MB)`, dialog "Really install the
  uploaded package?" → "Install".

**Where I hesitate**
- The LuCI dialog has a checkbox, "Allow overwriting conflicting package files". The Fedora
  page's instructions (15) don't say whether to tick it. The Flint panel's version (11–13) does
  say "(leave the other options as they are)" and "(log in with the same password)", but the
  Fedora page leaves both out. I'd leave it unticked, but I'd be nervous. **Low.**
- There's no screen for what happens after "Install" (did it work?), and nothing says how to
  get from LuCI (a different-looking site at `http://192.168.8.1:8080/cgi-bin/luci`) "Back in
  the router's admin page". I'd guess the old browser tab or 192.168.8.1. **Low.**

## Task 2 — Pair with the home server

**Where I click**
- 02: VPN → tailbond: "Looking for your home server on your tailnet…".
- 03: "Found your home server: homebox" → "Ask to connect". Easy.
- 04: "Waiting for approval on homebox… On homebox, tap Approve next to slate7. This router
  connects by itself once approved; you can close this page." It also tells me where to approve
  ("open http://homebox:8077 on any device signed in to your Tailscale account (a phone works
  too)").

**Where I hesitate**
- Words like "tailnet" and "Home server's name in Tailscale" (02, 03b, 03c) mean nothing to me.
  It didn't matter because 03 found the server by itself. If I'd landed on 03c ("Checked 4
  online machines… Or type its Tailscale name below") I'd be stuck and would have to call
  whoever set it up. **Low.**
- The router's badge at the top says "GL-BE3600", but the approval text says "slate7". I worked
  out that slate7 is mine, but nothing on the router page says "this router is called slate7"
  until after it connects ("this router is slate7" on 05). **Low.**

## Task 3 — Hotel Wi-Fi (Repeater) + phone USB tethering, both used

**Where I look**: 05. The top says "Connected to homebox · Using Repeater (Wi-Fi) + Tethering
(phone)", and both cards have a green "In use" with speeds. That's clearly "both are being used".

**Where I hesitate**
- The hotel Wi-Fi is only called "Repeater" (with "192.168.20.57" on other screens). It doesn't
  show the hotel network's name, so I have to trust that "Repeater" means the hotel Wi-Fi. The
  task told me "Repeater mode", so I got it. **Low.**

## Task 4 — Phone as backup only

**Where I click**: on 05, the Tethering card's "Use it" row → "Backup only". On 07 the card says
"Backup · saving data", with "Used only if your other connections stop working. Until then it
isn't used at all, so it uses no data." The top changes to "Tethering (phone) on standby". There's
even a hint under Speed or reliability: "Saving your phone's data? Set Tethering or Cellular to
'Backup only'". Very clear. No problems. (07b "Change limit" is also plain: "5 GB a month… 0 = no
limit.")

## Task 5 — Reach the NAS by its MagicDNS name

**Where I look**: 05, "Tailnet machines": "nas / nas.tail1234.ts.net / 100.101.2.2" with a "Copy
name" button. The Home server card says "Tailnet names: Work on every device connected to this
router while tailbond is on. Use the full name, like nas.tail1234.ts.net." So the name is
`nas.tail1234.ts.net`, and I click "Copy name".

**Where I hesitate**
- The word "MagicDNS" isn't anywhere on screen. I guessed the "full name" is what was meant. The
  explanation "Laptops and phones on this router's Wi-Fi or LAN reach these through the bond to
  homebox (not this router's own Tailscale), by their full name… or by their 100.x address.
  'Direct' / 'via relay' is how homebox reaches each machine." is mostly lost on me ("LAN",
  "100.x", "not this router's own Tailscale", "via relay"). The "Copy name" button saves the day.
  **Low.**

## Task 6 — Turn tailbond off

**Where I click**: the blue switch at the top right of the status card (05). 09 shows "tailbond is
off. Devices on this router use the internet directly, without your home server. Turn on to bond
your connections again." Clear.

**Where I hesitate**
- The switch has no label (no "On/Off" text next to it on the router). Meanwhile there's a red
  "Unpair" button lower down, next to "Ask to connect again", and "unpair" sounds like "turn off"
  to me. The confirmation (09b) saved me: "This router stops using tailbond and forgets the home
  server… (it will need to approve)", so I'd click Cancel. **Low.**
- On 09, with tailbond off, Repeater and Tethering both say "Ready · Used when tailbond is on.",
  and Tethering still shows "Backup only". I can't tell whether my phone's data is now being used
  or not while tailbond is off. Does "Backup only" still protect it? **Low.**

## Task 7 — Approve the travel router at home

**Where I click**
- Flint 3 (11): the card "Waiting for your approval · 1 new" with "slate7 wants to connect · Travel
  router on your tailnet · signed in as alex@example.com · GL-BE3600" → "Approve". 12 then shows
  slate7 "Connected just now · 2 connections bonded".
- Fedora (14): the same orange box at the very top, "slate7 wants to connect" → "Approve".

Both are obvious, at the top, and show the same GL-BE3600 I'm holding. No problems.

## Task 8 — Hotel Wi-Fi with a sign-in page (GL's repeater detected it)

**Where I click**
- 16: the top says "Using Tethering (phone) · Repeater (Wi-Fi) needs sign-in", and the blue banner
  says "Repeater needs you to sign in. GL's Repeater page handles this sign-in. Open it, click
  'Login Mode for Public Hotspots' (or 'Go to Login') and accept the hotel's terms." → "Open GL's
  Repeater page".
- (GL's Repeater page itself isn't in my screenshots.)
- 16b: "Signing in to Repeater through GL's Login Mode… so the page loads… It ends by itself once
  you've signed in (or click 'Quit Login Mode' on GL's Repeater page); tailbond then adds
  Repeater back to bonding." Repeater shows "Signing in…".

**Where I get stuck**
- **Where is the hotel's page I'm supposed to accept?** 16 says "accept the hotel's terms" and 16b
  says "so the page loads", but neither says where that page shows up: a new tab, a pop-up on my
  laptop, or do I have to open some website myself? The Ethernet version (17b) is clear: "Sign in
  on the page that opened (accept the terms, enter the room number…)". Here I'd sit on 16b
  waiting for "Signing in…" to change and not know I'm the one who needs to do something.
  **Medium.**
- 16b is a wall of text: the top card says "Router connected to homebox · devices signing in.
  Signing in on Repeater (Wi-Fi): all traffic from devices here goes out Repeater (Wi-Fi)
  directly, not through homebox, until you're signed in". The banner repeats it, plus "(that's
  what the top card shows)". There's also no "I've signed in, check now" button like the Ethernet
  flow has (17b), and no screen shows it bonded again afterwards. I'd have to trust "ends by
  itself". **Low.**

## Task 9 — Hotel Ethernet cable with a sign-in page

**Where I click**: 17 shows "Ethernet needs you to sign in" → "Open sign-in page". 17b: the
hotel page opens in a new tab, and the banner says "Sign in on the page that opened (accept the
terms, enter the room number…)", with "Open the sign-in page again", "I've signed in, check now"
and "Stop sign-in mode". This is the clearest flow in the whole set.

**Where I hesitate**
- The first line of 17's banner, in small print, is "GL's Login Mode covers the repeater Wi-Fi
  only; tailbond handles sign-in pages on other connections like this one." It reads like an
  explanation for someone else and comes before the sentence I need. I also wondered why Wi-Fi
  and cable work so differently. And "until 04:40 on this router's clock" makes me ask whether
  the router's clock is even my time. "9 more minutes" alone would do. **Low.**

## Task 10 — At home, everything set up for best speed

**Where I look**
- Flint 3: the "Setup health" card. 18c: "All good · Everything is set up for best speed". Easy
  to read.
- 18 ("2 to improve"): "On this router: NETWORK → UPnP, turn it on." I can do that. Then "If
  another router sits in front of this one (for example your internet provider's box), forward
  UDP port 41641 to this router there." I can't do that, but "Not sure how? Show this card to
  whoever set up your internet" tells me what to do.
- Fedora (14/15): a red strip "Setup health: 2 things to improve for best speed. Show", then the
  Devices tab card with the same kind of text and the same "Show this card to whoever set up
  your internet".

**Where I get confused**
- **18b says two opposite things.** The Setup health card says "1 thing to improve for best speed
  (tailbond works meanwhile, just slower)". But the red Step 1 box above it says "travel routers
  can't reach UDP 443 here… from away they can't until this is fixed", the Port row says "keeps
  travel routers from reaching it", and slate7 shows "Not connected right now". So is my travel
  router just slower, or does it not work at all from the hotel? With "just slower" I'd shrug
  and not bother calling anyone. **Medium.**
- Jargon in the health items: "Hard NAT (the outside port changes per destination); no port
  mapping on the router.", "To check one: tailscale ping pixel-8", "carrier-grade NAT", "public
  IPv4 address (or use public IPv6)". I can't act on any of it. The "show this card to whoever
  set up your internet" escape hatch keeps it from blocking me. **Low.**
- On the "all good" screens (11, 12, 18c), the Port row still says "Is your internet cable plugged
  straight into this router? Then nothing else to do. If it's plugged into another box…, that box
  must forward UDP 443 to this router". I don't know if it's "straight", so I worry I still have a
  job to do, even though health says "UDP 443 reached from the internet". **Low.**
- On the Fedora page (14), the health strip is red/pink like an error for what is only "things
  to improve for best speed". It made me think something was broken. **Low.**
- If I ever saw 13 (freshly set up), Step 1 sends me to "the Tailscale admin console, open
  homebox, choose Edit route settings, tick the one route listed (10.77.0.0/24)". I may not have
  that login, and "route" means nothing to me. Since someone else set this up, I'd forward it.
  **Low.**

---

## Findings

| # | task | finding | severity |
|---|---|---|---|
| 1 | 1 | Fedora page's install steps (15) omit "(leave the other options as they are)" / "(log in with the same password)" that the Flint panel has; LuCI's "Allow overwriting conflicting package files" checkbox left unexplained | low |
| 2 | 1 | No screen after LuCI "Install" (did it work?) and no hint how to get "Back in the router's admin page" from LuCI | low |
| 3 | 2 | "tailnet" / "Home server's name in Tailscale" / "Checked 4 online machines… type its Tailscale name" is jargon; only saved because auto-find worked | low |
| 4 | 2 | Router badge says "GL-BE3600" but approval text says "tap Approve next to slate7"; router never tells me it's called slate7 before pairing | low |
| 5 | 3 | Hotel Wi-Fi only shown as "Repeater" (no network name), so I have to trust that's the hotel | low |
| 6 | 5 | "MagicDNS" not on screen; "(not this router's own Tailscale)… 100.x address… 'via relay'" confusing; "Copy name" rescues it | low |
| 7 | 6 | Main on/off switch unlabeled; red "Unpair" next to "Ask to connect again" looks like "off" (confirm dialog catches it) | low |
| 8 | 6 | When off (09), cards say "Ready · Used when tailbond is on." with Tethering still "Backup only"; unclear if phone data is used now | low |
| 9 | 8 | Hotel Wi-Fi sign-in (16/16b): "accept the hotel's terms" / "so the page loads" never says where the hotel's page appears or that I must open it; the Ethernet flow does ("Sign in on the page that opened") | medium |
| 10 | 8 | 16b is a wall of text repeated in top card and banner; no "I've signed in, check now" like 17b; no confirmation of being bonded again | low |
| 11 | 9 | 17 starts with small-print "GL's Login Mode covers the repeater Wi-Fi only; tailbond handles sign-in pages on other connections…"; "until 04:40 on this router's clock" adds doubt | low |
| 12 | 10 | 18b contradicts itself: health says "(tailbond works meanwhile, just slower)" but Step 1 says travel routers "can't [connect] until this is fixed" and slate7 is "Not connected right now" | medium |
| 13 | 10 | Health items full of jargon ("Hard NAT (the outside port changes per destination)", "tailscale ping pixel-8", "carrier-grade NAT", "public IPv4"); the "show this card to whoever set up your internet" escape hatch helps | low |
| 14 | 10 | Port row keeps "If it's plugged into another box…, that box must forward UDP 443" even when health shows all good / "UDP 443 reached from the internet" | low |
| 15 | 10 | Fedora's health strip (14) is red/pink like an error for "things to improve for best speed" | low |
| 16 | 10 | 13's Step 1 needs the Tailscale admin console and "Edit route settings… (10.77.0.0/24)", which a family member can't do | low |

Totals: 0 high / 2 medium / 14 low
