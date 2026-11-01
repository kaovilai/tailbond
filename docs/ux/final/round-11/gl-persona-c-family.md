# Round 11 — GL persona C (family member)

**Who I am:** someone else set up the home server; I was handed a GL.iNet Slate 7 and told to "get it working". I know Wi-Fi, phone hotspots and the GL admin page's menus. I don't know what routes, ports or NAT are.

**What I used:** `task-script.md`, `screens.md` and the PNGs in `packet-gl/` (desktop for every screen, phone for 01b, 05, 14 and 17b). I read nothing else.

**Overall:** I could get through all ten tasks using only the screens. Nothing stopped me. Two places made me hesitate for real: the Fedora approval warning (task 7) and the "best speed" fixes, which I can't do myself (task 10). Everything else is wording or polish.

---

## Task 1: Install tailbond on the Slate 7 without SSH

**Where I'd click**
- On the home server's page I'd click "Download the router package". That's under "Add a GL.iNet travel router (recommended)" on the Fedora page (15), or "Add a GL.iNet travel router" on the Flint (11–13).
- Then I'd follow step 2. On the Slate 7: SYSTEM → Advanced Settings → "Go To LuCI" (01a). The page says "Use the same admin password to log in", which helps.
- In LuCI (01b): System → Software → "Upload Package…", pick the file, then "Install" in the "Manually install package" dialog.

**Where I'd hesitate**
- The LuCI dialog has a checkbox, "Allow overwriting conflicting package files". The Flint's step 2 says "(leave the other options as they are)". The Fedora page (15) only says "pick the file, Install." If I came from the Fedora page I wouldn't know whether to tick it. "Conflicting" sounds like something I should fix, so I might tick it. **Low**
- The file is called "tailbond_0.1.0-r42_aarch64_cortex-a53_neon-vfpv4.ipk". Nothing on the download card says this file is right for my Slate 7 (GL-BE3600). The Flint says "It's the same tailbond this router runs", but it's a different router model, so I'd still wonder. **Low**
- LuCI looks like a completely different website (blue bar, "Status System Services Network Logout"), and I don't see a way back to GL's page. Step 3 says "Back on the travel router's admin page". I'd have to remember the address or close the tab. **Low**
- No screen shows what happens after I click "Install" (a progress bar or "done" message). I can't judge that part, but 02 suggests VPN → tailbond simply appears.

## Task 2: Pair the Slate 7 with the home server

**Where I'd click**
- GL admin → VPN → tailbond (02). It shows "Looking for your home server on your tailnet…".
- A few seconds later (03) it says "Found your home server: homebox", and I click "Ask to connect". This is the easiest step of all.
- 04 then says "Waiting for approval on homebox… On homebox, tap Approve next to slate7. This router connects by itself once approved; you can close this page." That's clear.
- If Tailscale is off (03b), "Turn on Tailscale first" plus "Open Tailscale settings" is clear.

**Where I'd hesitate**
- "tailnet" is a word I don't know. If nothing is found (03c), it says "Or type its Tailscale name below". I don't know the home server's Tailscale name, and the Slate page doesn't say where to find it. (I only know it's on the Flint's page, "Tailscale name homebox.tail1234.ts.net", because I'm reading all the screens.) **Low**

## Task 3: Hotel Wi-Fi (Repeater) + phone over USB, confirm both are used

**Where I'd click:** nothing. 05 says "Connected to homebox — Using Repeater (Wi-Fi) + Tethering (phone)". Both cards show a green "In use" with their own speeds (38 and 21 Mbps, which add up to roughly the 59 Mbps at the top). It's easy to confirm.

**Where I'd hesitate**
- The top line reads "↓ 59 Mbps ↑ 7.1 Mbps Up 2 h 13 min". Right after an up arrow, "Up" looks like "upload", so I'd wonder why the upload is in hours. It actually means "how long it's been on". **Low**

## Task 4: Phone as backup only

**Where I'd click**
- On the Tethering card, the "Use it" row: "Backup only" (07). The badge changes to "Backup · saving data" and the card says "Used only if your other connections stop working. Until then tailbond sends nothing over it (GL itself may check the connection now and then, a few KB)." The top line says "Tethering (phone) on standby". Perfect.
- There's also a hint under "Speed or reliability": "Saving your phone's data? Set Tethering or Cellular to 'Backup only'".
- "Change limit" (07b) is clear too: "5 GB a month … 0 = no limit".

No problems.

## Task 5: Reach the NAS by name from the laptop

**Where I'd click:** 05 → "Tailnet machines" → **nas** → `nas.tail1234.ts.net` → "Copy name". The Home server card adds: "Tailnet names: Work on every device connected to this router while tailbond is on. Use the full name, like nas.tail1234.ts.net."

**Where I'd hesitate**
- The task says "MagicDNS name", and that word isn't anywhere on screen. I'd guess the "full name" is it, and I'd be right. The help text uses words I don't know: "(not this router's own Tailscale)", "'Direct' / 'via relay' is how homebox reaches each machine". I can ignore them, but they make me wonder whether I'm missing something. **Low**

## Task 6: Turn tailbond off

**Where I'd click:** the blue switch at the top right of the "Connected to homebox" card (05). Afterwards 09 says "tailbond is off — Devices on this router use the internet directly…". It also has a helpful warning: "While tailbond is off, GL may use Tethering (your phone's data) … unplug it or turn tethering off on it, or turn tailbond back on."

