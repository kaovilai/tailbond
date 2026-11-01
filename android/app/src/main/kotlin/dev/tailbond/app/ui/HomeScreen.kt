package dev.tailbond.app.ui

import androidx.compose.foundation.background
import androidx.compose.foundation.clickable
import androidx.compose.foundation.layout.Arrangement
import androidx.compose.foundation.layout.Box
import androidx.compose.foundation.layout.Column
import androidx.compose.foundation.layout.Row
import androidx.compose.foundation.layout.Spacer
import androidx.compose.foundation.layout.fillMaxWidth
import androidx.compose.foundation.layout.height
import androidx.compose.foundation.layout.padding
import androidx.compose.foundation.layout.size
import androidx.compose.foundation.layout.width
import androidx.compose.foundation.lazy.LazyColumn
import androidx.compose.foundation.lazy.items
import androidx.compose.foundation.selection.toggleable
import androidx.compose.foundation.shape.CircleShape
import androidx.compose.material.icons.Icons
import androidx.compose.material.icons.automirrored.filled.KeyboardArrowRight
import androidx.compose.material.icons.filled.Cable
import androidx.compose.material.icons.filled.Computer
import androidx.compose.material.icons.filled.ContentCopy
import androidx.compose.material.icons.filled.DataSaverOn
import androidx.compose.material.icons.filled.Dns
import androidx.compose.material.icons.filled.ErrorOutline
import androidx.compose.material.icons.filled.PhoneAndroid
import androidx.compose.material.icons.filled.Public
import androidx.compose.material.icons.filled.Search
import androidx.compose.material.icons.filled.SignalCellularAlt
import androidx.compose.material.icons.filled.Storage
import androidx.compose.material.icons.filled.SwapHoriz
import androidx.compose.material.icons.filled.Tv
import androidx.compose.material.icons.filled.Wifi
import androidx.compose.material3.Button
import androidx.compose.material3.Card
import androidx.compose.material3.CardDefaults
import androidx.compose.material3.HorizontalDivider
import androidx.compose.material3.Icon
import androidx.compose.material3.IconButton
import androidx.compose.material3.ListItem
import androidx.compose.material3.ListItemDefaults
import androidx.compose.material3.MaterialTheme
import androidx.compose.material3.OutlinedButton
import androidx.compose.material3.OutlinedTextField
import androidx.compose.material3.Switch
import androidx.compose.material3.Text
import androidx.compose.runtime.Composable
import androidx.compose.runtime.getValue
import androidx.compose.runtime.mutableStateOf
import androidx.compose.runtime.saveable.rememberSaveable
import androidx.compose.runtime.setValue
import androidx.compose.ui.Alignment
import androidx.compose.ui.Modifier
import androidx.compose.ui.draw.scale
import androidx.compose.ui.graphics.Color
import androidx.compose.ui.graphics.vector.ImageVector
import androidx.compose.ui.semantics.Role
import androidx.compose.ui.text.font.FontWeight
import androidx.compose.ui.text.style.TextOverflow
import androidx.compose.ui.unit.dp
import dev.tailbond.app.data.PeerNode
import dev.tailbond.app.model.Mode
import dev.tailbond.app.model.PathKind
import dev.tailbond.app.model.Words
import dev.tailbond.app.service.ConnState
import java.time.Duration
import java.time.Instant

val OnlineGreen = Color(0xFF22C55E)

