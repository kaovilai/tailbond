# Round 3: Persona A, "GL traveller"

I'm a consultant who travels every month with a GL.iNet Slate 7. I know GL's
admin panel (Repeater, VPN Dashboard, Plug-ins) but I'm not a network
engineer. I want the hotel setup to just work, banking and video calls to
keep working, and work traffic never to leave through the hotel.

Screens used: all PNGs in `out/ux/bypass-failover/glinet/` and
`out/ux/bypass-failover/android/`.

---

## Task 1: Make `mybank.com` skip the tunnel (router, then phone)

**Done?** Yes on both. On the router it only works once the router has
dnsmasq-full; if it doesn't, the steps to get it are risky for me.

Screens: `bypass-empty-desktop`, `bypass-add-website-presets-desktop`,
`bypass-rules-desktop`, `bypass-rules-phone`, `bypass-website-rules-wait-desktop`,
`bypass-add-streaming-preset-desktop`, `bypass-add-wifi-calling-desktop`,
`bypass-wifi-calling-added-desktop`, `46c-websites-none-yet`,
`47-add-website-dialog`, `47b-add-website-preset-banking`, `47c-…streaming`,
`47d-…wifi-calling`, `46-websites-skip-tailbond` (+dark),
`46b-websites-needs-android-13`, `48-settings-skip-rows-android-12`.

On the router I clicked "Add a rule", typed mybank.com, kept "Use the internet
directly (now: Repeater)" and the row shows "Bank · 37 connections". On the
phone I went to Settings → Websites that skip tailbond → Add a website. The
dialog even points me to "Apps that skip tailbond" for the bank's app.

### Findings

- **medium**: `bypass-rules-desktop`, `bypass-add-website-presets-desktop`
  - The card's fix for devices with their own secure DNS is "use a device
    rule for those". On my phone (Private DNS on) that sends the whole phone
    outside tailbond just to fix one bank, which is the opposite of what I
    want.
  - Suggestion: suggest turning off Private DNS / Secure DNS for this
    router's Wi-Fi, or using the tailbond phone app's website list. Mention
    the device rule last, with a warning that everything from that device
    then skips tailbond.
- **medium**: `bypass-website-rules-wait-desktop`
  - The Plug-ins step is fine for me. The fallback is not: SSH in and paste
    a command that includes `opkg remove dnsmasq`. That removes the router's
    DNS while I depend on hotel Wi-Fi.
  - The notice also says to "turn tailbond off and on here" without saying
    what my work laptop does in the meantime.
  - Suggestion: make the install a single button in the tailbond page, which
    can run the safe swap itself. Drop the SSH fallback or hide it behind
    "Advanced". Say what happens to "Always uses tailbond" devices during the
    off/on.
- **low**: `47b-add-website-preset-banking`
  - Tapping "Banking" highlights the chip but fills nothing. "Add" stays
    greyed out, and the instruction to type in the box is far below the box.
  - Suggestion: put the cursor in the Website field when Banking is tapped,
    with the placeholder "your bank, e.g. mybank.com".
- **low**: `47c-add-website-preset-streaming`, `bypass-add-streaming-preset-desktop`
  - The same "Streaming" quick pick adds 2 sites on the phone (Netflix only)
    and 5 on the router (Netflix, Disney+, Prime, Max).
  - Suggestion: use the same list on both, or label the phone one "Netflix".
- **low**: `46-websites-skip-tailbond`
  - Every row repeats "Covers X and its usual names (www., login., secure.,
    online. …)", including Netflix, where login./secure. make no sense. Rows
    are 4 lines each and hard to scan.
  - Suggestion: put the "covers usual names" note once at the top and keep
    each row to its name and category.
- **low**: `46-websites-skip-tailbond`, `47-add-website-dialog`
  - "On a tailbond travel router's Wi-Fi, these phone rules apply first; the
    router's own rules then treat the phone like any other device." I had to
    work out that the bank needs a rule on both the router and the phone.
  - Suggestion: say it plainly: "On your tailbond router's Wi-Fi, add the
    website on the router too."
