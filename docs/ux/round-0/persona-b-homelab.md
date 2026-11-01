# Round 0 - Persona (b): homelab power user

Background: I run Tailscale subnet routers and an exit node at home and use ACLs and MagicDNS. I know the official Tailscale Android app well. I judged only from the screenshots in `out/ux/before/` (01, 02, 03, 10, 11, 12, 13, 14, 15, 16, 17, 20, 30, plus the dark variants). No server web page screenshots were provided.

---

## Task 1 - Install tailbond and pair it with the home server, from the phone

**Tap path**
- No screen shows installing the app. I would sideload it or get it from a store, but nothing tells me how.
- 01: "Pair with your server" says "On your tailbond server's dashboard, open Clients → Add client, then scan its QR code or paste the tailbond://pair link."
- From the phone, I would open the server dashboard in a browser (no screenshot exists, so I can't see it), copy the link, come back, and tap **Paste**, then **Import**. Or I tap the link and land on 03, "Pair with this server?", where I tap **Pair**.

**Stuck points**
1. The only primary button is **"Scan QR code"**. If I'm doing everything on the phone, the QR code is on the same screen I'd need to point the camera at. The flow is built for a second device but doesn't say so. I can't find the dashboard URL anywhere in the app ("tailbond server's dashboard": where is it? On a tailnet name? A LAN IP? Do I need Tailscale on to reach it?). **High**: I could easily get stuck before pairing even starts.
2. The field's placeholder **"tailbond://pair?d=…"** looks like a real pre-filled value. In 02 the same text turns red with **"pairing data is not base64url"**. I can't tell if the field holds my pasted text or still the placeholder. The message is a developer-level error with no fix ("copy the whole link again"). Also, **"Import"** stays greyed out even though there seems to be text in the field. **Medium**.
3. 03 "Pair with this server?":
   - **"Name: pixel"**: is that my phone's name or the server's? Tailscale would say "This device: pixel".
   - **"Server: home.example.net:443"** and **"TLS name: homebox.tail1234.ts.net"** are two different hosts. I can work it out (public endpoint vs. cert SAN), but most users would wonder which one they're connecting to.
   - **"Mode: speed"** isn't explained. Lowercase "speed" doesn't match the "Speed" tab.
   - **"DNS: 10.77.0.1"**: will MagicDNS (`nas`, `*.ts.net`) still resolve through this? It doesn't say.
   - **"Tailnet routes: 100.64.0.0/10, fd7a:115c:a1e0::/48"**: useful to me. But nothing says the default is **Full tunnel** (Settings, 30), meaning all my internet traffic will exit at home. That's exit-node behaviour, and it isn't disclosed at pairing time.
   - Nothing warns that turning this on will **replace the Tailscale VPN** (Android allows only one VPN).

   **Medium**.

**Expected (Tailscale-like)**
Sign-in or pairing that works on one phone: a link or button that opens the server's add-client page, then automatically returns to the app. A confirmation dialog that clearly separates "This device" from "Server", shows the tailnet name, and states what gets routed: "Tailnet only" or "All traffic via home (exit node)". A plain-language warning: "Tailscale will be disconnected while tailbond is on."

---

## Task 2 - Connect

**Tap path**
- 12 "Disconnected": tap the big toggle at top right of the status card. I get 11 "Connecting…", then 10 "Connected".

**Stuck points**
1. The toggle in the status card works like Tailscale's. Good.
2. No screen shows the Android VPN consent dialog or what happens to the running Tailscale VPN. I'd expect Tailscale to drop silently. **Medium**: as a power user I'd figure it out, but nothing on screen says so.
3. 11 "Connecting…" shows "0 bps" and an empty "60 s" chart. Both links say **"idle"**. There's no stage info such as "contacting home.example.net" or "TLS handshake". **Low**.
4. Failures (13, 14): **"Error / Connection to the server failed"** and **"Error / TLS handshake failed"**.
   - There's no Retry button and no detail (which host or port, which link, cert name mismatch vs. expiry).
   - There's no link to logs. The toggle simply flips back off.
   - The line that showed "pixel · homebox.tail1234.ts.net" is replaced by the error text, so I lose which server it tried.
   - Wi-Fi and Cellular still say **"idle"** with the toggles on, which hides whether each link actually reached the server.

   **Medium** (High for a non-technical user facing a cert error).
5. When connected, the header reads **"pixel · homebox.tail1234.ts.net · 10.77.0.2"**. 10.77.0.2 isn't a tailnet address. Tailscale shows "This device: 100.x.y.z" plus the MagicDNS name. I can't tell what IP my phone has on the tailnet, or whether other tailnet machines see me as homebox (subnet-routed or SNAT'd). **Medium**: this matters for ACLs.

**Expected (Tailscale-like)**
A connect toggle, plus a "This device" row with the tailnet IP and MagicDNS name, both tap-to-copy. Errors in plain language with a Retry button and a "details/logs" link.

---

## Task 3 - Find and copy the NAS's MagicDNS name or Tailscale IP (`nas`)

**Tap path**
- 10 Dashboard: no machines list. 30 Settings: no machines list. There are only two tabs, "Dashboard" and "Settings".

**Stuck points**
1. **This task can't be done in tailbond.** There's no Machines/Peers list, no search, no online dots, and no tap-to-copy for any tailnet host. The only tailnet-related text is the CIDR list in Settings ("100.64.0.0/10, fd7a:115c:a1e0::/48, 10.77.0.0/24") and in the pairing dialog. To get `nas.tail1234.ts.net` or its 100.x I'd have to switch back to the Tailscale app, which means turning tailbond off. That defeats the point. **High**.
2. I also can't confirm MagicDNS works through tailbond. The pairing dialog says "DNS: 10.77.0.1", and nothing says whether short names like `nas` or `nas.tail1234.ts.net` resolve. **Medium**.
3. The Settings option **"Tailnet + home LAN only"** lists "10.77.0.0/24" as the "home LAN". That looks like the tunnel subnet, not my real LAN (e.g. 192.168.1.0/24 behind my subnet router). It's unclear whether my advertised subnet routes are reachable. **Medium**.

**Expected (Tailscale-like)**
A Machines list with search, online/offline dots, and long-press or tap to copy the MagicDNS name or 100.x IP. At minimum, a "Tailnet" section that shows the tailnet name and confirms MagicDNS is active.

---

## Task 4 - Tell whether Wi-Fi and 5G are both being used right now

**Tap path**
- 10 (scroll; see 10-full): the Wi-Fi card shows **"active"**, "↓ 38.4 Mbps ↑ 4.1 Mbps", "RTT 21 ms". The Cellular card shows **"active"**, "↓ 22.7 Mbps ↑ 2.9 Mbps", "RTT 48 ms". The sum matches the header's "61.1 Mbps". So yes, both are in use.

**Stuck points**
1. On the unscrolled screen (10), only the top of the Cellular card is visible ("Cellular … active"). I have to scroll to see its throughput. A one-line summary in the header ("Wi-Fi + Cellular bonded") would answer this immediately. **Low**.
2. The card is labelled **"Cellular"** with no carrier name or 5G/LTE indicator. The task says 5G, so I'm assuming. **Low**.
3. **"loss: tunnel 0.3%"** is identical on every link card. In 15 and 17 it still shows "loss: tunnel 0.3%" on a **"failed"** link with "0 B this session". So it's tunnel-wide loss, not per-link loss, and on a dead link it's misleading. **"RTT 0 ms"** on a failed link reads like a perfect link. **Medium**.
4. 15: Cellular shows **"failed"** with no reason (no signal? blocked UDP? cap reached?) and no retry. The header still says just "Connected" with no degraded warning. **Medium**.
5. The status chips (**"active"**, **"idle"**, **"failed"**, **"standby"**, **"unavailable"**) are outlined pills that look like buttons. **Low**.
6. Each link card has an **unlabelled toggle**. I guess it means "allow this link", but there's no text. **Medium** (it's the obvious control for task 5, so it needs a label).
7. **Ethernet: "unavailable"**, Priority **"Primary"**. Wi-Fi is also "Primary", so I see two primaries. It's unclear what "Priority" does in Speed mode at all. **Low**.
8. The header footer **"60 s↓ 1.84 GB ↑ 212.0 MB · RTT 27 ms · loss 0.3% · 12:35"**: "60 s" runs into "↓" and the time wraps onto its own line. Hard to parse: is 1.84 GB per 60 s or per session? **Low**.
9. The mode tabs are truncated: **"Streamin"**, **"Redundan"**. **Medium**: they look broken and I can't be sure of the names.

**Expected (Tailscale-like)**
A clear one-line status such as "Bonding Wi-Fi + Cellular" or "Using Wi-Fi only (Cellular failed: no signal)". Labelled per-link switches, and per-link loss when it's available.

---

## Task 5 - Make it save cellular data

**Tap path (three candidates, none labelled "save data")**
- a) Dashboard mode tabs: tap **"Failover"**. 16 says **"One primary link, others only on failure"**, and Cellular shows **"standby"**. This seems right.
- b) The unlabelled toggle on the Cellular card. It might turn cellular off entirely.
- c) Settings (30), **"Cellular data"**: "Monthly cap (MB, empty = none)", set to 5000. Helper text: "At the cap, cellular stops carrying paths until next month."

