package dev.tailbond.app

import dev.tailbond.app.data.AskState
import dev.tailbond.app.data.PairRequestClient
import dev.tailbond.app.data.PairRequestClient.Failure
import dev.tailbond.app.data.next
import dev.tailbond.app.data.phoneRequestName
import dev.tailbond.app.ui.askMessage
import org.junit.After
import org.junit.Assert.assertEquals
import org.junit.Assert.assertFalse
import org.junit.Assert.assertNotNull
import org.junit.Assert.assertNull
import org.junit.Assert.assertTrue
import org.junit.Assert.fail
import org.junit.Test
import java.io.IOException
import java.net.InetAddress

class PairRequestTest {
    private val home = InetAddress.getByAddress("homebox", byteArrayOf(100, 101, 1, 1))
    private val calls = mutableListOf<String>()

    private fun fakeServer(create: Pair<Int, String> = 202 to """{"id":"r1","token":"t0k","status":"pending","name":"pixel-8"}""") =
        PairRequestClient.Transport { _, port, method, path, body, token ->
            calls += "$method $path ${token ?: "-"} ${body ?: ""}".trim()
            assertEquals(PairRequestClient.PORT, port)
            when ("$method $path") {
                "GET /api/hello" -> 200 to """{"tailbond":true,"name":"homebox","version":"1","role":"server"}"""
                "POST /api/pair-requests" -> create
                "GET /api/pair-requests/r1" -> 200 to """{"id":"r1","status":"approved","uri":"tailbond://pair?d=xyz","name":"pixel-8"}"""
                else -> 404 to "{}"
            }
        }

    @After fun reset() {
        PairRequestClient.transport = PairRequestClient.Transport { _, _, _, _, _, _ -> throw IOException("no network in tests") }
        PairRequestClient.resolver = PairRequestClient.Resolver { listOf(InetAddress.getByName(it)) }
    }

    @Test fun tailnetRanges() {
        assertTrue(PairRequestClient.isTailnet(InetAddress.getByName("100.64.0.1")))
        assertTrue(PairRequestClient.isTailnet(InetAddress.getByName("100.127.255.254")))
        assertFalse(PairRequestClient.isTailnet(InetAddress.getByName("100.128.0.1")))
        assertFalse(PairRequestClient.isTailnet(InetAddress.getByName("192.168.8.1")))
        assertTrue(PairRequestClient.isTailnet(InetAddress.getByName("fd7a:115c:a1e0::5")))
        assertFalse(PairRequestClient.isTailnet(InetAddress.getByName("fd77:77::2")))
    }

    @Test fun hostOf() {
        assertEquals("homebox", PairRequestClient.hostOf(" homebox "))
        assertEquals("homebox.tail1234.ts.net", PairRequestClient.hostOf("http://homebox.tail1234.ts.net:8077/#add-phone"))
        assertEquals("100.101.1.1", PairRequestClient.hostOf("100.101.1.1:8077"))
        assertEquals("fd7a:115c:a1e0::1", PairRequestClient.hostOf("[fd7a:115c:a1e0::1]:8077"))
    }

    @Test fun locateOnlyTailnet() {
        PairRequestClient.resolver = PairRequestClient.Resolver { listOf(InetAddress.getByAddress(it, byteArrayOf(100, 101, 1, 1))) }
        assertEquals("100.101.1.1", PairRequestClient.locate("homebox").hostAddress)
        PairRequestClient.resolver = PairRequestClient.Resolver { listOf(InetAddress.getByAddress(it, byteArrayOf(203.toByte(), 0, 113, 7))) }
        expectFailure(Failure.NOT_TAILNET) { PairRequestClient.locate("homebox") }
        PairRequestClient.resolver = PairRequestClient.Resolver { throw java.net.UnknownHostException(it) }
        expectFailure(Failure.NOT_FOUND) { PairRequestClient.locate("homebox") }
    }

    @Test fun askAndPollApproved() {
        PairRequestClient.transport = fakeServer()
        val c = PairRequestClient.ask(home, "pixel-8")
        assertEquals("r1", c.id)
        assertEquals("POST /api/pair-requests - {\"name\":\"pixel-8\",\"kind\":\"phone\"}", calls[1])
        val p = PairRequestClient.poll(home, c)
        assertEquals("GET /api/pair-requests/r1 t0k", calls[2])
        var got: String? = null
        val next = AskState.Waiting("homebox", "pixel-8").next(p) { got = it }
        assertEquals(AskState.Idle, next)
        assertEquals("tailbond://pair?d=xyz", got)
    }

    @Test fun askFailures() {
        PairRequestClient.transport = fakeServer(429 to """{"ok":false}""")
        expectFailure(Failure.TOO_MANY) { PairRequestClient.ask(home, "x") }
        PairRequestClient.transport = fakeServer(403 to """{"ok":false}""")
        expectFailure(Failure.REFUSED) { PairRequestClient.ask(home, "x") }
        PairRequestClient.transport = PairRequestClient.Transport { _, _, _, _, _, _ -> 200 to """{"tailbond":true,"role":"client"}""" }
        expectFailure(Failure.NOT_SERVER) { PairRequestClient.ask(home, "x") }
        PairRequestClient.transport = PairRequestClient.Transport { _, _, _, _, _, _ -> throw java.net.ConnectException("refused") }
        expectFailure(Failure.UNREACHABLE) { PairRequestClient.ask(home, "x") }
    }

    @Test fun pollStates() {
        val w = AskState.Waiting("homebox", "pixel-8")
        assertNull(w.next(PairRequestClient.Polled(status = "pending")) { fail() })
        assertEquals(AskState.Declined("homebox"), w.next(PairRequestClient.Polled(status = "declined")) { fail() })
        assertEquals(AskState.Expired("homebox"), w.next(PairRequestClient.Polled(status = "expired")) { fail() })
        // approved without a link yet: keep waiting
        assertNull(w.next(PairRequestClient.Polled(status = "approved")) { fail() })
        // 404 (server restarted, request gone) reads as expired
        PairRequestClient.transport = PairRequestClient.Transport { _, _, _, _, _, _ -> 404 to "{}" }
        assertEquals("expired", PairRequestClient.poll(home, PairRequestClient.Created(id = "zz", token = "t")).status)
    }

    @Test fun phoneName() {
        assertEquals("pixel-8", phoneRequestName("Pixel 8"))
        assertEquals("sm-s921b", phoneRequestName("SM-S921B"))
        assertEquals("phone", phoneRequestName("  "))
    }

    @Test fun messagesNameTheServer() {
        assertNull(askMessage(AskState.Idle))
        val waiting = askMessage(AskState.Waiting("homebox", "pixel-8"))
        assertNotNull(waiting)
        assertTrue(waiting!!.second.contains("http://homebox:8077") && waiting.second.contains("\"pixel-8\""))
        for (f in Failure.entries) assertNotNull(askMessage(AskState.Failed("homebox", f)))
    }

    private fun expectFailure(f: Failure, block: () -> Unit) {
        try {
            block(); fail("expected $f")
        } catch (e: PairRequestClient.AskException) {
            assertEquals(f, e.failure)
        }
    }
}
