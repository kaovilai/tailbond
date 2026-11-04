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
| `tokens.json` / `tokens.css` | Colours, radii and fonts for every surface |

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

Status: connected `#22C55E`, warning `#F59E0B`, error `#EF4444`, info `#38BDF8`.

Type: Inter where available, otherwise the platform UI font (SF Pro, Segoe UI, Roboto); JetBrains Mono / SF Mono for addresses and commands. Radii: cards 16, controls 10, chips fully round, app-icon tile 15/64.
