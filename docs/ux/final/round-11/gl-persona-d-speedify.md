# Round 11 — GL persona D (Speedify user)

**Who I am:** I've run Speedify for years, including on a GL.iNet router through GL's VPN menu. I judge everything against it: bonding modes (Speed / Redundant / Streaming), per-connection priority (Always / Secondary / Backup / Never), data limits, live per-connection graphs, and what the UI does when a link drops.

**Materials:** task-script.md, screens.md and the PNGs in `packet-gl/` (desktop for every screen, phone for 07 and 17b as spot checks). I saw nothing else.

**Overall:** This is closer to Speedify than I expected. "Fastest / Smooth calls and video / Most reliable" maps directly onto Speed / Streaming / Redundant. "Always / Backup only / Off" per connection is the control I'm used to. Each connection shows its own speed, latency and data used. Failure states are worded better than Speedify's. I could finish all ten tasks. The one real stumble was on the home side: the page calls the Tailscale route "Optional" in one place and lists it as a best-speed check in another. The rest is polish, and most of it comes from me missing Speedify features (live graphs, billing-day limits).

---

## Task 1 — Install on the Slate 7 without SSH

**Clicks:**
- 15 (Fedora, Devices tab), "Add a GL.iNet travel router (recommended)" → **Download the router package**. Steps 2 and 3 are written right there: "SYSTEM → Advanced Settings → Go To LuCI, then System → Software → Upload Package…, pick the file, Install."
- 01a: SYSTEM → Advanced Settings → **Go To LuCI** ("LuCI is OpenWrt's own web interface. Use the same admin password to log in.").
- 01b: LuCI → System → Software → **Upload Package…** → dialog "Really install the uploaded package?" with `tailbond_0.1.0-r42_aarch64_cortex-a53_neon-vfpv4.ipk (6.1 MB)` → **Install**. The Flint version of the steps (11–13) adds "(leave the other options as they are). The router's internet keeps working.", so I leave "Allow overwriting conflicting package files" unticked.
- 02: VPN → **tailbond** appears in GL's own VPN menu, next to OpenVPN/WireGuard. That's where Speedify sits on GL, so it felt familiar.

**Stuck / misunderstood:** Nothing blocked me. The only doubt was the filename: `aarch64_cortex-a53_neon-vfpv4`. Speedify's GL download page names router models. Here, nothing on 15 or 11–13 says this one file is right for a Slate 7 (GL-BE3600). The Flint's "It's the same tailbond this router runs" is about the Flint, and the Fedora page doesn't say which routers the package fits. I installed it on trust. **Low.**

## Task 2 — Pair with the home server

**Clicks:**
- 02: "Looking for your home server on your tailnet…"
- 03: "Found your home server: homebox  homebox.tail1234.ts.net · 100.101.1.1" → **Ask to connect**.
- 04: "Waiting for approval on homebox… On homebox, tap Approve next to slate7. This router connects by itself once approved; you can close this page." It also says where to approve: "open http://homebox:8077 … At home with a Flint 3, its admin panel → VPN → tailbond works as well."

The error states are good. 03b says "Turn on Tailscale first" and has an **Open Tailscale settings** button. 03c says "No home server found on your tailnet — Checked 4 online machines…" and offers **Search again** plus a field to type the name.

**Stuck:** No. Speedify just logs in to an account. An approval step is extra, but the page explains it and I could close the page and go approve.

## Task 3 — Hotel Wi-Fi (repeater) + phone over USB, confirm both used

**Clicks:** 05.
- Header: "Connected to homebox — Using Repeater (Wi-Fi) + Tethering (phone)", "↓ 59 Mbps ↑ 7.1 Mbps Up 2 h 13 min".
- Repeater card: "In use", "↓ 38 Mbps ↑ 4.2 Mbps · 24 ms".
- Tethering card: "In use", "↓ 21 Mbps ↑ 2.9 Mbps · 48 ms".
- 38 + 21 = 59, so both links are really carrying traffic. "Speed or reliability" is on **Fastest**: "Adds all your connections together for the most speed." That's Speedify's Speed mode.

**Link drop (06):** Header reads "Using Repeater (Wi-Fi) · Tethering (phone) not working right now". The Tethering card turns red with "Not working right now" and "It's connected, but can't reach homebox. tailbond keeps trying; the other connections carry your traffic." That's clearer than Speedify's grey "Disconnected" connection tile.

**Where it falls short of Speedify:**
- On the Slate 7 there is no live per-connection graph. Each card shows a single current figure (e.g. "↓ 38 Mbps ↑ 4.2 Mbps · 24 ms"). In Speedify I watch the stacked per-adapter graph to see that bonding is really splitting traffic and spot a link that's flapping. **Low.**
- Live graphs do exist on the Fedora home page (14), but per *path*: "Path 0 · Pixel-8 · Min 31 Ms", "Path 1 · Pixel-8 · Min 58 Ms". Nothing says which one is Wi-Fi and which is cellular, and "Ms" in capitals reads like megaseconds. The Flint (12) shows only "2 connections bonded · downloading 48 Mbps", with no per-path figures. **Low.**

