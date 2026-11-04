package dev.tailbond.app.share

import android.content.Context
import android.net.ConnectivityManager
import android.net.Network
import android.net.NetworkCapabilities
import android.net.nsd.NsdManager
import android.net.nsd.NsdServiceInfo
import android.util.Log
import androidx.core.content.getSystemService
import dev.tailbond.app.data.PairRequestClient
import dev.tailbond.app.data.PeersClient
import dev.tailbond.app.data.Repository
import dev.tailbond.app.model.ServerList
import dev.tailbond.app.share.ShareProtocol as P
import kotlinx.coroutines.flow.MutableStateFlow
import kotlinx.coroutines.flow.StateFlow
import kotlinx.coroutines.flow.asStateFlow
import kotlinx.serialization.json.Json
import kotlinx.serialization.json.int
import kotlinx.serialization.json.jsonArray
import kotlinx.serialization.json.jsonObject
import kotlinx.serialization.json.jsonPrimitive
import java.net.DatagramSocket
import java.net.Inet4Address
import java.net.InetAddress
import java.net.InetSocketAddress
import java.util.concurrent.ConcurrentHashMap

/** A nearby phone's share as this phone (the hub) sees it. */
data class HubShare(
    val id: String,
    val label: String,
    val addr: String,
    /** mdns, server or manual (debug builds' e2e hook). */
    val via: String,
    val state: String,
    /** mqvpn's path name for it ("sh" + 6 hex), carried while ready. */
    val path: String,
    val up: Long,
    val down: Long,
    val reason: String,
    val battery: Int,
)

/** In-process hub-side share state, written by [ShareHub], read by the UI and the e2e hook. */
object ShareHubState {
    private val _live = MutableStateFlow<List<HubShare>>(emptyList())
    val live: StateFlow<List<HubShare>> = _live.asStateFlow()
    internal fun set(v: List<HubShare>) { _live.value = v }
    /**
     * Sharers added by hand, address → their key id (debug builds: the
     * emulator e2e, where mDNS can't reach the host).
     */
    val manual: MutableMap<InetSocketAddress, String> = ConcurrentHashMap()
}

/**
 * Pair & Share, hub side, on this phone (docs/pair-and-share.md): nearby
 * phones sharing their mobile data (mDNS _tailbond-share._udp, the home
 * server's list) each get a [ShareLink]; a ready link is one more path of
 * tailbond's tunnel, a relay path (patches/mqvpn/0002 + 0004) to the
 * link's local port, as tailbondd does on a travel router or a Mac.
 *
 * The link's LAN socket is bound to the Wi-Fi network and protect()ed, so it
 * never goes into tailbond's own VPN. Keys: the home server gives this hub a
 * pass per other device (GET /api/shares "passes", by key id); a sharer
 * without a pass here isn't linked. [round] runs on an IO thread while the
 * tunnel runs; [setRelays] hands the ready set to the VPN service.
 */
