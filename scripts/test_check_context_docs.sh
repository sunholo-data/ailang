#!/usr/bin/env bash
# Arms for check_context_docs.sh. bash 3.2 safe; no network.
#
# B-F run against a throwaway git repo: the gate reads `git ls-files`, so the
# fixtures must be a real repo with real tracked files, not a bare directory.
set -u
pass=0; fail=0
ck() { if [ "$2" = "$3" ]; then echo "  PASS $1"; pass=$((pass+1)); else echo "  FAIL $1 (got '$2' want '$3')"; fail=$((fail+1)); fi; }
ROOT="$(cd "$(dirname "$0")/.." && pwd)"

# A. the real repo passes (the gate's live assertion)
bash "$ROOT/scripts/check_context_docs.sh" >/dev/null 2>&1
ck "A repo currently clean" "$?" "0"

W=$(mktemp -d); cd "$W" || exit 1
git init -q .
mkdir -p scripts .claude/rules .claude/skills/demo internal/parser
cp "$ROOT/scripts/check_context_docs.sh" scripts/
echo "package parser" > internal/parser/p.go
echo "# Root" > CLAUDE.md
: > scripts/context_docs_baseline.txt

# A well-formed scoped rule, reused as the clean control between arms.
write_good_rule() {
	printf -- '---\npaths:\n  - "internal/parser/**"\n---\n\n# Good\n' > .claude/rules/good.md
}
write_good_rule
printf -- '---\nname: demo\ndescription: demo\n---\n\n# Demo\n' > .claude/skills/demo/SKILL.md
git add -A >/dev/null 2>&1

run() { git add -A >/dev/null 2>&1; bash scripts/check_context_docs.sh >/dev/null 2>&1; echo $?; }

ck "B well-formed fixture passes" "$(run)" "0"

# C. THE LIVE BUG: a paths glob matching nothing must fail. This is the failure
# mode that produced no symptom at all -- ailang-syntax.md scoped to `stdlib/**`
# on a tree with `std/`, so the rule simply stopped loading.
printf -- '---\npaths:\n  - "stdlib/**"\n---\n\n# Stale\n' > .claude/rules/stale.md
ck "C paths glob matching nothing -> fail" "$(run)" "1"
rm -f .claude/rules/stale.md

# D. an unscoped rule is allowed ONLY with a stated reason.
printf -- '# Unscoped\n\nbody\n' > .claude/rules/unscoped.md
ck "D unscoped rule without marker -> fail" "$(run)" "1"
printf -- '<!-- always-on: needed before paths are known -->\n\n# Unscoped\n' > .claude/rules/unscoped.md
ck "E unscoped rule WITH marker -> pass" "$(run)" "0"
rm -f .claude/rules/unscoped.md

# F. mid-segment globs must be accepted -- `cmd/ailang/observatory*.go` is a real
# live scope, and a naive prefix test reports it stale.
mkdir -p cmd/ailang; echo "package main" > cmd/ailang/observatory_seed.go
printf -- '---\npaths:\n  - "cmd/ailang/observatory*.go"\n---\n\n# Mid\n' > .claude/rules/mid.md
ck "F mid-segment glob accepted" "$(run)" "0"
rm -f .claude/rules/mid.md

# G. oversize SKILL.md fails, and the baseline grandfathers it.
awk 'BEGIN{print "---"; print "name: demo"; print "description: demo"; print "---";
     for(i=0;i<600;i++) print "line"}' > .claude/skills/demo/SKILL.md
ck "G SKILL.md over cap -> fail" "$(run)" "1"
echo ".claude/skills/demo/SKILL.md 604" > scripts/context_docs_baseline.txt
ck "H baselined oversize doc -> pass" "$(run)" "0"

# I. a baselined doc may shrink but never grow (the ratchet).
awk 'BEGIN{print "---"; print "name: demo"; print "description: demo"; print "---";
     for(i=0;i<700;i++) print "line"}' > .claude/skills/demo/SKILL.md
ck "I baselined doc grows -> fail" "$(run)" "1"

# J. once back under the cap the exemption must be REMOVED, not left to rot.
printf -- '---\nname: demo\ndescription: demo\n---\n\n# Demo\n' > .claude/skills/demo/SKILL.md
ck "J stale baseline entry -> fail" "$(run)" "1"
: > scripts/context_docs_baseline.txt
ck "K baseline cleaned -> pass" "$(run)" "0"

# L. a broken relative link in a rule fails: a dead pointer is a missing fact.
printf -- '---\npaths:\n  - "internal/parser/**"\n---\n\nSee [x](../../docs/gone.md).\n' > .claude/rules/good.md
ck "L broken relative link -> fail" "$(run)" "1"
write_good_rule
ck "M links resolve -> pass" "$(run)" "0"

# N. the gate must refuse to pass vacuously when there is nothing to check.
rm -f .claude/rules/*.md
ck "N no rules at all -> fail (not vacuous pass)" "$(run)" "1"

cd /; rm -rf "$W"
echo "  ---"
echo "  $pass passed, $fail failed"
[ "$fail" -eq 0 ] || exit 1
