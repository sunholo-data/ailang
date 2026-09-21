# M-ONE-ROLE-TABLE: the mission driver and the registry stop keeping two role tables

**Status**: Planned — **quorum guardrail spent (round 0 + 1 re-quorum, both BLOCKED 3/3; all six objections addressed in-session, see §Quorum History). Awaiting Mark's ratification of the Design Freeze items, not a third review round.** **Un-parks M8 of [M-MODEL-REGISTRY-SINGLE-SOURCE](v0_35_0/m-model-registry-single-source-sprint-plan.md#m8--mission-driver-adoption-d3a--parked-2026-08-27-mark-if-it-aint-broken-wont-fix)**, which Mark parked 2026-08-27. Un-parking is a design-freeze item: see §Design Freeze.
**Target**: release assignment at sprint planning (mission infrastructure, same convention as [m-mission-runtime-contract](m-mission-runtime-contract.md))
**Priority**: P1 — row 4 of the migration ledger, three incidents, and the precondition for retiring ledger row 3 (role dispatch)
**Estimated**: to be set at sprint planning; the discovery below removes most of what the park note priced
**Dependencies**: none blocking. Shares `internal/modelreg` with [m-secondary-model-fallback](m-secondary-model-fallback.md) — see §Related Documents for the boundary.
**Created**: 2026-09-21

---

## 0. The finding that reshapes this doc: M8's park is mostly stale

M8 was parked on four named capability gaps. **Three are closed at HEAD and the fourth is one
missing row.** Verified 2026-09-21, command by command, in §Verification Log:

| Park-note blocker (2026-08-27) | Status at HEAD (2026-09-21) |
|---|---|
| Registry cannot model **three-deep chains** | **CLOSED** — `evaluator` and `executor` both resolve 3 rungs today |
| Registry cannot model the **`pi` harness** | **CLOSED** — 12 rows carry `agent_cli: "pi"` |
| Registry cannot model the **ollama-cloud flat-rate tier** | **DATA GAP, not a capability gap** — `IsOllamaCloudRoute` (`models.go:284`) and `UsesLocalGPU` already classify `:cloud` rows as non-GPU so they survive the `LaneCloud` filter. What is missing is a *row*: all 4 pi×ollama rows are local-GPU qwen3.x; there is **zero** pi×ollama-cloud row |
| Registry yields **`claude:fable`**, silently downgrading the designer to opus | **CLOSED** — the `claude-fable-5-1` row carries `agent_model_name: "claude-fable-5-1"`, the full ID, with a comment naming that exact alias trap |

This is the same class of error this program keeps paying for: **a document's premise decayed
while the document stood still.** The park was correct when written. Acting on it today without
re-measuring would have built four things, three of which already exist.

**But the park note also under-described the real obstacle**, and that is what this doc is
actually about (§2). The planner lane is not a static role→chain lookup at all.

---

## Axiom Compliance

**Canonical reference:** [Design Axioms](/docs/references/axioms)

### Axiom Scoring

| Axiom | Score | Justification |
|-------|-------|---------------|
| A1: Determinism | +1 | One resolver replaces two that had silently diverged three times. Same inputs → same lane, and the lane becomes reproducible off-rig because the policy is a pure function of declared inputs |
| A2: Replayability | +1 | The resolution emits its inputs and its reason token, so a past routing decision can be replayed and explained rather than reconstructed from a driver log |
| A3: Effect Legibility | +1 | Probe results (provider availability) become a declared INPUT to the policy rather than an ambient env var read mid-script |
| A4: Explicit Authority | 0 | No authority change. The allowlist stays an allowlist and stays per-mission data |
| A5: Bounded Verification | +1 | A policy function is testable off-rig with table tests; today's equivalent needs bash 3.2 on the rig with zero CI coverage |
| A6: Safe Concurrency | 0 | No concurrency surface |
| A7: Machines First | +1 | One table a machine can read (`ailang models role`) instead of a table plus 261 lines of shell a machine must re-implement to predict |
| A8: Minimal Syntax | 0 | No new AILANG syntax; reuses the existing embed call path |
| A9: Cost Visibility | +1 | The ollama-cloud flat-rate tier becomes expressible in the registry, so the cheap-before-metered ordering is visible where pricing already lives |
| A10: Composability | +1 | The registry's `roles:` already serves the coordinator and the binary mission path; this makes the shell a third consumer of the same row rather than a second author |
| A11: Structured Failure | +1 | `resolve-role-spawn.sh`'s fail-closed reason tokens are preserved as a typed result, not a string convention re-parsed by each caller |
| A12: System Boundary | 0 | Boundary unchanged — the driver still shells out; only who answers the question moves |

**Net Score: +8** → **Decision: Move forward**

### Hard Violation Check

