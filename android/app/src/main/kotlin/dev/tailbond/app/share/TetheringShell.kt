package dev.tailbond.app.share

/*
 * Adapted from untether's HotspotShell and ShizukuHotspotController
 * (https://github.com/ImOstrovskiy/untether), and through them
 * supershadoe/delta's approach.
 *
 * Copyright (c) 2026 Oleksii Ostrovskyi
 *
 * Redistribution and use in source and binary forms, with or without
 * modification, are permitted provided that the following conditions are met:
 * 1. Redistributions of source code must retain the above copyright notice,
 *    this list of conditions and the following disclaimer.
 * 2. Redistributions in binary form must reproduce the above copyright notice,
 *    this list of conditions and the following disclaimer in the documentation
 *    and/or other materials provided with the distribution.
 * 3. Neither the name of the copyright holder nor the names of its contributors
 *    may be used to endorse or promote products derived from this software
 *    without specific prior written permission.
 * THIS SOFTWARE IS PROVIDED BY THE COPYRIGHT HOLDERS AND CONTRIBUTORS "AS IS"
 * AND ANY EXPRESS OR IMPLIED WARRANTIES, INCLUDING, BUT NOT LIMITED TO, THE
 * IMPLIED WARRANTIES OF MERCHANTABILITY AND FITNESS FOR A PARTICULAR PURPOSE
 * ARE DISCLAIMED. IN NO EVENT SHALL THE COPYRIGHT HOLDER OR CONTRIBUTORS BE
 * LIABLE FOR ANY DIRECT, INDIRECT, INCIDENTAL, SPECIAL, EXEMPLARY, OR
 * CONSEQUENTIAL DAMAGES (INCLUDING, BUT NOT LIMITED TO, PROCUREMENT OF
 * SUBSTITUTE GOODS OR SERVICES; LOSS OF USE, DATA, OR PROFITS; OR BUSINESS
 * INTERRUPTION) HOWEVER CAUSED AND ON ANY THEORY OF LIABILITY, WHETHER IN
 * CONTRACT, STRICT LIABILITY, OR TORT (INCLUDING NEGLIGENCE OR OTHERWISE)
 * ARISING IN ANY WAY OUT OF THE USE OF THIS SOFTWARE, EVEN IF ADVISED OF THE
 * POSSIBILITY OF SUCH DAMAGE. (BSD-3-Clause; the Apache-2.0 tailbond code
 * around it is Tiger Kaovilai's and tailbond contributors'.)
 */

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
 * The privileged half of hotspot control, for the Mac's one click
 * (docs/phone-hub.md): a Shizuku user service, that is a process Shizuku
 * starts from this APK as the shell user. Hidden API checks are off in such
 * processes, so the framework's tethering and Wi-Fi internals are called
 * directly, with the shell user's permissions, and the app itself needs no
 * hidden API at all.
 *
 * What tailbond needs is a fraction of untether's: start and stop the
 * hotspot with the network the user already has in Android's settings
 * (read back, never written), and tell the app what that network is.
 *
 * NewApi: TetheringManager (with TetheringRequest and the event callback)
 * and SoftApConfiguration became public in API 36 but exist as @SystemApi
 * since API 30; below API 30 there is no TetheringManager at all, so the
 * whole feature needs Android 11+ (checked in ShizukuTethering).
 * MissingPermission: the shell user holds them.
 */
@SuppressLint("NewApi", "InlinedApi", "MissingPermission", "PrivateApi")
class TetheringShell(private val ctx: Context) : ITetheringShell.Stub() {
    @Volatile private var events: ITetheringEvents? = null
    private var tm: TetheringManager? = null
    private var wm: WifiManager? = null

    /** TetheringManager/WifiManager take callerPkg from the context; it must match the calling uid (shell). */
    private val shellContext = object : ContextWrapper(ctx) {
        override fun getOpPackageName() = SHELL_PACKAGE
        override fun getAttributionTag(): String? = null
    }

    override fun destroy() = exitProcess(0)

    @Synchronized
    override fun open(events: ITetheringEvents): String? = attempt {
        this.events = events
        if (Build.VERSION.SDK_INT < 30) return "hotspot control needs Android 11+ (TetheringManager)"
        if (tm == null) {
            val tethering = service("tethering")
            tm = TetheringManager::class.java.getConstructor(Context::class.java, Supplier::class.java)
                .newInstance(shellContext, Supplier { tethering }) as TetheringManager
        }
        if (wm == null) {
            val wifi = asInterface("android.net.wifi.IWifiManager", service("wifi"))
            // (Context, IWifiManager) on Android 17; AOSP main adds a Looper.
            val ctor = WifiManager::class.java.constructors.first { it.parameterCount in 2..3 && it.parameterTypes[0] == Context::class.java }
            wm = ctor.newInstance(*arrayOf<Any?>(shellContext, wifi, Looper.getMainLooper()).copyOf(ctor.parameterCount)) as WifiManager
        }
    }

    @Synchronized
    override fun config(): Array<String>? {
        var net: Array<String>? = null
        val error = attempt {
            val c = wm!!.call("getSoftApConfiguration") as SoftApConfiguration
            val ssid = if (Build.VERSION.SDK_INT >= 33) c.wifiSsid?.bytes?.decodeToString().orEmpty() else @Suppress("DEPRECATION") c.ssid.orEmpty()
            val pass = c.passphrase.orEmpty()
            net = arrayOf(ssid, pass)
            events?.onNetwork(ssid, pass)
        }
        return if (error == null) net else null
    }

    @Synchronized
    override fun start(): String? = attempt {
        val request = TetheringManager.TetheringRequest.Builder(TetheringManager.TETHERING_WIFI).build()
        tm!!.startTethering(request, ctx.mainExecutor, object : TetheringManager.StartTetheringCallback {
            override fun onTetheringStarted() = log("Tethering started")
            override fun onTetheringFailed(error: Int) {
                log("Tethering failed: $error")
                events?.onFailed(error)
            }
        })
    }

    @Synchronized
    override fun stop(): String? = attempt {
        // Unlike stopTethering(TetheringRequest) this also stops a hotspot started from quick settings.
        tm!!.call("stopTethering", TetheringManager.TETHERING_WIFI)
    }

    override fun apState(): Int = runCatching { wm!!.call("getWifiApState") as Int }.getOrDefault(-1)

    private fun log(line: String) {
        runCatching { events?.onLog(line) }
    }

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
    }
}

/** Calls a (possibly hidden) public method by name and arity. */
fun Any.call(name: String, vararg args: Any?): Any? =
    javaClass.methods.first { it.name == name && it.parameterCount == args.size }.invoke(this, *args)
