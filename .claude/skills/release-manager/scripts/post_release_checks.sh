#!/usr/bin/env bash
# Verify release was created successfully

set -euo pipefail

if [[ $# -eq 0 ]]; then
    echo "Usage: $0 <version>" >&2
    echo "Example: $0 0.3.14" >&2
    exit 1
fi

VERSION="$1"
TAG="v$VERSION"

echo "Verifying release $TAG..."
echo

# Track failures
FAILURES=0

# Check git tag exists
echo "1/4 Checking git tag..."
if git tag -l "$TAG" | grep -q "$TAG"; then
    echo "  ✓ Tag $TAG exists"
else
    echo "  ✗ Tag $TAG not found"
    FAILURES=$((FAILURES + 1))
fi
echo

# Check GitHub release exists
echo "2/4 Checking GitHub release..."
if gh release view "$TAG" > /dev/null 2>&1; then
    echo "  ✓ GitHub release $TAG exists"
else
    echo "  ✗ GitHub release $TAG not found"
    FAILURES=$((FAILURES + 1))
fi
echo

# Check release binaries
# Updated for Gemini CLI naming convention (v0.5.7+)
# Format: {platform}.{arch}.ailang.{ext}
echo "3/4 Checking release binaries..."
EXPECTED_BINARIES=(
    "darwin.x64.ailang.tar.gz"
    "darwin.arm64.ailang.tar.gz"
    "linux.x64.ailang.tar.gz"
    "win32.x64.ailang.zip"
)

BINARY_FAILURES=0
for binary in "${EXPECTED_BINARIES[@]}"; do
    if gh release view "$TAG" --json assets --jq ".assets[].name" | grep -q "^$binary$"; then
        echo "  ✓ $binary"
    else
        echo "  ✗ $binary (missing)"
        BINARY_FAILURES=$((BINARY_FAILURES + 1))
    fi
done

if [[ $BINARY_FAILURES -eq 0 ]]; then
    echo "  ✓ All platform binaries present"
else
    echo "  ✗ $BINARY_FAILURES binaries missing"
    FAILURES=$((FAILURES + 1))
fi
echo

# Check CI status
echo "4/4 Checking CI status..."
if gh run list --limit 1 --json conclusion --jq '.[0].conclusion' | grep -q "success"; then
    echo "  ✓ Latest CI run passed"
else
    echo "  ⚠ Latest CI run did not pass (may still be running)"
    echo "  Check with: gh run list --limit 3"
fi
echo

# Summary
if [[ $FAILURES -eq 0 ]]; then
    echo "✓ Release $TAG verified successfully!"
    echo "URL: https://github.com/sunholo-data/ailang/releases/tag/$TAG"
    exit 0
else
    echo "✗ $FAILURES check(s) failed"
    echo "Release may be incomplete."
    exit 1
fi
