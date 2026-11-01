# Round 8 — GL persona A (traveller)

**Who I am:** I travel a lot and carry a GL.iNet router. I'm used to GL's Tailscale page, where you flip one toggle and it works. I'm not a network engineer: I know what Wi-Fi, a cable and "my phone's hotspot" are, and I've heard of "port forwarding", but I wouldn't know how to set it up myself.

**Materials:** task-script.md, screens.md and the PNGs in `out/ux/final/round-8/packet-gl/`. I looked at the desktop version of every screen and at the phone versions of 04, 05, 14 and 16b.

---

## Task 1: Install tailbond on the Slate 7 without SSH

**Where I click**
- On the home server's page I find the "Add a GL.iNet travel router" card. It's on the Fedora page (15, labelled "(recommended)") and on the Flint 3 panel (11–13). I click **Download the router package**.
- Step 2 on that card says: "On the travel router's admin page: SYSTEM → Advanced Settings → Go To LuCI". Screen 01a matches it exactly: SYSTEM → Advanced Settings, a "Go To LuCI" button, and "Use the same admin password to log in."
- In LuCI (01b) I go to System → Software → Upload Package…, pick the file, and get the "Manually install package" dialog ("Really install the uploaded package?"). I click **Install**.

**Where I'd hesitate**
- The dialog has an unticked checkbox, "Allow overwriting conflicting package files". The Flint card answers this: "(leave the other options as they are)". It also reminds me "(log in with the same password)". The Fedora card (15) has neither hint. It only says "pick the file, Install." **Low.**
- No screen shows what happens after I click Install. The next screen I have is 02, the tailbond page already in the GL menu. I'd like a line like "When LuCI says it's installed, go back to the GL admin page", or a mention of how long it takes. **Low.**
- The file is called `tailbond_0.1.0-r42_aarch64_cortex-a53_neon-vfpv4.ipk`. The Flint says "It's the same tailbond this router runs." But the Flint is a GL-BE9300 and my router is a GL-BE3600, so I'd wonder whether one file fits both. Neither page names the router models the package is for. **Low.**

**Verdict:** I can do it, and the steps match GL's own menus. That's much better than SSH.

---

## Task 2: Pair the Slate 7 with the home server

**Where I click**
- VPN → tailbond (02). It shows "Looking for your home server on your tailnet…". A few seconds later (03) it shows "Found your home server: homebox", and I click **Ask to connect**. That's nearly GL-simple.
- 04 shows "Waiting for approval on homebox…" with clear text: "This router connects by itself once approved; you can close this page."
- The error cases are friendly. 03b (Tailscale off) says "Turn on Tailscale first" and has an **Open Tailscale settings** button. 03c says "No home server found on your tailnet … Checked 4 online machines" and offers to type the name.

**Where I'd hesitate**
- Every pairing screen says "Your home server … must be signed in to the same Tailscale account as this router." The Fedora page (14) then says homebox is "a tagged machine, so it has no single owner". So is it on my account or not? It works anyway, but the two messages don't agree. **Low.**
- While I wait, the header card still says "Not connected to a home server / Not set up yet. Connect this router to your home server below." That makes it look as if nothing happened. "Waiting for approval" at the top would reassure me. **Low.**

---

## Task 3: Hotel Wi-Fi (Repeater) and phone tethering, confirm both are used

**Where I look:** 05. The header says "Connected to homebox / Using Repeater (Wi-Fi) + Tethering (phone)". Both cards show a green "In use" badge with their own speeds (Repeater "↓ 38 Mbps", Tethering "↓ 21 Mbps"). The header total, "↓ 59 Mbps", is their sum. That's clear confirmation. I'd do nothing else.

When the phone has trouble (06), the header says "Tethering (phone) not working right now", and the card says "It's connected, but can't reach homebox. tailbond keeps trying; the other connections carry your traffic." That's excellent and reassuring.

**Small thing**
- On desktop, "Up 2 h 13 min" sits right after "↑ 7.1 Mbps". The arrow means upload, so for a second I read "Up" as more upload. "Connected for 2 h 13 min" would be clearer. **Low.**

---

## Task 4: Phone as backup only

**Where I click:** on the Tethering card's "Use it" row, I click **Backup only** (07). The badge changes to "Backup · saving data". The card says "Used only if your other connections stop working. Until then it isn't used at all, so it uses no data." The "Speed or reliability" section also hints: "Saving your phone's data? Set Tethering or Cellular to 'Backup only'". This is exactly what I wanted.

I also tried **Change limit** (07b). It shows "5 GB a month" with Save/Cancel and explains: "At the limit, tailbond stops using this connection until the count restarts on the 1st of each month. 0 = no limit."

**Small things**
- The limit "Counts everything tailbond sends and receives over it". That isn't what my carrier counts, and the count restarts on the 1st, not on my billing date. When tailbond is off, 09 says "Backup only and data limits here apply only while tailbond is on." I'd only understand all three facts by reading two different screens. **Low.**

---

