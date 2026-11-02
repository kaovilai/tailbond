package dev.tailbond.app

import dev.tailbond.app.update.ApkCheck
import dev.tailbond.app.update.ApkInstaller
import dev.tailbond.app.update.HttpGet
import dev.tailbond.app.update.InstallResult
import dev.tailbond.app.update.ManifestVerifier
import dev.tailbond.app.update.Phase
import dev.tailbond.app.update.UpdateChannel
import dev.tailbond.app.update.UpdateEnv
import dev.tailbond.app.update.UpdatePolicy
import dev.tailbond.app.update.UpdateSource
import dev.tailbond.app.update.UpdateStatus
import dev.tailbond.app.update.UpdateWords
import dev.tailbond.app.update.Updater
import org.junit.Assert.assertEquals
import org.junit.Assert.assertFalse
import org.junit.Assert.assertNull
import org.junit.Assert.assertTrue
import org.junit.Rule
import org.junit.Test
import org.junit.rules.TemporaryFolder
import java.io.File
import java.io.IOException
import java.io.OutputStream
import java.security.MessageDigest

/** A web server in a map: url → body; missing = 404; [down] hosts throw. */
class FakeWeb : HttpGet {
    val files = mutableMapOf<String, ByteArray>()
    val down = mutableSetOf<String>()
    val asked = mutableListOf<Pair<String, Map<String, String>>>()

    override fun get(url: String, headers: Map<String, String>, sink: OutputStream, maxBytes: Long, progress: (Long) -> Unit): Int {
        asked += url to headers
        if (down.any { url.startsWith(it) }) throw IOException("connect timed out")
        val b = files[url] ?: return 404
        if (b.size > maxBytes) throw IOException("too big")
        sink.write(b)
        progress(b.size.toLong())
        return 200
    }
}

class FakeInstaller : ApkInstaller {
    var allowed = true
    var silent = false
    var check = ApkCheck.OK
    var refuse: String? = null
    val installed = mutableListOf<ByteArray>()
    override fun canInstallPackages() = allowed
    override fun silentUpdates() = silent
    override fun check(apk: File) = check
    override fun install(apk: File) {
        refuse?.let { throw IOException(it) }
        installed += apk.readBytes()
    }
}

class UpdaterTest {
    @get:Rule val tmp = TemporaryFolder()

    private val kp = TestKeys.pair()
    private val apk = ByteArray(5000) { (it * 7).toByte() }
    private val sha = MessageDigest.getInstance("SHA-256").digest(apk).joinToString("") { "%02x".format(it) }
    private val mirrorBase = "http://10.77.0.1:8077/updates"
    private val publicBase = "https://github.com/o/r/releases/download/channels"
    private val apkUrl = "https://github.com/o/r/releases/download/v0.2.0/tailbond-0.2.0.apk"
    private val web = FakeWeb()
    private val installer = FakeInstaller()

    private fun manifest(version: String = "0.2.0", security: Boolean = false, requires: Int = 1, size: Int = apk.size, hash: String = sha) =
        """{"format":1,"channel":"stable","version":"$version","released":"2026-11-02T10:00:00Z","protocol":2,
          |"requires_server_protocol":$requires,"security":$security,"notes_url":"https://github.com/o/r/releases/tag/v$version",
          |"artifacts":[{"name":"tailbond-$version.apk","kind":"apk","arch":"arm64-v8a","url":"$apkUrl","sha256":"$hash","size":$size,
          |"version_code":${dev.tailbond.app.update.VersionCode.of(version)}}]}""".trimMargin().toByteArray()

    private fun publish(base: String, m: ByteArray = manifest(), signer: java.security.KeyPair = kp) {
        web.files["$base/stable.json"] = m
        web.files["$base/stable.json.sig"] = TestKeys.sigFile(signer, m)
    }

