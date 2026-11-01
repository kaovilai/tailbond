package dev.tailbond.app.data

import android.content.Context
import android.content.SharedPreferences
import androidx.core.content.edit
import dev.tailbond.app.model.AppSettings
import dev.tailbond.app.model.ServerList
import dev.tailbond.app.model.ServerProfile
import dev.tailbond.app.pairing.PairingPayload
import kotlinx.coroutines.flow.MutableStateFlow
import kotlinx.coroutines.flow.StateFlow
import kotlinx.coroutines.flow.asStateFlow
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
        prefs.edit {
            if (list.isEmpty()) remove(KEY_SERVERS) else putString(KEY_SERVERS, SecureBox.seal(ServerList.toJson(list)))
            remove(KEY_PROFILE)
            if (list.isEmpty()) remove(KEY_PEERS)
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

    fun cellularCapReached(): Boolean {
        val cap = _settings.value.cellularCapMb
        return cap > 0 && _cellularBytes.value >= cap * 1_000_000
    }

    companion object {
        /** One-server app's pairing; migrated to [KEY_SERVERS] on first load. */
        private const val KEY_PROFILE = "profile_sealed"
        private const val KEY_SERVERS = "servers_sealed"
        private const val KEY_SETTINGS = "settings_json"
        private const val KEY_CELL_MONTH = "cell_month"
        private const val KEY_CELL_BYTES = "cell_bytes"
        private const val KEY_PEERS = "peers_json"

        @Volatile private var instance: Repository? = null

        fun get(ctx: Context): Repository =
            instance ?: synchronized(this) {
                instance ?: Repository(ctx.applicationContext).also { instance = it }
            }
    }
}
