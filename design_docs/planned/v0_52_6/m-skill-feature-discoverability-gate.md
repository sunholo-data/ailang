# M-SKILL-FEATURE-DISCOVERABILITY-GATE: a mandatory prompt-load step before agents write .ail

Refs #476

**Status**: Planned
**Target**: v0.52.6
**Priority**: P3 (matches `priority:P3` on the issue)
**Estimated**: 0.5 day (skill markdown edits + one measurement pass on the next showcase)
**Dependencies**: None
**Milestone ID**: M-SKILL-FEATURE-DISCOVERABILITY-GATE
**Source**: [sunholo-data/ailang#476](https://github.com/sunholo-data/ailang/issues/476) (world-coordinator). Triage 2026-10-08 verified the defect live on origin/dev `658ff76a3`.

---

## Problem Statement

World M1 — the flagship agent-built AILANG showcase — shipped **0 of AILANG's distinguishing
features** although all of them existed at the time (and still do): no `requires`/`ensures` Z3
contracts (plain `bool` predicates were used instead, producing **0 proof obligations**), no
real effect shells (`! {}` where `! {IO}` was meant), no inline `tests [(in, out)]`, no package
extensions. The issue's diagnosis, which this doc adopts: the root cause is **discoverability,
not capability**.

The delivery pipeline has a hole that makes this repeatable:

- The `use-ailang` skill names `mcp.ailang.sunholo.com` as the primary source of truth, but that
  MCP is wired into **no** mission repo (issue-verified: `mcpServers` empty in both ailang-world
  and ailang) and there is no `.mcp.json` on dev.
- **Nothing in `sprint-planner` or `sprint-executor` forces the `ailang prompt` fallback before
  an executor writes `.ail` code** — so executors write AILANG from priors and never see the
  feature families the language actually ships.

The active teaching prompt **already contains** everything the showcases missed:

| Missed feature family | Where the active prompt teaches it (`ailang prompt`) |
|---|---|
| Z3 contracts | `## Contracts & Verification (USE THIS — your biggest advantage!)` — `requires { … }` / `ensures { … }` with worked examples |
| Effects | `## Effects (Side Effects Must Be Declared)` — `! {IO, FS, Net, AI, …}` rows |
| Inline tests | `## Testing` — `pure func square(x: int) -> int tests [(0, 0), (5, 25)] { x * x }` (marked "recommended") |

So the gap is not content — it is that **nothing forces the agent to load the content before
writing**. That is the cheapest fix available: a gate step in the two skills that own
plan→execute.

**Impact**: flagship showcases undersell the language's identity (contracts, effects,
verification), every mission-built `.ail` module inherits the same blindness, and there is no
metric to notice.

## Goals

**Primary goal:** make every agent that plans or executes a milestone writing `.ail` load the
current teaching prompt first, and make the three distinguishing feature families an explicit
include-or-justify decision per showcase module.

**Success metrics:**

1. **Feature-use rate** (outcome): fraction of agent-authored showcase modules using ≥1 of
   {`requires`/`ensures` contract, a named effect row (`! {IO}`-style, not `! {}`), inline
   `tests`/property block}. Baseline = 0% (World M1). Target: **≥50% of modules in the first
   showcase generation after landing, ≥80% within three**.
2. **Prompt-load adherence** (process): fraction of showcase sprint sessions where
   `ailang prompt` (or MCP `prompt_get`) was invoked before the first `.ail` write, measured from
   banked transcripts. Target: ≥80% of sessions.
3. **No regression**: `make check-skills` still passes; the sprint skills' other sections are
   untouched.

## High-Impact Decisions

1. **The gate is a skill step, not an MCP wire, not a prompt change, not a CI gate.** Rationale
   in the comparison table below — it is the only option that reaches every harness
   unconditionally, because every mission repo runs the `ailang` binary.
