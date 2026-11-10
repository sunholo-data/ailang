#!/usr/bin/env bash
# Arms for tools/ci/require_checks.sh (bash 3.2). No network: a fake `gh` on PATH serves fixture
# pages of the check-runs API (by page=N, and by poll number for pending -> success), and the
# override marker is read from tags in a throwaway repo. Each arm names the branch it guards;
# m-v1-release-gates §1.5 requires every arm to die when its branch is neutered.
set -u
GATE="$(cd "$(dirname "$0")" && pwd)/require_checks.sh"
LIST="$(cd "$(dirname "$0")" && pwd)/release_required_checks.txt"
T=$(mktemp -d "${TMPDIR:-/tmp}/require-checks-test.XXXXXX") || exit 1
T=$(cd "$T" && pwd -P) || exit 1
trap 'rm -rf "$T"' EXIT
pass=0; fail=0
ck() { if [ "$2" = "$3" ]; then echo "  ok   — $1"; pass=$((pass+1)); else echo "  FAIL — $1 (got '$2' want '$3')"; fail=$((fail+1)); fi; }
command -v jq >/dev/null || { echo "  FAIL — jq not on PATH (the gate needs it)"; exit 1; }

mkdir -p "$T/bin"
cat > "$T/bin/gh" <<'EOF'
#!/bin/bash
# fake gh: `gh api <path>` -> $FIX/[pollN/]pageP.json; counts polls (page=1 requests).
[ "$1" = api ] || exit 9
p=$(printf '%s' "$2" | sed -n 's/.*[?&]page=\([0-9][0-9]*\).*/\1/p'); [ -n "$p" ] || p=1
n=$(cat "$FIX/polls" 2>/dev/null || echo 0)
[ "$p" = 1 ] && { n=$((n+1)); echo "$n" > "$FIX/polls"; }
d="$FIX"; [ -d "$FIX/poll$n" ] && d="$FIX/poll$n"
[ -f "$d/page$p.json" ] || { echo "fake gh: no $d/page$p.json" >&2; exit 1; }
cat "$d/page$p.json"
EOF
chmod +x "$T/bin/gh"
export PATH="$T/bin:$PATH"
export REQUIRE_CHECKS_INTERVAL=0 REQUIRE_CHECKS_ATTEMPTS=3
unset GITHUB_ACTIONS GITHUB_OUTPUT GITHUB_REPOSITORY

