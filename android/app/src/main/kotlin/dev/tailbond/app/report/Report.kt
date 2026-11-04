package dev.tailbond.app.report

/**
 * Report a problem (docs/reporting.md): a port of agent/internal/report.
 * The redactor, the report text and the prefilled GitHub issue URL must
 * match the Go reference byte for byte; ReportTest runs the shared
 * agent/internal/report/testdata/vectors.json.
 *
 * Nothing here sends anything: the app opens the URL in the browser, where
 * the user edits the issue and submits it.
 */
object ReportText {
    const val REPO = "kaovilai/tailbond"
    const val MAX_URL = 7000
    const val NOTICE = "Nothing is sent automatically: tailbond has no telemetry and makes no network call for this. " +
        "The report opens in your browser as a GitHub issue that you can still edit, and once you submit it, it is public. " +
        "Keys, pairing links, public IP addresses, email addresses, your tailnet's name and similar details were replaced with [brackets]; " +
        "read it and remove anything else you don't want public (Wi-Fi names, device names)."
    const val CONFIRM = "I've read this report and it has nothing I don't want public."
    const val SECURITY_NOTE = "A security problem? Don't open a public issue: report it privately on GitHub (Security → Report a vulnerability)."
    const val SECURITY_URL = "https://github.com/$REPO/security/advisories/new"
}

object Redactor {
    private class Rule(val re: Regex, val repl: String? = null, val fn: ((String) -> String)? = null)

    // the same rules, in the same order, as report.go
    private val rules = listOf(
        Rule(Regex("""(tailbond://[A-Za-z0-9._-]+)\?[^\s"'<>]+"""), "\$1?[redacted]"),
        Rule(Regex("""tskey-[A-Za-z0-9-]+"""), "[tailscale key]"),
        Rule(Regex("""(?i)\b(Basic|Bearer)\s+[A-Za-z0-9+/=._-]+"""), "\$1 [redacted]"),
        Rule(Regex("""(?i)(--?(?:password|passwd|pass|token|secret|psk|authkey|auth-key|key))[ \t]+[^\s-][^\s]*"""), "\$1 [redacted]"),
        Rule(Regex("""(?i)\b(share_key|private_key|client_key|api_key|key|token|secret|password|passwd|pass|psk|auth|cookie)("?[ \t]*[:=][ \t]*"?)[^\s",;&}\]]+"""), "\$1\$2[redacted]"),
        Rule(Regex("""\b[0-9A-Fa-f]{2}(?:[:-][0-9A-Fa-f]{2}){5}\b"""), "[mac]"),
        Rule(Regex("""\b[0-9A-Fa-f]{32,}\b"""), "[hex]"),
        Rule(Regex("""[A-Za-z0-9+/_-]{40,}={0,2}"""), fn = { m -> if (randomish(m)) "[secret]" else m }),
        Rule(Regex("""[A-Za-z0-9._%+-]+@[A-Za-z0-9.-]+\.[A-Za-z]{2,}"""), "[email]"),
        Rule(Regex("""(?i)\b([A-Za-z0-9-]+)\.[A-Za-z0-9-]+\.ts\.net\b"""), "\$1.[tailnet].ts.net"),
        Rule(Regex("""(/Users|/home)/[^/\s"']+"""), "\$1/[user]"),
        Rule(Regex("""\b[0-9]{1,3}\.[0-9]{1,3}\.[0-9]{1,3}\.[0-9]{1,3}\b(?:-[A-Za-z][A-Za-z0-9]*)?"""), fn = { m ->
            if (!m.contains('-') && publicIP(m)) "[public IPv4]" else m
        }),
        Rule(Regex("""[0-9A-Fa-f]{0,4}(?::[0-9A-Fa-f]{0,4}){2,7}"""), fn = { m ->
            when {
                publicIP(m) -> "[public IPv6]"
                m.startsWith(":") && !m.startsWith("::") && publicIP(m.substring(1)) -> ":[public IPv6]"
                else -> m
            }
        }),
    )

    fun redact(input: String): String {
        var s = input
        for (r in rules) {
            s = if (r.fn != null) r.re.replace(s) { r.fn.invoke(it.value) } else r.re.replace(s, r.repl!!)
        }
        return s
    }

    private fun randomish(s: String): Boolean {
        if (s.startsWith("/")) return false
        return s.any { it in 'A'..'Z' } && s.any { it in 'a'..'z' } && s.any { it in '0'..'9' }
    }

    internal fun publicIP(s: String): Boolean {
        parseIPv4(s)?.let { b ->
            return when {
                b[0] == 0 || b[0] == 10 || b[0] == 127 || b[0] >= 224 -> false
                b[0] == 100 && (b[1] and 0xc0) == 64 -> false
                b[0] == 169 && b[1] == 254 -> false
                b[0] == 172 && (b[1] and 0xf0) == 16 -> false
                b[0] == 192 && b[1] == 168 -> false
                else -> true
            }
        }
        val first = parseIPv6First(s) ?: return false
        return first in 0x2000..0x3fff
    }

    private fun parseIPv4(s: String): IntArray? {
        val parts = s.split('.')
        if (parts.size != 4) return null
        val b = IntArray(4)
        for ((i, p) in parts.withIndex()) {
            if (p.isEmpty() || p.length > 3 || (p.length > 1 && p[0] == '0')) return null
            if (!p.all { it in '0'..'9' }) return null
            val n = p.toInt()
            if (n > 255) return null
            b[i] = n
        }
        return b
    }

