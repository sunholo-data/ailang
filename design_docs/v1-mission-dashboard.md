# Mission Dashboard — V1

**Snapshot** (overwritten every iteration; history lives in the charter STATUS block + mission log).
Last updated: **2026-08-20, iteration 237**.

## Latest
- **Release**: v0.33.1 · `dev` green (16 checks SHA-addressed, zero not-green at `daf881eaf`).
- **Iteration 237 LANDED**: [#804](https://github.com/sunholo-data/ailang/pull/804) — LC-1 spike
  **M1+M2 of 6**. Evaluator (sonnet) PASS **93/100**, zero blocking.
  - **AC-2's control leg FIRES**: C0 `time(L=16384)/time(L=1024)` median **11.60×** (m=1024) /
    **12.37×** (m=4096) vs the ≥ 8 the criterion needs. C1 cons cells flat at **0.95×** / **1.08×**
    vs clause (a)'s ≤ 1.5. Heaviest cell **112 ms** C0 against **62 µs** C1.
  - *Provisional dev-loop reading, darwin/arm64, NOT M5's five-trial protocol.*

## In flight / next
- **LC-1 `m-list-repr-spike` — M3 RE-OWED, then M4–M6. The programme's go/no-go is still OPEN.**
  - **M3's C2 descope was REFUTED.** The executor marked chunked cons infeasible (the immutable API
    cannot detect unique chunk ownership). Premise true, conclusion false: the doc's C2 bound is
    already the *contended* one (copy ≤ K, O(K) = O(1) worst case), which always-copy meets with no
    ownership detection; and doc:419 tolerates an infeasible C2 only *"if C1 passes"*, which is M4
    data that does not exist. Independently confirmed by the evaluator. **Next iteration: M3 with
    an always-copy C2, then M4–M6.**
  - Why it matters: C2 is the candidate designed to pass where C1 is most at risk — clause (b)
    iteration at n=65536, clause (c) memory (C1 32 B/cell vs C2 ~16-18 B). Descoping it makes a
    **STOP reachable by descope rather than by measurement**, on a ~16-person-day gate.
- Nothing else in the cons-cells programme (LC-2…LC-5, LC-0) may route until LC-1 lands.
- Also queued above the programme: 8 decision-gated rows (D-2/8/9/10/11/13/14/COV-1) + 3 sweep orphans.

## Loop / routing
- Cadence: launchd, ~90 min. Controller opus · executor `codex:gpt-5.6-sol` · evaluator sonnet
  (own worktree) · designer rotation untouched this iteration (no new doc needed).
- Iteration 237 spend: **metered $0.00** of $5 — every lane was a quota bucket. No GPU, no `rig.lock`.

## Parked on Mark
- **None.** Decision ledger: 21 rows, **zero OPEN**.
- Carried, not a ledger row: rotate `AILANG_REGISTRY_API_KEY` (from iteration 232).

## Standing hazards worth knowing
- The installed `~/go/bin/ailang` is **37 commits stale** (`v0.33.1-125-gc575cd44e-dirty`). Anything
  that shells out to `ailang` from PATH — e.g. `tests/golden/codegen` — fails for the binary, not
  the repo. Two arms on the identical tree: stale **rc=1**, fresh **rc=0**.
