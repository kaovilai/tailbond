package dev.tailbond.app.service

import android.app.Notification
import android.app.NotificationChannel
import android.app.NotificationManager
import android.app.PendingIntent
import android.content.Context
import android.content.Intent
import android.content.pm.PackageManager
import android.content.pm.ServiceInfo
import android.net.IpPrefix
import android.os.Build
import android.os.ParcelFileDescriptor
import android.os.SystemClock
import android.util.Log
import androidx.core.app.NotificationCompat
import androidx.core.app.ServiceCompat
import androidx.core.content.ContextCompat
import androidx.core.content.getSystemService
import com.mqvpn.sdk.core.MqvpnVpnService
import com.mqvpn.sdk.core.model.MqvpnConfig
import com.mqvpn.sdk.core.model.MqvpnState
import com.mqvpn.sdk.core.model.PathInfo
import com.mqvpn.sdk.core.model.TunnelInfo
import com.mqvpn.sdk.core.model.VpnStats
import com.mqvpn.sdk.network.NetworkPath
import com.mqvpn.sdk.network.PathPolicy
import dev.tailbond.app.R
import dev.tailbond.app.data.DnsLookup
import dev.tailbond.app.data.Repository
import dev.tailbond.app.failover.FailoverEngine
import dev.tailbond.app.failover.FailoverPolicy
import dev.tailbond.app.failover.Switch
import dev.tailbond.app.failover.SwitchKind
import dev.tailbond.app.model.AppSettings
import dev.tailbond.app.model.PathKind
import dev.tailbond.app.model.RouteMode
import dev.tailbond.app.model.ServerProfile
import dev.tailbond.app.model.TunnelPlan
import dev.tailbond.app.model.WebsiteBypass
import dev.tailbond.app.model.WebsiteBypassPlan
import dev.tailbond.app.model.Words
import dev.tailbond.app.pairing.PairingPayload
import dev.tailbond.app.ui.MainActivity
import dev.tailbond.app.ui.formatBps
import kotlinx.coroutines.CoroutineScope
import kotlinx.coroutines.Dispatchers
import kotlinx.coroutines.Job
import kotlinx.coroutines.SupervisorJob
import kotlinx.coroutines.async
import kotlinx.coroutines.awaitAll
import kotlinx.coroutines.cancel
import kotlinx.coroutines.delay
import kotlinx.coroutines.flow.drop
import kotlinx.coroutines.isActive
import kotlinx.coroutines.launch
import kotlinx.coroutines.withContext
import java.net.InetAddress

/**
 * The tunnel. Started by the app (ACTION_CONNECT), the quick-settings tile,
 * or the system for always-on VPN (no extras: the stored servers are used).
 *
 * With more than one server it runs the failover engine
 * ([FailoverEngine], docs/architecture.md "Multiple servers and failover"):
 * a once-a-second tick feeds it the tunnel's health, standby servers are
 * probed every 10 s outside the tunnel, and a switch ends the session and
 * starts one to the next server (its own DNS and routes; app and website
 * bypass stay). The engine lives on the main thread only.
 */
class TailbondVpnService : MqvpnVpnService() {

    private lateinit var repo: Repository
    private val gate = PathGate()
    private val main = CoroutineScope(SupervisorJob() + Dispatchers.Main.immediate)
    private val clock: () -> Long = { SystemClock.elapsedRealtime() }

    /** What the running session was built from; null between sessions. */
    @Volatile private var session: Session? = null
    @Volatile private var restartPending = false
    @Volatile private var stopping = false

    private data class Session(val server: ServerProfile, val settings: AppSettings, val config: MqvpnConfig)

