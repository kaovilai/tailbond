package dev.tailbond.app.ui

import android.content.Intent
import android.provider.Settings
import androidx.compose.foundation.clickable
import androidx.compose.foundation.layout.Arrangement
import androidx.compose.foundation.layout.Column
import androidx.compose.foundation.layout.Row
import androidx.compose.foundation.layout.fillMaxWidth
import androidx.compose.foundation.layout.heightIn
import androidx.compose.foundation.layout.padding
import androidx.compose.foundation.lazy.LazyColumn
import androidx.compose.foundation.lazy.items
import androidx.compose.foundation.rememberScrollState
import androidx.compose.foundation.text.KeyboardOptions
import androidx.compose.foundation.verticalScroll
import androidx.compose.material3.AlertDialog
import androidx.compose.material3.Card
import androidx.compose.material3.Checkbox
import androidx.compose.material3.HorizontalDivider
import androidx.compose.material3.MaterialTheme
import androidx.compose.material3.OutlinedButton
import androidx.compose.material3.OutlinedTextField
import androidx.compose.material3.RadioButton
import androidx.compose.material3.Switch
import androidx.compose.material3.Text
import androidx.compose.material3.TextButton
import androidx.compose.runtime.Composable
import androidx.compose.runtime.collectAsState
import androidx.compose.runtime.getValue
import androidx.compose.runtime.mutableStateOf
import androidx.compose.runtime.remember
import androidx.compose.runtime.setValue
import androidx.compose.ui.Alignment
import androidx.compose.ui.Modifier
import androidx.compose.ui.platform.LocalContext
import androidx.compose.ui.text.input.KeyboardType
import androidx.compose.ui.unit.dp
import dev.tailbond.app.model.AppSettings
import dev.tailbond.app.model.RouteMode
import dev.tailbond.app.model.TunnelPlan
import dev.tailbond.app.pairing.PairingPayload