    inner class Env : UpdateEnv {
        override var currentVersion = "0.1.0"
        override val currentCode get() = dev.tailbond.app.update.VersionCode.of(currentVersion)!!.toLong()
        override var keys: List<ByteArray> = listOf(TestKeys.raw(kp))
        var policy = UpdatePolicy.AUTOMATIC
        var connected = true
        var protocol: Int? = 2
        var kbps = 0L
        var now = 1_000_000_000L
        var saved: UpdateStatus? = null
        var reconnectAsked = 0
        val announced = mutableListOf<Phase>()
        override fun policy() = policy
        override fun channel() = UpdateChannel.STABLE
        private fun mirror() = if (connected) UpdateSource(UpdateSource.MIRROR, mirrorBase, web, mapOf("Authorization" to "Bearer t"), "homebox") else null
        override fun sources() = listOfNotNull(mirror(), UpdateSource(UpdateSource.PUBLIC, publicBase, web, label = "github.com"))
        override fun downloadSources() = listOfNotNull(mirror(), UpdateSource(UpdateSource.PUBLIC, "", web, label = "link"))
        override fun serverProtocol() = if (connected) protocol else null
        override fun busyKbps() = kbps
        override fun now() = now
        override fun apkFile(name: String) = File(tmp.root, name)
        override fun load() = saved
        override fun save(s: UpdateStatus) { saved = s }
        override fun beforeInstall() { reconnectAsked++ }
        override fun announce(s: UpdateStatus) { announced += s.phase }
    }

    @Test fun automaticFromTheMirror() {
        publish(mirrorBase)
        web.files["$mirrorBase/files/tailbond-0.2.0.apk"] = apk
        val env = Env()
        val u = Updater(env, installer)
        u.check(manual = false)
        assertEquals(Phase.INSTALLING, u.status.value.phase)
        assertEquals("0.2.0", u.status.value.installing)
        assertEquals(UpdateSource.MIRROR, u.status.value.source)
        assertEquals(apk.toList(), installer.installed.single().toList())
        assertEquals(1, env.reconnectAsked)
        // the mirror got the client token
        assertTrue(web.asked.first().second["Authorization"] == "Bearer t")
        // nothing went to the public source
        assertTrue(web.asked.none { it.first.startsWith("https://") })

        // Android replaces the app; the new process starts on 0.2.0
        env.currentVersion = "0.2.0"
        val after = Updater(env, installer)
        assertEquals(Phase.IDLE, after.status.value.phase)
        assertEquals("Updated to 0.2.0", after.status.value.lastResult)
        assertNull(after.status.value.available)
    }

    @Test fun fallsBackToThePublicSource() {
        publish(publicBase)
        web.files[apkUrl] = apk
        web.down += "http://10.77.0.1"
        val u = Updater(Env(), installer)
        u.check(manual = false)
        assertEquals(Phase.INSTALLING, u.status.value.phase)
        assertEquals(UpdateSource.PUBLIC, u.status.value.source)
    }

    @Test fun mirrorWithABadSignatureFallsBackAndSaysSo() {
        publish(mirrorBase, signer = TestKeys.pair())
        publish(publicBase)
        web.files[apkUrl] = apk
        val u = Updater(Env(), installer)
        u.check(manual = false)
        assertEquals(UpdateSource.PUBLIC, u.status.value.source)
        assertEquals(Phase.INSTALLING, u.status.value.phase)
    }

    @Test fun untrustedEverywhereInstallsNothing() {
        val evil = TestKeys.pair()
        publish(mirrorBase, signer = evil)
        publish(publicBase, signer = evil)
        val u = Updater(Env(), installer)
        u.check(manual = true)
        assertTrue(installer.installed.isEmpty())
        assertTrue(u.status.value.lastResult, u.status.value.lastResult.startsWith("Refused an update that isn't signed"))
        assertFalse(u.status.value.lastResultOk)
    }

    @Test fun wrongHashIsRefused() {
        publish(mirrorBase)
        web.files["$mirrorBase/files/tailbond-0.2.0.apk"] = apk.copyOf().also { it[100] = 1 }
        web.files[apkUrl] = apk.copyOf().also { it[200] = 1 }
        val u = Updater(Env(), installer)
        u.check(manual = false)
        assertEquals(Phase.FAILED, u.status.value.phase)
        assertTrue(u.status.value.lastResult, u.status.value.lastResult.contains("SHA-256"))
        assertTrue(installer.installed.isEmpty())
        assertFalse(File(tmp.root, "tailbond-0.2.0.apk").exists())
    }

