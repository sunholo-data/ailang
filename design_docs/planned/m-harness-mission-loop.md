# M-HARNESS-MISSION-LOOP: A Fleet Loop That Owns the Loop Harness, So Product Loops Don't

**Status**: Planned. Quorum-blocked twice (all six objections correct; five fixed in-doc, one needs the Phase 3a spike). Design-freeze items HD-1 to HD-6 open for Mark. Phases 1–2 can be ratified independently of Phase 3.
**Created**: 2026-09-26 (attended, Mark: *"would it be better instead to have a dedicated loop to its own harness?"*)
**Target**: v0.44.0
**Priority**: P1
**Estimated**: 6.5 days across three phases, each shippable on its own (P1 1.5d, P2 2d, P3a 0.5d + P3b 2.5d)
**Dependencies**: None blocking. Reuses the driver pin (`tools/launchd/lib/pin-root.sh`), the mission
registry (`missions/*.toml`, [M-MISSION-LOOP-WORKBENCH](v0_36_0/m-mission-loop-workbench.md)) and the
message plane.
**Quorum trigger**: #1 fired (design-freeze items). #2 fired too: this overrides the Gate-2
admissibility rule in `.claude/skills/mission-control/resources/gate-2-pick.md`.

> **Terminology.** In this doc, *loop harness* means the mission machinery: the driver, the pin,
> `internal/mission`, the mission skills, the pi extensions, `missions/*.toml`, the plists and the
> mission env files. This is **not** PROGRAM.md's *harness* (motoko, pi and opencode as eval
> subjects). The new mission is named `fleet` (HD-5) so the two never share a word.

## Axiom Compliance

**Canonical reference:** [Design Axioms](/docs/references/axioms)

### Axiom Scoring

| Axiom | Score | Justification |
|-------|-------|---------------|
| A1: Determinism | +1 | Product loops run a promoted harness (driver, skills **and** `ailang` binary, all from one SHA) instead of `origin/dev`, the live working tree and whatever binary was last installed (V11, V12, V16). The skills part is conditional on the Phase 3a spike (P1–P3); until it passes, this is +1 for driver and binary only. |
| A2: Replayability | 0 | No trace format change. The ticket records the fire it came from, which helps replay but adds no new mechanism. |
| A3: Effect Legibility | 0 | No hidden effects added. Each loop's write scope becomes declared (A4). |
| A4: Explicit Authority | +1 | Today the product-vs-harness boundary is a prose rule the loop judges for itself (V3). It becomes a declared path scope per mission, checked mechanically. |
| A5: Bounded Verification | 0 | No change. |
| A6: Safe Concurrency | 0 | A fifth loop runs in its own clone, like docs and motoko (V8). The existing memory gate and overlap guard apply unchanged. |
| A7: Machines First | 0 | Tickets are structured (typed fields and a signature for dedupe), not prose. Neutral overall. |
| A8: Minimal Syntax | 0 | No language change. |
| A9: Cost Visibility | +1 | The cost of harness work becomes a separate, measured line (fleet-loop spend per fixed ticket) instead of hiding inside product iterations. |
| A10: Composability | 0 | Reuses the pin, the registry and the message plane. Adds no parallel mechanism. |
| A11: Structured Failure | +1 | Harness defects travel as typed tickets with a blocking flag and a resolution reply, instead of free text in four different charters (V4). |
| A12: System Boundary | +1 | The product/harness boundary becomes explicit and is enforced where a push crosses it. |

**Net Score: +6** → **Decision: Move forward**

### Hard Violation Check

- [x] A1 (Determinism): no implicit nondeterminism. This removes some (the live working tree as skill source).
- [x] A3 (Effects): no hidden side effects.
- [x] A4 (Authority): no ambient access granted. The fleet loop's scope is narrower than v1's is today.
- [x] A7 (Machines First): tickets are machine-readable first.

## Problem Statement

Mission loops drift into fixing their own plumbing, and it dilutes real progress. The fix so far
has been a prose rule plus attended maintenance sessions. That rule works while someone runs the
sessions, but it stalls the loops while they wait.

