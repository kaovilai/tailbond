# Round 3: persona D, "Speedify user"

Years on Speedify (silent app-store updates) and Peplink/SpeedFusion at work (scheduled firmware updates, config backup/restore files); expects a clear version number, one-click backup, a restore that just works on new hardware, and counts every extra step.

Screens used:
- Web: w01 (desktop + phone), w02, w03, w04, w05, w06 (desktop + phone), w07, w08, w09, w10, w11, w12, w13
- GL.iNet: r01, r02, r03, r04, r05, r06, r07, r08, r09, r10, r11, r13, s01, s02
- Android: 01, 03, 10, 11, 19, 30, 31b, 34, 37

## Task 1: Make sure everything stays up to date

**Done?** Partly. I can read a version for every machine and each one says how it updates. Getting everything current still means a terminal on the Fedora box, opening each router's own panel, and sideloading an APK on the phone. There is no single "update everything" path, and one state shows a false "up to date". Screens: w01, w02, w03, w04, w08, w09, w13, r01, r02, r03, r04, r05, r07, s01, 01, 03, 10, 11, 19.

**Findings**

- **medium**, `w01-homebox-dnf-update-available-desktop.png`: To make updates automatic, the screen says to set `apply_updates = yes` in `/etc/dnf/automatic.conf`. It doesn't warn that this makes dnf install updates for every package on the computer, not just tailbond (w02 admits this only after the change). A Speedify user who wants tailbond to update itself would quietly switch on unattended OS-wide updates.
  Suggestion: Say it in w01 before the command: "This turns on automatic installs for all packages on this computer, not just tailbond." If dnf supports it, offer a tailbond-only option (for example an `includepkgs` line).

- **medium**, `w01-homebox-dnf-update-available-desktop.png`: The banner says "Update available: tailbond 0.2.1" because GitHub has it, and the footnote says only that "a new version reaches dnf shortly after the release". If I run `sudo dnf upgrade tailbond-server` right away and get "Nothing to do", the screen hasn't told me why or when to try again.
  Suggestion: Only say "Update available" for dnf machines once the repo has it, or add "If dnf says Nothing to do, the package isn't in the repository yet; try again in about N hours."

- **medium**, `w09-vps-check-failed-desktop.png`, `w13-vps-updates-off-desktop.png`: In both screens the VPS runs 0.1.0 while 0.2.0 exists (w03 is the same VPS). slate7 and pixel-8 on 0.1.0 get a green "✓ up to date" because they match the server. When the check failed or is off, the server doesn't know the latest version, so the green tick is a false all-clear.
  Suggestion: When the latest version is unknown, show something neutral like "same as this server (0.1.0)" in grey, not a green "up to date".

- **medium**, `w01-homebox-dnf-update-available-desktop.png`, `s01-flint-devices-out-of-date-desktop.png`: The home server lists "2 of 3 out of date", but all it offers is the line "Routers: Update now in its tailbond panel". There is no link to the router's panel and no way to start the update from here. Peplink InControl and Speedify both handle this centrally.
  Suggestion: Put an "Open slate7's panel" link or a "Ask slate7 to update" button on each out-of-date row, or at least say which routers are already on Automatic.

- **medium**, `w01-homebox-dnf-update-available-desktop.png`, `s01-flint-devices-out-of-date-desktop.png`: beryl shows "version not known yet", and the explanation is "A version shows once the device has connected with this version of tailbond". "This version" could mean the server's version or beryl's. beryl clearly has connected before (it has a backup from Oct 31). I can't tell whether beryl is out of date or what to do about it.
  Suggestion: Name the cause and the action, e.g. "beryl hasn't connected since this server was updated (last seen Oct 31). Turn it on to see its version."

- **medium**, `11-updates-available.png`, `10-updates-up-to-date.png`: On the phone, "Download update" opens the browser, and Android then asks me to allow the browser to install apps. Automatic updates need a second third-party app (Obtainium) set up by hand with a GitHub address. Compared with Speedify updating silently through the store, that's several extra steps and a security prompt users are taught to refuse.
  Suggestion: Put the Obtainium setup front and centre with a one-tap deep link that adds tailbond in Obtainium, and say explicitly that the APK is signed with the same key, so Android will refuse a fake one.

