package dev.tailbond.app.ui

import dev.tailbond.app.backup.BackupHeader
import dev.tailbond.app.backup.RestorePreview
import dev.tailbond.app.data.AskState
import dev.tailbond.app.update.UpdateChannel
import dev.tailbond.app.update.UpdatePolicy
import dev.tailbond.app.update.UpdateStatus
import dev.tailbond.app.data.PeerList
import dev.tailbond.app.failover.FailoverPolicy
import dev.tailbond.app.model.AddCheck
import dev.tailbond.app.model.AppSettings
import dev.tailbond.app.model.ServerProfile
import dev.tailbond.app.model.SwitchRecord
import dev.tailbond.app.model.Mode
import dev.tailbond.app.model.PathKind
import dev.tailbond.app.model.PathSetting
import dev.tailbond.app.model.RouteMode
import dev.tailbond.app.model.TunnelPlan
import dev.tailbond.app.model.serverShortName
import dev.tailbond.app.model.tailnetDomain
import dev.tailbond.app.pairing.PairingPayload
import dev.tailbond.app.service.ConnState
import dev.tailbond.app.service.Live

/** The Updates page: the updater's status plus what Android says about installing. */
data class UpdateUi(
    val status: UpdateStatus = UpdateStatus(current = "0.1.0"),
    /** "Install unknown apps" is allowed for tailbond. */
    val canInstallPackages: Boolean = true,
    /** Signed with the debug key: can't take release updates. */
    val debugBuild: Boolean = false,
    /** Fixed clock for renders; 0 = now. */
    val nowMs: Long = 0,
)

/** Backup and restore: what is happening, and the restore steps (pick → passphrase → preview → done). */
data class BackupUi(
    /** Work in progress ("Encrypting…"); dialogs show it and disable their buttons. */
    val busy: String? = null,
    /** The last result ("Saved tailbond-pixel-8-2026-11-02.tbbackup"). */
    val message: String? = null,
    val error: String? = null,
    /** A file picked for restore, waiting for its passphrase. */
    val picked: BackupHeader? = null,
    /** What restoring it would change, waiting for "Restore". */
    val preview: RestorePreview? = null,
    /** Just restored (the done dialog with "Undo restore"). */
    val restoredAt: Long? = null,
    /** While "Undo restore" is possible: when the restore happened. */
    val undoAt: Long? = null,
)

/** Machine list state: the last list we got, plus whether a refresh failed. */
data class PeersUi(val list: PeerList? = null, val loading: Boolean = false, val error: String? = null)

/** Why the in-app "VPN handoff" explanation is showing. */
enum class Handoff {
    /** Another VPN (normally Tailscale) is running and will be turned off. */
    OTHER_VPN_ACTIVE,
    /** First connect: Android will ask for VPN permission next. */
    FIRST_PERMISSION,
}

/** Where a confirmed pairing goes. */
sealed interface PairTarget {
    /** "Add another server": appended as a backup. */
    data object Add : PairTarget
    /** "Pair again" for this server (or a deep link for a server already in the list). */
    data class Replace(val id: String) : PairTarget
}

/** Everything the screens render. Built by MainActivity from the view model; faked in screen renders. */
data class UiState(
    val profile: PairingPayload? = null,
    val settings: AppSettings = AppSettings(),
    val live: Live = Live(),
    val cellularBytes: Long = 0,
    val peers: PeersUi = PeersUi(),
    val pending: PairingPayload? = null,
    val pairError: String? = null,
    val handoff: Handoff? = null,
    val tailscaleInstalled: Boolean = false,
    val apps: List<AppEntry> = emptyList(),
    /** "Ask to connect": a pair request to the home server over Tailscale. */
    val ask: AskState = AskState.Idle,
    /** A Wi-Fi network wants a sign-in (captive portal). */
    val wifiSignIn: Boolean = false,
    /** The ordered server list (first = main); [profile] is the main one's pairing. */
    val servers: List<ServerProfile> = emptyList(),
    /** What the pairing screen and confirm dialog are for; null = first pairing (or decided by the link). */
    val pairTarget: PairTarget? = null,
    /** Where [pending] goes once confirmed (null = first pairing). */
    val pendingTarget: PairTarget? = null,
    /** Tailnet machines that answered as tailbond servers (one-tap "Add another server"). */
    val serverSuggestions: List<String> = emptyList(),
    /** A server that can't be added: its name and why. */
    val serverProblem: Pair<String, AddCheck>? = null,
    /** The last server switches, newest first. */
    val switches: List<SwitchRecord> = emptyList(),
    /** Build.VERSION.SDK_INT (renders fake older phones). */
    val sdkInt: Int = 35,
    val update: UpdateUi = UpdateUi(),
    val backup: BackupUi = BackupUi(),
) {
    /** The server the tunnel uses (or will use): failover's active one, else the main one. */
    val activeServer: ServerProfile?
        get() {
            val on = live.state == ConnState.CONNECTED || live.state == ConnState.CONNECTING || live.state == ConnState.RECONNECTING
            return (if (on) servers.firstOrNull { it.id == live.failover?.activeId } else null) ?: servers.firstOrNull()
        }

    val serverNames: Map<String, String> get() = servers.associate { it.id to it.name }

    val mode: Mode get() = profile?.let { TunnelPlan.effectiveMode(it, settings) } ?: (settings.mode ?: Mode.SPEED)

    /** "homebox": the server's tailnet host name when known, else from the pairing. */
    val serverName: String
        get() = if (servers.size > 1) activeServer?.name ?: "home server"
        else peers.list?.server?.takeIf { it.isNotBlank() } ?: profile?.let { serverShortName(it) } ?: "home server"

    /** "tail1234.ts.net" */
    val tailnet: String?
        get() = peers.list?.magicDnsSuffix?.takeIf { it.isNotBlank() } ?: profile?.let { tailnetDomain(it) }

    val capReached: Boolean
        get() = settings.cellularCapMb > 0 && cellularBytes >= settings.cellularCapMb * 1_000_000
}

