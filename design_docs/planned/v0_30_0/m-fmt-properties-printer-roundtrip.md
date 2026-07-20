# m-fmt-properties-printer-roundtrip — Contract-Clause Printer Round-Trip Fix

**Status**: **UNPARKED — Mark 2026-07-20 ("yes lets finish off ailang fmt")**: the one persistent
R2 objection is DATA-REFUTED (controller data-check, iter 66 — recorded in the Quorum Record); a
text reviewer rejecting against measured data does not outrank the data. Route to sprint-planner;
no re-quorum. fmt is a weak-model accessibility lever — finish the polish set.
**Target**: v0.30.0 (fmt Phase-1 correctness follow-up)
**Priority**: P1
**Estimated**: ~1 day
**Dependencies**: m-ailang-fmt (Phase 1, implemented), m-ailang-fmt-phase2 (comment preservation, implemented — this fix rides under its corpus gate)

## ⛔ Quorum Record (mission iteration 66, 2026-07-20) — PARKED needs-human-review

This NEW doc (authored by a crashed iter-66 designer pass, recovered from the working tree) ran the
QUORUM-AT-PICK text quorum (`gpt5-6-sol` + `gemini-3-1-pro`, controller verdict PASS both rounds).
The bounded gate (one revision + one re-quorum) is now **consumed** and the doc is **PARKED for
human review**. Total metered quorum spend: **$0.1347** ($0.0609 R1 + $0.0738 R2).

**Round 1 (original doc) — BLOCKED.** `gpt5-6-sol` reject: the parser append change expands the AST
seen by every `FuncDecl.Properties` consumer, but the doc asserted consumers already filter by
`Kind` without verifying it; no verification-log row enumerated those consumers. `gemini-3-1-pro`
pass; controller pass (live-reproduced the exit-2 bug + root cause at HEAD). → **Resolved by Rev-2**:
added the V17 consumer audit (6 sites, per-site Kind-handling table) + an acceptance-gated
`TestCombinedContractsAndPropertiesPipeline` integration test locking the combined case.

**Round 2 (Rev-2) — BLOCKED on ONE persistent objection.** `gpt5-6-sol` reject: V17 grepped only
`internal/` with textual `.Properties`, so it "could miss consumers under `cmd/`, tools or other
directories, as well as aliased, embedded, accessor-based, interface-driven, or visitor-based uses."
`gemini-3-1-pro` pass (no hard objections). controller pass (V17 audit is live-verified and complete).

**Controller post-quorum verification (in-session, no metered cost) — the R2 objection does NOT
materialize.** A repo-wide sweep `grep -rn "\.Properties" cmd/ tools/ internal/` (excl. `_test.go`,
`internal/format/`, `internal/parser/`) finds the ONLY genuine `ast.FuncDecl.Properties` (AST field)
consumers are exactly the V17 sites: `internal/elaborate/file.go:277,415` and `internal/testing/`
(collector + runner). The two `cmd/ailang/test.go:53,182` hits are a **different** `Properties`
field — `[]PropertyResult` test-result aggregation (confirmed via `reporter.go`'s
`formatPropertiesJSON([]PropertyResult)`), NOT the AST field. No accessor/interface/visitor
indirection exists on the AST field. The blast-radius analysis is therefore complete.

**Decision for Mark (#399):** the persistent gpt5-6-sol objection is a methodological concern
(grep scope) that the controller's cheap repo-wide re-check **data-refutes** — there are no
`FuncDecl.Properties` consumers outside the V17-enumerated sites. The doc is well-scoped (~1d, LOW
risk/conflict, printer-only + one-line parser append fix), fixes a live exit-2 defect on the entire
contract corpus AND a latent silent-contract-deletion data-loss bug, and is locked by an
acceptance-gated integration test. **Recommended: (1) authorize routing to sprint-planner** (the
one residual objection is now refuted; unlike fmt-phase2 this is not a deepening-gap pattern).
Alternatives: (2) authorize one more bounded revision round to fold the repo-wide audit into the
doc's Verification Log before planning; (3) keep parked.

## Scope Correction vs the Queue Tag

The mission-queue tag framed this as "`properties [...]` blocks fail the printer round-trip."
Live verification at HEAD (`v0.30.0-24-g5afa9a1e1`) shows that framing is wrong in both directions:

1. **The failing construct is `requires { … }` / `ensures { … }` contract clauses**, not
   `properties [...]` blocks. Genuine `forall(...)` properties blocks round-trip correctly today
   (live-verified below).
2. The failure is **comment-independent**: a minimal comment-free file with only
   `requires`/`ensures` fails `fmt --check` with exit 2 ("formatter defect"). This is a Phase-1
   printer correctness bug, NOT a Phase-2 comment-preservation gap.
3. Live corpus count at HEAD: **30 files** fail with the exit-2 defect via the CLI
   (29 under `examples/runnable/contracts/`, plus `examples/ai_devtools_workflow/discount_calculator.ail`).
   The Phase-2 gate (`TestCorpusCommentGate`) reports **28** as `preexisting-Phase1-rt-bug` — the
   other 2 (`inbox_injection_v2.ail`, `inbox_v2_app.ail`) are refused *earlier* by the enumerated
   comment-attachment carve-out (`comment at byte N … could not be attached to any boundary`) and
   never reach the gate's round-trip check. Those 2 stay fail-closed after this fix (separate,
   documented Phase-2 carve-out — out of scope here).

