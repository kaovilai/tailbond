# Round 9 — GL persona B (homelab)

Persona: homelab power user. I run Tailscale subnet routers, OpenWrt and a Fedora server at home. I care what actually happens on the wire: routes, DNS, direct vs relayed (DERP), NAT, who can approve, and where traffic goes while a captive portal is being signed in. I'd rather not SSH into a travel router from a hotel.

Materials: `task-script.md`, `screens.md` and the PNGs in `out/ux/final/round-9/packet-gl/`. I looked at every desktop screen and the phone versions of 05, 11, 14 and 17b. Nothing else.

Overall: this is the clearest round I've seen for my kind of user. The panel answers most of my questions on screen. It says the bond runs over plain UDP 443 to my home address, not over Tailscale ("bonding itself doesn't go through Tailscale"). It says where traffic goes during sign-in ("all traffic from devices here goes out Ethernet directly, not just the sign-in page"). It says the router blocks traffic when it can't reach home ("devices on this router have no internet, so nothing goes around your home server"). And it says which identity the bonded devices use on the tailnet ("as homebox, under homebox's access rules"). Every task could be finished without SSH. What's left is mostly polish, plus one piece of IPv6 advice that would mislead me.

---

## Task 1 — Install tailbond on the Slate 7 without SSH

**Clicks:** Fedora page (15), card "Add a GL.iNet travel router (recommended)" → **Download the router package**. On the Slate 7: SYSTEM → Advanced Settings → **Go To LuCI** (01a; "Use the same admin password to log in", `http://192.168.8.1:8080/cgi-bin/luci`). In LuCI: System → Software → **Upload Package…** → pick file → dialog "Really install the uploaded package?" → **Install** (01b). Then VPN → tailbond appears (02).

**What worked:** the steps on the Fedora and Flint pages match GL's real menu path exactly. Using LuCI's uploader instead of `opkg install` over SSH is exactly what I wanted. The Flint version adds "(leave the other options as they are)" and "The router's internet keeps working", which answer the two things I'd worry about: the "Allow overwriting conflicting package files" checkbox, and losing my connection mid-install.

**Where I hesitate:**
- The download button never says which architecture or which GL models the package is for. I only learn it's `tailbond_0.1.0-r42_aarch64_cortex-a53_neon-vfpv4.ipk` once I'm in LuCI's dialog (01b). The Fedora box is x86_64 and the Flint card only says "It's the same tailbond this router runs". I own other GL boxes, so I'd want "for aarch64 GL routers: Slate 7, Flint 3, …" next to the button.
- The Fedora page's step 2 just says "pick the file, Install." It doesn't have the Flint's "leave the other options as they are" hint, so I'd pause at the overwrite checkbox.

Severity: **low**.

## Task 2 — Pair the Slate 7 with the home server

**Clicks:** VPN → tailbond (02 "Looking for your home server on your tailnet…") → 03 "Found your home server: homebox — homebox.tail1234.ts.net · 100.101.1.1" → **Ask to connect** → 04 "Waiting for approval on homebox… On homebox, tap **Approve** next to **slate7**. This router connects by itself once approved; you can close this page."

**What worked:** discovery over the tailnet, a fallback where I type the name ("Not your home server? Type its name instead"), and failure states that say exactly what's wrong. 03b says "Turn on Tailscale first" and has "Open Tailscale settings". 03c says "Checked 4 online machines. On the home server, tailbond must be installed and running, and Tailscale on." 04 also tells me where to approve from anywhere: "`http://homebox:8077` on any device signed in to your Tailscale account".

**Where I hesitate:** the prerequisite wording differs between the two ends. The Slate 7 says the home server "must be signed in to the same Tailscale account as this router" (02/03/04). The home side says "Others on your Tailscale network, and people you share homebox with, can ask to connect but can't approve" (14/15/11). My tailnet has several users and shared nodes. From the travel page I'd think a router logged in as another user can't pair at all. The home page says it can, it just can't self-approve.

Severity: **low**.

## Task 3 — Hotel: repeater + USB tethering, confirm both are used

**Clicks:** none. 05 headline: "Connected to homebox — Using Repeater (Wi-Fi) + Tethering (phone)", with 59 Mbps down. Per-link: Repeater "In use ↓38 Mbps ↑4.2 Mbps · 24 ms", Tethering "In use ↓21 Mbps ↑2.9 Mbps · 48 ms". 38 + 21 = 59, so I can see that both are actually carrying traffic, not just that both are up. Good. In 06, a sick link is explained well: "It's connected, but can't reach homebox. tailbond keeps trying; the other connections carry your traffic."

