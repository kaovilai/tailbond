import CoreBluetooth
import Foundation

/// The menu bar app's Bluetooth login for one phone (GET /api/phone-connect/keys).
public struct BLELogin: Decodable, Equatable {
    public var kid: String
    public var user: String
    public var secret: String
    public init(kid: String, user: String, secret: String) { self.kid = kid; self.user = user; self.secret = secret }
}

/// Who answered over Bluetooth.
public struct PhoneHello: Equatable {
    public var id: String
    public var label: String
    public var kid: String
}

/// Why there's no phone over Bluetooth, or what the phone refused. `code`
/// is what tailbondd's POST /api/phone-connect takes as "ble".
public enum PhoneLinkError: Error, Equatable {
    case off, denied, unsupported, noPhone(String)
    case auth(String)
    case refused(code: String, message: String)
    case hotspot(String), hub(String)

    public var code: String {
        switch self {
        case .off: return "off"
        case .denied: return "denied"
        case .unsupported: return "unsupported"
        case .noPhone: return "no_phone"
        case .auth: return "auth"
        case .refused(let c, _): return c == "hub" ? "hub" : "hotspot"
        case .hotspot: return "hotspot"
        case .hub: return "hub"
        }
    }

    public var text: String {
        switch self {
        case .off: return "Bluetooth is off"
        case .denied: return "tailbond may not use Bluetooth"
        case .unsupported: return "no Bluetooth LE"
        case .noPhone(let s), .auth(let s), .hotspot(let s), .hub(let s): return s
        case .refused(_, let m): return m
        }
    }
}

/// A Bluetooth LE session with a phone running tailbond (PhoneProtocol):
/// scan, connect, log in, then commands. CoreBluetooth's callbacks arrive
/// on `queue`; every field is touched there only.
public final class PhoneLink: NSObject, CBCentralManagerDelegate, CBPeripheralDelegate {
    private let queue = DispatchQueue(label: "dev.tailbond.menu.ble")
    private var central: CBCentralManager?
    private var peripheral: CBPeripheral?
    private var rxChar: CBCharacteristic?
    private var reasm = PhoneProtocol.Reassembler()
    private var inbox: [Data] = []
    private var waiter: CheckedContinuation<Data, Error>?
    private var ready: CheckedContinuation<Void, Error>?
    private var key: Data?
    private var inCtr: UInt64 = 0
    private var outCtr: UInt64 = 0
    private var nextID = 0
    private var lostError: Error?
    private var tried = Set<UUID>()
    /// each wait's identity: a timer only ends the wait it was set for
    private var readyGen = 0
    private var recvGen = 0
    /// {"t":"state",...} pushes from the phone (the hub's state).
    public var onState: (([String: Any]) -> Void)?

    public override init() { super.init() }

    private let svc = CBUUID(string: PhoneProtocol.service)
    private let rx = CBUUID(string: PhoneProtocol.rx)
    private let tx = CBUUID(string: PhoneProtocol.tx)

    // MARK: waiting on the queue, with a timeout

    /// Runs `start` on the queue with `ready` set; a callback resumes it.
    private func until(_ seconds: Double, timeout: Error, _ start: @escaping () -> Void) async throws {
        try await withCheckedThrowingContinuation { (c: CheckedContinuation<Void, Error>) in
            queue.async {
                self.readyGen += 1
                let gen = self.readyGen
                self.ready = c
                start()
                self.queue.asyncAfter(deadline: .now() + seconds) {
                    if gen == self.readyGen, let r = self.ready { self.ready = nil; r.resume(throwing: timeout) }
                }
            }
        }
    }

    private func finish(_ err: Error? = nil) {
        guard let r = ready else { return }
        ready = nil
        if let err { r.resume(throwing: err) } else { r.resume() }
    }

    /// The next message from the phone (opened when logged in).
    private func receive(_ seconds: Double) async throws -> [String: Any] {
        let data = try await withCheckedThrowingContinuation { (c: CheckedContinuation<Data, Error>) in
            queue.async {
                if let e = self.lostError { c.resume(throwing: e); return }
                if !self.inbox.isEmpty { c.resume(returning: self.inbox.removeFirst()); return }
                self.recvGen += 1
                let gen = self.recvGen
                self.waiter = c
                self.queue.asyncAfter(deadline: .now() + seconds) {
                    if gen == self.recvGen, let w = self.waiter {
                        self.waiter = nil
                        w.resume(throwing: PhoneLinkError.noPhone("The phone stopped answering over Bluetooth"))
                    }
                }
            }
        }
        guard let o = (try? JSONSerialization.jsonObject(with: data)) as? [String: Any] else {
            throw PhoneLinkError.noPhone("The phone sent something that isn't tailbond's")
        }
        return o
    }

