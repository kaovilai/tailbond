package dev.tailbond.app.ui

import androidx.compose.foundation.clickable
import androidx.compose.foundation.layout.Arrangement
import androidx.compose.foundation.layout.Column
import androidx.compose.foundation.layout.Row
import androidx.compose.foundation.layout.Spacer
import androidx.compose.foundation.layout.fillMaxWidth
import androidx.compose.foundation.layout.padding
import androidx.compose.foundation.layout.width
import androidx.compose.foundation.rememberScrollState
import androidx.compose.foundation.selection.selectable
import androidx.compose.foundation.text.KeyboardOptions
import androidx.compose.foundation.verticalScroll
import androidx.compose.material3.AlertDialog
import androidx.compose.material3.Card
import androidx.compose.material3.DropdownMenu
import androidx.compose.material3.DropdownMenuItem
import androidx.compose.material3.HorizontalDivider
import androidx.compose.material3.Icon
import androidx.compose.material3.LinearProgressIndicator
import androidx.compose.material3.MaterialTheme
import androidx.compose.material3.OutlinedTextField
import androidx.compose.material3.RadioButton
import androidx.compose.material3.Switch
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
import androidx.compose.ui.text.input.KeyboardType
import androidx.compose.ui.unit.dp
import dev.tailbond.app.model.Mode
import dev.tailbond.app.model.PathKind
import dev.tailbond.app.model.Priority
import dev.tailbond.app.model.Words
import dev.tailbond.app.service.ConnState
import dev.tailbond.app.service.PathGate
import java.util.Locale

/** Bonding: which connections carry traffic, how (mode), and the mobile data limit. */
@Composable
fun ConnectionsScreen(state: UiState, actions: UiActions, modifier: Modifier = Modifier) {
    val live = state.live
    val on = live.state == ConnState.CONNECTED || live.state == ConnState.RECONNECTING
    Column(
        modifier.verticalScroll(rememberScrollState()).padding(bottom = 24.dp),
    ) {
        Card(Modifier.fillMaxWidth().padding(16.dp)) {
            Column(Modifier.padding(16.dp)) {
                Text(
                    Words.connectionsSummary(live, state.settings, state.mode, state.capReached),
                    style = MaterialTheme.typography.titleMedium,
                )
                if (on) {
                    Row(Modifier.padding(top = 8.dp)) {
                        Rate("Download", live.rxBps, Modifier.weight(1f))
                        Rate("Upload", live.txBps, Modifier.weight(1f))
                    }
                    RateGraph(listOf(live.rxHistory to DownColor, live.txHistory to UpColor), Modifier.padding(top = 8.dp), height = 72.dp)
                    Text(
                        "Last 60 seconds · ${formatBytes(live.rxBytes + live.txBytes)} this session",
                        style = MaterialTheme.typography.labelSmall,
                        color = MaterialTheme.colorScheme.onSurfaceVariant,
                    )
                }
            }
        }

        val kinds = PathKind.entries.filter { it != PathKind.ETHERNET || Words.sampleFor(live, it) != null }
        for (kind in kinds) LinkCard(kind, state, actions)

        SectionHeader("How to use your connections")
        for (m in Mode.entries) {
            Row(
                Modifier.fillMaxWidth()
                    .selectable(selected = m == state.mode, onClick = { actions.setMode(m) }, role = Role.RadioButton)
                    .padding(horizontal = 8.dp, vertical = 4.dp),
                verticalAlignment = Alignment.CenterVertically,
            ) {
                RadioButton(selected = m == state.mode, onClick = null, modifier = Modifier.padding(8.dp))
                Column {
                    Text(m.label, style = MaterialTheme.typography.bodyLarge)
                    Text(m.blurb, style = MaterialTheme.typography.bodySmall, color = MaterialTheme.colorScheme.onSurfaceVariant)
                }
            }
        }
        Hint("Changing this reconnects for a moment.")

        SectionHeader("Mobile data limit")
        DataLimit(state, actions)
    }
}

@Composable
private fun Rate(label: String, bps: Long, modifier: Modifier) {
    Column(modifier) {
        Text(label, style = MaterialTheme.typography.labelMedium, color = MaterialTheme.colorScheme.onSurfaceVariant)
        Text(formatBps(bps), style = MaterialTheme.typography.titleLarge, fontWeight = FontWeight.SemiBold)
    }
}

