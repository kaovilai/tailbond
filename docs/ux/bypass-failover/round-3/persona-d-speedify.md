# Round 3, persona D: "Speedify user"

I've used Speedify for years (one Bypass list for apps and domains, a server
list with "nearest server" chosen for me, ping shown next to each server)
and Peplink/SpeedFusion at work. My yardstick: setting up a bypass or picking
a server takes seconds, and one glance tells me which server I'm on.

Screens used: every PNG in `out/ux/bypass-failover/glinet/` and
`out/ux/bypass-failover/android/`, read against `glinet-screens.md` and
`android-screens.md`.

---

## Task 1: make `mybank.com` skip the tunnel (router, then phone)

**Done? Yes**, but only partly on a router without dnsmasq-full.
Router: `bypass-empty`, `bypass-add-website-presets`, `bypass-rules`,
`bypass-website-rules-wait`. Phone: `46c-websites-none-yet`,
`47-add-website-dialog`, `47b-add-website-preset-banking`,
`46-websites-skip-tailbond`.

On the router, "Add a rule" → type `mybank.com` → "Use the internet directly
(now: Repeater)" → Add takes about as long as it does in Speedify. The rule row then shows
"Uses the internet directly (now: Repeater) · Bank · 37 connections", so I can
see it working. On the phone it is Settings → Websites that skip tailbond →
Add a website → type it → Add.

### Findings

- **medium**: `bypass-website-rules-wait-desktop.png`: if the router's DNS
  can't do website rules, my bank rule shows "Waiting: goes through tailbond"
  and the bank stays blocked. The fix is a plug-in install, and if that
  fails, an SSH one-liner. The one-liner runs `opkg remove dnsmasq` and then
  falls back to `opkg install dnsmasq`. That fallback has to download, but the
  router's own DNS was just removed, so a failed install can leave the router
  with no DNS or DHCP while I'm in a hotel. Speedify never asked me to touch a
  package manager.
  *Suggestion:* add a one-click "Install website support" button that runs
  the plug-in install. In the SSH line, also download the stock `dnsmasq`
  before removing it, so the fallback is local. Until the rule works, offer
  "Use a device rule for your phone/laptop instead" as a shortcut.

- **medium**: `46-websites-skip-tailbond.png`, `47-add-website-dialog.png`:
  on the phone a website covers only "its usual names (www., login., secure.,
  online. …)". The router covers every name under it. If my bank logs in from
  `auth2.mybank.com` or a separate domain, the phone rule silently misses it
  and the bank still blocks me. Speedify's domain bypass covers all
  sub-domains, so I'd assume this does too.
  *Suggestion:* after a rule is added, show which names have actually matched
  ("seen: www.mybank.com, login.mybank.com"), and list names the phone looked
  up under that domain that weren't covered, with an "add" button for each.

- **medium**: `47b-add-website-preset-banking.png`,
  `47d-add-website-preset-wifi-calling.png`: two of the three "Or pick one"
  chips add nothing. Banking highlights, shows a paragraph telling me to type
  my bank in the box above, and leaves Add disabled. Wi-Fi calling says
  "Nothing to add". A preset that adds nothing is a wasted tap and looks
  broken.
  *Suggestion:* for Banking, put the cursor in the Website box with the hint
  "your bank, e.g. mybank.com" and tag the rule "Bank". For Wi-Fi calling,
  show the note as plain text, not a selectable chip, or drop the chip on
  the phone.

- **low**: `bypass-rules-phone.png`, `bypass-empty-desktop.png`: the Bypass
  card sits below Internet connections, Speed or reliability and Home server.
  On a phone browser that is about two-thirds of the way down a very long
  page. Speedify gives bypass its own tab.
  *Suggestion:* add a jump link or tab ("Bypass rules (4)") in the top status
  card.

- **low**: `bypass-add-website-presets-desktop.png`: the add dialog has three
  grey paragraphs about DNS lookups, shared hosting and Private DNS between
  the Website box and "It should". It makes a ten-second task look risky.
  *Suggestion:* keep one line ("Covers mybank.com and every name under it")
  and put the rest behind "How website rules work".

- **low**: `46-websites-skip-tailbond.png`, `47c-add-website-preset-streaming.png`,
  `bypass-add-streaming-preset-desktop.png`: the streaming preset is different
  on each device. The router adds 5 services as one rule. The phone adds 2
  Netflix domains as two separate rows (`netflix.com`, `nflxvideo.net`).
  *Suggestion:* use the same list on both, and show a preset as one row you
  can expand to its domains.

