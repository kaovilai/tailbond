package dev.tailbond.app.share;

/** From TetheringShell back to the app (one way). */
oneway interface ITetheringEvents {
    void onLog(String line) = 1;
    /** startTethering was refused (its callback's error code), or the AP failed to come up (-1). */
    void onFailed(int error) = 2;
    /** The network in the system hotspot config (pass empty for an open network). */
    void onNetwork(String ssid, String pass) = 3;
}
