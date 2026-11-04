# Updates

tailbond is released like most open-source projects: each version is a
[GitHub release](https://github.com/kaovilai/tailbond/releases) with every
file and a `SHA256SUMS` list (plus GitHub build-provenance attestations, so
anyone can check a file with `gh attestation verify FILE -R kaovilai/tailbond`),
and an attested [SBOM](sbom.md) of what each file contains.

## How releases happen

Releases are automatic; nobody has to cut one by hand:

1. **Dependency updates** arrive as one grouped weekly Dependabot PR (security fixes as soon as they're published). If none of the updates is a major version, the PR is labelled `automerge` and merges itself once CI is green on it (`.github/workflows/dependabot-automerge.yml`). Majors, and updates Dependabot can't classify (such as the mqvpn submodule), wait for a human.
2. **Every change on `main`**, whether an auto-merged update or a human merge with green CI, gets the next patch tag (`v0.1.0`, `v0.1.1`, …) from `.github/workflows/auto-release.yml`.
3. **The tag's CI run** builds and tests everything again and publishes the GitHub release only if every job passes. A failing build never reaches devices.
4. **Installs** see the new release on their next check and update per their policy (below).

Every tailbond install looks at the latest release (hourly with **Right
away** or **Automatic**, daily with **Tell me**, and right after it starts)
and shows **Update available: vX.Y (What's new)** in its dashboard, the GL
panel or the app. Each platform then updates the way it normally does:

| Device | How updates arrive | Turn them off |
|---|---|---|
| Home server on Fedora (RPM) | `dnf`, from the [COPR repository](fedora-server.md#updates): `sudo dnf upgrade tailbond-server`, or automatically with `dnf-automatic`. The dashboard only tells you. | Settings → Updates → Off (no checks); `dnf` is up to you |
| VPS / any Linux (bundle, `install.sh`) | **Right away** (new installs): by itself, as soon as it's safe. Or **Update now** in the dashboard, or `sudo /usr/local/lib/tailbond/install.sh --update` | Settings → Updates → Off, or `install.sh --update-policy off` |
| VPS with Ansible | `tailbond_update_policy` (empty: new servers get Right away, existing ones keep theirs); a pinned `tailbond_version` wins on the next role run ([vps.md](vps.md#updates)) | `tailbond_update_policy: off` |
| Server image (Docker, Podman, Kubernetes) | Never by itself: a release is a new image tag (`ghcr.io/kaovilai/tailbond:vX.Y.Z`, `latest`); pull it and replace the container, `/data` stays ([deploy.md](deploy.md#the-server-image)). The dashboard only tells you | `TAILBOND_UPDATE_POLICY=off` (no checks) |
| GL.iNet travel router | **Right away** (new installs), **Automatic (night)**, or **Update now** in the tailbond panel | tailbond panel → Updates → Off |
| GL.iNet router, Plug-ins page (custom source) | **APPLICATIONS → Plug-ins → Upgrade** from tailbond's signed opkg feed ([glinet.md](glinet.md#install-and-update-from-gls-plug-ins-page)). It's a plain `opkg upgrade`: no health check and no rollback. The panel's updates (row above) keep working alongside it | Plug-ins → Manage Sources → delete the `tailbond` source |
| Mac (bundle, `install.sh`) | **Right away** (new installs), or **Update now** in the dashboard, or `sudo /usr/local/lib/tailbond/install.sh --update` | Updates → Off |
| Mac (Homebrew) | the dashboard shows the commands: the release's `tailbond.rb` into the local tap, `brew reinstall local/tailbond/tailbond`, `sudo brew services start tailbond` ([macos.md](macos.md#updates)) | Updates → Off |
| Android phone | **Right away** (new installs): an hourly check and a notification, *Update ready: tap to install* (Android always asks before installing an app). Hands-off: [Obtainium](android.md#updates) set to check every hour | Settings → Updates → Off; remove the app from Obtainium |

### Policies

Set in the dashboard, the GL panel, the Android app, `install.sh
--update-policy`, the Ansible variable `tailbond_update_policy`, or
`"update": {"policy": …}` in `tailbond.json`:

| Policy | Checks | Installs |
|---|---|---|
| **Right away** (`immediate`, the default for new installs) | Every hour (plus up to 10 minutes of jitter), and right after tailbond starts | As soon as it's safe: not during a call or heavy traffic (traffic both ways at 48 kbit/s or more, or 2 Mbit/s together). It waits at most **30 minutes**, then installs anyway. A **security release** doesn't wait. On Android: a notification, *Update ready: tap to install* |
| **Automatic (night)** (`auto`) | Every hour, and at start | By itself between 03:00 and 05:00 local time |
| **Tell me** (`notify`) | Once a day, and at start | Only when you press **Update now** |
| **Off** (`off`) | Never by itself (Check now still works) | Never by itself |

Devices drop for a few seconds while tailbond restarts. A new version that
doesn't start goes back by itself, and isn't installed by itself again.

**Existing installs keep their policy.** Only new installs get Right away.
That covers fresh configs from `install.sh`, `tailbondd -init` (Mac), a
first boot without a config (GL.iNet), a new Ansible server and the
Android app's default. A `tailbond.json` written before this change that
has no `update.policy` keeps **Tell me**; choose Right away in the
dashboard to switch. On Fedora, updates come from dnf, and every policy
except Off means "check and tell me".

**Servers and clients, in any order within the hour.** The protocol
ordering is unchanged: a server keeps working with older clients and
lists them as outdated, and reports its `protocol` in `/api/hello`
([architecture.md](architecture.md#updates)). Right away on both sides
just shortens how long a fleet runs mixed versions.

**Security releases.** The release job marks a release with `[security]`
in its notes when any commit since the previous release has a
`Security: yes` trailer. A maintainer can also add the marker by editing
the notes, and devices see it at their next check. Right away installs
those without waiting for a quiet moment.

**Rate limits.** The check is one `GET /repos/…/releases/latest` with
`If-None-Match` (the last answer's ETag), so an unchanged release is a
small `304` (GitHub doesn't count conditional `304`s against its rate
limit). Even counted, one request an hour per device is far inside the
60 an hour GitHub allows each address without a token.

Pre-releases (`v1.2.3-beta.1`) are never offered.

## What "Update now" does

- **Check**: the file and `SHA256SUMS` come from GitHub over HTTPS; the
  file's SHA-256 must match its line, else nothing is installed.
- **VPS / Linux bundle**: the new bundle's `install.sh --update` replaces only
  the binaries and hooks (config, keys and devices stay), keeps the old ones
  as `tailbondd.prev` / `mqvpn.prev`, and restarts tailbond. If it isn't up on
  the new version within 60 seconds, the old binaries go back and the
  dashboard says "went back to vX". Going back by hand:
  `sudo /usr/local/lib/tailbond/install.sh --update --version vX.Y.Z`.
- **Mac**: the same as the Linux bundle with `tailbond-macos-arm64.tar.gz`;
  the LaunchDaemon is restarted (`/usr/local/var/log/tailbond.log`).
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
signing secrets are missing ([android.md](android.md#updates), "The release key").

**The opkg feed key**, made once: `scripts/release/gen-opkg-key.sh --backup
DIR` creates a usign key pair. It sets the secret key as repository secret
`TB_OPKG_SIGNING_KEY`, writes the public key to
`packaging/openwrt/files/etc/opkg/keys/<fingerprint>` (commit it: the
package installs it on routers), and saves an encrypted backup in DIR
(`--restore FILE.gpg` puts the key back). With the secret set, each
release publishes `Packages`, `Packages.gz`, `Packages.sig` and
`tailbond-opkg.pub` ([glinet.md](glinet.md#install-and-update-from-gls-plug-ins-page)).
Without it, the release goes out with no feed and a warning. If the
secret's public key isn't in the package, the release fails.

**Rotating the feed key.** A feed carries one signature. A router that
only has the old public key refuses a feed signed by a new key, so it
can't fetch the release that would bring that new key. While the current
secret key still works, rotate in two steps:

1. `scripts/release/gen-opkg-key.sh --next --backup DIR` makes the new key
   and backs it up. It adds the new public key next to the current one and
   leaves the secret alone. Commit with the command it prints. Releases
   keep being signed with the current key but now ship both public keys,
   so every router that updates also gets the new one.
2. Some releases later: `scripts/release/gen-opkg-key.sh --restore
   NEW.gpg --force`. The secret becomes the new key, and the old public
   key is removed. Commit with the printed command, which also stages the
   removal.

A router that hasn't updated since before step 1 needs the new key by
hand once ([glinet.md](glinet.md#install-and-update-from-gls-plug-ins-page),
step 1). So does every router if the key is lost or leaked, because then
step 1 is impossible (or unsafe): `--force` replaces the key at once.

The script stops if it can't list the repository's secrets (`gh` not
logged in, network), because it can't tell whether a key already exists.
