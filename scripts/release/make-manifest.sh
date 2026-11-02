#!/usr/bin/env bash
# SPDX-License-Identifier: Apache-2.0
#
# Write an update manifest <channel>.json (docs/architecture.md, "Updates")
# from a directory of release artifacts.
#
#   scripts/release/make-manifest.sh --version 0.2.0 --channel stable \
#       --base-url https://github.com/OWNER/REPO/releases/download/v0.2.0 \
#       --dir dist/ [--out DIR] [--protocol 1] [--requires-server-protocol 1] \
#       [--security] [--notes-url URL] [--released RFC3339] [--version-code N]
#
# Artifacts (anything else in --dir is ignored):
#   tailbond-linux-<arch>.tar.gz                  kind linux, arch <arch>
#   tailbond_<version>-rN_<arch>.ipk              kind ipk,   arch <arch>
#   tailbond-server-<version>-<rel>.<dist>.<arch>.rpm   kind rpm, arch <arch>
#   tailbond-<version>.apk                        kind apk,   arch arm64-v8a,
#                                                 plus version_code
# The ipk, rpm and apk names must carry --version (RPM pre-releases:
# 1.2.3-beta.1 is built as Version 1.2.3, Release 0.beta.1).
#
# version_code (Android) defaults to MAJOR*1000000 + MINOR*10000 + PATCH*100
# for a release (0.2.0 -> 20000) and that minus 100 plus N for a pre-release
# ending in a number N, 1..99; none or 0 counts as 1 (1.2.3-beta.4 -> 1020204,
# 1.2.3-rc -> 1020201), so a beta sorts below its release; the app's
# build.gradle.kts (android/scripts/build-release-apk.sh) uses the same rule.
#
# Output: DIR/<channel>.json (default DIR: .); prints its path.
# Needs: jq, sha256sum. Non-interactive.
set -euo pipefail

die() { echo "make-manifest: $*" >&2; exit 1; }
VERSION="" CHANNEL="" BASE="" DIR="" OUT="." PROTO=1 REQ=1 SECURITY=false NOTES="" RELEASED="" VCODE=""
while [ $# -gt 0 ]; do
    a="$1"
    case "$a" in --*=*) set -- "${a%%=*}" "${a#*=}" "${@:2}"; continue ;; esac
    case "$a" in
        --security) SECURITY=true; shift; continue ;;
        --no-security) SECURITY=false; shift; continue ;;
        -h|--help) sed -n '3,28p' "$0" | sed 's/^# \{0,1\}//'; exit 0 ;;
    esac
    [ $# -ge 2 ] || die "$a needs a value"
    case "$a" in
        --version) VERSION="${2#v}" ;;
        --channel) CHANNEL="$2" ;;
        --base-url) BASE="${2%/}" ;;
        --dir) DIR="$2" ;;
        --out) OUT="$2" ;;
        --protocol) PROTO="$2" ;;
        --requires-server-protocol) REQ="$2" ;;
        --notes-url) NOTES="$2" ;;
        --released) RELEASED="$2" ;;
        --version-code) VCODE="$2" ;;
        *) die "unknown option $a" ;;
    esac
    shift 2
done

command -v jq >/dev/null || die "jq not found"
[ -n "$VERSION" ] || die "--version is required"
[ -n "$CHANNEL" ] || die "--channel is required"
[ -n "$BASE" ] || die "--base-url is required"
[ -d "$DIR" ] || die "--dir: '$DIR' is not a directory"
case "$CHANNEL" in *[!a-z0-9-]*) die "--channel: lowercase letters, digits, - only" ;; esac
[[ "$VERSION" =~ ^([0-9]+)\.([0-9]+)\.([0-9]+)(-([0-9A-Za-z.-]+))?$ ]] ||
    die "--version '$VERSION' is not semver MAJOR.MINOR.PATCH[-pre]"
