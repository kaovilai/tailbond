# Round 11 — GL persona B (homelab)

Persona: homelab power user. I run Tailscale subnet routers, OpenWrt and a Fedora server at home. I care what actually happens: routes, DNS, direct vs relayed (DERP), NAT, who can approve, and where traffic goes during captive-portal sign-in. I'd rather not SSH into a travel router from a hotel.

Materials: `task-script.md`, `screens.md` and the PNGs in `out/ux/final/round-11/packet-gl/`. I looked at the desktop screens for every task and at the phone versions of 05, 14, 16b and 17b.

Overall: this is the clearest the GL panel has been about the plumbing. The panel answers most of my questions directly:
- The bond goes over the internet, not Tailscale: "Connects to home.example.net over the internet (UDP port 443)".
- Tailnet access goes through the home server, not the Slate's own Tailscale: "(not this router's own Tailscale)".
- It says per machine whether the home server reaches it "direct" or "via relay".
- When home is unreachable, the router fails closed and the screen says so.
- During captive-portal sign-in, it states exactly where traffic goes.

I could finish every task. What's left is one identity/approval contradiction, one captive-portal warning that understates what happens, and polish.

---

## Task 1 — Install on the Slate 7 without SSH

**Clicks:** on screen 15 (Fedora, Devices tab), "Add a GL.iNet travel router (recommended)" → **Download the router package**. On the Slate: SYSTEM → Advanced Settings → **Go To LuCI** (01a; the screen notes "Use the same admin password to log in"). In LuCI: System → Software → **Upload Package…**. The dialog in 01b reads "Really install the uploaded package?" with `tailbond_0.1.0-r42_aarch64_cortex-a53_neon-vfpv4.ipk (6.1 MB)` → **Install**.

**What worked:** no SSH, and it's standard opkg-via-LuCI, which I trust. The path on screen 15 matches the GL menus exactly.

**Friction:**
- The Fedora instructions just say "pick the file, Install." The Flint's version of the same card (11/12/13) adds "(log in with the same password)" and "(leave the other options as they are)". LuCI's dialog shows an "Allow overwriting conflicting package files" checkbox. I know to leave it off, but the Fedora page doesn't say so. **Low.**
- "Download the router package" names no architecture or supported models. I only see `aarch64_cortex-a53` in LuCI after uploading. The Flint version says "It's the same tailbond this router runs", which doesn't help when the home server is an x86 Fedora box. I'd want "for GL routers with … (e.g. Slate 7, Flint 3)" next to the button. **Low.**

## Task 2 — Pair with the home server

**Clicks:** VPN → tailbond. Screen 02 shows "Looking for your home server on your tailnet…". Screen 03 shows "Found your home server: homebox / homebox.tail1234.ts.net · 100.101.1.1" → **Ask to connect**. Screen 04 shows "Waiting for approval on homebox…", which tells me to open `http://homebox:8077` from any device on my account, or use the Flint's panel.

**What worked:** discovery over the tailnet, and the fallbacks are honest. 03b says "Turn on Tailscale first" with "Open Tailscale settings". 03c says "Checked 4 online machines. On the home server, tailbond must be installed and running, and Tailscale on." Both offer a manual name field.

**Friction:** see the identity finding under Task 7. Screen 02 states "Your home server … must be signed in to the same Tailscale account as this router". That's a hard requirement, and other screens contradict it.

## Task 3 — Hotel Wi-Fi (Repeater) + USB tethering, both in use

**Clicks:** none needed. Screen 05: "Connected to homebox / Using Repeater (Wi-Fi) + Tethering (phone)". Repeater shows "In use · ↓38 Mbps ↑4.2 Mbps · 24 ms" and Tethering shows "In use · ↓21 Mbps ↑2.9 Mbps · 48 ms". The headline is "↓59 Mbps", which is the sum. Good enough to confirm that both links carry traffic.

Screen 06 handles a failing link well: Tethering shows "Not working right now". The card explains: "It's connected, but can't reach homebox. tailbond keeps trying; the other connections carry your traffic." This is exactly the distinction I care about (link up, but no path to home).

**Friction (home side):** on the Fedora Connections tab (14), the per-path cards are labelled "Path 0 · Pixel-8 · Min 31 Ms" and "Path 1 … Min 58 Ms". Nothing says which uplink each path is (Wi-Fi vs cellular). "Ms" is capitalised oddly. Screen 15's device row ends in an unexplained "· wlb". From home I can't tell which hotel link is misbehaving. **Low.**

## Task 4 — Phone as backup only

**Clicks:** Tethering card → Use it: **Backup only** (07). The status changes to "Backup · saving data". The card says: "Used only if your other connections stop working. Until then tailbond sends nothing over it (GL itself may check the connection now and then, a few KB)." The header says "Tethering (phone) on standby". This is exactly what I wanted to know, including the GL health-check caveat.

Screen 09 handles tailbond being off honestly: "While tailbond is off, GL may use Tethering (your phone's data) … 'Backup only' and data limits only apply while tailbond is on."

