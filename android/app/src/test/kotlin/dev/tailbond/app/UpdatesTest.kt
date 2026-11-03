package dev.tailbond.app

import dev.tailbond.app.backup.AndroidBackup
import dev.tailbond.app.model.AppSettings
import dev.tailbond.app.update.Availability
import dev.tailbond.app.update.HttpsGet
import dev.tailbond.app.update.ReleaseException
import dev.tailbond.app.update.ReleaseParser
import dev.tailbond.app.update.SemVer
import dev.tailbond.app.update.UpdateChecker
import dev.tailbond.app.update.UpdateEnv
import dev.tailbond.app.update.UpdatePolicy
import dev.tailbond.app.update.UpdateRules
import dev.tailbond.app.update.UpdateStatus
import dev.tailbond.app.update.UpdateWords
import org.junit.Assert.assertEquals
import org.junit.Assert.assertFalse
import org.junit.Assert.assertNull
import org.junit.Assert.assertThrows
import org.junit.Assert.assertTrue
import org.junit.Test
import java.net.UnknownHostException
import java.time.ZoneOffset

/** A trimmed `GET /repos/<repo>/releases/latest` answer. */
private fun release(
    tag: String = "v0.2.0",
    assets: List<String> = listOf("tailbond-0.2.0.apk", "tailbond-0.2.0-debug.apk", "tailbond-linux-x86_64.tar.gz", "SHA256SUMS"),
    draft: Boolean = false,
    prerelease: Boolean = false,
    htmlUrl: String = "https://github.com/kaovilai/tailbond/releases/tag/$tag",
) = """
    {"url":"https://api.github.com/repos/kaovilai/tailbond/releases/1","html_url":"$htmlUrl","id":1,
     "tag_name":"$tag","name":"tailbond $tag","draft":$draft,"prerelease":$prerelease,
     "published_at":"2026-11-02T10:00:00Z","body":"## Changes\n- things",
     "assets":[${assets.joinToString(",") { """{"name":"$it","size":123,"browser_download_url":"https://github.com/kaovilai/tailbond/releases/download/$tag/$it"}""" }}]}
""".trimIndent()

class ReleaseParserTest {
    @Test fun parsesTheLatestRelease() {
        val r = ReleaseParser.parse(release())
        assertEquals("0.2.0", r.version)
        assertEquals("https://github.com/kaovilai/tailbond/releases/tag/v0.2.0", r.notesUrl)
        assertEquals("https://github.com/kaovilai/tailbond/releases/download/v0.2.0/tailbond-0.2.0.apk", r.apkUrl)
    }

    @Test fun tagWithoutV() = assertEquals("1.4.2", ReleaseParser.parse(release(tag = "1.4.2", assets = listOf("tailbond-1.4.2.apk"))).version)

    @Test fun missingApkMeansNoDownload() {
        assertNull(ReleaseParser.parse(release(assets = listOf("tailbond-linux-x86_64.tar.gz", "SHA256SUMS"))).apkUrl)
        assertNull(ReleaseParser.parse(release(assets = emptyList())).apkUrl)
        // no "assets" key at all
        assertNull(ReleaseParser.parse("""{"tag_name":"v0.2.0","html_url":"https://x/y"}""").apkUrl)
    }

    @Test fun debugApkAndOtherVersionsNeverCount() {
        assertNull(ReleaseParser.parse(release(assets = listOf("tailbond-0.2.0-debug.apk"))).apkUrl)
        assertNull(ReleaseParser.parse(release(assets = listOf("tailbond-0.1.9.apk", "tailbond-debug.apk", "app-release.apk"))).apkUrl)
    }

    @Test fun assetSelection() {
        val v = SemVer.parse("0.2.0")!!
        val pick = { names: List<String> -> ReleaseParser.apkAsset(names.map { it to "https://h/$it" }, v) }
        assertEquals("https://h/tailbond-0.2.0.apk", pick(listOf("tailbond-0.2.0-debug.apk", "tailbond-0.2.0.apk")))
        assertEquals("https://h/tailbond-v0.2.0.apk", pick(listOf("tailbond-v0.2.0.apk")))
        assertNull(pick(listOf("tailbond-0.2.0.apk.sha256", "Tailbond-0.2.0.APK", "tailbond-0.2.0-beta.1.apk")))
        // a download link that isn't https isn't used
        assertNull(ReleaseParser.apkAsset(listOf("tailbond-0.2.0.apk" to "http://h/tailbond-0.2.0.apk"), v))
        assertNull(ReleaseParser.apkAsset(listOf("tailbond-0.2.0.apk" to null), v))
    }

