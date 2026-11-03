# Round 1 triage: updates + backup

Reports: persona A (GL traveller) 3 high / 20 medium / 7 low, B (homelab)
3 / 18 / 7, C (family member) 3 / 14 / 11, D (Speedify user) 2 / 17 / 9.
Deduplicated below; each item names who raised it.

## High

| # | Finding | Who | Decision |
|---|---|---|---|
| H1 | "Restored" says "Devices reconnect by themselves", contradicting the preview's steps (new address, new links) | A, B, C | **Fixed**: the done dialog repeats the preview's warnings as next steps; only with none does it say devices reconnect by themselves (UI) |
| H2 | A server restore never says to turn off the old server (two machines, same keys) | B (A, D medium) | **Fixed**: restore warning "Turn tailbond off on <old> first (or wipe it)" (agent `restoreWarnings`) |
| H3 | "within a few hours" (kill switch: hours without internet); the Tailscale-name claim isn't guaranteed | A, B | **Fixed**: clients retry their server's pairing every 2 min while they can't connect, so the text says "within a few minutes", and only "as long as this machine has the old one's Tailscale name"; link-only devices are named as needing their link again |
| H4 | The Fedora home server updates only through `sudo dnf …` in a terminal | A, C (B, D medium) | **Kept by design** (the requested design: RPM installs update with dnf/COPR, like any package). Mitigated: the dashboard shows whether dnf-automatic is on (`dnf_automatic`) with the command to turn it on, and how to go back (`dnf downgrade`). Medium. |
| H5 | The "new machine before restoring" render looks like an established server | D (A, B, C medium) | **Fixed**: fixture bug (a fresh install: no devices, no backups) |
| H6 | Nothing says which devices need a new link; the phone has no "server moved" note | C | **Partly**: the warning says which kind of device needs a link (no Tailscale, paired by link/QR) and where; the server can't know which devices lack Tailscale. Medium. |

## Medium (fixed)

- Green "Up to date" while the last check failed → amber "Couldn't check" (A, B, C, D).
- After a rollback the same version is the primary "Update now" with no word on Automatic → calm note, "Automatic won't install it again", secondary "Try again" (A, B, C, D). The agent already skips a rolled-back version automatically.
- Update available / reinstalled-after-firmware only at the bottom of the page → also at the top (GL card, web header; phone home screen) (A, B, C, D).
- Offline device (beryl) missing from "out of date"; no per-device versions → every device with its version, "not known yet" when offline (A, B, D).
- Automatic doesn't say when it installs; could drop a call → installs only 03:00–05:00 local time, and says so (A, B, D).
- No word on where updates come from / how they're checked → the confirm says "from the GitHub release, checked against its SHA-256 list" (B).
- dnf: repository lag, no rollback word → "reaches dnf shortly after the release", `dnf downgrade` (B).
- Port 443 → 8443 after a restore without a firewall note → restore warning (B, D).
- "What's inside" doesn't say what's left out → "Not included: Tailscale login, the program, firewall settings" (B).
- The routers' and phones' pushed backups don't move with the server → a server backup now carries them (A, B, D).
- The scheduled-backup passphrase is kept on the server without saying so → said, with the folder (B).
- Backup file names and "made" dates disagree; homebox's list lacks the phone's pushed backups → fixtures (A, B, C, D).
- Phone: Android's "allow your browser to install apps" prompt and the tunnel after an update → reassurance; the tunnel reconnects after the app updates (`MY_PACKAGE_REPLACED`) (A, C).
- Phone debug build offers Download prominently → secondary, "back up first" (C).
- GL firmware note doesn't say where the package came from or what happens without Keep Settings → said (B).
- Sticky header drawn over the middle of phone screenshots → render fix (A, C).

## Medium / low (not changed, why)

- Updating a router from the home server's dashboard (D): each device updates itself (its panel or Automatic); a server-pushed install would bring back the mirror machinery the design dropped.
- A configurable install window (D): one fixed night window keeps it simple.
- Phone Automatic without Obtainium (D): installing silently needs PackageInstaller machinery the design dropped; Obtainium is the standard route.
- Cipher/KDF names in "What's inside" (B): in docs/backup-restore.md.
- "Add this phone" shown on the homebox phone page (C): another feature's screen, unchanged here.
- Which passphrase opens which file (A): the list says "each is locked with the passphrase it was made with"; device backups are labeled by device.
- Low findings: wording polish, folded into the fixes above where touched.
