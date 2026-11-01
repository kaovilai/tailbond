# Round 8 — GL persona D (Speedify user)

**Who I am:** I've run Speedify for years, including on a GL.iNet router through GL's VPN menu. I judge everything against Speedify: bonding modes (Speed / Streaming / Redundant), the per-connection priority (Always / Secondary / Backup / Never), data limits, the live per-connection graphs, and what happens when one link drops.

**Materials:** `task-script.md`, `screens.md` and the PNGs in `packet-gl/` only. I looked at every desktop screen and checked phone versions of 05 and 14. Note: `14-fedora-dashboard-request-desktop.png` is cut off at 800 px, so I can't read the per-path DOWN/UP/LATENCY/LOSS numbers on that screen.

**Overall:** this is the closest thing to Speedify-on-GL I've seen. The per-connection **Always / Backup only / Off** maps directly onto Speedify's priorities. **Fastest / Smooth calls and video / Most reliable** are Speedify's Speed / Streaming / Redundant in plainer words. I could finish every task. The rough edges are about trust and monitoring, not about getting stuck.

---

## Task 1 — Install on the Slate 7 without SSH

**Clicks:**
1. On the home server, click "Download the router package". It's on Fedora screen 15 under "Add a GL.iNet travel router (recommended)", and on the Flint 3 under "Add a GL.iNet travel router" (11–13).
2. On the Slate 7: SYSTEM → Advanced Settings → **"Go To LuCI"** (01a). The page says "Use the same admin password to log in."
3. In LuCI (01b): System → Software → **"Upload Package…"**, pick `tailbond_0.1.0-r42_aarch64_cortex-a53_neon-vfpv4.ipk (6.1 MB)`. The dialog asks "Really install the uploaded package?" and I click **Install**.
4. VPN → **tailbond** now appears in the GL menu (02).

**Friction:**
- The Flint's step 2 helps: "(log in with the same password)" and "pick the file and click **Install** (leave the other options as they are)". The Fedora page (15) only says "pick the file, Install." Without that, the unchecked "Allow overwriting conflicting package files" box in 01b makes me hesitate. **Low.**
- I get one file with a CPU name in it (`aarch64_cortex-a53_neon-vfpv4`). Nothing says it fits a GL-BE3600. The Flint's "It's the same tailbond this router runs" tells me it fits the *Flint*, not the Slate 7. With Speedify on GL I picked the package for my model. **Low.**
- The LuCI detour is more steps than Speedify's GL plugin, but the instructions name every click exactly, so it was fine.

## Task 2 — Pair with the home server

**Clicks:** 02 "Looking for your home server on your tailnet…" → 03 "Found your home server: homebox" → **"Ask to connect"** → 04 "Waiting for approval on homebox… On homebox, tap **Approve** next to **slate7**."

**Friction:**
- 02/03 say the home server "must be signed in to the same Tailscale account as this router". The Fedora page (14/15) says homebox is "a tagged machine, so it has no single owner". If the box isn't signed in to my account, would pairing fail? It did work (the fixture shows slate7 asking). **Low.**
- Speedify needs an account login; here there's nothing to type. Good. 03b ("Turn on Tailscale first", "Open Tailscale settings") and 03c ("Checked 4 online machines…", "Search again") are clear error states.

## Task 3 — Hotel Wi-Fi (Repeater) + phone over USB, confirm both are used

**Where I look:** 05. The top card says "Connected to homebox · Using Repeater (Wi-Fi) + Tethering (phone)". Repeater shows "In use · ↓ 38 Mbps ↑ 4.2 Mbps · 24 ms". Tethering shows "In use · ↓ 21 Mbps ↑ 2.9 Mbps · 48 ms". Bonding mode is "Fastest". So yes, both are in use, confirmed.

**Link drop (06):** Tethering shows a red "Not working right now" and "It's connected, but can't reach homebox. tailbond keeps trying; the other connections carry your traffic." The top card says "Tethering (phone) not working right now". That's better than Speedify's usual icon-only change.

**Friction:**
- The top card reads "↓ 59 Mbps ↑ 7.1 Mbps Up 2 h 13 min". With an ↑ arrow right next to it, "Up" first reads as upload. It's actually uptime. **Low.**
- Speedify's main draw for me is the live per-connection graph. The Slate 7 page shows current numbers only: no graph, no loss %. On the Flint 3 (12) slate7 is one line: "2 connections bonded · downloading 48 Mbps · uploading 6.1 Mbps". The Fedora page (14) does have per-path graphs, but they're labelled "Path 0 / Path 1 · Pixel-8 · Min 31 Ms" (15: "#0 active · 31 ms · 0.0%"). They aren't named Wi-Fi / Tethering, so I can't tell which link is which. **Low.**

