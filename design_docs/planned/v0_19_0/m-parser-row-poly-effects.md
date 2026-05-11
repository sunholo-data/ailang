# M-PARSER-ROW-POLY-EFFECTS: row-extension `|` sugar for effect annotations

**Status**: IMPLEMENTED — both Phase 1 (smoke.ail rewrite) and Phase 2 (`|` sugar) shipped same session.
**Target**: v0.19.0 (Phase 1 + Phase 2 both)
**Priority**: P1 → P2 after discovery (parser change was 60 LOC, tests 90 LOC)
**Estimated**: 0 LOC (Phase 1 used existing comma syntax) + ~150 LOC (Phase 2 `|` sugar — actual: 60+90 LOC)
**Dependencies**: M-EXT-PORTABILITY-GATE (v0.19.0) — surfaced during round-2 follow-up F1
**Author**: Claude Opus 4.7 + Mark
**Created**: 2026-05-11
**Source**: M-EXT-PORTABILITY-GATE round-2 evaluation residual ([report](../../../.ailang/state/evaluations/eval_M-EXT-PORTABILITY-GATE_round_2.json))

---

## Problem statement (as originally framed)

The round-2 evaluation flagged that `std/smoke.dispatchAllTools` was forced to declare a wide effect union:

```ailang
dispatch: (string) -> () ! {IO, Process, FS, AI, Env, Net, SharedMem, Clock, Stream}
) -> () ! {IO, Process, FS, AI, Env, Net, SharedMem, Clock, Stream}
```

Callers of `dispatchAllTools` would need `--caps Net,AI,SharedMem,...` even when their dispatcher only used Process+FS. This breaks the harness promise: callbacks should declare only their actual effects, and the consumer's caps requirement should be the union.

The hypothesis was that AILANG's surface parser couldn't accept row-polymorphic effect syntax in callback positions — `! e` and `! {IO | e}` both got `PAR015: bare assignment not supported`.

## Discovery during investigation

**Comma-separated row-var syntax already works:**

```ailang
-- ALL of these parse cleanly today (v0.18.10+):
func bareRow(f: (int) -> int ! {e}) -> int ! {e} = f(1)             -- ✓
func mixedComma(f: (int) -> int ! {e}) -> int ! {IO, e} = f(1)      -- ✓
func multiLabel(f: (int) -> int ! {e}) -> int ! {IO, FS, e} = f(1)  -- ✓
```

The parser at `internal/parser/parser_effect.go:65-74` already detects lowercase identifiers as row variables (`isRowVar = effectName[0] >= 'a' && effectName[0] <= 'z'`) and stores them in `EffectAnnotation{IsRowVar: true}`. The type system at `internal/types/effects.go:241` consumes them correctly: it iterates the effect slice and treats any `IsRowVar` entry as the row tail.

**The actual gap is narrower:** only the `|` separator sugar is missing. The `! e` form (no braces) is also unsupported, but that's pure syntactic sugar over `! {e}`.

**What got rejected and why:**

| Syntax | Status | Reason |
|---|---|---|
| `! {e}` | ✓ works | Existing parser handles lowercase IDENT in `{ }` as row var |
| `! {IO, e}` | ✓ works | Same path; row var is just one entry in the slice |
| `! {IO, FS, e}` | ✓ works | Same |
| `! {IO \| e}` | ✗ rejected | `\|` token not in parseEffectAnnotation's grammar |
| `! e` | ✗ rejected | Parser requires `{ }` after `!` |

The two failing forms are **syntactic sugar only**. They're idiomatic in Koka/Effekt-style effect-handler languages and would be nice to have, but the underlying capability is already shipped.

## Goals

**Immediate (v0.19.0)**:

1. **Fix `std/smoke.ail` to use comma syntax** — narrows callback signature from the 9-effect union to `! {e}`, restores the harness promise that callers pay only for their dispatcher's actual effects.
2. **Document the comma syntax** in stdlib reference + CHANGELOG so future stdlib helpers know the pattern.

**Future (v0.19.2 or later)**:

3. **Add `|` separator sugar** for row extension: `! {IO | e}` and `! {IO, FS | e}`.
4. **Add bare row-var sugar**: `! e` desugars to `! {e}`.

The future items are pure ergonomics — Koka-style readability. Not blocking.

## Non-goals

- Changing the type system's row-poly handling (already works correctly).
- Adding new effect-row capabilities (intersection, exclusion, etc.) — the existing model is sufficient.

## Conflict surface (for the future `|` sugar work)

Touches:
- `internal/parser/parser_effect.go::parseEffectAnnotation` — extend the loop to also accept `PIPE` (`|`) as a separator before a final row variable.
- Tokenizer: `|` is already a token (used for record types `{a: T | rest}`). No lexer change needed.
- AST: no change — the row-var entry in `EffectAnnotation[]` is the same shape regardless of `,` vs `|` separator.
- Type system: no change — already iterates the slice ignoring separator semantics.

**Programs that MUST still work** post-`|` change (regression fixtures, when that work happens):

1. All current concrete-effect annotations: `! {IO}`, `! {IO, FS, Clock}` — comma is still the canonical separator for concrete effects.
2. Row-var via comma: `! {IO, e}` continues to work.
3. Existing `|` usage in record types: `{name: string | rest}` — disambiguated by context (`!` precedes effect rows, `{` without `!` precedes record types).

