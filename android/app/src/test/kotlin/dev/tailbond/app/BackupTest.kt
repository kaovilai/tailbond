package dev.tailbond.app

import dev.tailbond.app.backup.AndroidBackup
import dev.tailbond.app.backup.AndroidConfig
import dev.tailbond.app.backup.BackupException
import dev.tailbond.app.backup.BackupFile
import dev.tailbond.app.backup.Restore
import dev.tailbond.app.backup.Scrypt
import dev.tailbond.app.failover.FailoverPolicy
import dev.tailbond.app.model.AppSettings
import dev.tailbond.app.model.Mode
import dev.tailbond.app.model.ServerProfile
import dev.tailbond.app.model.WebsiteRule
import dev.tailbond.app.pairing.PairingPayload
import dev.tailbond.app.update.UpdatePolicy
import org.junit.Assert.assertEquals
import org.junit.Assert.assertFalse
import org.junit.Assert.assertTrue
import org.junit.Assert.fail
import org.junit.Test
import java.util.Base64

fun hex(s: String): ByteArray = s.chunked(2).map { it.toInt(16).toByte() }.toByteArray()

class ScryptTest {
    private fun h(b: ByteArray) = b.joinToString("") { "%02x".format(it) }

    // RFC 7914 section 11 (PBKDF2-HMAC-SHA256) and section 12 (scrypt)
    @Test fun pbkdf2Vectors() {
        assertEquals(
            "55ac046e56e3089fec1691c22544b605f94185216dde0465e68b9d57c20dacbc49ca9cccf179b645991664b39d77ef317c71b845b1e30bd509112041d3a19783",
            h(Scrypt.pbkdf2Sha256("passwd".toByteArray(), "salt".toByteArray(), 1, 64)),
        )
        assertEquals(
            "4ddcd8f60b98be21830cee5ef22701f9641a4418d04c0414aeff08876b34ab56a1d425a1225833549adb841b51c9b3176a272bdebba1d078478f62b397f33c8d",
            h(Scrypt.pbkdf2Sha256("Password".toByteArray(), "NaCl".toByteArray(), 80000, 64)),
        )
    }

    @Test fun scryptEmpty() = assertEquals(
        "77d6576238657b203b19ca42c18a0497f16b4844e3074ae8dfdffa3fede21442fcd0069ded0948f8326a753a0fc81f17e8d3e0fb2e0d3628cf35e20c38d18906",
        h(Scrypt.derive(ByteArray(0), ByteArray(0), 16, 1, 1, 64)),
    )

    @Test fun scryptNaCl() = assertEquals(
        "fdbabe1c9d3472007856e7190d01e9fe7c6ad7cbc8237830e77376634b3731622eaf30d92e22a3886ff109279d9830dac727afb94a83ee6d8360cbdfa2cc0640",
        h(Scrypt.derive("password".toByteArray(), "NaCl".toByteArray(), 1024, 8, 16, 64)),
    )

    @Test fun scryptSodiumChloride() = assertEquals(
        "7023bdcb3afd7348461c06cd81fd38ebfda8fbba904f8e3ea9b543f6545da1f2d5432955613f0fcf62d49705242a9af9e61e85dc0d651e40dfcf017b45575887",
        h(Scrypt.derive("pleaseletmein".toByteArray(), "SodiumChloride".toByteArray(), 16384, 8, 1, 64)),
    )

    @Test fun salsa208Vector() {
        // RFC 7914 section 8
        val input = hex(
            "7e879a214f3ec9867ca940e641718f26baee555b8c61c1b50df846116dcd3b1d" +
                "ee24f319df9b3d8514121e4b5ac5aa3276021d2909c74829edebc68db8b8c25e",
        )
        val words = IntArray(16) { i -> (0 until 4).fold(0) { acc, k -> acc or ((input[i * 4 + k].toInt() and 0xff) shl (8 * k)) } }
        Scrypt.salsa208(words)
        val out = ByteArray(64) { i -> (words[i / 4] ushr (8 * (i % 4))).toByte() }
        assertEquals(
            "a41f859c6608cc993b81cacb020cef05044b2181a2fd337dfd7b1c6396682f29" +
                "b4393168e3c9e6bcfe6bc5b7a06d96bae424cc102c91745c24ad673dc7618f81",
            h(out),
        )
    }

