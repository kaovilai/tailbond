# Screenshot manifest: Android, updates and backup/restore

Phone screenshots, 1080x2400 (or the whole scrolling page when it's longer),
one row per PNG. `-dark` means the same screen in dark theme. The data is a test fixture: one
home server, **homebox**. GitHub's latest release is 0.2.0
(`kaovilai/tailbond`); the phone runs 0.2.0 where it's up to date and 0.1.0
where an update is available. The renders use a fixed time of 14:30 today,
with the last check at 14:02.

Renders go to `out/ux/updates-backup/android/` (not committed). They come from
`android/app/src/test/kotlin/dev/tailbond/app/ux/UpdatesBackupScreens.kt`:

```sh
android/scripts/gradle.sh :app:testDebugUnitTest --tests 'dev.tailbond.app.ux.UpdatesBackupScreens' \
    -Ptailbond.uxOut=$PWD/out/ux/updates-backup/android
```

| File | What it shows |
|---|---|
| 01-settings-updates-and-backup-rows | Settings page (long): the "Updates" row ("Up to date · checked today at 14:02") and the "Backup" row at the bottom |
| 02-settings-update-available-row | The same rows when 0.2.0 is out: "Update available: v0.2.0" |
| 10-updates-up-to-date | Settings → Updates. Up to date, checked today. Check now; policy "Tell me" / "Off"; what the check sends; "Want automatic updates? Use Obtainium" with how it works and "How to set up Obtainium" (the docs) |
| 10-updates-up-to-date-dark | The same in dark theme |
| 11-updates-available | "Update available: v0.2.0" with "Download update" (opens the APK in the browser) and "What's new" (the release page), and what happens after the download |
| 12-updates-check-failed | "Couldn't check for updates": no internet or GitHub unreachable; when it tried and when the last good check was |
| 13-updates-off | Policy Off, never checked: "Updates are off", Check now still works |
| 19-updates-debug-build | A debug build with 0.2.0 available: the debug key differs from the release key, so a release APK won't install over it; how to switch (back up, uninstall, install the release APK, restore) |
| 30-backup-page | Settings → Back up and restore |
| 30-backup-page-dark | The same in dark theme |
| 31-backup-save-dialog | After tapping "Save a backup file": passphrase typed twice, at least 10 characters |
| 31b-backup-send-to-homebox-dialog | After tapping "Send a backup to homebox" |
| 32-backup-saved | After saving through Android's file picker |
| 33-restore-type-passphrase | After picking a `.tbbackup` file: who made it, when, and with which version; asks for its passphrase |
| 33b-restore-wrong-passphrase | Wrong passphrase, or a damaged file |
| 34-restore-preview | What changes before anything changes (servers, mode, apps, websites, limit), a warning for an app that isn't installed, and a note that undo works once |
| 35-restore-done-with-undo | Restored, with "Undo restore" |
| 36-backup-page-undo-available | The backup page shortly after a restore: "Undo restore" is still there |
| 37-restore-a-server-backup | Picking a home server's backup: refused, with what it is |

The app doesn't install anything itself: "Download update" hands the APK to
the browser, and Android's own installer takes it from there (it installs
completely or not at all, so a failed install leaves the old version).
