# Simulated user testing: Tailscale familiarity

Goal: someone who uses the official Tailscale Android app every day finds
tailbond easy and familiar. Method: render every screen and state with
Roborazzi (fake state, 1080x2400, light and dark), then give fresh reviewer
agents only a persona, the [task script](task-script.md), the
[screenshot manifest](screens.md) (rounds 1+) and the PNGs. New agents each
round; raw reports are in `round-N/`. Each finding is triaged
**fix / partial / reject (reason)**. Rejected points are not re-litigated.

Personas:
- **A**: daily Tailscale-Android user, non-networking job;
- **B**: homelab power user who runs Tailscale subnet routers;
- **C**: less-technical family member someone set things up for.

Renders: `out/ux/before/` (round 0), `out/ux/after-N/` (round N) and
`out/ux/web/`. They're not committed (`out/` is ignored); regenerate with
the commands in [android.md](../android.md#screen-renders).

## Findings per round

| Round | Screens | A (H/M/L) | B (H/M/L) | C (H/M/L) | Total H/M/L |
|---|---|---|---|---|---|
| 0 | before | 5 / 14 / 10 | 4 / 17 / 6 | 4 / 10 / 6 | 13 / 41 / 22 |
| 1 | after-1 | 1 / 3 / 9 | 1 / 5 / 11 | 3 / 7 / 3 | 5 / 15 / 23 |
| 2 | after-2 | 0 / 2 / 7 | 0 / 3 / 7 | 1 / 5 / 10 | 1 / 10 / 24 |
| 3 | after-3 | 0 / 1 / 7 | 0 / 4 / 6 | 0 / 4 / 10 | 0 / 9 / 23 |
| 4 | after-4 | 0 / 3 / 6 | 0 / 1 / 7 | 0 / 3 / 7 | 0 / 7 / 20 |

**Outcome: the exit condition (no medium or high from any persona) was not met
within the 4-round cap.** Highs went 13 → 5 → 1 → 0 → 0. The 7 mediums left
in round 4 are mostly limits already triaged as partial or reject (below),
plus one wording regression fixed after round 4 without a re-review.

## Round 0 → redesign

All three personas couldn't do tasks 1 (pair from the phone alone), 3 (find the
NAS: no machine list at all) and 5 (save data: had to guess that "Failover"
means it). Errors were raw (`TLS handshake failed`, `pairing data is not
base64url`), the mode tabs were truncated (`Streamin`, `Redundan`), and nothing
mentioned that tailbond replaces the Tailscale VPN or how to go back.

The redesign (commit `bdb3fc9`, agent `cc24a66`, web `68cfd1f`):

| Finding | Change |
|---|---|
| Can't pair from the phone; needs "server's dashboard → Clients → Add client" | Web **Add this phone** (mobile-first, opens `tailbond://pair` directly); app pairing screen leads with *Open home server page*, then *Scan code from your home server*, then paste |
| No machine list | Agent `GET /api/peers` (tunnel IP, per-client HMAC token); Machines section with online dot, MagicDNS name, 100.x IP, OS, search, copy, machine page |
| Top bar says only "tailbond" | Server name + "via home server · tail1234.ts.net" |
| No "This device" | Tunnel address, tap to copy |
| Save data not discoverable; jargon mode names, truncated | Plain modes: Fastest / Smooth calls and video / Most reliable / Save mobile data, one-line explanations; contract names kept |
| "Is it using both?" needs scrolling and chips like `active` | One sentence ("Using both Wi-Fi and mobile data") plus two cards (In use / Not in use · saving data / Not working right now) |
| No warning that Tailscale gets turned off; no way back | Onboarding screen, handoff dialog before Android's, **Switch back to Tailscale** (stops, remembers, launches `com.tailscale.ipn`) |
| `TLS handshake failed`, `Connection to the server failed` | Errors name the cause and the fix (Try again / Pair again) |
| Always-on advice could lock Tailscale out | Settings: turn Tailscale's always-on off first; leave *Block connections without VPN* off |
| Jargon in settings (`backup_fec`, CIDRs, "carrying paths") | Plain words; experimental switch moved next to Save mobile data |
| Cap "5000" vs dashboard "(no cap)" | Rejected: a render fixture inconsistency (different settings per screen), not an app bug; fixtures now share settings |
| Per-link "loss: tunnel 0.3%" on every link | Removed from the cards (mqvpn has no per-path loss); delay per link stays on the Connections page |

## Round 1 triage

