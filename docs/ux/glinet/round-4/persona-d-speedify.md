# Round 4, Persona D: long-time Speedify-on-GL.iNet user

Background: I have run Speedify on GL routers for years. It sits under the GL admin VPN menu and uses the GL login. It has Speed / Redundant / Streaming modes, per-connection priority (Always / Secondary / Backup / Never), monthly data caps, live per-connection graphs and the "Using 2 connections" line. On a hotel captive portal it keeps the VPN up on my other links and notifies me. Tapping the notification opens the hotel's page, and once I sign in the Wi-Fi joins the bond.

Overall: tailbond on GL feels very close to Speedify on GL. It lives at **VPN → tailbond** (02, sidebar under "WireGuard Server"), uses the GL login, and the main page maps almost one-to-one: "Fastest" / "Smooth calls and video" / "Most reliable" match Speed / Streaming / Redundant, and "Always" / "Backup only" / "Off" match Speedify's priorities without "Secondary". Monthly caps are there too ("Data used: 1.2 GB of 5 GB this month", "Change limit"). Most of my friction was on the home-server side, where the setup-health card contradicts itself.

---

## Task 1: Install on the Slate 7 without SSH

**Clicks:** On the home server I click "Download the router package" (11/12/13 "Add a GL.iNet travel router", or 15 Devices tab). On the Slate 7 I go to SYSTEM → Advanced Settings → "Go To LuCI" (01a). The page says "LuCI is OpenWrt's own web interface. Use the same admin password to log in." In LuCI I go to System → Software → "Upload Package…", pick `tailbond_0.1.0-r42_aarch64_cortex-a53_neon-vfpv4.ipk (6.1 MB)`, and click "Install" at "Really install the uploaded package?" (01b). I leave "Allow overwriting conflicting package files" unticked, as the home page told me ("leave the other options as they are").

**Friction:** The steps are clear and match the instructions on the home-server page word for word. Speedify on GL comes from GL's own plug-in list, so a trip into LuCI and a file upload is more work. Still, the steps are spelled out and "The router's internet keeps working" is reassuring. I can't check whether the long `aarch64_cortex-a53_neon-vfpv4` name is the right one for my model, so I just trust the button. Nothing tells me to refresh the GL panel to see the new menu entry, but 02 shows it appearing under VPN.

## Task 2: Pair with the home server

**Clicks:** VPN → tailbond (02) shows "Looking for your home server on your tailnet…". A few seconds later (03) it shows "Found your home server: homebox" with "homebox.tail1234.ts.net · 100.101.1.1", and I click "Ask to connect". 04 then shows "Waiting for approval on homebox…" and "On homebox, tap Approve next to slate7 … you can close this page."

**Friction:** None to speak of. Better than Speedify's account login. The fallback screens are clear: 03b "Turn on Tailscale first" with "Open Tailscale settings", and 03c "No home server found on your tailnet … Checked 4 online machines".

## Task 3: Hotel Wi-Fi (Repeater) + phone tethering, confirm both are used

**Clicks:** None. 05 header: "Connected to homebox · Using Repeater (Wi-Fi) + Tethering (phone)". Both cards show a green "In use" with per-link speed and latency ("↓ 38 Mbps ↑ 4.2 Mbps · 24 ms", "↓ 21 Mbps ↑ 2.9 Mbps · 48 ms"). The phone layout (05-phone) reads well.

**Friction:** This is the "Using 2 connections" I know, written as names. Speedify gives me live per-connection graphs. Here I only get the numbers, with no graph on the GL page (low). When the phone link fails (06), the header says "Tethering (phone) not working right now", and the card says "It's connected, but can't reach homebox. tailbond keeps trying; the other connections carry your traffic." That is clearer than Speedify.

## Task 4: Phone as backup only

**Clicks:** I click "Backup only" on the Tethering card (07). The badge changes to "Backup · saving data" with the text "Used only if your other connections stop working. Until then it isn't used at all, so it uses no data." The header says "Tethering (phone) on standby". For the cap, I click "Change limit" (07b), enter "5" "GB a month", and click "Save".

**Friction:**
- Speedify has four levels (Always / Secondary / Backup / Never). Here there are three, with no "Secondary" (use it only when the main link is slow). I don't miss it much (low).
- 07b says "until the count restarts on the 1st of each month". My phone plan resets on the 14th, and there's no way to pick a reset day. Speedify has the same limit, so this is low.
- The hint under the modes ("Saving your phone's data? Set Tethering or Cellular to 'Backup only'") is a nice touch.

