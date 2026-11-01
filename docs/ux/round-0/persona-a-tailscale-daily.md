# Round 0: Persona A, daily Tailscale Android user (product designer, not a networking person)

What I'm used to: the official Tailscale app. It has a big connect toggle and a "This device" row with my 100.x IP and MagicDNS name (tap to copy). It has a Machines list with green dots, names, IPs, search and tap-to-copy, an Exit node row, and my tailnet/account in the top bar. Settings are simple. I judged only from the screenshots in `out/ux/before/`.

---

## Task 1: Install tailbond and pair it with the home server, from the phone

**Tap path:** Open the app and land on 01 ("Pair with your server"). Then either tap **Scan QR code**, or copy a link somewhere, tap **Paste**, then **Import**. Opening a link from another app gives 03 ("Pair with this server?"), where I tap **Pair**.

**Stuck points:**
1. **I don't know where the code comes from.** The screen says "On your tailbond server's dashboard, open Clients → Add client, then scan its QR code or paste the tailbond://pair link." I don't know what or where the "tailbond server's dashboard" is. There's no address, no "open dashboard" button, and no "ask your admin for a link". The person who set it up didn't mention a dashboard. With Tailscale I tap "Log in" and a browser opens, and I never hunt for a QR code. Also, "Scan QR code" needs a second screen showing the QR, but I'm doing this from the phone only. Pasting is the only phone-only route, and I still need to get the link from somewhere. **High.**
2. **Bad-paste error is jargon and gives no next step** (02). The error says "pairing data is not base64url". I don't know what base64url is or what I did wrong. Did I copy half the link? Is it the wrong link? Also, the field shows only the placeholder text "tailbond://pair?d=…" in red, so I can't see what I actually pasted. **Import** stays greyed out with no reason given. Expected: something like "This doesn't look like a tailbond pairing link. Copy the whole link from the server and try again." **Medium.**
3. **Confirm dialog is full of terms I can't judge** (03). It shows "TLS name: homebox.tail1234.ts.net", "Mode: speed", "DNS: 10.77.0.1", "Tailnet routes: 100.64.0.0/10, fd7a:115c:a1e0::/48" and "Certificate: verified by the system trust store". I'm asked to approve something I can't evaluate, so I'd tap **Pair** blindly. Tailscale's equivalent is one line: "Log in to tailnet X as you@…". **Medium.**
4. **"Name: pixel": is that my phone or the server?** The heading is "Pair with this server?", so I'd first think the server is called "pixel". Also, "Server: home.example.net:443" and "homebox.tail1234.ts.net" look like two different servers. **Low.**
5. **No warning that this replaces Tailscale.** I was told only one VPN app can run at a time. Nothing in onboarding or the pair dialog says "Tailscale will be disconnected while tailbond is on". I'd expect a line about it before I commit. **Medium.**
6. Installing the app itself isn't shown, so I can't judge it. There's no indication of where to get the app either.

---

## Task 2: Connect

**Tap path:** Dashboard (12 "Disconnected"), then tap the big toggle at top right. That gives 11 "Connecting…" and then 10 "Connected". This matches Tailscale's big toggle, and I found it right away.

**Stuck points:**
1. **The address isn't a Tailscale address.** The connected card says "pixel · homebox.tail1234.ts.net · 10.77.0.2". I expect a 100.x IP and a MagicDNS name for "this device". 10.77.0.2 makes me wonder whether I'm actually on my tailnet. Nothing says "You can reach your tailnet". The line also isn't obviously tappable to copy, the way Tailscale's "This device" row is. **Medium.**
2. **Errors give no next step** (13, 14). One says "Error / Connection to the server failed" and the other says "Error / TLS handshake failed". I don't know what a TLS handshake is, whose fault it is, or whether retrying helps. There's no Retry button and no "Contact whoever set up your server" hint. The toggle just flips back off. Tailscale usually shows a banner with an action. **Medium** for 13 and **High** for 14, because "TLS handshake failed" gives a non-technical user nothing to go on.
3. **Empty metrics while off or connecting.** The screen still shows "Download 0 bps / Upload 0 bps", an empty chart and "60 s". It's noise, and "0 bps" makes it look broken rather than off. **Low.**
4. No tailnet or account name in the top bar, only "tailbond". In Tailscale I always see which tailnet I'm in. **Low.**

---

## Task 3: Find and copy the NAS's MagicDNS name or Tailscale IP

**Tap path:** Dashboard (10/10-full), then Settings (30). There is no Machines, Devices or Peers screen anywhere. The app has only two tabs, **Dashboard** and **Settings**.