class ShareHub(
    private val ctx: Context,
    private val repo: Repository,
    private val protect: (DatagramSocket) -> Boolean,
    private val setRelays: (Map<String, Int>) -> Unit,
) {
    private class Found(val id: String, val label: String, val keyId: String, val addr: InetSocketAddress, val via: String)
    private class Entry(val link: ShareLink, val found: Found, var seen: Long)
    private class Pass(val key: ByteArray, val exp: Long)

    private val cm: ConnectivityManager = ctx.getSystemService()!!
    private val links = HashMap<String, Entry>()
    private val mdns = ConcurrentHashMap<String, Found>()
    /** The server's passes for this hub, by sharer key id (kept while it can't be reached). */
    @Volatile private var passes: Map<String, Pass> = emptyMap()
    private var discovery: NsdManager.DiscoveryListener? = null
    @Volatile private var resolving = false
    private val pending = java.util.ArrayDeque<NsdServiceInfo>()

    private fun wifi(): Network? {
        @Suppress("DEPRECATION")
        return cm.allNetworks.firstOrNull { cm.getNetworkCapabilities(it)?.hasTransport(NetworkCapabilities.TRANSPORT_WIFI) == true }
    }

    /** One round: discover, keep links, hand the ready ones to mqvpn. */
    @Synchronized
    fun round() {
        val server = repo.servers.value.firstOrNull()
        val key = repo.shareKey(server?.payload)
        if (server == null || key == null || !repo.settings.value.useShares) {
            stopAll()
            return
        }
        startDiscovery()
        val own = P.keyId(key.toByteArray(Charsets.UTF_8))
        val found = ArrayList<Found>()
        found += mdns.values
        found += registry(server.payload)
        for ((a, k) in ShareHubState.manual) found += Found("manual-${a.hostString}:${a.port}", "", k, a, "manual")
        val nowSec = System.currentTimeMillis() / 1000
        val usable = passes.filterValues { it.exp > nowSec }
        val now = System.currentTimeMillis()
        for (f in found) {
            if (f.keyId == own) continue // this phone's own share
            val pass = usable[f.keyId] ?: continue // no pass from the server for it
            val e = links[f.id]
            if (e != null && e.found.addr != f.addr && (f.via == "mdns" || e.found.via != "mdns")) {
                // moved: a new link on the same local port, so mqvpn's relay path stays
                val port = e.link.localPort
                e.link.stop()
                links[f.id] = Entry(newLink(f, pass, server.payload.user, port), f, now)
            } else if (e == null) {
                links[f.id] = Entry(newLink(f, pass, server.payload.user, 0), f, now)
                Log.i(TAG, "share hub: found ${f.label.ifEmpty { f.addr.toString() }} (${f.via})")
            } else {
                e.seen = now
            }
        }
        val gone = links.filter { (_, e) -> !e.link.ready && now - e.seen > LINK_TTL_MS }.keys
        for (id in gone) links.remove(id)?.link?.stop()
        val relays = links.values.filter { it.link.ready }.associate { ShareLink.pathName(it.found.id) to it.link.localPort }
        setRelays(relays)
        ShareHubState.set(links.values.map { e ->
            val st = e.link.status
            HubShare(
                id = e.found.id, label = st.label.ifEmpty { e.found.label }, addr = st.addr, via = e.found.via, state = st.state,
                path = ShareLink.pathName(e.found.id), up = st.up, down = st.down, reason = st.reason, battery = st.battery,
            )
        }.sortedBy { it.id })
    }

    private fun newLink(f: Found, pass: Pass, hubId: String, localPort: Int): ShareLink {
        val l = ShareLink(
            ShareLink.Config(
                sharer = f.addr, key = pass.key, exp = pass.exp, hubId = hubId.ifEmpty { repo.shareId() }, localPort = localPort,
                openLan = {
                    DatagramSocket(null).also { d ->
                        protect(d) // never into our own tunnel
                        // on the Wi-Fi's subnet: that network; elsewhere (a
                        // tethering interface, a hand-added address) the
                        // routing table picks
                        val w = wifi()
                        if (w != null && onSubnetOf(w, f.addr.address)) w.bindSocket(d)
                        d.bind(null)
                    }
                },
                log = { Log.i(TAG, it) },
            ),
        )
        l.start()
        return l
    }

    /**
     * The home server's list (through the tunnel, with the pairing's token):
     * this hub's passes, and the sharers on this phone's Wi-Fi subnet.
     */
    private fun registry(p: dev.tailbond.app.pairing.PairingPayload): List<Found> {
        val subnets = wifi()?.let { cm.getLinkProperties(it) }?.linkAddresses
            ?.filter { it.address is Inet4Address }?.map { it.address to it.prefixLength }.orEmpty()
        val (code, body) = try {
            PairRequestClient.socketExchange(
                InetAddress.getByName(ServerList.serverTunnelIp(p)), PairRequestClient.PORT, "GET", "/api/shares", null,
                PeersClient.token(p.key), connectMs = 2_000, readMs = 3_000,
            )
        } catch (e: Exception) {
            return emptyList()
        }
        if (code != 200) return emptyList()
        return try {
            val root = Json.parseToJsonElement(body).jsonObject
            val ps = HashMap<String, Pass>()
            for (x in root["passes"]?.jsonArray.orEmpty()) {
                val o = x.jsonObject
                val k = o["key"]?.jsonPrimitive?.content ?: continue
                val id = o["key_id"]?.jsonPrimitive?.content ?: continue
                val exp = o["exp"]?.jsonPrimitive?.content?.toLongOrNull() ?: continue
                ps[id] = Pass(k.chunked(2).map { it.toInt(16).toByte() }.toByteArray(), exp)
            }
            if (ps.isNotEmpty()) passes = ps
            if (subnets.isEmpty()) return emptyList()
            val out = ArrayList<Found>()
            for (s in root["sharers"]?.jsonArray.orEmpty()) {
                val o = s.jsonObject
                if (o["kind"]?.jsonPrimitive?.content == "proxy") continue // phone hubs' proxies aren't relays
                if (o["client"]?.jsonPrimitive?.content == p.user) continue
                val port = o["port"]?.jsonPrimitive?.int ?: continue
                val ip = o["lan"]?.jsonArray?.map { it.jsonPrimitive.content }?.firstOrNull { a ->
                    val ad = try { InetAddress.getByName(a) } catch (_: Exception) { null }
                    ad is Inet4Address && subnets.any { (net, len) -> sameSubnet(ad, net as Inet4Address, len) }
                } ?: continue
                out += Found(
                    o["id"]!!.jsonPrimitive.content, o["label"]?.jsonPrimitive?.content.orEmpty(),
                    o["key_id"]?.jsonPrimitive?.content.orEmpty(), InetSocketAddress(ip, port), "server",
                )
            }
            out
        } catch (e: Exception) {
            Log.i(TAG, "share hub: server list: ${e.message}")
            emptyList()
        }
    }

    private fun onSubnetOf(n: Network, a: InetAddress?): Boolean {
        if (a !is Inet4Address) return false
        return cm.getLinkProperties(n)?.linkAddresses.orEmpty().any { la ->
            val x = la.address
            x is Inet4Address && sameSubnet(a, x, la.prefixLength)
        }
    }

    private fun sameSubnet(a: Inet4Address, b: Inet4Address, len: Int): Boolean {
        val x = java.nio.ByteBuffer.wrap(a.address).int
        val y = java.nio.ByteBuffer.wrap(b.address).int
        val mask = if (len == 0) 0 else -1 shl (32 - len)
        return x and mask == y and mask
    }

    // ---- mDNS (NSD) ----

    private fun startDiscovery() {
        if (discovery != null) return
        val m = ctx.getSystemService<NsdManager>() ?: return
        val l = object : NsdManager.DiscoveryListener {
            override fun onDiscoveryStarted(serviceType: String) = Unit
            override fun onDiscoveryStopped(serviceType: String) = Unit
            override fun onStartDiscoveryFailed(serviceType: String, errorCode: Int) { Log.w(TAG, "share hub: mDNS discovery failed ($errorCode)") }
            override fun onStopDiscoveryFailed(serviceType: String, errorCode: Int) = Unit
            override fun onServiceFound(si: NsdServiceInfo) = resolve(m, si)
            override fun onServiceLost(si: NsdServiceInfo) {
                mdns.entries.removeIf { it.value.label == si.serviceName }
            }
        }
        try {
            m.discoverServices(P.SERVICE_TYPE, NsdManager.PROTOCOL_DNS_SD, l)
            discovery = l
        } catch (e: Exception) {
            Log.w(TAG, "share hub: mDNS: ${e.message}")
        }
    }

    /** One resolve at a time (older Androids refuse a second). */
    private fun resolve(m: NsdManager, si: NsdServiceInfo) {
        synchronized(pending) {
            if (resolving) {
                pending.add(si)
                return
            }
            resolving = true
        }
        @Suppress("DEPRECATION")
        m.resolveService(si, object : NsdManager.ResolveListener {
            override fun onResolveFailed(si: NsdServiceInfo, errorCode: Int) = next(m)
            override fun onServiceResolved(si: NsdServiceInfo) {
                val a = si.attributes
                val id = a["id"]?.let { String(it) }
                val k = a["k"]?.let { String(it) }
                @Suppress("DEPRECATION")
                val host = si.host
                if (id != null && k != null && host != null) {
                    mdns[id] = Found(id, a["n"]?.let { String(it) } ?: si.serviceName, k, InetSocketAddress(host, si.port), "mdns")
                }
                next(m)
            }
        })
    }

    private fun next(m: NsdManager) {
        val si = synchronized(pending) {
            resolving = false
            pending.poll()
        } ?: return
        resolve(m, si)
    }

    /** Ends every link and the discovery; mqvpn drops the relay paths. */
    @Synchronized
    fun stopAll() {
        for (e in links.values) e.link.stop()
        links.clear()
        discovery?.let { l -> try { ctx.getSystemService<NsdManager>()?.stopServiceDiscovery(l) } catch (_: Exception) {} }
        discovery = null
        mdns.clear()
        setRelays(emptyMap())
        ShareHubState.set(emptyList())
    }

    companion object {
        private const val TAG = "tailbond"
        const val LINK_TTL_MS = 120_000L
    }
}
