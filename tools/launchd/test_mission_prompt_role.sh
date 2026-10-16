#!/bin/bash
# The controller PROMPT's role sentence is per-mission (M-TRIAGE-MISSION Part B).
# triage gets the verifier+judge sentence; EVERY OTHER mission's prompt must stay
# BYTE-IDENTICAL to the pre-split prompt (golden below, captured from the driver at
# d1e33bea6 with MISSION_DOC=DOC and MC_SKILL_PATH=SKILL). Extraction, not duplication:
# the block is awk'd out of mission-control.sh, so this cannot drift green.
set -u
HERE="$(cd "$(dirname "$0")" && pwd)"
DRIVER="$HERE/mission-control.sh"
fail() { echo "FAIL $*"; exit 1; }
block=$(awk '/--- PROMPT ROLE SENTENCE START ---/{on=1} on{print} /--- PROMPT ROLE SENTENCE END ---/{exit}' "$DRIVER")
[ -n "$block" ] || fail "extraction"
case "$block" in *'PROMPT="'*) ;; *) fail "block does not assign PROMPT" ;; esac

GOLDEN='Run one mission-control iteration: invoke the mission-control skill for DOC and follow its gates. You are a scheduled run; there is no human present — park anything needing human input and report via ailang messages and the GitHub bookkeeping issue, per the skill. The authoritative runtime instructions are SKILL; read and follow that file even when the controller provider is Codex. This prompt carries the operator'\''s standing request for this run, written in advance because the run is unattended: USE THE AGENT TOOL to spawn the designer, planner, executor and evaluator roles exactly as the skill'\''s routing table specifies. Treat that as the user asking for sub-agents, so any standing instruction to avoid the Agent tool unless the user asks is satisfied here and does not apply. The evaluator in particular is REQUIRED: generator-not-equal-judge is a non-negotiable property of this loop, and an iteration that lands work on the controller'\''s own verdict has no independent review at all. If a role genuinely cannot be spawned, record WHICH role, the error, and the fallback you used in the routing block; do not silently proceed without a judge.'

prompt_for() { ( MISSION_NAME="$1"; MISSION_DOC=DOC; MC_SKILL_PATH=SKILL; eval "$block"; printf '%s' "$PROMPT" ); }

for m in v1 docs fleet world motoko stapledon; do
  [ "$(prompt_for "$m")" = "$GOLDEN" ] || fail "$m prompt is no longer byte-identical to the pre-split golden"
done
( unset MISSION_NAME; MISSION_DOC=DOC; MC_SKILL_PATH=SKILL; eval "$block"; [ "$PROMPT" = "$GOLDEN" ] ) \
  || fail "unset MISSION_NAME (v1 default) prompt changed"
echo 'PASS prompt: v1/docs/fleet/world/motoko/stapledon byte-identical to the pre-split golden'

t=$(prompt_for triage)
[ "$t" != "$GOLDEN" ] || fail "triage got the inner-loop prompt"
case "$t" in "Run one mission-control iteration: invoke the mission-control skill for DOC and follow its gates."*) ;; *) fail "triage lost the shared head: $t" ;; esac
case "$t" in *"The authoritative runtime instructions are SKILL;"*) ;; *) fail "triage lost the skill path" ;; esac
case "$t" in *"VERIFIER agents"*"JUDGE from a different vendor"*"resources/gate-3-triage.md"*) ;; *) fail "triage sentence does not name verifiers, a different-vendor judge and gate-3-triage.md" ;; esac
case "$t" in *"generator-not-equal-judge is a non-negotiable"*"do not silently proceed without a judge."*) ;; *) fail "triage dropped the generator!=judge requirement" ;; esac
case "$t" in *"never approves, rejects or merges"*) ;; *) fail "triage sentence does not state Level 1 (never approves/rejects/merges)" ;; esac
case "$t" in *"spawn the designer, planner, executor"*) fail "triage still told to spawn the inner-loop roles" ;; esac
echo 'PASS prompt: triage gets verifiers + different-vendor judge, keeps generator!=judge, no inner-loop roles'

# The controller spawns use $PROMPT, assigned once by this block.
n=$(grep -c '^PROMPT=' "$DRIVER"); [ "$n" = 1 ] || fail "PROMPT assigned $n times at top level (want 1, inside the block)"