    // --- failover (main thread) ---
    /** True from connect until disconnect or giving up; sessions come and go inside it. */
    private var running = false
    private var engine: FailoverEngine? = null
    private var servers: List<ServerProfile> = emptyList()
    /** Server for the next session after a restart. */
    @Volatile private var pendingServer: String? = null
    /** A server error ended the session: try again (or switch) at this time. */
    private var retryAt = 0L
    /** Servers whose session ended in an error since the tunnel last worked. */
    private val failedSinceWorking = HashSet<String>()
    private var tickJob: Job? = null
    private var probeJob: Job? = null
    private lateinit var prober: ServerProber

    // --- website bypass ---
    private val bypass = WebsiteBypassPlan(clock)
    private var bypassJob: Job? = null

    // metrics state (engine thread only)
    private val txMeter = RateMeter()
    private val rxMeter = RateMeter()
    private val pathTx = HashMap<String, RateMeter>()
    private val pathRx = HashMap<String, RateMeter>()
    private val pathHist = HashMap<String, List<Long>>()
    private val cellLastBytes = HashMap<Long, Long>()
    private var lastSent = -1L
    private var lastLost = -1L
    private var ticks = 0L
    private var lastCapReached = false

    override fun onCreate() {
        super.onCreate()
        repo = Repository.get(this)
        prober = ServerProber(this) { s -> protect(s) }
        createChannel()
        main.launch {
            // live settings changes: re-gate paths, or rebuild the session when needed
            repo.settings.drop(1).collect { onSettingsChanged(it) }
        }
        main.launch { repo.servers.drop(1).collect { onServersChanged(it) } }
    }

    override fun onStartCommand(intent: Intent?, flags: Int, startId: Int): Int {
        when (intent?.action) {
            ACTION_DISCONNECT -> {
                stopping = true
                restartPending = false
                endRun()
                if (session != null) stopTunnel() else finish()
                return START_NOT_STICKY
            }
            ACTION_USE_SERVER -> {
                val id = intent.getStringExtra(EXTRA_SERVER)
                if (running && id != null) {
                    engine?.useNow(id)?.let { switchTo(it) }
                    return START_STICKY
                }
                // not on: nothing to switch (and don't leave an idle service behind)
                if (session == null) stopSelf()
                return START_NOT_STICKY
            }
        }
        val list = repo.servers.value
        if (list.isEmpty()) {
            LiveState.reset(ConnState.ERROR, getString(R.string.err_not_paired), ErrorKind.NOT_PAIRED)
            finish()
            return START_NOT_STICKY
        }
        ServiceCompat.startForeground(
            this, NOTIFICATION_ID, buildNotification(getString(R.string.state_connecting), list.first().name),
            if (Build.VERSION.SDK_INT >= 34) ServiceInfo.FOREGROUND_SERVICE_TYPE_SPECIAL_USE else 0,
        )
        stopping = false
        // turning tailbond on again ends a "Switch back to Tailscale"
        if (repo.settings.value.handedToTailscale) repo.updateSettings { it.copy(handedToTailscale = false) }
        if (!running && session == null) startRun(list)
        return START_STICKY
    }

    private fun startRun(list: List<ServerProfile>) {
        val s = repo.settings.value
        servers = list
        running = true
        failedSinceWorking.clear()
        retryAt = 0
        val e = FailoverEngine(clock, list.map { it.id }, s.failoverPolicy, s.alwaysServerId, s.failoverTiming)
        engine = e
        bypass.retain(s.activeWebsites)
        tickJob?.cancel()
        tickJob = main.launch {
            while (isActive) {
                delay(1_000)
                tick()
            }
        }
        startProbes()
        begin(serverById(e.active) ?: list.first())
    }

    private fun endRun() {
        running = false
        tickJob?.cancel(); tickJob = null
        probeJob?.cancel(); probeJob = null
        bypassJob?.cancel(); bypassJob = null
        engine = null
        pendingServer = null
    }

    private fun serverById(id: String?): ServerProfile? = servers.firstOrNull { it.id == id }

