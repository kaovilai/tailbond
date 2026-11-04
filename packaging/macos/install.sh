#!/bin/bash
# SPDX-License-Identifier: Apache-2.0
#
# Install tailbond on a Mac (the client/hub role, docs/macos.md) from this
# unpacked bundle (tailbond-macos-arm64.tar.gz):
#
#   sudo ./install.sh                       # install and start (unpaired)
#   sudo ./install.sh --pair 'tailbond://pair?d=…'
#   sudo /usr/local/lib/tailbond/install.sh --update   # the latest release
#
# Puts tailbondd and mqvpn in /usr/local/bin, the hooks in
# /usr/local/lib/tailbond/hooks, the config in /usr/local/etc/tailbond,
# state in /usr/local/var/lib/tailbond, the log in
# /usr/local/var/log/tailbond.log, and the LaunchDaemon
# com.tailbond.tailbondd (root: mqvpn needs a utun device). The dashboard
# is http://127.0.0.1:8077 on this Mac only.
#
# Unsigned binaries, no Apple Developer ID: the bundle's quarantine flag
# (set when a browser downloads it) is removed here, which is all Gatekeeper
# needs for software you chose to install.
#
# --update replaces only the binaries and hooks (config and state stay),
# keeps the old ones as <file>.prev, restarts tailbond and puts .prev back if
# the new version isn't up within 60 s. Without --bundle-dir it downloads
# the latest release (checked against its SHA256SUMS). The dashboard's
# "Update now" runs the new bundle's install.sh --update.
#
# Env (tests): TB_API_URL (GitHub API, default https://api.github.com),
# TB_PREFIX (default /usr/local), TB_LAUNCHD_DIR (default
# /Library/LaunchDaemons), TB_NO_LAUNCHD=1 (don't touch launchd).
# Runs with macOS's own bash 3.2.
set -euo pipefail

LABEL=com.tailbond.tailbondd
PREFIX="${TB_PREFIX:-/usr/local}"
BINDIR="$PREFIX/bin" LIBDIR="$PREFIX/lib/tailbond" HOOKS="$PREFIX/lib/tailbond/hooks"
ETC="$PREFIX/etc/tailbond" STATE="$PREFIX/var/lib/tailbond" LOGDIR="$PREFIX/var/log"
CONF="$ETC/tailbond.json" MANIFEST="$ETC/install.env"
PLIST="${TB_LAUNCHD_DIR:-/Library/LaunchDaemons}/$LABEL.plist"
TB_API_URL="${TB_API_URL:-https://api.github.com}"
BUNDLE="$(cd "$(dirname "$0")" && pwd)"

say()  { printf '==> %s\n' "$*"; }
warn() { printf 'warning: %s\n' "$*" >&2; }
die()  { printf 'install.sh: %s\n' "$*" >&2; exit 1; }

usage() {
    sed -n '4,10p' "$0" | sed 's/^# \{0,1\}//'
    cat <<'U'
Options:
  --pair URI          pair with your home server (the link from its Devices tab)
  --update            update the binaries only (see the header of this script)
  --version vX.Y.Z    with --update: that release instead of the latest
  --repo OWNER/NAME   with --update: where releases come from (kaovilai/tailbond)
  --bundle-dir DIR    with --update: install this unpacked bundle, no download
  --force             with --update: reinstall even the same version
U
}

# set_kv FILE KEY VALUE: KEY=VALUE in a KEY=VALUE file (no sed -i: BSD and
# GNU sed disagree on it)
set_kv() {
    local tmp="$1.tbnew"
    grep -v "^$2=" "$1" >"$tmp" || true
    printf '%s=%s\n' "$2" "$3" >>"$tmp"
    mv -f "$tmp" "$1"
}

PAIR="" UPDATE=0 OPT_UVERSION="" OPT_REPO="" FORCE=0
while [ $# -gt 0 ]; do
    case "$1" in
        --pair) [ $# -ge 2 ] || die "--pair needs a value"; PAIR="$2"; shift ;;
        --update) UPDATE=1 ;;
        --version) [ $# -ge 2 ] || die "--version needs a value"; OPT_UVERSION="$2"; shift ;;
        --repo) [ $# -ge 2 ] || die "--repo needs a value"; OPT_REPO="$2"; shift ;;
        --bundle-dir) [ $# -ge 2 ] || die "--bundle-dir needs a value"; BUNDLE="$(cd "$2" && pwd)"; shift ;;
        --force) FORCE=1 ;;
        -h|--help) usage; exit 0 ;;
        *) die "unknown option $1 (--help)" ;;
    esac
    shift
