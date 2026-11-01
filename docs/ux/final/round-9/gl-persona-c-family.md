# Round 9 — GL persona C (family member)

**Who I am:** someone else set up the home server; I was handed a GL.iNet Slate 7 and asked to get it working. I know Wi-Fi and phone hotspots. I don't know what routes, ports or NAT are.

**What I looked at:** only `task-script.md`, `screens.md` and the PNGs in `out/ux/final/round-9/packet-gl/`. I looked at desktop screens for every task, plus the phone versions of 01b, 04, 05 and 14.

Overall I could get through every task. Most of the time the screen told me what to click in plain words. I got stuck on the "best speed at home" screens (port forwarding, carrier-grade NAT) and once on the Fedora page on a phone, where it offered to set up my phone as well.

---

## Task 1 — Install tailbond on the Slate 7 without SSH

**Where I'd click:**
- I followed the steps on the Fedora page (15, "Add a GL.iNet travel router (recommended)"): "Download the router package" → "On the travel router's admin page: SYSTEM → Advanced Settings → Go To LuCI, then System → Software → Upload Package…, pick the file, Install."
- 01a: SYSTEM → Advanced Settings. The page says "LuCI is OpenWrt's own web interface. Use the same admin password to log in." I click **Go To LuCI**. I don't know what LuCI or OpenWrt are, but the sentence tells me which password to use, so I'm fine.
- 01b: the "Manually install package" box with `tailbond_0.1.0-r42_aarch64_cortex-a53_neon-vfpv4.ipk (6.1 MB)`. I click **Install**.

**Where I'd hesitate:**
- 01b has a checkbox, "Allow overwriting conflicting package files". The Flint panel's version of the steps (11–13) says "(leave the other options as they are)" and "(log in with the same password)". The Fedora page (15), which is the one I'd use with a Fedora server, leaves both out. I'd wonder whether to tick it. (low)
- There's no screen after I click Install. LuCI's top bar only has "Status System Services Network Logout", with no way back to the GL page. Step 3 says "Back in the router's admin page, open VPN → tailbond", but I don't know whether the GL tab is still open or what address to type. 01a shows LuCI at `http://192.168.8.1:8080/...`, so I might guess 192.168.8.1. (low)

**I expected:** a "Done, go back to the GL admin page" message or link after installing.

## Task 2 — Pair with the home server

**Where I'd click:**
- 02: VPN → tailbond. It says "Looking for your home server on your tailnet…".
- 03: "Found your home server: homebox". I click **Ask to connect**. This is the nicest screen: one big button.
- 04: "Waiting for approval on homebox… On homebox, tap **Approve** next to **slate7**. This router connects by itself once approved; you can close this page." Very clear.

**Where I'd hesitate:**
- The word "tailnet" ("on your tailnet", "No home server found on your tailnet") is used as if I know it.
- 03c says "Checked 4 online machines. On the home server, tailbond must be installed and running, and Tailscale on." I can't check the home server. Unlike the Flint health card, it doesn't say "ask whoever set up the home server". (low)
- 03b ("Turn on Tailscale first" + "Open Tailscale settings") is clear.

## Task 3 — Hotel Wi-Fi (Repeater) + phone over USB, confirm both are used

**Where I'd look:** 05. The top says "Connected to homebox / Using Repeater (Wi-Fi) + Tethering (phone)". The Repeater and Tethering cards both show a green "In use". Done, no confusion. The phone screen (05 phone) shows the same.

06 (phone trouble) is also understandable: "Tethering (phone) not working right now" and "It's connected, but can't reach homebox. tailbond keeps trying; the other connections carry your traffic." That reassures me.

Small things I ignore: "eth0" under Ethernet, "UDP port 443", "This router's tunnel address 10.77.0.2". None of them asks me to do anything.

## Task 4 — Phone as backup only

**Where I'd click:** in the Tethering card, "Use it: Always | Backup only | Off". I click **Backup only**. The hint under Speed or reliability helps: "Saving your phone's data? Set Tethering or Cellular to "Backup only" under Internet connections." 07 then shows "Backup · saving data" and "Used only if your other connections stop working. Until then tailbond sends nothing over it". That's exactly what I wanted.

