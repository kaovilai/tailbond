package dev.tailbond.app

import com.mqvpn.sdk.core.model.MqvpnConfig
import dev.tailbond.app.model.AddCheck
import dev.tailbond.app.model.AppSettings
import dev.tailbond.app.model.Cidr
import dev.tailbond.app.model.Mode
import dev.tailbond.app.model.RouteMode
import dev.tailbond.app.model.ServerList
import dev.tailbond.app.model.ServerProfile
import dev.tailbond.app.model.TunnelPlan
import dev.tailbond.app.pairing.Pairing
import dev.tailbond.app.pairing.PairingPayload
import dev.tailbond.app.pairing.PairingResult
import org.junit.Assert.assertEquals
import org.junit.Assert.assertFalse
import org.junit.Assert.assertNull
import org.junit.Assert.assertTrue
import org.junit.Test

/** The server list: migration, naming, subnets, clashes, order, and payload → config per server. */
class ServerListTest {
    private val home = PairingPayload(
        name = "pixel-8", server = "home.example.net:443", sni = "homebox.tail1234.ts.net", user = "pixel-8",
        key = "k1", dns = listOf("10.77.0.1"), tailnetRoutes = listOf("100.64.0.0/10", "fd7a:115c:a1e0::/48"),
    )
    private val vps = PairingPayload(
        name = "pixel-8", server = "203.0.113.9:8443", sni = "vps.tail1234.ts.net", user = "pixel-8", key = "k2",
        dns = listOf("10.83.142.1"), tailnetRoutes = listOf("100.64.0.0/10"), serverName = "vps", subnet = "10.83.142.0/24",
        mode = "redundant",
    )
    private val list = listOf(ServerProfile("a", home), ServerProfile("b", vps))

    @Test fun naming() {
        assertEquals("vps", ServerList.nameOf(vps))
        assertEquals("homebox", ServerList.nameOf(home)) // first label of sni
        assertEquals("home.example.net", ServerList.nameOf(home.copy(sni = null))) // host of server
        assertEquals("203.0.113.7", ServerList.nameOf(home.copy(sni = null, server = "203.0.113.7:443")))
        assertEquals("office", ServerList.nameOf(home.copy(serverName = "office")))
        assertEquals("homebox", ServerList.nameOf(home.copy(serverName = "  ")))
    }

    @Test fun subnets() {
        assertEquals(Cidr("10.83.142.0", 24), ServerList.subnetOf(vps))
        assertEquals(Cidr("10.77.0.0", 24), ServerList.subnetOf(home)) // /24 around dns[0]
        assertEquals(Cidr("10.90.3.0", 24), ServerList.subnetOf(home.copy(dns = listOf("10.90.3.1"))))
        assertEquals(Cidr("10.77.0.0", 24), ServerList.subnetOf(home.copy(dns = emptyList()))) // default
        assertEquals(Cidr("10.80.0.0", 16), ServerList.subnetOf(home.copy(subnet = "10.80.4.0/16"))) // normalised
        assertEquals("10.83.142.1", ServerList.serverTunnelIp(vps))
        assertEquals("10.78.5.1", ServerList.serverTunnelIp(home.copy(dns = emptyList(), subnet = "10.78.5.0/24")))
    }

    @Test fun freeSubnetForAClash() {
        assertEquals(Cidr("10.78.0.0", 24), ServerList.freeSubnet(listOf(Cidr("10.77.0.0", 24))))
        assertEquals(Cidr("10.78.1.0", 24), ServerList.freeSubnet(listOf(Cidr("10.77.0.0", 24), Cidr("10.78.0.0", 24))))
        assertEquals(Cidr("10.79.0.0", 24), ServerList.freeSubnet(listOf(Cidr("10.78.0.0", 16))))
    }

    @Test fun overlap() {
        assertTrue(ServerList.overlaps(Cidr("10.77.0.0", 24), Cidr("10.77.0.0", 24)))
        assertTrue(ServerList.overlaps(Cidr("10.77.0.0", 16), Cidr("10.77.9.0", 24)))
        assertFalse(ServerList.overlaps(Cidr("10.77.0.0", 24), Cidr("10.78.0.0", 24)))
    }

    @Test fun refusesASubnetClash() {
        val clash = vps.copy(subnet = null, dns = listOf("10.77.0.1"))
        val c = ServerList.check(listOf(ServerProfile("a", home)), clash)
        assertTrue(c is AddCheck.SubnetClash)
        c as AddCheck.SubnetClash
        assertEquals("homebox", c.existing)
        assertEquals(Cidr("10.77.0.0", 24), c.subnet)
    }

    @Test fun refusesDuplicates() {
        val one = listOf(ServerProfile("a", home))
        // same address, re-issued key
        assertEquals(AddCheck.Duplicate("homebox"), ServerList.check(one, home.copy(key = "new")))
        // same name, different address
        assertEquals(AddCheck.Duplicate("homebox"), ServerList.check(one, home.copy(server = "198.51.100.1:443", subnet = "10.79.0.0/24")))
        assertEquals(AddCheck.Ok, ServerList.check(one, vps))
    }

