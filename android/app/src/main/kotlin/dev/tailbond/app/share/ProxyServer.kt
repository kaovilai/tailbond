package dev.tailbond.app.share

import java.io.BufferedInputStream
import java.io.EOFException
import java.io.InputStream
import java.io.OutputStream
import java.net.Inet4Address
import java.net.Inet6Address
import java.net.InetAddress
import java.net.InetSocketAddress
import java.net.ServerSocket
import java.net.Socket
import java.security.MessageDigest
import java.util.Base64
import java.util.concurrent.ConcurrentHashMap
import java.util.concurrent.atomic.AtomicLong
import javax.crypto.Mac
import javax.crypto.spec.SecretKeySpec

/**
 * The phone hub's proxy (docs/phone-hub.md): SOCKS5 (RFC 1928, password
 * RFC 1929; CONNECT only) and HTTP CONNECT on one port. A port of the Go
 * reference, agent/internal/proxy; the login derivation is pinned by its
 * testdata/vectors.json (ProxyServerTest).
 *
 * Login: a hub (the Mac) has the home server's pass for this phone,
 * ShareProtocol.passKey(this phone's key, hub name, expiry); its user is
 * "<hub>/<expiry>" and its password [credentials] of that pass. The phone
 * checks it offline from its own key ([verifier]).
 *
 * Plain JVM (no Android classes): HubService gives it [Config.dial], which
 * opens upstream sockets on tailbond's VPN network, so the traffic goes into
 * the bonded tunnel.
 */
class ProxyServer(private val cfg: Config) {
    data class Config(
        /** Required when set (SOCKS5 method 0x02, Proxy-Authorization: Basic): checks a login ([verifier]). */
        val auth: ((user: String, pass: String) -> Boolean)? = null,
        /** Opens the upstream TCP connection. */
        val dial: (host: String, port: Int) -> Socket = { h, p -> Socket().apply { connect(InetSocketAddress(h, p), 20_000) } },
        /** May this source connect at all? Default: private, link-local, loopback, CGNAT. */
        val allow: (InetAddress) -> Boolean = ::allowPrivate,
        /** May a connection arriving on this local address be served (not mobile data, not the VPN)? */
        val allowLocal: (InetAddress) -> Boolean = { true },
        val log: (String) -> Unit = {},
    )

    /** One connected device, by source address. */
    data class Client(val addr: String, val conns: Int, val up: Long, val down: Long, val lastSeenMs: Long)

    private class Counter(val addr: String) {
        val conns = java.util.concurrent.atomic.AtomicInteger()
        val up = AtomicLong()
        val down = AtomicLong()
        @Volatile var lastSeen = System.currentTimeMillis()
    }

    private val up = AtomicLong()
    private val down = AtomicLong()
    private val clients = ConcurrentHashMap<String, Counter>()
    private val open = ConcurrentHashMap.newKeySet<Socket>()
    @Volatile private var server: ServerSocket? = null

    val upBytes: Long get() = up.get()
    val downBytes: Long get() = down.get()

    fun clients(): List<Client> = clients.values.map { Client(it.addr, it.conns.get(), it.up.get(), it.down.get(), it.lastSeen) }
        .sortedBy { it.addr }

    /** Accepts until [stop]. Blocks. */
    fun serve(ss: ServerSocket) {
        server = ss
        while (!ss.isClosed) {
            val c = try {
                ss.accept()
            } catch (_: Exception) {
                break
            }
            Thread({ handle(c) }, "proxy-conn").apply { isDaemon = true; start() }
        }
    }

    fun stop() {
        try { server?.close() } catch (_: Exception) {}
        for (s in open) try { s.close() } catch (_: Exception) {}
        open.clear()
    }

