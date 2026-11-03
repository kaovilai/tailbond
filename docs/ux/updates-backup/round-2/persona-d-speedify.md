# Round 2: persona D, "Speedify user"

Persona: years of Speedify (silent app-store updates) and Peplink/SpeedFusion at work (scheduled firmware, one-file config backup/restore onto new hardware). Expects one version number, one click to update, one click to back up, and a restore that just works on a new box. Counts every extra step.

Screens used:
- Web dashboard: w01 (desktop + phone), w02, w03, w04, w05, w06 (desktop + phone), w07, w08, w09, w10, w11, w12, w13
- GL.iNet panel: r01, r02 (desktop + phone), r03, r04, r05, r06, r07, r08, r09, r10 (desktop + phone), r11, r13, s01, s02
- Android: 01, 03, 10, 11, 12, 19, 30, 31b, 34, 37

---

## Task 1: Make sure everything stays up to date

**Done?** Partly. I can read each version (home server card "This server runs tailbond 0.2.0", router card "This router runs tailbond 0.2.0", phone "tailbond 0.2.0"), and the home server lists device versions. But updating the Fedora server needs a terminal, the routers have to be updated one by one from their own panels, the phone needs a third-party app to update by itself, and the VPS screens mark out-of-date devices as up to date.
Screens: w01, w02, w03, w04, w08, w09, w13, r01, r02, r03, r04, r05, r07, s01, 01, 03, 10, 11, 12.

**Findings**

1. **medium**: `w01-homebox-dnf-update-available-desktop.png`, `w02-updated-devices-out-of-date-backups-desktop.png`
   The Fedora home server has no "Update now". I have to open a terminal and type `sudo dnf upgrade tailbond-server`, and to get automatic updates I have to run `sudo systemctl enable --now dnf-automatic.timer`. The VPS and the routers each update with one click. Speedify and Peplink never send me to a shell.
   Suggestion: add an "Update now" button that goes through PackageKit/polkit, plus an "Automatic" choice that turns on the timer for me. Keep the commands as a fallback under "Prefer the terminal?".

2. **medium**: `w02-updated-devices-out-of-date-backups-desktop.png`
   "Tell me (recommended)" is selected, yet the next line says "Automatic updates: on (dnf-automatic)". There's no "Automatic" button on this machine, so two controls contradict each other and I can't tell which one wins.
   Suggestion: when dnf-automatic is on, show the policy as "Automatic (via dnf-automatic)" with a "Turn off" link, and disable "Tell me" or explain that it only covers the notice.

3. **medium**: `w01-homebox-dnf-update-available-desktop.png`
   The screen offers `dnf-automatic.timer` as "Automatic updates" for tailbond. It doesn't say that this timer covers every package on the machine, or whether it installs or only downloads. I could turn on unattended OS-wide updates without knowing it.
   Suggestion: say plainly "This updates all Fedora packages on this machine, not just tailbond", and state what the default does (install vs. download only), or ship a tailbond-only timer.

4. **medium**: `w03-vps-update-available-desktop.png`, `w08-vps-rolled-back-desktop.png`, `w09-vps-check-failed-desktop.png`, `w13-vps-updates-off-desktop.png`
   slate7 and pixel-8 run 0.1.0 and show a green "✓ up to date" while the same card says 0.2.0 is out. On w13 updates are off, so the server can't know the latest version, but the checks are still green. "Up to date" here seems to mean "matches this server", which is not what the words say. I'd skip updating my devices.
   Suggestion: compare devices against the newest known release and label them "Same as this server (0.1.0)". Show green "Up to date" only when a device matches the latest release, and show "unknown" when checks are off or failing.

5. **medium**: `s01-flint-devices-out-of-date-desktop.png`, `w01-homebox-dnf-update-available-desktop.png`
   The home server knows slate7 and pixel-8 are "Out of date" but can only tell me "Routers: Update now in its tailbond panel". There's no button and no link to that panel. Peplink InControl lets me push firmware to the fleet from one place. Here I have to go to each router's admin page myself.
   Suggestion: add "Update slate7 now" (or "Ask slate7 to update") next to each out-of-date router. At minimum, link to the router's panel.

