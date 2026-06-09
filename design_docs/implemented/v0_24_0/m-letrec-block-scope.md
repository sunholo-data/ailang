# M-LETREC-BLOCK-SCOPE — Block-form `letrec` dropped its binding

**Status:** Implemented (2026-06-09)
**Severity:** High — `letrec` (recursive local bindings) was unusable in the form the
teaching prompt documents; surfaced as a core-tier eval gap (`config_file_parser`).
**Area:** `internal/elaborate` (ANF normalization)

## Summary

A statement-form `letrec` inside a block —

```ailang
export func main() -> int {
  letrec factorial = \n. if n <= 1 then 1 else n * factorial(n - 1);
  factorial(5)
}
```

— failed type-checking with `undefined variable: factorial`. The recursive name was
never in scope, neither in its own value nor in the rest of the block. **This is the
exact form the teaching prompt's canonical `letrec` example uses** (`prompts/v0.9.0.md`
line 286), so we were teaching a non-working pattern.

## How it was found

The first local-qwen **core** eval baseline (2026-06-09) listed `config_file_parser`
as a hard gap (`undefined variable: helper`). Its solution used a local recursive
helper via block-form `letrec`. The eval-gap-finder reduced it to the minimal repro
above and confirmed the prompt's own example fails identically.

## Root cause

The expression form `letrec f = ... in body` worked (it goes through
`Elaborator.normalizeLetRec`, which emits a `core.LetRec`, and the type checker's
`inferLetRec` correctly extends the environment with the binding *before* inferring
the value — proper recursive scoping).

The **statement form** does not carry a body; the continuation is the rest of the
enclosing block. `Elaborator.normalizeBlock` walks the block's statements backward,
wrapping each in the continuation. It special-cased `*ast.Let` (threading the binding
into the continuation) but **had no case for `*ast.LetRec`**. So a block-level
`letrec f = ...;` fell through to the generic branch, which normalized the whole
statement and bound it to a discarded `_block_N` wildcard. The bound name `f` never
reached the continuation — and because the discarded statement's body was `()`, the
recursive self-reference had nowhere to resolve either.

## Fix

Add an `*ast.LetRec` case to `normalizeBlock`'s statement loop, mirroring the existing
`*ast.Let` case but emitting a `core.LetRec` whose `Body` is the threaded continuation:

```go
} else if letrecExpr, ok := expr.(*ast.LetRec); ok && letrecExpr.Body == nil {
    value, err := e.normalize(letrecExpr.Value)
    if err != nil { return nil, err }
    result = &core.LetRec{
        CoreNode: e.makeNode(letrecExpr.Position()),
        Bindings: []core.RecBinding{{Name: letrecExpr.Name, Value: value}},
        Body:     result, // thread through to the rest of the block
    }
}
```

`inferLetRec` was already correct, so this single elaboration case closes the bug for
single-arg, curried, multi-param, and "letrec-after-another-statement" forms.

## Verification

- New Go test `internal/pipeline.TestLetRecBlockScope` (ModeCheck) — passes with the
  fix, fails without (confirmed by stashing the change).
- `examples/runnable/letrec_recursion.ail` rewritten to cover **both** forms — the
  pre-existing version used only the always-working `... in ...` expression form,
  which is why CI never caught the regression. Runs clean on the fixed binary, fails
  (`undefined variable: sumTo`) on the old one.
- Full Go suites (`elaborate`/`types`/`pipeline`/`repl`) green; `make verify-examples`
  at baseline (181 passed / 5 failed / 2 skipped — unchanged).

## Known remaining limitation

**Top-level `letrec`** (`letrec fac = \n. ... fac(...)` as a module-level declaration,
outside any function body) is still broken (`undefined variable: fac`) — a separate
elaboration path. The idiomatic and working alternative at module scope is a top-level
recursive `func`, which is what the prompt and stdlib already use. Tracked as a
follow-up; not addressed here to keep the fix focused on the in-block form that the
prompt teaches and that benchmarks actually hit.

## Process note

The feature had **zero `examples/` coverage of the statement form** despite being
documented in the prompt. `verify-examples` is also advisory in CI (`|| true`), so even
the existing (working-form) example wouldn't have gated. The fix therefore adds a
gating Go test in addition to the example.
