# Round 8 — GL persona B (homelab)

Persona: homelab power user. I run Tailscale subnet routers, OpenWrt and a Fedora server at home. I want to know what the thing actually does: routes, DNS, direct vs DERP, NAT, who can approve, where traffic goes during captive-portal sign-in. I'd rather not SSH into a travel router from a hotel.

Materials: only `task-script.md`, `screens.md` and the PNGs in `out/ux/final/round-8/packet-gl/`. I looked at the desktop versions of every screen, and the phone versions of 05 and 14.

Overall: this is the most honest consumer VPN UI I've used. It says where traffic goes in each state: bonded, sign-in, off, and server unreachable (fail-closed). It names direct vs relay per machine and gives me `tailscale ping` and port 41641 fixes. The one thing that would make me do something wrong is the Fedora page contradicting itself about who can approve devices.

---

## Task 1 — Install on the Slate 7 without SSH

**Clicks:** Fedora page (15) "Add a GL.iNet travel router (recommended)" → **Download the router package**. Then on the Slate: 01a SYSTEM → Advanced Settings → **Go To LuCI** ("LuCI is OpenWrt's own web interface. Use the same admin password to log in.", `http://192.168.8.1:8080/cgi-bin/luci`). Then 01b System → Software → **Upload Package…**, and LuCI's dialog "Really install the uploaded package?" shows `tailbond_0.1.0-r42_aarch64_cortex-a53_neon-vfpv4.ipk (6.1 MB)`. I leave "Allow overwriting conflicting package files" unticked and click **Install**. On 02 the GL panel now has VPN → tailbond.

The Flint 3 panel (11–13) has the same three steps, plus "(log in with the same password)" and "(leave the other options as they are). The router's internet keeps working."

**Friction:**
- The Fedora page's steps are shorter than the Flint's: "pick the file, Install." They leave out the "same password" and "leave the other options" hints. It's easy for me, but it's the weaker version of the same instructions.
- The Fedora box is presumably x86, but the page just says "Download the router package". It doesn't say which GL models or architecture it's for. The filename tells me it's `aarch64_cortex-a53`, which fits the Slate 7, but I only find that out in LuCI. The Flint's "It's the same tailbond this router runs" covers this on the Flint, but not on Fedora. (low)
- No checksum or signature is shown anywhere for the .ipk I'm side-loading as root on a router. That's fine since it comes from my own server, but I'd like a sha256 next to the download button. (low)

Severity: low. Doable without SSH.

## Task 2 — Pair with the home server

**Clicks:** 02 "Looking for your home server on your tailnet…" → 03 "Found your home server: homebox / homebox.tail1234.ts.net · 100.101.1.1" → **Ask to connect** → 04 "Waiting for approval on homebox… On homebox, tap **Approve** next to **slate7**. This router connects by itself once approved; you can close this page."

The failure states are good:
- 03b "Turn on Tailscale first" includes "Open Tailscale settings".
- 03c "Checked 4 online machines. On the home server, tailbond must be installed and running, and Tailscale on." That is real diagnostic info.

**Friction:**
- 04 tells me to open `http://homebox:8077`, a short name. The travel page (05) insists on "Use the full name, like nas.tail1234.ts.net." I know short names work on a device running Tailscale with MagicDNS, but the two pages read inconsistently. (low)
- After pairing, it isn't clear what job the Slate's own Tailscale still has. Is it only needed for discovery, or does it need to stay on? 05 just says "(not this router's own Tailscale)". (folded into the Task 3/5 notes)

Severity: none blocking.

## Task 3 — Hotel Wi-Fi (Repeater) + USB tethering, both used

**Clicks:** none. On 05 the hero says "Connected to homebox / Using Repeater (Wi-Fi) + Tethering (phone)", "↓ 59 Mbps ↑ 7.1 Mbps Up 2 h 13 min". Both cards show "In use":
- Repeater: "↓ 38 Mbps ↑ 4.2 Mbps · 24 ms"
- Tethering: "↓ 21 Mbps ↑ 2.9 Mbps · 48 ms"

38+21=59, so I can see it really aggregating. The Home server card says "Connects to home.example.net over the internet (UDP port 443)" and "This router's tunnel address 10.77.0.2". So the bond runs over the public internet to my home IP, not over Tailscale. Good to know, and it's stated. 06 shows a broken link clearly: "Not working right now … It's connected, but can't reach homebox. tailbond keeps trying; the other connections carry your traffic."