## Task 5: Reach the NAS by MagicDNS name

**Clicks:** I scroll to "Tailnet machines" (05), find "nas / nas.tail1234.ts.net / 100.101.2.2 / linux · Online · direct", and click "Copy name". The Home server card adds: "Tailnet names: Work on every device connected to this router while tailbond is on. Use the full name, like nas.tail1234.ts.net."

**Friction:** None. Speedify has no equivalent. The bold short name "nas" above the full name might make some people try just `nas`, but the text says to use the full name (low).

## Task 6: Turn tailbond off

**Clicks:** I flip the toggle at the top right of the status card. 09 then says "tailbond is off: Devices on this router use the internet directly, without your home server. Turn on to bond your connections again." "Unpair" is separate and asks for confirmation (09b: "Unpair from homebox? … it will need to approve").

**Friction:** None. The off/unpair split is clearer than in Speedify. I couldn't check whether tailbond also shows on GL's "VPN Dashboard" the way Speedify's status does; no screen shows it (low). On 05 there's an "Ask to connect again" button while I'm already "Connected", and I don't know why I'd press it (low).

## Task 7: Approve the travel router at home

**Clicks:**
- **Flint 3 (11):** The top card "Waiting for your approval · 1 new" shows "slate7 wants to connect · Travel router on your tailnet · signed in as alex@example.com · GL-BE3600", and I click "Approve". 12 then lists "slate7 Connected just now · 2 connections bonded".
- **Fedora (14):** Same flow: "slate7 wants to connect", then "Approve". On a phone (14-phone) the Approve button is right at the top.

**Friction:** On Fedora, to require a password I'd have to "set "ui_password" in /etc/tailbond/tailbond.json". The Flint does this with a "Set" link. Not needed for this task (low).

## Task 8: Hotel Wi-Fi sign-in page (GL's repeater detected it)

**Clicks:** 16 shows a banner: "Repeater needs you to sign in: GL's Repeater page handles this sign-in. Open it, click "Login Mode for Public Hotspots" (or "Go to Login") and accept the hotel's terms. tailbond steps aside while you sign in…". I click "Open GL's Repeater page" and start Login Mode. Back on the tailbond page (16b), the Repeater card shows "Signing in…" and "Devices here reach this network directly until you've signed in." Tethering keeps the tunnel up meanwhile ("Using Tethering (phone)").

**Friction:**
- Compared with Speedify, I go to GL's page instead of having the hotel page opened for me. The banner explains that well, so this is fine.
- 16b gives no "I've signed in, check now" button and no time limit. The Ethernet case (17b) has both, so the two sign-in flows behave differently. The banner title still reads "Repeater needs you to sign in" while it's already signing in (low).
- No screenshot shows the "back in the bond" state after signing in. I have to trust "adds Repeater back to bonding by itself afterwards" (low).
- My phone here is on "Always" (16), so hotel sign-in time burns phone data. Nothing on this screen points that out. Speedify doesn't either (low).

## Task 9: Hotel Ethernet with a sign-in page

**Clicks:** 17 shows a banner: "Ethernet needs you to sign in … Opening the sign-in page lets devices here reach Ethernet directly (without your home server) for up to 10 minutes". I click "Open sign-in page". The hotel page opens in a new tab, and 17b shows "Signing in to Ethernet…" with "Open the sign-in page again", "I've signed in, check now" and "Stop".

**Friction:**
- 17b says "Until 04:06, devices here use Ethernet directly". Is 04:06 a clock time (4:06 AM?) or a countdown (4 min 6 s left)? Since the earlier screen said "up to 10 minutes" I guess it's a countdown, but it reads like a time of day (low).
- The header still says "Using Repeater (Wi-Fi) · Ethernet needs sign-in" while the banner says devices use Ethernet directly. I can live with it (low).
- Otherwise this is the Speedify experience, and arguably better: I can click "I've signed in, check now" instead of waiting.

## Task 10: Home side set up for best speed

**Clicks:** On the Flint 3 I scroll to "Setup health".
- In 12 it shows "All good · Everything is set up for best speed" with ticks for direct Tailscale connections, "Network is friendly to direct connections (port mapping: UPnP)", "Route 10.77.0.0/24 approved", "Public internet address" and "UDP 443 reached from the internet". Done.
- In 18 it shows "2 to improve": "Tailscale: 2 machines direct, 2 via relay (slower)" and "Hard NAT on this network…", each with "Forward UDP 41641 … or turn on UPnP/NAT-PMP" and a "How to fix" link. Actionable.

