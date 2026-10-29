#!/usr/bin/env bash
# Bash 3.2/BSD fixtures; LEDGER_SCRIPT_UT is a test-only disposable mutation seam.
set -uo pipefail
cd "$(dirname "$0")/.." || exit 1
SCRIPT_UT="${LEDGER_SCRIPT_UT:-$PWD/scripts/mission_ledger_provenance.sh}"
SCRATCH="$(mktemp -d "${TMPDIR:-/tmp}/ledger_provenance.XXXXXX")" || exit 1
trap 'rm -rf "$SCRATCH"' EXIT
# Isolate even when called directly, without the launchd suite environment wrapper.
for git_var in $(env | sed -n 's/^\(GIT_[A-Za-z0-9_]*\)=.*/\1/p'); do unset "$git_var"; done
export GIT_CONFIG_GLOBAL=/dev/null GIT_CONFIG_NOSYSTEM=1
export LC_ALL=C
pass=0; fail=0
ok() { echo "ok $arm $*"; pass=$((pass+1)); }
notok() { echo "not ok $arm $*"; fail=$((fail+1)); }
repo() {
  arm="$1"; mkdir -p "$SCRATCH/$arm"; cd "$SCRATCH/$arm" || exit 1
  git -c core.hooksPath=/dev/null init -q || exit 1
  file='charter with spaces.md'
}
commit() {
  local author="$1" committer="${2:-$1}"
  git -c core.hooksPath=/dev/null add -- . || exit 1
  GIT_AUTHOR_NAME="$author" GIT_AUTHOR_EMAIL=creator@fixture.invalid \
    GIT_COMMITTER_NAME="$committer" GIT_COMMITTER_EMAIL=committer@fixture.invalid \
    git -c user.name=Fixture -c user.email=fixture@fixture.invalid \
    -c commit.gpgsign=false -c core.hooksPath=/dev/null commit -qm fixture || exit 1
  sha="$(git rev-parse HEAD)" || exit 1
}
row() { printf '| %s | %s | question | evidence |\n' "$1" "$2" > "$file"; }
check() { # expected rc, verdict, flip, creation, then explicit instrument args
  local expected_rc="$1" verdict="$2" flip="$3" creation="$4" rc out
  shift 4
  /bin/bash "$SCRIPT_UT" "$@" > "$SCRATCH/out" 2> "$SCRATCH/err"; rc=$?
  out="$(cat "$SCRATCH/out")"
  if [ "$rc" -eq "$expected_rc" ] && \
     [ "$out" = "$expected_id verdict=$verdict flip=$flip creation=$creation" ] && \
     [ "$(wc -l < "$SCRATCH/out" | tr -d ' ')" = 1 ]; then
    ok "$verdict rc=$rc SHA controls"
  else
    # Do not echo captured identity data, even when probing a leaking mutant.
    notok "expected $verdict rc=$expected_rc with independent SHA controls; rc=$rc"
  fi
  if grep -Eq '@|creator|committer@|resolver@' "$SCRATCH/out" "$SCRATCH/err"; then
    notok 'identity leakage'
  else ok 'no identity leakage'; fi
  if [ "$expected_rc" -eq 0 ]; then
    [ ! -s "$SCRATCH/err" ] && ok 'empty stderr' || notok 'unexpected stderr'
  else
    [ -s "$SCRATCH/err" ] && ok 'failure diagnostic' || notok 'missing diagnostic'
  fi
}
expect() {
  expected_id=D-1
  check "$@" --id D-1 --file "$file"
}

# A1 kills old -S occurrence-count / creation classification.
repo A1; row D-1 OPEN; commit sunholo-voight-kampff; creation=$sha
row D-1 'RESOLVED attended'; commit Human; flip=$sha
old="$(git log -1 --format=%H -S'| D-1 |' -- "$file")"
[ "$old" = "$creation" ] && [ "$flip" != "$creation" ] && ok 'old pickaxe selects creation' || notok 'old pickaxe control'
expect 0 attended "$flip" "$creation"

# A2 kills always-attended / creation-author selection; default bot alias.
repo A2; row D-1 OPEN; commit Human; creation=$sha
row D-1 RESOLVED; commit 'Voight-Kampff (bot)'; flip=$sha
expect 0 fleet "$flip" "$creation"

# A3 kills author-before-committer / ignored squash ambiguity, including aliases.
repo A3; row D-1 OPEN; commit Human; creation=$sha
row D-1 RESOLVED; commit sunholo-voight-kampff GitHub; flip=$sha
expect 0 undetermined:squash-merge "$flip" "$creation"
for committer in 'gItHuB[BoT]' web-flow; do
  row D-1 OPEN; commit Human
  row D-1 RESOLVED; commit sunholo-voight-kampff "$committer"
  # A later reopen + re-resolve is the current resolution, so its flip is selected.
  expect 0 undetermined:squash-merge "$sha" "$creation"