@Composable
private fun LinkCard(kind: PathKind, state: UiState, actions: UiActions) {
    val live = state.live
    val setting = state.settings.path(kind)
    val st = Words.linkStatus(kind, live, state.settings, state.mode, state.capReached)
    val sample = Words.sampleFor(live, kind)
    val on = live.state == ConnState.CONNECTED || live.state == ConnState.RECONNECTING
    Card(Modifier.fillMaxWidth().padding(horizontal = 16.dp, vertical = 6.dp)) {
        Column(Modifier.padding(16.dp), verticalArrangement = Arrangement.spacedBy(6.dp)) {
            Row(verticalAlignment = Alignment.CenterVertically) {
                Icon(kindIcon(kind), null)
                Spacer(Modifier.width(10.dp))
                Column(Modifier.weight(1f)) {
                    Text(kind.label, style = MaterialTheme.typography.titleMedium)
                    Row(verticalAlignment = Alignment.CenterVertically) {
                        StatusDot(if (!on || st == Words.LinkStatus.OFF) null else st.good)
                        Spacer(Modifier.width(6.dp))
                        Text(st.label, style = MaterialTheme.typography.bodyMedium)
                    }
                }
            }
            if (on && sample != null && sample.status == PathGate.STATUS_ACTIVE) {
                Text(
                    "↓ ${formatBps(sample.rxBps)}  ↑ ${formatBps(sample.txBps)} · delay ${sample.srttMs} ms",
                    style = MaterialTheme.typography.bodyMedium,
                )
                Sparkline(sample.history, MaterialTheme.colorScheme.primary)
            }
            HorizontalDivider()
            Row(verticalAlignment = Alignment.CenterVertically) {
                Column(Modifier.weight(1f)) {
                    Text("Use for tailbond")
                    Text(
                        "Turning this off doesn't turn off the phone's ${kind.phrase}",
                        style = MaterialTheme.typography.bodySmall,
                        color = MaterialTheme.colorScheme.onSurfaceVariant,
                    )
                }
                Switch(checked = setting.enabled, onCheckedChange = { actions.setPath(kind, setting.copy(enabled = it)) })
            }
            RolePicker(setting.priority, enabled = setting.enabled) { actions.setPath(kind, setting.copy(priority = it)) }
        }
    }
}

@Composable
private fun RolePicker(p: Priority, enabled: Boolean, onPick: (Priority) -> Unit) {
    var open by remember { mutableStateOf(false) }
    Row(verticalAlignment = Alignment.CenterVertically) {
        Column(Modifier.weight(1f)) {
            Text("Role")
            Text(p.explain, style = MaterialTheme.typography.bodySmall, color = MaterialTheme.colorScheme.onSurfaceVariant)
        }
        TextButton(onClick = { open = true }, enabled = enabled) { Text(p.label) }
        DropdownMenu(expanded = open, onDismissRequest = { open = false }) {
            Priority.entries.forEach {
                DropdownMenuItem(
                    text = { Column { Text(it.label); Text(it.explain, style = MaterialTheme.typography.bodySmall) } },
                    onClick = { onPick(it); open = false },
                )
            }
        }
    }
}

private fun gb(bytes: Long) = String.format(Locale.US, "%.2f GB", bytes / 1e9)

@Composable
private fun DataLimit(state: UiState, actions: UiActions) {
    val capMb = state.settings.cellularCapMb
    val cap = capMb * 1_000_000
    var resetOpen by remember { mutableStateOf(false) }
    Column(Modifier.padding(horizontal = 16.dp), verticalArrangement = Arrangement.spacedBy(8.dp)) {
        Text(
            if (cap > 0) "${gb(state.cellularBytes)} of ${gb(cap)} used this month" else "${gb(state.cellularBytes)} used this month · no limit",
            style = MaterialTheme.typography.bodyLarge,
        )
        if (cap > 0) {
            LinearProgressIndicator(progress = { (state.cellularBytes.toFloat() / cap).coerceIn(0f, 1f) }, modifier = Modifier.fillMaxWidth())
        }
        if (state.capReached) {
            Text("Limit reached: tailbond isn't using mobile data until next month.", color = MaterialTheme.colorScheme.error)
        }
        var text by remember(capMb) {
            mutableStateOf(if (capMb > 0) (capMb / 1000.0).toString().removeSuffix(".0") else "")
        }
        OutlinedTextField(
            value = text,
            onValueChange = { v ->
                text = v.filter { it.isDigit() || it == '.' }.take(7)
                actions.setCapMb(((text.toDoubleOrNull() ?: 0.0) * 1000).toLong())
            },
            label = { Text("Monthly limit in GB (empty = no limit)") },
            keyboardOptions = KeyboardOptions(keyboardType = KeyboardType.Decimal),
            singleLine = true,
            modifier = Modifier.fillMaxWidth(),
        )
        Text(
            "Counts only what tailbond sends over mobile data. At the limit, tailbond stops using mobile data until the 1st of next month.",
            style = MaterialTheme.typography.bodySmall,
            color = MaterialTheme.colorScheme.onSurfaceVariant,
        )
        Text(
            "Reset this month's count",
            color = MaterialTheme.colorScheme.primary,
            style = MaterialTheme.typography.labelLarge,
            modifier = Modifier.clickable { resetOpen = true }.padding(vertical = 8.dp),
        )
    }
    if (resetOpen) {
        AlertDialog(
            onDismissRequest = { resetOpen = false },
            title = { Text("Reset this month's count?") },
            text = { Text("tailbond's mobile data count starts again from 0. This doesn't change your carrier's count.") },
            confirmButton = { TextButton(onClick = { resetOpen = false; actions.resetCellular() }) { Text("Reset") } },
            dismissButton = { TextButton(onClick = { resetOpen = false }) { Text("Cancel") } },
        )
    }
}
