package dev.tailbond.app.update

import android.annotation.SuppressLint
import android.app.NotificationChannel
import android.app.NotificationManager
import android.app.PendingIntent
import android.content.Context
import android.content.Intent
import android.net.Uri
import androidx.core.app.NotificationCompat
import androidx.core.content.getSystemService
import dev.tailbond.app.R
import androidx.core.content.edit
import dev.tailbond.app.BuildConfig
import dev.tailbond.app.data.Repository
import kotlinx.coroutines.CoroutineScope
import kotlinx.coroutines.Dispatchers
import kotlinx.coroutines.ExperimentalCoroutinesApi
import kotlinx.coroutines.SupervisorJob
import kotlinx.coroutines.flow.StateFlow
import kotlinx.coroutines.launch
import java.io.ByteArrayOutputStream
import java.io.File
import java.io.IOException
import java.net.HttpURLConnection
import java.net.URL

/**
 * Process-wide: runs [UpdateChecker] on one background thread. The UI calls
 * [onForeground] when the app comes to the front or Settings → Updates
 * opens (a check runs when the last one is a day old), and [checkNow].
 */
@OptIn(ExperimentalCoroutinesApi::class)
class UpdateManager private constructor(private val ctx: Context) {
    private val repo = Repository.get(ctx)
    private val prefs = ctx.getSharedPreferences("tailbond_update", Context.MODE_PRIVATE)
    private val scope = CoroutineScope(SupervisorJob() + Dispatchers.IO.limitedParallelism(1))

    private val checker = UpdateChecker(
        object : UpdateEnv {
            override val currentVersion: String = BuildConfig.VERSION_NAME
            override val repo: String = UPDATE_REPO
            override fun policy() = this@UpdateManager.repo.settings.value.updatePolicy
            override fun now() = System.currentTimeMillis()
            override fun load() = UpdateStatus.fromJson(prefs.getString(KEY_STATUS, null))
            override fun save(s: UpdateStatus) = prefs.edit { putString(KEY_STATUS, s.toJson()) }
        },
        GitHubHttp,
    )

    val status: StateFlow<UpdateStatus> = checker.status

    init {
        // left over from the old self-updater (downloaded APKs)
        scope.launch { File(ctx.cacheDir, "updates").deleteRecursively() }
    }

    /** When the app comes to the front, and every few minutes from the tunnel's service: checks when due. */
    fun onForeground() {
        scope.launch {
            checker.checkIfDue()
            notifyIfReady() // also after switching to Right away with a release already found
        }
    }

    fun checkNow() {
        scope.launch {
            checker.check()
            notifyIfReady()
        }
    }

    /** Right away: "Update ready: tap to install", once per version (UpdateRules.notifyUpdate). */
    private fun notifyIfReady() {
        val s = checker.status.value
        val policy = repo.settings.value.updatePolicy
        if (!UpdateRules.notifyUpdate(policy, s, prefs.getString(KEY_NOTIFIED, null))) return
        val apk = s.apkUrl ?: return
        val nm = ctx.getSystemService<NotificationManager>() ?: return
        nm.createNotificationChannel(NotificationChannel(CHANNEL, "Updates", NotificationManager.IMPORTANCE_DEFAULT))
        val tap = PendingIntent.getActivity(
            ctx, 4, Intent(Intent.ACTION_VIEW, Uri.parse(apk)).addFlags(Intent.FLAG_ACTIVITY_NEW_TASK), PendingIntent.FLAG_IMMUTABLE,
        )
        val n = NotificationCompat.Builder(ctx, CHANNEL)
            .setSmallIcon(R.drawable.ic_stat_tailbond)
            .setContentTitle(UpdateWords.NOTIFY_TITLE)
            .setContentText(UpdateWords.notifyText(s.latest.orEmpty()))
            .setStyle(NotificationCompat.BigTextStyle().bigText(UpdateWords.notifyText(s.latest.orEmpty())))
            .setContentIntent(tap)
            .setAutoCancel(true)
            .build()
        try {
            nm.notify(NOTIFICATION_ID, n)
        } catch (_: SecurityException) {
            return // notifications not allowed: the Updates page still shows it
        }
        prefs.edit { putString(KEY_NOTIFIED, s.latest) }
    }

    /** HTTPS GET with HttpURLConnection; at most 1 MiB of answer. */
    private object GitHubHttp : HttpsGet {
        override fun get(url: String, headers: Map<String, String>): Pair<Int, String> {
            val u = URL(url)
            if (u.protocol != "https") throw IOException("refusing $url: https only")
            val c = u.openConnection() as HttpURLConnection
            try {
                c.connectTimeout = 10_000
                c.readTimeout = 15_000
                headers.forEach { (k, v) -> c.setRequestProperty(k, v) }
                val code = c.responseCode
                if (code != 200) return code to ""
                val out = ByteArrayOutputStream()
                c.inputStream.use { input ->
                    val buf = ByteArray(16 * 1024)
                    while (true) {
                        val n = input.read(buf)
                        if (n < 0) break
                        out.write(buf, 0, n)
                        if (out.size() > MAX_ANSWER) throw IOException("GitHub's answer is too long")
                    }
                }
                return code to out.toString("UTF-8")
            } finally {
                c.disconnect()
            }
        }
    }

    companion object {
        /** BuildConfig.UPDATE_REPO (Gradle TB_UPDATE_REPO), already checked by Gradle. */
        val UPDATE_REPO: String = UpdateRules.repo(BuildConfig.UPDATE_REPO) ?: "kaovilai/tailbond"
        private const val KEY_STATUS = "status"
        private const val KEY_NOTIFIED = "notified_version"
        private const val CHANNEL = "updates"
        private const val NOTIFICATION_ID = 5
        private const val MAX_ANSWER = 1 shl 20

        // holds the application context only (see get)
        @SuppressLint("StaticFieldLeak")
        @Volatile private var instance: UpdateManager? = null

        fun get(ctx: Context): UpdateManager =
            instance ?: synchronized(this) {
                instance ?: UpdateManager(ctx.applicationContext).also { instance = it }
            }
    }
}
