# Deploy a server: cloud-init, containers, Kubernetes, Fly.io

The supported, tested ways to run a server are the [Fedora RPM](fedora-server.md), the [VPS installer](vps.md), a [GL.iNet router](glinet.md) and, below, the **server image** in Docker, Podman or Kubernetes. Please [open an issue](https://github.com/kaovilai/tailbond/issues/new/choose) with what happened if you try one of the untested templates.

A tailbond server needs three things from where it runs:

1. **Inbound UDP** (mqvpn's tunnel port, 443 by default);
2. **a TUN device with network-admin rights** (`/dev/net/tun`, `NET_ADMIN`);
3. **a long-running process** that can set up NAT.

That rules out **Heroku, Vercel, Cloudflare Workers/Pages and Railway**: they route only HTTP(S) or TCP into unprivileged containers or functions, so tailbond can't run there.

| Template | Where | Files | Tested |
|---|---|---|---|
| [Any VPS via cloud-init](#any-vps-cloud-init) | DigitalOcean, Hetzner, Vultr, Linode/Akamai, Oracle Cloud, AWS, GCP… | `deploy/cloud-init/cloud-init.yaml` | no (the installer it runs is) |
| [Docker / Podman](#docker--podman) | Any Linux host with Docker or Podman | image `ghcr.io/kaovilai/tailbond`, `deploy/docker/compose.yaml` | CI |
| [Kubernetes](#kubernetes) | Any cluster that can route UDP to a pod | `deploy/kubernetes/` (kustomize) | CI (kind) |
| [Fly.io](#flyio) | Fly Machines (Firecracker VMs, UDP on a dedicated IPv4) | `deploy/fly/fly.toml` | no |

cloud-init downloads the same release bundle as the [VPS installer](vps.md) (`releases/latest/download/tailbond-linux-<arch>.tar.gz`, checked against the release's `SHA256SUMS`), so it needs a published [release](https://github.com/kaovilai/tailbond/releases).

## Any VPS (cloud-init)

1. Create a server with Debian 12, Ubuntu 24.04, AlmaLinux/Rocky 9–10 or Fedora. 512 MB RAM is enough ([sizing](vps.md)).
2. Paste [`deploy/cloud-init/cloud-init.yaml`](../deploy/cloud-init/cloud-init.yaml) into the provider's **User data** / **cloud-init** box. Optionally fill in `TAILSCALE_AUTHKEY` so the server joins your tailnet; then phones and routers reach your Tailscale machines through it. Clients dial the server's public IPv4 (looked up from the internet, so it's right behind cloud NAT); set `PUBLIC_ADDR` to use a DNS name instead.
3. Allow inbound UDP 443 in the provider's firewall (the installer picks another port if something already holds 443; see its log in `/var/log/cloud-init-output.log`).
4. Open the dashboard over Tailscale (`http://<server>:8077`) or an SSH tunnel (`ssh -L 8077:127.0.0.1:8077 root@<server>`) and add your devices.

## The server image

`ghcr.io/kaovilai/tailbond` (linux/amd64, linux/arm64) is the server from the Linux bundle (the same static `tailbondd` and patched static mqvpn) on Alpine, built in CI from the commit it's tagged with ([`deploy/docker/Dockerfile`](../deploy/docker/Dockerfile)). It is pushed only by a release, tagged `vX.Y.Z`, `X.Y` and `latest` (pre-releases: `vX.Y.Z-…` only), with a build-provenance attestation: `gh attestation verify oci://ghcr.io/kaovilai/tailbond:vX.Y.Z -R kaovilai/tailbond`.

- **Rights:** it runs as root *inside* the container with every capability dropped except **`NET_ADMIN`** (create the TUN device, set its address, load the nftables NAT) and **`NET_BIND_SERVICE`** (UDP 443; not needed on a port ≥ 1024), plus the device **`/dev/net/tun`**. No `--privileged`. Rootless Podman should also work (root in the container is your user on the host) but isn't tested.
- **IP forwarding** must be on in the container's network namespace: `--sysctl net.ipv4.ip_forward=1` (the Compose file and the Kubernetes manifests do it). Without it, devices reach only the server.
- **Environment:** `TAILBOND_PUBLIC_ADDR` (required: what clients dial; `host:port` when the published port differs from `TAILBOND_PORT`), `TAILBOND_PORT` (default 443), `TAILBOND_SUBNET` (default `10.77.0.0/24`), `TAILBOND_UI_PASSWORD`, `TAILBOND_MODE`, `TAILBOND_UPDATE_POLICY` (`notify`, the default, or `off`). They're re-applied on every start; paired devices stay.
- **Data:** config, paired devices, the server certificate and backups live in **`/data`**: keep it on a volume, and a new container (a new image) carries on with the same devices.
- **Dashboard:** TCP 8077. With `TAILBOND_UI_PASSWORD` set it listens on all of the container's addresses (publish it on `127.0.0.1` only and use an SSH tunnel); without one, only on the container's own `127.0.0.1`.
- **Health:** `HEALTHCHECK` (Docker, Podman) runs `tailbond-healthcheck`: the agent answers and mqvpn runs. The Kubernetes probes use the same command.
- **Updates:** the image never updates itself (the dashboard only says when a release is out). Update by pulling a new tag and replacing the container; `/data` stays. Pin `vX.Y.Z` to choose when.
- **No Tailscale in the container:** devices pair with the QR code or link from the dashboard, and the server is an internet exit. For tailnet access, use the [Kubernetes Tailscale overlay](#kubernetes), or cloud-init or the VPS installer on a host with Tailscale.

## Docker / Podman

```sh
cd deploy/docker
TAILBOND_PUBLIC_ADDR=vps.example.net TAILBOND_UI_PASSWORD=change-me podman compose up -d   # or: docker compose up -d
```

Or without Compose:

```sh
podman network create --subnet 10.89.77.0/24 tailbond
podman run -d --name tailbond --restart unless-stopped --network tailbond --ip 10.89.77.2 \
  --cap-drop all --cap-add NET_ADMIN --cap-add NET_BIND_SERVICE --device /dev/net/tun \
  --sysctl net.ipv4.ip_forward=1 --sysctl net.ipv6.conf.all.forwarding=1 \
  -p 443:443/udp -p 127.0.0.1:8077:8077 -v tailbond-data:/data \
  -e TAILBOND_PUBLIC_ADDR=vps.example.net -e TAILBOND_UI_PASSWORD=change-me \
  ghcr.io/kaovilai/tailbond:latest
```

- **A fixed container address** (`--ip`, `ipv4_address` in the Compose file): the host's connection tracking keeps sending a router's UDP flow to the address the container had. A replacement container on a new address never sees that router while it keeps sending; with the same address it reconnects by itself. (Or flush it by hand: `conntrack -D -p udp --dport 443`.)
- **Pin a release:** `TAILBOND_IMAGE=ghcr.io/kaovilai/tailbond:vX.Y.Z` for Compose. **Update:** `podman compose pull && podman compose up -d`.
- **Build from source:** build the Linux bundle (`packaging/linux/build-bundle.sh`, which needs `scripts/build-mqvpn-musl.sh x86_64` first), then `deploy/docker/build-image.sh` makes `localhost/tailbond:dev` (`PODMAN="sudo podman"` for rootful storage); run it with `TAILBOND_IMAGE=localhost/tailbond:dev`.
- Podman runs rootful (`sudo podman …`) in what's tested; `podman compose` needs a provider (`podman-compose` or `docker-compose`).

**What CI proves** (job "Server image", on amd64 and arm64; [`scripts/e2e-oci.sh`](../scripts/e2e-oci.sh), [`scripts/test-compose.sh`](../scripts/test-compose.sh)): the image runs in rootful Podman with only the two capabilities above; a real tailbond client in a network namespace bonds **two paths** (one with netem delay and loss) through the published UDP port; its internet traffic leaves through the container's NAT (a fake internet it can reach only through the tunnel sees the server side's address); the dashboard needs the password and answers; the `HEALTHCHECK` passes; a new container on the same volume keeps the paired device and certificate, and the client reconnects by itself. `compose.yaml` comes up under `podman compose` and `docker compose` (NAT loaded, Docker reports healthy). **Not tested:** rootless Podman, real hosts and cloud firewalls, IPv6 through the tunnel.

## Kubernetes

```sh
# edit deploy/kubernetes/base/configmap.yaml: TAILBOND_PUBLIC_ADDR (the Service's external address)
kubectl apply -k deploy/kubernetes/base
kubectl -n tailbond get svc tailbond                        # EXTERNAL-IP: what clients dial
kubectl -n tailbond port-forward deploy/tailbond 8077:8077  # the dashboard: http://127.0.0.1:8077
```

[`deploy/kubernetes/base`](../deploy/kubernetes/base) (kustomize): a namespace (Pod Security `privileged`: `NET_ADMIN` is above `baseline`), a ConfigMap with the environment, a Secret with the dashboard password (empty: the dashboard listens only on the pod's `127.0.0.1`, which `kubectl port-forward` reaches), a 1 Gi PVC for `/data`, a one-replica Deployment with the `Recreate` strategy (never two servers on one subnet) and a `LoadBalancer` Service for UDP 443 with `externalTrafficPolicy: Local` (routers' own addresses, no SNAT).

- **Rights:** the container drops all capabilities but `NET_ADMIN` and `NET_BIND_SERVICE`, read-only root filesystem, no privilege escalation, `RuntimeDefault` seccomp. `/dev/net/tun` comes from the node (a `hostPath` `CharDevice`; runc's and crun's default device rules let a container open it).
- **IP forwarding:** a short **privileged init container** runs `sysctl -w net.ipv4.ip_forward=1` in the pod's own network namespace. The alternative, `securityContext.sysctls`, needs `net.ipv4.ip_forward` in every kubelet's `--allowed-unsafe-sysctls`; the init container needs no node change, and the namespace is `privileged` for `NET_ADMIN` anyway.
- **No load balancer** (bare metal, kind, k3s without one): the Service still gets a nodePort. Set `TAILBOND_PUBLIC_ADDR=<node address>:<nodePort>` (pin it with `nodePort:` in an overlay).
- **Pin a release** with `images:` in an overlay of your own (`newTag: vX.Y.Z`); updates are a new tag and `kubectl apply` (`imagePullPolicy: IfNotPresent`).
- **Tailscale sidecar** ([`deploy/kubernetes/tailscale`](../deploy/kubernetes/tailscale)): the base plus `tailscale/tailscale` in kernel mode in the same pod (`TS_AUTHKEY` from Secret `tailscale-auth`, state on the PVC), advertising the tunnel subnet, so tailnet devices reach bonded clients and you reach the dashboard over the tailnet. `kubectl -n tailbond create secret generic tailscale-auth --from-literal=TS_AUTHKEY=tskey-auth-…`, then `kubectl apply -k deploy/kubernetes/tailscale`, and approve the route in the admin console.

**What CI proves** (job "Kubernetes"): both kustomizations pass `kubeconform -strict`; in a **kind** cluster the base runs the amd64 image as written (init container, capabilities, read-only root, `hostPath` TUN, PVC) and a client outside the cluster bonds two paths through the Service (host UDP 443 → nodePort → pod), its internet traffic goes through the pod's NAT, the dashboard answers through `kubectl port-forward`, and after `kubectl rollout restart` the new pod keeps the device on the PVC and the client reconnects. **Not tested:** real clusters and cloud load balancers (UDP `LoadBalancer` support varies), other CNIs and runtimes, Pod Security admission in a real cluster, and the **Tailscale sidecar** (it needs a tailnet).

## Fly.io

**Untested.**

```sh
cd deploy/fly
fly launch --copy-config --no-deploy
fly ips allocate-v4                       # UDP needs a dedicated IPv4
fly volumes create tailbond_data --size 1
fly secrets set TAILBOND_PUBLIC_ADDR=<that IPv4> TAILBOND_UI_PASSWORD=<password>
fly deploy
```

It runs the [server image](#the-server-image). **Open question to check first:** Fly delivers UDP to the `fly-global-services` address, not `0.0.0.0`. mqvpn listens on `0.0.0.0`, so the tunnel may not receive packets until tailbond gets a listen-address option.
