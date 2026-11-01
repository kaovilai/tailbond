package dev.tailbond.app

import dev.tailbond.app.data.DnsLookup
import dev.tailbond.app.model.AppSettings
import dev.tailbond.app.model.Cidr
import dev.tailbond.app.model.RouteMode
import dev.tailbond.app.model.TunnelPlan
import dev.tailbond.app.model.WebsiteBypass
import dev.tailbond.app.model.WebsiteBypassPlan
import dev.tailbond.app.model.WebsiteRule
import dev.tailbond.app.pairing.PairingPayload
import org.junit.Assert.assertEquals
import org.junit.Assert.assertFalse
import org.junit.Assert.assertNull
import org.junit.Assert.assertTrue
import org.junit.Test
import java.io.ByteArrayOutputStream

/** Per-website bypass: names, resolved addresses → excluded prefixes, the API gate, rebuild rate limit, DNS parsing. */
class WebsiteBypassTest {
    private val home = PairingPayload(
        server = "home.example.net:443", key = "k", dns = listOf("10.77.0.1"),
        tailnetRoutes = listOf("100.64.0.0/10", "fd7a:115c:a1e0::/48"),
    )
    private val bank = AppSettings(websites = listOf(WebsiteRule("bank.example", true, "Bank")))

    @Test fun normalize() {
        assertEquals("bank.example", WebsiteBypass.normalize("bank.example"))
        assertEquals("bank.example", WebsiteBypass.normalize("  https://www.Bank.example/login?x=1 "))
        assertEquals("netflix.com", WebsiteBypass.normalize("*.netflix.com"))
        assertEquals("my-bank.co.uk", WebsiteBypass.normalize("my-bank.co.uk:443"))
        assertNull(WebsiteBypass.normalize("localhost"))
        assertNull(WebsiteBypass.normalize("203.0.113.7"))
        assertNull(WebsiteBypass.normalize("bad_name.com"))
        assertNull(WebsiteBypass.normalize(""))
        assertEquals(listOf("bank.example", "www.bank.example"), WebsiteBypass.namesFor("bank.example"))
    }

    @Test fun resolvedAddressesBecomeHostPrefixes() {
        val got = WebsiteBypass.prefixes(
            listOf("203.0.113.10", "2001:db8::10", "203.0.113.10", "198.51.100.4"),
            TunnelPlan.keepInTunnel(home, listOf("10.77.0.1")),
        )
        assertEquals(listOf(Cidr("198.51.100.4", 32), Cidr("203.0.113.10", 32), Cidr("2001:db8::10", 128)), got)
    }

    @Test fun tunnelTailnetAndDnsAddressesStayInTheTunnel() {
        val keep = TunnelPlan.keepInTunnel(home, listOf("10.77.0.1"))
        val got = WebsiteBypass.prefixes(
            listOf("10.77.0.1", "10.77.0.9", "100.101.2.2", "fd7a:115c:a1e0::5", "203.0.113.10", "not-an-ip"),
            keep,
        )
        assertEquals(listOf(Cidr("203.0.113.10", 32)), got)
    }

    @Test fun apiLevelGate() {
        val addrs = listOf("203.0.113.10")
        assertEquals(emptyList<Cidr>(), TunnelPlan.excludedRoutes(home, bank, listOf("10.77.0.1"), addrs, sdkInt = 32))
        assertEquals(listOf(Cidr("203.0.113.10", 32)), TunnelPlan.excludedRoutes(home, bank, listOf("10.77.0.1"), addrs, sdkInt = 33))
        assertFalse(WebsiteBypass.supported(26))
        assertTrue(WebsiteBypass.supported(33))
    }

    @Test fun onlyInFullTunnelAndOnlyEnabledSites() {
        val addrs = listOf("203.0.113.10")
        val split = bank.copy(routeMode = RouteMode.TAILNET_LAN)
        assertEquals(emptyList<Cidr>(), TunnelPlan.excludedRoutes(home, split, listOf("10.77.0.1"), addrs, 35))
        val off = bank.copy(websites = listOf(WebsiteRule("bank.example", enabled = false)))
        assertEquals(emptyList<Cidr>(), TunnelPlan.excludedRoutes(home, off, listOf("10.77.0.1"), addrs, 35))
        assertEquals(setOf("bank.example"), bank.activeWebsites)
    }

