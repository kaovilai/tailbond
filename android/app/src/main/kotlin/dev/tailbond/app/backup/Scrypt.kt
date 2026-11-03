package dev.tailbond.app.backup

import java.security.MessageDigest

/**
 * scrypt (RFC 7914): PBKDF2-HMAC-SHA256, then ROMix with Salsa20/8 BlockMix.
 * Android has no scrypt in the platform; this follows the RFC's
 * pseudo-code directly (checked against its test vectors in ScryptTest).
 * With the backup parameters (N=32768, r=8, p=1) it needs 32 MiB and about
 * a second on a phone.
 */
object Scrypt {
    fun derive(password: ByteArray, salt: ByteArray, n: Int, r: Int, p: Int, dkLen: Int): ByteArray {
        require(n > 1 && n and (n - 1) == 0) { "N must be a power of 2 above 1" }
        require(r >= 1 && p >= 1 && dkLen >= 1) { "bad parameters" }
        require(r.toLong() * n * 128 <= Int.MAX_VALUE && r.toLong() * p * 128 <= Int.MAX_VALUE) { "parameters too large" }
        val blockInts = 32 * r
        val b = pbkdf2Sha256(password, salt, 1, p * 128 * r)
        val x = IntArray(blockInts)
        val v = IntArray(blockInts * n)
        val tmp = IntArray(blockInts)
        for (i in 0 until p) {
            val off = i * 128 * r
            for (k in 0 until blockInts) x[k] = le32(b, off + 4 * k)
            roMix(x, v, tmp, r, n)
            for (k in 0 until blockInts) put32(b, off + 4 * k, x[k])
        }
        return pbkdf2Sha256(password, b, 1, dkLen)
    }

    /** ROMix in place on [x] (32*r ints). */
    private fun roMix(x: IntArray, v: IntArray, tmp: IntArray, r: Int, n: Int) {
        val len = 32 * r
        for (i in 0 until n) {
            System.arraycopy(x, 0, v, i * len, len)
            blockMix(x, tmp, r)
        }
        for (i in 0 until n) {
            // Integerify: the first word of the last 64-byte block, mod N (N ≤ 2^31)
            val j = x[(2 * r - 1) * 16] and (n - 1)
            val base = j * len
            for (k in 0 until len) x[k] = x[k] xor v[base + k]
            blockMix(x, tmp, r)
        }
    }

    /** BlockMix_{Salsa20/8, r} in place on [b] (2r blocks of 16 ints), via [y]. */
    private fun blockMix(b: IntArray, y: IntArray, r: Int) {
        val xb = IntArray(16)
        System.arraycopy(b, (2 * r - 1) * 16, xb, 0, 16)
        for (i in 0 until 2 * r) {
            for (k in 0 until 16) xb[k] = xb[k] xor b[i * 16 + k]
            salsa208(xb)
            // even blocks to the first half, odd blocks to the second
            val dst = if (i % 2 == 0) (i / 2) * 16 else (r + i / 2) * 16
            System.arraycopy(xb, 0, y, dst, 16)
        }
        System.arraycopy(y, 0, b, 0, 32 * r)
    }