- [x] A1 (Determinism): no implicit nondeterminism — the probe result becomes an explicit input
- [x] A3 (Effects): no hidden side effects — resolution is pure; probing stays in the shell
- [x] A4 (Authority): no ambient access granted — allowlist semantics unchanged
- [x] A7 (Machines First): improves machine legibility; that is the point

---

## Problem Statement

**The mission fleet keeps two role tables and neither knows about the other.**

The shell driver resolves a role from `MISSION_<ROLE>_MODEL` / `MISSION_<ROLE>_FALLBACK`
environment pins plus 261 lines of derivation (`derive-planner-lane.sh` 182,
`resolve-role-spawn.sh` 79). The binary resolves the same role from `internal/modelreg`'s
`roles:` block via `ailang models role`.

**No shell script consumes `ailang models role`** — re-verified 2026-09-21 — despite
`models_cmd.go:79` carrying the comment *"the mission driver reads field 2"*. That comment has
been false for the life of the command.

Two independently maintained tables for one decision have now produced **three incidents in
eight days**, each found as an incident and each fixed for one role:

| Fix | Role | What the binary did |
|---|---|---|
| `7423434b4` (09-14) | evaluator | dispatched to opencode while the shell ran pi — the harness that cannot load `sprint-evaluator`, which is how M-MISSION-ITERATION-RELIABILITY M4's canary failed 3/3 with no verdict |
| `e9e8ce32e` (09-14) | executor | `[codex, opencode]` — with codex rationed, "no available candidate" |
| `5f67aa532` (09-21) | designer + planner | `[opencode]` and `[codex, opencode]` — designer had no skill-capable route at any time |

The third fix replaced per-role patching with a property
(`TestResolveRole_EveryRoleHasASkillCapableRung`), so *that* class cannot recur silently. **It
does not make the tables one.** The registry's rows are still an independent opinion that
happens, today, to be checked against a rule rather than against the shell.

**Impact**: every row the migration ledger moves from shell to binary inherits the registry's
opinion. Ledger row 3 (role dispatch to pi) cannot be retired while the table it would dispatch
from is authored separately from the one in use.

---

## Goals

**Primary goal**: one authority answers "what runs role R for mission M right now", and both the
shell driver and the binary read it.

**Success metrics**

1. `MISSION_DRY_RUN` resolves byte-identically to today for all four roles on all four missions (v1, docs, motoko, world) — the inertness bar M8 set.
2. A simulated stale binary (subcommand absent → non-zero exit) falls back to today's env pins; the loop still runs. This is M8's first ratified condition and is non-negotiable: `~/go/bin/ailang` drifts by design.
3. `MISSION_PLANNER_MODEL=opus` still overrides — the rollback ergonomic survives.
4. The planner lane is decidable off-rig: a table test reproduces every (doc, mission, availability) case that `derive-planner-lane.sh` handles today.
5. `ailang models role` and the shell agree by construction, asserted by a test that reds if they diverge.
6. The binary mission path resolves through the same resolver as the shell — not a static lookup beside it.

---

## 2. What the park note under-described: the planner lane is a policy function

The park note framed M8 as "read a value from a CLI" and then corrected itself to "the driver has
a multi-tier, multi-provider routing scheme". Both descriptions miss the shape that actually
matters.

**The planner lane is not a lookup. It is a function of three inputs, only one of which the
registry has:**

```
planner_lane = f( design_doc_paths, mission_allowlist, anthropic_available )
```

- **`design_doc_paths`** — `derive-planner-lane.sh` reads the design document and extracts the paths it declares. A doc touching `internal/` fails closed to opus, because a cheap planner has no business planning compiler changes.
- **`mission_allowlist`** — per-mission data since 2026-08-28 (`MISSION_PLANNER_ALLOWLIST`). The infra list is the default; the docs mission widens it. Before it was per-mission, every `docs/` doc failed closed and the docs mission's cheap planner pin "read as configured while OPUS actually runs, every iteration."
- **`anthropic_available`** — `MISSION_PLANNER_ANTHROPIC_FALLBACK` is applied **only when `MISSION_ANTHROPIC_AVAILABLE=0`** (`derive-planner-lane.sh:8-14`). It is a predicate-guarded substitution, not a rung in an ordered chain.

The registry's `roles:` block is a **static ordered list per (role, lane)**. It cannot express a
predicate over a per-task input. That is the genuine, still-open expressiveness gap — and unlike
the park note's other three, it does not close by adding rows.

**So the question this doc must answer is not "can the registry hold the driver's table?" It is
"where does per-task routing POLICY live, given the registry holds routing DATA?"**

---

## High-Impact Decisions

