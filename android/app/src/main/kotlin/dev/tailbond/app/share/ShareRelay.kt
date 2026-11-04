package dev.tailbond.app.share

import dev.tailbond.app.share.ShareProtocol as P
import java.net.DatagramPacket
import java.net.DatagramSocket
import java.net.SocketAddress
import java.net.SocketException
import java.net.SocketTimeoutException
import java.security.SecureRandom
import java.util.concurrent.ConcurrentHashMap
import java.util.concurrent.atomic.AtomicLong

/**
 * The sharer's relay (Pair & Share): hubs on the LAN open a session with an
 * HMAC challenge on their pass for this phone (derived from its share key,
 * [ShareProtocol.passKey]), then their mqvpn datagrams
 * go to the home server over [openUpstream] (the cellular network) and back.
 * The relay only ever talks to that one server. Plain JVM (sockets are
 * passed in), so it is unit tested over loopback.
 */
class ShareRelay(private val cfg: Config) {

    data class Config(
        val id: String,
        val label: String,
        /** This phone's own share key (its pairing's). */
        val key: ByteArray,
        /** A UDP socket to the home server over cellular (bound, connected), per session. */
        val openUpstream: () -> DatagramSocket,
        /** Stop once this many payload bytes went through (0 = no cap); [usedBytes] were used before. [updateLimits] changes it. */
        val capBytes: Long = 0,
        val usedBytes: Long = 0,
        /** Hub names allowed; empty = any device paired with the same server. [updateLimits] changes it. */
        val allowedHubs: Set<String> = emptySet(),
        /** May sharing go on? A reason ([P.R_NONE] = yes), battery percent (-1 unknown), charging. */
        val check: () -> Check = { Check(P.R_NONE, -1, false) },
        /** A hub came or went, or the counters moved (called from relay threads). */
        val onChange: () -> Unit = {},
        val log: (String) -> Unit = {},
        /** The clock passes are checked against (ms). */
        val now: () -> Long = System::currentTimeMillis,
    )

    data class Check(val reason: Int, val battery: Int, val charging: Boolean)

    data class HubUse(val hub: String, val addr: String, val upBytes: Long, val downBytes: Long, val sinceMs: Long)

    private class Pending(val sharerNonce: ByteArray, val hubId: String, val exp: Long, val addr: SocketAddress, val atMs: Long)

    /** [exp]: the hub's pass expiry (unix seconds); the session ends with it. */
    private inner class Session(val id: ByteArray, val hubId: String, val exp: Long, val hubAddr: SocketAddress, val up: DatagramSocket) {
        val upBytes = AtomicLong()
        val downBytes = AtomicLong()
        @Volatile var lastSeenMs = System.currentTimeMillis()
        val sinceMs = System.currentTimeMillis()
        @Volatile var closed = false
        val key get() = id.toKey()
    }

    private val rnd = SecureRandom()
    private val pending = ConcurrentHashMap<String, Pending>()
    private val sessions = ConcurrentHashMap<String, Session>()
    private val used = AtomicLong(cfg.usedBytes)
    @Volatile private var capBytes = cfg.capBytes
    @Volatile private var allowedHubs = cfg.allowedHubs
    @Volatile private var lan: DatagramSocket? = null
    @Volatile private var running = false

    val usedBytes: Long get() = used.get()

    fun hubs(): List<HubUse> = sessions.values.map { HubUse(it.hubId, it.hubAddr.toString(), it.upBytes.get(), it.downBytes.get(), it.sinceMs) }
        .sortedBy { it.sinceMs }

    private fun capLeft(): Long {
        val cap = capBytes
        if (cap <= 0) return P.NO_CAP
        return (cap - used.get()).coerceAtLeast(0)
    }

    private fun expired(exp: Long): Boolean = cfg.now() / 1000 > exp

    /**
     * New limits, at once: hubs no longer allowed are dropped, and a cap
     * now reached ends every session (a raised one lets hubs back in).
     */
    fun updateLimits(capBytes: Long, allowedHubs: Set<String>) {
        this.capBytes = capBytes
        this.allowedHubs = allowedHubs
        if (allowedHubs.isNotEmpty()) sessions.values.filter { it.hubId !in allowedHubs }.forEach { end(it, P.R_NOT_ALLOWED, true) }
        if (capLeft() == 0L) closeAll(P.R_CAP)
    }

