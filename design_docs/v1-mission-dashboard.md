# Mission Dashboard — V1

*Snapshot, overwritten every iteration. History lives in the charter STATUS block and the log.*

**Last iteration:** 270 · 2026-08-24 · **LANDED — `m-lint-unused-filter-vacuity` (`e194c2584`)**

## Latest release
v0.33.1 (dev at `e194c2584`) — **v0.34.0 is the outstanding ask, see Parked on Mark**

## What just landed
`make lint` filtered `unused` findings out **before** its own failure predicate, so a dead function
anywhere in scope left the gate at rc=0 printing a green checkmark. Reproduced with **no mutant** —
pristine `dev` reported `2 issues: * unused: 2` while `make lint` exited 0. Exposure ~7.5 months
(`f18bc48d8`). Both hidden findings resolved with commit-level provenance. `sonnet` **PASS 94/100**.

## Next picks (queue head first)
1. `m-protocol-closure-arm2-floor` — arm 2 lacks arm 1's stdlib-presence floor leg (judge, iter-268)
2. `m-protocol-closure-goos-scope` — closure gate blind to GOOS/build-tag files (judge, iter-268)
3. `m-lint-tmpfile-collision` — `make lint` judges a fixed shared `/tmp/lint.out` (iter-270)
4. `m-gemini-verdict-score-threshold` — `ValidateVerdict` enforces only half its invariant (iter-270)
5. `m-codex-streaming-test-flake` — parallel-load flake, rule 3m shape (judge, iter-270)

## Loop cadence + routing
launchd `dev.ailang.mission-control`, ~90 min. Controller `opus` · executor `codex:gpt-5.6-sol` ·
evaluator `sonnet` · designer rotation seed `claude:claude-fable-5` (untouched — no Fable spend).
Bookkeeping issue **#852** (namespaced key; the driver env still exports the stale bare `745`).

## Parked on Mark (OPEN ledger rows only)
- **`D-30`** — enforce the harness↔`ai-check` version coupling before the `not_applicable` split:
  (a) schema-versioned JSON, (b) `os.Executable()` same-binary bind, (c) accept + spot-check.
- **`D-31`** — split the designer rotation into authoring vs review lanes, or widen it? Two of its
  three entries cannot author (one is a quorum reviewer, one is sandbox-read-only).
- **`D-32`** — should an `inconclusive` verification obligation be exempted from the effective
  `cost_per_verified_success` arm, as `D-29` exempts `not_applicable`?
- **`D-34` (standing)** — `#764` is complete on `dev`; **the v0.34.0 tag is its delivery to World**,
  which pins upstream by release. The ask is pre-authorised; the tag is Mark's alone.
- **Running-skill reconcile** — main checkout is **2 ahead** / 18 behind with a concurrent agent's
  dirty tree, so the standing fast-forward authorization does NOT apply (it requires 0 ahead) and
  Principle 0 governs. Iteration 270's skill edit reaches origin but not the running copy.

## Quota posture
metered **$0.00** of $5. No Fable spend for three iterations. Buckets reset Mon 07:00 local — fresh.
