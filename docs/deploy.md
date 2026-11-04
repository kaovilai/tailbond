# One-click server templates

> [!WARNING]
> **None of these templates has been tested yet.** They're starting points that pass linting only. The supported, tested ways to run a server are the [Fedora RPM](fedora-server.md), the [VPS installer](vps.md) and a [GL.iNet router](glinet.md). Please [open an issue](https://github.com/kaovilai/tailbond/issues/new/choose) with what happened if you try one.

A tailbond server needs three things from where it runs:

1. **Inbound UDP** (mqvpn's tunnel port, 443 by default);
2. **a TUN device with network-admin rights** (`/dev/net/tun`, `NET_ADMIN`);
3. **a long-running process** that can set up NAT.

That rules out **Heroku, Vercel, Cloudflare Workers/Pages and Railway**: they route only HTTP(S) or TCP into unprivileged containers or functions, so tailbond can't run there.

| Template | Where | Files |
|---|---|---|
| [Any VPS via cloud-init](#any-vps-cloud-init) | DigitalOcean, Hetzner, Vultr, Linode/Akamai, Oracle Cloud, AWS, GCP… | `deploy/cloud-init/cloud-init.yaml` |
| [Docker / Podman](#docker--podman) | Any Linux host with Docker or Podman | `deploy/docker/` (Dockerfile, entrypoint, `compose.yaml`) |
| [Fly.io](#flyio) | Fly Machines (Firecracker VMs, UDP on a dedicated IPv4) | `deploy/fly/fly.toml` + `deploy/docker/` |

All three download the same release bundle as the [VPS installer](vps.md) (`releases/latest/download/tailbond-linux-<arch>.tar.gz`, checked against the release's `SHA256SUMS`), so they need a published [release](https://github.com/kaovilai/tailbond/releases).

## Any VPS (cloud-init)

1. Create a server with Debian 12, Ubuntu 24.04, AlmaLinux/Rocky 9–10 or Fedora. 512 MB RAM is enough ([sizing](vps.md)).
2. Paste [`deploy/cloud-init/cloud-init.yaml`](../deploy/cloud-init/cloud-init.yaml) into the provider's **User data** / **cloud-init** box. Optionally fill in `TAILSCALE_AUTHKEY` so the server joins your tailnet; then phones and routers reach your Tailscale machines through it. Clients dial the server's public IPv4 (looked up from the internet, so it's right behind cloud NAT); set `PUBLIC_ADDR` to use a DNS name instead.
3. Allow inbound UDP 443 in the provider's firewall (the installer picks another port if something already holds 443; see its log in `/var/log/cloud-init-output.log`).
4. Open the dashboard over Tailscale (`http://<server>:8077`) or an SSH tunnel (`ssh -L 8077:127.0.0.1:8077 root@<server>`) and add your devices.

## Docker / Podman

```sh
cd deploy/docker
TAILBOND_PUBLIC_ADDR=vps.example.net TAILBOND_UI_PASSWORD=change-me docker compose up -d --build
```

Podman: `podman compose up -d --build` (or `podman-compose`), run as root so the container can use `/dev/net/tun`.

- **Environment:** `TAILBOND_PUBLIC_ADDR` (required: what clients dial), `TAILBOND_PORT` (default 443), `TAILBOND_SUBNET` (default `10.77.0.0/24`), `TAILBOND_UI_PASSWORD`, `TAILBOND_MODE`. The Compose file passes all of them through and publishes `TAILBOND_PORT` on the host.
- **Data:** config, devices and certificates live in the `tailbond-data` volume (`/data`).
- **Dashboard:** published only on `127.0.0.1:8077` of the host. Reach it with an SSH tunnel. It listens inside the container only when `TAILBOND_UI_PASSWORD` is set.
- **No Tailscale in the container:** devices pair with the QR code or link from the dashboard, and the server is an internet exit without tailnet access. For tailnet access, use cloud-init or the VPS installer on a host with Tailscale instead.

## Fly.io

```sh
cd deploy/docker
fly launch --copy-config --no-deploy -c ../fly/fly.toml
fly ips allocate-v4                       # UDP needs a dedicated IPv4
fly volumes create tailbond_data --size 1
fly secrets set TAILBOND_PUBLIC_ADDR=<that IPv4> TAILBOND_UI_PASSWORD=<password>
fly deploy -c ../fly/fly.toml
```

**Open question to check first:** Fly delivers UDP to the `fly-global-services` address, not `0.0.0.0`. mqvpn listens on `0.0.0.0`, so the tunnel may not receive packets until tailbond gets a listen-address option. The same Docker notes apply (no Tailscale inside).
