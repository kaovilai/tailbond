# Round 3: persona C, "family member"

Persona: a relative of the person who set tailbond up. Uses the Android app, and the travel router's GL admin page only when asked. Doesn't know what a server, package manager, checksum or "dnf" is. Afraid of breaking things or losing settings. On every screen asks "is it working?" and "do I need to do anything?"

Screens used:
- Android: 01-settings-updates-and-backup-rows, 02-settings-update-available-row, 03-home-update-available, 10-updates-up-to-date, 11-updates-available, 12-updates-check-failed, 13-updates-off, 19-updates-debug-build, 30-backup-page, 31-backup-save-dialog, 31b-backup-send-to-homebox-dialog, 32-backup-saved, 33-restore-type-passphrase, 33b-restore-wrong-passphrase, 34-restore-preview, 35-restore-done-with-undo, 36-backup-page-undo-available, 37-restore-a-server-backup (dark variants skipped)
- GL panel: r01-router-up-to-date-desktop, r02-router-update-available-desktop/-phone, r03-router-installing-desktop, r04-router-rolled-back-desktop, r05-router-automatic-desktop, r06-router-reinstalled-after-firmware-upgrade-desktop, r07-router-update-confirm-desktop, r09-router-backup-download-dialog-phone, r10-router-restore-preview-desktop, r11-router-restore-done-desktop, r13-router-whats-inside-desktop, s01-flint-devices-out-of-date-desktop/-phone, s02-flint-whats-inside-desktop
- Web dashboard: w01-homebox-dnf-update-available-desktop, w02-updated-devices-out-of-date-backups-desktop, w03-vps-update-available-desktop, w05-backup-download-dialog-phone, w06-restore-preview-new-vps-desktop/-phone, w07-restore-done-desktop, w10-vps-new-machine-desktop, w12-whats-inside-desktop (w04, w08, w09, w11, w13 and r08 skipped: their router or phone versions were already reviewed)

---

## Task 1: Make sure everything stays up to date

**Done?** Partly. On the phone I can see my own version and whether an update is out (01, 02, 03, 11). On the router page I can see its version once I scroll to the bottom (r01, r02). I can't do anything about the home server myself: its page gives me a terminal command (w01). And I can't tell from the phone whether the router or the server are out of date.

Screens: 01, 02, 03, 10, 11, 12, 13, 19, r01, r02, r03, r04, r05, r07, s01, w01, w02, w03.

**Findings**

1. **medium**, `11-updates-available.png`. Updating the phone means "Your browser downloads tailbond-0.2.0.apk. Open it and tap Update…", then allowing the browser to install apps. All of that is one grey paragraph under the button. I've been told never to allow "install unknown apps", and the steps happen outside the app, where this text isn't visible anymore. "apk" means nothing to me.
   *Suggestion:* split it into 3 numbered steps with short labels ("1. Download 2. Open the file 3. Tap Install, then Allow if Android asks"), drop the file name, and repeat the key step in the notification or toast after the download.

2. **medium**, `10-updates-up-to-date.png`. Under the "Automatic updates" heading, the only way to get updates by themselves is to install a different app (Obtainium) and "add tailbond in it by that address" (a GitHub URL). I can't do that alone. The section looks like a setting, but it's a set of instructions.
   *Suggestion:* say plainly "Automatic updates need a helper app. Ask whoever set up tailbond to do this once." Offer a "Copy setup link" or a one-tap Obtainium deep link instead of a GitHub address to type.

3. **medium**, `w01-homebox-dnf-update-available-desktop.png`. The home server's update is "In a terminal on this server, run: sudo dnf upgrade tailbond-server". The "Automatic updates" line describes Fedora's default ("dnf-automatic only downloads updates… set apply_updates = yes in /etc/dnf/automatic.conf") rather than what this machine actually does, so I can't tell whether it will update itself. "by whoever looks after it" doesn't say who that is.
   *Suggestion:* state this machine's actual setting first ("This server does NOT install updates by itself" or "installs updates by itself every night"). Put the command and config detail behind "For the person who looks after this computer". When an update is waiting, show a short "Nothing for you to do here; tell <owner>" line.

4. **medium**, `03-home-update-available.png` / `01-settings-updates-and-backup-rows.png`. The phone only shows the phone's own version. Nothing tells me whether homebox or the travel router are up to date. That overview (s01, w02 "Your devices: 2 of 3 out of date") is only on the home server's page, which I never open.
   *Suggestion:* add a one-line summary on the phone's Updates page ("homebox 0.2.0 up to date · slate7 0.1.0, update in its router page"), or at least a line saying where to check the others.

