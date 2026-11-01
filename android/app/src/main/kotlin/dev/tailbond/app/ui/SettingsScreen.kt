package dev.tailbond.app.ui

import androidx.compose.material3.Switch
import androidx.compose.material.icons.filled.Wifi
import androidx.compose.foundation.clickable
import androidx.compose.foundation.layout.Arrangement
import androidx.compose.foundation.layout.Column
import androidx.compose.foundation.layout.Row
import androidx.compose.foundation.layout.fillMaxWidth
import androidx.compose.foundation.layout.padding
import androidx.compose.foundation.lazy.LazyColumn
import androidx.compose.foundation.lazy.items
import androidx.compose.foundation.rememberScrollState
import androidx.compose.foundation.selection.selectable
import androidx.compose.foundation.verticalScroll
import androidx.compose.material.icons.Icons
import androidx.compose.material.icons.automirrored.filled.KeyboardArrowRight
import androidx.compose.material.icons.filled.Apps
import androidx.compose.material.icons.filled.Dns
import androidx.compose.material.icons.filled.Search
import androidx.compose.material.icons.filled.SwapHoriz
import androidx.compose.material.icons.filled.VpnKey
import androidx.compose.material3.AlertDialog
import androidx.compose.material3.Checkbox
import androidx.compose.material3.HorizontalDivider
import androidx.compose.material3.Icon
import androidx.compose.material3.ListItem
import androidx.compose.material3.MaterialTheme
import androidx.compose.material3.OutlinedButton
import androidx.compose.material3.OutlinedTextField
import androidx.compose.material3.RadioButton
import androidx.compose.material3.Text
import androidx.compose.material3.TextButton
import androidx.compose.runtime.Composable
import androidx.compose.runtime.LaunchedEffect
import androidx.compose.runtime.getValue
import androidx.compose.runtime.mutableStateOf
import androidx.compose.runtime.remember
import androidx.compose.runtime.saveable.rememberSaveable
import androidx.compose.runtime.setValue
import androidx.compose.ui.Alignment
import androidx.compose.ui.Modifier
import androidx.compose.ui.semantics.Role
import androidx.compose.ui.unit.dp
import androidx.compose.material.icons.filled.Language
import dev.tailbond.app.model.RouteMode
import dev.tailbond.app.model.Words

