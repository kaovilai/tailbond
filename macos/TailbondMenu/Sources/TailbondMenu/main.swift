import AppKit
import ServiceManagement
import SwiftUI
import TailbondMenuKit

/// tailbond's menu bar app (docs/macos.md, "Menu bar").
///
///   tailbond.app              the menu bar icon and its window
///   tailbond --status [URL]   print what the menu would say, once, and exit
///                             (0 when the agent answered); CI and scripts
struct TailbondMenuApp: App {
    @StateObject private var model = MenuModel()
    @State private var openAtLogin = SMAppService.mainApp.status == .enabled

    init() {
        // the first launch asks macOS to open the app at login (the menu has the switch)
        if !UserDefaults.standard.bool(forKey: "loginItemAsked") {
            UserDefaults.standard.set(true, forKey: "loginItemAsked")
            try? SMAppService.mainApp.register()
        }
    }

    var body: some Scene {
        MenuBarExtra {
            MenuContent(model: model, loginItem: Binding(get: { openAtLogin }, set: { on in
                do {
                    if on { try SMAppService.mainApp.register() } else { try SMAppService.mainApp.unregister() }
                } catch {
                    model.message = "Open at login: \(error.localizedDescription)"
                }
                openAtLogin = SMAppService.mainApp.status == .enabled
            }))
            .onAppear { model.start() }
        } label: {
            Image(systemName: model.summary.symbol)
                .onAppear { model.start() }
        }
        .menuBarExtraStyle(.window)
    }
}

let args = CommandLine.arguments
if let i = args.firstIndex(of: "--status") {
    let url = args.count > i + 1 ? URL(string: args[i + 1]) : nil
    let client = url.map { AgentClient(base: $0) } ?? AgentClient()
    Task.detached {
        do {
            let s = try await client.status()
            print(MenuSummary(s).text)
            exit(0)
        } catch {
            print(MenuSummary(nil).text)
            exit(1)
        }
    }
    dispatchMain()
}
if args.contains("--version") {
    print(Bundle.main.infoDictionary?["CFBundleShortVersionString"] as? String ?? "dev")
    exit(0)
}
TailbondMenuApp.main()
