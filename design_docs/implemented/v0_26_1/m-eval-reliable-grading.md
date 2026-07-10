# M-EVAL-RELIABLE-GRADING: Grade the Model's Actual Workspace, Not an Isolated File

**Status**: ✅ Implemented (v0.26.1)
**Target**: v0.26.x
**Priority**: P0 (the eval score is currently unreliable for multi-file benchmarks — it measures the wrong thing)
**Estimated**: ~1 day
**Dependencies**: M-EVAL-REIMPLEMENT-BENCH (the benchmarks that exposed this), the executor agent_env fix (582cb74b2)

## Problem

The agent-mode grader **discards the modules the agent was asked to implement and grades a single
isolated file**. `validateSolution` (`agent_runner_multi.go:438`) reads the agent's
`benchmark/solution.ail`, then `runAILANGSolution` → `runner.Run(code)` runs it in a **fresh
workspace that re-seeds `input_files`** (`runner.go:318-329`) — so the stub (`docx_parser.ail`) is
reset to its empty form and the agent's real implementation is thrown away. Only `solution.ail`,
run in isolation against the stub, is measured.

Consequence: a benchmark that *tells* the model "reimplement the stubbed `docx_parser.ail`" only
passes if the model **ignores that instruction** and inlines a self-contained copy into
`solution.ail`. The score measures "did the model guess the solution.ail convention," not "did the
model implement the task."

**Proof (docx_reimplement, motoko-local, session f7261a35):**
- The model implemented `docx_parser.ail` (12 exports) and a `main.ail` entrypoint, then ran
  `ailang run --entry main main.ail` → exit 0, stdout **byte-for-byte identical to the golden**.
- The grader re-stubbed `docx_parser.ail` and ran the agent's `solution.ail` (which lacked `main`)
  → "entrypoint 'main' not found" → **FAIL**, despite a verified-correct solution in the workspace.
- markdown_reimplement PASSED only because that run happened to inline everything into
  `solution.ail`. Both results are measurement artifacts; neither reflected the model.

## Findings (de-risk the build)

- `validateSolution` already receives the agent's intact `workspace` (and `solutionPath` inside it).
  The agent's implemented modules are present at grade time — they are simply ignored in favor of a
  fresh re-stubbed workspace. Grading **in place** is feasible with no new plumbing.
- The fresh-workspace runner (`runner.Run`) re-seeds `spec.InputFiles` unconditionally — this is the
  exact line that overwrites the agent's implementation with the stub.
- The single-file standard benchmarks (74) legitimately want the current behavior (the answer *is*
  `solution.ail`). The fix must be **opt-in** so they are untouched.
- `error_category` already exists on the result record; today an entrypoint/compile failure in the
  graded file is indistinguishable from a real output mismatch.

## High-Impact Decisions

| Decision | Why | Change Cost |
|---|---|---|
| Grade in the agent's **preserved workspace** via a harness-owned probe entrypoint | Measures the agent's actual implementation, not an isolated re-stubbed file | med |
| Probe + fixed deps are **re-seeded from the spec** at grade time; only `solution_files` keep the agent's version | The agent can't tamper with the measurement (probe) or cheat via deps; its target implementation survives | low |
| Opt-in via `grade_entrypoint` (unset → current single-file behavior) | Zero risk to the 74 single-file benchmarks | low |
| Distinct `error_category: harness_setup` for probe compile/entrypoint/module errors | A measurement failure must never score as a model failure | low |
| Benchmark self-check (reference impl through the probe must PASS) | Catches a mis-framed benchmark before it scores any model | low |

## Solution Design / Sprint

**M1 — Workspace grading (the core fix)** (~4h)
- `spec.go`: add `GradeEntrypoint string \`yaml:"grade_entrypoint,omitempty"\`` (the probe file the
  harness runs, e.g. `main.ail`) and `SolutionFiles []string \`yaml:"solution_files,omitempty"\``
  (the file(s) the agent implements — keep the agent's version at grade time).
- `agent_runner_multi.go`: when `spec.GradeEntrypoint != ""`, grade in the agent's `workspace`:
  re-seed every `input_file` **except** `solution_files` (restores the probe + fixed deps to canonical,
  preserves the agent's implementation), then run
  `ailang run --entry main --quiet --relax-modules --stdlib-path … <grade_entrypoint>` from the
  workspace and `CompareOutput(golden, stdout)`. When unset, fall through to today's
  `runAILANGSolution` (single-file) path unchanged.
- Unit test: a workspace with an implemented module + canonical probe grades PASS; the same with a
  stubbed module grades FAIL; `grade_entrypoint` unset uses the legacy path.

**M2 — Honest failure categorization** (~2h)
- In the workspace-grade path, classify the probe outcome: `compile_error` / "entrypoint … not found"
  / module-mismatch → `error_category: "harness_setup"` (CompileOk=false, distinct from a clean run
  with a wrong stdout → `model_logic`). Surface it in the summary so the dashboard can separate
  "couldn't measure" from "model wrong."

**M3 — Convert the reimplement benchmarks + self-check + validate** (~2h)
- `markdown_reimplement.yml` + `docx_reimplement.yml`: add `grade_entrypoint: main.ail` and
  `solution_files: [docparse/services/<parser>.ail]`. The task_prompt already says "implement the
  stub"; grading now honors it (solution.ail no longer matters).
- Self-check (`make` target / Go test): seed the **reference** implementation (not the stub) into the
  workspace, run the probe → assert PASS. A benchmark whose canonical setup doesn't pass is mis-framed.
- Re-run `docx_reimplement` + `markdown_reimplement` on motoko-local. docx must now PASS (the model
  already produced the exact golden); markdown must still PASS.

## Conflict Surface

- Additive spec fields (`omitempty`) — existing benchmarks unaffected.
- The grading change is gated on `grade_entrypoint`; the 74 single-file benchmarks take the unchanged
  legacy path. Fixture: a single-file benchmark (e.g. records_book) still grades identically.
- Workspace grading runs from the agent's tmp workspace; `--stdlib-path` + `--relax-modules` mirror the
  legacy runner so module resolution is identical.

## Success Criteria
- [ ] M1: workspace-grade runs the probe against the agent's implemented module; legacy path unchanged when `grade_entrypoint` unset.
- [ ] M2: probe entrypoint/compile failures categorize as `harness_setup`, not a model fail.
- [ ] M3: `docx_reimplement` PASSES on motoko-local (was a measurement artifact FAIL); `markdown_reimplement` still PASSES; self-check green.
- [ ] No regression on the 74 single-file benchmarks (spot-check a representative one).

## Non-Goals
- Anti-cheat hardening beyond re-seeding fixed files (the probe + deps are canonical; that's enough for capability measurement).
- Reworking standard (0-shot) single-file grading — it is correct for its model.

---
**Document created**: 2026-06-27

DESIGN_DOC_PATH: design_docs/planned/m-eval-reliable-grading.md