    private fun handle(c: Socket) {
        open.add(c)
        var upstream: Socket? = null
        try {
            val remote = c.inetAddress
            if (!cfg.allow(remote) || !cfg.allowLocal(c.localAddress)) return
            c.soTimeout = 30_000
            val inp = BufferedInputStream(c.getInputStream())
            val out = c.getOutputStream()
            inp.mark(1)
            val first = inp.read()
            if (first < 0) return
            inp.reset()
            val req = try {
                if (first == 5) socks(inp, out) else httpConnect(inp, out)
            } catch (_: EOFException) {
                return // a client that only checks the password (a hub probe)
            }
            val (host, port, reply) = req ?: return
            upstream = try {
                cfg.dial(host, port)
            } catch (e: Exception) {
                cfg.log("proxy: ${remote.hostAddress} → $host:$port: ${e.message}")
                reply(false)
                return
            }
            open.add(upstream)
            reply(true)
            c.soTimeout = 0
            val cnt = clients.getOrPut(remote.hostAddress ?: "?") { Counter(remote.hostAddress ?: "?") }
            cnt.conns.incrementAndGet()
            cnt.lastSeen = System.currentTimeMillis()
            try {
                val u = upstream
                val t = Thread({
                    pump(inp, u.getOutputStream()) { n -> up.addAndGet(n); cnt.up.addAndGet(n); cnt.lastSeen = System.currentTimeMillis() }
                    try { u.shutdownOutput() } catch (_: Exception) {}
                }, "proxy-up").apply { isDaemon = true; start() }
                pump(u.getInputStream(), out) { n -> down.addAndGet(n); cnt.down.addAndGet(n); cnt.lastSeen = System.currentTimeMillis() }
                try { c.shutdownOutput() } catch (_: Exception) {}
                t.join()
            } finally {
                cnt.conns.decrementAndGet()
            }
        } catch (e: Exception) {
            cfg.log("proxy: ${e.message}")
        } finally {
            try { c.close() } catch (_: Exception) {}
            open.remove(c)
            upstream?.let { try { it.close() } catch (_: Exception) {}; open.remove(it) }
        }
    }

    private fun pump(i: InputStream, o: OutputStream, count: (Long) -> Unit) {
        val buf = ByteArray(32 * 1024)
        try {
            while (true) {
                val n = i.read(buf)
                if (n < 0) break
                o.write(buf, 0, n)
                o.flush()
                count(n.toLong())
            }
        } catch (_: Exception) {
        }
    }

    private fun readFully(i: InputStream, n: Int): ByteArray {
        val b = ByteArray(n)
        var off = 0
        while (off < n) {
            val r = i.read(b, off, n - off)
            if (r < 0) throw EOFException()
            off += r
        }
        return b
    }

    private fun readByte(i: InputStream): Int {
        val b = i.read()
        if (b < 0) throw EOFException()
        return b
    }


    /** SOCKS5 handshake; null when refused (already answered). */
    private fun socks(i: InputStream, o: OutputStream): Triple<String, Int, (Boolean) -> Unit>? {
        val hdr = readFully(i, 2)
        val methods = readFully(i, hdr[1].toInt() and 0xff)
        val want: Byte = if (cfg.auth != null) 2 else 0
        if (want !in methods) {
            o.write(byteArrayOf(5, 0xff.toByte())); o.flush()
            return null
        }
        o.write(byteArrayOf(5, want)); o.flush()
        if (want == 2.toByte()) {
            if (readByte(i) != 1) return null
            val u = String(readFully(i, readByte(i)))
            val p = String(readFully(i, readByte(i)))
            val ok = cfg.auth?.invoke(u, p) == true
            o.write(byteArrayOf(1, if (ok) 0 else 1)); o.flush()
            if (!ok) return null
        }
        val req = readFully(i, 4)
        if (req[0] != 5.toByte()) return null
        val host = when (req[3].toInt()) {
            1 -> InetAddress.getByAddress(readFully(i, 4)).hostAddress!!
            4 -> InetAddress.getByAddress(readFully(i, 16)).hostAddress!!
            3 -> String(readFully(i, readByte(i)))
            else -> return null
        }
        val pb = readFully(i, 2)
        val port = ((pb[0].toInt() and 0xff) shl 8) or (pb[1].toInt() and 0xff)
        if (req[1] != 1.toByte()) { // CONNECT only (UDP ASSOCIATE: docs/phone-hub.md)
            o.write(byteArrayOf(5, 7, 0, 1, 0, 0, 0, 0, 0, 0)); o.flush()
            return null
        }
        return Triple(host, port) { ok: Boolean ->
            o.write(byteArrayOf(5, if (ok) 0 else 5, 0, 1, 0, 0, 0, 0, 0, 0)); o.flush()
        }
    }

    private fun readLine(i: InputStream): String {
        val sb = StringBuilder()
        while (sb.length < 8192) {
            val b = readByte(i)
            if (b == '\n'.code) return sb.toString().trimEnd('\r')
            sb.append(b.toChar())
        }
        throw EOFException()
    }

