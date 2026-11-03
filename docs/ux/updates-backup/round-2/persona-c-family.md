# Round 2: persona C, "family member"

A relative of the person who set tailbond up: uses the Android app, opens the travel router's admin page only when asked, doesn't know what a server, package manager, checksum or "dnf" is, worries about breaking things or losing settings, and on every screen wants to know "is it working?" and "do I need to do anything?"

Screens used:
- Android: 01-settings-updates-and-backup-rows, 02-settings-update-available-row, 03-home-update-available, 10-updates-up-to-date, 11-updates-available, 12-updates-check-failed, 19-updates-debug-build, 30-backup-page, 31-backup-save-dialog, 31b-backup-send-to-homebox-dialog, 32-backup-saved, 33-restore-type-passphrase, 33b-restore-wrong-passphrase, 34-restore-preview, 35-restore-done-with-undo, 37-restore-a-server-backup
- GL panel: r01, r02 (desktop + phone), r03, r04, r05, r06, r07, r08, r09, r10, r11, r13, s01, s02 (all desktop unless noted)
- Web dashboard: w01 (desktop + phone), w02, w03, w05, w06, w07, w08, w09, w10, w11, w12

## Task 1: make sure everything stays up to date

**Done?** Partly. I can see what version the phone runs (01, 02, 10, 11) and what the travel router runs (r01, r02), and both say clearly whether an update is waiting. The home server (homebox, Fedora) is where it breaks down: its only way to update is typing a command in a terminal, and so is its only way to turn on automatic updates (w01, w02). I can't do that, and the page doesn't tell me to ask someone. Screens: 01, 02, 03, 10, 11, 12, 19, r01, r02, r03, r04, r05, r07, s01, w01, w01-phone, w02, w03, w08, w09.

**Findings**

1. **High**, `w01-homebox-dnf-update-available-desktop.png`: For the home server, both "update now" and "turn on automatic updates" are `sudo dnf …` commands with a Copy button, and "if an update misbehaves" is another one (`sudo dnf downgrade tailbond-server`). A family member can't do this, and nothing on the page says "this is a job for whoever set up the server" or offers a button. **Suggestion:** Say plainly at the top of the card that someone with access to the server's terminal has to do this (name the person if the setup recorded one). Put the commands behind a "For the person who manages this server" expander. If the agent can safely trigger the package update itself, give it the same Update now button as the VPS.

2. **Medium**, `w01-homebox-dnf-update-available-desktop.png`, `w03-vps-update-available-desktop.png`, `10-updates-up-to-date.png`, `r01-router-up-to-date-desktop.png`: There are three different update models. Routers and the VPS show Tell me / Automatic / Off. Homebox shows Tell me / Off, plus a terminal command for automatic. The phone shows Tell me / Off, plus a separate "Automatic: use Obtainium" section. I can't learn one rule like "pick Automatic everywhere". **Suggestion:** Show the same three choices on every device. Where Automatic can't be switched on from that page, show it disabled with one plain line saying why and who can turn it on.

3. **Medium**, `10-updates-up-to-date.png`, `11-updates-available.png`: To update the phone automatically, I'd have to install another app (Obtainium) from outside the Play Store and "add tailbond in it by that address", and the help is an external docs link. I'd skip this, and the screen doesn't say whether skipping it matters. **Suggestion:** Lead with a sentence that it's optional and what happens without it ("you'll see a card on the main screen when there's a new version; tap it"). Move the Obtainium setup into a step-by-step screen inside the app instead of an outside link.

4. **Medium**, `11-updates-available.png`: Updating means letting my browser "install apps", and Android shows a warning about apps from unknown sources. The note calls this normal, but it doesn't describe or show the warning I'll see, and I'm used to "never allow this". I'd likely back out halfway through. **Suggestion:** Show a picture of the exact Android prompts in order, and say whether to turn the permission back off afterwards. Also say up front, before anything is downloaded, that the update keeps my settings (this is currently in the middle of the paragraph).

