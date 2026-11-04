import Foundation
import MetricKit
import OSLog
import TailbondShareKit
import UIKit

/// Crash capture for tailbond Share (docs/reporting.md): Objective-C
/// exceptions are written as they happen; every other crash (a Swift trap,
/// a signal) arrives from MetricKit on the next launch. Both are kept in
/// Application Support/crash-last.txt until the user reports or dismisses
/// them. Nothing leaves the phone unless the user opens the report.
final class CrashReporter: NSObject, MXMetricManagerSubscriber {
    static let shared = CrashReporter()
    static let log = Logger(subsystem: "dev.tailbond.share", category: "share")

    static var file: URL {
        let dir = FileManager.default.urls(for: .applicationSupportDirectory, in: .userDomainMask)[0]
        try? FileManager.default.createDirectory(at: dir, withIntermediateDirectories: true)
        return dir.appendingPathComponent("crash-last.txt")
    }

    func install() {
        _ = Self.file // the path is resolved before a crash needs it
        NSSetUncaughtExceptionHandler { e in
            let text = "tailbond Share \(CrashReporter.version) stopped: \(e.name.rawValue): \(e.reason ?? "")\n" +
                e.callStackSymbols.joined(separator: "\n")
            try? text.write(to: CrashReporter.file, atomically: true, encoding: .utf8)
        }
        MXMetricManager.shared.add(self)
    }

    func didReceive(_ payloads: [MXDiagnosticPayload]) {
        guard let crash = payloads.compactMap({ $0.crashDiagnostics?.last }).last else { return }
        let meta = crash.metaData
        var text = "tailbond Share \(meta.applicationBuildVersion) crashed (MetricKit)"
        if let sig = crash.signal { text += ", signal \(sig)" }
        if let t = crash.exceptionType { text += ", exception type \(t)" }
        if let r = crash.terminationReason { text += ", \(r)" }
        let tree = String(decoding: crash.callStackTree.jsonRepresentation(), as: UTF8.self)
        text += "\n" + String(tree.prefix(48 * 1024))
        try? text.write(to: Self.file, atomically: true, encoding: .utf8)
        DispatchQueue.main.async { NotificationCenter.default.post(name: .crashRecorded, object: nil) }
    }

    /// The kept crash and when it was written, or nil.
    static func last() -> (text: String, at: Date)? {
        guard let text = try? String(contentsOf: file, encoding: .utf8), !text.isEmpty else { return nil }
        let at = (try? FileManager.default.attributesOfItem(atPath: file.path)[.modificationDate] as? Date) ?? Date()
        return (text, at)
    }

    static func dismiss() { try? FileManager.default.removeItem(at: file) }

    static var version: String {
        let i = Bundle.main.infoDictionary ?? [:]
        return "\(i["CFBundleShortVersionString"] as? String ?? "?") (\(i["CFBundleVersion"] as? String ?? "?"))"
    }

    static var device: String {
        var u = utsname()
        uname(&u)
        let machine = withUnsafeBytes(of: &u.machine) { raw in String(decoding: raw.prefix { $0 != 0 }, as: UTF8.self) }
        return "\(machine), iOS \(UIDevice.current.systemVersion)"
    }

    /// This app's own log lines from the last hour (OSLogStore, iOS 15+).
    static func recentLogs(limit: Int = 300) -> String {
        guard let store = try? OSLogStore(scope: .currentProcessIdentifier) else { return "" }
        let since = store.position(date: Date().addingTimeInterval(-3600))
        guard let entries = try? store.getEntries(at: since) else { return "" }
        let f = DateFormatter()
        f.dateFormat = "HH:mm:ss"
        var lines: [String] = []
        for case let e as OSLogEntryLog in entries where e.subsystem.hasPrefix("dev.tailbond") {
            lines.append("\(f.string(from: e.date)) [\(e.category)] \(e.composedMessage)")
        }
        return lines.suffix(limit).joined(separator: "\n")
    }
}

extension Notification.Name {
    static let crashRecorded = Notification.Name("tailbond.crashRecorded")
}
