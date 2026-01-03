# Trace Debugging Patterns

Common patterns for debugging AILANG using telemetry traces.

## Performance Baselines

Expected timing ranges for normal operations:

### Compiler Pipeline (small file ~50 LOC)

| Phase | Normal | Slow | Investigate |
|-------|--------|------|-------------|
| `compile.parse` | <10ms | 10-50ms | >50ms |
| `compile.elaborate` | <5ms | 5-20ms | >20ms |
| `compile.typecheck` | <20ms | 20-100ms | >100ms |
| `compile.validate` | <5ms | 5-20ms | >20ms |
| `compile.lower` | <10ms | 10-50ms | >50ms |
| **Total** | <50ms | 50-200ms | >200ms |

### Compiler Pipeline (large module ~500 LOC)

| Phase | Normal | Slow | Investigate |
|-------|--------|------|-------------|
| `compile.load` | <50ms | 50-200ms | >200ms |
| `compile.topo_sort` | <10ms | 10-50ms | >50ms |
| `compile.modules` | <500ms | 500ms-2s | >2s |
| **Total** | <1s | 1-3s | >3s |

### AI API Calls

| Provider | Normal | Slow | Timeout |
|----------|--------|------|---------|
| `openai.generate` | 2-10s | 10-30s | >60s |
| `anthropic.generate` | 2-10s | 10-30s | >60s |
| `gemini.generate` | 1-8s | 8-20s | >60s |
| `ollama.generate` | 1-30s | 30-60s | >120s |

## Common Debugging Patterns

### Pattern 1: Slow Compilation

**Symptom:** `compile.*` spans take unexpectedly long

**Investigation:**
```bash
# Run with tracing
GOOGLE_CLOUD_PROJECT=your-project ailang check file.ail

# Check trace
ailang trace list --filter "compile" --hours 1

# View hierarchy
ailang trace view <trace-id>
```

**Common causes:**
1. **`compile.typecheck` slow** → Complex polymorphism, recursive types
2. **`compile.parse` slow** → Very large file, complex expressions
3. **`compile.load` slow** → Many imports, deep dependency graph

### Pattern 2: Compilation Hangs

**Symptom:** Compilation never completes

**Investigation:**
```bash
# Use timeout flag (v0.5.9+)
ailang check --timeout 30s file.ail

# This produces stack dump on timeout, showing where it's stuck
```

**Common causes:**
1. Infinite loop in type unification (cyclic types)
2. Parser stuck in recursive descent
3. Module import cycle

### Pattern 3: Eval Benchmark Slow

**Symptom:** `eval.suite` takes much longer than expected

**Investigation:**
```bash
# Check per-benchmark timing
ailang trace list --filter "eval" --hours 2
ailang trace view <trace-id>

# Look at child spans for which benchmark is slow
```

**Look for:**
- Which `eval.benchmark` spans are longest?
- Is it the AI API call (`*.generate`) or something else?
- Check `benchmark.success` attribute for failures

### Pattern 4: Message Search Slow

**Symptom:** `messages.search` taking too long

**Investigation:**
```bash
ailang trace list --filter "messages.search"
ailang trace view <trace-id>
```

**Check attributes:**
- `search.use_neural` - Neural search is slower (~20-30s)
- `search.result_count` - Many results = larger response
- `search.threshold` - Lower threshold = more candidates

## Trace Attributes Reference

### Resource Attributes (Apply to ALL Spans)

These are set once at telemetry initialization and appear on every span:

```
service.name: string           # e.g., "ailang-compiler", "ailang-eval"
service.version: string        # AILANG version (e.g., "v0.6.3")
deployment.environment: string # "development" or "production"
process.runtime.name: string   # "go"
process.runtime.version: string # Go version
process.cwd: string            # Working directory at init time (critical for debugging!)
```

**The `process.cwd` attribute is especially valuable** for debugging module resolution issues
(see M-BUG-LOCAL-IMPORTS). It shows where the ailang command was invoked from.

### Compiler Spans

```
compile.pipeline / compile.module_pipeline
  file.path: string     # File being compiled
  file.size_bytes: int  # File size
  is_repl: bool         # REPL vs file mode

compile.parse
  ast.nodes: int        # Number of AST nodes parsed

compile.load
  modules.loaded: int   # Number of modules loaded

compile.topo_sort
  modules.sorted: int   # Number of modules in sorted order

compile.modules
  modules.count: int    # Total modules compiled
```

### Eval Spans

```
eval.suite
  eval.models: string        # Comma-separated model list
  eval.benchmarks: string    # Comma-separated benchmark list
  eval.total_runs: int       # Total benchmark runs
  eval.success_count: int    # Successful runs
  eval.fail_count: int       # Failed runs
  eval.success_rate: float   # Success percentage

eval.benchmark
  benchmark.id: string       # Benchmark identifier
  benchmark.model: string    # Model used
  benchmark.language: string # Target language
  benchmark.success: bool    # Pass/fail
  benchmark.duration_ms: int # Total time
  benchmark.input_tokens: int
  benchmark.output_tokens: int
  benchmark.cost_usd: float
```

