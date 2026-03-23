#!/usr/bin/env bash
# Validate an AILANG package: lockfile, type-check, exports
set -e

DIR="${1:-.}"
cd "$DIR"

echo "=== AILANG Package Validation ==="
echo ""

# Check ailang.toml exists
if [ ! -f ailang.toml ]; then
  echo "ERROR: No ailang.toml found in $DIR"
  echo "Run: ailang init package --name vendor/name"
  exit 1
fi

# Extract package name
PKG_NAME=$(grep '^name = ' ailang.toml | head -1 | sed 's/name = "//;s/"//')
echo "Package: $PKG_NAME"

# Generate lockfile
echo ""
echo "--- Resolving dependencies ---"
ailang lock 2>&1

# Run package-level type check
echo ""
echo "--- Type checking ---"
ailang check --package . 2>&1

echo ""
echo "=== Validation complete ==="
