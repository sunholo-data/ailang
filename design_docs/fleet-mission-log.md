# Fleet Mission Log

Append-only, one entry per iteration, newest at the bottom. Charter: [fleet-mission.md](fleet-mission.md).

> **Older entries are ARCHIVED.** This file holds the newest 20. The full record of every
> iteration is in `fleet-mission-log-archive.md`, and a one-line index of ALL of them —
> the thing to grep before picking work, so the loop never repeats itself — is in
> `fleet-mission-index.md`.

> **Older entries are ARCHIVED.** This file holds the newest 20. The full record of every
> iteration is in `fleet-mission-log-archive.md`, and a one-line index of ALL of them —
> the thing to grep before picking work, so the loop never repeats itself — is in
> `fleet-mission-index.md`.

> **Older entries are ARCHIVED.** This file holds the newest 20. The full record of every
> iteration is in `fleet-mission-log-archive.md`, and a one-line index of ALL of them —
> the thing to grep before picking work, so the loop never repeats itself — is in
> `fleet-mission-index.md`.

> **Older entries are ARCHIVED.** This file holds the newest 20. The full record of every
> iteration is in `fleet-mission-log-archive.md`, and a one-line index of ALL of them —
> the thing to grep before picking work, so the loop never repeats itself — is in
> `fleet-mission-index.md`.

> **Older entries are ARCHIVED.** This file holds the newest 20. The full record of every
> iteration is in `fleet-mission-log-archive.md`, and a one-line index of ALL of them —
> the thing to grep before picking work, so the loop never repeats itself — is in
> `fleet-mission-index.md`.

> **Older entries are ARCHIVED.** This file holds the newest 20. The full record of every
> iteration is in `fleet-mission-log-archive.md`, and a one-line index of ALL of them —
> the thing to grep before picking work, so the loop never repeats itself — is in
> `fleet-mission-index.md`.

> **Older entries are ARCHIVED.** This file holds the newest 20. The full record of every
> iteration is in `fleet-mission-log-archive.md`, and a one-line index of ALL of them —
> the thing to grep before picking work, so the loop never repeats itself — is in
> `fleet-mission-index.md`.

> **Older entries are ARCHIVED.** This file holds the newest 20. The full record of every
> iteration is in `fleet-mission-log-archive.md`, and a one-line index of ALL of them —
> the thing to grep before picking work, so the loop never repeats itself — is in
> `fleet-mission-index.md`.

> **Older entries are ARCHIVED.** This file holds the newest 20. The full record of every
> iteration is in `fleet-mission-log-archive.md`, and a one-line index of ALL of them —
> the thing to grep before picking work, so the loop never repeats itself — is in
> `fleet-mission-index.md`.

> **Older entries are ARCHIVED.** This file holds the newest 20. The full record of every
> iteration is in `fleet-mission-log-archive.md`, and a one-line index of ALL of them —
> the thing to grep before picking work, so the loop never repeats itself — is in
> `fleet-mission-index.md`.

> **Older entries are ARCHIVED.** This file holds the newest 20. The full record of every
> iteration is in `fleet-mission-log-archive.md`, and a one-line index of ALL of them —
> the thing to grep before picking work, so the loop never repeats itself — is in
> `fleet-mission-index.md`.

> **Older entries are ARCHIVED.** This file holds the newest 40. The full record of every
> iteration is in `fleet-mission-log-archive.md`, and a one-line index of ALL of them —
> the thing to grep before picking work, so the loop never repeats itself — is in
> `fleet-mission-index.md`.

> **Older entries are ARCHIVED.** This file holds the newest 20. The full record of every
> iteration is in `fleet-mission-log-archive.md`, and a one-line index of ALL of them —
> the thing to grep before picking work, so the loop never repeats itself — is in
> `fleet-mission-index.md`.

## 8 — 2026-09-30 — paired rotate-log design parked after two quorum blocks; independent Sonnet review completed, compatibility decision D-FLEET-9 [HARNESS]

**Picked**: `mission:rotate-log-registry-cwd` + `rotate-log:status-flag-mutates-and-world-resolves-to-status-archive`, first READY pair in Mark's 2026-09-29 order; heartbeat remains parked under D-FLEET-8. 18 open tickets. No human directive on bookkeeping issue #1380 (8 comments inspected); no pending coordinator approval acted on. Kill switch absent, gh identity `sunholo-voight-kampff`, billing clean. Both running and pin skill surfaces matched origin file by file. Inherited heartbeat design/plan/JSON in the original fleet pin remain untouched. Ambient M-DOCS sprint is not this task.
**Reality check**: At `64e10b72c`, explicit registry A with conflicting CWD registry B loads A but rotates B's shared-repo log (synthetic 3→1 entries, A unchanged): provenance mismatch REAL. `--status` is a mutating archive selector (synthetic World status 3→1 with index/archive writes): REAL. Default World log resolves correctly to registered synthetic Workdir: alleged automatic status selection REFUTED. Foreign CWD without override fails explicitly. Fixture's initial duplicate boot offsets failed validation, then were corrected before interpreting target behavior. Banked [reproduction + quorum evidence](planned/m-mission-rotate-log-safe-evidence.json).
**Shipped**: None. [Design](planned/m-mission-rotate-log-safe.md) and [independent evaluation](planned/m-mission-rotate-log-safe-evaluation.md) are parked for D-FLEET-9. No implementation, formal sprint plan, sprint JSON, merge, or ticket resolution. Record branch `fleet/iter8-rotate-log` from origin/dev; worktree moved safely from initial /tmp path to `~/.ailang-driver-pin/fleet-iter8-rotate-log`. Quorum round 1 BLOCKED, one designer revision, round 2 BLOCKED: Gemini now requires an audit of all shared loader callers or rotate-only strictness. The Sonnet quorum seat was ABSENT (quota) in both rounds; never a quorum PASS. Independent Anthropic judge's technical checkpoint PASS is separate, overall BLOCKED for human approval; implementation score UNMEASURED. Resume requires D-FLEET-9 ruling, revised audited design and independent quorum, then approved plan and execute authorization.
**Progress**: goal unmoved; clause 1 product share unmeasured, clause 2 turnaround at risk, clauses 3–4 MET, clause 5 upheld by shipping nothing unverified; 0 tickets resolved, 18 open.
**Routing evidence**: base=64e10b72cf248371968b7224fde91522b3307285@2026-09-30T13:28:56Z. Controller `codex:gpt-6.1-sol` (tok: not reported). Designer `codex:gpt-6.1-sol` via Agent tool (tok: not reported), FLAGGED fallback: next rotation `pi:ollama/glm-5.3:cloud` rejected by Agent as Unknown model; bounded pi probe rc 1 on duplicate extension tool registrations (harness launch fault, not model-quality evidence); Ollama/OpenRouter rations blocked cloud alternatives. Driver also reported preferred Opus over ration rc75. Rotation records actual Codex designer, next GLM. Planner `codex:gpt-6.1-sol` via Agent (tok: not reported): BLOCKED receipt, no formal plan before approval. Executor `codex:gpt-6.1-sol` via Agent (tok: not reported): GATE_NOT_PASSED read-only receipt, no implementation. Evaluator attempts `sonnet` then driver-resolved `claude-sonnet-4-6` each rejected by Agent as Unknown model (available enum only gpt-6.1-sol/gpt-6-astra/gpt-6-sol/gpt-6-luna/gpt-5.6-sol). Required judge fallback ACTUALLY RAN fresh `claude-sub --model claude-sonnet-4-6`, then resumed for source/provenance corrections: 1,906,817 cumulative provider tokens including cache (25 input + 92,204 cache creation + 1,797,819 cache read + 16,769 output); list-price estimate $1.3441797 is informational, subscription bucket is not metered. Driver route: evaluator primary Anthropic probe over ration → pi OpenRouter minimax rc75 → declared Claude Sonnet 4.6 fallback. Preliminary alias-sonnet probe was not the judge role. Generator OpenAI ≠ judge Anthropic. task-class=design/park rounds=2 designer-corrections=1 evaluator-rounds=2 implementation-score=UNMEASURED. External quorum Gemini: r1 3,326 in/264 out, $0.00982; r2 4,188 in/213 out, $0.010932; total $0.020752 metered. Anthropic quorum seat absent with HTTP429 usage probe/no Current-session fallback; working Sonnet CLI does not erase its absent quorum seat. No policy/ration changes.
**Verification**: Version-stamped scratch CLI `/tmp/fleet-iter8-bin/ailang` v0.49.0-2-g64e10b72c, commit full base SHA. Narrow unchanged baseline `go test ./cmd/ailang ./internal/mission/...` rc0 (7 packages); not an implementation done-gate. Synthetic fixtures only; no production log rotated in reproduction. Independent judge directly checked registry/CLI, world TOML, rotate implementation/tests, and original pin heartbeat artifacts; corrected its initial controller-provider attribution and unverified V4/V5 claims. Gate-1 CI complete SHA check set: required test RED, job 109871613282 in run 36710729375 fails Check changelog index hygiene (direct MCP header-auth entry in active Unreleased, change 31be3668b; dated fragment required); subsequent skipped tests are not passes. Handed to mission-v1 as inbox_1790773989056_1861bd82; no out-of-scope repair. Full CI log `/tmp/fleet-iter8-dev-ci.log`, baseline output `/tmp/fleet-iter8-base-tests.log` remain local scratch; decision-bearing subset is tracked in evidence JSON. Healthy/degraded dry runs, launchd suite, mutation tests: NOT RUN because no implementation; no mid-run reload. STATUS iteration4 full 17-line body moved losslessly into Fleet archive, SHA256 65fa4a381a7c0381452f1eccaaa02c04e50cea407147e041a5eb457c8cb46580; exactly 3 live STATUS stamps, goal/authority unchanged. Artifact ignore control and staged-file assertions precede push.
**Ruled out**: (1) Default World automatic status selection, refuted by first-party synthetic control. (2) Gemini round1 undefined MC_DRIVER_ROOT premise: driver sets it at line47 and uses it at2202/2385. (3) Replacing required independent judge with controller verdict: provider fallback performed actual fresh Anthropic review. (4) Proceeding on technical judge PASS while quorum BLOCKED or without compatibility approval. (5) Treating initial invalid-registry rejection, skipped CI steps, or subscription list-price estimate as target evidence, green tests, or metered cost.
**Retro lane**: Backlog/park: D-FLEET-9 carries immediate rejection versus guarded deprecation and next shared-loader audit scope. Existing D-FLEET-8 mirror decision unchanged. Agent model enumeration incompatibility recurs from iteration7; record exact role errors and provider fallback, do not self-change routing policy. Pi probe duplicate extension registrations are one measured launch friction, not model death. Existing decision-ledger markers absent, helper check fails inherited start=0/end=0; existing table retained rather than self-sourced process repair. No skill edit. Last 3 actual landings: iteration7 stall arm moves clause2; iteration6 kicker/sandbox move clause2; iteration3 env-scrub moves clause5. Not an all-none drift. Fleet last20 (only9 rows): 8/9 HARNESS, expected by fleet's inverted admissibility; product-loop share remains separate (V1 8/20, Motoko10/20, Docs0/12 at observation), not a 14-day clause1 measurement.
**Next**: Top READY is `mission-base:hardcoded-origin-dev` (3 slots lost, hardcoded REF confirmed at line10 and Stapledon external repo at missions/stapledon.toml), then quota429 verdict, then pre-dirty verdict. Paired rotate-log and heartbeat stay PARKED. This iteration's design pipeline ended at the needs-human-review gate; next fire takes the banked READY item. Record-only PR must meet Gate3b; no merge over inherited required red.

