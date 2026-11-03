package dev.tailbond.app.share

import java.nio.ByteBuffer
import javax.crypto.Mac
import javax.crypto.spec.SecretKeySpec

/**
 * Pair & Share wire format, the sharer's half (docs/architecture.md "Pair &
 * Share"; the Go reference is agent/internal/share/proto.go, and
 * agent/internal/share/testdata/vectors.json pins both).
 *
 *     HELLO      01 ver hub_nonce[16] idlen id
 *     CHALLENGE  02 ver hub_nonce[16] sharer_nonce[16] idlen id lablen label
 *     AUTH       03 ver hub_nonce[16] sharer_nonce[16] mac[32]
 *     WELCOME    04 ver hub_nonce[16] session[8] cap_left[8] mac[32]
 *     REFUSE     05 ver hub_nonce[16] reason mac[32]
 *     DATA       10 session[8] payload
 *     KEEPALIVE  11 session[8] seq[4]
 *     KA_ACK     12 session[8] seq[4] up[8] down[8] cap_left[8] battery flags
 *     BYE        13 session[8] reason
 */
object ShareProtocol {
    const val VERSION = 1
    const val DEFAULT_PORT = 47077
    const val SERVICE_TYPE = "_tailbond-share._udp"

    const val HELLO: Byte = 0x01
    const val CHALLENGE: Byte = 0x02
    const val AUTH: Byte = 0x03
    const val WELCOME: Byte = 0x04
    const val REFUSE: Byte = 0x05
    const val DATA: Byte = 0x10
    const val KEEPALIVE: Byte = 0x11
    const val KA_ACK: Byte = 0x12
    const val BYE: Byte = 0x13

    const val NONCE = 16
    const val SESSION = 8
    const val MAC = 32
    const val DATA_HEADER = 1 + SESSION
    const val MAX_ID = 64
    /** cap_left without a cap. */
    const val NO_CAP = -1L // all ones, as uint64

    // reasons
    const val R_NONE = 0
    const val R_NOT_ALLOWED = 1
    const val R_CAP = 2
    const val R_BATTERY = 3
    const val R_PAUSED = 4
    const val R_NO_CELLULAR = 5
    const val R_REPLACED = 6
    const val R_IDLE = 7

    const val FLAG_CHARGING = 1

    class Hello(val hubNonce: ByteArray, val hubId: String)
    class Auth(val hubNonce: ByteArray, val sharerNonce: ByteArray, val mac: ByteArray)

    private fun idBytes(s: String): ByteArray {
        val b = s.toByteArray(Charsets.UTF_8)
        return if (b.size > MAX_ID) b.copyOf(MAX_ID) else b
    }

    fun parseHello(b: ByteArray, n: Int = b.size): Hello? {
        if (n < 2 + NONCE + 1 || b[0] != HELLO || b[1].toInt() != VERSION) return null
        val len = b[2 + NONCE].toInt() and 0xff
        if (n < 3 + NONCE + len) return null
        return Hello(b.copyOfRange(2, 2 + NONCE), String(b, 3 + NONCE, len, Charsets.UTF_8))
    }

    fun parseAuth(b: ByteArray, n: Int = b.size): Auth? {
        if (n < 2 + 2 * NONCE + MAC || b[0] != AUTH || b[1].toInt() != VERSION) return null
        return Auth(b.copyOfRange(2, 2 + NONCE), b.copyOfRange(2 + NONCE, 2 + 2 * NONCE), b.copyOfRange(2 + 2 * NONCE, 2 + 2 * NONCE + MAC))
    }

    fun hello(hubNonce: ByteArray, hubId: String): ByteArray {
        val id = idBytes(hubId)
        return byteArrayOf(HELLO, VERSION.toByte()) + hubNonce + byteArrayOf(id.size.toByte()) + id
    }

    fun challenge(hubNonce: ByteArray, sharerNonce: ByteArray, sharerId: String, label: String): ByteArray {
        val id = idBytes(sharerId)
        val lab = idBytes(label)
        return byteArrayOf(CHALLENGE, VERSION.toByte()) + hubNonce + sharerNonce +
            byteArrayOf(id.size.toByte()) + id + byteArrayOf(lab.size.toByte()) + lab
    }

