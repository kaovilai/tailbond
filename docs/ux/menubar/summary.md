# Mac menu bar app and Report a problem: simulated user testing

Personas, tasks and method: [task-script.md](task-script.md); screens:
[screens.md](screens.md). The menu renders come from
`macos/TailbondMenu` SnapshotTests (CI artifact `menubar-screens`); the
report screens from `scripts/ux/report-screens.sh` (artifact
`report-screens-web`). Each round uses fresh reviewer agents that see only
the PNGs, the task script and the named doc excerpts. Every point is triaged
**fix**, **partial** or **reject (reason)**. Cap: 2 rounds.

## Round 1 ([A](round-1/persona-a-traveller.md), [N](round-1/persona-n-netbird.md))

Both would finish tasks 1-3; both would get stuck at the end of task 4.

| # | Finding (persona, severity) | Triage |
|---|---|---|
| 1 | Crash and log text don't wrap: you must confirm you read what you can't see (A, N high) | **fix**: the dashboard and GL panel boxes wrap (`pre-wrap`); the browser check asserts it |
| 2 | Private and work-VPN details (internal names, VPN addresses) stay in a public report (N high) | **partial**: the agent now replaces the work VPN's split-DNS names (`*.corp.example`) with `[work name]`; private addresses stay on purpose (they find bugs) and the notice now says so, in all three ports (shared vectors) |
| 3 | "Open GitHub issue" is off with no reason (A med) | **fix**: a line under it says what's missing (the text, then the box), on the dashboard, GL panel, Android and iPhone |
| 4 | Nothing says a GitHub account is needed, nor what to do without one (A med) | **fix**: a shared note in every app: *Submitting needs a GitHub account; no account? save the report file and send it to whoever set up tailbond* |
| 5 | Pre-filled "VPS / Linux install.sh", "Fedora Linux (WSL)"; the dashboard opened is the home server's (A, N med) | **reject (test setup)**: the report screens were captured from a Linux server-role agent. On a Mac the agent says "Mac" and macOS, and the menu opens the Mac's own dashboard with the form already open (`/#report`); the install test now checks `GET /api/report` on the macOS runner |
| 6 | The on/off switch looks off while connected (A, N med) | **fix**: an On/Off label under it, and the brand tint (renders show an inactive window's grey switch) |
| 7 | Which of Direct/Bonded is chosen is hard to see; neither choice is explained (A, N med) | **fix**: *Now: Direct* under the switch, and one line each: Direct = NetBird and its networks (listed) use the Mac's own internet; Bonded = through tailbond to homebox, then to the company |
| 8 | The MTU warning is jargon with no consequence (A, N med) | **fix**: the agent's text now says Bonded isn't possible on this connection and work traffic goes Direct, in plain words (the round's fixture had an invented warning; it now uses the agent's) |
| 9 | "Right away" next to "waiting for a quiet moment"; no Install now (A, N med) | **fix**: *It installs by itself as soon as no call or big download is running (within 30 minutes)*, and **Install now** when the install can (bundle installs). The fixture's "security" flag with a wait was impossible (security releases don't wait) and is gone |
| 10 | Ethernet "not working" where nothing is plugged in (A med) | **partial**: a failed link now says *can't reach homebox* (what failed); an unplugged adapter isn't a link at all in the agent |
| 11 | "2 links" but three listed (A low) | **fix**: *2 of 3 links in use* |
| 12 | Crash time raw UTC in the menu, local on the dashboard (A, N low) | **fix**: the menu shows the Mac's local date and time |
| 13 | Two "Report a problem…" after a crash (A, N low) | **fix**: the footer's is hidden while the crash box shows |
| 14 | Update policy shown twice; "up to date" with a download icon (A, N low) | **fix**: the line says the version only, with a check mark; the policy is the menu beside it |
| 15 | Placeholder text looks like an answer (A low) | **fix**: placeholders start with "e.g." |
| 16 | Interface names (en0, sh5a1b2c) are noise (A low) | **reject**: they're how a support thread or the dashboard names the link; kept in parentheses |
| 17 | Upload per link, a "learn more" link, testing work-name resolution (N low) | **reject (scope)**: the dashboard has per-link detail and the docs; the menu stays short |

## Round 2 ([A](round-2/persona-a-traveller.md), [N](round-2/persona-n-netbird.md))

A: no high; would finish every task. N: finishes tasks 1-3; task 4 only
through a hand-scrubbed file, because of the work addresses.

| # | Finding (persona, severity) | Triage |
|---|---|---|
| 1 | The NetBird address and the company subnets it pushes stay in a public report (N high) | **fix**: the agent replaces addresses inside the work VPN's networks with `[work address]` and the networks with `[work network]` (from its coexist status; the tailnet's and home addresses stay); the notice says so |
| 2 | "It will be public" is buried in the notice (A med) | **fix**: the notice now starts with *Once you submit this issue on GitHub, it is public* (all three ports, the shared vectors) |
| 3 | The orange warning while Direct is chosen looks broken; its numbers don't add up; Bonded still looks clickable (A, N med) | **fix**: the agent reports it as `bonded_blocked`, not a warning: *Bonded isn't possible on this connection (en0): NetBird's packets need 1360 bytes inside tailbond (1280 + 80), and this connection carries 1180. Work traffic goes Direct, which works.* The menu shows it as information and drops the switch |
| 4 | Direct doesn't say what happens to NetBird's own encrypted traffic (N med) | **fix**: *…and its own connections to its servers and peers, use this Mac's own internet* (the agent's host routes for NetBird's endpoints, checked by the macOS and netns e2e) |
| 5 | The update line doesn't show the installed version; nothing says installing restarts the tunnel (A, N med) | **fix**: *Update available: tailbond 0.2.1 (you have 0.2.0)… tailbond restarts, a few seconds without the tunnel* |
| 6 | "Sam's phone (sh5a1b2c)" is an unexplained device (N med) | **fix**: *Sam's phone, mobile data it shares* |
| 7 | The menu's Report a problem opens the dashboard, not the form (A med) | **reject (screens)**: it opens `/#report`, which opens the form directly; the browser check now asserts that |
| 8 | The switch track is grey (A med, N low) | **reject (render)**: the snapshots draw an inactive offscreen window, where macOS greys switches; the On/Off label (round 1) carries the state either way |
| 9 | The security update isn't marked (N med) | **reject (fixture)**: round 1 removed the impossible "security and still waiting" fixture; a security release says *a security fix* (unit test) |
| 10 | "Right away" vs "Automatic (night)" (A med) | **reject**: the policy names are the same in every app (docs/updates.md); the update line says when it installs |
| 11 | The log box is small; you confirm without seeing what goes in the link (A, N med) | **partial**: the boxes wrap (round 1) and the link carries the newest lines of the same text; "Copy full report" shows all of it. Not changed further |
| 12 | Times written two ways; `/tmp` paths and the server name stay; per-link upload (A, N low) | **reject (low)**: local formats per surface; the server's name and paths identify nothing outside the user's own setup |

Cap reached (2 rounds). Both highs of round 1 and the one high of round 2
are fixed; what's left is low or a rendering artefact.
