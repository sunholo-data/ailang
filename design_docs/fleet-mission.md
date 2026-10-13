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
**Human-facing reporting**: GitHub issue #1768 (rotated 2026-10-12 from #1584; live number in
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
- **Bookkeeping issue**: `#1768` (from `#1584`), rotates weekly; live number in `~/.ailang/state/mission-fleet-gh-issue`
- **CI workflows Gate 3b / Gate 1 poll**: `CI` (runs on every push; no push paths filter).
- **Verify profile**: `go-compiler`, plus the mission-loop-change pre-flight as the done-gate
  (below). Harness changes live in `tools/launchd/` (bash 3.2), `internal/mission/` and
  `cmd/ailang/mission*` (Go), and the mission skills.

---

## STATUS (rotation rule)

Newest **3** STATUS stamps live here; older ones move to `fleet-mission-status-archive.md`.

## STATUS 2026-10-13 — ITERATION 34: Phase 3a skill-resolution spike MEASURED — P1–P3 answered with controlled marker skills, judged PASS 88 by sonnet; Phase 3b mechanism parked for Mark as D-FLEET-16 [HARNESS]

Mark's "after P1, before P2" directive was the pick; it is measurement only. The codex planner wrote the protocol (`sprint-plan-phase3a-skill-resolution-spike.md`), the codex executor built `skill_resolution_spike.sh`, and the controller ran it outside any sandbox. The controller first repaired 11 instrument defects that only live runs could expose; each is recorded in `fleet-mission-evidence/iteration34/findings.md`. Each run used per-run random tokens with before/after explicit-path and negative controls: claude 51/51 controls, pi 44/45, codex 32/43, with every codex miss a named provider-capacity error or scorer limit.

Findings:
- **claude** discovers project `.claude/skills` from cwd, but a user-level same-name copy wins over every project copy and over `--add-dir`. A fresh `CLAUDE_CONFIG_DIR` keeps subscription auth under the loops' env token, and its `skills/` wins.
- **codex** discovers `.agents/skills` and `$CODEX_HOME/skills` (the latter wins a collision), never `.claude/skills`. A fresh `CODEX_HOME` returns 401 without `auth.json`.
- **pi** discovers nothing in `--print`. `--skill <abs>` loads a skill that is reachable only by `/skill:<name>`.

The independent sonnet judge (own worktree) scored **PASS 88**, 0 blocking, and replicated C1–C3 with its own method. Its N2 finding (the concurrency audit could mask a foreign deletion) was fixed and mutation-drilled both ways. Every provider is pinnable for skills: codex only through the `.agents` mirror, pi only explicitly. Phase 3b's mechanism (A retire the user symlinks / B per-fire config dirs / C no change) is **D-FLEET-16, OPEN for Mark**.

Clause map: **1** UNMEASURED; **2** UNMET (no ticket resolved; the directive is not a ticket); **3** MET (52 open); **4** prior evidence only; **5** preserved (no harness code shipped). Next: P2 #10 `skill:gate0-ledger-provenance-S`.

## STATUS 2026-10-12 — ITERATION 33: P1 #9 exit-path notices bounded and store-scoped LANDED — #1791 `e7dd6c127`, judged PASS 90 by sonnet; ticket resolved; P1 tier exhausted [HARNESS]

The three remaining unbounded exit-path notice pairs in `tools/launchd/mission-control.sh` (the refusal, late-kill and rc-fail notices) now go through `_mc_exit_notice`. It makes one `_mc_bounded "$NOTIFY_TIMEOUT"` attempt per channel and always returns 0; a failure is logged as a WARNING with its reason, and a duplicate title is logged as already in the inbox. All three, plus the slot-verdict send, are now store-scoped (`env AILANG_STORAGE_MESSAGING=gcp …`). Before this, they wrote to the rig's local SQLite inbox. Titles, bodies, exit codes and the gate-0 F8 literal are byte-identical.

The opus planner chose the helper over `_mc_notify` (no title change, and no retry that would stretch the exit path). The haiku executor (`claude-sub`) added hang, preservation and source-guard arms: 11 red before the fix, 115/0 after. The sonnet judge, in its own worktree, scored **PASS 90** with 0 blocking and M1–M5 mutants killed (M6 unmeasured). The judge's 2 red routing-note arms also failed on base and came from the controller's exported env; under `env -i` the suite is 115/0 (addendum banked).