**Where I'd hesitate:** the task says "limit", and there's a blue "Change limit" link right on the card. My first click might be that. It opens a GB-per-month box (07b: "Counts everything tailbond sends and receives over it. At the limit, tailbond stops using this connection…"). That's a monthly data cap, not backup. The text explains it, so I'd go back and press Backup only. (low)

## Task 5 — Reach the NAS by its MagicDNS name

**Where I'd look:** the "Tailnet machines" card (05/07). The **nas** row shows `nas.tail1234.ts.net` with a **Copy name** button. The Home server card says "Tailnet names: Work on every device connected to this router while tailbond is on. Use the full name, like nas.tail1234.ts.net." Found it.

**Where I'd hesitate:** the word "MagicDNS" from the task appears nowhere on screen. It's called "Tailnet names" or "full name". I'd guess that's the same thing. The line "reach these through the bond to homebox (not this router's own Tailscale)" made me wonder whether my laptop needs Tailscale. I'd ignore it and try the name. (low)

## Task 6 — Turn tailbond off

**Where I'd click:** the blue switch at the top right of the "Connected to homebox" card (05). 09 then shows "tailbond is off". It also warns me: "Backup only and data limits here apply only while tailbond is on, so GL may use your phone: unplug it (or turn tethering off) if you don't want that." I'm glad it told me that.

**Where I'd hesitate:**
- The switch has no "On/Off" label. The only button with words in the Home server card is the red **Unpair**. I might try Unpair, but its box (09b) explains "This router stops using tailbond and forgets the home server…", so I'd cancel. In 08 there's a proper "Turn tailbond off" button. (low)
- While it's off, the Tethering card still shows "Backup only" selected. The top text explains it doesn't apply, but the screen sends two messages. Also, the "Tailnet machines" card only says "The list appears once tailbond is connected." It doesn't say whether I can still reach the NAS while it's off. (low)

## Task 7 — Approve the travel router at home

**Flint 3 (11):** "Waiting for your approval · 1 new / slate7 wants to connect", with **Approve** and Decline. I click Approve. In 12, slate7 shows under Devices as "Connected just now · 2 connections bonded". Easy.

**Fedora (14 desktop, 15):** the "Waiting for your approval" box at the top, "slate7 wants to connect", **Approve**. Easy.

**Where I'd go wrong — Fedora page on a phone (14 phone):** the Slate 7 screen (04) told me "open http://homebox:8077 on any device signed in to your Tailscale account (a phone works too)", so I'd use my phone. Under the Approve box there's a big green-bordered card: "**Add this phone** … Two steps, then tailbond takes over from Tailscale on this phone. … Download tailbond for Android", with a big "+ Add this phone" button. After approving I'd think this is the next step and install an app and change my phone's settings, which I didn't need. "tailbond takes over from Tailscale on this phone" sounds like it changes how my phone connects. Nothing on the page says "you're done once you've approved". (medium)

**Other small things:**
- On the Flint page the tagline under "tailbond" is "Bond this router's internet connections into one tunnel to your home server." But this router *is* the home server. That confused me for a second. (low)
- The Fedora Connections tab shows "Path 0 / Path 1", "Pixel-8 · Min 31 Ms". I don't know what a path is, and "Ms" looks like a typo. (low)

## Task 8 — Hotel Wi-Fi with a sign-in page (GL's Login Mode)

**Where I'd click:**
- 16: the blue box "Repeater needs you to sign in … Open it and click "Login Mode for Public Hotspots" (or "Go to Login"). Then open any website on your laptop or phone". I click **Open GL's Repeater page**, then the Login Mode button.
- 16b: "Now open any website on your laptop or phone: the hotel's sign-in page appears there." I do that and accept the terms. It says "Login Mode ends by itself once you've signed in … tailbond then adds Repeater back to bonding."

