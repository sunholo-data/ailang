# M-OTEL-ENHANCED-TRACING-DX: Human-Friendly Tracing for Debugging

**Status:** Planned
**Target:** v0.6.3
**Priority:** P1
**Estimated:** 2 days
**Dependencies:** M-OTEL-EXTENDED (complete)
**Created:** 2026-01-03

## Problem Statement

Current AILANG tracing captures technical execution data but lacks the human-readable context needed for effective debugging. When investigating issues like:
- Why did a benchmark fail?
- Where is the type error coming from?
- What module resolution path was used?

...users must correlate multiple spans and decode technical attributes manually.

### Current Tracing Audit

**23 total spans across 8 areas:**

| Area | Spans | What's Captured | What's Missing |
|------|-------|-----------------|----------------|
| **Compiler Pipeline** | 6 spans | file.path, file.size_bytes, is_repl | Error details, AST node counts, import paths searched |
| **AI Providers** | 4 spans | model, tokens, cost | Truncated prompt preview, response preview, retry count |
| **Eval Suite** | 2 spans | benchmark.id, success, model | Error message text, generated code hash, repair attempts |
| **Executor (Agents)** | 2 spans | task directive, workspace | Files created/modified, command count, session ID |
| **Messaging** | 3 spans | inbox, from_agent, type | Payload preview (first 100 chars), GitHub issue # |
| **ailang run** | 1 span | filename | Entry point, caps granted, module count |
| **Module Pipeline** | 3 spans | modules.loaded, modules.sorted | Import graph depth, circular dep check |
| **Search** | 1 span | query, use_neural | Result count, threshold, time_to_first_result |

### Current Resource Attributes (ALL spans)

```
service.name          → "ailang-compiler" | "ailang-eval" | etc.
service.version       → AILANG version (e.g., "v0.6.2")
deployment.environment → "development" | "production"
process.runtime.name   → "go"
process.runtime.version → Go version
process.cwd            → Working directory (critical for module resolution!)
```

## Goals

**Primary Goal:** Make traces immediately useful for debugging without additional tooling

**Success Metrics:**
1. Human can identify issue root cause from trace in <30 seconds for common bugs
2. All error spans include error.message with actionable text
3. Key identifiers (file names, benchmark IDs, module paths) visible in span names
4. No more than 10 attributes per span (avoid noise)

## Solution Design

### Principle: "Span Name = Summary, Attributes = Details"

Span names should be immediately informative:
- ✅ `eval.benchmark: fizzbuzz` (shows WHAT)
- ✅ `compile: examples/hello.ail` (shows WHICH file)
- ❌ `eval.benchmark` (generic, requires drilling into attributes)

### Phase 1: Error Context Enhancement

**Add to ALL error spans:**

```go
span.SetAttributes(
    attribute.String("error.message", truncate(err.Error(), 200)),
    attribute.String("error.category", categorizeError(err)), // "parse_error" | "type_error" | "runtime_error"
)
```

**Error categories for filtering:**
- `parse_error` - Syntax issues
- `type_error` - Type inference/checking failures
- `module_error` - Import/resolution failures
- `runtime_error` - Execution failures
- `api_error` - External API failures
- `timeout` - Operation timeouts

### Phase 2: Compiler Tracing Improvements

**compile.parse:**
```go
// Add on success:
attribute.Int("ast.declarations", len(file.Statements))
attribute.Int("ast.imports", len(file.Imports))

// Add on error:
attribute.String("error.line", fmt.Sprintf("%d:%d", pos.Line, pos.Col))
attribute.String("error.snippet", getLineSnippet(src, pos.Line, 60))
```

**compile.typecheck:**
```go
// Add on success:
attribute.Int("types.inferred", typeChecker.InferredCount)
attribute.Int("constraints.solved", typeChecker.ConstraintsSolved)

// Add on error:
attribute.String("type.expected", expectedType.String())
attribute.String("type.actual", actualType.String())
attribute.String("error.location", fmt.Sprintf("%s:%d:%d", file, line, col))
```

**compile.load (module pipeline):**
```go
attribute.StringSlice("modules.search_paths", searchPaths)
attribute.String("modules.resolved_path", resolvedPath)
attribute.Int("modules.depth", importDepth)
```

### Phase 3: Eval/Benchmark Improvements

**eval.benchmark: {id}:**
```go
// On failure, add these:
attribute.String("error.summary", truncate(errorOutput, 200))
attribute.String("code.hash", shortHash(generatedCode, 8)) // For dedup
attribute.Int("repair.attempts", repairAttempts)
attribute.String("repair.last_error", truncate(lastRepairError, 100))

// On success, add:
attribute.String("code.preview", truncate(generatedCode, 100)) // First 100 chars
```