| ID | Decision | Recommendation | Chosen by | Change cost |
|---|---|---|---|---|
| **D1** | Where does per-task routing policy live? | **(a) An AILANG program.** Precedent is ratified and live: `internal/dashboard_transforms/budget_checker.ail` is the ONE implementation of the budget rule, called from Go via `engine.CallPreserveFloats` (`cmd/ailang/budget.go:197`), with the silent Go fallback deliberately removed. Routing is the same shape — a pure decision over declared inputs — and the north star says policy belongs in AILANG with Go as the shell. **The capability is now measured, not assumed (V16):** a probe doing prefix matching, allowlist membership, fail-closed quantification and nested branching is `ailang check`-clean. **(b)** Go-only in `internal/modelreg`. **(c)** Leave it in bash. | Mark (architectural; this is the un-park) | Medium |
| **D2** | Does the shell read the binary, or does the binary become the driver? | **(a) The shell reads the binary** (M8's original scope) and keeps scheduling. Ledger row 17 (scheduling) is an explicit non-goal of the whole migration, and row 2's opt-in is a separate decision. **(b)** Move the iteration into `mission iterate` — that is ledger row 2, not this doc. | Mark | High |
| **D3** | What is the fallback when the binary lacks the subcommand? | **Split, because DATA and POLICY degrade differently — see §3.** DATA (which models serve a role) falls back to today's env pins, unconditionally, with a loud log line; that is M8's ratified condition and it stays. POLICY (the path allowlist) must fall back **fail-closed to opus**, NOT to env pins: env pins do not encode the document-path refusal, so an env-pin fallback would let a stale binary route a compiler doc to the cheap planner. Never fail-closed to "no lane": that wedges four live loops | Ratified 2026-08-27 for DATA; the POLICY half is new here | Low |
| **D4** | Does the registry gain the ollama-cloud pi rows? | **Yes — data, landed first and independently.** It is a row, not a capability, and it is verifiable in isolation before any resolver moves | Agent-resolvable | Low |

### Design Freeze

- [ ] **Approve un-parking M8** (Mark). It was parked with "if it ain't broken won't fix"; §0 shows three of the four stated blockers are gone and §Problem shows it broke three times in eight days. This doc does not assume the park is lifted.
- [ ] **Approve D1** — policy in AILANG vs Go vs bash. Everything downstream depends on it.
- [ ] **Approve D2** — confirm the shell stays the scheduler for this doc.
- [ ] D3 and D4 are agent-resolvable and need no freeze.

---

## Solution Design

### Overview

Split the two things the driver conflates:

- **DATA** — which models serve a role, in what order, on which harness, at what price. Already in `internal/modelreg`. Gains the missing ollama-cloud pi rows.
- **POLICY** — which lane this *particular* task gets, given the doc's paths, the mission's allowlist and probed provider availability. Today: 261 lines of bash with zero CI coverage. Proposed: one AILANG function with declared inputs.

The shell keeps doing what only it can do (probing, scheduling, process supervision) and stops
being a second author of the table.

### Architecture

```
                 ┌─────────────────────────────┐
  probes  ──────▶│  mission-control.sh         │   scheduling, supervision,
  (shell)        │  (keeps: probe, kill switch,│   process trees — unchanged
                 │   heartbeat, pin, spawn)    │
                 └──────────────┬──────────────┘
                                │ role, mission, doc-paths, anthropic_available
                                ▼
                 ┌─────────────────────────────┐
                 │ ailang mission role-resolve │   ONE answer, one reason token
                 └──────────────┬──────────────┘
                     ┌──────────┴──────────┐
                     ▼                     ▼
        ┌────────────────────┐   ┌──────────────────────┐
        │ POLICY (.ail)      │   │ DATA (modelreg)      │
        │ allowlist, fail-   │   │ roles:, harnesses,   │
        │ closed, provider   │   │ pricing, lanes       │
        │ substitution       │   │                      │
        └────────────────────┘   └──────────────────────┘
```

`resolve-role-spawn.sh`'s output contract is **preserved exactly**: one line,
`<value...> <reason-token>`, always exit 0.

**There are 17 distinct reason tokens across the two scripts, and the port must carry all of
them.** Enumerated rather than summarised, because an earlier draft of this section listed only
the 8 from `resolve-role-spawn.sh` and silently dropped `derive-planner-lane.sh`'s 9 — which
would have under-scoped the port by more than half:

| Script | Tokens |
|---|---|
| `resolve-role-spawn.sh` (8) | `fail-closed:role-missing`, `fail-closed:role-unknown`, `fail-closed:derive-script-missing`, `fail-closed:derive-unparsable`, `fail-closed:<role>-model-missing`, `fail-closed:evaluator-collision-no-fallback`, `declared:provider-pin`, `declared:alias-pin`, plus the rerouting reason `generator-equals-judge` |
| `derive-planner-lane.sh` (9) | `declared:codex-ok`, `declared:opus-required`, `fail-closed:env-pin`, `fail-closed:no-doc`, `fail-closed:no-files-section`, `fail-closed:path-not-in-codex-allowlist`, `fail-closed:planner-lane-field-invalid`, `fail-closed:planner-lane-field-missing`, `fail-closed:unparsable-path-entry`, plus the `anthropic-fallback:<reason>` prefix that wraps any of them |

Each survives as a typed result rendered to the same string. The differential test (§Testing
Strategy) must cover every one — a token with no arm is a branch the port can silently drop.
Callers do not change in phase 1.

Note what this enumeration reveals about scope: `derive-planner-lane.sh` has **more** distinct
outcomes than the dispatcher that calls it, and six of its nine are refusals about the *design
document* (`no-doc`, `no-files-section`, `unparsable-path-entry`, `planner-lane-field-missing`,
`planner-lane-field-invalid`, `path-not-in-codex-allowlist`). That is further evidence for §2:
the planner lane is a document-parsing policy, not a table lookup.

### Implementation Plan

**Phase 0 — data only, independently landable (no resolver change).**
Add the pi×ollama-cloud rows the driver's fallbacks name (`ollama/kimi-k3:cloud`,
`ollama/deepseek-v4-flash:0731-cloud`), so the registry can *express* the flat-rate tier. Assert
they resolve on both lanes and are not treated as local GPU. Nothing reads them yet — this phase
is falsifiable on its own and reversible.

