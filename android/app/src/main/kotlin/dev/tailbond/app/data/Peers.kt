package dev.tailbond.app.data

import kotlinx.serialization.SerialName
import kotlinx.serialization.Serializable
import kotlinx.serialization.json.Json
import java.net.HttpURLConnection
import java.net.URL
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

/** Reads the machine list from the home server over the tunnel. */
object PeersClient {
    /** The server's tunnel address (contract "Network layout") and agent port. */
    const val URL = "http://10.77.0.1:8077/api/peers"
    private const val LABEL = "tailbond-api"

    /** Bearer token: lowercase hex HMAC-SHA256(key = pairing key, message "tailbond-api"). */
    fun token(pairingKey: String): String {
        val mac = Mac.getInstance("HmacSHA256")
        mac.init(SecretKeySpec(pairingKey.toByteArray(Charsets.UTF_8), "HmacSHA256"))
        return mac.doFinal(LABEL.toByteArray(Charsets.UTF_8)).joinToString("") { "%02x".format(it) }
    }

    /** Blocking; call off the main thread. Throws on network/HTTP errors. */
    fun fetch(pairingKey: String, url: String = URL): PeerList {
        val c = URL(url).openConnection() as HttpURLConnection
        try {
            c.connectTimeout = 5_000
            c.readTimeout = 5_000
            c.setRequestProperty("Authorization", "Bearer " + token(pairingKey))
            c.setRequestProperty("Accept", "application/json")
            val code = c.responseCode
            if (code != 200) throw java.io.IOException("home server answered $code")
            val body = c.inputStream.use { it.readBytes().toString(Charsets.UTF_8) }
            return PeerList.json.decodeFromString(PeerList.serializer(), body)
                .copy(fetchedAtMs = System.currentTimeMillis())
        } finally {
            c.disconnect()
        }
    }
}
