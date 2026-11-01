# Round 5, Persona D: long-time Speedify-on-GL.iNet user

Background: I've run Speedify on GL.iNet routers for years. It lives under VPN in
GL's left menu and uses the GL admin login. I'm used to Speed / Redundant / Streaming
modes, per-connection priority (Always / Secondary / Backup / Never), monthly caps,
live per-connection graphs, "Using 2 connections", and captive-portal handling where
Speedify keeps the VPN up on my other links, notifies me, and opens the hotel sign-in
page from the notification.

Inputs: task-script.md, screens.md, and the round-5 PNGs (all desktop screens, plus
05 on phone).

---

## Task 1: Install on the Slate 7 without SSH

**Where I click:** On the home side (13, 18c, or Fedora 15) I click "Download the router package". The instructions there read: "On the travel router's admin page: **SYSTEM → Advanced Settings → Go To LuCI** (log in with the same password), then **System → Software → Upload Package…**, pick the file and click **Install** (leave the other options as they are)." On 01a I click "Go To LuCI". The page tells me "Use the same admin password to log in." On 01b the dialog says "Really install the uploaded package?" and shows `tailbond_0.1.0-r42_aarch64_cortex-a53_neon-vfpv4.ipk (6.1 MB)`. I leave "Allow overwriting conflicting package files" unchecked and click **Install**.

**Compared with Speedify on GL:** This takes more steps than Speedify. You leave GL's UI for LuCI, and LuCI looks completely different (blue OpenWrt bar). The written steps match what I see on screen, though, and I got through without guessing. After the install, tailbond shows up under VPN → tailbond (02), right where Speedify lives. I like that.

- The LuCI detour feels foreign next to GL's own plugin flow, but it's well guided. (low)
- The Fedora page's step 2 drops "(log in with the same password)" and "(leave the other options as they are)", which the Flint panel includes. (low)

## Task 2: Pair with the home server

**Where I click:** 02 shows "Looking for your home server on your tailnet…", and a few seconds later 03 shows "Found your home server: homebox · homebox.tail1234.ts.net · 100.101.1.1". I click **Ask to connect**. 04 says "Waiting for approval on homebox… On homebox, tap **Approve** next to **slate7**. This router connects by itself once approved; you can close this page."

**Compared with Speedify:** Speedify only needs an account login. Here the home side has to approve me, and 04 explains that clearly, including "open http://homebox:8077 on any device signed in to your Tailscale account (a phone works too)". The error states are clear too: 03b says "Turn on Tailscale first" with "Open Tailscale settings", and 03c says "Checked 4 online machines…". No problems.

- None worth flagging.

## Task 3: Hotel Wi-Fi + phone tethering, confirm both used

**Where I look:** 05. The header says "Connected to homebox · Using Repeater (Wi-Fi) + Tethering (phone)" with "↓ 59 Mbps ↑ 7.1 Mbps Up 2 h 13 min". The Repeater and Tethering cards both show a green "In use" badge with per-link numbers ("↓ 38 Mbps ↑ 4.2 Mbps · 24 ms", "↓ 21 Mbps ↑ 2.9 Mbps · 48 ms"). This is the same idea as Speedify's "Using 2 connections", and it's easy to read.

The modes map straight onto Speedify: "Fastest" = Speed, "Smooth calls and video" = Streaming, "Most reliable" = Redundant ("Uses the most data"). I recognised all three right away.

06 (phone in trouble) is clear: "Tethering (phone) not working right now" and "It's connected, but can't reach homebox. tailbond keeps trying; the other connections carry your traffic."

- There are no live per-connection graphs on the travel router, only numbers. Speedify on GL has graphs, and so does tailbond's own Fedora page (14, "Path 0 / Path 1" mini charts). So the router I actually look at while travelling is the one without graphs. (low)
- The phone layout (05-phone) drops the "Up 2 h 13 min" uptime that desktop shows. (low)

## Task 4: Phone as backup only

**Where I click:** On the Tethering card, **Use it → Backup only** (07). The badge becomes "Backup · saving data" and the text says "Used only if your other connections stop working. Until then it isn't used at all, so it uses no data." The header says "Tethering (phone) on standby". The hint under the modes points to the same place: "Saving your phone's data? Set Tethering or Cellular to "Backup only" under Internet connections."

