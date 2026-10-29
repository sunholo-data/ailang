# M-VM-QUICK-WINS: Cheap bytecode-VM speedups, measured one at a time

**Status**: Phases 0, 1 and 3 IMPLEMENTED (2026-10-28, sprint M-VM-QUICK-WINS); Phase 2 (W5–W7, compiler) planned, waiting for M-VM-COVERAGE Phase 1 (PR #1989). See the Implementation Report at the end.
**Target**: v0.55.0
**Priority**: P2. Performance follows correctness: M-VM-COVERAGE stays first.
**Estimated**: 3–4 days. Phase 0 and Phase 1 (~2 days) can run now. Phase 2 (~1.5 days) waits for M-VM-COVERAGE Phase 1.
**Dependencies**:
- Phase 0 and Phase 1 have none. They touch only `internal/vm`.
- Phase 2 waits for [M-VM-COVERAGE](m-vm-coverage-letrec-prelude-bridge.md) Phase 1. Both edit `internal/bytecode/compiler/call.go` (callee classification).

**Planner-Lane**: sonnet-capable. Each change is localized, measured and reversible.

## Routing Lane (per design_docs/PROGRAM.md §4)

**Lane: AILANG fix.** This is runtime performance of the AILANG toolchain. No extension can
change the VM dispatch loop, and nothing here touches the frozen motoko core.

## Quorum Triggers

None fire, so the quorum is skipped.
- **Trigger 1 (design-freeze items): no.** The only human decision, a default GC setting, is out of
  scope. Phase 3 only measures and recommends.
- **Trigger 2 (overrides shared machinery): no.** Every change is internal to the VM or to the
  bytecode compiler's output. The language semantics and the evaluator are untouched.
- **Trigger 3 (cost/KPI or banked-data schema): no.** It adds VM metrics to the perf-sweep snapshot
  as new keys. Existing keys keep their meaning.
- **Trigger 4 (external systems): no.** Every premise is in-repo and re-measurable. See the Verification Log.

## Axiom Compliance

**Canonical reference:** [Design Axioms](/docs/references/axioms)

| Axiom | Score | Justification |
|-------|-------|---------------|
| A1: Determinism | 0 | Same results, bit for bit. Every change is gated on unchanged output for both the parity corpus and the new VM workloads |
| A2: Replayability | 0 | Trace semantics unchanged |
| A3: Effect Legibility | 0 | No effect-dispatch change |
| A4: Explicit Authority | 0 | No capability change |
| A5: Bounded Verification | 0 | Type checking unchanged |
| A6: Safe Concurrency | 0 | Per-VM scratch state only; a VM is single-threaded (see V8) |
| A7: Machines First | +1 | Model-written AILANG runs faster without anyone rewriting it. The tail-recursive loops models write gain the most (−28% measured) |
| A8: Minimal Syntax | 0 | No syntax change |
| A9: Cost Visibility | +1 | For the first time, VM speed becomes a banked, diffed perf-sweep metric. Today every perf-sweep number is evaluator-only (V9) |
| A10: Composability | 0 | No module-system change |
| A11: Structured Failure | 0 | Error paths unchanged. Division-by-zero and type-mismatch errors keep their codes |
| A12: System Boundary | 0 | No boundary change |

**Net Score: +2** → **Decision: Move forward.** No −1 on A1/A3/A4/A7.

## Problem Statement

The bytecode VM is about 40× faster than the tree-walking evaluator, but about 4× slower than
CPython on call-heavy code. Measured 2026-10-28 on this machine (V1), best of 3, `--strict-bytecode`,
binary built from HEAD `c0458fdfd`:

| Workload | Evaluator | VM | CPython 3 | Hand-written Go |
|---|---|---|---|---|
| `fib(32)`, call-heavy | ~40 s (2.5 s at `fib(27)`) | **0.50 s** | 0.14 s | 0.005 s |
| `loop(0, 10_000_000, 0)`, tail-recursive `acc + (i*i) % 7` | — | **1.07 s** | 0.32 s | 0.005 s |
| `foldl(+, 0, map(*2, range(0, 2_000_000)))` | 0.89 s at 300k | **0.18 s** | 0.04 s | 0.002 s |

Profiling (`ailang run --strict-bytecode --cpuprofile/--memprofile`, V2) shows the VM doing work
that a tight loop should not do:

1. **Every call to a known top-level function allocates a closure.** The compiler emits
   `CLOSURE rN, P0` and then `CALL rN` for `fib(n - 1)` (V3). `OpClosure` always runs
   `bytecode.NewClosure`, even for a prototype with zero captures (`internal/vm/vm.go:421`). In
   `fib(32)` this is **94.9% of all bytes allocated (330 MB)**, and `NewClosure` is 20% of CPU.
2. **Every tail call allocates an argument slice.** `TAIL_CALL` runs
   `args := make([]bytecode.Value, argCount)` to stash arguments before `reuseFor` reuses the
   register slab (`vm.go:373`). In the 10M-step loop this is **1.19 GB of 1.7 GB allocated**.
3. **Integer `%` is a builtin call, not an opcode.** `a % b` on ints compiles to
   `BUILTIN_CALL builtin#568` (`_mod_Int`) plus two `MOVE`s, while `+ - * /` and comparisons compile
   to opcodes (V4). `OpMod` already exists in the compiler's operator table
   (`internal/bytecode/compiler/expr.go:336`) and in `arith()` (`vm.go:662`).
4. **Integer arithmetic goes through a function call.** `OpAdd/Sub/Mul/...` call
   `arith(op, lhs, rhs)` (`vm.go:198`, `643`), which re-switches on the op and on the tag. In the loop
   profile `arith` is 9.9% flat.
5. **Every tail call zeroes the whole register slab** (`frame.go:110–112`), then writes the
   arguments over the first `NumParams` slots.
6. **GC overhead.** `runtime.madvise`, `mmap` and `memclr` together are about 25% of `fib`'s
   samples. Most of this should disappear with items 1 and 2.

**A prototype of items 1 and 2** (about 20 lines, scratch build, not committed; V5) measured:

| Workload | Before | After | Change |
|---|---|---|---|
| `fib(32)` | 0.50 s | 0.40 s | **−20%** |
| tail loop, 10M | 1.07 s | 0.77 s | **−28%** |
| map + fold, 2M | 0.18 s | 0.18 s | 0% |

Outputs were identical. The prototype did not run the test suite. That is this sprint's job.

**Impact:** every `--bytecode` user, which includes stapledons-godot's per-tick simulation
(`bridge/sim_bridge.gd:445`). Once M-VM-COVERAGE makes the VM the default, it also covers every
`ailang run`. The eval harness would gain too if F2 there is accepted.

## Goals

**Primary Goal:** remove the per-call and per-iteration allocations and dispatch overhead the
profile names, each change measured on its own and kept only if it pays.

**Success Metrics** (best of 3, same machine, against the Phase 0 banked baseline):
- `fib(32)` under `--strict-bytecode`: 0.50 s → **≤ 0.35 s** (−30%)
- 10M-step tail loop: 1.07 s → **≤ 0.65 s** (−40%)
- `fib(32)` bytes allocated: 330 MB → **< 10 MB**. Tail loop: 1.7 GB → **< 50 MB**
- **No regression above 3%** on any existing perf-sweep metric, and zero parity regressions
  (see the gate in Phase 0)

Python parity is **not** a goal of this sprint. It is the follow-on question (see Future Work).

## High-Impact Decisions

| Decision | Why High Impact | Chosen By | Deadline | Change Cost |
|----------|-----------------|-----------|----------|-------------|
| Zero-capture closures are shared, one per prototype per VM | Changes allocation identity of closure values. Safe only while nothing observes closure identity or mutates a zero-capture closure (V6, V7) | agent, within the invariants below | design | low |
| Known-function calls: new `CALL_KNOWN` opcode vs a pre-built constant closure loaded once | An opcode changes the image format and disassembler. A constant needs no format change but still goes through the closure-call path | agent; measure both if cheap | compile | med |
| Keep or drop each win | A change that measures under 5% on every workload is complexity without payoff | agent, using the keep rule in Phase 1 | runtime | low |
| Default GC tuning (`GOGC` / memory limit) for `ailang run` | Trades RSS for speed process-wide. The rig has panicked on memory before (see `.claude/rules/local-models.md`) | **human**, after Phase 3's measurement | after this sprint | med |

### Design Freeze

No item blocks implementation. The GC default is explicitly *not* changed in this sprint: Phase 3
measures it and writes a recommendation for a human decision.

## Solution Design

### Overview

Four phases. Phase 0 builds the measuring stick and the safety gate. Phase 1 makes the runtime-only
fixes in `internal/vm`, which can run alongside M-VM-COVERAGE. Phase 2 makes the compiler-side
fixes after coverage Phase 1 lands. Phase 3 measures GC settings and recommends, without changing
anything.

**Keep rule, applied to every win W1–W7:** land it only if (a) the parity gate is unchanged, (b) it
improves at least one VM workload by ≥ 5%, and (c) it regresses no perf-sweep metric by more than 3%.
Record each win's before/after numbers in its commit message and in the implementation report.

### Phase 0: VM baseline and parity gate (~0.5 day)

- [ ] Add VM workloads to perf-sweep (`.claude/skills/perf-sweep/scripts/sweep.sh`). Today it has
      **no VM metric at all** (V9). Add `vm_fib32_s`, `vm_tailloop10m_s`, `vm_mapfold2m_s`,
      `vm_fib32_alloc_mb` and `vm_tailloop_alloc_mb`. Put the three programs in
      `benchmarks/workloads/vm/` with expected outputs (`2178309`, `19999999`, `3999998000000`).
- [ ] Bank a baseline snapshot under `.ailang/state/perf/`.
- [ ] **Parity gate.** No CI target runs the VM parity harness today (V10). Before W1 lands, record
      `go run ./scripts/verify_bytecode_parity.go --json` and the `--strict-bytecode` pass/fail per
      file for `examples/runnable`. After each win, re-run it and diff: any file that changes outcome
      blocks the win. If M-VM-COVERAGE's `make vm-coverage` has landed by then, use that instead
      (its Phase 3).
- [ ] Optional, not a gate: time one stapledons-godot sim tick through its existing perf target
      before and after. It is the real `--bytecode` user, so its number goes in the report.

### Phase 1: Runtime-only wins in `internal/vm` (~1.5 days; can run now)

- [ ] **W1: share zero-capture closures.** In `OpClosure`, when `innerProto.NumCaptures == 0`,
      return one closure per image prototype, created lazily and cached on the VM (for example
      `vm.zeroCap []bytecode.Value`, indexed by image prototype index). Invariants, each with a test:
      closure values are never compared by identity, and a zero-capture closure's `Captures` is never
      written (V6, V7). M-VM-COVERAGE G1 may patch self-captures for local `letrec`. That only
      touches closures *with* captures, so W1 must not share closures where `NumCaptures > 0`.
- [ ] **W2: tail-call argument scratch.** Replace `make([]bytecode.Value, argCount)` in `TAIL_CALL`
      with a VM-owned scratch slice grown on demand. It is safe because the arguments are copied
      into `frame.Regs` before anything can re-enter the VM (`vm.go:373–381`; V8). The same pattern
      applies to the evaluator-only tail path at `vm.go:349`. Measure it, but that path is cold.
- [ ] **W3: inline int fast path.** In the dispatch cases for `OpAdd/Sub/Mul` and `OpLt/Le/Eq`,
      handle `lhs.Tag == TagInt && rhs.Tag == TagInt` inline and fall through to `arith()` /
      the comparison helper otherwise. `Div` and `Mod` keep their divisor check via
      `ailerrors.CheckIntDivisor`, so the RT001 behaviour is unchanged (`vm_divzero_test.go`).
- [ ] **W4: cheaper tail-call frame reset.** In `reuseFor`, write the arguments first and zero only
      registers `[NumParams + len(captures), NumRegs)`, instead of zeroing everything and then
      overwriting. The compiler's register allocation must not read a non-parameter register before
      writing it. That is an invariant to test, not to assume (V11). If it doesn't hold, keep the
      full zeroing and drop W4.

### Phase 2: Compiler-side wins (~1.5 days; after M-VM-COVERAGE Phase 1)

- [ ] **W5: integer `%` as an opcode.** Find where `%` on `int` becomes the `_mod_Int` builtin call
      rather than `core.OpMod` (the lowering at `internal/gen/lower/expr.go:363` and `445` already
      maps both to `stmt.OpMod`). Route it to `OpMod`. Keep the RT001 modulo-by-zero error and its
      text. Fix it once at the source, not by a special case in the bytecode compiler, and check
      whether any other `_*_Int` builtin is reached the same way.
- [ ] **W6: direct calls to known functions.** For a callee `classifyCallee` resolves to a
      top-level function, skip the `CLOSURE` + `CALL` pair. Use either a `CALL_KNOWN protoIdx`
      opcode or a constant closure loaded once per frame (see High-Impact Decisions). This comes after
      coverage Phase 1 because that phase rewrites lexical-scope precedence in the same function:
      a locally bound name must never resolve to a top-level function (G1b).
- [ ] **W7: MOVE peephole.** Remove redundant chains such as `LOAD_CONST r9, #1` followed by
      `MOVE r8, r9` (V3/V4), where the target register is dead after the move. If W5 removes most
      instances in the benchmarks, measure what is left before building a general pass.

### Phase 3: GC measurement only (~0.5 day)

- [ ] Re-profile after Phase 2. If runtime GC (`madvise`, `mmap`, `memclr`, `mallocgc`) is still
      above 10% of VM samples, measure `GOGC=200/400` and `GOMEMLIMIT` against RSS on the VM
      workloads and the perf-sweep RSS metrics.
- [ ] Write the numbers and a recommendation into the implementation report and route it to a human.
      Do **not** change process defaults in this sprint.

### Files to Modify/Create

- `internal/vm/vm.go` — W1 (`OpClosure`), W2 (`TAIL_CALL`), W3 (arith dispatch cases), ~60 LOC. It is 786 lines now; if it passes about 1,000, move the arith fast paths to `vm_arith.go` (the file-size gate is 1,200).
- `internal/vm/frame.go` — W4 `reuseFor`, ~15 LOC
- `internal/vm/vm_quickwins_test.go` (new) — invariant tests: shared zero-capture closure, tail-call scratch reuse, int fast path versus `arith()` on every tag pair, division and modulo by zero still RT001, register-read-before-write for W4, ~200 LOC
- `internal/bytecode/compiler/call.go` and `internal/bytecode/opcode.go` — W6, ~60 LOC; `internal/bytecode/disasm.go` if a new opcode is added
- `internal/gen/lower/expr.go` or the elaboration site found in W5 — ~10–20 LOC
- `.claude/skills/perf-sweep/scripts/sweep.sh` — Phase 0 VM metrics, ~40 LOC
- `benchmarks/workloads/vm/fib.ail`, `tailloop.ail` and `mapfold.ail` (new) — Phase 0 workloads
- `changelogs/unreleased/<date>-vm-quick-wins.md` — changelog fragment
- `docs/LIMITATIONS.md` and `docs/docs/reference/limitations.md` — update the "VM ~4× slower than CPython" numbers

## Examples

### Example 1: `fib` call site

**Before** (`ailang disasm`, today):
```
0007  CLOSURE      r3, P0            ; bench/bench.fib   <- allocates every call
0008  MOVE         r4, r2
0009  CALL         r3, args=1, results=1
```

**After W1:** the same bytecode, but `CLOSURE` returns the cached zero-capture closure and
allocates nothing. **After W6:** `CALL_KNOWN P0, r2, args=1`, or the equivalent constant-closure
form, with no per-call closure load.

### Example 2: tail-recursive loop step

**Before:** `BUILTIN_CALL _mod_Int` + 2 `MOVE`s; `TAIL_CALL` allocates `args`; `reuseFor` zeroes
14 registers. **After:** `MOD r6, r5, rK`; the scratch buffer is reused; only the non-argument
registers are zeroed.

## Success Criteria

- [ ] Phase 0: VM workloads in perf-sweep with a banked baseline; parity baseline recorded
- [ ] `fib(32)` ≤ 0.35 s and 10M tail loop ≤ 0.65 s under `--strict-bytecode`, best of 3, same machine as the baseline
- [ ] `fib(32)` < 10 MB allocated; tail loop < 50 MB (`--memprofile`, `alloc_space`)
- [ ] Parity diff empty: no file in `examples/runnable` changes outcome under `--bytecode` or `--strict-bytecode`
- [ ] perf-sweep: no metric regresses by more than 3% (`sweep.sh` exit code 0)
- [ ] Every kept win carries before/after numbers in its commit; every dropped win is listed in the report with its numbers
- [ ] `make test`, `make lint`, `make verify-examples` pass; `make check-file-sizes` passes
- [ ] Changelog fragment; LIMITATIONS numbers updated
- [ ] Phase 3 GC recommendation written and routed to a human

## Testing Strategy

**Unit tests** (`internal/vm/vm_quickwins_test.go`):
- W1: two calls return closures sharing one `ClosureObj` when `NumCaptures == 0`, and distinct objects when `NumCaptures > 0`
- W2: a 1M-step tail loop allocates O(1) bytes (`testing.AllocsPerRun`). This is an allocation assertion, not RSS (see the memory-tests feedback note)
- W3: a table over every tag pair, comparing the fast path with `arith()`, including mismatched tags (error text unchanged), `minInt / -1`, and division and modulo by zero (RT001)
- W4: a prototype whose locals are read only after being written. Assert that zeroing only the non-argument registers leaves results identical on the existing VM test corpus

**Mutation check:** for W1 and W2, revert the fix in a scratchpad copy and confirm the allocation
assertion fails.

**Integration:** the Phase 0 parity diff after each win, plus `go test ./internal/vm/... ./internal/bytecode/...`.

## Deferred Decisions

- `CALL_KNOWN` opcode vs constant-closure load for W6 — agent may choose. Measure both if each is under half a day.
- Whether W4 is worth keeping — agent decides by the keep rule.
- Exact perf-sweep metric names and workload sizes — agent may choose, provided each runs in under 2 s on the VM.

## Non-Goals

- **Shrinking the 40-byte `bytecode.Value`.** This is the real fix for copy-bound list building
  (LIMITATIONS: `--bytecode` is ~1.8× slower than the evaluator on `::` building). It is a
  representation change across the VM and needs its own design (D-19 lineage).
- **NaN-boxing, JIT, or generating Go/Rust/TS.** These are the next performance tier. They are
  decided after this sprint's numbers exist.
- **VM coverage and parity fixes.** Those belong to M-VM-COVERAGE.
- **Changing the GC defaults.** Phase 3 only measures.
- **Evaluator performance.**

## Timeline

**Day 1:** Phase 0 (baseline, parity gate) and W1, W2.
**Day 2:** W3, W4, measured and kept or dropped.
**Day 3** (after M-VM-COVERAGE Phase 1): W5, W6.
**Day 4:** W7 if warranted, Phase 3 measurement, report.

**Total: ~3–4 days, split around M-VM-COVERAGE Phase 1.**

## Risks & Mitigations

| Risk | Impact | Mitigation |
|------|--------|-----------|
| A shared zero-capture closure is mutated or compared by identity somewhere | High (wrong results) | V6/V7 grep found no such site. W1's tests enforce it, and any future capture-patching code is limited to `NumCaptures > 0` |
| W2's scratch slice is aliased by a re-entrant call | High | The arguments are copied into `frame.Regs` before any call can re-enter. A test runs a tail call whose arguments come from a HOF callback |
| W3 changes overflow or error behaviour | Med | A table test against `arith()` over every tag pair and edge value |
| W6 collides with M-VM-COVERAGE's `classifyCallee` rewrite | Med | Phase 2 is sequenced after coverage Phase 1 |
| Benchmark noise hides or fakes a win | Med | Best of 3, same machine, banked baseline; the keep rule's 5% threshold sits well above run-to-run noise. Re-check on a quiet machine before landing |
| The rig is busy (concurrent evals) during measurement | Med | Check `.claude/rules/local-models.md` rig state first; note load in the report |

## Verification Log

| # | Claim | How verified | Result |
|---|---|---|---|
| V1 | Baseline timings in the Problem Statement | `/usr/bin/time -p` best of 3, `ailang-base` built from HEAD `c0458fdfd`, `--strict-bytecode --entry …`; Python and Go timed in-process (2026-10-28) | Confirmed; outputs `2178309` / `19999999` / `3999998000000` on every engine |
| V2 | Allocation and CPU attribution | `--cpuprofile` / `--memprofile` + `go tool pprof -top`, `-sample_index=alloc_space`, `-list 'vm.\(\*VM\).run$'` | fib: `NewClosure` 330 MB = 94.9%; loop: `vm.go:373` 1.19 GB of 1.7 GB |
| V3 | Known-function calls compile to `CLOSURE` + `CALL` | `ailang disasm bench/bench.ail`, prototype `bench/bench.fib`, insns 0007–0009 and 0012–0014 | Confirmed |
| V4 | Only integer `%` compiles to a builtin call | `ailang disasm` on `f(a,b) = a+b-a*b+a/b+a%b` (ints), the float equivalent, and `<`/`==` | `BUILTIN_CALL builtin#568` only for int `%`; all others are opcodes |
| V5 | W1 + W2 prototype gains | Patched `vm.go` copy, built to a scratch binary, restored the original (`git status` clean), best of 3 against the same-HEAD baseline | fib −20%, loop −28%, map/fold 0%; outputs identical. Test suite NOT run |
| V6 | Nothing compares closures by identity | `runtimeEq` (`vm.go:692`) falls back to `Value.Equal` for closures; `Value.Equal` (`internal/bytecode/value.go:469`) compares `Proto` and then `Captures` structurally, never `Obj` pointers. The type checker rejects `Eq` on functions anyway (comment at `vm.go:741`) | Confirmed: equality is structural, so a shared zero-capture closure compares exactly as two fresh ones did. Re-check `show` during implementation |
| V7 | Nothing mutates a closure's `Captures` or `Proto` after creation | `grep -rnE '\.Captures\[[^]]*\]\s*='` and closure `.Proto =` assignments over `internal/` | None found |
| V8 | `TAIL_CALL` copies args before any re-entry | Read `vm.go:330–381`: args stashed, `reuseFor`, `copy(frame.Regs, args)`, captures written. No call in between | Confirmed |
| V9 | perf-sweep has no VM metric | `jq '.metrics \| keys' .ailang/state/perf/2026-09-17.json`; `grep -n bytecode .claude/skills/perf-sweep/scripts/sweep.sh` | 22 metrics, all evaluator/builtin/latency/RSS; none VM |
| V10 | No CI target runs VM parity | `grep -rnE 'verify_bytecode_parity\|strict-bytecode' Makefile make/*.mk .github/workflows/*.yml` | Empty. M-VM-COVERAGE Phase 3 plans `make vm-coverage` |
| V11 | `reuseFor` zeroes the whole slab on tail calls | Read `frame.go:100–114` | Confirmed. The read-before-write invariant W4 needs is UNVERIFIED and is W4's first task |
| V12 | No existing doc covers this | Duplicate gate: neural top 0.29 (planned) / 0.28 (implemented). Related prior work: M-PERF6 runtime hotspots (evaluator), #1501 Phase 3 (frame recycling, 40-byte Value), M-FOLDL-CONS-COST-MODEL | Distinct: none target closure/tail-call allocation or int-op dispatch in the VM |
| V13 | `vm.go` size headroom | `wc -l internal/vm/vm.go` | 786 lines |

