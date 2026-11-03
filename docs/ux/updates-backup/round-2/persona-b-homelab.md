# Round 2: persona B, "homelab"

Runs a Fedora home server and a rented VPS (Ansible-managed), uses Tailscale daily, knows dnf/COPR, systemd, TLS and supply-chain basics, and wants to know exactly how each device is updated and verified, how a bad update is undone, what a backup holds, and what changes when the server moves.

Screens used:
- web: w01, w01-phone, w02, w03, w04, w05, w06, w06-phone, w07, w08, w09, w10, w11, w12, w13
- glinet: r01, r02, r04, r05, r06, r07, r10, r11, r13, s01, s02
- android: 01, 02, 03, 10, 11, 12, 19, 30, 31b, 34, 37

## Task 1: Make sure everything stays up to date

**Done?** Partly. I can read each device's version (homebox 0.2.0 in w02, slate7 0.1.0 in r02, pixel-8 0.1.0 in android 11) and see who installs what: dnf on Fedora, one click or Automatic on the VPS and routers, browser or Obtainium on the phone. But I can't trust the "up to date" labels on the VPS screens. On Fedora I can't tell what "Automatic updates: on" really does. And the verification story stops at a checksum.

Screens: w01, w02, w03, w04, w08, w09, w13, r01, r02, r04, r05, r07, s01, android 01, 02, 03, 10, 11, 12, 19.

**Findings**

1. **high**, `w01-homebox-dnf-update-available-desktop.png` / `w02-updated-devices-out-of-date-backups-desktop.png`: The "turn automatic updates on" command is `sudo systemctl enable --now dnf-automatic.timer`. That timer acts on every package on the box, kernel included, not just tailbond. With Fedora's stock `automatic.conf` (`apply_updates = no`) it only downloads, so w02's "Automatic updates: on (dnf-automatic)" may claim tailbond auto-installs when nothing gets installed. Either way I'm making an unscoped, system-wide change from a tailbond card.
   *Suggestion:* Say plainly that this turns on unattended upgrades for the whole system. Show whether `apply_updates` is on, and base the "on" status on that, not just on the timer being active. Also offer a tailbond-only option, for example a drop-in `automatic.conf` limited to `tailbond-server`, or a tailbond-owned timer running `dnf upgrade tailbond-server`.

2. **medium**, `w03-vps-update-available-desktop.png`, `w08-vps-rolled-back-desktop.png`, `w09-vps-check-failed-desktop.png`, `w13-vps-updates-off-desktop.png`: slate7 and pixel-8 on 0.1.0 show a green "✓ up to date" while the same screen says 0.2.0 is out. In w09 and w13 the server doesn't even know the latest version. The label means "same as this server", not "current", so it answers "is anything out of date?" wrongly.
   *Suggestion:* Compare against the latest release (as homebox does in w02 with "Out of date"), or label it "same as this server". Show "unknown" when the check failed or updates are off.

3. **medium**, `w01-homebox-dnf-update-available-desktop.png`: The banner learns about 0.2.1 from GitHub, but the install comes from dnf, and "reaches dnf shortly after the release" means `dnf upgrade` can say "Nothing to do" right after I click Show. The screen never names the repo (COPR? which project?) or says whether packages are GPG-signed and with which key.
   *Suggestion:* Show the repo id and the version dnf actually sees (as `dnf repoquery` would), and say "0.2.1 isn't in the dnf repo yet" until it is. Show the RPM signing key's fingerprint, or link to it.

4. **medium**, `w01-homebox-dnf-update-available-desktop.png`: On Fedora a bad update is not undone automatically, unlike the VPS and routers. The only hint is `sudo dnf downgrade tailbond-server`, which only works if the repo still has the old build. Nothing says whether 0.2.x's state or config changes survive going back.
   *Suggestion:* Say explicitly that Fedora has no automatic rollback. Say whether the repo keeps older builds and whether downgrading is safe for state written by the newer version. Point to a pre-update backup (or take one automatically before a version change is noticed).