    @Test fun repairingDoesNotClashWithItself() {
        assertEquals(AddCheck.Ok, ServerList.check(list, home.copy(key = "renewed"), replacing = "a"))
        // ...but still with the others
        assertTrue(ServerList.check(list, vps.copy(key = "x"), replacing = "a") is AddCheck.Duplicate)
        assertEquals("b", ServerList.matching(list, vps.copy(key = "x"))?.id)
        assertNull(ServerList.matching(list, home.copy(server = "198.51.100.1:443", sni = "office.tail1234.ts.net")))
    }

    @Test fun ordering() {
        val three = list + ServerProfile("c", home.copy(server = "198.51.100.1:443", serverName = "office", subnet = "10.79.0.0/24"))
        assertEquals(listOf("b", "a", "c"), ServerList.move(three, "b", -1).map { it.id })
        assertEquals(listOf("a", "c", "b"), ServerList.move(three, "b", 1).map { it.id })
        assertEquals(three, ServerList.move(three, "a", -1)) // already first
        assertEquals(three, ServerList.move(three, "c", 1)) // already last
        assertEquals(three, ServerList.move(three, "zz", 1))
    }

    @Test fun migratesTheSingleProfile() {
        val migrated = ServerList.migrate(null, home.toJson()) { "id1" }
        assertEquals(listOf(ServerProfile("id1", home)), migrated)
        // a stored list wins over the legacy profile
        val stored = ServerList.toJson(list)
        assertEquals(list, ServerList.migrate(stored, home.toJson()) { "x" })
        // nothing stored, or garbage
        assertEquals(emptyList<ServerProfile>(), ServerList.migrate(null, null) { "x" })
        assertEquals(emptyList<ServerProfile>(), ServerList.migrate(null, "{not json") { "x" })
        // round trip keeps the order
        assertEquals(list, ServerList.fromJson(ServerList.toJson(list)))
    }

    @Test fun additiveKeysParse() {
        val json = """{"v":1,"server":"203.0.113.9:8443","key":"k","server_name":"vps","subnet":"10.83.142.0/24","dns":["10.83.142.1"]}"""
        val r = Pairing.parse(json)
        assertTrue(r is PairingResult.Ok)
        val p = (r as PairingResult.Ok).payload
        assertEquals("vps", p.serverName)
        assertEquals("10.83.142.0/24", p.subnet)
        // payloads without them still import
        assertTrue(Pairing.parse("""{"v":1,"server":"h:443","key":"k"}""") is PairingResult.Ok)
        assertTrue(Pairing.parse("""{"v":1,"server":"h:443","key":"k","subnet":"nope"}""") is PairingResult.Invalid)
        assertTrue(Pairing.parse("""{"v":1,"server":"h:443","key":"k","subnet":"fd77::/64"}""") is PairingResult.Invalid)
    }

    @Test fun configPerServer() {
        val s = AppSettings()
        val c1 = TunnelPlan.config(home, s)
        val c2 = TunnelPlan.config(vps, s)
        assertEquals("home.example.net", c1.serverAddress)
        assertEquals(443, c1.serverPort)
        assertEquals("homebox.tail1234.ts.net", c1.tlsServerName)
        assertEquals("k1", c1.authKey)
        assertEquals(listOf("10.77.0.1"), c1.dnsServers)
        assertEquals("203.0.113.9", c2.serverAddress)
        assertEquals(8443, c2.serverPort)
        assertEquals("vps.tail1234.ts.net", c2.tlsServerName)
        assertEquals("k2", c2.authKey)
        assertEquals(listOf("10.83.142.1"), c2.dnsServers)
        // each server's own suggested mode, unless the user picked one
        assertEquals(MqvpnConfig.Scheduler.WLB, c1.scheduler)
        assertEquals(MqvpnConfig.Scheduler.MIN_RTT, c2.scheduler)
        assertTrue(c2.redundantEnabled)
        assertEquals(MqvpnConfig.Scheduler.WLB, TunnelPlan.config(vps, s.copy(mode = Mode.SPEED)).scheduler)
        // no dns in the payload: the server's .1
        assertEquals(listOf("10.83.142.1"), TunnelPlan.config(vps.copy(dns = emptyList()), s).dnsServers)
    }

    @Test fun splitRoutesUseEachServersSubnet() {
        val s = AppSettings(routeMode = RouteMode.TAILNET_LAN)
        val r1 = TunnelPlan.routes(home, s, TunnelPlan.dns(home))
        val r2 = TunnelPlan.routes(vps, s, TunnelPlan.dns(vps))
        assertTrue(Cidr("10.77.0.0", 24) in r1)
        assertTrue(Cidr("10.83.142.0", 24) in r2)
        assertFalse(Cidr("10.77.0.0", 24) in r2)
        // DNS inside the subnet needs no host route
        assertFalse(Cidr("10.83.142.1", 32) in r2)
        assertEquals(listOf(Cidr("0.0.0.0", 0), Cidr("::", 0)), TunnelPlan.routes(vps, AppSettings(), TunnelPlan.dns(vps)))
    }
}
