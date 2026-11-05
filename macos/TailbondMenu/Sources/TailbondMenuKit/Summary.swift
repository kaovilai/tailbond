import Foundation

/// What the menu says, from a status (pure, unit tested). The words match
/// the dashboard's where both say the same thing (agent/web/app.js).
public struct MenuSummary: Equatable {
    public enum Tone: Equatable { case good, busy, off, bad }

    public struct Link: Equatable, Identifiable {
        public var id: String { name }
        public let name: String
        public let detail: String
        public let tone: Tone
    }

    public struct Line: Equatable, Identifiable {
        public var id: String { text }
        public let text: String
        public let warn: Bool
    }

    public let title: String
    public let subtitle: String
    public let tone: Tone
    /// SF Symbol for the menu bar
    public let symbol: String
    public let rates: String
    public let links: [Link]
    /// "Next to other VPNs": what tailbond found and does (empty: nothing to say)
    public let coexist: [Line]
    /// Offer the Work VPN traffic choice (another VPN with its own network is up)
    public let workVPNChoice: Bool
    public let workVPN: String
    /// what Direct and Bonded each mean here (shown under the switch)
    public let workExplain: [String]
    /// Bonded can't work here: why (the switch is replaced by this, Direct stays)
    public let bondedBlocked: String
    public let update: Line
    public let updatePolicy: String
    /// an update is ready and this install can install it on request (Install now)
    public let canInstall: Bool
    public let crash: String?
    public let connected: Bool
    /// "Share this Mac's connection" (nil: the agent doesn't offer it)
    public let share: Share?

    public struct Share: Equatable {
        /// the switch's state
        public let on: Bool
        /// a connection is picked, so the switch works; otherwise the
        /// dashboard's choices open
        public let canToggle: Bool
        public let text: String
        public let warn: Bool
    }

    public struct Choice: Hashable {
        public let wire: String
        public let label: String
        init(_ wire: String, _ label: String) { self.wire = wire; self.label = label }
    }

    public static let modes: [Choice] = [
        Choice("speed", "Fastest"), Choice("streaming", "Smooth calls and video"), Choice("redundant", "Most reliable"), Choice("failover", "Save mobile data"),
    ]
    public static let policies: [Choice] = [
        Choice("immediate", "Right away"), Choice("auto", "Automatic (night)"), Choice("notify", "Tell me"), Choice("off", "Off"),
    ]

    public static func policyLabel(_ p: String) -> String {
        policies.first { $0.wire == p }?.label ?? (p.isEmpty ? "Tell me" : p)
    }

    public static func kindLabel(_ kind: String) -> String {
        switch kind {
        case "wifi": return "Wi-Fi"
        case "ethernet": return "Ethernet"
        case "usb": return "iPhone or phone (USB)"
        case "cellular": return "Mobile data"
        case "shared-cellular": return "Shared by a nearby device"
        default: return "Connection"
        }
    }

    public static func rate(_ bps: Int64) -> String {
        let b = Double(bps)
        if b >= 1_000_000 { return String(format: "%.1f Mbps", b / 1_000_000) }
        if b >= 1_000 { return String(format: "%.0f kbps", b / 1_000) }
        return "0 kbps"
    }