done
# Independent first-flip fixtures exercise the other committer aliases.
for committer in 'gItHuB[BoT]' web-flow; do
  repo "A3-${committer}"; arm=A3; row D-1 OPEN; commit Human; creation=$sha
  row D-1 RESOLVED; commit Human "$committer"; expect 0 undetermined:squash-merge "$sha" "$creation"
done

# A4 kills OPEN-parent requirement and omitted root additions.
repo A4; echo '# Charter' > "$file"; commit Human
row D-1 RESOLVED; commit Human; expect 0 attended "$sha" "$sha"
repo A4-root; arm=A4; row D-1 RESOLVED; commit Human; expect 0 attended "$sha" "$sha"

# A14 kills earliest-transition selection: human resolves, human reopens, fleet
# re-resolves directly. The current resolution is fleet-authored (iteration 35 judge N1).
repo A14; row D-1 OPEN; commit sunholo-voight-kampff; creation=$sha
row D-1 RESOLVED; commit Human
row D-1 OPEN; commit Human
row D-1 RESOLVED; commit 'Voight-Kampff (bot)'; flip=$sha
expect 0 fleet "$flip" "$creation"

# A5 kills latest -G / -1 --reverse and RESOLVED reword-as-flip.
repo A5; row D-1 OPEN; commit sunholo-voight-kampff; creation=$sha
row D-1 'RESOLVED original'; commit Human; flip=$sha
row D-1 'RESOLVED fleet reword'; commit sunholo-voight-kampff
expect 0 attended "$flip" "$creation"

# An ordinary merge inherits an established resolution; it is not a new flip.
repo A5-merge; arm=A5; row D-1 OPEN; commit Human; creation=$sha
row D-1 RESOLVED; commit Human; flip=$sha
merge_sha="$(printf 'inherited merge\n' | git -c user.name=Human -c user.email=merge@fixture.invalid \
  -c commit.gpgsign=false -c core.hooksPath=/dev/null commit-tree "$(git write-tree)" -p "$flip" -p "$creation")" || exit 1
git -c core.hooksPath=/dev/null update-ref HEAD "$merge_sha" || exit 1
expect 0 attended "$flip" "$creation"

# A6 kills any-edit-as-resolution / fabricated flip.
repo A6; row D-1 OPEN; commit sunholo-voight-kampff; creation=$sha
row D-1 'OPEN more prose'; commit sunholo-voight-kampff
expect 0 never-resolved none "$creation"

# A7 kills missing-row-as-never-resolved.
repo A7; row D-2 OPEN; commit Human
expect 2 instrument-failure none none

# A8 kills swallowed file/git errors.
repo A8; row D-1 OPEN; commit Human; file=missing.md
expect 2 instrument-failure none none
# A tracked but unreadable-as-file path (directory) is invalid too.
file=.; expect 2 instrument-failure none none

# Explicit git failure, including identity-bearing raw diagnostics, must be suppressed.
repo A8-git-error; arm=A8; row D-1 OPEN; commit Human
real_git="$(command -v git)"; mkdir -p "$SCRATCH/git-stub"
cat > "$SCRATCH/git-stub/git" <<'STUB'
#!/bin/bash
if [ "$1" = log ]; then echo 'failed for creator@fixture.invalid' >&2; exit 1; fi
exec "$LEDGER_REAL_GIT" "$@"
STUB
chmod +x "$SCRATCH/git-stub/git"
export LEDGER_REAL_GIT="$real_git"
saved_path=$PATH; PATH="$SCRATCH/git-stub:$PATH"
expect 2 instrument-failure none none
PATH=$saved_path

# A9 kills removed-only -G hit selection (especially latest selection).
repo A9; row D-1 OPEN; commit sunholo-voight-kampff; creation=$sha
row D-1 RESOLVED; commit Human; flip=$sha
row D-1 OPEN; commit sunholo-voight-kampff
expect 0 attended "$flip" "$creation"

# A10 kills ID-prefix matching and fixed-space regex; closing status pipe matters.
repo A10
printf '  |D-1\t| OPEN | question | evidence |\n|  D-13  | OPEN | question | evidence |\n' > "$file"
commit Human; creation=$sha
printf '  |D-1\t| OPEN | question | evidence |\n|  D-13  |\tRESOLVED\t| question| evidence |\n' > "$file"
commit Human; flip=$sha
expect 0 never-resolved none "$creation"
expected_id=D-13; check 0 attended "$flip" "$creation" --id D-13 --file "$file"
# RESOLVED in the question/evidence or a status prefix is not a literal resolution.
printf '| D-1 | RESOLVEDish | RESOLVED question | RESOLVED |\n| D-13 | RESOLVED | question | evidence |\n' > "$file"
commit Human; expect 0 never-resolved none "$creation"

