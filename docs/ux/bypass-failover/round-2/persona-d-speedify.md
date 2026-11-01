# Round 2, persona D: Speedify user

Persona: has used Speedify for years (per-app and per-domain "Bypass" lists, a
server picker with automatic "nearest server") and Peplink/SpeedFusion at
work. Expects server choice and bypass lists to be quick to set up, and
expects to see which server is in use at a glance.

Inputs: `task-script.md`, `glinet-screens.md`, `android-screens.md`, and all
PNGs in `out/ux/bypass-failover/glinet/` and `out/ux/bypass-failover/android/`.

## Task 1: make `mybank.com` skip the tunnel (router, then phone)

**Done?** Yes on the router and yes on the phone, with caveats: on the phone I
had to guess which other bank hostnames to add, and on the router the rule
does nothing until dnsmasq-full is installed.
Screens: `bypass-empty-desktop`, `bypass-add-website-presets-desktop`,
`bypass-rules-desktop`/`-phone`, `bypass-website-rules-wait-desktop`,
`46c-websites-none-yet`, `47-add-website-dialog`, `47b-add-website-preset-banking`,
`46-websites-skip-tailbond` (+ dark), `46b-websites-needs-android-13`,
`48-settings-skip-rows-android-12`.

Finding the feature was fast. The router has a "Bypass tailbond" card with a
"Banking" quick pick. The phone has Settings → "Websites that skip tailbond".
This is close to what Speedify does.

Findings:

- **medium**, `bypass-website-rules-wait-desktop`: The notice says "these
  websites keep going through tailbond as before; nothing breaks". For a bank
  that blocks VPN addresses, that does break things: the bank keeps refusing
  me, even though the rule shows a green "Uses the internet directly" next to
  a small "Waiting" badge. The notice offers no workaround for the meantime.
  *Suggestion:* say plainly "mybank.com still goes through tailbond, so a bank
  that blocks VPNs will still block you", turn the action label on waiting
  rows grey or amber instead of green, and offer a workaround that works now,
  such as "skip tailbond for this device instead".
- **medium**, `bypass-website-rules-wait-desktop`: The SSH fallback runs
  `opkg remove dnsmasq && opkg install ...`. If the install fails after the
  remove, the travel router has no DHCP/DNS, and I'm locked out of it at a
  hotel with no cable. Nothing on the screen warns about this.
  *Suggestion:* install over the existing package, or use a command that
  restores dnsmasq on failure. At least add a warning such as "if this fails
  you lose Wi-Fi on the router; have a cable ready".
- **medium**, `bypass-add-website-presets-desktop` (also `bypass-add-wifi-calling-desktop`):
  "Use the internet directly" is explained as "the router's main internet
  connection". This router uses Repeater (the hotel) and Tethering (my phone)
  together, both set to "Always", so I can't tell which one is "main". The task
  needs the hotel's line specifically, and I'd be guessing.
  *Suggestion:* name the link right in the hint ("uses Repeater (hotel
  Wi-Fi) right now"), and say what happens when that link drops.
- **medium**, `47-add-website-dialog` / `46-websites-skip-tailbond`: On the phone a
  website covers only "the name and its www. name"; login.mybank.com has to be
  added separately. On the router, the same rule "includes all its pages and
  parts, like www.mybank.com and login.mybank.com". The same rule behaves
  differently on each platform. Banks often sign in on an auth or SSO
  subdomain, and from these screens I can't find out which one.
  *Suggestion:* cover subdomains on Android as well (Speedify's domain bypass
  does). If that isn't possible, show a hint after a bank rule is added that
  lists the names actually seen recently, e.g. "also seen: auth.mybank.com,
  add?".
- **low**, `47d-add-website-preset-wifi-calling`: The "Wi-Fi calling" chip in
  "Add a website" adds nothing; it only explains that nothing is needed.
  It's a dead end inside an add dialog. *Suggestion:* remove the chip from
  the phone dialog, or move the explanation to the page footer.
- **low**, `47c-add-website-preset-streaming` vs `bypass-add-streaming-preset-desktop`:
  The phone's "Streaming" preset adds 2 domains (Netflix only). The router's
  adds 5 (Netflix, Disney+, Prime, Max). *Suggestion:* use the same list on
  both, or label the phone one "Netflix".
- **low**, `46-websites-skip-tailbond`: The router rows show "37 connections",
  which proves a rule is matching. The phone rows show no such count, so I
  can't check that the bank rule works. *Suggestion:* show a "last matched"
  time or a count per row.

