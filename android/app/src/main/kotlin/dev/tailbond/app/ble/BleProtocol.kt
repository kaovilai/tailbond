package dev.tailbond.app.ble

import dev.tailbond.app.share.ProxyServer
import dev.tailbond.app.share.ShareProtocol
import java.io.ByteArrayOutputStream
import java.nio.ByteBuffer
import java.security.MessageDigest
import java.util.UUID
import javax.crypto.Cipher
import javax.crypto.Mac
import javax.crypto.spec.GCMParameterSpec
import javax.crypto.spec.SecretKeySpec

/**
 * The Bluetooth LE control channel between a Mac and this phone
 * (docs/phone-hub.md, "One click from the Mac"). The Go reference is
 * agent/internal/bleproto; its testdata/vectors.json pins these bytes
 * (BleProtocolTest).
 *
 * - GATT: [SERVICE]; the Mac writes to [RX], the phone notifies on [TX].
 * - Framing: a 2-byte big-endian length, then the message (at most
 *   [MAX_MESSAGE] bytes), cut into ATT payloads of MTU - 3 bytes.
 * - Login: the proxy's pass under the label "tbp1-ble" ([credentials]);
 *   hello both ways (nonces), auth from the Mac ([authMac]), the phone's
 *   [phoneProof] back.
 * - After the login every message is sealed with AES-256-GCM ([seal]).
 */
object BleProtocol {
    val SERVICE: UUID = UUID.fromString("5f1d0001-7462-4e64-8f6c-7461696c626f")
    val RX: UUID = UUID.fromString("5f1d0002-7462-4e64-8f6c-7461696c626f")
    val TX: UUID = UUID.fromString("5f1d0003-7462-4e64-8f6c-7461696c626f")
    val CCCD: UUID = UUID.fromString("00002902-0000-1000-8000-00805f9b34fb")

    const val VERSION = 1
    const val MAX_MESSAGE = 4096
    const val TO_PHONE: Byte = 1
    const val TO_MAC: Byte = 2

    const val HOTSPOT_ON = "hotspot.on"
    const val HOTSPOT_OFF = "hotspot.off"
    const val HOTSPOT_STATE = "hotspot.state"
    const val HUB_ON = "hub.on"
    const val HUB_OFF = "hub.off"
    const val STATUS = "status"

    const val CODE_SHIZUKU = "shizuku"
    const val CODE_UNSUPPORTED = "unsupported"
    const val CODE_HOTSPOT = "hotspot"
    const val CODE_HUB = "hub"
    const val CODE_UNKNOWN = "unknown"
    const val CODE_AUTH = "auth"

    private fun hmac(key: ByteArray, s: String): ByteArray =
        Mac.getInstance("HmacSHA256").run {
            init(SecretKeySpec(key, "HmacSHA256"))
            doFinal(s.toByteArray(Charsets.UTF_8))
        }

    private fun hex(b: ByteArray) = b.joinToString("") { "%02x".format(it) }

    /** A Mac's Bluetooth login: user "<hub>/<exp>", secret 16 lower-case base32 characters. */
    fun credentials(pass: ByteArray, hub: String, exp: Long): Pair<String, String> =
        "$hub/$exp" to ProxyServer.base32(hmac(pass, "tbp1-ble")).substring(0, 16).lowercase()

    private fun transcript(label: String, cn: String, sn: String, user: String) = "$label|$cn|$sn|$user"

    fun authMac(secret: String, cn: String, sn: String, user: String): String =
        hex(hmac(secret.toByteArray(), transcript("tbp1-ble-auth", cn, sn, user)))

    fun phoneProof(secret: String, cn: String, sn: String, user: String): String =
        hex(hmac(secret.toByteArray(), transcript("tbp1-ble-phone", cn, sn, user)))

    fun sessionKey(secret: String, cn: String, sn: String, user: String): ByteArray =
        hmac(secret.toByteArray(), transcript("tbp1-ble-session", cn, sn, user))

    /**
     * The phone's check of a login, as [ProxyServer.verifier]: the pass from
     * its own share key for the hub and expiry in user, not expired, the hub
     * allowed. Returns the secret, or null.
     */
    fun verifier(
        ownKey: ByteArray,
        nowSec: () -> Long = { System.currentTimeMillis() / 1000 },
        allowed: ((String) -> Boolean)? = null,
    ): (user: String, cn: String, sn: String, got: String) -> String? = check@{ user, cn, sn, got ->
        val i = user.indexOf('/')
        if (i <= 0) return@check null
        val hub = user.substring(0, i)
        val exp = user.substring(i + 1).toLongOrNull() ?: return@check null
        if (exp < 0 || exp < nowSec()) return@check null
        if (allowed != null && !allowed(hub)) return@check null
        val secret = credentials(ShareProtocol.passKey(ownKey, hub, exp), hub, exp).second
        if (MessageDigest.isEqual(got.toByteArray(), authMac(secret, cn, sn, user).toByteArray())) secret else null
    }

    private fun nonce(dir: Byte, ctr: Long): ByteArray =
        ByteBuffer.allocate(12).put(dir).put(byteArrayOf(0, 0, 0)).putLong(ctr).array()

    private fun gcm(mode: Int, key: ByteArray, dir: Byte, ctr: Long): Cipher =
        Cipher.getInstance("AES/GCM/NoPadding").apply {
            init(mode, SecretKeySpec(key, "AES"), GCMParameterSpec(128, nonce(dir, ctr)))
        }

    fun seal(key: ByteArray, dir: Byte, ctr: Long, plain: ByteArray): ByteArray =
        gcm(Cipher.ENCRYPT_MODE, key, dir, ctr).doFinal(plain)

    /** null: it doesn't open (another key, counter or direction, or changed). */
    fun open(key: ByteArray, dir: Byte, ctr: Long, sealed: ByteArray): ByteArray? =
        try {
            gcm(Cipher.DECRYPT_MODE, key, dir, ctr).doFinal(sealed)
        } catch (_: Exception) {
            null
        }

    fun frame(msg: ByteArray): ByteArray {
        require(msg.size <= MAX_MESSAGE) { "message too long" }
        return byteArrayOf((msg.size shr 8).toByte(), msg.size.toByte()) + msg
    }

    /** ATT payloads for [mtu] (MTU - 3 bytes, at least 20). */
    fun chunks(framed: ByteArray, mtu: Int): List<ByteArray> {
        val n = maxOf(mtu - 3, 20)
        return (framed.indices step n).map { framed.copyOfRange(it, minOf(it + n, framed.size)) }
    }

    /** Received pieces back into messages. */
    class Reassembler {
        private var buf = ByteArrayOutputStream()

        /** The messages [piece] completes; a length over [MAX_MESSAGE] throws (and drops the buffer). */
        fun add(piece: ByteArray): List<ByteArray> {
            buf.write(piece)
            val out = mutableListOf<ByteArray>()
            var b = buf.toByteArray()
            while (b.size >= 2) {
                val n = ((b[0].toInt() and 0xff) shl 8) or (b[1].toInt() and 0xff)
                if (n > MAX_MESSAGE) {
                    buf = ByteArrayOutputStream()
                    throw IllegalStateException("message too long")
                }
                if (b.size < 2 + n) break
                out += b.copyOfRange(2, 2 + n)
                b = b.copyOfRange(2 + n, b.size)
            }
            buf = ByteArrayOutputStream().apply { write(b) }
            return out
        }
    }
}
