package dev.tailbond.app.ui

import android.Manifest
import android.content.pm.PackageManager
import androidx.activity.compose.rememberLauncherForActivityResult
import androidx.activity.result.contract.ActivityResultContracts
import androidx.compose.foundation.layout.Arrangement
import androidx.compose.foundation.layout.Column
import androidx.compose.foundation.layout.Row
import androidx.compose.foundation.layout.Spacer
import androidx.compose.foundation.layout.aspectRatio
import androidx.compose.foundation.layout.fillMaxSize
import androidx.compose.foundation.layout.fillMaxWidth
import androidx.compose.foundation.layout.height
import androidx.compose.foundation.layout.padding
import androidx.compose.foundation.layout.size
import androidx.compose.foundation.layout.width
import androidx.compose.foundation.rememberScrollState
import androidx.compose.foundation.text.KeyboardOptions
import androidx.compose.foundation.verticalScroll
import androidx.compose.material.icons.Icons
import androidx.compose.material.icons.filled.ContentPaste
import androidx.compose.material.icons.filled.Devices
import androidx.compose.material.icons.filled.Lock
import androidx.compose.material.icons.filled.OpenInBrowser
import androidx.compose.material.icons.filled.QrCodeScanner
import androidx.compose.material.icons.filled.SwapHoriz
import androidx.compose.material.icons.filled.Warning
import androidx.compose.material.icons.filled.Wifi
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
import androidx.compose.ui.Alignment
import androidx.compose.ui.Modifier
import androidx.compose.ui.graphics.vector.ImageVector
import androidx.compose.ui.platform.LocalClipboardManager
import androidx.compose.ui.platform.LocalContext
import androidx.compose.ui.text.font.FontWeight
import androidx.compose.ui.text.input.ImeAction
import androidx.compose.ui.text.input.KeyboardType
import androidx.compose.ui.unit.dp
import androidx.core.content.ContextCompat
import dev.tailbond.app.model.serverShortName
import dev.tailbond.app.pairing.PairingPayload

/** First launch: what tailbond is to someone who uses the Tailscale app. */
@Composable
fun OnboardingScreen(onDone: () -> Unit, modifier: Modifier = Modifier) {
    Column(
        modifier.fillMaxSize().verticalScroll(rememberScrollState()).padding(24.dp),
        verticalArrangement = Arrangement.spacedBy(20.dp),
    ) {
        Spacer(Modifier.height(24.dp))
        Text("tailbond", style = MaterialTheme.typography.labelLarge, color = MaterialTheme.colorScheme.primary)
        Text(
            "Use tailbond instead of Tailscale on this phone",
            style = MaterialTheme.typography.headlineMedium,
            fontWeight = FontWeight.SemiBold,
        )
        Point(
            Icons.Filled.Wifi,
            "Wi-Fi and mobile data together",
            "tailbond sends your traffic over both connections at once, to your own home server. Faster, and it keeps going when one drops.",
        )
        Point(
            Icons.Filled.Devices,
            "Still on your tailnet",
            "Your home server is on Tailscale, so your NAS and other Tailscale machines stay reachable, by name or 100.x address, through it.",
        )
        Point(
            Icons.Filled.SwapHoriz,
            "One VPN at a time",
            "Android allows one VPN app. While tailbond is on, Tailscale's VPN is off on this phone. Switch back to Tailscale any time.",
        )
        Spacer(Modifier.height(8.dp))
        Button(onClick = onDone, modifier = Modifier.fillMaxWidth().height(52.dp)) { Text("Get started") }
        Text(
            "You'll need your home server's name or the link to its page. If someone else set it up, ask them for it.",
            style = MaterialTheme.typography.bodyMedium,
            color = MaterialTheme.colorScheme.onSurfaceVariant,
        )
    }
}

@Composable
private fun Point(icon: ImageVector, title: String, body: String) {
    Row(verticalAlignment = Alignment.Top) {
        Icon(icon, null, Modifier.size(28.dp), tint = MaterialTheme.colorScheme.primary)
        Spacer(Modifier.width(16.dp))
        Column {
            Text(title, style = MaterialTheme.typography.titleMedium)
            Text(body, style = MaterialTheme.typography.bodyMedium, color = MaterialTheme.colorScheme.onSurfaceVariant)
        }
    }
}

