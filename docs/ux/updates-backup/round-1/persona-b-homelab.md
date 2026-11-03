# Round 1: persona B, "homelab"

Persona: runs a Fedora home server and an Ansible-managed rented VPS, uses Tailscale daily, knows dnf/COPR, systemd, TLS and supply-chain basics, and wants to know exactly how each device is updated and verified, what happens when an update breaks, what a backup holds (secrets), and what changes when the server moves to a new machine with a new address.

Screens used:
- GL panel: r01, r02, r03, r04, r05, r06, r07, r09, r10, r11, r13, s01 (desktop)
- Web dashboard: w01, w02, w03, w04, w05, w06, w07, w08, w09, w10, w11, w12, w13 (desktop)
- Android: 02, 10, 11, 12, 19, 30, 31b, 34, 37

I skipped the phone-width variants after seeing the desktop versions. I also skipped r08, 01, the dark variants, 13, 31, 32, 33, 33b, 35 and 36: they repeat states I had already seen, and these tasks don't depend on them.

---

## Task 1: Make sure everything stays up to date

**Done?** Partly. I can read each device's version: homebox 0.2.0 (w01/w02), slate7 0.1.0 or 0.2.0 (r01/r02), the phone (10/11), and the VPS (w03). I can see which update path each one uses. But I can't tell whether the Fedora box actually updates by itself, whether the phone is managed by Obtainium, what version the offline router (beryl) runs, or how any download is verified.
Screens: w01, w02, w03, w04, w08, w09, w13, s01, r01, r02, r03, r04, r05, r07, 02, 10, 11, 12, 19.

### Findings

1. **medium**: `w03-vps-update-available-desktop.png`, `w04-vps-update-confirm-desktop.png`, `r07-router-update-confirm-desktop.png`, `11-updates-available.png`
   No screen says where the VPS and router updates are downloaded from, or how they are checked (checksum, signature, which key) before a root-level network daemon replaces itself. On the phone, nothing says that Android only installs an update signed with the same key as the installed app. "Checks GitHub once a day" is all you get.
   *Suggestion:* add one line under "Update available": "Downloaded from github.com/kaovilai/tailbond releases; checked against SHA256SUMS signed with key `<fingerprint>` before install." Repeat it in the confirm dialog. On Android, show the release signing-certificate SHA-256 so it can be pinned in Obtainium or AppVerifier.

2. **medium**: `w01-homebox-dnf-update-available-desktop.png`
   The card says "Checks GitHub once a day" and "Update available: 0.2.1", but updates actually come from dnf, and it never names the repo (COPR? which one?) or its GPG key. If the repo lags behind the GitHub release, `sudo dnf upgrade tailbond-server` returns "Nothing to do" while the card still says an update is available.
   *Suggestion:* show the repo id and GPG key fingerprint, and what dnf would install ("dnf repo `copr:…/tailbond` has 0.2.1"). If the repo doesn't have the GitHub version yet, say "0.2.1 released on GitHub, not in your dnf repo yet".

3. **medium**: `w01-homebox-dnf-update-available-desktop.png`, `w02-updated-devices-out-of-date-backups-desktop.png`
   "Automatic updates come from dnf-automatic" doesn't say whether dnf-automatic is enabled on this box, so "does it update by itself?" has no answer. Nothing says the "goes back by itself if the new version doesn't start" promise (w04, r07) does not cover a dnf update, or what to do if a dnf update breaks tailbond.
   *Suggestion:* detect `dnf-automatic.timer` (or `dnf5-automatic`) and show "dnf-automatic: on, applies updates" or "off". Add a line: "No automatic undo for dnf updates; to go back: `sudo dnf downgrade tailbond-server` or `dnf history undo last`."

4. **medium**: `w01-homebox-dnf-update-available-desktop.png`, `s01-flint-devices-out-of-date-desktop.png`, `w03-vps-update-available-desktop.png`
   "2 devices out of date" lists slate7 and pixel-8 but leaves out beryl, which is offline. I can't tell whether beryl is current or just unknown. On the VPS, "All 3 devices are on 0.1.0 or newer" gives a floor, not each device's version.
   *Suggestion:* always list every device with its version and when it was last seen, e.g. "beryl: 0.1.0 when last seen Oct 31 (offline)". Count unknown devices separately instead of dropping them.

