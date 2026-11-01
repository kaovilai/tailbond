# Round 4 – Persona (c): less-technical family member

Retired teacher, 60s. My son set up Tailscale. I flip its big switch and copy
"nas" from the machines list for my photo app. Red text and settings I can't
explain make me nervous, so I'd rather leave them alone. I only judged what
the screenshots in `out/ux/after-4/` show.

## Task 1: Install tailbond and pair it, from the phone

**Tap path:** I'd start with tailbond's own first screen, because that's what I'd
have after someone told me to install it (`01-onboarding`): "Get started" → `02-pairing`.
Or I'd start with the home server page in my browser (`web-01-add-this-phone`): "Download tailbond for
Android" → install → back to the page → "+ Add this phone" → `web-02-after-tap` "Open in
tailbond" → `03-deeplink-confirm` "Add phone".

**Stuck points:**
- **I can't find the home server page on my own.** `01-onboarding` says "You'll need your home
  server's name or the link to its page. If someone else set it up, ask them for it."
  `02-pairing` asks for "Home server's Tailscale name" and says "It's the machine that runs
  tailbond, listed in the Tailscale app's Machines. Not sure which one?" My Machines list has
  homebox, nas, apple-tv and others, and nothing marks which one "runs tailbond". I'd guess
  "homebox" or ring my son. The bold line "Came here from your home server's page? Go back to it
  in your browser" doesn't help me, because I never had that page open.
  *Expected:* Tailscale needed only a sign-in button. A single "Find my home server" button that
  works it out from Tailscale, or a list of machines to pick from, would suit me.
  **Severity: medium.**
- **The install warning would stop me.** `web-01` says "Android warns about apps from outside
  the Play Store and asks you to allow installs from your browser … Unsure? Ask whoever set up
  the server to install it for you." That's honest, but I would stop when the Android warning
  appeared and phone my son. I got Tailscale from the Play Store, so this would be my first time
  installing an app this way. **Severity: medium.**
- **One line in the confirmation is jargon.** `03-deeplink-confirm` says "It connects over the
  internet to home.example.net:443, and your internet traffic goes out from home (you can change
  that in Settings)." I don't know what ":443" means, or whether "traffic goes out from home" is
  a good thing. The rest of the dialog is clear: "Add this phone to homebox?" and "Secure
  connection to homebox". **Severity: low.**
- **The error appears in red.** `02-pairing-paste-error` shows "That isn't a complete tailbond
  link. Copy the whole link starting with tailbond://pair". The wording is fine, but red text
  makes me think I broke something. **Severity: low.**

## Task 2: Connect

**Tap path:** I'd tap the big switch beside "Disconnected" (`12-home-disconnected`) →
`04-handoff-other-vpn` "Switch from Tailscale to tailbond?" → "Turn on tailbond" → Android's
"OK" → `11-home-connecting` → `10-home-connected`.

When everything works, this is fine. It feels like the Tailscale switch, and the dialog tells me
in advance to "Tap OK" and how to go back later.

**Stuck points (when it fails):**
- **The error tells me about router settings.** `13-error-unreachable` shows a red box, "Can't
  reach your home server", followed by "the home router may need to forward UDP port 443 to it".
  I can't act on that. "show them this message" helps, but the red box plus words I don't know
  would worry me. **Severity: low.**
