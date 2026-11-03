# COPR "make_srpm" method (docs/fedora-server.md, COPR): COPR runs
#   make -f .copr/Makefile srpm outdir=<dir> spec=<spec>
# as root in a Fedora chroot with network, from a clone of the tag. The SRPM
# carries prebuilt binaries (mqvpn + tailbondd, built here), so the COPR
# project builds x86_64 chroots only.
outdir ?= $(CURDIR)/out/srpm

srpm:
	dnf -y -q install git golang cmake gcc gcc-c++ make perl libevent-devel rpm-build systemd-rpm-macros
	git config --global --add safe.directory '*'
	git fetch --tags --force >/dev/null 2>&1 || true
	SRPM_OUT=$(outdir) TB_UPDATE_REPO=$${TB_UPDATE_REPO:-kaovilai/tailbond} packaging/fedora/build-rpm.sh
