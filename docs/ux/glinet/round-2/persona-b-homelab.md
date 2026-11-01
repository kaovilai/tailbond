# Persona B (homelab power user): round 2, GL.iNet

Background: I run Tailscale subnet routers, OpenWrt and a Fedora server. I want to know what's actually going on: which path the tunnel uses, routes, DNS, how pairing is authenticated and who can approve. I don't want to SSH into a travel router from a hotel room.

Inputs: task-script.md, screens.md, and every PNG in `out/ux/glinet/round-2/` (all desktop screens; phone versions of 05, 11 and 14).

---

## Task 1: Install on the Slate 7 without SSH

**Clicks:** `01a-install-gl-advanced-settings`: SYSTEM → Advanced Settings → **Go To LuCI** ("LuCI is OpenWrt's own web interface. Use the same admin password to log in."). Then `01b-install-luci-upload`: System → Software → **Upload Package…**, pick `tailbond_0.1.0-r42_aarch64_cortex-a53_neon-vfpv4.ipk (6.1 MB)`, then **Install**.

**Assessment:** This is the standard OpenWrt path, so it took me no thought. The Flint and Fedora pages both repeat the same steps ("the tailbond file ending in _neon-vfpv4.ipk"), and the Fedora Devices tab has a **Download the router package** button. Small gaps:
- LuCI's dialog shows the "Allow overwriting conflicting package files" checkbox. No instruction mentions it. I'd leave it unchecked, but a hint ("leave unchecked") would remove the doubt. (low)
- No checksum or signature is shown for the .ipk. On the Flint page the source is "the tailbond releases page", with no link. The Fedora download is fine because it comes from my own box. (low)
- Nothing says whether the package survives a GL firmware upgrade. I'd look for that before a trip. (low)

## Task 2: Pair the Slate 7 with the home server

**Clicks:** `02-just-installed` shows "Looking for your home server on your tailnet…". `03-found-server` shows "Found your home server: homebox / homebox.tail1234.ts.net · 100.101.1.1", then **Ask to connect**. `04-waiting-approval` shows "Waiting for approval on homebox… On homebox, tap **Approve** next to **slate7**", with separate instructions for Flint and for a Fedora computer.

**Assessment:** Good. It shows the FQDN and the 100.x address, so I can check it's my box and not something else on the tailnet. The error states are useful:
- 03b: "Turn on Tailscale first", with an "Open Tailscale settings" button.
- 03c: "Checked 4 online machines. On the home server, tailbond must be installed and running, and Tailscale on."

Notes:
- The discovery step probes my tailnet machines ("Checked 4 online machines"). I'd like to know which port it probes. Not important. (low)
- The Slate page never shows its own tailnet IP (100.101.9.9) before approval. The request on the server shows "GL-BE3600 · 100.101.9.9", and I'd like to compare that with a number on the Slate side. The name "slate7" is enough in practice. (low)

## Task 3: Hotel Wi-Fi plus USB tethering, confirm both are used

**Where:** `05-connected-two-wans`. The header says "Connected to homebox / Using Repeater (Wi-Fi) + Tethering (phone)". Both cards show "In use" with their own rates and latency ("↓ 38 Mbps ↑ 4.2 Mbps · 24 ms", "↓ 21 Mbps ↑ 2.9 Mbps · 48 ms"). Confirming both links are used is easy.

**Where I got confused:** what path does the tunnel actually take? The Slate page says "Connects to `home.example.net` over the internet (UDP port 443)". The Fedora page header (14/15) says "Home server · reachable at homebox.tail1234.ts.net:443". Those are two different stories:
- a direct UDP connection to my public IP and DDNS name, with port 443 forwarded; or
- a connection over Tailscale (tunnel inside a tunnel, possibly through a DERP relay).

This matters to me for MTU, for speed (traffic inside Tailscale may go through DERP), and for whether a port forward is needed. The Flint's "Internet address home.example.net:443" plus the port-forward note suggest the first. The Fedora header contradicts it. I'd go and run `ss -ulpn` on the server to check. **(medium)**

Other notes:
- In 06, the Tethering card replaces its rate with "192.168.42.129" and the message "It's connected, but can't reach homebox. tailbond keeps trying; the other connections carry your traffic." This is clear. (No finding.)
- Only Ethernet shows the interface name (`eth0`). Repeater and Tethering show none, so I can't match them to `ip link` names. (low)

