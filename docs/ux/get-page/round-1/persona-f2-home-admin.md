# Round 1: persona F2, home admin (Mac + Android)

I went through both tasks as F2, using only the task script, the screen list and the 9 screenshots.

## Task 1: Install tailbond on my Mac

**Steps:** I open homebox/ in Safari and the Mac tab is already picked. I read "From homebox (simplest)", open Terminal from Applications → Utilities, press Copy on the first command, paste it and press Return. I type my password. Then I go back to Safari and tap "Open tailbond and ask homebox". tailbond's page opens with homebox filled in, and I tap "Ask homebox".

**Where I hesitate:**
- **Medium.** "Apple silicon Macs (M1 and later)": I don't know what chip my Mac has, and the page doesn't say what to do if it's an older Mac. *(web-05-get-mac-desktop-first-screen)* Suggestion: add "Check: Apple menu → About This Mac → Chip" and say what owners of Intel Macs should do.
- **Medium.** It says I'll need "your Mac's password (sudo)" but not that nothing shows on screen while I type it. I'd think Terminal had frozen. *(same)* Suggestion: "Type your Mac login password. Nothing appears as you type; press Return."
- **Medium.** The Gatekeeper paragraph ("no Apple Developer ID… clears the download's quarantine flag") reads like "this gets around Apple's safety check". This is where I'd stop and call my nephew. *(same)* Suggestion: put it in plain words, e.g. "This comes from your own homebox, so the installer tells macOS to trust it." Move the technical detail behind a "Details" link.
- **Medium.** Nothing tells me what Terminal should show when the install worked, so I don't know when to move on to step 2. *(web-05-get-mac-desktop)* Suggestion: "When Terminal says 'tailbond is running', go to step 2."
- **Medium.** "Is homebox yours? Approve it yourself at http://homebox:8077": someone else set homebox up for me, so I can't tell if it's "mine". Once I'm on tailbond's own page, that link is no longer in front of me. *(web-05-get-mac-desktop, web-07)* Suggestion: "If nobody else looks after homebox, open homebox:8077 and tap Approve." Also show that link on tailbond's own page while it waits.
- **Medium.** tailbond's own page shows "Disconnected", a switch, "BPS DOWN", "TUNNEL IP" and four modes. I'd be tempted to flip the switch. I can't see what happens after I tap "Ask homebox". *(web-07)* Suggestion: when a request is pending, show a clear "Waiting for homebox to approve…" banner and grey out the switch.
- **Low.** The page opens scrolled down to the tabs, so I never see the "Get tailbond" heading. *(web-05-get-mac-desktop-first-screen)*
- **Low.** The second command, the SHA-256 string and the whole Homebrew box are clutter for me. *(web-05-get-mac-desktop)* Suggestion: collapse them under "Other ways".

**Verdict:** I'd most likely get it installed, but the security wording and the unclear approval step could make me stop and phone someone.

## Task 2: From the Mac, get the app onto my Android phone

**Steps:** On the Mac I tap the "Android" tab and see a QR code with "point the phone's camera at this code". On the phone I turn Tailscale on and scan it. The phone page opens on the Android tab. I look at the five pictures, tap "Download tailbond", go through the Android prompts, then tap "Open tailbond and ask homebox".

**Where I hesitate:**
- **Medium.** On the Mac, the full phone instructions appear right under the QR code ("Look at these five pictures first…", and presumably the download button further down). I might think these are for the Mac and click Download, which would put the phone app file on the Mac. *(web-06-mac-picks-android)* Suggestion: on a computer, show only the QR code plus "The steps continue on your phone", or collapse the phone steps.
- **Medium.** The "Android tab on a computer" capture says "On this phone" and has no QR code. On the Mac I'd be stuck there, since there's nothing to scan. *(web-05-get-android-desktop, compare with web-06)* Suggestion: always show the QR code and say "On an Android phone" whenever the page is open on a computer.
- **Low, good.** The "File might be harmful" and "Allow from this source" drawings warn me in advance, and so does "you can turn it off afterwards". That's exactly what someone wary like me needs. *(web-05-get-android-phone)*
- **Medium.** Step 3 says "Tailscale's VPN is off on the phone… like using it as an exit node". That scares me that I've broken Tailscale, and I don't understand "exit node". *(web-05-get-android-phone)* Suggestion: lead with "Your Tailscale things keep working; tailbond takes over while it's on", and drop "exit node".
- **Low.** The phone page is very long. The download button is several screens down, and the "Is this safe?" paragraph uses jargon (http://, SHA-256, sha256sum). *(web-05-get-android-phone)*
- **Medium.** The same "Is homebox yours?" approval confusion as on the Mac. *(web-05-get-android-phone)*

**Verdict:** The QR handoff and the picture guide would most likely get the app installed, but the phone steps showing on the Mac and the unclear approval step put me at real risk of a wrong turn.
