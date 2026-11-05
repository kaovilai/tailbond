package dev.tailbond.app.ui

import android.content.res.Configuration
import androidx.compose.animation.core.AnimationSpec
import androidx.compose.animation.core.spring
import androidx.compose.foundation.ExperimentalFoundationApi
import androidx.compose.foundation.gestures.BringIntoViewSpec
import androidx.compose.foundation.background
import androidx.compose.foundation.layout.Box
import androidx.compose.material3.MaterialTheme
import androidx.compose.foundation.layout.BoxScope
import androidx.compose.foundation.layout.PaddingValues
import androidx.compose.foundation.layout.fillMaxSize
import androidx.compose.foundation.layout.padding
import androidx.compose.foundation.layout.widthIn
import androidx.compose.foundation.focusable
import androidx.compose.foundation.onFocusedBoundsChanged
import androidx.compose.foundation.relocation.BringIntoViewRequester
import androidx.compose.foundation.relocation.bringIntoViewRequester
import androidx.compose.runtime.Composable
import androidx.compose.runtime.LaunchedEffect
import androidx.compose.runtime.getValue
import androidx.compose.runtime.mutableStateOf
import androidx.compose.runtime.neverEqualPolicy
import androidx.compose.runtime.remember
import androidx.compose.runtime.setValue
import androidx.compose.runtime.staticCompositionLocalOf
import androidx.compose.ui.Alignment
import androidx.compose.ui.Modifier
import androidx.compose.ui.draw.drawWithContent
import androidx.compose.ui.focus.FocusDirection
import androidx.compose.ui.focus.FocusRequester
import androidx.compose.ui.focus.focusRequester
import androidx.compose.ui.focus.onFocusChanged
import androidx.compose.runtime.withFrameNanos
import androidx.compose.ui.geometry.CornerRadius
import androidx.compose.ui.geometry.Offset
import androidx.compose.ui.geometry.Size
import androidx.compose.ui.graphics.drawscope.Stroke
import androidx.compose.ui.layout.LayoutCoordinates
import androidx.compose.ui.layout.onGloballyPositioned
import androidx.compose.ui.unit.dp
import kotlin.math.min

/**
 * Android TV (docs/android.md, "Android TV"): the same screens, driven by a
 * remote. The phone UI is untouched: everything here is off unless
 * [LocalTv] is true (TailbondTheme sets it from the TV ui mode).
 */
val LocalTv = staticCompositionLocalOf { false }

/** The configuration's ui mode type is television (Android TV, Google TV). */
fun Configuration.isTelevision(): Boolean =
    (uiMode and Configuration.UI_MODE_TYPE_MASK) == Configuration.UI_MODE_TYPE_TELEVISION

/** TV: 5% overscan-safe margins (48dp × 27dp on a 960dp-wide screen) and a readable column width. */
object TvLayout {
    val overscan = PaddingValues(horizontal = 48.dp, vertical = 27.dp)
    val maxWidth = 760.dp
}

/**
 * TV: a ring around whatever has D-pad focus inside this element. Material's
 * own focus highlight is a 10% tint, too faint across a room; this draws one
 * ring for every button, switch, row, chip and field below it, without
 * touching each of them. Dialogs and menus are their own windows: each puts
 * this on its content too. On a phone it does nothing.
 */
@OptIn(ExperimentalFoundationApi::class)
@Composable
fun Modifier.tvFocusRing(): Modifier {
    if (!LocalTv.current) return this
    val color = LocalBrand.current.focus
    // neverEqual: the same coordinates object comes back when the focused
    // item scrolls, and the ring has to follow it
    var focused by remember { mutableStateOf<LayoutCoordinates?>(null, neverEqualPolicy()) }
    val self = remember { arrayOfNulls<LayoutCoordinates>(1) }
    return this
        .onGloballyPositioned { self[0] = it }
        .onFocusedBoundsChanged { focused = it }
        .drawWithContent {
            drawContent()
            val f = focused?.takeIf { it.isAttached } ?: return@drawWithContent
            val me = self[0]?.takeIf { it.isAttached } ?: return@drawWithContent
            val r = me.localBoundingBoxOf(f, clipBounds = true)
            if (r.width <= 0f || r.height <= 0f) return@drawWithContent
            val stroke = 3.dp.toPx()
            val out = 3.dp.toPx()
            val left = (r.left - out).coerceAtLeast(stroke / 2)
            val top = (r.top - out).coerceAtLeast(stroke / 2)
            val right = (r.right + out).coerceAtMost(size.width - stroke / 2)
            val bottom = (r.bottom + out).coerceAtMost(size.height - stroke / 2)
            val corner = min((bottom - top) / 2f, 14.dp.toPx())
            drawRoundRect(
                color = color,
                topLeft = Offset(left, top),
                size = Size(right - left, bottom - top),
                cornerRadius = CornerRadius(corner),
                style = Stroke(stroke),
            )
        }
}