    @Test fun oddTagsAndReleasesAreRefused() {
        assertThrows(ReleaseException::class.java) { ReleaseParser.parse(release(tag = "nightly")) }
        assertThrows(ReleaseException::class.java) { ReleaseParser.parse(release(tag = "v1.2")) }
        assertThrows(ReleaseException::class.java) { ReleaseParser.parse(release(tag = "")) }
        assertThrows(ReleaseException::class.java) { ReleaseParser.parse(release(tag = "v0.3.0-beta.1")) }
        assertThrows(ReleaseException::class.java) { ReleaseParser.parse(release(draft = true)) }
        assertThrows(ReleaseException::class.java) { ReleaseParser.parse(release(prerelease = true)) }
        assertThrows(ReleaseException::class.java) { ReleaseParser.parse("""{"message":"Not Found"}""") }
        assertThrows(ReleaseException::class.java) { ReleaseParser.parse("<html>") }
        assertThrows(ReleaseException::class.java) { ReleaseParser.parse("""[{"tag_name":"v0.2.0"}]""") }
    }

    @Test fun nonHttpsNotesAreDropped() = assertNull(ReleaseParser.parse(release(htmlUrl = "javascript:alert(1)")).notesUrl)
}

class UpdateRulesTest {
    private val h = 60 * 60 * 1000L
    private val day = 24 * h
    private val now = 100 * day

    @Test fun versionCompare() {
        assertEquals(Availability.AVAILABLE, UpdateRules.availability("0.1.0", "0.2.0", "https://a"))
        assertEquals(Availability.AVAILABLE, UpdateRules.availability("0.9.9", "0.10.0", "https://a"))
        assertEquals(Availability.AVAILABLE, UpdateRules.availability("0.2.0-beta.3", "0.2.0", "https://a"))
        assertEquals(Availability.UP_TO_DATE, UpdateRules.availability("0.2.0", "0.2.0", "https://a"))
        assertEquals(Availability.UP_TO_DATE, UpdateRules.availability("0.3.0", "0.2.0", "https://a"))
        assertEquals(Availability.UP_TO_DATE, UpdateRules.availability("v0.2.0", "0.2.0", null))
        assertEquals(Availability.NO_APK, UpdateRules.availability("0.1.0", "0.2.0", null))
        assertEquals(Availability.UNKNOWN, UpdateRules.availability("0.1.0", null, null))
    }

    @Test fun devBuildsShowButNeverNag() {
        for (dev in listOf("dev", "0.1.0-dirty", "0.1.0-3-gabc1234", "abc1234", "")) {
            assertEquals(dev, Availability.DEV_BUILD, UpdateRules.availability(dev, "0.2.0", "https://a"))
        }
        val s = UpdateStatus(current = "dev", latest = "0.2.0", apkUrl = "https://a", lastCheckMs = 1)
        assertEquals("Latest release: v0.2.0", UpdateWords.headline(s, UpdatePolicy.NOTIFY))
        assertFalse(UpdateWords.row(s, UpdatePolicy.NOTIFY, 1).contains("available"))
    }

    @Test fun checksAboutOnceADay() {
        val p = UpdatePolicy.NOTIFY
        assertTrue("never checked", UpdateRules.checkDue(p, 0, 0, now))
        assertFalse(UpdateRules.checkDue(p, now - 23 * h, now - 23 * h, now))
        assertTrue(UpdateRules.checkDue(p, now - 24 * h, now - 24 * h, now))
        assertTrue(UpdateRules.checkDue(p, now - 3 * day, now - 3 * day, now))
        // a failed try waits an hour before the next automatic one
        assertFalse(UpdateRules.checkDue(p, now - 2 * day, now - 10 * 60_000, now))
        assertTrue(UpdateRules.checkDue(p, now - 2 * day, now - h, now))
        // the clock went backwards
        assertTrue(UpdateRules.checkDue(p, now + h, now + h, now))
    }

