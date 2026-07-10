# M-NAMED-TEST-BLOCKS: Execute `test "name" { ... }` blocks — and stop reporting skipped suites as passing

**Status**: ✅ Implemented (unreleased, target v0.29.0) — M1 shipped 2026-07-09 (ec4996e45, 7389e84c1, + fixes fd75ce8d4/71d0d43a3); closeout-verified 2026-07-10 (v1-mission iteration 1): failing named test → FAIL + exit 1 ✓; `--allow-skips` + "NO TESTS RAN" honesty wired ✓; duckdb's previously-skipped shipped tests now execute 2/2 ✓; CHANGELOG ✓. NOT verifiable locally: deontic `engine_test.ail` 5/5 (package absent from local checkouts) and its AGENT.md workaround retirement — follow up when deontic is next touched.
**Target**: v0.29.0
**Priority**: P0 (silent-green test runner: a deliberately failing test currently reports "All tests passed!")
**Estimated**: 2–3 days (runner execution 1–1.5d, reporting honesty 0.5d, package-mode fixtures 0.5d)
**Dependencies**: reuses the Core-evaluation machinery from [m-testing-inline-core-evaluation (implemented v0.4.7)](../../implemented/v0_4_7/m-testing-inline-core-evaluation.md)

## Axiom Compliance

| Axiom | Score | Justification |
|-------|-------|---------------|
| A1: Determinism | +1 | Test bodies are pure expressions evaluated deterministically |
| A2: Replayability | 0 | No change |
| A3: Effect Legibility | +1 | Named test bodies are checked pure — pins the discipline |
| A4: Explicit Authority | 0 | No change |
| A5: Bounded Verification | +2 | An entire verification surface (named tests) goes from decorative to real |
| A6: Safe Concurrency | 0 | No change |
| A7: Machines First | +2 | Agents currently receive a FALSE "All tests passed!" signal — the worst possible machine-facing output |
| A8: Minimal Syntax | +1 | Syntax already parses; zero new surface |
| A9: Cost Visibility | 0 | No change |
| A10: Composability | +1 | Same runner path as inline tests; package mode included |
| A11: Structured Failure | +2 | Skip reasons exist internally but are discarded by the human reporter — surfaced |
| A12: System Boundary | 0 | No change |

**Net Score: +10** → **Decision: Move forward**

### Hard Violation Check
- [x] A1 / A3 / A4 / A7: no violations (A7 is the point of the fix)

## Problem Statement

`test "name" { <expr> }` blocks parse, type-check, and are silently **never executed**:

```go
// internal/testing/runner.go:~143
// Non-inline tests (test "name" { ... } blocks)
// For now, skip these - they're less common
result.Status = StatusSkip
result.Error = "Named test blocks not yet implemented"
```

Worse, the reporter treats an all-skipped suite as success. Verified live (v0.28.0):

```
$ ailang test standalone_test.ail    # contains: test "deliberately failing" { double(2) == 5 }
✓ All tests passed!
2 tests: 0 passed, 0 failed, 2 skipped
```

The skip Error string ("Named test blocks not yet implemented") is stored but never
displayed in human format — the ONE line that would have explained everything.

**Blast radius (verified):**
- `packages/duckdb/query_test.ail` (ailang-packages) — shipped test file, all tests skipped since creation.
- `packages/deontic/engine_test.ail` (M-DEONTIC-PKG) — 5 ground-truth assertions skipped; the package fell back to a runnable-demo diff as its gate.
- Any agent or CI step gating on `ailang test` exit code or "All tests passed!" has been green unconditionally.

**Systemic check:** the runner has SEVEN `StatusSkip` sites (properties without
generators, top-level ensures, missing lowered predicates, …) — all share the same
reporting hole: reasons invisible, skipped counted as green. The reporting fix must
cover all of them, not just named blocks.

## Goals

**Primary Goal:** `test "name" { expr }` blocks execute; a suite where nothing ran can never report success.

**Success Metrics:**
- The deliberately-failing standalone file exits non-zero with a FAIL line.
- duckdb + deontic package test files run for real (`--package` mode included).
- Every StatusSkip prints its reason in human output.
- 0-run / >0-skipped suites print "NO TESTS RAN" and exit non-zero unless `--allow-skips`.

## High-Impact Decisions

| Decision | Why High Impact | Chosen By | Deadline | Change Cost |
|----------|-----------------|-----------|----------|-------------|
| Non-zero exit for all-skipped suites (+ `--allow-skips` escape) | Flips CI semantics; today's false greens surface | human | design | low |
| Pass contract: body must evaluate to bool `true`; runtime error = FAIL with message | Defines test semantics forever | human | design | low |

