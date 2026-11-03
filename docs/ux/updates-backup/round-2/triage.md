# Round 2 triage: updates + backup

Reports: persona A 2 high / 12 medium / 12 low (round 1: 3 / 20 / 7),
B 3 / 16 / 8 (3 / 18 / 7), C 2 / 15 / 13 (3 / 14 / 11), D 1 / 16 / 14
(2 / 17 / 9).

## High

| # | Finding | Who | Decision |
|---|---|---|---|
| H1 | Fedora: updating, automatic updates and going back need `sudo dnf` on the computer | A, C (D medium) | **Kept by design** (RPM installs update with dnf/COPR, as requested). Mitigated: the card says it's done on the computer itself by whoever looks after it, shows whether dnf-automatic installs or only downloads, and gives the commands. Medium. |
| H2 | "Automatic updates: on" can be false (Fedora's dnf-automatic only downloads by default) and the timer updates every package | B | **Fixed**: `dnf_automatic` is install / download / off (timers + `apply_updates`), and the card says it covers every package |
| H3 | Which devices need their link again after a move; a kill-switch router without Tailscale is offline until then | A, B, C (D medium) | **Fixed as far as the server can know**: the warning names the devices to check, says which kind follows by itself (Tailscale on), and says to send the kill-switch router its link before switching over. The server can't tell which devices run Tailscale. Medium. |
| H4 | The new machine needs the old one's Tailscale name, with no how-to | B (A, C, D medium) | **Fixed**: the warning gives the admin-console steps (remove or rename the old machine, rename this one) |
| H5 | A restore puts the old machine's public address and port on the new one | D (B medium) | **Fixed**: a restore keeps this machine's public address and tunnel port (`backup.KeepOnRestore`), so the preview no longer lists them as changes |

## Medium (fixed)

- "Tell me" selected while dnf-automatic is on → for dnf the control is "Check for new versions: On / Off" (A, D).
- Devices on 0.1.0 marked "up to date" while 0.2.0 is out → compared with the newest release known (B, C, D).
- "Move to a new machine and your devices keep working" overpromises → "the restore preview says what each device needs" (B).
- The "turn the old one off" step shown only after restoring → preview "Before you restore", the command to do it, done dialog "If you haven't already" (A, B, C, D).
- Subnet change breaks an approved Tailscale route → warning (B).
- Restore/what's-inside screens show internal values (notify → auto, speed) → the dashboards' words (C).
- Router on Tell me shows a 03:06 update → fixture (D).
- Phone: failed check shown as a red card → calm note (C).
- GL firmware without Keep Settings says "pair again" → "restore the router's backup from homebox, or pair again" (A, B, D).
- Port forward on the home router for a home move → docs/backup-restore.md and the warning (A, C).

## Not changed (why)

- SHA-256 from the same release proves integrity, not authorship (B): the chosen design (HTTPS from GitHub + SHA256SUMS); build-provenance attestations are published for anyone who wants to check (`gh attestation verify`).
- Updating routers from the server's dashboard; a "Move to a new machine" wizard; a phone "Automatic" without Obtainium (D): out of the simple design.
- Time zones for a travelling router's night window (A): local router time; documented.
- How long devices drop during an update (A): "a few seconds" for clients reconnecting; the "about a minute" is the page coming back.
- Several passphrases; cipher/KDF details; phone backup push status (B, C, D): documented in backup-restore.md; low.
- "Add this phone" on the homebox phone page (C): another feature.
