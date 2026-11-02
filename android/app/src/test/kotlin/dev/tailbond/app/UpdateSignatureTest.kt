package dev.tailbond.app

import dev.tailbond.app.update.Ed25519
import dev.tailbond.app.update.ManifestVerifier
import dev.tailbond.app.update.Verified
import org.junit.Assert.assertEquals
import org.junit.Assert.assertFalse
import org.junit.Assert.assertTrue
import org.junit.Test
import java.math.BigInteger
import java.security.KeyPair
import java.security.KeyPairGenerator
import java.security.SecureRandom
import java.security.Signature
import java.security.spec.NamedParameterSpec
import java.util.Base64

/** Test keys from the JDK's own Ed25519 (an independent implementation), optionally from a fixed seed. */
object TestKeys {
    fun pair(seed: ByteArray? = null): KeyPair {
        val g = KeyPairGenerator.getInstance("Ed25519")
        if (seed != null) {
            g.initialize(NamedParameterSpec.ED25519, object : SecureRandom() {
                override fun nextBytes(bytes: ByteArray) { seed.copyInto(bytes, 0, 0, bytes.size) }
            })
        }
        return g.generateKeyPair()
    }

    /** The raw 32-byte key (the last 32 bytes of the X.509 encoding). */
    fun raw(kp: KeyPair): ByteArray = kp.public.encoded.takeLast(32).toByteArray()

    fun sign(kp: KeyPair, msg: ByteArray): ByteArray =
        Signature.getInstance("Ed25519").apply { initSign(kp.private); update(msg) }.sign()

    /** `<channel>.json.sig` content. */
    fun sigFile(kp: KeyPair, msg: ByteArray, keyId: String = ManifestVerifier.keyId(raw(kp))): ByteArray =
        """{"key_id":"$keyId","sig":"${Base64.getEncoder().encodeToString(sign(kp, msg))}"}""".toByteArray()
}

fun hex(s: String): ByteArray = s.chunked(2).map { it.toInt(16).toByte() }.toByteArray()

class Ed25519Test {
    @Test fun rfc8032Test1() {
        // RFC 8032 section 7.1, TEST 1 (empty message)
        val pub = hex("d75a980182b10ab7d54bfed3c964073a0ee172f3daa62325af021a68f707511a")
        val sig = hex(
            "e5564300c360ac729086e2cc806e828a84877f1eb8e5d974d873e065224901555fb8821590a33bacc61e39701cf9b46bd25bf5f0595bbe24655141438e7a100b",
        )
        assertTrue(Ed25519.verify(pub, ByteArray(0), sig))
        assertFalse(Ed25519.verify(pub, byteArrayOf(0), sig))
        // the same pair from the RFC's secret key through the JDK
        val kp = TestKeys.pair(hex("9d61b19deffd5a60ba844af492ec2cc44449c5697b326919703bac031cae7f60"))
        assertEquals(pub.toList(), TestKeys.raw(kp).toList())
        assertEquals(sig.toList(), TestKeys.sign(kp, ByteArray(0)).toList())
    }

    @Test fun rfc8032Test2() {
        // TEST 2: one byte 0x72
        val pub = hex("3d4017c3e843895a92b70aa74d1b7ebc9c982ccf2ec4968cc0cd55f12af4660c")
        val sig = hex(
            "92a009a9f0d4cab8720e820b5f642540a2b27b5416503f8fb3762223ebdb69da085ac1e43e15996e458f3613d0f11d8c387b2eaeb4302aeeb00d291612bb0c00",
        )
        assertTrue(Ed25519.verify(pub, byteArrayOf(0x72), sig))
    }

    @Test fun agreesWithTheJdk() {
        val rnd = java.util.Random(7)
        repeat(12) { i ->
            val kp = TestKeys.pair()
            val msg = ByteArray(i * 37).also { rnd.nextBytes(it) }
            val sig = TestKeys.sign(kp, msg)
            val pub = TestKeys.raw(kp)
            assertTrue(Ed25519.verify(pub, msg, sig))
            // any flipped bit in the message, R or S fails
            if (msg.isNotEmpty()) assertFalse(Ed25519.verify(pub, msg.copyOf().also { it[0] = (it[0].toInt() xor 1).toByte() }, sig))
            assertFalse(Ed25519.verify(pub, msg, sig.copyOf().also { it[5] = (it[5].toInt() xor 0x10).toByte() }))
            assertFalse(Ed25519.verify(pub, msg, sig.copyOf().also { it[40] = (it[40].toInt() xor 0x01).toByte() }))
        }
    }

