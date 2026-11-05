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