- **low**: `46-websites-skip-tailbond.png`: the footer says phone rules apply
  first and then "the router's own rules treat the phone like any other
  device". That means on the router's Wi-Fi I need the bank rule in both
  places, or the phone's skipped traffic goes back into the router's tunnel.
  This is buried in the last sentence of a grey footnote.
  *Suggestion:* when the phone is on a tailbond router, show "Also add this on
  the router" next to each website.

- **low**: `bypass-wifi-calling-added-desktop.png`: the "added" toast says
  "(now: repeater)" in lowercase, but the row says "Repeater". The toast also
  floats over the Home server card, far from the card I just changed.
  *Suggestion:* use the same capitalisation and anchor the toast to the
  Bypass card.

## Task 2: make the work laptop always use tailbond

**Done? Yes**, with caveats I can't verify. Screens: `bypass-add-device-picker`
(desktop and phone), `bypass-rules`.

"A device" lists devices on the router, marks the ones that already have a
rule, and defaults to "Always use tailbond". The "Work laptop" quick pick is
also there. The rule row says "Always uses tailbond".

### Findings

- **medium**: `bypass-add-device-picker-desktop.png`: the rule follows the
  laptop's MAC address. The grey help says that if the address changes, the
  laptop "goes through tailbond like everything else, but website rules then
  apply to it". So with a private/random Wi-Fi address, the work laptop's
  traffic to mybank.com or Netflix leaves through the hotel. This breaks the
  one promise of this rule, and nothing on the rule row would tell me.
  *Suggestion:* if a device rule's MAC hasn't been seen while an unnamed
  device with a random-looking MAC is online, flag the row: "work-laptop not
  seen: it may be using a private address". Also consider blocking bypass for
  unknown devices while an "Always use tailbond" rule is unmatched.

- **medium**: `bypass-add-device-picker-desktop.png`,
  `servers-home-active-vps-standby-desktop.png`: "With the kill switch on
  (the default), it has no internet while tailbond can't connect". I can't
  find a kill switch anywhere on the tailbond page to check that it's on. "No
  leaks" is the whole point of this task.
  *Suggestion:* show the kill-switch state (on/off) next to the "Always use
  tailbond" rule and in the Servers card, with a link to where it's set.

- **low**: `bypass-add-device-picker-phone.png`: on a phone the dialog ends at
  the grey help text, and the Add button is off-screen inside the modal.
  *Suggestion:* pin Cancel/Add to the bottom of the dialog.

- **low**: `bypass-rules-desktop.png`: the same page shows "work-laptop ·
  connected now" in Bypass and "work-laptop · Last seen 1 day ago" in Tailnet
  machines. They are different things (LAN device vs. Tailscale node), but it
  reads as a contradiction.
  *Suggestion:* label the Bypass row "on this router's Wi-Fi now", or note in
  Tailnet machines that its Tailscale is off.

## Task 3: add the VPS as a backup to `homebox`

**Done? Partly.** Router: yes. Phone: yes. Clash: understood on the router,
not on the phone. Router: `servers-single-home`, `add-server-open`,
`add-server-waiting-approval`, `add-server-subnet-clash`,
`add-server-paste-clash`, `servers-home-active-vps-standby`. Phone:
`44b-settings-one-server-add-backup`, `45-add-another-server`,
`45b-add-another-server-confirm`, `45c-add-another-server-same-addresses`.

The router is good: "Found on your tailnet: vps → Ask to add", then a
waiting state that tells me exactly where to click Approve. After that, the
server list has up/down arrows, "Use now", and three "When to switch" options.

### Findings

- **medium**: `45-add-another-server.png`: the phone suggests every Tailscale
  machine as a server (`nas`, `vps`, `apple-tv`, `pixel-tablet`). The router
  shows only the machine actually running tailbond ("Found on your tailnet:
  vps"). Tapping `apple-tv` sends a request that will never be answered. The
  page also offers four ways to pair, all expanded, which is a long scroll
  for "add server".
  *Suggestion:* list only machines that answer as tailbond servers, with
  one-tap "Ask to add", like the router. Put "Other ways (page, QR code,
  link)" behind one expander.

