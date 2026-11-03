import CryptoKit
import Foundation

/// Pair & Share wire format, the sharer's half. The Go reference is
/// agent/internal/share/proto.go; agent/internal/share/testdata/vectors.json
/// pins Go, Android and this file to the same bytes.
///
///     HELLO      01 ver hub_nonce[16] idlen id
///     CHALLENGE  02 ver hub_nonce[16] sharer_nonce[16] idlen id lablen label
///     AUTH       03 ver hub_nonce[16] sharer_nonce[16] mac[32]
///     WELCOME    04 ver hub_nonce[16] session[8] cap_left[8] mac[32]
///     REFUSE     05 ver hub_nonce[16] reason mac[32]
///     DATA       10 session[8] payload
///     KEEPALIVE  11 session[8] seq[4]
///     KA_ACK     12 session[8] seq[4] up[8] down[8] cap_left[8] battery flags
///     BYE        13 session[8] reason
public enum ShareProtocol {
    public static let version: UInt8 = 1
    public static let defaultPort: UInt16 = 47077
    public static let serviceType = "_tailbond-share._udp"

    public static let hello: UInt8 = 0x01
    public static let challenge: UInt8 = 0x02
    public static let auth: UInt8 = 0x03
    public static let welcome: UInt8 = 0x04
    public static let refuse: UInt8 = 0x05
    public static let data: UInt8 = 0x10
    public static let keepalive: UInt8 = 0x11
    public static let kaAck: UInt8 = 0x12
    public static let bye: UInt8 = 0x13

    public static let nonceLen = 16
    public static let sessionLen = 8
    public static let macLen = 32
    public static let dataHeader = 1 + sessionLen
    public static let maxID = 64
    public static let noCap = UInt64.max

    public enum Reason: UInt8 {
        case none = 0, notAllowed = 1, cap = 2, battery = 3, paused = 4, noCellular = 5, replaced = 6, idle = 7
    }

    public static let flagCharging: UInt8 = 1

    static func idBytes(_ s: String) -> [UInt8] { Array(Array(s.utf8).prefix(maxID)) }

    static func be64(_ v: UInt64) -> [UInt8] { (0..<8).map { UInt8(truncatingIfNeeded: v >> (56 - 8 * UInt64($0))) } }
    static func be32(_ v: UInt32) -> [UInt8] { (0..<4).map { UInt8(truncatingIfNeeded: v >> (24 - 8 * UInt32($0))) } }
    static func read32(_ b: [UInt8], _ at: Int) -> UInt32 { b[at..<at + 4].reduce(0) { $0 << 8 | UInt32($1) } }
    static func read64(_ b: [UInt8], _ at: Int) -> UInt64 { b[at..<at + 8].reduce(0) { $0 << 8 | UInt64($1) } }

    public struct Hello: Equatable { public var hubNonce: [UInt8]; public var hubID: String }
    public struct Auth: Equatable { public var hubNonce: [UInt8]; public var sharerNonce: [UInt8]; public var mac: [UInt8] }

    public static func parseHello(_ b: [UInt8]) -> Hello? {
        guard b.count >= 3 + nonceLen, b[0] == hello, b[1] == version else { return nil }
        let n = Int(b[2 + nonceLen])
        guard b.count >= 3 + nonceLen + n else { return nil }
        return Hello(hubNonce: Array(b[2..<2 + nonceLen]),
                     hubID: String(decoding: b[3 + nonceLen..<3 + nonceLen + n], as: UTF8.self))
    }

    public static func parseAuth(_ b: [UInt8]) -> Auth? {
        guard b.count >= 2 + 2 * nonceLen + macLen, b[0] == auth, b[1] == version else { return nil }
        return Auth(hubNonce: Array(b[2..<2 + nonceLen]), sharerNonce: Array(b[2 + nonceLen..<2 + 2 * nonceLen]),
                    mac: Array(b[2 + 2 * nonceLen..<2 + 2 * nonceLen + macLen]))
    }

