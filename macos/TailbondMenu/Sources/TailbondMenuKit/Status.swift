import Foundation

/// GET /api/status, the parts the menu shows (agent/internal/status,
/// docs/architecture.md). Every field is optional or defaulted: an older or
/// newer agent still decodes.
public struct AgentStatus: Decodable, Equatable {
    public var role: String = "client"
    public var mode: String = ""
    public var state: String = "disconnected"
    public var server: String = ""
    public var tunnelIP: String = ""
    public var uptimeSec: Int = 0
    public var totals: Totals = Totals()
    public var paths: [PathInfo] = []
    public var paired: Bool = false
    public var version: String = ""
    public var error: String = ""
    public var activeServer: String = ""
    public var update: UpdateInfo?
    public var coexist: CoexistInfo?
    public var phoneHub: PhoneHubInfo?
    public var crash: String = ""

    public init() {}

    enum CodingKeys: String, CodingKey {
        case role, mode, state, server, tunnelIP = "tunnel_ip", uptimeSec = "uptime_sec", totals, paths, paired, version, error
        case activeServer = "active_server", update, coexist, phoneHub = "phone_hub", crash
    }

    public init(from d: Decoder) throws {
        let c = try d.container(keyedBy: CodingKeys.self)
        role = try c.decodeIfPresent(String.self, forKey: .role) ?? "client"
        mode = try c.decodeIfPresent(String.self, forKey: .mode) ?? ""
        state = try c.decodeIfPresent(String.self, forKey: .state) ?? "disconnected"
        server = try c.decodeIfPresent(String.self, forKey: .server) ?? ""
        tunnelIP = try c.decodeIfPresent(String.self, forKey: .tunnelIP) ?? ""
        uptimeSec = try c.decodeIfPresent(Int.self, forKey: .uptimeSec) ?? 0
        totals = try c.decodeIfPresent(Totals.self, forKey: .totals) ?? Totals()
        paths = try c.decodeIfPresent([PathInfo].self, forKey: .paths) ?? []
        paired = try c.decodeIfPresent(Bool.self, forKey: .paired) ?? false
        version = try c.decodeIfPresent(String.self, forKey: .version) ?? ""
        error = try c.decodeIfPresent(String.self, forKey: .error) ?? ""
        activeServer = try c.decodeIfPresent(String.self, forKey: .activeServer) ?? ""
        update = try c.decodeIfPresent(UpdateInfo.self, forKey: .update)
        coexist = try c.decodeIfPresent(CoexistInfo.self, forKey: .coexist)
        phoneHub = try c.decodeIfPresent(PhoneHubInfo.self, forKey: .phoneHub)
        crash = try c.decodeIfPresent(String.self, forKey: .crash) ?? ""
    }
}

public struct Totals: Decodable, Equatable {
    public var txBps: Int64 = 0
    public var rxBps: Int64 = 0
    public init() {}
    enum CodingKeys: String, CodingKey { case txBps = "tx_bps", rxBps = "rx_bps" }
    public init(from d: Decoder) throws {
        let c = try d.container(keyedBy: CodingKeys.self)
        txBps = try c.decodeIfPresent(Int64.self, forKey: .txBps) ?? 0
        rxBps = try c.decodeIfPresent(Int64.self, forKey: .rxBps) ?? 0
    }
}

public struct PathInfo: Decodable, Equatable {
    public var iface: String = ""
    public var kind: String = ""
    public var state: String = ""
    public var label: String = ""
    public var srttMs: Int = 0
    public var txBps: Int64 = 0
    public var rxBps: Int64 = 0
    public var enabled: Bool = true
    public var captive: String = ""
    public init() {}
    enum CodingKeys: String, CodingKey {
        case iface, kind, state, label, srttMs = "srtt_ms", txBps = "tx_bps", rxBps = "rx_bps", enabled, captive
    }
    public init(from d: Decoder) throws {
        let c = try d.container(keyedBy: CodingKeys.self)
        iface = try c.decodeIfPresent(String.self, forKey: .iface) ?? ""
        kind = try c.decodeIfPresent(String.self, forKey: .kind) ?? ""
        state = try c.decodeIfPresent(String.self, forKey: .state) ?? ""
        label = try c.decodeIfPresent(String.self, forKey: .label) ?? ""
        srttMs = try c.decodeIfPresent(Int.self, forKey: .srttMs) ?? 0
        txBps = try c.decodeIfPresent(Int64.self, forKey: .txBps) ?? 0
        rxBps = try c.decodeIfPresent(Int64.self, forKey: .rxBps) ?? 0
        enabled = try c.decodeIfPresent(Bool.self, forKey: .enabled) ?? true
        captive = try c.decodeIfPresent(String.self, forKey: .captive) ?? ""
    }
}

