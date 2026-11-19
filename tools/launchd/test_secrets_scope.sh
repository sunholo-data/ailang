#!/bin/bash
# shellcheck shell=bash
# Security fixtures always run with a fresh HOME, empty startup controls and fake values.
set -u
HERE=$(cd "$(dirname "$0")" && pwd)
ROOT=$(cd "$HERE/../.." && pwd)
T=$(mktemp -d)
trap 'rm -rf "$T"' EXIT
PASS=0; FAIL=0
ok() { PASS=$((PASS+1)); printf '  PASS: %s\n' "$1"; }
bad() { FAIL=$((FAIL+1)); printf '  FAIL: %s\n' "$1"; }

# Each arm below names its single-line killing mutation; --mutations banks all results.
# The worker is shared by both interpreters. Assertions print only names, never values.
cat > "$T/worker.sh" <<'WORKER'
arm=$1; lib=$2; bridge=$3; host=$4
# This known-safe seed is distinct from the parser's hostile inventory.
. "$HOME/seed.sh"
assert_absent() {
  # Indirection is deliberately avoided: a child shell tests a supplied identifier.
  /usr/bin/awk -v key="$1" 'BEGIN { exit (key in ENVIRON) }' || return 90
}
assert_present() {
  /usr/bin/awk -v key="$1" 'BEGIN { exit !(key in ENVIRON) }' || return 91
}
check_lane() {
  expected=$1
  while IFS= read -r key; do
    case " $expected " in
      *" $key "*) assert_present "$key" || exit 92 ;;
      *) assert_absent "$key" || exit 93 ;;
    esac
  done < "$HOME/keys"
}
case "$arm" in
  source)
    . "$lib" > "$HOME/source.out" 2> "$HOME/source.err" || exit 94
    [ ! -s "$HOME/source.out" ] && [ ! -s "$HOME/source.err" ] || exit 95
    check_lane "$(tr '\n' ' ' < "$HOME/keys")"
    /usr/bin/awk '{if (ENVIRON[$0] != sprintf("FAKE_SCOPE_VALUE_%02d_END", NR)) exit 1}' "$HOME/keys" || exit 119
    [ "$UNRELATED" = unrelated ] || exit 120
    # Locals used by apply must not appear after return either.
    assert_present UNRELATED || exit 96
    ;;
  lane-*)
    lane=${arm#lane-}
    . "$lib" || exit 94
    mc_secrets_scope_lane_valid "$lane" || exit 97
    case "$lane" in
      *-controller) expected='CLAUDE_CODE_OAUTH_TOKEN OLLAMA_API_KEY OPENROUTER_API_KEY OPENAI_API_KEY' ;;
      claude-role) expected=CLAUDE_CODE_OAUTH_TOKEN ;;
      codex-role) expected='' ;;
      pi-role) expected='OLLAMA_API_KEY OPENROUTER_API_KEY' ;;
    esac
    actual=$(mc_secrets_scope_allow "$lane") || exit 98
    [ "$(printf '%s' "$actual" | tr '\n' ' ')" = "$expected" ] || exit 99
    mc_secrets_scope_apply "$lane" > "$HOME/apply.out" 2> "$HOME/apply.err" || exit 100
    [ ! -s "$HOME/apply.out" ] || exit 101
    # zsh does not split $expected: count allowlist lines independently instead.
    count=$(printf '%s\n' "$actual" | awk 'NF {n++} END {print n+0}')
    names=''; stripped=''; nstripped=0
    while IFS= read -r key; do
      case " $expected " in
        *" $key "*) names="${names}${names:+,}$key" ;;
        *) stripped="${stripped}${stripped:+,}$key"; nstripped=$((nstripped+1)) ;;
      esac
    done < "$HOME/keys"
    grep -Fxq "secrets-scope: lane=$lane kept=${count}[${names}] stripped=${nstripped}[${stripped}]" "$HOME/apply.err" || exit 102
    [ "$(wc -l < "$HOME/apply.err")" -eq 1 ] || exit 103
    cat "$HOME/apply.out" "$HOME/apply.err"
    check_lane "$expected"
    assert_present UNRELATED || exit 104
    # Caller-local helper names must disappear on return (checked without printing values).
    [ "${_mc_scope_lane+x}${_mc_scope_file+x}${_mc_scope_names+x}${_mc_scope_name+x}${_mc_scope_allow+x}${_mc_scope_rc+x}${_mc_scope_kept+x}${_mc_scope_stripped+x}${_mc_scope_nkept+x}${_mc_scope_nstripped+x}" = '' ] || exit 105
    ;;
  unknown)
    . "$lib" || exit 94
    mc_secrets_scope_lane_valid no-such-lane && exit 106
    mc_secrets_scope_allow no-such-lane > "$HOME/allow.out"; rc=$?
    [ "$rc" -eq 64 ] && [ ! -s "$HOME/allow.out" ] || exit 107
    mc_secrets_scope_apply no-such-lane > "$HOME/apply.out" 2> "$HOME/apply.err"; rc=$?
    [ "$rc" -eq 64 ] || exit 108
    [ ! -s "$HOME/apply.out" ] && grep -q FAIL-CLOSED "$HOME/apply.err" || exit 109
    cat "$HOME/apply.err"
    check_lane ''
    ;;
  future|parent|multiple|helper-collision)
    . "$lib" || exit 94
    mc_secrets_scope_apply codex-role || exit 100
    # Observe the caller's variable itself, as well as the child's exported environment.
    [ "${ANTHROPIC_API_KEY+x}${FUTURE_SCOPE_KEY+x}${OPENAI_API_KEY+x}" = '' ] || exit 121
    check_lane ''
    ;;
  forms|dedup|decoy|noeval|disclosure)
    . "$lib" || exit 94
    mc_secrets_scope_names "$HOME/.config/ailang/secrets.env" > "$HOME/names" || exit 110
    cmp -s "$HOME/keys" "$HOME/names" || exit 111
    mc_secrets_scope_apply codex-role || exit 100
    [ ! -e "$HOME/sentinel-dollar" ] && [ ! -e "$HOME/sentinel-tick" ] || exit 112
    check_lane ''
    ;;
  missing|unset-failure)
    . "$lib" || exit 94
    if [ "$arm" = missing ]; then
      rm "$HOME/.config/ailang/secrets.env"
      mc_secrets_scope_names "$HOME/.config/ailang/secrets.env" > "$HOME/names"; rc=$?
      [ "$rc" -eq 78 ] && [ ! -s "$HOME/names" ] || exit 113
    else
      readonly FUTURE_SCOPE_KEY
    fi
    mc_secrets_scope_apply claude-controller > "$HOME/apply.out" 2> "$HOME/apply.err"; rc=$?
    [ "$rc" -eq 78 ] || exit 114
    [ ! -s "$HOME/apply.out" ] && grep -q FAIL-CLOSED "$HOME/apply.err" || exit 115
    cat "$HOME/apply.err"
    # The seven-name floor is attempted even for a lane that would otherwise retain keys.
    while IFS= read -r key; do assert_absent "$key" || exit 116; done < "$HOME/seven"
    ;;
  bridge-unset|bridge-empty)
    unset AILANG_SECRETS_LANE
    [ "$arm" = bridge-empty ] && AILANG_SECRETS_LANE=''
    MISSION_DRIVER_ROOT="$HOME/nonexistent"
    . "$bridge" > "$HOME/bridge.out" 2> "$HOME/bridge.err" || exit 117
    [ ! -s "$HOME/bridge.out" ] && [ ! -s "$HOME/bridge.err" ] || exit 118
    check_lane "$(tr '\n' ' ' < "$HOME/keys")"
    ;;
  host-*)
    case "$arm" in
      host-unset) unset AILANG_SECRETS_LANE; MISSION_DRIVER_ROOT="$HOME/nonexistent" ;;
      host-empty) AILANG_SECRETS_LANE=''; MISSION_DRIVER_ROOT="$HOME/nonexistent" ;;
      *) AILANG_SECRETS_LANE=pi-role ;;
    esac
    case "$arm" in
      host-missing-root) MISSION_DRIVER_ROOT="$HOME/nonexistent" ;;
      host-missing-lib) MISSION_DRIVER_ROOT="$HOME/no-lib" ;;
      host-missing-bridge) MISSION_DRIVER_ROOT="$HOME/no-bridge" ;;
    esac
    # Simulated measured startup ordering: secrets source, exact doc line, billing unset, body.
    # A shell EXIT trap observes the environment even if the host line refuses the shell.
    trap 'while IFS= read -r key; do /usr/bin/awk -v key="$key" '\''BEGIN {if (key in ENVIRON) print key}'\''; done < "$HOME/keys" > "$HOME/exit-names"' EXIT
    . "$HOME/startup.sh"
    ;;
