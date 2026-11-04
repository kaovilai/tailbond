package dev.tailbond.app.report

import android.app.ActivityManager
import android.app.ApplicationExitInfo
import android.content.Context
import android.os.Build
import androidx.core.content.edit
import dev.tailbond.app.BuildConfig
import java.io.File
import java.io.PrintWriter
import java.io.StringWriter
import java.time.Instant

/**
 * Crash capture (docs/reporting.md): an uncaught exception's stack, and on
 * Android 11+ the system's record of a native crash or an ANR, kept in
 * files/crash-last.txt until the user reports or dismisses it. Nothing
 * leaves the phone unless the user opens the report themselves.
 */
object CrashCapture {
    private const val FILE = "crash-last.txt"
    private const val PREFS = "crash"
    private const val SEEN = "exit_seen_ms"

    fun file(ctx: Context) = File(ctx.filesDir, FILE)

    /** Application.onCreate: chain an uncaught-exception handler that writes the file first. */
    fun install(ctx: Context) {
        val app = ctx.applicationContext
        val prev = Thread.getDefaultUncaughtExceptionHandler()
        Thread.setDefaultUncaughtExceptionHandler { t, e ->
            try {
                val sw = StringWriter()
                e.printStackTrace(PrintWriter(sw))
                write(app, "tailbond ${BuildConfig.VERSION_NAME} stopped: uncaught ${e.javaClass.name} on thread ${t.name} at ${Instant.now()}\n$sw")
            } catch (_: Throwable) {
            }
            prev?.uncaughtException(t, e)
        }
        // the system's exit records: native crashes (mqvpn) and ANRs the handler can't see
        if (Build.VERSION.SDK_INT >= 30) {
            try {
                recordExits(app)
            } catch (_: Throwable) {
            }
        }
    }

    private fun write(ctx: Context, text: String) {
        file(ctx).writeText(text.take(64 * 1024))
    }

    @androidx.annotation.RequiresApi(30)
    private fun recordExits(ctx: Context) {
        val am = ctx.getSystemService(ActivityManager::class.java) ?: return
        val prefs = ctx.getSharedPreferences(PREFS, Context.MODE_PRIVATE)
        val seen = prefs.getLong(SEEN, 0L)
        val exits = am.getHistoricalProcessExitReasons(ctx.packageName, 0, 5)
        val newest = exits.maxOfOrNull { it.timestamp } ?: return
        if (seen == 0L) { // first run with this: don't report old history
            prefs.edit { putLong(SEEN, newest) }
            return
        }
        val bad = exits.filter {
            it.timestamp > seen && (it.reason == ApplicationExitInfo.REASON_CRASH_NATIVE || it.reason == ApplicationExitInfo.REASON_ANR)
        }.maxByOrNull { it.timestamp }
        prefs.edit { putLong(SEEN, maxOf(seen, newest)) }
        if (bad == null || file(ctx).exists()) return
        val kind = if (bad.reason == ApplicationExitInfo.REASON_ANR) "stopped responding (ANR)" else "crashed in native code"
        val trace = try {
            bad.traceInputStream?.bufferedReader()?.use { r -> r.readText().take(48 * 1024) }
        } catch (_: Exception) {
            null
        }.orEmpty()
        write(ctx, "tailbond ${BuildConfig.VERSION_NAME} $kind at ${Instant.ofEpochMilli(bad.timestamp)}: ${bad.description.orEmpty()}\n$trace")
    }

    /** The kept crash and when it was written, or null. */
    fun last(ctx: Context): Pair<String, Long>? {
        val f = file(ctx)
        if (!f.exists() || f.length() == 0L) return null
        return try { f.readText() to f.lastModified() } catch (_: Exception) { null }
    }

    fun dismiss(ctx: Context) {
        file(ctx).delete()
    }

    /** This process's recent log lines (logcat may show an app its own lines). */
    fun recentLogs(lines: Int = 300): String = try {
        val p = ProcessBuilder("logcat", "-d", "-v", "time", "-t", lines.toString(), "--pid=" + android.os.Process.myPid())
            .redirectErrorStream(true).start()
        val out = p.inputStream.bufferedReader().use { it.readText() }
        p.waitFor()
        out
    } catch (_: Exception) {
        ""
    }

    /** The issue form's "Device and firmware / OS". */
    fun device(): String = "${Build.MANUFACTURER} ${Build.MODEL}, Android ${Build.VERSION.RELEASE} (API ${Build.VERSION.SDK_INT})"

    fun version(): String = "${BuildConfig.VERSION_NAME} (${BuildConfig.VERSION_CODE})"
}
