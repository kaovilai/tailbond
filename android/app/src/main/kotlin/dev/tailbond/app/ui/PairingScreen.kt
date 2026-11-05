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
import androidx.compose.foundation.text.KeyboardActions
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
import androidx.compose.foundation.layout.FlowRow
import androidx.compose.material3.AlertDialog
import androidx.compose.material3.AssistChip
import androidx.compose.material3.Button
import androidx.compose.material3.Card
import androidx.compose.material3.CircularProgressIndicator
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
import dev.tailbond.app.data.AskState
import dev.tailbond.app.data.PairRequestClient
import dev.tailbond.app.model.ServerList
import dev.tailbond.app.model.ServerProfile
import dev.tailbond.app.model.TunnelPlan
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
            "Your home server is on Tailscale, so your NAS and other Tailscale machines stay reachable, by name or 100.x address, through it. They see this phone as the home server.",
        )
        Point(
            Icons.Filled.SwapHoriz,
            "One VPN at a time",
            "Android allows one VPN app. While tailbond is on, Tailscale's VPN is off on this phone. Hand it back to Tailscale any time.",
        )
        Spacer(Modifier.height(8.dp))
        Button(onClick = onDone, modifier = Modifier.fillMaxWidth().height(52.dp).tvInitialFocus()) { Text("Get started") }
        Text(
            "Next you'll type your home server's name (the one in the Tailscale app) and ask it to connect. If someone else set it up, they approve it. " +
                // a TV has no Chrome to come back from
                if (LocalTv.current) "" else "Installed tailbond from your home server's page in Chrome? Easier: go back to Chrome and tap step 2 there; it fills everything in.",
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
        val adding = state.pairTarget == PairTarget.Add
        val repairing = (state.pairTarget as? PairTarget.Replace)?.let { t -> state.servers.firstOrNull { it.id == t.id } }
        val primary = state.servers.firstOrNull()?.name ?: "your home server"
        Text(
            when {
                adding -> "Add a backup server"
                repairing != null -> "Pair this phone again with ${repairing.name}"
                else -> "Add this phone to your home server"
            },
            style = MaterialTheme.typography.headlineSmall,
        )
        if (adding) {
            Text(
                "A backup server is a second tailbond server, like a VPS. tailbond keeps using $primary and switches to the backup by itself when $primary is down. " +
                    "Add this phone to it the same way as to $primary.",
                style = MaterialTheme.typography.bodyMedium,
            )
        }

        // tailnet machines that answered as tailbond servers (probed by the view model)
        val suggestions = if (adding) state.serverSuggestions else emptyList()
        AskCard(state.ask, server, { server = it }, actions, adding, primary, suggestions)
        // after a failed ask the field shows the asked name; the page button uses it too
        val named = server.ifBlank { askedServer(state.ask) }

        // while asking, the other ways would only distract (and "go back to
        // the page" reads like a mistake to someone who just came from it)
        val asking = state.ask is AskState.Asking || state.ask is AskState.Waiting
        if (asking) {
            Text(
                "Other ways to add this phone show up here again if you tap Cancel.",
                style = MaterialTheme.typography.bodySmall,
                color = MaterialTheme.colorScheme.onSurfaceVariant,
            )
            return@Column
        }

        // the server's page opens in a browser; most TVs have none
        if (state.hasBrowser) Card(Modifier.fillMaxWidth()) {
            Column(Modifier.padding(16.dp), verticalArrangement = Arrangement.spacedBy(10.dp)) {
                Text(if (adding) "Or use the other server's page" else "Or use the home server's page", style = MaterialTheme.typography.titleMedium)
                Text(
                    if (adding) "Open the other server's page in your browser and tap Add this phone; it opens tailbond with the pairing."
                    else "Came here from your home server's page? Go back to it in your browser and tap step 2, Open tailbond and ask.",
                    style = MaterialTheme.typography.bodyMedium,
                    fontWeight = FontWeight.Medium,
                )
                OutlinedButton(onClick = { actions.openHomeServerPage(named) }, enabled = named.isNotBlank()) {
                    Icon(Icons.Filled.OpenInBrowser, null)
                    Text("  Open ${named.ifBlank { if (adding) "the other server's" else "home server" }} page")
                }
                if (named.isBlank()) {
                    Text(
                        if (adding) "Type the other server's name in the box above to open its page here."
                        else "Type the home server's name in the box above to open its page here.",
                        style = MaterialTheme.typography.bodySmall,
                        color = MaterialTheme.colorScheme.onSurfaceVariant,
                    )
                }
            }
        }

        // scanning needs a camera: TVs have none (docs/android.md, "Android TV")
        if (state.hasCamera) Card(Modifier.fillMaxWidth()) {
            Column(Modifier.padding(16.dp), verticalArrangement = Arrangement.spacedBy(10.dp)) {
                Text("From another screen", style = MaterialTheme.typography.titleMedium)
                Text(
                    if (adding) "On a computer, open the other server's dashboard (its name and :8077, e.g. http://${named.ifBlank { "vps" }}:8077) → Devices → Add a device with a code, and scan the code it shows."
                    else "On a computer, open the home server's dashboard (its name and :8077, e.g. http://homebox:8077) → Devices → Add a device with a code, and scan the code it shows. (Not the code on the \"Get tailbond\" page: that one only opens the install page.)",
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
                        Text(if (adding) "  Scan code from the other server" else "  Scan code from your home server")
                    }
                }
            }
        }

        Card(Modifier.fillMaxWidth()) {
            Column(Modifier.padding(16.dp), verticalArrangement = Arrangement.spacedBy(10.dp)) {
                Text("Have a link?", style = MaterialTheme.typography.titleMedium)
                if (!state.hasCamera) {
                    Text(
                        "This device has no camera to scan a code with. Easiest: ask your home server above. " +
                            "Or type the pairing link from the home server's dashboard (Devices → Add a device with a code shows it under the code).",
                        style = MaterialTheme.typography.bodyMedium,
                    )
                }
                OutlinedTextField(
                    value = text,
                    onValueChange = { text = it; actions.clearPairError() },
                    label = { Text("Pairing link") },
                    placeholder = { Text("tailbond://pair?d=…") },
                    // the TV's on-screen keyboard has no Add button of ours to reach: its Done adds
                    keyboardOptions = KeyboardOptions(keyboardType = KeyboardType.Uri, imeAction = ImeAction.Done),
                    keyboardActions = KeyboardActions(onDone = { if (text.isNotBlank()) actions.offerPairing(text) }),
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

/** The name a finished ask was for (the field shows it again after a failure). */
fun askedServer(ask: AskState): String = when (ask) {
    is AskState.Failed -> ask.server
    is AskState.Declined -> ask.server
    is AskState.Expired -> ask.server
    else -> ""
}

/** Title and body for an "Ask to connect" state; null when there's nothing to say. */
fun askMessage(s: AskState): Pair<String, String>? = when (s) {
    AskState.Idle -> null
    is AskState.Asking -> "Asking ${s.server}…" to "Looking for ${s.server} on your Tailscale network."
    is AskState.Waiting -> "Waiting for approval on ${s.server}…" to
        "Ask whoever runs ${s.server} to approve \"${s.name}\" on ${s.server}'s tailbond page (it's at the top of http://${s.server}:8077). " +
        (if (s.seenAs.isNotBlank()) "There it shows as ${s.seenAs}, so they can tell it's this phone. " else "") +
        "You can switch to another app to message them; tailbond keeps waiting for 10 minutes and carries on by itself once approved."
    is AskState.Declined -> "${s.server} said no" to "Whoever runs ${s.server} declined this phone. Ask them, then ask again."
    is AskState.Expired -> "Nobody approved it in time" to "The request ran out after 10 minutes. Ask again when someone can approve it on ${s.server}."
    is AskState.Failed -> when (s.failure) {
        PairRequestClient.Failure.NOT_FOUND -> "Can't find ${s.server}" to
            "Check that the Tailscale app says Connected, and that the name matches the one in its list of machines."
        PairRequestClient.Failure.NOT_TAILNET -> "${s.server} isn't on your Tailscale network" to
            "Use the home server's name from the Tailscale app's list of machines."
        PairRequestClient.Failure.UNREACHABLE -> "${s.server} doesn't answer" to
            "Is tailbond running on ${s.server}, and does the Tailscale app on this phone say Connected?"
        PairRequestClient.Failure.NOT_SERVER -> "${s.server} isn't a tailbond home server" to
            "Pick the machine that runs tailbond as the home server."
        PairRequestClient.Failure.REFUSED -> "${s.server} didn't accept the request" to
            "Make sure the Tailscale app on this phone says Connected, then ask again."
        PairRequestClient.Failure.TOO_MANY -> "Too many requests" to "Wait 10 minutes, then ask again."
    }
}

/** First choice while Tailscale still has the VPN: ask the home server, approve it there. */
@Composable
private fun AskCard(
    ask: AskState,
    typed: String,
    onServer: (String) -> Unit,
    actions: UiActions,
    adding: Boolean = false,
    primary: String = "",
    suggestions: List<String> = emptyList(),
) {
    // after a failed try the field keeps (or gets back) the name that was asked
    val server = typed.ifBlank { askedServer(ask) }
    val busy = ask is AskState.Asking || ask is AskState.Waiting
    val msg = askMessage(ask)
    val bad = ask is AskState.Failed || ask is AskState.Declined || ask is AskState.Expired
    Card(Modifier.fillMaxWidth()) {
        Column(Modifier.padding(16.dp), verticalArrangement = Arrangement.spacedBy(10.dp)) {
            Text(if (adding) "Ask the other server" else "Ask your home server", style = MaterialTheme.typography.titleMedium)
            if (!busy) {
                Text(
                    if (adding) "Type the other server's name as it appears in the Tailscale app, and tap Ask to connect. " +
                        "This works while tailbond is on (through $primary) or with Tailscale on. Then whoever runs that server " +
                        "approves this phone at the top of its tailbond page (its name and :8077, e.g. http://vps:8077); tailbond waits and carries on by itself."
                    else "Keep Tailscale on for now. Type your home server's name as it appears in the Tailscale app, " +
                        "and tap Ask to connect. Someone approves this phone on the home server; that's it.",
                    style = MaterialTheme.typography.bodyMedium,
                )
                OutlinedTextField(
                    value = server,
                    onValueChange = { onServer(it.trim()) },
                    label = { Text(if (adding) "Other server's name in Tailscale" else "Home server's name in Tailscale") },
                    placeholder = { Text(if (adding) "e.g. vps" else "e.g. homebox") },
                    singleLine = true,
                    keyboardOptions = KeyboardOptions(keyboardType = KeyboardType.Uri, imeAction = ImeAction.Go),
                    keyboardActions = KeyboardActions(onGo = { if (server.isNotBlank()) actions.askToConnect(server) }),
                    modifier = Modifier.fillMaxWidth().tvInitialFocus(),
                )
                if (suggestions.isNotEmpty()) {
                    Text("Found on your Tailscale (tap to ask):", style = MaterialTheme.typography.labelLarge)
                    FlowRow(horizontalArrangement = Arrangement.spacedBy(8.dp)) {
                        suggestions.forEach { name ->
                            AssistChip(onClick = { onServer(name); actions.askToConnect(name) }, label = { Text(name) })
                        }
                    }
                }
            }
            if (msg != null) {
                Row(verticalAlignment = Alignment.Top) {
                    if (busy) CircularProgressIndicator(Modifier.size(22.dp), strokeWidth = 2.5.dp)
                    else Icon(Icons.Filled.Warning, null, Modifier.size(22.dp), tint = MaterialTheme.colorScheme.error)
                    Spacer(Modifier.width(12.dp))
                    Column {
                        Text(msg.first, style = MaterialTheme.typography.titleSmall, color = if (bad) MaterialTheme.colorScheme.error else MaterialTheme.colorScheme.onSurface)
                        Text(msg.second, style = MaterialTheme.typography.bodyMedium, color = MaterialTheme.colorScheme.onSurfaceVariant)
                    }
                }
            }
            if (busy) {
                OutlinedButton(onClick = actions::cancelAsk, Modifier.tvInitialFocus()) { Text("Cancel") }
            } else {
                Button(onClick = { actions.askToConnect(server) }, enabled = server.isNotBlank()) {
                    Icon(Icons.Filled.Devices, null)
                    Text(if (bad) "  Ask again" else "  Ask to connect")
                }
                Text(
                    "Not sure of the name? It's the machine that runs tailbond, in the Tailscale app's list of machines. " +
                        "Or ask whoever set it up.",
                    style = MaterialTheme.typography.bodySmall,
                    color = MaterialTheme.colorScheme.onSurfaceVariant,
                )
            }
        }
    }
}

/** Confirmation before a scanned/pasted/deep-linked pairing is saved. [target]: null = first pairing. */
@Composable
fun ConfirmPairingDialog(
    p: PairingPayload,
    target: PairTarget?,
    servers: List<ServerProfile>,
    onConfirm: () -> Unit,
    onDismiss: () -> Unit,
) {
    var details by rememberSaveable { mutableStateOf(false) }
    val server = serverShortName(p)
    val replaced = (target as? PairTarget.Replace)?.let { t -> servers.firstOrNull { it.id == t.id } }
    val adding = target == PairTarget.Add
    val renewing = replaced != null && replaced.name.equals(server, ignoreCase = true)
    val replacing = replaced != null && !renewing
    val primary = servers.firstOrNull()?.name ?: "your home server"
    AlertDialog(
        onDismissRequest = onDismiss,
        title = {
            Text(
                when {
                    adding -> "Add $server as a backup server?"
                    renewing -> "Pair again with $server?"
                    replacing -> "Switch this phone to $server?"
                    else -> "Add this phone to $server?"
                },
            )
        },
        text = {
            Column(verticalArrangement = Arrangement.spacedBy(10.dp)) {
                if (adding) {
                    Text("tailbond keeps using $primary and switches to $server by itself when $primary is down, then back to $primary once it works again. You can change the order in Settings.")
                    Text("While $server is in use, your internet traffic goes out from $server instead of $primary, and your Tailscale machines see this phone as $server.")
                } else if (renewing) {
                    Text("This renews this phone's pairing with $server. Nothing else changes.")
                } else {
                    Text("tailbond will connect to your home server $server over Wi-Fi and mobile data. Your Tailscale machines stay reachable through it. They see this phone as $server (not as your own phone), so they allow it whatever they allow $server.")
                }
                Text(
                    if (adding) "It connects over the internet to ${p.server}."
                    else "It connects over the internet to ${p.server}, and your internet traffic goes out from $server (you can change that in Settings).",
                    style = MaterialTheme.typography.bodySmall,
                )
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
                if (replacing) Text("This replaces ${replaced?.name} on this phone.", style = MaterialTheme.typography.bodySmall)
                TextButton(onClick = { details = !details }) { Text(if (details) "Hide details" else "Technical details") }
                if (details) {
                    val small = MaterialTheme.typography.bodySmall
                    Text("Address: ${p.server}", style = small)
                    p.sni?.let { Text("Name: $it", style = small) }
                    Text("Mode: ${p.mode} · DNS: ${TunnelPlan.dns(p).joinToString()} · Tunnel: ${ServerList.subnetOf(p)}", style = small)
                    if (p.tailnetRoutes.isNotEmpty()) Text("Tailnet routes: ${p.tailnetRoutes.joinToString()}", style = small)
                    p.certSha256?.let { Text("Certificate SHA-256: ${it.take(16)}…", style = small) }
                }
            }
        },
        modifier = Modifier.tvFocusRing(),
        confirmButton = {
            TextButton(onClick = onConfirm, Modifier.tvInitialFocus()) {
                Text(
                    when {
                        adding -> "Add backup"
                        renewing -> "Pair again"
                        replacing -> "Switch"
                        else -> "Add phone"
                    },
                )
            }
        },
        dismissButton = { TextButton(onClick = onDismiss) { Text("Cancel") } },
    )
}
