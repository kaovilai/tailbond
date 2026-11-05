import CoreWLAN
import Foundation

/// Joins the phone's hotspot with CoreWLAN, as the user. false: it didn't
/// (no Wi-Fi, the network wasn't found, macOS refused): tailbondd then
/// joins it with networksetup as root.
enum WiFiJoin {
    static func join(ssid: String, pass: String) -> Bool {
        guard !ssid.isEmpty, let iface = CWWiFiClient.shared().interface() else { return false }
        if iface.ssid() == ssid { return true }
        do {
            guard let net = try iface.scanForNetworks(withName: ssid).first else { return false }
            try iface.associate(to: net, password: pass)
            return true
        } catch {
            return false
        }
    }
}