@Composable
fun SettingsScreen(
    state: UiState,
    actions: UiActions,
    onApps: () -> Unit,
    onWebsites: () -> Unit,
    onRepair: (String) -> Unit,
    onAddServer: () -> Unit,
    modifier: Modifier = Modifier,
) {
    val muted = MaterialTheme.colorScheme.onSurfaceVariant
    Column(modifier.verticalScroll(rememberScrollState()).padding(bottom = 24.dp)) {
        if (state.servers.isNotEmpty()) ServersSection(state, actions, onRepair, onAddServer)

        SectionHeader("Internet traffic")
        RouteMode.entries.forEach { r ->
            Row(
                Modifier.fillMaxWidth()
                    .selectable(selected = state.settings.routeMode == r, onClick = { actions.setRouteMode(r) }, role = Role.RadioButton)
                    .padding(horizontal = 8.dp, vertical = 4.dp),
                verticalAlignment = Alignment.CenterVertically,
            ) {
                RadioButton(selected = state.settings.routeMode == r, onClick = null, modifier = Modifier.padding(8.dp))
                Column {
                    Text(Words.routeLabel(r, state.servers.size > 1))
                    Text(Words.routeExplain(r, state.servers.map { it.name }), style = MaterialTheme.typography.bodySmall, color = muted)
                }
            }
        }

        HorizontalDivider(Modifier.padding(top = 8.dp))
        ListItem(
            headlineContent = { Text("Tell me when Wi-Fi needs a sign-in") },
            supportingContent = {
                Text("Hotel and airport Wi-Fi often show a sign-in page first. tailbond never bonds over a Wi-Fi until Android says it's online.")
            },
            leadingContent = { Icon(Icons.Filled.Wifi, null) },
            trailingContent = { Switch(checked = state.settings.captivePrompts, onCheckedChange = actions::setCaptivePrompts) },
        )
        ListItem(
            headlineContent = { Text("Apps that skip tailbond") },
            supportingContent = {
                Text(
                    if (state.settings.excludedApps.isEmpty()) "None. Every app goes through tailbond"
                    else state.settings.excludedApps.size.let { n -> if (n == 1) "1 app uses the phone's own connection" else "$n apps use the phone's own connection" },
                )
            },
            leadingContent = { Icon(Icons.Filled.Apps, null) },
            trailingContent = { Icon(Icons.AutoMirrored.Filled.KeyboardArrowRight, null) },
            modifier = Modifier.clickable(onClick = onApps),
        )
        ListItem(
            headlineContent = { Text("Websites that skip tailbond") },
            supportingContent = { Text(Words.websitesSummary(state.settings, state.sdkInt)) },
            leadingContent = { Icon(Icons.Filled.Language, null) },
            trailingContent = { Icon(Icons.AutoMirrored.Filled.KeyboardArrowRight, null) },
            modifier = Modifier.clickable(onClick = onWebsites),
        )
        ListItem(
            headlineContent = { Text("Switch back to Tailscale") },
            supportingContent = { Text(if (state.tailscaleInstalled) "Turns tailbond off and opens the Tailscale app. Turn Tailscale on there if it says Disconnected" else "Turns tailbond off. Tailscale isn't installed: opens its store page") },
            leadingContent = { Icon(Icons.Filled.SwapHoriz, null) },
            modifier = Modifier.clickable(onClick = actions::switchToTailscale),
        )

        SectionHeader("Keep tailbond on all the time (optional)")
        Column(Modifier.padding(horizontal = 16.dp), verticalArrangement = Arrangement.spacedBy(8.dp)) {
            Text(
                "To keep tailbond on all the time, turn on Always-on VPN for tailbond in Android's VPN settings. " +
                    "Turn off Always-on for Tailscale first: only one app can have it.",
                style = MaterialTheme.typography.bodyMedium,
            )
            Text(
                "Leave \"Block connections without VPN\" off if you want to switch back to Tailscale easily.",
                style = MaterialTheme.typography.bodySmall,
                color = muted,
            )
            OutlinedButton(onClick = actions::openVpnSettings) {
                Icon(Icons.Filled.VpnKey, null)
                Text("  Open VPN settings")
            }
        }

    }

}

/** Split tunnel: apps that bypass tailbond (VpnService addDisallowedApplication). */
@Composable
fun AppsScreen(state: UiState, actions: UiActions, modifier: Modifier = Modifier) {
    LaunchedEffect(Unit) { actions.loadApps() }
    var query by rememberSaveable { mutableStateOf("") }
    val excluded = state.settings.excludedApps
    val q = query.trim().lowercase()
    val shown = state.apps
        .filter { q.isEmpty() || it.label.lowercase().contains(q) || it.pkg.contains(q) }
        .sortedBy { it.pkg !in excluded }
    LazyColumn(modifier.fillMaxWidth()) {
        item {
            Hint("Checked apps use the phone's own connection instead of tailbond. They can't reach your Tailscale machines. Changing this reconnects for a moment.")
        }
        item {
            OutlinedTextField(
                value = query,
                onValueChange = { query = it },
                placeholder = { Text("Search apps") },
                leadingIcon = { Icon(Icons.Filled.Search, null) },
                singleLine = true,
                shape = MaterialTheme.shapes.extraLarge,
                modifier = Modifier.fillMaxWidth().padding(horizontal = 16.dp, vertical = 4.dp),
            )
        }
        if (state.apps.isEmpty()) item { Hint("Loading apps…") }
        items(shown, key = { it.pkg }) { a ->
            val checked = a.pkg in excluded
            ListItem(
                headlineContent = { Text(a.label) },
                supportingContent = { Text(a.pkg, style = MaterialTheme.typography.bodySmall) },
                trailingContent = { Checkbox(checked = checked, onCheckedChange = null) },
                modifier = Modifier.selectable(selected = checked, role = Role.Checkbox, onClick = { actions.setExcluded(a.pkg, !checked) }),
            )
        }
    }
}