**Stuck points:**
1. **I can't do this task in tailbond.** There's no list of tailnet machines, no `nas` entry, no search and no tap-to-copy. The closest thing is Settings → Routing → "Tailnet + home LAN only 100.64.0.0/10, fd7a:115c:a1e0::/48, 10.77.0.0/24", which is address ranges, not machines. Tailscale is off while tailbond runs, so I can't easily look it up there either. This is the single biggest gap compared with the Tailscale app. **High.**
2. **I can't tell if MagicDNS names will work at all.** Pairing showed "DNS: 10.77.0.1", and nothing says names like `nas` or `nas.tail1234.ts.net` will resolve. I'd type `nas` into my file app and hope. **High.**

---

## Task 4: Tell whether Wi-Fi and 5G are both being used right now

**Tap path:** Dashboard (10), then scroll (10-full). The Wi-Fi card shows "active" and the Cellular card shows "active", so both are in use.

**Stuck points:**
1. **Cellular is below the fold.** On the first screen (10), the Cellular card is cut off by the bottom nav and only "Cellular active" peeks out. A one-line summary near the top would answer this at a glance ("Using Wi-Fi + Cellular"). **Low.**
2. **"Priority: Primary / Secondary" contradicts "active / active".** In Speed mode, Wi-Fi says "Priority Primary" and Cellular says "Priority Secondary", yet both say "active". "Secondary" makes me think cellular is a backup and isn't really being used. **Medium.**
3. **Ethernet is "unavailable" but its toggle is ON and it says "Priority Primary".** That gives two "Primary" links, and one doesn't exist. **Low.**
4. **A failed link is shown misleadingly** (15). Cellular shows "failed", but next to that it says "RTT 0 ms" and "loss: tunnel 0.3%". There's no reason given and no hint of what to do. The top still says "Connected" with no warning that I'm down to one link. **Medium.**
5. **Takeover isn't announced** (17). When Wi-Fi shows "failed", cellular becomes "active" and I'm now silently using mobile data. There's no banner or notice like "Wi-Fi lost, now using cellular". That matters a lot to someone trying to save data. **Medium.**
6. **Jargon in the stats.** "RTT 21 ms", "loss: tunnel 0.3%" and the cramped line "60 s↓ 1.84 GB ↑ 212.0 MB · RTT 27 ms · loss 0.3% · 12:36" don't help me. "60 s" runs straight into "↓", which reads like "60 s↓". **Low.**
7. **Status chips look like buttons.** The outlined pills "active", "idle", "failed" and "standby" look tappable. **Low.**

---

## Task 5: Make it save cellular data

**Tap path:** I found three candidates and couldn't tell which one is right.
- (a) Dashboard mode bar → **Failover** ("One primary link, others only on failure"). In 16, Cellular then shows "standby". This seems right.
- (b) Turn off the toggle on the Cellular card. That probably stops cellular completely, so I'd lose the backup.
- (c) Settings → Cellular data → "Monthly cap (MB, empty = none)".

**Stuck points:**
1. **Mode names are cut off.** The mode buttons read "Streamin" and "Redundan". It looks broken, and I can't guess what the modes do until I tap each one and read the one-line caption. **Medium.**
2. **No mode says "save data".** I had to infer that "Failover" saves data from "others only on failure". Nothing warns that "Redundan(t)" probably *uses more* cellular data. I might pick it because "redundant" sounds safe. **Medium.**
3. **Cap and dashboard disagree.** Settings shows a cap of "5000", but the dashboard card says "Cellular data this month 1.23 GB (no cap)". Which one is true? I'd assume my cap isn't saved. **High.**
4. **Cap explanation is jargon.** It reads "At the cap, cellular stops carrying paths until next month." I don't know what "carrying paths" means for me. Does my phone stop working on 5G, or only tailbond? **Medium.**
5. **Settings has a "Failover" section I can't parse.** It's called "Experimental: backup_fec scheduler" with the text "Failover mode only. Uses mqvpn's FEC-protected backup scheduler." I can't tell if I should turn it on to save data. It also duplicates the "Failover" name from the dashboard. **Medium.**
6. **The blue "Primary"/"Secondary" text looks tappable** (link-coloured), but nothing explains what changing it does. **Low.**
7. **"Reset" next to "Cellular data this month"** is one tap from wiping my usage count. The screenshots show no confirmation. **Low.**

---

## Task 6: Turn tailbond off and go back to the Tailscale app

**Tap path:** Dashboard → big toggle off, which gives 12 "Disconnected". Then leave the app and open Tailscale.