    /** Salsa20/8 core, in place on 16 words. */
    internal fun salsa208(b: IntArray) {
        val x = b.copyOf()
        repeat(4) {
            x[4] = x[4] xor Integer.rotateLeft(x[0] + x[12], 7); x[8] = x[8] xor Integer.rotateLeft(x[4] + x[0], 9)
            x[12] = x[12] xor Integer.rotateLeft(x[8] + x[4], 13); x[0] = x[0] xor Integer.rotateLeft(x[12] + x[8], 18)
            x[9] = x[9] xor Integer.rotateLeft(x[5] + x[1], 7); x[13] = x[13] xor Integer.rotateLeft(x[9] + x[5], 9)
            x[1] = x[1] xor Integer.rotateLeft(x[13] + x[9], 13); x[5] = x[5] xor Integer.rotateLeft(x[1] + x[13], 18)
            x[14] = x[14] xor Integer.rotateLeft(x[10] + x[6], 7); x[2] = x[2] xor Integer.rotateLeft(x[14] + x[10], 9)
            x[6] = x[6] xor Integer.rotateLeft(x[2] + x[14], 13); x[10] = x[10] xor Integer.rotateLeft(x[6] + x[2], 18)
            x[3] = x[3] xor Integer.rotateLeft(x[15] + x[11], 7); x[7] = x[7] xor Integer.rotateLeft(x[3] + x[15], 9)
            x[11] = x[11] xor Integer.rotateLeft(x[7] + x[3], 13); x[15] = x[15] xor Integer.rotateLeft(x[11] + x[7], 18)
            x[1] = x[1] xor Integer.rotateLeft(x[0] + x[3], 7); x[2] = x[2] xor Integer.rotateLeft(x[1] + x[0], 9)
            x[3] = x[3] xor Integer.rotateLeft(x[2] + x[1], 13); x[0] = x[0] xor Integer.rotateLeft(x[3] + x[2], 18)
            x[6] = x[6] xor Integer.rotateLeft(x[5] + x[4], 7); x[7] = x[7] xor Integer.rotateLeft(x[6] + x[5], 9)
            x[4] = x[4] xor Integer.rotateLeft(x[7] + x[6], 13); x[5] = x[5] xor Integer.rotateLeft(x[4] + x[7], 18)
            x[11] = x[11] xor Integer.rotateLeft(x[10] + x[9], 7); x[8] = x[8] xor Integer.rotateLeft(x[11] + x[10], 9)
            x[9] = x[9] xor Integer.rotateLeft(x[8] + x[11], 13); x[10] = x[10] xor Integer.rotateLeft(x[9] + x[8], 18)
            x[12] = x[12] xor Integer.rotateLeft(x[15] + x[14], 7); x[13] = x[13] xor Integer.rotateLeft(x[12] + x[15], 9)
            x[14] = x[14] xor Integer.rotateLeft(x[13] + x[12], 13); x[15] = x[15] xor Integer.rotateLeft(x[14] + x[13], 18)
        }
        for (i in 0 until 16) b[i] += x[i]
    }

    /**
     * PBKDF2 with HMAC-SHA256 (RFC 8018). HMAC is done by hand on
     * MessageDigest: javax.crypto refuses an empty key, and RFC 7914's first
     * vector has an empty password.
     */
    fun pbkdf2Sha256(password: ByteArray, salt: ByteArray, iterations: Int, dkLen: Int): ByteArray {
        val hmac = HmacSha256(password)
        val out = ByteArray(dkLen)
        var block = 1
        var pos = 0
        while (pos < dkLen) {
            var u = hmac.mac(salt + byteArrayOf((block ushr 24).toByte(), (block ushr 16).toByte(), (block ushr 8).toByte(), block.toByte()))
            val t = u.copyOf()
            for (i in 1 until iterations) {
                u = hmac.mac(u)
                for (k in t.indices) t[k] = (t[k].toInt() xor u[k].toInt()).toByte()
            }
            val n = minOf(32, dkLen - pos)
            System.arraycopy(t, 0, out, pos, n)
            pos += n
            block++
        }
        return out
    }

    private class HmacSha256(key: ByteArray) {
        private val ipad = ByteArray(64)
        private val opad = ByteArray(64)

        init {
            val k = if (key.size > 64) MessageDigest.getInstance("SHA-256").digest(key) else key
            for (i in 0 until 64) {
                val b = if (i < k.size) k[i].toInt() else 0
                ipad[i] = (b xor 0x36).toByte()
                opad[i] = (b xor 0x5c).toByte()
            }
        }

        fun mac(msg: ByteArray): ByteArray {
            val md = MessageDigest.getInstance("SHA-256")
            md.update(ipad); md.update(msg)
            val inner = md.digest()
            md.update(opad); md.update(inner)
            return md.digest()
        }
    }

    private fun le32(b: ByteArray, off: Int): Int =
        (b[off].toInt() and 0xff) or ((b[off + 1].toInt() and 0xff) shl 8) or ((b[off + 2].toInt() and 0xff) shl 16) or ((b[off + 3].toInt() and 0xff) shl 24)

    private fun put32(b: ByteArray, off: Int, v: Int) {
        b[off] = v.toByte(); b[off + 1] = (v ushr 8).toByte(); b[off + 2] = (v ushr 16).toByte(); b[off + 3] = (v ushr 24).toByte()
    }
}