**Where I hesitate:**
- The per-link rows show rate and latency but no loss. The Fedora page's path cards do show it ("LOSS 0.0%", "#0 active · 31 ms · 0.0%"). Loss is the first thing I'd check on bad hotel Wi-Fi.
- Only Ethernet shows its interface name (`eth0`). Repeater and Tethering show neither an interface name nor, while connected, their IP.
- In 06 the failing tethering link gets no hint of why it can't reach homebox. Does the carrier block UDP 443? Is it a DNS failure?

Severity: **low**.

## Task 4 — Phone as backup only

**Clicks:** Tethering card → Use it: **Backup only** (07). The badge becomes "Backup · saving data", with "Used only if your other connections stop working. Until then tailbond sends nothing over it (GL itself may check the connection now and then, a few KB)." The headline says "Tethering (phone) on standby". **Change limit** (07b) gives "5 GB a month … Counts everything tailbond sends and receives over it. At the limit, tailbond stops using this connection until the count restarts on the 1st of each month."

**What worked:** it's honest about GL's own probe traffic. It also warns that the setting doesn't protect me when tailbond is off (09: "Backup only and data limits here apply only while tailbond is on, so GL may use your phone: unplug it…"). That's the exact trap I'd worry about with GL's own Multi-WAN.

**Where I hesitate:** "stop working" isn't defined. Does the backup take over only when the hotel Wi-Fi is dead, or also when it's lossy, slow, or sitting behind a portal? How fast does it switch? I also wondered whether "Most reliable — Sends everything over every connection" overrides Backup only. The footnote "Saving your phone's data? Set Tethering or Cellular to 'Backup only'" suggests it doesn't, but that's an inference.

Severity: **low**.

## Task 5 — Reach the NAS by MagicDNS name from the laptop

**Clicks:** Tailnet machines card (05): "nas — `nas.tail1234.ts.net` — `100.101.2.2` — linux · Online · direct" → **Copy name**. The Home server card says "Tailnet names: Work on every device connected to this router while tailbond is on. Use the full name, like nas.tail1234.ts.net." The section text says laptops "reach these through the bond to homebox (not this router's own Tailscale) … 'Direct' / 'via relay' is how homebox reaches each machine."

**What worked:** it's clear that the path goes through homebox, not the Slate's own tailscaled, and what "direct" refers to. Found the name in seconds.

**Where I hesitate:** a bare `nas` won't work ("Use the full name"). I'm used to the MagicDNS search domain doing that, but at least it's stated. The travel page doesn't say that, to the NAS, my laptop shows up as homebox and is subject to homebox's ACLs. The home pages do ("reach your Tailscale machines as homebox, under homebox's access rules"). I'd want that on the travel page too, because it decides whether my `tag:`-scoped ACLs let me in.

Severity: **low**.

## Task 6 — Turn tailbond off

**Clicks:** the header toggle in 05 → 09 "tailbond is off — Devices on this router use the internet directly, without your home server, over whichever connection GL's own settings pick (INTERNET page, top of the menu)…". "Unpair" is separate and confirmed (09b: "This router stops using tailbond and forgets the home server… (it will need to approve)"). "Not running" (10) is distinct: "Starting it is safe: devices here keep their internet while you set it up."

**Where I hesitate:**
- When it's off, I can't tell whether laptops still reach `nas.tail1234.ts.net` through the Slate's own Tailscale. All I get is "The list appears once tailbond is connected" and "Work on every device … while tailbond is on."
- On 08, the server is unreachable but the toggle is still ON. The link cards say "Ready" with "This choice applies when tailbond is on. While it's off, GL's own settings (INTERNET page) decide…". The text describes the off state while the switch shows on.
- "Ask to connect again" sits next to "Unpair" even in the Connected state (05). It's not clear what re-asking does when I'm already paired.

Severity: **low**.

## Task 7 — Approve the Slate 7 on the home side

**Clicks:** on the Flint (11), "Waiting for your approval — 1 new — slate7 wants to connect — Travel router on your tailnet · signed in as alex@example.com — GL-BE3600 · linux · 100.101.9.9 · asked 1 min ago" → **Approve**. 12 shows "slate7 Connected just now · 2 connections bonded". On Fedora (14/15) it's the same card → **Approve**.

**What worked:** I know who is asking: model, tailnet IP and login. The approver rule is spelled out: "anyone logged in to this admin panel, and alex@example.com … People you share this router with, and others on your Tailscale network, can ask to connect but can't approve." That matters on a multi-user tailnet.