    fun auth(hubNonce: ByteArray, sharerNonce: ByteArray, mac: ByteArray): ByteArray =
        byteArrayOf(AUTH, VERSION.toByte()) + hubNonce + sharerNonce + mac

    fun welcome(hubNonce: ByteArray, session: ByteArray, capLeft: Long, mac: ByteArray): ByteArray =
        byteArrayOf(WELCOME, VERSION.toByte()) + hubNonce + session + long(capLeft) + mac

    fun refuse(hubNonce: ByteArray, reason: Int, mac: ByteArray): ByteArray =
        byteArrayOf(REFUSE, VERSION.toByte()) + hubNonce + byteArrayOf(reason.toByte()) + mac

    /** DATA frame header + payload into [out] (size DATA_HEADER + len); returns the frame length. */
    fun data(out: ByteArray, session: ByteArray, payload: ByteArray, off: Int, len: Int): Int {
        out[0] = DATA
        System.arraycopy(session, 0, out, 1, SESSION)
        System.arraycopy(payload, off, out, DATA_HEADER, len)
        return DATA_HEADER + len
    }

    fun keepalive(session: ByteArray, seq: Int): ByteArray =
        byteArrayOf(KEEPALIVE) + session + ByteBuffer.allocate(4).putInt(seq).array()

    fun kaAck(session: ByteArray, seq: Int, up: Long, down: Long, capLeft: Long, battery: Int, flags: Int): ByteArray =
        byteArrayOf(KA_ACK) + session + ByteBuffer.allocate(4).putInt(seq).array() +
            long(up) + long(down) + long(capLeft) + byteArrayOf(battery.toByte(), flags.toByte())

    fun bye(session: ByteArray, reason: Int): ByteArray = byteArrayOf(BYE) + session + byteArrayOf(reason.toByte())

    /** Session id of a DATA/KEEPALIVE/BYE frame, null when too short. */
    fun sessionOf(b: ByteArray, n: Int): ByteArray? = if (n < 1 + SESSION) null else b.copyOfRange(1, 1 + SESSION)

    fun seqOf(b: ByteArray): Int = ByteBuffer.wrap(b, 1 + SESSION, 4).int

    private fun long(v: Long) = ByteBuffer.allocate(8).putLong(v).array()

    private fun mac(key: ByteArray, label: String, vararg parts: ByteArray): ByteArray {
        val m = Mac.getInstance("HmacSHA256")
        m.init(SecretKeySpec(key, "HmacSHA256"))
        m.update(label.toByteArray(Charsets.UTF_8))
        parts.forEach { m.update(it) }
        return m.doFinal()
    }

    fun authMac(key: ByteArray, hubNonce: ByteArray, sharerNonce: ByteArray, hubId: String): ByteArray =
        mac(key, "tbs1-auth", hubNonce, sharerNonce, hubId.toByteArray(Charsets.UTF_8))

    fun welcomeMac(key: ByteArray, hubNonce: ByteArray, sharerNonce: ByteArray, session: ByteArray, capLeft: Long): ByteArray =
        mac(key, "tbs1-welcome", hubNonce, sharerNonce, session, long(capLeft))

    fun refuseMac(key: ByteArray, hubNonce: ByteArray, sharerNonce: ByteArray, reason: Int): ByteArray =
        mac(key, "tbs1-refuse", hubNonce, sharerNonce, byteArrayOf(reason.toByte()))

    /** Public tag of a share key (mDNS TXT "k="): hubs skip other servers' sharers. */
    fun keyId(key: ByteArray): String = mac(key, "tbs1-keyid").copyOf(4).joinToString("") { "%02x".format(it) }

    fun macEquals(a: ByteArray, b: ByteArray): Boolean = java.security.MessageDigest.isEqual(a, b)

    fun reasonText(r: Int): String = when (r) {
        R_NOT_ALLOWED -> "not allowed"
        R_CAP -> "data limit reached"
        R_BATTERY -> "battery low"
        R_PAUSED -> "sharing turned off"
        R_NO_CELLULAR -> "no mobile data"
        R_REPLACED -> "replaced"
        R_IDLE -> "no answer"
        else -> "stopped"
    }
}