- **low**: `bypass-wifi-calling-added-desktop`
  - The confirmation toast says "(now: repeater)" in lowercase where the row
    says "Repeater", and it sits on top of the homebox card instead of near
    the new rule.
  - Suggestion: use the same capitalisation and anchor the toast to the
    Bypass card.
- **low**: `bypass-rules-desktop`
  - Fire-TV-Stick shows the same laptop icon as work-laptop.
  - Suggestion: use a TV or generic device icon.

## Task 2: Work laptop always uses tailbond

**Done?** Partly. I can add the rule, and the dialog says it overrides website
and port rules. The screens don't let me confirm that "never leaves through
the hotel" really holds.

Screens: `bypass-add-device-picker-desktop`, `bypass-add-device-picker-phone`,
`bypass-rules-desktop`, `servers-home-active-vps-standby-desktop`,
`failover-using-vps-home-unreachable-desktop`.

### Findings

- **medium**: `bypass-add-device-picker-desktop`, `servers-home-active-vps-standby-desktop`
  - Both say "With the kill switch on (the default)…". No screen shows a
    kill switch, whether it's on, or where to check it.
  - Nothing says what happens to an "Always uses tailbond" device when I flip
    the big tailbond toggle off, which the dnsmasq notice tells me to do.
  - Suggestion: show "Kill switch: On" with a link in the Bypass card or next
    to the main toggle. Under the work-laptop row, say "No internet while
    tailbond is off or reconnecting".
