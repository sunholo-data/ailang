#!/bin/bash
# test_pi_ext_args.sh — lib/pi-ext-args.sh: each pi extension loads exactly once.
set -u
HERE=$(cd "$(dirname "$0")" && pwd)
. "$HERE/lib/pi-ext-args.sh"
PASS=0; FAIL=0
ok(){ PASS=$((PASS+1)); echo "  PASS: $1"; }
bad(){ FAIL=$((FAIL+1)); echo "  FAIL: $1"; }
T=$(mktemp -d); trap 'rm -rf "$T"' EXIT

mkdir -p "$T/plain"
[ -z "$(_mc_pi_ext_args "$T/plain")" ] && ok "no .pi/extensions -> no flags (discovery unchanged: world, stapledon)" \
  || bad "no .pi/extensions must print nothing"

mkdir -p "$T/repo/.pi/extensions"; : > "$T/repo/.pi/extensions/a.ts"; : > "$T/repo/.pi/extensions/b.ts"; : > "$T/repo/.pi/extensions/README.md"
out=$(_mc_pi_ext_args "$T/repo")
[ "$(printf '%s\n' "$out" | head -1)" = "--no-extensions" ] && ok "repo extensions -> discovery off" || bad "first flag must be --no-extensions"
[ "$(printf '%s\n' "$out" | grep -c '^-e$')" = 2 ] && ok "one -e per .ts" || bad "expected 2 -e flags: $out"
printf '%s\n' "$out" | grep -q 'README' && bad "non-.ts files must not load" || ok "README not loaded"

mkdir -p "$T/empty/.pi/extensions"
[ "$(_mc_pi_ext_args "$T/empty")" = "--no-extensions" ] && ok "empty dir -> discovery off, nothing named" || bad "empty dir"

# Wiring: every pi launch site uses it (or, for the probe, disables discovery outright).
grep -q 'pi --mode json --no-session --no-tools --no-extensions' "$HERE/lib/lane-probe.sh" && ok "probe disables discovery" || bad "probe not wired"
grep -q '_mc_pi_ext_args "\$REPO"' "$HERE/mission-control.sh" && ok "pi controller wired" || bad "pi controller not wired"
grep -q '_mc_pi_ext_args "\$WORKDIR"' "$HERE/../../scripts/mission_pi_run.sh" && ok "role runner wired" || bad "role runner not wired"
echo "==== $PASS passed, $FAIL failed ===="
[ "$FAIL" -eq 0 ]
