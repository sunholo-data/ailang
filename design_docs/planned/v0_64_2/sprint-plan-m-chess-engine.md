# Sprint plan for M-CHESS-ENGINE

Ship a pure AILANG chess engine in `examples/chess/`. A human CLI exposes the legal moves, fixed-depth search and perft. The pure core yields reproducible best moves, scores, principal variations and perft counts across the evaluator and strict bytecode VM. [Approved design](m-chess-engine.md) · [Chatarunga reference](https://github.com/deedy/chess-chatarunga)

**Target** v0.64.2 · **Risk** medium · **Duration** six active development days within ten working days · **Estimate** 1,930 LOC (1,460 implementation and documentation, 470 tests and fixtures).

## Baseline and dependencies

The checkout contains the design merge (#2479), and the repository scan found no existing chess showcase. The incoming PR list and a live search of 40 open PRs found no overlapping chess implementation. The velocity helper could see only the design merge in this shallow checkout. It returned historical changelog entries and no usable recent LOC or completion-rate data. The allocation below uses the design's six active days plus four reserve days for move legality and VM parity. The planned pace is about 322 LOC per active day, or 193 across the full window. Coverage baseline was unavailable because this container lacks `make`. Acceptance uses explicit behavior coverage.

AILANG prompt version loaded: v0.16.20. This is the installed CLI's active teaching prompt identifier. The checkout's `std/VERSION` is v0.64.1, and the installed CLI warns about staleness. Start execution with a CLI built from the target checkout and repeat focused syntax and strict-bytecode checks for signed bitboards, field updates, inline tests and EOF handling.

The approved representation uses twelve bitboards in an immutable record with classical rays. Search uses a fixed depth and deterministic tie-breaking. The only runtime capability is IO in the CLI. Persistent collections are future optimization context. NNUE, Elo targets, timed search, SMP, transposition tables, GUI and the optional UCI and seeded-randomness stretches are outside this sprint estimate. There are no open human-decision gates.

## Registry reuse

Ran `ailang pkg search` separately for `chess`, `fen`, `bitboard`, `perft`, `negamax`, `minimax` and `uci`. Every search returned zero candidates. Each milestone records `none`. The implementation can use the existing standard library for bit operations, arrays and IO.

The approved design describes tests without numeric IDs. The TST-CHESS IDs below label those checks for executor and evaluator traceability. Each milestone is a checkpoint. Use separate sessions for coding, fixture debugging and long timing runs where needed.

## Milestones

M1 → M2 → M3 → M4 → M5 → M6 is the implementation order. Run the long depth-four fixture separately from the fast suite and resolve any mismatch before closing the sprint.

### M1: Position and FEN (~530 LOC)

Day 1. Implement `examples/chess/board.ail` and `fen.ail` (410 implementation, 120 tests). Define square mapping, immutable position metadata and ASCII display. Preserve the FEN fullmove number alongside the design's halfmove clock so valid six-field input round-trips. Validate syntax and structural invariants through Result. Exercise the high bit and logical right shift. Registry decision `none` based on chess, fen and bitboard searches.

- [ ] TST-CHESS-01 Exhaustively round-trip all 64 squares and exercise empty and high-bit boards. Run ailang verify for bounded coordinate and index contracts and record any inconclusive result with its diagnostic.
- [ ] TST-CHESS-02 Round-trip startpos and canonical Kiwipete plus a FEN with non-default counters and an en-passant field. Reject malformed ranks, side, rights, square and counters with explicit errors.

### M2: Attacks and pseudo-legal moves (~290 LOC)

Day 2. Start `movegen.ail` (220 implementation, 70 tests). Use cached leaper tables, edge-masked rays and stable traversal. Iterate occupied bits without allocating a 64-square range per node. Registry decision `none` based on chess and bitboard searches.

- [ ] TST-CHESS-03 Fixtures cover pawn, knight, king and slider attacks, file wrap, blocked rays and signed high-bit occupancy. Ordinary candidates exclude friendly squares and king captures.

### M3: Legal moves and position transitions (~300 LOC)

Day 3. Complete `movegen.ail` (205 implementation, 95 tests). Apply captures, all four promotion choices and special moves before filtering by king safety. Handle current, transit and destination attacks for castling. Update rights on king or rook moves and corner-rook captures, update en passant and counters, and flip the turn. Registry decision `none` based on chess and bitboard searches.

- [ ] TST-CHESS-04 Fixtures cover pinned moves, castling through check, rights changes, en-passant discovered check, and quiet and capture promotions. Piece counts change only as implied by the move. Applying a legal move preserves the original immutable position.

### M4: Perft fixture runner (~170 LOC)

Day 4. Add `perft.ail` (100 implementation, 70 tests). Count legal leaves with depth zero equal to one. Use canonical FENs and a root-move breakdown to isolate mismatches. Start the depth-four evidence run before search work. Registry decision `none` based on chess and perft searches. [Canonical perft fixtures](https://www.chessprogramming.org/Perft_Results)

- [ ] TST-CHESS-05 Fast regression counts are startpos depths 1–3 = 20, 400, 8902. Kiwipete depths 1–2 = 48, 2039. Positions 3, 4 and 5 depths 1–2 = 14/264, 6/264 and 44/1486 respectively.
- [ ] TST-CHESS-06 A recorded strict-VM startpos depth-four run returns 197281. Run it outside the fast test gate. A mismatch blocks movegen acceptance.

### M5: Deterministic search (~350 LOC)

Day 5. Add `search.ail` (275 implementation, 75 tests). Implement negamax with alpha-beta, material and cited PST values, stable MVV-LVA, and bounded quiescence. Handle mate and stalemate before static scoring. In check, search legal evasions within the quiescence budget and disallow stand-pat. Represent a terminal root without a best move. Registry decision `none` based on chess, negamax and minimax searches.

- [ ] TST-CHESS-07 Fixed-depth fixtures cover a forced mate, stalemate, checked quiescence and stable ties. Every returned move belongs to the root legal list and every PV ply replays legally. Repeated runs return identical move, score and PV.

### M6: CLI and showcase evidence (~290 LOC)

Day 6, with days 7–10 reserved for corrections, parity and timing. Add `cli.ail`, `README.md`, a deterministic stdin fixture and an unreleased changelog fragment beginning with `### ` (250 implementation and docs, 40 tests). Thread position through `readLineOpt`; handle `position <fen>`, `show`, `moves`, `go <depth>`, `perft <depth>`, `quit` and EOF. Document the fixed-depth engine's practical limits. Registry decision `none` based on chess and uci searches and existing std IO.

- [ ] TST-CHESS-08 Compare stdout byte-for-byte across two evaluator runs and two strict VM runs using one fixture. Cover valid commands, malformed FEN retaining the last valid position, invalid depth, unknown command, quit and EOF. Use count-only output for perft and keep timing outside replay stdout. Test quit and EOF in separate fixtures so both paths execute.
- [ ] TST-CHESS-09 Record startpos depths 3 and 4 for evaluator and strict VM in the README and the design implementation record. Include commit, machine, CLI version, commands, counts, elapsed seconds and nodes/sec. If depth four exceeds 30 minutes, schedule and record the completed manual run outside CI. The exact count remains required.

## Showcase module checklist

| Module | Contracts | Effects | Inline tests |
| --- | --- | --- | --- |
| board | Include bounded coordinate-to-index and index-to-coordinate range contracts. Attempt popcount bounds separately and document solver limits for recursive helpers | Include pure `! {}` | Include boundary pairs and square round-trip property |
| fen | Skip: test recursive parsing through typed results and fixtures | Include pure `! {}` | Include valid round-trips and error fixtures |
| movegen | Skip: test state transitions and recursive legality with fixtures and properties | Include pure `! {}` | Include attack and transition fixtures and piece-count property |
| perft | Skip: use exact published counts for recursive traversal | Include pure `! {}` | Include depth zero and fast perft battery |
| search | Skip: fixture and property coverage for recursive search | Include pure `! {}` | Include terminal positions, legal PV and tie fixtures |
| cli | Skip: verify effectful interaction through transcripts | Include `main() -> () ! {IO}` | Skip: test IO through scripted stdin and stdout |

## Execution checks

- Run `ailang check` and `ailang test` on each applicable chess file as it lands. Run `ailang verify examples/chess/board.ail` and retain contract outcomes.
- Exercise every chess module through an exported example entry point or imported from a smoke entry point with `ailang run --bytecode --strict-bytecode`. Use `--caps IO` only for entry points that print or read input. Compare runtime output with equivalent evaluator invocations.
- The focused chess commands cover `examples/chess/` directly. Before handoff, also run `make test-changed`, `make verify-examples`, `make check-file-sizes` and `make check-changelog`. Keep modules below 800 lines. Audit the diff for scope and review hot loops for repeated list append and per-node array writes.
- In the README, provide runnable CLI, replay and timing commands. Manually inspect a short depth-two session including promotion and en-passant positions. Treat perft as evidence for the covered move-generation cases and report solver coverage only for verified helpers.
