#!/bin/bash
# Pinned, checksum-verified install of the Antigravity CLI (`agy`).
#
# Replays the verify chain of Google's installer (https://antigravity.google/cli/install.sh)
# with three differences, each deliberate (design_docs/planned/m-agy-executor.md, D4):
#   1. The version is PINNED here, not "whatever the manifest says is latest". A silent
#      CLI change under a longitudinal series is the pi-pin lesson.
#   2. It never runs `agy install`, which is the step that edits shell rc files.
#   3. It replaces the binary with an atomic rename, never an in-place copy over a file
#      a running process may hold (memory: never cp over a live binary).
#
# To bump: read the manifest below, take version/url/sha512 from it, update the three
# AGY_* values together, run `make install-agy`, then `go test ./internal/executor/agy/`.
#   https://antigravity-cli-auto-updater-974169037036.us-central1.run.app/manifests/darwin_arm64.json
#
# Usage: tools/agy/install.sh [--dir <bin-dir>] [--check]
#   --check  only report whether the installed binary is the pinned version (exit 1 if not)
set -euo pipefail

AGY_VERSION="1.5.15"
AGY_URL_DARWIN_ARM64="https://storage.googleapis.com/antigravity-public/antigravity-cli/1.5.15-5514532022517760/darwin-arm/cli_mac_arm64.tar.gz"
AGY_SHA512_DARWIN_ARM64="b03a841eb3a29be918a66960b775f68ed4f3bcc02f33962131bbbc26c46ea983ceac6c832ad46efd11d73d9a2cb97df4ab24d4bc92ee47d3d5ece7b8d8b61235"

BIN_DIR="${HOME}/.local/bin"
CHECK_ONLY=0
while [ $# -gt 0 ]; do
  case "$1" in
    --dir) BIN_DIR="$2"; shift 2 ;;
    --check) CHECK_ONLY=1; shift ;;
    -h|--help) sed -n '2,18p' "$0"; exit 0 ;;
    *) echo "install-agy: unknown argument: $1" >&2; exit 2 ;;
  esac
done
TARGET="${BIN_DIR}/agy"

installed_version() {
  [ -x "$TARGET" ] || return 1
  # `agy --version` prints the bare version (e.g. "1.5.15"). Disable the self-updater
  # so a version probe can never trigger a background update.
  AGY_CLI_DISABLE_AUTO_UPDATE=true "$TARGET" --version 2>/dev/null </dev/null | head -1 | tr -d '[:space:]'
}

current=$(installed_version || true)
if [ "$CHECK_ONLY" = 1 ]; then
  if [ "$current" = "$AGY_VERSION" ]; then
    echo "agy ${current} at ${TARGET} (pinned ${AGY_VERSION}): OK"; exit 0
  fi
  echo "agy at ${TARGET} is '${current:-absent}', pinned ${AGY_VERSION}: run make install-agy" >&2; exit 1
fi
if [ "$current" = "$AGY_VERSION" ]; then
  echo "agy ${AGY_VERSION} already installed at ${TARGET}"; exit 0
fi

case "$(uname -s)/$(uname -m)" in
  Darwin/arm64) url="$AGY_URL_DARWIN_ARM64"; want="$AGY_SHA512_DARWIN_ARM64" ;;
  *) echo "install-agy: only darwin/arm64 is pinned (got $(uname -s)/$(uname -m)); add its manifest values" >&2; exit 1 ;;
esac

stage=$(mktemp -d "${TMPDIR:-/tmp}/agy-install.XXXXXX")
trap 'rm -rf "$stage"' EXIT
echo "Downloading agy ${AGY_VERSION}..."
curl -fsSL --retry 2 -o "$stage/payload.tar.gz" "$url"

got=$(shasum -a 512 "$stage/payload.tar.gz" | awk '{print $1}')
if [ "$got" != "$want" ]; then
  echo "install-agy: SHA512 MISMATCH — refusing to install." >&2
  echo "  want ${want}" >&2
  echo "  got  ${got}" >&2
  exit 1
fi
echo "SHA512 verified."

tar -xzf "$stage/payload.tar.gz" -C "$stage" antigravity
chmod +x "$stage/antigravity"
xattr -d com.apple.quarantine "$stage/antigravity" 2>/dev/null || true

mkdir -p "$BIN_DIR"
# Same filesystem as the target, so the final mv is an atomic rename.
tmp="${BIN_DIR}/.agy.new.$$"
cp "$stage/antigravity" "$tmp"
mv -f "$tmp" "$TARGET"

now=$(installed_version || true)
if [ "$now" != "$AGY_VERSION" ]; then
  echo "install-agy: installed binary reports '${now}', expected ${AGY_VERSION}" >&2; exit 1
fi
echo "agy ${AGY_VERSION} installed at ${TARGET} (shell rc files untouched)"