**Phase 1 — the policy function, off-rig, unwired.**
Port `derive-planner-lane.sh`'s decision (path allowlist, fail-closed, provider substitution) to
AILANG with declared inputs. Prove equivalence by a table test enumerating every case the shell
handles, including the two the shell's own comments record as measured: a `docs/` doc failing
closed under the infra allowlist, and the `set -f` pathname-expansion trap. **The shell is not
touched in this phase.** A differential test runs both and asserts identical output.

**Phase 2 — one command.**
`ailang mission role-resolve` returns the resolved lane plus reason token, reading DATA from
modelreg and POLICY from phase 1.

**Phase 3 — the shell reads it, attended.**
`resolve-role-spawn.sh` delegates, with D3's fallback branch as an acceptance criterion rather
than a nicety. This is the only phase that touches a live loop and it is attended-window only,
per M8's original ratification. `MISSION_DRY_RUN` byte-identity for all four roles × four
missions is the gate.

**Phase 3b — the BINARY reads it too. Without this phase the doc does not do what it claims.**
`gpt6-astra` blocked round 1 on exactly this: V11 names three binary consumers
(`iteration/runtime.go:68`, `iteration/runtime_stage.go:83`, `iteration/retry_review.go:260`)
and no earlier phase migrated them. As drafted, the shell would gain document- and
availability-aware routing while the binary kept a static `ResolveRole` lookup — **re-creating
the divergence this doc exists to end, with the two halves swapped.**

The three sites move to the same resolver. One input needs a decision the shell never had to
make: the binary mission path carries a **work item**, not a design document, so
`design_doc_paths` is not automatically populated. Two admissible answers, and the doc picks the
second:

1. The work item declares its paths and they feed the policy exactly as a design doc's do.
2. The binary passes *no* document, which the policy already has a token for —
   `fail-closed:no-doc` — and which therefore resolves to opus rather than to a cheap planner.

**(2) is the correct default** because it is fail-closed and needs no new schema; (1) becomes
available whenever a work item chooses to declare paths. Either way the ANSWER comes from one
resolver, which is the property being bought.

**Phase 4 — end the duplicate AUTHORITY. Do NOT delete the script.**
After one full attended iteration per mission completes green, `derive-planner-lane.sh` stops
being *an* answer and becomes *the degraded-mode* answer — the implementation the stale-binary
branch runs. It is not deleted by this doc.

This is a correction, not a hedge. An earlier draft had phase 4 delete the script while D3
promised a permanent stale-binary fallback, and those cannot both hold: after deletion the
fallback would be env pins alone, which do not encode the path allowlist, so a missing
subcommand would **silently widen** routing rather than preserve it — a cheap planner reaching a
compiler doc that today fails closed to opus. Physical deletion requires first proving the
fallback preserves the refusal, which is its own evidence and is out of scope here.

### Files to Modify/Create

