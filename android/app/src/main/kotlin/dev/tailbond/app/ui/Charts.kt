package dev.tailbond.app.ui

import androidx.compose.foundation.Canvas
import androidx.compose.foundation.layout.fillMaxWidth
import androidx.compose.foundation.layout.height
import androidx.compose.runtime.Composable
import androidx.compose.ui.Modifier
import androidx.compose.ui.geometry.Offset
import androidx.compose.ui.graphics.Brush
import androidx.compose.ui.graphics.Color
import androidx.compose.ui.graphics.Path
import androidx.compose.ui.graphics.drawscope.Stroke
import androidx.compose.ui.unit.Dp
import androidx.compose.ui.unit.dp
import dev.tailbond.app.service.LiveState

/**
 * A fixed 60-sample window (one per second), newest at the right edge. All
 * series share one y scale so download and upload compare honestly.
 */
@Composable
fun RateGraph(
    series: List<Pair<List<Long>, Color>>,
    modifier: Modifier = Modifier,
    height: Dp = 96.dp,
    fill: Boolean = true,
    window: Int = LiveState.HISTORY,
) {
    // the area under the first series is the brand beam (teal → volt)
    val beam = LocalBrand.current.let { listOf(it.beamStart, it.beamEnd) }
    Canvas(modifier.fillMaxWidth().height(height)) {
        val peak = series.maxOfOrNull { s -> s.first.maxOrNull() ?: 0L }?.coerceAtLeast(1L) ?: 1L
        val step = size.width / (window - 1).coerceAtLeast(1)
        val grid = Color.Gray.copy(alpha = 0.15f)
        for (i in 1..3) {
            val y = size.height * i / 4f
            drawLine(grid, Offset(0f, y), Offset(size.width, y), strokeWidth = 1f)
        }
        for ((idx, entry) in series.withIndex()) {
            val (values, color) = entry
            if (values.size < 2) continue
            val start = window - values.size
            val line = Path()
            values.forEachIndexed { i, v ->
                val x = (start + i) * step
                val y = size.height - (v.toFloat() / peak) * size.height * 0.92f
                if (i == 0) line.moveTo(x, y) else line.lineTo(x, y)
            }
            if (fill) {
                val area = Path().apply {
                    addPath(line)
                    lineTo((window - 1) * step, size.height)
                    lineTo(start * step, size.height)
                    close()
                }
                if (idx == 0) {
                    drawPath(area, Brush.horizontalGradient(beam.map { it.copy(alpha = 0.30f) }))
                } else {
                    drawPath(area, Brush.verticalGradient(listOf(color.copy(alpha = 0.22f), Color.Transparent)))
                }
            }
            drawPath(line, color, style = Stroke(width = 2.dp.toPx()))
        }
    }
}

@Composable
fun Sparkline(values: List<Long>, color: Color, modifier: Modifier = Modifier) =
    RateGraph(listOf(values to color), modifier, height = 36.dp, fill = true)

/** A rate bar in the brand beam (teal → volt): [fraction] of the track. */
@Composable
fun BeamBar(fraction: Float, modifier: Modifier = Modifier) {
    val beam = LocalBrand.current.let { listOf(it.beamStart, it.beamEnd) }
    val track = androidx.compose.material3.MaterialTheme.colorScheme.outlineVariant
    Canvas(modifier.fillMaxWidth().height(4.dp)) {
        val r = androidx.compose.ui.geometry.CornerRadius(size.height / 2, size.height / 2)
        drawRoundRect(track, cornerRadius = r)
        val w = size.width * fraction.coerceIn(0f, 1f)
        if (w > 0f) drawRoundRect(Brush.horizontalGradient(beam, endX = size.width), size = size.copy(width = w), cornerRadius = r)
    }
}
