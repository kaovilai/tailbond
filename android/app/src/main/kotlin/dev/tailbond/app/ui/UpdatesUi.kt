package dev.tailbond.app.ui

import androidx.compose.foundation.background
import androidx.compose.foundation.clickable
import androidx.compose.foundation.layout.Arrangement
import androidx.compose.foundation.layout.Box
import androidx.compose.foundation.layout.Column
import androidx.compose.foundation.layout.Row
import androidx.compose.foundation.layout.fillMaxSize
import androidx.compose.foundation.layout.fillMaxWidth
import androidx.compose.foundation.layout.padding
import androidx.compose.foundation.rememberScrollState
import androidx.compose.foundation.selection.selectable
import androidx.compose.foundation.text.KeyboardOptions
import androidx.compose.foundation.verticalScroll
import androidx.compose.material.icons.Icons
import androidx.compose.material.icons.automirrored.filled.KeyboardArrowRight
import androidx.compose.material.icons.automirrored.filled.Undo
import androidx.compose.material.icons.filled.CloudUpload
import androidx.compose.material.icons.filled.Download
import androidx.compose.material.icons.filled.Restore
import androidx.compose.material.icons.filled.Save
import androidx.compose.material.icons.filled.SystemUpdate
import androidx.compose.material3.AlertDialog
import androidx.compose.material3.Button
import androidx.compose.material3.Card
import androidx.compose.material3.CardDefaults
import androidx.compose.material3.Checkbox
import androidx.compose.material3.HorizontalDivider
import androidx.compose.material3.Icon
import androidx.compose.material3.LinearProgressIndicator
import androidx.compose.material3.ListItem
import androidx.compose.material3.MaterialTheme
import androidx.compose.material3.OutlinedButton
import androidx.compose.material3.OutlinedTextField
import androidx.compose.material3.ProvideTextStyle
import androidx.compose.material3.RadioButton
import androidx.compose.material3.Text
import androidx.compose.material3.TextButton
import androidx.compose.runtime.Composable
import androidx.compose.runtime.LaunchedEffect
import androidx.compose.runtime.getValue
import androidx.compose.runtime.mutableStateOf
import androidx.compose.runtime.saveable.rememberSaveable
import androidx.compose.runtime.setValue
import androidx.compose.ui.Alignment
import androidx.compose.ui.Modifier
import androidx.compose.ui.graphics.Color
import androidx.compose.ui.semantics.Role
import androidx.compose.ui.text.font.FontWeight
import androidx.compose.ui.text.input.KeyboardType
import androidx.compose.ui.text.input.PasswordVisualTransformation
import androidx.compose.ui.text.input.VisualTransformation
import androidx.compose.ui.unit.dp
import dev.tailbond.app.backup.BackupFile
import dev.tailbond.app.service.ConnState
import dev.tailbond.app.update.Availability
import dev.tailbond.app.update.UpdatePolicy
import dev.tailbond.app.update.UpdateWords

/** Settings: the "Updates" and "Backup" sections (one row each, opening their pages). */
@Composable
fun UpdatesBackupRows(state: UiState, onUpdates: () -> Unit, onBackup: () -> Unit) {
    val s = state.update.status
    val available = s.availability == Availability.AVAILABLE
    SectionHeader("Updates")
    ListItem(
        headlineContent = { Text("tailbond ${s.current}") },
        supportingContent = {
            Text(
                UpdateWords.row(s, state.settings.updatePolicy, state.now()),
                color = if (available) MaterialTheme.colorScheme.primary else MaterialTheme.colorScheme.onSurfaceVariant,
                fontWeight = if (available) FontWeight.SemiBold else null,
            )
        },
        leadingContent = { Icon(Icons.Filled.SystemUpdate, null) },
        trailingContent = { Icon(Icons.AutoMirrored.Filled.KeyboardArrowRight, null) },
        modifier = Modifier.clickable(onClick = onUpdates),
    )
    SectionHeader("Backup")
    ListItem(
        headlineContent = { Text("Back up and restore") },
        supportingContent = {
            Text(
                if (state.backup.undoAt != null) "Restored ${UpdateWords.whenText(state.backup.undoAt, state.now())}. You can still undo it"
                else "Your servers and settings in an encrypted file",
            )
        },
        leadingContent = { Icon(Icons.Filled.Save, null) },
        trailingContent = { Icon(Icons.AutoMirrored.Filled.KeyboardArrowRight, null) },
        modifier = Modifier.clickable(onClick = onBackup),
    )
}