/** What the screens can ask for. Default no-ops so renders and previews need nothing. */
interface UiActions {
    fun setOn(on: Boolean) {}
    fun retry() {}
    fun finishOnboarding() {}
    fun offerPairing(text: String): Boolean = false
    fun confirmPairing() {}
    fun dismissPairing() {}
    fun clearPairError() {}
    /** Asks the home server (its Tailscale name) to pair this phone; someone approves it there. */
    fun askToConnect(server: String) {}
    fun cancelAsk() {}
    /** Opens Android's Wi-Fi panel, where the network's sign-in page opens. */
    fun openWifiSignIn() {}
    fun setCaptivePrompts(on: Boolean) {}
    /** Opens http://<server>:8077/#add in the browser (works while Tailscale is still on). */
    fun openHomeServerPage(server: String) {}
    fun forgetPairing() {}
    fun setMode(m: Mode) {}
    fun setPath(kind: PathKind, s: PathSetting) {}
    fun setRouteMode(r: RouteMode) {}
    fun setCapMb(mb: Long) {}
    fun setBackupFec(on: Boolean) {}
    fun resetCellular() {}
    fun setExcluded(pkg: String, excluded: Boolean) {}
    fun loadApps() {}
    fun refreshPeers() {}
    fun confirmHandoff() {}
    fun dismissHandoff() {}
    /** Stops tailbond and opens the Tailscale app (or its store page). */
    fun switchToTailscale() {}
    fun openVpnSettings() {}
    /** Opens the pairing screen to add a backup server. */
    fun startAddServer() {}
    /** Opens the pairing screen to pair [id] again (null: the server in use). */
    fun startRepair(id: String?) {}
    fun useServerNow(id: String) {}
    /** "Go back to automatic" after picking a server by hand. */
    fun resumeAutomatic() {}
    fun moveServer(id: String, delta: Int) {}
    fun removeServer(id: String) {}
    fun setFailoverPolicy(policy: FailoverPolicy, server: String?) {}
    fun dismissServerProblem() {}
    /** Leaving the pairing screen without pairing. */
    fun clearPairTarget() {}
    fun addWebsite(domain: String, note: String) {}
    fun setWebsiteEnabled(domain: String, on: Boolean) {}
    fun removeWebsite(domain: String) {}
    // updates
    fun checkUpdates() {}
    fun installUpdate() {}
    /** Android's "Update this app?" prompt again (it went away, or the user came from the notification). */
    fun finishUpdate() {}
    /** Android's "Install unknown apps" page for tailbond. */
    fun allowInstalls() {}
    fun setUpdatePolicy(p: UpdatePolicy) {}
    fun setUpdateChannel(c: UpdateChannel) {}
    fun openUrl(url: String) {}
    // backup
    /** Asks where to save, then writes the encrypted file. */
    fun exportBackup(passphrase: String) {}
    /** Opens the file picker for a restore. */
    fun pickRestoreFile() {}
    fun openRestore(passphrase: String) {}
    fun confirmRestore() {}
    fun undoRestore() {}
    fun pushBackup(passphrase: String) {}
    /** Closes whatever backup dialog is open. */
    fun dismissBackup() {}
}

object NoActions : UiActions
