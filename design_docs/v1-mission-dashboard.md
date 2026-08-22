# Mission Dashboard — V1

_Snapshot; overwritten every iteration. History lives in `v1-mission.md` (STATUS) and `v1-mission-log.md`._

**Last iteration:** 250 · 2026-08-22 · M3 LANDED · evaluator 91/100 PASS r1, zero blocking

## Latest

- **Release:** v0.33.1 · `dev` @ [`bbf672df0`](https://github.com/sunholo-data/ailang/commit/bbf672df0) · 22 checks, 4/4 required pass, only Sonar (non-required) not green
- **Just landed:** [#828](https://github.com/sunholo-data/ailang/pull/828) — M3, typed aggregate preservation. `Array[T]` kept its identity for literals and helpers after M1/M2 but lost it at every *typed* boundary. Three measured divergences — an ADT field, a named-record field, and an `Array[UserADT]` — all now agree byte-for-byte. Lists are pinned untouched by one generated line carrying both halves: `&Both{Arr: tmp8.(ArrayVal), Lst: ConvertToInt64Slice(tmp9)}`.
- **Key find:** the sprint plan enumerated **2** array type mappers and the design doc called the sprint "self-contained to `internal/gen/golang/`". There are at least **7**. `cmd/ailang/compile_types.go` holds a second, duplicate mapper that populates the *call-site* registry while `adt.go` writes the *declaration* — two independent sources of truth for one fact, and the generated Go does not compile unless they agree. The judge then found two more in `internal/gen/lower/typeres.go`, inert today behind the off-by-default `--emit-go-v2`.

## In flight / next

1. **`m-array-show-diverges-run-vs-compile` M4** — CHANGELOG, doc move to `implemented/v0_34/`, VL-9 correction, adjacent-defect rows. **Take the coverage row below as M4's first task**, not as an afterthought.
2. `m-array-typed-boundary-lines-unpinned` **[NEW]** — two lines shipped in M3 that nothing kills: reverting `types.go`'s `TArray` case reds only its own unit test (golden + differential stay green); reverting `IsUserDefinedType`'s `"ArrayVal"` case reds **nothing at all**. Both reproduced first-party. SonarCloud independently flagged the same gap (66.7% new-code coverage vs an 80% bar) — two instruments, one finding.
3. `m-duplicate-go-type-mappers` **[NEW]** — 7+ array→Go mapping sites across 4 packages. The class defect behind this sprint's scope miss.
4. `m-list-of-array-compiled-panic` **[NEW]** — `[Array[int]]` panics in the compiled backend. **Pre-existing**, two-arm verified (base binary `not [][]int64`, post-M3 `not []main.ArrayVal`). Not an M3 regression.
5. `m-codegen-claim-must-match-source` with `m-list-builtins-codegen-only`.
- **Blocked-external, predicate re-run 2026-08-22:** `m-wasm-deterministic-typecheck-budget` — `#662` still OPEN, 1 comment, ours, `2026-08-18` (control `#613` = 2). Unchanged.

## Loop / routing

- Controller `claude:claude-opus-5` · designer ROTATION (pointer at `codex:gpt-5.6-sol`) · planner `codex:gpt-5.6-sol` · executor `codex:gpt-5.6-sol` · evaluator `sonnet`.
- generator≠judge held: OpenAI executor, Anthropic judge, each in its own worktree.
- **Standing hazard:** the shared `~/go/bin/ailang` is dozens of commits stale and reds the golden suite for reasons unrelated to any diff — that suite shells out to a bare `ailang`, so there is no `--version` to check. Build to a scratch dir and prepend to `PATH`; never `make quick-install` mid-iteration.

## Parked on Mark

- **`D-22`** — do LC-2…LC-5 build for `C1` (plain cons cells) or `C2K32` (chunked, K=32)? One word. *(`D-23` was answered `yes` and exercised at iteration 249; the main checkout has been 0 ahead / 0 behind ever since, including this iteration's fast-forward.)*

## Quota / cost

- metered **$0.00** of $5 this iteration — every lane was a quota bucket. Billing tripwire CLEAN.
- Bookkeeping [#745](https://github.com/sunholo-data/ailang/issues/745), 59 comments; rotates at the next Monday-07:00 CEST boundary (2026-08-24) or >80 comments.
