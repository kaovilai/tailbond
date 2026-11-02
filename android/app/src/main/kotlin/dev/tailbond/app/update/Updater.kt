package dev.tailbond.app.update

import kotlinx.coroutines.flow.MutableStateFlow
import kotlinx.coroutines.flow.StateFlow
import kotlinx.coroutines.flow.asStateFlow
import kotlinx.serialization.Serializable
import kotlinx.serialization.json.Json
import java.io.File

/** Where the updater is (contract `update.state`, plus the two Android-only waits). */
@Serializable
enum class Phase {
    IDLE, CHECKING, DOWNLOADING,
    /** Waiting: the tunnel is busy, or the server must update first. */
    WAITING,
    INSTALLING,
    /** Android asks for one tap to finish (first update, or Android 11 and older). */
    NEEDS_TAP,
    /** "Install unknown apps" isn't allowed for tailbond yet. */
    NEEDS_PERMISSION,
    FAILED,
    /** Policy Off, or no update key in this build. */
    OFF,
}

/** Everything the Updates page shows; also persisted (so it survives the restart an update causes). */
@Serializable
data class UpdateStatus(
    val current: String = "",
    val phase: Phase = Phase.IDLE,
    /** A newer version from the last check, if any. */
    val available: String? = null,
    val security: Boolean = false,
    val notesUrl: String? = null,
    val deferred: Deferral? = null,
    /** The server protocol the available version needs (it waits for the server). */
    val needsServer: Int? = null,
    /** `mirror` or `public`, and its name ("homebox", "github.com"). */
    val source: String? = null,
    val sourceLabel: String? = null,
    /** Last successful check (ms since epoch; 0 = never). */
    val lastCheckMs: Long = 0,
    val lastAttemptMs: Long = 0,
    /** What happened last, in plain words. */
    val lastResult: String = "",
    val lastResultOk: Boolean = true,
    /** Download progress 0..1 while [Phase.DOWNLOADING]. */
    val progress: Float? = null,
    /** The version handed to Android; checked after the restart. */
    val installing: String? = null,
    val noKeys: Boolean = false,
    /** Android 12+ and the app installed itself before: no tap needed. */
    val silent: Boolean = false,
    /** The last verified manifest (kept so Install works without checking again). */
    val manifest: Manifest? = null,
) {
    fun toJson(): String = json.encodeToString(serializer(), this)

    companion object {
        private val json = Json { ignoreUnknownKeys = true; encodeDefaults = true }
        fun fromJson(s: String?): UpdateStatus? = s?.let { runCatching { json.decodeFromString(serializer(), it) }.getOrNull() }
    }
}

/** What the updater needs from the app; faked in tests. */
interface UpdateEnv {
    val currentVersion: String
    val currentCode: Long
    val keys: List<ByteArray>
    fun policy(): UpdatePolicy
    fun channel(): UpdateChannel
    /** Manifest sources in order: the active server's mirror (while connected), then the public source. */
    fun sources(): List<UpdateSource>
    /** Where to get files: the mirror, then the artifact's own URL. */
    fun downloadSources(): List<UpdateSource>
    /** The active server's `protocol` (GET /api/hello; missing = 1); null = couldn't ask. */
    fun serverProtocol(): Int?
    /** Tunnel traffic (tx+rx) over the last minute, kbit/s. */
    fun busyKbps(): Long
    fun now(): Long
    fun apkFile(name: String): File
    fun load(): UpdateStatus?
    fun save(s: UpdateStatus)
    /** Just before Android replaces the app (which ends this process): remember to reconnect. */
    fun beforeInstall() {}
    /** Something the user should hear about outside the app (needs a tap, available, failed). */
    fun announce(s: UpdateStatus) {}
}

/**
 * The update state machine: check → decide → download → verify → install.
 * Blocking; the Android side ([UpdateManager]) runs it on a background
 * thread. Pure apart from [UpdateEnv] and [ApkInstaller].
 */
class Updater(private val env: UpdateEnv, private val installer: ApkInstaller) {
    private val _status = MutableStateFlow(start())
    val status: StateFlow<UpdateStatus> = _status.asStateFlow()

    private fun set(f: (UpdateStatus) -> UpdateStatus): UpdateStatus {
        val next = f(_status.value)
        _status.value = next
        if (next.phase != Phase.DOWNLOADING || next.progress == null) env.save(next)
        return next
    }

