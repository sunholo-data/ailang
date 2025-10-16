# Claude Instructions for AILANG Development

## ⚠️ CRITICAL PRINCIPLES

### 1. ALWAYS USE EXISTING TOOLS FIRST

**Before writing ANY new script or code:**
1. ✅ Check `make help` for existing targets
2. ✅ Check `tools/` directory for existing scripts
3. ✅ Check this CLAUDE.md for documented workflows
4. ✅ Search codebase: `grep -r "function_name" internal/`

**Common mistakes to avoid:**
- ❌ Writing new bash scripts when `make` targets or `ailang` commands exist
- ❌ Creating new analysis tools when M-EVAL-LOOP Go implementation exists
- ❌ Guessing model names instead of checking `internal/eval_harness/models.yml`
- ❌ Ignoring documented workflows in CLAUDE.md
- ❌ Manually extracting/formatting data when automated tools exist
- ❌ Guessing which tools to use for benchmarks/evals - ALWAYS use eval-orchestrator agent

### 2. NO SILENT FALLBACKS - FAIL LOUDLY

**CRITICAL LESSON**: Silent fallbacks hide bugs and produce wrong data that users trust.

**The Cost Calculation Bug (Oct 2024):**
```go
// ❌ WRONG - Silent fallback hid 61x cost overestimation for YEARS
rate, ok := rates[model]
if !ok {
    rate = 0.03  // Default to GPT-4 pricing if unknown
}
return float64(tokens) / 1000.0 * rate
```

**Impact**: All modern models (GPT-5, Gemini 2.5, Claude Sonnet 4.5) used wrong pricing.
Users trusted inflated costs. Bug was invisible until someone questioned the numbers.

**The Principle:**
```go
// ✅ CORRECT - Return 0 or error to force investigation
if GlobalModelsConfig == nil {
    return 0.0  // Better to see $0.00 than trust wrong data
}

cost, err := GlobalModelsConfig.CalculateCostForModel(model, inputTokens, outputTokens)
if err != nil {
    return 0.0  // NO SILENT FALLBACKS - we want to know when pricing is missing
}
```

