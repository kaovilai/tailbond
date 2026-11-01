# Round 5, persona B: homelab power user (Tailscale subnet routers, OpenWrt, Fedora server)

What I bring: I know LuCI, opkg, MagicDNS, subnet routes, DERP vs direct, NAT types. I want to see where traffic actually goes, who can approve a pairing, and what leaves the tunnel during a captive-portal sign-in. I don't want to SSH into a travel router in a hotel.

Inputs: task-script.md, screens.md, and the round-5 PNGs (every desktop screen; phone versions of 05 and 17b).

---

## Task 1: Install on the Slate 7 without SSH

**Where I click:** On the Flint panel (11/12/13/18*), the "Add a GL.iNet travel router" card has **Download the router package**. On Fedora (15), the same card is marked "(recommended)". Then on the Slate: 01a SYSTEM → Advanced Settings → **Go To LuCI** (the note there says "Use the same admin password to log in"). Next, 01b LuCI System → Software → **Upload Package…**. The dialog says "Really install the uploaded package?" and shows `tailbond_0.1.0-r42_aarch64_cortex-a53_neon-vfpv4.ipk (6.1 MB)`. I click **Install** and leave "Allow overwriting conflicting package files" unticked, as the Flint text says: "(leave the other options as they are)".

**Stuck/misunderstood:** No. This is the opkg path I already know, and the steps on the home page match the screens exactly.

**What I expected:**
- There's one download button with no architecture choice. The Slate 7 and Flint 3 use different SoCs, and the page doesn't say this one .ipk fits both. "It's the same tailbond this router runs" suggests the build is for the Flint. I'd install anyway because LuCI/opkg would reject a wrong arch. (low)
- No checksum or signature appears next to the download. I'm giving this package root on my router, so I'd like a sha256 or at least "signed by …". (low)

## Task 2: Pair with the home server

**Where I click:** 02 shows "Looking for your home server on your tailnet…". On 03 I see "Found your home server: homebox · homebox.tail1234.ts.net · 100.101.1.1" and click **Ask to connect**. 04 shows "Waiting for approval on homebox… On homebox, tap Approve next to slate7", with a direct link to `http://homebox:8077`.

The failure states are good. 03b says "Turn on Tailscale first" and has **Open Tailscale settings**. 03c says "Checked 4 online machines. On the home server, tailbond must be installed and running, and Tailscale on." Both are the right level of detail for me.

**Stuck/misunderstood:** No.

**What I expected:** A short code shown on both sides, so I can be sure the request I approve is this Slate. The approval card (11/14) does show "signed in as alex@example.com · GL-BE3600 · 100.101.9.9", and I can match the 100.x against the admin console. That's enough for me. (low)

## Task 3: Hotel Wi-Fi plus phone tethering, both in use

**Where I look:** 05. The header says "Connected to homebox · Using Repeater (Wi-Fi) + Tethering (phone)". Repeater shows "In use ↓38 Mbps ↑4.2 Mbps · 24 ms". Tethering shows "In use ↓21 Mbps ↑2.9 Mbps · 48 ms". The Home server card shows "Connects to home.example.net over the internet (UDP port 443)" and "This router's tunnel address 10.77.0.2". That's what I wanted to know.

06 is also clear: "Not working right now. It's connected, but can't reach homebox. tailbond keeps trying; the other connections carry your traffic."

**Stuck/misunderstood:** No.

**What I expected:** On the Fedora Connections tab (14), the paths are labelled "Path 0 / Path 1 · Pixel-8 · Min 31 Ms", and the Devices tab (15) shows "198.51.100.20:51234 · wlb". I can't tell which uplink each path is ("wlb" is jargon, and "Ms" is capitalised oddly). The GL panel side is fine. (low)

## Task 4: Phone as backup only

**Where I click:** On the Tethering card, Use it → **Backup only** (07). The badge changes to "Backup · saving data", and the card says "Used only if your other connections stop working. Until then it isn't used at all, so it uses no data." The header shows "Tethering (phone) on standby". **Change limit** (07b) opens a field: "5 GB a month… Counts everything tailbond sends and receives over it… 0 = no limit."

**Stuck/misunderstood:** No.

**What I expected:** "uses no data" is a strong claim, and the limit only counts tailbond traffic. On a router, the tethering interface also carries the router's own traffic: GL cloud, Tailscale control and DERP, NTP, and health probes that check whether the backup works. I'd like a line saying whether tailbond stops the router itself from using the phone, or only stops tailbond. (low)

## Task 5: Reach the NAS by its MagicDNS name

**Where I look:** On the Slate's "Tailnet machines" list (05), I find `nas.tail1234.ts.net`, 100.101.2.2, "linux · Online · direct", and click **Copy name**. The text says "Laptops and phones on this router's Wi-Fi or LAN reach these by their full name, e.g. nas.tail1234.ts.net, or by their 100.x address". The Home server card says "Tailnet names: Work on every device connected to this router while tailbond is on. Use the full name". So the short name `nas` won't resolve, meaning no search domain is pushed. Fine, I know the FQDN.

