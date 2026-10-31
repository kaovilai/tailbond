package dev.tailbond.app.ui

import android.Manifest
import android.content.Intent
import android.content.pm.PackageManager
import android.net.VpnService
import android.os.Build
import android.os.Bundle
import androidx.activity.ComponentActivity
import androidx.activity.compose.setContent
import androidx.activity.enableEdgeToEdge
import androidx.activity.result.contract.ActivityResultContracts
import androidx.activity.viewModels
import androidx.compose.foundation.layout.padding
import androidx.compose.material.icons.Icons
import androidx.compose.material.icons.filled.Dashboard
import androidx.compose.material.icons.filled.Settings
import androidx.compose.material3.ExperimentalMaterial3Api
import androidx.compose.material3.Icon
import androidx.compose.material3.NavigationBar
import androidx.compose.material3.NavigationBarItem
import androidx.compose.material3.Scaffold
import androidx.compose.material3.Text
import androidx.compose.material3.TopAppBar
import androidx.compose.runtime.getValue
import androidx.compose.runtime.mutableStateOf
import androidx.compose.runtime.saveable.rememberSaveable
import androidx.compose.runtime.setValue
import androidx.compose.ui.Modifier
import androidx.core.content.ContextCompat
import androidx.lifecycle.compose.collectAsStateWithLifecycle
import dev.tailbond.app.R
import dev.tailbond.app.pairing.Pairing

class MainActivity : ComponentActivity() {
    private val vm: MainViewModel by viewModels()

    private val vpnConsent = registerForActivityResult(ActivityResultContracts.StartActivityForResult()) {
        if (VpnService.prepare(this) == null) vm.connect()
    }
    private val notifPermission = registerForActivityResult(ActivityResultContracts.RequestPermission()) {}

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

    override fun onNewIntent(intent: Intent) {
        super.onNewIntent(intent)
        handleIntent(intent)
    }

    private fun handleIntent(intent: Intent?) {
        val data = intent?.data
        if (intent?.action == Intent.ACTION_VIEW && data?.scheme == Pairing.SCHEME) {
            vm.offerPairing(data.toString())
        } else if (intent?.action == ACTION_CONNECT && vm.profile.value != null) {
            connect()
        }
    }

    private fun connect() {
        val consent = VpnService.prepare(this)
        if (consent != null) vpnConsent.launch(consent) else vm.connect()
    }

    @OptIn(ExperimentalMaterial3Api::class)
    @androidx.compose.runtime.Composable
    private fun App() {
        val live by vm.live.collectAsStateWithLifecycle()
        val profile by vm.profile.collectAsStateWithLifecycle()
        val settings by vm.settings.collectAsStateWithLifecycle()
        val cellular by vm.cellularBytes.collectAsStateWithLifecycle()
        val pending by vm.pending.collectAsStateWithLifecycle()
        val pairError by vm.pairError.collectAsStateWithLifecycle()
        var tab by rememberSaveable { mutableStateOf(0) }
        var repairing by rememberSaveable { mutableStateOf(false) }

        val p = profile
        Scaffold(
            topBar = { TopAppBar(title = { Text(getString(R.string.app_name)) }) },
            bottomBar = {
                if (p != null && !repairing) {
                    NavigationBar {
                        NavigationBarItem(tab == 0, { tab = 0 }, { Icon(Icons.Filled.Dashboard, null) }, label = { Text("Dashboard") })
                        NavigationBarItem(tab == 1, { tab = 1 }, { Icon(Icons.Filled.Settings, null) }, label = { Text("Settings") })
                    }
                }
            },
        ) { pad ->
            val m = Modifier.padding(pad)
            when {
                p == null || repairing -> PairingScreen(vm, pairError, m)
                tab == 0 -> DashboardScreen(
                    live, p, settings, cellular,
                    onToggle = { on -> if (on) connect() else vm.disconnect() },
                    vm = vm, modifier = m,
                )
                else -> SettingsScreen(vm, p, settings, onRepair = { repairing = true }, modifier = m)
            }
        }
        pending?.let {
            ConfirmPairingDialog(
                it, replacing = p != null,
                onConfirm = { vm.confirmPairing(); repairing = false; tab = 0 },
                onDismiss = vm::dismissPairing,
            )
        }
    }

    companion object {
        const val ACTION_CONNECT = "dev.tailbond.app.UI_CONNECT"
    }
}
