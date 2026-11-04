# Round 1: persona A, daily Tailscale user (Mac + Pixel)

**Persona A: daily Tailscale user, MacBook (Apple silicon) and Pixel, has Homebrew**

## Task 1: Install tailbond on my Mac from homebox/

**Steps**
1. I open homebox/ in Safari. The Mac tab is already picked, which is good. The first screen opens scrolled down to the switcher, so I never see the "Get tailbond" title or the one-line explanation of what it is.
2. I have Homebrew, so I look at the "With Homebrew" box first. It's four commands, including `tap-new --no-git` and a curl to GitHub. I go back to "From homebox (simplest)", hit Copy and paste it into Terminal. I type my sudo password.
3. I come back to Safari and click "Open tailbond and ask homebox". tailbond's own page opens with homebox.tail1234.ts.net already filled in, and I click "Ask homebox".
4. homebox is mine, so I open http://homebox:8077 and approve the Mac there.
5. Back on tailbond's page. Is it connected now, or do I flip the "Disconnected" switch?

**Where I hesitate**
- At the Homebrew box: why four commands, and why from GitHub?
- After running the command: did it work?
- At the "Disconnected" switch.

**Findings**
- **Medium** (web-05-get-mac-desktop): The "simplest" command is curl over `http://` with `sudo`. The Android tab explains why http is safe here (Tailscale encrypts it); the Mac tab doesn't. The SHA-256 is shown but there's no command to check it. *Suggestion:* add the same one-line "Is this safe?" note and a `shasum -a 256 …` line I can copy.
- **Medium** (web-05-get-mac-desktop): The Homebrew route says "both install the same thing", but it pulls the formula from GitHub, not homebox. It's four commands, and `sudo brew services` looks wrong to someone who uses brew. *Suggestion:* make it one line (a real tap, or `brew install <url>`) and say where it downloads from.
- **Medium** (web-05-get-mac-desktop, web-07-mac-dashboard-ask-prefilled): Nothing tells me what success looks like after the install command. After approving, the page says "the Mac connects right after", but tailbond's page still shows "Disconnected" with a switch. *Suggestion:* add "You'll see 'tailbond is running'" after step 1, and in step 2 say whether I need to flip the switch.
- **Low** (web-05-get-mac-desktop): Step 2's main button is far below the fold, under the Gatekeeper paragraph. *Suggestion:* collapse the Gatekeeper text behind a "Why no double-click?" toggle.
- **Low** (web-05-get-mac-desktop-first-screen): The first screen starts at the switcher with no title or one-line explanation. *Suggestion:* scroll so the title stays visible.
- **Low** (web-05-get-mac-desktop): The Mac tab doesn't say whether tailbond on the Mac takes over from Tailscale, as the Android tab says it does on the phone. *Suggestion:* add one line.

**Verdict:** I'd succeed in about 5 minutes. The trust note and the "it worked" signal are missing.

## Task 2: Get the app onto my Android phone, starting from the Mac

**Steps**
1. I click Android. A QR code shows at the top with "point the phone's camera at this code" (web-06-mac-picks-android). Clear.
2. I scan it on the Pixel. The phone's first screen is the title, the switcher and a box saying "Look at these five pictures first". There's no download button anywhere on it (web-00a-phone-page-first-screen).
3. I scroll past five drawings to "Download tailbond (27 MB)", then go through Download anyway → Settings → Allow from this source → Install.
4. I come back, tap "Open tailbond and ask homebox", approve at homebox:8077, then tap Add phone and flip the switch.

**Where I hesitate**
- On the phone's first screen: where's the button?
- At step 3, when I find out Tailscale turns off.

**Findings**
- **Medium** (web-05-get-android-phone, web-00a-phone-page-first-screen): The download button is about 2.5 screens down on the phone. I skim, so I'd scroll right past the drawings and hit "File might be harmful" without having read them. *Suggestion:* put the "In short" line and the download button at the top, with the drawings below it or collapsible ("Show me each screen").
- **Medium** (web-05-get-android-desktop, step 3): Two things a daily Tailscale user cares about most are buried there:
  - "Tailscale's VPN is off on the phone" while tailbond is on.
  - By default, internet traffic goes out through homebox, which is in small grey text.

  *Suggestion:* state both in one line before I install.
- **Low** (web-05-get-android-desktop): I can't tell whether a sideloaded APK updates itself. Obtainium is hidden in a collapsed section. *Suggestion:* add one line on how updates arrive.
- **Low** (web-06-mac-picks-android): Below the QR code, the Mac shows the full set of phone steps, which aren't useful on the Mac. *Suggestion:* collapse them on desktop under "Steps on the phone".
- **Low** (web-05-get-linux-desktop compared with the Mac tab): Linux downloads over https from GitHub while the Mac downloads over http from homebox. The inconsistency makes me wonder which source is the trusted one. *Suggestion:* add one sentence on where each download comes from.

**Verdict:** The QR handoff is excellent, but the phone page makes me dig for the button and hides the Tailscale-off and exit-node caveats until after I've installed.
