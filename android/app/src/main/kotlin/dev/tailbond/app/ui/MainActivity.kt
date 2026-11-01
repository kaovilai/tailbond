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

    override fun onCreate(savedInstanceState: Bundle?) {
        super.onCreate(savedInstanceState)
        enableEdgeToEdge()
        handleIntent(intent)
        if (Build.VERSION.SDK_INT >= 33 &&
            ContextCompat.checkSelfPermission(this, Manifest.permission.POST_NOTIFICATIONS) != PackageManager.PERMISSION_GRANTED
        ) {
            notifPermission.launch(Manifest.permission.POST_NOTIFICATIONS)
        }
        setContent { TailbondTheme { App() } }
    }

    override fun onResume() {
        super.onResume()
        tailscaleInstalled = tailscaleLaunchIntent() != null
    }

    override fun onNewIntent(intent: Intent) {
        super.onNewIntent(intent)
        handleIntent(intent)
    }

    private fun handleIntent(intent: Intent?) {
        val data = intent?.data
        if (intent?.action == Intent.ACTION_VIEW && data?.scheme == Pairing.SCHEME) {
            vm.offerPairing(data.toString())
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
        if (consent != null) vpnConsent.launch(consent) else vm.connect()
    }

    /** Some other app's VPN (normally Tailscale) is up while ours isn't. */
    private fun otherVpnActive(): Boolean {
        val s = LiveState.live.value.state
        if (s != ConnState.DISCONNECTED && s != ConnState.ERROR) return false
        val cm = getSystemService<ConnectivityManager>() ?: return false
        @Suppress("DEPRECATION")
        return cm.allNetworks.any { cm.getNetworkCapabilities(it)?.hasTransport(NetworkCapabilities.TRANSPORT_VPN) == true }
    }

    private fun tailscaleLaunchIntent(): Intent? = packageManager.getLaunchIntentForPackage(TAILSCALE_PACKAGE)

    private fun switchToTailscale() {
        vm.handToTailscale()
        val launch = tailscaleLaunchIntent()
        try {
            startActivity(launch ?: Intent(Intent.ACTION_VIEW, "market://details?id=$TAILSCALE_PACKAGE".toUri()))
        } catch (_: ActivityNotFoundException) {
            startActivity(Intent(Intent.ACTION_VIEW, "https://play.google.com/store/apps/details?id=$TAILSCALE_PACKAGE".toUri()))
        }
    }

    private fun openHomeServerPage(server: String) {
        val host = server.trim().removePrefix("http://").removePrefix("https://").substringBefore('/')
        if (host.isEmpty()) return
        val withPort = if (':' in host) host else "$host:$AGENT_PORT"
        try {
            startActivity(Intent(Intent.ACTION_VIEW, "http://$withPort/#add-phone".toUri()))
        } catch (_: ActivityNotFoundException) {
        }
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
        override fun openVpnSettings() = startActivity(Intent(Settings.ACTION_VPN_SETTINGS))
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
        )
        TailbondApp(state, actions)
    }

    companion object {
        const val ACTION_CONNECT = "dev.tailbond.app.UI_CONNECT"
        const val TAILSCALE_PACKAGE = "com.tailscale.ipn"
        const val AGENT_PORT = 8077
    }
}
