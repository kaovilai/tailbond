package dev.tailbond.app.ui

import android.Manifest
import android.content.ActivityNotFoundException
import android.content.Intent
import android.content.pm.PackageManager
import android.net.ConnectivityManager
import android.net.NetworkCapabilities
import android.net.VpnService
import android.os.Build
import android.os.Bundle
import android.provider.Settings
import android.widget.Toast
import androidx.activity.ComponentActivity
import androidx.activity.compose.setContent
import androidx.activity.enableEdgeToEdge
import androidx.activity.result.contract.ActivityResultContracts
import androidx.activity.viewModels
import androidx.compose.runtime.Composable
import androidx.compose.runtime.getValue
import androidx.compose.runtime.mutableStateOf
import androidx.compose.runtime.setValue
import androidx.core.content.ContextCompat
import androidx.core.content.getSystemService
import androidx.core.net.toUri
import androidx.lifecycle.compose.collectAsStateWithLifecycle
import dev.tailbond.app.BuildConfig
import dev.tailbond.app.failover.FailoverPolicy
import dev.tailbond.app.update.UpdateManager
import dev.tailbond.app.update.UpdatePolicy
import dev.tailbond.app.model.Mode
import dev.tailbond.app.model.PathKind
import dev.tailbond.app.model.PathSetting
import dev.tailbond.app.model.RouteMode
import dev.tailbond.app.pairing.Pairing
import dev.tailbond.app.service.ConnState
import dev.tailbond.app.service.LiveState

class MainActivity : ComponentActivity() {
    private val vm: MainViewModel by viewModels()

    private val vpnConsent = registerForActivityResult(ActivityResultContracts.StartActivityForResult()) {
        if (VpnService.prepare(this) == null) vm.connect()
    }
    private val notifPermission = registerForActivityResult(ActivityResultContracts.RequestPermission()) {}

    private var tailscaleInstalled by mutableStateOf(false)

    /** What this device can open (TVs: no camera, often no browser or VPN settings screen). */
    private val hasCamera by lazy { packageManager.hasSystemFeature(PackageManager.FEATURE_CAMERA_ANY) }
    private val hasBrowser by lazy { canOpen(Intent(Intent.ACTION_VIEW, "https://github.com".toUri())) }
    private val hasVpnSettings by lazy { canOpen(Intent(Settings.ACTION_VPN_SETTINGS)) }

    private fun canOpen(i: Intent): Boolean = packageManager.resolveActivity(i, PackageManager.MATCH_DEFAULT_ONLY) != null

    /** startActivity, or nothing when no app takes it (a TV without a browser or a settings screen). */
    private fun tryStart(i: Intent): Boolean = try {
        startActivity(i)
        true
    } catch (_: ActivityNotFoundException) {
        false
    }

    /** The passphrase typed in "Save a backup", until the file picker answers. Memory only. */
    private var exportPassphrase: String? = null
    private var exportName: String = ""
    private val createBackup = registerForActivityResult(ActivityResultContracts.CreateDocument("application/octet-stream")) { uri ->
        val pass = exportPassphrase
        exportPassphrase = null
        if (uri != null && pass != null) vm.exportBackup(uri, pass, exportName)
        // the activity was recreated while the picker was open: the passphrase is gone on purpose
        else if (uri != null) vm.backupError("Nothing was saved: tap Save a backup again and type the passphrase.")
    }
    private val openBackup = registerForActivityResult(ActivityResultContracts.OpenDocument()) { uri ->
        if (uri != null) vm.restoreFilePicked(uri)
    }

    override fun onCreate(savedInstanceState: Bundle?) {
        super.onCreate(savedInstanceState)
        enableEdgeToEdge()
        handleIntent(intent)
        // a TV shows no notification shade: nothing to ask for there (every action is in the app)
        if (Build.VERSION.SDK_INT >= 33 && !resources.configuration.isTelevision() &&
            ContextCompat.checkSelfPermission(this, Manifest.permission.POST_NOTIFICATIONS) != PackageManager.PERMISSION_GRANTED
        ) {
            notifPermission.launch(Manifest.permission.POST_NOTIFICATIONS)
        }
        setContent { TailbondTheme { App() } }
    }