    @Test fun rejectsBadParameters() {
        for ((n, r, p) in listOf(Triple(15, 1, 1), Triple(1, 1, 1), Triple(16, 0, 1), Triple(16, 1, 0))) {
            try {
                Scrypt.derive(ByteArray(1), ByteArray(1), n, r, p, 32); fail("n=$n r=$r p=$p")
            } catch (_: IllegalArgumentException) {
            }
        }
    }
}

object BackupFixtures {
    val home = ServerProfile(
        "h",
        PairingPayload(
            name = "pixel-8", server = "198.51.100.7:443", sni = "homebox.tail1234.ts.net", user = "pixel-8", key = "not-a-real-key",
            dns = listOf("10.77.0.1"), mode = "speed", serverName = "homebox", subnet = "10.77.0.0/24",
        ),
    )
    val vps = ServerProfile(
        "v",
        PairingPayload(
            name = "pixel-8", server = "203.0.113.9:8443", sni = "vps.tail1234.ts.net", user = "pixel-8", key = "also-not-a-key",
            dns = listOf("10.78.0.1"), mode = "speed", serverName = "vps", subnet = "10.78.0.0/24",
        ),
    )
    val settings = AppSettings(
        onboarded = true, mode = Mode.STREAMING, cellularCapMb = 5000, excludedApps = setOf("com.netflix.mediaclient"),
        websites = listOf(WebsiteRule("mybank.com", true, "Bank")), updatePolicy = UpdatePolicy.OFF,
    )
    val backup = AndroidBackup(config = AndroidConfig(listOf(home, vps), settings))
}

class BackupFileTest {
    private val pass = "correct horse battery"
    private val plain = BackupFixtures.backup.toJson().toByteArray()

    // n=1024 keeps the many tamper cases quick; the round trip uses the real parameters
    private fun small(p: String = pass) = BackupFile.encrypt(plain, p, "pixel-8", "0.2.0", "2026-11-02T10:00:00Z", n = 1024)