    private fun begin(server: ServerProfile) {
        val settings = repo.settings.value
        val config = try {
            TunnelPlan.config(server.payload, settings)
        } catch (e: Exception) {
            endRun()
            LiveState.reset(ConnState.ERROR, e.message, ErrorKind.OTHER)
            finish()
            return
        }
        session = Session(server, settings, config)
        pendingServer = null
        gate.configure(settings, TunnelPlan.effectiveMode(server.payload, settings), repo.cellularCapReached())
        LiveState.restart(ConnState.CONNECTING)
        publishFailover()
        notifyText(getString(R.string.state_connecting), server.name)
        startTunnel(config)
    }

    // --- failover ---

    private fun multi(): Boolean = servers.size > 1

    /** Once a second: health in, switch decisions out. */
    private fun tick() {
        val e = engine ?: return
        if (!running) return
        val live = LiveState.live.value
        val working = session != null && live.state == ConnState.CONNECTED && live.paths.any { it.status == PathGate.STATUS_ACTIVE }
        e.reportHealth(working)
        if (working) failedSinceWorking.clear()
        val sw = e.tick()
        when {
            sw != null && sw.kind == SwitchKind.FAILOVER && session == null &&
                servers.all { it.id in failedSinceWorking } -> giveUp()
            sw != null -> switchTo(sw)
            session == null && !restartPending && clock() >= retryAt -> serverById(e.active)?.let { begin(it) }
        }
        publishFailover()
    }

    private fun switchTo(sw: Switch) {
        val to = serverById(sw.to) ?: return
        Log.i(TAG, "server switch ${sw.kind}: ${serverById(sw.from)?.name} -> ${to.name}")
        if (session != null) {
            pendingServer = to.id
            restartPending = true
            stopTunnel()
        } else {
            begin(to)
        }
        publishFailover()
    }

    /** Every server failed in a row: stop like the one-server app does, with the error. */
    private fun giveUp() {
        val live = LiveState.live.value
        val view = engine?.view()
        endRun()
        LiveState.reset(ConnState.ERROR, live.message, live.errorKind ?: ErrorKind.UNREACHABLE)
        LiveState.set { it.copy(failover = view) }
        finish()
    }

    private fun publishFailover() {
        val e = engine
        val view = if (e != null && multi()) e.view().let { v ->
            v.copy(sinceWallMs = v.lastSwitch?.let { System.currentTimeMillis() - (v.nowMs - it.atMs) })
        } else null
        LiveState.set { it.copy(failover = view) }
    }

    private fun startProbes() {
        probeJob?.cancel()
        probeJob = main.launch {
            while (isActive) {
                val e = engine ?: break
                if (multi()) {
                    val targets = servers.filter { it.id != e.active }
                    val results = withContext(Dispatchers.IO) {
                        targets.map { s ->
                            async { s.id to runCatching { prober.probe(s.payload.host, s.payload.port) }.getOrDefault(false) }
                        }.awaitAll()
                    }
                    results.forEach { (id, ok) -> engine?.reportProbe(id, ok) }
                    publishFailover()
                }
                delay(e.timing.probeEveryMs)
            }
        }
    }

    private fun onServersChanged(list: List<ServerProfile>) {
        if (!running) return
        if (list.isEmpty()) {
            stopping = true
            endRun()
            if (session != null) stopTunnel() else finish()
            return
        }
        servers = list
        engine?.setServers(list.map { it.id })
        // the server in use was paired again: reconnect with its new pairing
        val s = session
        val now = s?.let { cur -> list.firstOrNull { it.id == cur.server.id } }
        if (s != null && now != null && now.payload != s.server.payload) {
            pendingServer = now.id
            restartPending = true
            stopTunnel()
        }
        publishFailover()
    }

    // --- settings ---

