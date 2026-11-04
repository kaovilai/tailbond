package dev.tailbond.app.model

import com.mqvpn.sdk.core.model.MqvpnConfig
import dev.tailbond.app.pairing.PairingPayload

/** Pairing + settings → what the VPN service needs to bring the tunnel up. */
object TunnelPlan {
    /** Default tunnel subnet (contract "Network layout"); each server may have its own ([ServerList.subnetOf]). */
    val TUNNEL_SUBNET = ServerList.DEFAULT_SUBNET
    const val DEFAULT_DNS = "10.77.0.1"
    /**
     * The tunnel's MTU, the same as tailbondd's client default (client.mtu
     * 1280, IPv6's minimum): mqvpn's auto value (1382) assumes a 1500-byte
     * uplink, and cellular and hotel Wi-Fi often carry less, so larger
     * packets were lost.
     */
    const val TUN_MTU = 1280
    /**
     * mqvpn applies the cap, then takes its reorder header off (modes that
     * reorder: Speed); the cap includes it, so the TUN still gets [TUN_MTU].
     */
    const val REORDER_HEADER = 8

    /** The pairing's DNS, else the server's own tunnel address (x.y.z.1 of its subnet). */
    fun dns(p: PairingPayload): List<String> = p.dns.ifEmpty { listOf(ServerList.serverTunnelIp(p)) }

    fun effectiveMode(p: PairingPayload, s: AppSettings): Mode = s.mode ?: Mode.fromWire(p.mode) ?: Mode.SPEED

    fun config(p: PairingPayload, s: AppSettings): MqvpnConfig {
        val base = MqvpnConfig(
            serverAddress = p.host,
            serverPort = p.port,
            tlsServerName = p.sni?.takeIf { it.isNotBlank() },
            authKey = p.key,
            insecure = p.insecure,
            certSha256 = p.certSha256?.takeIf { it.isNotBlank() },
            dnsServers = dns(p),
            // Holding a transport's network is what keeps e.g. cellular up
            // next to Wi-Fi; a disabled one is neither held nor used.
            wifiPathEnabled = s.path(PathKind.WIFI).enabled,
            cellularPathEnabled = s.path(PathKind.CELLULAR).enabled,
            ethernetPathEnabled = s.path(PathKind.ETHERNET).enabled,
            reconnect = true,
        )
        val c = effectiveMode(p, s).applyTo(base, s.experimentalBackupFec)
        return c.copy(tunMtu = TUN_MTU + if (c.reorderEnabled) REORDER_HEADER else 0)
    }

    /** Routes for VpnService.Builder.addRoute. */
    fun routes(p: PairingPayload, s: AppSettings, dns: List<String>): List<Cidr> = when (s.routeMode) {
        RouteMode.FULL -> listOf(Cidr("0.0.0.0", 0), Cidr("::", 0))
        RouteMode.TAILNET_LAN -> buildList {
            add(ServerList.subnetOf(p))
            p.tailnetRoutes.mapNotNull { Cidr.parse(it) }.forEach { add(it) }
            // DNS servers outside the routed prefixes still have to go through the tunnel
            dns.mapNotNull { Cidr.parse(it) }.filter { d -> none { it.contains(d) } }.forEach { add(it) }
        }.distinct()
    }

    /**
     * What stays in the tunnel even when a website's address falls inside it:
     * the server's subnet, its DNS servers and the tailnet routes.
     */
    fun keepInTunnel(p: PairingPayload, dns: List<String>): List<Cidr> = buildList {
        add(ServerList.subnetOf(p))
        p.tailnetRoutes.mapNotNull { Cidr.parse(it) }.forEach { add(it) }
        dns.mapNotNull { Cidr.parse(it) }.forEach { add(it) }
    }

    /**
     * Prefixes for VpnService.Builder.excludeRoute: the bypassed websites'
     * addresses, only on Android 13+ and only when internet traffic goes
     * through the tunnel at all (in "Only Tailscale machines" it doesn't).
     */
    fun excludedRoutes(p: PairingPayload, s: AppSettings, dns: List<String>, addresses: Collection<String>, sdkInt: Int): List<Cidr> {
        if (!WebsiteBypass.supported(sdkInt) || s.routeMode != RouteMode.FULL || s.activeWebsites.isEmpty()) return emptyList()
        return WebsiteBypass.prefixes(addresses, keepInTunnel(p, dns))
    }

    private fun Cidr.contains(host: Cidr): Boolean {
        if (isV6 != host.isV6 || isV6) return this == host // v6 containment not needed here
        val a = v4ToInt(address)
        val b = v4ToInt(host.address)
        val mask = if (prefix == 0) 0 else (-1 shl (32 - prefix))
        return (a and mask) == (b and mask)
    }

    private fun v4ToInt(s: String): Int = s.split('.').fold(0) { acc, o -> (acc shl 8) or o.toInt() }
}
