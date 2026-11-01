package dev.tailbond.app.data

import kotlinx.serialization.Serializable
import kotlinx.serialization.json.Json
import kotlinx.serialization.json.JsonPrimitive
import java.io.IOException
import java.net.InetAddress
import java.net.InetSocketAddress
import java.net.Socket

/**
 * "Ask to connect" (docs/architecture.md, "Pairing over Tailscale"): while
 * the Tailscale app still runs the VPN, the phone reaches the home server's
 * agent on its tailnet address, asks to pair (POST /api/pair-requests), and
 * polls until someone approves it on the server (GET /api/pair-requests/{id}).
 *
 * Plain HTTP, but only to Tailscale addresses (100.64.0.0/10,
 * fd7a:115c:a1e0::/48): that traffic is inside Tailscale's encrypted tunnel,
 * and refusing anything else keeps this from ever going over the open
 * internet. A raw socket, because the app's network security config only
 * allows cleartext to the tunnel address.
 */
object PairRequestClient {
    const val PORT = 8077

    @Serializable
    data class Created(
        val id: String = "",
        val token: String = "",
        val status: String = "",
        val name: String = "",
        // how the server's approval card shows this phone (additive)
        val device: String = "",
        val user: String = "",
        val ip: String = "",
    )

    @Serializable
    data class Polled(val status: String = "", val uri: String? = null, val name: String = "", val error: String? = null)

    @Serializable
    private data class Hello(val tailbond: Boolean = false, val name: String = "", val role: String = "")

    val json = Json { ignoreUnknownKeys = true }

    /** Why asking failed, in terms the UI turns into words. */
    enum class Failure {
        /** The name doesn't resolve (Tailscale off, or a typo). */
        NOT_FOUND,
        /** It resolves, but not to a Tailscale address. */
        NOT_TAILNET,
        /** Nothing answers on the agent port. */
        UNREACHABLE,
        /** Something answers, but it isn't a tailbond home server. */
        NOT_SERVER,
        /** The server refused (rate limit, or it doesn't know this device). */
        REFUSED,
        TOO_MANY,
    }

    class AskException(val failure: Failure, msg: String) : IOException(msg)

    /** One HTTP exchange; replaced in tests. Returns status code and body. */
    fun interface Transport {
        fun exchange(addr: InetAddress, port: Int, method: String, path: String, body: String?, token: String?): Pair<Int, String>
    }

    /** Resolves [host]; replaced in tests. */
    fun interface Resolver {
        fun resolve(host: String): List<InetAddress>
    }

    var transport: Transport = Transport(::socketExchange)
    var resolver: Resolver = Resolver { InetAddress.getAllByName(it).toList() }

    /** "homebox", "homebox.tail1234.ts.net", "http://homebox:8077/", "100.101.1.1" -> host. */
    fun hostOf(typed: String): String =
        typed.trim().removePrefix("http://").removePrefix("https://").substringBefore('/').let { h ->
            if (h.startsWith("[")) h.substringAfter('[').substringBefore(']') else if (h.count { it == ':' } == 1) h.substringBefore(':') else h
        }

    fun isTailnet(a: InetAddress): Boolean {
        val b = a.address
        return when (b.size) {
            4 -> (b[0].toInt() and 0xff) == 100 && (b[1].toInt() and 0xc0) == 64
            16 -> b[0] == 0xfd.toByte() && b[1] == 0x7a.toByte() && b[2] == 0x11.toByte() && b[3] == 0x5c.toByte() &&
                b[4] == 0xa1.toByte() && b[5] == 0xe0.toByte()
            else -> false
        }
    }

    /** The home server's tailnet address for what the user typed. Blocking. */
    fun locate(typed: String): InetAddress {
        val host = hostOf(typed)
        if (host.isEmpty()) throw AskException(Failure.NOT_FOUND, "empty name")
        val addrs = try { resolver.resolve(host) } catch (e: Exception) { throw AskException(Failure.NOT_FOUND, "resolve $host: ${e.message}") }
        return addrs.firstOrNull { isTailnet(it) && it.address.size == 4 } ?: addrs.firstOrNull { isTailnet(it) }
            ?: if (addrs.isEmpty()) throw AskException(Failure.NOT_FOUND, "no address for $host")
            else throw AskException(Failure.NOT_TAILNET, "$host is ${addrs.first().hostAddress}, not a Tailscale address")
    }

