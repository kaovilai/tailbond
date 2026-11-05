import Foundation
import XCTest
@testable import TailbondMenuKit

/// "Connect via phone": the Bluetooth protocol against the Go reference
/// (agent/internal/bleproto/testdata/vectors.json), the menu's words, and
/// the status the macOS e2e left (TB_E2E_STATUS).
final class PhoneTests: XCTestCase {
    func vectors() throws -> [String: Any] {
        var root = URL(fileURLWithPath: #filePath)
        for _ in 0..<5 { root.deleteLastPathComponent() }
        let data = try Data(contentsOf: root.appendingPathComponent("agent/internal/bleproto/testdata/vectors.json"))
        return try XCTUnwrap(try JSONSerialization.jsonObject(with: data) as? [String: Any])
    }

    func unhex(_ s: String) -> Data {
        var d = Data()
        var i = s.startIndex
        while i < s.endIndex {
            let j = s.index(i, offsetBy: 2)
            d.append(UInt8(s[i..<j], radix: 16)!)
            i = j
        }
        return d
    }

    func testLoginAndSealMatchTheGoReference() throws {
        let v = try vectors()
        XCTAssertEqual(v["service"] as? String, PhoneProtocol.service)
        XCTAssertEqual(v["rx"] as? String, PhoneProtocol.rx)
        XCTAssertEqual(v["tx"] as? String, PhoneProtocol.tx)
        let logins = try XCTUnwrap(v["logins"] as? [[String: Any]])
        XCTAssertGreaterThanOrEqual(logins.count, 2)
        for o in logins {
            let s = { (k: String) in o[k] as! String }
            XCTAssertEqual(PhoneProtocol.authMac(secret: s("secret"), cn: s("cn"), sn: s("sn"), user: s("user")), s("auth_mac"))
            XCTAssertEqual(PhoneProtocol.phoneProof(secret: s("secret"), cn: s("cn"), sn: s("sn"), user: s("user")), s("phone_proof"))
            let key = PhoneProtocol.sessionKey(secret: s("secret"), cn: s("cn"), sn: s("sn"), user: s("user"))
            XCTAssertEqual(PhoneProtocol.hex(key), s("session_key"))
            let sealed = try PhoneProtocol.seal(key: key, dir: PhoneProtocol.toPhone, ctr: 0, Data(s("to_phone").utf8))
            XCTAssertEqual(PhoneProtocol.hex(sealed), s("to_phone_sealed"))
            let opened = try XCTUnwrap(PhoneProtocol.open(key: key, dir: PhoneProtocol.toMac, ctr: 1, unhex(s("to_mac_sealed"))))
            XCTAssertEqual(String(decoding: opened, as: UTF8.self), s("to_mac"))
            // not at another counter, nor the other way
            XCTAssertNil(PhoneProtocol.open(key: key, dir: PhoneProtocol.toMac, ctr: 2, unhex(s("to_mac_sealed"))))
            XCTAssertNil(PhoneProtocol.open(key: key, dir: PhoneProtocol.toPhone, ctr: 1, unhex(s("to_mac_sealed"))))
        }
    }

    func testChunksMatchTheGoReference() throws {
        let v = try vectors()
        for o in try XCTUnwrap(v["chunks"] as? [[String: Any]]) {
            let msg = o["message"] as! String
            let got = PhoneProtocol.chunks(try PhoneProtocol.frame(Data(msg.utf8)), mtu: o["mtu"] as! Int)
            XCTAssertEqual(got.map(PhoneProtocol.hex), o["chunks"] as! [String])
            var r = PhoneProtocol.Reassembler()
            var msgs = [Data]()
            for p in got { msgs += try r.add(p) }
            XCTAssertEqual(msgs.map { String(decoding: $0, as: UTF8.self) }, [msg])
        }
        var r = PhoneProtocol.Reassembler()
        XCTAssertThrowsError(try r.add(Data([0xff, 0xff])))
        XCTAssertThrowsError(try PhoneProtocol.frame(Data(count: PhoneProtocol.maxMessage + 1)))
    }

    func testErrorCodes() {
        XCTAssertEqual(PhoneLinkError.off.code, "off")
        XCTAssertEqual(PhoneLinkError.denied.code, "denied")
        XCTAssertEqual(PhoneLinkError.noPhone("x").code, "no_phone")
        XCTAssertEqual(PhoneLinkError.auth("x").code, "auth")
        XCTAssertEqual(PhoneLinkError.hotspot("x").code, "hotspot")
        XCTAssertEqual(PhoneLinkError.refused(code: "hub", message: "m").code, "hub")
        XCTAssertEqual(PhoneLinkError.refused(code: "shizuku", message: "m").text, "m")
    }

    func testMenuWords() throws {
        let b = try XCTUnwrap(MenuSummary(try fixture("phone-bonded")).phone)
        XCTAssertEqual(b.title, "Bonded via phone · Mobile data + Wi-Fi")
        XCTAssertEqual(b.detail, "Pixel 9 · 81% battery · 2 paths bonded")
        XCTAssertEqual(b.tone, .good)
        XCTAssertEqual(b.action, "Disconnect from phone")
        XCTAssertTrue(MenuSummary(try fixture("phone-bonded")).text.contains("Phone: Bonded via phone · Mobile data + Wi-Fi"))

        let f = try XCTUnwrap(MenuSummary(try fixture("phone-failed")).phone)
        XCTAssertEqual(f.title, "Couldn't connect via phone")
        XCTAssertTrue(f.detail.hasPrefix("Turning on the phone's hotspot: "), f.detail)
        XCTAssertTrue(f.detail.contains("Shizuku isn't running"), f.detail)
        XCTAssertEqual(f.tone, .bad)
        XCTAssertEqual(f.action, "Try again")

        let fb = try XCTUnwrap(MenuSummary(try fixture("phone-fallback")).phone)
        XCTAssertEqual(fb.title, "Bonded via phone")
        XCTAssertTrue(fb.detail.contains("No Bluetooth"), fb.detail)

        let i = try XCTUnwrap(MenuSummary(try fixture("phone-idle")).phone)
        XCTAssertEqual(i.action, "Connect via phone")
        XCTAssertFalse(i.connected)
        // an agent without phone hubs: no section
        XCTAssertNil(MenuSummary(try fixture("connected")).phone)
        XCTAssertEqual(MenuSummary.stepWords("internet"), "Checking the internet through the phone")
    }

    /// The real tailbondd's status, saved by scripts/e2e-macos.sh after the
    /// one click went through the stand-in phone hub: the menu says bonded.
    func testE2EStatus() throws {
        guard let p = ProcessInfo.processInfo.environment["TB_E2E_STATUS"], !p.isEmpty else {
            throw XCTSkip("set TB_E2E_STATUS to the status scripts/e2e-macos.sh saved")
        }
        let st = try JSONDecoder().decode(AgentStatus.self, from: Data(contentsOf: URL(fileURLWithPath: p)))
        let ph = try XCTUnwrap(MenuSummary(st).phone)
        XCTAssertEqual(ph.title, "Bonded via phone · Mobile data + Wi-Fi")
        XCTAssertEqual(ph.action, "Disconnect from phone")
        XCTAssertTrue(ph.connected)
        print("menu bar, from the e2e's status:\n\(MenuSummary(st).text)")
    }

    func testClientCalls() async throws {
        let cfg = URLSessionConfiguration.ephemeral
        cfg.protocolClasses = [StubAgent.self]
        StubAgent.seen = []
        defer { StubAgent.seen = [] } // SummaryTests' own list starts empty
        let c = AgentClient(base: URL(string: "http://127.0.0.1:8077")!, session: URLSession(configuration: cfg))
        let keys = try await c.phoneKeys()
        XCTAssertEqual(keys, [BLELogin(kid: "1a2b3c4d", user: "tiger-mbp/1793836800", secret: "7kceoolvxnekqfq2")])
        try await c.phoneConnect(["ble": "denied"])
        try await c.phoneDisconnect()
        XCTAssertEqual(StubAgent.seen, [
            "GET /api/phone-connect/keys", "POST /api/phone-connect {\"connect\":{\"ble\":\"denied\"}}", "POST /api/phone-connect {\"disconnect\":true}",
        ])
    }
}
