package dev.tailbond.app.ui

import androidx.compose.foundation.clickable
import androidx.compose.foundation.layout.Arrangement
import androidx.compose.foundation.layout.Box
import androidx.compose.foundation.layout.Column
import androidx.compose.foundation.layout.FlowRow
import androidx.compose.foundation.layout.Row
import androidx.compose.foundation.layout.Spacer
import androidx.compose.foundation.layout.fillMaxWidth
import androidx.compose.foundation.layout.padding
import androidx.compose.foundation.layout.width
import androidx.compose.foundation.selection.selectable
import androidx.compose.material.icons.Icons
import androidx.compose.material.icons.automirrored.filled.KeyboardArrowRight
import androidx.compose.material.icons.filled.Add
import androidx.compose.material.icons.filled.ArrowDownward
import androidx.compose.material.icons.filled.ArrowUpward
import androidx.compose.material.icons.filled.Dns
import androidx.compose.material.icons.filled.MoreVert
import androidx.compose.material3.AlertDialog
import androidx.compose.material3.Card
import androidx.compose.material3.DropdownMenu
import androidx.compose.material3.DropdownMenuItem
import androidx.compose.material3.Icon
import androidx.compose.material3.IconButton
import androidx.compose.material3.ListItem
import androidx.compose.material3.MaterialTheme
import androidx.compose.material3.OutlinedButton
import androidx.compose.material3.RadioButton
import androidx.compose.material3.Text
import androidx.compose.material3.TextButton
import androidx.compose.runtime.Composable
import androidx.compose.runtime.getValue
import androidx.compose.runtime.mutableStateOf
import androidx.compose.runtime.remember
import androidx.compose.runtime.setValue
import androidx.compose.ui.Alignment
import androidx.compose.ui.Modifier
import androidx.compose.ui.semantics.Role
import androidx.compose.ui.text.font.FontWeight
import androidx.compose.ui.text.style.TextOverflow
import androidx.compose.ui.unit.dp
import dev.tailbond.app.failover.FailoverPolicy
import dev.tailbond.app.failover.FailoverReason
import dev.tailbond.app.model.SwitchLog
import dev.tailbond.app.failover.ServerState
import dev.tailbond.app.model.ServerProfile
import dev.tailbond.app.model.Words
import dev.tailbond.app.service.ConnState

private fun UiState.tunnelOn(): Boolean =
    live.state == ConnState.CONNECTED || live.state == ConnState.CONNECTING || live.state == ConnState.RECONNECTING

/** Home screen: one row per server with its state (shown only with more than one server). */
@Composable
fun ServersOverview(state: UiState, onOpen: () -> Unit) {
    val on = state.tunnelOn()
    Column(Modifier.fillMaxWidth().clickable(onClick = onOpen, role = Role.Button)) {
        state.servers.forEachIndexed { i, s ->
            val (label, good) = Words.serverStatus(s.id, state.live.failover, on, i == 0)
            ListItem(
                headlineContent = {
                    Row(verticalAlignment = Alignment.CenterVertically) {
                        Text(s.name, fontWeight = FontWeight.SemiBold, maxLines = 1, overflow = TextOverflow.Ellipsis)
                        Spacer(Modifier.width(8.dp))
                        Text(
                            if (i == 0) "Main" else "Backup",
                            style = MaterialTheme.typography.labelSmall,
                            color = MaterialTheme.colorScheme.primary,
                        )
                    }
                },
                supportingContent = {
                    Column {
                        Row(verticalAlignment = Alignment.CenterVertically) {
                            StatusDot(good)
                            Spacer(Modifier.width(6.dp))
                            Text(label)
                        }
                        if (on && state.live.failover?.state(s.id) == ServerState.UNREACHABLE) {
                            Text(
                                "tailbond can't reach it (Tailscale may still show it online below)",
                                style = MaterialTheme.typography.bodySmall,
                                color = MaterialTheme.colorScheme.onSurfaceVariant,
                            )
                        }
                    }
                },
                leadingContent = { Icon(Icons.Filled.Dns, null) },
                trailingContent = if (i == 0) ({ Icon(Icons.AutoMirrored.Filled.KeyboardArrowRight, null) }) else null,
            )
        }
    }
}