6. **medium**: `10-updates-up-to-date.png`, `11-updates-available.png`
   The phone can't update itself. "Download update" hands an APK to the browser, Android asks me to allow the browser to install apps, and for automatic updates I have to install another app (Obtainium) and type in a GitHub address. The phone policy has only Tell me / Off, while the server and router have Tell me / Automatic / Off. Speedify just updates from the Play Store.
   Suggestion: publish to Google Play (or F-Droid) and say so first. Until then, add an "Open in Obtainium" button that pre-fills the address, and show "Automatic (Obtainium)" as a third policy choice once Obtainium is detected.

7. **medium**: `r01-router-up-to-date-desktop.png`
   The policy is "Tell me (recommended)" with "nothing installs until you click Update now", but "last update: 0.2.0, Nov 3, 03:06, worked" shows an install at 3 a.m. Either something installed by itself or the line is wrong. Both make me distrust the policy.
   Suggestion: say who triggered the last update ("by you", "automatically", "after a firmware upgrade") in the "last update" line.

8. **low**: `w01-homebox-dnf-update-available-desktop.png`, `w02-updated-devices-out-of-date-backups-desktop.png`, `w03-vps-update-available-desktop.png`
   The "Updates & backup" tab badge shows 3, 2 or 1 with no explanation of what it counts.
   Suggestion: add a tooltip/aria label such as "1 update, 2 devices out of date", or show a dot instead of a number.

9. **low**: `r01-router-up-to-date-desktop.png`, `r02-router-update-available-desktop.png`, `11-updates-available.png`
   The same version is written three ways: "0.2.0" in cards, "v0.2.0" on Android, and "tailbond 0.2.0-r1" / "0.1.0-r42" in the GL footer. "Clear version number" fails when the router footer and card disagree.
   Suggestion: use one format everywhere ("0.2.0") and put the package revision only in a details line.

10. **low**: `w01-homebox-dnf-update-available-desktop.png`, `s01-flint-devices-out-of-date-desktop.png`
    beryl shows "version not known yet", but it has a saved backup from Oct 31, so it has clearly talked to this server. "A version shows once the device has connected with this version of tailbond" doesn't say which version or what to do.
    Suggestion: "beryl hasn't connected since this server was updated (last seen Oct 31). Its version shows next time it connects."

11. **low**: `r01-router-up-to-date-desktop.png`, `s01-flint-devices-out-of-date-desktop.png`, `01-settings-updates-and-backup-rows.png`
    The version and update status sit at the very bottom of long pages (below Tailnet machines on the GL panel; the last row of a 5,000-px Android Settings page). On the Flint 3, the Devices card at the top lists devices with no versions, and versions appear only in the Updates card near the bottom.
    Suggestion: put "tailbond 0.2.0 · Up to date" in the top status card on every device, and add the version to each row of the Devices card.

12. **low**: `r05-router-automatic-desktop.png`
    "Installs between 03:00 and 05:00 (this router's time)." A travel router changes time zones and is often unplugged overnight. The screen doesn't say what happens if it's off during the window.
    Suggestion: add "If the router is off then, it installs the next time it's on between 03:00 and 05:00", and show the next planned install time.

13. **low**: `r04-router-rolled-back-desktop.png`
    The router is on "Tell me", but the note says "Automatic won't install 0.2.0 again", which doesn't apply to this policy. There's also no link to why 0.2.0 failed.
    Suggestion: show the Automatic sentence only when the policy is Automatic, and add "See log" next to "Try 0.2.0 again".

14. **low**: `w09-vps-check-failed-desktop.png`, `12-updates-check-failed.png`
    The web shows an amber "Couldn't check" with no last good check time. Android shows the same state as a red alarm card with "Last good check yesterday at 09:15". The two platforms disagree on severity and detail.
    Suggestion: on the web, add "Last good check: …"; on Android, use the calm amber style the web uses.

---

## Task 2: Move the home server to a new machine using a backup

**Done?** Partly. I can download a backup (w05) and restore it on a new machine (w06, w07), but only the VPS-to-VPS case is shown. There's no screen for moving the Fedora homebox or the Flint 3. After restoring, the new server has the old public address and port, and some devices need re-pairing, so "your devices keep working" is not true without extra manual steps.
Screens: w02, w05, w06 (desktop + phone), w07, w10, w11, w12, s01, s02, r10, r11, r13, 30, 31b, 34, 37.

**Findings**

