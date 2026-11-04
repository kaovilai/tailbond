package dev.tailbond.app

import dev.tailbond.app.share.ShareLink
import dev.tailbond.app.share.ShareProtocol as P
import dev.tailbond.app.share.ShareRelay
import org.junit.After
import org.junit.Assert.assertEquals
import org.junit.Assert.assertTrue
import org.junit.Test
import java.net.DatagramPacket
import java.net.DatagramSocket
import java.net.InetAddress
import java.net.InetSocketAddress

/** The Android hub's link (a port of link.go) against the Android sharer, over loopback. */
class ShareLinkTest {
    private val key = "0123456789abcdef0123456789abcdef".toByteArray()
    private val lo: InetAddress = InetAddress.getLoopbackAddress()
    private val closers = mutableListOf<() -> Unit>()

    @After fun tearDown() = closers.reversed().forEach { it() }

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

    private fun sharer(key: ByteArray = this.key): Pair<ShareRelay, InetSocketAddress> {
        val server = echoServer()
        val r = ShareRelay(
            ShareRelay.Config(
                id = "pixel-1", label = "Pixel's 5G", key = key, capBytes = 0, allowedHubs = emptySet(),
                check = { ShareRelay.Check(P.R_NONE, 80, true) },
                openUpstream = { DatagramSocket(0, lo).apply { connect(server) } },
            ),
        )
        val lan = DatagramSocket(0, lo)
        Thread { r.serve(lan) }.apply { isDaemon = true }.start()
        closers += { r.stop() }
        return r to InetSocketAddress(lo, lan.localPort)
    }

    private fun link(at: InetSocketAddress, key: ByteArray = this.key): ShareLink {
        val l = ShareLink(ShareLink.Config(sharer = at, key = key, hubId = "pixel-hub", openLan = { DatagramSocket(0, lo) }))
        l.start()
        closers += { l.stop() }
        return l
    }

    private fun waitFor(ms: Long, f: () -> Boolean): Boolean {
        val end = System.currentTimeMillis() + ms
        while (System.currentTimeMillis() < end) {
            if (f()) return true
            Thread.sleep(50)
        }
        return f()
    }

    @Test fun relaysMqvpnDatagramsBothWays() {
        val (_, at) = sharer()
        val l = link(at)
        assertTrue("ready: ${l.status}", waitFor(5_000) { l.ready })
        assertEquals("Pixel's 5G", l.status.label)
        assertEquals("pixel-1", l.status.sharerId)
        // mqvpn's relay path: a socket that sends to the link's local port
        val mq = DatagramSocket(0, lo).apply { soTimeout = 3_000 }
        closers += { mq.close() }
        val ping = "quic".toByteArray()
        mq.send(DatagramPacket(ping, ping.size, InetSocketAddress(lo, l.localPort)))
        val buf = ByteArray(2048)
        val p = DatagramPacket(buf, buf.size)
        mq.receive(p)
        assertEquals("re:quic", String(buf, 0, p.length))
        assertEquals(l.localPort, p.port) // answers come from the relay address mqvpn sent to
        assertTrue(waitFor(1_000) { l.status.up >= 4 && l.status.down >= 7 })
    }

    @Test fun anotherServersKeyIsRefused() {
        val (_, at) = sharer(key = "another home server's share key!".toByteArray())
        val l = link(at)
        assertTrue("refused: ${l.status}", waitFor(8_000) { l.status.state == ShareLink.REFUSED })
        assertTrue(l.status.reason.isNotEmpty())
    }

    @Test fun nobodyThereIsUnreachable() {
        val dead = DatagramSocket(0, lo).also { closers += { it.close() } } // bound, never answers
        val l = link(InetSocketAddress(lo, dead.localPort))
        assertTrue("unreachable: ${l.status}", waitFor(5_000) { l.status.state == ShareLink.UNREACHABLE })
    }

    @Test fun sharerGoneIsLost() {
        val (r, at) = sharer()
        val l = link(at)
        assertTrue(waitFor(5_000) { l.ready })
        r.stop()
        // a stopped sharer says BYE (refused) or just goes quiet (lost)
        assertTrue("gone: ${l.status}", waitFor(6_000) { !l.ready })
    }

    @Test fun pathNamesMatchTailbondd() {
        // agent/internal/agent/shares.go sharePathName: "sh" + hex(sha256(id)[:3])
        assertEquals("sh" + java.security.MessageDigest.getInstance("SHA-256").digest("sim-pixel".toByteArray())
            .copyOf(3).joinToString("") { "%02x".format(it) }, ShareLink.pathName("sim-pixel"))
        assertEquals(8, ShareLink.pathName("x").length)
    }
}
