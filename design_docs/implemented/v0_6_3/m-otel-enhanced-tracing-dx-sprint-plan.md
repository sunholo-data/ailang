# Sprint Plan: M-OTEL-ENHANCED-TRACING-DX

**Sprint ID:** M-OTEL-ENHANCED-TRACING-DX
**Design Doc:** [m-otel-enhanced-tracing-dx.md](m-otel-enhanced-tracing-dx.md)
**Duration:** 1 day (~6 hours implementation)
**Total LOC Estimate:** ~500 LOC (implementation + tests)
**Risk Level:** Low (additive changes, no breaking modifications)

## Sprint Goal

Make AILANG traces immediately useful for debugging by adding human-readable context to all spans, including error messages, code previews, and key identifiers.

## Prerequisites

- [x] M-OTEL-EXTENDED complete (base tracing infrastructure)
- [x] Benchmark ID in span names (completed earlier today)
- [x] Resource-level `process.cwd` attribute (completed earlier today)

## Milestones

### M1: Telemetry Helper Functions (~100 LOC, 1 hour)

**Goal:** Create reusable helper functions for safe attribute values

**Files:**
- Create `internal/telemetry/helpers.go` (~60 LOC)
- Create `internal/telemetry/helpers_test.go` (~80 LOC)

**Tasks:**
1. Implement `Truncate(s string, maxLen int) string` - Safe truncation with "..." suffix
2. Implement `CategorizeError(err error) string` - Returns error category for filtering
3. Implement `ShortHash(content string, length int) string` - SHA256-based short hash
4. Implement `LineSnippet(source string, lineNum int, maxLen int) string` - Extract code snippet
5. Write unit tests for all helper functions

**Acceptance Criteria:**
- [ ] All helper functions have 100% test coverage
- [ ] Truncate handles UTF-8 correctly (no broken characters)
- [ ] CategorizeError returns one of: parse_error, type_error, module_error, runtime_error, api_error, timeout, unknown
- [ ] ShortHash produces deterministic output

---

### M2: Error Context Enhancement (~80 LOC, 45 min)

**Goal:** Add error.message and error.category to all error spans

**Files to Modify:**
- `internal/pipeline/pipeline_single.go` - Parse/typecheck errors
- `internal/pipeline/pipeline_module.go` - Module load errors
- `cmd/ailang/eval_suite.go` - Benchmark errors
- `internal/ai/openai/client.go` - API errors
- `internal/ai/anthropic/client.go` - API errors
- `internal/ai/gemini/client.go` - API errors

**Pattern to apply everywhere:**
```go
if err != nil {
    span.SetAttributes(
        attribute.String("error.message", telemetry.Truncate(err.Error(), 200)),
        attribute.String("error.category", telemetry.CategorizeError(err)),
    )
    span.RecordError(err)
}
```

**Acceptance Criteria:**
- [ ] All error spans have error.message attribute
- [ ] All error spans have error.category attribute
- [ ] Error messages are truncated to 200 chars max
- [ ] `ailang trace view` shows readable error context

---

### M3: Eval/Benchmark Improvements (~100 LOC, 45 min)

**Goal:** Add debugging context to benchmark spans

**Files:**
- `cmd/ailang/eval_suite.go`

**Attributes to Add:**
- `error.summary` - First 200 chars of error output on failure
- `code.hash` - 8-char hash of generated code for dedup
- `code.preview` - First 100 chars of generated code on success
- `repair.attempts` - Number of self-repair attempts
- `repair.last_error` - Last repair error message (truncated)

**Acceptance Criteria:**
- [ ] Failed benchmarks show error.summary in trace
- [ ] Successful benchmarks show code.preview
- [ ] Self-repair attempts tracked in attributes
- [ ] Code hash enables deduplication of similar failures

---

### M4: AI Provider Improvements (~120 LOC, 45 min)

**Goal:** Add prompt/response context to AI spans

**Files:**
- `internal/ai/openai/client.go`
- `internal/ai/anthropic/client.go`
- `internal/ai/gemini/client.go`
- `internal/ai/ollama/client.go`

**Attributes to Add:**
- `prompt.preview` - First 100 chars of system prompt
- `prompt.user_length` - Length of user prompt (int)
- `response.preview` - First 100 chars of response
- `response.finish_reason` - Why generation stopped

**Acceptance Criteria:**
- [ ] All 4 AI providers have prompt/response previews
- [ ] Previews are safely truncated (no PII leakage beyond 100 chars)
- [ ] Finish reason captured for debugging

---

### M5: Compiler/CLI Improvements (~100 LOC, 45 min)

**Goal:** Add debugging context to compilation and CLI spans

**Files:**
- `internal/pipeline/pipeline_single.go`
- `internal/pipeline/pipeline_module.go`
- `cmd/ailang/main.go`

**Attributes to Add:**

**compile.parse (on success):**
- `ast.declarations` - Number of declarations parsed
- `ast.imports` - Number of imports

**compile.parse (on error):**
- `error.line` - Line:column of error
- `error.snippet` - Code snippet around error

**compile.load:**
- `modules.search_paths` - Paths searched for module (StringSlice)
- `modules.resolved_path` - Final resolved path

**ailang run:**
- `entry.function` - Entry point name
- `caps.granted` - Capabilities granted (StringSlice)
- `modules.total` - Total modules loaded

**Acceptance Criteria:**
- [ ] Parse errors show line/column and code snippet
- [ ] Module resolution shows search paths for LDR001 debugging
- [ ] CLI run shows entry point and capabilities

---

## Verification

After all milestones:

```bash
# Run quick eval with tracing
OTLP_GOOGLE_CLOUD_PROJECT=multivac-internal-dev \
  ./bin/ailang eval-suite --models gpt5-mini --benchmarks fizzbuzz

# Check traces have new attributes
./bin/ailang trace list --hours 1
./bin/ailang trace view <trace-id>
```

**Expected output improvements:**
```
eval.benchmark: fizzbuzz (22s)
    benchmark.success: true
    code.preview: "module benchmark/solution\n\nlet fizzbu..."
    code.hash: "a1b2c3d4"
    error.category: none
```

## Success Metrics

- [ ] All tests pass (`make test`)
- [ ] Linting passes (`make lint`)
- [ ] Error spans show readable error.message
- [ ] Benchmark traces show code.preview or error.summary
- [ ] AI provider traces show prompt/response previews
- [ ] Module resolution traces show search paths
