package dev.tailbond.app.ble

/** What the Share screen says about "One click from your Mac". */
object MacLinkWords {
    fun headline(l: MacLinkLive): String = when (l.status) {
        MacLinkStatus.OFF -> "Off"
        MacLinkStatus.WAITING -> "Waiting for your Mac over Bluetooth"
        MacLinkStatus.CONNECTED -> if (l.macs.isEmpty()) "A Mac is connecting over Bluetooth" else "Connected to ${l.macs.joinToString(" and ")} over Bluetooth"
        MacLinkStatus.NO_BLUETOOTH -> "Turn Bluetooth on"
        MacLinkStatus.NO_PERMISSION -> "tailbond needs the Nearby devices permission"
        MacLinkStatus.ERROR -> l.error.ifEmpty { "Bluetooth isn't working" }
    }

    fun shizuku(s: ShizukuStatus, sdk: Int): String = when {
        sdk < 30 -> "The Mac can't start the hotspot on this Android version (11 or newer is needed): turn it on yourself."
        s == ShizukuStatus.OK -> "Shizuku is running: your Mac can turn the hotspot on."
        s == ShizukuStatus.NO_PERMISSION -> "Shizuku is running, but tailbond may not use it yet."
        else -> "Shizuku isn't running, so your Mac can't turn the hotspot on. Install Shizuku and start it (wireless debugging, no root)."
    }
}
