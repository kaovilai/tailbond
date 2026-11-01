package dev.tailbond.app.ux

import androidx.test.ext.junit.runners.AndroidJUnit4
import dev.tailbond.app.failover.FailoverReason
import dev.tailbond.app.failover.FailoverView
import dev.tailbond.app.failover.ServerState
import dev.tailbond.app.failover.ServerStatus
import dev.tailbond.app.failover.Switch
import dev.tailbond.app.failover.SwitchKind
import dev.tailbond.app.model.AddCheck
import dev.tailbond.app.model.AppSettings
import dev.tailbond.app.model.Cidr
import dev.tailbond.app.model.ServerProfile
import dev.tailbond.app.model.WebsiteRule
import dev.tailbond.app.pairing.PairingPayload
import dev.tailbond.app.service.ConnState
import dev.tailbond.app.service.ErrorKind
import dev.tailbond.app.service.Live
import dev.tailbond.app.ui.NoActions
import dev.tailbond.app.ui.PairTarget
import dev.tailbond.app.ui.PeersUi
import dev.tailbond.app.ui.Routes
import dev.tailbond.app.ui.TailbondApp
import dev.tailbond.app.ui.UiState
import org.junit.Test
import org.junit.runner.RunWith
import org.robolectric.annotation.Config
import java.time.LocalDate
import java.time.ZoneId

/**
 * Multiple servers with failover, and websites that skip the tunnel:
 *
 *   android/scripts/gradle.sh :app:testDebugUnitTest --tests 'dev.tailbond.app.ux.BypassFailoverScreens' \
 *       -Ptailbond.uxOut=$PWD/out/ux/bypass-failover/android
 */
@RunWith(AndroidJUnit4::class)
@Config(sdk = [35])
class BypassFailoverScreens : UxRender() {
    private val vpsPayload = PairingPayload(
        name = "pixel-8", server = "203.0.113.9:8443", sni = "vps.tail1234.ts.net", user = "pixel-8", key = "not-a-real-key",
        dns = listOf("10.83.142.1"), mode = "speed", tailnetRoutes = listOf("100.64.0.0/10", "fd7a:115c:a1e0::/48"),
        serverName = "vps", subnet = "10.83.142.0/24",
    )
    private val home = ServerProfile("h", UxFixtures.profile)
    private val vps = ServerProfile("v", vpsPayload)
    private val settings = AppSettings(
        onboarded = true, cellularCapMb = 5000, excludedApps = setOf("com.netflix.mediaclient"),
        websites = listOf(
            WebsiteRule("mybank.com", true, "Bank"),
            WebsiteRule("netflix.com", true, "Streaming"),
            WebsiteRule("nflxvideo.net", true, "Streaming"),
            WebsiteRule("tax.example.gov", false, ""),
        ),
    )
    private val now = 1_000_000L

    private fun view(
        active: String,
        homeState: ServerState,
        vpsState: ServerState,
        reason: FailoverReason = FailoverReason.NONE,
        failbackIn: Long? = null,
        last: Switch? = null,
        since: Long? = null,
    ) = FailoverView(
        activeId = active,
        servers = listOf(ServerStatus("h", homeState, now - 5_000), ServerStatus("v", vpsState, now - 4_000)),
        reason = reason,
        failbackTo = if (failbackIn != null) "h" else null,
        failbackInMs = failbackIn,
        lastSwitch = last,
        nowMs = now,
        sinceWallMs = since,
    )

    /** Today at [h]:[m] in the renderer's time zone, so the PNGs say "since 14:02". */
    private fun at(h: Int, m: Int) = LocalDate.of(2026, 11, 1).atTime(h, m).atZone(ZoneId.systemDefault()).toInstant().toEpochMilli()
    private val toVps = Switch("v", "h", SwitchKind.FAILOVER, now - 25 * 60_000L)

    private fun state(live: Live, f: FailoverView?, servers: List<ServerProfile> = listOf(home, vps), s: AppSettings = settings) = UiState(
        profile = servers.first().payload,
        servers = servers,
        settings = s,
        live = live.copy(failover = f),
        cellularBytes = 1_230_000_000,
        peers = PeersUi(UxFixtures.peers),
        tailscaleInstalled = true,
    )

    private val onVps = UxFixtures.connectedBoth.copy(tunnelIp = "10.83.142.2")
    private val primaryStandby = view("h", ServerState.ACTIVE, ServerState.STANDBY)
    private val usingVps = view("v", ServerState.UNREACHABLE, ServerState.ACTIVE, FailoverReason.PRIMARY_DOWN, last = toVps, since = at(14, 2))
    private val failingBack = view("v", ServerState.STANDBY, ServerState.ACTIVE, FailoverReason.PRIMARY_DOWN, failbackIn = 80_000, last = toVps, since = at(14, 2))
    private val backOn = view("h", ServerState.ACTIVE, ServerState.STANDBY, last = Switch("h", "v", SwitchKind.FAILBACK, now - 40_000), since = at(14, 20))
    private val pickedVps = view("v", ServerState.STANDBY, ServerState.ACTIVE, FailoverReason.MANUAL, last = Switch("v", "h", SwitchKind.MANUAL, now - 60_000), since = at(15, 10))

    private fun app(
        name: String,
        state: UiState,
        dark: Boolean = false,
        tall: Int = PHONE_H,
        route: String = Routes.HOME,
        dialog: String? = null,
    ) = shot(name, dark, tall) { TailbondApp(state, NoActions, route, startDialog = dialog) }

