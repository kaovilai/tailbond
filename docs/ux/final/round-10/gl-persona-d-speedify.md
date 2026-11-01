# Round 10 — GL persona D (Speedify user)

**Who I am:** I've run Speedify for years, both as the app and through GL.iNet's VPN menu on a travel router. I know bonding modes (Speed / Streaming / Redundant), per-connection priority (Always / Secondary / Backup / Never), monthly and daily data caps, the live per-connection graphs, and what Speedify does when a link drops. I compare everything below with that.

**Materials used:** only `task-script.md`, `screens.md` and the PNGs in `out/ux/final/round-10/packet-gl/`. I looked at desktop screens for every task and phone screens for 05, 07b, 11 and 14.

---

## Task 1: Install tailbond on the Slate 7 without SSH

**Where I click:**
- On the home side I click **"Download the router package"**. It's on the Fedora page (15, Devices tab, "Add a GL.iNet travel router (recommended)") or the Flint 3 panel (11–13).
- On the Slate 7 I go to SYSTEM → Advanced Settings and click **"Go To LuCI"** (01a). The screen says "Use the same admin password to log in."
- In LuCI I go to System → Software → **"Upload Package…"**. The file is `tailbond_0.1.0-r42_aarch64_cortex-a53_neon-vfpv4.ipk (6.1 MB)`, and I click **Install** (01b).
- Then VPN → tailbond appears in the GL menu (02).

**Comparison:** Speedify on GL was a menu entry I just turned on. Here I have to go through LuCI. Still, the steps match the screens click for click, and nothing needs SSH. That's fine.

**Snags:**
- **The Fedora steps say less than the Flint steps.** Fedora (15) says only "pick the file, Install." Flint (11) adds "(log in with the same password)" and "(leave the other options as they are)". LuCI's dialog (01b) has an unticked box, "Allow overwriting conflicting package files". Following the Fedora page, I'd wonder whether to tick it. **Low.**
- **The download doesn't say which routers it fits.** Flint says "It's the same tailbond this router runs", so it's the Flint 3's build. The Fedora page doesn't name a model at all. Nothing on either page says it fits a Slate 7 (GL-BE3600). Only the file name (`aarch64_cortex-a53`) hints at it, and that means nothing to most people. **Low.**

## Task 2: Pair the Slate 7 with the home server

**Where I click:**
- VPN → tailbond first shows "Looking for your home server on your tailnet…" (02).
- A few seconds later it shows "Found your home server: homebox · homebox.tail1234.ts.net · 100.101.1.1" with **"Ask to connect"** (03).
- After clicking, it shows "Waiting for approval on homebox… On homebox, tap Approve next to slate7" (04).
- The edge cases are covered. With Tailscale off I get "Turn on Tailscale first" plus "Open Tailscale settings" (03b). With no server found I get "No home server found on your tailnet · Checked 4 online machines" (03c).

**Comparison:** Simpler than Speedify's account login. No problems.

## Task 3: At the hotel, use repeater + USB tethering and confirm both are used

**Where I look:** Screen 05.
- The top card says "Connected to homebox · Using Repeater (Wi-Fi) + Tethering (phone) · ↓ 59 Mbps ↑ 7.1 Mbps".
- Both cards show **"In use"** with their own speeds: Repeater "↓ 38 Mbps ↑ 4.2 Mbps · 24 ms", Tethering "↓ 21 Mbps ↑ 2.9 Mbps · 48 ms". The two add up to the total, which is how I'd confirm bonding in Speedify.
- The **Always / Backup only / Off** buttons map neatly to Speedify's Always / Backup / Never.
- "Speed or reliability" (Fastest / Smooth calls and video / Most reliable) maps cleanly to Speed / Streaming / Redundant.

**When a link drops (06):**
- The Tethering card turns red: "Not working right now · It's connected, but can't reach homebox. tailbond keeps trying; the other connections carry your traffic."
- The header changes to "Tethering (phone) not working right now".
- That's as clear as Speedify.

**Snags:**
- **No live graphs and no packet loss per connection on the travel router.** Speedify shows a live graph and loss for each link, and I watch them to spot a flaky link. The Slate 7 page shows only current Mbps and latency as text. Per-path graphs and "LOSS 0.0%" do exist, but only on the Fedora home page (14, Connections tab). There the paths are called "Path 0" / "Path 1" with "Pixel-8 · Min 31 Ms", which doesn't tell me which one is hotel Wi-Fi and which is the phone. The task can still be done. **Low.**
- **"Up" looks like "upload".** The header reads "↑ 7.1 Mbps   Up 2 h 13 min". Right next to the upload arrow, "Up" reads like another upload figure. "Connected for" or "Uptime" would be clearer. **Low.**

## Task 4: Phone as backup only

**Where I click:** On the Tethering card, under "Use it", I click **"Backup only"** (07).
- The pill changes to "Backup · saving data".
- The card explains: "Used only if your other connections stop working. Until then tailbond sends nothing over it (GL itself may check the connection now and then, a few KB)."
- The header changes to "Tethering (phone) on standby".
- There's also a pointer under the modes: "Saving your phone's data? Set Tethering or Cellular to 'Backup only'".

