# Round 12 — GL persona D (Speedify user)

**Who I am:** I've run Speedify on GL.iNet routers for years, through GL's VPN menu. I judge everything against it: bonding modes, the per-connection Always / Secondary / Backup / Never setting, data limits with a reset day, live graphs for each connection, and what happens when a link drops. Here I'm setting up a Slate 7 (GL-BE3600) with a Flint 3 or a Fedora box at home.

**Materials used:** task-script.md, screens.md, and every listed PNG (desktop; phone versions of 05 and 07b checked for layout).

---

## Task 1 — Install without SSH

**Where I click:**
- Fedora page (15, Devices tab): "Add a GL.iNet travel router (recommended)" → "Download the router package". On the Flint (11–13) it's the blue "Download the router package" button, followed by "It's the same tailbond this router runs."
- Slate 7: SYSTEM → Advanced Settings → "Go To LuCI" (01a; it says "Use the same admin password to log in"). Then LuCI → Software → "Upload Package…" → dialog "Really install the uploaded package?" showing `tailbond_0.1.0-r42_aarch64_cortex-a53_neon-vfpv4.ipk (6.1 MB)` → "Install" (01b).
- The steps on the home pages match the screens exactly, so I'd get through this.

**Where I hesitate:**
- The download never says which routers it fits. All I see is an architecture string (`aarch64_cortex-a53_neon-vfpv4`). The Flint's line "It's the same tailbond this router runs" makes me wonder whether a Flint 3 package is right for a Slate 7. Speedify's GL page just lists supported models. **Low.**
- Speedify comes from GL's own menus. Here I have to leave GL's UI for LuCI, a second admin interface. The instructions get me through, but it feels like a workaround. **Low.**

## Task 2 — Pair

**Where I click:** 02 shows "Looking for your home server on your tailnet…". 03 shows "Found your home server: homebox" → "Ask to connect". 04 shows "Waiting for approval on homebox… On homebox, tap **Approve** next to **slate7**. This router connects by itself once approved; you can close this page."

**Stuck?** No. The error states are clear too: 03b "Turn on Tailscale first" with "Open Tailscale settings", and 03c "No home server found on your tailnet … Checked 4 online machines". Easier than logging in to a Speedify account.

## Task 3 — Hotel Wi-Fi + phone tethering, confirm both used

**Where I look:** 05. The top card says "Connected to homebox / Using Repeater (Wi-Fi) + Tethering (phone)". Repeater and Tethering both show a green "In use" badge with live numbers ("↓ 38 Mbps ↑ 4.2 Mbps · 24 ms", "↓ 21 Mbps ↑ 2.9 Mbps · 48 ms"). Confirmed.

**Compared with Speedify:**
- There's no live graph per connection on the travel router, and no loss figure: just one set of current numbers per card. Speedify's per-adapter graphs are how I spot a flaky link over time. The Fedora page (14) does have small graphs per path, but they're labelled "Path 0 · Pixel-8 · Min 31 Ms" / "Path 1". They aren't named Wi-Fi or cellular, and "Ms" is capitalised oddly. **Low.**
- The modes "Fastest / Smooth calls and video / Most reliable" map clearly onto Speedify's Speed / Streaming / Redundant. Good.

**When a link drops (06):** Tethering shows a red "Not working right now" badge and "It's connected, but can't reach homebox. tailbond keeps trying; the other connections carry your traffic." The header changes to "Tethering (phone) not working right now". This is what I want, and clearer than Speedify.

**When the home server drops (08):** "Meanwhile, devices on this router have no internet, so nothing goes around your home server. To use this internet directly for now, turn tailbond off". Speedify just moves me to another server. Here, if the home line goes down while I'm at a hotel, everyone behind the Slate is offline until someone opens the admin panel and clicks "Turn tailbond off". I see no setting to fall back to direct internet automatically. The screen is honest and has a button, but family members can't fix this themselves. **Medium.**

## Task 4 — Phone as backup only