15. **high**: `w06-restore-preview-new-vps-desktop.png`, `w07-restore-done-desktop.png`
    Restoring onto the new machine (198.51.100.20, port 443) sets the public address to the old 203.0.113.9 and the tunnel port to 8443. Unless I go to the Devices tab and fix the address, and open a new firewall/port-forward rule, every device keeps dialing the old box. Devices without Tailscale need a new link anyway. The card I started from promised "Move to a new machine and your devices keep working". On Peplink, restoring onto replacement hardware keeps peers connecting; here the default result is a server that no device can reach.
    Suggestion: in the restore preview, add a choice for address and port, defaulting to "Keep this machine's address and port (198.51.100.20, 443)" with "Use the backup's" as the alternative. Push the new address to devices that can learn it. Remove "your devices keep working" unless that is true by default.

16. **medium**: `w06-restore-preview-new-vps-desktop.png`
    The warning splits devices into "with Tailscale" (follow by themselves) and "without Tailscale" (need their link again), but nothing tells me which of slate7, pixel-8 and beryl falls in which group. pixel-8 has "Hand the VPN back to Tailscale" in its settings, so I can't tell whether it counts as a Tailscale device while tailbond holds Android's VPN slot.
    Suggestion: list each device by name in the preview: "slate7: follows by itself", "pixel-8: needs a new link (QR)", "beryl: offline; will follow when it connects".

17. **medium**: `w06-restore-preview-new-vps-desktop.png`
    "as long as this machine has the old one's Tailscale name (vps)": nothing says how to get that name. Tailscale gives a second machine "vps-1", and I'd have to remove the old one in the Tailscale admin console first. The screen doesn't show this machine's current Tailscale name, so I can't check.
    Suggestion: show "This machine's Tailscale name: vps2 (needs to be vps)" in the preview, with a link and the two steps (remove or rename the old one, then rename this one). Re-check after restore.

18. **medium**: `w07-restore-done-desktop.png`
    "Next steps" pastes the pre-restore warnings word for word, including "Turn tailbond off on vps first", which is too late now. There are no buttons and no status. I can't see whether the address was changed, whether the port is reachable, or which devices have reconnected.
    Suggestion: turn it into a checklist with live state: "Old vps still answering? (yes/no)", "Public address: 203.0.113.9 [Change]", "udp/8443 reachable from outside: not yet [Test]", and "Devices back: 1 of 3" with a QR button for each device that hasn't reconnected.

19. **medium**: `w02-updated-devices-out-of-date-backups-desktop.png`, `w10-vps-new-machine-desktop.png`
    The old machine has no "Move to a new machine" path. The key instruction ("turn tailbond off on the old one first") appears only on the new machine, after I've uploaded the backup. I also only see this flow for a VPS. There's no screen for the Fedora homebox (dnf) or the Flint 3 home server (s01/s02), where the port forward on my ISP router points at the old box's LAN address.
    Suggestion: add "Move to a new machine" on the old server. It makes the backup, lists the steps for this kind of server (Fedora/VPS/Flint), and then offers "Turn tailbond off here" once the new one is up.

20. **medium**: `w05-backup-download-dialog-desktop.png`, `w11-scheduled-backups-change-desktop.png`, `w02-updated-devices-out-of-date-backups-desktop.png`
    There are too many passphrases. "Download a backup" asks for a new passphrase every time. Automatic backups use a different, stored passphrase I may have set months ago. Each device's backup carries its own. The "Download" links in Saved backups don't say which passphrase opens them. That isn't the one-click backup I expect.
    Suggestion: label each saved backup with its passphrase ("automatic-backup passphrase" or "pixel-8's own"). Offer "Download latest automatic backup" as the one-click option, and pre-select "use my automatic-backup passphrase" in the download dialog.

21. **medium**: `30-backup-page.png`, `31b-backup-send-to-homebox-dialog.png`
    homebox lists daily pixel-8 backups at 08:12, but the phone's backup page shows no automatic setting, no schedule and no "last sent". "Send a backup to homebox… Handy for a new phone", but there's no "Restore from homebox". Restore only takes a file, and nothing says how to get pixel-8's file off homebox onto a new phone.
    Suggestion: show "Automatic: sends to homebox daily, last sent today 08:12 [Change]". Add "Restore from homebox" that lists homebox's copies of this phone's backups.

22. **low**: `w07-restore-done-desktop.png`, `r11-router-restore-done-desktop.png`, `r10-router-restore-preview-desktop.png`
    "A copy of the previous settings was kept … locked with the same passphrase." Which passphrase? On w10 the new machine had automatic backups off, so it had no passphrase of its own.
    Suggestion: "locked with the passphrase you just typed for this backup."