## Task 4 — Phone as backup only

**Clicks:** 05 → Tethering card → "Use it" → **Backup only**. Result on 07:
- Header: "Using Repeater (Wi-Fi) · Tethering (phone) on standby".
- Tethering badge: "Backup · saving data". Card text: "Used only if your other connections stop working. Until then tailbond sends nothing over it (GL itself may check the connection now and then, a few KB)."
- The Speed card hints at this too: "Saving your phone's data? Set Tethering or Cellular to 'Backup only' under Internet connections."

Same as Speedify's Backup priority, and the "a few KB" note is honest. 09 also warns that "Backup only" and limits don't apply while tailbond is off, which a Speedify user wouldn't think of. Good.

**Data limit (07b, "Change limit"):** "5 GB a month" field. "Counts everything tailbond sends and receives over it. At the limit, tailbond stops using this connection until the count restarts on the 1st of each month. 0 = no limit."
- My phone plan resets on the 15th. Speedify lets me pick the reset day (and a daily limit). Here the reset is fixed to "the 1st of each month", so the count won't match my carrier's. **Low.**
- If the phone is the *only* working link and the limit is hit, "tailbond stops using this connection" doesn't say whether my devices then go offline or GL falls back on its own. **Low.**

## Task 5 — Reach the NAS by its MagicDNS name

**Clicks:** 05 → "Tailnet machines" → `nas  nas.tail1234.ts.net  100.101.2.2  linux · Online · direct` → **Copy name**. The card says devices reach these "by their full name, e.g. nas.tail1234.ts.net … (not this router's own Tailscale)". The Home server card's "Tailnet names" row adds "Use the full name, like nas.tail1234.ts.net." The answer is **nas.tail1234.ts.net**.

**Stuck:** No. Speedify has nothing like this.

## Task 6 — Turn tailbond off

**Clicks:** the toggle at the top right of the header card (05) → 09: "tailbond is off — Devices on this router use the internet directly, without your home server, over whichever connection GL's own settings pick." Then an amber warning: "While tailbond is off, GL may use Tethering (your phone's data)", with a **GL's INTERNET page** button. I didn't confuse it with **Unpair**: 09b's confirm says "forgets the home server… it will need to approve", which makes the difference clear.

**Small confusion:** 10 ("tailbond isn't running on this router") says "It may have been turned off or stopped after an error." I had just "turned it off" with the toggle, and 09 still showed the full page. So does the toggle stop the service or not? The two meanings of "off" overlap. **Low.**

## Task 7 — Approve on the home side

**Flint (11):**
- "Waiting for your approval · 1 new": "slate7 wants to connect — Travel router on your tailnet · signed in as alex@example.com · GL-BE3600 · linux · 100.101.9.9 · asked 1 min ago" → **Approve**.
- 12: Devices shows "slate7 — Connected just now · 2 connections bonded · downloading 48 Mbps · uploading 6.1 Mbps". Done.

**Fedora (14):** Same card → **Approve**. Under it, in orange: "It's signed in to alex@example.com, an account that may approve here: whoever holds this router can also open this dashboard and approve devices. If it's for someone else, sign it in to their own Tailscale account first, then let it ask again." It's my own router, so I approved, but the orange made me pause, and the Flint's card (11) doesn't show this warning for the same router. The "Who can approve" paragraph above it is also long ("they allow it whatever they allow homebox, which may be more than its owner's own account gets"). **Low.**

## Task 8 — Hotel Wi-Fi with a sign-in page (GL's repeater detected it)

**Clicks:**
- 16: header "Using Tethering (phone) · Repeater (Wi-Fi) needs sign-in". Blue box "Repeater needs you to sign in — GL's Repeater page handles this sign-in. Open it and click 'Login Mode for Public Hotspots' (or 'Go to Login'). Then open any website…" → **Open GL's Repeater page**.
- 16b, after starting Login Mode: header "Router connected to homebox · devices signing in". Text says "Login Mode ends by itself once you've signed in… tailbond then adds Repeater back to bonding." Repeater badge "Signing in…", Tethering "Router's link only".

Speedify on GL just shows a captive-portal nag. Here it's clear what to do and what happens to my traffic meanwhile.

**Polish:**
- 16 says "While you sign in (until it's done, or 10 minutes)". 16b has no countdown and doesn't say what happens if 10 minutes pass. The Ethernet flow (17b) *does* show "up to 9 more minutes (until 05:25 on this router's clock)".
- The sentence "everything devices here do goes out the hotel Wi-Fi directly" (16 and 16b) reads like a typo.

**Low.**

## Task 9 — Sign-in page on the hotel Ethernet

