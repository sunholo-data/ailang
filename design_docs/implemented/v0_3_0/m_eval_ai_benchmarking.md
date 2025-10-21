# M-EVAL: AI Usability Benchmarking Harness

**Milestone**: M-EVAL (AI Evaluation Framework)
**Version**: Parallel (v0.2.0+ experimental)
**Timeline**: 1–2 weeks (can progress in parallel to runtime/effects work)
**Estimated LOC**: ~600–800 (Go + .ail harness + YAML specs)
**Priority**: HIGH (validates AILANG's purpose vs Python baseline)

---

## Executive Summary

M-EVAL creates the first self-hosted evaluation system inside the AILANG project.

It will measure AI efficiency across AILANG and Python by running identical benchmark prompts, tracking:
- Token usage (prompt + completion)
- Compilation success/failure
- Runtime success/failure
- Time to completion

**Goal**: Prove that AILANG reduces AI effort vs Python in real-world coding tasks.

---

## Problem Statement

**Today:**
- We believe AILANG will make AI coding easier.
- But we lack empirical data to show it's faster, cheaper, or more reliable than Python.

**Without benchmarks:**
- We can't justify AILANG's niche.
- We can't iterate intelligently (no feedback loop).
- We can't compare across model sizes/vendors.

---

## Goals & Non-Goals

### Goals
1. Standardized benchmark tasks (FizzBuzz → JSON transform → CLI → mini-pipeline).
2. Dual implementation runners (AILANG + Python).
3. Collect metrics: tokens, time, success/failure, error modes.
4. Produce reproducible reports (JSON/CSV/Markdown).

### Non-Goals (v0.2.x)
- ❌ Model training/fine-tuning loops (future v0.3.x).
- ❌ Complex benchmarks (web frameworks, concurrency).
- ❌ Automated leaderboard UI (just CSV/Markdown in v0.2.x).

---

## Design

### Architecture Overview

```
┌──────────────┐
│ Benchmark    │  (YAML spec)
│ fizzbuzz.yml │
└───────┬──────┘
        │
        ▼
┌─────────────────┐
│ Eval Runner CLI │  ailang eval run fizzbuzz --langs python,ailang
└───────┬─────────┘
        │
   ┌────▼─────┐
   │ AI Agent │  (LLM call: prompt with <LANG>)
   └────┬─────┘
        │ code
        ▼
┌──────────────┐
│ Language Bin │  (python3 / ailang run)
└───────┬──────┘
        │ result
        ▼
┌─────────────────┐
│ Metrics Logger  │
│ JSON / CSV file │
└─────────────────┘
```

---

## Key Components

### 1. Benchmark Spec (YAML/JSON)

**File**: `benchmarks/fizzbuzz.yml`

```yaml
id: fizzbuzz
description: "Classic FizzBuzz problem"
languages: ["python", "ailang"]
entrypoint: "main"
caps: ["IO"]
prompt: |
  Write a program in <LANG> that prints FizzBuzz from 1 to 100.
expected_stdout: |
  1
  2
  Fizz
  4
  Buzz
  ...
```

### 2. Eval Runner CLI

**File**: `cmd/ailang/eval.go`

```bash
$ ailang eval run fizzbuzz --langs python,ailang --model gpt-5
```

**Responsibilities:**
- Load spec file
- Call AI with prompt (`<LANG>` replaced with target)
- Save raw completion
- Attempt compile/run
- Capture stdout/stderr
- Compare with `expected_stdout` (substring match OK)
- Log JSON result

### 3. Metrics JSON Output

**File**: `eval_results/fizzbuzz_gpt5.json`

```json
{
  "id": "fizzbuzz",
  "lang": "ailang",
  "model": "gpt-5",
  "tokens": 342,
  "compile_ok": true,
  "runtime_ok": true,
  "stdout_ok": true,
  "duration_ms": 212,
  "stderr": ""
}
```

### 4. Reporting Script

**File**: `tools/report_eval.sh`
- Aggregate JSON → CSV
- Summarize success rate, avg tokens, avg time per task
- Output Markdown table for README

---

## Example Benchmark Set

| ID | Task Type | Languages | Difficulty | Expected Gain |
|----|-----------|-----------|------------|---------------|
| fizzbuzz | Control flow | py/ailang | Easy | Low |
| json_parse | Data parsing | py/ailang | Medium | Medium |
| pipeline | IO + list | py/ailang | Medium | High |
| cli_args | IO + FS | py/ailang | Hard | High |
| adt_option | Algebraic | py/ailang | Medium | Very High |

---

## Implementation Plan

### Phase 1: Scaffolding (2 days)
- Create `benchmarks/` folder with YAML specs
- Add `cmd/ailang/eval.go` CLI skeleton
- Implement JSON logger

### Phase 2: AI Agent Harness (3 days)
- Implement prompt builder (replace `<LANG>`).
- Connect to AI APIs (OpenAI/Anthropic).
- Collect token usage + completions.

### Phase 3: Language Runners (2–3 days)
- `runPython(src.py)` using subprocess.
- `runAILANG(src.ail)` via `ailang run`.
- Capture exit code, stdout, stderr.

### Phase 4: Reporting (2 days)
- Aggregate JSON into CSV/Markdown.
- Add summary tables for README.

---

## Testing Strategy

### Unit Tests
- Spec loader works (YAML → struct).
- AI prompt builder inserts `<LANG>`.
- Logger writes valid JSON.

### Integration Tests
- Run local "mock" benchmarks with stub programs.
- Validate stdout comparisons.

### End-to-End
- Run 2 tasks (fizzbuzz, json_parse) across Python + AILANG.
- Ensure logs produced and CSV report generated.

---

## Risks & Mitigation

| Risk | Impact | Mitigation |
|------|--------|------------|
| Token accounting differs per API | Medium | Always log API usage fields; normalize later |
| Python baseline "too easy" | Low | Use tasks that stress AILANG's ADTs and effects |
| Benchmark creep | Medium | Start with 5 tasks, expand after |
| Model bias towards Python | High | Blind prompt format: `<LANG>` substitution only |

---

## Success Criteria
- ✅ At least 5 benchmarks runnable in both Python + AILANG
- ✅ Logs per run with tokens, compile/runtime success
- ✅ 1 aggregated CSV + Markdown report
- ✅ Demonstrated at least one benchmark where AILANG requires fewer tokens

---

## Scope & Limitations (Phase 1)

**What M-EVAL Measures:**
- ✅ Single-shot code generation quality
- ✅ Initial token efficiency (first attempt)
- ✅ Syntax familiarity (compile success rate)
- ✅ "Best case" scenario baseline

**What M-EVAL Does NOT Measure:**
- ❌ Multi-turn iteration/debugging
- ❌ Total effort (cumulative tokens across retries)
- ❌ AI learning curve (error → fix → success)
- ❌ Real-world agentic coding workflows

**Why Single-Shot First:**
1. **Fast feedback**: Baseline data available immediately
2. **Cheap to run**: One API call per benchmark
3. **Diagnostic value**: Reveals syntax/semantic gaps
4. **Prompt engineering**: Informs what context AI needs

**What We Learn:**
- Which AILANG syntax confuses the AI
- What documentation/examples are missing
- How to improve prompts for Phase 2
- Baseline token efficiency (best case)

---

## Future Extensions

### Phase 2: M-EVAL2 (v0.3.0) - Agentic Evaluation
- Multi-turn agent loop (3-5 iterations)
- Feedback mechanism: error → retry with context
- CLI integration: Claude Code, Gemini
- Cumulative token tracking
- **See**: [M-EVAL2 Design Doc](m_eval2_agentic.md)

### Phase 3: Advanced (v0.4.0+)
- Net/Clock benchmarks once effects land
- Concurrency benchmarks (spawn, channels)
- Tool use support (file search, web search)
- Fine-tuning dataset generation

### Phase 4: Production (v0.5.0+)
- Self-updating leaderboard
- Continuous benchmarking in CI
- Historical trending
- Model comparison dashboards

---

**Status**: ✅ COMPLETE (Phase 1 - Baseline Single-Shot)
**Parallelizable**: Yes (can start now without blocking runtime work)

**Next Phase**: M-EVAL2 (agentic evaluation) - design complete, awaiting baseline data
