#!/bin/bash
# shellcheck disable=SC2016
# Synthetic fixtures only. Positional scanner/driver copies and optional arm name
# make killing mutations possible without editing or sourcing the live driver.
HERE=$(cd "$(dirname "$0")" && pwd)
SCANNER=${1:-$HERE/transcript-secret-scan.sh}
DRIVER=${2:-$HERE/mission-control.sh}
SELECT=${3:-}
SCANNER=$(cd "$(dirname "$SCANNER")" && pwd)/$(basename "$SCANNER")
DRIVER=$(cd "$(dirname "$DRIVER")" && pwd)/$(basename "$DRIVER")
LAB=$(mktemp -d /tmp/transcript-scan-suite.XXXXXX) || exit 2
trap 'rm -rf "$LAB"' EXIT
# Set these before any code under test, including extraction/helper invocations.
HOME="$LAB/home" TMPDIR="$LAB/tmp"; export HOME TMPDIR
mkdir -p "$HOME" "$TMPDIR"
BASE_PATH=/usr/bin:/bin:/usr/sbin:/sbin

need() {
  local condition=$1; shift
  if ! "$@"; then printf '%s\n' "$condition" >> "$t/conditions"; return 1; fi
}
protect() {
  local v=$1 j s
  j=${v//\\/\\\\}; j=${j//\"/\\\"}
  j=${j//$'\b'/\\b}; j=${j//$'\f'/\\f}; j=${j//$'\n'/\\n}
  j=${j//$'\r'/\\r}; j=${j//$'\t'/\\t}; s=${j//\//\\/}
  guard_values+=("$v" "$j" "$s")
}
put() { printf '%s=%s\n' "$1" "$2" >> "$HOME/.config/ailang/secrets.env"; protect "$2"; }
fresh() { touch -t 202001010000.02 "$@"; }
scan() {
  local status
  set +e
  /bin/bash "$SCANNER" --since "$t/since" "$@" > "$t/out" 2> "$t/err"
  status=$?
  set -e
  need 'scanner exit status' test "$status" -eq "$want"
}
summary() {
  need 'exact terminal summary' /usr/bin/grep -Fxq \
    "transcript-secret-scan: verdict=$1 files=$2 keys=$3 leaks=$4 since=$t/since" "$t/out"
  need 'one terminal summary' test "$(/usr/bin/grep -c '^transcript-secret-scan: verdict=' "$t/out")" -eq 1
}
hit() { need 'expected key/path hit' /usr/bin/grep -Fxq "LEAK key=$1 file=$2" "$t/out"; }
extract() {
  awk -v start="# --- $1 START ---" -v end="# --- $1 END ---" \
    '$0 ~ "^[ ]*" start "$" {on=1} on {print} $0 ~ "^[ ]*" end "$" {on=0}' \
    "$DRIVER" > "$t/$2"
  need 'nonempty production block extraction' test -s "$t/$2"
}
# shellcheck disable=SC2329 # Invoked by the arm's EXIT trap.
guard_outputs() {
  local target pattern patterns="" bad=0
  for pattern in "${guard_values[@]}"; do
    [ -n "$pattern" ] || continue
    if [ -z "$patterns" ]; then patterns=$pattern; else patterns="$patterns"$'\n'"$pattern"; fi
  done
  for target in out err driver.log notices stdout stderr argv; do
    [ -f "$t/$target" ] || continue
    if /usr/bin/grep -F -c -f /dev/fd/3 -- "$t/$target" 3<<< "$patterns" >/dev/null 2>&1; then
      printf 'secret-free captured output (%s)\n' "$target" >> "$t/conditions"; bad=1
    fi
  done
  return "$bad"
}
# shellcheck disable=SC2329 # Invoked by trap, including assertion failures.
arm_exit() {
  local status=$?
  trap - EXIT
  guard_outputs || status=1
  [ -z "${scanner_pid:-}" ] || kill -KILL "$scanner_pid" 2>/dev/null || true
  [ -z "${child_pid:-}" ] || kill -KILL "$child_pid" 2>/dev/null || true
  exit "$status"
}
setup_driver() {
  extract 'HEARTBEAT STATE DIR' state.sh
  extract 'TRANSCRIPT SCAN MARKER' marker.sh
  extract 'TRANSCRIPT SCAN' scan.sh
  extract 'TRANSCRIPT SCAN DRY RUN' dry.sh
  MC_DRIVER_ROOT="$t/driver"
  mkdir -p "$MC_DRIVER_ROOT/tools/launchd" "$t/fallback"
  cp "$SCANNER" "$MC_DRIVER_ROOT/tools/launchd/transcript-secret-scan.sh"
  # shellcheck disable=SC2034 # Read by the extracted production blocks.
  STATE_DIR="$t/fallback" MISSION_NAME=fleet MISSION_FIRE_ID=synthetic-fire-85
  AILANG_STATE_DIR="$t/state"; export AILANG_STATE_DIR
  RC=37; _mc_slot_verdict='CRASHED at=gate-1'
  printf 'already recorded verdict rc=37\n' > "$t/state/verdict"
  before_verdict=$(cksum < "$t/state/verdict")
  # shellcheck disable=SC1091
  . "$t/state.sh"
  # shellcheck disable=SC1091
  . "$t/marker.sh"
}
log() { printf '%s\n' "$*" >> "$t/driver.log"; }
_mc_exit_notice() {
  printf 'notice label=%s title=%s\nmessage:\n%s\ngithub:\n%s\n' "$@" >> "$t/notices"
}
_mc_bounded() {
  local limit=$1 status; shift
  need 'production scan deadline is 120' test "$limit" -eq 120
  # shellcheck disable=SC2034 # Read by extracted scanner block.
  MC_BOUNDED_OUT=$("$@" 2>&1); status=$?
  return "$status"
}
run_driver_scan() {
  # Production driver has no errexit; scanner rc1/2 must reach its own status capture.
  set +e
  # shellcheck disable=SC1091
  . "$t/scan.sh"
  local status=$?
  set -e
  need 'driver scan block completes' test "$status" -eq 0
  need 'controller RC unchanged' test "$RC" -eq 37
  need 'controller verdict unchanged' test "$_mc_slot_verdict" = 'CRASHED at=gate-1'
  need 'written verdict unchanged' test "$(cksum < "$t/state/verdict")" = "$before_verdict"
  # shellcheck disable=SC2154 # Assigned by extracted marker block.
  need 'marker removed' test ! -e "$_mc_scan_marker"
}

arm() {
  case "$1" in
    Clean)
      printf 'ordinary transcript\n' > "$t/root/file"; fresh "$t/root/file"
      want=0; scan --root "$t/root"; summary clean 1 1 0 ;;
    Fresh_leak)
      printf '%s\n%s\n' "$A" "$A" > "$t/root/file"; fresh "$t/root/file"
      want=1; scan --root "$t/root"; hit ALPHA "$t/root/file"; summary LEAK 1 1 1 ;;
    Literal_metacharacters)
      : > "$HOME/.config/ailang/secrets.env"
      value='fake[abc].*+?^${}(meta)0123456789'; put META "$value"
      printf '%s\n' "$value" > "$t/root/file"; fresh "$t/root/file"
      want=1; scan --root "$t/root"; hit META "$t/root/file"; summary LEAK 1 1 1 ;;
    Older_leak|Marker_boundary)
      printf '%s\n' "$A" > "$t/root/old"
      if [ "$1" = Older_leak ]; then touch -t 201901010000.00 "$t/root/old"
      else touch -r "$t/since" "$t/root/old"; fi
      printf 'clean\n' > "$t/root/new"; fresh "$t/root/new"
      want=0; scan --root "$t/root"; summary clean 1 1 0 ;;
    Default_minimum)
      put SHORT tinyVal7; printf 'tinyVal7\n' > "$t/root/file"; fresh "$t/root/file"
      want=0; scan --root "$t/root"; summary clean 1 1 0 ;;
    Explicit_minimum)
      put SHORT tinyVal7; printf 'tinyVal7\n' > "$t/root/file"; fresh "$t/root/file"
      want=1; scan --min-len 8 --root "$t/root"; hit SHORT "$t/root/file"; summary LEAK 1 2 1 ;;
    Assignment_forms)
      : > "$HOME/.config/ailang/secrets.env"
      printf '# comment\n\nINVALID-NAME=%s\nPLAIN=%s\nSINGLE=\x27%s\x27\nexport DOUBLE="%s"' "$A" "$A" "$B" "$C" > "$HOME/.config/ailang/secrets.env"
      printf '%s\n%s\n%s\n' "$A" "$B" "$C" > "$t/root/file"; fresh "$t/root/file"
      want=1; scan --root "$t/root"; hit PLAIN "$t/root/file"; hit SINGLE "$t/root/file"; hit DOUBLE "$t/root/file"; summary LEAK 1 3 3 ;;
    Export_prefix)
      : > "$HOME/.config/ailang/secrets.env"; printf '  export EXPORTED=%s\n' "$B" > "$HOME/.config/ailang/secrets.env"
      printf '%s\n' "$B" > "$t/root/file"; fresh "$t/root/file"
      want=1; scan --root "$t/root"; hit EXPORTED "$t/root/file"; summary LEAK 1 1 1 ;;
    No_execution)
      value='$(touch "'"$HOME"'/sentinel")-FAKE-shell-value-0123456789'; put DANGER "$value"
      printf '%s\n' "$value" > "$t/root/file"; fresh "$t/root/file"
      want=1; scan --root "$t/root"; hit DANGER "$t/root/file"; summary LEAK 1 2 1
      need 'shell-looking value never executed' test ! -e "$HOME/sentinel" ;;
    Missing_secrets)
      want=2; scan --secrets-file "$HOME/absent" --root "$t/root"; summary instrument-failure 0 0 0 ;;
    Missing_marker)
      rm "$t/since"; make_find_spy
      want=2; scan --root "$t/root"; summary instrument-failure 0 0 0
      need 'no enumeration without marker' test ! -e "$HOME/find-called" ;;
    Zero_eligible_keys)
      : > "$HOME/.config/ailang/secrets.env"; put SHORT tinyVal7
      want=2; scan --root "$t/root"; summary instrument-failure 0 0 0 ;;
    Missing_root)
      printf 'clean\n' > "$t/root/file"; fresh "$t/root/file"
      want=0; scan --root "$t/absent" --root "$t/root"; summary clean 1 1 0
      need 'missing root named safely' /usr/bin/grep -Fxq "transcript-secret-scan: skipped missing root: $t/absent" "$t/err" ;;
    Repeated_roots)
      mkdir "$t/second"; printf '%s\n' "$A" > "$t/second/file"; fresh "$t/second/file"
      want=1; scan --root "$t/root" --root "$t/second"; hit ALPHA "$t/second/file"; summary LEAK 1 1 1 ;;
    Default_roots)
      for root in "$HOME/.codex-missions/sessions" "$HOME/.codex/sessions" "$HOME/.pi/agent/sessions" "$HOME/.claude/projects"; do
        mkdir -p "$root"; printf '%s\n' "$A" > "$root/file"; fresh "$root/file"
      done
      want=1; scan; summary LEAK 4 1 4 ;;
    JSON_escaping)
      : > "$HOME/.config/ailang/secrets.env"; value='FAKE-json-0123456789\path"quote'; put JSON "$value"
      printf '%s\n' 'FAKE-json-0123456789\\path\"quote' > "$t/root/file"; fresh "$t/root/file"
      want=1; scan --root "$t/root"; hit JSON "$t/root/file"; summary LEAK 1 1 1 ;;
    Escaped_slash)
      : > "$HOME/.config/ailang/secrets.env"; put SLASH 'FAKE-slash-0123456789/path/end'
      printf '%s\n' 'FAKE-slash-0123456789\/path\/end' > "$t/root/file"; fresh "$t/root/file"
      want=1; scan --root "$t/root"; hit SLASH "$t/root/file"; summary LEAK 1 1 1 ;;
    Deduplication)
      mkdir "$t/root/nested"; printf '%s\n%s\n' "$A" "$A" > "$t/root/nested/file"; fresh "$t/root/nested/file"
      want=1; scan --root "$t/root" --root "$t/root/nested" --root "$t/root"; summary LEAK 1 1 1
      need 'one hit despite overlapping roots' test "$(/usr/bin/grep -c '^LEAK ' "$t/out")" -eq 1 ;;
    Enumeration_failure)
      printf '#!/bin/bash\nprintf "unsafe diagnostic suppressed\\n" >&2\nexit 2\n' > "$t/bin/find"; chmod +x "$t/bin/find"
      want=2; scan --root "$t/root"; summary instrument-failure 0 1 0
      need 'raw find diagnostic suppressed' test "$(/usr/bin/grep -c 'unsafe diagnostic' "$t/err")" -eq 0 ;;
    Matcher_failure)
      printf 'clean\n' > "$t/root/file"; fresh "$t/root/file"
      printf '#!/bin/bash\nprintf "unsafe diagnostic suppressed\\n" >&2\nexit 2\n' > "$t/bin/grep"; chmod +x "$t/bin/grep"
      want=2; scan --root "$t/root"; summary instrument-failure 1 1 0
      need 'raw /usr/bin/grep diagnostic suppressed' test "$(/usr/bin/grep -c 'unsafe diagnostic' "$t/err")" -eq 0 ;;
    Secret_free_argv)
      cat > "$t/bin/grep" <<'SPY'