    @Test fun rejectsMalleableAndMalformed() {
        val kp = TestKeys.pair()
        val msg = "manifest".toByteArray()
        val sig = TestKeys.sign(kp, msg)
        val pub = TestKeys.raw(kp)
        // S + L is the same point but not canonical: refused
        val l = BigInteger.ONE.shiftLeft(252).add(BigInteger("27742317777372353535851937790883648493"))
        val s = BigInteger(1, sig.copyOfRange(32, 64).reversedArray()).add(l)
        val sBytes = s.toByteArray().reversedArray().copyOf(32)
        assertFalse(Ed25519.verify(pub, msg, sig.copyOfRange(0, 32) + sBytes))
        assertFalse(Ed25519.verify(pub.copyOf(31), msg, sig))
        assertFalse(Ed25519.verify(pub, msg, sig.copyOf(63)))
        // another key
        assertFalse(Ed25519.verify(TestKeys.raw(TestKeys.pair()), msg, sig))
    }
}

class ManifestSignatureTest {
    private val kp = TestKeys.pair()
    private val keys = listOf(TestKeys.raw(kp))
    private val manifest = """{"format":1,"channel":"stable","version":"0.2.0","protocol":2,"requires_server_protocol":1,"security":false,
        |"notes_url":"https://example.com/v0.2.0","future_field":{"x":1},
        |"artifacts":[{"name":"tailbond-0.2.0.apk","kind":"apk","arch":"arm64-v8a","url":"https://example.com/a.apk","sha256":"00","size":10,"version_code":20000}]}""".trimMargin().toByteArray()

    @Test fun good() {
        val v = ManifestVerifier.verify(manifest, TestKeys.sigFile(kp, manifest), keys, "stable")
        assertTrue(v is Verified.Ok)
        v as Verified.Ok
        assertEquals("0.2.0", v.manifest.version)
        assertEquals(20000L, v.manifest.apk?.versionCode)
        assertEquals(2, v.manifest.protocol)
    }

    @Test fun tampered() {
        val sig = TestKeys.sigFile(kp, manifest)
        val changed = String(manifest).replace("0.2.0", "9.9.9").toByteArray()
        assertEquals(Verified.BadSignature, ManifestVerifier.verify(changed, sig, keys))
        // a single added space is tampering too: the signature covers the exact bytes
        assertEquals(Verified.BadSignature, ManifestVerifier.verify(manifest + ' '.code.toByte(), sig, keys))
    }

    @Test fun wrongKeyWithTheTrustedKeyId() {
        val other = TestKeys.pair()
        // signed by another key, but claiming the trusted key's id
        val sig = TestKeys.sigFile(other, manifest, keyId = ManifestVerifier.keyId(keys[0]))
        assertEquals(Verified.BadSignature, ManifestVerifier.verify(manifest, sig, keys))
    }

    @Test fun unknownKeyId() {
        val other = TestKeys.pair()
        val v = ManifestVerifier.verify(manifest, TestKeys.sigFile(other, manifest), keys)
        assertEquals(Verified.UnknownKey(ManifestVerifier.keyId(TestKeys.raw(other))), v)
    }

    @Test fun secondTrustedKeyWorks() {
        val other = TestKeys.pair()
        val v = ManifestVerifier.verify(manifest, TestKeys.sigFile(other, manifest), keys + listOf(TestKeys.raw(other)))
        assertTrue(v is Verified.Ok)
    }

    @Test fun noKeysMeansOff() {
        assertEquals(Verified.NoKeys, ManifestVerifier.verify(manifest, TestKeys.sigFile(kp, manifest), emptyList()))
        assertTrue(ManifestVerifier.parseKeys("").isEmpty())
    }

    @Test fun channelAndFormat() {
        val beta = String(manifest).replace("\"stable\"", "\"beta\"").toByteArray()
        assertTrue(ManifestVerifier.verify(beta, TestKeys.sigFile(kp, beta), keys, "stable") is Verified.Malformed)
        val f2 = String(manifest).replace("\"format\":1", "\"format\":2").toByteArray()
        assertTrue(ManifestVerifier.verify(f2, TestKeys.sigFile(kp, f2), keys) is Verified.Malformed)
        assertTrue(ManifestVerifier.verify(manifest, "not json".toByteArray(), keys) is Verified.Malformed)
    }

    @Test fun keyIdAndParsing() {
        val b64 = Base64.getEncoder().encodeToString(keys[0])
        val parsed = ManifestVerifier.parseKeys(" $b64 , notbase64!!, ${Base64.getEncoder().encodeToString(ByteArray(16))}")
        assertEquals(1, parsed.size)
        assertEquals(keys[0].toList(), parsed[0].toList())
        assertEquals(16, ManifestVerifier.keyId(keys[0]).length)
    }
}
