# Round 2: persona F2, home admin (Mac + Android)

**Task 1: Install tailbond on the Mac**

Steps: I open homebox/ in Safari and the Mac tab is already picked, which is good. I check About This Mac, which says Apple M2. I open Terminal, press Copy, paste, and press Return. I type my password and see nothing appear, but the page warned me about that. I wait for "installed", then click "Open tailbond and ask homebox". On the tailbond page I click "Ask homebox", then call my nephew so he can approve it.

Where I hesitate:
- **High**, web-05-get-mac-desktop-first-screen.png: The "Is this safe?" note says the installer "tells [macOS] this one is yours, so you won't see that warning." To me that reads like it is getting around my Mac's security, and I nearly stopped there. Suggestion: say it plainly, e.g. "tailbond comes from your own homebox, not the App Store, so macOS has nothing to warn you about." Move the SHA-256 line into the "Other ways" section.
- **Medium**, web-05-get-mac-desktop.png, step 1: There's no help if it fails. "When Terminal says tailbond ··· is installed" doesn't tell me what an error looks like or what to do if the password is refused. Suggestion: show the exact success line, plus "If you see red text or 'could not resolve', check Tailscale is on and paste again."
- **Medium**, web-07-mac-dashboard-ask-prefilled.png: The tailbond page is full of "0 BPS DOWN", "TUNNEL IP" and four mode cards (Fastest, Smooth calls, Most reliable, Save mobile data). I thought I had to choose one first. I also wanted to flip the "Disconnected" switch. Suggestion: until the Mac is paired, highlight only the Ask homebox box and hide or dim the other controls.
- **Medium**, web-05-get-mac-desktop.png, step 2: "If that's you (nobody else set it up)…" Someone did set it up, but I'm the one who runs things here, so I couldn't tell which applies to me. The screenshots also never show what the Mac displays while it waits for the yes. Suggestion: show a "Waiting for homebox to approve" state, and phrase it as "If you have homebox's password / set it up yourself…"
- **Low**: The page says "tap" on a Mac; it should say "click". "(like the NAS)" means nothing to me. "Homebrew" in the fold-out is a word I don't know; keeping it folded away is fine.

**Task 2: Get the app onto my Android phone, starting from the Mac**

Steps: I click "Android" (web-06-mac-picks-android.png). I see a QR code and point my phone's camera at it. "Nothing to download on this computer" is very reassuring. The phone opens the long page (web-05-get-android-phone.png), where I follow the pictures, tap Download, then "Open tailbond and ask homebox", wait for approval, and turn it on.

Where I hesitate:
- **Medium**, web-06-mac-picks-android.png and web-00a-phone-page-first-screen.png: "the phone's internet goes out from homebox until you turn tailbond off". I first read "goes out" as "stops working". Suggestion: "all the phone's internet travels through homebox while tailbond is on."
- **Medium**, web-05-get-android-phone.png: The phone page is extremely long. It has five warning pictures ("File might be harmful", "Download anyway", "Allow from this source", "Scan app") before the button. The drawings prepare me, but scrolling past that many warnings makes me nervous, and I could lose my place. Suggestion: start with one calm sentence like "Android will show 3–4 warnings; that's expected for apps not from the Play Store." Keep "Skip to the download button" near the top.
- **Low**, web-06-mac-picks-android.png: "point the phone's camera". On some Androids the camera doesn't read codes. Suggestion: add "or use Google Lens".
- **Low**: The Mac tab never mentions the phone, so I only found the Android tab by guessing. Suggestion: add a line at the end of the Mac steps: "Have an Android phone too? Click Android above."
- **Low**: "Obtainium" and the SHA-256 text on the phone page are jargon to me. Fold them away.

I couldn't make out the smallest text on the phone's full-page screenshot (it's shown scaled down). Those points are based on what was legible.

**Verdicts**
- Task 1: I'd get it installed, but the "you won't see that warning" sentence nearly scared me off, and I'd need to phone whoever approves.
- Task 2: Scanning the code from the Mac is easy. The pile of Android warnings is survivable thanks to the pictures, but I'd go slowly and nervously.
