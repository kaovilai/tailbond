package dev.tailbond.app.failover

import kotlinx.serialization.Serializable

/** `client.failover.policy`. */
@Serializable
enum class FailoverPolicy(val wire: String) { PREFER("prefer"), ALWAYS("always") }

/** Contract timers; defaults are the contract's. */
data class FailoverTiming(
    val switchAfterMs: Long = 15_000,
    val failbackAfterMs: Long = 120_000,
    val probeEveryMs: Long = 10_000,
    /** A failed failback doubles that server's hold, up to this factor. */
    val maxHoldFactor: Int = 16,
    /** The hold resets once the server has stayed working this long. */
    val holdResetMs: Long = 600_000,
)

/** Per-server `state` of the contract's status. */
enum class ServerState(val wire: String) {
    ACTIVE("active"), CONNECTING("connecting"), STANDBY("standby"), UNREACHABLE("unreachable"), UNKNOWN("unknown"),
}

/** `failover.reason`: why the active server isn't the primary. */
enum class FailoverReason(val wire: String) { NONE(""), PRIMARY_DOWN("primary_down"), MANUAL("manual"), POLICY("policy") }

enum class SwitchKind { FAILOVER, FAILBACK, MANUAL, POLICY }

/** A decision: reconnect to [to]. */
data class Switch(val to: String, val from: String, val kind: SwitchKind, val atMs: Long)

data class ServerStatus(val id: String, val state: ServerState, val lastOkMs: Long?, val holdFactor: Int = 1)

/** What the UI and notification need, as of [nowMs]. */
data class FailoverView(
    val activeId: String,
    val servers: List<ServerStatus>,
    val reason: FailoverReason,
    /** Counting down to a failback to [failbackTo]. */
    val failbackTo: String? = null,
    val failbackInMs: Long? = null,
    val lastSwitch: Switch? = null,
    /** Since when the active server has worked without a break; null while it doesn't. */
    val workingSinceMs: Long? = null,
    val nowMs: Long = 0,
    /** Wall-clock time (ms since epoch) of [lastSwitch]; set by the caller, the engine's clock is monotonic. */
    val sinceWallMs: Long? = null,
) {
    val primaryId: String? get() = servers.firstOrNull()?.id
    fun state(id: String): ServerState? = servers.firstOrNull { it.id == id }?.state
}

/**
 * The client's server failover state machine (docs/architecture.md,
 * "Multiple servers and failover"). Pure: the caller feeds it the active
 * tunnel's health once a second and standby probe results, and acts on
 * the [Switch] that [tick] or [useNow] returns. [clock] is milliseconds.
 */
