#!/usr/bin/env bash
# Arms for scripts/hooks/pre-push. No network, no real repo, no ailang build:
# the selection logic runs via --needs-gate, and the ref filter via stdin against a
# throwaway repo whose pushes never reach the gate (or reach it and must refuse).
set -u
HOOK="$(cd "$(dirname "$0")" && pwd)/pre-push"
W=$(mktemp -d "${TMPDIR:-/tmp}/ailang-prepush-test.XXXXXX") || exit 1
trap 'rm -rf "$W"' EXIT
cd "$W" || exit 1
pass=0; fail=0
ck() { if [ "$2" = "$3" ]; then echo "  PASS $1"; pass=$((pass+1)); else echo "  FAIL $1 (got '$2' want '$3')"; fail=$((fail+1)); fi; }

git init -q repo; cd repo || exit 1
git config user.email t@t; git config user.name t
mkdir -p std docs internal/types internal/pipeline tools
echo a > docs/a.md; echo x > std/list.ail; git add -A; git commit -qm init
BASE=$(git rev-parse HEAD)

commit() { mkdir -p "$(dirname "$1")"; echo "$RANDOM" >> "$1"; git add -A; git commit -qm "$1"; git rev-parse HEAD; }

H=$(commit docs/a.md);                 ck "docs-only change skips gate"          "$(bash "$HOOK" --needs-gate "$BASE" "$H")" no
H2=$(commit std/list.ail);             ck "std/ change needs gate"               "$(bash "$HOOK" --needs-gate "$H" "$H2")" yes
H3=$(commit internal/types/print.go);  ck "internal/types change needs gate"     "$(bash "$HOOK" --needs-gate "$H2" "$H3")" yes
H4=$(commit internal/pipeline/x.go);   ck "unrelated Go change skips gate"       "$(bash "$HOOK" --needs-gate "$H3" "$H4")" no
H5=$(commit .stdlib-golden/list.json); ck "golden change needs gate"             "$(bash "$HOOK" --needs-gate "$H4" "$H5")" yes
H6=$(commit tools/verify-stdlib.sh);   ck "gate script change needs gate"        "$(bash "$HOOK" --needs-gate "$H5" "$H6")" yes
ck "range spanning a std/ commit needs gate" "$(bash "$HOOK" --needs-gate "$BASE" "$H4")" yes
ck "unknown base with no origin/dev fails closed" "$(bash "$HOOK" --needs-gate "" "$H4")" yes

Z=0000000000000000000000000000000000000000
# Non-dev refs are never gated, even when std/ changed.
echo "refs/heads/x $H2 refs/heads/feature $BASE" | bash "$HOOK" origin url >/dev/null 2>&1
ck "push to a non-dev branch passes" "$?" 0
# Deleting dev is not gated.
echo "(delete) $Z refs/heads/dev $BASE" | bash "$HOOK" origin url >/dev/null 2>&1
ck "deleting dev passes" "$?" 0
# docs-only push to dev passes without building anything.
echo "refs/heads/dev $H refs/heads/dev $BASE" | bash "$HOOK" origin url >/dev/null 2>&1
ck "docs-only push to dev passes" "$?" 0
# A std/ push to dev reaches the gate; this repo has no cmd/ailang, so the build fails
# and the push must be REFUSED (setup failure is a refusal, not a pass).
out=$(echo "refs/heads/dev $H2 refs/heads/dev $H" | bash "$HOOK" origin url 2>&1); rc=$?
ck "std/ push to dev with unbuildable tree is refused" "$rc" 1
ck "refusal names the fix" "$(printf '%s' "$out" | grep -c 'make freeze-stdlib')" 1
ck "gate worktree cleaned up" "$(git worktree list | wc -l | tr -d ' ')" 1

# ci-quick gate selection (--needs-quick): Go, examples, changelogs, make files and the
# generated docs need it; docs, design docs and state files do not.
Q1=$(commit design_docs/x.md);          ck "design doc skips ci-quick"            "$(bash "$HOOK" --needs-quick "$H6" "$Q1")" no
Q2=$(commit .ailang/state/x.json);      ck "state file skips ci-quick"            "$(bash "$HOOK" --needs-quick "$Q1" "$Q2")" no
ck "docs-only change skips ci-quick"    "$(bash "$HOOK" --needs-quick "$BASE" "$H")" no
ck "std/ .ail change alone skips ci-quick" "$(bash "$HOOK" --needs-quick "$H" "$H2")" no
ck "Go change needs ci-quick"           "$(bash "$HOOK" --needs-quick "$H3" "$H4")" yes
Q3=$(commit examples/manifest.json);    ck "examples change needs ci-quick"       "$(bash "$HOOK" --needs-quick "$Q2" "$Q3")" yes
Q4=$(commit changelogs/v1-current.md);  ck "changelog change needs ci-quick"      "$(bash "$HOOK" --needs-quick "$Q3" "$Q4")" yes
Q5=$(commit docs/docs/reference/cli.md); ck "cli reference change needs ci-quick" "$(bash "$HOOK" --needs-quick "$Q4" "$Q5")" yes
ck "unknown base with no origin/dev fails closed (ci-quick)" "$(bash "$HOOK" --needs-quick "" "$Q1")" yes
# A Go-only push to dev reaches the ci-quick gate; the unbuildable tree must be REFUSED
# with the ci-quick fix named (not the stdlib one).
out=$(echo "refs/heads/dev $H4 refs/heads/dev $H3" | bash "$HOOK" origin url 2>&1); rc=$?
ck "Go push to dev with unbuildable tree is refused" "$rc" 1
ck "ci-quick refusal names make ci-quick" "$(printf '%s' "$out" | grep -c 'Reproduce and fix:   make ci-quick')" 1
ck "ci-quick refusal does not blame the stdlib" "$(printf '%s' "$out" | grep -c 'make freeze-stdlib')" 0
ck "ci-quick gate worktree cleaned up" "$(git worktree list | wc -l | tr -d ' ')" 1
# A design-doc-only push to dev passes without building anything.
echo "refs/heads/dev $Q1 refs/heads/dev $H6" | bash "$HOOK" origin url >/dev/null 2>&1
ck "design-doc push to dev passes" "$?" 0

echo "pre-push: $pass passed, $fail failed"
[ "$fail" -eq 0 ]