private fun UiState.now() = update.nowMs.takeIf { it > 0 } ?: System.currentTimeMillis()

/** Settings → Updates: GitHub's latest release, the check policy, and Obtainium for automatic updates. */
@Composable
fun UpdatesScreen(state: UiState, actions: UiActions, modifier: Modifier = Modifier) {
    val u = state.update
    val s = u.status
    val policy = state.settings.updatePolicy
    val muted = MaterialTheme.colorScheme.onSurfaceVariant
    val a = s.availability
    LaunchedEffect(Unit) { actions.updatesShown() }
    Column(modifier.verticalScroll(rememberScrollState()).padding(bottom = 24.dp)) {
        val bad = s.error != null && a != Availability.AVAILABLE && !s.checking
        Card(
            colors = CardDefaults.cardColors(
                containerColor = if (bad) MaterialTheme.colorScheme.errorContainer else MaterialTheme.colorScheme.surfaceContainerHigh,
            ),
            modifier = Modifier.fillMaxWidth().padding(16.dp),
        ) {
            Column(Modifier.padding(16.dp), verticalArrangement = Arrangement.spacedBy(8.dp)) {
                Text("tailbond ${s.current}", style = MaterialTheme.typography.labelLarge, color = muted)
                Text(
                    UpdateWords.headline(s, policy),
                    style = MaterialTheme.typography.titleLarge, fontWeight = FontWeight.SemiBold,
                    color = if (a == Availability.AVAILABLE) MaterialTheme.colorScheme.primary else Color.Unspecified,
                )
                UpdateWords.detail(s, policy)?.let { Text(it, style = MaterialTheme.typography.bodyMedium) }
                if (s.checking) LinearProgressIndicator(Modifier.fillMaxWidth())
                Text(UpdateWords.lastChecked(s, state.now()), style = MaterialTheme.typography.bodySmall, color = muted)
                val apk = s.apkUrl
                if (a == Availability.AVAILABLE && apk != null) {
                    Row(horizontalArrangement = Arrangement.spacedBy(4.dp), verticalAlignment = Alignment.CenterVertically) {
                        Button(onClick = { actions.openUrl(apk) }) {
                            Icon(Icons.Filled.Download, null)
                            Text("  Download update")
                        }
                        s.notesUrl?.let { url -> TextButton(onClick = { actions.openUrl(url) }) { Text("What's new", maxLines = 1) } }
                    }
                    if (!u.debugBuild) Text(UpdateWords.howToInstall(s.latest), style = MaterialTheme.typography.bodySmall, color = muted)
                } else if (a == Availability.DEV_BUILD || a == Availability.NO_APK) {
                    s.notesUrl?.let { url -> TextButton(onClick = { actions.openUrl(url) }) { Text("What's new in v${s.latest}") } }
                }
                OutlinedButton(onClick = actions::checkUpdates, enabled = !s.checking) { Text("Check now") }
            }
        }

        if (u.debugBuild) Hint(UpdateWords.DEBUG_BUILD)

        SectionHeader("Check for updates")
        UpdatePolicy.entries.forEach { p ->
            Choice(policy == p, p.label, p.explain) { actions.setUpdatePolicy(p) }
        }
        Text(
            "The check only asks GitHub which release of github.com/${u.repo} is the latest. It sends nothing about you or your servers.",
            style = MaterialTheme.typography.bodySmall, color = muted,
            modifier = Modifier.padding(horizontal = 16.dp, vertical = 4.dp),
        )

        SectionHeader("Automatic updates")
        Column(Modifier.padding(horizontal = 16.dp), verticalArrangement = Arrangement.spacedBy(4.dp)) {
            Text(UpdateWords.OBTAINIUM_TITLE, style = MaterialTheme.typography.titleMedium)
            Text(UpdateWords.obtainium(u.repo), style = MaterialTheme.typography.bodyMedium)
            TextButton(onClick = { actions.openUrl("https://github.com/${u.repo}/blob/main/docs/android.md#updates") }) {
                Text("How to set up Obtainium")
            }
        }
    }
}