## Task 4 — Phone as backup only

**Clicks:** On the Tethering card, **"Backup only"** (07). The status changes to "Backup · saving data": "Used only if your other connections stop working. Until then it isn't used at all, so it uses no data." The top card says "Tethering (phone) on standby". The hint under the modes, "Saving your phone's data? Set Tethering or Cellular to 'Backup only'", points to the same place. Done in one click, the same as Speedify's Backup priority.

**Limit (07b):** "Change limit" opens a field with "5" and "GB a month", plus Save/Cancel. The help text: "Counts everything tailbond sends and receives over it. At the limit, tailbond stops using this connection until the count restarts on the 1st of each month. 0 = no limit."

**Friction:**
- Only a monthly limit, and it always resets "on the 1st of each month". My phone plan doesn't reset on the 1st, and Speedify lets me set the reset day and a daily limit. The counter also only counts tailbond traffic, so it won't match my carrier's meter. **Low.**
- "Stop working" isn't defined. Speedify's backup also kicks in on poor quality, not just a dead link. I can't tell whether a very slow or lossy hotel Wi-Fi will bring the phone in. **Low.**

## Task 5 — Reach the NAS by MagicDNS name from the laptop

**Where:** On 05, the Tailnet machines card shows "nas / nas.tail1234.ts.net / 100.101.2.2 · linux · Online · direct" with a **"Copy name"** button. The Home server card adds: "Tailnet names: Work on every device connected to this router while tailbond is on. Use the full name, like nas.tail1234.ts.net." The answer is **nas.tail1234.ts.net**. Easy.

**Friction:**
- The page says to use the full name but not why. A Tailscale user will try `nas` first. Also, when tailbond is off or can't connect (08, 09), the card only says "The list appears once tailbond is connected." So I can't look the name up unless tailbond is connected. **Low.**

## Task 6 — Turn tailbond off

**Clicks:** The blue switch top-right of the status card (05). Result (09): "tailbond is off. Devices on this router use the internet directly, without your home server, over whichever connection GL's own settings pick (INTERNET page). Backup only and data limits here apply only while tailbond is on." The Unpair button is clearly separate: 09b asks "Unpair from homebox?" and warns "it will need to approve" again.

**Friction:**
- The warning is honest, but it hands me a problem. Turning tailbond off drops my "Backup only" setting, and my phone tether may now carry everything. The page doesn't say which connection GL will pick, and "INTERNET page" isn't a link. With Speedify, "disconnected" never meant "start burning phone data". I'd have to go to GL's INTERNET page and fix multi-WAN priorities myself. **Medium.**
- On the GL page the switch has no label. The Fedora page labels its switch "Connected", and Speedify has a Connect/Disconnect button. 10 also says "It may have been turned off or stopped after an error", which blurs "turned off" (09) with "not running" (10). **Low.**

## Task 7 — Approve the travel router on the home side

**Flint 3 (11):** "Waiting for your approval · 1 new", "slate7 wants to connect · Travel router on your tailnet · signed in as alex@example.com · GL-BE3600 · linux · 100.101.9.9", then **Approve**. Afterwards (12): "slate7 · Connected just now · 2 connections bonded". Clear.

**Fedora (14/15):** the same card at the top, with **Approve**.

**Friction:**
- The Fedora page contradicts itself about who can approve. The approval card says: "Who can approve: anyone on your Tailscale network who can reach homebox (it's a tagged machine, so it has no single owner). Set a password below to limit that." Further down on the same page (15), the "Who can approve and add devices" card says: "Who can approve: alex@example.com (the Tailscale account homebox belongs to)… People you share homebox with, and others on your Tailscale network, can ask to connect but can't approve." One says anyone can approve, the other says only alex. I can't tell whether I need the password, and that's a security setting. **Medium.**

## Task 8 — Hotel Wi-Fi sign-in page (GL repeater detected it)

**Clicks:** 16 shows "Repeater needs you to sign in. GL's Repeater page handles this sign-in. Open it and click 'Login Mode for Public Hotspots'…". I click **"Open GL's Repeater page"**, then GL's Login Mode. 16b says "Signing in to Repeater through GL's Login Mode. Now open any website on your laptop or phone…" and "Login Mode ends by itself once you've signed in…; tailbond then adds Repeater back to bonding." Meanwhile Tethering stays "In use", so I keep working.

**Friction:**
- 16b is one long paragraph. The top card shows "↓ 12 Kbps ↑ 8 Kbps", which looks like the bond collapsed. The explanation ("it carries almost nothing meanwhile") is buried mid-paragraph. The Ethernet sign-in flow (17b) has an "I've signed in, check now" button. This one doesn't, so I just have to trust that it re-adds the Repeater by itself. **Low.**

