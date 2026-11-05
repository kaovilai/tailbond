package dev.tailbond.app

import dev.tailbond.app.share.BleProto
import dev.tailbond.app.share.ProxyServer
import dev.tailbond.app.share.ShareProtocol
import dev.tailbond.app.share.hexBytes
import kotlinx.serialization.json.Json
import kotlinx.serialization.json.jsonArray
import kotlinx.serialization.json.jsonObject
import kotlinx.serialization.json.jsonPrimitive
import org.junit.Assert.assertEquals
import org.junit.Assert.assertFalse
import org.junit.Assert.assertNotEquals
import org.junit.Assert.assertTrue
import org.junit.Test

/**
 * The BLE control channel of "One click from the Mac" (share/BleProto.kt,
 * docs/phone-hub.md): the framing (chunked to the ATT MTU), the
 * challenge-response login (the same pass as the proxy, its own label —
 * the Go reference's vectors), and every command, against a fake transport
 * and fake commands. The GATT server itself (Bluetooth) can't run in a
 * JVM test or on the emulator; these tests are its contract.
 */
class BleProtoTest {
    private companion object {
        val key = "home-share-key".toByteArray(Charsets.UTF_8)
        const val hub = "mac"
        const val exp = 4102444800L // far future, as a real pass would be
    }

    /** What the commands did, for the assertions. */
    private class Cmds : BleProto.Commands {
        var hubOn: Boolean? = null
        var hotspot = "off"
        val replies = ArrayDeque<Triple<String?, String?, String?>>() // ssid, pass, error
        var stopped: String? = null

        override fun hotspotOn(reply: (String?, String?, String?) -> Unit) {
            val r = replies.removeFirstOrNull() ?: Triple("Pixel", "hotspot-pass", null)
            if (r.third == null) hotspot = "on"
            reply(r.first, r.second, r.third)
        }

        override fun hotspotOff(reply: (String?) -> Unit) {
            stopped = "asked"
            hotspot = "off"
            reply(null)
        }

        override fun hotspotState() = BleProto.HotspotState(hotspot, if (hotspot == "on") "Pixel" else "")

        override fun hubSet(on: Boolean, reply: (String?) -> Unit) {
            hubOn = on
            reply(null)
        }

        override fun status() = BleProto.Status(
            battery = 78, charging = true, net = "mobile data + Wi-Fi", hub = if (hubOn == true) "ready" else "off",
            paths = 2, hotspot = hotspot, hotspotSsid = if (hotspot == "on") "Pixel" else "", shizuku = "",
        )
    }

    private class Link(cmds: Cmds, key: ByteArray, nowSec: () -> Long = { 0L }, val mtu: Int = 23) {
        val sent = mutableListOf<Pair<String, ByteArray>>()
        private var nonce = 0
        val proto = BleProto(
            keyId = { ShareProtocol.keyId(key) },
            ownKey = { key },
            cmds = cmds,
            nowSec = nowSec,
            rng = { n -> ByteArray(n) { ((++nonce + it) and 0xff).toByte() } }, // fresh per read, deterministic
            log = {},
        )

        init {
            proto.out = { d, f -> sent.add(d to f) }
        }

        /** The Mac's half: write one JSON message, chunked as CoreBluetooth would. */
        fun write(vararg json: Pair<String, Any?>, device: String = "mac") {
            val payload = Json.encodeToString(kotlinx.serialization.json.JsonObject.serializer(), jsonObj(*json)).toByteArray()
            for (c in BleProto.chunkFrame(BleProto.encode(payload), mtu)) {
                proto.onWrite(device, c)
            }
        }

        fun hello(device: String = "mac", hub: String = "mac", exp: Long = 4102444800L, pass: ByteArray = key, nonce: String = "00112233445566778899aabbccddeeff") {
            val ch = proto.challenge(device).copyOfRange(8, 24)
            val (_, bleKey) = ProxyServer.bleLogin(ShareProtocol.passKey(pass, hub, exp), hub, exp)
            write("id" to 1, "cmd" to "hello", "user" to "$hub/$exp", "nonce" to nonce, "proof" to BleProto.proof(bleKey, ch), device = device)
        }