@Composable
private fun Choice(selected: Boolean, label: String, explain: String, onClick: () -> Unit) {
    Row(
        Modifier.fillMaxWidth()
            .selectable(selected = selected, onClick = onClick, role = Role.RadioButton)
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

/**
 * Settings → Back up and restore. [startDialog] (renders only): "export-backup" / "push-backup"
 * opens that dialog; any value draws dialogs inline (a real dialog window with a text field
 * never settles under Robolectric, see WebsitesUi).
 */
@Composable
fun BackupScreen(state: UiState, actions: UiActions, modifier: Modifier = Modifier, startDialog: String? = null) {
    var dialog by rememberSaveable { mutableStateOf(startDialog?.takeIf { it == "export-backup" || it == "push-backup" }) }
    val inline = startDialog != null
    val b = state.backup
    val server = state.activeServer?.name ?: "your home server"
    val connected = state.live.state == ConnState.CONNECTED
    Column(modifier.verticalScroll(rememberScrollState()).padding(bottom = 24.dp)) {
        Hint(
            "A backup holds your servers (with their keys) and every setting: modes, apps and websites that skip tailbond, limits, updates. " +
                "It's always encrypted with a passphrase you choose; without it nobody can open the file, not even you.",
        )
        b.undoAt?.let { at ->
            Card(
                colors = CardDefaults.cardColors(containerColor = MaterialTheme.colorScheme.secondaryContainer),
                modifier = Modifier.fillMaxWidth().padding(horizontal = 16.dp, vertical = 8.dp),
            ) {
                Column(Modifier.padding(16.dp), verticalArrangement = Arrangement.spacedBy(8.dp)) {
                    Text("Restored ${UpdateWords.whenText(at, state.now())}", style = MaterialTheme.typography.titleMedium)
                    Text("Changed your mind? Undo puts back the servers and settings from before the restore. It works once.", style = MaterialTheme.typography.bodyMedium)
                    OutlinedButton(onClick = actions::undoRestore, enabled = b.busy == null) {
                        Icon(Icons.AutoMirrored.Filled.Undo, null)
                        Text("  Undo restore")
                    }
                }
            }
        }
        ListItem(
            headlineContent = { Text("Save a backup file") },
            supportingContent = { Text("You pick where: Downloads, Google Drive, a USB stick…") },
            leadingContent = { Icon(Icons.Filled.Save, null) },
            modifier = Modifier.clickable(enabled = b.busy == null) { dialog = "export-backup" },
        )
        ListItem(
            headlineContent = { Text("Restore from a file") },
            supportingContent = { Text("Shows what changes before anything changes") },
            leadingContent = { Icon(Icons.Filled.Restore, null) },
            modifier = Modifier.clickable(enabled = b.busy == null, onClick = actions::pickRestoreFile),
        )
        ListItem(
            headlineContent = { Text("Send a backup to $server") },
            supportingContent = {
                Text(if (connected) "$server keeps the last 3, still encrypted. Handy for a new phone" else "Connect first: it goes through tailbond")
            },
            leadingContent = { Icon(Icons.Filled.CloudUpload, null) },
            modifier = Modifier.clickable(enabled = connected && b.busy == null) { dialog = "push-backup" },
        )
        HorizontalDivider(Modifier.padding(top = 8.dp))
        b.busy?.let { Hint(it); LinearProgressIndicator(Modifier.fillMaxWidth().padding(horizontal = 16.dp)) }
        b.message?.let { Hint(it) }
        b.error?.let {
            Text(it, color = MaterialTheme.colorScheme.error, style = MaterialTheme.typography.bodyMedium, modifier = Modifier.padding(horizontal = 16.dp, vertical = 8.dp))
        }
    }

    when (dialog) {
        "export-backup" -> PassphraseDialog(
            inline = inline,
            title = "Save a backup",
            confirm = "Choose where to save",
            onConfirm = { actions.exportBackup(it); dialog = null },
            onDismiss = { dialog = null },
        )
        "push-backup" -> PassphraseDialog(
            inline = inline,
            title = "Send a backup to $server",
            confirm = "Send",
            onConfirm = { actions.pushBackup(it); dialog = null },
            onDismiss = { dialog = null },
        )
    }
    b.picked?.let { h -> RestorePassphraseDialog(h, b, actions, inline) }
    b.preview?.let { p -> RestorePreviewDialog(p, b, actions) }
    if (b.restoredAt != null && b.preview == null) RestoreDoneDialog(b, actions)
}

/** New passphrase, typed twice, at least [BackupFile.MIN_PASSPHRASE] characters. */
@Composable
private fun PassphraseDialog(inline: Boolean, title: String, confirm: String, onConfirm: (String) -> Unit, onDismiss: () -> Unit) {
    var pass by rememberSaveable { mutableStateOf("") }
    var again by rememberSaveable { mutableStateOf("") }
    var show by rememberSaveable { mutableStateOf(false) }
    val short = pass.length < BackupFile.MIN_PASSPHRASE
    val mismatch = again.isNotEmpty() && again != pass
    DialogFrame(
        inline = inline,
        onDismiss = onDismiss,
        title = { Text(title) },
        text = {
            Column(verticalArrangement = Arrangement.spacedBy(8.dp)) {
                Text("Choose a passphrase for this backup. You need it to restore. tailbond can't recover it if you forget it.")
                PassField(pass, { pass = it }, "Passphrase", show,
                    if (pass.isNotEmpty() && short) "At least ${BackupFile.MIN_PASSPHRASE} characters (${pass.length} so far)" else "At least ${BackupFile.MIN_PASSPHRASE} characters. A few words work well",
                    pass.isNotEmpty() && short)
                PassField(again, { again = it }, "Type it again", show, if (mismatch) "Doesn't match" else null, mismatch)
                Row(Modifier.selectable(selected = show, onClick = { show = !show }, role = Role.Checkbox), verticalAlignment = Alignment.CenterVertically) {
                    Checkbox(checked = show, onCheckedChange = null)
                    Text("  Show passphrase")
                }
                Text(
                    "The file holds your servers' keys, so it's always encrypted. Keep it and the passphrase in different places.",
                    style = MaterialTheme.typography.bodySmall, color = MaterialTheme.colorScheme.onSurfaceVariant,
                )
            }
        },
        confirmButton = { TextButton(onClick = { onConfirm(pass) }, enabled = !short && again == pass) { Text(confirm) } },
        dismissButton = { TextButton(onClick = onDismiss) { Text("Cancel") } },
    )
}

@Composable
private fun PassField(value: String, onChange: (String) -> Unit, label: String, show: Boolean, help: String?, error: Boolean) {
    OutlinedTextField(
        value = value,
        onValueChange = onChange,
        label = { Text(label) },
        singleLine = true,
        isError = error,
        supportingText = help?.let { { Text(it) } },
        visualTransformation = if (show) VisualTransformation.None else PasswordVisualTransformation(),
        keyboardOptions = KeyboardOptions(keyboardType = KeyboardType.Password, autoCorrectEnabled = false),
        modifier = Modifier.fillMaxWidth(),
    )
}

@Composable
private fun RestorePassphraseDialog(h: dev.tailbond.app.backup.BackupHeader, b: BackupUi, actions: UiActions, inline: Boolean) {
    var pass by rememberSaveable { mutableStateOf("") }
    DialogFrame(
        inline = inline,
        onDismiss = actions::dismissBackup,
        title = { Text("Restore a backup") },
        text = {
            Column(verticalArrangement = Arrangement.spacedBy(8.dp)) {
                Text("Backup of ${h.host.ifBlank { "a phone" }} from ${createdText(h.created)}, tailbond ${h.agentVersion}. Type its passphrase.")
                PassField(pass, { pass = it }, "Passphrase", false, b.error, b.error != null)
                b.busy?.let { Text(it, style = MaterialTheme.typography.bodySmall) }
            }
        },
        confirmButton = { TextButton(onClick = { actions.openRestore(pass) }, enabled = pass.isNotEmpty() && b.busy == null) { Text("Open") } },
        dismissButton = { TextButton(onClick = actions::dismissBackup) { Text("Cancel") } },
    )
}

@Composable
private fun RestorePreviewDialog(p: dev.tailbond.app.backup.RestorePreview, b: BackupUi, actions: UiActions) {
    AlertDialog(
        onDismissRequest = actions::dismissBackup,
        title = { Text("Restore this backup?") },
        text = {
            Column(Modifier.verticalScroll(rememberScrollState()), verticalArrangement = Arrangement.spacedBy(8.dp)) {
                Text("From ${p.host.ifBlank { "a phone" }}, ${createdText(p.created)}, tailbond ${p.appVersion}.")
                if (p.changes.isEmpty()) {
                    Text("Nothing changes: this phone already has the same servers and settings.")
                } else {
                    Text("What changes", style = MaterialTheme.typography.titleSmall)
                    p.changes.forEach { c ->
                        Column {
                            Text(c.what, style = MaterialTheme.typography.bodyMedium, fontWeight = FontWeight.SemiBold)
                            Text("${c.from}  →  ${c.to}", style = MaterialTheme.typography.bodyMedium)
                        }
                    }
                }
                p.warnings.forEach { Text(it, style = MaterialTheme.typography.bodyMedium, color = MaterialTheme.colorScheme.error) }
                Text(
                    "Your current servers and settings are kept on this phone, so you can undo this once.",
                    style = MaterialTheme.typography.bodySmall, color = MaterialTheme.colorScheme.onSurfaceVariant,
                )
            }
        },
        confirmButton = { TextButton(onClick = actions::confirmRestore, enabled = b.busy == null) { Text("Restore") } },
        dismissButton = { TextButton(onClick = actions::dismissBackup) { Text("Cancel") } },
    )
}

@Composable
private fun RestoreDoneDialog(b: BackupUi, actions: UiActions) {
    AlertDialog(
        onDismissRequest = actions::dismissBackup,
        title = { Text("Restored") },
        text = {
            Text(
                "Your servers and settings are now the ones from the backup. If tailbond was on, it reconnects with them in a moment. " +
                    "Not what you wanted? Undo puts everything back as it was.",
            )
        },
        confirmButton = { TextButton(onClick = actions::dismissBackup) { Text("Done") } },
        dismissButton = { TextButton(onClick = actions::undoRestore, enabled = b.undoAt != null) { Text("Undo restore") } },
    )
}

/** An AlertDialog, or the same drawn in place for renders ([inline]). */
@Composable
private fun DialogFrame(
    inline: Boolean,
    onDismiss: () -> Unit,
    title: @Composable () -> Unit,
    text: @Composable () -> Unit,
    confirmButton: @Composable () -> Unit,
    dismissButton: @Composable () -> Unit,
) {
    if (!inline) {
        AlertDialog(onDismissRequest = onDismiss, title = title, text = text, confirmButton = confirmButton, dismissButton = dismissButton)
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
                ProvideTextStyle(MaterialTheme.typography.bodyMedium) { text() }
                Row(Modifier.fillMaxWidth(), horizontalArrangement = Arrangement.End) { dismissButton(); confirmButton() }
            }
        }
    }
}

/** "2 Nov 2026, 10:00" from the header's RFC 3339 time. */
private fun createdText(created: String): String = runCatching {
    val t = java.time.Instant.parse(created).atZone(java.time.ZoneId.systemDefault())
    t.format(java.time.format.DateTimeFormatter.ofPattern("d MMM yyyy, HH:mm", java.util.Locale.US))
}.getOrDefault(created.ifBlank { "an unknown date" })
