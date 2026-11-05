package dev.tailbond.app.ui

import android.content.ClipData
import android.content.ActivityNotFoundException
import android.content.ClipboardManager
import android.content.Intent
import android.net.Uri
import android.widget.Toast
import androidx.activity.compose.rememberLauncherForActivityResult
import androidx.activity.result.contract.ActivityResultContracts
import androidx.compose.foundation.clickable
import androidx.compose.foundation.layout.Arrangement
import androidx.compose.foundation.layout.Column
import androidx.compose.foundation.layout.FlowRow
import androidx.compose.foundation.layout.ExperimentalLayoutApi
import androidx.compose.foundation.layout.Row
import androidx.compose.foundation.layout.fillMaxWidth
import androidx.compose.foundation.selection.toggleable
import androidx.compose.material3.minimumInteractiveComponentSize
import androidx.compose.ui.semantics.Role
import androidx.compose.foundation.layout.padding
import androidx.compose.foundation.rememberScrollState
import androidx.compose.foundation.text.KeyboardOptions
import androidx.compose.foundation.verticalScroll
import androidx.compose.material.icons.Icons
import androidx.compose.material.icons.automirrored.filled.KeyboardArrowRight
import androidx.compose.material.icons.filled.BugReport
import androidx.compose.material3.Button
import androidx.compose.material3.Card
import androidx.compose.material3.CardDefaults
import androidx.compose.material3.Checkbox
import androidx.compose.material3.Icon
import androidx.compose.material3.ListItem
import androidx.compose.material3.MaterialTheme
import androidx.compose.material3.OutlinedButton
import androidx.compose.material3.OutlinedTextField
import androidx.compose.material3.Text
import androidx.compose.material3.TextButton
import androidx.compose.runtime.Composable
import androidx.compose.runtime.LaunchedEffect
import androidx.compose.runtime.getValue
import androidx.compose.runtime.mutableStateOf
import androidx.compose.runtime.remember
import androidx.compose.runtime.rememberCoroutineScope
import androidx.compose.runtime.saveable.rememberSaveable
import androidx.compose.runtime.setValue
import androidx.compose.ui.Alignment
import androidx.compose.ui.Modifier
import androidx.compose.ui.platform.LocalContext
import androidx.compose.ui.platform.testTag
import androidx.compose.ui.text.font.FontFamily
import androidx.compose.ui.text.input.KeyboardCapitalization
import androidx.compose.ui.unit.dp
import androidx.compose.ui.unit.sp
import androidx.core.net.toUri
import dev.tailbond.app.report.CrashCapture
import dev.tailbond.app.report.Redactor
import dev.tailbond.app.report.Report
import dev.tailbond.app.report.ReportText
import kotlinx.coroutines.Dispatchers
import kotlinx.coroutines.launch
import kotlinx.coroutines.withContext
import java.text.DateFormat
import java.time.Instant
import java.util.Date

/** Settings → Report a problem. */
@Composable
fun ReportRow(onReport: () -> Unit) {
    SectionHeader("Help")
    ListItem(
        headlineContent = { Text("Report a problem") },
        supportingContent = { Text("Make a GitHub issue with this phone's details; you check it before anything is public") },
        leadingContent = { Icon(Icons.Filled.BugReport, null) },
        trailingContent = { Icon(Icons.AutoMirrored.Filled.KeyboardArrowRight, null) },
        modifier = Modifier.clickable(onClick = onReport).testTag("report-row"),
    )
}

