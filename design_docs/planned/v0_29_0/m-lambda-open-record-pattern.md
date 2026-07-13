# M-LAMBDA-OPEN-RECORD-PATTERN — Lambda + `{field, ...}` pattern doesn't propagate row polymorphism

**Status**: Planned — P2 (pre-existing, surfaced post-fix)
**Target**: v0.22.x or v0.23.0
**Priority**: P2 — pre-existing limitation, only surfaced when M-SCHEME-IMPORT-PRESERVE-ADT-HEAD removed the over-polymorphism that was hiding it.
**Estimated**: ~1 day (elaborate/typecheck/generalize trace)
**Dependencies**: None
**Source**: Surfaced 2026-05-20 during M3 regression sweep of [M-SCHEME-IMPORT-PRESERVE-ADT-HEAD](../implemented/v0_22_0/m-scheme-import-preserve-adt-head.md). [examples/runnable/record_patterns.ail](../../examples/runnable/record_patterns.ail) was relying on the bug; refactored as a workaround.

## Reproducer

```ailang
let getName = \obj.
  match obj {
    {name, ...} => name      -- explicit open-record pattern
  }
in
getName({name: "Grace", id: 123})   -- caller passes record with extra field
```

Result (post-v0.22.0):

```
Error: failed to unify parameter 0: record field mismatch: expected 1 fields, got 2
  expected fields: {name}
  actual fields:   {id, name}
  extra fields:    id
```

The lambda's parameter type ends up as a CLOSED record `{name: τ}` rather than an OPEN `{name: τ | r}`, despite the user explicitly writing `{name, ...}`.

**Top-level `let ... in match` (without the lambda) works correctly:**

```ailang
let big_record = {name: "Frank", age: 40, city: "NYC", active: true} in
match big_record {
  {name, ...} => name      -- works: produces "Frank"
}
```

So the issue is specifically when the scrutinee is a lambda parameter (a fresh TVar at pattern-check time, generalized to a closed record at lambda-exit).

## Root cause hypothesis

Two suspected sites:

### 1. Elaborator drops the Rest flag

[internal/elaborate/patterns.go:177-189](../../internal/elaborate/patterns.go#L177-L189) elaborates `ast.RecordPattern` to `core.RecordPattern` but the destination type has NO Rest field — the open/closed intent is erased at the AST→Core boundary:

```go
case *ast.RecordPattern:
    fields := make(map[string]core.CorePattern)
    for _, fp := range p.Fields {
        corePat, err := e.elaboratePattern(fp.Pattern)
        if err != nil { return nil, err }
        fields[fp.Name] = corePat
    }
    return &core.RecordPattern{Fields: fields}, nil
    // ^^ no Rest field on core.RecordPattern at all
```

### 2. Typechecker only uses rowVar when scrutinee is unresolved

[internal/types/typechecker_patterns.go:365-387](../../internal/types/typechecker_patterns.go#L365-L387) has two branches:

```go
case *core.RecordPattern:
    if recTy, ok := scrutType.(*TRecord); ok {
        fieldTypes = recTy.Fields            // CLOSED branch — uses scrut's fields verbatim
    } else {
        // ... builds &TRecord{Fields, Row: rowVar}  -- OPEN branch
    }
```

For lambda parameters at pattern-check time, scrutType IS a fresh TVar → branch #2 fires and produces an open-record constraint. So the constraint set should be sound.

### 3. (Most likely) generalize doesn't quantify the row variable

After my v0.22.0 fix to apply substitution before generalization, the lambda's parameter type gets resolved to `{name: τ | rowVar}`. `generalize` then walks the result for free type variables — but if it doesn't also quantify free ROW variables (or quantifies them but doesn't preserve the row through instantiation at the call site), the row variable gets eagerly bound to `<empty>` and the parameter type collapses to closed `{name: τ}`.

Pre-fix the substitution wasn't applied, so the lambda parameter stayed as a bare TVar and the call site unified with anything (over-polymorphic — wrong for type safety but accidentally accepted the test case).

## Goals

1. Make `\obj. match obj { {name, ...} => name }` produce a parameter type of `{name: τ | r}` (open record polymorphism) — not `{name: τ}`.
2. The reproducer above passes `ailang check` cleanly.
3. The existing top-level let+match case continues to work.
4. The closed-record case `\obj. match obj { {name} => name }` STILL rejects callers with extra fields (the new strictness should not be silently reversed).

## Proposed approach

1. **Add `Rest bool` to `core.RecordPattern`** and propagate in elaborate/patterns.go.
2. **Use the Rest flag in checkPattern**: when `Rest == true`, always produce a constraint with a rowVar, even when the scrutinee is already a TRecord (override the recTy.Row).
3. **Trace generalize for row-variable handling**: confirm row vars are quantified and that scheme instantiation produces a fresh row variable. If not, fix.
4. Add a regression test covering both lambda+open-pattern AND lambda+closed-pattern (with the closed one expected to reject extra-field callers — the strictness regression matters).

## Acceptance criteria

- [x] [`examples/runnable/record_patterns.ail`](../../examples/runnable/record_patterns.ail) can be restored to use `{name, ...}` patterns in lambdas with extra-field callers.
- [x] New regression test in `internal/pipeline/lambda_open_record_test.go` covers the four shapes:
  - Lambda + open pattern + extra-field caller → PASS
  - Lambda + open pattern + matching-shape caller → PASS
  - Lambda + closed pattern + matching-shape caller → PASS
  - Lambda + closed pattern + extra-field caller → FAIL (correctness check)
- [x] No regression in M-SCHEME-IMPORT-PRESERVE-ADT-HEAD tests or any other type-system tests.

## Risks

- **Row-variable generalization** is subtle in HM with rows. Care needed not to over-generalize and accept genuinely incompatible records.
- **Backwards compatibility**: code that currently works because of the over-polymorphism that hid this bug (other than `record_patterns.ail`) may need similar refactoring. Mitigation: full `make verify-examples` + go test sweep after the fix.

## Out of scope

- Open-record patterns inside `let`-bindings or non-match contexts.
- Row subtyping (different problem; row polymorphism is the AILANG approach).

## Related

- [M-SCHEME-IMPORT-PRESERVE-ADT-HEAD](../implemented/v0_22_0/m-scheme-import-preserve-adt-head.md) — sibling sprint that surfaced this gap.
- M-FIX-RECORD-UPDATE — historical row-polymorphism work; check for related decisions.