PR #1791 head `986def325`: 25 checks, 0 failures. Squash-merged → `e7dd6c127`. `make test-launchd-drivers` at the merge: rc 0 under `env -i`, 590 s. Healthy and degraded dry-runs under the idle stapledon profile: both `DRY RUN ok`, rc 0. Ticket resolved. Follow-ups, not ticketed: undated exit-notice titles, and unbounded sends in nightly-eval and other rig scripts.

Clause map: **1** UNMEASURED; **2** UNMET (this ticket ≈16.5 d); **3** MET (53 open); **4** prior evidence only; **5** preserved. Next: the Phase 3a skill-resolution spike (P1 is now exhausted; directive "after P1, before P2").

## STATUS 2026-10-12 — ITERATION 32: P1 #8 quorum zero-signal guard LANDED — #1767 `a5f9af982`, judged PASS 95 by sonnet; ticket resolved; orphans 30/31 credited [HARNESS]

Resumed iteration 28's PARKED-ON-LANE item. The predicate held this fire: Anthropic was available, and the evaluator resolved to `agent-tool sonnet`, which spawned. Two unrecorded slots had already carried the item:
- **Iteration 30** (10-09, glm controller): quorum rounds 1–2 BLOCKED, then revision 3 with the reviewers' verbatim fixes under the narrow-refinement carve-out.
- **Iteration 31** (10-12 01:08Z): replayed the revisions onto `0a2202ad6` and left an untracked plan.

Both are credited in the log. The opus planner adopted the plan with 10 corrections, two of which prevented vacuous tests: call `SynthesizeWithTier2` directly, and use non-zero absent costs. The haiku executor (`claude-sub`) produced the fix (`externalPresentCount`, no controller increment, guard text unchanged) and a matrix, seated and Tier-2 regression set; the new tests were red before the fix. The sonnet judge, in its own worktree, scored **PASS 95** with 0 blocking and 6 of 6 mutants killed; the controller re-ran MU1 independently (3 tests red, restore `cmp`-identical).

PR #1767 head `c55bc9a39`: 26 checks, 0 failures. Squash-merged → `a5f9af982`. Dev CI on `a5f9af982`: `test`, `lint`, `launchd drivers (bash 3.2)`, govulncheck, `test-windows` and all `Build` legs success (21 checks; release `build` still running at record time). `make test-launchd-drivers` at `a5f9af982`: rc 0 unpiped, 558 s. The pending record #1710 (iterations 28–29) also landed (`6d0e5fd76`) after a TempDir-race flake was re-run. Ticket resolved; #651 closed with a verdict.

Reach caveat: loops run `design-quorum` through the PATH binary, which carries the fix only after its next rebuild or release. Weekly sweep: 60 open issues enumerated, 53 with zero mentions, 0 harness-shaped, so no new queue items.

Clause map: **1** UNMEASURED; **2** UNMET (this ticket took ≈16 d, mostly a capacity park); **3** MET; **4** prior evidence only; **5** preserved.

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
  `.claude/skills/sprint-*/**`, `.agents/skills/mission-*/**`, `.agents/skills/sprint-*/**`, `.pi/extensions/**`, `scripts/hooks/**`, `scripts/mission_*`,
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
  **The fleet cannot dry-run itself mid-iteration** (its own overlap guard, driver :1895, yields
  before the dry-run exit at :1908; iterations 9 and 15). Dry-run the patched driver under an idle,
  armed sibling profile instead: `env -i HOME=$HOME PATH=$PATH USER=$USER TMPDIR=$TMPDIR
  AILANG_DRIVER_PINNED=<sha> MISSION_PROFILE=<idle armed mission> MISSION_DRY_RUN=1 /bin/bash
  tools/launchd/mission-control.sh` (exits before notices and the pidfile write; check its kill switch
  and pidfile first). Record which profile ran.
  **When no sibling is idle and armed** (iteration 27: world and stapledon mid-fire, the rest disabled), dry-run
  `MISSION_PROFILE=fleet` under a synthetic HOME instead: symlink every `~/.ailang/*` entry except `state/`,
  plus `~/.config`, `~/.codex`, `~/.claude`, `~/.local`, `~/.pi` and `~/.gitconfig`, into a temp dir, and pass it as `HOME=` in the
  `env -i` line. The kill switch and pidfile then resolve to the empty synthetic `state/`, the real lanes still probe,
  and nothing under the real `~/.ailang/state` is written.
