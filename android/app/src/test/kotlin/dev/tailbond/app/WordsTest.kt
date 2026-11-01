package dev.tailbond.app

import dev.tailbond.app.data.PeerList
import dev.tailbond.app.data.PeersClient
import dev.tailbond.app.failover.FailoverReason
import dev.tailbond.app.failover.FailoverView
import dev.tailbond.app.failover.ServerState
import dev.tailbond.app.failover.ServerStatus
import dev.tailbond.app.failover.Switch
import dev.tailbond.app.failover.SwitchKind
import dev.tailbond.app.model.AddCheck
import dev.tailbond.app.model.AppSettings
import dev.tailbond.app.model.Cidr
import dev.tailbond.app.model.Mode
import dev.tailbond.app.model.PathKind
import dev.tailbond.app.model.Words
import dev.tailbond.app.model.Words.LinkStatus
import dev.tailbond.app.model.serverShortName
import dev.tailbond.app.pairing.PairingPayload
import dev.tailbond.app.service.ConnState
import dev.tailbond.app.service.ErrorKind
import dev.tailbond.app.service.Live
import dev.tailbond.app.service.PathGate
import dev.tailbond.app.service.PathSample
import org.junit.Assert.assertEquals
import org.junit.Assert.assertNull
import org.junit.Assert.assertTrue
import org.junit.Test

class WordsTest {
    private fun sample(kind: PathKind, status: Int) = PathSample("${kind.name.lowercase()}-1", kind, status, 20, 0, 0, 0, 0)
    private fun live(vararg p: PathSample, standby: Boolean = false) =
        Live(state = ConnState.CONNECTED, paths = p.toList(), standbyActive = standby)

    @Test fun apiTokenMatchesAgent() {
        // same vector as agent/internal/agent/peers_test.go and docs/architecture.md
        assertEquals("68c8131634495b2e11cfcb57f604aa091793b004cfac7f5ae50ae49d553c1747", PeersClient.token("test-key-123"))
    }

    @Test fun peerListParsesContractShape() {
        val p = PeerList.fromJson(
            """{"server":"homebox","magic_dns_suffix":"tail1234.ts.net","tailscale_running":true,"client":"pixel",
              "peers":[{"hostname":"nas","dns_name":"nas.tail1234.ts.net","tailscale_ips":["100.101.2.2","fd7a:115c:a1e0::2"],
              "os":"linux","online":true}]}""",
        )!!
        assertEquals("nas.tail1234.ts.net", p.peers[0].dnsName)
        assertEquals("100.101.2.2", p.peers[0].ipv4)
        assertEquals("fd7a:115c:a1e0::2", p.peers[0].ipv6)
    }

    @Test fun bothInUse() {
        val l = live(sample(PathKind.WIFI, PathGate.STATUS_ACTIVE), sample(PathKind.CELLULAR, PathGate.STATUS_ACTIVE))
        assertEquals("Using both Wi-Fi and mobile data", Words.connectionsSummary(l, AppSettings(), Mode.SPEED, false))
        assertEquals("Wi-Fi + mobile data", Words.activeKinds(l))
    }

    @Test fun saveDataHoldsCellular() {
        val l = live(sample(PathKind.WIFI, PathGate.STATUS_ACTIVE), sample(PathKind.CELLULAR, -1))
        assertEquals(LinkStatus.SAVING, Words.linkStatus(PathKind.CELLULAR, l, AppSettings(), Mode.FAILOVER, false))
        assertTrue(Words.connectionsSummary(l, AppSettings(), Mode.FAILOVER, false).contains("saving mobile data"))
    }

    @Test fun takeoverWhenWifiFails() {
        val l = live(sample(PathKind.WIFI, PathGate.STATUS_CLOSED), sample(PathKind.CELLULAR, PathGate.STATUS_ACTIVE), standby = true)
        assertEquals(LinkStatus.TAKING_OVER, Words.linkStatus(PathKind.CELLULAR, l, AppSettings(), Mode.FAILOVER, false))
        assertEquals("Wi-Fi isn't working · using mobile data", Words.connectionsSummary(l, AppSettings(), Mode.FAILOVER, false))
    }

    @Test fun errorsNameTheFix() {
        assertEquals(Words.Fix.RETRY, Words.error(ErrorKind.UNREACHABLE, "homebox", "home.example.net:443", null).fix)
        assertTrue(Words.error(ErrorKind.UNREACHABLE, "homebox", "home.example.net:443", null).body.contains("UDP port 443"))
        // never offer re-pairing on a certificate mismatch: it could be an impostor
        val cert = Words.error(ErrorKind.IDENTITY, "homebox", "x:443", null)
        assertEquals(Words.Fix.RETRY, cert.fix)
        assertEquals(null, cert.second)
    }