**Stuck points:**
1. **Nothing confirms Tailscale can take over now.** "Disconnected" doesn't say whether tailbond has released the VPN slot. I'd expect a hint like "tailbond is off; you can now turn on Tailscale". **Medium.**
2. **Always-on VPN trap.** Settings → VPN says "To keep tailbond on all the time, enable Always-on VPN … in system settings" and offers **Open VPN settings**. If I followed that earlier, Tailscale won't be able to connect, and nothing in tailbond warns me or tells me to undo it. **Medium.**
3. **"Forget" vs off is unclear.** Settings → Pairing has "Pair again" and a red "Forget". I wasn't sure whether I need "Forget" to hand back to Tailscale. I'd avoid it because it's red. The toggle is enough here, but the app doesn't say so. **Low.**

---

## General
- The app looks clean, and dark mode (01/10/12/30-dark) reads fine.
- The big toggle is familiar and was the easiest part.
- Overall it reads as a **network stats tool**, not as **my way onto my tailnet**. What I use Tailscale for (find a machine, copy its name or IP, know which tailnet I'm on) is missing.

---

## Issue table

| # | issue | severity | screen |
|---|-------|----------|--------|
| 1 | Pairing assumes a "tailbond server's dashboard" with "Clients → Add client" that I can't find; no way to get the code from the phone alone | high | 01 |
| 2 | Error "pairing data is not base64url" is jargon with no fix; pasted text not visible; Import greyed out with no reason | medium | 02 |
| 3 | Pair dialog lists TLS name, DNS, Tailnet routes (CIDRs), Certificate trust store; can't judge, so I approve blindly | medium | 03 |
| 4 | "Name: pixel" under "Pair with this server?" reads as the server's name; two hostnames (home.example.net vs homebox.tail1234.ts.net) | low | 03 |
| 5 | No warning that pairing/connecting will replace the Tailscale VPN | medium | 01, 03 |
| 6 | Connected shows 10.77.0.2, not a 100.x tailnet IP; no "you're on your tailnet" cue; not tap-to-copy | medium | 10 |
| 7 | "Connection to the server failed": no retry, cause or next step | medium | 13 |
| 8 | "TLS handshake failed": meaningless to a non-technical user, no action offered | high | 14 |
| 9 | 0 bps / empty chart / "60 s" shown while off or connecting looks broken | low | 11, 12 |
| 10 | No tailnet/account name in the top bar | low | 10, 30 |
| 11 | No Machines list, search or tap-to-copy; can't find or copy `nas` | high | (missing) |
| 12 | No indication whether MagicDNS names resolve through tailbond ("DNS: 10.77.0.1") | high | 03, 10 |
| 13 | Cellular card below the fold; no at-a-glance "using Wi-Fi + Cellular" summary | low | 10 |
| 14 | "Priority Secondary" on an "active" link in Speed mode contradicts "both used" | medium | 10-full |
| 15 | Ethernet "unavailable" but toggle ON and "Priority Primary" | low | 10-full |
| 16 | Failed link shows "RTT 0 ms" and "loss: tunnel 0.3%", no reason; top still just "Connected" | medium | 15 |
| 17 | Silent switch to cellular when Wi-Fi fails; no notice that mobile data is now in use | medium | 17 |
| 18 | Jargon and cramped stats line ("RTT", "loss: tunnel", "60 s↓ 1.84 GB …") | low | 10 |
| 19 | Status chips (active/idle/failed/standby) look like buttons | low | 10, 12, 16 |
| 20 | Mode labels truncated: "Streamin", "Redundan" | medium | 10, 20 |
| 21 | No mode framed as "save data"; no warning that Redundant may use more cellular | medium | 10, 16, 20 |
| 22 | Settings cap "5000" vs dashboard "1.23 GB (no cap)": contradictory | high | 30, 10-full |
| 23 | Cap explanation "cellular stops carrying paths" is unclear | medium | 30 |
| 24 | "Experimental: backup_fec scheduler … mqvpn's FEC-protected backup scheduler" is unreadable and clashes with dashboard "Failover" | medium | 30 |
| 25 | Blue "Primary/Secondary" looks tappable, purpose unexplained | low | 10 |
| 26 | "Reset" cellular usage with no visible confirmation | low | 10-full |
| 27 | After turning off, no confirmation that Tailscale can now connect | medium | 12 |
| 28 | Always-on VPN advice with no warning that it blocks Tailscale | medium | 30 |
| 29 | Unclear whether "Forget" is needed to switch back; red label scares | low | 30 |
