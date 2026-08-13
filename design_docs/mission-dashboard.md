# Mission Dashboard — the 30-second control context
> **Contract**: ≤40 lines, overwritten by mission-control Gate 4 every iteration (history lives in
> the charter/log). Fresh session = THIS + MEMORY.md. Humans steer via the bookkeeping issue.

**Updated**: 2026-08-13 ~08:05 local (iteration 190)

## Now
- **v0.33.0** · `dev` @ `9ebdad07c` (mine) → `30cf295dc` (sibling pushed mid-poll). Gate 3b GREEN on
  my SHA: `checks=16`, `pending=0`, zero not-green — the count climbed **13→16** during the poll, so
  `pending=0` was required, not inferred.
- ⏸ **`#539` `m-dialect-keyword-diagnostics` — DOC LANDED, PARKED on `D-14`.** Defect real at HEAD
  (all 9 shapes: `case`/`switch` lead with a `PAR020` "add a `;`" that cannot work and never name
  `match`; `struct`/`class` don't even parse-error, degrading to `undefined variable`). Quorum
  BLOCKED ×2, `absent_reviewers: []` both rounds. R1 correctly re-scoped it 9 spellings/3.5d →
  `case` only/1–1.5d on a Minimal-Frozen-Core objection.
- **The R2 blocker is a fact about the parser, not the doc**: fixed **4-token lookahead**
  (`parser.go:134-139`), `nextToken()` a pure forward shift (`:214-220`), **ZERO**
  save/restore/rewind/backtrack methods (control: 125 `*Parser` methods). "Lookahead/reparse past an
  arbitrary subject expression" is infeasible as written.

## Parked on Mark (all on `#635`)
- `D-1`, `D-2`, `D-7`–`D-13`, **`D-14` (new)**.
- **`D-14`, one word** — `#539`'s detection mechanism: **(A)** recovery-site detection at
  `parser_literals.go:562`, no rewind needed, fits the parser as-is · **(B)** statement-initial soft
  keyword, fits 4 tokens only for simple subjects · **(C)** add parser backtracking, touches all 125
  methods, a core change against the north star · **(D)** drop it — one observed occurrence, and
  `ailang fmt` can't be the auto-fix (parses *before* formatting; invalid input returns identical).

## Next (if nothing unparks)
- `[SWEEP iter-158]` batch: `#611` (executor fallback chain — most actionable), `#581`
  (planner-lane parses fenced bullets as paths), `#554` (ollama tool-call collapse), plus `from:cli`
  `#610` (~49× memory), `#609` (`std/bytes` no `toInts`), `#607` (batch `exit()` panics the run).

## Loop health
- Controller **opus**; designer rotation `codex:gpt-5.6-sol` → advanced to `claude:claude-fable-5`.
  Planner/executor/evaluator not fired (parked before a sprint existed). `metered=$0.1077` vs `$5`.
- ⚠ **Two record defects repaired**: iter-189's STATUS rotation had **dropped iter-186's stamp**
  (recovered from `8ecebc0e1`); iters 188/189 both recorded a "Next" contradicting the queue order.
- ⚠ **`#665` end-to-end NOT yet confirmed** — fix is delivered, but the affected leg is the
  **Wednesday** motoko fmt A/B, so real confirmation is **2026-08-19**. The 08-13 run banked 24 rows
  but was INVALID (`infra_outage`, 4/12 timeouts) — that is `#649`.
- Binary trap: `~/go/bin/ailang` was **35 commits stale + dirty** at pick time. Rebuild before probes.
