# Round 10 — GL persona C (family member)

**Who I am:** someone else in the family set up the home server. I was handed a
GL.iNet Slate 7 and asked to "get it working". I know Wi-Fi, passwords and my
phone's hotspot. I don't know routes, ports or NAT. I only had the task script,
the manifest and the screenshots (desktop, plus phone versions of 01a, 01b, 05,
14, 16 and 17b).

**Overall:** I could do every task from the screens alone. The travel-router
page talks my language most of the time: "Using Repeater (Wi-Fi) + Tethering
(phone)", "Backup only", "Copy name", "Turn tailbond off". I got stuck where the
text asks me to do something I can't (approve from "your Tailscale account",
forward a port) or where one sentence points at something vague ("the top
card"). No high-severity findings.

---

## Task 1: Install tailbond on the Slate 7 without SSH

**Where I'd click**
- On the home server's page (15, Devices tab), the "Add a GL.iNet travel router
  (recommended)" box: **Download the router package**. (On the Flint 3, 11–13:
  the blue **Download the router package** button.)
- Slate 7 admin (01a): SYSTEM → Advanced Settings → **Go To LuCI**. The page says
  "Use the same admin password to log in."
- LuCI (01b): System → Software → **Upload Package…**, pick the file, then
  **Install** in "Manually install package — Really install the uploaded package?".

**Where I'd hesitate**
- The LuCI dialog has a checkbox, "Allow overwriting conflicting package files".
  The Flint's instructions cover it: "click **Install** (leave the other options
  as they are)". The Fedora page (15) only says "pick the file, Install." and
  also leaves out "(log in with the same password)". With only the Fedora
  page, I'd wonder whether to tick the box. (low)
- The Fedora page opens on the **Connections** tab (14). The download is under
  **Devices** (15), so the person who sent me there has to say which tab.
  The file is named `tailbond_0.1.0-r42_aarch64_cortex-a53_neon-vfpv4.ipk`, and
  the Flint says "It's the same tailbond this router runs". My router is a
  different model (GL-BE3600, not GL-BE9300), and nothing tells me this file
  fits the Slate 7. I'd install it anyway and hope. (low)
- No screen shows what happens after clicking Install: success, an error, or
  how long it takes. Step 3 just says "Back on the travel router's admin page,
  open VPN → tailbond". I'd guess, correctly, that the menu item appearing
  means it worked. (low)
- The LuCI screens look nothing like GL's admin (different colours, top menu
  instead of side menu). It was a bit scary, but the step text matches the
  labels exactly ("System", "Software", "Upload Package…"), so I could follow it.

## Task 2: Pair the Slate 7 with the home server

**Where I'd click**
- 02: "Looking for your home server on your tailnet…" A few seconds later (03):
  "Found your home server: homebox" → **Ask to connect**.
- If Tailscale is off (03b), "Turn on Tailscale first" with **Open Tailscale
  settings** is clear. If nothing is found (03c), "Search again" or typing the
  name is clear.

**Where I'd get stuck**
- 04: "Waiting for approval on homebox… On homebox, tap **Approve** next to
  **slate7**. … From anywhere: open http://homebox:8077 on any device signed in
  to your Tailscale account (a phone works too)." I'm not the one who set up
  homebox. I don't know whether my phone is "signed in to your Tailscale
  account", or whose account that is. The home server's own screen (14) names
  the approver: "Who can approve: alex@example.com". The Slate's waiting card
  doesn't name anyone and doesn't say "or ask the person who runs homebox". I'd
  sit here wondering whether I'm supposed to do something, then phone the
  family member. (medium)
- "tailnet" in "Looking for your home server on your tailnet…" and "No home
  server found on your tailnet" isn't a word I know. Its meaning is clear
  enough from context ("signed in to the same Tailscale account"). (low)
- The footer link "Setting up the router that stays at home? Make this the home
  server" sits right under the pairing box. I read it twice to make sure it
  wasn't for me. It's worded well enough that I wouldn't click it.

## Task 3: Hotel Wi-Fi (Repeater) + phone over USB, confirm both are used

**Where I'd look:** 05. The top card says "Connected to homebox — Using
Repeater (Wi-Fi) + Tethering (phone)". Under Internet connections, both
**Repeater** and **Tethering** show a green "In use" with their speeds. That's
exactly the confirmation I wanted. On the phone screen it's the same and easy
to read.

- The top row reads "↓ 59 Mbps ↑ 7.1 Mbps Up 2 h 13 min". After two arrows,
  "Up" first read to me as an upload number. It's how long it has been
  connected. (low)
- 06 (phone trouble): Tethering shows "Not working right now — It's connected,
  but can't reach homebox. tailbond keeps trying; the other connections carry
  your traffic." Reassuring and clear.

## Task 4: Phone as backup only

**Where I'd click:** the Tethering card → Use it: **Backup only** (07). The badge
changes to "Backup · saving data" and the text says "Used only if your other
connections stop working. Until then tailbond sends nothing over it (GL itself
may check the connection now and then, a few KB)." The top card says
"Tethering (phone) on standby". There's even a hint under Speed or reliability:
"Saving your phone's data? Set Tethering or Cellular to 'Backup only'". The
limit (07b): "5 GB a month … 0 = no limit" is clear.

