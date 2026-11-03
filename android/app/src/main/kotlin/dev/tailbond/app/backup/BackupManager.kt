package dev.tailbond.app.backup

import android.content.Context
import android.net.Uri
import android.os.Build
import androidx.core.content.edit
import dev.tailbond.app.BuildConfig
import dev.tailbond.app.data.PeersClient
import dev.tailbond.app.data.Repository
import dev.tailbond.app.data.SecureBox
import dev.tailbond.app.data.TunnelHttp
import dev.tailbond.app.data.phoneRequestName
import dev.tailbond.app.model.ServerList
import dev.tailbond.app.service.ConnState
import dev.tailbond.app.service.LiveState
import kotlinx.coroutines.flow.MutableStateFlow
import kotlinx.coroutines.flow.StateFlow
import kotlinx.coroutines.flow.asStateFlow
import java.io.IOException
import java.time.Instant
import java.time.temporal.ChronoUnit

/**
 * Export, restore (with one "Undo restore") and "Send to home server" for
 * the app's own configuration. Blocking: call off the main thread.
 *
 * The undo copy is the configuration from before the restore, sealed with
 * the app's Android Keystore key ([SecureBox]) like the live pairing, so it
 * needs no passphrase and never leaves the phone.
 */
class BackupManager(private val ctx: Context) {
    private val repo = Repository.get(ctx)
    private val prefs = ctx.getSharedPreferences("tailbond_backup", Context.MODE_PRIVATE)
    private val _undoAt = MutableStateFlow(prefs.getLong(KEY_UNDO_AT, 0).takeIf { it > 0 && prefs.contains(KEY_UNDO) })
    /** When the last restore happened, while "Undo restore" is still possible. */
    val undoAt: StateFlow<Long?> = _undoAt.asStateFlow()

    private fun current() = AndroidBackup(config = AndroidConfig(repo.servers.value, repo.settings.value))

    /** The encrypted file for the current configuration. */
    fun build(passphrase: String): ByteArray = BackupFile.encrypt(
        current().toJson().toByteArray(Charsets.UTF_8),
        passphrase,
        host = phoneRequestName(Build.MODEL ?: "phone"),
        appVersion = BuildConfig.VERSION_NAME,
        created = Instant.now().truncatedTo(ChronoUnit.SECONDS).toString(),
    )

    fun export(uri: Uri, passphrase: String) {
        val bytes = build(passphrase)
        ctx.contentResolver.openOutputStream(uri, "wt")?.use { it.write(bytes) } ?: throw IOException("can't write there")
    }

    fun read(uri: Uri): ByteArray = ctx.contentResolver.openInputStream(uri)?.use { input ->
        val out = java.io.ByteArrayOutputStream()
        val buf = ByteArray(16 * 1024)
        while (true) {
            val n = input.read(buf)
            if (n < 0) break
            out.write(buf, 0, n)
            if (out.size() > BackupFile.MAX_FILE) throw BackupException("This file is too big to be a tailbond backup", BackupException.Kind.NOT_A_BACKUP)
        }
        out.toByteArray()
    } ?: throw IOException("can't read that file")

    /** Opens [file] with [passphrase] and says what restoring it would change. Changes nothing. */
    fun preview(file: ByteArray, passphrase: String): RestorePreview {
        Restore.checkKind(BackupFile.peek(file).first)
        val opened = BackupFile.decrypt(file, passphrase)
        val b = AndroidBackup.fromJson(opened.plaintext.toString(Charsets.UTF_8))
        return Restore.preview(opened.header, b, repo.servers.value, repo.settings.value, ::isInstalled)
    }

    private fun isInstalled(pkg: String): Boolean = try {
        ctx.packageManager.getPackageInfo(pkg, 0); true
    } catch (_: Exception) {
        false
    }

    /** Applies a previewed backup, keeping the configuration from before it for [undo]. */
    fun restore(p: RestorePreview) {
        prefs.edit(commit = true) {
            putString(KEY_UNDO, SecureBox.seal(current().toJson()))
            putLong(KEY_UNDO_AT, System.currentTimeMillis())
        }
        apply(p.backup)
        _undoAt.value = prefs.getLong(KEY_UNDO_AT, 0)
    }

    /** Puts back what was there before the last restore (once). */
    fun undo(): Boolean {
        val sealed = prefs.getString(KEY_UNDO, null) ?: return false
        val json = SecureBox.open(sealed) ?: run { dropUndo(); return false }
        apply(AndroidBackup.fromJson(json))
        dropUndo()
        return true
    }

    fun dropUndo() {
        prefs.edit { remove(KEY_UNDO); remove(KEY_UNDO_AT) }
        _undoAt.value = null
    }

    private fun apply(b: AndroidBackup) = repo.replaceAll(b.config.servers, b.config.settings)

    /**
     * POST /api/backups/push to the server in use (client token, ≤ 1 MiB,
     * still encrypted). Returns the server's name.
     */
    fun push(passphrase: String): String {
        val live = LiveState.live.value
        if (live.state != ConnState.CONNECTED) throw IOException("connect first: the backup goes through the tunnel")
        val server = repo.servers.value.firstOrNull { it.id == live.failover?.activeId } ?: repo.servers.value.firstOrNull()
            ?: throw IOException("not paired")
        val bytes = build(passphrase)
        if (bytes.size > MAX_PUSH) throw IOException("the backup is ${bytes.size / 1024} KiB; the server takes at most 1 MiB")
        val headers = mapOf("Authorization" to "Bearer ${PeersClient.token(server.payload.key)}") + TunnelHttp.clientHeaders(BuildConfig.VERSION_NAME)
        val (code, text) = TunnelHttp.post(
            "http://${ServerList.serverTunnelIp(server.payload)}:8077/api/backups/push", headers, bytes, "application/octet-stream",
        )
        when (code) {
            200, 201, 204 -> return server.name
            404, 405 -> throw IOException("${server.name} doesn't take backups yet (update it)")
            401, 403 -> throw IOException("${server.name} didn't accept this phone's key")
            413 -> throw IOException("${server.name} says the backup is too big")
            else -> throw IOException("${server.name} answered $code ${text.take(120)}".trim())
        }
    }

    companion object {
        const val MAX_PUSH = 1 shl 20
        private const val KEY_UNDO = "undo_sealed"
        private const val KEY_UNDO_AT = "undo_at"
    }
}
