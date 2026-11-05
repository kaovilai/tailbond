package dev.tailbond.app.tv

import androidx.activity.ComponentActivity
import androidx.compose.ui.semantics.SemanticsProperties
import androidx.compose.ui.semantics.getOrNull
import androidx.compose.ui.state.ToggleableState
import androidx.compose.ui.test.SemanticsMatcher
import androidx.compose.ui.test.SemanticsNodeInteraction
import androidx.compose.ui.test.assertIsDisplayed
import androidx.compose.ui.test.hasAnySibling
import androidx.compose.ui.test.hasClickAction
import androidx.compose.ui.test.hasContentDescription
import androidx.compose.ui.test.hasSetTextAction
import androidx.compose.ui.test.hasTestTag
import androidx.compose.ui.test.hasText
import androidx.compose.ui.test.isFocused
import androidx.compose.ui.test.isToggleable
import androidx.compose.ui.test.junit4.createAndroidComposeRule
import androidx.compose.ui.test.performImeAction
import androidx.compose.ui.test.performTextInput
import androidx.test.ext.junit.runners.AndroidJUnit4
import dev.tailbond.app.data.PeerList
import dev.tailbond.app.model.AppSettings
import dev.tailbond.app.model.ServerProfile
import dev.tailbond.app.ui.Handoff
import dev.tailbond.app.ui.PeersUi
import dev.tailbond.app.ui.Routes
import dev.tailbond.app.ui.TailbondApp
import dev.tailbond.app.ui.TailbondTheme
import dev.tailbond.app.ui.UiActions
import dev.tailbond.app.ui.UiState
import dev.tailbond.app.ux.UxFixtures
import org.junit.Assert.assertEquals
import org.junit.Assert.assertTrue
import org.junit.Rule
import org.junit.Test
import org.junit.runner.RunWith
import org.robolectric.annotation.Config

/**
 * Android TV (docs/android.md, "Android TV"): every main screen driven by
 * the remote alone (D-pad, OK, Back) on a 1080p TV with no camera, no
 * browser and no VPN settings screen. Each primary action gets focus, shows
 * the focus ring, and does its job on OK; focus never gets stuck.
 */
@RunWith(AndroidJUnit4::class)
@Config(sdk = [35], qualifiers = TV)
class TvDpadTest {
    @get:Rule val compose = createAndroidComposeRule<ComponentActivity>()
    private val calls = mutableListOf<String>()
    private val actions = object : UiActions {
        override fun setOn(on: Boolean) { calls += "setOn($on)" }
        override fun finishOnboarding() { calls += "finishOnboarding" }
        override fun askToConnect(server: String) { calls += "askToConnect($server)" }
        override fun switchToTailscale() { calls += "switchToTailscale" }
        override fun confirmHandoff() { calls += "confirmHandoff" }
        override fun dismissHandoff() { calls += "dismissHandoff" }
        override fun startHub() { calls += "startHub" }
        override fun startShare() { calls += "startShare" }
        override fun setUseShares(on: Boolean) { calls += "setUseShares($on)" }
        override fun removeServer(id: String) { calls += "removeServer($id)" }
        override fun setShareMinBattery(pct: Int) { calls += "setShareMinBattery($pct)" }
    }
    private val home = ServerProfile("h", UxFixtures.profile)
    private val paired = UiState(
        profile = home.payload,
        servers = listOf(home),
        settings = AppSettings(onboarded = true),
        peers = PeersUi(PeerList(self = UxFixtures.peers.self, peers = UxFixtures.peers.peers.take(2))),
        live = UxFixtures.connectedBoth.copy(tunnelIp = "10.77.0.3"),
        tailscaleInstalled = true,
        hasCamera = false, hasBrowser = false, hasVpnSettings = false,
    )
    private val unpaired = UiState(settings = AppSettings(onboarded = true), hasCamera = false, hasBrowser = false, hasVpnSettings = false)

    private lateinit var remote: Remote

    private fun show(state: UiState, route: String = Routes.HOME) {
        remote = Remote(compose)
        compose.setContent { TailbondTheme { TailbondApp(state, actions, route) } }
        compose.waitForIdle()
    }

    private class Target(val m: SemanticsMatcher)

    private fun node(m: SemanticsMatcher) = Target(m)
    private val Target.one: SemanticsNodeInteraction get() = compose.onNode(m)

    /** Has focus (false while a lazy list hasn't composed it yet). */
    private fun focused(t: Target) = compose.onAllNodes(t.m and isFocused()).fetchSemanticsNodes().isNotEmpty()

    private fun focusedNow(): String = compose.onAllNodes(isFocused()).fetchSemanticsNodes()
        .joinToString { n -> (n.config.getOrNull(SemanticsProperties.Text)?.joinToString() ?: "") + (n.config.getOrNull(SemanticsProperties.ContentDescription)?.joinToString() ?: "") + "@" + n.boundsInWindow.top.toInt() }

    /** DOWN (or [step]) until [t] has focus: it's reachable, and the ring shows on it. */
    private fun moveTo(t: Target, max: Int = 40, step: () -> Unit = { remote.down() }) {
        val path = mutableListOf(focusedNow())
        for (i in 0 until max) {
            if (focused(t)) break
            step()
            path += focusedNow()
        }
        assertTrue("never reached ${t.m.description}; focus went: ${path.joinToString(" > ")}", focused(t))
        remote.assertRingAround(t.one)
    }
    private fun Target.assertIsFocused() = assertTrue("${m.description} not focused; focus is on ${focusedNow()}", focused(this))
    private fun Target.assertExists() = one.assertExists()
    private fun Target.performTextInput(s: String) = one.performTextInput(s)
    private fun Target.performImeAction() = one.performImeAction()
    private fun Target.config() = one.fetchSemanticsNode().config

