# M-HARNESS-DSL: AILANG as Self-Hosting Coordinator Specification Language

**Status**: Planned
**Target**: v0.23.0
**Priority**: P2 - Low (research/strategic)
**Estimated**: 2 weeks
**Dependencies**: Effect row type system (shipped), coordinator runtime (shipped), m-harness-state (recommended prerequisite), m-permission-model (recommended prerequisite)

## Axiom Compliance

**Canonical reference:** [Design Axioms](/docs/references/axioms)

### Axiom Scoring

| Axiom | Score | Justification |
|-------|-------|---------------|
| A1: Determinism | +1 | AILANG semantics are deterministic; workflow spec execution is reproducible |
| A2: Replayability | +1 | Workflow spec is a versioned artifact; any execution is replayable |
| A3: Effect Legibility | +1 | Stage effects declared explicitly in workflow spec |
| A4: Explicit Authority | +1 | Role assignments and escalation rules are explicit in spec |
| A5: Bounded Verification | +1 | Workflow specs are type-checked before execution |
| A6: Safe Concurrency | 0 | Concurrent stage execution handled by existing coordinator |
| A7: Machines First | +1 | Machine-interpretable spec replaces natural language instructions |
| A8: Minimal Syntax | 0 | Uses existing AILANG syntax; no new syntax required |
| A9: Cost Visibility | +1 | Stage costs visible in spec; convergence oracle is explicit |
| A10: Composability | +1 | Workflow specs compose with existing AILANG stdlib |
| A11: Structured Failure | +1 | Typed stage failures with escalation rules |
| A12: System Boundary | +1 | Boundary crossings (stage transitions, HITL gates) explicit in spec |

**Net Score: +11** → **Decision: ✅ Proceed**

### Hard Violation Check

- [x] A1 (Determinism): Workflow spec execution is deterministic for given inputs
- [x] A3 (Effects): Stage effects declared; no hidden side effects
- [x] A4 (Authority): Role assignments explicit; no ambient authority
- [x] A7 (Machines First): Spec is machine-interpretable, not natural language

## Problem Statement

AILANG coordinator workflows are currently specified across multiple formats:

- **Design docs** (Markdown, human-readable): what to build and why
- **Sprint plans** (Markdown tables): which milestones, in what order
- **CLAUDE.md** (natural language): session start routines, skill invocations
- **Skill SKILL.md files** (natural language): how to invoke coordinator tools
- **YAML/shell** (ad hoc): specific automation tasks

**Current State:**
- No single executable specification for a coordinator workflow
- The coordinator cannot validate a workflow spec before starting execution
- Role assignments (who is Planner? who is Verifier?) are implicit in natural language
- Effect constraints per stage ("this stage must not write to production") are undeclared
- Convergence oracles ("what command confirms success?") are prose, not typed

**Impact:**
- Coordinator sessions drift from intent; no machine-verifiable alignment between spec and execution
- Adding a new workflow type requires human authorship of natural language instructions
- "Code as Agent Harness" (arXiv:2605.18747) §4 frames this as the frontier: "Natural-Language Agent Harnesses (NLAH) — harness logic written as editable specifications executed by an Intelligent Harness Runtime." The paper frames orchestration as "a runtime interpretation problem." AILANG is uniquely positioned as both the language and the runtime for this.

## Goals

**Primary Goal:** Define a `.ail` workflow spec format that the coordinator runtime interprets, making AILANG the harness specification language for itself — a self-hosting harness. The sprint workflow is the first pilot.

**Success Metrics:**
- Sprint workflow expressible as a single `.ail` file that the coordinator executes without additional natural language instructions
- Coordinator type-checks workflow spec before execution; invalid specs rejected with typed errors
- Workflow spec is version-controlled and diffable like any other `.ail` file
- At least one real sprint completed using a `.ail` workflow spec

## High-Impact Decisions

