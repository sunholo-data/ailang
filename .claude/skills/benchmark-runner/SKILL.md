---
name: benchmark-runner
description: Run ai-coding-lang-bench benchmark trials against AILANG, monitor active sessions, analyze results (cost/tokens/time/pass-rate), compare prompt strategies, and identify language gaps. Use when user says "run benchmark", "benchmark AILANG", "run trials", "analyze results", "find gaps", "monitor benchmark", or "compare prompt strategies".
---

# Benchmark Runner

Run the ai-coding-lang-bench benchmark against AILANG, monitor sessions, analyze results, and identify language gaps.

## Quick Start

```bash
# Run 5 trials with the spec-prompted strategy
/benchmark-runner run ailang/spec 5

# Monitor live sessions
/benchmark-runner monitor

# Analyze results and compare strategies
/benchmark-runner analyze
```

## When to Use This Skill

Invoke this skill when:
- User says "run benchmark", "benchmark AILANG", "run trials"
- User wants to compare prompt strategies
- User wants to analyze benchmark results or find language gaps
- User says "monitor benchmark" or "check trial progress"

## Overview

**Fork repo**: https://github.com/sunholo-data/ai-coding-lang-bench
**Local path**: `/Users/mark/dev/sunholo/ai-coding-lang-bench/`

The benchmark has Claude Code implement "MiniGit" (a minimal git clone) in AILANG. It tests:
- v1: init, add, commit, log (11 tests)
- v2: extends v1 with status, diff, checkout, reset (30 tests)

Each trial invokes `claude -p` with a prompt + language-specific `extra_prompt`.

## Prompt Strategy Variants

AILANG has 5 variants in `benchmark.rb` to measure information value of different documentation layers:

| Variant | What Claude Gets | Measures |
|---------|-----------------|----------|
| `ailang/zero-shot` | "Use AILANG (.ail files)" | Baseline — hallucination rate |
| `ailang/cli` | "Run `ailang devtools-prompt`" | CLI self-documentation value |
| `ailang/spec` | AILANG-SYNTAX.md file + "Read it first" | Can a spec replace training data? |
| `ailang` | AILANG-SYNTAX.md + CLI + capability hints | Guided discovery — best realistic case |
| `ailang/rich` | Everything + pre-loaded imports + exit code workaround | Upper bound with hand-holding |

`AILANG-SYNTAX.md` is auto-copied into each trial's working directory for all ailang variants.

**Fair comparison**: `ailang/spec` is the most honest comparison to other languages — the syntax file is equivalent to what training data provides for known languages.

## Key Metrics

| Metric | Where | Why |
|--------|-------|-----|
| Cost (USD) | `logs/*.json` → `total_cost_usd` | Direct comparison to published results |
| Output tokens | `logs/*.json` → `usage.output_tokens` | Discovery overhead vs actual coding |
| Turns | `logs/*.json` → `num_turns` | How many iterations needed |
| Time (seconds) | `results/results.json` → `v1_time`, `v2_time` | Wall clock comparison |
| Test pass rate | `results/results.json` → `v1_passed_count` etc. | Correctness |
| LOC | `results/results.json` → `v1_loc`, `v2_loc` | Solution conciseness |
| Time to first write | Session JSONL analysis | Discovery vs implementation split |

## Available Scripts

### `scripts/run_trials.sh <variant> <count> [start]`
Run benchmark trials for a specific AILANG variant.

```bash
# Run 5 trials of ailang/spec starting from trial 1
.claude/skills/benchmark-runner/scripts/run_trials.sh ailang/spec 5 1

# Run 3 trials of ailang (guided) starting from trial 6
.claude/skills/benchmark-runner/scripts/run_trials.sh ailang 3 6
```

### `scripts/analyze_results.sh`
Parse results.json and logs to produce a metrics summary table.

```bash
.claude/skills/benchmark-runner/scripts/analyze_results.sh
```

### `scripts/compare_strategies.sh`
Side-by-side comparison of different prompt strategy variants.

```bash
.claude/skills/benchmark-runner/scripts/compare_strategies.sh
```