**Where I'd hesitate**
- The switch has no label on the GL page. The Fedora page puts "Connected" next to its switch. Near the bottom there's a red "Unpair" button, which also looks like "turn it off". The confirmation (09b) explains that Unpair "forgets the home server" and needs re-approval, so nothing breaks. But a word like "On" next to the switch would make me sure I'm clicking the right thing. **Low**

## Task 7: Approve the new travel router at home

**Where I'd click**
- On the Flint (11): "Waiting for your approval — slate7 wants to connect" → **Approve**. 12 then shows "slate7 Connected just now · 2 connections bonded". Easy.
- On the Fedora page (14): the same card at the top → **Approve**.

**Where I'd hesitate**
- The Fedora page (14) shows an orange warning under slate7: "It's signed in to alex@example.com, an account that may approve here: whoever holds this router can also open this dashboard and approve devices. If it's for someone else, sign it in to their own Tailscale account first, then let it ask again."
  - This router *is* for someone else: me. It came already signed in to Alex's account. Should I press Decline and get my own Tailscale account? I don't have one and wouldn't know how to "sign it in".
  - The Flint (11) shows the same "signed in as alex@example.com" with no warning, so the two home servers give different signals for the same situation.
  - I'd stop here and phone the person who set it up. **Medium**
- The "Who can approve" text above it is dense: "Approving lets the device use homebox's internet connection … and reach your Tailscale machines as homebox: they allow it whatever they allow homebox, which may be more than its owner's own account gets." I can't follow it, and "more than its owner's own account gets" sounds alarming just before I press Approve. **Low**

## Task 8: Hotel Wi-Fi with a sign-in page (GL's repeater detected it)

**Where I'd click**
- In 16, the "Repeater needs you to sign in" box says to open GL's Repeater page and click "Login Mode for Public Hotspots" (or "Go to Login"), then open any website. I'd click "Open GL's Repeater page".
- After starting Login Mode (16b), tailbond says "Now open any website on your laptop or phone: the hotel's sign-in page appears there … Login Mode ends by itself once you've signed in … tailbond then adds Repeater back to bonding."
- Once I've signed in, Repeater should be bonded again. No screen shows that "after" moment, but 05 is what I'd expect.

**Where I'd hesitate**
- GL's own Repeater page isn't in my screenshots, so I'm trusting that the button really says "Login Mode for Public Hotspots" or "Go to Login".
- 16 says the direct-to-hotel mode lasts "until it's done, or 10 minutes". 16b doesn't say what happens if I'm still typing my room number after 10 minutes. Do I start over? **Low**

## Task 9: Hotel with a sign-in page on the Ethernet cable

**Where I'd click**
- In 17, the "Ethernet needs you to sign in" box → **Open sign-in page**.
- In 17b, the hotel page opens in a new tab and tailbond shows "Signing in to Ethernet…". It offers "Open the sign-in page again", "I've signed in, check now" and "Stop sign-in mode", and says "tailbond adds Ethernet back by itself once you're signed in." This is the clearest flow in the packet.

**Where I'd hesitate**
- The 17 box starts with small print, "GL's Login Mode covers the repeater Wi-Fi only; tailbond handles sign-in pages on other connections like this one.", above the bigger main sentence. I read the small technical line first and wondered whether I needed GL's Login Mode here too. The main instruction should come first. **Low**