    /** After a (re)start: did the update we handed to Android land? */
    private fun start(): UpdateStatus {
        val saved = (env.load() ?: UpdateStatus()).copy(current = env.currentVersion, progress = null, silent = installer.silentUpdates())
        val own = SemVer.own(env.currentVersion)
        val target = saved.installing?.let { SemVer.parse(it) }
        val s = when {
            env.keys.isEmpty() -> saved.copy(phase = Phase.OFF, noKeys = true, available = null, installing = null)
            target != null && own != null && own >= target -> saved.copy(
                phase = Phase.IDLE, available = null, installing = null, deferred = null, needsServer = null,
                lastResult = "Updated to ${env.currentVersion}", lastResultOk = true,
            )
            saved.installing != null || saved.phase in setOf(Phase.CHECKING, Phase.DOWNLOADING, Phase.INSTALLING, Phase.NEEDS_TAP) -> saved.copy(
                phase = Phase.IDLE, installing = null,
                lastResult = saved.installing?.let { "The update to $it didn't finish. Tap Install to try again" } ?: saved.lastResult,
                lastResultOk = saved.installing == null && saved.lastResultOk,
            )
            env.policy() == UpdatePolicy.OFF -> saved.copy(phase = Phase.OFF)
            saved.phase == Phase.OFF -> saved.copy(phase = Phase.IDLE)
            else -> saved
        }.let { if (it.available != null && own != null && SemVer.parse(it.available)?.let { a -> a <= own } == true) it.copy(available = null, manifest = null) else it }
        env.save(s)
        return s
    }

    /** Policy or channel changed. */
    fun settingsChanged() {
        set {
            when {
                env.keys.isEmpty() -> it
                env.policy() == UpdatePolicy.OFF -> it.copy(phase = Phase.OFF, deferred = null)
                it.phase == Phase.OFF -> it.copy(phase = Phase.IDLE)
                else -> it
            }
        }
        val m = _status.value.manifest
        if (m != null && m.channel != env.channel().wire) set { it.copy(manifest = null, available = null, deferred = null, needsServer = null) }
    }

    /**
     * Whether a scheduled check should run now: [everyMs] after the last good
     * check (6 h; the app opening uses 30 min), and an hour after a failed one.
     */
    fun checkDue(everyMs: Long = UpdateRules.CHECK_EVERY_MS): Boolean {
        val s = _status.value
        if (env.keys.isEmpty() || env.policy() == UpdatePolicy.OFF) return false
        if (s.phase in BUSY) return false
        return UpdateRules.checkDue(s.lastCheckMs, env.now(), everyMs) &&
            UpdateRules.checkDue(s.lastAttemptMs, env.now(), minOf(everyMs, 60 * 60_000L))
    }

    /** Reads the manifest and acts on it. [manual]: "Check now" (works even with policy Off). */
    fun check(manual: Boolean) {
        if (env.keys.isEmpty()) {
            set { it.copy(phase = Phase.OFF, noKeys = true, lastResult = NO_KEY, lastResultOk = false) }
            return
        }
        if (_status.value.phase in BUSY) return
        if (!manual && env.policy() == UpdatePolicy.OFF) { set { it.copy(phase = Phase.OFF) }; return }
        val before = set { it.copy(phase = Phase.CHECKING, lastAttemptMs = env.now()) }
        when (val o = UpdateChecker(env.keys).check(env.channel(), env.sources())) {
            is CheckOutcome.Failed -> set {
                it.copy(
                    phase = idlePhase(before), noKeys = o.noKeys,
                    lastResult = (if (o.untrusted) "Refused an update that isn't signed by tailbond's key: " else "Couldn't check for updates: ") +
                        o.problems.joinToString("; "),
                    lastResultOk = false,
                )
            }
            is CheckOutcome.Found -> {
                set {
                    it.copy(
                        manifest = o.manifest, lastCheckMs = env.now(), source = o.source.kind, sourceLabel = o.source.label,
                        notesUrl = o.manifest.notesUrl.ifBlank { null }, security = o.manifest.security,
                    )
                }
                evaluate(checked = true)
            }
        }
    }

    /**
     * Applies the rules to the last manifest: up to date, wait, or install.
     * Also re-run every minute while an automatic install waits for a quiet tunnel.
     */
    fun evaluate(checked: Boolean = false) {
        val m = _status.value.manifest ?: return
        val d = UpdateRules.decide(env.currentVersion, env.currentCode, m, env.serverProtocol(), env.policy(), env.busyKbps())
        when (d) {
            Decision.UpToDate, Decision.NoApk -> set {
                it.copy(
                    phase = idlePhase(it), available = null, deferred = null, needsServer = null,
                    lastResult = if (checked) "Up to date" else it.lastResult, lastResultOk = if (checked) true else it.lastResultOk,
                )
            }
            is Decision.NeedsServer -> set {
                it.copy(
                    phase = Phase.WAITING, available = d.version, deferred = null, needsServer = d.needs,
                    lastResult = if (checked) "Version ${d.version} waits until your server is updated" else it.lastResult, lastResultOk = true,
                )
            }
            is Decision.Available -> {
                val s = set {
                    it.copy(
                        phase = if (d.deferred == Deferral.BUSY) Phase.WAITING else idlePhase(it),
                        available = d.version, deferred = d.deferred, needsServer = null,
                        lastResult = if (checked) "Version ${d.version} is available" else it.lastResult, lastResultOk = true,
                    )
                }
                if (d.install) install(manual = false)
                else if (checked && d.deferred == Deferral.ASK) env.announce(s)
            }
        }
    }

