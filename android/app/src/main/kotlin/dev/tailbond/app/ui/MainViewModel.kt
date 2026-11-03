package dev.tailbond.app.ui

import android.app.Application
import android.content.Intent
import android.os.Build
import android.util.Log
import androidx.lifecycle.AndroidViewModel
import androidx.lifecycle.viewModelScope
import android.net.Uri
import dev.tailbond.app.backup.BackupException
import dev.tailbond.app.backup.BackupFile
import dev.tailbond.app.backup.BackupManager
import dev.tailbond.app.data.AskState
import dev.tailbond.app.update.UpdateManager
import dev.tailbond.app.update.UpdatePolicy
import dev.tailbond.app.data.PairRequestClient
import dev.tailbond.app.data.PeersClient
import dev.tailbond.app.data.Repository
import dev.tailbond.app.data.next
import dev.tailbond.app.data.phoneRequestName
import dev.tailbond.app.failover.FailoverPolicy
import dev.tailbond.app.model.AddCheck
import dev.tailbond.app.model.Mode
import dev.tailbond.app.model.ServerList
import dev.tailbond.app.model.WebsiteBypass
import dev.tailbond.app.model.WebsiteRule
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
import kotlinx.coroutines.async
import kotlinx.coroutines.awaitAll
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

private const val TAG = "tailbond"

class MainViewModel(app: Application) : AndroidViewModel(app) {
    private val repo = Repository.get(app)

    val live = LiveState.live
    val profile = repo.profile
    val servers = repo.servers
    val switches = repo.switches

    private val _target = MutableStateFlow<PairTarget?>(null)
    /** What the pairing screen is for (add a backup, pair one again); null = first pairing. */
    val pairTarget: StateFlow<PairTarget?> = _target.asStateFlow()
    private val _pendingTarget = MutableStateFlow<PairTarget?>(null)
    /** Where the pending pairing goes once confirmed (null = first pairing). */
    val pendingTarget: StateFlow<PairTarget?> = _pendingTarget.asStateFlow()

    private val _serverProblem = MutableStateFlow<Pair<String, AddCheck>?>(null)
    val serverProblem: StateFlow<Pair<String, AddCheck>?> = _serverProblem.asStateFlow()
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

    // --- updates ---

    private val updates = UpdateManager.get(app)
    val updateStatus = updates.status

    /** The app came to the front, or the Updates page opened: check if the last check is a day old. */
    fun updatesDue() = updates.onForeground()
    fun checkUpdates() = updates.checkNow()
    fun setUpdatePolicy(p: UpdatePolicy) = repo.updateSettings { it.copy(updatePolicy = p) }

    // --- backup and restore ---

    private val backups = BackupManager(app)
    private val _backup = MutableStateFlow(BackupUi())
    val backup: StateFlow<BackupUi> = combine(_backup, backups.undoAt) { b, u -> b.copy(undoAt = u) }
        .stateIn(viewModelScope, SharingStarted.Eagerly, BackupUi(undoAt = backups.undoAt.value))
    /** The file picked for a restore, until it's restored or dismissed. */
    private var pickedFile: ByteArray? = null

    private fun backupJob(busy: String, work: suspend () -> Unit) {
        if (_backup.value.busy != null) return
        _backup.value = _backup.value.copy(busy = busy, error = null, message = null)
        viewModelScope.launch {
            try {
                work()
            } catch (e: BackupException) {
                _backup.value = _backup.value.copy(error = e.message)
            } catch (e: Exception) {
                _backup.value = _backup.value.copy(error = "Didn't work: ${e.message ?: e.javaClass.simpleName}")
            } finally {
                _backup.value = _backup.value.copy(busy = null)
            }
        }
    }

    fun exportBackup(uri: Uri, passphrase: String, name: String) = backupJob("Encrypting the backup…") {
        withContext(Dispatchers.IO) { backups.export(uri, passphrase) }
        _backup.value = _backup.value.copy(message = "Saved $name. Keep the passphrase somewhere else than the file.")
    }

    /** Suggested file name for a new backup. */
    fun backupFileName(): String =
        "tailbond-${dev.tailbond.app.data.phoneRequestName(Build.MODEL ?: "phone")}-${java.time.LocalDate.now()}.tbbackup"

    fun restoreFilePicked(uri: Uri) = backupJob("Reading the file…") {
        val bytes = withContext(Dispatchers.IO) { backups.read(uri) }
        val (h, _) = BackupFile.peek(bytes)
        dev.tailbond.app.backup.Restore.checkKind(h)
        pickedFile = bytes
        _backup.value = _backup.value.copy(picked = h, preview = null, restoredAt = null)
    }

