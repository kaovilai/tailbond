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
| 2 | 0 / 3 / 11 | 0 / 1 / 18 | 0 / 3 / 10 | 0 / 0 / 17 | 0 / 7 / 56 |
| 3 | 0 / 1 / 9 | 0 / 2 / 16 | 0 / 2 / 9 | 0 / 1 / 13 | 0 / 6 / 47 |
| 3 | 0 / 1 / 9 | 0 / 2 / 16 | 0 / 2 / 9 | 0 / 1 / 13 | 0 / 6 / 47 |

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

## Round 2 triage

| # | Finding (personas) | Sev | Decision |
|---|---|---|---|
| 1 | Machine names wrap mid-word on phones ("nas.tail1234.ts.ne" / "t") (C; A, B, D low) | M | **fix**: names break only after a dot; the 100.x address has its own line |
| 2 | The install steps aren't on the Slate 7 itself (C, A) | M | **partial**: before install the Slate only runs GL's own pages, which tailbond can't change, and GL's Plug-ins page has no upload. The steps live where the file comes from (Fedora page Devices tab, Flint panel); the task script now says the file was downloaded there, as it would be |
| 3 | Waiting screen: how to reach the Flint's panel from the hotel (A) | M | **fix**: one link for both kinds of home server, `http://homebox:8077`, which the agent serves on its Tailscale IP (the Flint's firewall rule opens tcp/8077 from the tailnet zone); the Flint admin panel is mentioned for when you're home |
| 4 | Port forwarding line unclear; nothing checks it (A) | M | **partial**: plain question ("Is your internet cable plugged straight into this router? Then nothing else to do"), what to do otherwise; "All set" became "Setup done … the first connection from away confirms the port works". An outside reachability test needs an external service: out of scope |
| 5 | Flint setup jargon: route, public IP, Dynamic DNS (C) | M | **fix**: route step says exactly what to tick; the address step links GL's Dynamic DNS page and says what the name looks like (abc1234.glddns.com) |
| 6 | Fedora header said `homebox.tail1234.ts.net:443`, Slate says `home.example.net` (B) | M | **fix** (render-timing bug): the header was drawn before settings loaded; it now redraws with the public address once they arrive |

Lows fixed: server rates labelled "Sending to devices / Receiving" (D); "At the limit, tailbond stops using this connection" (D);
"tailbondd" → "tailbond" in the Fedora footer (A); "router's Setup tab" → the GL panel's words (B); mode cards stack on phones.
Lows rejected: "Secondary" level and billing-day reset (D; Always already includes secondary), per-connection graphs (D, round 1),
port 80 in the Fedora invite link (B: render artifact, the mock serves the page on port 80; the real one is :8077),
who may approve on the Fedora page (B: anyone who can open its page; set `ui_password` to restrict; documented in docs/fedora-server.md).

## Round 3 triage

| # | Finding (personas) | Sev | Decision |
|---|---|---|---|
| 1 | The Flint panel has no way to download the router package (A, B, C, D) | M | **fix**: a home server on a GL router serves the package it runs: `GET /tailbond-glinet.ipk` rebuilds the installed .ipk from opkg's records, with pristine copies of the files that change after install and never anything under `/etc/tailbond/`. The panel's guide starts with **Download the router package** (same-origin through GL's nginx, GL session required). Covered by a unit test and the emulated install test |
| 2 | Fedora Devices tab: "Add a device" next to "Add a GL.iNet travel router" invites typing the router's name (C) | M | **fix**: the router guide comes first, "(recommended)", ends with "tap Approve. Nothing to type here."; the code form is "Add a device with a code (no Tailscale on it)" and says Tailscale devices just ask |
| 3 | Who can approve? (B) | M | **fix**: both approval cards say who can (Flint: anyone logged in to the GL panel, and anyone on the Tailscale account at `http://homebox:8077` unless it has a password; Fedora: anyone who can open the page), and the Flint panel can set that password |

Lows fixed: "leave the other options as they are" for LuCI's overwrite checkbox and "the router's internet keeps working" (A, B, C); example tailnet name taken from the server's name before the machine list loads (A, B).
Lows rejected: as in rounds 1–2 (graphs, SSID, Secondary, billing day, the Fedora phone view, port 80 in the mock).

## Round 3 triage

| # | Finding (personas) | Sev | Decision |
|---|---|---|---|
| 1 | The Flint panel has no way to download the router package (A, B, C, D) | M | **fix**: a home server on a GL router serves the package it runs: `GET /tailbond-glinet.ipk` rebuilds the installed .ipk from opkg's records, with pristine copies of the files that change after install and never anything under `/etc/tailbond/`. The panel's guide starts with **Download the router package** (same-origin through GL's nginx, GL session required). Covered by a unit test and the emulated install test |
| 2 | Fedora Devices tab: "Add a device" next to "Add a GL.iNet travel router" invites typing the router's name (C) | M | **fix**: the router guide comes first, "(recommended)", ends with "tap Approve. Nothing to type here."; the code form is "Add a device with a code (no Tailscale on it)" and says Tailscale devices just ask |
| 3 | Who can approve? (B) | M | **fix**: both approval cards say who can (Flint: anyone logged in to the GL panel, and anyone on the Tailscale account at `http://homebox:8077` unless it has a password; Fedora: anyone who can open the page), and the Flint panel can set that password |

Lows fixed: "leave the other options as they are" for LuCI's overwrite checkbox and "the router's internet keeps working" (A, B, C); example tailnet name taken from the server's name before the machine list loads (A, B).
Lows rejected: as in rounds 1–2 (graphs, SSID, Secondary, billing day, the Fedora phone view, port 80 in the mock).
