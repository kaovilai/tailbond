# Round 2: Persona A, "GL traveller"

Persona: consultant, travels monthly with a GL.iNet Slate 7, at home in GL's
admin panel (Repeater, VPN Dashboard), not a network engineer. Wants hotels to
just work, cares about banking and video calls, must not leak work traffic.

Screens reviewed: all 21 PNGs in `out/ux/bypass-failover/glinet/` and all 25 in
`out/ux/bypass-failover/android/`, with the two manifests.

---

## Task 1: Make `mybank.com` skip the tunnel (router, then phone)

**Done?** Router: yes. Phone: partly, because I can't be sure the bank's login
page is covered.
Screens: bypass-empty, bypass-add-website-presets, bypass-rules(-phone),
bypass-website-rules-wait, 46c, 47, 47b, 46, 46-dark, 46b, 48.

On the router I clicked "Banking", typed mybank.com and left "Use the internet
directly". The row then shows "Uses the internet directly · Bank · 37
connections", so I can see it working. On the phone the same steps give a row
that says it "Covers mybank.com and www.mybank.com" and nothing else.

### Findings

- **high** · `47-add-website-dialog.png`, `47b-add-website-preset-banking.png`, `46-websites-skip-tailbond.png` vs `bypass-add-website-presets-desktop.png`.
  The router includes every sub-name ("www.mybank.com and login.mybank.com"), but the phone covers only the name and its www. name and says to add login.mybank.com yourself. I don't know which names my bank uses for login, so on the phone the sign-in will probably still go through the VPN and get blocked. The phone list also gives me no way to see that this is happening.
  *Suggestion:* make the phone match the router and include all sub-names. If that's impossible, show a per-website status or "used N times today" on the phone like the router's "37 connections", so I can see when the bank isn't covered.

- **medium** · `bypass-website-rules-wait-desktop.png`.
  The notice says "these websites keep going through tailbond as before; nothing breaks". Each row underneath still says in green "Uses the internet directly", with only a small "Waiting" badge. For my bank, going through tailbond *is* the breakage: it still blocks me.
  *Suggestion:* while a rule is waiting, show the row as "Not active yet: still goes through tailbond" in amber, and drop "nothing breaks" from the notice, or change it to "nothing else changes".

- **medium** · `bypass-website-rules-wait-desktop.png`.
  The fix is "install dnsmasq-full", and if Plug-ins refuses, an SSH command that removes the router's DNS server (`opkg remove dnsmasq`). I've never used SSH, and a command that deletes DNS looks like it could take the whole router offline in a hotel.
  *Suggestion:* add a one-click "Install what website rules need" button that does this safely and rolls back on failure. If the SSH command must stay, say what happens if it fails halfway and how to recover.

- **medium** · `bypass-add-website-presets-desktop.png`, `bypass-add-wifi-calling-desktop.png`.
  "Uses the router's main internet connection" doesn't say which one. I have Repeater and Tethering both "In use", and Tethering has a 5 GB limit. If bypassed traffic (or the Fire TV's streaming) goes over my phone's data, I'd want to know.
  *Suggestion:* name the connection ("Uses Repeater (hotel Wi-Fi) directly, or Tethering if Repeater is down"), or add the existing "use only one of the router's connections" choice next to it with the connection names.

- **medium** · `bypass-rules-desktop.png`, `bypass-add-website-presets-desktop.png`.
  "Website rules can miss devices that use their own secure DNS (Private DNS on Android...)". My Android phone probably has Private DNS on. The fix offered ("use a device rule") would send *all* of that device's traffic outside the tunnel, which isn't what I want for a bank. Nothing tells me which of my devices are being missed.
  *Suggestion:* flag devices the router sees using their own DNS ("alex-phone uses its own DNS: website rules don't apply to it") and link to how to turn it off, instead of telling me to bypass the whole device.

- **medium** · Both apps (`bypass-rules-phone.png`, `46-websites-skip-tailbond.png`, `48-settings-skip-rows-android-12.png`).
  Nothing explains what happens when my phone runs tailbond *and* is on the Slate's Wi-Fi. Do I need the bank rule in both places? Does the router rule see the phone's bank traffic if the phone looks names up through its own tunnel? I'd add it in both places and still not know which one is doing the work.
  *Suggestion:* add one line on each side, e.g. on the phone: "On a tailbond router's Wi-Fi, the router's rules also apply; add the website there too." Better still, detect the router and say so.

- **low** · `46-websites-skip-tailbond.png`, `46c-websites-none-yet.png`.
  The bottom paragraph ("goes by each website's addresses... big hosting networks... private DNS") is too technical for me. Its one important point, that *other* sites on the same hosting network also skip tailbond, is buried at the end.
  *Suggestion:* cut it to "Other sites hosted alongside this one may also skip tailbond" and put the rest behind "How this works".