- `internal/modelreg/models.yml` — +2 pi×ollama-cloud rows (~60 lines with the required pricing/clamp provenance comments). Phase 0.
- `internal/modelreg/roles_test.go` — assert the new rows resolve on both lanes and are not local-GPU. Phase 0.
- `design_docs/planned/m_one_role_table_probe.ail` — the D1 capability probe, `ailang check`-clean with no flags; keep it beside the doc as the evidence for V16. Underscores, not hyphens: a module declaration cannot contain a hyphen.
- `internal/mission/policy/role_lane.ail` — NEW, the policy function. The probe is its skeleton. Path follows `internal/dashboard_transforms/*.ail` convention; final location is agent-resolvable. Phase 1.
- `internal/mission/policy/role_lane_test.go` — table test + differential test against the shell. Phase 1.
- `cmd/ailang/mission_role_resolve.go` — NEW, the command. Phase 2.
- `internal/mission/iteration/runtime.go` — migrate `ResolveRole` call at `:68` to the resolver. Phase 3b.
- `internal/mission/iteration/runtime_stage.go` — same, `:83`. Phase 3b.
- `internal/mission/iteration/retry_review.go` — same, `:260`. Phase 3b.
- `tools/launchd/resolve-role-spawn.sh` — delegate, with the stale-binary fallback branch. Phase 3.
- `tools/launchd/derive-planner-lane.sh` — **retained** as the degraded-mode implementation; loses authority in phase 3, is not deleted by this doc (see Phase 4).
- `tools/launchd/test_mission_routing.sh` — extend; this file is the only existing coverage of the shell path.
- `cmd/ailang/models_cmd.go` — fix the false comment at `:79` in phase 3, when it becomes true.

---

## Conflict Surface

Not required by the skill's trigger list (no parser/typechecker/codegen files), written anyway
because the measured risk here is higher than a parser change: **this surface is bash 3.2.57 with
zero CI coverage driving four live loops.**

**1. What positions does this change extend?**
Role resolution — the single point where a role name becomes a spawn recipe.

**2. What else already lives in those positions?**

| Occupant | Interaction | Decision |
|---|---|---|
| `MISSION_<ROLE>_MODEL` env overrides | Must keep winning — the documented rollback (`MISSION_PLANNER_MODEL=opus`) | **Reuse**: env override is applied before the resolver is consulted |
| The evaluator generator≠judge collision check (`resolve-role-spawn.sh:66-77`) | Reads `MISSION_EXECUTOR_RESOLVED`, i.e. the *already-resolved* executor. Order-dependent | **Reuse**: resolution order preserved; executor resolves before evaluator |
| `MISSION_PLANNER_ALLOWLIST` per-mission data | Set in `mission-env/*.env`, read by the derive script | **Reuse**: becomes a declared policy input, still per-mission data |
| The coordinator's `ResolveModelChain` (`retry_chain.go:133`) | Reads the same `roles:` rows on `LaneCloud` | **Do not disturb**, and the premise is now logged as **V15** rather than borrowed: the in-repo fixture has **34 agents, 0 without a `model:` pin, 4 carrying a role**. So the role path is dormant — but `TestCloudAgents_RegistryMatchesTheDeletedRoutingTable` deliberately DROPS each pin to exercise it, which is exactly how it caught a designer change on 2026-09-21. Phase 0 must not change resolved models on `LaneCloud` |
| `TestCloudAgents_RegistryMatchesTheDeletedRoutingTable` | Asserts resolved MODEL still matches config.cloud.yaml's deleted table | **Hard constraint**: measured 2026-09-21 — this test reds if a role's head model changes. It is why `5f67aa532` moved the *harness* and kept the *model* |
| The binary mission path (`iteration/runtime.go:68`, `runtime_stage.go:83`, `retry_review.go:260`) | Reads `roles:` on `LaneLocal` | **Reuse**: becomes a second consumer of one answer |

**3. How is ambiguity resolved?** Precedence is fixed and asserted: explicit env pin → policy
substitution (provider unavailable) → registry chain → fail-closed reason token. Never a silent
default.

**4. Which existing behaviours MUST still work?**
- `MISSION_PLANNER_MODEL=opus` overrides (the rollback).
- A `docs/` design doc resolves via the docs mission's widened allowlist, not fail-closed.
- An `internal/` doc still fails closed to opus.
- A stale `ailang` without the subcommand still runs the loop, **strictly following the D3 split**: DATA falls back to env pins, POLICY fails closed to opus. (An earlier revision of this bullet said "on env pins" flatly, which contradicted the D3 fix made in the same revision — caught by `gemini-3-1-pro` in round 1.)
- `tools/launchd/*` paths pass the default infra allowlist (the `set -f` case).

**5. What deliberately changes?** `derive-planner-lane.sh` and `resolve-role-spawn.sh` stop being
the authority. `models_cmd.go:79`'s comment becomes true. Nothing about which model runs a role
changes in phases 0–2.

---

## Success Criteria

- [ ] Phase 0: the two pi×ollama-cloud rows resolve on both lanes, are not local-GPU, and change no existing role chain
- [ ] `TestCloudAgents_RegistryMatchesTheDeletedRoutingTable` stays green at every phase
- [ ] Differential test: policy function and `derive-planner-lane.sh` agree on every enumerated case
- [ ] `MISSION_DRY_RUN` byte-identical for 4 roles × 4 missions
- [ ] Simulated stale binary falls back to env pins; loop completes
- [ ] `MISSION_PLANNER_MODEL=opus` still overrides
- [ ] One full attended iteration per mission green before loops are left unattended
- [ ] **Both** callers resolve through one path: the shell AND all three `internal/mission/iteration` sites
- [ ] A binary-path stage with no declared document resolves `fail-closed:no-doc`, not a cheap planner
- [ ] Ledger row 4 marked retired in `m-mission-runtime-contract.md`
- [ ] All tests passing; changelog updated

