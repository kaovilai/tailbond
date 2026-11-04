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
    // limits apply at once, also while sharing
    @Published var minBattery: Int = UserDefaults.standard.object(forKey: "minBattery") as? Int ?? 20 {
        didSet { Self.defaults.set(minBattery, forKey: "minBattery"); box.setMin(minBattery) }
    }
    @Published var capMB: Int = UserDefaults.standard.integer(forKey: "capMB") {
        didSet { Self.defaults.set(capMB, forKey: "capMB"); pushLimits() }
    }
    @Published var allowedHubsText: String = UserDefaults.standard.string(forKey: "allowedHubs") ?? "" {
        didSet { Self.defaults.set(allowedHubsText, forKey: "allowedHubs"); pushLimits() }
    }
    /// The keep-it-on-screen notice was shown (once, on the first share).
    @Published var screenNoticeSeen: Bool = UserDefaults.standard.bool(forKey: "screenNoticeSeen") {
        didSet { Self.defaults.set(screenNoticeSeen, forKey: "screenNoticeSeen") }
    }

    private func pushLimits() {
        relay?.updateLimits(capBytes: UInt64(max(capMB, 0)) * 1_000_000, allowedHubs: Set(allowedHubs))
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

    /// "Mobile data from dana-iphone": this iPhone's name on the home server
    /// (iOS doesn't tell apps the phone's own name), the same on every screen.
    var label: String {
        let n = pairing.map { $0.name.isEmpty ? $0.user : $0.name } ?? ""
        return "Mobile data from " + (n.isEmpty ? "an iPhone" : n)
    }

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
        box.setMin(minBattery)
        let box = self.box
        let core = SharerCore(
            config: .init(id: shareID, label: label, key: Array(key.utf8), capBytes: UInt64(max(capMB, 0)) * 1_000_000,
                          usedBytes: UInt64(usedBytesStored), allowedHubs: Set(allowedHubs)),
            check: { box.check() }
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

    // MARK: phone hub proxy (experimental, docs/phone-hub.md)

    @Published private(set) var proxyOn = false
    @Published private(set) var proxyStatus = ProxyServer.Status()
    private var proxy: ProxyServer?

    /// SOCKS5 / HTTP CONNECT on port 1888 for a Mac paired with the same home
    /// server (password from the share key). Upstream: this iPhone's mobile
    /// data. The bonded tunnel inside the app isn't built yet, so this is not
    /// bonded: the Mac gets the iPhone's mobile data through a proxy.
    func startProxy() {
        guard let p = pairing, let key = p.shareKey, !key.isEmpty else {
            pairError = "Open tailbond on a device paired with your home server once, then pair this iPhone again: the pairing has no sharing key."
            return
        }
        let c = ProxyCore.credentials(shareKey: Array(key.utf8))
        let name = p.name.isEmpty ? (p.user.isEmpty ? "iPhone" : p.user) : p.name
        let s = ProxyServer(user: c.user, pass: c.pass, dialer: DirectDialer(interface: .cellular),
                            advert: .init(name: name, id: shareID + "-hub", keyID: ShareProtocol.keyID(Array(key.utf8))))
        s.onChange = { [weak self] st in Task { @MainActor in self?.proxyStatus = st } }
        do {
            try s.start()
            proxy = s
            proxyOn = true
            UIApplication.shared.isIdleTimerDisabled = true
        } catch {
            proxyStatus.error = "\(error)"
        }
    }

    func stopProxy() {
        proxy?.stop()
        proxy = nil
        proxyOn = false
        proxyStatus = ProxyServer.Status()
        if !sharing { UIApplication.shared.isIdleTimerDisabled = false }
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
    private var min = 20

    func setMin(_ m: Int) {
        lock.lock()
        min = m
        lock.unlock()
    }

    func set(_ p: Int?, _ c: Bool) {
        lock.lock()
        pct = p
        chg = c
        lock.unlock()
    }

    func check() -> SharerCore.Check {
        lock.lock()
        defer { lock.unlock() }
        if min > 0, let p = pct, p < min, !chg { return .init(reason: .battery, battery: p, charging: chg) }
        return .init(battery: pct, charging: chg)
    }
}
