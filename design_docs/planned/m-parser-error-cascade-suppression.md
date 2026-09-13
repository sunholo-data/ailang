# Parser Error-Cascade Suppression Policy

**Status**: Planned
**Target**: v0.39.0
**Priority**: P1 (eval-wide DX cost — every model pays it, not one model)
**Estimated**: 3 days
**Dependencies**: None (builds on the coded-placement-diagnostic pattern: `PAR_MODULE_PLACEMENT`, import placement)
**Source**: GitHub issue #934 — frontier eval runs 2026-08-27 measured cascades of **342, 153, and 29 errors from a single bad token**
**Classification**: This is an **eval-wide DX policy change** for agent self-repair, not a language-semantics change.

## Problem Statement

One malformed token currently emits a cascade of hundreds of downstream `PAR_UNEXPECTED_TOKEN` / `PAR_NO_PREFIX_PARSE` errors. A model reading stderr to repair its own code must find the single real mistake buried in noise — and fails to, repeatedly, at eval scale. Issue #934 measured cascades of 342, 153, and 29 errors from one token on frontier runs (2026-08-27).

Targeted coded diagnostics already *replace* the cascade for two shapes — `PAR_MODULE_PLACEMENT` (module placement, `internal/parser/parser_file.go`, `module_placement_test.go`) and import placement (`parser_decl.go:383-392`, "consume the whole import so we don't cascade"). The general case is open: `parseParams` (`parser_expr.go:750`) has no recovery point — a bad token inside a parameter list continues normal parsing mid-corruption and reports at every subsequent sync failure.

**Impact:** every eval model that reads stderr for self-repair pays a token-budget and accuracy cost. The eval harness already surfaces `error_category` (`compile_error`) and `err_code` (`PAR001`, …) per benchmark row (`internal/eval_harness/metrics.go:101`), so cascade noise also corrupts the per-code failure statistics benchmarks are ranked on.

**Current state:**
- `ParserError` (`internal/parser/parser_error.go`) is already structured: `Code`, `Pos`, `NearToken`, `Expected`, `Suggestions`, `Confidence` — the plumbing for machine-readable output exists.
- `ParseFile` wraps panics into structured errors (`parser_file.go:14-16`) but has no cascade budget or panic-mode token synchronization.
- Tests assert recovery *finds* many errors (`error_recovery_test.go` TestMultipleErrors) — these must keep passing for genuinely independent errors.

## Goals

1. **One real error per real mistake.** A single malformed token must never produce more than a small, fixed number of diagnostics.
2. **First-real-error prominence.** The one diagnostic an agent needs appears first and is clearly marked as primary.
3. **Honest diagnostics (hard constraint).** Suppression must never (a) invent a false fix suggestion, nor (b) hide a second *genuinely independent* error (CLAUDE.md §2, no silent fallbacks).
4. **Machine-readable suppression metadata.** Agents and the eval harness can distinguish `primary` from `suppressed_cascade` errors without parsing prose.
5. **Preserve benchmark signal.** `err_code` / `error_category` stats fed to eval metrics must remain meaningful post-suppression.

## Non-Goals

- Recovering more valid ASTs (parser permissiveness changes) — only error *reporting* policy.
- Changing `ParserError` codes' existing meanings (`PAR001..PAR013`, `PAR_*` named diagnostics).
- Touching type-checker (`TY_*`) or module-pipeline (`MOD_*`, `LDR_*`) cascades; those follow the same pattern later if measurement demands.

## Proposal

### Ruling: cap + first-real-error prominence, with targeted recover-and-continue at declaration boundaries

Two candidate policies were considered:

- **A. Cap + prominence (global):** keep parsing as today, but tag the first `PAR_*` error by position as `primary`, demote all subsequent errors within the cascade window as `suppressed_cascade`, and cap total emitted diagnostics at N.
- **B. Recover-and-continue (heuristic panic-mode):** on first `PAR_*`, enter panic mode, synchronize to a declared token (top-level `let`/`func`/`import`/`module`/`test`/newline-anchored declaration start), then resume normal parsing — the classic dragon-book strategy.

**Ruling: A is the floor; B is applied only at the four declaration-boundary recovery points we can prove safe.**