**Where I click:** On the Tethering card, under "Use it", I click "Backup only". In 07 the badge reads "Backup · saving data" and the text says "Used only if your other connections stop working. Until then tailbond sends nothing over it (GL itself may check the connection now and then, a few KB)." The header says "Tethering (phone) on standby". The tip under the modes ("Saving your phone's data? Set Tethering or Cellular to 'Backup only'") pointed me straight there. Done.

**Compared with Speedify:**
- There are only three choices, "Always / Backup only / Off". Speedify also has Secondary (used when the main links are slow, not only when they're dead). "Stop working" isn't defined: if the hotel Wi-Fi is up but crawling, does the phone kick in? **Low.**
- Data limit (07b, "Change limit"): a box with "GB a month", and "the count restarts on the 1st of each month. 0 = no limit." I can't set the day my carrier's billing cycle resets, there's no daily limit, and there's no MB option. My plan resets on the 18th, so this cap won't line up with the real allowance. I'd have to keep adjusting it by hand or risk going over. Speedify lets me pick the reset day. **Medium.**

## Task 5 — Reach the NAS by MagicDNS name

**Where I look:** 05 → "Tailnet machines". It shows `nas`, `nas.tail1234.ts.net`, `100.101.2.2`, "linux · Online · direct" and a "Copy name" button. "Tailnet names: Work on every device connected to this router while tailbond is on. Use the full name, like nas.tail1234.ts.net." Found it.

**Minor:** As a Tailscale user I'd normally type just `nas`. The page warns me to use the full name, which is fine. But "reach these through the bond to homebox (not this router's own Tailscale)" made me stop: I turned on the Slate's Tailscale for this, so is it being used or not? **Low.**

## Task 6 — Turn tailbond off

**Where I click:** The blue switch in the top status card (05). Result in 09: "tailbond is off — Devices on this router use the internet directly…". There's also an orange warning: "While tailbond is off, GL may use Tethering (your phone's data) … 'Backup only' and data limits only apply while tailbond is on." Speedify on GL never told me that. Very good.

**Minor:** The switch has no label on the GL page. Next to "Unpair" there's an "Ask to connect again" button, shown even while connected, which reads like a "reconnect / off-on" control. The Unpair dialog (09b) clearly explains that Unpair is different from Off, so I wouldn't break anything. **Low.**

## Task 7 — Approve on the home side

**Where I click:**
- Flint (11): "Waiting for your approval · 1 new" → "slate7 wants to connect · Travel router on your tailnet · signed in as alex@example.com · GL-BE3600" → "Approve". 12 shows "slate7 Connected just now · 2 connections bonded".
- Fedora (14): the same card at the top with "Approve".

**Minor:** Each request card comes with a lot of small text. On Fedora: "…and any user or program on homebox itself (http://127.0.0.1:8077; set a password if homebox runs things you don't trust)… which may be more than its own owner's account gets." For my own router I'd skim past it and click Approve. It reads like a security warning even when there's nothing to worry about. **Low.**

## Task 8 — Hotel Wi-Fi sign-in page (GL repeater)

**Where I click:** In 16 the header reads "Using Tethering (phone) · Repeater (Wi-Fi) needs sign-in". A blue card, "Repeater needs you to sign in", tells me to click "Login Mode for Public Hotspots" (or "Go to Login") on GL's Repeater page, then open any website → "Open GL's Repeater page". In 16b: "Router connected to homebox · devices signing in", with "Login Mode ends by itself once you've signed in (or click 'Quit Login Mode' on GL's Repeater page); tailbond then adds Repeater back to bonding." Tethering shows "Router's link only". I can finish this, and it's better than Speedify, which mostly leaves captive portals to me.

**Minor:** This flow doesn't match the Ethernet one (17b). Here there's no "I've signed in, check now" button and no "until 05:34" deadline, only "(until it's done, or 10 minutes)". The wording "everything devices here do goes out the hotel Wi-Fi directly" is clumsy. **Low.**

## Task 9 — Sign-in page on the hotel Ethernet

