import Foundation

/// The sharer's state machine, without sockets: frames in, actions out.
/// `ShareRelay` runs it over Network.framework; tests drive it directly.
/// Same behaviour as the Go reference (agent/internal/share/sharer.go).
public final class SharerCore {
    public typealias Hub = String // the hub's endpoint, as the transport names it

    public struct Config {
        public var id: String
        public var label: String
        public var key: [UInt8]
        public var capBytes: UInt64 // 0 = no cap
        public var usedBytes: UInt64
        public var allowedHubs: Set<String>
        public init(id: String, label: String, key: [UInt8], capBytes: UInt64 = 0, usedBytes: UInt64 = 0, allowedHubs: Set<String> = []) {
            self.id = id
            self.label = label
            self.key = key
            self.capBytes = capBytes
            self.usedBytes = usedBytes
            self.allowedHubs = allowedHubs
        }
    }

    /// May sharing go on: a reason (.none = yes), battery percent (nil unknown), charging.
    public struct Check {
        public var reason: ShareProtocol.Reason
        public var battery: Int?
        public var charging: Bool
        public init(reason: ShareProtocol.Reason = .none, battery: Int? = nil, charging: Bool = false) {
            self.reason = reason
            self.battery = battery
            self.charging = charging
        }
    }

    public enum Action: Equatable {
        case toHub(Hub, [UInt8])
        /// Open the cellular upstream to the home server for this session.
        case openUpstream(session: [UInt8])
        case toUpstream(session: [UInt8], [UInt8])
        case closeUpstream(session: [UInt8])
    }

    public struct HubUse: Equatable {
        public var hub: String
        public var endpoint: Hub
        public var upBytes: UInt64
        public var downBytes: UInt64
        public var since: Date
    }

    struct Pending { var sharerNonce: [UInt8]; var hubID: String; var from: Hub; var at: Date }
    final class Session {
        let id: [UInt8], hubID: String, hub: Hub, since: Date
        var up: UInt64 = 0, down: UInt64 = 0
        var lastSeen: Date
        init(id: [UInt8], hubID: String, hub: Hub, now: Date) {
            self.id = id
            self.hubID = hubID
            self.hub = hub
            since = now
            lastSeen = now
        }
    }

    public var config: Config
    public var check: () -> Check
    private var pending: [[UInt8]: Pending] = [:]
    private var sessions: [[UInt8]: Session] = [:]
    public private(set) var used: UInt64

    public init(config: Config, check: @escaping () -> Check = { Check() }) {
        self.config = config
        self.check = check
        used = config.usedBytes
    }

    public var hubs: [HubUse] {
        sessions.values.map { HubUse(hub: $0.hubID, endpoint: $0.hub, upBytes: $0.up, downBytes: $0.down, since: $0.since) }
            .sorted { $0.since < $1.since }
    }

    var capLeft: UInt64 {
        if config.capBytes == 0 { return ShareProtocol.noCap }
        return used >= config.capBytes ? 0 : config.capBytes - used
    }

    static func random(_ n: Int) -> [UInt8] {
        var g = SystemRandomNumberGenerator()
        return (0..<n).map { _ in UInt8.random(in: 0...255, using: &g) }
    }