/** Settings: the ordered server list, the switching policy and "Add another server". */
@Composable
fun ServersSection(state: UiState, actions: UiActions, onRepair: (String) -> Unit, onAddServer: () -> Unit) {
    val servers = state.servers
    val on = state.tunnelOn()
    val muted = MaterialTheme.colorScheme.onSurfaceVariant
    var removing by remember { mutableStateOf<ServerProfile?>(null) }

    SectionHeader(if (servers.size > 1) "Servers" else "Home server")
    if (servers.size > 1) {
        Hint("tailbond uses one server at a time, from the top of this list. The first one is your main server.")
    }
    servers.forEachIndexed { i, s ->
        ServerCard(
            s, index = i, count = servers.size, state = state, on = on,
            onUp = { actions.moveServer(s.id, -1) },
            onDown = { actions.moveServer(s.id, 1) },
            onUseNow = { actions.useServerNow(s.id) },
            onRepair = { onRepair(s.id) },
            onRemove = { removing = s },
        )
    }
    Column(Modifier.padding(horizontal = 16.dp, vertical = 4.dp), verticalArrangement = Arrangement.spacedBy(4.dp)) {
        OutlinedButton(onClick = onAddServer) {
            Icon(Icons.Filled.Add, null)
            Text(if (servers.size > 1) "  Add another server" else "  Add a backup server")
        }
        if (servers.size == 1) {
            Text(
                "A second tailbond server (a VPS, a friend's home) takes over when ${servers.first().name} is down.",
                style = MaterialTheme.typography.bodySmall, color = muted,
            )
        }
    }

    if (servers.size > 1) {
        SectionHeader("When to switch")
        val names = servers.map { it.name }
        val s = state.settings
        PolicyRow(
            Words.preferLabel(names), Words.preferExplain(names.first(), s),
            selected = s.failoverPolicy == FailoverPolicy.PREFER,
        ) { actions.setFailoverPolicy(FailoverPolicy.PREFER, null) }
        val f = state.live.failover
        if (on && s.failoverPolicy == FailoverPolicy.PREFER && f?.reason == FailoverReason.MANUAL) {
            val picked = state.serverNames[f.activeId] ?: "a server"
            Row(Modifier.padding(start = 56.dp, end = 16.dp), verticalAlignment = Alignment.CenterVertically) {
                Text("Paused: you picked $picked", style = MaterialTheme.typography.bodyMedium, fontWeight = FontWeight.Medium, modifier = Modifier.weight(1f))
                TextButton(onClick = actions::resumeAutomatic) { Text("Go back to automatic") }
            }
        }
        servers.forEach { srv ->
            val selected = s.failoverPolicy == FailoverPolicy.ALWAYS && (s.alwaysServerId ?: servers.first().id) == srv.id
            PolicyRow("Always use ${srv.name}", "Never switches by itself.", selected) {
                actions.setFailoverPolicy(FailoverPolicy.ALWAYS, srv.id)
            }
        }
        Text(
            Words.HOW_IT_DECIDES,
            style = MaterialTheme.typography.bodySmall,
            color = muted,
            modifier = Modifier.padding(horizontal = 16.dp, vertical = 4.dp),
        )

        SectionHeader("Recent switches")
        if (state.switches.isEmpty()) {
            Hint("None yet.")
        } else {
            state.switches.take(SwitchLog.MAX).forEach { r ->
                Row(Modifier.padding(horizontal = 16.dp, vertical = 3.dp)) {
                    Text(Words.dayTime(r.atMs), style = MaterialTheme.typography.bodyMedium, fontWeight = FontWeight.Medium, modifier = Modifier.width(96.dp))
                    Text(Words.switchText(r), style = MaterialTheme.typography.bodyMedium)
                }
            }
        }
    }

    removing?.let { s ->
        val last = servers.size <= 1
        AlertDialog(
            onDismissRequest = { removing = null },
            title = { Text("Remove ${s.name} from this phone?") },
            text = {
                Text(
                    if (last) "tailbond turns off and forgets this home server. To use it again, add this phone from the home server's page."
                    else "tailbond forgets ${s.name}. If it's in use now, tailbond switches to ${servers.first { it.id != s.id }.name}.",
                )
            },
            confirmButton = { TextButton(onClick = { removing = null; actions.removeServer(s.id) }) { Text("Remove", color = MaterialTheme.colorScheme.error) } },
            dismissButton = { TextButton(onClick = { removing = null }) { Text("Cancel") } },
        )
    }
}

