import Foundation
import XCTest
@testable import TailbondShareKit

final class ProtocolTests: XCTestCase {
    func hex(_ s: String) -> [UInt8] {
        var out: [UInt8] = []
        var i = s.startIndex
        while i < s.endIndex {
            let j = s.index(i, offsetBy: 2)
            out.append(UInt8(s[i..<j], radix: 16)!)
            i = j
        }
        return out
    }

    /// agent/internal/share/testdata/vectors.json, the Go reference's pinned bytes.
    func vectors() throws -> [String: Any] {
        var root = URL(fileURLWithPath: #filePath)
        for _ in 0..<5 { root.deleteLastPathComponent() }
        let url = root.appendingPathComponent("agent/internal/share/testdata/vectors.json")
        let data = try Data(contentsOf: url)
        return try JSONSerialization.jsonObject(with: data) as! [String: Any]
    }

    func testWireFormatAndMACsMatchTheGoReference() throws {
        let v = try vectors()
        func s(_ k: String) -> String { v[k] as! String }
        let P = ShareProtocol.self
        let key = Array(s("key").utf8)
        let exp = (v["exp"] as! NSNumber).uint64Value
        let pass = P.passKey(sharerKey: key, hubID: s("hub_id"), exp: exp)
        let hn = hex(s("hub_nonce")), sn = hex(s("sharer_nonce")), sess = hex(s("session"))
        let cap = (v["cap_left"] as! NSNumber).uint64Value
        XCTAssertEqual(P.keyID(key), s("key_id"))
        XCTAssertEqual(pass, hex(s("pass_key")))
        XCTAssertEqual(P.authMAC(key: pass, hubNonce: hn, sharerNonce: sn, hubID: s("hub_id")), hex(s("auth_mac")))
        XCTAssertEqual(P.welcomeMAC(key: pass, hubNonce: hn, sharerNonce: sn, session: sess, capLeft: cap), hex(s("welcome_mac")))
        XCTAssertEqual(P.refuseMAC(key: pass, hubNonce: hn, sharerNonce: sn, reason: .battery), hex(s("refuse_mac")))
        XCTAssertEqual(P.helloFrame(hubNonce: hn, exp: exp, hubID: s("hub_id")), hex(s("hello")))
        XCTAssertEqual(P.challengeFrame(hubNonce: hn, sharerNonce: sn, sharerID: s("sharer_id"), label: s("label")), hex(s("challenge")))
        XCTAssertEqual(P.authFrame(hubNonce: hn, sharerNonce: sn, mac: hex(s("auth_mac"))), hex(s("auth")))
        XCTAssertEqual(P.welcomeFrame(hubNonce: hn, session: sess, capLeft: cap, mac: hex(s("welcome_mac"))), hex(s("welcome")))
        XCTAssertEqual(P.refuseFrame(hubNonce: hn, reason: .battery, mac: hex(s("refuse_mac"))), hex(s("refuse")))
        XCTAssertEqual(P.dataFrame(session: sess, payload: Array("quic".utf8)), hex(s("data")))
        XCTAssertEqual(P.keepaliveFrame(session: sess, seq: 7), hex(s("keepalive")))
        XCTAssertEqual(P.kaAckFrame(session: sess, seq: 7, up: 1000, down: 2000, capLeft: P.noCap, battery: 63, flags: P.flagCharging), hex(s("ka_ack")))
        XCTAssertEqual(P.byeFrame(session: sess, reason: .cap), hex(s("bye")))
        XCTAssertEqual(P.parseHello(hex(s("hello"))), ShareProtocol.Hello(hubNonce: hn, exp: exp, hubID: "slate7"))
        XCTAssertNil(P.parseHello(Array(hex(s("hello")).prefix(5))))
    }
}

final class SharerCoreTests: XCTestCase {
    let key = Array("0123456789abcdef0123456789abcdef".utf8)
    let P = ShareProtocol.self

    func core(allowed: Set<String> = [], cap: UInt64 = 0, check: @escaping () -> SharerCore.Check = { .init(battery: 80, charging: true) }) -> SharerCore {
        SharerCore(config: .init(id: "iphone-1", label: "Sam's iPhone", key: key, capBytes: cap, allowedHubs: allowed), check: check)
    }

    func frames(_ a: [SharerCore.Action], to hub: String) -> [[UInt8]] {
        a.compactMap { if case let .toHub(h, b) = $0, h == hub { return b } else { return nil } }
    }

    static let hour = UInt64(Date().timeIntervalSince1970) + 3600