23. **low**: `w12-whats-inside-desktop.png`, `s02-flint-whats-inside-desktop.png`, `r10-router-restore-preview-desktop.png`
    Raw internal values show through: "Updates: notify", "notify → auto", and "Router settings (UCI): no" on a Fedora server. The rest of the UI says "Tell me" and "Automatic".
    Suggestion: use the UI labels ("Tell me", "Automatic") and hide the UCI row on non-router machines.

24. **low**: `w06-restore-preview-new-vps-desktop.png`, `w06-restore-preview-new-vps-phone.png`
    On a phone, the "What it replaces" table is cut off below the warnings ("slate7, pixel-8, beryl" is clipped), while the red Restore button stays in view. It's easy to restore without reading what gets replaced.
    Suggestion: keep Restore disabled until the list has been scrolled to the end, or put the replace table above the warnings in a compact form.

25. **low**: `w02-updated-devices-out-of-date-backups-desktop.png`, `s01-flint-devices-out-of-date-desktop.png`
    "keeps the last 7 on this server" while Saved backups shows 3 (Fedora) or 2 (Flint) of its own, and 3 per device. The two retention rules (7 for the server, 3 per device) aren't explained in one place.
    Suggestion: "Keeps 7 of this server's (3 so far) and the last 3 of each device's."

26. **low**: `34-restore-preview.png`
    The date is "30 Oct 2026, 15:45" on Android and "Nov 2, 03:00" on the web and router. Two formats for the same kind of fact.
    Suggestion: use one date format across all three surfaces.

---

## Task 3: The router got a GL firmware update. Is tailbond still there?

**Done?** Yes. r06 tells me in the top card and in the Updates card that tailbond is back with my settings, and says "Nothing to do." The warning about Keep Settings only appears afterwards, the failure case isn't shown, and the recovery advice conflicts with the backup screens.
Screens: r01, r06, r13.

**Findings**

27. **medium**: `r06-router-reinstalled-after-firmware-upgrade-desktop.png`, `r13-router-whats-inside-desktop.png`
    r06 says that if I upgrade without Keep Settings, I should "install tailbond again and pair again". r13 says that on a new router I should "install tailbond, sign in to Tailscale, then restore", and homebox keeps the last 3 slate7 backups for exactly this. Pairing again throws away my servers, rules and connection settings.
    Suggestion: "If you upgraded without Keep Settings: install tailbond again, then restore your latest backup (homebox keeps a copy)."

28. **medium**: `r01-router-up-to-date-desktop.png`, `r06-router-reinstalled-after-firmware-upgrade-desktop.png`
    The only warning before a firmware upgrade is grey small print under Automatic backups ("kept through firmware upgrades with Keep Settings"). GL's own firmware page won't mention tailbond. I find out about Keep Settings only after it's too late.
    Suggestion: add a line to the Updates card: "Upgrading GL firmware? Leave Keep Settings on and tailbond comes back by itself." Make a backup to homebox automatically when a firmware upgrade starts, if that can be detected.

29. **medium**: `r06-router-reinstalled-after-firmware-upgrade-desktop.png`
    Only the success case is shown. If the reinstall fails (no copy on the router and GitHub unreachable from hotel Wi-Fi behind a sign-in page), the VPN → tailbond page itself is presumably gone, so nothing in the router panel would tell me. I'd find out when the tunnel doesn't come up.
    Suggestion: have homebox flag "slate7 hasn't connected since [time], possibly after a firmware upgrade" with the reinstall steps. Document what the GL panel shows in that case.

30. **low**: `r06-router-reinstalled-after-firmware-upgrade-desktop.png`
    "reinstalled itself from the copy kept on this router (or, if that was gone, from the GitHub release)." The router knows which one happened, but the text hedges, and a GitHub reinstall could mean a different version than before.
    Suggestion: say which source was used and which version: "Reinstalled 0.2.0 from the copy on this router."

31. **low**: `r06-router-reinstalled-after-firmware-upgrade-desktop.png`
    The green "back after the firmware update" banner has no dismiss control and doesn't say how long it stays, so it may become noise.
    Suggestion: add a close button, or say "shown for 24 h".

---

## Overall count

- **High:** 1
- **Medium:** 16
- **Low:** 14
