# Round 12 — GL persona B (homelab)

**Persona:** homelab power user. I run Tailscale subnet routers, OpenWrt and a Fedora server at home. I want to know what actually happens: routes, DNS, direct vs relayed (DERP), NAT, who can approve, and where traffic goes during captive-portal sign-in. I don't want to SSH into a travel router from a hotel.

**Materials used:** `task-script.md`, `screens.md` and the PNGs in `packet-gl/` only. I looked at all desktop screens and at the phone versions of 01b, 05, 09, 14 and 17b.

Overall: I could finish every task from the screenshots. None of them needed SSH. The wording answers most of my "what really happens" questions on the screen itself:
- The bond runs over "UDP port 443", not Tailscale ("bonding itself doesn't go through Tailscale").
- It says what happens to traffic when home can't be reached ("devices on this router have no internet").
- It says where traffic goes while I sign in to a hotel page.
- It spells out who can approve, down to "any user or program on homebox itself (http://127.0.0.1:8077…)".

My remaining problems are about how deep the information goes, not about getting stuck.

---

## Task 1 — Install on the Slate 7 without SSH

**Clicks:**
1. On the Fedora page (15), in "Add a GL.iNet travel router (recommended)", I click **Download the router package**. The Flint 3 panel (11–13) has the same button.
2. On the Slate 7 (01a) I go to SYSTEM → Advanced Settings and click **Go To LuCI** (`http://192.168.8.1:8080/cgi-bin/luci`). The page notes "Use the same admin password to log in."
3. In LuCI (01b) I go to System → Software → **Upload Package…**. The dialog shows `tailbond_0.1.0-r42_aarch64_cortex-a53_neon-vfpv4.ipk (6.1 MB)`. I leave "Allow overwriting conflicting package files" unticked, as the Flint text says ("leave the other options as they are"), and click **Install**.

**Where I hesitated:** Neither download button says which architecture or models the package is for. On the Flint it says "It's the same tailbond this router runs", which is fine Flint-to-Slate. The Fedora page (an x86 box, presumably) just says "Download the router package". I only learned the arch from the filename in LuCI's dialog. I'd have liked "for GL routers (aarch64_cortex-a53)" next to the button so I know before uploading. **Low.**

Good: the three-step recipe on the home server's page matches the real menu path exactly. The Flint version also reassures me that "The router's internet keeps working."

## Task 2 — Pair with the home server

**Clicks:** I open VPN → tailbond (02, "Looking for your home server on your tailnet…"). Next comes 03: "Found your home server: homebox — homebox.tail1234.ts.net · 100.101.1.1". I click **Ask to connect**. Screen 04 then says "Waiting for approval on homebox… On homebox, tap Approve next to slate7… From anywhere: open http://homebox:8077 on any device signed in to your Tailscale account".

The fallback screens are good:
- 03b says "Turn on Tailscale first", with "Open Tailscale settings".
- 03c says "Checked 4 online machines. On the home server, tailbond must be installed and running, and Tailscale on."

As a Tailscale user, I find this discovery-plus-approval model clear. No issues.

## Task 3 — Hotel Wi-Fi (Repeater) + USB tethering, confirm both used

**Where to look:** On screen 05, the top card says "Connected to homebox — Using Repeater (Wi-Fi) + Tethering (phone)". Both connection cards show **In use**, with per-link numbers: "↓ 38 Mbps ↑ 4.2 Mbps · 24 ms" and "↓ 21 Mbps ↑ 2.9 Mbps · 48 ms". These add up to the top card's "59 Mbps". On the Flint side (12), slate7 shows "2 connections bonded". Confirmed.

The Home server card answers my transport question: "Connects to home.example.net over the internet (UDP port 443)". The Tailnet machines card says LAN clients reach the tailnet "through the bond to homebox (not this router's own Tailscale)", and that "'Direct' / 'via relay' is how homebox reaches each machine." That is exactly the routing picture I wanted.

Screen 06 (phone trouble) is also clear: "It's connected, but can't reach homebox. tailbond keeps trying; the other connections carry your traffic."

No issues for this task.

## Task 4 — Phone as backup only

**Clicks:** On the Tethering card I click **Backup only** (07). The status becomes "Backup · saving data", with "Used only if your other connections stop working. Until then tailbond sends nothing over it (GL itself may check the connection now and then, a few KB)." **Change limit** (07b) gives a GB-per-month field with "Counts everything tailbond sends and receives over it… 0 = no limit."

This is honest, which I appreciate. Two questions remain:
- The page says what *tailbond* and *GL's connectivity check* send over the phone. It doesn't say which uplink the Slate's *own* traffic uses while tailbond is on, such as its own Tailscale daemon or GL cloud. That traffic is also outside the "Counts everything tailbond sends" limit. With a capped phone plan, I'd want one line about it. **Low.**
- "Most reliable — Sends everything over every connection. Uses the most data." It isn't clear whether that mode still respects "Backup only". The hint under it ("Saving your phone's data? Set Tethering… to 'Backup only'") suggests it does, but it doesn't say so. **Low.**

