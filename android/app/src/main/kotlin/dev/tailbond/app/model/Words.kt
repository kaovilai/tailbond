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
            "Stopped. Turn on to reach $server and your Tailscale machines, or hand the VPN back to Tailscale below."
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
            "tailbond tried ${ServerList.joinNames(servers)} and none of them answered, so it stopped. " +
                "Your phone is using its normal internet now, without tailbond (Android's \"Block connections without VPN\" stops that; see Settings). " +
                "Try again in a moment; if it keeps happening, " +
                "tell whoever set up your servers that ${if (servers.size == 2) "neither ${servers[0]} nor ${servers[1]}" else "none of ${ServerList.joinNames(servers)}"} answers.",
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
    /** A backup still connecting this soon after the switch reads as "Switching to…". */
    const val SWITCHING_MS = 60_000L

    /** "14:02" in the phone's time zone. */
    fun clockTime(wallMs: Long): String =
        java.time.Instant.ofEpochMilli(wallMs).atZone(java.time.ZoneId.systemDefault()).toLocalTime()
            .format(java.time.format.DateTimeFormatter.ofPattern("HH:mm"))

    /** A button with the failover note: switch to [serverId] (or stay on it), or open Android's VPN settings. */
    data class NoteAction(val label: String, val serverId: String?, val vpnSettings: Boolean = false, val filled: Boolean = false)

    /** The failover note: [line] in bold, [detail] under it, an optional button. */
    data class ServerNote(val line: String, val detail: String? = null, val action: NoteAction? = null)

    /**
     * What the home card and the notification say about which server is in use:
     * "Using vps since 14:02 (homebox is unreachable)", "Back to homebox in 1:20"
     * (with "Stay on vps"), "Back on homebox since 14:20"; null when the main
     * server is in use as normal.
     */
    fun serverNote(f: FailoverView?, names: Map<String, String>, time: (Long) -> String = ::clockTime): ServerNote? {
        f ?: return null
        fun n(id: String?) = id?.let { names[it] } ?: "the other server"
        val active = n(f.activeId)
        val primaryId = f.primaryId ?: return null
        val primary = n(primaryId)
        val since = f.sinceWallMs?.let { " since ${time(it)}" } ?: ""
        val sw = f.lastSwitch
        if (f.activeId == primaryId) {
            val recent = sw != null && sw.kind == SwitchKind.FAILBACK && sw.to == primaryId && f.nowMs - sw.atMs <= BACK_ON_MS
            return if (recent && f.state(f.activeId) == ServerState.ACTIVE) ServerNote("Back on $primary$since") else null
        }
        val switching = f.state(f.activeId) == ServerState.CONNECTING && sw != null && sw.to == f.activeId &&
            f.nowMs - sw.atMs <= SWITCHING_MS
        if (switching && f.reason != FailoverReason.MANUAL) {
            return ServerNote(
                "Switching to $active… ($primary isn't answering)",
                "For these few seconds your phone uses its normal internet, unless Android's \"Block connections without VPN\" " +
                    "is on (recommended with backup servers).",
                NoteAction("Open Android VPN settings", null, vpnSettings = true),
            )
        }
        return when (f.reason) {
            FailoverReason.MANUAL -> ServerNote(
                "Using $active$since (you picked it)",
                "Stays on $active until you pick $primary or $active stops working.",
                NoteAction("Go back to $primary", primaryId, filled = true),
            )
            FailoverReason.POLICY -> ServerNote("Always using $active", "You chose this in Settings → When to switch.")
            else -> {
                val fb = f.failbackTo
                if (fb != null && f.failbackInMs != null) {
                    ServerNote(
                        "Using $active$since (${n(fb)} is answering again)",
                        "Back to ${n(fb)} in ${mmss(f.failbackInMs)}. Calls may pause for a few seconds then. Nothing to do.",
                        NoteAction("Stay on $active", f.activeId),
                    )
                } else {
                    val why = when (f.state(primaryId)) {
                        ServerState.UNREACHABLE -> "$primary is unreachable"
                        ServerState.STANDBY -> "$primary is answering again"
                        else -> "$primary stopped working"
                    }
                    ServerNote("Using $active$since ($why)", "You're still connected. It goes back to $primary by itself once $primary works again.")
                }
            }
        }
    }

    /** One line for the notification text. */
    fun serverNoteShort(f: FailoverView?, names: Map<String, String>): String? {
        val note = serverNote(f, names) ?: return null
        val countdown = note.detail?.takeIf { it.startsWith("Back to ") }?.substringBefore('.')
        return listOfNotNull(note.line, countdown).joinToString(" · ")
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

    /** The one plain line on how switching works. */
    const val HOW_IT_DECIDES =
        "How it decides: a server counts as down when the tunnel to it has worked over none of your connections for the time above. " +
            "Waiting servers get a tiny check every 10 seconds, outside the tunnel. When the server changes, open connections start over."

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
            "${c.existing} keeps working; nothing changed on this phone. Both servers use ${c.subnet} inside the tunnel, so " +
            "tailbond can't switch between them. A new server with no devices yet moves to a free range by itself when you " +
            "ask it over Tailscale. $newName didn't (it already has devices, or it came from a link or code), so whoever runs " +
            "$newName needs to change its range, then you add it again. Copy a message to send them."
    }

    /** The message "Copy a message to send them" puts on the clipboard for a subnet clash. */
    fun clashMessage(newName: String, existing: String, subnet: Cidr): String =
        "Hi! Could you change tailbond's tunnel range on $newName? It uses $subnet, the same as $existing, so my phone " +
            "can't use both. Pick a free range such as 10.78.0.0/24: on a VPS installed with install.sh, run it again with " +
            "--subnet 10.78.0.0/24; on a GL.iNet router, use the tailbond panel or " +
            "uci set tailbond.server.subnet=10.78.0.0/24; otherwise set \"subnet\" under \"server\" in " +
            "/etc/tailbond/tailbond.json. Then restart tailbond there. Thanks!"

    // --- recent switches ---

    /** "14:02" today, "Oct 31, 14:02" before. */
    fun dayTime(ms: Long, nowMs: Long = System.currentTimeMillis()): String {
        val zone = java.time.ZoneId.systemDefault()
        val t = java.time.Instant.ofEpochMilli(ms).atZone(zone)
        val today = java.time.Instant.ofEpochMilli(nowMs).atZone(zone).toLocalDate()
        val hm = t.format(java.time.format.DateTimeFormatter.ofPattern("HH:mm"))
        return if (t.toLocalDate() == today) hm else t.format(java.time.format.DateTimeFormatter.ofPattern("MMM d", java.util.Locale.ENGLISH)) + ", " + hm
    }

    /** "homebox → vps: homebox stopped answering". */
    fun switchText(r: SwitchRecord): String = "${r.from} → ${r.to}: " + when (r.reason) {
        SwitchLog.DOWN -> "${r.from} stopped answering"
        SwitchLog.BACK -> "${r.to} was working again"
        SwitchLog.PICKED -> "you picked ${r.to}"
        else -> "your setting changed"
    }

    // --- staying protected ---

    /** Settings: what Android's "Block connections without VPN" does here. */
    fun blockAdvice(multi: Boolean): String =
        if (multi) "Recommended with backup servers: also turn on \"Block connections without VPN\". Then apps wait instead of " +
            "using the phone's normal internet for the few seconds tailbond switches servers, or while it can't reach any. " +
            "With it on, hotel Wi-Fi sign-in pages can't open and you can't hand the VPN back to Tailscale: turn it off for those."
        else "\"Block connections without VPN\" keeps apps off the phone's normal internet whenever tailbond isn't connected. " +
            "With it on, hotel Wi-Fi sign-in pages can't open and you can't hand the VPN back to Tailscale: turn it off for those."

    // --- internet traffic ---

    /** "Through your home server" with one server; with backups it names the server in use. */
    fun routeLabel(r: RouteMode, multi: Boolean): String = when {
        r == RouteMode.FULL && multi -> "Through your server"
        else -> r.label
    }

    fun routeExplain(r: RouteMode, names: List<String>): String = when {
        r == RouteMode.FULL && names.size > 1 ->
            "All internet traffic leaves from the server in use (${names[0]}, or ${ServerList.joinNames(names.drop(1), "or")} while ${names[0]} is down), like using an exit node. " +
                "A rented server's address may look like a VPN to some websites."
        names.size > 1 -> r.explain.replace("go through home", "go through the server in use").replace("looked up through home", "looked up through that server")
        else -> r.explain
    }

    // --- website bypass ---

    /** What a website rule covers, for its row. */
    fun websiteCovers(domain: String): String =
        "$domain and its usual names (" + WebsiteBypass.SUBNAMES.take(4).joinToString(", ") { "$it." } + " …)"

    /** The full list of names a rule covers. */
    fun websiteNames(domain: String): String = WebsiteBypass.namesFor(domain).joinToString(", ")

    const val WEBSITES_OTHER_NAMES =
        "Other names aren't covered: add them as their own website, or put the bank's app in Apps that skip tailbond. " +
            "(A tailbond travel router covers every name under a website; a phone can only cover names it looks up itself.)"

    const val WEBSITES_ON_ROUTER =
        "On a tailbond travel router's Wi-Fi, these phone rules apply first; the router's own rules then treat the phone like any other device."

    fun websitesSummary(s: AppSettings, sdkInt: Int): String = when {
        !WebsiteBypass.supported(sdkInt) -> "Needs Android 13 or newer"
        s.websites.isEmpty() -> "None"
        else -> s.activeWebsites.size.let { n -> if (n == 1) "1 website uses the phone's own connection" else "$n websites use the phone's own connection" }
    }

    const val WEBSITES_CHANGES =
        "Changes apply within a few seconds. tailbond looks the websites up again every 10 minutes; only when their addresses " +
            "change does the connection blip, briefly and at most once a minute (no full reconnect)."

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
