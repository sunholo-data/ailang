#!/usr/bin/env bash
# Arms for check_doc_review.sh. bash 3.2 safe; no network.
#
# B-K run against a throwaway git repo: the gate reads `git ls-files`, so the
# fixtures must be a real repo with real tracked files, not a bare directory.
set -u
pass=0; fail=0
ck() { if [ "$2" = "$3" ]; then echo "  PASS $1"; pass=$((pass+1)); else echo "  FAIL $1 (got '$2' want '$3')"; fail=$((fail+1)); fi; }
ROOT="$(cd "$(dirname "$0")/.." && pwd)"
GATE="$ROOT/scripts/check_doc_review.sh"

# A. the real repo has no malformed dates (overdue pages only warn)
(cd "$ROOT" && bash "$GATE" >/dev/null 2>&1)
ck "A repo currently clean" "$?" "0"

W=$(mktemp -d); cd "$W" || exit 1
git init -q .
mkdir -p docs/docs/guides docs/docs/prompts
export DOC_REVIEW_TODAY=2026-10-05 DOC_REVIEW_MINIMUM=1

page() { printf -- '---\ntitle: %s\n%b---\n\n# %s\n' "$2" "$3" "$2" > "docs/docs/guides/$1.md"; }
run() { git add -A >/dev/null 2>&1; bash "$GATE" "$@" 2>&1; }

# B. a dated page is clean
page good Good 'reviewBy: 2027-01-05\n'
out=$(run); ck "B dated page rc" "$?" "0"
ck "B no warnings" "$(echo "$out" | grep -c WARN)" "0"

# C. overdue warns but does not fail; --list-overdue lists oldest due first
page late Late 'reviewed: 2026-01-01\nreviewBy: 2026-09-01\n'
page later Later 'reviewBy: 2026-06-01\n'
out=$(run); ck "C overdue rc" "$?" "0"
ck "C overdue warned" "$(echo "$out" | grep -c 'review was due')" "2"
ck "C list order" "$(run --list-overdue | tr '\n' ' ')" "docs/docs/guides/later.md docs/docs/guides/late.md "
rm docs/docs/guides/late.md docs/docs/guides/later.md

# D. due today is not overdue
page today Today 'reviewBy: 2026-10-05\n'
ck "D due today clean" "$(run | grep -c WARN)" "0"

# E. no reviewBy (with and without frontmatter) warns, rc 0
page bare Bare ''
printf '# No frontmatter\n' > docs/docs/guides/nofm.md
out=$(run); ck "E missing rc" "$?" "0"
ck "E missing warned" "$(echo "$out" | grep -c 'no reviewBy')" "2"
rm docs/docs/guides/bare.md docs/docs/guides/nofm.md

# F. malformed reviewBy fails
page bad Bad 'reviewBy: 2027-13-01\n'
run >/dev/null; ck "F malformed rc" "$?" "1"
rm docs/docs/guides/bad.md

# G. reviewed after reviewBy fails
page inv Inv 'reviewed: 2027-02-01\nreviewBy: 2027-01-01\n'
run >/dev/null; ck "G inverted rc" "$?" "1"
rm docs/docs/guides/inv.md

# H. `never` and quoted values parse
page rec Rec 'reviewBy: never\n'
page dq Dq 'reviewBy: "2027-01-01"\n'
page sq Sq "reviewBy: '2027-01-01'\n"
out=$(run); ck "H never/quoted rc" "$?" "0"
ck "H never/quoted clean" "$(echo "$out" | grep -c WARN)" "0"

# I. a regenerated page is out of scope; the hand-written prompts index is in
printf '# Generated\n' > docs/docs/prompts/v1.md
printf '# Index\n' > docs/docs/prompts/index.md
out=$(run)
ck "I generated skipped" "$(echo "$out" | grep -c 'prompts/v1.md')" "0"
ck "I index checked" "$(echo "$out" | grep -c 'prompts/index.md')" "1"

# J. --github emits annotations
ck "J annotation" "$(run --github | grep -c '^::warning file=docs/docs/prompts/index.md::')" "1"
rm docs/docs/prompts/index.md

# K. floor: too few pages is an instrument failure, not a pass
DOC_REVIEW_MINIMUM=500 run >/dev/null; ck "K floor rc" "$?" "2"

cd / && rm -rf "$W"
echo "check_doc_review self-test: $pass passed, $fail failed"
[ "$fail" -eq 0 ]
