package dev.tailbond.app.model

import dev.tailbond.app.pairing.PairingPayload
import kotlinx.serialization.Serializable
import kotlinx.serialization.builtins.ListSerializer
import kotlinx.serialization.json.Json

/**
 * One server in the phone's ordered server list (docs/architecture.md,
 * "Multiple servers and failover"). [id] is local and stable across
 * re-pairing; the first entry of the list is the primary.
 */
@Serializable
data class ServerProfile(val id: String, val payload: PairingPayload) {
    val name: String get() = ServerList.nameOf(payload)
    val subnet: Cidr get() = ServerList.subnetOf(payload)
}

/** Why a server can't join the list. */
sealed interface AddCheck {
    data object Ok : AddCheck
    /** The same server (address or name) is already in the list as [existing]. */
    data class Duplicate(val existing: String) : AddCheck
    /** Its tunnel subnet [subnet] overlaps [existing]'s ([existingSubnet]). */
    data class SubnetClash(val existing: String, val subnet: Cidr, val existingSubnet: Cidr) : AddCheck
}

/** Pure rules for the server list: names, subnets, clashes, order, migration. */
object ServerList {
    /** Contract default (`server.subnet`). */
    val DEFAULT_SUBNET = Cidr("10.77.0.0", 24)

    private val json = Json { ignoreUnknownKeys = true; encodeDefaults = true }
    private val listSerializer = ListSerializer(ServerProfile.serializer())

    /** `server_name`, else the first label of `sni`, else the host of `server`. */
    fun nameOf(p: PairingPayload): String {
        p.serverName?.trim()?.takeIf { it.isNotEmpty() }?.let { return it }
        p.sni?.trim()?.takeIf { it.isNotEmpty() }?.let { sni ->
            return if (Cidr.isIpLiteral(sni)) sni else sni.substringBefore('.')
        }
        return runCatching { p.host }.getOrDefault(p.server)
    }

    /** `subnet`, else the /24 around `dns[0]`, else 10.77.0.0/24. Always the network address. */
    fun subnetOf(p: PairingPayload): Cidr {
        p.subnet?.let { Cidr.parse(it) }?.takeIf { !it.isV6 }?.let { return network(it) }
        p.dns.firstOrNull()?.let { Cidr.parse(it) }?.takeIf { !it.isV6 }?.let { return network(Cidr(it.address, 24)) }
        return DEFAULT_SUBNET
    }

    /** The server's own tunnel address (x.y.z.1 in its subnet): its DNS forwarder and agent API. */
    fun serverTunnelIp(p: PairingPayload): String =
        p.dns.firstOrNull { Cidr.parse(it)?.isV6 == false } ?: intToV4(v4ToInt(subnetOf(p).address) + 1)

    fun network(c: Cidr): Cidr {
        val mask = mask(c.prefix)
        return Cidr(intToV4(v4ToInt(c.address) and mask), c.prefix)
    }

    /** Two IPv4 prefixes overlap when they agree on the shorter prefix. */
    fun overlaps(a: Cidr, b: Cidr): Boolean {
        if (a.isV6 || b.isV6) return false
        val m = mask(minOf(a.prefix, b.prefix))
        return (v4ToInt(a.address) and m) == (v4ToInt(b.address) and m)
    }

    private fun endpoint(p: PairingPayload): String = runCatching { "${p.host.lowercase()}:${p.port}" }.getOrDefault(p.server.lowercase())

    /**
     * Whether [p] may join [list]. [replacing] is the entry being re-paired
     * (it doesn't clash with itself).
     */
    fun check(list: List<ServerProfile>, p: PairingPayload, replacing: String? = null): AddCheck {
        val others = list.filter { it.id != replacing }
        val name = nameOf(p)
        others.firstOrNull { endpoint(it.payload) == endpoint(p) || it.name.equals(name, ignoreCase = true) }
            ?.let { return AddCheck.Duplicate(it.name) }
        val subnet = subnetOf(p)
        others.firstOrNull { overlaps(it.subnet, subnet) }?.let { return AddCheck.SubnetClash(it.name, subnet, it.subnet) }
        return AddCheck.Ok
    }

    /** The entry [p] re-pairs, if any: same address or same name. */
    fun matching(list: List<ServerProfile>, p: PairingPayload): ServerProfile? =
        list.firstOrNull { endpoint(it.payload) == endpoint(p) } ?: list.firstOrNull { it.name.equals(nameOf(p), ignoreCase = true) }

    /** Moves [id] by [delta] places (negative = up), clamped to the list. */
    fun move(list: List<ServerProfile>, id: String, delta: Int): List<ServerProfile> {
        val i = list.indexOfFirst { it.id == id }
        if (i < 0) return list
        val j = (i + delta).coerceIn(0, list.size - 1)
        if (i == j) return list
        val out = list.toMutableList()
        val item = out.removeAt(i)
        out.add(j, item)
        return out
    }

    fun toJson(list: List<ServerProfile>): String = json.encodeToString(listSerializer, list)

    fun fromJson(s: String): List<ServerProfile>? = try { json.decodeFromString(listSerializer, s) } catch (_: Exception) { null }

    /**
     * The stored list: [listJson] when present, else the single legacy
     * profile ([legacyJson], the one-server app's pairing) as a one-element list.
     */
    fun migrate(listJson: String?, legacyJson: String?, newId: () -> String): List<ServerProfile> {
        listJson?.let { fromJson(it) }?.let { return it }
        val legacy = legacyJson?.let { runCatching { PairingPayload.fromJson(it) }.getOrNull() } ?: return emptyList()
        return listOf(ServerProfile(newId(), legacy))
    }

    /** "homebox", "homebox and vps", "homebox, vps and office". */
    fun joinNames(names: List<String>, last: String = "and"): String = when (names.size) {
        0 -> ""
        1 -> names[0]
        else -> names.dropLast(1).joinToString(", ") + " $last " + names.last()
    }

    private fun mask(prefix: Int): Int = if (prefix == 0) 0 else (-1 shl (32 - prefix))
    internal fun v4ToInt(s: String): Int = s.split('.').fold(0) { acc, o -> (acc shl 8) or o.toInt() }
    internal fun intToV4(v: Int): String = listOf(24, 16, 8, 0).joinToString(".") { ((v ushr it) and 0xff).toString() }
}