- **low** · `47d-add-website-preset-wifi-calling.png`.
  The Wi-Fi calling chip on the phone does nothing ("Nothing to add"). It also suggests bypassing Teams if calls break, and Teams is work traffic I must not leak.
  *Suggestion:* drop the chip from the phone dialog. If an apps hint stays, warn that a bypassed work app leaves through the hotel network.

---

## Task 2: Make the work laptop always use tailbond

**Done?** Yes (router).
Screens: bypass-add-device-picker(-desktop/-phone), bypass-rules(-phone),
servers-home-active-vps-standby.

"A device" → work-laptop → "Always use tailbond". The help text says it beats
website and port rules and waits rather than going around when tailbond is
down. That is what I needed.

### Findings

- **medium** · `bypass-add-device-picker-desktop.png`, `bypass-rules-desktop.png`.
  The rule is tied to the laptop's MAC address. My work laptop is managed by IT, and I may not be allowed to turn off "Private Wi-Fi address". If the address rotates, the rule silently stops matching and the laptop becomes an "Unnamed device" that bank, streaming and Wi-Fi-calling rules can send out through the hotel. The row would just look "not connected", with no warning.
  *Suggestion:* warn on the rule row when its device hasn't been seen since a new unknown device appeared ("work-laptop not seen since 09:12; a new device joined at 09:13: is it the same laptop?"). Also offer a router-wide "only listed devices may skip tailbond" switch.

- **medium** · `bypass-rules-desktop.png`.
  The promise that matters most ("even websites or ports set to skip tailbond... waits rather than going around it") appears only in the add dialog. The row just says "Always uses tailbond", and there's no edit button to bring that text back.
  *Suggestion:* add the guarantee to the row ("Always uses tailbond, never the hotel's network, even if tailbond is down"), and make rows clickable to edit.

- **low** · `bypass-rules-desktop.png`.
  The same page shows "work-laptop · connected now" in Bypass and "work-laptop · Last seen 1 day ago" under Tailnet machines. As a user I can't tell these are two different things (the laptop on the Wi-Fi vs its Tailscale entry).
  *Suggestion:* label the Bypass one "on this router's Wi-Fi", or merge or explain the two.

- **low** · `bypass-add-device-picker-phone.png`.
  On a phone-width browser, the device list scrolls inside the dialog, which scrolls too. The Cancel/Add buttons are cut off at the bottom.
  *Suggestion:* use a full-screen sheet on narrow screens with buttons fixed at the bottom.

---

## Task 3: Add the VPS as a backup to `homebox` (router and phone)

**Done?** Partly. Adding it is clear on both. The clash can only be fixed on
the VPS with a terminal.
Screens: servers-single-home, add-server-open, add-server-waiting-approval,
add-server-subnet-clash, add-server-paste-clash,
servers-home-active-vps-standby, 44b, 45, 45b, 45c, 44, 44-dark.

### Findings

- **medium** · `add-server-subnet-clash-desktop.png`, `add-server-paste-clash-desktop.png`, `45c-add-another-server-same-addresses.png`.
  The fix for the clash is to SSH into the VPS and rerun the installer or edit `/etc/tailbond/tailbond.json`. The router lists three methods; the phone gives only the JSON edit. The phone also says "Asking over Tailscale does this by itself while vps has no other devices", which leaves me wondering why it then failed and what I did wrong.
  *Suggestion:* have the new server pick a free range by itself when asked to join as a backup, or add a "Fix it on vps" button that asks vps to change (vps already trusts this router after approval). Use the same wording on both apps, starting with the installer command since that's how I set the VPS up.

- **low** · `add-server-open-desktop.png`.
  The button says "Add a backup server" but opens a section titled "Add another server". vps is "Found on your tailnet" but isn't in the Tailnet machines list further down.
  *Suggestion:* use "Add a backup server" everywhere. List vps (and homebox) under Tailnet machines, or explain why they aren't there.

- **low** · `add-server-waiting-approval-desktop.png`.
  It doesn't say how long the request stays open or what to do if nobody approves. I'd need to know that the vps page (`:8077`) only opens while tailbond or Tailscale is up.
  *Suggestion:* add "This request expires in 10 min" and a "Didn't get it? Use a pairing link instead" fallback.

- **low** · `45-add-another-server.png`.
  The phone shows four ways to add a server and makes me type the name. The router just found vps on the tailnet. The page is titled "Add another server" with a heading "Add a backup server" under it.
  *Suggestion:* show "Found on your tailnet: vps · Ask to connect" first, like the router, fold the other methods under "Other ways", and use one title.

