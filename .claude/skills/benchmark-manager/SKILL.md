---
name: benchmark-manager
description: Create and manage AILANG eval benchmarks. Use when user asks to create benchmarks, fix benchmark issues, debug failing benchmarks, or analyze benchmark results.
---

# Benchmark Manager

Manage AILANG evaluation benchmarks with correct prompt integration, debugging workflows, and best practices learned from real benchmark failures.

## Quick Start

**Debugging a failing benchmark:**
```bash
# 1. Show the full prompt that models see
.claude/skills/benchmark-manager/scripts/show_full_prompt.sh json_parse

# 2. Test a benchmark with a specific model
ailang eval-suite --models claude-haiku-4-5 --benchmarks json_parse

# 3. Check benchmark YAML for common issues
.claude/skills/benchmark-manager/scripts/check_benchmark.sh benchmarks/json_parse.yml
```

## When to Use This Skill

Invoke this skill when:
- User asks to create a new benchmark
- User asks to debug/fix a failing benchmark
- User wants to understand why models generate wrong code
- User asks about benchmark YAML format
- Benchmarks show 0% pass rate despite language support

## CRITICAL: prompt vs task_prompt

**This is the most important concept for benchmark management.**

### The Problem (v0.4.8 Discovery)

Benchmarks have TWO different prompt fields with VERY different behavior:

| Field | Behavior | Use When |
|-------|----------|----------|
| `prompt:` | **REPLACES** the teaching prompt entirely | Testing raw model capability (rare) |
| `task_prompt:` | **APPENDS** to teaching prompt | Normal benchmarks (99% of cases) |

### Why This Matters

```yaml
# BAD - Model never sees AILANG syntax!
prompt: |
  Write a program that prints "Hello"

# GOOD - Model sees teaching prompt + task
task_prompt: |
  Write a program that prints "Hello"
```

With `prompt:`, models generate Python/pseudo-code because they never learn AILANG syntax.

### How Prompts Combine

From `internal/eval_harness/spec.go` (lines 91-93):
```go
fullPrompt := basePrompt  // Teaching prompt from prompts/v0.4.x.md
if s.TaskPrompt != "" {
    fullPrompt = fullPrompt + "\n\n## Task\n\n" + s.TaskPrompt
}
```

The teaching prompt teaches AILANG syntax; `task_prompt` adds the specific task.

## Available Scripts

### `scripts/show_full_prompt.sh`

Shows the complete prompt that models receive for a benchmark.

**Usage:**
```bash
.claude/skills/benchmark-manager/scripts/show_full_prompt.sh <benchmark_id>

# Example:
.claude/skills/benchmark-manager/scripts/show_full_prompt.sh json_parse
```

### `scripts/check_benchmark.sh`

Validates a benchmark YAML file for common issues.

**Usage:**
```bash
.claude/skills/benchmark-manager/scripts/check_benchmark.sh benchmarks/<name>.yml
```

**Checks for:**
- Using `prompt:` instead of `task_prompt:` (warning)
- Missing required fields (including `tags` — enforced by LoadSpec in v0.14.0+)
- Invalid `tier` (must be smoke / core / stretch / vision; missing = warning, defaults to core)
- Non-canonical tags (validated against the 12-tag taxonomy in `internal/eval_harness/spec.go`)
- `tags` list length > 3 (hard limit from LoadSpec)
- Invalid capability names
- Syntax errors in YAML

### `scripts/test_benchmark.sh`

Runs a quick single-model test of a benchmark.

**Usage:**
```bash
.claude/skills/benchmark-manager/scripts/test_benchmark.sh <benchmark_id> [model]

# Examples:
.claude/skills/benchmark-manager/scripts/test_benchmark.sh json_parse
.claude/skills/benchmark-manager/scripts/test_benchmark.sh json_parse claude-haiku-4-5
```

## Benchmark YAML Format

### Required Fields

```yaml
id: my_benchmark              # Unique identifier (snake_case)
description: "Short description of what this tests"
languages: ["python", "ailang"]
entrypoint: "main"            # Function to call
caps: ["IO"]                  # Required capabilities
difficulty: "easy|medium|hard"
expected_gain: "low|medium|high"
tier: core                    # smoke | core | stretch | vision  (v0.14.0+; defaults to core)
tags: [type_safety, functional]   # 1-3 tags from the 12-tag canonical taxonomy
task_prompt: |                # ALWAYS use task_prompt, not prompt!
  Write a program in <LANG> that:
  1. Does something
  2. Prints the result

  Output only the code, no explanations.
expected_stdout: |            # Exact expected output
  expected output here
```

### Tier + Tags (v0.14.0+)

Every benchmark has a **tier** (execution budget) and up to **3 tags** (capability classification).
`LoadSpec` rejects benchmarks that violate the taxonomy, so these are not optional for new benchmarks.

**Tiers:**
- `smoke` — sanity checks; should never fail. Run in PR CI.
- `core` — headline metric. The "AILANG vs Python" rate is computed from Core.
- `stretch` — harder benchmarks; mixed pass/fail expected.
- `vision` — research-grade; expect low AILANG pass rate. Only runs with `--full`.

**Canonical tags (12):** `adt_pattern_match`, `algorithmic`, `contracts`, `data_transform`, `effects_io`,
`error_handling`, `functional`, `records`, `recursion`, `state_machine`, `string_algo`, `type_safety`.

