#!/usr/bin/env bash
set -uo pipefail

script_dir=$(CDPATH='' cd -- "$(dirname -- "$0")" && pwd)
probe=${PROBE_UNDER_TEST:-$script_dir/motoko_connection_probe.sh}
tmp_dir=$(mktemp -d "${TMPDIR:-/tmp}/test-motoko-connection-probe.XXXXXX") || exit 1
trap 'rm -rf "$tmp_dir"' EXIT
arms=0

pass_arm() {
  arms=$((arms + 1))
  echo "ok $arms - $1"
}

require_line() {
  local name=$1 expected=$2 file=$3
  if ! grep -Fqx -- "$expected" "$file"; then
    echo "not ok - $name: missing $expected" >&2
    exit 1
  fi
}

expect_failure() {
  local name=$1 expected=$2
  shift 2
  if "$@" >"$tmp_dir/stdout" 2>"$tmp_dir/stderr"; then
    echo "not ok - $name unexpectedly succeeded" >&2
    exit 1
  fi
  if ! grep -Fq -- "$expected" "$tmp_dir/stderr"; then
    echo "not ok - $name lacked expected message: $expected" >&2
    cat "$tmp_dir/stderr" >&2
    exit 1
  fi
  pass_arm "$name"
}

expect_success() {
  local name=$1
  shift
  if ! "$@" >"$tmp_dir/stdout" 2>"$tmp_dir/stderr"; then
    echo "not ok - $name failed" >&2
    cat "$tmp_dir/stderr" >&2
    exit 1
  fi
  pass_arm "$name"
}

cat > "$tmp_dir/or_ips" <<'EOF'
203.0.113.8
2001:db8::8
EOF
cat > "$tmp_dir/lsof.fixture" <<'EOF'
COMMAND PID USER FD TYPE DEVICE SIZE/OFF NODE NAME
motoko 101 user 10u IPv4 0x0 0t0 TCP 127.0.0.1:51000->127.0.0.1:11434 (ESTABLISHED)
motoko 101 user 11u IPv4 0x0 0t0 TCP 192.0.2.10:51001->203.0.113.8:443 (ESTABLISHED)
motoko 101 user 12u IPv6 0x0 0t0 TCP [::1]:51002->[::1]:11434 (ESTABLISHED)
motoko 101 user 13u IPv4 0x0 0t0 TCP 192.0.2.10:51003->198.51.100.4:443 (ESTABLISHED)
motoko 101 user 14u IPv6 0x0 0t0 TCP [2001:db8::10]:51004->[2001:db8::8]:443 (ESTABLISHED)
motoko 101 user 15u IPv6 0x0 0t0 TCP [2001:db8::10]:51005->[2001:db8::99]:443 (ESTABLISHED)
EOF

"$probe" --classify-fixture "$tmp_dir/or_ips" "$tmp_dir/lsof.fixture" > "$tmp_dir/classified"
require_line "classification fixture" $'loopback\t127.0.0.1:11434' "$tmp_dir/classified"
require_line "classification fixture" $'openrouter\t203.0.113.8:443' "$tmp_dir/classified"
require_line "classification fixture" $'loopback\t[::1]:11434' "$tmp_dir/classified"
require_line "classification fixture" $'other\t198.51.100.4:443' "$tmp_dir/classified"
# IPv6: lsof brackets the peer, dig does not. Without normalisation an IPv6
# OpenRouter peer reads as "other" — a FALSE NEGATIVE in AC-M2-treatment's
# negative half. The second row is the over-match control.
require_line "classification fixture" $'openrouter\t[2001:db8::8]:443' "$tmp_dir/classified"
require_line "classification fixture" $'other\t[2001:db8::99]:443' "$tmp_dir/classified"
pass_arm "classification partitions loopback, OpenRouter, and other peers"

cut -f2- "$tmp_dir/classified" > "$tmp_dir/all.peers"
printf '%s\n' '127.0.0.1:11434' '198.51.100.4:443' > "$tmp_dir/treatment.peers"
printf '%s\n' '203.0.113.8:443' > "$tmp_dir/control.peers"
: > "$tmp_dir/empty.peers"