/** Home: tailbond crashed last time; offer to report it. */
@Composable
fun CrashCard(onReport: () -> Unit) {
    val ctx = LocalContext.current
    var crash by remember { mutableStateOf(CrashCapture.last(ctx)) }
    val c = crash ?: return
    Card(
        colors = CardDefaults.cardColors(containerColor = MaterialTheme.colorScheme.errorContainer),
        modifier = Modifier.fillMaxWidth().padding(horizontal = 16.dp, vertical = 8.dp).testTag("crash-card"),
    ) {
        Column(Modifier.padding(16.dp), verticalArrangement = Arrangement.spacedBy(8.dp)) {
            Text("tailbond stopped unexpectedly", style = MaterialTheme.typography.titleSmall)
            Text(
                "It happened ${DateFormat.getDateTimeInstance(DateFormat.SHORT, DateFormat.SHORT).format(Date(c.second))}. Reporting it helps fix it.",
                style = MaterialTheme.typography.bodyMedium,
            )
            Row(horizontalArrangement = Arrangement.spacedBy(8.dp)) {
                Button(onClick = onReport) { Text("Report a problem") }
                TextButton(onClick = { CrashCapture.dismiss(ctx); crash = null }) { Text("Dismiss") }
            }
        }
    }
}

/**
 * The report preview (docs/reporting.md): redacted, editable; the notice;
 * a required checkbox; then GitHub in the browser, the full text to the
 * clipboard, or a file. Nothing is sent by the app.
 */