**Stuck/misunderstood:** I'm unsure **which path my laptop's NAS traffic actually takes**. The Slate's list labels pixel-8 "via relay (slower)" (05/06/07/16/17). The Flint list in 12/13/18b/18c labels the same pixel-8 "direct". The page says LAN clients reach tailnet machines "while tailbond is on", which suggests the traffic goes laptop → bond → homebox → tailnet. In that case, the relay label on the Slate describes the Slate's own Tailscale path, and my traffic doesn't use it. But the Slate also runs GL's Tailscale, so it could go Slate → DERP → pixel-8. The screen doesn't tell me which hop "direct / via relay" refers to, or whether tailnet traffic goes through the bond or through the Slate's own tailscaled. That matters to me when I'm deciding whether a slow path is worth fixing. **(medium)**

**What I expected:** One line such as "Your devices reach these through homebox; direct/relay is measured from homebox". Or show the path from homebox on the Slate as well.

## Task 6: Turn tailbond off

**Where I click:** The toggle in the header. 09 shows "tailbond is off. Devices on this router use the internet directly, without your home server." Each link shows "Ready · Used when tailbond is on." **Unpair** (09b) confirms first and explains: "forgets the home server… it will need to approve". 10 has **Start tailbond**, with "Starting it is safe: devices here keep their internet while you set it up."

I like that 08 tells me plainly it fails closed: "Meanwhile, devices on this router have no internet, so nothing goes around your home server. To use this internet directly for now, turn tailbond off."

**Stuck/misunderstood:** No.

**What I expected:** When tailbond is off, "Tailnet machines" says "The list appears once tailbond is connected", and tailnet names only work "while tailbond is on". The Slate's GL Tailscale is still logged in, though. Can my laptop still reach `nas` through GL's own Tailscale (if "Allow Remote Access LAN" is on), or is the NAS gone until I turn tailbond back on? It isn't stated. (low)

## Task 7: Approve on the home side

**Flint (11):** Under "Waiting for your approval · 1 new", the card says "slate7 wants to connect · Travel router on your tailnet · signed in as alex@example.com · GL-BE3600 · linux · 100.101.9.9 · asked just now", with **Approve** / Decline. 12 then shows slate7 "Connected just now · 2 connections bonded".

**Fedora (14):** The banner reads "slate7 wants to connect", with **Approve**.

Who can approve is spelled out, which is what I wanted. Flint: "anyone logged in to this admin panel, and anyone on your Tailscale account who opens http://homebox:8077. To require a password there, set one under Internet and Tailscale." The row there reads "tailbond page password: None … Set". Fedora: "anyone on your Tailscale account (and anyone you shared this machine with). To require a password, set "ui_password" in /etc/tailbond/tailbond.json".

**Stuck/misunderstood:** No, but I'd set a password straight away.

