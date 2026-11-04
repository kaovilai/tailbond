# tailbond home/VPS server in a container. UNTESTED TEMPLATE (see docs/deploy.md).
#
# Needs a host that gives the container a TUN device, NET_ADMIN and inbound
# UDP: Docker/Podman on a VM, or Fly.io. It can't run on Heroku, Vercel,
# Cloudflare Workers/Pages or Railway (no inbound UDP, no TUN).
#
# Uses the release bundle (tailbond-linux-<arch>.tar.gz) that install.sh uses
# on a VPS; set TAILBOND_VERSION to pin a release, default "latest".
FROM docker.io/library/debian:12-slim

ARG TAILBOND_REPO=kaovilai/tailbond
ARG TAILBOND_VERSION=latest
ARG TARGETARCH

# hadolint ignore=DL3008
RUN apt-get update \
 && apt-get install -y --no-install-recommends ca-certificates curl jq nftables iproute2 procps \
 && rm -rf /var/lib/apt/lists/*

SHELL ["/bin/bash", "-o", "pipefail", "-c"]
# The checksum check runs in a subshell in /tmp; nothing after it depends on the cwd.
# hadolint ignore=DL3003
RUN case "${TARGETARCH:-amd64}" in \
      amd64) arch=x86_64 ;; \
      arm64) arch=aarch64 ;; \
      *) echo "unsupported arch ${TARGETARCH}" >&2; exit 1 ;; \
    esac \
 && if [ "$TAILBOND_VERSION" = latest ]; then base="https://github.com/${TAILBOND_REPO}/releases/latest/download"; \
    else base="https://github.com/${TAILBOND_REPO}/releases/download/${TAILBOND_VERSION}"; fi \
 && curl -fsSLo /tmp/bundle.tar.gz "$base/tailbond-linux-$arch.tar.gz" \
 && curl -fsSLo /tmp/SHA256SUMS "$base/SHA256SUMS" \
 && (cd /tmp && grep " tailbond-linux-$arch.tar.gz\$" SHA256SUMS | sed "s|  .*|  bundle.tar.gz|" | sha256sum -c -) \
 && mkdir -p /opt/tailbond \
 && tar -C /opt/tailbond --strip-components=1 -xzf /tmp/bundle.tar.gz \
 && rm -f /tmp/bundle.tar.gz /tmp/SHA256SUMS

COPY entrypoint.sh /usr/local/bin/tailbond-entrypoint
RUN chmod 0755 /usr/local/bin/tailbond-entrypoint

# Config, users, certificates and backups live here: mount a volume.
VOLUME ["/data"]
# mqvpn's tunnel port (UDP); the dashboard (TCP 8077) is only published when
# TAILBOND_UI_PASSWORD is set.
EXPOSE 443/udp 8077/tcp

ENTRYPOINT ["/usr/local/bin/tailbond-entrypoint"]
