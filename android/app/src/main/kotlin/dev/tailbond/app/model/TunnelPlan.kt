package dev.tailbond.app.model

import com.mqvpn.sdk.core.model.MqvpnConfig
import dev.tailbond.app.pairing.PairingPayload

/** Pairing + settings → what the VPN service needs to bring the tunnel up. */
object TunnelPlan {
    /** Tunnel subnet (contract "Network layout"); routed in split mode so the server and DNS stay reachable. */
    val TUNNEL_SUBNET = Cidr("10.77.0.0", 24)
    const val DEFAULT_DNS = "10.77.0.1"

    fun effectiveMode(p: PairingPayload, s: AppSettings): Mode = s.mode ?: Mode.fromWire(p.mode) ?: Mode.SPEED

    fun config(p: PairingPayload, s: AppSettings): MqvpnConfig {
        val base = MqvpnConfig(
            serverAddress = p.host,
            serverPort = p.port,
            tlsServerName = p.sni?.takeIf { it.isNotBlank() },
            authKey = p.key,
            insecure = p.insecure,
            certSha256 = p.certSha256?.takeIf { it.isNotBlank() },
            dnsServers = p.dns.ifEmpty { listOf(DEFAULT_DNS) },
            // Holding a transport's network is what keeps e.g. cellular up
            // next to Wi-Fi; a disabled one is neither held nor used.
            wifiPathEnabled = s.path(PathKind.WIFI).enabled,
            cellularPathEnabled = s.path(PathKind.CELLULAR).enabled,
            ethernetPathEnabled = s.path(PathKind.ETHERNET).enabled,
            reconnect = true,
        )
        return effectiveMode(p, s).applyTo(base, s.experimentalBackupFec)
    }

    /** Routes for VpnService.Builder.addRoute. */
    fun routes(p: PairingPayload, s: AppSettings, dns: List<String>): List<Cidr> = when (s.routeMode) {
        RouteMode.FULL -> listOf(Cidr("0.0.0.0", 0), Cidr("::", 0))
        RouteMode.TAILNET_LAN -> buildList {
            add(TUNNEL_SUBNET)
            p.tailnetRoutes.mapNotNull { Cidr.parse(it) }.forEach { add(it) }
            // DNS servers outside the routed prefixes still have to go through the tunnel
            dns.mapNotNull { Cidr.parse(it) }.filter { d -> none { it.contains(d) } }.forEach { add(it) }
        }.distinct()
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