    /**
     * Asks the server at [addr] to pair this phone as [name]. [avoidSubnets]:
     * the tunnel subnets of the servers this phone already has, so a new
     * server without clients can move off them. Blocking.
     */
    fun ask(addr: InetAddress, name: String, avoidSubnets: List<String> = emptyList()): Created {
        val hello = try { transport.exchange(addr, PORT, "GET", "/api/hello", null, null) } catch (e: IOException) {
            throw AskException(Failure.UNREACHABLE, e.message ?: "unreachable")
        }
        val h = runCatching { json.decodeFromString(Hello.serializer(), hello.second) }.getOrNull()
        if (hello.first != 200 || h?.tailbond != true || h.role != "server") throw AskException(Failure.NOT_SERVER, "not a tailbond server")
        val avoid = if (avoidSubnets.isEmpty()) "" else ",\"avoid_subnets\":[" + avoidSubnets.joinToString(",") { jsonString(it) } + "]"
        val body = """{"name":${jsonString(name)},"kind":"phone"$avoid}"""
        val (code, text) = try { transport.exchange(addr, PORT, "POST", "/api/pair-requests", body, null) } catch (e: IOException) {
            throw AskException(Failure.UNREACHABLE, e.message ?: "unreachable")
        }
        return when (code) {
            202 -> json.decodeFromString(Created.serializer(), text).also { if (it.id.isEmpty()) throw AskException(Failure.REFUSED, "no id") }
            429 -> throw AskException(Failure.TOO_MANY, "rate limited")
            else -> throw AskException(Failure.REFUSED, "HTTP $code")
        }
    }

    /** Checks on a request. A 404 means it's gone (expired, or the server restarted). Blocking. */
    fun poll(addr: InetAddress, c: Created): Polled {
        val (code, text) = transport.exchange(addr, PORT, "GET", "/api/pair-requests/${c.id}", null, c.token)
        return when (code) {
            200 -> json.decodeFromString(Polled.serializer(), text)
            404 -> Polled(status = "expired")
            else -> throw IOException("HTTP $code")
        }
    }

    private fun jsonString(s: String) = JsonPrimitive(s).toString()

    /** Minimal HTTP/1.0 over a socket (the server closes after replying). */
    internal fun socketExchange(addr: InetAddress, port: Int, method: String, path: String, body: String?, token: String?): Pair<Int, String> {
        Socket().use { s ->
            s.connect(InetSocketAddress(addr, port), 5_000)
            s.soTimeout = 8_000
            val payload = body?.toByteArray(Charsets.UTF_8)
            val head = buildString {
                append("$method $path HTTP/1.0\r\nHost: ${addr.hostAddress}\r\nAccept: application/json\r\nConnection: close\r\n")
                if (token != null) append("Authorization: Bearer $token\r\n")
                if (payload != null) append("Content-Type: application/json\r\nContent-Length: ${payload.size}\r\n")
                append("\r\n")
            }
            s.getOutputStream().apply { write(head.toByteArray(Charsets.UTF_8)); payload?.let { write(it) }; flush() }
            val raw = s.getInputStream().readBytes().toString(Charsets.UTF_8)
            val code = raw.substringBefore("\r\n").split(' ').getOrNull(1)?.toIntOrNull() ?: throw IOException("bad HTTP reply")
            return code to raw.substringAfter("\r\n\r\n", "")
        }
    }
}

/** Where "Ask to connect" is. The UI renders it; the view model drives it. */
sealed interface AskState {
    data object Idle : AskState
    data class Asking(val server: String) : AskState
    /** [seenAs]: how the server's approval card shows this phone ("pixel-8 · sam@example.com · 100.101.8.8"). */
    data class Waiting(val server: String, val name: String, val seenAs: String = "") : AskState
    data class Declined(val server: String) : AskState
    data class Expired(val server: String) : AskState
    data class Failed(val server: String, val failure: PairRequestClient.Failure) : AskState
}

/** Next state for a poll answer; null keeps waiting. [onUri] gets an approved pairing link. */
fun AskState.Waiting.next(p: PairRequestClient.Polled, onUri: (String) -> Unit): AskState? = when (p.status) {
    "approved" -> p.uri?.let { onUri(it); AskState.Idle }
    "declined" -> AskState.Declined(server)
    "expired" -> AskState.Expired(server)
    else -> null
}

/** A request name for this phone: "Pixel 8" -> "pixel-8". */
fun phoneRequestName(model: String): String =
    model.lowercase().replace(Regex("[^a-z0-9._-]+"), "-").trim('-', '.').take(63).ifEmpty { "phone" }
