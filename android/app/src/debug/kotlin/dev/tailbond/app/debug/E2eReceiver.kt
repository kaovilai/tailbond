package dev.tailbond.app.debug

import android.content.BroadcastReceiver
import android.content.Context
import android.content.Intent
import android.util.Log
import dev.tailbond.app.data.Repository
import dev.tailbond.app.pairing.Pairing
import dev.tailbond.app.pairing.PairingResult
import dev.tailbond.app.service.LiveState
import kotlinx.serialization.json.JsonPrimitive
import kotlinx.serialization.json.buildJsonArray
import kotlinx.serialization.json.buildJsonObject
import kotlinx.serialization.json.put

/**
 * Debug builds only (src/debug): lets the emulator end-to-end test
 * (scripts/e2e-android.sh) read the app's state over adb. Guarded by
 * android.permission.DUMP, which adb's shell holds and apps don't.
 *
 *   adb shell am broadcast -n dev.tailbond.app/.debug.E2eReceiver -a dev.tailbond.app.debug.STATUS
 *     → result data: {"state":"CONNECTED","tunnel_ip":…,"paths":[…],"servers":1,"peers":{…}}
 *   … -a dev.tailbond.app.debug.PAIR --es uri 'tailbond://pair?d=…'
 *     → stores the pairing as the "Add phone" button does (fallback when the
 *       confirm dialog can't be tapped)
 */
class E2eReceiver : BroadcastReceiver() {
    override fun onReceive(context: Context, intent: Intent) {
        val repo = Repository.get(context)
        when (intent.action) {
            ACTION_PAIR -> {
                val uri = intent.getStringExtra("uri").orEmpty()
                when (val r = Pairing.parse(uri)) {
                    is PairingResult.Ok -> {
                        repo.saveProfile(r.payload)
                        repo.updateSettings { it.copy(mode = null, onboarded = true) }
                        resultData = "paired"
                    }
                    is PairingResult.Invalid -> resultData = "invalid: ${r.reason}"
                }
            }
            ACTION_STATUS -> {
                val live = LiveState.live.value
                val peers = repo.peers.value
                val json = buildJsonObject {
                    put("state", live.state.name)
                    put("message", live.message)
                    put("error_kind", live.errorKind?.name)
                    put("tunnel_ip", live.tunnelIp)
                    put("tx_bytes", live.txBytes)
                    put("rx_bytes", live.rxBytes)
                    put("paths", buildJsonArray {
                        for (p in live.paths) add(buildJsonObject {
                            put("name", p.name)
                            put("kind", p.kind?.name)
                            put("status", p.status)
                            put("tx_bytes", p.txBytes)
                            put("rx_bytes", p.rxBytes)
                        })
                    })
                    put("servers", repo.servers.value.size)
                    put("server_names", buildJsonArray { repo.servers.value.forEach { add(JsonPrimitive(it.name)) } })
                    put("mode", repo.settings.value.mode?.name)
                    put("peers", if (peers == null) null else buildJsonObject {
                        put("server", peers.server)
                        put("count", peers.peers.size)
                        put("fetched_at_ms", peers.fetchedAtMs)
                        put("names", buildJsonArray { peers.peers.forEach { add(JsonPrimitive(it.hostname)) } })
                    })
                }.toString()
                Log.i(TAG, "status $json")
                resultData = json
            }
        }
    }

    companion object {
        private const val TAG = "tailbond-e2e"
        const val ACTION_STATUS = "dev.tailbond.app.debug.STATUS"
        const val ACTION_PAIR = "dev.tailbond.app.debug.PAIR"
    }
}
