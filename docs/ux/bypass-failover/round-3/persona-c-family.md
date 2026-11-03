# Round 3: Persona C, "family member"

Persona: a relative of the person who set everything up. Uses the phone app,
and opens the travel router's admin page only when asked to. Doesn't know what
a VPN server, subnet or port is. Worries about breaking things. Wants to know
two things: "is it working?" and "what should I do?".

Screens used: every PNG under `out/ux/bypass-failover/glinet/` and
`out/ux/bypass-failover/android/`, read with the two manifests.

---

## Task 1: Make the bank's website skip the tunnel (router, then phone)

**Done?** Partly. On the router, yes: the "Banking" quick pick and "Add a rule"
are on the card, and the row then says "Uses the internet directly (now:
Repeater) · Bank · 37 connections", which reassures me. On the phone I added
`mybank.com`, but nothing tells me it works, and the "Banking" button confused
me. If the router has the "Website rules are waiting" problem, I'm stuck.

Screens: bypass-empty, bypass-add-website-presets, bypass-rules (desktop +
phone), bypass-website-rules-wait, 46-websites-skip-tailbond (+dark),
46c-websites-none-yet, 46b-websites-needs-android-13, 47-add-website-dialog,
47b/47c/47d-add-website-preset-*, 48-settings-skip-rows-android-12,
44-settings-servers.

### Findings

