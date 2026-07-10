# M-V1-STABILITY-PROMISE: The 1.x Stable-Surface Promise

**Status**: Planned
**Target**: v1.0.0
**Priority**: P0 (required-for-v1 — explicit clause of the ratified v1.0 bar)
**Estimated**: 2–3 days
**Dependencies**: None (m-diagnostic-coverage's CI-fixture mechanism is reusable but not required)

> **Mission context**: The ratified v1.0 bar ([v1-mission.md](../../v1-mission.md), Mark
> 2026-07-10) requires: *"Stability promise defined: what syntax/stdlib/CLI surface is stable
> in 1.x, written into docs"* and, under the core-frozen clause, *"LIMITATIONS.md accurate"*.
> This doc is that item (mission queue #6). Created iteration 5, headless.

## Axiom Compliance

**Canonical reference:** [Design Axioms](/docs/references/axioms)

This is a docs/policy feature — no language, runtime, or tooling semantics change. Scores are
mostly neutral; the positives come from making the machine-consumed truth surface accurate.

### Axiom Scoring

| Axiom | Score | Justification |
|-------|-------|---------------|
| A1: Determinism | 0 | No language change |
| A2: Replayability | 0 | No trace impact |
| A3: Effect Legibility | 0 | No effect change |
| A4: Explicit Authority | 0 | No capability change |
| A5: Bounded Verification | +1 | A written stable surface makes compat claims locally checkable (a doc'd tier per module/command vs folklore) |
| A6: Safe Concurrency | 0 | No concurrency change |
| A7: Machines First | +1 | LIMITATIONS.md feeds AI prompts/evals; today it teaches models that FIXED constructs are broken (verified below) — accuracy is directly machine-serving |
| A8: Minimal Syntax | 0 | No syntax |
| A9: Cost Visibility | 0 | — |
| A10: Composability | 0 | — |
| A11: Structured Failure | +1 | Every remaining limitation must carry a live-verified repro + date — failures documented as reproducible artifacts, not lore |
| A12: System Boundary | 0 | — |

**Net Score: +3** → **Decision: Move forward**

### Hard Violation Check

- [x] A1 (Determinism): No implicit nondeterminism introduced
- [x] A3 (Effects): No hidden side effects
- [x] A4 (Authority): No ambient access granted
- [x] A7 (Machines First): Accuracy pass is machine-serving, not human-cosmetic

## Problem Statement

Three connected gaps, all verified live on 2026-07-10 at `v0.28.0-139-gece12eab9`:

**1. No stability promise exists anywhere.** No file in `docs/`, `design_docs/`, or the website
defines what a user (human or AI agent) may rely on across 1.x releases. Search performed:
`find design_docs -iname "*stabil*" -o -iname "*promise*"` → only a v0.3-era REPL doc and an
archived phase report. A v1.0.0 without a written compatibility contract is a version number,
not a promise.

**2. `docs/LIMITATIONS.md` is frozen at the v0.13-era and actively wrong.** Last substantive
commit 99f76ec7a (v0.13.0 release; current: v0.28.0). Live verification of its two flagship
"still broken" entries (HARD GATE transcripts, 2026-07-10):

- **"Arithmetic operators panic with polymorphic lambdas"** (claims `add(3.14)(2.71)` →
  `panic: FloatValue, not IntValue`; "Phase 2 deferred to v0.4.2"). **FIXED**:
  ```ailang
  module test/polyarith
  export func main() -> () ! {IO} {
    let add = \x. \y. x + y in
    println(show(add(3.14)(2.71)))
  }
  ```
  `ailang check` → ✓ No errors; `ailang run --caps IO` → `5.85`.
- **"`match` inside block-body lambdas in HOF arguments fails"** (claims parser error, "planned
  v0.8.2"; its example uses the retired `match ... with |` syntax — stale at the syntax level).
  **FIXED** (current syntax):
  ```ailang
  module test/matchhof
  import std/list (map)
  export func main() -> () ! {IO} {
    let result = map(\item. {
      let status = match item { 0 => "zero", _ => "ok" };
      status
    }, [0, 1, 2]) in
    println(show(result))
  }
  ```
  `ailang check` → ✓ No errors; `ailang run --caps IO` → `[zero, ok, ok]`.

  The corresponding design doc was archived (`design_docs/archive/v0_13_0_m-dx-match-in-hof-block-lambda.md`)
  yet the limitation entry survived.

Plus three in-file promises pointing at long-past versions ("planned for v0.4.0+" ×2,
"Deferred to v0.4.2"). **Why this is P0-adjacent and not cosmetic**: LIMITATIONS.md is public,
linked, and exactly the kind of page AI coding agents ingest — it currently teaches that
working constructs are broken and prescribes workarounds for fixed bugs.

**3. Stale version promises live on the public website.** Sweep
(`grep -rniE "(planned for|coming in|will be ...) v0\.[0-9]" docs/`):

| Location | Promise | Repo reality |
|---|---|---|
| `docs/docs/guides/benchmarking.md:14` | "Multi-turn agentic evaluation (M-EVAL2) is planned for v0.3.0" | Agent-mode eval suite exists and is the mission's discrimination concern |
| `docs/docs/guides/module_execution.mdx:250` | "Runtime effect enforcement (capability checks) planned for v0.3.0" | `ailang run --caps` is the shipped, documented mechanism |
| `docs/docs/examples/ai-api-integration.mdx:287` | "JSON decoding is planned for v0.4.0" | `std/json.ail` exists in the stdlib |
| `docs/docs/why-ailang.mdx:268` | "Empirical validation planned for v0.8.0" | The eval program (benchmarks, dashboards, reports) has run for months |

**Impact:** every consumer deciding whether AILANG is trustworthy at 1.0 — humans reading the
site, AI agents ingesting LIMITATIONS.md, and the v1.0 release decision itself (two bar clauses
unmet until this ships).

## Goals

**Primary Goal:** Publish the 1.x stability promise (tiered stable surface for syntax, stdlib,
CLI) and make every statement in the public truth surface (LIMITATIONS.md + version promises)
verifiably accurate at HEAD.

**Success Metrics:**
- A `stability` reference page exists on the website defining tiers and enumerating the surface,
  linked from README and docs intro.
- LIMITATIONS.md: 100% of remaining entries carry a live-verified repro transcript + verification
  date at current HEAD; 0 entries describe fixed behavior as broken; 0 promises reference past
  versions.
- The vX-promise sweep grep returns only roadmap-page and current/future-version references.
- Docs build green; dev CI green on the merge.

## High-Impact Decisions

| Decision | Why High Impact | Chosen By | Deadline | Change Cost |
|----------|-----------------|-----------|----------|-------------|
| Tier taxonomy: **Stable / Experimental / Internal** (three tiers, defined below) | Every subsequent compat question resolves against these definitions | agent (this doc) — ratified by human at release | design | med |
| Semantics of "Stable" in 1.x: no breaking changes to Stable surface in 1.x minors; Experimental may change with changelog notice; deprecations live ≥1 minor before removal | This IS the promise; wrong scope = broken promises or frozen development | **human (Mark) ratifies the final text at release** — draft proceeds headless | compile (before v1.0.0 tag) | high |
| Per-module/per-command tier assignments (39 stdlib modules, ~40 CLI commands) | Determines what users may build on | agent DRAFTS from usage evidence (examples/, eval suite, docs); human ratifies with the promise text | compile | med |
| LIMITATIONS.md entry policy: every entry = live repro + `ailang` transcript + verified-at date; verified-fixed entries move to a dated "Resolved" section (pointer to fix), then age out | Prevents the v0.13-freeze recurring; makes staleness detectable | agent | design | low |
| Promise-sweep recurrence guard: lightweight grep gate (`make check-doc-promises`) flagging past-version promises in docs/ | Time-based staleness hits whoever observes next (mission lesson); a gate makes it CI's job | agent (see Deferred) | runtime | low |

### Design Freeze

Before implementation begins, these must be resolved:

- [x] Tier taxonomy fixed at three tiers: **Stable** (breaking change = major version),
  **Experimental** (may change in minors; marked in docs), **Internal** (no promise; explicitly
  listed so absence isn't ambiguity). Resolved by this doc.
- [x] The promise draft proceeds headless; **human ratification is required before the v1.0.0
  tag, not before this sprint** — the release is human-triggered anyway (mission guardrail), so
  ratification is a release-gate line-item, recorded in the doc header as `RATIFICATION: pending
  (Mark, at release)`. Resolved by this doc; flagged in the #329 report.
- [x] LIMITATIONS.md history is NOT preserved wholesale — fixed entries get a one-line dated
  entry in a "Resolved" section with a pointer (design doc / changelog), full prose deleted
  (testing-policy precedent: remove out-of-date material). Resolved by this doc.

## Solution Design

### Overview

Three milestones, all docs-side (no Go/parser/type code — **no Conflict Surface required**):

1. **M1 — The stability promise page**: new `docs/docs/reference/stability.md` defining the
   tiers, the 1.x semantics, and the enumerated surface (syntax constructs by reference to the
   canonical prompt/reference; stdlib modules tiered in a table; CLI commands tiered in a
   table). Linked from README.md and docs intro.
2. **M2 — LIMITATIONS.md accuracy pass**: re-verify every entry live at HEAD (the two known-fixed
   entries above are pre-verified); rewrite to the entry policy (repro + transcript + date);
   move fixed items to "Resolved"; delete stale version promises; update retired syntax in
   examples.
3. **M3 — vX-promise sweep**: fix the four inventoried website promises (retract-with-pointer
   for the three shipped features; why-ailang.mdx's "empirical validation" line gets replaced
   with a pointer to the actual eval program/dashboard); re-run the sweep grep to zero; add the
   recurrence guard if cheap (Deferred Decision).

### Architecture

**Tier definitions (M1, frozen here):**

- **Stable**: covered by the 1.x promise. Breaking change requires a major version. Candidate
  criteria for the draft: documented in the canonical prompt/reference, exercised by
  `examples/` + eval suite, no open P0 against it.
- **Experimental**: shipped but may change in a minor with a changelog entry. Explicitly marked.
  Likely candidates (draft judgment for the executor, evidence required per module): newer/
  niche stdlib (`std/cognition`, `std/game`, `std/dom`, `std/extension`, `std/sem`,
  `std/sharedmem`, `std/sharedindex`), bytecode VM paths, effect-refinement surface (pending its
  own decomposition, queue #7).
- **Internal**: no promise (e.g. `debug`, `doctor`, eval-harness CLI family, `iface` JSON shape).

**Sources of truth the executor draws from** (no new inventory work): `ls std/` (39 modules),
`ailang --help` (command list), `prompts/` canonical syntax reference, `examples/` coverage,
`docs/docs/roadmap/index.md` (current roadmap page — the one legitimate place for future-version
language).

**LIMITATIONS.md entry template (M2):**

```markdown
### <Title>
**Status**: Open | Design constraint
**Verified at**: v0.28.x (<date>, `ailang check/run` transcript below)
<repro snippet + actual output>
**Workaround**: ...
```

### Implementation Plan

**M1: Stability promise page** (~1 day)
- [ ] Write tier definitions + 1.x semantics (from Design Freeze)
- [ ] Draft stdlib tier table (39 modules) with one-line evidence each
- [ ] Draft CLI tier table with one-line evidence each
- [ ] Syntax: state the Stable set by reference to the canonical reference/prompt version, not by re-enumeration
- [ ] Header carries `RATIFICATION: pending (Mark, at release)` stamp
- [ ] Link from README.md + docs intro; docs build passes

**M2: LIMITATIONS.md accuracy pass** (~1 day)
- [ ] Re-verify every entry live at HEAD (script the repros; keep transcripts in the PR)
- [ ] Move verified-fixed entries (≥2 known: poly-arith lambdas, match-in-HOF) to "Resolved" with pointers
- [ ] Rewrite remaining entries to the template; current syntax only
- [ ] Delete stale version promises (3 in-file)

**M3: Promise sweep** (~0.5 day)
- [ ] Fix the 4 inventoried website promises (retract-with-pointer or re-date)
- [ ] Re-run sweep grep → only roadmap/current references remain
- [ ] (If cheap — see Deferred) `make check-doc-promises` grep gate + CI wiring
- [ ] CHANGELOG.md entry

### Files to Modify/Create

**New files:**
- `docs/docs/reference/stability.md` — the promise page (~250 lines)

**Modified files:**
- `docs/LIMITATIONS.md` — full accuracy rewrite (526 lines → likely shorter)
- `docs/docs/guides/benchmarking.md`, `docs/docs/guides/module_execution.mdx`,
  `docs/docs/examples/ai-api-integration.mdx`, `docs/docs/why-ailang.mdx` — promise fixes (~4×5 lines)
- `README.md`, docs intro — links (~5 lines)
- `CHANGELOG.md` — entry
- (optional) `Makefile` + CI workflow — `check-doc-promises` (~30 lines)

## Examples

### Example 1: LIMITATIONS.md entry, before → after

**Before** (current, wrong at HEAD):
```markdown
**What's Still Broken** (v0.4.0):
-- ❌ Arithmetic operators panic with polymorphic lambdas:
let add = \x. \y. x + y in
add(3.14)(2.71)  -- panic: FloatValue, not IntValue
```

**After**:
```markdown
## Resolved
- **Polymorphic arithmetic lambdas** (open v0.1.0–v0.x): `(\x. \y. x + y)(3.14)(2.71)` now
  evaluates correctly (`5.85`). Verified v0.28.0, 2026-07-10.
```

### Example 2: Website promise, before → after

**Before** (`docs/docs/guides/module_execution.mdx`):
```markdown
**Note**: Runtime effect enforcement (capability checks) is planned for v0.3.0 (M-R2).
```

**After**:
```markdown
**Note**: Runtime effect enforcement is live — capabilities are granted per-run with
`ailang run --caps IO,FS ...` (see the effects guide).
```

## Success Criteria

- [ ] `docs/docs/reference/stability.md` exists: tier definitions, 1.x semantics, full stdlib +
  CLI tier tables, ratification stamp (acceptance: page renders in docs build; every stdlib
  module in `ls std/` and every top-level CLI command appears in exactly one tier)
- [ ] LIMITATIONS.md: every remaining entry has a repro + transcript + verified-at date at HEAD
  (acceptance: PR includes the verification transcripts; the two pre-verified fixed entries are
  in Resolved)
- [ ] Sweep grep (`grep -rniE "(planned for|coming in|will be (added|refined|fixed|implemented)|deferred to) (in )?v0\.[0-9]" docs/ README.md`)
  returns only roadmap-page / legitimately-future references (acceptance: grep output in PR)
- [ ] All tests passing + docs build green (remote CI per Gate 3b)
- [ ] Documentation updated (this IS documentation; CHANGELOG entry present)
- [ ] Examples added — N/A (no language feature; repro snippets live in LIMITATIONS.md)

## Testing Strategy

**Unit tests:** none (docs-only) — unless the executor wires LIMITATIONS repros as CI fixtures
(see Deferred), in which case the m-diagnostic-coverage fixture pattern applies.

**Integration tests:** docs site build (`npm build` via CI Docs-Deploy workflow); sweep grep
re-run as PR evidence.

**Manual testing:** every LIMITATIONS repro executed live at HEAD with transcript captured —
this is the hard gate applied to an entire file.

## Deferred Decisions

The following are intentionally left open for the implementer:

- **Exact per-module/per-command tier assignments** — agent drafts from the evidence sources
  listed above; anything genuinely ambiguous goes in the table with a `⚠ proposed` marker for
  ratification rather than blocking. Agent may resolve (draft); human ratifies at release.
- **`make check-doc-promises` recurrence guard** — include if implementable in ≤30 lines with
  no false positives on the roadmap page; otherwise record as a follow-up line in the promise
  page. Agent may resolve.
- **Wiring LIMITATIONS repros as CI fixtures** (reusing m-diagnostic-coverage's mechanism) —
  nice-to-have; only if it doesn't blow the estimate. Agent may resolve.
- **Whether `docs/LIMITATIONS.md` also gets mirrored/moved into the website docs tree** — agent
  may resolve based on how the site currently links it.

## Non-Goals

**Not attempted in this feature:**
- **Deciding effect-refinement's stability tier semantics** — that surface is still being
  decomposed (mission queue #7); the promise page marks it Experimental-pending and moves on.
- **API stability for Go packages (`internal/`)** — AILANG's Go internals carry no public
  promise; out of scope by definition.
- **A deprecation tooling/mechanism** (compiler warnings for deprecated surface) — post-v1 work;
  the promise only needs the POLICY stated.
- **Re-auditing the language for unknown limitations** — this pass verifies/corrects what's
  WRITTEN; discovering new limitations is the eval program's job.

## Timeline

**Day 1**: M1 (promise page: definitions + tier tables)
**Day 2**: M2 (LIMITATIONS.md re-verification + rewrite)
**Day 3 (half)**: M3 (sweep fixes + optional guard) + CHANGELOG + PR

**Total: ~2.5 days**

## Risks & Mitigations

| Risk | Impact | Mitigation |
|------|--------|-----------|
| Promise text over-commits (something tiered Stable breaks in 1.x) | High | Ratification gate at release (Mark); draft criteria require examples+eval coverage and no open P0s; ambiguous → Experimental (the cheap direction to be wrong in) |
| LIMITATIONS re-verification finds entries that are STILL broken but with changed behavior | Med | That's a finding, not a failure — entry gets updated repro + date; genuinely new bugs route to the backlog, not this sprint |
| Sweep grep false-positives on legitimate roadmap language | Low | Roadmap page is the allowlisted location; guard is optional/deferred if noisy |
| Docs build breakage (mdx) | Low | CI Docs-Deploy is a Gate 3b workflow; fix-forward |

## Related Documents

<!-- Auto-populated by Ollama neural search on "stability promise"; all < 0.45 → no overlap gate triggered -->

**Implemented (may inform design):**
- [m-bugfix-sprint-plan.md](../../implemented/v0_5_8/m-bugfix-sprint-plan.md) (0.40)
- [m-dx26-ensures-result-binding-sprint-plan.md](../../implemented/v0_20_0/m-dx26-ensures-result-binding-sprint-plan.md) (0.37)
- [m-builtin-safety-type-checks.md](../../implemented/v0_7_0/m-builtin-safety-type-checks.md) (0.35)

**Planned (checked for overlap — distinct):**
- [m-diagnostic-coverage.md](../../planned/v0_29_0/m-diagnostic-coverage.md) (0.38) — its CI-fixture mechanism is reusable for LIMITATIONS repros (Deferred Decision)
- [m-oracle-adequacy.md](m-oracle-adequacy.md) (0.38) — eval-side truth, not docs-side
- [m-eval-regression-detector-contract.md](../../planned/v0_29_0/m-eval-regression-detector-contract.md) (0.36)

## References

- [v1-mission.md](../../v1-mission.md) — the ratified bar (stability-promise + LIMITATIONS clauses)
- [Design Axioms](/docs/references/axioms)
- Verification transcripts: this doc's Problem Statement (2026-07-10, v0.28.0-139-gece12eab9)

## Future Work

- Deprecation warnings in the compiler for Experimental→removed surface (post-v1)
- Automated LIMITATIONS freshness check (repros as CI fixtures, full coverage)
- The v1.1 promise review (effect handlers, CSP session types re-enter)

---

**Document created**: 2026-07-10
**Last updated**: 2026-07-10