## Related Documents

- [M-VM-COVERAGE](m-vm-coverage-letrec-prelude-bridge.md): correctness and coverage first; Phase 2 here waits on its Phase 1
- [M-FOLDL-CONS-COST-MODEL](../../implemented/v0_52_0/m-foldl-cons-cost-model.md): why list building is VM-slower (the 40-byte Value)
- [M-PERF6 runtime hotspots](../../implemented/v0_12_0/m-perf6-runtime-hotspots.md): the evaluator-side precedent
- [M-BYTECODE-VM](../../implemented/v0_11_0/m-bytecode-vm.md): VM architecture
- [M-PERF4](../v1_1_0/m-perf4-bytecode-interpreter.md): superseded; kept for history

## References

- [Design Axioms](/docs/references/axioms)
- `docs/LIMITATIONS.md`: "`--bytecode` silently falls back; VM ~4× slower than CPython"
- `.claude/skills/perf-sweep/SKILL.md`: banking and diffing perf snapshots

## Future Work

- **Python parity on call-heavy code.** Once these wins land, decide between a narrower `Value`,
  NaN-boxing, and superinstructions, using the new perf-sweep VM metrics.
- **A native tier.** Typed Go generated from the Statement IR (not the boxed `--emit-go`, which has
  no tail calls), for hot pure modules such as a game simulation. Decide only if the VM, after
  tuning, is still the bottleneck for a real user.

