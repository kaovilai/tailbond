package dev.tailbond.app

import dev.tailbond.app.share.ShareProtocol as P
import dev.tailbond.app.share.ShareRelay
import kotlinx.serialization.json.Json
import kotlinx.serialization.json.jsonObject
import kotlinx.serialization.json.jsonPrimitive
import kotlinx.serialization.json.long
import org.junit.After
import org.junit.Assert.assertArrayEquals
import org.junit.Assert.assertEquals
import org.junit.Assert.assertNull
import org.junit.Assert.assertTrue
import org.junit.Test
import java.io.File
import java.net.DatagramPacket
import java.net.DatagramSocket
import java.net.InetAddress
import java.net.InetSocketAddress
import java.net.SocketTimeoutException

/** Pair & Share: the Android relay against the Go reference (vectors.json) and over loopback. */
class ShareRelayTest {
    private fun hex(s: String) = s.chunked(2).map { it.toInt(16).toByte() }.toByteArray()

    private val vectors by lazy {
        // the Go reference's pinned vectors (agent/internal/share/testdata)
        val f = listOf("../../agent/internal/share/testdata/vectors.json", "../agent/internal/share/testdata/vectors.json")
            .map { File(it) }.first { it.exists() }
        Json.parseToJsonElement(f.readText()).jsonObject
    }

    private fun v(k: String) = vectors[k]!!.jsonPrimitive.content

    @Test fun wireFormatAndMacsMatchTheGoReference() {
        val key = v("key").toByteArray()
        val hn = hex(v("hub_nonce"))
        val sn = hex(v("sharer_nonce"))
        val sess = hex(v("session"))
        val cap = vectors["cap_left"]!!.jsonPrimitive.long
        assertEquals(v("key_id"), P.keyId(key))
        assertArrayEquals(hex(v("auth_mac")), P.authMac(key, hn, sn, v("hub_id")))
        assertArrayEquals(hex(v("welcome_mac")), P.welcomeMac(key, hn, sn, sess, cap))
        assertArrayEquals(hex(v("refuse_mac")), P.refuseMac(key, hn, sn, P.R_BATTERY))
        assertArrayEquals(hex(v("hello")), P.hello(hn, v("hub_id")))
        assertArrayEquals(hex(v("challenge")), P.challenge(hn, sn, v("sharer_id"), v("label")))
        assertArrayEquals(hex(v("auth")), P.auth(hn, sn, hex(v("auth_mac"))))
        assertArrayEquals(hex(v("welcome")), P.welcome(hn, sess, cap, hex(v("welcome_mac"))))
        assertArrayEquals(hex(v("refuse")), P.refuse(hn, P.R_BATTERY, hex(v("refuse_mac"))))
        val out = ByteArray(64)
        val n = P.data(out, sess, "quic".toByteArray(), 0, 4)
        assertArrayEquals(hex(v("data")), out.copyOf(n))
        assertArrayEquals(hex(v("keepalive")), P.keepalive(sess, 7))
        assertArrayEquals(hex(v("ka_ack")), P.kaAck(sess, 7, 1000, 2000, P.NO_CAP, 63, P.FLAG_CHARGING))
        assertArrayEquals(hex(v("bye")), P.bye(sess, P.R_CAP))
        val h = P.parseHello(hex(v("hello")))!!
        assertArrayEquals(hn, h.hubNonce)
        assertEquals("slate7", h.hubId)
        assertNull(P.parseHello(hex(v("hello")).copyOf(5)))
    }

    // ---- loopback relay ----

    private val key = "0123456789abcdef0123456789abcdef".toByteArray()
    private val closers = mutableListOf<() -> Unit>()

    @After fun tearDown() = closers.forEach { it() }

    private val lo: InetAddress = InetAddress.getLoopbackAddress()

    /** The home server: answers "re:" + datagram. */
    private fun echoServer(): InetSocketAddress {
        val s = DatagramSocket(0, lo)
        closers += { s.close() }
        Thread {
            val buf = ByteArray(2048)
            val p = DatagramPacket(buf, buf.size)
            while (!s.isClosed) {
                try {
                    p.setLength(buf.size)
                    s.receive(p)
                    val r = "re:".toByteArray() + buf.copyOf(p.length)
                    s.send(DatagramPacket(r, r.size, p.socketAddress))
                } catch (_: Exception) {
                }
            }
        }.apply { isDaemon = true }.start()
        return InetSocketAddress(lo, s.localPort)
    }

    private fun relay(
        allowed: Set<String> = emptySet(),
        capBytes: Long = 0,
        check: () -> ShareRelay.Check = { ShareRelay.Check(P.R_NONE, 80, true) },
    ): Pair<ShareRelay, InetSocketAddress> {
        val server = echoServer()
        val r = ShareRelay(
            ShareRelay.Config(
                id = "pixel-1", label = "Pixel's 5G", key = key, capBytes = capBytes, allowedHubs = allowed, check = check,
                openUpstream = { DatagramSocket(0, lo).apply { connect(server) } },
            ),
        )
        val lan = DatagramSocket(0, lo)
        Thread { r.serve(lan) }.apply { isDaemon = true }.start()
        closers += { r.stop() }
        return r to InetSocketAddress(lo, lan.localPort)
    }