**Friction:**
- Where my laptop's ordinary internet traffic exits, and whose DNS it uses, is only implied. The subtitle says "one tunnel to your home server". 08 says "nothing goes around your home server" and 09 says "use the internet directly, without your home server", so I infer everything exits at home. But no bonded screen says "your internet traffic exits from your home IP" or which resolver LAN clients use (home/MagicDNS vs hotel). (low)
- The GL page shows no per-path loss. The Fedora page does ("#0 active · 31 ms · 0.0%"). When a hotel link is flaky I'd want loss here too. (low, polish)

Severity: low.

## Task 4 — Phone as backup only

**Clicks:** Tethering card → Use it → **Backup only** (07). The badge reads "Backup · saving data" and the text says "Used only if your other connections stop working. Until then it isn't used at all, so it uses no data." The hero reads "Tethering (phone) on standby". The Speed card hint "Saving your phone's data? Set Tethering or Cellular to "Backup only"" pointed me straight there. **Change limit** (07b) gives "5 GB a month", and "At the limit, tailbond stops using this connection until the count restarts on the 1st of each month. 0 = no limit." 09 also warns "Backup only and data limits here apply only while tailbond is on." Good.

**Where I'd get stuck:**
- "it isn't used at all, so it uses no data" doesn't square with the limit help, "Counts everything **tailbond** sends and receives over it". That wording tells me tailbond only controls its own tunnel. The Slate's own traffic isn't covered: its own Tailscale (which is on), GL cloud, NTP, and GL's own multi-WAN choice for router-originated packets. Nothing says where that traffic goes while the phone is "on standby". I also can't tell how tailbond notices that the hotel Wi-Fi has failed without sending probes over the phone. As someone paying per GB, I can't confirm the "no data" claim from the UI. I'd end up SSHing in to check `ip route` / `mwan3`, which is exactly what I wanted to avoid. **(medium)**
- How "Most reliable — Sends everything over every connection. Uses the most data." interacts with a link set to Backup only isn't stated. I assume Backup only wins. (low)

Severity: medium.

## Task 5 — Reach the NAS by MagicDNS name

**Clicks:** 05 → Tailnet machines → "nas / nas.tail1234.ts.net / 100.101.2.2 / linux · Online · direct" → **Copy name**. The section explains: "Laptops and phones on this router's Wi-Fi or LAN reach these through the bond to homebox (not this router's own Tailscale), by their full name … or by their 100.x address. "Direct" / "via relay" is how homebox reaches each machine." The name is `nas.tail1234.ts.net`.

**Friction:**
- "Use the full name, like nas.tail1234.ts.net." Short `nas` won't work from the laptop. I'd normally rely on the MagicDNS search domain, so this is a small surprise, but it's stated. (low)
- Coming from subnet routers, I'd next ask whether the laptop can reach my home LAN (192.168.x behind homebox) or subnets advertised by my other subnet routers. Neither the page nor the approval text ("reach your Tailscale machines the way this router does") mentions subnet routes. It's not a failure I can see, just an unanswered question. (low)

Severity: low.

## Task 6 — Turn tailbond off

**Clicks:** the switch at the top right of the hero card (05). 09 then shows "tailbond is off — Devices on this router use the internet directly, without your home server, over whichever connection GL's own settings pick (INTERNET page). Backup only and data limits here apply only while tailbond is on." Each link card repeats that its choice "applies when tailbond is on". Unpair is separate and confirmed (09b: "This router stops using tailbond and forgets the home server … it will need to approve"). 08 offers "Turn tailbond off" in context, and 10 explains "Starting it is safe".

**Friction:** the hero switch has no label ("On/Off" or "tailbond") on desktop or phone. I found it, but it's the only control on the page without a word next to it. (low)

Severity: low.

## Task 7 — Approve on the home side

**Flint 3 (11):** "Waiting for your approval · 1 new / slate7 wants to connect / Travel router on your tailnet · signed in as alex@example.com / GL-BE3600 · linux · 100.101.9.9 · asked just now" → **Approve**. 12 then shows "slate7 Connected just now · 2 connections bonded". The text under it is exactly what I want: "Who can approve: anyone logged in to this admin panel, and alex@example.com … at http://homebox:8077. People you share this router with, and others on your Tailscale network, can ask to connect but can't approve. Approving lets a device reach your Tailscale machines the way this router does (they see it as this router, under its access rules)." This lines up with "tailbond page password: None: only alex@example.com's devices can use http://homebox:8077".

