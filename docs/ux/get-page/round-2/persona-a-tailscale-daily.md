# Round 2: persona A, daily Tailscale user (Mac + Pixel)

The Mac install would work on the first try, and so would getting tailbond onto the phone from the Mac. Most of what I hit was about trust and how much there is to read, not dead ends. The full-page phone screenshot was too shrunk to read closely, and there's no screenshot of what happens after "Ask homebox", so I couldn't judge those parts fully.

## Task 1: Install on my Mac

**Steps**
1. Safari, `homebox/`. The Mac tab is already picked. Good.
2. Copy the one-liner, paste it in Terminal, type my password.
3. Scroll down and tap "Open tailbond and ask homebox". tailbond's page opens on the Setup tab with homebox already filled in, so I tap "Ask homebox".
4. I'm the admin, so I open `http://homebox:8077` and tap Approve.

**Where I hesitate**
- At step 2, I have Homebrew and I'd rather use it than pipe a tarball into `sudo` over `http://`.
- At step 4, I don't know whether the Mac's page shows "waiting for approval".

**Findings**
- **Medium** (web-05-get-mac-desktop-first-screen): Homebrew is hidden in a collapsed "Other ways" link that isn't visible on the first screen. Brew users trust it more than `curl … && sudo ./install.sh`. Put a short "Homebrew: `brew install …`" line next to the one-liner.
- **Medium** (web-05-get-mac-desktop): "The installer tells [macOS] this one is yours, so you won't see that warning" reads like it's turning off a macOS safety check. It made me trust it less. Say exactly what it does, in plain words.
- **Medium** (web-05-get-mac-desktop, step 2): Approving is a separate trip to another page. If I'm the admin, the Mac's page should say "Waiting for homebox… Approve it here →".
- **Medium** (web-05-get-mac-desktop): The Android tab says the phone's internet goes out from homebox. The Mac tab never says whether the Mac's does. Add one line, since I travel and that affects my speed.
- **Low** (web-05-get-mac-desktop): The SHA-256 is shown but there's no command to check it. Add `shasum -a 256 tailbond-macos-arm64.tar.gz`.
- **Low** (web-05-get-mac-desktop vs web-05-get-linux-desktop): Linux downloads from GitHub over https while the Mac downloads from homebox over http. That's inconsistent and makes me wonder why.
- **Low** (web-07-mac-dashboard-ask-prefilled): The "Or paste a pairing link" box just under "Ask homebox" is a second path I don't need. Collapse it.
- **Low** (web-05-get-mac-desktop): The "Is this safe?" paragraph is a long block of text. Cut it to two lines.

**Verdict:** I'd finish it in about 2 minutes, a bit uneasy about trust, and I'd want Homebrew up front.

## Task 2: Get the app onto my Android phone from the Mac

**Steps**
1. Tap the Android tab. It shows a QR code, so I scan it with the Pixel camera.
2. On the phone, the page has "Skip to the download button" and a one-line "In short: Download anyway → Open → Settings → Allow → back → Install". That's exactly what I want.
3. Download the APK and get through the sideload prompts.
4. Tap "Open tailbond and ask homebox", approve on homebox, then turn on the big switch.

**Where I hesitate**
- At step 1, the "Good to know" box says tailbond takes over from Tailscale while it's on. I stop to work out what I lose.
- At step 3, sideloading.

**Findings**
- **Medium** (web-06-mac-picks-android): "Takes over from Tailscale" doesn't say what I give up while it's on, such as my own exit node or Taildrop. Add one line on what keeps working and what doesn't.
- **Medium** (web-05-get-android-phone): The phone page is very long: five drawings plus several paragraphs. The skip link and the "In short" line save it, but put "In short" directly above the download button too.
- **Low** (web-06-mac-picks-android): The QR code has no URL printed under it. Show the address it opens so I can check it and type it if the camera fails.
- **Low** (web-06-mac-picks-android): Obtainium (for updates) is mentioned only on the phone page. Mention it on the Mac's Android tab as well, since update handling matters for an app that isn't in the Play Store.
- **Low** (web-05-get-android-phone): The "Is this safe?" text with the SHA is fine, but nobody checks a hash on a phone. Shorten it.

**Verdict:** Fast and clear from the Mac. The sideloading part is well handled, and the only real pause is what "takes over from Tailscale" costs me.
