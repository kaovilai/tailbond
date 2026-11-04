package dev.tailbond.app

import dev.tailbond.app.share.ProxyServer
import dev.tailbond.app.share.ShareProtocol
import kotlinx.serialization.json.Json
import kotlinx.serialization.json.jsonArray
import kotlinx.serialization.json.jsonObject
import kotlinx.serialization.json.jsonPrimitive
import org.junit.After
import org.junit.Assert.assertEquals
import org.junit.Assert.assertFalse
import org.junit.Assert.assertTrue
import org.junit.Test
import java.io.File
import java.net.InetAddress
import java.net.InetSocketAddress
import java.net.ServerSocket
import java.net.Socket
import java.util.Base64

/** The phone hub's proxy against the Go reference (agent/internal/proxy). */
class ProxyServerTest {
    private val closers = mutableListOf<() -> Unit>()
    private val lo: InetAddress = InetAddress.getLoopbackAddress()

    @After fun tearDown() = closers.forEach { it() }

    @Test fun credentialsMatchTheGoReference() {
        val f = listOf("../../agent/internal/proxy/testdata/vectors.json", "../agent/internal/proxy/testdata/vectors.json")
            .map { File(it) }.first { it.exists() }
        val vs = Json.parseToJsonElement(f.readText()).jsonObject["credentials"]!!.jsonArray
        assertTrue(vs.size >= 3)
        for (v in vs) {
            val o = v.jsonObject
            val key = o["share_key"]!!.jsonPrimitive.content.toByteArray()
            val hub = o["hub"]!!.jsonPrimitive.content
            val exp = o["exp"]!!.jsonPrimitive.content.toLong()
            val pk = ShareProtocol.passKey(key, hub, exp)
            assertEquals(o["pass_key"]!!.jsonPrimitive.content, pk.joinToString("") { "%02x".format(it) })
            val (u, p) = ProxyServer.credentials(pk, hub, exp)
            assertEquals(o["user"]!!.jsonPrimitive.content, u)
            assertEquals(o["pass"]!!.jsonPrimitive.content, p)
            assertTrue("the phone accepts it", ProxyServer.verifier(key, { 0L })(u, p))
        }
        // RFC 4648 test vectors (no padding)
        assertEquals("MZXW6YTBOI", ProxyServer.base32("foobar".toByteArray()))
        assertEquals("MY", ProxyServer.base32("f".toByteArray()))
    }

    private fun echo(): InetSocketAddress {
        val ss = ServerSocket(0, 50, lo)
        closers += { ss.close() }
        Thread {
            while (!ss.isClosed) {
                val c = try { ss.accept() } catch (_: Exception) { break }
                Thread {
                    c.use {
                        val b = ByteArray(64)
                        val n = it.getInputStream().read(b)
                        it.getOutputStream().write("re:".toByteArray() + b.copyOf(maxOf(n, 0)))
                    }
                }.start()
            }
        }.apply { isDaemon = true }.start()
        return ss.localSocketAddress as InetSocketAddress
    }

    private fun serve(cfg: ProxyServer.Config): Pair<ProxyServer, Int> {
        val p = ProxyServer(cfg)
        val ss = ServerSocket(0, 50, lo)
        closers += { p.stop() }
        Thread { p.serve(ss) }.apply { isDaemon = true }.start()
        return p to ss.localPort
    }

    /** SOCKS5 CONNECT by name; null with the step that failed. */
    private fun socks(port: Int, user: String, pass: String, target: InetSocketAddress): Pair<Socket?, String> {
        val c = Socket(lo, port)
        c.soTimeout = 5_000
        val i = c.getInputStream()
        val o = c.getOutputStream()
        val m: Byte = if (user.isEmpty()) 0 else 2
        o.write(byteArrayOf(5, 1, m))
        val r = ByteArray(2)
        if (i.read(r) != 2 || r[1] != m) { c.close(); return null to "method" }
        if (user.isNotEmpty()) {
            o.write(byteArrayOf(1, user.length.toByte()) + user.toByteArray() + byteArrayOf(pass.length.toByte()) + pass.toByteArray())
            if (i.read(r) != 2 || r[1] != 0.toByte()) { c.close(); return null to "auth" }
        }
        val host = target.hostString.toByteArray()
        o.write(byteArrayOf(5, 1, 0, 3, host.size.toByte()) + host + byteArrayOf((target.port shr 8).toByte(), target.port.toByte()))
        val rep = ByteArray(10)
        var n = 0
        while (n < 10) { val k = i.read(rep, n, 10 - n); if (k < 0) break; n += k }
        if (n != 10 || rep[1] != 0.toByte()) { c.close(); return null to "connect" }
        return c to ""
    }