| Decision | Why High Impact | Chosen By | Deadline | Change Cost |
|----------|-----------------|-----------|----------|-------------|
| Full AILANG vs. restricted DSL subset | Full AILANG enables composability but may be too expressive for safety | human | design | high |
| Coordinator as AILANG interpreter vs. new runtime | Using existing evaluator is elegant; new runtime allows workflow-specific semantics | human | design | high |
| Workflow spec as `main()` entry vs. declared `workflow` keyword | `main()` reuses existing convention; `workflow` enables static analysis | human | design | med |
| Stage isolation: worktrees vs. shared workspace | Worktrees are safer; shared is simpler | agent | compile | med |

### Design Freeze

Before implementation begins:

- [ ] Restricted DSL subset vs. full AILANG confirmed (recommendation: restricted subset for v1, expand later)
- [ ] Interpreter strategy: existing AILANG evaluator (recommended) vs. new runtime

## Solution Design

### Overview

A workflow spec is a `.ail` file with a top-level `workflow` declaration. The coordinator's `WorkflowInterpreter` reads, type-checks, and executes it. Each stage in the workflow maps to a coordinator skill or camp invocation. The spec is the single source of truth for roles, effects, convergence oracles, and escalation rules.

### Workflow Spec Structure

```ailang
module sprint_workflow

-- Role declarations (maps to coordinator camps)
type Role = Planner | Executor | Verifier | Reviewer

-- Stage declaration
type Stage = {
  name:    string,
  role:    Role,
  effects: EffectRow,
  oracle:  () -> bool ! {Exec},  -- convergence test
  on_fail: FailAction
}

type FailAction = Retry Int | Escalate | Abort

-- The workflow
export workflow sprint : [Stage] =
  [ { name    = "plan",
      role    = Planner,
      effects = {},
      oracle  = fun () -> true,   -- plan stage always succeeds (human-reviewed)
      on_fail = Abort }

  , { name    = "execute",
      role    = Executor,
      effects = { FS, LLM },
      oracle  = fun () -> shell "make test",
      on_fail = Retry 2 }

  , { name    = "verify",
      role    = Verifier,
      effects = { Read },
      oracle  = fun () -> shell "make ci",
      on_fail = Escalate }

  , { name    = "review",
      role    = Reviewer,
      effects = {},
      oracle  = fun () -> true,   -- human approval gate
      on_fail = Abort }
  ]
```

### Architecture

**Components:**

1. **WorkflowParser** (`internal/coordinator/workflow_parser.go`): Reads a `.ail` workflow spec, validates it has a `workflow` declaration, and passes it to the existing AILANG type-checker. Rejects invalid specs before execution.

2. **WorkflowInterpreter** (`internal/coordinator/workflow_interpreter.go`): Walks the `[Stage]` list returned by the workflow expression. For each stage: resolves the `Role` to a coordinator camp (using the camp-routing table), dispatches the task, polls the `oracle` function until it returns `true` or `on_fail` fires.

3. **RoleCampRouter** (`internal/coordinator/role_camp_router.go`): Maps `Role` values to actual coordinator camps. `Planner` → sprint-planner skill. `Executor` → claude-agentic or motoko. `Verifier` → eval-only. `Reviewer` → human HITL gate (see m-permission-model).

4. **CLI command** (`cmd/ailang/workflow.go`): `ailang workflow run <spec.ail>`. Validates, then executes. `ailang workflow check <spec.ail>` type-checks only.

### Implementation Plan

**Phase 1: WorkflowParser + Type System Extensions** (~3 days)
- [ ] Define `workflow` keyword as a recognized declaration in the AILANG evaluator (or as a stdlib type alias)
- [ ] `internal/coordinator/workflow_parser.go` — parse and validate `.ail` workflow specs
- [ ] Extend type-checker to recognize `EffectRow` in stage declarations
- [ ] Unit tests: valid/invalid workflow specs

