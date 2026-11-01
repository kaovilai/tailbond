# Round 3: persona (b), homelab power user

I run subnet routers, an exit node, ACLs and MagicDNS, and I use the official
Tailscale Android app every day. I'm judging only from the screenshots in
`out/ux/after-3/`.

## Task 1: Install and pair from the phone

**Tap path:** Tailscale stays on. Open the server page in the browser
(web-01-add-this-phone). Tap "Download tailbond for Android" and install the
app. Back on the page, tap "+ Add this phone" (web-02-after-tap), then "Open in
tailbond". The app shows "Add this phone to homebox?" (03-deeplink-confirm);
tap "Add phone".

The flow is clear and short, and the paste error is specific (02-pairing-paste-error).

**Stuck points**
- 03-deeplink-confirm says "tailbond will connect to your home server homebox
  over Wi-Fi and mobile data", and the next screen says "It connects over the
  internet to home.example.net:443". Nowhere does it say that all internet
  traffic will leave from home from now on. Settings later shows "Internet
  traffic: Through your home server ... like using an exit node", and it's
  already selected. The official app never turns on an exit node unless you
  choose one. **Low.**
- web-02-after-tap: "Opening tailbond···" uses a row of middle dots instead
  of an ellipsis. **Low.**

## Task 2: Connect

**Tap path:** Tap the switch. 04-handoff-other-vpn asks "Turn off Tailscale's
VPN?"; tap "Turn on tailbond", then OK on the Android prompt. The home screen
then shows "Connected" (10-home-connected).

**Stuck points**
- 10-home-connected, This device: "pixel-8 (this phone) 10.77.0.2 / On your
  tailnet through homebox (100.101.1.1)". The phone has no 100.x address and
  no MagicDNS name of its own. The app doesn't explain what that means:
  - Other tailnet machines can't reach the phone.
  - ACLs see the phone as homebox.
  - The copy button copies 10.77.0.2, which is useless to any peer.

  In the official app, "This device" is a real tailnet node with its own
  100.x IP and MagicDNS name. The screen needs one line such as "Other
  machines see this phone as homebox; it can't be reached from the
  tailnet". **Medium.**
- No screen says whether subnet routes advertised by my other subnet
  routers (for example 192.168.x LAN ranges) work through homebox. The only
  choices are "Through your home server" and "Only Tailscale machines"; there
  is no exit-node picker like the official app's Exit node row. I can't tell
  what is routed where. **Medium.**

## Task 3: Find and copy the NAS's MagicDNS name or IP

**Tap path:** On the home screen, type "nas" in "Search machines" and tap the
copy icon. The toast reads "Copied 100.101.2.2" (18-machines-search-copied).
Or tap the row to open 19-machine-detail, which lists MagicDNS name, Short
name, Tailscale IPv4 and IPv6, each with a copy button.

This matches the official app well. No real problems.

## Task 4: Are Wi-Fi and 5G both in use?

**Tap path:** The home screen answers it: "Using both Wi-Fi and mobile data",
with both tiles at "In use" and their rates (10-home-connected). There's more
detail under Connections (20-connections).

**Stuck points**
- 15-one-path-failed: the Mobile data tile says "Not working right now" with
  no reason (no signal, server unreachable over cellular, or blocked by the
  carrier) and nothing to do about it. I'd expect at least the last error, or
  "can't reach homebox over mobile data". **Low.**

## Task 5: Save cellular data

**Tap path:** Turn on the "Save mobile data" switch on the home screen. 16-save-data-standby
then shows "Using Wi-Fi · saving mobile data (backup only)" and "Not in use ·
saving data". This is easy.

**Stuck points**
- 21-connections-save-data has three controls that all limit mobile data, and
  only one of the relationships is explained:
  - the "Save mobile data" mode;
  - the "Use for tailbond" switch on the Mobile data card;
  - "Mobile data limit".

  The limit is described as "Separate from Save mobile data". The screen
  never says how turning off "Use for tailbond" for mobile data differs from
  "Save mobile data" (is there no failover at all?). **Low.**
- "Advanced: link roles" is unexplained jargon sitting next to the mode
  list. **Low.**

## Task 6: Turn tailbond off and go back to Tailscale

**Tap path:** Tap "Switch back to Tailscale" on the home screen. That leads to
12b-home-switched-back-to-tailscale; then tap "Open Tailscale" and turn on
Tailscale's switch.

**Stuck points**
- 12b contradicts itself. The status card still says "Tailscale stays off too
  until you switch back to it", but the card right below says "Tailscale is
  in charge on this phone". I just switched back, so which is true?
  **Medium.**
- The same card says "Tailscale is in charge on this phone", then "If
  Tailscale isn't connected yet, open it and turn on its switch". So the
  switch back doesn't actually bring Tailscale up, and "in charge" may be
  false. Settings (30-settings) promises the button "Turns tailbond off and
  opens the Tailscale app", but 12b shows me still inside tailbond with an
  extra "Open Tailscale" button. I need an extra tap, and it's unclear
  whether my tailnet is reachable right now. **Medium.**
- 12-home-disconnected: "Tailscale stays off too until you switch back to it"
  suggests I have to use tailbond's button. In fact, turning on the Tailscale
  app's own switch would take the VPN slot anyway. The wording makes
  tailbond seem to block Tailscale. **Low.**

## Other

- 14-error-cert says "tap Try again later", but the button is labelled "Try
  again". **Low.**
- 30-settings: "1 app(s) use the phone's own connection" uses "app(s)"
  instead of proper singular/plural. **Low.**

## Summary

| # | issue | severity | screen |
|---|---|---|---|
| 1 | The phone has no tailnet identity (10.77.0.2, no 100.x or MagicDNS name). Nothing says peers can't reach it or that ACLs see it as homebox, and the copy button copies a useless IP | medium | 10-home-connected |
| 2 | No word on whether subnet routes from other routers work, and no exit-node choice beyond home or none, so I can't tell what is routed where | medium | 30-settings, 10-home-connected-full |
| 3 | After switching back, the status card says "Tailscale stays off too until you switch back to it" while the card below says "Tailscale is in charge" | medium | 12b-home-switched-back-to-tailscale |
| 4 | "Switch back" doesn't bring Tailscale up ("If Tailscale isn't connected yet, open it..."), and Settings promises it "opens the Tailscale app" but 12b stays inside tailbond | medium | 12b-home-switched-back-to-tailscale, 30-settings |
| 5 | Pairing never mentions that all internet traffic will leave from home; the exit-node-like option is on by default | low | 03-deeplink-confirm, 30-settings |
| 6 | The mobile link "Not working right now" gives no reason or next step | low | 15-one-path-failed |
| 7 | The mobile card's "Use for tailbond" switch vs the "Save mobile data" mode isn't explained; "Advanced: link roles" is unexplained jargon | low | 21-connections-save-data |
| 8 | "Tailscale stays off too until you switch back to it" suggests the Tailscale app's own switch won't work | low | 12-home-disconnected |
| 9 | "tap Try again later" doesn't match the "Try again" button | low | 14-error-cert |
| 10 | "Opening tailbond···" middle-dot glyphs; "1 app(s)" plural | low | web-02-after-tap, 30-settings |