- **low** · `servers-home-active-vps-standby-desktop.png`, `44-settings-servers.png`.
  The two apps use different words for the same states: router "First choice / Backup / Standby / Can't reach", phone "Main / Backup 1 / Ready as backup / Unreachable". The "tunnel address... approve each server's route in Tailscale" text means nothing to me.
  *Suggestion:* use the same labels on both apps, and move route approval under "Advanced", since a traveller doesn't need it.

---

## Task 4: Notice the switch to the VPS, and the return to homebox

**Done?** Mostly yes while watching; partly after the fact.
Screens: failover-using-vps-home-unreachable(-desktop/-phone),
failback-countdown, failover-you-picked-vps, back-on-homebox-since, 40,
40-dark, 41, 41b, 41c, 42, 43, 43b, 44c, 49.

"Using vps since 14:12 (homebox is unreachable)", "Back to homebox in 1:20",
and "Back on homebox since 14:11" answer what happened, why, and when, on both
apps.

### Findings

- **medium** · `back-on-homebox-since-desktop.png`, `43-home-back-on-homebox.png`.
  Once it's back, the only trace is "Back on homebox since 14:11". There's no record that it switched, when, why, or for how long. If I open the panel the next morning, I can't tell whether my traffic left from a rented server overnight.
  *Suggestion:* add a short "Recent switches" list to the Servers card ("14:02–14:20 on vps: homebox stopped answering").

- **medium** · `failover-you-picked-vps-desktop.png`, `failback-countdown-desktop.png`, `42-home-back-to-homebox-in-1-20.png`.
  After "Use now" the header says "Stays on vps until you pick homebox", but "When to switch" still has "Prefer homebox, use vps if homebox is down" selected, so the page contradicts itself. Likewise, I can't tell whether "Stay on vps" during the countdown is permanent or just for now.
  *Suggestion:* while a manual pick is active, mark it on the radio group ("Paused: you picked vps at 14:04 · Resume automatic"). Label the button "Stay on vps for now".

- **medium** · `failback-countdown-desktop.png`.
  The router doesn't warn that going back will interrupt calls. The phone does ("Calls may pause for a few seconds then"). On a video call, a 1:20 countdown with no warning is exactly when I'd press "Stay on vps".
  *Suggestion:* copy the phone's line to the router header, e.g. "Back to homebox in 1:20. Calls may pause a few seconds · Stay on vps".

- **medium** · `43b-home-switching-to-vps.png`, `49-home-error-no-server-reachable.png`, `48-settings-skip-rows-android-12.png`.
  On the phone, while switching ("apps use the phone's own connection") and when both servers are down ("Your phone is using its normal internet now"), work traffic goes out through the hotel. Settings tells me to leave "Block connections without VPN" *off*. The router promises "nothing goes out unprotected", so the two products behave opposite ways.
  *Suggestion:* add a phone setting like "Never send traffic outside tailbond (wait instead)" that matches the router. Say in the "Keep tailbond on" section that leaving Block off means brief leaks during switches.

- **medium** · `41-home-using-vps-homebox-unreachable.png`, `42-home-back-to-homebox-in-1-20.png`.
  Right under "It goes back to homebox by itself", a full-width "Switch back to Tailscale" button looks like the way to switch back. Tapping it turns tailbond off completely.
  *Suggestion:* rename it "Turn off tailbond and use Tailscale", and move it lower or into the menu.

- **low** · `failover-using-vps-home-unreachable-desktop.png`, `41b-home-using-vps-first-screen.png`.
  On the router the header icon stays green, and on the phone the card stays the same blue "Connected". Nothing visual changes when it moves to the backup, so I'd only notice by reading.
  *Suggestion:* use an amber accent or a "Backup" badge on the status card while not on the first-choice server.

- **low** · Headers on all failover screens.
  "since 14:12" has no date, which is ambiguous after a night or when I'm across time zones.
  *Suggestion:* add the day when it isn't today ("since yesterday 23:40").

- **low** · `49-home-error-no-server-reachable.png`.
  When nothing is connected, the top title still reads "homebox · via home server", and both server rows show only grey "Main server" and "Backup".
  *Suggestion:* show "Not connected" in the title and "Didn't answer" on each server row.

- **low** · `44-settings-servers-dark.png`.
  In dark mode the "Unreachable" dot is pale pink and barely reads as an error.
  *Suggestion:* use a stronger red in the dark theme.

- **low** · `bypass-wifi-calling-added-desktop.png`.
  The "Wi-Fi calling: uses the internet directly" toast appears over the Home server card, far from the Bypass card I was using.
  *Suggestion:* show the confirmation near the new row, or briefly highlight the row.

---

## Overall count

- **high: 1**
- **medium: 13**
- **low: 13**