    /// Hub side of the handshake; the session id, or the refusal reason. The
    /// hub's pass is what the server derives from `key` (the sharer's), or `pass`.
    func handshake(_ c: SharerCore, hub: String = "192.168.8.1:5000", id: String = "slate7", key k: [UInt8]? = nil,
                   exp: UInt64 = hour, pass: [UInt8]? = nil) -> ([UInt8]?, UInt8?) {
        let hn = [UInt8](repeating: 7, count: 16)
        let pk = pass ?? P.passKey(sharerKey: k ?? key, hubID: id, exp: exp)
        let ch = frames(c.handle(P.helloFrame(hubNonce: hn, exp: exp, hubID: id), from: hub), to: hub)
        XCTAssertEqual(ch.first?.first, P.challenge)
        let sn = Array(ch[0][(2 + 16)..<(2 + 32)])
        let out = c.handle(P.authFrame(hubNonce: hn, sharerNonce: sn, mac: P.authMAC(key: pk, hubNonce: hn, sharerNonce: sn, hubID: id)), from: hub)
        guard let reply = frames(out, to: hub).last else { return (nil, nil) }
        if reply[0] == P.refuse { return (nil, reply[2 + 16]) }
        XCTAssertEqual(reply[0], P.welcome)
        let sess = Array(reply[(2 + 16)..<(2 + 16 + 8)])
        let capLeft = P.read64(reply, 2 + 16 + 8)
        let mac = Array(reply[(2 + 16 + 16)...])
        XCTAssertTrue(P.macEqual(mac, P.welcomeMAC(key: pk, hubNonce: hn, sharerNonce: sn, session: sess, capLeft: capLeft)), "the sharer proves the key")
        XCTAssertTrue(out.contains(.openUpstream(session: sess)))
        return (sess, nil)
    }

    func testRelaysBothWaysAndCounts() {
        let c = core()
        let (s, _) = handshake(c)
        let sess = s!
        let up = c.handle(P.dataFrame(session: sess, payload: Array("quic".utf8)), from: "192.168.8.1:5000")
        XCTAssertEqual(up, [.toUpstream(session: sess, Array("quic".utf8))])
        let down = c.fromUpstream(session: sess, Array("re:quic".utf8))
        XCTAssertEqual(down, [.toHub("192.168.8.1:5000", P.dataFrame(session: sess, payload: Array("re:quic".utf8)))])
        XCTAssertEqual(c.hubs.first?.upBytes, 4)
        XCTAssertEqual(c.hubs.first?.downBytes, 7)
        XCTAssertEqual(c.used, 11)
        // the same session from another address is ignored
        XCTAssertEqual(c.handle(P.dataFrame(session: sess, payload: [1]), from: "192.168.8.66:5000"), [])
        // keepalive: battery and charging
        let ack = frames(c.handle(P.keepaliveFrame(session: sess, seq: 1), from: "192.168.8.1:5000"), to: "192.168.8.1:5000")[0]
        XCTAssertEqual(ack[0], P.kaAck)
        XCTAssertEqual(ack[ack.count - 2], 80)
        XCTAssertEqual(ack[ack.count - 1], P.flagCharging)
    }

    func testWrongKeyGetsNothing() {
        let c = core()
        let (s, r) = handshake(c, key: Array("another server's key............".utf8))
        XCTAssertNil(s)
        XCTAssertNil(r)
        XCTAssertTrue(c.hubs.isEmpty)
    }

    func testAllowedHubs() {
        let c = core(allowed: ["slate7"])
        XCTAssertEqual(handshake(c, hub: "a:1", id: "laptop").1, ShareProtocol.Reason.notAllowed.rawValue)
        XCTAssertNotNil(handshake(c, hub: "b:1", id: "slate7").0)
    }

    func testLowBatteryEndsAndRefuses() {
        var low = false
        let c = core(check: { low ? .init(reason: .battery, battery: 12) : .init(battery: 50) })
        let sess = handshake(c).0!
        low = true
        let bye = frames(c.handle(P.keepaliveFrame(session: sess, seq: 1), from: "192.168.8.1:5000"), to: "192.168.8.1:5000")[0]
        XCTAssertEqual(bye, P.byeFrame(session: sess, reason: .battery))
        XCTAssertEqual(handshake(c).1, ShareProtocol.Reason.battery.rawValue)
    }

    func testCapStopsSharing() {
        let c = core(cap: 10)
        let sess = handshake(c).0!
        _ = c.handle(P.dataFrame(session: sess, payload: [UInt8](repeating: 1, count: 6)), from: "192.168.8.1:5000")
        _ = c.fromUpstream(session: sess, [UInt8](repeating: 2, count: 6))
        let out = c.handle(P.dataFrame(session: sess, payload: [1]), from: "192.168.8.1:5000")
        XCTAssertTrue(out.contains(.toHub("192.168.8.1:5000", P.byeFrame(session: sess, reason: .cap))))
        XCTAssertEqual(handshake(c).1, ShareProtocol.Reason.cap.rawValue)
    }

