# Report a problem

Every tailbond app can turn a problem into a GitHub issue that you check
before anything is public:

| Where | How to open it |
|---|---|
| Dashboard (Fedora, VPS, Mac, Linux laptop) | **Report a problem** at the bottom of every page, or in the **Logs** tab; the red banner after a crash |
| GL.iNet admin panel (VPN → tailbond) | **Report a problem** at the bottom; the yellow alert after a crash |
| Android app | **Settings → Help → Report a problem**; the card on the home screen after a crash |
| tailbond Share (iPhone) | the **⚠︎ speech bubble** at the top right; the banner after a crash |

**Nothing is sent automatically.** tailbond has no telemetry, and making a
report makes no network call: the app builds a link and your browser opens
GitHub's issue form with it, filled in. You can still edit it there, and
nothing is public until you press **Submit new issue** on GitHub.

## What you see

1. **The notice** (the same words in every app): nothing is sent; the issue
   will be public; what was taken out; check the rest yourself.
2. **The report, editable:** what happened (required), steps, where (the
   issue form's component), the tailbond version, the device and OS, the
   crash details if there was a crash, and the newest log lines.
3. **A checkbox you must tick:** *I've read this report and it has nothing
   I don't want public.* **Open GitHub issue** stays off until it's ticked
   and *what happened* isn't empty.
4. **Copy full report** (the whole text, to the clipboard) and **Save
   report file** (`tailbond-report-<time>.txt`): for a security report,
   for a long log, or to send it another way.

A link holds about 7 KB (`MaxURL` 7000 characters), so the issue gets the
newest log lines that fit, with a note saying how many earlier lines were
left out; a crash always goes first. The copied and saved report have
everything.

## What is taken out

Every field is redacted when shown and again when the link or file is made
(you might paste something in). Replaced with a word in brackets:

| What | Becomes |
|---|---|
| pairing and share links (`tailbond://pair?…`) | `tailbond://pair?[redacted]` |
| Tailscale auth keys (`tskey-…`) | `[tailscale key]` |
| `Authorization: Basic/Bearer …` | `Basic [redacted]` |
| `key`, `token`, `secret`, `password`, `pass`, `psk`, `auth`, `cookie`, `share_key`… as `name=value`, `name: value`, `"name":"value"` or `--name value` | `name=[redacted]` |
| MAC addresses | `[mac]` |
| hex runs of 32+ characters (keys, hashes) | `[hex]` |
| key-like runs of 40+ characters (upper and lower case and digits) | `[secret]` |
| email addresses | `[email]` |
| your tailnet's name in MagicDNS names | `homebox.[tailnet].ts.net` |
| your user name in `/home/…` and `/Users/…` | `/home/[user]` |
| public IPv4 addresses | `[public IPv4]` |
| public IPv6 addresses (2000::/3) | `[public IPv6]` |

Kept, because they help and identify nobody: private, CGNAT/tailnet,
loopback and link-local addresses (`10.77.0.2`, `192.168.8.1`,
`100.101.1.1`, `fd7a:115c:a1e0::1`), version numbers (`6.18.33.2-microsoft`),
times, file and package names. Not detected: Wi-Fi names and device names;
the notice asks you to check for those.

The rules live in `agent/internal/report/report.go`, ported to Kotlin
(`android/.../report/Report.kt`) and Swift
(`ios/TailbondShareKit/.../Report.swift`). The three must agree byte for
byte: `agent/internal/report/testdata/vectors.json` (redaction cases, the
URL encoding, whole reports with their text and link, the notice's words)
is run by `go test ./internal/report`, the Android unit tests and the
TailbondShareKit tests. Change a rule in Go, regenerate the outputs
(`go test ./internal/report -update`), and port it; CI fails until all
three match.

## Crashes

| App | What's caught | Kept in |
|---|---|---|
| tailbondd (all servers, routers, Macs, Linux) | a Go fatal error or panic (`runtime/debug.SetCrashOutput`); mqvpn ending on SIGSEGV, SIGABRT, SIGBUS, SIGILL, SIGFPE or SIGTRAP, with its last 40 lines | `<state_dir>/crash-last.log` |
| Android app | an uncaught exception's stack; on Android 11+, the system's record of a native crash or an ANR | the app's `files/crash-last.txt` |
| tailbond Share | an Objective-C exception; any other crash from MetricKit on the next launch | Application Support/`crash-last.txt` |

The crash stays until you report it (Android, iPhone) or press **Dismiss**;
the dashboard and the GL panel show when it happened (`crash` in
`GET /api/status`). Logs come from the agent's log buffer (the Logs tab),
the app's own logcat lines (Android) or its own `os_log` lines (iPhone).

## A security problem

Don't open a public issue: every report screen links to GitHub's
[private vulnerability reporting](https://github.com/kaovilai/tailbond/security/advisories/new)
(see [SECURITY.md](../SECURITY.md)). Use **Copy full report** or **Save
report file** and paste it there.

## API (agent)

- `GET /api/report`: `component`, `version`, `device`, `logs` (the newest
  300 lines), `crash`, `crash_at`, redacted; and `notice`, `confirm`,
  `security_note`, `security_url` for the screen.
- `POST /api/report` `{component, version, device, what, steps, logs,
  crash}` → `{url, full, file_name}`, everything redacted again.
- `POST /api/report/crash/dismiss`: forget the kept crash.

## Tests

- `go test ./internal/report ./internal/agent` (vectors, URL length, crash
  capture, mqvpn signals, the API); the Android unit tests (`ReportTest`,
  `ReportScreenTest`: the checkbox gate, the URL the browser gets); the
  TailbondShareKit tests (`ReportTests`).
- `scripts/ux/report-screens.sh`: Chromium against a real tailbondd with a
  crash and secrets in its logs: the dashboard (desktop, phone) and the GL
  panel. It checks the flow and that nothing secret is in the preview, the
  link or the file, and saves screenshots (CI job `report-ux`, artifact
  `report-screens-web`; Android renders: `report-screens-android`).
