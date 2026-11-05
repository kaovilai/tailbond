package dev.tailbond.app.tv

import android.os.SystemClock
import android.view.InputDevice
import android.view.KeyEvent
import androidx.activity.ComponentActivity
import androidx.compose.ui.graphics.asAndroidBitmap
import androidx.compose.ui.graphics.toArgb
import androidx.compose.ui.test.SemanticsNodeInteraction
import androidx.compose.ui.test.captureToImage
import androidx.compose.ui.test.isRoot
import androidx.compose.ui.test.junit4.AndroidComposeTestRule
import androidx.compose.ui.test.onFirst
import androidx.compose.ui.test.isDialog
import androidx.test.platform.app.InstrumentationRegistry
import dev.tailbond.app.ui.BrandTokens
import org.junit.Assert.assertTrue
import org.robolectric.shadows.ShadowDialog
import org.robolectric.shadow.api.Shadow
import org.robolectric.shadows.ShadowInputDevice
import org.robolectric.shadows.ShadowInputEvent
import org.robolectric.util.ReflectionHelpers
import kotlin.math.abs
import kotlin.math.roundToInt

/** A 1080p Android TV (960×540dp at xhdpi), dark, with a D-pad and no touchscreen. */
const val TV = "w960dp-h540dp-land-television-night-xhdpi-notouch-keyshidden-dpad"

/**
 * The remote: real key events into whichever window has focus (an open
 * dialog, else the activity), from a D-pad, as Android TV delivers them.
 * Back goes the same way, so BackHandler and dialogs' dismiss-on-back run
 * as on a TV.
 */
class Remote(private val compose: AndroidComposeTestRule<*, out ComponentActivity>) {
    init {
        // a TV starts out of touch mode (no touchscreen); clickables only take
        // focus outside it. requestFocusFromTouch leaves touch mode first.
        compose.runOnUiThread { compose.activity.window.decorView.requestFocusFromTouch() }
        // and windows opened later (dialogs, menus) start out of it too, as on a TV
        InstrumentationRegistry.getInstrumentation().setInTouchMode(false)
    }

    /**
     * A real (not virtual) D-pad: text fields hand the D-pad back to focus
     * navigation only for keys from such a device (a keyboard's arrows move
     * the cursor instead).
     */
    private val dpad = ShadowInputDevice.makeInputDeviceNamed("tv remote").also {
        ReflectionHelpers.setField(it, "mId", REMOTE_ID)
        ReflectionHelpers.setField(it, "mSources", InputDevice.SOURCE_DPAD or InputDevice.SOURCE_KEYBOARD)
    }

    private fun dialog() = ShadowDialog.getLatestDialog()?.takeIf { it.isShowing }

    fun press(code: Int, times: Int = 1) = repeat(times) {
        val d = dialog()
        val t = SystemClock.uptimeMillis()
        for (action in listOf(KeyEvent.ACTION_DOWN, KeyEvent.ACTION_UP)) {
            val e = KeyEvent(t, t, action, code, 0, 0, REMOTE_ID, 0, 0, InputDevice.SOURCE_DPAD)
            Shadow.extract<ShadowInputEvent>(e).setDevice(dpad)
            if (d != null) d.dispatchKeyEvent(e) else compose.activity.dispatchKeyEvent(e)
            compose.waitForIdle()
        }
    }

    fun down(times: Int = 1) = press(KeyEvent.KEYCODE_DPAD_DOWN, times)
    fun up(times: Int = 1) = press(KeyEvent.KEYCODE_DPAD_UP, times)
    fun left(times: Int = 1) = press(KeyEvent.KEYCODE_DPAD_LEFT, times)
    fun right(times: Int = 1) = press(KeyEvent.KEYCODE_DPAD_RIGHT, times)
    fun ok() = press(KeyEvent.KEYCODE_DPAD_CENTER)
    fun back() = press(KeyEvent.KEYCODE_BACK)

    /**
     * The focus is visible: the ring (ui/Tv.kt, 3dp wide, 3dp outside the
     * focused item, or just inside it where the page ends) is drawn on at
     * least 3 of the item's 4 sides, looked for across each side's middle
     * in a screenshot of its window.
     */
    fun assertRingAround(node: SemanticsNodeInteraction, ring: Int = BrandTokens.Volt.toArgb()) {
        // the window's content: the dialog's when one is open
        val frame = if (dialog() != null) compose.onNode(isDialog()) else compose.onAllNodes(isRoot()).onFirst()
        val origin = frame.fetchSemanticsNode().boundsInWindow
        val b = node.fetchSemanticsNode().boundsInWindow.translate(-origin.left, -origin.top)
        val img = frame.captureToImage().asAndroidBitmap()
        val dp = compose.activity.resources.displayMetrics.density
        val cx = (b.left + b.right) / 2
        val cy = (b.top + b.bottom) / 2
        // from 8dp outside the edge to 3dp inside it
        val band = (-8..3).map { it * dp }
        val sides = listOf(
            band.map { b.left - it to cy }, band.map { b.right + it to cy },
            band.map { cx to b.top - it }, band.map { cx to b.bottom + it },
        )
        val hit = sides.count { side -> side.any { (x, y) -> near(img, x.roundToInt(), y.roundToInt(), ring) } }
        assertTrue("focus ring on $hit of 4 sides of $b", hit >= 3)
    }

    private fun near(img: android.graphics.Bitmap, x: Int, y: Int, color: Int): Boolean {
        for (dx in -1..1) for (dy in -1..1) {
            val px = x + dx
            val py = y + dy
            if (px !in 0 until img.width || py !in 0 until img.height) continue
            val c = img.getPixel(px, py)
            if (channels(c).zip(channels(color)).all { (a, b) -> abs(a - b) <= 40 }) return true
        }
        return false
    }

    private fun channels(c: Int) = listOf((c shr 16) and 0xff, (c shr 8) and 0xff, c and 0xff)

    private companion object {
        const val REMOTE_ID = 7
    }
}
