package dev.tailbond.app

import dev.tailbond.app.failover.FailoverEngine
import dev.tailbond.app.failover.FailoverPolicy
import dev.tailbond.app.failover.FailoverReason
import dev.tailbond.app.failover.FailoverTiming
import dev.tailbond.app.failover.QuicProbe
import dev.tailbond.app.failover.ServerState
import dev.tailbond.app.failover.Switch
import dev.tailbond.app.failover.SwitchKind
import org.junit.Assert.assertEquals
import org.junit.Assert.assertNotNull
import org.junit.Assert.assertNull
import org.junit.Assert.assertTrue
import org.junit.Test
import kotlin.random.Random

/** The contract's failover state machine ("Multiple servers and failover") on a fake clock. */
class FailoverEngineTest {
    private var now = 1_000_000L
    private val home = "homebox"
    private val vps = "vps"

    private fun engine(vararg ids: String = arrayOf(home, vps), policy: FailoverPolicy = FailoverPolicy.PREFER, always: String? = null) =
        FailoverEngine({ now }, ids.toList(), policy, always, FailoverTiming())

    /**
     * Advances the clock a second at a time for [ms], feeding the active
     * server's health and (every 10 s) probe results for the others; returns
     * the first switch, or null.
     */
    private fun run(
        e: FailoverEngine,
        ms: Long,
        working: (String) -> Boolean,
        probe: (String) -> Boolean = { true },
    ): Switch? {
        val end = now + ms
        while (now < end) {
            now += 1_000
            e.reportHealth(working(e.active))
            if (now % 10_000 == 0L) e.ids.filter { it != e.active }.forEach { e.reportProbe(it, probe(it)) }
            e.tick()?.let { return it }
        }
        return null
    }

    @Test fun startsOnThePrimary() {
        val e = engine()
        assertEquals(home, e.active)
        e.reportHealth(true)
        assertEquals(FailoverReason.NONE, e.view().reason)
        assertEquals(ServerState.ACTIVE, e.view().state(home))
    }

    @Test fun primaryDownSwitchesAt15s() {
        val e = engine()
        val start = now
        assertNull(run(e, 5_000, { true }))
        val downAt = now
        // the last working moment counts: 14 s without a working tunnel is not enough
        assertNull(run(e, 14_000, { false }))
        val sw = run(e, 2_000, { false })
        assertNotNull(sw)
        assertEquals(vps, sw!!.to)
        assertEquals(SwitchKind.FAILOVER, sw.kind)
        assertEquals(15_000L, sw.atMs - downAt)
        assertTrue(sw.atMs > start)
        val v = e.view()
        assertEquals(vps, v.activeId)
        assertEquals(FailoverReason.PRIMARY_DOWN, v.reason)
        assertEquals(ServerState.UNREACHABLE, v.state(home)) // it just failed: unreachable until a probe answers
        assertEquals(ServerState.CONNECTING, v.state(vps))
    }

    @Test fun neverWorkingCountsFromTheSwitch() {
        val e = engine()
        assertNull(run(e, 14_000, { false }))
        assertEquals(vps, run(e, 1_000, { false })?.to)
        // the new server gets its own 15 s, counted from the switch
        assertNull(run(e, 14_000, { false }))
        assertEquals(home, run(e, 1_000, { false })?.to) // wraps around
    }

    @Test fun skipsServersWhoseProbeFailedWhenAnotherAnswered() {
        val e = engine(home, vps, "office")
        // vps doesn't answer, office does
        assertNull(run(e, 20_000, { true }, { it == "office" }))
        val sw = run(e, 16_000, { false }, { it == "office" })
        assertEquals("office", sw?.to)
    }

    @Test fun nothingAnsweredTakesTheNextInOrder() {
        val e = engine(home, vps, "office")
        assertNull(run(e, 20_000, { true }, { false }))
        assertEquals(vps, run(e, 16_000, { false }, { false })?.to)
    }

    @Test fun failsBackAfter120sOfAnsweredProbes() {
        val e = engine()
        assertEquals(vps, run(e, 20_000, { it == vps }, { false })?.to)
        // vps works; homebox keeps failing its probes for a while
        assertNull(run(e, 60_000, { it == vps }, { false }))
        // homebox answers from now on: the first answered probe starts the count
        var firstOk = -1L
        val sw = run(e, 200_000, { it == vps }, { id -> if (firstOk < 0 && id == home) firstOk = now; true })
        assertNotNull(sw)
        assertEquals(home, sw!!.to)
        assertEquals(SwitchKind.FAILBACK, sw.kind)
        assertEquals(120_000L, sw.atMs - firstOk)
    }

    @Test fun countdownIsReportedWhileFailingBack() {
        val e = engine()
        run(e, 20_000, { it == vps }, { false })
        // probes at multiples of 10 s; homebox answers from the next one on
        run(e, 10_000, { it == vps }, { true })
        val v = e.view()
        assertEquals(home, v.failbackTo)
        assertNotNull(v.failbackInMs)
        assertTrue(v.failbackInMs!! in 110_000L..120_000L)
    }

    @Test fun flappingProbesNeverFailBack() {
        val e = engine()
        assertEquals(vps, run(e, 20_000, { it == vps }, { false })?.to)
        // homebox answers 10 probes, misses one, answers 10, misses one... (each run < 120 s)
        var n = 0
        val sw = run(e, 30 * 60_000L, { it == vps }, { (++n % 11) != 0 })
        assertNull(sw)
        assertEquals(vps, e.active)
    }

