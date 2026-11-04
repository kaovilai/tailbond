package dev.tailbond.app.share

import android.app.Notification
import android.app.NotificationChannel
import android.app.NotificationManager
import android.app.PendingIntent
import android.app.Service
import android.content.Context
import android.content.Intent
import android.content.IntentFilter
import android.content.pm.ServiceInfo
import android.net.ConnectivityManager
import android.net.LinkProperties
import android.net.Network
import android.net.NetworkCapabilities
import android.net.NetworkRequest
import android.net.nsd.NsdManager
import android.net.nsd.NsdServiceInfo
import android.os.BatteryManager
import android.os.Build
import android.os.IBinder
import android.util.Log
import androidx.core.app.NotificationCompat
import androidx.core.app.ServiceCompat
import androidx.core.content.ContextCompat
import androidx.core.content.getSystemService
import dev.tailbond.app.R
import dev.tailbond.app.data.PairRequestClient
import dev.tailbond.app.data.PeersClient
import dev.tailbond.app.data.Repository
import dev.tailbond.app.model.ServerList
import dev.tailbond.app.service.ConnState
import dev.tailbond.app.service.LiveState
import dev.tailbond.app.service.TailbondVpnService
import dev.tailbond.app.share.ShareProtocol as P
import dev.tailbond.app.ui.MainActivity
import kotlinx.coroutines.flow.MutableStateFlow
import kotlinx.coroutines.flow.StateFlow
import kotlinx.coroutines.flow.asStateFlow
import kotlinx.serialization.json.JsonArray
import kotlinx.serialization.json.JsonPrimitive
import kotlinx.serialization.json.buildJsonObject
import kotlinx.serialization.json.put
import java.net.DatagramSocket
import java.net.Inet4Address
import java.net.InetAddress
import java.net.InetSocketAddress

/** What "Share my mobile data" is doing right now. */
enum class ShareStatus {
    OFF,
    /** Started; waiting for mobile data and Wi-Fi. */
    STARTING,
    /** Ready; no hub uses it yet. */
    WAITING,
    SHARING,
    /** Mobile data isn't available (off, no signal). */
    NO_CELLULAR,
    /** Not on a Wi-Fi network: nothing nearby to share with. */
    NO_WIFI,
    /** Stopped: battery below the limit, not charging. */
    BATTERY,
    /** Stopped: this month's sharing limit is used up. */
    CAP,
    /** Not paired, or the pairing predates sharing and the server hasn't been reached yet. */
    NO_KEY,
}

data class ShareLive(
    val status: ShareStatus = ShareStatus.OFF,
    val hubs: List<ShareRelay.HubUse> = emptyList(),
    /** This month's shared bytes. */
    val usedBytes: Long = 0,
    val battery: Int = -1,
    val charging: Boolean = false,
    val label: String = "",
)

/** In-process share state, written by [ShareService], read by the UI. */
object ShareState {
    private val _live = MutableStateFlow(ShareLive())
    val live: StateFlow<ShareLive> = _live.asStateFlow()
    fun set(f: (ShareLive) -> ShareLive) {
        while (true) {
            val cur = _live.value
            if (_live.compareAndSet(cur, f(cur))) return
        }
    }
}

/**
 * "Share my mobile data" (Pair & Share): a hub on this phone's Wi-Fi (the
 * travel router, a laptop) bonds the phone's cellular data. The relay
 * listens on the Wi-Fi network and sends upstream over cellular only; both
 * sockets are bound to their network and kept out of tailbond's own VPN when
 * it runs, so nothing loops through our tunnel. Works with the VPN off.
 */
class ShareService : Service() {
    private lateinit var repo: Repository
    private lateinit var cm: ConnectivityManager
    @Volatile private var cellular: Network? = null
    @Volatile private var wifi: Network? = null
    private var relay: ShareRelay? = null
    private var relayThread: Thread? = null
    private var nsd: NsdManager.RegistrationListener? = null
    private var lastNotify = 0L
    private var lastRegister = 0L
    private var ticker: Thread? = null
    @Volatile private var running = false
    /** The month the running relay counts in, and the limits it was given. */
    private var relayMonth = ""
    private var appliedCap = 0L
    private var appliedHubs = emptySet<String>()
    /** Stopped for want of a share key (an older pairing, not yet fetched). */
    private var waitingForKey = false