    public static func helloFrame(hubNonce: [UInt8], hubID: String) -> [UInt8] {
        let id = idBytes(hubID)
        return [hello, version] + hubNonce + [UInt8(id.count)] + id
    }

    public static func challengeFrame(hubNonce: [UInt8], sharerNonce: [UInt8], sharerID: String, label: String) -> [UInt8] {
        let id = idBytes(sharerID), lab = idBytes(label)
        return [challenge, version] + hubNonce + sharerNonce + [UInt8(id.count)] + id + [UInt8(lab.count)] + lab
    }

    public static func authFrame(hubNonce: [UInt8], sharerNonce: [UInt8], mac: [UInt8]) -> [UInt8] {
        [auth, version] + hubNonce + sharerNonce + mac
    }

    public static func welcomeFrame(hubNonce: [UInt8], session: [UInt8], capLeft: UInt64, mac: [UInt8]) -> [UInt8] {
        [welcome, version] + hubNonce + session + be64(capLeft) + mac
    }

    public static func refuseFrame(hubNonce: [UInt8], reason: Reason, mac: [UInt8]) -> [UInt8] {
        [refuse, version] + hubNonce + [reason.rawValue] + mac
    }

    public static func dataFrame(session: [UInt8], payload: [UInt8]) -> [UInt8] { [data] + session + payload }

    public static func keepaliveFrame(session: [UInt8], seq: UInt32) -> [UInt8] { [keepalive] + session + be32(seq) }

    public static func kaAckFrame(session: [UInt8], seq: UInt32, up: UInt64, down: UInt64, capLeft: UInt64,
                                  battery: UInt8, flags: UInt8) -> [UInt8] {
        [kaAck] + session + be32(seq) + be64(up) + be64(down) + be64(capLeft) + [battery, flags]
    }

    public static func byeFrame(session: [UInt8], reason: Reason) -> [UInt8] { [bye] + session + [reason.rawValue] }

    /// Session id of a DATA/KEEPALIVE/BYE frame.
    public static func session(of b: [UInt8]) -> [UInt8]? { b.count < 1 + sessionLen ? nil : Array(b[1..<1 + sessionLen]) }

    static func mac(_ key: [UInt8], _ label: String, _ parts: [UInt8]...) -> [UInt8] {
        var msg = Array(label.utf8)
        for p in parts { msg += p }
        let code = HMAC<SHA256>.authenticationCode(for: msg, using: SymmetricKey(data: key))
        return Array(code)
    }

    public static func authMAC(key: [UInt8], hubNonce: [UInt8], sharerNonce: [UInt8], hubID: String) -> [UInt8] {
        mac(key, "tbs1-auth", hubNonce, sharerNonce, Array(hubID.utf8))
    }

    public static func welcomeMAC(key: [UInt8], hubNonce: [UInt8], sharerNonce: [UInt8], session: [UInt8], capLeft: UInt64) -> [UInt8] {
        mac(key, "tbs1-welcome", hubNonce, sharerNonce, session, be64(capLeft))
    }

    public static func refuseMAC(key: [UInt8], hubNonce: [UInt8], sharerNonce: [UInt8], reason: Reason) -> [UInt8] {
        mac(key, "tbs1-refuse", hubNonce, sharerNonce, [reason.rawValue])
    }

    /// Public tag of a share key (Bonjour TXT "k="): hubs skip other servers' sharers.
    public static func keyID(_ key: [UInt8]) -> String {
        mac(key, "tbs1-keyid").prefix(4).map { String(format: "%02x", $0) }.joined()
    }

    /// Constant-time compare.
    public static func macEqual(_ a: [UInt8], _ b: [UInt8]) -> Bool {
        guard a.count == b.count else { return false }
        var d: UInt8 = 0
        for i in 0..<a.count { d |= a[i] ^ b[i] }
        return d == 0
    }
}
