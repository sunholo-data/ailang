# Sprint Plan: M-IFC-FUNCTION-VALUE-SINKS (#1719)

**Design doc**: [m-ifc-function-value-sinks.md](m-ifc-function-value-sinks.md) (F1–F4 ruled 2026-10-26, Mark, attended)
**Sprint JSON**: `.ailang/state/sprints/sprint_M-IFC-FUNCTION-VALUE-SINKS.json`
**Branch**: `p0/ifc-function-value-sinks` (isolated worktree, based on origin/dev `8e634480f`)
**Approval**: pre-approved by Mark (no pause). F3 binds the scope: M1 and M2b ship together.
**Estimated**: ~1,150 LOC (impl ~650, tests ~400, docs/example ~100), 4 milestones, one executor session.
**Risk**: medium-high. Semantics change to `internal/types` (new rejections), bounded by construction to
modules that declare a `{not ℓ}` or `<ℓ>` parameter.
**AILANG prompt version loaded**: recorded by the executor (`ailang prompt --version-active`) before the
example is written; the example is `ailang check`-verified, not written from memory.

## Registry reuse

| Milestone | Decision | Reason |
|-----------|----------|--------|
| M1–M4 | none | Compiler-internal IFC pass (`internal/types`); no package-shaped capability. |

## Current state (read from code at `8e634480f`)

- `ifcVar{label, typ}` is the whole abstract value (`ifc_check.go:105`). Requirements live in `c.sigs`
  keyed by name and are read only when the callee expression *is* that name (`labelOfCall`, `:377`).
- Checks A/C run only under `isLocal`; every other application takes the label-only fallback.
- `labelVar` exists in the lattice (`labels.go:34`) with no IFC caller, so it is free for `ρ`.
- 16 leaking shapes + 9 controls reproduced as files (scratchpad `dd2/r/*.ail`); they become table tests.

## Implementation shape (executor's choice under the design's deferred decisions)

One walk computes the label **and** the callable of every expression (`valueOf → ifcVal{label, fn}`),
so lambdas are walked once (no exponential re-walk of nested closures). A callable is a set of
alternatives (the join is the union of alternatives, which is at least as strict as either side by
construction). An alternative is one of:

- **top** — a module function by name; demands from its sig, plus its summary (deferred applications
  and result callable) looked up at application time from the fixed point;
- **lambda / func-literal** — demands from typed params, plus its own summary recorded during the walk;
- **declared** — demands written in a function type annotation (`string{not email} -> string`);
- **probe** — "parameter *i* of the function being summarised" (symbolic HOF argument);
- **opaque** — imported / unknown function (triggers the conservative arg-join fallback, F2).

Every parameter of a summarised body is seeded with a label variable `ρ`. An application whose
argument labels contain an active `ρ` records a **deferred application** on the innermost summary
(that is the applied-at summary of M2a when the target is a probe, and the inferred demand of M2b when
the target carries demands). Applying a summarised callable substitutes `ρ ↦ actual labels` and
`probe ↦ actual callables` and replays its deferred applications, so errors land at the call site that
supplies the concrete label. `ρ` is erased from every result label, Check B input and diagnostic.

## Milestones

### M1 — Callable values + application-site checks (~350 LOC impl, ~150 test) — day 1
- `internal/types/ifc_callable.go`: `ifcVal`, `ifcCallable`/alternatives, join, `fromSig`,
  `fromFuncType`, demand check (Check A/C) with `via` in messages.
- `ifc_check.go`: `flowOf/labelOf/handOff/walkBlock` return `ifcVal`; identifiers resolve `env` →
  `c.sigs` (value position) → opaque; callables flow through let/letrec/block/if/match/record/record
  update/access/list/array/tuple/ctor/pattern vars; Checks A/C on the callee **value** at every
  application; direct-call message unchanged byte-for-byte (R0/C0 + `leak_attempt` expected-fail).
- Tests (`ifc_function_value_test.go`): R0, R1, R3–R9, R14, C0, C1, N1, N4–N6.
- **Acceptance**: listed rows pass; all existing `TestIFC*` green; `make test-changed` green.

### M2 — Summaries: applied-at (M2a) + wrapper inference (M2b) (~300 LOC impl, ~150 test) — day 2
- `internal/types/ifc_summary_local.go`: `ρ` seeding, owner stack, deferred applications, substitution,
  replay with depth guard, module fixed point (source order, bounded iterations), arg-join fallback
  for opaque callees, `ρ` erasure.
- Tests: R2, R10, R11, R12, R13, C2, N2, N3; mutual-recursion wrappers converge;
  `TestIFC_TransparentPassthrough_NoError` unchanged; assertion that no `labelVar` reaches a
  diagnostic or `effectiveBodyLabel`.
- **Acceptance**: all 25 repro rows match the design tables as unit tests **and** as fresh-file CLI
  checks with a scratch-built binary (cache cleared); REPL gate parity pinned.

### M3 — Mutation test + corpus delta (~0 LOC shipped; evidence) — day 2–3
- Mutation via scratch copies (never `git checkout <file>`): (a) disable the application-site check →
  R1, R3–R9, C1 must fail; (b) disable applied-at replay → R2, R10, C2 must fail; (c) disable
  inference recording → R11–R13 must fail. Record results in the sprint JSON and PR.
- Corpus delta: base binary (origin/dev) vs branch binary, `ailang check` over every `.ail` in this
  repo (examples, std, benchmarks) and read-only sibling repos under `~/dev/sunholo-data/`
  (`ailang-demos`, `ailang-packages`, `ailang-motoko`, …). Diff of diagnostics goes in the PR; every
  new diagnostic triaged (leak vs over-approximation). Flag-free form chosen (two binaries) instead of
  a new env-gated test, so no new env var enters the Registry.
- **Acceptance**: zero new diagnostics in modules with no refinement/positive param label.

### M4 — Docs, example, changelog (~100 LOC) — day 3
- `examples/runnable/secrets/sink_function_value.ail` (expected-fail: alias + HOF + lambda wrapper)
  and `examples/runnable/secrets/sink_function_value_ok.ail` (N-rows, passing) + manifest entries.
- `docs/docs/guides/ifc-labels.mdx`: remove the "unannotated parameters" limitation, add
  "Sinks as values" (incl. the conservative fallback for imported HOFs until #1771) and a
  cross-module scope note on "Why Identity Bypass Doesn't Work".
- `changelogs/unreleased/2026-10-26-ifc-function-value-sinks.md` (`### Fixed`).
- Prompt line (design M3 bullet) is **deferred to prompt-manager**: a new prompt version is a release
  artifact that changes eval inputs, so it is not cut inside a P0 code PR.
- **Acceptance**: `make verify-examples`, `make test-imports`, `make lint`, `make check-file-sizes`,
  `make check-changelog` green.

## Validation (executor gates — never full `make test`)

`make test-changed` · `make lint` · `make verify-examples` · `make test-imports` ·
`make check-file-sizes` · `make check-changelog` · focused `go test ./internal/types/ ./internal/repl/`.

## Risks

| Risk | Mitigation |
|------|-----------|
| Over-approximation (opaque fallback, joined callables, record-access callables) rejects safe code | Corpus delta triaged in PR; human ruling needed for any over-approx before merge |
| `ρ` escapes into a label | single erase function; assertion test |
| Fixed point slow on large modules | only modules with demands iterate past one pass; bounded iterations |
| Conflict with PR #1771 (`labelOfCall`) | This lands first; #1771 rebases its imported branch into identifier resolution (F4 already posted) |
