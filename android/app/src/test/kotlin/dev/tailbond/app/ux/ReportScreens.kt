package dev.tailbond.app.ux

import androidx.compose.ui.test.onNodeWithTag
import androidx.compose.ui.test.performClick
import androidx.compose.ui.test.performScrollTo
import androidx.compose.ui.test.performTextInput
import androidx.test.ext.junit.runners.AndroidJUnit4
import dev.tailbond.app.report.CrashCapture
import dev.tailbond.app.ui.CrashCard
import dev.tailbond.app.ui.ReportScreen
import org.junit.Test
import org.junit.runner.RunWith
import org.robolectric.RuntimeEnvironment
import org.robolectric.annotation.Config

/**
 * Report a problem (docs/reporting.md):
 *
 *   android/scripts/gradle.sh :app:testDebugUnitTest --tests 'dev.tailbond.app.ux.ReportScreens' \
 *       -Ptailbond.uxOut=$PWD/out/ux/report/android
 */
@RunWith(AndroidJUnit4::class)
@Config(sdk = [35])
class ReportScreens : UxRender() {
    private fun crash() = CrashCapture.file(RuntimeEnvironment.getApplication()).writeText(
        "tailbond 0.2.0 stopped: uncaught java.lang.IllegalStateException: path [public IPv4] gone\n\tat dev.tailbond.app.service.PathGate.add(PathGate.kt:88)",
    )

    @Test fun crashCard() {
        crash()
        shot("report-01-crash-card") { CrashCard(onReport = {}) }
    }

    @Test fun preview() {
        crash()
        shot("report-02-preview", tallDp = 1500) { ReportScreen() }
    }

    @Test fun ready() {
        crash()
        shot("report-03-ready", tallDp = 1500) { ReportScreen() }
        compose.onNodeWithTag("report-what").performScrollTo().performTextInput("The app closed by itself when I turned on Share my mobile data.")
        compose.onNodeWithTag("report-confirm").performScrollTo().performClick()
        compose.waitForIdle()
        com.github.takahirom.roborazzi.captureScreenRoboImage(java.io.File(System.getProperty("tailbond.uxOut"), "report-03-ready.png"))
    }

    @Test fun dark() {
        crash()
        shot("report-04-preview-dark", dark = true, tallDp = 1500) { ReportScreen() }
    }
}
