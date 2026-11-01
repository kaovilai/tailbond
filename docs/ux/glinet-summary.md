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

Renders: `out/ux/glinet/round-N/` and `out/ux/glinet/final/` (not
committed; `out/` is ignored).

## Findings per round

| Round | A (H/M/L) | B (H/M/L) | C (H/M/L) | D (H/M/L) | Total H/M/L |
|---|---|---|---|---|---|
| 1 | 0 / 3 / 12 | 0 / 3 / 22 | 0 / 6 / 11 | 0 / 3 / 13 | 0 / 15 / 58 |
| 2 | 0 / 3 / 11 | 0 / 1 / 18 | 0 / 3 / 10 | 0 / 0 / 17 | 0 / 7 / 56 |
| 3 | 0 / 1 / 9 | 0 / 2 / 16 | 0 / 2 / 9 | 0 / 1 / 13 | 0 / 6 / 47 |
| 4 | 0 / 3 / 11 | 0 / 5 / 13 | 1 / 3 / 11 | 0 / 3 / 15 | 1 / 14 / 50 |
| 5 | 0 / 1 / 16 | 0 / 2 / 12 | 0 / 0 / 23 | 0 / 1 / 15 | 0 / 4 / 66 |
| 6 | 0 / 1 / 12 | 0 / 3 / 13 | 0 / 0 / 16 | 0 / 1 / 14 | 0 / 5 / 55 |

**Outcome: the exit condition (no medium or high from any persona) was not
met within the 6-round cap.** Highs: 0 in rounds 1–3, 1 in round 4 (after
captive portals, setup health and the VPN-menu move were added), 0 in
rounds 5–6. Persona C (family member) had no medium in rounds 5 and 6. The
round-6 mediums were fixed afterwards without another review (below);
final renders: `out/ux/glinet/final/`.

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

## Scope added after round 3

