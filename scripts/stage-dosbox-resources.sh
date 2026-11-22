#!/usr/bin/env bash
# Stage DOSBox Staging's Unicode→codepage mapping tables into the config dir.
#
# Without them the DOSBox shell cannot translate its own UTF-8 box-drawing
# characters and every border renders as '?' (measured on 0.83.0 and 0.83.1).
#
# Usage: ./scripts/stage-dosbox-resources.sh [path/to/DOSBox Staging.app]
#   - with an argument: copy from that .app bundle
#   - without: mount the DMG given by $DOSBOX_DMG (or download v0.83.1)
set -euo pipefail

# Portable layout: DOSBox switches to its own config dir when a
# dosbox-staging.conf sits next to the executable, so resources staged there
# travel with the project folder instead of living in the user's home dir.
CONFIG_DIR="$(cd "$(dirname "$0")/.." && pwd)/binaries/dosbox"
DIRS=(mapping mapping-freedos.org mapping-unicode.org mapping-wikipedia.org)

tmp=""
cleanup() { [[ -n "$tmp" ]] && rm -rf "$tmp"; }
trap cleanup EXIT

if [[ $# -ge 1 ]]; then
  APP="$1"
else
  VERSION="${VERSION:-0.83.1}"
  DMG="${DOSBOX_DMG:-}"
  if [[ -z "$DMG" ]]; then
    tmp="$(mktemp -d)"
    DMG="$tmp/dosbox.dmg"
    echo "Downloading DOSBox Staging $VERSION…"
    curl -fL -o "$DMG" "https://github.com/dosbox-staging/dosbox-staging/releases/download/v${VERSION}/dosbox-staging-macOS-v${VERSION}.dmg"
  fi
  MNT="$(mktemp -d)"
  hdiutil attach -quiet -nobrowse -mountpoint "$MNT" "$DMG"
  APP="$MNT/DOSBox Staging.app"
fi

mkdir -p "$CONFIG_DIR"
for d in "${DIRS[@]}"; do
  if [[ -d "$APP/Contents/Resources/$d" ]]; then
    rm -rf "$CONFIG_DIR/$d"
    cp -R "$APP/Contents/Resources/$d" "$CONFIG_DIR/"
    echo "staged $d"
  fi
done
if [[ -n "${MNT:-}" ]]; then hdiutil detach -quiet "$MNT" || true; fi
echo "Done: DOSBox resources are in $CONFIG_DIR (portable layout)"