@Composable
fun HomeScreen(
    state: UiState,
    actions: UiActions,
    onConnections: () -> Unit,
    onSettings: () -> Unit,
    onRepair: () -> Unit,
    onMachine: (String) -> Unit,
    modifier: Modifier = Modifier,
    startSearch: String = "",
) {
    var query by rememberSaveable { mutableStateOf(startSearch) }
    val live = state.live
    val connected = live.state == ConnState.CONNECTED || live.state == ConnState.RECONNECTING
    val copy = LocalCopy.current

    LazyColumn(modifier.fillMaxWidth(), contentPadding = androidx.compose.foundation.layout.PaddingValues(bottom = 24.dp)) {
        item { StatusCard(state, actions) }
        if (live.state == ConnState.ERROR) item { ErrorCard(state, actions, onRepair) }
        if (state.wifiSignIn && state.settings.captivePrompts && state.settings.path(PathKind.WIFI).enabled) item { WifiSignInCard(state, actions) }
        if (state.settings.handedToTailscale && live.state == ConnState.DISCONNECTED) {
            item { HandedBackCard(state, actions) }
        } else {
            item {
                OutlinedButton(onClick = actions::switchToTailscale, modifier = Modifier.fillMaxWidth().padding(horizontal = 16.dp)) {
                    Icon(Icons.Filled.SwapHoriz, null)
                    Spacer(Modifier.width(8.dp))
                    Text("Switch back to Tailscale")
                }
            }
        }

        if (state.servers.size > 1) {
            item { SectionHeader("Servers") }
            item { ServersOverview(state, onSettings) }
        }

        item { SectionHeader("Connections") }
        item { ConnectionsSummary(state, onConnections) }
        item {
            val saving = state.mode == Mode.FAILOVER
            ListItem(
                headlineContent = { Text("Save mobile data") },
                supportingContent = { Text("Use mobile data only when Wi-Fi stops working") },
                leadingContent = { Icon(Icons.Filled.DataSaverOn, null) },
                trailingContent = { Switch(checked = saving, onCheckedChange = null) },
                modifier = Modifier.toggleable(
                    value = saving, role = Role.Switch,
                    onValueChange = { actions.setMode(if (it) Mode.FAILOVER else Mode.SPEED) },
                ),
            )
        }

        item { SectionHeader("This device") }
        item { ThisDevice(state) }

        item { SectionHeader("Machines") }
        item {
            OutlinedTextField(
                value = query,
                onValueChange = { query = it },
                placeholder = { Text("Search machines") },
                leadingIcon = { Icon(Icons.Filled.Search, null) },
                singleLine = true,
                shape = MaterialTheme.shapes.extraLarge,
                modifier = Modifier.fillMaxWidth().padding(horizontal = 16.dp, vertical = 4.dp),
            )
        }
        val list = state.peers.list
        val q = query.trim().lowercase()
        val machines = buildList {
            list?.self?.let { add(it to true) }
            list?.peers?.forEach { add(it to false) }
        }.filter { (n, _) ->
            q.isEmpty() || n.hostname.lowercase().contains(q) || n.dnsName.lowercase().contains(q) ||
                n.tailscaleIps.any { it.contains(q) }
        }
        val note = machinesNote(state, connected)
        if (note != null) item { Hint(note) }
        items(machines, key = { it.first.dnsName.ifEmpty { it.first.hostname } }) { (n, isServer) ->
            MachineRow(n, isServer, dim = !connected, onOpen = { onMachine(n.hostname) }, onCopy = { (n.ipv4 ?: n.dnsName).let(copy) })
        }
        if (list != null && machines.isEmpty() && q.isNotEmpty()) item { Hint("No machine matches \"$query\".") }

        item { SectionHeader("Internet") }
        item {
            ListItem(
                headlineContent = { Text("Internet traffic") },
                supportingContent = { Text(Words.routeLabel(state.settings.routeMode, state.servers.size > 1)) },
                leadingContent = { Icon(Icons.Filled.Public, null) },
                trailingContent = { Icon(Icons.AutoMirrored.Filled.KeyboardArrowRight, null) },
                modifier = Modifier.clickable(onClick = onSettings),
            )
        }

    }
}

private fun machinesNote(state: UiState, connected: Boolean): String? {
    val list = state.peers.list
    return when {
        list == null && !connected -> "Connect to see the Tailscale machines you can reach through ${state.serverName}."
        list == null && state.peers.loading -> "Loading machines from ${state.serverName}…"
        list == null -> "Couldn't load the machine list from ${state.serverName}" + (state.peers.error?.let { " ($it)" } ?: "") + "."
        !list.tailscaleRunning -> "${state.serverName} isn't connected to Tailscale right now, so tailnet machines are unreachable."
        state.live.state == ConnState.CONNECTING -> "Connecting… This is the list from ${ago(list.fetchedAtMs)}."
        !connected -> "Not connected. This is the list from ${ago(list.fetchedAtMs)}; connect to reach these machines."
        else -> null
    }
}