2. **The gate lives in both `sprint-planner` and `sprint-executor`**: the planner records the
   prompt version and the per-module feature checklist in the plan; the executor re-loads the
   prompt and must discharge the checklist. A gate in only one skill is defeated by the normal
   plan→execute split (the executor may be a different session/agent that never reads the plan's
   assumptions — the issue is exactly a cross-session discoverability failure).
3. **Include-or-justify, not mandate-all.** A showcase module that legitimately has no IO and no
   testable pure function must not fail; it must record *why* each family was skipped. The
   checklist's teeth are the recorded justification plus the artifact-level metric, not a
   per-mission veto.
4. **Advisory first, CI teeth deferred.** The issue itself marks the CI anti-vacuity
   feature-usage gate "optional teeth". Ship the advisory step, measure one showcase
   generation, then harden in a follow-up if adherence lags (see Non-Goals).

### Alternatives considered (why the skill step is the cheapest)

| Option | Cost | Reach | Verdict |
|---|---|---|---|
| **Skill gate step** (adopted) | ~26 lines of markdown across 2 skills (×2 mirrored trees, see Implementation) | Every harness that runs the `ailang` binary — which is every mission repo | **Adopted** — cheapest per unit of reach |
| Wire ailang-docs MCP via `.mcp.json` / mission bootstrap | One config file | Only MCP-wired harnesses; measured fact from [m-eval-slim-prompt-self-discovery](../v0_29_0/m-eval-slim-prompt-self-discovery.md): opencode (a real mission harness) does not use the AILANG MCP as a tool source, so `prompt_get` plumbing was "never the path taken". ailang-world already made this fix locally (issue body). | Deferred Non-Goal — owned by the mission/charter template per the issue |
| Add a section to the teaching prompt | Prompt changes are corpus-evidenced; only the one mutable head may be amended, with evidence per feature | Same reach as today (nothing forces loading it) | Rejected — the prompt already teaches all three families (V5); the gap is loading, not content |
| CI anti-vacuity feature-use gate (≥1 contract/test/effect per showcase module) | Script + workflow wiring per showcase repo | Only repos with showcase CI (ailang-world, not this repo) | Deferred — the issue marks it optional; enforce after the advisory step is measured |

## Design Freeze

- **Gate wording is normative but the load channel is flexible**: `ailang prompt` OR MCP
  `prompt_get` when wired. Never hard-require the MCP (offline-tolerant).
- **Version pinning is recorded, not enforced**: the plan/execution report records
  `$(ailang prompt --version-active)`; a mismatch does not block (eval baselines pin their own
  versions by other machinery).
- **No changes to the teaching prompt, the prompt registry, or any Go code.** Markdown-only.

## Solution Design

### Overview

Two skills get one mandatory step each; a metric defined on committed artifacts measures
whether it worked. No code ships in this milestone (a ~40-line optional audit script is
Phase 2, and only if the manual measurement is too tedious).

### The proposed skill steps (concrete text)

**`sprint-planner/SKILL.md`** — inside the planning workflow section, after milestones are
drafted:

> **AILANG Syntax Gate (mandatory when any milestone writes `.ail` code).** For the sprint plan:
> 1. Record `AILANG prompt version loaded: $(ailang prompt --version-active)` (or
>    `prompt_get` if the ailang-docs MCP is wired).
> 2. For every showcase/demo module the plan will create, fill a three-row checklist —
>    **contracts** (`requires`/`ensures`, verifiable via `ailang verify`), **effects** (a named
>    effect row like `! {IO}`, not `! {}`), **inline tests** (`tests [(in, out)]` or a property
>    block): each row is either `include` (with the planned signature detail) or `skip: <one-line
>    reason>`.

**`sprint-executor/SKILL.md`** — in Core Principles (it is a principle, not a phase):

> **Load AILANG syntax before writing `.ail` — no exceptions.** Before the first `.ail` write in
> a session, run `ailang prompt` (or MCP `prompt_get`) and note the version in the milestone
> report. AILANG has Z3 contracts, typed effect rows, and inline tests that models' priors miss —
> writing from memory is how showcases ship without them. For showcase modules, discharge the
> plan's contracts/effects/tests checklist; a `skip` without a recorded reason is a milestone
> failure.

