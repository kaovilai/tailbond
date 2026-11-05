import Foundation

/// The local tailbondd's API (the dashboard's, 127.0.0.1:8077 on a Mac).
public struct AgentClient {
    public var base: URL
    public var session: URLSession

    public init(base: URL = URL(string: UserDefaults.standard.string(forKey: "agentURL") ?? "http://127.0.0.1:8077")!,
                session: URLSession = .shared) {
        self.base = base
        self.session = session
    }

    public enum Failure: Error, Equatable {
        case unreachable
        case needsLogin
        case http(Int, String)
    }

    private func call(_ method: String, _ path: String, _ body: [String: Any]? = nil) async throws -> Data {
        var req = URLRequest(url: base.appendingPathComponent(path), timeoutInterval: 4)
        req.httpMethod = method
        if method != "GET" {
            // the API only takes JSON-typed changes (its CSRF guard)
            req.setValue("application/json", forHTTPHeaderField: "Content-Type")
            req.httpBody = try JSONSerialization.data(withJSONObject: body ?? [:])
        }
        let data: Data, resp: URLResponse
        do { (data, resp) = try await session.data(for: req) } catch { throw Failure.unreachable }
        let code = (resp as? HTTPURLResponse)?.statusCode ?? 0
        if code == 401 { throw Failure.needsLogin }
        guard (200..<300).contains(code) else {
            let msg = (try? JSONSerialization.jsonObject(with: data) as? [String: Any])?["error"] as? String ?? ""
            throw Failure.http(code, msg)
        }
        return data
    }

    public func status() async throws -> AgentStatus {
        try JSONDecoder().decode(AgentStatus.self, from: try await call("GET", "/api/status"))
    }

    /// What the menu bar reports to tailbondd about its halves of the one
    /// click (Bluetooth, the Wi-Fi join); both optional — without them
    /// this is the no-Bluetooth fallback.
    public struct ClickReport: Encodable {
        public struct BLE: Encodable {
            public var state = "" // ok, failed
            public var step = "" // scan, auth, hotspot
            public var error = ""
            public var keyId = ""
            public var label = ""
            public var ssid = ""
            public var pass = ""
            public var net = ""
            public var battery = 0
            public var charging = false
            public init() {}
            enum CodingKeys: String, CodingKey {
                case state, step, error, label, ssid, pass, net, battery, charging
                case keyId = "key_id"
            }
        }
        public struct WiFi: Encodable {
            public var ok = false
            public var ssid = ""
            public var error = ""
            public init() {}
            enum CodingKeys: String, CodingKey { case ok, ssid, error }
        }
        public var on = true
        public var ble: BLE?
        public var wifi: WiFi?
        public var checkUrl: String?
        public init() {}
        enum CodingKeys: String, CodingKey {
            case on, ble, wifi
            case checkUrl = "check_url"
        }
    }

    /// The one click: tailbondd turns the phone hub on (and checks the
    /// internet through it), or off and the system proxy back.
    public func clickPhone(_ report: ClickReport) async throws -> PhoneHubInfo {
        let data = try JSONEncoder().encode(report)
        var req = URLRequest(url: base.appendingPathComponent("/api/phone-hub/click"), timeoutInterval: 90)
        req.httpMethod = "POST"
        req.setValue("application/json", forHTTPHeaderField: "Content-Type")
        req.httpBody = data
        let (body, resp): (Data, URLResponse)
        do { (body, resp) = try await session.data(for: req) } catch { throw Failure.unreachable }
        let code = (resp as? HTTPURLResponse)?.statusCode ?? 0
        if code == 401 { throw Failure.needsLogin }
        guard (200..<300).contains(code) else {
            let msg = (try? JSONSerialization.jsonObject(with: body) as? [String: Any])?["error"] as? String ?? ""
            throw Failure.http(code, msg)
        }
        return try JSONDecoder().decode(PhoneHubInfo.self, from: body)
    }

    /// The phone hub list (GET /api/phone-hub), which carries the BLE
    /// logins to a loopback caller — this app — for the one click.
    public func phoneHub() async throws -> PhoneHubInfo {
        try JSONDecoder().decode(PhoneHubInfo.self, from: try await call("GET", "/api/phone-hub"))
    }

    public func setMode(_ m: String) async throws { _ = try await call("POST", "/api/mode", ["mode": m]) }
    public func connect(_ on: Bool) async throws { _ = try await call("POST", on ? "/api/connect" : "/api/disconnect") }
    public func setUpdatePolicy(_ p: String) async throws { _ = try await call("POST", "/api/update/policy", ["policy": p]) }
    public func checkUpdate() async throws { _ = try await call("POST", "/api/update/check") }
    public func installUpdate() async throws { _ = try await call("POST", "/api/update/install") }
    public func setWorkVPN(_ w: String) async throws { _ = try await call("POST", "/api/coexist", ["work_vpn": w]) }
    public func dismissCrash() async throws { _ = try await call("POST", "/api/report/crash/dismiss") }
    /// "Share this Mac's connection" on or off, as set up in the dashboard.
    public func setSharing(_ on: Bool) async throws { _ = try await call("POST", "/api/share", ["share_enabled": on]) }

    public var dashboard: URL { base }
    /// The dashboard's Report a problem (docs/reporting.md).
    public var report: URL { URL(string: base.absoluteString + "/#report")! }
    /// The dashboard's "Share this Mac's connection" choices.
    public var shareSettings: URL { URL(string: base.absoluteString + "/#share")! }
}
