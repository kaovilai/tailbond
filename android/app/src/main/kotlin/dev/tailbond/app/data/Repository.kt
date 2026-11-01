package dev.tailbond.app.data

import android.content.Context
import android.content.SharedPreferences
import androidx.core.content.edit
import dev.tailbond.app.model.AppSettings
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

    private val _profile = MutableStateFlow(loadProfile())
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

    private fun loadProfile(): PairingPayload? {
        val sealed = prefs.getString(KEY_PROFILE, null) ?: return null
        val json = SecureBox.open(sealed) ?: return null
        return try { PairingPayload.fromJson(json) } catch (_: Exception) { null }
    }

    fun saveProfile(p: PairingPayload) {
        prefs.edit { putString(KEY_PROFILE, SecureBox.seal(p.toJson())) }
        _profile.value = p
    }

    fun clearProfile() {
        prefs.edit { remove(KEY_PROFILE); remove(KEY_PEERS) }
        _profile.value = null
        _peers.value = null
    }

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
        private const val KEY_PROFILE = "profile_sealed"
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
