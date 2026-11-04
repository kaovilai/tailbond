package dev.tailbond.app.share

import android.app.Notification
import android.app.NotificationChannel
import android.app.NotificationManager
import android.app.PendingIntent
import android.app.Service
import android.content.Context
import android.content.Intent
import android.content.pm.ServiceInfo
import android.net.ConnectivityManager
import android.net.Network
import android.net.NetworkCapabilities
import android.net.nsd.NsdManager
import android.net.nsd.NsdServiceInfo
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
import dev.tailbond.app.share.ShareProtocol as P
import dev.tailbond.app.ui.MainActivity
import kotlinx.coroutines.flow.MutableStateFlow
import kotlinx.coroutines.flow.StateFlow
import kotlinx.coroutines.flow.asStateFlow
import kotlinx.serialization.json.JsonArray
import kotlinx.serialization.json.JsonPrimitive
import kotlinx.serialization.json.buildJsonObject
import kotlinx.serialization.json.put
import java.io.IOException
import java.net.Inet4Address
import java.net.InetAddress
import java.net.InetSocketAddress
import java.net.NetworkInterface
import java.net.ServerSocket
import java.net.Socket

/** What "Share bonded connection" is doing right now. */
enum class HubStatus {
    OFF,
    /** Listening; tailbond's tunnel is up; no laptop uses it yet. */
    READY,
    SHARING,
    /** Listening, but tailbond isn't connected: nothing goes out until it is. */
    NO_TUNNEL,
    /** Not paired, or the pairing predates sharing and the server hasn't been reached yet. */
    NO_KEY,
    /** Port 1888 is taken. */
    ERROR,
}

data class HubLive(
    val status: HubStatus = HubStatus.OFF,
    val clients: List<ProxyServer.Client> = emptyList(),
    val upBytes: Long = 0,
    val downBytes: Long = 0,
    val label: String = "",
    val error: String = "",
)

/** In-process phone hub state, written by [HubService], read by the UI. */
object HubState {
    private val _live = MutableStateFlow(HubLive())
    val live: StateFlow<HubLive> = _live.asStateFlow()
    fun set(f: (HubLive) -> HubLive) {
        while (true) {
            val cur = _live.value
            if (_live.compareAndSet(cur, f(cur))) return
        }
    }
}

/**
 * "Share bonded connection" (docs/phone-hub.md): a laptop on this phone's
 * hotspot, USB tethering or Wi-Fi uses tailbond's bonded tunnel through a
 * SOCKS5 / HTTP CONNECT proxy ([ProxyServer]) on port 1888.
 *
 * - Listens on every interface, but serves only connections that arrive on
 *   a LAN address (hotspot, USB, Bluetooth, Wi-Fi, loopback for adb): never
 *   on mobile data or the VPN; sources must be private or link-local.
 * - Password: derived from the home server's share key, as the Mac does.
 * - Upstream sockets are opened on tailbond's VPN network (never
 *   protect()ed): the laptop's traffic goes into the bonded tunnel, and
 *   when the tunnel is down nothing goes out at all (fails closed).
 * - Advertised over mDNS as _tailbond-proxy._tcp, and registered with the
 *   home server (kind "proxy") so a Mac that can't see mDNS finds it.
 */
class HubService : Service() {
    private lateinit var repo: Repository
    private lateinit var cm: ConnectivityManager
    private var proxy: ProxyServer? = null
    private var nsd: NsdManager.RegistrationListener? = null
    private var ticker: Thread? = null
    private var lastNotify = 0L
    private var lastRegister = 0L
    private var lastLine = ""
    @Volatile private var running = false

    override fun onBind(intent: Intent?): IBinder? = null

    override fun onCreate() {
        super.onCreate()
        repo = Repository.get(this)
        cm = getSystemService()!!
        getSystemService<NotificationManager>()?.createNotificationChannel(
            NotificationChannel(CHANNEL, getString(R.string.hub_channel), NotificationManager.IMPORTANCE_LOW),
        )
    }

    override fun onStartCommand(intent: Intent?, flags: Int, startId: Int): Int {
        if (intent?.action == ACTION_STOP) {
            stopHub()
            return START_NOT_STICKY
        }
        ServiceCompat.startForeground(
            this, NOTIFICATION_ID, notification(HubWords.headline(HubState.live.value.copy(status = HubStatus.READY)), null),
            if (Build.VERSION.SDK_INT >= 34) ServiceInfo.FOREGROUND_SERVICE_TYPE_SPECIAL_USE else 0,
        )
        if (!running) startHub()
        return START_STICKY
    }

    /** "pixel-e2e": this phone's name on the home server, as the Mac shows it. */
    private fun label(): String {
        val p = repo.servers.value.firstOrNull()?.payload
        return p?.name?.takeIf { it.isNotBlank() } ?: p?.user?.takeIf { it.isNotBlank() } ?: Build.MODEL ?: "Android phone"
    }

