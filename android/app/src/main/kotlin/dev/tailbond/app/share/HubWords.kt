package dev.tailbond.app.share

import dev.tailbond.app.ui.formatBytes

/** What "Share bonded connection" says, in plain words (notification and screen). */
object HubWords {
    private fun devices(n: Int) = if (n == 1) "1 device" else "$n devices"

    /** The headline: what is happening now. */
    fun headline(l: HubLive): String = when (l.status) {
        HubStatus.OFF -> "Not sharing"
        HubStatus.READY -> "Ready for your laptop"
        HubStatus.SHARING -> "Sharing bonded connection with ${devices(l.clients.size)}: ${formatBytes(l.upBytes + l.downBytes)}"
        HubStatus.NO_TUNNEL -> "Waiting for tailbond to connect"
        HubStatus.NO_KEY -> "Connect to your home server once first"
        HubStatus.ERROR -> "Couldn't start sharing"
    }

    /** The line under it: what to do, or what it means. */
    fun detail(l: HubLive): String = when (l.status) {
        HubStatus.OFF -> "A Mac paired with your home server can use this phone's bonded connection (mobile data, Wi-Fi and nearby phones together) through a proxy."
        HubStatus.READY -> "On the Mac, tailbond's menu bar → Connect via phone: one click starts this phone's hotspot and bonds through it. ${oneClick(l)} Without that: turn on Hotspot on this phone (Settings → Network → Hotspot & tethering) or USB tethering, join it on the Mac, then tailbond → Use a phone's connection → Use."
        HubStatus.SHARING -> "${formatBytes(l.upBytes)} up, ${formatBytes(l.downBytes)} down. Browsers and most apps on the laptop go through this phone's bonded connection. ${oneClick(l)}"
        HubStatus.NO_TUNNEL -> "Turn tailbond on. Nothing from the laptop goes out until the tunnel is up."
        HubStatus.NO_KEY -> "Turn tailbond on once so this phone can learn your home server's sharing key."
        HubStatus.ERROR -> "Port ${ProxyServer.DEFAULT_PORT} is in use by another app. ${l.error}".trim()
    }

    /** What the Mac's one click can reach on this phone (docs/phone-hub.md, "One click from the Mac"). */
    private fun oneClick(l: HubLive): String {
        val hotspot = when {
            l.hotspot.shizuku.isNotEmpty() -> l.hotspot.shizuku
            l.hotspot.state == "on" -> "the hotspot is on (${l.hotspot.ssid})"
            else -> "the hotspot starts on the Mac's click (through Shizuku)"
        }
        val ble = when (l.ble) {
            "on" -> "Bluetooth is listening for it"
            "denied" -> "Bluetooth permission is off, so the Mac must join the hotspot by hand"
            "unsupported" -> "this phone has no Bluetooth LE, so the Mac must join the hotspot by hand"
            else -> "Bluetooth is off, so the Mac must join the hotspot by hand"
        }
        return "$hotspot; $ble."
    }
}
