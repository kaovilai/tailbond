# Updates

Every tailbond install can update itself. Every release carries **GitHub's
proof of where it was built** (an artifact attestation: no signing key to
keep anywhere, see [How an update is checked](#how-an-update-is-checked-no-signing-key)),
and your home server keeps a checked copy of the latest one (the
**mirror**), so routers and phones update even where GitHub is blocked or
slow (hotel Wi-Fi, a captive portal, a phone on a train). How it fits together for developers:
[architecture.md, Updates](architecture.md#updates).

## What gets updated, and how

| Device | How it updates | What stays untouched |
|---|---|---|
| Travel router (GL.iNet, OpenWrt) | downloads the new package, installs it with `opkg`, checks that it comes back, else reinstalls the previous one | `/etc/config/tailbond`, `/etc/tailbond/` (pairing, keys, users) |
| Phone (Android app) | downloads the new APK and installs it (silently after the first time) | the app's settings |
| Home server (Fedora RPM) | unpacks the new version into `/var/lib/tailbond/update/`; the packaged `tailbondd` starts it | every file the RPM installed |
| VPS (generic bundle) | same as Fedora, under `/var/lib/tailbond/update/` | `/usr/local/bin/…`, the unit, the config |

On a Linux server the installed `tailbondd` is the **base**. Each time it
starts, it looks at `/var/lib/tailbond/update/state.json`; if a newer version
is installed there, it runs that one (and that version's `mqvpn`). To
install, the agent unpacks the new version, marks it "on trial" and exits
with status 75; systemd starts it again (`Restart=on-failure`, 3 s, and
`RestartForceExitStatus=75` so it restarts even if you changed `Restart=`).

**Fedora, `rpm -V` and `dnf upgrade`.** Updates never write to a file the
RPM owns: they live in `/var/lib/tailbond/update/` (the RPM owns only the
directory `/var/lib/tailbond`, not what's in it). So `rpm -V tailbond-server`
stays clean, and the unit's `ProtectSystem=full` (read-only `/usr`, `/etc`)
is fine: `/var/lib/tailbond` is in `ReadWritePaths=`, and nothing in the
unit forbids running programs from it (no `NoExecPaths=`). When you
`dnf upgrade` to a package newer than the self-installed version, the newer
package wins: its base runs itself and ignores the older update. If the
self-installed version is newer than the package, it keeps running.

**VPS bundle.** Re-running `install.sh` from an older bundle doesn't
downgrade a VPS that already updated itself, for the same reason. Choose
the policy at install time: `sudo ./install.sh --update-policy notify`
(`auto`, `notify`, `off`) and `--update-source URL` (your own release
server; `""` goes back to the built-in one). Re-runs without these options
keep what's there.

## Policies and the maintenance window

Set them in the dashboard (Settings → Updates) or in `tailbond.json`
(`"update": {…}`):

- **auto** (default): routers and phones install as soon as an update is
  there (unless you're in a call, below). Servers install only inside the
  **maintenance window** (`"window": "03:00-05:00"`, local time), since every
  connected device drops for a few seconds.
- **notify**: the dashboard shows the update; nothing installs until you
  press **Install**.
- **off**: no checks.
- **pin** (`"pin": "0.2.0"`): install exactly that version and nothing newer
  (for Ansible and other config management, below).
- **channel**: `stable` (default) or `beta` (pre-releases too).

## The home server mirror

A server keeps the current and the previous release of its channel (up to
`mirror_max_mb`, 600 MB by default). Its routers and phones ask it first
(`http://<server tunnel address>:8077/updates/…`, then its Tailscale name)
and fall back to GitHub. Nothing to set up. The server only offers what it
checked itself, and routers and phones take its copy as is: they reach it
through the tunnel, whose certificate they pinned when they paired, or over
your tailnet.

**No internet on the server, or GitHub blocked?** Upload a release by hand:
download `stable.json`, `stable.json.sigstore.json` (the attestation) and
the files you need from the release page (see "The channels release"
below), then in the dashboard **Settings → Updates → Upload**, or:

```sh
for f in stable.json.sigstore.json stable.json tailbond_0.2.0-r1_aarch64_cortex-a53_neon-vfpv4.ipk; do
  curl -s -b cookies.txt -H 'Content-Type: application/octet-stream' \
    --data-binary @"$f" "http://homebox:8077/api/update/upload?name=$f"
done
```

The manifest is checked against its attestation once both are there (no
internet needed: the server keeps Sigstore's public keys from its last
check), and every file against the manifest. With `"verify": "sha256-only"`
the attestation is optional (below).

## Order and compatibility

Each release says which server it needs (`requires_server_protocol`). A
router or phone installs a release only when its server already speaks
that protocol, so **servers update first**, clients follow, and a client
never gets ahead of its server. The server's dashboard lists clients that
are behind ("outdated").

## Not during a call

A router or phone doesn't install while the tunnel is busy (more than
300 kbit/s on average over the last minute: a video call, a stream). It
waits and says why in the dashboard ("deferred"). Security releases are the
exception: they install right away (a server still waits for its window).

## Rollback

- **Linux servers**: the new version has 90 seconds to show it works (its
  API answers, mqvpn runs, a server that had devices connected sees one
  again). If it doesn't, or it crashes on start more than 3 times, the base
  goes back to the previous version and that release is never installed
  again automatically. **Settings → Updates** can also go back by hand
  (`POST /api/update/install {"version": "<previous>"}`).
- **Routers**: `/usr/lib/tailbond/update-ipk` installs the new package,
  restarts tailbond and waits up to 90 seconds for the agent to answer in
  the same role (and for the tunnel to reconnect if it was connected). If
  not, it reinstalls the previous package. The outcome is in
  `/etc/tailbond/state/update-result.json` and `logread -e tailbond-update`.
  The router keeps the installed package and the previous one in
  `/etc/tailbond/pkg/` (`current` and `previous` name them); when tailbond
  was installed by hand, the running package is saved there first, so even
  the first update can roll back. Each package is about 30 MB (it carries
  the Android app), so this takes about 60 MB of the router's storage.
- **Phones**: Android installs or refuses the whole APK; a failed install
  leaves the old app.

## Firmware upgrades on GL.iNet routers

A firmware upgrade with **Keep Settings** keeps your settings but not the
packages you installed, so tailbond would be gone after it. tailbond
reinstalls itself on the first boot:

1. Before the router flashes, a hook (`/lib/upgrade/tailbond.sh`) makes
   sure `/etc/tailbond/pkg/` has the installed package and the
   `reinstall` script, and adds them to the files to keep.
2. `/etc/tailbond/` is kept (`/lib/upgrade/keep.d/tailbond`), and so is
   `/etc/rc.local`, where the package added one line.
3. On the first boot that line runs `/etc/tailbond/pkg/reinstall` in the
   background: tailbond isn't installed, so it installs the kept package
   (`opkg install`; your settings are already there and stay), then
   tailbond starts. It logs to the system log (`logread -e
   tailbond-reinstall`) and writes `update-result.json` with
   `reinstalled_after_firmware_upgrade`, which the panel shows.
4. If the kept package is missing, it asks your home server's mirror for
   one over Tailscale (the server named in your pairing, via GL's
   Tailscale), checks it against the server's manifest, and installs it.
   It retries for 20 minutes while the network or Tailscale comes up.

Without **Keep Settings** nothing survives (tailbond included); install the
package again and pair again.

What GL.iNet 4.x keeps, read from the GL-BE3600 (Slate 7) firmware 4.10.1
image (`be3600-4.10.1_release1-1129-0918-1789738103.bin`, OpenWrt
23.05-SNAPSHOT, `/sbin/sysupgrade` as in stock OpenWrt):

- **Verified** (from the image's scripts):
  - kept: every file listed in `/etc/sysupgrade.conf` and in
    `/lib/upgrade/keep.d/*` (read on the running system, so a package's own
    keep.d file counts), plus every package's changed conffiles; the whole
    `/etc/config/`; `/etc/rc.local` (`keep.d/base-files-essential`), which
    `/etc/init.d/done` (S95) runs at every boot;
  - not kept: the package itself (binaries, `/etc/init.d/tailbond`, opkg's
    records): the router comes back with tailbond's files but not tailbond;
  - `/lib/upgrade/*.sh` hooks run after the keep list is made, so files they
    create must be added to the list by name (GL's own
    `99-migrate-downgrade.sh` works the same way);
  - GL's upgrade page sends "Keep Installed Packages" together with "Keep
    Settings" (`sysupgrade -k`): it saves the list of installed packages in
    `/etc/backup/installed_packages.txt`.
- **Unverified** (inferred from strings in GL's compiled code): GL
  reinstalls that list later with `opkg install` from its own package feeds
  (Plug-ins page). tailbond isn't in GL's feeds, so that does nothing for
  it; the reinstall above doesn't depend on it.
- GL may in future mark settings not to keep for a specific firmware
  (`upgrade_control` in the firmware metadata; empty in 4.10.1). The image
  ships every package tailbond needs (`kmod-tun`, `ip-full`, `jshn`,
  `jsonfilter`), so the reinstall works offline.

## Ansible

The role (`packaging/ansible/roles/tailbond`, [vps.md](vps.md#ansible)) has
`tailbond_update_policy`:

- `auto` (default), `notify`, `off`: as above. With `auto` the agent updates
  itself, and a later playbook run never downgrades it: the role copies the
  bundle's binaries only when they're at least as new as what runs.
- `pinned`: the role is the source of truth. Set `tailbond_version` too
  (the bundle must be that version): the role installs it, pins the agent to
  it (`update.pin`), and drops a newer self-installed version.

`tailbond_update_window` and `tailbond_update_channel` set the window and
the channel.

## How an update is checked (no signing key)

There is no update-signing key to keep, rotate or lose. Instead:

- **Integrity**: the manifest (`stable.json`) lists the SHA-256 and size of
  every file; nothing is installed unless it matches. The manifest comes over
  HTTPS from GitHub's release, or from your home server.
- **Authenticity**: the release job attests the manifest and every file with
  [GitHub artifact attestations](https://docs.github.com/en/actions/security-guides/using-artifact-attestations-to-establish-provenance-for-builds)
  (`actions/attest`, Sigstore "keyless" signing): GitHub's OIDC token for
  that workflow run gets a short-lived certificate that names the repository,
  the workflow file and the tag, the signature is recorded in Sigstore's
  public transparency log, and the private key is thrown away. The release
  puts the attestation next to the manifest (`stable.json.sigstore.json`).
  A server checks it with [sigstore-go](https://github.com/sigstore/sigstore-go)
  against Sigstore's public-good trusted root (fetched and kept up to date
  with TUF, cached in `/var/lib/tailbond/update/sigstore`) and accepts the
  manifest only if the certificate says **this repository**
  (`github.com/<owner>/<repo>`), **this workflow**
  (`.github/workflows/ci.yml`), a **version tag** (`refs/tags/v…`), and the
  attestation is SLSA build provenance. If the attestation file is missing it
  asks GitHub's API (`/repos/<owner>/<repo>/attestations/sha256:<digest>`).
  You can check any release file yourself the same way:
  `gh attestation verify tailbond-linux-x86_64.tar.gz -R <owner>/<repo>`.
- **Routers and phones** don't check attestations. They take updates from
  their home server, which checked them, over a channel that's already
  authenticated (the tunnel's pinned certificate, or your tailnet), and then
  check every file's SHA-256. A server you paired with can already see and
  steer your traffic, so trusting its update copy adds no new party.
  A phone that downloads from GitHub directly also has Android's own check:
  an update must be signed with the same key as the installed app (below).

Settings (`"update": {…}`):

- `"verify": "attestation"` (default) or `"sha256-only"`. **sha256-only**
  is for setups without GitHub (uploads by hand, your own release server):
  every file is still checked against the manifest, but the manifest itself
  is trusted because of where it came from (HTTPS, or you uploaded it). That
  protects against damaged downloads, not against someone who can change the
  manifest at its source. Check uploads yourself first with
  `gh attestation verify` on a computer that has GitHub.
- `"repo": "owner/name"` and `"workflow": ".github/workflows/ci.yml"`: whose
  releases count; the default is the repository the build was made by
  (`TB_UPDATE_REPO` / `TB_UPDATE_WORKFLOW` at build time). Set it for a
  fork that publishes its own releases. A build made without a repository
  (a local build) only updates from its server's mirror, or with one of
  these set.
- `"trusted_root": "/path/trusted_root.json"`: for a private repository
  (GitHub's own Sigstore instance: `gh attestation trusted-root >file`).
- `TB_GITHUB_TOKEN` in the agent's environment raises GitHub's API limit
  (60 requests an hour without it; a check uses at most one, and none when
  the attestation file is there).

The dashboard shows how the offered update was checked: "verified: built
by <owner>/<repo>'s release workflow on GitHub", "verified by <server>"
(routers, phones) or "checksums only".

**The router doesn't carry the attestation check.** The verifier
(sigstore-go and its Sigstore, TUF and transparency-log code) costs, measured
on tailbondd for arm64 (stripped, Go 1.26): **+8.4 MB** binary (12.3 →
20.7 MB), **+2.7 MB** per compressed package, which the router keeps twice
for rollback (`/etc/tailbond/pkg/`), and **+5.5 MB** memory at start
(package initialisation: 4.6 → 10 MB before the agent does anything), plus
about 350 more dependencies in `go.sum`. That's too much for 128–256 MB
flash travel routers that only ever need their server's copy, so the router
package is built with `-tags noattest`. Servers (RPM, the Linux bundle,
VPS) carry it. A **GL router used as the home server** therefore can't
check attestations: its default is `sha256-only` (HTTPS from GitHub plus
checksums), the panel says so, and setting `"verify": "attestation"` there
makes it refuse every update with a clear message. If you want the full
check for a GL home server, upload releases you checked with
`gh attestation verify`, or run the home server on Linux.

**What it doesn't protect against**: GitHub itself, or someone who can push
a tag and run this repository's release workflow (they would produce a valid
attestation; that's what branch and tag protection are for). An old,
validly attested manifest replayed by a network attacker is harmless (a
device never installs a version older than it runs, except a version you
pinned), but it can hold devices back on the old version until the real one
gets through.

## Android signing key

Android needs every APK signed, and an update must carry **the same key** as
the app on the phone; this one key can't be avoided. It lives only in the
repository's secrets and one encrypted backup:

```sh
scripts/release/gen-android-keystore.sh --backup /path/outside/any/repo
```

makes the key once (RSA 4096, 30 years) and stores it as the secrets
`TB_RELEASE_KEYSTORE_B64`, `TB_RELEASE_KEYSTORE_PASSWORD`,
`TB_RELEASE_KEY_ALIAS` and `TB_RELEASE_KEY_PASSWORD` (`gh secret set`), the
certificate's SHA-256 as the variable `TB_RELEASE_CERT_SHA256`, and an
encrypted copy (`tailbond-android-release-key-<date>.tar.gpg`, gpg AES-256
with a passphrase you choose) in the folder you name, which must not be
inside a git checkout (a password manager's attachments, a USB stick). The
temporary copy is wiped; nothing goes into the repository. It refuses to
replace an existing key unless you pass `--force`.

CI signs release APKs with those secrets and checks that the APK's
certificate is `TB_RELEASE_CERT_SHA256`, so the key can't change by
accident. Without the secrets CI still builds the debug APK, but a release
tag **fails** with a message instead of publishing an APK that phones can't
update to.

Secrets deleted, or a new repository? Put the same key back:
`scripts/release/gen-android-keystore.sh --restore <backup>.tar.gpg`.

**If the key is lost** (no secrets, no backup): every phone has to uninstall
the app and install the new one once (Android refuses an update signed with
another key), then pair again or restore its settings from a backup
(`.tbbackup`, [backup-restore.md](backup-restore.md)). Keep the backup and
its passphrase.

## The `channels` release (CI)

On every `v*` tag, CI (`.github/workflows/ci.yml`, job `release`):

1. builds everything with `TB_UPDATE_REPO` (this repository),
   `TB_UPDATE_WORKFLOW` and
   `TB_UPDATE_SOURCE=https://github.com/<owner>/<repo>/releases/download/channels`
   compiled in;
2. makes the manifests (`scripts/release/publish-channels.sh prepare`,
   `make-manifest.sh`);
3. attests every file and the manifests in one attestation
   (`actions/attest`; permissions `id-token`, `attestations`,
   `artifact-metadata`);
4. creates the version's release with every file, the manifests and the
   attestation (`<channel>.json.sigstore.json`), then replaces
   `stable.json` / `beta.json` and their `.sigstore.json` in the release
   named `channels` (created on first use; that's the URL above)
   (`publish-channels.sh publish`).

A plain tag (`v1.2.3`) updates `stable.json` and `beta.json`; a pre-release
tag (`v1.2.3-beta.1`) is a GitHub pre-release and updates only `beta.json`.
A channel never moves back to an older version. Put `[security]` in the
tag message (`git tag -a v1.2.4 -m "[security] …"`) to mark the release as
a security release. No secret is needed for the attestations. Release APKs
need the Android signing secrets ([above](#android-signing-key)); without
them the release job fails.

Release files: `tailbond-linux-<arch>.tar.gz`,
`tailbond_<version>-r1_aarch64_cortex-a53.ipk`,
`tailbond_<version>-r1_aarch64_cortex-a53_neon-vfpv4.ipk`,
`tailbond-server-<version>-1.fc44.x86_64.rpm`, `tailbond-<version>.apk`.

## Android: Obtainium

The app updates itself (from your server's mirror, then GitHub). If you'd
rather have an app store do it, point [Obtainium](https://obtainium.imranr.dev/)
at this repository's GitHub releases; details in [android.md](android.md).
