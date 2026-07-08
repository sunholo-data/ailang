# M-EVAL-REIMPLEMENT-BENCH: Multi-file Reimplement Tasks as Tracked Benchmarks

**Status**: ✅ Implemented (v0.26.1)
**Target**: v0.26.x
**Priority**: P1 (the only instrument that surfaces the long-context harness fixes on the dashboard)
**Estimated**: ~1 day
**Dependencies**: motoko shim repointed to mk-ast (done 2026-06-26); v0.26.0 iface fix (done)

## Problem

The motoko harness reliability fixes this session — `iface --compact` carrying ADT
constructor fields (v0.26.0), system-prompt env-forward (mk-ast / PR #76), reliable
compaction (PR #75) — **only manifest on long, multi-file, ADT-construction tasks** that
grow context enough to trigger compaction and read dependency module interfaces. The 74
nightly benchmarks are all **single-file** (`entrypoint: main`, zero `files:` lists) and the
production agent path doesn't enable `MOTOKO_AST_AUTOREAD`, so they structurally cannot
exercise any of the fixes. We measured the fixes only via ad-hoc `eval_projects/*/run.sh`
scripts (docx: cat-fallbacks 11→8). We need the reimplement tasks as **proper benchmarks** so
the resume-immune nightly filler runs them on mk-ast, banks results, and the dashboard tracks
harness reliability over time — "we fixed it" becomes a number that moves on regression.

## Findings (this session's investigation — de-risks the build)

- `input_files:` (3 benchmarks already use it) seeds files into the agent workspace.
  `seedInputFiles` (`agent_runner_multi.go:410`) does `os.MkdirAll(filepath.Dir(...))`, so the
  `docparse/services/markdown_parser.ail` subdirectory layout works.
- Grading is `expected_stdout` (all 73 benchmarks). So a reimplement benchmark's entrypoint
  (`docparse/main.ail`) runs the implemented parser on ONE fixture and prints a deterministic
  representation, compared against the golden — collapsing the multi-fixture verify.sh model to
  a single `expected_stdout`.
- **Enabler added**: a per-benchmark `agent_env map[string]string` on `BenchmarkSpec`
  (`spec.go`), with `${WORKSPACE}` expansion, to set `MOTOKO_AST_AUTOREAD` /
  `MOTOKO_AST_READ_FULL` / `SYSTEM_MD` (none of which the standard single-file suite uses).
- **Discovery item**: the motoko exec env-build site is NOT `RunHeadlessSessionStreaming`
  (that's the Claude path, `config.ClaudePath`). The motoko dispatch routes separately
  (sets the agent CLI to the `motoko` shim). M1 must locate that exec's `cmd.Env` build and
  inject `agent_env` there — without touching the claude/gemini paths.

## High-Impact Decisions

| Decision | Why | Chosen By | Change Cost |
|---|---|---|---|
| Prompt delivery = `SYSTEM_MD` → seeded reference (not the lean syntax_reference path) | Tests the **full** fixed stack incl. the system-prompt env-forward fix; matches the reimplement run.sh strategy | agent | low |
| Single-fixture `expected_stdout` (not multi-fixture verify.sh) | Fits the existing grader with no harness change to grading | agent | low |
| `agent_env` injected only on the motoko exec path | Must not perturb claude/gemini agent envs | compiler | med |

## Solution Design / Sprint

**M1 — AgentEnv enabler (motoko exec path)** (~3h)
- `agent_env` field on `BenchmarkSpec` (done). Locate the motoko exec's `cmd.Env` build
  (the dispatch that sets the CLI to `motoko`); inject `agent_env` with `${WORKSPACE}`
  expansion. Unit test: a spec with `agent_env` exports the vars to the motoko subprocess; the
  claude path is unaffected.

**M2 — markdown reimplement benchmark** (~3h)
- `benchmarks/markdown_reimplement.yml`: `input_files` = the docparse modules (document.ail,
  the stubbed markdown_parser.ail, the helpers) + a deterministic `docparse/main.ail` that
  parses ONE fixture and prints blocks line-by-line; `agent_env` = `MOTOKO_AST_AUTOREAD=1`,
  `MOTOKO_AST_READ_FULL=markdown_parser.ail:main.ail`, `SYSTEM_MD=${WORKSPACE}/.agent_system_prompt.md`
  + seed that reference as an input_file; `expected_stdout` = golden from the real parser; `tier`,
  `tags`, `caps: [IO, FS, Env]`. Validate: `ailang eval-suite --agent --benchmarks markdown_reimplement --models motoko-local-qwen3-6-35b-a3b-mxfp8`.

**M3 — docx reimplement benchmark + nightly wiring** (~2h)
- Same pattern for docx (richer ADT — the iface fix's strongest case). Add both to the nightly
  tier the os-rotation-filler selects; confirm they run on mk-ast and land on the dashboard.

## Conflict Surface

Touches `internal/eval_harness` (the agent-runner env build) and the benchmark schema.
- `agent_env` is additive (`omitempty`) — existing benchmarks unaffected.
- The env-build change must be scoped to the **motoko** exec path; the claude/gemini paths
  build their own env (`RunHeadlessSessionStreaming` line 63, `agent_runner.go` line 373) and
  must not change. Fixtures: a claude-path benchmark (e.g. records_book) still runs identically.
- `expected_stdout` golden depends on `docparse/main.ail`'s output format — pin it.

## Success Criteria
- [ ] M1: a benchmark `agent_env` reaches the motoko subprocess; `system_prompt_built` /
  AST-autoread observable; claude path unchanged (records_book still passes).
- [ ] M2/M3: `markdown_reimplement` + `docx_reimplement` run via `ailang eval-suite --agent`
  on motoko-local, grade against the golden, and appear on the dashboard.
- [ ] No regression on the existing 74 benchmarks.

## Non-Goals
- Multi-fixture grading (one representative fixture per benchmark is enough to detect regressions).
- Porting the reimplement tasks to python/js/go (AILANG-only; they test the AILANG harness).

---
**Document created**: 2026-06-26

DESIGN_DOC_PATH: design_docs/planned/m-eval-reimplement-bench.md
