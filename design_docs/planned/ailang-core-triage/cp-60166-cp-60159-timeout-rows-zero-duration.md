# cp_60166/cp_60159 time out on every harness and bank duration_ms=0

- **Date**: 2026-11-03
- **Class**: bug (the instrument defect); the tiering question is answered below, not deferred to a doc
- **Recommend**: direct-fix — executed in-session under coordinator directive (task-c0b782ef)
- **Searched**: `duration_ms`, `CarryFailedRunSpend`, `park` in design_docs/; `60166|60159` in design_docs/ and benchmarks/; `long-frontier` tier definition in internal/eval_harness/spec.go
- **Estimate**: ~45 lines in `internal/executor/pi/pi.go` + ~50 in `internal/executor/opencode/opencode.go` + 2 regression-test files

Honesty note: this exceeds DIRECT_FIX_MAX_LINES/MAX_FILES (2 lines / 1 file), so the standing
rubric says `design-doc`. The coordinator directive ordered the fix explicitly ("fix timeout rows
so they bank the real elapsed duration"), which is the authority that waived the rubric; the fix
was mechanical populate-the-fields work, not a semantics decision.

## The instrument defect (root cause, fixed)

`CarryFailedRunSpend` (internal/eval_harness/idle_killed.go, added 2026-10-16) correctly copies a
failed run's wall time/tokens/cost onto its banked row — but it can only carry what the executor
put in the `Result`. The **pi** and **opencode** executors' timeout return paths (hard, prefill,
idle, ctx-cancel) built their `Result` without `DurationMS`, `NumTurns` or `ToolCallCount`:

- **pi** (`internal/executor/pi/pi.go`, the `hardTimer`/`ttftTimer`/`idleCheck`/`ctx.Done` cases):
  dropped `DurationMS` and the turn/tool tally → `CarryFailedRunSpend`'s all-zero guard rejected
  the row, so timeout rows banked duration_ms=0, 0 turns, $0 even when tokens HAD streamed.
- **opencode** (`internal/executor/opencode/opencode.go`, same four cases): dropped `DurationMS`
  AND `FinishReason` AND the tally. With `FinishReason` empty the row classified by error string,
  and with 0 turns reported `RunAgentBenchmarkWithExecutor` read the killed run as a 0-shot
  generation — which is why the rotation's opencode cp_60166 rows say "non_agentic 2 + api_error 2"
  for what were wall-clock timeouts.

motoko, claude and codex already bank real duration on their timeout paths; they were audited and
need no change. Fix: a `timeoutResult(errMsg, finishReason)` helper in each of pi and opencode
that carries every spend field in scope at kill time (duration, tokens incl. reasoning and cache
split, registry-rate cost, turns, tool histogram, session id, `FinishReason: timeout`), used by
all four kill paths. opencode's cancel path now distinguishes `DeadlineExceeded` (a timeout by
another name) from a caller abort, matching pi's `piCancelFinishReason`. Regression tests:
`internal/executor/pi/timeout_spend_test.go`, `internal/executor/opencode/timeout_spend_test.go`.
NOTE: this sandbox has no Go toolchain (repo needs go 1.26.10) — the fix is inspection-verified,
not compiled; CI must run `make test` on the two executor packages before merge.

## Question 1: mis-tiered or harness bug? — neither re-tier nor blame the model yet

- **The timeouts themselves are real, not a harness bug.** The wall-clock kill is by design and
  the problems are known-hard: `design_docs/planned/v0_55_0/m-int-overflow-trap-and-bigint.md`
  (line ~52) records that even **Python's** exact-bigint DP times out on 5/6 of cp_60166's big
  hidden tests; `m-eval-hidden-tests.md` (line ~72) records cp_60159's AILANG reference solution
  needs **8.5s** against a 10s AILANG validation limit. A 27B local model burning its 600s agent
  budget on these is the benchmark discriminating, and the "zero-pass = harness" heuristic does
  not survive that evidence.
- **But the zero-pass evidence was partly instrument corruption**, so a re-tier on these rows
  would decide from broken data: opencode's timeout rows were mislabelled non_agentic/api_error,
  and every timeout row hid its elapsed time and token spend. We do not know whether runs died at
  9:50 having nearly finished or thrashed from minute one — the fixed rows will say.
- **`long-frontier` is the wrong tier on its own definition.** spec.go defines it as benchmarks
  whose runs routinely exceed the **1h wall clock** (quine, gauntlet_10, commonmark_emphasis,
  legal_obligation_engine were re-tiered for exactly that on 2026-10-06). These die at the 600s
  bound, and cp_60149 — same family, same tier — took a PASS in this very rotation, so the family
  is solvable in-suite. A permanent attended-only stamp would also discard signal parking prices
  correctly.
- **The park-set mechanism (2b272cef, already shipped) is the right instrument**: zero-pass
  benchmarks run as one probe trial on one model per release, and the first passing probe
  un-parks automatically. That contains the GPU bleed without freezing a tier verdict.

**Recommendation**: keep tier=stretch; keep cp_60166/cp_60159 parked via `tools/eval-park-set.sh`;
after this fix lands, let one release bank fixed-instrument timeout rows (real duration_ms +
tokens + finish_reason=timeout) and re-examine then — if runs die deep in the bound with
thrashing token counts on every harness, the park set will keep holding them at probe cost
anyway, which is the outcome long-frontier re-tiering would have bought at the price of a
permanent tier change.

## What would make this actionable again

Nothing — the fix is in this task. The residual follow-up (not this row's job): the per-language
exec-time asymmetry (AILANG 10s vs Python 30s in agent_validation.go) documented in
m-eval-hidden-tests.md remains a separate, known marginality for cp_60159-class benchmarks.
