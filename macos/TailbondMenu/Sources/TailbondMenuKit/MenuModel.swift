import Foundation
import SwiftUI

/// The menu's state: the agent's status every 2 s, and its actions.
@MainActor
public final class MenuModel: ObservableObject {
    @Published public private(set) var status: AgentStatus?
    @Published public private(set) var summary = MenuSummary(nil)
    @Published public private(set) var busy = false
    @Published public var message: String?
    /// "Connect via phone" while the app's own Bluetooth half runs (then
    /// tailbondd's status says where it is); nil otherwise.
    @Published public private(set) var phoneStage: String?
    /// the Bluetooth session, kept for hub.off on disconnect
    private var link: PhoneLink? { didSet { phoneLinked = link != nil } }
    /// a Bluetooth session to the phone is open (it can turn the hotspot off)
    @Published public private(set) var phoneLinked = false

    public let client: AgentClient
    private var poller: Task<Void, Never>?

    public init(client: AgentClient = AgentClient()) {
        self.client = client
    }

    /// Preview and snapshot use: a fixed status, no polling.
    public init(fixed: AgentStatus?, unreachable: String? = nil, phoneStage: String? = nil) {
        client = AgentClient()
        status = fixed
        summary = MenuSummary(fixed, unreachable: unreachable)
        self.phoneStage = phoneStage
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
    /// "Connect via phone" (docs/phone-hub.md, "One click from the Mac"):
    /// over Bluetooth, the phone starts its hotspot and Share bonded
    /// connection, this Mac joins the hotspot (CoreWLAN), then tailbondd
    /// finds the phone hub, sets the proxy and checks the internet. Without
    /// Bluetooth (off, denied, no phone in range) tailbondd uses a phone
    /// hotspot this Mac is on already, and says so.
    public func connectViaPhone() {
        guard phoneStage == nil else { return }
        message = nil
        phoneStage = "Looking for your phone over Bluetooth…"
        Task {
            var req: [String: Any]
            do {
                let logins = try await client.phoneKeys()
                req = try await bluetoothHalf(logins)
            } catch let e as PhoneLinkError {
                link?.close()
                link = nil
                req = ["ble": e.code, "ble_error": e.text]
            } catch {
                message = "Connect via phone didn't work: tailbond didn't answer (\(error))"
                phoneStage = nil
                return
            }
            do { try await client.phoneConnect(req) } catch { message = "Connect via phone didn't work: \(error)" }
            phoneStage = nil
            await refresh()
        }
    }

    private func bluetoothHalf(_ logins: [BLELogin]) async throws -> [String: Any] {
        if logins.isEmpty {
            throw PhoneLinkError.auth("This Mac has no pass for a phone yet: connect to your home server once, with the phone paired too")
        }
        link?.close()
        let l = PhoneLink()
        link = l
        let hello = try await l.open(logins: logins)
        phoneStage = "Turning on \(hello.label)'s hotspot…"
        let hs: [String: Any]
        do { hs = try await l.command("hotspot.on") } catch PhoneLinkError.refused(_, let m) { throw PhoneLinkError.hotspot(m) }
        phoneStage = "Turning on Share bonded connection on \(hello.label)…"
        do { _ = try await l.command("hub.on") } catch PhoneLinkError.refused(_, let m) { throw PhoneLinkError.hub(m) }
        let st = (try? await l.command("status")) ?? [:]
        let ssid = hs["ssid"] as? String ?? "", pass = hs["pass"] as? String ?? ""
        phoneStage = "Joining \(ssid)…"
        let joined = await Task.detached { WiFiJoin.join(ssid: ssid, pass: pass) }.value
        return ["ble": "ok", "ssid": ssid, "pass": pass, "joined": joined,
                "phone": ["id": hello.id, "label": hello.label, "kid": hello.kid, "network": st["network"] as? String ?? "",
                          "battery": st["battery"] as? Int ?? 0, "paths": st["paths"] as? Int ?? 0] as [String: Any]]
    }

    /// The second click: the proxy settings go back, the phone stops sharing;
    /// its hotspot stays on unless `hotspotOff`.
    public func disconnectPhone(hotspotOff: Bool = false) {
        act("Disconnecting from the phone") {
            try await self.client.phoneDisconnect()
            if let l = self.link {
                _ = try? await l.command("hub.off", seconds: 10)
                if hotspotOff { _ = try? await l.command("hotspot.off", seconds: 10) }
                l.close()
                self.link = nil
            }
        }
    }

    public func setSharing(_ on: Bool) { act(on ? "Sharing this Mac's connection" : "Stopping sharing") { try await self.client.setSharing(on) } }
}
