/*
 * The Shizuku binding follows Untether (https://github.com/ImOstrovskiy/untether),
 * android/app/src/main/kotlin/io/github/imostrovskiy/untether/ShizukuHotspotController.kt.
 * Copyright (c) 2026 Oleksii Ostrovskyi. BSD 3-Clause License; the full
 * notice is in THIRD_PARTY_NOTICES.md at the repository root.
 */
package dev.tailbond.app.ble

import android.content.ComponentName
import android.content.Context
import android.content.ServiceConnection
import android.content.pm.PackageManager
import android.os.Build
import android.os.IBinder
import rikka.shizuku.Shizuku
import java.util.concurrent.CountDownLatch
import java.util.concurrent.TimeUnit

/** Shizuku, as tailbond sees it. */
enum class ShizukuStatus { OK, NOT_RUNNING, NO_PERMISSION }

/** Wi-Fi hotspot control (ShizukuHotspot; tests fake it). */
interface HotspotControl {
    fun shizuku(): ShizukuStatus
    /** null on success, or the error. */
    fun start(): String?
    fun stop(): String?
    /** [AP_ENABLED] etc.; -1 unknown. */
    fun apState(): Int
    /** The last start failure (TETHER_ERROR_*), 0 none. */
    fun lastError(): Int
    /** The hotspot network set in Android: ssid to passphrase. */
    fun network(): Pair<String, String>?

    companion object {
        // WifiManager's hidden AP states
        const val AP_DISABLING = 10
        const val AP_DISABLED = 11
        const val AP_ENABLING = 12
        const val AP_ENABLED = 13
        const val AP_FAILED = 14
    }
}

/**
 * Hotspot control through Shizuku: the privileged calls run in
 * [HotspotShell], a Shizuku user service; this side binds it when needed.
 * Uses the hotspot name and password set in Android's own settings (it
 * changes neither). Call off the main thread: binding waits for the main
 * thread's callback.
 */
class ShizukuHotspot(private val ctx: Context) : HotspotControl {
    @Volatile private var remote: IHotspotShell? = null
    private var bound = false

    private val args = Shizuku.UserServiceArgs(ComponentName(ctx.packageName, HotspotShell::class.java.name))
        .daemon(false)
        .processNameSuffix("hotspot")
        // a new install gets a fresh service process with the new code
        .version((ctx.packageManager.getPackageInfo(ctx.packageName, 0).lastUpdateTime / 1000).toInt())

    private var connected: CountDownLatch? = null
    private val connection = object : ServiceConnection {
        override fun onServiceConnected(name: ComponentName, binder: IBinder) {
            remote = IHotspotShell.Stub.asInterface(binder)
            connected?.countDown()
        }

        override fun onServiceDisconnected(name: ComponentName) {
            remote = null
            bound = false
        }
    }

    override fun shizuku(): ShizukuStatus = status()

    @Synchronized
    private fun shell(): IHotspotShell? {
        remote?.let { return it }
        if (Build.VERSION.SDK_INT < 30 || shizuku() != ShizukuStatus.OK) return null
        val latch = CountDownLatch(1)
        connected = latch
        if (!bound) {
            bound = runCatching { Shizuku.bindUserService(args, connection) }.isSuccess
        }
        latch.await(8, TimeUnit.SECONDS)
        return remote
    }

    private fun shellOrError(): IHotspotShell = shell() ?: throw IllegalStateException("Shizuku's hotspot service didn't start")

    override fun start(): String? = runCatching { shellOrError().startTethering() }.getOrElse { it.message ?: it.toString() }
    override fun stop(): String? = runCatching { shellOrError().stopTethering() }.getOrElse { it.message ?: it.toString() }
    override fun apState(): Int = runCatching { shell()?.apState() }.getOrNull() ?: -1
    override fun lastError(): Int = runCatching { shell()?.lastError() }.getOrNull() ?: 0
    override fun network(): Pair<String, String>? =
        runCatching { shell()?.network() }.getOrNull()?.takeIf { it.size == 2 && it[0].isNotEmpty() }?.let { it[0] to it[1] }

    fun close() {
        if (bound) runCatching { Shizuku.unbindUserService(args, connection, true) }
        bound = false
        remote = null
    }

    companion object {
        fun status(): ShizukuStatus = when {
            !runCatching { Shizuku.pingBinder() }.getOrDefault(false) -> ShizukuStatus.NOT_RUNNING
            runCatching { Shizuku.checkSelfPermission() }.getOrDefault(PackageManager.PERMISSION_DENIED) != PackageManager.PERMISSION_GRANTED -> ShizukuStatus.NO_PERMISSION
            else -> ShizukuStatus.OK
        }

        /** Asks Shizuku for its permission (Shizuku shows the dialog); the result updates [MacLinkState]. */
        fun requestPermission() {
            if (status() != ShizukuStatus.NO_PERMISSION) return
            Shizuku.addRequestPermissionResultListener(object : Shizuku.OnRequestPermissionResultListener {
                override fun onRequestPermissionResult(requestCode: Int, grantResult: Int) {
                    Shizuku.removeRequestPermissionResultListener(this)
                    MacLinkState.set { it.copy(shizuku = status()) }
                }
            })
            runCatching { Shizuku.requestPermission(REQUEST_CODE) }
        }

        private const val REQUEST_CODE = 0x7b1e
    }
}
