# Round 12 — GL persona A (traveller)

**Who I am:** I travel a lot and carry a GL.iNet Slate 7. I'm used to GL's own Tailscale page: one switch, log in, done. I'm not a network engineer. I know GL's menus (INTERNET, VPN, APPLICATIONS, SYSTEM), Repeater mode, USB tethering and GL's "Login Mode for Public Hotspots". I have never used LuCI.

**What I looked at:** only `task-script.md`, `screens.md` and the PNGs in `packet-gl/`. I checked the desktop version of every screen in the manifest, plus the phone versions of 05, 11 and 17b.

**Overall:** this felt like a GL page. Every task could be finished from what's on screen, and most screens tell me what to click next in GL's own words ("SYSTEM → Advanced Settings → Go To LuCI", "APPLICATIONS → Dynamic DNS", "NETWORK → Firewall → Open Ports on Router"). Nothing blocked me and nothing confused me badly. What's left is polish: some wording is dense, a few states are ambiguous, and I never get to see a "done" screen after two of the flows.

---

## Task 1: Install tailbond on the Slate 7 without SSH

**Where I click**
- I get the file from the home server first. On the Fedora page (15, Devices tab), the "Add a GL.iNet travel router (recommended)" card has a **Download the router package** button and three steps. On the Flint 3 (11–13) the same card is in bold steps.
- On the Slate 7 (01a): SYSTEM → Advanced Settings shows "LuCI is OpenWrt's own web interface. Use the same admin password to log in." I click **Go To LuCI**.
- In LuCI (01b): System → Software → **Upload Package…**, pick the file. The dialog says "Really install the uploaded package?" with `tailbond_0.1.0-r42_aarch64_cortex-a53_neon-vfpv4.ipk (6.1 MB)`. I click **Install**.
- Back in GL admin, VPN → tailbond now exists (02).

**What went well:** the steps on the home server match the LuCI screens word for word. The Flint version even warns me "(leave the other options as they are)" and "The router's internet keeps working", which is exactly what I'd worry about.

**Where I hesitate**
- The LuCI dialog has a checkbox, "Allow overwriting conflicting package files". The Fedora page's step 2 only says "pick the file, Install." It doesn't say to leave that box alone. (The Flint page does.) **Low.**
- Nothing tells me this one file is right for a Slate 7. The Flint page says "It's the same tailbond this router runs", which made me briefly wonder if it's a Flint-only build. The file name (`aarch64_cortex-a53_neon-vfpv4`) means nothing to me. The card title "Add a GL.iNet travel router" mostly reassured me. **Low.**
- After Install there's no screen showing success, and LuCI's menu has no link back to GL's panel (just "Status System Services Network Logout"). Step 3 says "Back in the router's admin page". I'd have to know to retype 192.168.8.1 myself. **Low.**

## Task 2: Pair the Slate 7 with the home server

**Where I click:** VPN → tailbond (02) shows "Looking for your home server on your tailnet…". A few seconds later (03) it says "Found your home server: homebox" with a blue **Ask to connect** button. That's it, and it felt as easy as GL's Tailscale page.

**What went well**
- If my Tailscale is off (03b), "Turn on Tailscale first" comes with an **Open Tailscale settings** button.
- If no server is found (03c), "Checked 4 online machines. On the home server, tailbond must be installed and running, and Tailscale on." That's a clear checklist.
- While waiting (04), the page says "This router connects by itself once approved; you can close this page", and it tells me I can approve from my phone at `http://homebox:8077`.

**Where I hesitate**
- While I'm waiting (04), the big status card still says "Not connected to a home server / Not set up yet. Connect this router to your home server below." Meanwhile the card underneath says "Waiting for approval on homebox…". The two messages slightly contradict each other. **Low.**

## Task 3: Hotel Wi-Fi (Repeater) + phone USB tethering, confirm both are used

**Where I look:** at the top of 05, "Connected to homebox / Using Repeater (Wi-Fi) + Tethering (phone)", with ↓59 Mbps ↑7.1 Mbps. Under "Internet connections", both the **Repeater** and **Tethering** cards show a green **In use** badge with their own speeds (38 + 21 = the 59 at the top). Confirmed in seconds; this is better than I expected.

When the phone has trouble (06), the Tethering card turns red: "Not working right now … It's connected, but can't reach homebox. tailbond keeps trying; the other connections carry your traffic." Very clear.

No findings.

## Task 4: Phone as backup only

**Where I click:** on the Tethering card, the "Use it" row, I click **Backup only** (07). The badge changes to "Backup · saving data", and the card explains "Used only if your other connections stop working. Until then tailbond sends nothing over it (GL itself may check the connection now and then, a few KB)." The status line at the top says "Tethering (phone) on standby". The hint under "Speed or reliability" ("Saving your phone's data? Set Tethering or Cellular to 'Backup only'") would have pointed me here anyway.

