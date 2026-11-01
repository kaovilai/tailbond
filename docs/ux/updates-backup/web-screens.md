# Screenshot manifest: updates and backups (tailbond's own dashboard)

This is the agent's web dashboard on a Fedora home server or a VPS
(`agent/web`). It has a new tab, **Updates & backup**, for both roles. The tab
shows a badge when an update is available, when devices are out of date, or
when an update failed. The tab has two cards, **Updates** and
**Backup & restore**. Every screen shows that tab.

Two machines:

- **homebox**, a Fedora home server. Its devices are slate7 (travel router),
  pixel-8 (Android phone) and beryl (travel router, off right now).
- **vps**, a rented server. In w10, w06 and w07 it's a new VPS
  (198.51.100.20) taking over from the old one (203.0.113.9).

Test data: `scripts/ux/glinet/ub-fixtures.mjs` (`webStates` /
`webActions`); the real `agent/web` files, with the API answered from the
test data. Render with `scripts/ux/updates-backup-screens.sh` →
`out/ux/updates-backup/web/<name>-desktop.png` (1280x800 window; pages at
full length, dialogs at window size), and `<name>-phone.png` (390x844) for
the three marked "phone too". All screens use the same clock: today at
22:00.

| Name | Situation |
|---|---|
| w01-update-available-waiting-for-window | homebox on 0.1.0 with 0.2.0 available, policy Automatic: "Waiting for 03:00. It installs between 03:00 and 05:00". The mirror already serves 0.2.0. Install window, Check now, Install now, Upload an update. Below: automatic backups on, saved backups |
| w02-updated-devices-out-of-date-backups | homebox updated to 0.2.0: "2 devices out of date" (slate7, travel router, on 0.1.0; pixel-8, Android phone, on 0.1.0), "Update mirror: serving 0.2.0", saved backups (homebox's and the routers'). Phone too |
| w03-install-confirm | After "Install 0.2.0 now": what restarts (2 connected devices drop for a moment), goes back by itself if it doesn't work |
| w04-upload-an-update | "Upload an update" on the VPS: five files picked (stable.json, .sig, .rpm, .ipk, .apk), uploaded, all added |
| w05-backup-download-dialog | "Download a backup" with a passphrase typed twice: strength hint, "Write it down". Phone too |
| w06-restore-preview-new-vps | New VPS, old VPS's backup checked: what it replaces (devices, public address, port, tunnel addresses, approvers), warnings (another machine with another public address: devices find the new one by themselves through the Tailscale name; devices paired with a link need a new link). Phone too |
| w07-restore-done | After Restore: "Restored. A copy of the previous settings was kept (before-restore-….tbbackup)" |
| w08-vps-rolled-back | VPS: 0.2.0 didn't work (the API didn't answer within 90 s after the restart), went back to 0.1.0 by itself; red badge on the tab |
| w09-vps-pinned-by-setup | VPS pinned to 0.1.0 by Ansible (`update.pin`) |
| w10-vps-new-machine | The new VPS before restoring: automatic backups off, no saved backups yet |
| w11-scheduled-backups-change | homebox, after "Change" under Automatic backups: passphrase (kept if empty), every N hours, keep N |
| w12-whats-inside | "See what's inside" (`GET /api/backup/summary`) |

Routers have the same cards in the GL panel ([glinet-screens.md](glinet-screens.md)).
The client role of this dashboard shows the same cards. There it says where
updates come from and offers "Also send a copy to homebox" for automatic
backups.
