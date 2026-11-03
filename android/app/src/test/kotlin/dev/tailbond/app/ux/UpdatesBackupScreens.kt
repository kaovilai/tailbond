package dev.tailbond.app.ux

import androidx.test.ext.junit.runners.AndroidJUnit4
import dev.tailbond.app.backup.BackupHeader
import dev.tailbond.app.backup.Restore
import dev.tailbond.app.model.AppSettings
import dev.tailbond.app.model.Mode
import dev.tailbond.app.model.ServerProfile
import dev.tailbond.app.model.WebsiteRule
import dev.tailbond.app.pairing.PairingPayload
import dev.tailbond.app.backup.AndroidBackup
import dev.tailbond.app.backup.AndroidConfig
import dev.tailbond.app.ui.BackupUi
import dev.tailbond.app.ui.NoActions
import dev.tailbond.app.ui.PeersUi
import dev.tailbond.app.ui.Routes
import dev.tailbond.app.ui.TailbondApp
import dev.tailbond.app.ui.UiState
import dev.tailbond.app.ui.UpdateUi
import dev.tailbond.app.update.UpdatePolicy
import dev.tailbond.app.update.UpdateStatus
import org.junit.Test
import org.junit.runner.RunWith
import org.robolectric.annotation.Config
import java.time.LocalDate
import java.time.ZoneId

/**
 * Updates and backup/restore:
 *
 *   android/scripts/gradle.sh :app:testDebugUnitTest --tests 'dev.tailbond.app.ux.UpdatesBackupScreens' \
 *       -Ptailbond.uxOut=$PWD/out/ux/updates-backup/android
 */
@RunWith(AndroidJUnit4::class)
@Config(sdk = [35])
class UpdatesBackupScreens : UxRender() {
    private fun at(h: Int, m: Int, daysAgo: Long = 0) =
        LocalDate.now().minusDays(daysAgo).atTime(h, m).atZone(ZoneId.systemDefault()).toInstant().toEpochMilli()

    private val now = at(14, 30)
    private val home = ServerProfile("h", UxFixtures.profile)
    private val vps = ServerProfile(
        "v",
        PairingPayload(
            name = "pixel-8", server = "203.0.113.9:8443", sni = "vps.tail1234.ts.net", user = "pixel-8", key = "not-a-real-key",
            dns = listOf("10.78.0.1"), mode = "speed", serverName = "vps", subnet = "10.78.0.0/24",
        ),
    )
    private val settings = AppSettings(onboarded = true, excludedApps = setOf("com.netflix.mediaclient"))
    private val notes = "https://github.com/kaovilai/tailbond/releases/tag/v0.2.0"
    private val apk = "https://github.com/kaovilai/tailbond/releases/download/v0.2.0/tailbond-0.2.0.apk"
    private val checked = UpdateStatus(
        current = "0.2.0", latest = "0.2.0", notesUrl = notes, apkUrl = apk,
        lastCheckMs = at(14, 2), lastAttemptMs = at(14, 2),
    )

    private fun state(
        status: UpdateStatus = checked,
        s: AppSettings = settings,
        backup: BackupUi = BackupUi(),
        debug: Boolean = false,
    ) = UiState(
        profile = home.payload,
        servers = listOf(home),
        settings = s,
        live = UxFixtures.connectedBoth.copy(tunnelIp = "10.77.0.3"),
        peers = PeersUi(UxFixtures.peers),
        tailscaleInstalled = true,
        sdkInt = 35,
        update = UpdateUi(status, repo = "kaovilai/tailbond", debugBuild = debug, nowMs = now),
        backup = backup,
    )

    private fun app(name: String, st: UiState, route: String, dark: Boolean = false, tall: Int = PHONE_H, dialog: String? = null) =
        shot(name, dark, tall) { TailbondApp(st, NoActions, route, startDialog = dialog) }

    /** This phone has 0.1.0; 0.2.0 is GitHub's latest release. */
    private val available = checked.copy(current = "0.1.0")

