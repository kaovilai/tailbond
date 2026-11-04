import Foundation
import XCTest
@testable import TailbondMenuKit

func fixture(_ name: String) throws -> AgentStatus {
    let url = Bundle.module.url(forResource: name, withExtension: "json", subdirectory: "Fixtures")!
    return try JSONDecoder().decode(AgentStatus.self, from: Data(contentsOf: url))
}

final class SummaryTests: XCTestCase {
    func testConnected() throws {
        let s = MenuSummary(try fixture("connected"))
        XCTAssertEqual(s.title, "Connected")
        XCTAssertEqual(s.subtitle, "to homebox · 2 links")
        XCTAssertEqual(s.tone, .good)
        XCTAssertEqual(s.rates, "↓ 23.4 Mbps  ↑ 1.2 Mbps")
        XCTAssertEqual(s.links.map(\.name), ["Wi-Fi (en0)", "iPhone (USB) (en7)", "Sam's phone (sh5a1b2c)"])
        XCTAssertEqual(s.links.map(\.detail), ["38 ms · ↓ 14.1 Mbps", "61 ms · ↓ 9.3 Mbps", "standby"])
        XCTAssertTrue(s.coexist.isEmpty)
        XCTAssertEqual(s.update.text, "tailbond 0.2.0 · up to date · Right away")
        XCTAssertEqual(s.updatePolicy, "immediate")
        XCTAssertNil(s.crash)
    }

    func testNetBird() throws {
        let s = MenuSummary(try fixture("netbird"))
        XCTAssertTrue(s.workVPNChoice)
        XCTAssertEqual(s.workVPN, "direct")
        XCTAssertEqual(s.coexist.map(\.text), [
            "NetBird is on (100.92.1.2/16): its traffic goes Direct (not through tailbond)",
            "Work names (corp.example, netbird.cloud) use the work VPN's DNS",
            "NetBird's network 10.20.0.0/16 is also your home LAN's: NetBird keeps it",
        ])
        XCTAssertEqual(s.coexist.map(\.warn), [false, false, true])
        XCTAssertEqual(s.links[1].detail, "not working")
        XCTAssertEqual(s.update.text, "tailbond 0.2.0 · up to date · Tell me")
    }

    func testCrashAndSecurityUpdate() throws {
        let s = MenuSummary(try fixture("crash-update"))
        XCTAssertEqual(s.crash, "2026-11-04T06:51:24Z")
        XCTAssertEqual(s.update.text, "Update available: 0.2.1 (security) (waiting for a quiet moment) · Right away")
        XCTAssertTrue(s.update.warn)
        XCTAssertTrue(s.text.contains("Report a problem"))
    }

    func testNotPairedAndUnreachable() throws {
        let s = MenuSummary(try fixture("not-paired"))
        XCTAssertEqual(s.title, "Not paired")
        XCTAssertEqual(s.tone, .off)
        let u = MenuSummary(nil)
        XCTAssertEqual(u.title, "tailbond isn't running")
        XCTAssertEqual(u.tone, .bad)
    }

    func testWords() {
        XCTAssertEqual(MenuSummary.policyLabel("immediate"), "Right away")
        XCTAssertEqual(MenuSummary.policyLabel("auto"), "Automatic (night)")
        XCTAssertEqual(MenuSummary.policyLabel(""), "Tell me")
        XCTAssertEqual(MenuSummary.rate(950), "0 kbps")
        XCTAssertEqual(MenuSummary.rate(64_000), "64 kbps")
    }

    /// The menu against a stand-in agent: the client's paths and JSON bodies.
    func testClientCalls() async throws {
        URLProtocol.registerClass(StubAgent.self)
        defer { URLProtocol.unregisterClass(StubAgent.self) }
        let cfg = URLSessionConfiguration.ephemeral
        cfg.protocolClasses = [StubAgent.self]
        let c = AgentClient(base: URL(string: "http://127.0.0.1:8077")!, session: URLSession(configuration: cfg))
        let st = try await c.status()
        XCTAssertEqual(st.state, "connected")
        try await c.setMode("redundant")
        try await c.setUpdatePolicy("auto")
        try await c.setWorkVPN("bonded")
        XCTAssertEqual(StubAgent.seen, [
            "GET /api/status", "POST /api/mode {\"mode\":\"redundant\"}", "POST /api/update/policy {\"policy\":\"auto\"}", "POST /api/coexist {\"work_vpn\":\"bonded\"}",
        ])
        XCTAssertEqual(c.report.absoluteString, "http://127.0.0.1:8077/#report")
    }
}

final class StubAgent: URLProtocol {
    static var seen: [String] = []
    override class func canInit(with request: URLRequest) -> Bool { true }
    override class func canonicalRequest(for request: URLRequest) -> URLRequest { request }
    override func startLoading() {
        var body = ""
        if let s = request.httpBodyStream {
            s.open()
            var buf = [UInt8](repeating: 0, count: 4096)
            let n = s.read(&buf, maxLength: buf.count)
            if n > 0 { body = String(decoding: buf[0..<n], as: UTF8.self) }
            s.close()
        } else if let b = request.httpBody {
            body = String(decoding: b, as: UTF8.self)
        }
        let m = request.httpMethod ?? "GET"
        StubAgent.seen.append("\(m) \(request.url!.path)" + (m == "GET" ? "" : " \(body)"))
        XCTAssertEqual(m == "GET" ? nil : request.value(forHTTPHeaderField: "Content-Type"), m == "GET" ? nil : "application/json")
        let data: Data
        if request.url!.path == "/api/status" {
            data = try! Data(contentsOf: Bundle.module.url(forResource: "connected", withExtension: "json", subdirectory: "Fixtures")!)
        } else {
            data = Data("{\"ok\":true}".utf8)
        }
        client?.urlProtocol(self, didReceive: HTTPURLResponse(url: request.url!, statusCode: 200, httpVersion: nil, headerFields: ["Content-Type": "application/json"])!, cacheStoragePolicy: .notAllowed)
        client?.urlProtocol(self, didLoad: data)
        client?.urlProtocolDidFinishLoading(self)
    }
    override func stopLoading() {}
}