5. **medium**, `r02-router-update-available-desktop.png`. The router says "Update available: tailbond 0.2.0" and gives me an "Update now" button. It doesn't say whether I should press it, whether it's safe to ignore, or whether it can wait until I'm home. The Tell me / Automatic / Off choice has no recommendation for someone like me.
   *Suggestion:* add one sentence: "Safe to install now; takes about a minute; your settings stay." or "You can leave this; nothing is broken." Mark one policy as "Recommended".

6. **medium**, `r03-router-installing-desktop.png`. "Installing 0.2.0… this page comes back by itself in about a minute" doesn't tell me whether I may close the page, unplug the router or move it. That's exactly what a worried person would want to do during an install at a hotel.
   *Suggestion:* add "Don't unplug the router until this says Up to date. Closing this page is fine."

7. **medium**, `19-updates-debug-build.png`. The fix is "Back up first… Then uninstall tailbond, install the release APK, and restore the backup." Uninstalling is the scariest step there is, and nothing checks that the backup was really made first. "Debug build", "signed with a different key" and "release APK" are jargon.
   *Suggestion:* lead with "This copy of tailbond can't update itself. Ask whoever set it up." Put a "Back up now" button right in the note, and only show the uninstall steps after a backup made today exists.

8. **low**, `r01-router-up-to-date-desktop.png`, `r02-router-update-available-desktop.png`, `11-updates-available.png`. The same version is written as "v0.2.0" (phone), "0.2.0" (cards) and "tailbond 0.2.0-r1" / "0.1.0-r42" (router footer). I'd wonder whether "-r42" is newer than "0.2.0".
   *Suggestion:* one format everywhere. Drop the build suffix from the footer or explain it.

9. **low**, `10-updates-up-to-date.png`, `r01-router-up-to-date-desktop.png`, `w02-updated-devices-out-of-date-backups-desktop.png`. Each device offers different choices: Tell me / Off on the phone, Tell me / Automatic / Off on the router, On / Off on the Fedora server. When the relative says "set it to Automatic", I can't find that on the phone.
   *Suggestion:* use the same three words everywhere. Show "Automatic" on the phone too, disabled, with "needs Obtainium".

10. **low**, `w01-homebox-dnf-update-available-desktop.png`, `w02-updated-devices-out-of-date-backups-desktop.png`. The tab badge shows "3" or "2" with no explanation: 3 updates? 3 problems?
    *Suggestion:* use a dot, or a tooltip/label like "2 devices out of date".

11. **low**, `13-updates-off.png`. "Updates are off · Not checked yet" on 0.1.0 is calm and grey, so I'd assume all is well, yet the server lists this phone as "Out of date".
    *Suggestion:* when updates are off and the last check is old or missing, add "You may be missing updates. Check now?"

12. **low**, `r07-router-update-confirm-desktop.png`. "devices drop for a few seconds": does that include the laptop I'm reading this on? The SHA-256 line means nothing to me.
    *Suggestion:* "Everything on this router's Wi-Fi, including this page, pauses for a few seconds." Drop or collapse the SHA-256 line.

---

## Task 2: Move the home server to a new machine using a backup

**Done?** Partly. Making the backup (w05, r09) and the restore itself (w06, w07) are clear enough to follow over someone's shoulder. What happens to my phone and the travel router can't be worked out from the screens: the instructions are conditional on things I can't check, and neither the phone app nor the router page shows the matching action.

Screens: s01, s02, w02, w05, w06 (desktop and phone), w07, w10, w12, r09, r10, r11, r13, 30, 31, 31b, 32, 33, 33b, 34, 35, 36, 37.

**Findings**

13. **high**, `w06-restore-preview-new-vps-desktop.png` / `w07-restore-done-desktop.png`. For my phone, the text says that a phone "with the Tailscale app" learns the new address by itself once someone renames machines in a "Tailscale admin console", and that a device "without Tailscale needs its link again… check slate7, pixel-8, beryl". I don't know which kind my phone is, and "check" doesn't say what to do. The phone app has no matching screen either: Settings (01) only offers "Pair again" and "Remove from this phone", with no hint about when to use them.
    *Suggestion:* replace the paragraph with a per-device list ("pixel-8: nothing to do" / "pixel-8: needs its new link: open tailbond on the phone → Pair again → scan this code"). In the phone app, when the server can't be reached after a move, show "Your home server moved? Tap to scan its new code."

