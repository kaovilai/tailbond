# Round 1: persona C, "family member"

Persona: a relative of the person who set tailbond up. Uses the Android app every day and opens the travel router's admin page only when asked to. Doesn't know what a server, package manager, checksum or "dnf" is, worries about breaking things or losing settings, and on every screen asks "is it working?" and "do I need to do anything?"

Screens used:
- Android: 01-settings-updates-and-backup-rows, 02-settings-update-available-row, 10-updates-up-to-date, 11-updates-available, 12-updates-check-failed, 13-updates-off, 19-updates-debug-build, 30-backup-page, 31-backup-save-dialog, 31b-backup-send-to-homebox-dialog, 32-backup-saved, 33-restore-type-passphrase, 33b-restore-wrong-passphrase, 34-restore-preview, 35-restore-done-with-undo, 36-backup-page-undo-available, 37-restore-a-server-backup
- GL.iNet: r01, r02 (desktop and phone), r03, r04, r05, r06, r07, r08, r09 (phone), r10, r11, r13, s01 (desktop)
- Web: w01 (desktop and phone), w02, w03, w04, w05, w06 (desktop and phone), w07, w08, w09, w10, w11, w12, w13

---

## Task 1: Make sure everything stays up to date

**Done?** Partly. The phone is fine: 02 and 11 show my version, that 0.2.0 is out, and a Download button. The travel router is fine too: r02 has "This router runs tailbond 0.1.0", "Update available: 0.2.0" and an Update now button. The home server is not. In w01 I can see it's on 0.2.0 and that 0.2.1 is out, but the only way to update it is a terminal command I don't understand, and I can't tell whether it will ever update by itself.
Screens: 01, 02, 10, 11, 12, 13, 19, r01, r02, r03, r04, r05, r07, s01, w01 (desktop and phone), w02, w03, w04, w08, w09, w11, w13.

**Findings**

1. **High**, w01-homebox-dnf-update-available-desktop.png: There is no way for me to update the home server. The card says to "run `sudo dnf upgrade tailbond-server`, or turn on dnf-automatic", with a Copy button, but it doesn't say where to type that, and "dnf-automatic" has no switch and no status, so I can't tell whether updates happen by themselves. I'd either paste the command somewhere random or give up.
   *Suggestion:* Lead with plain status: "This server gets updates with the computer's own system updates. Automatic system updates: on/off." Add "If you didn't set this computer up, ask the person who did." Put the command behind a "For the person who set this up" disclosure.

2. **Medium**, 02-settings-update-available-row.png: On the phone, the only sign of an update in these screens is a row at the very bottom of a long Settings page, below VPN, always-on and kill-switch text I'm afraid to touch. Nothing shown here tells me without digging that the phone needs attention.
   *Suggestion:* Show "Update available" on the app's main screen or as a notification, with a tap that goes straight to Settings → Updates.

3. **Medium**, 11-updates-available.png: "The first time, Android asks you to allow your browser to install apps." To me that sounds like a security warning I've been told never to accept. The screen also doesn't say whether my connection drops during the install, or whether I have to turn tailbond back on afterwards.
   *Suggestion:* Say "Android will warn you about installing apps from your browser. That's expected for tailbond. Tap Allow." Also say what happens to the connection: "tailbond reconnects by itself after the update" (or "open tailbond and turn it on again").

4. **Medium**, 19-updates-debug-build.png: The big blue "Download update" button is still there even though the text below says Android won't install it. The fix it offers is "uninstall tailbond … then restore the backup", which is exactly the kind of step where I'd lose my settings.
   *Suggestion:* On a debug build, hide or disable Download update and say "This copy of tailbond was installed by hand. Ask the person who set it up to switch it to the normal version."

5. **Medium**, w02-updated-devices-out-of-date-backups-desktop.png (same on s01-flint-devices-out-of-date-desktop.png): The card's chip is a green "Up to date", but just below it says "2 devices out of date", and one of them is my phone. I read the green chip, decide everything is fine, and stop reading.
   *Suggestion:* When devices are behind, make the chip amber, for example "Server up to date · 2 devices need updating". In the device list, put the action next to each device, e.g. "pixel-8: open tailbond on that phone → Settings → Updates".

6. **Medium**, r04-router-rolled-back-desktop.png (same on w08-vps-rolled-back-desktop.png): The red warning triangle and the red "Went back to 0.1.0" chip look like something broke. "Everything works as before" is buried in the red box. The bright blue "Update now" right below invites me to try again and drop everyone's connection a second time.
   *Suggestion:* Use a neutral or amber style that leads with "Nothing to do. tailbond is working on 0.1.0." Swap the primary Update now for a secondary "Try again" link, or wait for the next version.

7. **Medium**, w09-vps-check-failed-desktop.png: The card says "Up to date" in green and big text while the server runs 0.1.0 and the check failed ("Couldn't reach GitHub: api.github.com: i/o timeout"). It can't know that it's up to date, and the error is jargon.
   *Suggestion:* Use a "Couldn't check for updates" state like the Android 12 screen: "Couldn't check today. tailbond itself is still working. It will try again tomorrow."