This is a printer round-trip **fix** (~1d), not a scope expansion.

## Axiom Compliance

**Canonical reference:** [Design Axioms](/docs/references/axioms)

### Axiom Scoring

| Axiom | Score | Justification |
|-------|-------|---------------|
| A1: Determinism | 0 | Formatter tooling only; no language semantics touched |
| A2: Replayability | +1 | `fmt` becomes byte-stable and lossless over the entire contract corpus |
| A3: Effect Legibility | 0 | No effect changes; the `! {}` row emission is preserved as-is |
| A4: Explicit Authority | 0 | No authority changes |
| A5: Bounded Verification | +1 | Contract files (the Z3-verified corpus) become formatter-eligible; fixes silent contract deletion (parser clobber) that could erase verified `requires` clauses |
| A6: Safe Concurrency | 0 | N/A |
| A7: Machines First | +1 | `fmt` is an AI-loop tool; 30 fail-closed refusals on the contract corpus is direct machine-workflow friction |
| A8: Minimal Syntax | 0 | Zero new syntax; printer emits existing surface forms |
| A9: Cost Visibility | 0 | N/A |
| A10: Composability | 0 | N/A |
| A11: Structured Failure | 0 | Fail-closed behavior preserved for remaining carve-outs |
| A12: System Boundary | 0 | N/A |

**Net Score: +3** → **Decision: Move forward**

### Hard Violation Check

- [x] A1 (Determinism): No implicit nondeterminism introduced
- [x] A3 (Effects): No hidden side effects
- [x] A4 (Authority): No ambient access granted
- [x] A7 (Machines First): Directly serves machine workflows

## Problem Statement

`ailang fmt` (Phase 1 printer) cannot format any file that uses `requires`/`ensures` contract
clauses. The formatter fail-closes correctly (never writes lossy output), but the entire contract
corpus — the flagship Z3-verification examples — is formatter-ineligible.

**Current behavior (live-verified at HEAD `5afa9a1e1`, 2026-07-20):**

Minimal comment-free repro:

```ailang
module test/cf_contract

export func absolute(x: int) -> int ! {}
requires { x >= 0 }
ensures { result >= 0 }
{
  x
}
```

- `ailang check` → exit 0, `✓ No errors found!`
- `ailang fmt --check` → **exit 2**, `formatted output failed to re-parse (formatter defect)`

**Root cause (verified by reading the code at HEAD):**

The parser stores contract clauses and forall properties in the SAME slice,
`FuncDecl.Properties []*ast.Property`, discriminated by `Property.Kind ContractKind`
(`internal/ast/ast_decl.go:91-118`: `PropertyKind` / `RequiresKind` / `EnsuresKind`).
`parseContractBlocks` appends `RequiresKind` then `EnsuresKind` entries
(`internal/parser/parser_func.go:121-126`), with `Binders: nil`
(`internal/parser/parser_contracts.go:141-147`).

The printer ignores `Kind` entirely: `testsAndProperties` (`internal/format/decl.go:184-206`)
routes ALL `d.Properties` through `propertiesBlock` (`decl.go:259`, emits `properties [`), and
`property()` (`decl.go:285-304`) emits a binder-less entry as a **bare expression**. So the repro
above is printed as:

```
export func absolute(x: int) -> int ! {}
  properties [
    x >= 0,
    result >= 0
  ]
{ ... }
```

That output is guaranteed not to re-parse: `parseProperty` **requires** the `forall` keyword as
the first token of every properties-block entry (`internal/parser/parser_testing.go:242-245`,
`PAR_UNEXPECTED_TOKEN "expected forall in property"`). The round-trip re-parse in
`cmd/ailang/fmt.go:139-142` then fails → exit 2, fail-closed.

**Second (latent) bug found during verification — silent contract deletion:**

`internal/parser/parser_func.go:169` uses **assignment**, not append:
`fn.Properties = p.parsePropertiesBlock()`. Contracts are appended at lines 124-125 *before*
tests/properties are parsed, so on a function that has BOTH contracts and a `properties [...]`
block, the assignment clobbers the contract entries. Live-proven at HEAD:

