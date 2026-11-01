# Round 10 — GL persona A (traveller)

**Who I am:** I travel a lot and use a GL.iNet Slate 7. On GL's own Tailscale page I flip one toggle and it works. I'm not a network engineer. Words like "route", "NAT" and "port" are where I start to slow down.

**Materials:** `task-script.md`, `screens.md` and the PNGs in `packet-gl/` (desktop versions, plus the phone versions of 01b, 05, 11, 14, 16 and 17). I quote on-screen text exactly.

---

## Task 1 — Install tailbond on the Slate 7 without SSH

**Where I click:**
- I downloaded the file from the Fedora page. Screen 15, Devices tab, has a card "Add a GL.iNet travel router (recommended)" with a "Download the router package" button. Steps 2–3 are written out: "On the travel router's admin page: SYSTEM → Advanced Settings → Go To LuCI, then System → Software → Upload Package…, pick the file, Install."
- On the Slate 7 (01a): SYSTEM → Advanced Settings → "Go To LuCI". The hint "Use the same admin password to log in" tells me which password to use.
- In LuCI (01b): Upload Package…, then the "Manually install package" dialog shows `tailbond_0.1.0-r42_aarch64_cortex-a53_neon-vfpv4.ipk (6.1 MB)`. I click **Install**.

**Where I hesitate:**
- The dialog has a checkbox "Allow overwriting conflicting package files". The Flint 3 instructions (11–13) say "(leave the other options as they are)". The Fedora page I actually downloaded from (15) doesn't say that. I'd stop and wonder whether I need to tick it. → **low**
- Nothing on either download card says the package fits my Slate 7. Fedora just says "the router package". Flint says "It's the same tailbond this router runs", but my router isn't a Flint 3. The file name (`aarch64_cortex-a53…`) means nothing to me. I'd install it and hope. → **low**
- LuCI is unfamiliar ground compared with GL's APPLICATIONS → Plug-ins, which is where I'd look for add-ons. The steps are precise enough that I get through. No screen shows what LuCI says after Install, so I just trust that it worked and go back to the GL panel. → **low** (expectation only)

## Task 2 — Pair with the home server

**Where I click:** VPN → tailbond (02: "Looking for your home server on your tailnet…"). Then 03: "Found your home server: homebox", and I click **Ask to connect**. 04 says "Waiting for approval on homebox… On homebox, tap Approve next to slate7. This router connects by itself once approved; you can close this page." It also says where to approve: "open http://homebox:8077 … At home with a Flint 3, its admin panel → VPN → tailbond works as well."

This is close to the GL Tailscale feeling. The error states are just as clear: 03b "Turn on Tailscale first" with "Open Tailscale settings", and 03c "No home server found on your tailnet … Checked 4 online machines".

**Small thing:** Every pairing screen has a "Make this the home server (you can switch back here)" link at the bottom. I don't need it, but I'd keep away from it. Thankfully it says I can switch back. → **low**

## Task 3 — Hotel Wi-Fi + phone tethering, confirm both used

**Where I look:** 05. The top card says "Connected to homebox — Using Repeater (Wi-Fi) + Tethering (phone)", with "↓ 59 Mbps". The Repeater and Tethering cards both show a green "In use" badge with their own speeds (38 + 21 Mbps), and those add up to the total. Done, no confusion.

06 (phone trouble) is also clear: "Tethering (phone) not working right now" and "It's connected, but can't reach homebox. tailbond keeps trying; the other connections carry your traffic."

No findings.

## Task 4 — Phone as backup only

**Where I click:** The hint under Speed or reliability says exactly what to do: "Saving your phone's data? Set Tethering or Cellular to "Backup only" under Internet connections." So I click **Backup only** on the Tethering card. 07 then shows "Backup · saving data" and "Used only if your other connections stop working. Until then tailbond sends nothing over it (GL itself may check the connection now and then, a few KB)." The top card says "Tethering (phone) on standby". Very reassuring.

07b "Change limit": "5 GB a month … the count restarts on the 1st of each month. 0 = no limit."

**Small thing:** My phone plan doesn't reset on the 1st, and I can't choose the day. → **low**

## Task 5 — Reach the NAS by its MagicDNS name

**Where I look:** On 05, the "Tailnet machines" list shows **nas** with `nas.tail1234.ts.net`, `100.101.2.2`, "Online · direct" and a "Copy name" button. The Home server card says "Tailnet names: Work on every device connected to this router while tailbond is on. Use the full name, like nas.tail1234.ts.net."

