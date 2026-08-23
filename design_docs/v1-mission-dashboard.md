# Mission Dashboard — V1

> Snapshot only; history in `v1-mission.md` + `v1-mission-log.md`. Written: **iter 255, 2026-08-23**.

## Where we are

- **v0.33.1**. **BAR-FIRST** (`D-28`): bar items outrank cons-cells until all 5 clauses close.
- **dev CI** GREEN at `30176187f` — 16 checks, zero not-green. Main checkout reconciled to 0/0.
- **Ledger** 30 rows, **TWO OPEN — `D-29` and `D-30`, and BOTH gate the same queue row.**

## In flight / next

1. **`m-contract-verification-coverage` — designed, twice-reviewed, PARKED on `D-30`.** 730-line
   doc banked; two full quorum rounds; **no code shipped**.
2. **Un-blocked bar items, need no ruling:** `m-cohort-manifest-build-provenance` (small, blocks a
   *publishable* clause-5 baseline) · `m-run-selector-enumeration-floor` · `m-effect-clock-net-fs-modes`.
3. Then `m-v1-orchestration-flagship` · A1/A2 (`D-25`) · clause-3 prompt A/B · then LC-2.

## New this iteration

- **The quorum earned its keep twice.** R1 (2/2 present, absent=[]) caught a **writer-before-reader**
  milestone order that dropped the `not_applicable` count, then left the observatory reading a
  shrunken `VerifySkipped` — it would have **shipped the banned `D-29` flip**. Fixed verbatim;
  intermediate states are now identical *by construction*.
- **R2 found the hole commit order cannot close.** `RunAICheck` resolves its verifier child from
  **PATH** (`verify.go:47-53`); **2 of 2** live callers pass `""`, so parent and child are
  independently versioned — skew is **live on this rig now** (`-211-dirty` vs `-216`), and
  post-split an old reader silently banks a reduced count. That is `D-30`. Parked, not
  force-passed: the carve-out needs a fix requiring no controller judgment.
- **Designer rotation collapsed structurally a 3rd time** (codex *is* reviewer `gpt5-6-sol`; gemini
  cannot author) — the ≥3-evidence bar for a routing-policy change is now **MET**.

## Routing · Cost · Parked

Controller opus · designer fable **×2** (diet overspend, FLAGGED) · quorum `gpt5-6-sol` +
`gemini-3-1-pro` ×2. **No planner/executor/evaluator** — quorum blocked before routing.
Metered **$0.1960** of $5.

**PARKED ON MARK — two one-word calls, both on the same row:**
- **`D-29`** — no-`ensures` function counts against `isVerifiedSuccess`? **(a) exempt** →
  `$0.7778 → $0.2121` · **(b) keep strict** · **(c) publish both**.
- **`D-30`** — enforce the harness↔`ai-check` version coupling how? **(a) schema-version JSON** ·
  **(b) bind child to `os.Executable()`** · **(c) accept + spot-check**.
