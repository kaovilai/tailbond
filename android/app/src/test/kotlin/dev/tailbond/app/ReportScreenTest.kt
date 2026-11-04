package dev.tailbond.app

import android.content.Intent
import androidx.compose.ui.test.assertIsEnabled
import androidx.compose.ui.test.assertIsNotEnabled
import androidx.compose.ui.test.junit4.createComposeRule
import androidx.compose.ui.test.onNodeWithTag
import androidx.compose.ui.test.onNodeWithText
import androidx.compose.ui.test.performClick
import androidx.compose.ui.test.performScrollTo
import androidx.compose.ui.test.performTextInput
import androidx.test.ext.junit.runners.AndroidJUnit4
import dev.tailbond.app.report.CrashCapture
import dev.tailbond.app.report.ReportText
import dev.tailbond.app.ui.ReportScreen
import dev.tailbond.app.ui.TailbondTheme
import org.junit.Assert.assertEquals
import org.junit.Assert.assertFalse
import org.junit.Assert.assertTrue
import org.junit.Rule
import org.junit.Test
import org.junit.runner.RunWith
import org.robolectric.RuntimeEnvironment
import org.robolectric.Shadows.shadowOf
import org.robolectric.annotation.Config

/** Report a problem on Android: the checkbox gates the issue; the URL is redacted and short. */
@RunWith(AndroidJUnit4::class)
@Config(sdk = [35])
class ReportScreenTest {
    @get:Rule val compose = createComposeRule()

    @Test fun reportFlow() {
        val app = RuntimeEnvironment.getApplication()
        CrashCapture.file(app).writeText("tailbond 0.2.0 stopped: uncaught java.lang.IllegalStateException token=hunter2 via 203.0.113.7\n\tat X.y(X.kt:1)")
        compose.setContent { TailbondTheme { ReportScreen() } }
        compose.onNodeWithTag("report-notice").assertExists()
        compose.onNodeWithText(ReportText.NOTICE).assertExists()
        compose.onNodeWithText("Open GitHub issue").performScrollTo().assertIsNotEnabled()
        compose.onNodeWithTag("report-what").performScrollTo().performTextInput("The app closed; my key=hunter2")
        // still off: the box isn't ticked
        compose.onNodeWithText("Open GitHub issue").performScrollTo().assertIsNotEnabled()
        compose.onNodeWithTag("report-confirm").performScrollTo().performClick()
        compose.waitForIdle()
        compose.onNodeWithText("Open GitHub issue").performScrollTo().assertIsEnabled().performClick()
        val sent = shadowOf(app).nextStartedActivity
        assertEquals(Intent.ACTION_VIEW, sent.action)
        val url = sent.dataString!!
        assertTrue(url, url.startsWith("https://github.com/kaovilai/tailbond/issues/new?template=bug_report.yml&"))
        assertTrue(url.length <= ReportText.MAX_URL)
        assertFalse(url, url.contains("hunter2") || url.contains("203.0.113.7"))
        assertTrue(url, url.contains("component=Android%20app") && url.contains("token%3D%5Bredacted%5D"))
        // reported: the crash is no longer offered
        assertTrue(CrashCapture.last(app) == null)
    }
}