14. **high**, `w06-restore-preview-new-vps-desktop.png`. "A travel router with the kill switch on has no internet until it reconnects, so send it its new link before you switch over." The router I carry has the kill switch on (r01, "On (recommended)"). If the relative does step 1 (turn the old one off) first, I'm left at a hotel with no internet at all. Step 1 says to turn the old one off "before restoring", and this line says to send the link "before you switch over", so the order is contradictory. And nothing on the router page (r01: "Ask to connect again", "Unpair", "Add a backup server") shows where a "new link" would go.
    *Suggestion:* make the order explicit and checkable: "1. Send the new link to slate7 (shows ✓ when it has it). 2. Then turn the old server off." Hold the Restore button, or show a red banner, while a kill-switch router hasn't received its link. On the router page, add a visible "Got a new link from your home server? Paste it here" entry.

15. **medium**, `w06-restore-preview-new-vps-phone.png`. Step 2 is one paragraph of about 120 words with two IP:port pairs, Tailscale renames and three device names. On the phone it fills the screen, and "What it replaces" is below the fold. Even the relative would miss things.
    *Suggestion:* break step 2 into "Devices that fix themselves" and "Devices you need to visit", one line each. Move addresses and ports into a collapsible "Details".

16. **medium**, `w07-restore-done-desktop.png`. After the restore, the box says "If you haven't already: 1. Before restoring, turn tailbond off on vps…". It's already restored, so "before restoring" is too late. It also suggests both servers may be running at once, and devices "would switch back and forth", without saying whether that's happening now.
    *Suggestion:* reword it for after the fact ("Now turn tailbond off on the old server") and show whether the old server is still answering ("old server still on: devices may flip between them").

17. **medium**, `s01-flint-devices-out-of-date-desktop.png` / `w02-updated-devices-out-of-date-backups-desktop.png`. "Move to a new router/machine: the restore preview says what each device needs." So I only learn what my phone and router need at the very end, on the new machine, after the backup is uploaded. That's too late to warn me before I travel.
    *Suggestion:* add a "Planning a move?" link on the old server that shows the same per-device checklist before anything happens.

18. **medium**, `r11-router-restore-done-desktop.png` / `w07-restore-done-desktop.png`. "Automatic backups now follow the backup: if they were on on slate7, they stay on here". It doesn't say whether they're on now, and the page behind the dialog (r11) still reads "Automatic backups Off". I can't tell whether I'm protected.
    *Suggestion:* say the result: "Automatic backups: On, every day at 03:00." Refresh the card behind the dialog.

19. **medium**, `30-backup-page.png`. The phone's backup page never says when the last backup was made, or whether one goes to homebox by itself. The server lists pixel-8 backups every day at 08:12 (w02), but the phone gives no hint of that. I'd keep making manual backups out of worry, or assume none exist.
    *Suggestion:* add a status line at the top: "Last backup: today 08:12, sent to homebox automatically ✓" or "No backup yet".

20. **medium**, `31b-backup-send-to-homebox-dialog.png`. I choose a passphrase for the copy sent to homebox, but the person who would restore it for me is the relative. Nothing tells me to share the passphrase with them, or that homebox can't open it either (s01: "Each is locked with the passphrase it was made with").
    *Suggestion:* add "If someone else helps you restore, they'll need this passphrase too", and maybe a "Share passphrase…" action.

21. **medium**, `34-restore-preview.png`. "1 app from the backup isn't installed on this phone…" is in red error text, inside a dialog where I'm already nervous, and it doesn't name the app. It's harmless, but the red makes it look like something broke.
    *Suggestion:* use neutral text, name the app ("Spotify isn't installed; it'll skip tailbond once you install it"), and keep red for real problems.

22. **medium**, `36-backup-page-undo-available.png`. The "Undo restore" button is dark grey on a slate card with a faint outline, so it looks disabled. If the restore went wrong, I'd think undo isn't possible.
    *Suggestion:* give it the normal enabled button style (filled or high-contrast outline).

