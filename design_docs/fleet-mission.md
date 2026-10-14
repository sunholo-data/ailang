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

## STATUS 2026-10-14 — ITERATION 39: P2 #15 slot verdict UNCLASSIFIED made reachable — built and judged PASS 91 by sonnet (round 2); stacked on #1825, BLOCKED on dev's required reds; Gate 1 skill fix for records stranded on blocked PR branches [HARNESS]

The driver's SLOT VERDICT `case` ended in an unconditional `*:*) CRASHED` arm. As a result, a slot that exited **0** with a heartbeat label outside the stamp writer's closed set was reported as crashed, and the design's `UNCLASSIFIED` verdict was unreachable (`grep -c` = 0). The fix has three parts:
- `REAPED` now matches only the writer's gate labels (`gate-0`..`gate-5`, `gate-3b`).
- Any other rc=0 label reads `UNCLASSIFIED at=<label>` and goes through the existing episode-gated phase-2 notice.
- Non-zero `CRASHED`/`KILLED` and the `PAUSED-NO-CAPACITY` override are unchanged.

Built by codex (plan `cf055fd1a`, fix `81df7d609`): 11 new heartbeat arms (6 red before the fix) and 10 planned mutants killed. The sonnet judge ran 10 drills of its own and scored **PASS 86**. One of its drills survived: an empty last label at rc=0 had no covering arm. The controller added that arm (`e9783fae6`; it is the drill's sole killer) and fixed the Q4 table wording. Round 2 scored **PASS 91**, 0 blocking.

Done-gate at `b081a39a4`:
- `make test-launchd-drivers` passed under `env -i` (controller-capacity 15/0, heartbeat 37 arms).
- World dry-runs: healthy `lanes=ok` and degraded `lanes=DEGRADED(1)`, both rc 0, with no pidfile written.
- Real heartbeat, episode and verdict files are sha-identical before and after.

Not LANDED: dev `474995629` is still red on `test` (`check-git-exec`, `devapi_workspace.go`) and on `docs-gate`, so this waits behind #1824 and #1825 (D-FLEET-17).

Gate 5 skill fix (`b081a39a4`, both copies): Gate 1 now reads the newest record from the mission's own open PR branches, because iterations 37, 38 and 39 each booted on a charter that was stale by up to three iterations. This iteration also re-quorum'd #12's already-built doc for about $0.10 before noticing.

Clause map: **1** UNMEASURED; **2** UNMET (no landing is possible until dev is green; #12, #14 and #15 are ready); **3** MET (50 open); **4** prior evidence only; **5** preserved (nothing shipped). Next: when D-FLEET-17's predicate holds, land #1824, then #1825, then this PR, re-running the done-gate at each merge, and resolve 4 tickets.

## STATUS 2026-10-14 — ITERATION 38: P2 #14 quorum artifact dir designed, quorum'd, built and judged PASS 90 by sonnet — stacked PR on #1824, BLOCKED on the same dev required reds [HARNESS]

`ailang design-quorum` wrote its artifacts to a CWD-relative `.ailang/state/mission-quorum`, so under pin and sprint worktrees a reviewed doc read as unreviewed at Gate 2 (measured: 5 artifacts in the fleet pin, 164 in the main checkout, 15 in the hand-used `~/.ailang/state/mission-quorum`). Now the driver exports a repo-scoped `MISSION_QUORUM_ARTIFACT_DIR` (registered in `internal/config/mission.go`), the quorum writer uses it when the flag is left at its default, and Gate 2's new executable reader block consults the export first, then `<repo-root>/.ailang/state/mission-quorum`. A raw policy marker keeps every default write inside the ailang_only sandbox, and a real policytool host/child test proves it. Out of scope: the legacy importer, and the CLI pre-flight (`cmd/ailang/design_quorum.go` is outside fleet authority).

How it was built: the codex designer (rotation turn `codex:gpt-6.1-sol`) wrote the doc. Quorum round 1 BLOCKED (glm, kimi). The designer revised it. Round 2: 2 pass and 1 reject (kimi). The controller applied kimi's verbatim `proposed_fix` under the narrow-refinement carve-out: a fixed-order two-directory reader, AC11, and A1 downgraded to 0. The codex planner corrected 9 premises. The codex executor built M1 (16/16 mutants killed) and M2; the sandbox refused the `.agents` mirror sync, so the controller ran it. The controller also scrubbed every inherited `MISSION_*` in the new shell suite: `MISSION_NAME` from a controller env had reddened an arm. The independent sonnet judge scored **PASS 90**, 0 blocking, and 9 of its own mutants went red. Non-blocking: a failed driver init exits 2 before `_mc_notify` exists, so it logs but does not notify.

Done-gate at `b151064db`: `make test-launchd-drivers` rc 0 under `env -i`. World dry-runs (`AILANG_DRIVER_PINNED` set): healthy `lanes=ok`, degraded `lanes=DEGRADED(1)`, both rc 0, and both print `quorum artifact directory: …/mission-quorum-world`. No pidfile was written. Not LANDED: dev `d15217d24` is still red on `test` (`check-git-exec`) and `docs-gate`, so #1824 and this stacked PR wait on D-FLEET-17. The ticket stays open.

Clause map: **1** UNMEASURED; **2** UNMET (no landing is possible until dev is green; #12 and #14 are both ready); **3** MET (51 open); **4** prior evidence only; **5** preserved (nothing shipped). Next: when D-FLEET-17's predicate holds, land #1824, then this PR, run the done-gate at each merge, and resolve 3 tickets. Then P2 #15.

## STATUS 2026-10-14 — ITERATION 37: no landing possible — #12 still BLOCKED on dev's required reds (owner V1 disabled); #13 reproduction measured (B gone, A + an untracked timer flake live in `tools/eval/`, outside fleet scope), judged PASS 88 by sonnet; D-FLEET-17/18 filed [HARNESS]

Dev HEAD `42652c19e` is unchanged since iteration 36, and the required `test` and `docs-gate` checks are still red:
- `check-git-exec` fails on `internal/executor/managed_agents/devapi_workspace.go:237,259`, from `ffef3402a`.
- `docs-gate` fails on an MDX `{"sort":"throughput"}` literal. It sits in `env-vars.md`, generated from `internal/config/ai.go:43`.
- `Build windows-latest` is also red in the same package.

Every fix is outside fleet scope. V1 owns repo reds, and its kill switch has been set since 09-14, so no PR to dev can merge. That includes PR #1824 (#12, judged PASS 88). Resume predicate: `test` and `docs-gate` green on origin/dev. **D-FLEET-17** asks how to unblock it.

With landing impossible, the iteration took P2 #13's mandated first step, reproduction, which is measurement only. The window was ≥ 09-28 13:08Z, covering 21 failed `launchd drivers` jobs with controls:
- **B** (gh hang, `tools/launchd`): last FAIL 09-28 13:15Z, before filing; 0 since.
- **A** (OpenRouter peer): last 10-02.
- An untracked flake, `bound measurement refuses an unsuccessfully terminated timer`, failed 4× on dev between 10-07 and 10-12.

A and the timer flake both live in `tools/eval/*motoko_connection_probe*`. The guard refuses that path for the fleet, yet it runs inside the fleet's own done-gate (`make test-launchd-drivers`), so **D-FLEET-18** asks about the authority. #13 stays OPEN. The independent sonnet judge re-measured all three claims with its own commands: **PASS 88**. Its corrections are adopted: `docs/` is in scope but generated, the Windows red is noted, and the 13:08–17:01Z slice is covered. Evidence: `fleet-mission-evidence/iteration37/findings.md`.

Clause map: **1** UNMEASURED; **2** UNMET (no landing is possible until dev is green; #12 is ready); **3** MET (51 open); **4** prior evidence only; **5** preserved (nothing shipped). Next: when D-FLEET-17's predicate holds, land #1824 and resolve both #12 tickets. Then P2 #14 (`quorum:artifact-dir-cwd-relative`), or #13 if D-FLEET-18 = A.

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
| D-FLEET-17 | OPEN | **Dev's REQUIRED checks have been red since 2026-10-13 20:02Z, from causes outside fleet scope, and the owner (V1) has been disabled since 09-14. No PR to dev can merge, including fleet #1824 (judged PASS 88). Who fixes it?** Causes, measured by the controller and the judge: (1) `test`: `make check-git-exec` refuses 2 bare `git` exec sites, `internal/executor/managed_agents/devapi_workspace.go:237,259` (`ffef3402a`). (2) `docs-gate`/`docs-build`: an MDX/acorn error at `docs/docs/reference/env-vars.md:252`, the `'{"sort":"throughput"}'` example, generated from `internal/config/ai.go:43`. (3) Not required: `Build windows-latest` fails 2 tests in the same package. **A (recommended):** an attended session fixes (1) and (2) now. Route both git calls through `internal/gitexec` (or add a `scripts/git_exec_baseline.txt` row), and escape or reword the brace example in `ai.go`, then regenerate. Each is a small diff. **B:** grant the fleet a narrow standing authority to fix a REQUIRED-check red on dev when the owning mission's kill switch is set. Such a fix ships alone, is judged by an evaluator, and passes the done-gate. **C:** no change; the fleet keeps building and parks every landing. Default if unanswered: C. | OPEN (filed 2026-10-14, iteration 37; `fleet-mission-evidence/iteration37/findings.md` §1) |
| D-FLEET-18 | OPEN | **#13's live launchd-suite flakes sit in `tools/eval/`, which the scope guard refuses for the fleet, yet `make test-launchd-drivers` (the fleet's own done-gate, `make/test.mk:117`) runs them. Widen the authority?** Measured since 2026-09-28 13:08Z: signature A (OpenRouter peer, `tools/eval/motoko_connection_probe.sh:79`) last failed 10-02. The untracked `bound measurement refuses an unsuccessfully terminated timer` (`tools/eval/test_motoko_connection_probe.sh:1136`) failed 4× on dev, 10-07 to 10-12. Signature B (`tools/launchd`) has not failed since filing. The owning motoko loop has been disabled since 09-08. **A (recommended):** add exactly `tools/eval/motoko_connection_probe.sh` and `tools/eval/test_motoko_connection_probe.sh` to the fleet allowlist. The guard arm ships alone first (iteration-16 rule), then #13 is worked on both live signatures. **B:** route both flakes to motoko or an attended session; #13 stays parked. **C:** no change; #13 stays open. Default if unanswered: C. | OPEN (filed 2026-10-14, iteration 37; `fleet-mission-evidence/iteration37/findings.md` §2) |
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
10. [LANDED · iteration 35] `skill:gate0-ledger-provenance-S`: #1815 → `130486515` (judged PASS 92 by sonnet, round 2). Gate 0 rule 6(b) runs `scripts/mission_ledger_provenance.sh`: latest OPEN→RESOLVED transition, creation control, squash-merged flips `undetermined` (never re-opened). Ticket resolved. Follow-ups (judge), not ticketed: bold `**RESOLVED**` cells read as never-resolved (N2); the `Attended ruling` stamp is not checked (N4).
11. [LANDED · iteration 36, ghost] `weekly-report:unknown-mission`: already fixed by `08ce76f30` (attended,
    2026-10-06; the script reads `missions/*.toml`). Re-verified for all 6 missions, judged PASS (sonnet). Ticket resolved.
12. [IN-SPRINT · iteration 36 · PR #1824, judged PASS 88, BLOCKED on inherited dev red; re-checked iteration 37, still red; D-FLEET-17] `quorum:invalid-absent-on-quoted-literals`
    + `quorum:reviewer-invalid-json-discarded` (#941): redesigned (`m-quorum-salvage-retry.md`, quorum + carve-out), built and
    judged. Merge when `test` and `docs-gate` are green on origin/dev, then run the done-gate at the merge and resolve both.
13. [MEASURED · iteration 37 · D-FLEET-18] `ci:launchd-suite-flakes-openrouter-peer-and-gh-hang` (filed 2026-09-29). Reproduction
    measured: B (gh hang, `tools/launchd`) is gone (last 09-28 13:15Z). A (OpenRouter peer) last failed 10-02, and an untracked timer flake failed 4× on dev between 10-07 and 10-12. Both live ones are in `tools/eval/`, outside fleet scope. Ticket stays open.
14. [IN-SPRINT · iteration 38 · stacked PR on #1824, judged PASS 90, BLOCKED on the same dev reds; D-FLEET-17] `quorum:artifact-dir-cwd-relative`:
    designed (`m-quorum-artifact-dir-stable.md`, quorum r1 BLOCKED, r2 2/3 + carve-out), planned, built and judged. The driver exports a repo-scoped
    `MISSION_QUORUM_ARTIFACT_DIR`, and Gate 2 reads export then repo-root. Merge after #1824, done-gate at the merge, then resolve.
    Follow-ups (not ticketed): legacy importer; CLI pre-flight in `design_quorum.go` (outside scope); driver init exit 2 is not notified (judge N1).
15. [IN-SPRINT · iteration 39 · stacked PR on #1825, judged PASS 91 (round 2), BLOCKED on the same dev reds; D-FLEET-17] `driver:unclassified-state-unreachable`:
    rc=0 with an unknown heartbeat label is now `UNCLASSIFIED at=<label>` (notified); REAPED matches only the writer's gate labels.
    Merge after #1825, done-gate at the merge, then resolve.
16. `quorum:tier2-salvage-fold` (charter-tracked follow-up of #12; the fleet cannot file tickets, so this row is
    its record — landing precondition of M-QUORUM-SALVAGE AC10): `SynthesizeWithTier2` still drops a salvaged
    Tier-2 reject (designer V19 probe at `42652c19e`: `proceed`). Not live: `RunQuorumWithEscalation` has zero
    non-test callers. Trigger: its first non-test caller. Fix: route the Tier-2 absent branch through `foldAbsent`
    and wire `recordInvalid` in `agentic_caller.go` (~3 lines plus a test).

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
- `weekly-report:unknown-mission`: already fixed by `08ce76f30`; resolved iteration 36 (ghost).
- `skill:gate0-ledger-provenance-S`: #1815 `130486515` (iteration 35).
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
