# Round 3: persona B, "homelab"

Persona: runs a Fedora home server and an Ansible-managed rented VPS, uses Tailscale daily, knows dnf/COPR, systemd, TLS and supply-chain basics, and wants to know exactly how each device is updated and verified, what happens when an update breaks, what a backup holds (secrets), and what changes when the server moves to a new machine with a new address.

Screens used:
- GL panel: r01, r02, r03, r04, r05, r06, r07, r09, r10, r11, r13, s01, s02 (desktop)
- Web dashboard: w01, w02, w03, w04, w05, w06 (desktop + phone), w07, w08, w09, w10, w11, w12, w13 (desktop)
- Android: 03, 10, 11, 12, 19, 30, 31b, 34, 37

---

## Task 1: Make sure everything stays up to date

**Done?** Partly. Each device shows its own version ("This server runs tailbond 0.2.0", "This router runs tailbond 0.1.0", "tailbond 0.2.0" on the phone), and the home server lists the devices' versions. How each one updates is mostly clear: dnf on Fedora, Update now/Automatic on the VPS and router, browser APK or Obtainium on the phone. But two screens wrongly say devices are up to date, and the verification and failure stories have gaps.
Screens: w01, w02, w03, w04, w08, w09, w13, r01, r02, r03, r04, r05, r07, s01, 03, 10, 11, 12, 19.

### Findings

1. **high**: `w09-vps-check-failed-desktop.png`, `w13-vps-updates-off-desktop.png`
   When the check failed (w09) or updates are off (w13), slate7 and pixel-8 on 0.1.0 show a green "✓ up to date", but 0.2.0 is out (w03). The server can't know what the latest version is, yet it answers the exact question this task asks ("is anything out of date?") with a confident wrong answer.
   *Suggestion:* With no successful check, show device versions with no verdict ("0.1.0 · can't compare: no check"). Or compare against the last known good check and say when it ran ("behind 0.2.0, as of yesterday 09:15").

2. **medium**: `w01-homebox-dnf-update-available-desktop.png`
   "Updates for this server come from dnf" never says which repo (COPR? which name?), whether packages are GPG-signed or gpgcheck is on, or what "shortly after the release" means. The banner comes from a GitHub check, so `sudo dnf upgrade tailbond-server` can return "Nothing to do" while the banner says "Update available". The screen doesn't explain that.
   *Suggestion:* Name the repo and its signing key fingerprint ("from COPR kaovilai/tailbond, signed, gpgcheck=1"). Base "Update available" on what dnf can actually see (repo metadata), or add "may take up to N hours to reach dnf; if dnf says Nothing to do, try `dnf upgrade --refresh` later".

3. **medium**: `w01-homebox-dnf-update-available-desktop.png`, `w02-updated-devices-out-of-date-backups-desktop.png`
   The dnf-automatic line asserts a state ("only downloads updates (Fedora's default)" / "installs updates for every package") but never says whether dnf-automatic is installed or its timer is enabled. On a stock Fedora server it usually isn't, so the line suggests updates happen when they don't. It also doesn't say whether the RPM restarts tailbond.service on upgrade, or that dnf has no automatic rollback.
   *Suggestion:* Show the detected state ("dnf-automatic.timer: not enabled", with `sudo systemctl enable --now dnf-automatic.timer`). Add one line: "the package restarts tailbond after an upgrade; dnf doesn't roll back by itself, use the downgrade command below".

4. **medium**: `r07-router-update-confirm-desktop.png`, `w04-vps-update-confirm-desktop.png`
   "Checked against its SHA-256 list" from the same GitHub release only proves the download wasn't corrupted, not that it came from the maintainer. Anyone who can publish to the release (stolen token, compromised CI) controls both the file and the list. With Automatic (r05, w08) this dialog never appears, so even this statement isn't shown.
   *Suggestion:* Sign releases (minisign/cosign/GPG) and verify the signature against a key built into the installed version, then say so: "signature checked against the tailbond release key". Put the verification line on the Updates card too, not only in the confirm dialog.

5. **medium**: `w03-vps-update-available-desktop.png`, `w13-vps-updates-off-desktop.png`
   The VPS replaces its own binary, but nothing says where it lives or how to pin or switch the update policy from configuration. On an Ansible-managed host, a self-update silently drifts from the playbook, and the only control shown is a button in the web UI.
   *Suggestion:* Show the install path and the config key behind Tell me / Automatic / Off (e.g. `update.policy` in the config file, or a CLI `tailbond update --policy off`), so it can be managed from config management.

6. **medium**: `w08-vps-rolled-back-desktop.png`, `r04-router-rolled-back-desktop.png`
   "0.2.0 didn't start here (the service wasn't up within 60 seconds)" gives no reason and no way to find one. There is a Logs tab right there and it isn't linked. There's no `journalctl` hint, and nothing says whether the failed version's files or logs were kept for a bug report. "Try 0.2.0 again" invites the same failure blind.
   *Suggestion:* Add "See what went wrong" linking to the Logs tab, filtered to the failed start, plus `journalctl -u tailbond --since "<time>"` on the VPS. Add "Report this" with the version and the last log lines.