    private func send(_ o: [String: Any]) async throws {
        let plain = try JSONSerialization.data(withJSONObject: o)
        try await withCheckedThrowingContinuation { (c: CheckedContinuation<Void, Error>) in
            queue.async {
                do {
                    guard let p = self.peripheral, let ch = self.rxChar else { throw PhoneLinkError.noPhone("Not connected to the phone") }
                    var body = plain
                    if let k = self.key {
                        body = try PhoneProtocol.seal(key: k, dir: PhoneProtocol.toPhone, ctr: self.outCtr, plain)
                        self.outCtr += 1
                    }
                    // pieces of one ATT payload (withResponse would allow 512 through
                    // prepared writes); CoreBluetooth queues writes with response in order
                    let mtu = p.maximumWriteValueLength(for: .withoutResponse) + 3
                    for piece in PhoneProtocol.chunks(try PhoneProtocol.frame(body), mtu: mtu) {
                        p.writeValue(piece, for: ch, type: .withResponse)
                    }
                    c.resume()
                } catch {
                    c.resume(throwing: error)
                }
            }
        }
    }

    // MARK: the session

    /// Finds a phone, connects and logs in with the login for its key id.
    public func open(logins: [BLELogin], scanSeconds: Double = 10) async throws -> PhoneHello {
        // Bluetooth on (the first use asks the user for permission)
        try await until(15, timeout: PhoneLinkError.off) {
            self.central = CBCentralManager(delegate: self, queue: self.queue)
        }
        var lastRefusal: PhoneLinkError?
        // a phone that refuses (another home server's) doesn't stop the search
        for _ in 0..<3 {
            do {
                try await until(scanSeconds, timeout: lastRefusal ?? PhoneLinkError.noPhone("No phone with tailbond's \"One click from your Mac\" answered over Bluetooth")) {
                    self.central?.scanForPeripherals(withServices: [self.svc], options: nil)
                }
                return try await login(logins)
            } catch let e as PhoneLinkError {
                if case .auth = e { lastRefusal = e; disconnect(); continue }
                close()
                throw e
            }
        }
        close()
        throw lastRefusal ?? PhoneLinkError.noPhone("No phone answered")
    }

    private func login(_ logins: [BLELogin]) async throws -> PhoneHello {
        var nonce = [UInt8](repeating: 0, count: 16)
        guard SecRandomCopyBytes(kSecRandomDefault, 16, &nonce) == errSecSuccess else { throw PhoneLinkError.auth("no random bytes") }
        let cn = Data(nonce).base64EncodedString()
        try await send(["t": "hello", "v": PhoneProtocol.version, "cn": cn])
        let h = try await receive(10)
        guard h["t"] as? String == "hello", let sn = h["sn"] as? String else {
            throw PhoneLinkError.auth((h["error"] as? String) ?? "The phone didn't say hello")
        }
        let hello = PhoneHello(id: h["id"] as? String ?? "", label: h["label"] as? String ?? "phone", kid: h["kid"] as? String ?? "")
        guard let l = logins.first(where: { $0.kid == hello.kid }) else {
            throw PhoneLinkError.auth("This Mac has no pass for \(hello.label): is it paired with the same home server?")
        }
        try await send(["t": "auth", "user": l.user, "mac": PhoneProtocol.authMac(secret: l.secret, cn: cn, sn: sn, user: l.user)])
        let a = try await receive(10)
        guard a["ok"] as? Bool == true else {
            throw PhoneLinkError.auth("\(hello.label) refused this Mac's login: \(a["error"] as? String ?? "no reason given")")
        }
        guard a["proof"] as? String == PhoneProtocol.phoneProof(secret: l.secret, cn: cn, sn: sn, user: l.user) else {
            throw PhoneLinkError.auth("\(hello.label)'s answer doesn't prove it holds its key: not your phone")
        }
        let k = PhoneProtocol.sessionKey(secret: l.secret, cn: cn, sn: sn, user: l.user)
        queue.sync { self.key = k }
        return hello
    }