**Fedora (14/15):** **Approve** in the "Waiting for your approval" card. But the same page gives two different answers about who can approve:
- In the approval card (14, 15 top): "Who can approve: **anyone on your Tailscale network who can reach homebox** (it's a tagged machine, so it has no single owner). Set a password below to limit that."
- In the "Who can approve and add devices" card (15): "Who can approve: **alex@example.com** (the Tailscale account homebox belongs to), from any of their devices. People you share homebox with, and others on your Tailscale network, can ask to connect but can't approve."

Those can't both be true. If the first one is right, then with "Password for this page (empty: none)" anyone who can reach homebox:8077 can approve a device. That device then gets homebox's tailnet identity and ACLs ("they see it as homebox"), which is privilege escalation in a shared tailnet. My homebox *is* a tagged node, so the first statement is the one I'd worry about. But the second card is the one under the password field, and it tells me nothing needs fixing. I'd likely leave the password empty and be wrong. **(high)**

Severity: high (Fedora); Flint fine.

## Task 8 — Hotel Wi-Fi with a sign-in page (GL repeater detected it)

**Clicks:** 16 "Repeater needs you to sign in — GL's Repeater page handles this sign-in. Open it and click "Login Mode for Public Hotspots" (or "Go to Login")…" → **Open GL's Repeater page** → start Login Mode (GL's own page isn't in the packet; 16b is the result). 16b: "Router connected to homebox · devices signing in / Signing in on Repeater (Wi-Fi): all traffic from devices here goes out Repeater (Wi-Fi) directly, not through homebox, until you're signed in." The banner adds "so your phone's data isn't used for them. Only this router's own link to your home server stays up over your other connections (the top card; it carries almost nothing meanwhile). Login Mode ends by itself once you've signed in (or click "Quit Login Mode" …); tailbond then adds Repeater back to bonding."

That answers my exact question about where traffic goes during captive-portal sign-in. The Tailnet machines note even says devices "don't reach these until that's done".

**Friction:** the Ethernet sign-in (Task 9) has a hard cap ("up to 10 minutes", "until 04:57 on this router's clock"). Repeater Login Mode has no stated cap. If I never finish signing in, all LAN devices keep going out the hotel Wi-Fi unprotected, maybe indefinitely. It's GL's mode, but tailbond is the one telling me about it. (low)

Severity: low.

## Task 9 — Hotel Ethernet with a sign-in page

**Clicks:** 17 "Ethernet needs you to sign in — GL's Login Mode covers the repeater Wi-Fi only; tailbond handles sign-in pages on other connections like this one … Opening the sign-in page lets devices here reach Ethernet directly (without your home server) for up to 10 minutes" → **Open sign-in page**. 17b: "For up to 9 more minutes (until 04:57 on this router's clock), all traffic from devices here goes out Ethernet directly … The router's own tunnel stays up on your other connections." → **I've signed in, check now** (or **Stop sign-in mode**). Meanwhile Repeater stays "In use" and Tethering stays "Backup · saving data".

Clear, time-boxed, and it states the scope ("all traffic from devices here"). I could finish this without guessing.

Severity: none.

## Task 10 — Best speed at home

**Flint 3:** "Setup health":
- 18c "All good / Everything is set up for best speed": Tailscale: 3 machines direct; Network is friendly to direct connections (port mapping: UPnP); Route 10.77.0.0/24 approved; Public internet address 203.0.113.7; UDP 443 reached from the internet (pixel-8 connected from 198.51.100.20, 2 h ago).
- 18 "2 to improve" gives the real fix: "Through a relay: apple-tv … bonding itself doesn't go through Tailscale. To check: tailscale ping apple-tv ("via DERP" means relayed). On this router: NETWORK → Firewall → Open Ports on Router, add UDP 41641. If another router sits in front of this one … also forward UDP port 41641". apple-tv is also flagged "via relay (slower)" in the machine list. Excellent.
- 13 walks through route approval ("open homebox, choose Edit route settings, tick the one route listed (10.77.0.0/24)") and the internet name ("turn on APPLICATIONS → Dynamic DNS … Right now they'd use 192.168.8.1:443, which only works at home").