5. **Medium**, `w03-vps-update-available-desktop.png`, `w08-vps-rolled-back-desktop.png`, `w09-vps-check-failed-desktop.png`: The VPS lists slate7 and pixel-8 on 0.1.0 with a green "✓ up to date", while 0.2.0 is out and the phone app says "Update available: v0.2.0". In w09 the check failed, yet the devices are still marked up to date. Two screens give opposite answers to "is my phone out of date?" **Suggestion:** Compare devices with the newest release, not with this server's version. If a device matches the server but not the newest release, say "same as this server" in neutral grey, and never use a green tick when the check failed.

6. **Medium**, `s01-flint-devices-out-of-date-desktop.png`: On the Flint 3 home server, the Updates chip is a green "Up to date" and the top card says only "Home server is on", even though 2 of 3 devices are out of date. The device list near the top of the page has no out-of-date marker either. You only find out at the very bottom, under Updates. **Suggestion:** Add a line to the top card ("2 devices need an update", with Show) like the web dashboard's tab badge, and put the "Out of date" chip next to each device in the Devices card.

7. **Medium**, `r01-router-up-to-date-desktop.png`, `r05-router-automatic-desktop.png`, `w03-vps-update-available-desktop.png`: "Tell me (recommended)" is preselected, and its help text says "nothing installs until you click Update now". For someone like me, that means being nagged to press a button I'm afraid of. Automatic, which goes back by itself if the new version fails (r05), is the safer choice for me, but it isn't the recommended one. **Suggestion:** Say who each option is for ("Automatic: best if nobody checks this page"), or recommend Automatic on travel routers, since they already roll back by themselves.

8. **Medium**, `19-updates-debug-build.png`: On a debug build, the app tells me to "back up first… Then uninstall tailbond, install the release APK, and restore the backup". If I did this alone, a forgotten passphrase or a backup saved in the wrong place would lose everything, and nothing checks that the backup worked before I uninstall. The pointer "Settings → Backup" also doesn't match the row title "Back up and restore". **Suggestion:** Tell the family member to ask the person who installed the app, and hide the steps behind "I set this up myself". If the steps stay, make step 1 a button that saves the backup and then confirms it can be opened with the passphrase before saying "now uninstall".

9. **Medium**, `w01-homebox-dnf-update-available-phone.png`: When I open homebox's dashboard on my phone, which is already paired as pixel-8, the first thing I see is a large "Add this phone" card with a "+ Add this phone" button. The update banner sits below it. I'd wonder whether my phone is set up at all, and might add it a second time. **Suggestion:** Hide the card (or show "This phone is already added as pixel-8") when the visitor is a known device, and keep the update banner first.

10. **Medium**, `12-updates-check-failed.png`: A failed update check fills the top of the screen with a pink/red card titled "Couldn't check for updates". Red tells me tailbond is broken, but only the check failed: the connection itself is fine. The web version (w09) uses amber for the same thing. **Suggestion:** Use a neutral or amber style, and start with "tailbond is working normally; only the check for a new version failed. It tries again by itself."

11. **Low**, `03-home-update-available.png`, `11-updates-available.png`, `r01-router-up-to-date-desktop.png`, `r02-router-update-available-desktop.png`: The same version is written as "v0.2.0", "0.2.0", "tailbond 0.2.0-r1" and "tailbond 0.1.0-r42" (the router footers). I can't tell whether "0.2.0-r1" counts as 0.2.0. **Suggestion:** Show one form everywhere ("0.2.0"), and keep build suffixes for a details view.

12. **Low**, `r07-router-update-confirm-desktop.png`: "checked against its SHA-256 list before anything is installed" means nothing to me and reads like a warning. **Suggestion:** "tailbond checks the download is genuine before installing it."

13. **Low**, `r03-router-installing-desktop.png`: During the install, nothing says whether it's safe to close the page or whether I must not unplug the router. I'd sit and watch it, afraid to move. **Suggestion:** Add "You can close this page. If the router loses power, tailbond goes back to the old version by itself" (if true).

14. **Low**, `r04-router-rolled-back-desktop.png`: The policy is Tell me, yet the note explains what Automatic will do. "Try 0.2.0 again" sits next to Check now with the same weight, and nothing says whether pressing it is safe or should be left to someone else. **Suggestion:** Show the Automatic sentence only when Automatic is on. Add "Nothing to do; whoever manages tailbond can try again later" and make "Try again" a link.