    @Test fun rebuildsAreRateLimited() {
        var now = 0L
        val plan = WebsiteBypassPlan { now }
        // first build of the session
        plan.markApplied(plan.wanted())
        plan.record("bank.example", listOf("203.0.113.10"))
        assertEquals(setOf("203.0.113.10"), plan.decide(force = true)) // first lookup after connecting applies at once
        plan.markApplied(setOf("203.0.113.10"))
        // a new address 10 s later waits for the 60 s gap
        now += 10_000
        plan.record("bank.example", listOf("203.0.113.11"))
        assertNull(plan.decide())
        now += 50_000
        assertEquals(setOf("203.0.113.10", "203.0.113.11"), plan.decide())
        plan.markApplied(setOf("203.0.113.10", "203.0.113.11"))
        // the same answers again: nothing to do
        now += 600_000
        plan.record("bank.example", listOf("203.0.113.10", "203.0.113.11"))
        assertNull(plan.decide())
        // an address unseen for an hour only drops at the next rebuild
        now += 600_000
        plan.record("bank.example", listOf("203.0.113.10"))
        now += 3_000_000
        plan.record("bank.example", listOf("203.0.113.10"))
        assertNull(plan.decide())
        // removing a website applies at once
        plan.retain(emptySet())
        assertEquals(emptySet<String>(), plan.decide(force = true))
    }

    private fun reply(id: Int, name: String, answers: List<Pair<Int, ByteArray>>, rcode: Int = 0): ByteArray {
        val q = DnsLookup.query(name, DnsLookup.TYPE_A, id)
        val out = ByteArrayOutputStream()
        out.write(q, 0, 2)
        out.write(byteArrayOf(0x81.toByte(), (0x80 or rcode).toByte(), 0, 1, 0, answers.size.toByte(), 0, 0, 0, 0))
        out.write(q, 12, q.size - 12)
        for ((type, data) in answers) {
            out.write(byteArrayOf(0xc0.toByte(), 12, 0, type.toByte(), 0, 1, 0, 0, 0, 60, 0, data.size.toByte()))
            out.write(data)
        }
        return out.toByteArray()
    }

    @Test fun dnsReplies() {
        val cname = byteArrayOf(3, 'w'.code.toByte(), 'w'.code.toByte(), 'w'.code.toByte(), 0xc0.toByte(), 12)
        val r = reply(0x1234, "bank.example", listOf(5 to cname, 1 to byteArrayOf(203.toByte(), 0, 113, 10), 1 to byteArrayOf(203.toByte(), 0, 113, 11)))
        assertEquals(listOf("203.0.113.10", "203.0.113.11"), DnsLookup.parse(r, 0x1234, DnsLookup.TYPE_A))
        // wrong id, error rcode, truncated: nothing
        assertEquals(emptyList<String>(), DnsLookup.parse(r, 0x9999, DnsLookup.TYPE_A))
        assertEquals(emptyList<String>(), DnsLookup.parse(reply(1, "bank.example", emptyList(), rcode = 3), 1, DnsLookup.TYPE_A))
        assertEquals(emptyList<String>(), DnsLookup.parse(r.copyOf(20), 0x1234, DnsLookup.TYPE_A))
        val v6 = ByteArray(16).also { it[0] = 0x20; it[1] = 0x01; it[2] = 0x0d; it[3] = 0xb8.toByte(); it[15] = 0x10 }
        val r6 = reply(7, "bank.example", listOf(28 to v6))
        assertEquals(listOf("2001:db8:0:0:0:0:0:10"), DnsLookup.parse(r6, 7, DnsLookup.TYPE_AAAA))
        // the parsed v6 form still turns into a /128
        assertEquals(listOf(Cidr("2001:db8:0:0:0:0:0:10", 128)), WebsiteBypass.prefixes(DnsLookup.parse(r6, 7, DnsLookup.TYPE_AAAA), emptyList()))
    }

    @Test fun queryShape() {
        val q = DnsLookup.query("www.bank.example", DnsLookup.TYPE_AAAA, 0xbeef)
        assertEquals(0xbe.toByte(), q[0])
        assertEquals(0xef.toByte(), q[1])
        assertEquals(1, q[5].toInt()) // one question
        assertEquals(3, q[12].toInt()) // "www"
        assertEquals(28, q[q.size - 3].toInt()) // AAAA
    }
}
