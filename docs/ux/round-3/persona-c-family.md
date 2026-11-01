# Round 3 — Persona (c): less-technical family member

Retired teacher, 60s. Son set up Tailscale. I use its big switch and copy "nas" for the photo app.
I don't know IP/DNS/TLS/ports. Red text and settings I can't explain make me nervous.

Screens: `out/ux/after-3/`.

## Task 1 — Install tailbond and pair it

**Tap path:** son sends me the home server page → `web-01-add-this-phone`. Tap "Download tailbond for Android", install → back to browser → "+ Add this phone" (`web-02-after-tap`) → "Open in tailbond" → app shows "Add this phone to homebox?" (`03-deeplink-confirm`) → "Add phone".

**Stuck points:**
- `web-01`: "Android warns about apps from outside the Play Store and asks you to allow installs from your browser". Tailscale came from the Play Store. Here I'm told to push past a warning, and my son always tells me never to do that. The line "this app comes from your own home server" helps, but I'd still phone him first. **Medium.**
- `web-01` "Then come back to this page for step 2": once the app installs, Android offers "Open", and I'd tap it. That takes me to `01-onboarding` → `02-pairing`, which asks for "Home server's Tailscale name". I don't know that. The hint says "listed in the Tailscale app's Machines", but my list has homebox, nas, apple-tv… and nothing tells me which one "runs tailbond". I'd have to find the browser tab again myself. **Medium.**
- `03-deeplink-confirm`: "It connects over the internet to home.example.net:443." I don't know what this is or whether it's right, and nothing tells me how to check. I'd tap "Add phone" anyway. **Low.**
- `02-pairing-paste-error` (only if I pasted something): red text "That isn't a complete tailbond link. Copy the whole link starting with tailbond://pair". It says what to do, but red makes me think I broke something. **Low.**

**Expected (Tailscale-like):** Tailscale comes from the Play Store and has one "Log in" button. I'd like the app, after installing, to send me back to that page for step 2, or say plainly "go back to your browser".

## Task 2 — Connect

**Tap path:** big switch on the home screen → "Turn off Tailscale's VPN?" (`04-handoff-other-vpn`) → "Turn on tailbond" → Android OK → `10-home-connected`.

**Stuck points:**
- `04-handoff-other-vpn` title: "Turn off Tailscale's VPN?" My son told me to keep Tailscale on. A title asking me to turn it off makes me want to tap "Cancel". The reassurance ("You still reach your Tailscale machines, like your NAS") is in the second paragraph. I'd get there, but only after a moment of worry. **Low.**
- `13-error-unreachable`: a red box saying "the home router may need to forward UDP port 443 to it". It does say "if someone else set up your home server, show them this message" and offers "Try again", so I know what to do. Still a red panel of words I can't read. **Low.**
- `14-error-cert`: "Can't confirm this is your home server … If you're not sure, don't pair again: tap Try again later and tell whoever set it up." Next to that text is a "Pair again" button, the very thing it tells me not to tap. I might tap it because it's right there, and I'm never "sure". **Medium.**

**Expected:** Tailscale has one switch and, when something's wrong, one short line. A button I'm told not to press shouldn't sit next to the message.

## Task 3 — Find and copy the NAS address

**Tap path:** home screen → "Search machines" → type "nas" → copy icon → toast "Copied 100.101.2.2" (`18-machines-search-copied`). Or tap the row → `19-machine-detail` → copy "MagicDNS name".

**Stuck points:**
- `19-machine-detail` lists four things: "MagicDNS name", "Short name", "Tailscale IPv4", "Tailscale IPv6". My son only ever said "the nas address", and nothing says which one the photo app wants. The copy icon in the list picked the number for me, which is easier. Tailscale's own detail page is similar, so this isn't new. **Low.**
- `10-home-connected`: "This device · pixel-8 (this phone) · 10.77.0.2" has its own copy icon, right above the Machines list. In Tailscale my phone's own address starts with 100. Here it's 10.77…, and below it is "(100.101.1.1)", which is homebox's number, not mine. If I copied the wrong row, I'd paste my phone's address into the photo app. **Low.**

**Expected:** this matches Tailscale closely: search box, green dot, copy icon. It works.

## Task 4 — Are Wi-Fi and mobile data both being used?

**Tap path:** home screen, `10-home-connected`: "Using both Wi-Fi and mobile data", two green "In use" dots.

**Stuck points:** none. It's clear. In `15-one-path-failed`, the red dot "Not working right now" on Mobile data worried me a little, but the top still says "Connected", so I'd leave it.

