# Round 6 — Persona B (homelab power user)

Who I am: I run Tailscale subnet routers, OpenWrt and a Fedora server at home. I care about what is actually happening: routes, DNS, whether a Tailscale connection is direct or relayed through DERP, NAT, how secure pairing is, who can approve, and where traffic goes during a captive-portal sign-in. I would rather not SSH into a travel router from a hotel room.

Inputs: task-script.md, screens.md, and the round-6 PNGs (all desktop screens, plus the phone versions of 05, 14 and 17b).

---

## Task 1: Install tailbond on the Slate 7 without SSH

**Where I click:** On the home server's page I click "Download the router package" (15: "Add a GL.iNet travel router (recommended)"; Flint 11–13: "Download the router package … It's the same tailbond this router runs."). On the Slate 7 I go to SYSTEM → Advanced Settings → "Go To LuCI" (01a: "LuCI is OpenWrt's own web interface. Use the same admin password to log in."). Then LuCI → Software → "Upload Package…" → `tailbond_0.1.0-r42_aarch64_cortex-a53_neon-vfpv4.ipk (6.1 MB)` → "Install", leaving "Allow overwriting conflicting package files" unchecked as the home page says (01b).

**How it went:** No problems. I'm used to OpenWrt, so an `.ipk` uploaded in LuCI is normal for me. The path in the instructions matches the GL and LuCI screens step for step.

**Nits:**
- Neither home page shows a checksum or signature for the package. I'm installing a 6.1 MB root-level package with LuCI's opkg. Since it comes from my own server it's probably fine, but I'd like to see a sha256 next to the download button. (low)
- On the Fedora box (x86) there is one "Download the router package" button and no architecture choice. The file I got happens to be `aarch64_cortex-a53`, which is right for the BE3600. I had to check the filename myself to know it would fit, and nothing says which routers it supports. (low)

## Task 2: Pair the Slate 7 with the home server

**Where I click:** VPN → tailbond. 02 shows "Looking for your home server on your tailnet…". Then 03 shows "Found your home server: homebox / homebox.tail1234.ts.net · 100.101.1.1", and I click "Ask to connect". 04 shows "Waiting for approval on homebox… On homebox, tap **Approve** next to **slate7**."

**How it went:** Good. Discovery through the tailnet is what I'd expect. The no-server case (03c) says "Checked 4 online machines", which tells me it scanned my peers and didn't just give up. 03b ("Turn on Tailscale first") is clear. On the home side, the request shows "signed in as alex@example.com · GL-BE3600 · 100.101.9.9". That's enough to match it against `tailscale status`.

**Nits:**
- The Slate's waiting card (04) doesn't show the Slate's own 100.x address or any short code, so I can't check that the request on homebox really comes from this box. I can work it out from the Tailscale admin console, so this is low. (low)

## Task 3: Hotel Wi-Fi (Repeater) plus USB tethering, confirm both are used

**Where I look:** 05. The header says "Connected to homebox / Using Repeater (Wi-Fi) + Tethering (phone)". Both cards show "In use" with their own numbers: Repeater "↓ 38 Mbps ↑ 4.2 Mbps · 24 ms", Tethering "↓ 21 Mbps ↑ 2.9 Mbps · 48 ms". In 06, a failed path shows "Not working right now — It's connected, but can't reach homebox. tailbond keeps trying; the other connections carry your traffic." On the Fedora side, 15 shows "#0 active · 31 ms · 0.0%" and "#1 active · 58 ms · 0.0%". These per-path latency and loss numbers are exactly what I want.

**What I couldn't tell:**
- Does all my LAN traffic exit from home, or only tailnet traffic? The subtitle says "Bond this router's internet connections into one tunnel to your home server". The failure banner (08) says "devices on this router have no internet, so nothing goes around your home server". From those I can infer a full tunnel with a kill switch, but no screen says "your internet traffic leaves from your home IP". Nothing says which DNS resolver the LAN clients use either: the Slate's, homebox's, or MagicDNS at 100.100.100.100. (low)

