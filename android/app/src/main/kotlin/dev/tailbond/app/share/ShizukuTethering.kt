package dev.tailbond.app.share

/*
 * Adapted from untether's ShizukuHotspotController
 * (https://github.com/ImOstrovskiy/untether). BSD-3-Clause, Copyright (c)
 * 2026 Oleksii Ostrovskyi: the Shizuku binding and state flow shape; the
 * same notice as TetheringShell.kt's header applies. The Apache-2.0
 * tailbond code around it is Tiger Kaovilai's and tailbond contributors'.
 */

import android.content.BroadcastReceiver
import android.content.ComponentName
import android.content.Context
import android.content.Intent
import android.content.IntentFilter
import android.content.pm.PackageManager
import android.os.Build
import android.os.Handler
import android.os.Looper
import kotlinx.coroutines.flow.MutableStateFlow
import kotlinx.coroutines.flow.StateFlow
import kotlinx.coroutines.flow.asStateFlow
import rikka.shizuku.Shizuku

/** What Shizuku is doing: the one click's hotspot half can't work without it. */
enum class TetheringShizuku { OK, NOT_RUNNING, NO_PERMISSION, NEEDS_ANDROID_11 }

/** The hotspot, as the phone's UI and the Mac's one click see it. */
data class TetheringAp(
    /** off, starting, on, stopping, error */
    val state: String = "off",
    val ssid: String = "",
    val pass: String = "",
    /** why Shizuku isn't usable, in plain words; "" when it is */
    val shizuku: String = "",
)

/** The hotspot commands of the BLE control channel (docs/phone-hub.md). */
interface TetheringControl {
    /** Start the hotspot; onDone carries its network, or the error. */
    fun start(onDone: (ssid: String?, pass: String?, error: String?) -> Unit)

    /** Stop the hotspot (whoever started it). */
    fun stop(onDone: (error: String?) -> Unit)

    /** The hotspot now. */
    fun ap(): TetheringAp

    fun close()
}

/**
 * Hotspot control through Shizuku ([TetheringShell], the privileged half),
 * for the Mac's one click. The privileged calls run as the shell user, so
 * no root and no hidden API in the app itself. Without Shizuku running (or
 * below Android 11) it says so and changes nothing: the state is there for
 * the UI to show the Shizuku requirement, not to hide it.
 *
 * Call the public methods on the main thread.
 */
class ShizukuTethering(private val ctx: Context) : TetheringControl {
    private val main = Handler(Looper.getMainLooper())
    private val _ap = MutableStateFlow(TetheringAp())
    /** The hotspot's state, for the UI and the BLE status. */
    val live: StateFlow<TetheringAp> = _ap.asStateFlow()

    private var remote: ITetheringShell? = null
    private var bound = false
    private var closed = false
    /** a start is under way: a failure now is ours, and the done callback waits for the AP */
    private var starting: ((ssid: String?, pass: String?, error: String?) -> Unit)? = null
    private var stopping: ((error: String?) -> Unit)? = null
    private var startTimeout: Runnable? = null
    private var stopTimeout: Runnable? = null

    private val serviceArgs = Shizuku.UserServiceArgs(ComponentName(ctx.packageName, TetheringShell::class.java.name))
        .daemon(false)
        .processNameSuffix("shell")
        // A new install gets a fresh service process with the new code.
        .version((ctx.packageManager.getPackageInfo(ctx.packageName, 0).lastUpdateTime / 1000).toInt())

    private val connection = object : android.content.ServiceConnection {
        override fun onServiceConnected(name: ComponentName, binder: android.os.IBinder) {
            val shell = ITetheringShell.Stub.asInterface(binder)
            remote = shell
            call("Tethering shell") { shell.open(events) }
            readConfig()
        }

        override fun onServiceDisconnected(name: ComponentName) {
            remote = null
            bound = false
            // The process died on its own (Shizuku itself going away is
            // handled by its listeners): bind again.
            if (!closed) main.postDelayed(::refreshShizuku, 1_000)
        }
    }

