package dev.tailbond.app

import dev.tailbond.app.update.SemVer
import dev.tailbond.app.update.VersionCode
import org.junit.Assert.assertEquals
import org.junit.Assert.assertNull
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