**Clicks:**
- 17: header "Using Repeater (Wi-Fi) · Ethernet needs sign-in · Tethering (phone) on standby". Amber box "Ethernet needs you to sign in" → **Open sign-in page**.
- 17b: "Signing in to Ethernet… Sign in on the page that opened… For up to 9 more minutes (until 05:25 on this router's clock)…". Buttons: **Open the sign-in page again**, **I've signed in, check now**, **Stop sign-in mode**. "tailbond adds Ethernet back by itself once you're signed in."

**Polish:**
- On 17, the first line in the box is a small one about GL ("GL's Login Mode covers the repeater Wi-Fi only; tailbond handles sign-in pages on other connections like this one."). The main instruction comes after it, so I read the side note first. **Low.**
- On 17b the header still shows "↓ 36 Mbps ↑ 3.9 Mbps" even though Repeater says "Router's link only… (almost no data)" and devices go out Ethernet directly. On 16b the same state showed "↓ 12 Kbps ↑ 8 Kbps". As a Speedify user I read 36 Mbps as "bonded traffic is flowing" and wondered whether sign-in mode had really started. **Low.**

## Task 10 — Home side set up for best speed

**Flint:**
- 18c: "Setup health · All good — Everything is set up for best speed". Checks: "Tailscale: 3 machines direct", "Network is friendly to direct connections", "Route 10.77.0.0/24 approved", "Public internet address 203.0.113.7", "UDP 443 reached from the internet".
- 18: "2 to improve" — "Tailscale: 2 machines direct, 1 through a relay (slower)" and "This network makes direct Tailscale connections hard". The fix is specific: "NETWORK → Firewall → Open Ports on Router, add UDP 41641…".
- 18b: CGNAT — "Ask your provider for a public IPv4 address…", plus "it can't run through Tailscale's relays (they'd make it slower than a single link)".
- 13: "Step 1: tell travel routers where home is" points to APPLICATIONS → Dynamic DNS.

**Fedora:** 15 lists the same two items, with "forward UDP port 41641 to this machine (192.168.1.2)".

**Stuck: the Tailscale route.**
- On 13 it's labelled "**Optional**: let your Tailscale machines reach tailbond devices — Skip this unless you want your Tailscale machines… to open something on a device behind a travel router." The header counts only "1 setup step left", without the route.
- On 12 and 18c, "Route 10.77.0.0/24 approved" is a tick under "**Everything is set up for best speed**".

My task is literally "best speed", so I can't tell whether skipping the route costs speed. I'd go into the Tailscale admin console and approve it just in case, which is the extra effort the "Optional" label was meant to save me. **Medium.**

**Minor:** 18's fix starts "For whoever manages this router: NETWORK → Firewall…". I'm already logged in to this router's admin panel, so the hand-off wording reads oddly. **Low.**

---

## Findings

| # | Task | Finding | Severity |
|---|---|---|---|
| 1 | 10 | Tailscale route is "Optional… Skip this unless…" on 13, but "Route 10.77.0.0/24 approved" is listed under "Everything is set up for best speed" on 12/18c. I can't tell if skipping it costs speed, so I'd do the admin-console step anyway. | medium |
| 2 | 1 | Package `…aarch64_cortex-a53_neon-vfpv4.ipk` is never said to fit a Slate 7 (GL-BE3600); the Fedora page doesn't say which routers it's for. | low |
| 3 | 3 | No live per-connection graph on the Slate 7, only a current "↓ 38 Mbps ↑ 4.2 Mbps · 24 ms" per card (Speedify has per-adapter graphs). | low |
| 4 | 3 | Fedora graphs are labelled "Path 0 / Path 1 · Pixel-8 · Min 31 Ms" (not Wi-Fi/cellular; "Ms" capitalised); the Flint has no per-path view at all ("2 connections bonded"). | low |
| 5 | 4 | Data limit always resets "on the 1st of each month"; can't match the carrier's billing day (Speedify can). | low |
| 6 | 4 | "At the limit, tailbond stops using this connection" doesn't say what happens if that's the only working link. | low |
| 7 | 6 | 10 says "It may have been turned off…", which overlaps with the toggle-off state on 09 (page still running); two meanings of "off". | low |
| 8 | 7 | Fedora (14) shows an orange "whoever holds this router can also… approve devices" warning for my own router; the Flint (11) doesn't show it for the same request; approval text is dense. | low |
| 9 | 8 | "until it's done, or 10 minutes" (16) has no countdown on 16b and no word on what happens after; "everything devices here do goes out…" reads like a typo. | low |
| 10 | 9 | 17 banner leads with the small GL Login Mode side note before the main instruction. | low |
| 11 | 9 | 17b header still shows "↓ 36 Mbps ↑ 3.9 Mbps" during sign-in while links are "Router's link only… (almost no data)" (16b showed 12 Kbps). | low |
| 12 | 10 | "For whoever manages this router: NETWORK → Firewall…" (18) reads oddly to someone already in this router's admin panel. | low |

Totals: 0 high / 1 medium / 11 low