    func testAnotherHubsNameGetsNothing() {
        let c = core(allowed: ["slate7"])
        let laptopPass = P.passKey(sharerKey: key, hubID: "laptop", exp: Self.hour)
        let forged = handshake(c, id: "slate7", pass: laptopPass)
        XCTAssertNil(forged.0)
        XCTAssertNil(forged.1, "not even a refusal")
        let stretched = handshake(c, hub: "a:1", id: "laptop", exp: Self.hour + 3600, pass: laptopPass)
        XCTAssertNil(stretched.0)
        XCTAssertNil(stretched.1)
        XCTAssertTrue(c.hubs.isEmpty)
    }

    func testExpiredPassIsRefusedAndEndsTheSession() {
        let c = core()
        let past = UInt64(Date().timeIntervalSince1970) - 60
        XCTAssertEqual(handshake(c, exp: past).1, ShareProtocol.Reason.expired.rawValue)
        let sess = handshake(c).0!
        let out = c.tick(now: Date().addingTimeInterval(7200))
        XCTAssertTrue(out.contains(.toHub("192.168.8.1:5000", P.byeFrame(session: sess, reason: .expired))))
        XCTAssertTrue(c.hubs.isEmpty)
    }

    func testRestrictingHubsDropsTheOthersAtOnce() {
        let c = core()
        let laptop = handshake(c, hub: "a:1", id: "laptop").0!
        XCTAssertNotNil(handshake(c, hub: "b:1", id: "slate7").0)
        let out = c.updateLimits(capBytes: 0, allowedHubs: ["slate7"])
        XCTAssertTrue(out.contains(.toHub("a:1", P.byeFrame(session: laptop, reason: .notAllowed))))
        XCTAssertEqual(c.hubs.map(\.hub), ["slate7"])
        XCTAssertEqual(handshake(c, hub: "a:1", id: "laptop").1, ShareProtocol.Reason.notAllowed.rawValue)
    }

    func testCapChangesAndANewMonthApplyAtOnce() {
        let c = core()
        let sess = handshake(c).0!
        _ = c.fromUpstream(session: sess, [UInt8](repeating: 2, count: 30))
        let out = c.updateLimits(capBytes: 20, allowedHubs: [])
        XCTAssertTrue(out.contains(.toHub("192.168.8.1:5000", P.byeFrame(session: sess, reason: .cap))))
        XCTAssertEqual(handshake(c).1, ShareProtocol.Reason.cap.rawValue)
        c.resetUsage()
        XCTAssertNotNil(handshake(c).0)
        _ = c.updateLimits(capBytes: 0, allowedHubs: [])
        XCTAssertTrue(c.knows("192.168.8.1:5000"))
        XCTAssertFalse(c.knows("10.0.0.9:1"))
    }

    func testIdleHubIsDropped() {
        let c = core()
        let sess = handshake(c).0!
        let out = c.tick(now: Date().addingTimeInterval(31))
        XCTAssertTrue(out.contains(.closeUpstream(session: sess)))
        XCTAssertTrue(c.hubs.isEmpty)
    }
}

final class PairingTests: XCTestCase {
    func link(_ json: String) -> String {
        let b = Data(json.utf8).base64EncodedString().replacingOccurrences(of: "+", with: "-")
            .replacingOccurrences(of: "/", with: "_").replacingOccurrences(of: "=", with: "")
        return "tailbond://pair?d=" + b
    }

    func testParsesTheShareParts() throws {
        let p = try SharePairing.parse("Open this: " + link(#"{"v":1,"name":"sam-iphone","server":"203.0.113.7:8443","user":"sam-iphone","key":"k","server_name":"homebox","share_key":"sk"}"#) + " thanks")
        XCTAssertEqual(p.label, "homebox")
        XCTAssertEqual(p.host, "203.0.113.7")
        XCTAssertEqual(p.port, 8443)
        XCTAssertEqual(p.shareKey, "sk")
        XCTAssertEqual(SharePairing.split("[2001:db8::1]:443").0, "2001:db8::1")
    }

    func testRefusesOldPairingsAndJunk() {
        XCTAssertThrowsError(try SharePairing.parse(link(#"{"v":1,"server":"h:443","key":"k"}"#))) {
            XCTAssertEqual($0 as? SharePairing.ParseError, .noShareKey)
        }
        XCTAssertThrowsError(try SharePairing.parse("hello")) { XCTAssertEqual($0 as? SharePairing.ParseError, .notALink) }
    }
}
