package dev.tailbond.app.ble

import dev.tailbond.app.ble.BleProtocol as P
import kotlinx.serialization.json.Json
import kotlinx.serialization.json.JsonElement
import kotlinx.serialization.json.JsonObject
import kotlinx.serialization.json.JsonPrimitive
import kotlinx.serialization.json.buildJsonObject
import kotlinx.serialization.json.intOrNull
import kotlinx.serialization.json.jsonObject
import kotlinx.serialization.json.jsonPrimitive
import kotlinx.serialization.json.longOrNull
import kotlinx.serialization.json.put
import java.util.Base64

/** A command the phone refuses: [code] is one of BleProtocol.CODE_*. */
class Refused(val code: String, message: String) : Exception(message)

/** What a Mac can ask this phone for (AppPhoneControl; tests fake it). Each answer's fields go into the reply. */
interface PhoneControl {
    fun hotspotOn(): Map<String, JsonElement>
    fun hotspotOff(): Map<String, JsonElement>
    fun hotspotState(): Map<String, JsonElement>
    fun hubOn(): Map<String, JsonElement>
    fun hubOff(): Map<String, JsonElement>
    fun status(): Map<String, JsonElement>
}

/** Who this phone is to a Mac: its phone hub's id, label and share key id, and its login check. */
class Identity(
    val id: String,
    val label: String,
    val keyId: String,
    val verify: (user: String, cn: String, sn: String, got: String) -> String?,
)

/**
 * One Mac's session: hello, auth, then sealed commands. Not thread-safe:
 * BleServer calls it from one thread.
 */
class BleSession(
    private val identity: () -> Identity?,
    private val control: PhoneControl,
    private val random: (Int) -> ByteArray,
    private val log: (String) -> Unit = {},
) {
    /** What to send back, and whether to drop the Mac afterwards. */
    class Out(val messages: List<ByteArray>, val close: Boolean = false)

    var authed = false
        private set
    /** The Mac's user ("tiger-mbp/1793836800") once logged in. */
    var user: String? = null
        private set
    private var cn: String? = null
    private var sn: String? = null
    private var key: ByteArray? = null
    private var inCtr = 0L
    private var outCtr = 0L

    private fun plain(o: JsonObject) = o.toString().toByteArray()

    private fun sealed(o: JsonObject): ByteArray = P.seal(key!!, P.TO_MAC, outCtr++, plain(o))

    private fun refuse(why: String) = Out(listOf(plain(buildJsonObject {
        put("t", "error"); put("code", P.CODE_AUTH); put("error", why)
    })), close = true)

    fun onMessage(body: ByteArray): Out = if (authed) command(body) else login(body)

    /** A state change to push to a logged-in Mac ({"t":"state",...}); null before login. */
    fun event(fields: Map<String, JsonElement>): ByteArray? =
        if (authed) sealed(JsonObject(mapOf("t" to JsonPrimitive("state")) + fields)) else null

    private fun login(body: ByteArray): Out {
        val o = try {
            Json.parseToJsonElement(body.decodeToString()).jsonObject
        } catch (_: Exception) {
            return refuse("not JSON")
        }
        when (o["t"]?.jsonPrimitive?.content) {
            "hello" -> {
                val c = o["cn"]?.jsonPrimitive?.content
                if (c == null || runCatching { Base64.getDecoder().decode(c).size }.getOrNull() != 16) return refuse("hello without a 16-byte nonce")
                val me = identity() ?: return refuse("This phone isn't paired with a home server, or has no share key yet")
                cn = c
                sn = Base64.getEncoder().encodeToString(random(16))
                return Out(listOf(plain(buildJsonObject {
                    put("t", "hello"); put("v", P.VERSION); put("id", me.id); put("label", me.label); put("kid", me.keyId); put("sn", sn)
                })))
            }
            "auth" -> {
                val c = cn
                val s = sn
                val u = o["user"]?.jsonPrimitive?.content
                val m = o["mac"]?.jsonPrimitive?.content
                if (c == null || s == null) return refuse("auth before hello")
                val me = identity() ?: return refuse("not paired")
                val secret = if (u != null && m != null) me.verify(u, c, s, m) else null
                if (secret == null) {
                    log("ble: login refused for ${u ?: "?"}")
                    return Out(listOf(plain(buildJsonObject {
                        put("t", "auth"); put("ok", false); put("code", P.CODE_AUTH)
                        put("error", "login refused: a Mac paired with the same home server, with a current pass, may connect")
                    })), close = true)
                }
                key = P.sessionKey(secret, c, s, u!!)
                authed = true
                user = u
                log("ble: $u logged in")
                return Out(listOf(plain(buildJsonObject {
                    put("t", "auth"); put("ok", true); put("proof", P.phoneProof(secret, c, s, u))
                })))
            }
            else -> return refuse("log in first")
        }
    }

    private fun command(body: ByteArray): Out {
        // a message that doesn't open (replayed, reordered, changed) ends the session
        val p = P.open(key!!, P.TO_PHONE, inCtr++, body) ?: return Out(emptyList(), close = true)
        val o = try {
            Json.parseToJsonElement(p.decodeToString()).jsonObject
        } catch (_: Exception) {
            return Out(emptyList(), close = true)
        }
        val id = o["id"]?.jsonPrimitive?.longOrNull ?: o["id"]?.jsonPrimitive?.intOrNull?.toLong() ?: 0L
        val cmd = if (o["t"]?.jsonPrimitive?.content == "cmd") o["cmd"]?.jsonPrimitive?.content else null
        val fields = try {
            when (cmd) {
                P.HOTSPOT_ON -> control.hotspotOn()
                P.HOTSPOT_OFF -> control.hotspotOff()
                P.HOTSPOT_STATE -> control.hotspotState()
                P.HUB_ON -> control.hubOn()
                P.HUB_OFF -> control.hubOff()
                P.STATUS -> control.status()
                else -> throw Refused(P.CODE_UNKNOWN, "unknown command ${cmd ?: o["t"]?.jsonPrimitive?.content ?: "?"}")
            }
        } catch (r: Refused) {
            log("ble: $cmd refused: ${r.code} ${r.message}")
            return Out(listOf(sealed(buildJsonObject {
                put("t", "res"); put("id", id); put("ok", false); put("code", r.code); put("error", r.message ?: r.code)
            })))
        }
        log("ble: $cmd for $user")
        return Out(listOf(sealed(JsonObject(mapOf("t" to JsonPrimitive("res"), "id" to JsonPrimitive(id), "ok" to JsonPrimitive(true)) + fields))))
    }
}

