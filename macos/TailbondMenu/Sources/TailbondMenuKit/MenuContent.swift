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
                if let p = s.phone { section("Phone") { phoneBox(p) } }
                if let sh = s.share { section("Share this Mac's connection") { shareBox(sh) } }
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
                let on = s.connected || model.status?.state == "connecting"
                VStack(alignment: .trailing, spacing: 2) {
                    Toggle("", isOn: Binding(get: { on }, set: { model.setOn($0) }))
                        .toggleStyle(.switch)
                        .labelsHidden()
                        .tint(Brand.ok)
                        .disabled(model.busy)
                        .accessibilityLabel("tailbond on")
                    Text(on ? "On" : "Off").font(.caption2.weight(.semibold)).foregroundStyle(on ? Brand.ok : .secondary)
                }
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
                if s.bondedBlocked.isEmpty {
                    Picker("Work VPN traffic", selection: Binding(get: { s.workVPN }, set: { model.setWorkVPN($0) })) {
                        Text("Direct").tag("direct")
                        Text("Bonded").tag("bonded")
                    }
                    .pickerStyle(.segmented)
                    .disabled(model.busy)
                }
                Text("Work VPN traffic now: \(s.workVPN == "bonded" ? "Bonded" : "Direct")").font(.caption.weight(.semibold))
                if !s.bondedBlocked.isEmpty {
                    Label { Text(s.bondedBlocked).fixedSize(horizontal: false, vertical: true) } icon: {
                        Image(systemName: "info.circle").foregroundStyle(.secondary)
                    }
                    .font(.caption)
                }
                ForEach(s.bondedBlocked.isEmpty ? s.workExplain : Array(s.workExplain.prefix(1)), id: \.self) {
                    Text($0).font(.caption).foregroundStyle(.secondary).fixedSize(horizontal: false, vertical: true)
                }
            }
        }
    }

    /// "Connect via phone": one click, the phone's hotspot over Bluetooth and its bonded connection.
    private func phoneBox(_ p: MenuSummary.Phone) -> some View {
        let stage = model.phoneStage
        return VStack(alignment: .leading, spacing: 6) {
            HStack(alignment: .top, spacing: 8) {
                Image(systemName: p.connected ? "iphone.radiowaves.left.and.right" : "iphone")
                    .foregroundStyle(stage != nil ? Brand.warn : color(p.tone))
                VStack(alignment: .leading, spacing: 2) {
                    Text(stage != nil ? "Connecting via phone…" : p.title).font(.callout.weight(.semibold))
                    Text(stage ?? p.detail).font(.caption).foregroundStyle(p.tone == .bad && stage == nil ? Brand.bad : .secondary)
                        .fixedSize(horizontal: false, vertical: true)
                }
            }
            if stage == nil && !p.action.isEmpty {
                HStack {
                    Button(p.action) { p.connected ? model.disconnectPhone() : model.connectViaPhone() }
                        .disabled(model.busy)
                    if p.connected {
                        Button("…and turn off its hotspot") { model.disconnectPhone(hotspotOff: true) }
                            .disabled(model.busy)
                    }
                }
                .controlSize(.small)
            }
        }
    }

    private func shareBox(_ sh: MenuSummary.Share) -> some View {
        VStack(alignment: .leading, spacing: 6) {
            HStack(alignment: .top) {
                Text(sh.text).font(.caption).foregroundStyle(sh.warn ? Brand.warn : .secondary).fixedSize(horizontal: false, vertical: true)
                Spacer()
                if sh.canToggle {
                    Toggle("", isOn: Binding(get: { sh.on }, set: { model.setSharing($0) }))
                        .toggleStyle(.switch)
                        .labelsHidden()
                        .controlSize(.small)
                        .tint(Brand.ok)
                        .disabled(model.busy)
                        .accessibilityLabel("Share this Mac's connection")
                }
            }
            Button(sh.canToggle ? "Sharing settings…" : "Choose a connection…") { openURL(model.client.shareSettings) }
                .controlSize(.small)
        }
    }

    private var updatesBox: some View {
        VStack(alignment: .leading, spacing: 6) {
            Label {
                Text(s.update.text).fixedSize(horizontal: false, vertical: true)
            } icon: {
                Image(systemName: s.update.warn ? "exclamationmark.triangle.fill" : (s.update.text.hasPrefix("Update available") ? "arrow.down.circle" : "checkmark.circle"))
                    .foregroundStyle(s.update.warn ? Brand.warn : .secondary)
            }
            .font(.caption)
            HStack {
                Picker("Install updates", selection: Binding(get: { s.updatePolicy }, set: { model.setPolicy($0) })) {
                    ForEach(MenuSummary.policies, id: \.wire) { Text($0.label).tag($0.wire) }
                }
                .pickerStyle(.menu)
                .disabled(model.busy)
                if s.canInstall {
                    Button("Install now") { model.installUpdate() }.disabled(model.busy)
                } else {
                    Button("Check now") { model.checkUpdate() }.disabled(model.busy)
                }
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
                // after a crash the box above has it
                if s.crash == nil { Button("Report a problem…") { openURL(model.client.report) } }
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
