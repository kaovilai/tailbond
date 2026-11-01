# Round 11 — GL persona A (traveller)

Persona: I travel a lot and use GL.iNet travel routers. I'm used to GL's own Tailscale page: one toggle and it works. I'm not a network engineer. I know the GL admin menus (INTERNET, VPN, APPLICATIONS → Dynamic DNS, Repeater and its "Login Mode for Public Hotspots"). I had never opened LuCI before.

Materials: the GL packet only (task-script.md, screens.md, PNGs). I looked at every relevant desktop screenshot and checked the phone versions of 01b, 05, 11 and 17b.

---

## Task 1 — Install tailbond on the Slate 7 without SSH

**Where I click:**
- Fedora page (15), Devices tab, card "Add a GL.iNet travel router (recommended)": I click **Download the router package**. The Flint panel (11–13) has the same card.
- Slate 7 GL admin (01a): SYSTEM → Advanced Settings → **Go To LuCI**. It says "Use the same admin password to log in."
- LuCI (01b): System → Software → **Upload Package…**. I pick `tailbond_0.1.0-r42_aarch64_cortex-a53_neon-vfpv4.ipk (6.1 MB)`, then the dialog "Really install the uploaded package?" → **Install**.
- Then I go back to GL admin. 02 shows VPN → **tailbond** in the GL sidebar, right next to OpenVPN and WireGuard, which feels native.

**Where I hesitate:**
- The LuCI dialog has an unticked checkbox, "Allow overwriting conflicting package files". The Flint's instructions say "(leave the other options as they are)" and "The router's internet keeps working". The Fedora page's step 2 only says "pick the file, Install", so a Fedora user doesn't get that reassurance. I'd wonder whether I'm supposed to tick it. *(low)*

**Expected vs. got:** Three numbered steps whose menu names match the screens word for word. That's more than GL's own Tailscale (built in), but it's doable without SSH. Fine.

## Task 2 — Pair the Slate 7 with the home server

**Where I click:** 02 shows "Looking for your home server on your tailnet…". A few seconds later, 03 shows "Found your home server: homebox" with `homebox.tail1234.ts.net`. I click **Ask to connect**. 04 shows "Waiting for approval on homebox…" and "This router connects by itself once approved; you can close this page."

The fallback screens are clear. 03b says "Turn on Tailscale first" and has an "Open Tailscale settings" button. 03c says "No home server found on your tailnet" and has "Search again" plus a name field.

**Issues:** None. This felt like the GL Tailscale experience: it found the server by itself and needed one button.

## Task 3 — Hotel Wi-Fi (Repeater) + phone USB tethering, confirm both are used

**Where I look:** 05, the top card. "Connected to homebox — Using Repeater (Wi-Fi) + Tethering (phone)". Under Internet connections, both Repeater ("↓ 38 Mbps ↑ 4.2 Mbps · 24 ms") and Tethering ("↓ 21 Mbps ↑ 2.9 Mbps · 48 ms") show a green "In use", and they add up to the "↓ 59 Mbps" at the top. That convinces me both are being used. 06 shows Tethering with a red "Not working right now" and the plain explanation "the other connections carry your traffic". Good.

**Issues:**
- In the top line "↓ 59 Mbps ↑ 7.1 Mbps Up 2 h 13 min", the word "Up" sits right after the ↑ upload arrow, and I first read it as an upload figure. It's uptime. *(low)*

## Task 4 — Phone as backup only

**Where I click:** On the Tethering card, "Use it" → **Backup only**. The hint under "Speed or reliability" points to the same control: "Saving your phone's data? Set Tethering or Cellular to "Backup only"". In 07 the badge changes to "Backup · saving data", with the text "Used only if your other connections stop working. Until then tailbond sends nothing over it (GL itself may check the connection now and then, a few KB)." The top line says "Tethering (phone) on standby". 07b shows the "Change limit" editor ("GB a month", "0 = no limit"), and it explains that it counts only tailbond's traffic and resets on the 1st.

**Issues:** None. The change is confirmed by the badge and the top line.

## Task 5 — Reach the NAS by its MagicDNS name from the laptop

**Where I look:** 05, the "Tailnet machines" card. The first entry is **nas**, `nas.tail1234.ts.net`, `100.101.2.2`, "Online · direct", with a **Copy name** button. The Home server card says "Tailnet names: Work on every device connected to this router while tailbond is on. Use the full name, like nas.tail1234.ts.net."

**Issues:**
- From Tailscale habit I'd type just `nas`. The page says to use the full name, so I'll copy it. But the line "reach these through the bond to homebox (not this router's own Tailscale)" puzzles me. My Slate's own Tailscale is on, so which one am I using? It doesn't stop me, because Copy name gives the answer. *(low)*

