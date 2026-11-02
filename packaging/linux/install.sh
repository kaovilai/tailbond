#!/usr/bin/env bash
# SPDX-License-Identifier: Apache-2.0
#
# tailbond server installer for a generic Linux box (a VPS): AlmaLinux /
# RHEL / Rocky 9-10, Fedora, Debian 12, Ubuntu 22.04 / 24.04. Run it from the
# unpacked bundle (packaging/linux/build-bundle.sh), as root:
#
#   sudo ./install.sh                        # detect everything, install, start
#   sudo ./install.sh --dry-run              # print the decisions + files, change nothing
#   sudo ./install.sh --no-tailscale --public-addr vps.example.net
#   sudo ./install.sh --uninstall [--purge]
#
# What it decides (each printed, each overridable):
# - tunnel UDP port: 443 if free, else 8443, else the first free of 4443
#   10443 50443 (a web server with HTTP/3 holds udp/443); --port N insists;
# - phone page: port 80 of the Tailscale IPs, or 8078 when something listens
#   on *:80 (Caddy, nginx); Caddy's config is never touched;
# - tunnel subnet: a random free /24 in 10.78.0.0-10.99.255.0 that overlaps no
#   route or address here (and isn't 10.77.0.0/24, the home server's
#   default), plus a random fd77:xxxx::/112; kept on re-runs;
# - Tailscale: used when `tailscale status` works (real TLS cert, tailnet
#   access, subnet route); else (or --no-tailscale) self-signed + pinned
#   cert and internet exit only, paired with a link or QR code;
# - firewall: firewalld (zone + policies, like the Fedora RPM), else ufw
#   (persistent rules) + an nftables table for NAT, else nftables alone,
#   else iptables, else a warning;
# - systemd unit with MemoryHigh/MemoryMax, OOMScoreAdjust, hardening.
#
# Re-runs keep server.users, the subnet, the port, public_addr and the
# Tailscale choice of /etc/tailbond/tailbond.json, and apply only what is
# passed on the command line.
#
# Test knobs (CI, --dry-run): TB_TEST_LISTEN="udp:443 tcp:80" adds fake
# wildcard listeners, TB_TEST_ROUTES="10.89.0.0/24 ..." adds fake routes,
# TB_TEST_NO_TAILSCALE=1 pretends Tailscale isn't running.
set -euo pipefail

usage() {
    sed -n '/^# tailbond server installer/,/^# Test knobs/p' "$0" | sed 's/^# \{0,1\}//; $d'
    cat <<'EOF'
Options:
  --port N              tunnel UDP port (error if something else holds it)
  --subnet CIDR         IPv4 tunnel subnet (a /24 is plenty: 253 clients)
  --subnet6 CIDR        IPv6 tunnel subnet (/112)
  --public-addr HOST    address clients dial (default: this machine's
                        default-route source address)
  --no-tailscale        don't use Tailscale even if it runs here
  --tailscale           use Tailscale on a re-run that had it off (re-pair after)
  --landing-port N      phone page port on the Tailscale IPs (-1 = off)
  --update-policy P     auto (default: installs updates itself in the
                        maintenance window), notify (shows them, installs on
                        request) or off; see docs/updates.md
  --update-source URL   where to look for updates (default: built in;
                        "" goes back to it)
  --memory-high SIZE    systemd MemoryHigh= (default 96M)
  --memory-max SIZE     systemd MemoryMax= (default 160M)
  --prefix DIR          /usr/local (default) or /usr
  --config PATH         config file (default /etc/tailbond/tailbond.json)
  --bundle-dir DIR      unpacked bundle (default: this script's directory)
  --mqvpn PATH          mqvpn binary (default: <bundle>/bin/mqvpn)
  --tailbondd PATH      tailbondd binary (default: <bundle>/bin/tailbondd)
  --no-firewall         don't touch firewalld/ufw (the hooks still add NAT)
  --no-start            install, but don't enable/start the service
  --dry-run             print decisions, rendered config/unit, planned actions
  --uninstall           remove tailbond (keeps /etc/tailbond, /var/lib/tailbond)
  --purge               with --uninstall: also remove config and state
  -h, --help
EOF
}

# ---- options ----------------------------------------------------------------
OPT_PORT="" OPT_SUBNET="" OPT_SUBNET6="" OPT_PUBLIC="" OPT_TS="" OPT_LANDING=""
OPT_UPOLICY="" OPT_USOURCE="" OPT_USOURCE_SET=0
MEM_HIGH=96M MEM_MAX=160M GOMEMLIMIT=40MiB OOM_ADJ=-100
PREFIX=/usr/local CONF=/etc/tailbond/tailbond.json
BUNDLE="$(cd "$(dirname "${BASH_SOURCE[0]}")" && pwd)"
MQVPN_SRC="" TBD_SRC="" DRY=0 UNINSTALL=0 PURGE=0 NOFW=0 NOSTART=0

need() { [ $# -ge 2 ] || { echo "install.sh: $1 needs a value" >&2; exit 2; }; }
while [ $# -gt 0 ]; do
    a="$1"
    case "$a" in
        --*=*) set -- "${a%%=*}" "${a#*=}" "${@:2}"; continue ;;
    esac
    case "$a" in
        --port) need "$@"; OPT_PORT="$2"; shift ;;
        --subnet) need "$@"; OPT_SUBNET="$2"; shift ;;
        --subnet6) need "$@"; OPT_SUBNET6="$2"; shift ;;
        --public-addr) need "$@"; OPT_PUBLIC="$2"; shift ;;
        --no-tailscale) OPT_TS=off ;;
        --tailscale) OPT_TS=on ;;
        --landing-port) need "$@"; OPT_LANDING="$2"; shift ;;
        --update-policy) need "$@"; OPT_UPOLICY="$2"; shift ;;
        --update-source) need "$@"; OPT_USOURCE="$2"; OPT_USOURCE_SET=1; shift ;;
        --memory-high) need "$@"; MEM_HIGH="$2"; shift ;;
        --memory-max) need "$@"; MEM_MAX="$2"; shift ;;
        --prefix) need "$@"; PREFIX="${2%/}"; shift ;;
        --config) need "$@"; CONF="$2"; shift ;;
        --bundle-dir) need "$@"; BUNDLE="$(cd "$2" && pwd)"; shift ;;
        --mqvpn) need "$@"; MQVPN_SRC="$2"; shift ;;
        --tailbondd) need "$@"; TBD_SRC="$2"; shift ;;
        --no-firewall) NOFW=1 ;;
        --no-start) NOSTART=1 ;;
        --dry-run) DRY=1 ;;
        --uninstall) UNINSTALL=1 ;;
        --purge) PURGE=1 ;;
        -h|--help) usage; exit 0 ;;
        *) echo "install.sh: unknown option $a (see --help)" >&2; exit 2 ;;
    esac
    shift
