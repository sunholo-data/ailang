# M-TYPECHECK-NO-AUTO-UNWRAP-RESULT: Reject `.field` access on tagged unions without `match`

**Status**: Planned
**Target**: v0.20.0
**Priority**: P0
**Estimated**: ~3 days (~20 hours)
**Dependencies**: None (pure type-checker change + audit-fix sweep across stdlib + packages)
**Author**: Claude Opus 4.7 + Mark
**Created**: 2026-05-14
**Source**: 2026-05-13 PR #16 (motoko_agent) post-mortem. Bug class: AILANG silently auto-unwraps tagged unions during field access, masking missing `match` discipline. Latent bugs ship and fire only on the unhappy path (network blip, rate limit, model timeout, missing handler). Reported by `arniwesth` after `motoko_ext_compaction_ai 0.1.3` crashed his agent loop with `cannot access field of non-record value: *eval.StringValue` — the AILANG error message is itself misleading because by the time `.content` is being accessed, AILANG has already auto-unwrapped `Err(AIError)` to its `.message: string` field.

---

## Axiom Compliance

| Axiom | Score | Justification |
|-------|-------|---------------|
| A1: Determinism | 0 | Pure type-system change; no runtime semantics shift |
| A2: Replayability | 0 | No effect on traces |
| A3: Effect Legibility | +1 | Result-typed effect outcomes (network, AI, FS) become structurally inescapable — every consumer must declare what it does on Err |
| A4: Explicit Authority | 0 | No capability-system change |
| A5: Bounded Verification | +2 | This IS the verification — moves an entire bug class from "fires only on the unhappy path" to "rejected at compile time" |
| A6: Safe Concurrency | 0 | Not concurrency-related |
| A7: Machines First | +2 | AI-generated code currently passes `ailang check` despite missing `match` discipline. The check that today gives a false ✓ becomes a hard ✗ with a prescriptive fix message — AI agents iterate to a correct fix in one round instead of "passes type-check, crashes in production" |
| A8: Minimal Syntax | 0 | No new syntax; tightens semantics of existing field-access syntax |
| A9: Cost Visibility | 0 | Indirect: fewer wasted tokens on debugging crash logs that should have been compile errors |
| A10: Composability | +1 | Eliminates a category of cross-package incompatibility (a stdlib change from `T` to `Result[T, E]` no longer silently breaks downstream `.field` accesses; they refuse to compile) |
| A11: Structured Failure | +2 | The whole point. Forces every consumer of a Result-returning function to author an Err branch. No more "I forgot the Err case happens." |
| A12: System Boundary | 0 | No boundary change |

**Net Score: +8** → **Decision: ✅ Move forward**

### Hard Violation Check
- [x] A1, A3, A4, A7 — none violated; A3 + A7 + A11 strongly improved

---

## Problem Statement

AILANG today permits direct field access on values typed as a tagged union. The compiler's record-access elaboration silently descends into the union: when the runtime value is `Ok(StepResult)`, the access works; when it's `Err(AIError)`, the field name is looked up on AIError instead, returning a different type than the caller expected — at which point a downstream access crashes with a misleading `cannot access field of non-record value: *eval.<TypeName>Value` error.

Concrete recent example (motoko_ext_compaction_ai 0.1.3, fixed in 0.1.4):

```ailang
let result = step(model, msgs, []);
result.message.content    -- type-checks ✓ — but is wrong
```

`step` returns `Result[StepResult, AIError]`. The Ok path silently works:
- `result.message` → auto-unwrap Ok → access `.message` on StepResult → returns `Message{role, content, tool_calls, tool_call_id}`
- `.content` → access on Message → returns `string` ✓