    private val cellCb = object : ConnectivityManager.NetworkCallback() {
        override fun onAvailable(network: Network) {
            cellular = network
            restartRelay()
        }

        override fun onLost(network: Network) {
            if (cellular == network) {
                cellular = null
                restartRelay()
            }
        }
    }
    private val wifiCb = object : ConnectivityManager.NetworkCallback() {
        override fun onAvailable(network: Network) {
            wifi = network
            restartRelay()
        }

        override fun onLost(network: Network) {
            if (wifi == network) {
                wifi = null
                restartRelay()
            }
        }
    }

    override fun onBind(intent: Intent?): IBinder? = null

    override fun onCreate() {
        super.onCreate()
        repo = Repository.get(this)
        cm = getSystemService()!!
        getSystemService<NotificationManager>()?.createNotificationChannel(
            NotificationChannel(CHANNEL, getString(R.string.share_channel), NotificationManager.IMPORTANCE_LOW),
        )
    }

    override fun onStartCommand(intent: Intent?, flags: Int, startId: Int): Int {
        if (intent?.action == ACTION_STOP) {
            stopSharing()
            return START_NOT_STICKY
        }
        ServiceCompat.startForeground(
            this, NOTIFICATION_ID, notification(getString(R.string.share_starting), null),
            if (Build.VERSION.SDK_INT >= 34) ServiceInfo.FOREGROUND_SERVICE_TYPE_SPECIAL_USE else 0,
        )
        if (!running) {
            running = true
            ShareState.set { it.copy(status = ShareStatus.STARTING, usedBytes = repo.sharedBytes(), label = label()) }
            // holding a cellular request keeps mobile data up next to Wi-Fi
            cm.requestNetwork(
                NetworkRequest.Builder().addTransportType(NetworkCapabilities.TRANSPORT_CELLULAR)
                    .addCapability(NetworkCapabilities.NET_CAPABILITY_INTERNET).build(),
                cellCb,
            )
            cm.registerNetworkCallback(
                NetworkRequest.Builder().addTransportType(NetworkCapabilities.TRANSPORT_WIFI).build(),
                wifiCb,
            )
            ticker = Thread({ tickLoop() }, "share-tick").apply { isDaemon = true; start() }
        }
        return START_STICKY
    }

    /** "Mobile data from sam-pixel": this phone's name on the home server, the same everywhere. */
    private fun label(): String {
        val p = repo.servers.value.firstOrNull()?.payload
        val name = p?.name?.takeIf { it.isNotBlank() } ?: p?.user?.takeIf { it.isNotBlank() } ?: Build.MODEL ?: "Android phone"
        return "Mobile data from $name"
    }

    /** (Re)builds the relay for the current networks. */
    @Synchronized
    private fun restartRelay() {
        stopRelay()
        if (!running) return
        val server = repo.servers.value.firstOrNull()
        val key = repo.shareKey(server?.payload)
        val cell = cellular
        val lanNet = wifi
        waitingForKey = server == null || key == null
        when {
            server == null || key == null -> return status(ShareStatus.NO_KEY)
            cell == null -> return status(ShareStatus.NO_CELLULAR)
            lanNet == null -> return status(ShareStatus.NO_WIFI)
        }
        val payload = server!!.payload
        val s = repo.settings.value
        val lan = try {
            DatagramSocket(null).also { d ->
                d.reuseAddress = true
                TailbondVpnService.protectSocket(d)
                lanNet!!.bindSocket(d)
                d.bind(InetSocketAddress(P.DEFAULT_PORT))
            }
        } catch (e: Exception) {
            Log.w(TAG, "share: LAN socket: ${e.message}")
            return status(ShareStatus.NO_WIFI)
        }
        val serverAddr = InetSocketAddress(payload.host, payload.port)
        relayMonth = repo.shareMonth()
        appliedCap = s.shareCapMb * 1_000_000
        appliedHubs = s.shareAllowedHubs.toSet()
        val r = ShareRelay(
            ShareRelay.Config(
                id = repo.shareId(), label = label(), key = key!!.toByteArray(Charsets.UTF_8),
                capBytes = appliedCap, usedBytes = repo.sharedBytes(), allowedHubs = appliedHubs,
                openUpstream = {
                    val net = cellular ?: throw IllegalStateException("no mobile data")
                    DatagramSocket(null).also { d ->
                        TailbondVpnService.protectSocket(d)
                        net.bindSocket(d)
                        d.bind(null)
                        // resolve over cellular: the Wi-Fi's DNS may not answer for the server
                        val addr = if (serverAddr.isUnresolved) InetSocketAddress(net.getByName(payload.host), payload.port) else serverAddr
                        d.connect(addr)
                    }
                },
                check = ::check,
                onChange = ::changed,
                log = { Log.i(TAG, it) },
            ),
        )
        relay = r
        relayThread = Thread({ r.serve(lan) }, "share-relay").apply { isDaemon = true; start() }
        registerNsd(key, s.shareCapMb)
        status(ShareStatus.WAITING)
    }

