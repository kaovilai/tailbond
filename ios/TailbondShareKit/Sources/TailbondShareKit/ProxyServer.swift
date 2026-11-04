#if canImport(Network)
import Foundation
import Network

/// Opens the proxy's upstream connections.
public protocol ProxyDialer {
    /// Calls `done` on `queue` with a ready connection, or nil.
    func dial(host: String, port: UInt16, queue: DispatchQueue, done: @escaping (NWConnection?) -> Void)
}

/// Upstream straight out of an interface type (the iPhone: `.cellular`),
/// or the system's choice (nil). The bonded tunnel inside the app is not
/// built yet (docs/phone-hub.md, "iPhone"): this dialer is what the
/// experimental iPhone hub uses today.
public struct DirectDialer: ProxyDialer {
    public var interface: NWInterface.InterfaceType?
    public var timeout: TimeInterval = 20

    public init(interface: NWInterface.InterfaceType? = nil) { self.interface = interface }

    public func dial(host: String, port: UInt16, queue: DispatchQueue, done: @escaping (NWConnection?) -> Void) {
        guard let p = NWEndpoint.Port(rawValue: port) else { return queue.async { done(nil) } }
        let params = NWParameters.tcp
        if let i = interface { params.requiredInterfaceType = i }
        let c = NWConnection(host: NWEndpoint.Host(host), port: p, using: params)
        var fired = false
        func finish(_ ok: Bool) {
            guard !fired else { return }
            fired = true
            if !ok { c.cancel() }
            done(ok ? c : nil)
        }
        c.stateUpdateHandler = { st in
            switch st {
            case .ready: finish(true)
            case .failed, .cancelled: finish(false)
            default: break
            }
        }
        c.start(queue: queue)
        queue.asyncAfter(deadline: .now() + timeout) { finish(false) }
    }
}

/// The phone hub's proxy over Network.framework (docs/phone-hub.md): a
/// TCP listener (never on cellular), `ProxyCore.Handshake` per client,
/// then a byte relay to what the dialer opened. Bonjour:
/// `_tailbond-proxy._tcp` with the Pair & Share TXT keys.
public final class ProxyServer {
    public struct Client: Equatable {
        public var addr: String
        public var conns: Int
        public var up: UInt64
        public var down: UInt64
    }

    public struct Status: Equatable {
        public var listening = false
        public var port: UInt16 = 0
        public var error: String?
        public var up: UInt64 = 0
        public var down: UInt64 = 0
        public var clients: [Client] = []
        public init() {}
    }

    public struct Advert {
        public var name: String
        public var id: String
        public var keyID: String
        /// Whether the upstream is a bonded tunnel ("b" TXT key); the
        /// iPhone's isn't (mobile data only), and the Mac says so.
        public var bonded: Bool
        public init(name: String, id: String, keyID: String, bonded: Bool = false) {
            self.name = name
            self.id = id
            self.keyID = keyID
            self.bonded = bonded
        }
    }

    private let queue = DispatchQueue(label: "tailbond.proxy")
    private let auth: ((String, String) -> Bool)?
    private let port: NWEndpoint.Port
    private let dialer: ProxyDialer
    private let advert: Advert?
    private var listener: NWListener?
    private var open: [ObjectIdentifier: NWConnection] = [:]
    private var status = Status()
    private var clients: [String: Client] = [:]
    /// Called on the server's queue when something changed.
    public var onChange: ((Status) -> Void)?

    /// port 0: any (tests read `status.port`).
    /// auth: checks a login (ProxyCore.verifier); nil = no login asked (tests).
    public init(auth: ((String, String) -> Bool)?, port: UInt16 = ProxyCore.defaultPort, dialer: ProxyDialer, advert: Advert? = nil) {
        self.auth = auth
        self.port = port == 0 ? .any : (NWEndpoint.Port(rawValue: port) ?? .any)
        self.dialer = dialer
        self.advert = advert
    }

    public func start() throws {
        let params = NWParameters.tcp
        params.allowLocalEndpointReuse = true
        // hotspot, USB, Wi-Fi, loopback: never mobile data
        params.prohibitedInterfaceTypes = [.cellular]
        let l = try NWListener(using: params, on: port)
        if let a = advert {
            l.service = NWListener.Service(name: a.name, type: ProxyCore.serviceType, domain: nil,
                                           txtRecord: NWTXTRecord(["v": "1", "id": a.id, "k": a.keyID, "n": a.name, "b": a.bonded ? "1" : "0"]))
        }
        l.stateUpdateHandler = { [weak self] st in
            guard let self else { return }
            switch st {
            case .ready:
                self.status.listening = true
                self.status.port = l.port?.rawValue ?? 0
                self.status.error = nil
            case let .failed(e):
                self.status.listening = false
                self.status.error = "\(e)"
            case .cancelled:
                self.status.listening = false
            default: break
            }
            self.changed()
        }
        l.newConnectionHandler = { [weak self] c in self?.accept(c) }
        l.start(queue: queue)
        listener = l
    }

