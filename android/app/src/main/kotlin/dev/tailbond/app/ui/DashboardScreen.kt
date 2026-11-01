package dev.tailbond.app.ui

import androidx.compose.foundation.layout.Arrangement
import androidx.compose.foundation.layout.Column
import androidx.compose.foundation.layout.Row
import androidx.compose.foundation.layout.Spacer
import androidx.compose.foundation.layout.fillMaxWidth
import androidx.compose.foundation.layout.padding
import androidx.compose.foundation.layout.width
import androidx.compose.foundation.rememberScrollState
import androidx.compose.foundation.verticalScroll
import androidx.compose.material.icons.Icons
import androidx.compose.material.icons.filled.Cable
import androidx.compose.material.icons.filled.NetworkCell
import androidx.compose.material.icons.filled.Router
import androidx.compose.material.icons.filled.Wifi
import androidx.compose.material3.AssistChip
import androidx.compose.material3.Card
import androidx.compose.material3.CardDefaults
import androidx.compose.material3.DropdownMenu
import androidx.compose.material3.DropdownMenuItem
import androidx.compose.material3.Icon
import androidx.compose.material3.LinearProgressIndicator
import androidx.compose.material3.MaterialTheme
import androidx.compose.material3.SegmentedButton
import androidx.compose.material3.SegmentedButtonDefaults
import androidx.compose.material3.SingleChoiceSegmentedButtonRow
import androidx.compose.material3.Switch
import androidx.compose.material3.Text
import androidx.compose.material3.TextButton
import androidx.compose.runtime.Composable
import androidx.compose.runtime.LaunchedEffect
import androidx.compose.runtime.getValue
import androidx.compose.runtime.mutableLongStateOf
import androidx.compose.runtime.mutableStateOf
import androidx.compose.runtime.remember
import androidx.compose.runtime.setValue
import androidx.compose.ui.Alignment
import androidx.compose.ui.Modifier
import androidx.compose.ui.draw.scale
import androidx.compose.ui.graphics.Color
import androidx.compose.ui.graphics.vector.ImageVector
import androidx.compose.ui.text.font.FontWeight
import androidx.compose.ui.unit.dp
import dev.tailbond.app.model.AppSettings
import dev.tailbond.app.model.Mode
import dev.tailbond.app.model.PathKind
import dev.tailbond.app.model.PathSetting
import dev.tailbond.app.model.Priority
import dev.tailbond.app.pairing.PairingPayload
import dev.tailbond.app.service.ConnState
import dev.tailbond.app.service.Live
import dev.tailbond.app.service.PathGate
import dev.tailbond.app.service.PathSample
import kotlinx.coroutines.delay

@Composable
fun DashboardScreen(
    live: Live,
    profile: PairingPayload,
    settings: AppSettings,
    cellularBytes: Long,
    onToggle: (Boolean) -> Unit,
    vm: MainViewModel,
    modifier: Modifier = Modifier,
) {
    Column(
        modifier.verticalScroll(rememberScrollState()).padding(16.dp),
        verticalArrangement = Arrangement.spacedBy(12.dp),
    ) {
        HeaderCard(live, profile, onToggle)
        ModeSelector(settings.mode ?: Mode.fromWire(profile.mode) ?: Mode.SPEED, vm::setMode)
        val byKind = live.paths.groupBy { it.kind }
        for (kind in PathKind.entries) {
            val samples = byKind[kind].orEmpty()
            PathCard(kind, samples, settings, live, onChange = { vm.setPath(kind, it) })
        }
        CellularCard(cellularBytes, settings.cellularCapMb, onReset = vm::resetCellular)
    }
}

