package dev.tailbond.app.ux

import android.app.Application
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
import androidx.compose.runtime.Composable
import androidx.compose.ui.Modifier
import androidx.test.core.app.ApplicationProvider
import androidx.test.ext.junit.runners.AndroidJUnit4
import dev.tailbond.app.model.AppSettings
import dev.tailbond.app.model.Mode
import dev.tailbond.app.service.Live
import dev.tailbond.app.ui.ConfirmPairingDialog
import dev.tailbond.app.ui.DashboardScreen
import dev.tailbond.app.ui.MainViewModel
import dev.tailbond.app.ui.PairingScreen
import dev.tailbond.app.ui.SettingsScreen
import org.junit.Test
import org.junit.runner.RunWith
import org.robolectric.annotation.Config

/** Round-0 baseline: the app as it was before the Tailscale-familiarity redesign. */
@RunWith(AndroidJUnit4::class)
@Config(sdk = [35])
class BeforeScreens : UxRender() {
    private val vm by lazy { MainViewModel(ApplicationProvider.getApplicationContext<Application>()) }

    @OptIn(ExperimentalMaterial3Api::class)
    @Composable
    private fun Shell(tab: Int?, body: @Composable (Modifier) -> Unit) {
        Scaffold(
            topBar = { TopAppBar(title = { Text("tailbond") }) },
            bottomBar = {
                if (tab != null) NavigationBar {
                    NavigationBarItem(tab == 0, {}, { Icon(Icons.Filled.Dashboard, null) }, label = { Text("Dashboard") })
                    NavigationBarItem(tab == 1, {}, { Icon(Icons.Filled.Settings, null) }, label = { Text("Settings") })
                }
            },
        ) { body(Modifier.padding(it)) }
    }

    private fun dash(name: String, live: Live, settings: AppSettings = AppSettings(), dark: Boolean = false, tall: Int = PHONE_H, cell: Long = 1_230_000_000) =
        shot(name, dark, tall) {
            Shell(0) { DashboardScreen(live, UxFixtures.profile, settings, cell, onToggle = {}, vm = vm, modifier = it) }
        }

    @Test fun onboarding() = shot("01-onboarding-pairing") { Shell(null) { PairingScreen(vm, null, it) } }
    @Test fun onboardingDark() = shot("01-onboarding-pairing-dark", dark = true) { Shell(null) { PairingScreen(vm, null, it) } }
    @Test fun pairingError() = shot("02-pairing-paste-error") {
        Shell(null) { PairingScreen(vm, "pairing data is not base64url", it) }
    }
    @Test fun deepLinkConfirm() = shot("03-deeplink-confirm") {
        Shell(null) { PairingScreen(vm, null, it) }
        ConfirmPairingDialog(UxFixtures.profile, replacing = false, onConfirm = {}, onDismiss = {})
    }

    @Test fun connected() = dash("10-dashboard-connected", UxFixtures.connectedBoth)
    @Test fun connectedDark() = dash("10-dashboard-connected-dark", UxFixtures.connectedBoth, dark = true)
    @Test fun connectedLong() = dash("10-dashboard-connected-full", UxFixtures.connectedBoth, tall = 1500)
    @Test fun connecting() = dash("11-dashboard-connecting", UxFixtures.connecting)
    @Test fun disconnected() = dash("12-dashboard-disconnected", UxFixtures.disconnected)
    @Test fun disconnectedDark() = dash("12-dashboard-disconnected-dark", UxFixtures.disconnected, dark = true)
    @Test fun errUnreachable() = dash("13-error-unreachable", UxFixtures.errorUnreachable)
    @Test fun errCert() = dash("14-error-cert", UxFixtures.errorCert)
    @Test fun onePathFailed() = dash("15-one-path-failed", UxFixtures.onePathFailed, tall = 1500)
    @Test fun failoverStandby() = dash("16-failover-standby", UxFixtures.failoverStandby, AppSettings(mode = Mode.FAILOVER), tall = 1500)
    @Test fun failoverTakeover() = dash("17-failover-takeover", UxFixtures.failoverTakeover, AppSettings(mode = Mode.FAILOVER), tall = 1500)
    @Test fun modePicker() = dash("20-mode-picker", UxFixtures.connectedBoth, AppSettings(mode = Mode.STREAMING))

    @Test fun settings() = shot("30-settings", tallDp = 1400) {
        Shell(1) { SettingsScreen(vm, UxFixtures.profile, AppSettings(cellularCapMb = 5000), onRepair = {}, modifier = it) }
    }
    @Test fun settingsDark() = shot("30-settings-dark", dark = true) {
        Shell(1) { SettingsScreen(vm, UxFixtures.profile, AppSettings(cellularCapMb = 5000), onRepair = {}, modifier = it) }
    }
}
