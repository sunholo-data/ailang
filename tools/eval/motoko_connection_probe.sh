#!/usr/bin/env bash
set -uo pipefail

instrument_failure() {
  echo "INSTRUMENT FAILURE: $*" >&2
  exit 1
}

usage() {
  cat >&2 <<'EOF'
usage: motoko_connection_probe.sh TREATMENT_LANE CONTROL_LANE ARTIFACT [BENCHMARK]
       motoko_connection_probe.sh --classify-fixture OR_IPS_FILE LSOF_FILE
       motoko_connection_probe.sh --assert-nonempty PEERS_FILE
       motoko_connection_probe.sh --assert-treatment PEERS_FILE OR_IPS_FILE
       motoko_connection_probe.sh --assert-control PEERS_FILE OR_IPS_FILE

The live form is darwin/arm64-only. AILANG_BIN and PROBE_TIMEOUT_SECS may be
set to select the driver and bound each lane (defaults: ailang and 900).
EOF
  exit 2
}

peer_host() {
  local peer=$1
  if [[ "$peer" == \[*\]:* ]]; then
    printf '%s]\n' "${peer%]:*}"
  else
    printf '%s\n' "${peer%:*}"
  fi
}

is_loopback_host() {
  local host=$1
  [[ "$host" == 127.* || "$host" == "[::1]" || "$host" == "::1" || "$host" == "localhost" ]]
}

or_ip_member() {
  local host=$1 or_file=$2
  grep -Fqx -- "$host" "$or_file"
}

classify_lsof() {
  local or_file=$1 lsof_file=$2 peer host class
  awk '{ for (i=1; i<=NF; i++) if (index($i,"->")) { peer=$i; sub(/^.*->/,"",peer); print peer } }' "$lsof_file" |
    LC_ALL=C sort -u |
    while IFS= read -r peer; do
      [[ -n "$peer" ]] || continue
      host=$(peer_host "$peer")
      class=other
      if is_loopback_host "$host"; then
        class=loopback
      elif or_ip_member "$host" "$or_file"; then
        class=openrouter
      fi
      printf '%s\t%s\n' "$class" "$peer"
    done
}

assert_nonempty() {
  local peers_file=$1
  if [[ ! -s "$peers_file" ]]; then
    instrument_failure "empty peer set; absence of evidence cannot prove routing"
  fi
}

assert_treatment() {
  local peers_file=$1 or_file=$2 peer host
  assert_nonempty "$peers_file"
  if ! grep -Fqx -- "127.0.0.1:11434" "$peers_file"; then
    instrument_failure "treatment peer set lacks required 127.0.0.1:11434 connection"
  fi
  while IFS= read -r peer; do
    host=$(peer_host "$peer")
    if or_ip_member "$host" "$or_file"; then
      instrument_failure "treatment peer set contains OpenRouter endpoint: $peer"
    fi
  done < "$peers_file"
}

assert_control() {
  local peers_file=$1 or_file=$2 peer host
  assert_nonempty "$peers_file"
  while IFS= read -r peer; do
    host=$(peer_host "$peer")
    if or_ip_member "$host" "$or_file"; then
      return 0
    fi
  done < "$peers_file"
  instrument_failure "control peer set contains no resolved OpenRouter endpoint; treatment verdict is void"
}