```ailang
module test/both

export func f(x: int) -> int ! {}
requires { x >= 0 }
  properties [
    forall(y: int) => f(y) >= 0
  ]
{
  x
}
```

`ailang check` passes; `ailang fmt` **exits 0 and silently deletes the `requires` clause** from
its output. The round-trip verifier cannot catch this because the loss happens at parse time,
before printing — `Parse(fmt(x)) == Parse(x)` holds on the already-lossy AST. This is the one
case where fmt is currently **lossy despite exiting 0**, and it must be fixed in the same sprint
(one-line parser change), or `fmt --write` could destroy verified contracts. No corpus file
currently combines both forms (verified by grep), so this is latent, not yet user-visible.

**Impact:**
- 30 example files refuse to format (exit 2): all of `examples/runnable/contracts/` that reach
  the printer, plus `examples/ai_devtools_workflow/discount_calculator.ail`.
- The Phase-2 corpus gate carries a standing `preexisting-Phase1-rt-bug=28` exception
  (`internal/format/corpus_comment_test.go:88-125`) that this fix should drive to 0 and then
  harden.
- Any downstream fmt adoption (m-ailang-fmt-adoption) is blocked on the contract corpus.

## Goals

**Primary Goal:** Every parse-valid file using `requires`/`ensures` and/or `properties [...]`
round-trips through the Phase-1 printer with a structurally identical AST.

**Success Metrics:**
- `ailang fmt --check` produces **zero exit-2 defects** on `examples/runnable/contracts/*.ail`
  and `examples/ai_devtools_workflow/*.ail` (the 2 comment-attachment carve-out files excepted —
  they are refused for an unrelated, enumerated Phase-2 reason).
- `TestCorpusCommentGate`: `preexisting-Phase1-rt-bug` drops **28 → 0**, and the gate is hardened
  so any future non-zero count FAILS the test.
- The contracts+properties combination is lossless: `requires` clauses survive `fmt`.

## High-Impact Decisions

| Decision | Why High Impact | Chosen By | Deadline | Change Cost |
|----------|-----------------|-----------|----------|-------------|
| Split `FuncDecl.Properties` by `Kind` at print time (emit contracts as signature-position clauses, properties block for `PropertyKind` only) | This IS the fix; alternative (a dedicated AST field split) would ripple through parser/elaborator/verifier | agent (design here) | design | low |
| Fix parser clobber `parser_func.go:169` (`=` → `append`) in the same sprint | Without it, `fmt --write` silently deletes contracts on combined files — data loss with exit 0 | agent (design here) | design | low |
| Harden `TestCorpusCommentGate`: `preExistingRT` tolerated → must-be-zero | Locks the fix in CI; prevents the exception class from silently regrowing | agent | compile | low |
| Reformat the 28 now-eligible corpus files with `fmt --write` as part of M2 acceptance | Required for the "`fmt --check` exits 0" acceptance (exit 1 = needs-reformat is otherwise permanent); touches 28 checked-in examples | agent, mechanical; `ailang check` must re-pass on all | compile | low |

### Design Freeze

- [x] Emission strategy: print-time split by `ContractKind` (no AST shape change) — decided in this doc
- [x] Parser clobber fix is in scope — decided in this doc
- [x] The 2 comment-attachment carve-out files (`inbox_injection_v2.ail`, `inbox_v2_app.ail`) remain fail-closed and are NOT acceptance criteria — decided in this doc

## Solution Design

### Overview

Teach the printer that `FuncDecl.Properties` is a mixed-kind slice. Emit:

- `RequiresKind` entries as ONE `requires { p1, p2, … }` clause,
- `EnsuresKind` entries as ONE `ensures { p1, p2, … }` clause,

both in **signature position** — after the effect row, BEFORE `tests [...]`/`properties [...]`
and the body (this is the only order the parser accepts: `parseContractBlocks` runs before
tests/properties parsing, `parser_func.go:114-175`) — and route **only** `PropertyKind` entries
through the existing `properties [...]` emission (omitting the block when none remain).

Grammar constraints that make this canonicalization safe (all verified by reading
`parser_contracts.go`):
- The parser only accepts requires-before-ensures order, so the slice is always
  `requires*, ensures*, properties*` — printing grouped blocks in that order re-parses to the
  identical slice order.
- Duplicate `requires`/`ensures` blocks are a parse-time diagnostic
  (`PAR_DUPLICATE_REQUIRES`/`_ENSURES`), so emitting one merged block per kind is canonical.
- Empty contract blocks (`requires {}`) parse to zero entries → nothing to print → AST-identical.
- Contract predicates may contain `ForallExpr` (`forall i: lo..hi => body`); the expression
  printer already handles it (`internal/format/expr.go:75`, `forall()` at `expr.go:428`, covered
  by `node_coverage_test.go`).

