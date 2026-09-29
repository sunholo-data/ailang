#!/bin/bash
# The mission-loop skills pi and codex read (.agents/skills) must equal the source
# (.claude/skills), and every skill's frontmatter description must parse as YAML.
# Both failures are silent in production: a stale copy runs old instructions, and an
# unquoted ": " in a plain-scalar description makes pi reject the whole frontmatter
# and skip the skill (180dd4ce9 fixed four such skills in .agents only; the .claude
# mission-control source kept the bug until 2026-09-29).
set -u
ROOT="$(cd "$(dirname "$0")/../.." && pwd)"
PASS=0; FAIL=0
ok() { PASS=$((PASS+1)); echo "  PASS: $1"; }
bad() { FAIL=$((FAIL+1)); echo "  FAIL: $1 ($2)"; }

out=$(/bin/bash "$ROOT/tools/launchd/sync-agents-skills.sh" --check 2>&1)
if [ $? -eq 0 ]; then ok "mission-*/sprint-* .agents copies equal .claude"; else bad "mission-*/sprint-* .agents copies equal .claude" "$out — run tools/launchd/sync-agents-skills.sh"; fi

# Negative control: the check must SEE a drifted copy.
tmp=$(mktemp -d); mkdir -p "$tmp/tools/launchd" "$tmp/.claude/skills" "$tmp/.agents/skills"
cp "$ROOT/tools/launchd/sync-agents-skills.sh" "$tmp/tools/launchd/"
for s in mission-control mission-brief mission-loop-change sprint-planner sprint-executor sprint-evaluator; do
  mkdir -p "$tmp/.claude/skills/$s" "$tmp/.agents/skills/$s"; echo x > "$tmp/.claude/skills/$s/SKILL.md"; echo x > "$tmp/.agents/skills/$s/SKILL.md"
done
echo drifted > "$tmp/.agents/skills/sprint-planner/SKILL.md"
if /bin/bash "$tmp/tools/launchd/sync-agents-skills.sh" --check >/dev/null 2>&1; then bad "negative control: a drifted copy is detected" "check passed on a drifted copy"; else ok "negative control: a drifted copy is detected"; fi
rm -rf "$tmp"

# Frontmatter: an UNQUOTED description containing ": " is invalid YAML.
badfm=""
for f in "$ROOT"/.claude/skills/*/SKILL.md "$ROOT"/.agents/skills/*/SKILL.md; do
  d=$(awk 'NR==1 && $0!="---"{exit} NR>1 && $0=="---"{exit} /^description: /{print; exit}' "$f")
  v=${d#description: }
  case "$v" in \'*|\"*|'') continue ;; esac
  case "$v" in *": "*) badfm="$badfm ${f#$ROOT/}" ;; esac
done
if [ -z "$badfm" ]; then ok "no skill description is an unquoted scalar containing ': '"; else bad "no skill description is an unquoted scalar containing ': '" "$badfm"; fi

echo ""
echo "==== $PASS passed, $FAIL failed ===="
[ "$FAIL" -eq 0 ]