- **Every mission runs `origin/dev` until Phase 3b** (`harness-stable`). A fleet push reaches all
  loops on their next fire, so the done-gate is not optional.
- **A fix that widens the scope guard ships ALONE, first** (iteration 16). The guard that judges a
  push is the PINNED `origin/dev` copy (`core.hooksPath` = the pin worktree), so a PR cannot push
  paths that only its own guard change admits. Land the guard arm + Authority line as their own PR;
  the dependent change pushes on the next fire. Never `--no-verify`, never point `core.hooksPath` at
  the branch's own hook.

## Routing policy

Shared per-role routing from `mission-control`. Overrides in `~/.config/ailang/mission-fleet.env`:

- **Executor default**: shared default (codex, then pi, then opus).
- **Evaluator**: shared default (`sonnet`); generator ≠ judge is enforced by the skill.
- **Opus before pi**: on (`MISSION_OPUS_BEFORE_PI=1`), same as World. Harness fixes are
  high-blast-radius, so capability beats flat-rate here.

## Decisions (policy rulings the fleet may act on; HD-2a)

<!-- decision-ledger:start -->
| ID | Status | Ruling | Evidence |
|---|---|---|---|
| D-FLEET-1 | RESOLVED | Design docs without `planner_lane` default to the mission's planner pin, not `opus fail-closed`. | RULED 2026-09-26 (Mark, attended: "yes") |
| D-FLEET-2 | RESOLVED | The spawn-pin hook may walk the role's DECLARED fallback chain, in order, when the pinned model is dead. No undeclared model is ever allowed. | RULED 2026-09-26 (Mark, attended: "yes") |
| D-FLEET-3 | RESOLVED | Changes to `.pi/extensions/**` park for Mark with the diff and expected eval impact; the pi eval harness shares them. Affects `pi-runner:sandbox-extensions-not-wired` (P0 #4): the fleet designs it, and Mark approves before it lands. | RULED 2026-09-26 (Mark, attended: "do the follow ups") |
| D-FLEET-4 | RESOLVED | **YES to M1 only.** Measure `ps -S -o time` CPU deltas on the rig for the long-drill shape and both reference wedges (plan: `design_docs/planned/sprint-plan-stall-descendant-progress.md`), then bring a MEASURED threshold back as a new decision row. Until that row is ruled, thresholds and sample counts are unchanged; the provisional ≥2 CPU-s / 120 s is not live. | RULED 2026-09-27 (Mark, attended: "go with the recommendations") |
| D-FLEET-5 | RESOLVED | **YES.** `scripts/test_mission_*` is a harness path: the fleet may change it, product loops are refused on it (`tools/launchd/githooks/pre-push`, `_scope_is_harness`). The fleet may now repair `scripts/test_mission_pi_run.sh` TEST 3. | RULED 2026-09-27 (Mark, attended: "go with the recommendations") |
| D-FLEET-6 | RESOLVED | **YES to M1–M3 as one fleet sprint** (`design_docs/planned/sprint-plan-pi-runner-sandbox-wiring.md`): the extension fails closed and reads an explicit policy file; `scripts/mission_pi_run.sh` passes `-e` with typed verdicts rc 15/16/17; tests. The guard allows the fleet `tools/pi-extensions/sandbox/**` only; the rest of `tools/pi-extensions/**` stays outside its scope because the eval harness shares it. This ruling discharges D-FLEET-3 for this ticket. | RULED 2026-09-27 (Mark, attended: "go with the recommendations") |
| D-FLEET-7 | RESOLVED | **Stall-watchdog CPU arm: approve the measured threshold?** M1 found two things. On the rig, `ps -S` cannot see reaped-child CPU; `proc_pid_rusage` (`ri_child_*`) can. And an idle `claude` root accrues 0.57–1.60 CPU-s per 120 s. **Recommendation: YES to M2 with this arm:** count a window as progress when the cumulative rusage CPU of the controller's DESCENDANTS, excluding the root, grows by **≥10 CPU-s per 120-s sample**. Measured windows: drill 62.5–97.7, w1-git ≤1.72, w1-gh ≤0.31, w2 0 (`design_docs/planned/sprint-plan-stall-descendant-progress.md`, M1 result). It needs `python3` (ctypes) in the driver path. Sample counts and the 600 s budget stay unchanged. Risk: a W1 whose poll condition is itself heavy (for example a `go test`) would read live. **RULING: YES to M2 with the recommended arm** (≥10 CPU-s of descendant rusage per 120-s sample counts as progress; sample counts and the 600-s budget unchanged). The heavy-poll W1 risk is accepted. Note: #1391 (2026-09-29) separately bounded pi controller commands at 540 s; this arm is for the claude long-drill shape. | RULED 2026-09-29 (Mark, attended: "I agree with D-FLEET-7 recommendation") |
| D-FLEET-8 | RESOLVED | **ANSWERED — YES: the next heartbeat design fixes BOTH the `.claude/skills/mission-control/resources/**` and `.agents/skills/mission-control/resources/**` call sites and removes the divergence; the gate-3-route.md rc-19 line is in scope for both copies.** Heartbeat design blocked after two quorum rounds. Should the next design update both tracked `.claude/skills/mission-control/resources/**` and `.agents/skills/mission-control/resources/**` call sites? **YES (recommended):** fix both copies and remove the divergence; **NO:** scope to the user-named authoritative `.claude` copy and leave a separate mirror ticket. Default if unanswered: park this heartbeat ticket; no implementation. | RULED 2026-10-01 (Mark, attended: "go with all your recommendations") |

| D-FLEET-9 | RESOLVED | **ANSWERED — A: reject the old `--status` before any write, replace it with `--stream status`, and revise with a complete shared-loader caller audit (or rotate-only strictness). Still needs a fresh independent quorum and the normal approved-plan/execute gates.** Paired rotate-log design is needs-human-review after two BLOCKED quorums. Approve the proposed legacy flag migration and next design scope? **A (recommended):** reject old `--status` before writes, replace with `--stream status`, and revise with a complete shared-loader caller audit or rotate-only strictness; prevents misleading mutation. **B:** require guarded deprecation instead; next designer must specify opt-in mutation and a sunset before planning. Either option still requires fresh independent quorum and the normal approved-plan/execute gates. Default if unanswered: keep both rotate-log tickets parked and take the next READY charter item on the next fire. | RULED 2026-10-01 (Mark, attended: "go with all your recommendations") |

| D-FLEET-10 | RESOLVED | **ANSWERED — A: the measured two-shape fact (rc=127 for `bash -c` strings, rc=1 for script files, bash 3.2.57) is ruled correct; add the absoluteness guard on `MISSION_DRIVER_ROOT` and satisfy Kimi's concrete residuals in Revision 5; fresh quorum next fire. A reviewer that cannot run rig commands does not overrule a recorded rig measurement.** **Heartbeat design quorum deadlock — adjudicate the measurement dispute and the absoluteness guard?** gemini-3-1-pro rejected rounds 4 and 5 asserting bash `${...:?}` failures "consistently exit rc=1 (including 3.2)" and calling the doc's measured rc=127 "fabricated"; the controller reproduced rc=127 for `bash -c` command strings and rc=1 for script files in six shapes on the rig's only bash (3.2.57), commands recorded in the doc's Verification Log — the reviewer cannot run rig commands. Separately oc-kimi-k3 proposes a one-token absoluteness test on `MISSION_DRIVER_ROOT` (the `:?` guard accepts a nonempty RELATIVE root, silently restoring the CWD-dependence the ticket exists to kill). **A (recommended):** rule the measured two-shape fact correct, add the absoluteness guard and satisfy Kimi's concrete residuals (fresh foreign-CWD drills, content-based rc19 acceptance, fourteen setup sentences, neighboring root conventions) in Revision 5, fresh quorum next fire. **B:** bench gemini-3-1-pro from this doc's panel (2 recorded false factual rejections) and re-quorum Revision 5 at N-1. **C:** controller proceeds on the measured fact without re-quorum (not recommended — quorum is a gate). Default if unanswered: ticket stays parked, next fire takes the next ticket. | RULED 2026-10-02 (Mark, attended: "yes do that"; commit 7a771034c via #1508, outside any fleet slot). Status token normalized OPEN→RESOLVED by iteration 15, answer unchanged. Filed 2026-10-02, iteration 12. |
| D-FLEET-11 | RESOLVED | **ANSWERED — A: authorize the narrow scope as proposed (anchored HTTP 402 classification through the existing bounded demotion/re-walk and PAUSED-NO-CAPACITY handling, pause-aware final notification, controller regression tests). Thresholds, lane order, billing reader and rc stay unchanged.** **Controller HTTP 402 capacity handling — approve narrow scope?** Current admission already skips over-ration OpenRouter; start-time historical snapshots exclude that bucket. Confirmed residual: direct pi controller credit refusal misses runtime classification and reports CRASHED. **A (recommended):** authorize only anchored 402 classification through existing bounded demotion/re-walk and PAUSED-NO-CAPACITY handling, pause-aware final notification, and controller regression tests (proposal: `planned/m-controller-capacity-admission.md`); thresholds, lane order, billing reader and rc stay unchanged. **B:** require a separately measured admission/billing design and rc75-consumer audit first. **NO:** leave parked without changes. Default if unanswered: keep this ticket parked, take the rotate-log pair next fire. Any ruling still needs admitted independent quorum, approved plan, explicit execution and judged done-gates. | RULED 2026-10-02 (Mark, attended: "yes do that"; commit 2533a43d6 via #1508, outside any fleet slot). Status token normalized by iteration 15, answer unchanged. Filed 2026-10-02, iteration 14; HD-2a. |
| D-FLEET-12 | RESOLVED | **HD-2a PRE-AUTHORIZATION — these attended tickets may be planned and implemented without parking for a further ruling (normal quorum/plan/evaluate gates still apply):** (1) `routing:retired-codex-models-still-spawnable-natively` — mission roles may spawn only `gpt-6.1-sol` among OpenAI models; `gpt-6-astra` stays in the registry, dormant; no Terra rung until a GPT-6.x Terra ships. (2) `quota:opencode-reported-as-a-pool` — remove opencode from the quota report; count opencode-routed models under their backend bucket. **`quota:admission-has-no-headroom-margin-and-flaps` is being implemented ATTENDED (margins codex 2pp, anthropic 1pp, ollama 3pp, openrouter $0.60); fleet must not pick it up in parallel.** | RULED 2026-10-02 (Mark, attended; commit 7a771034c via #1508). Attended review of today's fleet parks (iterations 13, 14). Status token normalized by iteration 15. |
| D-FLEET-13 | RESOLVED | **ANSWERED — A (Mark Edmondson, attended 2026-10-08, recorded directly in this ledger). IMPLEMENTED ATTENDED in the same change:** `hit your session limit` added to `RUNTIME_QUOTA_SIG` under D-FLEET-11's narrow scope; new `session-limit-is-capacity` case in `test_controller_capacity.sh` (fails with the signature removed). No threshold, order or rc change; nothing left for the fleet to plan. **Anthropic "hit your session limit" is recorded as CRASHED: authorize the same narrow fix as D-FLEET-11?** Iteration 15 attempt 1 (fire 2026-10-02T21:39Z) died with `You've hit your session limit · resets 12:40am` → `CRASHED at=gate-3 rc=1`. `RUNTIME_QUOTA_SIG` matches "hit your usage limit" but not "hit your session limit", so a capacity stop skips demote/re-walk/PAUSED. The fleet cannot file a ticket for its own slot (`ticket file` refuses the fleet), so this row is the only queue entry. **A (recommended):** authorize adding the literal `hit your session limit` emitter to `RUNTIME_QUOTA_SIG`, under D-FLEET-11's narrow scope (same tests, no threshold/order/rc change). The next fleet fire plans it. **B:** a ticket only (an attended session files it) and wait for more occurrences. **NO:** leave as is; Anthropic session stops keep reading as crashes. Default if unanswered: no change; the fleet proceeds with the heartbeat ticket. | OPEN (filed 2026-10-03, iteration 15; HD-2a; evidence `/tmp/ailang-mission-fleet.log` 2026-10-02 23:56:52 local) |
| D-FLEET-14 | RESOLVED | **ANSWERED — A (Mark Edmondson, attended 2026-10-08, recorded directly in this ledger). ALREADY DONE before the ruling:** `b7028a7cb` (#1583) pinned both tests to `ollamaPaceFixtureNow` (Wednesday 2026-09-30 12:00Z); nothing left for the fleet to do. **Weekend-only red in the required `test` check: may the fleet pin `now` in the two `TestOllamaQuota*` tests?** `TestOllamaQuotaVerifiedLimits` and `TestOllamaQuotaHTTPAndCredentialBinding` (`internal/mission/ollama_quota_test.go`) build limits from `time.Now()`. `WeekdayPacePercent` counts only weekday hours, and #1524 added a 3pp ollama start margin, so both tests fail from Saturday to about Monday 08:00Z. That blocks every PR to dev, including fleet #1578 and #1579. D-FLEET-12 reserves this code for attended work, so the fleet did not touch it. **A (recommended):** the fleet may pin `now` to a fixed weekday in those two tests only, e.g. `time.Date(2026, 9, 30, 12, 0, 0, 0, time.UTC)` (`pace_test.go` already pins `paceLocation` to UTC). Test-only, with no change to thresholds, margins or pacing. **B:** an attended session fixes it. **NO:** leave it; weekend PRs stay blocked. Default if unanswered: the fleet waits for the Monday clock and does not touch the tests. | OPEN (filed 2026-10-03, iteration 17; reproduced at origin/dev `2a1f3f295`, CI job 111289727641) |
| D-FLEET-15 | RESOLVED | **ANSWERED — A (Mark Edmondson, attended 2026-10-08, recorded directly in this ledger): the fleet may implement the ff-only auto-sync of the main checkout exactly as scoped below** (only on `dev`, 0 ahead, no rebase/merge in progress, no dirty file touched by the incoming range; otherwise log the skip and why; never reset or stash). Treat it as HD-2a-authorised; plan it as the next P1. **P1 #6 `skill-surface:main-checkout-not-synced-to-dev`: may the driver auto-sync the shared main checkout?** Loops read skills through `~/.claude/skills/mission-control` → `~/dev/sunholo-data/ailang` (the main checkout's working tree), and nothing fast-forwards it. It was 22 behind at iteration 21 and 0 behind / 1 ahead at iteration 22. **A (recommended):** at fire start the driver runs `git -C <main checkout> merge --ff-only origin/dev` ONLY when it is on `dev`, 0 commits ahead, has no rebase or merge in progress, and no dirty file is touched by the incoming range; otherwise it logs a skip with the reason. Git refuses rather than clobbers. Consequence: merged skill fixes reach every loop within one fire, at the cost of the loop writing to a tree attended sessions use (ff-only, never reset or stash). **B:** no auto-sync; #6 waits for Phase 3b per-fire skill pinning (gated on the 3a spike). Loops keep reading whatever the main checkout holds until an attended pull. Default if unanswered: B. #6 stays parked; the fleet takes P1 #8/#9, then the Phase 3a spike. | OPEN (filed 2026-10-06, iteration 22; Gate 1 reserves standing authorisation to reconcile the shared checkout to Mark; measured `git rev-list --left-right --count HEAD...origin/dev` = `1 0`) |
| D-FLEET-16 | OPEN | **Phase 3b skill source: which mechanism pins each controller's skills to the promoted SHA?** The Phase 3a spike (iteration 34, PASS 88, `fleet-mission-evidence/iteration34/phase3b-decision.md`) measured: claude user-level `~/.claude/skills` beats every project copy and `--add-dir`; a per-fire `CLAUDE_CONFIG_DIR` keeps auth (env token) and its `skills/` wins. Codex reads `.agents/skills` + `$CODEX_HOME/skills` (CODEX_HOME wins), never `.claude/skills`; a per-fire CODEX_HOME needs a credential copy. pi: explicit `--skill` + `/skill:` only. **A (recommended):** retire the user-level mission-skill symlinks so the cwd=pin project copy resolves; World gets `--add-dir <pin>`; codex reads the pin's `.agents` mirror under a parity check (#544 becomes a gate) with `$CODEX_HOME/skills` free of mission names; pi gets `--skill <abs pin path>` per skill. Cost: attended sessions outside a checkout lose these skills. **B:** per-fire config dirs (claude `CLAUDE_CONFIG_DIR`; codex needs a credential-copy ruling); every fire must replicate settings, hooks and memory. **C:** no change for skills; Phase 3b promotes driver + binary only, and skills stay on the D-FLEET-15 fast-forward. Default if unanswered: C. | OPEN (filed 2026-10-13, iteration 34; measurement, no mechanism selected) |
<!-- decision-ledger:end -->

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

**LANDED · D-FLEET-11 = A (iteration 15):** `driver:controller-fallback-skips-openrouter-ration-and-402-reads-as-crash` — #1549 `27dab5bb4`; anchored `^402:` capacity classification + pause-aware final notice; ration-admission half not reproduced at HEAD (no guard added). Ticket resolved.

**LANDED · `blocking=all` (iteration 25):** `agent-tool:mission-role-pins-unavailable` — #1635 `0ceb1db01`; the codex controller now gets the role env and the scope-guard hooksPath per variable (`tools/launchd/lib/codex-env-args.sh`). Ticket resolved. Follow-up candidate, not ticketed (may be routing policy): a bare-alias evaluator pin still resolves to `agent-tool <alias>` under a codex controller.

**P0 — whole slots lost** (the two unbuilt rulings, D-FLEET-1 and D-FLEET-2, landed in #1398)
1. [LANDED · iteration 19] `skill:heartbeat-relative-path-absent-in-world`: #1578 `c55ca4398`. All 18
   stamp calls (`.claude` + `.agents`) go through an absolute `MISSION_DRIVER_ROOT` guard. Ticket resolved.
2. [LANDED · iteration 19] `mission:rotate-log-registry-cwd` + `rotate-log:status-flag-mutates-and-world-resolves-to-status-archive`:
   #1580 `e7628b05e` (D-FLEET-9 = A), plus the Windows registry-walk termination fix `cffc0447a`.
   Both tickets resolved. Its sibling half `rotate-log:index-regen-drops-orphan-rows` (also reported at #1306)
   stays open and ranks by `slots_lost`.
3. [LANDED] `mission-base:hardcoded-origin-dev`: #1418 `cb7c51c8e` (iteration 9), ticket resolved.
**P1 — a lane misreported as dead, or a failure nobody sees** (the class that cost 2026-09-28/29's
overnight: harness faults read as model faults)
4. [LANDED] `pi-runner:quota-429-reported-as-empty-worktree`: #1424 `94524a6fc` (iteration 10), ticket
   resolved. Verdict `provider_quota` rc 19; the gate-3-route.md rc-list line is in scope of the D-FLEET-8 heartbeat fix (both copies, ruled 2026-10-01).
5. [LANDED · iteration 21] `pi-runner:verdict-blind-to-commits-and-predirty`, the **pre-dirty half**: #1593 → `c2bf04af3` (judged PASS 85 at iteration 20; re-judged PASS 100 by minimax at iteration 21; dev CI green after the incident). Ticket resolved. Non-blocking follow-ups from the judges, not ticketed: per-entry hash cost (≈9 ms per untracked file), `readlink --`, the stand-in has no test pinning it, an out-of-tree symlink is followed, and nested-repo untracked edits are unseen.
6. [LANDED · iteration 27] `skill-surface:main-checkout-not-synced-to-dev`: #1647 → `15d47b5dc` (D-FLEET-15 = A; judged PASS 91 by sonnet). `tools/launchd/lib/skill-sync.sh` fast-forwards the checkout the skill symlink resolves to, behind every guard in the ruling; the driver calls it after the kill switch and overlap yield (report on dry-run, apply before the boot stagger). Ticket resolved. Non-blocking follow-ups (judge): no arm pins `GIT_OPTIONAL_LOCKS=0`; the dry-run field prints `synced:N` for a would-sync verdict; the 5 s per-call cap also bounds the merge (0.66 s measured for 804 files, so it is load-sensitive). This feeds the Phase 3a skill-resolution directive below.
7. [LANDED · iteration 26] `gate0:driver-crash-notices-invisible`: #1604 → `59c3e6a55` (built and judged PASS 93 at iteration 22; re-judged PASS 96 on the merged head by minimax at iteration 26). `scripts/mission_gate0_self_notices.sh` + Gate 0 step 6a in both skill copies. Ticket resolved; #1160 closed. Step 6a's text reaches a running loop only once the main checkout is fast-forwarded (D-FLEET-15; 36 behind at iteration 26).
8. [LANDED · iteration 32] `quorum:zero-signal-guard-vacuous-with-controller-verdict`: #1767 → `a5f9af982` (judged PASS 95 by sonnet). `synthesize` counts present EXTERNAL reviewers only, so a controller-only quorum blocks; the controller keeps its veto. Designed iteration 28, quorum + revisions iteration 30, planned/built/judged iteration 32. Ticket resolved; ailang#651 closed. Reach: the PATH binary picks it up on its next rebuild or release.
9. [LANDED · iteration 33] `driver:exit-path-notices-unbounded`: #1791 → `e7dd6c127` (judged PASS 90 by sonnet). The refusal, late-kill and rc-fail notices go through `_mc_exit_notice`: bounded per channel, non-fatal and store-scoped. The slot-verdict send is store-scoped too. Ticket resolved. Follow-ups (planner), not ticketed: `driver:exit-notice-titles-undated`, `nightly-eval:unbounded-sends`.

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

**Weekly external-issue sweep 2026-10-05 (iteration 19; 0 new queue items).** 59 open issues enumerated (count checked against the listing); 57 had zero mentions in the charter, log, archive and dashboard. Controls: the positive one hit `#1380` 2 times, and the negative one fired. Most of the 57 are language, stdlib or product reports, which are V1's lane and outside this charter's Authority. The harness-shaped ones already have open fleet tickets, matched by title:
- `#981` → `gate0:watermark-advanced-by-dying-fire`
- `#1160` → `gate0:driver-crash-notices-invisible`
- `#941` → `quorum:invalid-absent-on-quoted-literals`
- `#581` → `planner-lane:files-section-parses-fenced-bullets`
- `#563` → `sprint-skill:estimated-loc-zero-is-placeholder-sentinel`
- `#476` → `skills:showcase-feature-discoverability`
- `#651` → P1 #8

`#1306` was half fixed by #1580 and got a verdict comment; it stays open for the orphan-row half.

**Landed (history)**
- `driver:exit-path-notices-unbounded`: #1791 `e7dd6c127` (iteration 33).
- `quorum:zero-signal-guard-vacuous-with-controller-verdict`: #1767 `a5f9af982` (iteration 32).
- `skill-surface:main-checkout-not-synced-to-dev`: #1647 `15d47b5dc` (iteration 27, D-FLEET-15 = A).
- `gate0:driver-crash-notices-invisible`: #1604 `59c3e6a55` (built iteration 22, landed iteration 26).
- `agent-tool:mission-role-pins-unavailable`: #1635 `0ceb1db01` (iteration 25).
- `skill:heartbeat-relative-path-absent-in-world`: #1578 `c55ca4398` (iteration 19, D-FLEET-8/10).
- `mission:rotate-log-registry-cwd` + `rotate-log:status-flag-mutates-and-world-resolves-to-status-archive`: #1580 `e7628b05e` (built iteration 18, landed iteration 19, D-FLEET-9 = A).
- `driver:controller-fallback-skips-openrouter-ration-and-402-reads-as-crash`: #1549 `27dab5bb4` (iteration 15, D-FLEET-11 = A).
- `pi-runner:quota-429-reported-as-empty-worktree`: #1424 `94524a6fc` (iteration 10).
- `mission-base:hardcoded-origin-dev`: #1418 `cb7c51c8e` (iteration 9).
- `stall-watchdog:kills-controller-on-long-drill`: #1399 `ce1c0639f`, D-FLEET-7 M2.
- `driver:slot-kill-leaves-orphan-descendants`: #1325.
- `pi-runner:sandbox-extensions-not-wired`: #1377.
- `rig:aqua-session-lost:windowserver-watchdog`: #1377.
- `tests:driver-env-leaks-into-launchd-suite`: #1330.
- `pi-runner:verdict-blind-to-commits-and-predirty`, commit-blind half: #1329.
- `pi-runner:verdict-blind-to-commits-and-predirty`, pre-dirty half: #1593 `c2bf04af3` (built iteration 20, landed iteration 21).
- `pi-runner:sandbox-blocks-mission-inbox-handshake`: #1393.
- The pi fallback rung set (attended 2026-09-29): #1391.
- `resolver:planner-lane-field-missing-vs-spawn-pin` (D-FLEET-1 built): #1398 `49f18bdbd`.
- `spawn-pin-hook:no-fallback-mode` (D-FLEET-2 built): #1398 `49f18bdbd`.
- `skills:agents-copies-stale`, fleet scope (mission-*/sprint-*): #1398 `49f18bdbd`. The
  `.agents`-only edits in model-manager and design-doc-creator still need a reconciliation
  (V1's scope).

**[LANDED · iteration 34] [DIRECTIVE, Mark 2026-09-26] After P1, before P2** (still stands after the 2026-09-29 re-rank; P1 #10 feeds it): the Phase 3a skill-resolution spike
(design doc M-HARNESS-MISSION-LOOP, premises P1–P3). Measure how the claude, codex and pi controllers
each resolve skills, using marker skills in the pin worktree against conflicting user-level copies.
**Measurement only**: record the results in the design doc's Verification Log and park Phase 3b's
mechanism choice for Mark. This is what unblocks `harness-stable`.

---
**Document created**: 2026-09-26. Iteration 0 ratifies it with Mark before any ticket routes.