- **low**, `w01-homebox-dnf-update-available-desktop.png`, `w03-vps-update-available-desktop.png`: The tab badge shows "3" or "2" with no label. I had to work out that it adds this server's update to the out-of-date devices.
  Suggestion: Add a tooltip or aria label such as "1 update, 2 devices out of date", or use a dot instead of a count.

- **low**, `r01-router-up-to-date-desktop.png`, `r02-router-update-available-desktop.png`, `03-home-update-available.png`: The same version is written three ways: "0.2.0" in the card, "tailbond 0.2.0-r1" or "0.1.0-r42" in the router footer, and "v0.2.0" on Android. I can't easily confirm that two devices run the same build.
  Suggestion: Use one format everywhere, and if the "-rN" package revision matters, explain it.

- **low**, `w01-homebox-dnf-update-available-desktop.png`, `w12-whats-inside-desktop.png`: The dnf server offers "Check for new versions: On / Off", but its backup summary lists "Updates: Tell me". I can't tell what "Tell me" means on a dnf machine, or what it becomes after a restore onto a VPS.
  Suggestion: Use the same words in the summary as on the card ("Check: On (dnf installs)").

- **low**, `r04-router-rolled-back-desktop.png`: The router is on Tell me, but the note says "Automatic won't install 0.2.0 again". It doesn't give a reason beyond "the tunnel didn't come back within 60 seconds", and there's no link to the logs. The home server will keep flagging slate7 as out of date for a version that just failed on it.
  Suggestion: On Tell me, say "We won't offer 0.2.0 again", and add "See log". On the home server, mark the device as "0.2.0 didn't start here", not just out of date.

- **low**, `r07-router-update-confirm-desktop.png`: The confirm dialog doesn't give the download size. The router is on hotel Wi-Fi plus phone tethering, which is capped at 5 GB, and I can't tell which connection it will use.
  Suggestion: Add the size ("about N MB, over the connections in use").

- **low**, `s01-flint-devices-out-of-date-desktop.png`: On the Flint home server the Devices card at the top shows no versions, and "2 of 3 out of date" sits about 3000 px down the page. The top status card says nothing about it.
  Suggestion: Add an "Out of date" chip to the device rows in the top Devices card, or a line in the status card with Show.

## Task 2: Move the home server to a new machine using a backup

**Done?** Partly. Making and restoring the backup are each a single dialog. But the "Before you restore" box makes me work out the right order of steps and which devices need what. One sequence can strand a travel router whose kill switch is on. Screens: w02, w05, w06, w07, w10, w11, w12, s01, s02, r10, r11, r13, 30, 31b, 34, 37.

**Findings**

- **high**, `w06-restore-preview-new-vps-desktop.png`, `w07-restore-done-desktop.png`: Step 1 says to turn tailbond off on the old vps before restoring. Step 2 says a kill-switch travel router "has no internet until it reconnects, so send it its new link before you switch over". But the new link only exists on the new machine after the restore (Devices tab, QR). If I follow the steps in order, slate7 at the hotel has no internet before it can receive the link. Restore also doesn't enforce step 1: w07 shows "Before restoring, turn tailbond off…" after the restore has already happened.
  Suggestion: Give one ordered checklist that works: restore on the new machine, get new links, send them, then turn off the old one, or else turn off the kill switch on those routers first. Name the routers that have the kill switch on. Make "I've turned the old one off" a checkbox that has to be ticked before Restore is enabled.

- **medium**, `w06-restore-preview-new-vps-desktop.png`: Item 2 is a single paragraph of about 100 words. It mixes Tailscale admin-console renaming, QR re-linking and kill-switch caveats, and ends with "check slate7, pixel-8, beryl". The server knows which devices reached it over Tailscale and which by link, so it shouldn't leave me to guess per device. On Peplink this is "restore, done".
  Suggestion: Show a per-device table: slate7: needs new link (QR); pixel-8: nothing (Tailscale), once you rename the machine to "vps"; beryl: off, needs new link when back. Link the rename step straight to the Tailscale admin page.

