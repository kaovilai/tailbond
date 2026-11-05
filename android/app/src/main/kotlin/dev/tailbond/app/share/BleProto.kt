package dev.tailbond.app.share

import java.io.ByteArrayOutputStream
import java.security.MessageDigest
import java.security.SecureRandom
import java.util.UUID
import javax.crypto.Mac
import javax.crypto.spec.SecretKeySpec
import kotlinx.serialization.json.Json
import kotlinx.serialization.json.JsonObject
import kotlinx.serialization.json.JsonPrimitive
import kotlinx.serialization.json.buildJsonObject
import kotlinx.serialization.json.intOrNull
import kotlinx.serialization.json.jsonObject
import kotlinx.serialization.json.jsonPrimitive
import kotlinx.serialization.json.put

/**
 * The BLE control channel of "One click from the Mac" (docs/phone-hub.md):
 * JSON messages, length-prefixed and chunked to the negotiated ATT MTU,
 * over the GATT service [HubService] advertises ([BleGattServer] is the
 * Android glue; this class is plain JVM, tested with a fake transport).
 *
 * The shape is untether's (https://github.com/ImOstrovskiy/untether,
 * BSD-3-Clause, Copyright (c) 2026 Oleksii Ostrovskyi — see
 * TetheringShell.kt for the notice): a challenge the Mac signs, per-device
 * reassembly of chunked writes, characteristics that need an encrypted
 * link. The messages and the login are tailbond's: the same pass Pair &
 * Share's proxy uses, derived with its own label (ProxyServer.bleLogin),
 * so the Mac needs no new key and a leak of one credential is not the
 * other.
 *
 *   challenge (read)   8 bytes key id (ASCII) + 16 random bytes
 *   command  (write)   chunks: [flags][data]; flags bit 0 = START (the
 *                      data begins with a 2-byte big-endian frame length),
 *                      bit 1 = END. A frame is that length + JSON:
 *                      {"id":1,"cmd":"hello","user":"mac/123","proof":"…"}
 *   state  (notify)    frames the same way, replies and pushes:
 *                      {"id":1,"ok":true,…} / {"push":"status",…}
 *
 * Commands: hello, hotspot.on (→ the hotspot's SSID and passphrase),
 * hotspot.off, hotspot.state, hub.on, hub.off, status. Anything before a
 * verified hello, and any unknown command, is answered with an error.
 */