- **medium**: `bypass-website-rules-wait-desktop.png`. My bank rule shows
  "Waiting: goes through tailbond until website rules work". To fix it, the
  notice says to install "dnsmasq-full", or to "run this over SSH" a command
  that includes `opkg remove dnsmasq`. I don't know what SSH is. Pasting a
  command that *removes* something from the router is exactly what I'm afraid
  of, and the notice never says I can leave it to someone else.
  *Suggestion:* lead with "Your bank still goes through tailbond for now.
  Nothing is broken. Ask whoever set up this router to turn on website rules"
  and put a "Copy a message to send them" button there, as the server clash
  does. Put the Plug-ins/SSH steps behind a collapsed "How to do it yourself".
  Offer a fallback now ("Meanwhile, a device rule works: pick your phone or
  laptop").

- **medium**: `47b-add-website-preset-banking.png`. On the phone, tapping
  "Banking" turns the chip dark, but nothing is added and "Add" stays grey. The
  chip sits next to "Streaming", which *does* add websites ("Add 2 websites"),
  so I expected Banking to add my bank. Instead I had to read a paragraph to
  learn that I must type the name myself.
  *Suggestion:* when Banking is tapped, move focus into the Website box with
  hint "Type your bank's website, e.g. mybank.com". Or rename the chip "My
  bank (type its website)". Don't style it like the chips that add things.

- **medium**: `46-websites-skip-tailbond.png` (+ `-dark`). The phone list shows
  only a switch and a description per website. Nothing says it's actually
  working (no "used just now", no count). The router shows "37 connections";
  the phone shows nothing. I can't answer "is it working?".
  *Suggestion:* add a status line per row, e.g. "Skipping tailbond · used 2 min
  ago" or "Not used yet. Open mybank.com to check". Add "Off" on rows that are
  turned off (tax.example.gov is only a grey switch).

- **medium**: `46-websites-skip-tailbond.png`, `47-add-website-dialog.png`.
  The task asked me to set this up on the router *and* the phone. The phone
  page says "On a tailbond travel router's Wi-Fi, these phone rules apply
  first; the router's own rules then treat the phone like any other device",
  and the dialog says the router "covers every name under a website; a phone
  can only cover names it looks up itself". I still can't tell whether I need
  both, or whether the two rules will fight.
  *Suggestion:* one plain sentence on both sides, e.g. "Add your bank on the
  phone for when you're out. On the travel router's Wi-Fi, the router's rule
  covers it too. Having both is fine."

- **low**: `bypass-add-website-presets-desktop.png`. "It should: Use the
  internet directly (now: Repeater)". I don't know that "Repeater" means the
  hotel Wi-Fi. The top card says "Repeater (Wi-Fi)", but the dropdown doesn't.
  *Suggestion:* "Use the internet directly (now: hotel Wi-Fi via Repeater)" or
  "(now: Wi-Fi, Repeater)".

- **low**: `bypass-add-website-presets-desktop.png`,
  `bypass-add-streaming-preset-desktop.png`. Three gray paragraphs under the
  Website box (matching addresses, big hosting providers, Private DNS, Secure
  DNS in Chrome, "every name under it") are more than I can follow. The one
  thing I care about ("my phone may have Private DNS on, so this may not work
  for it") is buried.
  *Suggestion:* keep the first line ("Includes www.mybank.com and
  login.mybank.com") and put the rest behind "Why might it not work?".

- **low**: `44-settings-servers.png`, `48-settings-skip-rows-android-12.png`.
  On the phone, "Websites that skip tailbond" sits below the server cards,
  "When to switch", "Recent switches" and an "Internet traffic" choice full of
  words I don't know ("exit node", "MagicDNS", "100.x addresses", "LANs behind
  other subnet routers"). I could tap "Only Tailscale machines" by mistake
  while scrolling, and it changes where *all* my traffic goes.
  *Suggestion:* move the "skip tailbond" rows above "Internet traffic", or
  group the expert choice under an "Advanced" heading. Rewrite "Only Tailscale
  machines" as "Only use tailbond to reach your own computers. Websites use
  this phone's normal internet."

- **low**: `bypass-wifi-calling-added-desktop.png`. The "Wi-Fi calling: uses
  the internet directly (now: repeater)" confirmation pops up over the *Home
  server* card, far from the rule I just added. "repeater" is lower-case here
  and capitalised everywhere else.
  *Suggestion:* show the confirmation next to the new row, or highlight the new
  row briefly. Use "Repeater" consistently.

---

## Task 2: Make the work laptop always use tailbond

**Done?** Partly. The "Work laptop" quick pick and the device list make it easy
to choose the laptop, and the row says "Always uses tailbond". But it only
stays safe if a laptop setting that I can't check or change stays off, and if
it breaks, nothing tells me.

Screens: bypass-rules (desktop + phone), bypass-add-device-picker (desktop +
phone), bypass-empty.

### Findings

- **high**: `bypass-add-device-picker-desktop.png`,
  `bypass-add-device-picker-phone.png`. The help text says the rule follows the
  laptop's "MAC address" and that I should turn off "Private Wi-Fi address"
  ("Random hardware addresses") on the laptop. Then it says: "If its address
  changes anyway, it goes through tailbond like everything else, but website
  rules then apply to it." So if the work laptop changes its address, its
  traffic to the bank and streaming sites quietly goes out through the hotel's
  network. That's exactly what the task says must never happen. I don't know
  how to check this on a work laptop. The rule row would still say "Always uses
  tailbond", with nothing telling me it stopped matching.
  *Suggestion:* when an "Always use tailbond" device hasn't been seen with its
  saved address for a while, or a new unnamed device with a randomised address
  appears, show a warning on the row: "work-laptop hasn't been seen at this
  address since …: its traffic may be skipping tailbond for bank/streaming
  sites. Ask whoever set it up." Offer "This is my work laptop" on new unknown
  devices so the rule can be moved to them. In the dialog, say plainly what
  goes wrong: "If the laptop changes its address, bank and streaming sites
  from it will use the hotel's internet."

- **medium**: `bypass-rules-desktop.png` (and every client screen with both
  cards). The bypass row says "work-laptop … connected now", but under
  "Tailnet machines" the same name says "work-laptop … Last seen 1 day ago". I
  can't tell whether my laptop is connected and protected.
  *Suggestion:* label the two differently, e.g. "on this router's Wi-Fi now" in
  the bypass row and "Tailscale app on it: last seen 1 day ago" in the Tailnet
  list. Or add one line explaining that these are two different things.

- **low**: `bypass-add-device-picker-phone.png`. On a phone-sized screen the
  dialog ends at the "kill switch" text, and the Add/Cancel buttons are below
  the fold. "Kill switch" isn't explained in words I know.
  *Suggestion:* keep Add/Cancel pinned at the bottom of the dialog. Say "If
  tailbond is down, this device has no internet (so nothing leaks)" in place
  of "kill switch".

---

## Task 3: Add a VPS as a backup server, and understand a clash

**Done?** Partly. On the router: "Found on your tailnet: vps → Ask to add" and
the waiting message are clear. On the phone, the suggested machines include
things that aren't servers. When the clash appears, both apps assume someone
else runs vps, but in this task I rented it myself.

Screens: servers-single-home, add-server-open, add-server-waiting-approval,
add-server-subnet-clash, add-server-paste-clash,
servers-home-active-vps-standby, 44b-settings-one-server-add-backup,
45-add-another-server, 45b-add-another-server-confirm,
45c-add-another-server-same-addresses.

### Findings

- **medium**: `45-add-another-server.png`. "Or tap one of your Tailscale
  machines to ask it" lists nas, vps, apple-tv and pixel-tablet. I can't tell
  which one is a tailbond server, and could tap "nas" or "apple-tv". The router
  page only suggests vps ("Found on your tailnet: vps").
  *Suggestion:* show only machines that answer as tailbond servers, the way the
  router does ("Found: vps"), or mark the others "not a tailbond server".

- **medium**: `add-server-subnet-clash-desktop.png`,
  `add-server-paste-clash-desktop.png`,
  `45c-add-another-server-same-addresses.png`. The clash message says
  "whoever runs it changes its range" and offers "Copy a message to send
  them". In this task, *I* run vps, so there's no "them", and the actual fix is
  hidden under "How they change it". The phone version also uses "10.77.0.0/24
  inside the tunnel" and "range" with no explanation. The reassurance
  ("homebox keeps working; nothing changed") is good.
  *Suggestion:* split it into two options: "I run vps: show me what to change"
  (expands the steps) and "Someone else runs it: copy a message for them".
  Replace "uses 10.77.0.0/24" with "both servers want to give out the same
  addresses". Keep the numbers inside the copied message, not the headline.

- **low**: `add-server-paste-clash-desktop.png`. The warning says to try "Its
  name in Tailscale" above, but the field above is labelled "Or type its name
  in Tailscale". A "Found on your tailnet: vps · Ask to add" button is right
  there, and the warning doesn't point to it.
  *Suggestion:* "Tap **Ask to add** next to vps above instead (a pasted link
  can't do this)."

- **low**: `add-server-waiting-approval-desktop.png`. "Waiting for approval on
  vps…" tells me to open another page and click Approve next to "slate7". It
  doesn't say what happens if nobody approves, or whether I can close this page
  safely.
  *Suggestion:* add "You can close this page; homebox keeps working. If nobody
  approves within N minutes, this request ends and you can ask again."

- **low**: `servers-single-home-desktop.png`, `add-server-open-desktop.png`,
  `45-add-another-server.png`. Different words for the same thing: the button
  says "Add a backup server", the section that opens says "Add another server",
  the router says "Ask to add" and the phone says "Ask to connect". I wondered
  whether I'd clicked the wrong thing.
  *Suggestion:* use one name ("Add a backup server") and one action verb ("Ask
  to add") on both.

- **low**: `servers-single-home-desktop.png`,
  `servers-home-active-vps-standby-desktop.png`. Red "Unpair" sits right next
  to "Ask to connect again", and the red "Forget all servers" sits under the
  server list. Unlabeled ↑/↓ arrows on each server row change which server is
  first. An anxious user could click any of them to "try something".
  *Suggestion:* move Unpair/Forget into a "More" menu with a confirm dialog.
  Give the arrows a tooltip/label ("Make first choice").

---

## Task 4: Notice the switch, and that it came back

**Done?** Yes on the router, mostly yes on the phone. The router headline
"Using vps since 14:00 (homebox is unreachable). You're still online; nothing
to do" answers both of my questions. "Back to homebox in 1:20" and "Back on
homebox since 14:21" tell me when it came back. The phone says the same in its
main card. Problems: "you picked it" when I didn't, the notification, and the
"switching" screen.

Screens: servers-home-active-vps-standby, servers-recent-switches-all,
failover-using-vps-home-unreachable (desktop + phone), failback-countdown,
failover-you-picked-vps, back-on-homebox-since, 40/40-dark, 41, 41b, 41c, 41d,
42, 43, 43b, 49, 44/44-dark, 44c, 44d.

### Findings

- **medium**: `failover-you-picked-vps-desktop.png`,
  `41c-home-using-vps-you-picked-it.png`, `44d-settings-servers-paused-you-picked-vps.png`.
  It says "Using vps since 14:36 (you picked it)", "Paused: you picked vps".
  I didn't. Someone else (on another device, or the person who set it up)
  did. Seeing "you" makes me think I broke something by accident.
  *Suggestion:* "picked by hand at 14:36" (name the device/person if known,
  e.g. "from the phone pixel-8"). Keep the "Go back to homebox" button.

- **medium**: `41d-notification-using-vps.png`. The notification reads
  "Connected through vps · Using vps since 14:02 (homebox is unreachable)". The
  only button is "Disconnect". In the notification shade I don't see the
  app's "You're still connected… nothing to do", so "unreachable" and a single
  "Disconnect" button invite me to press it, which turns protection off.
  *Suggestion:* add "still protected, nothing to do" to the notification text
  (e.g. "homebox is down; using backup vps. You're still protected."). Consider
  "Open" as the first action and keep Disconnect second.

- **medium**: `43b-home-switching-to-vps.png`. During the few seconds of the
  switch, the card says "your phone uses its normal internet, unless Android's
  'Block connections without VPN' is on (recommended…)" and offers "Open
  Android VPN settings". I can't fix it in those seconds, and "uses its normal
  internet" makes me think something leaked. It gives me a job right when I
  should do nothing.
  *Suggestion:* during the switch show only "Switching to the backup server
  vps… (a few seconds). Nothing to do." Put the recommendation in Settings and
  as a one-time tip after the switch is done, not on this transient screen.

- **low**: `failover-using-vps-home-unreachable-desktop.png`,
  `41-home-using-vps-homebox-unreachable.png`. "homebox is unreachable" tells
  me *what* happened but gives no likely reason, and doesn't say whether I
  should tell anyone. My first thought is "did I do this?".
  *Suggestion:* add a plain hint: "Often a power or internet cut at home. If
  it lasts more than a day, tell whoever set up homebox."

- **low**: `41-home-using-vps-homebox-unreachable.png`. homebox shows a red
  "Unreachable" in Servers and a green dot under Machines. The note "(Tailscale
  may still show it online below)" helps, but two colours for one machine
  still worry me.
  *Suggestion:* grey out the homebox row under Machines while tailbond can't
  reach it, or add "(not answering tailbond)" on it.

- **low**: `44-settings-servers.png`, `44-settings-servers-dark.png`. The
  homebox card says "Unreachable" and then "Reached at home.example.net:443".
  "Reached" next to "Unreachable" reads like a contradiction. In dark mode the
  unreachable dot is pale pink and hard to see as an alarm.
  *Suggestion:* "Address: home.example.net". Use a stronger red in dark theme.

- **low**: `44c-settings-servers-homebox-answering-again.png`. The main screen
  says "Back to homebox in 1:20", but Settings shows only "Answering again" and
  a "Use this server now" link, with no countdown. Recent switches doesn't list
  "homebox answering again". I can't tell from here whether it will go back by
  itself.
  *Suggestion:* show "Going back to homebox in 1:20" on the homebox card in
  Settings too.

- **low**: `servers-home-active-vps-standby-desktop.png`,
  `servers-recent-switches-all-desktop.png`. Router "Recent switches" show only
  times (14:00, 14:21 …), with no date. The phone shows "Oct 31, 09:40" for
  older entries. On the router I can't tell whether 14:00 was today.
  *Suggestion:* add "today"/"yesterday"/the date as the phone does.

---

## Overall count

| Severity | Count |
|---|---|
| High | 1 |
| Medium | 10 |
| Low | 14 |