## Task 4: Phone link as backup only

**Clicks:** Tethering card, Use it, **Backup only** (`07-save-data-standby`). The badge changes to "Backup · saving data", with "Used only if your other connections stop working. Until then it isn't used at all, so it uses no data." The header changes to "Tethering (phone) on standby". `07b-set-limit` shows the cap: "5 GB a month … Counts everything tailbond sends and receives over it. At the limit it pauses; the count restarts on the 1st of each month."

**Assessment:** I found this right away. The "Speed or reliability" card also points to it ("Saving your phone's data? Set Tethering … to 'Backup only'"). Things I'd question:
- "it isn't used at all, so it uses no data": then how does it detect that the backup link still works? Are there no keepalives at all? (low)
- The limit only counts tailbond traffic. On this router, GL's own Tailscale is still on and can pick the USB interface for its own traffic (DERP, control plane). Firmware checks could too. The page doesn't say whether "Backup only" also keeps the router itself off the phone link. I'd want this stated. (low)
- "At the limit it pauses": does a paused link stay usable as a last-resort backup, or is it fully off? (low)

## Task 5: Reach the NAS from the laptop by MagicDNS name

**Where:** `05`, the Tailnet machines card. It shows "nas / `nas.tail1234.ts.net · 100.101.2.2` / linux · Online", with a **Copy name** button. Home server card: "Tailnet names: Work on every device connected to this router while tailbond is on. Use the full name, like nas.tail1234.ts.net."