5. **medium**: `w01-homebox-dnf-update-available-desktop.png`, `w02-updated-devices-out-of-date-backups-desktop.png`, `s01-flint-devices-out-of-date-desktop.png`
   For phones the server says "Obtainium installs it automatically", yet pixel-8 is listed as out of date. Either Obtainium isn't set up or it isn't working, and the server can't know which. The wording suggests a self-updating phone that may not exist.
   *Suggestion:* "Phones: update in the app (Settings → Updates), or with Obtainium if you set it up."

6. **medium**: `w09-vps-check-failed-desktop.png`
   The chip and headline say "Up to date" in green while the line below says the last check failed (`api.github.com: i/o timeout`). The last successful check isn't shown. The Android app handles this case correctly (`12-updates-check-failed.png`: "Couldn't check", plus when the last good check was).
   *Suggestion:* use a neutral or amber chip ("Unknown, last check failed") and "Last good check: <time>", as on Android.

7. **medium**: `w08-vps-rolled-back-desktop.png`, `r04-router-rolled-back-desktop.png`
   The rollback message has no timestamp and no link to logs. "Update now" is offered again with no hint that the same failure is likely. It isn't clear whether Automatic would retry 0.2.0 every day. On a hotel link, "the tunnel didn't come back within 60 s" may be the network's fault, not the release's, and the screen doesn't help tell the two apart.
   *Suggestion:* add "at <time>", a "See logs" link (to the Logs tab or `journalctl -u tailbond` lines), and a "Skip 0.2.0" option. State whether Automatic retries this version or waits for the next one.

8. **medium**: `r05-router-automatic-desktop.png`
   Automatic installs "by itself" and devices drop for a few seconds, but it doesn't say when. A travel router dropping in the middle of a call is the obvious bad case.
   *Suggestion:* show the install window ("installs between 03:00 and 05:00 local time") and make it configurable.

9. **low**: `r02-router-update-available-desktop.png`, `02-settings-update-available-row.png`
   "Update available" appears only in the Updates card at the very bottom of a long GL page, or at the bottom of the phone's long Settings page. The top status card says nothing.
   *Suggestion:* add a one-line "Update available: 0.2.0" to the top status card in the GL panel and on the app's main screen.

10. **low**: `r01-router-up-to-date-desktop.png`, `r02-router-update-available-desktop.png`, `11-updates-available.png`
    Version strings differ between places: the footer says "0.2.0-r1" or "0.1.0-r42", the card says "0.2.0", and Android says "v0.2.0" next to "0.2.0". When comparing against `opkg list-installed` or a release tag, it isn't clear which one is authoritative.
    *Suggestion:* use one format everywhere, and show the package revision in the card ("0.2.0 (package 0.2.0-r1)").

11. **low**: `w01-homebox-dnf-update-available-desktop.png`, `w02-updated-devices-out-of-date-backups-desktop.png`
    The tab badge shows "3" or "2" with no explanation (1 update + 2 devices? 2 devices?).
    *Suggestion:* add a tooltip ("1 update, 2 devices out of date"), or use a dot instead of a count.

12. **low**: `19-updates-debug-build.png`
    "Download update" is still the main button on a debug build, even though the text below says the release APK won't install over it.
    *Suggestion:* replace the button with "Save a backup first" and link to the switch steps.

13. **low**: `w01-homebox-dnf-update-available-desktop.png`, `s01-flint-devices-out-of-date-desktop.png`
    "Routers: Update now in their tailbond panel" gives no link or address for that panel.
    *Suggestion:* make each router name a link to its GL admin page (LAN or tailnet address).

---

## Task 2: Move the home server to a new machine using a backup

**Done?** Partly. I can download a backup (w05), see what's in it (w12), restore it on the new VPS (w06, w07), and the restore preview lists what changes. But the screens give conflicting answers on whether the travel router and phone will find the new address. They never say to shut down or wipe the old server, and the final "Restored" message drops the required follow-up steps. The screenshots only show a VPS-to-VPS move. Nothing covers moving the Fedora/dnf homebox (install the RPM first, firewall, Tailscale login).
Screens: w02, w05, w06, w07, w10, w11, w12, r10, r11, r13, 02, 30, 31b, 34, 37.

