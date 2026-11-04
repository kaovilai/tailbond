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
    public let update: Line
    public let updatePolicy: String
    public let crash: String?
    public let connected: Bool

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
        case "shared-cellular": return "Shared by a nearby phone"
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
            workVPNChoice = false; workVPN = ""; update = Line(text: "", warn: false); updatePolicy = ""; crash = nil; connected = false
            return
        }
        let server = s.activeServer.isEmpty ? s.server : s.activeServer
        let short = server.split(separator: ":").first.map { String($0.split(separator: ".").first ?? "") } ?? ""
        let active = s.paths.filter { $0.state == "active" }.count
        connected = s.state == "connected"
        switch s.state {
        case "connected":
            title = "Connected"
            subtitle = (short.isEmpty ? "" : "to \(short) · ") + (active == 1 ? "1 link" : "\(active) links")
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
            let name = p.label.isEmpty ? Self.kindLabel(p.kind) : p.label
            var detail: String
            var tone: Tone
            if !p.enabled {
                detail = "off"; tone = .off
            } else {
                switch p.state {
                case "active":
                    detail = "\(p.srttMs) ms · ↓ \(Self.rate(p.rxBps))"; tone = .good
                case "standby": detail = "standby"; tone = .off
                case "failed": detail = p.captive == "portal" ? "Wi-Fi sign-in needed" : "not working"; tone = .bad
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

        // updates (docs/updates.md)
        let u = s.update ?? UpdateInfo()
        updatePolicy = u.policy.isEmpty ? "notify" : u.policy
        let pol = Self.policyLabel(updatePolicy)
        let cur = u.current.isEmpty ? s.version : u.current
        switch true {
        case u.state == "installing":
            update = Line(text: "Installing tailbond \(u.available)…", warn: false)
        case u.state == "failed":
            update = Line(text: "An update didn't install; still on \(cur)", warn: true)
        case !u.available.isEmpty && updatePolicy != "off":
            let why = u.waiting.isEmpty ? "" : " (\(u.waiting))"
            update = Line(text: "Update available: \(u.available)\(u.security ? " (security)" : "")\(why) · \(pol)", warn: u.security)
        case updatePolicy == "off":
            update = Line(text: "tailbond \(cur) · updates off", warn: false)
        default:
            update = Line(text: "tailbond \(cur) · up to date · \(pol)", warn: false)
        }
        crash = s.crash.isEmpty ? nil : s.crash
    }

    /// --status: one block of text for a terminal (and the CI check).
    public var text: String {
        var out = ["tailbond: \(title)" + (subtitle.isEmpty ? "" : " (\(subtitle))")]
        if !rates.isEmpty { out.append("  \(rates)") }
        for l in links { out.append("  \(l.name): \(l.detail)") }
        for l in coexist { out.append("  \(l.warn ? "! " : "")\(l.text)") }
        if !update.text.isEmpty { out.append("  \(update.text)") }
        if let c = crash { out.append("  ! tailbond stopped unexpectedly (\(c)): Report a problem…") }
        return out.joined(separator: "\n")
    }
}
