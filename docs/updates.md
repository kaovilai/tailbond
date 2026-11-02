# Updates

Every tailbond install can update itself. Releases are **signed**, and your
home server keeps a copy of the latest one (the **mirror**), so routers and
phones update even where GitHub is blocked or slow (hotel Wi-Fi, a captive
portal, a phone on a train). How it fits together for developers:
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
and fall back to GitHub. Nothing to set up.

**No internet on the server, or GitHub blocked?** Upload a release by hand:
download `stable.json`, `stable.json.sig` and the files you need from the
release page (see "The channels release" below), then in the dashboard
**Settings → Updates → Upload**, or:

```sh
for f in stable.json stable.json.sig tailbond_0.2.0-r1_aarch64_cortex-a53_neon-vfpv4.ipk; do
  curl -s -b cookies.txt -H 'Content-Type: application/octet-stream' \
    --data-binary @"$f" "http://homebox:8077/api/update/upload?name=$f"
done
```

The manifest is checked against its signature once both are there, and
every file against the manifest.

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

## Signing keys

Builds trust the public key compiled into them (`TB_UPDATE_PUBKEY`); the
release job signs the manifests with the private key (`TB_UPDATE_KEY`). A
build without a key never updates itself ("this build has no update key").
Self-built servers can add keys in `"update": {"trusted_keys": ["<base64>"]}`.

Make a key (OpenSSL 3), outside the repository:

```sh
scripts/release/gen-update-key.sh            # -> ~/.config/tailbond/update-keys/
gh secret set TB_UPDATE_KEY < ~/.config/tailbond/update-keys/tailbond-update-<id>.pem
gh variable set TB_UPDATE_PUBKEY --body "$(cat ~/.config/tailbond/update-keys/tailbond-update-<id>.pub.b64)"
```

Keep the `.pem` offline as well (a password manager): whoever has it can
publish updates every install accepts.

**Rotating**: make a new key, set `TB_UPDATE_PUBKEY` to `old,new` (both
trusted) and release once while still signing with the old key. When your
devices run that release, switch `TB_UPDATE_KEY` to the new key, and later
drop the old one from `TB_UPDATE_PUBKEY`. If the old key leaked, devices that
trust only it must be updated by hand.

## The `channels` release (CI)

On every `v*` tag, CI (`.github/workflows/ci.yml`, job `release`):

1. builds everything with the key and
   `TB_UPDATE_SOURCE=https://github.com/<owner>/<repo>/releases/download/channels`
   compiled in;
2. makes the manifests (`scripts/release/make-manifest.sh`) and signs them
   (`sign-manifest.sh`);
3. creates the version's release with every file and the manifests, then
   replaces `stable.json` / `beta.json` (and `.sig`) in the release named
   `channels` (created on first use; that's the URL above).

A plain tag (`v1.2.3`) updates `stable.json` and `beta.json`; a pre-release
tag (`v1.2.3-beta.1`) is a GitHub pre-release and updates only `beta.json`.
A channel never moves back to an older version. Put `[security]` in the
tag message (`git tag -a v1.2.4 -m "[security] …"`) to mark the release as
a security release. Without the `TB_UPDATE_KEY` secret the release still
happens but `channels` isn't touched (a warning says so). Release APKs need
the secrets `TB_RELEASE_KEYSTORE_B64` (base64 of the `.jks`),
`TB_RELEASE_KEYSTORE_PASSWORD`, `TB_RELEASE_KEY_ALIAS` and
`TB_RELEASE_KEY_PASSWORD`.

Release files: `tailbond-linux-<arch>.tar.gz`,
`tailbond_<version>-r1_aarch64_cortex-a53.ipk`,
`tailbond_<version>-r1_aarch64_cortex-a53_neon-vfpv4.ipk`,
`tailbond-server-<version>-1.fc44.x86_64.rpm`, `tailbond-<version>.apk`.

## Android: Obtainium

The app updates itself (from your server's mirror, then GitHub). If you'd
rather have an app store do it, point [Obtainium](https://obtainium.imranr.dev/)
at this repository's GitHub releases; details in [android.md](android.md).
