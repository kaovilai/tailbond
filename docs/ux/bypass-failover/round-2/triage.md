# Round 2 triage

Fresh reviewers again, same inputs. Totals, high / medium / low:

| Persona | H | M | L |
|---|---|---|---|
| A traveller | 1 | 13 | 13 |
| B homelab | 2 | 14 | 10 |
| C family | 1 | 15 | 21 |
| D Speedify | 0 | 11 | 15 |

Round-1 highs (paste clash toast, raw codes, clash wording, "goes out from
home") didn't come back. Merged findings and decisions:

## High

| # | Finding | Who | Decision |
|---|---|---|---|
| H1 | Phone website rules cover only the name + `www.`; the router covers every sub-name, so a bank login on `login.mybank.com` may still be blocked on the phone | A C (B, D as medium) | **partial**: Android can only exclude addresses it resolves itself (no DNS interception inside the VPN), so it can't learn unknown sub-names. The app now also resolves the usual sub-names (`www`, `login`, `secure`, `online`, `auth`, `account(s)`, `my`, `m`, `app`, `api`), shows which names a rule covers, and the dialog says to add others or put the bank's app in "Apps that skip tailbond". The difference from the router is stated on both |
| H2 | The SSH fallback for dnsmasq-full removes dnsmasq first; a failed install leaves the LAN without DHCP/DNS | B (A, C, D as medium) | **fix**: download first, then swap, with a reinstall fallback in the same line (`cd /tmp && opkg update && opkg download dnsmasq-full && opkg remove dnsmasq && opkg install ./dnsmasq-full*.ipk \|\| opkg install dnsmasq`); GL's Plug-ins page stays the first suggestion |
| H3 | Phone: during a switch and with both servers down, apps use the phone's own internet; Settings advises leaving "Block connections without VPN" off; the router says "nothing goes out unprotected" | B (A, D as medium) | **fix**: the Android settings recommend "Block connections without VPN" when there are backup servers (with an "Open Android VPN settings" button), the switching card says what happens and links there; the router's line is qualified ("with the kill switch on, the default") |

## Medium

| # | Finding | Decision |
|---|---|---|
| M1 | "Waiting" website rules: "nothing breaks" is wrong for a blocked bank; the row is green "Uses the internet directly" | **fix**: row shows "Waiting" in amber; notice says the site keeps going through tailbond (and may stay blocked) until then |
| M2 | "The router's main internet connection" doesn't say which one (Repeater? Tethering with a cap?) | **fix**: the agent now reports which WAN carries the default route (`wans[].default`); the target reads "Use the internet directly (now: Repeater)" |
| M3 | No switch history | **fix**: the agent keeps the last 20 switches (`failover.history`); GL shows "Recent switches", Android keeps its own list |
| M4 | After a manual pick, "Prefer homebox" still looks selected | **fix**: the policy shows "paused: you picked vps" with "Go back to automatic" |
| M5 | "Switch back to Tailscale" (turns tailbond off) sits under the failover note on the phone | **fix**: shown less prominently, renamed "Hand the VPN back to Tailscale", not directly under failover notes |
| M6 | "Stay on vps" is the only filled button in the countdown card | **fix**: text button; the countdown is the main message |
| M7 | Router countdown doesn't warn about the pause on switch-back | **fix**: same line as the phone |
| M8 | Router: a failover header with no "you're online, nothing to do" | **fix** |
| M9 | Router: "Remove" next to homebox's "Can't reach" | **fix**: Remove moves behind the row's "More" |
| M10 | Device rule / MAC rotation: unclear what happens if it stops matching; "waits when tailbond is down" means no internet | **fix** wording on row and dialog: "If its address changes, the laptop goes through tailbond like everything else, but website rules then apply to it"; "No internet for it while tailbond is off or down" |
| M11 | Clash fix needs a terminal; phone and router disagree ("asking over Tailscale does this by itself") | **partial**: fixing it needs admin on the other server (by design: the router can't reconfigure someone else's server). Both now say the same: a new server with no devices yet moves by itself when asked over Tailscale; this one already has devices, so whoever runs it changes its range. A copyable message to send them |
| M12 | Phone: no tailnet suggestions when adding a server (router finds vps) | **fix**: suggestions from the tailnet machines list the app already has (through the server in use) |
| M13 | Phone: "someone approves this phone on that server": who/where | **fix** wording |
| M14 | "Approve each server's route in Tailscale" reads like an unfinished job | **fix**: optional, and only for reaching devices behind the router from the tailnet |
| M15 | Router doesn't say what changes on vps (public IP, connections restart, tailnet rules as vps) | **fix**: one line under the failover header |
| M16 | What doesn't trigger a switch (home internet down while homebox answers) | **fix**: added to "How it decides" |
| M17 | Router's machine list shows home machines online while home is dark | **fix**: caveat line while on a backup |
| M18 | Router doesn't explain website matching (by addresses; shared hosting) like the phone does | **fix**: one line |
| M19 | Website rule says "N connections" but can't show which devices are missed (secure DNS) | **reject**: the router can't see a device's own DNS-over-HTTPS; the footnote says so |
| M20 | Phone and router rules when the phone is on the Slate's Wi-Fi | **fix**: one line in docs/android.md and the phone dialog: the phone's own rules apply first; the router's rules then see the phone as any device |
| M21 | No notification screenshot for a switch | **partial**: Android already posts the switch in its notification; added to the render set. The router can't notify |
| M22 | DNS and IPv6 not stated | **partial**: docs (glinet.md: bypassed websites resolve via the WAN's DNS; IPv6 follows the same rules); not on screen |
| M23 | Suggested free range (10.78.0.0/24) not checked against the other server's routes | **reject**: only that server can check its own routes; install.sh does |
| M24 | Android 12 "separate browser" assumes knowledge | **fix** wording (example browsers, one line) |
| M25 | Fixture times out of order (14:12 vs 14:11) | **fix** (fixtures) |