| # | Finding (personas) | Sev | Decision |
|---|---|---|---|
| 1 | No way to install the app from the web page (A, B, C) | H | **fix**: agent serves `GET /tailbond.apk` when `android_apk` exists; web step 1 "Get the tailbond app" with a download button (or "ask whoever set up this server") |
| 2 | Server page's own switch/modes/stats look like the phone's controls (A, B, C) | H/M | **fix**: on a phone the page shows only *Add this phone*; the dashboard is behind "Show the home server's own dashboard (for whoever runs it)" |
| 3 | "Home server's Tailscale name": which machine is it? (C, A) | H/L | **partial**: placeholder shows short and full form; hint says it's the machine that runs tailbond, or ask whoever set it up for the name or a link. The app can't detect it while Tailscale owns the VPN |
| 4 | Certificate error pushes *Pair again* first though the text warns against it (C) | H | **fix**: *Try again* first, *Pair again* second; text explains when re-pairing is expected and when to tell the admin |
| 5 | Phone named `pixel-8` on the web, `pixel` in the app (A, B, C) | M | **reject** (fixture artifact: the render payload and the web guess differed); fixtures aligned |
| 6 | This device: 10.77.0.2 + "seen as homebox" confusing (A, C, B) | M/L | **partial**: "pixel-8 (this phone)"; says it isn't a Tailscale machine and its traffic reaches the tailnet from homebox (100.x). A real 100.x per phone would need a Tailscale node per client: out of scope |
| 7 | Mode names differ web vs app (A, B) | M/L | **fix**: web uses the app's names (contract names as tooltips) |
| 8 | Unreachable error asks a non-admin about UDP port forwarding (C, A) | M/L | **fix**: try again first; port forward framed as "show this to whoever set it up" |
| 9 | Save mobile data buried on the Connections page (C) | M | **fix**: *Save mobile data* switch on the home screen |
| 10 | *Switch back to Tailscale* below the machine list (A, B, C) | M/L | **fix**: right under the status card |
| 11 | After switching back, is Tailscale on? (C) | M | **fix**: card says to open Tailscale and turn on its switch if it isn't connected yet |
| 12 | Paste error shows "Details: pairing data is not base64url" (A, C) | M/L | **fix**: plain sentence only |
| 13 | Public endpoint only visible in errors and settings (B) | M | **fix**: confirm dialog says "It connects over the internet to home.example.net:443" |
| 14 | Subnet routes behind other subnet routers not shown (B) | M | **reject**: needs the server's `--accept-routes` and a contract change for routes; noted for later, not a UI problem |
| 15 | Green dot on a link held back on purpose (A, B) | L | **fix**: grey dot, "Not in use · saving data" |
| 16 | Main/Extra look tappable but aren't marked (A, B) | L | **fix**: dropdown arrow |
| 17 | Connecting shows "No address while disconnected" (B) | L | **fix**: "Getting an address…" |
| 18 | "Faster switch to backup" far from Save data (B) | L | **fix**: moved next to the mode list, shown with Save mobile data |
| 19 | Settings "None" vs apps list with Netflix checked (B) | L | **reject** (fixture artifact); fixtures share settings |
| 20 | Paste error clears the field (B) | L | **reject** (render artifact: the field is local UI state, the render starts empty) |
| 21 | Copy icon copies IPv4 in the list, the detail page copies what you tap (A, B) | L | **reject**: matches Tailscale (list copies the IP; machine page copies each address); the snackbar says what was copied |
| 22 | Could copy own address instead of the NAS (C) | L | **reject**: Tailscale has the same copy on This device; the rows are labelled "(this phone)" now |
| 23 | Server called home.example.net on the web header (A) | L | **reject** (test environment: Tailscale disabled, so the header shows the public address) |
| 24 | "certificate is checked", "holds this phone's key" worry (C) | L | **fix**: "Secure connection to homebox", "Keep this link to yourself" |
| 25 | "Not working" gives no reason (B) | L | **partial**: "Not working right now"; mqvpn exposes no per-path reason |

## Round 2 triage