The round-trip verifier (`cmd/ailang/fmt.go:143`) compares full ASTs via `cmp.Diff` ignoring only
pos/span — so `Kind`, order, and `Binders` identity are all enforced automatically once the
emission is right.

### Implementation Plan

**M1: Printer emission split + parser clobber fix** (~4h) — ✅ IMPLEMENTED (commit 58eadd3b5)
- [x] `internal/format/decl.go`: in `funcDecl`, after the effect row, partition `d.Properties`
      by `Kind`; emit `requires { … }` / `ensures { … }` clauses (comma-separated predicates,
      slice order). **Refactor `testsAndProperties` to accept a PRE-FILTERED
      `[]*ast.Property` (PropertyKind-only) parameter** instead of reading `d.Properties`
      itself — so it structurally CANNOT re-emit the contract clauses already printed in
      signature position (reviewer nit, gemini-3-1-pro).
- [x] `internal/parser/parser_func.go:169`: `fn.Properties = p.parsePropertiesBlock()` →
      `fn.Properties = append(fn.Properties, p.parsePropertiesBlock()...)` — explicit slice
      unpacking (reviewer nit, gemini-3-1-pro); preserves contracts when a `properties [...]`
      block follows; append order matches print order.
- [x] Unit tests (`internal/format/`): round-trip fixtures for (a) requires-only, (b) ensures-only,
      (c) both, (d) multi-predicate `requires { a, b }`, (e) ensures containing `ForallExpr`,
      (f) genuine `forall(...)` properties block (no parse-valid corpus file exercises this —
      verified: only `examples/experimental/factorial.ail` contains `properties [` and it does
      not parse — so a synthetic fixture is REQUIRED), (g) contracts + properties combined
      round-trip (must assert the `requires` entry survives; the same fixture also feeds the
      acceptance-gated integration test in M2).

**M2: Regression guard + corpus green** (~3h) — ✅ IMPLEMENTED
- [x] **Acceptance-gated combined-case integration test** (blocks acceptance; see Success
      Criteria): check in a synthetic fixture `internal/format/testdata/contracts_and_properties.ail`
      containing BOTH a contract and a genuine properties block on one function:
      `export func f(x: int) -> int ! {}` with `requires { x >= 0 }`, `ensures { result >= 0 }`,
      AND `properties [ forall(y: int) => f(y) >= 0 ]`. A Go integration test
      (`TestCombinedContractsAndPropertiesPipeline`) asserts, after the parser fix:
      - **(a) checks clean**: the fixture runs the full check pipeline without error
        (equivalent to `ailang check` exit 0), and the M2 acceptance sweep additionally runs
        the CLI `ailang check` + `ailang ai-check --json` on it — `ai-check` must report the
        contract under `verify` (contract verification reached), not as a check error.
      - **(b) contract reaches contract verification**: elaborate the fixture and assert
        `core.DeclMeta.Contracts` for `f` contains EXACTLY one `RequiresKind` and one
        `EnsuresKind` contract and NO entry derived from the forall property.
      - **(c) forall reaches ONLY the property pipeline**: parse-level — `fn.Properties` is
        EXACTLY `[RequiresKind, EnsuresKind, PropertyKind]` in that order (proves no
        duplication or omission at the source of truth); collector-level — the testing
        collector's suite contains EXACTLY one `PropertyCase` with `Kind == PropertyKind`
        (the forall), and the contract-kind cases it also emits (today's pre-existing
        behavior, audit site 5) resolve through the runner's Kind-filter without error.
      - **(d) no duplication, omission, or panic**: assertion counts in (b)/(c) are exact
        (`== 1`, not `>= 1`); the test completing without panic covers panic-freedom; plus
        the M1(g) fmt round-trip on the same fixture asserts `cmp.Diff` AST identity with the
        `requires` clause present in the output.
- [x] `internal/format/corpus_comment_test.go`: harden the gate — `preExistingRT` moves from
      logged-and-tolerated into the `t.Fatalf` condition (`preExistingRT != 0` fails).
- [x] Run `ailang fmt --write` over the now-eligible files (30, not 28 — two adjacent Phase-1
      printer bugs were also fixed, making `scoring.ail` + `per_function_depth_verify.ail`
      formattable); verified `ailang check` (and `ailang ai-check` for the verify-corpus files)
      still passes on every reformatted file. `discount_calculator.ail` retains a PRE-EXISTING,
      unrelated `++`-on-string type error (expected-fail demo), not a reformat regression.
- [x] Acceptance sweep (see Success Criteria) — passes.
- [x] CHANGELOG.md entry; silent-deletion data-loss fix noted explicitly.

