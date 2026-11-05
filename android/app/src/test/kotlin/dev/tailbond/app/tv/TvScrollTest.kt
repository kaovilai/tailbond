package dev.tailbond.app.tv

import dev.tailbond.app.ui.TvBringIntoViewSpec
import org.junit.Assert.assertEquals
import org.junit.Test

/** How a TV page scrolls to the focused item (ui/Tv.kt), on a 1200px page: top third 400, bottom quarter from 900. */
class TvScrollTest {
    private fun scroll(offset: Float, size: Float) = TvBringIntoViewSpec.calculateScrollDistance(offset, size, 1200f)

    @Test fun anItemInTheMiddleDoesNotScroll() {
        // e.g. the first focus, low on the page: the opening text above stays in view
        assertEquals(0f, scroll(400f, 100f))
        assertEquals(0f, scroll(800f, 100f))
    }

    @Test fun nearTheTopKeepsAThirdAbove() {
        // going up: what comes before it shows (at the page's start the scroll stops at the top)
        assertEquals(-300f, scroll(100f, 100f))
        assertEquals(-450f, scroll(-50f, 100f))
    }

    @Test fun nearTheBottomKeepsAQuarterBelow() {
        assertEquals(100f, scroll(900f, 100f))
        assertEquals(1150f, scroll(1950f, 100f))
    }

    @Test fun theTargetIsStableAsThePageMoves() {
        // after the scroll the item sits on the edge: asked again, nothing more
        assertEquals(0f, scroll(900f - 100f - 0f, 100f))
        assertEquals(0f, scroll(400f, 100f))
    }

    @Test fun aTallItemShowsItsStart() {
        assertEquals(0f, scroll(100f, 700f))
        assertEquals(-50f, scroll(-50f, 700f))
        assertEquals(800f, scroll(800f, 700f))
    }
}
