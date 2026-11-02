package dev.tailbond.app.update

import kotlinx.serialization.SerialName
import kotlinx.serialization.Serializable
import kotlinx.serialization.json.Json
import java.security.MessageDigest
import java.util.Base64

/** One file of a release (docs/architecture.md, "Manifest and signature"). */
@Serializable
data class Artifact(
    val name: String = "",
    val kind: String = "",
    val arch: String = "",
    val url: String = "",
    val sha256: String = "",
    val size: Long = 0,
    @SerialName("version_code") val versionCode: Long? = null,
)

/** `<channel>.json`, format 1. Unknown keys are ignored (additive fields). */
@Serializable
data class Manifest(
    val format: Int = 0,
    val channel: String = "",
    val version: String = "",
    val released: String = "",
    val protocol: Int = 1,
    @SerialName("requires_server_protocol") val requiresServerProtocol: Int = 1,
    val security: Boolean = false,
    @SerialName("notes_url") val notesUrl: String = "",
    val artifacts: List<Artifact> = emptyList(),
) {
    /** The Android APK of this release, if it has one. */
    val apk: Artifact? get() = artifacts.firstOrNull { it.kind == "apk" }
}

/** `<channel>.json.sig`. */
@Serializable
data class ManifestSig(@SerialName("key_id") val keyId: String = "", val sig: String = "")

/** What checking a manifest against the trusted keys gave. */
sealed interface Verified {
    data class Ok(val manifest: Manifest, val keyId: String) : Verified
    /** This build has no trusted key: updates are off. */
    data object NoKeys : Verified
    /** Signed by a key this build doesn't trust. */
    data class UnknownKey(val keyId: String) : Verified
    /** The signature doesn't match the bytes (tampered, or a truncated download). */
    data object BadSignature : Verified
    data class Malformed(val why: String) : Verified
}

object ManifestVerifier {
    private val json = Json { ignoreUnknownKeys = true }

    /** Hex of the first 8 bytes of SHA-256(raw public key). */
    fun keyId(publicKey: ByteArray): String =
        MessageDigest.getInstance("SHA-256").digest(publicKey).take(8).joinToString("") { "%02x".format(it) }

    /** `BuildConfig.UPDATE_KEYS`: comma-separated base64 raw 32-byte keys; anything else is skipped. */
    fun parseKeys(csv: String): List<ByteArray> = csv.split(',', ' ', '\n')
        .map { it.trim() }
        .filter { it.isNotEmpty() }
        .mapNotNull { runCatching { Base64.getDecoder().decode(it) }.getOrNull()?.takeIf { k -> k.size == 32 } }

    /**
     * Checks [sig] over the exact [manifest] bytes, then parses them. [channel]:
     * the channel asked for; a manifest for another channel is refused (a
     * mirror can't hand out beta as stable).
     */
    fun verify(manifest: ByteArray, sig: ByteArray, keys: List<ByteArray>, channel: String? = null): Verified {
        if (keys.isEmpty()) return Verified.NoKeys
        val s = try {
            json.decodeFromString(ManifestSig.serializer(), sig.toString(Charsets.UTF_8))
        } catch (e: Exception) {
            return Verified.Malformed("signature file isn't JSON")
        }
        val id = s.keyId.trim().lowercase()
        val key = keys.firstOrNull { keyId(it) == id } ?: return Verified.UnknownKey(id)
        val raw = runCatching { Base64.getDecoder().decode(s.sig.trim()) }.getOrNull()
            ?: return Verified.Malformed("signature isn't base64")
        if (raw.size != 64) return Verified.Malformed("signature is ${raw.size} bytes, not 64")
        if (!Ed25519.verify(key, manifest, raw)) return Verified.BadSignature
        val m = try {
            json.decodeFromString(Manifest.serializer(), manifest.toString(Charsets.UTF_8))
        } catch (e: Exception) {
            return Verified.Malformed("manifest isn't valid JSON")
        }
        if (m.format != 1) return Verified.Malformed("manifest format ${m.format} (this app reads format 1)")
        if (channel != null && m.channel != channel) return Verified.Malformed("manifest is for channel \"${m.channel}\", not \"$channel\"")
        return Verified.Ok(m, id)
    }
}
