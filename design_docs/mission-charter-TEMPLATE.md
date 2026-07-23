# <MISSION-NAME> Mission — <one-line north-star headline>

<!--
  M-MISSION-PORTABILITY M3 — mission charter TEMPLATE.
  Copy this file to design_docs/<mission-name>-mission.md and fill every <PLACEHOLDER>.
  The FORMAT is fixed (mission-control reads it); the CONTENT is your mission's.
  Iteration 0 = ratify the filled-in charter (bar + queue + guardrails) through the design quorum
  with Mark, exactly as the V1 mission did. See docs/docs/guides/mission-bootstrap.md.
-->

**Type**: Long-running mission (peer of [v1-mission.md](v1-mission.md)); advanced by a scheduled
outer loop on the always-on rig.
**North star**: <what "done" means for this mission — the single sentence the whole loop serves.>
**Traces to**: [PROGRAM.md](PROGRAM.md) — this mission is an operational instance of the program's
loop; every friction found here routes to a lane (skill fix / process fix / backlog item).
**Skill**: [.claude/skills/mission-control/SKILL.md](../.claude/skills/mission-control/SKILL.md)
runs ONE iteration — the SAME unforked skill every mission uses (M-MISSION-PORTABILITY).
**Scheduling**: launchd `dev.ailang.mission-<name>` (StartInterval + overlap guard) behind the
billing guard — API keys stripped (subscription-or-nothing), cheap auth probe first. **STAGGER the
StartInterval offset vs other live missions** (shared rig quota — e.g. V1 on even hours, this
mission on odd) so two loops never fire on top of each other.
**Log**: [<mission-name>-mission-log.md](<mission-name>-mission-log.md) — append-only, one entry per iteration.
**Human-facing reporting**: GitHub issue #<NNN> — every iteration posts its report there as a
comment (Mark follows by email); driver crashes post there too.

## Repo Profile (M-MISSION-PORTABILITY M2 — the per-mission values mission-control reads)

The single source of truth for the values that differ per mission. The one `mission-control` skill
reads this block (and the driver env it exports from `~/.config/ailang/mission-<name>.env`) instead
of hardcoding.

- **Repo slug**: `<owner>/<repo>` (driver: `MISSION_REPO`)
- **Mission doc**: `design_docs/<mission-name>-mission.md` (driver: `MISSION_DOC`)
- **Mission name / state namespace**: `<name>` (driver: `MISSION_NAME`; any name ≠ `v1` gets fully
  namespaced `~/.ailang/state/mission-<name>-*` paths — no collision with the V1 loop)
- **Bookkeeping issue**: `#<NNN>`, rotates weekly; live number in `~/.ailang/state/mission-<name>-gh-issue`
- **CI workflows Gate 3b / Gate 1 poll**: `<WorkflowName1>`, `<WorkflowName2>`, … (must EXIST in the
  target repo — Gate 3b is meaningless without them; hard precondition)
- **Verify profile**: `<go-compiler | ailang-code>` — pick the one that matches the repo:
  - `go-compiler` — the repo compiles a Go toolchain: `make quick-install && make build`, `make test`,
    dual-binary staleness rules.
  - `ailang-code` — the repo is AILANG source: the shipped `ailang` binary IS the gate —
    `ailang check` (types), `ailang test` (tests), `ailang ai-check --json` (unified check+verify:
    types + Z3 in one JSON — do not reinvent a split gate). No compile step, no `-dirty` staleness.

---

## STATUS (rotation rule)

Newest **3** STATUS stamps live here; older ones move to `<mission-name>-mission-status-archive.md`.
At Gate 4, after adding your stamp, move the now-4th stamp to the TOP of the archive file. Rationale:
every iteration re-reads this charter — unbounded STATUS history is a per-read token tax on the
scarcest model budget; the append-only history lives in the log + archive.

## STATUS <YYYY-MM-DD> — ITERATION 0: **charter ratification** — <fill after iteration 0>

## CURRENT GOAL

1. **Iteration 0 (definition)**: ratify the bar (below) with Mark through the design quorum, then
   score the backlog against it into `required` / `nice` / `post`. Output: ordered queue in this doc.
2. **Then**: work the queue through the inner loop (design-doc → sprint-plan → execute → evaluate),
   one sprint-sized item per iteration, recording routing evidence every time.

## The bar — <what a release/goal must meet> (RATIFY with Mark at iteration 0)

<!-- The concrete, checkable clauses that define "done". Written down, met, and verified.
     Number them (clause 1, clause 2, …) so queue items can clause-tag against them. -->

- **Clause 1**: <…>
- **Clause 2**: <…>

## Guardrails (mission-specific; the skill's Standing Rules always apply on top)

<!-- Anything this mission must never do / must always do, beyond the shared Standing Rules.
     Keep it short; the shared discipline (billing tripwire, one-item-per-iteration, bounded waits,
     data-before-conclusions, generator≠judge) is in SKILL.md and needs no restatement. -->

- <…>

## Routing policy

Uses the **shared** per-role model routing from `mission-control` (controller / designer-rotation /
planner / executor / evaluator, generator≠judge enforced). Overrides for THIS mission, if any, go in
`~/.config/ailang/mission-<name>.env` (e.g. a non-Anthropic executor default to spread quota):

- **Executor default**: `<MISSION_EXECUTOR_MODEL — e.g. codex:gpt-5.6-sol to avoid doubling Anthropic burn>`
- **Evaluator**: `<must differ in provider from the executor — generator≠judge>`
- <other overrides, or "none — inherits the shared defaults">

## Queue (top = next; tags: [NEXT] [IN-SPRINT] [PARKED] [LANDED] [RULED OUT])

<!-- Ordered backlog. Every open item carries a clause tag against the bar. NEW-DOC items start with
     design-doc-creator; existing-doc items start at Gate-2 reality-check. -->

1. [NEXT] <first item — id · clause-tag · one-line scope · est.>
2. …

---
**Document created**: <YYYY-MM-DD>. Iteration 0 ratifies it via the quorum with Mark before any
sprint routes.
