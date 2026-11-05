# tailbond brand

**Logo: "Braid".** Three links (Wi-Fi, mobile data, ethernet) start as round nodes and braid into one fast arrow: many connections, one lane. The round nodes nod to a mesh without imitating Tailscale's dot-grid mark.

| File (`assets/brand/`) | Use |
|---|---|
| `tailbond-icon.svg` | App icons, README, social preview (full colour on an ink tile) |
| `tailbond-mark.svg` | The mark alone on dark backgrounds; Android adaptive-icon foreground |
| `tailbond-mark-light.svg` | The mark alone on light backgrounds |
| `tailbond-mark-mono.svg` | One colour (`currentColor`): Android notification and quick-settings icons, monochrome adaptive icon, macOS menu bar |
| `favicon.svg` | Browser tabs (thicker strokes, reads at 16 px) |
| `tailbond-wordmark-dark.svg` / `-light.svg` | Icon + "tail**bond**" ("bond" heavy, in the beam gradient) |
| `tokens.json` | Colours, radii and fonts for every surface: **the single source** (below) |
| `tokens.css` | The CSS custom properties, generated from `tokens.json` |
| `social-preview.png` | 1280×640 GitHub social preview (the wordmark on ink), rendered by `scripts/brand/render-icons.sh` |

Keep clear space of one node diameter around the mark. Don't recolour single strands, add a fourth strand, or put the mark on teal (use ink or white).

## Palette: "Tide & Volt"

| Role | Dark UI | Light UI | Notes |
|---|---|---|---|
| Background / card | `#071A1D` / `#0B2A2E` | `#F4FBFA` / `#FFFFFF` | Ink, never pure black |
| Primary (buttons, toggles) | Teal 400 `#2DD4BF` | Teal 700 `#0F766E` | |
| Speed accent | Volt `#C6F432` | Volt dark `#65A30D` | Throughput, the beam gradient, "fast" moments only |
| Beam gradient | `#2DD4BF → #C6F432` | `#0D9488 → #65A30D` | Rate bars, the logo arrow, "bond" in the wordmark |

**Each connection kind has one colour, on every surface** (Android cards, GL panel, Fedora/VPS/Mac dashboard, phone page, iPhone share app):

| Kind | Dark | Light |
|---|---|---|
| Wi-Fi / repeater | `#38BDF8` | `#0284C7` |
| Mobile data | `#A78BFA` | `#7C3AED` |
| Ethernet | `#94A3B8` | `#475569` |
| USB tethering | `#FB923C` | `#EA580C` |
| Shared phone (Pair & Share) | `#F472B6` | `#DB2777` |
| Tailnet machines | `#99F6E4` | `#0D9488` |

Status: connected `#22C55E`, warning `#F59E0B`, error `#EF4444`, info `#38BDF8` on dark UIs. On light UIs, `status-light`: `#15803D`, `#B45309`, `#DC2626`, `#0284C7` (the dark set is under 3:1 on white).

**Contrast (WCAG AA).** Text is `text` or `muted` (or `primary` for links and selected tabs); connection and status colours are for **icons, chart lines and dots**, never body text: in light mode Wi-Fi, USB and tailnet are under 4.5:1 on white. Rates printed next to a chart use `primary` / `text` in light mode. `scripts/brand/contrast.py` checks every pair the UIs use (CI).

Type: Inter where available, otherwise the platform UI font (SF Pro, Segoe UI, Roboto); JetBrains Mono / SF Mono for addresses and commands. Radii: cards 16, controls 10, chips fully round, app-icon tile 15/64.

## One source, every surface

`assets/brand/tokens.json` is the only place colours are defined.
`scripts/brand/gen-tokens.py` writes them out for each platform, and CI
(`--check`) fails when one drifts:

| Generated | Used by |
|---|---|
| `assets/brand/tokens.css`, `agent/web/tokens.css` | the dashboard (Fedora, VPS, Mac, Linux client) and the "Get tailbond" page |
| `android/.../ui/BrandTokens.kt`, `res/values{,-night}/brand_colors.xml` | the Android app: Material 3 schemes (no dynamic colour), kind colours, the beam, splash and launcher background |
| `ios/TailbondShare/BrandTokens.swift`, `Assets.xcassets/AccentColor` | tailbond Share on iPhone |
| the `brand-tokens:begin … end` blocks in the GL panel view and the LuCI view | the GL.iNet admin panel (Element-UI look, brand primary and connection colours) and LuCI |

**Rasters.** The SVGs are the source; generated rasters stay out of git
and are made at build time:

- The macOS icon (the iconset PNGs and `tailbond.icns`):
  `scripts/brand/render-macos-icons.sh`, run by
  `packaging/macos/build-bundle.sh` and the Homebrew formula's `--HEAD`
  build (rsvg-convert; iconutil on a Mac). CI renders it on the macOS
  runner for every bundle.
- Android: the launcher, monochrome (themed) icon and notification/tile
  glyph are vector drawables traced from the same SVGs; no PNGs. The
  Android TV banner (`res/drawable/tv_banner.xml`, the dark wordmark on ink
  with the beam bar, 320x180) is one too: `scripts/brand/render-tv-banner.sh`
  makes it, with the wordmark's text as Inter's own glyph outlines
  (fontTools, in a container), since a vector drawable can't draw text.

Two rasters are kept in git, because the platform needs the file itself
and can't make it from an SVG:

| File | Why it's in git | Size |
|---|---|---|
| `ios/TailbondShare/Assets.xcassets/AppIcon.appiconset/AppIcon-1024.png` | Xcode's asset catalog takes a PNG app icon, and tailbond Share is built on the user's own Mac ([ios.md](ios.md)), which shouldn't need rsvg-convert | 29 KB |
| `assets/brand/social-preview.png` | GitHub's **Social preview** setting is a file upload | 35 KB |

`scripts/brand/render-icons.sh` re-renders both after an SVG change
(rsvg-convert with Inter, in a container). CI fails on any build output or
file over 1 MB in git (`scripts/ci/check-tracked-files.sh`).

**GitHub social preview:** upload `assets/brand/social-preview.png` in the
repository's **Settings → General → Social preview** (GitHub has no API for
it).
