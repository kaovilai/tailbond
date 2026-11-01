package dev.tailbond.app.service

import com.mqvpn.sdk.core.model.MqvpnError
import dev.tailbond.app.model.PathKind
import kotlinx.coroutines.flow.MutableStateFlow
import kotlinx.coroutines.flow.StateFlow
import kotlinx.coroutines.flow.asStateFlow

enum class ConnState { DISCONNECTED, CONNECTING, CONNECTED, RECONNECTING, ERROR }

/** Why the last attempt failed, so the UI can name the fix (see Words.error). */
enum class ErrorKind {
    /** No answer from the server (port forward, server off, wrong address). */
    UNREACHABLE,
    /** TLS: certificate doesn't verify or doesn't match the pin. */
    IDENTITY,
    /** The server refused this phone's key. */
    REJECTED,
    NOT_PAIRED,
    OTHER;

    companion object {
        fun of(e: MqvpnError): ErrorKind = when (e) {
            is MqvpnError.Timeout, is MqvpnError.ConnectionClosed -> UNREACHABLE
            is MqvpnError.TlsError -> IDENTITY
            is MqvpnError.AuthFailed, is MqvpnError.ProtocolError -> REJECTED
            else -> OTHER
        }
    }
}

data class PathSample(
    val name: String,
    val kind: PathKind?,
    /** mqvpn path status, see [PathGate]; -1 = held back by policy (not attached). */
    val status: Int,
    val srttMs: Long,
    val txBps: Long,
    val rxBps: Long,
    val txBytes: Long,
    val rxBytes: Long,
    /** Combined (rx+tx) bits/s, oldest first, up to [LiveState.HISTORY] samples. */
    val history: List<Long> = emptyList(),
)

data class Live(
    val state: ConnState = ConnState.DISCONNECTED,
    val message: String? = null,
    val errorKind: ErrorKind? = null,
    val tunnelIp: String? = null,
    val connectedAtMs: Long = 0,
    val txBps: Long = 0,
    val rxBps: Long = 0,
    val txBytes: Long = 0,
    val rxBytes: Long = 0,
    /** Tunnel-wide datagram loss over the last second, percent (mqvpn has no per-path loss on the client). */
    val lossPct: Double = 0.0,
    val srttMs: Int = 0,
    val rxHistory: List<Long> = emptyList(),
    val txHistory: List<Long> = emptyList(),
    val paths: List<PathSample> = emptyList(),
    val standbyActive: Boolean = false,
)

/** In-process live tunnel state, written by the VPN service, read by UI, tile and notification. */
object LiveState {
    const val HISTORY = 60

    private val _live = MutableStateFlow(Live())
    val live: StateFlow<Live> = _live.asStateFlow()

    fun set(f: (Live) -> Live) {
        while (true) {
            val cur = _live.value
            if (_live.compareAndSet(cur, f(cur))) return
        }
    }

    fun reset(state: ConnState = ConnState.DISCONNECTED, message: String? = null, errorKind: ErrorKind? = null) {
        _live.value = Live(state = state, message = message, errorKind = errorKind)
    }
}

/** Turns cumulative byte counters sampled ~1/s into bit rates; one per counter. */
class RateMeter {
    private var lastBytes = -1L
    private var lastMs = 0L

    fun sample(bytes: Long, nowMs: Long): Long {
        val prev = lastBytes
        val prevMs = lastMs
        lastBytes = bytes
        lastMs = nowMs
        if (prev < 0 || bytes < prev || nowMs <= prevMs) return 0
        return (bytes - prev) * 8_000 / (nowMs - prevMs)
    }
}

fun List<Long>.pushBounded(v: Long, max: Int = LiveState.HISTORY): List<Long> =
    if (size >= max) drop(size - max + 1) + v else this + v