        /** The proof the phone should answer with, over the Mac's nonce. */
        fun phoneProof(hub: String = "mac", exp: Long = 4102444800L, nonce: String = "00112233445566778899aabbccddeeff"): String {
            val (_, k) = ProxyServer.bleLogin(ShareProtocol.passKey(key, hub, exp), hub, exp)
            return BleProto.proof(k, nonce.hexBytes())
        }

        /** The frames the phone sent, as JSON. */
        fun replies(device: String = "mac"): List<kotlinx.serialization.json.JsonObject> =
            sent.filter { it.first == device }.map { (_, f) ->
                Json.parseToJsonElement(String(f, 2, f.size - 2)).jsonObject
            }
    }

    @Test fun framingRoundTripsAtAnyMtu() {
        // a whole conversation at a tiny MTU (5: 1 flags byte + 1 data byte
        // per chunk): the reassembly must put it back together
        for (mtu in intArrayOf(23, 512, 7, 6)) {
            val link = Link(Cmds(), key, mtu = mtu)
            link.hello() // a hello is the longest message: user + 64-char proof
            val r = link.replies().last()
            assertEquals("mtu $mtu", "true", r["ok"]!!.jsonPrimitive.content)
        }
        // and the chunks themselves fit what ATT carries
        val chunks = BleProto.chunkFrame(BleProto.encode(ByteArray(300) { 'x'.code.toByte() }), 23)
        for (c in chunks) assertTrue("chunk of ${c.size} bytes at MTU 23", c.size <= 20)
        assertTrue("a 300-byte frame needs many chunks at MTU 23", chunks.size > 15)
        // a length that lies: the frame is dropped, not half-delivered
        val link = Link(Cmds(), key)
        val lie = BleProto.encode("""{"id":9,"cmd":"status"}""".toByteArray())
        link.proto.onWrite("mac", byteArrayOf(0x03) + byteArrayOf(0xff.toByte(), 0xff.toByte()) + lie.copyOfRange(2, lie.size))
        assertEquals(0, link.sent.size)
    }

    @Test fun singleChunkWhenItFits() {
        val frame = BleProto.encode("""{"a":1}""".toByteArray())
        val chunks = BleProto.chunkFrame(frame, 23)
        assertEquals(1, chunks.size)
        assertEquals(0x03, chunks[0][0].toInt() and 0xff) // START | END
    }

    @Test fun helloWithTheRightPassIsAcceptedAndCommandsWork() {
        val cmds = Cmds()
        val link = Link(cmds, key)
        link.hello()
        val r = link.replies().last()
        assertEquals(1, r["id"]!!.jsonPrimitive.content.toInt())
        assertEquals("true", r["ok"]!!.jsonPrimitive.content)
        assertEquals(ShareProtocol.keyId(key), r["key_id"]!!.jsonPrimitive.content)
        // mutual: the phone proves it holds the key, over the Mac's nonce
        assertEquals(link.phoneProof(), r["proof"]!!.jsonPrimitive.content)

        // hotspot.on: the reply carries the hotspot's network
        link.write("id" to 2, "cmd" to "hotspot.on")
        val h = link.replies().last()
        assertEquals("true", h["ok"]!!.jsonPrimitive.content)
        assertEquals("Pixel", h["ssid"]!!.jsonPrimitive.content)
        assertEquals("hotspot-pass", h["pass"]!!.jsonPrimitive.content)

        // hub.on / hub.off toggle the proxy
        link.write("id" to 3, "cmd" to "hub.on")
        assertEquals("true", link.replies().last()["ok"]!!.jsonPrimitive.content)
        assertEquals(true, cmds.hubOn)
        link.write("id" to 4, "cmd" to "hub.off")
        assertEquals(false, cmds.hubOn)

        // hotspot.state
        link.write("id" to 5, "cmd" to "hotspot.off")
        link.write("id" to 6, "cmd" to "hotspot.state")
        val s = link.replies().last()
        assertEquals("off", s["state"]!!.jsonPrimitive.content)

        // status
        link.write("id" to 7, "cmd" to "status")
        val st = link.replies().last()["status"]!!.jsonObject
        assertEquals("78", st["battery"]!!.jsonPrimitive.content)
        assertEquals("true", st["charging"]!!.jsonPrimitive.content)
        assertEquals("mobile data + Wi-Fi", st["net"]!!.jsonPrimitive.content)
        assertEquals("2", st["paths"]!!.jsonPrimitive.content)
    }

