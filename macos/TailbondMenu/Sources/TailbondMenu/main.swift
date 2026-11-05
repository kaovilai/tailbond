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
// the one click without Bluetooth (docs/phone-hub.md, "One click from the
// Mac"): the menu app's fallback path — an already-joined phone hotspot and
// "Use phone hub" — driven from a terminal or CI:
//
//   tailbond --phone [URL] [on|off] [--check URL]
//
// (on by default; --check overrides what the internet check fetches)
if let i = args.firstIndex(of: "--phone") {
    var url: URL?
    var on = true
    var check: String?
    var j = i + 1
    while j < args.count {
        let a = args[j]
        if a == "--check", j + 1 < args.count {
            check = args[j + 1]
            j += 2
            continue
        }
        if a.hasPrefix("http") { url = URL(string: a) }
        if a == "off" { on = false }
        j += 1
    }
    let client = url.map { AgentClient(base: $0) } ?? AgentClient()
    Task.detached {
        do {
            var report = AgentClient.ClickReport()
            report.on = on
            report.checkUrl = check
            let ph = try await client.clickPhone(report)
            let line = ph.click.map { MenuSummary.clickText($0) } ?? "the phone click left no state"
            print("phone: \(line)")
            if on {
                exit(ph.click?.on == true ? 0 : 1)
            }
            exit(0)
        } catch {
            print("phone: it didn't work: \(error.localizedDescription)")
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
