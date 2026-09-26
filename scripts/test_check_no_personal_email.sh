#!/usr/bin/env bash
# Arms for check_no_personal_email.sh. bash 3.2 safe; no network.
set -u
pass=0; fail=0
ck() { if [ "$2" = "$3" ]; then echo "  PASS $1"; pass=$((pass+1)); else echo "  FAIL $1 (got '$2' want '$3')"; fail=$((fail+1)); fi; }
ROOT="$(cd "$(dirname "$0")/.." && pwd)"

# A. the real repo is clean (the gate's live assertion)
bash "$ROOT/scripts/check_no_personal_email.sh" >/dev/null 2>&1
ck "A repo currently clean" "$?" "0"

# B–E run against a throwaway git repo so the arms are hermetic.
W=$(mktemp -d); cd "$W" || exit 1
git init -q .; mkdir -p design_docs scripts .claude/skills
cp "$ROOT/scripts/check_no_personal_email.sh" scripts/
git add -A >/dev/null 2>&1; git -c user.email=t@t -c user.name=t commit -qm init >/dev/null 2>&1

run() { bash scripts/check_no_personal_email.sh >/dev/null 2>&1; echo $?; }

# B. a personal address in a mission doc must FAIL (the whole point).
# The fixture is ASSEMBLED AT RUNTIME on purpose: a literal address in this file would be
# flagged by the very gate it tests, which is exactly what happened when this file was first
# committed — the gate went red on its own fixture the moment `git ls-files` began listing it.
FIX="someone@realdomain"; FIX="$FIX.com"
echo "provenance is the commit author $FIX" > design_docs/v1-mission.md
git add -A >/dev/null 2>&1
ck "B personal addr in mission doc -> fail" "$(run)" "1"

# C. a GitHub noreply is ALLOWED (identities may still be recorded)
echo 'author 3155884+SomeUser@users.noreply.github.com' > design_docs/v1-mission.md
git add -A >/dev/null 2>&1
ck "C noreply allowed" "$(run)" "0"

# D. reserved-TLD placeholders are allowed (RFC 2606/6761)
echo 'to: you@example.com and test@example.invalid' > design_docs/v1-mission.md
git add -A >/dev/null 2>&1
ck "D reserved placeholders allowed" "$(run)" "0"

# E. OUT OF SCOPE by design: functional/governance files are not policed
echo "maintainer $FIX" > SECURITY.md
echo "const owner = \"$FIX\"" > access_control.go
rm -f design_docs/v1-mission.md
git add -A >/dev/null 2>&1
ck "E governance/code out of scope" "$(run)" "0"

echo "  ---- $pass passed, $fail failed"
cd /; rm -rf "$W"
[ "$fail" -eq 0 ]
