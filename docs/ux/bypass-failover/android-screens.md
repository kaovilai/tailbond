# Screenshot manifest: Android, backup servers and websites that skip tailbond

Phone screenshots (1080x2400 unless the page is long, then the whole
scrolling page). `-dark` = same screen in dark theme. The data is a test
fixture: the phone has two servers, **homebox** (at home, the main one) and
**vps** (a rented server, the backup). One story runs through the screens:
yesterday a short switch at 09:15–09:40; today homebox stopped answering at
14:02, tailbond went back to it at 14:20, and at 15:10 vps was picked by hand.

Renders: `out/ux/bypass-failover/android/` (not committed), from
`android/app/src/test/kotlin/dev/tailbond/app/ux/BypassFailoverScreens.kt`:

```sh
android/scripts/gradle.sh :app:testDebugUnitTest --tests 'dev.tailbond.app.ux.BypassFailoverScreens' \
    -Ptailbond.uxOut=$PWD/out/ux/bypass-failover/android
```

| File | What it shows |
|---|---|
| 40-home-on-homebox-vps-on-standby (+ -dark) | Main screen, connected through homebox; vps waits as a backup |
| 41-home-using-vps-homebox-unreachable (+ 41b first screen only) | homebox stopped answering at 14:02; tailbond switched to vps by itself |
| 41c-home-using-vps-you-picked-it | After tapping "Use this server now" on vps in Settings (at 15:10) |
| 41d-notification-using-vps | The phone's notification at three moments: on vps, counting down, back on homebox (drawn like Android's notification shade, with the app's own text) |
| 42-home-back-to-homebox-in-1-20 | homebox answers again; tailbond goes back to it after a countdown (or stays on vps if asked) |
| 43-home-back-on-homebox | Shortly after it went back (at 14:20) |
| 43b-home-switching-to-vps | The few seconds while it switches |
| 49-home-error-no-server-reachable | Neither server works (phone offline, or both down) |
| 44-settings-servers (+ -dark) | After tapping the gear icon, while on vps |
| 44c-settings-servers-homebox-answering-again | Same page during the countdown back to homebox |
| 44d-settings-servers-paused-you-picked-vps | Settings after picking vps by hand, with the recent switches |
| 44b-settings-one-server-add-backup | Same page for someone with one server |
| 45-add-another-server | After tapping "Add a backup server" / "Add another server" |
| 45b-add-another-server-confirm | After vps approved the request |
| 45c-add-another-server-same-addresses | Adding a server that uses the same tunnel addresses as homebox |
| 46-websites-skip-tailbond (+ -dark) | Settings → Websites that skip tailbond, with a few websites |
| 46c-websites-none-yet | Same page, empty |
| 46b-websites-needs-android-13 | Same page on a phone with Android 12 |
| 48-settings-skip-rows-android-12 | Settings on Android 12 (one server) |
| 47-add-website-dialog | After tapping "Add a website" |
| 47b/47c/47d-add-website-preset-* | After tapping the Banking / Streaming / Wi-Fi calling choice |

The add-website screens are drawn over the page rather than in a separate
dialog window (the renderer can't capture a text field in a dialog window);
on a phone it is a normal dialog. The menu behind each server's ⋮ (Pair
again, Remove from this phone) isn't shown open.

Changes after round 1 ([triage](round-1/triage.md), Android items H4, M3,
M4, M6–M14, M17, M22, M23): the server in use is named wherever internet
traffic is described; switch and return times ("since 14:02"); "Stay on vps"
during the countdown and "Go back to homebox" after a manual pick; "Use this
server now" hidden for an unreachable server; the server row says Tailscale
may still show it online; backup-specific wording on the add-server page;
one line on how switching decides; website rows and the dialog say which
names they cover; the refresh blip is described; Android 12 warns that the
whole app skips tailbond; the all-servers error drops the technical detail and
says the phone uses its normal internet.

Changes after round 2 ([triage](round-2/triage.md), Android items H1, H3,
M3–M6, M11–M14, M20, M21, M24, M25): website rules also cover the usual
sub-names (www, login, secure, online, auth, account(s), my, m, app, api) and
say so, with the router difference stated; "Block connections without VPN" is
recommended with backup servers (Settings and the switching note open
Android's VPN settings); "Recent switches" (last 20) in Settings; "Paused: you
picked vps" with "Go back to automatic"; "Hand the VPN back to Tailscale"
moved to the bottom of the main screen; "Stay on vps" is a quiet link; the
clash dialog explains who fixes it and copies a message to send them; the
add-server page suggests tailnet machines and says who approves where; the
route approval is optional; the notification is rendered; one consistent
timeline.