- Under "Speed or reliability", "Most reliable — Sends everything over every
  connection. Uses the most data." I can't tell whether choosing that would
  start using my backup phone too. I'd leave it on Fastest. (low)

## Task 5: Reach the NAS by name from the laptop; find the name

**Where I'd look:** "Tailnet machines" on the Slate 7's page (05). The **nas**
row shows `nas.tail1234.ts.net` with a **Copy name** button. The Home server
box also says: "Tailnet names — Work on every device connected to this router
while tailbond is on. Use the full name, like nas.tail1234.ts.net." Found it.

- The task says "MagicDNS name". The screen never uses that word, it says "full
  name". For me "full name" is better. Each machine shows three things (nas,
  nas.tail1234.ts.net, 100.101.2.2). The intro line, "(not this router's own
  Tailscale) … 'Direct' / 'via relay' is how homebox reaches each machine", is
  over my head, but I didn't need it. (low)

## Task 6: Turn tailbond off

**Where I'd click:** the blue switch on the top card (05). Afterwards (09) it
says "tailbond is off — Devices on this router use the internet directly…" A
useful warning follows: "While tailbond is off, GL may use Tethering (your
phone's data) … unplug it or turn tethering off on it". I wouldn't have thought
of that.

- The switch has no "On/Off" or "Connected" label next to it. The Fedora page's
  switch has "Connected". I found it because it's the only switch at the top.
  When it's off, "Ask to connect again" still shows below, and I could mistake
  that for "turn it back on". (low)
- "Unpair" is clearly not the off switch: the confirmation (09b) says "This
  router stops using tailbond and forgets the home server … it will need to
  approve". Good.

## Task 7: Approve the new travel router at home

**Flint 3 (11):** at the top, "Waiting for your approval · 1 new — slate7 wants
to connect — Travel router on your tailnet · signed in as alex@example.com ·
GL-BE3600" → **Approve**. GL-BE3600 matches the label on the box, so I know
it's mine. Afterwards (12): "slate7 — Connected just now · 2 connections
bonded". Easy.

**Fedora (14):** same card, **Approve** button. Fine on the phone too.

- Above the button, both places have a long "Who can approve: … Others on your
  Tailscale network, and people you share homebox with, can ask to connect but
  can't approve. Approving lets the device use homebox's internet connection
  (by default all its traffic goes out from here)…" paragraph. I skipped it.
  On the Fedora page, the rest ("Path 0 · Pixel-8 · Min 31 Ms", "7.5 GB to ·
  858 MB from devices") means nothing to me, but I didn't need it. (low)

## Task 8: Hotel Wi-Fi has a sign-in page (GL's repeater detected it)

**Where I'd click:** 16. "Repeater needs you to sign in — GL's Repeater page
handles this sign-in. Open it and click 'Login Mode for Public Hotspots' (or
'Go to Login'). Then open any website…" → **Open GL's Repeater page**, then that
button on GL's page. Then 16b: "Now open any website on your laptop or phone:
the hotel's sign-in page appears there."

**Where I got confused**
- 16b: "Until you've signed in, everything devices here do goes out the hotel
  Wi-Fi directly … so your phone's data isn't used for them. Only this router's
  own link to your home server stays up over your other connections (the top
  card; it carries almost nothing meanwhile)." I don't know what "the top card"
  is: the status box at the top? The first box under Internet connections? And
  the Tethering box right below says "In use" with "Always" selected, while
  this paragraph says my phone's data isn't used. I'd worry my phone data is
  draining. The first sentence ("Now open any website…") was all I needed, but
  the rest made me unsure. (medium)
- The key button is on GL's own Repeater page, which I didn't see. The hint
  names it two ways, "'Login Mode for Public Hotspots' (or 'Go to Login')",
  which suggests the writer isn't sure either. Once signed in, there's no "I've
  signed in, check now" button like the cable sign-in has (17b), only "Login
  Mode ends by itself once you've signed in (or click 'Quit Login Mode' on GL's
  Repeater page)". So hotel Wi-Fi and hotel cable work two different ways. The
  only explanation is a small line on 17: "GL's Login Mode covers the repeater
  Wi-Fi only; tailbond handles sign-in pages on other connections like this
  one." (low)

## Task 9: Sign-in page on the hotel's Ethernet cable

**Where I'd click:** 17. "Ethernet needs you to sign in" → **Open sign-in page**.
17b: "Signing in to Ethernet… Sign in on the page that opened (accept the terms,
enter the room number…)", with **Open the sign-in page again**, **I've signed
in, check now** and **Stop sign-in mode**. This was the clearest of the
sign-in flows.

- "For up to 9 more minutes (until 05:17 on this router's clock)": I don't know
  whether the router's clock matches mine. The "9 more minutes" is what I'd go
  by. (low)
