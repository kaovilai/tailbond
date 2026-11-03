package dev.tailbond.app.share

import dev.tailbond.app.ui.formatBytes

/** What "Share my mobile data" says, in plain words (notification and screen). */
object ShareWords {
    /** "slate7", "slate7 and laptop", "slate7, laptop and 1 more" */
    fun hubNames(hubs: List<ShareRelay.HubUse>): String {
        val n = hubs.map { it.hub }.distinct()
        return when (n.size) {
            0 -> ""
            1 -> n[0]
            2 -> "${n[0]} and ${n[1]}"
            else -> "${n[0]}, ${n[1]} and ${n.size - 2} more"
        }
    }

    /** The headline: what is happening now. */
    fun headline(l: ShareLive): String = when (l.status) {
        ShareStatus.OFF -> "Not sharing"
        ShareStatus.STARTING -> "Starting…"
        ShareStatus.WAITING -> "Ready to share"
        ShareStatus.SHARING -> "Sharing mobile data with ${hubNames(l.hubs)}: ${formatBytes(l.hubs.sumOf { it.upBytes + it.downBytes })}"
        ShareStatus.NO_CELLULAR -> "Paused: no mobile data"
        ShareStatus.NO_WIFI -> "Waiting for Wi-Fi"
        ShareStatus.BATTERY -> "Paused: battery low" + (if (l.battery >= 0) " (${l.battery}%)" else "")
        ShareStatus.CAP -> "Stopped: this month's limit is used"
        ShareStatus.NO_KEY -> "Connect to your home server once first"
    }

    /** The line under it: what to do, or what it means. */
    fun detail(l: ShareLive, minBattery: Int): String = when (l.status) {
        ShareStatus.OFF -> "Your travel router or laptop can add this phone's mobile data to its connection."
        ShareStatus.STARTING -> "Getting mobile data ready next to Wi-Fi."
        ShareStatus.WAITING -> "Your travel router or laptop on this Wi-Fi will start using it by itself."
        ShareStatus.SHARING -> "${formatBytes(l.usedBytes)} shared this month."
        ShareStatus.NO_CELLULAR -> "Turn on mobile data, or wait for signal. Sharing goes on by itself."
        ShareStatus.NO_WIFI -> "Join the same Wi-Fi as your travel router or laptop."
        ShareStatus.BATTERY -> "Below $minBattery% and not charging. Plug in, and sharing goes on by itself."
        ShareStatus.CAP -> "Raise the limit below, or wait for next month."
        ShareStatus.NO_KEY -> "Turn tailbond on once so this phone can learn your home server's sharing key."
    }

    /** The notification while the share service runs. */
    fun notification(l: ShareLive, minBattery: Int): Pair<String, String> = when (l.status) {
        ShareStatus.SHARING -> headline(l) to "Tap to see who's using it, or stop sharing"
        else -> headline(l) to detail(l, minBattery)
    }
}