**Where I hesitate:**
- What approving grants differs by platform. Fedora: "Approving lets the device use homebox's internet connection (by default all its traffic goes out from here) and your home network, and reach your Tailscale machines as homebox". Flint: "Approving lets a device reach your Tailscale machines the way this router does". The Flint text leaves out that the device gets my home IP for all its traffic and access to my home LAN, which is the bigger grant. Also, Fedora's "by default" hints at a split-tunnel option that I can't find on any screen.
- Delegating approval is configured differently on each side. Fedora has "Also allowed to approve (Tailscale logins, comma separated)", plus a password that "replaces all this". The Flint only offers "tailbond page password … set one to let others approve". On the Flint, adding a second approver means switching from Tailscale identity to a shared password.

Severity: **low**.

## Task 8 — Hotel Wi-Fi with a sign-in page (GL Login Mode)

**Clicks:** 16 shows the banner "Repeater needs you to sign in" → **Open GL's Repeater page** → GL's "Login Mode for Public Hotspots" → 16b "Router connected to homebox · devices signing in": "Now open any website on your laptop or phone: the hotel's sign-in page appears there… Login Mode ends by itself once you've signed in (or click 'Quit Login Mode' on GL's Repeater page); tailbond then adds Repeater back to bonding."

**What worked:** this is the answer I wanted about traffic during sign-in: "everything devices here do goes out the hotel Wi-Fi directly (not only the sign-in page, and not through your home server), so your phone's data isn't used for them." The tailnet list also says "While devices are signing in to a hotel page they go out directly, so they don't reach these until that's done." The tunnel stays up over tethering, and the page says so.

**Where I hesitate:**
- "click 'Login Mode for Public Hotspots' (or 'Go to Login')": hedging between two button names makes me wonder which firmware I'm on. That screen isn't in the packet, so I'm trusting the label.
- "open any website": I'd want it to name a plain-HTTP site. HSTS sites often won't trigger a portal, and "any website" will fail for people who type their usual HTTPS bookmark.

Severity: **low**.

## Task 9 — Hotel Ethernet with a sign-in page

**Clicks:** 17 shows "Ethernet needs you to sign in — GL's Login Mode covers the repeater Wi-Fi only; tailbond handles sign-in pages on other connections like this one. … Opening the sign-in page lets devices here reach Ethernet directly (without your home server) for up to 10 minutes" → **Open sign-in page** → 17b "Signing in to Ethernet… For up to 9 more minutes (until 05:08 on this router's clock), all traffic from devices here goes out Ethernet directly, not just the sign-in page" with **I've signed in, check now** / **Stop sign-in mode**.

**What worked:** a time limit with a clock time, a way to cancel, a "check now" button, and a plain statement of where traffic goes. The split between GL's Login Mode (Wi-Fi) and tailbond's own sign-in handling (other links) is explained in one sentence.

**Where I hesitate:** this is a design concern, not a usability one. In this scenario the Repeater is healthy and bonded ("In use ↓36 Mbps"). Even so, for up to 10 minutes every device's traffic goes out an untrusted hotel Ethernet, and the NAS becomes unreachable, all so one browser tab can load a portal. I'd rather only the portal's HTTP or DNS went direct, or only the device that clicked the button. Because it's disclosed, timed and cancellable, I'd accept it.

Severity: **low**.

## Task 10 — Home side set up for best speed

**Flint (18/18b/18c):** the "Setup health" card is the right tool. 18c is green: "Tailscale: 3 machines direct", "Route 10.77.0.0/24 approved", "Public internet address 203.0.113.7", "UDP 443 reached from the internet — pixel-8 connected from 198.51.100.20, 2 h ago". 18 explains the relay problem properly: "Through a relay: apple-tv. Only traffic between this machine and that machine is slower … bonding itself doesn't go through Tailscale. To check: tailscale ping apple-tv ('via DERP' means relayed)." The fix it gives is "On this router: NETWORK → Firewall → Open Ports on Router, add UDP 41641. If another router sits in front of this one … also forward UDP port 41641". The Port row for 443 handles the double-NAT case well ("If it's plugged into another box … that box must forward UDP 443 to this router").

**Fedora (15):** "Setup health: 2 things to improve" → "On the router in front of this machine, forward UDP port 41641 to it. (Turning on UPnP on that router also works, but lets any device on your network open ports.)" and "This machine's own firewall is already set up by tailbond." The UPnP caveat is exactly right for me.

