#!/usr/bin/env bash
# SPDX-License-Identifier: Apache-2.0
#
# The macOS installer's server helpers (packaging/macos/install.sh) on canned
# macOS output, so they're checked anywhere (CI: job ShellCheck, Linux):
#
# - lsof_local_holder: only a socket whose LOCAL port is P holds udp/P; a
#   browser's QUIC connection to some server's 443 doesn't (PR 19 review).
# - netstat_prefixes / ifconfig_prefixes: macOS's abbreviated route
#   destinations ("10/8", "10.64/10", "192.168.64") and hex netmasks.
# - subnet_clash / pick_subnet: a real CIDR overlap check, so a covering
#   route (10/8, 10.64/10) rules a /24 out (PR 19 review).
#
#   scripts/test-macos-install-helpers.sh
set -euo pipefail

ROOT="$(cd "$(dirname "$0")/.." && pwd)"
# shellcheck source=../packaging/macos/install.sh
TB_INSTALL_SOURCE_ONLY=1 . "$ROOT/packaging/macos/install.sh"
set +e
PASS=0 FAIL=0
ok()  { printf '  PASS %s\n' "$*"; PASS=$((PASS + 1)); }
bad() { printf '  FAIL %s\n' "$*"; FAIL=$((FAIL + 1)); }
eq()  { if [ "$2" = "$3" ]; then ok "$1"; else bad "$1: got [$2], want [$3]"; fi; }

LSOF='COMMAND     PID   USER   FD   TYPE             DEVICE SIZE/OFF NODE NAME
Google     1201 alex   41u  IPv4 0x1234567890abcdef      0t0  UDP 192.168.1.20:61234->142.250.80.36:443
Safari     1300 alex   22u  IPv6 0x1234567890abcde0      0t0  UDP [2001:db8::20]:50001->[2606:4700::1]:443
mDNSRespo   300 _mdns   8u  IPv4 0x1234567890abcde1      0t0  UDP *:5353'
eq "remote port 443 (browser QUIC) is not a holder of udp/443" "$(lsof_local_holder 443 <<<"$LSOF")" ""
eq "a local *:5353 is" "$(lsof_local_holder 5353 <<<"$LSOF")" "mDNSRespo"
CADDY="$LSOF
caddy      1500 root    9u  IPv6 0x1234567890abcde2      0t0  UDP *:443
mqvpn      1600 root    5u  IPv4 0x1234567890abcde3      0t0  UDP *:8443"
eq "a server on *:443 is" "$(lsof_local_holder 443 <<<"$CADDY")" "caddy"
eq "our own mqvpn isn't a foreign holder" "$(lsof_local_holder 8443 <<<"$CADDY")" ""
eq "a bound local address too" "$(lsof_local_holder 4443 <<<'COMMAND PID USER FD TYPE DEVICE SIZE/OFF NODE NAME
nc 1 a 3u IPv4 0x1 0t0 UDP 192.168.1.20:4443')" "nc"

NETSTAT='Routing tables

Internet:
Destination        Gateway            Flags               Netif Expire
default            192.168.64.1       UGScg                 en0
10/8               utun3              USc                 utun3
10.64/10           link#22            UCS                 utun4
127                127.0.0.1          UCS                   lo0
127.0.0.1          127.0.0.1          UH                    lo0
169.254            link#11            UCS                   en0      !
169.254.169.254    link#11            UHLSW                 en0      !
192.168.64         link#11            UCS                   en0      !
192.168.64.1/32    link#11            UCS                   en0      !
224.0.0/4          link#11            UmCS                  en0      !
fe80::%lo0/64      fe80::1%lo0        UcI                   lo0'
eq "netstat destinations as CIDRs" "$(netstat_prefixes <<<"$NETSTAT" | tr '\n' ' ')" \
  "10.0.0.0/8 10.64.0.0/10 127.0.0.0/8 127.0.0.1/32 169.254.0.0/16 169.254.169.254/32 192.168.64.0/24 192.168.64.1/32 224.0.0.0/4 "
IFC='en0: flags=8863<UP,BROADCAST,SMART,RUNNING,SIMPLEX,MULTICAST> mtu 1500
	inet 192.168.64.18 netmask 0xffffff00 broadcast 192.168.64.255
utun4: flags=8051<UP,POINTOPOINT,RUNNING,MULTICAST> mtu 1280
	inet 100.101.0.1 --> 100.101.0.2 netmask 0xff000000
en7: flags=8863<UP,BROADCAST,SMART,RUNNING,SIMPLEX,MULTICAST> mtu 1500
	inet 10.80.0.5 netmask 0xffff0000 broadcast 10.80.255.255'
eq "ifconfig addresses with their netmasks" "$(ifconfig_prefixes <<<"$IFC" | tr '\n' ' ')" "192.168.64.18/24 100.101.0.1/8 10.80.0.5/16 "

# a covering 10/8 (a work VPN) rules out every 10.N/24
U="$(netstat_prefixes <<<"$NETSTAT")"
eq "10/8 covers 10.78.0.0/24" "$(subnet_clash 10.78.0.0/24 "$U")" "10.0.0.0/8"
pick_subnet "$U" >/dev/null && bad "a subnet was picked under 10/8" || ok "nothing picked under 10/8 (the installer asks for --subnet)"
# without it: 10.64/10 covers 10.64-10.127, so 10.78 and 10.79 are out too
U2="$(grep -v '^10.0.0.0/8$' <<<"$U")"
eq "10.64/10 covers 10.78.0.0/24" "$(subnet_clash 10.78.0.0/24 "$U2")" "10.64.0.0/10"
pick_subnet "$U2" >/dev/null && bad "a subnet was picked under 10.64/10" || ok "nothing in 10.78-10.99 is free under 10.64/10"
# a narrower route and an address: the next free /24 is used
U3="10.78.0.0/16
10.79.0.0/16
$(ifconfig_prefixes <<<"$IFC")"
eq "10.78/16 and 10.79/16 in use, 10.80/16 on en7: 10.81.0.0/24" "$(pick_subnet "$U3")" "10.81.0.0/24"
eq "no clash on a free Mac" "$(subnet_clash 10.78.0.0/24 "192.168.64.0/24" || echo none)" "none"

printf '\n%d passed, %d failed\n' "$PASS" "$FAIL"
[ "$FAIL" -eq 0 ]