**Data limit:** I click **"Change limit"** (07b). It shows "5 GB a month" with Save and Cancel, and explains: "Counts everything tailbond sends and receives over it. At the limit, tailbond stops using this connection until the count restarts on the 1st of each month. 0 = no limit." That's clear, and it looks the same on the phone screen.

**Snags:**
- **Turning tailbond off can use the phone's data, and the screen that suggests turning it off doesn't warn me.**
  - When the home server is unreachable (08), the red banner says "Meanwhile, devices on this router have no internet … To use this internet directly for now, turn tailbond off." Its button is **"Turn tailbond off"**.
  - Nothing there mentions that this hands control back to GL. GL may then send traffic over my phone, ignoring the "Backup only" I just set and the 5 GB cap.
  - In the same screen the Tethering card says "Ready · This choice applies when tailbond is on. While it's off, GL's own settings decide". But tailbond *is* on there, so I can't tell whether my Backup-only setting still holds.
  - I only learn the truth after clicking (09): "While tailbond is off, GL may use Tethering (your phone's data) · 'Backup only' and data limits only apply while tailbond is on."
  - For someone protecting a capped phone plan, that warning belongs next to the "Turn tailbond off" button in 08.
  - **Medium.**
- **The limit is monthly only, and always resets on the 1st.** Speedify lets me set a daily limit and choose the day the month resets to match my carrier's billing cycle. Here the screen says the count "restarts on the 1st of each month", so it won't line up with a plan that renews on, say, the 15th. **Low.**

## Task 5: Reach the NAS by its MagicDNS name

**Where I look:** On 05, the "Tailnet machines" list shows "nas · nas.tail1234.ts.net · 100.101.2.2 · linux · Online · direct" with **"Copy name"**. "Home server → Tailnet names" says: "Work on every device connected to this router while tailbond is on. Use the full name, like nas.tail1234.ts.net."

The name is **nas.tail1234.ts.net**. Easy to find.

**Snags:**
- **The short name won't work.** Tailscale users habitually type just `nas`. The page says to use the full name and repeats it in three places, so I'd follow it, but it's a difference from the plain Tailscale experience. **Low.**

## Task 6: Turn tailbond off

**Where I click:** The blue switch at the top right of the status card (05). The result is 09: "tailbond is off · Devices on this router use the internet directly, without your home server…". The orange phone-data warning appears under it (good; see Task 4).

**Snags:**
- **The switch has no label, and "Unpair" is easy to reach.** On the router page the switch has no text next to it. The Fedora page labels its switch "Connected". The other thing a newcomer might try is **"Unpair"** in the Home server card. The confirm dialog (09b) does explain the difference: "This router stops using tailbond and forgets the home server… it will need to approve". So no harm done. **Low.**

## Task 7: Approve the travel router on the home side

**Where I click:**
- On the Flint 3 (11), "Waiting for your approval · 1 new" shows "slate7 wants to connect · Travel router on your tailnet · signed in as alex@example.com · GL-BE3600 · linux · 100.101.9.9", with **Approve**.
- After approving (12), slate7 appears under Devices: "Connected just now · 2 connections bonded · downloading 48 Mbps · uploading 6.1 Mbps".
- On Fedora (14), the same card is at the top of the page with a big **Approve**. On the phone the button is full width and easy to tap.

**Snags:**
- **The "Who can approve" paragraph is hard to read.** On Fedora (14) it is a single block of roughly five lines: Tailscale account, any signed-in device, 127.0.0.1:8077, people you share with, access rules. I just want to click Approve, and I skimmed past it. It doesn't block anything. **Low.**

## Task 8: Hotel Wi-Fi sign-in page (repeater)

**Where I click:**
- On 16 the header says "Repeater (Wi-Fi) needs sign-in". The Repeater card shows **"Needs sign-in"**, and the blue banner says "GL's Repeater page handles this sign-in… click 'Login Mode for Public Hotspots'".
- I click **"Open GL's Repeater page"** and start Login Mode.
- Back on tailbond (16b), the header says "Router connected to homebox · devices signing in".
- The banner says "Now open any website on your laptop or phone: the hotel's sign-in page appears there… Login Mode ends by itself once you've signed in… tailbond then adds Repeater back to bonding."
- Tethering keeps the tunnel up at "↓ 12 Kbps ↑ 8 Kbps".

**Comparison:** It's good that I'm told my devices' traffic goes out the hotel Wi-Fi directly during sign-in. Speedify never told me that.

**Snags:**
- **"Open any website on your laptop or phone" is ambiguous here.** In this scenario my phone is the USB tether. If I open a website on *that* phone, it uses its own cellular data and never sees the hotel's sign-in page. The text means "a phone on this router's Wi-Fi", but it doesn't say so. **Low.**
- I can't see a screen that confirms the Repeater is back "In use" after signing in. I'm trusting the banner's promise. That's not a finding, just noting what I can't check.

## Task 9: Sign-in page on the hotel's Ethernet cable