## Task 2: make the work laptop always use tailbond

**Done?** Yes (router).
Screens: `bypass-add-device-picker-desktop`/`-phone`, `bypass-rules-desktop`,
`bypass-empty-desktop` (the "Work laptop" quick pick).

The device picker defaults to "Always use tailbond". The help text says this
overrides websites and ports set to skip, and that the device waits rather
than going around tailbond when it is down. That answers what I needed.

Findings:

- **medium**, `bypass-rules-desktop` (all client screens): The same page
  shows two different "work-laptop" entries with contradictory status. The
  bypass rule says "connected now". The "Tailnet machines" list just below
  says "work-laptop · Last seen 1 day ago". A user will think the rule isn't
  working, or that the laptop isn't really on tailbond.
  *Suggestion:* label the source in each section ("on this router's Wi-Fi"
  vs "in Tailscale"), or hide the Tailnet row's status when a bypass rule
  covers the same name.
- **low**, `bypass-add-device-picker-phone`: The Cancel/Add buttons are cut off
  at the bottom of the 390x844 dialog. *Suggestion:* make the button row
  sticky, or make the device list shorter on phones.
- **low**, `bypass-rules-desktop`: A "must never leak" rule can be turned off
  with a single toggle and no confirmation, the same as a convenience rule.
  *Suggestion:* confirm before turning off an "Always uses tailbond" rule,
  or show a warning while it is off.

## Task 3: add the VPS as a backup (router and phone), handle a clash

**Done?** Router yes. Phone partly: I had to type the server's exact
Tailscale name, because the phone doesn't list servers it finds. I understood
the clash on the router, but fixing it needs a shell on the VPS, and the phone
explains the clash differently.
Screens: `servers-single-home-desktop`, `add-server-open-desktop`,
`add-server-waiting-approval-desktop`, `add-server-subnet-clash-desktop`,
`add-server-paste-clash-desktop`, `servers-home-active-vps-standby-desktop`,
`44b-settings-one-server-add-backup`, `45-add-another-server`,
`45b-add-another-server-confirm`, `45c-add-another-server-same-addresses`,
`44-settings-servers` (+ dark).

The router's "Found on your tailnet: vps" with an "Ask to add" button is
the quick path I expect from Speedify's server list.

Findings:

- **medium**, `add-server-subnet-clash-desktop` / `add-server-paste-clash-desktop`:
  The only way to fix the clash is a command run on the VPS
  (`install.sh --subnet`, `uci set ...`, or editing JSON). The router
  already had a channel to vps (the approval round-trip), but it offers no
  "fix it for me" button. For a Speedify/Peplink user, server setup that
  needs SSH is a hard stop.
  *Suggestion:* include "use 10.78.0.0/24" in the add request, so that the
  approval page on vps can switch the range with one click. Alternatively,
  have vps choose a free range automatically when it approves.
- **medium**, `45c-add-another-server-same-addresses`: The phone's clash
  dialog gives only the "edit /etc/tailbond/tailbond.json" route; the router
  gives three, including the installer one-liner. The dialog also says
  "Asking over Tailscale does this by itself while vps has no other
  devices". That contradicts the router screen, where asking over Tailscale
  still produced the clash, and nothing tells me whether vps "has other
  devices". *Suggestion:* use the router's text on both platforms. If the
  automatic case exists, say exactly when it applies ("vps has no devices yet:
  tap Ask again and it fixes itself").
- **medium**, `45-add-another-server`: The phone gives four ways to add a server
  (type name, open page, scan code, paste link) but never lists the servers it
  can see. The router shows "Found on your tailnet: vps" with a one-click
  "Ask to add". On the phone I have to know the exact Tailscale name.
  *Suggestion:* put a "Found on your Tailscale" list at the top of the phone
  page, as the router does, and fold the other three methods under "Other
  ways".
- **low**, `servers-home-active-vps-standby-desktop`: The server rows show
  "answered 9 s ago" but no latency or speed for each server. Speedify and
  Peplink show ping per server, which is how I'd judge whether the VPS is
  usable before I need it. Also, vps is shown by bare IP (203.0.113.9) while
  homebox gets a hostname. *Suggestion:* show the probe RTT ("38 ms") on each
  row.
- **low**, `servers-single-home-desktop` → `add-server-open-desktop`, `44-settings-servers`:
  The naming is inconsistent: the button says "Add a backup server" and the
  section it opens says "Add another server". On the phone the badge reads
  "Backup" on the home screen and "Backup 1" in Settings. *Suggestion:* pick
  one wording for each.

## Task 4: notice the switch to vps, and the return to homebox

**Done?** Yes on both. The router headline "Using vps since 14:12 (homebox is
unreachable)" and the phone page title "vps / via backup server" answer which
server is in use and why. The countdown and "Back on homebox since ..." answer
when it went back.
Screens: `failover-using-vps-home-unreachable-desktop`/`-phone`,
`failback-countdown-desktop`, `failover-you-picked-vps-desktop`,
`back-on-homebox-since-desktop`, `40-...`, `41-...`, `41b-...`, `41c-...`,
`42-...`, `43-...`, `43b-...`, `44c-...`, `49-...`.