**Assessment:** I found the name immediately. The machine list with Copy name is exactly right. Power-user gaps:
- The page doesn't explain how laptop traffic reaches the tailnet. I assume it goes through the bond tunnel to homebox and then out of homebox's Tailscale, with source NAT. The Flint setup step 1 (13), approving `10.77.0.0/24` as a subnet route, confirms half of that. On the Slate side it's not clear whether the Slate's own GL Tailscale also routes this traffic, which path wins, or whether I can turn the Slate's Tailscale off after pairing. (low)
- "Use the full name" implies short names (`nas`) don't work, so no search domain is pushed. That's fine because it's stated. I'd like one line on who answers DNS for `*.ts.net` (the Slate's dnsmasq forwarding over the tunnel?). (low)
- Phone layout (05-phone, 11-phone): the monospace FQDN wraps mid-word: "nas.tail1234.ts.ne" on one line, then "t · 100.101.2.2" on the next. If I read it off the screen to type it on another device, I'd get it wrong. Copy name saves this. (low)
- When disconnected (08, 09), the example text changes to "like nas.your-tailnet.ts.net" while the real tailnet is known (tail1234). This is a bit inconsistent. (low)

## Task 6: Turn tailbond off

**Clicks:** the toggle at the top right of the status card (05). Result in `09-turned-off`: "tailbond is off. Devices on this router use the internet directly, without your home server. Turn on to bond your connections again." In the failure state (08) there's also a **Turn tailbond off** button.

**Assessment:** Easy. Off, Unpair and "not running" are separate, which I like:
- 09b Unpair confirmation: "This router stops using tailbond and forgets the home server … it will need to approve".
- 10: "tailbond isn't running on this router … Starting it is safe: devices here keep their internet while you set it up."

Notes:
- 08 states clearly that the tunnel fails closed: "Meanwhile, devices on this router have no internet, so nothing goes around your home server." I appreciate that. The kill-switch behaviour is spelled out honestly.
- The GL toggle has no text label (On/Off). The Fedora page shows "Connected" next to its toggle. Minor. (low)
- "Ask to connect again" appears while already connected (05). It doesn't say whether this re-keys or needs another approval. (low)
- With tailbond off (09), "Tailnet machines: The list appears once tailbond is connected", even though GL's Tailscale on the Slate is still logged in. Can LAN clients still reach the tailnet another way? It doesn't say. (low)

## Task 7: Approve on the home side

**Flint (`11-server-pending-request`):** "Waiting for your approval · 1 new / slate7 wants to connect / Travel router on your tailnet · signed in as alex@example.com / GL-BE3600 · linux · 100.101.9.9 · asked just now", with **Approve**. In 12 it appears under Devices: "slate7 Connected just now · 2 connections bonded".

**Fedora (`14-fedora-dashboard-request`):** the same card at the top, with **Approve**.

**Assessment:** Clear on both. The request carries the Tailscale identity, so I can verify it. Notes:
- **Who can approve on Fedora?** The Slate tells me to "open http://homebox:8077 from any device on your Tailscale account (a phone works too)". The Fedora page has no login and doesn't show who I'm signed in as. If any node on the tailnet can open the page, then any node (or a shared-in user, depending on ACLs) can approve devices that get access through my home server. I'd want a line like "only alex@example.com can approve" or "approval checks your Tailscale identity". I'd check my ACLs before trusting this. (low; it doesn't block the task, but it's exactly what I care about)
- The dashboard port is inconsistent. The Slate says `http://homebox:8077`. The Fedora "Invite someone's phone" link is `http://homebox.tail1234.ts.net:80/#add-phone`. Are both served? (low)
- Fedora Add a device says "paste the link into the router's Setup tab". The GL panel has no Setup tab; it's "Other ways to connect → Paste a link". (low)
- Fedora stats: the top card reads "1.1 Mbps down / 12.0 Mbps up", and the pixel-8 row reads "↓ 12.0 Mbps ↑ 1.1 Mbps". So the server-side and device-side directions are flipped with no label. Also jargon: "wlb", "Min 31 Ms". (low)
- On a phone, the Fedora page puts a large "Add this phone" card right under the approval. Fine; the approval is still on top. (No finding.)
- Flint 13 (fresh setup) is excellent for my persona. It names the exact route to approve: "Machines → homebox → Edit route settings → turn on **10.77.0.0/24**". It is honest about what fails until then ("your Tailscale machines can't reach the devices connected through tailbond (those devices can still reach them)"). It also flags the double-NAT port-forward case.

---

## Findings

| # | task | finding | severity |
|---|---|---|---|
| 1 | 1 | LuCI's "Allow overwriting conflicting package files" checkbox isn't mentioned in any instructions | low |
| 2 | 1 | No checksum or signature for the .ipk; Flint's "tailbond releases page" has no link | low |
| 3 | 1 | Nothing says whether the package survives a GL firmware upgrade | low |
| 4 | 2 | Slate never shows its own tailnet IP to compare with "100.101.9.9" in the approval request | low |
| 5 | 2 | "Checked 4 online machines": it doesn't say what discovery probes (port) | low |
| 6 | 3 | Tunnel path contradicts itself: Slate says "Connects to home.example.net over the internet (UDP port 443)", Fedora header says "reachable at homebox.tail1234.ts.net:443"; can't tell if the bond runs direct or inside Tailscale (MTU, DERP, port-forward needed?) | medium |
| 7 | 3 | Only Ethernet shows an interface name (eth0); Repeater and Tethering don't | low |
| 8 | 4 | "isn't used at all, so it uses no data": unclear how the backup link's health is checked, and whether the router's own Tailscale or firmware traffic can still use the phone link | low |
| 9 | 4 | "At the limit it pauses": does a paused link still act as a last-resort backup? | low |
| 10 | 5 | Routing and DNS mechanism on the Slate side not explained (tunnel vs the Slate's own GL Tailscale; who resolves *.ts.net) | low |
| 11 | 5 | Phone: monospace FQDN wraps mid-word ("ts.ne" / "t · 100.101.2.2"); Copy name works around it | low |
| 12 | 5 | Disconnected states use the placeholder "nas.your-tailnet.ts.net" when the tailnet is known | low |
| 13 | 6 | GL on/off toggle has no text label | low |
| 14 | 6 | "Ask to connect again" shown while connected; unclear if it re-keys or needs re-approval | low |
| 15 | 6 | When off, unclear whether LAN clients can still reach the tailnet through the Slate's own Tailscale | low |
| 16 | 7 | Fedora approval page has no visible authentication or identity; "any device on your Tailscale account" can open it, so who can approve isn't stated | low |
| 17 | 7 | Dashboard port mismatch: homebox:8077 (Slate) vs homebox.tail1234.ts.net:80 (Fedora invite link) | low |
| 18 | 7 | Fedora text says "router's Setup tab"; GL panel calls it "Other ways to connect → Paste a link" | low |
| 19 | 7 | Fedora up/down directions flip between the server card and the device row; jargon "wlb", "Min 31 Ms" | low |
