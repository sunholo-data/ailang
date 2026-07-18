# M-TOOLING: Deterministic CLI for AI Agents (v0.3.15)

**Milestone**: M-TOOLING-DETERMINISTIC
**Version**: v0.3.15
**Status**: ⚠️ REALITY-CHECKED 2026-07-18 (mission iteration 48) — **PREMISE SUPERSEDED; scope-close decision PARKED for Mark** (queue-tagged "both in")
**Owner**: Tooling Team
**Created**: 2025-10-18
**Estimated Duration**: 3-4 days (24-32 hours)
**Dependencies**: M-LANG-JSON-DECODE (v0.3.14) ✅

---

## ⚠️ Reality check (mission iteration 48, 2026-07-18, HEAD `v0.29.2-362`)

This doc was authored **2025-10-18 (v0.3.15-era)**. Nine months of DX/eval work have moved the
ground out from under its premise. Live-repro at HEAD, **data-before-conclusions**:

**The doc's premise is obsolete.** It justifies the trio by: *"AIs generate single-shot fragments →
repaired by a slow, non-deterministic LLM (`prompts/repair_prompts/`)."*
- `prompts/repair_prompts/` **is deleted** — the LLM-repair path the doc argues against is gone.
- The eval flow is now **agentic** (`agent_mode:true`): multi-turn tool-use with **per-edit
  `ailang check` feedback** (the agentic-result gate requires `NumTurns>1 || ToolCallCount>0`,
  `agent_runner_multi.go`). Agents write whole modules and self-correct against the compiler — not
  one-shot fragments needing a normalize/apply pass.

