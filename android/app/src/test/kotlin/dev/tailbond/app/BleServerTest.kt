package dev.tailbond.app

import android.content.Context
import android.os.BatteryManager
import androidx.test.core.app.ApplicationProvider
import dev.tailbond.app.ble.AppPhoneControl
import dev.tailbond.app.ble.BleProtocol
import dev.tailbond.app.ble.BleServer
import dev.tailbond.app.ble.BleSession
import dev.tailbond.app.ble.GattPeer
import dev.tailbond.app.ble.HotspotControl
import dev.tailbond.app.ble.HubControl
import dev.tailbond.app.ble.Identity
import dev.tailbond.app.ble.ShizukuStatus
import dev.tailbond.app.model.PathKind
import dev.tailbond.app.service.ConnState
import dev.tailbond.app.service.Live
import dev.tailbond.app.service.PathGate
import dev.tailbond.app.service.PathSample
import dev.tailbond.app.share.HubLive
import dev.tailbond.app.share.HubStatus
import dev.tailbond.app.share.ShareProtocol
import kotlinx.serialization.json.Json
import kotlinx.serialization.json.JsonObject
import kotlinx.serialization.json.boolean
import kotlinx.serialization.json.int
import kotlinx.serialization.json.jsonObject
import kotlinx.serialization.json.jsonPrimitive
import org.junit.Assert.assertEquals
import org.junit.Assert.assertFalse
import org.junit.Assert.assertTrue
import org.junit.Before
import org.junit.Test
import org.junit.runner.RunWith
import org.robolectric.RobolectricTestRunner
import org.robolectric.Shadows.shadowOf
import org.robolectric.annotation.Config
import java.util.Base64

/**
 * The phone's side of "One click from your Mac", without Bluetooth: a fake
 * GATT peer (it reassembles what the server notifies), a fake Shizuku
 * hotspot and a fake "Share bonded connection", AppPhoneControl on a
 * Robolectric context, and a Mac played by the test with the protocol.
 */
@RunWith(RobolectricTestRunner::class)
@Config(sdk = [35])
class BleServerTest {
    private val phoneKey = "home-share-key".toByteArray()
    private val exp = System.currentTimeMillis() / 1000 + 86_400
    private val dev = "AA:BB:CC:DD:EE:FF"

    /** Shizuku's hotspot: the AP comes up after a few polls, or fails. */
    class FakeHotspot : HotspotControl {
        var shizuku = ShizukuStatus.OK
        var ap = HotspotControl.AP_DISABLED
        var failWith = 0
        var starts = 0
        var stops = 0
        private var polls = 0
        override fun shizuku() = shizuku
        override fun start(): String? {
            starts++
            ap = HotspotControl.AP_ENABLING
            polls = 0
            return null
        }
        override fun stop(): String? {
            stops++
            ap = HotspotControl.AP_DISABLED
            return null
        }
        override fun apState(): Int {
            if (ap == HotspotControl.AP_ENABLING && ++polls >= 3) ap = if (failWith != 0) HotspotControl.AP_FAILED else HotspotControl.AP_ENABLED
            return ap
        }
        override fun lastError() = if (ap == HotspotControl.AP_FAILED) failWith else 0
        override fun network() = "Pixel 9" to "correct horse"
    }

    class FakeHub : HubControl {
        var live = HubLive()
        var startsAs = HubStatus.READY
        var refuse = false
        override fun start() {
            if (refuse) throw IllegalStateException("ForegroundServiceStartNotAllowedException")
            live = live.copy(status = startsAs)
        }
        override fun stop() {
            live = HubLive()
        }
        override fun live() = live
    }

    class FakePeer : GattPeer {
        val pieces = mutableListOf<ByteArray>()
        val dropped = mutableListOf<String>()
        override fun send(device: String, piece: ByteArray) {
            pieces += piece
        }
        override fun disconnect(device: String) {
            dropped += device
        }
    }

    private lateinit var ctx: Context
    private lateinit var hotspot: FakeHotspot
    private lateinit var hub: FakeHub
    private lateinit var peer: FakePeer
    private lateinit var server: BleServer
    private var paired = true
    private var sdk = 35

