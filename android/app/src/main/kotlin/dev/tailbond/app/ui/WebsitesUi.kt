package dev.tailbond.app.ui

import androidx.compose.foundation.background
import androidx.compose.foundation.layout.Arrangement
import androidx.compose.foundation.layout.Box
import androidx.compose.foundation.layout.FlowRow
import androidx.compose.foundation.layout.fillMaxSize
import androidx.compose.foundation.layout.Column
import androidx.compose.foundation.layout.Row
import androidx.compose.foundation.layout.fillMaxWidth
import androidx.compose.foundation.layout.padding
import androidx.compose.foundation.lazy.LazyColumn
import androidx.compose.foundation.lazy.items
import androidx.compose.foundation.text.KeyboardOptions
import androidx.compose.material.icons.Icons
import androidx.compose.material.icons.filled.Add
import androidx.compose.material.icons.filled.Apps
import androidx.compose.material.icons.filled.Delete
import androidx.compose.material.icons.filled.Language
import androidx.compose.material3.AlertDialog
import androidx.compose.material3.Button
import androidx.compose.material3.Card
import androidx.compose.material3.CardDefaults
import androidx.compose.material3.FilterChip
import androidx.compose.material3.Icon
import androidx.compose.material3.IconButton
import androidx.compose.material3.ListItem
import androidx.compose.material3.MaterialTheme
import androidx.compose.material3.OutlinedButton
import androidx.compose.material3.OutlinedTextField
import androidx.compose.material3.ProvideTextStyle
import androidx.compose.material3.Switch
import androidx.compose.material3.Text
import androidx.compose.material3.TextButton
import androidx.compose.runtime.Composable
import androidx.compose.runtime.getValue
import androidx.compose.runtime.mutableStateOf
import androidx.compose.runtime.saveable.rememberSaveable
import androidx.compose.runtime.setValue
import androidx.compose.ui.Alignment
import androidx.compose.ui.Modifier
import androidx.compose.ui.graphics.Color
import androidx.compose.ui.text.input.ImeAction
import androidx.compose.ui.text.input.KeyboardType
import androidx.compose.ui.unit.dp
import dev.tailbond.app.model.RouteMode
import dev.tailbond.app.model.WebsiteBypass
import dev.tailbond.app.model.Words

/**
 * Websites that skip the tunnel (Android 13+: excludeRoute for their
 * addresses). [startDialog] ("add-website[:<preset title>]") is for renders.
 */
@Composable
fun WebsitesScreen(state: UiState, actions: UiActions, onApps: () -> Unit, modifier: Modifier = Modifier, startDialog: String? = null) {
    val supported = WebsiteBypass.supported(state.sdkInt)
    val initial = startDialog?.takeIf { it.startsWith("add-website") }
    var adding by rememberSaveable { mutableStateOf(initial != null) }
    val sites = state.settings.websites
    Box(modifier.fillMaxSize()) {
    LazyColumn(Modifier.fillMaxWidth()) {
        item {
            Hint(
                "These websites use the phone's own connection instead of tailbond, for example a bank that blocks logins " +
                    "from another place, or a site that should see where you really are.",
            )
        }
        if (!supported) {
            item {
                Card(
                    Modifier.fillMaxWidth().padding(horizontal = 16.dp, vertical = 4.dp),
                    colors = CardDefaults.cardColors(containerColor = MaterialTheme.colorScheme.surfaceContainerHigh),
                ) {
                    Column(Modifier.padding(16.dp), verticalArrangement = Arrangement.spacedBy(8.dp)) {
                        Text("Needs Android 13 or newer", style = MaterialTheme.typography.titleMedium)
                        Text(
                            "This phone runs an older Android, which can't let single websites skip a VPN. " +
                                "Use Apps that skip tailbond instead: pick the app (your bank's app, or the browser you use for it).",
                            style = MaterialTheme.typography.bodyMedium,
                        )
                        OutlinedButton(onClick = onApps) {
                            Icon(Icons.Filled.Apps, null)
                            Text("  Apps that skip tailbond")
                        }
                    }
                }
            }
            return@LazyColumn
        }
        if (state.settings.routeMode == RouteMode.TAILNET_LAN) {
            item {
                Hint("Right now websites already use the phone's own connection (Internet traffic: ${RouteMode.TAILNET_LAN.label}). This list applies when traffic goes through your server.")
            }
        }
        if (sites.isEmpty()) item { Hint("No websites yet.") }
        items(sites, key = { it.domain }) { w ->
            ListItem(
                headlineContent = { Text(w.domain) },
                supportingContent = {
                    Text(listOfNotNull(w.note.ifBlank { null }, "also www.${w.domain}").joinToString(" · "))
                },
                leadingContent = { Icon(Icons.Filled.Language, null) },
                trailingContent = {
                    Row {
                        Switch(checked = w.enabled, onCheckedChange = { actions.setWebsiteEnabled(w.domain, it) })
                        IconButton(onClick = { actions.removeWebsite(w.domain) }) { Icon(Icons.Filled.Delete, "Remove ${w.domain}") }
                    }
                },
            )
        }
        item {
            Button(
                onClick = { adding = true },
                enabled = sites.size < WebsiteBypass.MAX_RULES,
                modifier = Modifier.padding(horizontal = 16.dp, vertical = 8.dp),
            ) {
                Icon(Icons.Filled.Add, null)
                Text("  Add a website")
            }
        }
        item {
            Text(
                "Changes apply within a few seconds; new addresses for a website reconnect tailbond for a moment.",
                style = MaterialTheme.typography.bodySmall,
                color = MaterialTheme.colorScheme.onSurfaceVariant,
                modifier = Modifier.padding(horizontal = 16.dp, vertical = 4.dp),
            )
        }
        item {
            Text(
                Words.WEBSITES_LIMIT,
                style = MaterialTheme.typography.bodySmall,
                color = MaterialTheme.colorScheme.onSurfaceVariant,
                modifier = Modifier.padding(horizontal = 16.dp, vertical = 4.dp),
            )
        }
    }
    if (adding && supported) {
        AddWebsiteDialog(
            initialPreset = initial?.substringAfter(':', "")?.ifEmpty { null },
            onAdd = { domains, note -> domains.forEach { actions.addWebsite(it, note) }; adding = false },
            onDismiss = { adding = false },
            inline = initial != null,
        )
    }
    }
}