**Current state (all measured; see the Verification Log):**

- **V1 loop, iterations 309–348:** 18 of the 35 recorded entries are tagged `[HARNESS]` (V2). During
  2026-08-25 to 09-08 the loops touched 883 lines of mission docs and 16 lines of compiler and
  stdlib (gate-2-pick.md, V3).
- **World loop, iterations 164–180:** 12 of 13 recorded entries are `[HARNESS]`. Iterations 181–194
  are 10 of 10 product rows, after the loops were paused and the debt cleared in attended sessions
  (V1). World merged 8 product PRs between 09-24 and 09-26 (V1).
- **The current rule** (Mark, 2026-09-21) makes harness work attended-only. It tells a blocked loop
  to file a `[HARNESS]` row in its own queue and route shared defects "upstream as an issue" (V3).
  In practice those escalations are scattered: **zero** open queue rows are tagged `[HARNESS]`
  across the v1, world and docs charters (V4). World's escalations arrive as `mission-world`
  messages that are filed for a human and never dispatched (V5). No inbox exists for harness work
  (V6).
- **The rule is judgement-only.** Nothing mechanical stops a product loop that works in the ailang
  repo (v1, docs, motoko) from editing harness paths (V7). The 09-21 note itself says the cap has to
  sit *outside* the judgement it constrains, and today it does not.
- **Every product loop runs the live harness.** The driver pins to `origin/dev` by default (V11), so
  any harness commit reaches every loop on its next fire. The skills are worse: `~/.claude/skills/
  mission-control` and `sprint-*` are symlinks into the **main checkout's working tree** (V12).
  Uncommitted edits in that checkout become the controller's instructions for every mission.

**Impact:** product throughput depends on how often someone runs an attended harness session.
Harness regressions land in all loops at once. There is no measure of how much product time the
harness is costing.

## Goals

**Primary goal:** product loops spend their iterations on product work. Harness defects they hit
leave the loop as tickets and come back as fixes, without an attended session in the path for
routine repairs.

**Success metrics:**
1. `[HARNESS]`-tagged share of product-loop iterations is ≤ 10% over a rolling 14 days, measured
   from the mission logs with the V1/V2 method (baseline: v1 51%, world 92% in the windows above).
2. Median time from ticket filed to fix promoted is ≤ 48h for non-policy tickets.
3. Every product-loop escalation is findable in one place. A query against the `mission-fleet`
   inbox returns 100% of tickets filed after Phase 1. Verified by a planted ticket from each mission.
4. A harness regression reaches product loops only after promotion, for **all three** seams. Proof:
   three planted broken commits on `dev` (one in the driver, one in a mission skill, one in
   `internal/mission`) never run in a product fire, while the fleet fire does run them (Phase 3b
   acceptance).
5. The fleet loop costs nothing when idle. No open work means zero controller tokens (a
   driver-level pre-check exits before any spawn).

## High-Impact Decisions