### Architecture

No architecture change. The touched surface is instruction text for machine agents; the skills
are loaded by every harness (Claude Code reads `.claude/skills`, pi/motoko read
`.agents/skills`), and both trees are tracked mirrors of each other (V10).

### Implementation Plan

1. **Step 1** — Edit `.agents/skills/sprint-planner/SKILL.md`: add the AILANG Syntax Gate block
   (~12 lines). Testable: `make check-skills` passes; a read-through shows the step sits in the
   planning workflow, not buried in a resource file.
2. **Step 2** — Edit `.agents/skills/sprint-executor/SKILL.md`: add the Core Principle + gate
   text (~14 lines). Testable: same.
3. **Step 3** — Mirror both edits into `.claude/skills/sprint-planner/SKILL.md` and
   `.claude/skills/sprint-executor/SKILL.md` byte-identically (separate tracked files, same
   content today — V10). Testable: `git diff --stat` shows 4 files; `cmp` per pair.
4. **Step 4** — Run `make check-skills` and commit. Testable: green.
5. **Step 5 (measurement pass, no repo change)** — on the next agent-built showcase generation
   (world mission), compute the feature-use rate (definition below) over its committed modules
   and post the number to issue #476. If manual greps prove tedious, Phase 2 adds
   `scripts/audit_feature_usage.sh`.

### Files to Modify/Create

| File | Change | ~LOC |
|---|---|---|
| `.agents/skills/sprint-planner/SKILL.md` | add gate block | +12 |
| `.agents/skills/sprint-executor/SKILL.md` | add principle + gate | +14 |
| `.claude/skills/sprint-planner/SKILL.md` | mirror | +12 |
| `.claude/skills/sprint-executor/SKILL.md` | mirror | +14 |
| `scripts/audit_feature_usage.sh` (Phase 2, optional) | new audit script with anti-vacuity floor (exit 2 on empty glob) | ~40 |

## Measurement

### Outcome metric: feature-use rate

Per agent-authored showcase module `M` (module created by a mission under a showcase/demo
milestone):

```
feature_use(M) = 1 if M contains ANY of:
  - a contract          : grep -E 'requires \{|ensures \{'
  - a named effect row  : grep -E '! ?\{[A-Z]'        (an effect set with at least one named effect)
  - an inline test      : grep -E 'tests ?\[|test ?\[|property'
feature_use_rate = |{M : feature_use(M) = 1}| / |showcase modules|
```

