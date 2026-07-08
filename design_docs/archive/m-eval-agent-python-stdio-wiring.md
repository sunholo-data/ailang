# M-EVAL-AGENT-PYTHON-STDIO-WIRING — Python agent-mode runner drops `cli_args` and `stdin` from benchmark spec

**Status**: Planned
**Target**: v0.20.1 (patch — corrects benchmark trustworthiness; not a language feature)
**Priority**: P0 — directly invalidates published v0.20.0 agent-mode comparisons for any benchmark using stdin or argv
**Estimated**: ~0.5–1 day (~100–200 LOC + targeted re-run of 2 benchmarks)
**Dependencies**: None (fixes drift between `runner.go` standard path and `agent_runner_multi.go` agent path that's been latent for ≥2 releases)

## Axiom Compliance

**Canonical reference:** [Design Axioms](../../../docs/docs/references/axioms.md)

### Axiom Scoring

| Axiom | Score | Justification |
|-------|-------|---------------|
| A1: Determinism | 0 | Eval runner determinism unchanged; this fix makes existing nondeterminism (missing inputs → different outputs per language) go away, which is a determinism-positive side-effect |
| A2: Replayability | +1 | Benchmark replay-on-CI becomes meaningful — currently `cli_args`-style Python results are unreplayable because the input never reached the program |
| A3: Effect Legibility | 0 | No effect-row changes |
| A4: Explicit Authority | 0 | No capability changes |
| A5: Bounded Verification | +1 | The eval gate is the bound. Today it produces false positives ("AILANG beats Python") because Python wasn't actually evaluated on the same fixture content |
| A6: Safe Concurrency | 0 | No concurrency changes |
| A7: Machines First | +1 | Benchmark JSON becomes a trustworthy machine input again — agents consuming `latest.json` can act on the numbers |
| A8: Minimal Syntax | 0 | No syntax change |
| A9: Cost Visibility | 0 | No cost surface change |
| A10: Composability | 0 | No composition change |
| A11: Structured Failure | **+2** | Direct application of "NO SILENT FALLBACKS" — currently `r.spec.CliArgs` / `r.spec.Stdin` are silently dropped on the agent path; instead they must either propagate or fail-fast with a clear error |
| A12: System Boundary | +1 | Makes the standard-vs-agent boundary visible: same spec, same behaviour, regardless of which evaluator runs it |

**Net Score: +6** → **Decision: Proceed**

### Hard Violation Check

- [x] A1 (Determinism): No new nondeterminism; existing nondeterminism is being removed
- [x] A3 (Effects): No hidden side effects
- [x] A4 (Authority): No ambient access granted
- [x] A7 (Machines First): Aligns with — `latest.json` consumer trust goes UP, not down

## Problem Statement

`internal/eval_harness/agent_runner_multi.go` (the multi-language agent runner that drives Python / JavaScript / Go) does not propagate `BenchmarkSpec.CliArgs` and `BenchmarkSpec.Stdin` into the subprocess invocation. As a result, any benchmark whose Python solution reads `sys.argv[1]` or `sys.stdin` fails at runtime with `IndexError: list index out of range` or returns empty output, **even though the generated Python code is correct**.

The AILANG agent path correctly populates both surfaces (the AILANG runner reuses the standard-eval runner's logic for input wiring). This means standard-eval Python passes happen via the working path; agent-eval Python passes fail via the broken path. The asymmetry only became visible when the M-AILANG-LSP-FOR-AI release post-eval (v0.20.0, 414 agent runs, 9 models) ran enough models on enough benchmarks to make the pattern statistically obvious.

### How this manifested in v0.20.0 post-release eval (2026-05-16)

Two `core`-tier benchmarks that exercise standard input plumbing:

**`cli_args` benchmark** — spec reads `cli_args: ["numbers.txt"]` and writes `numbers.txt` to the workspace with the contents `1\n2\n3\n4\n5`, expects `15\n` on stdout (the sum).

Across 9 models in agent mode:

| Language | Passed | Failed | Most common error |
|---|---:|---:|---|
| AILANG | 4 / 9 | 5 / 9 | (mix of model quality) |
| Python | **1 / 9** | **8 / 9** | 6× `runtime_error: IndexError: list index out of range` at `sys.argv[1]` |

claude-sonnet-4-6's actual Python solution for `cli_args` (correctly generated):

```python
import sys

def main() -> None:
    file_path = sys.argv[1]
    with open(file_path) as f:
        total = sum(int(line.strip()) for line in f if line.strip())
    print(total)

if __name__ == "__main__":
    main()
```

When the agent runner invoked this, the captured `stdout` field shows:

```
Traceback (most recent call last):
  File "/var/folders/.../solution.py", line 12, in <module>
    main()
  File "/var/folders/.../solution.py", line 5, in main
    file_path = sys.argv[1]
                ~~~~~~~~^^^
IndexError: list index out of range
```

`sys.argv[1]` doesn't exist because the runner did not append the spec's `cli_args` to the `python solution.py` invocation. The same model on AILANG (which uses `getArgs()` → `std/env`) passed cleanly, because the AILANG runner *does* pass the args.

**`pipeline` benchmark** — spec has `stdin: |\n  1\n  2\n  3\n  4\n  5\n`, expects `2\n4\n6\n8\n10\n` (each doubled).

Across 9 models in agent mode:

| Language | Passed | Failed | Most common error |
|---|---:|---:|---|
| AILANG | 7 / 9 | 2 / 9 | (mostly the 0/9 motoko-gemma-4 model config bug) |
| Python | **2 / 9** | **7 / 9** | 6× `logic_error: empty stdout` (program produced `\n` because stdin was empty) |

claude-sonnet-4-6's Python solution for `pipeline` (correctly generated):

```python
import sys

def main() -> None:
    numbers = [int(line.strip()) for line in sys.stdin if line.strip()]
    result = [n * 2 for n in numbers]
    print("\n".join(str(n) for n in result))

if __name__ == "__main__":
    main()
```

`actual_stdout: "\n"`, `expected_stdout: "2\n4\n6\n8\n10\n"`. The runner did not pipe `1\n2\n3\n4\n5` to the subprocess's stdin, so the list-comprehension consumed nothing.

### Why this is a class, not a one-off

The asymmetry exists because of how the two paths grew:

- `internal/eval_harness/runner.go` lines 151–172 (Python standard path) **does** append `r.spec.CliArgs` to the `python solution.py` invocation, **and** wires `r.spec.Stdin` via `cmd.Stdin = strings.NewReader(r.spec.Stdin)`. This works correctly.
- `internal/eval_harness/runner.go` lines 351–363 (Go standard path), 574 (JS), 637–651 (additional paths) repeat the same correct pattern.
- `internal/eval_harness/agent_runner_multi.go` (the agent-mode multi-language path) does **not** thread the spec through to the subprocess invocation. The agent harness writes `solution.py` to a tmpdir, then invokes `python solution.py` directly (or via `uv run --python 3.12 solution.py` per the agent's preferred test command), without appending `spec.CliArgs` or supplying `spec.Stdin`.

**Affected benchmarks**: every benchmark with a non-empty `cli_args:` or `stdin:` field. Inventory:

```bash
$ grep -l "^cli_args:\|^stdin:" benchmarks/*.yml | wc -l
```

Currently: at minimum `cli_args.yml` and `pipeline.yml` in `core`; potentially others in `stretch` and `vision`. Need to audit during M1.

**The same bug almost-certainly affects JavaScript and Go agent paths** — this is the M-EVAL-AGENT-MULTI-RUNNER stdio-wiring class. The fix should cover all four languages, not just Python.

**Impact on v0.20.0 published numbers**: the agent-mode AILANG/Python gap of **−5pp** is harness-bug-inflated. If the 2 affected core benchmarks had passed Python on all 9 models (which is plausible given the code is correct), Python's agent total moves from 159/207 (76%) to ~174/207 (~84%), making the actual gap closer to **−13pp**. The dashboard JSON for v0.20.0 needs a note flagging this, and a re-run of the 2 benchmarks (~$2 spend) after the fix should produce a corrected JSON. The standard-eval numbers are not affected (standard path works correctly).

## Goals

**Primary Goal:** An agent-mode invocation that runs a benchmark whose spec has `cli_args:` or `stdin:` must wire both surfaces into the subprocess invocation identically to the standard-mode path, for all four supported languages (Python, JavaScript, Go, AILANG).

**Success Metrics:**
- `cli_args` agent-mode Python pass rate matches standard-mode Python pass rate (currently 1/9 vs ~8/9 — should converge)
- `pipeline` agent-mode Python pass rate matches standard-mode Python pass rate (currently 2/9 vs 7/9 — should converge)
- A new integration test (`internal/eval_harness/agent_runner_multi_test.go`) fixtures `cli_args` + a synthetic `stdin`-bearing benchmark and asserts the subprocess receives both
- Audit pass: `grep -rn "spec.CliArgs\|spec.Stdin" internal/eval_harness/` shows the agent runner uses the same propagation pattern as `runner.go`'s standard path
- After the targeted re-run (the two affected benchmarks × 9 models × Python only = 18 runs, ~$1.50), the v0.20.0 dashboard JSON updates to reflect corrected Python numbers

## High-Impact Decisions

| Decision | Why High Impact | Chosen By | Deadline | Change Cost |
|----------|-----------------|-----------|----------|-------------|
| Should the agent runner share the **standard runner's input-wiring code** (refactor to a common helper) OR duplicate the wiring in agent paths (independent code) | Sharing prevents future drift but increases coupling; duplicating preserves agent-runner independence but creates two truths to maintain | agent (refactor toward shared helper) | implementation | medium |
| Should the agent runner pass `cli_args` / `stdin` to the **agent's own test commands** (`uv run --python 3.12 solution.py`) too, or only to the **grader's verification invocation** | Affects how the agent measures its own work mid-iteration — if the agent runs `python solution.py` to test and gets the same IndexError, it may misdiagnose | agent (apply to both — model needs accurate feedback to iterate) | implementation | low |
| When `spec.Stdin` is set but the subprocess doesn't read stdin, should we still pipe (no-op) or skip? | Skipping is faster but masks the case where the model's code accidentally drains stdin and discards it | agent (always pipe when spec has it — verifiable behaviour) | implementation | low |
| Should this fix include JavaScript + Go runners (M-EVAL-AGENT-MULTI-RUNNER) or just Python (scope-bound)? | JS/Go runners almost-certainly have the same bug. Fixing only Python leaves the same hole for two other languages and we'll be back here in two releases. | author (recommend: fix all 4) | design | low |

### Design Freeze (proposed)

- [x] **Shared helper.** Refactor input-wiring into `internal/eval_harness/agent/input_wiring.go` (new package-internal helper) that takes a `*BenchmarkSpec` + workspace path + base command and returns the augmented `exec.Cmd`. Both the standard and agent runners use it. Drift becomes impossible.
- [x] **Apply to all 4 languages.** Python, JavaScript, Go, AILANG. Sweep the entire `agent_runner_multi.go`.
- [x] **Pipe stdin when spec has it, regardless of program behaviour.** No "smart" detection.
- [x] **Apply to BOTH agent's iteration runs AND grader's verification.** The agent CLI's `Bash` tool invocations of `python solution.py` need argv too — otherwise the agent debugs against a different signal than the grader uses.

## Solution Design

### Overview

Three changes:

1. **Extract** the existing input-wiring logic from `runner.go` (lines 151–172, 351–363, 574, 637–651 — all roughly the same shape) into a single helper that produces an `exec.Cmd` from `(baseCmd, args, spec, workspaceDir)`.
2. **Wire** the helper into `agent_runner_multi.go`'s per-language subprocess invocation paths (Python, JavaScript, Go) AND the agent CLI's test-command preamble (so when the model runs `python solution.py` in its own Bash turn, it gets the right args/stdin).
3. **Verify** with a new integration test fixturing a stdin-bearing benchmark in `t.TempDir()` and asserting the subprocess actually receives the bytes.

### Architecture

```
benchmark spec (yaml)                           agent_runner_multi.go
  cli_args: ["numbers.txt"]    ────────┐         per-language subprocess
  stdin: |                              │           e.g. python solution.py
    1\n2\n3\n4\n5                       │
                                        ▼
                            input_wiring.go (NEW)
                              ┌──────────────────────────┐
                              │ applyInputSpec(cmd, spec │ ────► cmd.Args += spec.CliArgs
                              │                  workspace)│ ────► cmd.Stdin = strings.NewReader(spec.Stdin)
                              │                            │
                              │ (replaces duplicated logic │
                              │  in runner.go × 4 langs)   │
                              └──────────────────────────┘
                                        ▲
runner.go (standard path) ─────────────┘
```

**Components:**

1. **`internal/eval_harness/agent/input_wiring.go`** — new file, ~50 LOC. Single function `ApplyInputSpec(cmd *exec.Cmd, spec *BenchmarkSpec, workspaceDir string) error` that:
   - Appends `spec.CliArgs` to `cmd.Args` (or resolves them relative to `workspaceDir` if they're path-like)
   - Sets `cmd.Stdin = strings.NewReader(spec.Stdin)` when non-empty
   - Writes `spec.InputFiles` to `workspaceDir` (today this happens inline; consolidating)

2. **Refactor `runner.go`** to call `ApplyInputSpec` instead of inlining the wiring. Four call-sites (Python std path, Go std path, JS std path, AILANG std path).

3. **Add `ApplyInputSpec` calls to `agent_runner_multi.go`** for Python, JS, Go agent subprocesses. AILANG agent already works; verify it now uses the shared helper.

4. **New test** `internal/eval_harness/agent_runner_multi_test.go` — fixturing a tmp benchmark with `cli_args: ["arg1", "arg2"]` and `stdin: "hello\n"`, mock the agent runner to capture the `exec.Cmd`, assert `Args` contains "arg1" "arg2" and `Stdin` reads "hello\n".

### Implementation Plan (sketch — sprint-planner to size)

**Phase 1: Helper extraction** (~2h, ~80 LOC)
- [ ] New `internal/eval_harness/agent/input_wiring.go` with `ApplyInputSpec`
- [ ] Refactor `runner.go`'s 4 standard-path call-sites to use the helper
- [ ] Unit test in `input_wiring_test.go` (table-driven, covers empty/cli-only/stdin-only/both/path-relative-args)
- [ ] Verify all existing `runner_test.go` tests still pass

**Phase 2: Agent path wiring** (~3h, ~80 LOC)
- [ ] Audit `agent_runner_multi.go` for every per-language subprocess invocation
- [ ] Insert `ApplyInputSpec` calls on each
- [ ] Audit the agent CLI's test-command preamble (the `Bash` tool turn where the model is told how to run `python solution.py`) — does the agent's *own* test invocation need argv too? If yes, the prompt template needs updating (the agent's tool-call args), not just the grader's invocation.
- [ ] Integration test fixture: stdin-bearing benchmark, mock-grader, assert subprocess receives bytes

**Phase 3: Targeted re-run + dashboard amendment** (~1h, $1.50 cost)
- [ ] Re-run `cli_args` + `pipeline` Python-only across the 9 agent_suite models
- [ ] Regenerate dashboard JSON
- [ ] Append a CHANGELOG note: "v0.20.0 agent Python numbers for cli_args + pipeline were undercounted due to M-EVAL-AGENT-PYTHON-STDIO-WIRING (fixed v0.20.1); corrected: …"

### Files to Modify/Create

**New files:**
- `internal/eval_harness/agent/input_wiring.go` — ~50 LOC
- `internal/eval_harness/agent/input_wiring_test.go` — ~120 LOC
- `internal/eval_harness/agent_runner_multi_test.go` — ~150 LOC (integration test)

**Modified files:**
- `internal/eval_harness/runner.go` — replace 4 inline call-sites with helper calls, ~−40 LOC net
- `internal/eval_harness/agent_runner_multi.go` — add helper calls to Python/JS/Go subprocess invocations, ~+40 LOC
- `cmd/ailang/eval_suite.go` — no change expected; if the spec→runner wiring is upstream of the agent-vs-standard split, no touch
- `prompts/agent-prompts/*.md` (if applicable) — update agent test-command preamble to mention `python solution.py <args>` so the model's own iteration runs against the same surface as the grader
- `changelogs/v0.10-current.md` — `[v0.20.1]` entry describing both the fix and the corrected v0.20.0 Python numbers

## Examples

### Example 1: Today's agent run of `cli_args` (Python) — broken

```bash
$ # what the agent runner currently invokes (inside multi-runner.go)
$ cd /tmp/ailang_eval/cli_args_*/cli_args_python_*/
$ python solution.py
Traceback (most recent call last):
  ...
IndexError: list index out of range
$ # stdout captured, error_category=runtime_error, marked failed
```

### Example 2: After fix — same benchmark, same code, passes

```bash
$ # what the fixed agent runner will invoke
$ cd /tmp/ailang_eval/cli_args_*/cli_args_python_*/
$ # numbers.txt was already written from spec.InputFiles
$ python solution.py numbers.txt
15
$ # stdout matches expected "15\n", marked passed
```

### Example 3: After fix — `pipeline` benchmark

```bash
$ # what the fixed agent runner will invoke
$ cd /tmp/ailang_eval/pipeline_*/pipeline_python_*/
$ printf '1\n2\n3\n4\n5\n' | python solution.py
2
4
6
8
10
$ # stdout matches expected, marked passed
```

## Success Criteria

- [ ] `internal/eval_harness/agent/input_wiring.go` created with `ApplyInputSpec` covering CliArgs + Stdin + InputFiles
- [ ] All 4 standard-path call-sites in `runner.go` refactored to use the helper; existing tests still pass
- [ ] `agent_runner_multi.go` Python/JS/Go subprocess invocations call the helper
- [ ] New integration test asserting argv + stdin propagation passes
- [ ] Targeted re-run of `cli_args` + `pipeline` Python on agent_suite shows Python pass rate matching the standard-eval pattern (≥7/9 expected)
- [ ] v0.20.0 dashboard JSON gets a corrective note + the v0.20.1 release notes call out the affected metrics
- [ ] `make ci` passes

## Testing Strategy

**Unit tests** (`internal/eval_harness/agent/input_wiring_test.go`):
- Empty spec: no args, no stdin → `cmd` unchanged
- CliArgs only: subprocess receives them in order
- Stdin only: subprocess's stdin reads the bytes
- Both: subprocess receives both
- Path-relative arg: relative path resolves against workspaceDir
- InputFiles + CliArgs both reference the same file: workspaceDir layout is correct

**Integration tests** (`internal/eval_harness/agent_runner_multi_test.go`):
- Fixture: synthetic benchmark with `cli_args: ["test.txt"]`, `input_files: {test.txt: "hello"}`, `expected_stdout: "hello\n"`
- Fake `python solution.py` script: `cat $1` (echoes the file)
- Run through the agent runner; assert `stdout == "hello\n"`
- Same but with `stdin: "world\n"` + a script that does `cat` (no args) — assert subprocess receives "world\n"

**Manual / cross-validation:**
- Re-run `cli_args` + `pipeline` Python-only after the fix
- Compare to the v0.20.0 dashboard JSON's recorded Python pass rates for the same benchmarks (should improve from 1/9 and 2/9 toward parity with standard-eval Python pass rates)

## Deferred Decisions

- **Whether to backfill historical eval baselines** (v0.13.0, v0.14.x, v0.15.x, v0.18.4 agent runs are all suspect for any benchmark using cli_args/stdin). For consistency we'd want to re-run, but cost is significant (~$200+ across all historical releases). Defer to a `M-EVAL-HISTORICAL-CORRECTION` follow-up; for now the v0.20.1 release notes flag that releases prior to v0.20.1 have this systematic Python-agent-mode undercount.
- **Auditing JS/Go agent runs** for similar drops — explicit non-goal here since JS/Go are not in default suites. Address when `lang_harness_suite` runs become standard.
- **Refactoring the agent prompt to teach the model about argv conventions** — the model already writes correct argv-using code, the problem is just the subprocess call. Prompt is fine.

## Non-Goals

- **Reworking the agent's test-command preamble for languages other than Python.** Out of scope; this fix is specifically about the **runner's invocation**, not the agent's own iteration loop. If the agent's `Bash` tool turns ALSO need argv, that's a separate prompt-update task (file as M-EVAL-AGENT-TEST-CMD-ARGV).
- **Changing the benchmark YAML schema.** `spec.CliArgs` and `spec.Stdin` already exist; this is purely about plumbing them through.
- **Adding new benchmarks.** Curation work is separate.
- **Fixing motoko-gemma-4's invalid model ID** in `models.yml`. Tracked separately as M-EVAL-MODELS-YML-GEMMA-ID-FIX (one-line config fix; not coupled to the harness bug).

## Risks & Mitigations

| Risk | Impact | Mitigation |
|------|--------|-----------|
| The shared helper introduces subtle behaviour change in standard-eval (cmd args ordering, stdin closing timing) | Standard-eval pass rates shift unexpectedly | Refactor in Phase 1 keeps existing tests green; explicit char-by-char `exec.Cmd.Args` assertions in input_wiring_test |
| Agent CLI's `Bash` tool turns (`python solution.py`) still get IndexError because we only fix the grader, not the agent's iteration | Models continue to misdiagnose during agent turns even though grading is fixed | Phase 2 explicitly audits the agent's test-command preamble; update prompt to instruct the model to pass args when iterating |
| The "8/9 Python passed cli_args" assumption turns out to be wrong (maybe Python *does* fail on real model output too) | The post-fix re-run shows a smaller Python lift than projected | Acceptable — corrected number is still the right number to publish |
| JavaScript/Go runners diverge from Python's wiring (have their own bugs) | Fix doesn't cover those languages | Phase 2 audits ALL per-language paths in agent_runner_multi.go, not just Python |

## Best Practices Push

After this ships, the eval test harness should add a **per-language smoke test** (`make eval-harness-smoke`) that fixtures a 1-benchmark synthetic spec with both cli_args and stdin, runs it through both the standard and agent paths for all 4 languages, and asserts identical pass/fail behaviour. This catches the next drift before it becomes a v0.20.0-style release-time discovery.

## Related Documents

- **[v0.20.0 release CHANGELOG](../../../changelogs/v0.10-current.md)** — section `[v0.20.0]` → "Benchmark Results (M-EVAL)" — the agent matrix that exposed this bug. The Python column for cli_args + pipeline is the empirical evidence.
- **`docs/static/benchmarks/latest.json`** — dashboard JSON whose Python numbers need amendment after this fix.
- **`internal/eval_harness/runner.go`** lines 151–172, 351–363, 574, 637–651 — the working standard-path wiring this fix consolidates.
- **`internal/eval_harness/agent_runner_multi.go`** — the file where the missing wiring lives today.

## Conflict Surface

This milestone does **not** touch `internal/parser/`, `internal/lexer/`, `internal/ast/`, `internal/types/`, `internal/elaborate/`, `internal/iface/`, `internal/codegen/`, `internal/eval/`, `internal/vm/`, `internal/effects/`, or `cmd/ailang/exec.go`. It's confined to `internal/eval_harness/` and is a pure plumbing change — no language semantics, no parser surface, no runtime behaviour.

## DESIGN_DOC_PATH

`design_docs/planned/v0_21_0/m-eval-agent-python-stdio-wiring.md`
