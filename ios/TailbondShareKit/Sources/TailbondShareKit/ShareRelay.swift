#if canImport(Network)
import Foundation
import Network

/// Runs `SharerCore` over Network.framework: the LAN listener on Wi-Fi
/// (advertised over Bonjour as `_tailbond-share._udp`), one cellular
/// connection to the home server per hub. iOS has no VPN here (free signing
/// can't use Network Extension), so nothing can loop through a tunnel.
public final class ShareRelay {
    public struct Status: Equatable {
        public var listening = false
        public var cellularReady = false
        public var error: String?
        public var hubs: [SharerCore.HubUse] = []
        public var used: UInt64 = 0
        public init() {}
    }

    private let queue = DispatchQueue(label: "tailbond.share")
    private let core: SharerCore
    private let host: NWEndpoint.Host
    private let port: NWEndpoint.Port
    private var listener: NWListener?
    private var hubs: [String: NWConnection] = [:]
    /// When each LAN peer last sent anything: peers with no session or
    /// handshake are forgotten after a while, and there are at most maxPeers.
    private var hubSeen: [String: Date] = [:]
    private static let maxPeers = 64
    private static let peerIdle: TimeInterval = 15
    private var upstreams: [[UInt8]: NWConnection] = [:]
    private var timer: DispatchSourceTimer?
    private var status = Status()
    private var cellularMonitor: NWPathMonitor?
    /// Called on the relay's queue when something changed.
    public var onChange: ((Status) -> Void)?

    public init(core: SharerCore, serverHost: String, serverPort: UInt16) {
        self.core = core
        host = NWEndpoint.Host(serverHost)
        port = NWEndpoint.Port(rawValue: serverPort) ?? 443
    }

    public func start() throws {
        let params = NWParameters.udp
        params.requiredInterfaceType = .wifi
        params.allowLocalEndpointReuse = true
        let l = try NWListener(using: params, on: NWEndpoint.Port(rawValue: ShareProtocol.defaultPort)!)
        let cfg = core.config
        l.service = NWListener.Service(
            name: cfg.label, type: ShareProtocol.serviceType, domain: nil,
            txtRecord: NWTXTRecord(["v": "1", "id": cfg.id, "k": ShareProtocol.keyID(cfg.key), "n": cfg.label])
        )
        l.stateUpdateHandler = { [weak self] st in
            guard let self else { return }
            switch st {
            case .ready: self.status.listening = true; self.status.error = nil
            case let .failed(e): self.status.listening = false; self.status.error = "\(e)"
            case .cancelled: self.status.listening = false
            default: break
            }
            self.changed()
        }
        l.newConnectionHandler = { [weak self] c in self?.accept(c) }
        l.start(queue: queue)
        listener = l

        let m = NWPathMonitor(requiredInterfaceType: .cellular)
        m.pathUpdateHandler = { [weak self] p in
            self?.status.cellularReady = p.status == .satisfied
            self?.changed()
        }
        m.start(queue: queue)
        cellularMonitor = m

        let t = DispatchSource.makeTimerSource(queue: queue)
        t.schedule(deadline: .now() + 2, repeating: 2)
        t.setEventHandler { [weak self] in
            guard let self else { return }
            self.run(self.core.tick())
            self.forgetIdlePeers(Date())
            self.changed() // counters: published here, not per datagram
        }
        t.resume()
        timer = t
    }

    /// Changes the cap and allowed hubs while sharing (they apply at once:
    /// hubs no longer allowed are dropped, a raised cap lets hubs back in).
    public func updateLimits(capBytes: UInt64, allowedHubs: Set<String>) {
        queue.async { [weak self] in
            guard let self else { return }
            self.run(self.core.updateLimits(capBytes: capBytes, allowedHubs: allowedHubs))
            self.changed()
        }
    }

    /// A new month: the cap counts from zero again.
    public func resetUsage() {
        queue.async { [weak self] in
            guard let self else { return }
            self.core.resetUsage()
            self.changed()
        }
    }

    /// Ends every session (telling the hubs why) and closes everything.
    public func stop(_ reason: ShareProtocol.Reason = .paused) {
        queue.sync {
            run(core.stopAll(reason))
            timer?.cancel()
            timer = nil
            cellularMonitor?.cancel()
            listener?.cancel()
            listener = nil
            hubs.values.forEach { $0.cancel() }
            hubs = [:]
            hubSeen = [:]
            upstreams.values.forEach { $0.cancel() }
            upstreams = [:]
        }
    }

    private func key(_ c: NWConnection) -> String { "\(c.endpoint)" }

    private func accept(_ c: NWConnection) {
        let now = Date()
        if hubs.count >= Self.maxPeers { forgetIdlePeers(now, all: true) }
        guard hubs.count < Self.maxPeers else {
            c.cancel()
            return
        }
        hubs[key(c)] = c
        hubSeen[key(c)] = now
        c.start(queue: queue)
        receiveHub(c)
    }

    private func receiveHub(_ c: NWConnection) {
        c.receiveMessage { [weak self] data, _, _, err in
            guard let self else { return }
            let k = self.key(c)
            if let data, !data.isEmpty {
                self.hubSeen[k] = Date()
                self.run(self.core.handle([UInt8](data), from: k))
            }
            guard self.hubs[k] === c else { return } // forgotten, maybe replaced
            if err == nil { self.receiveHub(c) } else { self.forget(k) }
        }
    }

    private func forget(_ k: String) {
        hubs.removeValue(forKey: k)?.cancel()
        hubSeen.removeValue(forKey: k)
    }

    /// Cancels LAN peers with no session or handshake: quiet for a while,
    /// or (all) any of them, to make room.
    private func forgetIdlePeers(_ now: Date, all: Bool = false) {
        for (k, seen) in hubSeen where !core.knows(k) && (all || now.timeIntervalSince(seen) > Self.peerIdle) {
            forget(k)
        }
    }

    private func openUpstream(_ session: [UInt8]) {
        let p = NWParameters.udp
        p.requiredInterfaceType = .cellular // mobile data only, never the Wi-Fi it's shared on
        let c = NWConnection(host: host, port: port, using: p)
        c.stateUpdateHandler = { [weak self] st in
            if case .failed = st, let self { self.run(self.core.upstreamFailed(session: session)) }
        }
        upstreams[session] = c
        c.start(queue: queue)
        receiveUpstream(c, session)
    }

    private func receiveUpstream(_ c: NWConnection, _ session: [UInt8]) {
        c.receiveMessage { [weak self] data, _, _, err in
            guard let self else { return }
            if let data, !data.isEmpty { self.run(self.core.fromUpstream(session: session, [UInt8](data))) }
            if err == nil, self.upstreams[session] != nil { self.receiveUpstream(c, session) }
        }
    }

    private func run(_ actions: [SharerCore.Action]) {
        var hubsChanged = false
        for a in actions {
            switch a {
            case let .toHub(hub, bytes):
                hubs[hub]?.send(content: Data(bytes), completion: .contentProcessed { _ in })
            case let .openUpstream(session):
                hubsChanged = true
                openUpstream(session)
            case let .toUpstream(session, bytes):
                upstreams[session]?.send(content: Data(bytes), completion: .contentProcessed { _ in })
            case let .closeUpstream(session):
                hubsChanged = true
                upstreams.removeValue(forKey: session)?.cancel()
            }
        }
        // a hub came or went: say so now; byte counters wait for the timer
        if hubsChanged { changed() }
    }

    private func changed() {
        status.hubs = core.hubs
        status.used = core.used
        onChange?(status)
    }
}
#endif