    private fun onSettingsChanged(next: AppSettings) {
        engine?.configure(next.failoverPolicy, next.alwaysServerId, next.failoverTiming)
        val s = session ?: return
        val profile = s.server.payload
        if (needsRestart(profile, s.settings, next)) {
            Log.i(TAG, "settings change needs a new session")
            pendingServer = s.server.id
            restartPending = true
            stopTunnel()
            return
        }
        val websitesChanged = s.settings.activeWebsites != next.activeWebsites
        session = s.copy(settings = next)
        gate.configure(next, TunnelPlan.effectiveMode(profile, next), repo.cellularCapReached())
        requestPathReconcile()
        if (websitesChanged) {
            bypass.retain(next.activeWebsites)
            if (LiveState.live.value.state == ConnState.CONNECTED) startBypassLoop(force = true)
        }
    }

    /** Changes the running session can't absorb by attaching/detaching paths. */
    internal fun needsRestart(p: PairingPayload, old: AppSettings, new: AppSettings): Boolean {
        if (TunnelPlan.effectiveMode(p, old) != TunnelPlan.effectiveMode(p, new)) return true
        if (old.routeMode != new.routeMode || old.excludedApps != new.excludedApps) return true
        if (old.experimentalBackupFec != new.experimentalBackupFec) return true
        // a transport the session doesn't hold can't be attached without a new monitor
        return PathKind.entries.any { !old.path(it).enabled && new.path(it).enabled }
    }

    // --- website bypass ---

    /**
     * Resolves every bypassed website through the tunnel's DNS now and every
     * 10 minutes while connected; new addresses rebuild the TUN (rate-limited
     * by [WebsiteBypassPlan]). Android 13+ only.
     */
    private fun startBypassLoop(force: Boolean) {
        bypassJob?.cancel()
        if (!WebsiteBypass.supported(Build.VERSION.SDK_INT)) return
        if (session == null) return
        bypassJob = main.launch {
            var first = force
            while (isActive) {
                val cur = session ?: break
                val domains = cur.settings.activeWebsites
                if (domains.isNotEmpty() && cur.settings.routeMode == RouteMode.FULL) {
                    val dns = cur.config.dnsServers.firstOrNull() ?: break
                    withContext(Dispatchers.IO) {
                        for (d in domains) {
                            val addrs = WebsiteBypass.namesFor(d).flatMap { n -> runCatching { DnsLookup.resolve(n, dns) }.getOrDefault(emptyList()) }
                            bypass.record(d, addrs)
                        }
                    }
                }
                if (session != null && bypass.decide(force = first) != null) {
                    Log.i(TAG, "website bypass: rebuilding the TUN with new addresses")
                    rebuildTun()
                }
                first = false
                delay(WebsiteBypass.REFRESH_MS)
            }
        }
    }

    // --- MqvpnVpnService hooks ---

    override fun pathPolicy(config: MqvpnConfig): PathPolicy =
        PathPolicy(config.wifiPathEnabled, config.cellularPathEnabled, config.ethernetPathEnabled, bluetooth = false)

    override fun isPathWanted(path: NetworkPath): Boolean {
        val kind = PathKind.of(path.type) ?: return true // OTHER (e.g. USB tether): no per-kind setting
        return gate.wanted(PathGate.Link(path.name, kind))
    }

