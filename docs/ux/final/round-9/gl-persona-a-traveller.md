# Round 9 — GL persona A (traveller)

**Who I am:** I travel a lot and carry a GL.iNet router. On GL's own Tailscale page I flip one switch, log in, and it works. I know GL's admin panel (INTERNET, Repeater, "Login Mode for Public Hotspots", Applications), but I'm not a network engineer. I don't know NAT types, I don't forward ports, and I've never opened a terminal on a router.

**Materials:** `task-script.md`, `screens.md` and the PNGs in `out/ux/final/round-9/packet-gl/`. I looked at the desktop version of every screen listed below, plus the phone versions of 04, 05, 09 and 14.

---

## Task 1 — Install tailbond on the Slate 7 without SSH

**Where I click**
- On the home server I find the "Add a GL.iNet travel router (recommended)" card. The Fedora page has it on the Devices tab (15). The Flint 3 panel has it as "Add a GL.iNet travel router" (11–13). I click **"Download the router package"**.
- Step 2 is written out on both: *"SYSTEM → Advanced Settings → Go To LuCI, then System → Software → Upload Package…, pick the file, Install."* On the Slate 7 that matches screen 01a exactly. There's a **"Go To LuCI"** button, and the text under it says *"Use the same admin password to log in."*
- In LuCI (01b) I click **Upload Package…** and pick `tailbond_0.1.0-r42_aarch64_cortex-a53_neon-vfpv4.ipk (6.1 MB)`. The dialog asks *"Really install the uploaded package?"* and I click **Install**.
- Step 3: *"Back in the router's admin page, open VPN → tailbond"*. Screen 02 shows "tailbond" in the VPN menu, right under WireGuard Server.

**Where I hesitate**
- LuCI's dialog has a checkbox: *"Allow overwriting conflicting package files"*. The Flint's instructions tell me *"(leave the other options as they are)"*. The Fedora page (15) doesn't say that. Coming from the Fedora page, I'd stop and wonder whether to tick it. **Low.**
- The file name means nothing to me, and nothing confirms it's the right file for a Slate 7. The Flint says *"It's the same tailbond this router runs"*, which makes me think "but this is for my Slate, not my Flint". The Fedora page says only "the router package". A line like "works on GL.iNet 4.x routers like Slate 7, Flint 3" would settle it. **Low.**
- The install itself is fine for me. LuCI looks old and plain next to GL's UI, but there's a button for it, the password is the same, and the path is spelled out step by step.

## Task 2 — Pair the Slate 7 with the home server

**Where I click:** VPN → tailbond (02) shows *"Looking for your home server on your tailnet…"*. A few seconds later (03) it shows *"Found your home server: homebox"*, and I click **Ask to connect**. Screen 04 says *"Waiting for approval on homebox… On homebox, tap Approve next to slate7. This router connects by itself once approved; you can close this page."*

This feels like GL's Tailscale page: it found my server on its own and gave me one button. The "Tailscale off" (03b) and "nothing found" (03c) states both tell me exactly what to do (*"Open Tailscale settings"*, *"Search again"*, or type the name).

**Where I hesitate**
- While I'm waiting, the top card still says *"Not connected to a home server — Not set up yet. Connect this router to your home server below."* (04). I've just asked to connect, so "Not set up yet" reads like nothing happened. "Waiting for approval" in the top card would match what I did. **Low.**

## Task 3 — Hotel Wi-Fi + phone tethering, confirm both are used

**Where I look:** On screen 05 the top card says *"Connected to homebox — Using Repeater (Wi-Fi) + Tethering (phone)"* with *"↓ 59 Mbps ↑ 7.1 Mbps"*. Both the Repeater and Tethering cards have a green *"In use"* badge with their own speeds (38 + 21 = 59, which adds up). Ethernet shows *"Not connected"* with *"Plug a cable from the hotel or modem into the WAN port. It's used automatically once connected."*

Clear. I can confirm both are being used without guessing. When the phone has trouble (06), it's just as clear: *"Tethering (phone) not working right now"* and *"It's connected, but can't reach homebox. tailbond keeps trying; the other connections carry your traffic."* No findings.

## Task 4 — Phone as backup only

**Where I click:** On the Tethering card, under "Use it", I click **Backup only** (07). The badge changes to *"Backup · saving data"* with *"Used only if your other connections stop working. Until then tailbond sends nothing over it (GL itself may check the connection now and then, a few KB)."* The top card says *"Tethering (phone) on standby"*. The hint under "Speed or reliability" also points me there: *"Saving your phone's data? Set Tethering or Cellular to 'Backup only'"*.