**Where I click:** In 17: "Ethernet needs you to sign in · GL's Login Mode covers the repeater Wi-Fi only; tailbond handles sign-in pages on other connections like this one." → "Open sign-in page". In 17b: "Signing in to Ethernet…" with "Open the sign-in page again", "I've signed in, check now" and "Stop sign-in mode", and "tailbond adds Ethernet back by itself once you're signed in." Clear and complete.

**Minor:** While I sign in, the hotel Wi-Fi, which was working at 36 Mbps, drops to "Router's link only", and "all traffic from devices here goes out Ethernet directly". So my laptop is stuck behind the portal for up to 9 minutes. It's explained, but a Speedify user expects working links to keep carrying traffic. The 17 card also uses three different text sizes in one box. **Low.**

## Task 10 — Best speed at home

**Where I look:**
- Flint: "Setup health". 18c shows "All good / Everything is set up". 18 shows "2 to improve" with a concrete fix: "On this router: NETWORK → Firewall → Open Ports on Router, add UDP 41641". 18b shows "Your internet line uses carrier-grade NAT … Ask your provider for a public IPv4 address". 13 shows "Step 1: tell travel routers where home is … APPLICATIONS → Dynamic DNS".
- Fedora (15): "Setup health: 2 optional speed-ups (tailbond works fine without them)", with port-forward instructions and "How to fix".
- I could follow all of this.

**Minor:** The Port row's test tip, "To test it now: turn tailbond on on a phone using mobile data (Wi-Fi off)", assumes the Android app. I only have a travel router. Coming from Speedify, I also expected something about the home line's upload speed, since that caps what I can get at the hotel. Setup health only covers reachability and relays, so "All good" doesn't tell me whether I'll be fast. **Low.**

---

## Findings

| # | Task | Finding | Severity |
|---|---|---|---|
| 1 | 1 | Download doesn't say which GL models it fits; only `aarch64_cortex-a53_neon-vfpv4` in the file name, and "It's the same tailbond this router runs" (on the Flint) makes a Slate 7 owner unsure | low |
| 2 | 1 | Install means leaving GL's UI for LuCI ("Go To LuCI" → "Upload Package…"); Speedify comes from GL's own menus | low |
| 3 | 3 | No live per-connection graphs or loss on the travel router, only current "↓ 38 Mbps ↑ 4.2 Mbps · 24 ms"; Fedora's graphs are labelled "Path 0 · Pixel-8 · Min 31 Ms", not by link type | low |
| 4 | 3 | Home server down: "devices on this router have no internet … turn tailbond off"; no automatic fallback to direct internet, so family is offline until someone opens the panel | medium |
| 5 | 4 | Only "Always / Backup only / Off" (no Speedify-style Secondary); "stop working" isn't defined for a slow-but-up link | low |
| 6 | 4 | Data limit is only "GB a month", "restarts on the 1st of each month": no billing reset day, no daily limit, no MB, so it won't match a carrier plan | medium |
| 7 | 5 | "(not this router's own Tailscale)" confuses someone who just turned the Slate's Tailscale on; short name `nas` doesn't work (warned) | low |
| 8 | 6 | On/off switch has no label; "Ask to connect again" sits next to "Unpair" even while connected and reads like a reconnect control | low |
| 9 | 7 | Approval cards carry long security text ("any user or program on homebox itself (http://127.0.0.1:8077…") even for your own router | low |
| 10 | 8 | Repeater sign-in (16b) has no "I've signed in, check now" or deadline unlike Ethernet (17b); "everything devices here do goes out…" is clumsy | low |
| 11 | 9 | During Ethernet sign-in the working hotel Wi-Fi drops to "Router's link only"; devices sit behind the portal for up to ~10 min; three text sizes in the 17 card | low |
| 12 | 10 | Port test tip assumes the phone app ("turn tailbond on on a phone using mobile data"); health says nothing about home upload speed, which caps hotel speed | low |

Totals: 0 high / 2 medium / 10 low