    private fun httpConnect(i: InputStream, o: OutputStream): Triple<String, Int, (Boolean) -> Unit>? {
        val line = readLine(i)
        val headers = mutableMapOf<String, String>()
        while (true) {
            val h = readLine(i)
            if (h.isEmpty()) break
            val c = h.indexOf(':')
            if (c > 0) headers[h.substring(0, c).trim().lowercase()] = h.substring(c + 1).trim()
        }
        val parts = line.split(' ')
        if (parts.size < 2 || parts[0] != "CONNECT") {
            o.write("HTTP/1.1 405 Method Not Allowed\r\nContent-Length: 0\r\n\r\n".toByteArray()); o.flush()
            return null
        }
        val check = cfg.auth
        if (check != null) {
            val auth = headers["proxy-authorization"].orEmpty()
            val ok = auth.startsWith("Basic ") && try {
                val dec = String(Base64.getDecoder().decode(auth.removePrefix("Basic ").trim()))
                val c = dec.indexOf(':')
                c > 0 && check(dec.substring(0, c), dec.substring(c + 1))
            } catch (_: IllegalArgumentException) {
                false
            }
            if (!ok) {
                o.write("HTTP/1.1 407 Proxy Authentication Required\r\nProxy-Authenticate: Basic realm=\"tailbond\"\r\nContent-Length: 0\r\n\r\n".toByteArray())
                o.flush()
                return null
            }
        }
        val target = parts[1]
        val c = target.lastIndexOf(':')
        if (c <= 0) return null
        val host = target.substring(0, c).removePrefix("[").removeSuffix("]")
        val port = target.substring(c + 1).toIntOrNull() ?: return null
        return Triple(host, port) { ok: Boolean ->
            o.write((if (ok) "HTTP/1.1 200 Connection established\r\n\r\n" else "HTTP/1.1 502 Bad Gateway\r\n\r\n").toByteArray()); o.flush()
        }
    }

    companion object {
        const val SERVICE_TYPE = "_tailbond-proxy._tcp"
        const val DEFAULT_PORT = 1888

        /**
         * A hub's login for a phone, from the pass the home server gave the
         * hub: user "<hubId>/<exp>", password the first 16 characters of
         * base32(HMAC-SHA256(pass, "tbp1-proxy")), lower case.
         */
        fun credentials(pass: ByteArray, hubId: String, exp: Long): Pair<String, String> {
            val mac = Mac.getInstance("HmacSHA256")
            mac.init(SecretKeySpec(pass, "HmacSHA256"))
            val sum = mac.doFinal("tbp1-proxy".toByteArray())
            return "$hubId/$exp" to base32(sum).substring(0, 16).lowercase()
        }

        /**
         * The phone's check of a login: the pass derived from its own share
         * key for the hub and expiry in user, not expired ([nowSec]: unix
         * seconds); [allowed] (optional) limits the hubs by name.
         */
        fun verifier(ownKey: ByteArray, nowSec: () -> Long = { System.currentTimeMillis() / 1000 }, allowed: ((String) -> Boolean)? = null): (String, String) -> Boolean =
            check@{ user, pass ->
                val i = user.indexOf('/')
                if (i <= 0) return@check false
                val hub = user.substring(0, i)
                val exp = user.substring(i + 1).toLongOrNull() ?: return@check false
                if (exp < 0 || exp < nowSec()) return@check false
                if (allowed != null && !allowed(hub)) return@check false
                val want = credentials(ShareProtocol.passKey(ownKey, hub, exp), hub, exp).second
                MessageDigest.isEqual(pass.toByteArray(), want.toByteArray())
            }

        /** RFC 4648 base32, no padding. */
        fun base32(b: ByteArray): String {
            val alpha = "ABCDEFGHIJKLMNOPQRSTUVWXYZ234567"
            val sb = StringBuilder()
            var buf = 0
            var bits = 0
            for (x in b) {
                buf = (buf shl 8) or (x.toInt() and 0xff)
                bits += 8
                while (bits >= 5) {
                    sb.append(alpha[(buf shr (bits - 5)) and 31])
                    bits -= 5
                }
            }
            if (bits > 0) sb.append(alpha[(buf shl (5 - bits)) and 31])
            return sb.toString()
        }

        /** RFC 1918, CGNAT, unique-local, link-local and loopback sources. */
        fun allowPrivate(a: InetAddress): Boolean {
            if (a.isLoopbackAddress || a.isLinkLocalAddress || a.isSiteLocalAddress) return true
            val b = a.address
            return when (a) {
                is Inet4Address -> (b[0].toInt() and 0xff) == 100 && (b[1].toInt() and 0xc0) == 64
                is Inet6Address -> (b[0].toInt() and 0xfe) == 0xfc
                else -> false
            }
        }
    }
}