@OptIn(ExperimentalLayoutApi::class)
@Composable
fun ReportScreen(modifier: Modifier = Modifier) {
    val ctx = LocalContext.current
    val scope = rememberCoroutineScope()
    val crashed = remember { CrashCapture.last(ctx) }
    var what by rememberSaveable { mutableStateOf("") }
    var steps by rememberSaveable { mutableStateOf("") }
    var version by rememberSaveable { mutableStateOf(CrashCapture.version()) }
    var device by rememberSaveable { mutableStateOf(Redactor.redact(CrashCapture.device())) }
    var crash by rememberSaveable { mutableStateOf(Redactor.redact(crashed?.first.orEmpty())) }
    var logs by rememberSaveable { mutableStateOf<String?>(null) }
    var ok by rememberSaveable { mutableStateOf(false) }
    LaunchedEffect(Unit) {
        if (logs == null) logs = Redactor.redact(withContext(Dispatchers.IO) { CrashCapture.recentLogs() })
    }
    fun report() = Report("Android app", version, device, what, steps, logs.orEmpty(), crash)
    val save = rememberLauncherForActivityResult(ActivityResultContracts.CreateDocument("text/plain")) { uri: Uri? ->
        if (uri != null) scope.launch {
            val done = withContext(Dispatchers.IO) {
                runCatching { ctx.contentResolver.openOutputStream(uri)?.use { it.write(report().full().toByteArray()) } }.isSuccess
            }
            Toast.makeText(ctx, if (done) "Report saved" else "Couldn't save the report", Toast.LENGTH_SHORT).show()
        }
    }
    // a TV may have no browser or file picker: say so instead of crashing
    fun open(i: Intent, what: String): Boolean = try {
        ctx.startActivity(i)
        true
    } catch (_: ActivityNotFoundException) {
        Toast.makeText(ctx, "This device has no $what. Use Copy full report instead.", Toast.LENGTH_LONG).show()
        false
    }
    val mono = MaterialTheme.typography.bodySmall.copy(fontFamily = FontFamily.Monospace, fontSize = 11.sp)
    Column(modifier.verticalScroll(rememberScrollState()).padding(16.dp), verticalArrangement = Arrangement.spacedBy(12.dp)) {
        Card(colors = CardDefaults.cardColors(containerColor = MaterialTheme.colorScheme.surfaceContainerHigh)) {
            Text(ReportText.NOTICE, style = MaterialTheme.typography.bodyMedium, modifier = Modifier.padding(12.dp).testTag("report-notice"))
        }
        Text(
            "A security problem? Don't open a public issue: report it privately.",
            style = MaterialTheme.typography.bodySmall,
            color = MaterialTheme.colorScheme.onSurfaceVariant,
        )
        TextButton(onClick = { open(Intent(Intent.ACTION_VIEW, ReportText.SECURITY_URL.toUri()), "browser") }) {
            Text("Report a vulnerability privately")
        }
        if (crashed != null) {
            Text(
                "tailbond stopped unexpectedly (${DateFormat.getDateTimeInstance(DateFormat.SHORT, DateFormat.SHORT).format(Date(crashed.second))}); its crash details are below.",
                style = MaterialTheme.typography.bodySmall,
                color = MaterialTheme.colorScheme.error,
            )
        }
        OutlinedTextField(
            value = what, onValueChange = { what = it },
            label = { Text("What happened, and what did you expect?") },
            minLines = 3,
            keyboardOptions = KeyboardOptions(capitalization = KeyboardCapitalization.Sentences),
            modifier = Modifier.fillMaxWidth().testTag("report-what"),
        )
        OutlinedTextField(value = steps, onValueChange = { steps = it }, label = { Text("Steps to reproduce (optional)") }, minLines = 2, modifier = Modifier.fillMaxWidth())
        OutlinedTextField(value = version, onValueChange = { version = it }, label = { Text("tailbond version") }, singleLine = true, modifier = Modifier.fillMaxWidth())
        OutlinedTextField(value = device, onValueChange = { device = it }, label = { Text("Phone and Android version") }, singleLine = true, modifier = Modifier.fillMaxWidth())
        if (crashed != null) {
            OutlinedTextField(value = crash, onValueChange = { crash = it }, label = { Text("Crash details") }, textStyle = mono, minLines = 4, maxLines = 12, modifier = Modifier.fillMaxWidth())
        }
        OutlinedTextField(
            value = logs ?: "Reading this app's log…", onValueChange = { logs = it },
            label = { Text("Logs (newest last)") },
            textStyle = mono, minLines = 6, maxLines = 14, enabled = logs != null,
            supportingText = { Text("A link holds about 7 KB, so the issue gets the newest lines; Copy full report and Save report file have all of them.") },
            modifier = Modifier.fillMaxWidth().testTag("report-logs"),
        )
        // one focus stop for the remote: the row is the checkbox
        Row(
            verticalAlignment = Alignment.CenterVertically,
            modifier = Modifier.toggleable(value = ok, role = Role.Checkbox, onValueChange = { ok = it }).testTag("report-confirm"),
        ) {
            Checkbox(checked = ok, onCheckedChange = null, modifier = Modifier.minimumInteractiveComponentSize())
            Text(ReportText.CONFIRM, style = MaterialTheme.typography.bodyMedium)
        }
        if (!(ok && what.isNotBlank())) {
            Text(
                if (what.isBlank()) "To open the issue: say what happened (the first box), then tick the box above."
                else "To open the issue: tick the box above once you have read the report.",
                style = MaterialTheme.typography.bodySmall,
                color = MaterialTheme.colorScheme.onSurfaceVariant,
                modifier = Modifier.testTag("report-why"),
            )
        }
        FlowRow(horizontalArrangement = Arrangement.spacedBy(8.dp)) {
            Button(
                onClick = {
                    if (open(Intent(Intent.ACTION_VIEW, report().issueUrl().toUri()), "browser") && crashed != null) CrashCapture.dismiss(ctx)
                },
                enabled = ok && what.isNotBlank() && logs != null,
                modifier = Modifier.testTag("report-open"),
            ) { Text("Open GitHub issue") }
            OutlinedButton(onClick = {
                val cm = ctx.getSystemService(ClipboardManager::class.java)
                cm?.setPrimaryClip(ClipData.newPlainText("tailbond report", report().full()))
                Toast.makeText(ctx, "Full report copied", Toast.LENGTH_SHORT).show()
            }) { Text("Copy full report") }
            OutlinedButton(onClick = {
                try {
                    save.launch(Report.fileName(Instant.now().toString().substringBefore('.') + "Z"))
                } catch (_: ActivityNotFoundException) {
                    Toast.makeText(ctx, "This device has no file picker. Use Copy full report instead.", Toast.LENGTH_LONG).show()
                }
            }) { Text("Save report file") }
        }
        Text(ReportText.ACCOUNT_NOTE, style = MaterialTheme.typography.bodySmall, color = MaterialTheme.colorScheme.onSurfaceVariant)
    }
}