**Friction:** under "Change limit" (07b), the counter "restarts on the 1st of each month" and "Counts everything tailbond sends and receives over it". My carrier's cycle doesn't start on the 1st, and GL's own checks aren't counted. So "1.2 GB of 5 GB" will drift from what my carrier bills, and I can't set a reset day. **Low.**

## Task 5 — Reach the NAS by MagicDNS name from the laptop

**Clicks:** scroll to "Tailnet machines" (05) → `nas` row → `nas.tail1234.ts.net · 100.101.2.2 · linux · Online · direct` → **Copy name**. The Home server card says: "Tailnet names: Work on every device connected to this router while tailbond is on. Use the full name, like nas.tail1234.ts.net."

**What worked:** it says outright that this goes through the bond to homebox, not the Slate's own Tailscale. "Direct"/"via relay" is from homebox's point of view, and the screen says that too.

**Friction:** short names (`nas`) apparently don't work; I'm told to use the FQDN. That's fine, but it doesn't say who answers DNS for my laptop or whether non-ts.net lookups also go over the bond. Not blocking. **Low.**

## Task 6 — Turn tailbond off

**Clicks:** the toggle on the status card (05) → screen 09, "tailbond is off". The text: "Devices on this router use the internet directly, without your home server, over whichever connection GL's own settings pick." Each card restates that its choice only applies while tailbond is on. **Unpair** is separate and confirmed (09b: "This router stops using tailbond and forgets the home server … it will need to approve"). Turning off vs forgetting is clear.

The failure mode is spelled out in screen 08: "Meanwhile, devices on this router have no internet, so nothing goes around your home server. To use this internet directly for now, turn tailbond off …". It also says what to check at home: "UDP port 443 reaches home.example.net:443 from the internet". I like that it fails closed and says so.

**Friction:** screen 10 ("tailbond isn't running on this router") says "It may have been turned off or stopped after an error. Starting it is safe: devices here keep their internet while you set it up." There are two problems:
- If I turn it off with the toggle, I get screen 09, not 10, so "turned off" here means something else.
- "Keep their internet" contradicts screen 08. If home is unreachable once tailbond starts, devices lose internet.

**Low.**

## Task 7 — Approve the Slate 7 at home

**Clicks:**
- Flint 3 (11): "Waiting for your approval · 1 new · slate7 wants to connect · Travel router on your tailnet · signed in as alex@example.com · GL-BE3600 · 100.101.9.9" → **Approve**. Screen 12 shows slate7 under Devices with "Connected just now · 2 connections bonded".
- Fedora (14/15): the same card at the top → **Approve**.

**What worked:** the "Who can approve" text is precise. Approving grants the home server's internet egress ("by default all its traffic goes out from here"), the home LAN, and its tailnet reach ("which may be more than its owner's own account gets"). That's exactly the subnet-router-style blast radius I'd worry about.

**Friction — identity contradiction (medium):**
- Slate, screen 02: "Your home server … must be signed in to the same Tailscale account as this router."
- Fedora, screen 14: "Others on your Tailscale network, and people you share homebox with, can ask to connect but can't approve." So the same account is not actually required.
- Fedora, screen 14, on the slate7 row: "It's signed in to alex@example.com, an account that may approve here: whoever holds this router can also open this dashboard and approve devices. If it's for someone else, sign it in to their own Tailscale account first …"
- The Flint 3 panel (11) shows no such warning for the identical request.

A travel router is the device most likely to be lost or stolen in a hotel. Do I follow the Slate's requirement (my admin account) or the Fedora warning? Would a separate user, or a tagged device, have worked? The screens disagree, and only one of the two home-server UIs flags the risk.

## Task 8 — Hotel Wi-Fi sign-in (GL repeater detected it)

**Clicks:** screen 16 shows "Repeater needs you to sign in" → **Open GL's Repeater page** → click "Login Mode for Public Hotspots" (or "Go to Login"). This leads to screen 16b ("Router connected to homebox · devices signing in"). Then I open any site on the laptop and sign in.

**What worked:** I know exactly where traffic goes:
- "everything devices here do goes out the hotel Wi-Fi directly (not only the sign-in page, and not through your home server), so your phone's data isn't used for them."
- "Only this router's own link to your home server stays up over your other connections."
- Tethering shows "Router's link only".
- "tailbond then adds Repeater back to bonding."

It also warns that tailnet machines are unreachable meanwhile ("While devices are signing in to a hotel page they go out directly, so they don't reach these until that's done").

**Friction:** screen 16 says this lasts "(until it's done, or 10 minutes)". Screen 16b drops the 10-minute limit and only says "Login Mode ends by itself once you've signed in (or click 'Quit Login Mode' …)". It shows no remaining time. The Ethernet flow (17b) shows "until 05:25 on this router's clock". So I don't know whether a cap applies in GL's Login Mode, or who enforces it. **Low.**

## Task 9 — Sign-in page on the hotel Ethernet

**Clicks:** screen 17 shows "Ethernet needs you to sign in" → **Open sign-in page** → screen 17b, "Signing in to Ethernet…", with "Open the sign-in page again" / "I've signed in, check now" / "Stop sign-in mode". The text: "For up to 9 more minutes (until 05:25 on this router's clock) … tailbond adds Ethernet back by itself once you're signed in." Good controls, and the explicit deadline is the right thing.