**Where I'd hesitate:**
- "(or "Go to Login")": the screen isn't sure what GL's button is called, and I can't see GL's Repeater page in this packet. Also "on your laptop or phone": my phone is the one plugged in by USB, so should I use it? I'd use the laptop. (low)
- 16b's main paragraph is long and hard to follow: "everything devices here do goes out the hotel Wi-Fi directly (not only the sign-in page, and not through your home server)… Only this router's own link to your home server stays up over your other connections (the top card; it carries almost nothing meanwhile)". I don't know which "top card" it means. The header shows "↓ 12 Kbps ↑ 8 Kbps", which looks like something broke. I also don't get a screen confirming I'm "bonded again". I'd have to trust it. (low)

## Task 9 — Hotel Ethernet cable with a sign-in page

**Where I'd click:** 17: "Ethernet needs you to sign in" → **Open sign-in page**. 17b: the hotel page opens in a new tab and I sign in. Back on the GL page I click "I've signed in, check now". Clear. I like that there's an explicit "I've signed in" button.

**Where I'd hesitate:**
- The first line of the 17 box is small print that reads like an engineer's note: "GL's Login Mode covers the repeater Wi-Fi only; tailbond handles sign-in pages on other connections like this one." It sits above the sentence I actually need.
- 17b says "until 05:08 on this router's clock". I don't know the router's clock, so "up to 9 more minutes" is the useful part. (low)

## Task 10 — At home: best speed

**Flint 3, all good (18c):** "Setup health — All good / Everything is set up for best speed". Nothing to do. Clear.

**Flint 3, fresh (13):** "Home server is on · 2 setup steps left".
- Step 1: "In the Tailscale admin console, open homebox, choose Edit route settings, tick the one route listed (10.77.0.0/24) and save. Done once, by whoever manages your Tailscale account." It's concrete enough that I could try. I don't know what a "route" is, but "tick the one listed" helps. (low)
- Step 2 (Dynamic DNS) says where to click and what to paste. OK.

**Flint 3, relayed (18):** "2 things to improve for best speed (tailbond works meanwhile, just slower)".
- The fix is "On this router: NETWORK → Firewall → Open Ports on Router, add UDP 41641. If another router sits in front of this one (for example your internet provider's box), also forward UDP port 41641 to this router there."
- I could find NETWORK → Firewall. But "UDP", "Open Ports" and "forward" mean nothing to me, and I don't know whether "another router sits in front". The grey text also tells me to "check: tailscale ping apple-tv ("via DERP" means relayed)", which is a command I don't know where to type.
- "Not sure how? Show this card to whoever set up your internet" is the line that saves me. I'd stop here and ask. (medium)

**Flint 3, carrier-grade NAT (18b):** red "Step 1: Your internet line uses carrier-grade NAT".
- The plain part I understand: "travel routers and phones away from home can't reach … here" and "Devices on your home Wi-Fi can still connect; from away they can't until this is fixed." So the Slate 7 won't work at a hotel at all.
- The fix is "enter this machine's public IPv6 address as the internet address (the router's status page shows it). Otherwise ask your provider for a public IPv4 address". I don't know which "status page" that is or which of the addresses there is the right one. I'd probably have to call my internet provider. "Not sure? Show this to whoever set up your internet" is there. I can't finish this myself, but the screen correctly tells me who can. (medium)

**Fedora (15, Devices tab):** "Setup health: 2 things to improve for best speed (tailbond works meanwhile)".
- The fix: "On the router in front of this machine, forward UDP port 41641 to it. (Turning on UPnP on that router also works, but lets any device on your network open ports.)" Plus "To check: tailscale ping apple-tv".
- It doesn't say where to click on that router, and it doesn't have the Flint's "Show this card to whoever set up your internet" line. It only says "Each fix is on your home router or in the Tailscale admin console; "How to fix" opens Tailscale's explanation." I'd be stuck and wouldn't even be told to ask someone. (medium)

---

## What worked well (for me)

- Pairing is two clicks: "Ask to connect", then "Approve" on the other screen. 04 tells me exactly what to do on the other device.
- Each connection card has one plain status word: "In use", "Backup · saving data", "Not working right now", "Needs sign-in".
- The "tailbond is off" screen warns me about phone data.
- Sign-in pages have explicit buttons, including "I've signed in, check now".

