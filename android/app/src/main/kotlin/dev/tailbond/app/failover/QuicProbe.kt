package dev.tailbond.app.failover

import kotlin.random.Random

/**
 * The standby-server probe (docs/architecture.md, "Health"): one 60-byte
 * QUIC short-header datagram with a random connection ID. Any QUIC server
 * answers it with a Stateless Reset (RFC 9000 §10.3), without creating state.
 */
object QuicProbe {
    const val SIZE = 60
    const val TIMEOUT_MS = 2_000

    /** First byte 0b01xxxxxx (short header, fixed bit set, random low bits), then random bytes. */
    fun packet(rnd: Random = Random.Default): ByteArray {
        val b = rnd.nextBytes(SIZE)
        b[0] = (0x40 or (b[0].toInt() and 0x3f)).toByte()
        return b
    }
}