15. **Low**, `w09-vps-check-failed-desktop.png`: "api.github.com: i/o timeout" is shown to everyone. **Suggestion:** Keep the plain sentence and move the raw error into Logs or a details expander.

16. **Low**, `w01-homebox-dnf-update-available-desktop.png`, `w02-updated-devices-out-of-date-backups-desktop.png`: The tab badge shows "3" on one screen and "2" on another, and nothing says what it counts. **Suggestion:** Give the badge a tooltip or label ("2 devices out of date, 1 update"), or use a dot.

## Task 2: move the home server to a new machine using a backup

**Done?** Partly. I could follow the backup (w05) and the restore (w06, w07) on the screens, though the only move shown is VPS → new VPS, not the Fedora homebox. What I could not work out is the part that concerns me: whether my phone and the travel router will keep working, or whether I'll have to do something on them. Screens: w02, w05, w06, w07, w10, w11, w12, r10, r11, r13, s02, 01, 30, 31, 31b, 32, 33, 33b, 34, 35, 37.

**Findings**

1. **High**, `w06-restore-preview-new-vps-desktop.png`, `w07-restore-done-desktop.png`, `01-settings-updates-and-backup-rows.png`, `03-home-update-available.png`: Whether my phone needs anything depends on two things. First, whether it is a "device with Tailscale" or one "paired with a link or QR code only". Second, whether the new machine "has the old one's Tailscale name". My phone app never tells me which kind of device it is. Its home screen even says "Not a Tailscale address", which points both ways. None of the Android screens shows what I'd see if the server moved and the phone needed its link again. So the honest answer to "does my phone need anything?" is "can't tell". **Suggestion:** In the restore preview, list each device by name with a verdict ("slate7: reconnects by itself"; "pixel-8: needs its link again, send it from the Devices tab"). In the phone app, add a clear state such as "homebox moved: ask for a new link / scan the new code" in place of an endless "connecting".

2. **Medium**, `w06-restore-preview-new-vps-desktop.png`, `w12-whats-inside-desktop.png`: Devices only follow by themselves "as long as this machine has the old one's Tailscale name (vps)". But nothing explains how to give the new machine that name, and the steps in "What's in a backup" ("install tailbond, sign in to Tailscale, then restore") don't mention it. **Suggestion:** Show the Tailscale name the new machine has now next to the one the backup expects ("this machine: vps2; backup expects: vps"). Give one plain step for making them match, or say that devices need their link again if they don't.

3. **Medium**, `w07-restore-done-desktop.png`: After the restore, "Next steps" repeats "Turn tailbond off on vps **first** (or wipe it)". "First" is too late at this point, and "wipe it" is a destructive instruction thrown in casually. A worried relative could wipe the old machine before checking that the new one works, leaving no way back. **Suggestion:** Rewrite this for after the restore ("Now turn tailbond off on vps"), drop "or wipe it", and add "keep the old machine until your devices show up here".

4. **Medium**, `w06-restore-preview-new-vps-desktop.png`: Item 3 asks me to "allow udp/8443 in this machine's firewall and in your provider's or home router's (port forward)", with nothing to show whether it's already done. The screens only cover VPS → VPS. For a home PC like homebox, the port forward on the home router still points at the old PC, even when the port number stays the same, and nothing here says so. **Suggestion:** After the restore, run the existing "reached from the internet" check (the setup-health check on s01) and show pass/fail. For a home server, say that the home router has to send traffic to the new PC, and that this is a job for whoever manages the router.

5. **Medium**, `30-backup-page.png`, `31b-backup-send-to-homebox-dialog.png`, compared with `w02-updated-devices-out-of-date-backups-desktop.png`: Homebox lists a pixel-8 backup every day at 08:12, so the phone must be sending them automatically. The phone's own backup page shows no automatic backups, no "last sent", and asks for a new passphrase every time I tap "Send a backup to homebox". I can't tell whether my phone is already backed up, or which passphrase protects those daily copies. **Suggestion:** Add a status line at the top of the phone's backup page ("Sent to homebox automatically every day; last one today 08:12"), and say whether a manual send uses the same passphrase.

