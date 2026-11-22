#!/usr/bin/env bash
# Fill the packaging manifests (Homebrew formula + Scoop manifest) from a
# GitHub Release: downloads the zips, computes SHA-256, rewrites <TAG> and the
# hash placeholders. Run this after publishing a release.
#
# Usage: scripts/update-packaging.sh <TAG>
#   e.g. scripts/update-packaging.sh 20261122-183000
set -euo pipefail

TAG="${1:?usage: scripts/update-packaging.sh <TAG>}"
REPO="bytecapsule/doscapsule"
BASE="https://github.com/${REPO}/releases/download/${TAG}"
ROOT="$(cd "$(dirname "$0")/.." && pwd)"
TMP="$(mktemp -d)"
trap 'rm -rf "$TMP"' EXIT

fetch_sha() { # <file-name> -> prints sha256
  local f="$1"
  curl -fsSL -o "$TMP/$f" "$BASE/$f"
  local sha
  if command -v shasum >/dev/null 2>&1; then
    sha="$(shasum -a 256 "$TMP/$f" | awk '{print $1}')"
  else
    sha="$(sha256sum "$TMP/$f" | awk '{print $1}')"
  fi
  printf '%s' "$sha"
}

echo "Resolving ${TAG}…"
SHA_ARM="$(fetch_sha "doscapsule-arm64-${TAG}.zip")"
SHA_X64="$(fetch_sha "doscapsule-x64-${TAG}.zip")"
SHA_WIN="$(fetch_sha "doscapsule-windows-x64-${TAG}.zip")"
echo "  arm64 : ${SHA_ARM}"
echo "  x64   : ${SHA_X64}"
echo "  win64 : ${SHA_WIN}"

# Homebrew formula: replace <TAG> and the two sha256 placeholders.
FORMULA="$ROOT/homebrew/doscapsule.rb"
python3 - "$FORMULA" "$TAG" "$SHA_ARM" "$SHA_X64" <<'PYEOF'
import re, sys
path, tag, sha_arm, sha_x64 = sys.argv[1:5]
s = open(path).read()
s = s.replace('<TAG>', tag)
# first placeholder is arm64, second is x64
s = s.replace('<SHA256_ARM64>', sha_arm, 1).replace('<SHA256_X64>', sha_x64, 1)
open(path, 'w').write(s)
print("  updated:", path)
PYEOF

# Scoop manifest.
SCOOP="$ROOT/scoop/doscapsule.json"
python3 - "$SCOOP" "$TAG" "$SHA_WIN" <<'PYEOF'
import sys
path, tag, sha_win = sys.argv[1:4]
s = open(path).read()
s = s.replace('REPLACE_WITH_TAG', tag).replace('REPLACE_WITH_SHA256', sha_win)
open(path, 'w').write(s)
print("  updated:", path)
PYEOF

echo "Done. Review the diff, then commit the packaging update."