**Governance:** pick tier/tags to match *how the benchmark will be used for rotation* — see
`benchmarks/CURATION.md` for the promotion/demotion workflow.

### Capability Names

Valid capabilities: `IO`, `FS`, `Clock`, `Net`

```yaml
# File I/O
caps: ["IO"]

# HTTP requests
caps: ["Net", "IO"]

# File system operations
caps: ["FS", "IO"]
```

## Creating New Benchmarks

### Step 1: Determine Requirements

- What language feature/capability is being tested?
- Can models solve this with just the teaching prompt?
- What's the expected output?

### Step 2: Write the Benchmark

```yaml
id: my_new_benchmark
description: "Test feature X capability"
languages: ["python", "ailang"]
entrypoint: "main"
caps: ["IO"]
difficulty: "medium"
expected_gain: "medium"
tier: core
tags: [functional, data_transform]    # pick 1-3 canonical tags
task_prompt: |
  Write a program in <LANG> that:
  1. Clear description of task
  2. Another step
  3. Print the result

  Output only the code, no explanations.
expected_stdout: |
  exact expected output
```

### Step 2b: Rotation Mindset

AILANG's benchmark suite is **curated, not accumulated** — we demote saturated benchmarks
and promote ones that reveal real gaps. Before adding, check where you'll sit in rotation:

```bash
ailang eval-matrix --show-saturated     # Python ≥ 95% AND AILANG ≥ 95% — demote candidates
ailang eval-matrix --ailang-wins        # AILANG ≥ Python by ≥ 10pp — keep as value evidence
ailang eval-matrix --by-tags            # gap distribution across the 12-tag taxonomy
```

A good new benchmark: (a) lives in an under-covered tag, (b) isn't already saturated by other
languages, and (c) has a clear expected gap (AILANG wins or AILANG loses but we can fix it).

### Step 3: Validate and Test

```bash
# Check for issues
.claude/skills/benchmark-manager/scripts/check_benchmark.sh benchmarks/my_new_benchmark.yml

# Test with cheap model first
ailang eval-suite --models claude-haiku-4-5 --benchmarks my_new_benchmark
```

## Debugging Failing Benchmarks

### Symptom: 0% Pass Rate Despite Language Support

**Check 1: Is it using `task_prompt:`?**
```bash
grep -E "^prompt:" benchmarks/failing_benchmark.yml
# If this returns a match, change to task_prompt:
```

**Check 2: What prompt do models see?**
```bash
.claude/skills/benchmark-manager/scripts/show_full_prompt.sh failing_benchmark
```

**Check 3: Is the teaching prompt up to date?**
```bash
# After editing prompts/v0.x.x.md, you MUST rebuild:
make quick-install
```

### Symptom: Models Copy Template Instead of Solving Task

The teaching prompt includes a template structure. If models copy it verbatim:

1. Make sure task is clearly different from examples in teaching prompt
2. Check that `task_prompt` explicitly describes what to do
3. Consider if the task description is ambiguous

### Symptom: compile_error on Valid Syntax

Common AILANG-specific issues models get wrong:

| Wrong | Correct | Notes |
|-------|---------|-------|
| `print(42)` | `print(show(42))` | print expects string |
| `a % b` | `mod_Int(a, b)` | No % operator |
| `def main()` | `export func main()` | Wrong keyword |
| `for x in xs` | `match xs { ... }` | No for loops |

If models consistently make these mistakes, the teaching prompt needs improvement (use prompt-manager skill).

## Common Mistakes

### 1. Using `prompt:` Instead of `task_prompt:`

```yaml
# WRONG - Models never see AILANG syntax
prompt: |
  Write code that...

# CORRECT - Teaching prompt + task
task_prompt: |
  Write code that...
```

### 2. Forgetting to Rebuild After Prompt Changes

```bash
# After editing prompts/v0.x.x.md:
make quick-install  # REQUIRED!
```

### 3. Putting Hints in Benchmarks

```yaml
# WRONG - Hints in benchmark
task_prompt: |
  Write code that prints 42.
  Hint: Use print(show(42)) in AILANG.

# CORRECT - No hints; if models fail, fix the teaching prompt
task_prompt: |
  Write code that prints 42.
```

If models need AILANG-specific hints, the teaching prompt is incomplete. Use the prompt-manager skill to fix it.

### 4. Testing Too Many Models at Once

```bash
# WRONG - Expensive and slow for debugging
ailang eval-suite --full --benchmarks my_test

# CORRECT - Use one cheap model first
ailang eval-suite --models claude-haiku-4-5 --benchmarks my_test
```

## Resources

### Reference Guide
See [`resources/reference.md`](resources/reference.md) for:
- Complete list of valid benchmark fields
- Capability reference
- Example benchmarks

### Related Skills
- **prompt-manager**: When benchmark failures indicate teaching prompt issues
- **eval-analyzer**: For analyzing results across many benchmarks
- **use-ailang**: For writing correct AILANG code
- **devtools-prompt**: For toolchain docs (debugging, tracing, eval workflows) — `ailang devtools-prompt`

## Notes

- Benchmarks live in `benchmarks/` directory
- Eval results go to `eval_results/` directory
- Teaching prompts are embedded in binary - rebuild after changes (`ailang prompt` for syntax, `ailang devtools-prompt` for toolchain)
- Use `<LANG>` placeholder in task_prompt - it's replaced with "AILANG" or "Python"