expect_success "positive treatment requires the synthetic localhost:11434 peer" \
  "$probe" --assert-treatment "$tmp_dir/treatment.peers" "$tmp_dir/or_ips"
expect_success "negative/control arm detects a peer in OR_IPS" \
  "$probe" --assert-control "$tmp_dir/control.peers" "$tmp_dir/or_ips"
expect_failure "anti-vacuity rejects an empty peer set" "empty peer set" \
  "$probe" --assert-nonempty "$tmp_dir/empty.peers"
expect_failure "treatment rejects missing localhost connection" "lacks required 127.0.0.1:11434" \
  "$probe" --assert-treatment "$tmp_dir/control.peers" "$tmp_dir/or_ips"
printf '%s\n' '127.0.0.1:11434' '203.0.113.8:443' > "$tmp_dir/leaky-treatment.peers"
expect_failure "treatment rejects an OpenRouter peer" "contains OpenRouter endpoint" \
  "$probe" --assert-treatment "$tmp_dir/leaky-treatment.peers" "$tmp_dir/or_ips"
expect_failure "control must demonstrate OpenRouter visibility" "treatment verdict is void" \
  "$probe" --assert-control "$tmp_dir/treatment.peers" "$tmp_dir/or_ips"
printf '%s\n' '127.0.0.1:11434' '[2001:db8::8]:443' > "$tmp_dir/leaky-v6.peers"
expect_failure "treatment rejects an OpenRouter peer reached over IPv6" "contains OpenRouter endpoint" \
  "$probe" --assert-treatment "$tmp_dir/leaky-v6.peers" "$tmp_dir/or_ips"

# Build a hermetic live-path toolchain. The driver is a stub: no eval, GPU, Ollama,
# or network access occurs. lsof derives the lane from the sampled driver's PID.
live_bin="$tmp_dir/live-bin"
mkdir -p "$live_bin"
for tool in awk bash cat cp cut date grep kill mkdir mktemp rm sleep sort; do
  tool_path=$(command -v "$tool") || exit 1
  ln -s "$tool_path" "$live_bin/$tool"