    override fun onCreateTun(info: TunnelInfo, config: MqvpnConfig): ParcelFileDescriptor {
        val s = session ?: throw IllegalStateException("no session")
        val profile = s.server.payload
        val b = Builder()
            .setSession(getString(R.string.app_name))
            .addAddress(info.assignedIp, info.prefix)
            .setMtu(info.mtu)
            .setBlocking(true)
            .setConfigureIntent(activityIntent())
        val ip6 = info.assignedIp6
        val hasV6 = info.hasV6 && ip6 != null
        if (hasV6) b.addAddress(ip6, info.prefix6)
        for (r in TunnelPlan.routes(profile, s.settings, config.dnsServers)) {
            // Full tunnel keeps ::/0 even without a tunnel v6 address, so IPv6 can't leak around it.
            if (r.isV6 && !hasV6 && s.settings.routeMode != RouteMode.FULL) continue
            b.addRoute(r.address, r.prefix)
        }
        var excluded = 0
        val wanted = bypass.wanted()
        if (Build.VERSION.SDK_INT >= 33) {
            for (r in TunnelPlan.excludedRoutes(profile, s.settings, config.dnsServers, wanted, Build.VERSION.SDK_INT)) {
                try {
                    b.excludeRoute(IpPrefix(InetAddress.getByName(r.address), r.prefix))
                    excluded++
                } catch (e: IllegalArgumentException) {
                    Log.w(TAG, "can't exclude $r: ${e.message}")
                }
            }
        }
        bypass.markApplied(wanted)
        config.dnsServers.forEach { b.addDnsServer(it) }
        for (pkg in s.settings.excludedApps) {
            try {
                b.addDisallowedApplication(pkg)
            } catch (_: PackageManager.NameNotFoundException) {
                Log.w(TAG, "excluded app $pkg is not installed")
            }
        }
        val pfd = b.establish() ?: throw IllegalStateException("VPN permission not granted")
        LiveState.set { it.copy(bypassAddresses = excluded) }
        return pfd
    }

    override fun onVpnStateChanged(newState: MqvpnState) {
        when (newState) {
            is MqvpnState.Connecting -> LiveState.set { it.copy(state = ConnState.CONNECTING, message = null, errorKind = null) }
            is MqvpnState.Reconnecting -> {
                LiveState.set { it.copy(state = ConnState.RECONNECTING, message = "retry in ${newState.info.delaySec}s") }
                notifyText(getString(R.string.state_reconnecting))
            }
            is MqvpnState.Connected -> {
                val t = newState.tunnelInfo
                LiveState.set {
                    it.copy(state = ConnState.CONNECTED, message = null, errorKind = null, tunnelIp = t.assignedIp, connectedAtMs = t.connectedAtMs)
                }
                notifyText(getString(R.string.notif_connected, session?.server?.name ?: getString(R.string.home_server)))
                main.launch { startBypassLoop(force = true) }
            }
            is MqvpnState.Error -> {
                val kind = ErrorKind.of(newState.error)
                val failed = session?.server
                session = null
                restartPending = false
                main.launch { bypassJob?.cancel() }
                val serverError = kind == ErrorKind.UNREACHABLE || kind == ErrorKind.IDENTITY || kind == ErrorKind.REJECTED
                if (serverError && multi() && !stopping && repo.settings.value.failoverPolicy == FailoverPolicy.PREFER) {
                    // the failover engine decides: try this server again, or switch after switch_after
                    main.launch {
                        if (!running) return@launch
                        failed?.let { failedSinceWorking += it.id }
                        retryAt = clock() + RETRY_MS
                        LiveState.set { it.copy(state = ConnState.CONNECTING, message = newState.error.message, errorKind = kind) }
                    }
                } else {
                    main.launch { endRun() }
                    LiveState.reset(ConnState.ERROR, newState.error.message, kind)
                    main.launch { finish() }
                }
            }
            is MqvpnState.Disconnected -> {
                val restart = restartPending && !stopping
                val next = pendingServer
                session = null
                restartPending = false
                resetMetrics()
                main.launch { bypassJob?.cancel() }
                if (restart) {
                    main.launch {
                        val server = serverById(next) ?: serverById(engine?.active) ?: repo.servers.value.firstOrNull()
                        if (server != null) begin(server) else finish()
                    }
                } else {
                    main.launch { endRun() }
                    LiveState.reset()
                    main.launch { finish() }
                }
            }
        }
    }

    override fun onLog(level: Int, message: String) {
        when (level) {
            0 -> Log.d(TAG, message)
            1 -> Log.i(TAG, message)
            2 -> Log.w(TAG, message)
            else -> Log.e(TAG, message)
        }
    }

    // --- metrics (engine thread, once a second) ---

