import Foundation
import SwiftUI

/// The menu's state: the agent's status every 2 s, and its actions.
@MainActor
public final class MenuModel: ObservableObject {
    @Published public private(set) var status: AgentStatus?
    @Published public private(set) var summary = MenuSummary(nil)
    @Published public private(set) var busy = false
    @Published public var message: String?

    public let client: AgentClient
    private var poller: Task<Void, Never>?

    public init(client: AgentClient = AgentClient()) {
        self.client = client
    }

    /// Preview and snapshot use: a fixed status, no polling.
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
}