    /** A new month: the cap counts from zero again. */
    fun resetUsage() {
        used.set(0)
        cfg.onChange()
    }

    /** Adds [n] payload bytes (negative: takes them back) to [c] and the total. */
    private fun count(c: AtomicLong, n: Long) {
        c.addAndGet(n)
        used.addAndGet(n)
    }

    /** Serve hubs on [socket] (the LAN listener) until [stop]; blocks. */
    fun serve(socket: DatagramSocket) {
        lan = socket
        running = true
        socket.soTimeout = 1000
        val buf = ByteArray(65536)
        val pkt = DatagramPacket(buf, buf.size)
        var lastJanitor = 0L
        try {
            while (running) {
                try {
                    pkt.setLength(buf.size)
                    socket.receive(pkt)
                    handle(buf, pkt.length, pkt.socketAddress)
                } catch (_: SocketTimeoutException) {
                }
                val now = System.currentTimeMillis()
                if (now - lastJanitor >= 2000) {
                    lastJanitor = now
                    janitor(now)
                }
            }
        } catch (e: SocketException) {
            if (running) cfg.log("share: LAN socket: ${e.message}")
        } finally {
            closeAll(P.R_PAUSED)
            running = false
        }
    }

    /** Ends every session (telling the hubs why) and stops [serve]. */
    fun stop(reason: Int = P.R_PAUSED) {
        closeAll(reason)
        running = false
        lan?.close()
    }

    private fun send(b: ByteArray, to: SocketAddress) {
        try {
            lan?.send(DatagramPacket(b, b.size, to))
        } catch (_: Exception) {
        }
    }

    private fun handle(b: ByteArray, n: Int, from: SocketAddress) {
        if (n < 1) return
        when (b[0]) {
            P.DATA -> {
                val s = sessionFrom(b, n, from) ?: return
                if (capLeft() == 0L) {
                    end(s, P.R_CAP, true)
                    return
                }
                val len = n - P.DATA_HEADER
                // counted before it goes: the server's reply can reach the hub
                // before send() returns, and the counters must include it
                count(s.upBytes, len.toLong())
                try {
                    s.up.send(DatagramPacket(b, P.DATA_HEADER, len))
                } catch (_: Exception) {
                    count(s.upBytes, -len.toLong())
                }
                s.lastSeenMs = System.currentTimeMillis()
            }
            P.KEEPALIVE -> {
                val s = sessionFrom(b, n, from) ?: return
                if (n < 1 + P.SESSION + 4) return
                s.lastSeenMs = System.currentTimeMillis()
                val c = cfg.check()
                val reason = if (c.reason == P.R_NONE && capLeft() == 0L) P.R_CAP else c.reason
                if (reason != P.R_NONE) {
                    end(s, reason, true)
                    return
                }
                val battery = if (c.battery in 0..100) c.battery else 255
                send(P.kaAck(s.id, P.seqOf(b), s.upBytes.get(), s.downBytes.get(), capLeft(), battery, if (c.charging) P.FLAG_CHARGING else 0), from)
                cfg.onChange()
            }
            P.BYE -> sessionFrom(b, n, from)?.let { end(it, P.R_NONE, false) }
            P.HELLO -> {
                val h = P.parseHello(b, n) ?: return
                if (pending.size >= 64) return
                val sn = ByteArray(P.NONCE).also { rnd.nextBytes(it) }
                pending[h.hubNonce.toKey()] = Pending(sn, h.hubId, h.exp, from, System.currentTimeMillis())
                send(P.challenge(h.hubNonce, sn, cfg.id, cfg.label), from)
            }
            P.AUTH -> {
                val a = P.parseAuth(b, n) ?: return
                val p = pending.remove(a.hubNonce.toKey()) ?: return
                // the hub's pass for this phone: bound to its name and expiry,
                // so a hub can't use another hub's name
                val pass = P.passKey(cfg.key, p.hubId, p.exp)
                // a wrong key, a replay or a stranger gets no answer at all
                if (!p.sharerNonce.contentEquals(a.sharerNonce) || p.addr != from ||
                    !P.macEquals(a.mac, P.authMac(pass, a.hubNonce, a.sharerNonce, p.hubId))
                ) return
                var reason = cfg.check().reason
                if (expired(p.exp)) reason = P.R_EXPIRED
                val allowed = allowedHubs
                if (reason == P.R_NONE && allowed.isNotEmpty() && p.hubId !in allowed) reason = P.R_NOT_ALLOWED
                if (reason == P.R_NONE && capLeft() == 0L) reason = P.R_CAP
                if (reason != P.R_NONE) {
                    send(P.refuse(a.hubNonce, reason, P.refuseMac(pass, a.hubNonce, a.sharerNonce, reason)), from)
                    cfg.log("share: refused ${p.hubId} (${P.reasonText(reason)})")
                    return
                }
                val s = try {
                    open(p.hubId, p.exp, from)
                } catch (e: Exception) {
                    cfg.log("share: no mobile data for ${p.hubId}: ${e.message}")
                    send(P.refuse(a.hubNonce, P.R_NO_CELLULAR, P.refuseMac(pass, a.hubNonce, a.sharerNonce, P.R_NO_CELLULAR)), from)
                    return
                }
                val cap = capLeft()
                send(P.welcome(a.hubNonce, s.id, cap, P.welcomeMac(pass, a.hubNonce, a.sharerNonce, s.id, cap)), from)
                cfg.log("share: ${p.hubId} ($from) connected")
                cfg.onChange()
            }
        }
    }