- **medium**, `w11-scheduled-backups-change-desktop.png`: The passphrase for automatic backups "is kept on this machine". Unlike the download dialog (w05, r09), there is no "Write it down" warning. When the old machine dies, which is exactly when a move happens, the only copy of the passphrase dies with it, and the backups can't be opened.
  Suggestion: Show the same "Write it down" warning when you set or change the automatic backup passphrase, and remind people once a month or so ("Do you still have your backup passphrase?").

- **medium**, `w02-updated-devices-out-of-date-backups-desktop.png`, `s01-flint-devices-out-of-date-desktop.png`: The home server's automatic backups are saved only on the home server (`/var/lib/tailbond/backups/`), with "Copy them somewhere else too" as the only safeguard. Routers send theirs to homebox, but homebox sends its own nowhere. If the home server's disk dies, there's nothing to restore from.
  Suggestion: Offer an off-machine destination (send to a paired router or VPS, or a folder or URL), or at least show "last copied off this machine: never" as a warning.

- **low**, `w06-restore-preview-new-vps-desktop.png`: "What it replaces" sits below the long warning and needs scrolling. Further down it includes "Update policy Tell me → Automatic", so a fresh server will start updating itself and I'd only notice if I scrolled.
  Suggestion: List setting changes that affect behaviour (update policy, automatic backups) in a short line above the fold.

- **low**, `w02-updated-devices-out-of-date-backups-desktop.png`, `w05-backup-download-dialog-desktop.png`: Each backup is "locked with the passphrase it was made with": one for the manual download, one for the automatic backups, and one for each device's own backups that travel inside the server backup. That's a lot of secrets to keep track of for something that should be one-click.
  Suggestion: Offer "use the same passphrase as automatic backups" in the download dialog, and say which passphrase each saved row needs.

- **low**, `w02-updated-devices-out-of-date-backups-desktop.png`: It says "keeps the last 7 on this server" but only 3 "this server" rows are listed (2 on the Flint, s01), so it looks as if some are missing.
  Suggestion: Add "(3 so far)" or show the start date of the schedule.

- **low**, `30-backup-page.png`, `31b-backup-send-to-homebox-dialog.png`: homebox lists daily pixel-8 backups at 08:12, but the phone's backup page shows nothing automatic, only a manual "Send a backup to homebox" that asks for a new passphrase each time. After the move I can't tell whether the phone keeps sending, or with which passphrase.
  Suggestion: Show "Automatic: sent to homebox daily, last today at 08:12" on the phone page.

## Task 3: The router got a GL firmware update. Is tailbond still there?

**Done?** Yes. The top status line says "tailbond is back after the firmware update, with your settings", and the Updates card says "Nothing to do". Screens: r06, r01, r08.

**Findings**

- **medium**, `r06-router-reinstalled-after-firmware-upgrade-desktop.png`, `r01-router-up-to-date-desktop.png`: The rule that matters ("If you upgrade the firmware without Keep Settings: install tailbond again, then restore…") appears only after a successful upgrade with Keep Settings, when it no longer helps. If someone unticks Keep Settings, tailbond and this card are gone. Beforehand, the only hint is grey small print under Automatic backups in r01.
  Suggestion: Show a permanent line in the Updates card such as "Before a GL firmware upgrade: leave Keep Settings ticked; tailbond reinstalls itself", plus the date of the last backup sent to homebox.

- **low**, `r06-router-reinstalled-after-firmware-upgrade-desktop.png`: "Reinstalled itself from the copy kept on this router (or, if that was gone, from the GitHub release)" doesn't say which one happened, or whether the version changed.
  Suggestion: Say what actually happened, e.g. "Reinstalled 0.2.0 from the copy on this router."

- **low**, `r06-router-reinstalled-after-firmware-upgrade-desktop.png`: "Install tailbond again" doesn't say how or where (there is no link), and "or pair again" doesn't say what pairing loses compared with restoring.
  Suggestion: Link the install steps (the same "Download the router package" steps as on the home server, s01), and add one line on what pairing again loses (rules and connection settings).

## Overall

high: 1 / medium: 10 / low: 12
