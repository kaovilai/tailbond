import SwiftUI
import TailbondShareKit

/// Report a problem (docs/reporting.md): the redacted, editable report; the
/// notice; a required checkbox; then GitHub in Safari, the full text to the
/// clipboard, or a file. The app sends nothing.
struct ReportView: View {
    @Environment(\.dismiss) private var dismiss
    @Environment(\.openURL) private var openURL
    @State private var what = ""
    @State private var steps = ""
    @State private var version = CrashReporter.version
    @State private var device = Redactor.redact(CrashReporter.device)
    @State private var crash = Redactor.redact(CrashReporter.last()?.text ?? "")
    @State private var logs = ""
    @State private var ok = false
    @State private var copied = false
    private let crashed = CrashReporter.last()

    private var report: ProblemReport {
        ProblemReport(component: "iPhone Share app", version: version, device: device, what: what, steps: steps, logs: logs, crash: crash)
    }

    private var reportFile: URL {
        let name = ProblemReport.fileName(ISO8601DateFormatter().string(from: Date()))
        let url = FileManager.default.temporaryDirectory.appendingPathComponent(name)
        try? report.full().write(to: url, atomically: true, encoding: .utf8)
        return url
    }

    var body: some View {
        NavigationStack {
            Form {
                Section {
                    Text(ReportText.notice).font(.callout)
                    Link("A security problem? Report it privately instead", destination: URL(string: ReportText.securityURL)!)
                        .font(.footnote)
                }
                if let c = crashed {
                    Section {
                        Text("tailbond Share stopped unexpectedly (\(c.at.formatted(date: .abbreviated, time: .shortened))); its crash details are below.")
                            .font(.footnote).foregroundStyle(.red)
                    }
                }
                Section("What happened, and what did you expect?") {
                    TextField("Sharing stops after a minute; I expected it to keep going.", text: $what, axis: .vertical)
                        .lineLimit(3...8)
                        .accessibilityIdentifier("report-what")
                }
                Section("Steps to reproduce (optional)") {
                    TextField("1. …", text: $steps, axis: .vertical).lineLimit(2...6)
                }
                Section("tailbond version") { TextField("", text: $version) }
                Section("iPhone and iOS") { TextField("", text: $device) }
                if crashed != nil {
                    Section("Crash details") {
                        TextEditor(text: $crash).font(.caption.monospaced()).frame(minHeight: 120)
                    }
                }
                Section {
                    TextEditor(text: $logs).font(.caption.monospaced()).frame(minHeight: 140)
                } header: { Text("Logs (newest last)") } footer: {
                    Text("A link holds about 7 KB, so the issue gets the newest lines; Copy full report and Save report file have all of them.")
                }
                Section {
                    Toggle(ReportText.confirm, isOn: $ok).accessibilityIdentifier("report-confirm")
                }
                Section {
                    Button("Open GitHub issue") {
                        if let u = URL(string: report.issueURL()) { openURL(u) }
                        if crashed != nil { CrashReporter.dismiss() }
                    }
                    .disabled(!ok || what.trimmingCharacters(in: .whitespacesAndNewlines).isEmpty)
                    .accessibilityIdentifier("report-open")
                    Button(copied ? "Copied" : "Copy full report") {
                        UIPasteboard.general.string = report.full()
                        copied = true
                    }
                    ShareLink("Save report file", item: reportFile)
                }
            }
            .navigationTitle("Report a problem")
            .toolbar { ToolbarItem(placement: .cancellationAction) { Button("Close") { dismiss() } } }
            .task { logs = Redactor.redact(CrashReporter.recentLogs()) }
        }
    }
}

/// Shown on the main screens after a crash.
struct CrashBanner: View {
    @State private var crashed = CrashReporter.last()
    let onReport: () -> Void

    var body: some View {
        Group {
            if let c = crashed { banner(c) }
        }
        .onReceive(NotificationCenter.default.publisher(for: .crashRecorded)) { _ in crashed = CrashReporter.last() }
    }

    private func banner(_ c: (text: String, at: Date)) -> some View {
            Section {
                VStack(alignment: .leading, spacing: 8) {
                    Text("tailbond Share stopped unexpectedly").font(.headline)
                    Text("It happened \(c.at.formatted(date: .abbreviated, time: .shortened)). Reporting it helps fix it.").font(.footnote)
                    HStack {
                        Button("Report a problem", action: onReport).buttonStyle(.borderedProminent)
                        Button("Dismiss") { CrashReporter.dismiss(); crashed = nil }
                    }
                }
            }
    }
}