**Where I click:**
- On 17 the Ethernet card shows "Needs sign-in", and the banner says "Ethernet needs you to sign in… Until then tailbond keeps Ethernet out of bonding and your other connections carry your traffic."
- I click **"Open sign-in page"**.
- On 17b the banner says "Signing in to Ethernet… For up to 9 more minutes (until 05:17 on this router's clock)". It offers **"Open the sign-in page again"**, **"I've signed in, check now"** and **"Stop sign-in mode"**.
- Tethering stays "Backup · saving data" throughout, which is what I want.

**Comparison:** Better than Speedify, which just showed the link as failing.

**Snags:**
- **Wi-Fi and Ethernet sign-ins work differently, and the reason is in small print.** Wi-Fi sign-in goes through GL's Repeater page and Login Mode (16). Ethernet sign-in is a tailbond button (17). The only explanation is the small line above the main text on 17: "GL's Login Mode covers the repeater Wi-Fi only; tailbond handles sign-in pages on other connections like this one." It's readable, but it's the least prominent line in the banner. **Low.**

## Task 10: Home side set up for best speed

**Where I look:** On the Flint 3, the "Setup health" card.
- **18c:** "All good · Everything is set up for best speed". Five ticks: "Tailscale: 3 machines direct", "Network is friendly to direct connections", "Route 10.77.0.0/24 approved", "Public internet address 203.0.113.7", "UDP 443 reached from the internet".
- **18:** "2 to improve". The fix is concrete: "On this router: NETWORK → Firewall → Open Ports on Router, add UDP 41641." It also says "bonding itself doesn't go through Tailscale", which reassures me that my bonded speed isn't relayed.
- **18b:** CGNAT is a clear red "Step 1: Your internet line uses carrier-grade NAT… Ask your provider for a public IPv4 address".
- **13:** The route and Dynamic DNS steps are numbered.
- **Fedora (15):** The same checks appear under Devices, with "forward UDP port 41641 to this machine (192.168.1.2)".

**Snags:**
- **Green and amber at the same time.** On 18, the green "Setup done · Travel routers and phones can connect from anywhere" box sits directly above an amber "2 to improve" card. It's correct (it works, just slower), but at a glance it's two different verdicts. **Low.**
- **The CGNAT address looks like a Tailscale address.** On 18b, "This machine's internet address 100.72.3.4" is a 100.x address. The Tailnet list further down is all 100.101.x addresses. Someone used to Tailscale could think "that's my Tailscale IP, not my internet address" and doubt the warning. **Low.**
- **Odd wording on the Fedora stats.** The Fedora data stat reads "7.5 GB to · 858 MB from devices" (14/15), and the path cards say "Min 31 Ms". It takes a second read. **Low.**

---

## Findings

| # | Task | Finding | Severity |
|---|---|---|---|
| 1 | 4 / 6 | On 08 the banner and its "Turn tailbond off" button say nothing about the phone's data. Only after turning it off does 09 say "'Backup only' and data limits only apply while tailbond is on". On 08 the Tethering card also says "While it's off, GL's own settings decide" even though tailbond is on, so I can't tell whether Backup-only still holds. | medium |
| 2 | 1 | Fedora's step 2 says only "pick the file, Install." It leaves out Flint's "(leave the other options as they are)", so I'd wonder about LuCI's "Allow overwriting conflicting package files" box. | low |
| 3 | 1 | "Download the router package" never says which GL models it fits (Slate 7 / GL-BE3600). Only the `.ipk` file name hints at it. | low |
| 4 | 3 | No live graph and no loss % per connection on the travel router. Per-path graphs exist only on the Fedora page, labelled "Path 0 / Path 1 · Pixel-8 · Min 31 Ms" rather than Repeater or Tethering. | low |
| 5 | 3 | "Up 2 h 13 min" right next to "↑ 7.1 Mbps" reads like an upload figure. | low |
| 6 | 4 | The limit is monthly only, and the count "restarts on the 1st of each month". No daily cap, and the reset day can't match my carrier's billing cycle. | low |
| 7 | 5 | Only full names work ("Use the full name, like nas.tail1234.ts.net"). Typing a short name like `nas`, a Tailscale habit, won't. | low |
| 8 | 6 | The router page's on/off switch has no label (Fedora's says "Connected"). The "Unpair" next to it could be mistaken for "off", though the 09b dialog prevents harm. | low |
| 9 | 7 | The "Who can approve" text on the approval card is one dense block of about five lines. | low |
| 10 | 8 | "Open any website on your laptop or phone": the phone is the USB tether in this scenario and wouldn't see the hotel's sign-in page. | low |
| 11 | 8 / 9 | Wi-Fi sign-in uses GL's Login Mode, while Ethernet uses tailbond's own "Open sign-in page". The explanation is the small line above the main text on 17. | low |
| 12 | 10 | On 18, the green "Setup done · … connect from anywhere" box sits right above the amber "2 to improve" card. | low |
| 13 | 10 | On 18b, the CGNAT address "100.72.3.4" looks like the Tailscale 100.x addresses listed below it. | low |
| 14 | 10 | The Fedora stats read "7.5 GB to · 858 MB from devices" and "Min 31 Ms". | low |

Totals: 0 high / 1 medium / 13 low
