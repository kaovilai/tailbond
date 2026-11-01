package dev.tailbond.app.service

import dev.tailbond.app.model.AppSettings
import dev.tailbond.app.model.Mode
import dev.tailbond.app.model.PathKind
import dev.tailbond.app.model.Priority

/**
 * Decides which available links carry a path (per-path enable, priority,
 * failover standby, cellular cap). Pure logic: the service feeds it the
 * live path table once a second and asks [wanted] per network.
 *
 * Tiers: [Mode.FAILOVER] uses only primary links; other modes use primary
 * and secondary. The rest are standby: attached only while no in-tier link
 * is healthy, and detached again once one has been healthy for
 * [RECOVER_MS] (hysteresis against flapping).
 */
class PathGate(
    private val pendingGraceMs: Long = PENDING_GRACE_MS,
    private val recoverMs: Long = RECOVER_MS,
) {
    /** A network as the gate sees it: its sdk-network name and kind. */
    data class Link(val name: String, val kind: PathKind)

    @Volatile private var settings = AppSettings()
    @Volatile private var mode = Mode.SPEED
    @Volatile private var capReached = false
    @Volatile var standbyActive = false
        private set

    private val firstSeen = HashMap<String, Long>()
    private var healthySince = -1L

    fun configure(settings: AppSettings, mode: Mode, capReached: Boolean) {
        this.settings = settings
        this.mode = mode
        this.capReached = capReached
    }

    fun inTier(p: Priority, m: Mode = mode): Boolean = when (m) {
        Mode.FAILOVER -> p == Priority.PRIMARY
        else -> p != Priority.BACKUP
    }

    private fun usable(kind: PathKind): Boolean =
        settings.path(kind).enabled && !(kind == PathKind.CELLULAR && capReached)

    fun wanted(link: Link): Boolean {
        if (!usable(link.kind)) return false
        return inTier(settings.path(link.kind).priority) || standbyActive
    }

    /**
     * Feed the current state. [available]: networks the monitor offers.
     * [status]: live mqvpn path status by name (0 PENDING, 1 ACTIVE,
     * 2 DEGRADED, 3 STANDBY, 4 CLOSED); absent = not attached.
     * Returns true when [standbyActive] changed (the caller reconciles).
     */
    @Synchronized
    fun update(nowMs: Long, available: List<Link>, status: Map<String, Int>): Boolean {
        firstSeen.keys.retainAll(available.map { it.name }.toSet())
        available.forEach { firstSeen.putIfAbsent(it.name, nowMs) }

        val healthy = available.any { link ->
            usable(link.kind) && inTier(settings.path(link.kind).priority) && when (status[link.name]) {
                STATUS_ACTIVE, STATUS_STANDBY -> true
                // still coming up: give it a grace period before calling it failed
                null, STATUS_PENDING -> nowMs - (firstSeen[link.name] ?: nowMs) < pendingGraceMs
                else -> false
            }
        }
        val before = standbyActive
        if (!healthy) {
            healthySince = -1L
            standbyActive = true
        } else {
            if (healthySince < 0) healthySince = nowMs
            if (standbyActive && nowMs - healthySince >= recoverMs) standbyActive = false
        }
        return before != standbyActive
    }

    companion object {
        const val STATUS_PENDING = 0
        const val STATUS_ACTIVE = 1
        const val STATUS_DEGRADED = 2
        const val STATUS_STANDBY = 3
        const val STATUS_CLOSED = 4
        const val PENDING_GRACE_MS = 10_000L
        const val RECOVER_MS = 10_000L
    }
}