    @Test fun serverName() {
        val p = PairingPayload(server = "203.0.113.7:443", sni = "homebox.tail1234.ts.net", key = "k")
        assertEquals("homebox", serverShortName(p))
        assertEquals("203.0.113.7", serverShortName(p.copy(sni = null)))
    }

    // --- servers and failover ---

    private val names = mapOf("h" to "homebox", "v" to "vps")
    private fun view(
        active: String,
        home: ServerState,
        vps: ServerState,
        reason: FailoverReason = FailoverReason.PRIMARY_DOWN,
        failbackIn: Long? = null,
        last: Switch? = null,
    ) = FailoverView(
        activeId = active,
        servers = listOf(ServerStatus("h", home, null), ServerStatus("v", vps, null)),
        reason = reason,
        failbackTo = if (failbackIn != null) "h" else null,
        failbackInMs = failbackIn,
        lastSwitch = last,
        nowMs = 1_000_000,
    )

    @Test fun failoverWording() {
        assertEquals("Using vps (homebox is unreachable)", Words.serverLine(view("v", ServerState.UNREACHABLE, ServerState.ACTIVE), names))
        assertEquals("Back to homebox in 1:20", Words.serverLine(view("v", ServerState.STANDBY, ServerState.ACTIVE, failbackIn = 80_000), names))
        assertEquals("Back to homebox in 0:01", Words.serverLine(view("v", ServerState.STANDBY, ServerState.ACTIVE, failbackIn = 400), names))
        val back = Switch("h", "v", SwitchKind.FAILBACK, 1_000_000 - 30_000)
        assertEquals("Back on homebox", Words.serverLine(view("h", ServerState.ACTIVE, ServerState.STANDBY, FailoverReason.NONE, last = back), names))
        // long after the failback, or while it's still connecting: nothing to say
        val old = back.copy(atMs = 1_000_000 - Words.BACK_ON_MS - 1)
        assertNull(Words.serverLine(view("h", ServerState.ACTIVE, ServerState.STANDBY, FailoverReason.NONE, last = old), names))
        assertNull(Words.serverLine(view("h", ServerState.CONNECTING, ServerState.STANDBY, FailoverReason.NONE, last = back), names))
        assertEquals("Using vps (you picked it)", Words.serverLine(view("v", ServerState.STANDBY, ServerState.ACTIVE, FailoverReason.MANUAL), names))
        assertEquals("Always using vps", Words.serverLine(view("v", ServerState.STANDBY, ServerState.ACTIVE, FailoverReason.POLICY), names))
        assertNull(Words.serverLine(null, names))
    }

    @Test fun serverStatusWords() {
        val v = view("h", ServerState.ACTIVE, ServerState.STANDBY, FailoverReason.NONE)
        assertEquals("In use" to true, Words.serverStatus("h", v, on = true, isPrimary = true))
        assertEquals("Ready as backup" to true, Words.serverStatus("v", v, on = true, isPrimary = false))
        assertEquals("Unreachable" to false, Words.serverStatus("h", view("v", ServerState.UNREACHABLE, ServerState.ACTIVE), true, true))
        assertEquals("Backup" to null, Words.serverStatus("v", null, on = false, isPrimary = false))
        assertEquals("Prefer homebox, use vps if homebox is down", Words.preferLabel(listOf("homebox", "vps")))
        assertEquals("Prefer homebox, then vps and office if homebox is down", Words.preferLabel(listOf("homebox", "vps", "office")))
        assertTrue(Words.preferExplain("homebox", AppSettings()).contains("15 seconds"))
        assertTrue(Words.preferExplain("homebox", AppSettings()).contains("2 minutes"))
    }

    @Test fun addProblemsSayWhatToDo() {
        val clash = Words.addProblem("vps", AddCheck.SubnetClash("homebox", Cidr("10.77.0.0", 24), Cidr("10.77.0.0", 24)))!!
        assertEquals("vps uses the same addresses as homebox", clash.first)
        assertTrue(clash.second.contains("\"subnet\" under \"server\""))
        assertTrue(clash.second.contains("10.77.0.0/24"))
        assertNull(Words.addProblem("vps", AddCheck.Ok))
        assertTrue(Words.addProblem("homebox", AddCheck.Duplicate("homebox"))!!.second.contains("Pair again"))
    }

    @Test fun allServersFailedError() {
        val e = Words.error(ErrorKind.UNREACHABLE, "vps", "x:443", null, listOf("homebox", "vps"))
        assertEquals("Can't reach any of your servers", e.title)
        assertTrue(e.body.contains("homebox and vps"))
    }

    @Test fun websitesSummaryNamesTheAndroidLimit() {
        assertEquals("Needs Android 13 or newer", Words.websitesSummary(AppSettings(), 32))
        assertEquals("None", Words.websitesSummary(AppSettings(), 33))
    }
}