# page FILE TOTAL — stdin lines "name|status|conclusion[|app-slug]"
page() {
	mkdir -p "$(dirname "$1")"
	jq -R -s --argjson total "$2" 'split("\n") | map(select(length > 0) | split("|")
		| {name: .[0], status: .[1], conclusion: (if (.[2] // "") == "" then null else .[2] end),
		   html_url: ("https://github.com/o/r/actions/runs/1/job/" + (.[0] | gsub(" "; "_"))),
		   app: {slug: (.[3] // "github-actions")}}) | {total_count: $total, check_runs: .}' > "$1"
}
newfix() { FIX="$T/fix$1"; export FIX; mkdir -p "$FIX"; }
SHA=0123456789abcdef0123456789abcdef01234567
gate() { OUT=$(bash "$GATE" "$@" 2>&1); RC=$?; }
N1="test"; N2="Build macos-latest"; N3="float determinism (arm64)"

echo "require_checks:"
newfix 1; page "$FIX/page1.json" 4 <<EOF
test|completed|success
Build macos-latest|completed|success
Build macos-latest|completed|success
float determinism (arm64)|completed|success
EOF
gate "$SHA" "$N1" "$N2" "$N3"
ck "all green: exit 0" "$RC" 0
ck "all green: says releasable" "$(printf '%s' "$OUT" | grep -c 'is releasable')" 1

newfix 2; page "$FIX/page1.json" 4 <<EOF
test|completed|success
Build macos-latest|completed|success
Build macos-latest|completed|failure
float determinism (arm64)|in_progress|
EOF
gate "$SHA" "$N1" "$N2" "$N3"
ck "every run per name: second same-name run red -> exit 1" "$RC" 1
ck "red: message names the check and conclusion" "$(printf '%s' "$OUT" | grep -c 'not releasable — Build macos-latest: failure')" 1
ck "red: stops at the first poll (fail fast)" "$(cat "$FIX/polls")" 1

newfix 3; page "$FIX/page1.json" 2 <<EOF
test|completed|success
test-windows|completed|cancelled
EOF
gate "$SHA" test test-windows
ck "cancelled is red" "$RC" 1

newfix 4; page "$FIX/poll1/page1.json" 1 <<EOF
test|in_progress|
EOF
page "$FIX/poll2/page1.json" 1 <<EOF
test|completed|success
EOF
gate "$SHA" test
ck "pending -> success: exit 0" "$RC" 0
ck "pending -> success: polled twice" "$(cat "$FIX/polls")" 2

newfix 5; page "$FIX/page1.json" 1 <<EOF
test|queued|
EOF
gate "$SHA" test
ck "pending forever: red after the last poll" "$RC" 1
ck "pending forever: polled every attempt" "$(cat "$FIX/polls")" 3

newfix 6; page "$FIX/page1.json" 1 <<EOF
test|completed|success
EOF
gate "$SHA" test lint
ck "missing name: red (fail closed)" "$RC" 1
ck "missing name: final verdict names it and the list file" "$(printf '%s' "$OUT" | grep -c 'poll(s): lint (missing).*release_required_checks.txt')" 1

newfix 7; page "$FIX/page1.json" 1 <<EOF
test|completed|success|some-other-app
EOF
gate "$SHA" test
ck "a non-Actions check-run with the name does not satisfy it" "$RC" 1

# >100 runs: the required one is on page 2.
newfix 8
i=0; : > "$T/p1"; while [ $i -lt 100 ]; do echo "noise $i|completed|success" >> "$T/p1"; i=$((i+1)); done
page "$FIX/page1.json" 102 < "$T/p1"
page "$FIX/page2.json" 102 <<EOF
noise 100|completed|success
Build macos-latest|completed|failure
EOF
gate "$SHA" "$N2"
ck "pagination: red run on page 2 is seen" "$RC:$(printf '%s' "$OUT" | grep -c 'Build macos-latest: failure')" "1:1"

# Default list: every name in release_required_checks.txt (spaces, parentheses, comments) green.
newfix 9
sed -e 's/[[:space:]]*#.*$//' -e 's/[[:space:]]*$//' "$LIST" | grep -v '^$' | sed 's/$/|completed|success/' > "$T/all"
page "$FIX/page1.json" "$(wc -l < "$T/all" | tr -d ' ')" < "$T/all"
gate "$SHA"
ck "default list: all listed names green -> exit 0" "$RC" 0
ck "default list: reads every listed name" "$(printf '%s' "$OUT" | grep -c '^  green ')" "$(wc -l < "$T/all" | tr -d ' ')"
grep -v '^test|' "$T/all" > "$T/most"
page "$FIX/page1.json" "$(wc -l < "$T/most" | tr -d ' ')" < "$T/most"
gate "$SHA"
ck "default list: one listed name missing -> red" "$RC" 1

gate "not-a-sha" test
ck "usage: bad SHA -> exit 2" "$RC" 2
gate "$SHA" ""
ck "empty required list: refused, never a vacuous pass" "$RC" 2

# ---- override marker, from tags in a throwaway repo ----
git init -q "$T/repo"; cd "$T/repo" || exit 1
git config user.email t@t; git config user.name t
echo a > a; git add a; git commit -qm init; RSHA=$(git rev-parse HEAD)
git tag -a v9.9.9 -m "release v9.9.9

release-gate-override: https://github.com/o/r/issues/42 macOS watchdog flake, see issue"
git tag -a v9.9.8 -m "release-gate-override: because I said so"
git tag -a v9.9.7 -m "plain release"
echo b > b; git add b; git commit -qm "x

release-gate-override: https://github.com/o/r/issues/1 smuggled in a commit message"
git tag v9.9.6
LSHA=$(git rev-parse HEAD)
newfix 10; page "$FIX/page1.json" 2 <<EOF
test|completed|success
Build macos-latest|completed|failure
EOF
gate --tag v9.9.9 "$RSHA" test "$N2"
ck "override in annotated tag: red check waived, exit 0" "$RC" 0
ck "override: waived check named" "$(printf '%s' "$OUT" | grep -c 'OVERRIDE (https://github.com/o/r/issues/42 macOS watchdog flake, see issue) — waived red checks: Build macos-latest')" 1
: > "$T/ghout"
rm -f "$FIX/polls"; OUT=$(GITHUB_OUTPUT="$T/ghout" bash "$GATE" --tag v9.9.9 "$RSHA" test "$N2" 2>&1)
ck "override: release-notes outputs written" "$(grep -cE '^override=https://github.com/o/r/issues/42 |^waived=Build macos-latest$' "$T/ghout")" 2
gate --tag v9.9.7 "$RSHA" test "$N2"
ck "annotated tag without a marker: red stays red" "$RC" 1
gate --tag v9.9.8 "$RSHA" test "$N2"
ck "malformed marker (no issue URL): exit 1" "$RC:$(printf '%s' "$OUT" | grep -c 'malformed override')" "1:1"
gate --tag v9.9.6 "$LSHA" test "$N2"
ck "lightweight tag: commit-message marker ignored, red stays red" "$RC:$(printf '%s' "$OUT" | grep -c 'lightweight tag')" "1:1"
gate --tag v9.9.9 "$LSHA" test "$N2"
ck "tag pointing at another commit: refused" "$RC:$(printf '%s' "$OUT" | grep -c 'refusing to judge one commit')" "1:1"
newfix 11; page "$FIX/page1.json" 1 <<EOF
Build macos-latest|completed|failure
EOF
gate --tag v9.9.9 "$RSHA" test "$N2"
ck "override does not waive a missing check" "$RC:$(printf '%s' "$OUT" | grep -c 'poll(s): test (missing)')" "1:1"

echo "  ---- $pass passed, $fail failed"
[ "$fail" -eq 0 ] && [ "$pass" -eq 27 ]