    @Test fun failedFailbackDoublesTheHold() {
        val e = engine()
        assertEquals(vps, run(e, 20_000, { it == vps }, { false })?.to)
        // homebox answers probes but can't carry a tunnel
        val back = run(e, 200_000, { it == vps }, { true })
        assertEquals(SwitchKind.FAILBACK, back?.kind)
        val away = run(e, 16_000, { it == vps }, { true })
        assertEquals(vps, away?.to)
        assertEquals(2, e.holdFactor(home))
        // the next try waits 240 s of answered probes
        val start = now
        val again = run(e, 400_000, { it == vps }, { true })
        assertEquals(home, again?.to)
        assertTrue("waited ${again!!.atMs - start}", again.atMs - start >= 240_000)
        // fails again: 4x, then 8x, 16x, capped at 16x
        run(e, 16_000, { it == vps }, { true })
        assertEquals(4, e.holdFactor(home))
        repeat(3) {
            run(e, 2_000_000, { it == vps }, { true })
            run(e, 16_000, { it == vps }, { true })
        }
        assertEquals(16, e.holdFactor(home))
    }

    @Test fun holdResetsAfterTenMinutesWorking() {
        val e = engine()
        run(e, 20_000, { it == vps }, { false })
        run(e, 200_000, { it == vps }, { true })
        run(e, 16_000, { it == vps }, { true })
        assertEquals(2, e.holdFactor(home))
        // this time homebox comes up and stays up
        val back = run(e, 400_000, { it == vps }, { true })
        assertEquals(home, back?.to)
        assertNull(run(e, 9 * 60_000L, { true }))
        assertEquals(2, e.holdFactor(home))
        assertNull(run(e, 61_000, { true }))
        assertEquals(1, e.holdFactor(home))
    }

    @Test fun manualSuppressesFailbackUntilThatServerFails() {
        val e = engine()
        run(e, 5_000, { true })
        val sw = e.useNow(vps)
        assertEquals(SwitchKind.MANUAL, sw?.kind)
        assertEquals(vps, e.active)
        assertEquals(FailoverReason.MANUAL, e.view().reason)
        // homebox answers every probe for 10 minutes: no failback while vps works
        assertNull(run(e, 10 * 60_000L, { true }, { true }))
        assertNull(e.view().failbackTo)
        // vps fails: prefer resumes, and the next server in order is homebox
        val back = run(e, 16_000, { it != vps }, { true })
        assertEquals(home, back?.to)
        assertEquals(SwitchKind.FAILOVER, back?.kind)
    }

    @Test fun goBackToAutomaticAfterAManualPick() {
        val e = engine()
        run(e, 5_000, { true })
        e.useNow(vps)
        assertNull(run(e, 5 * 60_000L, { true }, { true }))
        e.resumeAutomatic()
        assertEquals(FailoverReason.PRIMARY_DOWN, e.view().reason)
        // homebox has been answering all along: the failback comes at the next tick
        assertEquals(SwitchKind.FAILBACK, e.tick()?.kind)
        assertEquals(home, e.active)
    }

    @Test fun manualPrimaryAfterFailoverIsImmediate() {
        val e = engine()
        run(e, 20_000, { it == vps }, { false })
        assertEquals(vps, e.active)
        assertEquals(home, e.useNow(home)?.to)
        assertNull(e.useNow(home)) // already active
    }

    @Test fun alwaysNeverSwitches() {
        val e = engine(policy = FailoverPolicy.ALWAYS, always = vps)
        assertEquals(vps, e.active)
        assertNull(run(e, 10 * 60_000L, { false }, { true }))
        assertEquals(vps, e.active)
        assertEquals(FailoverReason.POLICY, e.view().reason)
        assertNull(e.view().failbackTo)
    }

    @Test fun policyChangeToAlwaysMovesAtOnce() {
        val e = engine()
        run(e, 5_000, { true })
        e.configure(FailoverPolicy.ALWAYS, vps, FailoverTiming())
        assertEquals(SwitchKind.POLICY, e.tick()?.kind)
        assertEquals(vps, e.active)
    }

    @Test fun removedActiveServerGoesBackToThePrimary() {
        val e = engine()
        run(e, 20_000, { it == vps }, { false })
        assertEquals(vps, e.active)
        e.setServers(listOf(home))
        assertEquals(home, e.tick()?.to)
        // a single server never switches
        assertNull(run(e, 60_000, { false }))
    }

    @Test fun reorderMakesTheNewFirstThePrimary() {
        val e = engine()
        run(e, 5_000, { true })
        e.setServers(listOf(vps, home))
        assertEquals(FailoverReason.PRIMARY_DOWN, e.view().reason)
        // vps (now first) answers for 120 s: back to the new primary
        val sw = run(e, 200_000, { true }, { true })
        assertEquals(vps, sw?.to)
        assertEquals(SwitchKind.FAILBACK, sw?.kind)
    }

    @Test fun customTimers() {
        val e = FailoverEngine({ now }, listOf(home, vps), timing = FailoverTiming(switchAfterMs = 30_000, failbackAfterMs = 60_000))
        assertNull(run(e, 29_000, { false }))
        assertEquals(vps, run(e, 1_000, { false })?.to)
    }

    @Test fun probePacketIsAQuicShortHeader() {
        repeat(50) { i ->
            val p = QuicProbe.packet(Random(i))
            assertEquals(60, p.size)
            assertEquals(0x40, p[0].toInt() and 0xc0) // header form 0, fixed bit 1
        }
    }
}
