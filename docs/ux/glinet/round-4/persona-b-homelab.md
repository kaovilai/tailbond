# Round 4 — Persona B (homelab power user)

Persona: runs Tailscale subnet routers, OpenWrt and a Fedora server. Wants to see
what is actually happening: routes, DNS, direct vs DERP, NAT, how pairing is
secured, who can approve, where traffic goes during a captive-portal sign-in.
Would rather not SSH into a travel router at a hotel.

Inputs: task-script.md, screens.md, PNGs in out/ux/glinet/round-4/ (all desktop
variants; phone variants of 05, 14, 17b).

---

## Task 1: Install on the Slate 7 without SSH

**Clicks:** I got the package from the Fedora page (15, Devices tab, "Add a GL.iNet travel router (recommended)" → "Download the router package"). Then on the Slate 7: SYSTEM → Advanced Settings → "Go To LuCI" (01a; it shows the URL `http://192.168.8.1:8080/cgi-bin/luci` and "Use the same admin password to log in"). In LuCI: System → Software → "Upload Package…" → dialog "Really install the uploaded package?" with `tailbond_0.1.0-r42_aarch64_cortex-a53_neon-vfpv4.ipk (6.1 MB)` → "Install", leaving "Allow overwriting conflicting package files" unchecked (01b).

**What worked:** The steps on the home pages (11/12/15) match the screens step for step, and they say to leave the other options as they are. I'm used to opkg in LuCI, so this went smoothly. The arch string in the filename lets me confirm it's the right build.

**Friction:**
- Nothing shows a checksum or signature for the .ipk. It came over my own tailnet, so the risk is small, but I'd like a sha256 next to "Download the router package" to compare before I click Install. (low)

## Task 2: Pair with the home server

**Clicks:** VPN → tailbond (02 "Looking for your home server on your tailnet…") → 03 "Found your home server: homebox / homebox.tail1234.ts.net · 100.101.1.1" → "Ask to connect" → 04 "Waiting for approval on homebox… On homebox, tap Approve next to slate7."

**What worked:** Discovery shows the FQDN and the 100.x address, so I can see which node it picked. "Not your home server? Type its name instead" is there for when I have several candidates. 03c gives a useful count: "Checked 4 online machines." 03b explains that discovery depends on Tailscale. The pending request on the server shows "signed in as alex@example.com · GL-BE3600 · linux · 100.101.9.9", which is enough for me to check it's really my device before I approve.

**Friction:**
- After pairing, the Home server card says "Connects to `home.example.net` over the internet (UDP port 443)". So Tailscale only handles discovery and pairing, and the bonded tunnel runs over the public internet, not through tailscaled. That's fine, but the travel router never says how that tunnel is authenticated or encrypted. Only the Flint shows "Secure connection: Certificate from Tailscale for homebox.tail1234.ts.net". A one-line "Encrypted, server identity checked via …" on the Slate's Home server card would answer my first question. (low)

## Task 3: Hotel Wi-Fi + phone tethering, both in use

**Clicks:** 05: header "Using Repeater (Wi-Fi) + Tethering (phone)". Both cards say "In use" with per-link "↓ 38 Mbps ↑ 4.2 Mbps · 24 ms" and "↓ 21 Mbps ↑ 2.9 Mbps · 48 ms". 06 shows the failure case clearly: "It's connected, but can't reach homebox. tailbond keeps trying; the other connections carry your traffic."

**What worked:** Per-path throughput and latency on each card is exactly what I want. On the Flint (12) the device row reads "2 connections bonded".

**Friction:**
- The phone layout of 05 drops the "Up 2 h 13 min" uptime from the header. Minor. (low)

## Task 4: Phone as backup only

**Clicks:** Tethering card → "Use it" → "Backup only" (07). The badge changes to "Backup · saving data" and the text says "Used only if your other connections stop working. Until then it isn't used at all, so it uses no data." The header now says "Tethering (phone) on standby". Optionally "Change limit" (07b): "Counts everything tailbond sends and receives over it. At the limit, tailbond stops using this connection until the count restarts on the 1st of each month. 0 = no limit."

