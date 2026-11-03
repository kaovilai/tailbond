# Round 2: persona A, "GL traveller"

A consultant who travels monthly with a GL.iNet Slate 7 and an Android phone. They know GL's admin panel (Repeater, VPN Dashboard, firmware upgrades) but aren't a network engineer, and a relative set up the Fedora home server. They want hotels to just work, calls not to drop, no work-traffic leaks and no update surprises.

Screens used:
- GL panel: r01, r02 (desktop and phone), r03, r04, r05, r06, r07, r08, r09, r10, r11, r13, s01, s02.
- Web dashboard: w01, w02, w05, w06, w07, w10, w11, w12.
- Android: 02, 03, 10, 11, 19, 30, 31b, 37.

I skipped the VPS-only update screens (w03, w04, w08, w09, w13) because I don't have a VPS. I also skipped the phone variants after seeing the desktop ones.

## Task 1: make sure everything stays up to date

**Done?** Partly. I can see every version: the home server runs 0.2.0 with 0.2.1 out (w01), the Slate 7 runs 0.1.0 with 0.2.0 out (r02), and the phone runs 0.1.0 with 0.2.0 out (03, 11). I can also see which devices are out of date (w01/w02 device list, s01). The router and the phone I can update myself. The home server I can't update myself, from home or from a hotel. Screens: w01, w02, r01, r02, r03, r04, r05, r07, r10, r13, 02, 03, 10, 11, 19.

Findings:

1. **high**, w01-homebox-dnf-update-available-desktop.png: The only way to update the home server is to "run in a terminal on this server" `sudo dnf upgrade tailbond-server`. Turning on automatic updates is a second sudo command. I don't have a terminal on the relative's Fedora PC, and from a hotel I can't reach one at all, so the server stays on 0.2.0 until the relative does it. Suggestion: add an "Update now" button that asks the system's package manager (PackageKit/dnf) to install the update, and ship the RPM with dnf-automatic on. At least add a line for people who can't do this: "Not your PC? Send this to whoever set it up", with a copy/share link.

2. **medium**, w02-updated-devices-out-of-date-backups-desktop.png: "When a new version is out" shows **Tell me (recommended)** selected. Right under it the page says "Automatic updates: on (dnf-automatic)". I can't tell whether new versions install by themselves or wait for me. The w01 toggle offers only Tell me / Off, so there is no "Automatic" choice that matches this state. Suggestion: when dnf-automatic is on, show the policy as "Automatic (dnf-automatic, runs at <time>)" and don't highlight "Tell me".

3. **medium**, r07-router-update-confirm-desktop.png (with r03, r04): The confirm dialog promises "devices drop for a few seconds". The installing card says "this page comes back by itself in about a minute". The rollback note says it waited 60 seconds for the tunnel. With the kill switch on, a minute of no traffic drops my video call. I can't tell whether it's safe to click Update now during a meeting. Suggestion: give one honest number in the dialog ("devices may be offline for up to about a minute; if the new version doesn't start, up to about 2 minutes before it goes back"). Add "don't do this during a call".

4. **medium**, r05-router-automatic-desktop.png: Automatic installs "between 03:00 and 05:00 (this router's time)". My router travels across time zones, and I can't see what its clock is set to. 03:00 router time could be the middle of my afternoon call in another country. The card also doesn't say what happens if the router is unplugged at night, which travel routers usually are. Suggestion: show the router's current time and time zone next to the window, and convert it to my browser's local time. Say "if the router is off then, it installs the next night it's on" (or whatever the real behaviour is).

5. **medium**, r10-router-restore-preview-desktop.png and r13-router-whats-inside-desktop.png: Restoring silently changes "Update policy notify → auto", shown as a small row in the table. "What's in a backup" says "Updates: auto" while the panel behind it shows **Tell me (recommended)** highlighted. After a restore, my router could start installing updates by itself at 03:00 without my having chosen it. Suggestion: in the preview, call out the policy change in plain words ("Updates: Tell me → Automatic (installs by itself at night)"), with a checkbox to keep the current choice. Make "What's inside" agree with the toggle.

6. **medium**, 11-updates-available.png: "tailbond reconnects by itself after the update." It doesn't say what my apps do while tailbond is being replaced. Unless "Block connections without VPN" is on (only mentioned on the Settings page, 02), work traffic goes out over hotel Wi-Fi during the install. Suggestion: add one line before Download update: "While Android installs it, apps use the phone's normal connection unless Block connections without VPN is on (Settings)", with a link.

