package dev.tailbond.app

import dev.tailbond.app.update.Artifact
import dev.tailbond.app.update.Decision
import dev.tailbond.app.update.Deferral
import dev.tailbond.app.update.Manifest
import dev.tailbond.app.update.SemVer
import dev.tailbond.app.update.UpdatePolicy
import dev.tailbond.app.update.UpdateRules
import dev.tailbond.app.update.VersionCode
import org.junit.Assert.assertEquals
import org.junit.Assert.assertNull
import org.junit.Assert.assertTrue
import org.junit.Test

class VersionTest {
    @Test fun parsing() {
        assertEquals(SemVer(0, 2, 0), SemVer.parse("v0.2.0"))
        assertEquals(SemVer(0, 2, 0), SemVer.parse("0.2.0-r1"))
        assertEquals(SemVer(1, 2, 3, listOf("beta", "4")), SemVer.parse("1.2.3-beta.4+build.7"))
        assertEquals(SemVer(1, 2, 3, listOf("beta", "4")), SemVer.parse("1.2.3-beta.4-r2"))
        assertNull(SemVer.parse("dev"))
        assertNull(SemVer.parse("abc1234"))
        assertNull(SemVer.parse("1.2"))
        assertNull(SemVer.parse("01.2.3"))
    }

    @Test fun ordering() {
        val ordered = listOf(
            "0.1.0", "0.1.1", "0.2.0-alpha.1", "0.2.0-alpha.2", "0.2.0-beta.1", "0.2.0-beta.2", "0.2.0-beta.10",
            "0.2.0-rc.1", "0.2.0", "0.2.1", "0.10.0", "1.0.0-rc1", "1.0.0",
        ).map { SemVer.parse(it)!! }
        assertEquals(ordered, ordered.shuffled(java.util.Random(3)).sorted())
        // revision and leading v don't count
        assertEquals(0, SemVer.parse("v0.2.0-r3")!!.compareTo(SemVer.parse("0.2.0")!!))
    }

    @Test fun ownVersionRefusesDevBuilds() {
        assertEquals(SemVer(0, 2, 0), SemVer.own("0.2.0"))
        assertEquals(SemVer(0, 2, 0, listOf("beta", "1")), SemVer.own("0.2.0-beta.1"))
        assertNull(SemVer.own("dev"))
        assertNull(SemVer.own("0.2.0-dirty"))
        assertNull(SemVer.own("0.2.0-3-gabc1234"))
        assertNull(SemVer.own("0.2.0-dev"))
    }

    @Test fun versionCodeFormula() {
        // agreed with scripts/release/make-manifest.sh
        assertEquals(20000, VersionCode.of("0.2.0"))
        assertEquals(20000, VersionCode.of("v0.2.0"))
        assertEquals(1_020_300, VersionCode.of("1.2.3"))
        assertEquals(1_020_204, VersionCode.of("1.2.3-beta.4"))
        assertEquals(19901, VersionCode.of("0.2.0-rc1"))
        assertEquals(19901, VersionCode.of("0.2.0-beta"))
        assertEquals(19999, VersionCode.of("0.2.0-rc.250"))
        assertNull(VersionCode.of("0.100.0"))
        assertNull(VersionCode.of("dev"))
    }

    @Test fun versionCodeGrowsWithEveryRelease() {
        val seq = listOf("0.1.0", "0.1.1", "0.1.99", "0.2.0-beta.1", "0.2.0-beta.2", "0.2.0-rc.3", "0.2.0", "0.2.1-beta.1", "0.2.1", "0.99.99", "1.0.0-rc.1", "1.0.0", "2.0.0")
        val codes = seq.map { VersionCode.of(it)!! }
        assertEquals(codes, codes.sorted())
        assertEquals(codes.size, codes.toSet().size)
    }

    @Test fun gradleUsesTheSameFormula() {
        // build.gradle.kts computes versionCode itself; this build's pair must agree
        assertEquals(VersionCode.of(BuildConfig.VERSION_NAME), BuildConfig.VERSION_CODE)
    }
}

