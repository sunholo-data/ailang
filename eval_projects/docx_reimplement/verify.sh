#!/usr/bin/env bash
# verify.sh <ailang-parse-dir> — grade a (re)implementation of docparse/services/docx_parser.ail
# against captured golden. Runs docparse on each DOCX fixture in the candidate repo and diffs the
# deterministic stdout (content blocks + summary) against golden/<fixture>.golden.
set -uo pipefail
AP="${1:?usage: verify.sh <ailang-parse-dir>}"
GOLDEN="$(cd "$(dirname "$0")/golden" && pwd)"
PASS=0; FAIL=0
for g in "$GOLDEN"/*.golden; do
  base=$(basename "$g" .golden)
  got=$(cd "$AP" && ailang run --entry main --caps IO,FS,Env docparse/main.ail "data/test_files/$base.docx" 2>/dev/null \
        | grep -vE "Type checking|Effect checking|✓ Running|written to")
  if diff <(printf '%s\n' "$got") "$g" >/dev/null 2>&1; then
    PASS=$((PASS+1))
  else
    FAIL=$((FAIL+1)); echo "  ✗ $base"
  fi
done
echo "Result: $PASS passed, $FAIL failed"
exit $FAIL
