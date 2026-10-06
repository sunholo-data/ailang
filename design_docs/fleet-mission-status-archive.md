# Fleet Mission — STATUS archive

Older STATUS stamps rotated out of [fleet-mission.md](fleet-mission.md) (newest 3 stay there). Append-only.

## STATUS 2026-10-03 — ITERATION 16: heartbeat fix BUILT + judged PASS 96, push-blocked by the pinned scope guard; guard arm LANDED #1575 `adab9b7d9`; ticket stays open, pushes next fire

D-FLEET-10 = A executed. Opus designer wrote Revision 5 (`case` absoluteness guard on `MISSION_DRIVER_ROOT`, all Kimi r5 residuals); quorum r6 BLOCKED on premise rows → one designer revision (R5.1); r7 BLOCKED on evidence only → controller applied the reviewers' fixes under the narrow-refinement carve-out (R5.2, incl. an end-to-end `MISSION_DRIVER_ROOT` measurement in this live driver-spawned controller shell). gpt6-1-sol absent both rounds (OpenAI API 429, no credits). Opus planner → `claude:claude-sonnet-5-5` executor (claude-sub; Agent alias denied by provider pin) → independent Opus Agent judge PASS 96/100 (evaluator chain walked: minimax skipped over-ration, sonnet-4-6 skipped same-family). Push of the 18-site fix was REFUSED by the pinned pre-push guard, which lacks the `.agents` arm the fix itself adds (core.hooksPath = pin worktree). Not bypassed: the guard arm + Authority line shipped alone as #1575 (23/23 green). Judged branch `fleet/i16-heartbeat-rev5` (rebased, judged bytes unchanged) pushes on the next fire once the pin carries `adab9b7d9`. Clause map: **1 product share** UNMEASURED; **2 turnaround** UNMET (0 tickets resolved this fire; heartbeat ticket filed 2026-09-26); **3 one queue** MET (41 open signatures, `ailang mission ticket open --count` at Gate 5); **4 idle is free** prior evidence only; **5 no regressions** preserved (nothing unjudged shipped; #1575 is a one-arm guard change, judged within the PASS-96 diff).

## STATUS 2026-10-03 — ITERATION 15: controller HTTP 402 classified as capacity — LANDED #1549 `27dab5bb4`, evaluator PASS 95; ticket resolved