    @Test fun nothingBeforeHello() {
        val link = Link(Cmds(), key)
        link.write("id" to 1, "cmd" to "hotspot.on")
        val r = link.replies().last()
        assertEquals("false", r["ok"]!!.jsonPrimitive.content)
        assertEquals("say hello first", r["error"]!!.jsonPrimitive.content)
    }

    @Test fun wrongPasswordIsRefusedAndStaysRefused() {
        val link = Link(Cmds(), key)
        link.hello(pass = "someone-elses-key".toByteArray())
        assertEquals("false", link.replies().last()["ok"]!!.jsonPrimitive.content)
        // and commands still don't work
        link.write("id" to 2, "cmd" to "hub.on")
        assertEquals("say hello first", link.replies().last()["error"]!!.jsonPrimitive.content)
    }

    @Test fun expiredPassIsRefused() {
        val link = Link(Cmds(), key, nowSec = { 4202444800L })
        link.hello(exp = 4102444800L)
        assertEquals("false", link.replies().last()["ok"]!!.jsonPrimitive.content)
    }

    @Test fun unknownCommandIsRefused() {
        val link = Link(Cmds(), key)
        link.hello()
        link.write("id" to 2, "cmd" to "ring")
        val r = link.replies().last()
        assertEquals("false", r["ok"]!!.jsonPrimitive.content)
        assertEquals("unknown command", r["error"]!!.jsonPrimitive.content)
    }

    @Test fun challengeIsFreshPerReadAndSingleUse() {
        val link = Link(Cmds(), key)
        val ch1 = link.proto.challenge("mac")
        val ch2 = link.proto.challenge("mac")
        assertFalse("a new read gives a new challenge", ch1.contentEquals(ch2))
        // hello signed over the stale challenge fails
        val (_, bleKey) = ProxyServer.bleLogin(ShareProtocol.passKey(key, "mac", exp), "mac", exp)
        link.write("id" to 1, "cmd" to "hello", "user" to "mac/$exp", "nonce" to "00112233445566778899aabbccddeeff", "proof" to BleProto.proof(bleKey, ch1.copyOfRange(8, 24)), device = "mac")
        assertEquals("false", link.replies().last()["ok"]!!.jsonPrimitive.content)
    }

    @Test fun helloWithoutANonceIsRefused() {
        // the Mac's nonce is what makes the phone prove itself: without it
        // there is nothing mutual, so the phone says no
        val link = Link(Cmds(), key)
        val ch = link.proto.challenge("mac").copyOfRange(8, 24)
        val (_, bleKey) = ProxyServer.bleLogin(ShareProtocol.passKey(key, "mac", exp), "mac", exp)
        link.write("id" to 1, "cmd" to "hello", "user" to "mac/$exp", "proof" to BleProto.proof(bleKey, ch))
        assertEquals("false", link.replies().last()["ok"]!!.jsonPrimitive.content)
        link.write("id" to 2, "cmd" to "hub.on")
        assertEquals("say hello first", link.replies().last()["error"]!!.jsonPrimitive.content)
    }

    @Test fun theReplyProofIsOverTheMacsNonce() {
        // a different nonce must produce a different proof (a replay of the
        // phone's proof from another attempt doesn't verify)
        val link = Link(Cmds(), key)
        link.hello(nonce = "ffffffffffffffffffffffffffffffff")
        val r = link.replies().last()
        assertEquals(link.phoneProof(nonce = "ffffffffffffffffffffffffffffffff"), r["proof"]!!.jsonPrimitive.content)
        assertNotEquals(link.phoneProof(), r["proof"]!!.jsonPrimitive.content)
    }

