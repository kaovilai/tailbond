package dev.tailbond.app

import dev.tailbond.app.model.AppSettings
import dev.tailbond.app.model.Captive
import dev.tailbond.app.model.Mode
import dev.tailbond.app.model.PathKind
import dev.tailbond.app.model.Words
import dev.tailbond.app.model.Words.LinkStatus
import dev.tailbond.app.service.ConnState
import dev.tailbond.app.service.Live
import dev.tailbond.app.service.PathGate
import dev.tailbond.app.service.PathSample
import org.junit.Assert.assertEquals
import org.junit.Assert.assertFalse
import org.junit.Assert.assertTrue
import org.junit.Test

class CaptiveTest {
    @Test fun classification() {
        // Android names the portal
        assertTrue(Captive.needsSignIn(captivePortal = true, validated = false, unvalidatedSinceMs = 1_000, nowMs = 1_000))
        // validated wins (signed in)
        assertFalse(Captive.needsSignIn(captivePortal = true, validated = true, unvalidatedSinceMs = null, nowMs = 0))
        // still validating: not yet
        assertFalse(Captive.needsSignIn(captivePortal = false, validated = false, unvalidatedSinceMs = 0, nowMs = 5_000))
        // never validated after the grace period: treat as a portal
        assertTrue(Captive.needsSignIn(captivePortal = false, validated = false, unvalidatedSinceMs = 0, nowMs = Captive.GRACE_MS))
        assertFalse(Captive.needsSignIn(captivePortal = false, validated = false, unvalidatedSinceMs = null, nowMs = 99_000))
    }

    private fun sample(kind: PathKind, status: Int) = PathSample(kind.name, kind, status, 30, 0, 0, 0, 0)

    @Test fun wifiCardSaysSignIn() {
        val live = Live(state = ConnState.CONNECTED, paths = listOf(sample(PathKind.CELLULAR, PathGate.STATUS_ACTIVE)))
        assertEquals(LinkStatus.NEEDS_SIGN_IN, Words.linkStatus(PathKind.WIFI, live, AppSettings(), Mode.SPEED, false, wifiSignIn = true))
        assertEquals("Wi-Fi needs sign-in · using mobile data", Words.connectionsSummary(live, AppSettings(), Mode.SPEED, false, wifiSignIn = true))
        // prompts off: back to the plain state
        assertEquals(LinkStatus.UNAVAILABLE, Words.linkStatus(PathKind.WIFI, live, AppSettings(captivePrompts = false), Mode.SPEED, false, wifiSignIn = true))
        // once Wi-Fi carries traffic, it's simply in use
        val both = live.copy(paths = live.paths + sample(PathKind.WIFI, PathGate.STATUS_ACTIVE))
        assertEquals(LinkStatus.IN_USE, Words.linkStatus(PathKind.WIFI, both, AppSettings(), Mode.SPEED, false, wifiSignIn = true))
    }
}
