import CryptoKit
import Foundation

/// The phone hub's proxy protocol (docs/phone-hub.md), as a pure state
/// machine: SOCKS5 (RFC 1928, password RFC 1929, CONNECT only) and HTTP
/// CONNECT on one port. A port of the Go reference, agent/internal/proxy;
/// `ProxyServer` runs it over Network.framework.
public enum ProxyCore {
    public static let serviceType = "_tailbond-proxy._tcp"
    public static let defaultPort: UInt16 = 1888
    public static let user = "tailbond"

    /// The password from the home server's share key: the first 16
    /// characters of base32(HMAC-SHA256(key, "tbp1-proxy")), lower case.
    /// Pinned by agent/internal/proxy/testdata/vectors.json.
    public static func credentials(shareKey: [UInt8]) -> (user: String, pass: String) {
        let mac = HMAC<SHA256>.authenticationCode(for: Array("tbp1-proxy".utf8), using: SymmetricKey(data: shareKey))
        return (user, String(base32(Array(mac)).prefix(16)).lowercased())
    }

    /// RFC 4648 base32, no padding.
    public static func base32(_ b: [UInt8]) -> String {
        let alpha = Array("ABCDEFGHIJKLMNOPQRSTUVWXYZ234567")
        var out = ""
        var buf = 0, bits = 0
        for x in b {
            buf = (buf << 8) | Int(x)
            bits += 8
            while bits >= 5 {
                out.append(alpha[(buf >> (bits - 5)) & 31])
                bits -= 5
            }
            buf &= (1 << bits) - 1
        }
        if bits > 0 { out.append(alpha[(buf << (5 - bits)) & 31]) }
        return out
    }

    /// What the handshake wants next.
    public enum Step: Equatable {
        /// Read more bytes.
        case more
        /// Send these bytes and read more.
        case reply([UInt8])
        /// Send these bytes and close (refused).
        case refuse([UInt8])
        /// Open host:port upstream, then send `ok` (or `fail`) and relay;
        /// `rest` are bytes the client already sent past the handshake.
        case connect(host: String, port: UInt16, ok: [UInt8], fail: [UInt8], rest: [UInt8])
    }

    /// One client's handshake. Feed it what arrives; act on each step.
    public struct Handshake {
        let user: String
        let pass: String
        var buf: [UInt8] = []
        var phase = 0 // SOCKS: 0 greeting, 1 password, 2 request

        public init(user: String, pass: String) {
            self.user = user
            self.pass = pass
        }

        public mutating func feed(_ b: [UInt8]) -> Step {
            buf += b
            guard let first = buf.first else { return .more }
            return first == 5 || phase > 0 ? socks() : http()
        }

        private static func eq(_ a: String, _ b: String) -> Bool {
            let x = Array(a.utf8), y = Array(b.utf8)
            guard x.count == y.count else { return false }
            var d: UInt8 = 0
            for i in 0..<x.count { d |= x[i] ^ y[i] }
            return d == 0
        }

        private mutating func socks() -> Step {
            if phase == 0 {
                guard buf.count >= 2, buf.count >= 2 + Int(buf[1]) else { return .more }
                let methods = buf[2..<2 + Int(buf[1])]
                let want: UInt8 = user.isEmpty ? 0 : 2
                buf.removeFirst(2 + Int(buf[1]))
                guard methods.contains(want) else { return .refuse([5, 0xff]) }
                phase = want == 2 ? 1 : 2
                return .reply([5, want])
            }
            if phase == 1 {
                guard buf.count >= 2 else { return .more }
                guard buf[0] == 1 else { return .refuse([1, 1]) }
                let ul = Int(buf[1])
                guard buf.count >= 2 + ul + 1 else { return .more }
                let pl = Int(buf[2 + ul])
                guard buf.count >= 3 + ul + pl else { return .more }
                let u = String(decoding: buf[2..<2 + ul], as: UTF8.self)
                let p = String(decoding: buf[3 + ul..<3 + ul + pl], as: UTF8.self)
                buf.removeFirst(3 + ul + pl)
                guard Self.eq(u, user), Self.eq(p, pass) else { return .refuse([1, 1]) }
                phase = 2
                return .reply([1, 0])
            }
            // request: ver cmd rsv atyp addr port
            guard buf.count >= 5 else { return .more }
            guard buf[0] == 5 else { return .refuse([5, 1, 0, 1, 0, 0, 0, 0, 0, 0]) }
            var host = ""
            var off = 4
            switch buf[3] {
            case 1:
                guard buf.count >= 10 else { return .more }
                host = buf[4..<8].map { String($0) }.joined(separator: ".")
                off = 8
            case 4:
                guard buf.count >= 22 else { return .more }
                var parts: [String] = []
                for i in stride(from: 4, to: 20, by: 2) { parts.append(String(UInt16(buf[i]) << 8 | UInt16(buf[i + 1]), radix: 16)) }
                host = parts.joined(separator: ":")
                off = 20
            case 3:
                let n = Int(buf[4])
                guard buf.count >= 5 + n + 2 else { return .more }
                host = String(decoding: buf[5..<5 + n], as: UTF8.self)
                off = 5 + n
            default:
                return .refuse([5, 8, 0, 1, 0, 0, 0, 0, 0, 0])
            }
            let port = UInt16(buf[off]) << 8 | UInt16(buf[off + 1])
            let cmd = buf[1]
            let rest = Array(buf[(off + 2)...])
            buf = []
            guard cmd == 1 else { return .refuse([5, 7, 0, 1, 0, 0, 0, 0, 0, 0]) } // CONNECT only
            return .connect(host: host, port: port, ok: [5, 0, 0, 1, 0, 0, 0, 0, 0, 0], fail: [5, 5, 0, 1, 0, 0, 0, 0, 0, 0], rest: rest)
        }