- **The second error sounds like a break-in.** `14-error-cert` shows a red box: "Can't confirm
  this is your home server … the server answering isn't the homebox this phone was paired with."
  That reads to me like someone is pretending to be our server. The next sentence, "they can add
  this phone again (Settings → Pair again)", gives me a path to try myself. I might tap "Pair
  again" without understanding what it does, even though the text means my son should do it.
  *Expected:* a calmer message with one safe action for me ("Try again" / "Switch back to
  Tailscale"), and the re-pair steps only for the person who set it up. **Severity: medium.**

## Task 3: Find and copy the NAS address

**Tap path:** On `10-home-connected` I'd scroll down to "Machines", type "nas" in "Search
machines" and tap the copy icon (`18-machines-search-copied` → "Copied 100.101.2.2"). The other
way is to tap "nas" → `19-machine-detail` → copy "nas.tail1234.ts.net".

**Stuck points:**
- **The machines list is far down the screen.** In Tailscale it's the main thing on the screen.
  Here I scroll past Connections, the Wi-Fi/Mobile data boxes, "Save mobile data" and "This
  device" before reaching it (`10-home-connected-full`). **Severity: low.**
- **Another copy button comes first.** The first copy icon I meet is under "This device" next to
  "pixel-8 (this phone) 10.77.0.2". I could copy that by mistake. It also looks different from
  the 100.x numbers I'm used to from Tailscale. **Severity: low.**
- **The detail screen shows four addresses.** `19-machine-detail` lists "MagicDNS name",
  "Short name", "Tailscale IPv4" and "Tailscale IPv6", and I don't know which one the photo app
  wants. Tailscale's own app has the same problem, so this isn't worse. **Severity: low.**

## Task 4: Are Wi-Fi and 5G both being used?

**Tap path:** None needed. `10-home-connected` shows "Using both Wi-Fi and mobile data" with a
green "In use" on both boxes. `15-one-path-failed` clearly shows "Using Wi-Fi only" and a red
"Not working right now". I have no problems to report.

## Task 5: Make it save cellular data

**Tap path:** I'd turn on the "Save mobile data" switch on the home screen (`10-home-connected`).
`16-save-data-standby` confirms it with "Using Wi-Fi · saving mobile data (backup only)" and
"Not in use · saving data". That's easy.

**Stuck points:**
- **"More options" gives me several data settings that overlap.** If I open "More options"
  (`20-connections`, `21-connections-save-data`), I find:
  - a "Use for tailbond" switch on Mobile data
  - four modes
  - "Faster switch to backup (experimental) … Uses a little mobile data"
  - a "Mobile data limit" box already filled in with "5"
  - "Separate from Save mobile data: …"

  That's three or four ways to "save data", and I couldn't tell which one I'm supposed to use.
  I'd back out without touching anything. The home-screen switch is enough, so it doesn't stop
  me. **Severity: low.**

## Task 6: Turn tailbond off and go back to Tailscale

**Tap path:** I'd tap "Switch back to Tailscale", which sits right under the big switch
(`10-home-connected`) → `12b-home-switched-back-to-tailscale` → "Open Tailscale" → turn on the
Tailscale switch if it says Disconnected.

**Stuck points:**
- **After switching back, the screen contradicts itself.** `12b` says "Tailscale is in charge
  on this phone", but further down the same screen still says "Will use Wi-Fi and mobile data
  together", both boxes say "Ready", and "This phone reaches your tailnet through homebox". I'd
  wonder whether tailbond is still doing something. **Severity: low.**
- **Turning the switch off leaves me without Tailscale.** If I use the big switch instead of the
  button, `12-home-disconnected` leaves me with neither app connected. The text "or switch back
  to Tailscale below" does point me the right way. **Severity: low.**

## Summary

| # | issue | severity | screen |
|---|---|---|---|
| 1 | I can't find the home server's name or page without help. "Home server's Tailscale name" says to look in Tailscale's Machines list, but nothing there marks which machine "runs tailbond" | medium | 01-onboarding, 02-pairing |
| 2 | Installing from outside the Play Store: I'd stop at Android's warning and phone my son ("Unsure? Ask whoever set up the server…") | medium | web-01-add-this-phone |
| 3 | The cert error reads like someone is impersonating our server, and "(Settings → Pair again)" invites me to re-pair even though it's meant for my son | medium | 14-error-cert |
| 4 | "home.example.net:443" and "your internet traffic goes out from home" in the confirmation dialog are jargon | low | 03-deeplink-confirm |
| 5 | The paste error is in red, which feels alarming even though the wording is fine | low | 02-pairing-paste-error |
| 6 | A red error box mentions "forward UDP port 443", which I can't act on | low | 13-error-unreachable |
| 7 | The Machines list (the part I use most) is below four other sections | low | 10-home-connected-full |
| 8 | The "This device" copy icon (10.77.0.2) comes before the machines list, so I could copy the wrong address | low | 10-home-connected, 18-machines-search-copied |
| 9 | "More options" has several overlapping data settings: Use for tailbond, modes, experimental backup, and a monthly limit already set to 5 GB | low | 20-connections, 21-connections-save-data |
| 10 | After switching back, the screen still says "Will use Wi-Fi and mobile data together", "Ready" and "reaches your tailnet through homebox" | low | 12b-home-switched-back-to-tailscale |
