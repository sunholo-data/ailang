#!/bin/bash
# mission_pi_run.sh must type a provider capacity refusal (verdict provider_quota, rc 19) and record
# provider_errors / provider_error on every verdict. pi exits 0 on a provider refusal, so the runner
# has to read the assistant message_end (stopReason "error" + errorMessage) itself.
#
# Fixtures (tools/launchd/testdata/pi-ndjson/) are REAL pi output, whole lines, no byte edited:
#   quota-429-ollama-session-limit  World iter-187 planner run, ~/.ailang/state/mission-world-iter187-evidence/planner.ndjson
#       (sha256 2e2082d7…60bd, 1208 lines): awk 'NR<=3 || (NR>=1170 && NR<=1179) || (NR>=1181 && NR<=1184) || NR>=1186'
#       (lines 1180 and 1185 dropped for size). sha256 73e2f039…ef0c, 40 lines.
#   quota-then-success-spliced      same source NR<=3, 1182-1184, 1186-1188, then probe-success NR==8 || NR>=16.
#       Line SELECTION is synthetic (pi's retry-then-recover shape), every line verbatim. sha256 e6310349…62fd.
#   error-401-openrouter-bogus-key  2026-10-01: pi --mode json --no-session --no-tools
#       --model openrouter/deepseek/deepseek-v4-flash-0731 --api-key sk-or-v1-bogus-capture -p ok
#       (zero tokens). sha256 2db2bbce…edd.
#   error-423-ollama-rig-lease      2026-10-01: same command with --model ollama/no-such-model-capture:latest. sha256 0e2a9148…c597.
#   success-openrouter-probe        2026-10-01 real success probe (minimax via openrouter). sha256 f95d67db…ef06.
# Raw captures: ~/.ailang/state/fleet-iter10-pi-error-captures/. NDJSON cannot carry comments, so provenance lives here.
# The stub emits fixtures with `cat`, never printf/echo, so backslashes in errorMessage survive.
set -u
ROOT=$(cd "$(dirname "$0")/../.." && pwd -P)
SUT="${MISSION_PI_TEST_SUT:-$ROOT/scripts/mission_pi_run.sh}"
FX="$ROOT/tools/launchd/testdata/pi-ndjson"
TMP=$(mktemp -d)
trap 'rm -rf "$TMP"' EXIT
mkdir -p "$TMP/bin" "$TMP/node_modules/@anthropic-ai/sandbox-runtime"
echo '{}' > "$TMP/node_modules/@anthropic-ai/sandbox-runtime/package.json"
export MISSION_PI_SANDBOX_NODE_MODULES="$TMP/node_modules"
export MISSION_PI_SANDBOX_STAGE_PARENT="$TMP"
export MISSION_PI_CLAUDE_TMP_DIR="$TMP/claude"
export MISSION_PI_POLL_SECONDS=1
PATH="$TMP/bin:$PATH"; export PATH
printf 'directive\n' > "$TMP/directive"

cat > "$TMP/bin/pi" <<'STUB'
#!/bin/bash
cat >/dev/null
: > "$PI_SANDBOX_READY_FILE"
[ "${PI_TEST_DIRTY:-0}" = 1 ] && echo changed > f.txt
cat "$PI_TEST_FIXTURE"
STUB
chmod +x "$TMP/bin/pi"

git -C "$TMP" init -q main
echo base > "$TMP/main/f.txt"
git -C "$TMP/main" add f.txt
git -C "$TMP/main" -c user.email=t@t -c user.name=t commit -qm base

passed=0; failed=0
check() { # label condition
  if [ "$2" = true ]; then echo "PASS: $1"; passed=$((passed+1));
  else echo "FAIL: $1"; failed=$((failed+1)); fi
}
field() { jq -r "$2" "$1" 2>/dev/null; }
run() { # id fixture dirty
  id="$1"; PI_TEST_FIXTURE="$2"; PI_TEST_DIRTY="$3"
  export PI_TEST_FIXTURE PI_TEST_DIRTY
  git -C "$TMP/main" worktree add -q --detach "$TMP/wt-$id" HEAD
  "$SUT" --model m --directive "$TMP/directive" --workdir "$TMP/wt-$id" \
    --out "$TMP/$id.ndjson" --max-seconds 30 --stall-seconds 10 >/dev/null 2>"$TMP/$id.stderr"
  actual_rc=$?
  verdict="$TMP/$id.ndjson.verdict.json"
}
is_verdict() { [ "$actual_rc" -eq "$1" ] && [ "$(field "$verdict" .verdict)" = "$2" ]; }
ok() { "$@" && echo true || echo false; }

# Precondition: a missing or empty fixture must be a loud FAIL, not a quiet pass of every arm.
for f in quota-429-ollama-session-limit quota-then-success-spliced error-401-openrouter-bogus-key \
         error-423-ollama-rig-lease success-openrouter-probe; do
  check "fixture $f has a message_end line" "$(ok grep -q '"type":"message_end"' "$FX/$f.ndjson")"
done

Q="$FX/quota-429-ollama-session-limit.ndjson"
E401="$FX/error-401-openrouter-bogus-key.ndjson"

# Derived inputs, produced by jq/awk from the real fixtures at test time.
TICKET_MSG='429: {"message":"you (marked) have reached your weekly usage limit, upgrade for higher limits: https://ollama.com/upgrade or add usage credits: https://ollama.com/settings (ref: …)","type":"api_error","param":null,"code":null}'
jq -c --arg m "$TICKET_MSG" 'if .type=="message_end" and .message.role=="assistant" and .message.stopReason=="error" then .message.errorMessage=$m else . end' "$E401" > "$TMP/in-ticket.ndjson"
last_me=$(grep -n '"type":"message_end"' "$Q" | tail -1 | cut -d: -f1)
awk -v n="$last_me" 'NR==n { print substr($0, 1, 400); next } { print }' "$Q" > "$TMP/in-trunc-last.ndjson"
awk '/"type":"message_end"/ && /"stopReason":"error"/ && !done { i = index($0, "\"stopReason\":\"error\""); print substr($0, 1, i + length("\"stopReason\":\"error\"") - 1); done=1; next } { print }' "$E401" > "$TMP/in-unparsed.ndjson"

