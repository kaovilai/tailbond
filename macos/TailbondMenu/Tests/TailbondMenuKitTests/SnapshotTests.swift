import AppKit
import SwiftUI
import XCTest
@testable import TailbondMenuKit

/// The menu window as PNGs (CI artifact menubar-screens, the persona
/// rounds' input): TB_SNAPSHOT_DIR=<dir> swift test. Rendered through an
/// NSHostingView in an offscreen window, so AppKit controls (pickers,
/// switches) draw as they do in the menu.
@MainActor
final class SnapshotTests: XCTestCase {
    func render(_ name: String, _ model: MenuModel, dark: Bool, dir: URL) throws {
        _ = NSApplication.shared
        let appearance = NSAppearance(named: dark ? .darkAqua : .aqua)!
        let root = MenuContent(model: model, openURL: { _ in }, quit: {}, loginItem: .constant(true))
            .background(Color(nsColor: .windowBackgroundColor))
            .environment(\.colorScheme, dark ? .dark : .light)
        let host = NSHostingView(rootView: root)
        host.appearance = appearance
        let size = host.fittingSize
        let win = NSWindow(contentRect: NSRect(origin: .zero, size: size), styleMask: [.borderless], backing: .buffered, defer: false)
        win.appearance = appearance
        win.contentView = host
        host.frame = NSRect(origin: .zero, size: size)
        host.layoutSubtreeIfNeeded()
        RunLoop.current.run(until: Date().addingTimeInterval(0.3))
        let rep = try XCTUnwrap(host.bitmapImageRepForCachingDisplay(in: host.bounds))
        host.cacheDisplay(in: host.bounds, to: rep)
        let png = try XCTUnwrap(rep.representation(using: .png, properties: [:]))
        try png.write(to: dir.appendingPathComponent("\(name)\(dark ? "-dark" : "").png"))
        XCTAssertGreaterThan(size.height, 100, name)
    }

    func testSnapshots() throws {
        guard let d = ProcessInfo.processInfo.environment["TB_SNAPSHOT_DIR"], !d.isEmpty else {
            throw XCTSkip("set TB_SNAPSHOT_DIR to render")
        }
        let dir = URL(fileURLWithPath: d)
        try FileManager.default.createDirectory(at: dir, withIntermediateDirectories: true)
        let cases: [(String, AgentStatus?)] = [
            ("menu-01-connected", try fixture("connected")),
            ("menu-02-netbird", try fixture("netbird")),
            ("menu-03-crash-update", try fixture("crash-update")),
            ("menu-04-not-paired", try fixture("not-paired")),
            ("menu-05-not-running", nil),
            ("menu-06-sharing", try fixture("sharing")),
        ]
        for (name, st) in cases {
            for dark in [false, true] {
                try render(name, MenuModel(fixed: st), dark: dark, dir: dir)
            }
        }
    }
}
