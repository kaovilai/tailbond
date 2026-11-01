package dev.tailbond.app.service

import android.content.Context
import android.net.ConnectivityManager
import android.net.Network
import android.net.NetworkCapabilities
import android.net.NetworkRequest
import android.os.Handler
import android.os.Looper
import androidx.core.content.getSystemService
import dev.tailbond.app.model.Captive
import kotlinx.coroutines.flow.MutableStateFlow
import kotlinx.coroutines.flow.StateFlow
import kotlinx.coroutines.flow.asStateFlow

/**
 * Watches Wi-Fi networks (validated or not) and publishes whether one
 * needs a sign-in ([Captive]). Android's own captive portal login app talks
 * to that network directly (it binds to the Wi-Fi network, not the default
 * route), so tailbond's VPN doesn't stand in its way; it only must not run
 * with "Block connections without VPN" (docs/android.md).
 */
class CaptiveWatcher(context: Context) {
    private val cm = context.getSystemService<ConnectivityManager>()
    private val handler = Handler(Looper.getMainLooper())
    private val nets = HashMap<Network, Pair<NetworkCapabilities, Long?>>()
    private var registered = false

    private val callback = object : ConnectivityManager.NetworkCallback() {
        override fun onCapabilitiesChanged(network: Network, caps: NetworkCapabilities) {
            handler.post {
                val validated = caps.hasCapability(NetworkCapabilities.NET_CAPABILITY_VALIDATED)
                val since = if (validated) null else nets[network]?.second ?: System.currentTimeMillis()
                nets[network] = caps to since
                publish()
                if (!validated) handler.postDelayed({ publish() }, Captive.GRACE_MS + 500)
            }
        }

        override fun onLost(network: Network) {
            handler.post { nets.remove(network); publish() }
        }
    }

    private fun publish() {
        val now = System.currentTimeMillis()
        _wifiSignIn.value = nets.values.any { (caps, since) ->
            Captive.needsSignIn(
                captivePortal = caps.hasCapability(NetworkCapabilities.NET_CAPABILITY_CAPTIVE_PORTAL),
                validated = caps.hasCapability(NetworkCapabilities.NET_CAPABILITY_VALIDATED),
                unvalidatedSinceMs = since,
                nowMs = now,
            )
        }
    }

    fun start() {
        if (registered || cm == null) return
        val req = NetworkRequest.Builder()
            .addTransportType(NetworkCapabilities.TRANSPORT_WIFI)
            .removeCapability(NetworkCapabilities.NET_CAPABILITY_NOT_VPN)
            .build()
        try {
            cm.registerNetworkCallback(req, callback)
            registered = true
        } catch (_: RuntimeException) {
        }
    }

    fun stop() {
        if (!registered) return
        try { cm?.unregisterNetworkCallback(callback) } catch (_: RuntimeException) {}
        registered = false
        nets.clear()
        _wifiSignIn.value = false
    }

    companion object {
        private val _wifiSignIn = MutableStateFlow(false)
        /** A Wi-Fi network is up but wants a sign-in first. */
        val wifiSignIn: StateFlow<Boolean> = _wifiSignIn.asStateFlow()
    }
}