done

case "$OPT_UPOLICY" in ""|auto|notify|off) ;; *) echo "install.sh: --update-policy $OPT_UPOLICY: auto, notify or off" >&2; exit 2 ;; esac
case "$OPT_USOURCE" in
    ""|http://*|https://*) ;;
    *) echo "install.sh: --update-source $OPT_USOURCE: an http(s):// URL" >&2; exit 2 ;;
esac
case "$OPT_USOURCE" in *[[:space:]\"\\]*) echo "install.sh: --update-source: no spaces or quotes" >&2; exit 2 ;; esac

BINDIR="$PREFIX/bin"
LIBDIR="$PREFIX/lib/tailbond"
HOOKS="$LIBDIR/hooks"
SHARE="$PREFIX/share/tailbond"
DOCDIR="$PREFIX/share/doc/tailbond"
ETC="$(dirname "$CONF")"
STATE=/var/lib/tailbond
UNIT=/etc/systemd/system/tailbond.service
SYSCTL=/etc/sysctl.d/90-tailbond.conf
MANIFEST=/etc/tailbond/install.env
TAILNET4=100.64.0.0/10

say()  { printf '%s\n' "$*"; }
note() { printf '  - %s\n' "$*"; }
warn() { printf 'WARNING: %s\n' "$*" >&2; }
die()  { printf 'install.sh: %s\n' "$*" >&2; exit 1; }
# act CMD...: run it, or print it under --dry-run.
act() {
    if [ "$DRY" = 1 ]; then printf '  + %s\n' "$*"; else "$@"; fi
}
have() { command -v "$1" >/dev/null 2>&1; }

# ---- the machine --------------------------------------------------------------
OS_ID=unknown OS_LIKE="" OS_NAME="Linux"
if [ -r /etc/os-release ]; then
    # shellcheck disable=SC1091
    . /etc/os-release
    OS_ID="${ID:-unknown}" OS_LIKE="${ID_LIKE:-}" OS_NAME="${PRETTY_NAME:-$OS_ID}"
fi
case " $OS_ID $OS_LIKE " in
    *" fedora "*|*" rhel "*|*" centos "*) FAMILY=rhel; PKG_HINT="dnf install" ;;
    *" debian "*|*" ubuntu "*) FAMILY=debian; PKG_HINT="apt-get install" ;;
    *) FAMILY=other; PKG_HINT="install" ;;
esac

selinux_on() { have selinuxenabled && selinuxenabled; }
systemd_on() { [ -d /run/systemd/system ] && have systemctl; }

# ---- uninstall ------------------------------------------------------------------
fw_running() { have firewall-cmd && firewall-cmd --state >/dev/null 2>&1; }
ufw_active() { have ufw && ufw status 2>/dev/null | grep -q '^Status: active'; }

do_uninstall() {
    [ "$DRY" = 1 ] || [ "$(id -u)" = 0 ] || die "run as root (sudo)"
    local port=""
    if [ -r "$MANIFEST" ]; then
        # shellcheck disable=SC1090
        . "$MANIFEST"
        PREFIX="${TB_PREFIX:-$PREFIX}" port="${TB_PORT:-}"
        BINDIR="$PREFIX/bin" LIBDIR="$PREFIX/lib/tailbond" SHARE="$PREFIX/share/tailbond" DOCDIR="$PREFIX/share/doc/tailbond"
    fi
    say "Removing tailbond (prefix $PREFIX)"
    if systemd_on && systemctl list-unit-files tailbond.service >/dev/null 2>&1; then
        act systemctl disable --now tailbond.service || true
    fi
    # The server-down hook ran on stop; clear what it may have left.
    if have nft && nft list table inet tailbond >/dev/null 2>&1; then act nft delete table inet tailbond; fi
    if [ "$NOFW" = 0 ] && fw_running; then
        local z tz
        for z in $(firewall-cmd --permanent --get-zones 2>/dev/null); do
            act firewall-cmd -q --permanent --zone="$z" --remove-service=tailbond 2>/dev/null || true
            if [ -n "$port" ]; then act firewall-cmd -q --permanent --zone="$z" --remove-port="$port/udp" 2>/dev/null || true; fi
        done
        tz="$(firewall-cmd --permanent --get-zone-of-interface=tailscale0 2>/dev/null)" || tz="$(firewall-cmd --get-default-zone)"
        firewall-cmd --permanent --zone="$tz" --list-rich-rules 2>/dev/null |
            grep -E 'source address="(100\.64\.0\.0/10|fd7a:115c:a1e0::/48)" port port="[0-9]+" protocol="tcp" accept' |
            grep -E "port=\"(8077|80|${TB_LANDING:-80})\"" |
            while read -r r; do act firewall-cmd -q --permanent --zone="$tz" --remove-rich-rule="$r" || true; done
    fi
    local x
    for x in services/tailbond zones/tailbond policies/tailbond-out policies/tailbond-in; do
        act rm -f "/etc/firewalld/$x.xml" "/etc/firewalld/$x.xml.old"
    done
    if [ "$NOFW" = 0 ] && fw_running; then act firewall-cmd -q --reload || true; fi
    if [ "$NOFW" = 0 ] && ufw_active; then
        local n
        for n in $(ufw status numbered | sed -n 's/^\[ *\([0-9]*\)\].*# tailbond$/\1/p' | sort -rn); do
            act ufw --force delete "$n"
        done
    fi
    act rm -f "$UNIT" "$SYSCTL" "$BINDIR/tailbondd" "$BINDIR/mqvpn"
    act rm -rf "$LIBDIR" "$SHARE" "$DOCDIR"
    if systemd_on; then act systemctl daemon-reload; fi
    if [ "$PURGE" = 1 ]; then
        act rm -rf /etc/tailbond "$STATE"
    else
        act rm -f "$MANIFEST"
        say "Kept /etc/tailbond (config, registered devices) and $STATE; --purge removes them."
    fi
    say "Done. IP forwarding stays on until reboot (sysctl net.ipv4.ip_forward=0 turns it off now)."
    if have tailscale && tailscale status >/dev/null 2>&1; then
        say "The tunnel route may still be advertised in Tailscale: tailscale set --advertise-routes=<your other routes> (empty withdraws all)."
    fi
}

