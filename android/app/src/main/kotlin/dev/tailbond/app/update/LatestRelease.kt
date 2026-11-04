package dev.tailbond.app.update

import kotlinx.serialization.json.Json
import kotlinx.serialization.json.JsonObject
import kotlinx.serialization.json.JsonPrimitive
import kotlinx.serialization.json.booleanOrNull
import kotlinx.serialization.json.contentOrNull
import kotlinx.serialization.json.jsonArray
import kotlinx.serialization.json.jsonObject

/**
 * The answer of `GET https://api.github.com/repos/<repo>/releases/latest`
 * (docs/architecture.md, "Updates"), reduced to what the app uses.
 */
data class LatestRelease(
    /** `tag_name` without its leading `v` ("0.2.0"). */
    val version: String,
    /** `html_url`: the release page, its changelog. Null when it isn't https. */
    val notesUrl: String?,
    /** The release APK's `browser_download_url` ([ReleaseParser.apkAsset]); null when the release has no APK (yet). */
    val apkUrl: String?,
)

/** The answer isn't a release this app can use; [message] is plain words for the Updates page. */
class ReleaseException(message: String) : Exception(message)

/** Pure parsing of the latest-release JSON; unit tested. */
object ReleaseParser {
    private val json = Json { ignoreUnknownKeys = true }

    fun parse(body: String): LatestRelease {
        val o = try {
            json.parseToJsonElement(body).jsonObject
        } catch (_: Exception) {
            throw ReleaseException("GitHub's answer wasn't a release")
        }
        // the "latest" endpoint never returns drafts or pre-releases; refuse one anyway
        if (o.bool("draft") == true) throw ReleaseException("GitHub answered with a draft release")
        if (o.bool("prerelease") == true) throw ReleaseException("GitHub answered with a pre-release")
        val tag = o.str("tag_name")?.trim().orEmpty()
        if (tag.isEmpty()) throw ReleaseException("GitHub's answer has no version")
        val v = SemVer.parse(tag) ?: throw ReleaseException("the latest release's name \"${tag.take(40)}\" isn't a version number")
        if (v.pre.isNotEmpty()) throw ReleaseException("the latest release ($tag) is a pre-release")
        val version = tag.removePrefix("v").removePrefix("V")
        val assets = runCatching { o["assets"]?.jsonArray }.getOrNull().orEmpty()
            .mapNotNull { runCatching { it.jsonObject }.getOrNull() }
            .mapNotNull { a -> a.str("name")?.let { it to a.str("browser_download_url") } }
        return LatestRelease(
            version = version,
            notesUrl = o.str("html_url")?.takeIf { it.startsWith("https://") },
            apkUrl = apkAsset(assets, v),
        )
    }

    /**
     * The release APK among [assets] (name, download URL): [STABLE_APK], the
     * name every release uses (its version is the release's tag), else
     * `tailbond-<version>.apk` for this release's version. A `-debug` APK
     * (or any other name) never counts.
     */
    fun apkAsset(assets: List<Pair<String, String?>>, version: SemVer): String? {
        val https = assets.filter { it.second?.startsWith("https://") == true }
        https.firstOrNull { it.first == STABLE_APK }?.let { return it.second }
        return https.firstOrNull { (name, _) ->
            val m = APK.matchEntire(name) ?: return@firstOrNull false
            val named = SemVer.parse(m.groupValues[1])
            named != null && named.pre.isEmpty() && named.compareTo(version) == 0
        }?.second
    }

    /** The APK's release asset name (releases/latest/download/tailbond-android.apk, docs/android.md). */
    const val STABLE_APK = "tailbond-android.apk"

    private val APK = Regex("""^tailbond-v?(\d+\.\d+\.\d+(?:-[0-9A-Za-z.-]+)?)\.apk$""")

    private fun JsonObject.str(k: String): String? = (this[k] as? JsonPrimitive)?.takeIf { it.isString }?.contentOrNull
    private fun JsonObject.bool(k: String): Boolean? = (this[k] as? JsonPrimitive)?.booleanOrNull
}