The turned-off warning on 09 covers the other gap well: "While tailbond is off, GL may use Tethering (your phone's data)".

## Task 5 — Reach the NAS by MagicDNS name

**Where to look:** On 05, the **Tailnet machines** card lists `nas`, `nas.tail1234.ts.net`, `100.101.2.2`, "linux · Online · direct", with a **Copy name** button. The Home server card says "Tailnet names: Work on every device connected to this router while tailbond is on. Use the full name, like nas.tail1234.ts.net."

I'd normally expect the short name `nas` to work through the search domain. The page heads that off clearly, so I wasn't confused.

What I couldn't find is where the laptop's *general* DNS goes while bonded: the hotel's resolver, or out from home along with the traffic. The page only says tailnet names work. For a "goes out from home" design, a DNS leak to the hotel is something I'd check. **Low.**

## Task 6 — Turn tailbond off

**Clicks:** I flip the switch at the top right of the status card. Screen 09 then says "tailbond is off — Devices on this router use the internet directly, without your home server, over whichever connection GL's own settings pick." Screen 08 also has a **Turn tailbond off** button.

- On the GL page, the main switch has no label next to it. The Fedora page has "Connected" beside its switch. It worked, but I had to guess that this switch was the main on/off. Separately, screen 10 ("tailbond isn't running… It may have been turned off or stopped after an error") blurs "turned off with the switch" and "service stopped". I wasn't sure whether 09 and 10 are the same state. **Low.**
- Screen 10 gives no reason when the service has stopped. There's "Start tailbond", but no error text or log link on the GL side. The Fedora page has a Logs tab; the GL page doesn't. If it keeps dying, my only option is SSH, which is what I'm trying to avoid. **Low.**
- The Unpair dialog (09b): "This router stops using tailbond and forgets the home server… it will need to approve." It doesn't say whether homebox also drops slate7 from its Devices list, or whether I should click Remove there too. **Low.**

## Task 7 — Approve on the home side

**Flint 3 (11):** Under "Waiting for your approval — 1 new" there's "slate7 wants to connect — Travel router on your tailnet · signed in as alex@example.com · GL-BE3600 · 100.101.9.9 · asked 1 min ago", with Decline and **Approve**. After approving (12), slate7 shows "Connected just now · 2 connections bonded".

**Fedora (14):** There's the same request card with **Approve**.

The approval model is spelled out to the level I want:
- "Who can approve: anyone logged in to this admin panel, and alex@example.com…"
- "any user or program on homebox itself (http://127.0.0.1:8077; set a password if homebox runs things you don't trust)"
- The ACL caveat "they allow it whatever they allow homebox, which may be more than its own owner's account gets."
- The warning that slate7 is itself signed in to an approver account.

One small thing: the Fedora card says "Approving lets the device use homebox's internet connection (by default all its traffic goes out from here)". "By default" suggests there's a split-tunnel setting, but I can't see it on any Slate or home screen. **Low.**

## Task 8 — Hotel Wi-Fi with a sign-in page (GL repeater)

**Clicks:**
1. Screen 16 says "Repeater needs you to sign in". I click **Open GL's Repeater page**, click "Login Mode for Public Hotspots", then open any website.
2. Screen 16b says "Router connected to homebox · devices signing in", and the Tethering card shows "Router's link only".
3. "Login Mode ends by itself once you've signed in… tailbond then adds Repeater back to bonding."

This answers my main captive-portal question exactly: "all traffic from devices here goes out Repeater (Wi-Fi) directly, not through homebox, until you're signed in", and the phone carries "almost no data" for the router's own link.

The Wi-Fi flow is thinner than the Ethernet flow in Task 9:
- 16 says "until it's done, or 10 minutes", but 16b shows no countdown.
- 16b has no "I've signed in, check now" or "Stop sign-in mode". I have to go back to GL's page and click "Quit Login Mode".

It's understandable because GL owns this mode, but the two flows feel inconsistent. **Low.**

## Task 9 — Ethernet cable with a sign-in page

**Clicks:** Screen 17 says "Ethernet needs you to sign in" and explains "GL's Login Mode covers the repeater Wi-Fi only; tailbond handles sign-in pages on other connections like this one". I click **Open sign-in page**. Screen 17b then shows a countdown ("For up to 9 more minutes (until 05:34 on this router's clock)"), **I've signed in, check now**, and **Stop sign-in mode**. Ethernet comes back into bonding "by itself once you're signed in." Done.

- While I sign in on the cable, the hotel Wi-Fi that was working goes to "Router's link only". All device traffic, including anything that would have gone to the tailnet, goes straight out the unauthenticated cable for up to 10 minutes. It's clearly disclosed ("everything devices here do goes out Ethernet directly… they can't reach your Tailscale machines"). Still, as a homelab person I'd have liked only the browser doing the sign-in to go direct, with everything else staying bonded over the Wi-Fi that works. **Low**, because it's disclosed and I chose to do it.
- In the 17 banner, the small-print line ("GL's Login Mode covers the repeater Wi-Fi only…") sits *above* the larger main sentence, "It shows a sign-in page before it lets anyone online." On first read, "It" seemed to refer to GL's Login Mode, not the Ethernet network. **Low.**

