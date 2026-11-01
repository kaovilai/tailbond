package dev.tailbond.app.service

import android.content.Context
import android.net.ConnectivityManager
import android.net.Network
import android.net.NetworkCapabilities
import androidx.core.content.getSystemService
import dev.tailbond.app.failover.QuicProbe
import java.io.IOException
import java.net.DatagramPacket
import java.net.DatagramSocket
import java.net.Inet4Address
import java.net.InetAddress
import java.net.InetSocketAddress

/**
 * Probes a standby server outside the tunnel (docs/architecture.md,
 * "Health"): a [QuicProbe] datagram from a socket that is protect()ed and
 * bound to an underlying (non-VPN) network; any reply within 2 s means the
 * server is reachable. Each validated network is tried in turn (Wi-Fi first),
 * since a server reachable over one of them is usable.
 */
class ServerProber(private val ctx: Context, private val protect: (DatagramSocket) -> Boolean) {

    /** Blocking (up to 2 s per network). */
    fun probe(host: String, port: Int): Boolean {
        val nets = underlyingNetworks()
        if (nets.isEmpty()) return false
        return nets.any { probeOn(it, host, port) }
    }

    private fun probeOn(net: Network, host: String, port: Int): Boolean = try {
        val addrs = net.getAllByName(host).toList()
        val addr: InetAddress = addrs.firstOrNull { it is Inet4Address } ?: addrs.first()
        DatagramSocket().use { s ->
            if (!protect(s)) return false
            net.bindSocket(s)
            s.soTimeout = QuicProbe.TIMEOUT_MS
            val pkt = QuicProbe.packet()
            s.send(DatagramPacket(pkt, pkt.size, InetSocketAddress(addr, port)))
            val buf = ByteArray(1500)
            s.receive(DatagramPacket(buf, buf.size))
            true
        }
    } catch (_: IOException) {
        false
    } catch (_: RuntimeException) {
        false
    }

    private fun underlyingNetworks(): List<Network> {
        val cm = ctx.getSystemService<ConnectivityManager>() ?: return emptyList()
        @Suppress("DEPRECATION")
        return cm.allNetworks.mapNotNull { n -> cm.getNetworkCapabilities(n)?.let { n to it } }
            .filter { (_, c) ->
                !c.hasTransport(NetworkCapabilities.TRANSPORT_VPN) &&
                    c.hasCapability(NetworkCapabilities.NET_CAPABILITY_INTERNET) &&
                    c.hasCapability(NetworkCapabilities.NET_CAPABILITY_VALIDATED)
            }
            .sortedBy { (_, c) -> if (c.hasTransport(NetworkCapabilities.TRANSPORT_WIFI)) 0 else 1 }
            .map { it.first }
    }
}
