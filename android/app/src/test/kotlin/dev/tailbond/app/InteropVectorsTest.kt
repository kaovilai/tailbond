package dev.tailbond.app

import dev.tailbond.app.backup.AndroidBackup
import dev.tailbond.app.backup.BackupFile
import dev.tailbond.app.update.ManifestVerifier
import dev.tailbond.app.update.Verified
import org.junit.Assert.assertEquals
import org.junit.Assert.assertTrue
import org.junit.Test
import java.io.File
import java.util.Base64

/**
 * Fixed-input vectors for checking other implementations (the Go agent)
 * against this app: `src/test/resources/interop/`. The files are made from
 * the inputs in [Vectors]; this test rebuilds them byte for byte and checks
 * that the committed ones still open. To write them again:
 *
 *     TB_WRITE_INTEROP=$PWD/android/app/src/test/resources/interop \
 *         android/scripts/gradle.sh :app:testDebugUnitTest --tests 'dev.tailbond.app.InteropVectorsTest'
 */
class InteropVectorsTest {
    object Vectors {
        /** Test-only ed25519 seed: never a real update key. */
        val seed = ByteArray(32) { (it + 1).toByte() }
        const val passphrase = "tailbond interop passphrase"
        val salt = ByteArray(16) { (0xa0 + it).toByte() }
        val nonce = ByteArray(12) { (0xb0 + it).toByte() }
        const val host = "pixel-8"
        const val version = "0.2.0"
        const val created = "2026-11-02T10:00:00Z"
        val manifest = (
            """{"format":1,"channel":"stable","version":"0.2.0","released":"2026-11-02T10:00:00Z","protocol":2,""" +
                """"requires_server_protocol":1,"security":false,"notes_url":"https://github.com/OWNER/REPO/releases/tag/v0.2.0",""" +
                """"artifacts":[{"name":"tailbond-0.2.0.apk","kind":"apk","arch":"arm64-v8a",""" +
                """"url":"https://github.com/OWNER/REPO/releases/download/v0.2.0/tailbond-0.2.0.apk",""" +
                """"sha256":"9f86d081884c7d659a2feaa0c55ad015a3bf4f1b2b0b822cd15d6c15b0f00a08","size":4,"version_code":20000}]}""" + "\n"
            ).toByteArray()
    }

    private fun build(): Map<String, ByteArray> {
        val kp = TestKeys.pair(Vectors.seed)
        val pub = TestKeys.raw(kp)
        val plaintext = BackupFixtures.backup.toJson().toByteArray()
        val backup = BackupFile.encrypt(
            plaintext, Vectors.passphrase, Vectors.host, Vectors.version, Vectors.created, Vectors.salt, Vectors.nonce,
        )
        val b64 = Base64.getEncoder()
        val vectors = buildString {
            append("{\n")
            append("  \"about\": \"Fixed inputs for stable.json(.sig) and android.tbbackup; made by android/app/src/test/kotlin/dev/tailbond/app/InteropVectorsTest.kt\",\n")
            append("  \"ed25519_seed_hex\": \"${Vectors.seed.joinToString("") { "%02x".format(it) }}\",\n")
            append("  \"public_key_b64\": \"${b64.encodeToString(pub)}\",\n")
            append("  \"key_id\": \"${ManifestVerifier.keyId(pub)}\",\n")
            append("  \"manifest_sha256\": \"${sha256(Vectors.manifest)}\",\n")
            append("  \"backup_passphrase\": \"${Vectors.passphrase}\",\n")
            append("  \"backup_salt_b64\": \"${b64.encodeToString(Vectors.salt)}\",\n")
            append("  \"backup_nonce_b64\": \"${b64.encodeToString(Vectors.nonce)}\",\n")
            append("  \"backup_scrypt\": {\"n\": 32768, \"r\": 8, \"p\": 1, \"key_len\": 32},\n")
            append("  \"backup_key_hex\": \"${dev.tailbond.app.backup.Scrypt.derive(Vectors.passphrase.toByteArray(), Vectors.salt, 32768, 8, 1, 32).joinToString("") { "%02x".format(it) }}\",\n")
            append("  \"backup_plaintext_sha256\": \"${sha256(plaintext)}\",\n")
            append("  \"backup_plaintext_file\": \"android-backup-plaintext.json\"\n")
            append("}\n")
        }
        return mapOf(
            "stable.json" to Vectors.manifest,
            "stable.json.sig" to TestKeys.sigFile(kp, Vectors.manifest),
            "android.tbbackup" to backup,
            "android-backup-plaintext.json" to plaintext,
            "vectors.json" to vectors.toByteArray(),
        )
    }

    private fun sha256(b: ByteArray) = java.security.MessageDigest.getInstance("SHA-256").digest(b).joinToString("") { "%02x".format(it) }

    private fun resource(name: String): ByteArray =
        javaClass.classLoader!!.getResourceAsStream("interop/$name")?.use { it.readBytes() } ?: error("missing interop/$name")

    @Test fun vectorsAreCurrent() {
        val built = build()
        System.getenv("TB_WRITE_INTEROP")?.takeIf { it.isNotBlank() }?.let { dir ->
            built.forEach { (name, bytes) -> File(dir, name).apply { parentFile?.mkdirs() }.writeBytes(bytes) }
        }
        built.forEach { (name, bytes) -> assertEquals("interop/$name changed", String(bytes), String(resource(name))) }
    }

    @Test fun committedFilesVerifyAndOpen() {
        val pub = TestKeys.raw(TestKeys.pair(Vectors.seed))
        val v = ManifestVerifier.verify(resource("stable.json"), resource("stable.json.sig"), listOf(pub), "stable")
        assertTrue(v.toString(), v is Verified.Ok)
        val opened = BackupFile.decrypt(resource("android.tbbackup"), Vectors.passphrase)
        assertEquals(String(resource("android-backup-plaintext.json")), String(opened.plaintext))
        assertEquals(BackupFixtures.backup, AndroidBackup.fromJson(String(opened.plaintext)))
    }
}
