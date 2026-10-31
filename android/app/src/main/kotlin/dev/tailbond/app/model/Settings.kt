package dev.tailbond.app.model

import com.mqvpn.sdk.network.PathType
import kotlinx.serialization.Serializable
import kotlinx.serialization.json.Json

/** Link kinds the dashboard shows a card for. */
@Serializable
enum class PathKind(val label: String) {
    WIFI("Wi-Fi"), CELLULAR("Cellular"), ETHERNET("Ethernet");

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

@Serializable
enum class Priority(val label: String) { PRIMARY("Primary"), SECONDARY("Secondary"), BACKUP("Backup") }

@Serializable
data class PathSetting(val enabled: Boolean = true, val priority: Priority = Priority.PRIMARY)

@Serializable
enum class RouteMode(val label: String) {
    FULL("Full tunnel"),
    TAILNET_LAN("Tailnet + home LAN only"),
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
) {
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
