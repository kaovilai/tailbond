package dev.tailbond.app.update

import android.app.PendingIntent
import android.content.BroadcastReceiver
import android.content.Context
import android.content.Intent
import android.content.pm.PackageInfo
import android.content.pm.PackageInstaller
import android.content.pm.PackageManager
import android.net.VpnService
import android.os.Build
import androidx.core.content.IntentCompat
import dev.tailbond.app.service.TailbondVpnService
import java.io.File

/**
 * [ApkInstaller] on a real phone: a PackageInstaller session for our own
 * package. On Android 12+ it asks for USER_ACTION_NOT_REQUIRED, which Android
 * honours once tailbond is the installer of record (after the first update
 * it installed itself); otherwise the session answers
 * STATUS_PENDING_USER_ACTION and the user taps "Update" once.
 */
class PlatformInstaller(private val ctx: Context) : ApkInstaller {
    private val pm: PackageManager get() = ctx.packageManager

    override fun canInstallPackages(): Boolean = pm.canRequestPackageInstalls()

    override fun silentUpdates(): Boolean = Build.VERSION.SDK_INT >= 31 && installerOfRecord() == ctx.packageName

    /** Which app installed us (the browser, adb's "com.android.shell", or tailbond itself after an update). */
    fun installerOfRecord(): String? = try {
        if (Build.VERSION.SDK_INT >= 30) pm.getInstallSourceInfo(ctx.packageName).installingPackageName
        else @Suppress("DEPRECATION") pm.getInstallerPackageName(ctx.packageName)
    } catch (_: Exception) {
        null
    }

    override fun check(apk: File): ApkCheck {
        val flags = if (Build.VERSION.SDK_INT >= 28) PackageManager.GET_SIGNING_CERTIFICATES else @Suppress("DEPRECATION") PackageManager.GET_SIGNATURES
        val theirs = pm.getPackageArchiveInfo(apk.path, flags) ?: return ApkCheck.WRONG_PACKAGE
        if (theirs.packageName != ctx.packageName) return ApkCheck.WRONG_PACKAGE
        val ours = try { pm.getPackageInfo(ctx.packageName, flags) } catch (_: Exception) { return ApkCheck.OK }
        if (signers(theirs) != signers(ours)) return ApkCheck.OTHER_SIGNER
        if (versionCode(theirs) <= versionCode(ours)) return ApkCheck.NOT_NEWER
        return ApkCheck.OK
    }

    private fun signers(p: PackageInfo): Set<String> {
        val sigs = if (Build.VERSION.SDK_INT >= 28) {
            p.signingInfo?.let { if (it.hasMultipleSigners()) it.apkContentsSigners else it.signingCertificateHistory }
        } else {
            @Suppress("DEPRECATION") p.signatures
        }
        return sigs.orEmpty().map { it.toCharsString() }.toSet()
    }

    private fun versionCode(p: PackageInfo): Long =
        if (Build.VERSION.SDK_INT >= 28) p.longVersionCode else @Suppress("DEPRECATION") p.versionCode.toLong()

    override fun install(apk: File) {
        val installer = pm.packageInstaller
        val params = PackageInstaller.SessionParams(PackageInstaller.SessionParams.MODE_FULL_INSTALL).apply {
            setAppPackageName(ctx.packageName)
            setSize(apk.length())
            if (Build.VERSION.SDK_INT >= 31) setRequireUserAction(PackageInstaller.SessionParams.USER_ACTION_NOT_REQUIRED)
        }
        val id = installer.createSession(params)
        installer.openSession(id).use { session ->
            try {
                session.openWrite("tailbond.apk", 0, apk.length()).use { out ->
                    apk.inputStream().use { it.copyTo(out, 64 * 1024) }
                    session.fsync(out)
                }
                val status = PendingIntent.getBroadcast(
                    ctx, id,
                    Intent(ctx, InstallStatusReceiver::class.java).setAction(ACTION_STATUS).setPackage(ctx.packageName),
                    // mutable: the installer adds EXTRA_STATUS to this explicit intent
                    PendingIntent.FLAG_UPDATE_CURRENT or (if (Build.VERSION.SDK_INT >= 31) PendingIntent.FLAG_MUTABLE else 0),
                )
                session.commit(status.intentSender)
            } catch (e: Exception) {
                session.abandon()
                throw e
            }
        }
    }

    companion object {
        const val ACTION_STATUS = "dev.tailbond.app.INSTALL_STATUS"
    }
}

/** PackageInstaller's answer for our session. */
class InstallStatusReceiver : BroadcastReceiver() {
    override fun onReceive(context: Context, intent: Intent) {
        if (intent.action != PlatformInstaller.ACTION_STATUS) return
        val status = intent.getIntExtra(PackageInstaller.EXTRA_STATUS, PackageInstaller.STATUS_FAILURE)
        val message = intent.getStringExtra(PackageInstaller.EXTRA_STATUS_MESSAGE)
        val confirm = if (status == PackageInstaller.STATUS_PENDING_USER_ACTION) {
            IntentCompat.getParcelableExtra(intent, Intent.EXTRA_INTENT, Intent::class.java)
        } else {
            null
        }
        UpdateManager.get(context).onInstallStatus(status, message, confirm)
    }
}

/**
 * After an update Android restarts nothing: the tunnel that ran before is
 * down. Reconnect it if it was up when the install started (and record the
 * result). MY_PACKAGE_REPLACED may start a foreground service.
 */
class PackageReplacedReceiver : BroadcastReceiver() {
    override fun onReceive(context: Context, intent: Intent) {
        if (intent.action != Intent.ACTION_MY_PACKAGE_REPLACED) return
        val m = UpdateManager.get(context)
        if (m.takeReconnectFlag() && VpnService.prepare(context) == null) TailbondVpnService.connect(context)
    }
}