- On 17, the first thing under the heading is a small, technical line ("GL's
  Login Mode covers the repeater Wi-Fi only; tailbond handles sign-in pages on
  other connections like this one."). Only after it comes the plain sentence
  ("It shows a sign-in page before it lets anyone online…"). I'd want the plain
  sentence first. (low)

## Task 10: At home, make sure everything is set up for best speed

**Flint 3, good case (18c):** "Setup health — All good — ✓ Everything is set up
for best speed". Done, nothing for me to do.

**Flint 3, relay case (18):** "2 things to improve for best speed (tailbond
works meanwhile, just slower)". The fix: "On this router: NETWORK → Firewall →
Open Ports on Router, add UDP 41641. If another router sits in front of this
one (for example your internet provider's box), also forward UDP port 41641 to
this router there." The second item says "('hard NAT') … no port mapping on the
router". I can't do this myself. I don't know what UDP 41641 is, or whether
"another router sits in front". The Fedora version (15) is the same ("Port
forwarding, also called Virtual server or NAT … UPnP"). There is a hand-off line
that helped: "Not sure how? Show this card to whoever set up your internet."
So I'd screenshot it and send it. I couldn't finish it alone. (medium)

- On 18, the box above says, in green, "Setup done — Travel routers and phones
  can connect from anywhere…". Further down, "Setup health" says "2 to
  improve". If I only read the top I'd think everything is fine and stop. On
  18c (All good), the Port row still says "If it's plugged into another box …
  that box must forward UDP 443 to this router". That made me doubt the "All
  good". (low)
- Fresh Flint (13), "Step 1: allow tailbond devices on your Tailscale network …
  tick the one route listed (10.77.0.0/24)". "Route" is a word I don't know,
  but it says "Done once, by whoever manages your Tailscale account", so I know
  it's not my job. Step 2 (Dynamic DNS) is doable: it links to APPLICATIONS →
  Dynamic DNS and says what the name looks like. (low)
- Carrier-grade NAT (18b) is red and scary ("1 to fix"). The advice is
  something I can actually do: "Ask your provider for a public IPv4 address
  (often free or a small monthly fee; say it's for reaching your home from
  outside)". That's good, so not a finding.

---

## Findings

| # | task | finding | severity |
|---|---|---|---|
| 1 | 2 | 04 "Waiting for approval": "open http://homebox:8077 on any device signed in to your Tailscale account". It doesn't name the approver (alex@example.com, as 14 does) or say "ask whoever runs homebox", so a family member doesn't know who is supposed to approve. | medium |
| 2 | 8 | 16b: "(the top card; it carries almost nothing meanwhile)": "the top card" is unclear, and Tethering shows "In use"/"Always" while the text says "your phone's data isn't used". I'd worry about phone data. | medium |
| 3 | 10 | 18 / 15 health fixes ("NETWORK → Firewall → Open Ports on Router, add UDP 41641", "hard NAT", "port mapping", "UPnP") are beyond me. The "Show this card to whoever set up your internet" hand-off helps, but I can't finish it alone. | medium |
| 4 | 1 | The Fedora install steps (15) leave out "(log in with the same password)" and "leave the other options as they are", which the Flint version has. LuCI's "Allow overwriting conflicting package files" checkbox is left unexplained. | low |
| 5 | 1 | The download is on the Fedora page's Devices tab, not the landing Connections tab (14). Nothing says the `…aarch64_cortex-a53…ipk` / "same tailbond this router runs" file fits a Slate 7 (GL-BE3600). | low |
| 6 | 1 | No screen shows install success or failure after LuCI's Install. | low |
| 7 | 2 | The jargon "tailnet" in "Looking for your home server on your tailnet…" and "No home server found on your tailnet". | low |
| 8 | 3 | "Up 2 h 13 min" next to the ↓/↑ speed arrows reads like an upload figure. | low |
| 9 | 4 | Unclear whether "Most reliable — Sends everything over every connection" would override "Backup only" on the phone. | low |
| 10 | 5 | The screen says "full name", never "MagicDNS". Three identifiers per machine plus "(not this router's own Tailscale)" and "Direct / via relay" are noise for me. | low |
| 11 | 6 | The top on/off switch has no label (Fedora's says "Connected"). When off, "Ask to connect again" looks like the way to turn it back on. | low |
| 12 | 7 | A long "Who can approve…" paragraph sits above Approve. The Fedora page shows "Path 0 · Min 31 Ms", "7.5 GB to · 858 MB from devices", which mean nothing to me. | low |
| 13 | 8 | GL's Repeater page isn't shown and the button is named two ways ("Login Mode for Public Hotspots" (or "Go to Login")). There's no "I've signed in, check now" like 17b, so the Wi-Fi and cable sign-in flows differ. | low |
| 14 | 9 | "until 05:17 on this router's clock": the router's clock may not match mine. On 17 the small technical line comes before the plain explanation. | low |
| 15 | 10 | 18: the green "Setup done … can connect from anywhere" above "2 to improve" gives mixed signals. On 18c "All good", the Port row still says "that box must forward UDP 443". | low |
| 16 | 10 | Fresh Flint (13), Step 1: "tick the one route listed (10.77.0.0/24)" is jargon. It's clearly handed to "whoever manages your Tailscale account". | low |

Totals: 0 high / 3 medium / 13 low
