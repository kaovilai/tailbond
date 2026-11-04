import SwiftUI

/// The brand colours for the current light/dark appearance (docs/brand.md),
/// from BrandTokens (generated from assets/brand/tokens.json).
struct BrandPalette {
    let bg: Color
    let card: Color
    let text: Color
    let muted: Color
    let primary: Color
    let onPrimary: Color
    let beam: [Color]
    let shared: Color
    let warn: Color
    let error: Color

    static func of(_ scheme: ColorScheme) -> BrandPalette {
        if scheme == .dark {
            typealias T = BrandTokens.Dark
            return BrandPalette(bg: T.bg, card: T.card, text: T.text, muted: T.muted, primary: T.primary, onPrimary: T.onPrimary,
                                beam: T.beam, shared: T.sharedCellular, warn: T.statusWarn, error: T.statusError)
        }
        typealias T = BrandTokens.Light
        return BrandPalette(bg: T.bg, card: T.card, text: T.text, muted: T.muted, primary: T.primary, onPrimary: T.onPrimary,
                            beam: T.beam, shared: T.sharedCellular, warn: T.statusWarn, error: T.statusError)
    }
}

/// The main action: the brand beam (teal → volt) with ink text, or the
/// error colour while it stops something.
struct BeamButtonStyle: ButtonStyle {
    var stop = false
    @Environment(\.colorScheme) private var scheme

    func makeBody(configuration: Configuration) -> some View {
        let p = BrandPalette.of(scheme)
        return configuration.label
            .foregroundStyle(stop ? Color.white : BrandTokens.ink950)
            .background {
                RoundedRectangle(cornerRadius: 14, style: .continuous)
                    .fill(stop ? AnyShapeStyle(p.error) : AnyShapeStyle(LinearGradient(colors: BrandTokens.Dark.beam, startPoint: .leading, endPoint: .trailing)))
            }
            .opacity(configuration.isPressed ? 0.85 : 1)
    }
}