8. **Medium**, w11-scheduled-backups-change-desktop.png and w01-homebox-dnf-update-available-phone.png: The top bar ("tailbond · Home server · Connected" plus its switch) floats over the middle of the Updates card. On w01-phone it covers the Tell me / Off buttons. On w11 it covers the help line under the device list and the "When a new version is out" heading. I also might hit the Connected switch by accident while scrolling.
   *Suggestion:* Pin the header to the top edge, or give the content enough top padding so it never covers the cards.

9. **Medium**, w01-homebox-dnf-update-available-phone.png: When I open the home server's page on my phone, the first thing I see is a big "Add this phone" card with a "+ Add this phone" button and a "Download tailbond for Android" button, even though my phone (pixel-8) is already paired. I'd worry that tapping it adds me twice or breaks my phone's setup.
   *Suggestion:* When this phone is already a device, hide the card or replace it with "This phone (pixel-8) is already added. Nothing to do."

10. **Low**, r02-router-update-available-desktop.png: The router's update status is the second-to-last card, below Internet connections, Kill switch, Tailnet machines and so on. The top card only says "Connected to homebox" and never mentions the update.
    *Suggestion:* Add one line to the top card ("Update available: 0.2.0, see below") or put a badge on the Updates card's heading near the top.

11. **Low**, 10-updates-up-to-date.png: The phone's only way to "update automatically" is to install a second app (Obtainium) and add tailbond to it "by that address". I wouldn't do that on my own, and the router and VPS just have an "Automatic" button.
    *Suggestion:* Say plainly "Optional. Ask whoever set up tailbond if you want this." Or offer a notification-only "Tell me" as the clear default for people who won't install another app.

12. **Low**, 12-updates-check-failed.png: A full red card ("Couldn't check for updates") makes me think tailbond itself is broken, when only the update check failed.
    *Suggestion:* Use amber and add "tailbond is still working. This only affects checking for new versions."

13. **Low**, 02-settings-update-available-row.png / r02-router-update-available-desktop.png: Versions are written three ways: "v0.2.0" (phone), "0.2.0" (cards), and "tailbond 0.1.0-r42" / "0.2.0-r1" in the router footer. I can't easily tell whether these are the same version.
    *Suggestion:* Write versions one way everywhere (e.g. "0.2.0") and drop the "-r1" suffix from what users see.

14. **Low**, w03-vps-update-available-desktop.png (same on w13): "All 3 devices are on 0.1.0 or newer" doesn't tell me whether my devices are current.
    *Suggestion:* Say "All 3 devices are up to date" or "2 of 3 devices need 0.2.0", in the same format as w02.

15. **Low**, s01-flint-devices-out-of-date-desktop.png: The Flint 3 home server's page says "Bond this router's internet connections into one tunnel to your home server", but this router is the home server.
    *Suggestion:* Use a home-server subtitle on this page, e.g. "Your travel routers and phones connect to this router."

---

## Task 2: Move the home server to a new machine using a backup

**Done?** Partly. Making the backup is clear (w05 and its "Write it down" box). The rest isn't. No screen shows putting tailbond on the new Fedora machine at all; the only "new machine" screens are a VPS (w10, w06, w07). The restore preview and the "Restored" dialog disagree about whether my phone and the router need anything, and the phone app tells me nothing.
Screens: w02, w05 (desktop and phone), w06 (desktop and phone), w07, w10, w12, s01, 01, 30, 31b, 33, 34, 35, 36, 37, r10.

**Findings**

16. **High**, w07-restore-done-desktop.png: The final dialog says "Devices reconnect by themselves too." One screen earlier, w06 said I must set the new public address myself (Devices tab), that devices pick it up "within a few hours", and that phones added by QR code need a new link. If I trust the last thing I read, I click Done, and the phone and travel router may stay offline with no one knowing why.
    *Suggestion:* Make the Restored dialog list the remaining steps with buttons, e.g. "1. Set the new public address [Open Devices]" and "2. Send new links to: pixel-8 [Show QR]". Don't say "reconnect by themselves" until both are done.

17. **High**, w06-restore-preview-new-vps-desktop.png (and the phone app, 01-settings-updates-and-backup-rows.png): The warning says devices "paired with a link and have no Tailscale (… a phone added by QR code)" need a new link, but it doesn't say which of slate7, pixel-8 and beryl that is. I can't tell whether my phone needs anything. The phone app has no matching message, such as "your home server moved, scan the new code". The only thing near it is a "Pair again" button with no context.
    *Suggestion:* In the preview and after the restore, mark each device: "pixel-8: needs a new link" or "slate7: will find the new server by itself". In the app, when the server can't be reached after a move, show "Your home server may have moved. Ask for a new link or scan the new code", linked to Pair again.

