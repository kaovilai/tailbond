# Backup and restore

tailbond can save everything a machine needs into one **encrypted** file
(`.tbbackup`), and put it back on the same machine or a new one. The format
and API are in [architecture.md](architecture.md#backup-and-restore).

## What's in a backup

| Machine | Contents |
|---|---|
| Home server (Fedora, VPS, Flint 3) | all settings: devices and their keys, approvers, public address, port, tunnel addresses, mode, dashboard password, update policy; the self-signed TLS certificate and key (devices pin it), the private mqvpn key; on a GL router also `/etc/config/tailbond` |
| Travel router (Slate 7) | all settings: servers and their pairings, failover policy, bypass rules, connection choices and data limits, captive-portal and update settings; `/etc/config/tailbond` |
| Android app | servers and pairings, settings, apps and websites that skip tailbond ([android.md](android.md#backup-and-restore)) |

The following are left out:
- Tailscale's own state and certificates: Tailscale reissues them.
- Usage counters.

The passphrase:
- needs at least 10 characters;
- is used with scrypt to derive an AES-256-GCM key, so the file can't be read or changed without it;
- **can't be recovered**: write it down somewhere other than the machine you're backing up.

There is no unencrypted export of keys or passwords. To see what a backup
would contain without any secrets: **See what's inside** in the dashboard,
or `tailbondd backup --plain`.

## Making a backup

- **Dashboard / GL panel**: Backup & restore → **Download a backup**.
- **Command line** (home server, VPS, router over SSH):

  ```sh
  sudo tailbondd backup --out homebox.tbbackup          # asks for the passphrase
  TB_BACKUP_PASSPHRASE=… sudo -E tailbondd backup --out homebox.tbbackup
  ```

- **Automatically**: set a backup passphrase under Backup & restore.
  - The agent then writes `<state_dir>/backups/<name>-<time>.tbbackup` once a
    day (`backup.every_hours`) and keeps the newest 7 (`backup.keep`).
  - `<state_dir>` is `/var/lib/tailbond` on Fedora and the VPS bundle, and
    `/etc/tailbond/state` on GL routers.
  - A travel router with a passphrase also sends its newest backup to its home
    server (`backup.push`), which keeps the last 3 per device in
    `<state_dir>/client-backups/<device>/`.
  - The server lists both under Backup & restore, so if the router dies its
    settings are at home. A server's own backup carries these too (each still
    locked with its device's passphrase), so they move with the server.

Copy the files somewhere else too. They're encrypted, so a cloud folder or
another machine is fine.

### On the VPS with Ansible (a `backup` role)

Add the scheduled backups directory to that role's paths:

```yaml
- /var/lib/tailbond/backups/          # encrypted .tbbackup files
- /var/lib/tailbond/client-backups/   # the routers' and phones' backups
```

Don't back up `/etc/tailbond/tailbond.json` in plain text elsewhere: it holds
the devices' keys.

## Restoring

1. Install tailbond on the target machine with the **same role** (a server
   backup goes on a server). A new machine is fine.
2. Backup & restore → **Restore from a backup**: pick the file and type the
   passphrase. A preview lists what will be replaced (devices, servers, public
   address, rules…) and warnings, e.g. "made on another machine".
3. Confirm. The agent:
   - first saves the current settings as
     `<state_dir>/backups/before-restore-<time>.tbbackup`, encrypted with the
     same passphrase, so you can go back;
   - writes the backup's settings, keeping this machine's own paths (programs,
     hooks, state directory, listen address);
   - restarts.

Command line, with the service stopped (or restart it right after):

```sh
sudo systemctl stop tailbond
sudo tailbondd restore homebox.tbbackup
sudo systemctl start tailbond
```

## Moving the home server to a new machine

Devices keep working after a restore:
- the device keys are the same;
- a self-signed certificate is the same, so the pins still match;
- with a `tailscale cert` certificate, the name is the same.

What can change is **where they connect**:

In this order, so no device is cut off (the restore preview lists the same
steps for your devices):

1. Restore on the new machine while the old one still runs; devices keep
   using the old one. A restore keeps the new machine's own public address
   and tunnel port (the ones its firewall allows).
2. Devices that have only a link (no Tailscale on them) get their new link
   from the new machine's Devices tab: a GL router takes it in its tailbond
   panel (Add another server, Paste a link: it replaces the old one), a phone
   by scanning the QR code. A travel router with the kill switch on needs
   this before step 3.
3. Turn tailbond off on the old machine (`sudo systemctl disable --now
   tailbond`) and move its Tailscale name to the new one (below); devices
   with Tailscale follow within a few minutes.

- **Same public address or DDNS name** (you moved the box, or the DDNS name
  follows it): nothing to do.
- **New address**: devices learn it as below. Port forward on your home
  router? Point it at the new machine.
  - Routers ask their servers for refreshed pairing details every 6 hours,
    and within 2 minutes once no server answers. They do this through the
    tunnel or over Tailscale (`GET /api/pairing`), so a router on Tailscale
    picks up the new address by itself.
  - Phones: tap **Ask to connect again** with the server's Tailscale name (the
    server re-issues the same credential), or scan the device's link again
    (Devices → the device → Show link).
- **Tailscale**: log the new machine into the same tailnet and give it the
  old machine's name (admin console → Machines: remove or rename the old one,
  then rename the new one). If the subnet route was approved for the old
  machine, approve it for the new one.


## A new travel router

Restore the router's backup on the new router (GL panel → tailbond → Backup &
restore). It comes back with its servers, bypass rules and choices. Its
devices on the home server are the same, so nothing to approve again.

If the router is gone, download its last pushed backup from the home server
(Backup & restore → the router's backups) and restore that.
