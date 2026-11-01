package dev.tailbond.app

import dev.tailbond.app.model.Mode
import dev.tailbond.app.pairing.Pairing
import dev.tailbond.app.pairing.PairingPayload
import dev.tailbond.app.pairing.PairingResult
import org.junit.Assert.assertEquals
import org.junit.Assert.assertNull
import org.junit.Assert.assertTrue
import org.junit.Test
import java.util.Base64

class PairingTest {
    // the contract's example payload (docs/architecture.md)
    private val contractJson = """
        {
          "v": 1,
          "name": "pixel",
          "server": "203.0.113.7:443",
          "sni": "homebox.tail1234.ts.net",
          "user": "pixel",
          "key": "c2VjcmV0LXBzaw",
          "insecure": false,
          "dns": ["10.77.0.1"],
          "mode": "speed",
          "tailnet_routes": ["100.64.0.0/10", "fd7a:115c:a1e0::/48"]
        }
    """.trimIndent()

    private fun uri(json: String) =
        "tailbond://pair?d=" + Base64.getUrlEncoder().withoutPadding().encodeToString(json.toByteArray())

    private fun ok(input: String): PairingPayload {
        val r = Pairing.parse(input)
        assertTrue("expected Ok, got $r", r is PairingResult.Ok)
        return (r as PairingResult.Ok).payload
    }

    private fun invalid(input: String): String {
        val r = Pairing.parse(input)
        assertTrue("expected Invalid, got $r", r is PairingResult.Invalid)
        return (r as PairingResult.Invalid).reason
    }

    @Test fun parsesContractExample() {
        val p = ok(uri(contractJson))
        assertEquals("pixel", p.name)
        assertEquals("203.0.113.7", p.host)
        assertEquals(443, p.port)
        assertEquals("homebox.tail1234.ts.net", p.sni)
        assertEquals("c2VjcmV0LXBzaw", p.key)
        assertEquals(listOf("10.77.0.1"), p.dns)
        assertEquals(Mode.SPEED, Mode.fromWire(p.mode))
        assertEquals(listOf("100.64.0.0/10", "fd7a:115c:a1e0::/48"), p.tailnetRoutes)
        assertNull(p.certSha256)
    }

    @Test fun roundTripsThroughToUri() {
        val p = ok(uri(contractJson))
        assertEquals(p, ok(p.toUri()))
    }

    @Test fun acceptsPaddedStandardBase64AndRawBlobAndJson() {
        val std = Base64.getEncoder().encodeToString(contractJson.toByteArray()) // may contain + / =
        assertEquals("pixel", ok("tailbond://pair?d=$std").name)
        assertEquals("pixel", ok(Base64.getUrlEncoder().encodeToString(contractJson.toByteArray())).name)
        assertEquals("pixel", ok(contractJson).name)
        assertEquals("pixel", ok("  " + uri(contractJson) + "\n").name)
    }

    @Test fun ddnsHostnameAndDefaultPort() {
        val p = ok(uri("""{"v":1,"server":"home.example.net","key":"k"}"""))
        assertEquals("home.example.net", p.host)
        assertEquals(443, p.port)
        assertEquals("speed", p.mode)
    }

    @Test fun ipv6Server() {
        val p = ok(uri("""{"v":1,"server":"[2001:db8::1]:8443","key":"k"}"""))
        assertEquals("2001:db8::1", p.host)
        assertEquals(8443, p.port)
    }

    @Test fun insecureWithPinIsNormalised() {
        val pin = "AB:" + "cd".repeat(31)
        val p = ok(uri("""{"v":1,"server":"1.2.3.4:443","key":"k","insecure":true,"cert_sha256":"$pin"}"""))
        assertTrue(p.insecure)
        assertEquals("ab" + "cd".repeat(31), p.certSha256)
    }

    @Test fun ignoresUnknownFields() {
        assertEquals("k", ok(uri("""{"v":1,"server":"h:1","key":"k","future":42}""")).key)
    }

    @Test fun rejectsBadInput() {
        invalid("https://example.com/pair?d=abc")
        invalid("tailbond://other?d=abc")
        invalid("tailbond://pair")
        invalid("tailbond://pair?d=!!!")
        invalid(uri("""{"v":2,"server":"h:1","key":"k"}"""))
        invalid(uri("""{"v":1,"server":"h:1","key":""}"""))
        invalid(uri("""{"v":1,"server":"h:99999","key":"k"}"""))
        invalid(uri("""{"v":1,"server":"h:1","key":"k","mode":"turbo"}"""))
        invalid(uri("""{"v":1,"server":"h:1","key":"k","cert_sha256":"abc"}"""))
        invalid(uri("""{"v":1,"server":"h:1","key":"k","tailnet_routes":["100.64.0.0/40"]}"""))
        invalid(uri("""{"v":1,"server":"h:1","key":"k","dns":["dns.google"]}"""))
        invalid(uri("""{"v":1,"server":"bad host:1","key":"k"}"""))
    }

    @Test
    fun askLinkFromThePhonePage() {
        assertEquals("homebox.tail1234.ts.net", Pairing.askServer("tailbond://ask?server=homebox.tail1234.ts.net"))
        assertEquals("homebox", Pairing.askServer("tailbond://ask?server=homebox"))
        assertEquals("100.101.1.1", Pairing.askServer("tailbond://ask?server=100.101.1.1"))
        assertNull(Pairing.askServer("tailbond://pair?d=abc"))
        assertNull(Pairing.askServer("tailbond://ask"))
        assertNull(Pairing.askServer("tailbond://ask?server=evil%20host"))
        assertNull(Pairing.askServer("tailbond://ask?server=http%3A%2F%2Fx"))
        assertNull(Pairing.askServer("https://ask?server=homebox"))
    }
}