**Where I'd trip without that line:** With Tailscale on my laptop I normally just type `nas`. Here I have to use the full `nas.tail1234.ts.net`. The page says so twice and has a copy button, so I'd get it, but my first try would be `nas`. → **low**

## Task 6 — Turn tailbond off

**Where I click:** The blue toggle at the top right of the status card. 09 says "tailbond is off — Devices on this router use the internet directly, without your home server…" It also shows a good warning: "While tailbond is off, GL may use Tethering (your phone's data) … To be sure the phone's data isn't used, unplug it or turn tethering off on it", with a "GL's INTERNET page" button. That's exactly the thing I'd worry about.

**Small things:**
- On 09, with tailbond off, the Home server card still shows "Ask to connect again" next to "Unpair". I'm not sure whether "Ask to connect again" turns tailbond back on, re-pairs, or does nothing while it's off. I'd use the toggle instead. → **low**
- I can see a red "Unpair" button next to it, but the confirm dialog (09b) is clear: "This router stops using tailbond and forgets the home server … it will need to approve". No risk of doing it by accident. No finding.

## Task 7 — Approve the travel router at home

**Flint 3 (11):** A "Waiting for your approval · 1 new" card at the top says "slate7 wants to connect — Travel router on your tailnet · signed in as alex@example.com · GL-BE3600", with **Decline / Approve** buttons. After I click Approve, 12 shows slate7 under Devices: "Connected just now · 2 connections bonded". Very clear.

**Fedora (14, desktop + phone):** The same "Waiting for your approval" card is at the very top, with "slate7 wants to connect" and **Approve**. Easy.

**Small things:**
- The paragraph above the Fedora card is dense: "anyone logged in on homebox itself (http://127.0.0.1:8077) … under homebox's access rules." On the phone (14-phone) it fills the whole first screen before I reach the slate7 line. I skip it. → **low**
- The Fedora header has a toggle labelled "Connected" (14/15). For a home server I'd expect "On" or "Running". "Connected" to what? → **low**

## Task 8 — Hotel Wi-Fi with a sign-in page (GL repeater)

**Where I click:** 16 says "Repeater needs you to sign in — GL's Repeater page handles this sign-in. Open it and click "Login Mode for Public Hotspots" (or "Go to Login"). Then open any website on your laptop or phone". I click **Open GL's Repeater page** and start Login Mode, which I already know from GL. 16b then shows "Router connected to homebox · devices signing in" and "Login Mode ends by itself once you've signed in … tailbond then adds Repeater back to bonding." That's what I want to hear.

**Small things:**
- The 16b explanation is one long, dense paragraph. One part confused me for a moment: "Only this router's own link to your home server stays up over your other connections (the top card; it carries almost nothing meanwhile)". "The top card" could mean the status card or the first card under Internet connections (Ethernet). → **low**
- No screen shows the moment after sign-in, when Repeater goes back to "In use". I'd assume it ends up looking like 05. → **low** (can't verify)

## Task 9 — Sign-in page on the hotel's Ethernet cable

**Where I click:** 17 says "Ethernet needs you to sign in", with a big **Open sign-in page** button. 17b then shows "Signing in to Ethernet…", "For up to 9 more minutes (until 05:17 on this router's clock)", and the buttons "Open the sign-in page again", "I've signed in, check now" and "Stop sign-in mode". Clear, and better than GL's flow.

