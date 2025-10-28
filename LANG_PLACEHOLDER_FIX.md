# Python Eval Agent Mode Bug Fix

## Problem

Python benchmarks in agent mode were generating AILANG code instead of Python code, causing syntax errors:

```json
{
  "id": "adt_option",
  "lang": "python",
  "stderr": "  File \"/var/folders/.../eval_246810308.py\", line 1\n    module benchmark/solution\n           ^^^^^^^^^\nSyntaxError: invalid syntax\n"
}
```

The AI model was generating `module benchmark/solution` (AILANG syntax) when it should have generated Python code.

## Root Cause

In [internal/eval_harness/agent_prompt.go:473-478](internal/eval_harness/agent_prompt.go#L473-L478), the code replaced several placeholders in benchmark task prompts (`{{DESCRIPTION}}`, `{{EXPECTED_OUTPUT}}`, etc.), but **never replaced the `<LANG>` placeholder**.

31 out of 35 benchmarks use `<LANG>` in their `task_prompt` field, e.g.:
```yaml
task_prompt: |
  Write a program in <LANG> that implements an Option type...
```

Without replacement, prompts said "Write a program in <LANG>" instead of "Write a program in Python", so the AI defaulted to AILANG (the project's primary language).

## The Fix

Added `<LANG>` placeholder replacement in `GenerateAgentPromptsWithSystemPrompt()`:

```go
// Replace <LANG> placeholder with actual language name (used in 31/35 benchmarks)
// e.g., "Write a program in <LANG>" → "Write a program in Python"
languageName := language
if language == "python" {
	languageName = "Python"
} else if language == "ailang" {
	languageName = "AILANG"
}
taskPrompt = strings.ReplaceAll(taskPrompt, "<LANG>", languageName)
```

## Verification

### System Prompt Loading (Already Correct)

The `LoadSystemPromptForLanguage()` function correctly loads language-specific teaching prompts:
- Python → `prompts/python.md` (Python guidelines)
- AILANG → Active version from `prompts/versions.json` (currently v0.3.23)

This part was working correctly all along.

### Tests Added

1. **`TestLanguagePlaceholderReplacement`**: Verifies `<LANG>` is replaced with "Python" or "AILANG" appropriately
2. **`TestSystemPromptLanguageSeparation`**: Verifies Python gets Python system prompt (not AILANG prompt)

Both tests pass.

## Impact

This fix affects **31 out of 35 benchmarks** that use the `<LANG>` placeholder:
- `adt_option`, `api_call_json`, `fizzbuzz`, `recursion_*`, etc.

Before: Python eval agent mode generated AILANG code (100% failures)
After: Python eval agent mode generates Python code (expected to work correctly)

## Files Changed

- `internal/eval_harness/agent_prompt.go`: Added `<LANG>` replacement (5 lines)
- `internal/eval_harness/agent_prompt_test.go`: Added 2 comprehensive tests (113 lines)

## Related Issues

This bug existed since agent mode was introduced but only affected multi-language benchmarks. Simple benchmarks without `<LANG>` in their prompts (like `simple_print`) worked fine, which is why the bug wasn't caught earlier.
