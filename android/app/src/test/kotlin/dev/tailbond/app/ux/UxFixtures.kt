package dev.tailbond.app.ux

import dev.tailbond.app.data.PeerList
import dev.tailbond.app.data.PeerNode
import dev.tailbond.app.model.PathKind
import dev.tailbond.app.pairing.PairingPayload
import dev.tailbond.app.service.ConnState
import dev.tailbond.app.service.ErrorKind
import dev.tailbond.app.service.Live
import dev.tailbond.app.service.PathGate
import dev.tailbond.app.service.PathSample
import kotlin.math.sin

/** Fake state for the screen renders: no tunnel, no Keystore, no network. */
object UxFixtures {
    val profile = PairingPayload(
        name = "pixel-8",
        server = "home.example.net:443",
        sni = "homebox.tail1234.ts.net",
        user = "pixel-8",
        key = "not-a-real-key",
        dns = listOf("10.77.0.1"),
        mode = "speed",
        tailnetRoutes = listOf("100.64.0.0/10", "fd7a:115c:a1e0::/48"),
    )

    private fun wave(base: Long, amp: Long, phase: Double = 0.0) =
        List(60) { i -> base + (amp * sin(i / 6.0 + phase)).toLong().coerceAtLeast(-base) }

    private fun path(name: String, kind: PathKind, status: Int, rx: Long, tx: Long, rtt: Long, phase: Double = 0.0) =
        PathSample(
            name = name, kind = kind, status = status, srttMs = rtt,
            txBps = tx, rxBps = rx, txBytes = tx * 40, rxBytes = rx * 40,
            history = if (status == PathGate.STATUS_ACTIVE) wave(rx + tx, (rx + tx) / 3, phase) else emptyList(),
        )

    val wifi = path("wifi-100", PathKind.WIFI, PathGate.STATUS_ACTIVE, 38_400_000, 4_100_000, 21)
    val cell = path("cellular-101", PathKind.CELLULAR, PathGate.STATUS_ACTIVE, 22_700_000, 2_900_000, 48, 1.3)
    val cellFailed = path("cellular-101", PathKind.CELLULAR, PathGate.STATUS_CLOSED, 0, 0, 0)
    val cellStandby = path("cellular-101", PathKind.CELLULAR, -1, 0, 0, 0)
    val wifiFailed = path("wifi-100", PathKind.WIFI, PathGate.STATUS_CLOSED, 0, 0, 0)
    val cellCarrying = path("cellular-101", PathKind.CELLULAR, PathGate.STATUS_ACTIVE, 31_000_000, 3_400_000, 52, 0.4)

    private fun connected(vararg paths: PathSample, standby: Boolean = false): Live {
        val rx = paths.sumOf { it.rxBps }
        val tx = paths.sumOf { it.txBps }
        return Live(
            state = ConnState.CONNECTED,
            tunnelIp = "10.77.0.2",
            connectedAtMs = System.currentTimeMillis() - 754_000,
            rxBps = rx, txBps = tx, rxBytes = 1_840_000_000, txBytes = 212_000_000,
            lossPct = 0.3, srttMs = 27,
            rxHistory = wave(rx, rx / 4), txHistory = wave(tx, tx / 4, 0.8),
            paths = paths.toList(), standbyActive = standby,
        )
    }

    val connectedBoth = connected(wifi, cell)
    val connecting = Live(state = ConnState.CONNECTING)
    val disconnected = Live()
    val errorUnreachable = Live(state = ConnState.ERROR, message = "Connection to the server failed", errorKind = ErrorKind.UNREACHABLE)
    val errorTimeout = Live(state = ConnState.ERROR, message = "Connection timeout", errorKind = ErrorKind.UNREACHABLE)
    val errorCert = Live(state = ConnState.ERROR, message = "TLS handshake failed", errorKind = ErrorKind.IDENTITY)
    val onePathFailed = connected(wifi, cellFailed)
    /** Failover mode, Wi-Fi primary healthy, cellular held back in standby. */
    val failoverStandby = connected(wifi, cellStandby)
    /** Failover mode, Wi-Fi gone, standby cellular took over. */
    val failoverTakeover = connected(wifiFailed, cellCarrying, standby = true)

    private fun node(host: String, ip: String, os: String, online: Boolean = true, lastSeen: String? = null) =
        PeerNode(host, "$host.tail1234.ts.net", listOf(ip, "fd7a:115c:a1e0::" + ip.substringAfterLast('.')), os, online, lastSeen)

    val peers = PeerList(
        server = "homebox",
        tailnet = "alex@example.com",
        magicDnsSuffix = "tail1234.ts.net",
        tailscaleRunning = true,
        self = node("homebox", "100.101.1.1", "linux"),
        peers = listOf(
            node("apple-tv", "100.101.4.4", "tvOS"),
            node("nas", "100.101.2.2", "linux"),
            node("pixel-tablet", "100.101.5.5", "android"),
            node("work-laptop", "100.101.3.3", "macOS", online = false, lastSeen = "2026-10-30T08:00:00Z"),
        ),
        fetchedAtMs = System.currentTimeMillis() - 20 * 60_000,
    )
}
