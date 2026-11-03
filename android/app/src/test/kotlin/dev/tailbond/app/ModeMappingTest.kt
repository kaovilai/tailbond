package dev.tailbond.app

import com.mqvpn.sdk.core.model.MqvpnConfig
import com.mqvpn.sdk.core.model.MqvpnConfig.CongestionControl
import com.mqvpn.sdk.core.model.MqvpnConfig.Reinjection
import com.mqvpn.sdk.core.model.MqvpnConfig.ReorderProfile
import com.mqvpn.sdk.core.model.MqvpnConfig.Scheduler
import dev.tailbond.app.model.AppSettings
import dev.tailbond.app.model.Cidr
import dev.tailbond.app.model.Mode
import dev.tailbond.app.model.PathKind
import dev.tailbond.app.model.PathSetting
import dev.tailbond.app.model.RouteMode
import dev.tailbond.app.model.TunnelPlan
import dev.tailbond.app.pairing.PairingPayload
import org.junit.Assert.assertEquals
import org.junit.Assert.assertFalse
import org.junit.Assert.assertNull
import org.junit.Assert.assertTrue
import org.junit.Test

/** Mode → mqvpn config, one test per row of the contract's mode table. */
class ModeMappingTest {
    private val base = MqvpnConfig(serverAddress = "h", authKey = "k")

    @Test fun speed() {
        val c = Mode.SPEED.applyTo(base)
        assertEquals(Scheduler.WLB, c.scheduler)
        assertTrue(c.hybridEnabled)
        assertTrue(c.reorderEnabled)
        assertEquals(ReorderProfile.FIBER_LTE, c.reorderProfile)
        assertEquals(listOf(443), c.reorderPorts)
        assertEquals(Reinjection.DEADLINE, c.reinjection)
        assertTrue(c.requestServerScheduler)
        assertFalse(c.redundantEnabled)
        assertEquals(CongestionControl.BBR2, c.cc)
    }

    @Test fun streaming() {
        val c = Mode.STREAMING.applyTo(base)
        assertEquals(Scheduler.WLB, c.scheduler)
        assertTrue(c.redundantEnabled)
        assertEquals(emptyList<Int>(), c.redundantPorts) // all UDP
        assertEquals(Reinjection.IDLE, c.reinjection)
        assertEquals(CongestionControl.BBR, c.cc)
        assertTrue(c.hybridEnabled)
        assertFalse(c.reorderEnabled)
        assertFalse(c.requestServerScheduler)
    }

    @Test fun redundant() {
        val c = Mode.REDUNDANT.applyTo(base)
        assertEquals(Scheduler.MIN_RTT, c.scheduler)
        assertTrue(c.redundantEnabled)
        assertEquals(Reinjection.IDLE, c.reinjection)
        assertEquals(CongestionControl.BBR, c.cc)
        assertTrue(c.hybridEnabled)
        assertTrue(c.requestServerScheduler)
        assertFalse(c.reorderEnabled)
    }

    @Test fun failover() {
        val c = Mode.FAILOVER.applyTo(base)
        assertEquals(Scheduler.MIN_RTT, c.scheduler)
        assertEquals(Reinjection.OFF, c.reinjection)
        assertFalse(c.redundantEnabled)
        assertFalse(c.hybridEnabled)
        assertFalse(c.reorderEnabled)
        assertEquals(Scheduler.BACKUP_FEC, Mode.FAILOVER.applyTo(base, experimentalBackupFec = true).scheduler)
        assertEquals(Scheduler.WLB, Mode.SPEED.applyTo(base, experimentalBackupFec = true).scheduler)
    }

    @Test fun switchingModesResetsKeys() {
        val c = Mode.FAILOVER.applyTo(Mode.STREAMING.applyTo(Mode.SPEED.applyTo(base)))
        assertEquals(Mode.FAILOVER.applyTo(base), c)
    }

    @Test fun wireNames() {
        assertEquals(listOf("speed", "streaming", "redundant", "failover"), Mode.entries.map { it.wire })
        assertEquals(Mode.REDUNDANT, Mode.fromWire("Redundant"))
        assertNull(Mode.fromWire("turbo"))
    }

    private val pairing = PairingPayload(
        server = "203.0.113.7:8443", sni = "homebox.tail1234.ts.net", key = "psk",
        dns = listOf("10.77.0.1", "1.1.1.1"), mode = "streaming",
        tailnetRoutes = listOf("100.64.0.0/10", "fd7a:115c:a1e0::/48"),
    )

    @Test fun pairingToConfig() {
        val c = TunnelPlan.config(pairing, AppSettings())
        assertEquals("203.0.113.7", c.serverAddress)
        assertEquals(8443, c.serverPort)
        assertEquals("homebox.tail1234.ts.net", c.tlsServerName)
        assertEquals("psk", c.authKey)
        assertEquals(listOf("10.77.0.1", "1.1.1.1"), c.dnsServers)
        assertEquals(Scheduler.WLB, c.scheduler) // streaming from the pairing
        assertTrue(c.redundantEnabled)
        assertFalse(c.insecure)
        assertNull(c.certSha256)
        assertEquals(1280, c.tunMtu) // like tailbondd's client.mtu default
        // the user's choice beats the pairing's suggestion
        assertEquals(Scheduler.MIN_RTT, TunnelPlan.config(pairing, AppSettings(mode = Mode.FAILOVER)).scheduler)
    }

    @Test fun pinnedSelfSigned() {
        val pin = "ab".repeat(32)
        val c = TunnelPlan.config(pairing.copy(insecure = true, certSha256 = pin), AppSettings())
        assertTrue(c.insecure)
        assertEquals(pin, c.certSha256)
        assertEquals(32, c.certPinBytes()!!.size)
        assertEquals(0xab.toByte(), c.certPinBytes()!![0])
    }

    @Test fun defaultDnsAndDisabledPaths() {
        val s = AppSettings().withPath(PathKind.CELLULAR, PathSetting(enabled = false))
        val c = TunnelPlan.config(pairing.copy(dns = emptyList()), s)
        assertEquals(listOf("10.77.0.1"), c.dnsServers)
        assertFalse(c.cellularPathEnabled)
        assertTrue(c.wifiPathEnabled)
    }

    @Test fun routes() {
        val full = TunnelPlan.routes(pairing, AppSettings(), pairing.dns)
        assertEquals(listOf(Cidr("0.0.0.0", 0), Cidr("::", 0)), full)

        val split = TunnelPlan.routes(pairing, AppSettings(routeMode = RouteMode.TAILNET_LAN), pairing.dns)
        assertEquals(
            listOf(
                Cidr("10.77.0.0", 24),
                Cidr("100.64.0.0", 10),
                Cidr("fd7a:115c:a1e0::", 48),
                Cidr("1.1.1.1", 32), // DNS outside the routed prefixes; 10.77.0.1 is covered
            ),
            split,
        )
    }

    @Test fun settingsJsonRoundTrip() {
        val s = AppSettings(
            mode = Mode.REDUNDANT, routeMode = RouteMode.TAILNET_LAN, excludedApps = setOf("a.b"),
            cellularCapMb = 500,
        ).withPath(PathKind.WIFI, PathSetting(true, dev.tailbond.app.model.Priority.BACKUP))
        assertEquals(s, AppSettings.fromJson(s.toJson()))
        assertEquals(AppSettings(), AppSettings.fromJson("garbage"))
    }
}
