# Simulated user testing: tailbond on GL.iNet routers

Goal: setting up tailbond on GL.iNet routers is as easy as GL's built-in
Tailscale page. Method (same as [the Android loop](summary.md)): render
every panel state inside a mock of the GL 4.x admin shell (left menu,
header) at desktop 1280x800 and phone 390x844, full page, with
`scripts/ux/glinet/glinet-screens.sh round-N`. The view is the real file
from `packaging/openwrt/gl-ui`, evaluated the way GL's app does; only
`$rpcRequest` answers from fixtures (`scripts/ux/glinet/fixtures.mjs`).
The install path (GL Advanced Settings, LuCI upload) and the Fedora
server's own page are rendered too.

Each round, four **fresh** reviewer agents get only the PNGs, the
[task script](glinet/task-script.md) and the
[manifest](glinet/screens.md). Raw reports: `glinet/round-N/`.
Each point is triaged **fix / partial / reject (reason)**; rejected points
aren't re-argued. Exit: no medium or high from any persona (max 6 rounds).

Personas:
- **A**: traveller used to GL's Tailscale toggle;
- **B**: homelab user (subnet routers, OpenWrt, Fedora);
- **C**: less-technical family member;
- **D**: long-time Speedify user.

Renders: `out/ux/glinet/round-N/` (not committed; `out/` is ignored).

## Findings per round

| Round | A (H/M/L) | B (H/M/L) | C (H/M/L) | D (H/M/L) | Total H/M/L |
|---|---|---|---|---|---|
| 1 | 0 / 3 / 12 | 0 / 3 / 22 | 0 / 6 / 11 | 0 / 3 / 13 | 0 / 15 / 58 |

## Round 1 triage

Mediums (all fixed or partially fixed):

| # | Finding (personas) | Sev | Decision |
|---|---|---|---|
| 1 | "Backup only" checkbox and the "Save mobile data" mode overlap; the mode stays "Fastest" after ticking backup (A, B, C, D) | M | **fix**: one place. Each WAN card has a Speedify-style choice **Always / Backup only / Off**; the mode picker ("Speed or reliability") shows Fastest / Smooth calls and video / Most reliable and says "Saving your phone's data? Set Tethering or Cellular to Backup only". Failover still exists (contract) and shows only if selected elsewhere. Unconfigured tethering/cellular default to *secondary*, so failover still keeps them on standby |
| 2 | Install path (GL → Advanced Settings → LuCI → Upload) not discoverable from GL's panel (A, C, D) | M | **partial**: GL's Plug-ins page installs from GL's repository only, so a local package must go through LuCI. The home side now shows the steps where you start: the Flint 3 panel and the Fedora page's Devices tab have **Add a GL.iNet travel router** (get the file, Advanced Settings → Go To LuCI → Software → Upload, then APPLICATIONS → tailbond → Ask to connect); a Fedora server can offer the file itself (`GET /tailbond-glinet.ipk`). The render now includes the GL Advanced Settings page |
| 3 | "Allow overwriting conflicting package files" ticked and greyed out (B, C; A, D low) | M | **reject** (render artifact): LuCI leaves it unticked; the mock was wrong and now matches LuCI's confirmation dialog |
| 4 | Flint setup never says UDP 443 must reach it; you only learn at the hotel (A) | M | **fix**: the Flint panel lists *Port: UDP 443, opened on this router automatically; forward it on any router in front*; the unreachable error tells the person who set it up what to check |
| 5 | Server unreachable: do my devices still have internet? (C) | M | **fix**: says plainly that devices are offline meanwhile (kill switch) so nothing bypasses home, with **Turn tailbond off** next to **Try again** |
| 6 | Fresh Flint says "Home server is on · Ready" over warnings full of jargon (C) | M | **fix**: hero says "Home server is on · 2 setup steps left" (orange) until done; steps are numbered and plain ("allow tailbond devices on your Tailscale network", "tell travel routers where home is") with the address field inline; the setup card moves above Devices while unfinished |
| 7 | Waiting screen has no address for a Fedora home server's page (C) | M | **fix**: lists both: Flint 3 → APPLICATIONS → tailbond; computer → link `http://homebox:8077` (any device on the Tailscale account) |
| 8 | Which endpoint does the bond dial? (B) | M | **fix**: Slate's home server card says "Connects to home.example.net over the internet (UDP port 443)"; the Fedora header says "reachable at home.example.net:443" (the public address, not the ts.net name) |
| 9 | Mode picker on both the home server and the travel router (D) | M | **fix**: the server role has no mode picker (Fedora page hides it; the Flint panel never had one); each device picks its own |

Lows fixed: "Names like nas.…ts.net" row reworded ("Tailnet names … use the full name, like nas.tail1234.ts.net") (A, B, C, D);
IPs no longer wrap mid-number on phones (B, C, D); full names vs short names stated (A, B, D);
"Off/Off." title (A, D); one "Ask to connect" when a server was found, typed name behind a link (A, D);
placeholder shortened (A); "tailnet" jargon on the pairing card (A, C); card and status line use the same names (D);
not-connected WANs say they're used automatically once connected (C, D); "Pair again" → "Ask to connect again" (B, C);
cap resets "on the 1st of each month" and standby uses no data (B, D); role-switch link says you can switch back (C, D);
server device rates say downloading/uploading (B).
Lows rejected or deferred: SSID on the repeater card (B; agent doesn't read GL's wireless state yet), per-connection graphs on the router (D), checksum/signature (B: releases page concern), logs in the panel (B; :8077 advanced page has them), shared-node flagging and autoApprovers (B), Fedora phone view leading with "Add this phone" (A, C: the phone view of the server page is for adding that phone, decided in the Android loop).
