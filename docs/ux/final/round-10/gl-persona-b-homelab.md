# Round 10 — GL persona B (homelab)

Persona: homelab power user. I run Tailscale subnet routers, OpenWrt and a Fedora server at home. I want to know what actually happens: routes, DNS, direct vs relayed (DERP), NAT, who can approve, and where traffic goes while I'm signing in to a captive portal. I'd rather not SSH into a travel router from a hotel.

Materials used: `task-script.md`, `screens.md` and the PNGs in `out/ux/final/round-10/packet-gl/` only. I looked at the desktop version of every screen and the phone versions of 05, 11, 14 and 17.

Overall: this round is in good shape for someone like me. The screens answer most of the "what really happens" questions without making me guess. Bonding is a separate UDP 443 tunnel, not traffic over Tailscale (05: "Connects to home.example.net over the internet (UDP port 443)"; 18b: "it can't run through Tailscale's relays"). LAN clients reach the tailnet through homebox, not through the Slate's own Tailscale. Behind homebox they show up as homebox, under homebox's ACLs. The router fails closed when home is unreachable (08). The pages also say exactly which traffic bypasses home during a portal sign-in. I found no blockers. My one real confusion is how screen 16 describes captive-portal traffic.

---

## Task 1 — Install on the Slate 7 without SSH

**Clicks:** On the home server I click "Download the router package" (15 "Add a GL.iNet travel router (recommended)", or 11–13 on the Flint). On the Slate I go to SYSTEM → Advanced Settings → "Go To LuCI" (01a: "LuCI is OpenWrt's own web interface. Use the same admin password to log in."). Then System → Software → "Upload Package…", which shows a dialog for `tailbond_0.1.0-r42_aarch64_cortex-a53_neon-vfpv4.ipk (6.1 MB)`, and I click Install (01b). Back in the GL admin, VPN → tailbond appears (02).

**What worked:** No SSH needed. The path on the server pages matches the LuCI screens step for step. The Flint copy adds "(leave the other options as they are). The router's internet keeps working.", which is the reassurance I want before installing a VPN-ish package from a hotel.

**Snags:**
- I only learn the package architecture from the filename in LuCI (`aarch64_cortex-a53_neon-vfpv4`). The download button on 11–15 doesn't say which models or architecture it's built for. I'd want "for GL-BE3600 / aarch64_cortex-a53" next to the button, so I know before the upload whether it'll fit a different GL model. **Low.**
- The Fedora page's step 2 (15) is shorter than the Flint's: "On the travel router's admin page: SYSTEM → Advanced Settings → Go To LuCI, then System → Software → Upload Package…, pick the file, Install." It leaves out "log in with the same password" and "leave the other options as they are". The LuCI dialog shows an unchecked "Allow overwriting conflicting package files" box, so someone who likes to poke at checkboxes will wonder about it. **Low.**

## Task 2 — Pair with the home server

**Clicks:** 02 "Looking for your home server on your tailnet…", then 03 "Found your home server: homebox / homebox.tail1234.ts.net · 100.101.1.1", then "Ask to connect". 04 shows "Waiting for approval on homebox… On homebox, tap Approve next to slate7… From anywhere: open http://homebox:8077 on any device signed in to your Tailscale account".

**What worked:** Discovery over the tailnet, with a manual-name fallback ("Not your home server? Type its name instead"). Fallbacks for Tailscale off (03b "Open Tailscale settings") and nothing found (03c "Checked 4 online machines…"). The approver card on the server shows "signed in as alex@example.com · GL-BE3600 · 100.101.9.9", so I can check the node identity before approving.

**Snag:**
- 02/03 say "Your home server … must be signed in to the same Tailscale account as this router." The server side (11, 14, 15) says "Others on your Tailscale network, and people you share homebox with, can ask to connect but can't approve." Those two statements contradict each other. As a tailnet admin with node sharing set up, I don't know whether a shared-in router can pair or not. **Low.**

## Task 3 — Hotel Wi-Fi + USB tethering, confirm both in use