    // settings: where the two sections are
    @Test fun settingsRows() = app("01-settings-updates-and-backup-rows", state(), Routes.SETTINGS, tall = 1700)
    @Test fun settingsRowAvailable() = app("02-settings-update-available-row", state(available), Routes.SETTINGS, tall = 1700)
    @Test fun homeBanner() = app("03-home-update-available", state(available), Routes.HOME)

    // updates
    @Test fun upToDate() = app("10-updates-up-to-date", state(), Routes.UPDATES, tall = 1000)
    @Test fun upToDateDark() = app("10-updates-up-to-date-dark", state(), Routes.UPDATES, dark = true, tall = 1000)
    @Test fun availableNow() = app("11-updates-available", state(available), Routes.UPDATES, tall = 1100)
    @Test fun checkFailed() = app(
        "12-updates-check-failed",
        state(checked.copy(lastAttemptMs = at(14, 28), lastCheckMs = at(9, 15, daysAgo = 1), error = "No internet connection, or GitHub can't be reached")),
        Routes.UPDATES,
    )
    @Test fun policyOff() = app(
        "13-updates-off",
        state(UpdateStatus(current = "0.1.0"), s = settings.copy(updatePolicy = UpdatePolicy.OFF)),
        Routes.UPDATES,
    )
    @Test fun debugBuild() = app("19-updates-debug-build", state(available, debug = true), Routes.UPDATES, tall = 1150)

    // backup and restore
    private val backupSettings = AppSettings(
        onboarded = true, mode = Mode.STREAMING, cellularCapMb = 5000,
        excludedApps = setOf("com.netflix.mediaclient", "com.example.banking"),
        websites = listOf(WebsiteRule("mybank.com", true, "Bank"), WebsiteRule("netflix.com", true, "Streaming")),
    )
    private val header = BackupHeader(host = "pixel-8", agentVersion = "0.1.0", created = "2026-10-30T19:45:00Z")
    private val preview = Restore.preview(
        header, AndroidBackup(config = AndroidConfig(listOf(home, vps), backupSettings)), listOf(home), settings,
    ) { it != "com.example.banking" }

    @Test fun backupPage() = app("30-backup-page", state(), Routes.BACKUP)
    @Test fun backupPageDark() = app("30-backup-page-dark", state(), Routes.BACKUP, dark = true)
    @Test fun exportDialog() = app("31-backup-save-dialog", state(), Routes.BACKUP, dialog = "export-backup")
    @Test fun pushDialog() = app("31b-backup-send-to-homebox-dialog", state(), Routes.BACKUP, dialog = "push-backup")
    @Test fun saved() = app(
        "32-backup-saved",
        state(backup = BackupUi(message = "Saved tailbond-pixel-8-2026-11-01.tbbackup. Keep the passphrase somewhere else than the file.")),
        Routes.BACKUP,
    )
    // "inline": dialogs with a text field are drawn in place (see BackupScreen)
    @Test fun restorePassphrase() = app("33-restore-type-passphrase", state(backup = BackupUi(picked = header)), Routes.BACKUP, dialog = "inline")
    @Test fun restoreWrong() = app(
        "33b-restore-wrong-passphrase",
        state(backup = BackupUi(picked = header, error = dev.tailbond.app.backup.BackupFile.WRONG)), Routes.BACKUP, dialog = "inline",
    )
    @Test fun restorePreview() = app("34-restore-preview", state(backup = BackupUi(preview = preview)), Routes.BACKUP, tall = 1000)
    @Test fun restoreDone() = app("35-restore-done-with-undo", state(backup = BackupUi(restoredAt = now, undoAt = now)), Routes.BACKUP)
    @Test fun afterRestore() = app("36-backup-page-undo-available", state(backup = BackupUi(undoAt = at(14, 28))), Routes.BACKUP)
    @Test fun wrongKind() = app(
        "37-restore-a-server-backup",
        state(backup = BackupUi(error = "This backup is from a home server (homebox), not from the tailbond app. Restore it there")),
        Routes.BACKUP,
    )
}