**Stuck points**
1. No mode or setting mentions saving data. I have to infer from "One primary link, others only on failure". Speed ("Aggregate bandwidth across every link") and Streaming ("Calls and streams survive link trouble") don't warn that they use cellular continuously. Redundant presumably **duplicates** traffic onto cellular, but its description isn't shown. A user picking "Redundan" to be safe would burn data. **Medium**.
2. **Contradiction:** Settings shows a cap of **"5000"**, but the dashboard card says **"Cellular data this month 1.23 GB (no cap)"** (10-full, 15, 16, 17). I can't tell which is true or whether the cap was ever saved. There's no Save button on the field either. **High**: this would make me distrust the cap.
3. **"Reset"** next to the monthly counter is a one-tap action with no visible confirmation. Does it reset the counter, the cap, or the billing day? **Medium**.
4. "cellular stops carrying **paths**": "paths" is internal jargon. There's no billing-cycle start day, and no in-app warning before the cap is reached. **Low**.
5. Settings "Failover: **Experimental: backup_fec scheduler** … Uses **mqvpn's** FEC-protected backup scheduler": implementation jargon. Does it make standby use *more* cellular data (FEC)? It doesn't say. **Low**.
6. 17 shows the failover taking over correctly: Wi-Fi "failed", Cellular "active". Good. But there's no banner such as "Now on cellular (counts toward data)". **Low**.
7. Mode changes while connected: it's unclear whether they apply live or need a reconnect. **Low**.