    fun openRestore(passphrase: String) {
        val file = pickedFile ?: return
        if (_backup.value.busy != null) return
        _backup.value = _backup.value.copy(busy = "Opening the backup… (this takes a second)", error = null)
        viewModelScope.launch {
            try {
                val p = withContext(Dispatchers.Default) { backups.preview(file, passphrase) }
                _backup.value = _backup.value.copy(picked = null, preview = p)
            } catch (e: BackupException) {
                _backup.value = _backup.value.copy(error = e.message)
            } catch (e: Exception) {
                _backup.value = _backup.value.copy(error = BackupFile.WRONG)
            } finally {
                _backup.value = _backup.value.copy(busy = null)
            }
        }
    }

    fun confirmRestore() {
        val p = _backup.value.preview ?: return
        backupJob("Restoring…") {
            withContext(Dispatchers.IO) { backups.restore(p) }
            pickedFile = null
            _backup.value = _backup.value.copy(preview = null, restoredAt = System.currentTimeMillis(), message = null)
        }
    }

    fun undoRestore() = backupJob("Putting everything back…") {
        val ok = withContext(Dispatchers.IO) { backups.undo() }
        _backup.value = _backup.value.copy(
            restoredAt = null, preview = null,
            message = if (ok) "Undone: your servers and settings are back as they were before the restore." else null,
            error = if (ok) null else "There's nothing to undo any more.",
        )
    }

    fun pushBackup(passphrase: String) = backupJob("Sending the backup…") {
        val name = withContext(Dispatchers.IO) { backups.push(passphrase) }
        _backup.value = _backup.value.copy(message = "Sent to $name. It keeps the last 3 backups from this phone, still encrypted.")
    }

    fun dismissBackup() {
        pickedFile = null
        _backup.value = _backup.value.copy(picked = null, preview = null, restoredAt = null, error = null)
    }

    fun backupError(msg: String) { _backup.value = _backup.value.copy(error = msg) }

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

    /** The server in use (failover's active one), else the main one. */
    private fun activeServer() = servers.value.firstOrNull { it.id == live.value.failover?.activeId } ?: servers.value.firstOrNull()

