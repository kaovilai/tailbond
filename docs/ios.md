# tailbond Share for iPhone

**tailbond Share** lends your iPhone's mobile data to your travel router or a
Linux laptop running tailbond, which bonds it with its other connections
([Pair & Share](pair-and-share.md)). It is not a VPN: the iPhone never runs
the tunnel itself. That is what lets it build with a **free Apple ID**:
personal-team signing can't use Network Extension (VPN), but sharing only
needs ordinary networking.

Source: `ios/` (SwiftUI app + the `TailbondShareKit` Swift package). It
doesn't touch `third_party/mqvpn/ios`.

## What it does

- **Pair:** scan the QR code from your home server (dashboard → Devices →
  Add a device), open the `tailbond://pair?d=…` link on the iPhone, or paste
  it. The pairing is kept in the iPhone's Keychain (this device only). Only
  its server address and share key are used.
- **Share mobile data:** one big button. While on, the app listens on the
  Wi-Fi (UDP 47077, advertised over Bonjour as `_tailbond-share._udp`) and
  sends what the hub gives it to your home server over **cellular only**
  (`NWParameters.requiredInterfaceType = .cellular`).
- **Who's using it:** each hub (e.g. `slate7`) and how much it moved.
- **Limits:** stop below 10/20/30/50 % battery unless charging; a monthly
  limit; only certain hubs (by their device name on your server).

### Keep it on the screen

iOS suspends an app shortly after it leaves the screen, and without a VPN
entitlement there's no way around that. tailbond Share says so on its main
screen:

> Keep tailbond Share on the screen while sharing. iOS pauses sharing as soon
> as you switch apps or lock the phone, and it starts again when you come
> back.

While sharing it keeps the screen on (`isIdleTimerDisabled`). Plug the phone
in for long sessions, or use USB tethering into the travel router instead
(no app needed; it shows as **iPhone (USB)**).

The first time you share, iOS asks to allow **Local Network** access: allow
it, or hubs can't find the phone.

## Build and run on your iPhone (free Apple ID)

Needs a Mac with Xcode 16 or newer and the iPhone's USB cable.

```sh
brew install xcodegen
git clone https://github.com/kaovilai/tailbond
cd tailbond/ios
xcodegen                     # makes TailbondShare.xcodeproj from project.yml
open TailbondShare.xcodeproj
```

In Xcode:

1. **Xcode → Settings → Accounts → +** → Apple ID: sign in with your normal
   Apple ID. It becomes a *Personal Team*.
2. Select the **TailbondShare** target → **Signing & Capabilities** →
   **Team**: your Personal Team. If Xcode says the bundle identifier isn't
   available, change it to something unique, e.g. `dev.tailbond.share.yourname`
   (or set `PRODUCT_BUNDLE_IDENTIFIER` in `project.yml` and run `xcodegen`
   again).
3. Connect the iPhone, unlock it, trust the Mac. On the iPhone turn on
   **Settings → Privacy & Security → Developer Mode** (iOS 16+; it restarts).
4. Pick the iPhone as the run destination and press **Run** (⌘R).
5. First launch only: on the iPhone, **Settings → General → VPN & Device
   Management** → your Apple ID → **Trust**.

**7-day limit:** apps signed by a free Personal Team stop opening after 7
days. Press Run in Xcode again to re-sign (your pairing stays). A free team
can have 3 such apps on a device at once.

**Refresh automatically:** [SideStore](https://sidestore.io) or
[AltStore](https://altstore.io) re-sign sideloaded apps every few days from
the phone (SideStore) or a computer on the same Wi-Fi (AltStore). Export an
`.ipa` (Xcode → Product → Archive → Distribute App → *Debugging*, or build for
"Any iOS Device" and zip `Payload/tailbond Share.app`), then add it in
SideStore/AltStore with the same Apple ID.

There is no App Store or TestFlight build (both need the paid developer
program).

## Tests

CI job `ios` (macOS): the relay core's unit tests (`swift test`: framing and
MACs against the Go reference's `agent/internal/share/testdata/vectors.json`,
handshake, wrong key, allowed hubs, battery, cap, idle hubs, pairing links),
the same tests on an iOS simulator, and an unsigned simulator build of the
app. Locally: `cd ios/TailbondShareKit && swift test`.

Not tested on a real iPhone yet: Bonjour advertising and the Local Network
prompt on a real Wi-Fi, cellular-only sockets next to Wi-Fi, and how quickly
iOS suspends the app after it leaves the screen.
