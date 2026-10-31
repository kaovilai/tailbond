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
