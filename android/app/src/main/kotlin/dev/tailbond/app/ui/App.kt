package dev.tailbond.app.ui

import androidx.activity.compose.BackHandler
import androidx.compose.foundation.layout.Arrangement
import androidx.compose.foundation.layout.Column
import androidx.compose.foundation.layout.padding
import androidx.compose.material.icons.Icons
import androidx.compose.material.icons.automirrored.filled.ArrowBack
import androidx.compose.material.icons.filled.Settings
import androidx.compose.material3.AlertDialog
import androidx.compose.material3.ExperimentalMaterial3Api
import androidx.compose.material3.Icon
import androidx.compose.material3.IconButton
import androidx.compose.material3.MaterialTheme
import androidx.compose.material3.Scaffold
import androidx.compose.material3.SnackbarDuration
import androidx.compose.material3.SnackbarHost
import androidx.compose.material3.SnackbarHostState
import androidx.compose.material3.Text
import androidx.compose.material3.TextButton
import androidx.compose.material3.TopAppBar
import androidx.compose.runtime.Composable
import androidx.compose.runtime.CompositionLocalProvider
import androidx.compose.runtime.LaunchedEffect
import androidx.compose.runtime.getValue
import androidx.compose.runtime.mutableStateOf
import androidx.compose.runtime.remember
import androidx.compose.runtime.rememberCoroutineScope
import androidx.compose.runtime.saveable.rememberSaveable
import androidx.compose.runtime.setValue
import androidx.compose.runtime.staticCompositionLocalOf
import androidx.compose.ui.Modifier
import androidx.compose.ui.platform.LocalClipboardManager
import androidx.compose.ui.text.AnnotatedString
import androidx.compose.ui.text.font.FontWeight
import androidx.compose.ui.text.style.TextOverflow
import androidx.compose.ui.unit.dp
import dev.tailbond.app.model.Words
import kotlinx.coroutines.launch

/** Copies a value and confirms it in a snackbar ("Copied nas.tail1234.ts.net"). */
val LocalCopy = staticCompositionLocalOf<(String) -> Unit> { {} }

object Routes {
    const val HOME = "home"
    const val CONNECTIONS = "connections"
    const val SETTINGS = "settings"
    const val APPS = "apps"
    const val WEBSITES = "websites"
    const val PAIR = "pair"
    const val MACHINE = "machine:" // + hostname
}

/**
 * The whole app: onboarding → pairing → home (status, connections, this
 * device, machines), with Connections, Settings, Apps and machine details as
 * pushed pages. [startRoute], [startSearch] (machine search, or the pairing
 * link field) and [startSnackbar] exist for the screen renders.
 */
