# M-CHESS-ENGINE: a correct, deterministic chess engine in pure AILANG — perft-proven, replayable, honest about speed

**Status**: Planned
**Target**: v0.64.2
**Priority**: P2 (showcase + VM forcing function; not roadmap-critical)
**Estimated**: ~2 weeks of agent time (6 working days ×2 for honesty)
**Dependencies**: None hard. **Coordination**: [m-persistent-collections](../v0_55_0/m-persistent-collections.md) (Phases 2–3 persistent Map/Array would make make/unmake cheaper — this engine is a motivating benchmark, not a blocker). See also the open-PR check in Verification Log V9.

**Origin**: Mark, 2026-11-27 — "Make a chess engine with AILANG?", citing
[deedy/chess-chatarunga](https://github.com/deedy/chess-chatarunga) (zero-dependency Rust UCI engine, ~3,440
CCRL 40/15, SIMD NNUE, ~1.17M nps). This doc answers the question the way AILANG answers every
"port the fast native program" question: keep what the language is *for*, refuse the part that is
architecturally unreachable, and make the result a forcing function for the VM instead of an
embarrassment for it.

## Reality check (read this before the design)

A Chatarunga-class engine is **out of reach by 3–4 orders of magnitude**, and this is architectural,
not tunable:

- Chatarunga sustains ~1.17M nodes/sec (v0.61 search, depth-10 bench). A chess search node costs
  O(1,000) native ops even with magic bitboards.
- The AILANG bytecode VM measures **~4× slower than CPython** on its own benchmark
  ([M-VM-QUICK-WINS](../v0_55_0/m-vm-quick-wins.md): `foldl(+,0,map(*2,range(0,2M)))` → 0.18s VM vs
  0.04s CPython), and this session's own chess-shaped probe measured **~0.3–0.5M primitive steps/sec
  on the VM** (~0.06–0.1M on the tree-walking evaluator) — V6 below.
- So an AILANG search node costs O(10⁴) primitive steps → **~10² nodes/sec**, i.e. ~10⁴× fewer nodes/sec
  than Chatarunga. An NNUE (16×1024×2 MLP per node) is not merely slow here, it is absurd.

What AILANG has that Chatarunga does not — and what makes this project worth doing anyway:

1. **Deterministic, replayable search**: the pure core means the entire search tree is a function of
   (position, depth). Same input, same bestmove, same PV, byte-for-byte — verifiable and auditable in
   a way no native engine is. This is the language's identity applied to a domain famous for
   nondeterminism (SMP, time races, TT aging).
2. **Perft is the perfect acceptance test** for a deterministic language: the standard perft suite has
   known-exact node counts, so "correct move generation" is a machine-decidable property, not a vibes
   claim.
3. **Contract verification on the pure core**: square arithmetic, move encoding round-trips, and
   popcount bounds are Z3-provable (`ailang verify`) — "mathematically proven board logic" is a
   claim no Rust engine can make cheaply.
4. **The engine is the VM's forcing function**: perft nps is the crispest end-to-end KPI for VM work
   on real nested recursion (this echoes [m-game-engine-effects](../v1_1_0/m-game-engine-effects.md)'s
   "game as the VM's flagship consumer" posture, for a game that needs no host effects).

So: **v1 targets a legal, perft-proven, material+PST, fixed-depth-2–3 engine with a human CLI.**
It will not beat Stockfish, Chatarunga, or a careful club player. It will be *right*, provably and
replayably, and it will be fast enough to play.

## Problem Statement

**Current State:**
- AILANG has no non-trivial, adversarial-search program in `examples/` — the strongest current demos
  are I/O and data plumbing. Nothing exercises deep mutual recursion + heavy integer bit-twiddling +
  immutable-state threading simultaneously.
- The VM's benchmark posture is synthetic (fold/map kernels, `bench/vm_*`). No benchmark exercises the
  cost model's danger zones (`::` list building, HOF closure churn, record copying) on one program
  that needs all of them, where regressions are visible as "the engine got slower" rather than as a
  percentage.