public struct UpdateInfo: Decodable, Equatable {
    public var current: String = ""
    public var available: String = ""
    public var policy: String = ""
    public var method: String = ""
    public var state: String = ""
    public var waiting: String = ""
    public var security: Bool = false
    public var error: String = ""
    public init() {}
    enum CodingKeys: String, CodingKey { case current, available, policy, method, state, waiting, security, error }
    public init(from d: Decoder) throws {
        let c = try d.container(keyedBy: CodingKeys.self)
        current = try c.decodeIfPresent(String.self, forKey: .current) ?? ""
        available = try c.decodeIfPresent(String.self, forKey: .available) ?? ""
        policy = try c.decodeIfPresent(String.self, forKey: .policy) ?? ""
        method = try c.decodeIfPresent(String.self, forKey: .method) ?? ""
        state = try c.decodeIfPresent(String.self, forKey: .state) ?? ""
        waiting = try c.decodeIfPresent(String.self, forKey: .waiting) ?? ""
        security = try c.decodeIfPresent(Bool.self, forKey: .security) ?? false
        error = try c.decodeIfPresent(String.self, forKey: .error) ?? ""
    }
}

public struct NetBirdInfo: Decodable, Equatable {
    public var cli = false, running = false, loggedIn = false, connected = false
    public var iface = "", addr = ""
    public init() {}
    enum CodingKeys: String, CodingKey { case cli, running, loggedIn = "logged_in", connected, iface, addr }
    public init(from d: Decoder) throws {
        let c = try d.container(keyedBy: CodingKeys.self)
        cli = try c.decodeIfPresent(Bool.self, forKey: .cli) ?? false
        running = try c.decodeIfPresent(Bool.self, forKey: .running) ?? false
        loggedIn = try c.decodeIfPresent(Bool.self, forKey: .loggedIn) ?? false
        connected = try c.decodeIfPresent(Bool.self, forKey: .connected) ?? false
        iface = try c.decodeIfPresent(String.self, forKey: .iface) ?? ""
        addr = try c.decodeIfPresent(String.self, forKey: .addr) ?? ""
    }
}

public struct ForeignVPN: Decodable, Equatable {
    public var name = "", kind = "", iface = ""
    /// the networks it routes (NetBird's own range, the company subnets it pushes)
    public var prefixes: [String] = []
    public init(name: String, kind: String, iface: String, prefixes: [String] = []) {
        self.name = name; self.kind = kind; self.iface = iface; self.prefixes = prefixes
    }
    enum CodingKeys: String, CodingKey { case name, kind, iface, prefixes }
    public init(from d: Decoder) throws {
        let c = try d.container(keyedBy: CodingKeys.self)
        name = try c.decodeIfPresent(String.self, forKey: .name) ?? ""
        kind = try c.decodeIfPresent(String.self, forKey: .kind) ?? ""
        iface = try c.decodeIfPresent(String.self, forKey: .iface) ?? ""
        prefixes = try c.decodeIfPresent([String].self, forKey: .prefixes) ?? []
    }
}

public struct CoexistInfo: Decodable, Equatable {
    public var netbird: NetBirdInfo?
    public var foreign: [ForeignVPN] = []
    public var splitDNS = false
    public var dnsDomains: [String] = []
    public var tailnetViaTailscaleApp = false
    public var workVPN = "", workVPNEffective = ""
    public var warnings: [String] = []
    public var conflicts: [String] = []
    public var error = ""
    public init() {}
    enum CodingKeys: String, CodingKey {
        case netbird, foreign, splitDNS = "split_dns", dnsDomains = "dns_domains", tailnetViaTailscaleApp = "tailnet_via_tailscale_app"
        case workVPN = "work_vpn", workVPNEffective = "work_vpn_effective", warnings, conflicts, error
    }
    public init(from d: Decoder) throws {
        let c = try d.container(keyedBy: CodingKeys.self)
        netbird = try? c.decodeIfPresent(NetBirdInfo.self, forKey: .netbird)
        foreign = try c.decodeIfPresent([ForeignVPN].self, forKey: .foreign) ?? []
        splitDNS = try c.decodeIfPresent(Bool.self, forKey: .splitDNS) ?? false
        dnsDomains = try c.decodeIfPresent([String].self, forKey: .dnsDomains) ?? []
        tailnetViaTailscaleApp = try c.decodeIfPresent(Bool.self, forKey: .tailnetViaTailscaleApp) ?? false
        workVPN = try c.decodeIfPresent(String.self, forKey: .workVPN) ?? ""
        workVPNEffective = try c.decodeIfPresent(String.self, forKey: .workVPNEffective) ?? ""
        warnings = try c.decodeIfPresent([String].self, forKey: .warnings) ?? []
        conflicts = try c.decodeIfPresent([String].self, forKey: .conflicts) ?? []
        error = try c.decodeIfPresent(String.self, forKey: .error) ?? ""
    }
}

public struct PhoneHubInfo: Decodable, Equatable {
    public struct Use: Decodable, Equatable {
        public var label = ""
        public var bonded = false
        enum CodingKeys: String, CodingKey { case label, bonded }
        public init(from d: Decoder) throws {
            let c = try d.container(keyedBy: CodingKeys.self)
            label = try c.decodeIfPresent(String.self, forKey: .label) ?? ""
            bonded = try c.decodeIfPresent(Bool.self, forKey: .bonded) ?? false
        }
    }
    public var using: Use?
    enum CodingKeys: String, CodingKey { case using }
    public init(from d: Decoder) throws {
        let c = try d.container(keyedBy: CodingKeys.self)
        using = try c.decodeIfPresent(Use.self, forKey: .using)
    }
}
