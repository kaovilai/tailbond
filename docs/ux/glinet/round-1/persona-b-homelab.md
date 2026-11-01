# Persona B (homelab power user): GL.iNet round 1, raw report

Background: I run Tailscale subnet routers, OpenWrt and a Fedora server. I want to know which routes, DNS and endpoints are involved, how pairing is secured, and who can approve a device. I'd rather not SSH into the travel router from a hotel room.

Inputs: task-script.md, screens.md, and the round-1 PNGs. I looked at the desktop version of every screen and the phone versions of 01, 05 and 11.

---

## Task 1: Install without SSH

**Where I click** (`01-not-installed-luci-upload`): the manifest gives the path GL admin → SYSTEM → Advanced Settings → Go To LuCI, then System → Software → "Upload Package…". In the dialog I pick `tailbond_0.1.0-r42_aarch64_cortex-a53_neon-vfpv4.ipk (6.1 MB)` and press **Install**. The dialog names the arch suffix ("for GL.iNet routers the file ends in _aarch64_cortex-a53_neon-vfpv4.ipk"), so I can check I downloaded the right build. I know LuCI already, so this is familiar.

**What bothers me:**
- "Allow overwriting conflicting package files" is **ticked and greyed out**, so I can't untick it. That amounts to a forced `--force-overwrite`. My first worry is that it could replace files from GL's own `tailscale` package or other firmware packages. I'd hesitate here and want to know what it overwrites and why I'm not allowed to choose. I'd still go ahead, but uneasily. (medium)
- No checksum or signature is shown for the package, and there's no hint how to verify it. As a power user I'd like a sha256 to compare. (low)
- None of the screens shows how to get from the GL panel to LuCI. I only know the path from the manifest. Someone who has never used "Go To LuCI" would be lost, but I'm fine. (low)
- The screens don't show what happens after Install, such as a success message or the APPLICATIONS → tailbond menu entry appearing. Screen 02 just assumes it worked. (low)

## Task 2: Pair with the home server

**Where I click**: APPLICATIONS → tailbond (`02-just-installed`) shows "Looking for your home server on your tailnet…". A few seconds later (`03-found-server`) it shows "Found your home server: homebox / homebox.tail1234.ts.net · 100.101.1.1" with an **Ask to connect** button, which I click. `04-waiting-approval` then says "Waiting for approval on homebox… tap Approve next to slate7."

**What works**: the FQDN and the 100.x address are both shown, so I can confirm it found the right box. The no-Tailscale case (`03b`: "Turn on Tailscale first … Open Tailscale settings") and the nothing-found case (`03c`: "Checked 4 online machines…") are both clear, and I can type a name by hand.

**Questions:**
- "Checked 4 online machines": what does it probe, and on which port? I don't love something scanning my tailnet without saying what it does, though it's harmless if it's just a port check over Tailscale. (low)
- There's no shared code or fingerprint to compare between the Slate page and the server page. The server shows "signed in as alex@example.com · GL-BE3600 · 100.101.9.9", but the Slate page (04) never shows its own Tailscale IP or identity, so I can't check the request against it. Tailscale identity is probably good enough, but showing "this router: slate7 · 100.101.9.9" on 04 would let me check both ends. (low)

## Task 3: Hotel Wi-Fi plus USB tethering, both in use

**Where I look** (`05-connected-two-wans`): the header says "Connected to homebox / Using Wi-Fi (repeater) + phone (USB tethering)" with ↓59 / ↑7.1 Mbps. The Repeater card shows "In use ↓38 Mbps ↑4.2 Mbps · 24 ms" and the Tethering card shows "In use ↓21 ↑2.9 · 48 ms". That's clear: both are carrying traffic, and the per-link numbers add up to the total. Good.

`06-one-wan-failed` handles a failure well: "Not working right now. It's connected, but can't reach homebox. tailbond keeps trying; the other connections carry your traffic."

