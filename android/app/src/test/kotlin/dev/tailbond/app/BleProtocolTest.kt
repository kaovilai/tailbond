package dev.tailbond.app

import dev.tailbond.app.ble.BleProtocol
import dev.tailbond.app.share.ProxyServer
import dev.tailbond.app.share.ShareProtocol
import kotlinx.serialization.json.Json
import kotlinx.serialization.json.JsonObject
import kotlinx.serialization.json.int
import kotlinx.serialization.json.jsonArray
import kotlinx.serialization.json.jsonObject
import kotlinx.serialization.json.jsonPrimitive
import org.junit.Assert.assertArrayEquals
import org.junit.Assert.assertEquals
import org.junit.Assert.assertNotEquals
import org.junit.Assert.assertNull
import org.junit.Assert.assertTrue
import org.junit.Test
import java.io.File

private fun unhex(s: String) = ByteArray(s.length / 2) { s.substring(it * 2, it * 2 + 2).toInt(16).toByte() }
private fun hex(b: ByteArray) = b.joinToString("") { "%02x".format(it) }

/** The Bluetooth control channel against the Go reference (agent/internal/bleproto/testdata/vectors.json). */
class BleProtocolTest {
    private val v: JsonObject = listOf("../../agent/internal/bleproto/testdata/vectors.json", "../agent/internal/bleproto/testdata/vectors.json")
        .map { File(it) }.first { it.exists() }.readText().let { Json.parseToJsonElement(it).jsonObject }

    private fun JsonObject.s(k: String) = this[k]!!.jsonPrimitive.content

    @Test fun uuids() {
        assertEquals(v.s("service"), BleProtocol.SERVICE.toString())
        assertEquals(v.s("rx"), BleProtocol.RX.toString())
        assertEquals(v.s("tx"), BleProtocol.TX.toString())
    }

    @Test fun loginAndSealMatchTheGoReference() {
        val logins = v["logins"]!!.jsonArray
        assertTrue(logins.size >= 2)
        for (e in logins) {
            val o = e.jsonObject
            val key = o.s("share_key").toByteArray()
            val hub = o.s("hub")
            val exp = o.s("exp").toLong()
            val pk = ShareProtocol.passKey(key, hub, exp)
            assertEquals(o.s("pass_key"), hex(pk))
            val (user, secret) = BleProtocol.credentials(pk, hub, exp)
            assertEquals(o.s("user"), user)
            assertEquals(o.s("secret"), secret)
            // not the proxy's password
            assertEquals(o.s("proxy_pass"), ProxyServer.credentials(pk, hub, exp).second)
            assertNotEquals(secret, o.s("proxy_pass"))
            val cn = o.s("cn")
            val sn = o.s("sn")
            assertEquals(o.s("auth_mac"), BleProtocol.authMac(secret, cn, sn, user))
            assertEquals(o.s("phone_proof"), BleProtocol.phoneProof(secret, cn, sn, user))
            val sk = BleProtocol.sessionKey(secret, cn, sn, user)
            assertEquals(o.s("session_key"), hex(sk))
            assertEquals(o.s("to_phone_sealed"), hex(BleProtocol.seal(sk, BleProtocol.TO_PHONE, 0, o.s("to_phone").toByteArray())))
            assertEquals(o.s("to_mac"), BleProtocol.open(sk, BleProtocol.TO_MAC, 1, unhex(o.s("to_mac_sealed")))!!.decodeToString())
            // the phone accepts the Mac's login, from its own key
            assertEquals(secret, BleProtocol.verifier(key, { 0L })(user, cn, sn, o.s("auth_mac")))
        }
    }

    @Test fun verifierRefuses() {
        val o = v["logins"]!!.jsonArray[0].jsonObject
        val key = o.s("share_key").toByteArray()
        val (u, cn, sn, m) = listOf(o.s("user"), o.s("cn"), o.s("sn"), o.s("auth_mac"))
        assertNull("another nonce", BleProtocol.verifier(key, { 0L })(u, cn, "eA==", m))
        assertNull("another phone", BleProtocol.verifier("other".toByteArray(), { 0L })(u, cn, sn, m))
        assertNull("expired", BleProtocol.verifier(key, { Long.MAX_VALUE })(u, cn, sn, m))
        assertNull("hub not allowed", BleProtocol.verifier(key, { 0L }, { false })(u, cn, sn, m))
        assertNull("no expiry", BleProtocol.verifier(key, { 0L })("mac", cn, sn, m))
    }

    @Test fun sealedMessagesDontOpenElsewhere() {
        val k = BleProtocol.sessionKey("abcdefghijklmnop", "a", "b", "mac/1")
        val s = BleProtocol.seal(k, BleProtocol.TO_PHONE, 7, "hi".toByteArray())
        assertNull("replay at another counter", BleProtocol.open(k, BleProtocol.TO_PHONE, 8, s))
        assertNull("reflected", BleProtocol.open(k, BleProtocol.TO_MAC, 7, s))
        s[0] = (s[0].toInt() xor 1).toByte()
        assertNull("changed", BleProtocol.open(k, BleProtocol.TO_PHONE, 7, s))
    }

    @Test fun chunksMatchTheGoReference() {
        for (e in v["chunks"]!!.jsonArray) {
            val o = e.jsonObject
            val got = BleProtocol.chunks(BleProtocol.frame(o.s("message").toByteArray()), o["mtu"]!!.jsonPrimitive.int)
            assertEquals(o["chunks"]!!.jsonArray.map { it.jsonPrimitive.content }, got.map(::hex))
            // and back
            val r = BleProtocol.Reassembler()
            val msgs = got.flatMap { r.add(it) }
            assertEquals(listOf(o.s("message")), msgs.map { it.decodeToString() })
        }
    }

    @Test fun reassemblerAcrossMessagesAndLimits() {
        val big = ByteArray(700) { 'x'.code.toByte() }
        val stream = BleProtocol.frame("{}".toByteArray()) + BleProtocol.frame(big) + BleProtocol.frame(ByteArray(0))
        val r = BleProtocol.Reassembler()
        val pieces = BleProtocol.chunks(stream, 23)
        assertTrue(pieces.all { it.size <= 20 })
        val msgs = pieces.flatMap { r.add(it) }
        assertEquals(3, msgs.size)
        assertArrayEquals(big, msgs[1])
        assertEquals(0, msgs[2].size)
        assertTrue(runCatching { BleProtocol.Reassembler().add(byteArrayOf(0xff.toByte(), 0xff.toByte())) }.isFailure)
        assertTrue(runCatching { BleProtocol.frame(ByteArray(BleProtocol.MAX_MESSAGE + 1)) }.isFailure)
        assertEquals(1, BleProtocol.chunks("abc".toByteArray(), 0).size)
    }
}
