#!/bin/bash
# Exercise the actual driver with a foreign project and counted fake providers.
set -eu
HERE=$(cd "$(dirname "$0")" && pwd)
TMP=$(mktemp -d)
trap 'rm -rf "$TMP"' EXIT
mkdir -p "$TMP/home/.local/bin" "$TMP/home/.ailang/state" "$TMP/project"
export TRACE="$TMP/trace"
for provider in claude codex pi; do
  cat > "$TMP/home/.local/bin/$provider" <<'SH'
#!/bin/sh
printf 'provider\n' >> "$TRACE"
exit 88
SH
  chmod +x "$TMP/home/.local/bin/$provider"
done
cat > "$TMP/home/.local/bin/ailang" <<'SH'
#!/bin/sh
printf '%s|%s|%s\n' "$PWD" "$*" "$AILANG_MISSION_REGISTRY" >> "$TRACE"
exit 3
SH
cat > "$TMP/home/.local/bin/sysctl" <<'SH'
#!/bin/sh
printf 'kern.boottime: { sec = 1, usec = 0 }\n'
SH
cat > "$TMP/home/.local/bin/vm_stat" <<'SH'
#!/bin/sh
printf 'Mach Virtual Memory Statistics: (page size of 16384 bytes)\nPages free: 9000000.\nPages occupied by compressor: 0.\n'
SH
chmod +x "$TMP/home/.local/bin/ailang" "$TMP/home/.local/bin/sysctl" "$TMP/home/.local/bin/vm_stat"
printf '{}' > "$TMP/item.json"
set +e
HOME="$TMP/home" MISSION_NAME=iteration-fixture MISSION_WORKDIR="$TMP/project" \
AILANG_DRIVER_PIN=0 AILANG_MISSION_WORK_ITEM="$TMP/item.json" \
MISSION_MEM_WAIT=0 /bin/bash "$HERE/mission-control.sh" > "$TMP/output" 2>&1
rc=$?
set -e
if [ "$rc" != 3 ]; then cat "$TMP/output"; echo "FAIL expected binary waiting exit 3 got $rc"; exit 1; fi
[ "$(wc -l < "$TRACE" | tr -d ' ')" = 1 ] || { cat "$TRACE"; exit 1; }
grep -F "$TMP/project|mission iterate --work-item $TMP/item.json|" "$TRACE" >/dev/null
grep -F "$HERE/../../missions" "$TRACE" >/dev/null || grep -F "$(cd "$HERE/../.." && pwd)/missions" "$TRACE" >/dev/null
# Kill switch must prevent binary invocation as well as legacy providers.
: > "$TRACE"
touch "$TMP/home/.ailang/state/mission-iteration-fixture.disabled"
HOME="$TMP/home" MISSION_NAME=iteration-fixture MISSION_WORKDIR="$TMP/project" \
AILANG_DRIVER_PIN=0 AILANG_MISSION_WORK_ITEM="$TMP/item.json" \
/bin/bash "$HERE/mission-control.sh" > "$TMP/output" 2>&1
[ ! -s "$TRACE" ] || { cat "$TRACE"; exit 1; }
echo 'PASS binary iteration: foreign project, one call, no legacy probes/retry, waiting code, kill switch'
