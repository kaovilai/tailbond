package dev.tailbond.app

import android.content.Intent
import dev.tailbond.app.service.UpdateReconnect
import org.junit.Assert.assertFalse
import org.junit.Assert.assertTrue
import org.junit.Test

class UpdateReconnectTest {
    private fun decide(
        action: String? = Intent.ACTION_MY_PACKAGE_REPLACED,
        wantedOn: Boolean = true,
        vpnPermission: Boolean = true,
        paired: Boolean = true,
        handedToTailscale: Boolean = false,
    ) = UpdateReconnect.shouldReconnect(action, wantedOn, vpnPermission, paired, handedToTailscale)

    @Test fun reconnectsWhenItWasOn() = assertTrue(decide())

    @Test fun staysOffWhenTheUserTurnedItOff() = assertFalse(decide(wantedOn = false))

    @Test fun neverAsksForVpnPermissionByItself() = assertFalse(decide(vpnPermission = false))

    @Test fun nothingToConnectTo() = assertFalse(decide(paired = false))

    @Test fun respectsSwitchBackToTailscale() = assertFalse(decide(handedToTailscale = true))

    @Test fun onlyAfterThisAppsOwnUpdate() {
        assertFalse(decide(action = Intent.ACTION_PACKAGE_REPLACED))
        assertFalse(decide(action = Intent.ACTION_BOOT_COMPLETED))
        assertFalse(decide(action = null))
    }
}
