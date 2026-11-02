package dev.tailbond.app.update

/**
 * A release version, `MAJOR.MINOR.PATCH[-pre]` (docs/architecture.md,
 * "Updates"). A leading `v`, build metadata (`+…`) and an OpenWrt `-rN`
 * revision are ignored for ordering. Pre-releases sort before the release,
 * by semver rules (numeric identifiers numerically, then ASCII).
 */
data class SemVer(val major: Int, val minor: Int, val patch: Int, val pre: List<String> = emptyList()) : Comparable<SemVer> {
    override fun compareTo(other: SemVer): Int {
        compareValues(major, other.major).let { if (it != 0) return it }
        compareValues(minor, other.minor).let { if (it != 0) return it }
        compareValues(patch, other.patch).let { if (it != 0) return it }
        if (pre.isEmpty() || other.pre.isEmpty()) return compareValues(other.pre.size.coerceAtMost(1), pre.size.coerceAtMost(1))
        for (i in 0 until minOf(pre.size, other.pre.size)) {
            val c = comparePre(pre[i], other.pre[i])
            if (c != 0) return c
        }
        return compareValues(pre.size, other.pre.size)
    }

    override fun toString(): String = "$major.$minor.$patch" + if (pre.isEmpty()) "" else "-" + pre.joinToString(".")

    companion object {
        private val RE = Regex("""^(0|[1-9]\d{0,8})\.(0|[1-9]\d{0,8})\.(0|[1-9]\d{0,8})(?:-([0-9A-Za-z-]+(?:\.[0-9A-Za-z-]+)*))?$""")
        private val OPENWRT_REV = Regex("""-r\d+$""")
        private val GIT_DESCRIBE = Regex("""^\d+-g[0-9a-f]{4,}$""")

        /** Null when [s] isn't a version this scheme understands. */
        fun parse(s: String?): SemVer? {
            val t = s?.trim()?.removePrefix("v")?.removePrefix("V")?.substringBefore('+')?.replace(OPENWRT_REV, "") ?: return null
            val m = RE.matchEntire(t) ?: return null
            val (a, b, c, pre) = m.destructured
            return SemVer(a.toInt(), b.toInt(), c.toInt(), if (pre.isEmpty()) emptyList() else pre.split('.'))
        }

        /**
         * This build's own version, or null for a development build (`dev`,
         * a git hash, `-dirty`, `git describe`'s `-3-gabc1234`): such a
         * build never installs anything by itself.
         */
        fun own(s: String?): SemVer? {
            val v = parse(s) ?: return null
            if (v.pre.any { it.equals("dirty", true) || it.equals("dev", true) || GIT_DESCRIBE.matches(it) }) return null
            return v
        }

        private fun comparePre(a: String, b: String): Int {
            val an = a.all { it.isDigit() }
            val bn = b.all { it.isDigit() }
            return when {
                an && bn -> compareValues(a.toBigInteger(), b.toBigInteger())
                an -> -1
                bn -> 1
                else -> a.compareTo(b)
            }
        }
    }
}

/**
 * Android `versionCode` from the version name (agreed with the release
 * scripts: scripts/release/make-manifest.sh writes the same number into the
 * apk artifact's `version_code`). Must grow with every release:
 *
 *     release       MAJOR*1_000_000 + MINOR*10_000 + PATCH*100   (0.2.0 → 20000)
 *     pre-release   that base - 100 + N, N = the number the pre-release ends in
 *                   (1.2.3-beta.4 → 1020204; 0.2.0-rc1 → 19901)
 *
 * so every pre-release sorts below its release and above the previous
 * patch. N is clamped to 1..99 (no trailing number counts as 1). Pre-releases
 * of one version order by N only (alpha.2 > beta.1): number them in one
 * sequence. Limits: MINOR and PATCH ≤ 99, MAJOR ≤ 2100. A leading `v` is
 * ignored. android/app/build.gradle.kts has the same function (it can't call
 * app code); a test keeps them in step.
 */
object VersionCode {
    private val TRAILING = Regex("""(\d+)$""")

    fun of(name: String): Int? {
        val v = SemVer.parse(name) ?: return null
        if (v.minor > 99 || v.patch > 99 || v.major > 2100) return null
        val base = v.major * 1_000_000 + v.minor * 10_000 + v.patch * 100
        if (v.pre.isEmpty()) return base
        val n = TRAILING.find(v.pre.joinToString("."))?.groupValues?.get(1)?.takeLast(4)?.toInt() ?: 1
        return (base - 100 + n.coerceIn(1, 99)).coerceAtLeast(1)
    }
}