/** The GATT server's other side: sends one notification piece to a device (BleControlService; tests fake it). */
interface GattPeer {
    fun send(device: String, piece: ByteArray)
    fun disconnect(device: String)
}

/**
 * The GATT server without Android: Macs that connect, their MTU, the
 * pieces they write, their sessions. BleControlService feeds it from the
 * BluetoothGattServer callbacks, on one thread.
 */
class BleServer(private val peer: GattPeer, private val newSession: () -> BleSession) {
    private class Link(var mtu: Int = 23, val rx: P.Reassembler = P.Reassembler(), val session: BleSession)

    private val links = LinkedHashMap<String, Link>()

    /** Logged-in Macs ("tiger-mbp"), for the notification. */
    val macs: List<String> get() = links.values.mapNotNull { l -> l.session.user?.substringBefore('/') }
    val connected: Int get() = links.size

    fun connected(device: String) {
        links[device] = Link(session = newSession())
    }

    fun mtu(device: String, mtu: Int) {
        links[device]?.mtu = mtu
    }

    fun disconnected(device: String) {
        links.remove(device)
    }

    /** A write to RX. */
    fun write(device: String, piece: ByteArray) {
        val l = links[device] ?: Link(session = newSession()).also { links[device] = it }
        val msgs = try {
            l.rx.add(piece)
        } catch (_: IllegalStateException) {
            drop(device)
            return
        }
        for (m in msgs) {
            val out = l.session.onMessage(m)
            out.messages.forEach { send(device, l, it) }
            if (out.close) {
                drop(device)
                return
            }
        }
    }

    /** A state change for every logged-in Mac. */
    fun push(fields: Map<String, JsonElement>) {
        for ((d, l) in links.entries.toList()) l.session.event(fields)?.let { send(d, l, it) }
    }

    private fun send(device: String, l: Link, msg: ByteArray) {
        for (p in P.chunks(P.frame(msg), l.mtu)) peer.send(device, p)
    }

    private fun drop(device: String) {
        links.remove(device)
        peer.disconnect(device)
    }
}