## Task 5: Reach the NAS from the laptop by its MagicDNS name

**Where I look:** the "Tailnet machines" card (05): "nas / nas.tail1234.ts.net / 100.101.2.2 / linux · Online · direct", with a **Copy name** button. The Home server card also says "Tailnet names: Work on every device connected to this router while tailbond is on. Use the full name, like nas.tail1234.ts.net." So the name is **nas.tail1234.ts.net**. Easy to find.

**Small thing**
- On my own Tailscale devices I'm used to typing just "nas". Here I have to type the full name, and the page doesn't say why "nas" alone won't work. The card also says "(not this router's own Tailscale)". As a GL user with GL's Tailscale switched on, that leaves me unsure whether I should keep GL's Tailscale on, or whether it gets in the way. **Low.**

---

## Task 6: Turn tailbond off

**Where I click:** the big toggle in the header card. Screen 09 confirms: "tailbond is off — Devices on this router use the internet directly, without your home server, over whichever connection GL's own settings pick (INTERNET page). Backup only and data limits here apply only while tailbond is on." This is a clear warning about my phone data. If I click Unpair by mistake, the dialog (09b) explains that it "forgets the home server" and needs approval again. Good.

**Small things**
- When tailbond is off, the Home server card still shows **Ask to connect again** next to Unpair. I might click it to turn tailbond back on instead of using the toggle. **Low.**
- Related (08, home server unreachable): the toggle is **on**, but the Repeater and Tethering cards say "This choice applies when tailbond is on. While it's off, GL's own settings…". That contradicts the toggle. The header also repeats itself: "Can't reach homebox / Can't reach homebox. See below." The red box itself is very good: "Meanwhile, devices on this router have no internet … turn tailbond off", with a **Turn tailbond off** button. **Low.**

---

## Task 7: Approve the new travel router at home

**Flint 3 (11 → 12):** "Waiting for your approval · 1 new", then "slate7 wants to connect · Travel router on your tailnet · signed in as alex@example.com · GL-BE3600", with **Approve**. After approving, slate7 appears under Devices as "Connected just now · 2 connections bonded". Very clear.

**Fedora (14, 15):** the "Waiting for your approval" card is at the top, with **Approve**. Easy to click. But the page contradicts itself about who is allowed to approve:
- Top card (14 and 15): "Who can approve: **anyone on your Tailscale network who can reach homebox** (it's a tagged machine, so it has no single owner). Set a password below to limit that."
- Devices tab, a little lower on the same page (15): "Who can approve: **alex@example.com (the Tailscale account homebox belongs to)** … People you share homebox with, and others on your Tailscale network, can ask to connect but **can't approve**."

I can't tell whether my family members, who share my tailnet, can approve devices right now, or whether I need to set the password. Since this is about who gets into my home network, it bothers me. **Medium.**

- Fedora page on a phone (14-phone): right under Approve is a big "Add this phone" card with "Download tailbond for Android". If I approve from my phone, I might think I also have to install the app. The header is cut off too: "reachable at home.···". **Low.**

---

## Task 8: Hotel Wi-Fi sign-in page with tailbond on

**Where I click:** 16 shows a blue box: "Repeater needs you to sign in — GL's Repeater page handles this sign-in. Open it and click 'Login Mode for Public Hotspots' (or 'Go to Login'). Then open any website on your laptop or phone". I click **Open GL's Repeater page** and use GL's Login Mode, which I already know. Meanwhile the header shows "Using Tethering (phone) · Repeater (Wi-Fi) needs sign-in", so I'm still online through my phone. Back on tailbond (16b) it says "Router connected to homebox · devices signing in" and gives instructions.

**Small things**
- The 16b text is long, especially on the phone, where it's a whole screen. One part confused me: "Only this router's own link to your home server stays up over your other connections (the top card; it carries almost nothing meanwhile)." I don't know what "the top card" means here, and "link" vs "tunnel" (used in 17b) sound like two different things. The first sentence, "Now open any website…", is the part I need, and it's at the start, so I can still finish. **Low.**
- I have no screen showing the result after I sign in. The text says "Login Mode ends by itself once you've signed in … tailbond then adds Repeater back to bonding." I'd trust that, but I can't confirm on screen that it went back to "Repeater + Tethering". **Low.**

---

## Task 9: Hotel Ethernet with a sign-in page

**Where I click:** 17 says "Ethernet needs you to sign in" and explains "GL's Login Mode covers the repeater Wi-Fi only; tailbond handles sign-in pages on other connections like this one." I click **Open sign-in page**, and the hotel page opens in a new tab. 17b: "Signing in to Ethernet… Sign in on the page that opened…". It has **I've signed in, check now**, **Open the sign-in page again** and **Stop sign-in mode**. Clear. It also tells me why the Wi-Fi and the cable are handled differently, which I would otherwise have wondered about.

**Small thing**
- "For up to 9 more minutes (until 04:57 on this router's clock)". I don't know what time my router's clock shows, and it may still be on a different time zone after a flight. The countdown alone is enough. **Low.**