    @Before fun setUp() {
        ctx = ApplicationProvider.getApplicationContext()
        shadowOf(ctx.getSystemService(BatteryManager::class.java)).setIntProperty(BatteryManager.BATTERY_PROPERTY_CAPACITY, 81)
        hotspot = FakeHotspot()
        hub = FakeHub()
        peer = FakePeer()
        val live = Live(
            state = ConnState.CONNECTED,
            paths = listOf(
                PathSample("cellular-1", PathKind.CELLULAR, PathGate.STATUS_ACTIVE, 40, 0, 0, 0, 0),
                PathSample("wifi-2", PathKind.WIFI, PathGate.STATUS_ACTIVE, 20, 0, 0, 0, 0),
            ),
        )
        val control = AppPhoneControl(ctx, hotspot, hub, live = { live }, sdk = 35, waitMs = 2_000, sleep = {})
        val sdkControl = { if (sdk == 35) control else AppPhoneControl(ctx, hotspot, hub, live = { live }, sdk = sdk, sleep = {}) }
        server = BleServer(peer) {
            BleSession(
                identity = {
                    if (!paired) null else Identity("an-1234-hub", "pixel-9", ShareProtocol.keyId(phoneKey), BleProtocol.verifier(phoneKey))
                },
                control = sdkControl(),
                random = { n -> ByteArray(n) { 7 } },
            )
        }
        server.connected(dev)
        server.mtu(dev, 23)
    }

    /** The Mac's half, as the menu bar app does it. */
    inner class Mac(val user: String, val secret: String) {
        var mtu = 23
        private val rx = BleProtocol.Reassembler()
        private var seen = 0
        private val cn = Base64.getEncoder().encodeToString(ByteArray(16) { 1 })
        private var key: ByteArray? = null
        private var outCtr = 0L
        private var inCtr = 0L
        private var id = 0

        fun write(body: ByteArray) {
            for (p in BleProtocol.chunks(BleProtocol.frame(body), mtu)) server.write(dev, p)
        }

        /** Everything the phone sent since the last call. */
        fun read(): List<JsonObject> {
            val new = peer.pieces.drop(seen)
            seen = peer.pieces.size
            assertTrue("a piece over MTU $mtu", new.all { it.size <= mtu - 3 })
            return new.flatMap { rx.add(it) }.map { m ->
                val plain = key?.let { k -> BleProtocol.open(k, BleProtocol.TO_MAC, inCtr++, m) ?: error("doesn't open") } ?: m
                Json.parseToJsonElement(plain.decodeToString()).jsonObject
            }
        }

        fun login(mac: String? = null): JsonObject {
            write("""{"t":"hello","v":1,"cn":"$cn"}""".toByteArray())
            val hello = read().single()
            assertEquals("hello", hello.s("t"))
            assertEquals("an-1234-hub", hello.s("id"))
            assertEquals(ShareProtocol.keyId(phoneKey), hello.s("kid"))
            val sn = hello.s("sn")
            write("""{"t":"auth","user":"$user","mac":"${mac ?: BleProtocol.authMac(secret, cn, sn, user)}"}""".toByteArray())
            val auth = read().single()
            if (auth["ok"]!!.jsonPrimitive.boolean) {
                assertEquals(BleProtocol.phoneProof(secret, cn, sn, user), auth.s("proof"))
                key = BleProtocol.sessionKey(secret, cn, sn, user)
            }
            return auth
        }

        fun cmd(c: String): JsonObject {
            val i = ++id
            write(BleProtocol.seal(key!!, BleProtocol.TO_PHONE, outCtr++, """{"t":"cmd","id":$i,"cmd":"$c"}""".toByteArray()))
            val r = read().single()
            assertEquals("res", r.s("t"))
            assertEquals(i, r["id"]!!.jsonPrimitive.int)
            return r
        }

        /** Any JSON, sealed, as a logged-in Mac. */
        fun sealedRaw(json: String) = write(BleProtocol.seal(key!!, BleProtocol.TO_PHONE, outCtr++, json.toByteArray()))

        /** A sealed message sent again (a replay). */
        fun replayLast(c: String) {
            write(BleProtocol.seal(key!!, BleProtocol.TO_PHONE, outCtr - 1, """{"t":"cmd","id":99,"cmd":"$c"}""".toByteArray()))
        }
    }

