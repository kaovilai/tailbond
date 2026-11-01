package dev.tailbond.app.model

import com.mqvpn.sdk.core.model.MqvpnConfig
import com.mqvpn.sdk.core.model.MqvpnConfig.CongestionControl
import com.mqvpn.sdk.core.model.MqvpnConfig.Reinjection
import com.mqvpn.sdk.core.model.MqvpnConfig.ReorderProfile
import com.mqvpn.sdk.core.model.MqvpnConfig.Scheduler

/**
 * User-facing bonding modes. The mqvpn keys each one sets are the contract's
 * mode table (docs/architecture.md, "Modes"); [applyTo] is the only place
 * that table is encoded on Android.
 */
enum class Mode(val wire: String, val label: String, val blurb: String) {
    SPEED("speed", "Speed", "Aggregate bandwidth across every link"),
    STREAMING("streaming", "Streaming", "Calls and streams survive link trouble"),
    REDUNDANT("redundant", "Redundant", "Every UDP packet on every link"),
    FAILOVER("failover", "Failover", "One primary link, others only on failure"),
    ;

    /**
     * [base] with this mode's mqvpn keys applied. Keys a mode row doesn't
     * mention are reset to mqvpn's defaults so switching modes never leaks
     * settings from the previous one.
     *
     * Reorder `Profile = fiber_lte`: mqvpn applies a profile per reorder
     * rule, so speed adds one rule for UDP/443 (QUIC) with that profile.
     *
     * [experimentalBackupFec] only affects [FAILOVER] (scheduler `backup_fec`).
     */
    fun applyTo(base: MqvpnConfig, experimentalBackupFec: Boolean = false): MqvpnConfig {
        val reset = base.copy(
            scheduler = Scheduler.MIN_RTT,
            requestServerScheduler = false,
            cc = CongestionControl.BBR2,
            reorderEnabled = false,
            reorderProfile = ReorderProfile.CELLULAR_BOND,
            reorderPorts = emptyList(),
            redundantEnabled = false,
            redundantPorts = emptyList(),
            hybridEnabled = false,
            reinjection = Reinjection.OFF,
        )
        return when (this) {
            SPEED -> reset.copy(
                scheduler = Scheduler.WLB,
                hybridEnabled = true,
                reorderEnabled = true,
                reorderProfile = ReorderProfile.FIBER_LTE,
                reorderPorts = listOf(QUIC_PORT),
                reinjection = Reinjection.DEADLINE,
                requestServerScheduler = true,
            )
            STREAMING -> reset.copy(
                scheduler = Scheduler.WLB,
                redundantEnabled = true,
                reinjection = Reinjection.IDLE,
                cc = CongestionControl.BBR,
                hybridEnabled = true,
            )
            REDUNDANT -> reset.copy(
                scheduler = Scheduler.MIN_RTT,
                redundantEnabled = true,
                reinjection = Reinjection.IDLE,
                cc = CongestionControl.BBR,
                hybridEnabled = true,
                requestServerScheduler = true,
            )
            FAILOVER -> reset.copy(
                scheduler = if (experimentalBackupFec) Scheduler.BACKUP_FEC else Scheduler.MIN_RTT,
                reinjection = Reinjection.OFF,
            )
        }
    }

    companion object {
        const val QUIC_PORT = 443
        fun fromWire(s: String?): Mode? = entries.firstOrNull { it.wire.equals(s?.trim(), ignoreCase = true) }
    }
}