    @Test fun policyOffNeverChecksByItself() {
        assertFalse(UpdateRules.checkDue(UpdatePolicy.OFF, 0, 0, now))
        assertFalse(UpdateRules.checkDue(UpdatePolicy.OFF, now - 30 * day, now - 30 * day, now))
    }

    @Test fun repoSetting() {
        assertEquals("kaovilai/tailbond", UpdateRules.repo("kaovilai/tailbond"))
        assertEquals("kaovilai/tailbond", UpdateRules.repo("https://github.com/kaovilai/tailbond/"))
        assertEquals("o/r.git-x", UpdateRules.repo("o/r.git-x"))
        assertNull(UpdateRules.repo("kaovilai"))
        assertNull(UpdateRules.repo("https://example.com/a/b"))
        assertNull(UpdateRules.repo("a/b/c"))
        assertEquals("https://api.github.com/repos/o/r/releases/latest", UpdateRules.latestUrl("o/r"))
    }

    @Test fun oldPoliciesStillLoad() {
        // settings and backups from the self-updating versions: "auto" and an update channel
        val old = """{"onboarded":true,"cellularCapMb":5000,"updatePolicy":"auto","updateChannel":"beta"}"""
        val s = AppSettings.fromJson(old)
        assertEquals(UpdatePolicy.NOTIFY, s.updatePolicy)
        assertEquals(5000L, s.cellularCapMb)
        assertTrue(s.onboarded)
        assertEquals(UpdatePolicy.OFF, AppSettings.fromJson("""{"updatePolicy":"off"}""").updatePolicy)
        assertTrue(AppSettings().toJson().contains("\"updatePolicy\":\"notify\""))
        val b = AndroidBackup.fromJson("""{"format":1,"config":{"servers":[],"settings":$old}}""")
        assertEquals(UpdatePolicy.NOTIFY, b.config.settings.updatePolicy)
    }
}

class UpdateCheckerTest {
    private val day = 24 * 60 * 60 * 1000L

    private inner class Env(override val currentVersion: String = "0.1.0") : UpdateEnv {
        override val repo = "kaovilai/tailbond"
        var policy = UpdatePolicy.NOTIFY
        var now = 100 * day
        var saved: UpdateStatus? = null
        override fun policy() = policy
        override fun now() = now
        override fun load() = saved
        override fun save(s: UpdateStatus) { saved = s }
    }

    private class Web(var code: Int = 200, var body: String = release(), var fail: Exception? = null) : HttpsGet {
        val asked = mutableListOf<Pair<String, Map<String, String>>>()
        override fun get(url: String, headers: Map<String, String>): Pair<Int, String> {
            asked += url to headers
            fail?.let { throw it }
            return code to body
        }
    }

    @Test fun findsAnUpdate() {
        val env = Env()
        val web = Web()
        val c = UpdateChecker(env, web)
        assertTrue(c.checkIfDue())
        val s = c.status.value
        assertEquals("https://api.github.com/repos/kaovilai/tailbond/releases/latest", web.asked.single().first)
        assertEquals("application/vnd.github+json", web.asked.single().second["Accept"])
        assertEquals("tailbond-android/0.1.0", web.asked.single().second["User-Agent"])
        assertEquals(Availability.AVAILABLE, s.availability)
        assertEquals("Update available: v0.2.0", UpdateWords.headline(s, env.policy))
        assertEquals(env.now, s.lastCheckMs)
        assertNull(s.error)
        assertFalse(s.checking)
        assertEquals(s, env.saved)
        // not again the same day
        env.now += day / 2
        assertFalse(c.checkIfDue())
        assertEquals(1, web.asked.size)
        // a day later, yes
        env.now += day / 2
        assertTrue(c.checkIfDue())
        assertEquals(2, web.asked.size)
    }

