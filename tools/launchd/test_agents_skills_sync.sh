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

# Check canonical rule 3(c); the equality check above covers the generated mirror.
# Collect wrapped prose, then inspect only inline commands between (c) and (d).
if awk '
  { text = text $0 "\n" }
  END {
    start = index(text, "**(c)**")
    if (!start) exit 1
    clause = substr(text, start + length("**(c)**"))
    stop = index(clause, "**(d)**")
    if (!stop) exit 1
    clause = substr(clause, 1, stop - 1)
    # BSD awk split() also breaks on newline, which would flip the inline-code parity.
    gsub(/\n/, " ", clause)
    n = split(clause, spans, "`")
    count = 0; valid = 1
    for (i = 2; i <= n; i += 2) {
      # The store scope prefix is checked by the scope arm below; strip it here.
      s = spans[i]; sub(/^env AILANG_STORAGE_MESSAGING=gcp AILANG_MESSAGES_PROJECT=ailang-multivac /, "", s)
      if (s ~ /^ailang messages read([[:space:]]|$)/) {
        count++
        if (s != "ailang messages read --peek --json <id>") valid = 0
      }
    }
    exit !(count == 1 && valid)
  }
' "$ROOT/.claude/skills/mission-control/resources/gate-5-retro.md"; then
  ok "Gate-5 rule 3(c) preserves unread status with flags before id"
else
  bad "Gate-5 rule 3(c) preserves unread status with flags before id" "expected sole inline command: ailang messages read --peek --json <id>; --peek preserves unread status and all flags must precede id"
fi

# Prescribed message commands must name the prod store: the local store refuses `controlplane`,
# and only a user dotfile made the sends work. The positive control proves the grep sees the sites.
cmds=$(grep -rnE 'ailang messages (send|read|ack|unack) ' "$ROOT/.claude/skills/mission-control" | grep -vE 'ailang messages (send|read|ack|unack) (--help|-h)')
unscoped=$(printf '%s\n' "$cmds" | grep -vE 'AILANG_STORAGE_MESSAGING=gcp AILANG_MESSAGES_PROJECT=ailang-multivac ailang messages|\$MSG ailang messages')
msg_bad=""
for f in $(grep -rlF '$MSG ailang messages' "$ROOT/.claude/skills/mission-control"); do
  grep -qF 'MSG="env AILANG_STORAGE_MESSAGING=gcp AILANG_MESSAGES_PROJECT=ailang-multivac"' "$f" || msg_bad="$msg_bad ${f#$ROOT/}"
done
if ! printf '%s\n' "$cmds" | grep -qE 'gate-5-retro\.md:.*send controlplane' || ! printf '%s\n' "$cmds" | grep -qE 'gate-3-route\.md:.*send approvals'; then
  bad "mission-control prescribed message commands are store-scoped" "positive control failed: grep no longer sees the controlplane or approvals send sites"
elif [ -n "$unscoped" ] || [ -n "$msg_bad" ]; then
  where=""
  while IFS= read -r l; do [ -n "$l" ] && where="$where $(printf '%s\n' "$l" | cut -d: -f1,2)"; done <<< "$unscoped"
  where=$(for w in $where; do printf '%s ' "${w#$ROOT/}"; done)
  bad "mission-control prescribed message commands are store-scoped" "unscoped: $where; \$MSG not defined in: $msg_bad"
else
  ok "mission-control prescribed message commands are store-scoped"
fi

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
