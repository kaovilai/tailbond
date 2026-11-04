import Foundation

/// Report a problem (docs/reporting.md): a port of agent/internal/report.
/// The redactor, the report text and the prefilled GitHub issue URL must
/// match the Go reference byte for byte; ReportTests runs the shared
/// agent/internal/report/testdata/vectors.json.
///
/// Nothing here sends anything: the app opens the URL in the browser, where
/// the user edits the issue and submits it.
public enum ReportText {
    public static let repo = "kaovilai/tailbond"
    public static let maxURL = 7000
    public static let notice = "Nothing is sent automatically: tailbond has no telemetry and makes no network call for this. " +
        "The report opens in your browser as a GitHub issue that you can still edit, and once you submit it, it is public. " +
        "Keys, pairing links, public IP addresses, email addresses, your tailnet's name, your work VPN's internal names (where tailbond knows them) and similar details were replaced with [brackets]. " +
        "Private addresses (10.x, 192.168.x, 100.x) stay: they help find the problem. " +
        "Read it and remove anything else you don't want public (Wi-Fi names, device names)."
    public static let accountNote = "Submitting the issue needs a GitHub account. No account? Save the report file and send it to whoever set up tailbond for you."
    public static let confirm = "I've read this report and it has nothing I don't want public."
    public static let securityNote = "A security problem? Don't open a public issue: report it privately on GitHub (Security → Report a vulnerability)."
    public static let securityURL = "https://github.com/\(repo)/security/advisories/new"
}

public enum Redactor {
    private struct Rule {
        let re: NSRegularExpression
        let template: String?
        let fn: ((String) -> String)?
        init(_ pattern: String, _ template: String) {
            re = try! NSRegularExpression(pattern: pattern)
            self.template = template
            fn = nil
        }
        init(_ pattern: String, fn: @escaping (String) -> String) {
            re = try! NSRegularExpression(pattern: pattern)
            template = nil
            self.fn = fn
        }
    }

