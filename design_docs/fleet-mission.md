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
**Human-facing reporting**: GitHub issue #1584 (rotated 2026-10-05 from #1380; live number in
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
- **Bookkeeping issue**: `#1584` (from `#1380`), rotates weekly; live number in `~/.ailang/state/mission-fleet-gh-issue`
- **CI workflows Gate 3b / Gate 1 poll**: `CI` (runs on every push; no push paths filter).
- **Verify profile**: `go-compiler`, plus the mission-loop-change pre-flight as the done-gate
  (below). Harness changes live in `tools/launchd/` (bash 3.2), `internal/mission/` and
  `cmd/ailang/mission*` (Go), and the mission skills.

---

## STATUS (rotation rule)

Newest **3** STATUS stamps live here; older ones move to `fleet-mission-status-archive.md`.

## STATUS 2026-10-06 — ITERATION 23: P1 #8 quorum guard verified live at HEAD but ALL role lanes over ration (ollama hard-capped mid-fire by the controller's own session) → PARKED-ON-LANE; zero roles spawned, nothing landed [HARNESS]

Gate 2 took P1 #8 `quorum:zero-signal-guard-vacuous-with-controller-verdict` (ailang#651): P1 #6 is parked on D-FLEET-15 and P1 #7's landing is blocked on V1's dev red, so it was the top routable row, and its premise was re-verified first-party at `04dc2b7c8` — `internal/mission/quorum/quorum.go:170` increments `presentCount` for the controller verdict before the zero-signal guard at `:176`, so a controller-only quorum reads `proceed` with zero external reviewers (measured on the 3 artifacts named in #651). Gate 3 could not route it: the resolver returned `recipe pi:ollama/...` for designer/planner/executor and `refuse over-ration:anthropic` for the evaluator; `mission-lane-check.sh fleet` reported every rung of every role `skip:over ration` (anthropic ENFORCED headroom 0.6pp < 1pp start margin, codex 0.4pp < 2pp, openrouter $3.59 of $2.33/day, ollama 0.8pp < 3pp); and this session exposes no Agent/Task tool, so no alias-lane path existed either. Mid-fire the ollama bucket crossed its hard cap — 6.1pp → 10.1pp of the 10pp/day ration ("new cloud routing blocked"), consumed by the controller's own pi-rung session, the only ollama consumer after stapledon's fire ended 18:47Z — so a pi-rung controller structurally starves the role lanes that share its bucket (1 instance; a routing-policy signal if it recurs). The evaluator is REQUIRED (generator ≠ judge; the operator's standing request for this run), it cannot run on any lane, so nothing may land: P1 #8 is parked PARKED-ON-LANE with the resume predicate "`mission-lane-check.sh fleet` rungs ok + a judge lane ≠ the executor model", and #1604/#1605's landing predicates were re-measured as commands, not transcribed: dev's required `test` check is still red at `04dc2b7c8` on `TestValidateModulePath_SingleFileInsidePackage` (V1's attended lane, handed over at iteration 22). Re-measured again at the END of the fire, after dev advanced to `c57d4f00e` (M-TEST-RUNNER Phase 2 + release v0.52.2): the `test` job still fails the same test (run 37515762996, 19:18Z), so the park stands on the newest head too. This record carries iteration 22's entry; its record PR #1605 is superseded by this PR and closed. Clause map: **1** UNMEASURED; **2** UNMET (0 resolved this fire); **3** MET (39 open signatures); **4** prior evidence only; **5** preserved trivially (nothing merged, nothing judged).

## STATUS 2026-10-06 — ITERATION 22: Gate-0 self-notice read BUILT + judged PASS 87→91→93 as PR #1604; merge blocked by an inherited dev red (07e1a89bc, V1's lane) → PARKED-ON-CLOCK; P1 #6 parked as D-FLEET-15