/** Plain-words version of a pairing parser error; the raw reason stays as a detail. */
fun pairErrorText(reason: String): String = when {
    reason.contains("base64", true) || reason.contains("d= parameter", true) || reason.contains("JSON", true) ||
        reason.contains("Unexpected", true) -> "That isn't a complete tailbond link. Copy the whole link starting with tailbond://pair"
    reason.contains("not a tailbond", true) || reason.contains("expected tailbond", true) ->
        "That's not a tailbond pairing link. It should start with tailbond://pair"
    reason.contains("version", true) -> "This link is from a newer tailbond. Update the app and try again"
    else -> "That pairing link doesn't work"
}

@Composable
fun PairingScreen(state: UiState, actions: UiActions, modifier: Modifier = Modifier, startText: String = "") {
    val ctx = LocalContext.current
    var scanning by rememberSaveable { mutableStateOf(false) }
    var text by rememberSaveable { mutableStateOf(startText) }
    var server by rememberSaveable { mutableStateOf("") }
    val camera = rememberLauncherForActivityResult(ActivityResultContracts.RequestPermission()) { granted -> scanning = granted }
    @Suppress("DEPRECATION")
    val clipboard = LocalClipboardManager.current
    val error = state.pairError

    Column(
        modifier.verticalScroll(rememberScrollState()).padding(16.dp),
        verticalArrangement = Arrangement.spacedBy(12.dp),
    ) {
        Text("Add this phone to your home server", style = MaterialTheme.typography.headlineSmall)

        Card(Modifier.fillMaxWidth()) {
            Column(Modifier.padding(16.dp), verticalArrangement = Arrangement.spacedBy(10.dp)) {
                Text("On this phone (easiest)", style = MaterialTheme.typography.titleMedium)
                Text(
                    "Keep Tailscale on for now. Open your home server's page, tap Add this phone, and tailbond opens with everything filled in.",
                    style = MaterialTheme.typography.bodyMedium,
                )
                OutlinedTextField(
                    value = server,
                    onValueChange = { server = it.trim() },
                    label = { Text("Home server's Tailscale name") },
                    placeholder = { Text("e.g. homebox or homebox.tail1234.ts.net") },
                    singleLine = true,
                    keyboardOptions = KeyboardOptions(keyboardType = KeyboardType.Uri, imeAction = ImeAction.Go),
                    modifier = Modifier.fillMaxWidth(),
                )
                Button(onClick = { actions.openHomeServerPage(server) }, enabled = server.isNotBlank()) {
                    Icon(Icons.Filled.OpenInBrowser, null)
                    Text("  Open home server page")
                }
                Text(
                    "It's the machine that runs tailbond, listed in the Tailscale app's Machines. " +
                        "Not sure which one? Ask whoever set it up for the name, or for a link to its page.",
                    style = MaterialTheme.typography.bodySmall,
                    color = MaterialTheme.colorScheme.onSurfaceVariant,
                )
            }
        }

        Card(Modifier.fillMaxWidth()) {
            Column(Modifier.padding(16.dp), verticalArrangement = Arrangement.spacedBy(10.dp)) {
                Text("From another screen", style = MaterialTheme.typography.titleMedium)
                Text(
                    "If the home server's page is open on a computer, tap Add a device there and scan the code it shows.",
                    style = MaterialTheme.typography.bodyMedium,
                )
                if (scanning) {
                    Card(Modifier.fillMaxWidth().aspectRatio(1f)) {
                        QrScanner(
                            onResult = { scanned -> scanning = false; actions.offerPairing(scanned) },
                            modifier = Modifier.fillMaxWidth().aspectRatio(1f),
                        )
                    }
                    OutlinedButton(onClick = { scanning = false }) { Text("Stop scanning") }
                } else {
                    OutlinedButton(onClick = {
                        if (ContextCompat.checkSelfPermission(ctx, Manifest.permission.CAMERA) == PackageManager.PERMISSION_GRANTED) {
                            scanning = true
                        } else {
                            camera.launch(Manifest.permission.CAMERA)
                        }
                    }) {
                        Icon(Icons.Filled.QrCodeScanner, null)
                        Text("  Scan code from your home server")
                    }
                }
            }
        }

        Card(Modifier.fillMaxWidth()) {
            Column(Modifier.padding(16.dp), verticalArrangement = Arrangement.spacedBy(10.dp)) {
                Text("Have a link?", style = MaterialTheme.typography.titleMedium)
                OutlinedTextField(
                    value = text,
                    onValueChange = { text = it; actions.clearPairError() },
                    label = { Text("Pairing link") },
                    placeholder = { Text("tailbond://pair?d=…") },
                    modifier = Modifier.fillMaxWidth(),
                    minLines = 2,
                    maxLines = 4,
                    isError = error != null,
                    supportingText = {
                        if (error != null) Text(pairErrorText(error))
                    },
                )
                Row(horizontalArrangement = Arrangement.spacedBy(8.dp)) {
                    OutlinedButton(onClick = { clipboard.getText()?.text?.let { text = it; actions.offerPairing(it) } }) {
                        Icon(Icons.Filled.ContentPaste, null)
                        Text("  Paste")
                    }
                    Button(onClick = { actions.offerPairing(text) }, enabled = text.isNotBlank()) { Text("Add") }
                }
            }
        }
    }
}

