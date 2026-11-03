# Screenshot manifest: updates and backups (tailbond's own dashboard)

This is the agent's web dashboard on a Fedora home server or a VPS
(`agent/web`). It has a new tab, **Updates & backup**, for both roles. The tab
shows a badge when an update is available, when devices are out of date, or
when an update failed. The tab has two cards, **Updates** and
**Backup & restore**. Every screen shows that tab.

Two machines:

- **homebox**, a Fedora home server with the RPM, so its updates come from
  dnf (`update.method` `dnf`). Its devices are slate7 (travel router),
  pixel-8 (Android phone) and beryl (travel router, off right now).
- **vps**, a rented server with the Linux bundle (`update.method` `static`:
  it updates itself). In w10, w06 and w07 it's a new VPS (198.51.100.20)
  taking over from the old one (203.0.113.9).

Test data: `scripts/ux/glinet/ub-fixtures.mjs` (`webStates` /
`webActions`); the real `agent/web` files, with the API answered from the
test data. Render with `scripts/ux/updates-backup-screens.sh` →
`out/ux/updates-backup/web/<name>-desktop.png` (1280x800 window; pages at
full length, dialogs at window size), and `<name>-phone.png` (390x844) for
the four marked "phone too". All screens use the same clock: today at
22:00.

| Name | Situation |
|---|---|
| w01-homebox-dnf-update-available | homebox on 0.2.0, 0.2.1 out: "Update available: 0.2.1" + What's new, no Update now button; "Updates for this server come from dnf: run `sudo dnf upgrade tailbond-server`, or turn on dnf-automatic" with a Copy button. "2 devices out of date" (slate7, pixel-8 on 0.1.0) with how each updates. Policy only Tell me / Off, plus "Automatic updates come from dnf-automatic". Phone too |
| w02-updated-devices-out-of-date-backups | homebox up to date on 0.2.0 (last update 0.2.0 worked): "2 devices out of date", automatic backups on, saved backups (homebox's and the routers'). Phone too |
| w03-vps-update-available | VPS (static bundle) on 0.1.0: "Update available: 0.2.0" + What's new, Update now, Check now; policy Tell me / Automatic / Off; "All 3 devices are on 0.1.0 or newer" |
| w04-vps-update-confirm | After "Update now": "Update to 0.2.0 now?" (tailbond restarts; devices drop for a few seconds; goes back by itself if the new version doesn't start) |
| w05-backup-download-dialog | "Download a backup" with a passphrase typed twice: strength hint, "Write it down". Phone too |
| w06-restore-preview-new-vps | New VPS, old VPS's backup checked: what it replaces (devices, public address, port, tunnel addresses, approvers), warnings (another machine with another public address: devices find the new one by themselves through the Tailscale name; devices paired with a link need a new link). Phone too |
| w07-restore-done | After Restore: "Restored. A copy of the previous settings was kept (before-restore-….tbbackup)" |
| w08-vps-rolled-back | VPS: chip "Went back to 0.1.0", "0.2.0 didn't start here, so tailbond went back to 0.1.0 by itself" (why: the service wasn't up on 0.2.0 within 60 s); red badge on the tab |
| w09-vps-check-failed | VPS up to date as far as it knows; the last check failed: "Checked 6 min ago · Couldn't reach GitHub: api.github.com: i/o timeout" |
| w10-vps-new-machine | The new VPS before restoring: up to date on 0.2.0, automatic backups off, no saved backups yet |
| w11-scheduled-backups-change | homebox, after "Change" under Automatic backups: passphrase (kept if empty), every N hours, keep N |
| w12-whats-inside | "See what's inside" (`GET /api/backup/summary`) |
| w13-vps-updates-off | VPS with policy Off: chip "Updates off", "tailbond doesn't check for new versions", no Check now |

Routers have the same cards in the GL panel ([glinet-screens.md](glinet-screens.md)).
The client role of this dashboard shows the same cards ("This machine runs
tailbond …", no device list) and offers "Also send a copy to homebox" for
automatic backups.
