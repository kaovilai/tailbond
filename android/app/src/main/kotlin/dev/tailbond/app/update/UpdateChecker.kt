package dev.tailbond.app.update

import java.io.ByteArrayOutputStream
import java.io.File
import java.io.FilterOutputStream
import java.io.IOException
import java.io.OutputStream
import java.security.MessageDigest

/**
 * One HTTP GET. Writes the body to [sink] (at most [maxBytes], else it
 * throws) and returns the status code. [progress] gets the bytes so far.
 * Real ones: [dev.tailbond.app.data.TunnelHttp] (the server's mirror,
 * through the tunnel) and [UrlHttp] (the public source); fakes in tests.
 */
fun interface HttpGet {
    fun get(url: String, headers: Map<String, String>, sink: OutputStream, maxBytes: Long, progress: (Long) -> Unit): Int
}

class HttpStatusException(val code: Int, url: String) : IOException("HTTP $code from $url")

/** Where manifests and files come from, tried in order. */
data class UpdateSource(
    /** `mirror` (the home server) or `public` (`update.source`), as the contract's status says. */
    val kind: String,
    /** Base URL for `<channel>.json`: `http://10.77.0.1:8077/updates` or the public base. */
    val base: String,
    val http: HttpGet,
    val headers: Map<String, String> = emptyMap(),
    /** Display name ("homebox", "github.com"). */
    val label: String = kind,
) {
    val isMirror: Boolean get() = kind == MIRROR

    /** Where this source serves [a]: the mirror's `/updates/files/<name>`, else the artifact's own URL. */
    fun fileUrl(a: Artifact): String? = if (isMirror) "${base.trimEnd('/')}/files/${a.name}" else a.url.ifBlank { null }

    fun fetch(url: String, maxBytes: Long): ByteArray {
        val out = ByteArrayOutputStream()
        val code = http.get(url, headers, out, maxBytes) {}
        if (code != 200) throw HttpStatusException(code, url)
        return out.toByteArray()
    }

    companion object {
        const val MIRROR = "mirror"
        const val PUBLIC = "public"
    }
}

/** A manifest and where it came from, or why none could be read. */
sealed interface CheckOutcome {
    data class Found(val manifest: Manifest, val source: UpdateSource, val manifestBytes: ByteArray) : CheckOutcome
    /** Every source failed; [problems] says why, per source ("homebox: …"). */
    data class Failed(val problems: List<String>) : CheckOutcome
}

/**
 * Reads `<channel>.json` from the first source that has a good one. Trust
 * comes from the channel, not from a signature on the list: the home
 * server's mirror is reached through the tunnel (which pins the server's
 * certificate) and only mirrors files it has verified itself; the public
 * source is https only. The list gives each file's SHA-256 and size, and the
 * APK itself must be signed with this app's own key ([ApkInstaller.check]).
 * The release's `.sigstore.json` (a GitHub attestation) isn't read here.
 */
object UpdateChecker {
    const val MAX_MANIFEST = 1L shl 20

    fun check(channel: UpdateChannel, sources: List<UpdateSource>): CheckOutcome {
        if (sources.isEmpty()) return CheckOutcome.Failed(listOf("not connected to a server, and this build has no public source"))
        val problems = mutableListOf<String>()
        for (src in sources) {
            val name = channel.wire + ".json"
            try {
                if (!src.isMirror && !src.base.startsWith("https://")) throw java.io.IOException("the public source must be https")
                val m = src.fetch("${src.base.trimEnd('/')}/$name", MAX_MANIFEST)
                return CheckOutcome.Found(ManifestParser.parse(m, channel.wire), src, m)
            } catch (e: HttpStatusException) {
                problems += "${src.label}: " + if (e.code == 404) "no $name there" else "answered ${e.code}"
            } catch (e: Exception) {
                problems += "${src.label}: ${e.message ?: e.javaClass.simpleName}"
            }
        }
        return CheckOutcome.Failed(problems)
    }
}

/** Why a downloaded file was refused. */
class ArtifactException(msg: String) : IOException(msg)