23. **medium**, `w06-restore-preview-new-vps-phone.png` / `r10-router-restore-preview-desktop.png`. On the web dashboard, Restore is a solid red button, which reads as "delete". On the router, Restore is pale pink text on pink, which reads as disabled. Neither looks like the safe, expected action, and the "can undo" reassurance on the router is in small grey text above.
    *Suggestion:* make Restore a normal primary button on both. Put "Your current settings are saved first; you can go back" next to it in readable text.

24. **low**, `r10-router-restore-preview-desktop.png`. "Mode (default) → Speed", "Bypass rules 0 → 4", "Connection settings 0 → 3" don't match the names on the page ("Fastest", "Rules that skip tailbond"). The Android preview (34) says "Smooth calls and video", in a third vocabulary.
    *Suggestion:* use the exact on-screen names, and list actual items instead of counts where short.

25. **low**, `32-backup-saved.png`. The confirmation is small grey text below a divider, easy to miss, and it doesn't say where the file went (Downloads? Drive?).
    *Suggestion:* show a snackbar or card with a check mark: "Saved to Downloads/tailbond-pixel-8-2026-11-01.tbbackup".

26. **low**, `33b-restore-wrong-passphrase.png`. "Wrong passphrase or damaged file": I can't tell which, so I don't know whether to keep trying or give up.
    *Suggestion:* say "Most likely a typo in the passphrase. Check caps and spaces." and mention the damaged-file case only after several tries.

27. **low**, `37-restore-a-server-backup.png`. "Restore it there" doesn't tell me how to get to "there", and it's shown in red as if I did something wrong.
    *Suggestion:* use neutral text: "This is homebox's backup, not your phone's. Your phone's backups are named tailbond-pixel-8-…". Link to the right place.

28. **low**, `35-restore-done-with-undo.png` / `36-backup-page-undo-available.png`. The dialog says Undo "puts everything back as it was" without mentioning it works only once, which the card does say. The card times also disagree (14:30 vs 14:28).
    *Suggestion:* add "(once)" to the dialog and use one timestamp.

29. **low**, `w10-vps-new-machine-desktop.png`. The new machine shows a green "Connected" with no devices and nothing suggesting "Moving from an old server? Restore its backup now". The relative might pair devices fresh instead of restoring.
    *Suggestion:* on an empty server, add a prominent "Moving from another server? Restore from a backup" call to action.

30. **low**, `r11-router-restore-done-desktop.png`. "connects to its home server again by itself within a minute" and "this page comes back by itself in a few seconds" are in the same dialog. Which is it, and what should I wait for?
    *Suggestion:* one sentence: "Wait about a minute until the top card says Connected."

---

## Task 3: The router got a GL firmware update. Is tailbond still there?

**Done?** Yes. The top card says "tailbond is back after the firmware update, with your settings" and "Connected to homebox" (r06). I didn't have to do anything. The Updates card then makes me doubt that.

Screens: r06, r01, r13.

**Findings**

31. **medium**, `r06-router-reinstalled-after-firmware-upgrade-desktop.png`. "Nothing to do." sits in the middle of a dense green paragraph, followed by instructions for a case that may or may not be mine ("If you upgrade the firmware without Keep Settings: install tailbond again, then restore the router's backup… or pair again"). I don't know what "Keep Settings" is or whether it was ticked, so I'd think I have homework.
    *Suggestion:* make "Nothing to do. tailbond and your settings are back." the headline. Move the what-if text behind "What if the router was reset?" or remove it from this success note.

32. **low**, `r06-router-reinstalled-after-firmware-upgrade-desktop.png`. "reinstalled itself from the copy kept on this router (or, if that was gone, from the GitHub release)" doesn't say which happened, or whether the version is the same as before. The note has no way to dismiss it.
    *Suggestion:* state the outcome ("Reinstalled 0.2.0, the same version as before") and add a Dismiss / "Got it" button.

33. **low**, `r01-router-up-to-date-desktop.png` / `r06-router-reinstalled-after-firmware-upgrade-desktop.png`. "Saved in /etc/tailbond/state/backups/ on this router (kept through firmware upgrades with Keep Settings). Copy them somewhere else too." The file path and the GL term "Keep Settings" mean nothing to me, and I don't know how to "copy them somewhere else".
    *Suggestion:* replace the path with "Saved on this router and sent to homebox". Hide the path under Details. Turn "Copy them somewhere else" into a "Download latest backup" button.

---

## Overall

- **high:** 2
- **medium:** 17
- **low:** 14