## Task 6 — Turn tailbond off

**Where I click:** The big toggle at the top right of the status card. That's exactly the GL Tailscale habit. 09 shows "tailbond is off — Devices on this router use the internet directly…". A yellow warning says "While tailbond is off, GL may use Tethering (your phone's data)", with a "GL's INTERNET page" button. That warning is genuinely useful to me. **Unpair** is a separate red button with a clear confirm (09b: "This router stops using tailbond and forgets the home server…"), so I won't mix the two up.

**Issues:**
- In 09, the Repeater and Tethering cards change to "This choice applies when tailbond is on…". The Ethernet card still says "It's used automatically once connected." That's slightly inconsistent while everything is off. *(low)*
- Screen 10 ("tailbond isn't running on this router — It may have been turned off or stopped after an error") uses "turned off", the same words as the normal toggle-off state in 09. If I'd just flipped the toggle, I'd wonder whether I broke something. *(low)*

## Task 7 — Approve the travel router at home

**Flint 3 (11):** A "Waiting for your approval — 1 new" card at the top: "slate7 wants to connect — Travel router on your tailnet · signed in as alex@example.com · GL-BE3600 · 100.101.9.9 · asked 1 min ago". I click **Approve**. In 12, slate7 appears under Devices ("Connected just now · 2 connections bonded"). Easy. The phone view (11-phone) is also fine: Approve is a big button.

**Fedora (14/15):** The same card at the top: "slate7 wants to connect", **Approve**.

**Where I hesitate:**
- On the Fedora page only, the request carries an orange warning: "It's signed in to alex@example.com, an account that may approve here: whoever holds this router can also open this dashboard and approve devices. If it's for someone else, sign it in to their own Tailscale account first, then let it ask again." It's my own router, but this reads like a security alarm, and nothing says "if this router is yours, just approve". I'd stop and reread it, and might decline to be safe. The Flint's card for the same router signed in to the same account has no such warning, so I can't tell which page to believe. *(medium)*
- The "Who can approve" paragraph (11, 14, 15) is dense and abstract: "they allow it whatever they allow homebox, which may be more than its owner's own account gets." I skipped it. *(low)*

## Task 8 — Hotel Wi-Fi sign-in page (GL's repeater detected it), with tailbond on

**Where I click:** 16 has a blue card, "Repeater needs you to sign in — GL's Repeater page handles this sign-in. Open it and click "Login Mode for Public Hotspots" (or "Go to Login")." I click **Open GL's Repeater page** and start Login Mode. Back on tailbond (16b): "Signing in to Repeater through GL's Login Mode — Now open any website on your laptop or phone: the hotel's sign-in page appears there…" and "Login Mode ends by itself once you've signed in… tailbond then adds Repeater back to bonding." Meanwhile Tethering shows "Router's link only", which is reassuring about phone data. It hands me back to the GL feature I already know, which is what I wanted.

**Issues:**
- The Ethernet sign-in flow (17b) has an "I've signed in, check now" button. The Wi-Fi Login Mode flow (16b) only has "Back to GL's Repeater page", so after signing in I just wait and hope. The wording "everything devices here do goes out the hotel Wi-Fi directly" (16, 16b) is also clumsy. *(low)*

## Task 9 — Sign-in page on the hotel's Ethernet cable

**Where I click:** 17 has an orange card, "Ethernet needs you to sign in", and the Ethernet card has a "Needs sign-in" badge. The card explains "GL's Login Mode covers the repeater Wi-Fi only; tailbond handles sign-in pages on other connections like this one." I click **Open sign-in page**. In 17b the hotel page opens in a new tab. tailbond shows "Signing in to Ethernet…" with **Open the sign-in page again**, **I've signed in, check now** and "Stop sign-in mode". "tailbond adds Ethernet back by itself once you're signed in." Clear.

**Issues:**
- In 17 the small-print line about GL's Login Mode comes *before* the main sentence ("It shows a sign-in page before it lets anyone online…"). My eye lands on the fine print first. "For up to 9 more minutes (until 05:25 on this router's clock)" makes me wonder whether the router's clock is on hotel time or home time. *(low)*

## Task 10 — At home, make sure everything is set up for best speed