/**
 * Downloads [artifact] from each source in turn (the mirror's copy first,
 * then its URL) into [dest], and keeps the first copy whose size and
 * SHA-256 match the manifest.
 */
object ArtifactDownloader {
    /** Largest APK accepted at all (the manifest's `size` is the real limit). */
    const val MAX_APK = 400L shl 20

    fun download(artifact: Artifact, sources: List<UpdateSource>, dest: File, progress: (Long, Long) -> Unit = { _, _ -> }): UpdateSource {
        if (artifact.size <= 0 || artifact.size > MAX_APK) throw ArtifactException("the manifest gives no usable size for ${artifact.name}")
        val want = artifact.sha256.trim().lowercase()
        if (!Regex("^[0-9a-f]{64}$").matches(want)) throw ArtifactException("the manifest gives no SHA-256 for ${artifact.name}")
        val problems = mutableListOf<String>()
        // the public URL is the same for every source: try it once
        val tried = mutableSetOf<String>()
        for (src in sources) {
            val url = src.fileUrl(artifact) ?: continue
            if (!tried.add(url)) continue
            try {
                val md = MessageDigest.getInstance("SHA-256")
                var n = 0L
                dest.parentFile?.mkdirs()
                val code = dest.outputStream().buffered().use { raw ->
                    val out = object : FilterOutputStream(raw) {
                        override fun write(b: Int) { md.update(b.toByte()); n++; raw.write(b) }
                        override fun write(b: ByteArray, off: Int, len: Int) { md.update(b, off, len); n += len; raw.write(b, off, len) }
                    }
                    src.http.get(url, src.headers, out, artifact.size) { progress(it, artifact.size) }
                }
                if (code != 200) throw HttpStatusException(code, url)
                if (n != artifact.size) throw ArtifactException("got $n bytes, the manifest says ${artifact.size}")
                val got = md.digest().joinToString("") { "%02x".format(it) }
                if (got != want) throw ArtifactException("the file's SHA-256 doesn't match the manifest")
                return src
            } catch (e: Exception) {
                dest.delete()
                problems += "${src.label}: ${e.message ?: e.javaClass.simpleName}"
            }
        }
        if (problems.isEmpty()) problems += "no source has ${artifact.name}"
        throw ArtifactException(problems.joinToString("; "))
    }
}

/**
 * Copies to [sink] while counting, and throws once more than [maxBytes]
 * arrive (a mirror can't fill the phone's storage). For [HttpGet] impls.
 */
fun copyLimited(input: java.io.InputStream, sink: OutputStream, maxBytes: Long, progress: (Long) -> Unit): Long {
    val buf = ByteArray(64 * 1024)
    var total = 0L
    var lastReport = 0L
    while (true) {
        val n = input.read(buf)
        if (n < 0) break
        total += n
        if (total > maxBytes) throw IOException("more than $maxBytes bytes")
        sink.write(buf, 0, n)
        if (total - lastReport >= 256 * 1024) { progress(total); lastReport = total }
    }
    progress(total)
    return total
}

/** HTTPS GET with HttpURLConnection (the public source; follows GitHub's redirects). */
object UrlHttp : HttpGet {
    override fun get(url: String, headers: Map<String, String>, sink: OutputStream, maxBytes: Long, progress: (Long) -> Unit): Int {
        val u = java.net.URL(url)
        if (u.protocol != "https") throw IOException("refusing $url: the public source must be https")
        val c = u.openConnection() as java.net.HttpURLConnection
        try {
            c.connectTimeout = 10_000
            c.readTimeout = 20_000
            c.instanceFollowRedirects = true
            headers.forEach { (k, v) -> c.setRequestProperty(k, v) }
            val code = c.responseCode
            if (code != 200) return code
            if (c.contentLengthLong > maxBytes) throw IOException("${c.contentLengthLong} bytes is more than $maxBytes")
            c.inputStream.use { copyLimited(it, sink, maxBytes, progress) }
            return code
        } finally {
            c.disconnect()
        }
    }
}
