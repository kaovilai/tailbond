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
import dev.tailbond.app.ble.MacLinkStatus
import dev.tailbond.app.ble.MacLinkWords
import dev.tailbond.app.ble.ShizukuStatus
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
        HubCard(state, actions)
        MacLinkCard(state, actions)
        SectionHeader("Lend your mobile data to a travel router")
        Card(
            Modifier.fillMaxWidth().padding(horizontal = 16.dp, vertical = 8.dp),
            colors = CardDefaults.cardColors(
                containerColor = if (l.status == ShareStatus.SHARING) MaterialTheme.colorScheme.primaryContainer else MaterialTheme.colorScheme.surfaceVariant,
            ),
        ) {
            Column(Modifier.padding(20.dp), verticalArrangement = Arrangement.spacedBy(8.dp)) {
                Row(verticalAlignment = Alignment.CenterVertically) {
                    Text("Share my mobile data", style = MaterialTheme.typography.titleLarge, fontWeight = FontWeight.SemiBold, modifier = Modifier.weight(1f))
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

        SectionHeader("Who's using your mobile data")
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

        SectionHeader("Mobile data: stop when the battery is low")
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

        SectionHeader("Mobile data: monthly limit")
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

        SectionHeader("Who can use your mobile data")
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
        SectionHeader("Use other phones' data")
        NearbySharesSection(state, actions)
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
                    Text("Device at ${c.addr}: ${formatBytes(c.up + c.down)}", style = MaterialTheme.typography.bodyMedium)
                }
            }
            if (l.status == HubStatus.NO_TUNNEL) {
                TextButton(onClick = { actions.setOn(true) }) { Text("Turn tailbond on") }
            }
            Text(
                "Only devices paired with ${state.serverName} can use it: they get its password from ${state.serverName}. The battery and monthly limits below are for lending mobile data, not for this.",
                style = MaterialTheme.typography.bodySmall, color = muted,
            )
            if (on) {
                Text(
                    "Only apps that follow the laptop's proxy setting, and only TCP (web, most apps; not video calls or games).",
                    style = MaterialTheme.typography.bodySmall, color = muted,
                )
            }
        }
    }
}

/**
 * "One click from your Mac" (docs/phone-hub.md): the Mac connects over
 * Bluetooth, starts this phone's hotspot (Shizuku) and Share bonded
 * connection, joins the hotspot and uses it.
 */
@Composable
private fun MacLinkCard(state: UiState, actions: UiActions) {
    val l = state.macLink
    val on = l.status != MacLinkStatus.OFF && l.status != MacLinkStatus.NO_PERMISSION
    val muted = MaterialTheme.colorScheme.onSurfaceVariant
    Card(
        Modifier.fillMaxWidth().padding(horizontal = 16.dp, vertical = 8.dp),
        colors = CardDefaults.cardColors(
            containerColor = if (l.status == MacLinkStatus.CONNECTED) MaterialTheme.colorScheme.primaryContainer else MaterialTheme.colorScheme.surfaceVariant,
        ),
    ) {
        Column(Modifier.padding(20.dp), verticalArrangement = Arrangement.spacedBy(8.dp)) {
            Row(verticalAlignment = Alignment.CenterVertically) {
                Text("One click from your Mac", style = MaterialTheme.typography.titleLarge, fontWeight = FontWeight.SemiBold, modifier = Modifier.weight(1f))
                Switch(checked = on, onCheckedChange = { if (it) actions.startMacLink() else actions.stopMacLink() })
            }
            Text(MacLinkWords.headline(l), style = MaterialTheme.typography.titleMedium)
            Text(MacLinkWords.shizuku(l.shizuku, state.sdkInt), style = MaterialTheme.typography.bodyMedium)
            if (l.shizuku == ShizukuStatus.NO_PERMISSION) {
                TextButton(onClick = { actions.allowShizuku() }) { Text("Allow in Shizuku") }
            }
            Text(
                "On a Mac paired with ${state.serverName}, the tailbond menu's \"Connect via phone\" turns on this phone's hotspot and Share bonded connection over Bluetooth, joins the hotspot and uses it. Without Shizuku, turn the hotspot on yourself: the Mac still finds Share bonded connection on it.",
                style = MaterialTheme.typography.bodySmall, color = muted,
            )
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