@Composable
private fun HeaderCard(live: Live, profile: PairingPayload, onToggle: (Boolean) -> Unit) {
    val on = live.state != ConnState.DISCONNECTED && live.state != ConnState.ERROR
    Card(Modifier.fillMaxWidth()) {
        Column(Modifier.padding(16.dp)) {
            Row(verticalAlignment = Alignment.CenterVertically) {
                Column(Modifier.weight(1f)) {
                    Text(stateLabel(live.state), style = MaterialTheme.typography.headlineSmall, fontWeight = FontWeight.SemiBold)
                    val sub = live.message ?: listOfNotNull(
                        profile.name.ifBlank { null },
                        profile.sni ?: profile.host,
                        live.tunnelIp,
                    ).joinToString(" · ")
                    Text(sub, style = MaterialTheme.typography.bodySmall, color = MaterialTheme.colorScheme.onSurfaceVariant)
                }
                Switch(
                    checked = on,
                    onCheckedChange = onToggle,
                    modifier = Modifier.padding(end = 8.dp).scale(1.5f),
                )
            }
            Spacer(Modifier.padding(6.dp))
            Row {
                RateStat("Download", live.rxBps, DownColor, Modifier.weight(1f))
                RateStat("Upload", live.txBps, UpColor, Modifier.weight(1f))
            }
            Spacer(Modifier.padding(4.dp))
            RateGraph(listOf(live.rxHistory to DownColor, live.txHistory to UpColor))
            Row(Modifier.fillMaxWidth().padding(top = 6.dp), horizontalArrangement = Arrangement.SpaceBetween) {
                val small = MaterialTheme.typography.labelSmall
                Text("60 s", style = small)
                if (live.state == ConnState.CONNECTED) {
                    var now by remember { mutableLongStateOf(System.currentTimeMillis()) }
                    LaunchedEffect(Unit) { while (true) { delay(1000); now = System.currentTimeMillis() } }
                    val up = if (live.connectedAtMs > 0) (now - live.connectedAtMs) / 1000 else 0
                    Text(
                        "↓ ${formatBytes(live.rxBytes)}  ↑ ${formatBytes(live.txBytes)} · RTT ${live.srttMs} ms · loss ${"%.1f".format(live.lossPct)}% · ${formatDuration(up)}",
                        style = small,
                    )
                }
            }
        }
    }
}

@Composable
private fun RateStat(label: String, bps: Long, color: Color, modifier: Modifier) {
    Column(modifier) {
        Text(label, style = MaterialTheme.typography.labelMedium, color = color)
        Text(formatBps(bps), style = MaterialTheme.typography.titleLarge, fontWeight = FontWeight.SemiBold)
    }
}

private fun stateLabel(s: ConnState) = when (s) {
    ConnState.CONNECTED -> "Connected"
    ConnState.CONNECTING -> "Connecting…"
    ConnState.RECONNECTING -> "Reconnecting…"
    ConnState.ERROR -> "Error"
    ConnState.DISCONNECTED -> "Disconnected"
}

@Composable
private fun ModeSelector(selected: Mode, onSelect: (Mode) -> Unit) {
    Column {
        SingleChoiceSegmentedButtonRow(Modifier.fillMaxWidth()) {
            Mode.entries.forEachIndexed { i, m ->
                SegmentedButton(
                    selected = m == selected,
                    onClick = { onSelect(m) },
                    shape = SegmentedButtonDefaults.itemShape(i, Mode.entries.size),
                    icon = {},
                ) { Text(m.label, maxLines = 1, style = MaterialTheme.typography.labelMedium) }
            }
        }
        Text(
            selected.blurb,
            style = MaterialTheme.typography.bodySmall,
            color = MaterialTheme.colorScheme.onSurfaceVariant,
            modifier = Modifier.padding(top = 4.dp, start = 4.dp),
        )
    }
}

private fun kindIcon(k: PathKind): ImageVector = when (k) {
    PathKind.WIFI -> Icons.Filled.Wifi
    PathKind.CELLULAR -> Icons.Filled.NetworkCell
    PathKind.ETHERNET -> Icons.Filled.Cable
}

/** Which of several links of one kind the card shows: the best-working one. */
private fun statusRank(status: Int): Int = when (status) {
    PathGate.STATUS_ACTIVE -> 0
    PathGate.STATUS_STANDBY -> 1
    PathGate.STATUS_PENDING -> 2
    PathGate.STATUS_DEGRADED -> 3
    -1 -> 4
    else -> 5
}

private fun statusLabel(status: Int): String = when (status) {
    -1 -> "standby"
    PathGate.STATUS_PENDING -> "pending"
    PathGate.STATUS_ACTIVE -> "active"
    PathGate.STATUS_DEGRADED -> "degraded"
    PathGate.STATUS_STANDBY -> "standby"
    PathGate.STATUS_CLOSED -> "failed"
    else -> "?"
}

