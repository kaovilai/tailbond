package dev.tailbond.app.model

import dev.tailbond.app.pairing.PairingPayload
import dev.tailbond.app.service.ConnState
import dev.tailbond.app.service.ErrorKind
import dev.tailbond.app.service.Live
import dev.tailbond.app.service.PathGate
import dev.tailbond.app.service.PathSample

/**
 * Everything the UI, tile and notification say about the tunnel, in plain
 * words (Tailscale app wording where there is an equivalent). The technical
 * names stay in the contract and [Mode.wire].
 */
object Words {
    fun stateTitle(s: ConnState): String = when (s) {
        ConnState.CONNECTED -> "Connected"
        ConnState.CONNECTING -> "Connecting…"
        ConnState.RECONNECTING -> "Reconnecting…"
        ConnState.ERROR -> "Not connected"
        ConnState.DISCONNECTED -> "Disconnected"
    }

    /** One line under the big switch. [server] is the home server's short name. */
    fun stateLine(live: Live, server: String, handedToTailscale: Boolean = false): String = when {
        handedToTailscale && live.state == ConnState.DISCONNECTED -> "Stopped. Tailscale is in charge on this phone."
        else -> stateLineFor(live, server)
    }

    private fun stateLineFor(live: Live, server: String): String = when (live.state) {
        ConnState.CONNECTED -> "Through $server · ${activeKinds(live).ifEmpty { "no connection in use" }}"
        ConnState.CONNECTING -> "Reaching $server…"
        ConnState.RECONNECTING -> "Lost touch with $server, trying again" + (live.message?.let { " ($it)" } ?: "")
        ConnState.ERROR -> "Stopped. See below for how to fix it."
        ConnState.DISCONNECTED ->
            "Stopped. Turn on to reach $server and your Tailscale machines, or switch back to Tailscale below."
    }

    /** "Wi-Fi + mobile data": link kinds carrying traffic right now. */
    fun activeKinds(live: Live): String =
        live.paths.filter { it.status == PathGate.STATUS_ACTIVE }.mapNotNull { it.kind }
            .distinct().sortedBy { it.ordinal }.joinToString(" + ") { it.phrase }

    /** Status of one link, as the Connections cards show it. */
    /** [good]: green dot (true), red (false), grey (null: waiting or not in use by choice). */
    enum class LinkStatus(val label: String, val good: Boolean?) {
        IN_USE("In use", true),
        BACKUP("Backup only · not in use", null),
        SAVING("Not in use · saving data", null),
        TAKING_OVER("In use · taking over", true),
        STARTING("Starting…", null),
        WEAK("Struggling", false),
        NOT_WORKING("Not working right now", false),
        CAP_REACHED("Paused · monthly limit reached", false),
        OFF("Not used by tailbond", null),
        UNAVAILABLE("Not available", null),
        IDLE("Ready", null),
    }

    private fun rank(status: Int): Int = when (status) {
        PathGate.STATUS_ACTIVE -> 0
        PathGate.STATUS_STANDBY -> 1
        PathGate.STATUS_PENDING -> 2
        PathGate.STATUS_DEGRADED -> 3
        -1 -> 4
        else -> 5
    }

    /** The best-working sample of [kind] (a kind can have more than one network). */
    fun sampleFor(live: Live, kind: PathKind): PathSample? =
        live.paths.filter { it.kind == kind }.minByOrNull { rank(it.status) }

    fun linkStatus(kind: PathKind, live: Live, settings: AppSettings, mode: Mode, capReached: Boolean): LinkStatus {
        val setting = settings.path(kind)
        if (!setting.enabled) return LinkStatus.OFF
        val on = live.state == ConnState.CONNECTED || live.state == ConnState.RECONNECTING
        if (!on) return LinkStatus.IDLE
        if (kind == PathKind.CELLULAR && capReached) return LinkStatus.CAP_REACHED
        val s = sampleFor(live, kind) ?: return LinkStatus.UNAVAILABLE
        return when (s.status) {
            PathGate.STATUS_ACTIVE -> if (live.standbyActive && isBackupRole(kind, settings, mode)) LinkStatus.TAKING_OVER else LinkStatus.IN_USE
            PathGate.STATUS_PENDING -> LinkStatus.STARTING
            PathGate.STATUS_DEGRADED -> LinkStatus.WEAK
            PathGate.STATUS_STANDBY, -1 ->
                if (kind == PathKind.CELLULAR) LinkStatus.SAVING else LinkStatus.BACKUP
            else -> LinkStatus.NOT_WORKING
        }
    }

