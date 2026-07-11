# M-DX-PACKAGE-DOGFOODING: DX Issues Found Building AILANG Packages in AILANG

**Status**: Planned
**Target**: v1.0.0
**Priority**: P1 — these are real friction points AI agents will hit
**Estimated**: 2-3 days
**Source**: Discovered while building `sunholo/registry_validator` in AILANG

---

## Issues Found

### Issue 1: Hyphens in Module Paths Cause Silent Parse Failure

**Severity**: High — no error message, just cascading parse errors on subsequent imports

```ailang
module sunholo/registry-validator/validate  -- FAILS: parser sees "registry" - "validator"
```

The parser interprets `-` as the minus operator. Module path `a-b` becomes `a - b` (subtraction). Every subsequent import then fails with `unexpected token: import`.

**Fix**: Either:
- (a) Support hyphens in module path segments (lexer change: allow `-` in IDENT when inside `module`/`import` context)
- (b) Emit a clear error: `"Hyphens not allowed in module paths. Use underscores: registry_validator"`

Option (b) is simpler and prevents the cascading silent failures. The error should trigger when the parser sees `IDENT MINUS IDENT` after `module` or `import`.

### Issue 2: `++` Ambiguity Between String and List Concatenation

**Severity**: Medium — confusing type error, AI agents will use `++` for lists

```ailang
let combined = list1 ++ list2  -- ERROR: cannot unify list type with string
```

`++` is defined as string concatenation. For lists, you must use `concat(list1, list2)` from `std/list`. But every AI agent (and human) will try `++` for lists first.

**Options**:
- (a) Make `++` polymorphic — works on both strings and lists (like Haskell)
- (b) Better error message: `"++ is for strings. For lists, use concat(xs, ys) from std/list"`
- (c) Add `++` as sugar that desugars to `concat` for list types

Option (a) or (c) is the best DX. This came up in docparse code too.

### Issue 3: `jnum` Requires `float`, Not `int`

**Severity**: Low — easy workaround but surprising

```ailang
kv("count", jnum(42))                    -- ERROR: expected float, got int
kv("count", jnum(intToFloat(42)))        -- works but verbose
```

The `JNumber(float)` constructor in `std/json` only accepts float. Since JSON numbers can be integers, this is a frequent friction point.

**Fix**: Add `jint(int) -> Json` convenience constructor to `std/json`:
```ailang
export pure func jint(n: int) -> Json = JNumber(intToFloat(n))
```

This is a 1-line stdlib addition.

---

## Implementation Plan

### M1: Better Error for Hyphens in Module Paths (~50 LOC)

**File**: `internal/parser/parser_file.go`

In `parseImportDecl()` and module declaration parsing, after parsing a path segment, check if the next token is MINUS. If so, emit:

```
Error: Hyphens (-) not allowed in module paths
  module sunholo/registry-validator/validate
                        ^ use underscore: registry_validator
```

### M2: `jint` Convenience Function (~5 LOC)

**File**: `std/json.ail`

```ailang
export pure func jint(n: int) -> Json = JNumber(intToFloat(n))
```

Update teaching prompt and devtools prompt to mention `jint`.

### M3: `++` for Lists (Design Decision Needed)

**Options**:
- Make `++` polymorphic via type class (larger change, needs `Concat` class)
- Add `++` desugaring for list types in the operator lowering pass
- Just improve the error message

**Recommendation**: Improve error message for v1.0, add polymorphic `++` in v1.1.

---

## Related

- [m-pkg-package-system.md](m-pkg-package-system.md) — Package system where these issues surfaced
- `sunholo/registry_validator` in ailang-packages — the dogfooding package that exposed these

---

**Document created**: 2026-03-20