@Composable
private fun PolicyRow(label: String, explain: String, selected: Boolean, onClick: () -> Unit) {
    Row(
        Modifier.fillMaxWidth().selectable(selected = selected, onClick = onClick, role = Role.RadioButton)
            .padding(horizontal = 8.dp, vertical = 4.dp),
        verticalAlignment = Alignment.CenterVertically,
    ) {
        RadioButton(selected = selected, onClick = null, modifier = Modifier.padding(8.dp))
        Column {
            Text(label)
            Text(explain, style = MaterialTheme.typography.bodySmall, color = MaterialTheme.colorScheme.onSurfaceVariant)
        }
    }
}

@Composable
private fun ServerCard(
    s: ServerProfile,
    index: Int,
    count: Int,
    state: UiState,
    on: Boolean,
    onUp: () -> Unit,
    onDown: () -> Unit,
    onUseNow: () -> Unit,
    onRepair: () -> Unit,
    onRemove: () -> Unit,
) {
    val muted = MaterialTheme.colorScheme.onSurfaceVariant
    val (label, good) = Words.serverStatus(s.id, state.live.failover, on, index == 0)
    val active = state.live.failover?.activeId == s.id || (count == 1 && on)
    var menu by remember { mutableStateOf(false) }
    val p = s.payload
    Card(Modifier.fillMaxWidth().padding(horizontal = 16.dp, vertical = 4.dp)) {
        Column(Modifier.padding(start = 16.dp, top = 12.dp, end = 4.dp, bottom = 4.dp)) {
            Row(verticalAlignment = Alignment.CenterVertically) {
                Column(Modifier.weight(1f)) {
                    Row(verticalAlignment = Alignment.CenterVertically) {
                        Text(s.name, style = MaterialTheme.typography.titleMedium, maxLines = 1, overflow = TextOverflow.Ellipsis)
                        if (count > 1) {
                            Spacer(Modifier.width(8.dp))
                            Text(
                                if (index == 0) "Main" else "Backup $index",
                                style = MaterialTheme.typography.labelSmall,
                                color = MaterialTheme.colorScheme.primary,
                            )
                        }
                    }
                    if (count > 1) {
                        Row(verticalAlignment = Alignment.CenterVertically) {
                            StatusDot(good)
                            Spacer(Modifier.width(6.dp))
                            Text(label, style = MaterialTheme.typography.bodyMedium)
                        }
                    }
                    Text(
                        "Reached at ${p.server} · this phone is \"${p.name.ifBlank { p.user }}\"",
                        style = MaterialTheme.typography.bodySmall, color = muted,
                    )
                }
                if (count > 1) Box {
                    IconButton(onClick = { menu = true }) { Icon(Icons.Filled.MoreVert, "More for ${s.name}") }
                    DropdownMenu(expanded = menu, onDismissRequest = { menu = false }) {
                        DropdownMenuItem(text = { Text("Pair again") }, onClick = { menu = false; onRepair() })
                        DropdownMenuItem(
                            text = { Text("Remove from this phone", color = MaterialTheme.colorScheme.error) },
                            onClick = { menu = false; onRemove() },
                        )
                    }
                }
            }
            if (count > 1) {
                Row(verticalAlignment = Alignment.CenterVertically) {
                    IconButton(onClick = onUp, enabled = index > 0) { Icon(Icons.Filled.ArrowUpward, "Move ${s.name} up") }
                    IconButton(onClick = onDown, enabled = index < count - 1) { Icon(Icons.Filled.ArrowDownward, "Move ${s.name} down") }
                    Spacer(Modifier.weight(1f))
                    // not offered for a server that doesn't answer: it would only drop the connection
                    val unreachable = state.live.failover?.state(s.id) == ServerState.UNREACHABLE
                    if (on && !active && !unreachable) TextButton(onClick = onUseNow) { Text("Use this server now") }
                }
            } else {
                FlowRow(horizontalArrangement = Arrangement.spacedBy(8.dp), modifier = Modifier.padding(bottom = 4.dp)) {
                    OutlinedButton(onClick = onRepair) { Text("Pair again") }
                    TextButton(onClick = onRemove) { Text("Remove from this phone", color = MaterialTheme.colorScheme.error) }
                }
            }
        }
    }
}