    override fun onResume() {
        super.onResume()
        tailscaleInstalled = tailscaleLaunchIntent() != null
        // about once a day: GitHub's latest release (docs/android.md, "Updates")
        vm.updatesDue()
    }

    override fun onNewIntent(intent: Intent) {
        super.onNewIntent(intent)
        handleIntent(intent)
    }

    private fun handleIntent(intent: Intent?) {
        val data = intent?.data
        if (intent?.action == Intent.ACTION_VIEW && data?.scheme == Pairing.SCHEME) {
            val ask = Pairing.askServer(data.toString())
            if (ask != null) vm.askFromLink(ask) else vm.offerPairing(data.toString())
        } else if (intent?.action == ACTION_CONNECT && vm.profile.value != null) {
            requestConnect()
        }
    }

    /** Explain the VPN handoff in our words before Android's own dialog. */
    private fun requestConnect() {
        when {
            otherVpnActive() -> vm.askHandoff(Handoff.OTHER_VPN_ACTIVE)
            VpnService.prepare(this) != null -> vm.askHandoff(Handoff.FIRST_PERMISSION)
            else -> vm.connect()
        }
    }

    private fun startConsentOrConnect() {
        vm.askHandoff(null)
        val consent = VpnService.prepare(this)
        if (consent == null) {
            vm.connect()
            return
        }
        try {
            vpnConsent.launch(consent)
        } catch (_: ActivityNotFoundException) {
            // some TV builds leave out Android's VPN permission dialog
            Toast.makeText(this, "This device has no screen to allow a VPN app, so tailbond can't turn on here.", Toast.LENGTH_LONG).show()
        }
    }

    /** Some other app's VPN (normally Tailscale) is up while ours isn't. */
    private fun otherVpnActive(): Boolean {
        val s = LiveState.live.value.state
        if (s != ConnState.DISCONNECTED && s != ConnState.ERROR) return false
        val cm = getSystemService<ConnectivityManager>() ?: return false
        @Suppress("DEPRECATION")
        return cm.allNetworks.any { cm.getNetworkCapabilities(it)?.hasTransport(NetworkCapabilities.TRANSPORT_VPN) == true }
    }

    // on Android TV, Tailscale may have only a TV (leanback) launcher entry
    private fun tailscaleLaunchIntent(): Intent? =
        packageManager.getLaunchIntentForPackage(TAILSCALE_PACKAGE) ?: packageManager.getLeanbackLaunchIntentForPackage(TAILSCALE_PACKAGE)

    private fun switchToTailscale() {
        vm.handToTailscale()
        val launch = tailscaleLaunchIntent()
        if (!tryStart(launch ?: Intent(Intent.ACTION_VIEW, "market://details?id=$TAILSCALE_PACKAGE".toUri()))) {
            tryStart(Intent(Intent.ACTION_VIEW, "https://play.google.com/store/apps/details?id=$TAILSCALE_PACKAGE".toUri()))
        }
    }

    /**
     * Android's own sign-in for a captive Wi-Fi: the internet panel (or Wi-Fi
     * settings) lists the network as "Sign in to network"; tapping it opens
     * the system's portal login, which uses the Wi-Fi directly, past our VPN.
     */
    private fun openWifiSignIn() {
        val panel = if (android.os.Build.VERSION.SDK_INT >= 29) Intent(Settings.Panel.ACTION_INTERNET_CONNECTIVITY) else Intent(Settings.ACTION_WIFI_SETTINGS)
        // TVs: no internet panel; their Settings has the Wi-Fi list
        if (!tryStart(panel)) tryStart(Intent(Settings.ACTION_WIFI_SETTINGS))
    }

