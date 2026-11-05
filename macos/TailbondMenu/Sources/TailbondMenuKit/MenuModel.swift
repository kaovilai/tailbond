import Foundation
import SwiftUI

/// The menu's state: the agent's status every 2 s, and its actions.
@MainActor
public final class MenuModel: ObservableObject {
    @Published public private(set) var status: AgentStatus?
    @Published public private(set) var summary = MenuSummary(nil)
    @Published public private(set) var busy = false
    @Published public private(set) var clicking = false
    @Published public var message: String?

    public let client: AgentClient
    private var poller: Task<Void, Never>?
    /// The Bluetooth link to the phone (nil: this Mac doesn't try — the
    /// CLI, previews, snapshots).
    public var phoneLink: PhoneLink?

    public init(client: AgentClient = AgentClient()) {
        self.client = client
    }

    /// Preview and snapshot use: a fixed status, no polling, no Bluetooth.
    public init(fixed: AgentStatus?, unreachable: String? = nil) {
        client = AgentClient()
        status = fixed
        summary = MenuSummary(fixed, unreachable: unreachable)
    }

    public func start() {
        guard poller == nil else { return }
        poller = Task { [weak self] in
            while !Task.isCancelled {
                await self?.refresh()
                try? await Task.sleep(nanoseconds: 2_000_000_000)
            }
        }
    }

    public func refresh() async {
        do {
            let s = try await client.status()
            status = s
            summary = MenuSummary(s)
        } catch AgentClient.Failure.needsLogin {
            status = nil
            summary = MenuSummary(nil, unreachable: "The dashboard has a password: open it to sign in, then the menu works again.")
        } catch {
            status = nil
            summary = MenuSummary(nil)
        }
    }

    private func act(_ what: String, _ f: @escaping () async throws -> Void) {
        busy = true
        Task {
            do { try await f() } catch { message = "\(what) didn't work: \(error)" }
            await refresh()
            busy = false
        }
    }

    public func setMode(_ m: String) { act("Changing the mode") { try await self.client.setMode(m) } }
    public func setOn(_ on: Bool) { act(on ? "Turning on" : "Turning off") { try await self.client.connect(on) } }
    public func setPolicy(_ p: String) { act("Changing updates") { try await self.client.setUpdatePolicy(p) } }
    public func checkUpdate() { act("Checking for updates") { try await self.client.checkUpdate() } }
    public func installUpdate() { act("Installing the update") { try await self.client.installUpdate() } }
    public func setWorkVPN(_ w: String) { act("Changing the work VPN's traffic") { try await self.client.setWorkVPN(w) } }
    public func dismissCrash() { act("Dismissing") { try await self.client.dismissCrash() } }
    public func setSharing(_ on: Bool) { act(on ? "Sharing this Mac's connection" : "Stopping sharing") { try await self.client.setSharing(on) } }

    /// The one click (docs/phone-hub.md, "One click from the Mac"). On:
    /// the Bluetooth half here (start the phone's hotspot, its proxy),
    /// then join its Wi-Fi, then tailbondd's half (find the proxy, point
    /// the system proxy at it, check the internet). A Bluetooth failure
    /// falls back to an already-joined hotspot, saying so. Off: the phone's
    /// proxy off over Bluetooth (best effort), then tailbondd's settings
    /// back; the hotspot is left on.
    public func clickPhone(_ on: Bool) {
        guard !clicking else { return }
        clicking = true
        Task {
            defer { clicking = false }
            do {
                if on {
                    var report = AgentClient.ClickReport()
                    (report.ble, report.wifi) = await bleAndWifiHalf()
                    _ = try await client.clickPhone(report)
                } else {
                    await phoneLink?.hubOff()
                    var report = AgentClient.ClickReport()
                    report.on = false
                    _ = try await client.clickPhone(report)
                }
            } catch {
                message = "\(on ? "Connecting via the phone" : "Disconnecting from the phone") didn't work: \(error)"
            }
            await refresh()
        }
    }

    /// The menu app's own halves of the click: Bluetooth to the phone
    /// (its hotspot on, its proxy on, its status) and joining the hotspot
    /// Wi-Fi. Both nil when Bluetooth isn't tried here (the CLI, which is
    /// the no-Bluetooth fallback); a failed half is reported as such, and
    /// tailbondd falls back.
    private func bleAndWifiHalf() async -> (AgentClient.ClickReport.BLE?, AgentClient.ClickReport.WiFi?) {
        if phoneLink == nil { phoneLink = PhoneLink() }
        guard let link = phoneLink else { return (nil, nil) }
        do {
            // the logins come from the hub list, which carries them only to
            // a loopback caller — this app
            let logins = (try? await client.phoneHub())?.logins ?? []
            let info = try await link.click(logins: logins)
            let ssid = info.ssid
            let pass = info.pass
            var ble = AgentClient.ClickReport.BLE()
            ble.state = "ok"
            ble.keyId = info.keyId
            ble.ssid = info.ssid
            ble.pass = info.pass
            ble.net = info.net
            ble.battery = info.battery
            ble.charging = info.charging
            var wifi: AgentClient.ClickReport.WiFi?
            if !info.ssid.isEmpty {
                do {
                    // the join scans and associates: seconds, off the main
                    // actor so the menu doesn't freeze
                    try await Task.detached(priority: .userInitiated) {
                        try WiFiJoiner.join(ssid: ssid, pass: pass)
                    }.value
                    wifi = AgentClient.ClickReport.WiFi()
                    wifi?.ok = true
                    wifi?.ssid = info.ssid
                } catch {
                    wifi = AgentClient.ClickReport.WiFi()
                    wifi?.ssid = info.ssid
                    wifi?.error = error.localizedDescription
                }
            }
            return (ble, wifi)
        } catch let e as PhoneLinkError {
            link.disconnect()
            var ble = AgentClient.ClickReport.BLE()
            ble.state = "failed"
            ble.step = e.step
            ble.error = e.message
            return (ble, nil)
        } catch {
            link.disconnect()
            var ble = AgentClient.ClickReport.BLE()
            ble.state = "failed"
            ble.step = "scan"
            ble.error = error.localizedDescription
            return (ble, nil)
        }
    }
}