18. **Medium**, w10-vps-new-machine-desktop.png: The new machine "before restoring" looks exactly like a working, established server: "Connected", "UPTIME 5d 0h", live traffic, "All 3 devices are on 0.2.0 or newer", and six saved backups including slate7's and beryl's. With both browser tabs open, I can't tell old from new, so I could restore onto the wrong one or think the move is already done.
    *Suggestion:* Give a fresh machine an obvious empty state ("New server: nothing set up yet. Moving from another server? Restore its backup here.") and show the machine's own address prominently on both dashboards.

19. **Medium**, w02-updated-devices-out-of-date-backups-desktop.png: The saved backups (its own, slate7's, beryl's, and the phone's from 31b) live on the very machine being replaced. "Keep a copy somewhere other than this machine too" is small grey print at the bottom. "Each is locked with the passphrase it was made with" means I'd need passphrases I never chose or saw, and nothing says which one.
    *Suggestion:* Before a move, prompt "Download these to your computer first". Show who set each passphrase or an optional hint ("made by automatic backups: passphrase set on Nov 1").

20. **Medium**, w02-updated-devices-out-of-date-backups-desktop.png: The card subtitle promises "Move to a new machine and your devices keep working". w06 contradicts that: devices can take hours, and some need a new link. The promise sets me up to ignore the warnings later.
    *Suggestion:* Change it to "Move to a new machine. Most devices find it by themselves; you'll be told which need a new link."

21. **Medium**, w06-restore-preview-new-vps-desktop.png: "What it replaces" is all jargon: "Tunnel port 443 → 8443", "Tunnel addresses 10.77.0.0/24 → 10.83.142.0/24", "Approvers (none) → sam@example.com", "Devices (clients)". I can't judge whether this is right. "(none) →" reads as if nothing is being replaced, which contradicts the red Restore button.
    *Suggestion:* Start with one plain line, e.g. "This machine takes over from vps (203.0.113.9): its 3 devices and settings". Put the technical rows under "Details".

22. **Medium**, w10-vps-new-machine-desktop.png: For the task as given (Fedora homebox to a new machine), nothing shows how to get tailbond onto the new machine so that it has a "Restore from a backup" button. Restore only exists inside an already-running dashboard.
    *Suggestion:* Next to "Download a backup", add a "Moving to a new machine?" link with the steps: install tailbond there, open its page, Restore from a backup, then set the address and send links.

23. **Low**, w06-restore-preview-new-vps-desktop.png (also r10-router-restore-preview-desktop.png): The file name says one date and the line under it another. "vps-2026-11-01T0300" says "made Nov 2, 16:00", and "slate7-2026-11-01" says "made Nov 2, 17:00". That makes me doubt I picked the right file.
    *Suggestion:* Make the file name and the "made" time match, or show only the "made" time.

24. **Low**, 34-restore-preview.png: The red line "1 app from the backup isn't installed on this phone" doesn't name the app, and its red colour makes it look like an error that blocks Restore.
    *Suggestion:* Name the app and use neutral text: "Netflix isn't installed yet. It will skip tailbond once you install it."

25. **Low**, 36-backup-page-undo-available.png: "Undo restore" is grey on a grey card and looks disabled. "It works once" doesn't say until when.
    *Suggestion:* Use a normal outlined button with enough contrast, and say how long undo stays available.

26. **Low**, 37-restore-a-server-backup.png: "Restore it there" doesn't say where "there" is or how to get to it.
    *Suggestion:* "Open homebox's page (http://homebox…) → Updates & backup → Restore from a backup", ideally as a link.

---

## Task 3: The router got a GL firmware update. Is tailbond still there?

**Done?** Yes, but only after scrolling to the bottom. The top card in r06 still says "Connected to homebox", which answers "is it working". The green box near the bottom ("The GL firmware update removed tailbond; it reinstalled itself with your settings … Nothing to do") answers "did I have to do anything".
Screens: r06, r01, r05.

**Findings**

27. **Medium**, r06-router-reinstalled-after-firmware-upgrade-desktop.png: The one message that answers my question sits in the Updates card near the bottom, below six cards of settings I don't understand. The top of the page looks no different from any other day, so I'd likely never see "Nothing to do" and would message the person who set it up anyway.
    *Suggestion:* After a reinstall, show a short banner at the top of the page: "GL firmware updated. tailbond reinstalled itself with your settings. Nothing to do." Keep the details in the Updates card.

28. **Low**, r06-router-reinstalled-after-firmware-upgrade-desktop.png: The wording starts with "The GL firmware update removed tailbond", which scares me before I get to the good news. "Checked 2 h ago" sits under "since Nov 3, 21:36" (24 minutes ago), so the timing looks wrong. "Now on 0.2.0" doesn't say whether that's the same version as before the firmware update.
    *Suggestion:* Reorder it to "tailbond is back after the GL firmware update, with your settings (same version, 0.2.0)", and check again right after the reinstall so "Checked" isn't older than it.

---

## Overall count

- High: 3 (#1, #16, #17)
- Medium: 14 (#2, #3, #4, #5, #6, #7, #8, #9, #18, #19, #20, #21, #22, #27)
- Low: 11 (#10, #11, #12, #13, #14, #15, #23, #24, #25, #26, #28)
