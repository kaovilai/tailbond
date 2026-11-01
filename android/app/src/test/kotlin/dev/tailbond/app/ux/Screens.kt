package dev.tailbond.app.ux

import androidx.test.ext.junit.runners.AndroidJUnit4
import dev.tailbond.app.data.AskState
import dev.tailbond.app.data.PairRequestClient
import dev.tailbond.app.model.AppSettings
import dev.tailbond.app.model.Mode
import dev.tailbond.app.service.Live
import dev.tailbond.app.ui.AppEntry
import dev.tailbond.app.ui.Handoff
import dev.tailbond.app.ui.NoActions
import dev.tailbond.app.ui.PeersUi
import dev.tailbond.app.ui.Routes
import dev.tailbond.app.ui.TailbondApp
import dev.tailbond.app.ui.UiState
import org.junit.Test
import org.junit.runner.RunWith
import org.robolectric.annotation.Config

/**
 * Every screen and state of the redesigned app, from fake [UiState]s.
 *
 *   android/scripts/gradle.sh :app:testDebugUnitTest --tests 'dev.tailbond.app.ux.*' -Ptailbond.uxOut=$PWD/out/ux/after-1
 */
@RunWith(AndroidJUnit4::class)
@Config(sdk = [35])
class Screens : UxRender() {
    private val settings = AppSettings(onboarded = true, cellularCapMb = 5000, excludedApps = setOf("com.netflix.mediaclient"))
    private val paired = UiState(
        profile = UxFixtures.profile,
        settings = settings,
        cellularBytes = 1_230_000_000,
        peers = PeersUi(UxFixtures.peers),
        tailscaleInstalled = true,
    )

    private fun app(
        name: String,
        state: UiState,
        dark: Boolean = false,
        tall: Int = PHONE_H,
        route: String = Routes.HOME,
        search: String = "",
        snackbar: String? = null,
    ) = shot(name, dark, tall) { TailbondApp(state, NoActions, route, search, snackbar) }

    private fun home(live: Live, s: AppSettings = settings) = paired.copy(live = live, settings = s)

    // first launch and pairing
    @Test fun onboarding() = app("01-onboarding", UiState())
    @Test fun onboardingDark() = app("01-onboarding-dark", UiState(), dark = true)
    @Test fun pairing() = app("02-pairing", UiState(settings = AppSettings(onboarded = true)), tall = 1100)
    @Test fun pairingError() = app(
        "02-pairing-paste-error",
        UiState(settings = AppSettings(onboarded = true), pairError = "pairing data is not base64url"),
        tall = 1100, search = "tailbond://pair?d=eyJ2IjoxLCJuYW1lIjoicGl4ZWwt",
    )
    // "Ask to connect" over Tailscale
    @Test fun askWaiting() = app("02b-pairing-ask-waiting", UiState(settings = AppSettings(onboarded = true), ask = AskState.Waiting("homebox", "pixel-8")), tall = 1100)
    @Test fun askNotFound() = app("02c-pairing-ask-not-found", UiState(settings = AppSettings(onboarded = true), ask = AskState.Failed("homebox", PairRequestClient.Failure.NOT_FOUND)), tall = 1100)
    @Test fun askDeclined() = app("02d-pairing-ask-declined", UiState(settings = AppSettings(onboarded = true), ask = AskState.Declined("homebox")), tall = 1100)
    @Test fun askApproved() = app("02e-pairing-ask-approved-confirm", UiState(settings = AppSettings(onboarded = true), pending = UxFixtures.profile), route = Routes.PAIR)
    @Test fun deepLinkConfirm() = app("03-deeplink-confirm", UiState(settings = AppSettings(onboarded = true), pending = UxFixtures.profile))
    @Test fun handoff() = app("04-handoff-other-vpn", home(UxFixtures.disconnected).copy(handoff = Handoff.OTHER_VPN_ACTIVE))
    @Test fun handoffDark() = app("04-handoff-other-vpn-dark", home(UxFixtures.disconnected).copy(handoff = Handoff.OTHER_VPN_ACTIVE), dark = true)
    @Test fun handoffFirst() = app("05-handoff-first-permission", home(UxFixtures.disconnected).copy(handoff = Handoff.FIRST_PERMISSION))

