import Foundation
import XCTest
@testable import TailbondShareKit

/// The phone hub's proxy (docs/phone-hub.md) against the Go reference
/// (agent/internal/proxy) and, on macOS, real curl.
final class ProxyTests: XCTestCase {
    func testCredentialsMatchTheGoReference() throws {
        var root = URL(fileURLWithPath: #filePath)
        for _ in 0..<5 { root.deleteLastPathComponent() }
        let data = try Data(contentsOf: root.appendingPathComponent("agent/internal/proxy/testdata/vectors.json"))
        let obj = try JSONSerialization.jsonObject(with: data) as! [String: Any]
        let vs = obj["credentials"] as! [[String: String]]
        XCTAssertGreaterThanOrEqual(vs.count, 3)
        for v in vs {
            let c = ProxyCore.credentials(shareKey: Array(v["share_key"]!.utf8))
            XCTAssertEqual(c.user, v["user"])
            XCTAssertEqual(c.pass, v["pass"])
        }
        XCTAssertEqual(ProxyCore.base32(Array("foobar".utf8)), "MZXW6YTBOI")
        XCTAssertEqual(ProxyCore.base32(Array("f".utf8)), "MY")
    }

    func testSocksHandshake() {
        var hs = ProxyCore.Handshake(user: "tailbond", pass: "pw")
        XCTAssertEqual(hs.feed([5, 1]), .more)
        XCTAssertEqual(hs.feed([2]), .reply([5, 2]))
        XCTAssertEqual(hs.feed([1, 8] + Array("tailbond".utf8) + [2] + Array("pw".utf8)), .reply([1, 0]))
        let req: [UInt8] = [5, 1, 0, 3, 9] + Array("10.77.0.1".utf8) + [0x1f, 0x90] + Array("GET".utf8)
        guard case let .connect(h, p, ok, _, rest) = hs.feed(req) else { return XCTFail("no connect") }
        XCTAssertEqual(h, "10.77.0.1")
        XCTAssertEqual(p, 8080)
        XCTAssertEqual(ok[1], 0)
        XCTAssertEqual(rest, Array("GET".utf8))

        var ipv4 = ProxyCore.Handshake(user: "", pass: "")
        XCTAssertEqual(ipv4.feed([5, 1, 0]), .reply([5, 0]))
        guard case let .connect(h4, p4, _, _, _) = ipv4.feed([5, 1, 0, 1, 192, 168, 1, 9, 0, 80]) else { return XCTFail() }
        XCTAssertEqual(h4, "192.168.1.9")
        XCTAssertEqual(p4, 80)

        var wrong = ProxyCore.Handshake(user: "tailbond", pass: "pw")
        _ = wrong.feed([5, 1, 2])
        XCTAssertEqual(wrong.feed([1, 8] + Array("tailbond".utf8) + [3] + Array("bad".utf8)), .refuse([1, 1]))

        var noAuth = ProxyCore.Handshake(user: "tailbond", pass: "pw")
        XCTAssertEqual(noAuth.feed([5, 1, 0]), .refuse([5, 0xff]))

        var udp = ProxyCore.Handshake(user: "", pass: "")
        _ = udp.feed([5, 1, 0])
        XCTAssertEqual(udp.feed([5, 3, 0, 1, 0, 0, 0, 0, 0, 0]), .refuse([5, 7, 0, 1, 0, 0, 0, 0, 0, 0]))
    }

    func testHTTPConnect() {
        var none = ProxyCore.Handshake(user: "tailbond", pass: "pw")
        guard case let .refuse(b) = none.feed(Array("CONNECT a:443 HTTP/1.1\r\nHost: a\r\n\r\n".utf8)) else { return XCTFail() }
        XCTAssertTrue(String(decoding: b, as: UTF8.self).contains("407"))
        var ok = ProxyCore.Handshake(user: "tailbond", pass: "pw")
        let auth = Data("tailbond:pw".utf8).base64EncodedString()
        guard case let .connect(h, p, reply, _, _) = ok.feed(Array("CONNECT [fd00::1]:8443 HTTP/1.1\r\nProxy-Authorization: Basic \(auth)\r\n\r\n".utf8)) else {
            return XCTFail()
        }
        XCTAssertEqual(h, "fd00::1")
        XCTAssertEqual(p, 8443)
        XCTAssertTrue(String(decoding: reply, as: UTF8.self).contains("200"))
        var get = ProxyCore.Handshake(user: "", pass: "")
        guard case let .refuse(g) = get.feed(Array("GET / HTTP/1.1\r\n\r\n".utf8)) else { return XCTFail() }
        XCTAssertTrue(String(decoding: g, as: UTF8.self).contains("405"))
    }

    func testAllowedSources() {
        for (a, want) in ["192.168.43.20": true, "172.20.10.3": true, "100.64.1.2": true, "fe80::1%en0": true, "127.0.0.1": true,
                          "10.1.2.3": true, "fd00::1": true, "::ffff:192.168.1.2": true, "8.8.8.8": false, "2001:db8::1": false,
                          "203.0.113.9": false] {
            XCTAssertEqual(ProxyCore.allowPrivate(a), want, a)
        }
    }

    #if os(macOS) && canImport(Network)
    /// A one-shot HTTP server on 127.0.0.1 (POSIX), answering "hello".
    private func httpServer() -> UInt16 {
        let fd = Darwin.socket(AF_INET, SOCK_STREAM, 0)
        var on: Int32 = 1
        Darwin.setsockopt(fd, SOL_SOCKET, SO_REUSEADDR, &on, socklen_t(MemoryLayout<Int32>.size))
        var sa = sockaddr_in()
        sa.sin_family = sa_family_t(AF_INET)
        sa.sin_addr.s_addr = inet_addr("127.0.0.1")
        sa.sin_port = 0
        _ = withUnsafePointer(to: &sa) { $0.withMemoryRebound(to: sockaddr.self, capacity: 1) { Darwin.bind(fd, $0, socklen_t(MemoryLayout<sockaddr_in>.size)) } }
        Darwin.listen(fd, 8)
        var len = socklen_t(MemoryLayout<sockaddr_in>.size)
        _ = withUnsafeMutablePointer(to: &sa) { $0.withMemoryRebound(to: sockaddr.self, capacity: 1) { Darwin.getsockname(fd, $0, &len) } }
        Thread.detachNewThread {
            while true {
                let c = Darwin.accept(fd, nil, nil)
                if c < 0 { return }
                var buf = [UInt8](repeating: 0, count: 4096)
                _ = Darwin.read(c, &buf, buf.count)
                let r = Array("HTTP/1.0 200 OK\r\nContent-Length: 5\r\nConnection: close\r\n\r\nhello".utf8)
                _ = Darwin.write(c, r, r.count)
                Darwin.close(c)
            }
        }
        return UInt16(bigEndian: sa.sin_port)
    }

    private func curl(_ args: [String]) -> String {
        let p = Process()
        p.executableURL = URL(fileURLWithPath: "/usr/bin/curl")
        p.arguments = ["-fsS", "-m", "10"] + args
        let out = Pipe()
        p.standardOutput = out
        p.standardError = Pipe()
        try? p.run()
        p.waitUntilExit()
        return String(decoding: out.fileHandleForReading.readDataToEndOfFile(), as: UTF8.self)
    }

    /// curl, a third-party SOCKS5 and HTTP CONNECT client, through the Swift server.
    func testCurlThroughTheServer() throws {
        let (user, pass) = ProxyCore.credentials(shareKey: Array("home-share-key".utf8))
        let s = ProxyServer(user: user, pass: pass, port: 0, dialer: DirectDialer())
        try s.start()
        defer { s.stop() }
        let deadline = Date().addingTimeInterval(5)
        while !s.current.listening && Date() < deadline { Thread.sleep(forTimeInterval: 0.05) }
        let port = s.current.port
        XCTAssertNotEqual(port, 0)
        let web = httpServer()
        XCTAssertEqual(curl(["--socks5-hostname", "127.0.0.1:\(port)", "-U", "\(user):\(pass)", "http://127.0.0.1:\(web)/"]), "hello")
        XCTAssertEqual(curl(["-p", "-x", "http://\(user):\(pass)@127.0.0.1:\(port)", "http://127.0.0.1:\(web)/"]), "hello")
        XCTAssertEqual(curl(["--socks5-hostname", "127.0.0.1:\(port)", "-U", "\(user):wrong", "http://127.0.0.1:\(web)/"]), "")
        Thread.sleep(forTimeInterval: 0.3)
        let st = s.current
        XCTAssertGreaterThan(st.up, 0)
        XCTAssertGreaterThan(st.down, 0)
        XCTAssertEqual(st.clients.first?.addr, "127.0.0.1")
    }
    #endif
}
