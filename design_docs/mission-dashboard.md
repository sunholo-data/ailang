# Mission Dashboard — the 30-second control context
> **Contract**: ≤40 lines, overwritten by mission-control Gate 4 every iteration (history lives in
> the charter/log). Fresh session = THIS + MEMORY.md. Humans steer via the bookkeeping issue.

**Updated**: 2026-08-12 ~23:30 local (iteration 187)

## Now
- **v0.33.0** · merge `0625059d3` — **`#505` FIXED AND CLOSED** (PR #684). Gate 3b GREEN:
  `checks=22`, `pending=0`, zero NOT-GREEN, 4/4 REQUIRED from real `pull_request` events.
  Evaluator **sonnet PASS 110/120 r1, zero blocking**. `metered=$0.07` (two quorum rounds).
- 🟢 Fixed-length list patterns now match **exactly n** under `--bytecode`. `recursion_quicksort`
  printed `[3]` at rc=0 with no error and no fallback; it prints the right list now.

## 🔴 The bug was invisible because nobody wrote the queue row
Mark's option-C ruling (2026-08-04) split the parent doc and spun this P0 out as ready-to-sprint.
The charter mentioned `pattern-arity` **0** times (controls: `bytecode` 15, `#505` 2), so for 8
days iterations declared the still-PARKED *parent* as Next while an unblocked P0 sat unrowed.
A landed human decision does not create its own queue row → **`D-12`**.

## Also worth knowing
- **Quorum blocked it twice and was right both times.** Both rounds were *premise* objections;
  rule 3f says measure, not forward. Doing so shrank the fix site from three packages to one
  function + one operator, and round 2 caught that my own repro's arm ordering masked n=2/n=3.
- **The pin is a gate**: restoring `OpGte` LANDS, BUILDS, reds both new tests — and the inverse
  arm (`-skip` them) is rc=0 across all of `./cmd/ailang`. Nothing else caught `#505`.
- **Planner refuted two of my VERIFIED-BY-ME facts**, both confirmed first-party: `go build ./...`
  is red at base (`cmd/wasm`), and `tests/golden/bytecode/` is a Go test *package*, so the doc's
  prescribed AC1/AC3 home was unimplementable.
- **`#683`** filed: `make fmt-check-ail` enumerates `stdlib/`, which has never existed — 400 files
  swept vs 446, so 46 stdlib files sit outside a gate that prints a green checkmark.

## Queued
**Next: `#616`** (`D-10`). Then **`#619`** `D-9` · **`#618`** `D-8` · #636 · #613 · #604/#614 ·
#649 · #651 · #654 · #669 · #670 · #680 · #683. Parent `m-bytecode-vm-parity-bugs` stays PARKED on A2.

## Loop + routing
Controller **opus** · designer **codex `gpt-5.6-sol`** (rotation) · planner **opus** (derived lane,
`fail-closed:planner-lane-field-missing`) · executor **codex `gpt-5.6-sol`** · evaluator **sonnet**;
generator≠judge held. ✅ **Skill divergence CLOSED** — main checkout was 7 behind for 3 iterations;
ff'd under Mark's ratified authorization (preconditions measured), running skill == origin.

## PARKED ON MARK — #635
**`D-12`** unrowed-P0 gap (NEW) · **`D-11`** slot-death guard · **`D-10`** `#616` · **`D-9`** `#619` ·
**`D-1`** `#613` · **`D-2`** `#604`/`#614` · **`D-7`** codex · **`D-8`** `#618`.
