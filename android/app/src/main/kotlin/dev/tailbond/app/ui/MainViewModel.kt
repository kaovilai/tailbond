package dev.tailbond.app.ui

import android.app.Application
import android.content.Intent
import androidx.lifecycle.AndroidViewModel
import androidx.lifecycle.viewModelScope
import dev.tailbond.app.data.Repository
import dev.tailbond.app.model.Mode
import dev.tailbond.app.model.PathKind
import dev.tailbond.app.model.PathSetting
import dev.tailbond.app.model.RouteMode
import dev.tailbond.app.pairing.Pairing
import dev.tailbond.app.pairing.PairingPayload
import dev.tailbond.app.pairing.PairingResult
import dev.tailbond.app.service.LiveState
import dev.tailbond.app.service.TailbondVpnService
import kotlinx.coroutines.Dispatchers
import kotlinx.coroutines.flow.MutableStateFlow
import kotlinx.coroutines.flow.StateFlow
import kotlinx.coroutines.flow.asStateFlow
import kotlinx.coroutines.launch
import kotlinx.coroutines.withContext

data class AppEntry(val pkg: String, val label: String)

class MainViewModel(app: Application) : AndroidViewModel(app) {
    private val repo = Repository.get(app)

    val live = LiveState.live
    val profile = repo.profile
    val settings = repo.settings
    val cellularBytes = repo.cellularBytes

    /** A pairing waiting for the user's confirmation (scanned, pasted or deep link). */
    private val _pending = MutableStateFlow<PairingPayload?>(null)
    val pending: StateFlow<PairingPayload?> = _pending.asStateFlow()

    private val _pairError = MutableStateFlow<String?>(null)
    val pairError: StateFlow<String?> = _pairError.asStateFlow()

    private val _apps = MutableStateFlow<List<AppEntry>>(emptyList())
    val apps: StateFlow<List<AppEntry>> = _apps.asStateFlow()

    /** Returns true when [text] parsed; the payload then waits in [pending]. */
    fun offerPairing(text: String): Boolean = when (val r = Pairing.parse(text)) {
        is PairingResult.Ok -> { _pending.value = r.payload; _pairError.value = null; true }
        is PairingResult.Invalid -> { _pairError.value = r.reason; false }
    }

    fun confirmPairing() {
        val p = _pending.value ?: return
        repo.saveProfile(p)
        repo.updateSettings { it.copy(mode = null) } // start from the server's suggested mode
        _pending.value = null
    }

    fun dismissPairing() { _pending.value = null }
    fun clearPairError() { _pairError.value = null }

    fun forgetPairing() {
        TailbondVpnService.disconnect(getApplication())
        repo.clearProfile()
    }

    fun connect() = TailbondVpnService.connect(getApplication())
    fun disconnect() = TailbondVpnService.disconnect(getApplication())

    fun setMode(m: Mode) = repo.updateSettings { it.copy(mode = m) }
    fun setPath(kind: PathKind, s: PathSetting) = repo.updateSettings { it.withPath(kind, s) }
    fun setRouteMode(r: RouteMode) = repo.updateSettings { it.copy(routeMode = r) }
    fun setCapMb(mb: Long) = repo.updateSettings { it.copy(cellularCapMb = mb.coerceAtLeast(0)) }
    fun setBackupFec(on: Boolean) = repo.updateSettings { it.copy(experimentalBackupFec = on) }
    fun resetCellular() = repo.resetCellularBytes()

    fun setExcluded(pkg: String, excluded: Boolean) = repo.updateSettings {
        it.copy(excludedApps = if (excluded) it.excludedApps + pkg else it.excludedApps - pkg)
    }

    fun loadApps() {
        if (_apps.value.isNotEmpty()) return
        viewModelScope.launch {
            _apps.value = withContext(Dispatchers.IO) {
                val pm = getApplication<Application>().packageManager
                val self = getApplication<Application>().packageName
                // launchable apps only (manifest <queries> for MAIN/LAUNCHER, no QUERY_ALL_PACKAGES)
                val launcher = Intent(Intent.ACTION_MAIN).addCategory(Intent.CATEGORY_LAUNCHER)
                pm.queryIntentActivities(launcher, 0)
                    .map { it.activityInfo.applicationInfo }
                    .filter { it.packageName != self }
                    .distinctBy { it.packageName }
                    .map { AppEntry(it.packageName, pm.getApplicationLabel(it).toString()) }
                    .sortedBy { it.label.lowercase() }
            }
        }
    }
}
