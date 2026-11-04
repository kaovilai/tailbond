import AppKit
import SwiftUI

/// The menu bar window (MenuBarExtra, window style): state, on/off, mode,
/// links, other VPNs (coexist), updates, crash, and the way to the
/// dashboard and Report a problem.
public struct MenuContent: View {
    @ObservedObject var model: MenuModel
    var openURL: (URL) -> Void
    var quit: () -> Void
    var loginItem: Binding<Bool>?

    public init(model: MenuModel, openURL: @escaping (URL) -> Void = { _ = NSWorkspace.shared.open($0) },
                quit: @escaping () -> Void = { NSApp.terminate(nil) }, loginItem: Binding<Bool>? = nil) {
        self.model = model
        self.openURL = openURL
        self.quit = quit
        self.loginItem = loginItem
    }

    private var s: MenuSummary { model.summary }

    private func color(_ t: MenuSummary.Tone) -> Color {
        switch t {
        case .good: return Brand.ok
        case .busy: return Brand.warn
        case .off: return .secondary
        case .bad: return Brand.bad
        }
    }

    public var body: some View {
        VStack(alignment: .leading, spacing: 10) {
            header
            if let c = s.crash { crashBox(c) }
            if model.status != nil {
                modes
                if !s.links.isEmpty { section("Connections") { linksList } }
                if !s.coexist.isEmpty { section("Next to other VPNs") { coexistBox } }
                section("Updates") { updatesBox }
            }
            if let m = model.message {
                Text(m).font(.caption).foregroundStyle(Brand.bad).fixedSize(horizontal: false, vertical: true)
            }
            Divider()
            footer
        }
        .padding(14)
        .frame(width: 340)
    }

    private var header: some View {
        HStack(alignment: .top, spacing: 10) {
            Image(systemName: s.symbol)
                .font(.title2)
                .foregroundStyle(color(s.tone))
                .frame(width: 28)
            VStack(alignment: .leading, spacing: 2) {
                Text(s.title).font(.headline)
                if !s.subtitle.isEmpty {
                    Text(s.subtitle).font(.caption).foregroundStyle(.secondary).fixedSize(horizontal: false, vertical: true)
                }
                if !s.rates.isEmpty { Text(s.rates).font(.caption.monospacedDigit()).foregroundStyle(.secondary) }
            }
            Spacer()
            if model.status?.paired == true {
                Toggle("", isOn: Binding(get: { s.connected || model.status?.state == "connecting" }, set: { model.setOn($0) }))
                    .toggleStyle(.switch)
                    .labelsHidden()
                    .disabled(model.busy)
                    .accessibilityLabel("tailbond on")
            }
        }
    }

    private var modes: some View {
        Picker("Mode", selection: Binding(get: { model.status?.mode ?? "" }, set: { model.setMode($0) })) {
            ForEach(MenuSummary.modes, id: \.wire) { Text($0.label).tag($0.wire) }
        }
        .pickerStyle(.menu)
        .disabled(model.busy || model.status?.paired != true)
    }

    private var linksList: some View {
        VStack(alignment: .leading, spacing: 4) {
            ForEach(s.links) { l in
                HStack {
                    Circle().fill(color(l.tone)).frame(width: 7, height: 7)
                    Text(l.name).lineLimit(1)
                    Spacer()
                    Text(l.detail).font(.caption).foregroundStyle(.secondary)
                }
                .font(.callout)
            }
        }
    }

    private var coexistBox: some View {
        VStack(alignment: .leading, spacing: 6) {
            ForEach(s.coexist) { l in
                Label {
                    Text(l.text).fixedSize(horizontal: false, vertical: true)
                } icon: {
                    Image(systemName: l.warn ? "exclamationmark.triangle.fill" : "checkmark.shield")
                        .foregroundStyle(l.warn ? Brand.warn : Brand.ok)
                }
                .font(.caption)
            }
            if s.workVPNChoice {
                Picker("Work VPN traffic", selection: Binding(get: { s.workVPN }, set: { model.setWorkVPN($0) })) {
                    Text("Direct").tag("direct")
                    Text("Bonded").tag("bonded")
                }
                .pickerStyle(.segmented)
                .disabled(model.busy)
            }
        }
    }

    private var updatesBox: some View {
        VStack(alignment: .leading, spacing: 6) {
            Label {
                Text(s.update.text).fixedSize(horizontal: false, vertical: true)
            } icon: {
                Image(systemName: s.update.warn ? "exclamationmark.triangle.fill" : "arrow.down.circle")
                    .foregroundStyle(s.update.warn ? Brand.warn : .secondary)
            }
            .font(.caption)
            HStack {
                Picker("Install updates", selection: Binding(get: { s.updatePolicy }, set: { model.setPolicy($0) })) {
                    ForEach(MenuSummary.policies, id: \.wire) { Text($0.label).tag($0.wire) }
                }
                .pickerStyle(.menu)
                .disabled(model.busy)
                Button("Check now") { model.checkUpdate() }.disabled(model.busy)
            }
        }
    }

    private func crashBox(_ at: String) -> some View {
        VStack(alignment: .leading, spacing: 6) {
            Text("tailbond stopped unexpectedly and restarted (\(at)).").font(.caption).fixedSize(horizontal: false, vertical: true)
            HStack {
                Button("Report a problem…") { openURL(model.client.report) }
                Button("Dismiss") { model.dismissCrash() }
            }
            .controlSize(.small)
        }
        .padding(8)
        .background(RoundedRectangle(cornerRadius: 8).fill(Brand.warn.opacity(0.14)))
    }

    private var footer: some View {
        VStack(alignment: .leading, spacing: 6) {
            HStack {
                Button("Open dashboard") { openURL(model.client.dashboard) }
                Button("Report a problem…") { openURL(model.client.report) }
                Spacer()
                Button("Quit") { quit() }
            }
            if let li = loginItem {
                Toggle("Open at login", isOn: li).toggleStyle(.checkbox).font(.caption)
            }
        }
        .controlSize(.small)
    }

    private func section<C: View>(_ title: String, @ViewBuilder _ content: () -> C) -> some View {
        VStack(alignment: .leading, spacing: 6) {
            Text(title.uppercased()).font(.caption2.weight(.semibold)).foregroundStyle(.secondary)
            content()
        }
    }
}

/// Brand colours (assets/brand/tokens.json).
enum Brand {
    static let ok = Color(red: 0x0D / 255, green: 0x94 / 255, blue: 0x88 / 255)
    static let warn = Color(red: 0xD9 / 255, green: 0x77 / 255, blue: 0x06 / 255)
    static let bad = Color(red: 0xDC / 255, green: 0x26 / 255, blue: 0x26 / 255)
}
