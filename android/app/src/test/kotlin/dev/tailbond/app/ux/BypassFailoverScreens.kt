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
import dev.tailbond.app.data.PeerNode
import dev.tailbond.app.model.SwitchLog
import dev.tailbond.app.model.SwitchRecord
import dev.tailbond.app.model.WebsiteRule
import dev.tailbond.app.model.Words
import dev.tailbond.app.ui.notificationText
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
import androidx.compose.foundation.background
import androidx.compose.foundation.layout.fillMaxSize
import androidx.compose.foundation.layout.fillMaxWidth
import androidx.compose.foundation.layout.padding
import androidx.compose.ui.unit.dp
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
        dns = listOf("10.78.0.1"), mode = "speed", tailnetRoutes = listOf("100.64.0.0/10", "fd7a:115c:a1e0::/48"),
        serverName = "vps", subnet = "10.78.0.0/24",
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
    /** The tailnet list from homebox, which also has vps (the add-server page suggests it). */
    private val peersWithVps = UxFixtures.peers.copy(
        peers = (UxFixtures.peers.peers + PeerNode("vps", "vps.tail1234.ts.net", listOf("100.101.9.9"), "linux", true)).sortedBy { it.hostname },
    )

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
    /** Today at [h]:[m] (so the history reads "14:02", like the home card), or [daysAgo] days earlier. */
    private fun at(h: Int, m: Int, daysAgo: Long = 0) =
        LocalDate.now().minusDays(daysAgo).atTime(h, m).atZone(ZoneId.systemDefault()).toInstant().toEpochMilli()

    // one story: yesterday's blip, homebox down at 14:02, back at 14:20, vps picked by hand at 15:10
    private val yesterday = listOf(
        SwitchRecord(at(9, 40, 1), "vps", "homebox", SwitchLog.BACK),
        SwitchRecord(at(9, 15, 1), "homebox", "vps", SwitchLog.DOWN),
    )
    private val down1402 = SwitchRecord(at(14, 2), "homebox", "vps", SwitchLog.DOWN)
    private val back1420 = SwitchRecord(at(14, 20), "vps", "homebox", SwitchLog.BACK)
    private val picked1510 = SwitchRecord(at(15, 10), "homebox", "vps", SwitchLog.PICKED)
    private val onVpsLog = listOf(down1402) + yesterday
    private val backLog = listOf(back1420, down1402) + yesterday
    private val pickedLog = listOf(picked1510, back1420, down1402) + yesterday
    private val toVps = Switch("v", "h", SwitchKind.FAILOVER, now - 25 * 60_000L)

    private fun state(
        live: Live,
        f: FailoverView?,
        servers: List<ServerProfile> = listOf(home, vps),
        s: AppSettings = settings,
        log: List<SwitchRecord> = onVpsLog,
    ) = UiState(
        switches = if (servers.size > 1) log else emptyList(),
        profile = servers.first().payload,
        servers = servers,
        settings = s,
        live = live.copy(failover = f),
        cellularBytes = 1_230_000_000,
        peers = PeersUi(peersWithVps),
        tailscaleInstalled = true,
    )

    // the phone's own tunnel addresses (the travel router in the GL renders has .2)
    private val onHome = UxFixtures.connectedBoth.copy(tunnelIp = "10.77.0.3")
    private val onVps = UxFixtures.connectedBoth.copy(tunnelIp = "10.78.0.3")
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
    @Test fun primaryWithBackup() = app("40-home-on-homebox-vps-on-standby", state(onHome, primaryStandby), tall = 1150)
    @Test fun primaryWithBackupDark() = app("40-home-on-homebox-vps-on-standby-dark", state(onHome, primaryStandby), dark = true)
    @Test fun usingBackup() = app("41-home-using-vps-homebox-unreachable", state(onVps, usingVps), tall = 1150)
    @Test fun usingBackupNotification() = app("41b-home-using-vps-first-screen", state(onVps, usingVps))
    @Test fun failingBack() = app("42-home-back-to-homebox-in-1-20", state(onVps, failingBack), tall = 1150)
    @Test fun backOnPrimary() = app("43-home-back-on-homebox", state(onHome, backOn, log = backLog))
    @Test fun switching() = app(
        "43b-home-switching-to-vps",
        state(Live(state = ConnState.CONNECTING), view("v", ServerState.UNREACHABLE, ServerState.CONNECTING, FailoverReason.PRIMARY_DOWN, last = Switch("v", "h", SwitchKind.FAILOVER, now - 3_000), since = at(14, 2))),
    )
    @Test fun pickedBackup() = app("41c-home-using-vps-you-picked-it", state(onVps, pickedVps, log = pickedLog))
    @Test fun settingsCountdown() = app("44c-settings-servers-homebox-answering-again", state(onVps, failingBack), tall = 1300, route = Routes.SETTINGS)
    @Test fun settingsPaused() = app(
        "44d-settings-servers-paused-you-picked-vps", state(onVps, pickedVps, log = pickedLog), tall = 1500, route = Routes.SETTINGS,
    )

    // the tunnel notification, drawn like Android's shade with the service's own text
    @Test fun notificationOnVps() = shot("41d-notification-using-vps") {
        NotificationMock(
            listOf(
                "Connected through vps" to notificationText(
                    Words.serverNoteShort(usingVps, mapOf("h" to "homebox", "v" to "vps")),
                    "Using both Wi-Fi and mobile data", 61_100_000, 7_000_000,
                ),
                "Connected through vps" to notificationText(
                    Words.serverNoteShort(failingBack, mapOf("h" to "homebox", "v" to "vps")),
                    "Using both Wi-Fi and mobile data", 58_300_000, 6_400_000,
                ),
                "Connected through homebox" to notificationText(
                    Words.serverNoteShort(backOn, mapOf("h" to "homebox", "v" to "vps")),
                    "Using both Wi-Fi and mobile data", 63_900_000, 7_800_000,
                ),
            ),
        )
    }
    @Test fun allDown() = app(
        "49-home-error-no-server-reachable",
        state(UxFixtures.errorTimeout.copy(errorKind = ErrorKind.UNREACHABLE), view("v", ServerState.UNREACHABLE, ServerState.CONNECTING, FailoverReason.PRIMARY_DOWN)),
    )

    // settings: server list, policy, add another
    @Test fun settingsServers() = app("44-settings-servers", state(onVps, usingVps), tall = 1900, route = Routes.SETTINGS)
    @Test fun settingsServersDark() = app("44-settings-servers-dark", state(onVps, usingVps), dark = true, route = Routes.SETTINGS)
    @Test fun settingsOneServer() = app(
        "44b-settings-one-server-add-backup",
        state(onHome, null, servers = listOf(home)), tall = 1500, route = Routes.SETTINGS,
    )
    @Test fun addServer() = app(
        "45-add-another-server",
        state(onHome, primaryStandby, servers = listOf(home)).copy(pairTarget = PairTarget.Add, serverSuggestions = listOf("vps")),
        tall = 1500, route = Routes.PAIR,
    )
    @Test fun addServerConfirm() = app(
        "45b-add-another-server-confirm",
        state(onHome, null, servers = listOf(home))
            .copy(pairTarget = PairTarget.Add, pending = vpsPayload, pendingTarget = PairTarget.Add),
        route = Routes.PAIR,
    )
    @Test fun addServerClash() = app(
        "45c-add-another-server-same-addresses",
        state(onHome, null, servers = listOf(home)).copy(
            pairTarget = PairTarget.Add,
            serverProblem = "vps" to AddCheck.SubnetClash("homebox", Cidr("10.77.0.0", 24), Cidr("10.77.0.0", 24)),
        ),
        route = Routes.PAIR,
    )

    // websites
    @Test fun websites() = app("46-websites-skip-tailbond", state(onHome, primaryStandby), tall = 1100, route = Routes.WEBSITES)
    @Test fun websitesDark() = app("46-websites-skip-tailbond-dark", state(onHome, primaryStandby), dark = true, route = Routes.WEBSITES)
    @Test fun websitesOldAndroid() = app(
        "46b-websites-needs-android-13",
        state(onHome, primaryStandby).copy(sdkInt = 32), route = Routes.WEBSITES,
    )
    @Test fun websitesEmpty() = app(
        "46c-websites-none-yet",
        state(onHome, primaryStandby, s = settings.copy(websites = emptyList())), route = Routes.WEBSITES,
    )
    @Test fun addWebsite() = app(
        "47-add-website-dialog",
        state(onHome, primaryStandby, s = settings.copy(websites = emptyList())),
        route = Routes.WEBSITES, dialog = "add-website",
    )
    @Test fun addWebsiteBanking() = app(
        "47b-add-website-preset-banking",
        state(onHome, primaryStandby, s = settings.copy(websites = emptyList())),
        route = Routes.WEBSITES, dialog = "add-website:Banking",
    )
    @Test fun addWebsiteStreaming() = app(
        "47c-add-website-preset-streaming",
        state(onHome, primaryStandby, s = settings.copy(websites = emptyList())),
        route = Routes.WEBSITES, dialog = "add-website:Streaming (where you are)",
    )
    @Test fun addWebsiteCalling() = app(
        "47d-add-website-preset-wifi-calling",
        state(onHome, primaryStandby, s = settings.copy(websites = emptyList())),
        route = Routes.WEBSITES, dialog = "add-website:Wi-Fi calling",
    )
    @Test fun settingsWebsitesRowOldAndroid() = app(
        "48-settings-skip-rows-android-12",
        state(onHome, null, servers = listOf(home)).copy(sdkInt = 32), tall = 1500, route = Routes.SETTINGS,
    )
}