D-FLEET-11 = A (attended) executed. Attempt 1 (21:39Z fire) revised the design (Rev 2), ran quorum r1/r2 (both BLOCKED on concrete, direction-preserving fixes), applied r2 verbatim as Rev 3 under the narrow-refinement carve-out, then died at gate-3 on an Anthropic "hit your session limit" (a capacity stop the driver also mis-records as CRASHED — new finding, not ticketable by the fleet; see log). Attempt 2 recovered the unpushed branch: opus planner → `claude:claude-sonnet-5-5` executor (claude-sub; Agent alias denied by provider pin) → independent `pi:openrouter/minimax/minimax-m3` judge PASS 95/100. Driver +7/−2: anchored `^402:` joins `RUNTIME_QUOTA_SIG`; pause-aware final branch suppresses the generic crash notice. Hermetic 14-case suite + chain-suite seams; 6/6 mutants red. Done-gate: `make test-launchd-drivers` rc0; healthy+degraded dry-runs `DRY RUN ok` on `eb39db9a2` (world profile — fleet's overlap guard yields during its own iteration). PR head 23/23 green. D-FLEET-10/11/12 status cells normalized to RESOLVED (answers unchanged). Clause map: **1 product share** UNMEASURED; **2 turnaround** UNMET (this ticket filed 10-02 05:52Z → resolved 10-03, ≈24h, first resolution since iteration 10); **3 one queue** MET (35 open signatures before this resolve); **4 idle is free** prior evidence only; **5 no regressions** preserved (done-gate passed, judged).

## STATUS 2026-09-26 — ITERATION 0: **charter RATIFIED as written** (Mark, attended)

Mark: *"yes as written"* — the bar (clauses 1–5), Authority and Guardrails below stand unchanged.
Kill switch lifted the same session. The loop idles at zero cost (driver pre-check) until the first
ticket is filed to `mission-fleet`. Plane: `mission-fleet` is declared triage in prod
(ailang-multivac d2f277d, promoted 2026-09-26).

## STATUS 2026-09-26 — ITERATION 1: **P0 #1 LANDED**, `driver:slot-kill-leaves-orphan-descendants` ([#1325](https://github.com/sunholo-data/ailang/pull/1325), `e3dadcd07`)

Both watchdogs now reap the controller's whole process tree through `_mc_kill_tree`, which
snapshots before TERM, re-walks before KILL and skips recycled PIDs. A triggered watchdog is no
longer cancelled mid-grace. Evaluator (sonnet) PASS 87, 0 blocking. Ticket resolved. Thresholds
are untouched. Clause map: **1 product share** unmeasured (no product-loop window since the
charter); **2 turnaround** first datum ≈3h (filed 14:50Z); **3 one queue** MET (16 tickets, all in
`mission-fleet`); **4 idle is free** MET by construction (driver pre-check, iteration 0 dry run);
**5 no regressions** MET for this ticket, with one caveat: the done-gate's dry-run line cannot
reach `_mc_run_once` edits (UNINFORMATIVE, see log). Next: P0 #2, mechanical half.

## STATUS 2026-09-27 — ITERATION 2: P0 #2 **PARKED (policy, D-FLEET-4)**; P0 #3 commit-blind half **built, evaluated PASS 97, merge blocked on the inherited dev red** ([#1329](https://github.com/sunholo-data/ailang/pull/1329))

**P0 #2** `stall-watchdog:kills-controller-on-long-drill`: the codex planner evaluated five progress
arms against the 09-26 04:45 World kill and the two reference wedges. None works without a new
numeric threshold, so it is policy (HD-2a). Decision row D-FLEET-4 carries the recommendation. Plan:
`design_docs/planned/sprint-plan-stall-descendant-progress.md`. **P0 #3**
`pi-runner:verdict-blind-to-commits-and-predirty`, **commit-blind half**: `mission_pi_run.sh` now
counts commits since launch as work, and a moved HEAD alone is not work. Evaluator (sonnet): round 1
FAIL 68 (HEAD_MOVED false greens), round 2 **PASS 97, 0 blocking**. `make test-launchd-drivers`
rc=0. **Not merged:** the required `test` check is red on dev at base (`internal/iface/builder.go`
801 > 800 lines since `79af650f7`, language core, handed to V1). Resume predicate: dev `test` green
→ re-run #1329 CI → squash-merge. The ticket stays open after the merge, because its pre-dirty
half (the D-58 design) remains. Clause map: **1 product share** unmeasured; **2
turnaround** at risk (P0 #2 now waits on Mark, P0 #3 waits on V1); **3 one queue** MET (16 tickets,
one new: `pi-runner:quota-429-reported-as-empty-worktree`); **4 idle is free** MET; **5 no regressions**
MET (nothing landed unverified).

## STATUS 2026-09-27 — ITERATION 3: P0 #4 **PARKED (D-FLEET-6: fix needs `tools/pi-extensions/**`)**, plan committed; P1 #0 driver-env scrub **built, evaluated PASS 97, merge blocked on the inherited dev red** ([#1330](https://github.com/sunholo-data/ailang/pull/1330))

**P0 #4** `pi-runner:sandbox-extensions-not-wired`: REAL at HEAD (`scripts/mission_pi_run.sh:165` passes no
`-e`). The codex planner verified P1–P5 and found more: `tools/pi-extensions/sandbox/index.ts` **fails
open**. When `SandboxManager.initialize` throws, bash runs unsandboxed (`:227`, `:290`, controller- and
evaluator-verified). Other findings: the rig's policy sits at a path pi 0.85.1 never reads, sandbox-runtime
resolves only from the main checkout, and linked-worktree commits would be fenced. So M1 must edit
`tools/pi-extensions/**`, which is outside the allowlist, and D-FLEET-3 already reserved this ticket for Mark.
Plan: `design_docs/planned/sprint-plan-pi-runner-sandbox-wiring.md`; decision **D-FLEET-6**. **P1 #0**
`tests:driver-env-leaks-into-launchd-suite` (Mark's directive): every `test-launchd-drivers` suite now runs
through the allowlisted `tools/launchd/lib/suite-env.sh`. In the live fire env, base rc=2 (routing 86/1)
and fix rc=0 (87/0). Evaluator (sonnet) **PASS 97, 0 blocking**; the CI `launchd drivers (bash 3.2)` job
is green. **Not merged:** `test` is red only on the inherited `builder.go` 801 > 800, and `test-windows`
is the inherited Go-suite red. Resume predicate, shared with #1329: dev `test` green → re-run CI →
squash-merge both. Clause map: **1 product share** unmeasured; **2 turnaround** at risk (three items wait:
P0 #2 and P0 #4 on Mark, #1329 and #1330 on V1's red); **3 one queue** MET (16 open tickets, none new);
**4 idle is free** MET; **5 no regressions** MET (nothing landed unverified).

## STATUS 2026-09-27 — ITERATION 4: P0 #2 M1 **measured**. On the rig, `ps -S` cannot see reaped-child CPU; `proc_pid_rusage` separates the drill from both wedges. Threshold returns to Mark as **D-FLEET-7**. Evaluator PASS 87.

#1329 and #1330 are **merged** (`c912320fe`, `4d8dff929`), so P0 #3's commit-blind half and P1 #0 have
landed. At Gate 1, dev `CI` was red on one `launchd drivers (bash 3.2)` timing assertion
(`test_driver_notify.sh`, "hanging gh comment", elapsed 8 s against a 7 s limit). A rerun was green:
that is the `ci:launchd-driver-suite-flakes` class (P2 #11), now reproduced once on dev.
**P0 #2 M1** (D-FLEET-4): a new `tools/launchd/measure_stall_cpu.sh` plus a `proc_rusage.py` helper.
Both are measurement only; the live watchdog is unchanged. **Finding:** on macOS 26.6.2, `ps -S` reads
`0:00.00` for a parent whose reaped child burned about 4 CPU-s, so the plan's candidate instrument
cannot work. Per ~121 s window, rusage (self plus reaped children) reads: drill **62.5–97.7**, w1-git
**0.73–1.72**, w1-gh **0.22–0.31**, w2 **0**. Idle `claude` roots accrue **0.57–1.60** in the same
window, so an arm must exclude the root. Recommendation in D-FLEET-7: 10 CPU-s per window over the
descendants only. Branch `fleet/iter4-stall-cpu-measure`; PR and merge at Gate 3b. Clause map: **1 product share**
unmeasured; **2 turnaround** at risk (P0 #2 back on Mark; P0 #4 routable); **3 one queue** MET (17
open, one new: `rotate-log:status-flag-mutates-and-world-resolves-to-status-archive`); **4 idle is free**
MET; **5 no regressions** MET (nothing live changed).

## STATUS 2026-09-28 — ITERATION 6: iteration 5's two fixes **re-judged (PASS 92, PASS 96) and LANDED in [#1377](https://github.com/sunholo-data/ailang/pull/1377) (`eec86ca4f`)**; both tickets resolved

Iteration 5 (07:56 fire) built both fixes and crashed at Gate 3b on an API DNS error (`ENOTFOUND`,
slot verdict `CRASHED_at=gate-3b`) with no record. The 23:09 fire before it died in the Aqua-session
loss that its own pick is about. This iteration resumed rather than redid that work. **(1)** The
`blocking=all` ticket `rig:aqua-session-lost:windowserver-watchdog`: `cron-kicker.sh` now checks the
gui domain first, logs `SESSION-LOST` once and sends one bounded, fail-soft notice. On restore it
re-stamps state, and it derives mission labels from `missions/*.toml`, so fleet and stapledon are
covered. **(2)** P0 #4 `pi-runner:sandbox-extensions-not-wired` (D-FLEET-6): fail-closed extension,
`-e` wiring, rc 15/16/17. Fresh sonnet evaluators: **PASS 92** and **PASS 96**, 0 blocking. Each
single-fix PR conflicted on the changelog within minutes, and the scope guard refuses a rebased
push. So both commits land in one PR, with their entries at the END of `[Unreleased]`. #1377 is
MERGEABLE, and every check except `lint` is green or pending. `lint` is required and red on dev itself
since `1fcc479f1` (a direct push; `internal/executor/motoko/healthcheck.go` gofmt), which is outside
the fleet's scope, so it is sent to `mission-v1`. **Update, same iteration:** `96fd6c5e1` fixed dev `lint`. The record push re-tested, all 4 required
checks went green, and Mark merged #1377 at 15:26Z (`eec86ca4f`). On the merge commit, `test`, `lint` and
`launchd drivers (bash 3.2)` are all success. Both tickets are resolved with that SHA (19 open). Clause map: **1 product share** unmeasured; **2
turnaround** at risk (two tickets resolved; a product red delayed them, the third time); **3 one queue** MET
(19 open); **4 idle is free** MET; **5 no regressions** MET (nothing landed unverified).

## STATUS 2026-09-29 — ITERATION 7: D-FLEET-7 stall-watchdog M2 merged in [#1399](https://github.com/sunholo-data/ailang/pull/1399) (`ce1c0639f`), independent evaluator PASS 88; merge CI SUCCESS (CI run 36562795612; 20 check rows settled success/skipped)

The approved arm counts descendant `proc_pid_rusage` growth of at least 10 CPU-s per existing 120-s sample, excluding the controller root's own CPU. The sample count and 600-s budget are unchanged. Focused stall suite 35/35 and full `make test-launchd-drivers` passed locally; all 22 PR checks settled success/skipped. Sonnet round 1 found two surviving root-accounting mutations and child churn; round 2 confirmed the fixes, PASS 88 with zero blocking. The ticket was resolved with merge SHA `ce1c0639f` after merge-commit CI. An initial Gate-2 mis-pick of `skill:heartbeat-relative-path-absent-in-world` missed Mark's ranked charter head; its design blocked at quorum twice and is parked for D-FLEET-8, with no heartbeat implementation. Clause map: **1 product share** unmeasured; **2 turnaround** improved for one ticket; **3 one queue** MET; **4 idle is free** MET; **5 no regressions** MET after merge CI.

## STATUS 2026-09-30 — ITERATION 8: paired rotate-log design PARKED · D-FLEET-9; quorum BLOCKED twice, independent Sonnet technical checkpoint PASS, implementation score UNMEASURED

Synthetic A/B registries reproduce a wrong-checkout write; `--status` mutates the status archive, while default World log selection is correct. No implementation or ticket resolution. Record-only [PR #1416](https://github.com/sunholo-data/ailang/pull/1416) is OPEN, needs-human-review; not landed and no auto-merge armed. The intentional flag break needs Mark, and the next design must audit all shared-loader callers or isolate stricter rotate resolution. [Design](planned/m-mission-rotate-log-safe.md), [independent evaluation](planned/m-mission-rotate-log-safe-evaluation.md), [evidence](planned/m-mission-rotate-log-safe-evidence.json). Goal unmoved: clause 1 unmeasured, 2 at risk, 3–4 met, 5 upheld by shipping nothing unverified. Dev CI is independently red on changelog hygiene; handed to V1.

## STATUS 2026-09-30 — ITERATION 9: `mission-base:hardcoded-origin-dev` LANDED in [#1418](https://github.com/sunholo-data/ailang/pull/1418) (`cb7c51c8e`), independent evaluator PASS 100; ticket resolved; iteration 8's record #1416 landed (`60af11ee0`)
`mission-base.sh` now takes `MISSION_BASE_REF`, else `origin/HEAD`'s target, else fails loudly (rc 1, both remedies named, no row written); no silent `origin/dev` fallback. Live A/B in the stapledon clone: old rc 1 `cannot resolve origin/dev`, new rc 0 recording `origin/main`; `origin/HEAD` measured in all 6 mission clones (5 → `origin/dev`, stapledon → `origin/main`). `test_mission_base.sh` 13/13, `make test-launchd-drivers` rc 0 (controller re-run), PR checks all green; merge-commit CI green except SonarCloud new-code coverage, red on the 4 prior dev commits too (inherited). Evaluator: pi minimax-m3 lane dead on launch (`sandbox_not_ready` rc 17: user-level `~/.pi/agent/extensions` tools collide with the worktree's `.pi/extensions`), declared fallback `claude-sonnet-4-6` PASS 100 with its own mutation drills. #1416 had been held only by dev's changelog-hygiene red, since fixed; branch updated, CI CLEAN, merged. Clause map: **1 product share** unmeasured; **2 turnaround** improved (5-occurrence ticket resolved, first filed 09-27 21:07Z, ~72h — over the 48h target); **3 one queue** MET (15 open); **4 idle is free** MET; **5 no regressions** MET.

## STATUS 2026-10-01 — ITERATION 10: `pi-runner:quota-429-reported-as-empty-worktree` LANDED in [#1424](https://github.com/sunholo-data/ailang/pull/1424) (`94524a6fc`), independent evaluator PASS 97; ticket resolved; stapledon's stale-pin re-file of `mission-base:hardcoded-origin-dev` resolved against `cb7c51c8e`

`mission_pi_run.sh` now types a run whose last assistant `message_end` is `stopReason:"error"` with a capacity `errorMessage` (429/402, usage limit, quota, rate limit, credits) as `provider_quota` rc 19, above `ok`/`empty_worktree`; every verdict carries `provider_errors`/`provider_error`. Planning found the worse twin: World iter 187's planner had banked a 429 stop as `ok` rc 0 with 10 files changed. Tests: `test_mission_pi_run_provider_quota.sh` 29 checks on real-pi fixtures, executor 10/10 mutants red, evaluator 5 own drills red; controller `make test-launchd-drivers` rc 0; PR checks CLEAN; merge-commit CI success (SonarCloud red inherited from `a03ec7013`). The gate-3-route.md rc list does not name 19 yet: its `.agents` mirror is outside the scope guard (D-FLEET-8 class); the existing text already falls back on any non-zero rc except 18. The pi evaluator lane died again on the extension collision (rc 17); fallback `claude-sonnet-4-6` judged. Clause map: **1 product share** unmeasured; **2 turnaround** at risk (this ticket filed 09-26 19:10Z, resolved ~4.5 days later); **3 one queue** MET (14 open); **4 idle is free** MET; **5 no regressions** MET.

## STATUS 2026-10-01 — ITERATION 11: heartbeat ticket PARKED-ON-LANE; independent evaluator unavailable in the requested Agent tool

D-FLEET-8 is acknowledged; its scope ruling is not re-asked. Designer, planner and executor ran read-only native `gpt-6.1-sol` checkpoints. All agree the inherited untracked design/plan/JSON must be revised for 18 heartbeat calls in 14 files, both skill mirrors, rc-19 text, and the pre-push mirror enforcement gap. Preferred evaluator `sonnet` and every declared fallback (`minimax-m3`, `claude-sonnet-4-6`, `opus`) were rejected by Agent spawn as Unknown model. No judge ran: evaluator score UNMEASURED, no implementation, no quorum PASS, no landing, no ticket resolution. Resume when a declared independent evaluator is spawnable with the requested Agent transport; then revise and follow the normal design/quorum/plan/execute gates. Clause map: **1 product share** UNMEASURED; **2 turnaround** UNMET/at risk (7 occurrences, first filed 09-26); **3 one queue** MET (25 open signatures at Gate 5); **4 idle is free** MET by prior evidence; **5 no regressions** preserved by shipping no fix. Record is review-only; see iteration 11 log.

## STATUS 2026-10-02 — ITERATION 12: heartbeat design advanced to Revision 4 across three quorum rounds, then PARKED needs-human-review on a reviewer-vs-measurement deadlock (D-FLEET-10); no implementation

D-FLEET-8 scope executed by the rotation designer (pi:openrouter/z-ai/glm-5.3, driver-resolved): Revision 2 extends the doc to both mirrors (18 calls / 14 files), the rc-19 line in both gate-3-route.md copies, and the pre-push guard allowlist seam; Revision 3 closes round 3 (catch-all sentence verified); Revision 4 replaces the over-general "not reproducible here" exit-status sentence with a measured two-shape fact (`bash -c` → rc=127, script file → rc=1, same /bin/bash 3.2.57; controller reproduced in six shapes). Quorum rounds 3/4/5 all BLOCKED: gemini-3-1-pro re-asserts "always rc=1, measurement fabricated" — empirically false for the -c shape on this rig, commands recorded in the doc; oc-kimi-k3 (round 5) raises a real residual: the `:?` guard tests nonemptiness, not absoluteness, so a nonempty RELATIVE root silently restores CWD-dependence (concrete one-token fix). One revision past the diet ceiling (FLAGGED). Ticket stays open; D-FLEET-10 asks Mark to adjudicate. Design doc banked as a record: `planned/m-mission-heartbeat-driver-root.md` Revision 4. Clause map: **1 product share** UNMEASURED; **2 turnaround** UNMET/at risk (ticket now 7 slots lost, first filed 09-26); **3 one queue** MET (25 open signatures); **4 idle is free** MET; **5 no regressions** preserved by shipping no unjudged change. See iteration 12 log.

## STATUS 2026-10-02 — ITERATION 13: iteration12 record recovered with separate Agent judge; heartbeat remains parked on D-FLEET-10

Record-only recovery; no implementation, plan approval or ticket resolution. Four native Agent role checkpoints ran; judge is a separate fresh-context Sol6.1 fallback, flagged. D-FLEET-10 remains OPEN; D-FLEET-8/9 preserved as RESOLVED. Quorum artifacts are banked with the rejected design; inherited stale plan/JSON and source worktrees remain untouched. See iteration13 log for verdict and exact-SHA CI disposition.

## STATUS 2026-10-02 — ITERATION 14: controller capacity ticket verified; narrow 402 proposal PARKED on D-FLEET-11, no implementation

Blocking-all ticket outranks the attended groom. Current fallback admission correctly skips blocked OpenRouter; historical start snapshots exclude OpenRouter and do not prove the reported bypass. Direct controller 402 refusal still misses runtime/transient signatures and becomes CRASHED. Proposed classifier and pause-notification diff is reviewable in `planned/m-controller-capacity-admission.md`; HD-2a ruling D-FLEET-11 required. Four native Agent roles ran; independent proposal review PASS91/100, implementation acceptance UNMEASURED. D-FLEET-10 remains OPEN; D-FLEET-8/9 remain RESOLVED. Clause map: 1 product share UNMEASURED; 2 turnaround UNMET/at risk (0 tickets resolved); 3 one queue MET (34 open signatures); 4 idle-is-free prior evidence only; 5 preserved by shipping no fix. Record-only PR pending exact-SHA CI; no runtime change, no ticket resolution.

## STATUS 2026-10-03 — ITERATION 17: heartbeat fix PUSHED as #1578, re-judged PASS 95; merge BLOCKED by a weekend-only red in `TestOllamaQuota*` (outside the diff) → PARKED-ON-CLOCK; ticket stays open

Resume predicate met at Gate 2 (pinned hook `_scope_is_harness .agents/skills/mission-control/x` → 0). Branch rebased onto `2a1f3f295`: 17/18 files blob-identical to the iteration-16 judged head `14b87879b` (the 18th is the design doc's status line). Pushed through the pinned guard, PR #1578. Independent Opus Agent judge re-scored the rebased head: **PASS 95/100**. It checked 18 guarded sites and 0 bare ones, ran a bash 3.2 + zsh drill from `/tmp` (rc0 with the absolute root; rc1 and a loud message for unset, relative and empty roots; never rc127; control: the old form gives rc127), got `make test-launchd-drivers` rc0, and confirmed one mutant red. Its two doc nits were applied (`ab0c539a1`). The required `test` check is RED on `TestOllamaQuotaVerifiedLimits` and `TestOllamaQuotaHTTPAndCredentialBinding` (`internal/mission/ollama_quota_test.go`). Both use `time.Now()` against weekday-only pacing plus #1524's 3pp ollama start margin, so they fail from Saturday until about Monday 08:00Z. It reproduces locally at `origin/dev` with no PR code, and dev's last CI at 14:43Z was green. Not fixed: D-FLEET-12 reserves the quota-margin work for attended sessions; handed to Mark. Clause map: **1** UNMEASURED; **2** UNMET (0 resolved; ticket open since 09-26); **3** MET (41 open signatures); **4** prior evidence only; **5** preserved (nothing merged).

## STATUS 2026-10-04 — ITERATION 18: rotate-log pair BUILT + judged PASS 97, PR #1580 PARKED-ON-CLOCK on the same weekend red as #1578; heartbeat #1578 re-probed still red

Both P0 heads are now blocked on one clock: the weekend-only `TestOllamaQuota*` red (D-FLEET-14, iteration 17's finding). Picked the rotate-log pair (D-FLEET-9 = A, top unblocked item). Design Rev 3 (designer `pi:openrouter/z-ai/glm-5.3`, 40 tool calls) carried the complete shared-loader caller audit (C1–C11); fresh quorum r3 BLOCKED (gemini: C7 normalize second-CWD-walk scope; kimi: path-formula ambiguity — the controller re-measured BOTH, correcting each reviewer's mechanism while confirming the substance); Rev 4 (designer) widened C7 to the resolved-root accessor and pinned `m.root` = parent of the registry dir (registry.go:245–246); quorum r4 BLOCKED on two NEW narrow objections (orphaned `repoRootFor`; `--status` audit directory-scoped) — both controller-verified first-party, both reviewer-verbatim and direction-preserving → **narrow-refinement carve-out round 5** applied by the controller (deletion specified; whole-repo `--status` caller enumeration with positive controls, V26/V27). Planner `pi:openrouter/moonshotai/kimi-k3` produced a baselined 4-milestone plan (345 LOC). Executor `pi:openrouter/deepseek/deepseek-v4.1-flash` landed all 4 milestones green; its one finding (grep count 2 vs plan's 1 — the rejection arm's own text contains `--status`) adjudicated PASS. Independent judge `pi:openrouter/minimax/minimax-m3` (handshake acked, isolated worktree): **PASS 97/100, 0 blocking**, mutation matrix 3/3 red-on-mutate/green-on-restore. Controller re-ran out-of-sandbox: 9/9 tests, `ok` Mission surface, `make test-launchd-drivers` rc0 (59 arms + new registry-env suite), healthy + degraded dry-runs `DRY RUN ok` (world profile, pinned `2bf95391f`; `lanes=DEGRADED(14)` is the genuine Sunday all-buckets-over-ration state, not a code defect). PR #1580: `launchd drivers` + `lint` + `govulncheck` + `CodeQL` green; required `test` red on exactly `TestOllamaQuotaVerifiedLimits`/`TestOllamaQuotaHTTPAndCredentialBinding` (04:09Z, outside the diff) → PARKED-ON-CLOCK. Clause map: **1** UNMEASURED; **2** UNMET (0 resolved; rotate-log ticket open since 09-26); **3** MET (41 open signatures); **4** prior evidence only; **5** preserved (nothing merged unjudged; the diff passed every check it can reach).