**What I expected:**
- The two pages disagree. Fedora warns that users I **shared the machine with** can approve. The Flint text only says "anyone on your Tailscale account". Is the Flint also open to share-ins? Approving a device gives it my home route, so I want the same warning on both pages. (low)
- The default is no password. (low, because it's disclosed)
- Port inconsistency: the Fedora invite link is `http://homebox.tail1234.ts.net:80/#add-phone`, but everywhere else the page is `http://homebox:8077`. Which port does the UI actually listen on, and what do I need to allow in an ACL? (low)

## Task 8: Hotel Wi-Fi sign-in (GL repeater) with tailbond on

**Where I click:** 16 shows the banner "Repeater needs you to sign in. GL's Repeater page handles this sign-in. Open it, click "Login Mode for Public Hotspots"…" and the button **Open GL's Repeater page**. The Repeater card shows "Needs sign-in. Until you sign in, tailbond keeps it out and your other connections carry traffic." On 16b the Repeater shows "Signing in…" and the banner says "GL's Login Mode is on: devices here use the hotel Wi-Fi directly, not through your home server, so its sign-in page loads. It ends by itself once you've signed in (or click "Quit Login Mode"…)". The steps are clear.

**Stuck/misunderstood:** During the sign-in window I can't tell **what traffic leaves outside the tunnel**. The banner says "devices here use the hotel Wi-Fi directly", which sounds like *all* client traffic goes plaintext onto the hotel network. Meanwhile the header still says "Connected to homebox · Using Tethering (phone)", and Tethering is "In use". So is my laptop's traffic bonded over the phone or sent direct over the hotel Wi-Fi? Is the NAS reachable during this window? Does a background sync on the laptop leak onto the hotel LAN? For a security-minded user this is the key question, and the two parts of the page give opposite answers. **(medium)**

**What I expected:** Either "only the sign-in page goes direct; everything else stays in the tunnel over Tethering", or "all traffic goes direct for up to N minutes; the tunnel is paused". With the second, the header shouldn't still say "Using Tethering".

## Task 9: Ethernet sign-in page

**Where I click:** 17 shows the banner "Ethernet needs you to sign in… Opening the sign-in page lets devices here reach Ethernet directly (without your home server) for up to 10 minutes", with **Open sign-in page**. On 17b: "Signing in to Ethernet… For up to 9 more minutes (until 04:13), devices here use Ethernet directly, not through your home server. tailbond adds it back by itself once you're signed in." The options are **I've signed in, check now** / **Stop sign-in mode**. I'm pleased the window is time-limited and that there's a way to stop it.

**Stuck/misunderstood:** The same ambiguity as task 8 (counted in finding #8): the header says "Using Repeater (Wi-Fi)", and the banner says "devices here use Ethernet directly". The phone layout (17b-phone) has the same wording. Also, "until 04:13" doesn't say whose clock it uses (the router's time zone or the browser's). (low)

## Task 10: Home side set up for best speed

**Flint, good (18c/12):** "Setup health · All good". The checks listed are "Tailscale: 2 machines direct", "Network is friendly to direct connections (port mapping: UPnP)", "Route 10.77.0.0/24 approved", "Public internet address 203.0.113.7", and "UDP 443 reached from the internet (slate7 connected from 198.51.100.9, 2 h ago)". This is exactly the checklist I'd run by hand.

**Flint, relayed (18):** "2 to improve". "Tailscale: 2 machines direct, 1 through a relay (slower). Through a relay: pixel-8. To check one: tailscale ping pixel-8. Usually fixed by forwarding UDP port 41641…" and "Hard NAT (the outside port changes per destination); no port mapping on the router." It's correct and actionable.

**Flint, CGNAT (18b):** "Step 1: Your internet line uses carrier-grade NAT. This machine's internet address 100.72.3.4 is shared by your provider, so travel routers can't reach UDP 443 here."

**Flint, fresh (13):** Step 1 is to approve the 10.77.0.0/24 route in the admin console, and Step 2 is to set the DDNS name. Clear.

**Fedora (15):** "Setup health: 2 to improve for best speed", with the same two Tailscale items.

**Stuck/misunderstood:** No, but:
- "Tailscale: 2 machines direct" in 18c and 12, while the Tailnet machines list right below shows **3** online machines all "direct" (nas, apple-tv, pixel-8). Does the count include homebox itself, or only peers with recent traffic? (low)
- 18b says travel routers "can't reach UDP 443 here", but the Devices card above still shows "pixel-8 Connected 3 h 0 min · 2 connections bonded". I assume pixel-8 is on the home LAN, but that isn't stated. I also expected a fallback to be mentioned, such as carrying the bond over Tailscale itself or via a relay. The only option offered is "Ask your provider for a public IPv4 address (or use public IPv6)". (low)
- Fedora health shows only the two Tailscale items. The Flint also shows route approved, public address, and UDP 443 reached, so on Fedora I can't tell whether those passed or simply aren't checked there. (low)

---

## Findings

| # | task | finding | severity |
|---|---|---|---|
| 1 | 1 | One "Download the router package" button with no word on whether the .ipk (`aarch64_cortex-a53…`) fits the Slate 7 as well as the Flint ("It's the same tailbond this router runs") | low |
| 2 | 1 | No checksum or signature shown for a package that gets root on the router | low |
| 3 | 2 | No pairing code to compare on both sides; the 100.x IP and signed-in user on the approval card are the only identity check | low |
| 4 | 3 | Fedora paths shown as "Path 0 / Path 1 · Pixel-8 · Min 31 Ms" and "wlb", without naming the uplink | low |
| 5 | 4 | "Backup only … uses no data" and the limit "Counts everything tailbond sends" leave it unclear whether the router's own traffic (Tailscale control/DERP, GL cloud, probes) still uses the phone | low |
| 6 | 5 | "direct / via relay (slower)" on the Slate's Tailnet machines list disagrees with the Flint's view of the same machine (pixel-8). It's unclear which hop it measures and whether laptop → NAS traffic goes through the bond via homebox or through the Slate's own Tailscale | medium |
| 7 | 6 | With tailbond off, unclear whether LAN clients can still reach tailnet machines through GL's own Tailscale ("The list appears once tailbond is connected") | low |
| 8 | 8, 9 | During captive sign-in the banner says "devices here use the hotel Wi-Fi / Ethernet directly, not through your home server", but the header still says "Connected to homebox · Using Tethering/Repeater". Can't tell whether all client traffic leaves the tunnel unencrypted or only the portal page | medium |
| 9 | 7 | "Who can approve" differs: Fedora includes "anyone you shared this machine with", Flint only says "anyone on your Tailscale account". The default is no page password | low |
| 10 | 7 | Fedora invite link uses `:80` while everywhere else says `http://homebox:8077` | low |
| 11 | 9 | "until 04:13" doesn't say which clock or time zone | low |
| 12 | 10 | "Tailscale: 2 machines direct" while 3 online direct machines are listed below | low |
| 13 | 10 | CGNAT step says travel routers can't connect while pixel-8 shows "Connected … 2 connections bonded"; no fallback (over Tailscale or a relay) is mentioned | low |
| 14 | 10 | Fedora setup health has no route / public address / UDP 443 checks, unlike the Flint, so I can't tell pass from not-checked | low |

Counts: high 0, medium 2, low 12.
