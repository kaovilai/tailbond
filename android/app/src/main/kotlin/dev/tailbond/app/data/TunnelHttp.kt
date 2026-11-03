package dev.tailbond.app.data

import java.io.BufferedInputStream
import java.io.ByteArrayOutputStream
import java.io.IOException
import java.io.InputStream
import java.io.OutputStream
import java.net.InetAddress
import java.net.InetSocketAddress
import java.net.Socket
import java.net.URI

/**
 * Plain HTTP/1.0 to a server's agent through the tunnel (its tunnel address,
 * port 8077): POST /api/backups/push.
 * A raw socket like [PairRequestClient], because every server has its own
 * tunnel subnet and the network security config can't list them; the
 * traffic never leaves the encrypted tunnel. Only IP literals in the
 * private ranges (10/8, 172.16/12, 192.168/16) are accepted.
 */
object TunnelHttp {
    /** Headers every API request a paired client makes with its token carries (contract "Updates": the server lists client versions). */
    fun clientHeaders(version: String): Map<String, String> = mapOf("X-Tailbond-Version" to version, "X-Tailbond-Kind" to "android")

    /** POST [body] ([contentType]) and return status + reply text (at most 64 KiB). */
    fun post(url: String, headers: Map<String, String>, body: ByteArray, contentType: String): Pair<Int, String> {
        val out = ByteArrayOutputStream()
        val code = exchange("POST", url, headers, body, contentType, out, 64 * 1024L) {}
        return code to out.toString("UTF-8")
    }

    fun exchange(
        method: String, url: String, headers: Map<String, String>, body: ByteArray?, contentType: String?,
        sink: OutputStream, maxBytes: Long, progress: (Long) -> Unit,
    ): Int {
        val u = URI(url)
        if (u.scheme != "http") throw IOException("tunnel requests are plain http: $url")
        val host = u.host ?: throw IOException("no host in $url")
        if (!host.matches(Regex("""^\d{1,3}\.\d{1,3}\.\d{1,3}\.\d{1,3}$"""))) throw IOException("$host isn't a server tunnel address")
        val addr = InetAddress.getByName(host)
        if (!addr.isSiteLocalAddress) throw IOException("$host isn't a private tunnel address")
        val port = if (u.port > 0) u.port else 80
        val path = (u.rawPath ?: "/").ifEmpty { "/" } + (u.rawQuery?.let { "?$it" } ?: "")
        Socket().use { s ->
            s.connect(InetSocketAddress(addr, port), 5_000)
            s.soTimeout = 20_000
            val head = buildString {
                append("$method $path HTTP/1.0\r\nHost: $host:$port\r\nConnection: close\r\n")
                headers.forEach { (k, v) -> append("$k: ${v.replace("\r", "").replace("\n", "")}\r\n") }
                if (body != null) append("Content-Type: ${contentType ?: "application/octet-stream"}\r\nContent-Length: ${body.size}\r\n")
                append("\r\n")
            }
            s.getOutputStream().apply { write(head.toByteArray(Charsets.UTF_8)); body?.let { write(it) }; flush() }
            val input = BufferedInputStream(s.getInputStream())
            val (code, length) = readHead(input)
            if (code != 200) return code
            if (length != null && length > maxBytes) throw IOException("$length bytes is more than $maxBytes")
            copyLimited(if (length != null) LimitedInput(input, length) else input, sink, maxBytes, progress)
            return code
        }
    }

    /** Status line + headers; returns the code and Content-Length (null when absent). */
    internal fun readHead(input: InputStream): Pair<Int, Long?> {
        val lines = mutableListOf<String>()
        val line = StringBuilder()
        var total = 0
        while (true) {
            val b = input.read()
            if (b < 0) throw IOException("connection closed before the reply headers ended")
            if (++total > 32 * 1024) throw IOException("reply headers too long")
            if (b == '\n'.code) {
                val l = line.toString().trimEnd('\r')
                line.setLength(0)
                if (l.isEmpty()) break
                lines += l
            } else {
                line.append(b.toChar())
            }
        }
        val code = lines.firstOrNull()?.split(' ')?.getOrNull(1)?.toIntOrNull() ?: throw IOException("bad HTTP reply")
        val length = lines.drop(1).firstOrNull { it.substringBefore(':').trim().equals("Content-Length", true) }
            ?.substringAfter(':')?.trim()?.toLongOrNull()
        return code to length
    }

    private class LimitedInput(private val inner: InputStream, private var left: Long) : InputStream() {
        override fun read(): Int = if (left <= 0) -1 else inner.read().also { if (it >= 0) left-- }
        override fun read(b: ByteArray, off: Int, len: Int): Int {
            if (left <= 0) return -1
            val n = inner.read(b, off, minOf(len.toLong(), left).toInt())
            if (n > 0) left -= n
            return n
        }
    }
}

/** Copies to [sink] while counting, and throws once more than [maxBytes] arrive. */
internal fun copyLimited(input: InputStream, sink: OutputStream, maxBytes: Long, progress: (Long) -> Unit): Long {
    val buf = ByteArray(64 * 1024)
    var total = 0L
    while (true) {
        val n = input.read(buf)
        if (n < 0) break
        total += n
        if (total > maxBytes) throw IOException("more than $maxBytes bytes")
        sink.write(buf, 0, n)
        progress(total)
    }
    return total
}