The cap: "Change limit" (07b) opens "5 GB a month", with "At the limit, tailbond stops using this connection until the count restarts on the 1st of each month. 0 = no limit." That's the same as Speedify's monthly cap.

- There are only three priorities (Always / Backup only / Off). Speedify also has "Secondary" (used only when the primary links are too slow). I didn't need it for this task. (low)
- I can only reset on the 1st of the month. Speedify lets me pick the billing day, and my phone plan doesn't reset on the 1st. (low)

## Task 5: Reach the NAS by MagicDNS name

**Where I look:** On 05, the "Tailnet machines" list shows "nas · nas.tail1234.ts.net · 100.101.2.2 · linux · Online · direct", and I click **Copy name**. The Home server card says "Tailnet names: Work on every device connected to this router while tailbond is on. Use the full name, like nas.tail1234.ts.net." Speedify has nothing like this, and it's a nice extra. The answer: **nas.tail1234.ts.net**.

- It's clear that the full name is required. The short name `nas` won't work, which I'd normally try first. (low)

## Task 6: Turn tailbond off

**Where I click:** The big toggle at the top right of the status card. 09 says "tailbond is off. Devices on this router use the internet directly, without your home server. Turn on to bond your connections again." That's exactly Speedify's off switch. "Unpair" is a separate button, and its dialog (09b) explains "forgets the home server… it will need to approve", so I won't mix them up. 08 (server down) also gives a "Turn tailbond off" button and warns plainly: "Meanwhile, devices on this router have no internet". 10 ("tailbond isn't running") offers "Start tailbond".

- In 09, Tethering shows "Always" again even though I'd set it to Backup only in task 4. If that's real behaviour (my choice was lost when I turned it off), I'd lose my phone-data protection without noticing. It's probably just the test fixture. (low)

## Task 7: Approve on the home side

**Flint 3 (11):** At the top there's "Waiting for your approval · 1 new · slate7 wants to connect · Travel router on your tailnet · signed in as alex@example.com · GL-BE3600 · linux · 100.101.9.9 · asked just now". I click **Approve**. 12 shows "slate7 · Connected just now · 2 connections bonded". Clear.

**Fedora (14):** The same card sits above the dashboard, and I click **Approve**. Clear.

- None worth flagging.

## Task 8: Hotel Wi-Fi sign-in (GL repeater detected it)

**Where I click:** 16 has the banner "Repeater needs you to sign in. GL's Repeater page handles this sign-in. Open it, click "Login Mode for Public Hotspots" (or "Go to Login") and accept the hotel's terms." I click **Open GL's Repeater page**, and from there use GL's Login Mode (that page isn't among the screenshots, so I'm trusting the label). 16b then says "GL's Login Mode is on… It ends by itself once you've signed in… tailbond then adds Repeater back to bonding." The Repeater badge shows "Signing in…". The header stays "Connected to homebox · Using Tethering (phone)", so like Speedify the tunnel stays up on the phone while I sign in.

**Compared with Speedify:** There's no notification on screen. I have to open the admin panel to find out the Wi-Fi needs signing in. In 16 the phone is on "Always", but if I'd set it to "Backup only" (task 4), my phone would silently take over and spend data until I happened to look. Also, "(or "Go to Login")" makes me unsure which button label I'll actually find on GL's page.

- I have to notice the sign-in myself by opening the panel. There's no push or notification like Speedify, so phone data gets used in the meantime. (low)
- The banner hedges on the button name: "Login Mode for Public Hotspots" (or "Go to Login"). (low)

## Task 9: Ethernet sign-in

**Where I click:** 17 shows "Ethernet needs you to sign in… Until then tailbond keeps Ethernet out of bonding and your other connections carry your traffic." I click **Open sign-in page**. On 17b the hotel page opens and the banner says "Signing in to Ethernet… For up to 9 more minutes (until 04:13), devices here use Ethernet directly, not through your home server." Then I click **I've signed in, check now**. This is the closest thing here to Speedify's "open notification → portal → link joins".

- There are two different sign-in flows for the same problem. For Wi-Fi I'm sent into GL's Repeater page; for Ethernet tailbond does it itself, with a timer and "I've signed in, check now". I'd have to learn both. (low)
- In 17b the header still says "Connected to homebox · Using Repeater (Wi-Fi)", but the banner says "devices here use Ethernet directly, not through your home server". For a moment I'm not sure whether my laptop is in the tunnel or not. It's only the sign-in window, but it reads like a privacy leak. (low)

