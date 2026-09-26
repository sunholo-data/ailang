#!/usr/bin/env bash
set -u
REPO_ROOT=$(cd "$(dirname "$0")/.." && pwd)
GATE="$REPO_ROOT/scripts/check_git_exec.sh"
TMP_DIR=$(mktemp -d) || exit 2
trap 'rm -rf "$TMP_DIR"' EXIT
FAILED=0; ARMS_RUN=0; ARMS_EXPECTED=8
pass(){ ARMS_RUN=$((ARMS_RUN+1)); printf '  PASS — %s\n' "$1"; }
fail(){ ARMS_RUN=$((ARMS_RUN+1)); FAILED=1; printf '  FAIL — %s\n' "$1"; }
mkroot(){ mkdir -p "$1/cmd" "$1/internal/gitexec"; printf 'package main\n' >"$1/cmd/main.go"; printf 'package gitexec\nimport "os/exec"\nfunc p(){ _,_=exec.LookPath("git") }\n' >"$1/internal/gitexec/gitexec.go"; }
baseline(){ : >"$1"; }
run(){ GIT_EXEC_ROOT="$1" GIT_EXEC_BASELINE="$2" GIT_EXEC_POSITIVE_FIXTURE="$3" bash "$GATE" >"$TMP_DIR/out" 2>&1; RC=$?; OUT=$(cat "$TMP_DIR/out"); }

R="$TMP_DIR/real"; mkroot "$R"; B="$TMP_DIR/base"; baseline "$B"; P="$REPO_ROOT/scripts/testdata/git_exec_gate_positive.txt"
run "$R" "$B" "$P"; [ "$RC" -eq 0 ] && pass S1 || fail "S1 real control rc=$RC: $OUT"
run "$R" "$B" "$TMP_DIR/missing"; [ "$RC" -eq 2 ] && printf '%s' "$OUT" | grep -q 'G1' && pass S2 || fail "S2 missing fixture: $OUT"
printf 'package fixture\n' >"$TMP_DIR/empty.txt"; run "$R" "$B" "$TMP_DIR/empty.txt"; [ "$RC" -eq 2 ] && printf '%s' "$OUT" | grep -q 'G2' && pass S2b || fail "S2b empty fixture: $OUT"
printf 'package main\nimport "os/exec"\nfunc f(){exec.Command("git")}\n' >"$R/cmd/add.go"; run "$R" "$B" "$P"; [ "$RC" -eq 1 ] && printf '%s' "$OUT" | grep -q 'absent from baseline' && pass S3b || fail "S3b absent baseline: $OUT"
printf 'cmd/add.go 0\n' >"$B"; run "$R" "$B" "$P"; [ "$RC" -eq 1 ] && printf '%s' "$OUT" | grep -q 'count increased' && pass S3 || fail "S3 increase: $OUT"
printf 'cmd/add.go 2\n' >"$B"; run "$R" "$B" "$P"; [ "$RC" -eq 1 ] && printf '%s' "$OUT" | grep -q 'tighten the baseline' && pass S4 || fail "S4 decrease: $OUT"
printf 'cmd/add.go 1\n' >"$B"; printf 'package main\nimport "os/exec"\nfunc p(){_,_=exec.LookPath("git")}\n' >"$R/cmd/look.go"; run "$R" "$B" "$P"; [ "$RC" -eq 1 ] && printf '%s' "$OUT" | grep -q 'outside internal/gitexec' && pass S5 || fail "S5 LookPath: $OUT"
rm "$R/cmd/look.go"; printf 'package main\nimport "os/exec"\nfunc g(){exec.Command(\n"git")}\n' >"$R/cmd/multi.go"; printf 'cmd/add.go 1\ncmd/multi.go 1\n' >"$B"; run "$R" "$B" "$P"; [ "$RC" -eq 1 ] && printf '%s' "$OUT" | grep -q 'AST/regex disagreement' && pass S6 || fail "S6 cross-check: $OUT"

[ "$ARMS_RUN" -eq "$ARMS_EXPECTED" ] || { FAILED=1; printf '  FAIL — %s of %s arms ran\n' "$ARMS_RUN" "$ARMS_EXPECTED"; }
[ "$FAILED" -eq 0 ] && { printf 'git exec gate self-test: OK (%s arms)\n' "$ARMS_RUN"; exit 0; }
printf 'git exec gate self-test: FAILED\n'; exit 1