Captive portals (own probe, sign-in mode, GL repeater Login Mode as the
source of truth for wwan), setup health (direct vs relayed Tailscale,
netcheck, carrier-grade NAT, route, UDP 443), the panel moved under GL's
VPN section (like GL's VPN client and Speedify), Android Wi-Fi sign-in.
Task script tasks 8–10 and persona D comparing directly against Speedify
on GL were added from round 4 on; round 4 is the first review of these.

## Round 4 triage

| # | Finding (personas) | Sev | Decision |
|---|---|---|---|
| 1 | Fresh Flint: "Everything is set up for best speed" under "2 setup steps left" (C high; A, B, D) | H/M | **fix**: the health card stays hidden until the setup steps are done; with only unchecked items it says "Nothing to fix. N checks need a device connecting from away (?)"; the badge counts problems only |
| 2 | Carrier-grade NAT: green "Setup done" above "travel routers can't reach UDP 443" (A, B, C, D) | M | **fix**: a blocking health problem becomes a numbered setup step (red), and "Setup done" only shows without one |
| 3 | Setup health missing on the Fedora Devices tab (A, B, C, D) | M | **fix** (real bug): the web UI's API helper treats `"ok": false` as an error; the health view's flag is now `all_good` |
| 4 | Relay fixes in jargon (UPnP/NAT-PMP, double NAT, tailscale ping); which box to change (A, C) | M | **fix**: "Usually fixed by forwarding UDP port 41641 to this machine, or by turning on UPnP on the router in front of it"; technical detail stays in the second line; a hand-off line "Show this card to whoever set up your internet" |
| 5 | Relayed machines in health don't match the machine list (B; C, D low) | M | **fix** (fixture inconsistency): fixtures share one peer list; the machine list itself marks "via relay (slower)" |
| 6 | GL Login Mode: what bypasses the tunnel, for how long, how to stop (B) | M | **fix**: "devices here use the hotel Wi-Fi directly, not through your home server … ends by itself once you've signed in (or click Quit Login Mode on GL's Repeater page)"; same words on both sign-in cards |

Lows fixed: "Until 04:06" → "For up to 9 more minutes (until 04:06)"; "Stop" → "Stop sign-in mode"; an all-good health render (18c).
Lows rejected: LuCI install (GL has no upload in Plug-ins), VPN vs APPLICATIONS placement (A: GL keeps VPN apps, including Speedify, under VPN), graphs/billing day (D, earlier rounds).

## Round 5 triage

| # | Finding (personas) | Sev | Decision |
|---|---|---|---|
| 1 | Flint health fix "forward UDP 41641 to this machine, or turn on UPnP on the router in front of it": on a Flint the router *is* this machine; same fix twice (A) | M | **fix**: on the GL panel the fix names GL's page ("On this router: NETWORK → UPnP, turn it on. If another router sits in front…"); a repeated fix shows once ("Same fix as above"); the Port row no longer says "opened" without saying carrier-grade NAT blocks it |
| 2 | Fedora: setup health only on the Devices tab (D) | M | **fix**: a banner on every tab when something needs improving ("Setup health: 2 things to improve … Show"), plus the hand-off line on the card |
| 3 | Direct/relay labels: which hop? Does LAN→NAS use the bond or the Slate's own Tailscale? (B) | M | **fix**: the Slate's list says LAN devices reach machines through the bond to homebox, and that direct/relay is how homebox reaches each machine; fixtures now agree between the Slate and the Flint |
| 4 | Sign-in mode: all device traffic or only the sign-in page? Header still says "Connected … using" (B) | M | **fix**: both sign-in cards and the status line say all traffic from devices goes out that connection directly (not only the sign-in page, not through homebox) until signed in, and that the router's own tunnel stays up on the other connections |

Lows fixed: GL Login Mode banner title ("Signing in to Repeater through GL's Login Mode", no second "click Login Mode") (A); explain why cable sign-in differs from Wi-Fi (GL covers the repeater only) (A, D); health counts match the list (B, C); "on this router's clock" (B); turned-off fixture keeps Backup only (D).
Lows rejected: on-screen push notification for portals (D: the GL admin has no notification channel for plug-ins; the summary line and card are where GL users look), graphs, Secondary, billing day (earlier rounds), checksum (B), port 80 in the Fedora invite (render mock).

## Round 6 (last round) triage

| # | Finding (personas) | Sev | Decision |
|---|---|---|---|
| 1 | Carrier-grade NAT screen: "travel routers can't reach UDP 443" next to pixel-8 connected; health card gone (A, B, D) | M | **fix, after round 6 (not re-reviewed)**: the step says devices on the home Wi-Fi still connect, from away they can't; the fixture shows pixel-8 on the home LAN and slate7 not connected; the health card stays visible when the only steps are health problems |
| 2 | Fedora: anyone the machine is shared with can approve; a password means editing JSON (B) | M | **fix, after round 6**: Devices tab → **Who can approve and add devices** sets or removes the page password; the approval card points there |
| 3 | GL Login Mode screen: "Connected to homebox" and Tethering in use while traffic goes out the hotel Wi-Fi (B; A, C, D low) | M | **fix, after round 6**: the card says the router itself stays connected to homebox over the other connections (that's the top card) while devices' traffic goes out the hotel Wi-Fi; the top card reads "Router connected to homebox · devices signing in" |

Lows not addressed: LuCI install (GL has no upload elsewhere), graphs/Secondary/billing day (D), two sign-in flows (explained on the card: GL covers the repeater only), checksum/signature (B), short MagicDNS names (A), port 80 in the Fedora mock.

## Android personas, re-run once on the new pairing flow

Renders `out/ux/after-glinet/` (Roborazzi + the web page), task script and
manifest as in [the Android loop](summary.md); raw reports in
`after-glinet/`.

| Persona | H / M / L | Task 1 (pair from the phone alone) |
|---|---|---|
| A (daily Tailscale user) | 1 / 1 / 9 | Pairing: yes (typed `homebox` from Tailscale's machine list → Ask to connect). High: no way to get the APK before knowing the server page's address |
| B (homelab) | 0 / 1 / 6 | Pairing: yes, two ways. Medium: same APK question |
| C (family) | 1 / 1 / 8 | Pairing: yes. High: same APK question; medium: "Ask again" with an empty field after a failure (render state; the field now keeps the asked name) |

The discovery medium from [the Android loop](summary.md) (round 4 #1, "can't
find the server's page or name without asking someone") is closed for
pairing: the phone asks the server by its Tailscale name and someone
approves it. What remains is getting the app onto the phone at all without
a store listing or a link from the admin: the same distribution limit as
round 3 #5 there (**partial, stands**). After the re-run the waiting screen
says the request shows at the top of the server's page on a phone too, and
a failed ask keeps the name in the field.

## Re-review of the round-6 fixes

Round 7 of [the final loop](final-summary.md) re-reviewed these screens
with fresh reviewers, together with the Android journey and the Fedora
dashboard.