Gate 2 skipped Mark's queue head P1 #6 `skill-surface:main-checkout-not-synced-to-dev` for a measured reason. Its fix is either a driver auto fast-forward of the shared main checkout, whose standing authorisation is reserved to Mark (gate-1-observe.md), or Phase 3b skill pinning, which the 2026-09-26 directive gates on the 3a spike and parks for Mark. So it is filed as **D-FLEET-15**; the main checkout was 0 behind / 1 ahead at this fire. Picked P1 #7 `gate0:driver-crash-notices-invisible` (ailang#1160): a port of World's judged `gate0_self_notices.sh` (ailang-world#137) into the shared harness. Planner: opus (Agent), plan `6934d2a0b`. Executor: `claude:claude-sonnet-5-5` (`claude-sub`). Its first run died after 227 s on an Anthropic `Server error mid-response`; every remaining chain lane was over ration, so the same lane was retried and finished M1–M4. Judge: opus (Agent, own worktree). The resolver rerouted it to `pi:openrouter/minimax/minimax-m3`, but openrouter was over ration ($3.09 against $2.33/day), and `claude-sonnet-4-6` is the executor's family; both were recorded dead, and opus is the declared bare fallback. Round 1 **PASS 87** (m3, prev-issue read dropped, survived). Round 2 `949d5952b` **PASS 91**: new arm kills m3, 6a block runnable from a fresh shell. Round 3 `2e0f92672` **PASS 93**: a precondition failure now prints rc=2, never the fire-died token. Suite 108/0; `make test-launchd-drivers` rc 0 (unpiped, executor and judge); fixtures `cmp`-identical to World's; the two skill copies `cmp`-identical; scope guard allows all 14 paths. Live reading from `/tmp`: rc 1, control 852:10 ok. It found the real rc=143 kill of iteration 18 (2026-10-04T04:28Z), whose Gate-4 record had already landed. PR #1604 `test` and `test-windows` are red on `TestValidateModulePath_SingleFileInsidePackage`, the same test that fails on dev `07e1a89bc` (attended `internal/pipeline` fix, 12:07Z). It is inherited, outside Authority, and was handed to V1 (`inbox_1791290588115`). Resume predicate: dev `CI` green → re-run #1604's failed jobs on `2e0f92672` → required green → merge `--match-head-commit` → resolve the ticket. Clause map: **1** UNMEASURED; **2** UNMET (0 resolved this fire); **3** MET (38 open signatures); **4** prior evidence only; **5** preserved (nothing merged).

## STATUS 2026-10-06 — ITERATION 21: pi-runner pre-dirty fix LANDED — #1593 `c2bf04af3` after the Actions incident cleared, re-judged PASS 100 (minimax); ticket resolved (39 → 38)

Resume of iteration 20's PARKED-ON-CLOCK item. Predicate met: githubstatus showed Actions `operational`, and the 19:11Z–22:49Z incident was resolved. On #1593 head `2e49cc9ac` the required `test`/`lint`/`build`/`docs-gate` were green, plus `launchd drivers` and `test-windows`. The one red, `UI build gate`, had 0 steps and no runner (an outage casualty); it went green on re-run (attempt 2, post-incident). Base unchanged: origin/dev `a12a319b5`. Because every PR green had run during the incident (Gate 3b: code evidence only), the fix was re-judged before merge. The resolver said `reroute pi:openrouter/minimax/minimax-m3 generator-equals-judge`, and openrouter was within ration this fire. The judge ran in an isolated worktree with the handshake acked and returned **PASS 100**, 0 blocking. Evidence: 36/36 tests; pi_run suites 9/13/29; mutations M-a (porcelain-only `ok`), M-b (no untracked hashes) and M-c (post fp = pre fp) all red, each restored byte-identical by `cmp`; a hand repro where the old code gives `ok` and the fix gives rc 10 when nothing changes and `ok` after a real edit. Squash-merged `--match-head-commit` → `c2bf04af3`. Dev `CI` **success** on `c2bf04af3`, after the incident, so it counts as the infrastructure green. Ticket `pi-runner:verdict-blind-to-commits-and-predirty` resolved; world replied. Record #1594 (iteration 20) merged → `eb2850427`. Other reds seen and handed to V1, none from this diff: Cloud Build `ailang-core-dev` fails every commit on the node pin (NodeSource now ships `v22.23.3`, pinned `v22.23.2` in `docker/Dockerfile.agent-base`, outside fleet Authority); `test-windows` hang guard on docs-only `eb2850427` (second instance, after `d273ea1cd`); `Build macos-latest` `TestMemprobeDebugLogDoesNotAccumulate` (known flake); Sonar (inherited). Done-gate: surface ✓ (`scripts/mission_pi_run.sh`, which every pi lane runs); reach ✓ (on origin/dev); `make test-launchd-drivers` ✓ (CI `launchd drivers (bash 3.2)` green on `c2bf04af3`); 0 driver bytes, so no dry-run; nothing reloaded ✓. Clause map: **1** UNMEASURED; **2** UNMET (this ticket ≈9.6 days filed→resolved); **3** MET (38 open signatures); **4** prior evidence only; **5** preserved (judged twice, CI green on the merge SHA).

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
| D-FLEET-13 | OPEN | **Anthropic "hit your session limit" is recorded as CRASHED: authorize the same narrow fix as D-FLEET-11?** Iteration 15 attempt 1 (fire 2026-10-02T21:39Z) died with `You've hit your session limit · resets 12:40am` → `CRASHED at=gate-3 rc=1`. `RUNTIME_QUOTA_SIG` matches "hit your usage limit" but not "hit your session limit", so a capacity stop skips demote/re-walk/PAUSED. The fleet cannot file a ticket for its own slot (`ticket file` refuses the fleet), so this row is the only queue entry. **A (recommended):** authorize adding the literal `hit your session limit` emitter to `RUNTIME_QUOTA_SIG`, under D-FLEET-11's narrow scope (same tests, no threshold/order/rc change). The next fleet fire plans it. **B:** a ticket only (an attended session files it) and wait for more occurrences. **NO:** leave as is; Anthropic session stops keep reading as crashes. Default if unanswered: no change; the fleet proceeds with the heartbeat ticket. | OPEN (filed 2026-10-03, iteration 15; HD-2a; evidence `/tmp/ailang-mission-fleet.log` 2026-10-02 23:56:52 local) |
| D-FLEET-14 | OPEN | **Weekend-only red in the required `test` check: may the fleet pin `now` in the two `TestOllamaQuota*` tests?** `TestOllamaQuotaVerifiedLimits` and `TestOllamaQuotaHTTPAndCredentialBinding` (`internal/mission/ollama_quota_test.go`) build limits from `time.Now()`. `WeekdayPacePercent` counts only weekday hours, and #1524 added a 3pp ollama start margin, so both tests fail from Saturday to about Monday 08:00Z. That blocks every PR to dev, including fleet #1578 and #1579. D-FLEET-12 reserves this code for attended work, so the fleet did not touch it. **A (recommended):** the fleet may pin `now` to a fixed weekday in those two tests only, e.g. `time.Date(2026, 9, 30, 12, 0, 0, 0, time.UTC)` (`pace_test.go` already pins `paceLocation` to UTC). Test-only, with no change to thresholds, margins or pacing. **B:** an attended session fixes it. **NO:** leave it; weekend PRs stay blocked. Default if unanswered: the fleet waits for the Monday clock and does not touch the tests. | OPEN (filed 2026-10-03, iteration 17; reproduced at origin/dev `2a1f3f295`, CI job 111289727641) |
| D-FLEET-15 | OPEN | **P1 #6 `skill-surface:main-checkout-not-synced-to-dev`: may the driver auto-sync the shared main checkout?** Loops read skills through `~/.claude/skills/mission-control` → `~/dev/sunholo-data/ailang` (the main checkout's working tree), and nothing fast-forwards it. It was 22 behind at iteration 21 and 0 behind / 1 ahead at iteration 22. **A (recommended):** at fire start the driver runs `git -C <main checkout> merge --ff-only origin/dev` ONLY when it is on `dev`, 0 commits ahead, has no rebase or merge in progress, and no dirty file is touched by the incoming range; otherwise it logs a skip with the reason. Git refuses rather than clobbers. Consequence: merged skill fixes reach every loop within one fire, at the cost of the loop writing to a tree attended sessions use (ff-only, never reset or stash). **B:** no auto-sync; #6 waits for Phase 3b per-fire skill pinning (gated on the 3a spike). Loops keep reading whatever the main checkout holds until an attended pull. Default if unanswered: B. #6 stays parked; the fleet takes P1 #8/#9, then the Phase 3a spike. | OPEN (filed 2026-10-06, iteration 22; Gate 1 reserves standing authorisation to reconcile the shared checkout to Mark; measured `git rev-list --left-right --count HEAD...origin/dev` = `1 0`) |
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
6. [PARKED · D-FLEET-15, iteration 22] `skill-surface:main-checkout-not-synced-to-dev` (filed 2026-09-29): loops read skills from the
    main checkout, which nothing fast-forwards. Merged skill fixes wait for a manual pull, and a paused
    rebase can freeze them. This feeds the Phase 3a skill-resolution directive below.
7. [IN-SPRINT · iteration 22, PR #1604 judged PASS 93; PARKED-ON-CLOCK on inherited dev red — re-measured at `04dc2b7c8` AND at `c57d4f00e` (19:18Z, iteration 23): `test` still fails `TestValidateModulePath_SingleFileInsidePackage`, V1's lane] `gate0:driver-crash-notices-invisible`: `gate-0-preflight.md` still reads directives but no slot
    verdicts or rc=143 notices. The World-local fix (row 73) was never ported.
8. [PARKED-ON-LANE · iteration 23: all four provider buckets over ration (resolver: designer/planner/executor `recipe pi:ollama/…`, evaluator `refuse over-ration:anthropic`); premise re-verified live at `04dc2b7c8` (quorum.go:170 before the guard at :176); resume = `mission-lane-check.sh fleet` rungs ok + a judge lane ≠ the executor model] `quorum:zero-signal-guard-vacuous-with-controller-verdict`: `quorum.go:164-165` counts the
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

**[DIRECTIVE, Mark 2026-09-26] After P1, before P2** (still stands after the 2026-09-29 re-rank; P1 #10 feeds it): the Phase 3a skill-resolution spike
(design doc M-HARNESS-MISSION-LOOP, premises P1–P3). Measure how the claude, codex and pi controllers
each resolve skills, using marker skills in the pin worktree against conflicting user-level copies.
**Measurement only**: record the results in the design doc's Verification Log and park Phase 3b's
mechanism choice for Mark. This is what unblocks `harness-stable`.

---
**Document created**: 2026-09-26. Iteration 0 ratifies it with Mark before any ticket routes.
