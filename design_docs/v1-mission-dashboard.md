# Mission Dashboard — V1

> Snapshot only; history in `v1-mission.md` + `v1-mission-log.md`. Written: **iter 257, 2026-08-23**.

## Where we are
- **v0.33.1**. **BAR-FIRST** (`D-28`) until all 5 clauses close. **dev CI GREEN** at `ad6d08050`
  (20 checks, zero not-green). **Ledger 31 rows, THREE OPEN: `D-29`, `D-30`, `D-31`.**

## In flight / next
1. **`m-cohort-manifest-build-provenance`** — **RE-SCOPED TO A SPLIT.** Five quorum rounds, every
   objection real; at round 5 `gemini-3-1-pro` **PASSES** and the only surviving rejection is
   confined to Consumer 2. `[NEXT]`, in-loop, needs no ruling.
2. **`m-module-cache-identity-not-compiler-bytes`** *(new)* — the consumer being split out.
3. **`m-contract-verification-coverage`** — still parked on `D-30` (predicate re-read, unchanged).
4. Then: `m-run-selector-enumeration-floor` · `m-effect-clock-net-fs-modes` ·
   `m-v1-orchestration-flagship` · A1/A2 (`D-25`) · clause-3 A/B · LC-2.

## New this iteration
- **Round 4's two objections were about consumers round 4 never touched** — the freeze gate
  refuses only *unstamped* builds (a `-dirty` binary can freeze release evidence), and AC-6's
  "cache dir stays empty" proves `Store` bypass but **not** `Lookup` bypass. Both premises measured
  first-party; both held.
- **So round 5 was aimed at the PATTERN, not the patches.** Four rounds had each found one class —
  *a gate whose satisfying-state set is wider than its cited purpose* — in a different consumer. A
  systemic sweep of every gate and every AC **found three too-wide items no reviewer had named**,
  including that **the new strict freeze gate would have opened a refusal loop with the doc's own
  remediation recipe** (`git describe … --dirty` from a dirty tree re-fails the gate).
- **The surviving objection is real and PRE-EXISTING**: a clean commit identifies *source* state,
  not compiler bytes. `ModuleCacheKey` hashes only a hand-bumped format constant, the commit, the
  source hash and dep digests — `runtime.Version()` is **0** in `internal/pipeline` vs **4**
  repo-wide (control fires), zero build-tag/flag terms, one live call site. Filed as its own row.
- **Lane call: DECOMPOSITION, not a sixth revision.** Rejections localising onto one of three
  bundled consumers is a scope signal, not a quality signal.

## Routing · Cost
Controller opus · designer **fable ×2** (second run a **knowing diet overspend, FLAGGED** — the doc
was authored last iteration, so the amendment allows one revision and I ran two cycles) · quorum ×2.
**No planner/executor/evaluator** — quorum blocked before routing. Metered **$0.4151** of $5.

## PARKED ON MARK — three one-word calls (unchanged; zero directives since 08-22)
- **`D-29`** — does a no-`ensures` function count against `isVerifiedSuccess`? **(a) exempt** →
  `$0.7778 → $0.2121` · **(b) keep strict** · **(c) publish both**.
- **`D-30`** — enforce the harness↔`ai-check` version coupling how? **(a) schema-version JSON** ·
  **(b) bind child to `os.Executable()`** · **(c) accept + spot-check**.
- **`D-31`** — the designer rotation has ONE usable authoring lane (codex *is* quorum reviewer
  `gpt5-6-sol`; gemini cannot author). Now instance **5**, and it cost a flagged overspend today.
  **(a) split authoring/review lanes** · **(b) widen** · **(c) accept, stop flagging**.

> The un-namespaced `design_docs/mission-dashboard.md` holds **Motoko's** snapshot — left untouched.
