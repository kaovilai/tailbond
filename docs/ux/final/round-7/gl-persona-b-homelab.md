# Round 7 — GL persona B (homelab)

Persona: homelab power user. I run Tailscale subnet routers, OpenWrt and a Fedora server at home. I want to know what actually happens: routes, DNS, direct vs relayed, NAT, who can approve, and where traffic goes during captive-portal sign-in. I'd rather not SSH into a travel router from a hotel.

Materials: `task-script.md`, `screens.md` and the PNGs in packet-gl. I looked at the desktop versions of every screen and at the phone versions of 05 and 17b. Nothing else.

Overall: this is the most honest travel-router UI I've seen on these points. It names the transport ("`home.example.net` over the internet (UDP port 443)"), says that tailnet access goes through homebox and not through the Slate's own Tailscale, says which machines are "direct" or "via relay", says that it fails closed when home can't be reached, and says plainly that during sign-in "all traffic from devices here goes out … directly". I could finish every task. My complaints are about the home-side health advice and about who is allowed to approve.

---

## Task 1 — Install on the Slate 7 without SSH

**Clicks:** Fedora page (15) → Devices tab → "Add a GL.iNet travel router (recommended)" → **Download the router package**. Or on the Flint (11–13), "Add a GL.iNet travel router" → **Download the router package**. Then on the Slate: 01a SYSTEM → Advanced Settings → **Go To LuCI** ("Use the same admin password to log in"). Then 01b LuCI System → Software → **Upload Package…** → pick `tailbond_0.1.0-r42_aarch64_cortex-a53_neon-vfpv4.ipk (6.1 MB)` → **Install**.

**Works.** No SSH needed. The Flint's step 2 says "(leave the other options as they are)" and "The router's internet keeps working". Those are the two things I'd have wondered about.

**Friction:**
- The download button doesn't say which routers or architectures the package fits. I only find out from the filename in LuCI (`aarch64_cortex-a53`). I happen to know that suits the BE3600, but someone with an older MIPS GL box finds out only when opkg refuses. The Flint's "It's the same tailbond this router runs" hints that it's built for the Flint's own architecture, and the Fedora box is x86, so which build does the Fedora page hand out? (low)
- The Fedora page's step 2 ends "pick the file, Install." It leaves out the Flint's "(leave the other options as they are)", and LuCI shows an unchecked "Allow overwriting conflicting package files" box. (low)

## Task 2 — Pair the Slate 7 with the home server

**Clicks:** 02 "Looking for your home server on your tailnet…" → 03 "Found your home server: homebox / homebox.tail1234.ts.net · 100.101.1.1" → **Ask to connect** → 04 "Waiting for approval on homebox… On homebox, tap Approve next to slate7".

**Works.** The other states are handled well. 03b says "Turn on Tailscale first" and has **Open Tailscale settings**. 03c says "Checked 4 online machines", which is exactly the diagnostic I want. 04 says "From anywhere: open http://homebox:8077 on any device signed in to your Tailscale account".

**Friction:**
- "must be signed in to the same Tailscale account as this router." My server nodes are tagged (`tag:server`), so they aren't "signed in" as a user. The Fedora page also says sharees count ("anyone you share this machine with"). Does a tagged home server or a shared node work? The UI doesn't say. (low)

## Task 3 — Hotel Wi-Fi (Repeater) + USB tethering, confirm both are used

**Clicks:** 05 hero: "Connected to homebox · Using Repeater (Wi-Fi) + Tethering (phone)", ↓59 Mbps. Repeater card: "In use · ↓38 Mbps ↑4.2 Mbps · 24 ms". Tethering card: "In use · ↓21 Mbps ↑2.9 Mbps · 48 ms".

**Works.** I can see both links, each with its own throughput and latency, and the per-link figures add up to the total. 06 handles a bad link well: "It's connected, but can't reach homebox. tailbond keeps trying; the other connections carry your traffic."

