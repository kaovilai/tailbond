# Screenshot manifest: updates and backups (tailbond's own dashboard)

This is the agent's web dashboard on a Fedora home server or a VPS
(`agent/web`). It has a new tab, **Updates & backup**, for both roles. The tab
shows a badge when an update is available or devices are out of date (red
"!" only when an update failed). News that matters (update available,
installing, went back, failed) is also a banner under the main card, on
every tab, with **Show**. The tab has two cards, **Updates** and
**Backup & restore**. Every screen shows that tab. Full-page captures show
the top bar at the top only (not sticky).

Two machines:

- **homebox**, a Fedora home server with the RPM, so its updates come from
  dnf (`update.method` `dnf`). Its devices are slate7 (travel router),
  pixel-8 (Android phone) and beryl (travel router, off right now; version
  not known yet). Scheduled backups at 03:00; slate7 sends its backup at
  17:00; pixel-8's are sent by hand from the app (Nov 3, Oct 31, Oct 26). homebox keeps the last 3 of each.
- **vps**, a rented server with the Linux bundle (`update.method` `static`:
  it updates itself), reached at 203.0.113.9, tunnel port 8443.
- **vps2**, a new VPS (198.51.100.20, port 443) just installed, taking over
  from vps (w10, w06, w07).

Test data: `scripts/ux/glinet/ub-fixtures.mjs` (`webStates` /
`webActions`); the real `agent/web` files, with the API answered from the
test data. Render with `scripts/ux/updates-backup-screens.sh` →
`out/ux/updates-backup/web/<name>-desktop.png` (1280x800 window; pages at
full length, dialogs at window size; long dialogs scroll with their buttons kept in view), and `<name>-phone.png` (390x844) for
the four marked "phone too". All screens use the same clock: today at
22:00.

| Name | Situation |
|---|---|
| w01-homebox-dnf-update-available | homebox on 0.2.0, 0.2.1 out. Banner under the main card: "Update available: tailbond 0.2.1 (this server runs 0.2.0)" · Show. Card: "Update available: 0.2.1" + What's new, no Update now; "Updates for this server come from dnf. In a terminal on this server, run:" `sudo dnf upgrade tailbond-server` + Copy. Devices: 3 listed, slate7 and pixel-8 "0.1.0 · behind the latest (0.2.1)", beryl version not known yet. "Check for new versions: On / Off" (no Tell me / Automatic for dnf); "Automatic updates: dnf-automatic only downloads updates (Fedora's default). To install them too: set apply_updates = yes in /etc/dnf/automatic.conf."; "These are done on the computer itself, by whoever looks after it", when versions reach dnf, how to downgrade. Phone too |
| w02-updated-devices-out-of-date-backups | homebox up to date on 0.2.0 (updated at 14:20), "Automatic updates: dnf-automatic installs updates for every package on this computer, tailbond included." Devices 2 of 3 out of date ("behind the latest (0.2.0)"). Backup card: "Move to a new machine: the restore preview says what each device needs." Automatic backups on, saved in /var/lib/tailbond/backups/; saved backups: homebox's three, slate7's last 3 (daily), pixel-8's last 3 (sent by hand, irregular dates), beryl's one. Phone too |
| w03-vps-update-available | VPS (static bundle) on 0.1.0: banner, "Update available: 0.2.0" + What's new, "You can update now or later; everything keeps working meanwhile.", Update now, Check now; slate7 and pixel-8 "0.1.0 · behind the latest (0.2.0)" (Out of date), beryl not known yet; policy Tell me / Automatic / Off |
| w04-vps-update-confirm | After "Update now": "Update to 0.2.0 now?" (tailbond restarts; devices drop for a few seconds; goes back by itself), then "Downloaded from the GitHub release and checked against its SHA-256 list before anything is installed." |
| w05-backup-download-dialog | "Download a backup" with a passphrase typed twice: strength hint, "Write it down". Phone too |
| w06-restore-preview-new-vps | vps2 checks vps's backup (vps-20261102-030000.tbbackup, made Nov 2, 03:00). First and prominent, "Do these in order", the agent's four steps: 1. restore here while vps still runs (slate7, pixel-8, beryl keep using it); 2. give link-only devices their new link (GL router: paste it in its tailbond panel, Add another server → Paste a link; phone: scan the new QR), kill-switch routers before step 3; 3. then turn tailbond off on vps, `sudo systemctl disable --now tailbond` as a code line with Copy, and move the Tailscale name vps to this machine; 4. approve 10.83.142.0/24. Then what it replaces (Mode, devices, tunnel addresses, approvers, Update policy; no public address or port: the restore keeps this machine's). Restore is the normal primary button. Phone too |
| w07-restore-done | After Restore: "Next, in order:" steps 2–4 (step 1, restore while vps still runs, is done), the stop command with Copy; automatic backups follow the backup; the previous settings kept as before-restore-….tbbackup |
| w08-vps-rolled-back | VPS on Automatic: chip "Went back to 0.1.0" (calm blue-grey, not red; no red tab badge), banner "Everything works as before on 0.1.0: 0.2.0 didn't start here…", a calm note "Automatic won't install 0.2.0 again; the next version installs as usual", secondary "Try 0.2.0 again"; devices on 0.1.0 shown behind the latest (0.2.0) |
| w09-vps-check-failed | VPS whose last check failed: chip "Couldn't check" (amber), "Couldn't check for updates", "Couldn't reach GitHub (api.github.com: i/o timeout). tailbond tries again tomorrow; Check now tries again now." Devices show just their versions, no "✓ up to date", and "Couldn't check for newer versions." |
| w10-vps-new-machine | vps2 just installed, before restoring: running for 12 min, no traffic, "No devices yet", automatic backups off, no saved backups |
| w11-scheduled-backups-change | homebox, after "Change" under Automatic backups: passphrase (kept if empty), "The passphrase is kept on this machine, in its settings, so it can back up by itself. The files go to /var/lib/tailbond/backups/", every N hours, keep N |
| w12-whats-inside | "See what's inside" (`GET /api/backup/summary`): keys and certificates, "Your devices' backups: slate7 (3), pixel-8 (3), beryl (1)", "Your devices' own backups are included (still locked with their passphrases)", "Not included: the Tailscale login, the installed program, firewall settings" |
| w13-vps-updates-off | VPS with policy Off: chip "Updates off", "tailbond doesn't check for new versions", no Check now; devices show just their versions and "Updates are off, so newer versions aren't checked." |

Routers have the same cards in the GL panel ([glinet-screens.md](glinet-screens.md)).
The client role of this dashboard shows the same cards ("This machine runs
tailbond …", no device list) and offers "Also send a copy to homebox" for
automatic backups.