| # | Finding (personas) | Sev | Decision |
|---|---|---|---|
| 1 | Can't start alone: the server page's address and the server's name are shown nowhere (C, A) | H/M | **partial**: the admin's Devices tab now has *Invite someone's phone* (copyable `http://<magicdns>:8077/#add-phone`); onboarding says up front you need the server's name or link from whoever set it up. The phone can't discover the server by itself while Tailscale owns the VPN, so a first hint from the admin stays necessary |
| 2 | Unknown-apps warning when installing from the browser (C) | M | **partial**: step 1 says Android will warn and why it's fine (the app comes from your own home server). Play Store distribution is out of scope |
| 3 | Nothing says to come back to the browser after installing (C) | M | **fix**: "Then come back to this page for step 2" |
| 4 | Cert error: which button is safe? (C) | M | **fix**: "If you're not sure, don't pair again: tap Try again later and tell whoever set it up." (no button reshuffle: no ping-pong with round 1 #4) |
| 5 | Machines list below the fold (C, A) | M/L | **partial**: Internet traffic row moved below Machines, the mode line shortened; the list now starts on the first screen. Connections stay above Machines because tasks 4–5 need them |
| 6 | Switch off leaves both VPNs off; nothing says Tailscale is still off (C) | M | **fix**: Disconnected line says Tailscale stays off until you switch back to it |
| 7 | Save-data is set in four places (B, A, C) | M/L | **fix**: link roles behind *Advanced: link roles*; Connections page says the home switch picks *Save mobile data*; the monthly limit says it's separate |
| 8 | Per-link speed on home unlabeled and summed (B) | M | **fix**: ↓/↑ per link |
| 9 | This device 10.77.0.2: what's it for? (A, C) | M/L | **partial**: "On your tailnet through homebox (100.101.1.1)"; the round-1 limitation (no per-phone 100.x) stands |
| 10 | Subnet routes / split DNS through homebox not described (B) | M | **reject**: same as round 1 #14 (needs server `--accept-routes` and a contract change) |
| 11 | Settings "opens Tailscale" vs 12b "If Tailscale isn't connected yet…" (A, B) | L | **fix**: "opens the Tailscale app, where you turn Tailscale on" |
| 12 | Data limit: do downloads count? (B) | L | **fix**: "counts everything tailbond sends and receives over mobile data" |
| 13 | Paste-error field looks empty (B, C) | L | **fix** (render): the render now shows the pasted text |
| 14 | "Keep this link to yourself" reads like a warning (C) | L | **fix**: "This link is just for this phone." |
| 15 | *Switch back to Tailscale* is a small text link (C) | L | **fix**: full-width outlined button |
| 16 | Always-on section is jargon (C) | L | **fix**: titled "Keep tailbond on all the time (optional)" |
| 17 | Copy icon copies the IP while the row leads with the name (A, B, C) | L | **reject**: as round 1 #21 |
| 18 | Web header shows home.example.net (A, B) | L | **reject**: as round 1 #23 (test server without Tailscale; manifest now says so) |
| 19 | "Internet traffic: Through your home server" on by default (A) | L | **reject**: full tunnel is the contract default (bonding is the point); one tap to change |
| 20 | Can't pick a tailnet exit node (B) | L | **reject**: the home server is the exit; choosing another exit node would be a server-side feature |
| 21 | Unreachable error has no share button (C) | L | **reject**: low value for its cost; the text is plain enough to read out or screenshot |
| 22 | "Remove from this phone" confirmation not visible (C) | L | **reject**: it does confirm (dialog); not rendered |
| 23 | "Can't reach" error doesn't say whether it retries (B) | L | **reject**: it doesn't retry by itself after a hard failure; *Try again* is the action |
| 24 | Jargon in the confirm dialog (address, Technical details) (C) | L | **reject**: the address line was asked for by B in round 1; details are collapsed |

## Round 3 triage

| # | Finding (personas) | Sev | Decision |
|---|---|---|---|
| 1 | After switching back, the status line says "Tailscale stays off too…" while the card says "Tailscale is in charge" (B, C) | M | **fix**: status line "Stopped. Tailscale is in charge on this phone." |
| 2 | Cert error says don't re-pair but shows a *Pair again* button (C, B) | M | **fix**: only *Try again*; re-pairing after a reinstall goes through Settings. This is the third pass at this card, each one moving toward the safe default, not back and forth |
| 3 | This device: whose is "(100.101.1.1)"? Is 10.77.0.2 useful? (A, B, C) | M/L | **partial**: says tailnet machines can reach the phone at this address through homebox (true once the server's 10.77.0.0/24 route is approved, which the server hooks advertise); the server's 100.x is gone from this row. A per-phone 100.x stays out of scope (round 1 #6) |
| 4 | Switch back doesn't turn Tailscale on by itself (B, A, C) | M/L | **partial**: Android lets only the Tailscale app start its own VPN; tailbond opens it, and the card and Settings say the same thing (turn it on there if it says Disconnected) |
| 5 | Installing past Android's "outside the Play Store" warning (C) | M | **partial**: adds "Unsure? Ask whoever set up the server to install it for you". Store distribution is out of scope |
| 6 | After installing, the app asks for a name instead of sending you back to the page (C) | M | **fix**: the pairing screen's first line says "Came here from your home server's page? Go back to it in your browser and tap Add this phone." |
| 7 | Subnet routes and exit-node choice (B) | M | **reject**: as round 1 #14 and round 2 #20 |
| 8 | "Tailscale stays off too until you switch back" sounds like the Tailscale app won't work (B, A, C) | L | **fix**: "…or switch back to Tailscale below" |
| 9 | Machines note says "Not connected" while Connecting… (A) | L | **fix**: "Connecting… This is the list from …" |
| 10 | "1 app(s)" (A, B) | L | **fix** |
| 11 | "Remove from this phone" not styled as destructive (A, C) | L | **fix**: error colour (it confirms in a dialog) |
| 12 | Handoff title "Turn off Tailscale's VPN?" invites Cancel (C) | L | **fix**: "Switch from Tailscale to tailbond?" |
| 13 | Pairing never says internet traffic leaves from home (B) | L | **fix**: confirm dialog says so and that Settings changes it |
| 14 | "Advanced: link roles" is jargon (B) | L | **fix**: "Advanced: choose which connection is main or backup" |
| 15 | Many data-sounding controls on More options (A, C) | L | **reject**: the home switch is the one control; the page explains it, and the rest is under Advanced or separately labelled |
| 16 | Four address kinds on the machine page (C) | L | **reject**: same as Tailscale's machine page |
| 17 | Red paste error / red unreachable card (C) | L | **reject**: errors need to stand out; the words are plain now |
| 18 | "Not working right now" gives no reason (B) | L | **reject**: as round 1 #25 |
| 19 | "Opening tailbond···" dots (B) | L | **reject**: that's the browser font's ellipsis glyph |

## Round 4 (last round) triage

| # | Finding (personas) | Sev | Decision |
|---|---|---|---|
| 1 | Can't find the server's page or name without asking someone (A, C) | M | **partial, stands** (round 2 #1): the admin has an invite link; the phone can't discover the server while Tailscale owns the VPN |
| 2 | Installing outside the Play Store (C) | M | **partial, stands** (round 3 #5) |
| 3 | This device: 10.77.0.2 instead of a 100.x / MagicDNS name; how ACLs see the phone (A, B) | M | **partial, stands** (round 1 #6): bonded phones are behind the server (masqueraded as it toward the tailnet; reachable at the tunnel IP once the route is approved). Documented in [android.md](../android.md#known-limitations) |
| 4 | Full tunnel by default, unlike Tailscale's "no exit node" (A, B) | M/L | **reject, stands** (round 2 #19): bonding all traffic is the product; the dialog says so and Settings changes it |
| 5 | Cert error "(Settings → Pair again)" invites a family member to re-pair (C, A, B) | M/L | **fix, after round 4 (not re-reviewed)**: "Try again later, and tell whoever set up homebox. Don't pair this phone again unless they ask you to." Renders: `out/ux/after-final/` |
| 6 | After switching back, the Connections area still says "Will use…" / "Ready" (A, B, C) | L | open (low) |
| 7 | Data limit with no Wi-Fi: does the tunnel drop? (B) | L | open (low): at the cap, mobile data stops carrying tunnel paths, so with no Wi-Fi tailbond can't connect; worth one sentence |
| 8 | Copy icon semantics, "Not working" reason, red errors, machines below the fold, several data controls (A, B, C) | L | rejected earlier (round 1 #21, #25; round 3 #15–17); not re-litigated |

## Before / after (key screens)

| Screen | Before | After (round 4 / final) |
|---|---|---|
| Onboarding | `out/ux/before/01-onboarding-pairing.png` | `out/ux/after-4/01-onboarding.png`, `02-pairing.png` |
| Dashboard / home, connected | `out/ux/before/10-dashboard-connected-full.png` | `out/ux/after-4/10-home-connected.png` (+ `-full`, `-dark`) |
| Machines list | (none) | `out/ux/after-4/18-machines-search-copied.png`, `19-machine-detail.png` |
| Connections | `out/ux/before/10-dashboard-connected-full.png` (cards), `20-mode-picker.png` | `out/ux/after-4/20-connections.png`, `21-connections-save-data.png`, `16-save-data-standby.png` |
| VPN handoff dialog | (none) | `out/ux/after-4/04-handoff-other-vpn.png`, `05-handoff-first-permission.png` |
| Errors | `out/ux/before/13-error-unreachable.png`, `14-error-cert.png` | `out/ux/after-final/13-error-unreachable.png`, `14-error-cert.png` |
| Web "Add this phone" | (none) | `out/ux/web/web-01-add-this-phone.png`, `web-02-after-tap.png`, `web-03-devices-scan-code.png` |


## Re-run after pairing over Tailscale (GL work)

The round-4 medium "can't find the server's page or name without asking
someone" is closed for pairing: the app now asks the server by its
Tailscale name and someone approves it there. Re-run results and the
remaining distribution limit (getting the APK without a store) are in
[glinet-summary.md](glinet-summary.md#android-personas-re-run-once-on-the-new-pairing-flow).
