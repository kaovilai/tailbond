# Phone hub: screens

Renders: Android (`PhoneHubScreens`, `ShareScreens.settingsRow` →
`out/ux/phone-hub/android/`), Mac dashboard (`scripts/ux/web-screens.sh`
web-08 → `out/ux/phone-hub/web/`).

| File | What it shows |
|---|---|
| android/share-01-settings-row.png | Android app, Settings tab (the row that opens the Share screen) |
| android/hub-01-off.png | Android Share screen, "Share bonded connection" off |
| android/hub-02-ready.png | … turned on, no laptop yet |
| android/hub-03-sharing.png | … a Mac using it; a nearby phone's data also bonded |
| android/hub-04-sharing-dark.png | the same, dark |
| android/hub-05-no-tunnel.png | … on, but tailbond itself isn't connected |
| web/web-08-mac-phone-hub-none.png | Mac, tailbond's dashboard (localhost:8077), Connections tab: no phone hub found |
| web/web-08-mac-phone-hub-found.png | … the Pixel found |
| web/web-08-mac-phone-hub-using.png | … using it |
| web/web-08-mac-phone-hub-lost.png | … after the phone went away |
| web/web-08-mac-phone-hub-iphone-found.png | … an iPhone sharing its mobile data found (not bonded) |
| web/web-08-mac-phone-hub-iphone-usb.png | … an iPhone plugged in over USB, bonded by the Mac itself (iPhone (USB) under the connections) |

Text excerpts (no renders: the iPhone app can't be rendered on the build
host): docs/phone-hub.md sections "iPhone (experimental)" and "Mac: Use
phone hub"; docs/macos.md "An iPhone over USB" and "Using a phone's bonded
connection"; docs/ios.md "For a Mac (experimental)" (what tailbond Share shows: a
"Share mobile data with a Mac" switch under "For a Mac (experimental)",
whose footer reads: "Better: plug the iPhone into the Mac with a cable and
turn on Personal Hotspot. tailbond on the Mac then bonds it with the
Wi-Fi, and you don't need this. This switch only lends mobile data, not
bonded, and it stops when you leave this screen or lock the iPhone.").
