#!/usr/bin/env bash
# SPDX-License-Identifier: Apache-2.0
#
# Git stays lean: release artifacts live only on GitHub Releases (README,
# Download). Fails when a tracked file is a build output (*.apk, *.ipk,
# *.rpm, *.tar.gz, *.dmg, *.pkg, *.icns, *.so, *.a) or is over 1 MB.
# Gradle's wrapper jar is the one binary the build needs from git.
#
#   scripts/ci/check-tracked-files.sh
set -euo pipefail

cd "$(dirname "$0")/../.."
max=$((1024 * 1024))
fail=0
while IFS= read -r -d '' f; do
    case "$f" in
        android/gradle/wrapper/gradle-wrapper.jar) continue ;;
        *.apk|*.ipk|*.rpm|*.tar.gz|*.dmg|*.pkg|*.icns|*.so|*.a)
            echo "::error file=$f::build output in git: $f. Release files go on GitHub Releases (scripts/release/collect-assets.sh); remove it with git rm --cached."
            fail=1
            continue ;;
    esac
    # submodules are gitlinks, not files
    [ -f "$f" ] && [ ! -L "$f" ] || continue
    size="$(wc -c <"$f")"
    if [ "$size" -gt "$max" ]; then
        echo "::error file=$f::$f is $size bytes, over 1 MB: keep big or generated files out of git (generate them in the build, or attach them to a release)."
        fail=1
    fi
done < <(git ls-files -z)
[ "$fail" = 0 ] && echo "tracked files: no build outputs, none over 1 MB"
exit "$fail"
