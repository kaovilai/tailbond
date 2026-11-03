package dev.tailbond.app.backup

import kotlinx.serialization.SerialName
import kotlinx.serialization.Serializable
import kotlinx.serialization.json.Json
import java.security.SecureRandom
import java.util.Base64
import javax.crypto.AEADBadTagException
import javax.crypto.Cipher
import javax.crypto.spec.GCMParameterSpec
import javax.crypto.spec.SecretKeySpec

/**
 * The `.tbbackup` header line (docs/architecture.md, "Backup and restore").
 * Field order is the contract's; the exact bytes of this line are the
 * AES-GCM additional data, so readers must never re-encode it.
 */
@Serializable
data class BackupHeader(
    val v: Int = 1,
    val kdf: String = "scrypt",
    val n: Int = 32768,
    val r: Int = 8,
    val p: Int = 1,
    val salt: String = "",
    val nonce: String = "",
    val role: String = "client",
    val platform: String = "android",
    val host: String = "",
    @SerialName("agent_version") val agentVersion: String = "",
    val created: String = "",
)

/** Why a file can't be opened. [message] is plain words for the UI. */
class BackupException(message: String, val kind: Kind) : Exception(message) {
    enum class Kind { NOT_A_BACKUP, UNSUPPORTED, WRONG_PASSPHRASE_OR_DAMAGED, WRONG_KIND, TOO_SHORT }
}

/**
 * `.tbbackup` format 1:
 *
 *     TBBACKUP1\n
 *     <header JSON, one line>\n
 *     <base64(AES-256-GCM(key = scrypt(passphrase, salt, n, r, p, 32), nonce, plaintext, aad = header line))>\n
 *
 * The AAD is the header line's bytes without its newline; the ciphertext
 * carries GCM's 16-byte tag at the end (as Go's crypto/cipher and Java's
 * Cipher both produce); base64 is standard with padding, one line. The
 * passphrase is UTF-8.
 */
object BackupFile {
    const val MAGIC = "TBBACKUP1"
    const val MIN_PASSPHRASE = 10
    /** A file larger than this isn't one of ours (the server takes ≤ 1 MiB pushes, ≤ 2 MiB bodies). */
    const val MAX_FILE = 4L shl 20

    private val json = Json { encodeDefaults = true; ignoreUnknownKeys = true }

    data class Opened(val header: BackupHeader, val headerLine: String, val plaintext: ByteArray)

    /**
     * Encrypts [plaintext]. [salt] (16 bytes) and [nonce] (12 bytes) are
     * random unless given (fixed only for the interop vectors).
     */
    fun encrypt(
        plaintext: ByteArray,
        passphrase: String,
        host: String,
        appVersion: String,
        created: String,
        salt: ByteArray = random(16),
        nonce: ByteArray = random(12),
        n: Int = 32768, r: Int = 8, p: Int = 1,
    ): ByteArray {
        require(passphrase.length >= MIN_PASSPHRASE) { "the passphrase needs at least $MIN_PASSPHRASE characters" }
        require(salt.size == 16 && nonce.size == 12)
        val b64 = Base64.getEncoder()
        val header = BackupHeader(
            n = n, r = r, p = p, salt = b64.encodeToString(salt), nonce = b64.encodeToString(nonce),
            host = host, agentVersion = appVersion, created = created,
        )
        val line = json.encodeToString(BackupHeader.serializer(), header)
        val key = Scrypt.derive(passphrase.toByteArray(Charsets.UTF_8), salt, n, r, p, 32)
        val c = Cipher.getInstance("AES/GCM/NoPadding")
        c.init(Cipher.ENCRYPT_MODE, SecretKeySpec(key, "AES"), GCMParameterSpec(128, nonce))
        c.updateAAD(line.toByteArray(Charsets.UTF_8))
        val ct = c.doFinal(plaintext)
        return "$MAGIC\n$line\n${b64.encodeToString(ct)}\n".toByteArray(Charsets.UTF_8)
    }

    /** The header only (no passphrase needed): shows what a file is before asking for it. */
    fun peek(file: ByteArray): Pair<BackupHeader, String> {
        if (file.size > MAX_FILE) throw BackupException("This file is too big to be a tailbond backup", BackupException.Kind.NOT_A_BACKUP)
        val text = file.toString(Charsets.UTF_8)
        val lines = text.split('\n')
        if (lines.size < 3 || lines[0].trimEnd('\r') != MAGIC) {
            throw BackupException("This isn't a tailbond backup file", BackupException.Kind.NOT_A_BACKUP)
        }
        val line = lines[1]
        val h = try {
            json.decodeFromString(BackupHeader.serializer(), line)
        } catch (_: Exception) {
            throw BackupException("This backup file is damaged (its header can't be read)", BackupException.Kind.WRONG_PASSPHRASE_OR_DAMAGED)
        }
        if (h.v != 1 || h.kdf != "scrypt") {
            throw BackupException("This backup was made by a newer tailbond (format ${h.v}, ${h.kdf}). Update the app first", BackupException.Kind.UNSUPPORTED)
        }
        // bounds before any work: a hostile file mustn't make the phone allocate gigabytes
        if (h.n < 2 || h.n > 1 shl 17 || h.n and (h.n - 1) != 0 || h.r !in 1..16 || h.p !in 1..4) {
            throw BackupException("This backup uses settings this app doesn't open (scrypt n=${h.n} r=${h.r} p=${h.p})", BackupException.Kind.UNSUPPORTED)
        }
        return h to line
    }

    /** Decrypts [file]. A wrong passphrase and a damaged file look the same (the GCM check fails). */
    fun decrypt(file: ByteArray, passphrase: String): Opened {
        val (h, line) = peek(file)
        val dec = Base64.getDecoder()
        val salt = runCatching { dec.decode(h.salt) }.getOrNull()
        val nonce = runCatching { dec.decode(h.nonce) }.getOrNull()
        val body = runCatching { dec.decode(file.toString(Charsets.UTF_8).split('\n')[2].trim()) }.getOrNull()
        if (salt == null || salt.size != 16 || nonce == null || nonce.size != 12 || body == null) {
            throw BackupException(WRONG, BackupException.Kind.WRONG_PASSPHRASE_OR_DAMAGED)
        }
        if (body.size < 16) throw BackupException(WRONG, BackupException.Kind.TOO_SHORT)
        val key = Scrypt.derive(passphrase.toByteArray(Charsets.UTF_8), salt, h.n, h.r, h.p, 32)
        val plain = try {
            val c = Cipher.getInstance("AES/GCM/NoPadding")
            c.init(Cipher.DECRYPT_MODE, SecretKeySpec(key, "AES"), GCMParameterSpec(128, nonce))
            c.updateAAD(line.toByteArray(Charsets.UTF_8))
            c.doFinal(body)
        } catch (_: AEADBadTagException) {
            throw BackupException(WRONG, BackupException.Kind.WRONG_PASSPHRASE_OR_DAMAGED)
        } catch (_: java.security.GeneralSecurityException) {
            throw BackupException(WRONG, BackupException.Kind.WRONG_PASSPHRASE_OR_DAMAGED)
        }
        return Opened(h, line, plain)
    }

    const val WRONG = "Wrong passphrase or damaged file"

    private val rng = SecureRandom()
    private fun random(n: Int) = ByteArray(n).also { rng.nextBytes(it) }
}