    private fun resetMetrics() {
        pathTx.clear(); pathRx.clear(); pathHist.clear(); cellLastBytes.clear()
        lastSent = -1; lastLost = -1
    }

    override fun onMetrics(stats: VpnStats, paths: List<PathInfo>) {
        val now = SystemClock.elapsedRealtime()
        ticks++
        val txBps = txMeter.sample(stats.bytesTx, now)
        val rxBps = rxMeter.sample(stats.bytesRx, now)

        val dSent = if (lastSent >= 0) stats.dgramSent - lastSent else 0
        val dLost = if (lastLost >= 0) stats.dgramLost - lastLost else 0
        lastSent = stats.dgramSent
        lastLost = stats.dgramLost
        val loss = if (dSent > 0) (dLost.coerceAtLeast(0) * 100.0 / dSent).coerceAtMost(100.0) else 0.0

        // cellular data counter
        var cellDelta = 0L
        for (p in paths) {
            if (PathKind.ofPathName(p.iface) != PathKind.CELLULAR) continue
            val total = p.bytesTx + p.bytesRx
            val prev = cellLastBytes.put(p.handle, total)
            if (prev != null && total > prev) cellDelta += total - prev
        }
        repo.addCellularBytes(cellDelta, persist = ticks % 10 == 0L)
        val cap = repo.cellularCapReached()

        // one entry per name: a released slot (CLOSED) can linger next to a re-added path of the same name
        val live = paths.groupBy { it.iface }.values.map { same -> same.minBy { it.status == PathGate.STATUS_CLOSED } }
        val samples = live.map { p ->
            val tx = pathTx.getOrPut(p.iface) { RateMeter() }.sample(p.bytesTx, now)
            val rx = pathRx.getOrPut(p.iface) { RateMeter() }.sample(p.bytesRx, now)
            val hist = (pathHist[p.iface] ?: emptyList()).pushBounded(tx + rx)
            pathHist[p.iface] = hist
            PathSample(p.iface, PathKind.ofPathName(p.iface), p.status, p.srttMs, tx, rx, p.bytesTx, p.bytesRx, hist)
        }
        val attachedNames = live.map { it.iface }.toSet()
        val available = availableNetworks()
        val held = available.filter { it.name !in attachedNames }.map { n ->
            PathSample(n.name, PathKind.of(n.type), -1, 0, 0, 0, 0, 0)
        }

        // path policy: standby links and the cellular cap
        val s = session
        var reconcile = false
        if (s != null && cap != lastCapReached) {
            lastCapReached = cap
            gate.configure(s.settings, TunnelPlan.effectiveMode(s.server.payload, s.settings), cap)
            reconcile = true
        }
        val links = available.mapNotNull { n -> PathKind.of(n.type)?.let { PathGate.Link(n.name, it) } }
        if (gate.update(now, links, live.associate { it.iface to it.status })) reconcile = true
        if (reconcile) requestPathReconcile()

        LiveState.set {
            it.copy(
                txBps = txBps, rxBps = rxBps, txBytes = stats.bytesTx, rxBytes = stats.bytesRx,
                lossPct = loss, srttMs = stats.srttMs,
                txHistory = it.txHistory.pushBounded(txBps), rxHistory = it.rxHistory.pushBounded(rxBps),
                paths = samples + held, standbyActive = gate.standbyActive,
            )
        }
        if (ticks % 2 == 0L) {
            val s2 = session
            val lv = LiveState.live.value
            val summary = if (s2 != null) {
                Words.connectionsSummary(lv, s2.settings, TunnelPlan.effectiveMode(s2.server.payload, s2.settings), cap)
            } else null
            val note = Words.serverNoteShort(lv.failover, servers.associate { it.id to it.name })
            notifyText(
                getString(R.string.notif_connected, s2?.server?.name ?: getString(R.string.home_server)),
                listOfNotNull(note, summary, "↓ ${formatBps(rxBps)}  ↑ ${formatBps(txBps)}").joinToString(" · "),
            )
        }
    }

