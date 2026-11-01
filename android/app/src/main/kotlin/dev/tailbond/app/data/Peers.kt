package dev.tailbond.app.data

import kotlinx.serialization.SerialName
import kotlinx.serialization.Serializable
import kotlinx.serialization.json.Json
import java.net.InetAddress
import javax.crypto.Mac
import javax.crypto.spec.SecretKeySpec

/** One tailnet machine, as the home server's GET /api/peers reports it (docs/architecture.md). */
@Serializable
data class PeerNode(
    val hostname: String = "",
    @SerialName("dns_name") val dnsName: String = "",
    @SerialName("tailscale_ips") val tailscaleIps: List<String> = emptyList(),
    val os: String = "",
    val online: Boolean = false,
    @SerialName("last_seen") val lastSeen: String? = null,
) {
    val ipv4: String? get() = tailscaleIps.firstOrNull { '.' in it }
    val ipv6: String? get() = tailscaleIps.firstOrNull { ':' in it }
}

@Serializable
data class PeerList(
    val server: String = "",
    val tailnet: String? = null,
    @SerialName("magic_dns_suffix") val magicDnsSuffix: String? = null,
    @SerialName("tailscale_running") val tailscaleRunning: Boolean = false,
    val self: PeerNode? = null,
    val peers: List<PeerNode> = emptyList(),
    /** When this list was fetched (ms since epoch); set by the app, not the server. */
    val fetchedAtMs: Long = 0,
) {
    fun toJson(): String = json.encodeToString(serializer(), this)

    companion object {
        val json = Json { ignoreUnknownKeys = true; encodeDefaults = true }
        fun fromJson(s: String): PeerList? = try { json.decodeFromString(serializer(), s) } catch (_: Exception) { null }
    }
}

/** Reads the machine list from the server in use, over the tunnel. */
object PeersClient {
    /** The default server's tunnel address (contract "Network layout"); each server has its own. */
    const val DEFAULT_HOST = "10.77.0.1"
    private const val LABEL = "tailbond-api"

    /** Bearer token: lowercase hex HMAC-SHA256(key = pairing key, message "tailbond-api"). */
    fun token(pairingKey: String): String {
        val mac = Mac.getInstance("HmacSHA256")
        mac.init(SecretKeySpec(pairingKey.toByteArray(Charsets.UTF_8), "HmacSHA256"))
        return mac.doFinal(LABEL.toByteArray(Charsets.UTF_8)).joinToString("") { "%02x".format(it) }
    }

    /**
     * Blocking; call off the main thread. Throws on network/HTTP errors.
     * [host] is the server's tunnel address (an IP literal); plain HTTP over
     * a raw socket, since it only ever goes through the tunnel and each
     * server has its own subnet (the network security config can't list them).
     */
    fun fetch(pairingKey: String, host: String = DEFAULT_HOST): PeerList {
        val addr = InetAddress.getByName(host)
        val (code, body) = PairRequestClient.socketExchange(addr, PairRequestClient.PORT, "GET", "/api/peers", null, token(pairingKey))
        if (code != 200) throw java.io.IOException("server answered $code")
        return PeerList.json.decodeFromString(PeerList.serializer(), body)
            .copy(fetchedAtMs = System.currentTimeMillis())
    }
}
