# Round 9 — GL persona D (Speedify user)

**Persona:** I've run Speedify on GL.iNet routers for years, from GL's VPN menu and from the Speedify app. I judge everything against that: bonding modes (Speed / Streaming / Redundant), per-connection priority (Always / Secondary / Backup / Never), monthly data limits, live per-connection graphs, and what happens when a link drops.

**Materials:** task-script.md, screens.md and the PNGs in `out/ux/final/round-9/packet-gl/`. I looked at the desktop version of every screen and at the phone versions of 05 and 17b (the phone layouts stack the same cards in a single column, and I didn't find anything that only goes wrong on the phone).

---

## Task 1: Install tailbond on the Slate 7 without SSH

**Where I click:**
- Screen 15 (Fedora, Devices tab): the "Add a GL.iNet travel router (recommended)" card. I click "Download the router package". Step 2 reads: "On the travel router's admin page: SYSTEM → Advanced Settings → Go To LuCI, then System → Software → Upload Package…, pick the file, Install."
- The Flint version (11–13, 18*) gives the same steps and adds "(log in with the same password)" and "(leave the other options as they are). The router's internet keeps working."
- 01a: SYSTEM → Advanced Settings → "Go To LuCI". It shows the hint "Use the same admin password to log in."
- 01b: LuCI → Software → "Upload Package…". The dialog reads "Really install the uploaded package?" and shows `tailbond_0.1.0-r42_aarch64_cortex-a53_neon-vfpv4.ipk (6.1 MB)`. I leave "Allow overwriting conflicting package files" unticked and click **Install**.
- 02: back in GL's panel, VPN → **tailbond** is now in the menu.

**Where I hesitate:** Speedify on GL shows up as a plugin and installs itself for the right model. Here I get one file from the home server. On the Fedora page the button just says "Download the router package", with no model list. On the Flint it says "It's the same tailbond this router runs", which makes me wonder whether a package built for the Flint 3 also fits a Slate 7. The filename shows an architecture (`aarch64_cortex-a53…`), but nothing on screen says which GL models that covers. LuCI accepted the file without complaint, so I'd carry on.

**Expected:** a short line such as "works on GL routers from firmware 4.x (ARM)" next to the download button.

**Severity:** low. The install finishes, and the doubt is only a moment's worry.

Fedora's step 2 doesn't say to leave the "Allow overwriting…" option alone; the Flint's version does. This isn't a problem in practice.

## Task 2: Pair the Slate 7 with the home server

**Where I click:**
- 02: "Looking for your home server on your tailnet…"
- 03: "Found your home server: homebox · homebox.tail1234.ts.net · 100.101.1.1" → **Ask to connect**.
- 04: "Waiting for approval on homebox… On homebox, tap Approve next to slate7. This router connects by itself once approved; you can close this page." It also points to `http://homebox:8077` or the Flint panel.

**Edge cases:** 03b ("Turn on Tailscale first", with "Open Tailscale settings") and 03c ("No home server found on your tailnet. Checked 4 online machines…", with "Search again" and a box to type the name) are both clear. This is easier than Speedify's account login.

**Findings:** none.

## Task 3: Hotel Wi-Fi (Repeater) and phone USB tethering, confirm both are used

**Where I look (05):**
- The header says "Connected to homebox · Using Repeater (Wi-Fi) + Tethering (phone)", with ↓59 / ↑7.1 Mbps and Up 2 h 13 min.
- Repeater shows "In use ↓ 38 Mbps ↑ 4.2 Mbps · 24 ms".
- Tethering shows "In use ↓ 21 Mbps ↑ 2.9 Mbps · 48 ms".
- 38 + 21 = 59, so the per-link figures are the live split of the total. That confirms both links are bonded.

**Bonding modes:** "Fastest / Smooth calls and video / Most reliable" match Speedify's Speed / Streaming / Redundant almost one to one, and the descriptions are plain. That's good.

**When a link drops (06):** the header changes to "Using Repeater (Wi-Fi) · Tethering (phone) not working right now", and the card reads "It's connected, but can't reach homebox. tailbond keeps trying; the other connections carry your traffic." That's better than Speedify's bare red dot.

**Where it falls short of Speedify:**
- The GL page shows per-link numbers as text only: no live graph per connection, and no loss figure.
- The Fedora page (14) does have per-path sparkline cards with LATENCY and LOSS. But they're labelled "Path 0 · Pixel-8 · Min 31 Ms" and "Path 1", not by what the link is (Wi-Fi or cellular), and "Ms" has a stray capital M.
- In Speedify I can see at a glance which link is jittery.

**Severity:** low. I can still confirm both links are in use, which is the task.

## Task 4: Phone as backup only

**Where I click:**
- 05/07: on the Tethering card, Use it → **Backup only**.
- 07 confirms with the badge "Backup · saving data" and the text "Used only if your other connections stop working. Until then tailbond sends nothing over it (GL itself may check the connection now and then, a few KB)."
- The header changes to "Tethering (phone) on standby".
- The hint under the modes, "Saving your phone's data? Set Tethering or Cellular to 'Backup only'", points the right way.
- Always / Backup only / Off is exactly the Speedify mental model.

**Data limit (07b):** "Change limit" opens a box with "5 GB a month" and the note "Counts everything tailbond sends and receives over it. At the limit, tailbond stops using this connection until the count restarts on the 1st of each month. 0 = no limit." There's a progress bar for "1.2 GB of 5 GB this month". It's clear, with two gaps compared with Speedify:
- **Reset day:** the count always restarts on the 1st. My phone plan resets mid-month, and Speedify lets me pick the day.
- **Limit plus backup:** it doesn't say what happens once the limit is reached and the phone is the only backup left. Does "stops using this connection" mean that even if the hotel Wi-Fi dies, I get no failover?

**"Most reliable" plus "Backup only":** Most reliable says "Sends everything over every connection. Uses the most data." With Tethering on Backup only, I can't tell from this screen whether Most reliable would still duplicate traffic onto the phone. The hint below the modes suggests Backup only wins, but the mode text says "every connection".

**Severity:** low for each. The task is done and the main wording is right.

## Task 5: Reach the NAS by its MagicDNS name

**Where I look (05):**
- The "Tailnet machines" card lists **nas**, `nas.tail1234.ts.net`, `100.101.2.2`, "linux · Online · direct", with a **Copy name** button.
- The card text says "Laptops and phones on this router's Wi-Fi or LAN reach these through the bond to homebox … by their full name, e.g. nas.tail1234.ts.net, or by their 100.x address."
- The Home server card repeats "Tailnet names: Work on every device connected to this router while tailbond is on. Use the full name, like nas.tail1234.ts.net."
- The name is `nas.tail1234.ts.net`. It's clear that the short name `nas` isn't promised to work.

**Findings:** none.

## Task 6: Turn tailbond off

**Where I click:** the blue switch on the right of the "Connected to homebox" card (05).

**Result (09):** "tailbond is off. Devices on this router use the internet directly, without your home server, over whichever connection GL's own settings pick (INTERNET page, top of the menu). Backup only and data limits here apply only while tailbond is on, so GL may use your phone: unplug it (or turn tethering off) if you don't want that."

That's exactly the warning a Speedify user needs. "Unpair" is separate, and it has a confirmation (09b): "This router stops using tailbond and forgets the home server…". Good.

**Related wording problem (08, server unreachable, tailbond still on):** the Repeater and Tethering cards say "This choice applies when tailbond is on. While it's off, GL's own settings…". That text belongs to the off state, but here the switch is on. It makes me double-check whether tailbond is actually on.

**Severity:** low.

## Task 7: Approve the travel router at home

**Flint 3 (11):**
- "Waiting for your approval · 1 new · slate7 wants to connect · Travel router on your tailnet · signed in as alex@example.com · GL-BE3600 · linux · 100.101.9.9" → **Approve**.
- Afterwards (12), slate7 appears under Devices as "Connected just now · 2 connections bonded · downloading 48 Mbps · uploading 6.1 Mbps". That's a nice confirmation that both of my links are bonded, seen from the home side.

**Fedora (14/15):** the same request card is at the top, with Approve and Decline.

**Findings:** none. This is clearer than anything Speedify has, because Speedify has no approval step.

## Task 8: Hotel Wi-Fi sign-in page, with tailbond on

**Where I click:**
- 16: the banner "Repeater needs you to sign in. GL's Repeater page handles this sign-in. Open it and click 'Login Mode for Public Hotspots' (or 'Go to Login')…" → **Open GL's Repeater page**. The packet doesn't include GL's Repeater page itself, so I'm going by these instructions.
- 16b: the header reads "Router connected to homebox · devices signing in". The banner explains that everything goes out the hotel Wi-Fi directly until I'm signed in, that "your phone's data isn't used for them", and that "Login Mode ends by itself once you've signed in (or click 'Quit Login Mode' on GL's Repeater page); tailbond then adds Repeater back to bonding."
- I open any site on the laptop, accept the terms and wait.

**Inconsistency:** the Ethernet sign-in flow (17b) has "I've signed in, check now" and a countdown ("until 05:08 on this router's clock"). The Wi-Fi flow (16b) only has "Back to GL's Repeater page", so after signing in I just wait with no button to speed things up and no way to tell how long it'll take. Speedify users are used to a manual "retry" button.

**Severity:** low.

## Task 9: Sign-in page on the hotel Ethernet

**Where I click:**
- 17: "Ethernet needs you to sign in" → **Open sign-in page**.
- 17b: "Signing in to Ethernet… For up to 9 more minutes (until 05:08 on this router's clock)…". I sign in, then click "I've signed in, check now". "Stop sign-in mode" is there if I change my mind.
- Tethering stays on "Backup · saving data" throughout. That's good.

**Layout of the 17 banner:** the first line under the title, "GL's Login Mode covers the repeater Wi-Fi only; tailbond handles sign-in pages on other connections like this one.", is in small print and comes before the main sentence ("It shows a sign-in page before it lets anyone online…"), which is in larger print. Then there's another small-print line. The reading order is: aside, main point, aside. I read the aside first and briefly wondered whether I needed GL's Login Mode here.

**Severity:** low.

## Task 10: Home side, best speed

**Flint 3:**
- 13: "Home server is on · 2 setup steps left", with Step 1 (approve route 10.77.0.0/24, plus an "Open admin console" button) and Step 2 (Dynamic DNS name). Clear.
- 18: Setup health "2 to improve", e.g. "On this router: NETWORK → Firewall → Open Ports on Router, add UDP 41641…", with "How to fix".
- 18b: the carrier-grade NAT case is red and honest: "Bonding needs that direct connection; it can't run through Tailscale's relays".
- 18c: "All good · Everything is set up for best speed".

**Fedora:** in 14, the Connections tab has a collapsed bar, "Setup health: 2 things to improve for best speed (tailbond works meanwhile). Show". The full list is on the **Devices** tab (15).

**Small confusions:**
- **Fedora tab:** I wouldn't think to look under "Devices" for router and port advice. The "Show" link gets me there, though.
- **"No port mapping" with a green check:** 18c (and 11/12) show the green check "Network is friendly to direct connections" with the subtitle "no port mapping on the router", while the card above says "UDP 443, opened on this router automatically". "No port mapping" under a green check looks like a contradiction. In 18 the same phrase appears as part of a *problem*.
- **Mixed signals in 18:** the green "Setup done" box sits above an amber "2 to improve" health card.
- **Wrong subtitle on the home server:** the Flint page subtitle still says "Bond this router's internet connections into one tunnel to your home server.", but this router *is* the home server.

**Severity:** low for each.

---

## Overall, from a Speedify user

The mental model carries over cleanly: three bonding modes, Always / Backup only / Off per link, monthly limits with a progress bar, and honest "not working right now" states. Pairing and approval are smoother than Speedify's account login. What I miss is mostly the instrumentation: live graphs and loss per connection on the router itself, links labelled by type on the home server's path view, a choice of reset day for limits, and a clear statement of how limits and "Most reliable" interact with Backup only.

## Findings

| # | Task | Finding | Severity |
|---|---|---|---|
| 1 | 1 | "Download the router package" (Fedora) and "It's the same tailbond this router runs" (Flint) don't say which GL models or firmware the file fits; the filename only shows `aarch64_cortex-a53`. | low |
| 2 | 3 | The GL page has per-link text only (↓/↑/ms): no live per-connection graph and no loss. Fedora's per-path cards are labelled "Path 0 / Path 1 · Min 31 Ms", not by link type. | low |
| 3 | 4 | The data limit always "restarts on the 1st of each month" (no reset day to pick), and it doesn't say whether a link at its limit still acts as backup if the main link fails. | low |
| 4 | 4 | "Most reliable: Sends everything over every connection" vs Tethering "Backup only": unclear whether Most reliable would still use the phone's data. | low |
| 5 | 6 | In 08 (tailbond on, but home server unreachable), the connection cards say "This choice applies when tailbond is on. While it's off, GL's own settings…", which is text for the off state. | low |
| 6 | 8 | The Wi-Fi sign-in flow (16b) has no "I've signed in, check now" button or time remaining, unlike the Ethernet flow (17b). | low |
| 7 | 9 | In the 17 banner, the small-print aside "GL's Login Mode covers the repeater Wi-Fi only…" comes before the main sentence, so the reading order is off. | low |
| 8 | 10 | On the Fedora page, the full setup health list is under the "Devices" tab; the Connections tab only has a collapsed "Show" bar. | low |
| 9 | 10 | A green check sits on "Network is friendly to direct connections · no port mapping on the router" next to "UDP 443, opened on this router automatically"; in 18, green "Setup done" sits above amber "2 to improve". Mixed signals. | low |
| 10 | 10 | The Flint home server page subtitle still says "Bond this router's internet connections into one tunnel to your home server." | low |

Totals: 0 high / 0 medium / 10 low
