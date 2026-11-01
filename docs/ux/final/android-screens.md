# Screenshot manifest: Android journey

Phone screenshots (1080x2400 unless the page is long, then the whole
scrolling page). `-dark` = same screen in dark theme. The data is a test
fixture.

**In the phone's browser** (folder `web/`):

| File | What it shows |
|---|---|
| web-00a-phone-page-first-screen | What opens after typing `homebox` into Chrome on the phone (Tailscale on): first screen |
| web-00-phone-page (+ -dark) | Same page, whole length |
| web-00b-phone-page-no-app-yet | Same page after tapping step 2's button while the app isn't installed yet (Chrome came back to the page) |
| web-00c-phone-page-on-computer | Same address opened on a computer's browser |

Android's own install prompts are not screenshots here: the page draws
what they look like (web-00). On a real phone they appear while installing.

**In the tailbond app** (folder `android/`):

| File | What it shows |
|---|---|
| 02a-pairing-asking-from-phone-page | The app right after tapping the phone page's step 2 button (app installed) |
| 02b-pairing-ask-waiting | A few seconds later |
| 02c-pairing-ask-not-found, 02d-pairing-ask-declined | Two ways asking can go wrong |
| 02e-pairing-ask-approved-confirm | After someone approved it on homebox |
| 01-onboarding (+ -dark) | The app's very first screen when opened from the launcher instead |
| 02-pairing | The next screen after that (also where 02a–02d live) |
| 02-pairing-paste-error | Same, after pasting something wrong |
| 04-handoff-other-vpn (+ -dark) | After tapping the switch to turn on, while Tailscale's VPN is on |
| 05-handoff-first-permission | After tapping the switch the first time, no other VPN on |
| 10-home-connected (+ -full, -dark) | Main screen, connected |
| 11-home-connecting | Main screen, connecting |
| 12-home-disconnected (+ -dark) | Main screen, off |
| 12b-home-switched-back-to-tailscale | Main screen after the user used the "back to Tailscale" option |
| 13-error-unreachable, 14-error-cert | Two connection failures |
| 15-one-path-failed | Connected, one connection broken |
| 16-save-data-standby | A data-saving setting chosen, connected |
| 17-save-data-takeover | Same setting, Wi-Fi has dropped |
| 16c-home-wifi-needs-sign-in | Connected; the phone joined a hotel Wi-Fi that shows a sign-in page |
| 18-machines-search-copied | Main screen after typing in the search box and tapping a copy icon |
| 19-machine-detail (+ -dark) | After tapping a machine in the list |
| 20-connections (+ -dark) | After tapping the Connections area |
| 21-connections-save-data | Same page with the data-saving setting chosen |
| 30-settings (+ -dark) | After tapping the gear icon |
| 31-apps-skip-tailbond | After tapping one of the settings rows |

Some screenshots (dialogs, the copy confirmation) show a moment in time; on a
real phone they appear after the tap described. Android's own screens
(VPN permission dialog, the Wi-Fi sign-in page, the Tailscale app) aren't
shown.
