# Benchmark Manager Reference

Complete reference for AILANG evaluation benchmarks.

## Benchmark YAML Schema

### All Available Fields

```yaml
# Required fields
id: string                    # Unique identifier (snake_case)
description: string           # Human-readable description
languages: [string]           # Languages to test: ["python", "ailang"]
entrypoint: string            # Function to call (usually "main")
caps: [string]                # Required capabilities
expected_stdout: string       # Exact expected output (with trailing newline)

# Prompt fields (use ONE)
task_prompt: string           # RECOMMENDED: Appends to teaching prompt
prompt: string                # NOT RECOMMENDED: Replaces teaching prompt entirely

# Metadata fields
difficulty: string            # "easy", "medium", "hard"
expected_gain: string         # "low", "medium", "high"
timeout: int                  # Execution timeout in seconds (default: 30)
skip_python: bool             # Skip Python baseline (default: false)
```

## Capability Reference

| Capability | Description | Example Use Case |
|------------|-------------|------------------|
| `IO` | Standard I/O (print, read) | Any program that outputs |
| `FS` | File system access | Reading/writing files |
| `Clock` | Time operations | Getting current time, sleep |
| `Net` | Network access | HTTP requests, API calls |

### Capability Combinations

```yaml
# Console output only
caps: ["IO"]

# HTTP API calls
caps: ["Net", "IO"]

# File manipulation
caps: ["FS", "IO"]

# Time-based operations
caps: ["Clock", "IO"]

# Full access (rare)
caps: ["IO", "FS", "Clock", "Net"]
```

## Example Benchmarks

### Simple IO (Easy)

```yaml
id: hello_world
description: "Print Hello, World!"
languages: ["python", "ailang"]
entrypoint: "main"
caps: ["IO"]
difficulty: "easy"
expected_gain: "low"
task_prompt: |
  Write a program in <LANG> that prints "Hello, World!" to stdout.

  Output only the code, no explanations.
expected_stdout: |
  Hello, World!
```

### JSON Parsing (Medium)

```yaml
id: json_parse
description: "Parse JSON array, filter, and output"
languages: ["python", "ailang"]
entrypoint: "main"
caps: ["IO"]
difficulty: "medium"
expected_gain: "medium"
task_prompt: |
  Write a program in <LANG> that:
  1. Parses this JSON array: [{"name":"Alice","age":30},{"name":"Bob","age":25}]
  2. Filters to keep only people aged 30 or older
  3. Prints the names, one per line

  Output only the code, no explanations.
expected_stdout: |
  Alice
```

### HTTP Request (Hard)

```yaml
id: api_call_json
description: "Make HTTP POST with JSON payload"
languages: ["python", "ailang"]
entrypoint: "main"
caps: ["Net", "IO"]
difficulty: "hard"
expected_gain: "high"
task_prompt: |
  Write a program in <LANG> that:
  1. Makes an HTTP POST request to https://httpbin.org/post
  2. Includes headers: "Content-Type: application/json"
  3. Sends body: {"message":"Hello","count":42}
  4. Prints ONLY the response status code

  Output only the code, no explanations.
expected_stdout: |
  200
```

## Prompt Field Behavior

### task_prompt (Recommended)

The `task_prompt` field is **appended** to the AILANG teaching prompt:

```
[Teaching prompt content - AILANG syntax, examples, etc.]

## Task

[Your task_prompt content here]
```

This ensures models know AILANG syntax before attempting the task.

### prompt (Not Recommended)

The `prompt` field **replaces** the teaching prompt entirely:

```
[Only your prompt content - no AILANG teaching]
```

Use only when:
- Testing raw model capability without language teaching
- Creating language-agnostic benchmarks
- Intentionally excluding AILANG-specific guidance

## Testing Workflow

### 1. Local Validation

```bash
# Check for common issues
.claude/skills/benchmark-manager/scripts/check_benchmark.sh benchmarks/my_test.yml
```

### 2. Single Model Test

```bash
# Test with cheap model
.claude/skills/benchmark-manager/scripts/test_benchmark.sh my_test claude-haiku-4-5
```

### 3. Full Suite

```bash
# Dev models (3 cheap models)
ailang eval-suite --benchmarks my_test

# All models
ailang eval-suite --full --benchmarks my_test
```

## Troubleshooting Matrix

| Symptom | Likely Cause | Fix |
|---------|--------------|-----|
| 0% pass rate | Using `prompt:` instead of `task_prompt:` | Change to `task_prompt:` |
| Python-like code | Teaching prompt not seen | Use `task_prompt:` |
| Template copied verbatim | Ambiguous task | Clarify task_prompt |
| `print(42)` errors | Models don't know `show()` | Improve teaching prompt |
| `%` operator errors | Models don't know `mod_Int` | Improve teaching prompt |
| Results unchanged | Binary not rebuilt | Run `make quick-install` |

## File Locations

| Item | Location |
|------|----------|
| Benchmark definitions | `benchmarks/*.yml` |
| Eval results | `eval_results/` |
| Teaching prompts | `prompts/v*.md` |
| Models config | `internal/eval_harness/models.yml` |
| Eval harness code | `internal/eval_harness/` |

## Related Documentation

- [Eval Architecture](docs/docs/guides/evaluation/architecture.md)
- [Models Configuration](internal/eval_harness/models.yml)
- [Teaching Prompt](prompts/v0.4.8.md)