private fun ago(ms: Long): String {
    if (ms <= 0) return "earlier"
    val m = Duration.ofMillis(System.currentTimeMillis() - ms).toMinutes()
    return when {
        m < 1 -> "just now"
        m < 60 -> "$m min ago"
        m < 48 * 60 -> "${m / 60} h ago"
        else -> "${m / 1440} days ago"
    }
}

fun lastSeenText(n: PeerNode): String {
    if (n.online) return "Online"
    val t = n.lastSeen?.let { runCatching { Instant.parse(it) }.getOrNull() } ?: return "Offline"
    return "Last seen " + ago(t.toEpochMilli())
}

@Composable
private fun StatusCard(state: UiState, actions: UiActions) {
    val live = state.live
    val on = live.state == ConnState.CONNECTED || live.state == ConnState.CONNECTING || live.state == ConnState.RECONNECTING
    val connected = live.state == ConnState.CONNECTED
    Card(
        Modifier.fillMaxWidth().padding(horizontal = 16.dp, vertical = 8.dp),
        colors = CardDefaults.cardColors(
            containerColor = if (connected) MaterialTheme.colorScheme.primaryContainer else MaterialTheme.colorScheme.surfaceContainerHigh,
        ),
    ) {
        Column(Modifier.padding(20.dp)) {
            Row(verticalAlignment = Alignment.CenterVertically) {
                Column(Modifier.weight(1f)) {
                    Text(Words.stateTitle(live.state), style = MaterialTheme.typography.headlineSmall, fontWeight = FontWeight.SemiBold)
                    Spacer(Modifier.height(4.dp))
                    Text(Words.stateLine(live, state.serverName, state.settings.handedToTailscale), style = MaterialTheme.typography.bodyMedium)
                }
                Spacer(Modifier.width(12.dp))
                Switch(checked = on, onCheckedChange = actions::setOn, modifier = Modifier.scale(1.3f))
            }
            // which server is in use and why, full width under the switch
            val note = if (on) Words.serverNote(live.failover, state.serverNames) else null
            if (note != null) {
                Spacer(Modifier.height(12.dp))
                Row(verticalAlignment = Alignment.Top) {
                    Icon(Icons.Filled.Dns, null, Modifier.size(18.dp).padding(top = 2.dp))
                    Spacer(Modifier.width(8.dp))
                    Column {
                        Text(note.line, style = MaterialTheme.typography.bodyMedium, fontWeight = FontWeight.Medium)
                        note.detail?.let { Text(it, style = MaterialTheme.typography.bodySmall) }
                        note.action?.let { a ->
                            Button(onClick = { actions.useServerNow(a.serverId) }, modifier = Modifier.padding(top = 8.dp)) { Text(a.label) }
                        }
                    }
                }
            }
        }
    }
}

@Composable
private fun ErrorCard(state: UiState, actions: UiActions, onRepair: () -> Unit) {
    val p = state.profile ?: return
    val all = if (state.servers.size > 1 && state.live.failover != null) state.servers.map { it.name } else emptyList()
    val e = Words.error(state.live.errorKind, state.serverName, state.activeServer?.payload?.server ?: p.server, state.live.message, all)
    val run: (Words.Fix) -> Unit = { f ->
        when (f) {
            Words.Fix.RETRY -> actions.retry()
            Words.Fix.REPAIR, Words.Fix.PAIR -> onRepair()
        }
    }
    Card(
        Modifier.fillMaxWidth().padding(horizontal = 16.dp, vertical = 4.dp),
        colors = CardDefaults.cardColors(containerColor = MaterialTheme.colorScheme.errorContainer),
    ) {
        Column(Modifier.padding(16.dp), verticalArrangement = Arrangement.spacedBy(8.dp)) {
            Row(verticalAlignment = Alignment.CenterVertically) {
                Icon(Icons.Filled.ErrorOutline, null, tint = MaterialTheme.colorScheme.onErrorContainer)
                Spacer(Modifier.width(8.dp))
                Text(e.title, style = MaterialTheme.typography.titleMedium, color = MaterialTheme.colorScheme.onErrorContainer)
            }
            Text(e.body, style = MaterialTheme.typography.bodyMedium, color = MaterialTheme.colorScheme.onErrorContainer)
            Row(horizontalArrangement = Arrangement.spacedBy(8.dp)) {
                Button(onClick = { run(e.fix) }) { Text(e.fix.label) }
                e.second?.let { s -> OutlinedButton(onClick = { run(s) }) { Text(s.label) } }
            }
        }
    }
}