**Phase 2: WorkflowInterpreter + RoleCampRouter** (~3 days)
- [ ] `internal/coordinator/workflow_interpreter.go` — stage dispatch loop
- [ ] `internal/coordinator/role_camp_router.go` — Role → camp mapping table
- [ ] Oracle invocation: `shell` builtin executes oracle command, returns bool
- [ ] Integration test: run a minimal 2-stage workflow spec end-to-end

**Phase 3: Sprint Workflow Pilot** (~2 days)
- [ ] Write `workflows/sprint.ail` — full sprint workflow spec
- [ ] Run one real sprint using the spec; note deviations
- [ ] Update spec based on findings
- [ ] CLI: `ailang workflow run`, `ailang workflow check`

**Phase 4: Documentation** (~1 day)
- [ ] `docs/docs/guides/workflow-dsl.md` — workflow spec format guide
- [ ] Example workflow specs in `workflows/` directory

### Files to Modify/Create

**New files:**
- `internal/coordinator/workflow_parser.go` (~150 LOC)
- `internal/coordinator/workflow_interpreter.go` (~200 LOC)
- `internal/coordinator/role_camp_router.go` (~80 LOC)
- `internal/coordinator/workflow_test.go` (~180 LOC)
- `cmd/ailang/workflow.go` (~80 LOC)
- `workflows/sprint.ail` — pilot workflow spec (~50 LOC)
- `docs/docs/guides/workflow-dsl.md` (~100 LOC)

**Modified files:**
- `cmd/ailang/main.go` — register `workflow` subcommand (~10 LOC)
- `internal/eval/` — add `shell` builtin for oracle evaluation (~30 LOC)

## Examples

### Example 1: Minimal Workflow Spec

```ailang
module hello_workflow

export workflow hello : [Stage] =
  [ { name    = "build",
      role    = Executor,
      effects = { FS, Exec },
      oracle  = fun () -> shell "make build",
      on_fail = Retry 1 }

  , { name    = "test",
      role    = Verifier,
      effects = { Read, Exec },
      oracle  = fun () -> shell "make test",
      on_fail = Abort }
  ]
```

```bash
$ ailang workflow check hello_workflow.ail
✓ Type check passed
  Stages: 2 | Effects: {FS, Exec, Read} | Max tier: T2

$ ailang workflow run hello_workflow.ail
[1/2] build — Executor (claude-agentic) ...
  ✓ oracle: make build → exit 0
[2/2] test — Verifier (eval-only) ...
  ✓ oracle: make test → exit 0
Workflow complete.
```

### Example 2: Type Error in Spec

```bash
$ ailang workflow check bad_workflow.ail
ERROR: Type mismatch in stage "deploy"
  oracle must have type: () -> bool ! {Exec}
  found:                 () -> string ! {Network}

  Hint: oracle must return bool (true = converged, false = retry)
```

### Example 3: Sprint Workflow with HITL Gate

```ailang
module sprint_workflow

export workflow sprint : [Stage] =
  [ { name    = "plan",
      role    = Planner,
      effects = {},
      oracle  = fun () -> true,
      on_fail = Abort }

  , { name    = "execute",
      role    = Executor,
      effects = { FS, LLM },
      oracle  = fun () -> shell "make test",
      on_fail = Retry 2 }

  , { name    = "evaluate",
      role    = Verifier,
      effects = { Read, Exec },
      oracle  = fun () -> shell "make ci",
      on_fail = Escalate }

  , { name    = "review",
      role    = Reviewer,   -- Reviewer role = HITL gate (see m-permission-model)
      effects = {},
      oracle  = fun () -> true,
      on_fail = Abort }
  ]
```

## Success Criteria

- [ ] `ailang workflow check <spec.ail>` type-checks a workflow spec in <1s
- [ ] `ailang workflow run <spec.ail>` executes a 4-stage sprint workflow end-to-end
- [ ] Invalid specs (wrong oracle type, unknown Role) rejected with typed errors before execution
- [ ] One real sprint completed using `workflows/sprint.ail`
- [ ] `workflows/sprint.ail` committed and used as the canonical sprint specification
- [ ] All tests passing (`make test`)
- [ ] Workflow DSL guide published in docs