    /// nil status: the agent didn't answer.
    public init(_ s: AgentStatus?, unreachable: String? = nil) {
        guard let s else {
            title = "tailbond isn't running"
            subtitle = unreachable ?? "Nothing answers on 127.0.0.1:8077. Start it: sudo launchctl kickstart -k system/com.tailbond.tailbondd"
            tone = .bad; symbol = "exclamationmark.triangle"; rates = ""; links = []; coexist = []
            workVPNChoice = false; workVPN = ""; workExplain = []; bondedBlocked = ""; update = Line(text: "", warn: false); updatePolicy = ""
            canInstall = false; crash = nil; connected = false; share = nil
            return
        }
        let server = s.activeServer.isEmpty ? s.server : s.activeServer
        let short = server.split(separator: ":").first.map { String($0.split(separator: ".").first ?? "") } ?? ""
        let active = s.paths.filter { $0.state == "active" }.count
        connected = s.state == "connected"
        switch s.state {
        case "connected":
            title = "Connected"
            let n = s.paths.count
            let used = active == n ? (active == 1 ? "1 link" : "\(active) links") : "\(active) of \(n) links in use"
            subtitle = (short.isEmpty ? "" : "to \(short) · ") + used
            tone = .good; symbol = "point.3.filled.connected.trianglepath.dotted"
        case "connecting":
            title = "Connecting…"
            subtitle = short.isEmpty ? "" : "to \(short)"
            tone = .busy; symbol = "arrow.triangle.2.circlepath"
        case "error":
            title = "Can't connect"
            subtitle = s.error.isEmpty ? "See the dashboard's Logs tab" : s.error
            tone = .bad; symbol = "exclamationmark.triangle"
        default:
            title = s.paired ? "Off" : "Not paired"
            subtitle = s.paired ? "Turn it on to bond your connections" : "Open the dashboard → Setup to pair with your home server"
            tone = .off; symbol = "point.3.connected.trianglepath.dotted"
        }
        rates = connected ? "↓ \(Self.rate(s.totals.rxBps))  ↑ \(Self.rate(s.totals.txBps))" : ""
        links = s.paths.map { p in
            var name = p.label.isEmpty ? Self.kindLabel(p.kind) : p.label
            // "Ethernet from tiger-mbp", "Mobile data from sam-pixel" say what they are
            if p.kind == "shared-cellular" && !p.label.isEmpty && !p.label.contains(" from ") { name += ", mobile data it shares" }
            var detail: String
            var tone: Tone
            if !p.enabled {
                detail = "off"; tone = .off
            } else {
                switch p.state {
                case "active":
                    detail = "\(p.srttMs) ms · ↓ \(Self.rate(p.rxBps))"; tone = .good
                case "standby": detail = "standby"; tone = .off
                case "failed": detail = p.captive == "portal" ? "Wi-Fi sign-in needed" : (short.isEmpty ? "can't reach your server" : "can't reach \(short)"); tone = .bad
                default: detail = "starting"; tone = .busy
                }
            }
            return Link(name: p.iface.isEmpty ? name : "\(name) (\(p.iface))", detail: detail, tone: tone)
        }

        // next to other VPNs (docs/macos.md "Using tailbond with NetBird")
        var cx: [Line] = []
        var choice = false
        let work = s.coexist?.workVPNEffective.isEmpty == false ? s.coexist!.workVPNEffective : (s.coexist?.workVPN ?? "")
        let workWord = work == "bonded" ? "Bonded (through tailbond)" : "Direct (not through tailbond)"
        if let c = s.coexist {
            if let nb = c.netbird, nb.cli {
                if nb.loggedIn {
                    cx.append(Line(text: "NetBird is on\(nb.addr.isEmpty ? "" : " (\(nb.addr))"): its traffic goes \(workWord)", warn: false))
                    choice = true
                } else if nb.running {
                    cx.append(Line(text: "NetBird is installed but not logged in", warn: false))
                }
            }
            for f in c.foreign where f.kind != "netbird" && f.kind != "tailscale" {
                cx.append(Line(text: "Another VPN is on (\(f.name), \(f.iface)): its traffic goes \(workWord)", warn: false))
                choice = true
            }
            if c.tailnetViaTailscaleApp {
                cx.append(Line(text: "The Tailscale app is on: your tailnet goes through it, not tailbond", warn: false))
            }
            if c.splitDNS && !c.dnsDomains.isEmpty {
                cx.append(Line(text: "Work names (\(c.dnsDomains.prefix(3).joined(separator: ", "))) use the work VPN's DNS", warn: false))
            }
            for w in c.warnings + c.conflicts { cx.append(Line(text: w, warn: true)) }
            if !c.error.isEmpty { cx.append(Line(text: c.error, warn: true)) }
        }
        coexist = cx
        workVPNChoice = choice
        workVPN = work.isEmpty ? "direct" : work
        var nets: [String] = []
        for f in s.coexist?.foreign ?? [] where f.kind != "tailscale" {
            for p in f.prefixes where !nets.contains(p) { nets.append(p) }
        }
        let netWords = nets.isEmpty ? "" : " and its networks (\(nets.prefix(3).joined(separator: ", "))\(nets.count > 3 ? ", …" : ""))"
        let home = short.isEmpty ? "your home server" : short
        workExplain = choice ? [
            "Direct: the work VPN\(netWords), and its own connections to its servers and peers, use this Mac's own internet, as without tailbond.",
            "Bonded: they go through tailbond to \(home), then out to your company: steadier, but through your home.",
        ] : []
        bondedBlocked = s.coexist?.bondedBlocked ?? ""

        // updates (docs/updates.md)
        let u = s.update ?? UpdateInfo()
        updatePolicy = u.policy.isEmpty ? "notify" : u.policy
        let cur = u.current.isEmpty ? s.version : u.current
        let available = !u.available.isEmpty && u.state != "installing"
        canInstall = available && u.method == "static"
        switch true {
        case u.state == "installing":
            update = Line(text: "Installing tailbond \(u.available)…", warn: false)
        case u.state == "failed":
            update = Line(text: "An update didn't install; still on \(cur)", warn: true)
        case available:
            var t = "Update available: tailbond \(u.available)\(u.security ? ", a security fix" : "") (you have \(cur))."
            if u.method == "brew" {
                t += " Update with Homebrew (the dashboard has the commands)."
            } else if updatePolicy == "immediate" && !u.waiting.isEmpty {
                t += " It installs by itself as soon as no call or big download is running (within 30 minutes); tailbond restarts, a few seconds without the tunnel."
            } else if updatePolicy == "auto" {
                t += " It installs by itself tonight (03:00-05:00)."
            }
            update = Line(text: t, warn: u.security)
        case updatePolicy == "off":
            update = Line(text: "tailbond \(cur): automatic checks are off", warn: false)
        default:
            update = Line(text: "tailbond \(cur) is up to date", warn: false)
        }
        crash = s.crash.isEmpty ? nil : Self.when(s.crash)
        share = s.sharing.map(Self.shareSummary)
    }

