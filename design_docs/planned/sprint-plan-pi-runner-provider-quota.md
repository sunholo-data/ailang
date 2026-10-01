# Sprint plan: pi runner types a provider quota refusal (ticket `pi-runner:quota-429-reported-as-empty-worktree`)

**One milestone (M1). Mechanical fix, no design doc: this plan is the spec.**
**Executor: work only in `/Users/voightkampff/.ailang-driver-pin/fleet-iter10-quota429`**
(branch `fleet/iter10-pi-provider-quota`, base origin/dev `a03ec7013`). Do not commit; the controller commits.
Fleet queue P1 #4, filed by World iter 196.

## Problem

`scripts/mission_pi_run.sh` reads no provider error at all. At HEAD,
`grep -n 'stopReason\|errorMessage' scripts/mission_pi_run.sh` hits only the header comment (line 54).
pi exits **0** when the provider refuses a request. The refusal arrives as an assistant
`message_end` with `stopReason:"error"` plus an `errorMessage`. pi itself retries a 429 three times:
in pi-ai `dist/utils/retry.js`, `"429"` is in `RETRYABLE_PROVIDER_ERROR_PATTERN` and the default is
`maxRetries 3`. So a quota stop shows up as 4 error messages and 4 `agent_end` events, then an
`auto_retry_end {"success":false}`. The runner's verdict therefore depends only on the worktree:

| Real run | Provider errors | Banked verdict | Should be |
|---|---|---|---|
| World iter 196 executor (the ticket) | 4 × `429 … weekly usage limit` | `empty_worktree` rc 10 | `provider_quota` rc 19 |
| World iter 187 planner, `~/.ailang/state/mission-world-iter187-evidence/planner.ndjson` | 4 × `429 … session usage limit` | **`ok` rc 0**, 10 changed files | `provider_quota` rc 19 |

The second row was found while planning. It is the more dangerous shape: a quota-truncated run that
had already written files reads as **success**.

## Measured facts this plan relies on (re-verified 2026-10-01)

- pi is `0.85.1` at `/opt/homebrew/lib/node_modules/@earendil-works/pi-coding-agent`. The ticket's
  `@mariozechner` path does not exist.
- Field shape: `node_modules/@earendil-works/pi-ai/dist/types.d.ts:287`
  `StopReason = "pending"|"stop"|"length"|"toolUse"|"error"|"aborted"|"deferred"`, and `:319-321`
  `stopReason: StopReason; … errorMessage?: string`. On the wire it is
  `{"type":"message_end","message":{"role":"assistant",…,"stopReason":"error",…,"errorMessage":"429: {…}"}}`.
- `message_end` is ALSO emitted for `role` `user`, `toolResult` and `custom` messages. "The last
  message" therefore means **the last `message_end` whose `.message.role == "assistant"`**.
- A real error event can be provoked at zero token cost. This was run on 2026-10-01, bounded at 60 s:
  `pi --mode json --no-session --no-tools --model openrouter/deepseek/deepseek-v4-flash-0731 --api-key sk-or-v1-bogus-capture -p ok`
  gave rc 0, 10 lines, and `stopReason:"error"`, `errorMessage:"401: {\"message\":\"User not found.\",\"code\":401}"`.
  A second real invocation, `--model ollama/no-such-model-capture:latest`, gave rc 0 and
  `errorMessage:"423: {\"message\":\"no lease token; rig held by …\",\"type\":\"rig_lease\"}"`. That is the rig
  gateway's lease refusal, not a provider quota.
- The capacity regex below was checked with jq 1.7 `test($re; "i")`. It matches
  `429: {…}`, `402: x`, `Insufficient_Quota` and `…Rate Limit`. It does NOT match `4290: x`, `x 429 y`,
  `401: {…}` or `500: internal`.
  NOTE: pass the regex with a SINGLE backslash (`'^\s*(402|429)\b|…'` via `--arg`). A doubled `\\s`
  in single quotes compiles to a literal backslash and silently disables the status-code arm.
