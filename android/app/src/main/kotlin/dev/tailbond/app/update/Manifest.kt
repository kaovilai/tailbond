package dev.tailbond.app.update

import kotlinx.serialization.SerialName
import kotlinx.serialization.Serializable
import kotlinx.serialization.json.Json

/** One file of a release (docs/architecture.md, "Updates"). */
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

/** Reads `<channel>.json`: format 1, for the channel asked for. */
object ManifestParser {
    private val json = Json { ignoreUnknownKeys = true }

    /**
     * [channel]: the channel asked for; a manifest for another channel is
     * refused (a mirror can't hand out beta as stable). Throws
     * [ManifestException] with the reason in plain words.
     */
    fun parse(bytes: ByteArray, channel: String? = null): Manifest {
        val m = try {
            json.decodeFromString(Manifest.serializer(), bytes.toString(Charsets.UTF_8))
        } catch (e: Exception) {
            throw ManifestException("the update list isn't valid JSON")
        }
        if (m.format != 1) throw ManifestException("update list format ${m.format} (this app reads format 1)")
        if (channel != null && m.channel != channel) throw ManifestException("the update list is for channel \"${m.channel}\", not \"$channel\"")
        if (m.version.isBlank()) throw ManifestException("the update list names no version")
        return m
    }
}

class ManifestException(msg: String) : java.io.IOException(msg)
