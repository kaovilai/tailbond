package dev.tailbond.app.share;

import dev.tailbond.app.share.ITetheringEvents;

/**
 * The privileged half of hotspot control for the Mac's one click
 * (docs/phone-hub.md), served by TetheringShell: a Shizuku user service, a
 * process Shizuku starts from this APK as the shell user, where hidden API
 * checks are off so the framework's tethering and Wi-Fi internals can be
 * called directly. Approach after untether's HotspotShell (BSD-3-Clause)
 * and, through it, supershadoe/delta. Methods that return a String return
 * null on success or the error.
 */
interface ITetheringShell {
    /** Shizuku's reserved transaction, called right before it stops the process. */
    void destroy() = 16777114;

    String open(ITetheringEvents events) = 1;
    /** The system hotspot config's network: [ssid, pass] (pass empty for an open network). */
    String[] config() = 2;
    /** Starts Wi-Fi tethering with the network set in the system config. */
    String start() = 3;
    /** Stops Wi-Fi tethering (whoever started it). */
    String stop() = 4;
    /** The hotspot's state, WifiManager's WIFI_AP_STATE_* (10-14); -1 unknown. */
    int apState() = 5;
}