**When to apply:**
- ✅ Pricing/cost calculations (return $0.00 if unknown)
- ✅ Model configurations (fail if model not in models.yml)
- ✅ Required environment variables (fail if missing, don't use defaults)
- ✅ Data validation (reject invalid data, don't silently fix)
- ✅ Configuration loading (fail if config invalid, don't use built-in defaults)

**When fallbacks ARE okay:**
- ✅ UI defaults (empty state, placeholder text)
- ✅ Optional features (graceful degradation of non-critical features)
- ✅ Caching (miss → fetch from source)
- ✅ Performance optimizations (slow path if fast path unavailable)

**Rule of thumb:** If the fallback value affects data integrity, business logic, or user decisions → **NO FALLBACK**. Return zero, null, or error instead.

**When asked to run evals, compare benchmarks, or update benchmark results:**

→ **ALWAYS use the [eval-orchestrator](.claude/agents/eval-orchestrator.md) agent**

The agent knows how to:
- Run benchmarks with cost-conscious defaults (cheap models for dev, --full for releases)
- Compare results, validate fixes, generate reports
- Update the benchmark dashboard (docs/BENCHMARK_COMPARISON.md)
- Use all available models and their pricing
- Route to appropriate `ailang eval-*` commands

**DO NOT:**
- ❌ Try to guess which make targets or scripts to use
- ❌ Write custom Python/bash scripts for benchmark analysis
- ❌ Manually regenerate dashboard files
- ❌ Call `ailang eval-*` commands directly (let the agent handle it)

---

## Project Overview
AILANG is an AI-first programming language designed for AI-assisted development. It features:
- ✅ **Pure functional programming** - First-class functions, closures, lambda calculus
- ✅ **Algebraic effects** - Capability-based effect system (IO, FS) with runtime security
- ✅ **Hindley-Milner type inference** - Full type system with type classes and row polymorphism
- ❌ Typed quasiquotes for safe metaprogramming (planned v0.4.0+)
- ❌ CSP-based concurrency with session types (planned v0.4.0+)
- ❌ Deterministic execution for AI training data generation (planned v0.4.0+)
- File extension: `.ail`

## What AILANG Can Do (Implementation Status)

**Language Features** (see [CHANGELOG.md](CHANGELOG.md) for version history):
- ✅ Pure functional programming (lambda calculus, closures, recursion)
- ✅ Hindley-Milner type inference with type classes and row polymorphism
- ✅ Algebraic effects with capability-based security (IO, FS)
- ✅ Pattern matching with ADTs
- ✅ Module system with runtime execution
- ✅ Interactive REPL with full type checking
- ✅ Block expressions `{ e1; e2; e3 }` for sequencing
- ❌ Typed quasiquotes (planned)
- ❌ CSP concurrency (planned)
- ❌ AI training data export (planned)

**Development Tools:**
- ✅ M-EVAL: AI code generation benchmarks (multi-model support)
- ✅ M-EVAL-LOOP v2.0: Native Go eval tools with 90%+ test coverage
- ✅ Plan validation and code scaffolding (`internal/planning/`)
- ✅ Structured error reporting with JSON schemas

**Quick Test:**
```bash
make test                # Run all tests
make verify-examples     # Check example files
ailang repl             # Start REPL
```

**For detailed version history, see [CHANGELOG.md](CHANGELOG.md)**

**🎉 MAJOR MILESTONE:** Module files now execute! Use `ailang run --caps IO,FS --entry main module.ail` to run module code with effects.

**⚠️ Important**: Flags must come BEFORE the filename when using `ailang run`.

## Key Design Principles
1. **Explicit Effects**: All side effects must be declared in function signatures
2. **Everything is an Expression**: No statements, only expressions that return values
3. **Type Safety**: Static typing with Hindley-Milner inference + row polymorphism
4. **Deterministic**: All non-determinism must be explicit (seeds, virtual time)
5. **AI-Friendly**: Generate structured execution traces for training

## Project Structure (v0.3.0+)
```
ailang/
├── cmd/ailang/         # CLI entry point ✅ COMPLETE
├── internal/
│   ├── ast/            # Surface AST ✅ COMPLETE
│   ├── lexer/          # Tokenizer ✅ COMPLETE
│   ├── parser/         # Parser ✅ COMPLETE
│   ├── core/           # Core AST (ANF) ✅ COMPLETE
│   ├── elaborate/      # Surface → Core elaboration ✅ COMPLETE
│   ├── types/          # Type system ✅ COMPLETE
│   ├── typeclass/      # Type classes ✅ COMPLETE (stub)
│   ├── link/           # Dictionary linking ✅ COMPLETE
│   ├── pipeline/       # Full compilation pipeline ✅ COMPLETE
│   ├── eval/           # Evaluator ✅ COMPLETE (Core + module support)
│   ├── repl/           # Interactive REPL ✅ COMPLETE
│   ├── runtime/        # Module execution runtime ✅ COMPLETE (v0.2.0)
│   ├── effects/        # Effect system runtime ✅ COMPLETE (v0.2.0)
│   ├── loader/         # Module loader ✅ COMPLETE
│   ├── errors/         # Error reporting ✅ COMPLETE
│   ├── schema/         # JSON schemas ✅ COMPLETE
│   ├── eval_harness/   # AI evaluation framework ✅ COMPLETE (M-EVAL)
│   ├── eval_analysis/  # Go eval tools ✅ COMPLETE (M-EVAL v2.0)
│   ├── eval_analyzer/  # Failure analyzer ✅ COMPLETE (M-EVAL v2.0)
│   ├── planning/       # Plan validation & scaffolding ✅ COMPLETE
│   ├── builtins/       # Builtin definitions ✅ COMPLETE
│   ├── dtree/          # Decision trees (pattern matching) ✅ COMPLETE
│   ├── iface/          # Interface definitions ✅ COMPLETE
│   ├── manifest/       # Module manifests ✅ COMPLETE
│   ├── module/         # Module system ✅ COMPLETE
│   ├── typedast/       # Typed AST ✅ COMPLETE
│   ├── channels/       # CSP implementation ❌ TODO (v0.4.0+)
│   └── session/        # Session types ❌ TODO (v0.4.0+)
├── stdlib/             # Standard library ✅ COMPLETE (std/io, std/fs, std/prelude)
├── tools/              # Development tools ✅ (eval, benchmarking, verification)
├── benchmarks/         # AI code generation benchmarks ✅
├── examples/           # Example .ail programs (66 files, 48 passing)
├── tests/              # Test suite ✅
└── docs/               # Documentation ✅ COMPLETE
```

## Development Workflow

### Building and Testing
```bash
make build          # Build the interpreter to bin/
make install        # Install ailang to system (makes it available everywhere)
make test           # Run all tests
make run FILE=...   # Run an AILANG file
make repl           # Start interactive REPL
```

### Adding Builtin Functions (✅ M-DX1 - v0.3.9)

**AILANG has a modern builtin development system that reduces implementation time from 7.5h to 2.5h (-67%).**

#### Quick Start (2.5 hours instead of 7.5)

**Step 1: Register the builtin** (~30 min)
```go
// internal/builtins/register.go
func init() {
    registerMyBuiltin()
}

func registerMyBuiltin() {
    RegisterEffectBuiltin(BuiltinSpec{
        Module:  "std/string",
        Name:    "_str_reverse",
        NumArgs: 1,
        IsPure:  true,        // or false with Effect: "IO"
        Type:    makeReverseType,
        Impl:    strReverseImpl,
    })
}

func makeReverseType() types.Type {
    T := types.NewBuilder()
    return T.Func(T.String()).Returns(T.String())
}

func strReverseImpl(ctx *effects.EffContext, args []eval.Value) (eval.Value, error) {
    str := args[0].(*eval.StringValue).Value
    runes := []rune(str)
    for i, j := 0, len(runes)-1; i < j; i, j = i+1, j-1 {
        runes[i], runes[j] = runes[j], runes[i]
    }
    return &eval.StringValue{Value: string(runes)}, nil
}
```

**Step 2: Write hermetic tests** (~1 hour)
```go
// internal/builtins/register_test.go
func TestStrReverse(t *testing.T) {
    ctx := testctx.NewMockEffContext()

    tests := []struct {
        input    string
        expected string
    }{
        {"hello", "olleh"},
        {"", ""},
        {"🎉", "🎉"},
    }

    for _, tt := range tests {
        result, err := strReverseImpl(ctx, []eval.Value{
            testctx.MakeString(tt.input),
        })
        assert.NoError(t, err)
        assert.Equal(t, tt.expected, testctx.GetString(result))
    }
}
```

**Step 3: Validate and inspect** (~30 min)
```bash
# Enable the new registry
export AILANG_BUILTINS_REGISTRY=1

# Validate the builtin
ailang doctor builtins
# ✅ All builtins are valid!

# List all builtins
ailang builtins list --by-module
# # std/string (2)
#   _str_len                       [pure]
#   _str_reverse                   [pure]

# Test in REPL (when M-DX1.5 is implemented)
ailang repl
> :type _str_reverse
string -> string
```

**Step 4: Wire to runtime** (~30 min)
- Already done! The registry automatically wires to runtime/link when `AILANG_BUILTINS_REGISTRY=1`

#### Key Components

**Central Registry** (`internal/builtins/spec.go`):
- Single-point registration with `RegisterEffectBuiltin()`
- Compile-time validation (arity, types, impl, effects)
- Feature flag: `AILANG_BUILTINS_REGISTRY=1`
- Freeze-safe (no registration after init)

**Type Builder DSL** (`internal/types/builder.go`):
- Fluent API: `T.Func(args...).Returns(ret).Effects(effs...)`
- Reduces type construction from 35→10 lines (-71%)
- Methods: `String()`, `Int()`, `Bool()`, `List()`, `Record()`, `Func()`, `Returns()`, `Effects()`

**Test Harness** (`internal/effects/testctx/`):
- `MockEffContext` with HTTP/FS mocking
- Value constructors: `MakeString()`, `MakeInt()`, `MakeRecord()`, etc.
- Value extractors: `GetString()`, `GetInt()`, `GetRecord()`, etc.
- Hermetic testing (no real network/FS)

**Validation & Inspection**:
- `ailang doctor builtins` - Health checks with actionable diagnostics
- `ailang builtins list` - Browse registry (--by-effect, --by-module)
- 6 validation rules: type, impl, arity, effect consistency, module

#### Examples

**Pure function:**
```go
RegisterEffectBuiltin(BuiltinSpec{
    Module:  "std/string",
    Name:    "_str_len",
    NumArgs: 1,
    IsPure:  true,
    Type:    func() types.Type {
        T := types.NewBuilder()
        return T.Func(T.String()).Returns(T.Int())
    },
    Impl: func(ctx *effects.EffContext, args []eval.Value) (eval.Value, error) {
        s := args[0].(*eval.StringValue).Value
        return &eval.IntValue{Value: len([]rune(s))}, nil
    },
})
```

**Effect function with HTTP:**
```go
RegisterEffectBuiltin(BuiltinSpec{
    Module:  "std/net",
    Name:    "_net_httpRequest",
    NumArgs: 4,
    Effect:  "Net",
    Type:    makeHTTPRequestType,
    Impl:    effects.NetHTTPRequest,  // Uses ctx.GetHTTPClient()
})
```

**Complex types with records:**
```go
func makeHTTPRequestType() types.Type {
    T := types.NewBuilder()

    headerType := T.Record(
        types.Field("name", T.String()),
        types.Field("value", T.String()),
    )

    responseType := T.Record(
        types.Field("status", T.Int()),
        types.Field("headers", T.List(headerType)),
        types.Field("body", T.String()),
    )

    return T.Func(
        T.String(),           // url
        T.String(),           // method
        T.List(headerType),   // headers
        T.String(),           // body
    ).Returns(
        T.App("Result", responseType, T.Con("NetError")),
    ).Effects("Net")
}
```

#### Testing Patterns

**Hermetic HTTP tests:**
```go
func TestNetHTTPRequest(t *testing.T) {
    server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
        w.WriteHeader(200)
        w.Write([]byte(`{"status": "ok"}`))
    }))
    defer server.Close()

    ctx := testctx.NewMockEffContext()
    ctx.GrantAll("Net")
    ctx.SetHTTPClient(server.Client())

    result, err := effects.NetHTTPRequest(ctx,
        testctx.MakeString(server.URL),
        testctx.MakeString("GET"),
        testctx.MakeList([]eval.Value{}),
        testctx.MakeString(""),
    )

    assert.NoError(t, err)
    resp := testctx.GetRecord(result)
    assert.Equal(t, 200, testctx.GetInt(resp["status"]))
}
```

#### Migration from Legacy Registry

**Before (legacy, 4 files, 35 lines of types):**
```go
// internal/eval/builtins.go
registry.Register("_str_len", func(args []Value) (Value, error) { ... })

// internal/link/builtin_module.go
iface.Decls["_str_len"] = &iface.FuncDecl{
    Type: &types.TFunc2{
        Params: []types.Type{&types.TCon{Name: "String"}},
        Return: &types.TCon{Name: "Int"},
        EffectRow: &types.Row{Kind: types.KEffect{}, Labels: map[string]types.Type{}, Tail: nil},
    },
}

// internal/runtime/builtins.go
br.RegisterPure("_str_len", ...)

// internal/types/builtins.go
builtinTypes["_str_len"] = ...
```

**After (new registry, 1 file, 10 lines):**
```go
// internal/builtins/register.go
RegisterEffectBuiltin(BuiltinSpec{
    Module:  "std/string",
    Name:    "_str_len",
    NumArgs: 1,
    IsPure:  true,
    Type: func() types.Type {
        T := types.NewBuilder()
        return T.Func(T.String()).Returns(T.Int())
    },
    Impl: strLenImpl,
})
```

#### Metrics

| Metric | Before | After | Improvement |
|--------|--------|-------|-------------|
| Files to edit | 4 | 1 | -75% |
| Type construction LOC | 35 | 10 | -71% |
| Development time | 7.5h | 2.5h | -67% |
| Test setup LOC | ~50 | ~15 | -70% |

#### Status

**Completed (v0.3.9-alpha3):**
- ✅ M-DX1.1: Central Registry with validation
- ✅ M-DX1.2: Type Builder DSL
- ✅ M-DX1.3: Doctor + List CLI commands
- ✅ M-DX1.4: Test Harness with mocking
- ✅ 2 proof-of-concept migrations (_str_len, _net_httpRequest)
- ✅ 57 tests (100% coverage on new code)

**Planned (v0.3.10+, see design_docs/planned/m-dx1-day3-polish.md):**
- ⏳ M-DX1.5: REPL :type command (~3h)
- ⏳ M-DX1.6: Enhanced diagnostics (~3h)
- ⏳ M-DX1.7: docs/ADDING_BUILTINS.md guide (~2h)

**For full documentation, see:**
- Detailed examples: (to be created in M-DX1.7)
- Design rationale: `design_docs/planned/easier-ailang-dev.md`
- Test coverage: `internal/builtins/*_test.go`, `internal/effects/testctx/*_test.go`

### M-EVAL-LOOP: AI Evaluation & Self-Improvement (✅ COMPLETE - v2.0)

**When user asks about evaluations, benchmarks, or testing AI code generation:**

→ **Use the [eval-orchestrator](.claude/agents/eval-orchestrator.md) agent**

The agent handles all eval workflows:
- Running benchmarks (defaults to cheap/fast models)
- Comparing results and validating fixes
- Generating reports and interpreting metrics
- Routing to appropriate `ailang eval-*` commands

**For automated fix implementation:**

→ **Use the [eval-fix-implementer](.claude/agents/eval-fix-implementer.md) agent**

**Documentation** (for detailed reference):
- [Architecture Overview](docs/docs/guides/evaluation/architecture.md) - Commands & workflows
- [Evaluation README](docs/docs/guides/evaluation/README.md) - Quick start guide

**⚠️ CRITICAL: Running Multiple Models**

**The `ailang eval-suite` command OVERWRITES the output directory!**

```bash
# ❌ WRONG - Second run overwrites first run's results
ailang eval-suite --models gpt5
ailang eval-suite --models claude-sonnet-4-5  # DELETES gpt5 results!

# ✅ CORRECT - Run all models in ONE command
ailang eval-suite --models gpt5,claude-sonnet-4-5,gemini-2-5-pro

# ✅ ALSO CORRECT - Use different output directories
ailang eval-suite --models gpt5 --output eval_results/gpt5_only
ailang eval-suite --models claude-sonnet-4-5 --output eval_results/claude_only
```

**Default model sets:**
- `ailang eval-suite` → Reads from `dev_models` in models.yml (currently: gpt5-mini, claude-haiku-4-5, gemini-2-5-flash)
- `ailang eval-suite --full` → gpt5, claude-sonnet-4-5, gemini-2-5-pro (expensive)

**For baselines with all 6 models:**
```bash
ailang eval-suite --models gpt5,gpt5-mini,claude-sonnet-4-5,claude-haiku-4-5,gemini-2-5-pro,gemini-2-5-flash
```

**Quick reference - Common eval commands:**
```bash
# Update benchmark dashboard (PRESERVES HISTORY!)
ailang eval-report eval_results/baselines/v0.3.9 v0.3.9 --format=json
# ✅ Automatically writes to docs/static/benchmarks/latest.json
# ✅ Preserves all historical versions
# ✅ Validates before writing
# ✅ Atomic writes (no corruption)

# Generate markdown dashboard
ailang eval-report eval_results/baselines/v0.3.9 v0.3.9 --format=markdown > docs/BENCHMARK_COMPARISON.md

# Run baseline
make eval-baseline              # Uses dev models by default
make eval-baseline FULL=true    # Uses expensive models

# Compare two baselines
ailang eval-compare eval_results/baselines/v0.3.8 eval_results/baselines/v0.3.9

# Generate performance matrix
ailang eval-matrix eval_results/baselines/v0.3.9 v0.3.9
```

**⚠️ CRITICAL - Dashboard Update Workflow (v0.3.10+)**

**The dashboard JSON now preserves history automatically!**

```bash
# ✅ CORRECT - Safe, preserves history
ailang eval-report eval_results/baselines/v0.3.10 v0.3.10 --format=json
# Reads existing latest.json → merges history → validates → writes atomically

# ❌ WRONG - Don't redirect stdout (bypasses history preservation)
ailang eval-report ... --format=json > docs/static/benchmarks/latest.json
```

**How it works:**
1. Loads existing `docs/static/benchmarks/latest.json`
2. Builds new entry from current results
3. Merges into history (updates if version exists, appends if new)
4. Validates JSON structure (no duplicates, required fields present)
5. Writes atomically (temp file + rename, no partial writes)
6. Also prints to stdout (for backwards compatibility)

**DO NOT**:
- ❌ Create new bash scripts for evals - agents use existing `ailang eval-*` commands
- ❌ Duplicate agent logic - just invoke the appropriate agent
- ❌ Write custom analysis tools - extend `internal/eval_analysis/` if needed
- ❌ Run multiple `ailang eval-suite` commands to same directory - results will be overwritten!
- ❌ Search for dashboard generation scripts - just use `ailang eval-report`
- ❌ Redirect `--format=json` to file (bypasses history preservation logic!)
- ❌ Manually edit latest.json (use eval-report to update it)

### Code Quality & Coverage
```bash
make test-coverage-badge  # Quick coverage check (shows: "Coverage: 29.9%")
make test-coverage        # Run tests with coverage, generates HTML report
make lint                 # Run golangci-lint
make fmt                  # Format all Go code
make fmt-check            # Check if code is formatted
make vet                  # Run go vet
```

### Example Management
```bash
make verify-examples      # Verify all example files work/fail
make update-readme        # Update README with example status
make flag-broken          # Add warning headers to broken examples
```

### Development Helpers
```bash
make deps                 # Install all dependencies
make clean                # Remove build artifacts and coverage files
make ci                   # Run full CI verification locally
make help                 # Show all available make targets
```

#### Keeping `ailang` Up to Date

**After making code changes to the ailang binary:**
```bash
make quick-install  # Fast reinstall (recommended for development)
# OR
make install        # Full reinstall with version info
```

**Important**: The `ailang` command in your PATH points to `/Users/mark/go/bin/ailang` (system install), NOT `bin/ailang` (local build). Always run `make install` or `make quick-install` after building to update the system binary. Otherwise your changes won't be used when running `ailang` commands.

**For local testing without install:**
```bash
./bin/ailang <command>  # Use local build directly
```

### IMPORTANT: Keeping Documentation Updated

**Required documentation updates for every change:**

#### 1. README.md
- Update implementation status when adding new features
- Update current capabilities when functionality changes
- Update examples when they're fixed or new ones added
- Keep line counts and completion status accurate
- Document new builtin functions and operators
- Update the roadmap as items are completed

#### 2. CHANGELOG.md
**Must be updated for every feature or bug fix:**
- Follow semantic versioning (vMAJOR.MINOR.PATCH)
- Group changes by category: Added, Changed, Fixed, Deprecated, Removed
- Include code locations for new features (e.g., `internal/schema/`)
- Note breaking changes clearly
- Add migration notes if needed
- Include metrics (lines of code, test coverage)

Example entry:
```markdown
## [v3.2.0] - 2024-09-28

### Added
- Schema Registry (`internal/schema/`) - Versioned JSON schemas
- Error JSON Encoder (`internal/errors/`) - Structured error reporting
- Test coverage: 100% for new packages
- Total new code: ~1,500 lines
```

#### 3. Design Documentation
- **Before starting**: Create design doc in `design_docs/planned/`
- **After completing**: Move to `design_docs/implemented/vX_Y/`
- Include implementation report with metrics and limitations

**CRITICAL: Example Files Required**
**Every new language feature MUST have a corresponding example file:**
- Create `examples/feature_name.ail` for each new feature
- Include comprehensive examples showing all capabilities
- Add comments explaining the behavior and expected output
- ⚠️ **Test that examples actually work with current implementation**
- ⚠️ **Add warning headers to examples that don't work**
- These examples will be used in documentation and tutorials
- Always test examples before documenting them as working

### Writing AILANG Code

**When writing AILANG code during development:**
Refer to the **AI Teaching Prompt** for comprehensive syntax guidance:
- **Current version**: [prompts/v0.3.8.md](prompts/v0.3.8.md)
- Validated through multi-model testing (Claude, GPT, Gemini)
- Covers syntax, limitations, common pitfalls, and working examples

**Quick reference:**
```bash
ailang run --caps IO,FS --entry main module.ail  # Run module
ailang repl                                        # Start REPL
:type expr                                         # Check type in REPL
```

**For detailed syntax, limitations, and examples:**
- See [prompts/v0.3.8.md](prompts/v0.3.8.md) - Complete AILANG teaching prompt
- See [docs/LIMITATIONS.md](docs/LIMITATIONS.md) - Known limitations and workarounds
- See [examples/](examples/) - 66 example files (48 working)

### Common Tasks

#### Adding a New Language Feature
1. Update token definitions in `internal/lexer/token.go`
2. Modify lexer in `internal/lexer/lexer.go` to recognize tokens
3. Add AST nodes in `internal/ast/ast.go`
4. Update parser in `internal/parser/parser.go`
5. Add type rules in `internal/types/`
6. Implement evaluation in `internal/eval/`
7. Write tests in corresponding `*_test.go` files
8. **Add examples in `examples/`** (REQUIRED!)
9. **Update CHANGELOG.md** (REQUIRED!)
10. **Update README.md** if public-facing (REQUIRED!)

**For detailed contributing guidelines:**
- See [docs/CONTRIBUTING.md](docs/CONTRIBUTING.md) - Full development guide
- See [design_docs/](design_docs/) - Architecture and design decisions

## 📐 Code Organization Principles (AI-First Design)

### File Size Guidelines

**AILANG is designed to be maintained by AI assistants. Keep files small and focused.**

**Target file sizes:**
- **Sweet spot**: 200-500 lines per file
- **Acceptable**: 500-800 lines
- **Problematic**: 800-1200 lines (consider splitting)
- **Critical**: 1200+ lines (MUST split before adding features)

**Why small files matter for AI:**
- Fits in AI context window (I can see the whole file at once)
- Single responsibility principle naturally enforced
- Easy to understand the full structure in one read
- Reduces merge conflicts
- Enables better testing isolation

**Check file sizes:**
```bash
make check-file-sizes    # Fails CI if any file >800 lines
make report-file-sizes   # Shows all files >500 lines
wc -l internal/path/file.go  # Check specific file
```

### Current Technical Debt

**Check current status:**
```bash
make report-file-sizes    # Detailed report of files >500 lines
make codebase-health      # Overall codebase metrics
make largest-files        # Top 20 largest files
```

As of October 2025, ~10 files exceed the 800 line limit (out of 183 total). Run `make report-file-sizes` for the current list.

**Before modifying these files:**
1. Check if splitting is needed first
2. Run tests before/after: `make test`
3. Use the `codebase-organizer` agent for safe refactoring

### File Organization Patterns

#### Pattern 1: One Concept Per File

```
❌ BAD: Everything in one file
internal/parser/parser.go (2518 lines)
  - Expression parsing
  - Statement parsing
  - Type parsing
  - Pattern parsing
  - Module parsing

✅ GOOD: Split by responsibility
internal/parser/
  ├── parser.go (200 lines)         # Main struct, entry points, package docs
  ├── expressions.go (300 lines)    # parseExpression, parseLambda, parseCall
  ├── statements.go (250 lines)     # parseLetDecl, parseFuncDecl, parseType
  ├── types.go (200 lines)          # parseType, parseEffects, parseTypeParams
  ├── patterns.go (280 lines)       # parsePattern, parseConstructor
  ├── modules.go (150 lines)        # parseModule, parseImport, parseExport
  └── helpers.go (140 lines)        # parseParams, parseBlock, utility functions
```

#### Pattern 2: Main File as Table of Contents

Every package should have a main file (usually `pkg.go` or matching package name) that serves as navigation:

```go
// internal/parser/parser.go (200 lines max)
package parser

// Package parser implements AILANG source code parsing.
//
// # Architecture
//
// The parser is split into several files by responsibility:
//   - parser.go: Main Parser struct and entry points (THIS FILE)
//   - expressions.go: Expression parsing (literals, lambdas, calls, etc.)
//   - statements.go: Top-level declarations (func, type, let)
//   - types.go: Type annotation parsing
//   - patterns.go: Pattern matching syntax
//   - modules.go: Module system (import/export)
//
// # Usage
//
//   p := parser.New(lexer)
//   file, err := p.Parse()
//
// # See Also
//
//   - internal/ast: AST node definitions
//   - internal/lexer: Token generation
//   - docs/parser/README.md: Detailed parser documentation

// Parser is the main entry point for parsing AILANG source code.
type Parser struct { /* ... */ }

// Parse parses a complete AILANG source file.
// Implementation delegates to parseFile() in statements.go.
func (p *Parser) Parse() (*ast.File, error) { /* ... */ }
```

#### Pattern 3: Tests Next to Implementation

```
✅ GOOD: Focused test files
internal/parser/
  ├── expressions.go
  ├── expressions_test.go (300 lines focused tests)
  ├── statements.go
  ├── statements_test.go (250 lines focused tests)
  └── integration_test.go (end-to-end tests)

❌ BAD: One giant test file
  └── parser_test.go (5000 lines)
```

#### Pattern 4: Clear File Naming

File names should match the main functions they contain:

```
✅ GOOD:
expressions.go → parseExpression(), parseCall(), parseLambda()
statements.go  → parseLetDecl(), parseFuncDecl(), parseTypeDecl()
patterns.go    → parsePattern(), parseConstructor()

❌ BAD:
parse_stuff.go → everything mixed together
utils.go       → vague, no clear responsibility
```

### Adding New Features (File Size Rules)

**Before adding any new feature to a file:**

```bash
# 1. Check current file size
wc -l internal/types/typechecker_core.go
# Output: 2736 lines

# 2. If >800 lines, STOP and split first
# 3. If 500-800 lines, consider if new feature pushes it over 800
# 4. If <500 lines, proceed normally

# 5. After changes, verify size
wc -l internal/types/typechecker_core.go
make check-file-sizes  # Fails if >800 lines
```

**Splitting workflow:**

```bash
# Option 1: Use the codebase-organizer agent (recommended)
# This agent safely refactors files while ensuring tests pass

# Option 2: Manual split (if you understand the code deeply)
make test                    # Baseline - all tests pass
# ... split files ...
make test                    # Verify - all tests still pass
git add internal/types/*.go
git commit -m "Split typechecker_core.go into 8 files (AI-friendly)"
```

### Package Documentation Standards

Every package with >3 files MUST have a README.md:

```markdown
# internal/parser

Parser for AILANG source code.

## Files

- `parser.go` - Main Parser struct, entry points
- `expressions.go` - Expression parsing: literals, lambdas, calls, operators
- `statements.go` - Declarations: func, type, let, import, export
- `types.go` - Type annotations: simple types, effects, type parameters
- `patterns.go` - Pattern matching: constructors, literals, wildcards, guards
- `modules.go` - Module system: module declarations, import resolution
- `helpers.go` - Shared utilities: parameter parsing, block parsing

## Entry Points

- `Parse()` → `parseFile()` in statements.go
- `parseExpression()` in expressions.go
- `parseType()` in types.go
- `parsePattern()` in patterns.go

## Cross-references

- Consumes: `internal/lexer` (tokens)
- Produces: `internal/ast` (syntax tree)
- Used by: `internal/pipeline`, `internal/repl`
```

### Automated Code Organization

**Use the codebase-organizer agent** for safe refactoring:

The `codebase-organizer` agent is available in `.claude/agents/codebase-organizer.md`. It:
- Monitors file sizes across the codebase
- Identifies files that need splitting
- Safely refactors large files into smaller, focused modules
- Ensures all tests pass before/after refactoring
- Maintains git history and commit hygiene

**Example usage:**
```bash
# Ask Claude to invoke the agent:
"Please use the codebase-organizer agent to check for files that need splitting"

# Or for specific refactoring:
"Use the codebase-organizer agent to split internal/parser/parser.go"
```

### Measuring Success

```bash
# CI checks (automatically run on PRs)
make check-file-sizes     # Fails if any file >800 lines

# Status reports
make report-file-sizes    # Lists all files >500 lines
make codebase-health      # Full codebase metrics
```

**Goal metrics:**
- 0 files over 800 lines ✅
- <5 files between 500-800 lines ⚠️
- Average file size: 300-400 lines 🎯

---

## 🚨 CRITICAL WARNINGS

### Testing Policy
**ALWAYS remove out-of-date tests. No backward compatibility.**
- When architecture changes, delete old tests completely
- Don't maintain legacy test suites
- Write new tests for new implementations
- Keep test suite clean and current

### Linting & "Unused" Code Warnings

**⚠️ LESSON LEARNED: Never blindly delete "unused" functions without understanding WHY they're unused!**

**The Import System Disaster (September 2025)**
In commit `eae08b6`, working import functions were deleted because linter said they were "unused".

**What actually happened:**
1. Function **calls** were renamed from `parseModuleDecl()` to `_parseModuleDecl()` (note underscore)
2. Function **definitions** kept original names (no underscore)
3. Calls were then **commented out**
4. Linter correctly said "hey, `parseModuleDecl` is never called!"
5. Functions were **blindly deleted**
6. Result: **Working import system completely broken** 💥

**Rules to Prevent This:**

1. **NEVER delete functions just because linter says "unused"**
   - First understand WHY they're unused
   - Check git history - were they just commented out?
   - Search entire codebase for references (including comments)
   - Run `make test-imports` and `make test` BEFORE deleting anything

2. **If renaming function calls, rename definitions too**
   - Use IDE refactoring tools, not manual find/replace
   - If adding `_` prefix to mark as TODO, add to BOTH call and definition
   - Better: use TODO comments instead of renaming

3. **Test between each change**
   - Don't combine: rename + comment out + delete
   - Run tests after EACH step:
     - After rename → `make test`
     - After commenting out → `make test-imports`
     - After deleting → `make test && make lint`

4. **When linter complains about unused code:**
   ```bash
   # Step 1: Check if it's really unused
   git log -p --all -S 'functionName' internal/
   grep -r "functionName" internal/

   # Step 2: Check recent changes
   git log --oneline internal/parser/parser.go | head -5
   git diff HEAD~1 internal/parser/parser.go | grep functionName

   # Step 3: If truly unused AND you know why, document it
   git commit -m "Remove unused parseOldFormat() - replaced by parseNewFormat() in commit abc123"
   ```

5. **Special warning for parser/module/import code**
   - These are **critical** for language functionality
   - If you break these, **nothing imports work**
   - Always run `make test-imports` before committing parser changes
   - Check that example files still work: `make verify-examples`

**Recovery Checklist (if this happens again):**
1. Find last working commit: `git log --all --oneline | grep "import"`
2. Check what was deleted: `git diff working_commit broken_commit`
3. Restore deleted functions: `git show working_commit:file.go`
4. Test imports: `make test-imports`
5. Document in commit message what was broken and how it was fixed

### Lexer/Parser Architecture - NEWLINE Tokens Don't Exist!

**⚠️ CRITICAL LESSON: The lexer NEVER generates NEWLINE tokens - it skips them as whitespace!**

**The Multi-line ADT Parser Bug (October 2025)**
While implementing multi-line ADT syntax support, code was written assuming the parser could see NEWLINE tokens:
```go
// ❌ WRONG - This code will never work!
p.skipNewlinesAndComments()  // Tries to skip NEWLINE tokens
if p.curTokenIs(lexer.NEWLINE) {  // This condition is NEVER true!
    ...
}
```

**The Root Cause:**
In `internal/lexer/lexer.go`, the `NextToken()` function calls `skipWhitespace()` which does this:
```go
func (l *Lexer) skipWhitespace() {
    for l.ch == ' ' || l.ch == '\t' || l.ch == '\n' || l.ch == '\r' {
        l.readChar()
    }
}
```

This means `\n` characters are **consumed and never returned as tokens**. Even though `lexer/token.go` defines a NEWLINE token type, the lexer never generates them!

**Implications for Parser Development:**

1. **Never check for NEWLINE tokens**
   ```go
   // ❌ WRONG - lexer skips newlines
   if p.curTokenIs(lexer.NEWLINE) { ... }
   if p.peekTokenIs(lexer.NEWLINE) { ... }

   // ✅ CORRECT - rely on lexer skipping whitespace
   // After RPAREN of Leaf(int), next token is PIPE or TYPE (not NEWLINE)
   if p.curTokenIs(lexer.PIPE) { ... }
   ```

2. **Multi-line syntax "just works"**
   - The lexer automatically handles multi-line constructs
   - For ADTs, after `Leaf(int)` on line 4, the next token is `|` on line 5
   - No need to explicitly skip newlines - the lexer already did it

3. **Token stream example:**
   ```ailang
   type Tree =
     | Leaf(int)
     | Node(Tree, int, Tree)
   ```

   **Token stream the parser sees:**
   ```
   TYPE Tree ASSIGN PIPE Leaf LPAREN int RPAREN PIPE Node LPAREN Tree COMMA int COMMA Tree RPAREN ...
   ```

   **NOT**:
   ```
   TYPE Tree ASSIGN NEWLINE PIPE Leaf LPAREN int RPAREN NEWLINE PIPE Node ...
   ```

4. **When you think you need newline handling:**
   - You probably don't! The lexer handles it
   - Focus on the semantic tokens (PIPE, TYPE, IDENT, etc.)
   - Trust that whitespace (including newlines) is already skipped

5. **If you genuinely need layout-sensitive parsing:**
   - Would require modifying the lexer's `skipWhitespace()` function
   - Would affect the ENTIRE language parsing
   - This is a breaking architectural change - avoid if possible
   - Consider alternative approaches (explicit delimiters, etc.)

**Debugging tip:** If you see unexpected tokens or "skipped too far" issues, check if:
1. You're assuming NEWLINE tokens exist (they don't!)
2. You're calling `skipNewlinesAndComments()` (usually unnecessary)
3. The lexer is already doing what you want (it skips whitespace automatically)

**Files to check if modifying lexer/parser interaction:**
- `internal/lexer/lexer.go` - `NextToken()` and `skipWhitespace()`
- `internal/parser/parser.go` - `nextToken()` wrapper
- Any parser code that checks for or handles whitespace

## Reference Documentation

**For detailed guides, see:**
- **AILANG Syntax**: [prompts/v0.3.8.md](prompts/v0.3.8.md) - Complete teaching prompt
- **REPL Guide**: [docs/guides/repl.md](docs/guides/repl.md) - Interactive development
- **Limitations**: [docs/LIMITATIONS.md](docs/LIMITATIONS.md) - Known issues and workarounds
- **Contributing**: [docs/CONTRIBUTING.md](docs/CONTRIBUTING.md) - Development workflow
- **Design Docs**: [design_docs/](design_docs/) - Architecture decisions
- **Examples**: [examples/](examples/) - 66 example programs

**For architecture details, see:**
- [design_docs/20250926/initial_design.md](design_docs/20250926/initial_design.md) - Original design
- [design_docs/implemented/](design_docs/implemented/) - Completed features
- [design_docs/planned/](design_docs/planned/) - Future work

## Important Notes
1. The language is expression-based - everything returns a value
2. Effects are tracked in the type system - never ignore them
3. Pattern matching must be exhaustive
4. All imports must be explicit
5. Row polymorphism allows extensible records and effects
6. Session types ensure protocol correctness in channels (when implemented)

## Quick Debugging Checklist
- [ ] Check lexer is producing correct tokens
- [ ] Verify parser is building proper AST
- [ ] Ensure all keywords are in the keywords map
- [ ] Confirm precedence levels are correct
- [ ] Check that all AST nodes implement correct interfaces
- [ ] Verify type substitution is working correctly

---

**Remember**: This is a living document. Update it when workflows change, but keep it focused on **actionable instructions** for Claude, not reference material that belongs in docs/.