class FailoverEngine(
    private val clock: () -> Long,
    ids: List<String>,
    policy: FailoverPolicy = FailoverPolicy.PREFER,
    alwaysId: String? = null,
    timing: FailoverTiming = FailoverTiming(),
) {
    private class Probe {
        /** Result of the last probe; null = not probed yet. */
        var last: Boolean? = null
        /** Start of the current run of answered probes. */
        var okSince: Long? = null
        var lastOkAt: Long? = null
        var lastAt: Long = 0
        fun reset() { last = null; okSince = null; lastAt = 0 }
    }

    var ids: List<String> = ids.distinct()
        private set
    var policy = policy
        private set
    var alwaysId = alwaysId
        private set
    var timing = timing
        private set

    var active: String
        private set
    private var activeSince: Long
    private var lastWorking: Long? = null
    private var workingSince: Long? = null
    private var workingNow = false
    /** Picked with "Use this server now": no failback while it works. */
    private var manual: String? = null
    /** A failback in progress: if it doesn't come up, its hold doubles. */
    private var failbackTo: String? = null
    private var lastSwitch: Switch? = null
    private val probes = HashMap<String, Probe>()
    private val hold = HashMap<String, Int>()

    init {
        require(this.ids.isNotEmpty()) { "no servers" }
        active = startServer()
        activeSince = clock()
    }

    /** Where a fresh start goes: the `always` server, else the primary. */
    fun startServer(): String =
        if (policy == FailoverPolicy.ALWAYS) alwaysId?.takeIf { it in ids } ?: ids.first() else ids.first()

    fun configure(policy: FailoverPolicy, alwaysId: String?, timing: FailoverTiming) {
        this.policy = policy
        this.alwaysId = alwaysId
        this.timing = timing
    }

    /** The list changed (added, removed, reordered). An active server that's gone is replaced on the next [tick]. */
    fun setServers(next: List<String>) {
        require(next.isNotEmpty()) { "no servers" }
        ids = next.distinct()
        probes.keys.retainAll(ids.toSet())
        hold.keys.retainAll(ids.toSet())
        if (manual !in ids) manual = null
    }

    fun holdFactor(id: String): Int = hold[id] ?: 1

    /** The active tunnel's health (connected and at least one path active). Call about once a second. */
    fun reportHealth(working: Boolean) {
        val now = clock()
        workingNow = working
        if (working) {
            lastWorking = now
            val since = workingSince ?: now.also { workingSince = it }
            if (failbackTo == active) failbackTo = null // the failback came up
            if (now - since >= timing.holdResetMs) hold.remove(active)
        } else {
            workingSince = null
        }
    }

    /** A standby server's probe: [ok] = it answered within the timeout. */
    fun reportProbe(id: String, ok: Boolean) {
        if (id == active || id !in ids) return
        val now = clock()
        val p = probes.getOrPut(id) { Probe() }
        p.last = ok
        p.lastAt = now
        if (ok) {
            p.lastOkAt = now
            if (p.okSince == null) p.okSince = now
        } else {
            p.okSince = null // hysteresis: one failed probe restarts the count
        }
    }

    /** "Use this server now": switch at once (null if it already is the active one). */
    fun useNow(id: String): Switch? {
        if (id !in ids) return null
        manual = id
        if (id == active) return null
        return switchTo(id, SwitchKind.MANUAL)
    }

    /** "Go back to automatic" after a manual pick: failback and failover rules apply again. */
    fun resumeAutomatic() { manual = null }

    /** Whether the active server should change now; at most one switch per call. */
    fun tick(): Switch? {
        if (active !in ids) return switchTo(startServer(), SwitchKind.POLICY)
        if (policy == FailoverPolicy.ALWAYS) {
            val target = alwaysId?.takeIf { it in ids } ?: ids.first()
            manual = null
            return if (active != target) switchTo(target, SwitchKind.POLICY) else null
        }
        val now = clock()
        if (!workingNow && ids.size > 1) {
            val from = maxOf(lastWorking ?: Long.MIN_VALUE, activeSince)
            if (now - from >= timing.switchAfterMs) {
                if (failbackTo == active) {
                    hold[active] = (holdFactor(active) * 2).coerceAtMost(timing.maxHoldFactor)
                    failbackTo = null
                }
                if (manual == active) manual = null
                return switchTo(nextServer(), SwitchKind.FAILOVER)
            }
        }
        if (manual == active) return null
        failbackCandidate(now)?.let { (id, remaining) ->
            if (remaining <= 0) {
                failbackTo = id
                return switchTo(id, SwitchKind.FAILBACK)
            }
        }
        return null
    }

    /** The earliest server before the active one that is counting towards a failback, and the time left. */
    private fun failbackCandidate(now: Long): Pair<String, Long>? {
        if (policy != FailoverPolicy.PREFER || manual == active) return null
        val idx = ids.indexOf(active)
        for (id in ids.take(idx.coerceAtLeast(0))) {
            val p = probes[id] ?: continue
            val since = p.okSince ?: continue
            if (p.last != true || now - p.lastAt > timing.probeEveryMs * 3) continue // stale: probing stopped
            return id to (timing.failbackAfterMs * holdFactor(id) - (now - since))
        }
        return null
    }

    /**
     * Next in list order after the active one (wrapping), skipping servers
     * whose last probe failed when another one answered.
     */
    private fun nextServer(): String {
        val i = ids.indexOf(active)
        val order = (1 until ids.size).map { ids[(i + it).mod(ids.size)] }
        val anyAnswered = order.any { probes[it]?.last == true }
        return if (anyAnswered) order.first { probes[it]?.last != false } else order.first()
    }

    private fun switchTo(id: String, kind: SwitchKind): Switch {
        val now = clock()
        val from = active
        if (from != id) {
            probes.getOrPut(from) { Probe() }.apply {
                reset()
                // it just failed to carry a tunnel: unreachable until a probe says otherwise
                if (kind == SwitchKind.FAILOVER) { last = false; lastAt = now }
            }
        }
        probes.remove(id)
        active = id
        activeSince = now
        lastWorking = null
        workingSince = null
        workingNow = false
        if (kind != SwitchKind.MANUAL && manual != id) manual = null
        return Switch(id, from, kind, now).also { lastSwitch = it }
    }

    fun view(): FailoverView {
        val now = clock()
        val servers = ids.map { id ->
            if (id == active) {
                ServerStatus(id, if (workingNow) ServerState.ACTIVE else ServerState.CONNECTING, lastWorking, holdFactor(id))
            } else {
                val p = probes[id]
                val st = when (p?.last) { true -> ServerState.STANDBY; false -> ServerState.UNREACHABLE; null -> ServerState.UNKNOWN }
                ServerStatus(id, st, p?.lastOkAt, holdFactor(id))
            }
        }
        val reason = when {
            active == ids.first() -> FailoverReason.NONE
            manual == active -> FailoverReason.MANUAL
            policy == FailoverPolicy.ALWAYS -> FailoverReason.POLICY
            else -> FailoverReason.PRIMARY_DOWN
        }
        val fb = failbackCandidate(now)
        return FailoverView(
            activeId = active, servers = servers, reason = reason,
            failbackTo = fb?.first, failbackInMs = fb?.second?.coerceAtLeast(0),
            lastSwitch = lastSwitch, workingSinceMs = workingSince, nowMs = now,
        )
    }
}
