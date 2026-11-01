# Round 3 — Persona (a): daily Tailscale Android user, product designer

Screens: `out/ux/after-3/`. Judged only from the screenshots.

## Task 1 — Install tailbond and pair it, from the phone

**Tap path:** Browser on the server page (web-01) → "Download tailbond for Android" → install → back to page → name field shows "pixel-8" → "+ Add this phone" (web-01) → "Open in tailbond" (web-02) → app dialog "Add this phone to homebox?" → "Add phone" (03-deeplink-confirm).
If I opened the app first: 01-onboarding "Get started" → 02-pairing "On this phone (easiest)" → type the server name → "Open home server page".

**Stuck points:**
- 02-pairing: the field says "Home server's Tailscale name". I don't know if that means `homebox` or `homebox.tail1234.ts.net`, and I have to go to the Tailscale app's Machines list to find it ("listed in the Tailscale app's Machines"). It's workable, but you have to leave the app to look it up. **Low.**

Otherwise this went smoothly. The web page explains the install warning, and the app's confirm dialog clearly names the server and the phone.

## Task 2 — Connect

**Tap path:** Home (12-home-disconnected) → main switch → "Turn off Tailscale's VPN?" (04) or "Allow tailbond's VPN" (05) → "Turn on tailbond" → Android OK → 11-home-connecting → 10-home-connected.

**Stuck points:**
- 11-home-connecting: the header says "Connecting…" while the Machines section says "Not connected. This is the list from 20 min ago". The two messages disagree for a moment. **Low.**

## Task 3 — Find and copy the NAS's MagicDNS name or IP

**Tap path:** Home → "Search machines" → type `nas` → copy icon → toast "Copied 100.101.2.2" (18). Or tap the row → detail page (19) → MagicDNS name copy → "Copied nas.tail1234.ts.net".

**Stuck points:** none for the NAS. This works just like Tailscale.

**Related confusion (This device):** 10-home-connected shows my phone as "10.77.0.2" with a copy icon, then "On your tailnet through homebox (100.101.1.1)". In Tailscale, "This device" shows my phone's own 100.x address. Here I can't tell whether 100.101.1.1 is my address or homebox's. I would probably copy 10.77.0.2 to give to someone on the tailnet, and I suspect that wouldn't work. **Medium.**

## Task 4 — Are Wi-Fi and 5G both being used?

**Tap path:** Home → "Connections" → "Using both Wi-Fi and mobile data", with both tiles showing a green dot and "In use" (10). Failure state (15): "Using Wi-Fi only" and "Not working right now" in red. Both are clear.

No issues.

## Task 5 — Save cellular data

**Tap path:** Home → "Save mobile data" switch → "Using Wi-Fi · saving mobile data (backup only)" and "Not in use · saving data" (16). Easy to find on the home screen.

**Stuck points:**
- 20/21-connections ("More options") also has the following:
  - "Use for tailbond" switches on each connection
  - a "Save mobile data" mode radio
  - "Faster switch to backup (experimental)"
  - a separate "Mobile data limit"
  - "Advanced: link roles"

  I'm not sure which one actually saves data, or whether I should also turn off "Use for tailbond" on Mobile data. The home switch is enough, but this page makes me second-guess it. "Link roles" means nothing to me. **Low.**

## Task 6 — Turn tailbond off and go back to Tailscale

**Tap path:** Home → "Switch back to Tailscale" (or Settings → "Switch back to Tailscale") → 12b "Tailscale is in charge on this phone" → "Open Tailscale".

**Stuck points:**
- My instinct is to flip the main switch off, but 12-home-disconnected says "Tailscale stays off too until you switch back to it." The text tells me what to do, so I wouldn't get lost. It's still a different model from Tailscale, where turning the switch off is all you do. **Low.**
- 12b: "If Tailscale isn't connected yet, open it and turn on its switch." After pressing "Switch back", I still have to open Tailscale and turn it on by hand. **Low.**

## Other observations
- 30-settings: "Remove from this phone" is plain link text right next to "Pair again". It's a destructive action with no destructive styling. **Low.**
- 30-settings: "1 app(s) use the phone's own connection" reads awkwardly. **Low.**

## Summary

| # | issue | severity | screen |
|---|---|---|---|
| 1 | "This device" shows "10.77.0.2" (copyable) instead of a 100.x tailnet IP; unclear whether "(100.101.1.1)" is mine or homebox's, so I'd likely copy and share the wrong address | medium | 10-home-connected, 15, 16, 17 |
| 2 | "Home server's Tailscale name" field: unclear whether to type short or full name, and I must leave the app to look it up | low | 02-pairing |
| 3 | "Connecting…" header vs. "Not connected. This is the list from 20 min ago" in Machines | low | 11-home-connecting |
| 4 | More options page has five data-related controls ("Use for tailbond", mode radio, "Faster switch to backup (experimental)", "Mobile data limit", "Advanced: link roles"); makes me doubt the home switch is enough | low | 20-connections, 21-connections-save-data |
| 5 | Main switch off doesn't give Tailscale back ("Tailscale stays off too until you switch back to it"); differs from Tailscale's mental model, but it is explained | low | 12-home-disconnected |
| 6 | After "Switch back", Tailscale may still need turning on by hand ("If Tailscale isn't connected yet, open it and turn on its switch") | low | 12b-home-switched-back-to-tailscale |
| 7 | "Remove from this phone" is plain link styling next to "Pair again", with no destructive cue | low | 30-settings |
| 8 | "1 app(s)" wording | low | 30-settings |
