# Mission Dashboard — V1

_Snapshot; overwritten every iteration. History lives in `v1-mission.md` (STATUS) and `v1-mission-log.md`._

**Last iteration:** 249 · 2026-08-22 · M1+M2 LANDED · evaluator 93/100 PASS r1, zero blocking

## Latest

- **Release:** v0.33.1 · `dev` @ [`d3b0185f5`](https://github.com/sunholo-data/ailang/commit/d3b0185f5) · 22 checks, only Sonar (non-required) not green
- **Just landed:** [#826](https://github.com/sunholo-data/ailang/pull/826) — M1+M2 of the array-`show` divergence. Arrays compile to a distinct Go type (`ArrayVal`, the in-repo `Tuple` precedent), so a compiled binary now prints `#[1, 2, 3]` as the interpreter always did: `cmp` between backends rc=0, against rc=1 at base. Five generated converters that silently returned `nil` on a type mismatch now `panic` with a converter-specific message (Principle 2).
- **Key find:** the milestone's own M1 unit test **passed for a program containing no array at all** — its assertions were unconditional runtime-preamble boilerplate. Found by the judge, reproduced first-party (`core.Array`→`core.List`: lands, builds, still rc=0), repaired over the emitted literal. My own mutants aim at the code; this was a defect in the test.

## In flight / next

1. **`m-array-show-diverges-run-vs-compile` M3** — typed aggregate preservation. Carries the sprint's whole risk (plan §2 R2/R3/R4) and closes the measured `MkBox(Array[int])` defect. Then M4 (fixture count 6 → 7, CHANGELOG).
2. `m-codegen-claim-must-match-source` with `m-list-builtins-codegen-only`.
3. New rows from this judge: `m-array-record-slice-converter-arm-untested` (the one mutant of six nothing killed — may be unreachable, so the deliverable is the reachability verdict) · `m-emitter-lint-evadable-by-rewording` (class audit of source-text gates standing in for behavioural ones).
- **Blocked-external, predicate re-run 2026-08-22:** `m-wasm-deterministic-typecheck-budget` — `#662` still OPEN, 1 comment, ours, `2026-08-18` (control `#613` = 2). Unchanged.

## Loop / routing

- Controller `claude:claude-opus-5` · designer ROTATION (pointer at `codex:gpt-5.6-sol`) · planner `codex:gpt-5.6-sol` · executor `codex:gpt-5.6-sol` · evaluator `sonnet`.
- generator≠judge held: OpenAI executor, Anthropic judge, each in its own worktree.
- **Standing hazard:** the shared `~/go/bin/ailang` is dozens of commits stale and reds the golden suite for reasons unrelated to any diff — that suite shells out to a bare `ailang`, so there is no `--version` to check. Build to a scratch dir and prepend to `PATH`; never `make quick-install` mid-iteration.

## Parked on Mark (both re-asked unchanged)

- **`D-22`** — do LC-2…LC-5 build for `C1` (plain cons cells) or `C2K32` (chunked, K=32)? One word.
- **`D-23`** — does `D-16`'s fast-forward authorisation extend to *local dev AHEAD but every ahead-commit content-duplicated upstream*? One word `yes`/`no`. The main checkout is **9 behind** today, and the running skill resolves through it.

## Quota / cost

- metered **$0.00** of $5 this iteration — every lane was a quota bucket. Billing tripwire CLEAN.
- Bookkeeping [#745](https://github.com/sunholo-data/ailang/issues/745), 56 comments; rotates at the next Monday-07:00 CEST boundary (2026-08-24) or >80 comments.