    // home
    @Test fun connected() = app("10-home-connected", home(UxFixtures.connectedBoth))
    @Test fun connectedDark() = app("10-home-connected-dark", home(UxFixtures.connectedBoth), dark = true)
    @Test fun connectedFull() = app("10-home-connected-full", home(UxFixtures.connectedBoth), tall = 1500)
    @Test fun wifiSignIn() = app("16c-home-wifi-needs-sign-in", home(UxFixtures.cellOnly).copy(wifiSignIn = true), tall = 1500)
    @Test fun connecting() = app("11-home-connecting", home(UxFixtures.connecting))
    @Test fun disconnected() = app("12-home-disconnected", home(UxFixtures.disconnected), tall = 1500)
    @Test fun disconnectedDark() = app("12-home-disconnected-dark", home(UxFixtures.disconnected), dark = true)
    @Test fun handedBack() = app(
        "12b-home-switched-back-to-tailscale",
        home(UxFixtures.disconnected, settings.copy(handedToTailscale = true)),
    )
    @Test fun errUnreachable() = app("13-error-unreachable", home(UxFixtures.errorUnreachable))
    @Test fun errCert() = app("14-error-cert", home(UxFixtures.errorCert))
    @Test fun onePathFailed() = app("15-one-path-failed", home(UxFixtures.onePathFailed))
    @Test fun failoverStandby() = app("16-save-data-standby", home(UxFixtures.failoverStandby, settings.copy(mode = Mode.FAILOVER)))
    @Test fun failoverTakeover() = app("17-save-data-takeover", home(UxFixtures.failoverTakeover, settings.copy(mode = Mode.FAILOVER)))

    // machines
    @Test fun machinesSearch() = app(
        "18-machines-search-copied", home(UxFixtures.connectedBoth), tall = 1500, search = "nas",
        snackbar = "Copied 100.101.2.2",
    )
    @Test fun machineDetail() = app("19-machine-detail", home(UxFixtures.connectedBoth), route = Routes.MACHINE + "nas", snackbar = "Copied nas.tail1234.ts.net")
    @Test fun machineDetailDark() = app("19-machine-detail-dark", home(UxFixtures.connectedBoth), dark = true, route = Routes.MACHINE + "nas")

    // connections (bonding)
    @Test fun connections() = app("20-connections", home(UxFixtures.connectedBoth), tall = 2000, route = Routes.CONNECTIONS)
    @Test fun connectionsDark() = app("20-connections-dark", home(UxFixtures.connectedBoth), dark = true, route = Routes.CONNECTIONS)
    @Test fun connectionsSaving() = app(
        "21-connections-save-data", home(UxFixtures.failoverStandby, settings.copy(mode = Mode.FAILOVER)),
        tall = 2000, route = Routes.CONNECTIONS,
    )

    // settings
    @Test fun settings() = app("30-settings", home(UxFixtures.connectedBoth), tall = 1500, route = Routes.SETTINGS)
    @Test fun settingsDark() = app("30-settings-dark", home(UxFixtures.connectedBoth), dark = true, route = Routes.SETTINGS)
    @Test fun apps() = app(
        "31-apps-skip-tailbond",
        home(UxFixtures.connectedBoth).copy(apps = APPS),
        route = Routes.APPS,
    )

    companion object {
        val APPS = listOf(
            AppEntry("com.android.chrome", "Chrome"),
            AppEntry("com.google.android.apps.photos", "Google Photos"),
            AppEntry("com.netflix.mediaclient", "Netflix"),
            AppEntry("com.plexapp.android", "Plex"),
            AppEntry("com.spotify.music", "Spotify"),
            AppEntry("com.whatsapp", "WhatsApp"),
            AppEntry("org.videolan.vlc", "VLC"),
        )
    }
}
