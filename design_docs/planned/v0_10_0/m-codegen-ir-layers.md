# M-CODEGEN-IR-LAYERS: Multi-Target Codegen IR Architecture

**Status**: Planned
**Target**: v0.10.0+
**Priority**: P1 (High — prerequisite for Rust/WASM targets, blocking codegen quality)
**Estimated**: TBD (phased, each phase independently valuable)
**Dependencies**: M-CODEGEN-REGISTRY-ONLY (v0.9.2 ✅)
**Milestone ID**: M-CODEGEN-IR-LAYERS
**Created**: 2026-03-18

---

## Problem Statement

The Go codegen (`internal/gen/golang/`) is a **10,610-line monolith** that has become the hardest subsystem to maintain. It has accumulated 55 separate milestone initiatives, 34 Generator state fields, 1,059 hand-rolled `g.writef()` calls, and exactly 1 integration test. Every DocParse bring-up session produces 3-8 codegen bugs that are only discovered at `go build` time.

Adding a Rust target would mean duplicating this entire system — including all the bugs and fragility. The current architecture cannot scale to multi-target codegen.

### Quantified Pain

| Metric | Value | Risk |
|--------|-------|------|
| Total LOC (non-test) | 10,610 across 29 files | Complexity saturation |
| Generator state fields | 34 maps/flags | Cache invalidation bugs |
| `g.writef()` calls | 1,059 | No intermediate representation |
| Integration tests | 1 (and it's broken) | Regressions undetected |
| M- milestone markers | 55 distinct codes | Tangled initiative history |
| Helper bodies as strings | ~60 entries | No IDE support, no type checking |
| Silent fallback paths | 5+ (ToPascalCase, GeInt default, etc.) | Broken code generated without error |

### Root Cause: No Intermediate Representation

The codegen goes directly from **Core AST → Go source text**. There is no intermediate layer where:
- Functions can be validated before emission
- Missing references can be detected
- ADT guard logic can be applied uniformly
- Multiple backends could share the same lowering

This means every new target (Rust, WASM, TypeScript) would need to re-solve: type mapping, ADT dispatch, pattern matching strategy, stdlib binding, type assertion placement, and module namespacing — all from scratch.

---

## Strategic Assessment: What Makes This Hard

### 1. Silent Failures Are the Default

When a builtin has no codegen spec, `codegen_expr_simple.go:171` emits `ToPascalCase(name)` — valid Go syntax referencing a function that doesn't exist. The codegen succeeds; `go build` fails. This is the #1 source of DocParse debugging time.

**Compare:** A Rust codegen would have the same problem but worse — Rust's type system would reject more things, producing even more cryptic errors from generated code.

### 2. State Management is Per-Function, Per-Module, AND Global

The Generator has three state lifetimes that must be manually managed:
- **Global**: `adtConstructors`, `recordTypes` (persist across all modules)
- **Per-module**: `topLevelFuncs`, `emittedVars` (reset between modules)
- **Per-function**: `interfaceCache`, `typedLocalVars` (reset between functions)

No safeguards prevent cross-contamination. A multi-module compilation bug from March 2026 was caused by exactly this — function names leaking across modules.

### 3. Type Dispatch is Scattered and Incomplete

Type-aware code generation happens at 8+ sites:
- ADT constructor calls (type assertions)
- Function call sites (parameter conversions)
- Match arm bindings (pattern variable typing)
- Record field access (pointer vs value)
- Comparison operators (int vs float vs string — **TODO: incomplete**)
- Slice conversions (interface{} → typed slices)

Each site has independent logic. Adding a new type category means updating all 8 sites.

### 4. Runtime Helpers Are Go Code in Go Strings

The registry system (post M-CODEGEN-REGISTRY-ONLY) stores helper function bodies as raw strings:
```go
Body: `list := toSlice(xs)
result := make([]interface{}, len(list))
for i, x := range list {
    result[i] = CallFunc(f, x)
}
return result`
```

No syntax highlighting, no type checking, no refactoring support. A typo in a helper body is only discovered at `go build` time on the generated output.

---

## Proposed Architecture: Phased Improvement

Rather than a big-bang rewrite, each phase is independently valuable and can ship separately.

### Phase 0: Compile-Check Gate (v0.9.3)

**Add `go build` verification to the codegen pipeline itself.**

After generating all `.go` files, run `go build` on the output directory. If it fails, report the errors as codegen errors — not as a surprise to the user. This is the single highest-leverage change: it turns every silent failure into a loud failure at the right time.

Implementation: ~50 lines in `cmd/ailang/compile.go` after the generation loop. Run `go build ./...` in the output directory, capture stderr, report as codegen diagnostics.

This also enables a CI test: compile every example to Go, verify `go build` passes.

### Phase 1: Reference Validation (v0.9.3)

**Track all emitted function names and all referenced function names. Error on unresolved references.**

During codegen, maintain two sets:
- `definedFuncs`: every `func Name(...)` emitted
- `referencedFuncs`: every `Name(...)` call emitted

After generation, `referencedFuncs - definedFuncs` = missing functions. Report these as codegen errors instead of silently producing broken Go.

This catches the `ToPascalCase` fallback problem at codegen time, not at `go build` time.

### Phase 2: Helper Body Validation (v0.10.0)

**Parse and type-check helper bodies at init time, not at generation time.**

When `registerCodegenSpecs()` runs, parse each `Helper.Body` string as a Go function body using `go/parser`. Report syntax errors immediately on startup. This catches typos in helper bodies before any compilation happens.

Could also generate helper bodies as actual Go files in a `codegen_helpers/` package that gets compiled as part of the AILANG build, giving full IDE support.

### Phase 3: Codegen IR (v0.10.0+)

**Introduce a target-independent intermediate representation between Core AST and target source.**

```
Core AST → Codegen IR → Go emitter
                      → Rust emitter (future)
                      → WASM emitter (future)
```

The Codegen IR would represent:
- Function declarations with typed parameters
- ADT constructor calls (target-agnostic)
- Pattern match dispatch strategies
- Stdlib function calls (resolved, not by name)
- Type assertions (where needed, target-specific)

Each emitter would only need to handle IR → source text, not the full complexity of Core → source.

**This is the big one** — but Phases 0-2 are prerequisites that make Phase 3 possible by establishing invariants the IR can rely on.

### Phase 4: Generator State Refactor (v0.10.0+)

**Replace the 34-field Generator struct with scoped state objects.**

```go
type CompilationState struct {
    Global  *GlobalState   // adtConstructors, recordTypes
    Module  *ModuleState   // topLevelFuncs, emittedVars
    Func    *FuncState     // typedLocalVars, interfaceCache
}
```

State transitions are explicit: `enterModule()` creates a new `ModuleState`, `enterFunction()` creates a new `FuncState`. No manual reset, no cross-contamination possible.

### Phase 5: Execution Profiles (v0.11.0+)

**Formalize the three entry-point shapes as "execution profiles" with effect budgets.**

*Extracted from the archived [execution-profiles.md](../../archive/execution-profiles.md) design doc (originally planned for v0.6.0). Profiles depend on having an IR (Phase 3) so that wrapper generation and effect budget validation can be target-independent.*

Every serious AILANG program takes one of three shapes:

| Profile | Entry Signature | Primary Use |
|---------|----------------|-------------|
| **SimProfile** | `func step(world: World, input: Input) -> (World, Output) ! {RNG, Debug, AI}` | Games, RL envs, agent worlds, workflow engines |
| **ServiceProfile** | `func handle(req: Request) -> Response ! {AI, Debug}` | Microservices, HTTP/gRPC handlers, agent tools |
| **CliProfile** | `func main(args: [string]) -> () ! {IO, FS, Env, Debug}` | CLI tools, scripts, config transformers |

Each profile defines:
- **Entry function shape** — validated at compile time
- **Effect budget** — which effects are allowed (others are compile errors)
- **Wrapper generation** — the IR emitter produces the appropriate entry wrappers per target

Implementation:
1. Add `--profile sim|service|cli` flag to `ailang compile`
2. Auto-detect profile from entry function signature when `--profile` is omitted
3. Validate effect usage against the profile's budget during IR lowering (Phase 3)
4. Generate profile-specific wrappers in each target emitter (Go: `Init()`/`Step()`, `Handle()`, `Main()`)

This is the payoff of the IR architecture: profiles are a ~200-line layer on top of the IR, not a cross-cutting concern tangled into every emitter.

---

## What This Means for Rust/WASM Targets

**Before Phase 3:** Don't start Rust codegen. It would duplicate all 10,610 lines of Go-specific logic, including the bugs. Every fix would need to be applied in two places.

**After Phase 0-2:** Rust codegen is possible but still duplicative. Each target is a separate emitter with shared nothing.

**After Phase 3:** Rust codegen shares the IR with Go. New targets are ~2,000 LOC each (IR → source text) instead of ~10,000 LOC each (Core → source text).

**Recommended path:**
1. Ship Phases 0-1 in v0.9.3 (2-3 days) — immediate quality improvement
2. Ship Phase 2 in v0.10.0 (1-2 days) — helper body reliability
3. Evaluate Phase 3 scope after DocParse is stable on Go codegen
4. Only start Rust target after Phase 3 IR exists

---

## Verification Plan

### Phase 0
```bash
# Add to make ci:
make compile-examples-go   # Compile all examples to Go + go build
# Expected: 0 failures (currently unknown)
```

### Phase 1
```bash
# Codegen should error on unresolved references:
ailang compile --emit-go examples/runnable/json_parsing.ail
# Should report: "codegen error: GetNumber referenced but not defined"
# Instead of silently generating broken Go
```

### Phase 2
```bash
# Registry init should fail on bad helper bodies:
make test
# Should catch: syntax errors in Helper.Body strings at test time
```

---

## Appendix: Current File Map

| File | Lines | Responsibility |
|------|-------|----------------|
| codegen.go | 710 | Generator struct, orchestration |
| codegen_match.go | 859 | Pattern matching (3 strategies) |
| contracts.go | 600 | Contract enforcement |
| codegen_ops.go | 586 | Binary/unary operators, type dispatch |
| codegen_expr_let.go | 567 | Let/LetRec binding flattening |
| codegen_type_analysis.go | 504 | Type inference for assertions |
| codegen_record.go | 489 | Record literal & update generation |
| adt.go | 485 | ADT type declaration generation |
| codegen_runtime_slices.go | 461 | Slice converter generation |
| codegen_expr_simple.go | 405 | Literals, variables, lambdas |
| codegen_expr_app.go | 386 | Function application |
| codegen_runtime_collections.go | 340 | List infrastructure (Cons, toSlice) |
| codegen_runtime_arith.go | 280 | Arithmetic helpers |
| codegen_runtime_misc.go | 220 | CallFunc, Show, Log |
| codegen_runtime_records.go | 180 | RecordUpdate, FieldGet |
| codegen_registry.go | 220 | Registry emission + ADT guards |
| codegen_runtime.go | 37 | Orchestration (post-migration) |
| codegen_dictionaries.go | 200 | Type class dictionaries |
| *29 files total* | *10,610* | |
