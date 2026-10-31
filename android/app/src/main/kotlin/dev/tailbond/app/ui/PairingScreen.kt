package dev.tailbond.app.ui

import android.Manifest
import android.content.pm.PackageManager
import androidx.activity.compose.rememberLauncherForActivityResult
import androidx.activity.result.contract.ActivityResultContracts
import androidx.compose.foundation.layout.Arrangement
import androidx.compose.foundation.layout.Column
import androidx.compose.foundation.layout.Row
import androidx.compose.foundation.layout.aspectRatio
import androidx.compose.foundation.layout.fillMaxWidth
import androidx.compose.foundation.layout.padding
import androidx.compose.foundation.rememberScrollState
import androidx.compose.foundation.verticalScroll
import androidx.compose.material.icons.Icons
import androidx.compose.material.icons.filled.ContentPaste
import androidx.compose.material.icons.filled.QrCodeScanner
import androidx.compose.material3.AlertDialog
import androidx.compose.material3.Button
import androidx.compose.material3.Card
import androidx.compose.material3.Icon
import androidx.compose.material3.MaterialTheme
import androidx.compose.material3.OutlinedButton
import androidx.compose.material3.OutlinedTextField
import androidx.compose.material3.Text
import androidx.compose.material3.TextButton
import androidx.compose.runtime.Composable
import androidx.compose.runtime.getValue
import androidx.compose.runtime.mutableStateOf
import androidx.compose.runtime.saveable.rememberSaveable
import androidx.compose.runtime.setValue
import androidx.compose.ui.Modifier
import androidx.compose.ui.platform.LocalClipboardManager
import androidx.compose.ui.platform.LocalContext
import androidx.compose.ui.unit.dp
import androidx.core.content.ContextCompat
import dev.tailbond.app.pairing.PairingPayload

@Composable
fun PairingScreen(vm: MainViewModel, error: String?, modifier: Modifier = Modifier) {
    val ctx = LocalContext.current
    var scanning by rememberSaveable { mutableStateOf(false) }
    var text by rememberSaveable { mutableStateOf("") }
    val camera = rememberLauncherForActivityResult(ActivityResultContracts.RequestPermission()) { granted ->
        scanning = granted
    }
    @Suppress("DEPRECATION")
    val clipboard = LocalClipboardManager.current

    Column(
        modifier.verticalScroll(rememberScrollState()).padding(16.dp),
        verticalArrangement = Arrangement.spacedBy(12.dp),
    ) {
        Text("Pair with your server", style = MaterialTheme.typography.headlineSmall)
        Text(
            "On your tailbond server's dashboard, open Clients → Add client, then scan its QR code " +
                "or paste the tailbond://pair link.",
            style = MaterialTheme.typography.bodyMedium,
        )
        if (scanning) {
            Card(Modifier.fillMaxWidth().aspectRatio(1f)) {
                QrScanner(
                    onResult = { scanned -> scanning = false; vm.offerPairing(scanned) },
                    modifier = Modifier.fillMaxWidth().aspectRatio(1f),
                )
            }
            OutlinedButton(onClick = { scanning = false }) { Text("Stop scanning") }
        } else {
            Button(onClick = {
                if (ContextCompat.checkSelfPermission(ctx, Manifest.permission.CAMERA) == PackageManager.PERMISSION_GRANTED) {
                    scanning = true
                } else {
                    camera.launch(Manifest.permission.CAMERA)
                }
            }) {
                Icon(Icons.Filled.QrCodeScanner, null)
                Text("  Scan QR code")
            }
        }
        OutlinedTextField(
            value = text,
            onValueChange = { text = it; vm.clearPairError() },
            label = { Text("tailbond://pair?d=…") },
            modifier = Modifier.fillMaxWidth(),
            minLines = 2,
            maxLines = 5,
            isError = error != null,
            supportingText = { if (error != null) Text(error) },
        )
        Row(horizontalArrangement = Arrangement.spacedBy(8.dp)) {
            OutlinedButton(onClick = { clipboard.getText()?.text?.let { text = it; vm.offerPairing(it) } }) {
                Icon(Icons.Filled.ContentPaste, null)
                Text("  Paste")
            }
            Button(onClick = { vm.offerPairing(text) }, enabled = text.isNotBlank()) { Text("Import") }
        }
    }
}

/** Confirmation before a scanned/pasted/deep-linked pairing replaces the stored one. */
@Composable
fun ConfirmPairingDialog(p: PairingPayload, replacing: Boolean, onConfirm: () -> Unit, onDismiss: () -> Unit) {
    AlertDialog(
        onDismissRequest = onDismiss,
        title = { Text(if (replacing) "Replace pairing?" else "Pair with this server?") },
        text = {
            Column(verticalArrangement = Arrangement.spacedBy(4.dp)) {
                Text("Name: ${p.name.ifBlank { "—" }}")
                Text("Server: ${p.server}")
                p.sni?.let { Text("TLS name: $it") }
                Text("Mode: ${p.mode}")
                Text("DNS: ${p.dns.joinToString().ifEmpty { "10.77.0.1" }}")
                if (p.tailnetRoutes.isNotEmpty()) Text("Tailnet routes: ${p.tailnetRoutes.joinToString()}")
                when {
                    p.certSha256 != null -> Text("Certificate: pinned (SHA-256 ${p.certSha256.take(16)}…)")
                    p.insecure -> Text(
                        "Warning: certificate verification is OFF and no pin was given. Anyone on the path can impersonate the server.",
                        color = MaterialTheme.colorScheme.error,
                    )
                    else -> Text("Certificate: verified by the system trust store")
                }
            }
        },
        confirmButton = { TextButton(onClick = onConfirm) { Text("Pair") } },
        dismissButton = { TextButton(onClick = onDismiss) { Text("Cancel") } },
    )
}
