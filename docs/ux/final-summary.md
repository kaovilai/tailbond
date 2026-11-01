# Simulated user testing: the final loop (rounds 7+)

Goal: zero high and zero medium findings from every persona on every
surface, confirmed by a fresh round after the last change (cap: 6 more
rounds after round 6, i.e. round 12). Same method as
[the Android loop](summary.md) and [the GL loop](glinet-summary.md): fresh
reviewer agents each round, given only the PNGs, a task script and a
manifest; every point triaged **fix / partial / reject (reason)**; rejected
points aren't re-argued.

Surfaces and personas:

| Surface | Personas | Task script / manifest |
|---|---|---|
| GL.iNet panel (Slate 7 + Flint 3, plus the Fedora page for tasks 7/10) | A traveller, B homelab, C family, D Speedify user | [glinet/task-script.md](glinet/task-script.md), [glinet/screens.md](glinet/screens.md) |
| Android, whole journey (phone page in the browser → app) | A daily Tailscale user, B homelab, C family | [final/android-task-script.md](final/android-task-script.md), [final/android-screens.md](final/android-screens.md) |
| Fedora web dashboard + install guide | F1 homelab admin, F2 non-networking home admin | [final/fedora-task-script.md](final/fedora-task-script.md), [final/fedora-screens.md](final/fedora-screens.md) |

Renders: `out/ux/final/round-N/{web,glinet,android}/`, reviewer packets
`out/ux/final/round-N/packet-*` (not committed; `out/` is ignored).
Regenerate a round with `scripts/ux/final-renders.sh round-N`. Raw
reports: [final/round-N/](final/).

## What was built for the Android "getting the app" high

The previous loops ended with one high on Android (personas A and C): no way
to get the APK onto a phone without a store listing or a link from the
admin. Options and what was picked:

