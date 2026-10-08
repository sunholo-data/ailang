# Sprint-state `estimated_loc` sentinel migration: `0` → `null`

**Refs #563** (sunholo-data/ailang, P3, open; tracks the issue — no new issue filed)

**Status**: Implemented
**Target**: v0.52.6
**Priority**: P3 (Low — matches the issue's `priority:P3` label)
**Estimated**: 0.5 day (two-line script edits ×2 trees, schema/doc text, fixture validation)
**Dependencies**: None

## Axiom Compliance

**Canonical reference:** [Design Axioms](/docs/references/axioms)

This is harness/skill tooling (Bash + jq + Python inside `.claude/skills/` and `.agents/skills/`), not a
language change — most axioms are untouched. The two it does move are the ones the defect violates.

### Axiom Scoring

| Axiom | Score | Justification |
|-------|-------|---------------|
| A1: Determinism | 0 | No change to any execution semantics; sentinel encoding only |
| A2: Replayability | 0 | No impact on traces |
| A3: Effect Legibility | 0 | No effects involved |
| A4: Explicit Authority | 0 | No capability surface touched |
| A5: Bounded Verification | +1 | Validator check becomes locally testable with plain jq fixtures (see Testing Strategy) |
| A6: Safe Concurrency | 0 | No concurrency |
| A7: Machines First | +1 | Distinct sentinel (`null`) instead of an overloaded value (`0`) — a machine can now tell "unset" from "net-zero by design" without reading a comment |
| A8: Minimal Syntax | 0 | No syntax; JSON schema gains one legal value |
| A9: Cost Visibility | 0 | No cost surface change |
| A10: Composability | 0 | Same consumers, same field |
| A11: Structured Failure | +1 | The validator now fails on the *right* condition (unfilled) instead of a legitimate value (net-zero), and its error message will name the actual sentinel |
| A12: System Boundary | 0 | No boundary crossings |

**Net Score: +3** → **Decision: Move forward**

### Hard Violation Check

**These axioms cannot have −1 scores (automatic rejection):**

- [x] A1 (Determinism): No implicit nondeterminism introduced
- [x] A3 (Effects): No hidden side effects
- [x] A4 (Authority): No ambient access granted
- [x] A7 (Machines First): Not optimizing for human convenience over machine analysis

### Decision Thresholds

| Net Score | Decision |
|-----------|-----------|
| ≥ +2 | ✅ Proceed to implementation |
| 0 to +1 | ⚠️ Needs stronger justification |
| < 0 | ❌ Reject or redesign |
| Any −1 on A1/A3/A4/A7 | ❌ Automatic rejection |

## Problem Statement

The sprint-state validator treats `estimated_loc == 0` as "the planner never filled this in", but `create_sprint_json.sh` *writes* `0` as its placeholder — and `0` is also a legitimate, honest estimate for exactly the milestone classes this repo's own process encourages (pure refactors under the `make check-file-sizes` 800-line gate; docs-only milestones). The overload is recorded twice in production (mission iterations 121 and 133) and each occurrence hard-blocks sprint execution and costs a controller round-trip.

**Current State:**
- `validate_sprint_json.sh:123-124` (both `.claude` and `.agents` copies, byte-identical):
  ```bash
  # Note: estimated_loc == 0 is the placeholder from create_sprint_json.sh (not 200 - that's a valid real estimate)
  INCOMPLETE_MILESTONES=$(jq -r '.features[] | select(.description == "Milestone description" or .estimated_loc == 0) | .id' "$PROGRESS_FILE")
  ```
- `create_sprint_json.sh` writes the `0` sentinel on **two** paths, not one:
  - `:130` — the markdown parser: `estimated_loc = int(heading_match.group(3)) if heading_match.group(3) else 0` (heading without a `(~NNN LOC)` clause → `0`);
  - `:165` — the no-milestones fallback template: `"estimated_loc": 0`.
- The comment at `:123` acknowledges the ambiguity is deliberate, which makes the wrongness self-documenting but no less blocking.
- **Live repro on this checkout (`1dfd5615`, 2026-10-08; triage independently verified on `origin/dev` `658ff76a3` same day)**: a fixture sprint whose only unusual property is a legitimate net-zero milestone (`M1_FILE_SPLIT`, "behaviour-free split … `git diff -M` must show moves only", `estimated_loc: 0`, real description, real criteria, populated `registry_reuse`) fails validation with `VALIDATION FAILED: 1 error(s)` and exit 1.

**Impact:**
- Every planner/executor pair for a refactor-only or docs-only milestone (both classes are process-mandated: `make check-file-sizes` forces refactor-first milestones; every change requires CHANGELOG/design-doc updates).
- The failure direction is the worst available: the executor refuses to start and the message points the reader back at a plan that is already correct ("sprint-planner must populate the JSON with real milestone data").
- Workaround damage: iteration 133 encoded `245` (lines *moved*) with an explanatory note — accurate but undiscoverable by the next planner; the honest number is inexpressible.

## Goals

**Primary Goal:** Make "unset" and "zero" distinct encodings in the sprint-state placeholder protocol, so a legitimately net-zero milestone validates and an unfilled placeholder still fails.

**Success Metrics:**
- A sprint JSON whose milestone carries `estimated_loc: 0` with real description/criteria passes `validate_sprint_json.sh` with exit 0 (fixture-verified).
- A milestone carrying `estimated_loc: null` **or** omitting the key entirely fails validation with exit 1 and its ID is named in the error (fixture-verified).
- `create_sprint_json.sh` emits `null` (JSON) from both placeholder paths; a plan heading with no `(~NNN LOC)` clause yields `null`, not `0`.
- All four touched scripts remain byte-identical between `.claude/skills/` and `.agents/skills/` after the change.
- No historical `.ailang/state/sprints/*.json` file is rewritten by this change.

## High-Impact Decisions

| Decision | Why High Impact | Chosen By | Deadline | Change Cost |
|----------|-----------------|-----------|----------|-------------|
| Sentinel is JSON `null`, written explicitly by the creator | Defines the placeholder protocol every planner/executor pair speaks; `null` is jq-natural (`== null` matches it) and cannot collide with any real estimate | agent (this doc) | design | med |
| Validator treats **both** `null` and an absent key as placeholder (`.estimated_loc == null`), not null-only | An absent key must not silently pass as "filled"; jq's `== null` catches both forms in one token — verified live below | agent (this doc) | design | low |
| `0` becomes a first-class valid estimate in the validator | This is the semantic change that un-blocks net-zero milestones; nothing downstream numerically consumes `estimated_loc` except display | agent (this doc) | design | low |
| No migration of historical state files; audit-only for in-flight sprints | 16 of 190 existing state JSONs carry `"estimated_loc": 0`; they are append-only execution records — rewriting history is out of scope, but unfilled `0`s in *non-completed* sprints would become invisible to the new validator | agent (this doc) | design | med |
| Fix lands in **both** skill trees simultaneously | `.claude` and `.agents` copies are byte-identical today (verified); #544's divergence is closed — landing in one tree only would reopen it | agent (this doc) | design | low |

### Design Freeze

Before implementation begins, these must be resolved:

- [x] Sentinel = explicit JSON `null` from the creator; validator accepts `null` **or** absent as placeholder (decided above — jq `== null` covers both)
- [x] Historical state files stay untouched; in-flight audit is a read-only planner check (decided above)
- [x] `estimated_total_loc == 1000` default stays as-is (warning-only path, and `TARGET_LOC_PER_DAY` arithmetic divides it — see Non-Goals)

## Solution Design

### Overview

Migrate the sprint-state placeholder protocol for `estimated_loc` from the overloaded numeric `0` to JSON `null`, in the two scripts that write and police it, in both skill trees, with matching schema/doc text. The string sentinels in the same validator (`description == "Milestone description"`, `id == "MILESTONE_ID"`, criteria `"Criterion 1"/"Criterion 2"`) are already collision-free — a string sentinel cannot equal a real value — and are the pattern being copied; they stay.

### Architecture

**Components:**
1. **Creator (`create_sprint_json.sh`, both trees)** — the two placeholder paths stop writing `0`:
   - `:130` (markdown parse): `estimated_loc = int(heading_match.group(3)) if heading_match.group(3) else 0` → `… else None`
   - `:165` (no-milestones fallback): `"estimated_loc": 0` → `"estimated_loc": None`
   - `python3 json.dumps` serializes `None` as `null` (verified live: `{"estimated_loc": null}`).
   - A *parsed* milestone whose heading *does* carry `(~NNN LOC)` keeps writing that integer, `0` included — the parser never fabricates a number.
2. **Validator (`validate_sprint_json.sh`, both trees)** — `:123-124`:
   ```bash
   # Note: estimated_loc == null (or absent) is the placeholder from create_sprint_json.sh.
   # 0 is a VALID real estimate — a net-zero milestone (pure refactor, docs-only) is legitimate.
   INCOMPLETE_MILESTONES=$(jq -r '.features[] | select(.description == "Milestone description" or .estimated_loc == null) | .id' "$PROGRESS_FILE")
   ```
   jq semantics (verified live on jq 1.7.1): `.estimated_loc == null` is `true` for an explicit `null` **and** for an absent key; `false` for `0` and `245`.
3. **Display (`session_start.sh:164`, both trees)** — cosmetic co-traveler: `"(estimated: \(.estimated_loc) LOC)"` would render `null` for an unstarted placeholder milestone. Guard with jq's alternative operator: `(.estimated_loc // "unset")` — note `//` does **not** treat `0` as empty, so real zero estimates still print as `0`.
4. **Schema/doc text (`json_progress_schema.md`, both trees)** — `"estimated_loc": "number"` → `"number | null"` with one sentence: *`null` (or absent) means the planner has not filled it in; `0` is a valid net-zero estimate.*

### Implementation Plan

**Phase 1: Sentinel migration** (~2 hours)
- [ ] Edit `create_sprint_json.sh` `:130` and `:165` in `.claude/skills/sprint-planner/scripts/` (→ `None`)
- [ ] Same two edits in `.agents/skills/sprint-planner/scripts/create_sprint_json.sh`
- [ ] Edit `validate_sprint_json.sh` `:123-124` in `.claude/skills/sprint-executor/scripts/` (→ `.estimated_loc == null`, comment rewritten)
- [ ] Same edit in `.agents/skills/sprint-executor/scripts/validate_sprint_json.sh`
- [ ] `session_start.sh:164` display guard (`// "unset"`) in both trees
- [ ] `json_progress_schema.md` field type + one-sentence semantics, both trees

**Phase 2: Fixture validation** (~2 hours)
- [ ] Fixture matrix (see Testing Strategy) run against both trees' validators; record exit codes
- [ ] Regression control: run the patched validator against 2–3 recent *completed* repo sprint JSONs (e.g. `.ailang/state/sprints/sprint_M-EVAL-ELO-PERSIST.json`) — must stay exit 0
- [ ] `diff` the four scripts across the two trees — must be empty

**Phase 3: In-flight audit + docs** (~1 hour)
- [ ] Read-only audit of non-completed state sprints carrying `estimated_loc: 0` (see Migration); record the ruling per sprint in the sprint JSON's `notes` — planner-owned
- [ ] Update this doc's status → implemented via `move_to_implemented.sh` after landing

### Files to Modify/Create

**New files:**
- None (fixtures are throwaway probe files, not committed; the validator takes `<sprint_id>` and reads a fixed relative path, so fixtures live in a temp working dir — mirroring how this doc's verification was run)

**Modified files:**
- `.claude/skills/sprint-planner/scripts/create_sprint_json.sh` — 2 lines (`:130`, `:165`), ~2 LOC
- `.agents/skills/sprint-planner/scripts/create_sprint_json.sh` — same, ~2 LOC
- `.claude/skills/sprint-executor/scripts/validate_sprint_json.sh` — `:123-124`, ~3 LOC (comment + predicate)
- `.agents/skills/sprint-executor/scripts/validate_sprint_json.sh` — same, ~3 LOC
- `.claude/skills/sprint-executor/scripts/session_start.sh` — `:164` display guard, ~1 LOC
- `.agents/skills/sprint-executor/scripts/session_start.sh` — same, ~1 LOC
- `.claude/skills/sprint-executor/resources/json_progress_schema.md` — field type + semantics sentence, ~3 LOC
- `.agents/skills/sprint-executor/resources/json_progress_schema.md` — same, ~3 LOC

## Examples

### Example 1: A legitimately net-zero milestone (the iteration-133 case)

**Before** (inexpressible — executor refuses to start):
```json
{
  "id": "M1_SPLIT_AI_STEP_FOR_HEADROOM",
  "description": "Behaviour-free split of ai_step.go into ai_decode.go + ai_encode.go; git diff -M must show moves only",
  "estimated_loc": 0,
  "acceptance_criteria": ["git diff -M --stat shows moves only", "make test-core green"]
}
```
```
ERROR: Milestones with default/placeholder values:
VALIDATION FAILED: 1 error(s) found     ← live-reproduced on this checkout, exit 1
```

**After** (valid — the estimate is honest and the validator agrees):
```json
{ "id": "M1_SPLIT_AI_STEP_FOR_HEADROOM", "description": "…same…", "estimated_loc": 0, … }
```
```
✓ All milestones have custom values
VALIDATION PASSED: Sprint JSON is ready for execution   ← live-verified on patched predicate, exit 0
```

### Example 2: The planner genuinely forgot to fill it in

**Before and after the fix, the validator correctly blocks both placeholder forms:**
```json
{ "id": "M1", "description": "Split ai_step files", "estimated_loc": null, … }   ← explicit null → exit 1
{ "id": "M1", "description": "Split ai_step files", … }                            ← key absent   → exit 1
```
```
ERROR: Milestones with default/placeholder values:
M1
VALIDATION FAILED: 1 error(s) found     ← live-verified on patched predicate, exit 1
```

### Example 3: What `create_sprint_json.sh` writes when the plan heading has no `(~NNN LOC)` clause

**Before:** `"estimated_loc": 0` — indistinguishable from a net-zero estimate.
**After:** `"estimated_loc": null` — the validator flags it; the planner fills a real number (which may honestly be `0`).

## Success Criteria

- [ ] Fixture: milestone with `estimated_loc: 0` + real description/criteria → `validate_sprint_json.sh` exit **0** (acceptance: run recorded in the implementation report)
- [ ] Fixture: `estimated_loc: null` → exit **1**, milestone ID named in the error
- [ ] Fixture: key absent → exit **1**
- [ ] Fixture: `estimated_loc: 245` (real estimate) → exit **0** (unchanged behavior)
- [ ] `create_sprint_json.sh` parse-without-LOC path and no-milestone fallback both emit `null` (verified by generating a sprint JSON from a minimal plan)
- [ ] `diff` of each modified script between `.claude/skills/` and `.agents/skills/` is empty
- [ ] Regression: patched validator exits 0 on recent completed repo sprint JSONs
- [ ] No `.ailang/state/sprints/*.json` historical file modified by this change
- [ ] All tests passing (`make test-core` unaffected — no Go touched)
- [ ] Documentation updated (`json_progress_schema.md` both trees; this doc moved to implemented)

## Testing Strategy

**Unit tests (bash fixtures, both trees):**
- Four-fixture matrix in a temp working dir with `.ailang/state/sprints/sprint_<ID>.json`: net-zero (`0`), explicit `null`, absent key, real estimate (`245`) — all other fields genuinely populated (real description, real criteria, populated `registry_reuse`) so the *only* variable is the sentinel form. Expected exits: 0, 1, 1, 0.
- Creator probe: run `create_sprint_json.sh` against a minimal plan with (a) a heading lacking `(~NNN LOC)` and (b) no `### M` headings at all; assert `"estimated_loc": null` in the emitted JSON for both.
- jq predicate probe (independent of the scripts): `echo '{"a":{"id":"M1"}}' | jq '.estimated_loc == null'` → `true` for absent; same for explicit `null` → `true`; `0` → `false`.

**Integration tests:**
- Regression control: patched validator over 2–3 recent completed sprint JSONs from `.ailang/state/sprints/` — exit 0, no new warnings.

**Manual testing:**
- Full round trip once: `create_sprint_json.sh` → hand-edit one milestone to `estimated_loc: 0` (net-zero, real description) → `validate_sprint_json.sh` passes → `session_start.sh` displays `(estimated: 0 LOC)`, and an unfilled `null` milestone displays `(estimated: unset LOC)`.

## Deferred Decisions

The following are intentionally left open for the implementer:

- Exact wording of the rewritten `:123` comment — agent may choose, provided it states both facts (null/absent = placeholder; 0 = valid net-zero).
- Whether the `session_start.sh` display guard also renders `unset` in a distinct color — agent may choose; cosmetic only.
- Fixture harness shape (one throwaway script vs. inline commands in the implementation report) — agent may choose; the acceptance requirement is recorded exit codes, not a committed test file.

## Non-Goals

**Not attempted in this feature:**
- `velocity.estimated_total_loc == 1000` default — **warning-only** in the validator (never blocks), and `TARGET_LOC_PER_DAY=$((ESTIMATED_TOTAL_LOC / ESTIMATED_DAYS))` does integer arithmetic on it; migrating it to `null` would need a second arithmetic-path design for no recorded friction. Deliberately left as the numeric default + warning. (If a third sentinel friction arrives, route it here.)
- Unifying the `.claude/skills/` ↔ `.agents/skills/` duplication (#544) — that issue is **closed** and the copies are byte-identical today; this design *preserves* that invariant rather than resolving the duplication.
- Any schema validator/tooling beyond the two scripts (no JSON Schema file exists for sprint state; introducing one is a separate feature).
- Rewriting historical state files (see Migration below for why audit-only).

## Timeline

**Week 1** (5 hours total — single half-day):
- Phase 1: Sentinel migration (2 h)
- Phase 2: Fixture validation + regression controls (2 h)
- Phase 3: In-flight audit + doc move (1 h)

**Total: ~5 hours across 1 week** (P3 — rides alongside other work; no release dependency)

## Risks & Mitigations

| Risk | Impact | Mitigation |
|------|--------|-----------|
| A sprint created pre-fix with an *unfilled* `0` placeholder would now pass validation silently | Med | Read-only audit of the 16 state files carrying `0` (done in this doc, below): most are `completed`; the non-completed ones get a planner ruling recorded in `notes`. Window is small: validation gates executor start, so only sprints not yet executed matter. |
| A future editor reintroduces `== 0` muscle-memory | Low | The rewritten `:123` comment states the contract explicitly; `json_progress_schema.md` documents `null` semantics beside the field type |
| `.claude`/`.agents` copies drift during the edit | Med | Land both trees in one commit; Phase 2's `diff` gate is a Success Criterion |
| Historical references (e.g. `m-planner-codex-lane.md` L23 backstop credit, mission-log entries) describe the old `0` sentinel | Low | They are dated records of past behavior — correct at their timestamp; leave untouched |

## Migration

**Historical state files: no rewrite.** `.ailang/state/sprints/*.json` are append-only execution records of sprints that already ran; changing them would falsify history and buy nothing (validation gates *executor start*, not retroactive audits).

**In-flight sprints: read-only audit.** Of 190 state files, 16 contain `"estimated_loc": 0`. Status audit (this session):

- 12 are `completed`/`complete` — closed history, untouched.
- 4 are non-completed (`M-CHAINS-EXECUTOR-TRANSCRIPTS` `not_started`; `M-LIST-ACCESSOR-API` `not_started`; `M-MISSION-AGENTIC-ROUTING` `in_progress`; `M-DX27-DOCS-SEARCH-GITHUB-FALLBACK` `planned`, empty features). For each milestone carrying `0`, the planner records a one-line ruling in the milestone's `notes`: *"0 = net-zero by design"* (keep) or *"was unfilled placeholder"* (set `null`). Probe command:
  ```bash
  jq -r '.features[] | select(.estimated_loc == 0) | "\(.id): \(.description)"' .ailang/state/sprints/sprint_<ID>.json
  ```

**Docs:** `json_progress_schema.md` (both trees) is the normative statement of the new protocol; `sprint-planner/SKILL.md`'s example (`"estimated_loc": 150`) already shows a filled value and needs no change.

## Related Documents

<!-- Auto-search (SimHash + neural) found no matches on "sprint loc null sentinel" — the scaffold script's
     index is empty in this checkout. Manually curated by grep; the duplicate/coverage gate was applied to these: -->

**Implemented (may inform design):**
- [m-planner-codex-lane.md](../v1_0_0/m-planner-codex-lane.md) — L23 records this exact validator line as a *partial downstream backstop* for the planner's placeholder path, with the `estimated_loc == 0` overload already flagged as a watch-item. This design resolves that watch-item; the L23 row itself is a dated record and stays as written.
- [v1-mission-log-archive.md](../../v1-mission-log-archive.md) — `:6398` (iteration 121, docs-only M5 refused; estimator set to 80 as workaround) and `:7327`/`:7347` (iteration 133, file-split milestone; `245` workaround; #563 routed to backlog). Both instances' workaround values are exactly what this design makes unnecessary.

**Planned (check for overlap):**
- (none — no planned doc touches the sprint-state placeholder protocol; grep over `design_docs/planned/` for `estimated_loc` returns only unrelated sprint-plan JSONs and a JSON-precedent row)

## References

- Issue #563 — sunholo-data/ailang — "sprint-state validator: estimated_loc == 0 is used as the 'unfilled placeholder' sentinel…" (open, P3, `area:mission`; read in full including zero comments)
- Issue #544 — closed — the `.claude`/`.agents` skill-tree divergence this design must not reopen
- [Design Axioms](/docs/references/axioms) — the 12 non-negotiable principles
- [m-planner-codex-lane.md](../v1_0_0/m-planner-codex-lane.md) — the planner lane whose placeholder path this validator backstops

## Verification Log

| # | Claim | Verification | Result |
|---|-------|--------------|--------|
| V1 | `validate_sprint_json.sh:124` rejects a *legitimate* net-zero milestone | Live repro on this checkout (`1dfd5615`), 2026-10-08: fixture sprint `SPLITTER` with real description, real criteria, populated `registry_reuse`, `estimated_loc: 0` → `VALIDATION FAILED: 1 error(s)`, exit 1. (Triage independently verified on `origin/dev` `658ff76a3`.) | Confirmed |
| V2 | jq `.estimated_loc == null` is `true` for explicit `null` **and** an absent key; `false` for `0` and `245` | Live probe, jq 1.7.1: four single-object fixtures printed `false, true, true, false` in the order (0, null, absent, 245) | Confirmed |
| V3 | The patched predicate passes the net-zero fixture and still rejects a real placeholder | Live probe: `sed`-patched copy of the validator → `SPLITTER` fixture exit **0** (PASSED); same fixture with `estimated_loc` set to `null` → exit **1**, `M1_FILE_SPLIT` named in the error | Confirmed |
| V4 | `create_sprint_json.sh` writes the `0` sentinel on two paths | Code read: `:130` `int(...) if heading_match.group(3) else 0` (no `(~NNN LOC)` clause → 0) and `:165` `"estimated_loc": 0` (no-milestones fallback) | Confirmed |
| V5 | `python3` serializes `None` to JSON `null` | Live probe: `json.dumps({'estimated_loc': None})` → `{"estimated_loc": null}` | Confirmed |
| V6 | The four affected scripts are byte-identical between `.claude/skills/` and `.agents/skills/` today | `diff` of `validate_sprint_json.sh` and `create_sprint_json.sh` across both trees → empty, both rc 0 | Confirmed |
| V7 | The validator is the only *blocking* consumer of the sentinel | `grep -rn "estimated_loc" --include='*.sh' --include='*.md' --include='*.go' .` (repo-wide, excluding git/state/history): consumers are the validator (blocks), `session_start.sh:164` (display-only interpolation), `create_sprint_json.sh` (writer), and example/schema text. No Go code, no numeric consumer. | Confirmed |
| V8 | 16 of 190 state sprint JSONs carry `"estimated_loc": 0`; statuses enumerated | `grep -l` + per-file `python3` status read; 12 completed/complete, 4 non-completed (named in Migration) | Confirmed |
| V9 | #544 (skill-tree divergence) is closed, not open | GitHub API read of issue #544: `state: closed` | Confirmed |
| V10 | `session_start.sh`'s jq `//` alternative does not swallow `0` | Live probe, jq 1.7.1: `{"estimated_loc":0} \| .estimated_loc // "unset"` → `0`; same on `null` → `"unset"` | Confirmed |

## Future Work

- **`create_planned_doc.sh` aborts on an empty search index** — observed while scaffolding this doc, out of scope here: `merge_results`' `grep -E "^[0-9]+\."` exits 1 when both search result sets are empty, and under `set -euo pipefail` the assignment `IMPLEMENTED=$(merge_results …)` kills the script *after* printing the search header but *before* creating the template. Any agent scaffolding a doc in a checkout without a search index hits this. Recorded here because it is the same sentinel-adjacent harness family; it deserves its own issue if it bites twice.
- A formal JSON Schema (machine-checkable) for sprint state, replacing prose schema docs — separate feature, blocked on nothing.
- `estimated_total_loc: 1000` numeric-default migration — only if a third sentinel friction is recorded (see Non-Goals).

---

**Document created**: 2026-10-08
**Last updated**: 2026-10-08

## Implementation Report

**Refs #563** — implemented 2026-10-08 on `coordinator/task-6f7a887f`.

Both skill trees now emit null on both creator placeholder paths, reject null/absent
estimates, accept numeric zero, display unset estimates as `unset`, and document the
number-or-null contract. No historical sprint JSON was changed.

Verification used temporary working directories and each tree's actual script by
absolute path. Validator fixtures were copies of this populated two-milestone sprint,
changing only M1's estimate. Invocation: `bash <tree>/skills/sprint-executor/scripts/validate_sprint_json.sh PROBE`.

| Probe (both trees) | Before | After |
|---|---|---|
| Validator: zero / null / absent / 245 | exits 1 / 0 / 0 / 0 | exits 0 / 1 / 1 / 0 |
| Creator: heading without LOC | numeric 0 | null |
| Creator: no milestone headings | numeric 0 | null |
| Creator: explicit ~0 / ~245 LOC | numeric 0 / 245 | numeric 0 / 245 |

Both rejected fixture variants named `M1_SENTINEL_MIGRATION`. Creator invocation:
`bash <tree>/skills/sprint-planner/scripts/create_sprint_json.sh <case> plan.md`.
Actual `session_start.sh DISPLAY` runs preserved `(estimated: 0 LOC)` and displayed
`(estimated: unset LOC)` with temporary fixtures; external ailang/make/bc commands
were stubbed to isolate display from network and unrelated tests.

Pairwise byte comparisons passed for all four file pairs; `bash -n` passed for all
six shell scripts; `git diff --check` passed. Completed controls
M-AGENT-AILANG-ONLY-EXECUTION, M-AI-DECIDE-SYSTEM-ONE and M-AILANG-FMT all returned
0 before and after in both trees. Each retained its single pre-existing warning
(missing pre-gate registry reuse audit).

The refreshed read-only audit found the same five non-completed zero milestones
recorded in the sprint plan. Four remain valid baseline/docs-only estimates;
M-MISSION-AGENTIC-ROUTING/M1b_CODEX_CROSS_PROVIDER_EXECUTOR remains unresolved and
requires its owning planner's confirmation before that sprint executes. No old
state record was rewritten and no estimate was fabricated.

The current checkout lacked the approved planner artifacts; they were retrieved
from coordinator/task-d071d41d (7a4faa07044f5ebdd94bb560c26a64fc3119e137).
Issue #563 and its empty comments list were read through the GitHub API before edits.
The PR body must include `Refs #563`; no new issue was opened.

Core regression: `make test-core` could not start (exit 127: make unavailable).
The same package list from `make/test.mk:404-411` was run directly with
`AILANG_TEST_FAST_LOOP=1 CGO_ENABLED=0 /usr/local/go/bin/go test <CORE_PKGS> -count=1`.
It returned 1: effects BrainStore SQLite tests require CGO and fail with the
SQLite stub error. All other listed core packages passed. This image also lacks
a C compiler, so CGO cannot be enabled here. No Go files changed. The startup
helper's green banner was not treated as evidence: its grep pipeline missed the
missing make command. Re-run `make test-core` in a compiler-equipped environment
before merge. Shell checks and the sentinel regression matrix passed.