    @Test fun mirrorCopyBadPublicCopyGood() {
        publish(mirrorBase)
        web.files["$mirrorBase/files/tailbond-0.2.0.apk"] = apk.copyOf(100)
        web.files[apkUrl] = apk
        val u = Updater(Env(), installer)
        u.check(manual = false)
        assertEquals(Phase.INSTALLING, u.status.value.phase)
    }

    @Test fun deferredWhileBusyThenInstalls() {
        publish(mirrorBase)
        web.files["$mirrorBase/files/tailbond-0.2.0.apk"] = apk
        val env = Env().apply { kbps = 900 }
        val u = Updater(env, installer)
        u.check(manual = false)
        assertEquals(Phase.WAITING, u.status.value.phase)
        assertEquals("Version 0.2.0 installs when the phone is quiet", UpdateWords.headline(u.status.value))
        assertTrue(installer.installed.isEmpty())
        env.kbps = 20
        u.evaluate()
        assertEquals(Phase.INSTALLING, u.status.value.phase)
    }

    @Test fun installNowIgnoresTrafficButNotTheServer() {
        publish(mirrorBase, manifest(requires = 3))
        web.files["$mirrorBase/files/tailbond-0.2.0.apk"] = apk
        val env = Env().apply { kbps = 900 }
        val u = Updater(env, installer)
        u.check(manual = false)
        assertEquals(Phase.WAITING, u.status.value.phase)
        assertEquals(3, u.status.value.needsServer)
        u.install(manual = true)
        assertTrue(installer.installed.isEmpty())
        env.protocol = 3
        u.install(manual = true)
        assertEquals(1, installer.installed.size)
    }

    @Test fun notifyPolicyAsks() {
        publish(mirrorBase)
        val env = Env().apply { policy = UpdatePolicy.NOTIFY }
        val u = Updater(env, installer)
        u.check(manual = false)
        assertEquals(Phase.IDLE, u.status.value.phase)
        assertEquals("0.2.0", u.status.value.available)
        assertEquals(listOf(Phase.IDLE), env.announced)
        assertTrue(installer.installed.isEmpty())
    }

    @Test fun offPolicyChecksOnlyWhenAsked() {
        publish(mirrorBase)
        val env = Env().apply { policy = UpdatePolicy.OFF }
        val u = Updater(env, installer)
        assertFalse(u.checkDue())
        u.check(manual = false)
        assertTrue(web.asked.isEmpty())
        u.check(manual = true)
        assertEquals("0.2.0", u.status.value.available)
        assertTrue(installer.installed.isEmpty())
    }

    @Test fun noKeyMeansOff() {
        val env = Env().apply { keys = emptyList() }
        val u = Updater(env, installer)
        assertEquals(Phase.OFF, u.status.value.phase)
        assertTrue(u.status.value.noKeys)
        u.check(manual = true)
        assertTrue(web.asked.isEmpty())
        assertEquals("This build doesn't update itself", UpdateWords.headline(u.status.value))
    }

    @Test fun firstUpdateNeedsATap() {
        publish(mirrorBase)
        web.files["$mirrorBase/files/tailbond-0.2.0.apk"] = apk
        val env = Env()
        val u = Updater(env, installer)
        u.check(manual = false)
        u.onInstallResult(InstallResult.of(InstallResult.STATUS_PENDING_USER_ACTION, null))
        assertEquals(Phase.NEEDS_TAP, u.status.value.phase)
        assertEquals(listOf(Phase.NEEDS_TAP), env.announced)
        // the user tapped Update; Android answers success
        u.onInstallResult(InstallResult.of(InstallResult.STATUS_SUCCESS, null))
        assertEquals("Updated to 0.2.0", u.status.value.lastResult)
    }