    @Synchronized
    private fun stopRelay() {
        relay?.let {
            repo.setSharedBytes(relayMonth, it.usedBytes)
            it.stop()
        }
        relay = null
        relayThread = null
        nsd?.let { l -> try { getSystemService<NsdManager>()?.unregisterService(l) } catch (_: Exception) {} }
        nsd = null
    }

    private fun registerNsd(key: String, @Suppress("UNUSED_PARAMETER") capMb: Long) {
        val m = getSystemService<NsdManager>() ?: return
        val info = NsdServiceInfo().apply {
            serviceName = label()
            serviceType = P.SERVICE_TYPE
            port = P.DEFAULT_PORT
            setAttribute("v", "1")
            setAttribute("id", repo.shareId())
            setAttribute("k", P.keyId(key.toByteArray(Charsets.UTF_8)))
            setAttribute("n", label())
        }
        val l = object : NsdManager.RegistrationListener {
            override fun onServiceRegistered(si: NsdServiceInfo) = Unit
            override fun onRegistrationFailed(si: NsdServiceInfo, err: Int) { Log.w(TAG, "share: mDNS registration failed ($err)") }
            override fun onServiceUnregistered(si: NsdServiceInfo) = Unit
            override fun onUnregistrationFailed(si: NsdServiceInfo, err: Int) = Unit
        }
        try {
            m.registerService(info, NsdManager.PROTOCOL_DNS_SD, l)
            nsd = l
        } catch (e: Exception) {
            Log.w(TAG, "share: mDNS: ${e.message}")
        }
    }

    /** Battery: may sharing go on? */
    private fun check(): ShareRelay.Check {
        val b = registerReceiver(null, IntentFilter(Intent.ACTION_BATTERY_CHANGED))
        val level = b?.getIntExtra(BatteryManager.EXTRA_LEVEL, -1) ?: -1
        val scale = b?.getIntExtra(BatteryManager.EXTRA_SCALE, 100) ?: 100
        val pct = if (level >= 0 && scale > 0) level * 100 / scale else -1
        val charging = (b?.getIntExtra(BatteryManager.EXTRA_PLUGGED, 0) ?: 0) != 0
        val min = repo.settings.value.shareMinBattery
        val reason = if (min > 0 && pct in 0 until min && !charging) P.R_BATTERY else P.R_NONE
        return ShareRelay.Check(reason, pct, charging)
    }

    private fun changed() {
        val r = relay ?: return
        val hubs = r.hubs()
        val c = check()
        val st = when {
            c.reason == P.R_BATTERY -> ShareStatus.BATTERY
            repo.settings.value.shareCapMb > 0 && r.usedBytes >= repo.settings.value.shareCapMb * 1_000_000 -> ShareStatus.CAP
            hubs.isNotEmpty() -> ShareStatus.SHARING
            else -> ShareStatus.WAITING
        }
        ShareState.set { it.copy(status = st, hubs = hubs, usedBytes = r.usedBytes, battery = c.battery, charging = c.charging) }
        val now = System.currentTimeMillis()
        if (now - lastNotify > 2_000) {
            lastNotify = now
            notifyNow()
        }
    }

    private fun status(s: ShareStatus) {
        val c = check()
        ShareState.set { it.copy(status = s, hubs = emptyList(), battery = c.battery, charging = c.charging, label = label()) }
        notifyNow()
    }

    private fun notifyNow() {
        val l = ShareState.live.value
        val (title, text) = ShareWords.notification(l, repo.settings.value.shareMinBattery)
        getSystemService<NotificationManager>()?.notify(NOTIFICATION_ID, notification(title, text))
    }

    /** Once a second: counters, the battery limit, and the home server's sharer list. */
    private fun tickLoop() {
        while (running) {
            try {
                Thread.sleep(1_000)
            } catch (_: InterruptedException) {
                return
            }
            tick()
            relay?.let { r ->
                changed()
                if (System.currentTimeMillis() % 60_000 < 1_000) repo.setSharedBytes(relayMonth, r.usedBytes)
            }
            val now = System.currentTimeMillis()
            if (relay != null && now - lastRegister > 60_000 && LiveState.live.value.state == ConnState.CONNECTED) {
                lastRegister = now
                register()
            }
        }
    }