    private fun isBackupRole(kind: PathKind, settings: AppSettings, mode: Mode): Boolean {
        val p = settings.path(kind).priority
        return p == Priority.BACKUP || (mode == Mode.FAILOVER && p != Priority.PRIMARY)
    }

    /** One sentence answering "is it using both?". */
    fun connectionsSummary(live: Live, settings: AppSettings, mode: Mode, capReached: Boolean): String {
        val on = live.state == ConnState.CONNECTED || live.state == ConnState.RECONNECTING
        if (!on) return when (mode) {
            Mode.FAILOVER -> "Will use Wi-Fi first and keep mobile data as backup"
            else -> "Will use Wi-Fi and mobile data together"
        }
        val wifi = linkStatus(PathKind.WIFI, live, settings, mode, capReached)
        val cell = linkStatus(PathKind.CELLULAR, live, settings, mode, capReached)
        val wifiUsed = wifi == LinkStatus.IN_USE || wifi == LinkStatus.TAKING_OVER
        val cellUsed = cell == LinkStatus.IN_USE || cell == LinkStatus.TAKING_OVER
        return when {
            wifiUsed && cellUsed -> "Using both Wi-Fi and mobile data"
            wifiUsed && cell == LinkStatus.SAVING -> "Using Wi-Fi · saving mobile data (backup only)"
            wifiUsed && cell == LinkStatus.CAP_REACHED -> "Using Wi-Fi only · mobile data limit reached"
            wifiUsed -> "Using Wi-Fi only"
            cellUsed && wifi == LinkStatus.NOT_WORKING -> "Wi-Fi isn't working · using mobile data"
            cellUsed -> "Using mobile data only"
            else -> "No connection is carrying traffic yet"
        }
    }

    data class ErrorText(val title: String, val body: String, val fix: Fix, val second: Fix? = null)

    enum class Fix(val label: String) { RETRY("Try again"), REPAIR("Pair again"), PAIR("Pair this phone") }

    fun error(kind: ErrorKind?, server: String, address: String, detail: String?): ErrorText = when (kind) {
        ErrorKind.UNREACHABLE -> ErrorText(
            "Can't reach your home server",
            "$server didn't answer at $address. Check that it's turned on and try again. " +
                "If it keeps happening, the home router may need to forward UDP port ${address.substringAfterLast(':', "443")} " +
                "to it: if someone else set up your home server, show them this message.",
            Fix.RETRY,
        )
        ErrorKind.IDENTITY -> ErrorText(
            "Can't confirm this is your home server",
            "tailbond stopped because the server answering isn't the $server this phone was paired with. " +
                "Try again later, and tell whoever set it up. If $server was just reinstalled, " +
                "they can add this phone again (Settings → Pair again).",
            Fix.RETRY,
        )
        ErrorKind.REJECTED -> ErrorText(
            "Your home server doesn't recognise this phone",
            "This phone may have been removed from $server. Add it again from the home server's page.",
            Fix.REPAIR, Fix.RETRY,
        )
        ErrorKind.NOT_PAIRED -> ErrorText(
            "This phone isn't paired yet",
            "Add this phone on your home server first.",
            Fix.PAIR,
        )
        ErrorKind.OTHER, null -> ErrorText(
            "Couldn't connect",
            "Something went wrong while connecting to $server" + (detail?.let { " ($it)" } ?: "") + ".",
            Fix.RETRY,
        )
    }

    /** Short tile / notification text. */
    fun tileSubtitle(live: Live): String = when (live.state) {
        ConnState.CONNECTED -> activeKinds(live).ifEmpty { "Connected" }
        ConnState.CONNECTING -> "Connecting…"
        ConnState.RECONNECTING -> "Reconnecting…"
        ConnState.ERROR -> "Can't connect"
        ConnState.DISCONNECTED -> "Disconnected"
    }
}

/** "homebox" from the pairing's TLS name (homebox.tail1234.ts.net) or server address. */
fun serverShortName(p: PairingPayload): String {
    val name = p.sni?.takeIf { it.isNotBlank() } ?: p.host
    return if (Cidr.isIpLiteral(name)) name else name.substringBefore('.')
}

/** "tail1234.ts.net" when the server has a MagicDNS name, else null. */
fun tailnetDomain(p: PairingPayload): String? =
    p.sni?.takeIf { it.endsWith(".ts.net") }?.substringAfter('.')
