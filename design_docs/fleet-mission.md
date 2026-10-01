# Fleet Mission — the loop harness is fixed by one loop, so the product loops never have to

<!--
  From design_docs/mission-charter-TEMPLATE.md. Design: design_docs/planned/m-harness-mission-loop.md
  (HD-1..HD-6 ratified by Mark 2026-09-26). Iteration 0 = ratify this charter with Mark, attended.
-->

**Type**: Long-running mission (peer of [v1-mission.md](v1-mission.md), [docs-mission.md](docs-mission.md)); advanced by a scheduled
outer loop on the always-on rig. **It fires only when there is open work.** The driver exits
before any probe or spawn when `ailang mission ticket open --count` is 0.
**North star**: product loops spend their iterations on product work. Every loop-harness defect
they hit leaves them as a ticket and comes back as a fix.
**Traces to**: [PROGRAM.md](PROGRAM.md) and [M-HARNESS-MISSION-LOOP](planned/m-harness-mission-loop.md).
Measured basis: World iterations 164–180 were 12/13 `[HARNESS]`; V1 309–348 were 18/35.
**Skill**: [.claude/skills/mission-control/SKILL.md](../.claude/skills/mission-control/SKILL.md), the SAME
unforked skill. Gate 2's **fleet branch** (`resources/gate-2-pick.md`) inverts admissibility for this
mission.
**Scheduling**: launchd `dev.ailang.mission-fleet`, registry entry [`missions/fleet.toml`](../missions/fleet.toml)
(interval 6h, boot offset 1680s). Billing guard as for every mission.
**Log**: [fleet-mission-log.md](fleet-mission-log.md), append-only, one entry per iteration.
**Human-facing reporting**: GitHub issue #1380 (rotated 2026-09-28 from #1321; live number in
`~/.ailang/state/mission-fleet-gh-issue`).

## Repo Profile (M-MISSION-PORTABILITY M2 — the per-mission values mission-control reads)

- **Repo slug**: `sunholo-data/ailang` (driver: `MISSION_REPO`)
- **Mission doc**: `design_docs/fleet-mission.md` (driver: `MISSION_DOC`)
- **Mission name / state namespace**: `fleet` (driver: `MISSION_NAME`; `~/.ailang/state/mission-fleet-*`)
- **Checkout**: work happens in the **pin worktree** `~/.ailang-driver-pin/fleet` (detached at the
  pinned `origin/dev`, fresh each fire). Every mission whose work repo IS `sunholo-data/ailang` runs
  this way (v1, docs, motoko; `pin-root.sh` `_set_pin_workdir`). The clone
  `~/dev/sunholo-data/ailang-fleet` is only launchd's working directory, as `ailang-docs` is for docs.
  Land changes through a branch or PR, never by editing the pin worktree in place.
- **Bookkeeping issue**: `#1380` (from `#1321`), rotates weekly; live number in `~/.ailang/state/mission-fleet-gh-issue`
- **CI workflows Gate 3b / Gate 1 poll**: `CI` (runs on every push; no push paths filter).
- **Verify profile**: `go-compiler`, plus the mission-loop-change pre-flight as the done-gate
  (below). Harness changes live in `tools/launchd/` (bash 3.2), `internal/mission/` and
  `cmd/ailang/mission*` (Go), and the mission skills.

---

## STATUS (rotation rule)

Newest **3** STATUS stamps live here; older ones move to `fleet-mission-status-archive.md`.

## STATUS 2026-10-01 — ITERATION 10: `pi-runner:quota-429-reported-as-empty-worktree` LANDED in [#1424](https://github.com/sunholo-data/ailang/pull/1424) (`94524a6fc`), independent evaluator PASS 97; ticket resolved; stapledon's stale-pin re-file of `mission-base:hardcoded-origin-dev` resolved against `cb7c51c8e`

