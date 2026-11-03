# Round 3 triage: updates + backup (final round)

Reports: persona A 1 high / 10 medium / 10 low, B 2 / 12 / 8, C 2 / 17 / 14,
D 1 / 10 / 12. Round 3 is the last round; the fixes below were applied
without another review.

## High

| # | Finding | Who | Decision |
|---|---|---|---|
| H1 | Server move steps in an order that cuts off a kill-switch travel router (turn the old server off first, but its new link only exists after the restore), and a router had nowhere to take a new link | A, B, C, D | **Fixed**: the restore preview lists numbered steps in a safe order (restore while the old machine still runs; link-only devices get their new link; then the old one goes off and its Tailscale name moves). A GL router takes the new link in its panel (Add another server → Paste a link), and the agent now **replaces** the old link with it when the credential matches (a moved server), instead of refusing it as a subnet clash. Done dialog: "Next, in order", the command with a Copy button. |
| H2 | Devices shown "up to date" when the update check failed or is off | B | **Fixed**: the tick only after a successful check |
| H3 | A phone user can't tell whether the phone needs anything after a move | C | **Partly**: the steps say what a phone without Tailscale does (scan the new QR in the app) and that devices with Tailscale follow by themselves. Which devices run Tailscale isn't known to the server. Medium. |

## Medium (fixed)

- Restore button looks like delete (web) or disabled (GL) → normal primary style (C).
- Installing on the router: "Leave the router on; it comes back by itself" (C).
- Router "Update available": "update now or later; everything keeps working meanwhile" (C).
- Phone: harmless "app not installed" note in red, "Undo restore" looking disabled, dense paragraph under Download (C).
- homebox's copies of the phone's backups looked daily → fixtures (B, C, D).

## Not changed (why)

- Fedora updates through dnf on the computer; dnf-automatic covers every package; COPR may lag GitHub by a little (A, B, C, D): the chosen design; the card says all three.
- SHA-256 from the same release (B): integrity over HTTPS by design; provenance attestations are published for anyone who wants to check.
- Router night window in a travelling time zone; router off at night (A): local time; it installs the next night it's on.
- Phone Automatic via Obtainium; no Play Store (C, D): the chosen design.
- Firewall / DNS / port forward don't move with a backup (B): said in the steps and in docs/backup-restore.md.
- Passphrase storage details, cipher/KDF, phone backup status (B, C, D): docs/backup-restore.md; low.
