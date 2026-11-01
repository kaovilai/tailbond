package dev.tailbond.app.ux

import androidx.compose.runtime.Composable
import androidx.compose.ui.test.junit4.createComposeRule
import com.github.takahirom.roborazzi.ExperimentalRoborazziApi
import com.github.takahirom.roborazzi.captureScreenRoboImage
import dev.tailbond.app.ui.TailbondTheme
import org.junit.Assume.assumeTrue
import org.junit.Before
import org.junit.Rule
import org.robolectric.RuntimeEnvironment
import java.io.File

/**
 * Base for screen renders: a 1080x2400 (360x800 dp at xxhdpi) phone, light or
 * dark. Skipped unless Gradle gets -Ptailbond.uxOut=<dir>.
 */
abstract class UxRender {
    @get:Rule val compose = createComposeRule()

    private val out: String? = System.getProperty("tailbond.uxOut")

    @Before fun requireOut() = assumeTrue("set -Ptailbond.uxOut to render", !out.isNullOrBlank())

    /** Renders [content] to `<uxOut>/<name>.png`; [tallDp] > 800 renders a long page instead of one screen. */
    @OptIn(ExperimentalRoborazziApi::class)
    fun shot(name: String, dark: Boolean = false, tallDp: Int = PHONE_H, content: @Composable () -> Unit) {
        RuntimeEnvironment.setQualifiers("w360dp-h${tallDp}dp-normal-long-notround-port-${if (dark) "night" else "notnight"}-xxhdpi-keyshidden-nonav")
        compose.setContent { TailbondTheme { content() } }
        compose.waitForIdle()
        val f = File(out, "$name.png")
        f.parentFile?.mkdirs()
        captureScreenRoboImage(f)
    }

    companion object {
        const val PHONE_H = 800
    }
}