### Message Spans

```
messages.send
  cwd: string                # Working directory
  message.to_inbox: string
  message.from_agent: string
  message.type: string
  message.id: string

messages.list
  cwd: string                # Working directory
  list.inbox: string
  list.unread_only: bool
  list.limit: int
  list.result_count: int

messages.read
  cwd: string                # Working directory
  message.id: string

messages.search
  cwd: string                # Working directory
  search.query: string
  search.use_neural: bool
  search.threshold: float
  search.result_count: int
```

### AI Provider Spans

```
anthropic.generate / openai.generate / gemini.generate / ollama.generate
  ai.model: string
  ai.tokens_in: int
  ai.tokens_out: int
  ai.tokens_total: int
  ai.cost_usd: float
  http.status_code: int  # API response status
```

## JSON Analysis Patterns

For programmatic analysis:

```bash
# Get all traces as JSON
ailang trace list --hours 2 --json > traces.json

# Extract slow benchmarks (>30s)
cat traces.json | jq '[.[] | select(.duration_ms > 30000)]'

# Find failed benchmarks
ailang trace list --filter "eval.benchmark" --json | \
  jq '[.[] | select(.labels["benchmark.success"] == "false")]'

# Calculate average compilation time
ailang trace list --filter "compile" --json | \
  jq '[.[].duration_ms] | add / length'
```

## Prioritized Future Instrumentation

Based on analysis of implemented design docs and actual bugs encountered:

### Priority 1: Type System Tracing (HIGH VALUE)

**Why:** Multiple P0 bugs traced back to type system issues:
- **M-PERF2**: Cyclic type traversal hang (v0.5.8)
- **Codegen Bug Pattern Analysis**: 4 bugs from TypeName metadata loss (v0.5.10)
- **M-DX11**: Type inference "why is this type X?" questions

**Proposed spans:**

| Span | Description | Key Attributes |
|------|-------------|----------------|
| `types.unify` | Unification of two types | `type.lhs`, `type.rhs`, `result`, `cycles_detected` |
| `types.substitute` | Type substitution | `original`, `substituted`, `typename_preserved` |
| `types.defaulting` | Num defaulting pass | `vars_defaulted`, `constraints` |
| `types.head` | Type head extraction | `input_type`, `depth`, `cycle_risk` |

**Debug value:** Would immediately show:
- Where type inference is stuck (unification loop detection)
- When TypeName metadata is lost (substitution issues)
- Why Num defaulted to Int vs Float

### Priority 2: Module Resolution Tracing (MEDIUM VALUE)

**Why:** Module import bugs are common and confusing:
- **M-BUG-LOCAL-IMPORTS**: Wrong basePath for resolution (v0.4.8)
- **M-TYPE-ALIAS-UNIFICATION**: Cross-module type aliases not found (v0.5.8)
- **Frequent user confusion**: "LDR001: module not found" with no search path info

**Proposed spans:**

| Span | Description | Key Attributes |
|------|-------------|----------------|
| `modules.resolve` | Module path resolution | `import_path`, `base_path`, `search_paths`, `found_path` |
| `modules.load` | Module file loading | `path`, `size_bytes`, `parse_time_ms` |
| `modules.dependency_graph` | Build dependency graph | `modules_count`, `edges_count`, `cycles` |
| `modules.type_imports` | Type import resolution | `imported_types`, `from_module`, `resolved` |

**Debug value:** Would immediately show:
- What paths were searched for an import
- Whether type aliases were properly imported
- Module dependency ordering issues

### Priority 3: Codegen Tracing (MEDIUM VALUE)

**Why:** Codegen fallbacks hide bugs:
- **Codegen Bug Pattern**: Silent fallback to `map[string]interface{}` (v0.5.10)
- **M-CODEGEN-CROSS-MODULE-IMPL**: Wrong function suffix generation
- **DEBUG_CODEGEN=1** already exists but is verbose

**Proposed spans:**

| Span | Description | Key Attributes |
|------|-------------|----------------|
| `codegen.decl` | Declaration code generation | `decl_type`, `name`, `output_type` |
| `codegen.type_lookup` | CoreTypeInfo lookup | `node_id`, `found`, `type_name`, `fallback_used` |
| `codegen.record` | Record type generation | `typed_struct` or `generic_map`, `typename` |

**Debug value:** Would immediately show:
- When fallbacks are triggered (catch issues before runtime)
- Which declarations used generic vs typed codegen
- Type metadata preservation through pipeline

### Priority 4: Pattern Matching Tracing (LOW VALUE - FUTURE)