5. **medium**, `w04-vps-update-confirm-desktop.png`, `r07-router-update-confirm-desktop.png`: "Checked against its SHA-256 list" proves the download wasn't corrupted, not who made it. The checksum list comes from the same GitHub release, so anyone who can replace the asset can replace the list. Automatic mode installs a root daemon on that basis alone.
   *Suggestion:* Sign releases (minisign, cosign or GPG) with a key pinned in the installed agent, verify the signature before installing, and say so here with the key fingerprint. If signing isn't there yet, say "checksum only, not signed" plainly.

6. **medium**, `11-updates-available.png`, `10-updates-up-to-date.png`: The phone flow tells me to let my browser install apps ("that's normal") and to add the GitHub repo to Obtainium. It gives no signing-certificate fingerprint to check on the first install or in Obtainium. It doesn't say to take the browser's install permission back afterwards. It has no route back from a bad phone update (Android won't downgrade over the old version).
   *Suggestion:* Show the release APK's SHA-256 certificate fingerprint (for AppVerifier or Obtainium). Add "turn this permission off again afterwards". Add one line on going back: back up, uninstall, install the older APK, restore (the same steps 19-updates-debug-build already explains).

7. **medium**, `w08-vps-rolled-back-desktop.png`, `r04-router-rolled-back-desktop.png`: "Went back to 0.1.0" gives no time, no log excerpt and no link to the failed attempt in Logs. The health check is "service/tunnel up within 60 s". On a travel router at a hotel at 03:00, a flaky uplink can fail a good release, and then Automatic never installs that version again.
   *Suggestion:* Show when it happened, the exit reason, and a "See the log of this attempt" link. Tell "the new version crashed" apart from "no uplink during the check", and don't block a version that failed only because the uplink was down.

8. **medium**, `r05-router-automatic-desktop.png`, `w03-vps-update-available-desktop.png`, `s01-flint-devices-out-of-date-desktop.png`: Routers can update themselves at night while the Fedora server waits for dnf (or the reverse). Nothing says whether a 0.2.x client works with a 0.1.x server, or which side should update first.
   *Suggestion:* State the compatibility rule (for example "clients and server work across one minor version"). Flag a device as "too new or too old for this server" in the device list when they don't.

9. **low**, `w01-homebox-dnf-update-available-desktop.png`, `w02-...`, `w03-...`: The tab badge reads 3, 2 or 1 with no hint of what it counts.
   *Suggestion:* Add a tooltip or aria label, for example "1 update, 2 devices out of date".

10. **low**, `r01-router-up-to-date-desktop.png`, `r02-router-update-available-desktop.png`, `r10-router-restore-preview-desktop.png`, `w12-whats-inside-desktop.png`, android `11`: Versions and policies are written in several ways: "0.2.0" in the card vs "0.2.0-r1" and "0.1.0-r42" in the footer, "v0.2.0" vs "0.1.0" on Android, and raw "notify"/"auto" vs "Tell me"/"Automatic".
    *Suggestion:* Use one version format everywhere (show the package revision once, labelled). Use the UI labels for policies in previews and summaries.

11. **low**, `w09-vps-check-failed-desktop.png`: "Checked 6 min ago" sits under "Couldn't check", so it's unclear when the last successful check was. Android 12 gets this right ("Last good check yesterday at 09:15").
    *Suggestion:* Use the Android wording: "Tried 6 min ago · last good check …".

12. **low**, `r05-router-automatic-desktop.png`: "Between 03:00 and 05:00 (this router's time)" on a travel router whose time zone may not follow me.
    *Suggestion:* Show the router's current time and time zone next to the window.

## Task 2: Move the home server to a new machine using a backup

**Done?** Partly. Making and restoring the backup is clear (w05, w06, w07, w12). But the only move shown is VPS to VPS by IP address. The instructions for keeping the travel router and phone connected depend on things the screens don't tell me how to do, and one path leaves the hotel router's clients offline. Nothing covers my actual case: a Fedora homebox reached as `home.example.net` behind a home-router port forward.

Screens: w01, w02, w05, w06, w06-phone, w07, w10, w11, w12, r01, r13, s01, s02, android 01, 30, 31b, 34, 37.

**Findings**

1. **high**, `w06-restore-preview-new-vps-desktop.png`, `r01-router-up-to-date-desktop.png`, `r13-router-whats-inside-desktop.png`: slate7 has no Tailscale of its own ("not this router's own Tailscale" in r01), so after the address change it is one of the devices that "need their link again". It sits at a hotel with "Kill switch: On (recommended)". Once the server moves, every client behind it loses internet, and re-pairing needs someone at the router's admin page. The warning mentions neither the kill switch nor doing this before the move.
   *Suggestion:* Add a step before the move: "while the old server still runs, push the new address to link-paired devices". Or tell people to keep a DNS name and repoint it. Name the kill-switch consequence in the warning, and list which paired devices are link-only, by name.

2. **high**, `w06-restore-preview-new-vps-desktop.png`, `w12-whats-inside-desktop.png`: Tailscale devices follow only "as long as this machine has the old one's Tailscale name (vps)". But the backup excludes the Tailscale login, and the old node still holds that name, so the new one comes up as `vps-1`. "Turn tailbond off on vps" doesn't free the name. Following the steps as written, Tailscale devices won't follow either.
   *Suggestion:* Spell out the steps: remove (or rename) the old node in the Tailscale admin console, then rename this one to `vps` (or `tailscale set --hostname`). Show this machine's current Tailscale name next to the required one, with a red/green check before Restore is enabled.

3. **medium**, `w06-restore-preview-new-vps-desktop.png`: The restore sets the public address to the old 203.0.113.9, though the dialog itself shows this machine is 198.51.100.20. I then have to remember to fix it in the Devices tab.
   *Suggestion:* Offer a choice in the preview, "Keep 198.51.100.20 (this machine)" or "Use 203.0.113.9 (from the backup)", with the right default when the addresses differ.

4. **medium**, `w06-restore-preview-new-vps-desktop.png`, `s01-flint-devices-out-of-date-desktop.png`: The tunnel addresses change from 10.77.0.0/24 to 10.83.142.0/24 with no warning. s01 shows "Optional route 10.77.0.0/24 approved" in Tailscale, so after the restore that approval no longer matches and tailnet-to-device traffic quietly stops.
   *Suggestion:* Add a 4th "thing to know" whenever the subnet changes: "approve route 10.83.142.0/24 in the Tailscale admin console (the old 10.77.0.0/24 approval no longer applies)".

5. **medium**, `w07-restore-done-desktop.png`: "Next steps" repeats "Turn tailbond off on vps **first**" after the restore has already run. It lives only in a dialog with a single Done button; after closing it, nothing on the tab reminds me.
   *Suggestion:* Before restoring, require a tick "vps is off or wiped". Afterwards, rewrite the items as remaining to-dos and keep them as a checklist on the Updates & backup tab until each one is dismissed.

6. **medium**, `w01-...`, `w02-...`, `w03-...`, `w10-vps-new-machine-desktop.png` (Backup & restore subtitle): "Move to a new machine and your devices keep working" contradicts the three caveats in w06 (a new address, re-linking, firewall and port forward).
   *Suggestion:* "Move to a new machine; devices follow if it keeps the address and Tailscale name, otherwise see the checks before restoring."

7. **medium**, `w11-scheduled-backups-change-desktop.png`, `w07-restore-done-desktop.png`, `w12-whats-inside-desktop.png`: The automatic-backup passphrase is "kept on this machine, in its settings", next to `/var/lib/tailbond/backups/`. So the local encrypted backups protect nothing against anyone who can read the disk. The screens don't give the settings file path or permissions. w07 shows the passphrase travels inside the backup ("same schedule and passphrase"), yet w12's list of what's inside doesn't mention it.
   *Suggestion:* Give the file path and mode (for example `/etc/tailbond/… 0600 root`). Add "automatic-backup passphrase" to the list of what's inside. Recommend a different passphrase for downloaded or off-site copies.

8. **medium**, `w05-backup-download-dialog-desktop.png`, `w12-whats-inside-desktop.png`, android `30-backup-page.png`: The file holds `server.key` and `global.key`, but no screen names the cipher, the key derivation, or a way to check or open the file without tailbond. I can't judge whether storing it on Google Drive, as Android suggests, is acceptable.
   *Suggestion:* Add one line on the format (for example "age, scrypt passphrase") and a CLI command to check or decrypt a `.tbbackup` offline.

9. **medium**, no screen (gap across w06 and s01): The only move shown is an IP-addressed VPS. For a home server reached as `home.example.net` behind a home router, the move means repointing the port forward to the new machine's LAN IP and possibly DDNS. The warning mentions port forwarding only when the port number changes.
   *Suggestion:* When the public address is a DNS name, say "keep the name, point it (and your router's UDP 443 forward) at the new machine; devices follow without re-linking". Add a fixture and screenshot for this case.

10. **low**, `w12-whats-inside-desktop.png`, `s02-flint-whats-inside-desktop.png`: A Fedora server shows "Router settings (UCI): no", and the Flint backup includes UCI with no word on whether it is applied when restoring onto a different model or a Fedora box.
    *Suggestion:* Hide the UCI row on non-OpenWrt machines. Say what happens to UCI when restoring across platforms.

11. **low**, `30-backup-page.png`, `31b-backup-send-to-homebox-dialog.png`: homebox lists a pixel-8 backup daily at 08:12, but the phone shows only a manual "Send a backup to homebox", with no schedule, no "last sent" and no word on which passphrase the automatic ones use.
    *Suggestion:* Show "Sent automatically every day · last sent today 08:12 · uses passphrase set on …" with a way to change it.

12. **low**, `w02-...`, `s01-...`: The server keeps 7 of its own backups and 3 per device, and the device limit can't be changed (w11 only covers the server's own).
    *Suggestion:* Show both limits in one place and let the device limit be changed.

