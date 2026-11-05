package dev.tailbond.app.ux

import androidx.activity.ComponentActivity
import androidx.compose.foundation.Image
import androidx.compose.foundation.layout.size
import androidx.compose.runtime.Composable
import androidx.compose.ui.Modifier
import androidx.compose.ui.platform.testTag
import androidx.compose.ui.res.painterResource
import androidx.compose.ui.test.junit4.createAndroidComposeRule
import androidx.compose.ui.test.onNodeWithTag
import androidx.compose.ui.unit.dp
import androidx.test.ext.junit.runners.AndroidJUnit4
import com.github.takahirom.roborazzi.ExperimentalRoborazziApi
import com.github.takahirom.roborazzi.captureRoboImage
import com.github.takahirom.roborazzi.captureScreenRoboImage
import dev.tailbond.app.R
import dev.tailbond.app.data.AskState
import dev.tailbond.app.model.AppSettings
import dev.tailbond.app.model.Mode
import dev.tailbond.app.model.ServerProfile
import dev.tailbond.app.share.HubLive
import dev.tailbond.app.share.ShareLive
import dev.tailbond.app.tv.Remote
import dev.tailbond.app.tv.TV
import dev.tailbond.app.ui.Handoff
import dev.tailbond.app.ui.NoActions
import dev.tailbond.app.ui.PeersUi
import dev.tailbond.app.ui.Routes
import dev.tailbond.app.ui.TailbondApp
import dev.tailbond.app.ui.TailbondTheme
import dev.tailbond.app.ui.UiState
import dev.tailbond.app.update.UpdateStatus
import org.junit.Assume.assumeTrue
import org.junit.Before
import org.junit.Rule
import org.junit.Test
import org.junit.runner.RunWith
import org.robolectric.RuntimeEnvironment
import org.robolectric.annotation.Config
import java.io.File

/**
 * Android TV (docs/android.md, "Android TV"): the key screens on a 1080p TV
 * (960×540dp, xhdpi), driven by a remote: no camera, no browser, no VPN
 * settings screen, and the D-pad focus ring where the remote starts.
 *
 *   android/scripts/gradle.sh :app:testDebugUnitTest --tests 'dev.tailbond.app.ux.TvScreens' \
 *       -Ptailbond.uxOut=$PWD/out/ux/tv/android
 */
@RunWith(AndroidJUnit4::class)
@Config(sdk = [35], qualifiers = TV)
class TvScreens {
    @get:Rule val compose = createAndroidComposeRule<ComponentActivity>()
    private val out: String? = System.getProperty("tailbond.uxOut")

    @Before fun requireOut() = assumeTrue("set -Ptailbond.uxOut to render", !out.isNullOrBlank())

    private val home = ServerProfile("h", UxFixtures.profile)
    private val tv = UiState(
        profile = home.payload,
        servers = listOf(home),
        settings = AppSettings(onboarded = true, cellularCapMb = 5000),
        cellularBytes = 1_230_000_000,
        peers = PeersUi(UxFixtures.peers),
        tailscaleInstalled = true,
        live = UxFixtures.connectedBoth.copy(tunnelIp = "10.77.0.3"),
        hasCamera = false, hasBrowser = false, hasVpnSettings = false,
    )
    private val unpaired = UiState(settings = AppSettings(onboarded = true), hasCamera = false, hasBrowser = false, hasVpnSettings = false)

    /** [name].png: the screen as the TV shows it after [keys] on the remote (none: where it starts). */
    @OptIn(ExperimentalRoborazziApi::class)
    private fun shot(name: String, light: Boolean = false, keys: Remote.() -> Unit = {}, content: @Composable () -> Unit) {
        if (light) RuntimeEnvironment.setQualifiers(TV.replace("-night-", "-notnight-"))
        val remote = Remote(compose)
        compose.setContent { TailbondTheme { content() } }
        compose.waitForIdle()
        remote.keys()
        compose.mainClock.advanceTimeBy(1_000)
        compose.waitForIdle()
        captureScreenRoboImage(File(out, "$name.png").also { it.parentFile?.mkdirs() })
    }

    private fun app(name: String, state: UiState, route: String = Routes.HOME, light: Boolean = false, keys: Remote.() -> Unit = {}) =
        shot(name, light, keys) { TailbondApp(state, NoActions, route) }

    @Test fun banner() {
        compose.setContent { Image(painterResource(R.drawable.tv_banner), "tailbond", Modifier.size(160.dp, 90.dp).testTag("banner")) }
        compose.onNodeWithTag("banner").captureRoboImage(File(out, "tv-00-banner.png"))
    }

    @Test fun onboarding() = app("tv-01-onboarding", UiState(hasCamera = false, hasBrowser = false))
    @Test fun pairing() = app("tv-02-pairing-no-camera", unpaired)
    @Test fun pairingLink() = app("tv-02b-pairing-link", unpaired) { down(4) }
    @Test fun pairingWaiting() = app("tv-02c-pairing-waiting", unpaired.copy(ask = AskState.Waiting("homebox", "living-room-tv", "living-room-tv · sam@example.com · 100.101.8.9")))
    @Test fun confirm() = app("tv-03-confirm-pairing", unpaired.copy(pending = UxFixtures.profile))
    @Test fun handoff() = app("tv-04-handoff", tv.copy(live = UxFixtures.disconnected, handoff = Handoff.FIRST_PERMISSION))

    @Test fun homeConnected() = app("tv-10-home-connected", tv)
    @Test fun homeLight() = app("tv-10b-home-connected-light", tv, light = true)
    @Test fun homeMachines() = app("tv-11-home-machines", tv) { down(6) }
    @Test fun homeDisconnected() = app("tv-12-home-disconnected", tv.copy(live = UxFixtures.disconnected))
    @Test fun connections() = app("tv-20-connections", tv.copy(settings = tv.settings.copy(mode = Mode.SPEED)), Routes.CONNECTIONS) { down() }
    @Test fun settings() = app("tv-30-settings", tv, Routes.SETTINGS) { down() }
    @Test fun settingsAlwaysOn() = app("tv-30b-settings-always-on", tv, Routes.SETTINGS) { down(8) }
    @Test fun share() = app("tv-40-sharing", tv.copy(hub = HubLive(), share = ShareLive(label = "living-room-tv's mobile data")), Routes.SHARE) { down() }
    @Test fun shareNearby() = app("tv-41-sharing-nearby-phones", tv.copy(settings = tv.settings.copy(useShares = true)), Routes.SHARE) { down(16) }
    @Test fun report() = app("tv-50-report", tv, Routes.REPORT) { down() }
    @Test fun updates() = app(
        "tv-60-updates-no-browser",
        tv.copy(update = tv.update.copy(status = UpdateStatus(current = "0.2.0", latest = "0.3.0", apkUrl = "https://github.com/kaovilai/tailbond/releases/download/v0.3.0/tailbond-android.apk", lastCheckMs = 1))),
        Routes.UPDATES,
    ) { down() }
}
