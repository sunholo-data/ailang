# Sprint Plan: M-DX-SPLIT-ARG — Compile-Time Warning for Reversed `split` Arguments

**Design doc**: [m-dx-split-argument-warning.md](./m-dx-split-argument-warning.md)
**Sprint ID**: M-DX-SPLIT-ARG
**Duration**: 1 day (~6h)
**Risk**: Low
**Created**: 2026-07-12 (mission-control iteration 17)

---

## Goal

Emit a **compile-time, non-blocking warning** when `split` is called with a short string literal
first arg and a non-literal second arg (the reversed-argument footgun — silently returns the
unsplit string). Design the warning surface to be **extensible** to other same-typed-arg swap traps.

## Reality-check discoveries (verified live at HEAD c94c67417, ailang v0.29.2)

1. **Footgun is real & reproducible**: `split("/", name)` → `["/"]` (silently wrong);
   `split(name, "/")` → `["api","keys","user123"]`. `split` is data-first `split(s, delimiter)`.
2. **No `warn_split` pass exists** — genuinely unimplemented.
3. **CONFLICT SURFACE (doc premise stale)**: the ONLY warning infra is exhaustiveness-specific.
   `result.Warnings` is `[]*elaborate.ExhaustivenessWarning` (pipeline.go:125,
   pipeline_module_compile.go:21), sourced from `elaborator.GetWarnings()`
   (internal/elaborate/core.go:233), rendered in **2 sites** that already call `warning.String()`
   (cmd/ailang/main_run_exec.go:272, internal/repl/planning.go:97). → generalize the slice element
   to an interface `Warning { String() string }`; render sites stay untouched.
4. **Detection point**: run on the **elaborated Core** where `split` is
   `App{Func: VarGlobal{Ref: {Module:"std/string", Name:"split"}}, Args:[…]}` — *before*
   lowering rewrites it to the `_str_split` builtin. Confirm module == "std/string" AND name ==
   "split" so a user function named `split` never triggers. Warning location = `App.OriginalSpan()`.
5. Core literal: `core.Lit{Kind: core.StringLit, Value: string}`. Heuristic: arg0 is a
   `StringLit` of 1–3 runes AND arg1 is NOT a `StringLit`.

## Milestones

### M1 — Generalize the warning surface (interface) — ~70 LOC + tests

- Add `internal/elaborate` interface `Warning interface { String() string }`.
- Make `*ExhaustivenessWarning` satisfy it (already has `String()` — no change needed).
- Change `pipeline.Result.Warnings` and `pipeline_module_compile.go` `Warnings` field from
  `[]*elaborate.ExhaustivenessWarning` to `[]elaborate.Warning`. Update the 3 append sites
  (pipeline_module.go:337, pipeline_module_compile.go:296, pipeline_single.go:189).
- Render sites (main_run_exec.go:272, repl/planning.go:97) call `.String()` → unchanged; verify.
- **Acceptance**: `make build` green; exhaustiveness warnings still render identically
  (regression check with a known non-exhaustive-match fixture).

### M2 — Split-arg detection pass — ~150 LOC + tests

- New file `internal/pipeline/warn_split_args.go`: a **extensible** post-elaboration walk.
  - A small table `swapTraps` keyed by `{Module, Name}` → detector fn, seeded with std/string.split
    (extensible per the doc's systemic-audit table; only split is armed now).
  - Walk the elaborated Core program (`unit.Core` / elaborated decls), find `App` nodes whose
    `Func` is `*core.VarGlobal` with `Ref.Module=="std/string" && Ref.Name=="split"`, `len(Args)==2`.
  - Heuristic: `Args[0]` is `*core.Lit{Kind:StringLit}` with 1–3 runes AND `Args[1]` is NOT a
    `*core.Lit{Kind:StringLit}` → emit `ArgOrderWarning{Location, Got, Suggestion, Note}`.
  - `ArgOrderWarning.String()` renders the doc's message format (hint: did you mean `split(x, "/")`?
    + note explaining what the reversed call does).
- Register the pass in the module pipeline (`pipeline_module_compile.go`, appending its warnings
  into `result.Warnings` next to the exhaustiveness collection) AND the single-file pipeline
  (`pipeline_single.go`) so both `run` and `check` surface it.
- **Acceptance (test plan from doc §Test Plan)**:
  - Triggers: `split(",", name)`, `split("/", path)`, `split("\n", text)`, `split("::", q)`.
  - No false positives: `split(name, ",")`, `split("hello world", " ")`, `split(a, b)`,
    `split(",", ",")`.
  - User-defined `split(x, y)` (own module) does NOT trigger (module-guard non-vacuity test).
  - Warning does **not** block compilation — program still runs (exit 0 + correct output).

### M3 — Docs + runnable example — ~40 LOC

- CHANGELOG.md entry (Added / DX).
- `examples/runnable/split_argument_order.ail`: demonstrates the wrong vs right call end-to-end,
  header comment carries the Phase-2 teaching (join vs split ordering). **Do NOT edit the embedded
  teaching prompt** — prompt-diet gate; teaching lives in the example header.
- LIMITATIONS.md: add a one-line note only if a real limitation exists (heuristic is best-effort,
  won't catch `split(delimVar, s)` where delim is a variable) — document the heuristic's bounds.
- Design doc → `implemented/v0_30_0/` on landing (executor leaves in planned/; controller moves).
- **Acceptance**: example type-checks + runs; `make verify-examples` unaffected; CHANGELOG present.

## Out of scope

- Phase 3 (`join` argument reorder) — separate breaking-change doc.
- Editing the embedded teaching prompt (prompt-diet gated).
- Arming other swap-trap functions (find/contains) — table is extensible; only split armed now.

## Success metrics

- 8+ table-driven test cases (4 trigger, 4 no-false-positive) + 1 module-guard non-vacuity test.
- Exhaustiveness-warning regression fixture still passes (M1 didn't break existing warnings).
- 1 runnable example verified.
- No new CI file-size violations; `make test` + `make lint` green.

## Risk factors

- **Low**: The generalization touches a small, well-mapped surface (5 sites). The detection pass is
  additive and read-only over Core. Main risk = false positives → mitigated by the tight heuristic
  (1–3 rune literal + module-guard) and explicit no-false-positive tests.

SPRINT_PLAN_PATH: design_docs/planned/v0_29_0/m-dx-split-argument-warning-sprint-plan.md
SPRINT_JSON_PATH: .ailang/state/sprints/sprint_M-DX-SPLIT-ARG.json
