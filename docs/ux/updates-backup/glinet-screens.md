# Screenshot manifest: updates and backups (GL.iNet panel)

GL admin panel, VPN → tailbond. The **Updates** and **Backup & restore**
cards sit at the bottom of the page, under the other cards (they show only
when the agent supports updates). News that matters (update available,
installing, went back, reinstalled after a firmware update) is also a line in
the top status card, with **Show** to jump to the Updates card. Two routers:

- **Slate 7** (GL-BE3600), a travel router at a hotel, paired with
  **homebox**. Its updates come from GitHub releases and install with opkg
  (`update.method` `opkg`).
- **Flint 3** (GL-BE9300) as the home server **homebox** (also `opkg`), with
  three devices: slate7 (travel router), pixel-8 (Android phone), beryl
  (travel router, off right now; its version isn't known yet).

Test data: `scripts/ux/glinet/ub-fixtures.mjs` (`glStates` / `glActions`).
Render with `scripts/ux/updates-backup-screens.sh` →
`out/ux/updates-backup/glinet/<name>-desktop.png` (1280x800 window; pages at
full length, dialogs at window size), and `<name>-phone.png` (390x844) for
the four marked "phone too". All screens use the same clock: today at 22:00.
On Tell me, slate7 and homebox were updated to 0.2.0 by hand at 14:20; on
Automatic it installs at 03:06. Scheduled backups run at
03:00; slate7 sends its backup to homebox at 17:00, pixel-8 at 08:12 (homebox
keeps the last 3 of each).

| Name | Situation |
|---|---|
| r01-router-up-to-date | Slate 7 on 0.2.0, "Up to date", "Checked 2 h ago · last update: 0.2.0, Nov 3, 14:20, worked". Check now; policy Tell me. Automatic backups on, a copy sent to homebox, "Saved in /etc/tailbond/state/backups/ on this router (kept through firmware upgrades with Keep Settings)" |
| r02-router-update-available | Slate 7 on 0.1.0. Top card: "Update available: tailbond 0.2.0 (this router runs 0.1.0)" · Show. Updates card: "Update available: 0.2.0" with What's new, Update now and Check now, policy Tell me. Phone too |
| r03-router-installing | Installing 0.2.0 (top card and Updates card): "tailbond restarts; this page comes back by itself in about a minute" |
| r04-router-rolled-back | Chip "Went back to 0.1.0" (amber, not red). Top card and a calm blue note first: "Everything works as before on 0.1.0. 0.2.0 didn't start here, so tailbond went back by itself (the tunnel didn't come back within 60 seconds). Automatic won't install 0.2.0 again; the next version installs as usual." Retry is a secondary "Try 0.2.0 again" next to Check now |
| r05-router-automatic | Up to date on 0.2.0, policy Automatic: "Installs new versions by itself between 03:00 and 05:00 (this router's time). Devices drop for a few seconds; if a new version doesn't start, it goes back by itself." |
| r06-router-reinstalled-after-firmware-upgrade | After a GL firmware upgrade (`detail` `reinstalled_after_firmware_upgrade`). Top card: "tailbond is back after the firmware update, with your settings". Updates card: same title, "reinstalled itself from the copy kept on this router (or, if that was gone, from the GitHub release) … If you upgrade the firmware without Keep Settings: install tailbond again, then restore the router's backup (homebox keeps the last 3: Backup & restore there), or pair again." |
| r07-router-update-confirm | After "Update now": "Update to 0.2.0 now?" (tailbond restarts; devices drop for a few seconds; goes back by itself if the new version doesn't start), then "Downloaded from the GitHub release and checked against its SHA-256 list before anything is installed." |
| r08-router-backups-off | Up to date. Automatic backups off ("Set a passphrase…", Turn on) |
| r09-router-backup-download-dialog | After "Download a backup", with a passphrase typed twice: strength hint, "Write it down". Phone too |
| r10-router-restore-preview | A factory-reset Slate 7 getting its own backup (tailbond-slate7-20261102-1700.tbbackup, made Nov 2, 17:00): what it replaces (Mode (default) → Speed, servers, rules, connection settings). Same router, so no warnings. Phone too |
| r11-router-restore-done | After Restore, no warnings: "tailbond connects to its home server again by itself within a minute"; automatic backups follow the backup (on if they were on, same schedule and passphrase); the previous settings kept as before-restore-….tbbackup |
| r13-router-whats-inside | "See what's inside" on Slate 7: what a backup holds, without secrets. "Not included: the Tailscale login, the installed program, firewall settings. On a new router: install tailbond, sign in to Tailscale, then restore." |
| s01-flint-devices-out-of-date | Flint 3 home server, up to date on 0.2.0 (updated by hand at 14:20). "Your devices: 2 of 3 out of date": slate7 (travel router, "0.1.0 · behind the latest (0.2.0)", Out of date), pixel-8 (Android phone, same), beryl (travel router, version not known yet), then how each updates ("Routers: Update now in its tailbond panel, or Automatic. Phones: the app shows it; with Obtainium it installs by itself.") and why a version can be unknown. Backup card: "Move to a new router: the restore preview says what each device needs." Saved backups: its own (03:00) and the last 3 of slate7 and pixel-8, beryl's one. Phone too |
| s02-flint-whats-inside | "See what's inside" on the Flint 3 home server: "Your devices' backups: slate7 (3), pixel-8 (3), beryl (1)", "Your devices' own backups are included (still locked with their passphrases), so they move with this router", and what's not included |

What the panel sends (docs/architecture.md "Updates", "Backup and restore"):
JSON calls go through the `tailbond` /rpc object, as before. The backup file
(`POST /api/backup`), the restore calls (`POST /api/restore/preview`,
`POST /api/restore`, bodies up to 2 MiB), the saved-backup downloads
(`GET /api/backups/<name>`) go through the same-origin `/tailbond-api/` proxy, which checks the same GL admin
session (`packaging/openwrt/gl-ui/etc/nginx/gl-conf.d/tailbond.conf`).