**Small things:**
- In the 17 warning box, the first line is small text: "GL's Login Mode covers the repeater Wi-Fi only; tailbond handles sign-in pages on other connections like this one." It comes before the larger sentence that explains the problem. I read the caveat before I know what's going on. It's useful (it stops me running off to GL's Repeater page like in task 8), but the order and the mixed font sizes feel off. → **low**
- "until 05:17 on this router's clock". I don't know whether the router's clock is on my time zone. A countdown alone would do. → **low**

## Task 10 — Home side set up for best speed

**Flint 3:**
- 13 (fresh): "Home server is on · 2 setup steps left".
  - Step 2 (Dynamic DNS) is easy: "turn on APPLICATIONS → Dynamic DNS and paste the name it shows here (it looks like abc1234.glddns.com)". That's GL language I know.
  - Step 1 sends me out of GL, into the Tailscale admin console: "open **homebox**, choose **Edit route settings**, tick the one route listed (10.77.0.0/24) and save." There's an "Open admin console" button, but I've never needed that console with GL's one-toggle Tailscale. "Route" and "10.77.0.0/24" are jargon to me.
  - The consequence text doesn't help me decide whether it matters: "Until then, devices connected through tailbond can still reach your Tailscale machines; only the other direction (your machines reaching those devices) waits for this." I'd do it because the header counts it as a step left, but with real effort and some doubt. → **medium**
- 18c: "All good — Everything is set up for best speed", with five green checks. That's what I'm looking for.
- 18 (relayed): "2 things to improve for best speed (tailbond works meanwhile, just slower)". The fix is GL language: "On this router: NETWORK → Firewall → Open Ports on Router, add UDP 41641." Good.
  - The second item, "This network makes direct Tailscale connections hard … ("hard NAT") … no port mapping on the router", is jargon. Luckily it ends with "Same fix as above (one change fixes both)". → **low**
  - The green "Setup done" box sits right above the amber "2 to improve" Setup health card. Mixed message: is it done or not? → **low**
- 18b (CGNAT): "Step 1: Your internet line uses carrier-grade NAT". The heading is jargon, but the body is plain: "Ask your provider for a public IPv4 address (often free or a small monthly fee; say it's for reaching your home from outside)". "Not sure? Show this to whoever set up your internet." I can't fix this in the panel, but I know exactly what to ask my ISP. → **low**

**Fedora (15):** "Setup health: 2 things to improve (tailbond works meanwhile)". It says to forward UDP 41641 on my home router ("Port forwarding, also called Virtual server or NAT") and adds "Same fix as above (one forward fixes both)". That's fine. Unlike the Flint, this screen only lists the problems, so I can't see whether everything else (route, port 443) is OK. → **low**

---

## Findings

| # | Task | Finding | Severity |
|---|---|---|---|
| 1 | 1 | LuCI dialog checkbox "Allow overwriting conflicting package files". The Fedora steps (15) don't say to leave it unticked (the Flint steps do: "leave the other options as they are"). | low |
| 2 | 1 | No reassurance that the download fits a Slate 7. Fedora: "the router package". Flint: "the same tailbond this router runs" (a Flint 3). File name `aarch64_cortex-a53_neon-vfpv4` means nothing to me. | low |
| 3 | 1 | Installing through LuCI instead of GL's own plug-in area. Works, but feels foreign. No screen confirms the install succeeded. | low |
| 4 | 2 | "Make this the home server" link on every travel-router pairing screen. It's an unneeded option I might click by mistake (it does say I can switch back). | low |
| 5 | 4 | Data limit "restarts on the 1st of each month". Can't match my phone plan's billing day. | low |
| 6 | 5 | Short name `nas` won't work; it must be "the full name, like nas.tail1234.ts.net". Clearly stated, but different from my Tailscale habit. | low |
| 7 | 6 | With tailbond off, "Ask to connect again" still shows next to "Unpair". Unclear what it does compared with the toggle. | low |
| 8 | 7 | Fedora approval card has a dense "Who can approve … (http://127.0.0.1:8077) … access rules" paragraph. On the phone it pushes the slate7 line below the first screen. | low |
| 9 | 7 | Fedora header toggle is labelled "Connected" for a home server. Connected to what? | low |
| 10 | 8 | In 16b, "(the top card; it carries almost nothing meanwhile)": unclear which card "the top card" means. The paragraph is long. | low |
| 11 | 8 | No screen shows Repeater going back to "In use" after sign-in. I have to assume it looks like 05. | low |
| 12 | 9 | In 17, the small-print caveat "GL's Login Mode covers the repeater Wi-Fi only…" comes before the main explanation, in mixed font sizes. | low |
| 13 | 9 | "until 05:17 on this router's clock". Is the router's time zone mine? | low |
| 14 | 10 | Flint Step 1 sends me to the Tailscale admin console to "Edit route settings, tick the one route listed (10.77.0.0/24)". That's jargon and outside GL. The consequence ("only the other direction … waits for this") doesn't tell me whether I need it. | medium |
| 15 | 10 | "hard NAT" / "no port mapping on the router" health item is jargon. Saved by "Same fix as above". | low |
| 16 | 10 | Green "Setup done" shown alongside amber "2 to improve" Setup health (18): mixed signal. | low |
| 17 | 10 | "carrier-grade NAT" heading is jargon (the body explains it well). It can only be fixed by calling the ISP. | low |
| 18 | 10 | Fedora health (15) lists only the problems, so I can't confirm the remaining checks are fine. | low |

Totals: 0 high / 1 medium / 17 low