7. **low**: `r04-router-rolled-back-desktop.png`
   The policy is Tell me, yet the note says "Automatic won't install 0.2.0 again". That doesn't apply here and makes you wonder whether the policy changed.
   *Suggestion:* Show that sentence only when the policy is Automatic. Under Tell me, say "0.2.0 won't be offered again; Try 0.2.0 again if you want to".

8. **low**: `r05-router-automatic-desktop.png`
   "Between 03:00 and 05:00 (this router's time)" on a travel router: which time zone is that, the hotel's or the one set at home? The drop could land in the middle of an evening call.
   *Suggestion:* Show the zone and the local equivalent ("03:00–05:00 Europe/Berlin, now 22:00 there").

9. **low**: `11-updates-available.png`, `10-updates-up-to-date.png`
   The router and VPS say how downloads are checked; the phone says nothing. It doesn't mention that Android refuses an update signed with a different key (19 implies it), or that Obtainium should be pinned to the release signing certificate.
   *Suggestion:* One line: "Android installs it only if it's signed with the same key as this app (fingerprint …)". In the Obtainium docs, say to verify the certificate fingerprint.

10. **low**: `w01-homebox-dnf-update-available-desktop.png`
    The tab badge reads "3" (w01) or "2" (w02) with no indication of what it counts (the server update plus the out-of-date devices?).
    *Suggestion:* Add a tooltip or aria label ("1 update, 2 devices out of date"), or use a dot.

---

## Task 2: Move the home server to a new machine using a backup

**Done?** Partly. Making the backup and restoring it is clear (w05, w06, w07, s02, w12), and the preview warns about split brain, the address change and the route. But the order of steps can lock out a travel router, the per-device guidance is vague, and network prerequisites outside tailbond are missing. The only move shown is VPS to VPS; nothing covers moving the Fedora homebox.
Screens: w02, w05, w06 (desktop + phone), w07, w10, w11, w12, s01, s02, r01, r10, r11, r13, 03, 30, 31b, 34, 37.

### Findings

1. **high**: `w06-restore-preview-new-vps-desktop.png`, `w07-restore-done-desktop.png`
   The steps contradict each other. Step 1: turn the old server off *before* restoring. Step 2: a kill-switch router needs "its new link before you switch over", but the link comes from the new machine's Devices tab, which only has devices *after* the restore. Followed in order, slate7 (kill switch on, at a hotel, no Tailscale of its own per r01) loses internet with no new link yet, and the person with it is stuck.
   *Suggestion:* Give a safe order and say it plainly: 1) restore on the new machine with tailbond still running on the old one (devices stay on the old address); 2) send new links to link-only devices; 3) then `systemctl disable --now` on the old one. Or offer a "restore without starting" option so the links can be made before cutover. Say how a locked-out kill-switch router recovers (temporarily turn off the kill switch in its local panel).

2. **medium**: `w06-restore-preview-new-vps-desktop.png`
   "A device without Tailscale needs its link again… check slate7, pixel-8, beryl" lists every device without saying which one uses which path. pixel-8 is on the tailnet (03 shows "via home server · tail1234.ts.net"; s01 lists it as a Tailscale machine), so it should follow by itself. As written, I'd re-pair a phone that doesn't need it and maybe miss the one that does.
   *Suggestion:* Show a per-device table: "slate7: link only, kill switch on → send new link first; pixel-8: Tailscale → follows after the rename; beryl: offline, unknown → send link when it's back".

3. **medium**: `w06-restore-preview-new-vps-desktop.png`, `w12-whats-inside-desktop.png`, `s01-flint-devices-out-of-date-desktop.png`
   The backup lists "Public address home.example.net" and "Tunnel port 443" (w12), but the restore keeps the new machine's address. Nothing mentions what lives outside tailbond and doesn't move: the DNS record, the port forward on the home router ("must forward UDP 443", s01) pointing at the old LAN IP, and the firewalld rule ("firewall settings" not included). For a homebox move, re-pointing DNS and the port forward is the whole job, and the preview is silent on it.
   *Suggestion:* For a home server backup, add a "Outside tailbond" checklist: update the DNS name or keep the old one, move the port forward to this machine's LAN IP, open UDP <port> (`firewall-cmd --add-port=443/udp --permanent`). Offer "use the backup's address (home.example.net)" as a choice when it's a DNS name.

