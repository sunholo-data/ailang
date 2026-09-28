#!/bin/bash
# #1096: commits must count as work even when the executor leaves clean porcelain.
set -u

SUT="$(cd "$(dirname "$0")/../.." && pwd)/scripts/mission_pi_run.sh"
TMP=$(mktemp -d)
trap 'rm -rf "$TMP"' EXIT
mkdir -p "$TMP/node_modules/@anthropic-ai/sandbox-runtime"
echo '{}' > "$TMP/node_modules/@anthropic-ai/sandbox-runtime/package.json"
MISSION_PI_SANDBOX_NODE_MODULES="$TMP/node_modules"
MISSION_PI_SANDBOX_STAGE_PARENT="$TMP"
MISSION_PI_CLAUDE_TMP_DIR="$TMP/claude"
export MISSION_PI_SANDBOX_NODE_MODULES MISSION_PI_SANDBOX_STAGE_PARENT MISSION_PI_CLAUDE_TMP_DIR
mkdir -p "$TMP/bin"
PATH="$TMP/bin:$PATH"; export PATH
MISSION_PI_POLL_SECONDS=1; export MISSION_PI_POLL_SECONDS

passed=0; failed=0
field() { sed -n "s/.*\"$2\": \"\{0,1\}\([^\",}]*\)\"\{0,1\}.*/\1/p" "$1"; }
check() { # name expected-rc expected-verdict actual-rc verdict-file [expected-commits]
  actual_verdict=$(field "$5" verdict)
  if [ "$2" = "$4" ] && [ "$3" = "$actual_verdict" ] && \
     { [ -z "${6:-}" ] || [ "$(field "$5" commits_since_start)" = "$6" ]; }; then
    echo "PASS: $1"; passed=$((passed+1))
  else
    echo "FAIL: $1 (rc=$4 verdict=${actual_verdict:-<missing>} commits=$(field "$5" commits_since_start); expected $2/$3${6:+ commits=$6})"
    failed=$((failed+1))
  fi
}
stub() {
  { echo '#!/bin/bash'; echo 'cat >/dev/null'; echo ': > "$PI_SANDBOX_READY_FILE"'; echo "$1"; } > "$TMP/bin/pi"
  chmod +x "$TMP/bin/pi"
}
repo() { # path base|unborn
  mkdir -p "$1"
  git -C "$1" init -q
  if [ "$2" = base ]; then
    echo base > "$1/f.txt"
    git -C "$1" add f.txt
    git -C "$1" -c user.email=t@t -c user.name=t commit -qm base
  fi
}
run() { # id expected-rc expected-verdict [expected-commits] [label]
  echo directive > "$TMP/$1.directive"
  "$SUT" --model m --directive "$TMP/$1.directive" --workdir "$TMP/$1" \
    --out "$TMP/$1.ndjson" --max-seconds 30 --stall-seconds 10 >/dev/null 2>&1
  actual_rc=$?
  check "${5:-$1}" "$2" "$3" "$actual_rc" "$TMP/$1.ndjson.verdict.json" "${4:-}"
}

repo "$TMP/A" base
A_BASE=$(git -C "$TMP/A" rev-parse HEAD)
stub 'echo changed > f.txt; git add f.txt; git -c user.email=t@t -c user.name=t commit -qm one'
run A 0 ok

repo "$TMP/B" base
stub 'echo one > f.txt; git add f.txt; git -c user.email=t@t -c user.name=t commit -qm one; echo two > f.txt; git add f.txt; git -c user.email=t@t -c user.name=t commit -qm two'
run B 0 ok

repo "$TMP/C" base
stub ':'
run C 10 empty_worktree

repo "$TMP/D" base
stub 'echo changed > f.txt'
run D 0 ok

repo "$TMP/E" unborn
stub 'echo first > f.txt; git add f.txt; git -c user.email=t@t -c user.name=t commit -qm first'
run E 0 ok

repo "$TMP/F" base
stub 'git -c user.email=t@t -c user.name=t commit --amend -qm x'
run F 0 ok 1 'F (amend counted as a commit)'

repo "$TMP/H" base
echo second > "$TMP/H/f.txt"
git -C "$TMP/H" add f.txt
git -C "$TMP/H" -c user.email=t@t -c user.name=t commit -qm second
stub 'git reset -q --hard HEAD~1'
run H 10 empty_worktree 0

repo "$TMP/I" base
stub 'rm -rf .git'
run I 10 empty_worktree

A_JSON="$TMP/A.ndjson.verdict.json"
B_JSON="$TMP/B.ndjson.verdict.json"
C_JSON="$TMP/C.ndjson.verdict.json"
D_JSON="$TMP/D.ndjson.verdict.json"
A_HEAD=$(git -C "$TMP/A" rev-parse HEAD)
if [ "$(field "$A_JSON" base_head)" = "$A_BASE" ] && \
   [ "$(field "$A_JSON" head_after)" = "$A_HEAD" ] && \
   [ "$(field "$A_JSON" commits_since_start)" = 1 ] && \
   [ "$(field "$B_JSON" commits_since_start)" = 2 ] && \
   [ "$(field "$C_JSON" commits_since_start)" = 0 ] && \
   [ "$(field "$D_JSON" commits_since_start)" = 0 ]; then
  echo 'PASS: G (JSON heads and commit counts)'; passed=$((passed+1))
else
  echo 'FAIL: G (JSON heads or commit counts)'; failed=$((failed+1))
fi

echo "passed=$passed failed=$failed"
[ "$failed" -eq 0 ]
