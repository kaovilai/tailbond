# Round 4: Persona (a), daily Tailscale Android user (product designer, not a networking person)

Screens judged: `out/ux/after-4/` only.

## Task 1: Install tailbond and pair it with the home server, from the phone

**Tap path:** The phone's browser opens the home server page (web-01) → "Download tailbond for Android" → install the APK → back to the page → "+ Add this phone" (web-02) → "Open in tailbond" → app opens the "Add this phone to homebox?" dialog (03) → "Add phone".

**Stuck points:**
- **No way to reach the web page in the first place.** The app isn't installed yet, so the only way in is the home server page, but nothing tells me its address. In my Tailscale app I can see and copy `homebox.tail1234.ts.net`, but the page is at `…:8077`, and nothing I can see would teach me that port. The app's onboarding (01) says "You'll need your home server's name or the link to its page. If someone else set it up, ask them for it." That text is inside the app I don't have yet. The app's pairing screen (02) can turn a "Home server's Tailscale name" into "Open home server page", but that also needs the app installed first. So to get started I have to ask someone for a link. *Tailscale-like expectation:* a store install or a short link I can find myself. If the app has to come from the server, typing the machine name I already know from Tailscale should be enough to get there. **Medium.**
- **Exit-node-style routing is on by default and only mentioned in small print.** The 03 dialog says in its smaller text: "your internet traffic goes out from home (you can change that in Settings)." In Tailscale, an exit node is something I choose to turn on. Here, after "Add phone", all my browsing goes through home without my choosing it. Settings (30) confirms "Through your home server … like using an exit node" is selected. *Expected:* off by default, or a clear choice in the dialog. **Medium.**
- Pairing screen (02), "Home server's Tailscale name": it doesn't say whether I type `homebox` or `homebox.tail1234.ts.net`. The hint "It's the machine that runs tailbond, listed in the Tailscale app's Machines" doesn't help much, because the Tailscale Machines list doesn't mark which machine runs tailbond. **Low.**

## Task 2: Connect

**Tap path:** Home switch → "Switch from Tailscale to tailbond?" (04) → "Turn on tailbond" → Android OK → "Connected · Through homebox · Wi-Fi + mobile data" (10). The big toggle and status card look like Tailscale, and that works well.

**Stuck points:**
- **"This device" shows `10.77.0.2`, not a 100.x address, and no MagicDNS name.** The card says "Your Tailscale machines can reach this phone at this address, through homebox". In Tailscale, This device is my 100.x IP and `name.tailnet.ts.net`, and I copy those to share. A 10.x address makes me doubt that the NAS or laptop can really reach it. `pixel-8` also isn't in the Machines list the way a tailnet device would be. I'd copy this address and probably use it in the wrong place. **Medium.**

## Task 3: Find and copy the NAS's MagicDNS name or Tailscale IP

**Tap path:** Machines → "Search machines" → type `nas` → copy icon → toast "Copied 100.101.2.2" (18). Or tap the row → detail screen (19) "Tap an address to copy it" → MagicDNS name copy → "Copied nas.tail1234.ts.net". Both are quick and feel like Tailscale.

**Stuck points:**
- In the list, the row's copy icon sits beside the MagicDNS name, but it copies the IP ("Copied 100.101.2.2"). I only find out which one I got from the toast. **Low.**

## Task 4: Tell whether Wi-Fi and 5G are both being used right now

**Tap path:** Home (10) → "Using both Wi-Fi and mobile data", with both tiles showing a green "In use". This is clear, and the one-failed state (15, "Using Wi-Fi only" / "Not working right now") is also clear. No problems.

## Task 5: Make it save cellular data

**Tap path:** Home → "Save mobile data" switch on → "Using Wi-Fi · saving mobile data (backup only)", mobile tile "Not in use · saving data" (16).

**Stuck points:**
- On "More options" (21) there are four controls that sound like they save data: the "Save mobile data" radio, the Mobile data "Use for tailbond" switch, "Faster switch to backup (experimental) … Uses a little mobile data", and "Mobile data limit". The home switch alone is enough, and the page explains "The Save mobile data switch on the home screen picks 'Save mobile data' here". Still, it's a lot to read through when I wander in. **Low.**

## Task 6: Turn tailbond off and go back to the Tailscale app

**Tap path:** Home → "Switch back to Tailscale" → result screen (12b): "Stopped. Tailscale is in charge on this phone." + "Open Tailscale" → turn Tailscale on there. This works.

**Stuck points:**
- On the home screen, the "Switch back to Tailscale" button doesn't say it also turns tailbond off. Only Settings (30) says "Turns tailbond off and opens the Tailscale app." I hesitated over whether to flip the switch off first. **Low.**
- The 12b screen still says "This phone reaches your tailnet through homebox" and "Will use Wi-Fi and mobile data together" right below "Tailscale is in charge on this phone." That contradicts the line above it, so for a moment I wasn't sure who is actually routing my traffic. **Low.**

## Side notes from error screens (not part of a task)
- 14-error-cert: "tell whoever set it up. If homebox was just reinstalled, they can add this phone again (Settings → Pair again)". "They" sounds like the admin, but "Settings → Pair again" is a button in my app. It's unclear who is supposed to tap it. **Low.**

## Summary table

| # | issue | severity | screen |
|---|---|---|---|
| 1 | Can't reach the server page (and the APK) without asking someone for the link: the `:8077` page address can't be found from the Tailscale app, and the "ask them" text only appears inside the not-yet-installed app | medium | web-01, 01, 02 |
| 2 | All internet traffic goes through home (like an exit node) by default; mentioned only in small print in the pairing dialog | medium | 03, 30 |
| 3 | "This device" shows a 10.77.0.2 address with no MagicDNS name; doesn't match Tailscale's 100.x + name, so I'd doubt it or share the wrong address | medium | 10, 15, 16 |
| 4 | "Home server's Tailscale name": unclear whether to type the short or full name; the hint relies on a Machines list that doesn't mark the tailbond server | low | 02 |
| 5 | List copy icon copies the IP though it sits beside the MagicDNS name; only the toast says which | low | 18 |
| 6 | Several data-saving controls on More options (radio, Use for tailbond, Faster switch to backup, limit) | low | 21 |
| 7 | Home "Switch back to Tailscale" button doesn't say it turns tailbond off | low | 10, 12 |
| 8 | After switching back, the screen still says "reaches your tailnet through homebox" / "Will use Wi-Fi and mobile data together" | low | 12b |
| 9 | Cert error: "they can add this phone again (Settings → Pair again)": unclear who should do it | low | 14 |