## Task 3: The router got a GL firmware update. Is tailbond still there?

**Done?** Yes. r06 says it's back with my settings and that I didn't need to do anything. But it doesn't say how it came back or what was verified.

Screens: r06, r01, r07.

**Findings**

1. **medium**, `r06-router-reinstalled-after-firmware-upgrade-desktop.png`: "Reinstalled itself from the copy kept on this router (or, if that was gone, from the GitHub release)" describes what already happened as an either/or. I can't tell which source was used, which version was fetched, or whether the GitHub path got the same SHA-256 check as "Update now".
   *Suggestion:* State the actual source and version ("from the copy kept on this router, 0.2.0" or "downloaded 0.2.0 from GitHub, checksum OK").

2. **medium**, `r06-router-reinstalled-after-firmware-upgrade-desktop.png`: "Without Keep Settings, install tailbond again and pair again" ignores the backup that r01 says homebox holds "so you can get it back if this router is lost or reset". Re-pairing creates a new device identity, where restoring would keep the old one.
   *Suggestion:* "Without Keep Settings: install tailbond, then restore the backup homebox keeps for slate7 (no pairing needed)."

3. **low**, `r06-router-reinstalled-after-firmware-upgrade-desktop.png`: No GL firmware version before and after, and nothing on what survived the upgrade to trigger the reinstall (for example a `sysupgrade.conf` entry), which I'd want to check on my own.
   *Suggestion:* Show "Firmware 4.x → 4.y" and one line on what is kept through Keep Settings, with the paths.

## Overall

- high: 3
- medium: 16
- low: 8