esac
WORKER

# One fake value per fixture key; inventory parsing must never evaluate these expressions.
fixture() {
  local home=$1 key i=0
  mkdir -p "$home/.config/ailang" "$home/.codex" "$home/zdot"
  printf '%s\n' ANTHROPIC_API_KEY CLAUDE_CODE_OAUTH_TOKEN GEMINI_AGENTS_API_KEY GOOGLE_API_KEY OLLAMA_API_KEY OPENAI_API_KEY OPENROUTER_API_KEY > "$home/seven"
  cat "$home/seven" > "$home/keys"
  printf '%s\n' FUTURE_SCOPE_KEY PLAIN_SCOPE_KEY EXPORTED_SCOPE_KEY INDENTED_SCOPE_KEY VALUE DOLLAR_VALUE TICK_VALUE >> "$home/keys"
  # Future names may collide with implementation locals: those globals must still be unset.
  printf '%s\n' _mc_scope_lane _mc_scope_file _mc_scope_names _mc_scope_name _mc_scope_allow _mc_scope_rc _mc_scope_kept _mc_scope_stripped _mc_scope_nkept _mc_scope_nstripped >> "$home/keys"
  : > "$home/seed.sh"; : > "$home/values"
  while IFS= read -r key; do
    i=$((i+1))
    printf "export %s='FAKE_SCOPE_VALUE_%02d_END'\n" "$key" "$i" >> "$home/seed.sh"
    printf 'FAKE_SCOPE_VALUE_%02d_END\n' "$i" >> "$home/values"
  done < "$home/keys"
  printf 'export UNRELATED=unrelated\n' >> "$home/seed.sh"
  # These safe seed records exercise plain, export and indented assignment forms in inventory.
  awk '
    NR==9 {sub(/^export /, "")}
    NR==11 {$0=" \t" $0}
    {print}
  ' "$home/seed.sh" | sed '/UNRELATED=/d' > "$home/.config/ailang/secrets.env"
  cat >> "$home/.config/ailang/secrets.env" <<'INVENTORY'
export OLLAMA_API_KEY='FAKE_SCOPE_VALUE_05_END'
# export COMMENT_DECOY=FAKE_COMMENT_DECOY_END
VALUE='export DECOY_KEY=FAKE_DECOY_END'
INVALID-NAME=FAKE_INVALID_END
9INVALID=FAKE_INVALID_DIGIT_END
DOLLAR_VALUE=$(touch "$HOME/sentinel-dollar")
TICK_VALUE=`touch "$HOME/sentinel-tick"`
INVENTORY
  printf '%s\n' FAKE_COMMENT_DECOY_END FAKE_DECOY_END FAKE_INVALID_END FAKE_INVALID_DIGIT_END >> "$home/values"
}

