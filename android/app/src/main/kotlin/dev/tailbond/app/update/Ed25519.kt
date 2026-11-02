/*
 * Ed25519 signature verification (verify only), ported to Kotlin from the
 * reference implementation in RFC 8032, section 6 ("Ed25519 Python
 * illustrative code").
 *
 * Copyright (c) 2017 IETF Trust and the persons identified as authors of
 * RFC 8032. All rights reserved.
 *
 * Redistribution and use in source and binary forms, with or without
 * modification, is permitted pursuant to, and subject to the license terms
 * contained in, the Simplified BSD License set forth in Section 4.c of the
 * IETF Trust's Legal Provisions Relating to IETF Documents
 * (https://trustee.ietf.org/license-info). See android/NOTICE.
 */
package dev.tailbond.app.update

import java.math.BigInteger
import java.security.MessageDigest

/**
 * Plain Ed25519 (RFC 8032, no prehash, no context) signature check.
 *
 * Android has no platform Ed25519 below API 33, and the app checks one
 * manifest every few hours, so a small BigInteger port of the RFC's own
 * reference code is enough: about 1 ms-50 ms per check, no dependency, easy
 * to audit against the RFC. It is not constant time, which doesn't matter
 * for verifying public data with a public key.
 */
object Ed25519 {
    private val P: BigInteger = BigInteger.ONE.shiftLeft(255).subtract(BigInteger.valueOf(19))
    private val Q: BigInteger = BigInteger.ONE.shiftLeft(252).add(BigInteger("27742317777372353535851937790883648493"))
    private val TWO = BigInteger.valueOf(2)
    private val D: BigInteger = BigInteger.valueOf(-121665).multiply(inv(BigInteger.valueOf(121666))).mod(P)
    private val SQRT_M1: BigInteger = TWO.modPow(P.subtract(BigInteger.ONE).shiftRight(2), P)
    private val MASK_255: BigInteger = BigInteger.ONE.shiftLeft(255).subtract(BigInteger.ONE)

    /** A point in extended homogeneous coordinates (X:Y:Z:T), x = X/Z, y = Y/Z, x*y = T/Z. */
    private class Pt(val x: BigInteger, val y: BigInteger, val z: BigInteger, val t: BigInteger)

    private val IDENTITY = Pt(BigInteger.ZERO, BigInteger.ONE, BigInteger.ONE, BigInteger.ZERO)
    private val G: Pt = run {
        val gy = BigInteger.valueOf(4).multiply(inv(BigInteger.valueOf(5))).mod(P)
        val gx = recoverX(gy, 0)!!
        Pt(gx, gy, BigInteger.ONE, gx.multiply(gy).mod(P))
    }

    private fun inv(x: BigInteger): BigInteger = x.modPow(P.subtract(TWO), P)

    private fun add(a: Pt, b: Pt): Pt {
        val aa = a.y.subtract(a.x).multiply(b.y.subtract(b.x)).mod(P)
        val bb = a.y.add(a.x).multiply(b.y.add(b.x)).mod(P)
        val cc = TWO.multiply(a.t).multiply(b.t).multiply(D).mod(P)
        val dd = TWO.multiply(a.z).multiply(b.z).mod(P)
        val e = bb.subtract(aa)
        val f = dd.subtract(cc)
        val g = dd.add(cc)
        val h = bb.add(aa)
        return Pt(e.multiply(f).mod(P), g.multiply(h).mod(P), f.multiply(g).mod(P), e.multiply(h).mod(P))
    }

    private fun mul(s: BigInteger, p: Pt): Pt {
        var q = IDENTITY
        var base = p
        for (i in 0 until s.bitLength()) {
            if (s.testBit(i)) q = add(q, base)
            base = add(base, base)
        }
        return q
    }

    private fun same(a: Pt, b: Pt): Boolean =
        a.x.multiply(b.z).subtract(b.x.multiply(a.z)).mod(P).signum() == 0 &&
            a.y.multiply(b.z).subtract(b.y.multiply(a.z)).mod(P).signum() == 0

    private fun recoverX(y: BigInteger, sign: Int): BigInteger? {
        if (y >= P) return null
        val x2 = y.multiply(y).subtract(BigInteger.ONE).multiply(inv(D.multiply(y).multiply(y).add(BigInteger.ONE))).mod(P)
        if (x2.signum() == 0) return if (sign != 0) null else BigInteger.ZERO
        var x = x2.modPow(P.add(BigInteger.valueOf(3)).shiftRight(3), P)
        if (x.multiply(x).subtract(x2).mod(P).signum() != 0) x = x.multiply(SQRT_M1).mod(P)
        if (x.multiply(x).subtract(x2).mod(P).signum() != 0) return null
        if (x.testBit(0) != (sign == 1)) x = P.subtract(x)
        return x
    }

    private fun leInt(b: ByteArray, from: Int, len: Int): BigInteger {
        val be = ByteArray(len) { b[from + len - 1 - it] }
        return BigInteger(1, be)
    }

    private fun decompress(b: ByteArray, from: Int): Pt? {
        val v = leInt(b, from, 32)
        val sign = if (v.testBit(255)) 1 else 0
        val y = v.and(MASK_255)
        val x = recoverX(y, sign) ?: return null
        return Pt(x, y, BigInteger.ONE, x.multiply(y).mod(P))
    }

    /** True when [sig] (64 bytes) is a valid signature of [msg] by the raw 32-byte [publicKey]. */
    fun verify(publicKey: ByteArray, msg: ByteArray, sig: ByteArray): Boolean {
        if (publicKey.size != 32 || sig.size != 64) return false
        val a = decompress(publicKey, 0) ?: return false
        val r = decompress(sig, 0) ?: return false
        val s = leInt(sig, 32, 32)
        if (s >= Q) return false
        val md = MessageDigest.getInstance("SHA-512")
        md.update(sig, 0, 32)
        md.update(publicKey)
        md.update(msg)
        val h = leInt(md.digest(), 0, 64).mod(Q)
        return same(mul(s, G), add(r, mul(h, a)))
    }
}