- **medium**: `45c-add-another-server-same-addresses.png`: the clash dialog
  says "whoever runs vps needs to change its range", but I run vps. It gives
  no steps, only "Copy a message to send them". The router at least has a
  "How they change it" expander. On the phone I'm stuck.
  *Suggestion:* add the same "How to change it" steps on the phone (or a
  "I run vps: show me" button), and stop saying "them" when the user owns
  the server.

- **low**: `add-server-subnet-clash-desktop.png`,
  `add-server-paste-clash-desktop.png`: the actual fix on the router is
  collapsed under "How they change it", below a big "Copy a message to send
  them" button. That ordering assumes someone else owns the VPS.
  *Suggestion:* open the steps by default, or put them first.

- **low**: `servers-home-active-vps-standby-desktop.png`, `44-settings-servers.png`:
  neither server list shows latency or quality, only "answered 5 s ago" or
  "Reached at …". In Speedify I always pick by the ping number.
  *Suggestion:* show round-trip time per server ("homebox 48 ms · vps 22 ms")
  next to its status.

- **low**: `add-server-open-desktop.png`, `45-add-another-server.png`,
  `44-settings-servers.png`: the names don't match. The button says "Add a
  backup server". The section and page title say "Add another server". The
  phone page then says "Add a backup server" again under that title. The
  router tag is "Backup"; the phone tag is "Backup 1".
  *Suggestion:* use one name ("Add a backup server") and the same tag on both
  devices.

## Task 4: notice the switch to the VPS, and the return to homebox

**Done? Yes.** Router: `failover-using-vps-home-unreachable` (desktop and phone),
`failback-countdown`, `back-on-homebox-since`, `failover-you-picked-vps`,
`servers-recent-switches-all`. Phone: `41`, `41b`, `41c`, `41d`, `42`,
`43`, `43b`, `44`, `44c`, `44d`, `49`.

Both devices pass the at-a-glance test. The router headline reads "Using vps
since 14:00 (homebox is unreachable)". The phone title is "vps · via backup
server", and the notification says "Connected through vps". The return is
also timed: "Back on homebox since 14:21" and "Back to homebox in 1:20".
Recent switches give the history with reasons.

### Findings

- **medium**: `40-home-on-homebox-vps-on-standby.png`,
  `41b-home-using-vps-first-screen.png`: on the phone's main screen I can see
  the servers but can't pick one. The vps row isn't tappable (no chevron),
  and "Use this server now" exists only in Settings. Speedify puts server
  choice on the main screen.
  *Suggestion:* make each server row on the main screen open a small sheet
  with "Use this server now" and "Move up/down".

- **medium**: `43b-home-switching-to-vps.png`, `49-home-error-no-server-reachable.png`:
  by default the phone sends traffic over the hotel's internet during every
  switch. If both servers are down, tailbond stops and the phone stays on
  the hotel internet until I tap "Try again". The way to block that is in
  Android settings, and it's only "recommended". The router's kill switch is
  on by default, so the two devices behave differently.
  *Suggestion:* keep retrying in the background after "Can't reach any of your
  servers" instead of stopping. When the first backup server is added, prompt
  once: "Turn on Block connections without VPN?", with the button.

- **low**: `failover-using-vps-home-unreachable-desktop.png`,
  `41-home-using-vps-homebox-unreachable.png`: the failover state uses the
  same green icon (router) or blue card (phone) as normal operation. Only the
  text changes.
  *Suggestion:* use an amber accent or a "Backup" badge on the status card
  while on a non-main server.

- **low**: `failback-countdown-desktop.png`: during the countdown, homebox's
  router row says grey "Standby". The phone says green "Answering again". The
  router headline also drops "since 14:00".
  *Suggestion:* show "Answering again · back in 1:20" on the router row, and
  keep "since 14:00" in the headline.

- **low**: `43b-home-switching-to-vps.png`: the screen title already says
  "vps · via backup server" while the card still says "Connecting… Reaching
  vps…".
  *Suggestion:* show "Switching to vps…" in the title until it connects.

- **low**: `44d-settings-servers-paused-you-picked-vps.png`: "Paused: you
  picked vps" is squeezed between two radio options and wraps onto two
  lines.
  *Suggestion:* show it as a full-width banner above "When to switch".

- **low**: `44-settings-servers-dark.png`: in dark mode the "Unreachable" dot
  is pale pink and hard to tell apart from the neutral grey dots.
  *Suggestion:* use a saturated red, or add an icon, in dark mode.

---

## Count

| Severity | Count |
|---|---|
| High | 0 |
| Medium | 9 |
| Low | 15 |
