import SwiftUI
import TailbondShareKit

struct ContentView: View {
    @EnvironmentObject var model: ShareModel

    var body: some View {
        NavigationStack {
            if model.pairing == nil { PairView() } else { ShareView() }
        }
    }
}

/// First run: get the pairing from the home server (QR code or link).
struct PairView: View {
    @EnvironmentObject var model: ShareModel
    @State private var text = ""
    @State private var scanning = false
    @State private var reporting = false

    var body: some View {
        Form {
            CrashBanner { reporting = true }
            Section {
                Text("tailbond Share lends this iPhone's mobile data to your travel router or laptop running tailbond, so it can use it next to its other connections.")
                Text("On your home server's page, choose **Devices → Add a device**, then scan the code here or open the link on this iPhone. Not sure where that page is? Ask whoever set up your home server to send you the link.")
                    .foregroundStyle(.secondary)
            }
            Section {
                Button { scanning = true } label: { Label("Scan the code", systemImage: "qrcode.viewfinder") }
                    .disabled(!QRScannerView.isAvailable)
            }
            Section("Or paste the link") {
                TextField("tailbond://pair?d=…", text: $text, axis: .vertical)
                    .textInputAutocapitalization(.never)
                    .autocorrectionDisabled()
                Button("Add") { model.pair(text: text) }.disabled(text.isEmpty)
            }
            if let e = model.pairError {
                Section { Text(e).foregroundStyle(.red) }
            }
        }
        .navigationTitle("tailbond Share")
        .toolbar { ToolbarItem(placement: .primaryAction) { ReportButton(reporting: $reporting) } }
        .sheet(isPresented: $reporting) { ReportView() }
        .sheet(isPresented: $scanning) {
            QRScannerView { code in
                scanning = false
                model.pair(text: code)
            }
            .ignoresSafeArea()
        }
    }
}

/// The share screen: one big button, what's happening, and the limits.
struct ShareView: View {
    @EnvironmentObject var model: ShareModel
    @Environment(\.colorScheme) private var scheme
    @State private var newHub = ""
    @State private var notice = false
    @State private var reporting = false

    var body: some View {
        let p = BrandPalette.of(scheme)
        List {
            CrashBanner { reporting = true }
            Section {
                Label {
                    Text("**Keep tailbond Share on the screen while sharing.** iOS pauses sharing as soon as you switch apps or lock the phone, and it starts again when you come back. The screen stays on while sharing; plug the phone in for long trips.")
                } icon: { Image(systemName: "exclamationmark.triangle.fill").foregroundStyle(p.warn) }
                    .font(.callout)
            }
            Section {
                VStack(spacing: 16) {
                    Button {
                        if model.sharing { model.stop() } else if !model.screenNoticeSeen { notice = true } else { model.start() }
                    } label: {
                        Text(model.sharing ? "Stop sharing" : "Share mobile data")
                            .font(.title2.bold())
                            .frame(maxWidth: .infinity, minHeight: 64)
                    }
                    .buttonStyle(BeamButtonStyle(stop: model.sharing))
                    Text(model.headline).font(.headline).multilineTextAlignment(.center)
                    Text("\(ShareModel.bytes(model.usedThisMonth)) shared this month").font(.footnote).foregroundStyle(.secondary)
                }
                .padding(.vertical, 8)
            }
            Section {
                Text("Connect this iPhone to your travel router's own Wi-Fi (the one it makes), not the hotel's.")
                    .font(.footnote).foregroundStyle(.secondary)
            }
            Section("Who's using it") {
                if model.status.hubs.isEmpty {
                    Text(model.sharing ? "Nobody yet. Your travel router starts using it by itself once this iPhone is on the router's own Wi-Fi." : "Nobody.")
                        .foregroundStyle(.secondary)
                }
                ForEach(model.status.hubs, id: \.endpoint) { h in
                    HStack {
                        Image(systemName: "wifi.router").foregroundStyle(p.shared)
                        Text(h.hub)
                        Spacer()
                        Text(ShareModel.bytes(h.upBytes + h.downBytes)).foregroundStyle(.secondary)
                    }
                }
            }
            Section {
                Toggle("Share mobile data with a Mac", isOn: Binding(get: { model.proxyOn }, set: { $0 ? model.startProxy() : model.stopProxy() }))
                if model.proxyOn {
                    Text(model.proxyStatus.clients.isEmpty ? "Ready. On the Mac: tailbond → Use a phone's connection → Use."
                         : "\(model.proxyStatus.clients.count) device(s): \(ShareModel.bytes(model.proxyStatus.up + model.proxyStatus.down))")
                }
                if let e = model.proxyStatus.error { Text(e).foregroundStyle(.red) }
            } header: { Text("For a Mac (experimental)") } footer: {
                Text("Better: plug the iPhone into the Mac with a cable and turn on Personal Hotspot. tailbond on the Mac then bonds it with the Wi-Fi, and you don't need this. This switch only lends mobile data, not bonded, and it stops when you leave this screen or lock the iPhone.")
            }
            Section {
                Picker("Stop below", selection: $model.minBattery) {
                    Text("Never").tag(0)
                    ForEach([10, 20, 30, 50], id: \.self) { Text("\($0)% battery").tag($0) }
                }
            } header: { Text("Battery") } footer: { Text("Unless the iPhone is charging.") }
            Section("Monthly limit") {
                Picker("Limit", selection: $model.capMB) {
                    Text("No limit").tag(0)
                    ForEach([500, 1000, 2000, 5000, 10000], id: \.self) { mb in
                        Text(mb < 1000 ? "\(mb) MB" : "\(mb / 1000) GB").tag(mb)
                    }
                }
            }
            Section {
                ForEach(model.allowedHubs, id: \.self) { h in
                    Text(h).swipeActions { Button("Remove", role: .destructive) { model.setAllowedHubs(model.allowedHubs.filter { $0 != h }) } }
                }
                HStack {
                    TextField("Device name, e.g. slate7", text: $newHub).textInputAutocapitalization(.never)
                    Button("Add") {
                        let n = newHub.trimmingCharacters(in: .whitespaces)
                        if !n.isEmpty, !model.allowedHubs.contains(n) { model.setAllowedHubs(model.allowedHubs + [n]) }
                        newHub = ""
                    }.disabled(newHub.isEmpty)
                }
            } header: { Text("Who can use it") } footer: {
                Text(model.allowedHubs.isEmpty ? "Any device paired with \(model.pairing?.label ?? "your home server")." : "Only these devices.")
            }
            Section {
                Button("Forget \(model.pairing?.label ?? "home server")", role: .destructive) { model.forget() }
            }
        }
        .scrollContentBackground(.hidden)
        .background(p.bg)
        .navigationTitle("Share mobile data")
        .toolbar { ToolbarItem(placement: .primaryAction) { ReportButton(reporting: $reporting) } }
        .sheet(isPresented: $reporting) { ReportView() }
        .alert("Keep this app on the screen", isPresented: $notice) {
            Button("Share mobile data") {
                model.screenNoticeSeen = true
                model.start()
            }
            Button("Cancel", role: .cancel) {}
        } message: {
            Text("iOS pauses sharing as soon as you switch apps or lock the iPhone, and it starts again when you come back. Leave tailbond Share open, and plug the iPhone in: the screen stays on while sharing.")
        }
    }
}

/// Report a problem (docs/reporting.md), in every screen's toolbar.
struct ReportButton: View {
    @Binding var reporting: Bool
    var body: some View {
        Button { reporting = true } label: { Label("Report a problem", systemImage: "exclamationmark.bubble") }
            .accessibilityIdentifier("report-button")
    }
}
