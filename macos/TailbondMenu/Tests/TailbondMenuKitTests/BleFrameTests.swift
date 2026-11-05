import XCTest
@testable import TailbondMenuKit

/// The Mac's half of the phone's BLE control channel (BleFrame): the
/// framing (chunked to the ATT MTU, put back together, a lying length
/// dropped) and the hello proof, pinned against the Go and Kotlin halves
/// (the RFC's own vectors, and the derivation the Go reference pins in
/// agent/internal/proxy/testdata/vectors.json). The Bluetooth transport
/// itself (CoreBluetooth) can't run in a test; CI's macOS runners have no
/// phone to talk to either.
final class BleFrameTests: XCTestCase {
    func testFrameChunksAndReassembly() {
        let payload = Data("{\"id\":1,\"cmd\":\"hello\",\"user\":\"mac/1793836800\",\"proof\":\"0123\"}".utf8)
        let frame = BleFrame.encode(payload)
        // the frame is [2-byte length][payload]
        XCTAssertEqual(Int(frame[frame.startIndex]) << 8 | Int(frame[frame.index(after: frame.startIndex)]), payload.count)
        // a small frame is one chunk, flagged START | END
        let small = BleFrame.chunks(BleFrame.encode(Data("{\"a\":1}".utf8)), mtu: 23)
        XCTAssertEqual(small.count, 1)
        XCTAssertEqual(small[0][small[0].startIndex], 0x03)
        // this one needs two at MTU 23
        XCTAssertEqual(BleFrame.chunks(frame, mtu: 23).count, 2)
        // round trip at every MTU that can carry a frame (6 is the floor:
        // ATT's 3 bytes, the flags byte, the 2-byte length)
        for mtu in [23, 512, 185, 7, 6] {
            let r = BleFrame.Reassembly()
            var got: Data?
            for c in BleFrame.chunks(frame, mtu: mtu) {
                XCTAssertLessThanOrEqual(c.count, mtu - 3, "chunk too big at MTU \(mtu)")
                if let p = r.feed(c) { got = p }
            }
            XCTAssertEqual(got, payload, "round trip at MTU \(mtu)")
        }
    }

    func testReassemblyDropsWhatDoesntAddUp() {
        let frame = BleFrame.encode(Data("{\"id\":9,\"cmd\":\"status\"}".utf8))
        // a length that lies: nothing delivered, and the next frame works
        let r = BleFrame.Reassembly()
        var lie = Data([0x03, 0xff, 0xff])
        lie.append(frame.dropFirst(2))
        XCTAssertNil(r.feed(lie))
        // a middle chunk with no start before it
        let r2 = BleFrame.Reassembly()
        XCTAssertNil(r2.feed(Data([0x00, 0x41])))
        // a frame too big to be one
        let r3 = BleFrame.Reassembly()
        var big = Data([0x03])
        big.append(UInt8(BleFrame.maxFrame >> 8))
        big.append(UInt8(BleFrame.maxFrame & 0xff))
        XCTAssertNil(r3.feed(big))
        // and after a dropped one, a good frame still arrives
        for c in BleFrame.chunks(frame, mtu: 23) {
            if let p = r.feed(c) {
                XCTAssertEqual(p, frame.dropFirst(2))
            }
        }
    }

    func testProofMatchesTheReferenceDerivation() {
        // the Go reference's vectors.json entry: share key
        // "0123456789abcdef0123456789abcdef", hub "mac", exp 1793836800 →
        // BLE key 7kceoolvxnekqfq2; the proof is HMAC-SHA256 over the
        // challenge with it (hex), as the phone checks (share/BleProto.kt)
        let challenge = Data((1...16).map { UInt8($0) })
        XCTAssertEqual(BleFrame.proof(key: "7kceoolvxnekqfq2", challenge: challenge),
                       "9bd1e94c2bf16c64dcf327a9baa08cf975f2a1a10ed63145b45f9984bbfe48fd")
        // and the primitives against the RFC's own vectors
        XCTAssertEqual(BleCrypto.sha256(Data("abc".utf8)).map { String(format: "%02x", $0) }.joined(),
                       "ba7816bf8f01cfea414140de5dae2223b00361a396177a9cb410ff61f20015ad")
        XCTAssertEqual(BleCrypto.hmacSha256(key: Data(repeating: 0x0b, count: 20), data: Data("Hi There".utf8)).map { String(format: "%02x", $0) }.joined(),
                       "b0344c61d8db38535ca8afceaf0bf12b881dc200c9833da726e9376c2e32cff7")
        XCTAssertEqual(BleCrypto.hmacSha256(key: Data("Jefe".utf8), data: Data("what do ya want for nothing?".utf8)).map { String(format: "%02x", $0) }.joined(),
                       "5bdcc146bf60754e6a042426089575c75a003f089d2739839dec58b964ec3843")
        // a key longer than the block size is hashed first (RFC 4231 case 6)
        XCTAssertEqual(BleCrypto.hmacSha256(key: Data(repeating: 0xaa, count: 131), data: Data("Test Using Larger Than Block-Size Key - Hash Key First".utf8)).map { String(format: "%02x", $0) }.joined(),
                       "60e431591ee0b67f0d8a26aacbf5b77f8e0bc6213728c5140546040f0ee37f54")
    }

    func testEncodeFromJSON() {
        let frame = BleFrame.encode(["cmd": "hello", "id": 2])
        let payload = frame.dropFirst(2)
        let obj = try? JSONSerialization.jsonObject(with: payload) as? [String: Any]
        XCTAssertEqual(obj?["cmd"] as? String, "hello")
        XCTAssertEqual(obj?["id"] as? Int, 2)
    }
}
