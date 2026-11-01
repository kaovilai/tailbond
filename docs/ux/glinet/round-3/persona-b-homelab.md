# Round 3 – Persona B (homelab power user)

Background: I run Tailscale subnet routers, OpenWrt and a Fedora server. I want to know what is actually happening (routes, DNS, who can approve) but I don't want to SSH into a travel router from a hotel. The home side is either the Fedora box (screens 14–15) or a Flint 3 (11–13). I went through the screens from both sides.

## Task 1 – Install on the Slate 7 without SSH

**Where I click:**
- Fedora (15, Devices tab): "Add a GL.iNet travel router" → "Download the router package" → I get `tailbond_0.1.0-r42_aarch64_cortex-a53_neon-vfpv4.ipk` (seen in 01b).
- Slate 7 (01a): SYSTEM → Advanced Settings → "Go To LuCI" (`http://192.168.8.1:8080/cgi-bin/luci`). The note "Use the same admin password to log in" is welcome.
- LuCI (01b): System → Software → "Upload Package…" → dialog "Really install the uploaded package?" → **Install**.

The steps on 15 match 01a/01b word for word. Using LuCI's package upload is exactly how I would do it.

**Snags:**
- **Flint 3 as the home server (11–13):** "Add a GL.iNet travel router" step 1 says "Get the router package on your computer: the tailbond file ending in _neon-vfpv4.ipk (from the tailbond releases page, or from a Fedora home server's tailbond page)." There is no download button and "the tailbond releases page" is not a link. If the Flint is my only home server, I have to go and find a releases page myself and pick the right asset by its file suffix. The Fedora page has a one-click download, and the Flint panel serves the same purpose.
- 01b shows the checkbox "Allow overwriting conflicting package files". Nothing tells me whether to tick it (I'd leave it off). I also can't tell whether the `.ipk` needs packages pulled from the opkg feeds (kmods etc.). If it does, installing at a hotel with no `opkg update` could fail, and no screen covers that.
- 15 offers one "router package" with no list of supported models or architectures. The Slate 7 is aarch64 so I'm fine, but I only know that because I read the filename in 01b.

## Task 2 – Pair

**Where I click:** On the Slate (02 → 03), the page auto-discovers "Found your home server: homebox / homebox.tail1234.ts.net · 100.101.1.1" → **Ask to connect** → 04: "Waiting for approval on homebox… On homebox, tap Approve next to slate7."

**Good:**
- Discovery shows the FQDN and the 100.x address, so I can check it's the right node.
- 03b ("Turn on Tailscale first") and 03c ("Checked 4 online machines. On the home server, tailbond must be installed and running…") are honest, useful errors.

**Snags:**
- **Who can approve?** 04 says "open http://homebox:8077 on any device signed in to your Tailscale account (a phone works too)". On the Fedora page (14/15) I see no login of any kind: anyone who can reach that page on the tailnet sees Approve/Decline. The "Invite someone's phone" link on 15 is designed so a family member opens it and taps "Add this phone" on their own. My tailnet has family devices and shared-in nodes, so I would stop and wonder whether any of them could approve a router or add a device. Nothing on screen says whether access is limited to my user, the tailnet owner, or anyone with network reach (ACLs?). I would still approve, but not comfortably.
- There's no way to confirm that the request I'm approving is the one I just made. The Slate's page (04) doesn't show its own 100.x IP or a short code that I could match against "100.101.9.9" on the approval card. The "signed in as alex@example.com" identity line on the card does help a lot.
- Port inconsistency: 04 says `http://homebox:8077`, but the invite link on 15 is `http://homebox.tail1234.ts.net:80/#add-phone`. Is this the same page on two ports, or is one going through `tailscale serve`?
- "Make this the home server" sits as a link under the pairing flow on the travel router (02–04). It says "(you can switch back here)", so a misclick is low risk, but it seems out of place on a travel router.

## Task 3 – Hotel: Repeater + USB tethering, confirm both used

**Where I look:** 05. The header reads "Connected to homebox – Using Repeater (Wi-Fi) + Tethering (phone)". Both cards show "In use" with per-link rates ("↓ 38 Mbps ↑ 4.2 Mbps · 24 ms", "↓ 21 Mbps ↑ 2.9 Mbps · 48 ms"), and the total "↓ 59 Mbps" equals the sum. That's clear.

The Home server card shows "Connects to home.example.net over the internet (UDP port 443)". That is exactly what I want to know: the data path is UDP/443 to my public address, not over Tailscale. 06 handles a failing link well ("It's connected, but can't reach homebox. tailbond keeps trying; the other connections carry your traffic").

**Snags (minor):**
- The Repeater card doesn't show which SSID it's attached to. It's a small thing, but at a hotel with several networks I'd like to see it.
- I'm inferring that this is a full tunnel. The page never says plainly that all traffic from devices exits at home. 08's kill-switch text ("devices on this router have no internet, so nothing goes around your home server") confirms it after the fact. I'd also like to know which DNS resolver clients use while connected.

## Task 4 – Phone as backup only

**Where I click:** On the Tethering card, "Use it" → **Backup only** (07). The status changes to "Backup · saving data" and the text reads "Used only if your other connections stop working. Until then it isn't used at all, so it uses no data." The header updates to "Tethering (phone) on standby". The hint under "Speed or reliability" points to the same control. Good.

The cap is in 07b, under "Change limit": "5 GB a month … Counts everything tailbond sends and receives over it. At the limit, tailbond stops using this connection … 0 = no limit." That is precise enough for me.

**Snags (minor):**
- I can't see what "stop working" means (detection time / failover delay), or whether keepalive probes still cost a few KB. "Uses no data" sounds a bit absolute.
- The segmented control has no Save button and no confirmation. I assume it applies instantly. That's fine, but a toast would reassure me.

## Task 5 – NAS by MagicDNS name

**Where I look:** "Tailnet machines" on 05/07: `nas` / `nas.tail1234.ts.net` / `100.101.2.2` / "linux · Online", with a **Copy name** button. The Home server card states the rule: "Use the full name, like nas.tail1234.ts.net." Done in seconds.

**Snags (minor):**
- The short name `nas` evidently won't resolve. I'm used to search domains on my tailnet clients, so I'd probably try `nas` first. The text covers this, but I'd still like to know why: is it a dnsmasq forward for `*.ts.net` only?
- On 08/09 the same row says "like nas.your-tailnet.ts.net", a placeholder, even though the router already knows the real tailnet (`tail1234`) from earlier screens.
- When tailbond is off (09), "Tailnet machines" shows "The list appears once tailbond is connected." The Slate's own GL Tailscale is still on, though. Can LAN clients reach the NAS through GL's Tailscale in that state, or not at all? This is also unclear while connected: which of the two paths do 100.x packets take when both exist?

## Task 6 – Turn tailbond off

**Where I click:** the toggle at the top right of the status card (05) → 09 "tailbond is off – Devices on this router use the internet directly, without your home server." That's clear. 08 also has a "Turn tailbond off" button inside the error, which is the right place for it.

The "Unpair" dialog (09b) clearly separates turning off from unpairing ("forgets the home server … it will need to approve"). As a power user I'd also want to know whether unpairing revokes the key on homebox, or leaves "slate7" in homebox's device list until I click Remove there.

**Snags (minor):**
- "Ask to connect again" still shows next to Unpair while connected (05) and while off (09). I don't know what it does on a router that's already paired: does it re-key, or re-request approval?

## Task 7 – Approve on the home side

**Flint 3 (11):** "Waiting for your approval · 1 new" → "slate7 wants to connect – Travel router on your tailnet · signed in as alex@example.com · GL-BE3600 · linux · 100.101.9.9 · asked just now" → **Approve**. 12 then lists slate7 under Devices ("Connected just now · 2 connections bonded"). Good: identity, model, IP and time are all there.

**Fedora (14):** the same card is pinned above everything. On the phone view (14-phone) it sits above a large "Add this phone" card that I don't need when I came to approve a router, but Approve is at the top, so that's fine.

**Route setup (13):** "Step 1: allow tailbond devices on your Tailscale network … tick the one route listed (10.77.0.0/24) … Devices can already reach your machines without it." This tells me the server masquerades into the tailnet, and the route is only needed for the reverse direction. I appreciate seeing the actual CIDR. I'd note that this only exposes tunnel IPs, not the Slate's LAN (192.168.8.0/24), so the NAS can't reach my laptop. That's acceptable, but it isn't stated. "Secure connection: This router's own certificate, checked by each device" vs "Certificate from Tailscale for homebox.tail1234.ts.net" (11/12) is good transparency.

**Snags:**
- The "who can approve" question again (see Task 2). Neither home UI says who is allowed to press Approve.
- "Decline": I can't tell whether this blocks the device or just dismisses the request.

## Findings

| # | task | finding | severity |
|---|---|---|---|
| 1 | 2, 7 | No indication of who can approve or add devices. The Fedora page (14/15) shows Approve/Decline and "Add this phone" with no login, and the invite link is meant for family to self-add. With family and shared nodes on my tailnet I can't tell whether they could approve a router or add a device. | medium |
| 2 | 1 | Flint 3 panel (11–13) has no download button for the router package. It says "from the tailbond releases page" (not a link) and asks me to pick by the "_neon-vfpv4.ipk" suffix. | medium |
| 3 | 1 | LuCI dialog (01b) "Allow overwriting conflicting package files" is not addressed. Nothing says whether the .ipk needs feed dependencies, which matters at a hotel without opkg update. | low |
| 4 | 1 | "Download the router package" (15) doesn't say which models/architectures it's for. Only the filename reveals aarch64. | low |
| 5 | 2 | No match code or own 100.x IP on the Slate (04) to cross-check against "100.101.9.9" on the approval card. | low |
| 6 | 2 | Port mismatch: "http://homebox:8077" (04) vs "http://homebox.tail1234.ts.net:80/#add-phone" (15). | low |
| 7 | 2 | "Make this the home server" link on the travel router's pairing page (02–04) is easy to misclick. It is reversible. | low |
| 8 | 3 | Full-tunnel behaviour and the DNS resolver clients use are only implied (spelled out only in 08's error text). | low |
| 9 | 3 | Repeater card doesn't show which hotel SSID it's on. | low |
| 10 | 4 | "Backup only": failover trigger/delay not shown. "uses no data" sounds absolute (keepalives?). No confirmation that the change was applied. | low |
| 11 | 5 | Short name "nas" won't work (full name required). Stated, but the reason (DNS forwarding scope) is not. | low |
| 12 | 5 | Placeholder "nas.your-tailnet.ts.net" on 08/09 even though the real tailnet (tail1234) is known. | low |
| 13 | 5, 6 | Unclear whether LAN clients reach the tailnet through the Slate's own GL Tailscale when tailbond is off (09), and which path 100.x traffic takes when both are up. | low |
| 14 | 6 | "Ask to connect again" shown on an already-paired, connected router (05/09). Its effect is unclear. | low |
| 15 | 6 | Unpair dialog (09b) doesn't say whether the device and key are also removed or revoked on homebox. | low |
| 16 | 7 | "Decline" outcome unclear (block vs dismiss). | low |
| 17 | 7 | Route step (13) doesn't say the Slate's LAN (192.168.8.0/24) isn't reachable from the tailnet. Only tunnel IPs (10.77.0.0/24) are. | low |
| 18 | 7 | Fedora phone view (14-phone): a large "Add this phone" card follows the approval card. It's a distraction when I only came to approve. | low |
