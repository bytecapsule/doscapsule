#!/usr/bin/env bash
# Package a launcher binary into a release zip (DOSBox NOT included).
#
# Usage: scripts/zip.sh <binary> [arch]
#   binary: path to the built executable (e.g. doscapsule or doscapsule-arm64)
#   arch:   optional label for the zip name (arm64 / x64 / amd64)
set -euo pipefail

BIN="${1:?usage: scripts/zip.sh <binary> [arch]}"
ARCH="${2:-}"

# Resolve project root.
ROOT="$(cd "$(dirname "$0")/.." && pwd)"
cd "$ROOT"

if [[ ! -f "$BIN" ]]; then
  echo "error: binary not found: $BIN (run 'make build' first)" >&2
  exit 1
fi

VERSION="$(./$BIN -version 2>/dev/null | awk '{print $3}')"
[[ -z "$VERSION" ]] && VERSION="$(date +%Y%m%d-%H%M%S)"
NAME="doscapsule"
[[ -n "$ARCH" ]] && NAME="${NAME}-${ARCH}"
[[ -n "$VERSION" ]] && NAME="${NAME}-${VERSION}"

STAGE="$(mktemp -d)"
trap 'rm -rf "$STAGE"' EXIT
mkdir -p "$STAGE/$NAME"

# Binary + docs + a tiny README inside the zip.
cp "$BIN" "$STAGE/$NAME/doscapsule"
cp README.md "$STAGE/$NAME/README.md"
mkdir -p "$STAGE/$NAME/docs"
cp docs/usage.md docs/configuration.md "$STAGE/$NAME/docs/" 2>/dev/null || true

# An OSX-unzip hint note.
cat > "$STAGE/$NAME/INSTALL-DOSBox.txt" <<'EOT'
DOSBox must be installed separately:

  macOS:  brew install dosbox-staging
  or download from: https://dosbox-staging.github.io/

Once installed, the launcher finds dosbox-staging automatically on PATH.
Point it at a games folder and start playing.
EOT

( cd "$STAGE" && zip -rq "$ROOT/${NAME}.zip" "$NAME" )
echo "packaged: $ROOT/${NAME}.zip"