/**
 * Type a website, or start from a preset. Adds every domain of a preset at once.
 * [inline] draws it over the page instead of in a dialog window (screen renders:
 * Robolectric never idles with a text field in a dialog window).
 */
@Composable
fun AddWebsiteDialog(initialPreset: String? = null, onAdd: (List<String>, String) -> Unit, onDismiss: () -> Unit, inline: Boolean = false) {
    var text by rememberSaveable { mutableStateOf("") }
    var presetTitle by rememberSaveable { mutableStateOf(initialPreset) }
    val preset = WebsiteBypass.PRESETS.firstOrNull { it.title == presetTitle }
    val typed = WebsiteBypass.normalize(text)
    val domains = buildList {
        preset?.domains?.let { addAll(it) }
        typed?.let { add(it) }
    }.distinct()
    val bad = text.isNotBlank() && typed == null
    val title: @Composable () -> Unit = { Text("Add a website") }
    val body: @Composable () -> Unit = {
        Column(verticalArrangement = Arrangement.spacedBy(10.dp)) {
            OutlinedTextField(
                value = text,
                onValueChange = { text = it },
                label = { Text("Website") },
                placeholder = { Text("e.g. mybank.com") },
                singleLine = true,
                isError = bad,
                supportingText = {
                    Text(if (bad) "Type a website name like mybank.com" else "Its www. name is included")
                },
                keyboardOptions = KeyboardOptions(keyboardType = KeyboardType.Uri, imeAction = ImeAction.Done),
                modifier = Modifier.fillMaxWidth(),
            )
            Text("Or pick one:", style = MaterialTheme.typography.labelLarge)
            FlowRow(horizontalArrangement = Arrangement.spacedBy(8.dp)) {
                WebsiteBypass.PRESETS.forEach { p ->
                    FilterChip(
                        selected = presetTitle == p.title,
                        onClick = { presetTitle = if (presetTitle == p.title) null else p.title },
                        label = { Text(p.title) },
                    )
                }
            }
            preset?.let { p ->
                Text(p.explain, style = MaterialTheme.typography.bodyMedium)
                if (p.domains.isNotEmpty()) {
                    Text("Adds ${p.domains.joinToString()}", style = MaterialTheme.typography.bodySmall, color = MaterialTheme.colorScheme.onSurfaceVariant)
                }
            }
        }
    }
    val confirm: @Composable () -> Unit = {
        TextButton(onClick = { onAdd(domains, preset?.note ?: "") }, enabled = domains.isNotEmpty() && !bad) {
            Text(if (domains.size > 1) "Add ${domains.size} websites" else "Add")
        }
    }
    val dismiss: @Composable () -> Unit = { TextButton(onClick = onDismiss) { Text("Cancel") } }
    if (!inline) {
        AlertDialog(onDismissRequest = onDismiss, title = title, text = body, confirmButton = confirm, dismissButton = dismiss)
        return
    }
    Box(Modifier.fillMaxSize().background(Color.Black.copy(alpha = 0.32f)), contentAlignment = Alignment.Center) {
        Card(
            Modifier.fillMaxWidth().padding(horizontal = 24.dp),
            shape = MaterialTheme.shapes.extraLarge,
            colors = CardDefaults.cardColors(containerColor = MaterialTheme.colorScheme.surfaceContainerHigh),
        ) {
            Column(Modifier.padding(24.dp), verticalArrangement = Arrangement.spacedBy(16.dp)) {
                ProvideTextStyle(MaterialTheme.typography.headlineSmall) { title() }
                ProvideTextStyle(MaterialTheme.typography.bodyMedium) { body() }
                Row(Modifier.fillMaxWidth(), horizontalArrangement = Arrangement.End) { dismiss(); confirm() }
            }
        }
    }
}