    private fun sessionFrom(b: ByteArray, n: Int, from: SocketAddress): Session? {
        val id = P.sessionOf(b, n) ?: return null
        val s = sessions[id.toKey()] ?: return null
        return if (s.hubAddr == from) s else null
    }

    private fun open(hubId: String, exp: Long, from: SocketAddress): Session {
        val old = sessions.values.firstOrNull { it.hubId == hubId }
        if (old == null && sessions.size >= 8) throw IllegalStateException("too many hubs")
        val up = cfg.openUpstream()
        val s = Session(ByteArray(P.SESSION).also { rnd.nextBytes(it) }, hubId, exp, from, up)
        sessions[s.key] = s
        old?.let { end(it, P.R_REPLACED, true) }
        Thread({ pumpDown(s) }, "share-down-$hubId").apply { isDaemon = true }.start()
        return s
    }

    /** server → hub */
    private fun pumpDown(s: Session) {
        val buf = ByteArray(65536)
        val out = ByteArray(65536 + P.DATA_HEADER)
        val pkt = DatagramPacket(buf, buf.size)
        try {
            s.up.soTimeout = 1000
        } catch (_: Exception) {
        }
        while (!s.closed) {
            try {
                pkt.setLength(buf.size)
                s.up.receive(pkt)
                val len = P.data(out, s.id, buf, 0, pkt.length)
                count(s.downBytes, pkt.length.toLong()) // before it goes, as upstream
                try {
                    lan?.send(DatagramPacket(out, len, s.hubAddr))
                } catch (e: Exception) {
                    count(s.downBytes, -pkt.length.toLong())
                    throw e
                }
            } catch (_: SocketTimeoutException) {
            } catch (_: Exception) {
                if (!s.closed) end(s, P.R_NO_CELLULAR, true)
                return
            }
        }
    }

    private fun end(s: Session, reason: Int, bye: Boolean) {
        synchronized(s) {
            if (s.closed) return
            s.closed = true
        }
        sessions.remove(s.key)
        s.up.close()
        if (bye) send(P.bye(s.id, reason), s.hubAddr)
        cfg.log("share: ${s.hubId} left (${P.reasonText(reason)})")
        cfg.onChange()
    }

    private fun closeAll(reason: Int) {
        sessions.values.toList().forEach { end(it, reason, true) }
    }

    private fun janitor(now: Long) {
        pending.entries.removeIf { now - it.value.atMs > 10_000 }
        // a removed hub gets no new pass: its sessions end with its last one
        sessions.values.filter { expired(it.exp) }.forEach { end(it, P.R_EXPIRED, true) }
        sessions.values.filter { now - it.lastSeenMs > 30_000 }.forEach { end(it, P.R_IDLE, true) }
        val r = cfg.check().reason
        if (r != P.R_NONE) closeAll(r) else if (capLeft() == 0L) closeAll(P.R_CAP)
    }

    private fun ByteArray.toKey(): String = joinToString("") { "%02x".format(it) }
}