`mission_pi_run.sh` now types a run whose last assistant `message_end` is `stopReason:"error"` with a capacity `errorMessage` (429/402, usage limit, quota, rate limit, credits) as `provider_quota` rc 19, above `ok`/`empty_worktree`; every verdict carries `provider_errors`/`provider_error`. Planning found the worse twin: World iter 187's planner had banked a 429 stop as `ok` rc 0 with 10 files changed. Tests: `test_mission_pi_run_provider_quota.sh` 29 checks on real-pi fixtures, executor 10/10 mutants red, evaluator 5 own drills red; controller `make test-launchd-drivers` rc 0; PR checks CLEAN; merge-commit CI success (SonarCloud red inherited from `a03ec7013`). The gate-3-route.md rc list does not name 19 yet: its `.agents` mirror is outside the scope guard (D-FLEET-8 class); the existing text already falls back on any non-zero rc except 18. The pi evaluator lane died again on the extension collision (rc 17); fallback `claude-sonnet-4-6` judged. Clause map: **1 product share** unmeasured; **2 turnaround** at risk (this ticket filed 09-26 19:10Z, resolved ~4.5 days later); **3 one queue** MET (14 open); **4 idle is free** MET; **5 no regressions** MET.

## STATUS 2026-09-30 — ITERATION 9: `mission-base:hardcoded-origin-dev` LANDED in [#1418](https://github.com/sunholo-data/ailang/pull/1418) (`cb7c51c8e`), independent evaluator PASS 100; ticket resolved; iteration 8's record #1416 landed (`60af11ee0`)

`mission-base.sh` now takes `MISSION_BASE_REF`, else `origin/HEAD`'s target, else fails loudly (rc 1, both remedies named, no row written); no silent `origin/dev` fallback. Live A/B in the stapledon clone: old rc 1 `cannot resolve origin/dev`, new rc 0 recording `origin/main`; `origin/HEAD` measured in all 6 mission clones (5 → `origin/dev`, stapledon → `origin/main`). `test_mission_base.sh` 13/13, `make test-launchd-drivers` rc 0 (controller re-run), PR checks all green; merge-commit CI green except SonarCloud new-code coverage, red on the 4 prior dev commits too (inherited). Evaluator: pi minimax-m3 lane dead on launch (`sandbox_not_ready` rc 17: user-level `~/.pi/agent/extensions` tools collide with the worktree's `.pi/extensions`), declared fallback `claude-sonnet-4-6` PASS 100 with its own mutation drills. #1416 had been held only by dev's changelog-hygiene red, since fixed; branch updated, CI CLEAN, merged. Clause map: **1 product share** unmeasured; **2 turnaround** improved (5-occurrence ticket resolved, first filed 09-27 21:07Z, ~72h — over the 48h target); **3 one queue** MET (15 open); **4 idle is free** MET; **5 no regressions** MET.

## STATUS 2026-09-30 — ITERATION 8: paired rotate-log design PARKED · D-FLEET-9; quorum BLOCKED twice, independent Sonnet technical checkpoint PASS, implementation score UNMEASURED

