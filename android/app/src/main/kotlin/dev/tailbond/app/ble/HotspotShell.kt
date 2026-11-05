/*
 * Adapted from Untether (https://github.com/ImOstrovskiy/untether),
 * android/app/src/main/kotlin/io/github/imostrovskiy/untether/HotspotShell.kt:
 * only the start, stop, state and network calls are kept.
 *
 * Copyright (c) 2026 Oleksii Ostrovskyi. All rights reserved.
 * Untether's approach follows delta (https://github.com/supershadoe/delta),
 * Copyright 2024, 2025 supershadoe.
 *
 * Redistribution and use in source and binary forms, with or without
 * modification, are permitted under the BSD 3-Clause License; the full
 * notice is in THIRD_PARTY_NOTICES.md at the repository root.
 */
package dev.tailbond.app.ble

import android.annotation.SuppressLint
import android.content.Context
import android.content.ContextWrapper
import android.net.TetheringManager
import android.net.wifi.SoftApConfiguration
import android.net.wifi.WifiManager
import android.os.Build
import android.os.IBinder
import android.os.Looper
import java.util.function.Supplier
import kotlin.system.exitProcess

/**
 * The privileged half of hotspot control: a Shizuku user service, that is a
 * process Shizuku starts from this APK as the shell user (uid 2000, which
 * holds TETHER_PRIVILEGED and NETWORK_SETTINGS). Hidden API checks are off in
 * such processes, so the framework's tethering and Wi-Fi classes are built
 * and called directly; the app itself uses no hidden API.
 *
 * NewApi: TetheringManager and its request became public in API 36 but exist
 * as @SystemApi since API 30; the app only binds this service on API 30+.
 * MissingPermission: the shell user holds them.
 */
@SuppressLint("NewApi", "InlinedApi", "MissingPermission", "PrivateApi", "DiscouragedPrivateApi")
class HotspotShell(private val ctx: Context) : IHotspotShell.Stub() {
    private var tm: TetheringManager? = null
    private var wm: WifiManager? = null
    @Volatile private var failed = 0

    /** TetheringManager/WifiManager take callerPkg from the context; it must match the calling uid (shell). */
    private val shellContext = object : ContextWrapper(ctx) {
        override fun getOpPackageName() = SHELL_PACKAGE
        override fun getAttributionTag(): String? = null
    }

    override fun destroy() = exitProcess(0)

    @Synchronized
    private fun open() {
        if (tm == null) {
            val tethering = service("tethering")
            tm = TetheringManager::class.java.getConstructor(Context::class.java, Supplier::class.java)
                .newInstance(shellContext, Supplier { tethering }) as TetheringManager
        }
        if (wm == null) {
            val wifi = asInterface("android.net.wifi.IWifiManager", service("wifi"))
            // (Context, IWifiManager), or with a Looper on newer releases
            val ctor = WifiManager::class.java.constructors.first { it.parameterCount in 2..3 && it.parameterTypes[0] == Context::class.java }
            wm = ctor.newInstance(*arrayOf<Any?>(shellContext, wifi, Looper.getMainLooper()).copyOf(ctor.parameterCount)) as WifiManager
        }
    }

    @Synchronized
    override fun startTethering(): String? = attempt {
        open()
        failed = 0
        val request = TetheringManager.TetheringRequest.Builder(TetheringManager.TETHERING_WIFI).build()
        tm!!.startTethering(request, ctx.mainExecutor, object : TetheringManager.StartTetheringCallback {
            override fun onTetheringStarted() = Unit
            override fun onTetheringFailed(error: Int) {
                failed = error
            }
        })
    }

    @Synchronized
    override fun stopTethering(): String? = attempt {
        open()
        // unlike stopTethering(TetheringRequest) this also stops a hotspot started from quick settings
        tm!!.call("stopTethering", TetheringManager.TETHERING_WIFI)
    }

    @Synchronized
    override fun apState(): Int = runCatching {
        open()
        wm!!.call("getWifiApState") as Int
    }.getOrDefault(-1)

    override fun lastError(): Int = failed

    @Synchronized
    override fun network(): Array<String>? = runCatching {
        open()
        val c = wm!!.call("getSoftApConfiguration") as SoftApConfiguration
        val ssid = if (Build.VERSION.SDK_INT >= 33) c.wifiSsid?.bytes?.decodeToString().orEmpty() else @Suppress("DEPRECATION") c.ssid.orEmpty()
        arrayOf(ssid, c.passphrase.orEmpty())
    }.getOrNull()

    /** Runs [block]; an exception becomes the error string (only a few exception types cross binder). */
    private inline fun attempt(block: () -> Unit): String? = try {
        block()
        null
    } catch (e: Throwable) {
        (if (e is java.lang.reflect.InvocationTargetException) e.targetException else e).toString()
    }

    private companion object {
        const val SHELL_PACKAGE = "com.android.shell"

        fun service(name: String): IBinder =
            Class.forName("android.os.ServiceManager").getMethod("getService", String::class.java).invoke(null, name) as IBinder

        /** `Stub.asInterface(binder)` for a hidden AIDL interface. */
        fun asInterface(aidl: String, binder: IBinder): Any =
            Class.forName("$aidl\$Stub").getMethod("asInterface", IBinder::class.java).invoke(null, binder)!!

        /** Calls a (possibly hidden) public method by name and arity. */
        fun Any.call(name: String, vararg args: Any?): Any? =
            javaClass.methods.first { it.name == name && it.parameterCount == args.size }.invoke(this, *args)
    }
}