**What worked:** It's clear and it says plainly that the backup uses no data.

**Friction:**
- The limit only counts tailbond's own traffic. If the tether is also the router's default route when tailbond is off, I can't tell whether that traffic counts. "Counts everything tailbond sends and receives" implies it doesn't. I'd keep the phone's own data cap as a backstop. (low)

## Task 5: Reach the NAS by MagicDNS name

**Clicks:** 05 "Tailnet machines": `nas` / `nas.tail1234.ts.net` / `100.101.2.2` / "linux · Online · direct" → "Copy name". The Home server card says "Tailnet names: Work on every device connected to this router while tailbond is on. Use the full name, like nas.tail1234.ts.net."

**What worked:** I found the name immediately. Showing direct vs "via relay (slower)" per peer is a nice touch.

**Friction:**
- It doesn't say how the names resolve: the Slate's own tailscaled/100.100.100.100 or forwarded to homebox. It also doesn't say why the short name `nas` won't work (no search domain pushed to LAN clients?). "Use the full name" is enough to get the task done. Still, I'd like one line such as "DNS for *.ts.net is answered by this router's Tailscale". (low)
- When tailbond is off (09), "The list appears once tailbond is connected." But this router's own Tailscale is still on. Can laptops still reach the NAS through the Slate's Tailscale (subnet/exit settings), or not? It's unclear whether tailnet reachability depends on tailbond or on GL's Tailscale. (low)

## Task 6: Turn tailbond off

**Clicks:** The header toggle (05 → 09 "tailbond is off — Devices on this router use the internet directly, without your home server."). 08 also has "Turn tailbond off". "Unpair" is separate and asks for confirmation (09b: "To connect again, ask the home server again (it will need to approve)").

**What worked:** Fail-closed behavior is stated explicitly in 08: "Meanwhile, devices on this router have no internet, so nothing goes around your home server." That's exactly the leak guarantee I want to know about. 10 ("Starting it is safe: devices here keep their internet while you set it up") is also good.

No issues.

## Task 7: Approve the travel router on the home side

**Flint (11):** "Waiting for your approval / slate7 wants to connect" → "Approve". The card shows "Who can approve: anyone logged in to this admin panel, and anyone on your Tailscale account who opens http://homebox:8077. To require a password there, set one under Internet and Tailscale." The "tailbond page password: None … Set" row is there too. 12 confirms "slate7 Connected just now · 2 connections bonded".

**Fedora (14):** Approve at the top. "Who can approve: anyone who can open this page, i.e. anyone on your Tailscale account (and anyone you shared this machine with). To require a password, set "ui_password" in /etc/tailbond/tailbond.json".

**What worked:** It's honest about the approval trust model, which is what I asked for.

**Friction:**
- The two home sides describe the approver set differently. Fedora mentions "anyone you shared this machine with" and the Flint doesn't. Node sharing applies to the Flint too, so a Flint owner might think only their own account can approve. The default (no password, any tailnet member or shared-in user can approve a device into the bond) is a real exposure, so I'd want the wording identical on both. (low; the password control is right there)
- On Fedora, setting a password means editing a JSON file on the server. That's fine for me, but it's the one step that needs a shell. (low)
- Fedora phone (14-phone): when I open homebox:8077 on my phone to approve the router, a large "Add this phone … Download tailbond for Android" card fills the screen under the approval. Approve is still at the top, so it's only noise. The header also truncates to "reachable at home.···". (low)

## Task 8: Hotel Wi-Fi with a sign-in page (GL repeater)

