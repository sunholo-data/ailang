# Mission Dashboard — the 30-second control context
> **Contract**: ≤40 lines, overwritten by mission-control Gate 4 every iteration (history lives in
> the charter/log). Fresh session = THIS + MEMORY.md. Humans steer via the bookkeeping issue.

**Updated**: 2026-08-13 ~11:50 local (iteration 191)

## Now
- **v0.33.1** · `dev` @ `14efcae22` (skill edit, Gate 3b pending at write time — verdict in the
  charter STATUS). Previous HEAD `b60e41946` was fully green (SHA-addressed `checks=16`, zero
  not-green).
- **`[SWEEP iter-158]` batch RESOLVED — all six dispositioned at HEAD:**
  - `#609` **CLOSED**: `toInts` shipped 5 days before triage (`1677fcff9`, in v0.33.1); live-verified.
  - `#611` **CLOSED**: driver half had landed at `d14f106bb`; the in-iteration half is iter-191's
    skill edit `14efcae22` (Gate-3 codex/pi fallbacks now follow the ratified chain).
  - `#607` **CONFIRMED** → new row `m-batch-exit-panic` (batch `exit(1)` = raw Go panic, item 2
    never runs; single-file path recovers cleanly — guard-one-path-miss-the-sibling class).
  - `#610` direction-CONFIRMED (~2.4× synthetic) but **49× does NOT reproduce without duckdb** →
    new row `m-mapE-queryall-retention`, repro infra gates the design.
  - `#581` re-CONFIRMED (fail-safe, low-P, stays open) · `#554` quiescent since 08-11 (ops watch).

## Parked on Mark (all on `#635`)
- `D-1`, `D-2`, `D-7`–`D-14` — no reply since 2026-08-13T04:58Z watermark; all still open.

## Next (if nothing unparks)
- **`m-batch-exit-panic` (`#607`)** — P1 reliability, ~0.5d, no new doc needed; fix shape
  established (mirror the `main_run_exec.go:549` recover into `executeBatchItem`), CI fixture +
  mutation drill specified in the queue row.

## Loop health
- ⚠ **Driver `$MODEL` env was UNSET this fire — controller rode fable-5 by session inheritance**,
  against opus-first. No sub-agents spawned (triage-only), so no other role affected. Driver-side
  fix needed; flagged in the iter-191 digest.
- Designer rotation pointer unchanged (`codex:gpt-5.6-sol` last-used; no designer fired).
  `metered=$0.00` — sqlite reads + local runs only.
- `#665` end-to-end confirmation still waits on the **2026-08-19** Wednesday motoko fmt A/B.
- Binary trap held again: `~/go/bin/ailang` warned stale at session start. Rebuild before probes.