    @Test fun upToDateAndRemembered() {
        val env = Env(currentVersion = "0.2.0")
        UpdateChecker(env, Web()).check()
        assertEquals(Availability.UP_TO_DATE, env.saved!!.availability)
        // after a restart (an update installed by hand) the saved status is read again, with this build's version
        val after = UpdateChecker(Env(currentVersion = "0.2.0").also { it.saved = env.saved }, Web())
        assertEquals("Up to date", UpdateWords.headline(after.status.value, UpdatePolicy.NOTIFY))
        val newer = UpdateChecker(Env(currentVersion = "0.1.0").also { it.saved = env.saved }, Web())
        assertEquals("0.1.0", newer.status.value.current)
        assertEquals(Availability.AVAILABLE, newer.status.value.availability)
    }

    @Test fun policyOffOnlyChecksWhenAsked() {
        val env = Env().apply { policy = UpdatePolicy.OFF }
        val web = Web()
        val c = UpdateChecker(env, web)
        assertFalse(c.checkIfDue())
        assertTrue(web.asked.isEmpty())
        assertEquals("Updates are off", UpdateWords.headline(c.status.value, env.policy))
        assertEquals("Automatic checks are off", UpdateWords.row(c.status.value, env.policy, env.now))
        c.check()
        assertEquals(1, web.asked.size)
        assertEquals(Availability.AVAILABLE, c.status.value.availability)
        // what Check now found still shows
        assertEquals("Update available: v0.2.0", UpdateWords.row(c.status.value, env.policy, env.now))
        env.now += 30 * day
        assertFalse(c.checkIfDue())
    }

    @Test fun releaseWithoutApk() {
        val env = Env()
        val c = UpdateChecker(env, Web(body = release(assets = listOf("SHA256SUMS"))))
        c.check()
        assertEquals(Availability.NO_APK, c.status.value.availability)
        assertEquals("Up to date", UpdateWords.headline(c.status.value, env.policy))
        assertEquals("Version 0.2.0 is out, but it has no Android app yet.", UpdateWords.detail(c.status.value, env.policy))
    }

    @Test fun failuresInPlainWords() {
        val env = Env()
        val web = Web(fail = UnknownHostException("api.github.com"))
        val c = UpdateChecker(env, web)
        c.check()
        var s = c.status.value
        assertEquals("Couldn't check for updates", UpdateWords.headline(s, env.policy))
        assertEquals(
            "No internet connection, or GitHub can't be reached. tailbond works as before; it tries again later by itself, or tap Check now.",
            UpdateWords.detail(s, env.policy),
        )
        assertEquals(0L, s.lastCheckMs)
        assertEquals(env.now, s.lastAttemptMs)
        // retried after an hour, not at once
        env.now += 10 * 60_000
        assertFalse(c.checkIfDue())
        env.now += 60 * 60_000
        web.fail = null
        web.code = 404
        assertTrue(c.checkIfDue())
        assertEquals("kaovilai/tailbond has no releases on GitHub yet", c.status.value.error)
        web.code = 403
        c.check()
        assertTrue(c.status.value.error!!.startsWith("GitHub is limiting"))
        web.code = 200
        web.body = release(tag = "latest")
        c.check()
        assertEquals("the latest release's name \"latest\" isn't a version number", c.status.value.error)
        // a good check clears the error
        web.body = release()
        c.check()
        s = c.status.value
        assertNull(s.error)
        assertEquals(Availability.AVAILABLE, s.availability)
    }

    @Test fun failureKeepsWhatWasKnown() {
        val env = Env()
        val web = Web()
        val c = UpdateChecker(env, web)
        c.check()
        val good = env.now
        env.now += 2 * day
        web.fail = java.net.SocketTimeoutException("read timed out")
        c.check()
        val s = c.status.value
        // still "Update available", with the error underneath
        assertEquals("Update available: v0.2.0", UpdateWords.headline(s, env.policy))
        assertEquals(good, s.lastCheckMs)
        assertEquals("GitHub didn't answer in time", s.error)
        assertTrue(UpdateWords.lastChecked(s, env.now, ZoneOffset.UTC).startsWith("Tried today at"))
        assertTrue(UpdateWords.lastChecked(s, env.now, ZoneOffset.UTC).contains("Last good check on"))
    }
}
