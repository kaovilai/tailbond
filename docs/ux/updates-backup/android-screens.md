# Screenshot manifest: Android, updates and backup/restore

Phone screenshots, 1080x2400 (or the whole scrolling page when it's longer).
`-dark` means the same screen in dark theme. The data is a test fixture: one
home server, **homebox**. The phone runs tailbond 0.1.0, and 0.2.0 is out. The
renders use a fixed time of 14:30 today, with the last check at 14:02.

Renders go to `out/ux/updates-backup/android/` (not committed). They come from
`android/app/src/test/kotlin/dev/tailbond/app/ux/UpdatesBackupScreens.kt`:

```sh
android/scripts/gradle.sh :app:testDebugUnitTest --tests 'dev.tailbond.app.ux.UpdatesBackupScreens' \
    -Ptailbond.uxOut=$PWD/out/ux/updates-backup/android
```

| File | What it shows |
|---|---|
| 01-settings-updates-and-backup-rows | Settings page (long): the new "Updates" and "Backup" rows at the bottom |
| 10-updates-up-to-date (+ -dark) | Settings → Updates. Up to date, checked from homebox. Policy, channel, "How updates work" (mirror first, signature, deferral during calls, restart, debug vs release, Obtainium) |
| 11-updates-available-tell-me-first | Policy "Tell me first": 0.2.0 is available, with "Install now" and "What's new in 0.2.0" |
| 12-updates-waiting-during-a-call | Automatic, but the tunnel carries more than 300 kbit/s: the update waits (Install now still works) |
| 12b-updates-waiting-for-the-server | 0.2.0 needs a newer server protocol: the phone waits for homebox to update |
| 13-updates-downloading | Downloading, 62 % |
| 14-updates-installing | Handed to Android. The tunnel drops for a few seconds |
| 15-updates-one-tap-the-first-time | The first update after a browser install: Android wants one tap ("Finish installing") |
| 16-updates-allow-installing | "Install unknown apps" isn't allowed for tailbond yet: "Allow" opens Android's page |
| 17-updates-couldnt-install | Couldn't install: the download failed from every source; still on 0.1.0 |
| 17b-updates-couldnt-install-other-key | Couldn't install: the APK has another signing key (debug vs release) |
| 18-updates-updated-silently | After a silent update overnight: "Updated to 0.2.0" |
| 19-updates-debug-build-no-key | A build without an update key (debug): updates are off, and the page says why |
| 19b-updates-android-11-tap-every-time | Android 11: every update needs a tap |
| 30-backup-page (+ -dark) | Settings → Back up and restore |
| 31-backup-save-dialog | After tapping "Save a backup file": passphrase typed twice, at least 10 characters |
| 31b-backup-send-to-homebox-dialog | After tapping "Send a backup to homebox" |
| 32-backup-saved | After saving through Android's file picker |
| 33-restore-type-passphrase | After picking a `.tbbackup` file: who made it, when, and with which version; asks for its passphrase |
| 33b-restore-wrong-passphrase | Wrong passphrase, or a damaged file |
| 34-restore-preview | What changes before anything changes (servers, mode, apps, websites, limit), a warning for an app that isn't installed, and a note that undo works once |
| 35-restore-done-with-undo | Restored, with "Undo restore" |
| 36-backup-page-undo-available | The backup page shortly after a restore: "Undo restore" is still there |
| 37-restore-a-server-backup | Picking a home server's backup: refused, with what it is |

Android has no "failed and rolled back" state. Android installs an APK
completely or not at all, so a failed update leaves the old version running
(17, 17b).