case "${1:-}" in
  --classify-fixture)
    [[ $# -eq 3 ]] || usage
    classify_lsof "$2" "$3"
    exit 0
    ;;
  --assert-nonempty)
    [[ $# -eq 2 ]] || usage
    assert_nonempty "$2"
    exit 0
    ;;
  --assert-treatment)
    [[ $# -eq 3 ]] || usage
    assert_treatment "$2" "$3"
    exit 0
    ;;
  --assert-control)
    [[ $# -eq 3 ]] || usage
    assert_control "$2" "$3"
    exit 0
    ;;
esac

[[ $# -ge 3 && $# -le 4 ]] || usage
treatment_lane=$1
control_lane=$2
artifact=$3
benchmark=${4:-hello_world}
ailang_bin=${AILANG_BIN:-ailang}
timeout_secs=${PROBE_TIMEOUT_SECS:-900}
[[ "$timeout_secs" =~ ^[1-9][0-9]*$ ]] || instrument_failure "PROBE_TIMEOUT_SECS must be a positive integer"
[[ $(uname -sm) == "Darwin arm64" ]] || instrument_failure "live probe requires darwin/arm64"
command -v dig >/dev/null || instrument_failure "dig is required"
command -v lsof >/dev/null || instrument_failure "lsof is required"
command -v pgrep >/dev/null || instrument_failure "pgrep is required"
command -v jq >/dev/null || instrument_failure "jq is required"
command -v "$ailang_bin" >/dev/null || instrument_failure "AILANG_BIN is not executable or not on PATH"

if [[ -n "${OPENROUTER_API_KEY:-}" ]]; then
  echo "OPENROUTER_API_KEY: SET" >&2
else
  echo "OPENROUTER_API_KEY: UNSET" >&2
fi

tmp_dir=$(mktemp -d "${TMPDIR:-/tmp}/motoko-connection-probe.XXXXXX") || instrument_failure "could not create temporary directory"
trap 'rm -rf "$tmp_dir"' EXIT
or_file="$tmp_dir/or_ips"
dig +short openrouter.ai | awk '/^([0-9]{1,3}\.){3}[0-9]{1,3}$/ || /:/' | LC_ALL=C sort -u > "$or_file"
[[ -s "$or_file" ]] || instrument_failure "dig returned no addresses for openrouter.ai"

descendant_pids() {
  local root=$1 deadline=$2 current child
  local -a queue=("$root") result=()
  while ((${#queue[@]} > 0)); do
    (( $(date +%s) <= deadline )) || instrument_failure "process-tree discovery deadline expired"
    current=${queue[0]}
    queue=("${queue[@]:1}")
    result+=("$current")
    while IFS= read -r child; do
      [[ -n "$child" ]] && queue+=("$child")
    done < <(pgrep -P "$current" 2>/dev/null || true)
  done
  local IFS=,
  printf '%s\n' "${result[*]}"
}

sample_tree() {
  local root=$1 raw_file=$2 deadline=$3 pids
  pids=$(descendant_pids "$root" "$deadline")
  lsof -nP -iTCP -sTCP:ESTABLISHED -a -p "$pids" 2>/dev/null >> "$raw_file" || true
}

lane_start=""
lane_end=""
lane_rc=0
run_lane() {
  local lane=$1 peers_file=$2 raw_file=$3 driver_log=$4 deadline pid now terminate_deadline
  lane_rc=0
  lane_start=$(date -u +%Y-%m-%dT%H:%M:%SZ)
  : > "$raw_file"
  "$ailang_bin" eval-suite --agent --models "$lane" --benchmarks "$benchmark" --trials 1 --dry-run=false \
    > "$driver_log" 2>&1 &
  pid=$!
  deadline=$(( $(date +%s) + timeout_secs ))
  while kill -0 "$pid" 2>/dev/null; do
    now=$(date +%s)
    if (( now > deadline )); then
      kill "$pid" 2>/dev/null || true
      terminate_deadline=$(( $(date +%s) + 5 ))
      while kill -0 "$pid" 2>/dev/null; do
        (( $(date +%s) <= terminate_deadline )) || {
          kill -9 "$pid" 2>/dev/null || true
          instrument_failure "lane $lane exceeded its bounded termination deadline"
        }
        sleep 1
      done
      instrument_failure "lane $lane exceeded ${timeout_secs}s sampling deadline"
    fi
    sample_tree "$pid" "$raw_file" "$deadline"
    sleep 1
  done
  wait "$pid" || lane_rc=$?
  lane_end=$(date -u +%Y-%m-%dT%H:%M:%SZ)
  classify_lsof "$or_file" "$raw_file" | cut -f2- | LC_ALL=C sort -u > "$peers_file"
  echo "lane=$lane driver_rc=$lane_rc peers:" >&2
  jq -Rsc 'split("\n") | map(select(length > 0))' "$peers_file" >&2
}

probe_start=$(date -u +%Y-%m-%dT%H:%M:%SZ)
treatment_peers="$tmp_dir/treatment.peers"
control_peers="$tmp_dir/control.peers"
run_lane "$treatment_lane" "$treatment_peers" "$tmp_dir/treatment.lsof" "$tmp_dir/treatment.driver.log"
treatment_start=$lane_start
treatment_end=$lane_end
treatment_rc=$lane_rc
run_lane "$control_lane" "$control_peers" "$tmp_dir/control.lsof" "$tmp_dir/control.driver.log"
control_start=$lane_start
control_end=$lane_end
control_rc=$lane_rc
probe_end=$(date -u +%Y-%m-%dT%H:%M:%SZ)
platform=$(uname -sm)

jq -n \
  --arg treatment_lane "$treatment_lane" --arg control_lane "$control_lane" \
  --arg probe_start "$probe_start" --arg probe_end "$probe_end" --arg platform "$platform" \
  --arg treatment_start "$treatment_start" --arg treatment_end "$treatment_end" \
  --arg control_start "$control_start" --arg control_end "$control_end" \
  --argjson treatment_rc "$treatment_rc" --argjson control_rc "$control_rc" \
  --argjson or_ips "$(jq -Rsc 'split("\n") | map(select(length > 0))' "$or_file")" \
  --argjson treatment_peers "$(jq -Rsc 'split("\n") | map(select(length > 0))' "$treatment_peers")" \
  --argjson control_peers "$(jq -Rsc 'split("\n") | map(select(length > 0))' "$control_peers")" \
  '{platform:$platform, probe:{start:$probe_start,end:$probe_end}, OR_IPS:$or_ips,
    treatment:{lane:$treatment_lane,start:$treatment_start,end:$treatment_end,driver_rc:$treatment_rc,peers:$treatment_peers},
    control:{lane:$control_lane,start:$control_start,end:$control_end,driver_rc:$control_rc,peers:$control_peers}}' > "$artifact" ||
  instrument_failure "could not write JSON artifact"
echo "connection probe artifact: $artifact"
assert_treatment "$treatment_peers" "$or_file"
assert_control "$control_peers" "$or_file"
