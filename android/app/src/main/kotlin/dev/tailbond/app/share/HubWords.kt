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
        HubStatus.READY -> "1. On this phone, turn on Hotspot (Settings → Network → Hotspot & tethering), or USB tethering with a cable. Most phones stay on the hotel Wi-Fi meanwhile, so the bond keeps both. 2. On the Mac, join the hotspot, then tailbond → Use a phone's connection → Use."
        HubStatus.SHARING -> "${formatBytes(l.upBytes)} up, ${formatBytes(l.downBytes)} down. Browsers and most apps on the laptop go through this phone's bonded connection."
        HubStatus.NO_TUNNEL -> "Turn tailbond on. Nothing from the laptop goes out until the tunnel is up."
        HubStatus.NO_KEY -> "Turn tailbond on once so this phone can learn your home server's sharing key."
        HubStatus.ERROR -> "Port ${ProxyServer.DEFAULT_PORT} is in use by another app. ${l.error}".trim()
    }
}