**Expected (Tailscale-like)**
A clearly named option such as "Use cellular only when Wi-Fi fails (saves data)". One source of truth for the cap, shown the same on Dashboard and Settings. "Reset" with a confirmation step.

---

## Task 6 - Turn tailbond off and go back to the Tailscale app

**Tap path**
- Dashboard: tap the top toggle. I get 12 "Disconnected". Then I leave the app, open Tailscale, and toggle it on (outside this app, not shown).

**Stuck points**
1. 12 "Disconnected" gives no hint that Tailscale needs to be re-enabled, and no "Open Tailscale" shortcut. The Wi-Fi and Cellular cards still show **"idle"** with their toggles **on**, which looks like something is still armed. **Medium**.
2. Settings "VPN" card: "To keep tailbond on all the time, enable Always-on VPN…". If I followed that advice earlier, Android will keep re-claiming the VPN slot for tailbond and Tailscale won't stay up. There's no warning about this conflict. **Medium**.
3. Settings "Pairing": a red **"Forget"** with no explanation. Is it needed to hand the VPN back? (I'd guess no, but I can't tell, and I'd hesitate.) **Low**.
4. Excluded apps ("Choose") could in theory leave room for Tailscale, but Android allows one VPN app at a time, and nothing explains this. **Low**.

**Expected (Tailscale-like)**
After turning it off: "tailbond is off. Your tailnet is no longer reachable through tailbond. Open Tailscale?" with a button. The Always-on VPN guidance should note the conflict with the Tailscale app.

---

## Cross-cutting
- There's no tailnet name in the top bar, only "tailbond" (Tailscale shows `tail1234.ts.net` or the org name). **Low**.
- "Full tunnel / All traffic goes through your home server" is effectively an exit node, but it isn't called that, and the dashboard doesn't say which routing mode is active. To know what's routed where, I have to open Settings. **Medium**.
- Dark theme (01, 10, 12, 30 dark): readable, with no contrast problems spotted.

---

## Summary table

| # | issue | severity | screen |
|---|---|---|---|
| 1 | No in-app install/source guidance; pairing needs a server dashboard whose URL and access path are never shown | high | 01 |
| 2 | Phone-only pairing: primary action "Scan QR code" can't scan a QR on the same phone; no "open dashboard" link | high | 01 |
| 3 | "pairing data is not base64url": developer error, no fix hint; placeholder vs. real text ambiguous; "Import" disabled with no reason | medium | 02 |
| 4 | Pair dialog: "Name: pixel" ambiguous (device vs. server); "Server" vs. "TLS name" differ unexplained; "Mode: speed" undefined | medium | 03 |
| 5 | Pair dialog doesn't disclose Full tunnel default (exit-node-like) or that the Tailscale VPN will be replaced; MagicDNS through "DNS: 10.77.0.1" unclear | medium | 03, 30 |
| 6 | No VPN-switch notice when connecting while Tailscale is active | medium | 11/10 |
| 7 | Errors "Connection to the server failed" / "TLS handshake failed": no Retry, no detail or logs, server identity line removed, links still "idle" | medium | 13, 14 |
| 8 | Header shows "10.77.0.2", not a tailnet 100.x / MagicDNS name for this device; unknown identity for ACLs | medium | 10 |
| 9 | No Machines/peer list, search, or tap-to-copy; cannot find `nas` at all | high | 10, 30 |
| 10 | No confirmation that MagicDNS names resolve through tailbond | medium | 03, 10 |
| 11 | "Tailnet + home LAN only" lists 10.77.0.0/24 as the "home LAN"; real advertised subnet routes not shown | medium | 30 |
| 12 | "loss: tunnel 0.3%" identical on every link, including failed ones; "RTT 0 ms" on a failed link is misleading | medium | 15, 17 |
| 13 | "failed" link shows no reason; header still a plain "Connected" with no degraded warning | medium | 15 |
| 14 | Per-link toggles unlabelled | medium | 10, 15, 16 |
| 15 | Mode tabs truncated: "Streamin", "Redundan" | medium | 10, 11, 12, 20 |
| 16 | Cellular below the fold; no one-line "Wi-Fi + Cellular bonded" summary; no 5G/LTE indicator | low | 10 |
| 17 | Status chips look like buttons; Ethernet and Wi-Fi both "Primary"; Priority meaning in Speed mode unclear | low | 10 |
| 18 | Header stats run together ("60 s↓ 1.84 GB … · 12:35" wraps); unclear time window | low | 10 |
| 19 | No mode or setting labelled for saving data; Redundant/Speed don't warn about cellular use | medium | 10, 16, 20 |
| 20 | Cap contradiction: Settings "5000" vs. Dashboard "1.23 GB (no cap)"; no save affordance | high | 30, 10 |
| 21 | "Reset" on monthly counter with no confirmation and unclear scope | medium | 10 |
| 22 | Jargon: "carrying paths", "backup_fec scheduler", "mqvpn's FEC-protected" | low | 30 |
| 23 | No banner when failover moves traffic onto cellular | low | 17 |
| 24 | Off state gives no "open Tailscale" hint; link toggles still on and "idle" | medium | 12 |
| 25 | Always-on VPN advice doesn't warn that it blocks returning to Tailscale | medium | 30 |
| 26 | "Forget" unexplained and red, causing hesitation | low | 30 |
| 27 | No tailnet name in top bar; active routing mode (full tunnel / exit-like) not shown on Dashboard | medium | 10 |

Counts: high 4, medium 17, low 6.