## Task 4: Phone as backup only

**Where I click:** On the Tethering card, Use it → "Backup only" (07). The badge changes to "Backup · saving data", and the card says "Used only if your other connections stop working. Until then it isn't used at all, so it uses no data." The header reads "Tethering (phone) on standby". "Change limit" (07b) opens "5 GB a month … At the limit, tailbond stops using this connection until the count restarts on the 1st of each month. 0 = no limit."

**How it went:** Easy to find, and the result is clear.

**Nits:**
- "Isn't used at all, so it uses no data": then how does it notice that the hotel Wi-Fi died, and how quickly does it fail over? I'd expect at least some keepalive traffic. A line like "switches over within ~N s" would help. (low)
- The Ethernet and Repeater cards say "Set limit" and the Tethering card says "Change limit". The note under the modes mentions "Set Tethering or Cellular", but there's no Cellular card on this router. (low)

## Task 5: Reach the NAS by its MagicDNS name

**Where I look:** The "Tailnet machines" card (05) lists "nas / nas.tail1234.ts.net / 100.101.2.2 / linux · Online · direct" with a "Copy name" button. The Home server card says "Tailnet names: Work on every device connected to this router while tailbond is on. Use the full name, like nas.tail1234.ts.net." The card explains the routing too: "reach these through the bond to homebox (not this router's own Tailscale) … 'Direct' / 'via relay' is how homebox reaches each machine."

**How it went:** Very good. It tells me the traffic goes through homebox and not through the Slate's own tailscaled, and that "direct" is measured from homebox. That's the level of detail I want.

**Nits:**
- The tunnel uses 10.77.0.0/24 (tunnel addresses 10.77.0.1 and 10.77.0.2; Flint "Route 10.77.0.0/24 approved"). My homelab uses 10.x a lot, and no screen shows a way to change that subnet if it collides with one of my routes. (low)

## Task 6: Turn tailbond off

**Where I click:** The header toggle. 09 shows "tailbond is off / Devices on this router use the internet directly, without your home server." "Unpair" is separate, and its dialog is clear (09b: "forgets the home server … it will need to approve").

**How it went:** Fine. Off and Unpair are clearly two different things.

**Nits:**
- With tailbond off, can LAN devices still reach the NAS through the Slate's own Tailscale (subnet route or exit node)? The card only says "The list appears once tailbond is connected." (low)

## Task 7: Approve on the home side (Flint 3 and Fedora)

**Flint (11):** "Waiting for your approval / slate7 wants to connect … Approve". The card says "Who can approve: anyone logged in to this admin panel, and anyone on your Tailscale account who opens http://homebox:8077. To require a password there, set one under Internet and Tailscale." There's a "tailbond page password … Set" link right there. That's good, and the approval list is spelled out.

**Fedora (14/15):** I click "Approve" in the same kind of card. The card says: "Who can approve: anyone who can open this page, i.e. anyone on your Tailscale account (and anyone you shared this machine with). To require a password, set "ui_password" in /etc/tailbond/tailbond.json (see the Fedora server guide)."
- I share homebox with a couple of friends through Tailscale node sharing. By default, every one of them can approve new devices. Those devices then get bonded into my home and onto my tailnet behind a subnet route. The page does warn about it, which I appreciate. But the only fix is editing a JSON file on the server, while the Flint has a "Set" button. Once I read that line I'd stop and go SSH into the Fedora box before approving anything. Approval on the box that bridges into my tailnet should be locked down by default, or at least lockable from the page. (medium)
- The dashboard header shows "1.1 MBPS DOWN / 12.0 MBPS UP", but the pixel-8 row on the same page shows "↓ 12.0 Mbps ↑ 1.1 Mbps". One is from the server's point of view and the other from the device's. The Flint avoids this with "Sending to devices / Receiving". (low)
- Ports don't match. The Slate (04) and the Flint pages say "http://homebox:8077", but the Fedora "Invite someone's phone" link is "http://homebox.tail1234.ts.net:80/#add-phone". Which port does the Fedora page actually listen on, and is :80 open on the tailnet interface too? (low)