I also tried **Change limit** (07b): "5 GB a month", Save.

**Where I hesitate**
- The limit text says the count "restarts on the 1st of each month". My phone plan doesn't reset on the 1st, and I don't see a way to change the day. **Low.**

## Task 5: Reach the NAS by its MagicDNS name from the laptop

**Where I look:** in the "Tailnet machines" card (05), the row "nas" shows `nas.tail1234.ts.net` and `100.101.2.2`, "Online · direct", with a **Copy name** button. The Home server card also says "Tailnet names: Work on every device connected to this router while tailbond is on. Use the full name, like nas.tail1234.ts.net." I'd copy it and use the full name. Found easily.

**Where I hesitate**
- The card says laptops reach these "through the bond to homebox (not this router's own Tailscale)". I turned on GL's Tailscale for this, so now I'm unsure: does GL's Tailscale switch still matter, and should I leave it on? And when tailbond is off (09), the list says only "The list appears once tailbond is connected." So I can't tell whether my laptop can still reach the NAS some other way. **Low.**

## Task 6: Turn tailbond off

**Where I click:** the blue switch in the top-right of the status card (05). In 09 the page says "tailbond is off / Devices on this router use the internet directly … Turn on to bond your connections again." It's the same pattern as GL's own toggle.

**What went well:** the yellow warning "While tailbond is off, GL may use Tethering (your phone's data) … 'Backup only' and data limits only apply while tailbond is on", with a **GL's INTERNET page** button, is exactly what a traveller needs to know. Unpair is separate, red, and asks for confirmation (09b: "This router stops using tailbond and forgets the home server … it will need to approve"). I won't hit it by accident.

**Where I hesitate**
- In the off state (09), the Home server card still shows **Ask to connect again**. Is that how I turn it back on, or is the switch? Ask again for what? I'd use the switch, but the button makes me second-guess. (It's also there while connected, in 05, where it means even less to me.) **Low.**
- There's a second "off" look (10): "tailbond isn't running on this router. It may have been turned off or stopped after an error." with **Start tailbond**. If I flip the switch off, will I see 09 or 10 next time? Two different "off" screens are confusing. **Low.**
- Related (screen 08, not a task): if home can't be reached, "devices on this router have no internet" until I turn tailbond off. With GL's Tailscale, the hotel internet never stopped. This surprised me, but the screen explains it and puts a **Turn tailbond off** button right there. **Low.**

## Task 7: Approve the new travel router on the home side

**Flint 3 (11):** VPN → tailbond has a "Waiting for your approval · 1 new" card at the top: "slate7 wants to connect / Travel router on your tailnet · signed in as alex@example.com · GL-BE3600". I click **Approve**. In 12, slate7 is in Devices: "Connected just now · 2 connections bonded".

**Fedora (14):** the same orange "Waiting for your approval" box at the top, with **Approve**.

It's easy, it's at the top, and the name and model match my router.

**Where I hesitate**
- The text around the button is heavy. For example: "Approving lets the device use homebox's internet connection … and reach your Tailscale machines as homebox: they allow it whatever they allow homebox, which may be more than its owner's own account gets." And "until then whoever holds it could also approve devices here." The key sentence for me, "If this router stays with you, just approve", is buried in the middle of an orange paragraph. **Low.**

## Task 8: Hotel Wi-Fi with a sign-in page (GL repeater detected it)

**Where I click:** in 16, the top says "Using Tethering (phone) · Repeater (Wi-Fi) needs sign-in". The blue box "Repeater needs you to sign in" tells me "GL's Repeater page handles this sign-in. Open it and click 'Login Mode for Public Hotspots' (or 'Go to Login')." I click **Open GL's Repeater page** and start Login Mode, which I already know from GL. Back on tailbond (16b): "Signing in to Repeater through GL's Login Mode … Now open any website on your laptop or phone: the hotel's sign-in page appears there … Login Mode ends by itself once you've signed in … tailbond then adds Repeater back to bonding."

**What went well:** it reuses the GL feature I already trust, and it tells me my phone data is protected meanwhile (the Tethering card says "Router's link only … (almost no data)").

**Where I hesitate**
- None of the screens shows what it looks like once I'm signed in. I have to trust "adds Repeater back to bonding by itself". Screen 16 also says "(until it's done, or 10 minutes)" but not what happens if I'm still typing my room number after 10 minutes. **Low.**

## Task 9: Hotel Ethernet cable with a sign-in page