- `jq -Rn '[inputs|fromjson?]'` tolerates a trailing `\r` and skips a truncated line instead of
  aborting. A slurped `jq -s` over the file aborts on the first bad line and reports nothing.
- No driver code routes on these rcs. `grep -n mission_pi_run tools/launchd/mission-control.sh` finds
  comments only, and `test_spawn_pin_hook.sh:230` uses `"tool_hang rc 18"` as evidence text only. Without the
  skill edit, rc 19 already falls under "anything non-zero except 18 is a LANE FAILURE → fall back",
  which is the desired routing. The skill edit adds only the capacity-park classification.
- `scripts/test_mission_pi_run.sh` is **not** run by `make test-launchd-drivers`, by any make target or
  by CI (`grep -rn test_mission_pi_run Makefile make .github` finds only the two `tools/launchd/` files).
  New tests therefore go in a NEW wired file, not there.
- Fleet scope guard (`tools/launchd/githooks/pre-push`, `_scope_verdict fleet`): every path in this plan
  is allowed **except `.agents/skills/mission-control/resources/gate-3-route.md`**, which is refused
  ("outside the fleet allowlist"). Yet `tools/launchd/test_agents_skills_sync.sh` (part of the done gate)
  fails if `.claude/skills/mission-*` and `.agents/skills/mission-*` differ. See **Landing** below.

## Design decisions

1. **New verdict `provider_quota`, rc 19.** Trigger: the run `finished`, AND the last assistant
   `message_end` has `stopReason == "error"`, AND its `errorMessage` matches (case-insensitively)
   `^\s*(402|429)\b|usage.?limit|quota|rate.?limit|insufficient.?credits|too many requests`.
   (`insufficient_quota` is covered by `quota`.)
2. **Do not fire when quota errors occurred but the run ended on a later non-error assistant message.**
   In that case pi's own retry, or the next turn, recovered: the provider did not truncate the work, and a
   transient 429 throttle that pi absorbed is normal operation. The runner records the count
   (`provider_errors`) and the text (`provider_error`) on the normal verdict, so a controller can still
   see it.
3. **Precedence inside `finished`:** `sandbox_not_ready` 17 → **`provider_quota` 19** → `launch_failed` 14 →
   `ok` 0 → `empty_worktree` 10. A quota stop with a non-empty diff is still 19, because the work is
   truncated. `worktree_changed_files` and `commits_since_start` stay in the JSON so the partial can be
   measured (the same idea as gate-3 `wall_timeout` rule (b)). The killed outcomes (`reasoning_stall`,
   `stream_dead`, `tool_hang`, `wall_timeout`) are unchanged. They are decided by the clock, and no
   measurement shows a quota stop presenting as one: pi returns the 429 promptly and exits.
4. **Fields on EVERY verdict, including `preflight_fail`'s minimal JSON:** `provider_errors` (int, the
   count of assistant `message_end` with `stopReason=="error"`) and `provider_error` (the errorMessage
   of the LAST such message, even if a later message succeeded, cut to its first 300 characters, then
   JSON-escaped with `jq -Rs .` the way `hung_tool` is). Values are `0` / `""` when absent.
5. **Anti-vacuity:** if any `message_end` line contains `"stopReason":"error"` but the parser extracted 0,
   set `provider_errors` to that grep count and `provider_error` to `"(unparsed provider error)"`, write
   a stderr WARNING, and do NOT classify it as quota (it cannot be classified). This mirrors
   `hung_tool`'s `"(unparsed tool call)"`.
6. **Out of scope, deliberately:** the 423 `rig_lease` refusal is not a provider quota, and its remedy
   (wait for the rig lease) is different. It stays a normal verdict and a test pins that. If the
   controller wants it typed, that is a separate ticket.

## Files in scope (touch nothing else)

