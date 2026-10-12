#!/bin/bash
# shellcheck disable=SC1090,SC2015,SC2016,SC2030,SC2031,SC2034
# test_codex_controller_env.sh — a codex controller must receive the driver's role env + scope
# guard through `-c shell_environment_policy.set.*` (lib/codex-env-args.sh), and no secrets.
# Ticket agent-tool:mission-role-pins-unavailable. bash 3.2; sourced helper, never `. <(…)`.
set -u
HERE=$(cd "$(dirname "$0")" && pwd)
DRIVER="$HERE/mission-control.sh"
HELPER="$HERE/lib/codex-env-args.sh"
RESOLVER="$HERE/resolve-role-spawn.sh"
PASS=0; FAIL=0; SKIP=0
ok(){ PASS=$((PASS+1)); echo "  PASS: $1"; }
bad(){ FAIL=$((FAIL+1)); echo "  FAIL: $1"; }
skip(){ SKIP=$((SKIP+1)); echo "  SKIP $1"; echo "SKIP $1" >&2; }
T=$(mktemp -d); trap 'rm -rf "$T"' EXIT

# The real call-site block, closed with `fi` (the markers sit inside the if/elif chain).
awk '/# --- CODEX CONTROLLER EXEC START ---/,/# --- CODEX CONTROLLER EXEC END ---/' "$DRIVER" > "$T/block.sh"
echo "fi" >> "$T/block.sh"
[ "$(wc -l < "$T/block.sh")" -gt 4 ] && ok "call-site markers found in the real driver" || bad "CODEX CONTROLLER EXEC markers missing from driver"

mkdir -p "$T/bin"
cat > "$T/bin/codex" <<'STUB'
#!/bin/bash
printf '%s\0' "$@" > "$ARGV_OUT"
STUB
chmod +x "$T/bin/codex"

# run_block HELPER BLOCK [VAR=val ...] — runs the block under set -u in a clean env; argv -> $T/argv, log -> $T/log
run_block() {
  local helper="$1" block="$2"; shift 2
  : > "$T/log"; rm -f "$T/argv"
  env -i HOME="$T/home" PATH="$T/bin:/usr/bin:/bin" ARGV_OUT="$T/argv" LOG="$T/log" CODEX_HOME="$T/nocodex" \
    CONTROLLER_PROVIDER=codex MODEL=m REPO=/r PROMPT=PROMPTX RB_HELPER="$helper" RB_BLOCK="$block" "$@" \
    RB_AUTH="$HERE/lib/codex-auth-profile.sh" /bin/bash -c 'set -u; . "$RB_HELPER"; . "$RB_AUTH"; . "$RB_BLOCK"; wait'
}
argv_lines() { tr '\0' '\n' < "$T/argv"; }
mkdir -p "$T/home"

# --- (a) role vars reach argv, before the prompt
run_block "$HELPER" "$T/block.sh" MISSION_NAME=fleet MISSION_DESIGNER_MODEL=claude:claude-opus-5-5 \
  MISSION_EVALUATOR_MODEL=sonnet MISSION_OVER_RATION=0 AILANG_MESSAGES_PROJECT=ailang-multivac AILANG_DRIVER_PINNED=abc123
a=$(argv_lines)
miss=""
for n in MISSION_NAME MISSION_DESIGNER_MODEL MISSION_EVALUATOR_MODEL MISSION_OVER_RATION AILANG_MESSAGES_PROJECT AILANG_DRIVER_PINNED; do
  printf '%s\n' "$a" | grep -q "^shell_environment_policy\.set\.$n=" || miss="$miss $n"
done
[ -z "$miss" ] && ok "argv-carries-mission-vars" || bad "argv-carries-mission-vars: missing:$miss"
[ "$(printf '%s\n' "$a" | tail -1)" = PROMPTX ] && [ "$(printf '%s\n' "$a" | head -1)" = exec ] && ok "-c pairs sit between exec and the prompt positional" || bad "argv shape: $a"

# --- scope guard
run_block "$HELPER" "$T/block.sh" GIT_CONFIG_COUNT=2 GIT_CONFIG_KEY_0=http.extraHeader \
  "GIT_CONFIG_VALUE_0=Authorization: Bearer x" GIT_CONFIG_KEY_1=core.hooksPath GIT_CONFIG_VALUE_1=/g MISSION_NAME=fleet