## Task 10: Home side best speed

**Flint 3:** The "Setup health" card makes this easy. 18c shows "All good · Everything is set up for best speed" plus five checkmarks. 18 shows "2 to improve" with "Tailscale: 2 machines direct, 1 through a relay (slower)… Usually fixed by forwarding UDP port 41641 to this machine, or by turning on UPnP on the router in front of it. How to fix". 18b turns CGNAT into a blocking "Step 1: Your internet line uses carrier-grade NAT… Ask your provider for a public IPv4 address". 13 lists "2 setup steps left" with "Open admin console" and the Dynamic DNS hint.

**Fedora:** The page opens on the **Connections** tab (14), which shows graphs and paths but no health status at all. The health summary ("Setup health: 2 to improve for best speed") is only on the **Devices** tab (15). For a "best speed" check I'd look under Connections or **Tailscale** first, not Devices. I'd only find it by clicking through every tab.

- On Fedora, setup health is hidden on the "Devices" tab. The default Connections tab gives no hint that anything needs improving. (**medium**)
- On the Flint, the fix text says "Forward UDP port 41641 to this machine, or turn on UPnP on the router in front of it". If the Flint is my main router, there is no "router in front of it", and I'm not sure what to do. (low)
- In 18b, CGNAT says "travel routers can't reach UDP 443 here", yet "pixel-8 · Connected 3 h 0 min · 2 connections bonded" sits right below. That looks contradictory. (low)
- Fedora's wording is jargon compared with GL's plain style: "Path 0 · Pixel-8 · Min 31 Ms", "wlb", "#0 active · 31 ms · 0.0%". Its big counters ("1.1 MBPS DOWN / 12.0 MBPS UP") are flipped from pixel-8's "↓ 12.0 Mbps ↑ 1.1 Mbps", because one is the server's view and the other is the device's. (low)

---

## Overall vs Speedify on GL

It feels at home. It's in the same VPN menu with the same admin login, the three modes map directly onto Speed / Streaming / Redundant, there are per-link Always / Backup only / Off controls and monthly caps, and the header line plays the role of "Using 2 connections". The main gaps against Speedify are no live graphs on the router itself, no sign-in notification, two different captive-portal flows, and a more manual install through LuCI. Tailnet machine names are something Speedify never had.

## Findings

| # | task | finding | severity |
|---|---|---|---|
| 1 | 1 | Install detours through LuCI ("Go To LuCI" → "Upload Package…"), which feels foreign next to GL's own UI, though the steps are well guided | low |
| 2 | 1 | Fedora page's install step 2 drops "(log in with the same password)" / "(leave the other options as they are)" that the Flint panel has | low |
| 3 | 3 | No live per-connection graphs on the travel router (Speedify on GL and the Fedora page both have them) | low |
| 4 | 3 | Phone layout drops "Up 2 h 13 min" uptime | low |
| 5 | 4 | Only Always / Backup only / Off; no Speedify-style "Secondary" | low |
| 6 | 4 | Data cap always resets "on the 1st of each month"; can't set a billing day | low |
| 7 | 5 | Short name `nas` won't work; "Use the full name" must be read | low |
| 8 | 6 | 09 shows Tethering back on "Always" after turn-off; if real, my backup-only choice was lost | low |
| 9 | 8 | No notification that the Wi-Fi needs sign-in; with phone on Backup only, phone data is used silently until I open the panel | low |
| 10 | 8 | Banner hedges on GL's button label: "Login Mode for Public Hotspots" (or "Go to Login") | low |
| 11 | 9 | Two different captive-portal flows (GL Login Mode for Wi-Fi, tailbond's own for Ethernet) | low |
| 12 | 9 | 17b header "Connected… Using Repeater" vs banner "devices here use Ethernet directly, not through your home server": unclear if my traffic is tunnelled | low |
| 13 | 10 | Fedora: setup health only on the "Devices" tab; default Connections tab gives no hint anything needs improving | medium |
| 14 | 10 | Flint fix text "turn on UPnP on the router in front of it" is confusing when the Flint is the main router | low |
| 15 | 10 | 18b: CGNAT says travel routers can't reach here, yet pixel-8 shows "Connected… 2 connections bonded" | low |
| 16 | 10 | Fedora jargon ("Min 31 Ms", "wlb", "#0 active") and down/up counters flipped vs the device row | low |
