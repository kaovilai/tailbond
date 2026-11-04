import Foundation

/// The part of a `tailbond://pair?d=<base64url(JSON)>` pairing that sharing
/// needs (docs/architecture.md, "Pairing payload").
public struct SharePairing: Codable, Equatable {
    public var name: String
    public var server: String
    public var user: String
    public var serverName: String?
    public var shareKey: String?

    enum CodingKeys: String, CodingKey {
        case name, server, user
        case serverName = "server_name"
        case shareKey = "share_key"
    }

    public init(name: String = "", server: String, user: String = "", serverName: String? = nil, shareKey: String? = nil) {
        self.name = name
        self.server = server
        self.user = user
        self.serverName = serverName
        self.shareKey = shareKey
    }

    public init(from d: Decoder) throws {
        let c = try d.container(keyedBy: CodingKeys.self)
        name = try c.decodeIfPresent(String.self, forKey: .name) ?? ""
        server = try c.decode(String.self, forKey: .server)
        user = try c.decodeIfPresent(String.self, forKey: .user) ?? ""
        serverName = try c.decodeIfPresent(String.self, forKey: .serverName)
        shareKey = try c.decodeIfPresent(String.self, forKey: .shareKey)
    }

    /// "homebox": server_name, else the host of server.
    public var label: String {
        if let n = serverName?.trimmingCharacters(in: .whitespaces), !n.isEmpty { return n }
        return host
    }

    /// Host and port of `server` (port 443 when absent, [v6] literals unwrapped).
    public var host: String { Self.split(server).0 }
    public var port: UInt16 { Self.split(server).1 }

    static func split(_ s: String) -> (String, UInt16) {
        if s.hasPrefix("["), let end = s.firstIndex(of: "]") {
            let host = String(s[s.index(after: s.startIndex)..<end])
            let rest = s[s.index(after: end)...]
            return (host, rest.hasPrefix(":") ? UInt16(rest.dropFirst()) ?? 443 : 443)
        }
        let parts = s.split(separator: ":")
        if parts.count == 2, let p = UInt16(parts[1]) { return (String(parts[0]), p) }
        return (s, 443)
    }

    public enum ParseError: Error, Equatable {
        case notALink, badData, noShareKey
    }

    /// Parses a pairing link (or the text around one, as pasted).
    public static func parse(_ text: String) throws -> SharePairing {
        let t = text.trimmingCharacters(in: .whitespacesAndNewlines)
        guard let r = t.range(of: "tailbond://pair?") else { throw ParseError.notALink }
        let link = String(t[r.lowerBound...]).split(whereSeparator: { $0.isWhitespace }).first.map(String.init) ?? ""
        guard let comps = URLComponents(string: link), let d = comps.queryItems?.first(where: { $0.name == "d" })?.value else {
            throw ParseError.notALink
        }
        var b64 = d.replacingOccurrences(of: "-", with: "+").replacingOccurrences(of: "_", with: "/")
        while b64.count % 4 != 0 { b64 += "=" }
        guard let json = Data(base64Encoded: b64), let p = try? JSONDecoder().decode(SharePairing.self, from: json) else {
            throw ParseError.badData
        }
        guard let k = p.shareKey, !k.isEmpty else { throw ParseError.noShareKey }
        return p
    }
}
