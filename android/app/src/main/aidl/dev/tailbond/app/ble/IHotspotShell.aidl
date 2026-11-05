// Adapted from Untether (https://github.com/ImOstrovskiy/untether),
// android/app/src/main/aidl/.../IHotspotShell.aidl.
// Copyright (c) 2026 Oleksii Ostrovskyi. BSD-3-Clause; the notice is in
// THIRD_PARTY_NOTICES.md at the repository root.
package dev.tailbond.app.ble;

/**
 * The privileged half of hotspot control (HotspotShell, a Shizuku user
 * service running as the shell user). Methods returning a String return
 * null on success, or the error.
 */
interface IHotspotShell {
    /** Shizuku's reserved transaction, called right before it stops the process. */
    void destroy() = 16777114;

    /** Starts Wi-Fi tethering with the hotspot network set in Android's settings. */
    String startTethering() = 1;
    String stopTethering() = 2;
    /** WifiManager's hidden AP state: 10 disabling, 11 disabled, 12 enabling, 13 enabled, 14 failed; -1 unknown. */
    int apState() = 3;
    /** The last startTethering failure (TetheringManager.TETHER_ERROR_*), 0 none. */
    int lastError() = 4;
    /** The hotspot network in Android's settings: [ssid, passphrase], or null. */
    String[] network() = 5;
}
