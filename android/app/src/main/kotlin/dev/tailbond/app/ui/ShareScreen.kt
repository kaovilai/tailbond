package dev.tailbond.app.ui

import androidx.compose.foundation.layout.Arrangement
import androidx.compose.foundation.layout.Column
import androidx.compose.foundation.layout.ExperimentalLayoutApi
import androidx.compose.foundation.layout.FlowRow
import androidx.compose.foundation.layout.Row
import androidx.compose.foundation.layout.fillMaxWidth
import androidx.compose.foundation.layout.padding
import androidx.compose.foundation.rememberScrollState
import androidx.compose.foundation.verticalScroll
import androidx.compose.material.icons.Icons
import androidx.compose.material.icons.filled.Check
import androidx.compose.material.icons.filled.Close
import androidx.compose.material.icons.filled.Router
import androidx.compose.material3.Card
import androidx.compose.material3.CardDefaults
import androidx.compose.material3.FilterChip
import androidx.compose.material3.Icon
import androidx.compose.material3.InputChip
import androidx.compose.material3.ListItem
import androidx.compose.material3.MaterialTheme
import androidx.compose.material3.OutlinedTextField
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
import androidx.compose.ui.text.font.FontWeight
import androidx.compose.ui.unit.dp
import androidx.compose.material.icons.filled.Laptop
import dev.tailbond.app.share.HubStatus
import dev.tailbond.app.share.HubWords
import dev.tailbond.app.share.ShareStatus
import dev.tailbond.app.share.ShareWords

/**
 * "Share my mobile data" (Pair & Share): this phone lends its mobile data to
 * the travel router or a laptop on the same Wi-Fi, which bonds it with its
 * other connections.
 */
@OptIn(ExperimentalLayoutApi::class)
@Composable
fun ShareScreen(state: UiState, actions: UiActions, modifier: Modifier = Modifier) {
    val l = state.share
    val s = state.settings
    val on = l.status != ShareStatus.OFF
    val server = state.serverName
    val muted = MaterialTheme.colorScheme.onSurfaceVariant
    Column(modifier.fillMaxWidth().verticalScroll(rememberScrollState())) {
        Card(
            Modifier.fillMaxWidth().padding(16.dp),
            colors = CardDefaults.cardColors(
                containerColor = if (l.status == ShareStatus.SHARING) MaterialTheme.colorScheme.primaryContainer else MaterialTheme.colorScheme.surfaceVariant,
            ),
        ) {
            Column(Modifier.padding(20.dp), verticalArrangement = Arrangement.spacedBy(8.dp)) {
                Row(verticalAlignment = Alignment.CenterVertically) {
                    Text("Share my mobile data", style = MaterialTheme.typography.headlineSmall, fontWeight = FontWeight.SemiBold, modifier = Modifier.weight(1f))
                    Switch(checked = on, onCheckedChange = { if (it) actions.startShare() else actions.stopShare() })
                }
                Text(ShareWords.headline(l), style = MaterialTheme.typography.titleMedium)
                Text(ShareWords.detail(l, s.shareMinBattery), style = MaterialTheme.typography.bodyMedium)
            }
        }
        Text(
            "Your travel router (or a laptop) paired with $server adds this phone's mobile data to its connection, " +
                "like a second SIM. Connect this phone to the travel router's own Wi-Fi, not the hotel's. " +
                "You don't need to turn tailbond on for this, and this phone's own apps aren't affected.",
            style = MaterialTheme.typography.bodyMedium, color = muted,
            modifier = Modifier.padding(horizontal = 16.dp),
        )

        HubCard(state, actions)
        NearbySharesSection(state, actions)

        SectionHeader("Who's using it")
        if (l.hubs.isEmpty()) {
            Text(
                if (on) "Nobody yet." else "Turn sharing on to see who uses it.",
                style = MaterialTheme.typography.bodyMedium, color = muted, modifier = Modifier.padding(horizontal = 16.dp),
            )
        }
        for (h in l.hubs) {
            ListItem(
                headlineContent = { Text(h.hub) },
                supportingContent = { Text("${formatBytes(h.upBytes + h.downBytes)} since it started") },
                leadingContent = { Icon(Icons.Filled.Router, null, tint = LocalBrand.current.shared) },
            )
        }

        SectionHeader("Stop when the battery is low")
        Text(
            "Unless the phone is charging.", style = MaterialTheme.typography.bodySmall, color = muted,
            modifier = Modifier.padding(horizontal = 16.dp),
        )
        FlowRow(Modifier.padding(horizontal = 16.dp), horizontalArrangement = Arrangement.spacedBy(8.dp)) {
            for (pct in listOf(0, 10, 20, 30, 50)) {
                FilterChip(
                    selected = s.shareMinBattery == pct, onClick = { actions.setShareMinBattery(pct) },
                    label = { Text(if (pct == 0) "Never" else "Below $pct%") },
                    leadingIcon = if (s.shareMinBattery == pct) ({ Icon(Icons.Filled.Check, null) }) else null,
                )
            }
        }

        SectionHeader("Monthly limit")
        Text(
            "${formatBytes(l.usedBytes)} shared this month." + if (s.shareCapMb > 0) " Sharing stops at ${formatBytes(s.shareCapMb * 1_000_000)}." else "",
            style = MaterialTheme.typography.bodySmall, color = muted, modifier = Modifier.padding(horizontal = 16.dp),
        )
        FlowRow(Modifier.padding(horizontal = 16.dp), horizontalArrangement = Arrangement.spacedBy(8.dp)) {
            for (mb in listOf(0L, 500L, 1_000L, 2_000L, 5_000L, 10_000L)) {
                FilterChip(
                    selected = s.shareCapMb == mb, onClick = { actions.setShareCapMb(mb) },
                    label = { Text(if (mb == 0L) "No limit" else if (mb < 1_000) "$mb MB" else "${mb / 1_000} GB") },
                    leadingIcon = if (s.shareCapMb == mb) ({ Icon(Icons.Filled.Check, null) }) else null,
                )
            }
        }

        SectionHeader("Who can use it")
        Text(
            if (s.shareAllowedHubs.isEmpty()) "Any device paired with $server." else "Only these devices (their names on $server):",
            style = MaterialTheme.typography.bodyMedium, modifier = Modifier.padding(horizontal = 16.dp),
        )
        FlowRow(Modifier.padding(horizontal = 16.dp), horizontalArrangement = Arrangement.spacedBy(8.dp)) {
            for (h in s.shareAllowedHubs) {
                InputChip(
                    selected = false, onClick = { actions.setShareAllowedHubs(s.shareAllowedHubs - h) },
                    label = { Text(h) }, trailingIcon = { Icon(Icons.Filled.Close, "Remove $h") },
                )
            }
        }
        var name by rememberSaveable { mutableStateOf("") }
        Row(Modifier.padding(horizontal = 16.dp, vertical = 8.dp), verticalAlignment = Alignment.CenterVertically) {
            OutlinedTextField(
                value = name, onValueChange = { name = it.take(64) }, singleLine = true,
                label = { Text("Device name, e.g. slate7") }, modifier = Modifier.weight(1f),
            )
            TextButton(
                onClick = {
                    val n = name.trim()
                    if (n.isNotEmpty() && n !in s.shareAllowedHubs) actions.setShareAllowedHubs(s.shareAllowedHubs + n)
                    name = ""
                },
                enabled = name.isNotBlank(),
            ) { Text("Add") }
        }
    }
}