I also tried **Change limit** (07b): *"5 GB a month — Save / Cancel"* and *"…At the limit, tailbond stops using this connection until the count restarts on the 1st of each month. 0 = no limit."*

**Where I hesitate**
- "Speed or reliability" sits right under the connections, and *"Most reliable — Sends everything over every connection. Uses the most data."* makes me wonder whether choosing it would override "Backup only" and burn my phone data. Nothing on screen says how the two settings relate. **Low.**
- The data counter always resets *"on the 1st of each month"*. My phone plan resets on a different day, so the limit won't line up with my real allowance. I'm only noting what the screen says; I can't see any other option. **Low.**

## Task 5 — Reach the NAS by its MagicDNS name

**Where I look:** "Tailnet machines" (05, near the bottom) lists **nas**, `nas.tail1234.ts.net`, `100.101.2.2`, *"linux · Online · direct"*, with a **Copy name** button. The Home server box also says *"Tailnet names — Work on every device connected to this router while tailbond is on. Use the full name, like nas.tail1234.ts.net."* So the name is `nas.tail1234.ts.net`, and I click Copy name.

**Where I hesitate**
- The page never says "MagicDNS". It says "full name". I worked it out because the format looks like what I've seen in Tailscale, but someone searching for "MagicDNS" wouldn't find it. **Low.**
- *"…reach these through the bond to homebox (not this router's own Tailscale)"*. My GL Tailscale is still on and logged in, and now there are two "Tailscales" on one router. I'm not sure whether I should change anything on GL's Tailscale page (for example, turn it off), and nothing tells me to leave it alone. **Low.**

## Task 6 — Turn tailbond off

**Where I click:** The blue switch in the top-right of the top card. Screen 09 says *"tailbond is off — Devices on this router use the internet directly, without your home server, over whichever connection GL's own settings pick (INTERNET page, top of the menu). Backup only and data limits here apply only while tailbond is on, so GL may use your phone: unplug it (or turn tethering off) if you don't want that. Turn on to bond your connections again."* **Unpair** is a separate button with its own confirmation (09b), so I wouldn't hit it by accident.

**Where I hesitate**
- **Medium.** The thing I care about most is that my phone's data could now be used. It's buried in the middle of a dense five-line paragraph. Meanwhile the Tethering card below still shows **"Backup only"** highlighted in blue with a grey *"Ready"* badge. My eye goes to that card and reads "still backup only, fine". Only the small grey text says *"This choice applies when tailbond is on."* I'd expect the warning to be its own short line or banner ("Your phone may now be used for data — unplug it to avoid charges"), or the setting to be visibly greyed out while tailbond is off.
- *"Ask to connect again"* stays under Home server, both while connected (05) and while off (09). To get bonding back I might click that instead of the switch, and I don't know whether that would trigger a new approval. **Low.**

## Task 7 — Approve the new travel router at home

**Flint 3 (11):** A card at the top says *"Waiting for your approval · 1 new"* and *"slate7 wants to connect — Travel router on your tailnet · signed in as alex@example.com · GL-BE3600 · 100.101.9.9"*. I click **Approve**. Screen 12 then lists *"slate7 — Connected just now · 2 connections bonded"*. Easy.

**Fedora (14):** Same at the top: *"slate7 wants to connect"* with **Decline / Approve**. Easy too.