**Clicks:** Nothing to do. 05 says "Connected to homebox / Using Repeater (Wi-Fi) + Tethering (phone)". The Repeater card shows "In use ↓38 Mbps ↑4.2 Mbps · 24 ms" and the Tethering card "In use ↓21 Mbps ↑2.9 Mbps · 48 ms". 06 shows the failure case clearly: "Not working right now … It's connected, but can't reach homebox. tailbond keeps trying".

**What worked:** I get per-uplink throughput and RTT, and the header sums them up. That's what I'd want.

**Snag (home side):**
- On the Fedora page (14) the connections are just "Path 0 / Pixel-8 · Min 31 Ms" and "Path 1 … Min 58 Ms". The paths aren't named by uplink (Wi-Fi vs cellular, or source address), and "Min 31 Ms" is an odd label: minimum RTT? From home I can't tell which of the slate's uplinks is which. **Low.**

## Task 4 — Phone as backup only

**Clicks:** Tethering card → "Backup only" (07). The status changes to "Backup · saving data": "Used only if your other connections stop working. Until then tailbond sends nothing over it (GL itself may check the connection now and then, a few KB)." "Change limit" (07b) gives "5 GB a month … Counts everything tailbond sends and receives over it. At the limit, tailbond stops using this connection until the count restarts on the 1st of each month. 0 = no limit."

**What worked:** It's clear, and it's honest about GL's own probes. The limit applies to tailbond's own traffic only, and the page says so.

**Snag:**
- "GL itself may check the connection now and then" doesn't tell me whether the router's own traffic stays off the phone while tailbond is on. I mean GL's built-in Tailscale (control plane, DERP), GL cloud, and GL's multi-WAN choice. 09 warns that GL uses tethering when tailbond is off, so I'd assume tailbond pins it while on, but the page never says so. **Low.**

## Task 5 — Reach the NAS by MagicDNS name

**Clicks:** 05 Tailnet machines: "nas / nas.tail1234.ts.net / 100.101.2.2 / linux · Online · direct" → "Copy name". The Home server card says "Tailnet names: Work on every device connected to this router while tailbond is on. Use the full name, like nas.tail1234.ts.net." The section text says "reach these through the bond to homebox (not this router's own Tailscale) … 'Direct' / 'via relay' is how homebox reaches each machine."

**What worked:** I found the name in one look, and I know the short name won't resolve. I also know which Tailscale node actually reaches the NAS, and what direct/relay refers to.

**Snag:**
- "Work on every device connected to this router" is stronger than DNS can promise. Nothing says how names resolve (presumably the router's resolver forwarding *.ts.net to homebox). A laptop with its own DNS (DoH in the browser, a hard-coded 1.1.1.1, iCloud Private Relay) won't get those names. A hint like "devices must use the router's DNS" would save me a debugging session. **Low.**

## Task 6 — Turn tailbond off

**Clicks:** The header toggle on 05 leads to 09: "tailbond is off / Devices on this router use the internet directly, without your home server, over whichever connection GL's own settings pick." There's a warning, "While tailbond is off, GL may use Tethering (your phone's data) … 'Backup only' and data limits only apply while tailbond is on", with a "GL's INTERNET page" button. Unpair is separate (09b: "forgets the home server … it will need to approve").

**What worked:** It's clear what off means for routing, and it warns me about the phone's data. Off and Unpair are clearly different things.

**Snag:**
- The main toggle has no text label on GL (05/09). On the Fedora page (14/15) the server's toggle is labelled "Connected", which reads oddly on a home server: does it disconnect all clients, or stop the server? Minor. **Low.**

## Task 7 — Approve on the home side

**Clicks:** Flint 11: "Waiting for your approval · 1 new / slate7 wants to connect … Approve". 12 shows "slate7 Connected just now · 2 connections bonded · downloading 48 Mbps". On Fedora (14) the same card is at the very top. On the phone version (14-phone) the card is the first thing on screen with big Approve/Decline buttons.

**What worked:** The disclosure is what I want: "Approving lets a device use this router's internet connection (by default all its traffic goes out from here) and your home network, and reach your Tailscale machines as this router, under its access rules." That tells me about SNAT as homebox and that homebox's ACLs apply. Who can approve is spelled out.

