# Updates + backup: UX summary

Three persona rounds (GL traveller, homelab, family member, Speedify user),
each with fresh reviewers who saw only `task-script.md`, the screen manifests
and the PNGs in `out/ux/updates-backup/{glinet,web,android}/`. Tasks: keep
everything up to date; move the home server to a new machine with a backup;
after a GL firmware update, is tailbond still there?

| Round | A | B | C | D | high total |
|---|---|---|---|---|---|
| 1 | 3 / 20 / 7 | 3 / 18 / 7 | 3 / 14 / 11 | 2 / 17 / 9 | 11 |
| 2 | 2 / 12 / 12 | 3 / 16 / 8 | 2 / 15 / 13 | 1 / 16 / 14 | 8 |
| 3 | 1 / 10 / 10 | 2 / 12 / 8 | 2 / 17 / 14 | 1 / 10 / 12 | 6 |

(high / medium / low per persona.) Task 3 (firmware) passed in every round
from round 2 on; tasks 1 and 2 stayed "partly" for the reasons below.

## What the rounds changed

- **Product fixes** (agent): Automatic installs only at night (03:00–05:00);
  a version that rolled back isn't retried automatically; `dnf_automatic`
  reports install / download / off; a server backup carries the clients'
  pushed backups; a restore keeps this machine's public address and port;
  restore warnings became ordered steps that cut nobody off; a router takes
  a moved server's new link in place of the old one (it was refused as a
  subnet clash before); a router restore from another device warns to turn
  that one off.
- **Screens**: "Couldn't check" instead of a green "Up to date"; calm
  rollback notes with "Try again" secondary; update notices at the top of
  each page and on the phone's home screen; every device with its version,
  compared with the newest release, ticks only after a successful check;
  dnf wording with the commands; "What's inside" says what's left out;
  fixtures that match reality (fresh new machine, file names and dates).

## Accepted (by design, or the server can't know)

- **Fedora updates through dnf** (rounds 1–3, high for the family member and
  the traveller): the requested design is the platform's own mechanism. The
  dashboard says it's done on the computer itself, whether dnf-automatic
  installs or only downloads, and gives the commands.
- **Which devices need a new link after a move**: the server can't tell which
  devices run Tailscale; the steps say what each kind of device does.
- **Phone automatic updates through Obtainium**, not silently: the requested
  design.
- **SHA-256 from the release itself**: integrity over HTTPS; provenance
  attestations exist for anyone who checks.

Reports and triage: `round-1/`, `round-2/`, `round-3/` (each with
`triage.md`).