/**
 * "Share bonded connection" (phone hub, docs/phone-hub.md): a laptop on this
 * phone's hotspot or USB uses tailbond's whole bonded tunnel through a proxy.
 */
@Composable
private fun HubCard(state: UiState, actions: UiActions) {
    val l = state.hub
    val on = l.status != HubStatus.OFF
    val muted = MaterialTheme.colorScheme.onSurfaceVariant
    Card(
        Modifier.fillMaxWidth().padding(horizontal = 16.dp, vertical = 8.dp),
        colors = CardDefaults.cardColors(
            containerColor = if (l.status == HubStatus.SHARING) MaterialTheme.colorScheme.primaryContainer else MaterialTheme.colorScheme.surfaceVariant,
        ),
    ) {
        Column(Modifier.padding(20.dp), verticalArrangement = Arrangement.spacedBy(8.dp)) {
            Row(verticalAlignment = Alignment.CenterVertically) {
                Text("Share bonded connection", style = MaterialTheme.typography.titleLarge, fontWeight = FontWeight.SemiBold, modifier = Modifier.weight(1f))
                Switch(checked = on, onCheckedChange = { if (it) actions.startHub() else actions.stopHub() })
            }
            Text(HubWords.headline(l), style = MaterialTheme.typography.titleMedium)
            Text(HubWords.detail(l), style = MaterialTheme.typography.bodyMedium)
            for (c in l.clients) {
                Row(verticalAlignment = Alignment.CenterVertically, horizontalArrangement = Arrangement.spacedBy(8.dp)) {
                    Icon(Icons.Filled.Laptop, null, tint = MaterialTheme.colorScheme.primary)
                    Text("${c.addr}: ${formatBytes(c.up + c.down)}", style = MaterialTheme.typography.bodyMedium)
                }
            }
            if (on) {
                Text(
                    "Only apps that follow the laptop's proxy setting, and only TCP (web, most apps; not video calls or games).",
                    style = MaterialTheme.typography.bodySmall, color = muted,
                )
            }
        }
    }
}

/** Pair & Share, hub side: nearby phones' shared mobile data in this phone's tunnel. */
@Composable
private fun NearbySharesSection(state: UiState, actions: UiActions) {
    val on = state.settings.useShares
    val muted = MaterialTheme.colorScheme.onSurfaceVariant
    Row(Modifier.fillMaxWidth().padding(horizontal = 16.dp, vertical = 8.dp), verticalAlignment = Alignment.CenterVertically) {
        Text("Use nearby phones' shared data", style = MaterialTheme.typography.titleMedium, modifier = Modifier.weight(1f))
        Switch(checked = on, onCheckedChange = { actions.setUseShares(it) })
    }
    val words = mapOf("ready" to "Sharing", "connecting" to "Connecting…", "refused" to "Stopped", "unreachable" to "Can't reach it", "lost" to "Stopped answering")
    if (!on) {
        Text("Off: other phones' shared data isn't used.", style = MaterialTheme.typography.bodyMedium, color = muted, modifier = Modifier.padding(horizontal = 16.dp))
    } else if (state.hubShares.isEmpty()) {
        Text(
            "While tailbond is on, phones on the same Wi-Fi sharing their mobile data are added to this phone's connection.",
            style = MaterialTheme.typography.bodyMedium, color = muted, modifier = Modifier.padding(horizontal = 16.dp),
        )
    }
    if (on) {
        for (h in state.hubShares) {
            ListItem(
                headlineContent = { Text(h.label.ifEmpty { h.addr }) },
                supportingContent = {
                    Text(
                        listOfNotNull(
                            words[h.state] ?: h.state,
                            h.reason.takeIf { h.state != "ready" && it.isNotEmpty() },
                            if (h.battery >= 0) "battery ${h.battery}%" else null,
                            formatBytes(h.up + h.down).takeIf { h.up + h.down > 0 },
                        ).joinToString(" · "),
                    )
                },
                leadingContent = { Icon(Icons.Filled.Router, null, tint = LocalBrand.current.shared) },
            )
        }
    }
}