**Where I hesitate**
- On the Fedora page in a phone browser (14-phone), right under the approval card there's a big green card: *"Add this phone — …tailbond takes over from Tailscale on this phone"* with a large **"+ Add this phone"** button. I opened the page on my phone only to approve the router. That big button made me wonder whether I also had to add my phone, which I don't want. **Low** (Approve comes first, so I'd most likely just tap it).
- The Fedora header shows a switch labelled *"Connected"* next to "Home server · reachable at home.example.net:443". On a server that's odd. Connected to what? Would switching it off disconnect everyone? **Low.**
- The Fedora "Connections" tab (14) shows *"Path 0 / Path 1"*, *"Pixel-8 · Min 31 Ms"*. "Path", "Min … Ms" and the capital "Ms" read like an engineer's screen. On the Flint I see *"2 connections bonded"*, which I understand. **Low.**

## Task 8 — Hotel Wi-Fi with a sign-in page (GL Repeater)

**Where I click:** On screen 16 the top card says *"Repeater (Wi-Fi) needs sign-in"*. A blue banner says *"Repeater needs you to sign in — GL's Repeater page handles this sign-in. Open it and click 'Login Mode for Public Hotspots' (or 'Go to Login'). Then open any website…"* and has an **Open GL's Repeater page** button. That's exactly the GL feature I already know. After starting Login Mode (16b), the tailbond page says *"Signing in to Repeater through GL's Login Mode"*, and it says tailbond adds Repeater back *"once you've signed in (or click 'Quit Login Mode' on GL's Repeater page)"*. I sign in on the hotel page and wait.

**Where I hesitate**
- The 16b banner is one long paragraph: *"…Until you've signed in, everything devices here do goes out the hotel Wi-Fi directly (not only the sign-in page, and not through your home server), so your phone's data isn't used for them. Only this router's own link to your home server stays up over your other connections (the top card; it carries almost nothing meanwhile)."* "The top card" — which card? The top of the page is the status card. "Everything devices here do goes out" also reads awkwardly. I got the gist (sign in and it fixes itself), but I read it twice. **Low.**
- At the same moment the Tethering card shows a green *"In use"* badge. That made me worry my phone data was being used, until I found the sentence above that says it isn't. **Low.**
- The top card says *"Router connected to homebox · devices signing in"*. I'm the one signing in, not "devices". **Low.**
- The Ethernet flow (17b) has an **"I've signed in, check now"** button. The Repeater flow has nothing like it: I either wait or go back to GL to "Quit Login Mode". I'd expect the same button on both. **Low.**

## Task 9 — Hotel Ethernet with a sign-in page

**Where I click:** On screen 17 a banner says *"Ethernet needs you to sign in"* and the Ethernet card shows *"Needs sign-in"*. The small line on top tells me *"GL's Login Mode covers the repeater Wi-Fi only; tailbond handles sign-in pages on other connections like this one."* (Good: my habit would have been to go to GL's Login Mode.) I click **Open sign-in page** and the hotel page opens in a new tab. On 17b I have **Open the sign-in page again**, **I've signed in, check now** and **Stop sign-in mode**, and the Ethernet card shows *"Signing in…"*. Clear, and the phone stays *"Backup · saving data"* throughout.

**Where I hesitate**
- *"For up to 9 more minutes (until 05:08 on this router's clock)"*. When I'm travelling the router's clock may not be in my timezone, so "05:08" might mean nothing to me. The "9 more minutes" part is enough. **Low.**
- On screen 17 the banner text comes in three different sizes, and the small GL-Login-Mode note is first, so my eye goes to the side note before the main message. **Low.**

## Task 10 — At home, make sure everything is set up for best speed

**Flint 3:** "Setup health" shows *"All good — Everything is set up for best speed"* with checkmarks (11, 12, 18c). On a fresh Flint (13) the top card says *"Home server is on · 2 setup steps left"*, with Step 1 (Tailscale admin console → homebox → *"Edit route settings, tick the one route listed (10.77.0.0/24) and save"* plus an **Open admin console** button) and Step 2 (*"APPLICATIONS → Dynamic DNS"*, paste the name). I can follow both, though Step 1 sends me into Tailscale's admin console, which is a step up from GL's one switch.

**Flint relayed (18):** *"2 things to improve for best speed (tailbond works meanwhile, just slower)"*. Fix: *"On this router: NETWORK → Firewall → Open Ports on Router, add UDP 41641. If another router sits in front of this one…, also forward UDP port 41641…"*. I can follow the GL menu path. Mixed in, though, are *"To check: tailscale ping apple-tv ('via DERP' means relayed)"* and *"('hard NAT')… no port mapping on the router"*. Those I skip. *"Not sure how? Show this card to whoever set up your internet"* helps. **Low.**

**Flint CGNAT (18b):** *"Your internet line uses carrier-grade NAT … travel routers and phones away from home can't reach UDP 443 here."* I understand the outcome: from away it won't work until this is fixed. The fixes are *"If your line has IPv6, enter this machine's public IPv6 address as the internet address (the router's status page shows it). Otherwise ask your provider for a public IPv4 address"*. The IPv6 option loses me: which "status page", and which of the addresses there? Calling my provider is something I can do. **Medium** (I finish only by phoning the provider or asking someone; the IPv6 route isn't usable for me as written).

**Fedora (15, Devices tab):** *"Setup health: 2 things to improve for best speed"*, with *"On the router in front of this machine, forward UDP port 41641 to it. (Turning on UPnP on that router also works, but lets any device on your network open ports.)"*, plus *"tailscale ping apple-tv ('via DERP' means relayed)"* and *"hard NAT"*.

**Where I hesitate**
- **Medium.** Unlike the Flint, the Fedora page gives no menu path. It just says "forward UDP port 41641". I don't know how to forward a port to "this machine", the UPnP alternative comes with a security warning, and the check is a terminal command. There's a "How to fix" link and *"Each fix is on your home router or in the Tailscale admin console"*, but I'd be stuck or reading Tailscale's docs.
- On the Fedora page, Setup health is collapsed on the Connections tab (*"Setup health: 2 things to improve… Show"*), and the full card sits under the **Devices** tab. I wouldn't look for setup health under "Devices". **Low.**
- Flint "All good" list: *"✓ Network is friendly to direct connections — no port mapping on the router"*. The grey subtext sounds like something is missing, right under a green tick. **Low.**
- Flint 18: the top card says *"Setup done"* in green, but Setup health just below says *"2 to improve"* in orange. Is it done or not? I can piece together that it works but is slower. **Low.**

---

## Findings

| # | Task | Finding | Severity |
|---|---|---|---|
| 1 | 1 | Fedora install steps don't say to leave LuCI's *"Allow overwriting conflicting package files"* unticked (the Flint says *"leave the other options as they are"*) | low |
| 2 | 1 | Nothing confirms the downloaded `.ipk` fits a Slate 7; the Flint's *"It's the same tailbond this router runs"* suggests a Flint package | low |
| 3 | 2 | While waiting for approval, the top card still says *"Not set up yet"* | low |
| 4 | 4 | How "Speed or reliability" (*"Most reliable … Uses the most data"*) relates to "Backup only" isn't stated; I worry it overrides the data saving | low |
| 5 | 4 | Data limit resets *"on the 1st of each month"*, which may not match my phone plan's billing date | low |
| 6 | 5 | "MagicDNS" never appears; the name is called the "full name" | low |
| 7 | 5 | *"(not this router's own Tailscale)"*: no word on whether GL's own Tailscale should stay on, be changed or be left alone | low |
| 8 | 6 | When off, the phone-data warning is buried mid-paragraph in the top card, while the Tethering card still shows **Backup only** highlighted, so it looks like it still applies | medium |
| 9 | 6 | *"Ask to connect again"* shows while connected and while off; I might click it instead of the switch to resume | low |
| 10 | 7 | On a phone, the Fedora page shows a big *"+ Add this phone"* card right under the approval, which suggests an extra step I don't want | low |
| 11 | 7 | The Fedora header switch is labelled *"Connected"* on the server itself; unclear what turning it off would do | low |
| 12 | 7/10 | Fedora Connections tab: *"Path 0 / Path 1"*, *"Min 31 Ms"* is engineer wording | low |
| 13 | 8 | Login-Mode banner: *"(the top card; it carries almost nothing meanwhile)"* and *"everything devices here do goes out"* are hard to parse | low |
| 14 | 8 | During Login Mode, Tethering shows green *"In use"*, which made me think my phone data was being used | low |
| 15 | 8/9 | Top card *"Router connected to homebox · devices signing in"*: I'm the one signing in, not "devices" | low |
| 16 | 8 | The Repeater sign-in has no *"I've signed in, check now"* button, unlike the Ethernet flow | low |
| 17 | 9 | *"until 05:08 on this router's clock"* may be in the wrong timezone when travelling | low |
| 18 | 9 | Ethernet sign-in banner uses three text sizes and the side note about GL Login Mode comes first | low |
| 19 | 10 | Flint relayed fix mixes a usable GL path with *"tailscale ping … 'via DERP'"* and *"hard NAT"* jargon | low |
| 20 | 10 | CGNAT fix *"enter this machine's public IPv6 address … (the router's status page shows it)"* isn't actionable for me; calling the provider is my only real option | medium |
| 21 | 10 | Fedora health fix *"forward UDP port 41641 to it"* has no menu path; the alternative is UPnP with a security caveat, and the check is a terminal command | medium |
| 22 | 10 | Fedora Setup health lives under the **Devices** tab (collapsed "Show" banner on Connections) | low |
| 23 | 10 | Green tick *"Network is friendly to direct connections"* with subtext *"no port mapping on the router"* reads like a problem | low |
| 24 | 10 | Flint 18: *"Setup done"* (green) next to *"2 to improve"* (orange) | low |

Totals: 0 high / 3 medium / 21 low
