# M-EXTERNAL-CONSUMER-DX Sprint Plan

**Sprint ID**: M-EXTERNAL-CONSUMER-DX  
**Design doc**: [design_docs/planned/v0_17_0/m-external-consumer-dx.md](design_docs/planned/v0_17_0/m-external-consumer-dx.md)  
**Target**: v0.17.0  
**Estimated**: 3.5 days (~680 LOC)  
**Risk level**: Medium (M2 touches inference path)  

---

## Goal

Ship three DX improvements sourced directly from motoko_agent's `.agent/learnings/` — the first production-scale external AILANG consumer. Every item has a concrete reproduction scenario.

---

## Milestone Summary

| # | ID | Description | LOC | Days | Risk |
|---|---|---|---|---|---|
| M1 | MOD013_PREFIX_OVERLAP | module_prefix overlap diagnostic | ~250 | 1.5 | Low |
| M2 | EFFECT_ROW_PROVENANCE | Effect-row mismatch with call-site pointer | ~280 | 2.0 | Medium |
| M3 | ERROR_CODES_ARTIFACT | error_codes.json release artifact | ~150 | 1.0 | Low |

Note: Design doc says "MOD012" but that code is already taken in `internal/errors/codes.go` (implicit module). Correct code is **MOD013**.

---

## M1 — MOD013 module_prefix overlap diagnostic (Day 1–1.5)

**Problem**: When root project and a dep package share the same `module_prefix`, imports are silently misrouted. The compiler resolves the wrong package boundary and produces a confusing resolution failure instead of naming the ambiguity.

**Reproduction** (motoko scenario):
```toml
# ailang.toml (root)
[package]
name = "local/motoko_agent"
module_prefix = "src"

[dependencies]
"sunholo/motoko_core" = ">=0.1.0"

# .packages/motoko_core/ailang.toml
[package]
name = "sunholo/motoko_core"
module_prefix = "src"
```
Result today: `module "sunholo/motoko_core/tool_contract" is not exported` (opaque). Expected: `Error MOD013: ambiguous module ownership under shared module_prefix`.

**Files**:
| File | Change | Est LOC |
|------|--------|---------|
| `internal/errors/codes.go` | Add `MOD013` constant + doc comment | +5 |
| `internal/pipeline/package_resolver.go` | Export root package name (`currentRootPkgName`) alongside prefix map | +10 |
| `internal/pipeline/pipeline_module.go` | Add `detectModulePrefixOverlap(currentModulePrefixMap, currentRootPkgName)` called after MOD011 at line 127 (~80 LOC new func) | +80 |
| `internal/pipeline/module_collision_test.go` | `TestDetectModulePrefixOverlap_*` (3 cases: overlap fires, different prefixes clean, root-only-prefix clean) | +100 |
| `docs/docs/reference/errors/mod013.md` | New doc page with 3 fix options | +55 |

**Algorithm**:
```go
func detectModulePrefixOverlap(prefixMap map[string]string, rootPkg string) error {
    // Group packages by module_prefix value
    byPrefix := map[string][]string{}
    for pkg, prefix := range prefixMap {
        if prefix != "" {
            byPrefix[prefix] = append(byPrefix[prefix], pkg)
        }
    }
    for prefix, pkgs := range byPrefix {
        if len(pkgs) < 2 { continue }
        // Only fire if root is one of the claimants
        rootInGroup := false
        for _, p := range pkgs { if p == rootPkg { rootInGroup = true } }
        if !rootInGroup { continue }
        sort.Strings(pkgs)
        return fmt.Errorf("Error MOD013: ambiguous module ownership ... prefix=%q claimants=%v", prefix, pkgs)
    }
    return nil
}
```

**Acceptance criteria**:
- [ ] `TestDetectModulePrefixOverlap_SamePrefixRootAndDep` fails with `Error MOD013:` and names both packages
- [ ] `TestDetectModulePrefixOverlap_DifferentPrefixes` passes clean
- [ ] `TestDetectModulePrefixOverlap_RootOnlyPrefix` passes clean (root has prefix, no dep shares it)
- [ ] Error message includes all 3 fix options from design doc
- [ ] `make test && make lint` clean
- [ ] `docs/docs/reference/errors/mod013.md` exists

---

## M2 — Effect-row mismatch with call-site provenance (Day 2–3)

**Problem**: `newEffectRowError` in `internal/types/errors.go` names the inferred row vs expected row but not which call introduced/missed a label. User must trace the entire function body manually.

**Reproduction** (motoko `register()` scenario):
```ailang
func register() -> ExtensionHooks ! {Env} =
  ExtensionHooks {
    on_budget_plan = \(ctx, plan) ->
      -- this lambda only reads cfg from Env; no FS call
      BudgetPatch {}
    -- slot type: (ExtCtx, BudgetPlan) -> BudgetPatch ! {Env, FS}
  }
```
Today: `missing required effects: {FS}` — doesn't say why FS is missing or that the slot requires it.  
After: `missing required effects: {FS} — slot on_budget_plan at dummy.ail:42 expects ! {Env, FS}; lambda body at dummy.ail:47 has no FS-effectful call`.