**Clicks:** 16: banner "Repeater needs you to sign in — GL's Repeater page handles this sign-in. Open it, click "Login Mode for Public Hotspots" (or "Go to Login") … tailbond steps aside while you sign in, and adds Repeater back to bonding by itself afterwards. Your other connections keep working meanwhile." → "Open GL's Repeater page". Then 16b: Repeater shows "Signing in…" and "Devices here reach this network directly until you've signed in." Banner: "Signing in through GL's Login Mode now: tailbond is letting devices here reach the sign-in page directly."

**What worked:** It's clear that I use GL's own flow, and that the Repeater is kept out of the bond (so the bond isn't stalled).

**Friction:**
- The banner and the card describe the bypass differently. The banner says devices reach "the sign-in page directly". The card says "Devices here reach this network directly until you've signed in". Is only portal traffic going direct, or is all LAN traffic routed out the hotel Wi-Fi unencrypted while Login Mode runs? Fail-closed was promised in 08, so this matters to me. The Ethernet case (17/17b) gives a scope and a deadline ("for up to 10 minutes", "Until 04:06"), but the GL Login Mode case gives neither. No end time and no "Stop" button. I'd hold back from using it on a laptop with sensitive sessions open until I understood it. (medium)

## Task 9: Ethernet with a sign-in page

**Clicks:** 17: "Ethernet needs you to sign in … Opening the sign-in page lets devices here reach Ethernet directly (without your home server) for up to 10 minutes, so the page can load." → "Open sign-in page". 17b: "Signing in to Ethernet… Until 04:06, devices here use Ethernet directly, without your home server. tailbond adds it back by itself once you're signed in." Buttons: "Open the sign-in page again", "I've signed in, check now", "Stop".

**What worked:** This is the model I want: an explicit bypass scope, a time limit, a stop button, and a manual re-check.

**Friction:**
- "Until 04:06" has no timezone and doesn't say whose clock it uses (the router's time zone setting vs my laptop at a hotel in another zone). A countdown ("9 min left") would be unambiguous. (low)
- It says "devices here use Ethernet directly" (all devices, not just the portal), but it doesn't say what DNS they get during that window (the hotel's DHCP DNS?). (low)

## Task 10: Home side set up for best speed

**Flint healthy (12):** "Setup health — All good / ✓ Everything is set up for best speed": "Tailscale: 4 machines direct", "Network is friendly to direct connections — port mapping: UPnP", "Route 10.77.0.0/24 approved", "Public internet address 203.0.113.7", "UDP 443 reached from the internet — slate7 connected from 198.51.100.9, 2 h ago". These are exactly the checks I'd run by hand, and the route check matters to me as a subnet-router user.

**Friction:**
- **13 contradicts itself.** The header says "Home server is on · 2 setup steps left", and the setup health card shows two "?" items ("Behind your home router — Forward UDP 443 on that router to 192.168.8.1", "UDP 443: not confirmed from outside yet"). Yet the card headline says "✓ Everything is set up for best speed" and the badge reads "0 to improve". In that state I'd believe the green headline and think I was done, but Step 1 (route) and Step 2 (internet address) are still open above it. (medium)
- **18 lists the wrong relayed machines.** The health item says "Tailscale: 2 machines direct, 2 via relay (slower) — Via relay: slate7, work-laptop. Check with: tailscale ping slate7". The Tailnet machines list on the same page shows pixel-8 as "via relay (slower)", and work-laptop is offline ("Last seen 1 days ago"), so it can't be relaying at all. slate7 isn't in the list. I'd run `tailscale ping` against the wrong hosts and distrust the card. (medium)
- **18b contradicts itself.** The setup health says "Your internet line uses carrier-grade NAT … so travel routers can't reach UDP 443 here." Right above it, "Internet and Tailscale" still shows the green "Setup done — Travel routers and phones can connect from anywhere" with "UDP 443, opened on this router automatically". One of those is wrong, and the more severe one is in the lower card. It also leaves open whether pairing will work for me at all. A relay or Tailscale-carried fallback isn't mentioned, so with CGNAT I'm apparently stuck until I get a public IP. (medium)
- 18 "Hard NAT on this network … Forward UDP 41641 to this machine, or turn on UPnP/NAT-PMP on the router." When the Flint is the edge router, "this machine" and "the router" are the same box, and the advice doesn't say whether the Flint is behind another NAT (double NAT). A line with the WAN address vs public address would tell me which case I'm in. (low)
- **Fedora path:** the manifest says 15 includes setup health, but the Devices tab I can see (15) has no setup health card. It isn't on Connections (14) either. Maybe it's under the "Tailscale" tab, but no screenshot shows it, so on Fedora I couldn't check direct vs relayed, the route, or the port from what's shown. (medium)
- Fedora 15 "Invite someone's phone" link is `http://homebox.tail1234.ts.net:80/#add-phone`, while everywhere else the page is `http://homebox:8077`. That's two different ports/hosts for the same UI, so I'd wonder if something else is listening on :80. (low)
- Fedora 14 top stats read "1.1 MBPS DOWN / 12.0 MBPS UP", but the only device (pixel-8, 15) is "↓ 12.0 Mbps ↑ 1.1 Mbps". So the top numbers are from the server's point of view and the row is from the device's. The Flint labels this clearly ("Sending to devices / Receiving"), but Fedora doesn't. (low)