4. **medium**: `w11-scheduled-backups-change-desktop.png`, `w12-whats-inside-desktop.png`, `w05-backup-download-dialog-desktop.png`
   The backup holds server.key, global.key and the devices' backups, and the automatic-backup passphrase is "kept on this machine, in its settings", but the screen gives no file path or permissions. The "Copy them somewhere else" off-site copies are only as safe as that settings file. Neither screen names the cipher or KDF.
   *Suggestion:* Name the file and its mode ("/etc/tailbond/config, root-only 0600"). Add one line with the algorithm ("age / scrypt" or whatever is actually used). Recommend a separate passphrase for copies taken off the box, or let the scheduled job encrypt to a public key so the box can't decrypt its own backups.

5. **medium**: `w06-restore-preview-new-vps-desktop.png`
   The one command that prevents split brain, `sudo systemctl disable --now tailbond`, sits inside running prose in a proportional font. On desktop it wraps as "disable --" / "now" and there's no Copy button. w01 has a code box with Copy, so it's inconsistent.
   *Suggestion:* Render it as a code block with Copy, as on w01, and name the machine it runs on ("on vps, 203.0.113.9").

6. **medium**: `30-backup-page.png`, `31b-backup-send-to-homebox-dialog.png`, `w02-updated-devices-out-of-date-backups-desktop.png`
   homebox shows pixel-8 backups every day at 08:12, which looks automatic. The phone only offers a manual "Send a backup to homebox" that asks for a new passphrase each time, with no schedule or on/off state. I can't tell whether the phone backs up by itself, with which passphrase, or how to stop it.
   *Suggestion:* Add an "Automatic backups" row on the phone (on/off, schedule, last sent, "passphrase stored on this phone"), matching the router's card.

7. **low**: `w07-restore-done-desktop.png`, `r11-router-restore-done-desktop.png`
   After the restore, the list still says "If you haven't already: 1. Before restoring, turn tailbond off…". "Locked with the same passphrase" is ambiguous on a fresh machine that had no passphrase (w10): the same as which?
   *Suggestion:* Reword to "Now, if you haven't: turn tailbond off on vps". Say "locked with the passphrase of the backup you just restored".

8. **low**: `w03-vps-update-available-desktop.png`, `w06-restore-preview-new-vps-desktop.png`
   vps's dashboard shows tunnel IP 10.77.0.1 (10.77.0.0/24), yet restoring vps's backup says the tunnel addresses "become 10.83.142.0/24 (the backup's)". Cross-checking, I can't tell which subnet to approve in the Tailscale console.
   *Suggestion:* Make the old dashboard and the backup agree, or say why they differ (e.g. "changed on Nov 2 after this backup").

9. **low**: `w06-restore-preview-new-vps-desktop.png`
   The "What it replaces" list is cut off below the warnings (desktop stops at Approvers; phone doesn't reach the list), with no hint that it scrolls. A change of update policy (Tell me → Automatic) matters on an Ansible host and is easy to miss.
   *Suggestion:* Show the row count ("What it replaces (6)") and add a fade or scroll cue. Flag the update-policy change in the warnings when it switches to Automatic.

10. **low**: `r13-router-whats-inside-desktop.png`
    "Router settings (UCI): included" next to "Not included: … firewall settings". The firewall is UCI too, so which UCI configs are in the file?
    *Suggestion:* List the UCI sections ("tailbond, network.tailbond; not firewall").

---

## Task 3: The router got a GL firmware update. Is tailbond still there?

**Done?** Yes. r06 says, in the top card and the Updates card, that tailbond is back with its settings, when, and "Nothing to do". The questions that remain are about provenance and about the case where it wouldn't come back.
Screens: r01, r06, s01.

### Findings

1. **medium**: `r06-router-reinstalled-after-firmware-upgrade-desktop.png`
   "Reinstalled itself from the copy kept on this router (or, if that was gone, from the GitHub release)" doesn't say which one actually happened. It also doesn't say whether a GitHub reinstall was checked (SHA-256 / signature) or which version it fetched (the same 0.2.0, or the newest?). A package that installs itself after a firmware flash is exactly what I want to audit.
   *Suggestion:* State the fact, not the alternatives: "Reinstalled 0.2.0 from the copy on this router (/etc/tailbond/…), checksum OK", or "Downloaded 0.2.0 from GitHub, checked against the release key".

2. **medium**: `r01-router-up-to-date-desktop.png`, `r06-router-reinstalled-after-firmware-upgrade-desktop.png`
   The one case that loses tailbond (upgrading without Keep Settings) is explained only on r06, after an upgrade that went fine, when it no longer matters. Before an upgrade, r01 has only a grey parenthetical, "(kept through firmware upgrades with Keep Settings)", under Automatic backups.
   *Suggestion:* Put a short line on the router's Updates or Backup card before the fact: "GL firmware upgrade: leave Keep Settings on. Without it, tailbond and its local backups are erased; homebox has the last 3 (last sent 5 h ago)". Show the r06 explanation only on the not-recovered path.

---

## Overall count

- **high: 2** (Task 1 #1, Task 2 #1)
- **medium: 12** (Task 1: 5, Task 2: 5, Task 3: 2)
- **low: 8** (Task 1: 4, Task 2: 4)