Findings:

- **medium**, `failover-you-picked-vps-desktop`: The headline says "Using vps
  since 14:04 (you picked it). Stays on vps until you pick homebox". Under
  "When to switch", the selected option is still "Prefer homebox, use vps if
  homebox is down". The page contradicts itself, and I can't tell which one
  wins. *Suggestion:* while a manual pick is active, show it in that section
  ("Manual: vps, ignoring the choice below until you go back"), or select a
  temporary state there.
- **medium**, `41-home-using-vps-homebox-unreachable`, `41c-...`, `42-...`:
  Just below the "Using vps ... goes back to homebox" card, the largest
  button on screen says "Switch back to Tailscale". If I want to go back
  to homebox, "Switch back" is exactly what I'd tap, and it turns tailbond
  off. *Suggestion:* rename it "Turn off tailbond and use Tailscale", and
  move it out of the hero area (it already exists in Settings) or make it a
  text link.
- **medium**, `43b-home-switching-to-vps` with `44-settings-servers`: During
  each failover the phone says apps use the phone's own connection unless
  "Block connections without VPN" is on. Settings tells me to leave that off
  so I can switch back to Tailscale easily. With that advice, every switch
  leaks traffic to the hotel network, and the switch screen gives no way to
  change it. *Suggestion:* explain the trade-off in one place ("Off = easy
  switch back, but a few seconds unprotected during each server switch"),
  and add an "Open VPN settings" link on the switching card.
- **low**, `failover-using-vps-home-unreachable-desktop`/`-phone`, `41-...`:
  Apart from the text, the failover state looks the same as normal. The
  router keeps the same green icon and the phone the same blue "Connected"
  card. Peplink and Speedify use a distinct colour or badge for "on backup".
  *Suggestion:* use an amber icon or a "Backup" chip in the status card
  while not on the first-choice server.
- **low**, `failback-countdown-desktop`: The headline changes to "Using vps
  (homebox is answering again)" and drops the "since 14:12" shown on the
  previous screen. *Suggestion:* keep the "since" time.
- **low**, `42-home-back-to-homebox-in-1-20`: "Stay on vps" is the filled
  primary button, so it looks like the action I'm expected to take, while the
  default (go back) has no button. The screen also doesn't say how long
  "stay" lasts. *Suggestion:* make it a secondary or text button and add
  "until you pick homebox".
- **low**, `44c-settings-servers-homebox-answering-again`: During the countdown,
  Settings shows "Answering again" and "Use this server now" but not the 1:20
  countdown or "Stay on vps", so the two screens disagree about what will
  happen. *Suggestion:* show the countdown line on the homebox row.
- **low**, `back-on-homebox-since-desktop`, `43-home-back-on-homebox`: Coming
  back later, the only trace is "Back on homebox since ...". Nothing says
  how long I was on vps, so I can't tell how long work or bank traffic left
  from the VPS. Peplink keeps an event log. *Suggestion:* add a line like "On
  vps 14:12–14:21 (homebox unreachable)", or a short switch history.
- **low**, `failover-using-vps-home-unreachable-desktop` vs `back-on-homebox-since-desktop`:
  The fixture times don't fit together: on vps since 14:12, but back on
  homebox since 14:11. Testers will notice. *Suggestion:* fix the fixture.
- **low**, `49-home-error-no-server-reachable`: The page header still says
  "homebox / via home server" while nothing is connected. *Suggestion:*
  show "Not connected" or "tailbond off" in the header in this state.
- **low**, `44-settings-servers-dark`: In dark mode the "Unreachable" dot is
  pale pink and hard to tell apart from a neutral dot. *Suggestion:* use a
  more saturated red in dark mode.

## Overall count

- high: 0
- medium: 11
- low: 15
