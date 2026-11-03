package dev.tailbond.app.share

import dev.tailbond.app.data.PairRequestClient
import dev.tailbond.app.data.PeersClient
import kotlinx.serialization.json.Json
import kotlinx.serialization.json.jsonObject
import kotlinx.serialization.json.jsonPrimitive
import java.net.InetAddress

/** The home server's share key for pairings made before it existed: GET /api/shares through the tunnel. */
object ShareKey {
    /** Blocking; null when the server has none or can't be reached. */
    fun fetch(pairingKey: String, host: String): String? = try {
        val (code, body) = PairRequestClient.socketExchange(
            InetAddress.getByName(host), PairRequestClient.PORT, "GET", "/api/shares", null, PeersClient.token(pairingKey),
        )
        if (code != 200) null else Json.parseToJsonElement(body).jsonObject["key"]?.jsonPrimitive?.content?.takeIf { it.isNotBlank() }
    } catch (_: Exception) {
        null
    }
}