    private fun JsonObject.s(k: String) = this[k]!!.jsonPrimitive.content
    private fun JsonObject.ok() = this["ok"]!!.jsonPrimitive.boolean

    private fun mac(): Mac {
        val (u, s) = BleProtocol.credentials(ShareProtocol.passKey(phoneKey, "tiger-mbp", exp), "tiger-mbp", exp)
        return Mac(u, s)
    }

    @Test fun oneClick() {
        val m = mac()
        assertTrue(m.login().ok())
        assertEquals(listOf("tiger-mbp"), server.macs)
        val on = m.cmd(BleProtocol.HOTSPOT_ON)
        assertTrue(on.ok())
        assertEquals("Pixel 9", on.s("ssid"))
        assertEquals("correct horse", on.s("pass"))
        assertEquals(1, hotspot.starts)
        // already on: not started again
        assertTrue(m.cmd(BleProtocol.HOTSPOT_ON).ok())
        assertEquals(1, hotspot.starts)
        assertEquals("on", m.cmd(BleProtocol.HOTSPOT_STATE).s("state"))
        val h = m.cmd(BleProtocol.HUB_ON)
        assertTrue(h.ok())
        assertEquals("ready", h.s("hub"))
        val st = m.cmd(BleProtocol.STATUS)
        assertEquals(81, st["battery"]!!.jsonPrimitive.int)
        assertEquals("Mobile data + Wi-Fi", st.s("network"))
        assertEquals(2, st["paths"]!!.jsonPrimitive.int)
        assertEquals("ready", st.s("hub"))
        assertEquals("on", st.s("hotspot"))
        // a state push reaches the logged-in Mac, sealed
        server.push(mapOf("hub" to kotlinx.serialization.json.JsonPrimitive("sharing")))
        assertEquals("sharing", m.read().single().s("hub"))
        assertEquals("off", m.cmd(BleProtocol.HUB_OFF).s("hub"))
        assertEquals(HubStatus.OFF, hub.live.status)
        assertTrue(m.cmd(BleProtocol.HOTSPOT_OFF).ok())
        assertEquals(1, hotspot.stops)
        assertTrue(peer.dropped.isEmpty())
    }

    @Test fun unknownCommandIsRefusedButTheSessionStays() {
        val m = mac()
        m.login()
        val r = m.cmd("selfdestruct")
        assertFalse(r.ok())
        assertEquals(BleProtocol.CODE_UNKNOWN, r.s("code"))
        assertTrue(m.cmd(BleProtocol.STATUS).ok())
    }

    @Test fun nothingBeforeLogin() {
        val m = mac()
        m.write("""{"t":"cmd","id":1,"cmd":"hotspot.on"}""".toByteArray())
        val r = m.read().single()
        assertEquals("error", r.s("t"))
        assertEquals(BleProtocol.CODE_AUTH, r.s("code"))
        assertEquals(listOf(dev), peer.dropped)
        assertEquals(0, hotspot.starts)
    }

    @Test fun wrongLoginIsRefused() {
        val r = mac().login(mac = "00".repeat(32))
        assertFalse(r.ok())
        assertEquals(BleProtocol.CODE_AUTH, r.s("code"))
        assertEquals(listOf(dev), peer.dropped)
        // another home server's Mac: same, the pass doesn't match
        setUp()
        val (u, s) = BleProtocol.credentials(ShareProtocol.passKey("another-key".toByteArray(), "tiger-mbp", exp), "tiger-mbp", exp)
        assertFalse(Mac(u, s).login().ok())
    }

    @Test fun notPaired() {
        paired = false
        setUp()
        val m = mac()
        m.write("""{"t":"hello","v":1,"cn":"${Base64.getEncoder().encodeToString(ByteArray(16))}"}""".toByteArray())
        val r = m.read().single()
        assertEquals("error", r.s("t"))
        assertTrue(r.s("error").contains("isn't paired"))
    }

