# Mission Dashboard — V1

*Snapshot, overwritten each iteration. History lives in the charter STATUS block + `v1-mission-log.md`.*

**Last iteration:** 273 · 2026-08-25 · **LANDED** `fcc220c0e` (PR #874)

## Now
- **Latest release:** `v0.33.2` (2026-08-24). `dev` is ahead; no release owed.
- **Just landed:** `m-lint-tmpfile-collision` — `make lint` and `make verify-stdlib-selftest`
  wrote to fixed `/tmp` paths shared across all three missions' clones. `lint` computed its
  **verdict** from a shared file (false green AND false red, both reproduced); the selftest
  restored a **tracked** source file from a shared backup (tracked-file corruption, reproduced
  deterministically). Both now use per-invocation `mktemp` + cleanup traps, and a new
  `check-tmpfile-hygiene` CI gate (11-arm self-test) refuses the class.
- **Next picks:** `m-gemini-verdict-score-threshold` → `m-codex-streaming-test-flake`, then the
  two rows filed this iteration (`m-ci-wiring-unpinned`, `m-tmpfile-hygiene-residual`).

## Loop health
- Cadence: nightly launchd, pinned worktree at `origin/dev`; running skill byte-identical to origin.
- Routing: controller `opus` · executor `codex:gpt-5.6-sol` · evaluator `sonnet` (generator≠judge).
  Designer/planner unspawned for 4 consecutive direct-fix iterations — **Fable unspent**.
- Cost: metered **$0.00** of $5 this iteration; quota buckets only.
- Last 4 iterations all LANDED with an evaluator PASS (272: 88 · 271: 96 · 270: 94 · 273: 86).

## Parked on Mark (3 open decisions — see the ledger, none new)
- **D-30** — how to enforce the harness↔`ai-check` version coupling before the `not_applicable`
  split lands. Options: (a) versioned JSON schema, (b) `os.Executable()` same-binary bind,
  (c) accept + spot-check. *Blocks the headline cost-per-verified-success KPI.*
- **D-31** — split the designer rotation into authoring vs review lanes (or widen it). Two of its
  three entries cannot author at all; the usable rotation has ONE entry. 4 instances recorded.
- **D-32** — should an `inconclusive` verification obligation be exempted from the effective
  `cost_per_verified_success` arm, as `D-29` exempts `not_applicable`?

## Standing
- `D-34` discharged (iter-272): `v0.33.2` shipped `serveapi/protocol`; `#764` closed. Do not re-ask.
- Releases remain Mark's sole decision; the loop stops at ready-to-release.