**Flint 3:**
- **18c:** "Setup health — All good — ✓ Everything is set up for best speed". It lists 3 machines direct, route approved, public address, and "UDP 443 reached from the internet". That's what I want to see.
- **13 (fresh):** "Home server is on · 1 setup step left". "Step 1: tell travel routers where home is" points me to APPLICATIONS → Dynamic DNS, which I know from GL, and says to paste the `…glddns.com` name. The route box is clearly marked "Optional… Skip this unless…". Good.
- **18 (relay):** "Setup health — 2 to improve". **I get stuck here:**
  - The fix reads "For whoever manages this router: NETWORK → Firewall → Open Ports on Router, add UDP 41641." I *am* the one managing this router, and I'm logged in to its admin panel. tailbond says UDP 443 was "opened on this router automatically", so why can't it open 41641 too, or at least give me a button? Instead I have to navigate GL's firewall page myself.
  - The same page shows a green "Setup done" box above "2 things to improve", which is mixed messaging.
  - The relay concerns only `apple-tv` ("via relay (slower)"), and the text adds "The cause can also be on that machine's own network". So I might open the port and still see the warning, with no way of knowing whether it helped. The second item, "This network makes direct Tailscale connections hard… ("hard NAT")", is jargon, though "Same fix as above" helps. *(medium)*
- **18b (CGNAT):** "Step 1: Your internet line uses carrier-grade NAT… Ask your provider for a public IPv4 address (often free or a small monthly fee…)". I understand I have to phone my ISP, and "Not sure? Show this to whoever set up your internet" helps. The headline jargon ("carrier-grade NAT") still made me nervous at first. This can't be fixed in the panel, but it's honest. *(low)*

**Port forwarding, both home servers:**
- On the Flint, the Port row says: "If it's plugged into another box (your internet provider's modem-router), that box must forward UDP 443 to this router; see the box's manual under 'port forwarding'."
- The Setup done box says "The first connection from away from home confirms the port works."
- Many people (me included) have the Flint behind an ISP modem. I can't confirm from home that the forward works. On the freshly set-up page (13) there is no Setup health check at all, so I'd only find out at the hotel, the worst place to discover it. *(medium)*

**Fedora (15):** The Devices tab shows "Setup health: 2 things to improve (tailbond works meanwhile)", with "On your home router's admin page (Port forwarding, also called Virtual server or NAT), forward UDP port 41641 to this machine (192.168.1.2)" and a "How to fix" link. That's clearer than the Flint wording because it names the machine's IP, though it's still a manual job on another device.

---

## Findings

| # | Task | Finding | Severity |
|---|---|---|---|
| 1 | 7 | Fedora approval card warns "whoever holds this router can also open this dashboard and approve devices. If it's for someone else, sign it in to their own Tailscale account first…". It has no "if it's yours, approve" reassurance, and the Flint's card for the same router has no such warning. I hesitated and might decline. | medium |
| 2 | 10 | Flint relay fix "For whoever manages this router: NETWORK → Firewall → Open Ports on Router, add UDP 41641": I'm the one managing it, and UDP 443 was "opened on this router automatically", but there's no button for 41641. "Setup done" shows next to "2 to improve". The relay is only apple-tv and "The cause can also be on that machine's own network", so I can't tell whether the fix will help. | medium |
| 3 | 10 | Behind an ISP modem I can't confirm the port forward from home: "The first connection from away from home confirms the port works". The fresh page (13) has no health check, so I'd discover a broken forward at the hotel. | medium |
| 4 | 1 | Fedora install step says only "pick the file, Install". Unlike the Flint ("leave the other options as they are"), it doesn't address LuCI's "Allow overwriting conflicting package files" checkbox. | low |
| 5 | 3 | "↓ 59 Mbps ↑ 7.1 Mbps Up 2 h 13 min": "Up" next to the upload arrow reads as an upload figure, but it's uptime. | low |
| 6 | 5 | "Use the full name, like nas.tail1234.ts.net" clashes with my MagicDNS short-name habit, and "(not this router's own Tailscale)" is puzzling when the Slate's Tailscale is on. Copy name resolves it. | low |
| 7 | 6 | When tailbond is off, the Ethernet card still says "It's used automatically once connected" while the other cards say "This choice applies when tailbond is on". | low |
| 8 | 6 | Screen 10 "It may have been turned off or stopped after an error" overlaps with the normal toggle-off wording in 09 ("tailbond is off"). | low |
| 9 | 7 | The "Who can approve" paragraph is dense jargon ("they allow it whatever they allow homebox, which may be more than its owner's own account gets"). | low |
| 10 | 8 | The GL Login Mode flow (16b) has no "I've signed in, check now" button, unlike the Ethernet flow (17b). "everything devices here do goes out the hotel Wi-Fi directly" is clumsy. | low |
| 11 | 9 | In 17 the small-print GL Login Mode note comes before the main sentence. "until 05:25 on this router's clock" raises a which-timezone doubt. | low |
| 12 | 10 | The CGNAT headline "Your internet line uses carrier-grade NAT" is jargon. The text explains it (call the ISP), but it's alarming at first sight. | low |

Totals: 0 high / 3 medium / 9 low