## Task 8: Hotel Wi-Fi with a sign-in page (GL repeater detected it)

**Where I click:** 16: "Repeater needs you to sign in … GL's Repeater page handles this sign-in. Open it, click 'Login Mode for Public Hotspots' … While you sign in, devices here use the hotel Wi-Fi directly, not through your home server." I click "Open GL's Repeater page" and start Login Mode. Back on tailbond, 16b says: "GL's Login Mode is on: all traffic from devices here goes out the hotel Wi-Fi directly (not just the sign-in page), not through your home server … It ends by itself once you've signed in (or click 'Quit Login Mode' …)".

**How it went:** The page is upfront that everything leaks to the hotel network during sign-in, not just the portal. Good. But the rest of 16b contradicts that:
- The header still says "Connected to homebox", and the Tethering card still says "In use" with live numbers ("↓ 21 Mbps ↑ 2.9 Mbps") and Use it = Always. If all traffic goes out the Repeater, what is the phone carrying? Is it burning my mobile data on a tunnel nobody uses? Does nas.tail1234.ts.net still resolve and route through homebox during Login Mode? The Tailnet machines list is still there, which suggests yes. Compare 17b, which states "The router's own tunnel stays up on your other connections". 16b has no such line, so I can't tell whether this is the same model or whether GL's Login Mode overrides tailbond completely. (medium)
- 17 caps its own sign-in at "up to 10 minutes". 16b gives no time limit for GL's Login Mode. It's GL's feature, but a note like "GL doesn't time this out; quit it when done" would tell me how long I'm exposed. (low)

## Task 9: Hotel Ethernet with a sign-in page

**Where I click:** 17: "Ethernet needs you to sign in … Opening the sign-in page lets devices here reach Ethernet directly (without your home server) for up to 10 minutes, so the page can load." I click "Open sign-in page". 17b: "For up to 9 more minutes (until 04:18 on this router's clock), all traffic from devices here goes out Ethernet directly, not just the sign-in page … The router's own tunnel stays up on your other connections." I click "I've signed in, check now", and "Stop sign-in mode" is available.

**How it went:** This is the best screen in the set for my persona. It tells me what leaks, for how long, until when on the router's clock, what happens to the tunnel, and how to end it early. No real problems. The phone layout (17b-phone) keeps all three buttons readable.

## Task 10: Home side set up for best speed

**Flint, all good (18c):** "Everything is set up for best speed": "Tailscale: 3 machines direct", "Network is friendly to direct connections (port mapping: UPnP)", "Route 10.77.0.0/24 approved", "Public internet address 203.0.113.7", "UDP 443 reached from the internet (pixel-8 connected from 198.51.100.20, 2 h ago)". These are real checks with evidence. Great.

**Flint, relayed (18):** "Tailscale: 2 machines direct, 1 through a relay (slower) / Through a relay: pixel-8. To check one: tailscale ping pixel-8 / On this router: NETWORK → UPnP, turn it on … forward UDP 41641". This gives me the exact command and port. "Show 3 more checks" hides the passing checks.
- pixel-8 is flagged "via relay (slower)", but pixel-8 is also a tailbond device that is "Connected … 2 connections bonded" over UDP 443. As far as I can tell, the relay only affects traffic to pixel-8's 100.x address, not its bond. The card doesn't say whether this warning affects tailbond speed for that device. (low)

