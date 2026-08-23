# Mission Dashboard — V1

> Snapshot only; history in `v1-mission.md` + `v1-mission-log.md`. Written: **iter 256, 2026-08-23**.

## Where we are
- **v0.33.1**. **BAR-FIRST** (`D-28`) until all 5 clauses close. **dev CI GREEN** at `96381331a`
  (16 checks, zero not-green). **Ledger 31 rows, THREE OPEN: `D-29`, `D-30`, `D-31`.**

## In flight / next
1. **`m-cohort-manifest-build-provenance`** — designed, 3 quorum rounds, blocked on ONE unanimous
   one-line predicate fix. **`[NEXT]`, in-loop, needs no ruling**; 4-step resume is in the doc.
2. **`m-contract-verification-coverage`** — still parked on `D-30` (predicate re-read, unchanged).
3. Then: `m-run-selector-enumeration-floor` · `m-effect-clock-net-fs-modes` ·
   `m-v1-orchestration-flagship` · A1/A2 (`D-25`) · clause-3 A/B · LC-2.

## New this iteration
- **`go build` in a linked git worktree silently emits an unidentifiable binary; `-buildvcs=true`
  does not even error** (rc=0, 0 vcs lines). A branch-attached worktree fails identically, so it is
  the worktree, not the detached HEAD. This loop mandates worktrees + scratch builds, so **every
  binary it builds is `"dev"`** — and three consumers silently accept it: the frozen cohort
  manifest (release evidence), the module cache's *compiler identity*, the `--bank-by-version` bucket.
- **The confirming quorum I was not required to run is what caught the final defect.** R3, 2/2
  present: my own `CommitClean()` checks `-dirty` on `Commit`, but ldflags sets
  `Commit=$(git rev-parse HEAD)` — a plain SHA — and dirtiness rides on `Version`. **The two
  stamping paths dirty different fields.**
- **`gpt5-6-sol` dropped out of R2 on `budget`** (the N−1 hole) and was **restored** with a solo
  raised-cap review rather than deciding short; restored, it rejected. **Rounds 2 and 3 both
  blocked on defects the CONTROLLER wrote** — stopped rather than force a third unreviewed one.

## Routing · Cost
Controller opus · designer fable ×2 (**diet-COMPLIANT**, iter-255's DOC-unit amendment) · sonnet ×1
(mechanical carve-out propagation, deliberately not a 3rd Fable run) · quorum ×3. **No
planner/executor/evaluator** — quorum blocked before routing. Metered **$0.3865** of $5.

## PARKED ON MARK — three one-word calls
- **`D-29`** — does a no-`ensures` function count against `isVerifiedSuccess`? **(a) exempt** →
  `$0.7778 → $0.2121` · **(b) keep strict** · **(c) publish both**.
- **`D-30`** — enforce the harness↔`ai-check` version coupling how? **(a) schema-version JSON** ·
  **(b) bind child to `os.Executable()`** · **(c) accept + spot-check**.
- **`D-31` (new)** — the designer rotation has ONE usable authoring lane (codex *is* quorum
  reviewer `gpt5-6-sol`; gemini cannot author). Instance 4, never filed until now.
  **(a) split authoring/review lanes** · **(b) widen** · **(c) accept, stop flagging**.

> The un-namespaced `design_docs/mission-dashboard.md` holds **Motoko's** snapshot — left untouched.
