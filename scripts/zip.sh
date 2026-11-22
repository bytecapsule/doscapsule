#!/usr/bin/env bash
# Package a launcher binary into a release zip. DOSBox is NOT included.
#
# Usage: scripts/zip.sh <binary> [arch-label] [version]
#   binary      path to the built executable
#   arch-label  arm64 / x64 / windows-x64 (optional; adds a suffix to the name)
#   version     release tag (optional; defaults to asking the binary, then a
#               timestamp). Pass it explicitly when cross-compiling, since a
#               Windows .exe cannot be executed on Linux to read its version.
#
# The zip is FLAT (binary at the archive root) on purpose: both Homebrew's
# `bin.install "doscapsule"` and Scoop's `"bin": "doscapsule.exe"` resolve
# relative to the extraction root, and a wrapper directory would break them.
set -euo pipefail

BIN="${1:?usage: scripts/zip.sh <binary> [arch-label] [version]}"
ARCH="${2:-}"
VERSION_ARG="${3:-}"

ROOT="$(cd "$(dirname "$0")/.." && pwd)"
cd "$ROOT"

if [[ ! -f "$BIN" ]]; then
  echo "error: binary not found: $BIN (run 'make build' first)" >&2
  exit 1
fi

if [[ -n "$VERSION_ARG" ]]; then
  VERSION="$VERSION_ARG"
else
  VERSION="$(./"$BIN" -version 2>/dev/null | awk '{print $3}' || true)"
  [[ -z "$VERSION" ]] && VERSION="$(date +%Y%m%d-%H%M%S)"
fi

NAME="doscapsule"
[[ -n "$ARCH" ]] && NAME="${NAME}-${ARCH}"
NAME="${NAME}-${VERSION}"

STAGE="$(mktemp -d)"
trap 'rm -rf "$STAGE"' EXIT

# Flat archive root: binary + docs, nothing above them.
case "$BIN" in
  *.exe) cp "$BIN" "$STAGE/doscapsule.exe" ;;
  *)     cp "$BIN" "$STAGE/doscapsule" ;;
esac
cp README.md "$STAGE/README.md"
mkdir -p "$STAGE/docs"
cp docs/usage.md docs/configuration.md "$STAGE/docs/" 2>/dev/null || true

cat > "$STAGE/INSTALL-DOSBox.txt" <<'EOT'
DOSBox must be installed separately:

  macOS:    brew install dosbox-staging
  Windows:  scoop install dosbox-staging
  or download from: https://dosbox-staging.github.io/

Once installed, the launcher finds dosbox-staging automatically on PATH.
Point it at a games folder and start playing.
EOT

( cd "$STAGE" && zip -rq "$ROOT/${NAME}.zip" . -x '.*' )
echo "packaged: $ROOT/${NAME}.zip"