| # | Decision | Options | Recommendation | Who decides | Change cost later |
|---|----------|---------|----------------|-------------|-------------------|
| HD-1 | Where the loop-harness code lives | (a) stays in `sunholo-data/ailang`, with a separate lane (mission, scope, release branch) · (b) new repo `sunholo-data/ailang-fleet` · (c) inside ailang-world | **(a).** `internal/mission` ships in the `ailang` binary (V10). World's forked driver silently missed every fleet fix until the de-fork (`missions/world.toml` header). A repo split rebuilds that problem. Revisit only if a consumer outside AILANG (Daneel, aitana) needs the harness as a dependency. | Mark | High. A repo split is weeks of path and CI work. |
| HD-2 | Fleet-loop autonomy | (a) mechanical fixes land on `dev` autonomously; routing, quota, ration and billing-guard changes park in the decision ledger for Mark · (b) every fleet change needs Mark's approval · (c) fully autonomous | **(a).** Routing and quota policy is where a wrong call spends money or starves a lane (today's opus-before-pi change was a policy call). Mechanical fixes (a log-order bug, a probe timeout) are not. | Mark | Low. It's a skill rule. |
| HD-3 | Release channel for product loops | (a) product loops pin `origin/harness-stable`; the fleet loop runs `origin/dev` as the canary and promotes · (b) everyone stays on `origin/dev` (today) | **(a).** Makes goal 4 true. Uses a **branch**, not a tag (V13). | Mark | Low. One env var per mission. |
| HD-4 | Enforcing the product-loop scope | (a) mechanical: a pre-push path guard keyed on `MISSION_NAME` refuses harness paths unless `MISSION_NAME=fleet`; attended sessions (no `MISSION_NAME`) are unaffected · (b) rule-only (today) | **(a).** The 09-21 rule already concluded the cap has to sit outside the loop's judgement. | Mark | Low. |
| HD-6 | What the pinned `ailang` binary covers | (a) product fires put `$PIN_DIR/bin/ailang`, built from `harness-stable`, first on PATH, so harness commands **and** language use share one promoted binary · (b) a separately named harness binary (`ailang-fleet`) for harness commands only, leaving the language binary on `~/go/bin` | **(a).** Simplest, and it is what closes goal 4. Cost: world's language use lags `dev` by at most one promotion interval (≤48h target). v1 and docs build their own binary in their worktree for product tests anyway. (b) doubles the install surface. | Mark | Low. One PATH line. |
| HD-5 | Mission name | `fleet` · `harness` · `loopworks` | **`fleet`**. World's charter already calls the shared machinery "the fleet" (D-WORLD-DRIVER-1), and `harness` collides with PROGRAM.md. | Mark | Low before first fire, medium after (state paths are namespaced by name). |

### Design Freeze

- [ ] HD-1 ratified
- [ ] HD-2 ratified
- [ ] HD-3 ratified
- [ ] HD-4 ratified
- [ ] HD-5 ratified
- [ ] HD-6 ratified

## Solution Design

### Overview

Four parts, each reusing something that already exists:

1. **Intake.** A `mission-fleet` inbox plus a typed ticket. Product loops file a ticket instead of
   fixing, then continue with the next product item (or yield when fully blocked).
2. **The fleet loop.** A fifth mission (`missions/fleet.toml`) in its own clone. Its queue is the
   inbox, ranked by product slots lost. It edits only harness paths.
3. **Scope guard.** Product loops cannot push changes to harness paths. The fleet loop cannot push
   changes to product paths.
4. **Release channel.** Product loops run `origin/harness-stable` for all three seams: the driver
   (via the existing pin), the skills (mechanism chosen by the Phase 3a spike) and the `ailang`
   binary (built from the promoted SHA). The fleet loop runs `dev` first and promotes after its own
   fires survive.

### Architecture

```
product loop (v1 / docs / motoko / world)
  │  hits a harness defect at any gate
  │  1. file ticket ──────────────► mission-fleet inbox  (message plane, prod Firestore)
  │  2. record [HARNESS] row with ticket id in its own charter
  │  3. next product item, or yield if blocking=all
  ▼
fleet loop (MISSION_NAME=fleet, clone ~/dev/sunholo-data/ailang-fleet, runs origin/dev)
  │  pre-check: OPEN work == 0 → exit 0, no controller spawned
  │    (open = tickets with no resolution reply, plus any candidate SHA awaiting promotion)
  │  Gate 0: read tickets, dedupe by signature, rank by slots_lost
  │  Gate 2: pick top ticket; policy class → park in decision ledger (HD-2)
  │  Gates 3–4: design → plan → execute → evaluate (harness paths only)
  │  land on dev → own next fire runs it (canary)
  │  promote: fast-forward origin/harness-stable when promotion gate passes
  │  reply on ticket: {resolution, harness_stable_sha}
  ▼
product loop, next fire: pin resolves origin/harness-stable → fix present
  Gate 0 reads the reply, unparks the [HARNESS] row
```

**Ticket schema** (message body JSON, `--type harness-friction`):