## Task 5 — Save mobile data

**Tap path:** home screen "Save mobile data" switch → on (`16-save-data-standby`): "Using Wi-Fi · saving mobile data (backup only)".

**Stuck points:**
- If I tap "More options" (`20-connections`, `21-connections-save-data`), I see three more things that sound like saving data. There's a "Use for tailbond" switch on the Mobile data card. There's a "Mobile data limit" box with "5" and "1.23 GB of 5.00 GB used this month". And under "Save mobile data" there's "Faster switch to backup (experimental)", which says "Uses a little mobile data". The note "Separate from Save mobile data" tells me these are different things but not which one I want. Switching off "Use for tailbond" on mobile data sounds like "save data" too, and then I'd have no backup when Wi-Fi drops. I'd leave the page without touching anything. **Low** (the home-screen switch already did the job).

**Expected:** one switch, which the home screen has. Fine.

## Task 6 — Turn tailbond off and go back to Tailscale

**Tap path:** home screen "Switch back to Tailscale" → `12b-home-switched-back-to-tailscale` → "Open Tailscale" → flip Tailscale's switch.

**Stuck points:**
- `12b`: I just switched back, but the top card still says "Disconnected … Tailscale stays off too until you switch back to it." Right under it is "Tailscale is in charge on this phone". The two messages contradict each other, so I can't tell whether I did it right and might tap "Switch back" again (it's gone now) or go looking for it. **Medium.**
- `12-home-disconnected`: if I just flip the big switch off (what I do in Tailscale), it says "Tailscale stays off too until you switch back to it." So the switch I know doesn't bring Tailscale back, and I need a different button. The screen does say this, but it's not what my habit expects. **Low.**
- `30-settings` row "Switch back to Tailscale" says "Turns tailbond off and opens the Tailscale app". But `12b` stays in tailbond and gives me an "Open Tailscale" button, so the two places promise different things. **Low.**
- `30-settings`: "Remove from this phone" sits next to "Pair again" with no warning visible, and "Block connections without VPN" / "Always-on VPN" are words I don't know. I'd avoid this screen entirely. **Low.**

**Expected:** in Tailscale, off is off. Here I'd expect the top card on `12b` to agree with "Tailscale is in charge", e.g. "tailbond is off. Tailscale is in charge."

## Summary

| # | issue | severity | screen |
|---|---|---|---|
| 1 | Install text tells me to allow installs from my browser past Android's "outside the Play Store" warning; I'd stop and call my son | medium | web-01-add-this-phone |
| 2 | After install, "Open" lands on the app asking for "Home server's Tailscale name", which I don't know; the app doesn't send me back to the web page for step 2 | medium | web-01-add-this-phone, 02-pairing |
| 3 | Cert error says "If you're not sure, don't pair again" but puts a "Pair again" button right next to it | medium | 14-error-cert |
| 4 | After "Switch back to Tailscale", top card still says "Tailscale stays off too until you switch back to it", which contradicts "Tailscale is in charge on this phone" | medium | 12b-home-switched-back-to-tailscale |
| 5 | Confirm dialog shows "connects over the internet to home.example.net:443" with no way for me to tell whether it's right | low | 03-deeplink-confirm |
| 6 | Paste error shown in red text | low | 02-pairing-paste-error |
| 7 | Dialog title "Turn off Tailscale's VPN?" makes me want to Cancel; the reassurance comes second | low | 04-handoff-other-vpn |
| 8 | Unreachable error is a red block mentioning "forward UDP port 443" | low | 13-error-unreachable |
| 9 | Four address kinds with no hint which one an app wants | low | 19-machine-detail |
| 10 | "This device" copy row (10.77.0.2) sits above Machines; its "(100.101.1.1)" is homebox's number, easy to copy the wrong thing | low | 10-home-connected |
| 11 | More options shows three more data-saving-sounding controls (Use for tailbond, Mobile data limit, Faster switch to backup); turning mobile off would remove the backup | low | 20-connections, 21-connections-save-data |
| 12 | Flipping the big switch off leaves Tailscale off too, unlike my Tailscale habit | low | 12-home-disconnected |
| 13 | Settings says Switch back "opens the Tailscale app", but the home-screen result stays in tailbond with an "Open Tailscale" button | low | 30-settings, 12b-home-switched-back-to-tailscale |
| 14 | "Remove from this phone" next to "Pair again"; "Block connections without VPN" / "Always-on" jargon | low | 30-settings |