1. `scripts/mission_pi_run.sh`
2. `tools/launchd/test_mission_pi_run_provider_quota.sh` (new)
3. `tools/launchd/testdata/pi-ndjson/*.ndjson` (new, 5 fixtures, generated in Task 0, never hand-edited)
4. `make/test.mk` (one line)
5. `changelogs/unreleased/2026-10-01-pi-runner-provider-quota.md` (new)
6. `.claude/skills/mission-control/resources/gate-3-route.md`, plus its mirror
   `.agents/skills/mission-control/resources/gate-3-route.md`, which is written ONLY by running
   `/bin/bash tools/launchd/sync-agents-skills.sh` and never edited by hand

Bash 3.2.57: no `declare -A`, no `${v,,}`, no `timeout`, no `mapfile`. Bound every command you run with
a `date +%s` deadline.

## M1 tasks

### Task 0: generate the fixtures from real pi output (no hand typing)

The fixtures come from three sources: one real quota run, two real error captures (2026-10-01) and one
real success probe. The raw captures are banked in `~/.ailang/state/fleet-iter10-pi-error-captures/`,
together with their `.stderr` files. Run exactly the following from the worktree root:

```bash
S="$HOME/.ailang/state/mission-world-iter187-evidence/planner.ndjson"
C="$HOME/.ailang/state/fleet-iter10-pi-error-captures"
D=tools/launchd/testdata/pi-ndjson
mkdir -p "$D"
# Real quota run: header + the last toolUse turn + the 4 x 429 tail. Whole-line selection only, with no byte
# edited. Line 1180 (a 31 KB turn_end) and line 1185 (an 846 KB agent_end echoing the whole conversation)
# are dropped for size.
awk 'NR<=3 || (NR>=1170 && NR<=1179) || (NR>=1181 && NR<=1184) || NR>=1186' "$S" > "$D/quota-429-ollama-session-limit.ndjson"
# Splice of two real streams (pi's retry-then-recover shape): the first 429 + auto_retry_start + restart,
# then a real successful assistant turn. The line SELECTION is synthetic; every line is verbatim.
{ awk 'NR<=3 || (NR>=1182 && NR<=1184) || (NR>=1186 && NR<=1188)' "$S"
  awk 'NR==8 || NR>=16' "$C/probe-success-minimax.ndjson"; } > "$D/quota-then-success-spliced.ndjson"
cp "$C/openrouter-401-bogus-key.ndjson" "$D/error-401-openrouter-bogus-key.ndjson"
cp "$C/ollama-423-rig-lease.ndjson"     "$D/error-423-ollama-rig-lease.ndjson"
cp "$C/probe-success-minimax.ndjson"    "$D/success-openrouter-probe.ndjson"
shasum -a 256 "$S" "$D"/*.ndjson
```

Acceptance: the hashes equal this table (the planner produced the same bytes on 2026-10-01).

| File | sha256 | lines |
|---|---|---|
| source `planner.ndjson` | `2e2082d71d230d49b98afb948244da4e3c1e2da809b78b2d563a1154062c60bd` | 1208 |
| `error-401-openrouter-bogus-key.ndjson` | `2db2bbced0a8363f7cea5055be1318404aef12203b8564cbd8cb26f29c8ededd` | 10 |
| `error-423-ollama-rig-lease.ndjson` | `0e2a91481e66cd7669db1a760ffa365f4bab4fa2ce125aec92c446a3b280c597` | 10 |
| `quota-429-ollama-session-limit.ndjson` | `73e2f03928f95cbe4cc1c5c3824311b103754744486f78f1caa47a4b1d65ef0c` | 40 |
| `quota-then-success-spliced.ndjson` | `e63103494c4dacc6054b61cc7155513379c596fd300b0c0ca41f609c0862cfd9` | 14 |
| `success-openrouter-probe.ndjson` | `f95d67db55ca265ead9225fef087c1374cc7e1a958a8ffe65c967019cbccef06` | 19 |

If your sandbox cannot read `~/.ailang/state/`, STOP and say so. The controller runs Task 0 verbatim
outside the sandbox. Do not substitute typed JSON. If the bank has been cleaned, re-capture the 401
with the bogus-key command in "Measured facts" (zero tokens) and record the new hash. Put the
provenance (source path, line selection, capture command, date) in the test file's header comment.
NDJSON cannot carry comments, and a `#` line would be banked as a parse-skipped line.