6. **Medium**, `r10-router-restore-preview-desktop.png`, `r13-router-whats-inside-desktop.png`, `s02-flint-whats-inside-desktop.png`, `w12-whats-inside-desktop.png`: The restore and "what's inside" screens show internal values instead of the words used elsewhere: "Update policy notify → auto" (elsewhere "Tell me" / "Automatic"), "Mode speed" (elsewhere "Fastest"), "Bypass rules" (elsewhere "Rules that skip tailbond"), "Router settings (UCI)", "global.key", "Tunnel addresses 10.77.0.0/24". I can't check whether the backup holds my settings if they're named differently. **Suggestion:** Use the same labels as the main page, and leave key file names and address ranges out of anything a non-admin sees.

7. **Low**, `w10-vps-new-machine-desktop.png`: The brand-new machine shows a green "Connected" chip and toggle while it has no devices and no traffic. "Connected" to what? **Suggestion:** Say "Ready, no devices yet", and on an empty machine add a line "Moving from another server? Restore from a backup" next to the button.

8. **Low**, `w06-restore-preview-new-vps-desktop.png`: "Approvers (none) → sam@example.com" appears with no explanation. A stranger's email address gaining a role on my server looks alarming. **Suggestion:** Explain it in words ("sam@example.com will be able to approve new devices, as on the old machine").

9. **Low**, `r11-router-restore-done-desktop.png`: The same dialog says "connects… within a minute" and "comes back by itself in a few seconds"; it has a typo ("if they were on on slate7"); and the card behind it still says "Automatic backups: Off" right after it says they follow the backup. **Suggestion:** Use one time estimate, fix the typo, and refresh the card behind the dialog (or say "updates after the restart").

10. **Low**, `32-backup-saved.png`: The only confirmation is faint body text at the bottom ("Saved tailbond-pixel-8-2026-11-01.tbbackup"), and the date in the name isn't today's. It doesn't say where the file went. **Suggestion:** Show a clear success message with the location ("Saved to Downloads") and today's date.

11. **Low**, `w07-restore-done-desktop.png`: "Restored" has a padlock icon, which reads as "locked" rather than "done". **Suggestion:** Use the tick icon that the GL version (r11) uses.

## Task 3: the router got a GL firmware update. Is tailbond still there?

**Done?** Yes, for the case shown: the top card says "tailbond is back after the firmware update, with your settings", and the Updates card says "Nothing to do" (r06). There's no screen for the bad case, where tailbond is gone. Screens: r06, r01, r08.

**Findings**

1. **Medium**, `r06-router-reinstalled-after-firmware-upgrade-desktop.png`: The only instruction for when things go wrong ("If you upgrade the firmware without Keep Settings, install tailbond again and pair again") is on the tailbond page, which only exists when tailbond survived. If it didn't, VPN → tailbond is simply missing from the menu, and nothing tells me what happened or who to call. The phone and homebox screens don't show "slate7 hasn't been seen since…" either. **Suggestion:** Have homebox (and the phone app, when it uses that router) show a device that has gone missing ("slate7 not seen since 21:30; if its firmware was updated without Keep Settings, tailbond needs installing again"). Before a firmware update, show a line in the router panel telling me to tick Keep Settings.

2. **Low**, `r06-router-reinstalled-after-firmware-upgrade-desktop.png`: "Keep Settings" isn't explained (it's an option in GL's own firmware screen), and "(or, if that was gone, from the GitHub release)" is detail I don't need. **Suggestion:** Say "the 'Keep Settings' box on GL's firmware upgrade page", and drop the GitHub parenthesis.

3. **Low**, `r06-router-reinstalled-after-firmware-upgrade-desktop.png`: It is 22:00 and tailbond reinstalled itself at 21:36, yet the top card says "Up 2 h 13 min" and homebox has been "Connected for 2 h 13 min". After a firmware update and a reboot, that can't be right, and it makes me doubt the rest of the message. **Suggestion:** Make sure the uptime and connected time reset after the reinstall, or show "back since 21:36".

## Overall count

- High: 2
- Medium: 15
- Low: 13
