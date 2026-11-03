package dev.tailbond.app.ui

import java.util.Locale

/** 12.3 Mbps style, bits per second in. */
fun formatBps(bps: Long): String = when {
    bps >= 1_000_000_000 -> String.format(Locale.US, "%.2f Gbps", bps / 1e9)
    bps >= 1_000_000 -> String.format(Locale.US, "%.1f Mbps", bps / 1e6)
    bps >= 1_000 -> String.format(Locale.US, "%.0f kbps", bps / 1e3)
    else -> "$bps bps"
}

/** Decimal units (what carriers bill in). */
fun formatBytes(b: Long): String = when {
    b >= 1_000_000_000 -> String.format(Locale.US, "%.2f GB", b / 1e9)
    b >= 1_000_000 -> String.format(Locale.US, "%.1f MB", b / 1e6)
    b >= 1_000 -> String.format(Locale.US, "%.0f kB", b / 1e3)
    else -> "$b B"
}

fun formatDuration(sec: Long): String {
    val h = sec / 3600
    val m = (sec % 3600) / 60
    val s = sec % 60
    return if (h > 0) String.format(Locale.US, "%d:%02d:%02d", h, m, s) else String.format(Locale.US, "%d:%02d", m, s)
}

/** "↓38.4 ↑4.1 Mbps": download and upload in one unit, for narrow cards. */
fun formatRatePair(rxBps: Long, txBps: Long): String {
    val peak = maxOf(rxBps, txBps)
    val (div, unit) = when {
        peak >= 1_000_000_000 -> 1e9 to "Gbps"
        peak >= 1_000_000 -> 1e6 to "Mbps"
        else -> 1e3 to "kbps"
    }
    return String.format(Locale.US, "↓%.1f ↑%.1f %s", rxBps / div, txBps / div, unit)
}

/** The tunnel notification's text: the failover note, the connections sentence, the rates. */
fun notificationText(failoverNote: String?, connections: String?, rxBps: Long, txBps: Long): String =
    listOfNotNull(failoverNote, connections, "↓ ${formatBps(rxBps)}  ↑ ${formatBps(txBps)}").joinToString(" · ")
