package dev.tailbond.app.model

import kotlinx.serialization.Serializable

/** A website that skips the tunnel (Android 13+): the domain and its www. name. */
@Serializable
data class WebsiteRule(val domain: String, val enabled: Boolean = true, val note: String = "")

/**
 * Per-website bypass (docs/architecture.md, "Bypass rules", Android
 * paragraph): each website is resolved through the tunnel's DNS when the
 * tunnel comes up and every 10 minutes, and its addresses are left out of
 * the VPN with `VpnService.Builder.excludeRoute` (API 33+).
 */
object WebsiteBypass {
    const val MIN_API = 33
    const val REFRESH_MS = 10 * 60_000L
    /** Rebuilding the TUN (a short reconnect) at most this often for new addresses. */
    const val MIN_REBUILD_GAP_MS = 60_000L
    /** Addresses a website used in the last hour stay excluded (CDNs rotate them). */
    const val KEEP_MS = 60 * 60_000L
    const val MAX_PREFIXES = 512
    const val MAX_RULES = 64

    fun supported(sdkInt: Int): Boolean = sdkInt >= MIN_API

    private val LABEL = Regex("^[a-z0-9]([a-z0-9-]{0,61}[a-z0-9])?$")

    /** "https://www.Bank.example/login" → "bank.example"; null if it isn't a website name. */
    fun normalize(input: String): String? {
        var s = input.trim().lowercase()
        s = s.substringAfter("://")
        s = s.substringBefore('/').substringBefore('?').substringBefore('#')
        s = s.substringAfterLast('@')
        if (s.count { it == ':' } == 1) s = s.substringBefore(':')
        s = s.removePrefix("*.").removePrefix("www.").trimEnd('.')
        if (s.length > 253 || '.' !in s || Cidr.isIpLiteral(s)) return null
        val labels = s.split('.')
        if (labels.any { !LABEL.matches(it) } || labels.last().all { it.isDigit() }) return null
        return s
    }

    /**
     * Usual sub-names looked up besides the website itself. A phone can only
     * leave out addresses it looked up itself (no DNS interception inside the
     * VPN), so it can't learn other sub-names; a travel router covers all of them.
     */
    val SUBNAMES = listOf("www", "login", "secure", "online", "auth", "account", "accounts", "my", "m", "app", "api")

    /** Names looked up for a website: the name itself and its usual sub-names. */
    fun namesFor(domain: String): List<String> = listOf(domain) + SUBNAMES.map { "$it.$domain" }

    /**
     * Excluded prefixes for resolved addresses: one /32 or /128 each, minus
     * anything inside [keep] (the tunnel subnet, its DNS, the tailnet ranges:
     * those must stay in the tunnel), sorted, at most [MAX_PREFIXES].
     */
    fun prefixes(addresses: Collection<String>, keep: List<Cidr>): List<Cidr> =
        addresses.asSequence()
            .mapNotNull { Cidr.parse(it) }
            .map { Cidr(it.address, if (it.isV6) 128 else 32) }
            .filter { a -> keep.none { k -> contains(k, a) } }
            .distinct()
            .sortedWith(compareBy({ it.isV6 }, { it.address }))
            .take(MAX_PREFIXES)
            .toList()

    private fun contains(net: Cidr, host: Cidr): Boolean {
        if (net.isV6 != host.isV6) return false
        val a = ipBytes(net.address) ?: return false
        val b = ipBytes(host.address) ?: return false
        var bits = net.prefix
        var i = 0
        while (bits > 0) {
            val m = if (bits >= 8) 0xff else (0xff shl (8 - bits)) and 0xff
            if ((a[i].toInt() and m) != (b[i].toInt() and m)) return false
            bits -= 8; i++
        }
        return true
    }

    /** Literal → bytes without DNS (Cidr.parse already checked the shape). */
    internal fun ipBytes(s: String): ByteArray? {
        if (':' !in s) return s.split('.').map { it.toInt().toByte() }.toByteArray().takeIf { it.size == 4 }
        var body = s
        val tail = s.substringAfterLast(':').takeIf { '.' in it }
        val tailBytes = tail?.split('.')?.map { it.toInt().toByte() }
        if (tail != null) body = s.dropLast(tail.length) + "0:0"
        val halves = body.split("::")
        fun groups(h: String) = if (h.isEmpty()) emptyList() else h.split(':').map { it.toInt(16) }
        val head = groups(halves[0])
        val rest = if (halves.size > 1) groups(halves[1]) else emptyList()
        val all = head + List(8 - head.size - rest.size) { 0 } + rest
        if (all.size != 8) return null
        val out = ByteArray(16)
        all.forEachIndexed { i, g -> out[i * 2] = (g shr 8).toByte(); out[i * 2 + 1] = g.toByte() }
        if (tailBytes != null) for (i in 0..3) out[12 + i] = tailBytes[i]
        return out
    }

    /** Ready-made choices in the "Add a website" dialog. */
    data class Preset(val title: String, val explain: String, val domains: List<String>, val note: String)

    val PRESETS = listOf(
        Preset(
            "Banking",
            "Banks may block a login that seems to come from somewhere else. Type your bank's website in the box above, e.g. mybank.com " +
                "(its login., secure. and other usual names come with it). Use your bank's app? Add the app under Apps that skip tailbond instead.",
            emptyList(), "Bank",
        ),
        Preset(
            "Streaming (where you are)",
            "Netflix in the browser shows the catalogue for where the phone is, not home. For a streaming app, use Apps that skip tailbond instead: it covers everything the app uses.",
            listOf("netflix.com", "nflxvideo.net"), "Streaming",
        ),
        Preset(
            "Wi-Fi calling",
            "Nothing to add: your phone's own Wi-Fi calling already goes straight over Wi-Fi, not through tailbond. Calling apps (WhatsApp, Teams) are apps: add them under Apps that skip tailbond if calls break.",
            emptyList(), "",
        ),
    )
}

/**
 * Which addresses the TUN leaves out, and when it has to be rebuilt.
 * Rebuilding is a short reconnect, so a new address triggers one at most
 * every [WebsiteBypass.MIN_REBUILD_GAP_MS]; addresses that only expire wait
 * for the next rebuild. A change to the website list applies at once.
 */
class WebsiteBypassPlan(private val clock: () -> Long) {
    /** domain → address → last time a lookup returned it. */
    private val seen = HashMap<String, HashMap<String, Long>>()
    var applied: Set<String> = emptySet()
        private set
    private var lastRebuild = Long.MIN_VALUE / 2

    @Synchronized
    fun record(domain: String, addresses: Collection<String>) {
        val now = clock()
        val m = seen.getOrPut(domain) { HashMap() }
        addresses.forEach { m[it] = now }
    }

    /** Forget websites no longer in the list (or switched off). */
    @Synchronized
    fun retain(domains: Set<String>) { seen.keys.retainAll(domains) }

    @Synchronized
    fun wanted(): Set<String> {
        val now = clock()
        return seen.values.flatMap { m -> m.filterValues { now - it <= WebsiteBypass.KEEP_MS }.keys }.toSet()
    }

    /** The address set to rebuild the TUN with now, or null to keep the current one. [force]: the user changed the list. */
    @Synchronized
    fun decide(force: Boolean = false): Set<String>? {
        val want = wanted()
        if (want == applied) return null
        if (force) return want
        if ((want - applied).isNotEmpty() && clock() - lastRebuild >= WebsiteBypass.MIN_REBUILD_GAP_MS) return want
        return null
    }

    @Synchronized
    fun markApplied(set: Set<String>) {
        applied = set
        lastRebuild = clock()
    }
}
