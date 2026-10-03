# Sprint Plan: M-EVAL-TAIL-CALLS

**Design doc**: [m-eval-tail-calls.md](m-eval-tail-calls.md) (ratified 2026-10-02) · **Issue**: #1486
**Duration**: 3 days · **Risk**: high (core evaluator call path) · **Registry reuse**: none (evaluator internals)

## Milestones

### ✅ M1 — Tail-position plumbing, no behaviour change (~150 LOC)
- `tail bool` through `evalCoreIf`, `evalCoreLet`, `evalCoreLetRec`, `evalCoreMatch`, decision-tree arm bodies,
  `evalDictAbs`; `evalTail(expr, tail)` dispatch; unexported `tailCall` value type.
- Nothing produces a `tailCall` yet (every caller passes `tail=false`).
- **AC**: `go test ./internal/eval/... ./internal/runtime/... ./internal/testing/...` green and unchanged;
  `make test-core` green.

### ✅ M2 — Trampoline with per-frame contracts (~300 LOC)
- `internal/eval/eval_apply.go`: `applyFunctionValue(fn, args, name, firstFrameContract)`; `replaceable(fn)` (D4);
  first-frame contract from the caller, tail frames follow the `evalCoreApp` contract (D3); depth unchanged on a
  tail call (D5); env/resolver base captured and restored on every path; resolver wraps the current chain with
  `resolverCovers` dedupe; budget boundary per iteration, non-no-op restores retained; trace enter per frame,
  exits replayed innermost-first with today's per-frame rules (D6).
- `evalCoreApp` (`*FunctionValue` case) and `CallFunction` delegate.
- **AC** (`internal/eval/tail_call_test.go`): the four shapes at n = 10^6, default depth; non-tail recursion still
  RT_REC_003; `ensures` checked on every call (inner violation reported); budget and rand-mode frames unchanged;
  `requires` failure at iteration k gives exits for 1..k-1 only; body error inside a chain gives all exits; trace
  sequence identical (names/args/results/depths/spans) on a small loop; `CallFunction` frame contract unchanged,
  and helpers it tail-calls are traced; `*tailCall` never escapes a public entry; three-module resolver chain gives
  the same lookups, bounded chain. Mutation check: disable `replaceable` → contract/budget tests fail.

### ✅ M3 — End-to-end parity, example, docs (~150 LOC)
- `cmd/ailang/tail_call_parity_test.go`: the four shapes, interpreter vs `--strict-bytecode`; #1486 stdin repro
  with 20,000 lines and no flag.
- `examples/runnable/tail_recursion_loop.ail` + manifest entry; changelog fragment; RT_REC_003 hint mentions tail
  position.
- **AC**: parity test green; `make verify-examples`, `make lint`, `make check-file-sizes` green; full `make test`
  (load flakes re-run in isolation).

## Order
M1 → M2 → M3, sequential (all in internal/eval).

## Outcome (2026-10-02)

- [x] M1 `37ba2153a`: tail position threaded through If/Let/LetRec/Match/DTree/DictAbs via evalCoreT (stack-hop safe); no behaviour change
- [x] M2 `52b9c7297`: applyFunctionValue trampoline; 12 eval tests + cross-module runtime test; 4 mutations each caught; trace goldens captured BEFORE the change match after
- [x] M3 `3c77de8e4`: CLI parity vs --strict-bytecode at 200k; #1486 stdin loop unflagged; non-tail still RT_REC_003; example; RT_REC_003 hint
- Measured: cons.ail tail form 364 MB → 54 MB (untraced), 405 MB → 74 MB (deep); shapes.ail 10^6 iterations peak 105 MB
- Tests updated because their premise changed: TestStackOverflow (infinite tail loop now runs), TestRTREC003DoesNotAdvertiseTailRecursion → AdvisesTailCalls, memprobe cons.ail made non-tail, max_recursion_depth fixture made non-tail
- Not built: interpreter vs VM depth semantics differ by one frame for a CallFunction entry that tail-calls (entry frame uncounted, as before)
