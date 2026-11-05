import CryptoKit
import Foundation

/// The Bluetooth LE control channel to a phone running tailbond
/// (docs/phone-hub.md, "One click from the Mac"). The Go reference is
/// agent/internal/bleproto; its testdata/vectors.json pins these bytes
/// (PhoneProtocolTests). The Mac's login secret comes from tailbondd
/// (GET /api/phone-connect/keys): it holds the passes.
public enum PhoneProtocol {
    public static let service = "5f1d0001-7462-4e64-8f6c-7461696c626f"
    public static let rx = "5f1d0002-7462-4e64-8f6c-7461696c626f"
    public static let tx = "5f1d0003-7462-4e64-8f6c-7461696c626f"
    public static let version = 1
    public static let maxMessage = 4096
    public static let toPhone: UInt8 = 1
    public static let toMac: UInt8 = 2

    public enum Failure: Error, Equatable { case tooLong, sealing }

    static func hmac(_ key: String, _ s: String) -> Data {
        Data(HMAC<SHA256>.authenticationCode(for: Data(s.utf8), using: SymmetricKey(data: Data(key.utf8))))
    }

    static func hex(_ d: Data) -> String { d.map { String(format: "%02x", $0) }.joined() }

    static func transcript(_ label: String, _ cn: String, _ sn: String, _ user: String) -> String {
        "\(label)|\(cn)|\(sn)|\(user)"
    }

    public static func authMac(secret: String, cn: String, sn: String, user: String) -> String {
        hex(hmac(secret, transcript("tbp1-ble-auth", cn, sn, user)))
    }

    public static func phoneProof(secret: String, cn: String, sn: String, user: String) -> String {
        hex(hmac(secret, transcript("tbp1-ble-phone", cn, sn, user)))
    }

    public static func sessionKey(secret: String, cn: String, sn: String, user: String) -> Data {
        hmac(secret, transcript("tbp1-ble-session", cn, sn, user))
    }

    static func nonce(_ dir: UInt8, _ ctr: UInt64) -> Data {
        var n = Data([dir, 0, 0, 0])
        for i in (0..<8).reversed() { n.append(UInt8((ctr >> (UInt64(i) * 8)) & 0xff)) }
        return n
    }

    /// AES-256-GCM: ciphertext then the 16-byte tag, as Go's cipher.AEAD.Seal.
    public static func seal(key: Data, dir: UInt8, ctr: UInt64, _ plain: Data) throws -> Data {
        let box = try AES.GCM.seal(plain, using: SymmetricKey(data: key), nonce: AES.GCM.Nonce(data: nonce(dir, ctr)))
        return box.ciphertext + box.tag
    }

    /// nil: it doesn't open (another key, counter or direction, or changed).
    public static func open(key: Data, dir: UInt8, ctr: UInt64, _ sealed: Data) -> Data? {
        guard sealed.count >= 16, let n = try? AES.GCM.Nonce(data: nonce(dir, ctr)),
              let box = try? AES.GCM.SealedBox(nonce: n, ciphertext: sealed.dropLast(16), tag: sealed.suffix(16)) else { return nil }
        return try? AES.GCM.open(box, using: SymmetricKey(data: key))
    }

    public static func frame(_ msg: Data) throws -> Data {
        guard msg.count <= maxMessage else { throw Failure.tooLong }
        return Data([UInt8(msg.count >> 8), UInt8(msg.count & 0xff)]) + msg
    }

    /// ATT payloads for an MTU (MTU - 3 bytes, at least 20).
    public static func chunks(_ framed: Data, mtu: Int) -> [Data] {
        let n = max(mtu - 3, 20)
        let b = [UInt8](framed)
        return stride(from: 0, to: b.count, by: n).map { Data(b[$0..<min($0 + n, b.count)]) }
    }

    /// Received pieces back into messages.
    public struct Reassembler {
        var buf = [UInt8]()
        public init() {}

        public mutating func add(_ piece: Data) throws -> [Data] {
            buf += piece
            var out = [Data]()
            while buf.count >= 2 {
                let n = Int(buf[0]) << 8 | Int(buf[1])
                if n > PhoneProtocol.maxMessage {
                    buf = []
                    throw Failure.tooLong
                }
                if buf.count < 2 + n { break }
                out.append(Data(buf[2..<(2 + n)]))
                buf.removeFirst(2 + n)
            }
            return out
        }
    }
}
