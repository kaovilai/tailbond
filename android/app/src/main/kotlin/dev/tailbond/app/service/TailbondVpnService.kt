package dev.tailbond.app.service

import android.app.Notification
import android.app.NotificationChannel
import android.app.NotificationManager
import android.app.PendingIntent
import android.content.Context
import android.content.Intent
import android.content.pm.PackageManager
import android.content.pm.ServiceInfo
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
import dev.tailbond.app.data.Repository
import dev.tailbond.app.model.AppSettings
import dev.tailbond.app.model.PathKind
import dev.tailbond.app.model.RouteMode
import dev.tailbond.app.model.TunnelPlan
import dev.tailbond.app.model.Words
import dev.tailbond.app.model.serverShortName
import dev.tailbond.app.pairing.PairingPayload
import dev.tailbond.app.ui.MainActivity
import dev.tailbond.app.ui.formatBps
import kotlinx.coroutines.CoroutineScope
import kotlinx.coroutines.Dispatchers
import kotlinx.coroutines.SupervisorJob
import kotlinx.coroutines.cancel
import kotlinx.coroutines.flow.drop
import kotlinx.coroutines.launch

/**
 * The tunnel. Started by the app (ACTION_CONNECT), the quick-settings tile,
 * or the system for always-on VPN (no extras: the stored pairing is used).
 */
class TailbondVpnService : MqvpnVpnService() {

    private lateinit var repo: Repository
    private val gate = PathGate()
    private val main = CoroutineScope(SupervisorJob() + Dispatchers.Main.immediate)

    /** What the running session was built from; null between sessions. */
    @Volatile private var session: Session? = null
    @Volatile private var restartPending = false
    @Volatile private var stopping = false

