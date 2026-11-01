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
import dev.tailbond.app.model.RouteMode
import dev.tailbond.app.model.SwitchLog
import dev.tailbond.app.model.SwitchRecord
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
import org.junit.Assert.assertFalse
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

    private val at = { _: Long -> "14:02" }
    private fun note(f: FailoverView) = Words.serverNote(f, names, at)

    @Test fun failoverWording() {
        val since = view("v", ServerState.UNREACHABLE, ServerState.ACTIVE).copy(sinceWallMs = 1L)
        assertEquals("Using vps since 14:02 (homebox is unreachable)", note(since)?.line)
        assertTrue(note(since)!!.detail!!.contains("goes back to homebox by itself"))
        // no switch time known: no "since"
        assertEquals("Using vps (homebox is unreachable)", note(view("v", ServerState.UNREACHABLE, ServerState.ACTIVE))?.line)

        val counting = note(view("v", ServerState.STANDBY, ServerState.ACTIVE, failbackIn = 80_000))!!
        assertEquals("Using vps (homebox is answering again)", counting.line)
        assertTrue(counting.detail!!.startsWith("Back to homebox in 1:20"))
        assertEquals(Words.NoteAction("Stay on vps", "v"), counting.action)
        assertTrue(note(view("v", ServerState.STANDBY, ServerState.ACTIVE, failbackIn = 400))!!.detail!!.startsWith("Back to homebox in 0:01"))
        assertEquals(
            "Using vps (homebox is answering again) · Back to homebox in 1:20",
            Words.serverNoteShort(view("v", ServerState.STANDBY, ServerState.ACTIVE, failbackIn = 80_000), names),
        )

        val back = Switch("h", "v", SwitchKind.FAILBACK, 1_000_000 - 30_000)
        val backOn = view("h", ServerState.ACTIVE, ServerState.STANDBY, FailoverReason.NONE, last = back).copy(sinceWallMs = 1L)
        assertEquals("Back on homebox since 14:02", note(backOn)?.line)
        // long after the failback, or while it's still connecting: nothing to say
        val old = back.copy(atMs = 1_000_000 - Words.BACK_ON_MS - 1)
        assertNull(note(view("h", ServerState.ACTIVE, ServerState.STANDBY, FailoverReason.NONE, last = old)))
        assertNull(note(view("h", ServerState.CONNECTING, ServerState.STANDBY, FailoverReason.NONE, last = back)))

        val manual = note(view("v", ServerState.STANDBY, ServerState.ACTIVE, FailoverReason.MANUAL))!!
        assertEquals("Using vps (you picked it)", manual.line)
        assertEquals("Stays on vps until you pick homebox or vps stops working.", manual.detail)
        assertEquals(Words.NoteAction("Go back to homebox", "h", filled = true), manual.action)
        assertEquals("Always using vps", note(view("v", ServerState.STANDBY, ServerState.ACTIVE, FailoverReason.POLICY))?.line)

        val switching = view("v", ServerState.UNREACHABLE, ServerState.CONNECTING, last = Switch("v", "h", SwitchKind.FAILOVER, 1_000_000 - 3_000))
        assertEquals("Switching to vps… (homebox isn't answering)", note(switching)?.line)
        assertTrue(note(switching)!!.action!!.vpnSettings)
        assertFalse(counting.action!!.filled) // the countdown is the message; staying is a quiet option
        assertNull(Words.serverNote(null, names))
    }

    @Test fun internetTrafficNamesTheServerInUse() {
        assertEquals("Through your home server", Words.routeLabel(RouteMode.FULL, multi = false))
        assertEquals("Through your server", Words.routeLabel(RouteMode.FULL, multi = true))
        val explain = Words.routeExplain(RouteMode.FULL, listOf("homebox", "vps"))
        assertTrue(explain.contains("homebox, or vps while homebox is down"))
        assertEquals(RouteMode.FULL.explain, Words.routeExplain(RouteMode.FULL, listOf("homebox")))
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
        assertTrue(clash.second.contains("homebox keeps working"))
        assertTrue(clash.second.contains("10.77.0.0/24"))
        assertTrue(clash.second.contains("no devices yet moves to a free range by itself"))
        val msg = Words.clashMessage("vps", "homebox", Cidr("10.77.0.0", 24))
        assertTrue(msg.contains("--subnet 10.78.0.0/24"))
        assertTrue(msg.contains("uci set tailbond.server.subnet=10.78.0.0/24"))
        assertTrue(msg.contains("\"subnet\" under \"server\""))
        assertNull(Words.addProblem("vps", AddCheck.Ok))
        assertTrue(Words.addProblem("homebox", AddCheck.Duplicate("homebox"))!!.second.contains("Pair again"))
    }

    @Test fun allServersFailedError() {
        val e = Words.error(ErrorKind.UNREACHABLE, "vps", "x:443", "Connection timeout", listOf("homebox", "vps"))
        assertEquals("Can't reach any of your servers", e.title)
        assertTrue(e.body.contains("homebox and vps"))
        assertTrue(e.body.contains("normal internet now, without tailbond"))
        assertTrue(e.body.contains("neither homebox nor vps answers"))
        assertFalse(e.body.contains("timeout")) // no technical detail
    }

    @Test fun websitesSummaryNamesTheAndroidLimit() {
        assertEquals("Needs Android 13 or newer", Words.websitesSummary(AppSettings(), 32))
        assertEquals("None", Words.websitesSummary(AppSettings(), 33))
    }

    @Test fun recentSwitches() {
        var log = emptyList<SwitchRecord>()
        repeat(25) { i -> log = SwitchLog.add(log, SwitchRecord(i.toLong(), "homebox", "vps", SwitchLog.DOWN)) }
        assertEquals(SwitchLog.MAX, log.size)
        assertEquals(24L, log.first().atMs) // newest first
        assertEquals(log, SwitchLog.fromJson(SwitchLog.toJson(log)))
        assertEquals(emptyList<SwitchRecord>(), SwitchLog.fromJson("nope"))
        assertEquals("homebox → vps: homebox stopped answering", Words.switchText(SwitchRecord(0, "homebox", "vps", SwitchLog.DOWN)))
        assertEquals("vps → homebox: homebox was working again", Words.switchText(SwitchRecord(0, "vps", "homebox", SwitchLog.BACK)))
        assertEquals("homebox → vps: you picked vps", Words.switchText(SwitchRecord(0, "homebox", "vps", SwitchLog.PICKED)))
        val zone = java.time.ZoneId.systemDefault()
        val day = java.time.LocalDate.of(2026, 10, 21)
        val t = day.atTime(14, 2).atZone(zone).toInstant().toEpochMilli()
        val before = day.minusDays(1).atTime(14, 2).atZone(zone).toInstant().toEpochMilli()
        assertEquals("14:02", Words.dayTime(t, t + 3_600_000))
        assertEquals("Oct 20, 14:02", Words.dayTime(before, t))
    }

    @Test fun blockConnectionsIsRecommendedWithBackups() {
        assertTrue(Words.blockAdvice(multi = true).startsWith("Recommended with backup servers"))
        assertFalse(Words.blockAdvice(multi = false).contains("Recommended"))
    }
}