    @Test fun roundTripWithTheContractParameters() {
        val file = BackupFile.encrypt(plain, pass, "pixel-8", "0.2.0", "2026-11-02T10:00:00Z")
        val text = String(file)
        val lines = text.split('\n')
        assertEquals("TBBACKUP1", lines[0])
        assertTrue(lines[1].startsWith("""{"v":1,"kdf":"scrypt","n":32768,"r":8,"p":1,"salt":""""))
        assertTrue(lines[1].contains(""""role":"client","platform":"android","host":"pixel-8","agent_version":"0.2.0","created":"2026-11-02T10:00:00Z"}"""))
        assertEquals("", lines[3])
        val opened = BackupFile.decrypt(file, pass)
        assertEquals(plain.toList(), opened.plaintext.toList())
        assertEquals(BackupFixtures.backup, AndroidBackup.fromJson(String(opened.plaintext)))
        // never readable without the passphrase
        assertFalse(text.contains("not-a-real-key"))
        assertFalse(text.contains("homebox"))
    }

    @Test fun freshSaltAndNonceEveryTime() {
        val a = String(small()).split('\n')[1]
        val b = String(small()).split('\n')[1]
        assertTrue(a != b)
    }

    private fun expectWrong(file: ByteArray, p: String = pass) {
        try {
            BackupFile.decrypt(file, p); fail("opened")
        } catch (e: BackupException) {
            assertEquals(BackupFile.WRONG, e.message)
        }
    }

    @Test fun wrongPassphrase() = expectWrong(small(), "correct horse battery!")

    @Test fun corruptedCiphertext() {
        val lines = String(small()).split('\n').toMutableList()
        val ct = Base64.getDecoder().decode(lines[2])
        ct[ct.size / 2] = (ct[ct.size / 2].toInt() xor 1).toByte()
        lines[2] = Base64.getEncoder().encodeToString(ct)
        expectWrong(lines.joinToString("\n").toByteArray())
        // truncated
        lines[2] = lines[2].take(40)
        expectWrong(lines.joinToString("\n").toByteArray())
    }

    @Test fun headerTamper() {
        // the header is the AAD: changing any of it (here the host name) fails the check
        val file = String(small()).replace("\"host\":\"pixel-8\"", "\"host\":\"pixel-9\"").toByteArray()
        expectWrong(file)
        // even whitespace (a reader that re-encodes the header would break this)
        expectWrong(String(small()).replaceFirst("{\"v\":1,", "{\"v\": 1,").toByteArray())
    }

    @Test fun notABackupOrNewer() {
        for (bad in listOf("hello", "TBBACKUP2\n{}\nAAAA\n", "")) {
            try {
                BackupFile.decrypt(bad.toByteArray(), pass); fail(bad)
            } catch (e: BackupException) {
                assertEquals(BackupException.Kind.NOT_A_BACKUP, e.kind)
            }
        }
        val v2 = String(small()).replaceFirst("\"v\":1", "\"v\":2").toByteArray()
        try {
            BackupFile.decrypt(v2, pass); fail()
        } catch (e: BackupException) {
            assertEquals(BackupException.Kind.UNSUPPORTED, e.kind)
        }
        // a hostile n would need gigabytes: refused before any work
        val huge = String(small()).replaceFirst("\"n\":1024", "\"n\":1048576").toByteArray()
        try {
            BackupFile.decrypt(huge, pass); fail()
        } catch (e: BackupException) {
            assertEquals(BackupException.Kind.UNSUPPORTED, e.kind)
        }
    }

    @Test fun shortPassphraseRefused() {
        try {
            small("too short"); fail()
        } catch (_: IllegalArgumentException) {
        }
    }

    @Test fun crlfLineEndingsStillOpen() {
        // a file that went through a Windows editor: the header bytes are the AAD, so only the first line may change
        val lines = String(small()).split('\n')
        val crlf = (lines[0] + "\r\n" + lines[1] + "\n" + lines[2] + "\r\n").toByteArray()
        assertEquals(plain.toList(), BackupFile.decrypt(crlf, pass).plaintext.toList())
    }
}

class RestorePreviewTest {
    private val h = dev.tailbond.app.backup.BackupHeader(host = "pixel-8", agentVersion = "0.2.0", created = "2026-11-02T10:00:00Z")

    @Test fun onlyAndroidClientBackups() {
        for ((role, platform) in listOf("server" to "fedora", "client" to "openwrt", "server" to "android")) {
            try {
                Restore.checkKind(h.copy(role = role, platform = platform)); fail("$role/$platform")
            } catch (e: BackupException) {
                assertEquals(BackupException.Kind.WRONG_KIND, e.kind)
            }
        }
        Restore.checkKind(h)
    }

    @Test fun listsWhatChanges() {
        val current = AppSettings(onboarded = true, handedToTailscale = true)
        val p = Restore.preview(h, BackupFixtures.backup, listOf(BackupFixtures.home.copy(payload = BackupFixtures.home.payload.copy(key = "old"))), current) {
            it != "com.netflix.mediaclient"
        }
        val by = p.changes.associateBy { it.what }
        assertEquals("homebox" to "homebox and vps", by.getValue("Servers").let { it.from to it.to })
        assertTrue(by.containsKey("Pairing for homebox"))
        assertEquals("the server's choice" to "Smooth calls and video", by.getValue("Mode").let { it.from to it.to })
        assertEquals("none" to "1 app", by.getValue("Apps that skip tailbond").let { it.from to it.to })
        assertEquals("none" to "mybank.com", by.getValue("Websites that skip tailbond").let { it.from to it.to })
        assertEquals("no limit" to "5 GB a month", by.getValue("Mobile data limit").let { it.from to it.to })
        assertEquals("Tell me" to "Off", by.getValue("Updates").let { it.from to it.to })
        assertFalse(by.containsKey("Server choice"))
        assertEquals(1, p.warnings.size)
        assertTrue(p.warnings[0].startsWith("1 app from the backup isn't installed"))
        // this phone's own state stays
        assertTrue(p.backup.config.settings.handedToTailscale)
        assertTrue(p.backup.config.settings.onboarded)
    }

    @Test fun sameSetupChangesNothing() {
        val b = BackupFixtures.backup
        val p = Restore.preview(h, b, b.config.servers, b.config.settings)
        assertTrue(p.changes.toString(), p.changes.isEmpty())
        assertTrue(p.warnings.isEmpty())
    }

    @Test fun alwaysServerNamesIt() {
        val b = BackupFixtures.backup.let { it.copy(config = it.config.copy(settings = it.config.settings.copy(failoverPolicy = FailoverPolicy.ALWAYS, alwaysServerId = "v"))) }
        val p = Restore.preview(h, b, b.config.servers, BackupFixtures.settings)
        assertEquals("always vps", p.changes.single { it.what == "Server choice" }.to)
    }
}