a=$(argv_lines)
g=1
for want in 'set.GIT_CONFIG_COUNT="1"' 'set.GIT_CONFIG_KEY_0="core.hooksPath"' 'set.GIT_CONFIG_VALUE_0="/g"'; do
  printf '%s\n' "$a" | grep -qF "$want" || g=0
done
[ "$g" = 1 ] && ok "argv-scope-guard-forwarded (renumbered to index 0)" || bad "scope guard not forwarded: $a"
printf '%s\n' "$a" | grep -q 'Bearer\|extraHeader' && bad "other GIT_CONFIG entries leaked into argv" || ok "other GIT_CONFIG entries (credential-bearing) stay behind"

# --- secrets denied
run_block "$HELPER" "$T/block.sh" FOO_API_KEY=sekretFOO MISSION_TEST_TOKEN=sekretTOK AILANG_REGISTRY_API_KEY=sekretREG \
  MISSION_X_SECRET=sekretSEC MISSION_AUTH_COOKIE=sekretCOO MISSION_NAME=fleet
raw=$(tr '\0' ' ' < "$T/argv")
leak=0
for s in FOO_API_KEY MISSION_TEST_TOKEN AILANG_REGISTRY_API_KEY MISSION_X_SECRET MISSION_AUTH_COOKIE sekretFOO sekretTOK sekretREG sekretSEC sekretCOO; do
  case "$raw" in *"$s"*) leak=1 ;; esac
done
[ "$leak" = 0 ] && ok "secret-names-denied: no secret name or value in argv" || bad "secret leaked into argv: $raw"
summ=$(grep '^codex-env:' "$T/log")
dn=1; for n in MISSION_TEST_TOKEN MISSION_X_SECRET MISSION_AUTH_COOKIE; do case "$summ" in *denied=*"$n"*) ;; *) dn=0 ;; esac; done
[ "$dn" = 1 ] && ok "summary lists denied MISSION_* names" || bad "summary: $summ"
case "$summ" in *sekret*) bad "summary printed a value" ;; *) ok "summary is names-only" ;; esac

# --- empty env under set -u
run_block "$HELPER" "$T/block.sh"; rc=$?
[ "$rc" = 0 ] && [ "$(argv_lines | head -1)" = exec ] && ok "empty-env-set-u-safe (stub invoked, rc 0)" || bad "empty env: rc=$rc argv=$(argv_lines | tr '\n' ' ')"

# --- toml encoder unit
( . "$HELPER"
  hostile=$'a`b"c\\d$HOME\ne\tf\'g→h**'
  want='"a`b\"c\\d$HOME\ne\tf'"'"'g→h**"'
  got=$(mc_toml_basic_string "$hostile")
  [ "$got" = "$want" ] || { echo "  encode got=$got want=$want" >&2; exit 1; }
  mc_toml_basic_string $'x\001y' >/dev/null && exit 2
  exit 0 ); rc=$?
[ "$rc" = 0 ] && ok "toml-encode-unit" || bad "toml-encode-unit rc=$rc"
run_block "$HELPER" "$T/block.sh" "MISSION_BADCTL=$(printf 'x\001y')" MISSION_NAME=fleet
grep '^codex-env:' "$T/log" | grep -q 'skipped=.*MISSION_BADCTL' && argv_lines | grep -q BADCTL \
  && bad "control-char var forwarded" || { grep '^codex-env:' "$T/log" | grep -q 'skipped=.*MISSION_BADCTL' && ok "control-char var lands under skipped=" || bad "skipped= missing"; }

# --- include_only tripwire
mkdir -p "$T/ch"; printf '[shell_environment_policy]\ninclude_only = ["HOME"]\n' > "$T/ch/config.toml"
: > "$T/log"; ( CODEX_HOME="$T/ch"; export MISSION_NAME=x; . "$HELPER"; mc_codex_env_args 2>>"$T/log" )
grep -q 'WARNING include_only-present' "$T/log" && ok "include_only tripwire warns" || bad "include_only tripwire silent"