Synthetic A/B registries reproduce a wrong-checkout write; `--status` mutates the status archive, while default World log selection is correct. No implementation or ticket resolution. Record-only [PR #1416](https://github.com/sunholo-data/ailang/pull/1416) is OPEN, needs-human-review; not landed and no auto-merge armed. The intentional flag break needs Mark, and the next design must audit all shared-loader callers or isolate stricter rotate resolution. [Design](planned/m-mission-rotate-log-safe.md), [independent evaluation](planned/m-mission-rotate-log-safe-evaluation.md), [evidence](planned/m-mission-rotate-log-safe-evidence.json). Goal unmoved: clause 1 unmeasured, 2 at risk, 3–4 met, 5 upheld by shipping nothing unverified. Dev CI is independently red on changelog hygiene; handed to V1.

## CURRENT GOAL

1. **Iteration 0 (definition)**: DONE 2026-09-26, ratified as written.
2. **Then**: each fire takes the top open ticket through the inner loop (design only when the fix
   warrants one, then plan, execute, evaluate), lands it on `dev`, and resolves it.

## The bar (RATIFIED 2026-09-26)

- **Clause 1 — product share**: `[HARNESS]`-tagged share of product-loop iterations ≤ 10% over a
  rolling 14 days (design goal 1; measured by header scan of each `<name>-mission-log.md`).
- **Clause 2 — turnaround**: median ticket filed → resolved ≤ 48h for non-policy tickets.
- **Clause 3 — one queue**: every escalation is findable in `mission-fleet` (no scattered issues).
- **Clause 4 — idle is free**: a fire with no open tickets spends zero controller tokens.
- **Clause 5 — no regressions shipped**: every resolved ticket passed the done-gate below.

## Authority (the write scope — enforced by `tools/launchd/githooks/pre-push`)

- **May change**, as an allowlist enforced by the guard: the loop harness (`tools/launchd/**`,
  `internal/mission/**`, `cmd/ailang/mission*`, `missions/**`, `.claude/skills/mission-*/**`,
  `.claude/skills/sprint-*/**`, `.pi/extensions/**`, `scripts/hooks/**`, `scripts/mission_*`,
  `scripts/test_mission_*` (D-FLEET-5), its own
  `design_docs/fleet-mission*`), plus support paths (`design_docs/**`, `changelogs/**`,
  `CHANGELOG.md`, `docs/**`, `.ailang/state/sprints/**`, `internal/config/mission.go`, `make/test.mk`,
  `tools/pi-extensions/sandbox/**` (D-FLEET-6)).
  **Anything else is refused**, including CI workflows, `go.mod`, the server, the UI and non-mission
  commands. A fix that genuinely needs one of those parks for Mark.
- **May not change** (language core, refused by the guard): `internal/{parser,lexer,ast,types,
  elaborate,core,eval,vm,codegen,effects,builtins,pipeline,runtime,link,iface}/**`, `std/**`,
  `examples/**`, `benchmarks/**`.
- **Parks for Mark (HD-2a, policy class)**: any fix that changes routing, lane order, quota or
  ration thresholds, or a billing guard. **Also any change to `.pi/extensions/**`** (D-FLEET-3): the
  pi EVAL harness loads the same extensions, so a fleet fix there can shift eval results unnoticed.
  Such a ticket parks with the proposed diff and its expected eval impact. File a decision row with a recommendation; take the next
  ticket.

## Guardrails (on top of the skill's Standing Rules)

- **Ticket-driven only.** The queue is `ailang mission ticket open --json` plus Mark's directives.
  No self-sourced audits, no "while I'm here". The 2026-09-21 attended hunt found harness defects
  faster than they could be fixed and never ran out.
- **Unread = open.** Never `ailang messages read` on `mission-fleet`. Close a ticket only with
  `ailang mission ticket resolve <signature> --resolution … --sha <commit on origin/dev>`.
- **Done-gate = the mission-loop-change skill's five-line pre-flight**: the surface that actually
  runs was edited; reach was confirmed (pushed, since every mission runs the pinned `origin/dev`);
  dry-run healthy **and** degraded (set `AILANG_DRIVER_PINNED=<sha>` so the pin cannot re-exec into
  another copy, design V15); `make test-launchd-drivers` green; nothing reloaded mid-iteration.
- **Every mission runs `origin/dev` until Phase 3b** (`harness-stable`). A fleet push reaches all
  loops on their next fire, so the done-gate is not optional.

## Routing policy

Shared per-role routing from `mission-control`. Overrides in `~/.config/ailang/mission-fleet.env`:

- **Executor default**: shared default (codex, then pi, then opus).
- **Evaluator**: shared default (`sonnet`); generator ≠ judge is enforced by the skill.
- **Opus before pi**: on (`MISSION_OPUS_BEFORE_PI=1`), same as World. Harness fixes are
  high-blast-radius, so capability beats flat-rate here.

## Decisions (policy rulings the fleet may act on; HD-2a)

| ID | Status | Ruling |
|---|---|---|
| D-FLEET-1 | RULED 2026-09-26 (Mark, attended: "yes") | Design docs without `planner_lane` default to the mission's planner pin, not `opus fail-closed`. |
| D-FLEET-2 | RULED 2026-09-26 (Mark, attended: "yes") | The spawn-pin hook may walk the role's DECLARED fallback chain, in order, when the pinned model is dead. No undeclared model is ever allowed. |
| D-FLEET-3 | RULED 2026-09-26 (Mark, attended: "do the follow ups") | Changes to `.pi/extensions/**` park for Mark with the diff and expected eval impact; the pi eval harness shares them. Affects `pi-runner:sandbox-extensions-not-wired` (P0 #4): the fleet designs it, and Mark approves before it lands. |
| D-FLEET-4 | RULED 2026-09-27 (Mark, attended: "go with the recommendations") | **YES to M1 only.** Measure `ps -S -o time` CPU deltas on the rig for the long-drill shape and both reference wedges (plan: `design_docs/planned/sprint-plan-stall-descendant-progress.md`), then bring a MEASURED threshold back as a new decision row. Until that row is ruled, thresholds and sample counts are unchanged; the provisional ≥2 CPU-s / 120 s is not live. |
| D-FLEET-5 | RULED 2026-09-27 (Mark, attended: "go with the recommendations") | **YES.** `scripts/test_mission_*` is a harness path: the fleet may change it, product loops are refused on it (`tools/launchd/githooks/pre-push`, `_scope_is_harness`). The fleet may now repair `scripts/test_mission_pi_run.sh` TEST 3. |
| D-FLEET-6 | RULED 2026-09-27 (Mark, attended: "go with the recommendations") | **YES to M1–M3 as one fleet sprint** (`design_docs/planned/sprint-plan-pi-runner-sandbox-wiring.md`): the extension fails closed and reads an explicit policy file; `scripts/mission_pi_run.sh` passes `-e` with typed verdicts rc 15/16/17; tests. The guard allows the fleet `tools/pi-extensions/sandbox/**` only; the rest of `tools/pi-extensions/**` stays outside its scope because the eval harness shares it. This ruling discharges D-FLEET-3 for this ticket. |
| D-FLEET-7 | RULED 2026-09-29 (Mark, attended: "I agree with D-FLEET-7 recommendation") | **Stall-watchdog CPU arm: approve the measured threshold?** M1 found two things. On the rig, `ps -S` cannot see reaped-child CPU; `proc_pid_rusage` (`ri_child_*`) can. And an idle `claude` root accrues 0.57–1.60 CPU-s per 120 s. **Recommendation: YES to M2 with this arm:** count a window as progress when the cumulative rusage CPU of the controller's DESCENDANTS, excluding the root, grows by **≥10 CPU-s per 120-s sample**. Measured windows: drill 62.5–97.7, w1-git ≤1.72, w1-gh ≤0.31, w2 0 (`design_docs/planned/sprint-plan-stall-descendant-progress.md`, M1 result). It needs `python3` (ctypes) in the driver path. Sample counts and the 600 s budget stay unchanged. Risk: a W1 whose poll condition is itself heavy (for example a `go test`) would read live. **RULING: YES to M2 with the recommended arm** (≥10 CPU-s of descendant rusage per 120-s sample counts as progress; sample counts and the 600-s budget unchanged). The heavy-poll W1 risk is accepted. Note: #1391 (2026-09-29) separately bounded pi controller commands at 540 s; this arm is for the claude long-drill shape. |
| D-FLEET-8 | RULED 2026-10-01 (Mark, attended: "go with all your recommendations") | **ANSWERED — YES: the next heartbeat design fixes BOTH the `.claude/skills/mission-control/resources/**` and `.agents/skills/mission-control/resources/**` call sites and removes the divergence; the gate-3-route.md rc-19 line is in scope for both copies.** Heartbeat design blocked after two quorum rounds. Should the next design update both tracked `.claude/skills/mission-control/resources/**` and `.agents/skills/mission-control/resources/**` call sites? **YES (recommended):** fix both copies and remove the divergence; **NO:** scope to the user-named authoritative `.claude` copy and leave a separate mirror ticket. Default if unanswered: park this heartbeat ticket; no implementation. |

| D-FLEET-9 | RULED 2026-10-01 (Mark, attended: "go with all your recommendations") | **ANSWERED — A: reject the old `--status` before any write, replace it with `--stream status`, and revise with a complete shared-loader caller audit (or rotate-only strictness). Still needs a fresh independent quorum and the normal approved-plan/execute gates.** Paired rotate-log design is needs-human-review after two BLOCKED quorums. Approve the proposed legacy flag migration and next design scope? **A (recommended):** reject old `--status` before writes, replace with `--stream status`, and revise with a complete shared-loader caller audit or rotate-only strictness; prevents misleading mutation. **B:** require guarded deprecation instead; next designer must specify opt-in mutation and a sunset before planning. Either option still requires fresh independent quorum and the normal approved-plan/execute gates. Default if unanswered: keep both rotate-log tickets parked and take the next READY charter item on the next fire. |

## Queue (top = next; tags: [NEXT] [IN-SPRINT] [PARKED] [LANDED] [RULED OUT])

The live tickets are **`ailang mission ticket open`**. This section is **Mark's triage order**,
**RE-RANKED 2026-09-29 (Mark, attended: "rerank and remove anything stale")**. It outranks the
`slots_lost` ranking. Now that tickets recur, the ranking uses `slots_lost` inside each tier; the
2026-09-26 backfill order no longer applies. New tickets rank by `slots_lost` BELOW this list unless
they are `blocking=all`. **Every open ticket was re-verified at `131286748` on 2026-09-29** (attended,
evidence in each line). Still re-check at HEAD in Gate 2 before working a ticket.

**Stale items removed 2026-09-29:**
- `ci:launchd-driver-suite-flakes` was RESOLVED: its signature did not occur in the 3 days before
  2026-09-29. It was refiled as `ci:launchd-suite-flakes-openrouter-peer-and-gh-hang` (P2 #17).
- `pi-runner:sandbox-blocks-mission-inbox-handshake` was RESOLVED by #1393.
- Landed entries were removed from the queue: the old P0 #1 (#1325), P0 #4 (#1377), the aqua-session
  queue-jumper (#1377), and the P1 #0 env-scrub directive (#1330).
- The "likely stale" note on the heartbeat ticket was WRONG: World re-filed it on 2026-09-28.

**P0 — whole slots lost** (the two unbuilt rulings, D-FLEET-1 and D-FLEET-2, landed in #1398)
1. [UNPARKED · D-FLEET-8 = YES, both copies] `skill:heartbeat-relative-path-absent-in-world`: 9 relative `bash
   tools/launchd/mission-heartbeat.sh stamp` calls remain across gate-0..gate-5 resources. They do not
   exist from World's (or Stapledon's) CWD. `$AILANG_DRIVER_SRC` is used 0 times in `resources/`.
   **4 slots lost.**
2. [UNPARKED · D-FLEET-9 = A] `mission:rotate-log-registry-cwd` + `rotate-log:status-flag-mutates-and-world-resolves-to-status-archive`
   (one command, one sprint):
   - The registry still defaults to CWD-relative `missions` (`mission_cmd.go:25,155-186`).
     `AILANG_MISSION_REGISTRY` (absolute path, ad1bf98d3) is an escape hatch that nothing sets.
   - `--status` means "rotate the status archive" and always writes. It needs a rename or a real
     report-only mode.
   - **3 + 1 slots lost** (stapledon, v1, world).
3. [LANDED] `mission-base:hardcoded-origin-dev`: #1418 `cb7c51c8e` (iteration 9), ticket resolved.
**P1 — a lane misreported as dead, or a failure nobody sees** (the class that cost 2026-09-28/29's
overnight: harness faults read as model faults)
4. [LANDED] `pi-runner:quota-429-reported-as-empty-worktree`: #1424 `94524a6fc` (iteration 10), ticket
   resolved. Verdict `provider_quota` rc 19; the gate-3-route.md rc-list line is in scope of the D-FLEET-8 heartbeat fix (both copies, ruled 2026-10-01).
5. [NEXT] `pi-runner:verdict-blind-to-commits-and-predirty`, the **pre-dirty half**: there is no porcelain
   snapshot before the run (`:338-339` says so itself). The commit-blind half landed in #1329.
6. `skill-surface:main-checkout-not-synced-to-dev` (filed 2026-09-29): loops read skills from the
    main checkout, which nothing fast-forwards. Merged skill fixes wait for a manual pull, and a paused
    rebase can freeze them. This feeds the Phase 3a skill-resolution directive below.
7. `gate0:driver-crash-notices-invisible`: `gate-0-preflight.md` still reads directives but no slot
    verdicts or rc=143 notices. The World-local fix (row 73) was never ported.
8. `quorum:zero-signal-guard-vacuous-with-controller-verdict`: `quorum.go:164-165` counts the
    controller as present before the zero-signal guard (:176). A controller-only "quorum" proceeds
    (ailang#651).
9. `driver:exit-path-notices-unbounded` (**PARTIAL**): the slot-verdict notices are bounded
    (10448bad5). Unbounded: `:2579/2582`, `:2596/2599`, `:2114/2117` and `:2133`.

**P2 — hygiene, or correctness with a workaround**
10. `skill:gate0-ledger-provenance-S`: `gate-0-preflight.md:174` still uses `-S'| D-nn |'`, which
    returns a row's CREATION commit, so attended rulings read as self-resolution. Use `-G` with the
    status.
11. `weekly-report:unknown-mission`: `tools/mission-weekly-report.py:28-33` hardcodes
    `[v1, world, motoko]`. Read `missions/*.toml`.
12. `quorum:invalid-absent-on-quoted-literals`: `ParseReviewResult` is still strict. The repair is
    designed (`design_docs/planned/m-quorum-salvage-retry.md`) but not built.
13. `ci:launchd-suite-flakes-openrouter-peer-and-gh-hang` (filed 2026-09-29): 5 launchd-job failures
    between 09-26 and 09-28 on two signatures, none since 09-28 13:44Z. Verify one reproduces first.
14. `quorum:artifact-dir-cwd-relative`: `artifact.go:14` is still CWD-relative. `--artifact-dir` is
    the workaround.
15. `driver:unclassified-state-unreachable`: `UNCLASSIFIED` appears 0 times in the driver.
    Low severity.

**Landed (history)**
- `pi-runner:quota-429-reported-as-empty-worktree`: #1424 `94524a6fc` (iteration 10).
- `mission-base:hardcoded-origin-dev`: #1418 `cb7c51c8e` (iteration 9).
- `stall-watchdog:kills-controller-on-long-drill`: #1399 `ce1c0639f`, D-FLEET-7 M2.
- `driver:slot-kill-leaves-orphan-descendants`: #1325.
- `pi-runner:sandbox-extensions-not-wired`: #1377.
- `rig:aqua-session-lost:windowserver-watchdog`: #1377.
- `tests:driver-env-leaks-into-launchd-suite`: #1330.
- `pi-runner:verdict-blind-to-commits-and-predirty`, commit-blind half: #1329.
- `pi-runner:sandbox-blocks-mission-inbox-handshake`: #1393.
- The pi fallback rung set (attended 2026-09-29): #1391.
- `resolver:planner-lane-field-missing-vs-spawn-pin` (D-FLEET-1 built): #1398 `49f18bdbd`.
- `spawn-pin-hook:no-fallback-mode` (D-FLEET-2 built): #1398 `49f18bdbd`.
- `skills:agents-copies-stale`, fleet scope (mission-*/sprint-*): #1398 `49f18bdbd`. The
  `.agents`-only edits in model-manager and design-doc-creator still need a reconciliation
  (V1's scope).

**[DIRECTIVE, Mark 2026-09-26] After P1, before P2** (still stands after the 2026-09-29 re-rank; P1 #10 feeds it): the Phase 3a skill-resolution spike
(design doc M-HARNESS-MISSION-LOOP, premises P1–P3). Measure how the claude, codex and pi controllers
each resolve skills, using marker skills in the pin worktree against conflicting user-level copies.
**Measurement only**: record the results in the design doc's Verification Log and park Phase 3b's
mechanism choice for Mark. This is what unblocks `harness-stable`.

---
**Document created**: 2026-09-26. Iteration 0 ratifies it with Mark before any ticket routes.