    @Test fun cancelledOrFailedInstall() {
        publish(mirrorBase)
        web.files["$mirrorBase/files/tailbond-0.2.0.apk"] = apk
        val u = Updater(Env(), installer)
        u.check(manual = false)
        u.onInstallResult(InstallResult.of(InstallResult.STATUS_FAILURE_ABORTED, null))
        assertEquals(Phase.IDLE, u.status.value.phase)
        assertEquals("0.2.0", u.status.value.available)
        u.install(manual = true)
        u.onInstallResult(InstallResult.of(InstallResult.STATUS_FAILURE_STORAGE, "INSTALL_FAILED_INSUFFICIENT_STORAGE"))
        assertEquals(Phase.FAILED, u.status.value.phase)
        assertEquals("Couldn't install 0.2.0", UpdateWords.headline(u.status.value))
        assertEquals("Not enough storage space. tailbond is still on version 0.1.0", UpdateWords.detail(u.status.value, "homebox"))
    }

    @Test fun otherSignerAndPermission() {
        publish(mirrorBase)
        web.files["$mirrorBase/files/tailbond-0.2.0.apk"] = apk
        installer.allowed = false
        val u = Updater(Env(), installer)
        u.check(manual = false)
        assertEquals(Phase.NEEDS_PERMISSION, u.status.value.phase)
        installer.allowed = true
        installer.check = ApkCheck.OTHER_SIGNER
        u.install(manual = true)
        assertEquals(Phase.FAILED, u.status.value.phase)
        assertTrue(u.status.value.lastResult.contains("Debug and release builds can't update each other"))
        assertTrue(installer.installed.isEmpty())
    }

    @Test fun androidRefusesTheSession() {
        publish(mirrorBase)
        web.files["$mirrorBase/files/tailbond-0.2.0.apk"] = apk
        installer.refuse = "session error"
        val u = Updater(Env(), installer)
        u.check(manual = false)
        assertEquals(Phase.FAILED, u.status.value.phase)
        assertEquals("Couldn't install 0.2.0: session error", u.status.value.lastResult)
    }

    @Test fun interruptedInstallIsReportedAfterRestart() {
        val env = Env().apply { saved = UpdateStatus(phase = Phase.INSTALLING, installing = "0.2.0", available = "0.2.0") }
        val u = Updater(env, installer)
        assertEquals(Phase.IDLE, u.status.value.phase)
        assertTrue(u.status.value.lastResult.startsWith("The update to 0.2.0 didn't finish"))
    }

    @Test fun upToDate() {
        publish(mirrorBase, manifest("0.1.0"))
        val u = Updater(Env(), installer)
        u.check(manual = true)
        assertEquals("Up to date", UpdateWords.headline(u.status.value))
        assertTrue(u.status.value.lastCheckMs > 0)
        assertTrue(UpdateWords.lastChecked(u.status.value, u.status.value.lastCheckMs).endsWith("· from homebox"))
    }

    @Test fun schedule() {
        publish(mirrorBase, manifest("0.1.0"))
        val env = Env()
        val u = Updater(env, installer)
        assertTrue(u.checkDue())
        u.check(manual = false)
        assertFalse(u.checkDue())
        env.now += 5 * 3_600_000L
        assertFalse(u.checkDue())
        assertTrue(u.checkDue(30 * 60_000L))
        env.now += 3_600_000L
        assertTrue(u.checkDue())
    }

    @Test fun installStatusWords() {
        assertEquals(InstallResult.Success, InstallResult.of(0, null))
        assertEquals(InstallResult.NeedsUserAction, InstallResult.of(-1, null))
        val c = InstallResult.of(InstallResult.STATUS_FAILURE_CONFLICT, "INSTALL_FAILED_UPDATE_INCOMPATIBLE")
        assertTrue(c is InstallResult.Failed && c.why.contains("debug and release"))
        assertTrue((InstallResult.of(3, null) as InstallResult.Failed).blockedByUser)
        // the key id helper is what a signed manifest names
        assertEquals(16, ManifestVerifier.keyId(TestKeys.raw(kp)).length)
    }
}