        private mutating func http() -> Step {
            let end: [UInt8] = [13, 10, 13, 10]
            guard let r = find(end) else { return buf.count > 16384 ? .refuse([]) : .more }
            let head = String(decoding: buf[0..<r], as: UTF8.self)
            let rest = Array(buf[(r + 4)...])
            buf = []
            let lines = head.components(separatedBy: "\r\n")
            let parts = lines[0].split(separator: " ")
            guard parts.count >= 2, parts[0] == "CONNECT" else {
                return .refuse(Array("HTTP/1.1 405 Method Not Allowed\r\nContent-Length: 0\r\n\r\n".utf8))
            }
            if !user.isEmpty {
                var ok = false
                for l in lines.dropFirst() {
                    let kv = l.split(separator: ":", maxSplits: 1)
                    guard kv.count == 2, kv[0].lowercased() == "proxy-authorization" else { continue }
                    let v = kv[1].trimmingCharacters(in: .whitespaces)
                    if v.hasPrefix("Basic "), let d = Data(base64Encoded: String(v.dropFirst(6))),
                       let s = String(data: d, encoding: .utf8), let c = s.firstIndex(of: ":") {
                        ok = Self.eq(String(s[..<c]), user) && Self.eq(String(s[s.index(after: c)...]), pass)
                    }
                }
                guard ok else {
                    return .refuse(Array("HTTP/1.1 407 Proxy Authentication Required\r\nProxy-Authenticate: Basic realm=\"tailbond\"\r\nContent-Length: 0\r\n\r\n".utf8))
                }
            }
            let target = String(parts[1])
            guard let c = target.lastIndex(of: ":"), let port = UInt16(target[target.index(after: c)...]) else {
                return .refuse(Array("HTTP/1.1 400 Bad Request\r\nContent-Length: 0\r\n\r\n".utf8))
            }
            var host = String(target[..<c])
            if host.hasPrefix("["), host.hasSuffix("]") { host = String(host.dropFirst().dropLast()) }
            return .connect(host: host, port: port, ok: Array("HTTP/1.1 200 Connection established\r\n\r\n".utf8),
                            fail: Array("HTTP/1.1 502 Bad Gateway\r\n\r\n".utf8), rest: rest)
        }

        private func find(_ pat: [UInt8]) -> Int? {
            guard buf.count >= pat.count else { return nil }
            for i in 0...(buf.count - pat.count) where Array(buf[i..<i + pat.count]) == pat { return i }
            return nil
        }
    }

    /// RFC 1918, CGNAT, unique-local, link-local and loopback sources.
    public static func allowPrivate(_ host: String) -> Bool {
        var h = host
        if let z = h.firstIndex(of: "%") { h = String(h[..<z]) }
        let v4 = h.split(separator: ".").compactMap { UInt8($0) }
        if v4.count == 4, !h.contains(":") {
            let (a, b) = (v4[0], v4[1])
            return a == 10 || a == 127 || (a == 172 && (16...31).contains(b)) || (a == 192 && b == 168) ||
                (a == 169 && b == 254) || (a == 100 && (64...127).contains(b))
        }
        let l = h.lowercased()
        if l.hasPrefix("::ffff:") { return allowPrivate(String(l.dropFirst(7))) }
        return l == "::1" || l.hasPrefix("fe8") || l.hasPrefix("fe9") || l.hasPrefix("fea") || l.hasPrefix("feb") ||
            l.hasPrefix("fc") || l.hasPrefix("fd")
    }
}