    private fun startHub() {
        running = true
        val server = repo.servers.value.firstOrNull()
        val key = repo.shareKey(server?.payload)
        if (server == null || key == null) {
            HubState.set { HubLive(status = HubStatus.NO_KEY, label = label()) }
            notifyNow()
            return
        }
        val ss = try {
            ServerSocket().apply {
                reuseAddress = true
                // "::" takes IPv4 too (v4-mapped); IPv6 link-local clients included
                bind(InetSocketAddress(InetAddress.getByName("::"), ProxyServer.DEFAULT_PORT), 64)
            }
        } catch (e: IOException) {
            HubState.set { HubLive(status = HubStatus.ERROR, label = label(), error = e.message.orEmpty()) }
            notifyNow()
            return
        }
        val p = ProxyServer(
            ProxyServer.Config(
                // hubs log in with the pass their server gave them for this phone
                auth = ProxyServer.verifier(key.toByteArray(Charsets.UTF_8), allowed = { hub ->
                    repo.settings.value.shareAllowedHubs.let { it.isEmpty() || hub in it }
                }),
                dial = ::dialThroughTunnel, allowLocal = ::onLan,
                log = { Log.i(TAG, it) },
            ),
        )
        proxy = p
        Thread({ p.serve(ss) }, "hub-proxy").apply { isDaemon = true; start() }
        registerNsd(key)
        HubState.set { HubLive(status = status(p), label = label()) }
        Log.i(TAG, "hub: proxy on port ${ProxyServer.DEFAULT_PORT}")
        ticker = Thread({ tickLoop() }, "hub-tick").apply { isDaemon = true; start() }
    }

    /** tailbond's own VPN network, when it's connected. */
    private fun vpnNetwork(): Network? {
        if (LiveState.live.value.state != ConnState.CONNECTED) return null
        @Suppress("DEPRECATION")
        return cm.allNetworks.firstOrNull { cm.getNetworkCapabilities(it)?.hasTransport(NetworkCapabilities.TRANSPORT_VPN) == true }
    }

    /** The upstream: a socket on the VPN network (names resolved there too); no tunnel, no connection. */
    private fun dialThroughTunnel(host: String, port: Int): Socket {
        val vpn = vpnNetwork() ?: throw IOException("tailbond isn't connected")
        val addr = vpn.getAllByName(host).first()
        val s = vpn.socketFactory.createSocket()
        try {
            s.connect(InetSocketAddress(addr, port), 20_000)
        } catch (e: IOException) {
            s.close()
            throw e
        }
        return s
    }

    /** Addresses of mobile data and the VPN: never serve connections arriving there. */
    private fun wanAddrs(): Set<InetAddress> {
        val out = mutableSetOf<InetAddress>()
        @Suppress("DEPRECATION")
        for (n in cm.allNetworks) {
            val caps = cm.getNetworkCapabilities(n) ?: continue
            if (caps.hasTransport(NetworkCapabilities.TRANSPORT_CELLULAR) || caps.hasTransport(NetworkCapabilities.TRANSPORT_VPN)) {
                cm.getLinkProperties(n)?.linkAddresses?.forEach { out += it.address }
            }
        }
        return out
    }

    private fun onLan(local: InetAddress): Boolean = local.isLoopbackAddress || local !in wanAddrs()

    /** Private IPv4 addresses of the LAN side (Wi-Fi, hotspot, USB), for the server's list. */
    private fun lanIpv4s(): List<String> {
        val wan = wanAddrs()
        return try {
            NetworkInterface.getNetworkInterfaces().toList()
                .filter { it.isUp && !it.isLoopback }
                .flatMap { it.inetAddresses.toList() }
                .filterIsInstance<Inet4Address>()
                .filter { it.isSiteLocalAddress && it !in wan }
                .mapNotNull { it.hostAddress }
                .distinct().take(8)
        } catch (_: Exception) {
            emptyList()
        }
    }

    private fun hubId() = repo.shareId() + "-hub"

    private fun registerNsd(key: String) {
        val m = getSystemService<NsdManager>() ?: return
        val info = NsdServiceInfo().apply {
            serviceName = label()
            serviceType = ProxyServer.SERVICE_TYPE
            port = ProxyServer.DEFAULT_PORT
            setAttribute("v", "1")
            setAttribute("id", hubId())
            setAttribute("k", P.keyId(key.toByteArray(Charsets.UTF_8)))
            setAttribute("n", label())
            setAttribute("b", "1") // the upstream is tailbond's bonded tunnel
        }
        val l = object : NsdManager.RegistrationListener {
            override fun onServiceRegistered(si: NsdServiceInfo) = Unit
            override fun onRegistrationFailed(si: NsdServiceInfo, err: Int) { Log.w(TAG, "hub: mDNS registration failed ($err)") }
            override fun onServiceUnregistered(si: NsdServiceInfo) = Unit
            override fun onUnregistrationFailed(si: NsdServiceInfo, err: Int) = Unit
        }
        try {
            m.registerService(info, NsdManager.PROTOCOL_DNS_SD, l)
            nsd = l
        } catch (e: Exception) {
            Log.w(TAG, "hub: mDNS: ${e.message}")
        }
    }

