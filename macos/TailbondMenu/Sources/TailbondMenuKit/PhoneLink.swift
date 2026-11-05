import CoreBluetooth
import Foundation

/// What the Mac learned from the phone over Bluetooth: its hotspot's
/// network and its status.
public struct PhoneInfo: Equatable {
    public var keyId = ""
    public var ssid = ""
    public var pass = ""
    public var net = ""
    public var battery = 0
    public var charging = false
    public init() {}
}

/// A failure of the Bluetooth half, sorted by step (scan, auth, hotspot,
/// hub, status): the one click falls back to an already-joined hotspot
/// when it gets one of these, saying so.
public struct PhoneLinkError: Error, Equatable {
    public let step: String
    public let message: String
    public init(step: String, _ message: String) {
        self.step = step
        self.message = message
    }
}

/// The Mac's half of the phone's BLE control channel (docs/phone-hub.md,
/// "One click from the Mac"; the phone's half is share/BleProto.kt): a
/// CoreBluetooth central that scans for the tailbond service, says hello
/// with the key tailbondd derived for that phone, and carries the
/// hotspot/hub/status commands. The characteristics need an encrypted,
/// bonded link, so the very first click pairs the Mac with the phone (a
/// one-time numeric comparison on both screens).
///
/// Untether (https://github.com/ImOstrovskiy/untether, BSD-3-Clause,
/// Copyright (c) 2026 Oleksii Ostrovskyi) showed this shape — the menu bar
/// central that stays connected to the phone; the protocol it carries is
/// tailbond's.
public final class PhoneLink: NSObject {
    public static let service = CBUUID(string: "54424C31-5A1C-4B7E-9F2D-6E8B1D0A3C5F")
    static let challenge = CBUUID(string: "54424C31-5A1C-4B7E-9F2D-6E8B1D0A3C60")
    static let command = CBUUID(string: "54424C31-5A1C-4B7E-9F2D-6E8B1D0A3C61")
    static let state = CBUUID(string: "54424C31-5A1C-4B7E-9F2D-6E8B1D0A3C62")

    /// The last status the phone pushed (battery, network, hub), for
    /// whoever wants a fresher one than the click's.
    public private(set) var pushed: [String: Any]?

    private let queue = DispatchQueue(label: "dev.tailbond.menu.phonelink")
    private var manager: CBCentralManager?
    private var peripheral: CBPeripheral?
    private var challengeChar: CBCharacteristic?
    private var commandChar: CBCharacteristic?
    private var stateChar: CBCharacteristic?
    private var reassembly = BleFrame.Reassembly()
    private var nextId = 1
    private var authed = false

    /// The delegate events, and who waits for them: every waiter resumes
    /// exactly once (matched by an event, or by its timeout, or by a
    /// reset), and events nobody waited for yet are replayed to the next
    /// waiter that wants them.
    private enum Event {
        case managerState
        case found(CBPeripheral)
        case connected
        case charsReady
        case value(CBUUID, Data)
        case wrote(Error?)
        case notifyOn(Error?)
        case gone
    }

    private final class Waiter {
        let predicate: (Event) -> Bool
        let continuation: CheckedContinuation<Event, Error>
        init(_ predicate: @escaping (Event) -> Bool, _ continuation: CheckedContinuation<Event, Error>) {
            self.predicate = predicate
            self.continuation = continuation
        }
    }

    private var events: [Event] = []
    private var waiters: [Waiter] = []

    private func emit(_ e: Event) {
        if let i = waiters.firstIndex(where: { $0.predicate(e) }) {
            let w = waiters.remove(at: i)
            w.continuation.resume(returning: e)
        } else {
            events.append(e)
        }
    }

    /// Waits for an event the predicate wants (a missed one is replayed);
    /// throws [PhoneLinkError] with [message] after [timeout].
    private func wait(for predicate: @escaping (Event) -> Bool, timeout: TimeInterval, step: String, _ message: String) async throws -> Event {
        try await withCheckedThrowingContinuation { cont in
            let w = Waiter(predicate, cont)
            queue.async {
                if let i = self.events.firstIndex(where: { predicate($0) }) {
                    let e = self.events.remove(at: i)
                    cont.resume(returning: e)
                    return
                }
                self.waiters.append(w)
                self.queue.asyncAfter(deadline: .now() + timeout) {
                    if let i = self.waiters.firstIndex(where: { $0 === w }) {
                        self.waiters.remove(at: i)
                        w.continuation.resume(throwing: PhoneLinkError(step: step, message))
                    }
                }
            }
        }
    }

    // MARK: the one click's sequence

