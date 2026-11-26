# Sprint Plan: M-AI-PREFERRED-V2 (v0.63.0)

## Placement and Human-Decision Rules

Stored in `design_docs/planned/v0_63_0/sprint-plan-m-ai-preferred-v2.md`.
All design decisions (**D1=b, D2=a, D3=a**) were ratified by Mark on 2026-11-26 in `design_docs/planned/v0_63_0/m-ai-preferred-v2.md`. **There are no remaining open human-decision gates** (except the final push/release approval owned by the Coordinator).

## Summary
Execute the 4 independent slices of `M-AI-PREFERRED-V2` in parallel isolated `share` workspaces to eliminate the #1 remaining `stretch` parse failure cluster (`PAR_MATCH_ARM_BLOCK` / `PAR_IF_BRANCH_BLOCK`), unify CLI + LSP quick-fixes with `TC_MATCH_002` typed-hole stubs, and add missing $O(n)$ `std/map` + `std/list` combinators.

**Duration:** 1–2 days (parallel across 3 author workers + 1 verifier)
**Dependencies:** `v0.62.0` (`f7a8ffb23`)
**Risk Level:** Low–Medium (Slice A touches `internal/parser`; Slices B & C are isolated and additive)

## Registry Reuse Audit
- Slice A (`internal/parser`): compiler internals — `none`
- Slice B (`internal/quickfix`, `internal/lsp`, `cmd/ailang`): Go tooling internals — `none`
- Slice C (`std/map`, `std/list`): core stdlib modules — `none`

## Proposed Milestones

### Milestone 1 (Slice A): Multi-Statement `match` Arms & `if` Branches
**Goal:** Allow unambiguous unbraced multi-statement `let` chains in `match` arms and `if`/`else` branches (normalized to `{ ... }` by `ailang fmt`), or fall back to deterministic `check --fix` auto-wrapping if any grammar ambiguity surfaces.
**Estimated:** ~200 LOC implementation + ~150 LOC tests
**Owned Paths:** `internal/parser/**`, `internal/format/**`, `examples/tests/syntax_ergonomics_v3_test.ail`
**Acceptance Criteria:**
- [ ] `go test ./internal/parser/... ./internal/format/... ./internal/testing/...` passes (including `TestPrinterRoundTrip`)
- [ ] `internal/parser/parser_expr.go` stays ≤800 lines
- [ ] New tests cover multi-statement `match` arms, nested `match`, and multi-statement `if`/`then`/`else` branches

### Milestone 2 (Slice B): Shared `internal/quickfix` + LSP `codeAction` + `TC_MATCH_002` Stub
**Goal:** Extract `computeCheckEdits` / `applyCheckEdits` from `cmd/ailang/check_fix.go` into `internal/quickfix/`, wire `textDocument/codeAction` in `internal/lsp/`, and add `TC_MATCH_002` missing-arm stub (`<pattern> => _?`).
**Estimated:** ~300 LOC implementation + ~200 LOC tests
**Owned Paths:** `internal/quickfix/**`, `internal/lsp/**`, `cmd/ailang/check_fix*.go`, `cmd/ailang/check_output.go`
**Acceptance Criteria:**
- [ ] `go test ./internal/quickfix/... ./internal/lsp/... ./cmd/ailang -run "TestComputeCheckEdits|TestRunCheckFix"` passes
- [ ] `ailang check --json` and LSP `codeAction` produce identical `TextEdit` ranges
- [ ] `make check-boundaries` passes

### Milestone 3 (Slice C): Stdlib Ergonomics (`std/map` + `std/list`)
**Goal:** Add `std/map`: `update`, `mapValues`, `filter` and `std/list`: `span`, `chunk`, `groupBy` in pure AILANG ($O(n)$, 0 VM fallbacks).
**Estimated:** ~120 LOC stdlib + ~120 LOC tests/examples
**Owned Paths:** `std/map.ail`, `std/list.ail`, `examples/tests/stdlib_combinators_test.ail`, `examples/runnable/list_combinators.ail`
**Acceptance Criteria:**
- [ ] `ailang test examples/tests/stdlib_combinators_test.ail` passes on both evaluator and `--strict-bytecode` (0 VM fallbacks)
- [ ] `examples/runnable/list_combinators.ail` passes on `--strict-bytecode`
- [ ] Worker does **NOT** run `make freeze-stdlib` or edit `prompts/` (handled by Coordinator post-merge)

### Milestone 4 (Slice D): Coordinator Merge, Shared Hotspots & Verifier Eval Gate
**Goal:** Coordinator merges Slices A → B → C, runs `make freeze-stdlib && make verify-stdlib`, updates `prompts/v0.16.20.md` (or bumps to `v0.16.21`) + `llms.txt` + `examples/manifest.json` + `changelogs/unreleased/`. Verifier runs full test suite + `gemini-2-5-flash` `stretch` A/B.
**Acceptance Criteria:**
- [ ] `make ci-quick` and `make verify-stdlib` green
- [ ] `stretch` eval shows reduction in `PAR_MATCH_ARM_BLOCK` / `PAR_IF_BRANCH_BLOCK` with zero `core` regressions

## Swarm Worker & Model Assignment

| Worker | Milestone(s) | Workspace | Assigned Model | Rationale |
|--------|--------------|-----------|----------------|-----------|
| `coordinator` | M4 (merge, `.stdlib-golden/`, `prompts/`, `llms.txt`, `manifest.json`) | main | Top reasoning (`pro` / inherit) | Owns merge order, conflict resolution, and final verification |
| `parser-ergonomics` | M1 (Slice A) | `share` | Top reasoning (`pro`) | Grammar/delimiter disambiguation in `internal/parser` requires high precision |
| `quickfix-lsp` | M2 (Slice B) | `share` | Standard / fast (`inherit` or `flash`) | Well-scoped Go extraction + LSP `codeAction` wiring |
| `stdlib-ergonomics` | M3 (Slice C) | `share` | Standard / fast (`inherit` or `flash`) | Pure AILANG stdlib functions + tests |
| `verifier` | M4 verification & evals | `inherit` | Standard / fast (`inherit` or `flash`) | Read-only test & eval verification |

- **Mid-run quota fallback**: Switch any `waiting` worker in-place via `swarm(action:"model", ...)` without losing conversation history.

## Success Metrics
- All unit, integration, round-trip, and `--strict-bytecode` tests passing
- Zero files >800 lines (`make check-file-sizes`)
- Zero architecture boundary violations (`make check-boundaries`)
- Measured reduction in `PAR_MATCH_ARM_BLOCK` / `PAR_IF_BRANCH_BLOCK` on `stretch` eval