### Design Freeze
- [ ] Mark confirms non-zero exit for 0-run suites (with `--allow-skips` escape hatch)
- [ ] Mark confirms bool-true pass contract (matches existing *_test.ail bodies in the wild — verified: duckdb + deontic bodies are bool expressions)

## Conflict Surface

Touches `internal/testing/` (runner + reporter) and `cmd/ailang/test.go`. NOT the
parser (syntax already parses) or the elaborator (bodies already type-check).

1. **Positions extended:** none syntactically — execution semantics added to an already-parsed decl form.
2. **Existing constructs in the position:** inline tests (M-TESTING-INLINE) and property/ensures tests share the runner; their pass/fail/skip flows must stay unchanged except reporting. Fixture: one file mixing all kinds, per-kind statuses asserted stable.
3. **Disambiguation:** named blocks are already a distinct runner case arm — the skip arm becomes the execute arm.
4. **Programs that must still work:** `std/trace_test.ail`; duckdb/deontic test files (now actually running — bodies verified to be bool exprs); the inline_tests eval benchmark.
5. **Deliberate change:** all-skipped ⇒ exit 1. Breaking for anything relying on the current false green — that reliance is the bug.

## Solution Design

1. **Execution** (~150 LOC): named block body = pure expression elaborated in module
   scope — exactly the v0.4.7 inline-test core-evaluation path (reuse, don't fork).
   `true` = PASS; `false` = FAIL; runtime error = FAIL carrying the error text.
2. **Reporting honesty** (~60 LOC): human formatter prints `⊘ name — <reason>`;
   summary requires `run>0 && failed==0` for "All tests passed!"; `run==0 && skipped>0`
   ⇒ `⚠ NO TESTS RAN (N skipped)` + exit 1 unless `--allow-skips`.
3. **Fixtures** (~80 LOC): named pass/fail/runtime-error; mixed-kind file; package-relative
   imports under `--package .` (the deontic/duckdb shape); all-skipped exit-code test.

### Files to Modify
| File | Change |
|---|---|
| `internal/testing/runner.go` | execute named blocks |
| `internal/test/reporter.go` | skip reasons, honest summary, exit semantics |
| `cmd/ailang/test.go` / `main.go` | `--allow-skips` flag |
| `internal/testing/*_test.go` | fixtures above |

## Success Criteria
- [x] Deliberately-failing named test ⇒ non-zero exit + FAIL line with location (verified live 2026-07-10)
- [ ] deontic `engine_test.ail` 5/5 passing via `ailang test --package .` — NOT VERIFIABLE locally (package absent); duckdb equivalent verified instead: previously-skipped shipped tests now 2/2 via `--package .`
- [x] All-skipped ⇒ "NO TESTS RAN" + exit 1; `--allow-skips` ⇒ exit 0 (reporter.go:205, result.go AllSkipped/SuccessAllowingSkips, cmd/ailang/test.go:80)
- [x] Skip reasons visible for remaining StatusSkip sites (6 remain; one replaced by execution)
- [x] CHANGELOG updated (M-NAMED-TEST-BLOCKS entry under [Unreleased]); deontic AGENT.md retirement deferred with the deontic criterion above

## Verification Log

| Claim | Method | Result |
|---|---|---|
| Named blocks parse + typecheck | `ailang check` on deontic engine_test.ail | ✓ No errors found |
| Runner skips them by design | internal/testing/runner.go ~L143 read | quoted above |
| Failing test reports success | live run, standalone file, v0.28.0 | transcript above |
| duckdb shipped tests skip | `ailang test query_test.ail` in ailang-packages | 0 passed, 2 skipped |
| Bodies in the wild are bool exprs | read duckdb + deontic test files | confirmed |

## Non-Goals
- New test syntax (setup/teardown, fixtures, mocks) — bodies stay pure bool expressions.
- Property-test generator coverage (its skips become VISIBLE here, not fixed — separate doc if prioritized).

## Related Documents
- [m-testing-inline-core-evaluation (v0.4.7)](../../implemented/v0_4_7/m-testing-inline-core-evaluation.md) — machinery to reuse (neural 0.47; distinct: that shipped inline tests, this ships named blocks + reporting honesty)
- [m-eval-rig-reliability](m-eval-rig-reliability.md) — the same "false green" failure class at rig level
- ailang-packages `packages/deontic/AGENT.md` — documents the workaround this retires

---
**Document created**: 2026-07-09