if [ "$UNINSTALL" = 1 ]; then do_uninstall; exit 0; fi

# ---- preflight --------------------------------------------------------------------
[ "$DRY" = 1 ] || [ "$(id -u)" = 0 ] || die "run as root (sudo), or try --dry-run"
case "$PREFIX" in /usr|/usr/local) ;; *) warn "prefix $PREFIX: SELinux labels and PATH assume /usr or /usr/local" ;; esac
for c in ip ss; do
    have "$c" || die "'$c' not found: $PKG_HINT $([ "$FAMILY" = debian ] && echo iproute2 || echo iproute)"
done
if have rpm && rpm -q tailbond-server >/dev/null 2>&1; then
    die "the tailbond-server RPM is installed (docs/fedora-server.md); remove it first or keep using it"
fi
[ "$FAMILY" = other ] && warn "untested distribution ($OS_NAME); continuing"

MQVPN_SRC="${MQVPN_SRC:-$BUNDLE/bin/mqvpn}"
TBD_SRC="${TBD_SRC:-$BUNDLE/bin/tailbondd}"
VERSION="$(cat "$BUNDLE/VERSION" 2>/dev/null || echo dev)"
check_bins() {
    local ok=1 h b
    for b in "$MQVPN_SRC" "$TBD_SRC"; do
        [ -x "$b" ] && continue
        ok=0
        if [ "$DRY" = 1 ]; then warn "$b missing"; else die "$b missing (use --bundle-dir, --mqvpn, --tailbondd)"; fi
    done
    [ "$ok" = 1 ] || return 0
    # --help exits non-zero; check that it printed the usage text.
    h="$("$MQVPN_SRC" --help 2>&1 || true)"
    case "$h" in *Usage:*) ;; *) die "$MQVPN_SRC does not run here ($(uname -m)): ${h:0:200}" ;; esac
    "$TBD_SRC" -version >/dev/null 2>&1 || die "$TBD_SRC does not run here ($(uname -m))"
}
check_bins

# ---- the existing config ----------------------------------------------------------
# json_get FILE KEY.PATH: a string/number/bool, "" when missing.
json_get() {
    local f="$1" k="$2"
    [ -r "$f" ] || return 0
    if have jq; then
        jq -r --arg k "$k" 'getpath($k | split(".")) // empty | tostring' "$f" 2>/dev/null
    elif have python3; then
        python3 -c 'import json,sys
v=json.load(open(sys.argv[1]))
for p in sys.argv[2].split("."):
    v=v.get(p) if isinstance(v,dict) else None
print("" if v is None else (str(v).lower() if isinstance(v,bool) else v))' "$f" "$k" 2>/dev/null
    else
        # tailbondd writes indented JSON, one key per line; leaf names below
        # are unique in a server config.
        sed -n "s/^ *\"${k##*.}\": *\"\{0,1\}\([^\",]*\)\"\{0,1\},\{0,1\} *$/\1/p" "$f" | head -1
    fi
}
json_users() { # number of server.users
    local f="$1"
    [ -r "$f" ] || { echo 0; return; }
    if have jq; then jq '.server.users // [] | length' "$f" 2>/dev/null || echo 0
    elif have python3; then python3 -c 'import json,sys; print(len((json.load(open(sys.argv[1])).get("server") or {}).get("users") or []))' "$f" 2>/dev/null || echo 0
    else grep -c '"key":' "$f" || true
    fi
}
json_has() { # FILE TOPKEY: present at the top level
    [ -r "$1" ] && grep -q "^ *\"$2\":" "$1"
}

EXISTS=0
if [ -e "$CONF" ]; then
    [ -r "$CONF" ] || die "$CONF exists but isn't readable (run as root)"
    EXISTS=1
fi
E_PORT="$(json_get "$CONF" server.port)"
E_SUBNET="$(json_get "$CONF" server.subnet)"
E_SUBNET6="$(json_get "$CONF" server.subnet6)"
E_PUBLIC="$(json_get "$CONF" server.public_addr)"
E_NOTS="$(json_get "$CONF" disable_tailscale)"
E_LANDING="$(json_get "$CONF" landing_port)"
E_USERS="$(json_users "$CONF")"

# ---- addresses and listeners ---------------------------------------------------------
ip2int() { local IFS=.; read -r a b c d <<<"$1"; echo $(( (a << 24) | (b << 16) | (c << 8) | d )); }
# in_prefix ADDR CIDR (IPv4)
in_prefix() {
    local n="${2%/*}" l="${2#*/}"
    [ "$l" = "$2" ] && l=32
    local m=$(( l == 0 ? 0 : (0xFFFFFFFF << (32 - l)) & 0xFFFFFFFF ))
    [ $(( $(ip2int "$1") & m )) -eq $(( $(ip2int "$n") & m )) ]
}
# overlaps CIDR CIDR (IPv4)
overlaps() {
    local a="${1%/*}" la="${1#*/}" b="${2%/*}" lb="${2#*/}"
    [ "$la" = "$1" ] && la=32
    [ "$lb" = "$2" ] && lb=32
    local l=$(( la < lb ? la : lb ))
    local m=$(( l == 0 ? 0 : (0xFFFFFFFF << (32 - l)) & 0xFFFFFFFF ))
    [ $(( $(ip2int "$a") & m )) -eq $(( $(ip2int "$b") & m )) ]
}
is_v4() { [[ "$1" =~ ^[0-9]{1,3}(\.[0-9]{1,3}){3}$ ]]; }
is_cidr4() { [[ "$1" =~ ^[0-9]{1,3}(\.[0-9]{1,3}){3}/([0-9]|[12][0-9]|3[0-2])$ ]]; }
is_private4() {
    local p
    for p in 10.0.0.0/8 172.16.0.0/12 192.168.0.0/16 100.64.0.0/10 169.254.0.0/16 127.0.0.0/8; do
        in_prefix "$1" "$p" && return 0
    done
    return 1
}