**Flint, CGNAT (18b):** "Home server is on · 1 setup step left … Step 1: Your internet line uses carrier-grade NAT … 100.72.3.4 is shared by your provider, so travel routers can't reach UDP 443 here. Ask your provider for a public IPv4 address (or use public IPv6)."
- On the same screen, Devices shows "pixel-8 Connected 3 h 0 min · 2 connections bonded · downloading 12 Mbps". So how is pixel-8 connected if travel routers can't reach UDP 443? Is there a fallback through Tailscale, or is pixel-8 only connected because it's on the home LAN? Nothing explains it. Every device here is already on my tailnet, and nothing says whether tailbond can run over Tailscale when the line is CGNAT. "Ask your provider" is the only path offered. I'd be unsure whether my Slate 7 will work from the hotel, and I'd end up testing it from a hotspot. The Setup health card is also missing from this state, so I can't see the other checks. (medium)

**Fedora (15):** The setup health card under Devices lists the two problems ("2 machines direct, 1 through a relay"; "Hard NAT … no port mapping on the router. Forward UDP port 41641 …") with "How to fix" links. On the Connections tab it shows as a red bar: "Setup health: 2 things to improve for best speed. Show".
- Fedora only lists problems. Unlike the Flint, it doesn't show the passing checks (route approved, public address, UDP 443 reached). I'd like the same "Show N more checks" here so I know the route was approved. (low)

---

## Findings

| # | task | finding | severity |
|---|---|---|---|
| 1 | 1 | No checksum or signature shown for the downloaded `.ipk` installed with root rights through LuCI | low |
| 2 | 1 | Fedora's single "Download the router package" button doesn't say which architecture or routers it covers; I had to check the filename (`aarch64_cortex-a53`) | low |
| 3 | 2 | Slate's "Waiting for approval" card shows no own 100.x address or code to match against the request on homebox | low |
| 4 | 3 | No explicit statement that all LAN internet traffic exits at home, or which DNS resolver LAN clients use; only inferred from "nothing goes around your home server" | low |
| 5 | 4 | "Isn't used at all, so it uses no data": failover detection and switch time not explained | low |
| 6 | 4 | "Set limit" vs "Change limit" wording; the note mentions "Cellular" on a router with no Cellular card | low |
| 7 | 5 | Tunnel subnet 10.77.0.0/24 is fixed, with no visible way to change it if it collides with homelab routes | low |
| 8 | 6 | When off, no word on whether LAN devices still reach the tailnet through the Slate's own Tailscale | low |
| 9 | 7 | Fedora: by default anyone homebox is shared with can approve new devices into my tailnet; the only lock is editing `/etc/tailbond/tailbond.json` (the Flint has a "Set" button) | medium |
| 10 | 7 | Fedora header "1.1 MBPS DOWN / 12.0 MBPS UP" vs pixel-8 row "↓ 12.0 ↑ 1.1": point of view flips on the same page | low |
| 11 | 7 | Port mismatch: "http://homebox:8077" (Slate and Flint) vs Fedora invite link "http://homebox.tail1234.ts.net:80/" | low |
| 12 | 8 | 16b says "all traffic … goes out Repeater directly", yet the header says "Connected to homebox" and Tethering is "In use" with live Mbps; unclear whether phone data is spent or tailnet names still work (17b answers this, 16b doesn't) | medium |
| 13 | 8 | No time limit or exposure window stated for GL's Login Mode (tailbond's own mode is capped at 10 min) | low |
| 14 | 10 | CGNAT state says "travel routers can't reach UDP 443 here" while pixel-8 shows "Connected 3 h 0 min · 2 connections bonded"; no explanation and no over-Tailscale option, only "Ask your provider"; setup health card hidden | medium |
| 15 | 10 | pixel-8 flagged "via relay (slower)" though it's bonded directly over UDP 443; doesn't say whether the relay affects tailbond speed | low |
| 16 | 10 | Fedora health shows only failing checks; no "Show N more checks" to confirm route approval and UDP 443 reachability | low |

Totals: high 0, medium 3, low 13.
