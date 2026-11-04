package dev.tailbond.app.share

import dev.tailbond.app.share.ShareProtocol as P
import java.net.DatagramPacket
import java.net.DatagramSocket
import java.net.InetAddress
import java.net.InetSocketAddress
import java.nio.ByteBuffer
import java.security.SecureRandom

/**
 * The hub's end of one share (Pair & Share, docs/pair-and-share.md): a port
 * of agent/internal/share/link.go. mqvpn's relay path sends to [localPort]
 * (127.0.0.1); the link frames each datagram for the sharer, and the
 * sharer's frames go back to mqvpn's socket.
 *
 * Plain JVM: the app gives it [Config.openLan], a socket bound to the Wi-Fi
 * network and kept out of tailbond's own VPN.
 */
class ShareLink(private val cfg: Config) {
    data class Config(
        val sharer: InetSocketAddress,
        val key: ByteArray,
        val hubId: String,
        /** The LAN socket toward the sharer. */
        val openLan: () -> DatagramSocket = { DatagramSocket() },
        /** Keep mqvpn's relay port when the sharer moves (0 = any). */
        val localPort: Int = 0,
        val log: (String) -> Unit = {},
        /** Called (no locks held) when the state changes. */
        val onChange: (Status) -> Unit = {},
    )

    data class Status(
        val state: String = CONNECTING,
        val sharerId: String = "",
        val label: String = "",
        val addr: String = "",
        val reason: String = "",
        val up: Long = 0,
        val down: Long = 0,
        val capLeft: Long = P.NO_CAP,
        val battery: Int = -1,
        val charging: Boolean = false,
        val unanswered: Int = 0,
    )

    private val lock = Object()
    private val rnd = SecureRandom()
    private val local = bindLocal(cfg.localPort)
    private val lan = cfg.openLan()
    @Volatile private var st = Status(addr = "${cfg.sharer.hostString}:${cfg.sharer.port}")
    private val hubNonce = ByteArray(P.NONCE)
    private var sharerNonce = ByteArray(P.NONCE)
    private var session = ByteArray(P.SESSION)
    @Volatile private var mqvpn: InetSocketAddress? = null
    private var lastAck = 0L
    private var retryAt = 0L
    private var seq = 0
    private var authTries = 0
    @Volatile private var running = true

    val localPort: Int get() = local.localPort
    val status: Status get() = st
    val ready: Boolean get() = st.state == READY

    private fun bindLocal(port: Int): DatagramSocket {
        val lo = InetAddress.getByName("127.0.0.1")
        return try {
            DatagramSocket(InetSocketAddress(lo, port))
        } catch (e: Exception) {
            if (port == 0) throw e
            DatagramSocket(InetSocketAddress(lo, 0))
        }
    }

    /** Starts its threads. */
    fun start() {
        Thread({ fromLan() }, "share-link-lan").apply { isDaemon = true; start() }
        Thread({ fromMqvpn() }, "share-link-mqvpn").apply { isDaemon = true; start() }
        Thread({
            while (running) {
                tick(System.currentTimeMillis())
                try { Thread.sleep(HELLO_EVERY_MS) } catch (_: InterruptedException) { break }
            }
        }, "share-link-tick").apply { isDaemon = true; start() }
    }

    /** Says BYE when ready, then closes. */
    fun stop() {
        running = false
        val (s, r) = synchronized(lock) { session.copyOf() to (st.state == READY) }
        if (r) send(P.bye(s, P.R_PAUSED))
        lan.close()
        local.close()
    }

    private fun send(b: ByteArray, n: Int = b.size) {
        try { lan.send(DatagramPacket(b, n, cfg.sharer)) } catch (_: Exception) {}
    }

    private inline fun change(f: () -> Unit) {
        val before: String
        val after: Status
        synchronized(lock) {
            before = st.state
            f()
            after = st
        }
        if (after.state != before) cfg.onChange(after)
    }

