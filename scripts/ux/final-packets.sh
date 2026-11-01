#!/usr/bin/env bash
# Assemble what each round's fresh reviewers get, per surface: only the
# PNGs, the task script and the manifest (plus the Fedora install guide).
#
#   scripts/ux/final-packets.sh out/ux/final/round-N
#
# Expects the round's renders in <round>/web (web-screens.sh),
# <round>/glinet (glinet-screens.sh) and <round>/android (Roborazzi).
set -euo pipefail

REPO="$(cd "$(dirname "$0")/../.." && pwd)"
R="$(cd "$1" && pwd)"
D="$REPO/docs/ux"
rm -rf "$R"/packet-*
mkdir -p "$R"/packet-gl "$R"/packet-android/{web,android} "$R"/packet-fedora/{web,glinet}

cp "$D/glinet/task-script.md" "$D/glinet/screens.md" "$R/packet-gl/"
for f in "$R"/glinet/*.png; do
    case "$(basename "$f")" in 1[45][b-d]-fedora-*) ;; *) cp "$f" "$R/packet-gl/" ;; esac
done

cp "$D/final/android-task-script.md" "$R/packet-android/task-script.md"
cp "$D/final/android-screens.md" "$R/packet-android/screens.md"
cp "$R"/web/web-00*.png "$R/packet-android/web/"
cp "$R"/android/*.png "$R/packet-android/android/"
rm -f "$R/packet-android/android/03-deeplink-confirm.png"  # same dialog as 02e

cp "$D/final/fedora-task-script.md" "$R/packet-fedora/task-script.md"
cp "$D/final/fedora-screens.md" "$R/packet-fedora/screens.md"
cp "$REPO/docs/fedora-server.md" "$R/packet-fedora/"
cp "$R"/glinet/1[45]*-fedora-*.png "$R/packet-fedora/glinet/"
cp "$R/web/web-00-phone-page.png" "$R/web/web-00c-phone-page-on-computer.png" "$R/web/web-04-dashboard-devices-phone-card.png" "$R/packet-fedora/web/"
for p in "$R"/packet-*; do echo "$(basename "$p"): $(find "$p" -name '*.png' | wc -l) PNGs"; done