Transformations between pi and the parser, and how each is pinned:
1. The runner's awk filter, which drops `message_update` and passes every other line verbatim. The fixtures
   go through it because the stub `cat`s them, and `success-openrouter-probe.ndjson` still carries
   7 `message_update` lines.
2. Shell quoting. The stub must emit fixtures with `cat '<abs path>'`, never `printf`/`echo` of the
   content, because the backslash escapes inside `errorMessage` would be mangled.
3. VCS checkout. `* text=auto` could CRLF the fixtures on a Windows checkout. `.gitattributes` is outside
   the fleet allowlist, so it is not pinned there. The suite runs only on macOS (ci.yml
   `runs-on: macos-latest`), and the parser tolerates `\r` (`fromjson?`).
4. A truncated last line from a killed write. Arm H pins it.

### Task 1: `scripts/mission_pi_run.sh`

a. **Header exit-code table (after line 70, the `tool_hang` entry)**, add:
```
#   19 provider_quota   — pi finished, but its LAST assistant message is a provider refusal on
#                          capacity (HTTP 429/402, usage limit, quota, rate limit, credits). pi
#                          exits 0 on these. The MODEL did not fail and the work is truncated
#                          even if files changed. See provider_error / provider_errors in the verdict JSON.
```
b. **`preflight_fail` (line 130-134):** add `provider_errors:0, provider_error:""` to the jq object.
c. **After line 373 (`HUNG_TOOL_JSON=…`)**, insert the extraction. Keep the shape below; wording of
   comments is yours, and keep them short with no war story beyond one measured line:
```bash
# Provider refusals: pi exits 0 and reports them as an assistant message_end with stopReason "error".
# Measured: World iter-187 planner, 4 x "429 … usage limit", banked ok rc 0 with 10 files changed.
# Per-line fromjson? so a truncated line (killed write) cannot zero the count.
PROVIDER_CAPACITY_RE='^\s*(402|429)\b|usage.?limit|quota|rate.?limit|insufficient.?credits|too many requests'
PERR=$(grep '"type":"message_end"' "$OUT" 2>/dev/null | jq -cRn --arg re "$PROVIDER_CAPACITY_RE" '
  [inputs | fromjson? | .message? | select(type == "object" and .role == "assistant")] as $a
  | [$a[] | select(.stopReason == "error")] as $e
  | ($a | last) as $l
  | {n: ($e | length),
     last: ((($e | last) // {}) | .errorMessage // "" | .[0:300]),
     quota: (($l != null) and ($l.stopReason == "error") and (($l.errorMessage // "") | test($re; "i")))}' 2>/dev/null)
PROVIDER_ERRORS=$(printf '%s' "$PERR" | jq -r '.n // 0' 2>/dev/null); case "$PROVIDER_ERRORS" in ''|*[!0-9]*) PROVIDER_ERRORS=0 ;; esac
PROVIDER_ERROR=$(printf '%s' "$PERR" | jq -r '.last // ""' 2>/dev/null)
PROVIDER_QUOTA=$(printf '%s' "$PERR" | jq -r '.quota // false' 2>/dev/null); [ "$PROVIDER_QUOTA" = true ] || PROVIDER_QUOTA=false
_err_lines=$(grep '"type":"message_end"' "$OUT" 2>/dev/null | grep -c '"stopReason":"error"' | tr -d ' ')
if [ "${_err_lines:-0}" -gt 0 ] && [ "$PROVIDER_ERRORS" -eq 0 ]; then   # anti-vacuity
  PROVIDER_ERRORS="$_err_lines"; PROVIDER_ERROR="(unparsed provider error)"; PROVIDER_QUOTA=false
  echo "pi lane verdict: WARNING — $_err_lines message_end line(s) carry stopReason error but none parsed" >&2
fi
PROVIDER_ERROR_JSON=$(printf '%s' "$PROVIDER_ERROR" | jq -Rs .)
```
d. **`case "$OUTCOME"` → `finished)` (line 376-380):** insert
   `elif [ "$PROVIDER_QUOTA" = true ]; then VERDICT_NAME="provider_quota"; RC=19`
   directly after the `sandbox_not_ready` line and before `launch_failed`. Do not touch the other arms.
