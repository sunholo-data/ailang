# M-EVAL: CLI Arguments & Language Capability Checking

**Status**: Planned
**Target**: v0.3.24
**Priority**: P1 (Medium) - Improves agent eval accuracy
**Estimated**: 1 day (4-6 hours)
**Dependencies**: None

## AI-First Alignment Check

**Score this feature against AILANG's core principles:**

| Principle | Impact | Score | Notes |
|-----------|--------|-------|-------|
| Reduce Syntactic Noise | Neutral | 0 | Doesn't change AILANG syntax |
| Preserve Semantic Clarity | Positive | +1 | Makes benchmark requirements explicit (test inputs, CLI args) |
| Increase Determinism | Positive | +1 | Tests become fully reproducible (fixed inputs, no file searching) |
| Lower Token Cost | Positive | +1 | Agents don't waste 30+ turns exploring missing features |
| **Net Score** | | **+3** | **Decision: Move forward** |

**Decision rule:** Net score > +1 → Move forward | ≤ 0 → Reject or redesign

**Reference:** See [AI-first DX philosophy](../v0_3_15/example-parity-vision-alignment.md#-design-principle-ai-first-dx)

## Problem Statement

**Current State:**
- **cli_args benchmark broken**: Agent eval shows AILANG took 80 turns ($0.15), Python failed with wrong output
- **Missing CLI argument support**: Python solutions expect `sys.argv`, AILANG has no `std/os` module
- **Agent wastes turns**: Spent 30 turns searching for non-existent test_input.txt files
- **No capability checks**: Agents try unsupported features (list pattern matching, file I/O) without early failure

**Impact:**
- **Agent efficiency**: 80 turns vs 10-15 expected (5-8x overhead)
- **Cost**: $0.15 vs $0.03 expected (5x cost)
- **False positives**: AILANG "succeeded" by hardcoding `print("15")` instead of solving the problem
- **False negatives**: Python solution correctly implemented CLI args but harness doesn't pass them

**Root Cause Analysis:**
1. Benchmark YAML has no way to specify test inputs or CLI arguments
2. Runner executes `python3 solution.py` and `ailang run solution.ail` with no args
3. Agent templates don't warn about missing language features
4. No way to mark benchmarks as "unsupported" for specific languages

## Goals

**Primary Goal:** Make agent evals reliable, efficient, and accurate by supporting CLI arguments and early capability checking

**Success Metrics:**
- cli_args benchmark: AILANG and Python both succeed with <20 turns
- Agent turns reduced by 50-70% for I/O-heavy benchmarks
- Zero false positives (solutions must actually solve the problem, not cheat)
- Clear "unsupported" status for language/benchmark combinations that don't work yet

## Solution Design

### Overview

Add two complementary features to the eval harness:

1. **CLI Argument Support**: Benchmarks can specify test inputs and command-line arguments
2. **Language Capability Checks**: Agent templates include upfront warnings about missing features

### Architecture

**Components:**

1. **Benchmark Specification** (`BenchmarkSpec`):
   - Add `test_input` field (optional): Test data to create as files
   - Add `cli_args` field (optional): Arguments to pass when running solution
   - Add `unsupported_langs` field (optional): Languages that can't solve this benchmark yet

2. **Test Input Setup** (agent_runner.go):
   - Create test input files in workspace before agent starts
   - Pass filenames/paths in agent prompt

3. **CLI Argument Passing** (runner.go):
   - Modify `executeCode()` to accept CLI args
   - Pass to `python3 solution.py <args>` and `ailang run ... solution.ail <args>`

4. **Agent Template Updates**:
   - Add "Language Capabilities" section warning about missing features
   - Add "Test Inputs" section showing available files
   - Add "CLI Arguments" section showing what arguments harness will pass

### Implementation Plan

**Phase 1: Benchmark Specification** (~2 hours)
- [ ] Add fields to `BenchmarkSpec` struct in spec.go
- [ ] Update YAML parsing to handle new fields
- [ ] Add validation (cli_args requires test_input)
- [ ] Update spec_test.go with examples

**Phase 2: Test Input & CLI Args** (~2 hours)
- [ ] Update `agent_runner.go` to create test input files
- [ ] Modify `executeCode()` in runner.go to accept args parameter
- [ ] Update Python execution: `exec.Command("python3", append([]string{"solution.py"}, args...)...)`
- [ ] Update AILANG execution: `exec.Command("ailang", append([]string{"run", "--entry", "main", "--caps", caps, "solution.ail"}, args...)...)`
- [ ] Test with cli_args benchmark

**Phase 3: Agent Template Updates** (~1 hour)
- [ ] Update `agent_task_ailang.txt` with capability warnings
- [ ] Update `agent_task_python.txt` with test input info
- [ ] Add conditional sections (only show if test_input/cli_args present)
- [ ] Update agent_prompt.go template rendering

**Phase 4: cli_args Benchmark Fix** (~1 hour)
- [ ] Create `benchmarks/test_data/cli_args_input.txt` with `5\n10`
- [ ] Update `benchmarks/cli_args.yml` with test_input and cli_args fields
- [ ] Test with agent eval (both Python and AILANG)
- [ ] Mark AILANG as unsupported if it doesn't have CLI arg support yet

### Files to Modify/Create

**New files:**
- `benchmarks/test_data/cli_args_input.txt` - Test input file (~2 lines)

**Modified files:**
- `internal/eval_harness/spec.go` - Add fields to BenchmarkSpec (~20 LOC)
- `internal/eval_harness/agent_runner.go` - Create test input files (~40 LOC)
- `internal/eval_harness/runner.go` - Pass CLI args to executeCode (~30 LOC)
- `internal/eval_harness/templates/agent_task_ailang.txt` - Add capability warnings (~50 LOC)
- `internal/eval_harness/templates/agent_task_python.txt` - Add test input info (~30 LOC)
- `internal/eval_harness/agent_prompt.go` - Conditional template rendering (~20 LOC)
- `benchmarks/cli_args.yml` - Add test_input, cli_args, unsupported_langs fields (~10 LOC)

**Total new code**: ~200 LOC

## Examples

### Example 1: cli_args Benchmark (Fixed)

**Before (broken):**
```yaml
# benchmarks/cli_args.yml
id: cli_args
description: "Read file from CLI argument, process, write result (IO + FS)"
languages: ["python", "ailang"]
caps: ["IO", "FS"]
expected_stdout: |
  15
```

Agent behavior:
- Searches for test_input.txt (doesn't exist)
- Tries multiple approaches
- Python: Implements CLI args but harness runs without args → fails
- AILANG: Gives up and hardcodes `print("15")` → false positive

**After (working):**
```yaml
# benchmarks/cli_args.yml
id: cli_args
description: "Read file from CLI argument, process, write result (IO + FS)"
languages: ["python", "ailang"]
caps: ["IO", "FS"]
test_input:
  input.txt: |
    5
    10
cli_args: ["input.txt"]
unsupported_langs: ["ailang"]  # No std/os module yet
expected_stdout: |
  15
```

Agent sees in prompt:
```
## Test Inputs

The harness has created these files in your workspace:
- input.txt (contents: "5\n10\n")

## CLI Arguments

When running your solution, the harness will pass: input.txt

Example:
  python3 solution.py input.txt

Your solution should read the filename from sys.argv[1].

## Language Capabilities

⚠️ AILANG Status: This benchmark is marked as UNSUPPORTED for AILANG.

Reason: AILANG does not yet have CLI argument support (no std/os module).

If you believe this is incorrect, you can attempt a solution, but it may not work.
```

Result:
- Python: Works correctly (10-15 turns, $0.02)
- AILANG: Marked as unsupported, skipped or attempted with clear warning

### Example 2: Agent Template Updates

**agent_task_ailang.txt (additions):**

```
{{if .UnsupportedLang}}
## ⚠️ Language Support Warning

This benchmark is marked as **UNSUPPORTED** for AILANG.

**Reason:** {{.UnsupportedReason}}

You can attempt a solution, but it may fail due to missing language features.
{{end}}

{{if .TestInputs}}
## Test Inputs

The following test files have been created in your workspace:

{{range .TestInputs}}
- **{{.Filename}}**:
  ```
  {{.Contents}}
  ```
{{end}}
{{end}}

{{if .CLIArgs}}
## Command-Line Arguments

When running your solution, the harness will pass these arguments:

```
ailang run --entry main --caps {{.Caps}} solution.ail {{.CLIArgs}}
```

**Arguments:** {{.CLIArgs}}

{{if .TestInputFilename}}
**Note:** The first argument is the test input filename: {{.TestInputFilename}}
{{end}}

⚠️ **AILANG Limitation:** AILANG does not currently have a standard library for reading CLI arguments (no `std/os` module). If this benchmark requires CLI args, it may be unsupported.
{{end}}
```

## Success Criteria

- [ ] `BenchmarkSpec` supports `test_input`, `cli_args`, `unsupported_langs` fields
- [ ] Agent runner creates test input files before agent starts
- [ ] Runner passes CLI args when executing Python solutions
- [ ] Runner passes CLI args when executing AILANG solutions (if supported)
- [ ] Agent templates show test inputs, CLI args, and capability warnings
- [ ] cli_args benchmark: Python succeeds with <20 turns
- [ ] cli_args benchmark: AILANG either marked unsupported OR succeeds if we add CLI arg support
- [ ] No false positives (agents can't cheat by hardcoding answers)
- [ ] All tests passing
- [ ] Documentation updated (agent eval README)

## Testing Strategy

**Unit tests:**
- spec_test.go: Parse new YAML fields
- agent_prompt_test.go: Render templates with test inputs and CLI args
- runner_test.go: Pass CLI args to execution commands

**Integration tests:**
- Test cli_args with Python (should succeed)
- Test cli_args with AILANG marked unsupported (should skip or warn)
- Test that test input files are created correctly
- Test that CLI args are passed in correct order

**Manual testing:**
- Run `ailang eval-suite --agent --benchmarks cli_args --langs python`
- Verify agent sees test_input.txt in workspace
- Verify solution is called with correct arguments
- Check agent transcript for capability warnings

## Non-Goals

**Not in this feature:**
- **AILANG CLI argument support** - Requires `std/os` module (separate feature)
- **Complex test input formats** - Only simple text files for now (JSON/binary later)
- **Environment variables** - Out of scope (could add in v0.4.0)
- **Interactive input** - stdin/stdout only, no terminal interaction

## Timeline

**Day 1** (6 hours):
- Phase 1: Benchmark specification (2h)
- Phase 2: Test input & CLI args (2h)
- Phase 3: Agent template updates (1h)
- Phase 4: cli_args benchmark fix (1h)

**Total: ~6 hours in 1 day**

## Risks & Mitigations

| Risk | Impact | Mitigation |
|------|--------|-----------|
| AILANG can't support CLI args yet | High | Mark as unsupported, focus on Python first |
| Breaking existing benchmarks | Medium | Make all new fields optional, add validation |
| Agent confusion with too many warnings | Low | Keep warnings concise, only show when relevant |
| Test input files not cleaned up | Low | Use temp workspace that's automatically deleted |

## References

- [Agent KPI Analysis](../../.claude/skills/eval-analyzer/SKILL.md) - Shows cli_args took 80 turns
- [Agent Eval Verification](AGENT_EVAL_VERIFICATION.md) - Original test results
- [cli_args benchmark](../../benchmarks/cli_args.yml) - Current broken version

## Future Work

**v0.4.0+:**
- Add `std/os` module to AILANG for CLI argument support
- Support complex test inputs (JSON, binary files)
- Add environment variable support (`env` field in benchmark YAML)
- Add stdin support (pipe test input instead of files)
- Add benchmark difficulty auto-detection based on agent turn count

---

**Document created**: 2025-10-28
**Last updated**: 2025-10-28