    /** Wrong JSON types from a peer end its session, never the app. */
    @Test fun malformedMessagesDropThePeer() {
        val m = mac()
        m.write("""{"t":{}}""".toByteArray())
        assertEquals(BleProtocol.CODE_AUTH, m.read().single().s("code"))
        assertEquals(listOf(dev), peer.dropped)
        setUp()
        val m2 = mac()
        m2.write("""{"t":"hello","cn":[1,2]}""".toByteArray())
        assertEquals("error", m2.read().single().s("t"))
        setUp()
        val m3 = mac()
        assertTrue(m3.login().ok())
        m3.sealedRaw("""{"t":"cmd","id":"x","cmd":{"a":1}}""")
        val r = m3.read().single()
        assertFalse(r.ok())
        assertEquals(BleProtocol.CODE_UNKNOWN, r.s("code"))
        m3.sealedRaw("[1,2,3]")
        assertEquals(listOf(dev), peer.dropped)
        assertEquals(0, hotspot.starts)
    }

    /** Something the phone didn't expect while serving a command: an answer, not a crash. */
    @Test fun unexpectedFailureIsAnAnswer() {
        val m = mac()
        m.login()
        hub.live = HubLive()
        hub.refuse = false
        hotspot.shizuku = ShizukuStatus.OK
        val boom = object : HotspotControl by hotspot {
            override fun apState(): Int = throw IllegalStateException("binder died")
        }
        // a session whose hotspot control throws
        server = BleServer(peer) {
            BleSession({ Identity("an-1234-hub", "pixel-9", ShareProtocol.keyId(phoneKey), BleProtocol.verifier(phoneKey)) },
                AppPhoneControl(ctx, boom, hub, sdk = 35, sleep = {}), { n -> ByteArray(n) { 7 } })
        }
        server.connected(dev)
        peer.pieces.clear()
        val m2 = mac()
        assertTrue(m2.login().ok())
        val r = m2.cmd(BleProtocol.HOTSPOT_ON)
        assertFalse(r.ok())
        assertTrue(r.s("error"), r.s("error").contains("IllegalStateException"))
        assertTrue(m2.cmd(BleProtocol.HUB_ON).ok())
    }

    @Test fun replayEndsTheSession() {
        val m = mac()
        m.login()
        m.cmd(BleProtocol.STATUS)
        m.replayLast(BleProtocol.HOTSPOT_ON)
        assertEquals(listOf(dev), peer.dropped)
        assertEquals(0, hotspot.starts)
    }

    @Test fun shizukuNotRunning() {
        hotspot.shizuku = ShizukuStatus.NOT_RUNNING
        val m = mac()
        m.login()
        val r = m.cmd(BleProtocol.HOTSPOT_ON)
        assertFalse(r.ok())
        assertEquals(BleProtocol.CODE_SHIZUKU, r.s("code"))
        assertTrue(r.s("error").contains("Shizuku isn't running"))
        hotspot.shizuku = ShizukuStatus.NO_PERMISSION
        assertTrue(m.cmd(BleProtocol.HOTSPOT_ON).s("error").contains("may not use Shizuku"))
        assertEquals(0, hotspot.starts)
    }

    @Test fun hotspotRefused() {
        hotspot.failWith = 5
        val m = mac()
        m.login()
        val r = m.cmd(BleProtocol.HOTSPOT_ON)
        assertFalse(r.ok())
        assertEquals(BleProtocol.CODE_HOTSPOT, r.s("code"))
        assertTrue(r.s("error"), r.s("error").contains("error 5"))
    }

    @Test fun oldAndroid() {
        sdk = 29
        setUp()
        val m = mac()
        m.login()
        assertEquals(BleProtocol.CODE_UNSUPPORTED, m.cmd(BleProtocol.HOTSPOT_ON).s("code"))
    }

    @Test fun hubRefused() {
        hub.startsAs = HubStatus.NO_KEY
        val m = mac()
        m.login()
        val r = m.cmd(BleProtocol.HUB_ON)
        assertFalse(r.ok())
        assertEquals(BleProtocol.CODE_HUB, r.s("code"))
        hub.live = HubLive()
        hub.refuse = true
        assertTrue(m.cmd(BleProtocol.HUB_ON).s("error").contains("open tailbond on the phone"))
    }

    @Test fun bigMtuAndDisconnect() {
        server.mtu(dev, 185)
        val m = mac().apply { mtu = 185 }
        assertTrue(m.login().ok())
        assertTrue(peer.pieces.any { it.size > 20 })
        server.disconnected(dev)
        assertEquals(0, server.connected)
        assertTrue(server.macs.isEmpty())
    }
}