---

## Task 10: At home, set up for best speed

**Flint 3:**
- 18c "Setup health · All good · Everything is set up for best speed". Clear: that's what done looks like.
- 18 "2 things to improve for best speed (tailbond works meanwhile, just slower)". The fix is a GL menu path I can follow: "On this router: NETWORK → Firewall → Open Ports on Router, add UDP 41641". The "To check: tailscale ping apple-tv ('via DERP' means relayed)" part is a command, and I have no SSH or command line, so I'd skip it. **Low.**
- 13 (fresh setup): "Step 1 … In the Tailscale admin console, open homebox, choose Edit route settings, tick the one route listed (10.77.0.0/24)". "Route" is jargon, but the **Open admin console** button and the exact words to click make it doable. Step 2 ("APPLICATIONS → Dynamic DNS … paste the name") is GL-friendly. **Low.**
- 18b CGNAT: "Step 1: Your internet line uses carrier-grade NAT … travel routers can't reach UDP 443 here. Ask your provider for a public IPv4 address (or use public IPv6)." I was told "the home server is on your Tailscale network", and Tailscale works fine from my phone even on this line. So I'd assume tailbond goes over Tailscale and would be surprised that it doesn't. The fix is a phone call to my ISP, which I wouldn't know how to phrase. The page also says "Devices on your home Wi-Fi can still connect", which doesn't help me as a traveller. I'd finish with **How to fix** and a call, but only after real confusion. **Medium.**

**Fedora (15):**
- "Setup health: 2 things to improve". Both items give the same instruction twice: "On the router in front of this machine, forward UDP port 41641 to it, and allow 41641/udp in this machine's firewall. (Turning on UPnP on that router also works, but lets any device on your network open ports.)" I don't know how to "allow 41641/udp in this machine's firewall" on Fedora; there's no button or menu path. The checking step is again a command ("tailscale ping apple-tv"). The help pointer "the guide fedora-server.md has a section per item" is a file name, not a link, and I don't know where to find it. "How to fix" links exist, but compared with the Flint's menu path, I'd be stuck here without help. **Medium.**

---

## Findings

| # | Task | Finding | Severity |
|---|---|---|---|
| 1 | 7 | Fedora page contradicts itself on who can approve: top card says "anyone on your Tailscale network who can reach homebox (it's a tagged machine…)", Devices tab says only "alex@example.com … others … can't approve" | medium |
| 2 | 10 | Flint CGNAT: "travel routers can't reach UDP 443 here. Ask your provider for a public IPv4 address": surprising when I was told it runs over my Tailscale network; fix is outside the panel (ISP call); "Devices on your home Wi-Fi can still connect" doesn't help me | medium |
| 3 | 10 | Fedora health fixes assume admin skills: "allow 41641/udp in this machine's firewall" (no how), same text repeated for both items, "tailscale ping apple-tv" command, "the guide fedora-server.md" is a filename, not a link | medium |
| 4 | 1 | Fedora install steps lack the Flint's "(log in with the same password)" and "(leave the other options as they are)" for LuCI's "Allow overwriting conflicting package files" checkbox; no screen shows the install succeeded | low |
| 5 | 1 | Package (`…aarch64_cortex-a53_neon-vfpv4.ipk`) doesn't say which routers it fits; "It's the same tailbond this router runs" on a BE9300 for a BE3600 | low |
| 6 | 2 | "must be signed in to the same Tailscale account as this router" vs Fedora's "tagged machine, so it has no single owner" | low |
| 7 | 2 | While waiting for approval the header still says "Not connected to a home server / Not set up yet" | low |
| 8 | 3 | "Up 2 h 13 min" next to "↑ 7.1 Mbps" reads like an upload figure | low |
| 9 | 4 | Data limit "Counts everything tailbond sends and receives" (not the carrier's count) and limits don't apply when tailbond is off; explained only across two screens | low |
| 10 | 5 | Must use full name ("Use the full name, like nas.tail1234.ts.net"); "(not this router's own Tailscale)" leaves unclear whether to keep GL's Tailscale on | low |
| 11 | 6 | "Ask to connect again" shown while tailbond is off, could be mistaken for the way to turn it back on; on 08 (toggle on) cards say "While it's off…" and the header repeats "Can't reach homebox" | low |
| 12 | 7 | Fedora page on phone: big "Add this phone" card right under Approve; header truncated "reachable at home.···" | low |
| 13 | 8 | 16b text very long (a full phone screen); "Only this router's own link … (the top card; …)" unclear; no screen showing bonding resumed after sign-in | low |
| 14 | 9 | "until 04:57 on this router's clock": router clock/time zone unknown to me | low |
| 15 | 10 | Flint relay check "tailscale ping apple-tv ('via DERP' means relayed)" is a command I can't run | low |
| 16 | 10 | Flint fresh setup: "Edit route settings, tick the one route listed (10.77.0.0/24)" is jargon (button helps) | low |

Totals: 0 high / 3 medium / 13 low