/**
 * TV: how a page scrolls to the item the D-pad moves to. Much of tailbond
 * is explanation between the buttons, and the remote only lands on the
 * buttons, so the focused item keeps a third of the screen above it and a
 * quarter below it: going up shows the text before it (at the page's start
 * the scroll stops at the top, so the opening text stays), going down the
 * text after it. Inside that middle part the page doesn't move, so the
 * first focus usually doesn't scroll at all. Phones keep Compose's default.
 */
@OptIn(ExperimentalFoundationApi::class)
object TvBringIntoViewSpec : BringIntoViewSpec {
    @Suppress("OVERRIDE_DEPRECATION", "DEPRECATION")
    override val scrollAnimationSpec: AnimationSpec<Float> = spring()

    override fun calculateScrollDistance(offset: Float, size: Float, containerSize: Float): Float {
        // the result is asked again as the page moves: each target sits on its zone's edge
        val top = containerSize / 3f
        val bottom = containerSize - containerSize / 4f
        val trailing = offset + size
        return when {
            // too tall for the middle part: its start at the top, unless it's all on screen
            size > bottom - top -> if (offset >= 0f && trailing <= containerSize) 0f else offset
            offset < top -> offset - top
            trailing > bottom -> trailing - bottom
            else -> 0f
        }
    }
}

/**
 * TV: the page area, inside the overscan margins and no wider than
 * [TvLayout.maxWidth], centred. On a phone: [content] as it was.
 */
@Composable
fun TvFrame(content: @Composable BoxScope.() -> Unit) {
    if (!LocalTv.current) {
        Box(content = content)
        return
    }
    Box(
        Modifier.fillMaxSize().background(MaterialTheme.colorScheme.background).padding(TvLayout.overscan),
        contentAlignment = Alignment.TopCenter,
    ) {
        Box(Modifier.widthIn(max = TvLayout.maxWidth).fillMaxSize(), content = content)
    }
}

/**
 * TV: when page [key] appears and nothing in it took focus by itself
 * ([tvInitialFocus]), focus its first item, so the remote always has
 * somewhere to start (a page with no focus can leave the D-pad dead).
 */
@Composable
fun Modifier.tvPageFocus(key: Any?): Modifier {
    if (!LocalTv.current) return this
    val r = remember { FocusRequester() }
    val has = remember { booleanArrayOf(false) }
    LaunchedEffect(key) {
        withFrameNanos { } // the page's own tvInitialFocus goes first
        // Down: the page's top item (Enter would pick its left-most one)
        if (!has[0]) r.requestFocus(FocusDirection.Down)
    }
    return onFocusChanged { has[0] = it.hasFocus }.focusRequester(r)
}

/**
 * TV: put D-pad focus here when this appears (or [key] changes), so the
 * remote starts on the page's main action, scrolled into view. Phones:
 * nothing (touch has no focus, and focusing a field would open the keyboard).
 */
@OptIn(ExperimentalFoundationApi::class)
@Composable
fun Modifier.tvInitialFocus(key: Any? = Unit): Modifier {
    if (!LocalTv.current) return this
    val r = remember { FocusRequester() }
    val view = remember { BringIntoViewRequester() }
    LaunchedEffect(key) {
        r.requestFocus()
        withFrameNanos { } // laid out first: there's nothing to scroll to before that
        view.bringIntoView()
    }
    return bringIntoViewRequester(view).focusRequester(r)
}

/**
 * TV: a stop for the remote on read-only content the D-pad would otherwise
 * skip and scroll past for good (the live rates at the top of Connections).
 */
@Composable
fun Modifier.tvReadable(): Modifier = if (LocalTv.current) focusable() else this
