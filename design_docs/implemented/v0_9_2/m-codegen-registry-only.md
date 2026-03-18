# M-CODEGEN-REGISTRY-ONLY: Delete Legacy Stdlib, Registry-Only Runtime

**Status**: Implemented
**Target**: v0.9.2
**Priority**: P0 (Critical — blocking DocParse codegen)
**Estimated**: 4h (actual: ~3h across 4 iterations)
**Dependencies**: M-CODEGEN-SUSTAINABILITY (registry system foundation)
**Milestone ID**: M-CODEGEN-REGISTRY-ONLY
**Created**: 2026-03-18
**Implemented**: 2026-03-18

---

## Problem Statement

The Go codegen runtime had **two overlapping systems** emitting helper functions into generated `runtime.go`:

1. **Legacy** (`codegen_runtime_stdlib.go`) — 72 functions via hardcoded `g.writef()` calls, with broken ADT guards that used unqualified keys (`"Some"` instead of `"Option.Some"`)
2. **Registry** (`registry_codegen.go` + `codegen_registry.go`) — 60 functions via `GoCodegenSpec` entries with lazy emission
3. **Bridge** (`markLegacyHelpersEmitted`) — a fragile 70-name dedup list preventing double-emission

This architecture caused a **3-cycle whack-a-mole** during DocParse codegen bring-up:
- **Cycle 1**: Fixed ADT guard keys → exposed 9 duplicate function declarations
- **Cycle 2**: Removed duplicates → regressed 4 previously-working functions
- **Cycle 3**: Fixed string literal assertions → more regressions from dedup list

Each fix to one system silently broke the other through the dedup bridge.

**Root Causes:**
- Legacy guards used unqualified keys (`"Some"`) against qualified map keys (`"Option.Some"`) — guards **never worked**, meaning legacy helpers were either all-or-nothing
- The dedup bridge was a static list that didn't track what was actually emitted, so it blocked registry emission for functions that legacy claimed but conditionally skipped
- No single source of truth — the same function could be defined in legacy code, registry specs, AND effect handlers

---

## Solution: Registry-Only Architecture

### Design Decisions

1. **Delete legacy entirely** — `codegen_runtime_stdlib.go` was removed, not refactored. The registry already had specs for 56 of 72 functions; only 6 needed to be added.

2. **ADT-guarded eager emission** — New `RequiresADT` field on `GoCodegenSpec` enables conditional emission. When a Json/Option/Result ADT is registered, ALL helpers for that ADT emit as a group. This solves transitive dependencies (e.g., `GetString` calls `JsonGet`, `IsNone`, `AsString`, `OptionGetOrElse`).

3. **Infrastructure vs stdlib separation** — `toSlice` (used by 24+ helpers) was moved to `codegen_runtime_collections.go` as infrastructure. It doesn't belong in either legacy or registry — it's a core utility like `CallFunc` and `toInt64`.

### Architecture After

```
writeRuntimeHelpers()
  ├── writeRuntimeRecordHelpers()      // FieldGet, RecordUpdate
  ├── writeRuntimeListHelpers()        // Cons, Head, Tail, toSlice
  ├── writeRuntimeArithmeticHelpers()  // AddInt, toInt64, etc.
  ├── writeRuntimeMiscHelpers()        // CallFunc, Show, Log
  ├── writeRuntimeSliceConverters()    // ConvertTo*Slice
  ├── writeArrayRuntimeFunctions()     // FromList, ToList, Get, Set
  ├── writeADTSliceConverters()        // ConvertTo<ADT>Slice (dynamic)
  ├── writeRecordSliceConverters()     // ConvertTo<Record>Slice (dynamic)
  ├── writeValueTypeConverters()       // As<Type> (dynamic)
  └── writeRegistryHelpers()           // ★ SOLE source for stdlib helpers
        ├── eagerRegisterADTHelpers()  // Force-register all Json/Option/Result helpers
        └── emit lazy + eager helpers  // Sorted, deduped by name
```

### Emission Modes

| Mode | Trigger | Example |
|------|---------|---------|
| **Lazy** | `resolveBuiltinViaRegistry()` encounters builtin during codegen | `Map`, `Filter`, `Split` |
| **Eager (ADT)** | ADT constructor registered for Json/Option/Result | `GetString`, `IsNone`, `Js` |
| **Always** | Infrastructure in `codegen_runtime_*.go` | `toSlice`, `CallFunc`, `toInt64` |