MAJ="${BASH_REMATCH[1]}" MIN="${BASH_REMATCH[2]}" PAT="${BASH_REMATCH[3]}" PRE="${BASH_REMATCH[5]}"
for n in "$PROTO" "$REQ"; do [[ "$n" =~ ^[0-9]+$ ]] || die "protocol numbers must be integers: $n"; done
RELEASED="${RELEASED:-$(date -u ${SOURCE_DATE_EPOCH:+-d "@$SOURCE_DATE_EPOCH"} +%Y-%m-%dT%H:%M:%SZ)}"

if [ -z "$VCODE" ]; then
    VCODE=$(( MAJ * 1000000 + MIN * 10000 + PAT * 100 ))
    if [ -n "$PRE" ]; then
        n=1 # no trailing number, or 0: 1 (as the app's build.gradle.kts)
        [[ "$PRE" =~ ([0-9]+)$ ]] && n=$(( 10#${BASH_REMATCH[1]} ))
        [ "$n" -ge 1 ] || n=1
        [ "$n" -le 99 ] || die "pre-release number $n > 99: pass --version-code"
        VCODE=$(( VCODE - 100 + n ))
    fi
fi
[[ "$VCODE" =~ ^[0-9]+$ ]] && [ "$VCODE" -gt 0 ] || die "--version-code must be a positive integer"

# RPM: no '-' in Version; a pre-release is Release 0.<pre>.
RPM_PREFIX="tailbond-server-$MAJ.$MIN.$PAT-$([ -n "$PRE" ] && echo "0.${PRE//-/_}." || echo 1.)"

arts="$(mktemp)"
trap 'rm -f "$arts"' EXIT
add() { # NAME KIND ARCH [VERSION_CODE]
    local f="$DIR/$1" sum size
    sum="$(sha256sum "$f" | cut -d' ' -f1)"
    size="$(stat -c %s "$f")"
    jq -nc --arg name "$1" --arg kind "$2" --arg arch "$3" --arg url "$BASE/$1" \
        --arg sha "$sum" --argjson size "$size" --arg vc "${4:-}" \
        '{name: $name, kind: $kind, arch: $arch, url: $url, sha256: $sha, size: $size}
         + (if $vc == "" then {} else {version_code: ($vc | tonumber)} end)' >>"$arts"
    echo "  $2 $3 $1" >&2
}

while IFS= read -r -d '' f; do
    n="$(basename "$f")"
    case "$n" in
        tailbond-linux-*.tar.gz)
            arch="${n#tailbond-linux-}"; arch="${arch%.tar.gz}"
            add "$n" linux "$arch" ;;
        tailbond_*.ipk)
            rest="${n#tailbond_}"; ver="${rest%%_*}"; arch="${rest#*_}"; arch="${arch%.ipk}"
            case "$ver" in "$VERSION"|"$VERSION"-r[0-9]*) ;; *) die "$n: version $ver, expected ${VERSION}[-rN]" ;; esac
            add "$n" ipk "$arch" ;;
        tailbond-server-*.src.rpm) ;;
        tailbond-server-*.rpm)
            case "$n" in "$RPM_PREFIX"*) ;; *) die "$n: expected $RPM_PREFIX…" ;; esac
            arch="${n%.rpm}"; arch="${arch##*.}"
            add "$n" rpm "$arch" ;;
        tailbond-"$VERSION".apk)
            add "$n" apk arm64-v8a "$VCODE" ;;
    esac
done < <(find "$DIR" -maxdepth 1 -type f -print0 | sort -z)

[ -s "$arts" ] || die "no artifacts in $DIR"
mkdir -p "$OUT"
out="$OUT/$CHANNEL.json"
jq -s --arg channel "$CHANNEL" --arg version "$VERSION" --arg released "$RELEASED" \
    --argjson protocol "$PROTO" --argjson req "$REQ" --argjson security "$SECURITY" --arg notes "$NOTES" \
    '{format: 1, channel: $channel, version: $version, released: $released, protocol: $protocol,
      requires_server_protocol: $req, security: $security, notes_url: $notes, artifacts: .}' \
    "$arts" >"$out.tmp"
mv "$out.tmp" "$out"
echo "$out"