7. **low**, w01-homebox-dnf-update-available-desktop.png / w02: The tab badge says 3 on w01 and 2 on w02, but the page never explains what is being counted. Suggestion: make the badge a plain "!" or dot, or say "3 things need an update" in the card header.

8. **low**, r01/r02/r05 footers: The footer says "tailbond 0.2.0-r1" / "0.1.0-r42", while the cards say "0.2.0" / "0.1.0". When the server says slate7 is on "0.1.0", I can't tell whether "-r42" matters. Suggestion: show the same version string everywhere, or label the suffix ("package build 42").

9. **low**, r04-router-rolled-back-desktop.png: The note says "Automatic won't install 0.2.0 again" while the policy is Tell me. It doesn't say whether I should report the failure or leave it. Suggestion: under Tell me, say "You'll be told about the next version; 0.2.0 won't be offered again", and add "Send a report" or link the logs.

10. **low**, 10-updates-up-to-date.png / 19-updates-debug-build.png: The phone has no "Automatic" choice. The answer is "install Obtainium", a separate sideloaded app, and the app can't tell me whether Obtainium is already set up. On the debug build (19) it still recommends Obtainium, even though the same screen says a release APK won't install over a debug build. Suggestion: on debug builds, hide the Obtainium advice. On release builds, say "Obtainium found / not found" if the app can detect it.

## Task 2: move the home server to a new machine using a backup

**Done?** Partly. I can make the backup (w05) and restore it (w06/w07). I can't work out whether my Slate 7 and phone will keep working or need re-pairing, and there are no screens for a Fedora-to-Fedora move, only VPS-to-VPS. Screens: w02, w05, w06, w07, w10, w11, w12, s01, s02, r01, r13, 02, 30, 31b, 37.

Findings:

11. **high**, w06-restore-preview-new-vps-desktop.png (with w02 and the phone's 02): The Backup card promises "Move to a new machine and your devices keep working." The restore check then says devices "with Tailscale" follow by themselves, and devices "paired with a link or QR code only" need their link again. Nowhere is it shown which kind slate7 and pixel-8 are. The device lists (w01, s01) show only name, type and version. The router (r01) says "Connects to home.example.net over the internet", and the phone (02) says "Reached at home.example.net:443". If I guess wrong while abroad with the kill switch on, the router blocks everything and I can't re-pair it from a hotel. Suggestion: in the restore check, list each device by name with "will reconnect by itself" or "needs its link again (do this before you travel)". Add a "how it's paired" column to the device list, and take the "keep working" promise out of the card subtitle.

12. **medium**, w06-restore-preview-new-vps-desktop.png and w12-whats-inside-desktop.png: Tailscale devices follow only "as long as this machine has the old one's Tailscale name". What's inside says "On a new machine: install tailbond, sign in to Tailscale, then restore." That normally gives the new machine a new name (homebox-1) while the old one still exists, and no screen says how to get the old name. Suggestion: add the step to the w12 text ("in the Tailscale admin console, remove the old homebox, then rename this one to homebox"). After restore, show whether this machine's Tailscale name matches.

13. **medium**, w06-restore-preview-new-vps-desktop.png: "Turn tailbond off on vps first (or wipe it)" has no instructions. I can't tell whether the "Connected" toggle in the old server's header counts, or whether it stays off after a reboot or a dnf update. If the old PC comes back, my devices "keep switching between them", which means dropped calls. Suggestion: give the exact step (in the old server's dashboard, Turn off for good, or `sudo systemctl disable --now tailbond-server`). After restore, show a warning if the old server is seen online again.

14. **medium**, w06-restore-preview-new-vps-desktop.png: The only firewall or port-forward advice appears because the port changes (443 → 8443). For a home move, the port stays the same, but the home router's port forward still points at the old PC's local address. Nothing on these screens mentions that, so my router and phone would lose the home server. Suggestion: on a home-server restore, always say "If your home router forwards UDP <port> to the old PC, point it at this one (<LAN address>)". Re-run the "UDP 443 reached from the internet" health check shown in s01 after restore.

15. **medium**, w07-restore-done-desktop.png: After Restore there is only "Done". Nothing shows whether slate7 and pixel-8 have actually come back. The whole point of the task is that they still work. Suggestion: after restore, show a live list ("slate7: connected 22:03 ✓, pixel-8: waiting…") that ticks as each device reconnects, with the link/QR action next to any that don't.

16. **medium**, w02-updated-devices-out-of-date-backups-desktop.png / w11-scheduled-backups-change-desktop.png: The home server's automatic backups are saved only on the same machine (/var/lib/tailbond/backups/), with "Copy them somewhere else too" but no way to do that besides one Download link at a time. My routers send their backups to homebox, but homebox sends its own backup nowhere. If the old PC dies, which is the usual reason to move, there's no backup to move with. Suggestion: offer "also send to" (a travel router, a phone, a folder or cloud). At least show a warning when the newest copy exists only on this machine.

17. **low**, w11-scheduled-backups-change-desktop.png: "A passphrase is set. Leave the fields empty to keep it." The relative set it, I don't know it, and the 03:00 backups are useless to me without it. Suggestion: next to the saved backups, say "locked with the automatic-backup passphrase (set on <date>)", and offer "check a passphrase" so I can test one before the old machine is gone.

18. **low**, w12-whats-inside-desktop.png (and s02, r13): Raw values appear: "Mode: speed", "Updates: notify/auto", "Router settings (UCI): no" on a Fedora PC. The rest of the UI says "Fastest" and "Tell me / Automatic". Suggestion: use the same labels as the controls, and drop the UCI row on non-router servers.

19. **low**, r10-router-restore-preview-desktop.png: The Restore button is pale pink and looks disabled next to a normal Back button. Suggestion: use the same solid red as on w06.

20. **low**, r11-router-restore-done-desktop.png: "if they were on on slate7, they stay on here". The doubled "on" is a typo, and the sentence is conditional while the card behind it shows Off. Suggestion: state the actual result ("Automatic backups: off, as in the backup").

21. **low**, 30-backup-page.png / 31b-backup-send-to-homebox-dialog.png: The phone shows only a manual "Send a backup to homebox" that asks for a new passphrase each time. Yet homebox lists pixel-8 backups every day at 08:12. I can't tell from the phone whether it backs up by itself. Suggestion: show "Automatic: on, daily, last sent <time>" on the phone's backup page.

22. **low**, w10-vps-new-machine-desktop.png: The fresh machine with no devices shows a green "Connected" chip and toggle. Suggestion: say "Running, no devices yet".

## Task 3: the router got a GL firmware update. Is tailbond still there?

**Done?** Yes. The top card and the Updates card say "tailbond is back after the firmware update, with your settings… Nothing to do." The version is 0.2.0 and the connection is up. Screens: r06, r01, r13.

Findings:

23. **medium**, r06-router-reinstalled-after-firmware-upgrade-desktop.png: Between the firmware reboot and tailbond reinstalling itself, tailbond (and its kill switch) wasn't running. The card doesn't say whether my laptop's work traffic went out over hotel Wi-Fi during that gap, or for how long. Suggestion: state it ("tailbond was not running from 21:31 to 21:36; devices here used the hotel's internet directly"). Or, if the kill-switch rules were kept, say "devices were blocked, not exposed".

24. **medium**, r06-router-reinstalled-after-firmware-upgrade-desktop.png: The "without Keep Settings" warning only appears after a successful upgrade, which is too late. It also tells me to "install tailbond again and pair again" instead of restoring my backup from homebox, which r01 says exists "so you can get it back if this router is lost or reset". Suggestion: show the Keep Settings warning permanently in the Updates card or near the backup line ("Upgrading GL firmware? Keep Settings must stay ticked"). Change the recovery advice to "install tailbond, then Restore from a backup (your copy on homebox)".

25. **low**, r06-router-reinstalled-after-firmware-upgrade-desktop.png: The top card says "Up 2 h 13 min", but the reinstall happened at 21:36, 24 minutes before now. That makes me doubt either the reinstall or the uptime. Suggestion: reset or label the uptime ("tunnel up since 21:37").

26. **low**, r06-router-reinstalled-after-firmware-upgrade-desktop.png: "from the copy kept on this router (or, if that was gone, from the GitHub release)". It hedges instead of saying which one happened. Suggestion: say which source was used, and whether the version changed.

## Overall

- high: 2
- medium: 12
- low: 12