    /**
     * Downloads, verifies and hands the APK to Android. [manual] ("Install
     * now") skips the traffic deferral, never the server compatibility.
     */
    fun install(manual: Boolean) {
        if (_status.value.phase in setOf(Phase.DOWNLOADING, Phase.INSTALLING, Phase.CHECKING)) return
        if (_status.value.manifest == null && manual) check(manual = true)
        val m = _status.value.manifest ?: return
        val apk = m.apk ?: return
        val d = UpdateRules.decide(env.currentVersion, env.currentCode, m, env.serverProtocol(), UpdatePolicy.AUTOMATIC, 0)
        if (d !is Decision.Available) { evaluate(); return }
        if (!installer.canInstallPackages()) {
            val s = set { it.copy(phase = Phase.NEEDS_PERMISSION, lastResult = "Allow tailbond to install updates first", lastResultOk = false) }
            if (!manual) env.announce(s)
            return
        }
        set { it.copy(phase = Phase.DOWNLOADING, progress = 0f) }
        val file = env.apkFile(apk.name)
        try {
            ArtifactDownloader.download(apk, env.downloadSources(), file) { got, total ->
                _status.value = _status.value.copy(progress = if (total > 0) (got.toFloat() / total).coerceIn(0f, 1f) else null)
            }
        } catch (e: Exception) {
            fail(m.version, "couldn't download it: ${e.message}", manual)
            return
        }
        when (installer.check(file)) {
            ApkCheck.OK -> {}
            ApkCheck.WRONG_PACKAGE -> { file.delete(); fail(m.version, "the file isn't the tailbond app", manual); return }
            ApkCheck.OTHER_SIGNER -> {
                file.delete()
                fail(m.version, "it's signed with a different key than this app. Debug and release builds can't update each other: install the release APK once by hand", manual)
                return
            }
            ApkCheck.NOT_NEWER -> { file.delete(); set { it.copy(phase = Phase.IDLE, available = null, progress = null, lastResult = "Up to date", lastResultOk = true) }; return }
        }
        set { it.copy(phase = Phase.INSTALLING, progress = null, installing = m.version) }
        env.beforeInstall()
        try {
            installer.install(file)
        } catch (e: Exception) {
            fail(m.version, e.message ?: "Android refused to start the install", manual)
        }
    }

    /** The PackageInstaller session's answer (from the status broadcast). */
    fun onInstallResult(r: InstallResult) {
        val v = _status.value.installing ?: _status.value.available ?: "the new version"
        when (r) {
            InstallResult.Success -> set { it.copy(phase = Phase.IDLE, available = null, installing = null, lastResult = "Updated to $v", lastResultOk = true) }
            InstallResult.NeedsUserAction -> env.announce(set { it.copy(phase = Phase.NEEDS_TAP) })
            is InstallResult.Failed -> if (r.blockedByUser) {
                set { it.copy(phase = Phase.IDLE, installing = null, lastResult = "You cancelled the update to $v. Tap Install to try again", lastResultOk = false) }
            } else {
                fail(v, r.why, manual = false)
            }
        }
    }

    /** The tap Android needed can't happen any more (its prompt went away). */
    fun tapLost() {
        if (_status.value.phase == Phase.NEEDS_TAP) set { it.copy(phase = Phase.IDLE, installing = null) }
    }

    private fun fail(version: String, why: String, manual: Boolean) {
        val s = set { it.copy(phase = Phase.FAILED, progress = null, installing = null, lastResult = "Couldn't install $version: $why", lastResultOk = false) }
        if (!manual) env.announce(s)
    }

    private fun idlePhase(s: UpdateStatus): Phase = when {
        env.keys.isEmpty() || env.policy() == UpdatePolicy.OFF -> Phase.OFF
        s.phase == Phase.FAILED -> Phase.FAILED
        else -> Phase.IDLE
    }

    companion object {
        const val NO_KEY = "This build has no update key, so it can't update itself. Release builds can"
        private val BUSY = setOf(Phase.CHECKING, Phase.DOWNLOADING, Phase.INSTALLING)
    }
}
