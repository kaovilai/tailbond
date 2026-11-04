package dev.tailbond.app.update

import java.time.Instant
import java.time.LocalDate
import java.time.ZoneId
import java.time.format.DateTimeFormatter
import java.util.Locale

/** What the Updates page says, in the app's plain words. Pure; unit tested. */
object UpdateWords {
    /** The big line. */
    fun headline(s: UpdateStatus, policy: UpdatePolicy): String {
        val a = s.availability
        return when {
            s.checking -> "Checking for updates…"
            a == Availability.AVAILABLE -> "Update available: v${s.latest}"
            s.error != null -> "Couldn't check for updates"
            a == Availability.DEV_BUILD -> "Latest release: v${s.latest}"
            a == Availability.UP_TO_DATE || a == Availability.NO_APK -> "Up to date"
            policy == UpdatePolicy.OFF -> "Updates are off"
            else -> "Not checked yet"
        }
    }

    /** The line under it; null when the headline says it all. */
    fun detail(s: UpdateStatus, policy: UpdatePolicy): String? {
        val a = s.availability
        return when {
            s.checking -> null
            a == Availability.AVAILABLE -> "You have ${s.current}."
            // calm on purpose: a failed check isn't tailbond failing
            s.error != null -> s.error.trimEnd('.') + ". tailbond works as before; it tries again later by itself, or tap Check now."
            a == Availability.DEV_BUILD ->
                "This build (${s.current.ifBlank { "no version" }}) isn't a numbered release, so tailbond doesn't tell you to update it."
            a == Availability.NO_APK -> "Version ${s.latest} is out, but it has no Android app yet."
            a == Availability.UNKNOWN && policy == UpdatePolicy.OFF -> "tailbond doesn't check by itself. Check now still works."
            else -> null
        }
    }

    /** "Checked today at 14:02"; after a failed try also when the last good check was. */
    fun lastChecked(s: UpdateStatus, nowMs: Long, zone: ZoneId = ZoneId.systemDefault()): String = when {
        s.error != null && s.lastAttemptMs > 0 && s.lastCheckMs > 0 ->
            "Tried ${whenText(s.lastAttemptMs, nowMs, zone)}. Last good check ${whenText(s.lastCheckMs, nowMs, zone)}"
        s.error != null && s.lastAttemptMs > 0 -> "Tried ${whenText(s.lastAttemptMs, nowMs, zone)}"
        s.lastCheckMs > 0 -> "Checked ${whenText(s.lastCheckMs, nowMs, zone)}"
        else -> "Not checked yet"
    }

    /** The Settings row's second line. */
    fun row(s: UpdateStatus, policy: UpdatePolicy, nowMs: Long, zone: ZoneId = ZoneId.systemDefault()): String = when {
        s.availability == Availability.AVAILABLE -> "Update available: v${s.latest}"
        policy == UpdatePolicy.OFF -> "Automatic checks are off"
        s.error != null -> "Couldn't check for updates"
        s.lastCheckMs > 0 -> headline(s, policy) + " · checked " + whenText(s.lastCheckMs, nowMs, zone)
        else -> headline(s, policy)
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

    /** Under "Download update": what happens. */
    fun howToInstall(version: String?): String =
        "Your browser downloads tailbond-${version ?: "<version>"}.apk. Open it and tap Update. Your servers and settings stay."

    /** And the one prompt that may look scary. */
    const val ALLOW_PROMPT = "The first time, Android asks to allow your browser to install apps. That's normal. " +
        "tailbond reconnects by itself after the update."

    /** A debug build can't take the release APKs; how to switch. */
    const val DEBUG_BUILD = "This is a debug build. It's signed with a different key than the release APKs on GitHub, " +
        "so Android won't install a release over it. Back up first (Settings → Backup) so you keep your settings. " +
        "Then uninstall tailbond, install the release APK, and restore the backup."

    const val OBTAINIUM_TITLE = "Hands-off: use Obtainium"

    fun obtainium(repo: String): String =
        "Android doesn't let an app install its own updates without your tap, so tailbond can only tell you (Right away: within the hour). " +
            "To have them installed for you, install Obtainium, a free app that follows tailbond's GitHub releases (github.com/$repo) " +
            "and installs each new version. Add tailbond in it by that address, and set Obtainium to check every hour " +
            "(Obtainium → Settings → Background update checking interval → 1 hour)."

    const val NOTIFY_TITLE = "Update ready: tap to install"
    fun notifyText(version: String) = "tailbond $version is out. Tap to download it; Android then asks you to install it."
}
