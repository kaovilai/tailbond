package dev.tailbond.app.update

import kotlinx.serialization.KSerializer
import kotlinx.serialization.Serializable
import kotlinx.serialization.descriptors.PrimitiveKind
import kotlinx.serialization.descriptors.PrimitiveSerialDescriptor
import kotlinx.serialization.encoding.Decoder
import kotlinx.serialization.encoding.Encoder

/**
 * `update.policy` on the phone: [label] and [explain] are what Settings shows.
 * Stored as `immediate` / `notify` / `off`; anything else (an older `auto`)
 * reads as [NOTIFY], so old settings and backups still load. New installs
 * get [IMMEDIATE] (AppSettings' default); saved settings keep theirs.
 *
 * Android never lets an app install itself without a tap, so "Right away"
 * is an hourly check and a notification, "Update ready: tap to install";
 * Obtainium (checking hourly) is the hands-off way.
 */
@Serializable(with = UpdatePolicySerializer::class)
enum class UpdatePolicy(val wire: String, val label: String, val explain: String) {
    IMMEDIATE(
        "immediate", "Right away",
        "Checks every hour. When a new version is out you get a notification, \"Update ready: tap to install\". " +
            "Android always asks before installing an app, so it needs your tap",
    ),
    NOTIFY("notify", "Tell me", "Checks GitHub about once a day and shows a new version here"),
    OFF("off", "Off", "Never checks by itself. Check now still works"),
}

object UpdatePolicySerializer : KSerializer<UpdatePolicy> {
    override val descriptor = PrimitiveSerialDescriptor("dev.tailbond.app.update.UpdatePolicy", PrimitiveKind.STRING)
    override fun serialize(encoder: Encoder, value: UpdatePolicy) = encoder.encodeString(value.wire)
    override fun deserialize(decoder: Decoder): UpdatePolicy =
        when (decoder.decodeString().lowercase()) {
            UpdatePolicy.OFF.wire -> UpdatePolicy.OFF
            UpdatePolicy.IMMEDIATE.wire -> UpdatePolicy.IMMEDIATE
            else -> UpdatePolicy.NOTIFY
        }
}

/** What the last good check means for this build. */
enum class Availability {
    /** Never checked successfully. */
    UNKNOWN,
    UP_TO_DATE,
    /** A newer release with an APK: "Update available". */
    AVAILABLE,
    /** A newer release without an Android APK (yet). */
    NO_APK,
    /** This build's version isn't semver (a dev build): shown, never nagged about. */
    DEV_BUILD,
}

/** Pure update rules (docs/architecture.md, "Updates"); unit tested. */
object UpdateRules {
    /** Tell me: about once a day (when the app comes to the front and the last good check is older than this). */
    const val CHECK_EVERY_MS = 24 * 60 * 60 * 1000L
    /** Right away: every hour, also from the tunnel's service while it runs. */
    const val CHECK_HOURLY_MS = 60 * 60 * 1000L
    /** After a failed check, the next automatic try waits at least this long (GitHub allows 60 requests an hour). */
    const val RETRY_MS = 60 * 60 * 1000L

    /**
     * Whether an automatic check is due. Never with policy Off. [lastCheckMs]:
     * the last good check, [lastAttemptMs]: the last try (0 = never). A clock
     * that went backwards counts as due.
     */
    fun checkDue(policy: UpdatePolicy, lastCheckMs: Long, lastAttemptMs: Long, nowMs: Long): Boolean {
        if (policy == UpdatePolicy.OFF) return false
        fun older(t: Long, gap: Long) = t <= 0 || nowMs < t || nowMs - t >= gap
        val every = if (policy == UpdatePolicy.IMMEDIATE) CHECK_HOURLY_MS else CHECK_EVERY_MS
        return older(lastCheckMs, every) && older(lastAttemptMs, RETRY_MS)
    }

    /**
     * Whether to post "Update ready: tap to install": with Right away, for
     * an installable newer release (it has an APK; this is a release build),
     * once per version ([notifiedVersion]: the last one notified about).
     */
    fun notifyUpdate(policy: UpdatePolicy, s: UpdateStatus, notifiedVersion: String?): Boolean =
        policy == UpdatePolicy.IMMEDIATE && s.availability == Availability.AVAILABLE && s.latest != notifiedVersion

    /** [current]: this build's versionName; [latest]: the latest release's version (null = never checked). */
    fun availability(current: String, latest: String?, apkUrl: String?): Availability {
        val l = latest?.let { SemVer.parse(it) } ?: return Availability.UNKNOWN
        val own = SemVer.own(current) ?: return Availability.DEV_BUILD
        return when {
            l <= own -> Availability.UP_TO_DATE
            apkUrl == null -> Availability.NO_APK
            else -> Availability.AVAILABLE
        }
    }

    /** `owner/name`, also from `https://github.com/owner/name`; null when it's neither. */
    fun repo(s: String?): String? {
        val t = s?.trim()?.removeSuffix("/")?.removeSuffix(".git")?.removePrefix("https://github.com/") ?: return null
        return t.takeIf { REPO.matches(it) }
    }

    fun latestUrl(repo: String) = "https://api.github.com/repos/$repo/releases/latest"

    private val REPO = Regex("""^[A-Za-z0-9](?:[A-Za-z0-9-]{0,38})/[A-Za-z0-9._-]{1,100}$""")
}