    /** A minimal hub: handshake, then DATA frames. */
    private class Hub(val at: InetSocketAddress, val key: ByteArray, val id: String) {
        val sock = DatagramSocket(0, InetAddress.getLoopbackAddress()).apply { soTimeout = 1500 }
        var session: ByteArray? = null
        var refused: Int = -1

        fun recv(): ByteArray? = try {
            val b = ByteArray(65536)
            val p = DatagramPacket(b, b.size)
            sock.receive(p)
            b.copyOf(p.length)
        } catch (_: SocketTimeoutException) {
            null
        }

        fun send(b: ByteArray) = sock.send(DatagramPacket(b, b.size, at))

        fun handshake(): Boolean {
            val hn = ByteArray(P.NONCE) { it.toByte() }
            send(P.hello(hn, id))
            val c = recv() ?: return false
            assertEquals(P.CHALLENGE, c[0])
            val sn = c.copyOfRange(2 + P.NONCE, 2 + 2 * P.NONCE)
            send(P.auth(hn, sn, P.authMac(key, hn, sn, id)))
            val w = recv() ?: return false
            when (w[0]) {
                P.WELCOME -> {
                    val s = w.copyOfRange(2 + P.NONCE, 2 + P.NONCE + P.SESSION)
                    val cap = java.nio.ByteBuffer.wrap(w, 2 + P.NONCE + P.SESSION, 8).long
                    val mac = w.copyOfRange(2 + P.NONCE + P.SESSION + 8, 2 + P.NONCE + P.SESSION + 8 + P.MAC)
                    assertTrue("the sharer proves the key", P.macEquals(mac, P.welcomeMac(key, hn, sn, s, cap)))
                    session = s
                    return true
                }
                P.REFUSE -> {
                    refused = w[2 + P.NONCE].toInt()
                    return false
                }
            }
            return false
        }

        fun roundTrip(msg: String): String? {
            val out = ByteArray(2048)
            val n = P.data(out, session!!, msg.toByteArray(), 0, msg.length)
            send(out.copyOf(n))
            val r = recv() ?: return null
            assertEquals(P.DATA, r[0])
            return String(r, P.DATA_HEADER, r.size - P.DATA_HEADER)
        }
    }

    @Test fun carriesTrafficBothWaysAndCounts() {
        val (r, at) = relay()
        val hub = Hub(at, key, "slate7")
        assertTrue(hub.handshake())
        repeat(3) { assertEquals("re:quic-$it", hub.roundTrip("quic-$it")) }
        val use = r.hubs().single()
        assertEquals("slate7", use.hub)
        assertEquals(3L * 6, use.upBytes)
        assertEquals(3L * 9, use.downBytes)
        assertEquals(use.upBytes + use.downBytes, r.usedBytes)
        // keepalive: counters, battery and charging come back
        hub.send(P.keepalive(hub.session!!, 1))
        val ack = hub.recv()!!
        assertEquals(P.KA_ACK, ack[0])
        assertEquals(80, ack[ack.size - 2].toInt())
        assertEquals(P.FLAG_CHARGING, ack[ack.size - 1].toInt())
    }

    @Test fun wrongKeyGetsNothing() {
        val (r, at) = relay()
        val hub = Hub(at, "another server's key.............".toByteArray(), "slate7")
        assertTrue(!hub.handshake())
        assertEquals(-1, hub.refused) // not even a refusal
        assertTrue(r.hubs().isEmpty())
    }

    @Test fun onlyAllowedHubs() {
        val (_, at) = relay(allowed = setOf("slate7"))
        val other = Hub(at, key, "laptop")
        assertTrue(!other.handshake())
        assertEquals(P.R_NOT_ALLOWED, other.refused)
        assertTrue(Hub(at, key, "slate7").handshake())
    }

    @Test fun lowBatteryRefusesAndEnds() {
        var low = false
        val (_, at) = relay(check = { if (low) ShareRelay.Check(P.R_BATTERY, 12, false) else ShareRelay.Check(P.R_NONE, 50, false) })
        val hub = Hub(at, key, "slate7")
        assertTrue(hub.handshake())
        low = true
        hub.send(P.keepalive(hub.session!!, 1))
        val bye = hub.recv()!!
        assertEquals(P.BYE, bye[0])
        assertEquals(P.R_BATTERY, bye[1 + P.SESSION].toInt())
        val again = Hub(at, key, "slate7")
        assertTrue(!again.handshake())
        assertEquals(P.R_BATTERY, again.refused)
    }

    @Test fun capStopsSharing() {
        val (r, at) = relay(capBytes = 20)
        val hub = Hub(at, key, "slate7")
        assertTrue(hub.handshake())
        hub.roundTrip("0123456789") // 10 up + 13 down = 23 > 20
        hub.send(P.keepalive(hub.session!!, 2))
        val bye = hub.recv()!!
        assertEquals(P.BYE, bye[0])
        assertEquals(P.R_CAP, bye[1 + P.SESSION].toInt())
        assertTrue(r.usedBytes >= 20)
    }
}