### Phase 4: AI Provider Improvements

**openai.generate / anthropic.generate / gemini.generate:**
```go
// Request context:
attribute.String("prompt.preview", truncate(systemPrompt, 100))
attribute.Int("prompt.user_length", len(userPrompt))

// Response context:
attribute.String("response.preview", truncate(response, 100))
attribute.Int("response.finish_reason", finishReason)

// Retry tracking:
attribute.Int("retry.count", retries)
attribute.String("retry.last_error", truncate(lastRetryError, 100))
```

### Phase 5: CLI Run Improvements

**ailang run: {filename}:**
```go
attribute.String("entry.function", entryPoint)
attribute.StringSlice("caps.granted", capabilities)
attribute.Int("modules.total", moduleCount)
attribute.Int64("runtime.duration_ms", runtimeMs)
```

### Implementation Checklist

**Files to Modify:**

| File | Changes |
|------|---------|
| `internal/pipeline/pipeline_single.go` | Add parse/typecheck error details |
| `internal/pipeline/pipeline_module.go` | Add module resolution paths |
| `cmd/ailang/eval_suite.go` | Add error summaries, code previews |
| `internal/ai/openai/client.go` | Add prompt/response previews, retry info |
| `internal/ai/anthropic/client.go` | Add prompt/response previews, retry info |
| `internal/ai/gemini/client.go` | Add prompt/response previews, retry info |
| `internal/ai/ollama/client.go` | Add prompt/response previews |
| `internal/executor/claude/claude.go` | Add files modified, command count |
| `internal/messaging/inbox.go` | Add payload preview, GitHub # |
| `cmd/ailang/main.go` | Add entry point, caps, module count |

**Helper functions to add (`internal/telemetry/helpers.go`):**
```go
// Truncate safely for span attributes (avoids huge spans)
func Truncate(s string, maxLen int) string

// Categorize error types for filtering
func CategorizeError(err error) string

// Short hash for deduplication
func ShortHash(content string, length int) string

// Get line snippet for error context
func LineSnippet(source string, lineNum int, maxLen int) string
```

### Attribute Naming Convention

Follow OpenTelemetry semantic conventions where applicable:

| Pattern | Example | Use For |
|---------|---------|---------|
| `error.*` | `error.message`, `error.category` | Error details |
| `file.*` | `file.path`, `file.size_bytes` | File metadata |
| `code.*` | `code.preview`, `code.hash` | Generated code |
| `prompt.*` | `prompt.preview`, `prompt.user_length` | AI prompts |
| `response.*` | `response.preview`, `response.finish_reason` | AI responses |
| `benchmark.*` | `benchmark.id`, `benchmark.success` | Eval benchmarks |
| `modules.*` | `modules.count`, `modules.search_paths` | Module system |

### Privacy Considerations

**Truncation is mandatory for:**
- User code (may contain secrets)
- AI prompts (may contain user data)
- AI responses (may contain sensitive output)
- Error messages (may leak paths/data)

**Max lengths:**
- Previews: 100 characters
- Error messages: 200 characters
- Paths: Full (but exclude home directory in logs)

## Testing Strategy

1. **Unit tests for helper functions** - Truncate, CategorizeError, ShortHash
2. **Integration tests** - Verify spans have expected attributes
3. **Manual verification** - Run `ailang trace list/view` after eval runs

## Success Criteria

- [ ] Error spans include `error.message` with readable text
- [ ] Benchmark failures show `error.summary` with problem description
- [ ] AI calls show truncated prompt/response previews
- [ ] Module resolution shows `modules.search_paths` on LDR001 errors
- [ ] Type errors show `type.expected` vs `type.actual`
- [ ] `ailang trace view` output is immediately understandable by humans

## Future Work

After this DX enhancement:
1. **Structured span events** - Add span events for intermediate steps
2. **Trace sampling** - Sample based on error status (100% errors, 10% success)
3. **Dashboard integration** - Show traces in Collaboration Hub
4. **Local trace viewer** - `ailang trace ui` for terminal-based exploration

## Related Documents

- [M-OTEL-EXTENDED-INSTRUMENTATION](m-otel-extended-instrumentation.md) - Base tracing infrastructure
- [trace_patterns.md](../../.claude/skills/trace-debugger/resources/trace_patterns.md) - Current tracing patterns