    private fun status(p: ProxyServer): HubStatus = when {
        vpnNetwork() == null -> HubStatus.NO_TUNNEL
        p.clients().any { it.conns > 0 || System.currentTimeMillis() - it.lastSeenMs < 30_000 } -> HubStatus.SHARING
        else -> HubStatus.READY
    }

    /** Once a second: counters, the notification, the home server's list. */
    private fun tickLoop() {
        while (running) {
            try {
                Thread.sleep(1_000)
            } catch (_: InterruptedException) {
                return
            }
            val p = proxy ?: continue
            val clients = p.clients()
            HubState.set { it.copy(status = status(p), clients = clients, upBytes = p.upBytes, downBytes = p.downBytes) }
            val line = "hub: up ${p.upBytes} down ${p.downBytes}" + clients.joinToString("") { "; client ${it.addr} conns ${it.conns} up ${it.up} down ${it.down}" }
            if (line != lastLine) {
                lastLine = line
                Log.i(TAG, line)
            }
            val now = System.currentTimeMillis()
            if (now - lastNotify > 2_000) {
                lastNotify = now
                notifyNow()
            }
            if (now - lastRegister > 60_000 && LiveState.live.value.state == ConnState.CONNECTED) {
                lastRegister = now
                register()
            }
        }
    }

    private fun register() {
        val server = repo.servers.value.firstOrNull() ?: return
        val key = repo.shareKey(server.payload) ?: return
        val body = buildJsonObject {
            put("id", hubId())
            put("label", label())
            put("key_id", P.keyId(key.toByteArray(Charsets.UTF_8)))
            put("port", ProxyServer.DEFAULT_PORT)
            put("kind", "proxy")
            put("lan", JsonArray(lanIpv4s().map { JsonPrimitive(it) }))
        }.toString()
        try {
            PairRequestClient.socketExchange(
                InetAddress.getByName(ServerList.serverTunnelIp(server.payload)), PairRequestClient.PORT, "POST", "/api/shares",
                body, PeersClient.token(server.payload.key),
            )
        } catch (e: Exception) {
            Log.i(TAG, "hub: register: ${e.message}")
        }
    }

    private fun notifyNow() {
        val l = HubState.live.value
        getSystemService<NotificationManager>()?.notify(NOTIFICATION_ID, notification(HubWords.headline(l), HubWords.detail(l)))
    }

    private fun stopHub() {
        running = false
        ticker?.interrupt()
        proxy?.stop()
        proxy = null
        nsd?.let { l -> try { getSystemService<NsdManager>()?.unregisterService(l) } catch (_: Exception) {} }
        nsd = null
        HubState.set { HubLive(label = label()) }
        ServiceCompat.stopForeground(this, ServiceCompat.STOP_FOREGROUND_REMOVE)
        stopSelf()
    }

    override fun onDestroy() {
        if (running) stopHub()
        super.onDestroy()
    }

    private fun notification(title: String, text: String?): Notification {
        val open = PendingIntent.getActivity(
            this, 0, Intent(this, MainActivity::class.java).addFlags(Intent.FLAG_ACTIVITY_SINGLE_TOP), PendingIntent.FLAG_IMMUTABLE,
        )
        val stop = PendingIntent.getService(this, 3, Intent(this, HubService::class.java).setAction(ACTION_STOP), PendingIntent.FLAG_IMMUTABLE)
        return NotificationCompat.Builder(this, CHANNEL)
            .setSmallIcon(R.drawable.ic_stat_tailbond)
            .setContentTitle(title)
            .setContentText(text)
            .setOngoing(true)
            .setOnlyAlertOnce(true)
            .setContentIntent(open)
            .addAction(0, getString(R.string.hub_stop), stop)
            .build()
    }

    companion object {
        private const val TAG = "tailbond"
        private const val CHANNEL = "hub"
        private const val NOTIFICATION_ID = 3
        const val ACTION_START = "dev.tailbond.app.HUB_START"
        const val ACTION_STOP = "dev.tailbond.app.HUB_STOP"

        fun start(ctx: Context) =
            ContextCompat.startForegroundService(ctx, Intent(ctx, HubService::class.java).setAction(ACTION_START))

        fun stop(ctx: Context) {
            ctx.startService(Intent(ctx, HubService::class.java).setAction(ACTION_STOP))
        }
    }
}
