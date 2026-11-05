package dev.tailbond.app.ble

import android.content.Context
import android.os.BatteryManager
import android.os.Build
import androidx.core.content.getSystemService
import dev.tailbond.app.ble.BleProtocol as P
import dev.tailbond.app.ble.HotspotControl.Companion.AP_ENABLED
import dev.tailbond.app.ble.HotspotControl.Companion.AP_ENABLING
import dev.tailbond.app.ble.HotspotControl.Companion.AP_FAILED
import dev.tailbond.app.service.Live
import dev.tailbond.app.service.LiveState
import dev.tailbond.app.service.PathGate
import dev.tailbond.app.share.HubLive
import dev.tailbond.app.share.HubStatus
import kotlinx.serialization.json.JsonElement
import kotlinx.serialization.json.JsonPrimitive

/** "Share bonded connection" on and off (HubService; tests fake it). */
interface HubControl {
    /** Starts it; throws when Android refuses (a foreground service from the background). */
    fun start()
    fun stop()
    fun live(): HubLive
}

/**
 * What a Mac may ask over Bluetooth, done on this phone: the hotspot
 * (through [hotspot], Shizuku), "Share bonded connection" ([hub]), and the
 * phone's status. Blocking: BleControlService calls it off the main thread.
 */
class AppPhoneControl(
    private val ctx: Context,
    private val hotspot: HotspotControl,
    private val hub: HubControl,
    private val live: () -> Live = { LiveState.live.value },
    private val sdk: Int = Build.VERSION.SDK_INT,
    private val waitMs: Long = 20_000,
    private val sleep: (Long) -> Unit = { Thread.sleep(it) },
) : PhoneControl {

    private fun s(v: String): JsonElement = JsonPrimitive(v)

    private fun needShizuku() {
        if (sdk < 30) throw Refused(P.CODE_UNSUPPORTED, "Starting the hotspot needs Android 11 or newer")
        when (hotspot.shizuku()) {
            ShizukuStatus.NOT_RUNNING -> throw Refused(P.CODE_SHIZUKU, "Shizuku isn't running on the phone: start it in the Shizuku app, then try again")
            ShizukuStatus.NO_PERMISSION -> throw Refused(P.CODE_SHIZUKU, "tailbond may not use Shizuku yet: allow it in tailbond → Share on the phone")
            ShizukuStatus.OK -> Unit
        }
    }

    private fun apWord(s: Int) = when (s) {
        HotspotControl.AP_DISABLING -> "stopping"
        HotspotControl.AP_DISABLED -> "off"
        AP_ENABLING -> "starting"
        AP_ENABLED -> "on"
        AP_FAILED -> "error"
        else -> "unknown"
    }

    override fun hotspotOn(): Map<String, JsonElement> {
        needShizuku()
        if (hotspot.apState() != AP_ENABLED) {
            hotspot.start()?.let { throw Refused(P.CODE_HOTSPOT, "Android refused to start the hotspot: $it") }
            var waited = 0L
            while (true) {
                val st = hotspot.apState()
                if (st == AP_ENABLED) break
                val err = hotspot.lastError()
                if (st == AP_FAILED || err != 0) {
                    throw Refused(P.CODE_HOTSPOT, "Android couldn't start the hotspot (error $err): is mobile data on, and does the plan allow tethering?")
                }
                if (waited >= waitMs) throw Refused(P.CODE_HOTSPOT, "The hotspot didn't come up within ${waitMs / 1000} s")
                sleep(250)
                waited += 250
            }
        }
        val (ssid, pass) = hotspot.network() ?: throw Refused(P.CODE_HOTSPOT, "The hotspot is on, but its name couldn't be read")
        return mapOf("state" to s("on"), "ssid" to s(ssid), "pass" to s(pass))
    }

    override fun hotspotOff(): Map<String, JsonElement> {
        needShizuku()
        hotspot.stop()?.let { throw Refused(P.CODE_HOTSPOT, "Android refused to stop the hotspot: $it") }
        return mapOf("state" to s("stopping"))
    }

    override fun hotspotState(): Map<String, JsonElement> {
        val out = mutableMapOf("shizuku" to s(hotspot.shizuku().name.lowercase()))
        if (sdk >= 30 && hotspot.shizuku() == ShizukuStatus.OK) {
            val st = hotspot.apState()
            out["state"] = s(apWord(st))
            if (st == AP_ENABLED) hotspot.network()?.let { out["ssid"] = s(it.first) }
        } else {
            out["state"] = s("unknown")
        }
        return out
    }

    override fun hubOn(): Map<String, JsonElement> {
        if (hub.live().status == HubStatus.OFF) {
            try {
                hub.start()
            } catch (e: Exception) {
                throw Refused(P.CODE_HUB, "Android didn't let tailbond start sharing from the background (${e.javaClass.simpleName}): open tailbond on the phone once, then try again")
            }
            var waited = 0L
            while (hub.live().status == HubStatus.OFF && waited < 5_000) {
                sleep(100)
                waited += 100
            }
        }
        val l = hub.live()
        when (l.status) {
            HubStatus.NO_KEY -> throw Refused(P.CODE_HUB, "This phone has no share key yet: connect tailbond to the home server once")
            HubStatus.ERROR -> throw Refused(P.CODE_HUB, "The proxy couldn't start: ${l.error.ifEmpty { "port 1888 is taken" }}")
            HubStatus.OFF -> throw Refused(P.CODE_HUB, "Share bonded connection didn't start")
            else -> Unit
        }
        return mapOf("hub" to s(l.status.name.lowercase()))
    }

    override fun hubOff(): Map<String, JsonElement> {
        hub.stop()
        return mapOf("hub" to s("off"))
    }

    override fun status(): Map<String, JsonElement> {
        val bm = ctx.getSystemService<BatteryManager>()
        val battery = bm?.getIntProperty(BatteryManager.BATTERY_PROPERTY_CAPACITY) ?: -1
        val l = live()
        // the tunnel's paths in use: what the Mac's traffic is bonded over
        val used = l.paths.filter { it.status == PathGate.STATUS_ACTIVE || it.status == PathGate.STATUS_STANDBY }
        val kinds = used.mapNotNull { it.kind?.label }.distinct()
        return mapOf(
            "battery" to JsonPrimitive(battery),
            "charging" to JsonPrimitive(bm?.isCharging == true),
            "network" to s(if (kinds.isEmpty()) "not connected" else kinds.joinToString(" + ")),
            "paths" to JsonPrimitive(used.count { it.status == PathGate.STATUS_ACTIVE }),
            "tunnel" to s(l.state.name.lowercase()),
            "hub" to s(hub.live().status.name.lowercase()),
        ) + hotspotState().mapKeys { (k, _) -> if (k == "state") "hotspot" else k }
    }
}