/** A notification as Android's shade shows it (renders only; the text comes from the service's own builder). */
@androidx.compose.runtime.Composable
private fun NotificationMock(items: List<Pair<String, String>>) {
    androidx.compose.foundation.layout.Column(
        androidx.compose.ui.Modifier.fillMaxSize().background(androidx.compose.ui.graphics.Color(0xFF1F1F23)).padding(12.dp),
        verticalArrangement = androidx.compose.foundation.layout.Arrangement.spacedBy(10.dp),
    ) {
        androidx.compose.material3.Text("14:30  ·  notifications, three moments", color = androidx.compose.ui.graphics.Color(0xFFBBBBBB), style = androidx.compose.material3.MaterialTheme.typography.labelMedium)
        items.forEach { (title, text) ->
            androidx.compose.material3.Card(
                colors = androidx.compose.material3.CardDefaults.cardColors(containerColor = androidx.compose.ui.graphics.Color(0xFF2E2E33)),
                modifier = androidx.compose.ui.Modifier.fillMaxWidth(),
            ) {
                androidx.compose.foundation.layout.Column(androidx.compose.ui.Modifier.padding(14.dp)) {
                    androidx.compose.material3.Text("tailbond", color = androidx.compose.ui.graphics.Color(0xFFAAB4FF), style = androidx.compose.material3.MaterialTheme.typography.labelMedium)
                    androidx.compose.material3.Text(title, color = androidx.compose.ui.graphics.Color.White, style = androidx.compose.material3.MaterialTheme.typography.titleSmall)
                    androidx.compose.material3.Text(text, color = androidx.compose.ui.graphics.Color(0xFFDDDDDD), style = androidx.compose.material3.MaterialTheme.typography.bodyMedium)
                    androidx.compose.material3.Text("Disconnect", color = androidx.compose.ui.graphics.Color(0xFFAAB4FF), style = androidx.compose.material3.MaterialTheme.typography.labelLarge, modifier = androidx.compose.ui.Modifier.padding(top = 8.dp))
                }
            }
        }
    }
}