done
ln -s "$(command -v jq)" "$live_bin/jq"
cat > "$live_bin/uname" <<'EOF'
#!/bin/bash
echo "${PROBE_TEST_UNAME:-Darwin arm64}"
EOF
cat > "$live_bin/dig" <<'EOF'
#!/bin/bash
[[ "${PROBE_TEST_DIG_EMPTY:-0}" == 1 ]] || echo 203.0.113.8
EOF
cat > "$live_bin/pgrep" <<'EOF'
#!/bin/bash
if [[ "${PROBE_TEST_PGREP_LOOP:-0}" == 1 ]]; then
  while [[ $# -gt 1 ]]; do shift; done
  echo "$1"
fi
exit 0
EOF
cat > "$live_bin/lsof" <<'EOF'
#!/bin/bash
pid=""
while [[ $# -gt 0 ]]; do
  if [[ "$1" == -p ]]; then pid=$2; break; fi
  shift
done
lane_file="${PROBE_STUB_STATE}.${pid}"
[[ -f "$lane_file" ]] || exit 1
lane=$(cat "$lane_file")
echo "COMMAND PID USER FD TYPE DEVICE SIZE/OFF NODE NAME"
if [[ "$lane" == treatment ]]; then
  echo "stub $pid user 1u IPv4 0 0t0 TCP 127.0.0.1:51000->127.0.0.1:11434 (ESTABLISHED)"
else
  echo "stub $pid user 1u IPv4 0 0t0 TCP 192.0.2.1:51000->203.0.113.8:443 (ESTABLISHED)"
fi
EOF
cat > "$live_bin/ailang-stub" <<'EOF'
#!/bin/bash
lane=""
while [[ $# -gt 0 ]]; do
  if [[ "$1" == --models ]]; then lane=$2; break; fi
  shift
done
echo "$lane" > "${PROBE_STUB_STATE}.$$"
echo "stub driver lane=$lane"
if [[ "${PROBE_TEST_IGNORE_TERM:-0}" == 1 ]]; then trap '' TERM; fi
sleep "${PROBE_TEST_DRIVER_SLEEP:-2}"
rm -f "${PROBE_STUB_STATE}.$$"
EOF
chmod +x "$live_bin/uname" "$live_bin/dig" "$live_bin/pgrep" "$live_bin/lsof" "$live_bin/ailang-stub"

run_live() {
  env PATH="$live_bin" AILANG_BIN=ailang-stub PROBE_TIMEOUT_SECS=4 \
    PROBE_STUB_STATE="$tmp_dir/lane" "$@" /bin/bash "$probe" treatment control "$tmp_dir/live.json"
}

expect_failure "live usage rejects wrong arity" "usage:" /bin/bash "$probe"
expect_failure "classify usage rejects wrong arity" "usage:" /bin/bash "$probe" --classify-fixture
expect_failure "nonempty usage rejects wrong arity" "usage:" /bin/bash "$probe" --assert-nonempty
expect_failure "treatment usage rejects wrong arity" "usage:" /bin/bash "$probe" --assert-treatment
expect_failure "control usage rejects wrong arity" "usage:" /bin/bash "$probe" --assert-control
expect_failure "timeout rejects non-positive values" "positive integer" \
  env PROBE_TIMEOUT_SECS=0 /bin/bash "$probe" treatment control "$tmp_dir/reject.json"
expect_failure "timeout rejects non-integer values" "positive integer" \
  env PROBE_TIMEOUT_SECS=no /bin/bash "$probe" treatment control "$tmp_dir/reject.json"
expect_failure "platform gate rejects non-darwin arm64" "requires darwin/arm64" \
  run_live PROBE_TEST_UNAME='Linux x86_64'

make_dependency_path() {
  local omitted=$1 dep_dir="$tmp_dir/dep-$1" name
  mkdir -p "$dep_dir"
  for name in "$live_bin"/*; do
    [[ "${name##*/}" == "$omitted" ]] || ln -s "$name" "$dep_dir/${name##*/}"
  done
  printf '%s\n' "$dep_dir"
}
for dependency in dig lsof pgrep jq ailang-stub; do
  dependency_path=$(make_dependency_path "$dependency")
  expected_dependency="$dependency is required"
  [[ "$dependency" == ailang-stub ]] && expected_dependency="AILANG_BIN is not executable"
  expect_failure "dependency gate rejects missing $dependency" "$expected_dependency" \
    env PATH="$dependency_path" AILANG_BIN=ailang-stub PROBE_TIMEOUT_SECS=2 \
      PROBE_STUB_STATE="$tmp_dir/lane" /bin/bash "$probe" treatment control "$tmp_dir/dep.json"
done
expect_failure "empty dig result refuses the live verdict" "dig returned no addresses" \
  run_live PROBE_TEST_DIG_EMPTY=1
expect_failure "temporary-directory creation failure refuses" "could not create temporary directory" \
  run_live TMPDIR="$tmp_dir/missing/parent"
expect_failure "empty pid scope fails loudly instead of widening lsof" "invalid empty or malformed pid scope" \
  run_live PROBE_TEST_PID_SCOPE=
expect_failure "descendant discovery deadline refuses at the caller" "process-tree discovery failed" \
  run_live PROBE_TEST_DESCENDANT_FAILURE=1
expect_failure "lane sampling deadline refuses" "exceeded 1s sampling deadline" \
  run_live PROBE_TEST_DRIVER_SLEEP=10 PROBE_TIMEOUT_SECS=1
expect_failure "bounded termination deadline refuses" "bounded termination deadline" \
  run_live PROBE_TEST_DRIVER_SLEEP=20 PROBE_TEST_IGNORE_TERM=1 PROBE_TIMEOUT_SECS=1

success_artifact="$tmp_dir/success/probe.json"
mkdir -p "$tmp_dir/success"
expect_success "hermetic live success retains both lanes diagnostics" \
  env PATH="$live_bin" AILANG_BIN=ailang-stub PROBE_TIMEOUT_SECS=4 \
    PROBE_STUB_STATE="$tmp_dir/lane-success" /bin/bash "$probe" treatment control "$success_artifact"
for retained in treatment.driver.log treatment.lsof control.driver.log control.lsof; do
  [[ -s "$success_artifact.$retained" ]] || { echo "not ok - missing retained $retained" >&2; exit 1; }
done

refusal_artifact="$tmp_dir/refusal/probe.json"
mkdir -p "$tmp_dir/refusal"
expect_failure "refusing live path still retains both lanes diagnostics" "treatment verdict is void" \
  env PATH="$live_bin" AILANG_BIN=ailang-stub PROBE_TIMEOUT_SECS=4 \
    PROBE_STUB_STATE="$tmp_dir/lane-refusal" PROBE_TEST_FORCE_TREATMENT=1 \
    /bin/bash "$probe" treatment treatment "$refusal_artifact"
for retained in treatment.driver.log treatment.lsof control.driver.log control.lsof; do
  [[ -s "$refusal_artifact.$retained" ]] || { echo "not ok - refusal lost $retained" >&2; exit 1; }
done

unwritable="$tmp_dir/not-a-directory"
: > "$unwritable"
expect_failure "JSON artifact write failure refuses" "could not write JSON artifact" \
  env PATH="$live_bin" AILANG_BIN=ailang-stub PROBE_TIMEOUT_SECS=4 \
    PROBE_STUB_STATE="$tmp_dir/lane-write" /bin/bash "$probe" treatment control "$unwritable/probe.json"

if [[ $(uname -s) == Darwin ]] && command -v nc >/dev/null && command -v lsof >/dev/null; then
  port=$((42000 + ($$ % 10000)))
  socket_deadline=$(( $(date +%s) + 5 ))
  nc -l 127.0.0.1 "$port" > /dev/null 2>&1 &
  listener_pid=$!
  sleep 1
  if ! kill -0 "$listener_pid" 2>/dev/null || (( $(date +%s) > socket_deadline )); then
    kill "$listener_pid" 2>/dev/null || true
    echo "UNINFORMATIVE UNDER SANDBOX: loopback bind denied; fixture arm remains authoritative"
    listener_pid=""
  fi
  if [[ -n "$listener_pid" ]]; then
    { sleep 2; } | nc 127.0.0.1 "$port" >/dev/null 2>&1 &
    client_pid=$!
    capture_deadline=$(( $(date +%s) + 5 ))
    : > "$tmp_dir/socket.lsof"
    while kill -0 "$client_pid" 2>/dev/null; do
      lsof -nP -a -p "$client_pid" -iTCP -sTCP:ESTABLISHED >> "$tmp_dir/socket.lsof" 2>/dev/null || true
      [[ -s "$tmp_dir/socket.lsof" ]] && break
      if (( $(date +%s) > capture_deadline )); then
        break
      fi
      sleep 1
    done
    wait "$client_pid" 2>/dev/null || true
    kill "$listener_pid" 2>/dev/null || true
    wait "$listener_pid" 2>/dev/null || true
    if "$probe" --classify-fixture "$tmp_dir/or_ips" "$tmp_dir/socket.lsof" | grep -Eq "^loopback[[:space:]]+127\\.0\\.0\\.1:${port}$"; then
      pass_arm "live synthetic child connection is sampled and classified"
    else
      echo "UNINFORMATIVE UNDER SANDBOX: loopback socket sampling yielded no peer; fixture arm remains authoritative"
    fi
  fi
else
  echo "UNINFORMATIVE UNDER SANDBOX: live synthetic socket arm requires darwin nc+lsof; fixture arm remains authoritative"
fi

if (( arms == 0 )); then
  echo "not ok - zero test arms ran" >&2
  exit 1
fi
echo "PASS: $arms probe self-test arms ran"
