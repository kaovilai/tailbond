package dev.tailbond.app.update

import kotlinx.coroutines.flow.MutableStateFlow
import kotlinx.coroutines.flow.StateFlow
import kotlinx.coroutines.flow.asStateFlow
import kotlinx.serialization.Serializable
import kotlinx.serialization.Transient
import kotlinx.serialization.json.Json
import java.io.IOException
import java.net.SocketTimeoutException
import java.net.UnknownHostException

/** Everything the Updates page shows; persisted between runs. */
@Serializable
data class UpdateStatus(
    /** This build's versionName. */
    val current: String = "",
    /** The latest release's version from the last good check (null = never checked). */
    val latest: String? = null,
    /** Its release page (the changelog). */
    val notesUrl: String? = null,
    /** Its `tailbond-<version>.apk` (null = the release has no APK). */
    val apkUrl: String? = null,
    /** Last good check, ms since epoch (0 = never). */
    val lastCheckMs: Long = 0,
    /** Last try, good or not. */
    val lastAttemptMs: Long = 0,
    /** Why the last try failed, in plain words; null when it worked. */
    val error: String? = null,
    @Transient val checking: Boolean = false,
) {
    val availability: Availability get() = UpdateRules.availability(current, latest, apkUrl)

    fun toJson(): String = json.encodeToString(serializer(), this)

    companion object {
        private val json = Json { ignoreUnknownKeys = true; encodeDefaults = true }
        fun fromJson(s: String?): UpdateStatus? = s?.let { runCatching { json.decodeFromString(serializer(), it) }.getOrNull() }
    }
}

/** One HTTPS GET: status code and body. Real one in [UpdateManager]; fakes in tests. */
fun interface HttpsGet {
    fun get(url: String, headers: Map<String, String>): Pair<Int, String>
}

/** What the checker needs from the app; faked in tests. */
interface UpdateEnv {
    val currentVersion: String
    /** `owner/name` of the GitHub repository whose releases are tailbond's. */
    val repo: String
    fun policy(): UpdatePolicy
    fun now(): Long
    fun load(): UpdateStatus?
    fun save(s: UpdateStatus)
}

/**
 * Asks GitHub for the latest release, about once a day ([checkIfDue], when
 * the app comes to the front) or when asked ([check], "Check now"). Blocking;
 * [UpdateManager] runs it on a background thread.
 */
class UpdateChecker(private val env: UpdateEnv, private val http: HttpsGet) {
    private val _status = MutableStateFlow(
        (env.load() ?: UpdateStatus()).copy(current = env.currentVersion, checking = false),
    )
    val status: StateFlow<UpdateStatus> = _status.asStateFlow()

    fun checkIfDue(): Boolean {
        val s = _status.value
        if (s.checking || !UpdateRules.checkDue(env.policy(), s.lastCheckMs, s.lastAttemptMs, env.now())) return false
        check()
        return true
    }

    /** Checks now, whatever the policy. */
    fun check() {
        if (_status.value.checking) return
        val started = env.now()
        _status.value = _status.value.copy(checking = true, lastAttemptMs = started)
        val next = try {
            val r = fetch()
            _status.value.copy(
                latest = r.version, notesUrl = r.notesUrl, apkUrl = r.apkUrl,
                lastCheckMs = started, error = null,
            )
        } catch (e: Exception) {
            _status.value.copy(error = why(e))
        }.copy(checking = false)
        _status.value = next
        env.save(next)
    }

    private fun fetch(): LatestRelease {
        val (code, body) = http.get(
            UpdateRules.latestUrl(env.repo),
            mapOf(
                "Accept" to "application/vnd.github+json",
                "X-GitHub-Api-Version" to "2022-11-28",
                "User-Agent" to "tailbond-android/${env.currentVersion}",
            ),
        )
        when (code) {
            200 -> return ReleaseParser.parse(body)
            404 -> throw ReleaseException("${env.repo} has no releases on GitHub yet")
            403, 429 -> throw ReleaseException("GitHub is limiting how often it can be asked. Try again in an hour")
            else -> throw ReleaseException("GitHub answered with an error ($code)")
        }
    }

    private fun why(e: Exception): String = when (e) {
        is ReleaseException -> e.message ?: "GitHub's answer wasn't usable"
        is UnknownHostException -> "No internet connection, or GitHub can't be reached"
        is SocketTimeoutException -> "GitHub didn't answer in time"
        is IOException -> "Couldn't reach GitHub" + (e.message?.let { ": $it" } ?: "")
        else -> "Something went wrong: ${e.message ?: e.javaClass.simpleName}"
    }
}
