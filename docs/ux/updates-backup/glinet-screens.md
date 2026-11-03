# Screenshot manifest: updates and backups (GL.iNet panel)

GL admin panel, VPN → tailbond. The **Updates** and **Backup & restore**
cards sit at the bottom of the page, under the other cards (they show only
when the agent supports updates). Two routers:

- **Slate 7** (GL-BE3600), a travel router at a hotel, paired with
  **homebox**. Its updates come from GitHub releases and install with opkg
  (`update.method` `opkg`).
- **Flint 3** (GL-BE9300) as the home server **homebox** (also `opkg`), with
  three devices: slate7 (travel router), pixel-8 (Android phone), beryl
  (travel router, off right now).

Test data: `scripts/ux/glinet/ub-fixtures.mjs` (`glStates` / `glActions`).
Render with `scripts/ux/updates-backup-screens.sh` →
`out/ux/updates-backup/glinet/<name>-desktop.png` (1280x800 window; pages at
full length, dialogs at window size), and `<name>-phone.png` (390x844) for
the four marked "phone too". All screens use the same clock: today at 22:00.
homebox updated to 0.2.0 at 03:06 this morning.

| Name | Situation |
|---|---|
| r01-router-up-to-date | Slate 7 on 0.2.0, "Up to date", "Checked 2 h ago · last update: 0.2.0, Nov 3, 03:06, worked". Check now; policy Tell me (recommended). Automatic backups on, with a copy sent to homebox |
| r02-router-update-available | Slate 7 on 0.1.0: "Update available: 0.2.0" with "What's new", Update now and Check now, policy Tell me. Phone too |
| r03-router-installing | Installing 0.2.0: "tailbond restarts; this page comes back by itself in about a minute" |
| r04-router-rolled-back | Chip "Went back to 0.1.0": "0.2.0 didn't start here, so tailbond went back to 0.1.0 by itself" (why: the tunnel didn't come back within 60 s). 0.2.0 is still offered (Update now tries again) |
| r05-router-automatic | Up to date on 0.2.0, policy Automatic selected, with its help line (installs by itself, devices drop for a few seconds, goes back by itself) |
| r06-router-reinstalled-after-firmware-upgrade | After a GL firmware upgrade (`detail` `reinstalled_after_firmware_upgrade`): "The GL firmware update removed tailbond; it reinstalled itself with your settings" |
| r07-router-update-confirm | After "Update now": "Update to 0.2.0 now?" (tailbond restarts; devices drop for a few seconds; goes back by itself if the new version doesn't start) |
| r08-router-backups-off | Up to date. Automatic backups off ("Set a passphrase…", Turn on) |
| r09-router-backup-download-dialog | After "Download a backup", with a passphrase typed twice: strength hint, "Write it down". Phone too |
| r10-router-restore-preview | A factory-reset Slate 7 getting slate7's backup: file + passphrase checked, what it replaces, two warnings (old router still running, Tailscale isn't in the backup). Phone too |
| r11-router-restore-done | After clicking Restore: "Restored. A copy of the previous settings was kept" |
| r13-router-whats-inside | "See what's inside": what a backup holds, without secrets |
| s01-flint-devices-out-of-date | Flint 3 home server, up to date on 0.2.0: "2 devices out of date" (slate7 travel router on 0.1.0, pixel-8 Android phone on 0.1.0) with how each updates (routers: Update now in their panel, or Automatic; phones: the app shows it, Obtainium installs it). Saved backups (its own and the travel routers'). Phone too |

What the panel sends (docs/architecture.md "Updates", "Backup and restore"):
JSON calls go through the `tailbond` /rpc object, as before. The backup file
(`POST /api/backup`), the restore calls (`POST /api/restore/preview`,
`POST /api/restore`, bodies up to 2 MiB), the saved-backup downloads
(`GET /api/backups/<name>`) go through the same-origin `/tailbond-api/` proxy, which checks the same GL admin
session (`packaging/openwrt/gl-ui/etc/nginx/gl-conf.d/tailbond.conf`).