/** A Wi-Fi with a sign-in page: what tailbond does meanwhile, and the way to sign in. */
@Composable
private fun WifiSignInCard(state: UiState, actions: UiActions) {
    val on = state.live.state == ConnState.CONNECTED || state.live.state == ConnState.RECONNECTING
    Card(Modifier.fillMaxWidth().padding(horizontal = 16.dp, vertical = 4.dp)) {
        Column(Modifier.padding(16.dp), verticalArrangement = Arrangement.spacedBy(8.dp)) {
            Text("Wi-Fi needs sign-in", style = MaterialTheme.typography.titleMedium)
            Text(
                "This Wi-Fi shows a sign-in page before it lets you online. " +
                    (if (on) "tailbond keeps you connected over mobile data meanwhile, and adds Wi-Fi by itself once you're signed in."
                    else "Sign in first; tailbond uses Wi-Fi once Android says it's online."),
                style = MaterialTheme.typography.bodyMedium,
            )
            Text(
                "Tap the Wi-Fi network (it says \"Sign in to network\") to open the page. Android's sign-in page uses the Wi-Fi directly, so you don't need to turn tailbond off.",
                style = MaterialTheme.typography.bodySmall,
                color = MaterialTheme.colorScheme.onSurfaceVariant,
            )
            Button(onClick = actions::openWifiSignIn) { Text("Sign in to Wi-Fi") }
        }
    }
}

@Composable
private fun HandedBackCard(state: UiState, actions: UiActions) {
    Card(Modifier.fillMaxWidth().padding(horizontal = 16.dp, vertical = 4.dp)) {
        Column(Modifier.padding(16.dp), verticalArrangement = Arrangement.spacedBy(8.dp)) {
            Text("Last step: turn Tailscale on", style = MaterialTheme.typography.titleMedium)
            Text(
                "tailbond is off. Open the Tailscale app and check that it says Connected; if it says " +
                    "Disconnected, tap its switch. Until then, your Tailscale machines aren't reachable.",
                style = MaterialTheme.typography.bodyMedium,
            )
            if (state.tailscaleInstalled) {
                Button(onClick = actions::switchToTailscale) { Text("Open Tailscale") }
            }
        }
    }
}

@Composable
fun SectionHeader(title: String, modifier: Modifier = Modifier) {
    Text(
        title,
        style = MaterialTheme.typography.titleSmall,
        color = MaterialTheme.colorScheme.primary,
        modifier = modifier.padding(start = 16.dp, end = 16.dp, top = 20.dp, bottom = 4.dp),
    )
}

@Composable
fun Hint(text: String) {
    Text(
        text,
        style = MaterialTheme.typography.bodyMedium,
        color = MaterialTheme.colorScheme.onSurfaceVariant,
        modifier = Modifier.padding(horizontal = 16.dp, vertical = 8.dp),
    )
}

fun kindIcon(k: PathKind): ImageVector = when (k) {
    PathKind.WIFI -> Icons.Filled.Wifi
    PathKind.CELLULAR -> Icons.Filled.SignalCellularAlt
    PathKind.ETHERNET -> Icons.Filled.Cable
}

@Composable
fun StatusDot(good: Boolean?, modifier: Modifier = Modifier) {
    val c = when (good) {
        true -> OnlineGreen
        false -> MaterialTheme.colorScheme.error
        null -> MaterialTheme.colorScheme.outline
    }
    Box(modifier.size(10.dp).background(c, CircleShape))
}