    /// One command; a refusal throws .refused(code, message).
    public func command(_ cmd: String, seconds: Double = 40) async throws -> [String: Any] {
        let id: Int = queue.sync { nextID += 1; return nextID }
        try await send(["t": "cmd", "id": id, "cmd": cmd])
        while true {
            let r = try await receive(seconds)
            guard r["t"] as? String == "res", (r["id"] as? Int) == id else { continue }
            if r["ok"] as? Bool == true { return r }
            throw PhoneLinkError.refused(code: r["code"] as? String ?? "", message: r["error"] as? String ?? "refused")
        }
    }

    private func disconnect() {
        queue.sync {
            if let p = peripheral { central?.cancelPeripheralConnection(p) }
            peripheral = nil
            rxChar = nil
            key = nil
            inCtr = 0
            outCtr = 0
            inbox = []
            reasm = PhoneProtocol.Reassembler()
            lostError = nil
        }
    }

    public func close() {
        disconnect()
        queue.sync {
            central?.stopScan()
            central = nil
        }
    }

    // MARK: CoreBluetooth

    public func centralManagerDidUpdateState(_ c: CBCentralManager) {
        switch c.state {
        case .poweredOn: finish()
        case .poweredOff: finish(PhoneLinkError.off)
        case .unauthorized: finish(PhoneLinkError.denied)
        case .unsupported: finish(PhoneLinkError.unsupported)
        default: break // resetting, unknown: wait
        }
    }

    public func centralManager(_ c: CBCentralManager, didDiscover p: CBPeripheral, advertisementData: [String: Any], rssi: NSNumber) {
        guard peripheral == nil, !tried.contains(p.identifier) else { return }
        tried.insert(p.identifier)
        c.stopScan()
        peripheral = p
        p.delegate = self
        c.connect(p, options: nil)
    }

    public func centralManager(_ c: CBCentralManager, didConnect p: CBPeripheral) {
        p.discoverServices([svc])
    }

    public func centralManager(_ c: CBCentralManager, didFailToConnect p: CBPeripheral, error: Error?) {
        peripheral = nil
        finish(PhoneLinkError.noPhone("Couldn't connect to the phone over Bluetooth"))
    }

    public func centralManager(_ c: CBCentralManager, didDisconnectPeripheral p: CBPeripheral, error: Error?) {
        guard p.identifier == peripheral?.identifier else { return }
        let e = PhoneLinkError.noPhone("The phone dropped the Bluetooth connection")
        lostError = e
        finish(e)
        if let w = waiter { waiter = nil; w.resume(throwing: e) }
    }

    public func peripheral(_ p: CBPeripheral, didDiscoverServices error: Error?) {
        guard let s = p.services?.first(where: { $0.uuid == svc }) else {
            finish(PhoneLinkError.noPhone("The phone has no tailbond service"))
            return
        }
        p.discoverCharacteristics([rx, tx], for: s)
    }

    public func peripheral(_ p: CBPeripheral, didDiscoverCharacteristicsFor s: CBService, error: Error?) {
        rxChar = s.characteristics?.first { $0.uuid == rx }
        guard rxChar != nil, let t = s.characteristics?.first(where: { $0.uuid == tx }) else {
            finish(PhoneLinkError.noPhone("The phone's tailbond service is incomplete"))
            return
        }
        p.setNotifyValue(true, for: t)
    }

    public func peripheral(_ p: CBPeripheral, didUpdateNotificationStateFor ch: CBCharacteristic, error: Error?) {
        if let error { finish(PhoneLinkError.noPhone("Bluetooth: \(error.localizedDescription)")) } else { finish() }
    }

    public func peripheral(_ p: CBPeripheral, didUpdateValueFor ch: CBCharacteristic, error: Error?) {
        guard ch.uuid == tx, let v = ch.value else { return }
        guard let msgs = try? reasm.add(v) else {
            let e = PhoneLinkError.noPhone("The phone sent a message too long")
            lostError = e
            if let w = waiter { waiter = nil; w.resume(throwing: e) }
            return
        }
        for m in msgs {
            var body = m
            if let k = key {
                guard let plain = PhoneProtocol.open(key: k, dir: PhoneProtocol.toMac, ctr: inCtr, m) else {
                    let e = PhoneLinkError.noPhone("A message from the phone didn't open: the session is over")
                    lostError = e
                    if let w = waiter { waiter = nil; w.resume(throwing: e) }
                    return
                }
                inCtr += 1
                body = plain
            }
            if key != nil, let o = (try? JSONSerialization.jsonObject(with: body)) as? [String: Any], o["t"] as? String == "state" {
                onState?(o)
                continue
            }
            if let w = waiter { waiter = nil; w.resume(returning: body) } else { inbox.append(body) }
        }
    }
}
