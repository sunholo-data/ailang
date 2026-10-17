# M-PROCESS-RESULT-SEMANTICS: explicit start / completion / truncation semantics for `std/process.exec`

- **Date**: 2026-10-17 (triage iteration 6, coordinator task-48f9aa82)
- **Class**: bug-consolidation (design request; routed through design-doc-creator; the triage loop approves, rejects and merges nothing)
- **Recommend**: design-doc → operator review → (on approval) sprint-planner
- **Consolidates**: [#1878](https://github.com/sunholo-data/ailang/issues/1878) (P1), [#1879](https://github.com/sunholo-data/ailang/issues/1879) (P1), [#1880](https://github.com/sunholo-data/ailang/issues/1880) (P2)
- **Searched** (duplicate gate): `create_planned_doc.sh` doc search on "process result semantics" → **no matches** (SimHash + neural, implemented and planned). Manual: `ErrWaitDelay|OutputLimitExceeded|partial output|process result semantics` over `design_docs/` and `docs/` → hits only in docs that *describe the current behaviour* (`docs/docs/reference/effects.md`, `design_docs/implemented/v0_8_1/m-process-exec.md`), not in any doc that *fixes* it; `grep -rln "1878\|1879\|1880" design_docs/ docs/` → all hits are SHA substrings / line numbers, false positives. Nearest neighbours, read and confirmed distinct: `design_docs/implemented/v0_8_1/m-process-exec.md` (the original Process effect — this doc's *premise*), `design_docs/planned/v0_54_7/spawn-process-result.md` (the `spawnProcess`/`spawnProcessResult` call, a different surface), `design_docs/planned/v1_1_0/m-process-modes.md` (replay-contract `[mode=...]` modes, orthogonal), `design_docs/implemented/v0_54_4/m-background-worker-lifecycle.md` (worker lifecycle), `process-subcmd-allowlist-already-shipped.md` (shipped v0.38.0). No coordinator duplicate: `ailang coordinator pending` is unavailable in this sandbox (CGO-stub binary), so the repo's design-doc backlog was searched directly instead.
- **Evidence**: defects verified on origin/dev `0384450a6b5cb2d852df7e7b04fdb754772a6f25` (= release v0.54.15, 2026-10-17) by read-only codex:gpt-6-luna verification and independent openrouter:minimax/minimax-m3 judge (10 CONCUR / 0 DISSENT) — [triage iteration 6 evidence comment](https://github.com/sunholo-data/ailang/issues/1835#issuecomment-6240929372). All code citations below were re-read at that SHA in this session.

**Status**: Planned
**Target**: next minor after v0.54.x (breaking ADT change — see HD-2; must NOT ship as a silent patch)
**Priority**: P1 (driven by #1878/#1879; #1880 is the P2 documentation half of the same semantics)
**Estimated**: 3–4 days (~10h implementation + 6h tests + 3h docs + buffer)
**Dependencies**: none (self-contained in `internal/effects/process.go` + `std/process.ail` + `internal/builtins/process.go`)

## Problem Statement

`std/process.exec` classifies outcomes by *which Go error came back*, not by *what happened to the target program*. Three issues, one root cause: the result type does not carry the three facts every caller of a subprocess needs — **did the program start**, **did it complete under its own control**, and **was the captured output truncated**. Today those facts are destroyed, inferred, or misreported:

1. **#1878 — captured output and exit status are discarded on `Timeout` and `OutputLimitExceeded`.** `internal/effects/process.go` returns `Err(Timeout(durationMs))` (Step 6, ~line 227) and `Err(OutputLimitExceeded(totalBytes))` (Step 5, ~line 220) while `stdoutBuf`/`stderrBuf` hold everything captured up to that point and — in the over-limit case — the process has often already exited with a real status. `ProcessOutput.truncated` was *designed* for exactly this ("true if output was capped", `design_docs/implemented/v0_8_1/m-process-exec.md` line 107) but every `makeProcessResultOk` call site passes `false` (verified: the only `truncated` binding in `process.go` is the parameter, always called with `false`). Additionally the `--process-max-output` help promises "Maximum stdout+stderr bytes **before kill**" (`cmd/ailang/main_run.go:108`) but `limitedWriter` silently discards past the cap and never kills — the repro in #1878 shows a command running past the limit and exiting 0.
2. **#1879 — a command that ran to completion and exited 0 is reported as `SpawnFailed`.** When a background child holds the stdout pipe past `cmd.WaitDelay`, `cmd.Run()` returns `exec.ErrWaitDelay`. That error is not an `*exec.ExitError`, not `exec.ErrNotFound`, and does not contain "permission denied", so it falls through the catch-all to `Err(SpawnFailed("exec: WaitDelay expired before I/O complete"))` (~line 257) — although Go documents `ErrWaitDelay` as returned only when the process otherwise exited successfully. The asymmetry is observable: the *same* command exiting non-zero returns `Ok` with its exit code and output.
3. **#1880 — the docs don't say which `ProcessError` arms prove the program never started, and the code doesn't guarantee it.** `NotAllowed` and `NotFound` are pre-start by construction. But `PermissionDenied` is assigned by `strings.Contains(err.Error(), "permission denied")` on the error of `cmd.Run()`, which merges start and wait errors; and `SpawnFailed` is the catch-all that (per #1879) already contains a post-start case. Callers cannot soundly tell a user or model "nothing ran".

**Downstream impact (measured by the consumer on record):** motoko_agent at `e69aaad7844c122d268100bc46aadc79ac9422d1` renders every `ProcessError` as `exit_code: 1` with empty stdout (`src/core/tool_runtime.ail`, `run_process_result`). A verbose build that over-runs the 10 MB cap and *succeeds* is reported to the model as "failed, printed nothing" — the model's next move is to run it again: work already done gets repeated, and "write succeeded, reported failed" misrepresents the world. motoko is forced to mark these results "effects unknown"; it could say "effects none" only where the language guarantees it, and today only `NotAllowed`/`NotFound` carry that guarantee.

**The systemic audit (all seven arms, not just the three reported):** reading Step 6 end-to-end, *three* arms discard captured output (`Timeout`, `OutputLimitExceeded`, and also `AbnormalExit` — a `kill -9 $$` loses everything the process printed before the signal). Any fix that widens only the two reported arms leaves the third as the next issue; this doc treats all of them.

## Goals

**Primary Goal:** Make `std/process.exec` report the three facts explicitly — started / completed / truncated — for every outcome, so a caller can decide "did effects happen?" from the returned value alone, with the guarantee stated in the module docs.

**Success Metrics:**
- Every arm that can occur after the program started carries the captured stdout/stderr (up to the cap) and the process's exit status where one exists.
- The documented guarantee holds by construction: no post-start outcome maps to a pre-start arm (`NotFound`, `PermissionDenied`, `SpawnFailed`), and the docs state it per arm in `std/process.ail` and `docs/docs/reference/effects.md`.
- `ProcessOutput.truncated` is `true` exactly when captured output was capped by `--process-max-output` or cut by pipe force-close — never a hard-coded `false`.
- The three repro commands from #1878/#1879 (partial output before timeout; over-limit successful command; exit-0-with-held-pipe) return values consistent with the new semantics, verified by tests in the sprint.
- motoko's `BashExec` can distinguish "effects none" (pre-start arms) from "effects happened, ended like this" without string-matching error text.

## High-Impact Decisions

| Decision | Why High Impact | Chosen By | Deadline | Change Cost |
|---|---|---|---|---|
| HD-1: Split `cmd.Run()` into `cmd.Start()` + `cmd.Wait()`; classify `NotFound`/`PermissionDenied`/`SpawnFailed` **only** from `Start`'s error (using `errors.Is(err, fs.ErrPermission)` instead of the substring match) | Makes the #1880 guarantee true by construction; kills the #1879 misclassification at the root | human (design) | design | med |
| HD-2: How partial output travels on post-start arms — (A) widen the ADT constructors vs (B) Ok-for-everything-that-started with a `status` field | **Verified in this session: pattern-arity mismatches pass `ailang check` and silently fall through at eval** (V8). Any ADT shape change is a *silent* break for every consumer's match — the choice determines motoko's migration and the release's breaking-change note | human (design — this is the operator's main review point) | design | high |
| HD-3: `OutputLimitExceeded` for a process that completes on its own → `Ok` with real exit code, first N bytes, `truncated: true` (the field's designed purpose) | Changes an error into a success for a whole class of real commands; matches the #1878 reporter's expectation | human (design) | design | med |
| HD-4: Kill at the output limit (align with the `--process-max-output` "before kill" help and `m-process-exec`'s "termination is safer" ruling) vs fix the help text to match today's run-to-completion | Decides whether an over-limit process keeps having effects after we've stopped recording them — a determinism/authority question, not a cosmetic one | human (design) | design | med |
| HD-5: `exec.ErrWaitDelay` after a successful exit → `Ok` with the real exit code and `truncated: true` (or a dedicated pipe-close flag) | The command's effects happened; reporting SpawnFailed is a falsehood | agent (the issue's suggested fix is sound) | design | low |
| HD-6: Which arms count as "never started" in the documented guarantee table | motoko's "effects none" marker is only sound if the table is exactly right | human (design) | design | low |

### Design Freeze

- [ ] HD-2 chosen: ADT widening (A) vs status-record (B) — must be locked before coding; it determines every other signature
- [ ] HD-4 chosen: kill-at-limit vs capture-to-completion
- [ ] HD-1 confirmed (Start/Wait split) — expected to be uncontroversial, but it changes `PermissionDenied`'s detection from substring to `errors.Is`, so the arm's meaning shifts subtly and needs sign-off
- [ ] HD-6 guarantee table drafted into `std/process.ail`

## Deferred Decisions

- Exact field/constructor naming (`TimedOut` vs widened `Timeout`; `pipesClosed` vs reusing `truncated`) — agent may choose, subject to the HD-2 shape
- Whether `AbnormalExit` also gains the captured buffers in the same pass or a follow-up (recommended: same pass — see systemic audit) — agent may choose
- Test organization (which repro commands become which fixtures) — agent may choose
- Whether the stderr budget bug (V7: `pc.MaxOutput - int64(stdoutBuf.Len())` is computed once at setup, when `stdoutBuf.Len()` is 0, so the two streams get independent caps summing to 2× the flag) is fixed here or routed separately — human at review
- The pattern-arity checker gap (V8) is a *separate language defect discovered during verification — route it to its own triage row, not this sprint*

## Solution Design

### Overview

Re-classify `processExec`'s outcome space along the three facts, then make the value carry them. The effect row (`! {Process}`), the capability gate (`--caps Process`), the allowlist/confine authorizer, the process-group containment (`proctree.ConfigureContained`) and the `WaitDelay` bound are all **unchanged** — this design touches only how a finished attempt is *reported*, never when a process is *permitted*.

**The target semantics — one table for all seven arms:**

| Arm | Program started? | Completed under own control? | Carries captured output? | Carries exit status? |
|---|---|---|---|---|
| `NotAllowed` | never (refused pre-`exec`) | — | n/a | n/a |
| `NotFound` | never | — | n/a | n/a |
| `PermissionDenied` | never (post-split: only from `Start`) | — | n/a | n/a |
| `SpawnFailed` | never (post-split: only from `Start`) | — | n/a | n/a |
| `Timeout(ms, …)` | yes | no — we killed the group at the deadline | yes — partial stdout/stderr up to the cap | killed (no exit code exists) |
| `OutputLimitExceeded` | yes | per HD-4: killed at cap (no exit code) — or, if the process completed first, HD-3 applies and it's `Ok` | yes — first N bytes | per HD-3/HD-4 |
| `AbnormalExit(sig, name, …)` | yes | no — died to a signal | yes — partial output | signal number/name |
| `Ok(ProcessOutput)` | yes | yes | full or truncated | real exit code, `truncated` set honestly |

The invariant the docs will state: **`Err` ⇒ the target program never executed** (post-split, true by construction); **`Ok` ⇒ it ran to an exit status we observed**. That is exactly the "effects none vs effects happened" line motoko needs.

### Architecture

**Components (all in the existing files — no new packages):**

1. **Start/Wait split in `processExec`** (`internal/effects/process.go`): replace `cmd.Run()` with `cmd.Start()` + `cmd.Wait()` (in-repo precedent: `internal/effects/stream_process.go:77` already does exactly this for the streaming source). Only `Start`'s error may produce `NotFound` / `PermissionDenied` (`errors.Is(err, fs.ErrPermission)` and `errors.Is(err, exec.ErrNotFound)`) / `SpawnFailed`. Everything after a successful `Start` is by definition post-start and can never map to those arms. `cmd.Cancel` must be set so the context deadline still kills the group (today `exec.CommandContext` wires that automatically; with an explicit Start/Wait the same `execCtx` continues to drive it — the sprint must keep the v0.54.14 group-kill on timeout, regression-tested by `internal/effects/process_timeout_group_test.go`).
2. **Partial-output plumbing (HD-2)**: the `Err` construction helpers grow a partial-output variant used by the three post-start arms. Under shape (A) the constructors widen in `std/process.ail`, e.g. `Timeout(int, bytes, bytes)` / `AbnormalExit(int, string, bytes, bytes)`; under shape (B) `ProcessOutput` gains a `status` field (`Completed | TimedOut | Signaled | …`) and only pre-start failures remain in `ProcessError`. **The choice is the operator's; the verification log's V8 finding (silent fall-through) applies to either.**
3. **Honest truncation**: `limitedWriter.exceeded` flows to `makeProcessResultOk`'s `truncated` parameter (today always `false`); `exec.ErrWaitDelay` maps to `Ok` with the real exit code from `cmd.ProcessState` and truncation flagged (HD-5).
4. **Kill-at-limit (HD-4, if chosen)**: when either limited writer trips, invoke the same group-kill path the timeout uses, then report `OutputLimitExceeded` with the captured prefix; if the process exits on its own first, HD-3's `Ok` applies.
5. **Docs as contract**: `std/process.ail` header comments gain the per-arm guarantee table; `docs/docs/reference/effects.md` (which currently shows `Err(Timeout(ms))` at line ~313) and the builtin `LongDesc` in `internal/builtins/process.go` are updated to match.

### Implementation Plan

**Phase 1: Split and classify** (~4 hours)
- [ ] Start/Wait split; pre-start arms classified from `Start` only; `errors.Is`-based `PermissionDenied`
- [ ] `ErrWaitDelay` → `Ok` with real exit code + truncation flag (HD-5)
- [ ] Existing tests green: `process_test.go`, `process_timeout_group_test.go`, `process_confined*_test.go` (assertions are on constructor *names* only — verified — so they survive unless HD-2 renames)

**Phase 2: Partial output + truncation** (~4 hours)
- [ ] HD-2 shape implemented per the frozen decision; buffers reach every post-start arm
- [ ] `truncated` set from `limitedWriter.exceeded`; HD-3 `Ok`-with-truncation for completed over-limit commands; HD-4 kill-at-limit if chosen
- [ ] `std/process.ail` ADT + `internal/builtins/process.go` type/metadata updated in lockstep (the `T.App("Result", outputType, T.Con("ProcessError"))` signature in `makeProcessExecType` is generated from the same shape)

**Phase 3: Contract docs + repro tests** (~5 hours)
- [ ] Guarantee table in `std/process.ail`, `docs/docs/reference/effects.md`, builtin `LongDesc`
- [ ] One regression fixture per repro row from #1878/#1879 (partial output before timeout; over-limit exit 0; exit-0-with-held-pipe; `kill -9` with prior output)
- [ ] `make test`, `make fmt`, `make lint`, `make check-boundaries` green

### Files to Modify/Create

**Modified files:**
- `internal/effects/process.go` (+60/−25 LOC) — Start/Wait split, classification, partial-output helpers, honest `truncated`
- `std/process.ail` (+25/−10 LOC) — ADT shape (HD-2), guarantee table, example matches
- `internal/builtins/process.go` (+10/−5 LOC) — `makeProcessExecType` + `LongDesc` metadata
- `docs/docs/reference/effects.md` (+15/−5 LOC) — semantics table
- `internal/effects/process_test.go` (+~120 LOC) — repro fixtures from all three issues

**No new files.** Effect row, capability check, allowlist, confinement and process-group behaviour untouched.

## Examples

**Before (#1879 repro, verified by the reporter on v0.52.5 / code-read at v0.54.15):**

```ailang
exec("bash", ["-lc", "echo DONE; echo landed > marker_fg; (sleep 9; true) & exit 0"])
-- → Err(SpawnFailed("exec: WaitDelay expired before I/O complete"))   -- after ~5 s
-- marker_fg exists. The command exited 0. The model is told it "never started".
```

**After (HD-5):** `Ok({stdout: "DONE\n", stderr: "", exitCode: 0, truncated: true, resolvedPath: "/usr/bin/bash"})` — the caller sees the work happened.

**Before (#1878 repro):** `exec("bash", ["-lc", "echo PARTIAL; sleep 30"])` under `--process-timeout 2s` → `Err(Timeout(2002))` — the 8 captured bytes are gone.

**After (shape A):** `Err(Timeout(2002, "PARTIAL\n", ""))` — the caller sees how far the command got. Under shape (B): `Ok({status: TimedOut, stdout: "PARTIAL\n", …})`. Either way, motoko can render partial output and mark "effects happened" instead of "effects unknown".

## Success Criteria

- [ ] The three repro commands from #1878/#1879 return values consistent with the target table (one test fixture each, in `process_test.go`)
- [ ] No code path after a successful `cmd.Start()` can produce `NotFound` / `PermissionDenied` / `SpawnFailed` (asserted by a test that runs a held-pipe exit-0 command)
- [ ] `ProcessOutput.truncated` is `true` iff output was capped or pipe-cut (no `false` literal remains at `makeProcessResultOk` call sites)
- [ ] `std/process.ail` and `docs/docs/reference/effects.md` state, per arm, whether it proves the program never started
- [ ] `make test`, `make fmt`, `make lint`, `make check-boundaries` all pass
- [ ] Breaking-change note drafted (HD-2) with motoko's migration path (`tool_runtime.ail` `run_process_result` at `e69aaad`)

## Conflict Surface

`internal/effects/` is on the mandatory-conflict-surface list. This change touches **no grammar and no type-inference position** — the conflict surface is the **stdlib ADT shape and its consumers' pattern matches**.

### Positions touched

- `ProcessError` constructor arities in `std/process.ail` (only under HD-2 shape A; shape B changes the arm set instead)
- The `Err`-side construction helpers in `internal/effects/process.go` and the generated builtin signature in `internal/builtins/process.go:makeProcessExecType`

### What else lives here

| Consumer | Current use | Effect of shape change |
|---|---|---|
| `motoko_agent` `tool_runtime.ail` `run_process_result` (`e69aaad`) | matches all seven arms | **Verified silent-break risk (V8):** too-few/too-many sub-patterns pass `ailang check` and the arm falls through at eval. A widened `Timeout` makes motoko's `Timeout(ms)` arm silently stop firing; motoko must migrate in lockstep |
| `docs/docs/reference/effects.md` (~line 313) | documents `Err(Timeout(ms))` | doc text must move with the shape |
| `std/process.ail` doc-comment example | `Err(Timeout(ms)) => println("timed out")` | same |
| `internal/effects/process_test.go:261`, `process_timeout_group_test.go:37` | assert constructor **name** only (`unwrapErr` returns ctor + fields; tests check the name) | survives arity changes; must be extended, not rewritten |
| `examples/runnable/process_subcmd_allowlist.ail` | `NotAllowed` arm only | arity unchanged — safe (fixture must still pass) |
| `spawnProcessResult` / `spawnProcess` paths | share `Authorize` and `ProcessError` for pre-start arms | **untouched by design** — post-start arms never occur on that call; regression fixture: `process_spawn_result_test.go` |
| `cmd/ailang` serve-api / run-policy tests (`serve_api_process_test.go`, `run_policy_*`) | exercise the capability/allowlist plumbing | no signature change on those paths |

### Disambiguation strategy

N/A (no parser/typechecker grammar change). The ADT remains a closed sum in one module; exhaustiveness is enforced by the existing match checker against whatever shape HD-2 freezes.

### Programs that MUST still work

1. `examples/runnable/process_subcmd_allowlist.ail` (allowlist + `NotAllowed`)
2. `internal/effects/process_timeout_group_test.go` scenario (timeout group-kill of grandchildren)
3. `internal/effects/process_spawn_result_test.go` (shared-authorizer spawn surface)
4. The 8-arm exhaustive match from the #1878 repro (compiles today — verified V8a)
5. motoko `BashExec` happy path (`Ok` arm) — unchanged

### What deliberately changes

- Under HD-2 shape A: the arity of `Timeout` / `OutputLimitExceeded` / `AbnormalExit` widens — **a silent break at match time (V8), so the release note must be loud and motoko must migrate in the same window.**
- Under HD-3: over-limit-but-completed commands flip from `Err(OutputLimitExceeded)` to `Ok` with `truncated: true` — an intentional semantic change, the reporter's preferred shape.
- Under HD-1: `PermissionDenied` stops being substring-matched; a wait-phase "permission denied" error (if one ever surfaces) now classifies as post-start, not pre-start. This is the fix for #1880, not a regression.

Anything else that breaks is a regression.

## Testing Strategy

**Unit tests** (`internal/effects/process_test.go` + new repro fixtures):
- One fixture per #1878/#1879 repro row: partial-output timeout, over-limit exit-0, held-pipe exit-0 (→ `Ok`, exit code real, `truncated: true`), held-pipe exit-nonzero (still `Ok` — the pre-existing asymmetry case), `kill -9` with prior output, pre-start `NotFound`/`PermissionDenied` (a mode-644 script, from #1880's table)
- Post-split invariant: no post-`Start` error maps to a pre-start arm

**Integration/regression:**
- The five Conflict-Surface fixtures above, pinned
- `make check-boundaries` (effects layering)
- Windows skip rules mirror the existing `process_test.go` `runtime.GOOS` guards

**Manual:**
- Run the #1878/#1879 repro programs verbatim from the issues against the new binary

## Non-Goals

- **Not changing the effect/capability boundary**: `! {Process}`, `--caps Process`, `--process-allowlist`, confinement and `proctree` group-kill are untouched (explicit directive constraint).
- **Not touching `spawnProcess`/`spawnProcessResult`/`cancelProcess`** — different calls, already routed in `spawn-process-result.md` (v0.54.7); their pre-start arms already have the guarantee this doc generalises.
- **Not fixing the pattern-arity checker gap (V8)** — separate language defect, to be routed to its own triage row; this doc only depends on and documents it.
- **Not redesigning `WorkerCancelError`** — cancellation of managed workers keeps its own taxonomy.
- **Not adding streaming/incremental output** — the capture model stays buffered-with-cap.

## Timeline

**Week 1** (~15 hours): Phase 1 (4h) + Phase 2 (4h) + Phase 3 (5h) + review buffer (2h)
Total: ~15 hours across one sprint.

## Risks & Mitigations

| Risk | Impact | Mitigation |
|---|---|---|
| ADT shape change silently breaks consumers (V8 — verified) | High | Loud release note; motoko migrates in lockstep; repro fixtures pin the new shapes; consider a deprecation window only if the operator picks shape A (deferred) |
| Start/Wait split subtly changes timeout group-kill timing | High | Keep `exec.CommandContext(execCtx, …)` driving cancellation; `process_timeout_group_test.go` is the regression fixture |
| `PermissionDenied` reclassification narrows an arm users relied on | Medium | #1880's own measurement table shows every produced case was a start failure; fixture each case from that table |
| Kill-at-limit (HD-4) changes observable behaviour for long-running over-limit commands | Medium | The flag help already promises "before kill"; fixture both orders (kill vs natural exit) regardless of the choice |
| stderr budget bug (V7) confuses the truncation tests | Low | Decide its routing at review (deferred); tests assert against the *documented* single total cap |

## Axiom Compliance

**Canonical reference:** [Design Axioms](/docs/references/axioms)

### Axiom Scoring

| Axiom | Score | Justification |
|-------|-------|---------------|
| A1: Determinism | +1 | The result value becomes a faithful function of what happened; today identical executions report different *kinds* of outcome (exit 0 vs SpawnFailed) depending on pipe timing |
| A2: Replayability | 0 | Trace payloads gain fields; replay structure unchanged |
| A3: Effect Legibility | 0 | Effect row unchanged (by directive) |
| A4: Explicit Authority | +1 | The never-started guarantee makes capability denials (`NotAllowed`) *provably* effect-free — the authority system's refusal now carries a semantic warranty |
| A5: Bounded Verification | 0 | No type-level verification change |
| A6: Safe Concurrency | 0 | Group-kill behaviour preserved; no concurrency change |
| A7: Machines First | +1 | This is the core of the issue: a model consuming the result can decide "did effects happen?" from the value's shape, without parsing error strings or guessing — exactly the machine-decidability the language exists for |
| A8: Minimal Syntax | 0 | No new grammar |
| A9: Cost Visibility | +1 | A timeout/over-limit result now shows the partial work already paid for; repeated work caused by false "failed" reports stops |
| A10: Composability | +1 | Same `Result`/ADT machinery; `spawnProcessResult` inherits the shared pre-start guarantee |
| A11: Structured Failure | +1 | **Primary goal.** Failure taxonomy gains the missing axis (started/completed/truncated) instead of a catch-all that lies |
| A12: System Boundary | +1 | The host-OS crossing's outcome is fully reported; the boundary's failure modes become typed facts |

**Net Score: +7** → **Decision: Proceed (pending operator approval — this doc is a design request; nothing is approved here)**

### Hard Violation Check

- [x] A1: no new nondeterminism — the change *removes* outcome-shape nondeterminism
- [x] A3: no hidden effects; effect row unchanged
- [x] A4: no ambient access; capability boundary preserved by design
- [x] A7: strictly improves machine decidability of outcomes

## Verification Log

| # | Claim | Evidence |
|---|---|---|
| V1 | `OutputLimitExceeded` is checked before `err` and returns only `totalBytes`, discarding buffers and exit status | Code read, `internal/effects/process.go` Step 5 (~lines 218–221) at `0384450a` |
| V2 | `Timeout` returns only `durationMs`; buffers in hand are dropped | Code read, Step 6 (~lines 225–227) |
| V3 | `truncated` is always `false`: every `makeProcessResultOk` call site passes `false`; the field's design intent was "true if output was capped" | `grep -n "truncated" internal/effects/process.go` (parameter + literal at 3 call sites); `design_docs/implemented/v0_8_1/m-process-exec.md` line 107 |
| V4 | `exec.ErrWaitDelay` falls through to `SpawnFailed` (not `ExitError`, not `ErrNotFound`, no "permission denied") | Code read, Step 6 fall-through (~line 257); reporter's repro table in #1879; judge CONCUR at `0384450a` |
| V5 | `PermissionDenied` is assigned by `strings.Contains(err.Error(), "permission denied")` on `cmd.Run()`'s merged start/wait error | Code read, Step 6 (~line 251) |
| V6 | `--process-max-output` help says "before kill" but `limitedWriter` discards without killing | `cmd/ailang/main_run.go:108`; `limitedWriter.Write` in `process.go` |
| V7 | stderr's budget is computed once at setup with `stdoutBuf.Len()==0`, so streams get independent caps (`pc.MaxOutput` each) | Code read, Step 3 (`pc.MaxOutput - int64(stdoutBuf.Len())` at ~line 205) — defect noted, routing deferred |
| V8 | **Pattern-arity mismatch passes `ailang check` and silently falls through at eval** — `Err(Timeout(ms, partial))` against `Timeout(int)`: check rc=0, runtime prints the *next* arm's result; `Err(AbnormalExit(x))` against `AbnormalExit(int,string)`: check rc=0, runtime falls to `Timeout(ms)` | Live runs on this workspace's binary (v0.54.15-era source tree): two transcripts in session record; reproduced twice (tmp files, `--caps IO,Process`) |
| V8a | An exhaustive 8-arm match on `ProcessError` compiles today | Live `ailang check` rc=0 ("✓ No errors found") |
| V9 | No existing design doc covers this fix (duplicate gate) | `create_planned_doc.sh` search: no matches; manual greps per the Searched header — all "1878/1879/1880" hits are SHA/line-number false positives |
| V10 | motoko at `e69aaad` maps every `ProcessError` to empty output / exit 1 in `run_process_result` | #1878/#1879 issue bodies; triage iteration 6 judge evidence (10 CONCUR / 0 DISSENT) — code-read, not run, from this sandbox |
| V11 | The Start/Wait split has in-repo precedent in the same package | `internal/effects/stream_process.go:77` (`cmd.Start()`) + `:169` (`cmd.Wait()`) |
| V12 | Existing timeout tests assert constructor *name* only, so they survive arity changes | Read: `process_test.go:261` (`ctorName != "Timeout"`), `process_timeout_group_test.go:37` (`ctor != "Timeout"`) |
| V13 | The three issues remain open and unfixed at v0.54.15 / `0384450a` | GitHub API `state: open` for #1878; triage iteration 6 verification + judge CONCUR (evidence link in header); defects re-read in source at that SHA in this session |
| V14 | `ProcessError` is a stdlib ADT defined in `std/process.ail` (single source of truth for the shape) and the builtin signature is generated in `internal/builtins/process.go:makeProcessExecType` | Read both files end-to-end |

## References

- **Issues consolidated**: [#1878](https://github.com/sunholo-data/ailang/issues/1878) · [#1879](https://github.com/sunholo-data/ailang/issues/1879) · [#1880](https://github.com/sunholo-data/ailang/issues/1880)
- **Evidence**: [triage iteration 6, issue #1835 comment](https://github.com/sunholo-data/ailang/issues/1835#issuecomment-6240929372)
- **Prior art in repo**: `design_docs/implemented/v0_8_1/m-process-exec.md` (original Process effect — its `truncated` design this doc completes), `design_docs/planned/v0_54_7/spawn-process-result.md` (sibling call), `design_docs/planned/v1_1_0/m-process-modes.md` (orthogonal replay modes), `design_docs/implemented/v0_54_4/m-background-worker-lifecycle.md`
- **Consumer on record**: motoko_agent `e69aaad7844c122d268100bc46aadc79ac9422d1`, `src/core/tool_runtime.ail` (`run_process_result`), `BashExec`
- **Axiom reference**: [Design Axioms](/docs/references/axioms)

## Future Work

- Pattern-arity checking in `match` (the V8 gap) — route as its own triage row; it turns silent ADT breaks into loud ones
- The V7 stderr-budget defect, if not pulled into this sprint
- `execText` convenience wrapper (existing TODO in `std/process.ail`)