### Files to Modify/Create

**Modified:**
- `internal/format/decl.go` — contract-clause emission + kind split (~40 LOC)
- `internal/parser/parser_func.go` — 1-line append fix (line 169)
- `internal/format/corpus_comment_test.go` — gate hardening (~5 LOC)
- `internal/format/` test file (new or existing) — round-trip fixtures (~150 LOC)
- `examples/runnable/contracts/*.ail`, `examples/ai_devtools_workflow/discount_calculator.ail` —
  mechanical `fmt --write` reformat (28 files)
- `CHANGELOG.md`

**No new source files. No AST, lexer, type-system, eval, or verifier changes.**

## Conflict Surface

This touches the printer/AST-emission path (`internal/format/decl.go`) and one line of
`internal/parser/parser_func.go`. Enumeration:

**1. What syntactic positions does this change extend?**
The printer's function-declaration emission between the effect row and the body gains two new
clause forms (`requires { … }`, `ensures { … }`). The parser change extends nothing syntactic —
it only stops discarding already-parsed contract entries.

**2. What OTHER valid constructs already live in those positions?**
Between signature and body the parser accepts, in fixed order: `requires` block → `ensures`
block → `tests [...]` → `properties [...]` → body (`{ … }` block-form or `= expr` equation-form)
(`parser_func.go:114-190`). Extern functions have no body. The printer must emit contracts FIRST
in that window; emitting them after `tests`/`properties` would fail re-parse because
`parseContractBlocks` only runs before tests/properties parsing. The emission order in this
design (requires → ensures → tests → properties) matches the parser's fixed acceptance order.

**3. How does the parser disambiguate?**
`REQUIRES`/`ENSURES` are dedicated lexer keywords peeked before tests/properties/body
(`parser_contracts.go:22-56`); the body starts at `LBRACE`/`ASSIGN`. A `requires {` clause is
therefore unambiguous vs a body block `{`. Inside `properties [...]`, `parseProperty` requires a
leading `forall` — which is exactly why the current bare-expression emission fails, and why
after the split only `forall`-carrying `PropertyKind` entries may appear there.

**4. Existing programs/behaviors that MUST still work (regression fixtures — all verified to exist):**
- `examples/runnable/contracts/basic.ail` — requires+ensures, multi-predicate (`safeDivide`),
  `! {}` empty effect row before contracts.
- `examples/runnable/contracts/list_verify.ail` — ensures containing `forall i: lo..hi =>`
  (`ForallExpr`) predicates.
- `examples/runnable/contracts/quantifier_verify.ail`, `record_verify.ail`, `hof_verify.ail`,
  `string_verify.ail` — the ai-check/Z3 verify corpus; `ailang ai-check` must still pass after
  reformat.
- Genuine `forall(...)` properties blocks — round-trip GREEN at HEAD (live-verified with a
  synthetic file: output re-parses, AST identical). The kind-split must not disturb this path;
  the synthetic unit fixture in M1(f) locks it (no parse-valid corpus file covers it).
- `tests [...]` block emission (`testsBlock`, `decl.go:208`) — shares `testsAndProperties`;
  unchanged for functions without contracts.
- Equation-form vs block-form body identity (`funcBody`, `decl.go:148-163`) — contracts insert
  before the body writer runs; the Block-vs-bare re-parse identity invariant must be untouched.
- `! {}` nil-vs-empty effect-row distinction from the fmt Phase-1 fix
  (`formatEffectRow`, `internal/format/types.go:27-39`) — contract clauses are emitted AFTER the
  effect row; the fix must not reorder or suppress it.
- Phase-1/Phase-2 invariants: idempotence (`fmt(fmt(x)) == fmt(x)`, corpus-enforced), comment
  fail-closed envelope (the 2 `comment-unattached` contract files must STILL be refused with the
  same error, not silently formatted), and the fail-closed round-trip verifier in
  `cmd/ailang/fmt.go` (unchanged).

