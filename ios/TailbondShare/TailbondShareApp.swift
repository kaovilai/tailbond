import SwiftUI

/// tailbond Share: lends this iPhone's mobile data to the travel router or a
/// laptop on the same Wi-Fi (Pair & Share, docs/ios.md). No VPN: it builds
/// with a free Apple ID.
@main
struct TailbondShareApp: App {
    @StateObject private var model = ShareModel()
    @Environment(\.scenePhase) private var phase

    var body: some Scene {
        WindowGroup {
            ContentView()
                .environmentObject(model)
                .onOpenURL { model.pair(text: $0.absoluteString) }
        }
        .onChange(of: phase) { model.scenePhase($0) }
    }
}