    /** Arrive on binder threads; the flow is thread-safe. */
    private val events = object : ITetheringEvents.Stub() {
        override fun onLog(line: String) = Unit
        override fun onFailed(error: Int) {
            startDone(null, null, "starting the hotspot was refused (error $error)")
        }
        override fun onNetwork(ssid: String, pass: String) {
            _ap.value = _ap.value.copy(ssid = ssid, pass = pass)
        }
    }

    private val apReceiver = object : BroadcastReceiver() {
        override fun onReceive(c: Context, i: Intent) = onApState(i.getIntExtra(EXTRA_AP_STATE, AP_DISABLED))
    }
    private val onBinder = Shizuku.OnBinderReceivedListener { refreshShizuku() }
    private val onDead = Shizuku.OnBinderDeadListener { refreshShizuku() }
    private val onPermission = Shizuku.OnRequestPermissionResultListener { _, _ -> refreshShizuku() }

    fun open() {
        registerExported(ctx, apReceiver, IntentFilter(ACTION_AP_STATE))
        Shizuku.addBinderReceivedListenerSticky(onBinder)
        Shizuku.addBinderDeadListener(onDead)
        Shizuku.addRequestPermissionResultListener(onPermission)
        refreshShizuku()
    }

    override fun close() {
        closed = true
        main.removeCallbacksAndMessages(null)
        Shizuku.removeBinderReceivedListener(onBinder)
        Shizuku.removeBinderDeadListener(onDead)
        Shizuku.removeRequestPermissionResultListener(onPermission)
        runCatching { ctx.unregisterReceiver(apReceiver) }
        if (bound) runCatching { Shizuku.unbindUserService(serviceArgs, connection, true) }
    }

    override fun ap(): TetheringAp = _ap.value

    override fun start(onDone: (ssid: String?, pass: String?, error: String?) -> Unit) {
        val why = shizukuProblem()
        if (why != null) return onDone(null, null, why)
        val cur = _ap.value
        if (cur.state == "on") {
            readConfig { ssid, pass, err -> onDone(ssid, pass, err) }
            return
        }
        if (starting != null) return onDone(null, null, "the hotspot is already starting")
        starting = onDone
        startTimeout = Runnable { startDone(null, null, "the hotspot didn't come up (it may need starting by hand once, in Settings → Hotspot & tethering)") }
            .also { main.postDelayed(it, START_TIMEOUT_MS) }
        if (!call("Start") { remote!!.start() }) {
            startDone(null, null, "Shizuku's shell service didn't answer")
        }
    }

    override fun stop(onDone: (error: String?) -> Unit) {
        val why = shizukuProblem()
        if (why != null) return onDone(why)
        if (_ap.value.state == "off") return onDone(null)
        if (stopping != null) return onDone("the hotspot is already stopping")
        stopping = onDone
        stopTimeout = Runnable { stopDone("the hotspot didn't stop in time") }.also { main.postDelayed(it, STOP_TIMEOUT_MS) }
        if (!call("Stop") { remote!!.stop() }) {
            stopDone("Shizuku's shell service didn't answer")
        }
    }

    /** The hotspot's network, read back from the system config (never written). */
    private fun readConfig(onDone: ((String?, String?, String?) -> Unit)? = null) {
        val c = runCatching { remote?.config() }.getOrNull()
        when {
            c == null -> onDone?.invoke(null, null, "couldn't read the hotspot's network (set one in Android's hotspot settings)")
            c[0].isNullOrBlank() -> onDone?.invoke(null, null, "the hotspot has no network set up (set its name and password in Android's hotspot settings)")
            else -> {
                _ap.value = _ap.value.copy(ssid = c[0], pass = c[1] ?: "")
                onDone?.invoke(c[0], c[1] ?: "", null)
            }
        }
    }

    private fun startDone(ssid: String?, pass: String?, error: String?) {
        val cb = starting ?: return
        starting = null
        startTimeout?.let { main.removeCallbacks(it) }
        startTimeout = null
        cb(ssid, pass, error)
    }