**5. What deliberately changes?**
- Files with `requires`/`ensures` stop exiting 2 and become formattable (the fix).
- A file with BOTH contracts and a properties block currently formats with exit 0 while
  **silently deleting the contracts**; after the parser append fix it formats losslessly. This is
  an intentional behavior change — the old behavior is a data-loss bug. Side effect: any
  consumer of `FuncDecl.Properties` now sees contract entries even when a properties block is
  present (previously clobbered). **Consumer audit — performed pre-approval, all sites read at
  HEAD (full table: Verification Log V17):** `elaborateContracts`
  (`internal/elaborate/file.go:277` → `internal/elaborate/file_funcs.go:75-102`, site 3)
  **explicitly Kind-filters** — entries whose Kind is not
  `RequiresKind`/`EnsuresKind`/`InvariantKind` are skipped, so forall `PropertyKind` entries
  never reach `core.DeclMeta.Contracts`. The property runner
  (`findLoweredContractPredicate`, `internal/testing/runner.go:560-585`, site 6) also
  **Kind-filters** (`if p.Kind != astKind { continue }`, runner.go:573-576) with documented
  same-kind-index matching into `DeclMeta.Contracts`. The test collector
  (`internal/testing/collector.go:136-146`, site 5) does **NOT** Kind-filter — it emits a
  `PropertyCase` per entry regardless of Kind — but it already receives
  `RequiresKind`/`EnsuresKind` entries today for every contracts-only function
  (`parser_func.go:169` only fires when a `properties [...]` block is present, so
  `fn.Properties` currently holds the contract entries appended at `parser_func.go:124-125`),
  and the resulting `PropertyCase.Kind` is filtered downstream by the runner (site 6). The
  append fix therefore introduces no new collector behavior beyond the currently-unexercised
  combined case, which is (contracts-only handling) ∘ (properties-only handling) concatenated
  in one slice. That the 28 contracts-only corpus files pass `ailang check` end-to-end today
  proves sites 3/4/5/6 already tolerate contract entries in `.Properties`; no corpus file
  combines both forms (V14), so the net blast radius of the parser fix is exactly the combined
  case — locked by the acceptance-gated integration test in M2.
- 28 checked-in example files get mechanically reformatted (`fmt --write`).

## Success Criteria

- [x] The minimal repro (`cf_contract.ail` above) formats: `fmt --check` exits 1 (needs
      reformat) or 0 — never exit 2 — and `fmt` output re-parses with identical AST. (Verified: exit 1.)
- [x] The contracts+properties combined file round-trips WITHOUT losing the `requires` clause.
- [x] **Acceptance-gated**: `TestCombinedContractsAndPropertiesPipeline` (M2) passes — the
      combined fixture checks clean, the contract reaches `DeclMeta.Contracts` (and `ai-check`
      verification), the forall reaches only the property pipeline, with exact-count
      (no-duplication/no-omission) assertions and no panic.
- [x] `for f in examples/runnable/contracts/*.ail examples/ai_devtools_workflow/*.ail` —
      zero exit-2 results from `ailang fmt --check`, excepting exactly
      `inbox_injection_v2.ail` and `inbox_v2_app.ail` (comment-attachment carve-out, unchanged
      error message).
- [x] After `fmt --write` on the eligible files (30): `ailang fmt --check` on them exits **0**, and
      `ailang check` passes on every one (verify-corpus files also re-pass `ailang ai-check`;
      `discount_calculator.ail` retains a pre-existing unrelated `++` type error).
- [x] `go test ./internal/format/`: `TestCorpusCommentGate` reports
      `preexisting-Phase1-rt-bug=0` and the hardened gate fails on any future non-zero count.
- [x] `make test` green; `make verify-examples` green (0 drift).
- [x] CHANGELOG.md updated.

## Testing Strategy

**Unit tests:** the seven round-trip fixtures in M1 (requires/ensures/both/multi-predicate/
ForallExpr-predicate/genuine-properties/combined), each asserting parse → print → re-parse →
`cmp.Diff` identity AND idempotence.

**Integration tests:** (1) the acceptance-gated combined-case test
`TestCombinedContractsAndPropertiesPipeline` (M2) — full pipeline over the
contracts+properties fixture: check-clean, contract → `DeclMeta.Contracts`/`ai-check`
verification, forall → property pipeline only, exact-count assertions, no panic; (2) the
hardened corpus gates (`TestCorpusCommentFreeRoundTrips`, `TestCorpusCommentGate`) sweep every
example automatically — the reformatted corpus itself becomes the regression fixture set.

**Manual:** `ailang fmt --check` sweep command from Success Criteria; spot-read one reformatted
verify file's diff to confirm contracts are byte-reasonable (clause per line, corpus style).

## Deferred Decisions

- Exact indentation of emitted `requires`/`ensures` clauses (column-0 flush with `func`, matching
  prevailing corpus style, vs one-level indent like `tests`/`properties`) — **agent may choose**;
  both re-parse identically, idempotence is the only constraint.
- Whether multi-predicate clauses break across lines past a width threshold — **agent may
  choose**; single-line `requires { a, b }` is acceptable for v1.
- Whether the `.Properties` consumer sanity sweep (Conflict Surface §5) warrants follow-up
  hardening (e.g., a dedicated `Contracts` accessor) — **agent may propose**, out of scope to
  implement.

## Non-Goals

- **Comment-attachment carve-out** (`comment-unattached` refusals, incl. the 2 inbox files) —
  separate Phase-2 limitation with its own enumerated evidence gate.