    // the same rules, in the same order, as report.go
    private static let rules: [Rule] = [
        Rule(#"(tailbond://[A-Za-z0-9._-]+)\?[^\s"'<>]+"#, "$1?[redacted]"),
        Rule(#"tskey-[A-Za-z0-9-]+"#, "[tailscale key]"),
        Rule(#"(?i)\b(Basic|Bearer)\s+[A-Za-z0-9+/=._-]+"#, "$1 [redacted]"),
        Rule(#"(?i)(--?(?:password|passwd|pass|token|secret|psk|authkey|auth-key|key))[ \t]+[^\s-][^\s]*"#, "$1 [redacted]"),
        Rule(#"(?i)\b(share_key|private_key|client_key|api_key|key|token|secret|password|passwd|pass|psk|auth|cookie)("?[ \t]*[:=][ \t]*"?)[^\s",;&}\]]+"#, "$1$2[redacted]"),
        Rule(#"\b[0-9A-Fa-f]{2}(?:[:-][0-9A-Fa-f]{2}){5}\b"#, "[mac]"),
        Rule(#"\b[0-9A-Fa-f]{32,}\b"#, "[hex]"),
        Rule(#"[A-Za-z0-9+/_-]{40,}={0,2}"#, fn: { randomish($0) ? "[secret]" : $0 }),
        Rule(#"[A-Za-z0-9._%+-]+@[A-Za-z0-9.-]+\.[A-Za-z]{2,}"#, "[email]"),
        Rule(#"(?i)\b([A-Za-z0-9-]+)\.[A-Za-z0-9-]+\.ts\.net\b"#, "$1.[tailnet].ts.net"),
        Rule(#"(/Users|/home)/[^/\s"']+"#, "$1/[user]"),
        Rule(#"\b[0-9]{1,3}\.[0-9]{1,3}\.[0-9]{1,3}\.[0-9]{1,3}\b(?:-[A-Za-z][A-Za-z0-9]*)?"#, fn: { m in
            (!m.contains("-") && publicIP(m)) ? "[public IPv4]" : m
        }),
        Rule(#"[0-9A-Fa-f]{0,4}(?::[0-9A-Fa-f]{0,4}){2,7}"#, fn: { m in
            if publicIP(m) { return "[public IPv6]" }
            if m.hasPrefix(":") && !m.hasPrefix("::") && publicIP(String(m.dropFirst())) { return ":[public IPv6]" }
            return m
        }),
    ]

    public static func redact(_ input: String) -> String {
        var s = input
        for r in rules {
            let ns = s as NSString
            let all = NSRange(location: 0, length: ns.length)
            if let t = r.template {
                s = r.re.stringByReplacingMatches(in: s, range: all, withTemplate: t)
            } else if let fn = r.fn {
                let out = NSMutableString(string: s)
                for m in r.re.matches(in: s, range: all).reversed() {
                    out.replaceCharacters(in: m.range, with: fn(ns.substring(with: m.range)))
                }
                s = out as String
            }
        }
        return s
    }

    static func randomish(_ s: String) -> Bool {
        if s.hasPrefix("/") { return false }
        let u = s.unicodeScalars
        return u.contains { $0 >= "A" && $0 <= "Z" } && u.contains { $0 >= "a" && $0 <= "z" } && u.contains { $0 >= "0" && $0 <= "9" }
    }

    static func publicIP(_ s: String) -> Bool {
        if let b = parseIPv4(s) {
            if b[0] == 0 || b[0] == 10 || b[0] == 127 || b[0] >= 224 { return false }
            if b[0] == 100 && (b[1] & 0xc0) == 64 { return false }
            if b[0] == 169 && b[1] == 254 { return false }
            if b[0] == 172 && (b[1] & 0xf0) == 16 { return false }
            if b[0] == 192 && b[1] == 168 { return false }
            return true
        }
        guard let first = parseIPv6First(s) else { return false }
        return first >= 0x2000 && first <= 0x3fff
    }

    private static func parseIPv4(_ s: String) -> [Int]? {
        let parts = s.split(separator: ".", omittingEmptySubsequences: false)
        if parts.count != 4 { return nil }
        var b: [Int] = []
        for p in parts {
            if p.isEmpty || p.count > 3 || (p.count > 1 && p.first == "0") { return nil }
            if !p.unicodeScalars.allSatisfy({ $0 >= "0" && $0 <= "9" }) { return nil }
            guard let n = Int(p), n <= 255 else { return nil }
            b.append(n)
        }
        return b
    }

    private static func groups(_ p: Substring) -> [Int]? {
        if p.isEmpty { return [] }
        var out: [Int] = []
        for g in p.split(separator: ":", omittingEmptySubsequences: false) {
            if g.isEmpty || g.count > 4 { return nil }
            var n = 0
            for c in g.unicodeScalars {
                let v: Int
                switch c {
                case "0"..."9": v = Int(c.value) - 48
                case "a"..."f": v = Int(c.value) - 97 + 10
                case "A"..."F": v = Int(c.value) - 65 + 10
                default: return nil
                }
                n = n * 16 + v
            }
            out.append(n)
        }
        return out
    }

    private static func parseIPv6First(_ s: String) -> Int? {
        if let r = s.range(of: "::") {
            let rest = s[r.upperBound...]
            if rest.contains("::") { return nil }
            guard let l = groups(s[..<r.lowerBound]), let rr = groups(rest), l.count + rr.count <= 7 else { return nil }
            return l.first ?? 0
        }
        guard let g = groups(s[...]), g.count == 8 else { return nil }
        return g[0]
    }
}

/// One problem report; every field is redacted when used (report.go Report).
public struct ProblemReport: Equatable {
    public var component: String
    public var version: String
    public var device: String
    public var what: String
    public var steps: String
    public var logs: String
    public var crash: String

    public init(component: String = "", version: String = "", device: String = "", what: String = "", steps: String = "", logs: String = "", crash: String = "") {
        self.component = component; self.version = version; self.device = device
        self.what = what; self.steps = steps; self.logs = logs; self.crash = crash
    }

    private static func trimSpace(_ s: String) -> String { s.trimmingCharacters(in: .whitespacesAndNewlines) }
    private static func trimNewlines(_ s: String) -> String {
        var t = Substring(s)
        while t.hasSuffix("\n") { t = t.dropLast() }
        return String(t)
    }

    private func redacted() -> ProblemReport {
        ProblemReport(
            component: Redactor.redact(Self.trimSpace(component)),
            version: Redactor.redact(Self.trimSpace(version)),
            device: Redactor.redact(Self.trimSpace(device)),
            what: Redactor.redact(Self.trimSpace(what)),
            steps: Redactor.redact(Self.trimSpace(steps)),
            logs: Redactor.redact(Self.trimNewlines(logs)),
            crash: Redactor.redact(Self.trimNewlines(crash)))
    }

    private var logsField: String {
        if crash.isEmpty { return logs }
        if logs.isEmpty { return "--- crash ---\n" + crash }
        return "--- crash ---\n" + crash + "\n--- logs ---\n" + logs
    }

    /// The whole report as text ("Copy full report", "Save report file").
    public func full() -> String {
        let r = redacted()
        var b = ""
        func sec(_ h: String, _ v: String) { b += "### " + h + "\n\n" + (v.isEmpty ? "(not given)" : v) + "\n\n" }
        sec("Where does it happen?", r.component)
        sec("What happened, and what did you expect?", r.what)
        sec("Steps to reproduce", r.steps)
        sec("tailbond version", r.version)
        sec("Device and firmware / OS", r.device)
        b += "### Logs\n\n```text\n" + r.logsField + "\n```\n"
        return b
    }

    /// The prefilled bug_report.yml issue, at most maxURL long (oldest log lines go first).
    public func issueURL() -> String {
        let r = redacted()
        var lines: [String] = r.logs.isEmpty ? [] : r.logs.components(separatedBy: "\n")
        var dropped = 0
        while true {
            var rr = r
            rr.logs = lines.joined(separator: "\n")
            if dropped > 0 { rr.logs = "(\(dropped) earlier lines left out: the full report has them)\n" + rr.logs }
            let u = rr.url()
            if u.utf8.count <= ReportText.maxURL || lines.isEmpty {
                if u.utf8.count > ReportText.maxURL {
                    rr.crash = ""
                    rr.logs = "(the crash report is too long for a link: paste the full report here)"
                    return rr.url()
                }
                return u
            }
            let n = max(1, lines.count / 10)
            lines.removeFirst(n)
            dropped += n
        }
    }

    private func url() -> String {
        let title = "bug: " + Self.firstLine(what, 60)
        let q = [
            "template=bug_report.yml",
            "title=" + Self.escape(title),
            "component=" + Self.escape(component),
            "what=" + Self.escape(what),
            "steps=" + Self.escape(steps),
            "version=" + Self.escape(version),
            "device=" + Self.escape(device),
            "logs=" + Self.escape(logsField),
        ]
        return "https://github.com/\(ReportText.repo)/issues/new?" + q.joined(separator: "&")
    }

    /// Percent-encoding as report.go Escape: A-Z a-z 0-9 - . _ ~ kept, the rest %XX of UTF-8.
    public static func escape(_ s: String) -> String {
        let hex = Array("0123456789ABCDEF".utf8)
        var out: [UInt8] = []
        for c in s.utf8 {
            switch c {
            case UInt8(ascii: "A")...UInt8(ascii: "Z"), UInt8(ascii: "a")...UInt8(ascii: "z"), UInt8(ascii: "0")...UInt8(ascii: "9"),
                 UInt8(ascii: "-"), UInt8(ascii: "."), UInt8(ascii: "_"), UInt8(ascii: "~"):
                out.append(c)
            default:
                out.append(UInt8(ascii: "%")); out.append(hex[Int(c >> 4)]); out.append(hex[Int(c & 15)])
            }
        }
        return String(decoding: out, as: UTF8.self)
    }

    static func firstLine(_ s: String, _ n: Int) -> String {
        let line = s.components(separatedBy: "\n").first ?? ""
        let scalars = Array(trimSpace(line).unicodeScalars)
        if scalars.count > n {
            var v = String.UnicodeScalarView()
            v.append(contentsOf: scalars[0..<(n - 1)])
            return String(v) + "…"
        }
        return String(String.UnicodeScalarView(scalars))
    }

    /// "Save report file"'s name, from an RFC 3339 UTC time.
    public static func fileName(_ at: String) -> String {
        let t = at.replacingOccurrences(of: ":", with: "").replacingOccurrences(of: "-", with: "")
            .replacingOccurrences(of: "T", with: "-").replacingOccurrences(of: "Z", with: "")
        return t.isEmpty ? "tailbond-report.txt" : "tailbond-report-\(t).txt"
    }
}