- **Baseline**: 0% — World M1 shipped 0 of 4 feature families (issue #476; the modules live in
  sunholo-data/ailang-world, so the baseline is the issue's own verified count).
- **Corroborating instrument** (no new code): `ailang verify --json` on each showcase module
  reports proof obligations — the baseline showcase had 0, and the metric is directly the
  "did contracts actually reach Z3" check the issue asks for.
- **Reported**: per showcase generation, to issue #476 (comment) and the world mission log.
  Three data points constitute the trend: first post-change showcase (target ≥50% of modules),
  and the two following (target ≥80%).

### Process metric: prompt-load adherence

The banked `agent_transcript` holds tool calls (see CLAUDE.md's instruments table), so for each
showcase sprint chain: did an `ailang prompt` / `prompt_get` call precede the first `.ail` write?
Prior art for transcript-based usage counting already exists
(`scripts/audit_skill_usage.sh`); the same technique applied to the two commands is a grep, not
a build. Target: ≥80% of showcase sessions within two generations.

### Why this measurement is honest

- It measures the **committed artifact**, not the agent's self-report — an agent can tick the
  checklist without effect only if the `.ail` file itself carries the feature, and greps do not
  care what the report claimed.
- The `skip: <reason>` escape hatch is itself measurable: a module with 0 features and no
  recorded justification is visible as checklist-box-ticking, which is the known failure mode
  of advisory gates and the trigger for the deferred CI teeth.

## Examples

### Before (the World-M1 pattern: compiles, and undersells the language)

```ailang
-- No contract (so `ailang verify` reports "1 without contracts"), no test, and
-- nothing here tells a reader what Z3 could check.
func applyDiscountOK(subtotal: int, pct: int) -> bool
= pct >= 0 && pct <= 100
```

Verified this session: `ailang check` → `✓ No errors found!` (V15).

### After (same function, post-gate plan: contract + inline test + Z3)

```ailang
pure func applyDiscount(subtotal: int, pct: int) -> int
requires { subtotal >= 0, pct >= 0, pct <= 100 }
ensures { result <= subtotal }
tests [((100, 50), 50), ((0, 100), 0)]
{
  subtotal - subtotal * pct / 100
}
```

Verified this session, end to end (V15):

- `ailang check` → `✓ No errors found!`
- `ailang test` → `applyDiscount_test_1 ✓`, `applyDiscount_test_2 ✓`, `All tests passed!`
- `ailang verify` → Z3 4.8.12: `✓ VERIFIED applyDiscount` ("1 verified, 1 without
  contracts"), while the contract-free before-module verifies as `○ NO CONTRACTS` ("1 without
  contracts") — the exact difference the issue's showcase missed

Note for implementers writing similar examples: a function with `tests [...]` must use a
block body `{ … }`; an expression-style `= …` body fails to parse after the tests clause
(`PAR_UNEXPECTED_TOKEN`, V15) — worth remembering because the prompt's testing section shows
only single-arg examples and multi-arg test pairs are `((arg1, arg2), expected)`.

## Success Criteria

- [ ] `sprint-planner` SKILL.md contains the mandatory AILANG Syntax Gate step
- [ ] `sprint-executor` SKILL.md contains the load-before-writing principle + gate
- [ ] `.claude/skills` mirrors updated byte-identically; `make check-skills` green
- [ ] First post-change agent showcase reports a feature-use rate ≥50% of modules
- [ ] Measurement method + first number posted to issue #476
- [ ] All tests passing; no Go code changed in this milestone

## Testing

- `make check-skills` (frontmatter + name/description gate) after each edit.
- `cmp` between each `.agents/skills/…/SKILL.md` and its `.claude/skills` mirror.
- Manual: read both skills end-to-end once — the gate must be findable by an agent that skims
  (Core Principles / planning workflow sections, not a linked resource), because a step an
  executor never loads is the very defect this fixes.
- Edge cases: milestones with no `.ail` output (gate does not apply — condition is explicit);
  harnesses without MCP (CLI fallback is the default path); no compact variant of the active
  prompt exists (V14), so the gate's wording must not promise one.

## Risks & Mitigations

- **Advisory gates get skipped.** The measured trap this repo already knows ("advisory text gets
  skipped — this has been measured", AGENTS.md). Mitigation: threefold — the executor-side step
  sits in Core Principles (always loaded), the metric reads the artifact, and the CI-teeth
  option is pre-designed and deferred only until one measurement says the advisory step lags.
- **Checklist box-ticking.** A `skip` requires a recorded one-line reason; the artifact-level
  feature-use rate cannot be ticked.
- **Context cost of the full prompt.** Measured this session: the active prompt is 97,530 bytes
  on disk, and `ailang prompt --compact` ERRORS for the active version (V14: `"v0.16.6-compact"
  is not a known prompt version`) — so the gate must budget the full prompt, not promise a
  compact escape that does not currently exist. Acceptable: showcase milestones are a small
  fraction of sessions, and the recorded prompt version makes the cost visible in the report.
- **Stale prompt copy.** `ailang prompt` may serve a cached copy; `--source mcp` forces refresh
  when reachable. The recorded version makes staleness visible in the report.

## Non-Goals

- **Wiring `.mcp.json` / mission-bootstrap MCP registration** — issue ask (1). Owned by the
  mission charter/bootstrap template; ailang-world already applied it locally. This doc's gate
  must not depend on it (the m-eval-slim-prompt-self-discovery negative result on
  MCP-as-tool-source).
- **Changing the teaching prompt** — the prompt already covers all three families (V5); prompt
  amendments are evidence-gated by other machinery.
- **A CI hard gate** — issue ask (3), explicitly optional there. Deferred with a design sketch
  (feature_use_rate as a CI assertion) to be picked up if the advisory measurement misses target.
- **Any Go/runtime/compiler change.**

## Related Documents

- [Issue #476 — AILANG feature discoverability gap](https://github.com/sunholo-data/ailang/issues/476) (source of the defect report and the baseline)
- [m-eval-slim-prompt-self-discovery](../v0_29_0/m-eval-slim-prompt-self-discovery.md) — RULED OUT doc; records that opencode never used the MCP as a tool source — the negative result that forces the CLI-first gate design
- [m-agent-mcp-onboarding](../../implemented/v0_15_0/m-agent-mcp-onboarding.md) — the MCP server's onboarding lane (the reach this doc deliberately does not rely on)
- [m-dx-pi-harness](../../implemented/v0_35_0/m-dx-pi-harness.md) — the harness/extension doctrine this repo's agent surfaces ride
- [AGENTS.md](../../AGENTS.md) — work-routing gates; "advisory text gets skipped" measured trap

## Verification Log

| # | Claim | Check performed | Result |
|---|---|---|---|
| V1 | `sprint-planner` SKILL.md contains no prompt-load step (negative-existence; the defect) | `grep -inE "ailang prompt\|prompt_get" .agents/skills/sprint-planner/SKILL.md` | 0 hits — **Confirmed** |
| V2 | `sprint-executor` SKILL.md contains no prompt-load step (negative-existence) | same grep on `.agents/skills/sprint-executor/SKILL.md` | 0 hits — **Confirmed** |
| V3 | No `.mcp.json` on dev (negative-existence) | `ls .mcp.json` at repo root | "No such file or directory" — **Confirmed**. (`mcpServers` empty in both mission repos: issue #476 body, world-coordinator-verified; lives outside this repo, cited not re-run) |
| V4 | Contracts + named effects + inline tests all exist and compile together | Fixture (module with `requires { income >= 0 }` / `ensures { result >= 0 }`, `! {IO}` effect on `main`, `pure func square … tests [(0, 0), (5, 25)]`) through `ailang check` then `ailang test` | `✓ No errors found!`; inline tests ran (`square_test_1 ✓` — 15.8 ms) — **Confirmed** |
| V5 | The active teaching prompt already covers all three missed families | `ailang prompt` piped and inspected (97,530 bytes on disk) | `## Effects (Side Effects Must Be Declared)` (L401), `## Contracts & Verification (USE THIS — your biggest advantage!)` (L2110), `## Testing` with `tests [(0, 0), (5, 25)]` (L2250–2257) — **Confirmed** |
| V6 | `ailang prompt --version-active` exists (the gate records it) | ran it | prints `v0.16.6`, exit 0 — **Confirmed** |
| V7 | `ailang verify` is Z3-backed, local, machine-readable | `ailang verify --help` | flags `-json`, `-strict`, per-function Z3 timeout — **Confirmed** |
| V8 | No prior/planned doc covers this topic (duplicate gate) | `ailang docs search --stream planned/implemented` ×3 queries, SimHash + neural (neural served in `fallback-simhash` mode — noted as a search-quality caveat) | top matches (cli-flags, IFC scoping, string conversions…) all unrelated; nearest on-topic docs cited in Related Documents — **Confirmed, no coverage** |
| V9 | `make check-skills` is the skill-format CI gate | `make/code-health.mk:226` + read `scripts/check_skills.sh` | target exists; checks YAML frontmatter on `.claude/skills/*/SKILL.md` — **Confirmed** |
| V10 | `.claude/skills` is a separate tracked mirror of `.agents/skills` | `git ls-files -s` (same blob hash `fc16320e…` for both sprint-planner SKILL.md paths), `ls -i` (distinct inodes) | mirror, not symlink — **both trees must be edited** — **Confirmed** |
| V11 | Baseline: World M1 shipped 0 of 4 feature families | issue #476 body (source of record; artifacts in sunholo-data/ailang-world, outside this repo) | **Cited** (0 comments on the issue; the body is the full record) |
| V12 | Transcript-based usage auditing has prior art | read `scripts/audit_skill_usage.sh` header | counts skill invocations from session JSONL — technique reusable for the adherence metric — **Confirmed** |
| V13 | `create_planned_doc.sh` could not scaffold this doc | ran it; `bash -x` trace | dies at `IMPLEMENTED=$(merge_results …)` when search returns 0 matches (`set -euo pipefail` + grep exit 1). Scaffolded manually from the skill's structure guide instead. Out of scope here; recorded for the friction log |
| V14 | `ailang prompt --compact` is a usable escape for small contexts (negative-existence check on the gate's cost claim) | ran `ailang prompt --compact >out 2>err` | RC=1, stderr: `Error: "v0.16.6-compact" is not a known prompt version` — the help text advertises `--compact` but no compact variant of the ACTIVE version exists — gate wording must not depend on it — **Confirmed (absence)** |
| V15 | The doc's Before/After showcase example is real, runnable AILANG | both fixtures through `ailang check`; the After fixture also through `ailang test` and `ailang verify`, the Before fixture through `ailang verify` | Before: check `✓ No errors found!`; verify `○ NO CONTRACTS` ("1 without contracts"). After: check `✓ No errors found!`; test `applyDiscount_test_1 ✓`, `applyDiscount_test_2 ✓`, `All tests passed!`; verify: Z3 4.8.12 `✓ VERIFIED applyDiscount` ("1 verified, 1 without contracts"). Also caught while verifying: `tests [...]` demands a block body — expression-style `= …` body → `PAR_UNEXPECTED_TOKEN at 9:3` (the doc's first draft had this bug; fixed). Multi-arg test pairs are `((arg1, arg2), expected)` per `examples/inline_tests_arithmetic.ail:10-19` — **Confirmed** |

No Conflict Surface section: this change touches no
parser/lexer/ast/types/codegen/eval/effects code — it is markdown in instruction files
(V1–V10 cover its entire surface).

## Axiom Compliance

**Canonical reference:** [Design Axioms](../../docs/docs/references/axioms.mdx)

| Axiom | Score | Justification |
|-------|-------|---------------|
| A1: Determinism | 0 | No runtime change; instruction text only |
| A2: Replayability | 0 | No execution-semantics change |
| A3: Effect Legibility | +1 | Expected artifact outcome: showcase code declares real, named effect rows instead of `! {}` — the gate names the effect surface explicitly |
| A4: Explicit Authority | 0 | No caps/authority surface change |
| A5: Bounded Verification | +1 | The gate pushes Z3 contracts, verifiable locally and bounded (`ailang verify`, per-function timeout V7) |
| A6: Safe Concurrency | 0 | No concurrency change |
| A7: Machines First | +2 | The entire change targets the machine agents' instruction surface, and both metrics (feature-use rate, adherence) are computed from machine-readable artifacts |
| A8: Minimal Syntax | 0 | No syntax change |
| A9: Cost Visibility | 0 | Prompt-load context cost is bounded (one prompt per session, version recorded in the report) |
| A10: Composability | 0 | No interface change |
| A11: Structured Failure | 0 | No error-surface change |
| A12: System Boundary | 0 | No boundary change |

**Net score: +4** → **Decision: proceed.** No hard violations (A1/A3/A4/A7 clean).

## Timeline

- **Day 1 (half)**: Steps 1–4 (skill edits, mirrors, `make check-skills`, commit).
- **Next showcase generation**: measurement pass (Step 5), post number to #476.
- **One generation later**: trend check; decide CI-teeth follow-up (Non-Goal → new doc if needed).
