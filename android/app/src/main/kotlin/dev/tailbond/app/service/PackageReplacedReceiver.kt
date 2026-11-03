package dev.tailbond.app.service

import android.content.BroadcastReceiver
import android.content.Context
import android.content.Intent
import android.net.VpnService
import android.util.Log
import androidx.core.content.edit
import dev.tailbond.app.data.Repository

/**
 * Whether the user wants the tunnel on: set when tailbond starts connecting,
 * cleared when the user (or another VPN) turns it off. It survives the
 * process being killed, which is what an app update does.
 */
object TunnelWanted {
    private const val PREFS = "tailbond_tunnel"
    private const val KEY = "wanted_on"

    fun set(ctx: Context, on: Boolean) =
        ctx.getSharedPreferences(PREFS, Context.MODE_PRIVATE).edit { putBoolean(KEY, on) }

    fun get(ctx: Context): Boolean = ctx.getSharedPreferences(PREFS, Context.MODE_PRIVATE).getBoolean(KEY, false)
}

/** Pure: whether to reconnect after this app was updated. Unit tested. */
object UpdateReconnect {
    /**
     * [action]: the broadcast's action; [wantedOn]: the tunnel was on (or
     * connecting) when the update killed the app; [vpnPermission]:
     * VpnService.prepare() returned null (Android still lets tailbond be the
     * VPN, so no dialog is needed); [paired]: at least one server.
     */
    fun shouldReconnect(action: String?, wantedOn: Boolean, vpnPermission: Boolean, paired: Boolean, handedToTailscale: Boolean): Boolean =
        action == Intent.ACTION_MY_PACKAGE_REPLACED && wantedOn && vpnPermission && paired && !handedToTailscale
}

/**
 * After an update Android restarts nothing: the tunnel that ran before is
 * down. Reconnect it if it was on. MY_PACKAGE_REPLACED may start a
 * foreground service.
 */
class PackageReplacedReceiver : BroadcastReceiver() {
    override fun onReceive(context: Context, intent: Intent) {
        if (intent.action != Intent.ACTION_MY_PACKAGE_REPLACED) return
        val repo = Repository.get(context)
        val ok = UpdateReconnect.shouldReconnect(
            action = intent.action,
            wantedOn = TunnelWanted.get(context),
            vpnPermission = VpnService.prepare(context) == null,
            paired = repo.servers.value.isNotEmpty(),
            handedToTailscale = repo.settings.value.handedToTailscale,
        )
        if (!ok) return
        try {
            TailbondVpnService.connect(context)
        } catch (e: Exception) {
            // e.g. Android refused a foreground start: the user turns it on again
            Log.w("tailbond", "reconnect after update failed", e)
        }
    }
}