    // --- lifecycle helpers ---

    private fun finish() {
        if (ticks > 0) repo.addCellularBytes(0, persist = true)
        ServiceCompat.stopForeground(this, ServiceCompat.STOP_FOREGROUND_REMOVE)
        getSystemService<NotificationManager>()?.cancel(NOTIFICATION_ID)
        stopSelf()
    }

    override fun onRevoke() {
        // another VPN took over, or the user revoked us
        stopping = true
        session = null
        main.launch { endRun() }
        LiveState.reset()
        super.onRevoke()
    }

    override fun onDestroy() {
        main.cancel()
        if (LiveState.live.value.state != ConnState.ERROR) LiveState.reset()
        super.onDestroy()
    }

    // --- notification ---

    private fun createChannel() {
        getSystemService<NotificationManager>()?.createNotificationChannel(
            NotificationChannel(CHANNEL_ID, getString(R.string.notification_channel), NotificationManager.IMPORTANCE_LOW)
                .apply { setShowBadge(false) },
        )
    }

    private fun activityIntent(): PendingIntent = PendingIntent.getActivity(
        this, 0, Intent(this, MainActivity::class.java).addFlags(Intent.FLAG_ACTIVITY_SINGLE_TOP),
        PendingIntent.FLAG_IMMUTABLE or PendingIntent.FLAG_UPDATE_CURRENT,
    )

    private fun buildNotification(title: String, text: String?): Notification {
        val stop = PendingIntent.getService(
            this, 1, Intent(this, TailbondVpnService::class.java).setAction(ACTION_DISCONNECT),
            PendingIntent.FLAG_IMMUTABLE or PendingIntent.FLAG_UPDATE_CURRENT,
        )
        return NotificationCompat.Builder(this, CHANNEL_ID)
            .setSmallIcon(R.drawable.ic_stat_tailbond)
            .setContentTitle(title)
            .setContentText(text)
            .setStyle(NotificationCompat.BigTextStyle().bigText(text))
            .setOngoing(true)
            .setOnlyAlertOnce(true)
            .setSilent(true)
            .setShowWhen(false)
            .setCategory(NotificationCompat.CATEGORY_SERVICE)
            .setContentIntent(activityIntent())
            .addAction(0, getString(R.string.action_disconnect), stop)
            .build()
    }

    private fun notifyText(title: String, text: String? = null) {
        if (stopping) return
        getSystemService<NotificationManager>()?.notify(NOTIFICATION_ID, buildNotification(title, text))
    }

    companion object {
        private const val TAG = "tailbond"
        private const val CHANNEL_ID = "tunnel"
        private const val NOTIFICATION_ID = 1
        /** After a server error, try again (or let failover switch) this much later. */
        private const val RETRY_MS = 3_000L
        const val ACTION_CONNECT = "dev.tailbond.app.CONNECT"
        const val ACTION_DISCONNECT = "dev.tailbond.app.DISCONNECT"
        const val ACTION_USE_SERVER = "dev.tailbond.app.USE_SERVER"
        const val EXTRA_SERVER = "server"

        /** VPN permission must already be granted (VpnService.prepare == null). */
        fun connect(ctx: Context) {
            ContextCompat.startForegroundService(
                ctx, Intent(ctx, TailbondVpnService::class.java).setAction(ACTION_CONNECT),
            )
        }

        fun disconnect(ctx: Context) {
            ctx.startService(Intent(ctx, TailbondVpnService::class.java).setAction(ACTION_DISCONNECT))
        }

        /** "Use this server now"; only acts while tailbond is on. */
        fun useServer(ctx: Context, id: String) {
            ctx.startService(Intent(ctx, TailbondVpnService::class.java).setAction(ACTION_USE_SERVER).putExtra(EXTRA_SERVER, id))
        }
    }
}