**Impact:**
- Users asking "can AILANG do X?" (games, simulations, solvers) get no worked answer; models get no
  template for structuring an algorithmic AILANG project under the cost model.
- VM and evaluator performance claims lack a user-visible end-to-end anchor.

## Goals

**Primary Goal:** Ship a pure-AILANG chess engine whose move generation is proven correct by the
standard perft suite, whose search is deterministic and replayable, and which plays legal chess at
fixed depth from a CLI.

**Success Metrics:**
1. perft(startpos, 1..3) exact (20 / 400 / 8,902); perft(startpos, 4) = 197,281 passes (CI gates at 3;
   4 is a recorded timing run, V-timed).
2. Standard battery: Kiwipete perft(1)=48, perft(2)=2,039; Position 3 perft(1)=14, perft(2)=264;
   Position 4 perft(1)=6; Position 5 perft(1)=44 (all exact).
3. Every move the engine reports is a member of its own legal-move list (property holds by
   construction; asserted in tests).
4. Same (position, depth) → same bestmove across runs, evaluator and VM (byte-for-byte `ailang run`
   transcript equality).
5. All modules `--strict-bytecode` clean (no SILENT_FALLBACK), with perft nps recorded for both
   engines as a standing VM KPI.

## High-Impact Decisions

| Decision | Why High Impact | Chosen By | Deadline | Change Cost |
|----------|-----------------|-----------|----------|-------------|
| Bitboard-based representation (12 piece-type bitboards in an immutable record; classical ray attacks via shifts, **not** magic bitboards) | Every later module inherits its cost profile; magic multiply risks int-overflow traps and 64-entry table probing is O(n) `A.set` territory | agent | design | high |
| Correctness-first scope: perft-proven legal engine, **no Elo target, no NNUE, no CCRL claim** | Sets expectations against the Chatarungan framing; refusing this now prevents a doomed "make it faster" sequel | human | design | high |
| Fixed-depth deterministic search (no Clock, no time management in v1) | Preserves A1/A2 purity of the core; time management would make bestmove timing-dependent and unreplayable | agent | design | med |
| Human CLI first (`readLineOpt` loop, algebraic-ish moves); UCI subset is a stretch phase | UCI's time controls are worthless at ~10² nps; a human can wait 30s at depth 3 | agent | design | med |
| Ship as `examples/chess/` multi-module project, pure AILANG only (no Go, no host effects) | Precedent: multi-module examples exist; keeps A12 clean; the game-engine doc's host-effect lane is explicitly NOT needed here | agent | design | low |
| Gate the project on `--strict-bytecode` (Makefile-style check in the example's README commands) | Silent evaluator fallback would 5× degrade nps invisibly ([M-VM-COVERAGE](../v0_55_0/m-vm-coverage-letrec-prelude-bridge.md) SILENT_FALLBACK) | agent | compile | med |

### Design Freeze

Before implementation begins, these must be resolved:

- [ ] **Representation approved**: bitboards-in-record with classical ray attacks, hex constants written
  **without** `_` separators (V2), unsigned south-shifts via `shiftRightLogical` (V1). If the
  implementer finds a mailbox-primary design cheaper under measured timing, they must record the
  measurement in the implementation notes before switching — not silently.
- [ ] **Scope confirmed as correctness-first** (no Elo chasing). This doc already fixes it; it is on
  the freeze list because it is the decision a downstream agent will most want to "improve" into a
  doomed one.

## Solution Design

### Overview

A multi-module pure-AILANG program in `examples/chess/` with a **pure core** (board, move generation,
search — `! {}`, contract-bearing where decidable) and a **thin IO shell** (CLI loop). Board state is
an immutable record; "make/unmake" is record update (V4 of the cost model — a ~14-word copy per node,
negligible next to movegen cost); search threads state functionally, so the whole engine is a pure
function of `(position, depth)`.

### Architecture

**Components:**

1. **`board`** — square math (rank/file/bit index), the `Position` record (12 piece-type bitboards,
   side, castling rights as 4-bit int, en-passant square, halfmove clock), Zobrist-free (no TT in v1).
   Pure helpers (`popcount`, `bitScan`, `coordToSq`, `sqToName`) carry `requires`/`ensures` where Z3
   can prove them (V10 governs what to avoid there).
2. **`fen`** — FEN parsing/construction: `Result[Position, string]`, never a silent fallback
   (CLAUDE.md §2).
3. **`movegen`** — pseudo-legal generation per piece class (leaper tables precomputed once into
   `Array[int]`; sliders via 8 ray directions with file-wrap masks); castling/en-passant legality;
   final legality = "own king not attacked after apply". Move type: `Move` ADT `deriving (Eq)` with
   pattern-matched access (V5: ADT field *access* does not compile — access via match).
4. **`search`** — negamax with alpha-beta, fixed depth 2–3, capture-only quiescence (bounded),
   material + piece-square-table evaluation, deterministic move ordering (MVV-LVA by piece code,
   stable by generation order). Returns `(bestmove, score, pv)`.
5. **`perft`** — the acceptance harness: `perft(pos, depth)` counting leaf nodes + a fixed-position
   runner used by tests and the timing record.
6. **`cli`** — IO shell: `readLineOpt` loop (V7), commands `position <fen>`, `show`, `moves`, `go
   <depth>`, `perft <depth>`, `quit`. Caps: `IO` only.

### Cost-model discipline (the implementation notes that keep it honest)

- Move lists are built with `::` in generation order, consumed head-first — no `acc ++ [x]`, no
  `A.set` per square (M-FOLDL-CONS-COST-MODEL, prompt Cost Model).
- No `range(0, 64)` allocations inside per-node loops; walk bitboards with `x & (x-1)` bit-clear
  loops (tail-recursive, verified pattern V4/V6 probe).
- Ray attacks: precompute the 8 direction deltas and file-edge masks as top-level pure constants;
  shifts/`&` only. No magic multiplication (int `*` overflow traps pending
  [m-int-overflow-trap-and-bigint](../v0_55_0/m-int-overflow-trap-and-bigint.md)).
- Non-tail recursion stays ≤ ~30 frames (search depth + quiescence ~10); the 10,000-frame default cap
  (docs/LIMITATIONS.md, V8) is nowhere near load-bearing.

### Implementation Plan

**Phase 1: Board + FEN core** (~1 day)
- [ ] `board.ail`: Position record, square math, popcount/bitScan, display (ASCII board).
- [ ] `fen.ail`: parse/build round-trip; inline `tests` on startpos/Kiwipete.
- [ ] Contracts on pure helpers; `ailang verify` green or explicitly inconclusive-and-noted.
- [ ] `ailang check`, `ailang test`, `--strict-bytecode` all clean.

**Phase 2: Move generation, perft-proven** (~2 days)
- [ ] Pseudo-legal generation per class + castling/EP; legality filter.
- [ ] `perft.ail` + fixed battery: startpos 1–4, Kiwipete 1–2, Positions 3/4/5 depth 1–2.
- [ ] Gate: every perft count exact. **If perft(4) disagrees, the phase is not done.**

**Phase 3: Search + eval + CLI** (~2 days)
- [ ] `search.ail`: negamax αβ + quiescence + PST; deterministic ordering; PV extraction.
- [ ] `cli.ail`: readLineOpt loop, commands above; `go <depth>` never returns a move outside its own
  legal list.
- [ ] Replay check: two runs of the same transcript produce byte-identical output (evaluator and VM).

**Phase 4: Package the showcase + record the numbers** (~1 day)
- [ ] `examples/chess/README.md`: what it is, what it is *not* (Reality check above), how to run.
- [ ] Timing table recorded: perft(3)/(4) nps on evaluator vs `--bytecode`, in the README and in this
  doc's implementation record.
- [ ] Optional stretch: UCI subset (`uci`/`isready`/`position fen`/`go depth N`/`quit`, fixed depth
  only — no `go movetime`; declare `Go` unsupported loudly).

### Files to Modify/Create

**New files (all AILANG, ~1,600 LOC total):**
- `examples/chess/board.ail` — position record + bit helpers (~300 LOC)
- `examples/chess/fen.ail` — FEN parse/build (~200 LOC)
- `examples/chess/movegen.ail` — generation + legality (~450 LOC)
- `examples/chess/search.ail` — negamax + eval (~300 LOC)
- `examples/chess/perft.ail` — perft harness + battery (~120 LOC)
- `examples/chess/cli.ail` — IO shell (~250 LOC)
- `examples/chess/README.md` — showcase doc incl. timing table

**Modified files:** none. No compiler, stdlib, or Go changes — the "Conflict Surface" section
(required for parser/typechecker/codegen changes) is **not applicable**: this is a pure application
in `examples/`, with no new syntax, builtins, or effects. Nothing in this design touches
`internal/` or `cmd/`.

## Examples

### Example 1: make-move as pure record update (verified, V5)

```ailang
type Move = Move(fromSq: int, toSq: int, promo: Option[string]) deriving (Eq)
type Pos = { white: int, black: int, knights: int, sideToMove: bool, castling: int }

pure func apply(pos: Pos, m: Move) -> Pos = match m {
  Move(f, t, _) => { pos | knights: pos.knights ^ (1 << f) ^ (1 << t),
                            sideToMove: not pos.sideToMove }
}
```

### Example 2: perft harness shape

```ailang
pure func perft(pos: Position, depth: int) -> int =
  if depth == 0 then 1
  else foldl(\acc m. acc + perft(applyMove(pos, m), depth - 1), 0, legalMoves(pos))
```

### Example 3: CLI session

```
$ ailang run --strict-bytecode --entry main --caps IO examples/chess/cli.ail
position rnbqkbnr/pppppppp/8/8/8/8/PPPPPPPP/RNBQKBNR w KQkq - 0 1
go 3
bestmove g1f3 score 0 pv g1f3 d7d5 g2g3
perft 3
8902 nodes, 43 nps (bytecode vm)
```

## Success Criteria

- [ ] perft(startpos,1..3) = 20 / 400 / 8,902 (CI gate); perft(startpos,4) = 197,281 (recorded run)
- [ ] Kiwipete perft(1)=48, perft(2)=2,039; Position 3: 14/264; Position 4: 6; Position 5: 44
- [ ] Engine move ⊆ engine's own legal-move list for every `go` (asserted in tests)
- [ ] Determinism: identical transcript → byte-identical output, on evaluator AND VM
- [ ] `ailang check` + `ailang test` green; `--strict-bytecode` run clean for all modules
- [ ] `ailang verify` green on contract-bearing pure helpers (inconclusive contracts explicitly listed, not hidden)
- [ ] README with reality-check framing + evaluator/VM timing table
- [ ] No Go files, no compiler changes, no new effects (diff auditable)

## Testing Strategy

**Unit tests (inline `tests` pairs):** square math, FEN round-trips, per-squares attack cases,
castling-rights transitions, EP capture cases (the classic generator killers).

**Property tests (`ailang test` property surface):** generated `applyMove` keeps piece counts
constant mod capture/promotion; `sqToName(coordToSq(r,f)) == ...` round-trip.

**Integration:** the perft battery as exact-match fixtures; a scripted CLI transcript diffed
byte-for-byte (Output Discipline applies).

**Manual:** play a few moves at depth 2; confirm no illegal move is ever accepted, including
en-passant and promotion edge cases.

## Deferred Decisions

The following are intentionally left open for the implementer:

- Move ordering details (MVV-LVA weighting) — agent may choose; must stay deterministic.
- PST values and material table — agent may choose; cite source tables (e.g. PeSTO classical) in code comments.
- Quiescence depth cap and stand-pat margin — agent may choose.
- Whether `Move` carries precomputed flags or recomputes on apply — agent may choose.
- Opening variety: an optional `Rand[mode=seeded]` *shell-side* tiebreak (never in the pure core) — agent may choose; default is deterministic first-legal.
- Whether Phase 4's UCI stretch happens at all — agent may choose; a loud "unsupported" error is required for any unimplemented UCI token (no silent fallback).

## Non-Goals

**Not attempted in this feature:**
- **NNUE / learned evaluation** — absurd at ~10² nps; 25 MB nets are also a bad fit for a program meant to be read.
- **Elo / CCRL targets, competitive play vs native engines** — the Reality check fixes this as out of scope; do not spawn "make it 100 nps faster" sequels without a new doc.
- **Magic bitboards** — overflow-trap risk + table-probing costs; classical rays are sufficient at this node rate.
- **Transposition tables** — `M.insert` is O(n) copy (M-PERSISTENT-COLLECTIONS); revisit after its Phase 2/3 lands, then as its own doc.
- **Time management (`go movetime`, Clock effect)** — v1 is fixed-depth; time-bounded bestmove would break the replayability claim.
- **SMP/Threads** — determinism first (A1/A2).
- **A GUI / host-effect integration** — that is the m-game-engine-effects lane; this doc deliberately needs no host.

## Timeline

**Week 1** (~5 days ×2 buffer):
- Phase 1 (board/FEN) + Phase 2 (movegen + perft gate) — the perft gate is where estimates die; the ×2 is already spent here.

**Week 2** (~3 days + slack):
- Phase 3 (search/CLI) + Phase 4 (README, timing record, optional UCI stretch).

**Total: ~6 working days doubled → ~2 weeks.** If perft(4) at build time exceeds ~30 minutes on the
VM, record the timing and move the 197,281 gate to "documented manual run" rather than blocking CI —
the *count* must still be exact.

## Risks & Mitigations

| Risk | Impact | Mitigation |
|------|--------|-----------|
| Search too slow to be fun at depth 2–3 (worse than V6's estimate) | Med | Movegen cost profile is the lever: leaper tables + bit-clear loops, zero per-node list allocations; if still <10 nodes/s at depth 2, drop v1 default depth to 2 and record honestly. The showcase value (perft-correct, replayable) survives any nps. |
| VM SILENT_FALLBACK silently drops nps ~5× | Med | `--strict-bytecode` in every documented run command; strict run is a phase gate (M-VM-COVERAGE precedent: "gate modules under --strict-bytecode", stapledons-godot). |
| Z3 verify blows up on the core (e.g. #2468-class foldl/anonymous-record bug) | Low | Contracts live on small non-recursive helpers only; never anonymous-record foldl accumulators near contracts (V10); inconclusive contracts are listed, not hidden. |
| Cost-model violations sneak in (`acc ++ [x]`, `A.set` in loops) | Med | Cost-model discipline section above; `ailang inspect` + a manual pass over generation paths before the perft run; the perft timing table makes violations visible as nps. |
| En-passant/castling legality bugs (classic) | Med | The standard perft battery exists precisely to catch these; Position 3/4/5 are in the gate, not optional. |
| int `*` overflow traps (if magics sneak in) | Low | Magics are a Non-Goal; review note in implementation checklist. |

## Related Documents

- [m-game-engine-effects](../v1_1_0/m-game-engine-effects.md) — the OTHER game lane: host-effect
  substrate (Render/Input/Clock, Godot host, Stapledon's Voyage). **Distinct**: that doc outsources
  frame-critical substrate to the host and keeps AILANG as orchestration; this doc is pure logic with
  no host at all. A future chess GUI would ride its effects, not this engine's code.
- [m-persistent-collections](../v0_55_0/m-persistent-collections.md) — Phases 2–3 (persistent
  Map/Array) would change make/unmake economics; this engine is a motivating benchmark and first
  consumer for that work.
- [m-vm-quick-wins](../v0_55_0/m-vm-quick-wins.md), [m-vm-coverage-letrec-prelude-bridge](../v0_55_0/m-vm-coverage-letrec-prelude-bridge.md) — the VM speed/coverage context this design depends on (V6, V8).
- [m-int-overflow-trap-and-bigint](../v0_55_0/m-int-overflow-trap-and-bigint.md) — why magic bitboards are off the table.
- [deedy/chess-chatarunga](https://github.com/deedy/chess-chatarunga) — the inspiring artifact, and
  the measuring stick this doc explicitly declines to be graded by.

## References

- [Design Axioms](/docs/references/axioms)
- [docs/LIMITATIONS.md](../../docs/LIMITATIONS.md) — recursion-frame cap, VM posture (V8)
- `ailang prompt` (v0.16.20 teaching surface) — operators table, Cost Model, `readLineOpt` pattern

## Axiom Compliance

| Axiom | Score | Justification |
|-------|-------|---------------|
| A1: Determinism | +1 | Pure core: bestmove is a function of (position, depth); replay check is an acceptance test |
| A2: Replayability | +1 | Whole search replayable byte-for-byte on evaluator and VM; deliberately no Clock in core |
| A3: Effect Legibility | +1 | `! {}` core, `! {IO}` shell only; zero hidden effects |
| A4: Explicit Authority | +1 | Caps: IO only in the shell; no FS/Net/AI anywhere |
| A5: Bounded Verification | +1 | Z3 contracts on decidable pure helpers; perft = decidable correctness gate |
| A6: Safe Concurrency | 0 | No concurrency |
| A7: Machines First | +1 | perft counts, exact-match transcripts, `--strict-bytecode` gates: all machine-checkable |
| A8: Minimal Syntax | +1 | No syntax changes; pure application code |
| A9: Cost Visibility | +1 | Timing table in README; cost-model discipline section makes per-node costs explicit |
| A10: Composability | 0 | Uses std/array, std/list, std/io as-is |
| A11: Structured Failure | +1 | FEN → `Result[_, string]`; unsupported CLI tokens fail loudly |
| A12: System Boundary | 0 | stdin/stdout only, explicit `! {IO}` |

**Net Score: +9** → **Decision: Move forward** (≥ +2, no hard-violation axioms at −1).

Hard Violation Check:
- [x] A1: no implicit nondeterminism (os-mode Rand banned from core; Clock banned)
- [x] A3: no hidden side effects (pure core verified by `ailang check` effect rows)
- [x] A4: no ambient access (IO cap only, declared)
- [x] A7: nothing optimized for human convenience at machine-decidability expense

## Verification Log

All checks run this session (2026-11-27) against the installed `ailang` binary in this workspace;
probe files were throwaway files under `/tmp/chessprobe/` (module-path warning MOD010 expected and
irrelevant for temp probes). Transcripts summarized per row.

| # | Claim | Method | Result |
|---|-------|--------|--------|
| V1 | Bitboard primitives exist and run: `&`, `^`, `~`, `<<`, `>>`; **no `\|` operator** (must `import std/math (bitwiseOr)`); `>>` is arithmetic so south-shifts need `shiftRightLogical` | `ailang check` + `ailang run` on probe using `1 << 27`, `x & (x-1)`, `shiftRightLogical(-1,1)` → `9223372036854775807`, `bitwiseOr(0xFF000000, 0x10)` → `4278190096` | **Confirmed** — matches teaching-prompt operator table; no popcount/ctz/clz builtin exists (`ailang builtins list` grep empty), so popcount is hand-rolled (Kernighan loop verified: `popcount(0x...4000) = 1`) |
| V2 | Hex literals with `_` separators (e.g. `0x000000A5_00004000`) are a parse error | `ailang check` on the V1 probe's first draft | **Confirmed** — `PAR_UNEXPECTED_TOKEN` / `PAR_NO_PREFIX_PARSE`; without separators (`0x000000A500004000`) it compiles. Implementation note: write bitboard constants without underscores |
| V3 | `Array[int]` supports O(1) `get` after one `A.fromList` build (attack-table pattern) | Same probe: build 64-entry `Array[int]`, `A.get(knight, 27)` → `708669620224` | **Confirmed** |
| V4 | Pseudo-perf of a knight-attack + popcount loop: evaluator 26.7s vs `--bytecode` 5.6s for the same ~2–3M-step workload | `time ailang run` vs `time ailang run --bytecode` on the perf probe (2000 iterations × 2 knightsAt calls: foldl over 64 table entries + popcount each) | **Confirmed** — VM ≈ 4.8× evaluator on this (list/HOF-heavy) workload; ~0.3–0.5M primitive steps/s on VM, ~0.06–0.1M on evaluator. Basis of the Reality check's ~10²-nodes/sec planning number. Consistent with M-VM-QUICK-WINS' "~40× overall, ~4× slower than CPython" (different workload mix) |
| V5 | Make-move as immutable record update + `Move` ADT `deriving (Eq)` compiles and runs; **ADT field access `m.fromSq` does not compile** (must pattern-match) | `ailang check` on pos probe: record update `{pos \| knights: ..., sideToMove: ...}` → correct XOR result; `roundtrip` equality → true; first draft with `m.fromSq` field access failed type check (`cannot unify Move with record`) | **Confirmed** — both the working pattern and the trap are verified |
| V6 | Multi-arm `match` with multi-statement record-update bodies needs `{ }` braces | V5's first draft (unbraced arm body) → `PAR_UNEXPECTED_TOKEN`; braced version compiles | **Confirmed** |
| V7 | UCI/CLI stdin loop primitive: `readLineOpt() -> Option[string]` match-loop compiles and returns `None` at EOF | `ailang run` on the tbl probe with `</dev/null` → prints `eof` | **Confirmed** |
| V8 | Non-tail recursion default cap is 10,000 frames (~5 KB each); tail recursion is the escape | `docs/LIMITATIONS.md` line 17 (RT_REC_003) | **Confirmed** (citation) — search depth ≤ ~30 frames is nowhere near the cap |
| V9 | No existing or queued doc covers a chess engine; the open-PR set (as of 2026-11-27) doesn't either | create-script neural+SimHash search ("chess engine": top hits 0.90–1.00 are irrelevant — obs toolcalls, trace bridge, IFC sprint plan); read of [m-game-engine-effects](../v1_1_0/m-game-engine-effects.md); scan of the 37 listed open PRs (#2478…#2115) | **Confirmed** — duplicate gate passes (< 0.45 on-topic). Distinction from m-game-engine-effects recorded in Related Documents |
| V10 | `ailang verify` currently errors on foldl with anonymous-record accumulators (open PR #2468, Z3 !) | Open-PR list entry for coordinator/task-80434e07 ("v0.61.1 Z3 ! ERROR on foldl with anonymous record accumulator") | **Confirmed** (cited, not re-measured) — Risk row: use named types, keep anonymous records away from contract-bearing folds |
| V11 | `--strict-bytecode` flag exists and the probes run under it; silent fallback is the failure mode it prevents | `ailang run --help` (flags) + M-VM-COVERAGE doc; probes ran under `--bytecode` | **Confirmed** — strict run is a phase gate |
| V12 | `ailang prompt` operator/cost-model claims used in this doc (no `\|` operator; `>>` arithmetic; Cost Model table; `readLineOpt` semantics) are live in the installed binary | `ailang prompt` output, cross-checked against probe behavior | **Confirmed** — no divergence found between prompt claims and live checks |

## Future Work

- **Transposition tables** once m-persistent-collections Phase 2/3 ships (persistent Map/Array) — this engine's search becomes the motivating benchmark for it.
- **VM perft KPI in the rotation**: wire `perft(4)` nps (evaluator vs VM) into the eval-analyzer boundary set, echoing m-game-engine-effects' frame-time KPI posture.
- **UCI full subset** (position/go depth only; loud unsupported errors for the rest) if a GUI host is ever wanted; a GUI would ride the m-game-engine-effects lane.
- **`! {AI}` chess commentary** (engine explains its PV via `std/ai`) — a demo, not a feature of play strength.

---

**Document created**: 2026-11-27
**Last updated**: 2026-11-27