# Subprocess isolation uses Python's explicit clean environment, never an inherited startup file.
run_child() {
  python3 - "$@" <<'PY'
import subprocess, sys
shell, home, worker, arm, lib, bridge, host, root = sys.argv[1:]
e = dict(HOME=home, ZDOTDIR=home+'/zdot', CODEX_HOME=home+'/.codex',
         PATH='/usr/bin:/bin:/usr/sbin:/sbin', BASH_ENV='', ENV='',
         AILANG_STATE_DIR=home+'/.ailang/state', MISSION_DRIVER_ROOT=root,
         LC_ALL='C')
r = subprocess.run([shell, '-c', 'worker=$1; shift; . "$worker"', 'scope-test', worker, arm, lib, bridge, host], env=e)
sys.exit(r.returncode)
PY
}

extract_host() {
  awk '/<!-- SECRETS-SCOPE HOST LINE START -->/ {inside=1; next}
       /<!-- SECRETS-SCOPE HOST LINE END -->/ {inside=0}
       inside && !/^```/ && NF {print}' "$1" > "$2"
  [ -s "$2" ] && [ "$(wc -l < "$2")" -eq 1 ]
}

# Validate all captured outputs, including auxiliary redirects inside the worker, without echoing leaks.
no_disclosure() {
  local home=$1 output=$2 value file
  while IFS= read -r value; do
    for file in "$output" "$home"/*.out "$home"/*.err "$home/names"; do
      [ -f "$file" ] || continue
      if grep -Fq "$value" "$file"; then return 1; fi
    done
  done < "$home/values"
}

run_arm() {
  local shell=$1 arm=$2 root=$3 host=$4 home output rc=0
  home=$(mktemp -d "$T/home.XXXXXX")
  fixture "$home"
  output="$home/captured"
  mkdir -p "$home/no-lib/tools/launchd/lib"
  cp "$root/tools/launchd/lib/secrets-scope-zshenv.sh" "$home/no-lib/tools/launchd/lib/" || return 1
  {
    # shellcheck disable=SC2016 # Written for the isolated child to expand.
    printf '. "$HOME/seed.sh"\n'
    cat "$host"
    printf 'unset ANTHROPIC_API_KEY ANTHROPIC_AUTH_TOKEN\nprintf "BODY\\n"\n'
  } > "$home/startup.sh"
  run_child "$shell" "$home" "$T/worker.sh" "$arm" \
    "$root/tools/launchd/lib/secrets-scope.sh" "$root/tools/launchd/lib/secrets-scope-zshenv.sh" "$host" "$root" > "$output" 2>&1 || rc=$?
  no_disclosure "$home" "$output" || { printf "    no-disclosure assertion failed\n"; return 1; }
  case "$arm" in
    host-missing-*)
      [ "$rc" -eq 78 ] && ! grep -q '^BODY$' "$output" &&
        [ -f "$home/exit-names" ] && [ ! -s "$home/exit-names" ] &&
        [ "$(grep -c FAIL-CLOSED "$output")" -eq 1 ]
      ;;
    host-*)
      [ "$rc" -eq 0 ] && [ "$(grep -c '^BODY$' "$output")" -eq 1 ] || return 1
      case "$arm" in
        host-unset|host-empty) sed '/^ANTHROPIC_API_KEY$/d' "$home/keys" > "$home/expected" ;;
        *) printf '%s\n' OLLAMA_API_KEY OPENROUTER_API_KEY > "$home/expected" ;;
      esac
      cmp -s "$home/expected" "$home/exit-names"
      ;;
    *) if [ "$rc" -ne 0 ]; then printf "    worker assertion rc=%s\n" "$rc"; return 1; fi ;;
  esac
}

DOC="$ROOT/design_docs/planned/m-fleet-secrets-scope-per-lane.md"
if ! extract_host "$DOC" "$T/host.sh" 2>/dev/null; then
  bad 'host extraction: exactly one nonempty physical line required'
  printf '%s passed, %s failed\n' "$PASS" "$FAIL"
  exit 1
fi
ok 'host extraction: exactly one nonempty physical line'
for shell in /bin/bash /bin/zsh; do
  if [ ! -x "$shell" ]; then
    bad "UNINFORMATIVE: missing $shell"
  elif "$shell" -n "$T/host.sh"; then
    ok "$shell host syntax"
  else
    bad "$shell host syntax"
  fi
done

# source: append mc_secrets_scope_apply codex-role at library file scope.
# lane-<lane>: remove one kept key (codex-role: add OPENAI_API_KEY) in that lane mapping.
# unknown: change unknown allowlist return 64 to a controller allowlist.
# future: ignore FUTURE_SCOPE_KEY in parser output (fixed-inventory regression).
# forms: remove optional export prefix, thereby rejecting plain assignments.
# dedup: remove seen[n]++ guard in awk.
# decoy: remove beginning-of-record anchor in awk.
# noeval: source the inventory inside apply.
# disclosure: print inventory records from apply.
# parent: run the unset call in a subshell (the pipeline-loop regression).
# helper-collision: replace namespace preclear call with : (globals shadowed by locals).
# missing: change inventory-error return from 78 to 0.
# unset-failure: remove the emergency-floor call after a failed unset.
# bridge-unset/empty: remove marker guard from bridge.
# host-good: remove bridge apply.
# host-unset/empty: make outer marker guard unconditional.
# host-missing-lib: make bridge return 0 if library is absent.
# host-missing-root/bridge: replace host failure exit 78 with return 0.
# multiple: replace caller loop with for name in $names (no zsh splitting).
ARMS='source lane-claude-controller lane-codex-controller lane-pi-controller lane-claude-role lane-codex-role lane-pi-role unknown future helper-collision forms dedup decoy noeval disclosure parent missing unset-failure bridge-unset bridge-empty host-good host-unset host-empty host-missing-lib host-missing-root host-missing-bridge multiple'

if [ "${1-}" != --mutations ]; then
  for shell in /bin/bash /bin/zsh; do
    [ -x "$shell" ] || continue
    for arm in $ARMS; do
      if run_arm "$shell" "$arm" "$ROOT" "$T/host.sh"; then ok "$shell $arm"; else bad "$shell $arm"; fi
    done
  done
else
  # Every control must pass before its mutant can be called killed.
  # Mutation construction checks exactly one changed/added physical line.
  for arm in $ARMS; do
    scratch=$(mktemp -d "$T/mutant.XXXXXX")
    mkdir -p "$scratch/tools/launchd/lib"
    cp "$ROOT/tools/launchd/lib/secrets-scope.sh" "$ROOT/tools/launchd/lib/secrets-scope-zshenv.sh" "$scratch/tools/launchd/lib/"
    cp "$T/host.sh" "$scratch/host.sh"
    if ! python3 - "$scratch" "$arm" <<'PYMUT'
import difflib, json, pathlib, sys
root, arm = pathlib.Path(sys.argv[1]), sys.argv[2]
lib = root/'tools/launchd/lib/secrets-scope.sh'
bridge = root/'tools/launchd/lib/secrets-scope-zshenv.sh'
host = root/'host.sh'
p = lib
s = p.read_text()
if arm == 'source':
    new = s + 'mc_secrets_scope_apply codex-role\n'
elif arm.startswith('lane-'):
    lane = arm[5:]
    allowed = {
        'claude-controller': 'CLAUDE_CODE_OAUTH_TOKEN OLLAMA_API_KEY OPENROUTER_API_KEY',
        'codex-controller': 'CLAUDE_CODE_OAUTH_TOKEN OLLAMA_API_KEY OPENROUTER_API_KEY',
        'pi-controller': 'CLAUDE_CODE_OAUTH_TOKEN OLLAMA_API_KEY OPENROUTER_API_KEY',
        'claude-role': '', 'codex-role': 'OPENAI_API_KEY', 'pi-role': 'OLLAMA_API_KEY',
    }[lane]
    output = "printf '%s\\n' " + allowed if allowed else ':'
    new = s.replace('mc_secrets_scope_allow() {', 'mc_secrets_scope_allow() {\n  if [ "${1-}" = "'+lane+'" ]; then '+output+'; return 0; fi', 1)
elif arm == 'unknown':
    marker = '    *) return 64 ;;'
    idx = s.index(marker, s.index('mc_secrets_scope_allow() {'))
    new = s[:idx] + s[idx:].replace(marker, '    *) printf "%s\\n" OPENAI_API_KEY ;;', 1)
elif arm == 'future':
    new = s.replace('if (!seen[n]++) print n', 'if (n=="FUTURE_SCOPE_KEY") next; if (!seen[n]++) print n', 1)
elif arm == 'forms':
    new = s.replace('(export[ \\t]+)?', 'export[ \\t]+', 1)
elif arm == 'dedup':
    new = s.replace('if (!seen[n]++) print n', 'print n', 1)
elif arm == 'decoy':
    new = s.replace("awk '/^", "awk '/", 1)
elif arm in ('noeval', 'disclosure'):
    action = '(. "$_mc_scope_file") >/dev/null 2>&1' if arm == 'noeval' else 'cat "$_mc_scope_file"'
    new = s.replace('  # Apply the already parsed names;', '  '+action+'\n  # Apply the already parsed names;', 1)
elif arm == 'parent':
    new = s.replace('if ! mc_secrets_scope_unset "$_mc_scope_name"; then', 'if ! (mc_secrets_scope_unset "$_mc_scope_name"); then', 1)
elif arm == 'multiple':
    new = s.replace('while IFS= read -r _mc_scope_name; do', 'for _mc_scope_name in $_mc_scope_names; do', 1)
elif arm == 'helper-collision':
    line = next(x for x in s.splitlines() if 'if ! mc_secrets_scope_prepare' in x)
    new = s.replace(line, '  if ! :; then', 1)
elif arm == 'missing':
    new = s.replace("    return 78\n  fi\n  set --", "    return 0\n  fi\n  set --", 1)
elif arm == 'unset-failure':
    new = s.replace('if [ "$_mc_scope_rc" -eq 78 ]; then\n    mc_secrets_scope_floor', 'if [ "$_mc_scope_rc" -eq 78 ]; then\n    :', 1)
elif arm.startswith('bridge-'):
    p = bridge; s = p.read_text()
    new = s.replace('[ -n "${AILANG_SECRETS_LANE-}" ] || return 0', ':', 1)
elif arm == 'host-good':
    p = bridge; s = p.read_text()
    new = s.replace('mc_secrets_scope_apply "$AILANG_SECRETS_LANE"', ':', 1)
elif arm in ('host-unset', 'host-empty'):
    p = host; s = p.read_text()
    new = s.replace('if [ -n "${AILANG_SECRETS_LANE-}" ]; then', 'if :; then', 1)
elif arm == 'host-missing-lib':
    p = bridge; s = p.read_text()
    new = s.replace('] || return 78', '] || return 0', 1)
elif arm in ('host-missing-root', 'host-missing-bridge'):
    p = host; s = p.read_text()
    new = s.replace('exit 78', 'return 0', 1)
else:
    raise SystemExit('unknown mutation')
assert new != s, 'ineffective mutation construction'
# Exactly one changed line, or one inserted line. No silent malformed mutants.
a, b = s.splitlines(), new.splitlines()
if len(a) == len(b):
    assert sum(x != y for x,y in zip(a,b)) == 1
else:
    assert len(b) == len(a)+1
    assert any(b[:i]+b[i+1:] == a for i in range(len(b)))
p.write_text(new)
print('    mutant='+arm+' target='+p.name)
print('    MUTATION-DIFF arm='+arm+' '+json.dumps(''.join(difflib.unified_diff(s.splitlines(True), new.splitlines(True), fromfile=p.name, tofile=p.name, n=0))))
PYMUT
    then
      bad "$arm mutation construction"
      continue
    fi
    for shell in /bin/bash /bin/zsh; do
      [ -x "$shell" ] || continue
      # This particular mutant is intentionally equivalent under bash word splitting.
      if [ "$arm" = multiple ] && [ "$shell" = /bin/bash ]; then
        printf '  EQUIVALENT: %s %s (bash splits this fixture; zsh arm is decisive)\n' "$shell" "$arm"
        continue
      fi
      if ! run_arm "$shell" "$arm" "$ROOT" "$T/host.sh"; then
        bad "$shell $arm UNINFORMATIVE: control failed"
      elif run_arm "$shell" "$arm" "$scratch" "$scratch/host.sh"; then
        bad "$shell $arm SURVIVED"
      else
        ok "$shell $arm KILLED"
      fi
    done
  done
fi
# M2 integration arms execute real production blocks and have their own killing mutants.
if /bin/bash "$HERE/test_secrets_scope_spawn.sh" "${1-}" "${2-}"; then
  ok 'M2 actual spawn arms'
else
  bad 'M2 actual spawn arms'
fi
if /bin/bash "$HERE/test_secrets_scope_dry_run.sh" "${1-}" "${2-}"; then
  ok 'M3 actual dry-run arms'
else
  bad 'M3 actual dry-run arms'
fi
printf '%s passed, %s failed\n' "$PASS" "$FAIL"
[ "$FAIL" -eq 0 ]
