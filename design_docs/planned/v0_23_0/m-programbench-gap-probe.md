---
status: planned
target: v0.23.0+
priority: P2
estimated: 6-10h (smoke test) + 4-6h (gap writeup)
---

# M-PROGRAMBENCH-GAP-PROBE: Probe AILANG against ProgramBench-style "rebuild a CLI from spec" tasks

**Status**: Planned
**Target**: v0.23.0+ (gap-finding sprint, not benchmark bring-up)
**Priority**: P2 — Pre-research for whether full ProgramBench integration is worth pursuing
**Dependencies**:
- `internal/eval_harness/` agent-based evaluation (complete)
- `std/io`, `std/fs`, `std/env`, `std/string` (complete)
- Existing `benchmarks/cli_args.yml` format (reusable)

---

## Executive Summary

[ProgramBench (Meta FAIR, 2026)](https://github.com/facebookresearch/ProgramBench) is "given a compiled binary + docs, agent rebuilds the program from scratch", graded by behavioral equivalence against ~248k fuzzing-generated tests across 200 tasks (jq, ripgrep, FFmpeg, SQLite, etc.). The paper includes a *different-language ablation* that forces models to reimplement in a non-original language — exactly the slot AILANG fits.

**This sprint is NOT a full ProgramBench integration.** It is a *gap probe*: pick 5–8 of the *simplest* ProgramBench-style tasks (small terminal utilities, single-binary, no network), attempt them in AILANG, and write up what's missing. The output is a **prioritized stdlib/effect-system/prompt gap list**, not a leaderboard entry.

This follows the gap-analysis-first rule from prior language-comparison work: lead with a benchmark-gap probe, not toolchain bring-up. If the probe shows AILANG is within striking distance on the easy tier, then-and-only-then plan full integration in a follow-up sprint.

**Hypothesis**: AILANG can already express the bottom-tier ProgramBench tasks (cat, wc-l, head -n, uniq, simple grep) given current stdlib. Real gaps will emerge in: argv parsing ergonomics, exit-code conventions, streaming I/O for large files, and regex.

---

## Why This Probe Now

1. **ProgramBench is independent + Meta-published** — non-gerrymandered third-party benchmark with a published leaderboard. Same strategic value as VeraBench (M-VERA-BENCH-INTEGRATION), but at *whole-program* scope rather than function scope.
2. **The "different-language ablation" is the natural slot** — the test harness is binary-in/binary-out and language-agnostic; nothing prevents AILANG from being a target *if* AILANG can produce a runnable executable matching the I/O contract.
3. **Gap-finding is the primary value** — evals exist to surface what AILANG can't do yet. ProgramBench tasks stress whole-program capability (argv, exit codes, streaming, signals) in ways our existing function-level benchmarks don't.
4. **Cheap to run** — 5–8 tasks × Claude/Motoko executor is hours, not days. If results are bad, we learn cheaply; if good, we have a wedge for a larger sprint.

---

## Scope: What This Sprint Does and Doesn't Do

### IN scope
- Pick 5–8 ProgramBench tasks from the *small terminal utility* end of the spectrum
- Translate each task's prompt + a representative subset of behavioral tests into the existing `benchmarks/*.yml` format
- Run them through `ailang eval` against AILANG + Python (control) using existing executors (claude, motoko)
- Write a **GAP REPORT** at `design_docs/implemented/v0_23/m-programbench-gap-probe.md` documenting:
  - Per-task: passed / failed / "AILANG can't express this"
  - Aggregated: top 5 missing stdlib capabilities, top 3 prompt-teaching gaps, top 3 effect-system pain points
  - Recommendation: continue to full integration, or shelve

### OUT of scope (explicit non-goals)
- Running the actual ProgramBench harness (`uvx programbench`) — that requires a compiled AILANG binary target, which AILANG doesn't have yet ([project_codegen_strategic_decision](../../../../.claude/projects/-Users-mark-dev-sunholo-ailang/memory/project_codegen_strategic_decision.md): evaluator-first + bytecode VM, no native binary emission).
- Anywhere near the full 200-task set
- Fuzzing-based test generation (we hand-pick ~10 test cases per task)
- Submitting to the ProgramBench leaderboard

The full ProgramBench integration is gated by **both** (a) this probe showing AILANG is competitive on simple tasks, **and** (b) AILANG having a CLI distribution story (bytecode VM packager, or `ailang run --binary`-style entrypoint that ProgramBench's harness can `exec`).

---

## Task Selection Criteria

Tasks must satisfy ALL of:
- Single-file CLI, no network, no subprocess
- Effects fit within current AILANG: `IO`, `FS`, `Env` (no `Process`, no `Clock`-required-for-correctness)
- Stdin OR file-as-argv input (both worth covering)
- Output is text on stdout, exit code 0/non-zero (no binary output)
- Behavioral test suite is expressible as `expected_stdout` + optional `expected_exit_code` in our existing yml format

### Candidate task list (final selection in sprint kickoff)

| # | Task | What it tests | Effects needed | Risk |
|---|------|---------------|----------------|------|
| 1 | `cat` (concat files from argv to stdout) | argv loop, FS read, IO write | FS, IO | Low |
| 2 | `wc -l` (count newlines in file or stdin) | stdin handling, string counting | FS, IO | Low — stdin path may be a gap |
| 3 | `head -n N FILE` (first N lines) | argv flag parsing, early termination | FS, IO | Medium — flag parsing |
| 4 | `uniq` (collapse adjacent dup lines from stdin) | streaming, line-by-line | IO (stdin) | Medium |
| 5 | `tr A B` (1:1 char translation) | string iteration, argv positional args | IO | Low |
| 6 | `seq A B` (print integers A..B, one per line) | int parsing from argv, loops | IO | Low |
| 7 | `grep PATTERN FILE` (literal substring, no regex) | substring search per line | FS, IO | Medium — line iteration |
| 8 | `sort` (read all lines, sort lex, print) | bounded memory, list ops | FS, IO | Medium — large input handling |

Start with #1, #2, #5, #6 (lowest risk); add stretch tasks if those pass.

---

## Deliverables

### D1: Benchmark fixtures
- `benchmarks/programbench_probe_cat.yml`
- `benchmarks/programbench_probe_wc_l.yml`
- … (5–8 total)
- Each follows the existing `cli_args.yml` schema (id, prompt, languages, expected_stdout, input_files, cli_args, caps, tier: `experimental`)

### D2: Eval run + raw results
- Run AILANG + Python through claude executor (and motoko if cheap)
- Store raw outputs under `eval_results/programbench_probe_<timestamp>/`
- Include compile/typecheck failures (those are the real signal)

### D3: GAP REPORT
- File: `design_docs/implemented/v0_23/m-programbench-gap-probe.md`
- Sections:
  1. Per-task table (task, AILANG pass rate, Python pass rate, dominant failure mode)
  2. **Top stdlib gaps** (e.g., "no `io.readStdin`", "no `string.splitLines` lazy variant")
  3. **Top prompt gaps** (e.g., "agents try `argv[1]` syntax that doesn't exist")
  4. **Top language gaps** (e.g., "no early-return from `for`-style iteration without explicit recursion")
  5. **Recommendation**: full integration / refine + retry / shelve

### D4: GitHub issues for each P1 gap
- One issue per stdlib/language gap with reproduction snippet, tagged `gap:programbench`

---

## Axiom Compliance

| Axiom | Score | Justification |
|-------|-------|---------------|
| A1: Determinism | 0 | Eval-suite work; no semantic change |
| A2: Replayability | +1 | Probe results version-pinned with AILANG SHA + model versions |
| A7: Machines First | +2 | Direct measurement of LLM-writes-AILANG-CLI capability |
| A11: Structured Failure | +1 | Output is a structured gap list, not a vague "looks good" |
| (others) | 0 | N/A |

**Net Score: +4** → Proceed (low cost, high information).

---

## Out-of-band considerations

- **Don't strip quota failures from results** (per [feedback_gemini_quota_in_benchmarks](../../../../.claude/projects/-Users-mark-dev-sunholo-ailang/memory/feedback_gemini_quota_in_benchmarks.md)) — they're valid signal about operational reality.
- **Hold model + executor constant across AILANG and Python arms** so any gap is attributable to the language, not the harness.
- **WRONG_LANG categorization** already exists in the harness ([reference_harness_wrong_lang_gating](../../../../.claude/projects/-Users-mark-dev-sunholo-ailang/memory/reference_harness_wrong_lang_gating.md)) — agents that emit Python when asked for AILANG should already be filtered.

---

## Success Criteria

The sprint *succeeds* — regardless of pass rate — if D3 produces a **specific, prioritized gap list** that informs the next sprint's planning. A 0% AILANG pass rate that yields 5 actionable stdlib issues is more valuable than an 80% pass rate with no gap insight.

The sprint **fails** if results are inconclusive: no clear failure pattern, no clear gap recommendation, no follow-up worth scheduling. In that case, document the inconclusiveness honestly and shelve ProgramBench until language/runtime maturity changes the picture.
