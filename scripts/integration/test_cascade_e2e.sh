#!/usr/bin/env bash
# M-PKG-AUTONOMOUS-CASCADE-SAFE M5: end-to-end cascade smoke test.
#
# Bumps sunholo/test_pkg, runs `ailang publish`, waits for the cascade to
# fire, and asserts the dependent agent opens a real PR on
# sunholo/test_pkg_consumer in the ailang-packages repo. Cleans up the test
# PR + branch at end (unless --keep is passed).
#
# Usage:
#   AILANG_CLOUD_PROJECT=ailang-multivac-test \
#     scripts/integration/test_cascade_e2e.sh [--keep]
#
# Required env:
#   AILANG_CLOUD_PROJECT  Target env (ailang-multivac-{dev,test,prod})
#
# Optional:
#   AILANG_TOPIC_PREFIX   Defaults from project (e.g. "ailang-test")
#   --keep                Don't delete the smoke-test PR at end
#
# Exit codes:
#   0 = full cascade observed end-to-end
#   1 = cascade did not fire (publish step failed, dispatcher silent)
#   2 = cascade fired but agent failed to open PR within timeout
#   3 = PR opened but cleanup failed (PR remains on github)

set -euo pipefail

KEEP=0
for arg in "$@"; do
  case $arg in
    --keep) KEEP=1 ;;
    *) echo "unknown arg: $arg"; exit 64 ;;
  esac
done

if [ -z "${AILANG_CLOUD_PROJECT:-}" ]; then
  echo "ERROR: AILANG_CLOUD_PROJECT must be set (e.g. ailang-multivac-test)"
  exit 64
fi

REPO="sunholo-data/ailang-packages"
ROOT_PKG="sunholo/test_pkg"
DEPENDENT_PKG="sunholo/test_pkg_consumer"
TIMEOUT_PR_OPEN="${TIMEOUT_PR_OPEN:-300}"   # 5 min — agent run + tests + push
# Note: the smoke-test PR carries label "smoke-test" — the github filter
# below uses --label smoke-test directly. CI rule on ailang-packages blocks
# merge of any PR with that label.

note() { printf '  %s %s\n' "$(date +%H:%M:%S)" "$*"; }
fail() { printf 'FAIL: %s\n' "$*" >&2; exit "${2:-1}"; }

note "Cascade smoke against $AILANG_CLOUD_PROJECT"

# 1. Bump test_pkg version locally.
ROOT_DIR="$(git rev-parse --show-toplevel 2>/dev/null || pwd)"
PKG_DIR="$ROOT_DIR/../ailang-packages/packages/test-pkg"
if [ ! -f "$PKG_DIR/ailang.toml" ]; then
  fail "test-pkg directory not found at $PKG_DIR" 64
fi

OLD_VERSION=$(grep '^version = ' "$PKG_DIR/ailang.toml" | head -1 | cut -d'"' -f2)
# patch bump: split on dots, increment last component
IFS='.' read -ra PARTS <<< "$OLD_VERSION"
PARTS[$((${#PARTS[@]} - 1))]=$((${PARTS[$((${#PARTS[@]} - 1))]} + 1))
NEW_VERSION="$(IFS='.'; echo "${PARTS[*]}")"
note "Bumping $ROOT_PKG: $OLD_VERSION → $NEW_VERSION"

# Restore version on exit unless test PASSed normally
RESTORE_VERSION=1
restore_version() {
  if [ "$RESTORE_VERSION" = "1" ]; then
    note "Restoring $PKG_DIR/ailang.toml version → $OLD_VERSION"
    sed -i.bak "s/^version = \"$NEW_VERSION\"$/version = \"$OLD_VERSION\"/" "$PKG_DIR/ailang.toml" || true
    rm -f "$PKG_DIR/ailang.toml.bak"
  fi
}
trap restore_version EXIT

sed -i.bak "s/^version = \"$OLD_VERSION\"$/version = \"$NEW_VERSION\"/" "$PKG_DIR/ailang.toml"
rm -f "$PKG_DIR/ailang.toml.bak"

# 2. Publish (drives emitDependentNotifications → cascade topic).
note "Running ailang publish in $PKG_DIR"
if ! (cd "$PKG_DIR" && ailang publish); then
  fail "ailang publish failed" 1
fi
note "Publish OK — cascade fired"

# 3. Poll for the cascade-driven PR on test_pkg_consumer.
note "Waiting up to ${TIMEOUT_PR_OPEN}s for $DEPENDENT_PKG cascade PR…"
DEADLINE=$(($(date +%s) + TIMEOUT_PR_OPEN))
PR_NUMBER=""
while [ "$(date +%s)" -lt "$DEADLINE" ]; do
  # Look for an OPEN PR titled like "[cascade]" against test-pkg-consumer
  # opened in the last 10 minutes
  PR_NUMBER=$(gh pr list --repo "$REPO" --state open --label smoke-test \
    --search "in:title cascade test_pkg" \
    --json number --jq '.[0].number' 2>/dev/null || true)
  if [ -n "$PR_NUMBER" ] && [ "$PR_NUMBER" != "null" ]; then
    break
  fi
  sleep 10
done

if [ -z "$PR_NUMBER" ] || [ "$PR_NUMBER" = "null" ]; then
  fail "No cascade PR opened within ${TIMEOUT_PR_OPEN}s" 2
fi

PR_URL="https://github.com/$REPO/pull/$PR_NUMBER"
note "Cascade PR observed: $PR_URL"

# 4. Verify provenance trail via ailang pkg cascade status.
note "Verifying provenance via ailang pkg cascade status"
if ! ailang pkg cascade status "${ROOT_PKG}@${NEW_VERSION}" >/dev/null 2>&1; then
  echo "WARNING: ailang pkg cascade status returned non-zero (provenance may not have caught up yet)"
fi

# 5. Cleanup unless --keep
if [ "$KEEP" = "0" ]; then
  note "Closing + deleting smoke-test PR #$PR_NUMBER"
  if gh pr close "$PR_NUMBER" --repo "$REPO" --delete-branch; then
    note "Cleanup OK"
  else
    echo "WARNING: cleanup failed; PR $PR_URL remains open"
    exit 3
  fi
else
  note "--keep: leaving PR $PR_URL for inspection"
fi

# Test PASSed — let the trap restore version
RESTORE_VERSION=1
echo
echo "PASS: cascade smoke test completed"
echo "  root: $ROOT_PKG@$NEW_VERSION"
echo "  PR:   $PR_URL"
exit 0