**What confuses me:**
- **I can't tell which endpoint the bonded tunnel dials.** The Slate's Home server card says "Address home.example.net:443". The Fedora dashboard header (14) says "Home server · homebox.tail1234.ts.net:443". Only the error screen (08) reveals that "UDP port 443 must reach home.example.net:443 from the internet". So is the bond running over the public internet with a port forward, or over the tailnet? That affects my firewall and port-forward setup. I only worked it out by piecing together 08 and 13 ("Set this router's internet address"). The Slate's normal Connected view should say plainly that this is a public UDP endpoint, and the Fedora header shouldn't show a different ts.net endpoint. (medium)
- The Repeater card doesn't show which upstream SSID I'm on (the hotel's network name). (low)
- In 05 the Tethering card shows speeds but no 192.168.42.x address, while 06 shows the IP instead of speeds. It's a minor inconsistency, but I'd like the interface name and IP on every card the way Ethernet shows "eth0". (low)

## Task 4: Phone as backup only

**Where I click**: on the Tethering card I tick **Backup only**. `07-save-data-standby` then shows "Backup · saving data, Used only if your other connections stop working" and the header says "phone (USB tethering) on standby". `07b`'s "Change limit" opens "5 GB a month … At the limit it pauses until next month. 0 = no limit." That's clear.

**What confuses me:**
- **There are two controls that seem to do the same thing.** "How to use your connections" has a **Save mobile data** mode: "Uses Ethernet and Wi-Fi; your phone and cellular only when those stop working". Each card also has a **Backup only** checkbox. In 07 the checkbox is ticked while the mode is still **Fastest**. Do I need both? Does the mode override the checkbox, or does the checkbox override the mode? If I choose "Save mobile data", does it tick Backup only on Tethering automatically? I'd probably set both just to be sure, without knowing which one actually matters. (medium)
- The limit text says "Counts everything tailbond sends and receives over it". Does standby still send keepalives or probes over the phone, and do they count against the limit? That's my real concern with metered data. (low)
- "Backup only" sits on the same row as the "Use" toggle. I can work out that Use=off means never and Backup only means only on failover, but a three-state control (Use / Backup / Off) would be clearer. (low)

## Task 5: Reach the NAS by its MagicDNS name

**Where I look**: the Tailnet machines list on 05 shows "nas, nas.tail1234.ts.net · 100.101.2.2, linux · Online" with a **Copy name** button. The intro line reads "Laptops and phones on this router's Wi-Fi or LAN reach these by name, e.g. nas.tail1234.ts.net." The Home server card adds "Names like nas.…ts.net: Work on every device connected to this router while tailbond is on." Easy to find.

**Questions:**
- Do short names (`nas`) work, or only the FQDN? Does the laptop get 100.100.100.100 as its resolver, or does the router forward `ts.net` queries? The screen doesn't say, and I'd test it with `dig`. (low)
- The label "Names like nas.…ts.net" looks truncated and odd. (low)
- On the server side (11/12) the wording is "reach these machines by name or 100.x address", while the Slate side only mentions names. Can laptops behind the Slate also use 100.x IPs? (low)
- Whether all internet traffic leaves from home, like an exit node, is only implied by the Off text ("Devices on this router use the internet directly, without your home server"). I'd like a line saying "all traffic exits at homebox". (low)
- On the phone screens, Tailscale IPs wrap in the middle ("100.101. / 2.2", "10 / 0.101.4.4"), which makes them hard to read or copy. (low)

## Task 6: Turn tailbond off

**Where I click**: the big toggle at the top right. `09-turned-off` shows "Off. Devices on this router use the internet directly, without your home server." The Home server card shows "Off", and Tailnet machines shows "The list appears once tailbond is connected." Unpair is a separate action with a good confirmation (09b: "forgets the home server … it will need to approve"). Clear, and no trouble.

- When tailbond is off, can LAN clients still reach tailnet names through GL's own Tailscale? The text implies not ("while tailbond is on"), but it isn't stated outright. (low)
- What "Pair again" does next to Unpair is unclear. Does it re-run the request and need approval again? (low)
- Screen 10 ("tailbond isn't running … Start tailbond") is reassuring, but it has no log link. The Fedora page has a Logs tab and the GL page has none, so I'd want to see why it stopped. (low)

## Task 7: Approve on the home side

**Flint 3 (`11-server-pending-request`)**: the "Waiting for your approval" card shows "slate7 wants to connect. Travel router on your tailnet · signed in as alex@example.com · GL-BE3600 · linux · 100.101.9.9 · asked just now" with **Decline / Approve** buttons. That's exactly the identity information I want. After approval (12) slate7 appears under Devices as "Connected just now · 2 connections bonded". Good.

**Fedora (`14`)**: the same approval card sits at the top of the page. Easy.

**Notes:**
- Who can approve? Anyone with the Flint admin password or access to the Fedora page at `http://homebox:8077`. That page is plain HTTP on port 8077. Is it limited to the tailnet, or open on the LAN? Is there any auth? Nothing on the screen says. (low)
- Can a node shared into my tailnet by another user ask to connect? The "signed in as" line helps, but it isn't highlighted if the owner differs from the server's owner. (low)
- 13 (route pending): "Approve the route 10.77.0.0/24 for homebox (Machines → homebox → Edit route settings)" with "Open admin console" is clear, and that's how I'd do it. Mentioning that `autoApprovers` can handle this would be nice. "Self-signed certificate (pinned by each device)" against "Certificate from Tailscale" tells me what I'd want to know. (low)
- Fedora wording differs from GL: "Every packet twice" vs "Sends everything over every connection", and "Backup only on failure" vs "Uses Ethernet and Wi-Fi…". Path cards also read "Min 31 Ms" (capital M in "Ms"). (low)
- Server header speeds (↓7.4 ↑61) are from the server's point of view, so they look reversed compared with the Slate (↓59 ↑7.1). That's fine once you notice it, but it's unlabeled. (low)

---

## Findings

| # | task | finding | severity |
|---|---|---|---|
| 1 | 1 | "Allow overwriting conflicting package files" is forced on and greyed out; I fear it clobbers GL's own Tailscale or firmware files and can't opt out | medium |
| 2 | 1 | No checksum or signature to verify the .ipk | low |
| 3 | 1 | No on-screen path from the GL panel to LuCI; only the manifest gives it | low |
| 4 | 1 | No post-install confirmation shown | low |
| 5 | 2 | "Checked 4 online machines": doesn't say what or which port is probed on my tailnet | low |
| 6 | 2 | Waiting screen (04) doesn't show the Slate's own identity or 100.x to cross-check against the server's request | low |
| 7 | 3 | Can't tell which endpoint the bond dials: Slate "Address home.example.net:443" vs Fedora header "homebox.tail1234.ts.net:443"; that it's public UDP 443 shows up only in the error screen | medium |
| 8 | 3 | Repeater card doesn't show the upstream SSID | low |
| 9 | 3 | Tethering card shows speeds in 05 but IP in 06; no interface or IP shown consistently | low |
| 10 | 4 | "Backup only" checkbox and "Save mobile data" mode overlap; checkbox ticked while mode stays "Fastest"; unclear which wins or whether both are needed | medium |
| 11 | 4 | Unclear whether standby keepalives use phone data or count toward the 5 GB limit | low |
| 12 | 4 | "Use" toggle plus "Backup only" checkbox would read better as one three-state control | low |
| 13 | 5 | Doesn't say whether short names work or which DNS resolver clients get | low |
| 14 | 5 | "Names like nas.…ts.net" label looks truncated | low |
| 15 | 5 | Slate side mentions names only; server side says "name or 100.x address" | low |
| 16 | 5 | All traffic exiting at home (exit-node-like) is only implied by the Off text | low |
| 17 | 5 | Phone layout wraps 100.x IPs mid-address | low |
| 18 | 6 | Unclear whether GL's own Tailscale still gives LAN clients tailnet access when tailbond is off | low |
| 19 | 6 | "Pair again" meaning unclear | low |
| 20 | 6 | GL page has no logs (screen 10 "isn't running" gives no reason or log link) | low |
| 21 | 7 | Fedora page is `http://homebox:8077`; no indication of who can reach it or approve | low |
| 22 | 7 | Shared-in nodes from another user aren't flagged in the approval card | low |
| 23 | 7 | Route-approval step could mention `autoApprovers` | low |
| 24 | 7 | Fedora vs GL mode wording differs; "Min 31 Ms" capitalization | low |
| 25 | 7 | Server-side ↓/↑ figures are from the server's point of view and unlabeled | low |
