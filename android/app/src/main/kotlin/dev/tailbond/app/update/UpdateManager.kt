package dev.tailbond.app.update

import android.Manifest
import android.annotation.SuppressLint
import android.app.NotificationChannel
import android.app.NotificationManager
import android.app.PendingIntent
import android.content.Context
import android.content.Intent
import android.content.pm.PackageManager
import android.os.Build
import androidx.core.app.NotificationCompat
import androidx.core.app.NotificationManagerCompat
import androidx.core.content.ContextCompat
import androidx.core.content.edit
import dev.tailbond.app.BuildConfig
import dev.tailbond.app.R
import dev.tailbond.app.data.PeersClient
import dev.tailbond.app.data.Repository
import dev.tailbond.app.data.TunnelHttp
import dev.tailbond.app.model.ServerList
import dev.tailbond.app.model.ServerProfile
import dev.tailbond.app.service.ConnState
import dev.tailbond.app.service.LiveState
import dev.tailbond.app.ui.MainActivity
import kotlinx.coroutines.CoroutineScope
import kotlinx.coroutines.Dispatchers
import kotlinx.coroutines.ExperimentalCoroutinesApi
import kotlinx.coroutines.SupervisorJob
import kotlinx.coroutines.channels.BufferOverflow
import kotlinx.coroutines.delay
import kotlinx.coroutines.flow.MutableSharedFlow
import kotlinx.coroutines.flow.SharedFlow
import kotlinx.coroutines.flow.StateFlow
import kotlinx.coroutines.flow.distinctUntilChanged
import kotlinx.coroutines.flow.drop
import kotlinx.coroutines.flow.map
import kotlinx.coroutines.launch
import kotlinx.serialization.json.Json
import kotlinx.serialization.json.int
import kotlinx.serialization.json.jsonObject
import kotlinx.serialization.json.jsonPrimitive
import java.io.ByteArrayOutputStream
import java.io.File
import java.net.URI

/**
 * Process-wide updater: runs [Updater] on one background thread, checks
 * every 6 h while connected and when the app opens, retries a deferred
 * install every minute, and turns PackageInstaller's answers into state and
 * notifications. Started by the UI and by the VPN service.
 */
@OptIn(ExperimentalCoroutinesApi::class)
class UpdateManager private constructor(private val ctx: Context) {
    private val repo = Repository.get(ctx)
    private val prefs = ctx.getSharedPreferences("tailbond_update", Context.MODE_PRIVATE)
    private val worker = Dispatchers.IO.limitedParallelism(1)
    private val scope = CoroutineScope(SupervisorJob() + worker)
    val installer = PlatformInstaller(ctx)
    private val updater = Updater(Env(), installer)

    val status: StateFlow<UpdateStatus> = updater.status

    /** Android's "Update this app?" screen, while a session waits for that tap. */
    @Volatile var confirmIntent: Intent? = null
        private set
    private val _confirmNow = MutableSharedFlow<Intent>(extraBufferCapacity = 1, onBufferOverflow = BufferOverflow.DROP_OLDEST)
    /** Shown at once when the user started the install themselves (the activity collects it while visible). */
    val confirmNow: SharedFlow<Intent> = _confirmNow
    @Volatile private var manualInstall = false
    @Volatile private var started = false