**Files**:
| File | Change | Est LOC |
|------|--------|---------|
| `internal/types/effects.go` | Add `Provenance map[string]ast.SourceSpan` pointer field to `Row`; ignore in `Equals()`/hash | +15 |
| `internal/types/effects.go` | `WithProvenance(label, span)` helper to attach provenance without mutating shared rows | +20 |
| `internal/types/typechecker.go` | Populate provenance when effectful builtin call injects a label; thread through union ops | +60 |
| `internal/types/errors.go` | Update `newEffectRowError` to consume provenance; emit "introduced at X:Y via call to Z" for extra labels; "expected by slot at X:Y" for missing | +55 |
| `internal/types/effect_provenance_test.go` | New test file: reproduce `register()`/`on_budget_plan` mismatch; verify error message names slot and lambda location | +100 |
| `docs/docs/reference/errors/typ_effect_row_mismatch.md` | Update with new message format and both mismatch cases | +30 |

**Provenance strategy**: `Row.Provenance` is a pointer (`*map[string]ast.SourceSpan`), nil when not set. `Row.Equals()` ignores it. When we do `UnionEffectRows(a, b)` the result carries merged provenance (a's for labels from a, b's for labels from b). On unification failure, `newEffectRowError` reads provenance from the actual row.

**Acceptance criteria**:
- [ ] `TestEffectRowMismatch_MissingLabel_NamesSlot` — error message contains slot location for missing labels
- [ ] `TestEffectRowMismatch_ExtraLabel_NamesCallSite` — error message contains call site for extra labels
- [ ] `make test -count=20 ./internal/types/...` clean (determinism under map iteration)
- [ ] `make bench ./internal/types/...` within 5% of baseline (provenance is nil-checked fast path)
- [ ] `make test && make lint` clean

---

## M3 — `error_codes.json` release artifact (Day 3–3.5)

**Goal**: Machine-readable `{code, category, summary, fix_hint, doc_url}` rows for every emitted error code, published as a release asset and hosted at `ailang.sunholo.com/error_codes/v<ver>.json`.

**Files**:
| File | Change | Est LOC |
|------|--------|---------|
| `tools/gen-error-codes/main.go` | Go tool: parse `internal/errors/codes.go` via `go/parser`, extract const decls + doc comments, emit JSON schema v1 | +90 |
| `Makefile` | `make error-codes` target: `go run tools/gen-error-codes/main.go > dist/error_codes.json` | +8 |
| `.github/workflows/release.yml` | Upload `dist/error_codes.json` as release asset alongside binaries | +12 |
| `docs/docs/reference/errors/index.md` | Add `error_codes.json` consumption section | +30 |
| `tools/gen-error-codes/main_test.go` | Verify every code in `codes.go` appears in generated JSON; schema validates | +40 |

**JSON schema** (from design doc):
```json
{
  "schema": "ailang.error_codes/v1",
  "ailang_version": "v0.17.0",
  "generated_at": "...",
  "codes": [
    {
      "code": "MOD010",
      "category": "module",
      "summary": "module declaration does not match file path",
      "fix_hint": "set `module <relative_path>` so it mirrors the file system path from project root",
      "doc_url": "https://ailang.sunholo.com/docs/reference/errors/mod010"
    }
  ]
}
```

**Fix hints**: extracted from `// <Code> indicates <description>` doc comments in `codes.go`. Generator maps category from code prefix (PAR→parser, MOD→module, LDR→loader, TYP→type, etc.).

**Acceptance criteria**:
- [ ] `make error-codes` produces valid JSON at `dist/error_codes.json`
- [ ] `TestGenErrorCodes_AllCodesPresent` passes (every constant in `codes.go` has a row)
- [ ] `TestGenErrorCodes_SchemaValid` passes (required fields non-empty for all rows)
- [ ] Release workflow step uploads `error_codes.json` as release asset (verify step exists in YAML)
- [ ] `docs/docs/reference/errors/index.md` has consumption section linking to the artifact
- [ ] `make test && make lint` clean

---

## Day-by-Day Plan

| Day | Work |
|-----|------|
| 1 | M1: Add MOD013 to codes.go; implement `detectModulePrefixOverlap`; write 3 test cases |
| 1.5 | M1: doc page; `make test && make lint`; mark M1 complete |
| 2 | M2: Add `Provenance` to `Row`; `WithProvenance` helper; thread through `UnionEffectRows` |
| 2.5 | M2: Populate provenance in typechecker for effectful calls; update `newEffectRowError` |
| 3 | M2: Write `effect_provenance_test.go`; `make test -count=20`; perf check; doc update |
| 3 | M3: Write `tools/gen-error-codes/main.go`; `make error-codes` target |
| 3.5 | M3: CI gate; release workflow step; `docs/errors/index.md`; `make test && make lint` |

---

## Success Metrics

- `make test && make lint` clean throughout
- All 3 milestone acceptance criteria met
- No existing test regressions
- `make bench` within 5% on `internal/types/` (M2 provenance is nil-checked fast path)
- `dist/error_codes.json` covers all codes in `internal/errors/codes.go`
- motoko reproduction scenarios produce the new error messages (manually verified)

## Out of Scope

- `ailang.ebnf` stretch (defer to v0.17.x)
- `docs/docs/guides/external-consumers.md` new guide (defer — design doc calls it out but it's low-priority vs the three core items)