### Findings

1. **high**: `w06-restore-preview-new-vps-desktop.png`, `w07-restore-done-desktop.png`
   Nothing tells you to stop, unpair or wipe the old server. After the restore, two machines hold the same server keys (`global.key`, `server.key`, w12) and the same device list. The old one is a rented VPS that will be handed back with those keys on disk. The router restore (r10) does warn "Both can't use it at once… turn tailbond off there first"; the server restore doesn't.
   *Suggestion:* add to "Before you restore": "Stop tailbond on the old server (203.0.113.9) before restoring, and wipe /etc/tailbond (or the bundle's state directory) before giving the machine back." Optionally offer "rotate server keys after restore" if devices can re-pair.

2. **high**: `w06-restore-preview-new-vps-desktop.png`, `r13-router-whats-inside-desktop.png`
   w06 says devices will find the new address "by themselves within a few hours, because vps has a Tailscale name (vps.tail1234.ts.net)". But slate7's backup contents show it stores the server as a raw IP, `vps (203.0.113.9:8443)`. Tailscale isn't in the backup (r10), so the new box joins the tailnet as a new node, and while the old node exists it gets a name like `vps-1`, not `vps`. Following the screens, the travel router can end up pointing at the old machine or at nothing while I'm travelling. No router or phone screen shows which address it is using or that it switched.
   *Suggestion:* in the preview, list each device with how it reaches the server (Tailscale name / DNS name / IP only) and what it needs. Add a step: "Remove the old `vps` node in the Tailscale admin and name this machine `vps` before restoring." After the restore, show a status for each device ("slate7: reached new address at 22:14"). Replace "within a few hours" with the real interval.

3. **high**: `w07-restore-done-desktop.png`
   The last thing you see says "Devices reconnect by themselves too". It leaves out the two required steps from w06: set the new public address (the restore just wrote the old 203.0.113.9 into this machine), and send new links to devices paired by QR or link. Anyone who closes w06 and trusts w07 leaves those devices broken.
   *Suggestion:* turn the done dialog into a checklist: "1. Set public address (now 203.0.113.9, the old machine's). 2. New links for: <devices>." Keep a banner on the dashboard until the address matches this machine.

4. **medium**: `w06-restore-preview-new-vps-desktop.png`
   The restore changes the tunnel port from 443 to 8443 and sets the public address to the old machine's IP. Nothing mentions the firewall or the cloud security group, which must allow UDP 8443 on the new box. There's no option to keep this machine's address.
   *Suggestion:* offer "Keep this machine's address (198.51.100.20)" as a choice. Add "Open UDP 8443 on this machine (`firewall-cmd --add-port=8443/udp --permanent`) and in your provider's firewall."

5. **medium**: `w06-restore-preview-new-vps-desktop.png`, `w07-restore-done-desktop.png`, `r11-router-restore-done-desktop.png`
   Timestamps don't agree. The file is `vps-2026-11-01T0300.tbbackup` but "made Nov 2, 16:00". The before-restore copy is named `…2026-11-02T2201` while the clock reads Nov 3, 22:00. When picking the newest backup during a migration, it isn't clear which time to trust.
   *Suggestion:* use one timezone everywhere, label it (UTC or local, with offset), and build the file name from the same timestamp shown as "made".

6. **medium**: `w10-vps-new-machine-desktop.png`
   The "new machine before restoring" can't be told apart from an established server. It shows uptime 5d, devices, traffic, "last update … worked", "All 3 devices…", and a full Saved backups history. Before a destructive restore I can't confirm I'm on the right box.
   *Suggestion:* in the restore preview, state the target plainly: "Restoring onto: <hostname>, 198.51.100.20, tailbond 0.2.0, 0 devices." On a fresh install, show empty states (no devices, no backups).

