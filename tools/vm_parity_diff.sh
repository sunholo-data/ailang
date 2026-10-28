#!/bin/bash
# vm_parity_diff.sh — VM parity gate for VM performance work (M-VM-QUICK-WINS).
#
# Builds ailang from the working tree, runs scripts/verify_bytecode_parity.go
# over examples/runnable under --bytecode AND --strict-bytecode, and reduces
# each run to one outcome per file: the harness status plus the VM exit code.
# Stdout is not stored: MATCH already means "VM stdout == evaluator stdout",
# and timing-printing examples would make raw stdout differ run to run.
#
# Usage:
#   tools/vm_parity_diff.sh --bank <file.json>      # record a baseline
#   tools/vm_parity_diff.sh <baseline.json>         # exit 1 if any outcome changed
#   tools/vm_parity_diff.sh --control <baseline.json>
#       positive control: flips one baseline row in a copy and must exit 1
#
# bash 3.2 / BSD tool compatible.
set -euo pipefail
cd "$(git rev-parse --show-toplevel)"

MODE=diff; BASE=""
case "${1:-}" in
  --bank) MODE=bank; BASE="${2:?--bank needs a file}" ;;
  --control) MODE=control; BASE="${2:?--control needs a baseline}" ;;
  -h|--help|"") sed -n '2,15p' "$0"; exit 64 ;;
  *) BASE="$1" ;;
esac

WORK="$(mktemp -d -t vm-parity.XXXXXX)"
trap 'rm -rf "$WORK"' EXIT

outcomes() { # outcomes <parity.json> <engine> -> one {file, engine, status, vm_exit} per line
  jq -c --arg e "$2" '.[] | {file, engine: $e, status, vm_exit}' "$1"
}

if [ "$MODE" = control ]; then
  # Flip the first MATCH row; the diff below must notice.
  awk 'BEGIN{done=0} { if (!done && $0 ~ /"status":"MATCH"/) { sub(/"status":"MATCH"/, "\"status\":\"VM_RUNTIME\""); done=1 } print }' "$BASE" > "$WORK/now.jsonl"
else
  go build -o "$WORK/ailang" ./cmd/ailang
  go run ./scripts/verify_bytecode_parity.go --json --bin "$WORK/ailang" > "$WORK/bytecode.json"
  go run ./scripts/verify_bytecode_parity.go --json --strict --bin "$WORK/ailang" > "$WORK/strict.json"
  { outcomes "$WORK/bytecode.json" bytecode; outcomes "$WORK/strict.json" strict; } | sort > "$WORK/now.jsonl"
fi

if [ "$MODE" = bank ]; then
  mkdir -p "$(dirname "$BASE")"
  cp "$WORK/now.jsonl" "$BASE"
  echo "banked $(wc -l < "$BASE" | tr -d ' ') outcomes to $BASE"
  jq -rs 'group_by(.engine)[] | "\(.[0].engine): " + (group_by(.status) | map("\(.[0].status)=\(length)") | join(" "))' "$BASE"
  exit 0
fi

sort "$BASE" > "$WORK/base.jsonl"
sort "$WORK/now.jsonl" > "$WORK/now.sorted.jsonl"
if diff "$WORK/base.jsonl" "$WORK/now.sorted.jsonl" > "$WORK/diff.txt"; then
  echo "vm parity: unchanged ($(wc -l < "$WORK/base.jsonl" | tr -d ' ') outcomes)"
  exit 0
fi
echo "vm parity: OUTCOME CHANGED (< baseline, > now):"
cat "$WORK/diff.txt"
exit 1