    private inner class Env : UpdateEnv {
        override val currentVersion: String = BuildConfig.VERSION_NAME
        override val currentCode: Long = BuildConfig.VERSION_CODE.toLong()
        override val debugBuild: Boolean = BuildConfig.DEBUG
        override fun hasSource(): Boolean = repo.servers.value.isNotEmpty() || publicSource() != null
        override fun policy() = repo.settings.value.updatePolicy
        override fun channel() = repo.settings.value.updateChannel

        override fun sources(): List<UpdateSource> = listOfNotNull(mirror(), publicSource())
        override fun downloadSources(): List<UpdateSource> =
            listOfNotNull(mirror(), UpdateSource(UpdateSource.PUBLIC, "", UrlHttp, label = "the release's download link"))

        override fun serverProtocol(): Int? {
            val server = connectedServer() ?: return null
            return try {
                val out = ByteArrayOutputStream()
                val code = TunnelHttp.get("http://${ServerList.serverTunnelIp(server.payload)}:8077/api/hello", emptyMap(), out, 64 * 1024L) {}
                if (code != 200) return null
                val o = Json.parseToJsonElement(out.toString("UTF-8")).jsonObject
                o["protocol"]?.jsonPrimitive?.int ?: 1
            } catch (_: Exception) {
                null
            }
        }

        override fun busyKbps(): Long {
            val l = LiveState.live.value
            return if (l.state == ConnState.CONNECTED) UpdateRules.busyKbps(l.rxHistory, l.txHistory) else 0
        }

        override fun now(): Long = System.currentTimeMillis()

        override fun apkFile(name: String): File {
            val dir = File(ctx.cacheDir, "updates")
            dir.listFiles()?.forEach { it.delete() }
            dir.mkdirs()
            return File(dir, name.replace(Regex("[^A-Za-z0-9._-]"), "_").ifEmpty { "tailbond.apk" })
        }

        override fun load(): UpdateStatus? = UpdateStatus.fromJson(prefs.getString(KEY_STATUS, null))
        override fun save(s: UpdateStatus) = prefs.edit { putString(KEY_STATUS, s.toJson()) }

        override fun beforeInstall() {
            val st = LiveState.live.value.state
            val up = st == ConnState.CONNECTED || st == ConnState.CONNECTING || st == ConnState.RECONNECTING
            prefs.edit(commit = true) { putBoolean(KEY_RECONNECT, up) }
        }

        override fun announce(s: UpdateStatus) = notify(s)
    }

    /** The server the tunnel uses right now; null while not connected. */
    private fun connectedServer(): ServerProfile? {
        val l = LiveState.live.value
        if (l.state != ConnState.CONNECTED) return null
        val list = repo.servers.value
        return list.firstOrNull { it.id == l.failover?.activeId } ?: list.firstOrNull()
    }

    private fun mirror(): UpdateSource? {
        val server = connectedServer() ?: return null
        val host = ServerList.serverTunnelIp(server.payload)
        val headers = mapOf("Authorization" to "Bearer ${PeersClient.token(server.payload.key)}") + TunnelHttp.clientHeaders(BuildConfig.VERSION_NAME)
        return UpdateSource(UpdateSource.MIRROR, "http://$host:8077/updates", TunnelHttp, headers, label = server.name)
    }

    private fun publicSource(): UpdateSource? {
        val base = BuildConfig.UPDATE_SOURCE.trim()
        // https only (Gradle refuses anything else too): integrity is the manifest's SHA-256 over TLS
        if (!base.startsWith("https://")) return null
        val label = runCatching { URI(base).host }.getOrNull() ?: "the public source"
        return UpdateSource(UpdateSource.PUBLIC, base, UrlHttp, label = label)
    }

    /** Starts the schedule once per process. */
    fun start() {
        if (started) return
        started = true
        scope.launch {
            // policy/channel changes
            repo.settings.map { it.updatePolicy to it.updateChannel }.distinctUntilChanged().drop(1).collect {
                updater.settingsChanged()
                if (it.first != UpdatePolicy.OFF) updater.check(manual = false)
            }
        }
        scope.launch {
            // pairing the first server (or removing the last) turns updates on or off
            repo.servers.map { it.isNotEmpty() }.distinctUntilChanged().collect { updater.settingsChanged() }
        }
        scope.launch {
            var first = true
            while (true) {
                val connected = LiveState.live.value.state == ConnState.CONNECTED
                val s = updater.status.value
                when {
                    // when the app (process) starts, then every 6 h while connected
                    first && updater.checkDue(APP_START_GAP_MS) -> updater.check(manual = false)
                    connected && updater.checkDue() -> updater.check(manual = false)
                    connected && s.phase == Phase.WAITING -> updater.evaluate()
                }
                first = false
                delay(60_000)
            }
        }
    }

    fun checkNow() = scope.launch { updater.check(manual = true) }

