# Phone hub: simulated user testing

Personas, tasks and method: [task-script.md](task-script.md), screens
[screens.md](screens.md). The renders come from `PhoneHubScreens` /
`ShareScreens` (Android) and `scripts/ux/web-screens.sh` web-08 (Mac
dashboard). Each round uses fresh reviewer agents, given only the PNGs,
the task script and the named doc sections. Every point is triaged **fix**,
**partial** or **reject (reason)**. Cap: 2 rounds.

## Round 1 ([A](round-1/persona-a-traveller.md), [F2](round-1/persona-f2-home-admin.md))

Both personas would finish task 1 after wrong turns. Both would likely end
task 2 on the wrong (unbonded) option.

| # | Finding (persona, severity) | Triage |
|---|---|---|
| 1 | Mac says "Disconnected", "No connections yet" while the hub works (F2 high, A med) | **fix**: the state chip says *via pixel-8 (proxy)* and the power label *Own tunnel off*. The empty state says the Mac's own tunnel is off and that this is fine |
| 2 | The Mac card sends iPhone owners to the unbonded proxy (A, F2 high) | **fix**: the card's empty state says an iPhone needs no hub (USB + Personal Hotspot, bonded by the Mac). iPhone hubs advertise `b=0`, and the Mac lists them as *mobile data only, not bonded*, with a warning when in use |
| 3 | Hard to find on Android: the Settings row and screen are named after mobile data only, and the Mac's hint skips steps (A high, F2 med) | **fix**: the Settings row and screen are now called **Sharing**, with a status for both features. *Share bonded connection* is first on the screen. The Mac's hint gives the full path |
| 4 | "Who's using it" / "0 B this month" / limits shown next to an active bonded share (A, F2 med) | **fix**: the mobile-data sections are labelled *your mobile data*, and the hub card says the limits don't apply to it |
| 5 | No steps for the hotspot itself; will the hotel Wi-Fi stay up? (F2 med) | **fix**: the ready state gives numbered steps (Hotspot & tethering or USB tethering; most phones stay on the hotel Wi-Fi) |
| 6 | Jargon: proxy setting, TCP, proxy auto-config; checkbox unexplained (A, F2 med) | **fix**: plain words ("Browsers and most apps…", "Video calls, games and other non-web traffic…"). The checkbox says what on and off do |
| 7 | Where non-web traffic goes; undo steps (F2, A med) | **fix**: the in-use text says it goes out directly over the phone's hotspot, not bonded, and how to undo on both sides |
| 8 | Security only in the docs; which network it was found on (F2 med) | **fix**: both cards say only devices paired with homebox can use it; the Mac shows the interface the hub is reached on. **Partial**: the phone lists the client as "Device at <address>"; the proxy has no paired-device identity to show, and *Stop sharing* ends all clients |
| 9 | No way to remove one client (F2 med) | **reject**: the password is per home server, not per device. Stopping sharing is the control; per-device keys would mean a new protocol version |
| 10 | iOS footer jargon; the iPhone stops when the screen locks (A, F2 med) | **fix**: renamed *For a Mac → Share mobile data with a Mac*. The footer leads with the cable, says it isn't bonded and that it stops off-screen |
| 11 | No render of the iPhone (USB) state or the iOS screen (F2 med) | **partial**: a Mac render with *iPhone (USB)* bonded and one with an iPhone hub found were added. iOS can't be rendered on this Linux host (no Xcode) |
| 12 | "Backup only" unclear; no undo for USB (F2 low) | **fix**: docs/macos.md names the card's dropdown, the undo, and the hotspot password note |
| 13 | No "Turn tailbond on" button when the tunnel is off (A, F2 low) | **fix** |
| 14 | Header overlapping in the renders (A, F2 low) | **fix (render)**: the shots no longer scroll before the full-page capture |
| 15 | Device names differ between the Android and Mac renders (A, F2 low) | **fix (render)**: one fixture name |
| 16 | Fails open when the phone goes away (F2 low) | **reject**: keeping the Mac online is the safer default for most users. The banner says what happened |
| 17 | "Use nearby phones' data" sits between two share-out features (F2 low) | **fix**: moved to the bottom under *Use other phones' data* |
| 18 | Dark-mode card styling of the off mobile-data card (F2 low) | **reject**: it's the M3 surface-variant colour in dark mode; it is not the "on" (primary container) colour |

## Round 2 ([A](round-2/persona-a-traveller.md), [F2](round-2/persona-f2-home-admin.md))

Round-1 fixes held: neither persona was misled about "Disconnected" while
using the hub or pointed at the iPhone proxy over USB again. The one
remaining high was shared by both.

| # | Finding (persona, severity) | Triage |
|---|---|---|
| 1 | Undo leaves the Mac on the phone's hotspot, using unbonded mobile data (A, F2 high) | **fix**: the in-use text lists the whole undo (rejoin the usual Wi-Fi, turn off the hotspot and sharing). After **Stop using it**, the card keeps a note saying so (`HubStoppedNote`) |
| 2 | The render shows "en7 · USB Tether", not the promised "iPhone (USB)" (A high, F2 med) | **fix (real bug)**: dashboard path cards showed the interface name, not the agent's label. They now show the label ("iPhone (USB)", "Pixel's 5G (shared)") with the interface under it |
| 3 | The iPhone USB steps don't say tailbond must be on (F2 med) | **fix**: the card's hint and docs/macos.md say to turn it on (the switch at the top) |
| 4 | No USB advice when an iPhone hub is found (A med) | **fix**: shown under the list when a found hub isn't bonded |
| 5 | The local-addresses checkbox has no recommendation (A, F2 med) | **fix**: on by default, marked recommended (PAC mode, proven in the Mac e2e) |
| 6 | "Shared by nearby phones" sends iPhone owners to the in-app switch (A med) | **fix**: "a cable is better" there too |
| 7 | Docs say "Backup only", the dropdown says "Backup"; the checkbox wording differs (A, F2 med/low) | **fix**: docs match the UI |
| 8 | Does Android USB tethering work with a Mac? (F2 med) | **fix (docs)**: NCM phones (Pixel 6+) yes, otherwise use the hotspot. Not tested on real devices |
| 9 | No battery cut-off for the bonded share (F2 med) | **reject (for now)**: the user is sharing with their own laptop and the notification's **Stop sharing** is one tap; the hotspot's own drain dominates. Listed in docs as a follow-up |
| 10 | Client shown by IP, no per-device revoke; a family member's iPhone gets a Use button (F2 med) | **reject**: same as round 1, #9. Phones paired with the same home server are trusted alike by design (Pair & Share has the same model). Using a hub is always an explicit tap |
| 11 | The top counters read 0 while the proxy moved data (A, F2 low) | **reject**: they count this Mac's own tunnel. The state chip says *via <phone> (proxy)*, and the card shows the proxy's bytes |

Cap reached (2 rounds). The round-2 fixes are in the code, but these
personas haven't re-checked them.
