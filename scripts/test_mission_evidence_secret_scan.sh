#!/bin/bash
# Synthetic acceptance fixtures for the pre-publication scanner.
set +x
umask 077

helper="$(cd "$(dirname "$0")" && pwd)/mission_evidence_secret_scan.sh"
selected=''
while [ "$#" -gt 0 ]; do
  case "$1" in
    --helper) [ "$#" -ge 2 ] || exit 2; helper=$2; shift 2 ;;
    --arm) [ "$#" -ge 2 ] || exit 2; selected=$2; shift 2 ;;
    *) exit 2 ;;
  esac
done
case "$helper" in /*) ;; *) helper="$PWD/$helper" ;; esac

suite=$(mktemp -d "${TMPDIR:-/tmp}/evidence-scan-test.XXXXXX") || exit 2
worker_pid=''
helper_pid=''
cleanup() {
  if [ -n "$helper_pid" ]; then
    kill -KILL "$helper_pid" 2>/dev/null || true
    wait "$helper_pid" 2>/dev/null || true
  fi
  if [ -n "$worker_pid" ]; then kill -KILL "$worker_pid" 2>/dev/null || true; fi
  rm -rf "$suite"
}
trap cleanup EXIT
trap 'exit 2' HUP INT TERM

tool_path=/opt/homebrew/bin:/usr/local/bin:/usr/bin:/bin
export HOME="$suite/home" TMPDIR="$suite" LC_ALL=C PATH="$tool_path"
mkdir -p "$HOME" || exit 2
cd "$suite" || exit 2

control='SyntheticControl020!'
short='SyntheticShort019!!'
# shellcheck disable=SC2016 # Shell syntax is literal fixture data.
literal='literal.[x]*$(touch PWNED)'
oauth='synthetic-oauth-credential-136'
lower='synthetic-lower-credential-136'
from_file='synthetic-duplicate-file-136'
from_env='synthetic-duplicate-env-136'
hidden='synthetic-path-credential-136'
values=("$control" "$short" "$literal" "$oauth" "$lower" "$from_file" "$from_env" "$hidden")

fail() {
  printf 'FAIL %s/%s rc=%s\n' "$arm" "$1" "${2:-$rc}" >&2
  failures=$((failures + 1))
}

begin_case() {
  fixture=$(mktemp -d "$suite/case.XXXXXX") || exit 2
  fixture_home="$fixture/home"
  fixture_tmp="$fixture/tmp"
  mkdir -p "$fixture_home/.config/ailang" "$fixture_tmp" || exit 2
  config="$fixture_home/.config/ailang/secrets.env"
  exports="$fixture/exports"
  out="$fixture/stdout"
  err="$fixture/stderr"
  helper_path=$tool_path
  : > "$exports"
}

launch_helper() {
  cd "$fixture" || exit 2
  # shellcheck disable=SC2016 # The scrubbed Bash reads the fixture assignments.
  exec env -i HOME="$fixture_home" TMPDIR="$fixture_tmp" PATH="$helper_path" LC_ALL=C \
    /bin/bash -c '
      while IFS= read -r assignment; do export "$assignment"; done < "$1"
      shift
      exec /bin/bash "$@"
    ' fixture "$exports" "$helper" "$@" >"$out" 2>"$err"
}

run_helper() {
  rc=0
  (launch_helper "$@") || rc=$?
  assertions_done=0
}

expected_rc() { [ "$rc" -eq "$2" ] || fail "$1"; }

verdict() {
  local label=$1 status=$2 values_count=$3 scanned=$4 hits=$5 control_state=$6
  local expected="mission_evidence_secret_scan: verdict=$status values=$values_count scanned=$scanned hits=$hits control=$control_state"
  [ "$(tail -n 1 "$out")" = "$expected" ] || fail "$label"
  [ "$(awk '/^mission_evidence_secret_scan: /{n++} END{print n+0}' "$out")" -eq 1 ] ||
    fail single-verdict
  [ "$(awk '/^LEAK key=/{n++} END{print n+0}' "$out")" -eq "$hits" ] || fail finding-count
}

finding() { grep -Fqx -- "$2" "$out" || fail "$1"; }

finish_case() {
  local value result entries
  # Never display capture files, even for a leaking mutant.
  for value in "${values[@]}"; do
    result=0
    grep -aFq -f /dev/stdin "$out" "$err" <<< "$value" || result=$?
    case "$result" in 0) fail no-raw-values ;; 1) ;; *) fail output-inspection ;; esac
  done
  [ ! -s "$err" ] || fail empty-stderr
  entries=$(find "$fixture_tmp" -mindepth 1 -print) || fail scratch-inspection
  [ -z "$entries" ] || fail scratch-cleanup
  assertions_done=1
}

control_export() { printf 'CONTROL_TOKEN=%s\n' "$control" >> "$exports"; }

arm_clean_dir() {
  begin_case
  control_export
  mkdir -p "$fixture/tree/sub"
  printf 'credential name only: CONTROL_TOKEN\n' > "$fixture/tree/one"
  printf '\000clean\377\n' > "$fixture/tree/sub/two"
  run_helper tree
  expected_rc clean-rc 0
  verdict recursive-count clean 1 2 0 ok
  [ "$(wc -l < "$out")" -eq 1 ] || fail clean-record-count
}

arm_file_value() {
  begin_case
  control_export
  printf '  # comment\n\n  export FROM_FILE="%s"\n' "$literal" > "$config"
  printf '\000prefix %s suffix\377\n' "$literal" > "$fixture/evidence"
  run_helper evidence
  expected_rc file-literal-rc 1
  finding file-literal-finding 'LEAK key=FROM_FILE path=evidence'
  verdict file-literal-verdict LEAK 2 1 1 ok
  [ ! -e "$fixture/PWNED" ] || fail no-shell-execution
}

arm_env_only() {
  begin_case
  control_export
  printf 'CLAUDE_CODE_OAUTH_TOKEN=%s\nfixture_password=%s\n' "$oauth" "$lower" >> "$exports"
  printf '%s and %s\n' "$oauth" "$lower" > "$fixture/evidence"
  run_helper evidence
  expected_rc env-only-rc 1
  finding oauth-finding 'LEAK key=CLAUDE_CODE_OAUTH_TOKEN path=evidence'
  finding lowercase-finding 'LEAK key=fixture_password path=evidence'
  verdict env-only-verdict LEAK 3 1 2 ok
}

arm_make_dump() {
  begin_case
  printf "SHARED_KEY='%s'\n" "$from_file" > "$config"
  printf 'SHARED_KEY=%s\nBOTH_SECRET=%s\n' "$from_env" "$from_env" >> "$exports"
  printf 'NAME = %s\n' "$from_file" > "$fixture/file-dump"
  printf 'NAME = %s %s\n' "$from_env" "$from_env" > "$fixture/env-dump"
  run_helper file-dump env-dump env-dump
  expected_rc make-dump-rc 1
  finding file-side-finding 'LEAK key=SHARED_KEY path=file-dump'
  finding env-side-finding 'LEAK key=SHARED_KEY path=env-dump'
  finding second-key-finding 'LEAK key=BOTH_SECRET path=env-dump'
  verdict make-dump-verdict LEAK 2 2 3 ok
}

arm_short_value() {
  begin_case
  control_export
  printf 'SHORT_SECRET=%s\n' "$short" > "$config"
  printf '%s\n' "$short" > "$fixture/evidence"
  [ "${#short}" -eq 19 ] && [ "${#control}" -eq 20 ] || fail threshold-fixture 2
  run_helper evidence
  expected_rc short-value-rc 0
  verdict threshold-verdict clean 1 1 0 ok
}

arm_zero_values() {
  begin_case
  printf 'NOPE=%s\n' "$short" > "$config"
  : > "$fixture/evidence"
  run_helper evidence
  expected_rc zero-values-rc 2
  verdict empty-inventory instrument-failure 0 0 0 failed
}

arm_control() {
  begin_case
  control_export
  : > "$fixture/evidence"
  run_helper evidence
  expected_rc control-rc 0
  verdict firing-control clean 1 1 0 ok
}

# Positive assertions for the disposable broken-matcher probe.
arm_broken_matcher() {
  begin_case
  control_export
  : > "$fixture/evidence"
  run_helper evidence
  expected_rc broken-matcher-rc 2
  verdict broken-matcher-fail-closed instrument-failure 1 0 0 failed
}

arm_unreadable_path() {
  begin_case
  control_export
  : > "$fixture/locked"
  chmod 000 "$fixture/locked" || exit 2
  if [ -r "$fixture/locked" ]; then
    printf 'SKIP %s/permission-readable\n' "$arm"
  else
    run_helper locked
    expected_rc unreadable-rc 2
    verdict unreadable-verdict instrument-failure 1 0 0 ok
    finish_case
  fi
  begin_case
  control_export
  printf '%s\n' "$control" > "$fixture/leaking"
  run_helper absent leaking
  expected_rc missing-path-rc 2
  finding later-input-finding 'LEAK key=CONTROL_TOKEN path=leaking'
  verdict missing-path-precedence instrument-failure 1 1 1 ok
}

arm_invalid_inputs() {
  begin_case
  control_export
  printf 'BROKEN="unterminated\n' > "$config"
  : > "$fixture/evidence"
  run_helper evidence
  expected_rc malformed-source-rc 2
  verdict malformed-source-verdict instrument-failure 1 0 0 failed
  finish_case

  begin_case
  control_export
  mkdir "$fixture/tree"
  mkfifo "$fixture/tree/pipe"
  ln -s tree "$fixture/link"
  ln -s nowhere "$fixture/tree/dangling"
  : > "$fixture/tree/regular"
  run_helper link/ tree
  expected_rc unsafe-input-rc 2
  verdict unsafe-input-verdict instrument-failure 1 1 0 ok
  finish_case

  begin_case
  control_export
  mkdir "$fixture/bin" || exit 2
  cat > "$fixture/bin/python3" <<'WORKER'
#!/bin/bash
printf '%s\n' "$CONTROL_TOKEN"
printf '%s\n' "$CONTROL_TOKEN" >&2
exit 7
WORKER
  chmod +x "$fixture/bin/python3" || exit 2
  helper_path="$fixture/bin:$tool_path"
  run_helper evidence
  expected_rc worker-failure-rc 2
  verdict worker-failure-verdict instrument-failure 0 0 0 failed
}

arm_output_guard() {
  begin_case
  control_export
  printf 'PATH_TOKEN=%s\n' "$hidden" >> "$exports"
  printf '%s\n' "$control" > "$fixture/name-$hidden"
  printf '%s\n' "$control" > "$fixture/"$'line\nbreak'
  run_helper "name-$hidden" $'line\nbreak'
  expected_rc output-guard-rc 1
  finding redacted-path 'LEAK key=CONTROL_TOKEN path=[redacted]'
  finding escaped-path 'LEAK key=CONTROL_TOKEN path=line\u000abreak'
  verdict output-guard-verdict LEAK 2 2 2 ok
}

arm_interrupted() {
  begin_case
  control_export
  : > "$fixture/evidence"
  mkdir "$fixture/bin"
  cat > "$fixture/bin/python3" <<'WORKER'
#!/bin/bash
PATH=${PATH#*:} exec python3 -I -B "$HOME/worker.py"
WORKER
  cat > "$fixture_home/worker.py" <<'PYTHON'
import os
import signal
from pathlib import Path

home = Path.home()
(home / "pid.pending").write_text(str(os.getpid()))
(home / "pid.pending").rename(home / "worker.pid")
signal.pause()
PYTHON
  chmod +x "$fixture/bin/python3" || exit 2
  helper_path="$fixture/bin:$tool_path"
  (launch_helper evidence) &
  helper_pid=$!
  assertions_done=0
  local deadline=$(( $(date +%s) + 8 ))
  while [ ! -s "$fixture_home/worker.pid" ] && kill -0 "$helper_pid" 2>/dev/null &&
    [ "$(date +%s)" -lt "$deadline" ]; do sleep 0.05; done
  rc=2
  if [ ! -s "$fixture_home/worker.pid" ]; then
    fail interruption-worker-ready
    return
  fi
  worker_pid=$(cat "$fixture_home/worker.pid")
  [ -n "$(find "$fixture_tmp" -mindepth 1 -print)" ] || fail interruption-scratch-ready
  kill -TERM "$helper_pid" 2>/dev/null || fail send-term
  deadline=$(( $(date +%s) + 8 ))
  while kill -0 "$helper_pid" 2>/dev/null && [ "$(date +%s)" -lt "$deadline" ]; do sleep 0.05; done
  if kill -0 "$helper_pid" 2>/dev/null; then
    fail bounded-reap
    kill -KILL "$helper_pid" 2>/dev/null || true
  fi
  rc=0
  wait "$helper_pid" 2>/dev/null || rc=$?
  helper_pid=''
  expected_rc interrupted-rc 2
  verdict interrupted-verdict instrument-failure 0 0 0 failed
  if kill -0 "$worker_pid" 2>/dev/null; then fail worker-reaped; else worker_pid=''; fi
}

arms=(clean-dir file-value env-only make-dump short-value zero-values control
  unreadable-path invalid-inputs output-guard interrupted)
failures=0
ran=0
for arm in "${arms[@]}" broken-matcher; do
  if [ -z "$selected" ]; then [ "$arm" != broken-matcher ] || continue
  else [ "$arm" = "$selected" ] || continue
  fi
  before=$failures
  rc=2
  assertions_done=1
  "arm_${arm//-/_}"
  if [ "$assertions_done" -eq 0 ]; then finish_case; fi
  if [ "$failures" -eq "$before" ]; then printf 'PASS %s\n' "$arm"; fi
  ran=$((ran + 1))
done
[ "$ran" -gt 0 ] || { printf 'FAIL unknown-arm rc=2\n' >&2; exit 2; }
[ "$failures" -eq 0 ]
