package dev.tailbond.app.data

import android.content.Context
import android.content.SharedPreferences
import androidx.core.content.edit
import dev.tailbond.app.model.AppSettings
import dev.tailbond.app.model.ServerList
import dev.tailbond.app.model.ServerProfile
import dev.tailbond.app.model.SwitchLog
import dev.tailbond.app.model.SwitchRecord
import dev.tailbond.app.pairing.PairingPayload
import kotlinx.coroutines.flow.MutableStateFlow
import kotlinx.coroutines.flow.StateFlow
import kotlinx.coroutines.flow.asStateFlow
import kotlinx.serialization.json.Json
import kotlinx.serialization.json.JsonObject
import kotlinx.serialization.json.JsonPrimitive
import kotlinx.serialization.json.jsonObject
import kotlinx.serialization.json.jsonPrimitive
import java.time.YearMonth

/**
 * Process-wide store for the pairing (encrypted), user settings and the
 * cellular data counter. Shared by the UI, the VPN service and the tile.
 */
class Repository private constructor(ctx: Context) {
    private val prefs: SharedPreferences = ctx.getSharedPreferences("tailbond", Context.MODE_PRIVATE)

    // declared before _cellularBytes: its initializer reads it
    private var cellMonth = YearMonth.now().toString()

    private val _servers = MutableStateFlow(loadServers())
    /** The ordered server list; the first is the primary. */
    val servers: StateFlow<List<ServerProfile>> = _servers.asStateFlow()

    private val _profile = MutableStateFlow(_servers.value.firstOrNull()?.payload)
    /** The primary server's pairing (null = not paired). */
    val profile: StateFlow<PairingPayload?> = _profile.asStateFlow()

    private val _settings = MutableStateFlow(AppSettings.fromJson(prefs.getString(KEY_SETTINGS, null) ?: "{}"))
    val settings: StateFlow<AppSettings> = _settings.asStateFlow()

    private val _cellularBytes = MutableStateFlow(loadCellularBytes())
    /** Cellular bytes (tx+rx through tunnel paths) this calendar month. */
    val cellularBytes: StateFlow<Long> = _cellularBytes.asStateFlow()

    private val _peers = MutableStateFlow(prefs.getString(KEY_PEERS, null)?.let { PeerList.fromJson(it) })
    /** Last machine list from the home server (kept so it shows while disconnected). */
    val peers: StateFlow<PeerList?> = _peers.asStateFlow()

    private val _switches = MutableStateFlow(SwitchLog.fromJson(prefs.getString(KEY_SWITCHES, null)))
    /** The last server switches, newest first ("Recent switches"). */
    val switches: StateFlow<List<SwitchRecord>> = _switches.asStateFlow()

    @Synchronized
    fun addSwitch(r: SwitchRecord) {
        val next = SwitchLog.add(_switches.value, r)
        prefs.edit { putString(KEY_SWITCHES, SwitchLog.toJson(next)) }
        _switches.value = next
    }

    fun savePeers(p: PeerList) {
        prefs.edit { putString(KEY_PEERS, p.toJson()) }
        _peers.value = p
    }

    /** The sealed list, or the one-server app's sealed profile migrated to a one-element list. */
    private fun loadServers(): List<ServerProfile> {
        val list = prefs.getString(KEY_SERVERS, null)?.let { SecureBox.open(it) }
        val legacy = if (list == null) prefs.getString(KEY_PROFILE, null)?.let { SecureBox.open(it) } else null
        val out = ServerList.migrate(list, legacy, ::newId)
        if (list == null && out.isNotEmpty()) {
            prefs.edit { putString(KEY_SERVERS, SecureBox.seal(ServerList.toJson(out))); remove(KEY_PROFILE) }
        }
        return out
    }

    @Synchronized
    fun saveServers(list: List<ServerProfile>) {
        // share keys fetched for pairings that are gone go with them
        val keys = shareKeys().filterKeys { id -> list.any { pairingId(it.payload) == id } }
        prefs.edit {
            if (list.isEmpty()) remove(KEY_SERVERS) else putString(KEY_SERVERS, SecureBox.seal(ServerList.toJson(list)))
            remove(KEY_PROFILE)
            if (list.isEmpty()) remove(KEY_PEERS)
            putString(KEY_SHARE_KEYS, JsonObject(keys.mapValues { JsonPrimitive(it.value) }).toString())
            remove(KEY_SHARE_KEY) // the single key of earlier builds: whose was it?
        }
        _servers.value = list
        _profile.value = list.firstOrNull()?.payload
        if (list.isEmpty()) _peers.value = null
        // an "always use" server that was removed falls back to the primary
        val always = _settings.value.alwaysServerId
        if (always != null && list.none { it.id == always }) updateSettings { it.copy(alwaysServerId = null) }
    }

    /** First pairing, or re-pairing [replacing] (default: the primary). */
    @Synchronized
    fun saveProfile(p: PairingPayload, replacing: String? = null) {
        val list = _servers.value
        val id = replacing ?: list.firstOrNull()?.id
        val i = list.indexOfFirst { it.id == id }
        saveServers(if (i < 0) list + ServerProfile(newId(), p) else list.toMutableList().also { it[i] = ServerProfile(it[i].id, p) })
    }

    /** Appends [p] as a backup; the caller checked it with [ServerList.check]. */
    @Synchronized
    fun addServer(p: PairingPayload): ServerProfile {
        val entry = ServerProfile(newId(), p)
        saveServers(_servers.value + entry)
        return entry
    }

    @Synchronized
    fun removeServer(id: String) = saveServers(_servers.value.filter { it.id != id })