**Snags:**
- Flint (11/12) says "tailbond page password: None: only alex@example.com's devices can use http://homebox:8077; set one to let others approve". Fedora (15) says "A password instead replaces all this: then everyone who approves, you included, needs it", and also offers "Also allowed to approve (Tailscale logins, comma separated)". The Flint copy doesn't warn that a password replaces login-based approval, and it has no allow-list field. A Flint owner who sets a password "to let others approve" will then need it themselves. **Low.**
- "by default all its traffic goes out from here" implies a non-default option, such as split tunnel or a different exit. None of these screens shows such a setting, and the travel-router side never says where its internet exits. I can work it out from 08/09, but I'd want it stated. **Low.**

## Task 8 — Hotel Wi-Fi with a sign-in page, tailbond on

**Clicks:** 16 shows "Repeater needs you to sign in … Open it and click 'Login Mode for Public Hotspots' (or 'Go to Login') …" with an "Open GL's Repeater page" button. GL's own Repeater page isn't in the packet, so I'm relying on the text there. 16b shows "Router connected to homebox · devices signing in", the sign-in page on my laptop, then "Login Mode ends by itself once you've signed in … tailbond then adds Repeater back to bonding."

**What worked:** 16b is exactly what I want to know: "Until you've signed in, everything devices here do goes out the hotel Wi-Fi directly (not only the sign-in page, and not through your home server), so your phone's data isn't used for them. Only this router's own link to your home server stays up over your other connections (the top card; it carries almost nothing meanwhile)." Tailnet machines adds "While devices are signing in to a hotel page they go out directly, so they don't reach these until that's done."

**Snag:**
- 16, the screen I read *before* deciding, says: "While you sign in, devices here use the hotel Wi-Fi directly, not through your home server. Afterwards tailbond adds Repeater back to bonding by itself. **Your other connections keep working meanwhile.**" I read that last sentence as: my devices keep bonding over tethering, and only the portal traffic goes direct. 16b then says the reverse: *all* device traffic goes out the hotel Wi-Fi unprotected, and the tethering link carries almost nothing. I only learn that this isn't a portal-only exception after I've started Login Mode. If I'd had a sync or a work VPN running, I'd have wanted to know first. **Medium.**

## Task 9 — Ethernet with a sign-in page

**Clicks:** 17: "Ethernet needs you to sign in / GL's Login Mode covers the repeater Wi-Fi only; tailbond handles sign-in pages on other connections like this one." Then "Open sign-in page". 17b: "Signing in to Ethernet… For up to 9 more minutes (until 05:17 on this router's clock), all traffic from devices here goes out Ethernet directly, not just the sign-in page, and not through your home server. The router's own tunnel stays up on your other connections." Buttons: "Open the sign-in page again", "I've signed in, check now", "Stop sign-in mode".

**What worked:** A bounded window, a manual re-check, and an explicit stop. That's good. Ethernet is kept out of bonding until I'm signed in, so it can't blackhole the bond.

**Snag:**
- On 17 the hierarchy is odd. The small line on top ("GL's Login Mode covers the repeater Wi-Fi only…") is followed by a large line, then a small line: "Opening the sign-in page lets devices here reach Ethernet directly (without your home server) for up to 10 minutes". That understates things: only 17b says "all traffic … not just the sign-in page". At least it says "devices here … without your home server" before I click, which is better than 16. **Low.**

## Task 10 — Best speed at home

**Clicks:** Flint: Setup health. 18 says "2 to improve": "Tailscale: 2 machines direct, 1 through a relay (slower) … On this router: NETWORK → Firewall → Open Ports on Router, add UDP 41641. If another router sits in front of this one … also forward UDP port 41641" and "This network makes direct Tailscale connections hard … ('hard NAT') … Same fix as above". 18b is CGNAT: "This machine's internet address 100.72.3.4 is shared by your provider … Bonding needs that direct connection; it can't run through Tailscale's relays … Ask your provider for a public IPv4 address … a public IPv6 address also works, but only from networks that have IPv6 too". 18c: "All good / Everything is set up for best speed", with checks for route 10.77.0.0/24, public IP, and "UDP 443 reached from the internet / pixel-8 connected from 198.51.100.20, 2 h ago". 13 covers first setup: approve route 10.77.0.0/24 in the admin console, and set DDNS. Fedora (15): "forward UDP port 41641 to this machine (192.168.1.2). (Turning on UPnP on that router also works, but lets any device on your network open ports.)"

