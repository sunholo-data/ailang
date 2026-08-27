# Mission Dashboard — V1

*Snapshot, overwritten each iteration. History: `v1-mission.md` (STATUS) + `v1-mission-log.md`.
Written by iteration 290, 2026-08-27.*

**Release** v0.34.0 · **dev HEAD** `0911d1089` · **iteration 290 metered $0.00** of $5

**Just landed — `m-fmt-printer-line-width-limit` is COMPLETE** (M3, PR #931). Corpus lines
>120 runes **159 → 100**; max line **1315 → 316**; `LET_CHAIN_2PLUS` residual **20 → 0**.
50 files, no printer code touched. Doc + plan → `implemented/v0_35_0/`. Judge **PASS 94/100,
zero blocking** — and it found two defects in the controller's own record (both fixed).

## Next two picks

1. **`m-fmt-corpus-gate-freeze`** — `D-39` sequences the gate freeze behind the width work,
   which is now done. The judge **measured** why it matters: reverting two corpus files to
   their pre-M3 spelling leaves **every gate rc=0**, because the corpus test asserts
   `Format(Format(x))==Format(x)` and never `data==Format(data)`. All 450 files can silently
   de-canonicalise with CI green.
2. **`m-browser-session-serving-mode`** *(AITANA-DEMAND)* — Aitana wants `ailang browser serve`
   so a non-Go consumer can use browser sessions as a Cloud Run service. Ghost-disciplined
   REAL at HEAD (no serve verb, no HTTP server under `internal/browser`; controls fire).
   Does not outrank the queue.

## Routing

Controller `opus` · designer ROTATION (`fable-5` → `pi:ollama/kimi-k3:cloud`), **unspent 2
iterations** — no new doc needed · planner + executor `codex:gpt-5.6-sol` · evaluator `sonnet`,
always its own worktree. ⚠ Driver has fired **UNPINNED six consecutive times**
(`MISSION_WORKDIR`/`AILANG_DRIVER_SRC` unset) — harmless so far (running skill `cmp`-identical
to origin) but provenance is the main checkout, not the pin.

## Parked on Mark

**`D-41`** — the only OPEN ledger row of 41. May an ACTIVE prompt version be edited in place, or
must a content change bump the version? Bears on eval-baseline reproducibility. One word decides it.

## Standing amber

- **SonarCloud `failure` on dev for ≥6 consecutive commits** — inherited, **non-required**, does
  not block merges. Conditions: 60.1% coverage-on-new-code (needs ≥80%) + **B security rating on
  new code**. Named so a standing red stays visible.
- **`#847`** (nightly-eval `explicit_dataflow_ssa`) open by design — local-model capability gap,
  triaged iteration 266, verdict on the issue.
- **Main checkout's `dev` is 8+ behind `origin/dev`.** State read from origin, records land via
  worktrees. A reconcile is provably non-destructive (obligations measured) but needs Mark's OK.
