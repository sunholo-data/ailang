#!/usr/bin/env bash
#
# feedback_flood_drill.sh — OFFLINE flood simulation for the feedback cost/abuse
# gate (M-FEEDBACK-TRIAGE-GATE).
#
# Feeds N synthetic public submissions from a set of distinct contacts through a
# fully-assembled internal/feedbackgate gate (deterministic rules + in-memory
# sliding cooldown) and prints a verdict histogram + a simulated-spend line.
#
# This is DELIBERATELY OFFLINE: it exercises the in-repo gate/test env only.
#   - NO cloud credentials
#   - NO live Anthropic / Sonnet / Haiku calls (the classifier stage is not wired
#     here; rules + cooldown alone demonstrate the flood cap)
#   - NO Ollama / GPU
#
# The real cloud flood drill (1,000 msgs against the live test env with actual
# spend measurement) is a separate ops task, not this script.
#
# Usage:
#   scripts/security/feedback_flood_drill.sh [N] [CONTACTS]
#
# Defaults: N=1000 submissions across CONTACTS=10 distinct contacts.

set -euo pipefail

N="${1:-1000}"
CONTACTS="${2:-10}"

# Locate the repo root (this script lives in scripts/security/).
SCRIPT_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")" && pwd)"
REPO_ROOT="$(cd "${SCRIPT_DIR}/../.." && pwd)"

echo "Running OFFLINE feedback flood drill: N=${N} submissions, CONTACTS=${CONTACTS}"
echo "(no cloud creds, no live API, no Ollama)"
echo

cd "${REPO_ROOT}"

# Drive the offline drill engine via its test entrypoint. -count=1 disables the
# test cache so the report always prints. FEEDBACK_FLOOD_DRILL=1 un-skips it.
FEEDBACK_FLOOD_DRILL=1 FLOOD_N="${N}" FLOOD_CONTACTS="${CONTACTS}" \
  go test ./internal/feedbackgate/ \
    -run TestFloodDrillReportEntrypoint \
    -count=1 -v 2>&1 |
  grep -vE '^(=== RUN|--- PASS|--- FAIL|PASS|FAIL|ok|\?)' || true

echo
echo "Flood drill complete."