**Where I hesitate:**
1. **CGNAT advice (18b), medium.** The text is "If your line has IPv6, enter this machine's public IPv6 address as the internet address (the router's status page shows it)." Two problems:
   - With an IPv6-only address, a travel link can reach home only if that link has IPv6. Lots of hotel Wi-Fi and some tethering is IPv4-only, and those links would quietly fail to join the bond. The screen doesn't warn about that.
   - Pasting a raw WAN IPv6 address breaks whenever the ISP changes my prefix. Step 2 in 13 already points to Dynamic DNS for IPv4, and an AAAA record there would be the durable fix.

   I'd follow this advice, think CGNAT was solved, and find out at the next IPv4-only hotel. The other way out offered is "ask your provider for a public IPv4 address", and no further alternative is shown.
2. On the Flint, UDP 443 is "opened on this router automatically", but UDP 41641 needs a manual trip to "NETWORK → Firewall → Open Ports on Router". Why the difference? Also, the green tick "Network is friendly to direct connections" has the sub-line "no port mapping on the router", which reads like a failure under a pass.
3. The Fedora page shows only its two failing checks. There's no "Show N more checks" like the Flint has, so I can't confirm "UDP 443 reached from the internet" for the Fedora box. The screens I have never tell me to forward UDP 443 to it (the header just says "reachable at home.example.net:443"). That's the forward bonding actually needs, while 41641 only improves Tailscale.
4. 13: "Step 1 … tick the one route listed (10.77.0.0/24)". The prefix is fixed and I see no way to change it. If it overlaps one of my existing subnet routes, I'm stuck. There's also no hint that a tailnet-policy `autoApprovers` entry would do the same job without clicking.

Severity: item 1 **medium**; items 2–4 **low**.

---

## Findings

| # | task | finding | severity |
|---|---|---|---|
| 1 | 10 | CGNAT fix "enter this machine's public IPv6 address as the internet address" doesn't warn that IPv4-only hotel/tethering links then can't reach home, and a raw WAN IPv6 address breaks when the prefix changes (DDNS AAAA not suggested) | medium |
| 2 | 1 | "Download the router package" doesn't say which arch/GL models it fits; arch only visible in LuCI filename `…aarch64_cortex-a53_neon-vfpv4.ipk`; Fedora step 2 lacks Flint's "leave the other options as they are" | low |
| 3 | 2 | Travel page: home server "must be signed in to the same Tailscale account as this router" vs home page: "Others on your Tailscale network … can ask to connect" — inconsistent prerequisite | low |
| 4 | 3 | Per-link rows show rate + latency but no loss (Fedora path cards show "0.0%"); only Ethernet shows interface name; 06 gives no hint why a link "can't reach homebox" | low |
| 5 | 4 | "Used only if your other connections stop working": failover criterion and speed undefined; interaction with "Most reliable — Sends everything over every connection" only inferred | low |
| 6 | 5 | Travel page doesn't say laptops reach the NAS "as homebox, under homebox's access rules" (only home pages do); bare short names not supported ("Use the full name") | low |
| 7 | 6 | When off, unclear whether tailnet names still work via the Slate's own Tailscale; 08 cards say "While it's off…" with toggle on; "Ask to connect again" shown while already connected | low |
| 8 | 7 | Flint approval text ("reach your Tailscale machines the way this router does") omits that the device gets home internet egress + home LAN, which Fedora states; Fedora's "by default" implies a split-tunnel option not visible anywhere | low |
| 9 | 7 | Delegating approval differs: Fedora has "Also allowed to approve (Tailscale logins)", Flint only "set [a password] to let others approve" | low |
| 10 | 8 | Button name hedged ("'Login Mode for Public Hotspots' (or 'Go to Login')"); "open any website" should name a plain-HTTP site so HSTS doesn't block the portal | low |
| 11 | 9 | Sign-in mode sends "all traffic from devices here … not just the sign-in page" out hotel Ethernet for up to 10 min and cuts tailnet access, even while Repeater bond is healthy (disclosed, timed, cancellable) | low |
| 12 | 10 | Flint opens UDP 443 "automatically" but UDP 41641 needs manual "NETWORK → Firewall → Open Ports on Router"; green tick "Network is friendly…" with sub-line "no port mapping on the router" reads as a failure | low |
| 13 | 10 | Fedora health shows only failing checks (no "Show more checks"); no visible confirmation or instruction for forwarding UDP 443 to the Fedora box, while 41641 is instructed | low |
| 14 | 10 | Fixed route "10.77.0.0/24" with no visible way to change it if it clashes with existing subnet routes; no `autoApprovers` hint for the manual route approval | low |

Totals: 0 high / 1 medium / 13 low