    fun installNow() = scope.launch {
        manualInstall = true
        updater.install(manual = true)
    }

    /** "Finish installing": Android's own prompt, or a new session when that prompt is gone. */
    fun finishInstall(from: Context) {
        val c = confirmIntent
        if (c != null) {
            confirmIntent = null
            try { from.startActivity(c); return } catch (_: Exception) {}
        }
        scope.launch { updater.tapLost(); manualInstall = true; updater.install(manual = true) }
    }

    fun onInstallStatus(status: Int, message: String?, confirm: Intent?) {
        scope.launch {
            val r = InstallResult.of(status, message)
            if (r == InstallResult.NeedsUserAction) {
                confirmIntent = confirm?.addFlags(Intent.FLAG_ACTIVITY_NEW_TASK)
                if (manualInstall && confirm != null) _confirmNow.tryEmit(confirm)
            } else {
                confirmIntent = null
                manualInstall = false
                cancelNotification()
            }
            updater.onInstallResult(r)
        }
    }

    /** True once after an update that interrupted a running tunnel. */
    fun takeReconnectFlag(): Boolean {
        val v = prefs.getBoolean(KEY_RECONNECT, false)
        if (v) prefs.edit { putBoolean(KEY_RECONNECT, false) }
        return v
    }

    private fun notify(s: UpdateStatus) {
        if (Build.VERSION.SDK_INT >= 33 &&
            ContextCompat.checkSelfPermission(ctx, Manifest.permission.POST_NOTIFICATIONS) != PackageManager.PERMISSION_GRANTED
        ) return
        val (title, text) = when (s.phase) {
            Phase.NEEDS_TAP -> "Tap to finish updating tailbond" to "Android asks once before tailbond can update itself. Version ${s.installing ?: s.available}"
            Phase.NEEDS_PERMISSION -> "tailbond ${s.available} is ready" to "Allow tailbond to install updates: tap here, then Settings"
            Phase.FAILED -> "tailbond couldn't update" to s.lastResult
            else -> if (s.available != null) "tailbond ${s.available} is available" to "Tap to see what's new and install it" else return
        }
        val nm = ctx.getSystemService(NotificationManager::class.java)
        nm.createNotificationChannel(NotificationChannel(CHANNEL, "Updates", NotificationManager.IMPORTANCE_DEFAULT))
        val open = PendingIntent.getActivity(
            ctx, 7,
            Intent(ctx, MainActivity::class.java).setAction(if (s.phase == Phase.NEEDS_TAP) ACTION_FINISH else ACTION_OPEN)
                .addFlags(Intent.FLAG_ACTIVITY_SINGLE_TOP),
            PendingIntent.FLAG_UPDATE_CURRENT or PendingIntent.FLAG_IMMUTABLE,
        )
        val n = NotificationCompat.Builder(ctx, CHANNEL)
            .setSmallIcon(R.drawable.ic_stat_tailbond)
            .setContentTitle(title)
            .setContentText(text)
            .setStyle(NotificationCompat.BigTextStyle().bigText(text))
            .setContentIntent(open)
            .setAutoCancel(true)
            .build()
        try { NotificationManagerCompat.from(ctx).notify(NOTE_ID, n) } catch (_: SecurityException) {}
    }

    private fun cancelNotification() = NotificationManagerCompat.from(ctx).cancel(NOTE_ID)

    companion object {
        const val ACTION_OPEN = "dev.tailbond.app.OPEN_UPDATES"
        const val ACTION_FINISH = "dev.tailbond.app.FINISH_UPDATE"
        private const val CHANNEL = "updates"
        private const val NOTE_ID = 7
        private const val APP_START_GAP_MS = 30 * 60_000L
        private const val KEY_STATUS = "status"
        private const val KEY_RECONNECT = "reconnect_after_update"

        // holds the application context only (see get)
        @SuppressLint("StaticFieldLeak")
        @Volatile private var instance: UpdateManager? = null

        fun get(ctx: Context): UpdateManager =
            instance ?: synchronized(this) {
                instance ?: UpdateManager(ctx.applicationContext).also { instance = it }
            }
    }
}