    private fun roundTrip(c: Socket): String {
        c.getOutputStream().write("hello".toByteArray())
        val b = ByteArray(64)
        val n = c.getInputStream().read(b)
        return String(b, 0, maxOf(n, 0))
    }

    @Test fun socksWithThePassword() {
        val phone = "0123456789abcdef0123456789abcdef".toByteArray()
        val exp = System.currentTimeMillis() / 1000 + 86_400
        val (u, p) = ProxyServer.credentials(ShareProtocol.passKey(phone, "mac", exp), "mac", exp)
        val (srv, port) = serve(ProxyServer.Config(auth = ProxyServer.verifier(phone)))
        val target = echo()
        val (c, step) = socks(port, u, p, target)
        assertEquals("", step)
        assertEquals("re:hello", roundTrip(c!!))
        c.close()
        assertEquals("auth", socks(port, u, "wrong", target).second)
        assertEquals("method", socks(port, "", "", target).second)
        assertEquals("auth", socks(port, "laptop/$exp", p, target).second) // another hub's name
        val old = System.currentTimeMillis() / 1000 - 60
        val (ou, op) = ProxyServer.credentials(ShareProtocol.passKey(phone, "mac", old), "mac", old)
        assertEquals("auth", socks(port, ou, op, target).second) // expired
        Thread.sleep(100)
        assertTrue("up ${srv.upBytes}", srv.upBytes >= 5)
        assertTrue("down ${srv.downBytes}", srv.downBytes >= 8)
        assertEquals(1, srv.clients().size)
    }

    @Test fun httpConnect() {
        val (_, port) = serve(ProxyServer.Config(auth = { u, p -> u == "mac/9" && p == "pw" }))
        val target = echo()
        fun attempt(auth: String?): Pair<Socket, String> {
            val c = Socket(lo, port)
            c.soTimeout = 5_000
            var req = "CONNECT ${target.hostString}:${target.port} HTTP/1.1\r\nHost: x\r\n"
            if (auth != null) req += "Proxy-Authorization: Basic ${Base64.getEncoder().encodeToString(auth.toByteArray())}\r\n"
            c.getOutputStream().write("$req\r\n".toByteArray())
            val b = ByteArray(256)
            val n = c.getInputStream().read(b)
            return c to String(b, 0, maxOf(n, 0))
        }
        assertTrue(attempt(null).second.contains("407"))
        val (c, line) = attempt("mac/9:pw")
        assertTrue(line, line.contains("200"))
        assertEquals("re:hello", roundTrip(c))
        c.close()
    }

    @Test fun refusedSourcesAndLocalAddresses() {
        val (_, port) = serve(ProxyServer.Config(allow = { false }))
        assertEquals("method", socks(port, "", "", echo()).second)
        val (_, port2) = serve(ProxyServer.Config(allowLocal = { false }))
        assertEquals("method", socks(port2, "", "", echo()).second)
        for ((a, want) in mapOf(
            "192.168.43.20" to true, "172.20.10.3" to true, "100.64.1.2" to true, "fe80::1" to true, "127.0.0.1" to true,
            "10.1.2.3" to true, "fd00::1" to true, "8.8.8.8" to false, "2001:db8::1" to false, "203.0.113.9" to false,
        )) assertEquals(a, want, ProxyServer.allowPrivate(InetAddress.getByName(a)))
    }

    @Test fun udpAssociateRefused() {
        val (_, port) = serve(ProxyServer.Config())
        Socket(lo, port).use { c ->
            c.soTimeout = 5_000
            c.getOutputStream().write(byteArrayOf(5, 1, 0))
            val r = ByteArray(2)
            c.getInputStream().read(r)
            c.getOutputStream().write(byteArrayOf(5, 3, 0, 1, 0, 0, 0, 0, 0, 0))
            val rep = ByteArray(10)
            var n = 0
            while (n < 10) { val k = c.getInputStream().read(rep, n, 10 - n); if (k < 0) break; n += k }
            assertEquals(7, rep[1].toInt())
        }
    }

    @Test fun dialHookIsTheUpstream() {
        var dialed = ""
        val (_, port) = serve(ProxyServer.Config(dial = { h, p -> dialed = "$h:$p"; Socket(h, p) }))
        val target = echo()
        val (c, _) = socks(port, "", "", target)
        c!!.close()
        assertEquals("${target.hostString}:${target.port}", dialed)
        val (_, port2) = serve(ProxyServer.Config(dial = { _, _ -> throw java.io.IOException("no tunnel") }))
        assertEquals("connect", socks(port2, "", "", target).second)
        assertFalse(dialed.isEmpty())
    }
}