**Friction:**
- 08 (server unreachable) is a good fail-closed message: "devices on this router have no internet, so nothing goes around your home server." But the Repeater and Tethering cards on that same screen say "Ready · Used when tailbond is on." while the tailbond toggle is on. For a moment I thought the links weren't enabled. (low)

## Task 4 — Phone as backup only

**Clicks:** 05 Tethering → Use it: **Backup only** → 07 "Backup · saving data" and hero "Tethering (phone) on standby". Optionally **Change limit** → 07b "5 GB a month" → **Save**.

**Works.** It's also easy to find, because the "Speed or reliability" card says "Saving your phone's data? Set Tethering or Cellular to 'Backup only'".

**Friction:**
- "Until then it isn't used at all, so it uses no data." Then how does tailbond know the phone link is healthy enough to fail over to, and how long does failover take? I'd expect some probe traffic and a failover time. "No data at all" reads like marketing to me. (low)
- 07b: "Counts everything tailbond sends and receives over it … until the count restarts on the 1st of each month." My carrier's billing cycle doesn't start on the 1st. Also, traffic the router itself sends over the phone (GL's own Tailscale, firmware checks) apparently isn't counted, so the real usage will be higher than the bar shows. (low)

## Task 5 — Reach the NAS by MagicDNS name from the laptop

**Clicks:** 05 → "Tailnet machines" → **nas** `nas.tail1234.ts.net` / `100.101.2.2` / "linux · Online · direct" → **Copy name**.

**Works.** The page also explains the routing: "Laptops and phones on this router's Wi-Fi or LAN reach these through the bond to homebox (not this router's own Tailscale) … 'Direct' / 'via relay' is how homebox reaches each machine." Home server card: "Tailnet names — Work on every device connected to this router while tailbond is on. Use the full name, like nas.tail1234.ts.net." So short names won't work, and it says so.

**Friction:**
- The page never says where DNS for ordinary names is resolved while tailbond is on: at home through the tunnel, or by the hotel's resolver. It's implied by "one tunnel to your home server", but I care about it. (low)

## Task 6 — Turn tailbond off

**Clicks:** 05 hero toggle → 09 "tailbond is off — Devices on this router use the internet directly, without your home server." Unpair is separate and has a clear confirm dialog (09b): "This router stops using tailbond and forgets the home server … it will need to approve".

**Works.**

**Friction:**
- With tailbond off, can the laptop still reach the NAS through the Slate's own GL Tailscale? The only hint is "Tailnet machines — The list appears once tailbond is connected." (low)

## Task 7 — Approve the travel router at home

**Clicks:** Flint 11 "Waiting for your approval · 1 new — slate7 wants to connect · Travel router on your tailnet · signed in as alex@example.com · GL-BE3600 · linux · 100.101.9.9 · asked just now" → **Approve** → 12 slate7 appears under Devices: "Connected just now · 2 connections bonded". Fedora 14/15 has the same card with **Approve**.

**Works.** The request shows account, model and 100.x address, so I can check it's really my Slate.

**Friction:**
- **Who can approve is broad by default, and the two home servers describe it differently.** Fedora (14/15): "anyone who can open this page (people on your Tailscale account, and anyone you share this machine with)". Flint (11): "anyone logged in to this admin panel, and anyone on your Tailscale account who opens http://homebox:8077". So on Fedora, someone I only shared homebox with can approve an arbitrary device. Because homebox advertises `10.77.0.0/24`, that device then reaches my whole tailnet ("your Tailscale machines reach them" / devices reach nas, apple-tv…). That is more than a sharee normally gets. The UI lists who can approve but not what an approval grants, and there's no password by default ("None: anyone on your Tailscale account can open http://homebox:8077"). I'd expect the consequence to be spelled out next to the "Set" password control, and the two servers to agree on whether sharees count. (medium)
- Fedora 14 shows "1.1 MBPS DOWN / 12.0 MBPS UP" from the server's point of view. The Flint makes the direction clear with "Sending to devices 61 Mbps / Receiving 7.4 Mbps". Fedora doesn't. (low)

## Task 8 — Hotel Wi-Fi sign-in page (GL's Login Mode) with tailbond on

**Clicks:** 16 hero "Using Tethering (phone) · Repeater (Wi-Fi) needs sign-in". Banner "Repeater needs you to sign in — GL's Repeater page handles this sign-in. Open it, click 'Login Mode for Public Hotspots' (or 'Go to Login')" → **Open GL's Repeater page** → (GL's page isn't in the packet; I followed the quoted instruction) → 16b "Router connected to homebox · devices signing in". It ends "by itself once you've signed in (or click 'Quit Login Mode' on GL's Repeater page); tailbond then adds Repeater back to bonding."

**Works.** It answers my main question exactly: "all traffic from devices here goes out the hotel Wi-Fi directly (not just the sign-in page), not through your home server". I knew before clicking that my traffic would leave unprotected.

**Friction:**
- 16b: "The router itself stays connected to your home server over your other connections (that's what the top card shows), so they keep being used." Tethering is "Always · In use" at ↓21 Mbps, but device traffic bypasses the tunnel. So what is the phone carrying, and is it burning data for nothing? The "they" is also ambiguous. The Tailnet machines list still shows green dots and "reach these through the bond", even though during this window device traffic doesn't go through the bond, so the NAS presumably isn't reachable. (low)

## Task 9 — Hotel Ethernet sign-in page

**Clicks:** 17 hero "Using Repeater (Wi-Fi) · Ethernet needs sign-in · Tethering (phone) on standby". Banner "Ethernet needs you to sign in — GL's Login Mode covers the repeater Wi-Fi only; tailbond handles sign-in pages on other connections like this one." → **Open sign-in page** → 17b "For up to 9 more minutes (until 04:40 on this router's clock), all traffic from devices here goes out Ethernet directly, not just the sign-in page … tailbond adds Ethernet back by itself once you're signed in." There are also **I've signed in, check now** and **Stop sign-in mode**.

**Works.** It has a bounded window, an explicit bypass statement, a manual re-check and an abort. Exactly what I'd want.

**Friction:**
- Before clicking, 17 only says "lets devices here reach Ethernet directly (without your home server) for up to 10 minutes, so the page can load." The words "all traffic … not just the sign-in page" appear only after clicking (17b). The Wi-Fi flow (16) is clearer up front. (low)

## Task 10 — Home side set up for best speed

**Clicks:** Flint panel → "Setup health". 18c "All good": "Tailscale: 3 machines direct / Network is friendly to direct connections (port mapping: UPnP) / Route 10.77.0.0/24 approved / Public internet address 203.0.113.7 / UDP 443 reached from the internet (pixel-8 connected from 198.51.100.20, 2 h ago)". That's exactly the checklist I'd run by hand. On a fresh Flint, 13 gives "Step 1: allow tailbond devices… Edit route settings, tick the one route listed (10.77.0.0/24)" with **Open admin console**, and "Step 2: tell travel routers where home is" (DDNS). Fedora: 15 Devices tab → "Setup health: 2 to improve for best speed".

**Friction:**
- **Flint relay fix (18):** "Tailscale: 2 machines direct, 1 through a relay (slower) … On this router: NETWORK → UPnP, turn it on." On the Flint, the home server is the edge router. Turning on its UPnP *server* lets LAN clients open ports on it, but I don't see how that helps the Flint's *own* Tailscale traffic. The fix I'd expect is "allow UDP 41641 in on the WAN firewall", or a forward on the upstream box, which the text does mention for that case. Turning on UPnP router-wide also has a security cost that isn't mentioned. I'd likely flip it, see no change, and have widened my LAN's attack surface. (Fedora 15 phrases it correctly: "forwarding UDP port 41641 to this machine, or by turning on UPnP on the router in front of it.") (medium)
- **CGNAT is labelled as a speed issue (18b).** The setup card at the top is correct and red: "Step 1: Your internet line uses carrier-grade NAT … travel routers can't reach UDP 443 here … from away they can't until this is fixed." But the Setup health card on the same page says "1 thing to improve for best speed (tailbond works meanwhile, just slower)". That's wrong in this case: away from home it doesn't work at all. One card says blocker, the other says slowdown. (medium)
- 18b, CGNAT fix: "Ask your provider for a public IPv4 address (or use public IPv6)." Nothing on screen shows how to use IPv6: whether the Internet address field or the DDNS name publishes an AAAA record, or whether "UDP 443, opened on this router" covers v6. Also, the whole tailnet is already right there and every device is on it, but the only fix offered goes through the ISP. Many homelabbers are on CGNAT lines (5G home internet, Starlink). (low)
- 18 (relay): "2 things to improve for best speed". The bond tunnel runs over public UDP 443, not over Tailscale. So a relayed Tailscale path only slows traffic between tailbond devices and *that* machine (here pixel-8), not the bond itself. The Slate's own wording ("how homebox reaches each machine") gets this right. The Flint card makes it sound like the whole tunnel is slower. (low)
- 13 Step 2: "Right now they'd use 192.168.8.1:443, which only works at home." 192.168.8.1 is also every GL router's default LAN address, including the Slate's own. So "works at home" isn't even reliable while both routers are on GL's default subnet. (low)
- Fedora 15 lists only failing checks. Unlike the Flint's "Show 3 more checks", I can't see that "Route 10.77.0.0/24 approved" or "UDP 443 reached from the internet" passed. (low)

---

## Findings

| # | Task | Finding | Severity |
|---|---|---|---|
| 1 | 7 | Default approvers are broad, and the two home servers disagree: Fedora "anyone you share this machine with" can approve; Flint text omits sharees. No statement that an approved device reaches the whole tailnet via 10.77.0.0/24; no password by default | medium |
| 2 | 10 | Flint relay fix "On this router: NETWORK → UPnP, turn it on" likely doesn't help the edge router's own Tailscale and widens attack surface; expected "allow UDP 41641 in on WAN" | medium |
| 3 | 10 | 18b CGNAT: setup card says travel routers "can't reach" from away, but Setup health says "tailbond works meanwhile, just slower" | medium |
| 4 | 1 | "Download the router package" doesn't say which architecture or models it fits; only the LuCI filename shows `aarch64_cortex-a53` | low |
| 5 | 1 | Fedora install step lacks the Flint's "(leave the other options as they are)" next to LuCI's "Allow overwriting conflicting package files" | low |
| 6 | 2 | "signed in to the same Tailscale account": unclear for tagged server nodes or shared nodes | low |
| 7 | 3 | 08: link cards say "Ready · Used when tailbond is on." while tailbond is on | low |
| 8 | 4 | "isn't used at all, so it uses no data": no word on health probing or failover time | low |
| 9 | 4 | Data limit resets "on the 1st of each month" only and counts only tailbond's traffic | low |
| 10 | 5 | Where ordinary DNS is resolved while tailbond is on (home or hotel) is never stated | low |
| 11 | 6 | When tailbond is off, unclear whether the laptop still reaches the NAS via the Slate's own Tailscale | low |
| 12 | 8 | 16b: Tethering "In use" at 21 Mbps while devices bypass; "so they keep being used" is ambiguous; Tailnet list still claims reach "through the bond" | low |
| 13 | 9 | 17 (before clicking) doesn't say "all traffic, not just the sign-in page"; only 17b does | low |
| 14 | 10 | CGNAT fix "(or use public IPv6)" has no visible way to do it; the only path offered is the ISP | low |
| 15 | 10 | Relay item framed as slowing "best speed" overall, though the bond runs over UDP 443; it only affects traffic to the relayed machine | low |
| 16 | 10 | 13: fallback "192.168.8.1:443 … only works at home" collides with GL's default LAN IP on the Slate itself | low |
| 17 | 10 | Fedora health shows only failing checks; can't confirm route approved or UDP 443 reached | low |
| 18 | 7 | Fedora 14 shows server-relative "1.1 MBPS DOWN / 12.0 MBPS UP" with no direction cue (Flint says "Sending to devices") | low |

Totals: 0 high / 3 medium / 15 low