### `scripts/find_gaps.sh`
Parse Claude session JSONL files to identify repeated failed searches — these indicate AILANG language gaps.

```bash
.claude/skills/benchmark-runner/scripts/find_gaps.sh
```

### `scripts/monitor_session.sh`
Live monitoring of active benchmark sessions (wrapper around monitor.sh in fork repo).

```bash
.claude/skills/benchmark-runner/scripts/monitor_session.sh        # One-shot
.claude/skills/benchmark-runner/scripts/monitor_session.sh --watch # Auto-refresh
```

## Workflow

### 1. Run Trials

```bash
# Start with 1 trial to verify setup
ruby benchmark.rb --lang ailang/spec --trials 1 --start 1

# Then run batch of 5
.claude/skills/benchmark-runner/scripts/run_trials.sh ailang/spec 5
```

### 2. Monitor Progress

```bash
# In another terminal
cd /Users/mark/dev/sunholo/ai-coding-lang-bench
./monitor.sh --watch
```

### 3. Analyze Results

```bash
.claude/skills/benchmark-runner/scripts/analyze_results.sh
```

### 4. Identify Gaps

```bash
.claude/skills/benchmark-runner/scripts/find_gaps.sh
```

Output shows repeated failed searches grouped by topic:
```
exit/exitCode (14 searches across 3 trials)
  - Trial 1 v1: 14 tool calls searching for exit
  - Trial 2 v1: 8 tool calls
  - Trial 3 v1: 2 tool calls (file-based, faster)

:: cons expression (5 searches across 2 trials)
  - Trial 1 v2: tried x :: xs, got error
  - Trial 2 v2: same pattern
```

### 5. Create Design Docs for Gaps

For each identified gap, create a design doc:
```bash
# Use design-doc-creator skill
.claude/skills/design-doc-creator/scripts/create_planned_doc.sh m-gap-name v0_10_0
```

### 6. Implement Fixes and Re-benchmark

After implementing a gap fix:
1. Rebuild: `make quick-install`
2. Regenerate: `ailang prompt > AILANG-SYNTAX.md` (in fork repo)
3. Run trials again with same variant
4. Compare before/after metrics

## Data Locations

```
/Users/mark/dev/sunholo/ai-coding-lang-bench/
  results/
    results.json         # Structured metrics (time, LOC, pass/fail)
    meta.json            # Environment metadata
    report.md            # Upstream report for other languages
  logs/
    minigit-*.json       # Claude cost/tokens per trial
    session-*.jsonl      # Full chat history (thinking, tool calls, errors)
  generated/
    minigit-*/           # Generated AILANG source code + shell wrappers
  AILANG-SYNTAX.md       # Full ailang prompt output (copied to working dirs)
  monitor.sh             # Live session monitor
  benchmark.rb           # Main benchmark runner
```

## Published Baseline Results

See [resources/published_results.md](resources/published_results.md) for other language results.

Key comparisons:
- Ruby: $0.36, 73s, 40/40 tests
- Python: $0.38, 75s, 40/40 tests
- Haskell: $0.74, 174s, 39/40 tests

## Known Issues

1. **Trial isolation**: `generated/` directory from prior trials is visible — Claude may read previous solutions. Clean `generated/` between strategy comparisons.
2. **Cache warming**: Trial 2+ benefits from Claude prompt caching. First trial of a session is always slower.
3. **Exit code gap**: AILANG has no `exit(code)` builtin. Design doc at `design_docs/planned/v0_10_0/m-exit-code.md`.
4. **Cons expression gap**: `::` only works in patterns, not expressions. Design doc at `design_docs/planned/v0_10_0/m-cons-expression.md`.

## Notes

- Regenerate `AILANG-SYNTAX.md` after language changes: `ailang prompt > /path/to/fork/AILANG-SYNTAX.md`
- The benchmark uses `claude -p --dangerously-skip-permissions --output-format json`
- Each trial costs ~$0.50-$2.00 depending on discovery overhead
- v2 is always faster than v1 because it extends existing code from v1