## Testing Strategy

The load-bearing test is **differential, not unit**: run the existing shell and the new policy on
the same inputs and assert identical output. A unit test of the new policy proves only that it
matches its author's belief about the shell — which is precisely the failure mode that produced
two role tables.

Mutation requirement (repo standard): each guard must be shown to red. For the differential test
that means perturbing the allowlist, the availability flag and the doc paths one at a time and
confirming both implementations move together.

## Deferred Decisions

Agent latitude, no freeze needed:
- Exact path/module name for the `.ail` policy program.
- Whether phase 2's command is `mission role-resolve` or an extension of `models role`.
- Whether the reason token becomes a typed enum or stays a string at the shell boundary.

## Non-Goals

- **Scheduling.** Ledger row 17, explicitly deferred by the whole migration.
- **Moving the iteration into the binary.** That is ledger row 2 and its own opt-in decision.
- **Deleting `derive-planner-lane.sh`.** It becomes degraded-mode code, not dead code. Deleting it needs proof that the stale-binary fallback preserves the path refusal — separate evidence, separate doc.
- **Changing which model runs any role.** Phases 0–2 are inert by construction; any routing change is a separate, dated departure with its own assertion (the precedent is `7423434b4`).
- **The cloud plane's chain walk.** That is [m-secondary-model-fallback](m-secondary-model-fallback.md).

## Risks & Mitigations

| Risk | Mitigation |
|---|---|
| Wedging four live loops via a stale binary | D3's fallback branch is an acceptance criterion; `MISSION_DRY_RUN` first; attended window; phase 3 is the only phase touching a loop |
| bash 3.2.57, zero CI coverage | Phases 0–2 touch no shell at all. Phase 3 is one delegation with a fallback, verified by the existing `test_mission_routing.sh` plus dry-run byte-identity |
| The policy port silently differs from the shell | Differential test is the gate, not a unit test |
| Phase 0 disturbs the cloud plane | Phase 0 adds rows without changing any chain; the cloud transcription guard is the tripwire and it already demonstrated it reds (2026-09-21) |
| This doc's own premises decay | §Verification Log dates every claim and names the command. Re-run before sprint planning — that is the lesson §0 records |

## Related Documents

- [M-MODEL-REGISTRY-SINGLE-SOURCE](v0_35_0/m-model-registry-single-source.md) and its [sprint plan](v0_35_0/m-model-registry-single-source-sprint-plan.md) — **this doc un-parks its M8**.
- [m-mission-runtime-contract](m-mission-runtime-contract.md) — the migration ledger; this is **row 4**.
- [m-secondary-model-fallback](m-secondary-model-fallback.md) — **distinct, and the boundary is worth stating.** That doc fixes the *cloud* plane: 37 coordinator agents, every one a chain of one because each carries an explicit `model:` pin, walked in-container by `ClassifyFailure`. This doc fixes the *mission driver*: four rig loops whose lane comes from shell env pins and a doc-derived allowlist. They share `internal/modelreg` and nothing else. Notably that doc's own §1.2 records the same defect class this one exists to end — *"the declared tails name `opencode` harnesses while the agents run in `pi` containers"* — which is further evidence the two-author problem is systemic rather than local.
- [m-mission-comms-into-the-binary](v0_36_0/m-mission-comms-into-the-binary.md) — the sibling migration workstream.

## Quorum Trigger

Attended session, so the quorum runs only if one of the four mechanical triggers fires.

| # | Trigger | Fired? |
|---|---|---|
| 1 | Design-freeze items | **YES** — un-parking M8 is Mark's park to lift, and D1/D2 are his |
| 2 | Overrides shared machinery | No — every Conflict Surface row is *reuse* or *do not disturb*; nothing overrides |
| 3 | Cost/KPI semantics or banked schema | **YES** — Phase 0 adds priced model rows, and the ollama-cloud tier exists precisely to order flat-rate before metered |
| 4 | Load-bearing premises about external systems | **YES** — the ollama-cloud rate is a vendor contract we cannot re-check in-repo, which is why it is listed as *not verified* below |