    internal fun tick(now: Long) = change {
        when (st.state) {
            READY -> {
                if (now - lastAck > LINK_TIMEOUT_MS) {
                    st = st.copy(state = LOST, reason = P.reasonText(P.R_IDLE))
                    cfg.log("share: ${st.label} (${st.addr}) stopped answering")
                    hello(now)
                    return@change
                }
                if (now - lastAck >= KEEPALIVE_EVERY_MS / 2 || seq == 0) {
                    seq++
                    send(P.keepalive(session, seq))
                }
            }
            REFUSED -> if (now > retryAt) hello(now)
            else -> hello(now)
        }
    }

    /** (Re)starts the handshake. Holds lock. */
    private fun hello(now: Long) {
        if (st.state == REFUSED) st = st.copy(state = CONNECTING)
        if (authTries >= 3) {
            authTries = 0
            st = st.copy(state = REFUSED, reason = "it's paired with a different home server")
            retryAt = now + REFUSED_RETRY_MS
            return
        }
        rnd.nextBytes(hubNonce)
        var u = st.unanswered + 1
        val state = if (u >= 3 && st.state != LOST && st.state != REFUSED) UNREACHABLE else st.state
        st = st.copy(unanswered = u, state = state)
        send(P.hello(hubNonce, cfg.hubId))
    }

    private fun fromLan() {
        val buf = ByteArray(65536)
        val p = DatagramPacket(buf, buf.size)
        while (running) {
            try {
                p.setLength(buf.size)
                lan.receive(p)
            } catch (_: Exception) {
                if (!running || lan.isClosed) return
                continue
            }
            val n = p.length
            if (n < 1 || p.address != cfg.sharer.address || p.port != cfg.sharer.port) continue
            if (buf[0] == P.DATA) {
                var dst: InetSocketAddress?
                val ok = synchronized(lock) {
                    val good = st.state == READY && n >= P.DATA_HEADER && sameSession(buf)
                    if (good) st = st.copy(down = st.down + (n - P.DATA_HEADER))
                    dst = mqvpn
                    good
                }
                val d = dst
                if (ok && d != null) {
                    try { local.send(DatagramPacket(buf, P.DATA_HEADER, n - P.DATA_HEADER, d)) } catch (_: Exception) {}
                }
                continue
            }
            control(buf.copyOf(n), System.currentTimeMillis())
        }
    }

    private fun sameSession(b: ByteArray): Boolean {
        for (i in 0 until P.SESSION) if (b[1 + i] != session[i]) return false
        return true
    }