#!/bin/bash
printf '%s\n' "$@" >> "$HOME/argv"
exec /usr/bin/grep "$@"
SPY
      chmod +x "$t/bin/grep"
      printf '%s\n' "$A" > "$t/root/file"; fresh "$t/root/file"
      want=1; scan --root "$t/root"; summary LEAK 1 1 1
      cp "$HOME/argv" "$t/argv"
      need 'dedicated pattern descriptor observed' /usr/bin/grep -Fxq /dev/fd/3 "$t/argv" ;;
    Termination_cleanup) termination_arm ;;
    Driver_marker)
      setup_driver
      need 'resolved override root and fire namespace' test "$_mc_scan_marker" = "$t/state/mission-fleet-transcript-scan-synthetic-fire-85.since"
      need 'marker exists before spawn' test -f "$_mc_scan_marker"
      need 'fallback state untouched' test "$(find "$t/fallback" -type f | wc -l | tr -d ' ')" -eq 0
      need 'marker block precedes controller function' awk '
        /# --- HEARTBEAT STATE DIR END ---/ {state=NR}
        /# --- TRANSCRIPT SCAN MARKER START ---/ {marker=NR}
        /^_mc_run_once\(\)/ {spawn=NR}
        END {exit !(state && state<marker && marker<spawn)}' "$DRIVER"
      sleep 1; printf 'synthetic controller session\n' > "$HOME/session-write"
      need 'marker predates session write' test "$HOME/session-write" -nt "$_mc_scan_marker" ;;
    Driver_leak)
      setup_driver; put BETA "$B"; driver_transcript "$A" "$B"
      run_driver_scan
      need 'one leak notice' test "$(/usr/bin/grep -c '^notice label=' "$t/notices")" -eq 1
      need 'fire-specific title' /usr/bin/grep -Fq 'fire=synthetic-fire-85' "$t/notices"
      for key in ALPHA BETA; do
        need 'private message body contains every key/path' test "$(/usr/bin/grep -Fxc "LEAK key=$key file=$HOME/.codex/sessions/session" "$t/notices")" -eq 1
      done
      need 'public issue body names no key or path' test -z "$(sed -n '/^github:$/,$p' "$t/notices" | /usr/bin/grep -E 'LEAK|ALPHA|BETA|/\.codex/')"
      need 'public issue body carries the hit count' /usr/bin/grep -Fq ': 2 key/file hit(s).' "$t/notices"
      need 'scan holds overlap PID after durable verdict' awk '
        /# --- SLOT VERDICT END ---/ {v=NR}
        /# --- TRANSCRIPT SCAN START ---/ {s=NR}
        /^rm -f "\$PIDFILE"/ {p=NR}
        END {exit !(v && v<s && s<p)}' "$DRIVER" ;;
    Driver_clean)
      setup_driver; driver_transcript 'ordinary transcript'; run_driver_scan
      need 'clean sends no notice' test ! -s "$t/notices"
      need 'clean summary logged' /usr/bin/grep -Fq 'verdict=clean files=1 keys=1 leaks=0' "$t/driver.log" ;;
    Driver_failure)
      setup_driver; rm "$HOME/.config/ailang/secrets.env"; run_driver_scan
      need 'instrument failure logged' /usr/bin/grep -Fq 'INSTRUMENT FAILURE' "$t/driver.log"
      need 'failure summary logged' /usr/bin/grep -Fq 'verdict=instrument-failure' "$t/driver.log"
      need 'failure sends no notice' test ! -s "$t/notices"
      need 'failure never clean' test "$(/usr/bin/grep -c 'verdict=clean' "$t/driver.log")" -eq 0 ;;
    Driver_timeout) timeout_arm ;;
    Driver_dry_run)
      extract 'TRANSCRIPT SCAN DRY RUN' dry.sh
      _mc_scan_marker="$t/state/mission-fleet-transcript-scan-synthetic-fire-85.since"
      # shellcheck disable=SC1091
      . "$t/dry.sh"
      need 'dry run explains marker and 120-second scan' /usr/bin/grep -Eq 'DRY RUN.*marker.*120-second' "$t/driver.log"
      need 'dry run creates no marker' test ! -e "$_mc_scan_marker"
      need 'dry run sends no notice' test ! -s "$t/notices"
      need 'dry run has no scanner scratch' test "$(find "$TMPDIR" -type f | wc -l | tr -d ' ')" -eq 0
      need 'dry run block only logs' test "$(/usr/bin/grep -Evc '^[[:space:]]*(#|log |$)' "$t/dry.sh")" -eq 0 ;;
    Additional_contracts) additional_contracts ;;
    *) need 'known arm selected' false ;;
  esac
}
make_find_spy() {
  printf '#!/bin/bash\n: > "$HOME/find-called"\nexec /usr/bin/find "$@"\n' > "$t/bin/find"; chmod +x "$t/bin/find"
}
driver_transcript() {
  mkdir -p "$HOME/.codex/sessions"
  printf '%s\n' "$@" > "$HOME/.codex/sessions/session"
  touch -t 203001010000.00 "$HOME/.codex/sessions/session"
}
termination_arm() {
  printf 'clean\n' > "$t/root/file"; fresh "$t/root/file"
  cat > "$t/bin/grep" <<'CHILD'
#!/bin/bash
printf '%s\n' "$$" > "$HOME/child.pid"
exec /bin/sleep 30
CHILD
  chmod +x "$t/bin/grep"
  /bin/bash "$SCANNER" --since "$t/since" --root "$t/root" > "$t/out" 2> "$t/err" & scanner_pid=$!
  local n
  for ((n=0; n<100; n++)); do [ ! -s "$HOME/child.pid" ] || break; sleep 0.05; done
  need 'active matcher observed before TERM' test -s "$HOME/child.pid"
  child_pid=$(cat "$HOME/child.pid")
  kill -TERM "$scanner_pid"
  for ((n=0; n<60; n++)); do kill -0 "$scanner_pid" 2>/dev/null || break; sleep 0.05; done
  need 'scanner terminates promptly' test "$n" -lt 60
  set +e; wait "$scanner_pid"; local status=$?; set -e
  scanner_pid=''
  need 'handled TERM yields instrument failure' test "$status" -eq 2
  need 'active child terminated' test "$(kill -0 "$child_pid" 2>/dev/null; echo $?)" -ne 0
  child_pid=''
  need 'scratch directory removed on TERM' test "$(find "$TMPDIR" -mindepth 1 | wc -l | tr -d ' ')" -eq 0
  summary instrument-failure 1 1 0
}
timeout_arm() {
  setup_driver
  printf '#!/bin/bash\nexec /bin/sleep 5\n' > "$MC_DRIVER_ROOT/tools/launchd/transcript-secret-scan.sh"
  awk '/^_mc_bounded\(\) \{/,/^}/' "$HERE/lib/lane-probe.sh" > "$t/bounded.sh"
  need 'nonempty bounded helper extraction' test -s "$t/bounded.sh"
  # shellcheck disable=SC1091
  . "$t/bounded.sh"
  # Save the real helper under a test-only name; production still calls literal 120.
  declare -f _mc_bounded | sed '1s/_mc_bounded/real_bounded/' > "$t/real-bounded.sh"
  # shellcheck disable=SC1091
  . "$t/real-bounded.sh"
  # shellcheck disable=SC2329 # Called via the extracted driver.
  bounded_with_deadline() { local deadline=$1; shift; real_bounded "$deadline" "$@"; }
  # shellcheck disable=SC2329 # Called via the extracted driver.
  _mc_bounded() { shift; bounded_with_deadline 1 "$@"; }
  # shellcheck disable=SC2034 # Consumed by extracted helper.
  MC_BOUNDED_TERMINATION_GRACE=0
  local start; start=$(date +%s)
  run_driver_scan
  need 'bounded scanner cutoff' test "$(( $(date +%s) - start ))" -lt 5
  need 'timeout explicitly instrument failure' /usr/bin/grep -Fq 'INSTRUMENT FAILURE (rc=124;' "$t/driver.log"
  need 'timeout sends no notice' test ! -s "$t/notices"
}
additional_contracts() {
  # Last duplicate replaces prior value; preserve embedded equals, spaces, slashes.
  : > "$HOME/.config/ailang/secrets.env"
  put LAST "$A"; put LAST tinyVal7
  value='FAKE-equals=with spaces\backslash-0123456789'; put COMPLEX "$value"
  printf '%s\n' "$value" > "$t/root/space name"; fresh "$t/root/space name"
  want=1; scan --root "$t/root"; hit COMPLEX "$t/root/space name"; summary LEAK 1 1 1
  local invalid
  for invalid in 0 -1 nope 1.5 ''; do
    want=2; scan --min-len "$invalid" --root "$t/root"; summary instrument-failure 0 0 0
  done
  want=1; scan --min-len 00020 --root "$t/root"; summary LEAK 1 1 1
  # Newline and tab in filenames must remain single-line hit records.
  mv "$t/root/space name" "$t/root/line"$'\n\t'"name"
  want=1; scan --root "$t/root"
  hit COMPLEX "$t/root/line\\u000a\\u0009name"; summary LEAK 1 1 1
  need 'one escaped pathname record' test "$(wc -l < "$t/out" | tr -d ' ')" -eq 2
  setup_driver
  # A partial or inconsistent summary must never be presented as a clean scan.
  printf '#!/bin/bash\nprintf "transcript-secret-scan: verdict=clean files=0 keys=1 leaks=0\\n"\n' > "$MC_DRIVER_ROOT/tools/launchd/transcript-secret-scan.sh"
  run_driver_scan
  need 'malformed summary refused' /usr/bin/grep -Fq 'INSTRUMENT FAILURE' "$t/driver.log"
  need 'malformed summary sends no notice' test ! -s "$t/notices"
}