---

## Findings

| # | task | finding | severity |
|---|---|---|---|
| 1 | 1 | No checksum/signature shown next to "Download the router package" to verify the .ipk before LuCI install | low |
| 2 | 2 | Slate's Home server card says tunnel goes "over the internet (UDP port 443)" but not how it's authenticated/encrypted (only Flint shows "Secure connection") | low |
| 3 | 3 | Phone layout of 05 drops uptime ("Up 2 h 13 min") from header | low |
| 4 | 4 | Data limit counts only tailbond traffic; unclear what counts when tailbond is off | low |
| 5 | 5 | "Use the full name" with no hint how *.ts.net resolves for LAN clients or why short `nas` won't work | low |
| 6 | 5/6 | With tailbond off, "The list appears once tailbond is connected", so unclear whether NAS is still reachable via the Slate's own Tailscale | low |
| 7 | 7 | Approver wording differs: Fedora includes "anyone you shared this machine with", Flint omits it, though default (no password) lets any tailnet/shared user approve | low |
| 8 | 7 | Fedora password requires editing /etc/tailbond/tailbond.json (shell) | low |
| 9 | 7 | Fedora phone: large "Add this phone" card below the approval; header truncated "home.···" | low |
| 10 | 8 | GL Login Mode bypass scope unclear ("reach the sign-in page directly" vs card "reach this network directly") and no time limit/Stop, unlike Ethernet's "up to 10 minutes"/"Until 04:06"/"Stop" | medium |
| 11 | 9 | "Until 04:06" has no timezone/countdown; DNS during the bypass window not stated | low |
| 12 | 10 | 13: health headline "✓ Everything is set up for best speed" + "0 to improve" while "2 setup steps left" and two "?" items | medium |
| 13 | 10 | 18: "Via relay: slate7, work-laptop" contradicts the list (pixel-8 relayed, work-laptop offline, slate7 absent) | medium |
| 14 | 10 | 18b: CGNAT says "travel routers can't reach UDP 443 here" while the card above still says "Setup done … connect from anywhere"; no fallback mentioned | medium |
| 15 | 10 | 18 Hard NAT advice ("this machine" / "the router") doesn't distinguish Flint-as-edge vs double NAT | low |
| 16 | 10 | Fedora: setup health promised for 15 but not on Devices or Connections tab; can't verify direct/relay/route/port on Fedora from shown screens | medium |
| 17 | 10 | Fedora invite link `http://homebox.tail1234.ts.net:80/#add-phone` vs `http://homebox:8077` elsewhere | low |
| 18 | 10 | Fedora top stats (1.1 down / 12.0 up) are server-perspective, device row is device-perspective, unlabeled | low |
