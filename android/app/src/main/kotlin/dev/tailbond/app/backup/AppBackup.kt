package dev.tailbond.app.backup

import dev.tailbond.app.failover.FailoverPolicy
import dev.tailbond.app.model.AppSettings
import dev.tailbond.app.model.ServerList
import dev.tailbond.app.model.ServerProfile
import kotlinx.serialization.Serializable
import kotlinx.serialization.json.Json

/** The Android app's own configuration: its servers (pairings, with their keys) and every setting. */
@Serializable
data class AndroidConfig(val servers: List<ServerProfile> = emptyList(), val settings: AppSettings = AppSettings())

/**
 * The backup plaintext on Android: `{"format":1,"config":{"servers":[…],"settings":{…}}}`,
 * the same outer shape as the agent's (`format`, `config`), with the app's
 * own JSON as `config`. Excluded apps and websites are part of `settings`.
 */
@Serializable
data class AndroidBackup(val format: Int = 1, val config: AndroidConfig = AndroidConfig()) {
    fun toJson(): String = json.encodeToString(serializer(), this)

    companion object {
        val json = Json { ignoreUnknownKeys = true; encodeDefaults = true }

        fun fromJson(s: String): AndroidBackup = try {
            json.decodeFromString(serializer(), s)
        } catch (_: Exception) {
            throw BackupException("This backup's contents can't be read", BackupException.Kind.WRONG_PASSPHRASE_OR_DAMAGED)
        }
    }
}

/** One line of the restore preview: [what] changes from [from] to [to]. */
data class Change(val what: String, val from: String, val to: String)

/** What restoring a file would do, before anything changes. */
data class RestorePreview(
    val host: String,
    val created: String,
    val appVersion: String,
    val changes: List<Change>,
    val warnings: List<String>,
    val backup: AndroidBackup,
)

/** Pure restore rules: what may be restored here, and what it changes. */
object Restore {
    /** A tbbackup from another kind of install can't go on a phone (the contract restores onto the same role). */
    fun checkKind(h: BackupHeader) {
        if (h.role != "client" || h.platform != "android") {
            val what = when {
                h.role == "server" -> "a home server (${h.host.ifBlank { h.platform }})"
                h.platform == "openwrt" -> "a travel router (${h.host.ifBlank { "OpenWrt" }})"
                else -> "${h.platform} ${h.role}"
            }
            throw BackupException("This backup is from $what, not from the tailbond app. Restore it there", BackupException.Kind.WRONG_KIND)
        }
    }

    /**
     * Settings as restored: everything from the backup, except what belongs
     * to this phone right now (the first-run screen stays done, a
     * "handed to Tailscale" choice stays as it is).
     */
    fun merge(current: AppSettings, incoming: AppSettings): AppSettings =
        incoming.copy(onboarded = true, handedToTailscale = current.handedToTailscale)

    fun preview(
        h: BackupHeader,
        b: AndroidBackup,
        currentServers: List<ServerProfile>,
        current: AppSettings,
        isInstalled: (String) -> Boolean = { true },
    ): RestorePreview {
        checkKind(h)
        if (b.format != 1) throw BackupException("This backup was made by a newer tailbond. Update the app first", BackupException.Kind.UNSUPPORTED)
        val inc = merge(current, b.config.settings)
        val changes = mutableListOf<Change>()
        fun add(what: String, from: Any?, to: Any?) {
            val f = from?.toString() ?: ""
            val t = to?.toString() ?: ""
            if (f != t) changes += Change(what, f.ifEmpty { "none" }, t.ifEmpty { "none" })
        }

        val names = { l: List<ServerProfile> -> ServerList.joinNames(l.map { it.name }).ifEmpty { "none" } }
        add("Servers", names(currentServers), names(b.config.servers))
        b.config.servers.forEach { s ->
            val same = currentServers.firstOrNull { it.name.equals(s.name, true) }
            if (same != null && same.payload != s.payload) changes += Change("Pairing for ${s.name}", "this phone's", "the one in the backup")
        }
        add("Mode", current.mode?.label ?: "the server's choice", inc.mode?.label ?: "the server's choice")
        add("Internet traffic", current.routeMode.label, inc.routeMode.label)
        add("Apps that skip tailbond", count(current.excludedApps.size, "app"), count(inc.excludedApps.size, "app"))
        add("Websites that skip tailbond", current.websites.joinToString(", ") { it.domain }, inc.websites.joinToString(", ") { it.domain })
        add("Mobile data limit", cap(current.cellularCapMb), cap(inc.cellularCapMb))
        add("Server choice", policy(current, currentServers), policy(inc, b.config.servers))
        add("Switch to a backup server after", "${current.switchAfterSec} s", "${inc.switchAfterSec} s")
        add("Go back to the main server after", "${current.failbackAfterSec} s", "${inc.failbackAfterSec} s")
        add("Wi-Fi sign-in prompts", onOff(current.captivePrompts), onOff(inc.captivePrompts))
        add("Updates", current.updatePolicy.label, inc.updatePolicy.label)
        add("Update channel", current.updateChannel.label, inc.updateChannel.label)
        val linksChanged = current.paths != inc.paths
        if (linksChanged) changes += Change("Wi-Fi / mobile data / Ethernet roles", "this phone's", "the backup's")

        val warnings = mutableListOf<String>()
        val missing = inc.excludedApps.filterNot(isInstalled)
        if (missing.isNotEmpty()) {
            warnings += "${count(missing.size, "app")} from the backup ${if (missing.size == 1) "isn't" else "aren't"} installed on this phone; " +
                "${if (missing.size == 1) "it" else "they"} will skip tailbond once installed"
        }
        if (b.config.servers.isEmpty()) warnings += "The backup has no servers: this phone would be unpaired"
        return RestorePreview(h.host, h.created, h.agentVersion, changes, warnings, b.copy(config = b.config.copy(settings = inc)))
    }

    private fun count(n: Int, what: String) = if (n == 0) "none" else if (n == 1) "1 $what" else "$n ${what}s"
    private fun cap(mb: Long) = if (mb <= 0) "no limit" else if (mb % 1000 == 0L) "${mb / 1000} GB a month" else "$mb MB a month"
    private fun onOff(b: Boolean) = if (b) "on" else "off"
    private fun policy(s: AppSettings, servers: List<ServerProfile>) = when (s.failoverPolicy) {
        FailoverPolicy.PREFER -> "the first one that works"
        FailoverPolicy.ALWAYS -> "always " + (servers.firstOrNull { it.id == s.alwaysServerId }?.name ?: "the main one")
    }
}