**Where I click:** in 17, the box "Ethernet needs you to sign in" has **Open sign-in page**. The hotel page opens in a new tab. In 17b, "Signing in to Ethernet…" offers **Open the sign-in page again**, **I've signed in, check now** and "Stop sign-in mode". Clear, with three useful buttons.

**Where I hesitate**
- "For up to 9 more minutes (until 05:34 on this router's clock)". As a traveller I'm often in a different time zone from the router's clock, so "05:34" doesn't help me. The "9 more minutes" part does. **Low.**
- In 17, the first line of the box, in small print, says "GL's Login Mode covers the repeater Wi-Fi only; tailbond handles sign-in pages on other connections like this one." The larger, main sentence comes after it. I read the box in the wrong order. **Low.**

## Task 10: At home, set up for best speed

**Flint 3:** when it's freshly set up (13), the status card says "Home server is on · 1 setup step left". Step 1 says "Easiest: turn on APPLICATIONS → Dynamic DNS and paste the name it shows here (it looks like abc1234.glddns.com)". That's GL language, so I can do it. With relayed machines (18), the "Setup health · 2 to improve" fix is "On this router: NETWORK → Firewall → Open Ports on Router, add UDP 41641". I can do that too. With carrier-grade NAT (18b), the box is red ("1 to fix") and says "Ask your provider for a public IPv4 address … Not sure? Show this to whoever set up your internet." When everything is right (18c), it says "All good / Everything is set up".

**Fedora (14 / 15):** the Connections tab shows a banner, "Setup health: 2 optional speed-ups (tailbond works fine without them). Show". The details are on the Devices tab, with a fix ("forward UDP port 41641 to this machine (192.168.1.2)").

**Where I hesitate**
- Some of the wording is still engineer-speak: "This network makes direct Tailscale connections hard … ('hard NAT')", "carrier-grade NAT" as a heading, "Optional route 10.77.0.0/24 approved". The fixes themselves are concrete, so I'd still manage. **Low.**
- In 13, the "Optional: let your Tailscale machines reach tailbond devices" box is orange with a warning icon, exactly like the required Step 1, even though it says "Skip this unless…". Since my task is "make sure everything is set up", I'd probably go into the Tailscale admin console and do it anyway, just in case. **Low.**
- On Fedora, setup health lives under the **Devices** tab, which isn't where I'd look for "is my setup fast". The **Show** link on the main tab saves it. **Low.**

---

## Findings

| # | task | finding | severity |
|---|---|---|---|
| 1 | 1 | Fedora step 2 ("pick the file, Install.") doesn't say to leave LuCI's "Allow overwriting conflicting package files" unchecked (the Flint page does) | low |
| 2 | 1 | Nothing confirms the one "router package" fits a Slate 7; the Flint page's "It's the same tailbond this router runs" briefly suggests it's Flint-only | low |
| 3 | 1 | No success screen after LuCI's Install and no way back from LuCI to GL's panel; step 3 just says "Back in the router's admin page" | low |
| 4 | 2 | While waiting (04), the status card still says "Not connected to a home server / Not set up yet" | low |
| 5 | 4 | Data limit "restarts on the 1st of each month"; I can't match it to my phone plan's billing day | low |
| 6 | 5 | "(not this router's own Tailscale)" leaves me unsure whether GL's Tailscale switch still matters; when off it only says "The list appears once tailbond is connected." | low |
| 7 | 6 | "Ask to connect again" still shows when connected and when off; unclear vs the main switch | low |
| 8 | 6 | Two different "off" screens: 09 "tailbond is off" and 10 "tailbond isn't running … Start tailbond" | low |
| 9 | 6 | (screen 08) home unreachable means "devices on this router have no internet" until tailbond is off; a surprise compared with GL's Tailscale, but well explained with a button | low |
| 10 | 7 | Approval text is dense ("they allow it whatever they allow homebox, which may be more than its owner's own account gets"); "If this router stays with you, just approve" is buried | low |
| 11 | 8 | No "signed in, back to bonding" confirmation screen; "or 10 minutes" doesn't say what happens after that | low |
| 12 | 9 | "until 05:34 on this router's clock" doesn't help when travelling across time zones | low |
| 13 | 9 | In 17 the small "GL's Login Mode covers the repeater Wi-Fi only…" line comes before the main message, so I read it in the wrong order | low |
| 14 | 10 | Jargon in setup health: "hard NAT", "carrier-grade NAT", "Optional route 10.77.0.0/24" | low |
| 15 | 10 | The "Optional: let your Tailscale machines reach tailbond devices" box looks like a required warning (same orange as Step 1) | low |
| 16 | 10 | Fedora setup health sits under the Devices tab (the Show link on the main tab rescues it) | low |

Totals: 0 high / 0 medium / 16 low
