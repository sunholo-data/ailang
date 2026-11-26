# M-AI-PREFERRED-V2: Closing Measured Stretch Gaps & Self-Healing Tooling (v0.63.0)

**Status**: Approved (D1=b, D2=a, D3=a ratified by Mark 2026-11-26; ready for sprint execution)
**Target**: v0.63.0
**Priority**: P1 (High — directly targets the top remaining measured failure clusters from `v0.62.0` evals)
**Estimated**: 4 independent slices (A–D), designed for parallel `/swarm` execution with isolated `workspace:"share"` worktrees
**Dependencies**:
- `design_docs/planned/v0_62_0/m-ai-preferred-language.md` (shipped in `v0.62.0`: M1, M2, M3, M4, M5a)
- `M-AGENT-DIAGNOSTICS-ERGONOMICS` (`cf3140a33`)
- `M-AILANG-LSP-FOR-AI` (`design_docs/implemented/v0_20_0/m-ailang-lsp-for-ai.md`)
**Author**: Antigravity (attended session with Mark)
**Created**: 2026-11-26

---

## Axiom Compliance

| Axiom | Score | Justification |
|-------|-------|---------------|
| A1: Determinism | +1 | All parser/fix/stdlib transforms are 100% deterministic |
| A2: Replayability | 0 | No runtime nondeterminism |
| A3: Effect Legibility | +1 | LSP quick-fixes and typed-hole match stubs preserve explicit effect rows |
| A4: Explicit Authority | 0 | No capability or authority changes |
| A5: Bounded Verification | 0 | No changes to SMT verification bounds |
| A6: Safe Concurrency | 0 | No concurrency changes |
| A7: Machines First | +2 | Eliminates the #1 remaining parse failure cluster (7/16 stretch fails) and exposes machine-applicable edits over LSP |
| A8: Minimal Syntax | 0 | Zero new keywords or operators; D1 either accepts already-unambiguous block arms or auto-wraps via `--fix` |
| A9: Cost Visibility | +1 | New `std/map` & `std/list` combinators are single-pass $O(n)$ with 0 VM fallbacks |
| A10: Composability | +2 | `std/map` (`update`, `mapValues`, `filter`) and `std/list` (`groupBy`, `chunk`, `span`) remove hand-rolled recursion |
| A11: Structured Failure | +2 | `TC_MATCH_002` auto-stub (`=> _?`) + LSP `textDocument/codeAction` turn compiler diagnostics into 1-click/1-call repairs |
| A12: System Boundary | 0 | No host/FFI boundary changes |

**Net Score: +9** (≥ +2, no −1 on A1/A3/A4/A7).

---

## Problem Statement (Measured in `v0.62.0` Evals)

In our 30-benchmark `tier: stretch` A/B on `v0.62.0` (`gemini-2-5-flash`, `prompt v0.16.20`):
1. **7 of 16 failures (44%)** were `PAR_MATCH_ARM_BLOCK` (5) or `PAR_IF_BRANCH_BLOCK` (2) — models naturally write multi-statement `let` bindings inside `match` arms and `if`/`else` branches without wrapping `{ ... }`.
2. **`ailang check --fix` works in CLI, not yet in LSP**: `cmd/ailang/check_fix.go` computes deterministic `TextEdit`s, but `internal/lsp/` does not yet expose them via `textDocument/codeAction`, and `TC_MATCH_002` (missing match arm) does not yet emit a mechanical stub edit (`<missing_pattern> => _?`).
3. **Remaining stdlib boilerplate in caches/graphs/grouping**: `std/map` still lacks `update` / `mapValues` / `filter` and `std/list` lacks `groupBy` / `chunk` / `span`.

---

## High-Impact Decisions (Ratify Before `/swarm`)

| ID | Decision | Options | Recommendation | Owner | Change Cost |
|----|----------|---------|----------------|-------|-------------|
| **D1** | How to eliminate `PAR_MATCH_ARM_BLOCK` & `PAR_IF_BRANCH_BLOCK`? | **(a)** Auto-fix only: teach `ailang check --fix` + LSP to deterministically wrap unbraced multi-statement arms/branches in `{ ... }` (zero grammar change). **(b)** Grammar extension: allow unbraced `let ...; ...` inside `match` arms / `if` branches (and `ailang fmt` normalizes to `{ ... }`). | **(b)** if unambiguous with `,`/`}`/`else`, falling back to **(a)** if any ambiguity appears in parser tests | Mark | Medium (grammar) / Low (`--fix`) |
| **D2** | Extract `computeCheckEdits` into a shared `internal/quickfix/` package? | **(a)** Yes — move mechanical edit engine from `cmd/ailang/check_fix.go` to `internal/quickfix/` so both CLI (`check --fix` / `--json`) and `internal/lsp/` share one source of truth. **(b)** Duplicate minimal logic in `internal/lsp/`. | **(a)** Single source of truth, zero drift | Mark | Low |
| **D3** | Exact `v0.63.0` stdlib additions | **(a)** `std/map`: `update(m, k, default, f)`, `mapValues(f, m)`, `filter(p, m)` + `std/list`: `span(p, xs)`, `chunk(n, xs)`, `groupBy(keyFn, xs)`. **(b)** `std/map` only (`update`, `mapValues`, `filter`). | **(a)** All pure, $O(n)$, 0 VM fallbacks | Mark | Low |

> *Model assignments per `/swarm` worker will be set during `sprint-planner` before launch (keeping this design doc model-agnostic).*

### Design Freeze Checklist
- [x] **D1** ratified (`b` — grammar support for unbraced multi-statement `match` arms / `if` branches, falling back to `a` auto-fix if any ambiguity appears)
- [x] **D2** ratified (`a` — shared `internal/quickfix/` package used by CLI `check --fix` and `internal/lsp/` `codeAction`)
- [x] **D3** ratified (`a` — `std/map`: `update`, `mapValues`, `filter` + `std/list`: `span`, `chunk`, `groupBy`)

