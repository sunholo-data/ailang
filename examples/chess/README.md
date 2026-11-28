# AILANG chess engine

Explore standard chess positions with an engine written in AILANG. The engine
uses immutable bitboards for move generation and fixed-depth alpha-beta
negamax for move selection. Equal scores retain the first move in the stable
search order. Search uses material and piece-square scores, capture ordering,
and a bounded quiescence search that includes legal evasions when in check.
The material values and positional priorities are based on Tomasz
Michniewski's [Simplified Evaluation Function](https://www.chessprogramming.org/Simplified_Evaluation_Function).
[Chatarunga](https://github.com/deedy/chess-chatarunga) inspired the showcase.

## Run from the repository root

```bash
ailang run --caps IO --no-print examples/chess/cli.ail
```

| Command | Result |
| --- | --- |
| `position <fen>` | Load a six-field FEN. An error preserves the current position. |
| `show` | Print the board and side to move. |
| `moves` | List legal moves using coordinate notation such as `e2e4` and `a7a8q`. |
| `go <depth>` | Search one to three plies and report best move, score and principal variation. |
| `perft <depth>` | Count legal move paths at depth zero to four. |
| `quit` | Exit. End of input also exits. |

The starting position is loaded on launch. `go` analyzes the current position
independently each time. Load the next position with `position` to continue an
analysis. Scores are in centipawns from the side-to-move perspective, with
separate values for mate. A terminal position prints `bestmove -`.

Use depth one or two for interactive exploration. Depth three can take longer
in positions with many legal moves and captures. The engine covers checkmate
and stalemate. Draws that require game history or material adjudication,
including repetition and the fifty-move rule, are outside this showcase.

## Verify

Run the inline fixtures and board contracts on the checkout CLI.

```bash
for module in board fen movegen perft search; do
  ailang test "examples/chess/$module.ail" || exit 1
done
ailang verify examples/chess/board.ail
```

Z3 verifies the coordinate bounds for `coordToSq`, `fileOf` and `rankOf`.
It skips the `popcount` contract because its verifier cannot encode
`bitwiseXor_Int`. The board tests cover all 64 coordinate and bit-index
round trips.

The [published perft fixtures](https://www.chessprogramming.org/Perft_Results)
include Position 3 at depth two with 191 leaves and Position 4 at depth two
with 264 leaves. Use the pure `divide(position, depth)` entry point from
`perft.ail` in another AILANG module to inspect the count by root move.

Compare two evaluator runs and two strict VM runs for each fixture. The
first fixture tests `quit` by supplying a command after it. The second
fixture ends with a command whose final line has no newline.

```bash
outputs=$(mktemp -d)
for fixture in replay replay-eof; do
  for trial in 1 2; do
    ailang run --caps IO --no-print examples/chess/cli.ail \
      < "examples/chess/$fixture.txt" > "$outputs/$fixture-eval$trial" || exit 1
    ailang run --strict-bytecode --caps IO --no-print examples/chess/cli.ail \
      < "examples/chess/$fixture.txt" > "$outputs/$fixture-vm$trial" || exit 1
  done
  for trial in eval2 vm1 vm2; do
    diff -u "$outputs/$fixture-eval1" "$outputs/$fixture-$trial" || exit 1
  done
done
rm -r "$outputs"
```

## Perft timings

Use Bash's `time -p` to record wall-clock time separately from the result.

```bash
for depth in 3 4; do
  time -p env GOMAXPROCS=2 ailang run --entry startCount --args-json "$depth" examples/chess/perft.ail
  time -p env GOMAXPROCS=2 ailang run --strict-bytecode --entry startCount --args-json "$depth" examples/chess/perft.ail
done
```

Measured on 2026-11-28 in a Linux x86-64 container on an Intel Xeon 2.00 GHz
host. The perft source is commit `78afd8dbf5e12c3573c22536bce2afb8499959b8`.
The checkout-built CLI reports `AILANG dev`, commit
`3185145fed277dfc9e3c8fb185d71c66ad901e50`. Runs used `GOMAXPROCS=2`,
overlapped with other verification, and include CLI startup and module loading.

| Runtime | Depth | Leaves | Wall seconds | Leaves/sec |
| --- | ---: | ---: | ---: | ---: |
| Evaluator | 3 | 8,902 | 10.98 | 811 |
| Evaluator | 4 | 197,281 | 143.70 | 1,373 |
| Strict bytecode VM | 3 | 8,902 | 3.40 | 2,618 |
| Strict bytecode VM | 4 | 197,281 | 16.81 | 11,736 |