| Option | Decision |
|---|---|
| A **phone page on the server's Tailscale IP, port 80**: typing `homebox/` in the phone's browser (Tailscale on) shows *Get tailbond*: the APK, what each Android install prompt looks like, and *Open tailbond and ask homebox* | **built**. It serves only the page, the APK and the icon (no API, no dashboard) whatever the password says; also at `:8077/get` when port 80 is taken. On a GL Flint (nginx owns port 80) the agent binds 8078 and the firewall redirects tailnet tcp/80 to it |
| `tailscale serve` for `https://homebox.<tailnet>.ts.net` | **documented, not default** ([fedora-server.md](../fedora-server.md)): fine when pointed at the phone page only; pointing it at :8077 would hide every requester's identity (all requests come from tailscaled), so pair requests would be refused. Not needed: Tailscale already encrypts the http page |
| Sideload guidance at the moment Android warns | **built** into the page: drawn versions of Chrome's "File might be harmful", *Install unknown apps → Allow from this source*, *Install*, Play Protect's scan; where the file went if the message disappeared; SHA-256 of the APK |
| QR "Install on your phone" | **built**: Fedora dashboard Devices tab and GL panel, *Add an Android phone* (QR of `http://homebox.<tailnet>.ts.net/` + the link) |
| APK always available | **built**: the RPM and the IPK carry `/usr/share/tailbond/tailbond.apk` when the build has one (`TAILBOND_APK`, else `out/android/tailbond-debug.apk`) |
| App deep link | **built**: `tailbond://ask?server=<name>` (Android intent link with a fallback to the page's install step); the app skips its intro and asks the server right away |
| Obtainium / F-Droid-style update channel | **future option only** ([android.md](../android.md#updates-future-option)); nothing published |

The remaining platform limit, stated on the page and here: Android always
asks before installing an app from outside a store (and Play Protect may
offer a scan). tailbond can explain those prompts but can't remove them.

## Findings per round

ROUNDS_TABLE

## Round 7 triage

First review of the phone page, the QR cards, the whole Android journey,
and the Fedora dashboard/guide; GL round-6 fixes re-reviewed.

GL round-6 mediums re-checked: #2 (Fedora page password) and the Login Mode
top card ("Router connected to homebox · devices signing in") weren't raised
again as such; the CGNAT screen (#1) and the Login Mode phone-data question
(#3) came back in a new form (below).

| # | Finding (personas) | Sev | Decision |
|---|---|---|---|
| 1 | Sharees can approve their own devices; every bonded device gets homebox's tailnet permissions; nothing warns (Fedora F1) | **H** | **fix**: without a password the dashboard API now answers tailnet devices only when they belong to the server's Tailscale owner (`whois` vs the owner in `tailscale status`); others get a short "this page is only for …" note pointing to the phone page, and can still ask to connect. Approval cards (Fedora and Flint) say an approved device reaches the tailnet as homebox, under its access rules; the guide has an ACL example (tag + grants) and says a tagged server needs a password |
| 2 | CGNAT: health card says "tailbond works meanwhile, just slower" while the step says travel routers can't connect from away (GL A, B, C, D) | M | **fix**: a `bad` item heads the health card as "N thing(s) to fix before travel routers and phones can connect from away", red chip "1 to fix"; "works meanwhile, just slower" only when nothing blocks (GL panel and Fedora page) |
| 3 | GL Login Mode: what uses the phone's data while devices bypass the tunnel (GL A; B, D low) | M | **fix**: the card says devices' traffic doesn't use the phone's data and the router's own link carries almost nothing meanwhile; fixture shows that trickle instead of 21 Mbps; the tailnet list says signing-in devices don't reach machines until done |
| 4 | GL Login Mode: where does the hotel's page appear (GL C) | M | **fix**: "open any website on your laptop or phone: the hotel's sign-in page appears there" on both GL sign-in cards |
| 5 | Turned off: which connection devices use, does Backup only still apply (GL D; C low) | M | **fix**: "over whichever connection GL's own settings pick (INTERNET page). Backup only and data limits here apply only while tailbond is on", also on each WAN card |
| 6 | Flint relay fix "NETWORK → UPnP" is wrong for the edge router and opens the LAN (GL B) | M | **fix**: "NETWORK → Firewall → Open Ports on Router, add UDP 41641", plus the forward on a box in front |
| 7 | Who can approve on the Flint leaves out sharees; nothing says what an approved device reaches (GL B) | M | **fix**: same owner rule and wording as #1 on the Flint panel |
| 8 | Pairing screen while asking: "Go back to the page and tap step 2" reads like a mistake; greyed button (Android A; B, C low) | M | **fix**: while asking or waiting, the other ways are hidden ("show up here again if you tap Cancel") |
| 9 | No APK checksum / signing info; `http://` page (Android B; F1 low) | M | **fix**: the page shows the APK's SHA-256, says Tailscale encrypts the page, and that updates must carry the same signature |
| 10 | Approval matched only by the phone's name (Android B) | M | **fix**: the pair-request reply carries the requester's Tailscale identity; the waiting screen says "There it shows as pixel-8 · sam@example.com · 100.101.8.8" |
| 11 | Bond doesn't go over Tailscale; full tunnel by default, only in small print (Android B) | M | **fix**: the page's step 3 says so (like an exit node; Settings changes it). Full tunnel stays the default (rejected in the Android loop, round 2 #19) |
| 12 | What identity the phone has for ACLs (Android B) | M | **partial**: This device says "Not a Tailscale address: to your Tailscale machines this phone's traffic comes from homebox, under its access rules". A per-phone tailnet identity would need a Tailscale node per phone (out of scope since the Android loop, round 1 #6) |
| 13 | Where the downloaded file is when Chrome's message is gone (Android C) | M | **fix**: "Chrome's menu ⋮ → Downloads, or the Files app → Downloads" |
| 14 | Step 2 before installing jumps back to step 1 with only an outline (Android C; A, B low) | M | **fix**: a visible note "tailbond isn't on this phone yet, so step 2 brought you back here…" |
| 15 | "Keep this screen open": is leaving the app safe? Flint jargon (Android C) | M | **fix**: "You can switch to another app to message them; tailbond keeps waiting for 10 minutes and carries on by itself once approved"; the Flint panel mention is gone |
| 16 | Public address / UDP 443 isn't a setup step; banner at the bottom, no how-to (F1, F2) | M | **fix**: "Setup step: tell devices where home is" at the top of Devices with the port forward, DHCP reservation, finding the address and a CGNAT hint; a line on every tab |
| 17 | Guide says "Clients tab" (F1, F2) | M | **fix** |
| 18 | Typing `homebox` in Chrome on Android searches instead (F1; F2 low) | M | **fix**: everywhere says `homebox/` "with the slash at the end, so it opens the page instead of searching" |
| 19 | Phone page step 3 names no button; no evidence the phone connected (F2) | M | **fix**: step 3 names *Add phone*, the switch and Android's OK; a Fedora render of a phone's request (14c) |
| 20 | 41641 isn't in the guide; firewalld must allow it too; UPnP safe? (F1, F2) | M | **fix**: one fix text for relay and hard NAT ("forward UDP 41641 … and allow 41641/udp in this machine's firewall; UPnP also works but lets any device open ports"); a health table in the guide with the `firewall-cmd` line |
| 21 | "Through a relay: pixel-8" while pixel-8 is the bonded phone; which traffic it slows (F1, F2) | M | **fix**: fixture inconsistency (a bonded phone's Tailscale is off; now apple-tv is relayed and pixel-8 shows last seen); the item says only traffic to that machine is slower and bonding doesn't use Tailscale; "hard NAT" explained |
| 22 | "Nothing here changes homebox's settings" but the page links the unprotected dashboard (F1, F2) | M | **fix**: #1 (the dashboard answers only the owner); the footer link is gone, only "You run homebox yourself?" keeps it |
| 23 | Assumes the WAN NIC is in the default zone (F1) | M | **fix**: the hook opens UDP 443 in the zone of the default-route interface; the guide lists what changes in firewalld and how to check it |
| 24 | What bonded devices reach on the LAN/host (F1) | M | **fix**: the guide says it (internet, home LAN, tailnet, this host's tunnel-address services) |
| 25 | Does the password protect `/api/clients`; curl examples (F1) | M | **fix**: the guide says it protects the whole API and shows the login for curl |
| 26 | No ACL guidance; would limiting tcp 8077 break asking (F1) | M | **fix**: the ACL example keeps tcp 80/8077 open to members |

Lows fixed: "null" line in the all-good health card; speed tips no longer
in an error-red banner; "1 days ago"; swipe-back wording; the GL phone card
text. Lows left (no change): the copy icon copies the IP (Android loop,
round 1 #21), several data-saving controls (round 3 #15), short MagicDNS
names on the travel router, per-connection graphs, billing-day resets,
checksum of the router package, the LuCI install path (GL has no upload in
Plug-ins).