**Friction:**
- **13 contradicts itself.** The header says "Home server is on · 2 setup steps left", with orange Step 1 / Step 2 boxes. But Setup health shows a green "✓ Everything is set up for best speed" next to an orange "0 to improve" badge and two "?" items ("Behind your home router", "UDP 443: not confirmed from outside yet"). Is it fine or not? I'd probably believe the green line and skip Step 2 (the internet address). Without it, travel routers "use 192.168.8.1:443, which only works at home". (medium)
- **18b says two opposite things.** "Internet and Tailscale" shows a green "Setup done: Travel routers and phones can connect from anywhere". Further down, Setup health says "Your internet line uses carrier-grade NAT … travel routers can't reach UDP 443 here". If travel routers can't reach it, it isn't "Setup done", and the green box made me think I was fine. (medium)
- In 18, "Via relay: slate7, work-laptop" doesn't match the Tailnet machines list on the same page. That list shows pixel-8 as "via relay (slower)" and work-laptop offline ("Last seen 1 days ago"). Which machine is actually slow? The fix is the same either way (low). "Check with: tailscale ping slate7" is a command line, and there's no terminal in this panel (low).
- **Fedora:** The manifest says the Devices tab (15) includes setup health, but I can't find any Setup health card on 15. It shows "Add a GL.iNet travel router", "Add a device with a code", "Invite someone's phone", then pixel-8 and ipad. On the Fedora box I couldn't do this task from what's shown; maybe it's under the "Tailscale" tab, but I'd be hunting. (medium)
- Fedora 14 header: "1.1 MBPS DOWN / 12.0 MBPS UP". The Devices tab shows pixel-8 with "↓ 12.0 Mbps ↑ 1.1 Mbps". The direction flips depending on the view (server vs device), which threw me for a second (low). Path cards say "Path 0 / Path 1 · Pixel-8 · Min 31 Ms" instead of naming the link (Wi-Fi / cellular) the way Speedify's per-connection graphs do (low).

---

## Findings

| # | task | finding | severity |
|---|---|---|---|
| 1 | 1 | Install takes a LuCI detour plus a file upload (Speedify installs from GL's plug-in list); steps are clear but longer | low |
| 2 | 3 | No live per-connection graphs on the GL page, only per-link numbers (Speedify has graphs) | low |
| 3 | 4 | Only Always / Backup only / Off; no "Secondary" priority like Speedify | low |
| 4 | 4 | Data limit always resets "on the 1st of each month"; can't match my phone plan's reset day | low |
| 5 | 5 | Bold short name "nas" above the full name may tempt users to type just `nas` (text does say use the full name) | low |
| 6 | 6 | "Ask to connect again" shown while already "Connected"; unclear why I'd press it | low |
| 7 | 6 | No screen shows whether tailbond status appears on GL's "VPN Dashboard" like other GL VPNs | low |
| 8 | 7 | Fedora approval password needs editing "/etc/tailbond/tailbond.json" (Flint has a "Set" link) | low |
| 9 | 8 | Repeater "Signing in…" has no "I've signed in, check now" or timer (Ethernet flow does); banner title still says "needs you to sign in" | low |
| 10 | 8 | No screen shows the "Repeater back in the bond" state after signing in | low |
| 11 | 9 | "Until 04:06" reads like a clock time; probably a countdown | low |
| 12 | 9 | Header "Using Repeater (Wi-Fi)" while the banner says devices use Ethernet directly | low |
| 13 | 10 | 13: Setup health shows green "Everything is set up for best speed" next to an orange "0 to improve" badge and "?" items while the header says "2 setup steps left"; might make me skip Step 2 | medium |
| 14 | 10 | 18b: green "Setup done … can connect from anywhere" contradicts "carrier-grade NAT … travel routers can't reach UDP 443 here" | medium |
| 15 | 10 | Fedora Devices tab (15) shows no Setup health card despite the manifest; can't check best-speed setup on Fedora from what's shown | medium |
| 16 | 10 | 18: "Via relay: slate7, work-laptop" doesn't match the Tailnet machines list (pixel-8 relayed, work-laptop offline) | low |
| 17 | 10 | "Check with: tailscale ping slate7" is a CLI hint inside a GUI panel | low |
| 18 | 10 | Fedora: up/down figures flip between header (1.1 down / 12.0 up) and device row (↓12.0 ↑1.1); path cards say "Path 0/1" not the link name | low |
