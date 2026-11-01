package dev.tailbond.app.ui

import dev.tailbond.app.data.AskState
import dev.tailbond.app.data.PeerList
import dev.tailbond.app.model.AppSettings
import dev.tailbond.app.model.Mode
import dev.tailbond.app.model.PathKind
import dev.tailbond.app.model.PathSetting
import dev.tailbond.app.model.RouteMode
import dev.tailbond.app.model.TunnelPlan
import dev.tailbond.app.model.serverShortName
import dev.tailbond.app.model.tailnetDomain
import dev.tailbond.app.pairing.PairingPayload
import dev.tailbond.app.service.Live

/** Machine list state: the last list we got, plus whether a refresh failed. */
data class PeersUi(val list: PeerList? = null, val loading: Boolean = false, val error: String? = null)

/** Why the in-app "VPN handoff" explanation is showing. */
enum class Handoff {
    /** Another VPN (normally Tailscale) is running and will be turned off. */
    OTHER_VPN_ACTIVE,
    /** First connect: Android will ask for VPN permission next. */
    FIRST_PERMISSION,
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
) {
    val mode: Mode get() = profile?.let { TunnelPlan.effectiveMode(it, settings) } ?: (settings.mode ?: Mode.SPEED)

    /** "homebox": the server's tailnet host name when known, else from the pairing. */
    val serverName: String
        get() = peers.list?.server?.takeIf { it.isNotBlank() } ?: profile?.let { serverShortName(it) } ?: "home server"

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
}

object NoActions : UiActions
