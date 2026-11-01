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

## Round 8 triage

| # | Finding (personas) | Sev | Decision |
|---|---|---|---|
| 1 | Fedora page contradicts itself on who can approve: the approval card said "tagged machine… anyone on your Tailscale network", the Devices card "alex@example.com" (F1, F2, GL B high; GL A, C, D medium) | **H** | **fix** (real bug): the approval card was drawn before settings loaded and took the "no owner" branch; it's redrawn when settings arrive, and says nothing until then |
| 2 | Only a shared password to let someone else approve; no named approvers; tagging forces the password (F1; F2) | M | **fix**: `approvers` (Tailscale logins) next to the owner, editable on the Devices tab; on a tagged server they are the only approvers; the card says a password replaces all of it and then everyone needs it |
| 3 | Approval banner leaves out internet/LAN/exit (F1) | M | **fix**: "use homebox's internet connection (by default all its traffic goes out from here) and your home network, and reach your Tailscale machines as homebox, under homebox's access rules" |
| 4 | "Set a password below" on the Connections tab (F2) | M | **fix**: names the place (Devices → Who can approve) |
| 5 | Header 1.1 down / 12 up vs the device's 12 down (GL D) | M | **fix**: the server's header, path cards and data total say "to devices / from devices" |
| 6 | Turning off: GL may now use the phone, Backup only still selected (GL C, D) | M | **fix**: the summary and the phone's card say GL may use the phone and to unplug it (or turn tethering off) to be sure |
| 7 | Standby "uses no data" vs GL's own checks (GL B) | M | **fix**: "tailbond sends nothing over it (GL itself may check the connection now and then, a few KB)" |
| 8 | CGNAT: dead end; "I was told it runs over Tailscale" (GL A, B, C) | M | **partial (platform limit, stated in the UI)**: the item now says bonding needs a direct connection and can't run through Tailscale's relays (they'd be slower than one link), and gives the IPv6 route (enter the public IPv6 address) before "ask your provider". A tailnet fallback isn't possible: DERP relays carry WireGuard over TCP/HTTPS and can't carry multipath QUIC paths at useful speed |
| 9 | "Allow 41641/udp in this machine's firewall" with no how (GL A, C; F2 zone placeholder) | M | **fix**: the Fedora package opens 41641/udp itself (WAN zone), so the fix is only the router forward; repeated fixes say "Same fix as above (one forward fixes both)" |
| 10 | Route step: needs the admin console, reads contradictory (GL C) | M | **fix**: "Done once, by whoever manages your Tailscale account. Until then devices can still reach your machines; only the other direction waits" |
| 11 | Can't check the UDP 443 forward from home (F2) | M | **fix**: the item says how to test now: a phone with tailbond on mobile data (Wi-Fi off) |
| 12 | Family phone must be on my tailnet / reach tcp 80, 8077 (F1, F2) | M | **fix**: phone card and guide §4 say so (invite or share; default policy allows it) |
| 13 | No screen confirms the family phone connected (F2) | M | **fix**: new render 14d (pixel-7a listed, connected, after Approve) |
| 14 | Guide §4 says `homebox` without the slash (F1) | M | **fix** |
| 15 | Hash comes from the same page; how to check; signing key (Android B) | M | **partial**: the page explains the trust root (Tailscale authenticates homebox, so nobody on the Wi-Fi or mobile network can change the file) and how to compare the hash on homebox; the signing certificate fingerprint isn't shown (debug-signed builds; release signing is future work) |
| 16 | Always-on conflict only in Settings (Android B) | M | **fix**: the handoff dialog says to turn Tailscale's Always-on off first, and mentions the exit-node default |
| 17 | Four prompts; Play Protect not drawn (Android C) | M | **partial (platform limit, stated)**: Play Protect's prompt is drawn too, and the page says it's Android's normal one-time check and the drawings aren't buttons. The prompts themselves can't be removed for an app outside a store |
| 18 | Tapping "Open" after install lands on onboarding; greyed button (Android C; A low) | M | **fix**: onboarding says "Installed tailbond from your home server's page in Chrome? Easier: go back to Chrome and tap step 2"; the greyed button explains itself |
| 19 | Switched back: "Tailscale is in charge" sounds finished (Android C) | M | **fix**: "Last step: turn Tailscale on", with Open Tailscale as the main button and what happens until then |
| 20 | "From another screen": "Add a device" isn't on the phone page (Android A) | M | **fix**: names the dashboard's "Add a device with a code" and warns the phone page's QR only opens the install page |

Lows fixed: Fedora 14 rendered whole-page; "1 day ago"; Play Protect drawing;
03-deeplink-confirm in the manifest; health footer no longer names a file;
"port mapping: UPnP" no longer in the all-good fixture.

## Round 9 triage

No highs on any surface. GL persona D: no medium.

| # | Finding (personas) | Sev | Decision |
|---|---|---|---|
| 1 | Turned off: the warning is buried, Tethering still shows "Backup only" (GL A) | M | **fix**: its own alert under the top card, "While tailbond is off, GL may use Tethering (your phone's data)", with what to do and GL's INTERNET page |
| 2 | CGNAT: the IPv6 route isn't actionable for a traveller/family member; a pasted IPv6 address fails on IPv4-only hotels (GL A, B, C) | M | **partial (platform limit, stated in the UI)**: the item leads with asking the provider ("often free or a small monthly fee"), IPv6 is marked *Advanced* with its limit ("only from networks that have IPv6 too"). Nothing on the home side can make a shared provider address reachable |
| 3 | Relay fix "forward UDP 41641" has no "where", a CLI check, no hand-off line (GL A, C; Fedora) | M | **fix**: "On your home router's admin page (Port forwarding, also called Virtual server or NAT), forward UDP port 41641 to this machine (192.168.1.2)"; the `tailscale ping` check moved to the guide; the Fedora card has "Show this card to whoever set up your internet" again. On the Flint the GL menu path stays |
| 4 | Fedora on a phone: "Add this phone" under Approve (GL C, F2; A low) | M | **fix**: on a phone with requests waiting, the page shows the requests and not "Add this phone" |
| 5 | Green "no port mapping on the router" after forwarding (F2; A, B, D low) | M | **fix**: an OK network check has no detail line |
| 6 | Can family still ask to connect with a password set? (F2) | M | **fix**: the card says phones and routers can still ask without it |
| 7 | The setup step's CGNAT hint compares the WAN address with itself (F2) | M | **fix**: "Setup health below checks for that and says what to do" |
| 8 | Firewall changes: temporary or saved; does uninstall close them (F1) | M | **fix**: the guide says both runtime and saved, nothing to do yourself, and `dnf remove` closes 443/41641 and the tailnet rules (spec already did) |
| 9 | The route is filed under "best speed" but is an exposure choice (F1; F2 low) | M | **fix**: "(optional)", what approving exposes, "If you want that: …"; not counted as a problem in the headline |
| 10 | Can't test the 443 forward before it's needed (F1) | M | **fix**: the item and the guide say how: a phone with tailbond, Wi-Fi off |
| 11 | Approved devices reach the home LAN, no way shown to limit; do own policy edits survive (F1) | M | **fix**: the guide gives the `tailbond-out` rich rule to block the LAN and says the hook leaves own rules alone |
| 12 | "Anyone using homebox itself" ambiguous; local users; tunnel access to :8077 (F1) | M | **fix**: "anyone logged in on homebox itself (http://127.0.0.1:8077)"; the guide says to set a password for untrusted local users, and that the tunnel address answers only the machine list |
| 13 | "From any of their devices" includes lent devices (F1) | M | **fix**: card and guide say so and what to do (borrower's account, or tag it) |
| 14 | Tagging the server makes everyone an approver; only grey text (F1) | M | **fix**: a setup-health warning "Anyone on your Tailscale network can approve devices" while a tagged server has no approvers or password |
| 15 | "Open home server page" greyed out while the box shows the name (Android A) | M | **fix** (real bug): the button uses the name of the failed ask, like the field does |
| 16 | ACL identity only stated after switching (Android B) | M | **fix**: the confirm dialog says the server's access rules apply, not the person's own |
| 17 | Android's screens cover the picture guide (Android C) | M | **fix**: "Read the pictures below first (or take a screenshot)" above the download button |
| 18 | Always-on: a family member can't know (Android C) | M | **partial (platform limit, stated)**: the dialog says the symptom ("switches itself off right after this"), where to turn it off or who to ask, and that Android doesn't let tailbond check that setting |

Lows fixed: the download button no longer wraps; 02-pairing-paste-error is
rendered tall enough to show the error; the 14d toast no longer covers the
cards; 03-deeplink-confirm (a duplicate of 02e) dropped from the packet;
"tailscale ping" gone from the dashboards.

## Round 10 triage

No highs. Mediums by surface: GL 6, Android 3, Fedora 11.

| # | Finding (personas) | Sev | Decision |
|---|---|---|---|
| 1 | Flint: the tunnel route is a numbered setup step, sends a GL user to the admin console (GL A; C low) | M | **fix**: an *Optional* item, not counted in "setup steps left": "Skip this unless you want your Tailscale machines … to open something on a device behind a travel router"; "Setup done" no longer claims the reverse direction without it |
| 2 | Screen 08 (on, can't reach home): cards say "While it's off…", and "Turn tailbond off" doesn't warn about phone data (GL D) | M | **fix** (real bug): cards say they're used once tailbond reaches home again; the off advice says GL may then use the phone's data |
| 3 | Waiting screen doesn't say who approves (GL C) | M | **fix**: "Not you? Ask whoever runs your home server to approve it" |
| 4 | Login Mode: "the top card" unclear; Tethering "In use" (GL C) | M | **fix**: no card reference; the other link's chip reads "Router's link only · almost no data" while devices sign in |
| 5 | Relay fix is beyond a family member (GL C) | M | **partial**: the fix line starts "For whoever manages this router:"; with the hand-off line, a family member's job is to pass it on. Changing router settings stays an admin task |
| 6 | Before Login Mode, "Your other connections keep working meanwhile" hides that devices bypass home (GL B) | M | **fix**: the first card already says all device traffic goes out the hotel Wi-Fi for up to 10 minutes and doesn't reach Tailscale machines |
| 7 | ACL identity explained too late, in jargon (Android A, B) | M | **fix**: the phone page's step 3, the first app screen, the confirm dialog and This device all say "They see this phone as homebox … so they allow it whatever they allow homebox" |
| 8 | The download button is above the pictures it says to read first (Android C; A low) | M | **fix**: the button moved below the five pictures |
| 9 | "Setup health: all set (1 optional)" right under the public-address setup step (F1, F2) | M | **fix**: "nothing to fix here … · finish the setup step above first" |
| 10 | Guide §3 says "Approve the route once" vs "(optional)" (F1) | M | **fix**: §3 calls it optional and says what it exposes |
| 11 | Guide: test the forward before telling tailbond the address, with no device yet (F1, F2) | M | **fix**: the test is step 4, "once a phone is paired" |
| 12 | A request signed in to the owner's account: whoever holds that router can approve (F1, F2) | M | **fix**: the request card says so and what to do (sign it in to the borrower's account first) |
| 13 | "as homebox, under its access rules" reads like a limit, is an escalation (F1) | M | **fix**: "they allow it whatever they allow homebox, which may be more than its owner's own account gets"; the guide's ACL example stays the way to narrow it |
| 14 | Tagged server: field says "Also allowed"; guide ordering leaves a window (F1) | M | **fix**: on a tagged server the field reads "Allowed to approve (… only they can)"; the guide says to add approvers *before* tagging, and that the grants only narrow once allow-all is removed |
| 15 | Approved slate7 shows only "offline" (F2) | M | **fix**: "not connected right now (approved devices connect by themselves when they're on and online)" |
| 16 | Invite vs share for family isn't explained (F2) | M | **fix**: the phone card says what each does, where it is in the admin console, and that shared users type the full name |

## Round 11 triage

No highs. Mediums: GL 8, Android 4 (persona A: none), Fedora 8.

| # | Finding (personas) | Sev | Decision |
|---|---|---|---|
| 1 | The "signed in to an approver's account" note reads like an alarm; no "if it's yours, approve"; missing on the Flint (GL A, B, C; F1, F2) | M | **fix**: "If this device stays with you, just approve. If you'll give it to someone: approve, then switch its Tailscale account (or tag it) before handing it over … Decline only refuses this request"; the same note on the Flint's request card |
| 2 | Relay fix "For whoever manages this router" while I am that person; worth it? (GL A, C; D low) | M | **fix**: "On this router: …"; the card heads warn-only items "Optional speed-ups (tailbond works fine without them; they only speed up reaching the machines named)" |
| 3 | Flint: the UDP 443 forward can't be checked from home (GL A) | M | **fix**: the Port row says how to test now (a phone on mobile data) |
| 4 | Route "optional" on 13 but "approved" under "set up for best speed" on 12/18c (GL D; B, C low) | M | **fix**: the OK item reads "Optional route … approved: your Tailscale machines can reach tailbond devices"; all-good says "Everything is set up" |
| 5 | Slate says the server must be on "the same Tailscale account" (GL B) | M | **fix**: "the same Tailscale network" |
| 6 | Ethernet sign-in: before clicking, it sounds like it only adds access (GL B) | M | **fix**: the first card says all device traffic goes out the cable for up to 10 minutes and can't reach Tailscale machines |
| 7 | Phone page invites the requesting phone to approve itself (Android B) | M | **fix**: "Is homebox yours? Then you approve it yourself … Nobody else on your Tailscale network can approve from there unless you named them" |
| 8 | Who can open connections to 10.77.0.2 (Android B) | M | **fix**: "only if homebox's tailbond route is approved in the Tailscale admin console (your access rules apply)" |
| 9 | "Only Tailscale machines": subnet routers' LANs, DNS (Android B) | M | **partial (stated limit)**: the option says names are still looked up through home (MagicDNS works) and that LANs behind other subnet routers aren't reachable through tailbond (the server doesn't accept other routers' routes; kept out of scope since the Android loop, round 1 #14) |
| 10 | Five Android prompts to remember (Android C) | M | **partial (platform limit, stated)**: a one-line "In short: Download anyway → Open → Settings → Allow from this source → back → Install (→ Scan app)" to screenshot; dashed rings and "they're drawings" make the pictures read as guidance. Android shows its own screens over the page; that can't be changed |
| 11 | Route advertised automatically, no opt-out; autoApprovers (F1) | M | **fix**: `"server": {"no_advertise_route": true}` (hook env `TB_ADVERTISE_ROUTE=0`); the route item and guide mention it and autoApprovers |
| 12 | "Share just this machine" sounds limited, but an approved phone reaches what homebox reaches (F1, F2) | M | **fix**: the card recommends inviting and says an approved phone reaches what the server reaches whichever way they joined; the guide says the same |
| 13 | "anyone logged in on homebox itself" is really any local program (F1) | M | **fix**: "any user or program on homebox itself … set a password if it runs things you don't trust" (card and guide). Loopback stays trusted, like most local admin pages |
| 14 | Tagged-server warning shown as "to improve (tailbond works meanwhile)" with router boilerplate (F1) | M | **fix**: its own headline "1 security setting to review" and footer "fixed on this page"; speed items read "optional speed-ups" |