    private suspend fun refreshPeersNow() {
        val server = activeServer() ?: return
        val key = server.payload.key
        val host = ServerList.serverTunnelIp(server.payload)
        peerStatus.value = peerStatus.value.copy(loading = true)
        val err = withContext(Dispatchers.IO) {
            try {
                val list = PeersClient.fetch(key, host, dev.tailbond.app.BuildConfig.VERSION_NAME)
                repo.savePeers(list)
                Log.i(TAG, "machine list from $host: ${list.peers.size} machines")
                null
            } catch (e: Exception) {
                (e.message ?: e.javaClass.simpleName).also { Log.w(TAG, "machine list from $host failed: $it") }
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
            val avoid = avoidSubnets()
            val (addr, created) = try {
                withContext(Dispatchers.IO) {
                    val a = PairRequestClient.locate(typed)
                    a to PairRequestClient.ask(a, phoneRequestName(Build.MODEL ?: "phone"), avoid)
                }
            } catch (e: PairRequestClient.AskException) {
                _ask.value = AskState.Failed(server, e.failure); return@launch
            } catch (e: Exception) {
                _ask.value = AskState.Failed(server, PairRequestClient.Failure.UNREACHABLE); return@launch
            }
            val seenAs = listOf(created.device, created.user, created.ip).filter { it.isNotBlank() }.joinToString(" · ")
            var waiting = AskState.Waiting(server, created.name, seenAs)
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

    /** From the home server's phone page: skip the intro and ask right away (unless already paired). */
    fun askFromLink(server: String) {
        if (profile.value != null) return
        finishOnboarding()
        askToConnect(server)
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

    /** Subnets of the servers this phone keeps (sent as avoid_subnets when asking a new one). */
    private fun avoidSubnets(): List<String> {
        val list = servers.value
        return when (val t = _target.value) {
            PairTarget.Add -> list.map { it.subnet.toString() }
            is PairTarget.Replace -> list.filter { it.id != t.id }.map { it.subnet.toString() }
            null -> emptyList()
        }
    }

    fun setPairTarget(t: PairTarget?) {
        if (_target.value != t) cancelAsk()
        _target.value = t
        _pairError.value = null
        if (t == PairTarget.Add) findServers() else { suggestJob?.cancel(); _suggestions.value = emptyList() }
    }

    private val _suggestions = MutableStateFlow<List<String>>(emptyList())
    /** Tailnet machines that answered as tailbond servers ("Add another server" suggestions). */
    val serverSuggestions: StateFlow<List<String>> = _suggestions.asStateFlow()
    private var suggestJob: Job? = null

    /**
     * Asks every online machine in the tailnet list the app already has
     * (through the server in use) whether it's a tailbond server, in parallel
     * with a short timeout; only those that say so are offered.
     */
    private fun findServers() {
        suggestJob?.cancel()
        _suggestions.value = emptyList()
        val known = servers.value.map { it.name.lowercase() }.toSet()
        val candidates = repo.peers.value?.peers.orEmpty()
            .filter { it.online && it.hostname.isNotBlank() && it.hostname.lowercase() !in known && it.ipv4 != null }
            .take(32)
        if (candidates.isEmpty()) return
        suggestJob = viewModelScope.launch {
            _suggestions.value = withContext(Dispatchers.IO) {
                candidates.map { n ->
                    async { n.hostname.takeIf { runCatching { PairRequestClient.isServer(java.net.InetAddress.getByName(n.ipv4)) }.getOrDefault(false) } }
                }.awaitAll().filterNotNull().sortedBy { it.lowercase() }
            }
        }
    }

    /** Returns true when [text] parsed; the payload then waits in [pending] (or [serverProblem] says why it can't be added). */
    fun offerPairing(text: String): Boolean = when (val r = Pairing.parse(text)) {
        is PairingResult.Ok -> { offer(r.payload); _pairError.value = null; true }
        is PairingResult.Invalid -> { _pairError.value = r.reason; false }
    }

    private fun offer(p: PairingPayload) {
        val list = servers.value
        val target: PairTarget? = if (list.isEmpty()) null else when (val t = _target.value) {
            is PairTarget.Replace -> t
            PairTarget.Add -> PairTarget.Add
            // a link from outside: renews the server it belongs to, else adds it as a backup
            null -> ServerList.matching(list, p)?.let { PairTarget.Replace(it.id) } ?: PairTarget.Add
        }
        val check = when (target) {
            null -> AddCheck.Ok
            PairTarget.Add -> ServerList.check(list, p)
            is PairTarget.Replace -> ServerList.check(list, p, replacing = target.id)
        }
        if (check != AddCheck.Ok) {
            _serverProblem.value = ServerList.nameOf(p) to check
            return
        }
        _pendingTarget.value = target
        _pending.value = p
    }

    fun confirmPairing() {
        val p = _pending.value ?: return
        when (val t = _pendingTarget.value) {
            null -> {
                repo.saveProfile(p)
                // start from the server's suggested mode
                repo.updateSettings { it.copy(mode = null, onboarded = true) }
            }
            PairTarget.Add -> repo.addServer(p)
            is PairTarget.Replace -> {
                repo.saveProfile(p, replacing = t.id)
                if (servers.value.firstOrNull()?.id == t.id) repo.updateSettings { it.copy(mode = null, onboarded = true) }
            }
        }
        _pending.value = null
        _pendingTarget.value = null
        _target.value = null
    }

    fun dismissPairing() { _pending.value = null; _pendingTarget.value = null }
    fun dismissServerProblem() { _serverProblem.value = null }

    // --- servers ---

    fun moveServer(id: String, delta: Int) = repo.moveServer(id, delta)
    fun removeServer(id: String) {
        if (servers.value.size <= 1) forgetPairing() else repo.removeServer(id)
    }

    /** "Use this server now": with "Always use …" it becomes the always-used server. */
    fun useServerNow(id: String) {
        if (repo.settings.value.failoverPolicy == FailoverPolicy.ALWAYS) {
            repo.updateSettings { it.copy(alwaysServerId = id) }
        } else {
            TailbondVpnService.useServer(getApplication(), id)
        }
    }

    fun resumeAutomatic() = TailbondVpnService.resumeAutomatic(getApplication())

    fun setFailoverPolicy(policy: FailoverPolicy, server: String?) =
        repo.updateSettings { it.copy(failoverPolicy = policy, alwaysServerId = if (policy == FailoverPolicy.ALWAYS) server else null) }

    // --- websites ---

    fun addWebsite(domain: String, note: String) {
        val d = WebsiteBypass.normalize(domain) ?: return
        repo.updateSettings { s ->
            if (s.websites.any { it.domain == d } || s.websites.size >= WebsiteBypass.MAX_RULES) s
            else s.copy(websites = s.websites + WebsiteRule(d, true, note.trim().take(64)))
        }
    }

    fun setWebsiteEnabled(domain: String, on: Boolean) = repo.updateSettings { s ->
        s.copy(websites = s.websites.map { if (it.domain == domain) it.copy(enabled = on) else it })
    }

    fun removeWebsite(domain: String) = repo.updateSettings { s -> s.copy(websites = s.websites.filter { it.domain != domain }) }
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