The Err path crashes:
- `result.message` → auto-unwrap Err → access `.message` on AIError → returns `string` (AIError's `.message: string`)
- `.content` → access on a `string` → 💥 `*eval.StringValue` runtime panic, misleading error message

This bug existed in production for 3 days and fired only when a long-running agent loop hit a transient `Err` from `step` (rate limit, network blip, timeout). Manual review missed it. `ailang check` passed. The smoke harness only exercised `register_with_config`, never the `step` callsite.

**The systemic issue**: AILANG's auto-field-access on tagged unions is a language footgun that silently masks the absence of `match` discipline. Every consumer of every Result-returning function in stdlib (`std/ai`, `std/net`, `std/fs`, `std/json`, ...) is exposed to the same class.

**Reach** (pre-fix audit, est.):
- ~150-200 callsites across stdlib + 30+ published packages
- Most ARE correct (they `match` first). The ones that AREN'T are landmines that fire on the rare Err path.
- AI-generated AILANG code is especially exposed — LLMs frequently emit `result.field` chains and "tested by running once" passes the happy path.

---

## Goals

**Primary Goal**: AILANG's type-checker rejects `.field` access on a value typed as a tagged union (`Result[T, E]`, `Option[T]`, user-defined ADTs) unless the access is inside a `match` arm that has bound a constructor's payload to a name. Forces every consumer of a Result-returning function to author both an `Ok` and an `Err` branch.

**Success Metrics**:
- Type-checker rejects `result.field` on `Result[T, E]` with a prescriptive error pointing to the missing `match`
- All in-repo AILANG code (stdlib examples + tests + bundled examples) audited and fixed (~50-150 fixups expected)
- Audit script (`tools/find_unsafe_field_access.sh`) ships and is wired into CI
- Drift-checker for the published-package corpus (every `sunholo/*` package fetched from the registry, type-checked locally, all pass)
- AI-targeted prompts (`ailang prompt`, `ailang devtools-prompt`) updated to teach the new discipline and the prescriptive error message
- The motoko_ext_compaction_ai-style bug is structurally impossible to ship — `summarize_with_ai` won't compile without an Err arm

---

## High-Impact Decisions

| Decision | Why High Impact | Chosen By | Deadline | Change Cost |
|----------|-----------------|-----------|----------|-------------|
| Whether to error or warn on first release | Hard error breaks every existing package; warn-then-error gives one cycle to migrate | human | design | high |
| Scope: only `.field` access, or also subscript / pattern-match-without-arm? | Scope creep risk; `.field` covers 95% of the bug class | agent | design | med |
| Allow `.field` on tagged-union types where every variant has the same field with the same type? | Gives auto-unwrap a principled escape hatch; opens "but my variants differ in subtle ways" rabbit hole | agent | design | med |
| Migration path for stdlib auto-field accesses we want to keep | Some stdlib uses might be intentional (e.g., builder patterns); need an opt-in marker (`@unsafe-field-access` or similar) | agent | compile | high |
| Error message format and "prescribed fix" hint | Determines whether AI agents one-shot or thrash on the new error | agent | compile | low |

### Design Freeze
- [ ] **Severity**: hard error in v0.20.0 (NOT a warning). Net win is on AI-self-correct loops; warnings get ignored. Migration grace via the audit script + a one-version `--allow-unsafe-field-access` rescue flag (warns loudly on every use). Locked.
- [ ] **Scope**: phase 1 covers `.field` access only. Subscript (`xs[i]` on lists, `m[k]` on maps) is already a different bug class (Out-of-Bounds, MissingKey) and gets its own sprint. Locked.
- [ ] **Same-field-shape exception**: allow `.field` on a tagged union if every variant is a record AND every variant has the same field name with a structurally compatible type. Rare but principled — covers e.g. `Result[Record{x:int}, Record{x:int, err: string}]` where `.x` is unambiguous. Author flag this case in tests; agent decides based on whether it adds >50 LOC of compiler complexity. Soft locked.
- [ ] **Migration flag**: `--allow-unsafe-field-access` flag (default off) downgrades the error to a warning so package authors can update at their pace within v0.20.x. Removed in v0.21.0. Locked.

---

## Solution Design

### Overview

Type-checker phase: when elaborating a record-access expression `expr.field`, if `type-of(expr)` is a tagged union (Result, Option, or any user-defined ADT with multiple constructors), reject with `RECORD_ACCESS_ON_TAGGED_UNION` and a prescriptive fix.

The bug-class boundary is precise: a value is "tagged-union-typed" if its inferred type is a constructor application whose head is an ADT with more than one constructor. `Result[T, E]` qualifies. A user record like `{a: int, b: string}` doesn't.

### Architecture

```
internal/types/typechecker.go
  ↓ inferring a *ast.RecordAccess
  ↓
isTaggedUnion(receiverType) → bool
  ↓ true
ErrRecordAccessOnTaggedUnion at receiver position
  with hint:
    "step(...) returns Result[StepResult, AIError]. Use match:
       match step(model, msgs, []) {
         Ok(result) => result.message.content,
         Err(e)     => /* handle e */
       }"
  ↓ false
proceed with existing record-access elaboration
```

### Components

1. **`internal/types/typechecker.go::inferRecordAccess`** (~30 LOC): the gating check before existing elaboration. Calls `isTaggedUnion(receiverType)`; on true, returns the new error. Pre-existing single-constructor ADT field-access continues to work (e.g., a record-shaped single-variant ADT used as a struct).

2. **`internal/types/tagged_union_predicate.go`** (NEW, ~80 LOC): `isTaggedUnion(t Type) bool` resolves the type, recursively unwraps type aliases, and returns true if the resolved head is a multi-constructor ADT. Plus the same-field-shape exception per the Soft Locked design freeze item (~50 LOC if we land it; gate behind a separate boolean).

3. **`internal/elaborate/error_codes.go`**: register `TYP_RECORD_ACCESS_ON_TAGGED_UNION` with the prescriptive hint template (sub-in receiver type + first-three-constructor-names). Gets surfaced via `ailang check`'s structured-error envelope.

4. **`tools/find_unsafe_field_access.sh`** (NEW, ~50 LOC): pre-migration audit. Greps for `.field` access patterns where the receiver is bound from a function known to return a Result/Option. Limited (it's regex-based, not type-aware) but catches the obvious ones. Output: file:line:expr.

5. **Migration flag** (`--allow-unsafe-field-access`): in `cmd/ailang/main.go`, downgrades `TYP_RECORD_ACCESS_ON_TAGGED_UNION` errors to WARN-level diagnostics. Logged-but-loud so they're noticed.

6. **`prompts/v0.20.0/syntax.md`** + `devtools.md`: teach the discipline.

7. **In-repo audit-fix sweep**: every `.ail` file under `examples/`, `stdlib/`, `internal/builtins/`, `examples/runnable/` is type-checked; failures are fixed by adding `match`. Estimated 30-60 fixes (most stdlib code is already match-disciplined).

8. **Published-package validator**: a CI job that fetches the latest release of every `sunholo/*` package from the registry, type-checks each with the new strict checker, surfaces any failures. Helps coordinate the package-side migration before v0.20.0 ships.

### Implementation Plan

**Phase 1 — Type-checker core** (~1 day)
- Day 1 AM: `isTaggedUnion` + `inferRecordAccess` gate + error code. Failing tests in `internal/types/tagged_union_field_access_test.go` for `Result.field`, `Option.field`, user-ADT.field. Implementation passes the tests.
- Day 1 PM: `--allow-unsafe-field-access` migration flag. Hint message tuning + golden-file tests for the structured-error envelope.

**Phase 2 — Audit sweep** (~1 day)
- Day 2 AM: `tools/find_unsafe_field_access.sh`. Run against entire repo; capture failures.
- Day 2 PM: Fix every in-repo failure. Most are 1-line additions of an Err arm. ~30-60 sites expected.

**Phase 3 — Ecosystem migration** (~1 day)
- Day 3 AM: Run the strict checker against every published `sunholo/*` package. Catalog failures.
- Day 3 PM: PRs to each affected package (similar to how M-WASM-AI-STEP-BYO-KEY's compaction_ai 0.1.4 fix already landed). Update docs guide + prompts. CHANGELOG + design-doc move to implemented.

### Files to Modify/Create

**New files:**
- `internal/types/tagged_union_predicate.go` (~80 LOC) — `isTaggedUnion` predicate
- `internal/types/tagged_union_field_access_test.go` (~150 LOC) — failing-tests-first
- `tools/find_unsafe_field_access.sh` (~50 LOC) — pre-migration audit
- `prompts/v0.20.0/syntax.md` updates — teach the new discipline (~30 LOC)

**Modified files:**
- `internal/types/typechecker.go` (~+30 LOC) — wire the gate
- `internal/elaborate/error_codes.go` (~+15 LOC) — register the new error code
- `cmd/ailang/main.go` (~+10 LOC) — `--allow-unsafe-field-access` flag plumbing
- ~30-60 `.ail` files across `examples/runnable/`, `examples/`, `std/` — add Err arms
- `changelogs/v0.10-current.md` (~+50 LOC entry under [v0.20.0])

---

## Examples

### Example 1: The bug we just shipped (compaction_ai 0.1.3)

**Before (type-checks today, crashes at runtime on Err):**
```ailang
import std/ai (Message, step)

func summarize_with_ai(prompt: string, model: string) -> string ! {AI} {
  let msgs: [Message] = [{ role: "user", content: prompt, tool_calls: [], tool_call_id: "" }];
  let result = step(model, msgs, []);
  result.message.content   -- ❌ v0.20.0: TYP_RECORD_ACCESS_ON_TAGGED_UNION
                           -- "step returns Result[StepResult, AIError].
                           --  Use match { Ok(result) => ..., Err(e) => ... }"
}
```

**After (compiles in v0.20.0):**
```ailang
import std/ai (Message, step)
import std/result (Result, Ok, Err)

func summarize_with_ai(prompt: string, model: string) -> string ! {AI} {
  let msgs: [Message] = [{ role: "user", content: prompt, tool_calls: [], tool_call_id: "" }];
  match step(model, msgs, []) {
    Ok(result) => result.message.content,
    Err(e)     => "[summarizer unavailable: ${e.code}: ${e.message}]"
  }
}
```

### Example 2: Single-constructor record (still works)

```ailang
type Point = { x: int, y: int }

func origin_distance(p: Point) -> int {
  p.x + p.y                 -- ✓ works in v0.20.0 — Point isn't tagged-union
}
```

### Example 3: Same-field-shape exception (if landed)

```ailang
type CallResult =
  | Success({ tokens: int, content: string })
  | Cached({ tokens: int, content: string })

func tokens_used(r: CallResult) -> int {
  r.tokens                  -- ✓ optionally works in v0.20.0 — both variants have tokens:int
}
```

---

## Success Criteria

- [ ] Type-checker rejects `.field` access on a tagged-union receiver
- [ ] Error message includes a prescriptive `match` template with actual type info substituted
- [ ] `--allow-unsafe-field-access` migration flag downgrades to warn
- [ ] All in-repo `.ail` files type-check under strict mode (audit-fix sweep complete)
- [ ] All published `sunholo/*` packages type-check under strict mode
- [ ] `tools/find_unsafe_field_access.sh` ships, used by the audit
- [ ] AI prompts (`ailang prompt` v0.20.0) teach the new discipline
- [ ] CHANGELOG entry under [v0.20.0] documents the breaking change + migration path
- [ ] Sprint-evaluator scores ≥85/100
- [ ] All Go-side tests pass: `make test`
- [ ] `make lint` clean

---

## Testing Strategy

**Unit tests** (`internal/types/tagged_union_field_access_test.go`):
- `Result[Int, String].foo` → `TYP_RECORD_ACCESS_ON_TAGGED_UNION`
- `Option[Record{a:int}].a` → `TYP_RECORD_ACCESS_ON_TAGGED_UNION`
- `MultiVariantADT.field` → `TYP_RECORD_ACCESS_ON_TAGGED_UNION`
- `SingleVariantADT.field` → ALLOWED (exception 1: single-ctor wraps)
- `Record{a:int}.a` → ALLOWED (not a tagged union)
- Inside `Ok(result) => result.field` — ALLOWED (receiver is now `T`, not `Result[T, E]`)
- `--allow-unsafe-field-access` flag: error becomes warning, code still type-checks

**Integration tests:**
- The exact compaction_ai 0.1.3 source as a fixture: must fail to compile. Add to `examples/expected_fail/`.
- The compaction_ai 0.1.4 source: must compile.
- Drift checker: every published `sunholo/*` package fetched and type-checked.

**Manual verification:**
- Replay arniwesth's PR #16 scenario: `make verify_extensions` against compaction_ai 0.1.3 must fail with the new error before runtime would have crashed.

---

## Conflict Surface

**Touches `internal/types/`** — triggers the regression-surface evaluator rule.

- `internal/types/typechecker.go` — adds the gate. Risk: existing programs that legitimately use `.field` on a tagged union (e.g., for builder patterns or struct-emulation via single-constructor ADTs) suddenly fail. Mitigation: same-field-shape exception (if landed), single-ctor ADT exception (always), and the `--allow-unsafe-field-access` migration flag.
- `internal/types/tagged_union_predicate.go` — net new. Single function, well-scoped.
- `internal/elaborate/error_codes.go` — new error code, additive.
- `cmd/ailang/main.go` — new flag, additive.

**Programs that MUST still work** (regression fixtures):
1. Every `examples/runnable/*.ail` after the audit-fix sweep — `make verify-examples` green.
2. The `std/ai`, `std/net`, `std/fs`, `std/json` example programs in their `.ail` modules.
3. Hindley-Milner type inference unchanged — no regression in `internal/types/` test suite (~600 tests).
4. Pattern-matching elaboration unchanged.

**Deliberately changes** (intentional incompatibilities):
1. Programs that did `result.field` on a `Result`-typed value WITHOUT match-binding — these become compile errors. This is the entire point.

---

## Timeline

**Day 1**: Type-checker core + tests + migration flag
**Day 2**: Audit sweep + in-repo fixes
**Day 3**: Ecosystem migration + docs + CHANGELOG + ship

**Total: ~3 days (~20 hours)**

## Risks & Mitigations

| Risk | Impact | Mitigation |
|------|--------|-----------|
| Audit sweep finds too many fixes (>200) | Med | If the count balloons, ship the migration flag with WARN-default for one cycle, hard-error in v0.21.0. Falls back to "Option B (warn-then-error)" without re-planning |
| Same-field-shape exception adds compiler complexity | Med | Defer to a follow-up sprint if it grows past 50 LOC. The simple version (always reject) is the floor and ships if exceptions blow scope |
| External package authors miss the migration | Med | Drift checker against the registry catches before v0.20.0 ships; PRs land alongside the AILANG release |
| AI agents emit `result.field` patterns from training | High at first, declining | Update prompts (`ailang prompt` + `devtools-prompt`) to lead with the discipline. The new error message includes the prescribed fix — agents one-shot to correctness |
| Single-ctor ADTs being used as struct-emulation suddenly start being flagged | Low | The `isTaggedUnion` predicate explicitly EXCLUDES single-constructor ADTs (always allow `.field` access). Test fixture pins this behavior |

---

## Related Documents

**Source**:
- 2026-05-13 PR #16 incident on `arniwesth/motoko_agent` — the bug that motivated this design
- `motoko_ext_compaction_ai 0.1.4` fix (reactive, in production) — the proof that the bug class is real and downstream

**Companion v0.20.0 work** (sister sprints, complementary):
- M-SMOKE-FAULT-INJECTION (planned) — `ailang publish --smoke-with-faults` injects synthetic Err returns at every Result-returning effect callsite. Catches missing-match-arm bugs at publish time. Belt-and-braces with this design (different mechanism, different coverage).

**Builds on**:
- AILANG type-checker stack (`internal/types/`)
- Existing structured error envelope (`internal/elaborate/error_codes.go`)
- M-EXT-PORTABILITY-GATE (v0.18.11) — established the published-package validator infrastructure this design's drift-check would reuse

---

## Future Work

- **Phase 2 — subscript discipline**: `xs[i]` on lists / `m[k]` on maps should similarly require an `Option[T]` return type rather than panicking on out-of-bounds. Separate sprint.
- **`.field` on records-with-extension-types**: verify behavior on row-polymorphic records is unchanged (they're not tagged unions, so the gate doesn't fire — but worth a regression fixture).
- **Auto-fix tool**: `ailang fix --add-err-arm` could mechanically add `Err(e) => /* TODO */` to every flagged callsite. Lower priority than the type-checker gate itself.

---

**Document created**: 2026-05-14
**Last updated**: 2026-05-14