Rationale:
- Pure B (recover-and-continue everywhere) changes which errors are *findable* and risks hiding independent errors — a violation of the honest-diagnostics constraint whenever the sync token skips past real damage (this is exactly how gemini's late-module false-MOD002 bug arose; the state-isolation rule in `parser_file.go:424` was the fix for its recovery path).
- Pure A leaves the parser continuing in a corrupted state, emitting garbage-position diagnostics after the cap.
- The composition is honest by construction: the cap only ever demotes errors *after* the first one, and panic-mode sync only skips tokens *within* the current declaration (never across a declaration boundary, so a second independent error at a later declaration is still reported as its own primary).

### Mechanism

1. **Primary selection.** The first `PAR_*` error by source position in a parse is tagged `primary: true`. Exactly one primary per file parse. If no `PAR_*` error exists, no suppression happens at all.
2. **Cascade window.** All `PAR_*` errors located *after* the primary whose position is within the same or immediately-following declaration (i.e., not separated by a successfully-parsed declaration boundary) are `suppressed_cascade`. They are counted, not emitted (human stderr shows `(+N suppressed cascade errors)`; JSON carries `suppressed_count` + the suppressed codes' *code multiset*, e.g. `{"PAR_UNEXPECTED_TOKEN": 41}` — codes survive so eval stats stay honest about what went wrong, positions/messages are dropped as noise).
3. **Hard cap.** Emitted diagnostics per file parse: max **3** (primary + up to 2 genuinely independent errors). Independent errors are only those at a position separated from all prior errors by a successfully-parsed declaration boundary. If a 4th+ independent error exists, emit `PAR_ERROR_BUDGET` as a final diagnostic stating how many were truncated — never a silent drop.
4. **Recovery points (targeted panic-mode).** Extend the existing consume-the-declaration pattern to: `parseParams` (sync to `RPAREN`, then consume to end-of-signature; a bad param token today cascades into the body), `parseImport` (already done, `parser_decl.go:383`), module declaration (already done, `PAR_MODULE_PLACEMENT`), and record/match arm lists. Each recovery point must follow the documented state-isolation rule: the recovery path must not mutate parser state a subsequent declaration's diagnostics depend on.
5. **Prominence in stderr.** Human format prints the primary first with `PRIMARY:` prefix; suppressed-cascade line last, one line. No reordering of the remaining independent errors.

### What counts as the ONE real error

**First `PAR_*` by position**, with this caveat: if the first `PAR_*` is a generic `PAR_UNEXPECTED_TOKEN`/`PAR_NO_PREFIX_PARSE` *and* a coded diagnostic (`PAR_MODULE_PLACEMENT`, `PAR_HYPHEN_IN_MODULE`, …) exists at a position ≤ 5 tokens later, the coded diagnostic is primary — this matches the measured failure mode where the generic cascade fires before the placement check gets its turn. Context surviving around the primary: the source line, the near token, expected-token set, and any `Suggestions` the primary already carries — all unchanged; suppression never edits the primary's content or adds suggestions it did not already have.

### Honest-diagnostics constraint (verification obligations)

- Suppression **never** fabricates or rewrites a `Suggestions`/`Fix` field; it only reorders/omits whole errors.
- A second genuinely independent error (separated by a clean declaration boundary) **must** still be emitted — regression tests must cover the two-independent-errors case (extend `error_recovery_test.go` TestMultipleErrors as a guard: those three errors are in separate declarations and must all survive).
- Any truncation is announced (`PAR_ERROR_BUDGET`), never silent (CLAUDE.md §2).
- Existing guard tests keep passing: `module_placement_test.go` (cascade replaced by one `PAR_MODULE_PLACEMENT`; second late module still reported; `PAR_HYPHEN_IN_MODULE` truncation inside a recovery), `import_placement_test.go`.

### Machine-readable shape

`ailang check --json` (and eval-harness capture) extends each parser diagnostic with:

```json
{
  "error_category": "compile_error",
  "err_code": "PAR001",
  "suppression": {
    "primary": true,
    "suppressed_count": 341,
    "suppressed_codes": {"PAR_UNEXPECTED_TOKEN": 41, "PAR_NO_PREFIX_PARSE": 300}
  }
}
```

- `suppression` is omitted entirely when no cascade was detected (no behavior change for clean parses).
- `error_category` and `err_code` field names match the eval-harness row schema (`internal/eval_harness/metrics.go:101`) so PAR_* categories keep feeding benchmarks without a mapping change.
- Agents consuming stderr (non-JSON) see the primary first and the one-line suppressed count — sufficient for self-repair without JSON.

### Eval-metric impact

- `err_code` per-row stats become *more* accurate: today a cascade logs 342 rows' worth of PAR_UNEXPECTED_TOKEN impressions, inflating that code's share and diluting the real primary codes. Post-change, primary codes dominate; suppressed codes remain visible via the multiset for honesty.
- Benchmark regression risk: none expected — evaluation of tasks is by test/verify outcomes, not stderr; but the PAR_* category dashboards must be re-baselined after landing (note in the eval baseline update, standard post-release step).
- Expected agent-repair win: fewer wasted repair turns targeting cascade symptoms; measure via `ailang eval-paired` on/off across one smoke tier before widening.

## Alternatives Considered

- **Recover-and-continue everywhere (pure B):** rejected — risks masking independent errors, and sync-token choice is unprovable in the general expression grammar.
- **Pure cap, no recovery points:** rejected as the sole policy — post-corruption parsing still produces garbage-position secondaries that eat the cap slots meant for real independent errors.
- **Prompt-side mitigation ("ignore cascades"):** rejected — measured advisory-skip problem; stderr policy must be enforced in the tool, not the prompt.

## Acceptance Criteria

- [ ] Issue #934's three repro cases (342 / 153 / 29-error cascades) each emit ≤ 3 diagnostics with exactly one `primary`.
- [ ] Two genuinely independent errors in one file both emitted (regression test added to `error_recovery_test.go`).
- [ ] `PAR_ERROR_BUDGET` emitted whenever truncation occurs; zero silent drops (fuzz corpus re-run asserts emitted+suppressed == total).
- [ ] Existing placement-diagnostic tests pass unchanged (`module_placement_test.go`, `import_placement_test.go`).
- [ ] `check --json` carries the `suppression` block; eval harness rows unchanged for clean parses.
- [ ] `ailang eval-paired` smoke-tier run on/off reported in the implementation report; PAR_* category baselines noted for re-baselining.

## Testing Strategy

- Unit: primary-selection rule (incl. coded-diagnostic-over-generic override), cascade-window boundary (declaration separation), cap + budget error, each new recovery point with a state-isolation mirror of the gemini late-module test.
- Fuzz: invariant `emitted + suppressed_count == total PAR_* errors found`, never fewer (no hiding), and primary is always the minimum-position PAR_*.
- Eval: paired on/off smoke tier before merge into the eval rotation.

## Open Questions

- Should `suppressed_codes` also carry one representative position per code (for debugging) or positions only in `--verbose`? Default: positions in verbose only, to keep agent-facing JSON small.
- Exact sync set for `parseParams` recovery (RPAREN-only vs RPAREN-or-declaration-start) — decide in sprint from the fuzz results.