    internal fun control(b: ByteArray, now: Long) = change {
        val n = b.size
        when (b[0]) {
            P.CHALLENGE -> {
                // type, ver, hub nonce, sharer nonce, id len, id, label len, label
                if (n < 2 + 2 * P.NONCE + 2 || st.state == READY) return@change
                if (!b.copyOfRange(2, 2 + P.NONCE).contentEquals(hubNonce)) return@change
                val sn = b.copyOfRange(2 + P.NONCE, 2 + 2 * P.NONCE)
                var o = 2 + 2 * P.NONCE
                val idLen = b[o].toInt() and 0xff
                if (n < o + 1 + idLen + 1) return@change
                val id = String(b, o + 1, idLen, Charsets.UTF_8)
                o += 1 + idLen
                val labLen = b[o].toInt() and 0xff
                if (n < o + 1 + labLen) return@change
                val label = String(b, o + 1, labLen, Charsets.UTF_8)
                sharerNonce = sn
                authTries++
                st = st.copy(unanswered = 0, sharerId = id, label = label)
                send(P.auth(hubNonce, sn, P.authMac(cfg.key, hubNonce, sn, cfg.hubId)))
            }
            P.WELCOME -> {
                // type, ver, hub nonce, session, cap (8), mac
                if (n < 2 + P.NONCE + P.SESSION + 8 + P.MAC || st.state == READY) return@change
                if (!b.copyOfRange(2, 2 + P.NONCE).contentEquals(hubNonce)) return@change
                val s = b.copyOfRange(2 + P.NONCE, 2 + P.NONCE + P.SESSION)
                val cap = ByteBuffer.wrap(b, 2 + P.NONCE + P.SESSION, 8).long
                val mac = b.copyOfRange(2 + P.NONCE + P.SESSION + 8, 2 + P.NONCE + P.SESSION + 8 + P.MAC)
                if (!P.macEquals(mac, P.welcomeMac(cfg.key, hubNonce, sharerNonce, s, cap))) return@change
                session = s
                authTries = 0
                lastAck = now
                seq = 0
                st = st.copy(state = READY, reason = "", unanswered = 0, capLeft = cap)
                cfg.log("share: ${st.label} (${st.addr}) ready")
            }
            P.REFUSE -> {
                // type, ver, hub nonce, reason, mac
                if (n < 2 + P.NONCE + 1 + P.MAC) return@change
                if (!b.copyOfRange(2, 2 + P.NONCE).contentEquals(hubNonce)) return@change
                val r = b[2 + P.NONCE].toInt() and 0xff
                val mac = b.copyOfRange(3 + P.NONCE, 3 + P.NONCE + P.MAC)
                if (!P.macEquals(mac, P.refuseMac(cfg.key, hubNonce, sharerNonce, r))) return@change
                st = st.copy(state = REFUSED, reason = P.reasonText(r), unanswered = 0)
                retryAt = now + REFUSED_RETRY_MS
                cfg.log("share: ${st.label} (${st.addr}) refused: ${st.reason}")
            }
            P.KA_ACK -> {
                // type, session, seq (4), up, down, cap (8 each), battery, flags
                if (n < 1 + P.SESSION + 4 + 24 + 2 || st.state != READY || !sameSession(b)) return@change
                val o = 1 + P.SESSION + 4
                val cap = ByteBuffer.wrap(b, o + 16, 8).long
                val bat = b[o + 24].toInt() and 0xff
                val flags = b[o + 25].toInt() and 0xff
                lastAck = now
                st = st.copy(capLeft = cap, battery = if (bat <= 100) bat else -1, charging = flags and P.FLAG_CHARGING != 0)
            }
            P.BYE -> {
                if (n < 1 + P.SESSION + 1 || st.state != READY || !sameSession(b)) return@change
                val r = b[1 + P.SESSION].toInt() and 0xff
                st = st.copy(state = REFUSED, reason = P.reasonText(r))
                retryAt = if (r == P.R_REPLACED || r == P.R_IDLE) now else now + REFUSED_RETRY_MS
                cfg.log("share: ${st.label} (${st.addr}) ended: ${st.reason}")
            }
        }
    }

    private fun fromMqvpn() {
        val buf = ByteArray(65536)
        val out = ByteArray(65536 + P.DATA_HEADER)
        val p = DatagramPacket(buf, buf.size)
        while (running) {
            try {
                p.setLength(buf.size)
                local.receive(p)
            } catch (_: Exception) {
                if (!running || local.isClosed) return
                continue
            }
            val n = p.length
            val s: ByteArray
            val ready = synchronized(lock) {
                mqvpn = InetSocketAddress(p.address, p.port)
                val r = st.state == READY
                if (r) st = st.copy(up = st.up + n)
                s = session
                r
            }
            if (!ready) continue // mqvpn retries; the path comes up once the session does
            val len = P.data(out, s, buf, 0, n)
            send(out, len)
        }
    }

    companion object {
        const val CONNECTING = "connecting"
        const val READY = "ready"
        const val REFUSED = "refused"
        const val UNREACHABLE = "unreachable"
        const val LOST = "lost"
        const val HELLO_EVERY_MS = 1_000L
        const val KEEPALIVE_EVERY_MS = 1_000L
        const val LINK_TIMEOUT_MS = 3_500L
        const val REFUSED_RETRY_MS = 30_000L

        /** The path name for a sharer: "sh" + 6 hex of SHA-256(id), as tailbondd names it. */
        fun pathName(sharerId: String): String {
            val h = java.security.MessageDigest.getInstance("SHA-256").digest(sharerId.toByteArray())
            return "sh" + h.copyOf(3).joinToString("") { "%02x".format(it) }
        }
    }
}
