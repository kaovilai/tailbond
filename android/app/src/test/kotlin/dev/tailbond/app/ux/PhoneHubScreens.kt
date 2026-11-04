package dev.tailbond.app.ux

import androidx.test.ext.junit.runners.AndroidJUnit4
import dev.tailbond.app.model.AppSettings
import dev.tailbond.app.model.ServerProfile
import dev.tailbond.app.share.HubLive
import dev.tailbond.app.share.HubShare
import dev.tailbond.app.share.HubStatus
import dev.tailbond.app.share.ProxyServer
import dev.tailbond.app.share.ShareLive
import dev.tailbond.app.ui.NoActions
import dev.tailbond.app.ui.PeersUi
import dev.tailbond.app.ui.Routes
import dev.tailbond.app.ui.TailbondApp
import dev.tailbond.app.ui.UiState
import org.junit.Test
import org.junit.runner.RunWith
import org.robolectric.annotation.Config

/**
 * Phone hub (docs/phone-hub.md): "Share bonded connection" and nearby
 * phones' shared data, on the Share screen:
 *
 *   android/scripts/gradle.sh :app:testDebugUnitTest --tests 'dev.tailbond.app.ux.PhoneHubScreens' \
 *       -Ptailbond.uxOut=$PWD/out/ux/phone-hub/android
 */
@RunWith(AndroidJUnit4::class)
@Config(sdk = [35])
class PhoneHubScreens : UxRender() {
    private val home = ServerProfile("h", UxFixtures.profile)

    private fun state(hub: HubLive, shares: List<HubShare> = emptyList(), s: AppSettings = AppSettings(onboarded = true)) = UiState(
        profile = home.payload, servers = listOf(home), settings = s,
        live = UxFixtures.connectedBoth.copy(tunnelIp = "10.77.0.3"), peers = PeersUi(UxFixtures.peers),
        tailscaleInstalled = true, sdkInt = 35, share = ShareLive(label = "Mobile data from pixel-9"), hub = hub, hubShares = shares,
    )

    private fun app(name: String, st: UiState, dark: Boolean = false) = shot(name, dark, 2200) { TailbondApp(st, NoActions, Routes.SHARE) }

    private val mac = ProxyServer.Client("192.168.43.112", 4, 48_200_000, 912_000_000, 0)
    private val sam = HubShare("and-sam", "Mobile data from sam-pixel", "192.168.1.37:47077", "mdns", "ready", "sh3a91c0",
        310_000_000, 1_450_000_000, "", 71)

    @Test fun off() = app("hub-01-off", state(HubLive(label = "pixel-8")))
    @Test fun ready() = app("hub-02-ready", state(HubLive(HubStatus.READY, label = "pixel-8")))
    @Test fun sharing() = app("hub-03-sharing", state(HubLive(HubStatus.SHARING, listOf(mac), 48_200_000, 912_000_000, "pixel-8"), listOf(sam)))
    @Test fun sharingDark() = app("hub-04-sharing-dark", state(HubLive(HubStatus.SHARING, listOf(mac), 48_200_000, 912_000_000, "pixel-8"), listOf(sam)), dark = true)
    @Test fun noTunnel() = app("hub-05-no-tunnel", state(HubLive(HubStatus.NO_TUNNEL, label = "pixel-8")))
}