    @Test fun hotspotRefusalCarriesTheError() {
        val cmds = Cmds()
        cmds.replies.add(Triple(null, null, "Shizuku isn't running on this phone (start it, or start the hotspot by hand)"))
        val link = Link(cmds, key)
        link.hello()
        link.write("id" to 2, "cmd" to "hotspot.on")
        val r = link.replies().last()
        assertEquals("false", r["ok"]!!.jsonPrimitive.content)
        assertTrue(r["error"]!!.jsonPrimitive.content.contains("Shizuku"))
        assertEquals("off", cmds.hotspot) // nothing changed
    }

    @Test fun garbageIsAnsweredWithAnError() {
        val link = Link(Cmds(), key)
        link.write("cmd" to "hello") // no id at all, but parseable
        val r = link.replies().last()
        assertEquals("false", r["ok"]!!.jsonPrimitive.content)
        // a frame that isn't JSON at all: no crash, nothing sent
        val before = link.sent.size
        val payload = "not json".toByteArray()
        for (c in BleProto.chunkFrame(BleProto.encode(payload), 23)) {
            link.proto.onWrite("mac", c)
        }
        assertEquals(before + 1, link.sent.size) // one "not understood"
        assertEquals("not understood", link.replies().last()["error"]!!.jsonPrimitive.content)
    }

    @Test fun pushesGoToLoggedInDevicesOnly() {
        val cmds = Cmds()
        val link = Link(cmds, key)
        link.proto.pushStatus("mac") // nobody there: nothing sent
        assertEquals(0, link.sent.size)
        link.hello()
        link.sent.clear()
        link.proto.pushStatus("mac")
        val p = link.replies().last()
        assertEquals("status", p["push"]!!.jsonPrimitive.content)
        assertEquals("mobile data + Wi-Fi", p["status"]!!.jsonObject["net"]!!.jsonPrimitive.content)
        // the same status again within the heartbeat window: not sent twice
        val n = link.sent.size
        link.proto.pushStatus("mac")
        assertEquals(n, link.sent.size)
    }

    @Test fun deviceGoneForgetsEverything() {
        val link = Link(Cmds(), key)
        link.hello()
        link.proto.deviceGone("mac")
        link.write("id" to 2, "cmd" to "hub.on")
        assertEquals("say hello first", link.replies().last()["error"]!!.jsonPrimitive.content)
    }

    @Test fun bleKeyDiffersFromTheProxyPassword() {
        val pass = ShareProtocol.passKey(key, "mac", exp)
        val (_, bleKey) = ProxyServer.bleLogin(pass, "mac", exp)
        val (_, proxyPass) = ProxyServer.credentials(pass, "mac", exp)
        assertNotEquals(proxyPass, bleKey)
        assertEquals(16, bleKey.length)
    }

    @Test fun vectorsMatchTheGoReference() {
        val f = listOf("../../agent/internal/proxy/testdata/vectors.json", "../agent/internal/proxy/testdata/vectors.json")
            .map { java.io.File(it) }.first { it.exists() }
        val vs = Json.parseToJsonElement(f.readText()).jsonObject["credentials"]!!.jsonArray
        assertTrue(vs.size >= 3)
        for (v in vs) {
            val o = v.jsonObject
            val k = o["share_key"]!!.jsonPrimitive.content.toByteArray()
            val hub = o["hub"]!!.jsonPrimitive.content
            val exp = o["exp"]!!.jsonPrimitive.content.toLong()
            val pk = ShareProtocol.passKey(k, hub, exp)
            val (u, bk) = ProxyServer.bleLogin(pk, hub, exp)
            assertEquals(o["user"]!!.jsonPrimitive.content, u)
            assertEquals(o["ble_key"]!!.jsonPrimitive.content, bk)
        }
    }
}

/** {"k":"v"} from pairs, stringly typed as the test writes messages. */
private fun jsonObj(vararg pairs: Pair<String, Any?>): kotlinx.serialization.json.JsonObject =
    kotlinx.serialization.json.buildJsonObject {
        for ((k, v) in pairs) put(k, kotlinx.serialization.json.JsonPrimitive(v.toString()))
    }