## Implementation Report: Phases 0, 1 and 3 (2026-10-28)

Commits on `dev`: `7b3170880` (M1), `9546c255b` (M2), `bc2816371` (vm.go split), `d042d947b` (M3), `b95dd32ab` (M4).

**End to end** (`ailang run --strict-bytecode`, best of 3, same machine, binary from the
morning baseline vs HEAD, load about 7 on 16 cores):

| Workload | Before | After | Target | CPython |
|---|---|---|---|---|
| `fib(32)` | 0.51 s | **0.33 s** (−35%) | ≤ 0.35 s ✓ | 0.14 s |
| 10M tail loop | 1.10 s | **0.55 s** (−50%) | ≤ 0.65 s ✓ | 0.32 s |
| map + fold 2M | 0.18 s | 0.16 s | no regression ✓ | 0.04 s |

**VM allocations** (`go test -bench BenchmarkVM`): `fib(25)` went from 11.66 MB / 242,847 allocs
to 12.7 KB / 65 allocs, and the 1M tail loop from 176 MB / 2,000,014 allocs to 3.8 KB / 8 allocs.
The `fib(32)` process total under `--memprofile` is 10.05 MB, all of it compile-time (regexp,
pipeline), so the "< 10 MB" criterion is met for the VM, not for the whole process. The tail
loop's process total is 16.3 MB.

