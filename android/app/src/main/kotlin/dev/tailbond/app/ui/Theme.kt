package dev.tailbond.app.ui

import androidx.compose.foundation.ExperimentalFoundationApi
import androidx.compose.foundation.gestures.LocalBringIntoViewSpec
import androidx.compose.ui.platform.LocalConfiguration

import androidx.compose.foundation.isSystemInDarkTheme
import androidx.compose.material3.MaterialTheme
import androidx.compose.material3.darkColorScheme
import androidx.compose.material3.lightColorScheme
import androidx.compose.runtime.Composable
import androidx.compose.runtime.CompositionLocalProvider
import androidx.compose.runtime.Immutable
import androidx.compose.runtime.ReadOnlyComposable
import androidx.compose.runtime.staticCompositionLocalOf
import androidx.compose.ui.graphics.Color
import dev.tailbond.app.model.PathKind

/**
 * The brand ("Tide & Volt", docs/brand.md): Material 3 colour schemes and
 * the connection-kind, beam and status colours, all from BrandTokens
 * (generated from assets/brand/tokens.json by scripts/brand/gen-tokens.py).
 * The brand colours are used on every Android version: no dynamic colour,
 * so the app looks like tailbond everywhere.
 */
@Immutable
data class BrandColors(
    val beamStart: Color,
    val beamEnd: Color,
    val wifi: Color,
    val cellular: Color,
    val ethernet: Color,
    val usb: Color,
    val shared: Color,
    val tailnet: Color,
    val ok: Color,
    val warn: Color,
    val error: Color,
    /** Android TV's D-pad focus ring (ui/Tv.kt): volt on dark, deep teal on light. */
    val focus: Color,
)

private val DarkBrand = with(BrandTokens.Dark) {
    BrandColors(BeamStart, BeamEnd, Wifi, Cellular, Ethernet, Usb, SharedCellular, Tailnet, StatusOk, StatusWarn, StatusError, focus = BrandTokens.Volt)
}
private val LightBrand = with(BrandTokens.Light) {
    BrandColors(BeamStart, BeamEnd, Wifi, Cellular, Ethernet, Usb, SharedCellular, Tailnet, StatusOk, StatusWarn, StatusError, focus = BrandTokens.Teal700)
}

val LocalBrand = staticCompositionLocalOf { LightBrand }

private val Light = with(BrandTokens.Light) {
    lightColorScheme(
        primary = Primary, onPrimary = OnPrimary,
        primaryContainer = BrandTokens.Mint200, onPrimaryContainer = Text,
        secondary = BrandTokens.Teal600, onSecondary = OnPrimary,
        tertiary = BrandTokens.VoltDark, onTertiary = OnPrimary,
        background = Bg, onBackground = Text,
        surface = Bg, onSurface = Text,
        surfaceVariant = Bg, onSurfaceVariant = Muted,
        surfaceContainerLowest = Card, surfaceContainerLow = Card, surfaceContainer = Card,
        surfaceContainerHigh = Card, surfaceContainerHighest = Card,
        outline = Muted, outlineVariant = Border,
        error = StatusError,
    )
}
private val Dark = with(BrandTokens.Dark) {
    darkColorScheme(
        primary = Primary, onPrimary = OnPrimary,
        primaryContainer = BrandTokens.Teal700, onPrimaryContainer = Text,
        secondary = BrandTokens.Mint200, onSecondary = OnPrimary,
        tertiary = BrandTokens.Volt, onTertiary = BrandTokens.Ink950,
        background = Bg, onBackground = Text,
        surface = Bg, onSurface = Text,
        surfaceVariant = Card, onSurfaceVariant = Muted,
        surfaceContainerLowest = Bg, surfaceContainerLow = Card, surfaceContainer = Card,
        surfaceContainerHigh = Card, surfaceContainerHighest = Card,
        outline = Muted, outlineVariant = Border,
        error = StatusError,
    )
}

/** Download / upload lines: the beam's two ends (teal, volt). */
val DownColor: Color
    @Composable @ReadOnlyComposable get() = LocalBrand.current.beamStart
val UpColor: Color
    @Composable @ReadOnlyComposable get() = LocalBrand.current.beamEnd

/** One colour per connection kind, on every surface (docs/brand.md). */
@Composable
@ReadOnlyComposable
fun kindColor(k: PathKind): Color = when (k) {
    PathKind.WIFI -> LocalBrand.current.wifi
    PathKind.CELLULAR -> LocalBrand.current.cellular
    PathKind.ETHERNET -> LocalBrand.current.ethernet
}

/** On a TV (ui mode television) also the remote's focus ring and scrolling (ui/Tv.kt). */
@OptIn(ExperimentalFoundationApi::class)
@Composable
fun TailbondTheme(content: @Composable () -> Unit) {
    val dark = isSystemInDarkTheme()
    val tv = LocalConfiguration.current.isTelevision()
    CompositionLocalProvider(
        LocalBrand provides if (dark) DarkBrand else LightBrand,
        LocalTv provides tv,
        LocalBringIntoViewSpec provides if (tv) TvBringIntoViewSpec else LocalBringIntoViewSpec.current,
    ) {
        MaterialTheme(colorScheme = if (dark) Dark else Light, content = content)
    }
}
