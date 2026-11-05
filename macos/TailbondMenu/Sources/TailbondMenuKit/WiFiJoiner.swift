import CoreLocation
import CoreWLAN
import Foundation

/// Joining the phone's hotspot Wi-Fi (docs/phone-hub.md, "One click from
/// the Mac"), the Mac's half of the click after Bluetooth: CoreWLAN, as
/// untether does it (https://github.com/ImOstrovskiy/untether,
/// BSD-3-Clause, Copyright (c) 2026 Oleksii Ostrovskyi — see
/// TetheringShell.kt in the Android app for the notice).
///
/// Scanning for a network by SSID needs Location permission on macOS 14+
/// (asked for once, here, before the first scan); the join itself is the
/// association CoreWLAN has always had.
enum WiFiJoiner {
    private static var askedForLocation = false
    private static let location = CLLocationManager()

    enum JoinError: LocalizedError {
        case noInterface
        case notVisible(String)

        var errorDescription: String? {
            switch self {
            case .noInterface: return "this Mac has no Wi-Fi interface"
            case .notVisible(let ssid): return "the hotspot \"\(ssid)\" isn't visible (too far, or it didn't start)"
            }
        }
    }

    static func currentSSID() -> String? {
        CWWiFiClient.shared().interface()?.ssid()
    }

    /// Joins [ssid] with [pass] ("" for an open network); already being on
    /// it is fine.
    static func join(ssid: String, pass: String) throws {
        guard let iface = CWWiFiClient.shared().interface() else { throw JoinError.noInterface }
        if iface.ssid() == ssid { return }
        if !iface.powerOn() {
            try iface.setPower(true)
        }
        if !askedForLocation {
            askedForLocation = true
            // a no-op when already decided; the prompt is asynchronous, and
            // the scan below says whether it was allowed
            location.requestWhenInUseAuthorization()
        }
        let networks = try iface.scanForNetworks(withSSID: ssid.data(using: .utf8))
        guard let network = networks.first else { throw JoinError.notVisible(ssid) }
        try iface.associate(toNetwork: network, password: pass.isEmpty ? nil : pass)
    }
}
