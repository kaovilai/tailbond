import SwiftUI
import TailbondShareKit
import UIKit

/// What the screens show, and the relay behind them.
@MainActor
final class ShareModel: ObservableObject {
    @Published private(set) var pairing: SharePairing?
    @Published var pairError: String?
    @Published private(set) var sharing = false
    @Published private(set) var status = ShareRelay.Status()
    /// Paused because the app left the screen (iOS suspends it).
    @Published private(set) var pausedInBackground = false
    @Published private(set) var stopReason: String?

    private static let defaults = UserDefaults.standard
    @Published var minBattery: Int = UserDefaults.standard.object(forKey: "minBattery") as? Int ?? 20 {
        didSet { Self.defaults.set(minBattery, forKey: "minBattery") }
    }
    @Published var capMB: Int = UserDefaults.standard.integer(forKey: "capMB") {
        didSet { Self.defaults.set(capMB, forKey: "capMB") }
    }
    @Published var allowedHubsText: String = UserDefaults.standard.string(forKey: "allowedHubs") ?? "" {
        didSet { Self.defaults.set(allowedHubsText, forKey: "allowedHubs") }
    }
    private var usedMonth: String {
        get { Self.defaults.string(forKey: "usedMonth") ?? "" }
        set { Self.defaults.set(newValue, forKey: "usedMonth") }
    }
    private var usedBytesStored: Double {
        get { Self.defaults.double(forKey: "usedBytes") }
        set { Self.defaults.set(newValue, forKey: "usedBytes") }
    }
    private var shareID: String {
        get { Self.defaults.string(forKey: "shareID") ?? "" }
        set { Self.defaults.set(newValue, forKey: "shareID") }
    }

    private var relay: ShareRelay?

    init() {
        if let d = Keychain.load() { pairing = try? JSONDecoder().decode(SharePairing.self, from: d) }
        if shareID.isEmpty { shareID = "ios-" + UUID().uuidString.prefix(10).lowercased() }
        UIDevice.current.isBatteryMonitoringEnabled = true
        for n in [UIDevice.batteryLevelDidChangeNotification, UIDevice.batteryStateDidChangeNotification] {
            NotificationCenter.default.addObserver(forName: n, object: nil, queue: .main) { [weak self] _ in
                Task { @MainActor in self?.refreshBattery() }
            }
        }
    }

    var label: String { "\(UIDevice.current.name)'s mobile data" }

    var allowedHubs: [String] {
        allowedHubsText.split(separator: ",").map { $0.trimmingCharacters(in: .whitespaces) }.filter { !$0.isEmpty }
    }

    func setAllowedHubs(_ hubs: [String]) { allowedHubsText = hubs.joined(separator: ",") }

    private var month: String {
        let f = DateFormatter()
        f.dateFormat = "yyyy-MM"
        return f.string(from: Date())
    }

    /// This month's shared bytes.
    var usedThisMonth: UInt64 {
        if usedMonth != month { return 0 }
        return max(UInt64(usedBytesStored), status.used)
    }

    // MARK: pairing

    func pair(text: String) {
        do {
            let p = try SharePairing.parse(text)
            Keychain.save(try JSONEncoder().encode(p))
            pairing = p
            pairError = nil
        } catch SharePairing.ParseError.noShareKey {
            pairError = "This pairing is from an older home server. Update tailbond on it, then make a new code."
        } catch {
            pairError = "That isn't a tailbond pairing link or code."
        }
    }

    func forget() {
        stop()
        Keychain.delete()
        pairing = nil
    }

    // MARK: sharing

    var battery: Int? {
        let l = UIDevice.current.batteryLevel
        return l < 0 ? nil : Int((l * 100).rounded())
    }

    var charging: Bool {
        let s = UIDevice.current.batteryState
        return s == .charging || s == .full
    }

    /// Battery as last seen on the main thread; the relay reads it from its queue.
    private let box = BatteryBox()

    private func refreshBattery() { box.set(battery, charging) }

    func start() {
        guard let p = pairing, let key = p.shareKey, relay == nil else { return }
        if usedMonth != month {
            usedMonth = month
            usedBytesStored = 0
        }
        refreshBattery()
        let min = minBattery
        let box = self.box
        let core = SharerCore(
            config: .init(id: shareID, label: label, key: Array(key.utf8), capBytes: UInt64(max(capMB, 0)) * 1_000_000,
                          usedBytes: UInt64(usedBytesStored), allowedHubs: Set(allowedHubs)),
            check: { box.check(min: min) }
        )
        let r = ShareRelay(core: core, serverHost: p.host, serverPort: p.port)
        r.onChange = { [weak self] st in
            Task { @MainActor in
                guard let self else { return }
                self.status = st
                self.usedBytesStored = Double(st.used)
            }
        }
        do {
            try r.start()
            relay = r
            sharing = true
            stopReason = nil
            pausedInBackground = false
            // iOS suspends apps that leave the screen; keep it on while sharing
            UIApplication.shared.isIdleTimerDisabled = true
        } catch {
            stopReason = "Couldn't start: \(error.localizedDescription)"
        }
    }

    func stop(reason: ShareProtocol.Reason = .paused) {
        relay?.stop(reason)
        relay = nil
        sharing = false
        status.hubs = []
        UIApplication.shared.isIdleTimerDisabled = false
    }

    func scenePhase(_ p: ScenePhase) {
        switch p {
        case .background:
            if sharing {
                stop()
                pausedInBackground = true
            }
        case .active:
            if pausedInBackground {
                pausedInBackground = false
                start()
            }
        default:
            break
        }
    }

    // MARK: words

    var headline: String {
        if !sharing { return pausedInBackground ? "Paused: tailbond Share left the screen" : "Not sharing" }
        if let b = battery, minBattery > 0, b < minBattery, !charging { return "Paused: battery low (\(b)%)" }
        if !status.cellularReady { return "Paused: no mobile data" }
        if !status.listening { return "Waiting for Wi-Fi" }
        if status.hubs.isEmpty { return "Ready to share" }
        let names = status.hubs.map(\.hub)
        let bytes = status.hubs.reduce(UInt64(0)) { $0 + $1.upBytes + $1.downBytes }
        return "Sharing mobile data with \(names.joined(separator: " and ")): \(Self.bytes(bytes))"
    }

    static func bytes(_ b: UInt64) -> String {
        ByteCountFormatter.string(fromByteCount: Int64(clamping: b), countStyle: .decimal)
    }
}

/// Thread-safe battery reading for the relay's queue.
final class BatteryBox: @unchecked Sendable {
    private let lock = NSLock()
    private var pct: Int?
    private var chg = false

    func set(_ p: Int?, _ c: Bool) {
        lock.lock()
        pct = p
        chg = c
        lock.unlock()
    }

    func check(min: Int) -> SharerCore.Check {
        lock.lock()
        defer { lock.unlock() }
        if min > 0, let p = pct, p < min, !chg { return .init(reason: .battery, battery: p, charging: chg) }
        return .init(battery: pct, charging: chg)
    }
}