Disambiguation: `!` precedes the row, so the parser knows whether `|` is "record extension" (preceded by `{` without `!`) or "effect extension" (preceded by `! {`). Two-token lookahead is sufficient.

## Solution

### Phase 1 (immediate, v0.19.0) — `std/smoke.ail` fix

Replace the wide-union signatures with the comma-syntax row-var form:

```diff
 export func dispatchAllTools(
   tools: [string],
-  dispatch: (string) -> () ! {IO, Process, FS, AI, Env, Net, SharedMem, Clock, Stream}
-) -> () ! {IO, Process, FS, AI, Env, Net, SharedMem, Clock, Stream} =
+  dispatch: (string) -> () ! {e}
+) -> () ! {IO, e} =
   ...
```

Same change for `dispatchTool`. `okSuite` remains `! {IO}` (no callback). Verified end-to-end: caller now needs only `--caps IO` instead of `--caps Net,AI,SharedMem,...`.

### Phase 2 (future, v0.19.2+) — `|` separator sugar

Extend `parseEffectAnnotation` to accept `|` before a final row variable:

```ailang
-- Sugar:
! {IO | e}        -- desugars to: ! {IO, e}
! {IO, FS | e}    -- desugars to: ! {IO, FS, e}
! e               -- desugars to: ! {e}
```

Implementation sketch:

```go
// In parseEffectAnnotation, after parsing each concrete effect:
if p.peekTokenIs(lexer.PIPE) {
    p.nextToken()  // consume |
    if !p.expectPeek(lexer.IDENT) || !isLowercaseStart(p.curToken.Literal) {
        p.report("PAR_EFF015_ROW_VAR_AFTER_PIPE", ...)
    }
    rowVarName := p.curToken.Literal
    effects = append(effects, ast.EffectAnnotation{
        Name: rowVarName, IsRowVar: true, ...
    })
    if !p.expectPeek(lexer.RBRACE) { ... }
    return effects
}
```

~30 LOC + ~30 LOC tests (4 cases: `! {IO | e}`, `! {IO, FS | e}`, `! e`, error case `! {IO | IO2}`).

## Acceptance

**Phase 1 (v0.19.0)**:
- [x] `std/smoke.dispatchAllTools` rewritten with `! {IO, e}` row-var signature
- [x] `std/smoke.dispatchTool` same
- [x] Caller test verifies `--caps IO` alone is sufficient (was previously requiring `--caps IO,Process,FS,AI,Env,Net,SharedMem,Clock,Stream`)
- [x] Caller test with `--caps IO,FS` dispatcher works correctly (row-poly propagates)
- [x] CHANGELOG note: comma-syntax row-var is the canonical pattern; `|` sugar planned for later

**Phase 2 (shipped this session)**:
- [x] Parser accepts `! {IO | e}` and `! {IO, FS | e}`
- [ ] Parser accepts bare `! e` (no braces) — DEFERRED, comma + pipe forms cover the use case
- [x] All existing parser tests pass unchanged
- [x] `TestEffectRowExtensionPipeSugar` covers 5 cases (single label + pipe, multi label + pipe, comma regression, typo-after-pipe, case-insensitive-typo-after-pipe)
- [x] `std/smoke.ail` updated to use the canonical `! {IO | e}` form
- [x] CHANGELOG entry folded into [v0.19.0]

## Why the original premise was wrong

The PAR015 error message ("bare assignment not supported, missing 'let' keyword") was misleading. The parser was failing at the `|` token AFTER successfully parsing `! {IO`, then trying to parse `| e}` as an expression — at which point `|` looked like a binary operator missing its left operand, leading to the `let`-suggestion error.

A more helpful error would be: "row-extension syntax `! {Label | rowVar}` not yet supported; use `! {Label, rowVar}` instead." That's a v0.19.2 follow-up bundled with the `|` parser work.

## Files affected (Phase 1 only)

| File | Change | LOC |
|---|---|---|
| `std/smoke.ail` | replace wide-union signatures with `! {e}` / `! {IO, e}` | ~6 lines changed |
| `changelogs/v0.10-current.md` | note comma-syntax row-var pattern, downgrade `|` sugar to v0.19.2 | ~25 added |
| `design_docs/planned/v0_19_0/m-parser-row-poly-effects.md` | THIS doc | new |
| **Total Phase 1** | | **~30 LOC** |

## Why this matters for AI-author workflows

Stdlib helpers that take callbacks are about to multiply (`forEach`, `runWithRetry`, `withResource`, etc.). Without row polymorphism in the surface syntax, every helper would force callers to over-grant capabilities. The fix preserves the principle: capability requirements at the CLI mirror the actual effects of the program, not stdlib's defensive wide-union.

For LLM agents writing AILANG: an over-broad `--caps` requirement is a confusing signal — "why does my Process+FS code need Net?" The narrow signature makes the failure mode obvious and the right answer cheap.

## Refs

- M-EXT-PORTABILITY-GATE round-2 evaluation: `.ailang/state/evaluations/eval_M-EXT-PORTABILITY-GATE_round_2.json` (F1 residual)
- Existing row-var parser: `internal/parser/parser_effect.go:65-74`
- Existing row-var consumer: `internal/types/effects.go:241`
- Koka effect rows: https://koka-lang.github.io/koka/doc/book.html#sec-effect-rows