    private fun openHomeServerPage(server: String) {
        val host = server.trim().removePrefix("http://").removePrefix("https://").substringBefore('/')
        if (host.isEmpty()) return
        val withPort = if (':' in host) host else "$host:$AGENT_PORT"
        // the server's phone page ("Get tailbond"), also served on the dashboard port
        tryStart(Intent(Intent.ACTION_VIEW, "http://$withPort/get".toUri()))
    }

    private val actions = object : UiActions {
        override fun setOn(on: Boolean) = if (on) requestConnect() else vm.disconnect()
        override fun retry() = requestConnect()
        override fun finishOnboarding() = vm.finishOnboarding()
        override fun offerPairing(text: String) = vm.offerPairing(text)
        override fun confirmPairing() = vm.confirmPairing()
        override fun dismissPairing() = vm.dismissPairing()
        override fun clearPairError() = vm.clearPairError()
        override fun openHomeServerPage(server: String) = this@MainActivity.openHomeServerPage(server)
        override fun askToConnect(server: String) = vm.askToConnect(server)
        override fun cancelAsk() = vm.cancelAsk()
        override fun openWifiSignIn() = this@MainActivity.openWifiSignIn()
        override fun setCaptivePrompts(on: Boolean) = vm.setCaptivePrompts(on)
        override fun forgetPairing() = vm.forgetPairing()
        override fun setMode(m: Mode) = vm.setMode(m)
        override fun setPath(kind: PathKind, s: PathSetting) = vm.setPath(kind, s)
        override fun setRouteMode(r: RouteMode) = vm.setRouteMode(r)
        override fun setCapMb(mb: Long) = vm.setCapMb(mb)
        override fun setBackupFec(on: Boolean) = vm.setBackupFec(on)
        override fun resetCellular() = vm.resetCellular()
        override fun setExcluded(pkg: String, excluded: Boolean) = vm.setExcluded(pkg, excluded)
        override fun loadApps() = vm.loadApps()
        override fun refreshPeers() = vm.refreshPeers()
        override fun confirmHandoff() = startConsentOrConnect()
        override fun dismissHandoff() = vm.askHandoff(null)
        override fun switchToTailscale() = this@MainActivity.switchToTailscale()
        override fun openVpnSettings() { tryStart(Intent(Settings.ACTION_VPN_SETTINGS)) }
        override fun startAddServer() = vm.setPairTarget(PairTarget.Add)
        override fun startRepair(id: String?) {
            val target = id ?: vm.servers.value.firstOrNull { it.id == vm.live.value.failover?.activeId }?.id ?: vm.servers.value.firstOrNull()?.id
            vm.setPairTarget(target?.let { PairTarget.Replace(it) })
        }
        override fun useServerNow(id: String) = vm.useServerNow(id)
        override fun resumeAutomatic() = vm.resumeAutomatic()
        override fun moveServer(id: String, delta: Int) = vm.moveServer(id, delta)
        override fun removeServer(id: String) = vm.removeServer(id)
        override fun setFailoverPolicy(policy: FailoverPolicy, server: String?) = vm.setFailoverPolicy(policy, server)
        override fun dismissServerProblem() = vm.dismissServerProblem()
        override fun clearPairTarget() = vm.setPairTarget(null)
        override fun addWebsite(domain: String, note: String) = vm.addWebsite(domain, note)
        override fun setWebsiteEnabled(domain: String, on: Boolean) = vm.setWebsiteEnabled(domain, on)
        override fun removeWebsite(domain: String) = vm.removeWebsite(domain)
        override fun updatesShown() = vm.updatesDue()
        override fun checkUpdates() = vm.checkUpdates()
        override fun setUpdatePolicy(p: UpdatePolicy) = vm.setUpdatePolicy(p)
        override fun openUrl(url: String) {
            if (!url.startsWith("https://")) return
            tryStart(Intent(Intent.ACTION_VIEW, url.toUri()))
        }
        override fun exportBackup(passphrase: String) {
            exportPassphrase = passphrase
            exportName = vm.backupFileName()
            try {
                createBackup.launch(exportName)
            } catch (_: ActivityNotFoundException) {
                exportPassphrase = null
                vm.backupError("This phone has no file picker to save with.")
            }
        }
        override fun pickRestoreFile() {
            try {
                openBackup.launch(arrayOf("*/*"))
            } catch (_: ActivityNotFoundException) {
                vm.backupError("This phone has no file picker.")
            }
        }
        override fun openRestore(passphrase: String) = vm.openRestore(passphrase)
        override fun confirmRestore() = vm.confirmRestore()
        override fun undoRestore() = vm.undoRestore()
        override fun pushBackup(passphrase: String) = vm.pushBackup(passphrase)
        override fun dismissBackup() = vm.dismissBackup()
        override fun startShare() = dev.tailbond.app.share.ShareService.start(this@MainActivity)
        override fun startHub() = dev.tailbond.app.share.HubService.start(this@MainActivity)
        override fun stopHub() = dev.tailbond.app.share.HubService.stop(this@MainActivity)
        override fun stopShare() = dev.tailbond.app.share.ShareService.stop(this@MainActivity)
        override fun setShareMinBattery(pct: Int) = vm.setShareMinBattery(pct)
        override fun setShareCapMb(mb: Long) = vm.setShareCapMb(mb)
        override fun setShareAllowedHubs(hubs: List<String>) = vm.setShareAllowedHubs(hubs)
        override fun setUseShares(on: Boolean) = vm.setUseShares(on)
    }