    @Synchronized
    fun moveServer(id: String, delta: Int) = saveServers(ServerList.move(_servers.value, id, delta))

    fun clearProfile() = saveServers(emptyList())

    private fun newId(): String = java.util.UUID.randomUUID().toString().take(8)

    fun updateSettings(f: (AppSettings) -> AppSettings) {
        val next = f(_settings.value)
        prefs.edit { putString(KEY_SETTINGS, next.toJson()) }
        _settings.value = next
    }

    /** Restore (or undo one): the server list and every setting at once. The VPN service picks the change up. */
    @Synchronized
    fun replaceAll(list: List<ServerProfile>, settings: AppSettings) {
        prefs.edit { putString(KEY_SETTINGS, settings.toJson()) }
        _settings.value = settings
        saveServers(list)
        if (list.isEmpty()) updateSettings { it.copy(onboarded = true) }
    }

    private fun monthKey() = YearMonth.now().toString()

    private fun loadCellularBytes(): Long =
        if (prefs.getString(KEY_CELL_MONTH, null) == cellMonth) prefs.getLong(KEY_CELL_BYTES, 0) else 0

    /** Adds [delta] bytes to this month's cellular counter (starts over in a new month). */
    @Synchronized
    fun addCellularBytes(delta: Long, persist: Boolean) {
        val month = monthKey()
        if (month != cellMonth) {
            cellMonth = month
            _cellularBytes.value = 0
        }
        if (delta > 0) _cellularBytes.value += delta
        if (persist) {
            prefs.edit { putString(KEY_CELL_MONTH, cellMonth); putLong(KEY_CELL_BYTES, _cellularBytes.value) }
        }
    }

    @Synchronized
    fun resetCellularBytes() {
        cellMonth = monthKey()
        _cellularBytes.value = 0
        prefs.edit { putString(KEY_CELL_MONTH, cellMonth); putLong(KEY_CELL_BYTES, 0) }
    }

    // --- Pair & Share ---

    /**
     * This phone's share key for pairing [p]: the pairing's, else the one
     * fetched from that pairing's server (pairings made before they carried
     * it). Each pairing has its own.
     */
    @Synchronized
    fun shareKey(p: PairingPayload?): String? {
        if (p == null) return null
        return p.shareKey?.takeIf { it.isNotBlank() } ?: shareKeys()[pairingId(p)]?.takeIf { it.isNotBlank() }
    }

    @Synchronized
    fun saveShareKey(p: PairingPayload, k: String) {
        val keys = shareKeys() + (pairingId(p) to k)
        prefs.edit { putString(KEY_SHARE_KEYS, JsonObject(keys.mapValues { JsonPrimitive(it.value) }).toString()) }
    }

    private fun shareKeys(): Map<String, String> = try {
        prefs.getString(KEY_SHARE_KEYS, null)?.let { s ->
            Json.parseToJsonElement(s).jsonObject.mapValues { it.value.jsonPrimitive.content }
        }.orEmpty()
    } catch (_: Exception) {
        emptyMap()
    }

    /** Mobile data shared with hubs this month (bytes). */
    @Synchronized
    fun sharedBytes(): Long = if (prefs.getString(KEY_SHARE_MONTH, null) == monthKey()) prefs.getLong(KEY_SHARE_BYTES, 0) else 0

    /** [v] bytes shared in [month] (yyyy-MM): the month the relay counted them in, not today's. */
    @Synchronized
    fun setSharedBytes(month: String, v: Long) = prefs.edit { putString(KEY_SHARE_MONTH, month); putLong(KEY_SHARE_BYTES, v) }

    /** This month, as the share counter names it (yyyy-MM). */
    fun shareMonth(): String = monthKey()

    /** A stable id for this phone as a sharer. */
    fun shareId(): String = prefs.getString(KEY_SHARE_ID, null) ?: ("an-" + java.util.UUID.randomUUID().toString().take(10)).also {
        prefs.edit { putString(KEY_SHARE_ID, it) }
    }

    fun cellularCapReached(): Boolean {
        val cap = _settings.value.cellularCapMb
        return cap > 0 && _cellularBytes.value >= cap * 1_000_000
    }

    companion object {
        /** One-server app's pairing; migrated to [KEY_SERVERS] on first load. */
        private const val KEY_PROFILE = "profile_sealed"
        private const val KEY_SERVERS = "servers_sealed"
        private const val KEY_SWITCHES = "switches_json"
        private const val KEY_SETTINGS = "settings_json"
        private const val KEY_CELL_MONTH = "cell_month"
        private const val KEY_CELL_BYTES = "cell_bytes"
        private const val KEY_PEERS = "peers_json"
        private const val KEY_SHARE_KEY = "share_key"
        private const val KEY_SHARE_KEYS = "share_keys"
        private const val KEY_SHARE_MONTH = "share_month"
        private const val KEY_SHARE_BYTES = "share_bytes"
        private const val KEY_SHARE_ID = "share_id"

        @Volatile private var instance: Repository? = null

        /** Names a pairing in the share key cache (not by its key). */
        fun pairingId(p: PairingPayload): String {
            val d = java.security.MessageDigest.getInstance("SHA-256")
                .digest("tbs1-pairing\u0000${p.server}\u0000${p.user}\u0000${p.key}".toByteArray())
            return d.copyOf(8).joinToString("") { "%02x".format(it) }
        }

        fun get(ctx: Context): Repository =
            instance ?: synchronized(this) {
                instance ?: Repository(ctx.applicationContext).also { instance = it }
            }
    }
}
