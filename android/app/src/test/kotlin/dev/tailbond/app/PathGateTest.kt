package dev.tailbond.app

import dev.tailbond.app.model.AppSettings
import dev.tailbond.app.model.Mode
import dev.tailbond.app.model.PathKind
import dev.tailbond.app.model.PathSetting
import dev.tailbond.app.model.Priority
import dev.tailbond.app.service.PathGate
import dev.tailbond.app.service.PathGate.Link
import org.junit.Assert.assertFalse
import org.junit.Assert.assertTrue
import org.junit.Test

class PathGateTest {
    private val wifi = Link("wifi-1", PathKind.WIFI)
    private val cell = Link("cellular-2", PathKind.CELLULAR)
    private val ACTIVE = PathGate.STATUS_ACTIVE
    private val DEGRADED = PathGate.STATUS_DEGRADED

    private fun gate(settings: AppSettings = AppSettings(), mode: Mode = Mode.SPEED, cap: Boolean = false) =
        PathGate(pendingGraceMs = 5_000, recoverMs = 10_000).apply { configure(settings, mode, cap) }

    @Test fun speedUsesPrimaryAndSecondary() {
        val g = gate()
        assertTrue(g.wanted(wifi))
        assertTrue(g.wanted(cell))
    }

    @Test fun failoverKeepsSecondaryInStandbyUntilPrimaryFails() {
        val g = gate(mode = Mode.FAILOVER)
        val both = listOf(wifi, cell)
        g.update(0, both, mapOf(wifi.name to PathGate.STATUS_PENDING))
        assertTrue(g.wanted(wifi))
        assertFalse(g.wanted(cell))
        g.update(1_000, both, mapOf(wifi.name to ACTIVE))
        assertFalse(g.wanted(cell))
        // primary degrades: standby comes up at once
        assertTrue(g.update(2_000, both, mapOf(wifi.name to DEGRADED)))
        assertTrue(g.wanted(cell))
        // primary back: standby stays until it has been healthy for recoverMs
        g.update(3_000, both, mapOf(wifi.name to ACTIVE, cell.name to ACTIVE))
        assertTrue(g.wanted(cell))
        g.update(12_000, both, mapOf(wifi.name to ACTIVE, cell.name to ACTIVE))
        assertTrue(g.wanted(cell))
        assertTrue(g.update(13_000, both, mapOf(wifi.name to ACTIVE, cell.name to ACTIVE)))
        assertFalse(g.wanted(cell))
    }

    @Test fun primaryNetworkGoneBringsUpBackup() {
        val s = AppSettings().withPath(PathKind.CELLULAR, PathSetting(true, Priority.BACKUP))
        val g = gate(s)
        g.update(0, listOf(wifi, cell), mapOf(wifi.name to ACTIVE))
        assertFalse(g.wanted(cell))
        g.update(1_000, listOf(cell), emptyMap()) // Wi-Fi lost
        assertTrue(g.wanted(cell))
    }

    @Test fun pendingPrimaryGetsAGracePeriod() {
        val s = AppSettings().withPath(PathKind.CELLULAR, PathSetting(true, Priority.BACKUP))
        val g = gate(s)
        g.update(0, listOf(wifi, cell), emptyMap())
        assertFalse(g.wanted(cell))
        g.update(4_000, listOf(wifi, cell), mapOf(wifi.name to PathGate.STATUS_PENDING))
        assertFalse(g.wanted(cell))
        g.update(6_000, listOf(wifi, cell), mapOf(wifi.name to PathGate.STATUS_PENDING))
        assertTrue(g.wanted(cell))
    }

    @Test fun disabledAndCappedPathsAreNeverWanted() {
        val off = gate(AppSettings().withPath(PathKind.WIFI, PathSetting(enabled = false)))
        assertFalse(off.wanted(wifi))
        val capped = gate(cap = true)
        assertFalse(capped.wanted(cell))
        assertTrue(capped.wanted(wifi))
        // capped cellular stays out even as a standby
        capped.update(0, listOf(cell), emptyMap())
        assertFalse(capped.wanted(cell))
    }
}
