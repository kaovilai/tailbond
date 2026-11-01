# Round 2: persona (b), homelab power user

Screens: `out/ux/after-2/`. I judged only what's on screen. My baseline is the official Tailscale Android app.

## Task 1: Install and pair from the phone

**Tap path:** In the browser, open the home server's page (web-01): tap "Download tailbond for Android", install it, come back, keep "pixel-8", tap "+ Add this phone". That shows web-02 ("Opening tailbond… In the app, tap Add phone"). Then tap "Open in tailbond". The app shows the 03-deeplink-confirm dialog "Add this phone to homebox?". Tap "Add phone".
In-app alternative (02-pairing): type the "Home server's Tailscale name" and tap "Open home server page".

**What works:** Two numbered steps. The web text says "Android turns Tailscale's VPN off; your Tailscale machines stay reachable through this server". The confirm dialog shows the public endpoint ("It connects over the internet to home.example.net:443") and the device name. I wasn't stuck.

**Stuck points:**
- The web page header says only "Home server · home.example.net:443". The name "homebox" never appears on the web page, but the app calls the server "homebox" everywhere. I opened this page by its tailnet name, so the page should show the same machine name I see in the Tailscale Machines list. **Low.**
- 02-pairing-paste-error: the error says "That isn't a complete tailbond link. Copy the whole link starting with tailbond://pair", but the field is empty. I can't see what I pasted, so I can't tell how it was cut off. **Low.**

## Task 2: Connect

**Tap path:** Home screen toggle (12-home-disconnected). Dialog 04 "Turn off Tailscale's VPN?" → "Turn on tailbond" → Android's VPN prompt → OK. Shows 11 "Connecting… Reaching homebox…" then 10 "Connected · Through homebox · Wi-Fi + mobile data".

**What works:** The handoff is explained before Android's prompt appears. The card and switch work like Tailscale's. The error cards (13, 14) name the endpoint and the fix ("forward UDP port 443", "Pair again").

**Stuck points:**
- 13-error-unreachable: "Not connected / Stopped. See below for how to fix it." Nothing says whether it keeps retrying on its own. Tailscale keeps retrying, so here I have to press "Try again" myself. **Low.**

## Task 3: Find and copy the NAS's MagicDNS name or IP

**Tap path:** Home → Machines → type "nas" in "Search machines" (18) → tap the copy icon. Or tap the row to open 19-machine-detail, which lists "MagicDNS name", "Short name", "Tailscale IPv4", "Tailscale IPv6", each with its own copy button.

**What works:** Online dots, OS, "Last seen 42 h ago", search, and the detail page all match Tailscale. The "Home server" tag on homebox helps. I finished quickly.

**Stuck points:**
- 18: The row's main line is "nas.tail1234.ts.net", but the row's copy icon copied "Copied 100.101.2.2". Nothing tells you which value the icon copies until the toast appears. I wanted the MagicDNS name and got the IP. **Low.**
- Reach beyond tailnet nodes isn't stated anywhere. 01 says "NAS and other Tailscale machines stay reachable, by name or 100.x address". 30 says "only your tailnet goes through home". 19 says "apps … can use these addresses just like with Tailscale". Nothing says whether **subnet routes** advertised by my subnet routers (e.g. 192.168.1.0/24) are reachable through homebox. Nothing says whether non-MagicDNS split-DNS names resolve. With Tailscale those just work. Here I'd have to test by hand. **Medium.**

## Task 4: Are Wi-Fi and 5G both in use right now?

**Tap path:** Home (10): "Using both Wi-Fi and mobile data", with both tiles "In use". Tap Connections (20) for per-link ↓/↑, delay and graphs.

**What works:** The answer is clear on the home screen. Failure states are clear too: 15 "Using Wi-Fi only / Not working right now", 17 "Wi-Fi isn't working · using mobile data". This is better than anything Tailscale shows.

