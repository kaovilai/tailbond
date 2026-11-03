package dev.tailbond.app.model

import com.mqvpn.sdk.network.PathType
import dev.tailbond.app.failover.FailoverPolicy
import dev.tailbond.app.failover.FailoverTiming
import dev.tailbond.app.update.UpdatePolicy
import kotlinx.serialization.Serializable
import kotlinx.serialization.json.Json

/** Link kinds the Connections section shows a card for. [phrase] is the mid-sentence form. */
@Serializable
enum class PathKind(val label: String, val phrase: String) {
    WIFI("Wi-Fi", "Wi-Fi"), CELLULAR("Mobile data", "mobile data"), ETHERNET("Ethernet", "Ethernet");

    companion object {
        fun of(type: PathType): PathKind? = when (type) {
            PathType.WIFI -> WIFI
            PathType.CELLULAR -> CELLULAR
            PathType.ETHERNET -> ETHERNET
            else -> null
        }

        /** sdk-network names paths "<transport>-<id>" (NetworkMonitor.networkName). */
        fun ofPathName(name: String): PathKind? = when (name.substringBefore('-').lowercase()) {
            "wifi" -> WIFI
            "cellular" -> CELLULAR
            "ethernet" -> ETHERNET
            else -> null
        }
    }
}

/** Contract names primary/secondary/backup; [label] and [explain] are what the app shows. */
@Serializable
enum class Priority(val label: String, val explain: String) {
    PRIMARY("Main", "Always carries traffic"),
    SECONDARY("Extra", "Adds speed. In \"Save mobile data\" it waits as a backup"),
    BACKUP("Backup only", "Used only when the others stop working"),
}

@Serializable
data class PathSetting(val enabled: Boolean = true, val priority: Priority = Priority.PRIMARY)

@Serializable
enum class RouteMode(val label: String, val explain: String) {
    FULL("Through your home server", "All internet traffic leaves from home, like using an exit node"),
    TAILNET_LAN(
        "Only Tailscale machines",
        "Websites and apps use this phone's own connection; only your Tailscale machines (100.x addresses) go through home. " +
            "Names are still looked up through home, so MagicDNS names work. LANs behind other subnet routers aren't reachable through tailbond.",
    ),
}

/** Everything the user can change besides the pairing itself. Persisted as JSON. */
@Serializable
data class AppSettings(
    /** null = the mode from the pairing payload. */
    val mode: Mode? = null,
    val paths: Map<PathKind, PathSetting> = DEFAULT_PATHS,
    val routeMode: RouteMode = RouteMode.FULL,
    /** Package names routed outside the tunnel (VpnService addDisallowedApplication). */
    val excludedApps: Set<String> = emptySet(),
    /** Monthly cellular cap in MB; 0 = no cap. At the cap cellular stops carrying paths. */
    val cellularCapMb: Long = 0,
    /** Failover only: mqvpn's experimental backup_fec scheduler instead of minrtt. */
    val experimentalBackupFec: Boolean = false,
    /** First-launch explainer done (tailbond replaces the Tailscale VPN on this phone). */
    val onboarded: Boolean = false,
    /** The user handed the VPN back to Tailscale; nothing reconnects tailbond until they turn it on again. */
    val handedToTailscale: Boolean = false,
    /** Tell the user when Wi-Fi needs a sign-in (hotel/airport portals). */
    val captivePrompts: Boolean = true,
    /** Server failover (`client.failover`): prefer the first server, or always use [alwaysServerId]. */
    val failoverPolicy: FailoverPolicy = FailoverPolicy.PREFER,
    val alwaysServerId: String? = null,
    /** Switch to the next server after this long without a working tunnel. */
    val switchAfterSec: Int = 15,
    /** Go back to an earlier server once it has answered every probe this long. */
    val failbackAfterSec: Int = 120,
    /** Websites that skip the tunnel (Android 13+, excludeRoute). */
    val websites: List<WebsiteRule> = emptyList(),
    /** `update.policy` (docs/architecture.md, "Updates"): tell me (check once a day) or off. */
    val updatePolicy: UpdatePolicy = UpdatePolicy.NOTIFY,
) {
    val failoverTiming: FailoverTiming
        get() = FailoverTiming(
            switchAfterMs = switchAfterSec.coerceIn(5, 600) * 1000L,
            failbackAfterMs = failbackAfterSec.coerceIn(10, 3600) * 1000L,
        )

    /** Domains of the websites that are switched on. */
    val activeWebsites: Set<String> get() = websites.filter { it.enabled }.map { it.domain }.toSet()

    fun path(kind: PathKind): PathSetting = paths[kind] ?: DEFAULT_PATHS.getValue(kind)

    fun withPath(kind: PathKind, s: PathSetting) = copy(paths = paths + (kind to s))

    fun toJson(): String = json.encodeToString(serializer(), this)

    companion object {
        val DEFAULT_PATHS = mapOf(
            PathKind.WIFI to PathSetting(true, Priority.PRIMARY),
            PathKind.ETHERNET to PathSetting(true, Priority.PRIMARY),
            PathKind.CELLULAR to PathSetting(true, Priority.SECONDARY),
        )
        private val json = Json { ignoreUnknownKeys = true; encodeDefaults = true }

        fun fromJson(s: String): AppSettings = try {
            json.decodeFromString(serializer(), s)
        } catch (_: Exception) {
            AppSettings()
        }
    }
}