/** The two link tiles plus one summary sentence; taps open the Connections page. */
@Composable
private fun ConnectionsSummary(state: UiState, onOpen: () -> Unit) {
    val live = state.live
    val on = live.state == ConnState.CONNECTED || live.state == ConnState.RECONNECTING
    Column(
        Modifier.fillMaxWidth().clickable(onClick = onOpen, role = Role.Button).padding(horizontal = 16.dp, vertical = 4.dp),
        verticalArrangement = Arrangement.spacedBy(8.dp),
    ) {
        Text(
            Words.connectionsSummary(live, state.settings, state.mode, state.capReached, state.wifiSignIn),
            style = MaterialTheme.typography.bodyLarge,
            fontWeight = FontWeight.Medium,
        )
        Row(horizontalArrangement = Arrangement.spacedBy(8.dp)) {
            for (kind in listOf(PathKind.WIFI, PathKind.CELLULAR)) {
                val st = Words.linkStatus(kind, live, state.settings, state.mode, state.capReached, state.wifiSignIn)
                val sample = Words.sampleFor(live, kind)
                Card(Modifier.weight(1f)) {
                    Column(Modifier.padding(12.dp)) {
                        Row(verticalAlignment = Alignment.CenterVertically) {
                            Icon(kindIcon(kind), null, Modifier.size(20.dp))
                            Spacer(Modifier.width(6.dp))
                            Text(kind.label, style = MaterialTheme.typography.titleSmall)
                        }
                        Spacer(Modifier.height(6.dp))
                        Row(verticalAlignment = Alignment.CenterVertically) {
                            StatusDot(if (!on || st == Words.LinkStatus.OFF) null else st.good)
                            Spacer(Modifier.width(6.dp))
                            Text(st.label, style = MaterialTheme.typography.bodySmall, maxLines = 2)
                        }
                        if (on && sample != null && sample.status == dev.tailbond.app.service.PathGate.STATUS_ACTIVE) {
                            Text(
                                formatRatePair(sample.rxBps, sample.txBps),
                                style = MaterialTheme.typography.bodySmall,
                                color = MaterialTheme.colorScheme.onSurfaceVariant,
                            )
                        }
                    }
                }
            }
        }
        Row(verticalAlignment = Alignment.CenterVertically) {
            Text(
                "Mode: ${state.mode.label}",
                style = MaterialTheme.typography.bodyMedium,
                color = MaterialTheme.colorScheme.onSurfaceVariant,
                modifier = Modifier.weight(1f),
            )
            Text("More options", style = MaterialTheme.typography.labelLarge, color = MaterialTheme.colorScheme.primary)
            Icon(Icons.AutoMirrored.Filled.KeyboardArrowRight, null, tint = MaterialTheme.colorScheme.primary)
        }
    }
}

@Composable
private fun ThisDevice(state: UiState) {
    val copy = LocalCopy.current
    val ip = state.live.tunnelIp
    val name = state.profile?.name?.ifBlank { null } ?: state.profile?.user?.ifBlank { null } ?: "This phone"
    ListItem(
        headlineContent = { Text("$name (this phone)", fontWeight = FontWeight.SemiBold) },
        supportingContent = {
            Column {
                Text(
                    ip ?: if (state.live.state == ConnState.CONNECTING || state.live.state == ConnState.RECONNECTING) {
                        "Getting an address…"
                    } else {
                        "No address while disconnected"
                    },
                )
                Text(
                    if (ip != null) "Not a Tailscale address: your Tailscale machines see this phone as ${state.serverName} and allow it whatever they allow ${state.serverName}. " +
                        "They can open connections to the phone at this address only if ${state.serverName}'s tailbond route is approved in the Tailscale admin console (your access rules apply)."
                    else "This phone reaches your tailnet through ${state.serverName}",
                    style = MaterialTheme.typography.bodySmall,
                    color = MaterialTheme.colorScheme.onSurfaceVariant,
                )
            }
        },
        leadingContent = { Icon(Icons.Filled.PhoneAndroid, null) },
        trailingContent = {
            if (ip != null) IconButton(onClick = { copy(ip) }) { Icon(Icons.Filled.ContentCopy, "Copy $ip") }
        },
        modifier = if (ip != null) Modifier.clickable { copy(ip) } else Modifier,
    )
}