| Field | Meaning |
|---|---|
| `mission`, `iteration`, `fire_started` | Where it happened |
| `signature` | Stable dedupe key, e.g. `stall:gate-3:pi-openrouter-executor` |
| `slot_verdict` | Copied from `mission-<name>-slot-verdicts.log` |
| `slots_lost` | Fires killed or degraded by this defect so far (the loop increments it on refile) |
| `blocking` | `none` \| `item` \| `all` |
| `evidence` | Log lines (bounded to 2 KB) plus paths |
| `workaround` | The one reversible local workaround applied, or `none` |

**Fleet scope (write allowlist):** `tools/launchd/**`, `internal/mission/**`, `cmd/ailang/mission*`,
`missions/**`, `.claude/skills/mission-*/**`, `.claude/skills/sprint-*/**`, `.pi/extensions/**`,
`scripts/hooks/**`, `scripts/mission_*`, `design_docs/fleet-mission*.md`. Product loops get the
complement. The exact list is agent-resolvable at planning (Deferred Decisions).

**The fleet loop does not look for work.** Its queue is tickets plus Mark's directives, never
self-sourced audits. The 09-21 note measured that an attended hunt for harness defects "never ran
out" (V3). A loop that sources its own work would become the new sink.

### Implementation Plan

**Phase 1: intake and scope (1.5d). Useful with no new loop: attended sessions consume the inbox.**
- `mission-fleet` inbox registered as a TRIAGE (non-dispatched) inbox.
- `ailang mission ticket` subcommand: validates the schema, computes the signature, refiles
  increment `slots_lost` instead of duplicating.
- Rewrite `gate-2-pick.md` steps 2 and 5: file a ticket, record the ticket id in the `[HARNESS]`
  row, and route here instead of opening an ad-hoc issue.
- Pre-push path guard (HD-4) in `scripts/hooks/`, wired for the claude and pi controllers
  (`.pi/extensions/prepush-gate.ts`), keyed on `MISSION_NAME`.
- Acceptance: a planted ticket from each of v1, docs, motoko and world lands in the inbox. A product
  push that touches `tools/launchd/` is refused with an error naming the fleet inbox.

**Phase 2: the fleet loop (2d).**
- `missions/fleet.toml` (interval schedule, 6h) plus a clone at `~/dev/sunholo-data/ailang-fleet`,
  installed with `ailang mission install fleet`.
- Driver pre-check: when `MISSION_NAME=fleet` and **open work** is 0, log and exit 0 before any
  probe or spawn. Open work is tickets without a resolution reply, plus a candidate SHA on `dev` that
  is ahead of `harness-stable` and touches fleet paths. It is **not** the unread count: a ticket read
  at Gate 0 is still open until its fix is promoted and replied to. `ailang mission ticket --open
  --count` computes it.
- Charter `design_docs/fleet-mission.md`: ranking rule, HD-2 policy classes, promotion gate.
- Skill: `gate-2-pick.md` gains a fleet branch that inverts admissibility (harness admissible,
  product inadmissible). Everything else is shared.
- Acceptance: one real ticket goes filed → fixed → replied end to end. An empty inbox logs the
  pre-check exit and spends zero tokens.

**Phase 3a: skill-resolution spike (0.5d, gates 3b).** The skills seam is proven live (V12) but
how each controller *resolves* skills is not measured, so no mechanism is chosen here. The spike
answers three PENDING premises, each by planting a marker skill in the pin worktree and a
conflicting one on the user-level path, then asking the controller which it loaded:

- **P1 (claude):** does `claude -p` with cwd = `$REPO` load skills from `$REPO/.claude/skills`,
  from `~/.claude/skills`, or from an `--add-dir`? Does a per-fire `CLAUDE_CONFIG_DIR` keep
  subscription auth working (the keychain credential is the billing path; see the driver's
  BILLING GUARD)?
- **P2 (codex):** does the codex controller discover skills at all, or only follow the prompt's
  instruction to read `SKILL.md` by path?
- **P3 (pi):** pi does not discover `.agents/skills/` (driver comment around line 1584) but takes
  `--skill`. Confirm `--skill <abs path>` works for the controller role.