@Composable
private fun PathCard(
    kind: PathKind,
    samples: List<PathSample>,
    settings: AppSettings,
    live: Live,
    onChange: (PathSetting) -> Unit,
) {
    val setting = settings.path(kind)
    val sample = samples.minByOrNull { statusRank(it.status) }
    val connected = live.state == ConnState.CONNECTED || live.state == ConnState.RECONNECTING
    val dim = !setting.enabled || sample == null
    Card(
        Modifier.fillMaxWidth(),
        colors = if (dim) CardDefaults.cardColors(containerColor = MaterialTheme.colorScheme.surfaceContainerLow) else CardDefaults.cardColors(),
    ) {
        Column(Modifier.padding(12.dp)) {
            Row(verticalAlignment = Alignment.CenterVertically) {
                Icon(kindIcon(kind), contentDescription = null)
                Spacer(Modifier.width(8.dp))
                Text(kind.label, style = MaterialTheme.typography.titleMedium, modifier = Modifier.weight(1f))
                val chip = when {
                    !setting.enabled -> "off"
                    sample == null -> if (connected) "unavailable" else "idle"
                    else -> statusLabel(sample.status)
                }
                AssistChip(onClick = {}, label = { Text(chip) })
                Spacer(Modifier.width(8.dp))
                Switch(checked = setting.enabled, onCheckedChange = { onChange(setting.copy(enabled = it)) })
            }
            if (sample != null && sample.status >= 0) {
                Row(Modifier.fillMaxWidth().padding(top = 4.dp), horizontalArrangement = Arrangement.SpaceBetween) {
                    Text("↓ ${formatBps(sample.rxBps)}  ↑ ${formatBps(sample.txBps)}", style = MaterialTheme.typography.bodyMedium)
                    Text("RTT ${sample.srttMs} ms", style = MaterialTheme.typography.bodyMedium)
                }
                Sparkline(sample.history, MaterialTheme.colorScheme.primary, Modifier.padding(top = 4.dp))
                Text(
                    "${formatBytes(sample.rxBytes + sample.txBytes)} this session · loss: tunnel ${"%.1f".format(live.lossPct)}%",
                    style = MaterialTheme.typography.labelSmall,
                    color = MaterialTheme.colorScheme.onSurfaceVariant,
                )
            }
            PriorityPicker(setting.priority, enabled = setting.enabled) { onChange(setting.copy(priority = it)) }
        }
    }
}

@Composable
private fun PriorityPicker(p: Priority, enabled: Boolean, onPick: (Priority) -> Unit) {
    var open by remember { mutableStateOf(false) }
    Row(verticalAlignment = Alignment.CenterVertically) {
        Text("Priority", style = MaterialTheme.typography.bodySmall, modifier = Modifier.weight(1f))
        TextButton(onClick = { open = true }, enabled = enabled) { Text(p.label) }
        DropdownMenu(expanded = open, onDismissRequest = { open = false }) {
            Priority.entries.forEach {
                DropdownMenuItem(text = { Text(it.label) }, onClick = { onPick(it); open = false })
            }
        }
    }
}

@Composable
private fun CellularCard(bytes: Long, capMb: Long, onReset: () -> Unit) {
    Card(Modifier.fillMaxWidth()) {
        Column(Modifier.padding(12.dp)) {
            Row(verticalAlignment = Alignment.CenterVertically) {
                Icon(Icons.Filled.Router, contentDescription = null)
                Spacer(Modifier.width(8.dp))
                Text("Cellular data this month", style = MaterialTheme.typography.titleMedium, modifier = Modifier.weight(1f))
                TextButton(onClick = onReset) { Text("Reset") }
            }
            val cap = capMb * 1_000_000
            Text(
                if (cap > 0) "${formatBytes(bytes)} of ${formatBytes(cap)}" else "${formatBytes(bytes)} (no cap)",
                style = MaterialTheme.typography.bodyLarge,
            )
            if (cap > 0) {
                LinearProgressIndicator(
                    progress = { (bytes.toFloat() / cap).coerceIn(0f, 1f) },
                    modifier = Modifier.fillMaxWidth().padding(top = 6.dp),
                )
                if (bytes >= cap) {
                    Text(
                        "Cap reached: cellular is paused until next month or a reset.",
                        style = MaterialTheme.typography.bodySmall,
                        color = MaterialTheme.colorScheme.error,
                    )
                }
            }
        }
    }
}