Three of four fired; the quorum is required. Per the skill's own guidance the unlogged-claims
sweep was done BEFORE round 0 rather than one-per-round — it caught V5 (a negative-existence
claim) and V9b (this doc's own incomplete enumeration).

## Quorum History

**Round 0 (2026-09-21, $0.1277): BLOCKED 3/3.** Every objection was correct and every one is
addressed below rather than argued — the skill's rule, and all three found something real.

| Reviewer | Objection | Resolution |
|---|---|---|
| `gpt6-astra` | Phase 4 deleted `derive-planner-lane.sh` while D3 promised a permanent stale-binary fallback to env pins. Env pins do not encode the path refusal, so after deletion a stale binary would **widen** routing, not preserve it | **Design changed.** D3 split: DATA degrades to env pins, POLICY degrades **fail-closed to opus**. Phase 4 no longer deletes the script — it becomes the degraded-mode implementation. Deletion moved to Non-Goals with its evidence condition named |
| `gemini-3-1-pro` | The "all 37 cloud agents carry a pin" premise — which Phase 0's safety rests on — had no Verification Log row | **Verified and corrected → V15.** 34 agents, 0 unpinned, 4 role-bearing. The "37" was m-secondary-model-fallback's live-plane count, borrowed rather than measured; both figures and their dates are now stated |
| `oc-glm-5-2` | D1's central premise — that AILANG can express path parsing, set membership and predicate-guarded branching — was argued by analogy to `budget_checker.ail`, which is scalar arithmetic | **Verified → V16.** Wrote the probe and ran `ailang check`: clean. It also found a real constraint the analogy would have hidden — `std/list` has no `all`, so universal quantification is `length(filter(not p, xs)) == 0`. The first probe failed `IMP010` |

The pattern is worth naming, because it is this doc's own subject: **all three objections were
unverified premises, and two of them I had inherited from another document rather than measured.**
That is the same failure as M8's stale park note in §0, committed by the author of the doc
complaining about it.

**Round 1 (2026-09-21, $0.1472): BLOCKED 3/3.** Again all correct; all three are addressed above.

| Reviewer | Objection | Resolution |
|---|---|---|
| `gpt6-astra` | **The plan did not establish the single authority it promised.** V11 names three binary consumers of `ResolveRole` and no phase migrated them, so the shell would gain doc/availability-aware routing while the binary kept a static lookup — the same divergence with the halves swapped | **Design changed: Phase 3b added**, migrating all three `internal/mission/iteration` sites, plus the input question the binary raises (it carries a work item, not a document) resolved fail-closed to the existing `fail-closed:no-doc` token. Added to Goals and Success Criteria — the omission was real and the doc did not do what its title said |
| `gemini-3-1-pro` | **The doc contradicted its own round-0 fix.** D3 was split so POLICY fails closed, but Conflict Surface item 4 still demanded a stale binary "runs the loop on env pins" | **Bullet corrected** to state the split explicitly. A fix that leaves its own counter-example standing elsewhere in the document is not a fix |
| `oc-glm-5-2` | **V16 was measured under `AILANG_RELAX_MODULES=1`** with no justification that production needs no such relaxation, nor that the module surface matches what the Go embed engine exposes | **Re-measured: plain `ailang check`, no flags, passes** once the probe lives at a path matching its module declaration. The relaxation was an artifact of the scratch dir. Insisting on the unrelaxed run surfaced two further constraints (`MOD010` path equality; hyphens illegal in module declarations) that the flag had been hiding |

**The guardrail is now spent.** Round 0 plus one re-quorum is the documented limit, and the skill
is explicit that grinding further rounds is what parks sound designs. Every round-1 objection was
closable in-session and has been closed, with evidence rather than argument. **This doc now goes
to Mark with its status labelled rather than to a third round.** What he is being asked to ratify
is unchanged and is listed in Design Freeze; what changed across two rounds is that four
load-bearing premises are now measured instead of assumed, and one phase that the doc needed in
order to mean what it says has been added.

## Verification Log

Every load-bearing claim, with the command that produced it. All 2026-09-21 against HEAD.

| # | Claim | How verified | Result |
|---|---|---|---|
| V1 | No shell consumes `ailang models role` | `grep -rn "models role" tools/ scripts/` | **0 hits** — confirms the comment at `models_cmd.go:79` is false |
| V2 | Registry expresses three-deep chains | `ailang models role evaluator` / `executor` | 3 rungs each — park blocker CLOSED |
| V3 | Registry expresses the pi harness | `grep -c 'agent_cli: "pi"' internal/modelreg/models.yml` | **12** — park blocker CLOSED |
| V4 | ollama-cloud machinery exists | read `models.go:284` `IsOllamaCloudRoute`, `UsesLocalGPU` | `:cloud`/`-cloud` rows classified non-GPU, so they survive the LaneCloud filter |
| V5 | **NEGATIVE**: no pi×ollama-cloud row exists | parsed every block for `agent_cli: pi` + `agent_model_name: ollama/*` | **4 hits, all local-GPU qwen3.x** — the gap is data, not capability |
| V6 | Registry does NOT yield the `fable` alias | read the `claude-fable-5-1` block | `agent_model_name: "claude-fable-5-1"` (full ID) with the alias trap documented — park blocker CLOSED |
| V7 | `PLANNER_ANTHROPIC_FALLBACK` is predicate-guarded | read `derive-planner-lane.sh:5-17`, `mission-control.sh:1220-1224` | applied ONLY when `MISSION_ANTHROPIC_AVAILABLE=0` — a substitution, not a rung. **The one genuinely open gap** |
| V8 | Planner lane depends on the DESIGN DOC | read `derive-planner-lane.sh` | takes a doc argument; `PLANNER_ALLOWLIST` per-mission since 2026-08-28; `set -f` load-bearing against pathname expansion |
| V9 | Shell surface size | `wc -l` | `derive-planner-lane.sh` 182, `resolve-role-spawn.sh` 79 |
| V9b | **17 distinct reason tokens**, not the 8 an earlier draft of this doc listed | `grep -oE "fail-closed:[a-z-]*\|declared:[a-z-]*\|generator-equals-judge\|anthropic-fallback"` over both scripts, `sort -u` | 19 raw matches → 17 distinct outcomes. Caught by verifying this doc's own claim; recorded because it is the same decay §0 documents, committed by this doc's own author |
| V10 | The AILANG-policy precedent is real and live | read `cmd/ailang/budget.go:178-205`; `ls internal/dashboard_transforms/*.ail` | `budget_checker.ail` is the ONE budget rule, called via `CallPreserveFloats`, silent Go fallback removed. 5 `.ail` transform programs exist |
| V11 | `ResolveRole` consumers | `grep -rn ResolveRole` (non-test) | `models_cmd.go:80`, `coordinator/retry_chain.go:133` (LaneCloud), `iteration/{runtime.go:68, runtime_stage.go:83, retry_review.go:260}` (LaneLocal) |
| V15 | **Cloud agents all carry an explicit `model:` pin** (the premise Phase 0's safety rests on) | parsed `internal/coordinator/testdata/cloud_agents_20260827.json` | **34 agents, 0 unpinned**, 4 role-bearing (one per role). Added after `gemini-3-1-pro` blocked round 0 for asserting this with no log row. **Correction:** an earlier draft said "all 37" — 37 is m-secondary-model-fallback's LIVE-plane count of 2026-09-14, a measurement this doc did not make. The number verified here is 34, from the in-repo fixture snapshot dated 2026-08-27; if the live plane matters for phase 0, re-snapshot rather than reuse either figure |
| V16 | **AILANG can express the policy shape** (D1's load-bearing premise) | wrote `design_docs/planned/m_one_role_table_probe.ail` and ran **plain `ailang check`, no flags** | **✓ No errors found.** Exercises `startsWith` prefix matching, `any` over an allowlist, fail-closed over a path list, nested conditionals and a record result. Added after `oc-glm-5-2` blocked round 0 for asserting AILANG capability by analogy to `budget_checker.ail` (scalar arithmetic) without checking. **Real constraint found: `std/list` exports `any` but NOT `all`** — universal quantification is written `length(filter(not p, xs)) == 0`. The first probe failed with `IMP010: symbol 'all' not exported`, which is precisely the assertion-without-checking this gate exists to catch.

**Measurement condition, after `oc-glm-5-2` blocked round 1 for not justifying it:** the first run used `AILANG_RELAX_MODULES=1` only because the probe sat in a scratch dir, so its `module` declaration could not match its path. Re-run at a repo path whose declaration matches, **plain `ailang check` passes with no flags** — so the relaxation was an artifact of where the file lived, not a condition production depends on. Production's `internal/mission/policy/role_lane.ail` will likewise match its own path and need no flag. Two further constraints surfaced only by insisting on the unrelaxed run: `MOD010` requires the module declaration to equal the file path, and **hyphens are illegal in a module declaration** (legal in directory names) — the probe had to be renamed `m_one_role_table_probe.ail`. The module surface used (`std/string.startsWith`, `std/list.{any,filter,length}`) is ordinary stdlib, identical to what `budget_checker.ail` imports through the same embed engine |
| V12 | The cloud transcription guard reds on a head-model change | observed live 2026-09-21 while developing `5f67aa532` | redded on `design-doc-creator`, went green when the model was kept and only the harness moved |
| V13 | Three incidents, one class | `git log` + `git show` | `7423434b4`, `e9e8ce32e`, `5f67aa532` |
| V14 | M8's park text and reason | read `m-model-registry-single-source-sprint-plan.md:445-470` | parked 2026-08-27, Mark, "if it ain't broken won't fix" |

**Not verified, and named as such:** whether the two new pi×ollama-cloud rows price correctly
against a metered run (no such run exists yet — Phase 0 must derive the rate from one, as
`pi-or-minimax-m3` did, or declare none rather than guess).

## Future Work

- Ledger row 3 (role dispatch to pi) becomes retirable once one table exists.
- Row 2's opt-in (which mission runs `iterate`, under what supervision) is unblocked but separate.
- If D1 lands as AILANG, the admission/ration policy (ledger row 1) is the next natural tenant of the same pattern.