    private fun stopDone(error: String?) {
        val cb = stopping ?: return
        stopping = null
        stopTimeout?.let { main.removeCallbacks(it) }
        stopTimeout = null
        cb(error)
    }

    private fun onApState(s: Int) {
        when (s) {
            AP_ENABLING -> _ap.value = _ap.value.copy(state = "starting")
            AP_ENABLED -> {
                _ap.value = _ap.value.copy(state = "on")
                readConfig()
                startDone(_ap.value.ssid, _ap.value.pass, null)
            }
            AP_DISABLING -> _ap.value = _ap.value.copy(state = "stopping")
            AP_FAILED -> {
                _ap.value = _ap.value.copy(state = "error")
                startDone(null, null, "the hotspot failed to start (-1)")
            }
            AP_DISABLED -> {
                _ap.value = _ap.value.copy(state = "off")
                stopDone(null)
            }
        }
    }

    /** "" when Shizuku can be used, or why not, in plain words. */
    private fun shizukuProblem(): String? {
        val s = try {
            when {
                !Shizuku.pingBinder() -> TetheringShizuku.NOT_RUNNING
                Shizuku.checkSelfPermission() != PackageManager.PERMISSION_GRANTED -> TetheringShizuku.NO_PERMISSION
                Build.VERSION.SDK_INT < 30 -> TetheringShizuku.NEEDS_ANDROID_11
                else -> TetheringShizuku.OK
            }
        } catch (_: Throwable) {
            // Shizuku itself missing or broken: the same words as not running
            TetheringShizuku.NOT_RUNNING
        }
        _ap.value = _ap.value.copy(shizuku = words(s))
        return when (s) {
            TetheringShizuku.OK -> null
            else -> words(s)
        }
    }

    private fun refreshShizuku() {
        if (closed) return
        val problem = shizukuProblem()
        if (problem == null && !bound) {
            bound = runCatching { Shizuku.bindUserService(serviceArgs, connection) }.isSuccess
            if (!bound) _ap.value = _ap.value.copy(shizuku = "Shizuku's shell service didn't bind")
        }
        if (problem != null) {
            remote = null
            bound = false
        }
    }

    /** Logs the failure of a shell call and returns success. */
    private fun call(what: String, block: () -> String?): Boolean {
        val error = try {
            block()
        } catch (e: Exception) {
            e.toString()
        }
        return error == null
    }

    companion object {
        private const val START_TIMEOUT_MS = 25_000L
        private const val STOP_TIMEOUT_MS = 15_000L

        fun words(s: TetheringShizuku): String = when (s) {
            TetheringShizuku.OK -> ""
            TetheringShizuku.NOT_RUNNING -> "Shizuku isn't running on this phone (start it, or start the hotspot by hand)"
            TetheringShizuku.NO_PERMISSION -> "Shizuku hasn't been allowed for tailbond (Shizuku's notification → Allow)"
            TetheringShizuku.NEEDS_ANDROID_11 -> "the one click's hotspot control needs Android 11+ (start the hotspot by hand)"
        }

        // Hidden WifiManager constants (untether's, kept as they are).
        private const val ACTION_AP_STATE = "android.net.wifi.WIFI_AP_STATE_CHANGED"
        private const val EXTRA_AP_STATE = "wifi_state"
        private const val AP_DISABLING = 10
        private const val AP_DISABLED = 11
        private const val AP_ENABLING = 12
        private const val AP_ENABLED = 13
        private const val AP_FAILED = 14
    }
}

/** Receiver for system broadcasts; the exported flag is required since API 33. */
private fun registerExported(ctx: Context, receiver: BroadcastReceiver, filter: IntentFilter) {
    if (Build.VERSION.SDK_INT >= 33) ctx.registerReceiver(receiver, filter, Context.RECEIVER_EXPORTED)
    else ctx.registerReceiver(receiver, filter)
}
