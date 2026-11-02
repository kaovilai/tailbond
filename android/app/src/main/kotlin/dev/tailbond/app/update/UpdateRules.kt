package dev.tailbond.app.update

import kotlinx.serialization.SerialName
import kotlinx.serialization.Serializable

/** `update.policy`: [label] and [explain] are what Settings shows. */
@Serializable
enum class UpdatePolicy(val wire: String, val label: String, val explain: String) {
    @SerialName("auto") AUTOMATIC("auto", "Automatic", "Installs updates by itself, but not during a call or a video"),
    @SerialName("notify") NOTIFY("notify", "Tell me first", "Checks and tells you; you tap Install"),
    @SerialName("off") OFF("off", "Off", "Never checks. Check now still works"),
}

/** `update.channel`. */
@Serializable
enum class UpdateChannel(val wire: String, val label: String, val explain: String) {
    @SerialName("stable") STABLE("stable", "Stable", "Tested releases"),
    @SerialName("beta") BETA("beta", "Beta", "New versions a bit earlier, sometimes with rough edges"),
}

/** What to do with a verified manifest. */
sealed interface Decision {
    /** This build is that version or newer. */
    data object UpToDate : Decision
    /** The release has no Android APK. */
    data object NoApk : Decision
    /**
     * A newer version needs a newer server ([needs] > [has]; [has] null =
     * the server couldn't be asked): the phone waits until its server updates.
     */
    data class NeedsServer(val version: String, val needs: Int, val has: Int?) : Decision
    /**
     * [version] is available. [install]: install now without asking.
     * [deferred]: why an automatic install waits (null = it doesn't).
     */
    data class Available(val version: String, val install: Boolean, val deferred: Deferral? = null) : Decision
}

enum class Deferral {
    /** The tunnel is busy (a call, a stream). */
    BUSY,
    /** Policy "Tell me first": waits for a tap. */
    ASK,
    /** A development build never installs anything by itself. */
    DEV_BUILD,
}

/** Pure update rules (docs/architecture.md, "Updates"); unit tested. */
object UpdateRules {
    const val CHECK_EVERY_MS = 6 * 60 * 60 * 1000L
    /** `defer_kbps` default: tx+rx through the tunnel, averaged over the last minute. */
    const val DEFER_KBPS = 300L

    /**
     * [current]: this build's version name; [currentCode]: its versionCode.
     * [serverProtocol]: the active server's `protocol` from GET /api/hello
     * (missing field = 1; null = couldn't ask). [busyKbps]: tunnel traffic
     * over the last minute.
     */
    fun decide(
        current: String,
        currentCode: Long,
        m: Manifest,
        serverProtocol: Int?,
        policy: UpdatePolicy,
        busyKbps: Long,
        deferKbps: Long = DEFER_KBPS,
    ): Decision {
        val apk = m.apk ?: return Decision.NoApk
        val latest = SemVer.parse(m.version) ?: return Decision.UpToDate
        val own = SemVer.own(current)
        val newerCode = apk.versionCode?.let { it > currentCode }
        when {
            own != null && latest <= own -> return Decision.UpToDate
            // Android refuses a versionCode that isn't higher, whatever the name says
            newerCode == false -> return Decision.UpToDate
        }
        val needs = m.requiresServerProtocol
        // every server speaks protocol 1, so only a higher requirement needs an answer from it
        if (needs > 1 && (serverProtocol ?: 0) < needs) return Decision.NeedsServer(m.version, needs, serverProtocol)
        return when {
            own == null -> Decision.Available(m.version, install = false, deferred = Deferral.DEV_BUILD)
            policy != UpdatePolicy.AUTOMATIC -> Decision.Available(m.version, install = false, deferred = Deferral.ASK)
            !m.security && busyKbps > deferKbps -> Decision.Available(m.version, install = false, deferred = Deferral.BUSY)
            else -> Decision.Available(m.version, install = true)
        }
    }

    /** Whether a scheduled check is due ([lastCheckMs] 0 = never checked). */
    fun checkDue(lastCheckMs: Long, nowMs: Long, everyMs: Long = CHECK_EVERY_MS): Boolean =
        lastCheckMs <= 0 || nowMs - lastCheckMs >= everyMs || nowMs < lastCheckMs

    /**
     * Average tunnel traffic (tx+rx, kbit/s) over the last minute from the
     * per-second rate history (LiveState keeps 60 samples, bits/s).
     */
    fun busyKbps(rxHistory: List<Long>, txHistory: List<Long>, samples: Int = 60): Long {
        val rx = rxHistory.takeLast(samples)
        val tx = txHistory.takeLast(samples)
        val n = maxOf(rx.size, tx.size)
        if (n == 0) return 0
        return (rx.sum() + tx.sum()) / n / 1000
    }
}