done

[ "$(uname -s)" = Darwin ] || die "this is the macOS installer (Linux: packaging/linux/install.sh)"
[ "$(id -u)" = 0 ] || die "run as root: sudo $0"

# ---- launchd ------------------------------------------------------------------------
launchd_on() { [ -z "${TB_NO_LAUNCHD:-}" ]; }
tb_stop() {
    launchd_on || return 0
    launchctl bootout "system/$LABEL" 2>/dev/null || true
}
tb_start() {
    launchd_on || return 0
    launchctl bootstrap system "$PLIST" 2>/dev/null || launchctl kickstart -k "system/$LABEL" || true
}
tb_restart() {
    launchd_on || return 0
    if launchctl print "system/$LABEL" >/dev/null 2>&1; then launchctl kickstart -k "system/$LABEL" || true
    else tb_start; fi
}
# hello_version: the running agent's version (GET /api/hello)
hello_version() {
    curl -fsS -m 3 http://127.0.0.1:8077/api/hello 2>/dev/null | sed -n 's/.*"version":"\([^"]*\)".*/\1/p'
}
unquarantine() { xattr -dr com.apple.quarantine "$@" 2>/dev/null || true; }

# ---- update ---------------------------------------------------------------------------
update_result() { # RESULT VERSION FROM DETAIL
    local d="${4//\\/\\\\}"
    install -d -m 0700 "$STATE"
    printf '{"result":"%s","version":"%s","from":"%s","at":"%s","detail":"%s"}\n' \
        "$1" "$2" "$3" "$(date -u +%Y-%m-%dT%H:%M:%SZ)" "${d//\"/\\\"}" >"$STATE/update-result.json.tmp" &&
        mv -f "$STATE/update-result.json.tmp" "$STATE/update-result.json"
}
fetch() { # URL FILE
    case "$1" in
        https://*) ;;
        http://*) case "$TB_API_URL" in http://*) ;; *) die "refusing $1: not https" ;; esac ;;
        *) die "bad URL $1" ;;
    esac
    curl -fsSL --retry 2 -o "$2" "$1"
}
# fetch_release: download and check the release bundle; sets BUNDLE
fetch_release() {
    local repo tag api rel name url sums want got tmp
    repo="${OPT_REPO:-$(sed -n 's/.*"repo": *"\([^"]*\)".*/\1/p' "$CONF" 2>/dev/null | head -n1)}"
    repo="${repo#https://github.com/}"; repo="${repo%.git}"; repo="${repo%/}"; repo="${repo:-kaovilai/tailbond}"
    tag="$OPT_UVERSION"; [ -z "$tag" ] || tag="v${tag#v}"
    api="$TB_API_URL/repos/$repo/releases/${tag:+tags/$tag}"; [ -n "$tag" ] || api="${api}latest"
    tmp="$(mktemp -d "${TMPDIR:-/tmp}/tailbond-update.XXXXXX")"
    trap 'rm -rf "$tmp"' EXIT
    fetch "$api" "$tmp/release.json" || die "can't read $api"
    rel="$(tr -d '\n' <"$tmp/release.json")"
    tag="$(printf '%s' "$rel" | sed -n 's/.*"tag_name": *"\([^"]*\)".*/\1/p')"
    [ -n "$tag" ] || die "no release found at $api"
    name="tailbond-macos-$(uname -m).tar.gz"
    asset_url() { printf '%s' "$rel" | grep -o "\"browser_download_url\": *\"[^\"]*/$1\"" | head -n1 | sed 's/.*: *"\(.*\)"/\1/'; }
    url="$(asset_url "$name")"; sums="$(asset_url SHA256SUMS)"
    [ -n "$url" ] || die "release $tag has no $name"
    [ -n "$sums" ] || die "release $tag has no SHA256SUMS: not installing anything unchecked"
    say "downloading $repo $tag ($name)"
    fetch "$sums" "$tmp/SHA256SUMS" || die "download of SHA256SUMS failed"
    fetch "$url" "$tmp/$name" || die "download of $name failed"
    want="$(awk -v n="$name" '$2 == n || $2 == "*" n { print $1 }' "$tmp/SHA256SUMS" | head -n1)"
    got="$(shasum -a 256 "$tmp/$name" | cut -d' ' -f1)"
    [ -n "$want" ] || die "SHA256SUMS has no line for $name"
    [ "$want" = "$got" ] || die "$name doesn't match SHA256SUMS (got $got): not installing it"
    tar -xzf "$tmp/$name" -C "$tmp" || die "can't unpack $name"
    BUNDLE="$tmp/tailbond-macos-$(uname -m)"
    [ -x "$BUNDLE/bin/tailbondd" ] || die "$name has no bin/tailbondd"
}
swap() { # SRC DST MODE: replace DST keeping DST.prev
    if [ -e "$2" ]; then cp -p "$2" "$2.prev"; fi
    install -m "$3" "$1" "$2.tbnew" && mv -f "$2.tbnew" "$2"
}
install_hooks() {
    rm -rf "$HOOKS.prev"
    if [ -d "$HOOKS" ]; then cp -a "$HOOKS" "$HOOKS.prev"; fi
    install -d -m 0755 "$HOOKS"
    local h
    for h in "$BUNDLE"/hooks/*; do install -m 0755 "$h" "$HOOKS/$(basename "$h")"; done
}
# the menu bar app (docs/macos.md "Menu bar"): into $APPDIR, ad-hoc signed
APPDIR="${TB_APP_DIR:-/Applications}"
install_app() {
    [ -d "$BUNDLE/tailbond.app" ] || return 0
    pkill -x tailbond 2>/dev/null || true
    rm -rf "$APPDIR/tailbond.app.tbnew"
    install -d -m 0755 "$APPDIR"
    ditto "$BUNDLE/tailbond.app" "$APPDIR/tailbond.app.tbnew"
    unquarantine "$APPDIR/tailbond.app.tbnew"
    rm -rf "$APPDIR/tailbond.app"
    mv "$APPDIR/tailbond.app.tbnew" "$APPDIR/tailbond.app"
    say "menu bar app: $APPDIR/tailbond.app"
}
# open_app: start it for the person at the screen (install.sh runs as root)
open_app() {
    [ -d "$APPDIR/tailbond.app" ] && [ -z "${TB_NO_LAUNCHD:-}" ] && [ -z "${TB_NO_APP_OPEN:-}" ] || return 0
    local u uid
    u="$(stat -f %Su /dev/console 2>/dev/null || true)"
    [ -n "$u" ] && [ "$u" != root ] || return 0
    uid="$(id -u "$u" 2>/dev/null)" || return 0
    launchctl asuser "$uid" sudo -u "$u" open "$APPDIR/tailbond.app" 2>/dev/null || true
}
do_update() {
    [ -f "$MANIFEST" ] || die "no tailbond install found ($MANIFEST): run install.sh without --update first"
    local cur new b ok=0 why
    cur="$(sed -n 's/^TB_VERSION=//p' "$MANIFEST")"
    if [ -n "$OPT_UVERSION" ] || [ ! -x "$BUNDLE/bin/tailbondd" ]; then fetch_release; fi
    unquarantine "$BUNDLE"
    new="$(cat "$BUNDLE/VERSION" 2>/dev/null || echo dev)"
    if [ "$new" = "$cur" ] && [ "$FORCE" = 0 ]; then say "tailbond $cur is installed already (--force reinstalls it)"; return 0; fi
    "$BUNDLE/bin/tailbondd" -version >/dev/null 2>&1 || die "$BUNDLE/bin/tailbondd doesn't run here: nothing changed"
    say "updating tailbond $cur -> $new"
    for b in tailbondd mqvpn; do swap "$BUNDLE/bin/$b" "$BINDIR/$b" 0755; done
    install_hooks
    install -m 0755 "$BUNDLE/install.sh" "$LIBDIR/install.sh"
    install -m 0755 "$BUNDLE/uninstall.sh" "$LIBDIR/uninstall.sh"
    if [ -d "$APPDIR/tailbond.app" ] || [ -d "$BUNDLE/tailbond.app" ]; then install_app; open_app; fi
    set_kv "$MANIFEST" TB_VERSION "$new"
    if launchd_on; then
        tb_restart
        for _ in $(seq 1 60); do
            if [ "$(hello_version)" = "$new" ]; then ok=1; break; fi
            sleep 1
        done
    else
        ok=1
    fi
    if [ "$ok" = 1 ]; then
        update_result ok "$new" "$cur" ""
        say "tailbond $new is running (the previous version is kept as $BINDIR/tailbondd.prev)"
        return 0
    fi
    why="tailbond $new didn't come up within 60 s"
    warn "$why: going back to $cur"
    # mqvpn first, each through a new file and a rename (as on Linux): never
    # write into a binary that may be running; an in-place overwrite of a
    # signed Mach-O can get the process using it killed
    for b in mqvpn tailbondd; do
        if [ -e "$BINDIR/$b.prev" ]; then
            install -m 0755 "$BINDIR/$b.prev" "$BINDIR/$b.tbnew" && mv -f "$BINDIR/$b.tbnew" "$BINDIR/$b"
        fi
    done
    if [ -d "$HOOKS.prev" ]; then rm -rf "$HOOKS"; cp -a "$HOOKS.prev" "$HOOKS"; fi
    set_kv "$MANIFEST" TB_VERSION "$cur"
    tb_restart
    update_result rolled_back "$new" "$cur" "$why; see $LOGDIR/tailbond.log"
    return 1
}

if [ "$UPDATE" = 1 ]; then do_update; exit $?; fi

# ---- install --------------------------------------------------------------------------
[ -x "$BUNDLE/bin/tailbondd" ] && [ -x "$BUNDLE/bin/mqvpn" ] || die "run this from the unpacked bundle (bin/tailbondd, bin/mqvpn)"
[ "$(uname -m)" = arm64 ] || die "this bundle is for Apple silicon (arm64); see docs/macos.md for Intel Macs"
unquarantine "$BUNDLE"
VERSION="$(cat "$BUNDLE/VERSION" 2>/dev/null || echo dev)"
"$BUNDLE/bin/tailbondd" -version >/dev/null || die "$BUNDLE/bin/tailbondd doesn't run on this Mac"

say "installing tailbond $VERSION"
tb_stop
install -d -m 0755 "$BINDIR" "$LIBDIR" "$LOGDIR" "$(dirname "$PLIST")"
install -d -m 0700 "$ETC" "$STATE"
for b in tailbondd mqvpn; do install -m 0755 "$BUNDLE/bin/$b" "$BINDIR/$b"; done
install_hooks
install -m 0755 "$BUNDLE/install.sh" "$LIBDIR/install.sh"
install -m 0755 "$BUNDLE/uninstall.sh" "$LIBDIR/uninstall.sh"
if [ ! -f "$CONF" ]; then
    "$BINDIR/tailbondd" -init -role client -config "$CONF" >/dev/null
    # the defaults are /usr/local paths
    if [ "$PREFIX" != /usr/local ]; then
        sed "s|/usr/local/|$PREFIX/|g" "$CONF" >"$CONF.tbnew" && mv -f "$CONF.tbnew" "$CONF"
    fi
    chmod 0600 "$CONF"
    say "wrote $CONF (client role)"
else
    say "keeping $CONF"
fi
sed -e "s|/usr/local/bin/tailbondd|$BINDIR/tailbondd|" -e "s|/usr/local/etc/tailbond/|$ETC/|" \
    -e "s|/usr/local/var/log/|$LOGDIR/|g" -e "s|/usr/local/bin:|$BINDIR:|" "$BUNDLE/$LABEL.plist" >"$PLIST.tbnew"
chmod 0644 "$PLIST.tbnew"
mv -f "$PLIST.tbnew" "$PLIST"
printf 'TB_PREFIX=%s\nTB_VERSION=%s\nTB_PLIST=%s\n' "$PREFIX" "$VERSION" "$PLIST" >"$MANIFEST"
tb_start
install_app
open_app

if launchd_on; then
    for _ in $(seq 1 20); do [ -n "$(hello_version)" ] && break; sleep 1; done
    [ -n "$(hello_version)" ] || warn "tailbond isn't answering on 127.0.0.1:8077 yet: see $LOGDIR/tailbond.log"
    if [ -n "$PAIR" ]; then
        if curl -fsS -m 10 -H 'Content-Type: application/json' -d "{\"uri\":\"$PAIR\"}" http://127.0.0.1:8077/api/pair >/dev/null; then
            say "paired: tailbond connects now"
        else
            warn "pairing failed: paste the link in the dashboard's Setup tab"
        fi
    fi
fi
say "tailbond $VERSION is installed. Dashboard: http://127.0.0.1:8077"
[ -n "$PAIR" ] || say "Pair it: open the dashboard, Setup tab, and paste the link from your home server's Devices tab."
say "Update later: sudo $LIBDIR/install.sh --update    Remove: sudo $LIBDIR/uninstall.sh"