# A: clean tree, 4 x 429 usage limit
run A "$Q" 0
check 'A verdict provider_quota rc 19' "$(ok is_verdict 19 provider_quota)"
check 'A provider_errors == 4' "$(ok [ "$(field "$verdict" .provider_errors)" = 4 ])"
pe=$(field "$verdict" .provider_error)
check 'A provider_error is the LAST 429 usage-limit text' "$(ok sh -c '
  case "$1" in 429:*) ;; *) exit 1 ;; esac
  case "$1" in *"usage limit"*) ;; *) exit 1 ;; esac
  case "$1" in *8954f740*) ;; *) exit 1 ;; esac
  [ "${#1}" -le 300 ]' sh "$pe")"
check 'A worktree_changed_files == 0' "$(ok [ "$(field "$verdict" .worktree_changed_files)" = 0 ])"
# B: same, dirty tree — truncated work is still quota, not ok
run B "$Q" 1
check 'B dirty tree still provider_quota rc 19' "$(ok is_verdict 19 provider_quota)"
check 'B worktree_changed_files >= 1' "$(ok [ "$(field "$verdict" .worktree_changed_files)" -ge 1 ])"
# C: quota then a later successful assistant message — recovered, not quota
run C "$FX/quota-then-success-spliced.ndjson" 0
check 'C recovered run is empty_worktree rc 10' "$(ok is_verdict 10 empty_worktree)"
check 'C provider_errors == 1' "$(ok [ "$(field "$verdict" .provider_errors)" = 1 ])"
check 'C provider_error still recorded' "$(ok sh -c 'case "$1" in *"usage limit"*) exit 0 ;; *) exit 1 ;; esac' sh "$(field "$verdict" .provider_error)")"
# D: 401 is not capacity
run D "$E401" 0
check 'D 401 stays empty_worktree rc 10' "$(ok is_verdict 10 empty_worktree)"
check 'D provider_errors == 1' "$(ok [ "$(field "$verdict" .provider_errors)" = 1 ])"
check 'D provider_error starts 401:' "$(ok sh -c 'case "$1" in 401:*) exit 0 ;; *) exit 1 ;; esac' sh "$(field "$verdict" .provider_error)")"
# E: 423 rig lease is not provider quota
run E "$FX/error-423-ollama-rig-lease.ndjson" 0
check 'E 423 rig lease stays rc 10' "$(ok is_verdict 10 empty_worktree)"
check 'E provider_errors == 1' "$(ok [ "$(field "$verdict" .provider_errors)" = 1 ])"
# F: success carries the fields with zero values
run F "$FX/success-openrouter-probe.ndjson" 0
check 'F success rc 10 (no diff)' "$(ok is_verdict 10 empty_worktree)"
check 'F fields present and zero' "$(ok sh -c '[ "$1" = true ] && [ "$2" = 0 ] && [ "$3" = "" ]' sh "$(field "$verdict" 'has("provider_errors") and has("provider_error")')" "$(field "$verdict" .provider_errors)" "$(field "$verdict" .provider_error)")"
# F2: preflight_fail JSON carries the fields too
"$SUT" --model m --directive "$TMP/absent-directive" --workdir "$TMP/main" --out "$TMP/f2.ndjson" >/dev/null 2>&1
f2rc=$?; v2="$TMP/f2.ndjson.verdict.json"
check 'F2 preflight launch_failed rc 14 with zeroed fields' "$(ok sh -c '[ "$1" -eq 14 ] && [ "$2" = launch_failed ] && [ "$3" = 0 ] && [ "$4" = true ] && [ "$5" = "" ]' sh "$f2rc" "$(field "$v2" .verdict)" "$(field "$v2" .provider_errors)" "$(field "$v2" 'has("provider_error")')" "$(field "$v2" .provider_error)")"
# G: the ticket's own text
run G "$TMP/in-ticket.ndjson" 0
check 'G ticket weekly-limit text is provider_quota rc 19' "$(ok is_verdict 19 provider_quota)"
check 'G provider_error contains weekly usage limit' "$(ok sh -c 'case "$1" in *"weekly usage limit"*) exit 0 ;; *) exit 1 ;; esac' sh "$(field "$verdict" .provider_error)")"
# H: killed write — last line truncated
run H "$TMP/in-trunc-last.ndjson" 0
check 'H truncated last line still provider_quota rc 19' "$(ok is_verdict 19 provider_quota)"
check 'H provider_errors == 3' "$(ok [ "$(field "$verdict" .provider_errors)" = 3 ])"
# I: error line that greps as an error but is not JSON
run I "$TMP/in-unparsed.ndjson" 0
check 'I unparsed error is rc 10 not quota' "$(ok is_verdict 10 empty_worktree)"
check 'I provider_errors == 1 and (unparsed provider error)' "$(ok sh -c '[ "$1" = 1 ] && [ "$2" = "(unparsed provider error)" ]' sh "$(field "$verdict" .provider_errors)" "$(field "$verdict" .provider_error)")"
check 'I stderr carries WARNING' "$(ok grep -q WARNING "$TMP/I.stderr")"

echo "passed=$passed failed=$failed"
[ "$failed" -eq 0 ]