@Composable
fun SettingsScreen(vm: MainViewModel, profile: PairingPayload?, settings: AppSettings, onRepair: () -> Unit, modifier: Modifier = Modifier) {
    val ctx = LocalContext.current
    var appsOpen by remember { mutableStateOf(false) }
    var forgetOpen by remember { mutableStateOf(false) }
    Column(
        modifier.verticalScroll(rememberScrollState()).padding(16.dp),
        verticalArrangement = Arrangement.spacedBy(12.dp),
    ) {
        Section("Routing") {
            RouteMode.entries.forEach { r ->
                Row(
                    Modifier.fillMaxWidth().clickable { vm.setRouteMode(r) },
                    verticalAlignment = Alignment.CenterVertically,
                ) {
                    RadioButton(selected = settings.routeMode == r, onClick = { vm.setRouteMode(r) })
                    Column {
                        Text(r.label)
                        val detail = when (r) {
                            RouteMode.FULL -> "All traffic goes through your home server"
                            RouteMode.TAILNET_LAN -> (profile?.tailnetRoutes.orEmpty() + TunnelPlan.TUNNEL_SUBNET.toString())
                                .joinToString(", ")
                        }
                        Text(detail, style = MaterialTheme.typography.bodySmall, color = MaterialTheme.colorScheme.onSurfaceVariant)
                    }
                }
            }
            HorizontalDivider()
            Row(verticalAlignment = Alignment.CenterVertically) {
                Column(Modifier.weight(1f)) {
                    Text("Excluded apps")
                    Text(
                        if (settings.excludedApps.isEmpty()) "None" else "${settings.excludedApps.size} app(s) bypass the tunnel",
                        style = MaterialTheme.typography.bodySmall,
                    )
                }
                OutlinedButton(onClick = { vm.loadApps(); appsOpen = true }) { Text("Choose") }
            }
        }

        Section("Cellular data") {
            var capText by remember(settings.cellularCapMb) {
                mutableStateOf(if (settings.cellularCapMb > 0) settings.cellularCapMb.toString() else "")
            }
            OutlinedTextField(
                value = capText,
                onValueChange = { v ->
                    capText = v.filter { it.isDigit() }.take(9)
                    vm.setCapMb(capText.toLongOrNull() ?: 0)
                },
                label = { Text("Monthly cap (MB, empty = none)") },
                keyboardOptions = KeyboardOptions(keyboardType = KeyboardType.Number),
                modifier = Modifier.fillMaxWidth(),
                singleLine = true,
            )
            Text(
                "Counts tunnel traffic carried over cellular. At the cap, cellular stops carrying paths until next month.",
                style = MaterialTheme.typography.bodySmall,
            )
        }

        Section("Failover") {
            Row(verticalAlignment = Alignment.CenterVertically) {
                Column(Modifier.weight(1f)) {
                    Text("Experimental: backup_fec scheduler")
                    Text("Failover mode only. Uses mqvpn's FEC-protected backup scheduler.", style = MaterialTheme.typography.bodySmall)
                }
                Switch(checked = settings.experimentalBackupFec, onCheckedChange = vm::setBackupFec)
            }
        }

        Section("VPN") {
            Text(
                "To keep tailbond on all the time, enable Always-on VPN (and optionally Block connections without VPN) in system settings.",
                style = MaterialTheme.typography.bodySmall,
            )
            OutlinedButton(onClick = { ctx.startActivity(Intent(Settings.ACTION_VPN_SETTINGS)) }) { Text("Open VPN settings") }
        }

        Section("Pairing") {
            if (profile != null) {
                Text("${profile.name.ifBlank { profile.user }} → ${profile.server}", style = MaterialTheme.typography.bodyMedium)
                Row(horizontalArrangement = Arrangement.spacedBy(8.dp)) {
                    OutlinedButton(onClick = onRepair) { Text("Pair again") }
                    TextButton(onClick = { forgetOpen = true }) { Text("Forget", color = MaterialTheme.colorScheme.error) }
                }
            } else {
                OutlinedButton(onClick = onRepair) { Text("Pair") }
            }
        }
    }

    if (appsOpen) AppPicker(vm, settings.excludedApps) { appsOpen = false }
    if (forgetOpen) {
        AlertDialog(
            onDismissRequest = { forgetOpen = false },
            title = { Text("Forget this server?") },
            text = { Text("The tunnel stops and the stored key is deleted. You'll need a new pairing code.") },
            confirmButton = { TextButton(onClick = { forgetOpen = false; vm.forgetPairing() }) { Text("Forget") } },
            dismissButton = { TextButton(onClick = { forgetOpen = false }) { Text("Cancel") } },
        )
    }
}

@Composable
private fun Section(title: String, content: @Composable () -> Unit) {
    Card(Modifier.fillMaxWidth()) {
        Column(Modifier.padding(12.dp), verticalArrangement = Arrangement.spacedBy(8.dp)) {
            Text(title, style = MaterialTheme.typography.titleMedium)
            content()
        }
    }
}

@Composable
private fun AppPicker(vm: MainViewModel, excluded: Set<String>, onClose: () -> Unit) {
    val apps by vm.apps.collectAsState()
    AlertDialog(
        onDismissRequest = onClose,
        title = { Text("Apps that bypass the tunnel") },
        text = {
            if (apps.isEmpty()) {
                Text("Loading…")
            } else {
                LazyColumn(Modifier.heightIn(max = 420.dp)) {
                    items(apps, key = { it.pkg }) { a ->
                        val checked = a.pkg in excluded
                        Row(
                            Modifier.fillMaxWidth().clickable { vm.setExcluded(a.pkg, !checked) },
                            verticalAlignment = Alignment.CenterVertically,
                        ) {
                            Checkbox(checked = checked, onCheckedChange = { vm.setExcluded(a.pkg, it) })
                            Column {
                                Text(a.label)
                                Text(a.pkg, style = MaterialTheme.typography.labelSmall)
                            }
                        }
                    }
                }
            }
        },
        confirmButton = { TextButton(onClick = onClose) { Text("Done") } },
    )
}
