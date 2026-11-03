# Updates

tailbond is released like most open-source projects: each version is a
[GitHub release](https://github.com/kaovilai/tailbond/releases) with every
file and a `SHA256SUMS` list (plus GitHub build-provenance attestations, so
anyone can check a file with `gh attestation verify FILE -R kaovilai/tailbond`).

Every tailbond install looks at the latest release about once a day and
shows **Update available: vX.Y (What's new)** in its dashboard, the GL panel
or the app. Each platform then updates the way it normally does:

| Device | How updates arrive | Turn them off |
|---|---|---|
| Home server on Fedora (RPM) | `dnf`, from the [COPR repository](fedora-server.md#updates): `sudo dnf upgrade tailbond-server`, or automatically with `dnf-automatic`. The dashboard only tells you. | Settings → Updates → Off (no checks); `dnf` is up to you |
| VPS / any Linux (bundle, `install.sh`) | **Update now** in the dashboard, or `sudo /usr/local/lib/tailbond/install.sh --update`. **Automatic** installs them by itself. | Settings → Updates → Off, or `install.sh --update-policy off` |
| VPS with Ansible | bump `tailbond_version` and run the playbook ([vps.md](vps.md#updates)) | `tailbond_update_policy: off` |
| GL.iNet travel router | **Update now** in the tailbond panel, or **Automatic** | tailbond panel → Updates → Off |
| Android phone | [Obtainium](android.md#updates) installs new releases by itself (recommended); or the app's **Update available → Download update** | Settings → Updates → Off; remove the app from Obtainium |

Policies (dashboard, GL panel, `"update": {"policy": …}` in `tailbond.json`):
**Tell me** (`notify`, the default: shows new releases, installs when you
press Update now), **Automatic** (`auto`: installs them as soon as they're
found), **Off** (`off`: no checks). Pre-releases (`v1.2.3-beta.1`) are never
offered.

## What "Update now" does

- **Check**: the file and `SHA256SUMS` come from GitHub over HTTPS; the
  file's SHA-256 must match its line, else nothing is installed.
- **VPS / Linux bundle**: the new bundle's `install.sh --update` replaces only
  the binaries and hooks (config, keys and devices stay), keeps the old ones
  as `tailbondd.prev` / `mqvpn.prev`, and restarts tailbond. If it isn't up on
  the new version within 60 seconds, the old binaries go back and the
  dashboard says "went back to vX". Going back by hand:
  `sudo /usr/local/lib/tailbond/install.sh --update --version vX.Y.Z`.
- **GL router**: `opkg install` of the release's package; if the agent
  doesn't answer within 90 seconds, the previous package goes back
  (`logread -e tailbond-update`).
- Devices drop for a few seconds while their server restarts. Update the home
  server first, then routers and phones (the server's dashboard lists devices
  that are behind).

## GL.iNet firmware upgrades

A GL firmware upgrade with **Keep Settings** keeps your settings but removes
packages you installed, tailbond included. tailbond puts itself back: the
installed package is kept in `/etc/tailbond/pkg/` (on GL's keep list), and a
line in `/etc/rc.local` reinstalls it on the first boot after the upgrade
(if the copy is missing, it downloads that version from the GitHub release
and checks it against `SHA256SUMS`). The panel then says "tailbond was
reinstalled after a firmware update". Details and what GL keeps:
[glinet.md](glinet.md#firmware-upgrades).

## Releasing (maintainers)

Push a `v*` tag. CI builds everything, writes `SHA256SUMS`, attests the files
(`actions/attest-build-provenance`), creates the GitHub release, and starts a
COPR build when the `COPR_CONFIG` secret is set
([fedora-server.md](fedora-server.md#copr)). The release fails if the Android
signing secrets are missing ([android.md](android.md#release-key)).