**What worked:** This is the strongest part. It separates the Tailscale-path problem (41641, relay vs direct, hard NAT) from the bonding-path requirement (UDP 443, CGNAT). It's concrete about which router and which port, the UPnP caveat is honest, and it says "bonding itself doesn't go through Tailscale". 13's "Until then, devices connected through tailbond can still reach your Tailscale machines; only the other direction … waits for this" tells me exactly what the route approval affects.

**Snags:**
- On the Flint, 12/18c say "Tailscale: 3 machines direct", but the Tailnet machines list on the same page shows only 2 online (nas, apple-tv; pixel-8 and work-laptop last seen 3 h / 1 day ago). I can't tell what's being counted. It might be slate7, which isn't in the list. When I'm judging direct vs relay, that kind of mismatch makes me distrust the number. **Low.**
- The Fedora health card (15) lists only the 2 problems. Unlike the Flint ("Show 3 more checks"), it never shows the checks that passed: UDP 443 reached from the internet, route approved, public address. "this machine's own firewall is already set up by tailbond" doesn't say which ports or firewalld zone it opened. On my own Fedora box I want to know what changed. **Low.**

---

## Findings

| # | Task | Finding | Severity |
|---|---|---|---|
| 1 | 1 | "Download the router package" (11–15) doesn't state the architecture or models; `aarch64_cortex-a53_neon-vfpv4` shows up only in LuCI's dialog (01b) | low |
| 2 | 1 | Fedora step 2 (15) leaves out "log in with the same password" and "leave the other options as they are", while LuCI shows an "Allow overwriting conflicting package files" checkbox | low |
| 3 | 2 | 02/03 "must be signed in to the same Tailscale account as this router" contradicts the server's "people you share homebox with, can ask to connect" | low |
| 4 | 3 | Fedora path cards "Path 0 / Pixel-8 · Min 31 Ms" aren't named by uplink, and "Min 31 Ms" is unclear | low |
| 5 | 4 | "GL itself may check the connection now and then" doesn't say whether GL's own Tailscale, cloud or multi-WAN traffic stays off the phone while tailbond is on | low |
| 6 | 5 | "Tailnet names: Work on every device connected to this router" doesn't explain the DNS mechanism; devices using their own DNS or DoH won't resolve *.ts.net | low |
| 7 | 6 | GL main toggle has no label; on the Fedora server the toggle is labelled "Connected", which is ambiguous | low |
| 8 | 7 | Flint "set one to let others approve" doesn't warn that a password replaces login approval (Fedora does: "everyone who approves, you included, needs it"); Flint has no "Also allowed to approve" list | low |
| 9 | 7 | "by default all its traffic goes out from here" implies a non-default exit option that no screen shows; the travel-router side never states where its internet exits | low |
| 10 | 8 | 16 says "Your other connections keep working meanwhile"; 16b then says all device traffic goes out the hotel Wi-Fi directly and the other link "carries almost nothing", so you learn the full scope only after starting Login Mode | medium |
| 11 | 9 | 17's mixed text sizes bury the scope; "lets devices here reach Ethernet directly" understates the "all traffic … not just the sign-in page" that only 17b states | low |
| 12 | 10 | Flint health "Tailscale: 3 machines direct" while the Tailnet machines list shows only 2 online; unclear what's counted | low |
| 13 | 10 | Fedora health (15) shows no passed checks (UDP 443 reached, route approved), unlike Flint's "Show 3 more checks"; "firewall is already set up by tailbond" doesn't list what it opened | low |

Totals: 0 high / 1 medium / 12 low