e. **Verdict heredoc:** after `"hung_tool": $HUNG_TOOL_JSON,` add
   `"provider_errors": ${PROVIDER_ERRORS:-0},` and `"provider_error": $PROVIDER_ERROR_JSON,`.
f. **stderr, after the `HUNG_TOOL` echo (line 412)**, before `exit "$RC"`:
```bash
[ "$RC" -eq 19 ] && echo "pi lane verdict: the PROVIDER refused on capacity — park the lane, not the model: $PROVIDER_ERROR" >&2
[ "$RC" -ne 19 ] && [ "$PROVIDER_ERRORS" -gt 0 ] && echo "pi lane verdict: $PROVIDER_ERRORS provider error(s) during the run; last: $PROVIDER_ERROR" >&2
```
   (`exit "$RC"` stays the last line, so a false `&&` cannot leak into the exit code.)

Acceptance: `/bin/bash -n scripts/mission_pi_run.sh` rc 0. No new `declare -A` / `${v,,}` / `timeout`
(`grep -nE 'declare -A|\$\{[a-zA-Z_]+,,\}|(^|[^_a-z])timeout ' scripts/mission_pi_run.sh` finds no new
hit). The file stays under 500 lines.

### Task 2: `tools/launchd/test_mission_pi_run_provider_quota.sh` (new, `#!/bin/bash`, `set -u`)

Reuse the harness of `test_mission_pi_run_sandbox.sh` / `_commits.sh`: TMP with fake
`node_modules/@anthropic-ai/sandbox-runtime/package.json`, `MISSION_PI_SANDBOX_NODE_MODULES`,
`MISSION_PI_SANDBOX_STAGE_PARENT=$TMP`, `MISSION_PI_CLAUDE_TMP_DIR`, `MISSION_PI_POLL_SECONDS=1`, and a
`pi` stub on `PATH` that does `cat >/dev/null; : > "$PI_SANDBOX_READY_FILE"`, then optionally dirties
`f.txt`, then `cat '<fixture>'`. Use a git repo with one base commit per arm. Read fields with `jq -r`.
Start with a precondition: every fixture exists and contains at least one `"type":"message_end"` line,
and a missing fixture is a loud FAIL. Print `passed=N failed=M`, then `[ "$failed" -eq 0 ]`.

