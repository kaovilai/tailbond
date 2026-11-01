package dev.tailbond.app

import dev.tailbond.app.data.PeerList
import dev.tailbond.app.data.PeersClient
import dev.tailbond.app.model.AppSettings
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
        // never push re-pairing first on a certificate mismatch: it could be an impostor
        val cert = Words.error(ErrorKind.IDENTITY, "homebox", "x:443", null)
        assertEquals(Words.Fix.RETRY, cert.fix)
        assertEquals(Words.Fix.REPAIR, cert.second)
    }

    @Test fun serverName() {
        val p = PairingPayload(server = "203.0.113.7:443", sni = "homebox.tail1234.ts.net", key = "k")
        assertEquals("homebox", serverShortName(p))
        assertEquals("203.0.113.7", serverShortName(p.copy(sni = null)))
    }
}