class UpdateRulesTest {
    private fun manifest(
        version: String = "0.2.0",
        code: Long? = 20000,
        requires: Int = 1,
        security: Boolean = false,
        apk: Boolean = true,
    ) = Manifest(
        format = 1, channel = "stable", version = version, requiresServerProtocol = requires, security = security,
        artifacts = if (apk) listOf(Artifact("tailbond-$version.apk", "apk", "arm64-v8a", "https://x/a.apk", "00", 10, code)) else emptyList(),
    )

    private fun decide(
        m: Manifest = manifest(),
        current: String = "0.1.0",
        code: Long = 10000,
        proto: Int? = 1,
        policy: UpdatePolicy = UpdatePolicy.AUTOMATIC,
        kbps: Long = 0,
    ) = UpdateRules.decide(current, code, m, proto, policy, kbps)

    @Test fun newerInstallsAutomatically() = assertEquals(Decision.Available("0.2.0", install = true), decide())

    @Test fun sameOrOlderIsUpToDate() {
        assertEquals(Decision.UpToDate, decide(current = "0.2.0", code = 20000))
        assertEquals(Decision.UpToDate, decide(current = "0.3.0", code = 30000))
        // a revision suffix isn't a newer version
        assertEquals(Decision.UpToDate, decide(m = manifest("0.2.0-r2"), current = "0.2.0", code = 20000))
    }

    @Test fun versionCodeMustGrowToo() =
        assertEquals(Decision.UpToDate, decide(m = manifest(code = 10000)))

    @Test fun notifyWaitsForATap() =
        assertEquals(Decision.Available("0.2.0", install = false, deferred = Deferral.ASK), decide(policy = UpdatePolicy.NOTIFY))

    @Test fun busyTunnelDefers() {
        assertEquals(Decision.Available("0.2.0", false, Deferral.BUSY), decide(kbps = 301))
        assertEquals(Decision.Available("0.2.0", true), decide(kbps = 300))
    }

    @Test fun securityIgnoresTraffic() =
        assertEquals(Decision.Available("0.2.0", true), decide(m = manifest(security = true), kbps = 5000))

    @Test fun serverProtocolGatesClients() {
        assertEquals(Decision.NeedsServer("0.2.0", 2, 1), decide(m = manifest(requires = 2), proto = 1))
        assertEquals(Decision.NeedsServer("0.2.0", 2, null), decide(m = manifest(requires = 2), proto = null))
        assertEquals(Decision.Available("0.2.0", true), decide(m = manifest(requires = 2), proto = 3))
        // protocol 1 needs no answer from the server
        assertEquals(Decision.Available("0.2.0", true), decide(m = manifest(requires = 1), proto = null))
        // compatibility comes before the security flag
        assertEquals(Decision.NeedsServer("0.2.0", 2, 1), decide(m = manifest(requires = 2, security = true), proto = 1))
    }

    @Test fun devBuildNeverInstallsByItself() =
        assertEquals(Decision.Available("0.2.0", false, Deferral.DEV_BUILD), decide(current = "0.1.0-dirty"))

    @Test fun noApk() = assertEquals(Decision.NoApk, decide(m = manifest(apk = false)))

    @Test fun checkSchedule() {
        val h = 3_600_000L
        assertTrue(UpdateRules.checkDue(0, 100))
        assertTrue(!UpdateRules.checkDue(10 * h, 15 * h))
        assertTrue(UpdateRules.checkDue(10 * h, 16 * h))
        // clock went backwards
        assertTrue(UpdateRules.checkDue(10 * h, 9 * h))
    }

    @Test fun trafficAverage() {
        assertEquals(0, UpdateRules.busyKbps(emptyList(), emptyList()))
        // 60 s of a 400 kbit/s call down + 100 up = 500
        assertEquals(500, UpdateRules.busyKbps(List(60) { 400_000L }, List(60) { 100_000L }))
        // a 10 s burst at 3 Mbit/s in an otherwise idle minute averages 500
        assertEquals(500, UpdateRules.busyKbps(List(50) { 0L } + List(10) { 3_000_000L }, List(60) { 0L }))
        // only the last minute counts
        assertEquals(0, UpdateRules.busyKbps(List(30) { 9_000_000L } + List(60) { 0L }, emptyList()))
    }
}
