package dev.tailbond.app.model

import dev.tailbond.app.failover.FailoverReason
import dev.tailbond.app.failover.FailoverView
import dev.tailbond.app.failover.ServerState
import dev.tailbond.app.failover.SwitchKind
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
        handedToTailscale && live.state == ConnState.DISCONNECTED -> "Stopped. Back to Tailscale: turn it on in its app."
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
        NEEDS_SIGN_IN("Needs sign-in · not used yet", false),
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

    fun linkStatus(kind: PathKind, live: Live, settings: AppSettings, mode: Mode, capReached: Boolean, wifiSignIn: Boolean = false): LinkStatus {
        val setting = settings.path(kind)
        if (!setting.enabled) return LinkStatus.OFF
        if (kind == PathKind.WIFI && wifiSignIn && settings.captivePrompts &&
            sampleFor(live, kind)?.status != PathGate.STATUS_ACTIVE
        ) return LinkStatus.NEEDS_SIGN_IN
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
    fun connectionsSummary(live: Live, settings: AppSettings, mode: Mode, capReached: Boolean, wifiSignIn: Boolean = false): String {
        val on = live.state == ConnState.CONNECTED || live.state == ConnState.RECONNECTING
        if (!on) return when (mode) {
            Mode.FAILOVER -> "Will use Wi-Fi first and keep mobile data as backup"
            else -> "Will use Wi-Fi and mobile data together"
        }
        val wifi = linkStatus(PathKind.WIFI, live, settings, mode, capReached, wifiSignIn)
        val cell = linkStatus(PathKind.CELLULAR, live, settings, mode, capReached)
        val wifiUsed = wifi == LinkStatus.IN_USE || wifi == LinkStatus.TAKING_OVER
        val cellUsed = cell == LinkStatus.IN_USE || cell == LinkStatus.TAKING_OVER
        return when {
            wifiUsed && cellUsed -> "Using both Wi-Fi and mobile data"
            wifiUsed && cell == LinkStatus.SAVING -> "Using Wi-Fi · saving mobile data (backup only)"
            wifiUsed && cell == LinkStatus.CAP_REACHED -> "Using Wi-Fi only · mobile data limit reached"
            wifiUsed -> "Using Wi-Fi only"
            cellUsed && wifi == LinkStatus.NEEDS_SIGN_IN -> "Wi-Fi needs sign-in · using mobile data"
            wifi == LinkStatus.NEEDS_SIGN_IN -> "Wi-Fi needs sign-in"
            cellUsed && wifi == LinkStatus.NOT_WORKING -> "Wi-Fi isn't working · using mobile data"
            cellUsed -> "Using mobile data only"
            else -> "No connection is carrying traffic yet"
        }
    }

    data class ErrorText(val title: String, val body: String, val fix: Fix, val second: Fix? = null)

    enum class Fix(val label: String) { RETRY("Try again"), REPAIR("Pair again"), PAIR("Pair this phone") }

    /** [servers]: every server's name when the phone has more than one (all of them were tried). */
    fun error(kind: ErrorKind?, server: String, address: String, detail: String?, servers: List<String> = emptyList()): ErrorText = when {
        servers.size > 1 && kind != ErrorKind.NOT_PAIRED -> ErrorText(
            "Can't reach any of your servers",
            "tailbond tried ${ServerList.joinNames(servers)} and none of them worked" + (detail?.let { " ($it)" } ?: "") + ". " +
                "Check that this phone is online and that at least one of them is turned on, then try again.",
            Fix.RETRY,
        )
        else -> errorOne(kind, server, address, detail)
    }

    private fun errorOne(kind: ErrorKind?, server: String, address: String, detail: String?): ErrorText = when (kind) {
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
                "Try again later, and tell whoever set up $server. Don't pair this phone again " +
                "unless they ask you to.",
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

    // --- servers and failover ---

    /** "1:20" for a countdown (rounded up to the second). */
    fun mmss(ms: Long): String {
        val sec = ((ms.coerceAtLeast(0) + 999) / 1000)
        return "${sec / 60}:${"%02d".format(sec % 60)}"
    }

    /** How long "Back on homebox" stays on screen after a failback. */
    const val BACK_ON_MS = 5 * 60_000L

    /**
     * The failover sentence for the home card and the notification:
     * "Using vps (homebox is unreachable)", "Back to homebox in 1:20",
     * "Back on homebox"; null when the primary is in use as normal.
     */
    fun serverLine(f: FailoverView?, names: Map<String, String>): String? {
        f ?: return null
        fun n(id: String?) = id?.let { names[it] } ?: "the other server"
        val active = n(f.activeId)
        val primaryId = f.primaryId
        val primary = n(primaryId)
        if (f.failbackTo != null && f.failbackInMs != null) return "Back to ${n(f.failbackTo)} in ${mmss(f.failbackInMs)}"
        if (f.activeId == primaryId) {
            val sw = f.lastSwitch
            val recent = sw != null && sw.kind == SwitchKind.FAILBACK && sw.to == primaryId && f.nowMs - sw.atMs <= BACK_ON_MS
            return if (recent && f.state(f.activeId) == ServerState.ACTIVE) "Back on $primary" else null
        }
        return when (f.reason) {
            FailoverReason.MANUAL -> "Using $active (you picked it)"
            FailoverReason.POLICY -> "Always using $active"
            else -> when (f.state(primaryId ?: "")) {
                ServerState.UNREACHABLE -> "Using $active ($primary is unreachable)"
                ServerState.STANDBY -> "Using $active ($primary is answering again)"
                else -> "Using $active ($primary stopped working)"
            }
        }
    }

    /** Status of one server in the lists, and its dot: green, red, grey (null). */
    fun serverStatus(id: String, f: FailoverView?, on: Boolean, isPrimary: Boolean): Pair<String, Boolean?> {
        if (!on || f == null) return (if (isPrimary) "Main server" else "Backup") to null
        val st = f.state(id) ?: return "Not checked yet" to null
        return when (st) {
            ServerState.ACTIVE -> "In use" to true
            ServerState.CONNECTING -> "Connecting…" to null
            ServerState.STANDBY -> (if (isPrimary) "Answering again" else "Ready as backup") to true
            ServerState.UNREACHABLE -> "Unreachable" to false
            ServerState.UNKNOWN -> "Checking…" to null
        }
    }

    /** The policy choices in Settings, in plain words. */
    fun preferLabel(names: List<String>): String = when (names.size) {
        0, 1 -> "Use ${names.firstOrNull() ?: "the main server"}"
        2 -> "Prefer ${names[0]}, use ${names[1]} if ${names[0]} is down"
        else -> "Prefer ${names[0]}, then ${ServerList.joinNames(names.drop(1))} if ${names[0]} is down"
    }

    fun preferExplain(primary: String, s: AppSettings): String =
        "Switches after ${seconds(s.failoverTiming.switchAfterMs)} without a working connection, and goes back to $primary " +
            "once it has answered for ${seconds(s.failoverTiming.failbackAfterMs)} in a row."

    private fun seconds(ms: Long): String {
        val s = ms / 1000
        return if (s % 60 == 0L && s >= 60) (if (s == 60L) "1 minute" else "${s / 60} minutes") else "$s seconds"
    }

    /** Why a server can't be added: title and what to do. */
    fun addProblem(newName: String, c: AddCheck): Pair<String, String>? = when (c) {
        AddCheck.Ok -> null
        is AddCheck.Duplicate -> "$newName is already in your list" to
            "This phone already has ${c.existing}. To renew its pairing, use Pair again next to ${c.existing} in Settings."
        is AddCheck.SubnetClash -> "$newName uses the same addresses as ${c.existing}" to
            "Both servers use ${c.subnet} inside the tunnel, so tailbond can't switch between them. " +
            "On $newName, change \"subnet\" under \"server\" in /etc/tailbond/tailbond.json to a free range such as 10.78.0.0/24 " +
            "and restart tailbond there (or ask whoever runs $newName), then add it again. " +
            "Asking over Tailscale does this by itself while $newName has no other devices."
    }

    // --- website bypass ---

    fun websitesSummary(s: AppSettings, sdkInt: Int): String = when {
        !WebsiteBypass.supported(sdkInt) -> "Needs Android 13 or newer"
        s.websites.isEmpty() -> "None"
        else -> s.activeWebsites.size.let { n -> if (n == 1) "1 website uses the phone's own connection" else "$n websites use the phone's own connection" }
    }

    const val WEBSITES_LIMIT =
        "tailbond goes by each website's addresses, looked up through your server when it connects and every 10 minutes, " +
            "so parts of a site loaded from other names still use tailbond, other sites sharing those addresses (big hosting " +
            "networks) skip it too, and apps with their own private DNS may not follow it."

    /** Short tile / notification text. */
    fun tileSubtitle(live: Live): String = when (live.state) {
        ConnState.CONNECTED -> activeKinds(live).ifEmpty { "Connected" }
        ConnState.CONNECTING -> "Connecting…"
        ConnState.RECONNECTING -> "Reconnecting…"
        ConnState.ERROR -> "Can't connect"
        ConnState.DISCONNECTED -> "Disconnected"
    }
}

/** "homebox": the payload's server_name, else the first label of its TLS name, else the server's host. */
fun serverShortName(p: PairingPayload): String = ServerList.nameOf(p)

/** "tail1234.ts.net" when the server has a MagicDNS name, else null. */
fun tailnetDomain(p: PairingPayload): String? =
    p.sni?.takeIf { it.endsWith(".ts.net") }?.substringAfter('.')