/** Confirmation before a scanned/pasted/deep-linked pairing is saved. */
@Composable
fun ConfirmPairingDialog(p: PairingPayload, replacing: Boolean, onConfirm: () -> Unit, onDismiss: () -> Unit) {
    var details by rememberSaveable { mutableStateOf(false) }
    val server = serverShortName(p)
    AlertDialog(
        onDismissRequest = onDismiss,
        title = { Text(if (replacing) "Switch this phone to $server?" else "Add this phone to $server?") },
        text = {
            Column(verticalArrangement = Arrangement.spacedBy(10.dp)) {
                Text("tailbond will connect to your home server $server over Wi-Fi and mobile data. Your Tailscale machines stay reachable through it.")
                Text("It connects over the internet to ${p.server}.", style = MaterialTheme.typography.bodySmall)
                if (p.name.isNotBlank()) Text("This phone will be called \"${p.name}\" on $server.")
                when {
                    p.certSha256 != null -> Row(verticalAlignment = Alignment.CenterVertically) {
                        Icon(Icons.Filled.Lock, null, Modifier.size(18.dp))
                        Text("  Locked to $server's certificate", style = MaterialTheme.typography.bodySmall)
                    }
                    p.insecure -> Row {
                        Icon(Icons.Filled.Warning, null, Modifier.size(18.dp), tint = MaterialTheme.colorScheme.error)
                        Text(
                            "  This link doesn't let tailbond check that it's really your home server. Only add it if you created it yourself just now.",
                            color = MaterialTheme.colorScheme.error,
                            style = MaterialTheme.typography.bodySmall,
                        )
                    }
                    else -> Row(verticalAlignment = Alignment.CenterVertically) {
                        Icon(Icons.Filled.Lock, null, Modifier.size(18.dp))
                        Text("  Secure connection to $server", style = MaterialTheme.typography.bodySmall)
                    }
                }
                if (replacing) Text("This replaces the home server this phone uses now.", style = MaterialTheme.typography.bodySmall)
                TextButton(onClick = { details = !details }) { Text(if (details) "Hide details" else "Technical details") }
                if (details) {
                    val small = MaterialTheme.typography.bodySmall
                    Text("Address: ${p.server}", style = small)
                    p.sni?.let { Text("Name: $it", style = small) }
                    Text("Mode: ${p.mode} · DNS: ${p.dns.joinToString().ifEmpty { "10.77.0.1" }}", style = small)
                    if (p.tailnetRoutes.isNotEmpty()) Text("Tailnet routes: ${p.tailnetRoutes.joinToString()}", style = small)
                    p.certSha256?.let { Text("Certificate SHA-256: ${it.take(16)}…", style = small) }
                }
            }
        },
        confirmButton = { TextButton(onClick = onConfirm) { Text(if (replacing) "Switch" else "Add phone") } },
        dismissButton = { TextButton(onClick = onDismiss) { Text("Cancel") } },
    )
}
