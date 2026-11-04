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
        XCTAssertEqual(s.subtitle, "to homebox · 2 of 3 links in use")
        XCTAssertEqual(s.tone, .good)
        XCTAssertEqual(s.rates, "↓ 23.4 Mbps  ↑ 1.2 Mbps")
        XCTAssertEqual(s.links.map(\.name), ["Wi-Fi (en0)", "iPhone (USB) (en7)", "Sam's phone, mobile data it shares (sh5a1b2c)"])
        XCTAssertEqual(s.links.map(\.detail), ["38 ms · ↓ 14.1 Mbps", "61 ms · ↓ 9.3 Mbps", "standby"])
        XCTAssertTrue(s.coexist.isEmpty)
        XCTAssertEqual(s.update.text, "tailbond 0.2.0 is up to date")
        XCTAssertFalse(s.canInstall)
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
        ])
        XCTAssertEqual(s.coexist.map(\.warn), [false, false])
        XCTAssertTrue(s.bondedBlocked.hasPrefix("Bonded isn't possible on this connection (en0)"))
        XCTAssertEqual(s.workExplain, [
            "Direct: the work VPN and its networks (100.92.0.0/16, 10.20.0.0/16), and its own connections to its servers and peers, use this Mac's own internet, as without tailbond.",
            "Bonded: they go through tailbond to homebox, then out to your company: steadier, but through your home.",
        ])
        XCTAssertEqual(s.links[1].detail, "can't reach homebox")
        XCTAssertEqual(s.update.text, "tailbond 0.2.0 is up to date")
    }

    func testCrashAndSecurityUpdate() throws {
        let s = MenuSummary(try fixture("crash-update"))
        XCTAssertNotNil(s.crash)
        // (newer ICU puts a narrow no-break space before AM: compare the parts)
        let w = MenuSummary.when("2026-11-04T06:51:24Z", timeZone: TimeZone(identifier: "UTC")!, locale: Locale(identifier: "en_US"))
        XCTAssertTrue(w.hasPrefix("Nov 4, 2026") && w.contains("6:51") && w.hasSuffix("AM"), w)
        XCTAssertEqual(MenuSummary.when("not a time"), "not a time")
        XCTAssertEqual(s.update.text, "Update available: tailbond 0.2.1 (you have 0.2.0). It installs by itself as soon as no call or big download is running (within 30 minutes); tailbond restarts, a few seconds without the tunnel.")
        XCTAssertFalse(s.update.warn)
        XCTAssertTrue(s.canInstall)
        XCTAssertTrue(s.text.contains("Report a problem"))
        XCTAssertTrue(s.text.contains("Install updates: Right away"), s.text)
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