- **Phase-2 comment features generally** — this is a Phase-1 correctness fix.
- **`properties [...]` grammar changes** (named properties, bare-expression properties) — printer
  fix only; the parser's `forall`-required grammar is untouched.
- **fmt adoption / CI formatting enforcement** — belongs to m-ailang-fmt-adoption; this doc only
  makes the contract corpus eligible.
- **AST refactor** splitting contracts out of `FuncDecl.Properties` — print-time split is
  sufficient and avoids elaborator/verifier ripple.

## Timeline

**Day 1** (~7h): M1 printer split + parser append fix + unit fixtures (4h); M2 gate hardening,
corpus reformat, acceptance sweep, CHANGELOG (3h).

## Risks & Mitigations

| Risk | Impact | Mitigation |
|------|--------|-----------|
| Reformatting 28 checked-in examples churns diffs / breaks `verify-examples` manifest stats | Med | Reformat in a dedicated commit; re-run `make verify-examples`; known failure mode is manifest drift, not type errors |
| A `.Properties` consumer assumed the clobber (only-properties-or-only-contracts) | Med | Consumer audit COMPLETED pre-approval (Verification Log V17: 6 sites, per-site Kind-handling); residual risk locked by the acceptance-gated `TestCombinedContractsAndPropertiesPipeline` (M2) |
| Emitted clause order/position fails re-parse in an untested corner (e.g., extern funcs, equation-form bodies) | Low | The fmt round-trip verifier is fail-closed — any miss is exit 2, never silent; add extern/equation-form variants to fixtures if hit |
| Idempotence regression from new hardline placement | Low | Corpus idempotence check runs on every file automatically |

## Verification Log

All claims verified live at HEAD `v0.30.0-24-g5afa9a1e1` on 2026-07-20 unless noted.

| # | Claim | Method | Result |
|---|-------|--------|--------|
| V1 | Comment-free `requires`/`ensures` file passes `ailang check`, fails `fmt --check` exit 2 | live repro (Problem Statement) | ✅ Confirmed: check exit 0; fmt exit 2 "formatted output failed to re-parse (formatter defect)" |
| V2 | Printer routes ALL `Properties` (incl. contract kinds) into `properties [...]` | read `internal/format/decl.go:184-206, 259-304` | ✅ Confirmed: no `Kind` reference anywhere in `internal/format/` |
| V3 | `properties [...]` entries MUST start with `forall` → bare-expr emission cannot re-parse | read `internal/parser/parser_testing.go:242-245` | ✅ Confirmed: `PAR_UNEXPECTED_TOKEN "expected forall in property"` |
| V4 | Contract entries have `Binders: nil` → printed as bare exprs | read `internal/parser/parser_contracts.go:141-147` + `format/decl.go:285-304` | ✅ Confirmed |
| V5 | Genuine `forall(...)` properties blocks round-trip at HEAD (NOT part of the defect) | live synthetic file | ✅ Confirmed: `fmt --check` exit 1 (reformat only), `fmt` output re-parses |
| V6 | Corpus impact: 30 CLI exit-2 files; gate counts 28; the 2-file delta is the comment-attachment carve-out | live sweep + `go test -run TestCorpusCommentGate` + `ailang fmt` on both inbox files | ✅ Confirmed: gate logs `preexisting-Phase1-rt-bug=28`; both inbox files error "could not be attached to any boundary" |
| V7 | Parser clobber: `parser_func.go:169` assignment drops contracts when properties block present; fmt then exits 0 while DELETING `requires` | read + live combined-file repro | ✅ Confirmed: output lacks `requires { x >= 0 }`, exit 0 |
| V8 | fmt verifies BOTH re-parse and full AST identity (Kind/order enforced) | read `cmd/ailang/fmt.go:135-148` | ✅ Confirmed: `cmp.Diff(prog.File, reprog.File, fmtIgnorePos)` |
| V9 | Grammar fixes slice order requires*→ensures*; duplicates diagnosed; empty blocks drop out | read `parser_contracts.go:22-122` | ✅ Confirmed (incl. `PAR_DUPLICATE_REQUIRES/_ENSURES`) |
| V10 | `ForallExpr` inside contract predicates is already printable | grep/read `internal/format/expr.go:75,428` + `node_coverage_test.go:96` | ✅ Confirmed |
| V11 | `! {}` nil-vs-empty distinction exists and precedes contract position | read `internal/format/types.go:27-39`, `decl.go:77-79` | ✅ Confirmed |
| V12 | NEGATIVE: no `Kind` split exists anywhere in the printer | grep `Kind\|RequiresKind\|EnsuresKind` in `internal/format/` | ✅ Confirmed: zero hits |
| V13 | NEGATIVE: no parse-valid corpus file contains a `properties [` block (synthetic fixture required for M1(f)) | grep corpus + `ailang check examples/experimental/factorial.ail` | ✅ Confirmed: only factorial.ail matches and it does NOT parse (property-syntax parse errors) |
| V14 | NEGATIVE: no corpus file combines contracts AND a properties block (clobber is latent) | grep sweep over `examples/runnable/contracts/` + `ai_devtools_workflow/` | ✅ Confirmed: zero files |
| V15 | Cited regression fixtures exist | `ls`/`head` on basic.ail, list_verify.ail, quantifier_verify.ail, etc. | ✅ Confirmed (basic.ail content read; all listed files appear in the live failure sweep) |
| V16 | Gate hardening target: `preExistingRT` currently tolerated (logged, excluded from `t.Fatalf`) | read `internal/format/corpus_comment_test.go:117-125` | ✅ Confirmed |
| V17 | Consumer audit: every `FuncDecl.Properties` read site in `internal/` enumerated and its Kind-handling read (excl. `internal/format/`, which this doc rewrites, and `_test.go`) | `grep -rn "\.Properties" internal/` + read each site | ✅ 6 sites total; per-site table below |