## Testing Strategy

**Unit tests:**
- `TestWorkflowParser_Valid` — valid specs parse without error
- `TestWorkflowParser_TypeErrors` — invalid oracle types, unknown roles rejected
- `TestRoleCampRouter` — Role → camp mapping is correct for all roles

**Integration tests:**
- `TestWorkflowInterpreter_2Stage` — run a 2-stage workflow against test camps
- `TestWorkflowInterpreter_OracleFail_Retry` — oracle fails twice, retries, then succeeds

**Manual testing:**
- Run `workflows/sprint.ail` against a real sprint plan; confirm all stages execute in order

## Deferred Decisions

- Parallel stage execution (stages with disjoint write-sets could run concurrently) — agent may add in v2
- Workflow spec composition (`import` of reusable stage definitions) — agent may add using existing AILANG module system
- Workflow versioning and migration (what happens when `sprint.ail` changes mid-sprint) — deferred

## Non-Goals

- **Full Turing-complete workflow logic** — the spec should declare stages, not implement them; complex logic belongs in the skills
- **Workflow spec as primary user interface** — design docs and sprint plans remain the human-authored artifacts; the `.ail` spec is the machine-interpretable translation
- **Cross-team workflow sharing** — single-repository assumption for v0.x

## Timeline

**Week 1** (~5 days):
- Phase 1: WorkflowParser + type system (days 1–3)
- Phase 2: WorkflowInterpreter + RoleCampRouter (days 4–5)

**Week 2** (~4 days):
- Phase 3: Sprint workflow pilot (days 1–2)
- Phase 4: Documentation + CLI polish (days 3–4)

**Total: ~9 days**

## Risks & Mitigations

| Risk | Impact | Mitigation |
|------|--------|-----------|
| Workflow spec too restrictive for real sprints | High | Pilot with `workflows/sprint.ail` before generalizing; expand spec format based on findings |
| `shell` oracle creates non-determinism (flaky tests) | Med | Oracle runs are idempotent reads; document that oracle must be side-effect-free |
| Existing sprint workflow too complex to express in one spec | Med | Allow workflow spec to `import` reusable stage definitions from `stdlib/workflow/` |

## Related Documents

**Planned (same cluster — prerequisites):**
- [design_docs/planned/v0_23_0/m-harness-state.md](design_docs/planned/v0_23_0/m-harness-state.md) — Doc 3: workflow interpreter reads harness state per stage
- [design_docs/planned/v0_23_0/m-permission-model.md](design_docs/planned/v0_23_0/m-permission-model.md) — Doc 2: Reviewer role = HITL gate from permission model

**Planned (downstream):**
- [design_docs/planned/v0_23_0/m-trace-feedback.md](design_docs/planned/v0_23_0/m-trace-feedback.md) — Doc 1: diagnostic reports can be triggered per stage by the workflow interpreter

## References

- **Ning et al. (2026).** Code as Agent Harness. arXiv:[2605.18747](https://arxiv.org/abs/2605.18747) — §4 "Natural-Language Agent Harnesses (NLAH)"; "orchestration as a runtime interpretation problem"; PLAN.md / Implement.md as "executable harness specifications"
- [Design Axioms](/docs/references/axioms)
- [Coordinator Guide](../../docs/docs/guides/coordinator.md)
- [Three-Camps Architecture](../../docs/docs/guides/three-camps-self-audit.md)

## Future Work

- **Workflow spec library** (`workflows/` directory): canonical specs for sprint, release, post-release, eval-run workflows
- **Workflow diff**: `ailang workflow diff spec_v1.ail spec_v2.ail` shows what changed between versions
- **Parallel stages**: stages with disjoint write-sets (from m-harness-state) can execute concurrently
- **Self-improving workflow**: trace-feedback diagnostics (m-trace-feedback) propose edits to the `.ail` spec

---

**Document created**: 2026-05-21
**Last updated**: 2026-05-21