    public func stop() {
        queue.sync {
            listener?.cancel()
            listener = nil
            for c in open.values { c.cancel() }
            open.removeAll()
            status = Status()
            clients.removeAll()
        }
    }

    /// A snapshot (from any thread).
    public var current: Status { queue.sync { status } }

    private func changed() {
        status.clients = clients.values.sorted { $0.addr < $1.addr }
        onChange?(status)
    }

    private func track(_ c: NWConnection) { open[ObjectIdentifier(c)] = c }
    private func untrack(_ c: NWConnection) { open[ObjectIdentifier(c)] = nil }

    private func remoteHost(_ c: NWConnection) -> String? {
        if case let .hostPort(h, _) = c.endpoint {
            switch h {
            case let .ipv4(a): return "\(a)"
            case let .ipv6(a): return "\(a)"
            case let .name(n, _): return n
            @unknown default: return nil
            }
        }
        return nil
    }

    private func accept(_ c: NWConnection) {
        guard let host = remoteHost(c), ProxyCore.allowPrivate(host) else {
            c.cancel()
            return
        }
        track(c)
        c.start(queue: queue)
        var hs = ProxyCore.Handshake(auth: auth)
        func readMore() {
            c.receive(minimumIncompleteLength: 1, maximumLength: 16384) { [weak self] data, _, done, err in
                guard let self else { return }
                if let d = data, !d.isEmpty {
                    switch hs.feed(Array(d)) {
                    case .more: readMore()
                    case let .reply(b):
                        c.send(content: Data(b), completion: .contentProcessed { _ in })
                        readMore()
                    case let .refuse(b):
                        c.send(content: Data(b), completion: .contentProcessed { _ in self.close(c) })
                    case let .connect(h, p, ok, fail, rest):
                        self.connect(c, from: host, to: h, port: p, ok: ok, fail: fail, rest: rest)
                    }
                    return
                }
                if done || err != nil { self.close(c) }
                else { readMore() }
            }
        }
        readMore()
    }

    private func close(_ c: NWConnection) {
        c.cancel()
        untrack(c)
    }

    private func connect(_ c: NWConnection, from: String, to host: String, port: UInt16, ok: [UInt8], fail: [UInt8], rest: [UInt8]) {
        dialer.dial(host: host, port: port, queue: queue) { [weak self] up in
            guard let self else { return }
            guard let up else {
                c.send(content: Data(fail), completion: .contentProcessed { _ in self.close(c) })
                return
            }
            self.track(up)
            c.send(content: Data(ok), completion: .contentProcessed { _ in })
            var cl = self.clients[from] ?? Client(addr: from, conns: 0, up: 0, down: 0)
            cl.conns += 1
            self.clients[from] = cl
            self.changed()
            if !rest.isEmpty { self.count(from, up: UInt64(rest.count)); up.send(content: Data(rest), completion: .contentProcessed { _ in }) }
            var ended = 0
            let end = {
                ended += 1
                if ended == 2 {
                    self.close(c)
                    self.close(up)
                    self.clients[from]?.conns -= 1
                    self.changed()
                }
            }
            self.pump(from: c, to: up, end: end) { self.count(from, up: $0) }
            self.pump(from: up, to: c, end: end) { self.count(from, down: $0) }
        }
    }

    private func count(_ from: String, up: UInt64 = 0, down: UInt64 = 0) {
        status.up += up
        status.down += down
        clients[from]?.up += up
        clients[from]?.down += down
    }

    private func pump(from a: NWConnection, to b: NWConnection, end: @escaping () -> Void, counted: @escaping (UInt64) -> Void) {
        a.receive(minimumIncompleteLength: 1, maximumLength: 65536) { [weak self] data, _, done, err in
            guard let self else { return }
            if let d = data, !d.isEmpty {
                counted(UInt64(d.count))
                b.send(content: d, completion: .contentProcessed { e in
                    if e != nil { end() } else if done { b.send(content: nil, contentContext: .finalMessage, isComplete: true, completion: .contentProcessed { _ in end() }) } else { self.pump(from: a, to: b, end: end, counted: counted) }
                })
                return
            }
            if done || err != nil {
                b.send(content: nil, contentContext: .finalMessage, isComplete: true, completion: .contentProcessed { _ in end() })
                return
            }
            self.pump(from: a, to: b, end: end, counted: counted)
        }
    }
}
#endif