# A11 kills ignored override / case-sensitive identity / empty-pattern acceptance.
repo A11; row D-1 OPEN; commit Human; creation=$sha
row D-1 RESOLVED; commit 'MiSsIoN BoT'; flip=$sha
expected_id=D-1
check 0 fleet "$flip" "$creation" --id D-1 --file "$file" --self 'mission bot'
check 2 instrument-failure none none --id D-1 --file "$file" --self ''
check 2 instrument-failure none none --id D-1 --file "$file" --self '['
check 0 attended "$flip" "$creation" --id D-1 --file "$file" --self 'bot'

# A12 kills malformed-input / dirty-target / shallow-history acceptance.
repo A12; row D-1 OPEN; commit Human
expected_id=D-1
check 2 instrument-failure none none --id D-1 --file
expected_id=D-unknown
check 2 instrument-failure none none --id 'D-1.*' --file "$file"
check 2 instrument-failure none none --id $'D-1\nD-2' --file "$file"
check 2 instrument-failure none none --file "$file"
expected_id=D-1
check 2 instrument-failure none none --id D-1 --file "$file" --bogus
check 2 instrument-failure none none --id D-1 --id D-1 --file "$file"
row D-1 RESOLVED; expect 2 instrument-failure none none
commit Human
printf '| D-1 | another | OPEN | evidence |\n' >> "$file"; commit Human
expect 2 instrument-failure none none
# Actual shallow clone, not a fabricated .git/shallow sentinel.
repo A12-source; row D-1 OPEN; commit Human
row D-1 RESOLVED; commit Human
source_repo=$PWD
cd "$SCRATCH" || exit 1
git -c core.hooksPath=/dev/null clone -q --depth 1 "file://$source_repo" A12-shallow || exit 1
cd A12-shallow || exit 1; arm=A12
expect 2 instrument-failure none none
# Reused ID and historical rename cannot invent a unique creation control.
repo A12-reuse; arm=A12; row D-1 OPEN; commit Human; creation=$sha
echo '# no row' > "$file"; commit Human
row D-1 RESOLVED; commit Human
expect 2 instrument-failure none "$creation"
repo A12-rename; arm=A12; row D-1 OPEN; commit Human; creation=$sha
mv "$file" old.md; commit Human
mv old.md "$file"; commit Human
expect 2 instrument-failure none "$creation"
# Conservative failure on a merge that introduces the resolution.
repo A12-merge; arm=A12; row D-1 OPEN; commit Human; creation=$sha
main_branch="$(git symbolic-ref --short HEAD)"
git -c core.hooksPath=/dev/null checkout -qb side || exit 1
echo side > side.txt; commit Human
side=$sha
git -c core.hooksPath=/dev/null checkout -q "$main_branch" || exit 1
echo main > main.txt; commit Human
row D-1 RESOLVED
git -c core.hooksPath=/dev/null add -- . || exit 1
# commit-tree permits a literal two-parent fixture without invoking merge hooks.
merge_sha="$(printf 'merge fixture\n' | git -c user.name=Human -c user.email=merge@fixture.invalid \
  -c commit.gpgsign=false -c core.hooksPath=/dev/null commit-tree "$(git write-tree)" -p "$sha" -p "$side")" || exit 1
git -c core.hooksPath=/dev/null update-ref HEAD "$merge_sha" || exit 1
expect 2 instrument-failure none "$creation"

# A13 kills %ae / raw identity diagnostics; distinct reserved resolver address.
repo A13; row D-1 OPEN; commit Human; creation=$sha
row D-1 RESOLVED
git -c core.hooksPath=/dev/null add -- . || exit 1
GIT_AUTHOR_NAME=Human GIT_AUTHOR_EMAIL=resolver@fixture.invalid \
  GIT_COMMITTER_NAME=Human GIT_COMMITTER_EMAIL=resolver-committer@fixture.invalid \
  git -c user.name=Fixture -c user.email=fixture@fixture.invalid -c commit.gpgsign=false \
  -c core.hooksPath=/dev/null commit -qm resolution || exit 1
flip="$(git rev-parse HEAD)"
expect 0 attended "$flip" "$creation"

echo "ledger provenance: $pass passed, $fail failed"
[ "$fail" -eq 0 ]
