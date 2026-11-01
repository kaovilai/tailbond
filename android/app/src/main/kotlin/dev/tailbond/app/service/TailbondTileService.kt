package dev.tailbond.app.service

import android.content.Intent
import android.net.VpnService
import android.os.Build
import android.service.quicksettings.Tile
import android.service.quicksettings.TileService
import android.app.PendingIntent
import dev.tailbond.app.R
import dev.tailbond.app.data.Repository
import dev.tailbond.app.ui.MainActivity
import dev.tailbond.app.ui.formatBps
import kotlinx.coroutines.CoroutineScope
import kotlinx.coroutines.Dispatchers
import kotlinx.coroutines.Job
import kotlinx.coroutines.SupervisorJob
import kotlinx.coroutines.cancel
import kotlinx.coroutines.launch

/** Quick-settings tile: shows the tunnel state and the combined rate; tap toggles. */
class TailbondTileService : TileService() {
    private val scope = CoroutineScope(SupervisorJob() + Dispatchers.Main.immediate)
    private var job: Job? = null

    override fun onStartListening() {
        job = scope.launch { LiveState.live.collect { render(it) } }
    }

    override fun onStopListening() {
        job?.cancel()
        job = null
    }

    override fun onDestroy() {
        scope.cancel()
        super.onDestroy()
    }

    private fun render(live: Live) {
        val tile = qsTile ?: return
        val on = live.state == ConnState.CONNECTED || live.state == ConnState.CONNECTING || live.state == ConnState.RECONNECTING
        tile.state = if (on) Tile.STATE_ACTIVE else Tile.STATE_INACTIVE
        tile.label = getString(R.string.app_name)
        if (Build.VERSION.SDK_INT >= 29) {
            tile.subtitle = when (live.state) {
                ConnState.CONNECTED -> formatBps(live.rxBps + live.txBps)
                ConnState.CONNECTING -> getString(R.string.state_connecting)
                ConnState.RECONNECTING -> getString(R.string.state_reconnecting)
                ConnState.ERROR -> getString(R.string.state_error)
                ConnState.DISCONNECTED -> getString(R.string.tile_off)
            }
        }
        tile.updateTile()
    }

    override fun onClick() {
        val state = LiveState.live.value.state
        if (state != ConnState.DISCONNECTED && state != ConnState.ERROR) {
            TailbondVpnService.disconnect(this)
            return
        }
        // Not paired or no VPN permission yet: the app has to ask.
        if (Repository.get(this).profile.value == null || VpnService.prepare(this) != null) {
            openApp()
            return
        }
        try {
            TailbondVpnService.connect(this)
        } catch (_: Exception) {
            openApp() // background-start restrictions: let the activity start it
        }
    }

    private fun openApp() {
        val intent = Intent(this, MainActivity::class.java)
            .addFlags(Intent.FLAG_ACTIVITY_NEW_TASK)
            .setAction(MainActivity.ACTION_CONNECT)
        if (Build.VERSION.SDK_INT >= 34) {
            startActivityAndCollapse(
                PendingIntent.getActivity(this, 0, intent, PendingIntent.FLAG_IMMUTABLE or PendingIntent.FLAG_UPDATE_CURRENT),
            )
        } else {
            @Suppress("DEPRECATION", "StartActivityAndCollapseDeprecated")
            startActivityAndCollapse(intent)
        }
    }
}
