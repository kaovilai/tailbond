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
ROUNDS_PLACEHOLDER

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
ROUND2_PLACEHOLDER