### V17 — `FuncDecl.Properties` read-site audit (all sites at HEAD)

| # | Site | Role | Kind-handling | Affected by the append fix? |
|---|------|------|---------------|------------------------------|
| 1 | `internal/parser/parser_func.go:124-125` | producer | writes `RequiresKind` then `EnsuresKind` entries (from `parseContractBlocks`) | No — unchanged |
| 2 | `internal/parser/parser_func.go:169` | producer — **the bug** | `fn.Properties = p.parsePropertiesBlock()`: the `=` clobbers the contract entries from site 1 whenever a `properties [...]` block follows | **YES — the fix site**: `=` → `fn.Properties = append(fn.Properties, p.parsePropertiesBlock()...)` |
| 3 | `internal/elaborate/file.go:277` → `elaborateContracts` (`internal/elaborate/file_funcs.go:75-102`) | consumer | **filters**: `if prop.Kind != ast.RequiresKind && prop.Kind != ast.EnsuresKind && prop.Kind != ast.InvariantKind { continue }` (file_funcs.go:83) — only contract kinds reach `core.DeclMeta.Contracts`; forall entries skipped | No — appending contract entries to a slice it already reads is its expected input; forall entries explicitly skipped |
| 4 | `internal/elaborate/file.go:415` (`Props: f.Properties`) | pass-through | none — stores the raw slice on `FuncSig`; downstream consumer is the property runner (site 6), which is Kind-aware | No — pure plumbing |
| 5 | `internal/testing/collector.go:136-146` | consumer | **does NOT filter** — emits a `PropertyCase` per entry, any Kind | No new behavior: it already processes contract entries today on contracts-only functions (site 2 doesn't fire without a properties block), and `PropertyCase.Kind` is filtered downstream by site 6. The combined case is newly exercised → locked by the M2 acceptance-gated integration test |
| 6 | `internal/testing/runner.go:560-585` (`findLoweredContractPredicate`) | consumer | **filters**: `if p.Kind != astKind { continue }` (runner.go:573-576); documented same-kind-index → `DeclMeta.Contracts` matching ("The elaborator skips forall properties when emitting Contracts … requires/ensures contracts are emitted in their original source order", runner.go:556-559) | No — designed for the mixed-kind slice |

## Related Documents

**Implemented (inform design):**
- [design_docs/implemented/v0_30_0/m-ailang-fmt.md](../../implemented/v0_30_0/m-ailang-fmt.md) — Phase-1 printer, fail-closed round-trip verifier, `! {}` fix (neural 0.37)
- [design_docs/planned/v0_30_0/m-ailang-fmt-phase2-sprint-plan.md](m-ailang-fmt-phase2-sprint-plan.md) — comment preservation; its V22 corpus gate surfaced this bug (neural 0.39)

**Planned (checked for overlap — distinct):**
- [design_docs/planned/v0_30_0/m-ailang-fmt-adoption.md](m-ailang-fmt-adoption.md) (0.36) — corpus-wide adoption/enforcement; THIS doc only makes the contract class eligible. No scope overlap: adoption consumes this fix.

## Future Work

- Comment-attachment carve-out reduction (would recover the 2 inbox files).
- Optional `FuncDecl.Contracts` accessor / AST-level split if `.Properties` consumers accumulate.

---

**Document created**: 2026-07-20
**Last updated**: 2026-07-20 (rev 2 — quorum revision pass: consumer audit V17 encoded, §5
corrected, combined-case integration test acceptance-gated, gemini impl nits folded into M1)
