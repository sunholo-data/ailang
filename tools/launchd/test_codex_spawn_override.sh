#!/bin/bash
# Socket-free checks of driver arrays and both mission-control skill roots.
set -u
HERE=$(cd "$(dirname "$0")" && pwd)
ROOT=$(cd "$HERE/../.." && pwd)
. "${CODEX_SPAWN_HELPER:-$HERE/lib/codex-auth-profile.sh}"
flag='features.multi_agent_v2.expose_spawn_agent_model_overrides=false'
failures=0
for runtime in daemon ''; do
  AILANG_CODEX_RUNTIME=$runtime
  MC_CODEX_COMMAND=()
  found=0
  if mc_codex_command; then
    for ((i=0; i+1<${#MC_CODEX_COMMAND[@]}; i++)); do
      if [ "${MC_CODEX_COMMAND[$i]}" = -c ] && [ "${MC_CODEX_COMMAND[$((i+1))]}" = "$flag" ]; then found=1; fi
    done
  fi
  if [ "$found" -ne 1 ]; then
    echo "FAIL: runtime ${runtime:-legacy}: missing adjacent -c $flag"
    failures=$((failures+1))
  fi
done

skill_roots=${CODEX_SPAWN_SKILL_ROOTS-"$ROOT/.claude/skills/mission-control
$ROOT/.agents/skills/mission-control"}
launch_pattern='codex exec --model|[(] exec codex exec'
root_count=0
while IFS= read -r skill_root || [ -n "$skill_root" ]; do
  root_count=$((root_count+1))
  launches=0
  for file in "$skill_root"/resources/*.md; do
    [ -f "$file" ] && [ -r "$file" ] || continue
    count=$(awk -v pattern="$launch_pattern" '$0 ~ pattern { n++ } END { print n+0 }' "$file")
    launches=$((launches+count))
    if ! awk -v pattern="$launch_pattern" -v flag="-c $flag" '
      $0 ~ pattern && !index($0, flag) {
        printf "FAIL: %s:%d: missing spawn override flag\n", FILENAME, FNR
        unsafe=1
      }
      END { exit unsafe }
    ' "$file"; then failures=$((failures+1)); fi
  done
  echo "LAUNCH COUNT: $skill_root: $launches"
  if [ "$launches" -lt 2 ]; then
    echo "FAIL: $skill_root: instrument saw no launches (minimum 2; saw $launches)"
    failures=$((failures+1))
  fi
done <<< "$skill_roots"
if [ "$root_count" -eq 0 ]; then
  echo 'FAIL: instrument saw no launches (no skill roots)'
  failures=$((failures+1))
fi
echo "codex spawn override: $failures failures"
[ "$failures" -eq 0 ]