    // home
    @Test fun primaryWithBackup() = app("40-home-on-homebox-vps-on-standby", state(UxFixtures.connectedBoth, primaryStandby), tall = 1150)
    @Test fun primaryWithBackupDark() = app("40-home-on-homebox-vps-on-standby-dark", state(UxFixtures.connectedBoth, primaryStandby), dark = true)
    @Test fun usingBackup() = app("41-home-using-vps-homebox-unreachable", state(onVps, usingVps), tall = 1150)
    @Test fun usingBackupNotification() = app("41b-home-using-vps-first-screen", state(onVps, usingVps))
    @Test fun failingBack() = app("42-home-back-to-homebox-in-1-20", state(onVps, failingBack), tall = 1150)
    @Test fun backOnPrimary() = app("43-home-back-on-homebox", state(UxFixtures.connectedBoth, backOn))
    @Test fun switching() = app(
        "43b-home-switching-to-vps",
        state(Live(state = ConnState.CONNECTING), view("v", ServerState.UNREACHABLE, ServerState.CONNECTING, FailoverReason.PRIMARY_DOWN, last = Switch("v", "h", SwitchKind.FAILOVER, now - 3_000), since = at(14, 2))),
    )
    @Test fun pickedBackup() = app("41c-home-using-vps-you-picked-it", state(onVps, pickedVps))
    @Test fun settingsCountdown() = app("44c-settings-servers-homebox-answering-again", state(onVps, failingBack), tall = 1300, route = Routes.SETTINGS)
    @Test fun allDown() = app(
        "49-home-error-no-server-reachable",
        state(UxFixtures.errorTimeout.copy(errorKind = ErrorKind.UNREACHABLE), view("v", ServerState.UNREACHABLE, ServerState.CONNECTING, FailoverReason.PRIMARY_DOWN)),
    )

    // settings: server list, policy, add another
    @Test fun settingsServers() = app("44-settings-servers", state(onVps, usingVps), tall = 1900, route = Routes.SETTINGS)
    @Test fun settingsServersDark() = app("44-settings-servers-dark", state(onVps, usingVps), dark = true, route = Routes.SETTINGS)
    @Test fun settingsOneServer() = app(
        "44b-settings-one-server-add-backup",
        state(UxFixtures.connectedBoth, null, servers = listOf(home)), tall = 1500, route = Routes.SETTINGS,
    )
    @Test fun addServer() = app(
        "45-add-another-server",
        state(UxFixtures.connectedBoth, primaryStandby, servers = listOf(home)).copy(pairTarget = PairTarget.Add),
        tall = 1500, route = Routes.PAIR,
    )
    @Test fun addServerConfirm() = app(
        "45b-add-another-server-confirm",
        state(UxFixtures.connectedBoth, null, servers = listOf(home))
            .copy(pairTarget = PairTarget.Add, pending = vpsPayload, pendingTarget = PairTarget.Add),
        route = Routes.PAIR,
    )
    @Test fun addServerClash() = app(
        "45c-add-another-server-same-addresses",
        state(UxFixtures.connectedBoth, null, servers = listOf(home)).copy(
            pairTarget = PairTarget.Add,
            serverProblem = "vps" to AddCheck.SubnetClash("homebox", Cidr("10.77.0.0", 24), Cidr("10.77.0.0", 24)),
        ),
        route = Routes.PAIR,
    )

    // websites
    @Test fun websites() = app("46-websites-skip-tailbond", state(UxFixtures.connectedBoth, primaryStandby), tall = 1100, route = Routes.WEBSITES)
    @Test fun websitesDark() = app("46-websites-skip-tailbond-dark", state(UxFixtures.connectedBoth, primaryStandby), dark = true, route = Routes.WEBSITES)
    @Test fun websitesOldAndroid() = app(
        "46b-websites-needs-android-13",
        state(UxFixtures.connectedBoth, primaryStandby).copy(sdkInt = 32), route = Routes.WEBSITES,
    )
    @Test fun websitesEmpty() = app(
        "46c-websites-none-yet",
        state(UxFixtures.connectedBoth, primaryStandby, s = settings.copy(websites = emptyList())), route = Routes.WEBSITES,
    )
    @Test fun addWebsite() = app(
        "47-add-website-dialog",
        state(UxFixtures.connectedBoth, primaryStandby, s = settings.copy(websites = emptyList())),
        route = Routes.WEBSITES, dialog = "add-website",
    )
    @Test fun addWebsiteBanking() = app(
        "47b-add-website-preset-banking",
        state(UxFixtures.connectedBoth, primaryStandby, s = settings.copy(websites = emptyList())),
        route = Routes.WEBSITES, dialog = "add-website:Banking",
    )
    @Test fun addWebsiteStreaming() = app(
        "47c-add-website-preset-streaming",
        state(UxFixtures.connectedBoth, primaryStandby, s = settings.copy(websites = emptyList())),
        route = Routes.WEBSITES, dialog = "add-website:Streaming (where you are)",
    )
    @Test fun addWebsiteCalling() = app(
        "47d-add-website-preset-wifi-calling",
        state(UxFixtures.connectedBoth, primaryStandby, s = settings.copy(websites = emptyList())),
        route = Routes.WEBSITES, dialog = "add-website:Wi-Fi calling",
    )
    @Test fun settingsWebsitesRowOldAndroid() = app(
        "48-settings-skip-rows-android-12",
        state(UxFixtures.connectedBoth, null, servers = listOf(home)).copy(sdkInt = 32), tall = 1500, route = Routes.SETTINGS,
    )
}