class BleProto(
    /** This phone's share key id, 8 hex characters (the Mac picks its login by it). */
    private val keyId: () -> String,
    /** This phone's own share key (null: not paired yet; nothing works but a refusal). */
    private val ownKey: () -> ByteArray?,
    private val cmds: Commands,
    private val nowSec: () -> Long = { System.currentTimeMillis() / 1000 },
    private val rng: (Int) -> ByteArray = { n -> ByteArray(n).also { SecureRandom().nextBytes(it) } },
    private val log: (String) -> Unit = {},
) {
    /** What the phone does for each command (HubService implements it). */
    interface Commands {
        /** Start the hotspot; the reply carries its network, or the error. */
        fun hotspotOn(reply: (ssid: String?, pass: String?, error: String?) -> Unit)

        fun hotspotOff(reply: (error: String?) -> Unit)

        /** The hotspot now: off, starting, on, stopping, error (+ its SSID). */
        fun hotspotState(): HotspotState

        /** Turn the phone-hub proxy on or off. */
        fun hubSet(on: Boolean, reply: (error: String?) -> Unit)

        /** What a "status" reply and a pushed update say. */
        fun status(): Status
    }

    data class HotspotState(val state: String, val ssid: String)

    data class Status(
        val battery: Int,
        val charging: Boolean,
        /** what carries the phone's tunnel: "mobile data + Wi-Fi" */
        val net: String,
        /** the phone-hub proxy: off, ready, sharing, no_tunnel, no_key, error */
        val hub: String,
        /** bonded paths of the phone's tunnel */
        val paths: Int,
        val hotspot: String,
        val hotspotSsid: String,
        /** "" when the hotspot can be started; otherwise why not */
        val shizuku: String,
    )

    /** Set by the glue: send one frame (length-prefixed JSON) to a device. */
    var out: (device: String, frame: ByteArray) -> Unit = { _, _ -> }

    /** The challenge a device reads: its key id and fresh random bytes. */
    fun challenge(device: String): ByteArray {
        val ch = rng(16)
        challenges[device] = ch
        return keyId().toByteArray(Charsets.US_ASCII) + ch
    }

    /** A write chunk arrived (the Mac chunks frames to the ATT MTU). */
    fun onWrite(device: String, chunk: ByteArray) {
        val payload = reassemblies.getOrPut(device) { Reassembly() }.feed(chunk)
        if (payload != null) handle(device, payload)
    }

    /** A device went: forget its reassembly and login. */
    fun deviceGone(device: String) {
        reassemblies.remove(device)
        challenges.remove(device)
        authed.remove(device)
        pushed.remove(device)
    }

    /**
     * Push a status update to a device that has said hello: at once when
     * it changed, otherwise as a heartbeat ([PUSH_EVERY_MS]).
     */
    fun pushStatus(device: String) {
        if (device !in authed) return
        val frame = statusPush(cmds.status())
        val now = System.currentTimeMillis()
        val last = pushed[device]
        if (last != null && last.second.contentEquals(frame) && now - last.first < PUSH_EVERY_MS) return
        pushed[device] = now to frame
        send(device, frame)
    }

    // the GATT callbacks arrive on binder threads and the pushes on the
    // service's tick: concurrent maps, like untether's
    private val challenges = java.util.concurrent.ConcurrentHashMap<String, ByteArray>()
    private val authed = java.util.concurrent.ConcurrentHashMap.newKeySet<String>()
    private val reassemblies = java.util.concurrent.ConcurrentHashMap<String, Reassembly>()
    private val pushed = java.util.concurrent.ConcurrentHashMap<String, Pair<Long, ByteArray>>()

    private fun handle(device: String, payload: ByteArray) {
        val el = try {
            Json.parseToJsonElement(payload.toString(Charsets.UTF_8)).jsonObject
        } catch (_: Exception) {
            return send(device, err(null, "not understood"))
        }
        val id = el.int("id")
        val cmd = el.str("cmd")
        if (cmd != "hello" && device !in authed) {
            return send(device, err(id, "say hello first"))
        }
        when (cmd) {
            "hello" -> hello(device, id, el)
            "hotspot.on" -> cmds.hotspotOn { ssid, pass, error ->
                send(device, if (error != null) err(id, error) else buildJsonObject {
                    put("id", id ?: 0); put("ok", true); put("ssid", ssid!!); put("pass", pass!!)
                })
            }
            "hotspot.off" -> cmds.hotspotOff { error -> send(device, if (error != null) err(id, error) else ok(id)) }
            "hotspot.state" -> {
                val h = cmds.hotspotState()
                send(device, buildJsonObject { put("id", id ?: 0); put("ok", true); put("state", h.state); put("ssid", h.ssid) })
            }
            "hub.on" -> cmds.hubSet(true) { error -> send(device, if (error != null) err(id, error) else ok(id)) }
            "hub.off" -> cmds.hubSet(false) { error -> send(device, if (error != null) err(id, error) else ok(id)) }
            "status" -> send(device, buildJsonObject {
                put("id", id ?: 0); put("ok", true); put("status", statusObj(cmds.status()))
            })
            else -> send(device, err(id, "unknown command"))
        }
    }

    private fun hello(device: String, id: Int?, el: JsonObject) {
        val user = el.str("user").orEmpty()
        val proof = el.str("proof").orEmpty()
        // the Mac's own nonce: the reply must prove the phone holds the
        // key over it, or a rogue device could answer hello itself and
        // hand the Mac its own Wi-Fi to join
        val macNonce = el.str("nonce").orEmpty().hexBytes()
        val key = ownKey()
        val challenge = challenges[device]
        challenges.remove(device) // single use
        val slash = user.indexOf('/')
        val hub = if (slash > 0) user.substring(0, slash) else ""
        val exp = if (slash > 0) user.substring(slash + 1).toLongOrNull() else null
        var bleKey: String? = null
        val ok = key != null && challenge != null && exp != null && exp > nowSec() && macNonce.size == NONCE && run {
            // the BLE key of the same pass the proxy uses, its own label
            val (_, k) = ProxyServer.bleLogin(ShareProtocol.passKey(key, hub, exp!!), hub, exp)
            bleKey = k
            val mac = Mac.getInstance("HmacSHA256")
            mac.init(SecretKeySpec(k.toByteArray(Charsets.UTF_8), "HmacSHA256"))
            val want = mac.doFinal(challenge).joinToString("") { "%02x".format(it) }
            MessageDigest.isEqual(want.toByteArray(), proof.toByteArray(Charsets.UTF_8))
        }
        if (ok && bleKey != null) {
            authed.add(device)
            // mutual: the phone proves it holds the same key, over the
            // Mac's nonce (the Mac checks it before trusting anything)
            val mac = Mac.getInstance("HmacSHA256")
            mac.init(SecretKeySpec(bleKey!!.toByteArray(Charsets.UTF_8), "HmacSHA256"))
            val mine = mac.doFinal(macNonce).joinToString("") { "%02x".format(it) }
            send(device, buildJsonObject { put("id", id ?: 0); put("ok", true); put("key_id", keyId()); put("proof", mine) })
            log("ble: $device said hello as $hub")
        } else {
            send(device, err(id, "wrong password (paired with another home server?)"))
            log("ble: $device failed hello as $hub")
        }
    }

    private fun statusObj(s: Status): JsonObject = buildJsonObject {
        put("battery", s.battery)
        put("charging", s.charging)
        put("net", s.net)
        put("hub", s.hub)
        put("paths", s.paths)
        put("hotspot", s.hotspot)
        put("ssid", s.hotspotSsid)
        put("shizuku", s.shizuku)
    }

    private fun statusPush(s: Status): ByteArray = encode(buildJsonObject {
        put("push", "status")
        put("status", statusObj(s))
    }.toString().toByteArray(Charsets.UTF_8))

    private fun ok(id: Int?) = buildJsonObject { put("id", id ?: 0); put("ok", true) }

    private fun err(id: Int?, error: String) = buildJsonObject {
        if (id != null) put("id", id)
        put("ok", false)
        put("error", error)
    }

    private fun send(device: String, o: JsonObject) = send(device, encode(o.toString().toByteArray(Charsets.UTF_8)))

    private fun send(device: String, frame: ByteArray) = out(device, frame)

    private fun JsonObject.int(k: String): Int? = (this[k] as? JsonPrimitive)?.intOrNull

    private fun JsonObject.str(k: String): String? = (this[k] as? JsonPrimitive)?.content

    /** One device's write reassembly (untether's, kept as it is). */
    private class Reassembly {
        private var buf = ByteArrayOutputStream()
        private var want = -1

        /** Feeds a chunk; the frame's payload once its END arrives (and its length matches), else null. */
        fun feed(chunk: ByteArray): ByteArray? {
            if (chunk.isEmpty()) return null
            val flags = chunk[0].toInt() and 0xff
            val data = chunk.copyOfRange(1, chunk.size)
            if (flags and START != 0) {
                buf.reset()
                want = -1
                if (data.size < 2) return null
                want = ((data[0].toInt() and 0xff) shl 8) or (data[1].toInt() and 0xff)
                if (want > MAX_FRAME) return null
                buf.write(data, 2, data.size - 2)
            } else if (want < 0) {
                return null // a middle chunk with no start
            } else {
                buf.write(data)
            }
            if (buf.size() > want) {
                // more data than the frame declared: drop it all rather
                // than grow without bound waiting for an END that never comes
                buf.reset()
                want = -1
                return null
            }
            if (flags and END != 0) {
                val payload = buf.toByteArray()
                buf.reset()
                val wanted = this.want
                this.want = -1
                return if (payload.size == wanted) payload else null
            }
            return null
        }
    }

    companion object {
        const val MAX_FRAME = 4096
        private const val START = 0x01
        private const val END = 0x02

        /** A whole frame in one chunk: START | END. */
        const val CHUNK_WHOLE: Byte = (START or END).toByte()

        /** the challenge's and the hello nonce's length, bytes */
        const val NONCE = 16

        /** a heartbeat to every connected Mac, even when nothing changed */
        private const val PUSH_EVERY_MS = 10_000L

        /** [2-byte big-endian length][payload]: one frame on either side. */
        fun encode(payload: ByteArray): ByteArray =
            byteArrayOf((payload.size shr 8).toByte(), payload.size.toByte()) + payload

        /**
         * Splits a frame into [flags][data] chunks that fit the ATT MTU
         * (which must be at least 6: ATT's 3 bytes, the flags byte and the
         * 2-byte length the first chunk carries).
         */
        fun chunkFrame(frame: ByteArray, mtu: Int): List<ByteArray> {
            val room = (mtu - 4).coerceAtLeast(2) // ATT uses 3, the flags byte 1
            val out = ArrayList<ByteArray>()
            var at = 0
            while (at < frame.size || out.isEmpty()) {
                val end = minOf(at + room, frame.size)
                var flags = 0
                if (at == 0) flags = flags or START
                if (end == frame.size) flags = flags or END
                out.add(byteArrayOf(flags.toByte()) + frame.copyOfRange(at, end))
                at = end
            }
            return out
        }

        /** The proof the Mac sends: HMAC-SHA256 over the challenge with the BLE key. */
        fun proof(bleKey: String, challenge: ByteArray): String {
            val mac = Mac.getInstance("HmacSHA256")
            mac.init(SecretKeySpec(bleKey.toByteArray(Charsets.UTF_8), "HmacSHA256"))
            return mac.doFinal(challenge).joinToString("") { "%02x".format(it) }
        }
    }
}

/** Hex, or an empty array when it isn't. */
internal fun String.hexBytes(): ByteArray {
    if (length % 2 != 0) return ByteArray(0)
    val out = ByteArray(length / 2)
    for (i in out.indices) {
        val v = substring(i * 2, i * 2 + 2).toIntOrNull(16) ?: return ByteArray(0)
        out[i] = v.toByte()
    }
    return out
}

/** BLE UUIDs of the tailbond control service (the phone is the peripheral). */
object Ble {
    val SERVICE: UUID = UUID.fromString("54424c31-5a1c-4b7e-9f2d-6e8b1d0a3c5f")
    val CHALLENGE: UUID = UUID.fromString("54424c31-5a1c-4b7e-9f2d-6e8b1d0a3c60")
    val COMMAND: UUID = UUID.fromString("54424c31-5a1c-4b7e-9f2d-6e8b1d0a3c61")
    val STATE: UUID = UUID.fromString("54424c31-5a1c-4b7e-9f2d-6e8b1d0a3c62")
    val CCCD: UUID = UUID.fromString("00002902-0000-1000-8000-00805f9b34fb")
}
