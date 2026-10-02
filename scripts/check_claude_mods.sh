#!/usr/bin/env bash
# check_claude_mods.sh — validate, type-check and test every Claude Code mod
# under tools/claude-mods/ (M-CLAUDE-CODE-MODS).
#
#   scripts/check_claude_mods.sh            # all mods
#   scripts/check_claude_mods.sh ailang-lens
#
# Needs the `claude` CLI. Type-checking needs the engine's declarations: set
# CLAUDE_CODE_TYPES to a claude-code.d.ts, or the script uses the one the
# engine wrote beside a loaded mod (.claude-plugin/types/claude-code/index.d.ts)
# or the newest one the plugin-authoring skill laid under the temp dir.
#
# `claude plugin test` is server-gated per process: when it reports the
# rollout switch off, the tests are reported as NOT RUN, never as passed.
set -uo pipefail

ROOT="$(cd "$(dirname "$0")/.." && pwd)"
MODS_DIR="$ROOT/tools/claude-mods"

if ! command -v claude >/dev/null 2>&1; then
  echo "check_claude_mods: claude CLI not found; skipping" >&2
  exit 0
fi

find_types() {
  if [ -n "${CLAUDE_CODE_TYPES:-}" ] && [ -f "$CLAUDE_CODE_TYPES" ]; then
    echo "$CLAUDE_CODE_TYPES"; return
  fi
  local laid
  laid=$(ls "$MODS_DIR"/*/.claude-plugin/types/claude-code/index.d.ts 2>/dev/null | head -1)
  if [ -n "$laid" ]; then echo "$laid"; return; fi
  ls -t "${TMPDIR:-/tmp}"/../claude-*/bundled-skills/*/*/plugin-authoring/types/claude-code.d.ts \
    /private/tmp/claude-*/bundled-skills/*/*/plugin-authoring/types/claude-code.d.ts 2>/dev/null | head -1
}

TYPES="$(find_types)"
fail=0
mods=("$@")
[ ${#mods[@]} -eq 0 ] && mods=($(ls "$MODS_DIR" | grep -v '\.md$'))

for mod in "${mods[@]}"; do
  dir="$MODS_DIR/$mod"
  [ -f "$dir/.claude-plugin/plugin.json" ] || continue
  echo "== $mod"

  if claude plugin validate "$dir" >/tmp/ccm-validate.$$ 2>&1; then
    echo "  validate: ok"
  else
    echo "  validate: FAILED"; sed 's/^/    /' /tmp/ccm-validate.$$; fail=1
  fi

  if [ -n "$TYPES" ]; then
    tmp=$(mktemp -d)
    cat >"$tmp/tsconfig.json" <<EOF
{ "compilerOptions": { "target": "es2023", "lib": ["es2023"], "types": [],
    "module": "esnext", "moduleResolution": "bundler", "strict": true,
    "noUncheckedIndexedAccess": true, "noEmit": true, "skipLibCheck": true,
    "jsx": "react", "jsxFactory": "h", "jsxFragmentFactory": "Fragment" },
  "include": ["$TYPES", "$dir/hooks", "$dir/types", "$dir/tests"] }
EOF
    if npx -y -p typescript@5.6 tsc -p "$tmp" >"$tmp/out" 2>&1; then
      echo "  tsc: ok"
    else
      echo "  tsc: FAILED"; sed 's/^/    /' "$tmp/out" | head -30; fail=1
    fi
    rm -rf "$tmp"
  else
    echo "  tsc: NOT RUN (no claude-code.d.ts found; set CLAUDE_CODE_TYPES)"
  fi

  if ls "$dir"/tests/*.test.ts* >/dev/null 2>&1; then
    out=$(claude plugin test "$dir" 2>&1); code=$?
    if echo "$out" | grep -q "rollout switch served off"; then
      echo "  test: NOT RUN (rollout switch off in this process)"
    elif [ $code -eq 0 ]; then
      echo "  test: ok ($(echo "$out" | grep -Eo '[0-9]+ pass' | head -1))"
    else
      echo "  test: FAILED"; echo "$out" | tail -25 | sed 's/^/    /'; fail=1
    fi
  fi
done

rm -f /tmp/ccm-validate.$$
exit $fail