## Task 10: At home, make sure everything is set up for best speed

**Where I'd look**
- On the Flint, the "Setup health" card: 18c shows "All good — Everything is set up for best speed". 18 shows "2 to improve". 18b shows "1 to fix".
- On the Fedora page: the Connections tab banner "Setup health: 2 things to improve (tailbond works meanwhile). Show" (14), and the Devices tab card (15).

**Where I'd get stuck**
- The fixes are things I can't do. On the Flint (18): "For whoever manages this router: NETWORK → Firewall → Open Ports on Router, add UDP 41641. If another router sits in front of this one … also forward UDP port 41641". On the Fedora page (15): "forward UDP port 41641 to this machine (192.168.1.2). (Turning on UPnP on that router also works…)", plus "hard NAT", "relay", "port mapping".
  - I'm at home with the Flint's admin page open, so "whoever manages this router" might be me. But I don't know what UDP 41641 is, and the Open Ports page isn't shown.
  - The line "Not sure how? Show this card to whoever set up your internet" is my way out, and I'm glad it's there. I can't finish this task myself. **Medium**
- Mixed signals on the Flint:
  - 18 shows a green "Setup done" box directly above "2 things to improve".
  - 13 calls the route step "Optional … Skip this unless you want your Tailscale machines … to open something on a device behind a travel router", but 18c lists "Route 10.77.0.0/24 approved" under "Everything is set up for best speed". If I skip it, will I never see "All good"? **Low**
- In 18b, the heading "Your internet line uses carrier-grade NAT" is jargon. The next line is very helpful, though: "Ask your provider for a public IPv4 address (often free or a small monthly fee; say it's for reaching your home from outside)". It's a script I could read out on the phone to my internet company. Only the heading and the "IPv4/IPv6" words are hard. **Low**
- 13 tells me how to set up the home address: "turn on APPLICATIONS → Dynamic DNS and paste the name it shows here". The Dynamic DNS page isn't shown, but the steps are concrete enough that I'd try them.

---

## Findings

| # | Task | Finding | Severity |
|---|---|---|---|
| 1 | 7 | Fedora page (14) warns "It's signed in to alex@example.com … If it's for someone else, sign it in to their own Tailscale account first, then let it ask again". A family member handed a pre-signed router can't tell whether to Approve or Decline, and the Flint (11) shows no warning for the same case | medium |
| 2 | 10 | The best-speed fixes need port forwarding ("add UDP 41641", "hard NAT", "UPnP"), which I can't do myself. The only way forward is "Show this card to whoever set up your internet" | medium |
| 3 | 1 | LuCI dialog checkbox "Allow overwriting conflicting package files". The Fedora page's step 2 doesn't say to leave it (the Flint's does) | low |
| 4 | 1 | Nothing tells me the download "tailbond_0.1.0-r42_aarch64_cortex-a53_neon-vfpv4.ipk" is the right file for a Slate 7 | low |
| 5 | 1 | No visible way back from LuCI to GL's page, though step 3 says "Back on the travel router's admin page" | low |
| 6 | 2 | "tailnet" is unexplained. In 03c, "Or type its Tailscale name below" doesn't say where to find that name | low |
| 7 | 3 | "↑ 7.1 Mbps Up 2 h 13 min": "Up" next to the upload arrow reads as upload, not uptime | low |
| 8 | 5 | The word "MagicDNS" never appears. "(not this router's own Tailscale)" and "Direct / via relay" are jargon | low |
| 9 | 6 | The on/off switch on the GL page has no label, and a red "Unpair" also looks like "turn off" (the confirmation makes it safe) | low |
| 10 | 7 | The "Who can approve" paragraph is dense, and "which may be more than its owner's own account gets" sounds alarming next to Approve | low |
| 11 | 8 | GL's Repeater page isn't shown. "or 10 minutes" in 16 isn't followed up in 16b (what if sign-in takes longer?) | low |
| 12 | 9 | 17 opens with small technical print ("GL's Login Mode covers the repeater Wi-Fi only…") before the main instruction | low |
| 13 | 10 | Green "Setup done" sits above "2 things to improve". The route is "Optional … Skip this" in 13 but listed under "Everything is set up for best speed" in 18c | low |
| 14 | 10 | Heading "Your internet line uses carrier-grade NAT" and "IPv4/IPv6" are jargon (the provider script under it is good) | low |

Totals: 0 high / 2 medium / 12 low
