package dev.tailbond.app.update

import java.io.File

/** What a downloaded APK is, checked before handing it to Android. */
enum class ApkCheck {
    OK,
    /** Not dev.tailbond.app (or unreadable). */
    WRONG_PACKAGE,
    /**
     * Signed with another key than the installed app. Android only updates
     * an app with an APK signed by the same key; a debug build can't take a
     * release APK, or the other way round.
     */
    OTHER_SIGNER,
    /** Its versionCode isn't higher than the installed one. */
    NOT_NEWER,
}

/** How a PackageInstaller session ended (PackageInstaller.STATUS_*). */
sealed interface InstallResult {
    data object Success : InstallResult
    /** Android wants a tap (first update, Android 11 and older, or the app isn't the installer of record yet). */
    data object NeedsUserAction : InstallResult
    data class Failed(val why: String, val blockedByUser: Boolean = false) : InstallResult

    companion object {
        // android.content.pm.PackageInstaller constants (stable since API 21)
        const val STATUS_PENDING_USER_ACTION = -1
        const val STATUS_SUCCESS = 0
        const val STATUS_FAILURE = 1
        const val STATUS_FAILURE_BLOCKED = 2
        const val STATUS_FAILURE_ABORTED = 3
        const val STATUS_FAILURE_INVALID = 4
        const val STATUS_FAILURE_CONFLICT = 5
        const val STATUS_FAILURE_STORAGE = 6
        const val STATUS_FAILURE_INCOMPATIBLE = 7
        const val STATUS_FAILURE_TIMEOUT = 8

        /** Plain words for a session status; [message] is Android's own text (EXTRA_STATUS_MESSAGE). */
        fun of(status: Int, message: String?): InstallResult = when (status) {
            STATUS_SUCCESS -> Success
            STATUS_PENDING_USER_ACTION -> NeedsUserAction
            STATUS_FAILURE_ABORTED -> Failed("the install was cancelled", blockedByUser = true)
            STATUS_FAILURE_BLOCKED -> Failed("Android blocked it" + detail(message))
            STATUS_FAILURE_CONFLICT -> Failed("it conflicts with the installed app (signed with a different key?)" + detail(message))
            STATUS_FAILURE_INVALID -> Failed("Android says the file isn't a valid app" + detail(message))
            STATUS_FAILURE_STORAGE -> Failed("not enough storage space")
            STATUS_FAILURE_INCOMPATIBLE -> Failed("it doesn't fit this phone" + detail(message))
            STATUS_FAILURE_TIMEOUT -> Failed("Android didn't confirm in time")
            else -> Failed("Android said no" + detail(message))
        }

        private fun detail(m: String?) = m?.trim()?.takeIf { it.isNotEmpty() }?.let { " ($it)" } ?: ""
    }
}

/**
 * An APK's signing certificates, as PackageManager reports them
 * (`Signature.toCharsString()`). [history]: the rotation lineage, oldest
 * first and the current certificate last (API 28+, single signer); empty
 * below API 28 or with several signers.
 */
data class ApkSigners(val current: Set<String>, val history: List<String> = emptyList()) {
    /**
     * Whether [update] is signed by the same key as this (installed) app, as
     * Android requires: the same set of signers, or, after a key rotation,
     * a lineage that includes this app's current certificate. Nothing
     * readable on either side is never a match.
     */
    fun accepts(update: ApkSigners): Boolean {
        if (current.isEmpty() || update.current.isEmpty()) return false
        if (current == update.current) return true
        return current.size == 1 && update.current.size == 1 && current.single() in update.history
    }
}

/**
 * The PackageInstaller side, behind an interface: [PlatformInstaller] on a
 * phone, a fake in tests. [install] starts a session and returns; the result
 * comes back later (a broadcast) through [Updater.onInstallResult].
 */
interface ApkInstaller {
    /** "Install unknown apps" is allowed for tailbond (PackageManager.canRequestPackageInstalls). */
    fun canInstallPackages(): Boolean
    /** Android 12+ and tailbond is its own installer of record: updates need no tap. */
    fun silentUpdates(): Boolean
    /**
     * Before installing: is [apk] our package, signed with the same key as
     * the installed app ([ApkSigners.accepts]), and newer? Android enforces
     * the signer too; this gives a clear message instead of a failed install.
     */
    fun check(apk: File): ApkCheck
    /** Writes [apk] into a session and commits it. Throws when Android refuses to open the session. */
    fun install(apk: File)
}
