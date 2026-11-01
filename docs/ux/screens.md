# Screenshot manifest (rounds 1+)

Phone screenshots, 1080x2400 unless the page is long (then the whole
scrolling page). `-dark` = same screen in dark theme.

| File | What state it shows |
|---|---|
| web-01-add-this-phone | The home server's own web page (http://<server>.<tailnet>.ts.net:8077), opened in the phone's browser while Tailscale is still on |
| web-02-after-tap | Same page after tapping its main button |
| web-03-devices-scan-code | Same web page, Devices tab, after adding a device from another screen |
| 01-onboarding | tailbond app, very first launch |
| 02-pairing | Next screen after the first launch |
| 02-pairing-paste-error | Same, after pasting something wrong |
| 03-deeplink-confirm | The app after a pairing link opened it |
| 04-handoff-other-vpn | After tapping the switch to turn on, while another VPN app is on |
| 05-handoff-first-permission | After tapping the switch the first time, no other VPN on |
| 10-home-connected (+ -full) | Main screen, connected |
| 11-home-connecting | Main screen, connecting |
| 12-home-disconnected | Main screen, off |
| 12b-home-switched-back-to-tailscale | Main screen after the user used the "back to Tailscale" option |
| 13-error-unreachable, 14-error-cert | Two different connection failures |
| 15-one-path-failed | Connected, one connection broken |
| 16-save-data-standby | A data-saving setting chosen, connected |
| 17-save-data-takeover | Same setting, Wi-Fi has dropped |
| 18-machines-search-copied | Main screen after typing in the search box and tapping a copy icon |
| 19-machine-detail | After tapping a machine in the list |
| 20-connections | After tapping the Connections area |
| 21-connections-save-data | Same page with the data-saving setting chosen |
| 30-settings | After tapping the gear icon |
| 31-apps-skip-tailbond | After tapping one of the settings rows |

Some screenshots (dialogs, the copy confirmation) show a moment in time; on a
real phone they appear after the tap described.

The web screenshots come from a test server that isn't on Tailscale, so the
page shows its addresses as `home.example.net:443` / `127.0.0.1` where a real
one shows `homebox.tail1234.ts.net`.