**The doc's core capability already ships (differently packaged).** Goal 1 (deterministic
fragment normalization — wrap-in-module, add module decl, inject imports, fix bare calls,
synthesize main) **exists** as `normalizeProgram` in `internal/eval_harness/normalize.go`
(regex-based, no LLM; `RepairLog` tracks `Wrapped`/`AddedModule`/`AddedImports`/`CallFixes`/
`AddedMainFunc`). It is an **internal eval-harness function**, not the doc's standalone
`ailang normalize`/`suggest-imports`/`apply` CLI trio. Coverage in `normalize_test.go` incl. the
new guard `TestNormalizeProgram_MToolingMotivatingFragment` (the doc's exact json_parse fragment).

**Per-goal disposition at HEAD:**
| Doc goal | State at HEAD |
|---|---|
| 1. Normalize (wrap fragment → runnable module) | **SHIPPED** internally as `normalizeProgram` (deterministic). |
| 2. Suggest imports (missing symbols → minimal imports) | **PARTIAL / ABSORBED** — `normalizeProgram` auto-injects `std/io` only. General symbol→import resolution was never built; the need is now met by agentic `ailang check` feedback, **implicit prelude imports** (m-prelude-option-result, iter 27), and `ailang docs`/unknown-module did-you-mean (m-dx-ai-discovery, iter 30). |
| 3. Apply (JSON edits → code) | **NOT SHIPPED, and obsolete** — in agentic mode the agent edits files with its own tools; no JSON-edit-application command is needed. |
| 4. Byte-stable JSON output | N/A — no CLI trio to emit JSON; `normalizeProgram` is deterministic (guarded). |

Also eroding the fragment-repair need since Oct 2025: **MOD014** module-less-run-fail-loud (fragments
without `module` now fail loud with guidance) and **`--caps auto`** effect-row inference (auto_caps M1,
iter 32).

**What genuinely remains unbuilt:** the *public CLI packaging* of the trio (so an agent OUTSIDE the
eval harness could call `ailang normalize`), and the `apply` edit infra. Both solve the single-shot
model the architecture moved past.

**PARKED for Mark (scope-close decision).** The queue tags DX tooling "both in" (Mark's directive),
so the mission controller does **not** unilaterally rule this out. **Recommendation:** close as
**SUPERSEDED** (premise obsolete, capability shipped internally, remaining pieces solve an obsolete
problem) and, if any DX-tooling budget is spent, prefer `m-ailang-fmt` (a real, unshipped gap).
Alternatively, if a *non-agentic* external-agent integration is a real v1.0 audience, scope a
**much smaller** doc: expose the existing `normalizeProgram` as `ailang normalize` (Goal 1 only) —
NOT the full 3-4d trio. Awaiting Mark's call on #399.

Regression guard landing with this reality-check: `TestNormalizeProgram_MToolingMotivatingFragment`
(pins the shipped capability + its std/json boundary; deterministic-output assertion).

*(Everything below is the original 2025-10-18 design, preserved for the record.)*

---

## Executive Summary

Implement the deterministic tooling trio (`normalize`, `suggest imports`, `apply`) to enable AI agents to self-correct AILANG code fragments without LLM inference. This is **Phase 2** of the benchmark recovery + deterministic tooling initiative.

### Goals

1. **Primary**: Enable deterministic fragment normalization (wrap into runnable modules)
2. **Secondary**: Automate import resolution (missing symbols → minimal imports)
3. **Tertiary**: Provide edit application infrastructure (JSON → code transformations)
4. **Quality**: Ensure byte-stable JSON output (deterministic, schema-validated)

### Non-Goals (Deferred to v0.4.0+)

- LSP integration (requires daemon)
- Real-time diagnostics streaming
- Effect composer (auto-infer effect requirements)
- Test generator (from specifications)
- Import wildcard syntax: `import std/io (*)` (language feature, not tooling)
- FX001 diagnostic fix-it (requires diagnostic infrastructure enhancement)

---

## Problem Statement

### Current State

**AI Code Generation Challenges**:
1. **Fragment Generation**: AIs often generate code fragments, not complete modules
2. **Missing Imports**: AIs frequently omit `import` statements
3. **Effect Annotations**: AIs forget to add `! {IO}`, `! {FS}`, etc.
4. **Manual Repair**: Current eval harness uses LLM-based repair (slow, non-deterministic)

**Example Failure** (from json_parse benchmark):
```ailang
-- AI generates this fragment (no module, no imports, no effects)
func main() {
  let data = decode("[{\"name\":\"Alice\"}]")
  println(show(data))
}
```

**Problems**:
- ❌ Not a valid module (missing `module` declaration)
- ❌ Missing `import std/json (decode)`
- ❌ Missing `import std/io (println)`
- ❌ Missing effect annotation `! {IO}`
- ❌ Missing `export` on main

**Current Solution** (v0.3.13):
- Use LLM to repair code (prompts/repair_prompts/)
- Cost: ~$0.002-0.005 per repair
- Latency: 500ms-2s per repair
- Non-deterministic: Different repairs each run
- Unreliable: Sometimes LLM makes it worse

### Desired State (v0.3.15)

**Deterministic Repair Pipeline**:
```bash
# Step 1: Normalize (wrap fragment into module)
ailang normalize fragment.ail --output edits.json

# Step 2: Suggest imports (resolve missing symbols)
ailang suggest-imports fragment.ail --output imports.json

# Step 3: Apply edits (combine all fixes)
ailang apply fragment.ail edits.json imports.json --output fixed.ail
```

**Benefits**:
- ✅ Deterministic: Same input → same output (reproducible)
- ✅ Fast: <100ms per operation (no LLM calls)
- ✅ Free: No API costs
- ✅ Accurate: Rule-based, no hallucination
- ✅ Composable: Chain tools via JSON

---

## Technical Design

### Architecture Overview

```
┌─────────────────────────────────────────────────────────┐
│ Input: Code Fragment (AILANG)                           │
│                                                         │
│  func main() {                                          │
│    let data = decode("[{\"name\":\"Alice\"}]")         │
│    println(show(data))                                  │
│  }                                                      │
└─────────────────────────────────────────────────────────┘
           │
           ▼
┌─────────────────────────────────────────────────────────┐
│ Command: ailang normalize fragment.ail                  │
│                                                         │
│ Output: edits.json                                      │
│  {                                                      │
│    "version": "1.0",                                    │
│    "edits": [                                           │
│      {"type": "prepend", "line": 1,                     │
│       "text": "module benchmark/solution\n\n"},        │
│      {"type": "modify", "line": 1,                      │
│       "old": "func main()",                             │
│       "new": "export func main() -> () ! {IO}"}        │
│    ]                                                    │
│  }                                                      │
└─────────────────────────────────────────────────────────┘
           │
           ▼
┌─────────────────────────────────────────────────────────┐
│ Command: ailang suggest-imports fragment.ail            │
│                                                         │
│ Output: imports.json                                    │
│  {                                                      │
│    "version": "1.0",                                    │
│    "imports": [                                         │
│      {"module": "std/json", "symbols": ["decode"]},    │
│      {"module": "std/io", "symbols": ["println"]},     │
│      {"module": "std/prelude", "symbols": ["show"]}    │
│    ],                                                   │
│    "insert_after_line": 1  // After module decl        │
│  }                                                      │
└─────────────────────────────────────────────────────────┘
           │
           ▼
┌─────────────────────────────────────────────────────────┐
│ Command: ailang apply fragment.ail edits.json \         │
│          imports.json --output fixed.ail                │
│                                                         │
│ Output: fixed.ail                                       │
│  module benchmark/solution                              │
│                                                         │
│  import std/json (decode)                               │
│  import std/io (println)                                │
│  import std/prelude (show)                              │
│                                                         │
│  export func main() -> () ! {IO} {                      │
│    let data = decode("[{\"name\":\"Alice\"}]")         │
│    println(show(data))                                  │
│  }                                                      │
└─────────────────────────────────────────────────────────┘
           │
           ▼
┌─────────────────────────────────────────────────────────┐
│ Verification: ailang run fixed.ail                      │
│                                                         │
│ ✅ Compiles                                             │
│ ✅ Runs                                                 │
│ ✅ Produces expected output                             │
└─────────────────────────────────────────────────────────┘
```

### Component Breakdown

#### 1. Normalize Command (cmd/ailang/normalize.go)

**Purpose**: Wrap code fragments into valid modules

**Input**: AILANG code (fragment or complete)

**Output**: JSON edit list (schema: schemas/normalize_edits_v1.json)

**Algorithm**:
```
1. Parse input code (allow partial AST)
2. Detect if module declaration exists
3. If not:
   a. Prepend: "module benchmark/solution\n\n"
   b. Set insert_line = 1
4. Find main function
5. If exists and not exported:
   a. Add "export" keyword
6. If exists and missing effect annotation:
   a. Infer effects from function body (IO, FS, Net)
   b. Add "! {Effect1, Effect2}" to signature
7. If exists and missing return type:
   a. Add "-> ()" (assuming unit return)
8. Return edit list as JSON
```

**Example Output** (schemas/normalize_edits_v1.json):
```json
{
  "$schema": "https://ailang.io/schemas/normalize_edits_v1.json",
  "version": "1.0",
  "source_file": "fragment.ail",
  "edits": [
    {
      "type": "prepend",
      "line": 1,
      "text": "module benchmark/solution\n\n"
    },
    {
      "type": "modify",
      "line": 3,
      "old": "func main()",
      "new": "export func main() -> () ! {IO}"
    }
  ],
  "metadata": {
    "inferred_effects": ["IO"],
    "added_module": true,
    "exported_main": true
  }
}
```

**Estimated**: 6 hours
- Code: ~200 LOC (parsing, inference, JSON generation)
- Tests: ~100 LOC (golden tests)
- Schema: ~50 LOC (JSON Schema)

#### 2. Suggest Imports Command (cmd/ailang/suggest_imports.go)

**Purpose**: Resolve missing symbols to minimal imports

**Input**: AILANG code (fragment or complete)

**Output**: JSON import list (schema: schemas/import_suggestions_v1.json)

**Algorithm**:
```
1. Parse input code (collect all identifiers)
2. Run type checker (record unresolved symbols)
3. For each unresolved symbol:
   a. Search stdlib for matching export
   b. If found in multiple modules, pick most specific
   c. Add to import list
4. Group imports by module
5. Determine insertion point (after module declaration)
6. Return import list as JSON
```

**Symbol Resolution Strategy**:
```
Unresolved: "decode"
1. Search: stdlib/std/json.ail → export func decode
2. Search: stdlib/std/prelude.ail → no export
3. Result: import std/json (decode)

Unresolved: "println"
1. Search: stdlib/std/io.ail → export func println
2. Result: import std/io (println)

Unresolved: "map"
1. Search: stdlib/std/list.ail → export func map
2. Search: stdlib/std/prelude.ail → no export (TODO)
3. Result: import std/list (map)
```

**Example Output** (schemas/import_suggestions_v1.json):
```json
{
  "$schema": "https://ailang.io/schemas/import_suggestions_v1.json",
  "version": "1.0",
  "source_file": "fragment.ail",
  "imports": [
    {
      "module": "std/json",
      "symbols": ["decode"],
      "reason": "unresolved: decode"
    },
    {
      "module": "std/io",
      "symbols": ["println"],
      "reason": "unresolved: println"
    },
    {
      "module": "std/prelude",
      "symbols": ["show"],
      "reason": "unresolved: show"
    }
  ],
  "insert_after_line": 1,
  "metadata": {
    "total_unresolved": 3,
    "total_resolved": 3,
    "total_unresolvable": 0
  }
}
```

**Estimated**: 6 hours
- Code: ~250 LOC (parsing, symbol resolution, stdlib search)
- Tests: ~120 LOC (golden tests)
- Schema: ~60 LOC (JSON Schema)

#### 3. Apply Command (cmd/ailang/apply.go)

**Purpose**: Apply JSON edits to source file

**Input**:
- Source file (AILANG)
- Edit files (JSON) - 1 or more

**Output**: Modified source code (AILANG)

**Algorithm**:
```
1. Load source file into memory
2. Load all edit JSON files
3. Validate schemas (fail if invalid)
4. Merge edit lists (dedup, sort by line number)
5. Apply edits in reverse order (bottom-up):
   a. Type "prepend": Insert text before line
   b. Type "append": Insert text after line
   c. Type "modify": Replace exact match
   d. Type "delete": Remove line
6. Validate resulting code (parse check)
7. Output modified code (to stdout or --output file)
```

**Edit Application Order**:
```
Original (3 lines):
  1: func main() {
  2:   println("hello")
  3: }

Edits (merged):
  [
    {"type": "prepend", "line": 1, "text": "module test\n\n"},
    {"type": "modify", "line": 1, "old": "func main()", "new": "export func main() -> () ! {IO}"}
  ]

Apply (reverse order to preserve line numbers):
  Step 1: Modify line 1
    1: export func main() -> () ! {IO} {
    2:   println("hello")
    3: }

  Step 2: Prepend line 1
    1: module test
    2:
    3: export func main() -> () ! {IO} {
    4:   println("hello")
    5: }

Output: 5 lines (was 3)
```

**Estimated**: 4 hours
- Code: ~150 LOC (edit merging, application, validation)
- Tests: ~80 LOC (golden tests)

#### 4. JSON Schemas (schemas/)

**Files**:
- `schemas/normalize_edits_v1.json` - Edit list format
- `schemas/import_suggestions_v1.json` - Import list format
- `schemas/apply_result_v1.json` - Application summary

**Validation**:
- Use https://github.com/xeipuuv/gojsonschema for Go validation
- Fail fast on schema violations
- Provide actionable error messages

**Estimated**: 4 hours
- Schemas: ~150 LOC (JSON Schema definitions)
- Validation: ~50 LOC (Go integration)
- Tests: ~50 LOC (schema violation tests)

#### 5. Golden Test Infrastructure

**Purpose**: Ensure determinism and prevent regressions

**Structure**:
```
testdata/
  tooling/
    normalize/
      fragment_no_module.ail           (input)
      fragment_no_module.golden.json   (expected output)

      fragment_no_export.ail
      fragment_no_export.golden.json

      fragment_no_effects.ail
      fragment_no_effects.golden.json

    suggest_imports/
      missing_decode.ail
      missing_decode.golden.json

      missing_multiple.ail
      missing_multiple.golden.json

    apply/
      fragment_with_edits.ail          (input source)
      fragment_with_edits.edits.json   (edits)
      fragment_with_edits.golden.ail   (expected output)
```

**Test Pattern** (cmd/ailang/normalize_test.go):
```go
func TestNormalize_Golden(t *testing.T) {
    cases := []struct {
        name       string
        inputFile  string
        goldenFile string
    }{
        {"fragment_no_module", "fragment_no_module.ail", "fragment_no_module.golden.json"},
        {"fragment_no_export", "fragment_no_export.ail", "fragment_no_export.golden.json"},
        // ...
    }

    for _, tc := range cases {
        t.Run(tc.name, func(t *testing.T) {
            input := readFile(t, "testdata/tooling/normalize/"+tc.inputFile)
            expected := readFile(t, "testdata/tooling/normalize/"+tc.goldenFile)

            result := runNormalize(input)
            resultJSON := toJSON(result)

            // Byte-stable comparison
            assert.JSONEq(t, expected, resultJSON)

            // Update golden file if --update flag
            if *update {
                writeFile(t, "testdata/tooling/normalize/"+tc.goldenFile, resultJSON)
            }
        })
    }
}
```

**Estimated**: 2 hours
- Infrastructure: ~80 LOC (test helpers)
- Golden files: ~300 LOC (10 test cases × 3 commands)

---

## Implementation Plan

### Day 1: Normalize Command + Schema (8 hours)

**Morning (4h)**:
1. Create `cmd/ailang/normalize.go` (~200 LOC)
   - CLI flag parsing (--output, --format)
   - Fragment parsing (tolerate incomplete AST)
   - Module detection and insertion
   - Main function analysis
2. Create `schemas/normalize_edits_v1.json` (~50 LOC)
   - JSON Schema definition
   - Validation rules

**Afternoon (4h)**:
3. Implement effect inference (~80 LOC)
   - Scan function body for effect-requiring calls
   - Map calls to effects (println → IO, readFile → FS, etc.)
4. Write unit tests (~100 LOC)
   - Test module insertion
   - Test export addition
   - Test effect inference
   - Test return type addition
5. Create golden tests (~3 cases, ~60 LOC)
   - fragment_no_module
   - fragment_no_export
   - fragment_no_effects

**Deliverable**: Working `ailang normalize` command

### Day 2: Suggest Imports Command (8 hours)

**Morning (4h)**:
1. Create `cmd/ailang/suggest_imports.go` (~250 LOC)
   - CLI flag parsing
   - Parse and type check input
   - Collect unresolved symbols
2. Implement stdlib search (~100 LOC)
   - Index stdlib modules (cache in memory)
   - Match symbols to exports
   - Handle ambiguity (prefer specific over general)

**Afternoon (4h)**:
3. Create `schemas/import_suggestions_v1.json` (~60 LOC)
4. Write unit tests (~120 LOC)
   - Test symbol resolution
   - Test multi-symbol imports
   - Test ambiguity resolution
5. Create golden tests (~3 cases, ~60 LOC)
   - missing_decode
   - missing_multiple
   - missing_ambiguous

**Deliverable**: Working `ailang suggest-imports` command

### Day 3: Apply Command + Integration (6-8 hours)

**Morning (3-4h)**:
1. Create `cmd/ailang/apply.go` (~150 LOC)
   - CLI flag parsing (--output, --validate)
   - Load and validate edit JSONs
   - Merge edit lists
   - Apply edits (bottom-up)
2. Create `schemas/apply_result_v1.json` (~40 LOC)

**Afternoon (3-4h)**:
3. Write unit tests (~80 LOC)
   - Test edit merging
   - Test edit application (prepend, modify, append)
   - Test validation
4. Create golden tests (~3 cases, ~90 LOC)
   - apply_normalize_only
   - apply_imports_only
   - apply_both
5. Integration tests (~50 LOC)
   - End-to-end: fragment → normalize → suggest → apply → run

**Deliverable**: Complete tooling trio

### Day 3.5-4: CI Integration + Documentation (2-4 hours)

**Tasks**:
1. Add CI test step (~30 LOC in .github/workflows/, 1h)
   - `make test-tooling-json` - Run all golden tests
   - Validate determinism: run twice, compare outputs

2. Create documentation (~2h)
   - `docs/tools/normalize.md` - User guide
   - `docs/tools/suggest_imports.md` - User guide
   - `docs/tools/apply.md` - User guide
   - `docs/tools/json_schemas.md` - Schema reference

3. Update teaching prompts (~30min)
   - Mention tooling availability (for agent authors)
   - Add to AI FAQ: "How to fix code fragments"

4. Update CHANGELOG.md and README.md (~30min)

**Deliverable**: Documented, CI-integrated tooling

---

## Testing Strategy

### Unit Tests (cmd/ailang/*_test.go)

**Normalize Tests** (~10 tests):
- Module insertion: fragment → with module
- Export addition: func main() → export func main()
- Effect inference: println call → ! {IO}
- Return type: func main() → func main() -> ()
- Edge cases: already complete, multiple functions, nested blocks

**Suggest Imports Tests** (~12 tests):
- Single symbol: decode → import std/json (decode)
- Multiple symbols: decode, println → 2 imports
- Ambiguous: map → prefer std/list over std/prelude
- Already imported: no duplicate
- Unresolvable: unknown symbol → metadata flag

**Apply Tests** (~10 tests):
- Prepend: insert before line 1
- Append: insert after last line
- Modify: exact string replacement
- Delete: remove line
- Multiple edits: merge and apply bottom-up
- Validation: fail if result doesn't parse

**Coverage Target**: ≥85% line coverage on new code

### Golden Tests (testdata/tooling/)

**Format**:
- Input: `test_case.ail`
- Expected: `test_case.golden.json` (for commands) or `test_case.golden.ail` (for apply)
- Run with `go test -update` to regenerate

**Determinism Check**:
```bash
# Run twice, compare outputs byte-for-byte
ailang normalize fragment.ail > out1.json
ailang normalize fragment.ail > out2.json
diff out1.json out2.json  # Must be identical
```

### Integration Tests (internal/pipeline/tooling_integration_test.go)

**End-to-End Tests**:
```go
func TestTooling_EndToEnd_JsonParse(t *testing.T) {
    // Fragment from AI (no module, no imports, no effects)
    fragment := `
func main() {
  let data = decode("[{\"name\":\"Alice\"}]")
  println(show(data))
}
`

    // Step 1: Normalize
    edits := runNormalize(fragment)
    assert.Equal(t, "module benchmark/solution", edits.Edits[0].Text)

    // Step 2: Suggest imports
    imports := runSuggestImports(fragment)
    assert.Len(t, imports.Imports, 3)  // json, io, prelude

    // Step 3: Apply
    fixed := runApply(fragment, edits, imports)

    // Step 4: Verify it runs
    _, err := pipeline.CompileAndRun(fixed, []string{}, []string{"IO"})
    assert.NoError(t, err)
}
```

### CI Integration

**Makefile targets**:
```makefile
.PHONY: test-tooling-json
test-tooling-json:
	@echo "Running tooling JSON golden tests..."
	go test -v ./cmd/ailang -run Golden
	@echo "Verifying determinism..."
	./tools/verify_determinism.sh

.PHONY: test-tooling-integration
test-tooling-integration:
	@echo "Running tooling integration tests..."
	go test -v ./internal/pipeline -run Tooling
```

**GitHub Actions** (.github/workflows/ci.yml):
```yaml
- name: Test tooling (JSON determinism)
  run: make test-tooling-json

- name: Test tooling (integration)
  run: make test-tooling-integration
```

---

## Files Changed

### New Files

| File | Purpose | LOC | Tests |
|------|---------|-----|-------|
| `cmd/ailang/normalize.go` | Normalize command | ~200 | N/A |
| `cmd/ailang/normalize_test.go` | Unit + golden tests | ~160 | 13 tests |
| `cmd/ailang/suggest_imports.go` | Suggest imports command | ~250 | N/A |
| `cmd/ailang/suggest_imports_test.go` | Unit + golden tests | ~180 | 15 tests |
| `cmd/ailang/apply.go` | Apply command | ~150 | N/A |
| `cmd/ailang/apply_test.go` | Unit + golden tests | ~130 | 13 tests |
| `schemas/normalize_edits_v1.json` | JSON Schema | ~50 | N/A |
| `schemas/import_suggestions_v1.json` | JSON Schema | ~60 | N/A |
| `schemas/apply_result_v1.json` | JSON Schema | ~40 | N/A |
| `internal/pipeline/tooling_integration_test.go` | Integration tests | ~100 | 5 tests |
| `tools/verify_determinism.sh` | CI determinism check | ~30 | N/A |
| `testdata/tooling/**/*.ail` | Golden test inputs | ~200 | N/A |
| `testdata/tooling/**/*.golden.json` | Golden test outputs | ~300 | N/A |
| `docs/tools/normalize.md` | User guide | ~100 | N/A |
| `docs/tools/suggest_imports.md` | User guide | ~100 | N/A |
| `docs/tools/apply.md` | User guide | ~80 | N/A |
| `docs/tools/json_schemas.md` | Schema reference | ~120 | N/A |

**Total New Code**: ~2,200 LOC (implementation + tests + docs + schemas)

### Modified Files

| File | Changes | Reason |
|------|---------|--------|
| `cmd/ailang/main.go` | +~30 LOC | Register new commands |
| `Makefile` | +~15 LOC | Add tooling test targets |
| `.github/workflows/ci.yml` | +~10 LOC | Add tooling CI steps |
| `CHANGELOG.md` | +~80 LOC | Document v0.3.15 |
| `README.md` | +~20 LOC | Update tooling status |

**Total Modified**: ~155 LOC

**Grand Total**: ~2,355 LOC

---

## Success Metrics

| Metric | Target | Measurement |
|--------|--------|-------------|
| **Determinism** | 100% (byte-stable output) | Run each command 100x, compare outputs |
| **Golden tests** | All pass | `make test-tooling-json` |
| **Integration tests** | All pass | `make test-tooling-integration` |
| **Test coverage** | ≥85% on new code | `go test -cover cmd/ailang` |
| **Schema validation** | All outputs valid | CI validates against schemas |
| **Performance** | <100ms per operation | Benchmark suite |
| **CI green** | All tooling tests pass | GitHub Actions |
| **Documentation** | User guides + schemas | Manual review |
| **Timeline** | 3-4 days | Git commit timestamps |

---

## Risk Assessment

| Risk | Probability | Impact | Mitigation |
|------|------------|--------|------------|
| **Schema drift** | Medium | Medium | Version schemas (v1, v2, ...); validate in CI |
| **Effect inference errors** | Medium | High | Conservative inference; manual override flag (--effects IO,FS) |
| **Symbol resolution ambiguity** | Medium | Medium | Preference rules (std/list > std/prelude); log warnings |
| **Edit application ordering bugs** | Low | High | Extensive golden tests; reverse-order application |
| **Performance issues** | Low | Low | Stdlib index cached in memory; <100ms target |
| **Scope creep to run/diagnose** | Medium | Medium | Lock scope to trio only; defer other tools to v0.3.16+ |

---

## Dependencies

### Prerequisites
- ✅ M-LANG-JSON-DECODE (v0.3.14) - JSON encode/decode for testing
- ✅ Parser (v0.3.0+) - Fragment parsing
- ✅ Type checker (v0.3.0+) - Symbol resolution
- ✅ Stdlib (v0.3.13+) - Import candidates
- ✅ JSON Schema library (gojsonschema) - Validation

### Blocking Issues
- None identified

### Related Work
- **M-LANG-JSON-DECODE**: Required for JSON I/O testing
- **M-DX1**: Builtin system (used for effect inference)
- **v0.4.0 LSP**: Will consume these tools for real-time suggestions

---

## Future Enhancements (v0.4.0+)

### v0.4.0: LSP Integration
- Daemon mode for `normalize`, `suggest-imports`
- Real-time diagnostics streaming
- IDE integration (VSCode, Vim, Emacs)

### v0.4.1: Advanced Features
- Effect composer: Auto-infer minimal effect sets
- Test generator: From function signatures
- Import wildcard: `import std/io (*)` language support
- FX001 fix-it: Auto-add effect annotations via diagnostics

### v0.5.0: Optimization
- Incremental parsing (don't re-parse unchanged code)
- Parallel symbol resolution (multi-threaded stdlib search)
- Caching layer (memoize common transformations)

---

## Acceptance Checklist

**Code Complete**:
- [ ] `cmd/ailang/normalize.go` implemented (~200 LOC)
- [ ] `cmd/ailang/suggest_imports.go` implemented (~250 LOC)
- [ ] `cmd/ailang/apply.go` implemented (~150 LOC)
- [ ] JSON schemas created (3 files, ~150 LOC)
- [ ] Unit tests written (≥85% coverage)
- [ ] Golden tests created (~30 cases)
- [ ] Integration tests written (~5 cases)

**Testing**:
- [ ] All unit tests pass
- [ ] All golden tests pass
- [ ] All integration tests pass
- [ ] Determinism verified (100 runs)
- [ ] Schema validation passes
- [ ] Performance benchmarks <100ms

**CI**:
- [ ] `make test-tooling-json` passes
- [ ] `make test-tooling-integration` passes
- [ ] GitHub Actions green
- [ ] No regressions: `make test` all green

**Documentation**:
- [ ] User guides: normalize, suggest-imports, apply
- [ ] Schema reference: json_schemas.md
- [ ] CHANGELOG updated: v0.3.15 entry
- [ ] README updated: tooling status

**Release**:
- [ ] All checklist items complete
- [ ] CI green on dev branch
- [ ] Tag: `git tag v0.3.15`
- [ ] Announce in changelog

---

## Appendix: Usage Examples

### Example 1: Fix Fragment (json_parse)

**Input** (fragment.ail):
```ailang
func main() {
  let data = decode("[{\"name\":\"Alice\"}]")
  println(show(data))
}
```

**Step 1: Normalize**
```bash
ailang normalize fragment.ail --output normalize.json
```

**Output** (normalize.json):
```json
{
  "version": "1.0",
  "edits": [
    {
      "type": "prepend",
      "line": 1,
      "text": "module benchmark/solution\n\n"
    },
    {
      "type": "modify",
      "line": 1,
      "old": "func main()",
      "new": "export func main() -> () ! {IO}"
    }
  ]
}
```

**Step 2: Suggest Imports**
```bash
ailang suggest-imports fragment.ail --output imports.json
```

**Output** (imports.json):
```json
{
  "version": "1.0",
  "imports": [
    {"module": "std/json", "symbols": ["decode"]},
    {"module": "std/io", "symbols": ["println"]},
    {"module": "std/prelude", "symbols": ["show"]}
  ],
  "insert_after_line": 1
}
```

**Step 3: Apply**
```bash
ailang apply fragment.ail normalize.json imports.json --output fixed.ail
```

**Output** (fixed.ail):
```ailang
module benchmark/solution

import std/json (decode)
import std/io (println)
import std/prelude (show)

export func main() -> () ! {IO} {
  let data = decode("[{\"name\":\"Alice\"}]")
  println(show(data))
}
```

**Step 4: Verify**
```bash
ailang run --caps IO --entry main fixed.ail
# ✅ Compiles and runs
```

### Example 2: Chain Commands (Pipeline)

```bash
# One-liner: normalize + suggest + apply
ailang normalize fragment.ail | \
  jq -s '.[0]' > /tmp/edits.json && \
ailang suggest-imports fragment.ail | \
  jq -s '.[0]' > /tmp/imports.json && \
ailang apply fragment.ail /tmp/edits.json /tmp/imports.json --output fixed.ail
```

---

## References

- **Phase 1 Design**: design_docs/20251018/M-LANG-JSON-DECODE.md
- **Original Sprint Ticket**: /plan-sprint arguments (2025-10-18)
- **Eval Harness**: docs/docs/guides/evaluation/README.md
- **Parser**: internal/parser/
- **Type Checker**: internal/types/
- **Stdlib**: stdlib/std/