Two cases need a specific answer. **World** has no project `.claude/skills` (V12), so retiring the
user symlinks without a replacement would leave it with none. **v1** runs with cwd = the main
checkout (V8), so any cwd-based project-skill resolution keeps reading the live tree unless v1
moves to a worktree. If P1–P3 cannot yield a per-fire, pin-rooted skill source for a provider,
that provider is recorded as **unpinned for skills**, and goal 4 is reported per seam rather than
claimed.

**Phase 3b: release channel (2.5d).**
- `origin/harness-stable` branch. Product mission env files set
  `AILANG_DRIVER_REF=origin/harness-stable`.
- **Binary:** once per promoted SHA, the driver builds `go build -o $AILANG_DRIVER_PIN_DIR/bin/ailang
  ./cmd/ailang` in the pin worktree (63s measured, V16), cached by SHA. Product fires prepend that
  directory to PATH (HD-6). A failed build keeps the previous SHA's binary and reports loudly;
  it never falls back to `~/go/bin` silently.
- **Skills:** the mechanism the spike selected, per provider.
- **The fleet canary uses the same three-seam mechanism, pointed at `dev`.** Fleet sets
  `AILANG_DRIVER_REF=origin/dev`, builds its binary from that pin into its own pin dir and puts it
  first on PATH, and loads skills through the spike-selected mechanism from its pin worktree. It is
  the same code path as product fires with a different ref, so a fleet fire on the candidate SHA
  actually executes the candidate's driver, skills and `internal/mission`. Each fleet fire logs
  the three resolved SHAs, and promotion requires all three to equal the candidate.
- Promotion gate, run by the fleet loop: `make test-launchd-drivers` green; a dry run of every
  product profile, healthy **and** drought-simulated (the mission-loop-change skill's Gate 4
  recipe, V15), run with `AILANG_DRIVER_PINNED=<sha>` so the pin cannot re-exec into another copy
  (V15); the binary builds; and 2 completed fleet fires whose logged driver, skills and binary SHAs all equal the candidate. Then
  `git push origin <sha>:harness-stable` (fast-forward only).
- Acceptance: goal 4's three planted-bad-commit tests (driver, skill, `internal/mission`), then
  their reverts. Each product fire logs the SHA it resolved for all three seams.

### Files to Modify/Create

- `missions/fleet.toml` (new, ~30 lines)
- `design_docs/fleet-mission.md` (new charter, ~150 lines)
- `.claude/skills/mission-control/resources/gate-2-pick.md` (rewrite the admissibility steps, fleet branch, ~40 lines)
- `cmd/ailang/mission_ticket.go` and `internal/mission/ticket.go` (new, ~250 lines with tests)
- `tools/launchd/mission-control.sh` (fleet empty-inbox pre-check, absolute skill path in the prompt, ~40 lines)
- `tools/launchd/mission-env/mission-{v1,docs,motoko,world}.env` (set `AILANG_DRIVER_REF`)
- `scripts/hooks/mission_scope_guard.sh` (new, ~80 lines) and `.pi/extensions/prepush-gate.ts` (call it)
- `tools/launchd/test_mission_scope_guard.sh` (new, wired in `make/test.mk`)

## Examples

### Example 1: today's stall, under this design

At 04:45 on 2026-09-26, world's fire was stall-killed at gate 3 after a `pi:openrouter` executor made
no progress for 40+ minutes. Under this design the next world fire files:

```json
{"mission":"world","iteration":192,"signature":"stall:gate-3:pi-openrouter-executor",
 "slot_verdict":"KILLED_at=gate-3 rc=143","slots_lost":1,"blocking":"none","workaround":"none"}
```

World carries on with row 24. The fleet loop picks the ticket up. This one is policy class
(routing), so under HD-2(a) it parks a decision row. Mark rules "opus before pi". The fleet loop
lands it, promotes it, and replies with the `harness-stable` SHA. (Today this happened in an attended
session instead: commit `d68e51e8e`.)

### Example 2: a product loop tries to fix the driver

A v1 controller edits `tools/launchd/mission-control.sh` to work around a probe timeout. The push is
refused:

```
mission scope guard: MISSION_NAME=v1 may not change tools/launchd/mission-control.sh.
File it instead: ailang mission ticket --mission v1 --signature probe-timeout:... (inbox mission-fleet)
```

## Success Criteria

- [ ] Phase 1: planted ticket from each of the 4 missions found in `mission-fleet`; scope-guard refusal and positive control both tested
- [ ] Phase 2: one real ticket completes filed → fixed → replied; empty-inbox fire spends zero controller tokens (log proof)
- [ ] Phase 3a: P1–P3 answered with marker-skill evidence; any provider left unpinned for skills is named
- [ ] Phase 3b: planted broken commits in the driver, a skill and `internal/mission` do not run in any product fire; the fleet fire does run them
- [ ] Each product fire logs the resolved SHA for driver, skills and binary
- [ ] 14 days after Phase 2: `[HARNESS]` share ≤ 10% across product loops
- [ ] `make test-launchd-drivers` green; new suite wired; no orphans
- [ ] CHANGELOG, mission-loop-change skill and `docs/internal/message-plane-topology.md` updated

## Testing Strategy

- **Scope guard:** table-driven shell test over (MISSION_NAME × path class) → allow/refuse,
  including unset `MISSION_NAME` (attended, allowed). Mutation-tested by reverting the guard.
- **Ticket CLI:** Go unit tests for schema validation, signature stability, and refile incrementing
  `slots_lost` instead of duplicating. The inbox round trip is an integration test against the dev
  message project.
- **Pre-check:** driver test with a stubbed `ailang messages` returning 0 and then 1. It asserts no
  spawn in the first case (the positive control is required, so the gate is not vacuous).
- **Promotion:** dry-run each product profile with `AILANG_DRIVER_REF=origin/harness-stable` and
  assert the resolved SHA in the pin log line.

## Deferred Decisions

Agent-resolvable at planning:
- The exact fleet scope glob list (start from the list above; the scope-guard test pins it).
- Fleet schedule (6h interval to start; tune from slot verdicts).
- Whether `slots_lost` is incremented by the filer or computed by the fleet loop from slot-verdict logs.

## Non-Goals

- Moving the harness to another repo (HD-1).
- Rewriting harness policy in AILANG. A good follow-on (policy decisions belong in AILANG, e.g.
  `design_docs/planned/m_one_role_table_probe.ail`), but independent of this doc.
- Changing world's own in-repo tooling (`scripts/`, its CI). That is world's product infrastructure,
  not the fleet harness.

## Timeline

| Phase | Work | Estimate |
|---|---|---|
| P1 | Inbox, ticket CLI, gate-2 rewrite, scope guard | 1.5d |
| P2 | `fleet` mission, pre-check, charter, skill branch | 2d |
| P3a | Skill-resolution spike (gates P3b) | 0.5d |
| P3b | `harness-stable`, binary and skill pinning, promotion gate | 2.5d |

P1 is worth shipping alone. It turns scattered escalations into one queue that attended sessions can
drain today.

## Risks & Mitigations