    /// Scans, connects, says hello, starts the phone's hotspot and its
    /// proxy, and asks its status: everything but joining the Wi-Fi.
    public func click(logins: [PhoneHubInfo.Login]) async throws -> PhoneInfo {
        try await ensureManager()
        try await ensureConnected()
        let (keyId, challenge) = try await readChallenge()
        let login = logins.first { $0.keyId == keyId }
        guard let login else {
            throw PhoneLinkError(step: "auth", message: logins.isEmpty
                ? "connect to your home server once first: the phone's password comes from it"
                : "this Mac has no password for that phone (paired with another home server?)")
        }
        let hello = try await send(["cmd": "hello", "user": login.user, "proof": BleFrame.proof(key: login.key, challenge: challenge)], timeout: 15, step: "auth")
        if hello["ok"] as? Bool != true {
            throw PhoneLinkError(step: "auth", message: (hello["error"] as? String) ?? "the phone refused this Mac's password")
        }
        authed = true
        let on = try await send(["cmd": "hotspot.on"], timeout: 40, step: "hotspot")
        if on["ok"] as? Bool != true {
            throw PhoneLinkError(step: "hotspot", message: (on["error"] as? String) ?? "the phone couldn't start its hotspot")
        }
        var info = PhoneInfo()
        info.keyId = keyId
        info.ssid = (on["ssid"] as? String) ?? ""
        info.pass = (on["pass"] as? String) ?? ""
        // the proxy: asked for, on when it can be (it may already be;
        // tailbondd looks for it either way)
        _ = try? await send(["cmd": "hub.on"], timeout: 15, step: "hub")
        if let st = try? await send(["cmd": "status"], timeout: 10, step: "status"), let s = st["status"] as? [String: Any] {
            info.net = (s["net"] as? String) ?? ""
            info.battery = (s["battery"] as? Int) ?? 0
            info.charging = (s["charging"] as? Bool) ?? false
        }
        return info
    }

    /// Asks the phone to turn its proxy off (best effort; the hotspot is
    /// left on unless the user asks for it off).
    public func hubOff() async {
        guard authed, commandChar != nil else { return }
        _ = try? await send(["cmd": "hub.off"], timeout: 10, step: "hub")
    }

    /// Tears the link down (a failed click leaves nothing behind).
    public func disconnect() {
        queue.async {
            if let p = self.peripheral {
                self.manager?.cancelPeripheralConnection(p)
            }
            self.reset()
        }
    }

    private func reset() {
        peripheral = nil
        challengeChar = nil
        commandChar = nil
        stateChar = nil
        authed = false
        reassembly = BleFrame.Reassembly()
        events.removeAll()
        let ws = waiters
        waiters.removeAll()
        ws.forEach { $0.continuation.resume(returning: .gone) }
    }

    private func ensureManager() async throws {
        if manager == nil {
            manager = CBCentralManager(delegate: self, queue: queue)
        }
        let m = manager
        _ = try await wait(
            for: { if case .managerState = $0 { return true }; return false },
            timeout: 5, step: "scan", "Bluetooth didn't answer",
        )
        switch m?.state {
        case .poweredOn: return
        case .unauthorized: throw PhoneLinkError(step: "scan", message: "Bluetooth isn't allowed for tailbond (System Settings → Privacy & Security → Bluetooth)")
        case .unsupported: throw PhoneLinkError(step: "scan", message: "this Mac has no Bluetooth")
        case .poweredOff: throw PhoneLinkError(step: "scan", message: "Bluetooth is off")
        default: throw PhoneLinkError(step: "scan", message: "Bluetooth isn't ready")
        }
    }

    private func ensureConnected() async throws {
        if peripheral != nil, challengeChar != nil { return }
        manager?.scanForPeripherals(withServices: [Self.service], options: nil)
        let found = try await wait(
            for: { if case .found = $0 { return true }; return false },
            timeout: 10, step: "scan", "no phone was found over Bluetooth (is tailbond's \"Share bonded connection\" on?)",
        )
        guard case .found(let p) = found else { throw PhoneLinkError(step: "scan", message: "the phone's Bluetooth link is gone") }
        queue.async {
            self.peripheral = p
            p.delegate = self
            self.manager?.connect(p, options: nil)
        }
        let connected = try await wait(
            for: { if case .connected = $0 { return true }; if case .gone = $0 { return true }; return false },
            timeout: 15, step: "scan", "the phone didn't take the Bluetooth connection (a pairing prompt on either screen, unanswered?)",
        )
        if case .gone = connected {
            throw PhoneLinkError(step: "scan", message: "the phone's Bluetooth connection dropped")
        }
        queue.async { self.peripheral?.discoverServices([Self.service]) }
        _ = try await wait(
            for: { if case .charsReady = $0 { return true }; return false },
            timeout: 10, step: "scan", "the phone's Bluetooth service didn't answer",
        )
        // the phone's pushes come on the state characteristic
        if let sc = stateChar {
            queue.async { self.peripheral?.setNotifyValue(true, for: sc) }
            _ = try? await wait(
                for: { if case .notifyOn = $0 { return true }; return false },
                timeout: 5, step: "scan", "",
            )
        }
    }