7. **medium**: `w12-whats-inside-desktop.png`, `r13-router-whats-inside-desktop.png`
   "What's in a backup" lists what's in it, not what's left out. Tailscale state, the package or dnf repo, firewall rules, systemd overrides, and the device backups stored on the server are not mentioned. Encryption is just "locked" (no cipher or KDF). The router's "Router settings (UCI): included" doesn't say which UCI config (Wi-Fi keys? GL admin password?).
   *Suggestion:* add a "Not in the backup, redo on the new machine" list (install package, Tailscale login, firewall). Add one line on crypto ("age / scrypt" or whatever is used). List which UCI sections are included.

8. **medium**: `w11-scheduled-backups-change-desktop.png`, `w02-updated-devices-out-of-date-backups-desktop.png`
   Automatic backups mean the server keeps the passphrase ("A passphrase is set. Leave the fields empty to keep it"), next to the encrypted files on the same disk. That isn't disclosed. The on-disk location isn't shown either, so I can't pick the files up with restic or an Ansible fetch, which "Keep a copy somewhere other than this machine" asks me to do.
   *Suggestion:* "The passphrase is stored on this server (root-only) so it can make backups unattended." Show the directory path (e.g. `/var/lib/tailbond/backups/`).

9. **medium**: `w07-restore-done-desktop.png`, `r11-router-restore-done-desktop.png`
   After a restore, the page behind the dialog shows "Automatic backups: Off". The schedule (and for routers, the copy to homebox) didn't come back, and neither done dialog says so.
   *Suggestion:* in the done dialog: "Automatic backups are off on this machine; turn them on (needs a passphrase)", with a button.

10. **medium**: `30-backup-page.png`, `31b-backup-send-to-homebox-dialog.png`, `w02-updated-devices-out-of-date-backups-desktop.png`
    The phone says "homebox keeps the last 3", but homebox's Saved backups list shows no pixel-8 entries. I can't check that they exist, or whether they (and the routers' copies) move with the server backup.
    *Suggestion:* list phone backups in Saved backups. In "What's inside", say whether the devices' stored backups are included.

11. **low**: `r10-router-restore-preview-desktop.png`, `w06-restore-preview-new-vps-desktop.png`
    "The current settings are saved first (same passphrase)" doesn't say same as what (the backup's passphrase? the scheduled one?). The router's Restore button is pale pink and looks disabled.
    *Suggestion:* "…locked with the passphrase you just typed." Give the Restore button the same solid style as on the web dashboard.

---

## Task 3: The router got a GL firmware update. Is tailbond still there?

**Done?** Yes. r06 says the firmware removed tailbond, it reinstalled itself with my settings, it's now on 0.2.0, and I had nothing to do. What it doesn't say is where the package came from, how it was verified, or what happened to traffic in between.
Screens: r06, r01.

### Findings

1. **medium**: `r06-router-reinstalled-after-firmware-upgrade-desktop.png`
   The message doesn't say where the reinstalled package came from (local copy? GitHub?), how it was verified, or whether it reinstalled the same version or the latest one. The policy is "Tell me", so a silent version change during reinstall would break that promise.
   *Suggestion:* "Reinstalled 0.2.0 (the version you had) from <source>, checksum verified." If it installed a newer version, say so and why.

2. **medium**: `r06-router-reinstalled-after-firmware-upgrade-desktop.png`
   The notice appears only in the Updates card at the bottom of the page. The top status card looks normal. Nothing says how long tailbond was missing after the flash, or whether the LAN was blocked or went out unprotected during that time (kill switch on).
   *Suggestion:* show it in the top status card too, with "tailbond was missing from 21:31 to 21:36; devices here were blocked (kill switch)", or whatever actually happened.

3. **medium**: `r06-router-reinstalled-after-firmware-upgrade-desktop.png`
   "It does this after every firmware update" doesn't cover a firmware upgrade without GL's "Keep settings", or a GL reset. That's exactly when the copy on homebox matters.
   *Suggestion:* add "If you upgraded without Keep settings, install tailbond again and use Restore from a backup (homebox has a copy from <time>)."

4. **low**: `r06-router-reinstalled-after-firmware-upgrade-desktop.png`
   "Checked 2 h ago" is older than the 21:36 reinstall, so it isn't clear whether "Up to date" was checked against the latest release after the reinstall.
   *Suggestion:* run a check right after a reinstall and show that time.

---

## Overall count

- **high:** 3
- **medium:** 18
- **low:** 7
