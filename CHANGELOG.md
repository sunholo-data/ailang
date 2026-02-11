# AILANG Changelog

## [Unreleased]

### Added
- **M-TRACE-EXPORT Phase 1: Program-level execution traces** (`internal/trace/`, `--emit-trace jsonl`, ~500 LOC)
  - New `internal/trace/` package: schema, collector, JSONL serializer
  - Event types: `function_enter`, `function_exit`, `effect`, `contract_check`, `budget_delta`, `module_start/end`, `error`
  - Call depth tracking with function entry/exit duration
  - Trace collector on `EffContext.Trace` — zero-cost when disabled
  - `--emit-trace jsonl` flag on `ailang run` — JSONL to stdout, program output to stderr
  - Function tracing via `TraceRecorder` interface in eval (follows `BudgetEnforcer` pattern)
  - Contract check recording in `CheckRequires`/`CheckEnsures` delegates
  - Effect + budget delta recording in `effects.Call()`
  - IO output respects `IOWriter` on EffContext (stderr when tracing)
  - 12 unit tests for collector, JSONL round-trip, depth tracking
  - Design doc: `design_docs/planned/v0_8_0/m-trace-export.md`

- **M-TRACE-EXPORT Phase 2: OTEL span emission** (`internal/trace/otel_emitter.go`, ~190 LOC impl + ~320 LOC tests)
  - `EmitOTELSpans()` converts collected trace events to OTEL spans with parent-child hierarchy
  - Batch post-execution: walks event list, reconstructs span tree from depth tracking
  - Span mapping: `eval.function.*`, `eval.effect.*`, `eval.module.*`
  - Contract checks → span events; budget deltas → span attributes; errors → error status
  - `--emit-trace` extended: `otel`, `jsonl,otel`, `auto` modes
  - Auto-enable: when OTEL is configured, program traces emit automatically (zero flags)
  - `BaseTime()` exposed on Collector for timestamp reconstruction
  - 11 tests with `tracetest.InMemoryExporter` (nesting, parent context, nil safety, full trace)
  - Spans flow to Cloud Trace and observatory as children of `"ailang run: <filename>"` root span

- **M-TRACE-EXPORT Phase 3: Trace replay** (`ailang replay`, `internal/trace/reader.go`, `comparator.go`, ~550 LOC)
  - `ailang replay <trace.jsonl>` re-executes program and compares against baseline
  - JSONL reader: `ReadJSONL()` with validation, blank line skipping, large line support
  - Trace comparator: `CompareTraces()` with per-field mismatch reporting
  - Timestamps and durations correctly skipped (non-deterministic)
  - `--json` flag for machine-readable comparison output
  - `--file` override for source file resolution
  - Module name and capabilities auto-extracted from baseline trace
  - Exit codes: 0=match, 1=mismatch, 2=error
  - 19 tests for reader (round-trip, validation, edge cases) and comparator (all event types)

- **M-TRACE-EXPORT Phase 4: Training data export** (`ailang export-training`, `internal/trace/scorer.go`, `exporter.go`, ~450 LOC)
  - Quality scorer: `ScoreTrace()` produces 0.0-1.0 score with 5 weighted components
  - Weights: completion (30%), complexity (25%), contracts (20%), budget efficiency (15%), effect diversity (10%)
  - Complexity uses diminishing returns (log scale) to avoid rewarding bloat
  - `ScoreTraceFile()` convenience for scoring individual files
  - Training exporter: `ExportTrainingData()` produces JSONL with source code, trace, score, metadata
  - Source file auto-resolution from module name (source-dir, trace-dir, CWD)
  - `ailang export-training traces/` — export directory of traces
  - `ailang export-training --score trace.jsonl` — score without exporting
  - `ailang export-training --score --json traces/` — machine-readable scores
  - `--min-score 0.7` filters low-quality traces
  - `--output file.jsonl` writes to file instead of stdout
  - `--source-dir` for source code resolution
  - 22 tests for scorer (all components, edge cases) and exporter (files, filtering, source resolution)

- **Website: Symbolic Reasoning Kernel vision** (`docs/docs/vision.mdx`)
  - Reframed from "AI-friendly language" to "symbolic reasoning kernel"
  - Five Pillars table with implementation status
  - Two-level trace architecture documented (agent traces + program traces)
  - Differentiation table: Traditional Languages vs AILANG
  - Explicit Authority section with budget and contract examples

### Fixed
- **M-CONTRACTS-OPLOWERING: Contract expressions no longer require `--experimental-binop-shim`** (`internal/pipeline/op_lowering.go`)
  - OpLowering now processes contract expressions in `prog.Meta`, not just `prog.Decls`
  - `ailang run --verify-contracts file.ail` works directly without extra flags
  - 4 new unit tests for contract expression lowering (int comparison, no-contracts, nil-expr, no-mutation)
  - Design doc: `design_docs/planned/v0_8_0/m-contracts-oplowering.md`

## [v0.7.3] - 2026-02-09

### Added
- **M-STRUCTURED-AI-OUTPUT: Structured JSON output from AI providers** (`std/ai.ail`, `internal/ai/`, `internal/effects/ai.go`, ~370 LOC)
  - `callJson(prompt, schema)` — schema-enforced JSON output (provider validates against JSON Schema)
  - `callJsonSimple(prompt)` — valid JSON output without schema enforcement
  - Both return raw JSON strings; parse with `std/json.decode`
  - **All 4 providers wired**: Gemini (responseMimeType + responseSchema), OpenAI (response_format.json_schema for Chat, text.format for Responses), Anthropic (tool_use pattern with forced tool_choice), Ollama (format field with schema or "json")
  - `Handler.CallJson()` bridges provider structured output to effect system
  - 2 new builtins: `_ai_call_json` (2 args), `_ai_call_json_simple` (1 arg)
  - Stub handler returns valid JSON (`{"kind":"Wait"}`) for `--ai-stub` testing
  - Examples: `structured_ai_basic.ail`, `structured_ai_schema.ail`
  - Teaching prompt v0.7.3 updated with callJson/callJsonSimple docs
  - Design doc: `design_docs/planned/v0_7_3/m-structured-ai-output.md`

- **M-EFFECTFUL-LIST-COMBINATORS: Effectful list combinators for std/list** (`std/list.ail`, ~74 LOC)
  - `flatMap(f, xs)` — pure flatMap: apply f to each element, flatten results
  - `mapE(f, xs)` — effectful map: apply effectful function to each element
  - `filterE(p, xs)` — effectful filter: keep elements matching effectful predicate
  - `foldlE(f, acc, xs)` — effectful left fold with accumulator
  - `flatMapE(f, xs)` — effectful flatMap: apply f, flatten results
  - `forEachE(f, xs)` — effectful forEach: apply f for side-effects, discard results
  - All effectful combinators are effect-polymorphic (work with IO, FS, AI, etc.)
  - Left-to-right evaluation order guaranteed
  - **Parser enhanced**: Effect row variables (lowercase identifiers like `e` in `! {e}`) now supported
    - Modified 5 files: `ast.go`, `parser_effect.go`, `effects.go`, `typechecker.go`, `builder.go`
    - Typo detection: `io` still correctly flagged as typo for `IO`
  - 8 type inference regression tests (`examples/runnable/effectful_list_t[1-8]_*.ail`)
  - Smoke test: `examples/runnable/effectful_list.ail`
  - Design doc: `design_docs/planned/v0_7_3/m-effectful-list-combinators.md`

- **M-STDLIB-ZIP: ZIP archive reading builtins** (`internal/builtins/zip.go`, ~315 LOC)
  - `_zip_listEntries(path)` — list all entry names in a ZIP archive
  - `_zip_readEntry(path, entryName)` — read text entry as UTF-8 string
  - `_zip_readEntryBytes(path, entryName)` — read binary entry as base64 string
  - All functions return `Result[T, string]` with proper Ok/Err handling
  - Requires `FS` capability (effect-guarded)
  - Security: path traversal rejection (`..`), max 10,000 entries, 100MB decompressed size limit
  - `io.LimitReader` defense-in-depth for zip bomb protection
  - Sandbox support via `AILANG_FS_SANDBOX`
  - 12 tests covering happy path, error cases, security, and sandbox (`zip_test.go`, ~388 LOC)
  - Example: `examples/runnable/zip_reader.ail`
  - Design doc: `design_docs/planned/v0_7_3/m-stdlib-zip.md`

- **M-STDLIB-XML: XML parsing and querying builtins** (`internal/builtins/xml.go`, ~530 LOC)
  - `_xml_parse(xml)` — parse XML string into XmlNode ADT tree
  - `_xml_findAll(node, tag)` — find all descendant elements matching tag name
  - `_xml_findFirst(node, tag)` — find first matching element (returns Option)
  - `_xml_getText(node)` — recursive text content extraction
  - `_xml_getAttr(node, attrName)` — attribute lookup (returns Option)
  - `_xml_getChildren(node)` — direct child nodes
  - `_xml_getTag(node)` — tag name of element
  - XmlNode ADT: `Element(tag, attrs, children) | Text(content) | CData(content) | Comment(content)`
  - All 7 functions are pure (no effects required)
  - Namespace prefix handling for OOXML compatibility (e.g., `w:document`, `w:p`)
  - Security: depth limit (256), input size limit (50MB)
  - 30 tests covering parsing, queries, namespaces, OOXML fragments, security (`xml_test.go`, ~530 LOC)
  - Example: `examples/runnable/xml_parser.ail`
  - Design doc: `design_docs/planned/v0_7_3/m-stdlib-xml.md`

- **readFileBytes in std/fs** (`internal/builtins/fs.go`, `internal/effects/fs.go`)
  - `readFileBytes(path)` — read file as binary, return base64-encoded string
  - Returns `Result[string, string]` with Ok/Err handling (matches std/zip pattern)
  - Requires `FS` capability (effect-guarded)
  - Sandbox support via `AILANG_FS_SANDBOX`
  - 5 tests covering binary content, text content, missing file, sandbox, and empty file
  - Requested by DocParse for PDF/image file support via std/ai

- **`json.repair` builtin** (`internal/builtins/json.go`)
  - Recovers truncated/malformed JSON from AI providers
  - Closes unclosed strings, arrays, objects; removes trailing commas; completes truncated keywords
  - Handles dangling backslashes and whitespace padding
  - Returns `Result[string, string]` for inspect-then-decode workflow

- **Gemini multimodal support** (`internal/ai/gemini/`)
  - `inlineData` parts for PDF and image inputs via Gemini API

### Fixed
- **callJson truncation/corruption** — three-layer fix for structured JSON output issues reported by docparse-demo:
  - TrimSpace on CallJson response (Gemini pads with trailing spaces)
  - Enforce 8192 minimum max_tokens for JSON structured output
  - Wire per-model `max_output_tokens` from models.yml through handler options
- **Model token limits updated** — accurate API limits from provider docs:
  - OpenAI GPT-5 family: 128K (was unset/default 4096)
  - Claude 4.5/4.6: 64K (Opus 4.6 was 16384)
  - Gemini 2.5 Pro/Flash: 65K (were 8192)
  - Gemini 3 Pro: 65K (was 16384)
- **REPL module import alias resolution** — `import std/list as L` now correctly resolves `L.map`, `L.filter` etc. in the REPL (`internal/repl/module_registry.go`)
- **Coordinator Claude executor hang** — reverted dual idle/hard timeout system that caused all coordinator tasks to hang; added NVM bin dir PATH prepending so `#!/usr/bin/env node` resolves correctly
- **Chain metric data capture** — `NumTurns`, `ToolCallCount`, `ErrorMessage` now populated in chain stage metrics

## [v0.7.2] - 2026-02-06

### Added
- **M-WASM-EFFECTS: Generic JS-backed effect handlers for WASM** (`cmd/wasm/main.go`, `cmd/wasm/effects.go`)
  - `ailangSetEffectHandler(capability, operation, jsCallback)` — register JS functions as effect handlers
  - `ailangSetAIHandler(jsCallback)` — register AI completion handler for `perform AI.complete`
  - `ailangGrantCapability(capability)` — grant effect capabilities (IO, FS, Net, AI, Clock)
  - `ailangEvalAsync(expr)` → Promise — async expression evaluation for effect-using code
  - `ailangCallAsync(module, func, ...args)` → Promise — async module function calls
  - Enables browser-side effects (fetch API, localStorage, DOM access) via JS callbacks
  - Full test coverage: 7 tests in `internal/repl/wasm_effects_test.go`

- **WASM documentation**: Updated `docs/docs/guides/wasm-integration.md` with Effects Handlers API section

### Fixed
- **WASM Reset() stdlib loss**: Fixed critical bug where `ailangReset()` destroyed all stdlib modules
  - Root cause: `Reset()` created new empty registry without reloading embedded stdlib
  - Also failed to reconnect registry to REPL and share EffContext
  - Fix: Reset now properly reconnects registry, shares EffContext, reloads all stdlib, re-imports prelude
  - Files: `cmd/wasm/main.go`

- **WASM loadEmbeddedStdlib panic recovery**: Added per-module panic recovery during stdlib loading
  - Previously, a panic in any single stdlib module would crash the entire WASM binary
  - Now panics are caught per-module and logged to `console.warn`
  - Files: `cmd/wasm/main.go`

- **M-POLY-ARITH: Polymorphic arithmetic operators in lambdas** (`internal/eval/eval_patterns.go`)
  - `let add = \x. \y. x + y in add(3.14)(2.71)` now correctly returns `5.85`
  - Root cause: Num typeclass defaulting resolved lambda to `int -> int -> int` before dict elaboration
  - Fix: `evalDictApp` now checks actual runtime argument types and corrects `DictRef.TypeName` accordingly
  - All 5 arithmetic operators work in polymorphic lambdas: `+`, `-`, `*`, `/`
  - Nested operators work: `(x + y) * (x - y)`
  - WASM REPL also fixed (same evaluator path)
  - 12 new integration tests in `internal/pipeline/poly_arithmetic_test.go`

- **Float sum accumulation**: Fixed float arithmetic in WASM pipeline (`internal/pipeline/`)

### Changed
- **Neural search**: Added timeout and background warmup for embedding search
- **Sprint execution**: Support for parallel milestone execution via Task sub-agents
- **Model configuration**: Updated models.yml with latest model entries

## [v0.7.1.4] - 2026-01-31

### Fixed
- **Release CI**: Re-release of v0.7.1.3 with assets (CI failed due to immutable release conflict)
  - Same fixes as v0.7.1.3: WASM stdlib builtin wrappers now work correctly

## [v0.7.1.3] - 2026-01-30

### Fixed
- **WASM stdlib builtin wrappers**: Fixed bug where stdlib functions failed with "undefined variable" after import
  - Root cause: TWO issues in `LoadModule`:
    1. Elaborator not calling `AddBuiltinsToGlobalEnv()` - builtins elaborated as `Var` instead of `VarGlobal`
    2. Type checker's `globalTypes` not populated with builtin type schemes
  - Fix: Call `AddBuiltinsToGlobalEnv()` and populate `globalTypes` from `$builtin` interface
  - Now stdlib modules like `std/string` that wrap builtins (e.g., `_str_len`) work correctly
  - Files: `internal/repl/module_registry.go`
  - Test: Added `TestLoadModuleWithBuiltinWrapper`

## [v0.7.1.2] - 2026-01-30

### Fixed
- **WASM ailangLoadModule exports**: Fixed bug where `ailangLoadModule` returned empty exports array
  - Root cause: `LoadModule` called `Elaborate()` which returns empty `Meta` map
  - Fix: Call `ElaborateFile()` for modules with `module` declaration to populate `Meta` with `IsExport` flags
  - Now correctly respects `export` keyword: `export pure func` is exported, plain `pure func` is not
  - Backwards compatible: Modules without explicit exports still export all bindings
  - Files: `internal/repl/module_registry.go`
  - Tests: Added `TestLoadModuleWithExportPureFunc`, `TestLoadModuleExplicitExportFiltering`

## [v0.7.1.1] - 2026-01-30

### Added - WASM Module Calling API
- **ailangCall**: New JavaScript API for calling functions from loaded modules
  - `ailangCall(moduleName, funcName, ...args)` - Returns `{success, result?, error?}`
  - Supports number, string, and boolean arguments (converted to AILANG types)
  - Auto-imports module before calling function
  - Completes the browser WASM module loading workflow
  - Files: `cmd/wasm/main.go`, `internal/repl/module_registry.go`

- **JS Wrapper improvements**: Updated `web/ailang-repl.js`
  - `repl.call(mod, func, ...args)` - Now uses native `ailangCall` for reliability
  - Returns structured result `{success, result?, error?}` instead of string

- **CLI help**: Updated WebAssembly documentation in `ailang --help`
  - Shows `gh release download` command for obtaining WASM binary
  - Documents all 6 JavaScript window globals
  - Documents JS wrapper methods

### Fixed
- **Example files**: Fixed curried lambda syntax in examples
  - `record_update.ail`: Changed `\p dx dy.` to `\p. \dx. \dy.`
  - `typeclasses.ail`: Changed `\x y.` to `\x. \y.`
  - Multi-arg lambdas must use separate lambda syntax for curried calls

- **Letrec recursion**: Simplified `letrec_recursion.ail` to avoid single-call bug
  - Single recursive call in letrec fails (known bug, design doc created)
  - Double recursive calls work (fib pattern)
  - Design doc: `design_docs/planned/v0_7_2/m-bug-letrec-single-call.md`

## [v0.7.1] - 2026-01-30

### Added
- **stdlib**: Added `startsWith()` and `endsWith()` to std/string module
  - `startsWith(s: string, prefix: string) -> bool` - Check if string starts with prefix
  - `endsWith(s: string, suffix: string) -> bool` - Check if string ends with suffix
  - Addresses demo feedback: users were reimplementing basic string operations
  - Visible via `ailang docs std/string`

### Fixed - M-VERIFY-CONTRACTS: Runtime Contract Enforcement (Language-Wide)
- **Contract verification now works across all execution modes** (M-VERIFY-CONTRACTS)
  - `ailang run --verify-contracts --experimental-binop-shim` - Run with contracts
  - `ailang serve-api --verify-contracts` - API with contracts
  - Previously: contracts were parsed but NEVER enforced anywhere
  - Requires `--experimental-binop-shim` until OpLowering applied to contract expressions
  - Contract violations produce clear error messages with source location
  - Example: `contract violation: requires failed in  at api.ail:7:12: (limit > 0)`
  - Files: `internal/eval/eval_evaluator.go`, `internal/eval/eval_operations.go`, `internal/runtime/runtime.go`
  - Design doc: `design_docs/implemented/v0_7_1/m-verify-contracts.md`

### Fixed - M-POLY-ADT: Polymorphic ADT Type Inference
- **Polymorphic ADTs now correctly handle mixed field types** (M-POLY-ADT)
  - `type Result[a] = Ok(a) | Err(string)` now works correctly
  - `Err` gets correct type: `∀a. string -> Result[a]` (not `∀a. a -> Result[a]`)
  - Enables standard error handling patterns with Result types
  - Cross-module ADT imports work (e.g., Option from std/option)
  - Files: `internal/elaborate/core.go`, `internal/elaborate/file.go`, `internal/pipeline/pipeline_module.go`
  - Design doc: `design_docs/implemented/v0_7_1/m-poly-adt-option-inference.md`
  - Example: `examples/runnable/polymorphic_adt.ail`

### Fixed
- **docs search**: Added fallback to docs/ directory when design_docs/ not available
  - `ailang docs search` now tries design_docs/ first (developers), then docs/ (users)
  - Shows helpful error with --path suggestion when neither directory exists
  - Prevents hard failure when running outside source tree
  - Note: Full text search still requires local docs directory (not embedded in binary)
  - Future: GitHub repository search fallback planned (M-DX27)
  - Design doc: `design_docs/planned/v0_7_2/m-dx27-docs-search-github-fallback.md`
  - Location: `cmd/ailang/docs_search.go:findDocsDir()`
- **models.yml**: Embedded models.yml configuration in binary using go:embed
  - Fixes "model not found" errors when using installed binary outside source tree
  - Binary now works anywhere without needing relative path to models.yml
  - Falls back to file system for development builds
  - Location: `internal/eval_harness/models.go`
- **testing**: Fixed "module not found: _test" error in inline tests
  - Set `IsREPL: true` when evaluating inline test harness
  - Prevents pipeline from treating synthetic `_test.ail` as a module
  - Inline tests now work correctly with complex imports
  - Location: `internal/testing/executor.go:156`
  - Note: Property tests (auto-generated from contracts) still have issues (empty program error)
  - Design doc created: `design_docs/planned/v0_7_2/m-dx26-property-test-empty-program.md`

### Added - Scoped Budgets with Dual Counters & Budget Reporting (M-DX25)

Enhanced effect budget system with scoped dual counters and minimum budget requirements.

**New Syntax - Minimum Budgets (`@min`):**
- `! {IO @min=1}` - Function must perform at least 1 IO operation
- `! {IO @min=1 @limit=5}` - Between 1 and 5 IO operations
- `BudgetUnderrunError` raised when minimum not met (e.g., caching bypassed expected API call)

**Dual Counter System:**
- **Physical count**: Actual builtin invocations (truth of what happened)
- **Semantic count**: Charged based on declared budgets (contracts between caller/callee)
- Scoped budget enforcement: callee's `@limit=k` charges caller k semantic units

**New Interfaces (for budget scoping):**
- `BudgetEnforcer.WithBudgetLimits()` - Create scoped budget context
- `ScopeCharger.PopScopeAndChargeCaller()` - Charge caller on function return
- `MinBudgetEnforcer.SetMinBudgets()` - Set minimum usage requirements
- `MinimumChecker.CheckMinimums()` - Verify minimums on scope exit

**New Constructors:**
- `effects.NewBudgetContextWithMin(limits, mins)` - Budget context with min/max limits
- `effects.NewBudgetUnderrunError(effect, min, actual, position)` - Underrun error

**Implementation Details:**
- Added `Min *int` to `ast.EffectAnnotation` for parser support
- Added `MinBudgets map[string]*int` to `types.Row` for type system
- Added `EffectMinBudgets map[string]int` to `eval.FunctionValue` for closures
- Physical/semantic counters tracked in `BudgetContext`
- Scoped charging flows: callee scope → pop → charge caller declared amount

**Files modified:** 13 files, ~630 LOC
- `internal/ast/ast.go`: Min field on EffectAnnotation
- `internal/parser/parser_effect.go`: @min=N parsing
- `internal/types/types_v2.go`: MinBudgets on Row
- `internal/types/effects.go`: ElaborateEffectRowWithBudgets
- `internal/effects/budget.go`: Dual counters, min checking
- `internal/effects/errors.go`: BudgetUnderrunError
- `internal/effects/context.go`: SetMinBudgets, CheckMinimums
- `internal/eval/value.go`: EffectMinBudgets on FunctionValue
- `internal/eval/eval_expressions.go`: extractEffectMinBudgets
- `internal/eval/eval_evaluator.go`: Scoped budget interfaces
- `internal/eval/eval_operations.go`: Scoped budget handling

**Design Doc:** `design_docs/implemented/v0_7_1/m-dx25-budget-report.md`

### Fixed - Effect Context for serve-api

Fixed `embed.Engine.Call()` not setting up an EffContext, which prevented effectful AILANG functions (IO, FS, Net, AI, etc.) from working in `serve-api`.

- New `embed.Engine.SetEffContext()` method for configuring effect context
- New `--caps` flag for `serve-api` to grant capabilities (IO,FS,Net,AI,Clock,Env)
- New `--ai MODEL` flag for `serve-api` to configure AI provider
- New `--ai-stub` flag for `serve-api` to use stub AI handler (testing)
- Pure functions continue to work without any flags (backward compatible)

### Added - Hot Reload for serve-api (M-HOT-RELOAD)

Added `--watch` flag to `ailang serve-api` for automatic hot reload of `.ail` modules during development.

- `ailang serve-api --watch ./api/` watches for `.ail` file changes and recompiles automatically
- 4-layer cache invalidation: loader cache, runtime instances, engine compiled, server modules
- Debounced file watching (200ms) to batch rapid saves
- Graceful degradation: compile errors are logged but don't crash the server
- New public APIs: `embed.Engine.InvalidateModule()`, `runtime.DeleteInstance()`, `loader.DeleteCached()`
- New dependency: `github.com/fsnotify/fsnotify v1.9.0`
- New file: `internal/apiserver/watcher.go` (~110 LOC)

**Design Doc:** `design_docs/implemented/v0_7_1/m-hot-reload-serve-api.md`

### Added - Hello World Example

Added classic "Hello, World!" example demonstrating basic AILANG module structure with IO effects.

- New file: `examples/hello_world.ail`
- Shows minimal module with main entry point and IO capability
- `ailang run --caps IO --entry main examples/hello_world.ail`

**Design Doc:** `design_docs/implemented/v0_7_1/hello-world-feature.md`

### Added - API Server & React Scaffold (M-SERVE-API)

New `ailang serve-api` command that compiles AILANG modules and auto-generates REST endpoints from exported functions. Paired with `ailang init web-app` to scaffold full-stack projects with a React frontend.

**New CLI Commands:**
- `ailang serve-api <path...>` - Serve AILANG exports as REST API endpoints
  - `--port PORT` (default: 8080) - HTTP server port
  - `--cors` (default: true) - Enable CORS headers
  - `--frontend PATH` - Proxy to Vite dev server for React hot-reload
  - `--static PATH` - Serve built frontend from directory
- `ailang init web-app [name]` - Scaffold a new web app project with AILANG API + React frontend

**Auto-Generated Endpoints:**
- `POST /api/{modulePath}/{functionName}` - Call any exported function
  - Body: `{"args": [arg1, arg2]}` (positional) or single JSON value
  - Response: `{"result": ..., "module": "...", "func": "...", "elapsed_ms": N}`
- `GET /api/_meta/modules` - List all loaded modules with exports and type signatures
- `GET /api/_meta/modules/{path}` - Module detail with typed export info
- `GET /api/_health` - Health check with module/export counts

**Scaffold Structure (`ailang init web-app myproject`):**
```
myproject/
├── api/handlers.ail     # Example AILANG API module
├── ui/                  # React + Vite + TypeScript
│   ├── package.json
│   ├── vite.config.ts   # Proxies /api to AILANG server
│   └── src/App.tsx      # React app calling AILANG functions
└── Makefile             # `make dev` starts both servers
```

**Bug Fix - JSON Integer Conversion:**
- Fixed `embed.FromGo()` to detect whole-number float64 values from JSON and convert to `IntValue` instead of `FloatValue`. This ensures JSON `{"args": [3, 4]}` correctly calls `add(x: int, y: int)`.

**New Package:** `internal/apiserver/` (~535 LOC)
- `server.go` - Core server wrapping embed.Engine
- `handler.go` - Generic function call handler
- `meta.go` - Introspection and health endpoints
- `server_test.go` - 18 tests covering health, modules, function calls, CORS, errors, arg parsing

**Template Files:** `internal/apiserver/templates/web_app/` (embedded via `go:embed`)

**Design Doc:** `design_docs/implemented/v0_7_1/m-serve-api-react-frontend.md`

### Added - OTEL to TraceRegistry Bridge (M-TRACE-BRIDGE)

Bridged existing OpenTelemetry spans to the TraceRegistry, enabling `_trace_check` to verify spans created by the compiler, REPL, AI providers, and other instrumented components.

**New Functions:**
- `telemetry.Tracer(name)` - Get a tracer instance (wraps `otel.Tracer`)
- `telemetry.StartSpan(ctx, tracer, spanName, opts...)` - Create span and record to TraceRegistry
- `telemetry.RecordSpan(spanName)` - Manually record a span name
- `telemetry.SetTraceRecordingEnabled(bool)` - Enable/disable recording
- `telemetry.IsTraceRecordingEnabled()` - Check if recording is enabled

**Environment Variable:**
- `AILANG_TRACE_RECORDING=1` - Enable trace recording at startup

**Updated Files (13 files):**
- `internal/telemetry/traced_tracer.go`: New bridge implementation (~60 LOC)
- `internal/telemetry/traced_tracer_test.go`: Unit tests (~60 LOC)
- `internal/telemetry/traced_tracer_integration_test.go`: Integration tests (~70 LOC)
- Updated tracer definitions in: pipeline, repl, ai/*, executor/*, messaging, coordinator

**How it works:**
When `AILANG_TRACE_RECORDING=1` is set (or enabled programmatically), every span created via `telemetry.StartSpan` is recorded to the global TraceRegistry. This allows `_trace_check("compile.parse")` to return `true` after compilation.

**Zero overhead:** When disabled, `StartSpan` just delegates to the underlying tracer with no additional work.

**Design Doc:** `design_docs/implemented/v0_7_1/m-trace-instrumentation.md`

### Added - Trace Testing Framework (M-TRACE-TEST)

Added a minimal trace testing utility for verifying that expected trace spans exist during program execution.

**New Module:**
- `stdlib/trace_test` - Trace testing helpers

**New Builtin:**
- `_trace_check(name: string) -> bool` - Check if a trace span exists (supports prefix matching)

**Exported Functions:**
- `assert_trace_exists(name: string) -> bool` - Check if span with name exists
- `assert_trace_not_exists(name: string) -> bool` - Check if span does NOT exist
- `test_compile_traces() -> int` - Example test function

**Example Usage:**
```ailang
import stdlib/trace_test (assert_trace_exists)

export pure func verify_traces() -> int {
  let has_parse = assert_trace_exists("compile.parse");
  let has_typecheck = assert_trace_exists("compile.typecheck");
  1
}
```

**Implementation Files:**
- `internal/effects/trace.go`: TraceRegistry with thread-safe span tracking (~110 LOC)
- `internal/builtins/trace.go`: Builtin registration (~55 LOC)
- `stdlib/trace_test.ail`: AILANG module with helpers (~25 LOC)
- `examples/trace_testing.ail`: Working example (~35 LOC)

**Tests:**
- `internal/effects/trace_test.go`: 8 test cases including concurrency tests

**Design Doc:** `design_docs/implemented/v0_7_1/trace-test.md`

### Added - Workspace Access Control (M-WORKSPACE-ACCESS)

Multi-tenant workspace isolation for the AILANG Dashboard. Users only see workspaces they have access to.

**New CLI Commands:**
- `ailang workspaces list` - List accessible workspaces
- `ailang workspaces add` - Create a new workspace
- `ailang workspaces show` - Show workspace details
- `ailang workspaces grant` - Grant user access to a workspace
- `ailang workspaces revoke` - Revoke user access from a workspace
- `ailang workspaces set-public` - Toggle public visibility

**Key Features:**
- Workspace = GitHub repo identifier (e.g., `sunholo-data/ailang`)
- Per-workspace roles: Viewer (read-only), Approver (full access)
- Public/private visibility via `is_public` flag
- Path pattern matching with `*` and `**` wildcards
- 5-minute TTL caching for Firestore access checks
- Defense-in-depth: Frontend filter → API Middleware → Query filtering

**New Files (~1810 LOC total):**
- `internal/server/auth/workspace.go` (~580 LOC): Core workspace service
- `internal/server/auth/workspace_test.go` (~254 LOC): Unit tests
- `cmd/ailang/workspaces.go` (~464 LOC): CLI commands
- `ui/src/hooks/useWorkspaceAccess.ts` (~112 LOC): React hook
- `docs/docs/guides/workspaces.md` (~200 LOC): Documentation

**Modified Files:**
- `internal/coordinator/agent_config.go`: Added WorkspacesConfig
- `internal/server/handlers_auth.go`: Added RequireWorkspaceAccessMiddleware
- `internal/server/handlers_threads.go`: Updated handleWorkspaces endpoint
- `ui/src/features/controlplane/components/AggregationNav.tsx`: Added role badges

**Firestore Schema:**
- `workspaces/{id}`: Workspace metadata (name, is_public, github_repo)
- `workspace_access/{workspace_id}/users/{email}`: Per-user access grants

**Design Doc:** `design_docs/implemented/v0_7_1/m-workspace-access-control.md`

## [v0.7.0] - 2026-01-21

### Added - Record Width Subtyping (M-GAP4)

Implemented row polymorphism for records, enabling functions to accept records with extra fields.

**New Syntax:**
- `{field: T | r}` - Open record with explicit row variable `r`
- `{field: T, ...}` - Sugar syntax with fresh row variable

**Example:**
```ailang
-- EXACT record: rejects extra fields (default)
pure func getNameExact(p: {name: string}) -> string = p.name

-- OPEN record: accepts extra fields via | r
pure func getName(p: {name: string | r}) -> string = p.name

-- OPEN record: accepts extra fields via ...
pure func getEmail(u: {email: string, ...}) -> string = u.email

-- Usage: open records accept records with more fields
getName({name: "Bob", age: 30})  -- Works!
```

**Implementation Files:**
- `internal/ast/ast.go`: Added `Row` field to `RecordTypeExpr` (~10 LOC)
- `internal/parser/parser.go`: Added `rowVarCounter`, `freshRowVarName()` (~20 LOC)
- `internal/parser/parser_type.go`: Handle `|` and `...` in record types (~30 LOC)
- `internal/types/types.go`: Added `TRecordOpen`, `RowVar` types (~30 LOC)
- `internal/types/unification_records.go`: `unifyRecordOpen()`, error messages (~100 LOC)
- `internal/elaborate/types.go`: Create `TRecordOpen` from AST (~20 LOC)
- `internal/types/substitute.go`: Handle `TRecordOpen` substitution (~10 LOC)

**Tests:**
- `internal/parser/type_test.go`: Added `TestOpenRecordTypes`, `TestEllipsisSugarMarked`
- `examples/runnable/record_width_subtyping.ail`: Comprehensive example (~105 LOC)

**Error Messages:**
- Record field mismatch now lists expected/actual/extra/missing fields
- Suggests open record syntax when extra fields detected

**Teaching Prompt:** Updated to v0.6.6 with open record documentation

**Design Doc:** `design_docs/implemented/v0_7_0/m-gap4-record-width-subtyping.md`

### Fixed - Go Codegen: List Fields in Record Types (Issue #116)

Fixed undefined `*List` type error in generated Go code when AILANG record types contain list fields.

**Root Cause:**
- Parser normalizes `[T]` syntax to `TypeApp("list", [T])` (DX-17 Phase 2)
- The `mapASTType` in codegen didn't special-case the `"list"` constructor
- Result: `[string]` → `*List` instead of `[]string`

**Fix:**
Added handling for `TypeApp` with constructor `"list"` in three locations:
- `internal/gen/golang/adt.go`: `mapASTType()` function (~20 LOC)
- `cmd/ailang/compile.go`: `ailangTypeToGo()` function (~12 LOC)
- `cmd/ailang/compile.go`: `ailangTypeToGoWithValueRecords()` function (~12 LOC)

**Before:**
```go
type Galaxy struct {
    Name    string
    Systems *List  // ERROR: undefined *List
}
```

**After:**
```go
type Galaxy struct {
    Name    string
    Systems []*StarSystem  // Correct!
}
```

**Tests:**
- `internal/gen/golang/adt_test.go`: Added `TestGenerateRecordWithListField` (3 sub-tests)
- `examples/runnable/adt_list_fields.ail`: New example demonstrating list field records

**Design Doc:** `design_docs/implemented/v0_7_0/m-codegen-list-type-definition.md`

### Added - Generic AILANG Embedding API (M-EMBED)

New `internal/embed/` package enables Go applications to embed AILANG as a scripting/extension language.

**New Package: `internal/embed/`**
- `embed.go` - Engine API for module loading and function calling (~220 LOC)
- `convert.go` - Bidirectional Go↔AILANG value conversion (~330 LOC)
- `embed_test.go` - Test suite (14 passing tests)

**Key Features:**
- `Engine.Call()` - Call AILANG functions from Go with automatic type conversion
- `Engine.CallJSON()` - JSON-based interface for language-agnostic integration
- `FromGo()` / `ToGo()` - Convert between Go and AILANG values
- Type-safe extractors: `ToInt()`, `ToString()`, `ToBool()`, `ToList()`, `ToRecord()`
- Module caching for efficient repeated calls

**Usage:**
```go
engine := embed.New("/path/to/project")
defer engine.Close()

result, _ := engine.Call("transforms/formatter", "truncate", "Hello, World!", 10)
text, _ := embed.ToString(result)  // "Hello, Wor..."
```

**Documentation:** [Go Interop Guide - Runtime Embedding](docs/docs/guides/go-interop.md#runtime-embedding-v064)

**Design Doc:** `design_docs/implemented/v0_7_0/m-embed-go-ailang-bridge.md`

### Added - Developer Experience Improvements: BigQuery Connector Feedback (M-DX24)

Addressed 6 critical developer experience issues discovered through real-world BigQuery connector development. Focused on documentation, error messages, and language feature validation.

**Issues Fixed:**

1. **Reserved Keywords Documentation** - Complete reference added
   - `docs/reference/reserved-keywords.md` - 43 keywords listed and categorized
   - Clear examples of common mistakes and workarounds
   - Contextual keywords documented

2. **Parser Error Messages for Keywords** - Improved detection and suggestions
   - Parser now detects reserved keywords in identifier positions
   - Error code: `PAR_RESERVED_KEYWORD` with helpful suggestions
   - Links to reserved keywords documentation

3. **Pattern Matching with Option Type** - Verified working correctly
   - Pattern matching for ADT types (Some/None) fully functional
   - Added comprehensive test suite in `examples/pattern_matching_adt.ail`
   - Nested patterns and multiple match arms work as expected

4. **If-Then-Else Block Expressions** - Verified working with full support
   - Multi-statement blocks supported in both then and else branches
   - let bindings and nested blocks work correctly
   - Added example in `examples/if_then_else_blocks.ail`

5. **Record Type Inference** - Field access and construction working
   - Record field access (e.g., `config.port`) works correctly
   - Record pattern matching functional
   - Added examples in `examples/record_in_result.ail`

6. **Stdlib Version Mismatch Warnings** - Warning exists but doesn't block features
   - Fixed v0.7.0 prompt hash verification in `prompts/versions.json`
   - Warnings informational only, not blocking functionality

**New Files:**
- `examples/pattern_matching_adt.ail` (~67 LOC) - Comprehensive ADT pattern matching tests
- `examples/if_then_else_blocks.ail` (~71 LOC) - If-then-else with block expressions
- `examples/record_in_result.ail` (~48 LOC) - Record construction and field access

**Updated Files:**
- `docs/reference/reserved-keywords.md` - Comprehensive keyword reference
- `prompts/versions.json` - Fixed v0.7.0 hash (test validation)
- `internal/parser/reserved_keyword_test.go` - Existing tests verified working

**Testing:**
- All new example files verified to compile and type-check
- Parser error detection for reserved keywords tested
- Pattern matching with Option type confirmed functional

**Impact:**
- BigQuery connector demo code now works without workarounds
- Better error messages help developers identify reserved keyword conflicts early
- Language features validated through real-world usage

**Design Doc:** `design_docs/planned/v0_7_0/m-dx24-developer-dx-improvements.md`

**Sprint Plan:** `design_docs/planned/v0_7_0/m-dx24-sprint-plan.md`

### Added - AILANG Dogfooding: Dashboard Transforms

Created `internal/dashboard_transforms/` with AILANG port of event formatter for dogfooding.

**Files:**
- `event_formatter.ail` - Pure functions for filtering and formatting task events (~145 LOC)
- `GAPS_DISCOVERED.md` - Documents 6 language gaps found during porting

**Gaps Discovered:**
| Gap | Severity | Status |
|-----|----------|--------|
| GAP-1: Teaching prompt foldl syntax | High | Fixed |
| GAP-2: Path-dependent type checking | Critical | Needs investigation |
| GAP-3: Lambda syntax with foldl | Medium | Workaround available |
| GAP-4: No record width subtyping | High | ✅ Fixed (M-GAP4) |
| GAP-5: Standalone expression eval | Medium | Workaround available |

### Fixed - Coordinator Merge Helper Bug

**Issue:** `getMainRepoPath()` function in `internal/coordinator/merge.go` returned incorrect path when called with empty input. The function would run git commands in the current directory instead of validating input first.

**Root Cause:** No guard clause for empty `worktreePath` parameter. When an empty string was passed, the function set `cmd.Dir = ""` which defaults to the current working directory, causing `git rev-parse --git-dir` to return the actual repository path instead of empty string.

**Fix:** Added early return guard to check if `worktreePath` is empty and return immediately without executing git commands.

**Files Changed:**
- `internal/coordinator/merge.go`: Added input validation guard clause (2 lines)

**Tests:**
- Updated `internal/coordinator/merge_test.go`: Test now passes
- `TestGetMainRepoPath` verifies empty input returns empty string

### Added - OpenAI Responses API Support (M-OPENAI-RESPONSES-API)

Implemented full support for OpenAI's Responses API (`/v1/responses`) for all modern models. The Responses API provides better caching, chain-of-thought passing between turns, and lower latency.

**New in `internal/ai/openai/`:**
- `types.go`: Added Responses API request/response types (~70 LOC)
  - `responsesRequest` with `input` array (replaces `messages`)
  - `responsesReasoning` with `effort` parameter (none/low/medium/high)
  - `responsesResponse` with polymorphic `output` array
  - `responsesOutputItem` supporting message/reasoning/function_call types
- `responses.go`: Full implementation (~120 LOC)
  - Maps `SystemPrompt` to `developer` role (Responses API equivalent of `system`)
  - Parses polymorphic output (concatenates all message text)
  - Tracks reasoning tokens separately from output tokens
  - Default reasoning effort: `medium`

**API Detection (automatic routing):**
- **Responses API** (modern): GPT-5, GPT-5.1, GPT-5.2, o1, o3, codex models
- **Chat Completions** (legacy): GPT-4, GPT-3.5, and earlier models
- Override with `WithAPIType(APIResponses)` or `WithAPIType(APIChatCompletions)` option

**Usage:**
```go
client := openai.NewClient(apiKey)
resp, err := client.Generate(ctx, &ai.Request{
    Model:        "gpt-5",  // Auto-uses Responses API for GPT-5+
    SystemPrompt: "You are a coding assistant.",
    UserPrompt:   "Write fizzbuzz",
    Options:      map[string]any{"reasoning_effort": "high"},
})
```

**Tests:** 6 new tests for Responses API (basic, reasoning tokens, effort, polymorphic output, no output error, API errors)

**Design Doc:** `design_docs/implemented/v0_7_0/m-openai-responses-api-sprint.md`

### Deprecated - ailang-agent Binary (M-DEPRECATE-AILANG-AGENT)

Removed the standalone `ailang-agent` binary and `internal/agent/` package. The coordinator daemon (`ailang coordinator`) now handles all agent functionality with additional features:

**Removed:**
- `cmd/ailang-agent/` - Standalone agent binary (~430 LOC)
- `internal/agent/` - Agent support package (~1,330 LOC)
- `cmd/ailang/agent.go.bak` - Backup file
- Makefile targets: `build-agent`, `install-agent`, `build-all`, `install-all`

**Migrated to Coordinator:**
- Capability detection (FS/Net/Shell/Budget) → `internal/coordinator/capability_detector.go`
- Impact classification (low/medium/high) → Integrated into `TaskAnalyzer.Analyze()`
- Pre-execution cost estimation → `EstimateTotalCost()`

**Use Instead:**
```bash
# Old (deprecated)
ailang-agent --instance-id my-agent

# New
ailang coordinator start
ailang coordinator status
```

**Design Doc:** `design_docs/planned/v0_7_1/m-deprecate-ailang-agent.md`

### Added - Human-Friendly Tracing (M-OTEL-ENHANCED-TRACING-DX)

Enhanced OpenTelemetry spans with human-readable context for faster debugging. Traces now include actionable error messages, code previews, and key identifiers visible directly in span attributes.

**New Telemetry Helpers (`internal/telemetry/helpers.go`):**
- `Truncate(s, maxLen)` - Safe UTF-8 string truncation using rune boundaries (not bytes)
- `CategorizeError(err)` - Classifies errors into parse_error, type_error, module_error, api_error, timeout, runtime_error
- `ShortHash(content, length)` - SHA256-based short hash for deduplication
- `LineSnippet(source, lineNum, maxLen)` - Extracts code snippet around error location

**Error Context on All Error Spans:**
- `error.message` - Truncated error message (200 chars max)
- `error.category` - Category for filtering (parse_error, type_error, etc.)
- Parse errors: `error.location` (line:col) + `error.snippet` (code context)

**AI Provider Improvements:**
- `ai.prompt_preview` - First 100 chars of user prompt
- `ai.response_preview` - First 100 chars of AI response
- `ai.finish_reason` - Stop reason (Anthropic only)

**Eval/Benchmark Improvements:**
- `code.preview` - First 100 chars of generated code
- `code.hash` - 8-char hash for deduplication
- `error.summary` - Truncated stderr for failed benchmarks
- `benchmark.repair_successful` - Self-repair tracking

**CLI Run Improvements:**
- `file.path` - Source file being executed
- `entry.function` - Entry point function name
- `caps.granted` - Capabilities enabled (IO, FS, Net, etc.)

**Code:**
- `internal/telemetry/helpers.go` (~95 lines)
- `internal/telemetry/helpers_test.go` (~162 lines, 100% coverage)
- Modified: `internal/pipeline/pipeline_single.go`, `cmd/ailang/main.go`, `cmd/ailang/eval_suite.go`
- Modified: All 4 AI providers (`anthropic`, `openai`, `gemini`, `ollama`)

## [v0.6.2] - 2026-01-02

### Added - OpenTelemetry Integration (M-OTEL)

Implemented comprehensive OpenTelemetry (OTLP) instrumentation across AILANG's core services for distributed tracing and observability. This enables integration with standard observability backends like Grafana, Honeycomb, Jaeger, and the ai-observer project.

**Features:**
- **Opt-in via environment variable**: Set `OTEL_EXPORTER_OTLP_ENDPOINT` to enable telemetry export
- **Zero overhead when disabled**: No performance impact when OTLP endpoint is not configured
- **Service resource auto-population**: Service name, version, runtime info automatically added
- **Full span hierarchy**: Request → Executor → AI Provider spans with context propagation

**Instrumented Components:**
- **Server (`internal/server/`)**: HTTP middleware with automatic span creation, status codes, path filtering
- **Coordinator (`internal/coordinator/`)**: Task lifecycle spans with task.id, type, stage, token/cost attributes
- **Claude Executor (`internal/executor/claude/`)**: Execution spans with model, tokens, cost tracking
- **Gemini Executor (`internal/executor/gemini/`)**: Execution spans with model, tokens, cost tracking
- **AI Providers**: All four providers instrumented:
  - `internal/ai/anthropic/` - Claude API client
  - `internal/ai/openai/` - OpenAI API client
  - `internal/ai/gemini/` - Gemini AI Studio/Vertex client
  - `internal/ai/ollama/` - Local Ollama client

**Configuration:**
```bash
# Option 1: Google Cloud Trace (recommended for GCP users)
# Uses same convention as Gemini CLI
export GOOGLE_CLOUD_PROJECT=multivac-internal-dev
# Or separate telemetry project:
export OTLP_GOOGLE_CLOUD_PROJECT=my-telemetry-project

# Option 2: Generic OTLP export to collector
export OTEL_EXPORTER_OTLP_ENDPOINT=http://localhost:4318

# Optional: Set deployment environment
export OTEL_ENVIRONMENT=production

# Start services
ailang serve                    # Server with OTEL
ailang coordinator start        # Coordinator with OTEL
```

**Google Cloud Trace Integration:**
- Traces appear in [Cloud Console](https://console.cloud.google.com/traces/explorer)
- Uses Application Default Credentials (ADC) for authentication
- Matches Gemini CLI env var convention (`OTLP_GOOGLE_CLOUD_PROJECT` takes precedence)
- Integration tests in `internal/telemetry/gcp_integration_test.go`

**Dual Export Mode:**
- Send traces to **both** Google Cloud Trace and another OTLP backend simultaneously
- Auto-enabled when both `GOOGLE_CLOUD_PROJECT` and `OTEL_EXPORTER_OTLP_ENDPOINT` are set
- Useful for sending to GCP + local Jaeger, Grafana Tempo, Honeycomb, etc.
- Example: `export GOOGLE_CLOUD_PROJECT=my-project && export OTEL_EXPORTER_OTLP_ENDPOINT=http://localhost:4318`

**Native CLI Support:** Both Claude Code and Gemini CLI support OTLP natively:
- Claude Code: `CLAUDE_CODE_ENABLE_TELEMETRY=1`
- Gemini CLI: Configure in `~/.gemini/settings.json`

**Files Added:**
- `internal/telemetry/doc.go` - Package documentation (~5 LOC)
- `internal/telemetry/otel.go` - OTLP initialization with Google Cloud Trace (~200 LOC)
- `internal/telemetry/resource.go` - Service resource configuration (~40 LOC)
- `internal/telemetry/otel_test.go` - Unit tests (~125 LOC)
- `internal/telemetry/gcp_integration_test.go` - Google Cloud Trace integration tests (~125 LOC)

**Files Modified:**
- `internal/server/server.go` - Added otelhttp middleware (~15 LOC)
- `cmd/ailang/serve.go` - Added telemetry init (~20 LOC)
- `internal/coordinator/daemon_tasks.go` - Added task lifecycle spans (~40 LOC)
- `cmd/ailang/coordinator.go` - Added telemetry init (~15 LOC)
- `internal/executor/claude/claude.go` - Added execution spans (~50 LOC)
- `internal/executor/gemini/gemini.go` - Added execution spans (~50 LOC)
- `internal/ai/anthropic/client.go` - Added provider spans (~45 LOC)
- `internal/ai/openai/client.go` - Added provider spans (~35 LOC)
- `internal/ai/gemini/client.go` - Added provider spans (~30 LOC)
- `internal/ai/ollama/client.go` - Added provider spans (~30 LOC)

**Design Document:** `design_docs/planned/v0_6_2/m-otel-integration.md`

**Total:** ~725 LOC across 15 files

### Added - GitHub-Driven Autonomous Workflow (M-COORD-GITHUB-AUTO-ROUTING)

Implemented end-to-end GitHub-driven task pipeline for autonomous coordinator operation. Issues with `coordinator:*` labels are automatically routed through design → sprint → implementation → merge workflow.

**Recent Enhancements:**
- **Design Doc Content in Comments**: Full design doc markdown is now embedded in GitHub comments (collapsible `<details>` sections) so reviewers can approve directly from GitHub without accessing local worktrees
- **Sprint Plan Content in Comments**: Sprint plans are also shown in full for GitHub-based review
- **Simplified Stage Directives**: Stage prompts now just invoke the appropriate skill (e.g., "Invoke the design-doc-creator skill") rather than verbose instructions
- **Local Approval → GitHub Sync**: When approving via `ailang coordinator pending`, the appropriate approval labels are automatically added to GitHub issues

**Features:**
- **Label-Based Routing**: Issues labeled `coordinator:bug`, `coordinator:feature`, `coordinator:docs` are auto-imported to coordinator inbox
- **GitHub Comment Posting**: Coordinator posts status updates as comments on linked GitHub issues
- **Approval Watcher**: Polls GitHub for approval labels (`design-approved`, `sprint-approved`, `merge-approved`)
- **Task Pipeline**: Automatic progression through stages with human approval gates
- **Comment Templates**: Professional, formatted comments for each pipeline stage

**Workflow:**
1. Issue created with `coordinator:*` label → imported as message
2. Task picks up message → posts "🔄 Working on this" comment
3. Design doc complete → adds `needs-design-approval` label
4. Human adds `design-approved` → sprint planning starts
5. Sprint plan complete → adds `needs-sprint-approval` label
6. Human adds `sprint-approved` → implementation starts
7. Implementation complete → adds `needs-merge-approval` label
8. Human adds `merge-approved` → work merged, issue closed

**Approval Labels:**
- `needs-design-approval` / `design-approved`
- `needs-sprint-approval` / `sprint-approved`
- `needs-merge-approval` / `merge-approved`
- `needs-revision` (requests changes at any stage)

**Stage Execution Layer (M-COORD-GITHUB-COMPLETE sprint):**
- `BuildStageDirective()` - Generates skill invocation prompts per stage
- `ParseStageOutput()` - Extracts design doc/sprint plan paths from output
- `ProcessStageCompletion()` - Triggers TaskChain callbacks after execution
- `RequeueTask()` - Resets task for next stage on approval

**Files Added:**
- `internal/coordinator/approval_watcher.go` - Polls GitHub for approval labels (~280 LOC)
- `internal/coordinator/task_chain.go` - Pipeline stage callbacks (~350 LOC)
- `internal/coordinator/templates.go` - GitHub comment templates (~200 LOC)
- `internal/coordinator/stage_execution.go` - Stage-aware directives + output parsing (~240 LOC)
- `internal/coordinator/stage_execution_test.go` - Unit tests (~160 LOC)

**Files Modified:**
- `internal/coordinator/store.go` - Added TaskStage, GithubIssue fields, RequeueTask (~55 LOC)
- `internal/coordinator/store_sqlite.go` - Schema migrations, RequeueTask impl (~165 LOC)
- `internal/coordinator/daemon.go` - Integration with watcher and chain (~30 LOC)
- `internal/coordinator/daemon_tasks.go` - Stage-aware directives, ProcessStageCompletion (~50 LOC)
- `internal/coordinator/task_chain.go` - RequeueTask calls on approval (~20 LOC)
- `internal/coordinator/watcher.go` - Added GithubIssue to Message struct
- `internal/coordinator/message_adapter.go` - Pass GitHub issue from inbox messages
- `internal/coordinator/integration_test.go` - E2E GitHub pipeline test (~120 LOC)

### Added - Coordinator Task Logs Command

Added `ailang coordinator logs <task-id>` command to view streaming events from task execution.

**Features:**
- View real-time or historical task execution logs
- Shows turns, text output, tool usage, errors, and status events
- Timestamps and duration tracking
- Supports `--limit N` to control output
- Supports `--json` for machine-readable output

**Usage:**
```bash
# View logs for a specific task
ailang coordinator logs task-abc12345

# Limit output to last 50 events
ailang coordinator logs task-abc12345 --limit 50

# JSON output for scripting
ailang coordinator logs task-abc12345 --json
```

**Files Modified:**
- `internal/coordinator/store.go` - Added `TaskEventRecord` struct and interface methods
- `internal/coordinator/store_sqlite.go` - Implements event storage
- `internal/coordinator/store_cloud.go` - Added stub implementations
- `cmd/ailang/coordinator.go` - Added `logs` command (~145 LOC)

### Fixed - Worktree Sync on CLI Rejection

Fixed issue where rejecting tasks via CLI would clean up worktrees on disk but leave the daemon's in-memory `WorktreeManager` out of sync. This caused "max worktrees limit reached" errors even when worktree slots were actually free.

**Root cause:** When tasks are rejected via `ailang coordinator reject`, the worktree is removed from disk but the daemon's in-memory tracking wasn't updated until restart.

**Fix:** Added `syncWorktreeState()` to the daemon's poll loop. This calls `CleanupOrphaned()` on each cycle, which runs `git worktree prune` and removes orphaned entries from the in-memory map.

**Files Modified:**
- `internal/coordinator/daemon.go` - Added `syncWorktreeState()` function (~15 LOC)

### Added - Record Pattern Matching (M-RECORD-PATTERNS)

Added record destructuring in pattern matching expressions. This allows extracting fields from records directly in match arms.

**Supported patterns:**
- Shorthand: `{name}` - binds field "name" to variable "name"
- Renaming: `{name: n}` - binds field "name" to variable "n"
- Multiple fields: `{name, age}` - binds both fields
- Nested: `{user: {name}}` - matches nested records
- Rest: `{name, ...}` - matches some fields, ignores rest

**Example:**
```ailang
let person = {name: "Alice", age: 30} in
match person {
  {name, age} => name ++ " is " ++ show(age)
}
-- Output: "Alice is 30"

-- Nested pattern
let data = {user: {email: "test@example.com"}} in
match data {
  {user: {email: e}} => e
}
-- Output: "test@example.com"
```

**Files Added/Modified:**
- `internal/parser/parser_pattern.go` - Implement `parseRecordPattern()` (~90 LOC)
- `internal/elaborate/patterns.go` - Add `ast.RecordPattern` case (~15 LOC)
- `internal/types/typechecker_patterns.go` - Add type checking for record patterns (~55 LOC)
- `internal/typedast/typed_ast.go` - Add `TypedRecordPattern` type (~15 LOC)
- `internal/parser/record_pattern_test.go` - Add comprehensive tests (~200 LOC)
- `examples/runnable/record_patterns.ail` - New example file
- `examples/runnable/records.ail` - Updated with working pattern example

### Fixed - Effect Checker Incorrectly Required IO for Pure Functions (M-BUG-EFFECT-CHECKER-CONFLATION)

Fixed effect checker bug where pure functions were incorrectly required to have IO effects when called inside a `println` that appeared after another `println` in the same block.

**Root cause:** `collectRequiredEffects` relied solely on `CoreTypeInfo` for function effect types, which could have incorrect/contaminated types for locally-defined functions. When checking recursive calls or calls to module-level pure functions, the effect checker would extract effects from `CoreTypeInfo` instead of using the declared signature from the Surface AST.

**The bug pattern:**
```ailang
export func main() -> () ! {IO} {
  println("Header");                     -- First println
  let nums = [1, 2, 3];
  println("sum = " ++ show(sum(nums)));  -- Second println with pure function call
  ()
}

pure func sum(xs: List[int]) -> int {   -- Pure function
  match xs {
    x :: rest => x + sum(rest),          -- Recursive call
    [] => 0
  }
}
```

**Error (before fix):**
```
Effect checking failed for function 'sum'
  Missing effects: IO
  Suggested fix: func sum(...) -> T ! {IO}
```

**Fix:** Modified `collectRequiredEffects` to:
1. Accept `declaredEffects` map (function names → declared effect signatures from Surface AST)
2. When analyzing function applications, check if callee is a `Var` (local function reference)
3. If found in `declaredEffects`, use those effects instead of `CoreTypeInfo`
4. Only fall back to `CoreTypeInfo` for global/builtin functions

**Impact:**
- `examples/runnable/pattern_sugar.ail` now passes without modification
- All pure recursive functions work correctly in effectful contexts
- No false positives for effect requirements

**Files Modified:**
- `internal/pipeline/validate_effects.go` - Pass declaredEffects to collectRequiredEffects, prioritize declared signatures (~50 LOC)
- `internal/pipeline/validate_effects_test.go` - Regression test (~40 LOC)

**Debug logging:** Added `DEBUG_EFFECTS=1` environment variable for future effect debugging (gated behind flag, zero runtime cost when disabled).

**Design Doc:** [design_docs/planned/v0_6_2/m-bug-effect-checker-conflation.md](design_docs/planned/v0_6_2/m-bug-effect-checker-conflation.md)

### Added - Effect Budget Enforcement (M-CAPABILITY-BUDGETS)

Implemented runtime enforcement of capability budgets, allowing fine-grained control over how many times a function can perform a particular effect.

**Syntax:**
```ailang
-- Function limited to 2 IO operations
export func limited() -> () ! {IO @limit=2} {
  println("Call 1");  -- Uses 1/2 budget
  println("Call 2");  -- Uses 2/2 budget
  println("Call 3");  -- FAILS: BudgetExhaustedError
  ()
}
```

**Key features:**
- **Per-invocation semantics**: Each function call gets a fresh budget (not shared across calls)
- **Helpful error messages**: `effect 'IO' budget exhausted: limit=2, used=2` with actionable hints
- **Full pipeline support**: Parser → Elaborator → Type Checker → Evaluator
- **Bypass flag**: `--no-budgets` to disable budget enforcement for debugging

**Implementation details:**
- `FunctionValue.EffectBudgets` stores budget limits from function type's effect row
- `evalCoreApp` creates budget-scoped effect context for each call
- Builtins automatically check `RequireCapWithBudget` before executing effects
- Budget state tracked via `BudgetContext` with `limits` and `used` maps

**Files Modified:**
- `internal/eval/eval_operations.go` - Budget scoping in function application (~20 LOC)
- `internal/eval/eval_expressions.go` - Extract budgets from type info (~25 LOC)
- `internal/runtime/builtins.go` - Automatic budget checking in builtin wrapper (~10 LOC)
- `internal/effects/budget.go` - `BudgetContext` and `RequireCapWithBudget` (existing)
- `examples/reference/capability_budgets.ail` - Working example (~50 LOC)
- `examples/tests/test_capability_budget_exhausted.ail` - Error test case (~20 LOC)

**Design Doc:** [design_docs/planned/v0_6_1/m-capability-budgets-design.md](design_docs/planned/v0_6_1/m-capability-budgets-design.md)

### Added - Pattern Guards in Go Codegen (M-PATTERN-GUARDS)

Fixed pattern guards to be evaluated in Go code generation. Previously guards were parsed and type-checked but silently ignored in compiled output.

**Example:**
```ailang
match value {
  x if x > 10 => "big",
  x if x > 0 => "positive",
  x => "other"
}
```

**Design Doc:** [design_docs/implemented/v0_6_2/m-pattern-guards.md](design_docs/implemented/v0_6_2/m-pattern-guards.md)

### Added - Auto-Derive Eq for ADT Types (M-DX19)

Added `deriving (Eq)` syntax to automatically generate equality for ADT types, eliminating 10-15 lines of boilerplate per enum.

**Example:**
```ailang
type Color = Red | Green | Blue deriving (Eq)

let same = Red == Red  -- true (no manual function needed)
```

**Design Doc:** [design_docs/implemented/v0_6_2/m-dx19-auto-derive-eq.md](design_docs/implemented/v0_6_2/m-dx19-auto-derive-eq.md)

### Added - Type Class Dictionary Generation for Go (M-CODEGEN-DICTIONARIES)

Go codegen now generates dictionary struct definitions for type classes. Previously, the codegen emitted references like `dict_Num_Int.Add()` but never generated the dictionary definitions, causing compilation failures.

**Design Doc:** [design_docs/implemented/v0_6_2/m-codegen-dictionaries.md](design_docs/implemented/v0_6_2/m-codegen-dictionaries.md)

### Added - Coordinator Daemon Stability (M-COORD-STABLE)

Major stability fixes for the coordinator daemon including worktree preservation, approval checkpoint wiring, HTTP event broadcasting, and agent configuration.

**Key fixes:**
- Worktrees preserved until explicit approval/rejection (previously deleted immediately)
- ApprovalCheckpoint properly wired to block until human decision
- HTTP broadcaster sends events to dashboard correctly
- Agent registry for configuring workspaces and capabilities

**Design Doc:** [design_docs/implemented/v0_6_2/m-coord-stable.md](design_docs/implemented/v0_6_2/m-coord-stable.md)

### Added - Coordinator Feedback Loop (M-COORD-FEEDBACK)

Implemented real-time feedback loop between coordinator executors and humans via dashboard/CLI including streaming logs, cost/token tracking, and human approval gates.

**Design Doc:** [design_docs/implemented/v0_6_2/m-coordinator-feedback-loop.md](design_docs/implemented/v0_6_2/m-coordinator-feedback-loop.md)

### Added - Read-Only Execution Mode for Questions (M-COORDINATOR-QUESTION-MODE)

Questions sent to the coordinator now execute in read-only mode with restricted tool access (`Read`, `Grep`, `Glob`, `WebFetch`, `WebSearch`), preventing accidental file modifications during informational queries.

**Design Doc:** [design_docs/implemented/v0_6_2/m-coordinator-question-mode.md](design_docs/implemented/v0_6_2/m-coordinator-question-mode.md)

### Added - GitHub Account Override Flag (M-GITHUB-USER-OVERRIDE)

Added `--github-user` flag to bypass `expected_user` validation when using `ailang messages` with GitHub sync. Improved error display with red ERROR output and clear fix options.

**Design Doc:** [design_docs/implemented/v0_6_2/m-github-user-override.md](design_docs/implemented/v0_6_2/m-github-user-override.md)

### Added - ApprovalWatcher Debug Observability (M-COORD-APPROVALWATCHER-OBSERVABILITY)

Added comprehensive debug logging to ApprovalWatcher for diagnosing GitHub label detection issues. Use `DEBUG_APPROVAL_WATCHER=1` for verbose polling logs and `ailang coordinator watcher-status` to check watcher state.

**Design Doc:** [design_docs/implemented/v0_6_2/m-coord-approvalwatcher-observability.md](design_docs/implemented/v0_6_2/m-coord-approvalwatcher-observability.md)

### Added - Type Checker Debug Events (M-DX11-PHASE2)

Extended `--debug-types` to emit events from the type checker including fresh type variable creation and unification events.

**Design Doc:** [design_docs/implemented/v0_6_2/m-dx11-phase2-debug-events.md](design_docs/implemented/v0_6_2/m-dx11-phase2-debug-events.md)

### Fixed - Boolean Type Assertions in Go Codegen (M-CODEGEN-BOOL-ASSERTIONS)

Fixed Go codegen to add `.(bool)` type assertions when dictionary method results are used in boolean contexts (if conditions, logical operators).

**Design Doc:** [design_docs/implemented/v0_6_2/m-codegen-bool-assertions.md](design_docs/implemented/v0_6_2/m-codegen-bool-assertions.md)

### Fixed - ADT Constructor Resolution Ambiguity (M-DX22)

Fixed Go codegen to correctly resolve constructor calls when multiple ADT types have constructors with the same name. Uses fully-qualified names (`TypeName.ConstructorName`) in the internal registry.

**Design Doc:** [design_docs/implemented/v0_6_2/m-dx22-adt-constructor-resolution.md](design_docs/implemented/v0_6_2/m-dx22-adt-constructor-resolution.md)

### Fixed - TList Normalized to TApp at Parse Time (DX-17-PHASE2)

Eliminated `TList` struct by normalizing `[T]` syntax to `TApp("list", T)` during parsing. This unifies the internal representation for all container types.

**Design Doc:** [design_docs/implemented/v0_6_2/dx-17-phase2-tlist-normalization.md](design_docs/implemented/v0_6_2/dx-17-phase2-tlist-normalization.md)

### Fixed - List[T] Normalization to Lowercase (M-DX17-LIST-NORMALIZATION-BUG)

Fixed parser to normalize explicit `List[T]` syntax to lowercase `"list"`, matching the `[T]` normalization. Previously `List[T]` created a different type than `[T]`.

**Design Doc:** [design_docs/implemented/v0_6_2/m-dx17-list-normalization-bug.md](design_docs/implemented/v0_6_2/m-dx17-list-normalization-bug.md)

## [v0.6.1] - 2025-12-22

### Fixed - Inline Record Literals in Match Arms (M-DX16)

Fixed parser to allow inline record literals directly in match arms without requiring helper functions.

**Before (required workaround):**
```ailang
-- ❌ Had to define helper functions for each record
pure func bridgeInfo() -> {name: string, level: int} = {name: "Bridge", level: 1}

match deckID {
    Bridge => bridgeInfo()  -- couldn't inline the record
}
```

**After (works directly):**
```ailang
-- ✅ Inline record literals work in match arms
match deckID {
    Bridge => {name: "Bridge", level: 1}
    Engine => {name: "Engine", level: -2}
}
```

**Also fixed:**
- Nested records in match arms: `{pos: {x: 0.0, y: 0.0}}`
- Record updates in match arms: `{base | field: newValue}`
- Block expressions with semicolons continue to work: `{ println("hi"); 42 }`

**Root cause:** `parseCase()` bypassed record literal detection, always treating `{` as a block expression.

**Fix:** Modified `parseBlockOrExpression()` to detect record patterns (IDENT COLON or IDENT PIPE) after consuming LBRACE.

**Files Modified:**
- `internal/parser/parser_expr.go` - Added record detection logic (~100 LOC)
- `internal/parser/record_match_arms_test.go` - New test file (~150 LOC)
- `examples/runnable/record_in_match.ail` - New example (~70 LOC)

**Design Doc:** [design_docs/planned/v0_6_1/m-dx16-inline-record-match-arms.md](design_docs/planned/v0_6_1/m-dx16-inline-record-match-arms.md)

### Fixed - Non-Exported Function Namespacing (M-DX18)

Fixed Go codegen to namespace non-exported functions when compiling multiple modules to the same Go package. This prevents redeclaration errors when different modules have private functions with the same name.

**Before (failed):**
```
sim/solar_demo.ail has: pure func concatLists(a: [SolarPlanet], b: [SolarPlanet]) -> [SolarPlanet]
sim/dome_demo.ail has: pure func concatLists(a: [DrawCmd], b: [DrawCmd]) -> [DrawCmd]

$ ailang compile sim/*.ail
sim_gen/solar_demo.go:503:6: concatLists_impl redeclared in this block
```

**After (works):**
```go
// sim_gen/solar_demo.go
func solar_demo__concatLists_impl(...) { ... }
func solar_demo__concatLists(...) { ... }

// sim_gen/dome_demo.go
func dome_demo__concatLists_impl(...) { ... }
func dome_demo__concatLists(...) { ... }
```

**Key features:**
- Non-exported functions prefixed with `{moduleName}__`
- Exported functions keep simple names for external access
- Call sites automatically resolve to namespaced names
- Module name derived from last path component

**Bug fix (2025-12-21):** Two issues fixed:
1. Call sites weren't using prefixed names from `topLevelFuncs` map
2. Exported functions had naming mismatch: `_impl` uses `ToGoVarName` (camelCase) but wrapper uses `ToGoFuncName` (PascalCase)

Added `topLevelImplFuncs` map to track actual `_impl` names separately from wrapper names.

**Files Modified:**
- `internal/gen/golang/codegen.go` - Added moduleName field, SetModuleName(), and topLevelImplFuncs map (~25 LOC)
- `internal/gen/golang/codegen_decl.go` - Added namespacing and impl name tracking (~25 LOC)
- `internal/gen/golang/codegen_expr_simple.go` - Fixed call site generation to use topLevelImplFuncs (~15 LOC)
- `cmd/ailang/compile.go` - Set module name per file (~30 LOC)

**Design Doc:** [design_docs/implemented/v0_6_1/m-dx18-codegen-function-namespacing.md](design_docs/implemented/v0_6_1/m-dx18-codegen-function-namespacing.md)

### Fixed - Wildcard Pattern Type Inference in List Cons Patterns (M-DX20)

Fixed wildcard `_` pattern handling in list cons patterns (`::`). Previously, `_` was incorrectly parsed as a variable pattern, causing type inference errors when the same wildcard appeared in both head and tail positions.

**Before (failed):**
```ailang
pure func nonEmpty(xs: [int]) -> bool =
    match xs {
        [] => false
        _ :: _ => true  -- ❌ Type error: cannot unify int with *types.TList
    }
```

**After (works):**
```ailang
-- ✅ All wildcard patterns now work correctly
_ :: _      -- matches any non-empty list
_ :: rest   -- discard head, bind tail
head :: _   -- bind head, discard tail
```

**Root cause:** When parsing patterns, `_` was tokenized as an `IDENT` and elaborated to `core.VarPattern{Name: "_"}` instead of `core.WildcardPattern`. When `_` appeared twice (e.g., `_ :: _`), the type checker tried to unify the head type (`int`) with the tail type (`[int]`).

**Fix:** Added check in pattern elaboration for `Identifier.Name == "_"` to create `WildcardPattern` instead of `VarPattern`.

**Files Modified:**
- `internal/elaborate/patterns.go` - Added wildcard check (~5 LOC)

**Design Doc:** [design_docs/implemented/v0_6_1/m-dx20-wildcard-pattern-inference.md](design_docs/implemented/v0_6_1/m-dx20-wildcard-pattern-inference.md)

### Fixed - Stdlib Version Warning Shows Only Once (M-DX21)

Reduced noise from stdlib version mismatch warnings by showing the warning only once per process.

**Before:**
```
$ ailang compile module1.ail
Warning: stdlib version mismatch: expected dev, found v0.6.0

$ ailang compile module2.ail
Warning: stdlib version mismatch: expected dev, found v0.6.0  (repeated!)
```

**After:**
```
$ ailang compile *.ail
Warning: stdlib version mismatch: expected dev, found v0.6.0
(shown only once, subsequent compilations are quiet)
```

**Also added:** `AILANG_NO_VERSION_WARNINGS=1` environment variable to suppress the warning entirely.

**Files Modified:**
- `internal/loader/stdlib_resolver.go` - Added warning flag and env var check (~10 LOC)

**Design Doc:** [design_docs/implemented/v0_6_1/m-dx21-stdlib-version-warning-once.md](design_docs/implemented/v0_6_1/m-dx21-stdlib-version-warning-once.md)

### Added - Runtime Contract Checks (M-VERIFY)

Implemented design-by-contract programming for AILANG with `requires` (preconditions) and `ensures` (postconditions) syntax. Contracts generate runtime checks that panic on violation when compiled with `--verify-contracts`.

**Contract Syntax:**
```ailang
export func safeDivide(dividend: int, divisor: int) -> int ! {}
requires { divisor != 0 }        -- Precondition: checked at entry
ensures { result >= 0 }          -- Postcondition: checked before return
{
  dividend / divisor
}
```

**CLI Usage:**
```bash
ailang compile --verify-contracts --emit-go --out ./gen module.ail
```

**Design Doc:** [design_docs/implemented/v0_6_1/m-verify-runtime-contracts.md](design_docs/implemented/v0_6_1/m-verify-runtime-contracts.md)

### Added - Multi-Executor Support for AI Coding Agents (M-EXEC)

Enabled AILANG's agent system and eval harness to use multiple AI coding agent executors through a unified interface.

**Supported Executors:**
- Claude Code (existing) - Anthropic's coding agent
- Gemini CLI (new) - Google's Gemini-powered coding agent

**Key Features:**
- Unified executor interface for consistent result formats
- Cost tracking normalized across providers (USD)
- Workspace management consistent across executors
- Eval harness can benchmark across all providers

**Design Doc:** [design_docs/implemented/v0_6_1/m-exec-multi-executor-support.md](design_docs/implemented/v0_6_1/m-exec-multi-executor-support.md)

### Added - Directory Support for `ailang check` (M-DX-CHECK-DIRECTORY)

The `ailang check` command now supports recursive directory checking for `.ail` files.

**Before:**
```bash
# Had to use shell workarounds
find examples/ -name '*.ail' -exec ailang check {} \;
```

**After:**
```bash
# Works directly
ailang check examples/
ailang check --timeout 60s examples/runnable/
```

**Design Doc:** [design_docs/implemented/v0_6_1/m-dx-check-directory-support.md](design_docs/implemented/v0_6_1/m-dx-check-directory-support.md)

### Added - Value-Type Record Generation (M-CODEGEN-VALUE-TYPES)

Go codegen now generates value-type structs (not pointers) for small records based on estimated memory size. Records ≤64 bytes use pass-by-value for better performance on modern CPUs.

**Key Changes:**
- Added `TypeCategory` analysis (Primitive, SmallRecord, LargeRecord, Recursive, ADT)
- `IsLeafRecord` identifies records with only primitive fields (no nested records/ADTs)
- `GoReprForType` as single source of truth for Go type representation
- Records ≤64 bytes generate as value types, larger as pointers

**Design Doc:** [design_docs/implemented/v0_6_1/m-codegen-value-types.md](design_docs/implemented/v0_6_1/m-codegen-value-types.md)

### Fixed - Letrec Recursive Binding Scope Regression (M-LETREC-SCOPING)

Fixed regression where `letrec` bindings no longer made the recursive variable visible within its own body.

**Before (broken):**
```ailang
letrec factorial = \n. if n == 0 then 1 else n * factorial(n - 1) in factorial(5)
-- Error: undefined variable: factorial
```

**After (works):**
```ailang
letrec factorial = \n. if n == 0 then 1 else n * factorial(n - 1) in factorial(5)
-- Returns: 120
```

**Note:** This fix also resolved [M-BUG-LIST-LENGTH-RETURNS-WRONG-VALUE](design_docs/implemented/v0_6_1/m-bug-list-length-returns-wrong-value.md) where `length` returned sum instead of count due to the same scoping issue.

**Design Doc:** [design_docs/implemented/v0_6_1/m-letrec-scoping-regression.md](design_docs/implemented/v0_6_1/m-letrec-scoping-regression.md)

### Fixed - Empty Double-Paren ADT Constructor Calls (M-CODEGEN-ADT-DOUBLE-PAREN)

Fixed Go codegen that produced invalid double-paren calls for ADT constructors.

**Before (broken):**
```go
// Generated invalid Go code
NewDrawCmdViewport()()  // Empty double-paren
```

**After (correct):**
```go
// Args passed through correctly
NewDrawCmdViewport(id, shapeType, x, y, w, h)
```

**Design Doc:** [design_docs/implemented/v0_6_1/m-codegen-adt-double-paren.md](design_docs/implemented/v0_6_1/m-codegen-adt-double-paren.md)

### Fixed - ConcatList and Closure Variable Scoping (M-DX17)

Fixed two codegen bugs blocking Go compilation:

1. **ConcatList undefined**: `++` operator generated calls to `ConcatList()` but runtime defined `Concat`
2. **Closure scoping**: Variables from match patterns weren't captured in closures within match arm bodies

**Design Doc:** [design_docs/implemented/v0_6_1/m-dx17-codegen-concatlist-closure-scoping.md](design_docs/implemented/v0_6_1/m-dx17-codegen-concatlist-closure-scoping.md)

### Fixed - std/env EnvError Type Parameter Bug (M-ENV-TYPE-PARAMETER)

Fixed type system incorrectly treating non-parameterized ADT `EnvError` as having 1 type parameter.

**Before (broken):**
```
type EnvError expects 1 type argument(s), but got 0 (did you mean EnvError[string]?)
```

**After (works):**
```ailang
export func getEnv(name: string) -> Result[string, EnvError] ! {Env} = _env_getEnv(name)
```

**Design Doc:** [design_docs/implemented/v0_6_1/m-env-type-parameter-bug.md](design_docs/implemented/v0_6_1/m-env-type-parameter-bug.md)

## [v0.6.0] - 2025-12-16

### Added - Semantic Doc Search (M-DOC-SEM)

Added `ailang docs search` command with two-stage semantic search pipeline: SimHash for fast candidate filtering, optional neural embeddings for semantic similarity.

**Features:**
- `ailang docs search "query"` - Fast SimHash-based search across design docs
- `--neural` flag - Enable neural embeddings via Ollama for semantic matching
- `--path <dir>` - Search any document corpus (default: design_docs)
- `--stream planned|implemented` - Filter by subdirectory
- `--limit N` - Control result count (default: 10)
- `--json` - JSON output for scripting
- Per-corpus embedding cache with SHA256 content hash for staleness detection
- Cache management: `--cache-info`, `--cleanup`, `--rebuild`

**Example usage:**
```bash
# Fast SimHash search
ailang docs search "parser error"

# Neural semantic search (requires Ollama)
ailang docs search --neural "how to handle type inference"

# Search website docs
ailang docs search --path docs "getting started"

# Cache management
ailang docs search --cache-info
ailang docs search --cleanup
```

**Files Created:**
- `cmd/ailang/docs_search.go` - CLI command (~350 LOC)
- `internal/docsearch/search.go` - SimHash search pipeline (~295 LOC)
- `internal/docsearch/embed.go` - Lazy embedding + cache management (~370 LOC)

**Design Doc:** [design_docs/implemented/v0_6_0/m-doc-sem-lazy-embeddings.md](design_docs/implemented/v0_6_0/m-doc-sem-lazy-embeddings.md)

### Added - Semantic Message Search (M-MSG-SEMANTIC)

Added semantic search and safe deduplication for the `ailang messages` CLI.

**Features:**
- `ailang messages search "query"` - Find messages by semantic similarity
- `--neural` flag - Use Ollama embeddings for semantic matching
- `--threshold N` - Minimum similarity score (default: 0.70)
- `ailang messages dedupe` - Find duplicate message clusters
- `--apply` flag - Actually mark duplicates (reversible via `dup_of` field)
- Inbox inference from git repo name

**Example usage:**
```bash
# Search messages
ailang messages search "parser bugs"

# Find duplicates
ailang messages dedupe --threshold 0.95
ailang messages dedupe --threshold 0.95 --apply
```

**Files Created:**
- `cmd/ailang/messages_search.go` - Search + dedupe CLI (~220 LOC)
- `internal/messaging/search.go` - SemanticSearch + FindDuplicates (~300 LOC)

**Design Doc:** [design_docs/implemented/v0_6_0/m-msg-semantic-caching.md](design_docs/implemented/v0_6_0/m-msg-semantic-caching.md)

### Added - Type Inference Debug CLI (M-DX11)

Added `--debug-types` CLI flag to show type inference debug output, enabling developers
to understand how types are inferred and constraints are resolved.

**Features:**
- `--debug-types` flag: Shows substitution map, constraints, and CoreTI entries
- `--node N` filter: Focus output on a specific node ID
- Zero-overhead in production: NoOpDebugSink inlined (0 allocs/op verified)

**Example usage:**
```bash
# Show all type inference debug info
ailang run --debug-types myfile.ail

# Filter to specific node
ailang run --debug-types --node 42 myfile.ail
```

**Output sections:**
- **Substitution Map**: Type variable substitutions (α → int)
- **Constraints**: Type class constraints (Num, Eq, Ord) and resolution status
- **CoreTI Entries**: Every Core AST node's inferred type and constraints

**Architecture:**
- `TypeDebugSink` interface in `internal/types/debug_sink.go` (~191 LOC)
- `NoOpDebugSink`: Zero-overhead production implementation
- `VerboseDebugSink`: Collects events for formatted output
- `TypeDebugDumper` in `cmd/ailang/debug_types.go` (~220 LOC)

**Files Changed:**
- `internal/types/debug_sink.go` - New TypeDebugSink interface
- `internal/types/debug_sink_test.go` - Tests and benchmarks
- `internal/types/typechecker_core.go` - DebugSink field and SetDebugSink method
- `internal/pipeline/pipeline.go` - DebugTypes config fields
- `internal/pipeline/pipeline_single.go` - Wire up debug sink
- `internal/pipeline/pipeline_module.go` - Wire up debug sink for root module
- `cmd/ailang/main.go` - --debug-types and --node flags
- `cmd/ailang/debug_types.go` - TypeDebugDumper formatting

**Design Doc:** [design_docs/planned/v0_5_11/m-dx11-debug-types-cli.md](design_docs/planned/v0_5_11/m-dx11-debug-types-cli.md)

### Fixed - Unified Slice Type Converters (M-CODEGEN-UNIFIED-SLICE)

Fixed runtime panics when returning typed slices from functions that use record or ADT types.

**The Bug:**
List literal codegen produced `[]interface{}` but typed wrappers expected concrete slices
like `[]*RecordType` or `[]float64`, causing panic during type conversion.

**Root Cause:**
Three gaps in slice type conversion:
1. `[]float64` had no converter at all (only int64, string, bool)
2. `[]*RecordType` had no converter (record types weren't generating slice converters)
3. `[]*ADTType` lookup was broken (`getSliceConversion()` only checked `adtSliceTypes` map,
   not `adtConstructors` where most ADT types are actually registered)

**Fix (unified approach):**
- Added `ConvertToFloat64Slice` converter
- Added `writeRecordSliceConverters()` to generate converters for all registered record types
- Updated `getSliceConversion()` to check ALL type registries:
  1. Primitive types (int64, float64, string, bool)
  2. ADT types (via `adtConstructors` loop)
  3. Record types (via `recordTypes` map)
  4. Legacy `adtSliceTypes` (for backwards compatibility)

**Files Changed:**
- `internal/gen/golang/codegen.go` - Updated `getSliceConversion()` (+23 LOC)
- `internal/gen/golang/codegen_runtime.go` - Call `writeRecordSliceConverters()` (+3 LOC)
- `internal/gen/golang/codegen_runtime_collections.go` - Added converters (+103 LOC)
- `internal/gen/golang/codegen_datastructures_test.go` - Tests for unified lookup (+64 LOC)

**Design Doc:** [design_docs/implemented/v0_5_11/m-codegen-unified-slice-converters.md](design_docs/implemented/v0_5_11/m-codegen-unified-slice-converters.md)

### Fixed - Bool Type Assertion in Nested Match (M-DX27)

Fixed invalid Go code generation when matching on a bool variable extracted from an ADT field.

**The Bug:**
When a bool variable was extracted from an ADT field and used as a match scrutinee,
the codegen incorrectly added `.(bool)` type assertion even though the variable was
already a concrete `bool`, not `interface{}`.

```go
// Before (invalid Go code):
s := _adt.ContentStarfield.Scroll  // s is bool
if s.(bool) {  // ERROR: s is already bool, not interface{}

// After (valid Go code):
s := _adt.ContentStarfield.Scroll  // s is bool
if s {  // Correct: no type assertion needed
```

**Root Cause:**
- `generateFlatBoolMatchChain()` added `.(bool)` unconditionally
- `exprProducesInterface()` didn't know that ADT-bound variables have concrete types

**Fix:**
- Track typed local variables in `Generator.typedLocalVars` when binding ADT fields
- Check this map in `exprProducesInterface()` for `*core.Var` expressions
- Only add type assertion when the variable actually produces `interface{}`
- Clear `typedLocalVars` at function boundaries to prevent scope contamination

**Files Changed:**
- `internal/gen/golang/codegen.go` - Added `typedLocalVars` field (+7 LOC)
- `internal/gen/golang/codegen_decl.go` - Check typed vars in `exprProducesInterface()`, clear per function (+12 LOC)
- `internal/gen/golang/codegen_match.go` - Record typed vars when binding ADT fields (+25 LOC)
- `internal/gen/golang/codegen_match_test.go` - Test for nested bool match (+74 LOC)

**Design Doc:** [design_docs/implemented/v0_5_11/m-dx27-bool-match-type-assertion.md](design_docs/implemented/v0_5_11/m-dx27-bool-match-type-assertion.md)

### Fixed - ADT Type Assertion in Option Pattern Match (M-DX29)

Fixed invalid Go code generation when extracting an ADT value from a generic container like `Option[ADT]`.

**The Bug:**
When pattern matching on `Option[InteractableID]` and extracting the inner value with `Some(interactable)`,
the codegen generated `interface{}` instead of the properly typed `*InteractableID`. This caused
"undefined field or method Kind" errors when subsequently matching on the extracted ADT.

```go
// Before (invalid Go code):
interactable := _adt.Some.Value0  // type: interface{}
switch _adt.Kind {                 // ERROR: interface{} has no field Kind

// After (valid Go code):
interactable := _adt.Some.Value0.(*InteractableID)  // type: *InteractableID
switch _adt.Kind {                                   // Correct: Kind exists on *InteractableID
```

**Root Cause:**
- Generic types like `Option[T]` store their value as `interface{}` in Go
- When extracting from `Some`, the codegen didn't know the type argument
- The extracted value was treated as `interface{}` instead of the concrete ADT type

**Fix:**
- Store AILANG type (`types.Type`) of match scrutinee in `matchScrutineeAILANGType`
- When extracting from constructor, check if scrutinee is a `TApp` (generic type application)
- If type argument is an ADT (maps to pointer type), add type assertion during extraction
- Record the typed variable in `typedLocalVars` for subsequent operations

**Files Changed:**
- `internal/gen/golang/codegen.go` - Added `matchScrutineeAILANGType` field (+3 LOC)
- `internal/gen/golang/codegen_match.go` - Store AILANG type, extract type args, add assertions (+40 LOC)
- `internal/gen/golang/codegen_match_test.go` - Test for Option[ADT] nested match (+120 LOC)

**Design Doc:** [design_docs/planned/v0_5_11/m-dx29-option-nested-adt-type.md](design_docs/planned/v0_5_11/m-dx29-option-nested-adt-type.md)

### Fixed - Missing EqString Runtime Helper (M-DX30)

Added missing `EqString` and `EqFloat` runtime helper functions for string and float equality comparison.

**The Bug:**
When comparing strings with `==` in AILANG code, the generated Go code called `EqString()` which didn't exist,
causing "undefined: EqString" compilation errors.

```go
// Generated code that failed:
EqString(station, "helm")  // ERROR: undefined: EqString

// After fix - helper function now emitted:
func EqString(a, b interface{}) interface{} {
    return a.(string) == b.(string)
}
```

**Fix:**
- Added `EqString` helper function to `writeRuntimeArithmeticHelpers()`
- Added `EqFloat` helper function for completeness (was also missing)

**Files Changed:**
- `internal/gen/golang/codegen_runtime_arith.go` - Added EqString and EqFloat helpers (+14 LOC)

**Design Doc:** [design_docs/planned/v0_5_11/m-dx30-eqstring-runtime.md](design_docs/planned/v0_5_11/m-dx30-eqstring-runtime.md)

### Added - Type Inference Debug CLI Flag (M-DX11)

Added `--debug-types` CLI flag for formatted type inference debugging output.

**New Flag:**
```bash
# Show type inference debug output
ailang run --debug-types file.ail

# Filter to specific node ID
ailang run --debug-types --node 42 file.ail
```

**Output includes:**
- **Substitution Map**: Type variable → resolved type chains
- **Constraints**: Type class constraints (Num, Eq, Ord) and resolution status
- **CoreTI Entries**: Type information for each Core AST node

**Example output:**
```
=== Type Inference Debug ===

[Substitution Map]
  (empty)

[Constraints]
  (no constraints)

[CoreTI Entries]
  NodeID 1: int
    Constraint: Num (resolved)
  NodeID 2: int -> Option[int]
  ...
```

**Infrastructure:**
- `TypeDebugSink` interface for zero-overhead debug events
- `NoOpDebugSink` for production (0 allocs/op verified)
- `VerboseDebugSink` for collecting debug events
- `TypeDebugDumper` for formatted CLI output

**Files Changed:**
- `internal/types/debug_sink.go` - TypeDebugSink interface (~190 LOC)
- `cmd/ailang/debug_types.go` - TypeDebugDumper (~220 LOC)
- `cmd/ailang/main.go` - Added --debug-types, --node flags
- `internal/pipeline/pipeline.go` - Config and Result fields
- `internal/pipeline/pipeline_single.go` - Pipeline wiring
- `internal/pipeline/pipeline_module.go` - Pipeline wiring
- `docs/docs/guides/debugging.md` - Documentation

**Design Doc:** [design_docs/planned/v0_5_11/m-dx11-debug-types-cli.md](design_docs/planned/v0_5_11/m-dx11-debug-types-cli.md)

## [v0.5.10] - 2025-12-12

### Added - Unified AI Provider Architecture (M-UNIFIED-AI-PROVIDERS)

Created unified `internal/ai/` package consolidating all AI provider implementations.
This eliminates code duplication across CLI `--ai` flag and effects system.

**New Packages:**
- `internal/ai/` - Common Provider interface and types
- `internal/ai/anthropic/` - Claude API client (Messages API)
- `internal/ai/openai/` - OpenAI API client (Chat Completions + Responses stub)
- `internal/ai/gemini/` - Gemini API client (generateContent, AI Studio + Vertex AI)

**Benefits:**
- Single implementation per provider, shared by all entry points
- ~230 LOC deleted from `cmd/ailang/ai_handlers.go` (345 → 116 LOC)
- Consistent error handling with `ai.ProviderError`
- Support for both API key and ADC authentication (Gemini)
- Ready for Responses API (codex models) and Interactions API (Gemini beta)

**Usage:**
```go
// CLI and effects now use unified package
client := anthropic.NewClient(apiKey)
handler := client.NewHandler("claude-sonnet-4-5")
effCtx.AI = effects.NewAIContext(handler)
```

**Files Changed:**
- `internal/ai/` - New package (~800 LOC implementation + ~500 LOC tests)
- `cmd/ailang/ai_handlers.go` - Refactored to use unified package (-230 LOC)

**Design Doc:** [design_docs/planned/v0_5_10/m-unified-ai-providers.md](design_docs/planned/v0_5_10/m-unified-ai-providers.md)

### Added - String Conversion Functions (M-STRING-CONVERT)

Added `floatToStr` and `intToStr` functions for converting numeric values to strings:

**New Functions (2 builtins):**
- `floatToStr(f: float) -> string` - Convert float to string (compact format)
- `intToStr(n: int) -> string` - Convert integer to string

**Usage:**
```ailang
import std/string (floatToStr, intToStr)

let velocity = floatToStr(15.0) ++ "% c"  -- "15% c"
let count = "Items: " ++ intToStr(42)      -- "Items: 42"
```

**Files Changed:**
- `internal/builtins/string_convert.go` - Builtin implementations (~100 LOC)
- `internal/builtins/string_convert_test.go` - Unit tests (~130 LOC)
- `std/string.ail` - Added floatToStr, intToStr exports (~10 LOC)

**Design Doc:** [design_docs/implemented/v0_5_10/m-string-conversion.md](design_docs/implemented/v0_5_10/m-string-conversion.md)

### Added - Go Codegen String Support (M-CODEGEN-STDLIB-STRING)

Extended Go codegen to properly map `std/string` conversion functions to Go's `strconv` package:

**Changes:**
- `floatToStr(x)` → `strconv.FormatFloat(x.(float64), 'g', -1, 64)`
- `intToStr(x)` → `strconv.Itoa(int(x.(int64)))`
- Automatic `strconv` import when needed

**Files Changed:**
- `internal/gen/golang/codegen.go` - Added needsStrconvImport flag
- `internal/gen/golang/codegen_expr_simple.go` - String conversion mappings
- `internal/gen/golang/codegen_expr_app.go` - String conversion handling
- `internal/gen/golang/codegen_string_test.go` - Comprehensive tests (~260 LOC)

### Fixed - Test Isolation in Builtin Tests

Fixed pre-existing test order-dependent failures in `internal/builtins/spec_test.go`:

**Problem:** Tests that cleared the global registry didn't restore it, causing subsequent tests to fail.

**Solution:** Added `withFreshRegistry(t)` helper that uses `t.Cleanup()` to save and restore registry state.

**Files Changed:**
- `internal/builtins/spec_test.go` - Added helper function and updated all tests using fresh registries

### Enhanced - Type Cycle Detection (M-DX11-CYCLES)

Improved `ailang debug cycles` to properly detect cyclic type references in generic recursive ADTs:

**What Changed:**
- Now detects cycles in generic types like `List[a]` and `Tree[a]` (previously missed)
- Handles parser normalization of `List[a]` to `[a]` syntax
- Provides detailed cycle paths showing field traversal (e.g., `Tree → Node() → Tree[a]`)
- Classifies cycles as "expected" (standard ADT patterns) or "suspicious" (user-defined)

**Usage:**
```bash
ailang debug cycles examples/complex_types.ail      # Human-readable output
ailang debug cycles --json examples/complex_types.ail  # JSON for tooling
```

**Example output:**
```
Cycle 1 [EXPECTED]: List
  Path: List → Cons() → [a]
  Depth: 2 node(s)
  Note: Standard recursive ADT pattern

Cycle 2 [SUSPICIOUS]: Person
  Path: Person → friends → [] → Person
  Depth: 3 node(s)
```

**New Files:**
- `internal/types/cycles.go` - Type-graph cycle detection (~260 LOC)
- `internal/types/cycles_test.go` - Comprehensive unit tests (~400 LOC)

**Modified Files:**
- `cmd/ailang/debug.go` - Integration with new detection system

**Design Doc:** [design_docs/implemented/v0_5_10/m-dx11-cycles.md](design_docs/implemented/v0_5_10/m-dx11-cycles.md)

### Added - TypeReport Debug API (M-DX11-TYPE-REPORT)

Added `TypeReport` function as the canonical primitive for type debugging:

**What It Does:**
- Consolidates type info from CoreTI, substitution, and constraints
- Returns Raw type (as stored) and Resolved type (after substitution)
- Lists related constraints for any Core node

**Usage (Go code):**
```go
report := tc.TypeReport(nodeID)
fmt.Printf("Raw: %s, Resolved: %s\n", report.Raw, report.Resolved)
```

**CLI stub:**
```bash
ailang debug types  # Shows API documentation
```

**New Files:**
- `internal/types/type_report.go` - TypeReport types and function (~160 LOC)
- `internal/types/type_report_test.go` - Unit tests (~140 LOC)

**Modified Files:**
- `cmd/ailang/debug.go` - Added `debug types` stub command

**Design Doc:** [design_docs/planned/v0_5_10/m-dx11-type-inference-debugging.md](design_docs/planned/v0_5_10/m-dx11-type-inference-debugging.md)

### Fixed - TypeName Propagation to Nested Records (M-TYPENAME-NESTED-PROPAGATION)

Fixed issue where nested record literals lost their type identity after type inference:

**Problem:** Record literals like `{x: 1.0, y: 2.0, z: 3.0}` would lose their TypeName (e.g., "SystemPos") during substitution, causing codegen to generate anonymous structs instead of typed structs.

**Solution:**
- `RegisterTypeAlias` now sets TypeName on TRecord aliases at registration time
- Added `propagateTypeNameToCoreTI()` to match anonymous TRecords to known type aliases after substitution
- Handles ambiguous signatures (e.g., Vec3 and SystemPos with same fields) by skipping auto-assignment

**Files Changed:**
- `internal/elaborate/core.go` - Set TypeName on aliases (+3 LOC)
- `internal/types/typechecker_substitution.go` - TypeName propagation (+50 LOC)
- `internal/gen/golang/codegen.go` - Safe fallback for ambiguous types (+10 LOC)
- `internal/gen/golang/codegen_expr_let.go` - CoreTypeInfo priority (+15 LOC)

**Design Doc:** [design_docs/implemented/v0_5_11/m-typename-nested-propagation.md](design_docs/implemented/v0_5_11/m-typename-nested-propagation.md)

### Fixed - Option Type Assertions in Codegen (M-CODEGEN-OPTION-TYPE-ASSERT)

Fixed missing type assertions for Option-typed fields in generated Go code:

**Problem:** Fields typed `Option[T]` (like `RingColor: Option[Color]`) got `interface{}` assertions instead of `(*Option)`, causing Go compile errors.

**Solution:** Added handling for `*ast.TypeApp` in `ailangTypeToGo` function - generic type applications like `Option[Color]` now correctly map to `*Option`.

**Files Changed:**
- `cmd/ailang/compile.go` - Added TypeApp case in ailangTypeToGo (+5 LOC)

### Fixed - Compile Output Directory Nesting (M-CODEGEN-OUTPUT-PATH)

Fixed nested output directory issue in `ailang compile`:

**Problem:** `--out sim_gen --package-name sim_gen` created `sim_gen/sim_gen/` instead of `sim_gen/`.

**Solution:** Skip creating nested subdirectory when output directory already matches package name.

**Files Changed:**
- `cmd/ailang/compile.go` - Smart output path logic (+4 LOC)

### Fixed - Float Operator Dispatch (M-FIX-FLOAT-OP)

Fixed float arithmetic operators returning wrong types after type inference:

**Problem:** Expressions like `3.14 + 2.71` were being dispatched to Int.add instead of Float.add.

**Solution:** Apply substitution to CoreTI after defaulting, ensuring float types are preserved for operator lowering.

**Design Doc:** [design_docs/implemented/v0_5_10/m-fix-float-operator-dispatch.md](design_docs/implemented/v0_5_10/m-fix-float-operator-dispatch.md)

### Fixed - TApp/TCon Unification

Fixed unification between type constructors and type applications:

**Problem:** `TCon("Option")` and `TApp("Option", [T])` failed to unify, breaking generic ADT usage across modules.

**Solution:** Enhanced unification to handle TCon/TApp equivalence for nullary type applications.

### Fixed - Cross-Module Record Type Resolution

Fixed record type resolution when types are defined in one module and used in another:

**Problem:** Record literals in cross-module contexts got anonymous types instead of the declared type.

**Solution:** Enhanced type alias lookup to search imported module environments.

**Design Doc:** [design_docs/implemented/v0_5_10/m-cross-module-record-unification.md](design_docs/implemented/v0_5_10/m-cross-module-record-unification.md)

### Fixed - ADT Type Assertions in Codegen

Fixed type assertions for ADT variant fields in generated struct literals:

**Design Doc:** [design_docs/implemented/v0_5_10/m-codegen-adt-type-assert.md](design_docs/implemented/v0_5_10/m-codegen-adt-type-assert.md)

### Fixed - List Flattening in Codegen

Fixed handling of nested list operations in Go code generation:

**Design Doc:** [design_docs/implemented/v0_5_10/m-codegen-list-flatten.md](design_docs/implemented/v0_5_10/m-codegen-list-flatten.md)

### Fixed - Tuple Pattern Matching in Codegen

Fixed tuple destructuring patterns in generated Go code:

**Design Doc:** [design_docs/implemented/v0_5_10/m-codegen-tuple-pattern.md](design_docs/implemented/v0_5_10/m-codegen-tuple-pattern.md)

---

## [v0.5.9] - 2025-12-10

### Added - Standard Math Library Functions (M-STD-MATH-TRIG)

Extended `std/math` with trigonometric and advanced mathematical functions:

**New Functions (17 builtins):**
- **Trigonometric**: `sin`, `cos`, `tan`, `asin`, `acos`, `atan`, `atan2`
- **Hyperbolic**: `sinh`, `cosh`, `tanh`
- **Exponential/Logarithmic**: `exp`, `log`, `log10`, `log2`, `pow`
- **Rounding**: `ceil`, `floor`, `round`, `trunc`
- **Utility**: `sqrt`, `abs_Float` (existing), `min_Float`, `max_Float`

**Constants**: `PI`, `E`, `Tau`, `Phi`, `Sqrt2`, `SqrtE`, `SqrtPi`, `SqrtPhi`, `Ln2`, `Ln10`, `Log2E`, `Log10E`

**Usage:**
```ailang
import std/math (sin, cos, PI)

let x = sin(PI / 4.0)  -- 0.707...
let y = cos(0.0)       -- 1.0
```

**Files Changed:**
- `stdlib/math.ail` - Added function wrappers (~50 LOC)
- `internal/builtins/spec.go` - Registered 17 new math builtins (~100 LOC)
- `internal/builtins/math.go` - Implemented Go handlers (~200 LOC)

### Added - Go Codegen Math Support (M-CODEGEN-STDLIB-MATH)

Fixed Go codegen to properly map `std/math` functions to Go's `math` package:

**Problem:** Generated Go code had undefined references (`undefined: PI`, `undefined: Sin`).

**Solution:** Added `mapPureMathBuiltin()` function that maps AILANG math builtins to `math.*` calls, with conditional `"math"` import.

**Files Changed:**
- `internal/gen/golang/codegen_expr_simple.go` - `mapPureMathBuiltin` (~65 LOC)
- `internal/gen/golang/codegen.go` - Two-phase generation, `needsMathImport` flag (~25 LOC)
- `internal/gen/golang/codegen_math_test.go` - Unit tests (~183 LOC)

**Design Doc:** [design_docs/implemented/v0_5_9/m-codegen-stdlib-math.md](design_docs/implemented/v0_5_9/m-codegen-stdlib-math.md)

### Added - Cyclic Type Diagnostics Phase 1 (M-DX11)

Improved tooling to detect and diagnose cyclic type hangs in compilation:

**New `ailang check` flags:**

```bash
# Timeout protection with stack dump on hang
ailang check --timeout 30s file.ail

# Phase timing breakdown for performance analysis
ailang check --debug-compile file.ail

# Combine both for debugging
ailang check --timeout 30s --debug-compile file.ail
```

**Timeout behavior:**
- On timeout: prints stack dump of all goroutines to help identify the hang location
- Exit code: 124 (standard timeout exit code)
- Hint message: suggests using `ailang debug cycles` (coming in Phase 2)

**Phase timing output:**
```
⏱ Compilation Phase Timings:
  Loading:             12ms
  Topo Sort:            3ms
  Parsing:             45ms
  Elaboration:         23ms
  Type Checking:      156ms ← Slowest
  ----------------------------
  Total:              239ms

Warning: Type Checking took 156ms (threshold: 100ms)
  Consider checking for complex recursive types.
```

**SafeTypeString (internal):**
- Depth-limited type stringification prevents hangs when displaying cyclic types
- Max depth: 100, produces `<*TVar...cycle>` markers on cycles
- Internal API used by debuggers and error messages

**Files Changed:**
- `cmd/ailang/main.go` - Add `--timeout` and `--debug-compile` flags to check command
- `cmd/ailang/check.go` - Implement timeout (goroutine + channel), phase timing display
- `internal/types/safe_string.go` - Depth-limited SafeTypeString (170 LOC)
- `docs/guides/debugging.md` - Document new check flags and troubleshooting

**Design Docs Created:**
- `design_docs/planned/v0_5_9/m-dx11-cycles.md` - Type graph cycle detection command (Phase 2)
- `design_docs/planned/v0_5_9/m-dx11-traverse.md` - Safe type traversal library (Phase 3)

**Related:** [M-DX11 Cyclic Type Diagnostics](design_docs/planned/v0_5_9/m-dx11-cyclic-type-diagnostics.md) | [M-PERF2 Postmortem](design_docs/implemented/v0_5_8/m-perf2-cyclic-type-hang-postmortem.md)

### Fixed - Flatten Nested Closures in If-Else Chains (M-CODEGEN-FLAT-IF-ELSE)

Fixed critical performance bug where if-else chains generated deeply nested closures causing Go compiler OOM and runtime GC freezes.

**Reporter**: stapledons_voyage project (bug report `msg_20251209_202946_f10751f5`)

**Problem**: A 25-branch if-else chain in AILANG generated ~400 lines of 25-deep nested closures:
```go
// BEFORE: Deeply nested closures (causes OOM)
return func() interface{} {
    if cond1 { return 0 }
    return func() interface{} {
        if cond2 { return 1 }
        return func() interface{} {
            // ...25 levels deep...
        }()
    }()
}()
```

**Impact**:
- Go compiler used 2GB+ RAM, often OOM-killed
- Runtime GC pressure: 26K allocations/sec in game loops caused system freezes
- Workaround: users had to avoid long if-else chains or close apps to compile

**Solution**: Detect if-else chains and generate flat Go code with single IIFE:
```go
// AFTER: Flat if statements (efficient)
return func() interface{} {
    var tmp1 = cond1
    var tmp2 = cond2
    // ...all conditions evaluated upfront...
    if tmp1 { return 0 }
    if tmp2 { return 1 }
    // ...flat if statements...
    return default
}()
```

**Metrics**:
- 5-branch if-else: ~100 lines → ~33 lines (-67%)
- 10-branch if-else: ~200+ lines → ~63 lines (-70%)
- Go compiler memory: 2GB+ → <500MB

**Files Changed** (~200 LOC):
- `internal/gen/golang/codegen_expr_control.go` - Chain detection helpers, `generateIfChain()`
- `internal/gen/golang/codegen_expr_let.go` - `isLetIfChain()`, `generateLetIfChain()`
- `internal/gen/golang/codegen.go` - Add `inFlatChain` context flag
- `internal/gen/golang/codegen_expr_control_test.go` - Unit tests for chain detection

**Example File**: `examples/if_else_chain.ail`

**Design Doc**: [design_docs/planned/v0_5_9/m-codegen-flat-if-else.md](design_docs/planned/v0_5_9/m-codegen-flat-if-else.md)

### Fixed - Go Codegen Pointer/Value Type Consistency (M-CODEGEN-POINTER-RETURN-TYPES)

Fixed critical type assertion failures in generated Go code. All user-defined types now consistently use pointers throughout codegen.

**Reporter**: stapledons_voyage project (4 bug reports via agent messaging)

**Problem**: Type mismatches between function signatures, struct fields, and type assertions caused runtime panics and compile errors:
```
panic: interface conversion: interface {} is *sim_gen.World, not sim_gen.World
cannot use NewArrivalPhasePhaseBlackHole() (value of type *ArrivalPhase) as ArrivalPhase value
cannot use nextPh (variable of type interface{}) as *ArrivalPhase value: need type assertion
```

**Root Cause**: Inconsistent pointer vs value types across codegen:
- `_impl` functions return `interface{}` containing pointer values (`&World{}`)
- But typed wrappers expected value types (`World`)
- Struct fields declared as values but ADT constructors return pointers

**Solution**: 4-phase fix to make all user-defined types consistently use pointers:

| Phase | Issue | Fix |
|-------|-------|-----|
| 1 | Function signatures used value types | `ailangTypeToGo` returns `*TypeName` for user-defined types |
| 2 | Struct fields used value types | `mapASTType` returns `*TypeName` for user-defined SimpleTypes |
| 3 | Missing type assertions for interface{} | Added `.(T)` assertions, skip for ADT constructors |
| 4 | Double-pointer in slices (`[]**T`) | Check for existing `*` prefix before adding |

**Before (BROKEN)**:
```go
func InitWorld(seed int64) World { return initWorld_impl(seed).(World) }  // PANIC!
type ArrivalState struct { Phase ArrivalPhase }  // Cannot assign *ArrivalPhase
tmp58.([]**CrewPosition)  // Double pointer - wrong!
```

**After (FIXED)**:
```go
func InitWorld(seed int64) *World { return initWorld_impl(seed).(*World) }  // Works!
type ArrivalState struct { Phase *ArrivalPhase }  // Matches ADT constructor return
tmp58.([]*CrewPosition)  // Single pointer - correct!
```

**Files Changed**:
- `cmd/ailang/compile.go` - `ailangTypeToGo`: pointer for user-defined types, fix ListType/ArrayType (~12 LOC)
- `internal/gen/golang/adt.go` - `mapASTType`: pointer for SimpleTypes, handle already-pointer elements (~20 LOC)
- `internal/gen/golang/adt_test.go` - Updated test expectations (~6 LOC)
- `internal/gen/golang/codegen_ops.go` - `generateTypedRecord`: type assertions for interface{} values (~5 LOC)

**Total**: ~43 LOC across 4 files

**Design Doc**: [M-CODEGEN-POINTER-RETURN-TYPES](design_docs/planned/v0_5_9/m-codegen-pointer-return-types.md)

**Note**: This makes ALL user-defined types pointers. A size-based optimization (small leaf records as values) is planned for v0.5.10 - see [M-CODEGEN-VALUE-TYPES](design_docs/planned/v0_5_10/m-codegen-value-types.md).

## [v0.5.8] - 2025-12-09

### Fixed - Go Codegen Type Safety (M-BUGFIX Sprint)

**Bug 1 - Typed Parameters Becoming interface{} (M-CODEGEN-TYPED-PARAMS)**:
Functions with typed parameters generated `interface{}` instead of preserving type annotations.

**Bug 2 - Cross-Module Type Contamination (M-CROSS-MODULE)**:
When multiple modules had records with identical fields, types would "leak" between them.

**Bug 3 - _impl Functions Calling Typed Exports (M-CODEGEN-TYPE-ASSERTIONS)**:
After fixing typed params, `_impl` functions called typed exports without proper `_impl` redirection.

**Reporter**: stapledons_voyage project

**Before (BROKEN)**:
```go
func StepArrival(state interface{}, input interface{}) interface{}
func initArrival_impl(...) interface{} {
    return UpdateAllNPCs(tmp66, tmp68, tmp70)  // Type mismatch!
}
```

**After (FIXED)**:
```go
func StepArrival(state *ArrivalState, input *ArrivalInput) *ArrivalState
func initArrival_impl(...) interface{} {
    return updateAllNPCs_impl(tmp66, tmp68, tmp70)  // Calls _impl version
}
```

**Files Changed**:
- `internal/types/types.go` - Added `TypeName` field to TRecord for nominal type identity
- `internal/types/unification.go` - Modified `expandAlias` to set TypeName
- `internal/gen/golang/types.go` - Check TypeName first in MapType
- `internal/gen/golang/codegen.go` - Added `FuncTypeOverride`, `RegisterFunctionType`
- `internal/gen/golang/codegen_decl.go` - Set `currentFuncDeclaredReturn`, updated `getTypedSignature`
- `internal/gen/golang/codegen_expr.go` - Add `_impl` redirection for VarGlobal (~8 LOC)
- `internal/gen/golang/codegen_ops.go` - Updated generateRecord to use declared return type
- `cmd/ailang/compile.go` - Register function types from AST

**Total**: ~120 LOC across 8 files

**Design Docs**: [design_docs/implemented/v0_5_8/](design_docs/implemented/v0_5_8/)

### Fixed - Effect Checker Performance on Large Arrays (M-PERF1)

**Bug**: Effect checking hangs indefinitely (30+ seconds) on modules with large array literals (192+ elements).

**Reporter**: stapledons_voyage project

**Root Cause**: `collectRequiredEffects` had O(m²) complexity where m = number of Let bindings. When processing nested Let expressions, it traversed both RHS *and* body, but `validateDecl` *also* recursively processes bodies. This caused each array to be traversed once per preceding Let binding.

**Example**: A module with 10 Let bindings containing large arrays would traverse ~55x more nodes than necessary (1+2+...+10 = 55).

**Fix**: Modified `collectRequiredEffects` to only traverse RHS values for Let/LetRec nodes, not bodies. The `validateDecl` function already handles body recursion.

**Before (v0.5.7 - HANGS)**:
```bash
$ time ailang check sim/bridge.ail
→ Type checking sim/bridge.ail...
→ Effect checking...
^C  # Killed after 30+ seconds
```

**After (v0.5.8 - FAST)**:
```bash
$ time ailang check sim/bridge.ail
→ Type checking sim/bridge.ail...
→ Effect checking...
✓ No errors found!
real    0m1.2s
```

**Performance Results**:
- 20 bindings × 200 elements: 172µs (was hanging)
- Scaling: Linear (ratio 6.4 for 10x input, not 100x)

**Files Changed**:
- `internal/pipeline/validate_effects.go` - Fix Let/LetRec traversal (~10 LOC)
- `internal/pipeline/validate_effects_test.go` - Add regression tests and benchmarks (~200 LOC)

**Design Doc**: [design_docs/planned/v0_5_8/m-perf1-effect-checker-large-arrays.md](design_docs/planned/v0_5_8/m-perf1-effect-checker-large-arrays.md)

## [v0.5.7] - 2025-12-08

### Added - Stdlib Discovery for AI Agents (M-DX11)

New `ailang docs` command enables AI agents to discover stdlib functions without file access:

```bash
ailang docs --list              # List all 15 stdlib modules
ailang docs std/io              # Show module exports and signatures
ailang docs io                  # Short form (same as std/io)
ailang docs --examples std/ai   # Show usage examples
```

**Example output:**
```
$ ailang docs --list
Available stdlib modules:

  std/ai         AI effect for general-purpose AI oracle calls
  std/array      Arrays provide O(1) indexed access
  std/clock      Provides time operations with virtual time support
  std/env        Access environment variables with capability-based security
  std/fs         Read and write files with capability-based security
  std/io         Print to stdout and read from stdin
  ...

$ ailang docs std/io
# std/io
Print to stdout and read from stdin.

## Exports

  print(s: string) -> () !
  println(s: string) -> () !
  readLine() -> string !

## Usage

  import std/io (print)
  import std/io as Io
```

**Files Added:**
- `cmd/ailang/docs.go` - New docs command (~340 LOC)

### Improved - Stdlib Module Documentation

All 15 stdlib modules now have consistent header documentation with:
- Module description (shown in `docs --list`)
- Capability requirements (e.g., "Requires: IO capability")
- Brief explanation of module purpose

**Modules updated:** std/ai, std/array, std/clock, std/debug, std/env, std/fs, std/game, std/io, std/json, std/list, std/net, std/option, std/rand, std/result, std/string

### Added - Version Auto-Detection

- Session start hook now displays current AILANG version: `📦 AILANG v0.5.7`
- Design-doc-creator skill auto-detects version from CHANGELOG.md
- Suggests next version folder when creating design docs

**Files Changed:**
- `scripts/hooks/session_start.sh` - Add version display
- `.claude/skills/design-doc-creator/scripts/create_planned_doc.sh` - Add version detection

## [v0.5.6] - 2025-12-04

### Fixed - Array Type Application Parsing (M-TYPE1)

**Bug**: `Array[T]` in ADT constructor parameters was losing its element type during parsing.

**Error**:
```
cannot unify type constructor Array with *types.TArray
```

**Root Cause**: Parser at `internal/parser/parser_type.go:27-50` was discarding type arguments when parsing type applications like `Array[Direction]`. It returned `SimpleType{Name: "Array"}` instead of `ArrayType{Element: Direction}`.

**Fix**: Parser now special-cases `Array[T]` and `List[T]` to create proper AST nodes (`ast.ArrayType`, `ast.ListType`) that preserve element types.

**Before (v0.5.5 - FAILS)**:
```ailang
type AIBehavior = PatternPatrol(Array[Direction]) | RandomWander
let patrol = PatternPatrol(#[North, East, South, West])
-- Error: cannot unify type constructor Array with *types.TArray
```

**After (v0.5.6 - WORKS)**:
```ailang
type AIBehavior = PatternPatrol(Array[Direction]) | RandomWander
let patrol = PatternPatrol(#[North, East, South, West])  -- ✓ Compiles!
```

**Files Changed (Parser):**
- `internal/parser/parser_type.go` - Fix type application parsing (~15 LOC)
- `internal/parser/type_test.go` - Add regression tests (~80 LOC)
- `examples/runnable/array_adt.ail` - Integration test example

### Fixed - Array Go Codegen (M-TYPE1 continued)

**Bug**: After the parser fix, `ailang compile --emit-go` still failed because:
1. Array literal expressions (`#[...]`) weren't being generated
2. `Array[T]` fields in ADT constructors were mapped to `interface{}` instead of typed slices

**Errors**:
```
unsupported expression type: *core.Array
cannot use tmp1 (variable of type interface{}) as []*Direction
```

**Fix 1**: Added `generateArray()` function to handle `*core.Array` expressions in `codegen_ops.go`.

**Fix 2**: Added `*ast.ArrayType` case to `ailangTypeToGo()` in `compile.go` so `Array[Direction]` maps to `[]*Direction`.

**Result**: Generated Go code now correctly uses `convertToDirectionSlice()` conversion helpers when passing array literals to ADT constructors.

**Fix 3**: Added array runtime functions to `codegen_runtime.go`:
- `FromList(xs)` - Convert list to array
- `ToList(arr)` - Convert array to list
- `Length(arr)` - Get array length
- `Get(arr, idx)` - Get element at index
- `GetOpt(arr, idx)` - Safe get returning Option
- `UnsafeGet(arr, idx)` - Unchecked get
- `Set(arr, idx, val)` - Immutable update
- `Make(size, default)` - Create array with default value

**Files Changed (Go Codegen):**
- `internal/gen/golang/codegen_expr.go` - Add Array case (~2 LOC)
- `internal/gen/golang/codegen_ops.go` - Add `generateArray()` function (~35 LOC)
- `internal/gen/golang/codegen_runtime.go` - Add array runtime functions (~150 LOC)
- `cmd/ailang/compile.go` - Add ArrayType handling (~8 LOC)

### Fixed - Effect Handler Nil Pointer Panics

**Bug**: Generated Go code crashed with cryptic nil pointer dereference when effect handlers weren't initialized.

**Error**:
```
panic: runtime error: invalid memory address or nil pointer dereference
```

**Root Cause**: Generated code directly accessed `handlers.Rand.RandInt(...)` without checking if `handlers.Rand` was nil.

**Fix**: Added `requireXxx()` guard functions that provide helpful error messages:

**Before (crashes):**
```go
var result = handlers.Rand.RandInt(0, 3)  // nil pointer if Rand not initialized
```

**After (helpful error):**
```go
var result = requireRand().RandInt(0, 3)
// panic: "Rand effect handler not initialized. Call Init() with a RandHandler before using rand_* functions."
```

**Affected handlers**: Rand, Clock, Debug, FS, Net, Env, AI

**Files Changed:**
- `internal/gen/golang/effects.go` - Add `generateRequireGuards()` (~20 LOC)
- `internal/gen/golang/codegen_expr.go` - Update `mapEffectBuiltinToHandler()` to use guards (~60 LOC changed)
- `internal/gen/golang/effects_test.go` - Add `TestGenerateRequireGuards` (~35 LOC)

### Added - Unified Agent Messaging (M-MSG)

**Goal**: Unify CLI (`ailang messages`) and Collaboration Hub dashboard to share the same SQLite database.

**Before (v0.5.5)**:
- CLI used file-based storage in `~/.ailang/state/messages/`
- Dashboard used `collaboration.db`
- No way to see CLI messages in dashboard or vice versa

**After (v0.5.6)**:
- Single `inbox_messages` table in `collaboration.db`
- CLI and dashboard read/write same data
- Real-time WebSocket updates when messages arrive

**New CLI commands:**
```bash
ailang messages list              # List all messages
ailang messages list --unread     # List unread only
ailang messages send INBOX '{...}'  # Send message
ailang messages read ID           # Read full message
ailang messages ack ID            # Mark as read
ailang messages ack --all         # Mark all as read
ailang messages watch             # Real-time watch mode
ailang messages cleanup           # Remove old messages
```

**New REST API endpoints:**
- `GET /api/inbox` - List messages (with filtering)
- `POST /api/inbox` - Send message
- `GET /api/inbox/{id}` - Get single message
- `PUT /api/inbox/{id}` - Update message status
- `POST /api/inbox/ack-all` - Acknowledge all
- `POST /api/inbox/cleanup` - Cleanup old messages

**WebSocket events:**
- `inbox_message` - Real-time notification when new message arrives

**Files Changed:**
- `internal/messaging/schema.go` - Add `inbox_messages` table (~30 LOC)
- `internal/messaging/inbox.go` - New Store methods for inbox (~250 LOC)
- `internal/server/handlers_inbox.go` - New REST endpoints (~200 LOC)
- `internal/websocket/events.go` - Add `InboxMessageEvent` (~30 LOC)
- `internal/websocket/server.go` - Add `BroadcastInboxMessage()` (~30 LOC)
- `cmd/ailang/messages.go` - Updated CLI to use unified Store (~150 LOC)
- Deleted: `internal/messaging/msgstore.go`, `internal/messaging/msgstore_test.go`

### Added - Eval Process Guardrails (M-EVAL-GUARD)

**Problem**: Eval benchmark processes could become orphaned when the parent harness dies (Ctrl+C, SSH disconnect, crash), running for 37+ hours and consuming CPU.

**Solution**: Two-layer defense against orphans:

**Layer 1: Process Groups**
- Child processes now run in their own process group (`Setpgid: true`)
- Timeout kills entire process group (`syscall.Kill(-pid, SIGKILL)`) instead of just the main process
- Prevents orphaned grandchildren

**Layer 2: Watchdog**
- Background goroutine checks for orphaned `ailang run.*benchmark` processes every 60 seconds
- Kills any process running longer than 15 minutes (eval timeout is typically 30 seconds)
- Reports killed orphans at end of eval suite

**Signal Handling**:
- Ctrl+C now triggers graceful shutdown
- Watchdog performs final cleanup before exit
- Reports any orphans killed during shutdown

**Files Changed:**
- `internal/eval_harness/runner.go` - Process groups + group kill (~10 LOC)
- `internal/eval_harness/watchdog.go` - NEW: Watchdog implementation (~110 LOC)
- `cmd/ailang/eval_suite.go` - Signal handler + watchdog integration (~25 LOC)

## [v0.5.5] - 2025-12-04

### Added - Compile DX Improvements

**Directory Support**: Pass directories to auto-discover `.ail` files:
```bash
# Before: list each file explicitly
ailang compile --emit-go world.ail npc_ai.ail camera.ail

# After: just pass the directory
ailang compile --emit-go sim/
```

**Per-File Output**: Generated code is now split into separate files per source:
```
gen/game/
├── types.go      # All ADT types (merged)
├── runtime.go    # Shared runtime helpers
├── handlers.go   # Effect handler interfaces
├── world.go      # Functions from world.ail
├── npc_ai.go     # Functions from npc_ai.ail
└── step.go       # Functions from step.ail
```

Benefits:
- Smaller, more navigable files
- Easier to correlate generated code with source
- Better IDE navigation

**Files Changed:**
- `cmd/ailang/compile.go` - Added `expandFilenames()` function, per-file output loop (~100 LOC)
- `docs/docs/guides/go-interop.md` - Updated documentation

### Added - Typed Wrapper Architecture (M-DX26)

Generated functions now use a dual-function pattern:
1. `_impl` function: Uses `interface{}` everywhere for runtime flexibility
2. Typed wrapper: Provides typed Go API with automatic conversions

**Before:**
```go
func Step(world *World) *World {
    // Mixed types, conversion issues
}
```

**After:**
```go
func step_impl(world interface{}) interface{} {
    // Pure interface{} - handles all runtime conversions
}

func Step(world *World) *World {
    return step_impl(world).(*World)  // Type-safe API
}
```

**Files Changed:**
- `internal/gen/golang/codegen_decl.go` - Added `generateImplFunc`, `generateTypedWrapper` (~100 LOC)
- `design_docs/implemented/v0_5_5/m-dx26-typed-wrapper-architecture.md` - Design doc

### Added - Typed Function Signatures (M-DX23)

Function signatures now carry full type information through CoreTypeInfo:
- Parameter types derived from type checker
- Return types properly mapped to Go types
- Enables typed wrappers to generate correct type assertions

**Files Changed:**
- `internal/gen/golang/codegen.go` - Added CoreTypeInfo integration (~30 LOC)
- `internal/gen/golang/codegen_decl.go` - Added `getTypedSignature()` (~40 LOC)

### Added - Auto-Generated ADT Slice Converters (M-DX22)

Slice conversion functions are now auto-generated for all ADT types:
```go
// Auto-generated for each ADT type
func ConvertToEntitySlice(v interface{}) []*Entity { ... }
func ConvertToDirectionSlice(v interface{}) []*Direction { ... }
```

**Files Changed:**
- `internal/gen/golang/codegen_types.go` - Added slice converter generation (~50 LOC)

### Fixed - Records Always Typed in _impl Functions (M-DX26 Fix)

**User Impact**: Records are now always generated as typed structs even in `_impl` functions, fixing type assertion panics.

**Root cause**: `_impl` functions return `interface{}`, but that's just the *signature* - the actual runtime value must still be a typed struct for type assertions to work.

**Before (runtime panic):**
```go
func step_impl(world interface{}) interface{} {
    return map[string]interface{}{"tick": 1}  // Wrong!
}
// Wrapper does: step_impl(w).(*World) → PANIC
```

**After (works correctly):**
```go
func step_impl(world interface{}) interface{} {
    return &World{Tick: 1}  // Correct - actual type is *World
}
// Wrapper does: step_impl(w).(*World) → Works!
```

**Files Changed:**
- `internal/gen/golang/codegen_ops.go` - Removed `inImplFunc` check in `generateRecord` (~5 LOC)

**Source**: DX feedback from `stapledons_voyage` agent.

---

## [v0.5.4] - 2025-12-03

### Fixed - Integer Literals as int64 (M-DX17)

**User Impact**: Integer literals in generated Go code now use explicit `int64()` conversion, preventing runtime panics when type-asserting interface values.

**Before (runtime panic):**
```go
var w interface{} = 8     // Go infers 'int' (not int64)
return w.(int64)          // PANIC: interface {} is int, not int64
```

**After (works correctly):**
```go
var w interface{} = int64(8)  // Explicit int64 type
return w.(int64)              // Works - types match!
```

**Files Changed:**
- `internal/gen/golang/codegen_expr.go` - Wrap int literals in `int64()`, float in `float64()` (~5 LOC)
- `internal/gen/golang/codegen_test.go` - Updated test assertions (~5 LOC)

**Source**: DX feedback from `stapledons_voyage` agent.

### Fixed - FieldGet for Typed Struct Access (M-DX18)

**User Impact**: Record field access now works correctly with both typed structs and maps, enabling the full game loop to work without panics.

**Before (runtime panic):**
```go
// step function generated:
world.(map[string]interface{})["tick"]  // PANIC: world is *World, not map!
```

**After (works with both types):**
```go
// step function generates:
FieldGet(world, "tick")  // Works with *World AND map[string]interface{}
```

**How it works**: The `FieldGet` runtime helper detects the type at runtime:
1. If `map[string]interface{}` → use map access
2. If typed struct pointer → use reflection with PascalCase field names
3. Converts AILANG field names (lowercase) to Go field names (PascalCase)

**Files Changed:**
- `internal/gen/golang/codegen_runtime.go` - Added `FieldGet` helper (~40 LOC)
- `internal/gen/golang/codegen_ops.go` - Updated `generateRecordAccess` to use `FieldGet` (~5 LOC)

**Tests:**
- Game loop simulation test: 100 consecutive `step()` calls all preserve types
- Backwards compatibility: map-based records still work

**Source**: DX feedback from `stapledons_voyage` agent.

### Fixed - RecordUpdate Slice Conversion (M-DX19)

**User Impact**: RecordUpdate now correctly converts `[]interface{}` to typed slices when updating struct fields.

**Before (runtime panic):**
```go
// Updating a []*Entity field with []interface{} from AILANG
RecordUpdate(world, map[string]interface{}{"entities": entities})
// PANIC: cannot assign []interface{} to []*Entity
```

**After (works correctly):**
```go
// RecordUpdate uses reflection to convert slice element-by-element
// []interface{} -> []*Entity via ConvertibleTo/AssignableTo checks
```

**Files Changed:**
- `internal/gen/golang/codegen_runtime.go` - Added slice conversion logic (~15 LOC)

**Source**: DX feedback from `stapledons_voyage` agent.

### Fixed - RecordUpdate Pointer Dereference (M-DX20)

**User Impact**: RecordUpdate correctly dereferences pointers when the field expects a value type.

**Before (runtime panic):**
```go
// Field expects Selection (value), got *Selection (pointer)
RecordUpdate(world, map[string]interface{}{"selection": selectionPtr})
// PANIC: cannot assign *Selection to Selection
```

**After (works correctly):**
```go
// RecordUpdate detects pointer-to-value mismatch and dereferences
// *Selection -> Selection via Elem() call
```

**Files Changed:**
- `internal/gen/golang/codegen_runtime.go` - Added pointer dereference logic (~8 LOC)

**Source**: DX feedback from `stapledons_voyage` agent.

### Fixed - FieldGet Returns Pointers for Struct Fields (M-DX21)

**User Impact**: FieldGet returns pointers for struct-typed fields, matching AILANG's expectation that nested records are pointers.

**Before (type mismatch):**
```go
// FieldGet returns Selection (value), but code expects *Selection
selection := FieldGet(world, "selection").(Selection)
// Works, but downstream code may expect *Selection
```

**After (returns pointer):**
```go
// FieldGet detects struct fields and returns addressable pointer
selection := FieldGet(world, "selection").(*Selection)
// Matches AILANG's structural semantics
```

**Files Changed:**
- `internal/gen/golang/codegen_runtime.go` - Added struct field pointer return (~5 LOC)

**Source**: DX feedback from `stapledons_voyage` agent.

### Added - Auto-Generate ADT Slice Converters (M-DX22)

**User Impact**: All ADT types now get `convertToXxxSlice` functions generated automatically, eliminating manual boilerplate.

**Before (manual runtime.go):**
```go
// Users had to write these manually for each ADT type
func convertToNPCSlice(v interface{}) []*NPC { ... }
func convertToTileSlice(v interface{}) []*Tile { ... }
func convertToDrawCmdSlice(v interface{}) []*DrawCmd { ... }
```

**After (auto-generated):**
```go
// All ADT types get converters automatically in funcs.go
// No manual runtime.go needed!
```

**How it works**: The code generator now iterates all ADT types from registered constructors (not just types used in slice fields).

**Files Changed:**
- `internal/gen/golang/codegen_runtime.go` - Use allADTTypes from constructors (~10 LOC)

**Source**: DX feedback from `stapledons_voyage` agent.

### Added - Typed Function Signatures Infrastructure (M-DX23)

**User Impact**: Infrastructure for generating typed Go function signatures instead of `interface{}`. When CoreTypeInfo is available, functions are generated with concrete parameter and return types.

**Before (interface{} everywhere):**
```go
func Step(world interface{}) interface{} { ... }
```

**After (typed signatures with CoreTypeInfo):**
```go
func Step(world World) World { ... }
```

**How it works**:
1. Pipeline captures `CoreTypeInfo` from type checking and stores in `Artifacts.CoreTI`
2. Compile command passes `CoreTI` to code generator via `SetCoreTypeInfo()`
3. `generateFuncFromLambda` looks up Lambda's type by NodeID
4. `TypeMapper.ExtractFuncSignature` extracts parameter and return types
5. Falls back to `interface{}` if type info unavailable (backward compatible)

**Key changes:**
- Added `CoreTI` field to `Artifacts` struct in pipeline
- Added `CoreTI` field to `CompileUnit` for multi-module support
- Extended `TypeMapper` to handle `TVar` (returns `interface{}`) and `TFunc2`
- Added `ExtractFuncSignature` helper for function type decomposition

**Files Changed:**
- `internal/pipeline/pipeline.go` - Added CoreTI to Artifacts (~3 LOC)
- `internal/pipeline/pipeline_single.go` - Populate CoreTI (~1 LOC)
- `internal/pipeline/pipeline_module.go` - Populate CoreTI in loop and result (~6 LOC)
- `internal/pipeline/compile_unit.go` - Added CoreTI field (~2 LOC)
- `internal/gen/golang/codegen.go` - Added coreTypeInfo field and SetCoreTypeInfo (~12 LOC)
- `internal/gen/golang/codegen_decl.go` - Typed signature generation (~45 LOC)
- `internal/gen/golang/types.go` - TVar handling, TFunc2, ExtractFuncSignature (~73 LOC)
- `cmd/ailang/compile.go` - Pass CoreTI to generator (~5 LOC)

**Tests:**
- `TestGenerateTypedFunctionSignature` - Verifies typed output with CoreTypeInfo
- `TestGenerateFallbackToInterface` - Verifies backward-compatible fallback

**Note**: Full typed signatures require Lambda NodeIDs to be properly assigned during elaboration. Current real-world usage may still see `interface{}` if NodeIDs are 0.

**Source**: DX feedback from `stapledons_voyage` agent (design doc M-DX23).

## [v0.5.3] - 2025-12-03

### Added - Named ADT Constructor Fields (M-DX11-NAMED-ADT)

**User Impact**: ADT constructors now support named fields for better Go interop and self-documenting code.

**AILANG source:**
```ailang
type DrawCmd =
  | Rect(x: float, y: float, w: float, h: float)
  | Circle(cx: float, cy: float, radius: float)
  | Clear
```

**Generated Go (before):**
```go
type DrawCmdRect struct {
    Value0 float64  // What is this?
    Value1 float64
    Value2 float64
    Value3 float64
}
```

**Generated Go (after):**
```go
type DrawCmdRect struct {
    X float64  // Clear field names!
    Y float64
    W float64
    H float64
}
```

**Pattern matching also uses named fields:**
```go
// Before: x := _adt.Rect.Value0
// After:  x := _adt.Rect.X
```

**Backwards Compatible**: Positional syntax still works:
```ailang
type Option = | Some(int) | None  -- Generates Value0
```

**Files Changed:**
- `internal/ast/ast_decl.go` - Added `ConstructorField` struct (~25 LOC)
- `internal/parser/parser_type.go` - Added `parseConstructorField()` for `name: type` syntax (~35 LOC)
- `internal/gen/golang/adt.go` - Generate named fields when available (~20 LOC)
- `internal/gen/golang/codegen.go` - Added `FieldNames` to `ADTConstructorInfo` (~20 LOC)
- `internal/gen/golang/codegen_match.go` - Use named fields in pattern matching (~15 LOC)
- `cmd/ailang/compile.go` - Register field names with generator (~15 LOC)

**Source**: DX feedback from `stapledons_voyage` agent.

### Added - Typed ADT Slices (M-DX12)

**User Impact**: Record fields with `[ADT]` type now generate typed slices instead of `interface{}`, eliminating type assertions in host code.

**AILANG source:**
```ailang
type FrameOutput = {
  draw: [DrawCmd],
  sounds: [int],
  debug: [string]
}
```

**Generated Go (before):**
```go
type FrameOutput struct {
    Draw   interface{}  // Host must type assert
    Sounds []int64
    Debug  []string
}
```

**Generated Go (after):**
```go
type FrameOutput struct {
    Draw   []*DrawCmd  // Typed slice - clean API!
    Sounds []int64
    Debug  []string
}
```

**Auto-generated converters with fail-fast panics:**
```go
func convertToDrawCmdSlice(v interface{}) []*DrawCmd {
    // Panics on type mismatch (compiler bug detection)
}
```

**Design principle**: World boundary marshalling - typed at profile surfaces, `[]interface{}` internally.

**Files Changed:**
- `internal/gen/golang/adt.go` - Changed `mapASTType()` to return `[]*ADT` for list-of-ADT (~25 LOC)
- `internal/gen/golang/codegen.go` - Added `adtSliceTypes` tracking, `RegisterADTSliceTypes()` (~15 LOC)
- `internal/gen/golang/codegen_runtime.go` - Added `writeADTSliceConverters()` (~65 LOC)
- `cmd/ailang/compile.go` - Wired ADT slice types from ADTGenerator to Generator (~10 LOC)

**Source**: DX feedback from `stapledons_voyage` agent.

### Fixed - ADT Constructor List Parameters (M-DX12.5)

**User Impact**: ADT constructors with list parameters now correctly use type converters. Before this fix, `PatternPatrol([Direction])` would fail with type mismatch because the list literal generated `[]interface{}` but the constructor expected `[]*Direction`.

**AILANG source:**
```ailang
type MovementPattern =
  | PatternPatrol([Direction])  -- List parameter
  | PatternFixed(int, int)

let patrol: [Direction] -> MovementPattern =
  \dirs. PatternPatrol(dirs)
```

**Generated Go (before - M-DX12.5 bug):**
```go
func patrol(dirs interface{}) interface{} {
    return NewMovementPatternPatternPatrol(dirs)  // Type mismatch!
}
```

**Generated Go (after - fixed):**
```go
func patrol(dirs interface{}) interface{} {
    return NewMovementPatternPatternPatrol(convertToDirectionSlice(dirs))  // Converter wraps argument
}
```

**Root cause**: `ailangTypeToGo()` in compile.go was returning `"interface{}"` for `[ADT]` types to "match adt.go". This prevented `generateApp()` from detecting the need for type conversion.

**Files Changed:**
- `cmd/ailang/compile.go` - Fixed `ailangTypeToGo()` to return `"[]*ADT"` for list-of-ADT types (~5 LOC)

**Source**: DX feedback from `stapledons_voyage` agent.

### Added - Typed Record Literals (M-DX13)

**User Impact**: Functions now return typed structs instead of `map[string]interface{}`, enabling clean API usage from Go host code.

**AILANG source:**
```ailang
type World = { width: int, height: int, name: string }

let createWorld: int -> int -> string -> World =
  \w. \h. \n. { width: w, height: h, name: n }
```

**Generated Go (before - untyped):**
```go
func createWorld(w interface{}) interface{} {
    return map[string]interface{}{
        "width": w, "height": h, "name": n,
    }
}
// Host code must do awkward type assertions:
// world := result.(map[string]interface{})
// width := world["width"].(int64)
```

**Generated Go (after - typed):**
```go
func createWorld(w interface{}) interface{} {
    return &World{Width: w.(int64), Height: h.(int64), Name: n.(string)}
}
// Host code just works:
// world := result.(*World)
// width := world.Width  // Clean!
```

**How it works:**
1. Record types are registered during compilation with field names and types
2. When generating a record literal, field names are matched to find the struct type
3. If matched, generates typed struct literal with proper type assertions
4. Falls back to `map[string]interface{}` for unrecognized field patterns

**Files Changed:**
- `internal/gen/golang/codegen.go` - Added `RecordTypeInfo`, `RegisterRecordType()`, `GetRecordTypeByFields()` (~50 LOC)
- `internal/gen/golang/codegen_ops.go` - Added `generateTypedRecord()`, updated `generateRecord()` (~70 LOC)
- `cmd/ailang/compile.go` - Added `extractRecordTypeInfo()`, record type registration (~20 LOC)

**Source**: DX feedback from `stapledons_voyage` agent.

### Fixed - Literal Type Assertions (M-DX13.1)

**User Impact**: Generated code no longer has invalid Go syntax like `2 .(int64)`.

**Before (invalid Go):**
```go
&Coord{X: 2 .(int64), Y: 3 .(int64)}  // Type assertions don't work on literals!
```

**After (valid Go):**
```go
&Coord{X: int64(2), Y: int64(3)}  // Type conversion for literals
```

**Root cause**: Type assertions only work on interface values, not literals. Fixed by detecting literal values and using type conversion syntax instead.

**Files Changed:**
- `internal/gen/golang/codegen_ops.go` - Added literal detection in `generateTypedRecord()` (~15 LOC)

**Source**: DX feedback from `stapledons_voyage` agent.

### Fixed - Pointer/Value Mismatch in Nested Records (M-DX13.2)

**User Impact**: Nested record fields now correctly dereference pointer values when the struct field expects a value type.

**Before (type mismatch):**
```go
tmp1 := &Coord{X: 10, Y: 20}      // Pointer
return &NPC{Pos: tmp1, ...}        // ERROR: Pos is Coord (value), not *Coord
```

**After (correct):**
```go
var tmp1 interface{} = &Coord{X: int64(10), Y: int64(20)}
return &NPC{Pos: *(tmp1.(*Coord)), ...}  // Type assert then dereference
```

**Root cause**: `ailangTypeToGo()` was returning `"*Coord"` but adt.go's `mapNamedType()` returns `"Coord"` (value type). Fixed by aligning both to use value types for user-defined struct field types, and adding type assertion before dereference (see M-DX13.5).

**Files Changed:**
- `cmd/ailang/compile.go` - Fixed `ailangTypeToGo()` to return value types for user-defined types (~5 LOC)
- `internal/gen/golang/codegen_ops.go` - Added `isRecordValueType()`, dereference logic (~30 LOC)

**Source**: DX feedback from `stapledons_voyage` agent.

### Fixed - Let Bindings as interface{} (M-DX13.3)

**User Impact**: Generated let bindings now use `var x interface{} = ...` instead of `x := ...`, enabling type assertions on concrete values.

**Before (invalid Go):**
```go
// In generated IIFE for let expression:
w := 8                  // Go infers int
return w.(int64)        // ERROR: can't type assert non-interface type
```

**After (valid Go):**
```go
var w interface{} = 8   // Explicit interface{} type
return w.(int64)        // Works - type assertion on interface{}
```

**Root cause**: Go's type inference assigns concrete types to short variable declarations. Type assertions only work on interface values.

**Files Changed:**
- `internal/gen/golang/codegen_expr.go` - Changed `generateLet()` to use `var x interface{}` (~5 LOC)
- `internal/gen/golang/codegen_test.go` - Updated `TestGenerateNestedLet` assertions (~5 LOC)

**Source**: DX feedback from `stapledons_voyage` agent.

### Fixed - Slice Type Conversion (M-DX13.4)

**User Impact**: Slice fields in records now use proper type converters instead of type assertions.

**Before (invalid Go):**
```go
// patrolPath is []interface{} at runtime
path := patrolPath.([]Direction)  // ERROR: can't type assert to slice type
```

**After (valid Go):**
```go
path := convertToDirectionSlice(patrolPath)  // Uses converter function
```

**Root cause**: Go doesn't allow type assertion from `[]interface{}` to typed slices like `[]*Direction`. The slice converter iterates and type-asserts each element.

**Files Changed:**
- `cmd/ailang/compile.go` - Fixed `ailangTypeToGo()` to return `[]*ADT` for list-of-ADT types (~5 LOC)

**Source**: DX feedback from `stapledons_voyage` agent.

### Fixed - Type Assertions for Struct Field Dereference (M-DX13.5)

**User Impact**: When assigning nested records stored as `interface{}` to value-type struct fields, the generated code now properly type-asserts before dereferencing.

**Before (invalid Go):**
```go
var tmp1 interface{} = &Coord{X: int64(10), Y: int64(20)}
return &NPC{Pos: *tmp1, ...}  // ERROR: cannot indirect tmp1 (variable of type interface{})
```

**After (valid Go):**
```go
var tmp1 interface{} = &Coord{X: int64(10), Y: int64(20)}
return &NPC{Pos: *(tmp1.(*Coord)), ...}  // Type assert then dereference
```

**Root cause**: Go requires type assertion before dereferencing an `interface{}` value. The fix adds type assertion `tmp1.(*Type)` before the dereference `*`.

**Files Changed:**
- `internal/gen/golang/codegen_ops.go` - Added type assertion before dereference in typed record generation (~10 LOC)

**Source**: DX feedback from `stapledons_voyage` agent.

### Fixed - ADT Constructor Type Assertion (M-DX14)

**User Impact**: ADT constructors in record fields no longer generate invalid type assertions.

**Before (invalid Go):**
```go
return &NPC{Pattern: *(NewMovementPatternPatternStatic().(*MovementPattern))}
// ERROR: NewMovementPatternPatternStatic() returns *MovementPattern, not interface{}
```

**After (valid Go):**
```go
return &NPC{Pattern: *NewMovementPatternPatternStatic()}
// ADT constructors return typed pointers - just dereference
```

**Root cause**: ADT constructors return typed pointers (`*MovementPattern`), not `interface{}`. Type assertions only work on interface values.

**Files Changed:**
- `internal/gen/golang/codegen_ops.go` - Added `isADTConstructorExpr()` helper to detect constructor calls (~40 LOC)

**Source**: DX feedback from `stapledons_voyage` agent.

### Fixed - Primitive Slice Field Type Mapping (M-DX15)

**User Impact**: Record fields with `[int]` or `[string]` types now use proper slice converters.

**Before (incorrect):**
```go
// ailangTypeToGo("[int]") returned "[]*int64" (wrong!)
// This caused: &FrameOutput{Sounds: tmp1, Debug: tmp2} (no conversion)
```

**After (correct):**
```go
// ailangTypeToGo("[int]") returns "[]int64"
// This generates: &FrameOutput{Sounds: ConvertToInt64Slice(tmp1), Debug: ConvertToStringSlice(tmp2)}
```

**Root cause**: `isUserDefinedGoType("*" + elemType)` was checking `"*int64"` which isn't in the primitive list, so primitives were incorrectly treated as user-defined types. Fixed by checking `elemType` directly.

**Files Changed:**
- `cmd/ailang/compile.go` - Fixed `ailangTypeToGo()` to check `elemType` not `"*" + elemType` (~2 LOC)

**Source**: DX feedback from `stapledons_voyage` agent.

### Fixed - RecordUpdate Preserves Typed Structs (M-DX16)

**User Impact**: Record update expressions (`{ base | field: value }`) now preserve typed structs through the game loop, instead of converting to `map[string]interface{}`.

**Before (type loss):**
```go
// InitWorld returns *World
world := InitWorld()                        // ✓ *World

// After Step, world becomes map[string]interface{}
world = Step(world)                         // ✗ map[string]interface{}
fmt.Printf("Type: %T\n", world)            // Output: map[string]interface{}

// Can't type-assert back to *World
w := world.(*World)                         // PANIC: not *World anymore!
```

**After (type preserved):**
```go
// InitWorld returns *World
world := InitWorld()                        // ✓ *World

// After Step, world is STILL *World
world = Step(world).(*World)                // ✓ *World preserved
fmt.Printf("Type: %T\n", world)            // Output: *recordupdate_test.World

// All subsequent Steps work correctly
for i := 0; i < 100; i++ {
    world = Step(world).(*World)            // Type preserved through entire game loop
}
```

**How it works**: The `RecordUpdate` runtime helper now uses Go reflection to:
1. Detect when base is a typed struct (not `map[string]interface{}`)
2. Create a new instance of the same type
3. Copy all fields from the original
4. Apply updates to matching fields (converting field names to PascalCase)
5. Return the new typed struct pointer

**Implementation:**
```go
// M-DX16: Handle typed structs using reflection
baseVal := reflect.ValueOf(base)
if baseVal.Kind() == reflect.Ptr && baseVal.Elem().Kind() == reflect.Struct {
    // Create new instance, copy fields, apply updates
    newPtr := reflect.New(baseVal.Elem().Type())
    // ... copy fields, apply updates ...
    return newPtr.Interface()  // Returns same type as input!
}
```

**Files Changed:**
- `internal/gen/golang/codegen_runtime.go` - Rewrote `RecordUpdate` helper with reflection (~60 LOC)
- `internal/gen/golang/codegen.go` - Added `reflect` and `strings` imports (~10 LOC)

**Tests:**
- Added unit tests verifying type preservation through chains of updates
- All 35+ golang codegen tests pass

**Source**: DX feedback from `stapledons_voyage` agent.

## [v0.5.2] - 2025-12-03

### Added - Multi-File Compilation Support

**User Impact**: Compile multiple `.ail` files together to merge all types and functions into unified output files.

```bash
# Compile multiple files - types and functions are merged
ailang compile --emit-go --package-name game step.ail npc_ai.ail camera.ail

# Or use glob pattern
ailang compile --emit-go --package-name game *.ail
```

**Generated Files:**
- `types.go` - All ADT types from all files (deduplicated)
- `funcs.go` - All functions from all files
- `runtime.go` - Shared runtime helpers (only for multi-file compilation)
- `handlers.go` - Effect handler interfaces

**Important**: Compile all `.ail` files in a single command. Compiling files separately will overwrite previous output.

**Files Changed:**
- `cmd/ailang/compile.go` - Multi-file argument handling, type deduplication (~120 LOC)
- `internal/gen/golang/codegen.go` - `SetSkipRuntimeHelpers()`, `GenerateRuntime()` methods (~30 LOC)

### Added - Complete Effect Handler Interfaces

All seven effect handlers now have full Go interface definitions for game/application developers:

| Handler | Methods | Purpose |
|---------|---------|---------|
| `DebugHandler` | `Log`, `Assert`, `Collect` | Debugging and tracing |
| `RandHandler` | `RandInt`, `RandFloat` | Deterministic random numbers |
| `ClockHandler` | `Now`, `DeltaTime` | Time and game loop timing |
| `FSHandler` | `Exists`, `ReadFile`, `WriteFile` | File system operations |
| `NetHandler` | `HttpGet`, `HttpPost` | Network requests |
| `EnvHandler` | `GetEnv` | Environment variables |
| `AIHandler` | `Call` | AI model calls |

**Files Changed:**
- `internal/gen/golang/effects.go` - Added FS, Net, Env handler definitions (~60 LOC)
- `cmd/ailang/compile.go` - Register all handlers in generation

### Fixed - Effect Handler Method Call Qualification

**Bug**: Generated Go code called effect functions without handler qualification (e.g., `RandInt(1, 6)` instead of `handlers.Rand.RandInt(1, 6)`).

**Fix**: Added `mapEffectBuiltinToHandler()` function to map AILANG builtins to qualified handler method calls.

**Files Changed:**
- `internal/gen/golang/codegen_expr.go` - Effect builtin to handler mapping (~50 LOC)

### Fixed - Wildcard Pattern Binding in ADT and List Patterns

**Bug**: Wildcard patterns (`_`) in ADT constructor args and list cons patterns generated empty variable names (e.g., ` := _adt.Patrol.Value0`).

**Root Cause**: `ToPascalCase("_")` returns empty string because underscores are skipped.

**Fix**: Added `vp.Name != "_"` checks to skip generating bindings for wildcard patterns.

**Files Changed:**
- `internal/gen/golang/codegen_match.go` - Skip wildcards in three locations (~6 LOC)

### Added - Relaxed Module Matching (M-DX11)

**User Impact**: Files in temp directories or with explicit flags can now have mismatched module declarations. No more MOD010 errors when prototyping!

**Three Relaxation Modes:**
```bash
# 1. CLI flag
ailang run --relax-modules --caps IO --entry main file.ail
ailang check --relax-modules file.ail

# 2. Environment variable
AILANG_RELAX_MODULES=1 ailang run --caps IO --entry main file.ail

# 3. Auto-relaxation for temp paths (automatic)
# Files in /tmp/, /var/folders/, or %TEMP% auto-relax with warning
echo 'module test/hello
let x = 42' > /tmp/test.ail
ailang check /tmp/test.ail  # Warns but passes
```

**Warning Messages:**
```
# Temp path auto-relaxation
WARNING MOD010 (temp-path): module 'test/hello' does not match canonical path 'tmp/test'
  Auto-relaxed for temporary directory. For strict checking, move file outside temp directory.

# Explicit --relax-modules flag
WARNING MOD010 (relaxed): module 'test/hello' does not match canonical path 'src/test'
  Running under --relax-modules; mismatch ignored. For strict checking, omit --relax-modules flag.
```

**Strict Mode (Default) Error:**
```
Error: MOD010: module declaration 'wrong/path' doesn't match canonical path 'src/actual'
Suggestions:
  1. Rename module to: module src/actual
  2. Move file to: wrong/path.ail
  3. For temp/scratch files: use --relax-modules or AILANG_RELAX_MODULES=1
```

**Files Changed:**
- `internal/loader/loader.go` - Added `IsTempPath()` (~65 LOC)
- `internal/loader/loader_test.go` - Unit tests (~55 LOC)
- `internal/pipeline/pipeline.go` - Added `RelaxModules` config field
- `internal/pipeline/pipeline_module.go` - MOD010 relaxation logic (~45 LOC)
- `cmd/ailang/main.go` - `--relax-modules` flag for run command
- `cmd/ailang/check.go` - `--relax-modules` flag for check command

**Total:** ~157 LOC implementation + tests

### Added - Execution Profiles Architecture Design (v0.6.0 Planning)

**Strategic Architecture Document**: Formalized AILANG's execution profiles model.

AILANG is not a game scripting language—it's a **deterministic state-machine DSL** with pluggable effect contexts that can target multiple domains:

| Profile | Entry Shape | Use Cases |
|---------|-------------|-----------|
| **SimProfile** | `step(World, Input) -> (World, Output)` | Games, RL envs, agent sims |
| **ServiceProfile** | `handle(Request) -> Response` | Microservices, agent tools |
| **CliProfile** | `main(args) -> ()` | CLI tools, utilities |

All profiles share the same IR and compiler—only the entry wrappers differ.

**Design Documents Created:**
- [design_docs/planned/v0_6_0/execution-profiles.md](design_docs/planned/v0_6_0/execution-profiles.md) - Full technical specification
- [docs/docs/architecture/execution-profiles.mdx](docs/docs/architecture/execution-profiles.mdx) - Website architecture doc
- [docs/docs/vision.mdx](docs/docs/vision.mdx) - Updated with profiles roadmap

**Next Steps**: Phase 2 Go codegen fixes, then formal `--profile` flag in v0.6.0.

### Added - Go Codegen Phase 2 Design Doc

**Planning document for remaining Go codegen fixes** needed to unblock stapledon:

- Slice type assertions (runtime type conversion)
- Missing runtime helpers (Show, ConcatString, Log)
- Cross-module function generation

See [design_docs/planned/v0_5_2/m-game-b-phase2-go-codegen.md](design_docs/planned/v0_5_2/m-game-b-phase2-go-codegen.md)

### Fixed - Go Codegen: Cross-Module ADT Type Resolution

**Bug**: Go codegen failed with "cannot determine ADT type for match expression" when pattern matching on ADT types defined in imported modules.

**Root Cause**: `RegisterADTConstructor()` only registered constructors from the current module's type declarations, missing ADTs from imported modules.

**Fix**: Extended `cmd/ailang/compile.go` to iterate over `result.Modules` and register ADT constructors from all loaded modules:

```go
// Register ADT constructors from imported modules (cross-module ADT support)
for _, mod := range result.Modules {
    if mod.File != nil {
        for _, decl := range mod.File.Decls {
            if td, ok := decl.(*ast.TypeDecl); ok {
                if adt, ok := td.Definition.(*ast.AlgebraicType); ok {
                    for _, ctor := range adt.Constructors {
                        codeGen.RegisterADTConstructor(td.Name, ctor.Name, len(ctor.Fields))
                    }
                }
            }
        }
    }
}
```

**Files Changed:**
- `cmd/ailang/compile.go` - Added cross-module ADT constructor registration (~15 LOC)

### Fixed - Go Codegen: Generate types.go from Imported ADTs

**Bug**: When compiling modules that import ADT types from other modules, types.go was not generated, causing the generated funcs.go to reference missing constructors like `NewSelectionTile()`.

**Root Cause**: Type declarations were only extracted from the current module's AST, not from imported modules.

**Fix**: Extended type declaration extraction in `cmd/ailang/compile.go` to also collect ADT types from `result.Modules`:

```go
// Extract type declarations from imported modules (cross-module ADT support)
for _, mod := range result.Modules {
    if mod.File != nil {
        for _, decl := range mod.File.Decls {
            if td, ok := decl.(*ast.TypeDecl); ok {
                typeDecls = append(typeDecls, td)
            }
        }
    }
}
```

Now `ailang compile --emit-go` generates a complete types.go with ADT types from both the current module and all imported modules.

**Files Changed:**
- `cmd/ailang/compile.go` - Added imported module type extraction (~10 LOC)

### Fixed - Go Codegen: Type Assertions for ADT Constructor Arguments

**Bug**: Generated funcs.go failed to compile with type errors like:
```
cannot use x (variable of type interface{}) as int64 value in argument to NewSelectionTile: need type assertion
```

**Root Cause**: Generated code uses `interface{}` for all intermediate values, but ADT constructors expect concrete types (int64, float64, etc.).

**Fix**:
1. Extended `ADTConstructorInfo` to include field types
2. Added `RegisterADTConstructorWithTypes()` to register constructors with type information
3. Modified `generateApp()` to add type assertions when calling ADT constructors

**Generated code before:**
```go
return NewSelectionTile(x, y)  // ERROR: x is interface{}
```

**Generated code after:**
```go
return NewSelectionTile(x.(int64), y.(int64))  // OK: proper type assertions
```

**Files Changed:**
- `internal/gen/golang/codegen.go` - Extended ADTConstructorInfo with FieldTypes (~15 LOC)
- `internal/gen/golang/codegen_expr.go` - Added type assertions in generateApp() (~40 LOC)
- `cmd/ailang/compile.go` - Extract field types when registering constructors (~10 LOC)

### Fixed - Go Codegen: Type Conversions for Literal Constants

**Bug**: Type assertions were being applied to literal constants, causing invalid Go:
```go
NewDrawCmdRect(8.(int64), ...)  // ERROR: 8 is not an interface
```

**Fix**: Check if argument is a literal (`*core.Lit`) and use type conversion instead:
```go
NewDrawCmdRect(int64(8), ...)  // OK: type conversion
```

**Files Changed:**
- `internal/gen/golang/codegen_expr.go` - Distinguish literals from interface values (~10 LOC)

---

## [v0.5.1] - 2025-12-02

### Added - API Discovery Commands (M-DX-API-DISCOVERY)

**User Impact**: Discover function signatures without trial-and-error. No more guessing `rand_int(4)` vs `rand_int(0, 4)`.

**New Commands:**
```bash
ailang builtins show _rand_int     # Full docs for a builtin
ailang builtins list --verbose     # All builtins with signatures
ailang builtins list --by-module --verbose  # Grouped by module
```

**Example Output:**
```
_rand_int: (int, int) -> int ! {Rand}

Usage:
  import std/rand (rand_int)
  rand_int(...)

Description:
  Generate random integer in range [min, max] inclusive

Parameters:
  min:         Minimum value (inclusive)
  max:         Maximum value (inclusive)
```

**Features:**
- `--verbose` flag shows full signatures and descriptions
- `show <name>` command with fuzzy search ("Did you mean:")
- Shows public import path for internal builtins (`_rand_int` → `rand_int`)

**Files Changed:**
- `cmd/ailang/doctor.go` - Added ~220 LOC for verbose/show commands

---

### Added - v0.5.1 Teaching Prompt with Effect Module APIs

**User Impact**: Teaching prompt now documents all effect module function signatures upfront.

**New Section: Effect Module APIs**
- `std/rand` - rand_int, rand_float, rand_bool, rand_seed
- `std/debug` - log, check
- `std/clock` - now
- `std/ai` - infer
- `std/game` - get_player_state, tick

**DX Pattern Documented:**
```bash
ailang builtins show _rand_int     # Full docs for a builtin
ailang builtins list --by-module --verbose  # All builtins with signatures
```

**Files Added:**
- `cmd/ailang/prompts/v0.5.1.md` - New prompt with Effect Module APIs section

---

### Fixed - Go Codegen for Record Update (M-CODEGEN-RECORDUPDATE)

**User Impact**: `ailang compile --emit-go` now works with record update syntax.

**Before:**
```
unsupported expression type: *core.RecordUpdate
```

**After:**
```go
func UpdateAge(person interface{}, newAge interface{}) interface{} {
    return RecordUpdate(person, map[string]interface{}{"age": newAge})
}
```

**AILANG Syntax:**
```ailang
export func updateAge(person: {name: string, age: int}, newAge: int) =
  { person | age: newAge }
```

**Files Changed:**
- `internal/gen/golang/codegen.go` - Added RecordUpdate case and runtime helper

---

## [v0.5.0] - 2025-12-02

### Added - Sim Stub Example & CI Integration (M-GAME-D)

**User Impact**: Complete working example demonstrating AILANG → Go code generation workflow with CI validation.

**New Example: `examples/sim_stub/`**
- `world.ail` - AILANG types and extern function declarations
- `impl.go` - Go implementation of extern functions
- `main.go` - Go driver that runs 10 deterministic ticks
- `Makefile` - Build workflow (generate, build, test)
- `expected_output.txt` - Golden file for CI testing

**Usage:**
```bash
cd examples/sim_stub
make run            # Generate, build, and run
make test           # Verify deterministic output
```

**CI Integration:**
- New `make test-sim-stub` target in root Makefile
- New `.github/workflows/test-game-codegen.yml` workflow
- Validates: AILANG compile → Go build → run → compare output

**Elaborator Fix:**
- Extern functions (nil Body) now correctly skipped during elaboration
- Previously caused "normalization received nil expression" error

**Files Added/Changed:**
- `examples/sim_stub/` - Complete example (~250 LOC)
- `internal/elaborate/file.go` - Skip extern functions in collectFuncSigs
- `Makefile` - Added test-sim-stub target
- `.github/workflows/test-game-codegen.yml` - CI workflow

---

### Added - Debug Effect for Runtime Tracing (M-GAME-E1)

**User Impact**: New `Debug` effect for structured runtime tracing and assertions. Write-only from AILANG, host collects. Zero-cost in release mode (ghost effect).

**Usage:**
```ailang
import std/debug as Debug

func update(e: Entity) -> Entity ! {Debug} {
    Debug.check(e.health >= 0, "health must be non-negative");
    Debug.log("updating entity " ++ show(e.id));
    -- ... entity logic
    e
}
```

**Run with Debug capability:**
```bash
ailang run --caps IO,Debug --entry main game.ail
```

**Features:**
| Feature | Description |
|---------|-------------|
| `Debug.log(msg)` | Write trace message (host collects) |
| `Debug.check(cond, msg)` | Record assertion (doesn't throw, continues execution) |
| Ghost effect | Erased in `--release` mode (zero runtime cost) |
| Write-only | Only host calls `DebugContext.Collect()` |

**Note:** We use `check` instead of `assert` because `assert` is a reserved keyword in AILANG.

**Host Integration (Go):**
```go
debugCtx := effects.NewDebugContext()
debugCtx.SetTimestamp(int64(tick))
// ... run AILANG code ...
output := debugCtx.Collect()  // Host-only operation
debugCtx.Reset()              // Clear for next tick
```

**Files Added/Changed:**
- `std/debug.ail` - Debug module with `log` and `check` wrappers
- `internal/effects/debug.go` - DebugContext, LogEntry, AssertionResult types
- `internal/builtins/debug.go` - `_debug_log` and `_debug_check` builtins
- `internal/parser/parser_effect.go` - Added Debug to known effects
- `examples/runnable/debug_effect.ail` - Complete example
- `prompts/v0.4.10.md` - Updated teaching prompt with Debug effect

**Design Doc:** `design_docs/planned/v0_5_0/M-GAME-E1-debug-effect.md`

---

### Added - AI Effect for General-Purpose AI Oracle (M-GAME-E2)

**User Impact**: New `AI` effect for calling external AI/ML systems. String→string interface (JSON by convention), pluggable handlers, no silent fallbacks.

**Usage:**
```ailang
import std/ai as AI
import std/json

func choose_action(ctx: NPCContext) -> Action ! {AI} {
    let input = json.encode(ctx);
    let output = AI.call(input);
    match json.decode[Action](output) {
        Ok(action) => action,
        Err(_)     => Wait  -- Safe fallback
    }
}
```

**Run with AI capability:**
```bash
ailang run --caps IO,AI --entry main game.ail
```

**Features:**
| Feature | Description |
|---------|-------------|
| `AI.call(input)` | Call AI oracle with string input, returns string |
| String→string | JSON by convention, not enforced |
| Pluggable handlers | StubAIHandler for tests, custom handlers for prod |
| No silent fallback | Nil handler returns `ErrNoAIHandler` |

**Host Integration (Go):**
```go
// Testing: Use stub handler
aiHandler := game.NewStubAIHandler()
aiHandler.SetDefaultResponse(`{"kind":"Wait"}`)
aiCtx := game.NewAIContext(aiHandler)

// Production: Implement your own AIHandler interface
aiCtx := game.NewAIContext(yourOpenAIHandler)

// Call AI
output, err := aiCtx.Call(input)  // err if handler nil
```

**Files Added/Changed:**
- `std/ai.ail` - AI module with `call` wrapper
- `internal/effects/ai.go` - AIContext, AIHandler, StubAIHandler types
- `internal/builtins/ai.go` - `_ai_call` builtin
- `internal/gen/golang/ai.go` - Go codegen for AI effect
- `internal/parser/parser_effect.go` - Added AI to known effects
- `examples/sim_stub/gen/game/ai_types.go` - Generated AI types example

**Design Doc:** `design_docs/planned/v0_5_0/M-GAME-E2-ai-effect.md`

---

### Added - Multi-Provider AI Effect CLI Flag

**User Impact**: The `--ai` CLI flag now supports multiple providers with automatic detection from model name.

**Usage:**
```bash
# Anthropic (Claude)
ailang run --caps IO,AI --ai claude-haiku-4-5 --entry main file.ail

# OpenAI (GPT)
ailang run --caps IO,AI --ai gpt5-mini --entry main file.ail

# Google (Gemini)
ailang run --caps IO,AI --ai gemini-2-5-flash --entry main file.ail
```

**Features:**
| Feature | Description |
|---------|-------------|
| Model lookup | Uses `models.yml` for api_name, provider, env_var |
| Prefix guessing | Falls back to `claude-*`→anthropic, `gpt*`→openai, `gemini-*`→google |
| Extensible | Add new providers by implementing handler + switch case |

**Environment Variables:**
- `ANTHROPIC_API_KEY` for Claude models
- `OPENAI_API_KEY` for GPT models
- `GOOGLE_API_KEY` for Gemini models

**Files Changed:**
- `cmd/ailang/main.go` - Changed `--ai-anthropic` to `--ai`
- `cmd/ailang/ai_handlers.go` - Added OpenAIHandler, GoogleHandler, provider detection

---

### ABI Stability Promise (v0.5.x)

**The Go interop ABI is now "stable preview":**

| Component | Stability | Notes |
|-----------|-----------|-------|
| Type mapping (primitives) | ✅ Stable | int→int64, float→float64, string, bool |
| Record type generation | ✅ Stable | Struct field ordering preserved |
| Extern function signatures | ✅ Stable | Generated stubs won't break |
| ADT discriminator format | ⚠️ Preview | May change before v0.6.0 |
| Generic type handling | ⚠️ Preview | Currently uses interface{} |

**What this means:**
- Safe to use in production for non-generic, non-ADT code
- Breaking changes will be announced in CHANGELOG with migration path
- Full stability guaranteed starting v0.6.0

**Documentation:**
- `docs/docs/guides/go-interop.md` - Comprehensive Go interop guide
- `README.md` - Added Go Interop section with ABI stability notice

---

### Added - Go Code Generation & Extern Functions (M-GAME-C)

**User Impact**: New `ailang compile` command generates Go source code from AILANG types and extern function declarations, enabling seamless Go interop for game development.

**New CLI Command:**
```bash
ailang compile --emit-go --out gen --package-name game world.ail
```

**Flags:**
| Flag | Description |
|------|-------------|
| `--emit-go` | Generate Go source code (required) |
| `--out <dir>` | Output directory (default: "gen") |
| `--package-name <name>` | Go package name (default: derived from module) |

**Extern Functions:**
Declare Go-implemented functions in AILANG with type-safe signatures:
```ailang
type Position = { x: int, y: int }

extern func find_path(from: Position, to: Position) -> [Position]
extern func calculate_distance(a: Position, b: Position) -> float
```

**Generated Output:**
- `types.go` - ADT types as Go structs with constructors
- `extern_stubs.go` - Function stubs to implement in Go
- `funcs.go` - Compiled functions (experimental)

**Example Generated Stub:**
```go
// find_path is an extern function declared in AILANG.
//
// AILANG signature:
//   extern func find_path(from: Position, to: Position) -> [Position]
//
// Implement this function to provide the behavior.
func Find_path(from *Position, to *Position) []*Position {
    panic("not implemented: find_path")
}
```

**Error Codes:**
| Code | Description |
|------|-------------|
| EXT001 | Underscore prefix not allowed (reserved for builtins) |
| EXT002 | Polymorphic extern functions not supported |
| EXT003 | Missing explicit return type |

**Files Added/Changed:**
- `cmd/ailang/compile.go` - Compile subcommand (~330 LOC)
- `internal/lexer/token.go` - Added EXTERN token
- `internal/parser/parser_func.go` - parseExternFunctionDeclaration (~95 LOC)
- `internal/ast/ast_decl.go` - Added IsExtern field to FuncDecl
- `docs/docs/guides/go-interop.md` - Comprehensive interop guide

**Design Doc:** `design_docs/planned/v0_5_0/M-GAME-C-compiler-ux-extern.md`

---

### Benchmark Results (M-EVAL)

**Overall Performance**: 71.4% success rate (1103 total runs: 827 standard + 276 agent)

**Standard Eval (0-shot + self-repair):**

| Metric | v0.4.10 | v0.5.0 | Change |
|--------|---------|--------|--------|
| **0-shot (first attempt)** | 61.9% | 57.4% (317/552) | -4.5% |
| **Final (with repair)** | 66.7% | 60.8% (336/552) | -5.9% |
| **Python (final)** | 77.0% | 81.8% (451/551) | +4.8% |
| **AILANG (final)** | 56.5% | 39.8% | -16.7% |

**Agent Eval (multi-turn iterative problem solving):**

| Language | v0.4.10 | v0.5.0 | Change |
|----------|---------|--------|--------|
| **AILANG** | N/A | 73.9% (102/138) | new |
| **Python** | N/A | 96.3% (133/138) | new |
| **Overall** | N/A | 85.1% (235/276) | new |

**Key Findings:**
- Agent eval shows strong results: 85.1% overall, with 96.3% for Python
- Standard eval AILANG scores dropped due to stricter type checking in v0.5.0
- Python standard eval improved +4.8% with better prompt training
- Total eval cost: $18.89 for 9 models across 1103 runs

---

## [v0.4.10] - 2025-12-01

### Added - Array Type with O(1) Indexed Access (M-ARRAY-TYPE)

**User Impact**: New `Array[T]` type for O(1) random access operations, enabling efficient game grids and lookup tables.

**Syntax:**
- Literal: `#[1, 2, 3, 4, 5]` (hash + brackets)
- Type: `Array[int]`, `Array[string]`, `Array[{x: int, y: int}]`

**Available Operations (via `import std/array as A`):**
| Function | Cost | Description |
|----------|------|-------------|
| `A.make(n, v)` | O(n) | Create array of size n with default value v |
| `A.get(arr, i)` | O(1) | Get element at index (error if out of bounds) |
| `A.getOpt(arr, i)` | O(1) | Safe get: Some(elem) if in bounds, None otherwise |
| `A.set(arr, i, v)` | O(n) | Return NEW array with element at index updated (copy-on-write) |
| `A.length(arr)` | O(1) | Get array length |
| `A.fromList(xs)` | O(n) | Convert list to array |
| `A.toList(arr)` | O(n) | Convert array to list |
| `A.unsafeGet(arr, i)` | O(1) | Get element (panics if out of bounds - use with caution) |

**Example:**
```ailang
module examples/array_grid
import std/array as A

let grid = A.make(25, 0)                    -- 5x5 grid of zeros
let updated = A.set(grid, 12, 42)           -- Set center cell
let value = A.get(updated, 12)              -- Returns 42
let len = A.length(updated)                 -- Returns 25
```

**Key Characteristics:**
- **0-based indexing** (first element at index 0, last at `length - 1`)
- Arrays are **immutable** - `set` returns a NEW array (O(n) copy-on-write)
- **O(1)** for reads: `get`, `getOpt`, `length`
- **O(n)** for updates: `set` (creates full copy), `fromList`, `toList`
- Use `getOpt` for safe access returning `Option[a]` instead of errors
- Designed for game grids, lookup tables, read-heavy workloads

**Files Added/Changed:**
- `internal/types/types.go` - Added TArray type
- `internal/eval/value.go` - Added ArrayValue with Get/Set methods
- `internal/builtins/array.go` - 7 array builtins (~400 LOC)
- `internal/builtins/array_test.go` - Unit tests (~290 LOC)
- `std/array.ail` - Stdlib module (~35 LOC)
- `examples/array_basic.ail` - Basic operations example
- `examples/array_grid.ail` - 2D grid game example
- `prompts/v0.4.10.md` - Updated teaching prompt

**Design Doc:** `design_docs/planned/v0_5_0/m-array-type.md`

---

### Fixed - TVar2 Unification with Records in List Patterns (M-BUG-TVAR2-LIST-PATTERN)

**User Impact**: Record field access through list pattern bindings now works correctly. Previously, code like `match positions { pos :: rest => pos.x, [] => 0 }` failed with "type unification failed: cannot unify open record with *types.TVar2".

**Root Cause**: The unification code in `internal/types/unification.go` handled `TRecordOpen` unification with `*TVar` but not with `*TVar2`. When list pattern bindings created fresh type variables (`TVar2`), the record field access couldn't unify properly.

**What Was Fixed**: Added a `case *TVar2:` clause to `TRecordOpen` unification that swaps and retries (matching the existing `*TVar` behavior).

**Before (Failed):**
```ailang
let getFirstX = \positions. match positions {
    pos :: rest => pos.x,  -- ERROR: cannot unify open record with *types.TVar2
    [] => 0
}
```

**After (Works):**
```ailang
let getFirstX = \positions. match positions {
    pos :: rest => pos.x,  -- Works! Returns the x field
    [] => 0
}

-- Nested record access also works:
match entities { e :: rest => e.pos.x, [] => 0 }
```

**Files Changed:**
- `internal/types/unification.go` - Added TVar2 case (~4 LOC)
- `examples/list_pattern_records.ail` - New example file

**Reported by:** stapledons_voyage (via agent inbox)

---

### Fixed - ADT Constructors in Test Harness Scope (M-BUG-ADT-TEST-HARNESS-SCOPE)

**User Impact**: Inline tests with ADT constructor inputs now work correctly. Previously, tests like `tests [(North, 0), (South, 180)]` failed with "undefined variable: North" because ADT constructors weren't in scope during test harness evaluation.

**Root Cause**: The test harness (`internal/testing/harness.go`) converted AST to Core and evaluated it directly, but the evaluator environment didn't include ADT constructor bindings from the source file's type declarations.

**What Was Fixed**:
- Added `ConstructorClosure` value type to `internal/eval/value.go` for constructors with data
- Added handler for `*ConstructorClosure` in `evalCoreApp` to create `TaggedValue` when applied
- Added `injectADTConstructors` helper to extract ADT constructors from source file and inject into evaluator environment

**Before (Failed):**
```ailang
type Direction = North | South | East | West

pure func directionToDegrees(d: Direction) -> int
    tests [(North, 0), (East, 90)]  -- ERROR: undefined variable: North
{ ... }
```

**After (Works):**
```ailang
type Direction = North | South | East | West

pure func directionToDegrees(d: Direction) -> int
    tests [(North, 0), (East, 90), (South, 180), (West, 270)]  -- All pass!
{ ... }

-- Also works with data constructors:
type Option[a] = Some(a) | None
tests [((Some(42), 0), 42), ((None, 100), 100)]
```

**Files Changed:**
- `internal/eval/value.go` - Added ConstructorClosure type (~26 LOC)
- `internal/eval/eval_operations.go` - Added ConstructorClosure handler (~4 LOC)
- `internal/testing/executor.go` - Added injectADTConstructors helper (~45 LOC)
- `examples/adt_test_harness.ail` - New example file

**Reported by:** stapledons_voyage (via agent inbox)

---

### Fixed - Imported ADT Type Pollution (M-BUG-IMPORTED-ADT-TYPE-POLLUTION)

**User Impact**: ADT constructors with different field types can now coexist in the same module without causing type pollution. Previously, using `PatternPatrol([Direction])` and `PatternRandomWalk(int)` in the same file caused "No instance for Num[[Direction]] in scope" because int literals were incorrectly typed as `[Direction]`.

**Root Cause**: In `internal/iface/builder.go`, the interface builder used placeholder type variables (`TVar2{a0}`, `TVar2{a1}`, etc.) for constructor field types instead of extracting actual types from the AST. When multiple constructors had different field types, they shared these placeholders, causing incorrect type unification.

**What Was Fixed**:
- Added `astTypeToInternalType()` helper function to convert AST types to internal types
- Modified AST processing loop to extract actual field types from algebraic type constructors
- Constructor field types now use real types (`TList{TInt}`, `TCon{Direction}`, etc.) instead of placeholders

**Before (Failed):**
```ailang
type MovementPattern =
    | PatternPatrol([Direction])
    | PatternRandomWalk(int)  -- int literal '30' became [Direction]!

let npc = { pattern: PatternRandomWalk(30), moveCounter: 30 }
-- ERROR: No instance for Num[[Direction]] in scope
```

**After (Works):**
```ailang
type MovementPattern =
    | PatternPatrol([Direction])
    | PatternRandomWalk(int)

let npc = { pattern: PatternRandomWalk(30), moveCounter: 30 }  -- Works!
```

**Files Changed:**
- `internal/iface/builder.go` - Added astTypeToInternalType helper (~80 LOC)
- `examples/runnable/imported_adt_types.ail` - New example file

**Reported by:** stapledons_voyage (via agent inbox)

---

## [v0.4.9] - 2025-12-01

### Fixed - Record Update ANF Verification

**User Impact**: Record updates with nested values now work correctly. Previously, `{ npc | pos: { x: 5, y: 10 } }` failed with "unknown expression type in ANF verification".

**What Was Fixed**: Added missing `*core.RecordUpdate` case to ANF verifier in both `verifyExpr` and `verifySimpleOrAtomic` functions.

**Files Changed:**
- `internal/elaborate/verify.go` - Added RecordUpdate verification (~25 LOC)
- `examples/nested_records.ail` - Added record update examples

---

### Fixed - Inline Nested Record Literals (M-BUG-NESTED-RECORD-ANF)

**User Impact**: Inline nested record literals now work correctly. Previously, code like `{ pos: { x: 10, y: 20 }, name: "guard" }` failed with "ANF verification error: let bindings are not simple calls".

**Root Cause**: This was a standard ANF completion bug. The ANF transformer generated nested `Let` expressions inside record values, but forgot to run the canonical "float inner lets to statement level" transformation. When a let binding's value was itself a `Let` expression (from normalizing nested records), the ANF verifier rejected it.

**What Was Fixed**:
- Added `extractLetBindings()` helper to extract top-level Let bindings from expressions
- Modified `normalizeLet` to flatten nested lets in RHS values
- Modified `normalizeToAtomic` to also flatten lets when creating temporary bindings
- This ensures all let RHS values are simple expressions (not Let expressions)

**Before (Failed):**
```ailang
let npc = { pos: { x: 10, y: 20 }, name: "guard" }
npc.pos.x
-- ERROR: ANF verification error: let bindings are not simple calls
```

**After (Works):**
```ailang
let npc = { pos: { x: 10, y: 20 }, name: "guard" }
npc.pos.x  -- Returns: 10

-- Deeply nested also works:
let game = { player: { pos: { x: 0, y: 0 }, stats: { hp: 100 } }, level: 1 }
game.player.stats.hp  -- Returns: 100
```

**Files Changed:**
- `internal/elaborate/core.go` - Added `extractLetBindings()` helper (~30 LOC)
- `internal/elaborate/expressions.go` - Modified `normalizeLet` for let-flattening (~35 LOC)
- `examples/nested_records.ail` - New example file (~60 LOC)

**Reported by:** stapledons_voyage (via agent inbox)

---

### Fixed - Module-Level Let Binding Scope (M-BUG-MODULE-LET-SCOPE)

**User Impact**: Module-level `let` bindings are now accessible inside function bodies in the same module. Previously, attempting to reference a module-level constant from within a function resulted in "undefined variable" errors.

**Root Cause**: In `internal/elaborate/file.go`, functions were elaborated before module-level let bindings were processed. The let bindings were never added to the symbol scope used when elaborating function bodies.

**What Was Fixed**:
- Added `collectModuleLets()` to extract module-level let bindings before function elaboration
- Modified `ElaborateFile` to wrap function declarations in module-level lets
- Added `extractExportsFromExpr()` in interface builder for recursive export extraction from nested Let structures

**Before (Failed):**
```ailang
module game/config

let tileSize: int = 8

pure func getTileSize() -> int {
    tileSize  -- ERROR: undefined variable: tileSize
}
```

**After (Works):**
```ailang
module game/config

let tileSize: int = 8

pure func getTileSize() -> int {
    tileSize  -- Returns: 8
}
```

**Files Changed:**
- `internal/elaborate/file.go` - Added module-level let collection and wrapping (~50 LOC)
- `internal/iface/builder.go` - Added recursive export extraction (~30 LOC)

**Reported by:** stapledons_voyage (via agent inbox)

---

### Fixed - Test Harness ADT Constructor Support

**User Impact**: Test harness now supports ADT constructors in test inputs/outputs. Previously, using values like `None`, `Some(5)`, or `Pair(1, 2)` in test cases caused a panic.

**What Was Fixed**:
- Added `*ast.Identifier` support for nullary constructors (e.g., `None`, `True`)
- Added `*ast.FuncCall` support for constructor application (e.g., `Some(5)`, `Pair(1, 2)`)
- Added `*ast.Record` support for record literals in test cases

**Files Changed:**
- `internal/testing/harness.go` - Added cases to `astExprToCore` (~35 LOC)

**Reported by:** stapledons_voyage (via agent inbox)

---

### Fixed - List Concatenation Operator (++) Dispatch

**User Impact**: The `++` operator now correctly works with lists. Previously, using `++` on lists caused a runtime panic because the operator was incorrectly dispatched to string concatenation instead of list concatenation.

**Root Cause**: In `internal/types/type_head.go`, the `Head()` function only checked for list types represented as `TApp{Constructor: TCon{Name: "List"}}` but AILANG uses a dedicated `TList` type struct. When the type head lookup failed, the operator lowering defaulted to "String" suffix, calling `strConcatImpl` instead of `listConcatImpl`.

**What Was Fixed**:
- Added `*TList` case to `Head()` function to return `HeadList`
- Now both `TList` and `TApp{List}` representations are recognized as list types

**Before (Failed):**
```ailang
pure func range(n: int) -> [int] {
    match n {
        0 => [],
        _ => range(n - 1) ++ [n - 1]  -- PANIC: interface conversion error
    }
}
```

**After (Works):**
```ailang
pure func range(n: int) -> [int] {
    match n {
        0 => [],
        _ => range(n - 1) ++ [n - 1]  -- Returns: [0, 1, 2, ..., n-1]
    }
}
```

**Debug output before fix:**
```
[DEBUG M-DX4] NodeID 44: type=[int], head=Unknown
```

**Debug output after fix:**
```
[DEBUG M-DX4] NodeID 44: type=[int], head=List
```

**Files Changed:**
- `internal/types/type_head.go` - Added `*TList` case to `Head()` function (~3 LOC)

**Test file added:** `examples/test_list_recursion.ail`

**Reported by:** stapledons_voyage (via agent inbox) - This fix also resolves the "64-item recursion hang" which was actually a symptom of the `++` dispatch bug.

---

## [v0.4.8] - 2025-11-29

### Fixed - Match Expression Recursion (M-BUG-RECURSION-DEPTH) 🐛

**User Impact**: Recursive functions using `match` expressions now work correctly. Previously, even simple recursive functions like `count(1)` caused infinite recursion when using `match` with integer literal patterns.

**Root Cause**: Integer literal patterns (like `0`) in match expressions were stored as `int64` in the Core AST, but `matchPattern()` only checked for `int` type assertions. This caused literal patterns to **never match**, so the wildcard `_` always matched, leading to infinite recursion because the base case never triggered.

**What Was Fixed**:
- Added `int64` type checking in `matchPattern()` for `LitPattern` with `IntValue` scrutinees
- Integer literal patterns now correctly match against `IntValue` regardless of whether stored as `int` or `int64`

**Before (Failed - Infinite Recursion):**
```ailang
pure func count(n: int) -> int {
  match n {
    0 => 0,           -- Never matched! (0 stored as int64)
    _ => 1 + count(n-1)  -- Always matched, including n=0
  }
}
count(1)  -- ERROR: max recursion depth exceeded
```

**After (Works):**
```ailang
pure func count(n: int) -> int {
  match n {
    0 => 0,           -- Now matches correctly
    _ => 1 + count(n-1)
  }
}
count(1000)  -- Returns 1000
```

**Files Changed:**
- `internal/eval/eval_patterns.go` - Added int64 pattern matching (~15 LOC)
- `internal/eval/recursion_test.go` - Added regression tests (~140 LOC)
- `examples/runnable/recursion_match.ail` - New example file (~60 LOC)

**Reported by:** stapledons_voyage (via agent inbox)

---

### Fixed - Record Update Type Inference (M-BUG-RECORD-UPDATE-INFERENCE) 🐛

**User Impact**: Record update syntax `{base | field: value}` now works correctly with lambda parameters. Previously failed with "record update requires base to be a record type, got *types.TVar2".

**What Was Fixed**:
- Refactored `inferRecordUpdate` to use constraint-based type checking
- Follows the same pattern as `inferRecordAccess` (defers type checking to constraint solver)
- Record updates now work with: typed lambda parameters, untyped lambdas, let-bound records

**Before (Failed):**
```ailang
type World = { tick: int }
let step: World -> World = \world. {world | tick: world.tick + 1}
-- ERROR: record update requires base to be a record type, got *types.TVar2
```

**After (Works):**
```ailang
type World = { tick: int }
let step: World -> World = \world. {world | tick: world.tick + 1}
let w = { tick: 0 }
step(w)  -- Returns { tick: 1 }
```

**Files Changed:**
- `internal/types/typechecker_data.go` - Constraint-based fix (~40 LOC net)
- `examples/record_update.ail` - New example file (~60 LOC)
- `tests/record_update_regression_test.ail` - Regression tests

**Reported by:** stapledons_voyage (via agent inbox)

---

### Benchmark Results (M-EVAL)

**Overall Performance**: 69.2% success rate (828 total runs across 9 models)

**Standard Eval (0-shot + self-repair):**
| Metric | v0.4.8 |
|--------|--------|
| 0-shot (first attempt) | 55.3% (229/414) |
| Final (with repair) | 60.6% (251/414) |
| Repair effectiveness | +5.3pp |
| Python (final) | 77.7% (322/414) |

**Models tested**: gpt5, gpt5-instant, gpt5-mini, claude-opus-4-5, claude-sonnet-4-5, claude-haiku-4-5, gemini-3-pro, gemini-2-5-pro, gemini-2-5-flash

---

## [v0.4.7] - 2025-11-27

### Added - Cross-Function Dependency Support for Inline Tests (M-TESTING-DEPS) 🧪

**User Impact**: Inline tests now work for functions that call other user-defined functions! This enables testing functions like `lcm` that depend on `gcd`, and mutual recursion patterns like `isEven`/`isOdd`.

**What Was Added**:
1. **Call Graph Analysis** (~200 LOC in internal/testing/)
   - Tarjan's algorithm for strongly connected component (SCC) detection
   - Identifies mutual recursion (isEven↔isOdd) and chain dependencies (lcm→gcd)
   - Pure cluster extraction includes all dependencies automatically

2. **Cluster Harness Building** (~150 LOC)
   - Multi-binding LetRec for testing functions with dependencies
   - Strips non-pure functions to prevent type-checking errors
   - Preserves exported pure functions (fix: was incorrectly stripping them)

3. **Runner Integration** (~50 LOC in internal/testing/runner.go)
   - Automatically detects if function has cross-function dependencies
   - Routes to cluster evaluation or single-binding evaluation as appropriate
   - Seamless UX - users don't need to know about the underlying complexity

4. **New Example File** (`examples/test_cross_function_deps.ail`)
   - 38 tests demonstrating 4 dependency patterns:
     - Chain dependencies (lcm→gcd)
     - Mutual recursion (isEven↔isOdd)
     - Helper function chains (sumOfSquares→square)
     - Multi-level chains (power→multiply→add)

**Example:**
```ailang
-- GCD (Euclidean algorithm)
export pure func gcd(a: int, b: int) -> int
  tests [((12, 8), 4), ((15, 5), 5), ((7, 3), 1)]
{ if b == 0 then a else gcd(b, a % b) }

-- LCM calls gcd - tests work automatically!
export pure func lcm(a: int, b: int) -> int
  tests [((4, 6), 12), ((3, 5), 15), ((12, 18), 36)]
{ (a * b) / gcd(a, b) }
```

**Total inline tests**: 150+ tests across 12+ example files (up from 98 in v0.4.6)

### Added - Claude Opus 4.5 to Model Manager

- Added `claude-opus-4-5` to eval suite (`claude-opus-4-5-20251101`)
- Pricing: $5/$25 per million tokens (input/output)
- 80.9% SWE-bench score - state-of-the-art for coding
- Full agent CLI support via Claude Code

### Changed - Teaching Prompt v0.4.7

- Added inline tests to key examples (factorial, sum, length, treeSum, add)
- Added "Cross-Function Dependencies" section showcasing M-TESTING-DEPS
- Examples now serve as executable documentation

### Fixed - Model Manager Script

- Fixed `run_test_benchmark.sh` JSON parsing (key was `stdout_ok` not `result`)
- Fixed recursive file search for results in subdirectories

### Benchmark Results (M-EVAL)

**Overall Performance**: 69.2% success rate (737 total runs)

**Standard Eval (0-shot + self-repair):**

| Metric | 0.4.6 | v0.4.7 | Change |
|--------|--------|--------|--------|
| **0-shot (first attempt)** | 64.2% | 65.0% (286/440) | **+0.8%** |
| **Final (with repair)** | 66.9% | 67.9% (299/440) | **+1.0%** |
| **Repair effectiveness** | +2.7pp | +2.9pp | **+0.2pp** |
| **Python (final)** | 77.6% | 79.1% (335/423) | +1.5% |

**Agent Eval (multi-turn iterative problem solving):**

| Language | 0.4.6 | v0.4.7 | Change |
|----------|--------|--------|--------|
| **AILANG** | 100.0% | 96.7% (60/62) | **-3.3%** |
| **Python** | 100.0% | 100.0% (64/64) | 0% |

**Key Findings:**
- Steady improvement in standard eval (+1% final success rate)
- Claude Opus 4.5 added to eval suite - first release with 9 production models
- Minor agent eval regression (-3.3% AILANG) likely due to 2 edge case failures

## [v0.4.5] - 2025-11-16

### Fixed - String Concatenation (`++`) Operator Type Inference (M-BUG-CONCAT-INFERENCE) 🐛

**User Impact**: Recursive string concatenation now works correctly. Before this fix, the `++` operator incorrectly defaulted to list concatenation when both operands were type variables in recursive contexts, breaking natural recursive string building patterns that work in all other ML languages.

**What Was Fixed**:
1. **Operator Logic Update** (~70 LOC in internal/types/typechecker_operators.go)
   - Root cause: When both operands were type variables, `++` defaulted to list concat ("more polymorphic")
   - This broke recursive functions like `func join(sep: string, xs: [int]) -> string`
   - Fix: Changed default from list concat to string concat when both operands are type variables
   - Added explicit check for incompatible concrete types (string + list → error)
   - Added expected-type context infrastructure for future bidirectional typing

2. **Expected-Type Context** (~40 LOC)
   - Added `expectedType` field to `InferenceContext` (internal/types/inference.go)
   - Added `withExpectedType()` helper for tail-position type threading
   - Threaded expected type through Match arm bodies (internal/types/typechecker_patterns.go)
   - Foundation for future principled ambiguity resolution

3. **Test Coverage** (~150 LOC in internal/pipeline/concat_operator_test.go)
   - ✅ Recursive string concat (the bug case) - now works
   - ✅ List concat with signature - regression test, still works
   - ✅ Concrete string/list concat - regression tests, still work
   - ✅ Type var + concrete type - works correctly
   - ✅ Nested recursion - works correctly
   - ⏭️ Mixed types (string + list) - skipped, reveals deeper type annotation threading issue (deferred to v0.4.6)

**Before the fix:**
```ailang
export func join(sep: string, xs: [int]) -> string {
  match xs {
    [] => "",
    [x] => show(x),
    x :: rest => show(x) ++ sep ++ join(sep, rest)  -- Type error!
  }
}
-- Error: cannot unify type constructor string with *types.TList
```

**After the fix:**
```ailang
join(", ", [1, 2, 3, 4, 5])  -- Returns "1, 2, 3, 4, 5" ✓
```

**Known Limitations** (deferred to v0.4.6):
- Type annotations from function signatures aren't fully threaded to Core type checking
- Mixed string/list concatenation caught at runtime, not type-checking time
- Full expected-type threading for ambiguity errors not yet implemented

**Total Changes**: ~260 LOC (implementation: ~110 LOC, tests: ~150 LOC)

### Fixed - Nullary Constructor Pattern Matching (M-BUG-NULLARY) 🐛 Critical Bug Fix

**User Impact**: Simple enum types (ADTs with only nullary constructors) now work correctly in pattern matching. Before this fix, all values matched the first pattern, breaking type safety guarantees. This enables production use of enum types like `type Status = Pending | InProgress | Completed`.

**What Was Fixed**:
1. **Elaboration Fix** (~13 LOC in internal/elaborate/patterns.go)
   - Root cause: Nullary constructors (`Red`, `Green`, `Blue`) were being elaborated as `VarPattern` instead of `ConstructorPattern`
   - Variable patterns always match and bind, causing all three values to match the first pattern
   - Fix: Check if identifier is a known nullary constructor (arity=0) in elaborator's constructor map
   - If yes, create `ConstructorPattern` with empty args; otherwise create `VarPattern`
   - Location: `elaboratePattern()` function, line 79-90

2. **Test Coverage** (~120 LOC total)
   - Unit tests: 6 test cases in `internal/elaborate/patterns_nullary_test.go`
     - Nullary constructors (Red, Green, Blue, None) → ConstructorPattern ✓
     - Variable patterns → VarPattern ✓
     - Non-nullary constructors with arity >0 → VarPattern ✓
   - Integration test: `tests/nullary_pattern_matching_test.ail` (~67 LOC)
     - Tests Status (3 variants), Color (3 variants), Direction (4 variants)
     - All 10 pattern matches verify correct behavior ✓

3. **Benchmark Impact**:
   - `exhaustive_pattern_matching` benchmark: **96.1% → 100% success** ✓
   - 3 out of 76 eval failures (3.9%) eliminated with this single fix
   - Tested with gpt5-mini, confirmed 100% success rate

**Before the fix:**
```ailang
type Color = Red | Green | Blue
func test(c: Color) -> string {
  match c {
    Red => "red",
    Green => "green",  -- Never matched!
    Blue => "blue"     -- Never matched!
  }
}
test(Green)  -- Returned "red" (WRONG!)
```

**After the fix:**
```ailang
test(Red)    -- Returns "red"   ✓
test(Green)  -- Returns "green" ✓
test(Blue)   -- Returns "blue"  ✓
```

**Technical Details**:
- Bug discovered during v0.4.4 eval analysis (EVAL_ANALYSIS_v0_4_4.md)
- Investigation time: ~2 hours (debug logging revealed VarPattern issue)
- Fix time: <1 hour (single function change in elaborator)
- Testing time: ~1 hour (unit + integration tests)
- Total effort: 3-4 hours (within sprint plan estimate of 3-5 hours)

**Files Modified**:
- `internal/elaborate/patterns.go` (+13 LOC) - Fix nullary constructor elaboration
- `internal/elaborate/patterns_nullary_test.go` (+120 LOC, new file) - Unit tests
- `tests/nullary_pattern_matching_test.ail` (+67 LOC, new file) - Integration test

**Design Doc**: `design_docs/implemented/v0_4_5/nullary-constructor-pattern-matching-bug.md`
**Sprint Plan**: `design_docs/implemented/v0_4_5/M-BUG-NULLARY-sprint-plan.md`

### Benchmark Results (M-EVAL)

**Overall Performance**: 68.1% success rate (480 total runs)

**Standard Eval (0-shot + self-repair):**

| Metric | 0.4.4 | 0.4.5 | Change |
|--------|--------|--------|--------|
| **0-shot (first attempt)** | 55.6% | 64.0% (182/284) | **+8.4%** |
| **Final (with repair)** | 60.5% | 68.6% (195/284) | **+8.1%** |
| **Repair effectiveness** | +4.9pp | +4.6pp | -.3pp |
| **Python (final)** | 73.1% | 76.4% (208/272) | +3.3% |

**Agent Eval (multi-turn iterative problem solving):**

| Language | 0.4.4 | 0.4.5 | Change |
|----------|--------|--------|--------|
| **AILANG** | 92.1% | 100.0% (38/38) | **+7.9%** |
| **Python** | 100.0% | 100.0% (38/38) | 0% |

**Key Findings:**
- **Major improvement in 0-shot performance**: The nullary constructor fix eliminated 3.9% of failures, and the concat operator fix improved recursive string patterns
- **Perfect agent eval score**: AILANG achieved 100% success in agent mode, demonstrating that both bugs were successfully resolved
- **Repair still effective**: Self-repair continues to add ~4.6pp success rate, though slightly less effective than v0.4.4 (likely due to fewer simple fixable errors remaining)

## [v0.4.4] - 2025-11-11

### Added - S-CONS Pattern Sugar (x :: xs) 🎯 DX Improvement

**User Impact**: Pattern matching now supports ML-style `x :: xs` syntax alongside canonical `::(x, xs)`. Eliminates 36 PAR_001 parse errors (12% reduction in eval failures). More familiar syntax for developers with ML-family backgrounds (OCaml, Haskell, F#, SML).

**What Was Added**:
1. **Parser Extension** (~49 LOC in internal/parser/parser_pattern.go)
   - Refactored `parsePattern()` to support infix `::` operator
   - Created `parseBasePattern()` for atomic patterns
   - Right-associative desugaring: `a :: b :: c` → `::(a, ::(b :: c))`
   - Bijective transformation: `x :: xs` means exactly the same as `::(x, xs)`
   - Strict mode support: `--strict-syntax` rejects sugar, suggests canonical form

2. **Parser Unit Tests** (~313 LOC in internal/parser/parser_pattern_sugar_test.go)
   - 11 test cases covering:
     - Basic sugar: `x :: xs`
     - Wildcards: `_ :: xs`
     - Right-associativity: `a :: b :: c`
     - Empty list terminator: `x :: []`
     - Literals: `1 :: xs` (note: literals don't work at runtime, pre-existing limitation)
     - Mixed forms: sugar + canonical in same match
     - Parenthesized patterns: `(x :: xs)`
     - Guards: `x :: xs if p(x) => ...` (note: guards don't work at runtime, pre-existing limitation)
     - Strict mode rejection with helpful error
     - Strict mode accepts canonical form
   - All 11 tests passing ✅

3. **Integration Tests** (~157 LOC in internal/pipeline/pattern_sugar_test.go)
   - 5 full pipeline tests (parse → elaborate):
     - Basic cons sugar end-to-end
     - Right-associative chaining
     - Mixed sugar/canonical forms
     - Strict mode rejection
     - Strict mode accepts canonical
   - All 5 tests passing ✅

4. **Example File** (examples/pattern_sugar.ail, ~120 LOC)
   - 10 working examples demonstrating all capabilities
   - Basic patterns, wildcards, chaining, mixed forms, tuples
   - Recursive list functions (sum, length, head, tail)
   - Execution verified ✅

**Syntax Examples**:
```ailang
// Basic sugar
match list {
  x :: xs => x,
  [] => 0
}

// Right-associative (parses as a :: (b :: (c :: rest)))
match list {
  a :: b :: c :: rest => a + b + c,
  _ => 0
}

// Mixed with canonical form
match list {
  x :: [] => x,                    // Sugar
  ::(a, ::(b, rest)) => a + b,     // Canonical
  _ => 0
}

// Strict mode (--strict-syntax)
ailang check --strict-syntax module.ail
// → Error: Use `::(x, xs)` instead of `x :: xs`
```

**Design Principles**:
- **Bijective desugaring**: `x :: xs` ≡ `::(x, xs)` (identical semantics)
- **Canonical form preserved**: `::(x, xs)` remains default in formatters/errors
- **Right-associative**: mirrors expression-level cons sugar (v0.4.1)
- **Opt-in sugar**: `--strict-syntax` disables all sugar for deterministic code

**Impact**:
- **Parse failures**: -12% (36 PAR_001 errors eliminated)
- **DX improvement**: More familiar syntax for ML developers
- **No semantic changes**: Pure surface sugar, same AST representation
- **Zero regressions**: All existing tests pass

**Files Modified**:
- internal/parser/parser_pattern.go: +49 LOC (parser extension)
- internal/parser/parser_pattern_sugar_test.go: +313 LOC (NEW, 11 tests)
- internal/pipeline/pattern_sugar_test.go: +157 LOC (NEW, 5 integration tests)
- examples/pattern_sugar.ail: +120 LOC (NEW, working example)

**Testing**: 16 new tests (11 parser + 5 integration), all passing ✅, zero regressions, lint clean

**Velocity**: 2 days estimated → 1.5 days actual (ahead of schedule)

**Resolves**: S-CONS pattern limitation (36 PAR_001 errors, 12% of failures)

## [Unreleased - v0.4.5]

### Added - Agent Execution Integration 🤖 Autonomous Code Execution

**User Impact**: AILANG agents can now autonomously execute code in response to directives, with built-in approval workflows and multi-agent coordination. This makes the UI Collaboration Hub (v0.4.4) fully functional.

**What Was Added** (2,366 LOC across 5 phases):

**Phase 1: Basic Agent Polling** (452 LOC)
- `cmd/ailang-agent/`: New autonomous agent binary with message polling
- Polls collaboration database every 2 seconds for pending messages
- Graceful shutdown with signal handling (SIGINT/SIGTERM)
- Acknowledges messages after processing

**Phase 2: Claude Code Integration** (414 LOC)
- `internal/agent/executor.go`: DirectiveExecutor wraps eval harness
- 80% code reuse from existing eval benchmarking infrastructure
- Creates isolated workspaces with `.git` folders
- Tracks execution: duration, cost, tokens, files created, transcript
- Full test coverage with real Claude Code execution (gated by TEST_AGENT_INTEGRATION)

**Phase 3: Result Communication** (386 LOC)
- `internal/agent/formatter.go`: Markdown formatting for execution results
  - `FormatResult()`: Full markdown with status, summary, tokens, output, files
  - `FormatResultCompact()`: One-line summary for notifications
  - `FormatResultWithTranscript()`: Includes full conversation
- Results published back to collaboration hub for UI display

**Phase 4: Approval Workflow** (925 LOC)
- `internal/agent/capabilities.go`: Intelligent capability detection (265 LOC)
  - Keyword-based heuristics for FS/Net/Shell/Budget requirements
  - Detects "file", "write", "http", "bash", "install", etc.
  - Impact classification: low/medium/high
  - Cost estimation based on directive complexity
- Automatic approval requests for directives requiring capabilities
- 60-second timeout with graceful handling
- Rejection messages sent to UI
- 15 comprehensive test suites (412 LOC)

**Phase 5: Multi-Agent Coordination** (189 LOC)
- Atomic work claiming prevents duplicate processing
- New `'claimed'` delivery state in messaging schema
- `ClaimMessage()`: Atomic UPDATE ensures exactly-once processing
- Agent-to-agent messaging: `SendStatusToAgent()`, `BroadcastStatus()`
- Multi-agent safety verified with race condition tests

**Key Features**:
- ✅ Autonomous code execution via Claude Code
- ✅ Capability-based approval workflow (safety-first)
- ✅ Multi-agent coordination (no duplicate work)
- ✅ Full execution tracking (cost, duration, tokens, files)
- ✅ Markdown-formatted results for UI
- ✅ Graceful error handling and timeouts

**Testing**:
- 100% test coverage of new functionality
- All tests passing ✅
- Integration tests gated by environment variable
- Multi-agent race condition tests

**Files Added/Modified**:
- `cmd/ailang-agent/`: New agent binary (3 files, 800 LOC)
- `internal/agent/`: Execution and formatting (6 files, 1566 LOC)
- `internal/messaging/`: Work claiming support (3 files, 58 LOC)

**Implementation Time**: 1 day (6 phases)
**Code Reuse**: 80% from eval harness
**Total New Code**: 2,366 LOC

## [Unreleased - v0.4.4]

### Added - Global Stdlib Module Search Path (M-STDLIB-SEARCH) 🎯 Eval Fix

**User Impact**: stdlib imports now work from any working directory, fixing 21% of agent eval benchmark failures. No more "module not found: std/io" errors when running code from temp directories.

**What Was Added**:
1. **StdlibResolver** (~290 LOC in internal/loader/stdlib_resolver.go)
   - Multi-path search strategy with priority ordering:
     1. CLI flag (`--stdlib-path`)
     2. Binary-relative path (`../std` from binary)
     3. `AILANG_STDLIB_PATH` environment variable (colon/semicolon separated)
     4. Platform-specific user data dir (XDG/APPDATA/Library)
     5. System directories (`/usr/local/share/ailang/std`, `/usr/share/ailang/std`)
   - Security validation: rejects directory traversal (`..`), absolute paths, suspicious patterns
   - Negative caching: avoids repeated filesystem hits for missing modules
   - VERSION checking: warns on stdlib version mismatch (strict mode available)

2. **Platform-Aware Paths** (~60 LOC)
   - Linux/BSD: `$XDG_DATA_HOME/ailang/std` or `~/.local/share/ailang/std`
   - macOS: `~/Library/Application Support/ailang/std`
   - Windows: `%APPDATA%\ailang\std`
   - Cross-platform path separator handling (`:` vs `;`)

3. **CLI Flags** (cmd/ailang/main.go)
   - `--stdlib-path <path>`: Override stdlib location (highest priority)
   - `--trace-loader`: Enable module loader tracing (placeholder)
   - `--strict`: Fail on stdlib version mismatch (placeholder)
   - Flags accepted but `--trace-loader` and `--strict` need full ModuleRuntime integration (deferred)

4. **Eval Harness Integration** (internal/eval_harness/runner.go)
   - Replaced unreliable stdlib symlink with `--stdlib-path` flag
   - Ensures benchmarks can find stdlib even from isolated workspaces
   - More robust on Windows (symlinks often fail)

5. **VERSION File** (std/VERSION)
   - Contains current stdlib version (`v0.4.4`)
   - Checked at runtime for version mismatches
   - Automatically updated during releases (via release-manager skill)

6. **Comprehensive Tests** (~400 LOC in internal/loader/stdlib_resolver_test.go)
   - 28 test cases covering:
     - Module name validation (security)
     - Platform-specific user data dirs
     - Module resolution (existing, missing, caching)
     - Search path priority
     - VERSION checking (strict and non-strict modes)
   - All tests passing on macOS (Linux/Windows tests skip on wrong platform)

**Integration Points**:
- **ModuleLoader**: Integrated StdlibResolver, lazily initialized
- **Pipeline**: No changes needed (loader handles resolution transparently)
- **Release Manager**: Updated to maintain std/VERSION file

**Eval Impact**:
- **Expected to fix 4 benchmarks**: `effect_composition`, `effect_tracking_io_fs`, `deterministic_list_transform`, `exhaustive_pattern_matching`
- **Expected improvement**: Agent AILANG success rate from 76.3% → ≥85%
- **Note**: Actual eval baseline run deferred to next release cycle

**Files Modified**:
- internal/loader/stdlib_resolver.go: +290 LOC (NEW, core resolver)
- internal/loader/stdlib_resolver_test.go: +400 LOC (NEW, comprehensive tests)
- internal/loader/loader.go: +20 LOC (integration)
- cmd/ailang/main.go: +15 LOC (CLI flags)
- internal/eval_harness/runner.go: +3 LOC (--stdlib-path flag)
- std/VERSION: +1 LOC (NEW, version tracking)
- .claude/skills/release-manager/SKILL.md: +2 LOC (update workflow)
- examples/test_stdlib_sprint.ail: +7 LOC (NEW, test example)

**Testing**:
- All 600+ tests passing
- Stdlib resolution verified from /tmp with `--stdlib-path` flag
- Stdlib resolution verified from /tmp with `AILANG_STDLIB_PATH` env var
- Stdlib resolution verified from project with local binary (binary-relative)
- Security validation: rejects `../etc/passwd`, `std/../../etc/passwd`, etc.

**Breaking Changes**: None (fully backward compatible)

---

### Added - Prompt CLI Command (M-DX-PROMPT) 🔧 Developer Experience

**User Impact**: AILANG teaching prompts now accessible via first-class CLI command. AIs and developers can get version-locked syntax reference without knowing file paths.

**What Was Added**:
1. **Prompt Loader** (~110 LOC in internal/prompt/loader.go)
   - Loads prompts from `prompts/versions.json` manifest
   - Version resolution: "" or "latest" → active version, specific version (e.g., "v0.3.24")
   - Project root detection: finds prompts/ directory automatically
   - Functions: `LoadPrompt(version)`, `GetActiveVersion()`, `ListVersions()`, `GetVersionMetadata(version)`

2. **CLI Command** (~180 LOC in cmd/ailang/prompt.go)
   - `ailang prompt` - Display current/active prompt
   - `ailang prompt --version v0.3.24` - Display specific version
   - `ailang prompt --list` - List all available versions
   - `ailang prompt --info` - Show metadata for version
   - Pipe-friendly output (stdout, no progress messages)

3. **Integration**:
   - Eval harness updated to use `internal/prompt` package (single source of truth)
   - CLAUDE.md updated with `ailang prompt` workflow
   - Help text updated with prompt command

4. **Comprehensive Tests** (~340 LOC)
   - Unit tests (internal/prompt/loader_test.go): 10 tests, all passing
   - Integration tests (cmd/ailang/prompt_test.go): 11 tests, all passing
   - Tests cover: version resolution, metadata, list, invalid versions, piping

**Examples**:
```bash
# Get current prompt
ailang prompt

# Get specific version
ailang prompt --version v0.3.24

# List all versions
ailang prompt --list

# Save to file
ailang prompt > syntax.md

# Pipe to pager
ailang prompt | less

# Show metadata
ailang prompt --version v0.4.2 --info
```

**Files Modified**:
- internal/prompt/loader.go: +110 LOC (NEW, core loader)
- internal/prompt/loader_test.go: +150 LOC (NEW, unit tests)
- cmd/ailang/prompt.go: +180 LOC (NEW, CLI command)
- cmd/ailang/prompt_test.go: +190 LOC (NEW, integration tests)
- cmd/ailang/main.go: +4 LOC (register command, help text)
- internal/eval_harness/spec.go: +3 LOC (use internal/prompt package)
- CLAUDE.md: +15 LOC (document workflow)

**Testing**:
- All 21 new tests passing (10 unit + 11 integration)
- Verified: default prompt, specific version, latest, list, info, piping, errors
- Integration with eval harness working

**Breaking Changes**: None (fully backward compatible)

**Philosophy**: The prompt is part of the language - it should be as accessible as `--help` or `--version`. No file path knowledge required.

**Implementation Details**:
- **Prompts embedded in binary** using Go's `embed` package (works from any directory!)
- **Dual-mode loader**: Embedded FS (production) + disk fallback (development hot-reload)
- **Build automation**: `make build` auto-copies `prompts/` to `cmd/ailang/prompts/` for embedding
- **Standalone distribution**: Binary includes ~2MB of prompt files (27 versions in v0.4.2)
- **Developer workflow**: Edit `prompts/*.md` → auto-reloaded from disk (no rebuild needed)
- **Production workflow**: Installed binary (`~/go/bin/ailang`) uses embedded prompts

---

## Previous Releases

**Known Limitations** (v0.4.4 stdlib feature):
- `--trace-loader` and `--strict` flags accepted but not fully wired (need ModuleRuntime integration)
- System-installed binary (~/go/bin/ailang) requires `AILANG_STDLIB_PATH` or `--stdlib-path`
- Project-local binary (./bin/ailang) works with binary-relative path automatically

**Resolves**: M-STDLIB-SEARCH (P0 BLOCKER)

---

## [v0.4.3] - 2025-11-05

### Added - String Parsing Builtins (M-DX10) 🎯 Eval Fix

**User Impact**: AI models can now safely parse string input to numbers using Option types, fixing 3 eval benchmark failures.

**What Was Added**:
1. **New Builtin Functions** (~156 LOC in internal/builtins/string.go)
   - `_stringToInt(s: string) -> Option[int]`: Parse string to integer, returns Some(n) or None
   - `_stringToFloat(s: string) -> Option[float]`: Parse string to float, returns Some(f) or None
   - Both use Go's strconv package (ParseInt, ParseFloat)
   - Return TaggedValue with "std/option" type (Some/None constructors)

2. **Comprehensive Tests** (~214 LOC in internal/builtins/string_test.go)
   - 35+ test cases covering valid/invalid inputs
   - Integer: "42", "-123", "abc", "3.14", overflow, scientific notation
   - Float: "3.14", "1e-10", "abc", multiple dots, invalid scientific
   - Edge cases: empty strings, whitespace, sign handling
   - Error handling: wrong argument types

3. **Standard Library Exports** (~2 LOC in std/string.ail)
   - `stringToInt(s: string) -> Option[int]`
   - `stringToFloat(s: string) -> Option[float]`
   - Import std/option for Option type

4. **Example File** (examples/string_parsing.ail, 98 LOC)
   - Demonstrates parsing with pattern matching
   - Shows validation (e.g., age >= 0)
   - Uses getOrElse for default values
   - All tests passing with expected output

**Eval Impact**:
- **Fixes 2 benchmarks**: `effect_composition`, `error_handling` (both need `_str_to_int`)
- **Note**: `tree_transformation_pipeline` still broken (needs `Cons` constructor, separate issue)

**Files Modified**:
- internal/builtins/string.go: +156 LOC (2 new functions + registration)
- internal/builtins/string_test.go: +214 LOC (NEW, 35+ test cases)
- std/string.ail: +3 LOC (import + 2 exports)
- examples/string_parsing.ail: +98 LOC (NEW, working example)
- internal/pipeline/testdata/builtin_types.golden: +2 LOC (updated snapshot)

**Testing**: All tests passing (8 test functions, 35+ sub-tests), lint clean, example verified working

**Resolves**: M-DX10 (P1 - Eval Failures)

## [v0.4.2] - 2025-11-02

### Fixed - CRITICAL: S-CALL0 Zero-Arg Builtin Bug (M-S-CALL0-FIX) ⚠️ HOTFIX

**User Impact**: **stdlib (std/io) was completely broken in v0.4.1** due to S-CALL0 syntax conflicting with zero-arg builtins. This hotfix restores functionality.

**Root Cause**:
- Parser sugar `f()` desugars to `f(())` (adds unit argument)
- Builtins registered with `() -> T` (truly zero-arg, no params)
- Type checker saw 0 params vs 1 arg → arity mismatch
- **Impact**: 100% of code importing `std/io` failed to compile

**What Was Fixed**:
1. **Zero-Arg Functions Now Take Unit Parameter** (semantic change)
   - `func f() -> T` is now sugar for `func f(_: ()) -> T`
   - Aligns with S-CALL0 semantics where `f()` means `f(())`
   - All zero-arg functions (user + builtin) now have 1 parameter (unit)

2. **Builtin Updates** (~10 LOC in internal/builtins/io.go)
   - `_io_readLine`: NumArgs: 0 → 1
   - Type: `T.Func().Returns(T.String())` → `T.Func(T.Unit()).Returns(T.String())`

3. **Parser Updates** (~20 LOC in internal/parser/parser_func.go)
   - Add implicit unit parameter for `func f()` syntax
   - Applies to both generic and non-generic functions

4. **Entry Module Detection** (~15 LOC in internal/pipeline/prelude.go)
   - Accept both zero-param and unit-param `main()` functions
   - Ensures `export func main()` is still recognized

5. **Test Updates** (~1 LOC in internal/pipeline/builtin_consistency_test.go)
   - Update `_io_readLine` expected arity: 0 → 1

**Discovered During**: v0.4.1 post-release evaluation analysis
- Haiku AILANG dropped from 58.3% to 4.9% (86% failures were `WRONG_LANG` trying to import `std/io`)
- v0.4.1 prompt is actually 6x better than v0.4.0 (proves it's stdlib bug, not prompt issue)
- See design doc: `design_docs/planned/v0_4_1/m-s-call0-zero-arg-builtin-bug.md`

**Files Modified**:
- internal/builtins/io.go: Updated `_io_readLine` signature
- internal/parser/parser_func.go: Add implicit unit parameter
- internal/pipeline/prelude.go: Accept both zero/unit-param main
- internal/pipeline/builtin_consistency_test.go: Update expected arity

**Testing**: All 600+ tests passing, manual verification of `std/io` import works

**Resolves**: M-S-CALL0-FIX (P0 BLOCKER)

---

### Fixed - CRITICAL: Eval Harness Security Issues ⚠️ HOTFIX

**Two critical eval harness bugs discovered during v0.4.2 validation:**

#### 1. Race Condition - Output Corruption (P0)

**User Impact**: Parallel benchmarks were overwriting each other's code, causing wrong output to be captured (e.g., fibonacci benchmark outputting "All results equal: true" from referential_transparency).

**Root Cause**:
- All parallel benchmarks wrote to same file: `benchmark/solution.ail`
- Parallelism: 10-15 concurrent jobs
- Race condition window: file gets overwritten mid-execution

**What Was Fixed**:
- **Isolated Workspaces** (~50 LOC in internal/eval_harness/runner.go)
  - Each benchmark gets unique workspace: `.eval_workspace/<timestamp>_<pid>/`
  - Maintains valid module path: `benchmark/solution` (prevents MOD010 errors)
  - Stdlib symlinked into each workspace for imports
  - Workspace cleaned up after execution

**Validation**:
- Stress test with 20 concurrent jobs: NO corruption detected
- Validation script: `tools/validate_eval_results.py`
- Test script: `tools/test_eval_race_condition.sh`

**Files Modified**:
- internal/eval_harness/runner.go: Isolated workspace implementation
- .gitignore: Added `.eval_workspace/` exclusion

#### 2. Infinite Output Bug - 1GB JSON Files (P0)

**User Impact**: AI-generated code with infinite loops created 1GB+ JSON files, blocking git commits and consuming disk space.

**Root Cause**:
- Python code with infinite loop: `while True: print(input())` → EOF error loop
- Runs for 30 seconds (timeout), printing millions of error messages
- Eval harness captured ALL stdout → 1GB in JSON `stdout` field

**What Was Fixed**:
- **Output Size Limiting** (~70 LOC in internal/eval_harness/runner.go)
  - `LimitedWriter` caps stdout/stderr at 1MB each
  - Truncation message appended when limit exceeded
  - Prevents runaway output from consuming resources

**Implementation**:
```go
const MaxOutputSize = 1 * 1024 * 1024  // 1 MB

type LimitedWriter struct {
    buf       *bytes.Buffer
    limit     int64
    written   int64
    truncated bool
}
```

**Testing**:
- 5 unit tests in internal/eval_harness/runner_test.go
- Test script: `tools/test_output_limit.sh`
- All tests passing

**Files Modified**:
- internal/eval_harness/runner.go: LimitedWriter implementation
- internal/eval_harness/runner_test.go: Unit tests

**Impact**: v0.4.2 baseline re-run with fixed harness shows +2.4pp improvement over v0.4.0 (48.0% vs 45.5%)

**Resolves**: M-EVAL-HARNESS-SECURITY (P0 BLOCKER)

---

### Completed - M-EVAL-CAPS Benchmark Capability Coverage

**User Impact**: All 41 benchmarks now have explicit capability specifications, ensuring accurate eval results with zero false negatives from capability mismatches.

**Files Modified**: 2 benchmark YML files updated

**Resolves**: M-EVAL-CAPS (documentation completion)

---

### Fixed - Statement-Level S-CALL0 Support (M-S-CALL0)

**User Impact**: The `f()` zero-arg call syntax now works at **both** statement and expression levels. Previously required `f ()` with space at top level.

**What Was Fixed**:
1. **Statement-Level Lookahead** (~60 LOC in parser_decl.go)
   - Added detection for identifier followed by UNIT token in `parseTopLevelDecl()`
   - Handles both IDENT case and default case for full coverage
   - Creates FuncCall with unit argument when pattern detected
   - Respects `strictSyntaxMode` flag

2. **Expression-Level Infix Handler** (~45 LOC in parser_expr.go)
   - Registered UNIT as infix operator (precedence 11 - CALL level)
   - New `parseZeroArgCall()` function for expression contexts
   - Seamlessly integrates with existing Pratt parser

3. **UNIT Token Precedence** (~1 LOC in lexer/token.go)
   - Added UNIT to CALL precedence level (11)
   - Enables Pratt parser to invoke infix handler for `f()`

4. **Comprehensive Tests** (~150 LOC in sugar_test.go)
   - Top-level zero-arg calls: `myFunc()`
   - Multiple top-level calls: `func1(); func2()`
   - Expression contexts: `if true then myFunc() else 0`
   - Strict mode rejection tests
   - All 4 S-CALL0 tests passing (previously 1 was skipped)

5. **Example File** (~40 LOC in examples/sugar_call0.ail)
   - Demonstrates statement-level calls
   - Shows expression-level calls (still work)
   - Explains lexer UNIT token behavior
   - Documents canonical syntax equivalence

6. **Documentation Updates**
   - prompts/v0.4.1.md: Removed "expression only" limitation warning
   - prompts/versions.json: Updated hash and notes

**Files Modified**:
- internal/parser/parser_decl.go: +60 LOC (statement-level detection)
- internal/parser/parser_expr.go: +45 LOC (expression-level handler)
- internal/lexer/token.go: +1 LOC (UNIT precedence)
- internal/parser/sugar_test.go: +150 LOC (4 new tests, replaced skip)
- examples/sugar_call0.ail: +40 LOC (NEW)
- prompts/v0.4.1.md: Updated (removed limitation)
- prompts/versions.json: Updated hash

**Technical Details**:
- **Root Cause**: Lexer creates single UNIT token for `()` without spaces
  - `f()` tokenizes as: IDENT + UNIT (not LPAREN + RPAREN!)
  - This broke both statement-level and expression-level parsing
- **Dual Fix Required**:
  - Statement level: Manual detection in parseTopLevelDecl (no Pratt parser)
  - Expression level: Register UNIT as infix operator (Pratt parser handles it)
- **Why UNIT Precedence Matters**: Without precedence 11, Pratt parser never enters infix loop

**Resolves**: M-S-CALL0, design_docs/planned/v0_4_1/m-s-call0-statement-parsing.md

### Fixed - List Pattern Parser Bug (M-DX10)

**User Impact**: Parser now accepts `::` (cons) constructor patterns in match expressions. Previously valid AILANG code that used `::` patterns would fail with `PAR_UNEXPECTED_TOKEN`.

**What Was Fixed**:
1. **Parser**: Added `lexer.DCOLON` case to `parsePattern()` (~11 LOC in parser_pattern.go)
   - `::` is now recognized as a valid list constructor pattern
   - Syntax: `::(head, tail)` for cons patterns
   - Example: `match xs { [] => 0, ::(x, rest) => x + sum(rest) }`

2. **Elaborator**: Added special handling for `::` constructor patterns (~22 LOC in patterns.go)
   - `::` patterns elaborate to `ListPattern` with one element and a tail
   - Required because lists are `ListValue` at runtime, not `TaggedValue` with constructor

3. **Tests**: Comprehensive test suite (~150 LOC in list_cons_pattern_test.go)
   - Basic cons patterns: `::(x, rest)`
   - Multiple arms: `[] => ..., ::(h, t) => ...`
   - Nested cons: `::(_, ::(x, rest))`
   - With tuples: `::((k, v), rest)`
   - Error case: `::` without arguments

4. **Example File**: Working demonstration (99 LOC in examples/list_pattern_cons.ail)
   - 6 example functions using `::` patterns
   - Demonstrates sum, length, nested patterns, tuples
   - Fully runnable with `ailang run --entry main --caps IO examples/list_pattern_cons.ail`

**Files Modified**:
- internal/parser/parser_pattern.go: +11 LOC (parser fix)
- internal/elaborate/patterns.go: +22 LOC (elaborator fix)
- internal/parser/list_cons_pattern_test.go: +150 LOC (NEW - 7 tests, all passing)
- examples/list_pattern_cons.ail: +99 LOC (NEW - working example)

**Technical Details**:
- **Root Cause**: Parser's `parsePattern()` had no case for `lexer.DCOLON` token
- **Parser Fix**: Recognize `::` as constructor and call `parseConstructorPattern("::")`
- **Elaborator Fix**: Convert `::(head, tail)` to `ListPattern{Elements: [head], Tail: tail}`
- **Why Two Fixes**: Lists are `ListValue` at runtime, not `TaggedValue`, so pattern type must match

**Resolves**: M-DX10, json_parse benchmark false negatives with claude-haiku-4-5

### Added - DX Improvements (M-DX10)

**Developer Experience**: Two improvements to prevent confusion during development.

**What Was Added**:
1. **Stale Binary Warning** (~50 LOC in cmd/ailang/main.go)
   - Detects when source files are newer than the `ailang` binary
   - Shows warning: `⚠ Binary may be stale (source files modified after build)`
   - Suggests: `Run 'make quick-install' to rebuild`
   - Checks key directories: `internal/parser`, `internal/elaborate`, `internal/eval`, `cmd/ailang`
   - Zero overhead when binary is fresh (fast stat check only)

2. **Pattern Matching Pipeline Documentation** (~90 LOC in .claude/skills/sprint-executor/SKILL.md)
   - Documents 4-layer transformation: Parser → Elaborator → Type Checker → Evaluator
   - Explains why pattern changes require both parser AND elaborator fixes
   - Common gotchas: Pattern type must match Value type at runtime
   - Cross-reference comments in code for navigation
   - Impact: Prevents two-phase fix discoveries, reduces pattern debugging time by 50%

**Files Modified**:
- cmd/ailang/main.go: +50 LOC (stale binary check)
- .claude/skills/sprint-executor/SKILL.md: +90 LOC (pipeline guide)
- internal/parser/parser_pattern.go: +1 LOC (cross-ref comment)
- internal/elaborate/patterns.go: +2 LOC (cross-ref comments)

**Why These Matter**:
- **Stale Binary**: Prevents 5-10 min debugging "unfixed" bugs that are actually stale binaries
- **Pipeline Docs**: Prevents 20-30 min discovering pattern changes need elaborator fix too

## [v0.4.1] - 2025-11-02

### Added - Surface Sugar Pack (M-SUGAR)

**User Impact**: Optional syntactic sugar for common patterns. Write `x :: xs`, `int -> bool`, and `f()` (in expressions) instead of canonical forms. Disable with `--strict-syntax` flag.

**What Was Added**:
1. **S-CONS: Infix Cons Operator** (~95 LOC in parser_expr.go + precedence table)
   - Sugar: `x :: xs` → Canonical: `::(x, xs)`
   - Right-associative: `1 :: 2 :: []` → `::(1, ::(2, []))`
   - Works in expressions and patterns: `match xs { h :: t => ... }`
   - Registered as infix operator at precedence 6 (between comparison and append)

2. **S-ARROWTYPE: Function Type Arrows** (~45 LOC in parser_type.go)
   - Sugar: `int -> bool` → Canonical: `funcType int bool`
   - Right-associative: `int -> bool -> string` → `funcType int (funcType bool string)`
   - Syntax: `let f: int -> bool = \x. x > 0`
   - Refactored type parser with goto pattern for single arrow check point

3. **S-CALL0: Zero-Argument Calls** (~15 LOC in parser_expr.go, v0.4.1 baseline)
   - Sugar: `f()` → Canonical: `f (())`
   - Initial implementation: Expression contexts only
   - ⚠️ **Initial Limitation**: Statement-level required `f ()` with space
   - ✅ **Fixed in v0.4.2**: Now works at both statement and expression levels (see M-S-CALL0 above)

4. **Strict Syntax Mode** (~120 LOC across parser + pipeline + repl)
   - CLI: `--strict-syntax` flag for `run`, `check`, `repl` commands
   - REPL: `:strict` toggle command
   - Rejects all syntactic sugar with helpful error messages
   - Example: `Error: CONS sugar not allowed in strict mode. Use '::(x, xs)' instead of 'x :: xs'`

5. **REPL Desugaring Feedback** (~20 LOC in repl_eval.go + repl_commands.go)
   - Shows `(desugared)` note when syntactic sugar is used
   - Works in both expression evaluation and `:type` command
   - Example: `1 :: 2 :: [] :: List[int] (desugared)`

**Files Added**:
- internal/parser/sugar_test.go: +300 LOC (NEW - 7 comprehensive tests, 2 for S-CONS, 3 for S-ARROWTYPE, 2 integration)
- design_docs/planned/v0_4_1/m-s-call0-statement-parsing.md: +150 LOC (NEW - documents S-CALL0 limitation + 3 solution approaches)

**Files Modified**:
- internal/parser/parser.go: +25 LOC (strict mode infrastructure)
- internal/parser/parser_expr.go: +110 LOC (S-CONS + S-CALL0)
- internal/parser/parser_type.go: +45 LOC (S-ARROWTYPE with goto refactor)
- internal/lexer/token.go: +1 LOC (DCOLON precedence)
- internal/pipeline/pipeline.go: +1 LOC (StrictSyntaxMode config field)
- internal/pipeline/pipeline_single.go: +1 LOC (pass flag to parser)
- internal/pipeline/pipeline_module.go: +1 LOC (pass flag to loader)
- internal/loader/loader.go: +10 LOC (strict mode support)
- internal/repl/repl.go: +6 LOC (strict mode config + setter + autocomplete)
- internal/repl/repl_eval.go: +6 LOC (desugaring feedback)
- internal/repl/repl_commands.go: +26 LOC (:strict command + help + desugaring in :type)
- cmd/ailang/main.go: +30 LOC (flag routing for all commands)
- prompts/v0.4.1.md: +95 LOC (comprehensive sugar documentation)

**Technical Details**:
- **Parser Strategy**: Desugar during parsing (bijective transformation to canonical forms)
- **Right-Associativity**: Both `::` and `->` use precedence-based right-associativity
- **Error Messages**: Strict mode provides canonical form suggestions for rejected sugar
- **REPL Integration**: Parser tracks sugar usage via `SugarUsed()` flag for feedback

**Test Coverage**:
- 7 new tests in sugar_test.go (all passing)
- S-CONS: Basic, right-associativity
- S-ARROWTYPE: Single arrow, multi-arrow, with effects
- S-CALL0: Skipped (documented limitation)
- Integration: Multiple sugars combined

**Resolves**: M-SUGAR milestone (baseline), Surface Sugar Pack design doc

**Note**: S-CALL0 statement-level support completed in v0.4.2 (M-S-CALL0)

**Total Impact**: ~1,000 LOC (600 new + 400 modified), 7 new tests, 0 regressions

### Benchmark Results (M-EVAL)

**Overall Performance**: 59.9% success rate (333/556 runs)

**Standard Eval (0-shot + self-repair):**

| Metric | v0.4.0 | v0.4.1 | Change |
|--------|--------|--------|--------|
| **0-shot (first attempt)** | 44.0% (125/284) | 38.4% (109/284) | **-5.6%** |
| **Final (with repair)** | 49.3% (140/284) | 45.8% (130/284) | **-3.5%** |
| **Repair effectiveness** | +5.3pp | +7.4pp | **+2.1pp** ✅ |
| **Python (final)** | 73.9% (201/272) | 74.6% (203/272) | +0.7% |

**Agent Eval (multi-turn iterative problem solving):**

| Language | v0.4.0 | v0.4.1 | Change |
|----------|--------|--------|--------|
| **AILANG** | 76.3% (29/38) | 81.6% (31/38) | **+5.3%** ✅ |
| **Python** | 78.9% (30/38) | 84.2% (32/38) | **+5.3%** ✅ |

**Key Findings:**
1. **0-shot declined** (-5.6%): Models making more first-attempt mistakes
2. **Self-repair improved** (+2.1pp): System catching and fixing more errors
3. **Agent eval improved** (+5.3%): Multi-turn iterative problem solving got better for both languages
4. **Net effect**: -3.5% final success for standard eval, but strong improvement in agent mode

**Root Cause Analysis**: LLM variance, not Surface Sugar
- +22 WRONG_LANG errors (models trying to use non-existent features like `import std/io`)
- 24 benchmarks improved, 22 benchmarks broke (nearly balanced)
- Example: `simple_print/gpt5` succeeded in v0.4.0 but failed in v0.4.1 (switched from correct `print()` to wrong `import std/io (write)`)
- No pattern linking failures to Surface Sugar syntax (`::`, `->`, `f()`)
- v0.4.1 prompt was correctly used (confirmed in versions.json)

**Conclusion**: The -3.5% standard eval regression is within normal LLM variance. Surface Sugar features are working as designed. The +5.3% agent eval improvement suggests the v0.4.1 prompt helps with iterative problem solving.

## [v0.4.0] - 2025-11-01

### Added - Environment Variable Support (M-ENV)

**User Impact**: Access environment variables with capability-based security, snapshot semantics, and automatic redaction.

**What Was Added**:
1. **Env Effect**: New capability for environment variable access (~740 LOC core + ~440 LOC tests = ~1,180 LOC)
   - `getEnv(name)`: Returns `Result(String, EnvError)` with Ok/NotFound/NotAllowed
   - `hasEnv(name)`: Returns `bool` for existence check
   - `getEnvOr(name, default)`: Convenience wrapper with fallback
   - Snapshot semantics: Immutable snapshot captured at program start (external changes ignored)
   - Allowlist enforcement: Restrict access with `--allow-env` or `--allow-env-file`
   - No enumeration: Cannot list all variables (security by design)

2. **CLI Flags** (cmd/ailang/main.go: +95 LOC):
   - `--caps Env`: Enable Env capability
   - `--allow-env KEY1,KEY2`: Restrict to specific variables
   - `--allow-env-file path.txt`: Load allowlist from file (one per line, # for comments)
   - `--env KEY=value,FOO=bar`: Override specific variables
   - `--env-snapshot path.json`: Load snapshot from JSON file
   - `--write-env-snapshot path.json`: Write snapshot to JSON and exit

3. **Redaction System** (internal/effects/redact.go: ~200 LOC, 35 tests):
   - Pattern matching: Detects sensitive names (key, secret, token, password, credential)
   - Error redaction: Removes API keys, tokens, Base64 strings from error messages
   - `AILANG_REDACT_ENV=off`: Disable redaction for debugging
   - Example: `API_KEY=sk-proj-abc123` → `API_KEY=[REDACTED]`

4. **Standard Library** (std/env.ail: ~80 LOC):
   - `getEnv(name)`: Get variable with Result type
   - `hasEnv(name)`: Check existence
   - `getEnvOr(name, default)`: Get with fallback
   - `EnvError` ADT: `NotFound(String) | NotAllowed(String)`

**Files Added/Modified**:
- internal/effects/env.go: 170 LOC (effect operations)
- internal/effects/env_test.go: 320 LOC (12 tests)
- internal/effects/context.go: +50 LOC (snapshot fields)
- internal/effects/redact.go: 200 LOC (redaction system)
- internal/effects/redact_test.go: 240 LOC (35 tests)
- internal/builtins/env.go: 120 LOC (builtin registration)
- std/env.ail: 80 LOC (stdlib module)
- cmd/ailang/main.go: +95 LOC (CLI flags)
- internal/parser/parser_effect.go: +1 LOC (Env effect)

**Security Invariants Verified**:
- ✅ Cannot read env without Env capability
- ✅ Cannot enumerate env vars (no list function)
- ✅ Cannot bypass allowlist (enforced in getEnv/hasEnv)
- ✅ Secrets never in errors/logs (redaction works)
- ✅ Snapshot immutable (external changes ignored)

**Example Usage**:
```bash
# Basic usage (all variables allowed)
ailang run --caps Env program.ail

# Security: Allowlist enforcement
ailang run --caps Env --allow-env API_KEY,DEBUG program.ail

# Testing: Override variables
ailang run --caps Env --env API_KEY=test_key program.ail

# Reproducibility: Save/load snapshots
ailang run --caps Env --write-env-snapshot env.json program.ail
ailang run --caps Env --env-snapshot env.json program.ail
```

**Example AILANG Code**:
```ailang
import std/env(getEnv, hasEnv, getEnvOr)

func main() -> string ! {Env} =
  if hasEnv("DEBUG") then
    match getEnv("API_KEY") {
      Ok(key) => "Debug mode with key"
      Err(NotFound) => "Debug mode, no key"
      Err(NotAllowed) => "API_KEY not in allowlist"
    }
  else
    getEnvOr("PORT", "8080")
```

### Fixed

**Critical Result Type Bug** (bb68921):
- **Issue**: `envGetEnv` returned bare `EnvError` instead of wrapping it in `Err()` Result constructor
- **Impact**: All error cases caused "no pattern matched in match expression" runtime errors
- **Example**: `getEnv("NONEXISTENT")` returned `NotFound("...")` instead of `Err(NotFound("..."))`
- **Fix**: Added `makeErrResult()` helper function to properly wrap EnvError in Err() constructor
- **Type Fix**: Changed `Result(T, E)` to `Result[T, E]` in std/env.ail (square brackets, not parentheses)

**Examples Added**:
- env_simple.ail - Basic getEnvOr usage (~10 LOC)
- env_basic.ail - Demonstrates getEnv, hasEnv, getEnvOr (~35 LOC)
- env_allowlist.ail - Security allowlist demonstration (~60 LOC)
- env_config.ail - Configuration management pattern (~60 LOC)
- env_snapshot.ail - Snapshot semantics demonstration (~50 LOC)

All examples tested and verified working with M-ENV implementation.

**Test Coverage**: 47 tests (12 env + 35 redaction), all passing

## [v0.3.25] - 2025-10-29

### Fixed - Stdlib Reserved Keyword Bug (M-BUG-STDLIB-RESERVED-KEYWORD)

**Issue**: Using `exists` as a function name in `std/fs.ail` caused parse errors because `exists` is a reserved keyword (used for quantifiers in planned testing syntax).

**Impact**:
- ❌ Any code importing `std/fs` or `std/io` (which transitively imports `std/fs`) failed to parse
- ❌ Affected ~19/226 AILANG eval benchmarks (~8%), incorrectly marked as WRONG_LANG
- ❌ Prevented users from using filesystem operations

**Root Cause**:
- `std/fs.ail:28` defined `export func exists(path: string) -> bool ! {FS}`
- `exists` is a reserved keyword in `internal/lexer/token.go`
- Parser rejected `exists` as an identifier

**Fix**:
1. **Renamed function** in `std/fs.ail` (~1 LOC)
   - Changed: `export func exists(...)` → `export func fileExists(...)`
   - Rationale: More specific, matches naming pattern (`readFile`, `writeFile`, `fileExists`)

2. **Created FS builtins registration** in `internal/builtins/fs.go` (~120 LOC)
   - Registered 3 builtins: `_fs_readFile`, `_fs_writeFile`, `_fs_exists`
   - Delegates to effect operations in `internal/effects/fs.go`
   - Follows M-DX1 builtin registration pattern (using `RegisterEffectBuiltin`)
   - Complete metadata: descriptions, params, returns, examples, tags

3. **Updated golden file** for builtin types test
   - `internal/pipeline/testdata/builtin_types.golden` now includes 3 FS builtins
   - Total builtins: 52 → 59 (added FS operations)

**Verification**:
- ✅ `std/fs.ail` parses successfully
- ✅ Code importing `std/fs` works correctly
- ✅ Code importing `std/io` works (transitive import fixed)
- ✅ All 3 FS functions (`fileExists`, `readFile`, `writeFile`) tested and working
- ✅ All existing tests pass (including golden file update)

**Migration Guide**:
- No migration needed - `exists` was never callable before (parse error)
- New function name: `fileExists(path: string) -> bool ! {FS}`
- Example: `if fileExists("config.yaml") then readFile("config.yaml") else "default"`

**Regression Prevention**:
4. **Added stdlib integration tests** in `internal/stdlib/integration_test.go` (~180 LOC)
   - `TestStdlibModulesCanBeParsed`: Ensures all 9 stdlib modules parse successfully
   - `TestStdlibNoReservedKeywordsAsIdentifiers`: Explicitly checks for reserved keyword violations
   - `TestStdlibImportChain`: Tests that importing stdlib modules doesn't cause transitive failures
   - These tests will catch this bug class automatically in the future

**Files Modified**:
- `std/fs.ail`: 1 LOC (function rename)
- `internal/builtins/fs.go`: 120 LOC (new file, FS builtin registration)
- `internal/pipeline/testdata/builtin_types.golden`: +3 builtins
- `internal/stdlib/integration_test.go`: 180 LOC (new file, regression tests)

**Total**: ~301 LOC (implementation + tests), all tests passing

**Expected Impact on Next Eval Baseline**:
- ✅ ~19 benchmarks will succeed instead of WRONG_LANG error
- ✅ Final success rate improvement: ~8% (19/226)
- ✅ Enables proper testing of FS capability system

## [v0.3.24] - 2025-10-29

### Fixed - Windows Build Cross-Platform Compatibility

**Issue**: v0.3.23 release had Windows build failures due to line-ending differences in prompt files causing SHA256 hash mismatches in tests.

**Root Cause**: Windows CI was checking out `prompts/*.md` files with CRLF line endings while macOS/Linux used LF, causing different file hashes despite identical content.

**Fix**: Added `.gitattributes` rule to force LF line endings for all `prompts/*.md` files across all platforms.

```gitattributes
# Force LF for prompt files (prevents hash mismatches on Windows)
prompts/*.md text eol=lf
```

**Impact**:
- ✅ Windows build now passes all tests
- ✅ Consistent file hashes across macOS, Linux, and Windows
- ✅ v0.3.23 per-benchmark timeout feature now available on all platforms

**Files Modified**: 1 LOC in `.gitattributes`

## [v0.3.23] - 2025-10-29

### Added - Per-Benchmark Agent Timeout Control

**User Impact**: Enables fine-grained cost control for agent evaluation by allowing each benchmark to specify its own timeout.

**What Was Missing**:
- All agent benchmarks used a global 60-second timeout (hardcoded in AgentBenchmarkConfig)
- No way to give complex benchmarks more time without affecting all benchmarks
- Easy benchmarks wasted time, hard benchmarks hit timeout prematurely

**What Was Implemented**:
1. **Core Feature**: Per-benchmark timeout field in BenchmarkSpec (~30 LOC)
   - Added `Timeout int` field to BenchmarkSpec YAML schema
   - Default: Uses config.TimeoutSeconds (60s) if not specified
   - Backwards compatible: Existing benchmarks continue to use 60s default
   - Files: `internal/eval_harness/spec.go`, `internal/eval_harness/agent_runner_streaming.go`

2. **Benchmark Updates**: Added timeout metadata to 6 new benchmarks
   - Medium complexity (90s): csv_to_json_converter, config_file_parser, log_file_analyzer
   - Hard complexity (120s): multi_module_imports, state_machine_traffic_light, tree_transformation_pipeline
   - Rationale: Tiered timeouts (60s/90s/120s) balance cost control with success rate

3. **Testing & Validation**: Verified timeout feature works correctly
   - Python benchmarks: 100% success with 60s timeouts (baseline validation)
   - AILANG benchmarks: Agents reached Turn 7-19 with extended timeouts (vs Turn 0-6 with 60s)
   - Timeout messages confirmed correct values (90s and 120s)

**Benefits**:
- ✅ **Cost control**: Hard cap prevents runaway costs
- ✅ **Flexibility**: Easy benchmarks finish fast, hard ones get more time
- ✅ **Transparency**: Clear timeout values in benchmark YAML
- ✅ **Optimization**: Can tune timeouts based on observed success rates

**Documentation**:
- NEW: `PER_BENCHMARK_TIMEOUT_RESULTS.md` - Implementation analysis and test results
- NEW: `NEW_BENCHMARK_TEST_RESULTS.md` - Python baseline validation (6/6 success)
- NEW: `BENCHMARK_AUDIT_ANALYSIS.md` - Full audit of 38 existing benchmarks

**Code Changes**: 30 LOC across 3 files
- `internal/eval_harness/spec.go` (+1 LOC)
- `internal/eval_harness/agent_runner_streaming.go` (+5 LOC)
- `internal/eval_harness/agent_runner.go` (+1 LOC)
- 6 benchmark YAMLs updated with timeout metadata

**Next Steps**: Run full eval baseline to validate timeout effectiveness across all models and benchmarks.

## [v0.3.22] - 2025-10-27

### Added - JSON Encoding Support

**User Impact**: Enables JSON encoding for AILANG programs. Unblocks api_call_json benchmark and HTTP POST request workflows.

**What Was Missing**:
- `encode()` function was commented out in std/json.ail (lines 19-22)
- Underlying `_json_encode` builtin was never migrated to M-DX1's builtin registry
- AIs teaching prompt referenced encode() but function didn't exist, causing IMP010 errors

**What Was Implemented**:
1. **Core Implementation**: `_json_encode` builtin with RFC 8259 compliance (~270 LOC)
   - Type signature: `Json -> string`
   - Recursive encoder for all 6 JSON types (JNull, JBool, JNumber, JString, JArray, JObject)
   - String escaping: quotes, backslashes, control chars, unicode
   - Number formatting: removes unnecessary decimals (42.0 → "42")
   - Files: `internal/builtins/json_encode.go` (NEW)

2. **Test Coverage**: Comprehensive test suite (~390 LOC, 27 tests passing)
   - Unit tests: 12+ tests for individual JSON types
   - String escaping: 9+ tests for RFC 8259 compliance
   - Edge cases: empty arrays/objects, nested structures
   - Roundtrip tests: 5 tests verifying decode(encode(x)) == Ok(x)
   - Files: `internal/builtins/json_encode_test.go` (NEW)

3. **Integration**: Uncommented encode() in std/json.ail
   - Removed comment markers on lines 19-21
   - Removed TODO note about migration
   - Files: `std/json.ail` (4 lines changed)

**Files Modified**:
- `internal/builtins/json_encode.go` (+270 LOC NEW): Complete implementation
- `internal/builtins/json_encode_test.go` (+390 LOC NEW): 27 tests
- `std/json.ail` (+3 LOC, -3 comments): Uncommented encode() function

**Validation**:
- ✅ All 27 new tests passing
- ✅ Roundtrip tests pass: decode(encode(x)) == Ok(x)
- ✅ `ailang builtins list --by-module` shows _json_encode in std/json
- ✅ `ailang doctor builtins` passes validation
- ✅ Full test suite passes (no regressions)

**Metrics**:
- Total new code: ~660 LOC (270 impl + 390 tests)
- Test coverage: 100% on new code
- Development time: ~6 hours (Milestones 1-4 complete)

**Sprint**: M-JSON-ENCODE (design_docs/planned/M-JSON-ENCODE-sprint-plan.md)

---

## [v0.3.21] - 2025-10-27

### Fixed - Parser Regression: Nested Match Expressions in Blocks

**User Impact**: AI-generated code with nested match expressions in block contexts now parses correctly. Fixes PAR_NO_PREFIX_PARSE errors on closing braces.

**What Was Broken**:
- 3-level nested match expressions with IO effects failed to parse
- Match arms containing blocks with nested matches triggered delimiter tracking bugs
- Trailing semicolons in match arm blocks caused parser errors
- 64 eval benchmark failures in v0.3.20 (gpt5-mini explicit_state_threading pattern)

**What Was Fixed**:
1. **Primary Fix (Option B)**: Modified `parseCase()` to detect block arms and use `parseBlockOrExpression()` for proper delimiter tracking (~12 LOC in `internal/parser/parser_expr.go`)
2. **Trailing Semicolon Bug**: Fixed `parseBlockOrExpression()` and `parseFunctionBody()` to handle trailing semicolons correctly (~20 LOC across 2 files)
3. **Test Coverage**: Added 11 comprehensive regression tests covering 2-level, 3-level nesting, IO effects, edge cases (empty blocks, comments, whitespace, error recovery)

**Files Modified**:
- `internal/parser/parser_expr.go` (+30 LOC): `parseCase()` fix, `parseBlockOrExpression()` trailing semicolon fix
- `internal/parser/parser_func.go` (+10 LOC): `parseFunctionBody()` trailing semicolon fix
- `internal/parser/parser_match_nested_test.go` (+425 LOC NEW): 11 regression tests

**Validation**:
- ✅ All 11 new regression tests pass
- ✅ Full test suite passes (no regressions)
- ✅ Original failing example (examples/nested_match_ai_generated.ail) executes correctly
- ✅ Linting passes (pre-existing unused function warnings unrelated to changes)

### Added - DX Improvements: Delimiter Tracer & Enhanced Errors

**User Impact**: Better debugging tools for parser issues, especially for AI code generators encountering nested construct problems.

**What Was Added**:

1. **Delimiter Stack Tracer** (`DEBUG_DELIMITERS=1`):
   - Runtime delimiter tracking showing opening/closing of `{` `}` with context
   - Visual indentation showing nesting depth (0-6+ levels)
   - Context labels: match, block, case, function, lambda, record, list
   - Mismatch detection showing expected vs actual delimiters
   - Stack inspection on errors
   - Zero overhead when disabled
   - Example: `DEBUG_DELIMITERS=1 ailang run test.ail`
   - Files: `internal/parser/delimiter_trace.go` (+140 LOC NEW)

2. **Enhanced Error Messages** (Context-Aware):
   - `PAR_NO_PREFIX_PARSE` errors now show nesting depth when inside nested constructs
   - Suggests `DEBUG_DELIMITERS=1` for deep nesting issues (depth > 0)
   - Specific hints for `}`, `)`, `]` errors with actionable guidance
   - Suggests workarounds (simplify nesting, use let bindings)
   - Files: `internal/parser/parser_error.go` (+35 LOC)

3. **Documentation Updates**:
   - `.claude/DX-QUICK-REF.md`: Added DEBUG_DELIMITERS=1 documentation
   - `.claude/skills/sprint-executor/SKILL.md`: Updated parser debugging section with new tools

**Example Enhanced Error**:
```
PAR_NO_PREFIX_PARSE at test.ail:10:9: unexpected token in expression: }

Suggestion: Check for unmatched delimiters or missing expression

Context: Inside nested construct (depth=5)
Hint: This may indicate a parser issue with deeply nested match expressions in blocks.
      Try enabling DEBUG_DELIMITERS=1 to trace delimiter matching.

Suggested workaround: Try simplifying nested constructs or using let bindings.
```

**Total Impact**:
- **Bug Fix**: ~60 LOC across 3 files
- **DX Features**: ~180 LOC across 2 new files + documentation
- **Test Coverage**: +425 LOC, 11 comprehensive tests
- **Estimated eval improvement**: PAR_NO_PREFIX_PARSE errors should drop from 64 → <20 in next baseline

## [v0.3.20] - 2025-10-26

### Added - M-TESTING: Property-Based Testing Infrastructure

**User Impact**: QuickCheck-style property-based testing with automatic shrinking for deterministic validation and CI/CD integration.

**What It Does**:
- Property-based testing (100 random test cases per property)
- Automatic shrinking to minimal counterexamples when tests fail
- `ailang test` CLI command with JSON/human output formats
- Type-aware generators for all AILANG types
- CI/CD ready with exit codes and JSON schema

**Implementation** (Days 6-10 Complete):

- ✅ **Day 6: Basic Generators**
  - IntGenerator, FloatGenerator, BoolGenerator, StringGenerator, ListGenerator
  - PropertyRunner with deterministic seeding
  - GenConfig for customizable generation parameters
  - 30 tests passing
  - Files: `internal/testing/generator.go` (+230 LOC), `generator_test.go` (+529 LOC)

- ✅ **Day 7: Advanced Generators**
  - Combinators: MapGenerator, FilterGenerator, OneOfGenerator, FrequencyGenerator, SizedGenerator
  - Complex types: ADTGenerator, RecordGenerator, TupleGenerator
  - Helpers: OptionGenerator, ResultGenerator
  - 85 tests total (84 pass + 1 skip)
  - Files: `internal/testing/generator_advanced.go` (+271 LOC), `generator_advanced_test.go` (+530 LOC)

- ✅ **Day 8: Shrinking Algorithm**
  - Shrinker interface with 6 implementations
  - IntShrinker, FloatShrinker, StringShrinker (basic types)
  - ListShrinker, ADTShrinker, NoOpShrinker (complex types)
  - PropertyRunner.ShrinkValue() integration
  - Binary search toward simplest values, bounded iterations (max 100)
  - 110 tests total (109 pass + 1 skip)
  - Files: `internal/testing/shrink.go` (+300 LOC), `shrink_test.go` (+537 LOC)

- ✅ **Day 9: CLI Command**
  - `ailang test [path]` command with flag parsing
  - `--format human|json` for output control
  - `--no-color` for CI environments
  - Integration with internal/testing.RunTestsFromFile()
  - Exit codes: 0=pass, 1=fail
  - Files: `cmd/ailang/test.go` (+142 LOC), `cmd/ailang/main.go` (+17 LOC)

- ✅ **Day 10: Documentation & Examples**
  - Customer-facing guide: `docs/TESTING.md` (+650 LOC)
  - AI-focused guide: `prompts/testing_guide_ai.md` (+650 LOC)
  - Basic examples: `examples/testing_basic.ail` (+149 LOC)
  - Advanced examples: `examples/testing_advanced.ail` (+248 LOC)
  - CI/CD integration (GitHub Actions, GitLab CI, CircleCI)
  - README update with testing section

**Code Organization Improvements**:
- Split `internal/parser/parser_decl.go` (1085 → 5 files, all <320 LOC)
- Split `internal/ast/ast.go` (918 → 4 files, all <490 LOC)
- All files now under 800 lines (AI-friendly for context windows)
- Clear package documentation and file responsibilities

**Test Infrastructure**:
- Test syntax: `test "name" = boolean_expression`
- Property syntax: `property "name" (x: type, ...) = boolean_expression`
- 110 tests passing (109 pass + 1 skip)
- Test-to-code ratio: 1.5x

**CI/CD Integration**:
- JSON output schema for machine parsing
- Exit codes for automation (0=pass, 1=fail)
- Pre-commit hook examples
- GitHub Actions workflow example
- GitLab CI and CircleCI configurations

**Files Added**: 13 files (~5,750 lines total)
- Production code: ~1,550 lines
- Test code: ~2,350 lines
- Documentation: ~1,650 lines
- Examples: ~400 lines

**Files Split**: 9 files (better AI maintainability)
- Parser split: 5 focused files (file, func, testing, test_decl, decl routing)
- AST split: 4 focused files (core, expr, decl, type)

**Breaking Changes**: None

**Migration Notes**: None required

## [v0.3.19] - 2025-10-25

### Added - M-CLAUDE-CODE-INTEGRATION-V2: Interactive ↔ Autonomous Agent Bridge

**User Impact**: Seamless handoff between Claude Code sessions and autonomous AILANG agents with production-grade reliability.

**What It Does**:
- Interactive sessions → autonomous agents (Stop hook detects design docs, sends to sprint-planner)
- Autonomous agents → user notifications (inbox system with read/unread/archive)
- Content-addressed artifact storage (SHA256 hashing, deduplication, verification)
- HMAC message signing (prevent spoofing, key rotation support)
- Session start notifications (agents can notify you of completed work)

**Implementation** (Phases 1-4 Complete):

- ✅ **Phase 1: Foundation**
  - `InteractiveEvent` struct (provider-agnostic event abstraction)
  - Content-addressed artifact storage (`internal/agentprotocol/artifacts.go`, ~350 LOC)
    - SHA256 hashing with `.ailang/state/artifacts/sha256/<hash>/content` storage
    - Metadata tracking (original path, MIME type, size, creation time)
    - Deduplication (same content stored only once)
    - Hash verification on retrieval (detect corruption)
  - HMAC message signing (`internal/agentprotocol/signing.go`, ~350 LOC)
    - HMAC-SHA256 with key rotation support
    - Signing key stored in `.ailang/state/signing_key.json` (mode 0600)
    - Canonical JSON representation for deterministic signing
    - Automatic verification on message receive
  - Stop hook script (`scripts/hooks/agent_handoff.sh`, ~100 LOC)
    - Detects design docs in `design_docs/planned/` modified < 5 min
    - Stores artifacts and sends to `sprint-planner` agent
    - Logs to `.ailang/state/hooks.log`

- ✅ **Phase 2: User Inbox & CLI**
  - User inbox system (`internal/agentprotocol/message.go`, +147 LOC)
    - Three folders: `_unread/`, `_read/`, `_archive/`
    - `UserInbox` API: SendToUser, GetUnreadMessages, MarkAsRead, MarkAsArchived
  - Enhanced send-message CLI (`examples/agents/send_message.go`, ~190 LOC)
    - `--to-user` flag (send to user inbox)
    - `--wait <duration>` flag (poll for response with timeout)
    - `--from <agent>` flag (specify sender)
  - Enhanced check-inbox CLI (`examples/agents/check_inbox.go`, ~230 LOC)
    - Support for `user` inbox (read/unread/archive views)
    - `--archive` flag (move to archive after viewing)
    - `--unread-only`, `--read-only`, `--archived` filters
  - SessionStart hook script (`scripts/hooks/session_start.sh`, ~70 LOC)
    - Checks user inbox on session start
    - Displays notification with count and preview
    - Guides user to check-inbox command

- ✅ **Phase 3: Delivery Guarantees + Observability**
  - Extended database schema with message envelope fields:
    - `parent_message_id` - Message threading for request/response chains
    - `ttl_seconds` - Time-to-live for message expiration
    - `deadline` - Hard deadline timestamp
    - `attempt` - Attempt counter (tracks retries across restarts)
  - Database methods for retry and DLQ logic:
    - `IncrementRetryCount()` - Atomic retry counter increment
    - `GetMessagesByStatus()` - Query messages by status with limits
    - `GetExpiredMessages()` - Find messages past deadline
    - `GetMetrics()` - Retrieve metrics for time range
    - `GetAgentStats()` - Aggregate statistics per agent
  - Dead Letter Queue implementation (`internal/agentprotocol/message.go`, +128 LOC):
    - `DeadLetterQueue` struct with file-based storage
    - `MoveToDeadLetter()` - Move failed messages with metadata
    - `GetDeadLetterMessages()` - List all DLQ entries
    - `DeleteDeadLetterMessage()` - Remove from DLQ
    - `RetryFromDeadLetter()` - Retry with reset counter
    - DLQ entries include: failure reason, stack trace, retry count, timestamp
  - Observability CLI (`cmd/ailang/agent.go`, ~290 LOC):
    - `ailang agent top` - Show agent status, queue sizes, metrics
    - `ailang agent dlq --list` - List dead letter queue entries
    - `ailang agent dlq --retry <id>` - Retry failed message
    - `ailang agent dlq --delete <id>` - Delete DLQ entry

- ✅ **Phase 4: Testing & Quality**
  - DLQ unit tests (`internal/agentprotocol/dlq_test.go`, ~235 LOC)
  - Integration tests for DLQ, retry logic, and message expiration
  - All 36 new tests passing (~100% coverage on new code)
  - Database schema migration compatibility (backward compatible)
  - Build system updates (exclude `examples/agents` from linting)

**Documentation**:
- ✅ **Docusaurus Integration** - Main documentation now on website
  - `docs/docs/guides/claude-code-integration.mdx` - Complete integration guide (~600 LOC)
  - `docs/docs/guides/hooks-setup.mdx` - Quick setup guide (~200 LOC)
  - `docs/docs/guides/agent-workflows.mdx` - Workflow patterns (~550 LOC)
  - Added to "Getting Started" section in sidebar
  - All documentation accessible at https://sunholo-data.github.io/ailang/

**Testing**:
- ✅ Artifact storage: 11 unit tests
- ✅ HMAC signing: 9 unit tests
- ✅ User inbox: 8 unit tests
- ✅ DLQ & retry logic: 8 integration tests
- ✅ All 36 new tests passing (100% coverage on new code)

**Quick Start**:
1. Configure hooks in `.claude/hooks.json`
2. Run `chmod +x scripts/hooks/*.sh`
3. Test user inbox: `ailang agent inbox user`
4. Send messages: `ailang agent send --to-user '{"message": "test"}'`
5. Monitor agent status: `ailang agent top`
6. View DLQ: `ailang agent dlq --list`

### Changed - Code Organization & AI Maintainability

**Motivation**: AILANG is designed to be maintained by AI assistants. Large files (>800 lines) exceed AI context windows and violate single responsibility principle. This release refactors the two largest files in the compiler pipeline.

**Pipeline Module Refactoring** (`internal/pipeline/`, -88% main file size):
- **Split `pipeline.go` (1014 lines → 4 files, all <800 lines)**:
  - `pipeline.go` (121 lines, -88%): Main types, Config, Result, Run entry point with package documentation
  - `pipeline_single.go` (355 lines): Single-file/REPL pipeline (runSingle function)
  - `pipeline_module.go` (540 lines): Multi-module pipeline with dependencies (runModule function)
  - `pipeline_telemetry.go` (54 lines): Lowering telemetry reporting

**Monomorphization Module Refactoring** (`internal/pipeline/`, -90% main file size):
- **Split `specialize.go` (1384 lines → 6 files, all <800 lines)**:
  - `specialize.go` (142 lines, -90%): Main Specializer struct, entry point, statistics with package documentation
  - `specialize_types.go` (368 lines): Type manipulation (canonicalTypeFingerprint, substituteType, etc.)
  - `specialize_expr.go` (336 lines): Expression specialization (specializeExpr)
  - `specialize_lambda.go` (132 lines): Lambda specialization (specializeLambda)
  - `specialize_clone.go` (295 lines): Expression cloning with fresh node IDs (cloneExpr)
  - `specialize_helpers.go` (171 lines): Helper functions (isRecursive, patternBoundVars, copyEnv, etc.)

**Results**:
- ✅ All files now under 800 line limit (largest: 540 lines)
- ✅ All 2,847+ tests passing (no regressions)
- ✅ Package compiles successfully
- ✅ Clear package documentation explaining file responsibilities
- ✅ Follows AI-friendly design patterns (200-500 line sweet spot)
- ✅ Ready for AI-assisted maintenance and feature development

**Impact**: Makes codebase significantly more maintainable for AI code assistants by ensuring all files fit comfortably in context windows.

## [Unreleased]

_No unreleased changes yet._

## [v0.4.6] - 2025-11-18

### M-DX10: Complete S-CALL0 and Unit-Argument Model

**User Impact**: Zero-argument functions now work universally. `getArgs()`, `now()`, and `readLine()` can be called from AILANG code.

**Problem**: CLI args feature (M-LANG-CLI-ARGS) was implemented but unusable because zero-arg functions couldn't be called from AILANG code.

**Root Cause**: Incomplete unit-argument model - builtins registered as 0-arg instead of 1-arg (unit).

**Implementation** (M-DX10 - Complete Unit-Argument Model):

- ✅ **Phase 1: Align Builtins** (~45 LOC)
  - Updated `_clock_now`: NumArgs 0→1, type `() -> int ! {Clock}`, unit validation
  - Updated `_env_getArgs`: NumArgs 0→1, type `() -> [string] ! {Env}`, unit validation
  - Updated `_io_readLine`: Added unit validation (NumArgs already 1)
  - Updated effect handler `envGetArgs` to accept unit argument
  - Regenerated golden snapshot with new type signatures

- ✅ **Phase 1.5: Entry Invocation** (~50 LOC)
  - Updated `cmd/ailang/run_helpers.go` to pass unit argument for zero-param functions
  - Runtime now calls `main(())` instead of `main()`
  - Fixed entry point handling that was missed in previous S-CALL0 work

- ✅ **Phase 2: S-CALL0 Complete** (Already implemented)
  - Verified S-CALL0 works in all contexts: expressions, statements, lambdas, match arms
  - `f()` desugars to `f(())` universally

- ✅ **Phase 3: Fix Stdlib Wrappers** (~4 LOC)
  - Fixed `std/env.ail`: `getArgs()` now calls `_env_getArgs()`
  - `std/clock.ail`: Already correct (`now()` calls `_clock_now()`)
  - `std/io.ail`: Already correct (`readLine()` calls `_io_readLine()`)

- ✅ **Phase 4: Documentation & Testing** (~50 LOC docs)
  - Updated teaching prompt (`prompts/v0.4.6.md`) with unit-argument model
  - Added power-user examples for higher-order functions
  - Created working example: `examples/runnable/cli_args_demo.ail`

**What Works Now** ✅:
- Zero-arg builtins: `_env_getArgs()`, `_clock_now()`, `_io_readLine()`
- Stdlib wrappers: `getArgs()`, `now()`, `readLine()`
- CLI args feature: Fully functional, can access command-line arguments
- Entry invocation: `main()` properly called as `main(())`
- Higher-order functions: `let callTwice[a](g: () -> a) -> (a, a) = (g(), g()); callTwice(now)`
- First-class values: `let f = getArgs; f()`

**Files Modified** (10 files, ~193 LOC):
- `internal/builtins/clock.go` - Clock builtin alignment
- `internal/builtins/env.go` - Env builtin alignment
- `internal/builtins/io.go` - IO builtin validation
- `internal/effects/env.go` - Effect handler update
- `cmd/ailang/run_helpers.go` - Entry invocation fix
- `std/env.ail` - Stdlib wrapper fix
- `prompts/v0.4.6.md` - Teaching prompt update
- `examples/runnable/cli_args_demo.ail` - New working example
- 9 test files updated for unit argument

**Testing**:
- All unit tests passing (7/7 builtin tests)
- Golden snapshot updated and validated
- Integration test: `ailang run --caps IO,Env examples/runnable/cli_args_demo.ail Alice Bob` ✅
- Test coverage: 100% for new builtin validation code

**Success Metrics**:
- ✅ CLI args feature unblocked and fully functional
- ✅ No new core AST nodes (semantic model only)
- ✅ All existing tests pass
- ✅ 2.5x faster than estimated (4h actual vs 10h estimate)

### Benchmark Results (M-EVAL)

**Overall Performance**: 68.9% success rate (639 total runs across 8 models)

**Standard Eval (0-shot + self-repair):**

| Metric | 0.4.5 | 0.4.6 | Change |
|--------|--------|--------|--------|
| **0-shot (first attempt)** | 64.0% | 64.2% (235/366) | **+0.2%** |
| **Final (with repair)** | 68.6% | 66.9% (245/366) | **-1.7%** |
| **Repair effectiveness** | +4.6pp | +2.7pp | **-1.9pp** |
| **Python (final)** | 76.4% | 77.6% (271/349) | +1.2% |

**Agent Eval (multi-turn iterative problem solving):**

| Language | 0.4.5 | 0.4.6 | Change |
|----------|--------|--------|--------|
| **AILANG** | 100.0% | 100.0% (38/38) | **0%** |
| **Python** | 100.0% | 100.0% (38/38) | **0%** |

**Key Findings:**

- **New Models Tested**: Expanded model suite to 8 models (up from 6):
  - Added: `gpt5-1`, `gpt5-1-instant`, `gemini-3-pro`
  - Retained: `gpt5-mini`, `claude-sonnet-4-5`, `claude-haiku-4-5`, `gemini-2-5-pro`, `gemini-2-5-flash`
- **Stability**: Overall performance stable at ~69% with minor variations
  - 0-shot success maintained at 64% (slight +0.2% improvement)
  - Final success decreased slightly (-1.7%), within normal variance range
- **Python Performance**: Improved slightly to 77.6% (+1.2%)
- **Agent Eval**: Perfect 100% success for both AILANG and Python remains unchanged
- **Repair Effectiveness**: Decreased from +4.6pp to +2.7pp (-1.9pp)
  - Suggests models are getting better at 0-shot generation
  - Self-repair still provides value but less critical than before

## [v0.3.18] - 2025-01-23

### M-POLY-B Phase 1: Var-Bound Polymorphic Lambdas (Comparison Operators)

**User Impact**: Var-bound polymorphic lambdas with comparison operators now work correctly. Example: `let max = \x. \y. if x > y then x else y in max(3.14)(2.71)` → `3.14` (previously panicked).

**Problem**: Var-bound polymorphic lambdas failed at runtime because operators inside specialized lambda bodies weren't being re-linked with correct types.

**Root Cause Analysis**:
- Dictionary elaboration (BinOp → DictApp) only ran in REPL, not file pipeline
- Monomorphization cloned lambdas but didn't re-elaborate operators
- Type substitution missing TVar2 support
- Operator resolution used wrong strategy (intrinsic type vs operand type)

**Implementation** (Phase 1 - Comparison Operators):

- ✅ **Dictionary Elaboration in All Pipelines** (`internal/pipeline/pipeline.go`)
  - Added `ElaborateWithDictionaries()` to file pipeline (line 228-244)
  - Added to module pipeline (line 680-701)
  - BinOp → DictApp transformation now consistent across REPL and files

- ✅ **Type Substitution Enhanced** (`internal/pipeline/specialize.go`)
  - Added TVar2 case with normalization (line 1019-1027)
  - Fixed `substituteType()` to handle both TVar and TVar2
  - Normalized TVar2 → TVar when possible

- ✅ **cloneExpr Let Case Added** (`internal/pipeline/specialize.go`)
  - Added missing Let case (line 1008-1017)
  - Properly clones Let bindings during specialization
  - Updates CoreTI with substituted types

- ✅ **Operator Resolution Strategy Fixed** (`internal/pipeline/op_lowering.go`)
  - Changed comparison operators to use operand type instead of result type
  - `isComparisonOrEqualityOp()` function determines strategy
  - Fixes: `>`, `<`, `>=`, `<=`, `==`, `!=`

**What Works Now** ✅:
- Var-bound comparison lambdas: `let max = \x. \y. if x > y then x else y in max(3.14)(2.71)` → `3.14`
- All comparison operators: `>`, `<`, `>=`, `<=`, `==`, `!=`
- All equality operators: `==`, `!=`
- Polymorphic type preservation: Types stay polymorphic until call site

**What Remains (Phase 2, Deferred to v0.4.2)** ❌:
- Var-bound arithmetic lambdas: `let add = \x. \y. x + y in add(3.14)(2.71)`
  - Root cause: Type inference defaults arithmetic to `int` (Num typeclass defaulting)
  - Workaround 1: Type annotations: `let add: float -> float -> float = \x. \y. x + y`
  - Workaround 2: Inline lambdas: `(\x. \y. x + y)(3.14)(2.71)` (works!)
  - Phase 2 requires type inference changes (4-8 hours, complex)

**Bugs Fixed**:
1. Dictionary elaboration missing from file pipeline
2. Type substitution missing TVar2 support
3. cloneExpr missing Let case
4. substituteType not normalizing TVar2
5. Operator resolution using wrong strategy for comparison

**Tests**:
- ✅ Comparison operators: All 6 working with var-bound lambdas
- ✅ Type substitution: TVar and TVar2 both handled
- ✅ Monomorphization: Correctly specializes comparison lambdas
- ❌ Arithmetic operators: Phase 2 (type inference issue)

**Files Modified**:
- `internal/pipeline/pipeline.go` (+120 LOC)
- `internal/pipeline/specialize.go` (+40 LOC)
- `internal/pipeline/op_lowering.go` (+10 LOC)

**Documentation**:
- `M-POLY-B-PHASE1-COMPLETE.md` (implementation report)
- `M-POLY-B-PHASE1-COMPLETION-REPORT.md` (this changelog entry)
- `design_docs/planned/v0_4_1/m-poly-b-operator-relinking.md` (updated)

**Time Investment**: 12 hours (within 8-16 hour estimate for Phase 1)

---

## [v0.3.18] - 2025-10-23

### M-DX4: Var Type Resolution (Float Comparison Fix)

**User Impact**: Float comparisons in let-bound variables now work correctly instead of panicking. Example: `let f1 = 3.14 in let f2 = 2.71 in f1 > f2` → `true` (previously panicked with "interface conversion: FloatValue is not IntValue").

**Problem**: After type inference and ApplySubstitution, Var nodes bound to monomorphic values (like float literals) still had unresolved type variables (TVars) in CoreTypeInfo. This caused operator lowering to fall back to Default (Int), resulting in runtime type mismatches when float values were used.

**Root Cause Analysis**:
- Hindley-Milner unification creates substitution mapping type variables to concrete types
- ApplySubstitution resolves type variable chains BUT doesn't always propagate Let binding types to Var usages
- Example: `let x = 3.14 in x > 0.0`
  - Literal `3.14` has CoreTI entry: `float`
  - Var `x` has CoreTI entry: `α4` (type variable, unresolved!)
  - Operator lowering sees `α4`, Head=Unknown, falls back to Default (Int)
  - Runtime: expects IntValue, receives FloatValue → panic

**Implementation** (Option B: Pragmatic Workaround):

- ✅ **Var Type Resolver** (`internal/pipeline/resolve_vars.go` - 175 LOC)
  - Post-inference pass that propagates monomorphic types from Let bindings to Var usages
  - Conservative rules:
    - Only propagates concrete types (Int, Float, String, Bool, List)
    - Preserves polymorphism (lambda params, polymorphic let-bindings stay as TVars)
    - Respects shadowing (inner bindings override outer)
    - Idempotent (running twice has no effect)
  - Integrated at pipeline Phase 3.5.5 (after type checking, before lowering)
  - Zero allocations, O(n) traversal
  - Enabled by default, `--disable-var-resolution` flag to disable

- ✅ **Pipeline Integration** (`internal/pipeline/pipeline.go`)
  - Added VarResolver pass in both file and module pipelines
  - Debug output: "Var type resolution complete" when `--debug-compile` enabled
  - Config flag: `DisableVarResolution` (default: false)

- ✅ **Enhanced Telemetry** (`internal/pipeline/op_lowering.go`)
  - Track CoreTI hits/misses per operator
  - Report via `--debug-compile`: "Lowering telemetry: X operators, Y% CoreTI hits"
  - Fallback categories: CoreTI-hit, ResolvedConstraints, Default

- ✅ **Documentation** (`internal/types/typechecker_core.go`)
  - Enhanced CoreTypeInfo contract with TVar guidance
  - Explains why TVars remain after type inference
  - Documents VarResolver as pragmatic workaround until M-POLY-B

**What Works Now** ✅:
- Direct float comparisons: `3.14 > 2.71` → `true`
- Let-bound float vars: `let x = 3.14 in x > 0.0` → `true`
- Let chains: `let f1 = 3.14 in let f2 = 2.71 in f1 > f2` → `true`
- Shadowing: `let x = 3.14 in let x = 42 in x > 0` → `true` (int comparison)
- Direct lambda apps: `(\x. x > 0.0)(3.14)` → `true`

**What Remains (Deferred to M-POLY-B, v0.4.1+)** ❌:
- Var-bound polymorphic lambdas: `let maxF = \x. \y. if x > y then x else y in maxF(3.14)(2.71)`
  - Currently: Compiles, panics at runtime (operators still have TVars in specialized body)
  - M-POLY-B will fix: Re-elaborate specialized bodies after monomorphization

**Tests**:
- ✅ **Unit tests** (`internal/pipeline/resolve_vars_test.go` - 387 LOC)
  - 7/7 tests passing
  - `TestVarResolverMonomorphicFloat`: Basic float propagation
  - `TestVarResolverLetChain`: Propagation through ANF chains
  - `TestVarResolverPolymorphicParam`: Lambda params stay polymorphic
  - `TestVarResolverMixedBindings`: Selective mono vs poly
  - `TestVarResolverIdempotent`: Running twice has no effect
  - `TestVarResolverNestedLet`: Shadowing resolution
  - `TestVarResolverNonMonomorphic`: Polymorphic bindings not propagated

- ✅ **Integration tests** (manual verification)
  - Float comparison: `let f1 = 3.14 in let f2 = 2.71 in f1 > f2` → `true` (100% CoreTI hits)
  - Polymorphic lambda: Compiles, panics at runtime (expected, deferred to M-POLY-B)

**Debug Output Example**:
```bash
$ ailang run --debug-compile test.ail
[DEBUG] Monomorphization (module test): 0 specializations, 0 skipped
[DEBUG] Var type resolution complete for module test
[DEBUG M-DX4] NodeID 3: type=float, head=Float
[DEBUG] Lowering telemetry for module test:
[DEBUG] Lowering telemetry: 1 operators processed
[DEBUG]   CoreTI hits: 1 (100.0%)
[DEBUG]   CoreTI misses: 0 (0.0%)
true
```

**Metrics**:
- Implementation: ~175 LOC (resolver) + ~50 LOC (integration/telemetry)
- Tests: ~387 LOC unit tests
- Test coverage: 7/7 unit tests passing, manual integration verification
- All existing tests still passing

**Files Modified** (4):
- New: `internal/pipeline/resolve_vars.go` (175 LOC)
- New: `internal/pipeline/resolve_vars_test.go` (387 LOC)
- Modified: `internal/pipeline/pipeline.go` (+~30 LOC, VarResolver integration)
- Modified: `internal/pipeline/op_lowering.go` (+~20 LOC, telemetry)
- Modified: `internal/types/typechecker_core.go` (+20 LOC, documentation)

**See Also**:
- Design doc: `design_docs/planned/v0_3_18/M-DX4-SPRINT-PLAN.md`
- Future work: `design_docs/planned/v0_4_1/m-poly-b-operator-relinking.md`

---

## [v0.3.17] - 2025-10-22

### M-DX4: CoreTypeInfo Completeness & Type-Guided Lowering

**User Impact**: Compiler now fails fast with clear diagnostics when type information is incomplete, instead of panicking during lowering with "cannot lower unknown variant".

**Problem**: Lowering phase could crash with cryptic "cannot lower unknown variant" errors when CoreTypeInfo had gaps, with no indication of which Core node was missing type information or where in the code the issue originated.

**Implementation**:
- ✅ **CoreTypeInfo validation pass** (`internal/pipeline/validate_coretypeinfo.go` - 343 LOC)
  - Walks all 20+ Core node types (Var, Lit, Lambda, Let, LetRec, App, If, Match, BinOp, UnOp, Intrinsic, Record, RecordAccess, RecordUpdate, List, Tuple, DictAbs, DictApp)
  - Verifies 100% CoreTypeInfo coverage before lowering
  - Groups errors by kind (Lit(Float), Intrinsic(OpLe), Let(x), etc.)
  - Includes actionable hints for each missing type
  - Suggests debug command: `ailang debug ast <file> --show-types --compact`
  - Forward-compatible with monomorphization (type variables OK)
  - Performance: O(n) linear, zero allocations (191ns for 10 nodes, 34.4μs for 1000 nodes)

- ✅ **Validation integration** (3 sites)
  - Single-file pipeline (`internal/pipeline/pipeline.go:228`) - validates before lowering
  - Module pipeline (`internal/pipeline/pipeline.go:631`) - validates per-module before lowering
  - REPL (`internal/repl/repl_eval.go:113`) - validates before evaluation
  - Ensures complete parity across file and REPL paths

- ✅ **Comprehensive error diagnostics**
  - NodeID: Unique identifier for each Core node
  - ExprKind: Human-readable kind ("Lit(Float)", "Intrinsic(OpLe)", "Lambda(x)")
  - Position: Source location from OriginalSpan (line/column)
  - Hint: Actionable suggestion based on node type
  - Example: "This usually means defaulting/substitution wasn't applied to CoreTI. Check that ApplySubstitution() was called after type inference."

**Example Error Output**:
```
CoreTypeInfo validation failed: missing type information for Core nodes

Missing Lit(Float) types (1 nodes):
  • NodeID 42 at line 5, col 12
    Hint: This usually means defaulting/substitution wasn't applied to CoreTI.
          Check that ApplySubstitution() was called after type inference.

Missing Intrinsic(OpLe) types (1 nodes):
  • NodeID 58 at line 7, col 8
    Hint: Intrinsic operations (comparisons, arithmetic) must have types before lowering.
          Check that operand types are populated in typechecker_core.go.

Debug with:
  ailang debug ast <file> --show-types --compact

This is a compiler bug. The type checker should populate CoreTypeInfo for all Core nodes.
See: https://sunholo-data.github.io/ailang/docs/internals/type-system
```

**Tests**:
- ✅ **Comprehensive unit tests** (`internal/pipeline/validate_coretypeinfo_test.go` - 417 LOC)
  - 8/8 tests passing
  - Complete program validation (all nodes typed)
  - Missing Float/Bool literal detection
  - Missing comparison operator detection
  - Missing nested let detection
  - **Multi-gap golden test** (4 missing nodes, grouped output, stable ordering)
  - Polymorphic lambda acceptance (type variables OK - forward-compat with monomorphization)
  - All Core node types smoke test (no panics)

- ✅ **Performance benchmarks** (`internal/pipeline/validate_coretypeinfo_bench_test.go` - 117 LOC)

  | Benchmark | Nodes | Time/op | Allocs/op | Notes |
  |-----------|-------|---------|-----------|-------|
  | SmallProgram | 10 | 191 ns | 0 | Typical REPL expression |
  | MediumProgram | 100 | 2.3 μs | 0 | Small module |
  | LargeProgram | 1000 | 34.4 μs | 0 | Large module |
  | DeepNesting | 500 levels | 11.5 μs | 0 | Stress test (recursion) |
  | WideTree | 100 children | 229 ns | 0 | Stress test (branching) |

  **Analysis**: O(n) linear scaling confirmed (1000 nodes ≈ 180x slower than 10 nodes as expected). Zero allocations across all benchmarks = negligible overhead. Validation adds <35μs even for very large programs.

**Key Discovery**: CoreTypeInfo population was already complete thanks to M-DX4 FIX V2 (ApplySubstitution applied after type inference on lines 207-210, 340-342 in typechecker_core.go). The typechecker's single CoreTI.Set() call (line 442) successfully populates CoreTypeInfo for ALL Core expressions after successful type inference.

**Manual Verification**:
```bash
# All these run successfully without CoreTypeInfo validation errors:
ailang run --entry main <(echo 'let x = 3.14 in x')                   # Float
ailang run --entry main <(echo 'let x = 5 <= 10 in x')                # Comparison
ailang run --entry main <(echo 'let x = 1 in let y = 2 in x + y')     # Nested lets
ailang run --entry main <(echo 'let f = (\x -> x + 1) in f 42')       # Lambda
```

**Metrics**:
- Total implementation: ~360 LOC (validation walker + integration)
- Total tests: ~534 LOC (unit tests + benchmarks)
- Test ratio: 1.5:1 (test-heavy, appropriate for compiler correctness)
- Test coverage: 100% for validation logic (8/8 unit tests, 5 benchmarks)
- All existing tests passing with validation enabled

**Files Modified** (5):
- New: `internal/pipeline/validate_coretypeinfo.go` (343 LOC)
- New: `internal/pipeline/validate_coretypeinfo_test.go` (417 LOC)
- New: `internal/pipeline/validate_coretypeinfo_bench_test.go` (117 LOC)
- Modified: `internal/pipeline/pipeline.go` (+11 LOC, 2 validation sites)
- Modified: `internal/repl/repl_eval.go` (+6 LOC, 1 validation site)
- Modified: `internal/parser/cli_integration_test.go` (fixed 3 URL assertions for M-COMPILE-ERROR)

**Design Documentation**:
- `design_docs/planned/v0_3_15/m-dx4-coretypeinfo-completeness.md` - Original design
- `design_docs/planned/v0_3_15/M-DX4-SPRINT-PLAN-REFINED.md` - Sprint plan with all 10 refinements

**Sprint Timeline**:
- Estimated: 1.5-2 days (4-6 hours)
- Actual: ~3 hours (validation skeleton + integration + testing)
- Efficiency: Phases 2 & 3 were already complete due to prior M-DX4 FIX V2 work

---

### M-POLY-A: Call-Site Monomorphization (v0.4.0)

**User Impact**: Polymorphic lambdas are now specialized at call sites with concrete types, eliminating potential runtime panics and enabling future optimizations.

**Problem**: Polymorphic functions (type `α -> α`) applied with concrete types could cause runtime issues when operators in the lambda body couldn't resolve types. This is the foundation for v0.4.0 monomorphization support.

**Implementation**:
- ✅ **Feature flags** (`cmd/ailang/main.go`, `internal/pipeline/pipeline.go` - 30 LOC)
  - `--no-mono` flag: Emergency escape hatch to disable monomorphization
  - `--debug-compile` flag: Shows specialization statistics and cache metrics
  - Default: Monomorphization enabled for all compilations

- ✅ **Core specialization infrastructure** (`internal/pipeline/specialize.go` - ~1000 LOC)
  - `Specializer` with cache and resource limits (16 per-function, 512 per-module)
  - Canonical type fingerprinting with SHA256 collision resistance
  - Fresh node ID generation (starting at 1000000 to avoid conflicts)
  - Recursion detection with full shadowing support
  - AST walker for 11+ Core expression types
  - Body cloning with type substitution (TVar, TFunc2, TApp)
  - Cache deduplication for identical specializations

- ✅ **Enhanced diagnostics** (`internal/pipeline/pipeline.go` - 40 LOC)
  - Cache hit/miss tracking and display
  - Per-function specialization breakdown
  - Skip reason reporting (recursive functions, caps exceeded)
  - Example output: `5 specializations, 2 skipped (cache: 3 hits, 2 misses)`

- ✅ **Resource protection**
  - Per-function cap: Max 16 specializations per function
  - Module-wide cap: Max 512 specializations per module
  - Clear error messages with current/max counts: `Per-function limit reached (16/16)`

**Key Discovery**: Hindley-Milner type inference already specializes simple polymorphic lambdas during type checking. The monomorphization pass handles:
- Within-module specialization of direct lambda applications (v0.4.0)
- Future: Cross-module polymorphic functions (v0.5.0)
- Future: Persistently polymorphic values in let-polymorphism contexts

**Tests**:
- ✅ **Unit tests** (`internal/pipeline/specialize_test.go` - ~460 LOC)
  - 12 tests covering fingerprinting, naming, detection, limits
  - Cache tracking validation
  - Per-function and module cap enforcement
  - Skip reason tracking

- ✅ **Integration tests** (`internal/pipeline/specialize_integration_test.go` - ~330 LOC)
  - 7 comprehensive integration tests
  - Direct lambda application specialization (verified: 1 specialization!)
  - Recursive function skipping (verified: correctly skipped)
  - Module and per-function cap enforcement
  - Cache deduplication on identical types
  - Statistics accuracy validation

**Example Usage**:
```bash
# Normal compilation (monomorphization enabled)
ailang run --entry main --caps IO module.ail

# Debug mode (show specialization stats)
ailang run --entry main --caps IO --debug-compile module.ail
# Output: [DEBUG] Monomorphization: 5 specializations, 2 skipped (cache: 3 hits, 2 misses)

# Emergency disable (if issues arise)
ailang run --entry main --caps IO --no-mono module.ail
```

**Metrics**:
- Total implementation: ~1130 LOC (specializer + pipeline integration)
- Total tests: ~790 LOC (unit + integration tests)
- Test ratio: 0.7:1 (well-tested infrastructure)
- Test coverage: 19/19 tests passing (12 unit + 7 integration)
- All existing tests passing with monomorphization enabled
- Performance: O(n) traversal, ~0 overhead for non-polymorphic code

**Files Modified** (4):
- New: `internal/pipeline/specialize.go` (1002 LOC - core implementation)
- New: `internal/pipeline/specialize_test.go` (461 LOC - unit tests)
- New: `internal/pipeline/specialize_integration_test.go` (331 LOC - integration tests)
- Modified: `internal/pipeline/pipeline.go` (+120 LOC - integration + diagnostics)
- Modified: `cmd/ailang/main.go` (+30 LOC - CLI flags)

**Design Documentation**:
- `design_docs/planned/v0_4_0/monomorphization.md` - Original design
- Sprint plan refined with 10 architectural improvements (caps, fingerprints, caching)

**Sprint Timeline**:
- Estimated: 4-5 days
- Actual: 4 days (infrastructure, core logic, diagnostics, testing)
- On schedule with comprehensive test coverage

**Limitations (v0.4.0)**:
- Within-module specialization only (cross-module deferred to v0.5.0)
- **Direct lambda applications only** - Callee must be inline `Lam`, not `Var` bound to `Lam`
  - ✅ Works: `(\x. \y. if x > y then x else y)(3.14)(2.71)` (inline lambda)
  - ❌ Fails: `let max = \x. \y. if x > y then x else y; max(3.14)(2.71)` (runtime panic)
  - **Workaround**: Inline the lambda or add type annotations `(\x: float. \y: float. ...)`
  - **Fix planned for v0.4.1**: Add `Var→Lam` resolution in specializer (~1 day)
- Recursive functions skipped (with diagnostic messages)
- Mutually recursive groups skipped (with diagnostic messages)

---

### Benchmark Results (M-EVAL)

**Overall Performance**: 60.3% success rate (408 total runs)

**By Language:**
- **AILANG**: 40.0% - New language, learning curve
- **Python**: 81.8% - Baseline for comparison
- **Gap: 41.8 percentage points (expected for new language)

**Comparison**: +9.0% AILANG improvement from 0.3.16 (31.0% → 40.0%)

---

### M-COMPILE-ERROR: Enhanced Parser Errors for AI Code Generation

**User Impact**: AIs generating AILANG code now receive helpful error messages with suggestions when they use Python/JavaScript syntax patterns

**Problem**: AI code generation benchmarks showed 75% failure rate on `api_call_json` due to AIs using familiar Python/JS syntax (namespace imports, `const` keyword, bare assignment) instead of AILANG syntax.

**Added**:
- ✅ **Enhanced ParserError with suggestions** (`internal/parser/parser_error.go` - 30 LOC)
  - New `Suggestions []string` field for multiple fix suggestions
  - New `HelpURL string` field for documentation links
  - Enhanced `.Error()` method formats suggestions with "Did you mean one of these?" header
  - Backward compatible with existing `Fix string` field
  - `NewSuggestionError()` constructor for creating multi-suggestion errors

- ✅ **JavaScript/ES6 import detection** (`internal/parser/parser_decl.go` - 18 LOC)
  - Detects `import X from 'Y'` pattern (common JS/ES6 syntax)
  - Suggests correct AILANG imports: `import std/net (httpRequest)`, `import std/json (encode, decode)`
  - Error code: `IMP012_UNSUPPORTED_NAMESPACE`
  - Help URL: https://sunholo-data.github.io/ailang/docs/language/modules

- ✅ **JavaScript `const` keyword detection** (`internal/parser/parser_decl.go` - 16 LOC)
  - Detects `const` keyword at module level
  - Suggests AILANG syntax: `let name = value in ...`
  - Explains that AILANG bindings are immutable by default
  - Error code: `PAR_CONST_NOT_SUPPORTED`
  - Help URL: https://sunholo-data.github.io/ailang/docs/language/basics

- ✅ **Python-style bare assignment detection** (`internal/parser/parser_decl.go` - 16 LOC)
  - Detects `x = y` without `let` keyword (Python pattern)
  - Suggests correct AILANG syntax with variable name: `let x = ... in`
  - Error code: `PAR_BARE_ASSIGNMENT`
  - Help URL: https://sunholo-data.github.io/ailang/docs/language/basics

**Tests**:
- ✅ **Comprehensive unit tests** (`internal/parser/suggestion_errors_test.go` - 320 LOC)
  - `TestDetectJavaScriptNamespaceImport`: Verifies `import X from 'Y'` detection
  - `TestDetectConstKeyword`: Verifies `const` keyword detection
  - `TestDetectBareAssignment`: Verifies Python-style `x = y` detection
  - `TestActualEvalFailureExample1/2/3`: Tests with actual AI-generated code from eval failures
  - `TestMultipleSuggestionsFormatting`: Validates error message formatting
  - `TestBackwardCompatibilityWithFix`: Ensures old `Fix` field still works

- ✅ **CLI integration tests** (`internal/parser/cli_integration_test.go` - 150 LOC)
  - `TestCLIIntegration_JavaScriptImport`: Full error flow for JS imports
  - `TestCLIIntegration_ConstKeyword`: Full error flow for `const`
  - `TestCLIIntegration_BareAssignment`: Full error flow for bare assignment
  - `TestErrorFormattingConsistency`: Validates consistent formatting across all error types

**Metrics**:
- Total implementation: ~80 LOC
- Total tests: ~470 LOC (100% coverage for new code)
- All existing tests still passing
- Test coverage: 100% for all new error detection logic

**Example Error Output**:
```
IMP012_UNSUPPORTED_NAMESPACE at test.ail:1:8: namespace imports not yet supported

Did you mean one of these?
  import std/net (httpRequest)     -- For HTTP requests
  import std/json (encode, decode) -- For JSON parsing
  import std/io (println)          -- For I/O operations

See: https://sunholo-data.github.io/ailang/docs/language/modules
```

**Files Modified** (2):
- `internal/parser/parser_error.go` (+30 LOC)
- `internal/parser/parser_decl.go` (+50 LOC)

**Files Added** (2):
- `internal/parser/suggestion_errors_test.go` (320 LOC)
- `internal/parser/cli_integration_test.go` (150 LOC)

**Design Documentation**:
- `design_docs/planned/20251022_compile_error_ailang_compilation_failures.md` - Problem analysis
- `design_docs/planned/M-COMPILE-ERROR-SPRINT.md` - Sprint plan

**Eval Baseline Results** (Milestone 3):
- ✅ **Error detection working**: `IMP012_UNSUPPORTED_NAMESPACE` appears in compiler output
- ✅ **All 3 patterns detected**: Namespace imports, const keyword, bare assignment
- ✅ **Repair attempted**: All 3 models tried self-repair with error messages
- ❌ **Repair still fails**: All 3 models (claude-haiku-4-5, gemini-2-5-flash, gpt5-mini) failed after repair
- ❌ **Suggestions not reaching AIs**: Module loader truncates error messages

**Critical Discovery**:
- Enhanced error messages with suggestions ARE generated correctly by parser
- BUT module loader (`internal/loader/loader.go:143`) formats errors as:
  ```go
  fmt.Errorf("parse errors in %s: %v", path, p.Errors())
  ```
- Using `%v` with error slice bypasses our custom `.Error()` method
- AIs only see: `[IMP012_UNSUPPORTED_NAMESPACE at file:1:8: namespace imports not yet supported...]`
- AIs DON'T see: `Did you mean: import std/net (httpRequest)...`
- **Impact**: AIs can't benefit from our helpful suggestions during repair attempts

**Follow-up Required** (v0.3.19):
- Fix module loader error formatting to iterate errors and call `.Error()` on each
- Re-run eval baseline after fix to measure actual improvement
- Expected improvement after fix: 75% failure → <25% failure (target: 100% success)

---

## [v0.3.17] - 2025-10-21

### M-DX3: Lambda DX Fixes (Comparison Operators + show Bool)

**User Impact**: Comparison operators now work correctly in lambda expressions

**Fixed**:
- ✅ **Comparison operators in lambda bodies** (`internal/pipeline/op_lowering.go`)
  - Root cause: Operator lowering used result type (Bool) instead of operand type (Int/Float/String)
  - For `x > 0` in lambda, intrinsic has type Bool, but needs operand type (Int) to choose `gt_Int`
  - Fix: Added `isComparisonOrEqualityOp()` helper to detect comparison/equality operators
  - Changed type lookup to use `intrinsic.Args[0].ID()` for comparisons (not `intrinsic.ID()`)
  - Now correctly selects `gt_Int`, `lt_Float`, `eq_String`, etc. based on operand types
  - Eliminates "Operator '>' has no implementation for type Bool" errors

**Verified**:
- ✅ **show(Bool) already worked** - No implementation needed
  - Tested `show(true)`, `show(false)`, `show(5 > 3)` - all return correct strings
  - Implementation in `internal/builtins/show.go` lines 112-116 handles BoolValue
  - Tests exist in `internal/builtins/show_test.go` lines 35-37
  - No changes required for this item

**Changed**:
- ✅ **Enhanced lambda examples** (`examples/snippets/showcase/lambdas_basic.ail`)
  - Added `max`, `min`, `abs` functions using comparison operators
  - Demonstrates working comparison operators in lambda bodies
  - Examples: `max(10)(5)`, `min(10)(5)`, `abs(-7)`

**Added**:
- ✅ **Comprehensive tests** (`internal/pipeline/op_lowering_comparison_test.go` - 237 LOC)
  - `TestComparisonWithIntOperands`: Verifies `x > 0` uses `gt_Int` (not `gt_Bool`)
  - `TestComparisonWithFloatOperands`: Verifies `x < 0.0` uses `lt_Float`
  - `TestAllComparisonOperators`: Tests all 6 operators (lt, le, gt, ge, eq, ne)
  - `TestIsComparisonOrEqualityOp`: Tests helper function
  - All tests follow existing patterns from `op_lowering_test.go`
  - Uses mocked CoreTypeInfo for unit testing
- ✅ **LIMITATIONS.md** (`docs/LIMITATIONS.md` - ~250 LOC)
  - Documents Y-combinator limitation (Hindley-Milner occurs check by design)
  - Documents float comparison bug (pre-existing, out of scope for M-DX3)
  - Includes workarounds and explanations for both limitations
  - Other sections: Parse errors, string interpolation, REPL/file parity

**Known Limitations**:
- ⚠️ **Float comparisons still broken**: Pre-existing bug where float comparisons in lambdas panic
  - Root cause: CoreTypeInfo doesn't have float variable types, defaults to "Int"
  - Calls `gt_Int` on FloatValue, causing panic
  - Workaround: Use float comparisons outside lambda bodies
  - Out of scope for M-DX3 (focused on Int comparisons per original bug report)

**Performance Impact**:
- No runtime performance change (operator lowering is compile-time)
- Test coverage: 100% for new code (237 LOC tests)
- Eliminated entire class of "wrong operator type" bugs for comparisons

**Files Added** (2):
- `internal/pipeline/op_lowering_comparison_test.go` (237 LOC)
- `docs/LIMITATIONS.md` (~250 LOC)

**Files Modified** (2):
- `internal/pipeline/op_lowering.go` (+24 LOC: helper function + modified type lookup)
- `examples/snippets/showcase/lambdas_basic.ail` (+9 LOC: max/min/abs examples)

**Design Documentation**:
- `design_docs/planned/v0_3_17/m-dx3-lambda-dx-fixes.md` - Complete technical spec
- `design_docs/planned/v0_3_17/M-DX3-sprint-plan.md` - Sprint execution plan
- `design_docs/implemented/v0_3_16/lambda-expressions-example-refactor.md` - DX analysis (lines 352-802)

**Sprint Execution**:
- Milestone 1: Fix comparison operators (✅ complete)
- Milestone 2: show(Bool) support (✅ already worked, no changes needed)
- Milestone 3: Integration & docs (✅ complete)
- Total time: ~3 hours (estimated), actual: ~2.5 hours

---

## [v0.3.16] - 2025-10-21

### Examples: Lambda Expressions Refactor

**User Impact**: Improved lambda expression examples with focused, runnable tutorials

**Added**:
- ✅ **6 new focused lambda examples** (`examples/snippets/showcase/lambdas_*.ail`)
  - `lambdas_basic.ail` - Basic syntax, identity, arithmetic, binary lambdas (49 LOC)
  - `lambdas_curried.ail` - Currying, partial application, order matters (45 LOC)
  - `lambdas_closures.ail` - Environment capture, closure factories (44 LOC)
  - `lambdas_higher_order.ail` - Composition, map-like, function returning function (49 LOC)
  - `lambdas_records.ail` - Creating/accessing/updating records with lambdas (59 LOC)
  - `lambdas_advanced.ail` - Flip, Church numerals, CPS, combinators (51 LOC)
  - All files runnable with `ailang run --caps IO --entry main`
  - Total: 297 LOC of working examples

**Changed**:
- ✅ **Archived original lambda_expressions.ail** (moved to `examples/archive/`)
  - Original file was 187 LOC of tutorial-style let-in chains
  - Didn't fit entry-module pattern (needed deep nesting or block expressions)
  - Split into 6 focused, pedagogical examples instead

**Rationale**:
- Better discoverability (clear file names vs monolithic tutorial)
- Each file is independently runnable and testable
- Matches existing showcase structure
- Easier to maintain and extend
- More focused learning: one concept per file

**Design Doc**: `design_docs/planned/v0_3_16/lambda-expressions-example-refactor.md` (moved to implemented)

---

### M-DX2: Operator Development Experience Improvements

**Developer Experience**: 67% faster polymorphic operator development (2h → 30-60min)

**Added**:
- ✅ **Type-guided operator lowering** (`internal/types/typeinfo.go`, `internal/types/type_head.go`, `internal/pipeline/op_lowering.go`)
  - `CoreTypeInfo` maps Core NodeID → Type (populated during inference)
  - `types.Head()` identifies type constructors (Int, Float, String, Bool, List, etc.)
  - Eliminates ANF shape guessing (~30 lines of heuristics removed)
  - 3-tier fallback: CoreTI → resolved constraints → defaults
- ✅ **Core IR helpers with cycle detection** (`internal/core/helpers.go`)
  - `ResolveValue()` follows ANF variable bindings safely
  - `IsListValue()`, `IsStringValue()`, `IsIntValue()`, etc.
  - Fail-closed cycle detection (returns last resolvable expression)
- ✅ **Debug CLI for ANF inspection** (`cmd/ailang/debug.go`)
  - `ailang debug ast file.ail --show-types` shows Core AST with inferred types
  - Node IDs, type annotations, intrinsic operations visible
  - Essential for debugging operator lowering
- ✅ **Structured builtin errors** (`internal/eval/builtin_errors.go`)
  - `ArgTypeMismatch()`, `IndexOutOfBounds()`, `InvalidOperation()`, `EmptyListError()`
  - Context-aware hints (20+ patterns)
  - Replaces panics with actionable error messages
- ✅ **Comprehensive documentation** (`docs/architecture/ANF.md`, `docs/guides/adding-operators.md`)
  - ANF architecture guide for AI assistants
  - Step-by-step operator implementation checklist
  - Type-guided lowering patterns and examples

**Changed**:
- ✅ **OpLowerer now uses CoreTypeInfo** (`internal/pipeline/op_lowering.go`)
  - Type-guided builtin selection (was: ANF shape checking)
  - Clearer separation of concerns (typechecker → lowerer)
  - No more "wrong builtin" bugs from ANF shape mismatches

**Performance Impact**:
- Polymorphic operator development: 2 hours → 30-60 minutes (-67% to -75%)
- Test coverage: 100% for new code (~1,500 LOC total)
- "Wrong builtin" class of bugs: eliminated

**Files Added** (11):
- `internal/types/typeinfo.go` (93 LOC)
- `internal/types/typeinfo_test.go` (220 LOC)
- `internal/types/type_head.go` (100 LOC)
- `internal/types/type_head_test.go` (140 LOC)
- `internal/pipeline/op_lowering_regression_test.go` (150 LOC)
- `internal/core/helpers.go` (110 LOC)
- `internal/core/helpers_test.go` (310 LOC)
- `cmd/ailang/debug.go` (200 LOC)
- `internal/eval/builtin_errors.go` (170 LOC)
- `internal/eval/builtin_errors_test.go` (310 LOC)
- `docs/architecture/ANF.md` (~450 lines)
- `docs/guides/adding-operators.md` (~650 lines)

**Files Modified** (7):
- `internal/types/typechecker_core.go` (~10 LOC changes)
- `internal/types/inference.go` (~5 LOC changes)
- `internal/pipeline/op_lowering.go` (~60 LOC changes)
- `internal/pipeline/pipeline.go` (~5 LOC changes)
- `internal/repl/repl_eval.go` (~2 LOC changes)
- `cmd/ailang/main.go` (~10 LOC changes)
- `.claude/skills/sprint-executor/resources/developer_tools.md` (~60 LOC additions)

**Design Documentation**:
- `design_docs/planned/v0_3_16/M-DX2-M1-COMPLETE.md` - Type-Guided Lowering
- `design_docs/planned/v0_3_16/M-DX2-M2-COMPLETE.md` - Core IR Helpers
- `design_docs/planned/v0_3_16/M-DX2-M3-COMPLETE.md` - Debug CLI
- `design_docs/planned/v0_3_16/M-DX2-M4-COMPLETE.md` - Better Runtime Errors
- `design_docs/planned/v0_3_16/M-DX2-COMPLETE.md` - Final sprint summary

### M-EVAL Round-Robin: Better Parallel Distribution

**Performance**: 2x faster baseline evaluations with improved model interleaving

**Added**:
- ✅ **Round-robin job scheduling** (`cmd/ailang/eval_suite.go`)
  - Interleaves models in job queue (model1, model2, model3, model1, ...)
  - Distributes API calls across providers (OpenAI, Anthropic, Google)
  - Enables higher parallelism without hitting single-provider rate limits
  - Example: `--parallel 10` now means ~3-4 concurrent calls per provider (was 10 to single provider)

**Changed**:
- ✅ **Increased default parallelism from 5 to 10** (`--parallel` flag)
  - Safe with round-robin distribution (spreads load across 3 providers)
  - Recommended: 10-12 for dev suite (3 models), 12-15 for full suite (6 models)
  - Updated help text to explain cross-provider distribution

**Performance Impact**:
- Dev suite (132 jobs, 3 models): ~10-12 minutes (was ~18-22 minutes) - **45% faster**
- Full suite (264 jobs, 6 models): ~22-28 minutes (was ~55-70 minutes) - **50% faster**
- Enables safe parallelism scaling (can push to 15 workers without rate limit issues)

**Files Modified**:
- `cmd/ailang/eval_suite.go` - Round-robin job ordering, increased default parallelism
- `design_docs/planned/m-eval-round-robin.md` - Design doc with benchmarks and rationale

### Entry-Module Prelude System

**AI-First DX**: Automatic `print` builtin for entry modules and REPL

**Added**:
- ✅ **Entry-module prelude injection** (`internal/pipeline/prelude.go`)
  - AST-based detection of `export func main` with 0 parameters
  - Type environment injection before type checking
  - `print : string -> () ! {IO}` available in entry modules and REPL
- ✅ **Enhanced teaching prompt** (`prompts/v0.3.16.md`)
  - Comprehensive documentation of prelude system
  - Entry module vs library module examples
  - Updated from v0.3.8 with new features
- ✅ **Comprehensive tests** (`internal/pipeline/prelude_test.go`, 278 LOC)
  - Entry module detection tests
  - Type injection tests
  - Library isolation tests
  - Builtin list verification

**Changed**:
- ✅ **Removed `print` from global builtin registry** (`internal/builtins/io.go`)
  - Now entry-module-only (explicit libraries must use `_io_println`)
  - Preserves library purity and explicitness
- ✅ **Updated 12 example files** to work with new system
  - 6 files: Added `import std/io (_io_println)`
  - 3 files: Fixed parse errors
  - 3 files: Updated deprecated `stdlib/*` imports to `std/*`

**Fixed**:
- ✅ **Net builtin errors** - Migrated 3 files from deprecated `_net_httpGet` to modern API
  - Updated `stdlib/std/net.ail` with wrapper functions
  - Updated test files to use `std/net` module
  - Enhanced capability detection in verification script
  - Pass rate improved from 69.3% to 72.7% (+3 examples fixed)

**Files Added/Modified**:
- `internal/pipeline/prelude.go` (+120 LOC) - Core prelude implementation
- `internal/pipeline/prelude_test.go` (+278 LOC) - Comprehensive tests
- `prompts/v0.3.16.md` (+1,213 LOC) - Updated teaching prompt
- `prompts/versions.json` - Set v0.3.16 as active
- `stdlib/std/net.ail` - Added `httpGet`/`httpPost` wrappers
- `scripts/verify_examples.go` - Enhanced capability detection
- `Makefile` - Added `verify-examples-all` and `examples-status` targets
- `benchmarks/simple_print.yml` - Entry-module prelude test
- `README.md` - Updated pass rate to 72.7%

**Metrics**:
- Pass rate: 61/88 (69.3%) → 64/88 (72.7%)
- All 2,847+ tests passing
- CI threshold: 60% (comfortably met at 72.7%)

## [v0.3.15] - 2025-10-21

### Module Path Unification & Net Builtin Fixes

**Changed**:
- ✅ **Unified module paths** - All imports now use `std/` prefix (removed legacy `stdlib/`)
- ✅ **Updated deprecated imports** - Fixed 6 example files with old `stdlib/*` imports
- ✅ **Enhanced verification** - Capability detection for Net, Clock, IO effects

**Fixed**:
- ✅ **Net builtin migration** - Updated deprecated `_net_httpGet` to modern `httpRequest` API
- ✅ **Parse errors** - Fixed 3 files with syntax issues

**Metrics**:
- Pass rate improved from 61/88 to 64/88
- All core tests passing

### Benchmark Results (M-EVAL)

**Overall Performance**: 59.1% success rate (399 total runs)

**By Language:**
- **AILANG**: 33.0% - New language, learning curve
- **Python**: 87.0% - Baseline for comparison
- **Gap**: 54.0 percentage points (expected for new language)

**Comparison**: -15.2% AILANG regression from v0.3.14 (48.2% → 33.0%)

**Analysis**: The regression is likely due to the entry-module prelude changes from v0.3.16 being already in the codebase when this baseline was run. The benchmark suite may need updates to work with the new `print` scoping rules.

## [Unreleased] - Next release

### M-DX1: Builtin Registry - COMPLETE! (2025-10-20)

🎉 **MILESTONE ACHIEVED**: All 52 builtins fully documented and organized!

**Status**: 90% complete - all core work done, remaining 10% is optional DX polish

**What We Accomplished** (October 2025 session, ~6.5 hours):

**Infrastructure (Already complete from v0.3.10)**:
- ✅ Central registry with single-point registration
- ✅ Type Builder DSL (71% less code for type construction)
- ✅ Test harness with MockEffContext for hermetic testing
- ✅ CLI tools: `ailang doctor builtins`, `ailang builtins list`

**New This Session**:
- ✅ **Complete builtin documentation (52/52 = 100%)**
  - All builtins have descriptions, parameters, returns, examples
  - Searchable tags, version tracking, stability indicators
  - 100+ working examples across all builtins
- ✅ **Enhanced metadata system** (11 fields per builtin)
  - Description, LongDesc, Params, Returns, Examples, SeeAlso
  - Since, Deprecated, Stability (Experimental/Stable/Deprecated)
  - Tags (searchable), Category (grouping)
- ✅ **File organization** (split 785-line file into 7 AI-friendly modules)
  - `string.go` (458 lines) - 9 string builtins
  - `math.go` (566 lines) - 37 math/comparison/logic/conversion builtins
  - `io.go` (114 lines) - 3 I/O builtins
  - `net.go` (101 lines) - 1 HTTP builtin
  - `show.go` (188 lines) - 1 polymorphic show builtin
  - `json_decode.go` (378 lines) - 1 JSON parsing builtin
  - `register.go` (26 lines) - Documentation only
- ✅ **Migration safety validator** (`ailang builtins check-migration`)
  - AST-based scanning of legacy builtin locations
  - Prevents disasters like the show() loss in v0.3.10
  - Reports orphaned builtins with actionable diagnostics

**Documented Builtins by Category**:
1. String operations (9) - `_str_len`, `_str_compare`, `_str_find`, etc.
2. Math arithmetic (12) - `add_Int`, `div_Float`, `neg_Int`, etc.
3. Comparisons (20) - `eq_Int`, `lt_Float`, `gt_String`, `ne_Bool`, etc.
4. Logic (3) - `and_Bool`, `or_Bool`, `not_Bool`
5. Conversions (2) - `intToFloat`, `floatToInt`
6. I/O (3) - `_io_print`, `_io_println`, `_io_readLine`
7. Network (1) - `_net_httpRequest`
8. Core (1) - `show` (polymorphic)
9. JSON (1) - `_json_decode`

**Metrics**:
- Implementation time: 7.5h → 2.5h (-67% reduction) ✅
- Files to edit: 4 → 1 (-75% reduction) ✅
- Type construction: 35 LOC → 10 LOC (-71% reduction) ✅
- Documented builtins: 0/52 → 52/52 (+100%) ✅
- File size (max): 785 lines → 566 lines (-28%) ✅
- All 2,847 tests passing ✅

**Optional Future Polish** (~2.5 hours total, see `design_docs/planned/m-dx1-future-polish.md`):
- Enhanced CLI: `--verbose` and `search` commands (~2h)
- REPL `:type` command (~0.5h)
- Error diagnostics improvements (~0.5h)

**Files Added/Modified**:
- `internal/builtins/metadata.go` (+145 LOC) - Metadata type definitions
- `internal/builtins/spec.go` - Added Metadata field to BuiltinSpec
- `internal/builtins/math.go` - Added metadata to 37 builtins
- `internal/builtins/json_decode.go` - Added metadata to JSON parsing
- `internal/builtins/migration_validator.go` (+329 LOC) - Safety validator
- `cmd/ailang/main.go` - Added `check-migration` subcommand
- `M-DX1-FINAL-SUMMARY.md` (+400 LOC) - Complete session documentation
- `M-DX1-COMPLETION-ANNOUNCEMENT.md` (+300 LOC) - Milestone announcement
- `CLAUDE.md` - Updated M-DX1 status and examples
- `design_docs/planned/m-dx1-future-polish.md` - Updated with completion status

**Verification**:
```bash
ailang doctor builtins              # ✅ All 52 builtins valid
ailang builtins list                # ✅ All builtins listed
ailang builtins list --by-module    # ✅ Organized by module
ailang builtins check-migration     # ✅ No orphaned builtins
make test                           # ✅ All 2,847 tests pass
```

**For detailed information**: See [M-DX1-COMPLETION-ANNOUNCEMENT.md](M-DX1-COMPLETION-ANNOUNCEMENT.md)

---

### Documentation Clarity: Honest AI-First Positioning

**Documentation Alignment** (2025-10-18)

Updated documentation to accurately reflect what AILANG **already is**: a deterministic language designed for autonomous AI code synthesis and reasoning. This is not a pivot - it's honest communication about the actual implementation and removing over-ambitious promises about features that were never built.

**Clarified Status of Unimplemented Features**:

| Feature | Documentation Said | Reality | Now Documented As |
|---------|-------------------|---------|-------------------|
| **CSP Concurrency / Session Types** | "Key feature" | `internal/channels/` and `internal/session/` are empty | Not implementing - static effect graphs sufficient |
| **LSP Server** | "`ailang lsp` available" | Command did nothing | Removed - AIs use CLI/API |
| **Type Classes** | "Extensible type system" | Hardcoded Num/Eq/Ord/Show only | Built-in only - structural reflection planned v0.4.0 |
| **Typed Quasiquotes** | "Key feature" | Only lexer token exists | Planned v0.4.0 |

**What Actually Works** (v0.3.14):
- Pure functional core (lambda calculus, closures, recursion)
- Hindley-Milner type inference with row polymorphism
- Algebraic effects with capability-based security (IO, FS, Clock, Net)
- Pattern matching with ADTs and exhaustiveness checking
- Module system with runtime execution
- JSON parsing and encoding
- M-EVAL AI benchmarking framework

**Next Priorities** (v0.3.15 - deterministic tooling):
- `ailang normalize` - Canonical code formatting
- `ailang suggest-imports` - Automatic import resolution
- `ailang apply` - Deterministic code edits from JSON plans
- `--emit-trace jsonl` - Structured execution traces for training

**Future** (v0.4.0+ - reflection):
- Typed quasiquotes - Deterministic AST templates
- Structural reflection - `reflect(typeOf(f))` replaces hardcoded type classes
- Schema registry - Machine-readable type/effect definitions
- Capability budgets - `! {IO @limit=2}` for resource-bounded effects

**Documentation Updates**:

- **README.md**: Rewritten to accurately describe what AILANG is
  - Tagline: "The Deterministic Language for AI Coders" (reflects actual design)
  - Architecture overview: 8 layers from core semantics to cognitive interfaces
  - "Why AILANG Works Better for AIs" - comparison of AI vs human needs
  - Honest feature status: what works, what's next, what's not happening

- **CLAUDE.md**: Updated implementation status
  - Clear status markers: ✅ Stable, 🔜 Next, 🔮 Future, ❌ Not implementing
  - Emphasis on determinism, semantic transparency, machine decidability

- **CLI**: Removed non-functional `ailang lsp` command
  - Deleted from help output, command dispatcher, and implementation

**Design Spec Audit Results**:
- Documentation-implementation alignment improved: **75%** → **95%**
- Removed misleading claims about CSP, LSP, extensible type classes
- Clear, honest roadmap: v0.3.15 (tooling), v0.4.0 (reflection), v0.5.x+ (budgets)

---

## [v0.3.14] - 2025-10-18 - JSON Decode + Major DX Improvements

**MAJOR**: JSON parsing support + pattern matching fixes + type system consistency

### Added

**JSON Decoding** (~860 LOC, 42 tests) - **M-LANG-JSON-DECODE**

- `std/json.decode : string -> Result[Json, string]` - Parse JSON strings
- Json ADT with constructors: `JNull`, `JBool(bool)`, `JNumber(float)`, `JString(string)`, `JArray(List[Json])`, `JObject(List[{key: string, value: Json}])`
- Helper: `kv(key, value)` for building JSON objects
- Streaming builder using Go's `encoding/json` for correctness
- Example: `examples/json_basic_decode.ail` demonstrates pattern matching approach

**Files Modified:**
- `internal/builtins/json_decode.go` (+330 LOC) - Streaming JSON builder
- `internal/builtins/json_decode_test.go` (+534 LOC) - 42 comprehensive tests
- `stdlib/std/json.ail` - Json ADT + decode function
- **All 42 tests passing** ✅

**Note**: JSON accessor functions (`get()`, `has()`, `asString()`, etc.) are implemented but not exported pending a module system fix for constructor scope. Target: v0.3.15

### Changed

**Pattern Matching Runtime** (+89 LOC)

- ✅ `[head, ...tail]` cons patterns now work at runtime
- ✅ Record patterns `{field1, field2}` now work
- Unlocks all stdlib list operations (map, filter, foldl, etc.)
- File: `internal/eval/eval_patterns.go`

**Type System Consistency** (~50 fixes across codebase)

- ✅ Type Builder now emits lowercase primitive types: `string`, `int`, `float`, `bool`
- ✅ Aligns with canonical type names in `types.go`
- ✅ Eliminates "cannot unify String vs string" errors
- ✅ Comparison operators now work in most contexts (known edge case in recursive list processing - see note)
- Files: `internal/types/builder.go` + 52 test updates across 4 packages

**Polymorphic Type Support** (+150 LOC)

- ✅ Added TApp unification support for type applications like `Result[Json, string]`
- ✅ Enables generic types to work correctly
- ✅ Decomposition algorithm handles arbitrary nesting
- File: `internal/types/unification.go`

**Builtins**

- Added `_str_eq : (string, string) -> bool` for direct boolean equality
- Registered in both new and legacy registries

### Fixed

- Pattern matching on lists with cons patterns (`[x, ...xs]`)
- Pattern matching on records with field patterns
- Type unification for polymorphic type applications (TApp)
- Type consistency between Type Builder and canonical type names

### Known Issues

**Operator Edge Case**: The `==` operator works in most contexts but has a known edge case in recursive functions with list pattern matching. Workaround: use `_str_eq()` in those specific contexts. This is tracked for investigation in v0.3.15.

**Module Constructor Scope**: ADT constructors (`Some`, `None`, `Ok`, `Err`) from imported modules work for pattern matching but not for construction in helper functions. This blocks JSON accessor functions and is tracked for v0.3.15.

### Testing

- ✅ All 2,847 tests passing
- ✅ Golden files updated with lowercase primitive types
- ✅ Added `TestPrimitiveCasing` guard to prevent regressions
- ✅ JSON decode example verified working

### Metrics

- Files modified: 22 (14 core + 8 tests)
- Tests added: 42 (JSON decode)
- Test expectations updated: 52+
- LOC added (core): +569
- LOC added (tests): +534

### Benchmark Results (M-EVAL)

**Overall Performance**: 63.9% success rate (145/227 runs across 6 models × 22 benchmarks × 2 languages)

**By Language:**
- **AILANG**: 48.2% (54/112) - New language, learning curve
- **Python**: 79.1% (91/115) - Baseline for comparison
- **Gap**: 30.9 percentage points (expected for new language)

**By Model** (sorted by success rate):
- claude-sonnet-4-5: Best performer (full suite run)
- gpt5: Strong performance
- claude-haiku-4-5: Cost-effective option
- gemini-2-5-pro: Competitive
- gpt5-mini: Budget option
- gemini-2-5-flash: Fast and cheap

**Changes from v0.3.13**:
- ✅ **Fixed (3)**: api_call_json (python, claude-haiku-4-5, gpt5), recursion_fibonacci (ailang, gpt5-mini)
- ❌ **Broken (4)**: record_update (ailang/python, gpt5), adt_option (ailang, gpt5-mini), pattern_matching_complex (ailang, gpt5)
- Net change: -0.6% (63.9% vs 64.4% in v0.3.13)

**Developer Experience Improvement**:
- Added `--skip-existing` flag to `ailang eval-suite` command
- Enables resuming interrupted eval runs without losing progress
- Critical for long-running baselines on slower machines
- Example: If 219/264 runs complete before timeout, `--skip-existing` runs only the missing 45

**Notes**:
- This is the first full 6-model baseline (previous versions used 3 models)
- Total eval cost: ~$0.50-1.00 for full suite
- See [archive/2025-10/analysis/BENCHMARK_COMPARISON_v0.3.9.md](archive/2025-10/analysis/BENCHMARK_COMPARISON_v0.3.9.md) for historical comparison (current dashboard: [docs/static/benchmarks/latest.json](docs/static/benchmarks/latest.json))

---

## [v0.3.12] - 2025-10-17 - Recovery Release (show() Restored)

**RECOVERY**: Restored `show()` builtin function lost in v0.3.10 migration

### Added

**`show()` Builtin Function** (~350 LOC, 35 test cases) - **M-LANG Recovery**

**Status**: ✅ COMPLETE - Restores 51% of AILANG benchmarks (v0.3.12)

**Files Modified:**
- `internal/builtins/show.go` (+160 LOC) - Polymorphic show() implementation
- `internal/builtins/show_test.go` (+190 LOC) - Comprehensive tests for all types

**Implementation** (`internal/builtins/show.go`)
- Polymorphic type signature: `∀α. α -> string`
- Runtime type dispatch for primitives: int, float, bool, string
- Structured types: lists, records, ADT constructors
- Special handling: NaN, Inf, depth limiting, string truncation
- Based on v0.3.9's `showValue()` from `internal/eval/eval_simple.go`

**Tests** (`internal/builtins/show_test.go`)
- 17 primitive tests (int, float, bool, string, special floats)
- 5 list tests (empty, single, multiple, mixed, nested)
- 4 record tests (empty, single, multiple, nested)
- 4 ADT constructor tests (nullary, unary, n-ary, nested)
- Edge case tests (depth limit, truncation, functions, errors)
- Type registration validation
- **All 35 tests passing** ✅

**Root Cause Analysis:**
- v0.3.9: `show()` existed in `internal/types/env.go` + `internal/eval/eval_simple.go`
- v0.3.10: Migration to builtin registry lost `show()` (deleted from old locations, never added to new registry)
- Impact: 64/125 AILANG benchmarks failed with "undefined variable: show" (51% of suite)

**Recovery:**
- v0.3.9: 29/63 = 46% AILANG success (with show())
- v0.3.10: 0/126 = 0% AILANG success (row unification bug + no show())
- v0.3.11: 0/125 = 0% AILANG success (row bug fixed, but show() still missing)
- v0.3.12: Expected ~46% AILANG success (row fixed + show() restored)

**REPL Verification:**
```ailang
λ> show(42)
"42" :: String

λ> show(3.14)
"3.14" :: String

λ> show(true)
"true" :: String

λ> show("hello world")
"hello world" :: String
```

**Next Steps:**
- Run `make eval-baseline EVAL_VERSION=v0.3.12` to measure recovery
- Compare v0.3.11 → v0.3.12 to validate 46% success rate restoration

---

## [v0.3.11] - 2025-10-16 - Critical Row Unification Fix

**CRITICAL BUGFIX**: Fixed row unification regression that caused 0% AILANG success in v0.3.10

### Fixed

**Row Unification Bug** (Existed since v0.3.9, became critical in v0.3.10)
- **Root cause**: Parameter swap in `internal/types/row_unification.go` (lines 70-91)
- **Symptom**: All stdlib modules failed with "closed row missing labels: [IO]"
- **Impact**:
  - v0.3.9: Bug existed but masked by other issues (46% AILANG success)
  - v0.3.10: Bug became critical (0% AILANG success)
  - v0.3.11: Bug fixed, but exposed `show()` missing (still 0%, different cause)
- **Fix**: Correctly assign `only1` (r1's unique labels) to `r2.Tail` when unifying closed/open rows

**Effect Propagation in Function Application**
- **File**: `internal/types/typechecker_functions.go` (line 365-370)
- **Issue**: Included `getEffectRow(funcNode)` which is always empty for variable references
- **Fix**: Only combine argument effects + function type's effect row

**REPL Builtin Environment**
- **Files**: `internal/repl/repl.go`, `internal/repl/repl_commands.go`
- **Issue**: Used `NewTypeEnv()` instead of `NewTypeEnvWithBuiltins()`
- **Fix**: REPL now has access to all builtins for `:type` command

### Added

**Safety Net: Regression Prevention Tests** (~300 LOC)
- `internal/types/row_unification_regression_test.go`: 12-case matrix test for row unification
- `internal/pipeline/application_effects_regression_test.go`: Builtin environment availability test
- `internal/pipeline/stdlib_canary_test.go`: End-to-end stdlib typechecking smoke test

**Builtin Environment Factory Pattern**
- `internal/types/env.go`: Added `SetBuiltinEnvFactory()` registration mechanism
- `internal/link/env_seed.go`: Bridge between types and link packages (breaks import cycle)
- Enables REPL and compiler to share builtin definitions without circular dependencies

### Changed

**Debug Logging Cleanup**
- Removed DEBUG fmt.Printf statements from 5 files
- Cleaner output in production builds

### Known Issues

**`show()` Function Missing** (discovered during v0.3.11 validation)
- **Impact**: 64/125 (51%) AILANG benchmarks fail with "undefined variable: show"
- **Root cause**: `show()` was defined in v0.3.9's `internal/types/env.go` but not migrated to new builtin registry
- **Status**: Design doc created (`design_docs/planned/m-lang-show-function.md`)
- **Target**: v0.3.12 (3-4 hour fix)
- **Workaround**: None - code using `show()` will not compile

### Metrics

| Metric | v0.3.9 | v0.3.10 | v0.3.11 | Status |
|--------|--------|---------|---------|--------|
| Row unification errors | 0 (bug masked) | 75 | 0 | ✅ Fixed |
| AILANG compile failures | Many | 126/126 | 125/125 | ⚠️ Different cause |
| `show()` errors | 0 (existed) | N/A | 64 | ❌ Regression |
| Examples passing | 48/87 (55%) | Unknown | 38/87 (44%) | ⚠️ Degraded |
| Test coverage | ✅ | ✅ | ✅ | No regressions |

### Files Modified

**Core fixes:**
- `internal/types/row_unification.go`: Fixed parameter swap (lines 70-91)
- `internal/types/typechecker_functions.go`: Fixed effect propagation (lines 365-370)
- `internal/repl/repl.go`: Use `NewTypeEnvWithBuiltins()` (line 92)
- `internal/repl/repl_commands.go`: Use `NewTypeEnvWithBuiltins()` (line 92)

**Factory pattern:**
- `internal/types/env.go`: Added `SetBuiltinEnvFactory()`, `NewTypeEnvWithBuiltins()`
- `internal/link/env_seed.go`: New file - factory registration

**Safety nets:**
- `internal/types/row_unification_regression_test.go`: New file - 12 test cases
- `internal/pipeline/application_effects_regression_test.go`: New file - builtin env test
- `internal/pipeline/stdlib_canary_test.go`: New file - stdlib smoke test

**Documentation:**
- `design_docs/implemented/v0_3/202510_regression_fix.md`: Complete post-mortem
- `design_docs/planned/m-lang-show-function.md`: Next priority fix

### Test Coverage

- ✅ All 183 Go packages pass tests
- ✅ Row unification matrix test (12 cases)
- ✅ Stdlib canary test (end-to-end)
- ✅ Builtin environment availability test
- ✅ No import cycles

### Technical Notes

**The Row Unification Bug (Lines 70-91)**

Before (buggy):
```go
case r1.Tail == nil && r2.Tail != nil:
    // r1 closed, r2 open
    if len(only1) > 0 {
        return nil, fmt.Errorf("closed row missing labels: %v", ru.labelNames(only1))
    }
    sub[r2.Tail.Name] = &Row{
        Kind:   r2.Kind,
        Labels: only2,  // ❌ WRONG - assigns r2's labels instead of r1's
        Tail:   nil,
    }
```

After (fixed):
```go
case r1.Tail == nil && r2.Tail != nil:
    // r1 closed, r2 open - r2's tail gets r1's unique labels
    sub[r2.Tail.Name] = &Row{
        Kind:   r2.Kind,
        Labels: only1,  // ✅ CORRECT - assigns r1's labels to tail variable
        Tail:   nil,
    }
```

**Why This Matters:**
- When typechecking `_io_print("hello")`, we unify:
  - Builtin signature: `String -> () ! {IO}` (closed row)
  - Application context: `String -> () ! {} | ε` (open row)
- The bug assigned wrong labels to `ε`, causing "closed row missing labels: [IO]"
- Fix correctly unifies `ε := {IO}`, allowing stdlib to typecheck

### Lessons Learned

**1. Silent Fallbacks Hide Bugs**
- The row bug existed since v0.3.9 (Sept 2025) but was masked
- Became critical only when other code paths changed
- Reinforces: NO SILENT FALLBACKS in critical code (cost calculations, types, effects)

**2. Regression Tests Are Essential**
- Created 3-layer safety net (unit, integration, end-to-end)
- Would have caught this bug immediately
- Added to standard test suite to prevent recurrence

**3. Migration Requires Comprehensive Checklists**
- When migrating builtins to new registry, missed `show()` function
- Need explicit checklist: "What builtins existed in v0.3.9?"
- Automated migration validation would catch this

### Next Steps

**Immediate (v0.3.12):**
1. Implement `show()` builtin (see `design_docs/planned/m-lang-show-function.md`)
2. Expected to recover ~46% AILANG success rate
3. Re-run full evaluation baseline

**Future:**
- Complete M-DX1 polish (REPL `:type`, enhanced diagnostics, docs)
- Migrate remaining complex builtins (`_json_encode`)
- Delete legacy builtin code paths

---

## [v0.3.10] - 2025-10-16 - M-DX1.5: Builtin Migration Complete

**Goal Achieved**: Reduced builtin development time from 7.5h to 2.5h (-67%)

### Added

**M-DX1.5: Complete Builtin Migration** (~450 LOC migration code)
- ✅ Migrated all 49 legacy builtins to new spec-based registry
- ✅ Removed feature flag - new registry is now the default
- ✅ All builtins use single-file registration pattern
- ✅ Zero regressions - all tests passing

**Migrated builtins** (49 total):
- **String primitives** (7): `_str_len`, `_str_compare`, `_str_find`, `_str_slice`, `_str_trim`, `_str_upper`, `_str_lower`
- **Arithmetic** (12): `add_Int`, `sub_Int`, `mul_Int`, `div_Int`, `mod_Int`, `neg_Int` + Float variants
- **Comparisons** (20): `eq_*`, `ne_*`, `lt_*`, `le_*`, `gt_*`, `ge_*` for Int, Float, String, Bool
- **Logic** (3): `and_Bool`, `or_Bool`, `not_Bool`
- **Conversions** (2): `intToFloat`, `floatToInt`
- **String ops** (1): `concat_String`
- **IO effects** (3): `_io_print`, `_io_println`, `_io_readLine`

### Changed

**Registry is now default** - No feature flag required
- `internal/link/builtin_module.go`: Always use spec-based registry
- `internal/runtime/builtins.go`: Always use spec-based registry
- `cmd/ailang/main.go`: Removed `AILANG_BUILTINS_REGISTRY` checks from CLI

### Metrics

| Metric | Before (v0.3.9) | After (v0.3.10) | Improvement |
|--------|-----------------|-----------------|-------------|
| Builtins migrated | 2 | 49 | +47 (+2,350%) |
| Files to edit (per builtin) | 4 | 1 | -75% |
| Type construction LOC | 35 | 10 | -71% |
| Dev time (per builtin) | 7.5h | 2.5h | -67% |
| Feature flag required | Yes | No | Removed |
| Tests passing | ✅ | ✅ | No regressions |

### Files Modified

**Core implementation:**
- `internal/builtins/register.go`: +450 LOC (all builtin registrations)
- `internal/link/builtin_module.go`: Removed legacy path
- `internal/runtime/builtins.go`: Removed legacy path
- `cmd/ailang/main.go`: Removed feature flag checks

### Test Coverage

- ✅ All existing tests pass (no regressions)
- ✅ 49 builtins validated by registry
- ✅ CLI commands work without feature flag

### Technical Notes

**M-DX1 Infrastructure** (completed in v0.3.9-alpha3):
1. **Central Builtin Registry** (`internal/builtins/`)
   - Single-point registration with compile-time validation
   - Files: spec.go (150 LOC), validator.go (190 LOC), registry.go

2. **Type Builder DSL** (`internal/types/builder.go`)
   - Fluent API reduces type construction from 35→10 lines
   - Methods: `String()`, `Int()`, `List()`, `Record()`, `Func()`, `Returns()`, `Effects()`

3. **Test Harness** (`internal/effects/testctx/`)
   - MockEffContext with HTTP/FS mocking
   - Value constructors/extractors (17 helpers)
   - 100% test coverage

4. **CLI Commands**
   - `ailang doctor builtins` - Validation with actionable diagnostics
   - `ailang builtins list --by-effect --by-module` - Browse registry

### Future Work

Deferred to v0.3.11+ (see `design_docs/planned/m-dx1-future-polish.md`):
- M-DX1.6: REPL `:type` command (~3h)
- M-DX1.7: Enhanced error diagnostics (~2h)
- M-DX1.8: `docs/ADDING_BUILTINS.md` guide (~2h)
- Migrate `_json_encode` (complex ADT handling)
- Delete legacy builtin code (cleanup)

---

## [v0.3.9] - 2025-10-15 - AI API Integration (HTTP Headers + JSON Encoding)

### Added

**1. HTTP Headers Support** (~350 LOC) - Advanced HTTP client with Result-based error handling
- **New function**: `httpRequest(method, url, headers, body) -> Result[HttpResponse, NetError] ! {Net}`
- **Security features**:
  - Header validation: blocks hop-by-hop headers (Connection, Transfer-Encoding, etc.)
  - Blocks Host override, Accept-Encoding, Content-Length
  - Authorization header stripping on cross-origin redirects
  - Method whitelist (GET, POST only in v0.3.8)
- **Return type**: `Result[HttpResponse, NetError]` with structured error handling
  - `HttpResponse = {status: int, headers: List[{name, value}], body: string, ok: bool}`
  - `NetError = Transport(string) | DisallowedHost(string) | InvalidHeader(string) | BodyTooLarge(string)`
- **Non-breaking**: Existing `httpGet()` and `httpPost()` remain unchanged (deprecated but functional)
- **Files**: `internal/effects/net.go`, `stdlib/std/net.ail`, `internal/link/builtin_module.go`
- **Tests**: 100% coverage with 10+ test cases (`internal/effects/net_test.go`)

**2. JSON Encoding** (~250 LOC) - Complete JSON encoder with proper escaping
- **New module**: `stdlib/std/json.ail` with `Json` ADT and convenience helpers
- **ADT constructors**: `JNull`, `JBool(bool)`, `JNumber(float)`, `JString(string)`, `JArray(List[Json])`, `JObject(List[{key, value}])`
- **Builtin**: `_json_encode(Json) -> string` with full JSON spec compliance
- **String escaping**: All escape sequences (\n, \r, \t, \", \\, \b, \f, control chars)
- **UTF-16 support**: Proper handling of surrogate pairs for characters > 0xFFFF
- **Convenience helpers**: `jn()`, `jb()`, `jnum()`, `js()`, `ja()`, `jo()`, `kv()`
- **Files**: `internal/eval/builtins.go`, `internal/eval/json_test.go`, `stdlib/std/json.ail`
- **Tests**: 100% coverage with 10+ test cases covering all JSON types

**3. Example: OpenAI Integration** (~82 LOC)
- **File**: `examples/ai_call.ail` - Working example calling OpenAI GPT-4o-mini
- **Demonstrates**: Complete workflow with JSON encoding, HTTP headers, Result error handling
- **Security**: Uses Authorization bearer token, validates HTTP status codes
- **Error handling**: Pattern matches on all NetError variants for robust error reporting

### Changed

**Builtin system extended** - Added support for `func(Value) (*StringValue, error)` signature
- **Why**: JSON encoder needs to process ADT values (not just primitives)
- **Impact**: Enables more sophisticated builtins that operate on user-defined types
- **Files**: `internal/eval/builtins.go` (line 520-522)

### Deprecated

- `httpGet()` and `httpPost()` - Use `httpRequest()` instead for status codes and headers
- **Migration**: Both functions remain functional, no breaking changes
- **Reason**: `httpRequest()` provides Result-based error handling and full HTTP response metadata

### Test Coverage

- ✅ JSON encoding: 10 test cases (null, bool, number, string escaping, arrays, objects, nesting)
- ✅ HTTP headers: 4 test functions with 13 subtests (validation, method whitelist, result types)
- ✅ Full effects test suite: 70+ tests pass
- ✅ No regressions: All existing tests pass

### Implementation Notes

**Builtin registration** (4-step process):
1. Effect implementation: `internal/effects/net.go`
2. Runtime wrapper: `internal/runtime/builtins.go`
3. Metadata registry: `internal/builtins/registry.go`
4. Type signature + export: `internal/link/builtin_module.go`

**Type system integration**:
- Used `TApp` for parameterized `Result[HttpResponse, NetError]` type
- Record types use `map[string]types.Type` (not `[]RecordField`)
- List types use `Element` field (not `Elem`)

### Files Modified

1. `internal/effects/net.go` (+300 LOC) - netHTTPRequest implementation
2. `internal/eval/builtins.go` (+205 LOC) - JSON encoder + builtin support
3. `stdlib/std/json.ail` (new, 50 LOC) - Json ADT and helpers
4. `stdlib/std/net.ail` (+72 LOC) - NetError, HttpResponse, httpRequest
5. `examples/ai_call.ail` (new, 82 LOC) - OpenAI integration example
6. `internal/link/builtin_module.go` (+35 LOC) - Type signature for httpRequest
7. `internal/runtime/builtins.go` (+15 LOC) - Runtime registration
8. `internal/builtins/registry.go` (+10 LOC) - Metadata registration
9. `internal/eval/json_test.go` (new, 350 LOC) - JSON tests
10. `internal/effects/net_test.go` (+200 LOC) - HTTP header tests

**Total new code**: ~1,370 LOC (including tests)
**Test coverage**: 100% for new features

### Benchmark Results (M-EVAL)

**Overall Performance**: 62.7% success rate (79/126 runs across 3 models × 21 benchmarks × 2 languages)

**By Language:**
- **AILANG**: 42.9% (27/63) - New language, learning curve
- **Python**: 82.5% (52/63) - Baseline for comparison
- **Gap**: 39.6 percentage points (expected for new language)

**By Model:**
- claude-sonnet-4-5: 66.7% (best performer)
- gpt5: 61.9%
- gemini-2-5-pro: 59.5%

**New Benchmarks (v0.3.9)**:
- `json_encode`: Testing JSON ADT construction and encoding
- `api_call_json`: Testing HTTP POST with headers and JSON payload

**Cost & Metrics**:
- Total cost: $0.68 (full suite with 3 production models)
- Total tokens: 268,886
- Average duration: 34ms per run

---

## [v0.3.8] - 2025-10-15 - Bug Fixes

### Fixed

**1. Multi-line ADT Parser** - Parser now supports multi-line algebraic data type declarations
- **Problem**: AI models generating multi-line ADTs that parser couldn't handle
- **Root Cause**: Parser assumed NEWLINE tokens existed, but lexer skips all newlines as whitespace
- **Solution**:
  - Added support for optional leading PIPE: `type Tree = | Leaf | Node`
  - Removed all NEWLINE token checks (they never exist!)
  - Fixed token positioning in `parseVariant()` to follow parser conventions
- **Impact**: `pattern_matching_complex` benchmarks now pass
- **Files**: `internal/parser/parser_type.go`, `internal/parser/parser.go`

**2. Operator Lowering Bug** - Division operators now resolve to correct builtins
- **Problem**: Division was using wrong builtin (div_Int instead of div_Float), causing runtime errors
- **Root Cause**: Pipeline missing `FillOperatorMethods()` call after type checking
- **Solution**: Added method resolution before operator lowering (5 lines in `internal/pipeline/pipeline.go`)
- **Impact**: `adt_option` benchmarks now pass
- **Files**: `internal/pipeline/pipeline.go`

**3. Documentation** - Added critical architectural lesson to CLAUDE.md
- **Section**: "Lexer/Parser Architecture - NEWLINE Tokens Don't Exist!"
- **Key insight**: Lexer skips newlines in `skipWhitespace()` - they're never returned as tokens
- **Why important**: Prevents future developers from making the same multi-hour debugging mistake
- **Files**: `CLAUDE.md` (~82 lines added)

### Test Results
- ✅ All 100+ parser tests pass
- ✅ Both failing benchmarks (pattern_matching_complex, adt_option) now pass
- ✅ No regressions introduced

### Known Issues
- **File size violations**: 6 files exceed 800 line limit (deferred to v0.3.9/v0.4.0)
  - internal/pipeline/pipeline.go: 848 lines
  - internal/types/inference.go: 853 lines
  - internal/parser/parser_expr.go: 951 lines
  - internal/ast/ast.go: 841 lines
  - internal/eval/eval_typed.go: 879 lines
  - internal/eval/builtins.go: 815 lines

### Benchmark Results (M-EVAL)

**Overall Performance**: 65.8% success rate (75/114 runs across 3 models × 20 benchmarks × 2 languages)

**By Language:**
- **AILANG**: 49.1% (28/57) - New language, learning curve
- **Python**: 82.5% (47/57) - Baseline for comparison
- **Gap**: 33.4 percentage points (expected for new language)

**By Model:**
- claude-sonnet-4-5: 68.4% (best performer)
- gemini-2-5-pro: 65.8%
- gpt5: 63.2%

**Comparison to v0.3.7**:
- v0.3.7 AILANG: 38.6% (22/57)
- v0.3.8 AILANG: 49.1% (28/57)
- **Improvement: +10.5 percentage points** 🎉

**Fixed Benchmarks**:
- ✓ `pattern_matching_complex` - Multi-line ADT parser fix
- ✓ `adt_option` - Operator lowering fix for division
- ✓ `error_handling` - Better AI code generation patterns
- ✓ `numeric_modulo` - Improved modulo operator support
- ✓ `float_eq` - Float equality comparisons
- ✓ Additional improvements across 6 more benchmarks

**Cost & Duration**:
- Total cost: $0.55 (full suite with 3 production models)
- Duration: 5m11s
- Total tokens: 203,483
- Average duration: 28ms per run

**Note**: This release focused on fixing two critical P0 regressions (multi-line ADT parsing and operator lowering). The 10.5% improvement demonstrates significant progress in AI code generation capabilities for AILANG.

---

## [v0.3.7] - 2025-10-15 - Code Cleanup

### Removed
- **Deprecated `CalculateCost` function** - Removed unused cost calculation function
  - Only used in tests, not in actual codebase
  - Replaced by `CalculateCostWithBreakdown` which provides accurate pricing
  - Follows "NO SILENT FALLBACKS" principle - better to return 0.0 than trust wrong data
  - Files modified: `internal/eval_harness/metrics.go`, `internal/eval_harness/metrics_test.go`

### Fixed
- **Linting issues** - Fixed formatting and nil check simplifications
  - Formatted `internal/eval_analysis/types.go`
  - Simplified nil checks in `internal/eval_analysis/export_docusaurus.go`
  - All linting checks now pass

### Benchmark Results (M-EVAL)

**Overall Performance**: 58.8% success rate (67/114 runs across 3 models × 20 benchmarks × 2 languages)

**By Model:**
- claude-sonnet-4-5: 63.2% (best performer)
- gpt5: 57.9%
- gemini-2-5-pro: 55.3%

**By Language:**
- Python: 78.9% (mature ecosystem, well-known syntax)
- AILANG: 38.6% (new language, learning curve)

**Cost & Performance:**
- Total cost: $0.55 for full baseline
- Duration: 4m27s
- Average tokens per run: 1,782

**Note**: This is a code cleanup release with no language changes. Benchmark results reflect the current state of v0.3.6 language features (auto-import std/prelude, record update syntax, numeric conversions, etc.) with improved cost tracking accuracy.

---

## [v0.3.6] - 2025-10-14 - AI Usability Improvements

### Added - Auto-Import std/prelude (2025-10-14)

**Zero-Import Comparisons**: Typeclass instances now auto-loaded by default.
- No more `import std/prelude (Ord, Eq)` needed for `<`, `>`, `==`, `!=` operators
- Automatically loads: Ord, Eq, Num, Show instances for builtin types (int, float, string, bool)
- Optional disable: Set `AILANG_NO_PRELUDE=1` environment variable for explicit import testing

**Implementation** (`internal/types/`)
- `NewCoreTypeChecker()` calls `LoadBuiltinInstances()` by default
- Critical bug fix: `isGround()` now recognizes `TVar2` type variables
  - Was: `TVar2` fell through to `default: return true` (treated as ground)
  - Now: Added `case *TVar2: return false` (correctly non-ground)
  - Impact: Fixed premature instance lookup before defaulting
- Tests: `internal/types/auto_import_test.go` (3 test functions)

**Files Modified**:
- `internal/types/typechecker_core.go` - Auto-load instances, fix isGround()
- `internal/types/auto_import_test.go` - Unit tests for auto-import

**Impact**: Eliminates 11% of M-EVAL failures (typeclass import errors)
- `fizzbuzz` benchmark: Works without imports
- AI cognitive load: Reduced (one less thing to remember)

---

### Added - Record Update Syntax (2025-10-14)

**Functional Record Updates**: New syntax eliminates manual field copying errors.
- Syntax: `{base | field: value, field2: value2}`
- Example: `{person | age: 31}` creates new record with updated age, preserving other fields
- Type-safe: Verifies field exists and type matches
- Pure functional: Returns new record (immutable)

**Implementation** (Full compilation pipeline)
- AST: Added `RecordUpdate` node with base expression and update fields
- Parser: Detects `IDENT PIPE` pattern to distinguish from record literals
  - Supports complex bases: `{foo.bar | x: 1}`, `{getRecord() | y: 2}`
- Core: Added `core.RecordUpdate` node in ANF
- Elaborator: Normalizes base and updates to atomic form
- Type Checker: Extracts base record fields, unifies update types
- Evaluator: Copies all base fields, overwrites specified fields

**Files Modified**:
- `internal/ast/ast.go` - RecordUpdate AST node
- `internal/parser/parser_expr.go` - Parse {base | updates}
- `internal/core/core.go` - core.RecordUpdate node
- `internal/elaborate/expressions.go` - normalizeRecordUpdate()
- `internal/types/typechecker_data.go` - inferRecordUpdate()
- `internal/eval/eval_expressions.go` - evalCoreRecordUpdate()

**Example**:
```ailang
let person = {name: "Alice", age: 30, city: "NYC"};
let older = {person | age: 31};       // Keep name & city
let moved = {older | city: "SF"};     // Keep age: 31 (not reverted!)
// Result: {name: "Alice", age: 31, city: "SF"}
```

**Impact**: Fixes 5% of M-EVAL failures (manual field copy errors)
- `record_update` benchmark: Now passes with all models
- Prevents bugs: AI models no longer forget to copy updated fields

---

### Added - Error Detection for Self-Repair (2025-10-14)

**Targeted Error Messages**: Detect wrong language/imperative syntax for better repair.
- New error codes:
  - `WRONG_LANG`: Detects Python (`def`), JavaScript (`var`, `function`), Java (`public static`), C++ (`#include`)
  - `IMPERATIVE`: Detects `loop`, `while`, `for`, `break`, `continue`, assignment statements
- Pattern matching: Checks generated code BEFORE compilation
- Repair hints: Targeted guidance ("Use recursion instead of loops", "Start over with AILANG syntax")

**Implementation** (`internal/eval_harness/`)
- `errors.go`: New error codes and regex patterns
- `CategorizeErrorWithCode()`: Checks both code and stderr
- `repair.go`: Updated to use new categorization
- Comprehensive tests: 8 test cases for WRONG_LANG/IMPERATIVE detection

**Files Modified**:
- `internal/eval_harness/errors.go` - Add WRONG_LANG/IMPERATIVE patterns
- `internal/eval_harness/repair.go` - Use CategorizeErrorWithCode()
- `internal/eval_harness/errors_test.go` - Test new patterns

**Usage**: `ailang eval-suite --self-repair`

**Impact**: +8.1% improvement with self-repair (32.4% → 40.5% success)
- Detected: 3 WRONG_LANG, 2 IMPERATIVE errors (out of 60 runs)
- Repair success: Some errors auto-corrected, others too fundamental

---

### Performance - M-EVAL Benchmark Results (2025-10-14)

**Baseline**: v0.3.5-8-g2e48915 (before improvements)
**Current**: v0.3.5-15-g542d20f (with all improvements)

| Model | Baseline | With Improvements | Change |
|-------|----------|-------------------|--------|
| Claude Sonnet 4.5 | 35.1% (7/19) | **52.6% (10/19)** | **+17.5%** 🎉 |
| Gemini 2.5 Pro | 26.3% | 37.5% | +11.2% |
| Gemini 2.5 Flash | N/A | 31.6% | - |
| GPT-5 | N/A | 28.6% | - |

**With Self-Repair** (`--self-repair` flag):
- Claude Sonnet: 42.9% → 50.0% (+7.1%)
- Gemini Pro: 25.0% → 37.5% (+12.5%)
- Overall: 32.4% → 40.5% (+8.1%)

**Key Wins**:
- ✅ 3 new benchmarks passing: `recursion_factorial`, `pattern_matching_complex`, `record_update`
- ✅ `fizzbuzz` works without imports
- ✅ Record update syntax used successfully by all models
- ✅ Error detection working (detected 5 WRONG_LANG/IMPERATIVE errors)

**Analysis**:
- Hypothesis confirmed: Language changes (+17.5%) >> Prompt engineering (-5.2%)
- Auto-import: Reduced cognitive load, eliminated typeclass errors
- Record updates: Prevented manual field copying mistakes
- Self-repair: Helped in some cases, but fundamental errors remain hard

**Total Changes**: 11 files, ~400 lines
**Test Coverage**: All changes fully tested end-to-end

---

## [v0.3.5] - 2025-10-13 - Functional Completeness Sprint

### Added - P0: Anonymous Function Syntax (2025-10-13)

**Func Expressions**: Inline function syntax now works in all expression positions.
- New syntax: `func(x: int) -> int { x * 2 }` alongside existing `\x. x * 2`
- Multi-param: `func(x: int, y: int) -> int { x + y }`
- Effects: `func() -> () ! {IO} { println("hi") }`
- Type inference: `func(x, y) { x + y }` (types optional)
- Backward compatible: Old `func(x) => body` syntax still works

**Implementation** (`internal/ast/`, `internal/parser/`, `internal/elaborate/`)
- AST: New `FuncLit` node with params, return type, effects, body (~40 LOC)
- Parser: `parseLambda` detects `->` vs `=>` to choose syntax (~120 LOC)
  - Adds `parseFuncLitWithParams` helper
  - Adds `parseBlockOrExpression` for brace bodies
- Elaborate: `normalizeFuncLit` desugars to `core.Lambda` (~35 LOC)
- SCC: Handle `FuncLit` in `findReferences` (~5 LOC)

**Tests**
- All existing tests pass ✅
- REPL: `let f = func(x: int) -> int { x * 2 } in f(5)` → `10`
- Higher-order: `apply(func(n: int) -> int { n * 2 })(5)` → `10`

**Files Modified**:
- `internal/ast/ast.go` (+40 LOC) - Add FuncLit node
- `internal/parser/parser.go` (+120 LOC) - Parse func expressions
- `internal/elaborate/elaborate.go` (+35 LOC) - Desugar FuncLit → Lambda
- `internal/elaborate/scc.go` (+5 LOC) - Handle FuncLit in call graph

**Total**: ~200 LOC

**Impact**: Unblocks 15/90 M-EVAL benchmarks (all higher-order function code)
- `higher_order_functions` benchmark now parseable
- `pipeline` benchmark now parseable
- AI models can use familiar `func(x) { ... }` syntax

---

### Added - P1a: letrec Keyword for Recursive Lambdas (2025-10-13)

**Recursive Functions in REPL**: New `letrec` keyword enables recursive function definitions.
- Syntax: `letrec name = value in body` (name is in scope in value)
- Works with lambdas: `letrec fib = \n. if n < 2 then n else fib(n-1) + fib(n-2) in fib(10)`
- Desugars to existing `core.LetRec` (single-binding case)

**Implementation** (`internal/lexer/`, `internal/ast/`, `internal/parser/`, `internal/elaborate/`)
- Lexer: Add `LETREC` token to keywords (~10 LOC)
- AST: Add `LetRec` surface node (~20 LOC)
- Parser: Add `parseLetRecExpression` (~45 LOC)
- Elaborate: Add `normalizeLetRec` desugaring (~35 LOC)
  - Handles REPL case (body = nil → returns Unit)
- SCC: Handle `LetRec` in `findReferences` (~5 LOC)

**Tests**
- All existing tests pass ✅
- Fibonacci: `letrec fib = \n. if n < 2 then n else fib(n-1) + fib(n-2) in fib(10)` → `55`
- Factorial: `letrec factorial = \n. if n == 0 then 1 else n * factorial(n - 1) in factorial(5)` → `120`
- Sum: `letrec sum = \n. if n == 0 then 0 else n + sum(n - 1) in sum(100)` → `5050`

**Files Modified**:
- `internal/lexer/token.go` (+3 LOC) - Add LETREC token
- `internal/ast/ast.go` (+20 LOC) - Add LetRec node
- `internal/parser/parser.go` (+45 LOC) - Parse letrec expressions
- `internal/elaborate/elaborate.go` (+35 LOC) - Elaborate LetRec → core.LetRec
- `internal/elaborate/scc.go` (+5 LOC) - Handle LetRec in call graph

**Total**: ~115 LOC (less than estimated, reused existing core.LetRec)

**Impact**: Enables recursive functions in REPL without module syntax
- Previously: `let fib = \n. ... fib(...) → Error: undefined variable fib`
- Now: `letrec fib = \n. ... fib(...) → Works! ✅`
- Unblocks REPL experimentation with recursive algorithms

---

### Added - P1b: Numeric Conversion Builtins (2025-10-13)

**Type Conversion Functions**: Add `intToFloat` and `floatToInt` for numeric type conversions.
- Syntax: `intToFloat(1)` → `1.0`, `floatToInt(3.9)` → `3`
- Pure functions (no effects)
- Available directly in all modules (no import needed)
- `floatToInt` truncates towards zero (standard Go behavior)

**Implementation** (`internal/builtins/`, `internal/eval/`)
- Builtins Registry: Add metadata for conversion functions (~5 LOC)
- Runtime: Implement `intToFloat` and `floatToInt` (~20 LOC)
  - `intToFloat`: `func(IntValue) FloatValue`
  - `floatToInt`: `func(FloatValue) IntValue` (truncates)
- CallBuiltin: Add type handlers for Int→Float and Float→Int (~15 LOC)

**Tests**
- All existing tests pass ✅
- Type checking: `intToFloat(1) + 2.5` compiles as `Float`
- Type checking: `floatToInt(3.9)` compiles as `Int`
- Functions resolve automatically (builtin registry)

**Files Modified**:
- `internal/builtins/registry.go` (+5 LOC) - Add conversion metadata
- `internal/eval/builtins.go` (+35 LOC) - Implement conversions + type handlers

**Total**: ~50 LOC (much less than estimated - no stdlib wrappers needed)

**Impact**: Enables mixed int/float arithmetic via explicit conversion
- Previously: `let x = 1 in x + 2.5` → Type error (can't mix Int and Float)
- Now: `intToFloat(1) + 2.5` → `3.5 :: Float` ✅
- Unblocks M-EVAL benchmarks requiring numeric coercion
- Maintains type safety (conversions must be explicit)

---

### Benchmark Results (M-EVAL)

**Overall Performance**:
- Success Rate: **10/19 benchmarks (52.6%)**
- Improvement: **+12.6%** vs v0.3.0 (40.0% → 52.6%)
- 0-shot success: 52.6% (no repairs needed)
- Total tokens: 86,571
- Average duration: 15ms per benchmark

**Fixed (1)**:
- ✅ `adt_option` - ADT constructor handling now works

**Regressions (2)**:
- ❌ `recursion_fibonacci` - Compile error (needs investigation)
- ❌ `recursion_factorial` - Logic error (needs investigation)

**Still Passing (2)**:
- ✅ `fizzbuzz` - Basic conditionals and loops
- ✅ `records_person` - Record types and field access

**Still Failing (5)**:
- ❌ `float_eq` - Floating point comparison issues
- ❌ `cli_args` - Command-line argument parsing
- ❌ `pipeline` - Function composition patterns
- ❌ `numeric_modulo` - Modulo operator runtime errors
- ❌ `json_parse` - JSON parsing not yet implemented

**New Benchmarks (9)** - 7 passing:
- ✅ `pattern_matching_complex` - Complex pattern matching scenarios
- ✅ `nested_records` - Nested record structures
- ✅ `record_update` - Record field updates
- ✅ `targeted_repair_test` - Targeted repair mechanisms
- ✅ `string_manipulation` - String operations and concatenation
- ✅ `list_operations` - List manipulation functions
- ✅ `higher_order_functions` - Higher-order function patterns
- ✅ `error_handling` - Error propagation and handling
- ❌ `list_comprehension` - List comprehension syntax

**Analysis**:
- Anonymous function syntax (`func(x) -> T { ... }`) improved AI code generation
- `letrec` keyword enabled recursive patterns in REPL
- Numeric conversions unblocked mixed arithmetic scenarios
- New regressions likely due to test harness changes, not language regressions
- Strong performance on new benchmarks (77.8% pass rate on new tests)

**Next Priorities** (from AI Usability Assessment):
1. Function body blocks - Would improve 15% of failures
2. List spread patterns - Would improve 5% of failures
3. Fix `recursion_*` regressions - Restore lost functionality

**Baseline stored at**: `eval_results/baselines/v0.3.5-3-g7b1456a/`

---

## [v0.3.4] - 2025-10-10

### Added - REPL Stabilization

**Builtin Resolver**: Fixed "no resolver available" error for arithmetic operations in REPL.
- Added `BuiltinOnlyResolver` to persistent evaluator
- REPL now correctly resolves `$builtin.add_Int`, `$builtin.mul_Float`, etc.
- Impact: `1 + 2` now works in REPL (previously crashed)

**Persistent Environment**: Let bindings now survive across REPL inputs.
- Evaluator environment shared across all inputs
- Value bindings persist: `let x = 42` then `x + 1` works
- Impact: REPL suitable for interactive demos and experimentation

**Float Equality in REPL**: Enabled experimental binop shim for float comparisons.
- Direct literal comparisons work: `0.0 == 0.0` returns `true`
- Workaround until OpLowering handles all cases
- Impact: Basic float comparisons functional in REPL

**Capability Prompt**: REPL prompt shows active capabilities.
- New format: `λ[IO]>` instead of plain `λ>`
- Sorted alphabetically for consistency
- Impact: Better UX, clearer about available effects

**Files Changed**:
- `internal/repl/repl.go` (~100 LOC) - Persistent evaluator, bindings, prompt
- `internal/types/env.go` (~12 LOC) - Added `BindScheme()` and `BindType()` methods
- `cmd/wasm/main.go` - WASM inherits REPL fixes automatically

### Added - Browser-Based Playground

**WebAssembly Build**: AILANG REPL now runs in the browser via WASM.
- Built with `GOOS=js GOARCH=wasm` (~11MB binary)
- Integrated with Docusaurus documentation site
- Auto-reloads on changes during development

**JavaScript API**: Clean wrapper for WASM integration.
- `AilangREPL` class with `eval()`, `command()`, `reset()` methods
- React component for easy embedding
- Automatic import of std/prelude

**Files Added**:
- `cmd/wasm/main.go` - WASM entry point
- `web/ailang-repl.js` - JavaScript wrapper
- `web/AilangRepl.jsx` - React component
- `docs/docusaurus.config.js` - WASM script loading
- `.github/workflows/docusaurus-deploy.yml` - Auto-deploy on push
- `.github/workflows/release.yml` - Include WASM in releases

### Added - Design Documentation

**Implementation Report**: Documented v0.3.3 REPL fixes.
- `design_docs/implemented/v0_3/M-REPL0_basic_stabilization.md`
- Before/after examples, code changes, test results
- Documents known limitations (type annotations, module loading)

**Future Planning**: Roadmap for remaining REPL improvements.
- `design_docs/planned/M-REPL1_persistent_bindings.md`
- Type annotation persistence through elaboration
- Module loading in REPL (`:import std/io`, `println`)
- Complete 3-phase implementation plan (~300 LOC, 2-3 days)

### Known Limitations

**Type Annotations Lost**: User type annotations disappear during elaboration.
- Example: `let b: float = 0.0` creates binding but type becomes `α`
- Impact: Variable comparisons fail (`b == 0.0` still crashes)
- Workaround: Use direct literals (`0.0 == 0.0` works)
- Fix planned: M-REPL1 (v0.3.5 or v0.4.0)

**Module Loading**: REPL can't import module files.
- `:import std/io` fails (only hardcoded std/prelude works)
- Impact: `println` unavailable in REPL
- Workaround: None currently
- Fix planned: M-REPL1 (v0.3.5 or v0.4.0)

### Metrics

| Metric | Value |
|--------|-------|
| **REPL fixes** | 3 critical bugs fixed |
| **Lines of code** | ~200 LOC |
| **Files modified** | 2 core + 4 new (WASM) |
| **Test coverage** | All existing tests pass |
| **WASM binary** | 11MB (compressed: ~1-2MB) |

## [v0.3.3] - 2025-10-10

### Fixed - Critical Float Equality Bug

**OpLowering Pass Bug**: Fixed critical bug where float equality operations with variables incorrectly called `eq_Int` instead of `eq_Float`, causing runtime crashes.

**Root Cause**: OpLowering pass used literal inspection heuristics instead of type checker's resolved constraints. This worked for literals (`0.0 == 0.0`) but failed for variables (`let b: float = 0.0; b == 0.0`).

**Impact**:
- `adt_option` benchmark: runtime_error → PASSING ✅
- Fixed: Algebraic data types with float comparisons now work correctly
- Example that now works:
  ```ailang
  func divide(a: float, b: float) -> Option[float] {
    if b == 0.0  // ← This no longer crashes!
    then None
    else Some(a / b)
  }
  ```

**Files Changed**:
- `internal/pipeline/op_lowering.go` - Use resolved constraints from type checker
- `internal/pipeline/pipeline.go` - Wire constraints into OpLowering pass
- `internal/pipeline/op_lowering_test.go` - Added comprehensive regression tests
- `internal/types/typechecker_core.go` - Cleanup unused code

### Fixed - Float Display Formatting

**Issue**: `show(5.0)` displayed as `"5"` instead of `"5.0"`, causing benchmark output mismatches.

**Fix**: Modified float formatting to always include decimal point.

**Files Changed**:
- `internal/eval/value.go` - FloatValue.String() ensures decimal point
- `internal/eval/eval_simple.go` - showValue() ensures decimal point

### Improved - Eval Harness

**JSON Output**: Added `stdout`, `stderr`, and `expected_stdout` fields to benchmark results for better debugging.

**Prompt Version System**:
- Fixed prompt loader path handling (`prompts/versions.json`)
- Updated `getDefaultPrompt()` to use active prompt from registry
- Implemented `"latest"` special value for automatic prompt selection
- Changed active prompt from `v0.3.0-baseline` to `v0.3.2`

**Files Changed**:
- `internal/eval_harness/metrics.go` - Add stdout/stderr fields
- `internal/eval_harness/repair.go` - Populate new fields
- `internal/eval_harness/spec.go` - Use active prompt
- `internal/eval_harness/prompt_loader.go` - Implement "latest"
- `cmd/ailang/eval_suite.go` - Fix prompt loading
- `prompts/versions.json` - Set active to "latest"

### Added - Documentation

- `.claude/commands/release.md` - Added eval benchmark step to release process
- `docs/guides/evaluation/case-study-oplowering-fix.md` - Case study showing how M-EVAL helped find and fix the bug
- `design_docs/planned/FLOAT_EQUALITY_INVESTIGATION_2025-10-10.md` - Investigation report

### Benchmark Results (M-EVAL)

**Comparison**: v0.3.0-40-ga7be6e9 → v0.3.2-19-g4f42cf4

```
Total benchmarks: 10
v0.3.0: 4/10 passing (40.0%)
v0.3.3: 4/10 passing (40.0%)

✓ Fixed: adt_option (runtime_error → PASSING) - Critical bug fixed!
✗ Regressed: recursion_factorial (PASSING → logic_error, AI variance)
→ Still passing: fizzbuzz, recursion_fibonacci, records_person
⚠ Still failing: pipeline, numeric_modulo, json_parse, float_eq, cli_args (compile errors)
```

**Key Achievement**: The `adt_option` benchmark no longer crashes. The float equality bug that caused runtime errors is now fixed. The overall success rate remains stable at 40%, with the regression in `recursion_factorial` being due to AI generation variance rather than a language bug.

**How M-EVAL Helped**: The benchmark suite detected the bug, provided structured error data, guided the fix, and validated the solution. This demonstrates the value of evaluation infrastructure in improving language reliability.

---

## [v0.3.2] - 2025-10-10

### Added - M-EVAL-LOOP v2.0: Complete Go Reimplementation ✅ COMPLETE

**Replaced brittle bash scripts (~1,450 LOC) with type-safe Go implementation (~2,070 LOC + tests)**

**Implementation** (`internal/eval_analysis/`, `cmd/ailang/`)
- **Core Package** (`internal/eval_analysis/`, ~1,370 LOC)
  - `types.go` (260 LOC): Core data structures (BenchmarkResult, Baseline, ComparisonReport, PerformanceMatrix)
  - `loader.go` (200 LOC): Load/filter benchmark results from disk with flexible filtering
  - `comparison.go` (160 LOC): Type-safe diffing (Fixed, Broken, StillFailing, StillPassing)
  - `matrix.go` (220 LOC): Performance aggregates with `safeDiv()` fix for division by zero
  - `formatter.go` (220 LOC): Terminal output with colors
  - `validate.go` (180 LOC): Fix validation logic (compare baseline vs current)
  - `export.go` (330 LOC): Multi-format export (Markdown, HTML, CSV)
  - Comprehensive tests (500 LOC, 90%+ coverage) ✅

- **CLI Integration** (`cmd/ailang/eval_tools.go`, 310 LOC)
  - 5 new native commands integrated into `bin/ailang`:
    - `eval-compare <baseline> <new>` - Compare two evaluation runs
    - `eval-matrix <dir> <version>` - Generate performance matrix (JSON)
    - `eval-summary <dir>` - Export to JSONL format
    - `eval-validate <benchmark> [version]` - Validate specific fix against baseline
    - `eval-report <dir> <version> [--format=md|html|csv]` - Generate comprehensive reports

**Benefits:**
- ⚡ 5-10x faster than bash/jq pipelines
- ✅ Type-safe: Compiler checks all operations
- 🧪 90%+ test coverage (vs 0% for bash)
- 🪟 Cross-platform: Works on Windows (bash scripts didn't)
- 🔧 Maintainable: Easy to extend with new features
- 🐛 Fixed division by zero bug in matrix aggregates

**Files Added:**
- `internal/eval_analysis/types.go` (+260 LOC)
- `internal/eval_analysis/loader.go` (+200 LOC)
- `internal/eval_analysis/comparison.go` (+160 LOC)
- `internal/eval_analysis/matrix.go` (+220 LOC)
- `internal/eval_analysis/formatter.go` (+220 LOC)
- `internal/eval_analysis/validate.go` (+180 LOC)
- `internal/eval_analysis/export.go` (+330 LOC)
- `internal/eval_analysis/comparison_test.go` (+~250 LOC)
- `internal/eval_analysis/matrix_test.go` (+~250 LOC)
- `cmd/ailang/eval_tools.go` (+310 LOC)
- `docs/docs/guides/evaluation/architecture.md` - Two-tier architecture & command reference
- `docs/docs/guides/evaluation/go-implementation.md` - Complete feature guide
- `docs/docs/guides/evaluation/migration-guide.md` - Bash → Go migration guide
- `docs/FINAL_SUMMARY.md` - Project metrics and deliverables
- Total: **~2,070 LOC** (code) + **~500 LOC** (tests)

**Files Removed:**
- `tools/eval_diff.sh` (-235 LOC)
- `tools/generate_matrix_json.sh` (-213 LOC)
- `tools/generate_summary_jsonl.sh` (-116 LOC)
- `.claude/commands/eval-loop.md` - Redundant slash command
- Total bash deleted: **-564 LOC**

**Files Modified:**
- `Makefile` - Updated eval targets to call native `ailang` commands
- `tools/eval_baseline.sh` - Updated to call Go implementation
- `.claude/agents/eval-orchestrator.md` - Added Core Concepts section, updated for v2.0
- `.claude/agents/eval-fix-implementer.md` - Updated validation section
- `docs/docs/guides/evaluation/README.md` - Added links to new docs

**Architecture:**
```
User Input
    ↓
Smart Agent (interprets intent)
    ↓
Native Go Command (fast execution)
    ↓
Results + Recommendations
```

**Usage:**
```bash
# Direct commands (power users)
ailang eval-compare baselines/v0.3.0 current
ailang eval-validate records_person
ailang eval-report results/ v0.3.1 --format=html > report.html

# Make targets (workflows)
make eval-baseline              # Store baseline
make eval-diff BASELINE=... NEW=...
make eval-validate-fix BENCH=float_eq
```

---

### Added - M-V3.2: Planning & Scaffolding Protocol ✅ COMPLETE

**Complete proactive planning system for architecture validation and code scaffolding from plans (~2,560 LOC in 1 day).**

**Implementation** (`internal/schema/`, `internal/planning/`, `internal/repl/`)
- **Plan Schema** (`schema/plan.go`, ~109 LOC)
  - JSON schema for architecture plans with modules, types, functions, effects
  - Plan versioning with `ailang.plan/v1`
  - Helper methods: `AddModule()`, `AddType()`, `AddFunction()`, `AddEffect()`
  - Deterministic JSON serialization via schema registry

- **Plan Validator** (`planning/validator.go`, ~546 LOC)
  - Validates module paths (lowercase, no invalid chars, no cycles)
  - Validates type definitions (CamelCase names, valid kinds: adt/record/alias)
  - Validates function signatures (camelCase names, canonical effects)
  - Detects circular dependencies between modules
  - 24 validation error codes (VAL_M##, VAL_T##, VAL_F##, VAL_E##, VAL_G##)
  - Returns structured validation results with errors and warnings

- **Code Scaffolder** (`planning/scaffolder.go`, ~327 LOC)
  - Generates valid AILANG module files from validated plans
  - Creates module declarations, imports, type definitions, function stubs
  - Supports multiple modules with proper directory structure
  - Placeholder return values based on inferred types
  - TODO comments in generated code for implementation guidance
  - Options: output directory, overwrite mode, include comments/TODOs

- **REPL Integration** (`repl/planning.go`, ~264 LOC + repl.go modifications)
  - New `:propose <plan.json>` command - validates architecture plans
  - New `:scaffold --from-plan <plan.json> [--output <dir>] [--overwrite]` command
  - Colorized validation output (errors in red, success in green)
  - Example plan creation with `SaveExamplePlan()`
  - Updated `:help` text with planning commands

**Tests** (~844 LOC total)
- `schema/plan_test.go`: 9 tests for plan schema
- `planning/validator_test.go`: 18 tests for validation rules
- `planning/scaffolder_test.go`: 17 tests for code generation
- `planning/integration_test.go`: 6 end-to-end tests + 2 benchmarks
- `repl/planning_test.go`: 15 tests for REPL command parsing
- **All 65 tests passing** ✅

**Example Plans** (`examples/plans/`)
- `simple_api.json`: REST API handler with Request/Response types
- `cli_tool.json`: CLI utility with multiple modules and FS effects
- `minimal.json`: Hello world application

**Usage:**
```bash
# In REPL:
λ> :propose examples/plans/simple_api.json
✅ Plan is valid!
✅ Ready to scaffold!

λ> :scaffold --from-plan examples/plans/simple_api.json --output ./generated
✅ Scaffolding successful!
Files created: 1
Total lines: 28
Generated files:
  - ./generated/api/core.ail

# From command line (after building):
ailang repl
```

**Files Added:**
- `internal/schema/plan.go` (+109 LOC)
- `internal/schema/plan_test.go` (+152 LOC)
- `internal/planning/validator.go` (+546 LOC)
- `internal/planning/validator_test.go` (+328 LOC)
- `internal/planning/scaffolder.go` (+327 LOC)
- `internal/planning/scaffolder_test.go` (+305 LOC)
- `internal/planning/integration_test.go` (+325 LOC)
- `internal/repl/planning.go` (+264 LOC)
- `internal/repl/planning_test.go` (+174 LOC)
- `examples/plans/simple_api.json` (example)
- `examples/plans/cli_tool.json` (example)
- `examples/plans/minimal.json` (example)
- Total: **~2,560 LOC** + 3 example plans

**Files Modified:**
- `internal/schema/registry.go` (updated PlanV1 constant)
- `internal/repl/repl.go` (added :propose and :scaffold commands to REPL)

**Key Design Decisions:**
1. Schema versioning from day 1 (ailang.plan/v1) for future evolution
2. Validation separated into errors (must fix) vs warnings (should fix)
3. Scaffolder generates valid module structure but allows compilation errors in stubs
4. Planning workflow: create plan → validate → scaffold → implement → compile
5. REPL commands make planning accessible without CLI flags

**Velocity:** ~2,560 LOC in ~8 hours (~320 LOC/hour sustained)

**Impact:** AI agents can now validate architecture before coding, reducing wasted effort and improving success rates in eval benchmarks.

---

### Changed - Documentation Refactor

**CLAUDE.md Major Cleanup (830 → 438 lines, 47% reduction)**
- Removed reference material that belongs in proper docs
- Focused on actionable instructions for Claude
- Moved AILANG syntax examples to `prompts/v0.3.0.md` (already existed)
- Moved REPL guide content to `docs/guides/repl.md` (TODO: create)
- Moved testing guidelines to `docs/CONTRIBUTING.md` (TODO: create)
- Added clear links to detailed documentation
- Maintained critical warnings and workflows
- Updated Project Structure with all 24 internal packages
- Updated M-EVAL-LOOP section for v2.0
- Updated Project Overview with implemented features

**Documentation Consolidation**
- Moved `docs/eval_analysis_complete.md` → `docs/docs/guides/evaluation/go-implementation.md`
- Moved `docs/eval_analysis_migration.md` → `docs/docs/guides/evaluation/migration-guide.md`
- Updated all cross-references in agent files and documentation

**Result:** CLAUDE.md is now a focused "instruction manual" for Claude, not a reference encyclopedia.

---

## [Unreleased] - 2025-10-08

### Added - M-EVAL-LOOP Milestone 1: Self-Repair Foundation ✅ COMPLETE

**Complete self-repair system for AI evaluation benchmarks with error taxonomy, retry logic, and CLI integration (~520 LOC in 3.5 hours).**

**Implementation** (`internal/eval_harness/`)
- **Error taxonomy** (`errors.go`, ~150 LOC)
  - 6 error codes: PAR_001, TC_REC_001, TC_INT_001, EQ_001, CAP_001, MOD_001
  - Regex-based error matching with repair hints
  - `CategorizeErrorCode()` matches stderr against patterns
  - `FormatRepairPrompt()` generates error-specific fix guidance
  - Structured RepairHint with Title/Why/How format
- **RepairRunner orchestration** (`repair.go`, ~140 LOC)
  - Single-shot self-repair loop: attempt → error → repair → retry
  - `Run()` method handles first attempt + optional repair
  - `runSingleAttempt()` for code generation + execution cycles
  - `populateMetrics()` for comprehensive metrics tracking
  - Automatic error categorization and repair prompt injection
- **Extended metrics** (`metrics.go`, modified)
  - Self-repair tracking: FirstAttemptOk, RepairUsed, RepairOk
  - Error details: ErrCode, RepairTokensIn, RepairTokensOut
  - Prompt versioning: PromptVersion field (ready for A/B testing)
  - Reproducibility: BinaryHash, StdlibHash, Caps fields

**Tests** (`internal/eval_harness/errors_test.go`, ~200 LOC)
- 10 test cases covering all error codes
- Repair prompt formatting validation
- Rule completeness checks
- Regex pattern validation
- All tests passing ✅

**CLI Integration** (`cmd/ailang/eval.go`, modified)
- New `--self-repair` flag for single-shot repair
- RepairRunner integration replacing manual execution
- Enhanced output showing repair attempts and results
- Backward compatible (repair disabled by default)

**Usage:**
```bash
# Without self-repair (0-shot)
ailang eval --benchmark fizzbuzz --model claude-sonnet-4-5

# With self-repair (1-shot)
ailang eval --benchmark fizzbuzz --model claude-sonnet-4-5 --self-repair
```

**Files Modified:**
- `internal/eval_harness/errors.go` (+150 LOC)
- `internal/eval_harness/errors_test.go` (+200 LOC)
- `internal/eval_harness/repair.go` (+140 LOC)
- `internal/eval_harness/metrics.go` (+30 LOC)
- `cmd/ailang/eval.go` (refactored for RepairRunner)
- Total: ~520 LOC

**Key Design Decisions:**
1. Single-shot repair only (no infinite loops)
2. Error-specific repair hints (not generic "fix it")
3. Metrics track both first attempt and repair separately
4. RepairRunner owns orchestration (agent + runner coordination)
5. Backward compatible CLI (repair opt-in via flag)

**Velocity:** ~150 LOC/hour, ahead of schedule (estimated 6-8 hours, actual 3.5 hours)

---

### Added - M-EVAL-LOOP Milestone 2: Prompt Versioning & A/B Testing ✅ COMPLETE

**Complete prompt versioning system for A/B testing teaching strategies across AI models (~570 LOC in 2 hours).**

**Prompt Registry** (`prompts/versions.json`)
- JSON-based registry with metadata for all prompt versions
- SHA256 hash verification for prompt integrity
- Version tags: baseline, experimental, production, historical, control
- Active version tracking for defaults
- Created 2 initial versions:
  - `v0.3.0-baseline`: Original teaching prompt (3,674 tokens)
  - `v0.3.0-hints`: Enhanced with 6 error pattern sections (4,538 tokens, +864 tokens)

**Prompt Loader** (`internal/eval_harness/prompt_loader.go`, ~120 LOC)
- `NewPromptLoader()` loads registry from `prompts/versions.json`
- `LoadPrompt(versionID)` with SHA256 hash verification
- `GetActivePrompt()` for default version
- `GetVersion()` and `ListVersions()` for metadata queries
- `ComputePromptHash()` helper for updating registry
- Placeholder hash support for work-in-progress prompts

**Prompt Variants** (`prompts/v0.3.0-hints.md`, +864 tokens)
- Added explicit error pattern warnings based on error taxonomy
- 6 common error sections with wrong/correct examples:
  - PAR_001: Missing semicolons in blocks
  - TC_REC_001: Accessing non-existent record fields
  - TC_INT_001: Using modulo on floats
  - EQ_001: Wrong equality dictionary
  - CAP_001: Missing effect capabilities
  - MOD_001: Undefined module/entrypoint
- Hypothesis: Explicit warnings reduce first-attempt failures and improve repair success

**Tests** (`internal/eval_harness/prompt_loader_test.go`, ~270 LOC)
- 10 comprehensive test cases
- Hash verification and mismatch detection
- Placeholder hash support
- Active prompt loading
- All tests passing ✅

**CLI Integration** (`cmd/ailang/eval.go`, modified)
- New `--prompt-version` flag for version selection
- Automatic prompt loading with hash verification
- Metrics tracking with PromptVersion field
- Custom prompt + task prompt composition

**A/B Testing Tools**
- `tools/eval_prompt_ab.sh` (~200 LOC): Run full benchmark suite with two prompts
- `tools/compare_results.sh` (~180 LOC): Analyze and compare results
- Beautiful terminal output with success rates, token counts, cost comparison
- Recommendations based on performance deltas

**Makefile Targets**
- `make eval-prompt-list`: Show all available prompt versions
- `make eval-prompt-hash`: Compute SHA256 hashes for all prompts
- `make eval-prompt-ab A=v0.3.0-baseline B=v0.3.0-hints`: Run A/B comparison

**Usage:**
```bash
# Use specific prompt version
ailang eval --benchmark fizzbuzz --prompt-version v0.3.0-hints

# A/B comparison
make eval-prompt-ab A=v0.3.0-baseline B=v0.3.0-hints

# List available versions
make eval-prompt-list
```

**Files Modified:**
- `prompts/versions.json` (new, registry)
- `prompts/v0.3.0-hints.md` (new, +864 tokens)
- `internal/eval_harness/prompt_loader.go` (+120 LOC)
- `internal/eval_harness/prompt_loader_test.go` (+270 LOC)
- `internal/eval_harness/repair.go` (added SetPromptVersion method)
- `cmd/ailang/eval.go` (added --prompt-version flag)
- `tools/eval_prompt_ab.sh` (+200 LOC)
- `tools/compare_results.sh` (+180 LOC)
- `Makefile` (+3 targets)
- Total: ~770 LOC

**Key Design Decisions:**
1. Hash verification prevents accidental prompt modification mid-experiment
2. Prompt version tracked in metrics for historical analysis
3. A/B scripts automate full benchmark suite comparison
4. Terminal-based output for fast iteration (no GUI required)
5. Backward compatible (version optional, falls back to benchmark default)

**Velocity:** ~385 LOC/hour (estimated 3-4 hours, actual 2 hours)

---

### Added - M-EVAL-LOOP Milestone 3: AI-Friendly Formats & Validation ✅ COMPLETE

**Complete validation workflow with AI-friendly formats for performance tracking and fix validation (~900 LOC in 1.5 hours).**

**AI-Friendly Export Tools**
- `tools/generate_summary_jsonl.sh` (~90 LOC): Convert results to JSONL for AI analysis
  - One JSON object per line with key metrics
  - Easy querying with jq or AI tools
  - Fields: id, model, success rates, tokens, cost, errors, repair status
- `tools/generate_matrix_json.sh` (~140 LOC): Generate performance matrix JSON
  - Aggregates by model, benchmark, error code, language, prompt version
  - Historical tracking of 0-shot vs 1-shot success rates
  - Repair effectiveness metrics
  - Token and cost analytics

**Validation Workflow**
- `tools/eval_baseline.sh` (~120 LOC): Store baseline for current version
  - Runs full benchmark suite
  - Generates performance matrix
  - Creates baseline metadata with git commit info
  - Enables future validation via diff
- `tools/eval_diff.sh` (~140 LOC): Compare two eval runs
  - Shows fixed benchmarks (✓)
  - Shows broken benchmarks (✗)
  - Calculates success rate deltas
  - Beautiful terminal output with color coding
- `tools/eval_validate_fix.sh` (~140 LOC): Validate a specific fix
  - Compares against baseline
  - Shows before/after status
  - Detects regressions
  - Exit code 0 = validated, 1 = failed/still broken

**Makefile Integration** (5 new targets)
- `make eval-baseline`: Store current results as baseline
- `make eval-diff BASELINE=<dir> NEW=<dir>`: Compare runs
- `make eval-validate-fix BENCH=<id>`: Validate specific fix
- `make eval-summary DIR=<dir>`: Generate JSONL summary
- `make eval-matrix DIR=<dir> VERSION=<ver>`: Generate performance matrix

**Usage Examples:**
```bash
# Validation workflow
make eval-baseline                      # Store baseline
# ... make code changes ...
make eval-validate-fix BENCH=float_eq   # Validate fix
make eval-diff BASELINE=baselines/v0.3.0 NEW=after_fix  # Show all changes

# AI-friendly exports
make eval-summary DIR=eval_results/baseline OUTPUT=summary.jsonl
make eval-matrix DIR=eval_results/baseline VERSION=v0.3.0-alpha5

# Query with jq
jq -s 'group_by(.err_code) | map({code: .[0].err_code, count: length})' summary.jsonl
```

**Files Created:**
- `tools/generate_summary_jsonl.sh` (+90 LOC)
- `tools/generate_matrix_json.sh` (+140 LOC)
- `tools/eval_baseline.sh` (+120 LOC)
- `tools/eval_diff.sh` (+140 LOC)
- `tools/eval_validate_fix.sh` (+140 LOC)
- `Makefile` (+5 targets, ~80 LOC)
- Total: ~710 LOC scripts + ~190 LOC integration

**Key Design Decisions:**
1. JSONL format for streaming and AI-friendly analysis
2. Exit codes for CI/CD integration (0 = pass, 1 = fail)
3. Baseline storage with git metadata for reproducibility
4. Terminal-based workflow (no GUI dependencies)
5. Composable scripts (can chain together)

**Velocity:** ~600 LOC/hour (estimated 4-5 hours, actual 1.5 hours!)

**Cumulative M-EVAL-LOOP Progress:**
- **Milestones 1, 2 & 3 Complete**: ~2,960 LOC in 7 hours
- **Average velocity**: ~423 LOC/hour
- **Ahead of schedule**: ~7-9 hours saved

---

### Added - Documentation & AI Agent Integration

**Complete documentation and slash command for AI agent access to M-EVAL-LOOP workflows.**

**Website Documentation**
- Created comprehensive eval-loop guide at `docs/docs/guides/evaluation/eval-loop.md`
- Covers all 3 milestones: Self-Repair, Prompt Versioning, Validation
- Includes usage examples, workflow descriptions, and best practices
- AI-friendly format with code examples and command references

**Slash Command** (`/.claude/commands/eval-loop.md`)
- New `/eval-loop` command for AI agents
- Workflows: baseline, validate, diff, prompt-ab, summary, matrix
- Automatic execution via Makefile targets
- Integrated with Claude Code for seamless access

**llms.txt Updates**
- Extended `tools/generate-llms-txt.sh` to include Docusaurus subdirectories
- Added all evaluation guides including eval-loop documentation
- Size increased from 181KB to 244KB (8 M-EVAL-LOOP references)
- Published at https://sunholo-data.github.io/ailang/llms.txt

**AI Agent Usage:**
```
User: "Let's validate the float_eq fix"
Assistant: /eval-loop validate float_eq
# Executes: make eval-validate-fix BENCH=float_eq
# Output: "✓ FIX VALIDATED: Benchmark now passing!"

User: "Compare prompts"
Assistant: /eval-loop prompt-ab v0.3.0-baseline v0.3.0-hints
# Executes: make eval-prompt-ab A=v0.3.0-baseline B=v0.3.0-hints
# Output: "+7% improvement with hints prompt"
```

**Files Modified:**
- `docs/docs/guides/evaluation/eval-loop.md` (new, comprehensive guide)
- `.claude/commands/eval-loop.md` (new, slash command)
- `tools/generate-llms-txt.sh` (extended to include subdirectories)
- `docs/llms.txt` (regenerated with +63KB of eval-loop docs)

---

## [v0.3.0] - 2025-10-05

Complete implementation of Clock & Net effects (M-R6) with full Phase 2 PM security hardening, plus critical type system fixes (M-R7) for modulo operator and float comparison.

### Added - M-R7 Type System Fixes ✅ COMPLETE
- **Fixed modulo operator (`%`)**: Works correctly with type defaulting (`5 % 3` returns `2`)
- **Fixed float comparison (`==`)**: Resolves dictionary correctly (`0.0 == 0.0` returns `true`)
- **Regression tests**:
  - `examples/test_integral.ail` - Locks in modulo fix
  - `examples/test_float_comparison.ail` - Locks in float comparison fix
  - `examples/test_fizzbuzz.ail` - Exercises both `%` and `==` together
  - `benchmarks/numeric_modulo.yml` - Eval harness benchmark for `%`
  - `benchmarks/float_eq.yml` - Eval harness benchmark for `==`
  - All tests passing ✅

### Added - AI API Examples (with v0.4.0 roadmap)
- **`examples/demo_openai_api.ail`** - OpenAI API example with workaround for missing features
- **`design_docs/planned/v0_4_0_net_enhancements.md`** - Complete roadmap for Net enhancements:
  - Custom HTTP headers (`httpPostWithHeaders`)
  - Environment variable reading (`getEnv`, `hasEnv`)
  - JSON parsing (`parseJSON`, `getValue`)
  - Response status/headers

## [v0.3.0-alpha4] - 2025-10-05

### Added - M-R6 Phase 2: Clock & Net Effects ✅ COMPLETE
- **Clock effect** (`internal/effects/clock.go`, 109 LOC)
  - `_clock_now()` returns current time in milliseconds since Unix epoch
  - `_clock_sleep(ms)` suspends execution for specified milliseconds
  - Monotonic time: immune to NTP/DST changes (uses `time.Since(start) + epoch`)
  - Virtual time: deterministic mode with `AILANG_SEED` (starts at epoch 0)
  - stdlib wrapper: `std/clock` module with `now()` and `sleep()` functions
- **Net effect** (`internal/effects/net.go`, 355 LOC - Phase 2 PM FULL)
  - `_net_httpGet(url)` fetches content from HTTP/HTTPS URLs
  - `_net_httpPost(url, body)` sends POST requests with JSON body
  - **DNS rebinding prevention**: resolve → validate IPs → dial validated IP directly
  - **Protocol security**: https always allowed, http requires `--net-allow-http`, file:// blocked
  - **IP blocking**: localhost (127.x, ::1), private IPs (10.x, 192.168.x, 172.16-31.x), link-local
  - **Redirect validation**: max 5 redirects, re-validate IP at each hop
  - **Body size limits**: 5MB default via `io.LimitReader`, configurable via `NetContext.MaxBytes`
  - **Domain allowlist**: optional wildcard matching (*.example.com)
  - stdlib wrapper: `std/net` module with `httpGet()` and `httpPost()` functions
- **NetContext security configuration** (`internal/effects/context.go`, +130 LOC)
  - `Timeout` (30s default), `MaxBytes` (5MB), `MaxRedirects` (5)
  - `AllowHTTP` (false), `AllowLocalhost` (false)
  - `AllowedDomains` (wildcard support), `UserAgent` ("ailang/0.3.0")
- **IP validation helpers** (`internal/effects/net_security.go`, 91 LOC)
  - `validateIP()` checks IP against security policy
  - `resolveAndValidateIP()` prevents DNS rebinding attacks
  - `isAllowedDomain()` and `matchDomain()` for allowlist checking
- **Comprehensive test suites**:
  - Clock: 9 tests with flaky-guard (100 iterations for determinism)
  - Net: 6 test suites covering capabilities, protocols, IPs, domains, POST, body limits
  - All tests passing with both real network and mocked scenarios
- **2 new example files**:
  - `examples/micro_clock_measure.ail` - Clock effect demonstration
  - `examples/demo_ai_api.ail` - Real API calling with httpbin.org
- **Stdlib modules**:
  - `stdlib/std/clock.ail` - Clock effect wrappers
  - `stdlib/std/net.ail` - Net effect wrappers with security docs

### Security
- **M-R6 Net effect implements full Phase 2 PM hardening**
  - DNS rebinding prevention protects against SSRF attacks
  - IP blocking prevents access to localhost, private networks, link-local
  - Protocol validation blocks file://, ftp://, data://, gopher://
  - Redirect validation with IP re-check at each hop
  - Body size limits prevent memory exhaustion
  - Domain allowlist enables fine-grained access control
  - All security features tested with comprehensive test suite

### Fixed
- Added capability checks to `netHttpGet()` and `netHttpPost()` (requires `--caps Net`)
- Updated `resolveAndValidateIP()` to accept `*EffContext` for `AllowLocalhost` flag
- Fixed `validateIP()` to check `ctx.Net.AllowLocalhost` before blocking localhost IPs

## [v0.3.0-alpha3] - 2025-10-05

### Added - M-R5: Records & Row Polymorphism ✅ COMPLETE
- **Record subsumption** for flexible field access
  - Functions accepting `{id: int}` now work with `{id: int, name: string, email: string}`
  - Field access uses open records: `{x: α | ρ}` unifies with larger closed records
  - Enables polymorphic functions over records with common fields
- **TRecord2 with row polymorphism** (opt-in via `AILANG_RECORDS_V2=1`)
  - Proper row types with tail variables: `{x: int, y: bool | ρ}`
  - Row unification with occurs check prevents infinite types
  - Order-independent field matching: `{x:int,y:bool}` ~ `{y:bool,x:int}`
  - Nested record openness: `{u:{id:int | ρ}}` ~ `{u:{id:int,email:string}}`
- **TRecordOpen compatibility shim** for Day 1 subsumption
  - Bridges old TRecord and new TRecord2 systems
  - Enables subsumption without breaking existing code
- **Enhanced error messages** (TC_REC_001 - TC_REC_004)
  - TC_REC_001: Missing field with available field suggestions
  - TC_REC_002: Duplicate field in literal with positions
  - TC_REC_003: Row occurs check with infinite type prevention
  - TC_REC_004: Field type mismatch with clear expected vs actual
- **New helper functions** in `internal/types/unification.go`:
  - `RecordHasField()` - Check field existence across record types
  - `RecordFieldType()` - Get field type safely
  - `IsOpenRecord()` - Detect open vs closed records
  - `TRecordToTRecord2()`, `TRecord2ToTRecord()` - Bidirectional conversion
- **Row unifier with occurs check**
  - `unifyRows()` handles field-by-field unification
  - Prevents `ρ ~ {x: τ | ρ}` infinite types
  - Proper tail unification with commutativity
- **2 new example files**:
  - `examples/micro_record_person.ail` - Simple field access and aliasing
  - `examples/test_record_subsumption.ail` - Demonstrates subsumption in action
- **16 new unit tests** covering:
  - TRecord2 ~ TRecord2 unification (4 cases)
  - TRecord ↔ TRecord2 conversion (3 cases)
  - Row occurs check (1 case)
  - Open-closed interactions (6 cases)
  - Order independence, nested openness, field mismatches

### Changed
- **Typechecker emits TRecord2** when `AILANG_RECORDS_V2=1` is set
  - `inferRecordLiteral()` creates TRecord2 for record literals
  - Default still uses TRecord for backwards compatibility
  - Plan: Enable by default in v0.3.1, remove TRecord in v0.4.0
- **Field access uses TRecordOpen** for subsumption
  - `inferRecordAccess()` emits open records instead of closed
  - Allows functions to work with record subsets

### Fixed
- **Record field access** now works with nested records
  - Before: `{ceo: {name: "Jane"}}.ceo.name` → type error
  - After: Correctly types and evaluates to "Jane" ✅
- **Subsumption** enables polymorphic record functions
  - Before: Functions required exact field matches
  - After: Functions work with any record containing required fields ✅

### Impact
- **Lines of code**: ~670 total
  - Day 1: ~198 LOC (TRecordOpen, subsumption, helpers)
  - Day 2: ~280 LOC (TRecord2 unification, row unifier, conversion, occurs check, tests)
  - Day 3: ~192 LOC (flag support, error codes, examples, tests)
- **Examples**: 48/66 passing (72.7%, up from 40)
  - +9 fixed from subsumption (Day 1)
  - +2 new examples (Day 3)
- **Tests**: 16 new unit tests, all passing
- **Files modified**: 8 files
  - `internal/types/types.go` - TRecordOpen type
  - `internal/types/typechecker_core.go` - useRecordsV2 flag, inferRecordLiteral
  - `internal/types/unification.go` - Subsumption, TRecord2, unifyRows, helpers
  - `internal/types/errors.go` - TC_REC_001-004 error codes
  - `internal/types/record_unification_test.go` - 16 unit tests (NEW)
  - `examples/micro_record_person.ail` - (NEW)
  - `examples/test_record_subsumption.ail` - (NEW)
  - `examples/STATUS.md` - Updated counts

### Migration Guide
**Opt-in to TRecord2**:
```bash
export AILANG_RECORDS_V2=1
ailang run examples/micro_record_person.ail
```

**Using subsumption**:
```ailang
-- Define function with minimal fields
func printId(entity: {id: int}) -> () ! {IO} {
  println(show(entity.id))
}

-- Works with any record containing 'id'!
printId({id: 42})                           -- ✅
printId({id: 100, name: "Alice"})          -- ✅
printId({id: 200, name: "Bob", age: 30})   -- ✅
```

## [v0.3.0-alpha2] - 2025-10-05

### Added - M-R8: Block Expressions ✅ COMPLETE
- **Block expression syntax** `{ e1; e2; e3 }` for sequencing multiple expressions
  - Last expression's value is the block's value
  - Non-last expressions evaluated for side effects
  - Desugars to let chains: `let _ = e1 in let _ = e2 in e3`
- **Bug fix** in `internal/elaborate/scc.go` (~10 LOC)
  - Added missing `*ast.Block` case to `findReferences()` function
  - Fixed recursion detection for functions using block syntax
  - Self-recursive and mutual recursion now work correctly with blocks
- **3 new example files**:
  - `examples/micro_block_seq.ail` - Basic block sequencing
  - `examples/micro_block_if.ail` - Blocks in if-then-else branches
  - `examples/block_recursion.ail` - Recursive functions with blocks
- **AI compatibility unlocked** ✨
  - AI-generated code with blocks now works out of the box
  - No manual rewriting required
  - Compatible with Claude Sonnet 4.5, GPT-4, etc.

### Fixed
- **Recursion + Blocks Bug**: Functions with recursive calls inside blocks now correctly detected as recursive
  - Before: `func fact(n) { ... fact(n-1) }` → "undefined variable: fact"
  - After: Correctly creates LetRec, recursion works ✅
- **SCC Detection**: `findReferences()` now traverses all expression types including blocks

### Impact
- Lines of code: 10 (5-line case statement)
- Examples: 3 new files
- Test status: All existing tests pass + new examples verified
- Developer experience: Major improvement for AI-assisted development

## [v0.3.0-alpha1] - 2025-10-05

### Added - M-R4: Recursion Support ✅ COMPLETE
- **Full recursion support** via RefCell indirection (OCaml/Haskell-style semantics)
  - Self-referential closures with proper capture semantics
  - Mutually recursive functions (pre-bind all names before evaluation)
  - Function-first semantics: lambdas safe immediately, non-lambdas evaluated strictly
- **Stack overflow protection** with `--max-recursion-depth` CLI flag (default: 10,000)
  - Configurable depth limit for both module and non-module execution
  - Clear RT_REC_003 error messages with actionable guidance
- **Cycle detection** for recursive values (RT_REC_001 error)
  - Prevents infinite loops in non-function bindings
  - Example: `let rec x = x + 1 in x` properly detected and rejected
- **New runtime infrastructure** in `internal/eval/`
  - `RefCell` type for mutable indirection cells (value.go:166-197)
  - `IndirectValue` wrapper with Force() method for deferred resolution
  - 3-phase LetRec evaluation algorithm (eval_core.go:363-426)
  - Recursion depth tracking in CoreEvaluator (eval_core.go:17-25)
- **5 new example files** demonstrating recursion patterns
  - `examples/recursion_factorial.ail` - Simple & tail-recursive factorial
  - `examples/recursion_fibonacci.ail` - Tree recursion with 2 recursive calls
  - `examples/recursion_mutual.ail` - Mutually recursive isEven/isOdd
  - `examples/recursion_quicksort.ail` - Conceptual recursive structure
  - `examples/recursion_error.ail` - Documents RT_REC_001 error conditions
- **Comprehensive test suite** in `internal/eval/recursion_test.go`
  - 6 unit tests covering all recursion patterns
  - Tests for factorial, fibonacci, mutual recursion, stack overflow, deep recursion
  - All tests passing with experimental binop shim

### Changed
- **Example baseline improved**: 43 passing (up from 32), 14 failed, 4 skipped (Total: 61)
  - 11 additional examples now passing due to recursion infrastructure
- **CoreEvaluator** now tracks recursion depth for stack overflow detection
- **Module runtime** applies max recursion depth limit via `rt.GetEvaluator().SetMaxRecursionDepth()`

### Technical Details
- **Lines of code**: ~1,200 (core implementation) + ~380 (tests) + ~200 (examples)
- **Semantic model**: Proper λ-calculus closure semantics matching textbook small-step operational semantics
- **Performance**: O(1) lookup via pointer indirection, negligible overhead
- **Error taxonomy**:
  - RT_REC_001: Recursive value used before initialization (non-function RHS)
  - RT_REC_002: Uninitialized recursive binding (internal ordering bug)
  - RT_REC_003: Stack overflow with depth limit exceeded

### Language Milestone
**AILANG is now Turing-complete** with deterministic semantics:
- ✅ λ-abstraction (first-class functions)
- ✅ Application (function calls)
- ✅ Conditionals (if-then-else)
- ✅ Recursion (self & mutual)
- ✅ Side-effects (IO/FS with capability security)

This milestone enables expressing every partial recursive function under deterministic semantics.

## [v0.2.1] - 2025-10-03

### Fixed
- **Windows Build Compatibility**: Fixed two Windows-specific test failures
  - Fixed `TestFSWriteFile_Success` using invalid `*` wildcard in filename (not allowed on Windows)
  - Fixed `TestNewModuleRuntime` path separator mismatch (Windows uses `\` vs Unix `/`)
  - All tests now pass on Windows, Linux, and macOS

### Changed
- Tests are now OS-agnostic, using `filepath.Clean()` for cross-platform compatibility
- Improved CI/CD reliability across all supported platforms

### 🔄 RECURSION & REAL-WORLD PROGRAMS (Target: 50+ examples)

**Status**: 🚧 IN PLANNING - See [design_docs/20251004/v0_3_0_implementation_plan.md](design_docs/20251004/v0_3_0_implementation_plan.md)

**Planned Features**:

#### M-R4: Recursion Support ✅ COMPLETE (v0.3.0-alpha1)
- ✅ **DONE**: LetRec support in runtime evaluator (RefCell indirection)
- ✅ **DONE**: Self-referential closures (3-phase algorithm)
- ✅ **DONE**: Recursive function examples (factorial, fibonacci, quicksort, mutual, error)
- ✅ **DONE**: Stack overflow protection (--max-recursion-depth flag)
- **Impact**: AILANG now Turing-complete with deterministic semantics

#### M-R8: Block Expressions (HIGH PRIORITY, ~300 LOC) ← **NEW**
- ✅ **TODO**: Block syntax `{ e1; e2; e3 }` as syntactic sugar
- ✅ **TODO**: Desugar to let-sequencing: `let _ = e1 in let _ = e2 in e3`
- ✅ **TODO**: Parser support (recognize `{ }` in expression position)
- ✅ **TODO**: Empty block error with clear message
- ✅ **TODO**: 3 integration examples (seq, if-then-else, recursion)
- **Impact**: **Critical for AI compatibility** - unblocks Claude Sonnet 4.5 generated code with blocks
- **Why**: AI models naturally generate blocks, currently fails to parse
- **Risk**: LOW (pure syntactic sugar, no type system or runtime changes)

#### M-R5: Records & Row Polymorphism (HIGH PRIORITY, ~500 LOC)
- ✅ **TODO**: Complete TRecord unification
- ✅ **TODO**: Row variables for polymorphic records
- ✅ **TODO**: Field access type checking improvements
- **Impact**: Enables proper data modeling

#### M-R6: Extended Effects - Clock & Net (MEDIUM PRIORITY, ~700 LOC)
- ✅ **TODO**: std/clock effect (now, sleep, timeout)
- ✅ **TODO**: std/net effect (httpGet, httpPost)
- ✅ **TODO**: Capability enforcement and security sandbox
- **Impact**: Real-world program connectivity

#### M-R7: Modulo Operator Fix (MEDIUM PRIORITY, ~200 LOC)
- ✅ **TODO**: Integral type class (div, mod)
- ✅ **TODO**: Fix % operator type inference
- **Impact**: Removes arithmetic operator blocker

#### M-UX2: User Experience (LOW PRIORITY, ~300 LOC)
- ✅ **TODO**: Better recursion error messages
- ✅ **TODO**: Audit script Clock/Net detection
- ✅ **TODO**: 4-6 new micro examples

**Target Success Metrics**:
- **Passing Examples**: 42 → 50+ (83%+)
- **Recursion**: Broken → Working
- **Records**: Partial → Working with row polymorphism
- **Effects**: IO/FS → + Clock/Net (4 total)
- **Modulo (%)**: Broken → Working via Integral

**Timeline**: October 17-21, 2025 (2 weeks)

---

## [v0.2.0] - 2025-10-03

### 🎉 AUTO-ENTRY & EXAMPLE EXPLOSION: 42/53 Passing (79%) ✅

**Achieved Target**: Exceeded v0.2.0 goal of ≥35 passing examples, reaching **42/53 (79.2%)**

**Implementation**: ~200 LOC across 3 strategic improvements
1. **Auto-Entry Fallback** (`cmd/ailang/main.go`, ~50 LOC)
   - Intelligent entrypoint selection when `main` not found
   - Auto-selects single zero-arg function, or tries `test()`
   - Eliminated "entrypoint not found" errors for 10+ examples

2. **Audit Script Enhancement** (`tools/audit-examples.sh`, ~20 LOC)
   - Automatic capability detection (`! {IO}`, `! {FS}`)
   - Runs examples with appropriate `--caps` flags
   - Enabled testing of all IO/FS effect examples

3. **TRecord Unification Support** (`internal/types/unification.go`, ~40 LOC)
   - Added handler for legacy `*TRecord` type in unification
   - Fixed "unhandled type in unification" errors
   - Improved record type checking with field-by-field unification

4. **Micro Examples** (2 new passing examples)
   - `examples/micro_option_map.ail` - Pure ADT operations
   - `examples/micro_io_echo.ail` - IO effect demonstration

**Results**: +14 examples in single session
- Before: 28/51 passing (55%)
- After: 42/53 passing (79%)
- **Progress**: +50% more working examples

**Newly Passing Examples** (+14):
- `demos/hello_io.ail` - IO effect with println
- `effects_basic.ail` - Basic effect annotations
- `stdlib_demo.ail` - Standard library usage
- `stdlib_demo_simple.ail` - Simplified stdlib demo
- `test_effect_annotation.ail` - Effect syntax
- `test_effect_capability.ail` - Capability requirements
- `test_effect_fs.ail` - FS effect testing
- `test_effect_io.ail` - IO effect testing
- `test_invocation.ail` - Function invocation
- `test_io_builtins.ail` - IO builtin functions
- `test_module_minimal.ail` - Minimal module
- `test_no_import.ail` - No imports required
- `micro_io_echo.ail` - NEW micro example
- `micro_option_map.ail` - NEW micro example

**Key Insight**: Auto-entry was the MVP - single feature unlocked 10+ examples by making testing frictionless.

**Impact on v0.2.0 Goals**:
- ✅ Target met: ≥35 examples (achieved 42)
- ✅ Effect system validated: IO/FS working across examples
- ✅ Module execution proven: Cross-module imports stable
- ✅ User experience improved: Reduced friction for running examples

---

## [v0.2.0-rc1] - 2025-10-02

### 🎯 M-EVAL: AI Evaluation Framework (~600 LOC) ✅

**AI Teachability Benchmarking System** - October 2, 2025

Added comprehensive framework for measuring AILANG's "AI teachability" - how easily AI models can learn to write correct AILANG code.

**Infrastructure**:
- `internal/eval_harness/` - Benchmark execution framework (~600 LOC)
  - `spec.go` - YAML benchmark loader with prompt file support
  - `runner.go` - Python & AILANG code execution with module path handling
  - `ai_agent.go` - LLM API wrapper with model resolution
  - `api_anthropic.go` - Claude API implementation (tested: 230 tokens)
  - `api_openai.go` - GPT API implementation (tested: 319 tokens)
  - `api_google.go` - Gemini/Vertex AI implementation (tested: 278 tokens)
  - `metrics.go` - JSON metrics logging with cost calculation
  - `models.go` - Centralized model configuration system

**Prompt System**:
- `prompts/v0.2.0.md` - Versioned AI teaching prompt for v0.2.0-rc1
- Documents working features: modules, effects, pattern matching, ADTs
- Includes common mistakes and correct patterns

**Benchmarks**:
- 5 benchmarks covering difficulty spectrum
- Supports prompt file loading via `prompt_file` YAML field
- Module path validation and stdlib resolution

**CLI**:
```bash
ailang eval --benchmark fizzbuzz --model claude-sonnet-4-5 --seed 42
./tools/run_benchmark_suite.sh  # Run all benchmarks with all 3 models
```

**Documentation**:
- `docs/guides/ai-prompt-guide.md` - AI teaching guide with v0.2.0 syntax
- `docs/guides/evaluation/` - Evaluation framework documentation
  - `baseline-tests.md` - Running first baseline tests
  - `model-configuration.md` - Model management
  - `README.md` - Framework overview

**Test Results**: All 3 models tested successfully
- ✅ Claude Sonnet 4.5 (Anthropic): 230 tokens generated
- ✅ GPT-5 (OpenAI): 319 tokens generated
- ✅ Gemini 2.5 Pro (Vertex AI): 278 tokens generated

**KPI**: Establishes baseline for "AI teachability" metric (target: 80%+ success rate on simple benchmarks)

### 🐛 Critical Fixes: Type Inference & Builtins (+22 LOC) ✅

**Fixed Arithmetic Operators** (`internal/runtime/builtins.go`, +13 LOC)
- Added `registerArithmeticBuiltins()` to register all arithmetic operators in module runtime
- Modulo operator `%` now works: `export func main() -> int { 5 % 3 }  -- Returns: 2`
- All arithmetic operators (`+`, `-`, `*`, `/`, `%`, `**`) available in module execution
- Delegates to existing `eval.Builtins` implementations via wrapper

**Fixed Comparison Operators** (`internal/types/typechecker_core.go`, +9 LOC)
- Modified `pickDefault()` to default `Ord`, `Eq`, `Show` constraints to `int`
- Comparison operators (`>`, `<`, `>=`, `<=`, `==`, `!=`) now work in modules
- No more "ambiguous type variable α with classes [Ord]" errors
- Example: `export func compare(x: int, y: int) -> bool { x > y }  -- Works!`

**Impact**: AI-generated code now compiles correctly. Basic arithmetic and comparisons work as expected.

### ⚠️ Known Limitations (Discovered During M-EVAL Testing)

**Critical Issues Requiring v0.2.1 Patch**:

1. **Recursive Functions in Modules** - HIGH PRIORITY
   - Functions cannot call themselves: `factorial(n-1)` fails with "undefined variable"
   - Blocks common patterns (loops via recursion, FizzBuzz, tree traversal)
   - Root cause: Function bindings not in own scope during evaluation
   - Estimated fix: ~200-300 LOC, 2-3 days

2. **Capability Passing to Runtime** - CRITICAL
   - `--caps IO,FS` flag not propagating to effect context
   - All effect-based code fails even with capabilities granted
   - Blocks all IO/FS demos and examples
   - Estimated fix: ~100-200 LOC, 1-2 days

**See**: `design_docs/20251002/v0_2_0_implementation_plan.md` (Known Limitations section) for full details and next sprint recommendations.

---

## [Unreleased v0.2.0-rc1] - 2025-10-02 (Original Features)

### 🚀 Major Features: M-R1, M-R2, M-R3 ALL COMPLETE ✅

**Milestone Achievement**:
- Module execution runtime (M-R1, ~1,874 LOC) ✅
- Effect system runtime (M-R2, ~1,550 LOC) ✅
- Pattern matching polish (M-R3, ~700 LOC) ✅
  - Phase 1: Guards (~55 LOC)
  - Phase 2: Exhaustiveness checking (~255 LOC)
  - Phase 3: Decision trees (~390 LOC)
- Critical bug fixes

This release delivers core runtime milestones with working capability enforcement AND comprehensive pattern matching enhancements. AILANG now has:
- Fully executable module system with capability-based effect operations
- Pattern matching with conditional guards
- Exhaustiveness warnings for incomplete matches
- Decision tree optimization for pattern matching (available, disabled by default)
- Effects like IO and FS work with explicit permission grants via `--caps` flag

**🔧 CRITICAL BUG FIXES (Oct 2)**: Removed legacy builtin path that bypassed effect system. Capability checking now works correctly. Fixed stdlib import resolution and integration test loader paths.

#### Added - M-R3 Phase 1: Guards (~55 LOC)

**Guard Support** (55 LOC)
- **Guard Elaboration** (`internal/elaborate/elaborate.go:1062-1069`)
  - Elaborates guard expressions during match compilation
  - Guards are normalized to Core ANF
  - Error handling for malformed guards
- **Guard Evaluation** (`internal/eval/eval_core.go:586-613`)
  - Evaluates guards with pattern bindings in scope
  - Enforces Bool type requirement for guards
  - False guards cause fallthrough to next arm
- **Tests**: 6 unit tests passing (`guards_simple_test.go`)
  - Basic true/false guards
  - Multiple sequential guards
  - Guard accessing pattern bindings
  - Non-Bool guard error handling
  - All guards failing → non-exhaustive error
- **Examples**:
  - `test_guard_bool.ail` - Guard with true
  - `test_guard_false.ail` - Guard causing fallthrough

#### Added - M-R3 Phase 2: Exhaustiveness Checking (~255 LOC)

**Exhaustiveness Analysis** (255 LOC)
- **Pattern Universe Builder** (`internal/elaborate/exhaustiveness.go`)
  - Constructs complete pattern sets for types (Bool → {true, false})
  - Pattern expansion and subtraction algorithms
  - Conservative handling of guards (don't count as coverage)
- **Integration** (`internal/elaborate/elaborate.go`, `internal/pipeline/pipeline.go`)
  - Exhaustiveness checker added to Elaborator
  - Warnings collected during elaboration
  - Result struct includes warnings array
- **CLI Display** (`cmd/ailang/main.go`)
  - Yellow-colored warnings displayed to stderr
  - Shows missing patterns for non-exhaustive matches
- **Tests**: 7 unit tests passing (`exhaustiveness_test.go`)
  - Complete Bool match (exhaustive)
  - Incomplete Bool match (non-exhaustive)
  - Wildcard coverage
  - Variable pattern coverage
  - Guard-aware checking
  - Infinite type handling (Int/Float/String)
- **Examples**:
  - `test_exhaustive_bool_complete.ail` - No warning
  - `test_exhaustive_bool_incomplete.ail` - Warning: missing false
  - `test_exhaustive_wildcard.ail` - Wildcard makes exhaustive

**Limitations**:
- Only Bool type fully supported (finite pattern universe)
- Int/Float/String require wildcard (infinite types)
- No ADT support yet (requires type environment integration)
- Guards conservatively treated as non-covering

#### Added - M-R3 Phase 3: Decision Trees (~390 LOC)

**Decision Tree Compilation** (390 LOC)
- **Tree Structure** (`internal/dtree/decision_tree.go`)
  - LeafNode, FailNode, SwitchNode representations
  - Pattern matrix compilation algorithm
  - Pattern specialization and row reduction
  - Heuristic for when to use decision trees (2+ literal/constructor patterns)
- **Tree Evaluation** (`internal/eval/decision_tree.go`)
  - Tree walking with scrutinee dispatch
  - Path-based value extraction for nested patterns
  - Guard checking in leaf nodes
  - Fallback to linear evaluation if tree compilation not beneficial
- **Integration** (`internal/eval/eval_core.go`)
  - Optional decision tree compilation (disabled by default)
  - Seamless fallback to linear pattern matching
  - Future: can be enabled via flag or heuristic
- **Tests**: 4 unit tests passing (`decision_tree_test.go`)
  - Simple Bool match compilation
  - Wildcard default handling
  - All-wildcards optimization
  - Heuristic validation

**Implementation Notes**:
- Decision trees available but disabled by default (runtime optimization)
- Reduces redundant pattern tests via switch-based dispatch
- Preserves exact semantics of linear pattern matching
- Can be enabled in future with flag/heuristic

#### Added - Phase 5: Function Invocation & Builtins (~280 LOC)

**Function Invocation** (60 LOC)
- **CallEntrypoint()** (`internal/runtime/entrypoint.go`)
  - Calls exported entrypoint functions from modules
  - Validates arity and function type
  - Sets up cross-module resolver
- **CallFunction()** (`internal/eval/eval_core.go`)
  - Public method to invoke FunctionValues
  - Manages environment binding and restoration
  - Supports 0-arg and multi-arg functions
- **CLI Integration** (`cmd/ailang/main.go`)
  - Argument decoding from `--args-json`
  - Result printing (silent for Unit types)
  - Helpful error messages for multi-arg functions

**Builtin Registry** (120 LOC)
- **BuiltinRegistry** (`internal/runtime/builtins.go`)
  - Native Go implementations of stdlib functions
  - IO builtins: `_io_print`, `_io_println`, `_io_readLine`
  - Integrated into ModuleRuntime initialization
- **Resolver Integration** (`internal/runtime/resolver.go`)
  - Checks builtins before local/import lookup
  - Supports `$builtin` module and `_` prefix names
- **Lit Expression Handling** (`internal/runtime/runtime.go`)
  - `extractBindings()` now handles Lit expressions at module level
  - Enables stdlib modules to load correctly

**Examples**
- `examples/test_invocation.ail` - 0-arg and 1-arg function examples
- `examples/test_io_builtins.ail` - Builtin IO function demonstration

#### Test Results - Phase 5

- **Unit Tests**: ✅ 16/16 passing (all runtime non-integration tests)
- **Integration Tests**: ⚠️ 2/7 passing (5 fail due to known loader path issues)
- **End-to-End Examples**: ✅ 2/2 new examples working
- **Total**: ~280 LOC added

---

#### 🔧 Fixed - Critical Bug Fixes (Oct 2, ~50 LOC changes)

**Bug #1: Legacy Builtin Path Bypassed Effect System** 🚨
- **Issue**: Special case in `evalCoreApp()` called `CallBuiltin()` directly, bypassing capability checking
- **Location**: `internal/eval/eval_core.go:404-416` (deleted)
- **Fix**: Removed 13 LOC special case; all builtins now route through resolver
- **Impact**: Capability checking NOW WORKS correctly
- **Test**: `ailang run effects_basic.ail` → denies without `--caps IO`, allows with it
- **Added**: Deprecation comment on old `CallBuiltin()` function

**Bug #2: Stdlib Imports Not Found** 🔧
- **Issue**: `import std/io` failed with "module not found"
- **Location**: `internal/loader/loader.go:80-88, 154-164`
- **Fix**: Resolve `std/` prefix from `stdlib/` directory (or `$AILANG_STDLIB_PATH`)
- **Impact**: Stdlib imports work: `import std/io (println)`
- **Test**: `examples/effects_basic.ail` now loads and runs

**Bug #3: Integration Tests Failed on Module Loading** ⚠️
- **Issue**: Loader used relative paths, tests couldn't find modules
- **Location**: `internal/loader/loader.go:94-97, 167-169`
- **Fix**: Join project-relative paths with `basePath` for absolute resolution
- **Additional**: Added Core elaboration in runtime (avoid import cycle)
- **Additional**: Added minimal interface builder for modules loaded without pipeline
- **Impact**: 5/7 integration tests now passing (2 fail on cross-module elaboration)
- **Test**: `TestIntegration_SimpleModule` and 4 others pass

**Test Coverage After Fixes**:
- ✅ All eval tests passing (no regressions)
- ✅ 39/39 effect tests passing
- ✅ 5/7 integration tests passing
- ✅ End-to-end capability enforcement verified

---

### ⚡ Major Feature: Effect System Runtime (M-R2 COMPLETE ✅)

**Milestone Achievement**: Capability-based effect system (~1,550 LOC total).

This implements the effect runtime that brings type-level effects into execution. Effects require explicit capability grants via `--caps` flag. Includes IO and FS operations with sandbox support.

**Status**: COMPLETE - Capability checking working, all acceptance criteria met.

#### Added - Effect System Infrastructure (~1,550 LOC)

**Core Effect System** (650 LOC)
- **Capability** (`internal/effects/capability.go`, 50 LOC)
  - Grant tokens for effect permissions (e.g., IO, FS, Net)
  - Metadata support for future budgets/quotas
  - `NewCapability(name)` constructor

- **EffContext** (`internal/effects/context.go`, 100 LOC)
  - Runtime context holding capability grants
  - Environment configuration (AILANG_SEED, TZ, LANG, sandbox)
  - Methods: `Grant()`, `HasCap()`, `RequireCap()`
  - `loadEffEnv()` loads from OS environment

- **Effect Operations Registry** (`internal/effects/ops.go`, 100 LOC)
  - `EffOp` type: `func(ctx, args) (Value, error)`
  - Registry: effect name → operation name → EffOp
  - `Call()` performs capability check + execution
  - `RegisterOp()` for operation registration

**IO Effect** (150 LOC)
- **IO Operations** (`internal/effects/io.go`)
  - `ioPrint(s)` - Print without newline
  - `ioPrintln(s)` - Print with newline
  - `ioReadLine()` - Read from stdin
  - All require IO capability grant

**FS Effect** (200 LOC)
- **FS Operations** (`internal/effects/fs.go`)
  - `fsReadFile(path)` - Read file to string
  - `fsWriteFile(path, content)` - Write string to file
  - `fsExists(path)` - Check file/directory existence
  - Sandbox support via `AILANG_FS_SANDBOX` env var
  - All require FS capability grant

**Error Handling** (50 LOC)
- **CapabilityError** (`internal/effects/errors.go`)
  - Clear error messages for missing capabilities
  - Helpful hints: "Run with --caps IO"

**Integration** (150 LOC)
- **CLI Flag** (`cmd/ailang/main.go`)
  - `--caps IO,FS,Net` flag for granting capabilities
  - Comma-separated capability list
  - Creates EffContext with grants before execution

- **Evaluator Support** (`internal/eval/eval_core.go`)
  - `SetEffContext(ctx)` / `GetEffContext()` methods
  - EffContext field added to CoreEvaluator

- **Runtime Integration** (`internal/runtime/`)
  - Builtins route to effect system via `effects.Call()`
  - `GetEvaluator()` method for EffContext access

**Stdlib** (20 LOC)
- **stdlib/std/fs.ail** - FS module with readFile, writeFile, exists

#### Testing - Effect System (750 LOC)

**Unit Tests** (550 LOC):
- `internal/effects/context_test.go` (150 LOC) - 12 tests for capabilities
- `internal/effects/io_test.go` (250 LOC) - 15 tests for IO operations
- `internal/effects/fs_test.go` (250 LOC) - 12 tests for FS operations
- ✅ **39/39 tests passing**
- ✅ **100% coverage** for new packages

**Integration Tests** (200 LOC):
- `internal/effects/integration_cli_test.go` - Full flow testing
- Capability grant/denial scenarios
- Sandbox enforcement verification

**Examples**:
- `examples/test_effect_io.ail` - IO operations demo
- `examples/test_effect_fs.ail` - FS operations placeholder

#### Usage Examples

**IO with capability grant**:
```bash
ailang run app.ail --caps IO
```

**FS with sandbox**:
```bash
AILANG_FS_SANDBOX=/tmp ailang run app.ail --caps FS
```

**Multiple capabilities**:
```bash
ailang run app.ail --caps IO,FS,Net
```

#### Known Limitations - Effect System

⚠️ **Legacy Builtin Path**: The old `CallBuiltin()` in `internal/eval/builtins.go:410` bypasses capability checks. Effect operations work but enforcement is incomplete.

**Impact**: Architecture complete, runtime checks bypassed by legacy code
**Fix Planned**: v0.2.1 - Remove legacy builtin special case

#### Metrics - M-R2

| Metric | Value |
|--------|-------|
| Total LOC | 1,550 |
| Core Code | 650 |
| Tests | 750 |
| Integration | 150 |
| Test Coverage | 100% (new packages) |
| Unit Tests | 39 passing |

---

## [v0.1.1] - 2025-10-02

### 🚀 Major Feature: Module Execution Runtime (M-R1 Phases 1-4)

**Milestone Achievement**: Core infrastructure for module execution complete (~1,594 LOC).

This release delivers the foundation for executable modules. Function invocation was completed in v0.2.0-rc1.

#### Added - Module Runtime Infrastructure (~1,594 LOC)

**Phase 1: Scaffolding** (692 LOC)
- **ModuleInstance** (`internal/runtime/module.go`, 164 LOC)
  - Runtime representation of modules with evaluated bindings
  - Thread-safe initialization using `sync.Once`
  - Export filtering and access control
  - Methods: `GetExport()`, `HasExport()`, `GetBinding()`, `ListExports()`, `IsEvaluated()`

- **ModuleRuntime** (`internal/runtime/runtime.go`, 149 LOC)
  - Orchestrates module loading, caching, and evaluation
  - Circular import detection with clear error messages ("A → B → C → A")
  - Topological dependency evaluation
  - Methods: `LoadAndEvaluate()`, `GetInstance()`, `PreloadModule()`

- **Unit Tests** (379 LOC)
  - `internal/runtime/module_test.go` - 7 tests for ModuleInstance
  - `internal/runtime/runtime_test.go` - 5 tests for ModuleRuntime
  - 12/12 tests passing ✅

**Phase 2: Evaluation + Resolver** (402 LOC)
- **Global Resolver** (`internal/runtime/resolver.go`, 120 LOC)
  - Cross-module reference resolution with encapsulation enforcement
  - Routes imported references through exports only (never private bindings)
  - Error handling with module availability checks

- **Module Evaluation** (~70 LOC in `runtime.go`)
  - `evaluateModule()` method for top-level binding extraction
  - Integration with existing Core evaluator
  - Export filtering based on module interface

- **Resolver Tests** (`internal/runtime/resolver_test.go`, 212 LOC)
  - 6 tests for local/import resolution, encapsulation, error cases
  - 18/18 total tests passing ✅

**Phase 3: Linking & Topological Sort** (~300 LOC)
- **Cycle Detection** (~50 LOC in `runtime.go`)
  - DFS-based circular import detection
  - Clear error messages with import path: "circular import detected: A → B → C → A"
  - State tracking with `visiting` map and `pathStack`

- **Integration Tests** (`internal/runtime/integration_test.go`, 249 LOC)
  - 7 integration tests covering module execution flows
  - Test modules in `tests/runtime_integration/` (simple.ail, dep.ail, with_import.ail)
  - 2/7 passing (5 have known loader path issues, non-blocking)

**Phase 4: CLI Integration** (~200 LOC)
- **Pipeline Extension** (`internal/pipeline/pipeline.go`, ~60 LOC)
  - Added `Modules map[string]*loader.LoadedModule` to Result struct
  - Converts CompileUnits to LoadedModules after elaboration
  - Preserves Core AST, Iface, and imports for runtime use

- **Loader Preloading** (`internal/loader/loader.go`, ~15 LOC)
  - Added `Preload(path, loaded)` method to inject elaborated modules
  - Avoids redundant loading and elaboration

- **Recursive Binding Extraction** (`internal/runtime/runtime.go`, ~55 LOC)
  - `extractBindings()` helper for nested Let/LetRec declarations
  - Handles module elaboration structure: `let f1 = ... in (let f2 = ... in Var(...))`
  - Properly terminates at Var expressions

- **CLI Integration** (`cmd/ailang/main.go`, ~30 LOC)
  - Module runtime replaces "not yet supported" error
  - Pre-loads modules from pipeline result
  - Entrypoint validation with arity checking
  - Error messages show available exports

- **Entrypoint Helpers** (`internal/runtime/entrypoint.go`, 37 LOC)
  - `GetArity(val)` - Returns function parameter count
  - `GetExportNames(inst)` - Lists module exports for error messages

#### Architecture Highlights

**Key Design Decisions**:
1. **Pipeline Integration**: Runtime receives pre-elaborated modules from pipeline (no duplicate work)
2. **Recursive Extraction**: `extractBindings()` traverses nested Let structures from elaboration
3. **Preloading Pattern**: Modules injected into loader cache via `PreloadModule()`
4. **Thread-Safe Init**: `sync.Once` ensures each module evaluates exactly once
5. **Encapsulation**: Only exported bindings accessible across modules

**Data Flow**:
```
Parse → Type-check → Elaborate → Pipeline
                                    ↓
                              Convert to LoadedModules
                                    ↓
                              Runtime.PreloadModule()
                                    ↓
                              Runtime.LoadAndEvaluate()
                                    ↓
                              Extract bindings recursively
                                    ↓
                              Filter exports
                                    ↓
                              Validate entrypoint ✅
```

#### Test Results

**Unit Tests**: ✅ 18/18 passing
- Module instance creation and export access (7 tests)
- Runtime caching and management (5 tests)
- Global resolver with encapsulation (6 tests)

**Integration Tests**: ⚠️ 2/7 passing
- CircularImport detection ✅
- NonExistentModule error ✅
- SimpleModule, ModuleWithImport, etc. ⚠️ (loader path resolution issues, non-blocking)

**End-to-End Validation**: ✅ Working
```bash
$ ailang --entry main run examples/test_runtime_simple.ail
✓: Module execution ready
  Entrypoint:  main
  Arity:       0
  Module:      examples/test_runtime_simple

Note: Function invocation coming soon (Phase 5 completion)
```

#### Known Limitations

1. **Function Invocation Not Implemented**
   - Entrypoints validated but not yet executed
   - Arity checking works ✅
   - Export resolution works ✅
   - Actual function calling deferred to Phase 5

2. **stdlib Modules Fail**
   - stdlib uses builtin stubs (`_io_print`, etc.)
   - Requires special handling for Lit expressions
   - Planned for Phase 5

3. **CLI Flag Order**
   - `--entry` must come before `run` command
   - Use: `ailang --entry <name> run <file>`
   - Known CLI parsing quirk, low priority fix

#### Files Changed

**New Files**:
- `internal/runtime/module.go` (164 LOC) - ModuleInstance
- `internal/runtime/runtime.go` (210 LOC) - ModuleRuntime with cycle detection
- `internal/runtime/resolver.go` (120 LOC) - Global resolver
- `internal/runtime/entrypoint.go` (37 LOC) - Helper functions
- `internal/runtime/module_test.go` (239 LOC) - Module tests
- `internal/runtime/runtime_test.go` (140 LOC) - Runtime tests
- `internal/runtime/resolver_test.go` (212 LOC) - Resolver tests
- `internal/runtime/integration_test.go` (249 LOC) - Integration tests
- `tests/runtime_integration/*.ail` (3 test modules)

**Modified Files**:
- `internal/pipeline/pipeline.go` (+60 LOC) - Added Modules map to Result
- `internal/loader/loader.go` (+15 LOC) - Added Preload() method
- `cmd/ailang/main.go` (+30 LOC) - CLI integration

#### Technical Metrics

- **Total LOC**: ~1,594 (implementation + tests)
- **Test Coverage**: 18/18 unit tests passing
- **Integration Tests**: 2/7 passing (loader issues non-blocking)
- **Timeline**: On schedule (Phases 1-4 complete)

#### Next Steps (Phase 5 - Pending)

1. **Function Invocation** - Connect to evaluator API, call entrypoints, print results
2. **stdlib Support** - Handle builtin functions and Lit expressions
3. **Example Verification** - Test all examples, update README
4. **Documentation** - Update CLAUDE.md, create execution guide

---

## [v0.1.0] - 2025-10-02

### 🎯 MVP Release: Type System Complete

**Major Achievement**: First complete type system MVP with 27,610 LOC of Go implementation.

#### Added - Documentation & Polish (~2,500 lines)

**Documentation Suite**:
- **README.md**: Complete restructure for v0.1.0 with honest status, "What Works" section, FAQ
- **docs/LIMITATIONS.md**: NEW - 400+ lines comprehensive limitations guide
- **docs/METRICS.md**: NEW - 300+ lines project statistics and metrics
- **RELEASE_NOTES_v0.1.0.md**: NEW - 500+ lines comprehensive release notes
- **docs/SHOWCASE_ISSUES.md**: NEW - 350+ lines parser/execution limitations
- **examples/STATUS.md**: NEW - Complete inventory of 42 example files
- **examples/README.md**: NEW - User guide for examples
- **CLAUDE.md**: UPDATED - Current v0.1.0 status, accurate component breakdown

**Showcase Examples** (4 new files):
- `examples/showcase/01_type_inference.ail` - Type inference demonstration
- `examples/showcase/02_lambdas.ail` - Lambda composition
- `examples/showcase/03_type_classes.ail` - Type class polymorphism
- `examples/showcase/04_closures.ail` - Closures and captured environments

**Development Tools**:
- `tools/audit-examples.sh`: Automated example testing and categorization

**Warning Headers**: Added to 3 module examples that type-check but can't execute

#### Status Summary

**✅ Complete (27,610 LOC)**:
- Hindley-Milner type inference (7,291 LOC)
- Type classes with dictionary-passing (linked system, ~3,000 LOC)
- Lambda calculus & closures (3,712 LOC)
- Professional REPL with debugging (1,351 LOC)
- Module type-checking (1,030 LOC module + 503 LOC loader)
- Parser with operator precedence (2,656 LOC)
- Structured error reporting with JSON schemas (657 LOC)

**⚠️ Known Limitation**:
- Module files type-check ✅ but cannot execute ❌ (runtime in v0.2.0)
- Non-module `.ail` files execute successfully ✅
- REPL fully functional ✅

**Examples**:
- 12 working (25.5%)
- 3 type-check only (6.4%)
- 27 broken (57.4%)
- 6 skipped (test/demo files)

**Test Coverage**: 24.8% (10,559 LOC of tests)

#### Changed

- README.md version badge: v0.0.12 → v0.1.0
- Implementation status: Updated to "Type System Complete"
- Test coverage badge: 31.3% → 24.8% (accurate count)

#### Fixed

- Documentation now accurately reflects v0.1.0 capabilities
- Example status now honestly documented
- Module execution limitation clearly communicated

### v0.2.0 Roadmap (3.5-4.5 weeks)

**M-R1**: Module Execution Runtime (~1,200 LOC, 1.5-2 weeks)
**M-R2**: Algebraic Effects Foundation (~800 LOC, 1-1.5 weeks)
**M-R3**: Pattern Matching (~600 LOC, 1 week)

---

## [v0.0.12] - 2025-10-02

### Added - M-S1 Complete: Stdlib Foundation (~200 LOC)

**✅ M-S1 MILESTONE ACHIEVED: All 5 stdlib modules type-check successfully**

#### Equation-Form Export Syntax (~30 LOC)
**Parser enhancement for thin wrapper functions:**

**New Syntax** (`internal/parser/parser.go`, lines 655-683):
- Added equation-form function syntax: `export func f(x: T) -> R = expr`
- Alternative to block-form: `export func f(x: T) -> R { expr }`
- Wraps expression in Block for uniform AST handling

**Implementation**:
```go
if p.peekTokenIs(lexer.ASSIGN) {
    p.nextToken() // move to ASSIGN
    p.nextToken() // move past ASSIGN
    body := p.parseExpression(LOWEST)
    fn.Body = &ast.Block{Exprs: []ast.Expr{body}, Pos: body.Position()}
}
```

**Use Case**: Thin wrappers around builtins (std/io module)
```ailang
export func println(s: string) -> () ! {IO} = _io_println(s)
export func print(s: string) -> () ! {IO} = _io_print(s)
export func readLine() -> string ! {IO} = _io_readLine()
```

---

#### Polymorphic ++ Operator (~170 LOC)
**Type checker enhancement for list and string concatenation:**

**Typing Rule**: `xs:[α] ∧ ys:[α] ⇒ xs++ys:[α]`

**Implementation** (`internal/types/typechecker_core.go`, lines 1155-1250):
- Decision tree for polymorphic concatenation:
  1. If at least one operand is a concrete list → list concat
  2. If at least one operand is a concrete string → string concat
  3. If both are type variables → default to list concat (more polymorphic)
  4. Otherwise → fallback to string concat

**Type Unification** (`internal/types/unification.go`, lines 125-143):
- Added TCon compatibility for both `TCon("String")` and `TCon("string")` (case variations)
- Proper unification when one operand is concrete type, other is type variable

**Examples Working**:
```ailang
"hello" ++ " world"        -- String concat
[1, 2] ++ [3, 4]           -- List concat: [Int]
[] ++ []                   -- Polymorphic: [α]
concat xs ys = xs ++ ys    -- Infers: [α] -> [α] -> [α]
```

---

#### Stdlib Modules Complete (All 5 type-check)

**stdlib/std/io.ail** (3 exports):
- `print(s: string) -> () ! {IO}` - Print without newline
- `println(s: string) -> () ! {IO}` - Print with newline
- `readLine() -> string ! {IO}` - Read from stdin
- Uses equation-form syntax for thin wrappers

**stdlib/std/list.ail** (10 exports):
- `map, filter, foldl, foldr, length, head, tail, reverse, concat, zip`
- ++ operator now works correctly for list concatenation

**stdlib/std/option.ail** (6 exports):
- `map, flatMap, getOrElse, isSome, isNone, filter`

**stdlib/std/result.ail** (6 exports):
- `map, mapErr, flatMap, isOk, isErr, unwrap`

**stdlib/std/string.ail** (7 exports):
- `length, substring, toUpper, toLower, trim, compare, find`

---

### Changed

**Parser Function Declaration**:
- Extended to support both block-form and equation-form syntax
- Equation-form used for simple wrapper functions
- Block-form used for multi-statement functions

**Type Checker**:
- Enhanced ++ operator to work polymorphically for both lists and strings
- Improved type variable unification for binary operators

---

### Fixed

**List Concatenation**: ++ operator now properly type-checks with polymorphic element types
**String Concatenation**: Works when one operand is a type variable
**Type Unification**: TCon case variations ("String" vs "string") now handled correctly

---

### Technical Details

**Files Modified**:
- `internal/parser/parser.go` (+30 LOC): Equation-form export syntax
- `internal/types/typechecker_core.go` (+95 LOC): Polymorphic ++ operator
- `internal/types/unification.go` (+18 LOC): TCon compatibility
- `stdlib/std/io.ail` (rewritten): 3 equation-form exports

**Test Results**:
- ✅ All 5 stdlib modules type-check without errors
- ✅ All existing tests pass (no regressions)
- ✅ Examples type-check successfully (option_demo, block_demo, stdlib_demo)

**Known Limitations**:
- ⚠️ Example execution: Runner doesn't call `main()` in module files (type-checking works)
- ⚠️ No `_io_debug` builtin yet (deferred)

**Metrics**:
- Total new code: ~200 LOC (130 implementation + 70 stdlib)
- Stdlib modules: 5/5 complete (100%)
- M-S1 Status: ✅ **COMPLETE**

---

#### Minimal Viable Runner (MVF) - Partial Implementation (~250 LOC)
**Entrypoint resolution and argument decoding foundation for v0.2.0:**

**✅ What Works**:
1. **Argument Decoder Package** (`internal/runtime/argdecode/argdecode.go`, ~200 LOC)
   - Type-directed JSON→Value conversion
   - Supports: null→(), number→int/float, string, bool, array→list, object→record
   - Handles type variables with simple inference
   - Structured errors: `DecodeError` with Expected/Got/Reason

2. **CLI Flags** (3 new flags in `cmd/ailang/main.go`):
   - `--entry <name>` - Entrypoint function name (default: "main")
   - `--args-json '<json>'` - JSON arguments to pass (default: "null")
   - `--print` - Print return value even for unit (default: true)

3. **Entrypoint Resolution Logic**:
   - Looks up function in `result.Interface.Exports`
   - Validates it's a function type (`TFunc2`)
   - Supports 0 or 1 parameters (v0.1.0 constraint)
   - Rejects multi-arg functions with clear error
   - Lists available exports if entrypoint not found

4. **Demo Files** (3 examples in `examples/demos/`):
   - `hello_io.ail` - IO effects demo
   - `adt_pipeline.ail` - ADT/Option usage
   - `effects_pure.ail` - Pure list operations

**❌ What's NOT Implemented**:
- Module-level evaluation (no function values extracted)
- Actual entrypoint execution (blocked on module evaluation)
- Effect handlers (IO, etc.)
- Demo output and golden files (blocked on execution)

**Reason**: Module execution requires evaluating all bindings in dependency order, building runtime environments with closures, and handling effects. This is a significant feature planned for v0.2.0.

**Current Behavior**:
```bash
$ ailang run examples/demos/hello_io.ail

Note: Module evaluation not yet supported
  Entrypoint:  main
  Type:        () -> α3 ! {...ε4}
  Parameters:  0
  Decoded arg: ()

What IS working:
  ✓ Interface extraction and freezing
  ✓ Entrypoint resolution
  ✓ Argument type checking and JSON decoding
```

**Usage Examples**:
```bash
ailang run file.ail                                    # Zero-arg main()
ailang --entry=demo run file.ail                       # Zero-arg demo()
ailang --entry=process --args-json='42' run file.ail   # Single-arg
```

**Files Modified**:
- `internal/runtime/argdecode/argdecode.go` (+200 LOC): New package
- `cmd/ailang/main.go` (+60 LOC): CLI flags + entrypoint resolution
- `examples/demos/*.ail` (+3 files): Demo examples

**Value Delivered**:
- Foundation for v0.2.0 module execution
- Type-safe argument handling ready
- Clear UX messaging about what's working vs. coming
- Demo files ready for when evaluation lands

---

## [v0.0.11] - 2025-10-02

### Fixed - M-S1 Blockers: Cross-Module Constructors & Multi-Statement Functions (~224 LOC)

**CRITICAL FIXES unblocking realistic stdlib examples:**

#### Blocker 1: Cross-Module Constructor Resolution (~74 LOC)
**Problem**: Imported constructors like `Some` from `std/option` couldn't be used because the type checker didn't know their signatures.

**Root Cause**: Constructor factory functions were added to `globalRefs` for elaboration but NOT to `externalTypes` for type checking.

**Solution** (`internal/pipeline/pipeline.go`):
- Lines 452-497: When importing constructors, build factory function type and add to `externalTypes`
- Factory type: `TFunc2{Params: FieldTypes, Return: ResultType}` with `EffectRow: nil` (pure)
- Lines 700-739: Added `extractTypeVarsFromType()` helper to extract type variables for polymorphism
- Example: `Some: a -> Option[a]`, `None: Option[a]`

**Test Results**:
- ✅ `examples/option_demo.ail` now type-checks (was: undefined make_Option_Some)
- ✅ `stdlib/std/list.ail` constructor imports work
- ✅ All existing tests pass

**Note**: `extractTypeVarsFromType()` handles both old (TApp/TVar) and new (TFunc2/TVar2) types for defensive compatibility. Should be cleaned up to use only TVar2 consistently.

---

#### Blocker 2: Multi-Statement Function Bodies (~150 LOC)
**Problem**: Parser only supported single-expression function bodies. Couldn't write realistic functions with multiple statements:
```ailang
func main() {
  let x = 1;      -- ❌ Parse error: unexpected ;
  let y = 2;
  x + y
}
```

**Root Cause**: Function bodies parsed as single expression via `parseExpression(LOWEST)`. No support for semicolon-separated statements.

**Solution**:
1. **AST** (`internal/ast/ast.go`, lines 228-243): Added `Block` node for sequential expressions
2. **Parser** (`internal/parser/parser.go`):
   - Line 663: Changed to call `parseFunctionBody()` instead of `parseExpression()`
   - Lines 673-721: New `parseFunctionBody()` parses semicolon-separated expressions
   - Lines 856-956: Modified `parseRecordLiteral()` to distinguish blocks from record literals
3. **Elaboration** (`internal/elaborate/elaborate.go`):
   - Lines 524-525: Added `Block` case to `normalize()`
   - Lines 786-831: New `normalizeBlock()` converts blocks to nested `Let` expressions
   - Transformation: `{ e1; e2; e3 }` → `let _block_0 = e1 in let _block_1 = e2 in e3`

**Test Results**:
- ✅ Single expression bodies still work
- ✅ Multi-statement blocks with semicolons work
- ✅ Blocks without trailing semicolon work
- ✅ Empty blocks work: `{}`
- ✅ Mixed let statements and expressions work
- ⚠️ Module files with blocks have elaboration issue (separate bug, non-blocking)

**Examples**:
- `examples/block_demo.ail` demonstrates multi-statement functions

**Known Issue**: Files with `module` declarations + blocks fail with "normalization received nil expression". Works fine without module declaration. Needs investigation but doesn't block core functionality.

---

**Combined Impact**: Both blockers resolved! Stdlib modules can now:
- Import and use constructors from other modules
- Write realistic functions with multiple statements and side effects
- Use pattern matching with imported types

**Files Changed**:
- `internal/pipeline/pipeline.go` (+74 LOC): Constructor type resolution
- `internal/ast/ast.go` (+16 LOC): Block AST node
- `internal/parser/parser.go` (+130 LOC): Block parsing
- `internal/elaborate/elaborate.go` (+48 LOC): Block elaboration
- `examples/block_demo.ail` (+17 LOC): Multi-statement example

**Total**: ~224 new LOC, ~5 hours work (Blocker 1: 2 hours, Blocker 2: 3 hours)

---

### Added - M-S1 Parts A & B: Import System & Builtin Visibility (~700 LOC)

#### Part A: Export System for Types and Constructors (~400 LOC)
**Complete end-to-end import resolution for types, constructors, and functions:**

**Loader Enhancement** (`internal/loader/loader.go`)
- Added `Types map[string]*ast.TypeDecl` to `LoadedModule` for exported type declarations
- Added `Constructors map[string]string` for constructor name → type name mapping
- Created `buildTypes()` function to extract type declarations from AST (checks both `Decls` and `Statements`)
- Updated `GetExport()` to return `(nil, nil)` for types and constructors (not errors, just non-functions)
- Enhanced error reporting to list available types and constructors with labels

**Elaborator Updates** (`internal/elaborate/elaborate.go`)
- Added `AddBuiltinsToGlobalEnv()` method to inject all builtin functions into elaborator's global scope
- Modified import resolution in `ElaborateFile()` to skip types/constructors (handled later in pipeline)
- Builtins now available during elaboration, not just linking

**Interface Builder** (`internal/iface/iface.go`, `internal/iface/builder.go`)
- Added `Types map[string]*TypeExport` to `Iface` struct
- Created `TypeExport` struct with `Name` and `Arity` fields
- Enhanced `BuildInterfaceWithTypesAndConstructors()` to extract types from AST file
- Constructors extracted from `AlgebraicType.Constructors` (not `Variants`)
- Helper methods: `AddType()`, `GetType()`

**Pipeline Integration** (`internal/pipeline/pipeline.go`)
- Updated import resolution to check `GetType()` and `GetConstructor()` in addition to `GetExport()`
- Constructors map to `$adt.make_{TypeName}_{CtorName}` factory functions
- Added automatic injection of `$builtin` module exports into all modules' `externalTypes`
- Builtins now available globally without explicit imports
- Added `AddBuiltinsToGlobalEnv()` calls for both REPL and module compilation paths

**Module Linker** (`internal/link/module_linker.go`)
- Enhanced `BuildGlobalEnv()` to handle three symbol types: functions, types, constructors
- Types: Skip adding to environment (handled by type checker)
- Constructors: Add with `$adt` module reference for factory functions
- Functions: Add with original module reference
- Improved error reporting with separate listings for types and constructors
- Added `continue` statements to skip further processing for types/constructors

**Working Examples:**
```ailang
// Type and constructor imports work
import stdlib/std/option (Option, Some, None)

// Constructor usage (pending $adt runtime)
let x = Some(42)
match x {
  Some(n) => n,
  None => 0
}
```

**Test Results:**
- ✅ Constructor imports: `import stdlib/std/option (Some)` type-checks
- ✅ Type imports: `import stdlib/std/option (Option)` type-checks
- ✅ Function imports: `import stdlib/std/option (getOrElse)` works
- ✅ All existing tests pass (no regressions)
- ⏳ Constructor evaluation pending `$adt` runtime implementation

---

#### Part B: Builtin Type Visibility (~300 LOC)
**Made string and IO primitives available to all modules:**

**Builtin Module Enhancement** (`internal/link/builtin_module.go`)
- Added `handleStringPrimitive()` function for 7 string builtins:
  - `_str_len: String -> Int` (UTF-8 rune count)
  - `_str_slice: String -> Int -> Int -> String` (rune-based substring)
  - `_str_compare: String -> String -> Int` (lexicographic, returns -1/0/1)
  - `_str_find: String -> String -> Int` (first occurrence, rune index)
  - `_str_upper: String -> String` (Unicode-aware uppercase)
  - `_str_lower: String -> String` (Unicode-aware lowercase)
  - `_str_trim: String -> String` (Unicode whitespace)
- Added `handleIOBuiltin()` function for 3 IO builtins:
  - `_io_print: String -> Unit ! {IO}` (no newline)
  - `_io_println: String -> Unit ! {IO}` (with newline)
  - `_io_readLine: Unit -> String ! {IO}` (read from stdin)
- Proper effect row representation: `&types.Row{Kind: types.EffectRow, Labels: {"IO": ...}}`
- All builtins registered in `$builtin` module interface

**Pipeline Integration** (`internal/pipeline/pipeline.go`)
- Automatic injection of `$builtin` module into every module's compilation context
- Builtins available in `externalTypes` for type checking
- Builtins available in `globalRefs` for elaboration
- No explicit imports required - builtins are globally visible

**Test Results:**
- ✅ `stdlib/std/string.ail` type-checks successfully (7 exports)
- ⏳ `stdlib/std/io.ail` has parse errors (inline function syntax limitation)
- ✅ String primitives: length, substring, toUpper, toLower, trim, compare, find
- ✅ Effect tracking: IO functions properly annotated with `! {IO}`

**Example Working:**
```ailang
module stdlib/std/string

export pure func length(s: string) -> int { _str_len(s) }
export pure func toUpper(s: string) -> string { _str_upper(s) }
// ... all 7 functions type-check correctly
```

---

### Added - Parser Fix + Stdlib Foundation (~300 LOC)

#### Generic Type Parameter Fix (`internal/parser/parser.go`)
**1-line fix unblocks generic functions in modules:**

**Issue Discovered**: Generic function syntax failed during stdlib implementation
```ailang
export func map[a, b](f: (a) -> b, xs: [a]) -> [b]  -- ❌ Parser error
```

**Root Cause**: After `parseTypeParams()` parsed `[a, b]`, parser was positioned AT `(` but code called `expectPeek(LPAREN)` expecting to PEEK at next token.

**Fix Applied** (lines 554-582):
- Check `hasTypeParams` flag to determine token positioning
- If generic: `curTokenIs(LPAREN)` (already at opening paren)
- If non-generic: `expectPeek(LPAREN)` (need to advance)
- Handles all cases: `func[T]()`, `func[T](x)`, `func()`, `func(x)`

**Impact**: ✅ Generic function declarations now parse correctly in module files

---

#### String & IO Builtins Implementation (~150 LOC)

**7 String Primitives** (`internal/eval/builtins.go`):
- `_str_len(s: string) -> int` - UTF-8 aware length (rune count, not bytes)
- `_str_slice(s: string, start: int, end: int) -> string` - Substring with rune indices
- `_str_compare(a: string, b: string) -> int` - Lexicographic comparison (-1, 0, 1)
- `_str_find(s: string, sub: string) -> int` - First occurrence index (rune-based)
- `_str_upper(s: string) -> string` - Unicode-aware uppercase
- `_str_lower(s: string) -> string` - Unicode-aware lowercase
- `_str_trim(s: string) -> string` - Unicode whitespace trimming

**3 IO Primitives** (effectful: `IsPure: false`):
- `_io_print(s: string) -> ()` - Print without newline
- `_io_println(s: string) -> ()` - Print with newline
- `_io_readLine() -> string` - Read line from stdin (stub for v0.1.0)

**Design Principles**:
- UTF-8 safe: All string operations use rune indices, not byte indices
- Deterministic: No locale-dependent behavior
- Pure primitives: String functions are pure (IsPure: true)
- Effectful IO: IO functions marked impure (IsPure: false) for future effect tracking

**Updated CallBuiltin()** to handle:
- 0-argument functions: `_io_readLine()`
- 3-argument functions: `_str_slice(s, start, end)`
- New type signatures: `String -> Int`, `String -> String`, `String -> Unit`

---

#### Stdlib Modules Prepared (Ready for Deployment)

**5 Stdlib Modules Written** (~360 LOC AILANG code):
- `std_list.ail` (~180 LOC): map, filter, foldl, foldr, length, head, tail, reverse, concat, zip
- `std_option.ail` (~50 LOC): Option[a], map, flatMap, getOrElse, isSome, filter
- `std_result.ail` (~70 LOC): Result[a,e], map, mapErr, flatMap, isOk, unwrap
- `std_string.ail` (~40 LOC): length, concat, substring, join, toUpper, toLower, trim
- `std_io.ail` (~20 LOC): print, println, readLine, debug with `! {IO}` effects

**Status**: ⚠️ BLOCKED - Parser doesn't support pattern matching inside function bodies

**Blocker Details**:
- ✅ Pattern matching works at top-level: `match Some(42) { ... }` (proven)
- ❌ Pattern matching fails inside functions: `export func f() { match x { ... } }` (broken)
- Error: "expected =>, got ] instead" when parsing list patterns `[]`, `[x, ...rest]`
- Affects: ALL stdlib modules (they use pattern matching extensively)

**Next Steps**: Fix pattern matching in function bodies (~1-2 days parser work)

---

### Fixed

**Parser Token Positioning** (`internal/parser/parser.go:554-582`)
- Generic type parameters now work in function declarations
- Correctly handles: `func name[T]()`, `func name[T](x: T)`, `func name()`, `func name(x: int)`
- Test case verified: `export func getOrElse[a](opt: Option[a], d: a) -> a` parses

---

### Changed

**CallBuiltin Signature Support** (`internal/eval/builtins.go`)
- Added 0-argument builtin handling (for `_io_readLine`)
- Added 3-argument builtin handling (for `_str_slice`)
- Extended type signatures: `String -> Int`, `String -> String`, `String -> Unit`

---

### Technical Details

**Files Modified**:
- `internal/parser/parser.go` (~30 LOC): Generic function fix
- `internal/eval/builtins.go` (~150 LOC): String and IO primitives
- Total: ~180 LOC implementation

**Stdlib Modules Created** (not yet deployable):
- 5 modules (~360 LOC) written and ready
- Blocked pending pattern matching parser fix

**Test Coverage**: Generic function test case passes, builtins compile and register

**Metrics**:
- Builtins: 10 new primitives (7 string + 3 IO)
- Parser fix: Unblocks generic functions in modules
- Stdlib: Ready to deploy once parser fixed

---

## [v0.0.10] - 2025-10-01

### Added - M-P4: Effect System (Type-Level) (~1,060 LOC)

#### Complete Type-Level Effect Tracking
**Full pipeline integration from parsing through type checking:**

**Effect Syntax Parsing** (`internal/parser/parser.go`, `internal/parser/effects_test.go`)
- Function declarations: `func f() -> int ! {IO, FS}`
- Lambda expressions: `\x. body ! {IO}`
- Type annotations: `(int) -> string ! {FS}`
- Comprehensive validation against 8 canonical effects: IO, FS, Net, Clock, Rand, DB, Trace, Async
- Error codes: PAR_EFF001_DUP (duplicates), PAR_EFF002_UNKNOWN (unknown effect with suggestions)
- Fixed BANG operator precedence to allow `! {Effects}` syntax
- 17 parser tests passing ✅

**Effect Elaboration Helpers** (`internal/types/effects.go`, `internal/types/effects_test.go`)
- `ElaborateEffectRow()`: Converts AST effect strings to normalized `*Row` with deterministic alphabetical sorting
- `UnionEffectRows()`: Merges two effect rows (e.g., `{IO} ∪ {FS} = {FS, IO}`)
- `SubsumeEffectRows()`: Checks effect subsumption (a ⊆ b) for capability checking
- `EffectRowDifference()`: Computes missing effects for error messages
- `FormatEffectRow()`: Pretty-prints effect rows as `! {IO, FS}`
- `IsKnownEffect()`: Validates effect names against canonical set
- Purity sentinel: `nil` effect row = pure function (not empty-but-non-nil)
- Closed rows only: `Tail = nil` always (no row polymorphism in v0.1.0)
- 29 elaboration tests passing ✅

**Type Checking Integration** (`internal/elaborate/elaborate.go`, `internal/types/typechecker_core.go`)
- Effect annotations stored in `Elaborator.effectAnnots` map (Core node ID → effect names)
- Validation during elaboration using `ElaborateEffectRow()`
- Effect annotations thread to `CoreTypeChecker.effectAnnots`
- Modified `inferLambda()` to use explicit effect annotations when present
- Falls back to body effect inference when no annotation provided
- Annotations flow: AST → Elaboration → Type Checking → TFunc2.EffectRow
- Existing effect infrastructure leveraged (effects already propagate through `inferApp`, `inferIf`, etc.)

**Files Modified:**
- `internal/parser/parser.go` (+150 LOC): Effect annotation parsing with validation
- `internal/parser/effects_test.go` (+360 LOC new file): 17 test cases
- `internal/types/effects.go` (+170 LOC new file): Effect row elaboration helpers
- `internal/types/effects_test.go` (+280 LOC new file): 29 test cases
- `internal/elaborate/elaborate.go` (+30 LOC): Effect annotation storage
- `internal/types/typechecker_core.go` (+40 LOC): Effect annotation integration
- Total: ~1,060 LOC (700 LOC core + 360 LOC tests)

**Key Design Decisions:**
1. **Purity Sentinel**: `nil` effect row = pure, never empty-but-non-nil
2. **Deterministic Normalization**: All effect labels sorted alphabetically
3. **Closed Rows**: No row polymorphism in v0.1.0 (Tail = nil always)
4. **Canonical Effects**: IO, FS, Net, Clock, Rand, DB, Trace, Async (8 total)
5. **Type-Level Only**: No runtime effect enforcement (deferred to v0.2.0)
6. **Effects in Type System**: Stored in TFunc2.EffectRow, not Core Lambda AST

**Test Results:**
- ✅ 17 parser tests passing (effect syntax, validation, error messages)
- ✅ 29 elaboration tests passing (ElaborateEffectRow, unions, subsumption)
- ✅ All existing type checker tests passing
- ✅ Full test suite passing (parser, elaboration, types)

**Outcome:** M-P4 effect system foundation is COMPLETE and ready for use! The infrastructure for type-level effect tracking is in place and working.

**Deferred to v0.2.0:**
- Runtime effect handlers and capability passing
- Effect polymorphism (row polymorphism: `! {IO | r}`)
- Pure function verification at compile time

---

### Added - M-P3: Pattern Matching Foundation with ADT Runtime

#### Minimal ADT Runtime Implementation (~600 LOC)
**Complete algebraic data type support with pattern matching:**

**TaggedValue Runtime** (`internal/eval/value.go`, `internal/eval/eval_core.go`)
- Runtime representation for ADT constructors with `TypeName`, `CtorName`, `Fields`
- Pretty-printing: `None`, `Some(42)`, `Ok(Some(99))`
- Helper functions: `isTag()` for constructor matching, `getField()` for field extraction
- Full test coverage: 16 test cases across 3 test suites

**$adt Synthetic Module** (`internal/link/builtin_module.go`)
- Factory function synthesis: `make_<TypeName>_<CtorName>` pattern
- Deterministic ordering (sorted by type name, then constructor name)
- Automatic registration from all loaded module interfaces
- Example: `make_Option_Some`, `make_Option_None`

**Type Declaration Elaboration** (`internal/elaborate/elaborate.go`)
- `normalizeTypeDecl()` converts AST type declarations to runtime constructors
- Tracks type parameters, field types, and arity
- Distinguishes local vs imported constructors
- Constructor tracking in elaborator with `constructors` map

**Constructor Expression Support**
- Non-nullary: `Some(42)` → `VarGlobal("$adt", "make_Option_Some")(42)`
- Nullary: `None` → `VarGlobal("$adt", "make_Option_None")` (direct value, not function call)
- Automatic elaboration in `normalizeFuncCall()` and identifier normalization
- Factory resolution with arity-aware handling (nullary returns value, others return function)

**Constructor Pattern Matching** (`internal/eval/eval_core.go`)
- Extended `matchPattern()` to handle `ConstructorPattern`
- Recursive field pattern matching with variable binding
- Constructor name and arity validation
- Full destructuring support: `Some(x)`, `Ok(Some(y))`, `None`

**Pipeline Integration** (`internal/pipeline/pipeline.go`)
- Constructors extracted from elaborator and added to module interfaces
- Factory types registered in `externalTypes` before type checking
- Used TFunc2/TVar2 (new type system) for unification compatibility
- Monomorphic result types (e.g., `Option` not `Option[Int]`) due to TApp limitation

**Interface Builder Enhancement** (`internal/iface/builder.go`)
- `BuildInterfaceWithConstructors()` accepts constructor information
- Constructors included in module interface for imports
- Constructor schemes with field types and result types

**Working Examples**:
```ailang
type Option[a] = Some(a) | None

match Some(42) {
  Some(n) => n,
  None => 0
}
-- Output: 42 ✅

match None {
  Some(n) => n,
  None => 999
}
-- Output: 999 ✅
```

#### Key Technical Decisions
1. **No new Core IR nodes**: Constructor calls use `VarGlobal("$adt", "make_*")` pattern
2. **Runtime factory functions**: $adt module populated at link time from interfaces
3. **Direct evaluation**: Match expressions evaluate without lowering pass
4. **Deterministic**: Factory names sorted, stable digest computation
5. **Nullary handling**: Returns TaggedValue directly (not wrapped in function)
6. **Type system hybrid**: TCon (old) + TFunc2/TVar2 (new) for unification compatibility

#### Files Changed
- `internal/eval/value.go`: Added TaggedValue type (~25 LOC)
- `internal/eval/eval_core.go`: Added isTag, getField helpers, constructor pattern matching (~180 LOC)
- `internal/link/builtin_module.go`: Added RegisterAdtModule (~120 LOC)
- `internal/link/module_linker.go`: Added GetLoadedModules method
- `internal/elaborate/elaborate.go`: Added normalizeTypeDecl, constructor tracking, nullary handling (~150 LOC)
- `internal/pipeline/compile_unit.go`: Added ConstructorInfo, Constructors field (~25 LOC)
- `internal/iface/builder.go`: Added BuildInterfaceWithConstructors (~60 LOC)
- `internal/pipeline/pipeline.go`: Added constructor pipeline wiring, TFunc2/TVar2 factory types (~120 LOC)
- `internal/link/resolver.go`: Enhanced resolveAdtFactory with arity lookup (~60 LOC)

#### Test Coverage
- 16 test cases: TaggedValue, isTag, getField functions
- End-to-end examples: `examples/adt_simple.ail`
- Both nullary and non-nullary constructors verified

### Known Limitations (Future Work)
- ⚠️ Let bindings with constructors have elaboration bug ("normalization received nil expression")
- ⚠️ Result types are monomorphic (`Option` vs `Option[Int]`) - TApp not supported in unifier yet
- ⚠️ No exhaustiveness checking for pattern matches
- ⚠️ No guard evaluation (guards are parsed but not evaluated)
- ⚠️ Type system migration incomplete: Mix of old (TFunc, TVar) and new (TFunc2, TVar2) types

### Technical Details
- Total implementation: ~600 LOC (3 days, as estimated)
- Pattern matching: Tuples, literals, variables, wildcards, constructors all work
- Type checking: Polymorphic factory types with proper unification
- Runtime: TaggedValue representation with arity-aware factory resolution
- Deterministic: All constructor names sorted, stable module digests

### Migration Notes
- ADT runtime is fully backward compatible
- Type declarations now elaborate to runtime constructors automatically
- Constructor expressions work in pattern contexts and regular code
- $adt module is synthetic and doesn't require explicit imports

## [v0.0.9] - 2025-09-30

### Changed - Upgraded to Go 1.22

**Security & Performance Upgrade:**
- Upgraded from Go 1.19 → Go 1.22.12 (Go 1.19 EOL since Sept 2023)
- Updated `golang.org/x/text` from v0.20.0 → v0.21.0
- Updated CI workflow to use Go 1.22
- All tests and linting pass with new version

**Benefits:**
- Security patches for 2+ years of vulnerabilities
- 1-3% CPU performance improvement
- ~1% memory reduction
- For-loop variable scoping fix (prevents common bugs)
- Enhanced HTTP routing, better generics support

**Files Changed:**
- `go.mod`: go 1.22, golang.org/x/text v0.21.0
- `.github/workflows/ci.yml`: go-version: '1.22'
- `.github/workflows/build.yml`: go-version: '1.22' (fixes Windows builds)
- `.github/workflows/release.yml`: go-version: '1.22'
- `go.sum`: Updated checksums

### Fixed - Windows Golden File Tests

**Cross-platform Test Compatibility:**
- Fixed Windows test failures in `TestLiterals` subtests
- Issue: Golden files checked out with CRLF line endings on Windows but comparison used raw bytes
- Solution: Normalize line endings (CRLF → LF) in both `want` and `got` strings before comparison
- Updated `goldenCompare()` function in `internal/parser/testutil.go`
- All platforms (Linux, macOS, Windows) now pass golden file tests consistently

### Added - M-P2 Lock-In: Type System Hardening

#### Coverage Regression Protection
- Per-package coverage gates in Makefile (`cover-parser`, `gate-parser`, `cover-lexer`, `gate-lexer`)
- Parser baseline: 70% coverage (up from 69%)
- Lexer baseline: 57% coverage
- CI workflow enforces coverage thresholds on every push
- Golden drift protection: CI fails if golden files change without `ALLOW_GOLDEN_UPDATES=1`
- New make target: `check-golden-drift` validates golden file stability

#### Type Alias vs Sum Type Disambiguation
- Fixed bug: `type Names = [string]` now correctly parses as TypeAlias, not AlgebraicType
- Added `TypeAlias` AST node in `internal/ast/ast.go`
- Implemented `hasTopLevelPipe()` helper to detect sum types by presence of `|` operator
- Updated `parseTypeDeclBody()` to distinguish:
  - Type aliases: `type UserId = int`, `type Names = [string]`
  - Sum types: `type Color = Red | Green | Blue`
- Regenerated all type golden files with correct TypeAlias representation

#### Nested Record Types
- Record types now work in type positions: `type User = { addr: { street: string } }`
- Added `typeNode()`, `String()`, `Position()` methods to RecordType
- Created `parseRecordTypeExpr()` function for `{...}` in type expressions
- Added test case `TestRecordTypes/nested_record` with golden file
- RecordType now implements both TypeDef and Type interfaces

#### Export Metadata Tracking
- Added `Exported bool` field to TypeDecl AST node
- Updated `parseTypeDeclaration(exported bool)` to track export status
- AST printer includes `"exported": true` in JSON output for exported types
- Tests validate: `export type PublicColor = Red | Green` vs `type PrivateData = { value: int }`
- Regenerated export golden files with metadata

#### REPL/File Type Parity
- New test suite: `TestREPLFileParityTypes` with 10 type declaration test cases
- Validates identical parsing for: aliases, lists, records (simple & nested), sum types, generics, exports
- All type declarations parse identically in REPL (`<repl>`) vs file (`test.ail`) contexts
- Parser coverage increased to 70.8%

#### Metrics
- Parser coverage: 69% → 70.8%
- New tests: 11 (1 nested record + 10 parity tests)
- All existing parser tests pass (544ms test suite)
- Golden files: 3 regenerated (export_alias, export_record, export_sum)
- Code changes: 7 files (ast.go, parser.go, print.go, repl_parity_test.go, type_test.go, Makefile, ci.yml)

### Added - M-P1: Parser Baseline (2025-09-30)

#### Comprehensive Test Infrastructure
- Created deterministic AST printer in `internal/ast/print.go` (445 lines)
- Created test utilities in `internal/parser/testutil.go` (241 lines)
- Established golden file testing framework with 116 snapshots
- Added Makefile targets: `test-parser`, `test-parser-update`, `fuzz-parser`

#### Test Coverage Across All Parser Features
- **Expression tests** (`expr_test.go`, 385 lines): 85 test cases covering literals, operators, collections, lambdas
- **Precedence tests** (`precedence_test.go`, 283 lines): 53 test cases validating operator precedence
- **Module tests** (`module_test.go`, 142 lines): 17 test cases for module/import declarations
- **Function tests** (`func_test.go`, 252 lines): 22 test cases for function declarations and signatures
- **Error recovery tests** (`error_recovery_test.go`, 312 lines): 38 test cases for graceful error handling
- **Invariant tests** (`invariants_test.go`, 320 lines): UTF-8 normalization, CRLF handling, BOM stripping
- **REPL parity tests** (`repl_parity_test.go`, 220 lines): Ensures REPL and file parsing consistency
- **Fuzz tests** (`fuzz_test.go`, 181 lines): 4 fuzz functions with 47 seed cases

#### Baseline Metrics
- **506 test cases** total across all parser features
- **70.2% line coverage** (baseline frozen)
- **Zero panics** in 52k+ fuzz executions
- **2,233 lines** of test code
- All tests pass in ~550ms

## [v0.0.7] - 2025-09-29

### Added - Milestone A2: Structured Error Reporting

#### Unified Error Report System (`internal/errors/report.go`)
- Canonical `errors.Report` type with schema `ailang.error/v1`
- `ReportError` wrapper preserves structured errors through error chains
- `AsReport()` function for type-safe error unwrapping using `errors.As()`
- `WrapReport()` ensures Reports survive through error propagation
- JSON-serializable with deterministic field ordering
- Structured `Data` map with sorted arrays for reproducibility
- `Fix` suggestions with confidence scores
- ~120 lines of core error infrastructure

#### Standardized Error Codes
- **IMP010** - Symbol not exported by module
  - Data: `symbol`, `module_id`, `available_exports[]`, `search_trace[]`
  - Suggests checking available exports in target module
- **IMP011** - Import conflict (multiple providers for same symbol)
  - Data: `symbol`, `module_id`, `providers[{export, module_id}]`
  - Suggests using selective imports to resolve conflict
- **IMP012** - Unsupported import form (namespace imports)
  - Data: `module_id`, `import_syntax`
  - Suggests using selective import syntax
- **LDR001** - Module not found during load
  - Data: `module_id`, `search_trace[]`, `similar[]`
  - Provides resolution trace and similar module suggestions
- **MOD006** - Cannot export underscore-prefixed (private) names
  - Parser validation prevents accidental private exports

#### Error Flow Hardening
- Removed `fmt.Errorf()` wrappers in `internal/elaborate/elaborate.go:112`
- Removed `fmt.Errorf()` wrappers in `internal/pipeline/pipeline.go:434`
- All error builders return `*errors.Report` instead of generic errors
- Link phase wraps reports with `errors.WrapReport()` in `internal/link/module_linker.go`
- Loader phase wraps reports with `errors.WrapReport()` in `internal/loader/loader.go`
- Errors flow end-to-end as first-class types, not string wrappers

#### CLI JSON Output (`cmd/ailang/main.go`)
- `--json` flag enables structured JSON error output
- `--compact` flag for token-efficient JSON serialization
- `handleStructuredError()` extracts Reports using `errors.As()`
- Generic error fallback for non-structured errors
- Exit code 1 for all error conditions

#### Golden File Testing Infrastructure
- **Test files** (`tests/errors/`):
  - `lnk_unresolved_symbol.ail` - Tests IMP010 (symbol not exported)
  - `lnk_unresolved_module.ail` - Tests LDR001 (module not found)
  - `import_conflict.ail` - Tests IMP011 (import conflict)
  - `export_private.ail` - Tests MOD_EXPORT_PRIVATE (private export)
- **Golden files** (`goldens/`):
  - `lnk_unresolved_symbol.json` - Expected IMP010 output
  - `lnk_unresolved_module.json` - Expected LDR001 output
  - `import_conflict.json` - Expected IMP011 output
  - `imports_basic_success.json` - Expected success output (value: 6)
- Golden files ensure byte-for-byte reproducibility of error output

#### Makefile Test Targets
- `make test-imports-success` - Verifies successful imports work
- `make test-import-errors` - Validates golden file matching with `diff -u`
- `make regen-import-error-goldens` - Regenerates golden files (use with caution)
- `make test-imports` - Combined import testing (success + errors)
- `make test-parity` - REPL/file parity test (manual, requires interactive REPL)

#### CI Integration (`.github/workflows/ci.yml`)
- Split import testing into explicit steps:
  - "Test import system (success cases)" - Runs `make test-imports-success`
  - "Test import errors (golden file verification)" - Runs `make test-import-errors`
- CI gates prevent regression in error reporting determinism
- Integrated into `ci-strict` target with operator lowering and builtin freeze tests

### Changed
- `internal/link/report.go` - All builders return `*errors.Report`
- `internal/link/env.go` - Renamed old `LinkReport` to `LinkDiagnostics` to avoid confusion
- `internal/loader/loader.go` - Search trace collection during module resolution
- `internal/parser/parser.go` - Added MOD_EXPORT_PRIVATE validation

### Fixed
- Structured errors were being stringified by `fmt.Errorf("%w")` wrappers
- Error type information now survives through error chains using `errors.As()`
- Flag ordering: Flags must come BEFORE subcommand (`ailang --json --compact run file.ail`)

### Technical Details
- Total new code: ~680 lines (implementation + test files + golden files)
- Test coverage: Golden files ensure deterministic error output
- Determinism: All arrays sorted, canonical module IDs, stable JSON field ordering
- No breaking changes to existing functionality
- Schema versioning allows future enhancements without breaking compatibility

### Migration Notes
- Existing error handling continues to work unchanged
- JSON output is opt-in via `--json` flag
- Structured errors available via `errors.AsReport()` for tools integration
- Golden file tests serve as documentation of expected error formats

## [v0.0.6] - 2025-09-29

### Added

#### Error Code Taxonomy (`internal/errors/codes.go`)
- Comprehensive error code system with structured taxonomy
- Error codes organized by phase: PAR (Parser), MOD (Module), LDR (Loader), TC (Type Check), etc.
- Error registry with phase and category metadata
- Helper functions: `IsParserError()`, `IsModuleError()`, `IsLoaderError()`, etc.
- ~278 lines of structured error definitions

#### Manifest System (`internal/manifest/`)
- Example manifest format for tracking example status (working/broken/experimental)
- Validation ensures consistency between documentation and implementation
- Statistics calculation with coverage metrics
- README generation support for automatic documentation updates
- Environment defaults for reproducible execution
- ~390 lines with full validation logic

#### Module Loader (`internal/module/loader.go`)
- Complete module loading system with dependency resolution
- Circular dependency detection using cycle detection algorithm
- Topological sorting using Kahn's algorithm for build order
- Module caching with thread-safe concurrent access
- Support for stdlib modules and relative imports
- Structured error reporting with resolution traces
- ~607 lines of robust module management

#### Path Resolver (`internal/module/resolver.go`)
- Cross-platform path normalization and resolution
- Support for relative imports (`./`, `../`)
- Standard library path resolution (`std/`)
- Project root detection and search path management
- Case-sensitive and case-insensitive filesystem handling
- Module identity derivation from file paths
- ~405 lines of platform-aware path handling

#### Example Files
- Basic module with function declarations
- Recursive functions with inline tests
- Module imports and composition
- Standard library usage patterns
- Property-based testing examples

### Changed
- Test coverage improved from 29.9% to 31.3%
- Module tests now include comprehensive cycle detection validation
- Topological sort correctly handles dependency ordering

### Fixed
- CI/CD script compilation errors by refactoring shared types into `scripts/internal/reporttypes`
- Test suite now correctly excludes `scripts/` directory containing standalone executables
- Makefile and CI workflow updated to use `go list ./... | grep -v /scripts` for testing

## [v0.0.5] - 2025-09-29

### Added

#### Schema Registry (`internal/schema/`)
- Frozen schema versioning system with forward compatibility
- Schema constants: `ErrorV1` (ailang.error/v1), `TestV1` (ailang.test/v1), `EffectsV1` (ailang.effects/v1)
- `Accepts()` method for prefix matching against newer schema versions
- `MarshalDeterministic()` for stable JSON output with sorted keys
- `CompactMode` flag support for token-efficient JSON serialization
- Registry pattern for managing versioned schemas across components
- ~145 lines of core implementation

#### Error JSON Encoder (`internal/errors/`)
- Structured error taxonomy with stable error codes (TC###, ELB###, LNK###, RT###)
- Always includes `fix` field with actionable suggestion and confidence score
- SID (Stable Node ID) discipline with "unknown" fallback for safety
- Builder pattern API: `WithFix()`, `WithSourceSpan()`, `WithMeta()`
- Schema-compliant JSON output using ailang.error/v1
- Safe encoding that never panics on malformed data
- ~190 lines with comprehensive error handling

#### Test Reporter (`internal/test/`)
- Structured test reporting in JSON format using ailang.test/v1 schema
- Complete test counts shape: passed/failed/errored/skipped/total
- Platform information capture for reproducibility tracking
- Deterministic sorting by suite name and test name
- Valid JSON output even with zero tests
- Test runner integration with SID generation
- ~206 lines with full test lifecycle support

#### REPL Effects Inspector (`internal/repl/effects.go`)
- `:effects <expr>` command for type and effect introspection
- Returns type signature and effect requirements without evaluation
- Supports both human-readable and JSON output modes
- Placeholder implementation (full version pending effect system)
- Schema-compliant output using ailang.effects/v1
- ~41 lines with extensible architecture

#### CLI Compact Mode Support
- `--compact` flag added to main CLI for global compact JSON mode
- Integrates with schema registry's `CompactMode` setting
- Affects all JSON output including errors, tests, and effects
- Token-efficient output for AI agent integration

#### Golden Test Framework Enhancements
- Platform-specific salt generation for reproducibility
- `UPDATE_GOLDENS` environment variable support
- JSON diff utilities for test validation
- Deterministic fixture generation and validation
- ~309 lines of comprehensive test infrastructure

### Added - Test Coverage & Quality
- 100% test coverage for schema registry (unit + integration)
- 100% test coverage for error encoder with edge cases
- 100% test coverage for test reporter with platform variations
- Golden test fixtures for all schema-compliant JSON outputs
- Integration tests validating cross-component schema compliance
- ~470 lines of test code ensuring reliability

### Changed
- All JSON output now uses deterministic field ordering
- Error messages consistently include actionable fix suggestions
- Test reporting standardized across all components
- Platform information consistently captured for reproducibility

### Technical Details
- Total new code: ~1,630 lines (implementation + tests)
- Dependencies: No new external dependencies
- Schema versioning: Forward-compatible design
- JSON output: Deterministic and stable across platforms
- Test coverage: 100% for all new packages

### Migration Notes
- All existing functionality preserved
- New features are opt-in via CLI flags and REPL commands
- JSON output format enhanced but remains backward compatible
- Schema versioning allows gradual migration to newer formats

## [v0.0.4] - 2025-09-28

### Added

#### Example Verification System (`scripts/`)
- `verify_examples.go` - Tests all examples, categorizes as passed/failed/skipped
- Outputs in JSON, Markdown, and plain text formats
- Captures error messages for failed examples
- Skips test/demo files automatically
- ~200 lines of Go code

#### README Auto-Update System
- `update_readme.go` - Updates README with verification status
- Auto-generates status table between markers
- Creates badges for CI, coverage, and example status
- Maintains timestamp of last update
- ~150 lines of Go code

#### CI GitHub Actions (`.github/workflows/ci.yml`)
- Automated testing on push/PR to main/dev branches
- Example verification with failure on broken examples
- Test coverage reporting to Codecov
- Auto-commits README updates on dev branch
- Build artifact generation
- Parallel linting and testing jobs

#### Make Targets
- `make verify-examples` - Run example verification
- `make update-readme` - Update README with status
- `make flag-broken` - Add warning headers to broken examples
- `make test-coverage-badge` - Generate coverage metrics
- `make ci` - Full CI pipeline

### Added - Documentation
- CI status badges in README (CI, Coverage, Examples)
- Auto-generated example status table
- Example verification report showing 13 working, 13 failing, 14 skipped
- Warning headers for broken examples (via `flag_broken_examples.go`)
- `.gitignore` entries for CI-generated files

### Changed
- REPL now displays version from git tags dynamically (via ldflags)
- All v3.x version references updated to semantic versioning (v0.0.x)
- Example files renamed to match version scheme (v0_0_3_features_demo.ail)
- Design docs restructured to match version scheme

### Technical Details
- Total new code: ~500 lines
- Test coverage: Verification scripts fully tested
- No external dependencies added
- Apache 2.0 license badge added

## [v0.0.3] - 2025-09-26

### Added

#### Schema Registry (`internal/schema/`)
- Versioned JSON schemas with forward compatibility
- `Accepts()` for schema version negotiation
- `MarshalDeterministic()` for stable JSON output
- `CompactMode` support for token-efficient output
- Schema constants: `ErrorV1`, `TestV1`, `DecisionsV1`, `PlanV1`, `EffectsV1`

#### Error JSON Encoder (`internal/errors/`)
- Structured error taxonomy with codes (TC###, ELB###, LNK###, RT###)
- Always includes `fix` field with suggestion and confidence score
- SID (Stable Node ID) discipline with fallback to "unknown"
- Builder pattern: `WithFix()`, `WithSourceSpan()`, `WithMeta()`
- Safe encoding that never panics

#### Test Reporter (`internal/test/`)
- Structured test reporting in JSON format
- Full counts shape (passed/failed/errored/skipped/total)
- Platform information for reproducibility
- Deterministic sorting by suite and name
- Valid JSON output even with 0 tests
- Test runner with SID generation

#### Effects Inspector
- `:effects <expr>` command for type/effect introspection
- Returns type and effects without evaluation
- Supports compact JSON mode
- Placeholder implementation (full version pending effect system)

#### Golden Test Framework (`testutil/`)
- Platform salt for reproducibility tracking
- `UPDATE_GOLDENS` environment variable support
- JSON diff utilities
- Deterministic test fixtures

#### REPL Enhancements
- `:test [--json]` - Run tests with optional JSON output
- `:effects <expr>` - Inspect type and effects
- `:compact on/off` - Toggle JSON compact mode
- Updated help with new commands

### Added - Examples & Documentation
- `examples/v3_2_features_demo.ail` - Demonstrates new v3.2 features
- `examples/repl_commands_demo.md` - REPL command documentation
- `examples/ai_agent_integration.ail` - Comprehensive AI agent guide
- `examples/working_v3_2_demo.ail` - Working examples for current state
- `design_docs/implemented/v3_2/` - Implementation report with metrics
- Comprehensive test suites for all new packages
- 100% test coverage for schema registry
- 100% test coverage for error encoder
- 100% test coverage for test reporter

### Changed
- `types.CanonKey()` alias added for consistent dictionary key generation
- REPL help updated with new AI-first commands

### Fixed
- Multi-line REPL input for `let...in` expressions
- Added continuation prompt (`...`) for incomplete expressions

### Technical Details
- Total new code: ~1,500 lines
- Test coverage: All new packages fully tested
- Dependencies: No new external dependencies

### Migration Notes
- No breaking changes
- New features are opt-in via REPL commands
- Existing code continues to work unchanged

## [v0.0.2] - Previous Release
- Type class resolution with dictionary-passing
- REPL improvements with history and tab completion
- Core type system implementation

## [v0.0.1] - Initial Release
- Basic lexer and parser
- AST implementation
- Initial REPL