@OptIn(ExperimentalMaterial3Api::class)
@Composable
fun TailbondApp(
    state: UiState,
    actions: UiActions,
    startRoute: String = Routes.HOME,
    startSearch: String = "",
    startSnackbar: String? = null,
    /** Renders only: a dialog to show open ("add-website", "add-website:Banking"). */
    startDialog: String? = null,
) {
    var route by rememberSaveable { mutableStateOf(startRoute) }
    val snackbar = remember { SnackbarHostState() }
    val scope = rememberCoroutineScope()
    @Suppress("DEPRECATION")
    val clipboard = LocalClipboardManager.current
    val copy: (String) -> Unit = { text ->
        clipboard.setText(AnnotatedString(text))
        scope.launch { snackbar.currentSnackbarData?.dismiss(); snackbar.showSnackbar("Copied $text") }
    }
    LaunchedEffect(startSnackbar) { startSnackbar?.let { snackbar.showSnackbar(it, duration = SnackbarDuration.Indefinite) } }

    val p = state.profile
    val page = when {
        p == null && !state.settings.onboarded -> "onboarding"
        p == null -> Routes.PAIR
        else -> route
    }
    val toPair: (PairTarget?) -> Unit = { t ->
        when (t) {
            PairTarget.Add -> actions.startAddServer()
            is PairTarget.Replace -> actions.startRepair(t.id)
            null -> actions.startRepair(null)
        }
        route = Routes.PAIR
    }
    val back: (() -> Unit)? = when {
        page == Routes.APPS || page == Routes.WEBSITES -> { { route = Routes.SETTINGS } }
        page == "onboarding" || page == Routes.HOME -> null
        page == Routes.PAIR && p == null -> null
        page == Routes.PAIR && state.pairTarget != null -> { { actions.cancelAsk(); actions.clearPairTarget(); route = Routes.SETTINGS } }
        else -> { { route = Routes.HOME } }
    }
    if (back != null) BackHandler(onBack = back)

    CompositionLocalProvider(LocalCopy provides copy) {
        Scaffold(
            topBar = {
                when (page) {
                    "onboarding" -> {}
                    Routes.HOME -> TopAppBar(
                        title = {
                            Column {
                                Text(state.serverName, fontWeight = FontWeight.SemiBold, maxLines = 1, overflow = TextOverflow.Ellipsis)
                                Text(
                                    listOfNotNull(
                                        if (state.servers.size > 1 && state.activeServer?.id != state.servers.first().id) "via backup server" else "via home server",
                                        state.tailnet,
                                    ).joinToString(" · "),
                                    style = MaterialTheme.typography.bodySmall,
                                    color = MaterialTheme.colorScheme.onSurfaceVariant,
                                    maxLines = 1, overflow = TextOverflow.Ellipsis,
                                )
                            }
                        },
                        actions = {
                            IconButton(onClick = { route = Routes.SETTINGS }) { Icon(Icons.Filled.Settings, "Settings") }
                        },
                    )
                    else -> TopAppBar(
                        title = {
                            Text(
                                when {
                                    page == Routes.CONNECTIONS -> "Connections"
                                    page == Routes.SETTINGS -> "Settings"
                                    page == Routes.APPS -> "Apps that skip tailbond"
                                    page == Routes.WEBSITES -> "Websites that skip tailbond"
                                    page == Routes.PAIR && state.pairTarget == PairTarget.Add -> "Add another server"
                                    page == Routes.PAIR && state.pairTarget is PairTarget.Replace -> "Pair again"
                                    page == Routes.PAIR -> "Add this phone"
                                    page.startsWith(Routes.MACHINE) -> page.removePrefix(Routes.MACHINE)
                                    else -> "tailbond"
                                },
                                maxLines = 1, overflow = TextOverflow.Ellipsis,
                            )
                        },
                        navigationIcon = {
                            if (back != null) IconButton(onClick = back) { Icon(Icons.AutoMirrored.Filled.ArrowBack, "Back") }
                        },
                    )
                }
            },
            snackbarHost = { SnackbarHost(snackbar) },
        ) { pad ->
            val m = Modifier.padding(pad)
            when {
                page == "onboarding" -> OnboardingScreen(actions::finishOnboarding, m)
                page == Routes.PAIR -> PairingScreen(state, actions, m, startText = startSearch)
                page == Routes.CONNECTIONS -> ConnectionsScreen(state, actions, m)
                page == Routes.SETTINGS -> SettingsScreen(
                    state, actions,
                    onApps = { route = Routes.APPS },
                    onWebsites = { route = Routes.WEBSITES },
                    onRepair = { id -> toPair(PairTarget.Replace(id)) },
                    onAddServer = { toPair(PairTarget.Add) },
                    modifier = m,
                )
                page == Routes.APPS -> AppsScreen(state, actions, m)
                page == Routes.WEBSITES -> WebsitesScreen(state, actions, onApps = { route = Routes.APPS }, modifier = m, startDialog = startDialog)
                page.startsWith(Routes.MACHINE) -> MachineScreen(state, page.removePrefix(Routes.MACHINE), m)
                else -> HomeScreen(
                    state, actions,
                    onConnections = { route = Routes.CONNECTIONS },
                    onSettings = { route = Routes.SETTINGS },
                    onRepair = { toPair(null) },
                    onMachine = { route = Routes.MACHINE + it },
                    startSearch = startSearch,
                    modifier = m,
                )
            }
        }

        state.pending?.let {
            val target = state.pendingTarget
            ConfirmPairingDialog(
                it, target = target, servers = state.servers,
                onConfirm = { actions.confirmPairing(); route = if (target == PairTarget.Add) Routes.SETTINGS else Routes.HOME },
                onDismiss = actions::dismissPairing,
            )
        }
        state.serverProblem?.let { (name, check) ->
            Words.addProblem(name, check)?.let { (title, body) ->
                AlertDialog(
                    onDismissRequest = actions::dismissServerProblem,
                    title = { Text(title) },
                    text = { Text(body) },
                    confirmButton = { TextButton(onClick = actions::dismissServerProblem) { Text("OK") } },
                )
            }
        }
        state.handoff?.let { HandoffDialog(it, state, actions) }
    }
}

/** Shown before Android's own VPN dialog: what happens to Tailscale. */
@Composable
fun HandoffDialog(kind: Handoff, state: UiState, actions: UiActions) {
    val server = state.serverName
    AlertDialog(
        onDismissRequest = actions::dismissHandoff,
        title = {
            Text(if (kind == Handoff.OTHER_VPN_ACTIVE) "Switch from Tailscale to tailbond?" else "Allow tailbond's VPN")
        },
        text = {
            Column(verticalArrangement = Arrangement.spacedBy(10.dp)) {
                if (kind == Handoff.OTHER_VPN_ACTIVE) {
                    Text("Android runs one VPN at a time. Turning on tailbond turns off Tailscale (or any other VPN) on this phone.")
                } else {
                    Text("Android runs one VPN at a time. If Tailscale is on, turning on tailbond turns it off.")
                }
                Text("You still reach your Tailscale machines, like your NAS, through $server. Your internet traffic goes out from $server too (like an exit node; Settings changes that).")
                Text("Next, Android asks you to allow tailbond's connection request. Tap OK.")
                Text(
                    "If tailbond switches itself off right after this, Tailscale is set as Always-on VPN on this phone: turn that off in Android Settings → VPN → Tailscale's gear (or ask whoever set up the phone). Android doesn't let tailbond check this setting.",
                    style = MaterialTheme.typography.bodySmall,
                )
                Text(
                    "To go back later, tap Switch back to Tailscale on tailbond's main screen.",
                    style = MaterialTheme.typography.bodySmall,
                    color = MaterialTheme.colorScheme.onSurfaceVariant,
                )
            }
        },
        confirmButton = { TextButton(onClick = actions::confirmHandoff) { Text("Turn on tailbond") } },
        dismissButton = { TextButton(onClick = actions::dismissHandoff) { Text("Cancel") } },
    )
}