## Findings

| # | Task | Finding | Severity |
|---|---|---|---|
| 1 | 7 | Fedora page on a phone (14 phone): below "Approve", a big "**Add this phone** … then tailbond takes over from Tailscale on this phone" card with "Download tailbond for Android" / "+ Add this phone". After approving the router, I'd think this is the next step and install an app and change my phone, which I didn't need. Nothing says "done". | medium |
| 2 | 10 | Fedora setup health (15): "On the router in front of this machine, forward UDP port 41641 to it… UPnP…" gives no where-to-click and, unlike the Flint, no "Show this card to whoever set up your internet" line. I'm stuck with no next step. | medium |
| 3 | 10 | Flint relayed (18): "NETWORK → Firewall → Open Ports on Router, add UDP 41641. If another router sits in front … also forward UDP port 41641", plus "tailscale ping apple-tv ("via DERP" means relayed)". Port/forward jargon and a command I can't run. The "show this card to whoever set up your internet" line is the only way out. | medium |
| 4 | 10 | Flint CGNAT (18b): "carrier-grade NAT"; the fix is "enter this machine's public IPv6 address … (the router's status page shows it). Otherwise ask your provider for a public IPv4 address". I don't know which page or address that is. I can't finish alone, though the screen points me to someone who can. | medium |
| 5 | 1 | Fedora install steps (15) leave out the Flint's "(log in with the same password)" and "(leave the other options as they are)", so the LuCI checkbox "Allow overwriting conflicting package files" (01b) leaves me unsure. | low |
| 6 | 1 | No screen after LuCI "Install", and LuCI ("Status System Services Network Logout") has no way back to the GL page that step 3 sends me to. | low |
| 7 | 2 | "tailnet" used unexplained ("Looking for your home server on your tailnet…"). 03c "On the home server, tailbond must be installed and running" asks me to check something I can't, with no "ask whoever set it up". | low |
| 8 | 4 | The task says "limit", and "Change limit" sits on the Tethering card. It's a monthly GB cap, not backup, so it's an easy first wrong click (the text recovers me). | low |
| 9 | 5 | "MagicDNS" isn't shown anywhere (it says "Tailnet names" / "full name"). "(not this router's own Tailscale)" makes me wonder whether my laptop needs Tailscale. I still found `nas.tail1234.ts.net` + "Copy name". | low |
| 10 | 6 | The on/off switch has no label. The only worded button nearby is the red "Unpair" (its confirm box saves me). | low |
| 11 | 6 | When off, Tethering still shows "Backup only" selected while the top text says it doesn't apply. "The list appears once tailbond is connected" doesn't tell me whether the NAS still works. | low |
| 12 | 7 | The Flint (home server) page tagline says "Bond this router's internet connections into one tunnel to your home server", but this *is* the home server. | low |
| 13 | 7 | Fedora Connections tab: "Path 0 / Path 1", "Pixel-8 · Min 31 Ms". "Path" isn't explained and "Ms" looks like a typo. | low |
| 14 | 8 | 16: "click "Login Mode for Public Hotspots" (or "Go to Login")" hedges on GL's button name. "open any website on your laptop or phone": my phone is the USB-tethered one. | low |
| 15 | 8 | 16b's long paragraph ("Only this router's own link … (the top card; it carries almost nothing meanwhile)") is hard to follow. The header showing "12 Kbps" looks broken, and no screen confirms I'm bonded again. | low |
| 16 | 9 | 17: the first line is small engineer-style print ("GL's Login Mode covers the repeater Wi-Fi only; tailbond handles sign-in pages on other connections like this one."). 17b: "until 05:08 on this router's clock" means nothing to me. | low |
| 17 | 10 | Flint fresh (13), Step 1: "Edit route settings, tick the one route listed (10.77.0.0/24)": "route" jargon, but concrete enough to try. | low |

Totals: 0 high / 4 medium / 13 low
