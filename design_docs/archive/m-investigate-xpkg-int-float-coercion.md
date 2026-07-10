# Bug: `show()` builtin fails in serve-api when transitive deps also use `show()`

**Status**: Fixed — DX improvements + `Call` fix applied; deeper `show()` issue could not be reproduced
**Priority**: P2 (DX improvement; original serve-api issue may have been transient)
**Source**: docparse agent message (msg_20260402_105440_0b1e8e81)
**Date**: 2026-04-02

## Summary

The `show()` builtin returns raw `IntValue` instead of `StringValue` when:
1. Running in `serve-api` (not `ailang run`)
2. The handler imports a package that **transitively** imports a module also using `show()`
3. Specifically: importing `billing_store/usage_repo` which imports `firestore/fields`
   which calls `show(value)` in its `intVal()` function

The result: `concat_String: arg 1 - expected string, got int` — `show(x)` passes through
the raw int instead of converting to string.

## Minimal Reproduction

**Works** (no transitive `show` usage):
```ailang
import pkg/sunholo/billing_store/entitlements_repo (getEntitlements)
-- entitlements_repo → firestore/fields → intVal uses show()
-- BUT entitlements_repo only imports intVal, not show directly
-- This works fine
```

**Fails** (transitive dep also uses `show`):
```ailang
import pkg/sunholo/billing_store/usage_repo (getUsage)
-- usage_repo → firestore/fields → intVal → show(value)
-- Adding this import breaks show() in the calling module
```

**Exact test setup:**
```bash
# In /tmp/serve_test/ with proper ailang.toml + ailang.lock
# handler.ail imports billing_store/usage_repo
ailang serve-api --caps Net,FS,Env --port 8099 .
curl "http://localhost:8099/test?args=test_user&args=2026_04"
# → concat_String: arg 1 - expected string, got int
```

## Investigation Timeline

### Initial Theory: Cross-package int→float coercion (WRONG)
- Docparse reported `got *eval.FloatValue` — suggested JSON numbers losing int type
- `getInt()` → `floatToInt()` chain works correctly
- `makeJNumber()` always creates `FloatValue` but `floatToInt` converts properly

### Second Theory: `CallPreserveFloats` in serve-api (PARTIAL)
- serve-api used `CallPreserveFloats()` which keeps JSON float64 as `FloatValue`
- Changed to `Call()` — fixed the float issue
- But revealed the REAL error: `got int` (not `got float`)
- This means `show()` is returning the raw int, not a string

### Root Cause: `show()` polymorphic builtin resolution across modules

The `show` builtin is registered as:
- **Name**: `show` in `$builtin` module
- **Type**: `∀α. α -> string` (polymorphic)
- **Implementation**: `showImpl()` — uses type switch on `eval.Value`

Additionally, `types/instances.go` declares typeclass instances:
```go
"show": "builtin_show_int"   // for Show Int
"show": "builtin_show_float" // for Show Float
```

But `builtin_show_int`, `builtin_show_float`, etc. are **never registered as actual builtins**.
They exist only as dictionary keys in the typeclass system.

**Hypothesis**: When multiple modules across package boundaries use `show()`,
the compiler/elaborator creates conflicting monomorphization or dictionary entries.
The `show` call in the importing module resolves to an identity function or a no-op
instead of the actual `showImpl` builtin.

### Why it works in `ailang run` but not `serve-api`

`ailang run` compiles and evaluates a single entry point module. All transitive
dependencies are compiled in a single pass with a shared compilation context.

`serve-api` uses `embed.Engine` which may compile modules incrementally:
- Preloads package modules during startup
- May create separate compilation contexts per module
- `show` resolution might get cached/overridden between contexts

## What We Know For Sure

| Scenario | Works? |
|----------|--------|
| `ailang run` with local modules | ✅ |
| `ailang run` with `pkg/` imports | ✅ |
| `serve-api` with no pkg/ deps | ✅ |
| `serve-api` + `pkg/billing_entitlements` only | ✅ |
| `serve-api` + `pkg/billing_entitlements` + `pkg/firestore/fields` | ✅ |
| `serve-api` + `pkg/billing_store/entitlements_repo` | ✅ |
| `serve-api` + `pkg/billing_store/usage_repo` | ❌ |
| `serve-api` + `pkg/billing_store/usage_repo` + `pkg/billing_store/entitlements_repo` | ❌ |
| Full billing_service_api@0.5.3 | ❌ |

**The trigger**: `usage_repo` imports `usage_policy` AND `firestore/fields`.
`firestore/fields.intVal()` calls `show(value)` where `value: int`.
When this module is loaded as a transitive dependency, it corrupts `show`
resolution for the top-level handler module.

## Fixes Applied

### 1. Error messages (done)
`safe_cast.go`: All `SafeAs*` functions now show AILANG type names with conversion hints.
```
Before: concat_String: arg 1 - expected string, got *eval.FloatValue
After:  concat_String: arg 1 - expected string, got float. Use floatToStr() to convert
```

### 2. Switched serve-api from `CallPreserveFloats` to `Call` (done)
`apiserver/routes.go`: HTTP route handlers now use `Call()` so JSON-decoded whole numbers
are converted to `IntValue` instead of staying as `FloatValue`.

### 3. Debug tracing infrastructure (done)
Added `DEBUG_EVAL_APP=1` environment variable that traces:
- All `VarGlobal` resolutions (module.name → type)
- All function applications (function type, argument types/values)
- All function return values (type and value)

This is permanently available behind the env var for future debugging.

### 4. Root cause investigation (could not reproduce)
Added `DEBUG_EVAL_APP=1` tracing to `evalCoreApp` and `evalCoreVarGlobal`, then ran
the full diamond-dependency test case (`billing_store/usage_repo` + `billing_entitlements`
+ `firestore/fields`) through serve-api.

**Result**: `intToStr()` correctly resolved to `_string_intToStr` builtin, received
`IntValue`, and returned `StringValue` in all cases. The bug may have been:
- Fixed by the `CallPreserveFloats → Call` change (fix #2)
- Specific to a production deployment with a different ailang version
- A transient compilation cache issue

If the bug recurs, run with `DEBUG_EVAL_APP=1` to capture the full resolution trace.

## Files Changed

- `internal/builtins/safe_cast.go` — User-friendly type names + conversion hints
- `internal/builtins/string_convert.go` — intToStr/floatToStr error messages
- `internal/builtins/numeric.go` — intToFloat/floatToInt error messages
- `internal/apiserver/routes.go` — `Call()` instead of `CallPreserveFloats()`
- `internal/eval/eval_operations.go` — `DEBUG_EVAL_APP=1` tracing for function application
- `internal/eval/eval_expressions.go` — `DEBUG_EVAL_APP=1` tracing for VarGlobal resolution

## Regression Tests

- `internal/builtins/safe_cast_test.go` — `TestErrorMessagesShowAILANGTypes`, `TestConversionHintsInErrors`
- `internal/embed/embed_test.go` — `TestFromGoWholeFloatBecomesInt`, `TestFromGoPreserveFloatsKeepsFloat`

## Test Files

- `/tmp/serve_test/` — Minimal reproduction setup with diamond deps

## Related

- [m-string-conversion.md](../implemented/v0_5_10/m-string-conversion.md) — intToStr/floatToStr
- [m-dx-json-bool-coercion.md](m-dx-json-bool-coercion.md) — Similar JSON type issue
- M-DX-XPKG-RESOLVE (v0.9.11) — Prior cross-package fix (different root cause)

---

**Document created**: 2026-04-02
**Last updated**: 2026-04-02