**Record landing update (same iteration)**: [PR #1416](https://github.com/sunholo-data/ailang/pull/1416), record head `749cf32cdc7b2d10cde35a92c16809e6b35dcb93`, MERGEABLE/BLOCKED; required docs-gate PASS, lint/test pending at observation, full check set18 rows. Base required test is RED on the verified changelog-hygiene step. Record itself is parked needs-human-review, not LANDED; no auto-merge armed. Resume after owning V1 changelog repair: refresh record head safely, measure its complete CI check set, then merge only green and assert merge-commit CI. The design remains gated separately by D-FLEET-9 and quorum. Next fire should reconcile this open record before appending another iteration.

## 9 — 2026-09-30 — mission-base derives its base ref from origin/HEAD (stapledon's `main`); ticket resolved; iteration 8's record landed [HARNESS]

**Picked**: P0 #3 `mission-base:hardcoded-origin-dev`, the top READY row in Mark's 2026-09-29 rerank (P0 #1 PARKED D-FLEET-8, P0 #2 PARKED D-FLEET-9 by iteration 8's record #1416, which was open at pick). It was also the top ticket by slots lost (5, stapledon, recurred 2026-09-30 13:43Z). No directive on #1380 since the 13:06:28Z watermark (0 of 11 comments). Mechanical class, not policy (HD-2a).
**Reality check**: At base `440c7f01d`, `tools/launchd/mission-base.sh:10` still read `REF="${MISSION_BASE_REF:-origin/dev}"`. The old script run in the stapledon clone gave rc 1, `mission-base: cannot resolve origin/dev`. `git symbolic-ref --short refs/remotes/origin/HEAD` in all 6 mission clones gave 5 → `origin/dev` (ailang, -fleet, -world, -motoko, -docs) and stapledons-godot → `origin/main`, all rc 0. So derivation is correct for every current mission. The driver does not call the script; only the gate resources do.
**Shipped**: [#1418](https://github.com/sunholo-data/ailang/pull/1418), squash `cb7c51c8e`. `resolve_ref` order: `MISSION_BASE_REF` → `origin/HEAD` target → rc 1 naming both remedies, with no row written. It is resolved once per `snap`/`record`/`drift`; `last` needs no ref. `test_mission_base.sh` fixtures now set `origin/HEAD`, plus T1–T5. Changelog fragment `2026-09-30-mission-base-default-branch.md`. Ticket resolved with the merge SHA (5 occurrences; replied to stapledon). Also landed iteration 8's record [#1416](https://github.com/sunholo-data/ailang/pull/1416) (`60af11ee0`). Its only red was dev's changelog-hygiene failure, since fixed on dev (control: dev `440c7f01d` `test` success). `gh pr update-branch` re-ran CI; CLEAN; closing-keyword scan clean (matcher control fired); merged.
**Progress**: Clause 2: one 5-occurrence ticket resolved at ~72h from first filing (over the 48h target; it waited behind two parked designs). Clauses 3–5 met. Clause 1 not remeasured. Open tickets 16 → 15.
**Routing evidence**: base=60af11ee0875f6228fae9b7611532b64c0cf0577@2026-09-30T21:21:58Z (Gate 4); Gate-1 base `440c7f01dd178a1541f3318ab9307a859f9537b3@2026-09-30T19:46:21Z`; Gate-3b `cb7c51c8e8400435d866e889116b9a79e2d6e086@2026-09-30T20:48:56Z` (drift = our own merge). Controller `claude:claude-opus-5-5` (tok: not reported). Designer: not spawned (resolver `recipe claude:claude-opus-5-5 declared:provider-pin`). The charter says to design only when warranted, and this was a one-default fix. Planner `opus` via Agent tool (`opus fail-closed:env-pin`, derive-planner-lane verbatim; 52,408 tok). Executor `claude:claude-sonnet-5-5` via `claude-sub` recipe (`declared:provider-pin`; probe rc 0; tok: not reported). Evaluator: resolver `reroute pi:openrouter/minimax/minimax-m3 generator-equals-judge`. Its probe from the worktree failed, and from /tmp returned rc 0. The real run via `mission_pi_run.sh` returned `sandbox_not_ready` rc 17 (pi_rc 1, 10 s, 0 tool executions), recorded with `mission-lane-dead.sh`. FLAGGED fallback to the next declared lane `claude:claude-sonnet-4-6` via `claude-sub` (tok: not reported). Generator ≠ judge: Sonnet 5.5 generated and Sonnet 4.6 judged; different models, same vendor. Evaluator rounds 1, PASS 100.
**Verification**: Executor gates rc 0 (13 arms; launchd suite PASS=25; check-changelog). The controller re-ran `/bin/bash -n`, `test_mission_base.sh` unpiped rc 0, and `env -i HOME=$HOME PATH=$PATH make test-launchd-drivers` rc 0 on `aa78fff1e`. Evaluator mutation drills on the real script (cp backup + cmp restore): always-origin/dev → T1–T4 red; silent fallback → T3; ignore override → T2; drop drift resolve → T4 plus 2 existing drift arms. No vacuous arm. PR checks CLEAN, including `launchd drivers (bash 3.2)`, `test`, `lint` and `build`. Merge-commit `cb7c51c8e`: 15/16 green; SonarCloud failure = new-code coverage 74.5% < 80, also failing on `440c7f01d`, `834b6e1db`, `a60a72d17` and `b9cd1bd20` (inherited; bash change). Done-gate note: no driver dry run was performed. The driver never calls `mission-base.sh`, so a dry run cannot exercise it. The healthy/degraded pair is the live A/B (ailang pin records `origin/dev` = 440c7f01d; stapledon records `origin/main`) plus T3 (no `origin/HEAD` → loud rc 1). Nothing was reloaded.
**Ruled out**: A silent `origin/dev`/`main`/`master` fallback when `origin/HEAD` is unset. It would re-create this bug for any repo whose default is neither, and the repo forbids silent fallbacks. Network derivation (`git ls-remote --symref`): the helper deliberately reads the shared .git without fetching.
**Retro lane**: (1) New harness defect, measured: every pi role launched inside a `sunholo-data/ailang` worktree fails to start. The 8 ailang-managed tools in `~/.pi/agent/extensions` (installed 2026-09-29 09:26) collide with the same tools in the repo's project-local `.pi/extensions`, and pi 0.85.1 refuses duplicates. The runner then names it `sandbox_not_ready`, which misattributes the cause. Control: the same probe from /tmp returned rc 0. The fleet cannot file it (`ailang mission ticket file` → "the fleet mission works tickets; it does not file them"), so it is raised to Mark in the report rather than self-sourced into the queue. (2) The resolved ticket's evidence notes that the gate resources still call `bash tools/launchd/mission-base.sh` by relative path; from a non-ailang CWD use `$AILANG_DRIVER_SRC/...`. That is the parked D-FLEET-8 class, not reopened here. No skill edit.
**Next**: P1 #4 `pi-runner:quota-429-reported-as-empty-worktree` (then #5 pre-dirty). D-FLEET-8 and D-FLEET-9 stay parked for Mark.

## 10 — 2026-10-01 — pi runner types a provider quota refusal as `provider_quota` rc 19; stale-pin re-file of mission-base resolved [HARNESS]

**Picked**: P1 #4 `pi-runner:quota-429-reported-as-empty-worktree`, the top READY row in Mark's 2026-09-29 rerank (P0 #1 PARKED D-FLEET-8, P0 #2 PARKED D-FLEET-9, P0 #3 LANDED). Mechanical class: no driver code routes on the runner's rc (`grep mission_pi_run tools/launchd/mission-control.sh` → comments only), so a new verdict changes no lane order. No directive on #1380 since the 2026-09-30T13:06:28Z watermark (0 of 12 comments). Bookkeeping second pick: `mission-base:hardcoded-origin-dev` had re-opened (stapledon iter 5, filed 21:43Z).
**Reality check**: At base `a03ec7013`, `grep -n 'stopReason\|errorMessage' scripts/mission_pi_run.sh` → only header comments; the `finished` verdict reads only fence, pi_rc, porcelain and commits. Re-file: stapledon's driver banner for that slot reads `driver pin: running committed origin/dev @ 440c7f01d` (pre-fix; `cb7c51c8e` merged 20:48Z). At HEAD, `mission-base.sh snap` run in `stapledons-godot` → rc 0, `origin/main` `dd760b292`. So it was a stale pin, and it was resolved against `cb7c51c8e` with that evidence (stapledon notified).
**Shipped**: [#1424](https://github.com/sunholo-data/ailang/pull/1424), squash `94524a6fc`. The new verdict is `provider_quota` rc 19. It fires when the LAST assistant `message_end` is `stopReason:"error"` and its `errorMessage` matches a capacity pattern (429/402, usage limit, quota, rate limit, credits, too many requests). Precedence: after `sandbox_not_ready`, before `launch_failed`/`ok`/`empty_worktree`. A quota error followed by a later successful assistant turn does not fire it; it is counted instead. `provider_errors` and `provider_error` are on every verdict, including preflight. A `stopReason` error line that cannot be parsed counts as `(unparsed provider error)` with a stderr warning. New `tools/launchd/test_mission_pi_run_provider_quota.sh` (29 checks), wired into `make test-launchd-drivers`; 5 fixtures in `tools/launchd/testdata/pi-ndjson/`, cut from or captured with real pi 0.85.1 output (a real quota run, a bogus-key 401, the 423 rig_lease refusal, a success probe, and a splice of real lines), sha256 in the test header. Changelog fragment `2026-10-01-pi-runner-provider-quota.md`. Ticket resolved with the merge SHA (world notified).
**Progress**: Clause 2: one ticket resolved ~4.5 days after filing (09-26 19:10Z; it sat behind P0s). Clauses 3–5 met. Clause 1 not remeasured. Open tickets 16 → 14 (this one, plus the stale re-file).
**Routing evidence**: Gate-1 base `a03ec7013e0ddf7557aeffe8ea55ebeb7c37f1f2@2026-10-01T04:20:15Z`; Gate-3b `94524a6fc3d63622f93b0c10aba001095255efb5@2026-10-01T05:47:50Z` (drift = our own merge, 1 commit). Controller `claude:claude-opus-5-5` (tok: not reported). Designer: not spawned (resolver `recipe claude:claude-opus-5-5 declared:provider-pin`); a single-verdict runner fix with the plan as spec. Planner `opus` via Agent tool (`agent-tool opus fail-closed:env-pin`, foreground; 165,809 tok). Executor `claude:claude-sonnet-5-5` via `claude-sub` recipe (`declared:provider-pin`; probe rc 0; run rc 0 in ~23 min under a 45-min cap; tok: not reported). Evaluator: resolver `reroute pi:openrouter/minimax/minimax-m3 generator-equals-judge`. The real run via `mission_pi_run.sh` returned `sandbox_not_ready` rc 17 after 11 s with 0 tools, stderr `Tool "ailang_run" conflicts with …/.pi/extensions/ailang-exec.ts`. Recorded with `mission-lane-dead.sh`. FLAGGED fallback to the next declared lane, `claude:claude-sonnet-4-6` via `claude-sub` (probe ok; tok: not reported). Generator ≠ judge: Sonnet 5.5 generated and Sonnet 4.6 judged; different models, same vendor. Evaluator rounds 1, PASS 97, 0 blocking. Metered spend: one 11-s pi launch with 0 model calls ($0).
**Verification**: Executor: all gates rc 0; its first `make test-launchd-drivers` run was rc 2 on `test_mission_kill_tree.sh` (2/8, late-descendant timing, a file outside the diff), and the re-run was rc 0. Mutation table: 10 mutants, each red on its arm. Controller, outside any sandbox: `/bin/bash -n` rc 0, the new suite unpiped rc 0 (29/0), `make check-changelog` rc 0, `env -i HOME=$HOME PATH=$PATH make test-launchd-drivers` rc 0. Fixture scan: no emails or keys; the only personal string is the rig home path, already common in repo docs. Evaluator: 5 drills of its own, all red (delete the elif; precedence below ok; first instead of last; regex widened to any 4xx; anti-vacuity deleted). It confirmed the fixture hashes and the last-assistant rule (arm C). Non-blocking NB-1: a 500 whose body contains "too many requests" would read 19, and no test pins that boundary. PR checks CLEAN (22). Merge commit: CI success, 15/16 green; SonarCloud failure, also red on base `a03ec7013` at Gate 1 (inherited). Done-gate note: there was no driver dry run, because the driver never calls `mission_pi_run.sh`. The healthy/degraded pair is fixture-driven: quota-last → 19, recovered → normal verdict, 401/423 → not 19. Nothing was reloaded.
**Ruled out**: Classifying on ANY error in the run. pi retries a 429 three times on its own (`utils/retry.js`), and a run that recovers did real work. Treating the 423 `rig_lease` refusal as quota: it is a lease conflict, not a capacity bucket; a test pins it as not-19. If it needs a verdict, that is its own ticket. Editing `gate-3-route.md` without its `.agents` copy: `test_agents_skills_sync.sh` (in the done-gate suite) would go red, and the pre-push guard refuses `.agents/**`.
**Retro lane**: (1) The pi extension collision (iteration 9 retro item 1) recurred: the pi evaluator lane is dead on every ailang-worktree launch, for 2 fires running. It costs a FLAGGED fallback per iteration and still has no filer, because the fleet cannot file tickets. Escalated to Mark again. (2) The skill-text half of fleet fixes is now blocked twice on the `.agents` mirror scope (iteration 7's heartbeat design, this rc-list line). Both wait on D-FLEET-8. (3) The planner's false-premise list caught three charter-level facts the controller had stated wrongly: the pi install path, `scripts/test_mission_pi_run.sh` not being wired into any make target, and a real 429 run existing in `~/.ailang/state/` evidence that the `/tmp` search missed. Each was 1 instance, and the controller's own "empty search" claim was the one the rules warn about. No skill edit.
**Next**: P1 #5 `pi-runner:verdict-blind-to-commits-and-predirty` (pre-dirty half). D-FLEET-8 and D-FLEET-9 stay parked for Mark.

## 11 — 2026-10-01 — heartbeat ticket parked on unavailable independent Agent evaluator lanes [HARNESS]

**Picked**: P0 #1 `skill:heartbeat-relative-path-absent-in-world`, top attended groom and seven occurrences. Mark's D-FLEET-8 YES ruling (both mirrors and rc-19 text) acknowledged; D-FLEET-9 A also acknowledged as banked next work, not re-asked. No allowlisted issue directive since 2026-09-30T13:06:28Z (0 of 14 comments). Four coordinator approvals remain for the operator, untouched.
**Reality check**: Independent read-only executor observed nine operative relative heartbeat commands in seven `.claude` gate resources and the same nine in seven `.agents` resources. World and Stapledon README positive controls exist, relative helpers do not. Isolated old calls: rc 127, zero state files; absolute pinned-helper controls: rc 0, namespaced gate-0 rows, attempt 11. The old untracked design, plan and JSON explicitly exclude the mirror and omit rc19. Current `_scope_verdict fleet` refuses `.agents` mission files while permitting hook changes. D-FLEET-8 authorizes the intended scope, but its enforcement seam must be addressed in the next design.
**Shipped**: none. Three inherited untracked artifacts remain byte-preserved in the pin. This branch contains mission records only; not a fix or approved sprint. No work was delegated through the coordinator, no task approved, no ticket marked read or resolved.
**Progress**: goal unmoved. Clause 1 product-loop rolling share UNMEASURED (not replaced by fleet's own harness ratio); clause 2 turnaround UNMET/at risk; clauses 3 and 4 MET by existing evidence; clause 5 preserved by shipping no unjudged change. Open ticket signatures: 25 at Gate 5 (fresh JSON enumeration; Gate-1 list length was not banked). Fleet's prior ten work iterations are 10/10 HARNESS; the fleet charter intentionally inverts product-loop admissibility. Prior three landings (iterations 7, 9, 10) moved clause 2 turnaround; no drift alarm based on three nonmoving landings.
**Routing evidence**: Controller native Codex (tok: not reported). Gate-1 base `7c640e83d0f96aa35e0536e19ab4d2c885d45378@2026-10-01T18:54:39Z`; Gate-4 base `b9813dddae877e50c9d34734d112311c2d6c5086@2026-10-01T18:59:26Z`. Shared ref advanced; clean record branch rebased before writes. Authoritative user-named pin skill and resolved user symlink: SKILL plus all 12 resource files each MATCH origin at Gate 1. Native Agent transport is the operator's explicit override of provider CLI recipes; this is recorded, not silently treated as the prescribed recipe.
- Designer rotation last-used `codex:gpt-6.1-sol`: next `pi:ollama/glm-5.3:cloud` → `pi:ollama/kimi-k3:cloud` → `claude:claude-opus-5-5` each rejected by native spawn (`Unknown model`, respectively `glm-5.3:cloud`, `kimi-k3:cloud`, `claude-opus-5-5`); next native `gpt-6.1-sol` ran read-only requirements checkpoint (tok: not reported). No new design written or approved.
- Planner derived `codex:gpt-6.1-sol declared:planner-lane-default-pin`; native `gpt-6.1-sol` ran read-only plan-readiness checkpoint (tok: not reported), not a new sprint plan or approval.
- Executor default `codex:gpt-6.1-sol`; native `gpt-6.1-sol` ran read-only premise/guard checkpoint (tok: not reported), not sprint execution.
- Evaluator resolver `agent-tool sonnet declared:alias-pin`; native `sonnet` rejected Unknown model. Declared fallback order: `pi:openrouter/minimax/minimax-m3` (native `minimax-m3` Unknown model), `claude:claude-sonnet-4-6` (native `claude-sonnet-4-6` Unknown model), `opus` (Unknown model). Tool returned no numeric rc; these are spawn validation errors, not CLI quota probes. Available native list was gpt-6.1-sol, gpt-6-astra, gpt-6-sol, gpt-6-luna, gpt-5.6-sol. No undeclared judge substituted. Evaluator not spawned, score UNMEASURED, generator-not-equal-judge gate unsatisfied; nothing landed on the controller's verdict.
**Verification**: No production writes, so test/lint/build and done-gate are NOT RUN and no pass is claimed. Three role findings are independently read-only; they are not acceptance review. Billing tripwire CLEAN. Lane park resume predicate: retry `collaboration.spawn_agent` with declared evaluator model `sonnet` and walk only its declared fallback order; proceed only when accepted and a real independent verdict can be produced. No reset time exists: condition is Agent model support or an explicit operator routing/transport ruling, not waiting a quota window. Gate 1 dev was not green: Windows TestMaterializeRunPolicy failed in internal/executor, macOS matrix cancelled, main test pending at observed SHA. Current advance b9813dd includes a policy-workspace Windows fix, so original red is already being handled outside fleet; no competing fix. Latest named CI run was older cb6fc0f, not evidence for HEAD. CLI on PATH reports v0.50.0-6-g021c46907-dirty; no compiler semantics verdict quoted. Decision-ledger scripts fail on this charter's historical unmarked RULED table (no block); no row was inferred OPEN or reopened.
**Ruled out**: Controller approval in lieu of independent evaluator; substituting an undeclared available GPT model as judge; executing the seven-file inherited plan; bypassing pre-push scope; treating unknown-model as model-quality or quota failure; treating role checkpoints as completed design/planning/execution; resolving tickets without origin/dev landing. No second ticket selected because evaluator unavailability blocks the same acceptance gate for every landing.
**Retro lane**: backlog/routing evidence only. World already filed `agent-tool:sonnet-unavailable` three times; this is another capability reading, not authority for the fleet to copy World's D-WORLD-48 routing ruling. No policy or shared skill edit. Nothing reloaded mid-iteration. Current park is capacity (`PARKED-ON-LANE`), not a fabricated human decision.
**Next**: Re-probe evaluator support; on success revise P0 heartbeat design for both mirrors, rc19 and guard enforcement, fresh quorum, approved plan and execution. Banked P0 rotate-log pair D-FLEET-9 A follows; pre-dirty runner P1 follows that. DECISIONS FOR MARK: none newly filed; D-FLEET-8/9 already ruled.

## 12 — 2026-10-02 — heartbeat design Revision 4 after three quorum rounds; PARKED needs-human-review on reviewer-vs-measurement deadlock (D-FLEET-10) [HARNESS]

**Picked**: P0 #1 `skill:heartbeat-relative-path-absent-in-world` (top attended groom, 7 slots lost, UNPARKED by D-FLEET-8). Iteration 11's PARKED-ON-LANE resume predicate held: this fire's driver resolved every role lane, so the normal gates ran. No allowlisted directive since 2026-09-30T13:06:28Z (0 of 15 comments on #1380); watermark advanced to 2026-10-01T19:02:37Z. Iteration 11's orphaned record PR #1459 (still OPEN at Gate 2) was verified and merged first (squash `0bbdfa108`), banking its log entry, STATUS stamp and index row — the died-mid-flight deliverable was VERIFY-AND-LAND, and its branch held up against current origin/dev with every PR check green.
**Reality check**: defect re-verified at HEAD `c77fae8da`: 9 + 9 relative `bash tools/launchd/mission-heartbeat.sh stamp` calls across 14 gate files (`.claude` and byte-identical `.agents` mirrors), 0 uses of `$AILANG_DRIVER_SRC` in resources, relative call from a foreign CWD rc=127; gate-3-route.md:386 rc list lacks `19=provider_quota` (landed in runner at `94524a6fc`); pre-push guard `_scope_is_harness` line 17 allows `.claude/skills/mission-*` but no `.agents` paths; `test_agents_skills_sync.sh` green at base.
**Shipped**: no implementation (quorum never passed). Revision 4 of `planned/m-mission-heartbeat-driver-root.md` banked as the record (design-doc-creator role, 3 bounded runs). Revision 2 = D-FLEET-8 scope (both mirrors 18 sites, rc-19 both copies, guard allowlist + fleet-mission Authority in step, spaces-root drill measured, attended-setup sentences). Revision 3 = round-3 objection closed (catch-all sentence verified: both copies :387-388, grep rc=0). Revision 4 = round-4 objection closed honestly: `${...:?}` failure exit status is invocation-shape dependent on the same `/bin/bash` 3.2.57 — `bash -c` command strings exit **127**, script files exit **1** (controller reproduced in six shapes; designer re-ran both); the round-2-era "not reproducible here" sentence was an over-generalization and is retired; the operative contract stays "nonzero, no pinned code".
**Progress**: goal unmoved by this park (no landing). Clause map: 1 product share UNMEASURED; 2 turnaround UNMET/at risk (ticket 7 slots lost, first filed 09-26); 3 one queue MET (25 open signatures at Gate 0); 4 idle is free MET (probe-only exit when 0 open — 25 open here); 5 no regressions preserved by shipping nothing unjudged.
**Routing evidence**: Controller `pi:openrouter/z-ai/glm-5.3` (pi fallback rung; anthropic+codex buckets over ration; tok: not reported). Gate-1 base `c77fae8da@2026-10-02T01:22:51Z`; Gate-4 base recorded after the #1459 merge (`0bbdfa108`). All four roles provider-pinned `recipe` lanes this fire — the Agent tool is not valid for any of them per gate-3-route's spawn table (the prior controller substituted CLI recipes for the operator's requested Agent transport; that substitution did not satisfy the request, and planner/executor/evaluator were not spawned). Designer `pi:openrouter/z-ai/glm-5.3` via mission_pi_run.sh: 3 runs (authoring r2 in=24678 out=15765 cacheRead=468864; revision r3 in=22133 out=3589; revision r5 in=74951 out=4964) — one revision past the one-doc diet ceiling, FLAGGED. Planner/executor not reached (no quorum PASS). Evaluator was NOT spawned in this interrupted run. This violated the operator's requested role transport and required-judge instruction; iteration 13 supplies independent review of the recovered record, not implementation acceptance. Quorum: rounds 3 ($0.1326), 4 ($0.0174), 5 ($0.1834), metered total $0.3334; panels seated claude-sonnet-5@claude-p (absent quota ×3), gpt6-1-sol (absent auth ×3), gemini-3-1-pro (present, reject ×3), oc-kimi-k3 (round 5: present, reject); author benched oc-glm-5-3 correctly.
**Verification**: No production writes, so test/lint/build and the mission-loop-change done-gate are NOT RUN and no pass is claimed. Billing tripwire CLEAN at Gate 0. `ailang` CLI v0.50.x warned "binary may be stale" on every invocation this fire — a known fleet-hygiene signal (rebuild via `make quick-install` is V1's scope), recorded, not worked around. Decision-ledger provenance scripts not re-run (D-FLEET-10 filed OPEN, none resolved — the unattended loop never resolves rows).
**Ruled out**: Proceeding to planning on a BLOCKED quorum (Standing rule 2); a third revision round against gemini-3-1-pro's repeat objection — unfalsifiable from the reviewer's seat and empirically false for the `-c` shape on this rig (six-shape measurement in the doc); applying the narrow-refinement carve-out (oc-kimi-k3's absoluteness fix is concrete and non-directional, but Gemini did propose changing the exit-status assertion; that proposed fix conflicts with the rig measurements. This is a judgment dispute, so the carve-out cannot license routing to sprint-planner); pinning rc=127 in acceptance tests (shape-dependent: 1 vs 127); silently substituting an undeclared judge; resolving the open ticket (no fix on origin/dev); a second queue item this fire (standing rule 1 — the pick was a full work item, not bookkeeping-only).
**Retro lane**: (1) Quorum seat availability is the real bottleneck: 2 of 3 seats absent on every round this fire (claude-sonnet-5 quota, gpt6-1-sol auth) — one reviewer's false factual claim was unchallengeable from inside the protocol; D-FLEET-10 carries the adjudication. This is the second consecutive ticket whose quorum burned ≥2 rounds with blocked seats (iteration 8 rotate-log, this) — routing-policy signal for Mark. (2) The text reviewer cannot reproduce rig measurements by construction; round 5's gemini objection ("fabricated") is what that limitation looks like from the outside — a quorum artifact schema field for "controller-reproduced command + output" already exists (the doc's Verification Log); the missing piece is the reviewer's ability to run it. Candidate backlog row, not a skill edit (one edit per iteration, and the two-friction bar is not met on a single instance). (3) Fable-diet overspend FLAGGED above; the cause is protocol-mandated revision rounds blocking consecutively — same root as the diet note's "rate of blocked-round-1 docs" clause. No skill edit this iteration.
**Next**: D-FLEET-10 to Mark (adjudicate the rc=127/rc=1 two-shape fact + the absoluteness guard). On ruling: Revision 5 + fresh quorum next fire; D-FLEET-9 A rotate-log pair follows; pre-dirty pi-runner P1 after that. Ticket `skill:heartbeat-relative-path-absent-in-world` stays OPEN (unread = open).

## 13 — 2026-10-02 — interrupted iteration 12 record recovered; heartbeat remains parked for D-FLEET-10 [ADMIN]

**Picked**: VERIFY-AND-LAND of iteration 12's stranded record, per died-mid-flight gate. Found staged record/design files in sibling worktree `fleet-iter12-heartbeat` and no open fleet PR. No second implementation item selected. Three untracked artifacts in the pin and two stale plan/JSON artifacts in the old worktree are byte-preserved and excluded.
**Reality check**: Three source quorum JSON artifacts all synthesize BLOCKED. Reproduced on /bin/bash 3.2.57: unset-root expansion fails rc127 with a command string, rc1 with a script file; contract should assert nonzero. The absoluteness gap is real. Source log incorrectly said Gemini proposed no fix and that CLI recipes satisfied the operator's Agent request; those claims are corrected in the recovered record. Source quorum JSON is archived under `fleet-mission-evidence/iteration12/`.
**Shipped**: record only; rejected Revision 4 is retained as a historical proposal, never approved. No heartbeat code changed, no ticket resolved. D-FLEET-10 remains OPEN; D-FLEET-8/9 remain RESOLVED. Existing decision rows normalized into the standard four-column marked ledger without changing answers or attended evidence; scripts validate ten rows and generate only D-FLEET-10 as OPEN.
**Progress**: goal unmoved (0 tickets resolved). Clause 1 product share UNMEASURED; clause 2 turnaround UNMET/at risk; clause 3 one queue MET by ticket store; clause 4 idle-is-free prior evidence only; clause 5 preserved by shipping no fix. Fleet's previous work iterations are HARNESS by its charter's inverted admissibility; this record is ADMIN. Prior three landings (7,9,10) moved turnaround; no three-landing drift alarm.
**Routing evidence**: base=e513faa487a76d8fb2dfb41af530b88f17fa8b94@2026-10-02T08:34:14Z (Gate4). Controller native codex:gpt-6.1-sol (tok: not reported). User expressly requests Agent transport, overriding CLI recipes. Designer: rotation after Sol, native glm-5.3:cloud then kimi-k3:cloud then claude-opus-5-5 each rejected Unknown model; native gpt-6.1-sol fallback ran read-only design review (tok: not reported). Planner derive: codex:gpt-6.1-sol declared:planner-lane-default-pin; native gpt-6.1-sol read-only readiness review (tok: not reported). Executor default codex:gpt-6.1-sol; native gpt-6.1-sol read-only premises/inventory review (tok: not reported). Evaluator sonnet rejected Unknown model; native separate fresh-context gpt-6.1-sol fallback per current gate3 lines242-252, FLAG judge-independence:same-model-fresh-context (tok: not reported). This is four role checkpoints on a record recovery, not a completed design/plan/execute sprint. Evaluator record PASS 96/100, round1, no record blockers; implementation score UNMEASURED. Native validation errors have no numeric rc. Quota probe rc0: ollama and openrouter over ration, Anthropic unknown/blocked; no API spend incurred this run, native subscription quota consumption not reported.
**Verification**: original record source hashes preserved; all source plan/JSON files excluded. Ledger valid, design byte-identical to source, all decisions/queue/bar/Authority sections retained. Latest source CLI reports v0.50.0-6-g021c46907-dirty, so no compiler or production gate verdict claimed. Dev check set at e513faa had launchd and macOS build failures with other checks unfinished; named CI lookup returned older commits and is not HEAD evidence. Full product tests/done-gate not run for record-only changes. Remote record PR/merge CI will be tracked by exact SHA; record not called LANDED before confirmation. Base failures measured: launchd job110762944401 fails mirror sync for sprint-evaluator; macOS build job110762859936 fails TestMemWatchdogKillsAllocator after60s. These are base failures, not caused by this docs-only diff.
**Ruled out**: continuing stale plan, self-approving blocked quorum, resolving D10 unattended, controller-as-judge, using retired GPT models, reading/acking mission-fleet tickets, touching the sixteen pending coordinator approvals. New blocking-all controller ticket remains open: existing fallback now checks every rung's ration, but the 402 slot-classification half remains unverified; no full resolution claimed.
**Retro lane**: record corrections only, no shared skill or routing-policy edit. Independent roles caught stale plan scope, draft dashboard's unverified CI claim, and incomplete reviewer characterization. Same-model fresh-context judge is FLAGGED under the current skill rather than silently inheriting the controller verdict. Nothing reloaded.
**Next**: D-FLEET-10 human ruling with Revision5 and fresh quorum as recommended. Banked D-FLEET-9 rotate-log pair follows; pre-dirty runner next. Blocking-all controller-capacity ticket needs its two halves checked before policy routing or resolution.

## 14 — 2026-10-02 — controller HTTP 402 proposal parked for D-FLEET-11; admission-bypass premise refuted [HARNESS]

**Picked**: `driver:controller-fallback-skips-openrouter-ration-and-402-reads-as-crash`, new blocking=all ticket (one occurrence, filed by attended session); charter expressly puts blocking-all above the attended groom. Heartbeat remains parked on OPEN D-FLEET-10; D-FLEET-8/9 stay RESOLVED. No new allowlisted directive on #1380 since 2026-10-01T19:02:37Z (0 of 17 comments). Sixteen coordinator approvals are operator-owned and untouched. Inherited heartbeat sprint JSON and plan in pin, old worktrees, and active docs sprint are untouched.
**Reality check**: Current `select_model` checks admission for every fallback and `_mc_rung_bucket` maps OpenRouter correctly. Controller re-ran extracted source with stubs: blocked OpenRouter skips its probe and selects admitted local lane; healthy selects OpenRouter. All-blocked designer drill selects none. Historical Fleet 03:14:57 and Stapledon 02:16:39 blocked lists exclude OpenRouter; later snapshots were conflated in the ticket. Actual 402 emissions and CRASHED exits are confirmed at dated start/end timestamps. Current runtime signature misses 402, 429 positive control matches; transient Overloaded matches its own signature; healthy matches neither and completes. Original same-attempt selection/slot code, not role-runner rc19, is the instrument.
**Shipped**: no runtime fix, approved plan, sprint execution or ticket resolution. Designer wrote parked `planned/m-controller-capacity-admission.md` with two-hunk proposed diff (not applied), `git apply --check` verified. D-FLEET-11 asks A/B/NO under HD-2a. Planner and executor ran readiness/premise checkpoints only; those roles are not claimed as planning/execution. Independent proposal review PASS91/100, round1, no blockers to presenting the ruling; report `planned/m-controller-capacity-admission-evaluation.md`. Implementation acceptance UNMEASURED. Native judge is separate fresh-context Agent; text quorum/HD-2a/plan/execution gates remain open. Decision-bearing evidence is tracked under `fleet-mission-evidence/iteration14/`, including the complete 34-signature Gate1 snapshot. Independent record coherence review preserved the ledger/bar/Authority/queue and exact STATUS11 archive text, confirmed13 HARNESS/15 index entries; token and metered totals are controller provenance.
**Progress**: goal unmoved (0 tickets resolved). Clause1 rolling product-loop harness share UNMEASURED; clause2 turnaround UNMET/at risk; clause3 one queue MET (34 open signatures at Gate1); clause4 idle-is-free prior evidence only, not exercised with open work; clause5 preserved by shipping no unjudged fix. Previous three implementation landings (7,9,10) moved turnaround; no three-landing drift alarm. Fleet index mix after this record is13 HARNESS/15 entries, intentionally admitted by fleet's inverted scope, not the product-loop14-day KPI.
**Routing evidence**: Controller native Codex (tok: not reported); base-gate1=4460d91b91f9a4d78abadac5ae4026c823fb6a72@2026-10-02T15:18:12Z; base=4460d91b91f9a4d78abadac5ae4026c823fb6a72@2026-10-02T15:27:30Z (Gate4). Pin and resolved symlink SKILL + all12 resource files each MATCH origin; followed operator-named pin rulebook. User expressly requires native Agent transport, overriding recipe-only routes. Designer rotation last Sol → GLM (`glm-5.3:cloud`) → Kimi (`kimi-k3:cloud`) → Opus (`claude-opus-5-5`) rejected Unknown model; next `gpt-6.1-sol` authored proposal (tok: not reported), FLAGGED. Planner docless resolver `agent-tool opus fail-closed:no-doc` conflicts with declared `codex:gpt-6.1-sol`; role-spawn-routing(a) follows pin for read-only checkpoint (tok: not reported). Once doc exists derive returns `codex:gpt-6.1-sol declared:planner-lane-default-pin`, no plan created. Executor `codex:gpt-6.1-sol declared:provider-pin`, native `gpt-6.1-sol` read-only checkpoint (tok: not reported). Evaluator `agent-tool sonnet declared:alias-pin` rejected Unknown model; declared default chain `deepseek-v4-pro:0813-cloud`, `deepseek-v4-pro-0813`, tail `opus` each rejected Unknown model. Current gate3 rule3 permits fresh-context `gpt-6.1-sol`, separately spawned on isolated evaluator tree, FLAG `judge-independence:same-model-fresh-context` (tok: not reported). Validation errors supply no numeric rc; they are capability errors, not quota failures. Retired Astra/other GPT models not spawned. Metered=$0.00; subscription token counts unavailable.
**Verification**: CLI rebuilt in scratch from exact base with module-derived ldflags: v0.51.0-24-g4460d91b9/full4460d91b91f9a4d78abadac5ae4026c823fb6a72. Initial mistaken module-prefix build printed dev; corrected before relying on it. No shared binary installed or driver reloaded. Quota `--over`: ollama over11.8pp/22h6m versus10pp/day; Anthropic unknown403 and CLI fallback failed; OpenRouter over$6.57 versus$2.33/day. Author benches OpenAI quorum seat, other named cloud/subscription reviewer lanes are not admitted; text quorum NOT RUN, deferred for ruling and admitted lanes, never represented as PASS. Native independent judge reviews proposal record only. `/bin/bash -n` driver rc0; routing baseline104 passed0 failed. Existing chain suite rc0/six PASS but repeated missing helper commands, independently reproduced: green is not admission proof. Base mirror suite2 passed1 failed rc1; no full driver suite/done-gate green claimed. HEAD has20 check rows with launchd mirror failure; job110884609761 log58437B proves sprint-evaluator mirrors drift, one following syntax step skipped. V1 informed, no competing unrelated repair. Latest named CI successcb6fc0f is older than HEAD. Full product tests/lint and healthy/degraded live dry-runs not executed for parked record-only changes. Record PR exact-SHA CI disposition remains pending at commit; no auto-merge prediction is a landing.
**Ruled out**: Adding a redundant ration check; citing later blocked list for earlier admission; role-runner provider_quota certifying direct controller text path; classifying all402 prose or loose credits words; changing ration thresholds, order, reader or rc75 without ruling; treating chain suite's false green as proof; approving policy from judge score; skipping independent Agent judge; continuing heartbeat stale plan; resolving/reading/acking mission-fleet tickets; touching pending coordinator approvals.
**Retro lane**: backlog evidence only. Chain test helper omission is linked validation debt for this ticket, to cover in the approved scoped sprint, not an unsolicited audit/fix. Native lane capability fallback recurs and is recorded by actual model/errors; current skill already permits fresh Sol judge, no new routing-policy edit. No shared skill, installed config, quota ledger threshold, driver, extension or launchd reload changed.
**Next**: D-FLEET-11 A/B/NO policy ruling; regardless of pending ruling, next runnable ranked item is D-FLEET-9 rotate-log pair, then pi-runner pre-dirty. D-FLEET-10 remains OPEN; no repeat of resolved D8/9 asks. Ticket stays OPEN (unread=open); record/design are not a fix.

## 15 — 2026-10-03 — controller HTTP 402 classified as capacity; pause no longer reads as a crash — LANDED #1549 [HARNESS]

**Picked**: `driver:controller-fallback-skips-openrouter-ration-and-402-reads-as-crash` (blocking=all), unparked by D-FLEET-11 = A (Mark, attended 2026-10-02; commit 2533a43d6 at 18:11Z, outside any fleet slot window: iteration 14 ended 15:39Z, the next fire started 21:39Z). It outranks D-FLEET-10's heartbeat unpark by the charter's blocking-all rule. #1380 had 0 new allowlisted directives since 2026-10-02T08:51:41Z (19 comments). Died-mid-flight resume: attempt 1 (fire 21:39Z, opus controller) left worktree `fleet-i15` on unpushed branch `fleet/i15-controller-402` with Rev 2–3 committed. Its slot verdict was `CRASHED at=gate-3 rc=1` 4s after the planner spawn.
**Reality check**: Rev 3's deployment premise was re-measured first-party. `MISSION_RUNTIME_QUOTA_SIG` is set only at its definition (`tools/launchd/mission-control.sh:1077`): no env file, no plist, and `launchctl getenv` empty (control: 2 plists name `MISSION_NAME`). `MC_PAUSED=0` is bound at :894. Attempt 1's crash cause was read from the driver log rather than inferred. It was `You've hit your session limit · resets 12:40am`, and `RUNTIME_QUOTA_SIG` (`…|hit your usage limit|…`) does not match "hit your **session** limit". So an Anthropic session-limit stop is also recorded as CRASHED. This is a sibling emitter outside D-FLEET-11's ruled 402 scope. It was NOT fixed. `ailang mission ticket file` refuses the fleet itself ("the fleet mission works tickets; it does not file them"), so it is recorded here and raised to Mark.
**Shipped**: #1549 → `27dab5bb4`. Driver +7/−2: the anchored `^402:` emitter joins `RUNTIME_QUOTA_SIG`, and a new final branch `elif [ "${MC_PAUSED:-0}" -eq 1 ]` logs the pause and suppresses the generic crash notice. The existing pause `_mc_notify` already fired, so nothing is sent twice. The planner caught the controller's brief wrongly asking for a second notify, and the design governed. New hermetic `test_controller_capacity.sh` (14 cases), wired into `make test-launchd-drivers`. `test_controller_chain.sh` now loads the real ration and demote helpers (`command not found` 63 → 0) and adds blocked-OpenRouter, all-blocked and demoted-rung cases. Changelog fragment `2026-10-03-controller-402-capacity.md`. Ration-admission half not reproduced at HEAD, so no guard was added. Thresholds, lane order, billing reader, rc values, `lib/`, `.pi/` and `mission_pi_run.sh` are unchanged. D-FLEET-10/11/12 ledger status cells normalized from free text `RULED …` to the checker's `RESOLVED` (`mission_decisions.sh --check` had failed on all three; answers and evidence unchanged). Ticket resolved after merge.
**Progress**: goal moved: 1 ticket resolved, the first since iteration 10. Clause 1 product share UNMEASURED. Clause 2 turnaround UNMET: this ticket took ≈24h from filing (2026-10-02T05:52Z) to resolution, inside 48h, but the median across open tickets is still far over. Clause 3 one queue MET (35 open signatures at Gate 1). Clause 4 idle-is-free: prior evidence only. Clause 5 no regressions: preserved, done-gate passed and independently judged.
**Routing evidence**: base-gate1=76a5aef65c81ac2fb4657fafaeff4a89e9faaf75@2026-10-03T03:58:27Z; base-gate3b=27dab5bb40016f89c322a3b87cd8e122fe5566fb@2026-10-03T05:08:22Z (drift = this iteration's own merge only).
- **Controller**: `claude:claude-opus-5-5` (tok: not reported). SKILL plus all 12 resources: running copy (resolved symlink) = pin = origin/dev, per-file.
- **Designer**: no run this attempt. Rev 2 was by the attempt-1 designer `claude:claude-opus-5-5`. A designer Agent alias spawn was denied (`deny:provider-pin`, hook log 21:44:15Z); Rev 3 was controller-applied verbatim reviewer fixes (carve-out). Quorum r1 $0.0762 BLOCKED (sonnet-5/gemini/glm, author sol benched); r2 $0.2019 BLOCKED at N−1 (`gpt6-1-sol` absent: unreachable).
- **Planner**: resolver `agent-tool opus fail-closed:env-pin`; Agent `opus` ran (111,269 tok).
- **Executor**: resolver `recipe claude:claude-sonnet-5-5 declared:provider-pin`. The Agent `sonnet` spawn was DENIED by the hook (`deny:provider-pin`), the expected rule-(b) outcome. Routed to the pin via the `claude-sub` recipe (subscription-only wrapper, billing tripwire CLEAN): probe rc0, run rc0 (tok: not reported, text mode).
- **Evaluator**: resolver `reroute pi:openrouter/minimax/minimax-m3 generator-equals-judge` (sonnet judge = sonnet executor). Ran via `scripts/mission_pi_run.sh`, fenced, verdict `ok` rc0, 983s, 88 tool executions, 0 provider errors (in 67,556 / out 32,916 / cache-read 4,559,805 tok, $0.3334). The operator asked for Agent-tool spawns; the evaluator went over the pi recipe because the routing table re-routes it there, and the planner went over the Agent tool. The judge is independent of the generator (MiniMax vs Anthropic) and of the controller.
- **Metered**: $0.611 total ($0.278 quorum in attempt 1 + $0.333 evaluator).
- **Ration**: codex and ollama over (driver `MISSION_OVER_RATION`); OpenRouter admitted this fire.
**Verification**:
- Controller re-ran the gates rather than banking the executor's: capacity suite 14/14 rc0, chain suite 13 PASS rc0 with 0 `command not found`, production diff +7/−2 and in scope.
- Executor: `make test-launchd-drivers` rc0 (381s, 0 FAIL), 6/6 mutants red.
- Judge: PASS 95/100, no blocking findings, its own 4 mutants red. Its −5 was dry-runs "unmeasured". The controller closed that gap: the healthy and degraded dry-runs on `eb39db9a2` under the idle **world** profile (`env -i`, `AILANG_DRIVER_PINNED=<sha>`) both printed `DRY RUN ok`. The healthy arm reads `DEGRADED(2)` because codex is genuinely over ration; the degraded arm shows the forced bogus executor handed to sonnet. Evidence in `fleet-mission-evidence/iteration15/`.
- Fleet's own dry-run cannot run inside a fleet iteration (the overlap guard at :1895 precedes the dry-run exit at :1908), and docs is kill-switched.
- PR head `3c67b2b00`: 23/23 checks green. Merge `27dab5bb4`: dev CI run 37098847295 success, full check set 17/17 green (Gate 3b, SHA-pinned). LANDED. Ticket resolved with `--sha 27dab5bb4` (open count 35 → 34; reply sent to the filer).
**Ruled out**: widening the classifier to "hit your session limit" (outside D-FLEET-11 scope; a lane-selection change is HD-2a policy); a second pause `_mc_notify`; bypassing the overlap guard for dry-runs; banking the executor's greens; an Agent `sonnet` evaluator (same model as the executor); re-asking the resolved D-FLEET-10/11/12; reading or acking `mission-fleet` tickets; touching the 10 pending coordinator approvals.
**Retro lane**: process fix (charter). The fleet's done-gate dry-run is unsatisfiable from inside a fleet iteration. This is instance 2: iteration 9 also skipped it, and here the executor hit the same guard. The charter's done-gate now names the idle-sibling-profile recipe. Backlog only, no edit: the pi evaluator sandbox blocks `mktemp -t`/`-d` under `/var/folders`, so `make test-launchd-drivers` cannot run inside the pi judge (the judge shimmed `BASH_ENV` for the two suites). No shared skill edit and no reload.
**Next**: heartbeat ticket `skill:heartbeat-relative-path-absent-in-world` (D-FLEET-10 = A: Revision 5 with the absoluteness guard and Kimi's residuals, then a fresh quorum). Then the D-FLEET-9 rotate-log pair, then pi-runner pre-dirty. The D-FLEET-12 pre-authorized pair (`routing:retired-codex-models-still-spawnable-natively`, `quota:opencode-reported-as-a-pool`) is planner-ready without a further ruling.

## 16 — 2026-10-03 — heartbeat driver-root fix built + judged PASS 96; push blocked by the pinned scope guard, guard arm LANDED #1575 [HARNESS]

**Picked**: `skill:heartbeat-relative-path-absent-in-world` (charter P0 #1, `[NEXT]`; D-FLEET-10 = A and D-FLEET-8 = YES, both attended rulings). #1380: 0 new allowlisted directives since 2026-10-02T08:51:41Z (20 comments). D-FLEET-13 still OPEN, so the session-limit classifier was not taken. The ticket is still real at HEAD `c68ded4b2`: 18 relative `bash tools/launchd/mission-heartbeat.sh stamp` call sites in 14 files (9 per copy). No open fleet PR and no stale worktree for this item. The untracked 2026-09-29 plan/JSON in the pin is Revision-1 vintage and was ignored, not reused.
**Reality check**: running skill = resolved symlink = origin/dev, all 13 files, compared per file. Dev CI was green at the Gate-1 base `c68ded4b2` (42 checks, 0 not-green). The controller reproduced the designer's guard drill independently (bash + zsh: unset/empty/`.` → rc1, 0 rows; absolute → rc0, 1 row; absolute-nonexistent → rc127, 0 rows). It also measured the R5.2 premise rows first-party: in this live driver-spawned controller shell `printenv MISSION_DRIVER_ROOT` = the pin, with nothing exported by hand. The plists invoke `/bin/bash <absolute path>`, and `tools/launchd/lib/pin-root.sh:362` re-execs with an absolute `$wt`.
**Shipped**:
- **#1575 → `adab9b7d9`, MERGED.** `_scope_is_harness` gains `.agents/skills/mission-*|.agents/skills/sprint-*`, the Authority allowlist names both, and a changelog fragment is added.
- **Built and judged but NOT pushed: local branch `fleet/i16-heartbeat-rev5`** (head `9c513b8fe`, rebased on `adab9b7d9`; `git diff 14b87879b HEAD` over the judged paths is empty). It contains:
  - all 18 heartbeat calls → `case "${MISSION_DRIVER_ROOT:-}" in /*) bash "$MISSION_DRIVER_ROOT/tools/launchd/mission-heartbeat.sh" stamp … ;; *) echo … >&2; false ;; esac`
  - an `Attended setup:` sentence in each of the 14 files
  - rc-19 `provider_quota` added to the gate-3-route rc comment, in both copies
  - `.agents` produced by `sync-agents-skills.sh`
  - changelog `2026-10-03-heartbeat-driver-root.md`
  - design Revision 5.2 and its sprint plan/JSON

  The push was REFUSED: `mission scope guard: MISSION_NAME=fleet may not push .agents/skills/mission-control/resources/gate-5-retro.md`. The guard that ran is the pin's (`core.hooksPath` = `~/.ailang-driver-pin/fleet/tools/launchd/githooks`), which predates the arm the fix adds. No `--no-verify` and no self-selected hook; the arm shipped alone as #1575 instead.
**Progress**: goal unmoved (0 tickets resolved). Clause 1 UNMEASURED. Clause 2 turnaround UNMET: the heartbeat ticket has been open since 2026-09-26. Clause 3 MET (41 open signatures, `ailang mission ticket open --count` at Gate 5). Clause 4 prior evidence only. Clause 5 preserved: nothing unjudged shipped, and #1575's single arm is inside the PASS-96 diff.
**Routing evidence**: base-gate1=c68ded4b2d5e397d3719b32d6be591ad6fcd1bf8@2026-10-03T12:02:19Z. `MISSION_ROUTING_NOTE`: codex over daily ration → planner opus, executor sonnet-5-5.
- **Controller**: `claude:claude-opus-5-5` (tok: not reported).
- **Designer**: rotation last=`codex:gpt-6.1-sol` → glm/kimi (ollama over ration) → `claude:claude-opus-5-5`, resolver `recipe … declared:provider-pin`.
  - The operator-requested Agent `opus` spawn was DENIED (`deny:provider-pin`), as expected. Ran via `claude-sub`: probe rc0. Authoring run rc0 → R5 `21f045a38`; ONE revision run rc0 → R5.1 `5ad08e044`; within the Fable/Opus one-doc diet. (tok: not reported, text mode.) Rotation state written to `claude:claude-opus-5-5`.
  - Quorum, author benched:
    - r6 `12-07-53Z` BLOCKED $0.1953: kimi and gemini reject on premise rows; gpt6-1-sol ABSENT (OpenAI 429 no credits); glm ABSENT (invalid JSON, raw verdict pass).
    - r7 `12-14-33Z` BLOCKED $0.2312: gemini (swapped table cells), kimi and glm (runtime propagation, invocation shape, coherence, SKILL.md:338); sol absent.
    - All r7 objections were evidence-only and each carried a concrete fix. The controller applied them under the narrow-refinement carve-out → R5.2 `11a517b33`. Planner D1 path fix → `bd2df0b60`.
- **Planner**: resolver `agent-tool opus fail-closed:env-pin`; Agent `opus` (107,544 tok).
- **Executor**: resolver `recipe claude:claude-sonnet-5-5 declared:provider-pin`.
  - The operator-requested Agent `sonnet` spawn was DENIED (`deny:provider-pin`).
  - Ran via `claude-sub` (billing tripwire CLEAN): probe rc0, run rc0 (tok: not reported).
- **Evaluator**: resolver `reroute pi:openrouter/minimax/minimax-m3 generator-equals-judge`, but OpenRouter is in `MISSION_OVER_RATION`, and the resolver's reroute arm does NOT apply the ration gate (`resolve-role-spawn.sh` evaluator branch emits the chain head unchecked).
  - Declared chain walked with the ration applied: minimax skipped (over ration); `claude:claude-sonnet-4-6` skipped (same sonnet family as the executor per `family()`); `opus` via the Agent tool, allowed (`allow:alias-pin`), 112,695 tok.
  - Judge ≠ generator (opus vs sonnet). FLAG: same model as the designer and controller, in a fresh context.
- **Metered**: $0.4265 (quorum only).
- **Ration**: codex, ollama and openrouter over; Anthropic subscription OK.
**Verification**:
- **Executor**:
  - all acceptance items 1–8 + 6a
  - `make test-launchd-drivers` rc0 on the rerun; the first run hit a `test_hook_stdout.sh` 8 s timeout under load ≈97, and that test passed alone twice
  - 2 mutants red
- **Judge** (isolated detached worktree `fleet-i16-evaluator` at `14b87879b`):
  - **PASS 96/100**, 0 blocking
  - 40-case drill (bash/zsh × `-c`/script) TOTAL_FAILS=0
  - 6a: its own driver-descended shell = the pin
  - `make test-launchd-drivers` rc0 on the first run
  - 4 own mutants
  - −2 because no suite pins the guarded form; −2 for doc bookkeeping
- **Controller**: re-checked 18 guarded, 0 old-form, sync `--check` rc0, conflict-surface sites untouched. Guard arm `_scope_is_harness`: `.agents` mission/sprint → 0, model-manager → 1. `test_mission_scope_guard.sh` rc0.
- **#1575 head `82e03ae96`**: 23/23 green (the test job ran ≈36 min against a measured 24–30 min dev norm). Gate 3b (SHA-pinned): merge `adab9b7d9` dev CI run 37126587729 **success**; full check set 17, sole non-green `SonarCloud Code Analysis` (new-code coverage 54.9% < 80%) is INHERITED — also `failure` on parents `74d5a3b55`, `74a922d6b`, `e361f0cd2`; #1575 has no Go code. Dev reds on this repo are V1's (Gate 1 owner rule); recorded, not chased. #1575 LANDED.
- **Done-gate**: not claimed. No ticket was resolved, and the dry-runs are owed with the heartbeat push.
**Ruled out**:
- `git push --no-verify` and pointing `core.hooksPath` at the branch's own hook (an unattended loop widening its own guard in-flight)
- fast-forwarding the pin mid-fire (the driver runs from it)
- splitting the `.claude` half out alone (mirror drift reddens `test_agents_skills_sync`)
- Kimi's `|| { …; exit 1; }` snippet (kills a persistent shell; misattributes helper failures)
- aligning the three conflict-surface sites (out of D-FLEET-8/10 scope)
- re-asking D-FLEET-8/10
- reading or acking `mission-fleet` tickets
- touching the 3 pending coordinator approvals
**Retro lane**: process fix (charter Guardrails), "a fix that widens the scope guard ships ALONE, first". Instance 1 is iteration 12's design, which named the guard seam but not its bootstrap; instance 2 is this push refusal. Backlog, for an attended ticket filing because the fleet cannot file its own:
  - (a) `resolve-role-spawn.sh` evaluator reroute ignores `MISSION_OVER_RATION`
  - (b) judge finding: `bash tools/launchd/mission-base.sh …` (gate-1, gate-3 ×3, gate-3b ×2, gate-4, ref-drift) and `tools/launchd/resolve-role-spawn.sh` are also CWD-relative and rc127 from World. Same class as this ticket, outside its ruled scope.
  - (c) `skill:driver-root-guard-unify`: the `:-.` silent fallback at gate-3-route.md:618, plus the bare lane-dead calls
  - (d) no suite pins the guarded form or the `.agents` scope rows
  - (e) the spawn-pin hook compares the evaluator alias by string, not `family()`, so an Agent `sonnet` judge of a `claude:claude-sonnet-5-5` executor would be ALLOWED

  No shared skill edit and no reload.
**Next**: push `fleet/i16-heartbeat-rev5` (resume predicate in the charter queue row) → PR → Gate 3b → done-gate dry-runs (idle sibling profile) → resolve the ticket. Then the D-FLEET-9 rotate-log pair, then pi-runner pre-dirty. The D-FLEET-12 pair is planner-ready.

## 17 — 2026-10-03 — heartbeat fix pushed as #1578 and re-judged PASS 95; merge blocked by a weekend-only red in TestOllamaQuota* → PARKED-ON-CLOCK [HARNESS]

**Pick**: P0 #1 `skill:heartbeat-relative-path-absent-in-world` (8 slots lost, World), resuming iteration 16. Gate 0: kill switch armed. gh `sunholo-voight-kampff`. Billing CLEAN. 0 directives on #1380 since `2026-10-02T08:51:41Z`. Gate 1: pin = origin/dev `2a1f3f295`. SKILL.md and all 12 resources match origin, and the resolved symlink equals the pin copy. Dev CI `success`; the only non-green check of 17 is SonarCloud, inherited. Resume predicate run as a command: the pinned hook's `_scope_is_harness` has the `.agents/skills/mission-*` arm, so it is met. No new design, plan or code was needed, so the designer, planner and executor were not spawned. This is a resume of a judged build.
**Did**:
- Rebased `fleet/i16-heartbeat-rev5` onto `2a1f3f295` with no conflicts. Per-file blob check against the judged head `14b87879b`: 17 of 18 SAME. The DIFF is the design doc's one-line status header. The `pre-push` arm and the charter Authority line had already landed via #1575.
- Pushed. The pinned guard accepted the push. Opened PR #1578.
- Independent evaluator: **PASS 95/100**. Applied its two non-blocking doc nits in `ab0c539a1`: the stale M1 SHA `41e0005e9` became `bb573f466`, and the changelog section about the guard, which now ships in #1575's fragment, was removed.
- Gate 3b: the required `test` check and Build ubuntu/macos/windows are RED, every leg on the same two tests. `TestOllamaQuotaVerifiedLimits` (`ollama_quota_test.go:42`) and `TestOllamaQuotaHTTPAndCredentialBinding` (`:82`) build `limits` from `time.Now()`, with `WeeklyResetsAt = now+6d`. So the window starts yesterday, and `WeekdayPacePercent` counts only Mon–Fri hours (`quota_ledger.go:69`). On a weekend the allowance collapses to about 2.5% in the Fri→Sat window and 0% for Sat→Sun. #1524's 3pp ollama start margin (`quota_margin.go:33`) then marks even 0 usage `over`. It reproduces locally at `origin/dev` with no PR code, also under `env -i` with a temp HOME. Dev's own CI at 14:43Z was green because that window still held Friday hours. By arithmetic it goes green again around Mon 2026-10-05 08:00Z: test 1 needs about 6.6% allowance. `gh pr view 1578` reports `mergeStateStatus=BLOCKED`, and the required contexts are `test, lint, build, docs-gate`.
**Progress**: goal unmoved (0 tickets resolved). Clause 1 UNMEASURED. Clause 2 UNMET (the ticket has been open since 2026-09-26). Clause 3 MET (41 open signatures). Clause 4 prior evidence only. Clause 5 preserved (nothing merged).
**Routing evidence**: base-gate1=2a1f3f2952574356da196147b359acd2c11e4f65@2026-10-03T20:44:33Z; base=2a1f3f2952574356da196147b359acd2c11e4f65@2026-10-03T20:59:01Z. `MISSION_ROUTING_NOTE`: codex over daily ration → planner opus, executor sonnet-5-5 (not exercised).
- **Controller**: `claude:claude-opus-5-5` (tok: not reported).
- **Designer / planner / executor**: NOT SPAWNED. The item was a resume whose build was judged in iteration 16. There was nothing to design, plan or author beyond a rebase, a push and two doc-nit edits by the controller.
- **Evaluator**: resolver `reroute pi:openrouter/minimax/minimax-m3 generator-equals-judge`. OpenRouter is in `MISSION_OVER_RATION` (backlog (a) from iteration 16, still open). `claude:claude-sonnet-4-6` was skipped as the same family as the iteration-16 executor `claude:claude-sonnet-5-5`. Ran `opus` through the Agent tool, foreground, 76,510 tok, 25 tool calls, 9 min. Judge ≠ generator (opus vs sonnet-5-5). FLAG: same model as the controller, in a fresh context.
- **Metered**: $0.00. Chain `eefe0a7e` posted.
- **Ration**: codex, ollama and openrouter over; Anthropic subscription OK.
**Verification** (judge, isolated worktree `fleet-i17-evaluator`, since removed):
- 18 guarded sites (9 per copy), 0 bare matches (rc=1), `sync-agents-skills.sh --check` rc0.
- Drill from `/tmp` with a temporary `AILANG_STATE_DIR`, bash 3.2.57 and zsh: rc0 with an absolute root; rc1 with "must be absolute" for unset, relative and empty roots; 0 rc127. Control: the old bare form gives rc127.
- `make test-launchd-drivers` MAKE_RC=0 (6m25s, 167 ok). One mutant (bare form restored): grep 1 match, guarded count 17, sync check rc1.
- Controller: `go test ./internal/mission/...`. Only the two `TestOllamaQuota*` tests fail. Every other mission package is ok.
**Done-gate**: not claimed. The dry-runs are owed after the merge, and both armed siblings (world, stapledon) were mid-iteration this fire, so the overlap guard would yield before the dry-run exit.
**Ruled out**:
- Fixing the quota tests from the loop. D-FLEET-12 reserves the quota-margin work (`quota:admission-has-no-headroom-margin-and-flaps`, still an open ticket) for attended sessions. Also no ticket, no self-sourcing.
- Admin-merging past a required check (standing rule 2).
- Treating the red as caused by the PR. The PR has no Go code, and the red reproduces on `origin/dev`.
- Re-asking D-FLEET-13.
- Reading or acking `mission-fleet`.
- Touching the 3 pending coordinator approvals.
**Retro lane**: none new for the skill. The "scope-guard widening ships alone" guardrail worked as written: the push went through first try. New observation for Mark, not ticketable by the fleet: weekday pacing plus start margins make any `time.Now()`-anchored quota test calendar-dependent. Every PR to dev is blocked until about Monday 08:00Z, for every mission and every attended session. Suggested fix: pin `now` to a fixed weekday in both tests, e.g. `time.Date(2026, 9, 30, 12, 0, 0, 0, time.UTC)`, since `pace_test.go` already pins `paceLocation = time.UTC`.
**Next**: once `gh pr checks 1578` is all-green on the required checks, merge → Gate 3b SHA-pinned on the merge → done-gate dry-runs under an idle armed sibling → `ailang mission ticket resolve skill:heartbeat-relative-path-absent-in-world --sha <merge>`. Then the D-FLEET-9 rotate-log pair, then pi-runner pre-dirty. The record PR for this entry is blocked by the same red.

## 18 — 2026-10-04 — rotate-log pair built + judged PASS 97 as PR #1580; parked on the same weekend TestOllamaQuota* red; #1578 re-probed still red [HARNESS]

**Picked**: `mission:rotate-log-registry-cwd` + `rotate-log:status-flag-mutates-and-world-resolves-to-status-archive` (charter P0 #2, `[UNPARKED · D-FLEET-9 = A]`; iteration 17 banked it as next). P0 #1 (heartbeat) re-probed at Gate 2: `gh pr checks 1578` still red on `test`/`Build*` (03:05Z) — the weekend `TestOllamaQuota*` window, a clock park, so the queue moved to the next unblocked item. #1380: 0 new allowlisted directives since 2026-10-02T08:51:41Z (22 comments; watermark advanced to 2026-10-03T21:01:06Z after triage). D-FLEET-13 and D-FLEET-14 (filed iteration 17, on the i17-record branch) remain OPEN awaiting Mark. Kill switch armed, gh account `sunholo-voight-kampff`, billing tripwire CLEAN. Gate 1: pin = origin/dev = `2a1f3f295` (drift 11 on the main checkout's `dev`, pin authoritative); running skill (pin + resolved symlink, 13 files each) byte-identical to origin/dev; dev CI green at base with the standing SonarCloud red (V1-owned, recorded). Weekly external-issue sweep NOT due this fire (not the first fire after a Monday-07:00 rotation; due on the first fire after 2026-10-05 07:00Z).
**Reality check**: all ticket premises re-verified at HEAD `2a1f3f295` — `loadMissionRegistry` CWD/ancestor discovery confirmed (loader + 6 production calls + iteration default); `--status` mutates the status archive (flag loop → RotateLog); the driver exports `AILANG_MISSION_REGISTRY` ONLY in the binary branch (`mission-control.sh:1976`); prior design doc `m-mission-rotate-log-safe.md` (iteration 8) exists with rounds 1–2 blocked and the D-FLEET-9 = A ruling pending execution. The pin's untracked sprint JSON/plan were iteration-16 leftovers already inside PR #1578 (ignored, not adopted). CLAIM sent to controlplane (sibling telemetry dirty in the main checkout).
**Shipped**:
- **Design `m-mission-rotate-log-safe.md` Rev 3–5** (worktree `fleet-iter18-rotate-log`, commit `c2bf5bef0`): Rev 3 = complete shared-loader caller audit (C1–C11) + `--status` rejection specified before load + premises re-verified at HEAD; Rev 4 = C7 widened (normalize maps shared targets through the loaded registry's origin, loud no-registry failure preserved) + `m.root` formula pinned; Rev 5 = narrow-refinement carve-out, controller-applied reviewer-verbatim fixes (`repoRootFor` deletion specified; whole-repo `--status` caller enumeration with positive controls; V26/V27).
- **Sprint plan + JSON** (`3391da544`, planner kimi-k3): 4 milestones, 345 LOC, every acceptance command baselined at `c2bf5bef0` (B1–B12, incl. B10 RED→GREEN feature assertion and B11 UNINFORMATIVE-UNDER-SANDBOX labeling).
- **Implementation** (`db2a5b0b3` M1+M2, `ef6b9fe93` M3, `60f2b7716` M4, executor deepseek-v4.1-flash): additive `Mission.Root()`; strict `--stream log|status`; ruled `--status` rejection BEFORE registry load; rotate-log AND normalize shared targets via `m.Root()` with loud empty-root failure; `repoRootFor` deleted (0 references incl. tests); loader errors name tried locations; `_mc_export_mission_registry` (bash 3.2) in both driver branches with missing-root validation; 4-arm fixture wired into `make test-launchd-drivers`; 9 pinned synthetic tests (159 LOC, no t.Parallel, t.TempDir fixtures only); changelog fragment.
- **PR #1580** pushed through the pinned guard (which now carries #1575's `.agents` arm) — no refusal.
**Progress**: goal unmoved (0 tickets resolved; the pair resolves when #1580 merges). Clause 2 UNMET: the rotate-log ticket has been open since 2026-09-26. Clause 3 MET (41 open signatures). Clause 1 UNMEASURED. Clause 4 prior evidence only. Clause 5 preserved: nothing unjudged merged; every check the PR can reach is green and the sole red is the inherited weekend signature.
**Routing evidence**: base-gate1=2a1f3f2952574356da196147b359acd2c11e4f65@2026-10-04T03:02:54Z; worktree provenance base=2a1f3f295 (MATCH at snap). Resolver: planner `codex:gpt-6.1-sol anthropic-fallback:fail-closed:planner-lane-field-invalid` (doc's Planner-Lane field) — codex bucket over ration → walked the declared chain (ollama/kimi-k3:cloud over ration) → `pi:openrouter/moonshotai/kimi-k3`. The operator's standing request to spawn designer/planner/executor/evaluator via the Agent tool: this harness (pi controller) has no Agent/Task tool — every role ran as a model-pinned sub-agent via the skill's own cross-provider `scripts/mission_pi_run.sh` recipe (typed verdicts, sandboxed), which is what the routing table specifies for `recipe` lanes this fire; NO role ran inline on the controller's model, and the judge is a different model family from designer, planner, executor, and controller.
- **Controller**: `pi:openrouter/z-ai/glm-5.3` (session; tok: not reported by the harness).
- **Designer**: `pi:openrouter/z-ai/glm-5.3` (rotation resolved by the driver after anthropic/codex/ollama rungs refused, rc75) — probe rc0 (`--no-extensions`); r3 run rc0 234s, 40 turns, 1,762,806 tok gross (76,362 in / 26,415 out / 1,629,440 cache-read), $0.6468; r4 revision run rc0 213s, 32 turns, $0.6124.
- **Quorum** (author benched, Z-AI sits out): r3 BLOCKED $0.1235 — gemini reject (C7), kimi reject (path formula), sonnet ABSENT (quota), gpt6-1-sol ABSENT (auth); r5 on Rev 4 BLOCKED $0.1172 — gemini reject (orphaned `repoRootFor`), kimi reject (directory-scoped `--status` audit), same two absent. Both rounds' objections controller-verified first-party; round 5 applied under the carve-out.
- **Planner**: `pi:openrouter/moonshotai/kimi-k3` — probe rc0; run rc0 437s, 44 turns, 1,885,419 tok gross, $1.1465.
- **Executor**: `pi:openrouter/deepseek/deepseek-v4.1-flash` — probe rc0; run rc0 325s, 85 turns, 6,488,804 tok gross, $0.0505.
- **Evaluator**: `pi:openrouter/minimax/minimax-m3` — probe rc0; run rc0 376s, 53 turns, 1,932,479 tok gross, $0.1441. Judge ≠ generator (minimax vs glm/kimi/deepseek). Session handshake completed (protocol ack verified in its own session).
- **Metered**: $2.8410 total ($1.2592 designer + $0.2407 quorum + $1.1465 planner + $0.0505 executor + $0.1441 evaluator) against the $5 ceiling.
- **Ration**: codex, ollama, anthropic over daily ration (Sunday); openrouter charged metered.
**Verification**:
- **Controller** (out-of-sandbox): 9/9 new tests `-v` first-party; `go test ./cmd/ailang/... -run 'Mission' -count=1` ok; `make test-launchd-drivers` rc0 (59 arms + `mission registry env` suite); both dry-run arms `DRY RUN ok` (world profile, `AILANG_DRIVER_PINNED=2bf95391f`); diff scanned for out-of-worktree writes — none (fixtures t.TempDir only); main checkout status unchanged by the executor (sibling telemetry only).
- **Judge** (isolated worktree `fleet-iter18-evaluator` at `60f2b7716`, own branch): **PASS 97/100**, 0 blocking. Re-ran the full gate table (including a broader `repoRootFor` sweep: zero hits in cmd/internal/tools/scripts); re-ran the mutation matrix — 3/3 mutations RED on the named tests (wrong-repo rotation caught by the absolute-path pin; 0 headings rewritten; both `--status` tests red) and GREEN on restore (md5-verified). Adjudicated the executor's AC5 finding (grep 2 vs plan 1: the rejection arm's own error text contains `--status`; PASS). Non-blocking: a robustness note on `--stream` value parsing shape.
- **Gate 3b (PR #1580, checks read direct)**: `launchd drivers (bash 3.2)` SUCCESS (the new suite ran on CI), `lint`/`govulncheck`/`CodeQL`/`changes` SUCCESS; required `test` FAILURE at 04:09Z on exactly `TestOllamaQuotaVerifiedLimits` + `TestOllamaQuotaHTTPAndCredentialBinding` (job log read with `--allow-escape-sequences`, ANSI stripped, `--- FAIL` lines quoted; control: `=== RUN`/`PASS` lines present in the same log) — the D-FLEET-14 weekend signature, inherited from the base, outside this diff → **PARKED-ON-CLOCK**, same predicate class as #1578/#1579. No wait burned: the signature was confirmed from the run itself within one poll window.
- **Done-gate**: surface edited ✓; reach = PR #1580 pushed ✓; dry-runs healthy+degraded ✓ (world profile; `lanes=DEGRADED(14)` is the genuine Sunday all-buckets-over-ration state, not a code defect — each lane probed and handed off in chain order correctly); `make test-launchd-drivers` ✓; nothing reloaded mid-iteration ✓. Ticket resolution remains gated on the merge (Monday), per the queue row predicate.
**Ruled out**:
- fixing the `TestOllamaQuota*` tests from the fleet (D-FLEET-12: quota-margin work is attended; D-FLEET-14 pending Mark — default: wait for Monday)
- merging on an inherited red or arming auto-merge on it (auto-merge never clears a base-inherited red pinned to the PR head)
- basing the record on `dev` and conflicting with the unmerged i17 record (record 18 stacks on `fleet/i17-record` instead)
- re-quoruming after round 4 (carve-out conditions met: both objections reviewer-verbatim, concrete, direction-preserving; recorded V26/V27)
- spawning any role inline on the controller model, or letting the judge share a model with any generator
**Retro lane**: none new for the skill this fire. Two process observations: (a) the plan-predicted grep count (AC5) was wrong while the executor followed the plan exactly — the plan's baseline arithmetic was controller-planner-checked but not re-derived; a plan AC that counts its own output text needs the count re-derived after the text is fixed. (b) The first `mission-worktree.sh add` call with an abbreviated 9-char SHA created a worktree at a SHA-named path inside the pin — the helper accepts the short form silently; worth a one-line guard at an attended moment. Neither is ticketable by the fleet; both are recorded here for Mark.
**Next**: when `gh pr checks 1580` (and #1578/#1579) are green after Mon 2026-10-05 ~08:00Z: merge #1578 → done-gate dry-runs → resolve the heartbeat signature; merge #1580 → Gate 3b SHA-pinned on the merge → `ailang mission ticket resolve mission:rotate-log-registry-cwd --sha <merge>` + `... rotate-log:status-flag-mutates...`; merge the record PRs in order. Then pi-runner pre-dirty (P1 #5), then the Phase 3a skill-resolution directive. D-FLEET-13/14 await Mark.

## 19 — 2026-10-05 — both P0 heads LANDED: heartbeat #1578 and rotate-log pair #1580 (after a Windows walk-up hang fix); 3 tickets resolved [HARNESS]

**Pick**: resume both P0 rows, which were PARKED-ON-CLOCK on the weekend-only `TestOllamaQuota*` red. Gate 0: kill switch armed (`mission-fleet.disabled` absent). gh `sunholo-voight-kampff`. Billing CLEAN. 0 directives on #1380 since `2026-10-03T21:01:06Z` (25 comments). Gate 1: pin = origin/dev `9eac33b7b`. The running skill (resolved symlink, 13 files) is byte-identical to origin/dev. Dev CI `success`; the only non-green of 21 checks is SonarCloud, inherited (also `failure` on `2a1f3f295`, `adab9b7d9`, `74d5a3b55`). Records 17 and 18 had never reached dev (#1579 and #1581 were open), so this fire read them from their branches. Clock predicate run as a command: `go test ./internal/mission/ -run TestOllamaQuota -count=1` → `ok` on Monday 10:36Z.
**Did**:
- **#1578.** Re-ran the failed CI. Required `test`/`lint`/`build`/`docs-gate` all SUCCESS. The one non-required red, `Build macos-latest` (`TestMemprobeDebugLogDoesNotAccumulate`, a memory test; the PR has no Go code), passed on rerun attempt 3, so it was a flake. Merged with `--squash --match-head-commit ab0c539a1` → **`c55ca4398`**. Gate 3b: dev CI success; full check set 17, only the inherited Sonar red.
- **#1580: a real Windows defect, hidden all weekend behind the quota red.** `TestMissionNormalizeNoRegistryStillFailsLoudly` hung to the 10-min package timeout on both `test-windows` and `Build windows-latest`. Job logs were read with `--allow-escape-sequences`, sized at 4.1 MB and 131 KB, and the `running tests:` block names the test. Dev's last 5 commits are green on Windows. Cause: `loadMissionRegistry`'s walk `for d := wd; d != "/" && d != "."; d = filepath.Dir(d)` never terminates on Windows (`filepath.Dir(C:\) == C:\`). It pre-dates the PR (`origin/dev` `mission_cmd.go:178`); #1580's new test was the first to run it from a directory with no registry. The second dev site (`:392`, `repoRootFor`) had already been deleted by #1580.
  - Fix `cffc0447a` (executor). The helper `registryWalkCandidates` stops when `filepath.Dir(d) == d`, with the same Unix candidate list (root excluded before and after). It adds `TestRegistryWalkCandidatesTerminatesAtVolumeRoot` and the changelog fragment `2026-10-05-mission-registry-walk-windows.md`.
  - Pushed as a fast-forward onto the PR branch, through the pinned guard. Both Windows legs and all required checks then went green.
  - Merged with `--match-head-commit cffc0447a` → **`e7628b05e`**. Gate 3b: dev CI success; 21 checks, only the inherited Sonar red.
- **Resolved 3 tickets** (open 41 → 38): `skill:heartbeat-relative-path-absent-in-world` (`c55ca4398`, replied to world), `mission:rotate-log-registry-cwd` (`e7628b05e`, replied to stapledon and v1), `rotate-log:status-flag-mutates-and-world-resolves-to-status-archive` (`e7628b05e`, replied to world).
- **Records.** #1579 merged (`845bbbbdf`). #1581 (iteration 18) conflicted after that squash, so its commit was cherry-picked cleanly onto dev in this record branch, which supersedes it.
- **Weekly external-issue sweep** (first fire after the Monday rotation boundary): per-issue table in the charter queue. 0 new queue items. `#1306` got a half-fixed verdict comment (comment count 0 → 1, still OPEN).
**Progress**: 3 tickets resolved, the first since iteration 15. Clause 1 UNMEASURED. Clause 2 UNMET: the three resolutions took about 8–9 days each from filing, against a ≤48h median target. Clause 3 MET (38 open signatures). Clause 4 prior evidence only. Clause 5 preserved: every merge was judged and the required CI on each merge SHA was green.
**Routing evidence**: base-gate1=9eac33b7b62058c9718930237cbbd696ceae7040@2026-10-05T10:36:08Z. `MISSION_ROUTING_NOTE`: controller pi:openrouter/z-ai/glm-5.3 → claude:claude-opus-5-5 (probe ok); codex over daily ration → planner opus, executor claude:claude-sonnet-5-5.
- **Controller**: `claude:claude-opus-5-5` (tok: not reported).
- **Designer / planner**: NOT SPAWNED. Both P0 items were judged builds, and the Windows fix is a one-function termination guard whose defect, cause and fix were measured from the CI log. A design doc or plan would have added a gate and nothing to judge.
- **Executor**: resolver `recipe claude:claude-sonnet-5-5`. Ran via `claude-sub` in worktree `~/.ailang-worktrees/fleet-i19-win-walk`, billing tripwire CLEAN. Probe rc0, run rc0 (tok: not reported, text mode). Wrote `cffc0447a`.
- **Evaluator** (×2): resolver `reroute pi:openrouter/minimax/minimax-m3 generator-equals-judge`. OpenRouter is in `MISSION_OVER_RATION`, so minimax was skipped, and `claude:claude-sonnet-4-6` was skipped as the executor's family. Ran `opus` via the Agent tool, foreground, `MISSION-ROLE: evaluator`.
  - Judge 1 (78,553 tok, 24 tools, 24 min): #1578 **PASS 96**, #1580 **PASS 96**.
  - Judge 2 (52,981 tok, 13 tools, 10 min): `cffc0447a` **PASS 92**.
  - Judge ≠ generator (opus vs sonnet-5-5 / deepseek / glm / kimi). FLAG: same model as the controller, in a fresh context.
  - The first spawn was DENIED `fail-closed:role-missing` because the role token shared the first line with prose. Re-spawned with the token alone on line 1.
- **Metered**: $0.00. **Ration**: codex, ollama and openrouter over; Anthropic subscription OK.
**Verification**:
- **Judge 1, #1578**:
  - 9+9 guarded sites, 0 bare; sync check rc0; `test_agents_skills_sync.sh` rc0.
  - 36-run drill from `/tmp` under `env -i` with bash 3.2.57 and a temp state dir: absolute → rc0 ×9; unset, empty or relative → rc1 with "must be absolute"; control old form → rc127.
- **Judge 1, #1580**:
  - Delta `60f2b7716..2bf95391f` is docs only.
  - `go test ./cmd/ailang/... -run Mission` ok; `go test ./internal/mission/...` ok; `make test-launchd-drivers` rc0 (442 s).
  - Own mutants M1 (CWD target) and M2 (`--status` through) both red; restore green.
- **Judge 2, `cffc0447a`**:
  - Old-vs-new equivalence program on 9 Unix path shapes: all equal.
  - Windows `C:\`, UNC and drive-relative cases reasoned from Go 1.26 `filepathlite.Dir`.
  - `GOOS=windows go vet` rc0; `changelog_fold.sh --check` rc0.
  - Mutant B (guard dropped) red: "did not terminate". Mutant A (old guard) green on Unix: only Windows CI distinguishes it (non-blocking).
- **Controller**:
  - Guarded stamp from merged dev run from `/tmp` in this live driver-spawned shell → rc0 (this wrote the gate-4 heartbeat early, at 11:31Z).
  - `make test-launchd-drivers` MAKE_RC=0 at `e7628b05e` (504 s).
  - Driver diff `2bf95391f..e7628b05e` = 0 lines.
**Done-gate**:
- Surface ✓.
- Reach ✓ (both merges on origin/dev).
- `make test-launchd-drivers` ✓.
- Nothing reloaded ✓.
- Dry-run healthy/degraded **not re-run this fire**: no armed sibling was idle (world pid 18901 and stapledon pid 19130 ran all fire; v1, motoko and docs are disabled and exit at the kill switch). Carried instead:
  - #1578 changes 0 driver bytes.
  - #1580's driver bytes equal `2bf95391f`, which iteration 18 dry-ran healthy and degraded (world profile).
**Ruled out**:
- Admin-merging past the Windows red (standing rule 2), or treating it as a flake (it reproduced in two jobs, named one test, and has a deterministic cause).
- Fixing the walk in a separate PR after merging #1580 (that would merge a known hang into every Windows CI run).
- Spawning a designer or planner for a one-function guard.
- Touching `TestOllamaQuota*` (D-FLEET-12 / D-FLEET-14).
- Re-asking D-FLEET-13/14.
- Reading or acking `mission-fleet`.
- Touching the 3 pending coordinator approvals.
- Merging #1581 over its conflict.
**Retro lane**: none for the skill (one instance each). Two observations for Mark:
  - (a) A clock park can hide a real red. While every PR is red on the weekend signature, other reds on the same PR are never read. #1580's Windows hang sat behind the quota red for two fires. Resume predicates should re-read the whole failing set once the clock clears, as this fire did.
  - (b) D-FLEET-14 recurs every weekend. Next Saturday it blocks every PR to dev again, fleet and attended alike.
**Next**: P1 #5 `pi-runner:verdict-blind-to-commits-and-predirty` (pre-dirty half), then the Phase 3a skill-resolution directive (after P1). The D-FLEET-12 pair is planner-ready. D-FLEET-13/14 await Mark.

## 20 — 2026-10-05 — pi-runner pre-dirty fix built + judged PASS 86→85 as PR #1593; landing blocked by a GitHub Actions major outage → PARKED-ON-CLOCK [HARNESS]

**Pick**: P1 #5 `pi-runner:verdict-blind-to-commits-and-predirty`, the pre-dirty half (charter `[NEXT]`; Mark's 2026-09-29 triage order).
- Gate 0: kill switch armed (`mission-fleet.disabled` absent); gh `sunholo-voight-kampff`; billing CLEAN; 0 directives on #1584 since `2026-10-05T12:33:09Z` (1 comment, public). 38 open tickets.
- Gate 1:
  - Pin = origin/dev `a12a319b5`. Running skill: `SKILL.md` + 5 resources match origin. 7 gate resources DIFFER: the resolved main checkout is 22 commits behind origin. The delta is the absolute `MISSION_DRIVER_ROOT` heartbeat-stamp form + the pi rc-19 line (#1578). This fire followed the resolved copy but used the absolute stamp form.
  - Dev CI: last push run `d273ea1cd` failure = `test-windows`, `cmd/ailang` at 401 s / 96% of its timeout budget, with hang-guard FAILs (not logic). The 2 prior runs were green; V1 owns this lane, recorded only.
  - HEAD `a12a319b5` (6 dependabot merges) had **no** push CI. Dispatched `CI` → `test` success, `lint` success; `changes` cancelled (unassigned 15 min), so Windows is unverified at HEAD.
- Gate 2: the pick still reproduces at HEAD. `mission_pi_run.sh:346` gates `ok` on porcelain non-empty, and `:347` says a pre-dirty tree reads `ok` vacuously. No orphan iteration-20 PR or worktree. #1583 (`fix/ollama-quota-test-clock`, attended) addresses D-FLEET-14's subject; not fleet's PR, untouched.

**Did**:
- **Plan** (opus Agent) `6f60e0d53`, `design_docs/planned/sprint-plan-pi-runner-predirty.md`. Two load-bearing findings, both verified by the controller:
  - TEST 1/5 started dirty with no-op stubs, so they passed only because of the bug.
  - `scripts/test_mission_pi_run.sh` ran in no gate (`make/test.mk` listed only the three `tools/launchd/test_mission_pi_run_*` suites).
- **Executor r1** `d3974c34f`:
  - A worktree fingerprint is taken right after `BASE_HEAD` and again after the run: porcelain v1 -z + `git diff --binary HEAD` + untracked hashes → `git hash-object --stdin`.
  - `ok` = fingerprint changed OR commits made.
  - New JSON fields `predirty_files` and `worktree_changed_since_start`.
  - Header comment fixed; the suite wired into `make test-launchd-drivers`; changelog fragment added.
- **Judge r1 PASS 86**, 0 blocking. Finding 1 was reproduced first-party: the batched `xargs git hash-object` aborts at a dangling symlink (`fatal: could not open 'aaa'`, rc=1), so a later untracked edit reads `empty_worktree` (a live lane reported dead).
- **Executor r2** `2e49cc9ac`: untracked entries are hashed one at a time, with stand-ins (symlink target; nested repo HEAD + porcelain). Arms 8.8–8.11 added.
- **Judge r2 PASS 85**, 0 blocking.
- Pushed through the pinned guard (`core.hooksPath` = pin); PR **#1593**.
- **Gate 3b**:
  - `lint` and `docs-gate` green (docs-gate after one re-run). `test` still running at record time.
  - `changes` sat unassigned 15 min and was cancelled, so `launchd drivers` and `test-windows` were skipped.
  - githubstatus: **Actions: major_outage**, "Incident with Actions" investigating since 21:09Z. The same symptom hit the 19:17Z dev dispatch.
  - Not merged; ticket NOT resolved.

**Progress**: 0 tickets resolved (38 open).
- Clause 1 UNMEASURED.
- Clause 2 UNMET.
- Clause 3 MET.
- Clause 4 prior evidence only.
- Clause 5 preserved: nothing merged, and the judged head waits for a real `pull_request` CI suite.

**Routing evidence**: base-gate1=a12a319b5eb9b208f4d8f5df21b6a892033b08e7@2026-10-05T19:14:56Z. `MISSION_ROUTING_NOTE`: codex over daily ration → planner opus, executor claude:claude-sonnet-5-5.
- **Controller**: `claude:claude-opus-5-5` (tok: not reported).
- **Designer**: NOT SPAWNED. The resolver said `recipe claude:claude-opus-5-5`. The defect, cause and fix shape were measured at HEAD in one script, so the charter's "design only when the fix warrants one" applies, as in iteration 19.
- **Planner**: resolver `agent-tool opus fail-closed:env-pin` → opus via the Agent tool, foreground (87,227 tok, 17 tools, 13 min).
- **Executor**: resolver `recipe claude:claude-sonnet-5-5` → `claude-sub` in `~/.ailang-worktrees/fleet-i20-pi-predirty`. Billing CLEAN; probe rc0; r1 rc0, r2 rc0 (tok: not reported, text mode).
- **Evaluator**: resolver `reroute pi:openrouter/minimax/minimax-m3 generator-equals-judge`.
  - OpenRouter is in `MISSION_OVER_RATION`, so minimax was skipped. `claude:claude-sonnet-4-6` was skipped as the executor's family.
  - Ran `opus` via the Agent tool, foreground for r1 (92,747 tok, 20 tools, 22 min), resumed by SendMessage for r2 (105,477 tok cumulative, 16 min).
  - Judge ≠ generator (opus vs sonnet-5-5). FLAG: same model as the controller and the planner, in a fresh context.
- **Metered**: $0.00. **Ration**: codex, ollama and openrouter over; Anthropic subscription OK.

**Verification** (judge, first-party re-runs):
- `scripts/test_mission_pi_run.sh` 36/36.
- commits 9, sandbox 13, provider_quota 29, all rc0.
- `make test-launchd-drivers` rc0, now running this suite.
- `bash -n` rc0.
- shellcheck identical to base.
- `check-changelog` rc0.
- Mutations red-on-mutate, each restored byte-identical (cmp + `git diff --quiet`):
  - `ok` reverted to porcelain;
  - diff-only fingerprint;
  - porcelain-only fingerprint;
  - untracked hashes dropped;
  - batched `xargs` restored;
  - time-varying fingerprint;
  - fingerprint diffed against `BASE_HEAD`;
  - post-run fingerprint failure counted as changed.
- One judge mutation SURVIVED: the stand-in block replaced with `|| true` → 36/36, so no arm pins the stand-in (non-blocking follow-up).
- 2,000-untracked-file timing: 18.06 s per-entry vs 0.098 s batched, same hash. It runs outside the wall clock.

**Done-gate** (pending landing):
- Surface ✓ (`scripts/mission_pi_run.sh` is what every pi lane runs).
- Reach ✗ (not on origin/dev).
- `make test-launchd-drivers` ✓ on the branch.
- Nothing reloaded ✓.
- No dry-run needed: 0 driver bytes changed.

**Ruled out**:
- Merging past the skipped `launchd drivers`/`test-windows` legs, or treating an outage green as a verdict (Gate 3b).
- A third executor round for the judge-r2 non-blocking findings (one revision, bounded; queued as follow-ups instead).
- Spawning a designer for a one-script fix.
- Touching `TestOllamaQuota*` (D-FLEET-12/14) or #1583.
- Reading or acking `mission-fleet`.
- Re-asking D-FLEET-13/14.

**Retro lane**: none for the skill. One observation: the GitHub `changes` path-filter job is a single point of failure. When it cannot get a runner, every path-gated leg is SKIPPED, which renders as neutral rather than red. A rollup that counts only failures reads that as clean.
**Next**: land #1593 when Actions recovers (resume predicate in the queue row), resolve the ticket and reply to world. Then the Phase 3a skill-resolution directive. The D-FLEET-12 pair is planner-ready; D-FLEET-13/14 await Mark.

## 21 — 2026-10-06 — pi-runner pre-dirty fix LANDED: #1593 `c2bf04af3`, re-judged PASS 100 after the Actions incident cleared; ticket resolved [HARNESS]

**Pick**: resume of P1 #5 `pi-runner:verdict-blind-to-commits-and-predirty` (pre-dirty half). It was iteration 20's PARKED-ON-CLOCK item, and its predicate was met at Gate 2.
- Gate 0: kill switch armed (`mission-fleet.disabled` absent); gh `sunholo-voight-kampff`; billing CLEAN; 0 directives on #1584 since `2026-10-05T12:33:09Z` (2 comments, both public). Rotation-week catch: #1380 also had 0 directives since the watermark. 39 open tickets.
- Gate 1:
  - Pin = origin/dev `a12a319b5`.
  - Running skill: `SKILL.md` + 5 resources match origin; 7 gate resources DIFFER. The resolved main checkout is 22 commits behind (the same #1578 stamp-form delta as iteration 20). This fire followed the resolved copy and used the absolute stamp form. This is open ticket P1 #6, measured live.
  - Dev `CI` at `a12a319b5` was `failure`: iteration 20's `workflow_dispatch`, where `changes` was cancelled with no runner during the incident. Not a code verdict.
  - Full check set: 14 checks; Sonar red (inherited); `changes` cancelled.
- Gate 2: predicate re-run as commands, not transcribed.
  - githubstatus: Actions `operational`; "Incident with Actions" resolved 2026-10-05T22:49:42Z.
  - #1593 head `2e49cc9ac`: required `test`/`lint`/`build`/`docs-gate` green, plus `launchd drivers` and `test-windows` (CI attempt 2).
  - `UI build gate` red: 0 steps, empty runner, cancelled at 15 min. An outage casualty; re-run → success (attempt 2, post-incident).
  - Base still `a12a319b5`, so the judged bytes = the PR head.

**Did**:
- **Re-judged before merge.** Every PR green had run during the incident, which licenses a code inference only (Gate 3b).
  - Resolver: `reroute pi:openrouter/minimax/minimax-m3 generator-equals-judge`. openrouter was not over ration this fire.
  - The judge ran via `scripts/mission_pi_run.sh` in the isolated worktree `fleet-iter21-evaluator` at `2e49cc9ac`; session handshake acked (5× `acked:true` in its NDJSON).
  - **PASS 100**, 0 blocking.
- Squash-merged #1593 `--match-head-commit 2e49cc9ac` → `c2bf04af3`.
- Merged iteration 20's record #1594 → `eb2850427`.
- Resolved the ticket with sha `c2bf04af3`; world replied.
- Handed two dev reds to V1 (owner), message `inbox_1791263210185`; body verified via list.

**Gate 3b**:
- `CI` on `c2bf04af3` = **success** (run 37414632276, 04:38–05:05Z, after the incident). All 10 jobs green, including `launchd drivers (bash 3.2)` and `test-windows`.
- Full check set (17), not-green on `c2bf04af3`:
  - **Sonar**: inherited.
  - **`ailang-core-dev` Cloud Build failure**: step 7 `build-agent-base`, `node v22.23.3 != pinned v22.23.2`. NodeSource's `setup_22.x` moved; the pin is in `docker/Dockerfile.agent-base`, outside fleet Authority. It fails on `eb2850427` too; the last success was `58dd06d` (10-05 13:43Z).
  - **`Build macos-latest`**: `TestMemprobeDebugLogDoesNotAccumulate`, the known flake (iteration 19). It passed on #1593's PR head.
  - **2 `cancelled`**: fail-fast siblings.
- `CI` on `eb2850427` (docs-only) = failure: `test-windows` `cmd/ailang` hang guard (TestStdNumeric*, TestTailCallTrace*: "this PACKAGE has outgrown its go test -timeout budget"). It passed on the parent `c2bf04af3`. Second instance after `d273ea1cd`. V1's lane: recorded and handed over.

**Progress**: 1 ticket resolved (39 → 38 open).
- Clause 1 UNMEASURED.
- Clause 2 UNMET: this ticket took ≈9.6 days from filing (2026-09-26) to resolve.
- Clause 3 MET.
- Clause 4 prior evidence only.
- Clause 5 preserved: judged twice, and dev CI green on the merge SHA after the incident.

**Routing evidence**: base-gate1=a12a319b5eb9b208f4d8f5df21b6a892033b08e7@2026-10-06T03:50:59Z; base=eb28504275ce09ec57a62b21e9a964cdc30e9422@2026-10-06T04:48:03Z (gate4). `MISSION_ROUTING_NOTE`: codex over daily ration → planner opus, executor `claude:claude-sonnet-5-5` (neither spawned).
- **Controller**: `claude:claude-opus-5-5` (tok: not reported).
- **Designer / Planner / Executor**: NOT SPAWNED. This was a resume of work iteration 20 had already planned, built and judged; there was nothing to design, plan or execute. The operator's standing request to use the Agent tool for roles was honoured where a role ran.
- **Evaluator**: the resolver said `reroute pi:openrouter/minimax/minimax-m3`, a `recipe` lane, so the Agent tool is not the specified path. Ran via `mission_pi_run.sh`: pi_rc 0, 2094 s, 112 tool executions, 113 turns, 7,415,860 tok gross, $0.5452.
  - The runner verdict was rc 10 `empty_worktree`. Cause: the CONTROLLER's directive put the report under `.ailang/state/` (gitignored, `.gitignore:105`), so the porcelain check could not see it.
  - The report existed and was fresh: mtime 06:37 local, after the 06:02 start.
  - So this was an instrument artifact of the directive, not a lane failure. The verdict was taken from the report.
  - Judge ≠ generator: minimax vs sonnet-5-5 (executor) and opus (controller/planner).
- **Metered**: $0.5452. **Ration**: codex and ollama over; openrouter and Anthropic OK.

**Verification** (judge, first-party, in its worktree):
- `scripts/test_mission_pi_run.sh` 36/36; pi_run suites commits 9/9, sandbox 13/13, provider_quota 29/29; `bash -n` rc0 on both scripts; `check-changelog` rc0; shellcheck: no new findings.
- Mutations, each restored by `cp` + `cmp` rc0:
  - M-a (`ok` = porcelain only): 4 red, including 8.1;
  - M-b (untracked hashes dropped): 4 red, including 8.6;
  - M-c (post fp = pre fp): 12 red.
- Hand repro in a scratch repo:
  - A: mutated (old) code, pre-dirty, no-op pi → `ok` (the bug);
  - B: fixed code, same input → rc 10;
  - C: fixed code with a further edit → `ok`.
- Its `make test-launchd-drivers` gave rc 2 inside the pi sandbox, with the SAME 68 failures at the base `a12a319b5`. It is a sandbox artifact (`mktemp` denied under `DARWIN_USER_TEMP_DIR`, so the judge shimmed `mktemp`). The CI `launchd drivers (bash 3.2)` leg is green on both the PR head and `c2bf04af3`.

**Done-gate**:
- Surface ✓ (`scripts/mission_pi_run.sh`).
- Reach ✓ (`c2bf04af3` is an ancestor of origin/dev).
- `make test-launchd-drivers` ✓ (CI leg green on the merge SHA).
- 0 driver bytes, so no dry-run.
- Nothing reloaded ✓.

**Ruled out**:
- Merging on the incident-time greens alone (re-judged, and the merge SHA got its own post-incident CI).
- Fixing the node pin or the Windows test budget (outside Authority; handed to V1).
- Ticketing the judges' non-blocking follow-ups (no product loop has hit them; ticket-driven only).
- Reading or acking `mission-fleet`.
- Re-asking D-FLEET-13/14.

**Retro lane**: none for the skill. Two observations, one occurrence each:
- (1) A directive that sends a pi evaluator's report to a gitignored path makes `mission_pi_run.sh` read `empty_worktree` even when the judge did its work. The fix is the directive's path (a tracked scratch path, or check `check-ignore` before choosing one). Not yet ≥2 frictions.
- (2) The minimax judge ran `git stash` / `git stash pop` around a baseline run. The shared stash stack was empty (verified before the pop), so nothing was lost. A non-empty stack would have popped another session's entry.

**Next**: P1 #6 `skill-surface:main-checkout-not-synced-to-dev`. Measured live this fire: 22 behind, 7 gate resources differ. It feeds the Phase 3a skill-resolution directive. The D-FLEET-12 pair is pre-authorized; D-FLEET-13/14 await Mark.

## 22 — 2026-10-06 — Gate-0 self-notice read built + judged PASS 87→91→93 as PR #1604; merge blocked by an inherited dev red (07e1a89bc) → PARKED-ON-CLOCK; P1 #6 parked as D-FLEET-15 [HARNESS]

**Pick**: P1 #7 `gate0:driver-crash-notices-invisible` (ailang#1160). Mark's queue head, P1 #6 `skill-surface:main-checkout-not-synced-to-dev`, was skipped for a measured reason. Its fix is either (i) the driver auto-fast-forwarding the shared main checkout, whose standing authorisation `gate-1-observe.md` reserves to Mark, or (ii) Phase 3b per-fire skill pinning, which the 2026-09-26 directive gates on the 3a spike and parks for Mark. Neither is a mechanical fix the fleet can land alone, so it is filed as **D-FLEET-15** (rec A, default B). Measured this fire: the main checkout is `1 0` (ahead/behind origin/dev), so the symptom was not live.
- Gate 0: kill switch armed (`mission-fleet.disabled` absent); gh `sunholo-voight-kampff`; billing CLEAN; 0 directives on #1584 since `2026-10-05T21:22:17Z` (3 comments, all public). Not a sweep week (iteration 19 swept).
- Gate 1:
  - Pin = origin/dev `e68a264fb`, drift 0.
  - Running skill: `SKILL.md` + all 12 resources byte-identical to origin, for both the resolved symlink and the pin copy.
  - Dev `CI`, `Build and Release` and `Docs-Deploy` all success on `e68a264fb`. Full check set: 21 checks; Sonar red (inherited), `ailang-core-dev` Cloud Build red (node pin, handed to V1 at iteration 21).
- Gate 2: premise re-verified at HEAD. `gate-0-preflight.md` had no self-notice read (`grep -n 'slot-verdict\|rc=143\|crash'` hit only unrelated prose). The driver literal is at `mission-control.sh:2514`. World's implementation is `ailang-world` `50103fd` (#137).

**Did**:
- Planner opus (Agent, foreground): plan `6934d2a0b`. Premises it corrected:
  - The shared namespace has no non-rotating watermark, so the rule passes both issue-scoped files.
  - World's script holds the prefix twice (bash and a `\u`-escaped python copy), and its `--driver-src` check covered only the bash copy. The port keeps a single literal.
  - `--control 107:4` is World-only. Measured controls: `ailang#852` = 10, `world#107` = 4, `stapledons-godot#4` = 5.
  - World has no `scripts/` copy, so the root comes from the driver.
  - A second death notice, `Mission slot verdict`, is out of scope (follow-up).
- Executor `claude:claude-sonnet-5-5` (`claude-sub`, probe rc 0):
  - Run 1 died after 227 s with `API Error: Server error mid-response`, M1 committed (`a3cbffe1e`). The remaining chain was ollama/openrouter, both over ration, so the same lane was retried with a resume note.
  - Run 2 re-verified M1 and finished M2–M4: `0336d0e30`, `7b7b39afc`, `40fb95a7f`.
  - Round 2 `949d5952b` and round 3 `2e0f92672` fixed the judge's findings.
- Deliverables:
  - `scripts/mission_gate0_self_notices.sh`: exit codes 0/1/2, floors F0–F10, bounded `gh`, never prints a body.
  - `scripts/test_mission_gate0_self_notices.sh`: 108 arms, wired into `make test-launchd-drivers`. World's four fixtures are `cmp`-identical, plus a real #1380 capture under `scripts/mission_gate0_self_notices_testdata/`. The `driver-literal` arm turns red if the driver's prefix drifts.
  - Gate 0 step 6a in both skill copies (`cmp`-identical): a second read with no authority. rc 1 sends Gate 2 to its died-mid-flight traces and gives a log credit once per `url=`. rc 2 is an instrument or precondition failure, not a verdict.
  - Changelog fragment.
- Judge opus (Agent, own worktree `.wt-fleet-iter22-judge`), 3 rounds:
  - **R1 PASS 87**, 0 blocking. Mutants m1, m2, m5–m9 and `drv` were killed. m3 (prev-issue read dropped) SURVIVED, because the prev issue always doubled as the control. F3: the 6a block failed as written in a fresh shell (`$ISSUE` unset; `${…:?}` inside `$(…)` gave rc 127).
  - **R2 PASS 91**: a new arm kills m3, and the block is self-contained. New finding R2-1: a precondition failure printed `rc=1`, the "fire died" token.
  - **R3 PASS 93**: preconditions now print `rc=2 (… not a verdict)`. Cases B–E verified rc=2, and A printed rc=1 for real data.
- PR #1604 opened with no closing keyword (`Refs #1160`).
- Handed dev's new red to V1: `inbox_1791290588115_58891a9a`.

**Gate 3b / blocked**: #1604's `test` and `test-windows` fail `TestValidateModulePath_SingleFileInsidePackage` (`package_layout_test.go:107`). The negative control is dev `07e1a89bc` (attended `fix(pipeline)…MOD010…`, 12:07Z), which fails the SAME test on `test` and `test-windows` (`CI=failure`, `Build and Release=failure`). `8cddc7e2a` sits on top of it. The diff has no Go, `internal/pipeline` is outside fleet Authority, and V1 owns the repo's reds. Not merged; ticket stays open.

**Resume predicate**: dev `CI` success on a SHA containing a fix for that test → `gh run rerun --failed` on #1604 (head `2e0f92672`) → required `test`/`lint`/`build`/`docs-gate` + `launchd drivers` + `test-windows` green → merge `--match-head-commit 2e0f92672` → Gate 3b on the merge SHA → `ailang mission ticket resolve gate0:driver-crash-notices-invisible --sha <merge>` → comment the verdict on #1160 (`--body-file`), then close it.

**Done-gate (pre-merge)**:
- Surface ✓: `scripts/mission_gate0_self_notices.sh` + both `gate-0-preflight.md` copies, the files every mission's Gate 0 reads.
- Reach: pending merge.
- `make test-launchd-drivers` rc 0 (executor and judge, unpiped).
- 0 driver bytes, so no dry-run.
- Nothing reloaded.

**Live finding**: the instrument's first fleet reading (rc 1) is the real `rc=143` KILLED-at-gate-5 slot of iteration 18 (2026-10-04T04:28:01Z; `/tmp/ailang-mission-fleet.log`: `STALL … killing early`, `slot-verdict: KILLED at=gate-5 rc=143`). Iteration 18's Gate-4 record had already landed, so nothing was orphaned. Credited here once (url on #1380).

**Progress**: goal unmoved (0 tickets resolved; 38 open).
- Clause 1 UNMEASURED.
- Clause 2 UNMET.
- Clause 3 MET.
- Clause 4 prior evidence only.
- Clause 5 preserved: nothing merged.

**Routing evidence**: base-gate1=e68a264fbea787896a963f0fd7ae35cb321bf64b@2026-10-06T11:41:04Z; worktree base=e68a264fb (MATCH at snap); base-gate4=8cddc7e2ac8120429dc8904ad8cbeda12a8f6c91@2026-10-06T12:45:24Z (dev advanced by 4 attended commits, record branched from there). `MISSION_ROUTING_NOTE`: codex over daily ration → planner opus, executor `claude:claude-sonnet-5-5`.
- **Controller**: `claude:claude-opus-5-5` (tok: not reported).
- **Designer**: NOT SPAWNED. This is a port of World's implemented, judged design (`w-gate0-blind-to-its-own-crash-notices.md`); the plan cites it.
- **Planner**: opus via Agent (resolver `agent-tool opus fail-closed:env-pin`), 119,537 tok.
- **Executor**: `claude:claude-sonnet-5-5` via `claude-sub` recipe (`declared:provider-pin`), 4 runs (227 s died / ~13 min / round 2 / round 3), tok not reported (`claude -p` text output).
- **Evaluator**:
  - The resolver said `reroute pi:openrouter/minimax/minimax-m3 generator-equals-judge`, but openrouter was **over ration** (`ailang mission quota`: $3.09 against $2.33/day).
  - The `reroute` arm of `resolve-role-spawn.sh` never calls `_rs_over`, unlike `emit_recipe`.
  - Recorded dead via `mission-lane-dead.sh`: minimax (over ration), `claude:claude-sonnet-4-6` and `sonnet` (family sonnet = executor).
  - Spawned the declared bare fallback **opus** via Agent: 114,415 + 128,471 + 135,141 tok over 3 rounds.
  - Judge ≠ generator: opus vs sonnet-5-5.
- **Metered**: $0. **Ration**: codex, ollama and openrouter over; Anthropic OK (5h 33%, week 24%/25.7%).

**Ruled out**:
- Working P1 #6 autonomously (needs Mark: D-FLEET-15).
- Fixing `TestValidateModulePath_SingleFileInsidePackage` (language core, outside Authority; V1).
- Merging over the inherited red.
- Re-asking D-FLEET-13/14.
- Reading or acking `mission-fleet`.
- Resolving `weekly-report:unknown-mission`: attended `08ce76f30` appears to fix it, but it is unverified this fire. Next fire: verify, then resolve with that sha.

**Retro lane**: no skill edit (no gap has reached ≥2 frictions this fire). Observations, one occurrence each:
- (1) `resolve-role-spawn.sh`'s `reroute` arm skips the ration gate, so it pointed the judge at an over-ration bucket.
- (2) `ailang mission rotate-log fleet --stream status` resolves to the registry root (the PIN), not the CWD. It rotated the pin's `fleet-mission-status-archive.md` (18 → 3 entries plus `-old`/`-index` files) instead of moving the charter's stamps, so a record written in a separate worktree cannot use it. The controller backed up and restored the pin byte-for-byte from HEAD (status clean), then moved the stamp by hand with line-count and archive assertions.
- (3) A claude-sub executor can die on a transient API 5xx with an rc-0-looking partial transcript. The rc file (rc=1) and the commit log were the real signal.

**Next**: when dev `CI` is green again: rerun #1604, merge, resolve P1 #7. Then P1 #8 `quorum:zero-signal-guard-vacuous-with-controller-verdict` (premise re-read this fire: `internal/mission/quorum/quorum.go:164` increments `presentCount` for the controller before the zero-signal guard at :176; inside Authority). Then P1 #9 and the Phase 3a spike. D-FLEET-13/14/15 await Mark.

## 23 — 2026-10-06 — P1 #8 verified live at HEAD, but every role lane is over ration (ollama hard-capped mid-fire by this controller's own session) → PARKED-ON-LANE; zero roles spawned, nothing landed; both iteration-22 PRs still blocked [HARNESS]

**Pick**: P1 #8 `quorum:zero-signal-guard-vacuous-with-controller-verdict` (ailang#651). P1 #6 is parked on D-FLEET-15 (iteration 22); P1 #7 is built but its landing is blocked on V1's inherited dev red; P1 #8 was the top routable row of Mark's triage order.
- Gate 0: kill switch armed (`mission-fleet.disabled` absent); gh `sunholo-voight-kampff`; billing tripwire CLEAN; 0 directives on #1584 since `2026-10-06T05:38:39Z` (4 comments, all public or this loop's own); 39 open tickets. Weekly rotation not due: #1584 was created `2026-10-05T12:32:44Z`, after the Monday-07:00-local boundary, and holds 4 comments (<80).
- Gate 1: pin = origin/dev `04dc2b7c8`; local `dev`, the pin and the resolved main checkout are byte-identical to origin on `SKILL.md` + all 12 resources (full drift sweep over both readable copies, per-file verdicts printed). P1 #6's symptom is therefore measured ABSENT this fire — the main checkout authored HEAD itself (attended session committed `8cddc7e2a`→`3557cd22c`→`04dc2b7c8` today); the defect (nothing fast-forwards it) stands, parked on D-FLEET-15. Dev HEAD `04dc2b7c8`: `test` FAILURE on `TestValidateModulePath_SingleFileInsidePackage` (V1's attended lane; handed to V1 at iteration 22), `test-windows` FAILURE, `ailang-core-dev` Cloud Build node-pin failure (known since iteration 21), `docs-build` in flight; full check set `checks=18` (control fired).
- Gate 2: the pick's premise re-verified first-party at `04dc2b7c8` (`quorum.go:146-183`: the controller verdict increments `presentCount` at `:170`, ahead of the zero-signal guard at `:176`). Died-mid-flight sweep: iteration 22's PR #1604 (judged PASS 93, head `2e0f92672`) and record PR #1605 are open and unmerged, blocked by the same dev red — CREDITED here and carried by this record, not redone. No iteration-23 PR/worktree existed before this fire. Grep-the-index: no prior iteration worked P1 #8.

**Did**:
- Gate 3 routing attempt, all four roles measured before any spawn:
  - Resolver: designer `recipe pi:ollama/glm-5.3:cloud declared:provider-pin`; planner `recipe pi:ollama/kimi-k3:cloud over-ration-reroute:codex`; executor `recipe pi:ollama/deepseek-v4.1-flash:cloud declared:provider-pin`; evaluator `refuse over-ration:anthropic` (reason token recorded verbatim).
  - `mission-lane-check.sh fleet`: every rung of every role `skip:over ration` — anthropic ENFORCED (headroom 0.6pp < 1pp start margin; long window 31.0/31.5%), codex (0.4pp < 2pp), openrouter ($3.59 of $2.33/day), ollama (0.8pp < 3pp at check time).
  - This session exposes no Agent/Task tool, so the `agent-tool` spawn path was unavailable as well; the pi lanes require `scripts/mission_pi_run.sh`, which the same ration blocks.
  - Mid-fire the ollama bucket crossed its hard cap: 6.1pp → 10.1pp of the 10pp/day ration, consumed by the controller's own pi-rung session (fractional session gauge 23.7%) reading its rulebook — the only ollama consumer after stapledon's fire ended 18:47:54Z. A pi-rung controller structurally starves the role lanes sharing its bucket.
  - Generator ≠ judge: the REQUIRED evaluator cannot run on any lane, so nothing may land. P1 #8 parked **PARKED-ON-LANE** (standing rule 8b): role = evaluator (and designer/planner/executor), refusals above, resume predicate = "`mission-lane-check.sh fleet` rungs ok + a judge lane ≠ the executor model". It never entered DECISIONS (rule 8c): its resume is a clock, not an ask.
- Landing predicates re-run as commands: dev `test` still red at `04dc2b7c8` → #1604 and #1605 stay parked; no re-run attempted (the red is outside fleet Authority and nothing in either diff changed).
- Bookkeeping only, otherwise: this record carries iteration 22's log entry and STATUS stamp (its PR #1605 is superseded by this PR and closed with a comment), the dashboard overwritten, the index extended, the iteration chain posted (controller stage only, `quota_tokens` not reported — same treatment as iteration 21's controller row; the ollama ration reads the provider's own gauge, so it is not undercounted by that omission).

**Gate 3b**: nothing pushed for landing. The record PR inherits dev's required-`test` red, so it waits with #1604. Resume predicate for both: dev's required `test` green at HEAD (V1's `TestValidateModulePath_SingleFileInsidePackage`). **Late re-measure at end of fire:** dev advanced mid-flight to `c57d4f00e` (attended M-TEST-RUNNER Phase 2 `70e4f77c8` + release v0.52.2), so the predicate was re-run as a command against the newest head, bounded: CI run 37515762996 completed **failure** at 19:18Z, `test` and `test-windows` both failing the SAME `TestValidateModulePath_SingleFileInsidePackage` (known-positive control: 157 `ok`/`--- PASS` lines in the same log). The park therefore stands on that head; #1605 was closed as superseded by #1611 during this fire. A THIRD dev head landed while the routing row was being stamped — `261505a86` = "fix(test): MOD010 single-file layout test fixture outside /tmp", the failing test's own fixture file — and its CI run (37516731812) was still in flight at the bounded wait's deadline (19:36:40Z, ~28 min elapsed), so it is recorded UNMEASURED, neither green nor red; the next fire re-runs the predicate as a command at whatever head it finds.

**Progress**: 0 tickets resolved (39 open).
- Clause 1 UNMEASURED.
- Clause 2 UNMET (0 resolved this fire; oldest open tickets date to 09-26).
- Clause 3 MET (39 open signatures, all in `mission-fleet`).
- Clause 4 prior evidence only (39 open tickets, so the fire proceeded under the driver's exit-at-0 rule).
- Clause 5 preserved trivially: nothing merged, nothing judged, nothing landed unjudged.
**Progress** (digest line): goal unmoved — a capacity park; P1 #8 is verified-routable the moment a lane reopens.

**Routing evidence**: base-gate1=04dc2b7c875f5d165fca2af352d0b0d93c4f09ed@2026-10-06T18:49:20Z. base=261505a86f7951acea7e40a9eb36e8a877daf401@2026-10-06T19:19:00Z (gate4; the record worktree is branched from `8b6354778` = origin/fleet/iter22-record, carrying iteration 22's entry; origin/dev had moved twice mid-fire — `04dc2b7c8` → `c57d4f00e` → `261505a86` = "fix(test): MOD010 single-file layout test fixture outside /tmp", the failing test's own file — by the time this row was stamped). `MISSION_ROUTING_NOTE`: codex over daily ration → planner opus, executor `claude:claude-sonnet-5-5` (both lanes were over ration this fire; the note is the driver's plan, not what ran).
- **Controller**: `pi:ollama/glm-5.3:cloud` (pi fallback rung; tok: not reported; this session took the ollama bucket 6.1pp → 10.1pp/day, session gauge 23.7%).
- **Designer**: NOT SPAWNED — resolver `recipe pi:ollama/glm-5.3:cloud declared:provider-pin`; the ollama bucket is over its hard cap and this session exposes no Agent tool. No fallback link was runnable (declared chain all over ration).
- **Planner**: NOT SPAWNED — resolver `recipe pi:ollama/kimi-k3:cloud over-ration-reroute:codex`; same capacity block.
- **Executor**: NOT SPAWNED — resolver `recipe pi:ollama/deepseek-v4.1-flash:cloud declared:provider-pin`; same capacity block.
- **Evaluator**: NOT SPAWNED — resolver `refuse over-ration:anthropic`, recorded verbatim per the spawn-pattern rule; the declared chain (`pi:openrouter/minimax/minimax-m3` → `claude:claude-sonnet-4-6` → `opus`) is fully over ration. The operator's standing request for this run makes the independent judge REQUIRED, so rather than land anything on the controller's own verdict the iteration parked. No judge verdict exists this fire, by design — that absence is the recorded outcome, not an omission.
- **Metered**: $0.00. **Ration**: ALL FOUR buckets over — anthropic ENFORCED (0.6pp headroom), codex (0.4pp; 2 reset credits in reserve, attended only, next expires 2026-10-22), openrouter ($3.59/$2.33), ollama (10.1pp of 10pp/day, hard-capped).

**Ruled out**:
- Spawning any role on an over-ration lane, or re-prompting in place (the recipe rule: a non-zero verdict is a lane failure → fall back; every fallback link is over ration too).
- Landing #1604/#1605 across the inherited red, or re-running their checks while dev's `test` fails for a reason outside fleet Authority (V1's lane).
- Working P1 #6 without D-FLEET-15's ruling (default B holds: wait for Phase 3b skill pinning).
- Controller-authored implementation of P1 #8 (generator ≠ judge).
- Reading or acking `mission-fleet` (unread = open).
- Re-asking D-FLEET-13/14 (pending, unchanged).

**Retro lane**: none for the skill (nothing reached the ≥2 bar). Observations, one instance each:
- (1) A pi-rung controller consumes the same ollama daily ration its designer/planner/executor lanes need; this fire the controller's own gate-reading took the bucket to its hard cap. If it recurs it is a routing-policy question for Mark (controller rung vs role lanes), with the evidence bar at ≥3 rows.
- (2) This session's read tool elided large outputs to ~2KB (a "quality-monitor" truncation), so the prescribed "read the resource file NOW" had to go through `sed`/`cut` extraction for gate-3's long lines. A harness-instrument friction, recorded here; the fleet cannot file a ticket for its own slot.
- (3) Iteration 22's two unfiled harness notes (resolver reroute ignores the daily quota check; `rotate-log` writes to the pin worktree regardless of CWD) are now in the log via the carried entry; neither bit this fire — the STATUS rotation was done by hand with the line-count assertions, and the archive was verified to hold the moved stamp.

**Next**: when capacity returns — (a) re-probe with `mission-lane-check.sh fleet`; (b) rungs ok → route P1 #8 through designer → planner → executor → evaluator (#651 rules the fix direction; a small design doc plus a one-milestone plan suits it); (c) when dev's `test` is green → land #1604, resolve `gate0:driver-crash-notices-invisible`, and merge the record PRs. Then the Phase 3a skill-resolution spike (after P1, before P2). D-FLEET-13/14/15 await Mark.

## 24 — 2026-10-07 — native role pins absent; planner/evaluator Agent pins rejected → PARKED-ON-LANE, no acceptance [ADMIN]

**Pick**: banked P1 #8 `quorum:zero-signal-guard-vacuous-with-controller-verdict`; this fire cannot route it because the newly open `blocking=all` ticket `agent-tool:mission-role-pins-unavailable` applies first-party. P1 #6 is parked on D-FLEET-15 and P1 #7 is built in #1604, awaiting dev health. Pending record #1611 contains iterations 22–23 and was read before assigning iteration 24.

**Reality check**: verified by controller at origin/dev `261505a86f7951acea7e40a9eb36e8a877daf401`: `internal/mission/quorum/quorum.go` increments `presentCount` for a controller verdict before checking zero signal. No fix made. Both readable skill directories, pin and resolved user symlink, match origin for SKILL.md and each of 12 resource files (26 explicit MATCH readings). Historical main-checkout drift is not a live mismatch this fire; D-FLEET-15 is preserved rather than self-resolved.

**Preflight**: armed, clean pin tree, correct GitHub account, billing guard CLEAN. `mission_directives.sh --issue 1584 --since 2026-10-06T19:02:43Z` returned 0 new allowlisted directives of 6 comments. No messages read or acked in mission-fleet. Three GCP approvals task-d80f4d05/task-98730ece/task-50acfbe6 remain with their operator, untouched. Injected docs sprint is unrelated and was not advanced.

**Outcome**: PARKED-ON-LANE. Zero designs, plans, implementation milestones, judgements or merges this fire. The operator requires native Agent transport and a required independent evaluator. No declared usable independent judge exists here, so controller-only acceptance is forbidden.

**Routing evidence**: base-gate1=261505a86f7951acea7e40a9eb36e8a877daf401@2026-10-07T01:39:40Z; base=261505a86f7951acea7e40a9eb36e8a877daf401@2026-10-07T01:41:09Z (gate4). Record worktree is branched from origin/dev; five mission bookkeeping files from b5d516b85 (#1611) were carried forward after confirming origin changed none of those files since their merge base. Controller native Codex (model/token total not exposed). Resolver commands each returned shell rc 0: routing refusal is encoded in their output, not a nonzero exit. Native spawn failures were tool errors, with no shell rc. `MISSION_ROUTING_NOTE`: absent; not represented as configured. Driver per-role model pins, fallback chains and ration data are absent; quota status UNMEASURED. Metered external role calls $0.00.

| Role | Resolver / Agent result | Fallback actually used | Tokens |
|---|---|---|---|
| Designer | `refuse fail-closed:designer-model-missing` | None. No declared pin, native spawn cannot be selected without inventing a lane. | 0 role-run tokens |
| Planner | derive `opus fail-closed:env-pin`; resolver `agent-tool opus fail-closed:env-pin`; native spawn `model=opus` → `Unknown model opus` | None; required pin unsupported. | 0 role-run tokens |
| Executor | `refuse fail-closed:executor-model-missing` | None. No approved plan or declared executor pin; default session inheritance refused. | 0 role-run tokens |
| Evaluator | `refuse fail-closed:evaluator-model-missing`; native capability probe `model=sonnet` → `Unknown model sonnet` | None. No judgement, independent or controller-authored. | 0 role-run tokens |

Available native enum reported by both failed spawns: gpt-6.1-sol, gpt-6-astra, gpt-6-sol, gpt-6-luna, gpt-5.6-sol. Retired models and dormant Astra were not used. A same-model diagnostic agent is not an authorized independent model judge. No role ran or ended on a background wait.

**Resume predicate**: re-run the four `resolve-role-spawn.sh <role>` commands with the driver-exported environment; require valid declared lanes and a supported native planner plus independently routed evaluator that actually returns a judgement. Alternatively an attended routing ruling may supply supported native pins. The required native Agent transport is retained. No reset time is known. This is a lane/capability park, not a new human decision; never file it as needs-human-review. No fleet self-ticket filed (fleet may not file its own slot). The signature is already open from world/stapledon.

**Gate 3b**: no push of implementation, no merge, no LANDED verdict. Re-measured dev CI run 37516731812 on full SHA 261505a86: failure. Complete check set: 21; Sonar failure, test-windows failure, Build windows-latest failure, Build macos-latest cancelled; all remaining checks success/skipped. Linux test is green. Failed Windows log is 248206 bytes; named FAIL lines and PASS lines validate the reader. Three failures: TestTestCommandBytecodeFlags (different temporary paths in evaluator/bytecode errors), TestNamedBatch_RuntimeErrorPositionsMapToSource (error retains temporary-source positions), TestForall_IllTypedPropertyFailsAlone (compile error retains temporary-source position). These are pre-existing relative to this docs-only run, not attributed to a fleet change. V1 owns cmd/ailang/test* and internal/testing; handed over through mission-v1. #1604 remains parked; no inherited implementation or pending record was merged.

**Progress**: goal unmoved — 0 tickets resolved, 41 open signatures. Clause 1 UNMEASURED; clause 2 UNMET (no turnaround improvement); clause 3 MET (canonical ticket queue); clause 4 prior evidence only; clause 5 no new landing or acceptance credit.

**Verification**: PATH CLI observed newly built at 261505a86 after quick-install; make build returned rc 0 for bin/ailang; no source change. Record checks: ledger valid, 3 STATUS stamps, exact previous stamp archived, old log retained byte-for-byte as a prefix, index includes 22–24, git diff --check. No runtime tests claimed for this ADMIN record. Record is draft only, because the required evaluator cannot be spawned.

**Done-gate**: no harness bytes edited; no dry-run, launchd reload or ticket resolution. No done-gate credit claimed.

**Ruled out**: blindly trusting main's iteration-21 state while #1611 banks 22–23; silently inheriting Codex for all roles; CLI substitution against the operator's native Agent instruction; taking a controller-only verdict as an evaluator; resolving D-FLEET-15 from a current match; working the injected docs sprint; resolving pending GCP approvals.

**Retro lane**: backlog / routing-policy signal. World 239 and Stapledon 18 already reported native pins unavailable; fleet measured the same missing environment and unsupported opus/sonnet pins. No routing or skill changes made; a policy change cannot be self-approved. This is capacity, so it adds no DECISIONS row. Last-20 fleet harness share: 16/20 (80%), rows 21,20,19,18,17,16,15,14,12,11,10,9,8,7,6,5; the other four index rows carry their recorded classes. It exceeds one third under the fleet charter's explicit harness mandate, not a product-loop violation. This is reported separately from product clause 1; the fleet's harness mandate makes a high share expected, not a product-loop authority breach. Last three landings (iteration 19 heartbeat and rotate-log, iteration 21 pi predirty) address unresolved harness tickets / turnaround work, though the turnaround bar remains UNMET; no all-none drift verdict.

**Next**: restore admitted role routing → P1 #8; dev Windows green → re-run #1604 at its pinned head, require remote CI plus independent judgement before landing. Then P1 #9 exit-path notices; Phase 3a measurement after P1. D-FLEET-13/14/15 remain OPEN, unchanged. Draft record retains prior pending history; #1611 is not closed until the consolidated record is reviewed.

## 25 — 2026-10-07 — `blocking=all` codex-controller role-env ticket LANDED: #1635 `0ceb1db01`, judged PASS 93; ticket resolved [HARNESS]

**Pick**: `agent-tool:mission-role-pins-unavailable` (blocking=all, World 239/240 + fleet 24). The charter ranks new tickets below Mark's list unless `blocking=all`, so it outranks P1 #6 (parked on D-FLEET-15), P1 #7 (#1604, built) and P1 #8 (parked by iteration 23). Pending record PRs #1611 (22–23) and #1612 (24) were read before numbering this iteration; this record branches from #1612's head and supersedes both.

**Reality check**: controller, first-party. Rig `~/.codex/config.toml` has `shell_environment_policy.inherit = "core"` (mtime 2026-10-06 13:02). `codex sandbox -- /usr/bin/env` (names only) shows `HOME`/`PATH` but no `MISSION_*` and no test var; `-c shell_environment_policy.inherit=all` shows both. The World driver log at 2026-10-07 01:21:49 shows the driver DID resolve all four roles, so the defect is delivery, not resolution. Option measurements: `include_only` with `AILANG_*` admits `AILANG_REGISTRY_API_KEY` and strips the rig's `.set` vars; `inherit=core` + `include_only` adds nothing.

**Preflight**: armed (`mission-fleet.disabled` absent), gh `sunholo-voight-kampff`, billing CLEAN, `mission_directives.sh --issue 1584 --since 2026-10-06T19:02:43Z` → 0 directives of 9 comments. All 13 running-skill files `cmp`-identical to origin/dev (resolved symlink). Local pin == origin/dev `fcce2394c`. Dev check set at `fcce2394c`: `test-windows`/`Build windows-latest` failure (V1's lane, same tests as iteration 24), Linux `test` pending then green.

**Outcome**: LANDED. Helper `tools/launchd/lib/codex-env-args.sh` forwards each exported var on a name allowlist (`MISSION_*`, `AILANG_DRIVER_*`, `AILANG_STORAGE_MESSAGING`, `AILANG_MESSAGES_PROJECT`, `AILANG_MISSION_REGISTRY`, `AILANG_STATE_DIR`, `CONTROLLER_*`) minus a secret-name denylist, plus only the `core.hooksPath` git-config entry renumbered to 0, as `-c shell_environment_policy.set.NAME="<TOML basic string>"`. The codex branch of `_mc_run_once` expands it with the bash-3.2 `set -u`-safe `${arr[@]+…}` form; a names-only summary line goes to the driver log. Ticket resolved with `--sha 0ceb1db01`.

**Routing evidence**: base-gate1=fcce2394c2cb2eaec64081ef89de1994a5502a7e@2026-10-07T19:47:10Z; base-gate3b=0ceb1db01df661c3608f0c3520a6eb28bcd6bab2@2026-10-07T20:49:24Z (drift = this iteration's own merge, 1 commit). Controller `claude:claude-opus-5-5` (driver: codex over daily ration, probe ok → opus) (tok: not reported) · designer: not spawned (no doc; one call site, trade-off measured by controller) · planner `opus` Agent (resolver `agent-tool opus fail-closed:env-pin`; 93,399 tok) · executor `claude:claude-sonnet-5-5` via `claude-sub`, probe rc 0 (tok: not reported, `claude -p` text mode) · evaluator: resolver `reroute pi:openrouter/minimax/minimax-m3 generator-equals-judge`; openrouter in `MISSION_OVER_RATION`, so the next declared rung `claude:claude-sonnet-4-6` via `claude-sub`, probe rc 0, own detached worktree (tok: not reported). **FLAG**: judge is the same vendor and family as the executor, a different model (iteration 22 chose opus over this rung for that reason). No role ended on a background wait; every wait was a bounded `date +%s` poll on an rc/marker file.

**Gate 3b**: PR #1635 required `test`/`lint`/`build`/`docs-gate`/`UI build gate` all SUCCESS on `36f386a61`. `launchd drivers (bash 3.2)` pass, govulncheck pass. Linux `test` first attempt: `go: … proxy.golang.org … stream error: stream ID 1125; INTERNAL_ERROR` at module download (log 90,107 B, 0 `--- PASS` lines, i.e. no tests ran), so it was re-run → SUCCESS. `test-windows`: `TestTestCommandBytecodeFlags`, `TestMCPFile_ListedGolden`, `TestMCPUI_Golden`, `TestNamedBatch_RuntimeErrorPositionsMapToSource`, `TestForall_IllTypedPropertyFailsAlone`, the same signature as dev HEAD (non-required, V1's lane). `Build macos-latest` ended `The operation was canceled` (fail-fast casualty). Merged `--match-head-commit 36f386a61…` → `0ceb1db01`. Dev CI run 37684928778 on `0ceb1db01`: run 37684928778 concluded `failure` ONLY on the non-required `test-windows`/`Build windows-latest`, with the identical five-test set as parent `fcce2394c` (inherited, V1); required Linux `test`, `lint`, `launchd drivers (bash 3.2)`, govulncheck and the `ailang-core-dev` Cloud Build are all success, so this counts as the infrastructure green.

**Progress**: 1 ticket resolved (filed 2026-10-06T23:23Z → resolved 2026-10-07 ≈21h, inside clause 2's 48h). Clause 1 UNMEASURED; clause 2 UNMET (median still over 48h); clause 3 MET; clause 4 prior evidence only; clause 5 preserved.

**Verification**: executor suite 18/18 + `make test-launchd-drivers` MAKE_RC=0 + shellcheck (helper/test 0; driver 28→29, one SC1091 for the new `.` line, same class as the existing `lane-probe.sh` source) + `check-changelog` rc 0; executor mutations (append dropped 6 red, call-site expansion removed 4 red, `*TOKEN*` dropped 2 red, hooksPath forward off 1 red). Judge: re-ran suite and `make test-launchd-drivers`, own mutations J1 denylist removed → `secret-names-denied`, J2 control-char guard removed → `toml-encode-unit`, J3 scope-guard block removed → `argv-scope-guard-forwarded`. Controller live: helper forwarded=49 in this mission shell, and `codex sandbox <args> -- resolve-role-spawn.sh` matched the native shell's four lines exactly; the control reproduced `refuse fail-closed:{designer,executor,evaluator}-model-missing`.

**Done-gate**: surface ✓ (only codex-controller call site); reach ✓ (origin/dev); `make test-launchd-drivers` ✓ (executor, judge, CI); dry-run not run, because `MISSION_DRY_RUN=1` exits at driver :1926, before role-env export (:2180) and the call site (:2274), as the plan measured; the e2e arms and the live check cover it; nothing reloaded ✓. First live proof will be the next codex-controller fire's `codex-env: forwarded=N` log line.

**Ruled out**: `inherit=all` (exposes every key); `include_only` (leaks `AILANG_REGISTRY_API_KEY` via `AILANG_*`, strips rig `.set`); editing `~/.codex/config.toml` (rig config, outside Authority, and anything else that rewrites it would bring the defect back); a design doc (a single call site).

**Retro lane**: backlog. Two non-blocking judge notes: `mc_toml_basic_string $'\x00'` returns `""` rc 0 (unreachable, since env cannot hold NUL); forwarded values are visible in `ps` (denylist keeps secrets out). Follow-up candidate for an attended ticket or decision: under a codex controller, a bare-alias evaluator pin (`sonnet`/`opus`) still resolves to `agent-tool <alias>`, which codex `spawn_agent` rejects (planner measured 1 of 26 codex-controller fires). That is the other half of World's evidence and may be routing policy, so it was not built. No skill edit (no second friction on one gap).

**Next**: resume P1 #7, #1604 (dev Linux `test` green now): re-run its failed jobs on `2e0f92672`, require the required checks green, merge `--match-head-commit`, resolve `gate0:driver-crash-notices-invisible`. Then P1 #8 if the lanes admit it. D-FLEET-13/14/15 remain OPEN.

## 26 — 2026-10-08 — gate0 self-notice read LANDED: #1604 `59c3e6a55`, re-judged PASS 96 on the merged head; ticket resolved; record #1636 landed [HARNESS]

**Pick**: P1 #7 `gate0:driver-crash-notices-invisible`, iteration 25's named resume. No `blocking=all` ticket open (43 open; `ailang mission ticket open --json`). Its predicate was re-measured as a command, not transcribed: dev required `test` green at `0ceb1db01`.

**Preflight**: armed (`mission-fleet.disabled` absent); gh `sunholo-voight-kampff`; billing CLEAN; `mission_directives.sh --issue 1584 --since 2026-10-06T19:02:43Z` → 0 directives of 10 comments. Not a sweep week (iteration 19 swept), and no rotation: #1584 was created after Monday 07:00 local. Running skill: `SKILL.md` and all 12 resources `cmp`-identical to origin/dev at the resolved symlink. Pin == origin/dev `0ceb1db01`.

**Observe**: record PR #1636 (iterations 22–25) open and UNSTABLE: required checks green, Windows red inherited. Dev check set at `0ceb1db01`: 17 checks; NOT-GREEN = Sonar, `test-windows`, `Build windows-latest` (5 tests), and `Build macos-latest` cancelled. `mission-lane-check.sh fleet`: READY, every role has a usable rung (ollama rungs skipped as over ration; openrouter $0.10 of $2.33).

**Did**:
- Landed #1636 → `752ee765a` (closing-keyword scan empty, `--match-head-commit 3a94d3df8`), and closed superseded #1611 and #1612 with a comment.
- #1604: its `test` red on `2e0f92672` = inherited `TestValidateModulePath_SingleFileInsidePackage` (job 112275244977; 132,286 B log; one `--- FAIL`). 46 dev commits since its base `e68a264fb`. The only overlap with PR files is `make/test.mk`, where each side added a different line; `git merge-tree` clean. `gh pr update-branch` → `f4de90745`.
- Re-judged, because the bytes under review changed (a merge). Evaluator below. PASS 96, 0 blocking. Report banked at `design_docs/fleet-mission-evidence/iteration26/judge-report.md`.
- Controller reproduction, first-party, outside the sandbox, in the judge's worktree: suite `108 passed, 0 failed` rc 0. Mutant `: emit_for "$PREV_ISSUE"` → `103 passed, 5 failed` rc 1. Restore `cmp`-identical to `HEAD`; worktree porcelain = the report only.
- PR CI on `f4de90745`: `test` SUCCESS (the job that was red), `lint`, `docs-gate`, `UI build gate`, `launchd drivers (bash 3.2)`, govulncheck, CodeQL green. `test-windows` failed tests `diff`-identical to dev `0ceb1db01`'s (5; logs 527,124 B vs 528,078 B). Squash-merged `--match-head-commit f4de90745` → `59c3e6a55`.
- Resolved `gate0:driver-crash-notices-invisible` (`--sha 59c3e6a55`, "replied to v1"). Commented the verdict on #1160 via `--body-file` (comments 1 → 2, verified), then closed it.

**Gate 3b**: run 37724543642 on `59c3e6a55` (full SHA, via `mission-base.sh record gate3b`; drift from Gate 1 = #1636 + #1604, this iteration's own merges). `test` success, `lint` success, `build` success, `launchd drivers (bash 3.2)` success, govulncheck and float determinism success. `test-windows` failures `diff`-identical to the parent's 5 (528,733 B log). Sonar failure, also on parent `0ceb1db01`. Required set green → LANDED.

**Done-gate**: surface ✓ (`scripts/mission_gate0_self_notices.sh`, which every mission's Gate 0 runs via `MISSION_DRIVER_ROOT`, and both `gate-0-preflight.md` copies). Reach ✓ on origin/dev. **Running-skill reach PENDING**: `~/.claude/skills/mission-control` resolves to `~/dev/sunholo-data/ailang` (branch `dev`, `0 36` ahead/behind), where `grep -c "6a. SECOND, NO-AUTHORITY READ"` = 0, against 1 on origin. So no loop reads step 6a until that checkout is fast-forwarded (D-FLEET-15; not reconciled here, because Gate 1 reserves the standing authorisation to Mark). `make test-launchd-drivers` ✓ (CI leg on the merge SHA; the judge's in-sandbox rc 2 is the `mktemp` denial in the untouched `test_mission_lane_check.sh`). 0 driver bytes, so no dry-run. Nothing reloaded.

**Progress**: 1 ticket resolved (filed 2026-09-26 → resolved 2026-10-08, ≈12 d). Clause 1 UNMEASURED; clause 2 UNMET; clause 3 MET (42 open); clause 4 prior evidence only; clause 5 preserved.

**Routing evidence**: base-gate1=0ceb1db01df661c3608f0c3520a6eb28bcd6bab2@2026-10-08T03:20:14Z; base-gate3b=59c3e6a55bc58bb7202c0e9741fe46fc52547c61@2026-10-08T03:49:39Z; base=59c3e6a55bc58bb7202c0e9741fe46fc52547c61@2026-10-08T04:17:41Z (gate4). `MISSION_ROUTING_NOTE`: empty (as configured).
- **Controller**: `claude:claude-opus-5-5` (tok: not reported).
- **Designer / Planner / Executor**: NOT SPAWNED. This was a resume of work iteration 22 had already planned, built (executor `claude:claude-sonnet-5-5`) and judged. The operator's standing request to use the Agent tool for roles applies where a role runs; the evaluator's lane is a `recipe`, so the Agent tool was not its specified path.
- **Evaluator**: resolver `agent-tool sonnet declared:alias-pin`. `sonnet` is the executor's family (`claude-sonnet-5-5`), so the generator ≠ judge guard re-routed to the first declared fallback `pi:openrouter/minimax/minimax-m3`. Probe rc 0 (`ok`); openrouter within ration. Ran via `scripts/mission_pi_run.sh --max-seconds 3600` in `fleet-iter26-evaluator` (detached at `f4de90745`). Handshake acked (4× `"acked":true`). Verdict `ok` rc 0, 1190 s, 93 tool executions, 94 turns. 4,912,033 tok (407,371 in / 41,686 out / 4,462,976 cache-read), $0.4400 (provider-reported `usage.cost`). Judge ≠ generator: minimax vs sonnet-5-5 (executor) and opus (controller).
- **Metered**: $0.44. **Ration**: ollama over; codex 3.0% of 7.3%; Anthropic 5h 9%, week 51% of 58.7%; openrouter $0.10 of $2.33 before the judge.

**Ruled out**:
- Re-running the failed jobs on the old head `2e0f92672`, as iteration 22's predicate wrote. Its base predates dev's `test` fix, so it would have re-run the inherited red; updating the branch was the cheaper, correct form.
- Merging on iteration 22's PASS 93 alone: the merged bytes are new, and the operator requires a judge each landing.
- Fast-forwarding the main checkout to make step 6a live: that is D-FLEET-15, still OPEN.
- Fixing the Windows five or Sonar: V1's lane, outside Authority.

**Retro lane**: no skill edit (no gap reached ≥2 frictions this fire). Observations, one each:
- (1) A resume predicate written as "re-run failed jobs on head X" goes stale when dev moves under an unmerged PR. The robust form is "update the branch, then require green on the new head".
- (2) The landed rule text is not live for any loop until the shared main checkout syncs. This is the first measured instance of a fleet fix landing on origin and not reaching the running skill; it is evidence for D-FLEET-15, not a new ask.

**Next**: P1 #8 `quorum:zero-signal-guard-vacuous-with-controller-verdict` (ailang#651). Its PARKED-ON-LANE predicate held at this fire's Gate 3 (`mission-lane-check.sh fleet` READY; minimax judge lane ≠ executor), so re-measure it and take it. Then P1 #9 `driver:exit-path-notices-unbounded`. D-FLEET-13/14/15 remain OPEN.

## 27 — 2026-10-08 — main-checkout ff-only auto-sync LANDED: #1647 `15d47b5dc` (D-FLEET-15 = A), judged PASS 91; ticket resolved [HARNESS]

**Pick**: P1 #6 `skill-surface:main-checkout-not-synced-to-dev`. Mark's attended ruling D-FLEET-15 = A (2026-10-08) unparked it ("treat it as HD-2a-authorised; plan it as the next P1"), so it ranks above P1 #8. No `blocking=all` ticket was open (42 open).

**Preflight**: armed (`mission-fleet.disabled` absent); gh `sunholo-voight-kampff`; billing CLEAN; `mission_directives.sh --issue 1584 --since 2026-10-06T19:02:43Z` → 0 directives of 11 comments. Ledger valid (15 rows, 0 OPEN). Self-notices (step 6a) rc 1: the newest crash is the 2026-10-04 rc=143 kill, which iteration 22 already attributed to iteration 18's Gate-5 stall. Died-mid-flight traces: no open fleet PR, and the old `fleet-i1x` worktrees are all landed iterations. Running skill: `SKILL.md` and all 12 resources `cmp`-identical to origin/dev at the resolved symlink. Not a sweep or rotation week (#1584 was created after Monday 07:00 local).

**Observe**: origin/dev `c92739681`. Required `test` RED on `TestModels_CloudHeadroomEqualised` (job 113225065559): the motoko GLM-5.3-Flash rows were capped at 32000 by `69ddedd29`, in modelreg. Parent `f34214657` was green. Fleet does not own this red (Gate 1), so it went to `mission-v1` and `mission-motoko` with the repro. `382a8445d` (#1645) fixed it 25 minutes later. `test-windows` red as before (V1's lane).

**Did**:
- The planner (codex) wrote `design_docs/planned/sprint-plan-main-checkout-ff-sync.md` and the sprint JSON (`64a1bcab5`). Its premises section quotes the driver's file:line for every seam.
- The executor (codex) delivered M1 `d852c8969`: `tools/launchd/lib/skill-sync.sh` (`mc_skill_sync apply|report`) and `tools/launchd/test_skill_sync.sh`.
  - The helper uses bounded git calls through `_pin_bounded`, with a 5 s cap per call.
  - It merges only when the checkout is on `dev`, 0 ahead, has no op markers and no `index.lock`, and no dirty path intersects the incoming range (NUL-safe, both rename sides, directory prefixes).
  - It re-checks just before the one `merge --ff-only` and verifies HEAD and the count afterwards.
  - It always returns 0.
- The executor delivered M2 `4fe2da1f8`: a 10-line driver seam after the kill switch (:1625) and pidfile yield (:1908). Dry-run calls `report` and appends `skill-sync=` to the DRY RUN line; a real fire calls `apply` once before the boot stagger. Also: a `make/test.mk` line and a changelog fragment.
- The controller rebuilt both commits from the executor's `.snap/M1` and `.snap/M2`. `shasum -c` matched its final tree, and the suite was green at both boundaries.
- PR #1647. Every required check green on `4fe2da1f8`; the run included #1645, so `test` was green. Squash-merged `--match-head-commit` → `15d47b5dc`. Ticket resolved, with a reply to the attended filer.

**Gate 3b**: CI on `15d47b5dc` (full SHA; `mission-base.sh record gate3b` → `c114a1299`, i.e. dev moved past it with others' merges): `test`, `lint`, `build`, `launchd drivers (bash 3.2)` and govulncheck all success, out of 17 checks. NOT-GREEN = `test-windows` and `Build windows-latest` (inherited, as on parent `c92739681` and the PR head) plus a fail-fast `Build macos-latest` cancel.

**Done-gate**:
- Surface ✓.
- Reach ✓ on origin/dev.
- Dry-run ✓ healthy (`lanes=ok | pin=pinned | skill-sync=synced:4`) and ✓ degraded (`MISSION_EXECUTOR_MODEL=codex:bogus` → `lanes=DEGRADED(1)`). Both ran with `AILANG_DRIVER_PINNED=4fe2da1f8`, `MISSION_PROFILE=fleet`, `HOME=/tmp/fleet_it27_dryhome`: real config symlinked in, `.ailang/state` empty. That is how a fleet fire dry-runs its own profile past its own overlap guard. World and stapledon were mid-fire, and v1, docs and motoko were disabled, so no sibling was available.
- The real main checkout's HEAD and `.git/index` sha were identical before and after both dry-runs and both suite runs.
- `make test-launchd-drivers` ✓ (rc 0 unpiped; also the CI leg).
- Nothing reloaded ✓.

**Progress**: 1 ticket resolved (filed 2026-09-29 → 2026-10-08, ≈9 d, about 2 d of it parked on D-FLEET-15). Clause 1 UNMEASURED; clause 2 UNMET; clause 3 MET (41 open); clause 4 prior evidence only; clause 5 preserved.

**Routing evidence**: base-gate1=c92739681f50d33cf575a4e3a81b76aef360ea71@2026-10-08T08:55:31Z; base-gate3b=c114a1299bde5bbbad6d5f7c972afa3ecb6401b0@2026-10-08T11:01:39Z (drift: #1645, #1639 and this iteration's #1647). `MISSION_ROUTING_NOTE`: empty (as configured).
- **Controller**: `claude:claude-opus-5-5` (tok: not reported).
- **Designer**: not spawned. Resolver said `recipe claude:claude-opus-5-5 declared:provider-pin`, but there was no doc to write: one helper plus a seam inside an attended ruling's exact scope (the same basis as iteration 25).
- **Planner**: resolver `agent-tool opus fail-closed:no-doc`, while `MISSION_PLANNER_MODEL=codex:gpt-6.1-sol`. Per role-spawn-routing rule (a), the pin was followed under the codex recipe (probe rc 0), in a detached worktree `fleet-iter27-planner`. (93,606 tok)
- **Executor**: `codex:gpt-6.1-sol` (`recipe … declared:provider-pin`) in `fleet-iter27-exec`, rc 0 in 17 min. (126,669 tok)
- **Evaluator**: `agent-tool sonnet declared:alias-pin` → Agent tool `model=sonnet`, foreground, own detached worktree at `4fe2da1f8`. judge-independence: cross-vendor (Anthropic judge, OpenAI generator). PASS 91. Report banked at `design_docs/fleet-mission-evidence/iteration27/judge-report.md`. (57,043 tok)
- The operator's standing request to use the Agent tool was honoured for the one role whose resolved path is the Agent tool (the evaluator). Planner and executor resolve to `recipe codex:…`, a path the routing table says is NOT the Agent tool; the spawn-pin hook would deny an alias there. No role failed to spawn.
- **Metered**: $0 this fire (codex and Anthropic subscription buckets; no openrouter or pi role).

**Ruled out**:
- Opus planner via the Agent tool, the resolver's literal answer: the role carries a provider pin, so the hook would deny it (rule (a)/(c)).
- Waiting for an idle sibling to dry-run: none was idle, and the synthetic-HOME dry-run of the fleet's own profile exercises the same patched driver.
- Fixing `TestModels_CloudHeadroomEqualised`: modelreg is outside fleet scope; it was handed off, and the owner fixed it in #1645.
- Raising the 5 s merge cap now: measured 0.66 s for an 804-file ff and 0.05 s for status on the real tree. Kept as a judge follow-up.

**Retro lane**: process fix, charter Guardrails: the done-gate now names the synthetic-HOME recipe for when no sibling is idle. Frictions: iterations 9 and 15 recorded "the fleet cannot dry-run itself", and this iteration measured a way around it. No skill edit. Observations, one each:
- (1) `tools/launchd/mission-lane-check.sh fleet` did not return within 10 min and was killed by the harness cap. The resolver lines in the same command had already printed, so nothing was lost; this is the first measured hang. Not ticketed: fleet does not self-source.
- (2) The controller's own zsh loops twice failed to word-split `$g`/`$FILES`, producing rc 127. These were instrument failures that were re-run under `/bin/bash`, not verdicts.
- (3) Judge follow-ups for a later ticket: an arm pinning `GIT_OPTIONAL_LOCKS=0`; the dry-run field prints `synced:N` for a would-sync verdict.

**Next**: P1 #8 `quorum:zero-signal-guard-vacuous-with-controller-verdict` (ailang#651); re-measure its premise at HEAD. Then P1 #9 `driver:exit-path-notices-unbounded`. Also check the first post-merge fire's driver log for `skill-sync=synced:` (running-skill reach).