    /// "1.2 MB", "120 MB", "3.4 GB" (the dashboard's units)
    public static func bytes(_ b: UInt64) -> String {
        var v = Double(b)
        var i = 0
        let units = ["B", "KB", "MB", "GB", "TB"]
        while v >= 1024 && i < units.count - 1 { v /= 1024; i += 1 }
        if i == 0 { return "\(b) B" }
        return String(format: v >= 100 ? "%.0f" : "%.1f", v) + " " + units[i]
    }

    static func shareSummary(_ sh: SharingInfo) -> Share {
        let i = sh.ifaces.first { $0.name == sh.iface }
        let word = i.map { $0.label.isEmpty ? kindLabel($0.kind) : $0.label } ?? ""
        let name = sh.iface.isEmpty ? "" : (word.isEmpty ? sh.iface : "\(word) (\(sh.iface))")
        if sh.iface.isEmpty {
            return Share(on: false, canToggle: false, text: "Lend this Mac's other connection (the hotel's Ethernet, a USB modem) to your travel router: choose it in the dashboard.", warn: false)
        }
        if !sh.enabled {
            return Share(on: false, canToggle: true, text: "Off. Turn it on to lend \(name) to nearby hubs.", warn: false)
        }
        switch sh.state {
        case "sharing":
            let total = sh.hubs.reduce(UInt64(0)) { $0 + $1.upBytes + $1.downBytes }
            var t = sh.hubs.isEmpty ? "Sharing \(name): no hub is using it yet" :
                "Sharing \(name) with \(sh.hubs.map(\.hub).joined(separator: " and ")): \(bytes(total))"
            if !sh.note.isEmpty { t += ". " + sh.note }
            return Share(on: true, canToggle: true, text: t, warn: !sh.note.isEmpty)
        case "no_key", "error":
            return Share(on: true, canToggle: true, text: sh.error.isEmpty ? "Not sharing \(name)" : sh.error, warn: true)
        default:
            return Share(on: true, canToggle: true, text: "Starting to share \(name)…", warn: false)
        }
    }

    /// An RFC 3339 time as the Mac shows dates ("Nov 4, 2026 at 6:51 AM").
    public static func when(_ rfc3339: String, timeZone: TimeZone = .current, locale: Locale = .current) -> String {
        guard let d = ISO8601DateFormatter().date(from: rfc3339) else { return rfc3339 }
        let f = DateFormatter()
        f.dateStyle = .medium
        f.timeStyle = .short
        f.timeZone = timeZone
        f.locale = locale
        return f.string(from: d)
    }

    /// --status: one block of text for a terminal (and the CI check).
    public var text: String {
        var out = ["tailbond: \(title)" + (subtitle.isEmpty ? "" : " (\(subtitle))")]
        if !rates.isEmpty { out.append("  \(rates)") }
        for l in links { out.append("  \(l.name): \(l.detail)") }
        for l in coexist { out.append("  \(l.warn ? "! " : "")\(l.text)") }
        if !update.text.isEmpty { out.append("  \(update.text)") }
        if !updatePolicy.isEmpty { out.append("  Install updates: \(Self.policyLabel(updatePolicy))") }
        if let sh = share { out.append("  \(sh.warn ? "! " : "")Share this Mac's connection: \(sh.on ? "on" : "off"): \(sh.text)") }
        if let c = crash { out.append("  ! tailbond stopped unexpectedly (\(c)): Report a problem…") }
        return out.joined(separator: "\n")
    }
}