**Fedora (15):** the same two Tailscale items, plus "forward UDP port 41641 to it, and allow 41641/udp in this machine's firewall. (Turning on UPnP on that router also works, but lets any device on your network open ports.)"

**Where I'd get stuck:**
- **CGNAT (18b):** "Your internet line uses carrier-grade NAT — This machine's internet address 100.72.3.4 is shared by your provider, so travel routers can't reach UDP 443 here. Ask your provider for a public IPv4 address (or use public IPv6)… from away they can't until this is fixed." For a CGNAT user this is a dead end. Both ends are already on my tailnet, yet no fallback over Tailscale or a relay is offered. "or use public IPv6" has no how: the Internet address field shows `home.example.net:443`, and nothing says an IPv6 address or AAAA name is accepted. It's honest, but I can't act on it except by phoning my ISP. **(medium)**
- The two home pages give different advice on UPnP. The Flint shows "port mapping: UPnP" as a green check, while Fedora warns UPnP "lets any device on your network open ports". (low)
- Fedora says "the guide fedora-server.md has a section per item". That's a bare filename, not a link, and I don't know where that file lives. The "How to fix" links may cover it. (low)
- `10.77.0.0/24` is fixed. Nothing shows how to change it if it overlaps one of my subnets. "That lets your Tailscale machines reach devices that connect through tailbond" also doesn't say whether that means just tunnel addresses (10.77.0.x) or the travel router's LAN behind them. For someone running subnet routers, that's the first thing I'd want to know before ticking the route. (low)

Severity: medium (CGNAT), rest low.

---

## Findings

| # | Task | Finding | Severity |
|---|---|---|---|
| 1 | 7 | Fedora page contradicts itself about who can approve: approval card says "anyone on your Tailscale network who can reach homebox (it's a tagged machine…)", while the "Who can approve and add devices" card says only "alex@example.com … others on your Tailscale network … can't approve". With "Password for this page (empty: none)" I can't tell if approval is open to the whole tailnet, and an approved device gets homebox's ACLs. | high |
| 2 | 4 | "it isn't used at all, so it uses no data" vs "Counts everything tailbond sends and receives over it": nothing says whether the Slate's own traffic (its own Tailscale, GL services, GL multi-WAN) or failover probing uses the phone while on standby. I can't verify zero phone data without SSH. | medium |
| 3 | 10 | CGNAT: "Ask your provider for a public IPv4 address (or use public IPv6)" is the only path. No fallback over the tailnet both ends already share, and no how-to for "use public IPv6". | medium |
| 4 | 1 | Fedora page's "Download the router package" doesn't say which GL models or architecture it fits (Fedora is not the same arch). Its steps also omit "same password" and "leave the other options". | low |
| 5 | 1 | No checksum or signature shown for the side-loaded .ipk. | low |
| 6 | 2 | Waiting screen uses short `http://homebox:8077`, while the travel page says "Use the full name". Reads inconsistent. | low |
| 7 | 3 | Bonded screens never state outright that internet traffic exits from home, or which DNS resolver LAN clients use. No per-path loss on GL (Fedora shows "0.0%"). | low |
| 8 | 4 | Interaction between "Most reliable — Sends everything over every connection" and a Backup-only link isn't stated. | low |
| 9 | 5 | Short names don't work ("Use the full name"). No word on whether home LAN / other subnet routes are reachable. | low |
| 10 | 6 | Hero on/off switch has no text label. | low |
| 11 | 8 | Repeater Login Mode has no stated time cap ("ends by itself once you've signed in"), unlike Ethernet's "up to 10 minutes". All LAN traffic could stay outside the tunnel indefinitely. | low |
| 12 | 10 | Flint shows "port mapping: UPnP" as a green check while Fedora warns UPnP "lets any device on your network open ports". | low |
| 13 | 10 | "the guide fedora-server.md has a section per item": bare filename, no link or location. | low |
| 14 | 10 | Route 10.77.0.0/24 can't be changed if it collides, and "reach devices that connect through tailbond" doesn't say whether that includes the travel router's LAN. | low |

Totals: 1 high / 2 medium / 11 low