# --- e2e arms (real codex sandbox)
bounded() { # bounded SECONDS cmd... ; output to stdout
  local lim="$1" dl; shift; dl=$(( $(date +%s) + lim ))
  "$@" > "$T/b.out" 2> "$T/b.err" & local p=$!
  while kill -0 "$p" 2>/dev/null; do
    [ "$(date +%s)" -ge "$dl" ] && { kill "$p" 2>/dev/null; echo "bounded: timeout" >&2; return 124; }
    sleep 1
  done
  wait "$p"; local rc=$?; cat "$T/b.out"; return $rc
}
if command -v codex >/dev/null 2>&1; then
  e2e_rt() {
    . "$HELPER"
    export MISSION_RT=$'a`b"c\\d$HOME\ne\tf\'g→h**\n' MISSION_NAME=fleet MISSION_EXECUTOR_MODEL=claude:claude-sonnet-5-5
    mc_codex_env_args 2>/dev/null
    codex sandbox "${MC_CODEX_ENV_ARGS[@]}" -- /usr/bin/printenv MISSION_RT; echo x
  }
  out=$(bounded 120 e2e_rt; echo "rc=$?")
  want=$'a`b"c\\d$HOME\ne\tf\'g→h**\n'
  case "$out" in "$want"$'\n'x*) ok "roundtrip-codex-e2e: hostile value byte-exact through codex" ;; *) bad "roundtrip mismatch: $(printf '%s' "$out" | od -c | head -5)" ;; esac
  e2e_names() {
    . "$HELPER"; export MISSION_NAME=fleet MISSION_DESIGNER_MODEL=claude:claude-opus-5-5
    mc_codex_env_args 2>/dev/null
    codex sandbox "${MC_CODEX_ENV_ARGS[@]}" -- /usr/bin/env
  }
  names=$(bounded 120 e2e_names | sed 's/=.*//')
  printf '%s\n' "$names" | grep -qx MISSION_NAME && printf '%s\n' "$names" | grep -qx MISSION_DESIGNER_MODEL && ok "forwarded names visible inside the sandbox env" || bad "names not in sandbox env"

  e2e_res() { # e2e_res with|without
    . "$HELPER"
    export MISSION_NAME=fleet MISSION_DESIGNER_MODEL=claude:claude-opus-5-5
    MC_CODEX_ENV_ARGS=(); [ "$1" = with ] && mc_codex_env_args 2>/dev/null
    codex sandbox ${MC_CODEX_ENV_ARGS[@]+"${MC_CODEX_ENV_ARGS[@]}"} -- /bin/bash "$RESOLVER" designer
  }
  w=$(bounded 120 e2e_res with); wo=$(bounded 120 e2e_res without)
  case "$w" in recipe*) ok "resolver-e2e: with forwarding -> $(printf '%s' "$w" | cut -d' ' -f1-2)" ;; *) bad "resolver with args: $w" ;; esac
  case "$wo" in *fail-closed:designer-model-missing*) ok "resolver-e2e: without forwarding -> fail-closed (the ticket's symptom)" ;; *) bad "resolver without args: $wo" ;; esac
else
  skip "roundtrip-codex-e2e: codex not on PATH (CI) — run on the rig to prove arm (b)"
  skip "resolver-e2e: codex not on PATH (CI) — run on the rig"
fi

# --- (d) mutation: the assertion must be able to see a dropped forward
mkdir -p "$T/mut"
sed '/MC_CODEX_ENV_ARGS+=/d' "$HELPER" > "$T/mut/helper.sh"
sed 's/ \${MC_CODEX_ENV_ARGS\[@\]+"\${MC_CODEX_ENV_ARGS\[@\]}"}//' "$T/block.sh" > "$T/mut/block.sh"
cmp -s "$T/block.sh" "$T/mut/block.sh" && bad "call-site mutant is identical to the original"
sees() { run_block "$1" "$2" MISSION_NAME=fleet; argv_lines | grep -q '^shell_environment_policy\.set\.MISSION_NAME='; }
sees "$HELPER" "$T/block.sh" || bad "control: unmutated pair must pass"
sees "$T/mut/helper.sh" "$T/block.sh" && bad "arm (a) cannot see a dropped forward (helper mutant passed)" || ok "mutation-drop-forwarding: helper mutant goes red"
sees "$HELPER" "$T/mut/block.sh" && bad "arm (a) cannot see a dropped forward (call-site mutant passed)" || ok "mutation-drop-forwarding: call-site mutant goes red"

echo "==== passed=$PASS failed=$FAIL skipped=$SKIP ===="
[ "$FAIL" -eq 0 ]
