package dev.tailbond.app.model

/**
 * Is a Wi-Fi network behind a captive portal (hotel or airport sign-in
 * page)? Android says so with NET_CAPABILITY_CAPTIVE_PORTAL; a network that
 * stays without NET_CAPABILITY_VALIDATED for [graceMs] counts too (partial
 * connectivity, portals Android didn't name). mqvpn's NetworkMonitor only
 * offers validated networks, so such a Wi-Fi is never bonded and joins by
 * itself once Android validates it; this only decides what the UI says.
 */
object Captive {
    const val GRACE_MS = 10_000L

    fun needsSignIn(
        captivePortal: Boolean,
        validated: Boolean,
        unvalidatedSinceMs: Long?,
        nowMs: Long,
        graceMs: Long = GRACE_MS,
    ): Boolean = when {
        validated -> false
        captivePortal -> true
        else -> unvalidatedSinceMs != null && nowMs - unvalidatedSinceMs >= graceMs
    }
}
