# V1 Mission Dashboard

> 30-second control context. Snapshot, NOT a record — history lives in the charter STATUS block,
> `v1-mission-log.md` and `v1-mission-status-archive.md`. Overwritten every iteration.
> **Namespaced path**: `design_docs/v1-mission-dashboard.md`. Never write the bare
> `mission-dashboard.md` — that literal is shared by every mission on the rig.

**Updated**: 2026-08-20 (iteration 239) · **Release**: v0.33.1 · **Bookkeeping**: [#745](https://github.com/sunholo-data/ailang/issues/745)

## In flight

- **LC-1 `m-list-repr-spike` M6 — the kill criterion RAN. Verdict: GO.**
  PR [#810](https://github.com/sunholo-data/ailang/pull/810). Full matrix: 76 AC-1 points + 8 B-LEN,
  5 fresh-process trials each, 420 trials, 11m22s, darwin/arm64.
  All three candidates (C1, C2K8, C2K32) pass all five ratified clauses.
  Control leg fires at **9.85× / 10.38×** against a required ≥ 8×, so the gate is falsifiable.

## Next

- **LC-2 `m-list-accessor-api`** (2–3d) — the accessor seam over the unchanged slice + the
  `listrep` ratchet analyzer. Unblocked by the GO. Then LC-3a/3b/3c → LC-4 (riskiest) → LC-5.
- **LC-0 `m-list-interim-communication`** (0.5d) — `docs/LIMITATIONS.md` + a `#676` comment.

## Parked on Mark

- **Which representation LC-2…LC-5 build for.** The doc's tie-break ((c) then (b)) selects
  **C2(K=32)**, the chunked hybrid. The decomposition's **15.5–21.5 person-days** were scoped
  around **plain cons cells (C1)**, which passes every clause with margin ((c) 1.95× vs a 2.5×
  ceiling). Both are defensible; the matrix cannot decide it. **One word: `C1` or `C2K32`.**
- Carried, not a ledger row: rotate `AILANG_REGISTRY_API_KEY` (iter-232).

## Loop

- Cadence: launchd `dev.ailang.mission-control`, 6h watchdog. Kill switch armed (not set).
- Routing: controller **opus** · designer **rotation** (fable → codex → gemini) ·
  planner/executor **codex `gpt-5.6-sol`** · evaluator **sonnet** (generator≠judge).
- Decision ledger: **21 rows, 1 OPEN** (the representation choice above).
- Metered spend this iteration: **$0.00** of $5 — every lane used was a quota bucket.

## Quota posture

codex bucket reset 05:34 today and probed rc=0. Fable diet: at most one bounded run per
iteration (designer only) — not spent this iteration; no new doc was needed.