arms='Clean Fresh_leak Literal_metacharacters Older_leak Marker_boundary Default_minimum Explicit_minimum Assignment_forms Export_prefix No_execution Missing_secrets Missing_marker Zero_eligible_keys Missing_root Repeated_roots Default_roots JSON_escaping Escaped_slash Deduplication Enumeration_failure Matcher_failure Secret_free_argv Termination_cleanup Driver_marker Driver_leak Driver_clean Driver_failure Driver_timeout Driver_dry_run Additional_contracts'
pass=0 fail=0
for name in $arms; do
  [ -z "$SELECT" ] || [ "$SELECT" = "$name" ] || continue
  t=$(mktemp -d "$LAB/arm.XXXXXX") || exit 2
  (
    set -e
    HOME="$t/home" TMPDIR="$t/tmp" PATH="$t/bin:$BASE_PATH" AILANG_STATE_DIR="$t/state"
    export HOME TMPDIR PATH AILANG_STATE_DIR
    mkdir -p "$HOME/.config/ailang" "$TMPDIR" "$t/root" "$t/bin" "$t/state"
    guard_values=()
    A='alphaFAKEvalue0123456789-XYZ'; B='betaFAKEvalue0123456789-with space'; C='gammaFAKEvalue0123456789=tail'
    protect "$A"; protect "$B"; protect "$C"
    : > "$t/out"; : > "$t/err"; : > "$t/driver.log"; : > "$t/notices"
    trap arm_exit EXIT
    put ALPHA "$A"
    touch -t 202001010000.00 "$t/since"
    arm "$name"
  ) > "$t/stdout" 2> "$t/stderr"
  status=$?
  if [ "$status" -eq 0 ]; then
    printf 'PASS: %s\n' "$name"; pass=$((pass + 1))
  else
    printf 'FAIL: %s (rc=%s)\n' "$name" "$status"; fail=$((fail + 1))
    # Conditions are authored by need()/guard_outputs(), never captured subprocess output.
    [ ! -f "$t/conditions" ] || while IFS= read -r condition; do printf '  %s: %s\n' "$name" "$condition"; done < "$t/conditions"
  fi
done
if [ "$pass" -eq 0 ] || [ "$fail" -ne 0 ]; then exit 1; fi
printf 'transcript scan suite: %s arms passed\n' "$pass"
