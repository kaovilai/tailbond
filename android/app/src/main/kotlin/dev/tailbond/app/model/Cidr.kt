package dev.tailbond.app.model

/** An IPv4/IPv6 prefix as VpnService.Builder.addRoute takes it. */
data class Cidr(val address: String, val prefix: Int) {
    val isV6: Boolean get() = ':' in address
    override fun toString() = "$address/$prefix"

    companion object {
        private val V4 = Regex("^(\\d{1,3})\\.(\\d{1,3})\\.(\\d{1,3})\\.(\\d{1,3})$")
        private val V6_CHARS = Regex("^[0-9a-fA-F:.]+$")

        fun parse(s: String): Cidr? {
            val t = s.trim()
            val addr = t.substringBefore('/')
            val prefix = if ('/' in t) t.substringAfter('/').toIntOrNull() ?: return null
            else if (':' in addr) 128 else 32
            return when {
                isIpv4(addr) -> if (prefix in 0..32) Cidr(addr, prefix) else null
                isIpv6(addr) -> if (prefix in 0..128) Cidr(addr.lowercase(), prefix) else null
                else -> null
            }
        }

        fun isIpLiteral(s: String): Boolean = isIpv4(s.trim()) || isIpv6(s.trim())

        private fun isIpv4(s: String): Boolean {
            val m = V4.matchEntire(s) ?: return false
            return m.groupValues.drop(1).all { it.toInt() in 0..255 }
        }

        // Structural check only (hex groups, one "::", optional dotted tail); no DNS lookups.
        private fun isIpv6(s: String): Boolean {
            if (':' !in s || !V6_CHARS.matches(s)) return false
            if (s.indexOf("::") != s.lastIndexOf("::")) return false
            val tailV4 = s.substringAfterLast(':').takeIf { '.' in it }
            if (tailV4 != null && !isIpv4(tailV4)) return false
            val body = if (tailV4 != null) s.dropLast(tailV4.length) + "0:0" else s
            val groups = body.split(':')
            if (groups.any { it.length > 4 }) return false
            val nonEmpty = groups.count { it.isNotEmpty() }
            return if ("::" in body) nonEmpty <= 7 else groups.size == 8 && nonEmpty == 8
        }
    }
}