fun osIcon(os: String): ImageVector = when (os.lowercase()) {
    "android", "ios" -> Icons.Filled.PhoneAndroid
    "tvos" -> Icons.Filled.Tv
    "linux", "freebsd" -> Icons.Filled.Storage
    else -> Icons.Filled.Computer
}

@Composable
private fun MachineRow(n: PeerNode, isServer: Boolean, dim: Boolean, onOpen: () -> Unit, onCopy: () -> Unit) {
    val muted = MaterialTheme.colorScheme.onSurfaceVariant
    Column {
        ListItem(
            headlineContent = {
                Row(verticalAlignment = Alignment.CenterVertically) {
                    StatusDot(if (n.online && !dim) true else null)
                    Spacer(Modifier.width(8.dp))
                    Text(n.hostname, fontWeight = FontWeight.SemiBold, maxLines = 1, overflow = TextOverflow.Ellipsis)
                    if (isServer) {
                        Spacer(Modifier.width(8.dp))
                        Text(
                            "Home server",
                            style = MaterialTheme.typography.labelSmall,
                            color = MaterialTheme.colorScheme.primary,
                        )
                    }
                }
            },
            supportingContent = {
                Column {
                    Text(n.dnsName, maxLines = 1, overflow = TextOverflow.Ellipsis)
                    Text(
                        listOfNotNull(n.ipv4, n.os.ifBlank { null }, if (n.online) null else lastSeenText(n)).joinToString(" · "),
                        style = MaterialTheme.typography.bodySmall,
                        color = muted,
                    )
                }
            },
            leadingContent = { Icon(if (isServer) Icons.Filled.Dns else osIcon(n.os), null) },
            trailingContent = {
                IconButton(onClick = onCopy) { Icon(Icons.Filled.ContentCopy, "Copy ${n.ipv4 ?: n.dnsName}") }
            },
            colors = ListItemDefaults.colors(),
            modifier = Modifier.clickable(onClick = onOpen),
        )
        HorizontalDivider(Modifier.padding(start = 72.dp))
    }
}

/** Machine details: every address on its own tap-to-copy row (like Tailscale's machine page). */
@Composable
fun MachineScreen(state: UiState, hostname: String, modifier: Modifier = Modifier) {
    val copy = LocalCopy.current
    val list = state.peers.list
    val n = (listOfNotNull(list?.self) + list?.peers.orEmpty()).firstOrNull { it.hostname == hostname }
    LazyColumn(modifier.fillMaxWidth()) {
        if (n == null) {
            item { Hint("This machine isn't in the list any more.") }
            return@LazyColumn
        }
        item {
            ListItem(
                headlineContent = { Text(n.hostname, style = MaterialTheme.typography.titleLarge) },
                supportingContent = {
                    Row(verticalAlignment = Alignment.CenterVertically) {
                        StatusDot(if (n.online) true else null)
                        Spacer(Modifier.width(6.dp))
                        Text(listOf(lastSeenText(n), n.os).filter { it.isNotBlank() }.joinToString(" · "))
                    }
                },
                leadingContent = { Icon(osIcon(n.os), null) },
            )
        }
        item { SectionHeader("Tap an address to copy it") }
        val rows = listOfNotNull(
            n.dnsName.ifBlank { null }?.let { "MagicDNS name" to it },
            n.hostname.lowercase().let { "Short name" to it },
            n.ipv4?.let { "Tailscale IPv4" to it },
            n.ipv6?.let { "Tailscale IPv6" to it },
        )
        items(rows) { (label, value) ->
            ListItem(
                overlineContent = { Text(label) },
                headlineContent = { Text(value) },
                trailingContent = { Icon(Icons.Filled.ContentCopy, "Copy $value") },
                modifier = Modifier.clickable { copy(value) },
            )
        }
        item {
            Hint("Reached through ${state.serverName}. With tailbond on, apps on this phone can use these addresses just like with Tailscale.")
        }
    }
}
