package dev.tailbond.app.update

import java.time.Instant
import java.time.LocalDate
import java.time.ZoneId
import java.time.format.DateTimeFormatter
import java.util.Locale

/** What the Updates page says, in the app's plain words. Pure; unit tested. */
object UpdateWords {
    /** The big line: where things are. */
    fun headline(s: UpdateStatus): String {
        val v = s.installing ?: s.available
        return when (s.phase) {
            Phase.OFF -> if (s.noKeys) "This build doesn't update itself" else "Updates are off"
            Phase.CHECKING -> "Checking for updates…"
            Phase.DOWNLOADING -> "Downloading version $v…"
            Phase.INSTALLING -> "Installing version $v…"
            Phase.NEEDS_TAP -> "One tap to finish updating"
            Phase.NEEDS_PERMISSION -> "Version $v is ready to install"
            Phase.FAILED -> if (s.lastResult.startsWith("Couldn't install ")) s.lastResult.substringBefore(':') else "The last update didn't work"
            Phase.WAITING -> if (s.needsServer != null) "Version $v waits for your server" else "Version $v installs when the phone is quiet"
            Phase.IDLE -> if (s.available != null) "Version ${s.available} is available" else "Up to date"
        }
    }

    /** The line under it (why, what next); null when the headline says it all. */
    fun detail(s: UpdateStatus, server: String?): String? = when (s.phase) {
        Phase.OFF -> if (s.noKeys) {
            "It was built without tailbond's update key. Install the release APK once to get updates, or update it the way you installed it"
        } else {
            "tailbond doesn't check by itself. Check now still works"
        }
        Phase.DOWNLOADING -> s.progress?.let { "${(it * 100).toInt()} %" + (s.sourceLabel?.let { l -> " from $l" } ?: "") }
        Phase.INSTALLING -> "tailbond restarts in a moment. The connection drops for a few seconds and comes back by itself"
        Phase.NEEDS_TAP -> "Android asks “Do you want to update this app?” once, because another app installed tailbond. " +
            "Tap Update. After that, tailbond updates by itself"
        Phase.NEEDS_PERMISSION -> "Android needs your OK before tailbond can install updates: tap Allow, then turn on “Allow from this source”"
        Phase.FAILED -> s.lastResult.substringAfter(": ", "").ifEmpty { s.lastResult }.replaceFirstChar { it.uppercase() }
            .let { "$it. tailbond is still on version ${s.current}" }
        Phase.WAITING -> if (s.needsServer != null) {
            "It needs a newer ${server ?: "home server"}. The phone updates after the server does"
        } else {
            "Something is using tailbond right now (a call or a video). It installs when that's done" +
                if (s.security) "" else ", or tap Install now"
        }
        Phase.IDLE -> when {
            s.available != null && s.security -> "A security fix"
            s.available != null -> null
            !s.lastResultOk && s.lastResult.isNotBlank() -> s.lastResult
            else -> null
        }
        Phase.CHECKING -> null
    }

    /** "Checked today at 14:02 · from homebox". */
    fun lastChecked(s: UpdateStatus, nowMs: Long, zone: ZoneId = ZoneId.systemDefault()): String {
        if (s.lastCheckMs <= 0) return "Not checked yet"
        val from = when (s.source) {
            UpdateSource.MIRROR -> " · from ${s.sourceLabel ?: "your server"}"
            UpdateSource.PUBLIC -> " · from ${s.sourceLabel ?: "the internet"}"
            else -> ""
        }
        return "Checked ${whenText(s.lastCheckMs, nowMs, zone)}$from"
    }

    /** "today at 14:02", "yesterday at 09:15", "on 2 Nov at 10:00". */
    fun whenText(ms: Long, nowMs: Long, zone: ZoneId = ZoneId.systemDefault()): String {
        val t = Instant.ofEpochMilli(ms).atZone(zone)
        val today = Instant.ofEpochMilli(nowMs).atZone(zone).toLocalDate()
        val clock = t.format(DateTimeFormatter.ofPattern("HH:mm", Locale.US))
        val day: LocalDate = t.toLocalDate()
        return when (day) {
            today -> "today at $clock"
            today.minusDays(1) -> "yesterday at $clock"
            else -> "on ${t.format(DateTimeFormatter.ofPattern("d MMM", Locale.US))} at $clock"
        }
    }

    /** The first-install explanation, shown until tailbond can update itself without a tap. */
    fun firstTap(silent: Boolean, sdkInt: Int): String? = when {
        sdkInt < 31 -> "On this Android version, Android asks for one tap (“Update”) every time tailbond updates."
        silent -> null
        else -> "The first update needs one tap: Android asks “Do you want to update this app?” because your browser installed tailbond. " +
            "After that, tailbond installs updates by itself."
    }
}