---

## Changes

### Files Deleted
- `internal/gen/golang/codegen_runtime_stdlib.go` — 1075 lines of legacy runtime emission

### Files Modified

| File | Change |
|------|--------|
| `internal/builtins/registry.go` | Added `RequiresADT` field to `GoCodegenSpec`, added `GetHelpersRequiringADT()` |
| `internal/builtins/registry_codegen.go` | Added 9 missing specs (Contains, Repeat, Foldr, ParseElements, FilterStrings, GetObject + RequiresADT tags on all Json/Option/Result specs) |
| `internal/gen/golang/codegen_registry.go` | Added `adtIsRegistered()`, `eagerRegisterADTHelpers()`, removed `emittedHelpers` bridge checks, added `fixLiteralTypeAssertions()` |
| `internal/gen/golang/codegen_runtime.go` | Removed `writeRuntimeStdlibHelpers()` and `markLegacyHelpersEmitted()` calls |
| `internal/gen/golang/codegen.go` | Removed `emittedHelpers` field |
| `internal/gen/golang/codegen_runtime_collections.go` | Absorbed `toSlice` from legacy |
| `internal/gen/golang/codegen_dictionaries.go` | Added `generateFractionalDictionary()` for `dict_Fractional_Float` |

### New Registry Specs Added

| Builtin | Type | Purpose |
|---------|------|---------|
| `_str_contains` | Inline | `strings.Contains` wrapper |
| `_str_repeat` | Helper | `strings.Repeat` wrapper |
| `_list_foldr` | Helper | Right fold |
| `_xml_parseElements` | Helper (panic) | XML streaming stub |
| `_json_filterStrings` | Helper | Extract JString values from list |
| `_json_getObject` | Helper | Look up key, extract JObject |

### RequiresADT Tags

| ADT | Tagged Specs |
|-----|-------------|
| `"Json"` | Js, Jn, Jb, Jnum, Ja, Jo, Kv, Decode, Encode, JsonGet, JsonHas, GetString, GetInt, GetBool, GetArray, AsString, AsNumber, AsBool, AsArray, AsObject, JsonKeys, JsonGetOr, JsonRepair, FilterStrings, GetObject |
| `"Option"` | OptionGetOrElse, IsNone, IsSome |
| `"Result"` | IsOk, IsErr |

---

## Verification

- `make build` — passes
- `make test` — passes (same pre-existing `TestContractViolation_Integration` failure)
- `make verify-examples` — 136/152 pass (16 failures are pre-existing effect-related examples)
- Basic example (no Json/Option): 56 runtime functions, no JSON/Option helpers emitted
- JSON example (with Json/Option): 89 runtime functions, all JSON/Option helpers present
- `dict_Fractional_Float` generated in all examples

---

## Future Work

1. **`GetNumber` spec** — `examples/runnable/json_parsing.ail` uses `getNumber` which has no codegen spec yet
2. **Unused imports** — Generated `runtime.go` always imports `sort` and `strconv` even when unused. Need conditional import tracking.
3. **`TestContractViolation_Integration`** — Pre-existing failure due to module-prefixed function names (`basic__Absolute` vs test expecting `Absolute`)
4. **StringToInt/StringToFloat** — These reference `NewOptionSome`/`NewOptionNone` (ADT constructors) but aren't tagged with `RequiresADT: "Option"`. They should either be tagged or use `makeOptionSome`/`makeOptionNone` fallbacks.
5. **Registry completeness audit** — Run DocParse compilation and grep for `undefined:` to find all remaining unregistered builtins
6. **Delete `emittedVars` cleanup** — The `emittedVars` field in `codegen.go` may also be vestigial

---

## Lessons Learned

1. **Dual systems with dedup bridges are inherently fragile** — The bridge was a static list disconnected from runtime behavior. Any conditional logic (guards) in either system invalidated the bridge's assumptions.
2. **Fix root cause before patching symptoms** — Three cycles of "fix one thing, break another" could have been avoided by recognizing the architectural problem on cycle 1.
3. **Qualified vs unqualified keys** — The original guard bug (`"Some"` vs `"Option.Some"`) was a silent failure that masked the entire legacy system being dead code. No test caught this because the system "worked" by accident (guards never fired, all helpers always emitted).