    /// A frame from a hub on the LAN.
    public func handle(_ b: [UInt8], from: Hub, now: Date = Date()) -> [Action] {
        guard let t = b.first else { return [] }
        let P = ShareProtocol.self
        switch t {
        case P.data:
            guard let s = session(b, from) else { return [] }
            if capLeft == 0 { return end(s, .cap, bye: true) }
            let p = Array(b[P.dataHeader...])
            s.up += UInt64(p.count)
            used += UInt64(p.count)
            s.lastSeen = now
            return [.toUpstream(session: s.id, p)]
        case P.keepalive:
            guard let s = session(b, from), b.count >= 1 + P.sessionLen + 4 else { return [] }
            s.lastSeen = now
            let c = check()
            var reason = c.reason
            if reason == .none, capLeft == 0 { reason = .cap }
            if reason != .none { return end(s, reason, bye: true) }
            let battery = c.battery.map { UInt8(clamping: max(0, min(100, $0))) } ?? 255
            let seq = P.read32(b, 1 + P.sessionLen)
            return [.toHub(from, P.kaAckFrame(session: s.id, seq: seq, up: s.up, down: s.down, capLeft: capLeft,
                                               battery: battery, flags: c.charging ? P.flagCharging : 0))]
        case P.bye:
            guard let s = session(b, from) else { return [] }
            return end(s, .none, bye: false)
        case P.hello:
            guard let h = P.parseHello(b), pending.count < 64 else { return [] }
            let sn = Self.random(P.nonceLen)
            pending[h.hubNonce] = Pending(sharerNonce: sn, hubID: h.hubID, from: from, at: now)
            return [.toHub(from, P.challengeFrame(hubNonce: h.hubNonce, sharerNonce: sn, sharerID: config.id, label: config.label))]
        case P.auth:
            guard let a = P.parseAuth(b), let p = pending.removeValue(forKey: a.hubNonce) else { return [] }
            // a wrong key, a replay or a stranger gets no answer at all
            guard p.sharerNonce == a.sharerNonce, p.from == from,
                  P.macEqual(a.mac, P.authMAC(key: config.key, hubNonce: a.hubNonce, sharerNonce: a.sharerNonce, hubID: p.hubID))
            else { return [] }
            var reason = check().reason
            if reason == .none, !config.allowedHubs.isEmpty, !config.allowedHubs.contains(p.hubID) { reason = .notAllowed }
            if reason == .none, capLeft == 0 { reason = .cap }
            if reason != .none {
                return [.toHub(from, P.refuseFrame(hubNonce: a.hubNonce, reason: reason,
                                                    mac: P.refuseMAC(key: config.key, hubNonce: a.hubNonce, sharerNonce: a.sharerNonce, reason: reason)))]
            }
            var out: [Action] = []
            if let old = sessions.values.first(where: { $0.hubID == p.hubID }) {
                out += end(old, .replaced, bye: true)
            } else if sessions.count >= 8 {
                return []
            }
            let s = Session(id: Self.random(P.sessionLen), hubID: p.hubID, hub: from, now: now)
            sessions[s.id] = s
            let cap = capLeft
            out.append(.openUpstream(session: s.id))
            out.append(.toHub(from, P.welcomeFrame(hubNonce: a.hubNonce, session: s.id, capLeft: cap,
                                                   mac: P.welcomeMAC(key: config.key, hubNonce: a.hubNonce, sharerNonce: a.sharerNonce, session: s.id, capLeft: cap))))
            return out
        default:
            return []
        }
    }

    /// A datagram from the home server for a session.
    public func fromUpstream(session id: [UInt8], _ payload: [UInt8]) -> [Action] {
        guard let s = sessions[id] else { return [] }
        s.down += UInt64(payload.count)
        used += UInt64(payload.count)
        return [.toHub(s.hub, ShareProtocol.dataFrame(session: s.id, payload: payload))]
    }

    /// The upstream failed (no cellular).
    public func upstreamFailed(session id: [UInt8]) -> [Action] {
        guard let s = sessions[id] else { return [] }
        return end(s, .noCellular, bye: true)
    }

    /// Every couple of seconds: forget stale handshakes, idle hubs, and stop on battery/cap.
    public func tick(now: Date = Date()) -> [Action] {
        pending = pending.filter { now.timeIntervalSince($0.value.at) <= 10 }
        var out: [Action] = []
        for s in sessions.values where now.timeIntervalSince(s.lastSeen) > 30 {
            out += end(s, .idle, bye: true)
        }
        let r = check().reason
        if r != .none { out += stopAll(r) } else if capLeft == 0 { out += stopAll(.cap) }
        return out
    }

    /// Ends every session, telling the hubs why.
    public func stopAll(_ reason: ShareProtocol.Reason) -> [Action] {
        sessions.values.flatMap { end($0, reason, bye: true) }
    }

    private func session(_ b: [UInt8], _ from: Hub) -> Session? {
        guard let id = ShareProtocol.session(of: b), let s = sessions[id], s.hub == from else { return nil }
        return s
    }

    private func end(_ s: Session, _ reason: ShareProtocol.Reason, bye: Bool) -> [Action] {
        guard sessions.removeValue(forKey: s.id) != nil else { return [] }
        var out: [Action] = [.closeUpstream(session: s.id)]
        if bye { out.append(.toHub(s.hub, ShareProtocol.byeFrame(session: s.id, reason: reason))) }
        return out
    }
}
