# "Get tailbond" page: simulated user testing

Personas, tasks and method: [task-script.md](task-script.md), screens
[screens.md](screens.md) (rendered by `scripts/ux/web-screens.sh`). Fresh
reviewer agents each round, given only the PNGs, the task script and the
screen list; every point triaged **fix / partial / reject (reason)**. Cap:
2 rounds.

## Round 1 ([F2](round-1/persona-f2-home-admin.md), [A](round-1/persona-a-tailscale-daily.md))

No high findings; both personas would finish both tasks.

| # | Finding (persona, severity) | Triage |
|---|---|---|
| 1 | Mac: curl over http with sudo, no "is this safe?" (A, med) | **fix**: plain-language safety note under the command (Tailscale encrypts it; the installer tells macOS it's yours), SHA in it, `shasum -a 256` to copy under Other ways |
| 2 | Gatekeeper paragraph reads like "bypassing Apple's check" (F2, med) | **fix**: plain words in the safety note; the technical line (quarantine flag, `xattr`) moved under Other ways |
| 3 | No "it worked" signal after the install line (A, F2, med) | **fix**: "When Terminal says *tailbond … is installed*, go to step 2" (what install.sh prints) |
| 4 | Password: nothing shows while typing (F2, med) | **fix**: said next to the command |
| 5 | "Apple silicon" unclear; Intel Macs? (F2, med) | **fix**: how to check (About This Mac → Chip), Intel can't run it yet |
| 6 | Homebrew: four commands, from GitHub not homebox, `sudo brew services` odd (A, med) | **partial**: says where it downloads from and why `sudo brew services`; moved under "Other ways". One line needs a published tap (not in scope) |
| 7 | After Ask: does the Mac connect, flip the switch? (A, F2, med) | **fix**: page step 2 "connects by itself; leave the switch alone"; the dashboard's waiting text names where to approve and that it connects by itself |
| 8 | "Is homebox yours?" unclear when someone else set it up (F2, med) | **fix**: "If you look after homebox yourself (nobody else set it up) … otherwise tell whoever does", on Mac and Android |
| 9 | On the Mac, the phone steps under the QR read as Mac steps; Download would put the APK on the Mac (F2, med; A, low) | **fix**: on a computer the QR card says the rest happens on the phone; the phone steps are folded under "The steps on the phone (to read along)" |
| 10 | "Android tab on a computer" showed "On this phone" and no QR (F2, med) | **reject (render artifact)**: that capture used an Android browser at desktop width; the desktop captures now use a Mac / Linux browser (as a computer would) |
| 11 | Phone: download button ~2.5 screens down (A, med; F2, low) | **partial**: "Skip to the download button ↓" at the top of step 1. The pictures stay before the button on purpose: earlier rounds found people hit Android's prompts without having seen them |
| 12 | Phone: "Tailscale's VPN is off" and exit-node traffic buried in step 3; "exit node" jargon (A, F2, med) | **fix**: a "Good to know before you install" note at the top of the Android section in plain words; step 3 leads with "your Tailscale machines keep working", no "exit node" |
| 13 | Page opens scrolled to the switcher, title not visible (A, F2, low) | **fix**: links to a section (#android from the QR code, #mac) open at the top |
| 14 | Mac: does tailbond take over from Tailscale? (A, low) | **fix**: "Tailscale stays on next to it" |
| 15 | Linux https from GitHub vs Mac http from homebox (A, low) | **fix**: one sentence on each source |
| 16 | Updates of a sideloaded APK (A, low) | **fix**: "The app tells you when there's a new version" in the note; Obtainium stays the auto-update option |
| 17 | Phone "Is this safe?" jargon (F2, low) | **reject**: kept for the curious, already behind plain wording and in small print |

## Round 2

See [round-2/](round-2/).