    private fun toggle(label: String) = node(isToggleable() and hasAnySibling(hasText(label)))

    @Test fun onboardingStartsOnGetStarted() {
        show(UiState(hasCamera = false, hasBrowser = false))
        node(hasText("Get started")).assertIsFocused()
        // the page scrolls it fully into view, the note under it too
        node(hasText("Next you'll type", substring = true)).one.assertIsDisplayed()
        remote.assertRingAround(node(hasText("Get started")).one)
        remote.ok()
        assertEquals(listOf("finishOnboarding"), calls)
    }

    @Test fun pairingWithoutCameraAsksTheServer() {
        show(unpaired)
        // no camera, no browser: no scan button, no "open the server's page"
        assertTrue(compose.onAllNodes(hasText("Scan code", substring = true)).fetchSemanticsNodes().isEmpty())
        assertTrue(compose.onAllNodes(hasText("page", substring = true) and hasText("Open", substring = true)).fetchSemanticsNodes().isEmpty())
        val name = node(hasSetTextAction() and hasText("Home server's name in Tailscale"))
        name.assertIsFocused()
        remote.assertRingAround(name.one)
        // the TV keyboard types, then its Go key asks
        name.performTextInput("homebox")
        name.performImeAction()
        assertEquals(listOf("askToConnect(homebox)"), calls)
        // or down to the button and OK
        moveTo(node(hasText("Ask to connect", substring = true) and hasClickAction()))
        remote.ok()
        assertEquals(listOf("askToConnect(homebox)", "askToConnect(homebox)"), calls)
        // the link field is reachable too, and focus comes back up
        moveTo(node(hasSetTextAction() and hasText("Pairing link")))
        moveTo(name) { remote.up() }
    }

    @Test fun homeStartsOnTheSwitchAndReachesEverything() {
        show(paired.copy(live = UxFixtures.disconnected))
        val main = node(hasTestTag("main-switch"))
        main.assertIsFocused()
        remote.assertRingAround(main.one)
        remote.ok()
        assertEquals(listOf("setOn(true)"), calls)
        // the last row on the page, and no trap on the way
        moveTo(node(hasText("Hand the VPN back to Tailscale")), max = 30)
        remote.ok()
        assertEquals("switchToTailscale", calls.last())
        moveTo(main, max = 30) { remote.up() }
        // up from the switch: the top bar's Settings, then Back comes home
        moveTo(node(hasContentDescription("Settings")), max = 2) { remote.up() }
        remote.ok()
        node(hasText("Settings") and hasClickAction().not()).assertExists()
        remote.back()
        main.assertExists()
    }

    @Test fun settingsReachSharingReportAndBack() {
        show(paired, Routes.SETTINGS)
        // something has focus as soon as the page opens
        assertEquals(1, compose.onAllNodes(isFocused()).fetchSemanticsNodes().size)
        moveTo(node(hasText("Report a problem")))
        moveTo(node(hasText("Sharing") and hasClickAction())) { remote.up() }
        remote.ok()
        node(hasText("Share bonded connection")).assertExists()
        remote.back()
        node(hasText("Report a problem")).assertExists()
    }

    @Test fun sharingEverySwitchByRemote() {
        show(paired, Routes.SHARE)
        val hub = toggle("Share bonded connection")
        moveTo(hub)
        remote.ok()
        val share = toggle("Share my mobile data")
        moveTo(share)
        remote.ok()
        // down from the switch at the right: the right-most chip; left along the row
        moveTo(node(hasText("Below 50%")))
        moveTo(node(hasText("Below 20%")), max = 4) { remote.left() }
        remote.ok()
        val nearby = toggle("Use nearby phones' shared data")
        moveTo(nearby)
        remote.ok()
        assertEquals(listOf("startHub", "startShare", "setShareMinBattery(20)", "setUseShares(${!paired.settings.useShares})"), calls)
        remote.back()
        node(hasText("Report a problem")).assertExists()
    }

    @Test fun reportTickAndBack() {
        show(paired, Routes.REPORT)
        val box = node(hasTestTag("report-confirm"))
        moveTo(box)
        remote.ok()
        assertEquals(ToggleableState.On, box.config().getOrNull(SemanticsProperties.ToggleableState))
        moveTo(node(hasText("Copy full report")))
        remote.back()
        node(hasTestTag("main-switch")).assertExists()
    }

    @Test fun handoffDialogOkAndBack() {
        show(paired.copy(live = UxFixtures.disconnected, handoff = Handoff.FIRST_PERMISSION))
        val turnOn = node(hasText("Turn on tailbond"))
        turnOn.assertIsFocused()
        remote.assertRingAround(turnOn.one)
        remote.ok()
        assertEquals(listOf("confirmHandoff"), calls)
        moveTo(node(hasText("Cancel")), max = 2) { remote.left() }
        remote.back()
        assertEquals(listOf("confirmHandoff", "dismissHandoff"), calls)
    }

    @Test fun removeServerStartsOnCancel() {
        show(paired, Routes.SETTINGS)
        moveTo(node(hasText("Remove from this phone")), max = 3) { remote.right() }
        remote.ok()
        val cancel = node(hasText("Cancel"))
        cancel.assertIsFocused()
        remote.assertRingAround(cancel.one)
        remote.ok()
        assertTrue(calls.isEmpty())
        // again, and this time Remove
        moveTo(node(hasText("Remove from this phone")), max = 3) { remote.right() }
        remote.ok()
        moveTo(node(hasText("Remove")), max = 2) { remote.right() }
        remote.ok()
        assertEquals(listOf("removeServer(h)"), calls)
    }
}
