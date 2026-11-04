package dev.tailbond.app.ux

import androidx.test.ext.junit.runners.AndroidJUnit4
import dev.tailbond.app.model.AppSettings
import dev.tailbond.app.model.ServerProfile
import dev.tailbond.app.share.ShareLive
import dev.tailbond.app.share.ShareRelay
import dev.tailbond.app.share.ShareStatus
import dev.tailbond.app.ui.NoActions
import dev.tailbond.app.ui.PeersUi
import dev.tailbond.app.ui.Routes
import dev.tailbond.app.ui.TailbondApp
import dev.tailbond.app.ui.UiState
import org.junit.Test
import org.junit.runner.RunWith
import org.robolectric.annotation.Config

/**
 * Pair & Share, the phone that shares:
 *
 *   android/scripts/gradle.sh :app:testDebugUnitTest --tests 'dev.tailbond.app.ux.ShareScreens' \
 *       -Ptailbond.uxOut=$PWD/out/ux/share/android
 */
@RunWith(AndroidJUnit4::class)
@Config(sdk = [35])
class ShareScreens : UxRender() {
    private val home = ServerProfile("h", UxFixtures.profile)

    private fun state(share: ShareLive, s: AppSettings = AppSettings(onboarded = true)) = UiState(
        profile = home.payload, servers = listOf(home), settings = s,
        live = UxFixtures.connectedBoth.copy(tunnelIp = "10.77.0.3"), peers = PeersUi(UxFixtures.peers),
        tailscaleInstalled = true, sdkInt = 35, share = share,
    )

    private fun app(name: String, st: UiState, route: String, dark: Boolean = false, tall: Int = PHONE_H) =
        shot(name, dark, tall) { TailbondApp(st, NoActions, route) }

    private val slate = ShareRelay.HubUse("slate7", "192.168.8.1:51234", 84_000_000, 38_000_000, 0)

    @Test fun settingsRow() = app("share-01-settings-row", state(ShareLive()), Routes.SETTINGS, tall = 2400)
    @Test fun off() = app("share-02-off", state(ShareLive(label = "Pixel 8's mobile data")), Routes.SHARE, tall = 1300)
    @Test fun waiting() = app("share-03-waiting", state(ShareLive(ShareStatus.WAITING, usedBytes = 0, battery = 76)), Routes.SHARE, tall = 1300)
    @Test fun sharing() = app(
        "share-04-sharing", state(ShareLive(ShareStatus.SHARING, hubs = listOf(slate), usedBytes = 1_240_000_000, battery = 64)), Routes.SHARE, tall = 1300,
    )
    @Test fun sharingDark() = app(
        "share-05-sharing-dark", state(ShareLive(ShareStatus.SHARING, hubs = listOf(slate), usedBytes = 1_240_000_000, battery = 64)), Routes.SHARE, dark = true, tall = 1300,
    )
    @Test fun lowBattery() = app(
        "share-06-low-battery",
        state(ShareLive(ShareStatus.BATTERY, usedBytes = 2_100_000_000, battery = 18), AppSettings(onboarded = true, shareMinBattery = 20)),
        Routes.SHARE, tall = 1300,
    )
    @Test fun capAndAllowed() = app(
        "share-07-cap-allowed",
        state(ShareLive(ShareStatus.CAP, usedBytes = 2_000_000_000), AppSettings(onboarded = true, shareCapMb = 2_000, shareAllowedHubs = listOf("slate7"))),
        Routes.SHARE, tall = 1300,
    )
    @Test fun noWifi() = app("share-08-no-wifi", state(ShareLive(ShareStatus.NO_WIFI)), Routes.SHARE, tall = 1300)
}