Derived inputs are produced AT TEST TIME by jq from the real fixtures, never typed:
- **G input:** the 401 fixture with ONLY the assistant error's `errorMessage` replaced by the ticket's
  text, using
  `jq -c --arg m "$TICKET_MSG" 'if .type=="message_end" and .message.role=="assistant" and .message.stopReason=="error" then .message.errorMessage=$m else . end'`.
  `TICKET_MSG='429: {"message":"you (marked) have reached your weekly usage limit, upgrade for higher limits: https://ollama.com/upgrade or add usage credits: https://ollama.com/settings (ref: …)","type":"api_error","param":null,"code":null}'`
  (quoted from the ticket, which quotes World iter 196's NDJSON).
- **H input:** the quota fixture with its LAST `"type":"message_end"` line cut to its first 400 bytes
  (simulating a killed write), using awk on line numbers.
- **I input:** the 401 fixture with its assistant error `message_end` line cut just after
  `"stopReason":"error"`, so the line still greps as an error but is not valid JSON.

| Arm | Fixture | Tree | Expect | Mutation that must turn it red (revert after) |
|---|---|---|---|---|
| A | quota-429 | clean | rc 19 `provider_quota`; `provider_errors`==4; `provider_error` starts `429:`, contains `usage limit` and the LAST ref `8954f740`; length ≤300; `worktree_changed_files`==0 | (1) delete the `provider_quota` `elif` → rc 10. (2) take `$e \| first` for `last` → the ref check fails (first ref is `00e50a9f`) |
| B | quota-429 | stub writes `f.txt` | rc 19 (NOT 0); `worktree_changed_files`≥1 | move the `provider_quota` `elif` BELOW the `ok` branch → rc 0 (A stays green, which isolates precedence) |
| C | quota-then-success-spliced | clean | rc 10 `empty_worktree`; `provider_errors`==1; `provider_error` contains `usage limit` | classify on the last ERROR instead of the last assistant message (`($e\|last)` in `quota`) → rc 19 |
| D | error-401 | clean | rc 10; `provider_errors`==1; `provider_error` starts `401:` | widen the regex to any status code (e.g. `^\s*4[0-9][0-9]\b\|…`) or drop the `test` → rc 19 |
| E | error-423-rig-lease | clean | rc 10 (rig lease is not provider quota); `provider_errors`==1 | same as D → rc 19 |
| F | success-probe | clean | rc 10; `has("provider_errors")`, `provider_errors`==0, `provider_error`=="" | emit the two fields only when >0 (or drop them from the heredoc) → `has` fails |
| F2 | (none) missing `--directive` | n/a | rc 14 `launch_failed` preflight JSON also has `provider_errors`==0 and `provider_error`=="" | revert Task 1b → red |
| G | 401 + ticket text (derived) | clean | rc 19; `provider_error` contains `weekly usage limit` | read the wrong field (`.error` instead of `.errorMessage`) → red (A and B also go red) |
| H | quota-429, last line truncated (derived) | clean | rc 19; `provider_errors`==3 | replace the per-line `jq -cRn … inputs \| fromjson?` with a slurp (`jq -s` over the raw lines) → parse aborts, rc 10 |
| I | 401, error line truncated (derived) | clean | rc 10; `provider_errors`==1; `provider_error`==`(unparsed provider error)`; stderr has `WARNING` | delete the anti-vacuity block → `provider_errors`==0 |

Mutation protocol: apply each mutation to a SCRATCH COPY of the runner, never with `git checkout`/`git stash`.
The copy must be a SIBLING in `scripts/` (`cp scripts/mission_pi_run.sh scripts/mission_pi_run.mut.sh`),
because the runner resolves `RUNNER_ROOT` as `$(dirname "$0")/..` and needs `tools/pi-extensions/sandbox`
under it. A copy in `$TMP` or in `scripts/.mut/` preflight-fails with rc 15 for every arm, which is a
vacuous red. Run the suite with `MISSION_PI_TEST_SUT` pointed at the copy: support
`SUT="${MISSION_PI_TEST_SUT:-$ROOT/scripts/mission_pi_run.sh}"` as `test_mission_pi_run_sandbox.sh` does.
A mutant only counts as red if the arm fails on the asserted verdict or field, NOT on rc 15. Delete the
copy afterwards (`git status --short scripts/` must show only the real runner). Record each mutation, its
red arm(s) and the green-after-revert result in the executor report.

Acceptance: `/bin/bash -n` on the new file is rc 0. Running
`/bin/bash tools/launchd/lib/suite-env.sh tools/launchd/test_mission_pi_run_provider_quota.sh` unpiped
gives rc 0 with all arms PASS in under 60 s. Every mutation in the table turns its arm red.

### Task 3: `make/test.mk`

Add `@$(LAUNCHD_SUITE) tools/launchd/test_mission_pi_run_provider_quota.sh` directly after the
`test_mission_pi_run_sandbox.sh` line (line 65). Acceptance: `make -n test-launchd-drivers | grep provider_quota` has 1 hit.

### Task 4: changelog `changelogs/unreleased/2026-10-01-pi-runner-provider-quota.md`

Must open with `### Fixed — mission_pi_run.sh read a provider quota refusal as empty_worktree or ok (2026-10-01)`.
Then 3-4 bullets: the two measured runs (World iter 196 `empty_worktree`, iter 187 `ok` with 10 files);
the new verdict `provider_quota` rc 19 and its trigger (last assistant message is a capacity error);
the new `provider_errors` / `provider_error` fields on every verdict; and the fact that real-pi fixtures
back the test. Acceptance: `make check-changelog` rc 0.

### Task 5: the skill text (`.claude/skills/mission-control/resources/gate-3-route.md`)

a. In the rc comment block (lines 381-383), add `· 19=provider_quota` after `18=tool_hang` and keep
   "Anything non-zero except 18 is a LANE FAILURE". Re-wrap so the comment stays at three lines.
b. Directly after the `tool_hang (rc 18)` paragraph (ends "…(row 121)." near line 450), add ONE paragraph
   with no war story:
   > **`provider_quota` (rc 19) is a CAPACITY refusal, not a model failure.** The provider refused the
   > run (HTTP 429/402, usage limit, quota, credits); `provider_error` quotes it and `provider_errors`
   > counts it (pi retries some of these itself before giving up). Treat it as a lane failure: record it with
   > `mission-lane-dead.sh <role> <lane> "<verdict path> rc 19 <provider_error>"` and take the next chain
   > link. Never retry on the same lane this fire. If no link remains, the park is `PARKED-ON-LANE`
   > (SKILL.md standing rule 8) and never `needs-human-review`: name the lane, `rc 19`, and the reset hint
   > quoted from `provider_error`. A non-zero `worktree_changed_files` on rc 19 is truncated work, so measure
   > the partial as for `wall_timeout` rule (b) before handing the milestone to the next link.
c. Run `/bin/bash tools/launchd/sync-agents-skills.sh`. Then `/bin/bash tools/launchd/sync-agents-skills.sh --check` must be rc 0.
   Never hand-edit the `.agents` copy.

Do not change the fallback ORDER or any other verdict's semantics.

## Done gate (charter). The executor runs these; the controller re-runs every one OUTSIDE the sandbox

```bash
/bin/bash -n scripts/mission_pi_run.sh
/bin/bash -n tools/launchd/test_mission_pi_run_provider_quota.sh
/bin/bash tools/launchd/lib/suite-env.sh tools/launchd/test_mission_pi_run_provider_quota.sh   # unpiped, rc 0
/bin/bash tools/launchd/lib/suite-env.sh tools/launchd/test_mission_pi_run_commits.sh          # unpiped, rc 0
/bin/bash tools/launchd/lib/suite-env.sh tools/launchd/test_mission_pi_run_sandbox.sh          # unpiped, rc 0
/bin/bash scripts/test_mission_pi_run.sh            # unpiped, rc 0 (not wired into make, but still a regression check, ~1 min)
env -i HOME=$HOME PATH=$PATH make test-launchd-drivers   # rc 0, unpiped (a piped make reports the pipe's rc)
make check-changelog                                     # rc 0
```

The executor runs in a sandboxed claude session, where git/network/`make` behaviour can differ.
A sandbox-only red in a suite the plan did not touch is reported, not "fixed".

## Landing (controller)

The fleet scope guard refuses any fleet push containing `.agents/skills/…`, but Task 5c must write that
file or `test_agents_skills_sync.sh` reddens the done gate. This is the open D-FLEET-8 shape
(both skill copies vs the fleet allowlist). Commit in TWO units:

- **Commit A, landable unattended:** Tasks 0-4 (runner, test, fixtures, `make/test.mk`, changelog). On its
  own this already routes correctly: rc 19 is non-zero and not 18, so the existing skill text falls
  back. Both skill copies stay untouched, so the sync test stays green.
- **Commit B:** Task 5 (both `gate-3-route.md` copies). If the push is refused under `MISSION_NAME=fleet`,
  do not work around the guard. Keep B on the branch and raise it with the D-FLEET-8 decision (or an
  attended push). It adds only the PARKED-ON-LANE classification text.

## Estimate

About 40 LOC in the runner, about 170 in the test, 1 make line, about 8 changelog lines, about 10 skill
lines, and 5 data fixtures (69 KB). That is ~230 LOC with a 10-mutation drill, so pass
`--max-seconds 3600` if this runs on a pi lane (gate-3 wall-clock rule (a)). It stays ONE milestone:
under the ~250 LOC split line, and the arms share one harness.