**Why:** Less frequent but complex when it happens:
- **M-P5 Pattern Matching**: Pattern matching in function bodies (v0.2.0)
- **Nullary Constructor Bug**: Pattern matching ADT detection (v0.4.5)
- Decision tree construction is opaque

**Proposed spans:**

| Span | Description | Key Attributes |
|------|-------------|----------------|
| `match.compile` | Decision tree construction | `patterns_count`, `depth`, `exhaustive` |
| `match.coverage` | Coverage analysis | `missing_cases`, `redundant_cases` |

### Implementation Order

Based on bug frequency and debugging time saved:

1. **types.unify + types.substitute** (4+ hours debugging saved per incident)
2. **modules.resolve** (1-2 hours saved per incident, common user issue)
3. **codegen.type_lookup** (catches issues before Go compile)
4. **match.compile** (rare but complex when needed)

## When to Add More Traces

Consider adding instrumentation when:

1. **Repeated debugging of same area** - If you debug the same component multiple times, add tracing
2. **Unexplained slowness** - If timing breakdowns don't explain the issue, add finer-grained spans
3. **Complex control flow** - Areas with many conditional paths benefit from tracing
4. **Cross-component interactions** - When debugging involves multiple packages
5. **Silent fallbacks** - Any code path that silently degrades should emit spans

### How to Add Spans

```go
import "go.opentelemetry.io/otel"

var tracer = otel.Tracer("ailang.component")

func myFunction(ctx context.Context) {
    ctx, span := tracer.Start(ctx, "component.operation")
    defer span.End()

    // Add attributes
    span.SetAttributes(
        attribute.String("input.file", filename),
        attribute.Int("input.size", len(data)),
    )

    // ... work ...

    // Record result
    span.SetAttributes(attribute.Bool("success", true))
}
```

### Cycle-Safe Type Tracing Pattern

For type system spans (learned from M-PERF2):

```go
func safeTypeSpan(ctx context.Context, t types.Type) {
    ctx, span := tracer.Start(ctx, "types.operation")
    defer span.End()

    // NEVER call t.String() directly - use shallow head only
    head := shallowHead(t)  // O(1), no recursion
    span.SetAttributes(attribute.String("type.head", head))

    // Add cycle detection attribute
    if hasCycleRisk(t) {
        span.SetAttributes(attribute.Bool("type.cyclic", true))
    }
}

// shallowHead - O(1) type tag extraction, cycle-safe
func shallowHead(t types.Type) string {
    switch v := t.(type) {
    case *types.TCon:
        return v.Name
    case *types.TApp:
        return shallowHead(v.Constructor) + "[...]"
    case *types.TVar:
        return "α" + strconv.Itoa(v.ID)
    default:
        return reflect.TypeOf(t).String()
    }
}
```

## Tracing Scope: Tooling vs Generated Code

**Important:** Current OTEL tracing only covers AILANG tooling operations (compile, eval, messages). It does NOT trace the generated Go code at runtime.

### What IS Traced (AILANG Tooling)

```
User runs: ailang compile foo.ail
           ↓
    ┌──────────────────────────────────────────┐
    │  TRACED: compile.pipeline                │
    │    ├── compile.parse                     │
    │    ├── compile.elaborate                 │
    │    ├── compile.typecheck  ← bugs hide   │
    │    ├── compile.validate                  │
    │    └── compile.lower                     │
    └──────────────────────────────────────────┘
           ↓
    Generated: foo_generated.go
           ↓
    User runs: go build && ./foo
           ↓
    ┌──────────────────────────────────────────┐
    │  NOT TRACED: Generated Go runtime        │
    │    (unless you add tracing manually)     │
    └──────────────────────────────────────────┘
```

### Debugging Generated Go Code

For issues in the generated Go (runtime panics, wrong output):

1. **`DEBUG_CODEGEN=1`** - See exactly what Go code is generated
2. **`go tool pprof`** - Standard Go profiling
3. **Add prints to codegen templates** - `internal/codegen/templates/`
4. **Manual OTEL in generated code** - Modify templates to emit spans

### Future: Runtime Tracing in Generated Code

If we wanted runtime tracing, we'd need to:

1. **Add OTEL to codegen templates:**
   ```go
   // In internal/codegen/templates/function.go.tmpl
   func {{.Name}}({{.Params}}) {{.ReturnType}} {
       ctx, span := tracer.Start(ctx, "ailang.{{.ModuleName}}.{{.Name}}")
       defer span.End()
       // ... generated body ...
   }
   ```

2. **Thread context through:**
   - All functions would need `ctx context.Context` parameter
   - Breaks current pure functional model

3. **Emit spans for:**
   - Function calls
   - Pattern match branches taken
   - Effect invocations

**Trade-off:** Runtime tracing adds overhead and complexity. Current approach (trace compilation, not execution) is lightweight.