    /**
     * Settings and time, applied to the running relay: a changed cap or
     * allowed list at once (dropping hubs no longer allowed), a new month's
     * counter, and a share key that arrived while sharing waited for one.
     */
    @Synchronized
    private fun tick() {
        if (!running) return
        val r = relay
        if (r == null) {
            if (waitingForKey && repo.shareKey(repo.servers.value.firstOrNull()?.payload) != null) restartRelay()
            return
        }
        val month = repo.shareMonth()
        if (month != relayMonth) {
            relayMonth = month
            r.resetUsage()
            repo.setSharedBytes(month, 0)
        }
        val s = repo.settings.value
        val cap = s.shareCapMb * 1_000_000
        val hubs = s.shareAllowedHubs.toSet()
        if (cap != appliedCap || hubs != appliedHubs) {
            appliedCap = cap
            appliedHubs = hubs
            r.updateLimits(cap, hubs)
        }
    }

    /** Tells the home server where this phone shares (hubs that can't see mDNS find it there). Through tailbond's tunnel. */
    private fun register() {
        val server = repo.servers.value.firstOrNull() ?: return
        val lanIps = wifi?.let { cm.getLinkProperties(it) }?.let(::ipv4s).orEmpty()
        val key = repo.shareKey(server.payload) ?: return
        val body = buildJsonObject {
            put("id", repo.shareId())
            put("label", label())
            put("key_id", P.keyId(key.toByteArray(Charsets.UTF_8)))
            put("port", P.DEFAULT_PORT)
            put("kind", "android")
            put("lan", JsonArray(lanIps.map { JsonPrimitive(it) }))
        }.toString()
        try {
            PairRequestClient.socketExchange(
                InetAddress.getByName(ServerList.serverTunnelIp(server.payload)), PairRequestClient.PORT, "POST", "/api/shares",
                body, PeersClient.token(server.payload.key),
            )
        } catch (e: Exception) {
            Log.i(TAG, "share: register: ${e.message}")
        }
    }

    private fun ipv4s(lp: LinkProperties): List<String> =
        lp.linkAddresses.mapNotNull { (it.address as? Inet4Address)?.hostAddress }

    private fun stopSharing() {
        running = false
        ticker?.interrupt()
        try { cm.unregisterNetworkCallback(cellCb) } catch (_: Exception) {}
        try { cm.unregisterNetworkCallback(wifiCb) } catch (_: Exception) {}
        stopRelay()
        ShareState.set { ShareLive(usedBytes = repo.sharedBytes(), label = label()) }
        ServiceCompat.stopForeground(this, ServiceCompat.STOP_FOREGROUND_REMOVE)
        stopSelf()
    }

    override fun onDestroy() {
        if (running) stopSharing()
        super.onDestroy()
    }

    private fun notification(title: String, text: String?): Notification {
        val open = PendingIntent.getActivity(
            this, 0, Intent(this, MainActivity::class.java).addFlags(Intent.FLAG_ACTIVITY_SINGLE_TOP), PendingIntent.FLAG_IMMUTABLE,
        )
        val stop = PendingIntent.getService(this, 2, Intent(this, ShareService::class.java).setAction(ACTION_STOP), PendingIntent.FLAG_IMMUTABLE)
        return NotificationCompat.Builder(this, CHANNEL)
            .setSmallIcon(R.drawable.ic_stat_tailbond)
            .setContentTitle(title)
            .setContentText(text)
            .setOngoing(true)
            .setOnlyAlertOnce(true)
            .setContentIntent(open)
            .addAction(0, getString(R.string.share_stop), stop)
            .build()
    }

    companion object {
        private const val TAG = "tailbond"
        private const val CHANNEL = "share"
        private const val NOTIFICATION_ID = 2
        const val ACTION_START = "dev.tailbond.app.SHARE_START"
        const val ACTION_STOP = "dev.tailbond.app.SHARE_STOP"

        fun start(ctx: Context) =
            ContextCompat.startForegroundService(ctx, Intent(ctx, ShareService::class.java).setAction(ACTION_START))

        fun stop(ctx: Context) {
            ctx.startService(Intent(ctx, ShareService::class.java).setAction(ACTION_STOP))
        }
    }
}