    private data class Session(val profile: PairingPayload, val settings: AppSettings, val config: MqvpnConfig)

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
        createChannel()
        main.launch {
            // live settings changes: re-gate paths, or rebuild the session when needed
            repo.settings.drop(1).collect { onSettingsChanged(it) }
        }
    }

    override fun onStartCommand(intent: Intent?, flags: Int, startId: Int): Int {
        if (intent?.action == ACTION_DISCONNECT) {
            stopping = true
            restartPending = false
            if (session != null) stopTunnel() else finish()
            return START_NOT_STICKY
        }
        val profile = repo.profile.value
        if (profile == null) {
            LiveState.reset(ConnState.ERROR, getString(R.string.err_not_paired), ErrorKind.NOT_PAIRED)
            finish()
            return START_NOT_STICKY
        }
        ServiceCompat.startForeground(
            this, NOTIFICATION_ID, buildNotification(getString(R.string.state_connecting), serverName(profile)),
            if (Build.VERSION.SDK_INT >= 34) ServiceInfo.FOREGROUND_SERVICE_TYPE_SPECIAL_USE else 0,
        )
        stopping = false
        // turning tailbond on again ends a "Switch back to Tailscale"
        if (repo.settings.value.handedToTailscale) repo.updateSettings { it.copy(handedToTailscale = false) }
        if (session == null) begin(profile, repo.settings.value)
        return START_STICKY
    }

    private fun begin(profile: PairingPayload, settings: AppSettings) {
        val config = try {
            TunnelPlan.config(profile, settings)
        } catch (e: Exception) {
            LiveState.reset(ConnState.ERROR, e.message, ErrorKind.OTHER)
            finish()
            return
        }
        session = Session(profile, settings, config)
        gate.configure(settings, TunnelPlan.effectiveMode(profile, settings), repo.cellularCapReached())
        LiveState.reset(ConnState.CONNECTING)
        startTunnel(config)
    }

    private fun onSettingsChanged(next: AppSettings) {
        val s = session ?: return
        val profile = s.profile
        if (needsRestart(profile, s.settings, next)) {
            Log.i(TAG, "settings change needs a new session")
            restartPending = true
            stopTunnel()
            return
        }
        session = s.copy(settings = next)
        gate.configure(next, TunnelPlan.effectiveMode(profile, next), repo.cellularCapReached())
        requestPathReconcile()
    }

    /** Changes the running session can't absorb by attaching/detaching paths. */
    internal fun needsRestart(p: PairingPayload, old: AppSettings, new: AppSettings): Boolean {
        if (TunnelPlan.effectiveMode(p, old) != TunnelPlan.effectiveMode(p, new)) return true
        if (old.routeMode != new.routeMode || old.excludedApps != new.excludedApps) return true
        if (old.experimentalBackupFec != new.experimentalBackupFec) return true
        // a transport the session doesn't hold can't be attached without a new monitor
        return PathKind.entries.any { !old.path(it).enabled && new.path(it).enabled }
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
        val b = Builder()
            .setSession(getString(R.string.app_name))
            .addAddress(info.assignedIp, info.prefix)
            .setMtu(info.mtu)
            .setBlocking(true)
            .setConfigureIntent(activityIntent())
        val ip6 = info.assignedIp6
        val hasV6 = info.hasV6 && ip6 != null
        if (hasV6) b.addAddress(ip6, info.prefix6)
        for (r in TunnelPlan.routes(s.profile, s.settings, config.dnsServers)) {
            // Full tunnel keeps ::/0 even without a tunnel v6 address, so IPv6 can't leak around it.
            if (r.isV6 && !hasV6 && s.settings.routeMode != RouteMode.FULL) continue
            b.addRoute(r.address, r.prefix)
        }
        config.dnsServers.forEach { b.addDnsServer(it) }
        for (pkg in s.settings.excludedApps) {
            try {
                b.addDisallowedApplication(pkg)
            } catch (_: PackageManager.NameNotFoundException) {
                Log.w(TAG, "excluded app $pkg is not installed")
            }
        }
        return b.establish() ?: throw IllegalStateException("VPN permission not granted")
    }

    override fun onVpnStateChanged(newState: MqvpnState) {
        when (newState) {
            is MqvpnState.Connecting -> LiveState.set { it.copy(state = ConnState.CONNECTING, message = null) }
            is MqvpnState.Reconnecting -> {
                LiveState.set { it.copy(state = ConnState.RECONNECTING, message = "retry in ${newState.info.delaySec}s") }
                notifyText(getString(R.string.state_reconnecting))
            }
            is MqvpnState.Connected -> {
                val t = newState.tunnelInfo
                LiveState.set {
                    it.copy(state = ConnState.CONNECTED, message = null, tunnelIp = t.assignedIp, connectedAtMs = t.connectedAtMs)
                }
                notifyText(getString(R.string.notif_connected, serverName(repo.profile.value)))
            }
            is MqvpnState.Error -> {
                session = null
                restartPending = false
                LiveState.reset(ConnState.ERROR, newState.error.message, ErrorKind.of(newState.error))
                main.launch { finish() }
            }
            is MqvpnState.Disconnected -> {
                val restart = restartPending && !stopping
                session = null
                restartPending = false
                resetMetrics()
                if (restart) {
                    val profile = repo.profile.value
                    main.launch { if (profile != null) begin(profile, repo.settings.value) else finish() }
                } else {
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
            gate.configure(s.settings, TunnelPlan.effectiveMode(s.profile, s.settings), cap)
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
            val summary = if (s2 != null) {
                Words.connectionsSummary(LiveState.live.value, s2.settings, TunnelPlan.effectiveMode(s2.profile, s2.settings), cap)
            } else null
            notifyText(
                getString(R.string.notif_connected, serverName(s2?.profile)),
                listOfNotNull(summary, "↓ ${formatBps(rxBps)}  ↑ ${formatBps(txBps)}").joinToString(" · "),
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

    private fun serverName(p: PairingPayload?): String = p?.let { serverShortName(it) } ?: getString(R.string.home_server)

    companion object {
        private const val TAG = "tailbond"
        private const val CHANNEL_ID = "tunnel"
        private const val NOTIFICATION_ID = 1
        const val ACTION_CONNECT = "dev.tailbond.app.CONNECT"
        const val ACTION_DISCONNECT = "dev.tailbond.app.DISCONNECT"

        /** VPN permission must already be granted (VpnService.prepare == null). */
        fun connect(ctx: Context) {
            ContextCompat.startForegroundService(
                ctx, Intent(ctx, TailbondVpnService::class.java).setAction(ACTION_CONNECT),
            )
        }

        fun disconnect(ctx: Context) {
            ctx.startService(Intent(ctx, TailbondVpnService::class.java).setAction(ACTION_DISCONNECT))
        }
    }
}
