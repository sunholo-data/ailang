#!/usr/bin/env bash
set -uo pipefail

script_dir=$(CDPATH='' cd -- "$(dirname -- "$0")" && pwd)
probe="$script_dir/motoko_connection_probe.sh"
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