| Risk | Mitigation |
|---|---|
| The fleet loop becomes the new sink: harness work expands to fill it | Ticket-driven queue only; no self-sourced audits (V3's "never ran out" finding) |
| `harness-stable` lags and product loops run a known bug | Promotion gate is mechanical and runs every fleet fire; attended sessions can fast-forward by hand |
| Tickets silently lost (message-plane dispatch had a long silent-failure history) | TRIAGE inbox (read at Gate 0, not dispatched); Phase 1 acceptance plants one per mission |
| The fleet loop makes a bad routing or quota change autonomously | HD-2(a): policy classes park for Mark |
| Promotion misses skill changes because skills load from the working tree | Phase 3a measures resolution per provider before choosing a mechanism; 3b's planted-skill test proves it; an unpinnable provider is reported, not claimed |
| Promotion misses binary changes | The binary is built from the promoted SHA (HD-6); planted `internal/mission` commit in acceptance |
| Extra quota use | Empty-inbox pre-check spends zero; one loop at 6h interval; shares the fleet ration gate |

## Verification Log

| # | Claim | How verified | Result |
|---|---|---|---|
| V1 | World 164–180: 12/13 recorded entries `[HARNESS]`; 181–194: 10/10 product | Header scan of `world-mission-log.md` plus archive for `^## N — `, classified on `[HARNESS` | 164–168 H, 169 P, 170–175 H, 180 H; 181, 183–189, 193, 194 P. 176–179, 182 and 190–192 have no entry. 8 PRs merged 09-24..26 (`gh pr list -R sunholo-data/ailang-world`, #143–#150) |
| V2 | V1 309–348: 18/35 `[HARNESS]` | Same scan over `v1-mission-log*.md` | 18 of 35 headers found. gate-2-pick.md cites "55%" for the same window from a different count; this doc uses the measured 18/35 |
| V3 | The attended-only rule exists and is prose-only | Read `.claude/skills/mission-control/resources/gate-2-pick.md` lines 1–35 | Confirmed (Mark 2026-09-21), including "never ran out" and the 883-vs-16 lines figure |
| V4 | No open queue rows tagged `[HARNESS]` in the charters | `grep -E '^\|' <charter> \| grep -c HARNESS` on v1, world and docs | 0, 0, 0 |
| V5 | World escalations arrive as filed-not-dispatched messages | `ailang messages inboxes` (prod) | `mission-world` is TRIAGE, "filed for a human, on purpose" |
| V6 | No harness or fleet inbox exists | `ailang messages inboxes \| grep -ci harness` | 0 |
| V7 | No mechanical guard stops a product loop editing harness paths | `grep -n 'tools/launchd\|internal/mission'` over `.pi/extensions/prepush-gate.ts`, `session-protocol-gate.ts`, `scripts/hooks/*.sh` | No path guard; only a comment in a test |
| V8 | v1, docs and motoko work in the ailang repo; world does not | `grep -E '^(repo\|workdir)' missions/*.toml` | v1 = main checkout; docs and motoko = separate clones of `sunholo-data/ailang`; world = `ailang-world` |
| V9 | Loop-harness size | `wc -l` | Driver 2456 lines; `tools/launchd` shell total 9481 including tests; `internal/mission` 10177 (non-test); mission-control skill 5251 |
| V10 | `internal/mission` ships in the `ailang` binary | `ailang mission --help` lists report, iterate, status, quota and more | Confirmed |
| V11 | Driver pin defaults to `origin/dev` and accepts any ref | `pin-root.sh` lines 34 and 175–196: `ref="${AILANG_DRIVER_REF:-origin/dev}"`, `git rev-parse "$ref"` | Confirmed |
| V12 | Skills load from the main checkout's working tree, not the pin | `ls -la ~/.claude/skills/`: `mission-control`, `sprint-*` and `design-doc-creator` link to `~/dev/sunholo-data/ailang/.claude/skills/*`; `ailang-world` has no `.claude/skills`; the driver prompt uses a relative `.claude/skills/...` path (mission-control.sh around line 2199) with cwd `$REPO` | Confirmed. Driver pinned, skills live |
| V13 | A moving **tag** would not update under the pin's fetch | `pin-root.sh:187` runs `git fetch --quiet origin` (no `--tags --force`). Experiment with git 2.54.0: bare origin; clone B fetches tag `stable`; A force-moves and pushes it; B runs `git fetch --quiet origin` | `fetch rc=0`, B's `stable` stays on the **old** commit (4e2b256 vs origin c2fee49), silently. Hence a branch: `origin/harness-stable` updates as a remote-tracking ref |
| V15 | The dry-run and drought-simulation recipe exists and works on the working-tree driver | Read `.claude/skills/mission-loop-change/SKILL.md` Gate 4 (`MISSION_DRY_RUN=1`, drought via `MISSION_DESIGNER_MODEL='claude:claude-drought-sim'`). Ran a world dry run 2026-09-26 15:16 | The skill's `AILANG_DRIVER_PIN=0` recipe is **defeated for world**: its installed env runs `export AILANG_DRIVER_PIN=1`, so the first dry run re-executed into the pinned copy and tested the old driver (the old credential line appeared). Rerun with `AILANG_DRIVER_PINNED=worktree-test` tested the working tree (`DRY RUN ok`, roles resolved). The promotion gate therefore sets `AILANG_DRIVER_PINNED` explicitly |
| V16 | Building the binary from a pin worktree is affordable | `/usr/bin/time go build -o <scratch> ./cmd/ailang` in `~/.ailang-driver-pin/world` @ 28f4f7433; the result runs `mission quota --help` | real 62.9s; the binary works. The driver calls `ailang` by PATH (22 `ailang ` call sites in `mission-control.sh`), so PATH order selects the binary |
| P1–P3 | **PENDING**: per-provider skill resolution | Phase 3a spike | Unmeasured; see Phase 3a |
| V14 | Kill switches hold v1, docs and motoko today | `ls ~/.ailang/state/*.disabled` | `mission-control`, `mission-docs` and `mission-motoko` disabled; world live |

## Related Documents

- [M-MISSION-LOOP-WORKBENCH](v0_36_0/m-mission-loop-workbench.md): the registry and `ailang mission install`, which this reuses for `fleet`. Distinct: that doc is topology and config; this one is work routing.
- [M-MISSION-COMMS-INTO-THE-BINARY](v0_36_0/m-mission-comms-into-the-binary.md): typed comms. The ticket CLI should sit beside its `mission report`.
- [M-MISSION-RUNTIME-CONTRACT](m-mission-runtime-contract.md): durable work items. A ticket could later become a work item; not required here.
- `.claude/skills/mission-control/resources/gate-2-pick.md`: the rule this operationalises.

## Future Work

- Pin the `ailang` binary per release channel (build from `harness-stable` into the pin dir).
- Express the routing and ration policy in AILANG, with the fleet loop as its first dogfooding consumer.
- Revisit HD-1(b) if the harness gains a consumer outside AILANG.

## Quorum History

**Round 1 (2026-09-26, gpt6-astra, gemini-3-1-pro and oc-glm-5-3; controller pass): blocked 3/3.**
All three objections were correct and are addressed in this revision:

- *gpt6-astra:* the unpinned `~/go/bin/ailang` let unpromoted `internal/mission` changes reach
  product fires, contradicting goal 4. **Fixed:** the binary moved inside the promotion boundary
  (HD-6, Phase 3b, V16); acceptance plants an `internal/mission` commit.
- *gemini-3-1-pro:* the drought-simulation dependency had no Verification Log row. **Fixed:** V15,
  which also surfaced a real defect in the recipe (world's env defeats `AILANG_DRIVER_PIN=0`).
- *oc-glm-5-3:* the skill-pinning mechanism was asserted and deferred at once, per-provider
  resolution was unmeasured, and only a driver commit was planted. **Fixed:** Phase 3a spike with
  PENDING premises P1–P3 gating 3b; world and v1 cases named; planted skill commit in acceptance;
  goal 4 reported per seam if a provider cannot be pinned.

**Round 2 (2026-09-26, same seats; controller pass): blocked 3/3.** Again, every objection was correct:

- *gpt6-astra:* the idle pre-check counted *unread* tickets, so a read-but-unfixed ticket could
  starve its own promotion. **Fixed in-doc (unreviewed):** the pre-check counts *open* work
  (unresolved tickets plus an unpromoted candidate SHA).
- *oc-glm-5-3:* the canary did not itself run the candidate's binary and skills, so "2 fleet fires"
  could pass on stale code. **Fixed in-doc (unreviewed):** fleet uses the same three-seam
  mechanism on `dev`, and promotion requires the logged SHAs to match the candidate.
- *gemini-3-1-pro:* P1–P3 (per-provider skill resolution) are unmeasured. **Not fixable by
  editing:** it needs the Phase 3a spike to be run. Phases 1 and 2 do not depend on it.

**Per the re-quorum-once guardrail, there is no third round.** The doc goes to Mark with these gaps
labelled: the two round-2 fixes are unreviewed, and P1–P3 are pending. Phases 1–2 are independent
of both and can be ratified alone.