- **medium**: `bypass-add-device-picker-desktop`, `bypass-rules-desktop`
  - The rule follows the laptop's MAC. The dialog explains that if the
    address changes, website and port rules apply to it again: bank sites
    direct, and UDP 500/4500 (many work VPNs) direct.
  - The rule row would then just stop matching without any warning. I'd only
    know if I re-read the small print.
  - Suggestion: when a rule's device hasn't been seen for a while but a new
    unnamed device shows up, flag the row ("work-laptop not seen since …; a
    new device appeared, is it this one?") with a one-click re-link.
- **low**: `bypass-rules-desktop`
  - The bypass row says work-laptop is "connected now", while Tailnet machines
    on the same page says work-laptop was "Last seen 1 day ago". Same name,
    two answers.
  - Suggestion: in Tailnet machines, label it "its own Tailscale: last seen 1
    day ago", or explain that the two lists are different things.
- **low**: `bypass-add-wifi-calling-desktop`
  - The Wi-Fi calling rule sends UDP 500/4500 direct for every device. Those
    are also IPsec VPN ports. Nothing says this, or that device rules win.
  - Suggestion: add "Applies to every device except ones set to Always use
    tailbond."
- **low**: `bypass-add-device-picker-phone`
  - On the phone-sized panel the device dialog fills the screen and the
    Add/Cancel buttons are below the fold.
  - Suggestion: keep the action buttons pinned at the bottom of the dialog.

## Task 3: Add the VPS as backup

**Done?** Partly. Adding vps is easy on both, and the router even found vps
itself. When it clashed I understood what was wrong, but not that I'm the one
who has to fix it, or how.

Screens: `servers-single-home-desktop`, `add-server-open-desktop`,
`add-server-waiting-approval-desktop`, `add-server-subnet-clash-desktop`,
`add-server-paste-clash-desktop`, `servers-home-active-vps-standby-desktop`,
`44b-settings-one-server-add-backup`, `45-add-another-server`,
`45b-add-another-server-confirm`, `45c-add-another-server-same-addresses`.

### Findings

- **medium**: `add-server-subnet-clash-desktop`, `add-server-paste-clash-desktop`, `45c-add-another-server-same-addresses`
  - Both say "whoever runs vps needs to change its range" and offer "Copy a
    message to send them". I rented the VPS; I am "them".
  - On the router the steps are behind a collapsed "How they change it". On
    the phone there are no steps at all, only the copy button.
  - Suggestion: lead with "If you run vps: open http://vps…:8077 → … → change
    range" as visible steps. Make "Copy a message" secondary ("Someone else
    runs it?").
- **medium**: `45-add-another-server`
  - The phone offers nas, vps, apple-tv and pixel-tablet as chips to "ask".
    Only vps runs tailbond. The router shows only "Found on your tailnet:
    vps".
  - Below that are three more ways to add (server page, QR code, link). I
    don't know which one to use.
  - Suggestion: like the router, show only machines that run tailbond. Fold
    the other three ways under "Other ways".
- **low**: `add-server-open-desktop`, `servers-home-active-vps-standby-desktop`, `44-settings-servers`, `40-home-on-homebox-vps-on-standby`
  - The router and phone use different words for the same thing:
    - "First choice" vs "Main"
    - "Standby" vs "Ready as backup"
    - "Backup" vs "Backup 1"
    - "Use now" vs "Use this server now"
    - "Can't reach" vs "Unreachable"
    - "Add a backup server" vs "Add another server", which appear on the same
      router flow
  - Suggestion: pick one set of words for both.
- **low**: `45b-add-another-server-confirm`
  - "Your Tailscale machines see this phone as vps" reads oddly. The more
    useful point for me is that websites, including my bank, will see a
    datacentre address. That's in Settings but not here.
  - Suggestion: add "Websites will see vps's address, which some banks treat
    as a VPN. Your bank rule still goes direct."

## Task 4: Notice the switch and the return

**Done?** Yes. The router header ("Using vps since 14:00 (homebox is
unreachable)"), the countdown ("Back to homebox in 1:20 … Stay on vps"), "Back
on homebox since 14:21" and Recent switches tell the story. The phone does the
same, and its notification text follows along.

Screens: `failover-using-vps-home-unreachable-desktop` (+phone),
`failback-countdown-desktop`, `back-on-homebox-since-desktop`,
`failover-you-picked-vps-desktop`, `servers-recent-switches-all-desktop`,
`40-…`, `41-…`, `41b-…`, `41c-…`, `41d-notification-using-vps`, `42-…`, `43-…`,
`43b-home-switching-to-vps`, `44-settings-servers` (+dark), `44c-…`, `44d-…`,
`49-home-error-no-server-reachable`.

### Findings

- **medium**: `43b-home-switching-to-vps`, `49-home-error-no-server-reachable`, `44-settings-servers`, `48-settings-skip-rows-android-12`
  - By default the phone falls back to its normal internet while it
    switches, and whenever neither server answers. That leaks work traffic
    through the hotel.
  - The fix ("Block connections without VPN") is described alongside "hotel
    sign-in pages can't open and you can't hand back to Tailscale". So I'd
    have to toggle an Android system setting at every hotel.
  - Suggestion: let tailbond hold traffic during a switch by itself (as the
    router's kill switch does). Or let "Block" stay on and have tailbond open
    sign-in pages through its own "Tell me when Wi-Fi needs a sign-in" flow.
- **low**: `41d-notification-using-vps`
  - The switch only changes the text of the ongoing notification. There's no
    alert, and the countdown notification has only "Disconnect", not "Stay on
    vps".
  - Suggestion: add a one-time quiet alert on a switch, and a "Stay on vps"
    action during the countdown.
- **low**: `servers-recent-switches-all-desktop`, `failover-you-picked-vps-desktop`
  - The router's Recent switches show times only (14:00, 14:21). The phone
    shows "Oct 31, 09:40" for older entries. After a week of travel the
    router list would be ambiguous.
  - Suggestion: add the date to entries that aren't from today, like the
    phone does.
- **low**: `failback-countdown-desktop`, `42-home-back-to-homebox-in-1-20`
  - The router's countdown header drops "since 14:00", which the "using vps"
    header had. I lose how long I've been on vps.
  - Suggestion: keep "since 14:00" in the countdown header.
- **low**: `failover-using-vps-home-unreachable-desktop`
  - With home's power off, Tailnet machines still shows nas as "Online ·
    direct". The caveat under it explains, but the green dot wins.
  - Suggestion: grey out or mark "may be down" machines that are only
    reachable through the unreachable server.
- **low**: `49-home-error-no-server-reachable`
  - "Tell whoever set up your servers" assumes someone else did; I did. The
    server rows just say "Main server" / "Backup" in grey, not that each one
    failed.
  - Suggestion: show "Unreachable" on each row. Add a hint like "Check that
    homebox has power and internet; check vps in your hosting panel".

---

## Overall count

| Severity | Count |
|---|---|
| High | 0 |
| Medium | 7 |
| Low | 16 |