## Task 9 — Sign-in page on the hotel Ethernet

**Clicks:** 17 says "Ethernet needs you to sign in… Until then tailbond keeps Ethernet out of bonding and your other connections carry your traffic." I click **"Open sign-in page"**. 17b says: "Signing in to Ethernet… For up to 9 more minutes (until 04:57 on this router's clock)…" and offers "Open the sign-in page again", **"I've signed in, check now"** and "Stop sign-in mode". 17 also explains the split: "GL's Login Mode covers the repeater Wi-Fi only; tailbond handles sign-in pages on other connections like this one."

This is clear, and better than Speedify, which just fails on captive portals until you sign in some other way. No findings.

## Task 10 — Best speed at home

**Flint 3:** 18c "Setup health · All good · Everything is set up for best speed" with ticks for Tailscale direct, UPnP, route approved, public address, and UDP 443 reached. 18 (relay) gives a concrete fix: "On this router: NETWORK → Firewall → Open Ports on Router, add UDP 41641". 18b (CGNAT) says plainly "Ask your provider for a public IPv4 address (or use public IPv6)". 13 lists "Step 1… Edit route settings" and "Step 2… Dynamic DNS". Good.

**Fedora (14/15):** a banner says "Setup health: 2 things to improve for best speed (tailbond works meanwhile). Show". On the Devices tab, both items have the same fix text.

**Friction:**
- The Fedora header shows "1.1 MBPS DOWN" and "12.0 MBPS UP". The device row on the same page shows pixel-8 at "↓ 12.0 Mbps ↑ 1.1 Mbps". Down and up are swapped between the two (the server's view vs the device's view). On the speed page that's exactly what I'd misread: I'd think the bond downloads at 1.1 Mbps. The Flint avoids this with "Sending to devices 61 Mbps / Receiving 7.4 Mbps". **Medium.**
- 15 says "(the guide fedora-server.md has a section per item)". That's a filename I don't have or know where to find. Both items also repeat the same "forward UDP port 41641…" fix word for word, so "2 things" feels like one. **Low.**

---

## Findings

| # | Task | Finding | Severity |
|---|---|---|---|
| 1 | 1 | Fedora install steps omit the Flint's "log in with the same password" and "leave the other options as they are"; LuCI's "Allow overwriting conflicting package files" box causes hesitation | low |
| 2 | 1 | Single package `…aarch64_cortex-a53_neon-vfpv4.ipk`; nothing confirms it fits the GL-BE3600 ("same tailbond this router runs" only covers the Flint) | low |
| 3 | 2 | "must be signed in to the same Tailscale account as this router" vs Fedora's "tagged machine, so it has no single owner" | low |
| 4 | 3 | "↑ 7.1 Mbps Up 2 h 13 min": "Up" (uptime) next to the upload arrow reads as upload | low |
| 5 | 3 | No live per-connection graph or loss on the Slate; Fedora graphs are labelled "Path 0 / Path 1 · Min 31 Ms", not by link; Flint shows only "2 connections bonded" | low |
| 6 | 4 | Limit is monthly only, fixed reset "on the 1st of each month", and counts only tailbond traffic | low |
| 7 | 4 | "Used only if your other connections stop working": the trigger isn't defined (slow or lossy Wi-Fi?) | low |
| 8 | 5 | Why the short name doesn't work isn't explained; "The list appears once tailbond is connected" hides the names when tailbond is off | low |
| 9 | 6 | Turning off drops Backup only/limits: "over whichever connection GL's own settings pick (INTERNET page)". The phone may carry all traffic; no link, and it doesn't say which connection | medium |
| 10 | 6 | The main switch on the GL page has no label; "turned off or stopped after an error" (10) blurs the off (09) and not-running states | low |
| 11 | 7 | Fedora page contradicts itself: "anyone on your Tailscale network who can reach homebox" can approve vs "alex@example.com… others… can't approve" | medium |
| 12 | 8 | 16b is one long paragraph; "↓ 12 Kbps ↑ 8 Kbps" looks broken; no "I've signed in, check now" button as in the Ethernet flow | low |
| 13 | 10 | Fedora header "1.1 MBPS DOWN / 12.0 MBPS UP" vs device row "↓ 12.0 Mbps ↑ 1.1 Mbps": directions swapped on the same page | medium |
| 14 | 10 | Fedora health points to "the guide fedora-server.md"; both items repeat the same UDP 41641 fix | low |

Totals: 0 high / 3 medium / 11 low