    @Composable
    private fun App() {
        val state = UiState(
            profile = vm.profile.collectAsStateWithLifecycle().value,
            settings = vm.settings.collectAsStateWithLifecycle().value,
            live = vm.live.collectAsStateWithLifecycle().value,
            cellularBytes = vm.cellularBytes.collectAsStateWithLifecycle().value,
            peers = vm.peers.collectAsStateWithLifecycle().value,
            pending = vm.pending.collectAsStateWithLifecycle().value,
            pairError = vm.pairError.collectAsStateWithLifecycle().value,
            handoff = vm.handoff.collectAsStateWithLifecycle().value,
            tailscaleInstalled = tailscaleInstalled,
            apps = vm.apps.collectAsStateWithLifecycle().value,
            ask = vm.ask.collectAsStateWithLifecycle().value,
            wifiSignIn = vm.wifiSignIn.collectAsStateWithLifecycle().value,
            servers = vm.servers.collectAsStateWithLifecycle().value,
            switches = vm.switches.collectAsStateWithLifecycle().value,
            pairTarget = vm.pairTarget.collectAsStateWithLifecycle().value,
            serverSuggestions = vm.serverSuggestions.collectAsStateWithLifecycle().value,
            pendingTarget = vm.pendingTarget.collectAsStateWithLifecycle().value,
            serverProblem = vm.serverProblem.collectAsStateWithLifecycle().value,
            sdkInt = Build.VERSION.SDK_INT,
            update = UpdateUi(
                status = vm.updateStatus.collectAsStateWithLifecycle().value,
                repo = UpdateManager.UPDATE_REPO,
                debugBuild = BuildConfig.DEBUG,
            ),
            backup = vm.backup.collectAsStateWithLifecycle().value,
            share = dev.tailbond.app.share.ShareState.live.collectAsStateWithLifecycle().value,
            hub = dev.tailbond.app.share.HubState.live.collectAsStateWithLifecycle().value,
            hubShares = dev.tailbond.app.share.ShareHubState.live.collectAsStateWithLifecycle().value,
            hasCamera = hasCamera,
            hasBrowser = hasBrowser,
            hasVpnSettings = hasVpnSettings,
        )
        TailbondApp(state, actions)
    }

    companion object {
        const val ACTION_CONNECT = "dev.tailbond.app.UI_CONNECT"
        const val TAILSCALE_PACKAGE = "com.tailscale.ipn"
        const val AGENT_PORT = 8077
    }
}