**Stuck points:**
- The home tiles' numbers don't match the Connections page. The home Wi-Fi tile says "42.5 Mbps" and mobile says "25.6 Mbps". Connections says Wi-Fi "↓ 38.4 Mbps ↑ 4.1 Mbps" and mobile "↓ 22.7 Mbps ↑ 2.9 Mbps". The home number is apparently down + up added together, with no label. I read it as download (or link capacity), and then the two screens disagree. **Medium.**

## Task 5: Save cellular data

**Tap path:** Home: switch "Save mobile data / Use mobile data only when Wi-Fi stops working" on (16). The mode line changes to "Mode: Save mobile data" and the tile says "Not in use · saving data". Optionally set "Monthly limit in GB" on Connections (21).

**What works:** One switch does it, and the result is visible (16, 17 "In use · taking over").

**Stuck points:**
- The same setting has three overlapping controls:
  - the home switch "Save mobile data";
  - "Mode: … Change" → the radio list, which also contains "Save mobile data";
  - a per-link "Role" dropdown (Wi-Fi "Main", mobile "Extra"), described as "Adds speed. In "Save mobile data" it waits as a backup".

  Plus "Faster switch to backup (experimental)". Nothing says how they combine. For example: Role "Main" on mobile while in Save mobile data, or the switch on while the radio is set to "Most reliable". As a power user, I expect one source of truth for the routing policy. **Medium.**
- 20/21 "Counts only what tailbond sends over mobile data". It's unclear whether "sends" means upload only or both directions. Carriers bill both ways. **Low.**

## Task 6: Turn tailbond off and go back to Tailscale

**Tap path:** Home → "Switch back to Tailscale" (or Settings → "Switch back to Tailscale / Turns tailbond off and opens Tailscale"). Result 12b: "Tailscale is in charge on this phone … If Tailscale isn't connected yet, open it and turn on its switch." → "Open Tailscale" → turn Tailscale's switch on.

**What works:** The option is easy to find in every state, including the error states. The Always-on section warns about "Block connections without VPN".

**Stuck points:**
- Settings promises "Turns tailbond off and opens Tailscale", but 12b leaves me inside tailbond with a separate "Open Tailscale" button. It hedges: "If Tailscale isn't connected yet". Tailscale's VPN was revoked, so it won't be connected. Better to say plainly that you'll have to switch Tailscale back on yourself, or actually open Tailscale as promised. **Low.**

## Cross-cutting

- Exit node: "Internet traffic → Through your home server — like using an exit node" or "Only Tailscale machines". I can't pick one of my existing tailnet exit nodes the way Tailscale's "Exit node" row allows. I also can't tell whether homebox itself forwards through an exit node. Nothing is broken, but the routing isn't fully clear to someone who runs exit nodes. **Low.**

## Summary

| # | issue | severity | screen |
|---|---|---|---|
| 1 | Subnet routes / split DNS reachability never stated; only "Tailscale machines … by name or 100.x" | medium | 01, 19, 30 |
| 2 | Home link tiles show an unlabeled Mbps (down+up summed: 42.5 = 38.4+4.1) that doesn't match the Connections page | medium | 10, 20 |
| 3 | Save-data policy spread across home switch, Mode radio list, per-link Role dropdown and "Faster switch to backup"; how they combine is unclear | medium | 10, 16, 20, 21 |
| 4 | Machines row copy icon copies the IPv4 while the row headline is the MagicDNS name; no hint before the toast | low | 18 |
| 5 | Paste error leaves the field empty, so the bad paste can't be inspected | low | 02-pairing-paste-error |
| 6 | Web page never shows the server name "homebox", only home.example.net:443 | low | web-01, web-02 |
| 7 | "Stopped" on unreachable: no auto-retry indicated (Tailscale keeps retrying) | low | 13 |
| 8 | "Counts only what tailbond sends over mobile data": unclear if download counts | low | 20, 21 |
| 9 | "Turns tailbond off and opens Tailscale" vs 12b staying in tailbond with hedged "If Tailscale isn't connected yet" | low | 30, 12b |
| 10 | No choice of tailnet exit node; homebox's own egress not shown | low | 30 |

Counts: high 0, medium 3, low 7.
