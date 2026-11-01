# Screenshot manifest: updates and backups (GL.iNet panel)

GL admin panel, VPN → tailbond. The **Updates** and **Backup & restore**
cards sit at the bottom of the page, under the other cards (they show only
when the agent supports updates). Two routers:

- **Slate 7** (GL-BE3600), a travel router at a hotel, paired with
  **homebox**. Updates come from homebox's mirror.
- **Flint 3** (GL-BE9300) as the home server **homebox**, with three
  devices: slate7 (travel router), pixel-8 (Android phone), beryl (travel
  router, off right now).

Test data: `scripts/ux/glinet/ub-fixtures.mjs` (`glStates` / `glActions`).
Render with `scripts/ux/updates-backup-screens.sh` →
`out/ux/updates-backup/glinet/<name>-desktop.png` (1280x800 window; pages at
full length, dialogs at window size), and `<name>-phone.png` (390x844) for
the four marked "phone too". All screens use the same clock: today at 22:00.
Servers install updates between 03:00 and 05:00. homebox updated to 0.2.0
at 03:04 this morning.

| Name | Situation |
|---|---|
| r01-router-up-to-date | Slate 7 on 0.2.0, up to date, updates come from homebox. Automatic backups on, with a copy sent to homebox |
| r02-router-update-available-automatic | 0.2.0 is out, policy Automatic. The router waits because a call is in progress. "Install 0.2.0 now" is offered. Phone too |
| r03-router-installing | Installing 0.2.0 (tailbond restarts, devices offline for about a minute) |
| r04-router-rolled-back | 0.2.0 didn't work (the tunnel didn't come back within 90 s), so the router went back to 0.1.0 by itself |
| r05-router-updates-from-homebox | Policy "Tell me only": 0.2.0 available from homebox, waiting for the user to install it |
| r06-router-reinstalled-after-firmware-upgrade | After a GL firmware upgrade: "tailbond was reinstalled after a firmware update" (`last_result` ok, detail `reinstalled_after_firmware_upgrade`) |
| r07-router-pinned-by-setup | Pinned to 0.1.0 by the setup (`update.pin`): no Install button, no policy choice |
| r08-router-backups-off | Up to date. Automatic backups off ("Set a passphrase…", Turn on) |
| r09-router-backup-download-dialog | After "Download a backup", with a passphrase typed twice: strength hint, "Write it down". Phone too |
| r10-router-restore-preview | A factory-reset Slate 7 getting slate7's backup: file + passphrase checked, what it replaces, two warnings (old router still running, Tailscale isn't in the backup). Phone too |
| r11-router-restore-done | After clicking Restore: "Restored. A copy of the previous settings was kept" |
| r12-router-install-confirm | After "Install 0.2.0 now": the confirmation (what restarts, goes back by itself) |
| r13-router-whats-inside | "See what's inside": what a backup holds, without secrets |
| s01-flint-devices-out-of-date-mirror | Flint 3 home server on 0.2.0: "2 devices out of date" (slate7 travel router on 0.1.0, pixel-8 Android phone on 0.1.0), "Update mirror: serving 0.2.0 to your devices", install window 03:00–05:00, saved backups (its own and the travel routers'). Phone too |
| s02-flint-upload-an-update | "Upload an update" (offline home server): five files picked (stable.json, .sig, two .ipk, .apk), uploaded, all added |
| s03-flint-update-waiting-for-window | Flint 3 on 0.1.0, 0.2.0 available: "Waiting for 03:00. It installs between 03:00 and 05:00" |

What the panel sends (docs/architecture.md "Updates", "Backup and restore"):
JSON calls go through the `tailbond` /rpc object, as before. The backup file
(`POST /api/backup`), the restore calls (`POST /api/restore/preview`,
`POST /api/restore`, bodies up to 2 MiB), the saved-backup downloads
(`GET /api/backups/<name>`) and update uploads
(`POST /api/update/upload?name=<file>`, `application/octet-stream`) go
through the same-origin `/tailbond-api/` proxy, which checks the same GL admin
session (`packaging/openwrt/gl-ui/etc/nginx/gl-conf.d/tailbond.conf`).
