package dev.tailbond.app.data

import java.io.ByteArrayOutputStream
import java.net.DatagramPacket
import java.net.DatagramSocket
import java.net.InetAddress
import java.net.InetSocketAddress
import kotlin.random.Random

/**
 * A minimal DNS client (A and AAAA over UDP) for website bypass: it asks the
 * tunnel's DNS server directly, so the answers are the ones apps get through
 * tailbond, whatever the phone's own resolver or Private DNS setting does.
 */
object DnsLookup {
    const val TYPE_A = 1
    const val TYPE_AAAA = 28

    fun query(name: String, type: Int, id: Int): ByteArray {
        val out = ByteArrayOutputStream()
        out.write(byteArrayOf((id shr 8).toByte(), id.toByte(), 0x01, 0x00, 0, 1, 0, 0, 0, 0, 0, 0))
        for (label in name.trimEnd('.').split('.')) {
            val b = label.toByteArray(Charsets.US_ASCII)
            require(b.size in 1..63) { "bad label in $name" }
            out.write(b.size)
            out.write(b)
        }
        out.write(byteArrayOf(0, (type shr 8).toByte(), type.toByte(), 0, 1))
        return out.toByteArray()
    }

    /** Addresses of [type] in a reply to query [id]; CNAMEs are followed implicitly (their records come along). */
    fun parse(reply: ByteArray, id: Int, type: Int): List<String> {
        if (reply.size < 12) return emptyList()
        fun u16(i: Int) = ((reply[i].toInt() and 0xff) shl 8) or (reply[i + 1].toInt() and 0xff)
        if (u16(0) != id and 0xffff) return emptyList()
        if ((reply[2].toInt() and 0x80) == 0 || (reply[3].toInt() and 0x0f) != 0) return emptyList() // not a reply / rcode
        val qd = u16(4)
        val an = u16(6)
        var p = 12
        fun skipName() {
            while (p < reply.size) {
                val len = reply[p].toInt() and 0xff
                if (len == 0) { p++; return }
                if (len and 0xc0 == 0xc0) { p += 2; return }
                p += len + 1
            }
        }
        repeat(qd) { skipName(); p += 4 }
        val out = ArrayList<String>()
        repeat(an) {
            if (p >= reply.size) return out
            skipName()
            if (p + 10 > reply.size) return out
            val rtype = u16(p)
            val rdlen = u16(p + 8)
            p += 10
            if (p + rdlen > reply.size) return out
            if (rtype == type && ((type == TYPE_A && rdlen == 4) || (type == TYPE_AAAA && rdlen == 16))) {
                InetAddress.getByAddress(reply.copyOfRange(p, p + rdlen)).hostAddress?.let { out += it.substringBefore('%') }
            }
            p += rdlen
        }
        return out
    }

    /** Blocking: A and AAAA for [name] from [server]:53, 2 s timeout each. Unprotected socket, so it goes through the tunnel. */
    fun resolve(name: String, server: String, timeoutMs: Int = 2_000): List<String> {
        val addr = InetAddress.getByName(server) // a literal: no lookup
        val out = ArrayList<String>()
        DatagramSocket().use { s ->
            s.soTimeout = timeoutMs
            for (type in listOf(TYPE_A, TYPE_AAAA)) {
                val id = Random.nextInt(0, 0x10000)
                val q = query(name, type, id)
                try {
                    s.send(DatagramPacket(q, q.size, InetSocketAddress(addr, 53)))
                    val buf = ByteArray(1500)
                    val pkt = DatagramPacket(buf, buf.size)
                    s.receive(pkt)
                    out += parse(buf.copyOf(pkt.length), id, type)
                } catch (_: java.io.IOException) {
                    // timeout or unreachable: no addresses of this type this round
                }
            }
        }
        return out
    }
}