    private fun groups(p: String): List<Int>? {
        if (p.isEmpty()) return emptyList()
        return p.split(':').map { g ->
            if (g.isEmpty() || g.length > 4) return null
            var n = 0
            for (c in g) {
                n = n * 16 + when (c) {
                    in '0'..'9' -> c - '0'
                    in 'a'..'f' -> c - 'a' + 10
                    in 'A'..'F' -> c - 'A' + 10
                    else -> return null
                }
            }
            n
        }
    }

    private fun parseIPv6First(s: String): Int? {
        val i = s.indexOf("::")
        if (i >= 0) {
            if (s.substring(i + 2).contains("::")) return null
            val l = groups(s.substring(0, i)) ?: return null
            val r = groups(s.substring(i + 2)) ?: return null
            if (l.size + r.size > 7) return null
            return l.firstOrNull() ?: 0
        }
        val g = groups(s) ?: return null
        return if (g.size == 8) g[0] else null
    }
}

/** One problem report; every field is redacted when used (report.go Report). */
data class Report(
    val component: String = "",
    val version: String = "",
    val device: String = "",
    val what: String = "",
    val steps: String = "",
    val logs: String = "",
    val crash: String = "",
) {
    private fun redacted() = Report(
        component = Redactor.redact(component.trim()),
        version = Redactor.redact(version.trim()),
        device = Redactor.redact(device.trim()),
        what = Redactor.redact(what.trim()),
        steps = Redactor.redact(steps.trim()),
        logs = Redactor.redact(logs.trimEnd('\n')),
        crash = Redactor.redact(crash.trimEnd('\n')),
    )

    private fun logsField(): String = when {
        crash.isEmpty() -> logs
        logs.isEmpty() -> "--- crash ---\n$crash"
        else -> "--- crash ---\n$crash\n--- logs ---\n$logs"
    }

    /** The whole report as text ("Copy full report", "Save report file"). */
    fun full(): String {
        val r = redacted()
        val b = StringBuilder()
        fun sec(h: String, v: String) = b.append("### ").append(h).append("\n\n").append(v.ifEmpty { "(not given)" }).append("\n\n")
        sec("Where does it happen?", r.component)
        sec("What happened, and what did you expect?", r.what)
        sec("Steps to reproduce", r.steps)
        sec("tailbond version", r.version)
        sec("Device and firmware / OS", r.device)
        b.append("### Logs\n\n```text\n").append(r.logsField()).append("\n```\n")
        return b.toString()
    }

    /** The prefilled bug_report.yml issue, at most MAX_URL long (oldest log lines go first). */
    fun issueUrl(): String {
        val r = redacted()
        var lines = if (r.logs.isEmpty()) emptyList() else r.logs.split("\n")
        var dropped = 0
        while (true) {
            var logs = lines.joinToString("\n")
            if (dropped > 0) logs = "($dropped earlier lines left out: the full report has them)\n$logs"
            val rr = r.copy(logs = logs)
            val u = rr.url()
            if (u.length <= ReportText.MAX_URL || lines.isEmpty()) {
                if (u.length > ReportText.MAX_URL) {
                    return rr.copy(crash = "", logs = "(the crash report is too long for a link: paste the full report here)").url()
                }
                return u
            }
            val n = maxOf(1, lines.size / 10)
            lines = lines.drop(n)
            dropped += n
        }
    }

    private fun url(): String {
        val title = "bug: " + firstLine(what, 60)
        val q = listOf(
            "template=bug_report.yml",
            "title=" + escape(title),
            "component=" + escape(component),
            "what=" + escape(what),
            "steps=" + escape(steps),
            "version=" + escape(version),
            "device=" + escape(device),
            "logs=" + escape(logsField()),
        )
        return "https://github.com/${ReportText.REPO}/issues/new?" + q.joinToString("&")
    }

    companion object {
        /** Percent-encoding as report.go Escape: A-Z a-z 0-9 - . _ ~ kept, the rest %XX of UTF-8. */
        fun escape(s: String): String {
            val hex = "0123456789ABCDEF"
            val b = StringBuilder()
            for (byte in s.toByteArray(Charsets.UTF_8)) {
                val c = byte.toInt() and 0xff
                val ch = c.toChar()
                if (ch in 'A'..'Z' || ch in 'a'..'z' || ch in '0'..'9' || ch == '-' || ch == '.' || ch == '_' || ch == '~') {
                    b.append(ch)
                } else {
                    b.append('%').append(hex[c shr 4]).append(hex[c and 15])
                }
            }
            return b.toString()
        }

        internal fun firstLine(s: String, n: Int): String {
            val line = s.substringBefore('\n').trim()
            val cps = line.codePoints().toArray()
            if (cps.size > n) return String(cps, 0, n - 1) + "…"
            return line
        }

        /** "Save report file"'s name, from an RFC 3339 UTC time. */
        fun fileName(at: String): String {
            val t = at.replace(":", "").replace("-", "").replace("T", "-").replace("Z", "")
            return if (t.isEmpty()) "tailbond-report.txt" else "tailbond-report-$t.txt"
        }
    }
}
