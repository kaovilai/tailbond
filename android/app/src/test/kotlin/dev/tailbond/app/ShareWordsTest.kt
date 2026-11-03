package dev.tailbond.app

import dev.tailbond.app.share.ShareLive
import dev.tailbond.app.share.ShareRelay
import dev.tailbond.app.share.ShareStatus
import dev.tailbond.app.share.ShareWords
import org.junit.Assert.assertEquals
import org.junit.Assert.assertTrue
import org.junit.Test

class ShareWordsTest {
    private fun hub(n: String, b: Long = 0) = ShareRelay.HubUse(n, "x", b, 0, 0)

    @Test fun notificationSaysWhoAndHowMuch() {
        val l = ShareLive(ShareStatus.SHARING, hubs = listOf(hub("slate7", 120_000_000)))
        assertEquals("Sharing mobile data with slate7: 120.0 MB", ShareWords.notification(l, 20).first)
    }

    @Test fun hubNames() {
        assertEquals("slate7 and laptop", ShareWords.hubNames(listOf(hub("slate7"), hub("laptop"))))
        assertEquals("a, b and 2 more", ShareWords.hubNames(listOf(hub("a"), hub("b"), hub("c"), hub("d"))))
    }

    @Test fun lowBatterySaysHowToResume() {
        val l = ShareLive(ShareStatus.BATTERY, battery = 14)
        assertEquals("Paused: battery low (14%)", ShareWords.headline(l))
        assertTrue(ShareWords.detail(l, 20).contains("Below 20%"))
        assertTrue(ShareWords.detail(l, 20).contains("Plug in"))
    }
}