**Per win** (each measured on its own; keep rule ≥ 5%):

| Win | Result | Kept? |
|---|---|---|
| W1 shared zero-capture closures + W2 tail-call scratch | fib25 −27%, tail loop −37% | yes |
| W3 inline int fast path (inline: a helper cost 95 against Go's inline budget of 80) | fib25 −16%, tail loop −26%, map+fold −13% | yes |
| W4 tail-call frame reset | tail loop −5.1% mean in an interleaved A/B (control fib −1%) | yes, borderline. It also deletes W2's scratch |

**Deviation (W4).** The plan was to zero only the non-argument registers, which depends on an
unproven read-before-write invariant. Instead, `Frame.resetForTailCall` moves the arguments to
the front in place and zeroes the rest. Its end state is identical to the old sequence, so no
invariant is needed. A randomized equivalence test over 5000 slabs proves it.

**Gates.** The parity gate (`tools/vm_parity_diff.sh`, 442 outcomes under `--bytecode` and
`--strict-bytecode`) was unchanged after every win. `sweep.sh --against` the baseline exited 0
with no regressions. Mutation checks: reverting W1 fails 3 tests; reverting W2 fails 1; SUB→ADD
and LT→LE in W3 each fail; dropping the zeroing or reversing the arguments in W4 fails.

**Phase 3 (GC).** After Phase 1, `madvise`/`mmap` no longer appear in VM profiles. The
remaining `memclr` (about 14% of `fib`) is the VM's own frame clearing (`releaseFrame` clears
each frame's whole capacity on return), not GC. **No GC setting is recommended.** The next cheap
candidate is clearing only the registers a frame actually used.

**Not done (Phase 2):** W5 (`%` as an opcode; `builtinModInt` is 6% of tail-loop CPU), W6
(direct known calls), and W7 (MOVE peephole). They wait for PR #1989.

---

**Document created**: 2026-10-28
**Last updated**: 2026-10-28