## Task 10 — Best speed at home

**Flint 3:**
- 18 ("2 to improve") says: "Tailscale: 2 machines direct, 1 through a relay (slower) — Through a relay: apple-tv… bonding itself doesn't go through Tailscale", with the fix "NETWORK → Firewall → Open Ports on Router, add UDP 41641. If another router sits in front of this one… also forward UDP port 41641". It also flags "This network makes direct Tailscale connections hard… ('hard NAT')… Same fix as above".
- 18b (CGNAT) is excellent: "This machine's internet address 100.72.3.4 is shared by your provider… Bonding needs that direct connection; it can't run through Tailscale's relays". It also suggests a public IPv4 or IPv6.
- 18c lists every passing check: Tailscale direct, network friendly, "Optional route 10.77.0.0/24 approved", "Public internet address 203.0.113.7", "UDP 443 reached from the internet — pixel-8 connected from 198.51.100.20, 2 h ago."

That's exactly the checklist I'd want.

One small thing: tailbond says "UDP 443, opened on this router automatically", yet asks me to add UDP 41641 by hand in GL's firewall. If it can open one port, I'd expect it to offer to open the other. **Low.**

**Fedora (15):**
- "Setup health: 2 optional speed-ups (tailbond works fine without them)", with "forward UDP port 41641 to this machine (192.168.1.2)" and "this machine's own firewall is already set up by tailbond".
- Unlike the Flint, it only lists the problems. I can't confirm the things that matter most for a Fedora box behind my own router: that **UDP 443 is reached from the internet**, that the public address isn't CGNAT, and whether the 10.77.0.0/24 route is approved. There's no "Show N more checks" like on the Flint.
- The footer says "Router fixes are on your home router's admin page, route ones in the Tailscale admin console", but no route item is shown, so I'm left wondering whether the route needs doing.
- The only hint that 443 works is the header "reachable at home.example.net:443" and a pixel-8 public IP in Devices.

To "make sure everything is set up", I'd end up checking my router and the Tailscale admin console myself. **Medium.**

- On the Fedora Connections tab (14), the paths read "Path 0 · Pixel-8 · Min 31 Ms" and "Path 1". Nothing says which uplink each path is (Wi-Fi or cell), and "Ms" is capitalised. On the phone view, the device name wraps as "pixel- / 8". The phone screenshot is also 796 px wide against 780 for the GL pages, which suggests a slight sideways scroll. **Low.**

---

## Findings

| # | Task | Finding | Severity |
|---|---|---|---|
| 1 | 1 | "Download the router package" (esp. on the Fedora page) doesn't say which architecture or models it's for; I only see `aarch64_cortex-a53_neon-vfpv4` in LuCI's dialog | low |
| 2 | 4 | "tailbond sends nothing over it (GL itself may check…)" doesn't say which uplink the Slate's own traffic (its Tailscale, GL cloud) uses, and that traffic isn't in the "Counts everything tailbond sends" limit | low |
| 3 | 4 | Unclear whether "Most reliable — Sends everything over every connection" still respects "Backup only" | low |
| 4 | 5 | Only "Tailnet names: Work on every device…" is stated; nothing says where general DNS goes while bonded (hotel resolver or home) | low |
| 5 | 6 | The GL page's main on/off switch has no label (Fedora shows "Connected"); 10's "may have been turned off" blurs switch-off and service-stopped | low |
| 6 | 6 | 10 "tailbond isn't running… stopped after an error" gives no reason or log link on the GL page, so it's SSH or nothing | low |
| 7 | 6 | The Unpair dialog doesn't say whether homebox also drops or revokes slate7 | low |
| 8 | 7 | "by default all its traffic goes out from here" hints at a split-tunnel option I can't find anywhere | low |
| 9 | 8 | The Wi-Fi Login Mode flow (16b) has no countdown, "I've signed in, check now" or "Stop sign-in mode", unlike the Ethernet flow (17b) | low |
| 10 | 9 | During Ethernet sign-in, the working Wi-Fi drops to "Router's link only" and all device traffic goes raw out the cable for up to 10 min (disclosed; I'd prefer only the browser to go direct) | low |
| 11 | 9 | In the 17 banner, small print sits above the main line; "It shows a sign-in page…" reads as if "It" is GL's Login Mode | low |
| 12 | 10 | Fedora setup health lists only the 2 problems; I can't confirm UDP 443 reached from the internet, public IP vs CGNAT, or the 10.77.0.0/24 route, and the footer mentions "route ones" with no route item | medium |
| 13 | 10 | The Flint opens UDP 443 automatically but asks me to add UDP 41641 by hand ("NETWORK → Firewall → Open Ports on Router") | low |
| 14 | 10 | Fedora paths say "Path 0 · Pixel-8 · Min 31 Ms" with no uplink name; on the phone view, "pixel- / 8" wraps and the page is slightly wider than the screen | low |

Totals: 0 high / 1 medium / 13 low
