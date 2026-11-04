# tailbond server image: ghcr.io/kaovilai/tailbond (docs/deploy.md).
#
# Built from this repository's own Linux bundle, the one install.sh puts on
# a VPS (static tailbondd + the patched static mqvpn), not from a download:
# deploy/docker/build-image.sh unpacks out/linux/tailbond-linux-<arch>.tar.gz
# into bundle/<amd64|arm64>/ of a fresh build context and builds this file.
# CI tests the same image it publishes on a release.
#
# Runs as root inside the container with only NET_ADMIN (the TUN device,
# nftables NAT) and NET_BIND_SERVICE (UDP 443) needed: docs/deploy.md.
# Never updates itself: a new release is a new image tag.
FROM docker.io/library/alpine:3.24@sha256:be2656061b536c1883c4034790e21d0a09c5aaaf51968b89368a3c0371ea2c9a

ARG TARGETARCH
ARG VERSION=dev
ARG REVISION=unknown

LABEL org.opencontainers.image.title="tailbond server" \
      org.opencontainers.image.description="tailbond bonding server: mqvpn multipath tunnel endpoint, NAT to the internet, dashboard" \
      org.opencontainers.image.source="https://github.com/kaovilai/tailbond" \
      org.opencontainers.image.url="https://github.com/kaovilai/tailbond" \
      org.opencontainers.image.documentation="https://github.com/kaovilai/tailbond/blob/main/docs/deploy.md" \
      org.opencontainers.image.licenses="Apache-2.0" \
      org.opencontainers.image.version="${VERSION}" \
      org.opencontainers.image.revision="${REVISION}"

# bash + jq: entrypoint and hooks; nftables: the NAT the server-up hook
# loads; iproute2: mqvpn/hooks; tini: PID 1 that reaps the hooks' children.
# hadolint ignore=DL3018
RUN apk add --no-cache bash ca-certificates iproute2 jq nftables tini

COPY bundle/${TARGETARCH}/ /opt/tailbond/
COPY entrypoint.sh /usr/local/bin/tailbond-entrypoint
COPY healthcheck.sh /usr/local/bin/tailbond-healthcheck

# Config, devices, certificates and backups: mount a volume here.
VOLUME ["/data"]
# mqvpn's tunnel (TAILBOND_PORT, UDP) and the dashboard (TCP, listens on
# all addresses only when TAILBOND_UI_PASSWORD is set).
EXPOSE 443/udp 8077/tcp

# Kept by Docker and Podman (--format docker); Kubernetes uses its own
# probes (deploy/kubernetes).
HEALTHCHECK --interval=30s --timeout=5s --start-period=30s --retries=3 CMD ["/usr/local/bin/tailbond-healthcheck"]
ENTRYPOINT ["/sbin/tini", "--", "/usr/local/bin/tailbond-entrypoint"]