**Friction — before-click text understates the impact (medium):**
- Before clicking, screen 17 says: "Opening the sign-in page lets devices here reach Ethernet directly (without your home server) for up to 10 minutes, so the page can load." That reads as additive: Ethernet becomes reachable for the portal.
- After clicking, screen 17b says: "all traffic from devices here goes out Ethernet directly, not just the sign-in page, and not through your home server". Repeater, which was working and bonded, drops to "Router's link only". All tailnet machines become unreachable.

In this situation the hotel Wi-Fi bond is healthy. Clicking the button silently cuts every device off from home and the tailnet for up to 10 minutes. I'd lose an open SSH or NAS session I didn't expect to lose. The screen 16 wording for Wi-Fi ("everything devices here do goes out … directly") is the honest version, and 17 should match it.

**Friction (polish):** in the 17 banner, the small grey line "GL's Login Mode covers the repeater Wi-Fi only; tailbond handles sign-in pages on other connections like this one." sits above the main sentence. It reads like a footnote placed first. **Low.**

## Task 10 — Home side set up for best speed

**Flint 3:**
- 18 shows "2 to improve". "Tailscale: 2 machines direct, 1 through a relay (slower) … bonding itself doesn't go through Tailscale." The fix is concrete: "NETWORK → Firewall → Open Ports on Router, add UDP 41641. If another router sits in front of this one … also forward UDP port 41641". Plus "This network makes direct Tailscale connections hard … ('hard NAT') … Same fix as above".
- 18b is red, "Your internet line uses carrier-grade NAT … 100.72.3.4 … travel routers and phones away from home can't reach UDP 443 here … it can't run through Tailscale's relays". The fix is to ask the provider for public IPv4, or use IPv6 with its caveats.
- 18c: "All good · Everything is set up for best speed".

That's the right level of detail for me: ports, NAT type, which peer is relayed, and that bonding itself is independent of DERP.

**Fedora (15):** Devices tab → "Setup health: 2 things to improve" with "forward UDP port 41641 to this machine (192.168.1.2). (Turning on UPnP … also works, but lets any device … open ports.)" Good, including the UPnP trade-off.

**Friction:** the 10.77.0.0/24 route sends mixed signals. On the fresh Flint (13), it's "Optional: let your Tailscale machines reach tailbond devices. Skip this unless … Devices behind travel routers reach your Tailscale machines either way." But 12 and 18c list "Route 10.77.0.0/24 approved" as a check under "Everything is set up for best speed". As someone who approves subnet routes deliberately, I can't tell whether it affects speed or only reachability. **Low.**

---

## Findings

| # | Task | Finding | Severity |
|---|---|---|---|
| 1 | 2 / 7 | Identity contradiction. The Slate says the home server "must be signed in to the same Tailscale account as this router". The Fedora page says "Others on your Tailscale network … can ask to connect" and warns "whoever holds this router can also open this dashboard and approve devices". The Flint 3 panel (11) shows no such warning. I can't tell which account a loss-prone travel router should use. | medium |
| 2 | 9 | The pre-click text on 17, "lets devices here reach Ethernet directly … for up to 10 minutes", reads as additive. After clicking (17b), *all* traffic goes out Ethernet directly, the working Repeater bond drops to "Router's link only", and tailnet access is lost for every device. | medium |
| 3 | 1 | The Fedora install steps ("pick the file, Install.") omit the Flint's "(log in with the same password)" and "(leave the other options as they are)". LuCI shows "Allow overwriting conflicting package files". | low |
| 4 | 1 | "Download the router package" names no architecture or supported models. I first see `aarch64_cortex-a53` in LuCI, and "same tailbond this router runs" doesn't apply to an x86 Fedora host. | low |
| 5 | 3 | On the home-side path view, "Path 0 · Pixel-8 · Min 31 Ms" doesn't say which uplink each path is; "· wlb" is unexplained; "Ms" is capitalised oddly. | low |
| 6 | 4 | The data limit "restarts on the 1st of each month" with no reset-day setting, and it counts only tailbond traffic. It will drift from the carrier's bill. | low |
| 7 | 5 | "Use the full name, like nas.tail1234.ts.net": short names don't work, and it doesn't say who answers DNS for LAN clients. | low |
| 8 | 6 | Screen 10, "It may have been turned off … Starting it is safe: devices here keep their internet", conflicts with toggle-off showing screen 09 and with 08 ("devices on this router have no internet" when home is unreachable). | low |
| 9 | 8 | 16 says "(until it's done, or 10 minutes)". 16b has no time limit or countdown, unlike 17b's "until 05:25 on this router's clock". | low |
| 10 | 9 | In the 17 banner, the small grey note "GL's Login Mode covers the repeater Wi-Fi only…" comes before the main sentence and reads as a misplaced footnote. | low |
| 11 | 10 | The route 10.77.0.0/24 is "Optional … Skip this unless…" on 13, but listed under "Everything is set up for best speed" on 12/18c. It's unclear whether it affects speed. | low |

Totals: 0 high / 2 medium / 9 low
