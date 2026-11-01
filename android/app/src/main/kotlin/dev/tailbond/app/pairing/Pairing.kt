package dev.tailbond.app.pairing

import dev.tailbond.app.model.Cidr
import dev.tailbond.app.model.Mode
import kotlinx.serialization.SerialName
import kotlinx.serialization.Serializable
import kotlinx.serialization.json.Json
import java.net.URI
import java.util.Base64

/**
 * The pairing payload a tailbond server issues (docs/architecture.md, "Pairing payload"),
 * shared as `tailbond://pair?d=<base64url(JSON)>`.
 */
@Serializable
data class PairingPayload(
    val v: Int = 1,
    val name: String = "",
    val server: String,
    val sni: String? = null,
    val user: String = "",
    val key: String,
    val insecure: Boolean = false,
    @SerialName("cert_sha256") val certSha256: String? = null,
    val dns: List<String> = emptyList(),
    val mode: String = "speed",
    @SerialName("tailnet_routes") val tailnetRoutes: List<String> = emptyList(),
) {
    /** Host part of [server] (brackets stripped for IPv6 literals). */
    val host: String get() = splitHostPort(server).first

    /** Port part of [server], 443 when absent. */
    val port: Int get() = splitHostPort(server).second

    fun toJson(): String = json.encodeToString(serializer(), this)

    fun toUri(): String =
        "tailbond://pair?d=" + Base64.getUrlEncoder().withoutPadding()
            .encodeToString(toJson().toByteArray(Charsets.UTF_8))

    companion object {
        const val SUPPORTED_VERSION = 1
        private val json = Json { ignoreUnknownKeys = true; encodeDefaults = true }

        fun fromJson(text: String): PairingPayload = json.decodeFromString(serializer(), text)

        /** `host:port`, `[v6]:port`, bare host or bare v6 literal → (host, port). */
        fun splitHostPort(server: String): Pair<String, Int> {
            val s = server.trim()
            if (s.startsWith("[")) {
                val end = s.indexOf(']')
                require(end > 1) { "bad IPv6 literal in server: $server" }
                val host = s.substring(1, end)
                val rest = s.substring(end + 1)
                val port = if (rest.startsWith(":")) parsePort(rest.substring(1)) else 443
                return host to port
            }
            val colons = s.count { it == ':' }
            return when (colons) {
                0 -> s to 443
                1 -> s.substringBefore(':') to parsePort(s.substringAfter(':'))
                else -> s to 443 // bare IPv6 literal
            }
        }

        private fun parsePort(p: String): Int {
            val n = p.toIntOrNull()
            require(n != null && n in 1..65535) { "bad port: $p" }
            return n
        }
    }
}

sealed interface PairingResult {
    data class Ok(val payload: PairingPayload) : PairingResult
    data class Invalid(val reason: String) : PairingResult
}

object Pairing {
    const val SCHEME = "tailbond"
    const val HOST = "pair"
    const val ASK_HOST = "ask"

    private val HOSTNAME = Regex("^[A-Za-z0-9]([A-Za-z0-9.:\\[\\]-]{0,252})$")

    /**
     * The server name in `tailbond://ask?server=<name>`, the link on the home
     * server's phone page ("Open tailbond and ask homebox"); null for any
     * other URI or a name that isn't a plain host name or address.
     */
    fun askServer(text: String): String? = try {
        val uri = URI(text.trim())
        if (!uri.scheme.equals(SCHEME, ignoreCase = true) || !uri.host.equals(ASK_HOST, ignoreCase = true)) null
        else uri.rawQuery.orEmpty().split('&').firstOrNull { it.startsWith("server=") }
            ?.let { java.net.URLDecoder.decode(it.removePrefix("server="), "UTF-8").trim() }
            ?.takeIf { HOSTNAME.matches(it) }
    } catch (_: Exception) {
        null
    }

    private val HEX64 = Regex("^[0-9a-fA-F]{64}$")

    /**
     * Parses a pairing URI (or, leniently, the bare base64url blob or raw JSON a
     * user may paste) and validates it against the contract.
     */
    fun parse(input: String): PairingResult = try {
        val text = input.trim()
        val payloadJson = when {
            text.startsWith("{") -> text
            text.startsWith("$SCHEME:", ignoreCase = true) -> decodeBlob(blobFromUri(text))
            else -> decodeBlob(text)
        }
        validate(PairingPayload.fromJson(payloadJson))
    } catch (e: Exception) {
        PairingResult.Invalid(e.message ?: e.javaClass.simpleName)
    }

    private fun blobFromUri(text: String): String {
        val uri = URI(text)
        require(uri.scheme.equals(SCHEME, ignoreCase = true)) { "not a tailbond:// URI" }
        // tailbond://pair?d=… (host form) and tailbond:pair?d=… (opaque form)
        val ssp = uri.rawSchemeSpecificPart.removePrefix("//")
        require(ssp.substringBefore('?').trimEnd('/').equals(HOST, ignoreCase = true)) {
            "not a pairing URI (expected tailbond://pair)"
        }
        val query = ssp.substringAfter('?', "")
        val d = query.split('&').firstOrNull { it.startsWith("d=") }?.substring(2)
        require(!d.isNullOrEmpty()) { "pairing URI has no d= parameter" }
        return java.net.URLDecoder.decode(d, "UTF-8")
    }

    private fun decodeBlob(blob: String): String {
        // base64url per contract; accept standard alphabet and padding too
        val normalized = blob.trim().replace('+', '-').replace('/', '_').trimEnd('=')
        val bytes = try {
            Base64.getUrlDecoder().decode(normalized)
        } catch (e: IllegalArgumentException) {
            throw IllegalArgumentException("pairing data is not base64url")
        }
        return String(bytes, Charsets.UTF_8)
    }

    fun validate(p: PairingPayload): PairingResult {
        if (p.v != PairingPayload.SUPPORTED_VERSION) {
            return PairingResult.Invalid("unsupported pairing version ${p.v}")
        }
        if (p.server.isBlank()) return PairingResult.Invalid("server is empty")
        if (p.key.isBlank()) return PairingResult.Invalid("key is empty")
        val host = try {
            p.host
        } catch (e: IllegalArgumentException) {
            return PairingResult.Invalid(e.message ?: "bad server")
        }
        if (host.isBlank() || host.any { it.code !in 0x21..0x7E }) {
            return PairingResult.Invalid("server host is not a bare ASCII name or address")
        }
        p.sni?.let { sni ->
            if (sni.isBlank() || sni.any { it.code !in 0x21..0x7E }) {
                return PairingResult.Invalid("sni is not a bare ASCII name")
            }
        }
        if (Mode.fromWire(p.mode) == null) return PairingResult.Invalid("unknown mode ${p.mode}")
        val pin = p.certSha256?.replace(":", "")
        if (!pin.isNullOrEmpty() && !HEX64.matches(pin)) {
            return PairingResult.Invalid("cert_sha256 must be 64 hex characters")
        }
        for (r in p.tailnetRoutes) {
            if (Cidr.parse(r) == null) return PairingResult.Invalid("bad tailnet route $r")
        }
        for (d in p.dns) {
            if (!Cidr.isIpLiteral(d)) return PairingResult.Invalid("bad DNS server $d")
        }
        return PairingResult.Ok(p.copy(certSha256 = pin?.lowercase()?.ifEmpty { null }))
    }
}