    private func readChallenge() async throws -> (String, Data) {
        guard let c = challengeChar else { throw PhoneLinkError(step: "auth", message: "the phone's Bluetooth service has no challenge") }
        queue.async { self.peripheral?.readValue(for: c) }
        let e = try await wait(
            for: { if case .value(let uuid, _) = $0, uuid == Self.challenge { return true }; return false },
            timeout: 20, step: "auth", "the phone didn't answer the challenge (a pairing prompt on either screen, unanswered?)",
        )
        guard case .value(_, let data) = e, data.count >= 24 else {
            throw PhoneLinkError(step: "auth", message: "the phone's challenge was malformed")
        }
        let keyId = String(data: data.prefix(8), encoding: .utf8) ?? ""
        return (keyId, Data(data.suffix(16)))
    }

    /// One command: chunk the frame to the ATT MTU, write it, wait for the
    /// reply with the same id.
    private func send(_ json: [String: Any], timeout: TimeInterval, step: String) async throws -> [String: Any] {
        guard let c = commandChar else { throw PhoneLinkError(step: step, message: "the phone's Bluetooth link is gone") }
        let id = queue.sync { () -> Int in
            let v = nextId
            nextId += 1
            return v
        }
        var msg = json
        msg["id"] = id
        let frame = BleFrame.encode(msg)
        let mtu = (peripheral?.maximumWriteValueLength(for: .withResponse) ?? 20) + 3
        for chunk in BleFrame.chunks(frame, mtu: mtu) {
            queue.async { self.peripheral?.writeValue(chunk, for: c, type: .withResponse) }
            _ = try await wait(
                for: { if case .wrote = $0 { return true }; return false },
                timeout: 10, step: step, "the phone didn't take a Bluetooth write",
            )
        }
        let reply = try await wait(
            for: { e in
                if case .value(let uuid, let data) = e, uuid == Self.state {
                    if let obj = try? JSONSerialization.jsonObject(with: data) as? [String: Any], obj["id"] as? Int == id {
                        return true
                    }
                }
                return false
            },
            timeout: timeout, step: step, "the phone didn't answer in time",
        )
        guard case .value(_, let data) = reply,
              let obj = try? JSONSerialization.jsonObject(with: data) as? [String: Any] else {
            throw PhoneLinkError(step: step, message: "the phone's answer was malformed")
        }
        return obj
    }
}

extension PhoneLink: CBCentralManagerDelegate {
    public func centralManagerDidUpdateState(_ central: CBCentralManager) {
        emit(.managerState)
    }

    public func centralManager(_ central: CBCentralManager, didDiscover peripheral: CBPeripheral, advertisementData: [String: Any], rssi RSSI: NSNumber) {
        emit(.found(peripheral))
    }

    public func centralManager(_ central: CBCentralManager, didConnect peripheral: CBPeripheral) {
        emit(.connected)
    }

    public func centralManager(_ central: CBCentralManager, didFailToConnect peripheral: CBPeripheral, error: Error?) {
        emit(.gone)
    }

    public func centralManager(_ central: CBCentralManager, didDisconnectPeripheral peripheral: CBPeripheral, error: Error?) {
        reset()
        emit(.gone)
    }
}

extension PhoneLink: CBPeripheralDelegate {
    public func peripheral(_ peripheral: CBPeripheral, didDiscoverServices error: Error?) {
        guard let s = peripheral.services?.first(where: { $0.uuid == Self.service }) else {
            emit(.gone)
            return
        }
        peripheral.discoverCharacteristics([Self.challenge, Self.command, Self.state], for: s)
    }

    public func peripheral(_ peripheral: CBPeripheral, didDiscoverCharacteristicsFor service: CBService, error: Error?) {
        for c in service.characteristics ?? [] {
            switch c.uuid {
            case Self.challenge: challengeChar = c
            case Self.command: commandChar = c
            case Self.state: stateChar = c
            default: break
            }
        }
        emit(.charsReady)
    }

    public func peripheral(_ peripheral: CBPeripheral, didUpdateValueFor characteristic: CBCharacteristic, error: Error?) {
        guard let data = characteristic.value else { return }
        if characteristic.uuid == Self.challenge {
            emit(.value(characteristic.uuid, data))
        } else if characteristic.uuid == Self.state {
            // a chunk of a frame: replies and pushes arrive here
            guard let payload = reassembly.feed(data) else { return }
            if let obj = try? JSONSerialization.jsonObject(with: payload) as? [String: Any] {
                if obj["push"] != nil {
                    pushed = obj["status"] as? [String: Any]
                } else if obj["id"] != nil {
                    emit(.value(characteristic.uuid, payload))
                }
            }
        }
    }

    public func peripheral(_ peripheral: CBPeripheral, didWriteValueFor characteristic: CBCharacteristic, error: Error?) {
        emit(.wrote(error))
    }

    public func peripheral(_ peripheral: CBPeripheral, didUpdateNotificationStateFor characteristic: CBCharacteristic, error: Error?) {
        emit(.notifyOn(error))
    }
}