---

## Independent Workstreams (Slices A–D)

### Slice A — Multi-Statement `match` Arms & `if` Branches (Worker: `parser-ergonomics`)
- **Goal**: Eliminate `PAR_MATCH_ARM_BLOCK` and `PAR_IF_BRANCH_BLOCK` per **D1**.
- **Acceptance Criteria**:
  - `go test ./internal/parser/... ./internal/format/... ./internal/testing/...` passes (including `TestPrinterRoundTrip`).
  - `internal/parser/parser_expr.go` stays ≤800 lines.
  - New regression tests covering multi-statement `match` arms, nested `match`, and multi-statement `if`/`then`/`else` branches.

### Slice B — Shared `internal/quickfix` + LSP `codeAction` + `TC_MATCH_002` Stub (Worker: `quickfix-lsp`)
- **Goal**: Extract mechanical quick-fixes into `internal/quickfix/` (per **D2**), wire `textDocument/codeAction` in `internal/lsp/`, and add `TC_MATCH_002` missing-arm stub edit (`<pattern> => _?`) + redundant/unused import cleanup.
- **Acceptance Criteria**:
  - `go test ./internal/quickfix/... ./internal/lsp/... ./cmd/ailang -run "TestComputeCheckEdits|TestRunCheckFix"` passes.
  - `ailang check --json` and LSP `codeAction` return identical `TextEdit` ranges for `PAR017`, `PAR018`, `PAR_HASH_COMMENT`, `IMP010`, missing stdlib imports, and `TC_MATCH_002`.
  - `make check-boundaries` passes (no core → dashboard imports).

### Slice C — Stdlib Ergonomics (`std/map` + `std/list`) (Worker: `stdlib-ergonomics`)
- **Goal**: Implement ratified **D3** functions in pure AILANG with $O(n)$ complexity and 0 VM fallbacks.
- **Acceptance Criteria**:
  - `ailang test examples/tests/stdlib_combinators_test.ail` and `--strict-bytecode` pass with 0 VM fallbacks.
  - Runnable examples updated/added and pass under `--strict-bytecode`.
  - **Do NOT run `make freeze-stdlib` or edit `prompts/`** inside this worker's share (handled by Coordinator post-merge to avoid golden/hash conflicts).

### Slice D — Verifier & Eval Gate (Worker: `verifier`, read-only `workspace:"inherit"`)
- **Goal**: After Coordinator merges Slices A–C, run full verification (`make ci-quick`, `make verify-stdlib`, `ailang test --strict-bytecode`, and a `gemini-2-5-flash` `stretch` eval comparing `v0.62.0` vs `v0.63.0`).
- **Acceptance Criteria**:
  - Zero regressions on `core`, measurable reduction in `PAR_MATCH_ARM_BLOCK` / `PAR_IF_BRANCH_BLOCK` on `stretch`.

---

## Non-Goals (Hard Scope Fence for `/swarm`)
- **No `?` error propagation operator** (M5b remains deferred to `v1.1.0` per `m-error-propagation.md`).
- **No changes to `internal/types/typechecker_core.go`, `internal/eval/`, `internal/bytecode/`, or `internal/smt/`**.
- **No edits to unrelated dirty/untracked sibling-session files** (always stage explicit file lists).
- **No `git push`, tagging, or release cutting** without Mark's explicit go-ahead.

---

## Deferred Decisions (Agent Latitude)
- Exact internal helper function names and test file organization within each slice's owned directory.
- Exact `codeAction` title wording as long as it starts with a verb and names the fix.

---

## Swarm Execution Contract

> **Purpose**: Prevents merge conflicts, silent scope creep, and duplicate work when running `/swarm`.

### File Ownership Matrix
| Worker | Workspace | Owns (Write) | Read-Only | Do Not Touch |
|---|---|---|---|---|
| `parser-ergonomics` | `share` | `internal/parser/**`, `internal/format/**`, `examples/tests/syntax_ergonomics_v3_test.ail` | `internal/ast/**` | `cmd/ailang/**`, `internal/lsp/**`, `std/**`, `prompts/**`, `.stdlib-golden/**` |
| `quickfix-lsp` | `share` | `internal/quickfix/**`, `internal/lsp/**`, `cmd/ailang/check_fix*.go`, `cmd/ailang/check_output.go` | `internal/stdlibindex/**`, `internal/errors/**` | `internal/parser/**`, `std/**`, `prompts/**`, `.stdlib-golden/**` |
| `stdlib-ergonomics` | `share` | `std/map.ail`, `std/list.ail`, `examples/tests/stdlib_combinators_test.ail`, `examples/runnable/list_combinators.ail` | `std/option.ail`, `std/result.ail` | `.stdlib-golden/**`, `prompts/**`, `examples/manifest.json`, `cmd/**`, `internal/**` |
| `verifier` | `inherit` | *(none — read-only)* | Entire repo | Everything (reports only) |
| **Coordinator** | main | Merges A→B→C in order; runs `make freeze-stdlib`, updates `prompts/`, `llms.txt`, `examples/manifest.json`, `changelogs/unreleased/`; presents final diff for Mark's approval | Everything | Never pushes or releases without explicit user OK |

### Hard Stop-Gates for Every Worker
1. Work ONLY inside your owned paths; if you discover a needed edit outside your slice, **stop and post on the board** instead of editing it.
2. Run your slice's unit tests and post a concise summary (`files changed`, `tests run`, `pass/fail`, `open questions`) to the board, then **go idle (`waiting`)**.
3. Never run `git push`, `git reset --hard`, or `make freeze-stdlib`.
