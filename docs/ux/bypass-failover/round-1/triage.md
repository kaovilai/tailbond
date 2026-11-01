# Round 1 triage

Four personas (A traveller, B homelab, C family, D Speedify user), fresh
reviewers, given only the PNGs, the [task script](../task-script.md) and the
manifests. Totals, high / medium / low:

| Persona | H | M | L |
|---|---|---|---|
| A traveller | 2 | 16 | 12 |
| B homelab | 2 | 17 | 12 |
| C family | 3 | 14 | 19 |
| D Speedify | 2 | 11 | 12 |

Duplicates across personas are merged below. Decision per point.

## High

| # | Finding (screens) | Who | Decision |
|---|---|---|---|
| H1 | Pasting a clashing server: the red "subnet_in_use" banner and a green "added as a backup" toast show together (add-server-paste-clash) | A B C D | **fix**: no success toast on error; plain message, no raw code |
| H2 | Website rules "Waiting" says "Install dnsmasq-full" with no way to do it (bypass-website-rules-wait) | A C D | **fix**: name GL's APPLICATIONS → Plug-ins page, give the command with a copy button, and say the site keeps using tailbond meanwhile. GL 4.10 on Flint 3 / Slate 7 ships dnsmasq-full, so this shows only on other firmware |
| H3 | Subnet clash: alarming, doesn't say who should act, where, or that homebox is unaffected (add-server-subnet-clash) | C | **fix**: calm wording ("homebox keeps working"), "ask whoever runs vps", and where: a VPS installed with install.sh `--subnet`, a GL home router's panel / `uci set tailbond.server.subnet` |
| H4 | Android says traffic "goes out from home" while running on the VPS (44, 45b) | B | **fix**: name the server in use |

## Medium

| # | Finding | Decision |
|---|---|---|
| M1 | Bypass card missing on every two-server router screen | **fix** (render fixtures: the card must show on all client screens) |
| M2 | bypass-add-wifi-calling shows the empty card, not the result | **fix** (re-render after the click) |
| M3 | Android website rule covers only name + `www.`; the router covers all subdomains | **partial**: Android can only exclude addresses it resolves itself, so it can't know `login.mybank.com` in advance. Say it in the dialog and row ("covers mybank.com and www.mybank.com; add login.mybank.com too if your bank uses it") |
| M4 | Android: address refresh can reconnect the tunnel (calls, SSH) every 10 min; DNS answered at home/VPS, not the hotel | **partial**: the rebuild swaps the VPN interface in place (no new QUIC connection) and is rate-limited to once a minute; say "a brief blip, at most once a minute, only when the site's addresses change". DNS-locality: accepted limit on Android, stated in docs |
| M5 | Work laptop rule tied to a MAC; private/rotating MAC breaks it silently; no sign it matches | **fix**: device rows show online / last seen (from the LAN list) and connections; the dialog says to turn off "private Wi-Fi address" for this network on the laptop |
| M6 | "Always use tailbond" says "through homebox", wrong during failover | **fix**: "through your server" / the server in use |
| M7 | No time for the switch or the return; no history | **fix**: "since 14:02" from `failover.since` on both platforms; "Back on homebox since 14:20". A full history is **rejected** for now (the log has it); low value vs space |
| M8 | Manual "you picked it" doesn't say whether it reverts or how to undo | **fix**: "Stays on vps until you pick homebox or vps stops working" + "Go back to homebox" |
| M9 | "Use now" offered on an unreachable server | **fix**: hidden for unreachable servers |
| M10 | Android: homebox green in Machines while Servers says Unreachable | **fix** (wording): the Machines dot is Tailscale's view; the server row says "tailbond can't reach it (Tailscale may still show it online)" |
| M11 | Android "Add a backup server" page reuses home-server wording; no tailnet shortcut | **fix**: wording for the backup; the ask-by-name field first |
| M12 | No statement on what traffic does while switching / when no server works | **fix**: one line ("While it switches, devices wait a few seconds; nothing goes out unprotected" when the kill switch is on; the Android error says whether the phone is online without tailbond) |
| M13 | Failback can drop a call; no "stay on vps" | **fix**: the countdown line offers "Stay on vps" (= Use now) |
| M14 | Banking preset: "type below" but the box is above; bank app not mentioned | **fix** wording on both; Android points to "Apps that skip tailbond" for the bank app |
| M15 | "Forget all servers" red next to "Add another server" | **fix**: moved to a less prominent spot |
| M16 | Approval link uses the short name `http://vps:8077` | **fix**: full MagicDNS name when known |
| M17 | How failover decides isn't explained | **fix**: one muted line under "When to switch" |
| M18 | "Whichever connection GL picks" vague | **fix** wording ("the router's main internet connection, as without tailbond") |
| M19 | Router tunnel address changes on failover; VPS route approval for tailnet machines | **partial**: one line in the servers card; details in docs/vps.md |
| M20 | CDN neighbours leak (other sites on the same addresses skip too) buried in a footnote | **partial**: kept as a footnote (accurate, rare; prominent wording would scare more than help) |
| M21 | Approval "from any device on your Tailscale": who may approve | **reject**: existing approval wording (owner / approvers) is the same as the main pairing flow already reviewed in earlier rounds |
| M22 | Android 12 fallback (whole browser skips) not warned | **fix**: say the whole app then skips tailbond |
| M23 | Phone error "(Connection timeout)" jargon, asks to turn on servers | **fix** wording |