# listeners PROTO(u|t): "addr port process" per listening socket.
listeners() {
    { ss -Hln"$1"p 2>/dev/null || ss -Hln"$1" 2>/dev/null || true; } | awk '{
        la = $4; port = la; sub(/.*:/, "", port); addr = la; sub(/:[^:]*$/, "", addr)
        gsub(/[][]/, "", addr); sub(/%.*/, "", addr)
        proc = $6; if (proc == "") proc = "-"
        print addr, port, proc }'
    local f p
    for f in ${TB_TEST_LISTEN:-}; do
        p="${f%%:*}"
        [ "${p:0:1}" = "$1" ] && echo "* ${f#*:} users:((\"test\"))"
    done
    true
}
LIST_U="$(listeners u)"
LIST_T="$(listeners t)"
# Ours: mqvpn/tailbondd, or (not root, so ss can't name it) any listener
# while tailbond.service runs.
TB_ACTIVE=0
if systemd_on && systemctl is-active --quiet tailbond.service 2>/dev/null; then TB_ACTIVE=1; fi
is_ours() {
    case "$1" in *'"mqvpn"'*|*'"tailbondd"'*) return 0 ;; -) [ "$TB_ACTIVE" = 1 ] && return 0 ;; esac
    return 1
}
is_wild() { case "$1" in '*'|0.0.0.0|::) return 0 ;; esac; return 1; }
is_ts_addr() {
    case "$1" in fd7a:115c:a1e0:*) return 0 ;; esac
    is_v4 "$1" && in_prefix "$1" "$TAILNET4"
}
# holders PROTO PORT FILTER: foreign listeners on PORT, "addr (process)" each.
# FILTER any = every address; ts = wildcard or Tailscale address; dash =
# wildcard, loopback or Tailscale address (where the dashboard binds).
holders() {
    local list addr port proc
    if [ "$1" = u ]; then list="$LIST_U"; else list="$LIST_T"; fi
    while read -r addr port proc; do
        [ "$port" = "$2" ] || continue
        is_ours "$proc" && continue
        case "$3" in
            ts) is_wild "$addr" || is_ts_addr "$addr" || continue ;;
            dash) is_wild "$addr" || is_ts_addr "$addr" || [ "$addr" = 127.0.0.1 ] || [ "$addr" = ::1 ] || continue ;;
            wild) is_wild "$addr" || continue ;;
        esac
        procname="$(sed -n 's/^users:(("\([^"]*\)".*/\1/p' <<<"$proc")"
        printf '%s:%s (%s)\n' "$addr" "$port" "${procname:-unknown process}"
    done <<<"$list"
}
first_holder() { holders "$@" | head -1; }

# ---- Tailscale -------------------------------------------------------------------------
TS_UP=0 TS_NAME="" TS_IP4=""
if [ -z "${TB_TEST_NO_TAILSCALE:-}" ] && have tailscale && tailscale status >/dev/null 2>&1; then
    TS_UP=1
    TS_IP4="$(tailscale ip -4 2>/dev/null | head -1 || true)"
    TS_NAME="$(tailscale status --json 2>/dev/null | sed -n 's/^ *"DNSName": *"\([^"]*\)",*$/\1/p' | head -1 || true)"
    TS_NAME="${TS_NAME%.}"
fi
if [ "$OPT_TS" = off ]; then
    USE_TS=0; TS_WHY="--no-tailscale"
elif [ "$OPT_TS" = on ]; then
    [ "$TS_UP" = 1 ] || die "--tailscale: tailscale isn't running here (tailscale up first)"
    USE_TS=1; TS_WHY="--tailscale"
elif [ "$EXISTS" = 1 ] && [ "$E_NOTS" = true ]; then
    USE_TS=0; TS_WHY="kept from $CONF (pass --tailscale to switch; devices then pair again)"
elif [ "$TS_UP" = 1 ]; then
    USE_TS=1; TS_WHY="tailscale status works${TS_NAME:+ ($TS_NAME)}"
else
    USE_TS=0; TS_WHY="$(have tailscale && echo "tailscale isn't up" || echo "tailscale isn't installed")"
fi

# ---- tunnel port ----------------------------------------------------------------------------
valid_port() { [[ "$1" =~ ^[0-9]+$ ]] && [ "$1" -ge 1 ] && [ "$1" -le 65535 ]; }
if [ -n "$OPT_PORT" ]; then
    valid_port "$OPT_PORT" || die "--port $OPT_PORT: not a port"
    h="$(first_holder u "$OPT_PORT" any)"
    [ -z "$h" ] || die "--port $OPT_PORT: udp/$OPT_PORT is in use by $h"
    PORT="$OPT_PORT"; PORT_WHY="--port"
elif [ -n "$E_PORT" ] && [ "$EXISTS" = 1 ]; then
    PORT="$E_PORT"; PORT_WHY="kept from $CONF (paired devices use it)"
    h="$(first_holder u "$PORT" any)"
    [ -z "$h" ] || warn "udp/$PORT is now used by $h; mqvpn can't start until it's free (or re-run with --port N and pair devices again)"
else
    PORT="" PORT_WHY=""
    for p in 443 8443 4443 10443 50443; do
        h="$(first_holder u "$p" any)"
        if [ -z "$h" ]; then PORT="$p"; break; fi
        PORT_WHY="${PORT_WHY:+$PORT_WHY, }udp/$p is used by $h"
    done
    [ -n "$PORT" ] || die "no free UDP port among 443 8443 4443 10443 50443; pass --port N"
    PORT_WHY="${PORT_WHY:-free}"
fi

# ---- dashboard, DNS, phone page ----------------------------------------------------------------
h="$(first_holder t 8077 dash)"
[ -z "$h" ] || die "tcp/8077 (the tailbond dashboard and API, also how routers find this server over Tailscale) is used by $h; free it first"

DISABLE_DNS=false DNS_WHY="the forwarder binds only the tunnel address:53"
h="$(first_holder u 53 wild)"
[ -z "$h" ] && h="$(first_holder t 53 wild)"
if [ -n "$h" ]; then
    DISABLE_DNS=true
    DNS_WHY="$h listens on every address, so the forwarder could not bind the tunnel address; clients keep their own DNS (no MagicDNS names)"
fi

LANDING_PUBLIC=""
if [ "$USE_TS" = 0 ]; then
    LANDING=-1; LANDING_WHY="off (it's for phones on your tailnet)"
elif [ -n "$OPT_LANDING" ]; then
    [[ "$OPT_LANDING" =~ ^-?[0-9]+$ ]] || die "--landing-port $OPT_LANDING: not a port"
    LANDING="$OPT_LANDING"; LANDING_WHY="--landing-port"
elif [ "$EXISTS" = 1 ] && json_has "$CONF" landing_port; then
    LANDING="$E_LANDING"; LANDING_WHY="kept from $CONF"
    LANDING_PUBLIC="$(json_get "$CONF" landing_public_port)"
else
    h="$(first_holder t 80 ts)"
    if [ -z "$h" ]; then
        LANDING=80; LANDING_WHY="tcp/80 is free on the Tailscale address"
    else
        LANDING=""
        for p in 8078 8079 8088; do
            [ -z "$(first_holder t "$p" ts)" ] && { LANDING="$p"; break; }
        done
        [ -n "$LANDING" ] || LANDING=-1
        LANDING_WHY="tcp/80 is used by $h"
        CADDY_HINT=1
    fi
fi
[ -n "$LANDING_PUBLIC" ] || LANDING_PUBLIC="$LANDING"
[ "$LANDING" = -1 ] && LANDING_PUBLIC=0 # the agent rejects a negative public port

# ---- tunnel subnet -------------------------------------------------------------------------------
# used_prefixes: every IPv4 route (all tables, so routes accepted from the
# tailnet count) and address, except tb0's own.
used_prefixes() {
    ip -4 -o route show table all 2>/dev/null | awk '$0 ~ / dev tb0( |$)/ { next } $1 == "default" || $2 == "default" { next } {
        for (i = 1; i <= NF; i++) if ($i ~ /^[0-9]+\.[0-9]+\.[0-9]+\.[0-9]+(\/[0-9]+)?$/) { print $i; break } }'
    ip -4 -o addr show 2>/dev/null | awk '$2 == "tb0" { next } { for (i = 1; i < NF; i++) if ($i == "inet") print $(i + 1) }'
    local r
    for r in ${TB_TEST_ROUTES:-}; do echo "$r"; done
    echo 10.77.0.0/24 # the home server's default subnet
}
USED="$(used_prefixes | sort -u)"
subnet_clash() { # CIDR -> the first used prefix it overlaps
    local u
    while read -r u; do
        [ -n "$u" ] || continue
        if overlaps "$1" "$u"; then echo "$u"; return 0; fi
    done <<<"$USED"
    return 1
}
pick_subnet() {
    local o2 o3 s
    for _ in $(seq 1 64); do
        o2=$(( 78 + RANDOM % 22 )) o3=$(( RANDOM % 256 ))
        s="10.$o2.$o3.0/24"
        subnet_clash "$s" >/dev/null || { echo "$s"; return 0; }
    done
    for o2 in $(seq 78 99); do for o3 in $(seq 0 255); do
        s="10.$o2.$o3.0/24"
        subnet_clash "$s" >/dev/null || { echo "$s"; return 0; }
    done; done
    return 1
}
if [ -n "$OPT_SUBNET" ]; then
    is_cidr4 "$OPT_SUBNET" || die "--subnet $OPT_SUBNET: not an IPv4 CIDR"
    SUBNET="$OPT_SUBNET"; SUBNET_WHY="--subnet"
    c="$(subnet_clash "$SUBNET" || true)"
    [ -z "$c" ] || warn "--subnet $SUBNET overlaps $c (a route or address here)"
elif [ -n "$E_SUBNET" ] && ! { [ "$E_SUBNET" = 10.77.0.0/24 ] && [ "$E_USERS" = 0 ]; }; then
    SUBNET="$E_SUBNET"; SUBNET_WHY="kept from $CONF"
else
    SUBNET="$(pick_subnet)" || die "no free /24 in 10.78.0.0-10.99.255.0; pass --subnet"
    SUBNET_WHY="random free /24 (avoids $(echo "$USED" | grep -c .) routes/addresses here, and 10.77.0.0/24)"
fi
if [ -n "$OPT_SUBNET6" ]; then
    SUBNET6="$OPT_SUBNET6"; SUBNET6_WHY="--subnet6"
elif [ -n "$E_SUBNET6" ] && ! { [ "$E_SUBNET6" = fd77:77::/112 ] && [ "$E_USERS" = 0 ]; }; then
    SUBNET6="$E_SUBNET6"; SUBNET6_WHY="kept"
else
    while :; do r=$(( (RANDOM << 1 ^ RANDOM) & 0xffff )); [ "$r" -gt 0 ] && [ "$r" != $((0x77)) ] && break; done
    SUBNET6="$(printf 'fd77:%x::/112' "$r")"; SUBNET6_WHY="random"
fi
TUN_IP="$(awk -F. '{print $1"."$2"."$3"."($4+1)}' <<<"${SUBNET%/*}")"

# ---- public address ---------------------------------------------------------------------------------
detect_public() {
    local s
    s="$(ip -4 route get 1.1.1.1 2>/dev/null | sed -n 's/.* src \([0-9.]*\).*/\1/p' | head -1)"
    if [ -n "$s" ]; then echo "$s"; return; fi
    s="$(ip -6 route get 2606:4700:4700::1111 2>/dev/null | sed -n 's/.* src \([0-9a-f:]*\).*/\1/p' | head -1)"
    echo "$s"
}
if [ -n "$OPT_PUBLIC" ]; then
    case "$OPT_PUBLIC" in *[[:space:]/\\@?#]*) die "--public-addr: host or host:port only" ;; esac
    PUBLIC="$OPT_PUBLIC"; PUBLIC_WHY="--public-addr"
elif [ -n "$E_PUBLIC" ]; then
    PUBLIC="$E_PUBLIC"; PUBLIC_WHY="kept from $CONF"
else
    PUBLIC="$(detect_public)"; PUBLIC_WHY="default-route source address"
    if [ -z "$PUBLIC" ]; then
        [ "$USE_TS" = 1 ] || die "no default route to guess the public address from; pass --public-addr"
        PUBLIC_WHY="none found; set it in the dashboard"
    elif is_v4 "$PUBLIC" && is_private4 "$PUBLIC"; then
        warn "$PUBLIC is a private address: this VPS is behind NAT (common on AWS, GCP, Oracle). Re-run with --public-addr <its public IP or DNS name> and forward udp/$PORT to it in the provider's firewall."
        PUBLIC_WHY="default-route source address, PRIVATE: clients away from this network can't reach it"
    fi
fi

# ---- firewall ------------------------------------------------------------------------------------------
if [ "$NOFW" = 1 ]; then FW=skip
elif fw_running; then FW=firewalld
elif ufw_active; then FW=ufw
elif have nft; then FW=nftables
elif have iptables; then FW=iptables
else FW=none
fi

# ---- render ---------------------------------------------------------------------------------------------
APK_SRC=""
[ -f "$BUNDLE/share/tailbond.apk" ] && APK_SRC="$BUNDLE/share/tailbond.apk"
TS_CERT=$([ "$USE_TS" = 1 ] && echo true || echo false)
NO_TS=$([ "$USE_TS" = 1 ] && echo false || echo true)
jstr() { local s="${1//\\/\\\\}"; printf '"%s"' "${s//\"/\\\"}"; }

# update: only what was passed (--update-policy/--update-source); a re-run
# without them keeps the existing block. Nothing else in it is touched
# (window, channel, pin; the dashboard sets those too).
render_update() {
    local u=""
    [ -n "$OPT_UPOLICY" ] && u="\"policy\": $(jstr "$OPT_UPOLICY")"
    [ "$OPT_USOURCE_SET" = 1 ] && u="${u:+$u, }\"source\": $(jstr "$OPT_USOURCE")"
    [ -n "$u" ] && printf '\n  "update": {%s},' "$u"
    return 0
}
# The keys this installer owns. Everything else in an existing config
# (users, approvers, ui_password, mode, ...) is left alone.
render_patch() {
    cat <<EOF
{
  "role": "server",
  "mqvpn_bin": $(jstr "$BINDIR/mqvpn"),
  "state_dir": "$STATE",
  "hooks_dir": $(jstr "$HOOKS"),
  "disable_tailscale": $NO_TS,
  "landing_port": $LANDING,
  "landing_public_port": $LANDING_PUBLIC,$([ -n "$APK_SRC" ] && printf '\n  "android_apk": %s,' "$(jstr "$SHARE/tailbond.apk")")$(render_update)
  "server": {
    "port": $PORT,
    "public_addr": $(jstr "$PUBLIC"),
    "subnet": "$SUBNET",
    "subnet6": "$SUBNET6",
    "use_tailscale_cert": $TS_CERT,
    "disable_dns": $DISABLE_DNS
  }
}
EOF
}
render_full() {
    cat <<EOF
{
  "role": "server",
  "mqvpn_bin": $(jstr "$BINDIR/mqvpn"),
  "state_dir": "$STATE",
  "hooks_dir": $(jstr "$HOOKS"),
  "listen": ":8077",
  "ui_password": "",
  "mode": "speed",
  "disable_tailscale": $NO_TS,
  "landing_port": $LANDING,
  "landing_public_port": $LANDING_PUBLIC,$([ -n "$APK_SRC" ] && printf '\n  "android_apk": %s,' "$(jstr "$SHARE/tailbond.apk")")$(render_update)
  "server": {
    "port": $PORT,
    "public_addr": $(jstr "$PUBLIC"),
    "subnet": "$SUBNET",
    "subnet6": "$SUBNET6",
    "use_tailscale_cert": $TS_CERT,
    "disable_dns": $DISABLE_DNS
  },
  "client": {"pairing_uri": "", "paths": []}
}
EOF
}
# merged_config: the existing config with the patch applied (deep merge), or
# the full default when there is none. Prints nothing (rc 3) when a merge is
# needed but neither jq nor python3 is there.
merged_config() {
    if [ "$EXISTS" = 0 ]; then render_full; return; fi
    if have jq; then
        jq --argjson p "$(render_patch)" '. * $p' "$CONF"
    elif have python3; then
        python3 -c 'import json,sys
def merge(a,b):
    for k,v in b.items():
        a[k]=merge(a.get(k) or {},v) if isinstance(v,dict) else v
    return a
print(json.dumps(merge(json.load(open(sys.argv[1])),json.loads(sys.argv[2])),indent=2))' "$CONF" "$(render_patch)"
    else
        return 3
    fi
}

TPL="$BUNDLE/tailbond.service.in"
[ -r "$TPL" ] || die "$TPL missing (incomplete bundle)"
render_unit() {
    sed -e "s|@BINDIR@|$BINDIR|g" -e "s|@DOCDIR@|$DOCDIR|g" \
        -e "s|@MEMORY_HIGH@|$MEM_HIGH|g" -e "s|@MEMORY_MAX@|$MEM_MAX|g" \
        -e "s|@GOMEMLIMIT@|$GOMEMLIMIT|g" -e "s|@CONF@|$CONF|g" -e "s|@OOM_SCORE_ADJUST@|$OOM_ADJ|g" "$TPL"
}
for v in "$MEM_HIGH" "$MEM_MAX"; do
    [[ "$v" =~ ^([0-9]+[KMGT]?|infinity)$ ]] || die "memory limit $v: use e.g. 96M, 1G or infinity"
done

# IPv6 forwarding makes the kernel ignore router advertisements unless
# accept_ra=2; keep SLAAC-configured VPSes (Linode, Vultr) online.
WAN6="$(ip -6 route show default 2>/dev/null | awk '{for (i = 1; i < NF; i++) if ($i == "dev") { print $(i + 1); exit }}')"
render_sysctl() {
    echo "# tailbond: bonded clients' traffic is forwarded (and NATed) by this machine."
    echo "net.ipv4.ip_forward = 1"
    echo "net.ipv6.conf.all.forwarding = 1"
    if [ -n "$WAN6" ] && [ "$(cat "/proc/sys/net/ipv6/conf/$WAN6/accept_ra" 2>/dev/null)" = 1 ]; then
        echo "# $WAN6 gets its IPv6 route from router advertisements: keep accepting them."
        echo "net.ipv6.conf.$WAN6.accept_ra = 2"
    fi
}

# ---- summary -------------------------------------------------------------------------------------------------
HOST="$(hostname 2>/dev/null || echo this-machine)"
SHORT="${TS_NAME%%.*}"; SHORT="${SHORT:-$HOST}"
say "tailbond $VERSION server install on $OS_NAME ($(uname -m))$([ "$DRY" = 1 ] && echo ', DRY RUN: nothing is changed')"
note "tunnel port: udp/$PORT  [$PORT_WHY]"
note "public address: ${PUBLIC:-<none>}  [$PUBLIC_WHY]  -> pairing links carry ${PUBLIC:+$PUBLIC:}$PORT"
note "tunnel subnet: $SUBNET (server $TUN_IP)  [$SUBNET_WHY]; IPv6 $SUBNET6  [$SUBNET6_WHY]"
note "Tailscale: $([ "$USE_TS" = 1 ] && echo on || echo off)  [$TS_WHY]"
note "dashboard: http://127.0.0.1:8077/$([ "$USE_TS" = 1 ] && echo " and http://$SHORT:8077/ from your tailnet")"
if [ "$LANDING" = -1 ]; then note "phone page: off  [$LANDING_WHY]"
else note "phone page: http://$SHORT$([ "$LANDING_PUBLIC" = 80 ] || echo ":$LANDING_PUBLIC")/ on the Tailscale address  [$LANDING_WHY]"; fi
note "DNS for clients: $([ "$DISABLE_DNS" = true ] && echo off || echo "$TUN_IP:53")  [$DNS_WHY]"
note "firewall: $FW"
E_UPOLICY="$(json_get "$CONF" update.policy)"
if [ -n "$OPT_UPOLICY" ]; then UP_WHY="--update-policy"
elif [ -n "$E_UPOLICY" ]; then UP_WHY="kept from $CONF"
else UP_WHY="default"; fi
UP_SRC=""
[ "$OPT_USOURCE_SET" = 1 ] && UP_SRC=", from ${OPT_USOURCE:-the built-in source}"
note "updates: ${OPT_UPOLICY:-${E_UPOLICY:-auto}}  [$UP_WHY]$UP_SRC; installed updates live in $STATE/update"
note "binaries: $BINDIR/{mqvpn,tailbondd}; hooks: $HOOKS; config: $CONF; state: $STATE"
note "systemd: $UNIT (MemoryHigh=$MEM_HIGH MemoryMax=$MEM_MAX OOMScoreAdjust=$OOM_ADJ GOMEMLIMIT=$GOMEMLIMIT)"
note "SELinux: $(selinux_on && echo "$(getenforce 2>/dev/null || echo on), contexts restored with restorecon" || echo "not enabled")"
[ "$EXISTS" = 1 ] && note "existing config: $E_USERS registered device(s) kept"
if [ "${CADDY_HINT:-0}" = 1 ] && [ "$LANDING" != -1 ]; then
    cat <<EOF
  Optional: let phones type just "$SHORT/" by having your web server pass the
  tailnet name to the phone page (tailbond never edits it). Caddy:
      http://$SHORT${TS_NAME:+, http://$TS_NAME} {
          reverse_proxy ${TS_IP4:-<tailscale ip>}:$LANDING
      }
  then set "landing_public_port": 80 in $CONF and restart tailbond.
EOF
fi

if [ "$DRY" = 1 ]; then
    say ""
    say "---- $CONF ($([ "$EXISTS" = 1 ] && echo "merged into the existing file" || echo new), mode 0600)"
    if ! merged_config; then say "(existing file kept as is: install jq or python3 to merge the settings above)"; fi
    say "---- $UNIT"
    render_unit
    say "---- $SYSCTL"
    render_sysctl
    say "---- actions"
fi

# ---- install --------------------------------------------------------------------------------------------------------
# put SRC DST MODE: atomic replace (a running binary is fine).
put() {
    if [ "$DRY" = 1 ]; then printf '  + install -m %s %s %s\n' "$3" "$1" "$2"; return; fi
    install -d -m 0755 "$(dirname "$2")"
    install -m "$3" "$1" "$2.tbnew"
    mv -f "$2.tbnew" "$2"
}
# write DST MODE < content
write() {
    if [ "$DRY" = 1 ]; then printf '  + write %s (mode %s)\n' "$1" "$2"; cat >/dev/null; return; fi
    local t
    install -d -m 0755 "$(dirname "$1")"
    t="$(mktemp "$1.XXXXXX")"
    cat >"$t"
    chmod "$2" "$t"
    mv -f "$t" "$1"
}

WAS_ACTIVE=0
if systemd_on && systemctl is-active --quiet tailbond.service 2>/dev/null; then
    WAS_ACTIVE=1
    act systemctl stop tailbond.service
fi

put "$MQVPN_SRC" "$BINDIR/mqvpn" 0755
put "$TBD_SRC" "$BINDIR/tailbondd" 0755
for hk in server-up server-down; do put "$BUNDLE/hooks/$hk" "$HOOKS/$hk" 0755; done
[ -n "$APK_SRC" ] && put "$APK_SRC" "$SHARE/tailbond.apk" 0644
[ -f "$BUNDLE/README.md" ] && put "$BUNDLE/README.md" "$DOCDIR/README.md" 0644
for f in LICENSE.mqvpn NOTICE.mqvpn; do [ -f "$BUNDLE/$f" ] && put "$BUNDLE/$f" "$DOCDIR/$f" 0644; done

act install -d -m 0755 "$ETC"
act install -d -m 0700 "$STATE"
if [ "$DRY" = 0 ]; then
    umask 077
    if cfg="$(merged_config)"; then
        [ "$EXISTS" = 1 ] && cp -p "$CONF" "$CONF.bak"
        printf '%s\n' "$cfg" | write "$CONF" 0600
    else
        warn "kept $CONF unchanged: install jq or python3 so re-runs can merge settings into it"
    fi
    umask 022
    printf 'TB_PREFIX=%s\nTB_PORT=%s\nTB_LANDING=%s\nTB_VERSION=%s\n' "$PREFIX" "$PORT" "$LANDING" "$VERSION" | write "$MANIFEST" 0644
else
    printf '  + write %s (mode 0600)\n  + write %s\n' "$CONF" "$MANIFEST"
fi

render_sysctl | write "$SYSCTL" 0644
act sysctl -q -p "$SYSCTL" || warn "sysctl -p $SYSCTL failed (in a container this is expected)"

# firewall (persistent part; the hooks add the runtime part on every start)
if have firewall-cmd && [ "$NOFW" = 0 ]; then
    for x in zone-tailbond:zones/tailbond policy-tailbond-out:policies/tailbond-out policy-tailbond-in:policies/tailbond-in; do
        put "$BUNDLE/firewalld/${x%%:*}.xml" "/etc/firewalld/${x#*:}.xml" 0644
    done
    sed "s/port=\"443\"/port=\"$PORT\"/; s/on UDP 443/on UDP $PORT/; s/Forward this port on your home router\./Allow this port in your provider's firewall too./" \
        "$BUNDLE/firewalld/service-tailbond.xml" | write /etc/firewalld/services/tailbond.xml 0644
    if [ "$FW" = firewalld ]; then
        # Loads the new zone/policies/service. On hosts running containers,
        # netavark >= 1.9 restores its rules by itself (podman network reload
        # --all otherwise).
        act firewall-cmd -q --reload
    fi
fi
if [ "$FW" = ufw ]; then
    act ufw allow "$PORT/udp" comment tailbond
    act ufw allow in on tb0 comment tailbond
    if [ "$USE_TS" = 1 ]; then
        act ufw allow in on tailscale0 to any port 8077 proto tcp comment tailbond
        if [ "$LANDING" != -1 ]; then act ufw allow in on tailscale0 to any port "$LANDING" proto tcp comment tailbond; fi
        act ufw allow 41641/udp comment tailbond
    fi
fi
if [ "$FW" = nftables ] && nft list ruleset 2>/dev/null | grep -q 'hook input.*policy drop'; then
    warn "an nftables input chain here drops by default: allow udp dport $PORT (and iifname \"tb0\") in it, tailbond's own table can't override another table's drop"
fi
[ "$FW" = none ] && warn "no firewalld, nft or iptables: bonded clients won't get NAT to the internet ($PKG_HINT nftables)"

if systemd_on || [ "$DRY" = 1 ]; then
    render_unit | write "$UNIT" 0644
fi

if selinux_on; then
    act restorecon -RF "$BINDIR/mqvpn" "$BINDIR/tailbondd" "$LIBDIR" "$ETC" "$STATE" "$UNIT" "$SYSCTL"
    [ -d /etc/firewalld ] && act restorecon -RF /etc/firewalld
    [ -d "$SHARE" ] && act restorecon -RF "$SHARE"
fi

if ! systemd_on; then
    say "No systemd here: start it with  $BINDIR/tailbondd -config $CONF"
    [ "$DRY" = 1 ] || exit 0
fi
act systemctl daemon-reload
if [ "$NOSTART" = 1 ]; then
    say "Installed. Start with: systemctl enable --now tailbond"
    exit 0
fi
act systemctl enable --quiet tailbond.service
if [ "$WAS_ACTIVE" = 1 ] || [ "$DRY" = 1 ]; then act systemctl restart tailbond.service
else act systemctl start tailbond.service; fi
[ "$DRY" = 1 ] && exit 0

# GET /api/status without curl (minimal images don't have it).
api_status() {
    exec 3<>/dev/tcp/127.0.0.1/8077 || return 1
    printf 'GET /api/status HTTP/1.0\r\nHost: 127.0.0.1\r\n\r\n' >&3
    timeout 3 cat <&3
    exec 3<&-
}
st=""
for _ in $(seq 1 20); do
    st="$(api_status 2>/dev/null || true)"
    case "$st" in *'"state":"connected"'*) break ;; esac
    sleep 1
done
if [ -n "$st" ]; then
    say "tailbond is running: $(sed -n 's/.*"state":"\([a-z]*\)".*/\1/p' <<<"$st")"
else
    warn "the dashboard didn't answer yet: journalctl -u tailbond -n 50"
fi
if selinux_on && have ausearch; then
    if ausearch -m avc -ts recent 2>/dev/null | grep -qE 'tailbond|mqvpn'; then
        warn "SELinux denials mention tailbond/mqvpn: ausearch -m avc -ts recent (see README, SELinux)"
    fi
fi
cat <<EOF

Next:
- Allow udp/$PORT in your VPS provider's firewall / security group if it has one.
- Add a device (link + QR code):
    curl -s -X POST -H 'Content-Type: application/json' -d '{"name":"slate7"}' http://127.0.0.1:8077/api/clients
  The reply has "uri" (tailbond://pair?...) and "qr_svg" (a QR code to scan with the app).
EOF
if [ "$USE_TS" = 1 ]; then
    cat <<EOF
- Or from a router already paired with your home server: its panel, "Add another
  server", pick $SHORT; or the Android app's server list. Approve the request in
  the dashboard (http://$SHORT:8077/ from your tailnet).
- Approve the route $SUBNET for this machine in the Tailscale admin console if
  tailnet machines should reach bonded devices (optional).
EOF
fi
