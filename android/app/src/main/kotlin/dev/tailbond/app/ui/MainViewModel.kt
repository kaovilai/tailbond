package dev.tailbond.app.ui

import android.app.Application
import android.content.Intent
import android.os.Build
import androidx.lifecycle.AndroidViewModel
import androidx.lifecycle.viewModelScope
import dev.tailbond.app.data.AskState
import dev.tailbond.app.data.PairRequestClient
import dev.tailbond.app.data.PeersClient
import dev.tailbond.app.data.Repository
import dev.tailbond.app.data.next
import dev.tailbond.app.data.phoneRequestName
import dev.tailbond.app.model.Mode
import dev.tailbond.app.model.PathKind
import dev.tailbond.app.model.PathSetting
import dev.tailbond.app.model.RouteMode
import dev.tailbond.app.pairing.Pairing
import dev.tailbond.app.pairing.PairingPayload
import dev.tailbond.app.pairing.PairingResult
import dev.tailbond.app.service.CaptiveWatcher
import dev.tailbond.app.service.ConnState
import dev.tailbond.app.service.LiveState
import dev.tailbond.app.service.TailbondVpnService
import kotlinx.coroutines.Dispatchers
import kotlinx.coroutines.Job
import kotlinx.coroutines.delay
import kotlinx.coroutines.flow.MutableStateFlow
import kotlinx.coroutines.flow.SharingStarted
import kotlinx.coroutines.flow.StateFlow
import kotlinx.coroutines.flow.asStateFlow
import kotlinx.coroutines.flow.collectLatest
import kotlinx.coroutines.flow.combine
import kotlinx.coroutines.flow.distinctUntilChanged
import kotlinx.coroutines.flow.map
import kotlinx.coroutines.flow.stateIn
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

    private val _ask = MutableStateFlow<AskState>(AskState.Idle)
    val ask: StateFlow<AskState> = _ask.asStateFlow()
    private var askJob: Job? = null

    private val _handoff = MutableStateFlow<Handoff?>(null)
    val handoff: StateFlow<Handoff?> = _handoff.asStateFlow()

    private val peerStatus = MutableStateFlow(PeersUi())
    /** Machine list: the cached list plus the state of the last refresh. */
    val peers: StateFlow<PeersUi> = combine(repo.peers, peerStatus) { list, st -> st.copy(list = list) }
        .stateIn(viewModelScope, SharingStarted.Eagerly, PeersUi(repo.peers.value))

    private val captive = CaptiveWatcher(app).also { it.start() }
    val wifiSignIn = CaptiveWatcher.wifiSignIn

    override fun onCleared() {
        captive.stop()
        super.onCleared()
    }

    fun setCaptivePrompts(on: Boolean) = repo.updateSettings { it.copy(captivePrompts = on) }

    init {
        // refresh the machine list every 30 s while connected
        viewModelScope.launch {
            live.map { it.state == ConnState.CONNECTED }.distinctUntilChanged().collectLatest { up ->
                while (up) {
                    refreshPeersNow()
                    delay(30_000)
                }
            }
        }
    }

    private suspend fun refreshPeersNow() {
        val key = profile.value?.key ?: return
        peerStatus.value = peerStatus.value.copy(loading = true)
        val err = withContext(Dispatchers.IO) {
            try {
                repo.savePeers(PeersClient.fetch(key)); null
            } catch (e: Exception) {
                e.message ?: e.javaClass.simpleName
            }
        }
        peerStatus.value = PeersUi(loading = false, error = err)
    }

    fun refreshPeers() { viewModelScope.launch { refreshPeersNow() } }

    /**
     * "Ask to connect": find [typed] on the tailnet (Tailscale's VPN is still
     * on), ask it to pair this phone, wait for the approval, then show the
     * usual confirmation with the pairing it sent.
     */
    fun askToConnect(typed: String) {
        askJob?.cancel()
        val server = PairRequestClient.hostOf(typed).substringBefore('.').ifEmpty { typed.trim() }
        _ask.value = AskState.Asking(server)
        askJob = viewModelScope.launch {
            val (addr, created) = try {
                withContext(Dispatchers.IO) {
                    val a = PairRequestClient.locate(typed)
                    a to PairRequestClient.ask(a, phoneRequestName(Build.MODEL ?: "phone"))
                }
            } catch (e: PairRequestClient.AskException) {
                _ask.value = AskState.Failed(server, e.failure); return@launch
            } catch (e: Exception) {
                _ask.value = AskState.Failed(server, PairRequestClient.Failure.UNREACHABLE); return@launch
            }
            var waiting = AskState.Waiting(server, created.name)
            _ask.value = waiting
            val deadline = System.currentTimeMillis() + 15 * 60_000
            while (System.currentTimeMillis() < deadline) {
                delay(3_000)
                val p = try { withContext(Dispatchers.IO) { PairRequestClient.poll(addr, created) } } catch (_: Exception) { continue }
                val next = waiting.next(p) { uri -> offerPairing(uri) }
                if (next != null) { _ask.value = next; return@launch }
                waiting = waiting.copy(name = p.name.ifEmpty { waiting.name })
            }
            _ask.value = AskState.Expired(server)
        }
    }

    fun cancelAsk() {
        askJob?.cancel()
        _ask.value = AskState.Idle
    }

    fun finishOnboarding() = repo.updateSettings { it.copy(onboarded = true) }

    fun askHandoff(h: Handoff?) { _handoff.value = h }

    /** "Switch back to Tailscale": stop, and remember it so nothing turns tailbond back on by itself. */
    fun handToTailscale() {
        repo.updateSettings { it.copy(handedToTailscale = true) }
        disconnect()
    }

    /** Returns true when [text] parsed; the payload then waits in [pending]. */
    fun offerPairing(text: String): Boolean = when (val r = Pairing.parse(text)) {
        is PairingResult.Ok -> { _pending.value = r.payload; _pairError.value = null; true }
        is PairingResult.Invalid -> { _pairError.value = r.reason; false }
    }

    fun confirmPairing() {
        val p = _pending.value ?: return
        repo.saveProfile(p)
        // start from the server's suggested mode
        repo.updateSettings { it.copy(mode = null, onboarded = true) }
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
