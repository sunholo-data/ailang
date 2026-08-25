# Mission Dashboard — V1

*Snapshot, overwritten each iteration. History lives in the charter STATUS block + `v1-mission-log.md`.*

**Last iteration:** 274 · 2026-08-25 · **LANDED** `92376bad3` (PR #876) · evaluator **80/100**

## Now
- **Latest release:** `v0.33.2` (2026-08-24). `dev` is ahead; no release owed.
- **Just landed:** `m-ci-wiring-unpinned` — nothing validated that the repo's `make check-*` gates
  were **wired into** GitHub Actions: deleting a gate step from `ci.yml` left all eight local gates
  at rc=0. The audit found a bigger target the row never named: `make ci` says *"Run full CI
  verification"* and `.claude/rules/dev-workflow.md:22` tells every agent to run it, while its
  measured overlap with what Actions really invokes was **8 of 46**. Three assertions now live in
  `internal/cihygiene` — already run by `go test ./...`, so the wiring gate is CI-connected **by
  construction** rather than needing a step that could itself be un-wired. `make ci` gained 11 gates.
- **Next picks:** `m-gemini-verdict-score-threshold` → `m-codex-streaming-test-flake`, then the two
  rows filed this iteration (`m-verify-targets-unwired`, `m-ci-composite-action-blind-spot`).

## Loop health
- Cadence: nightly launchd, pinned worktree at `origin/dev`; running skill byte-identical to origin.
- Routing: controller `opus` · executor `codex:gpt-5.6-sol` · evaluator `sonnet` (generator≠judge).
  Designer/planner unspawned for **5** consecutive direct-fix iterations — **Fable unspent**.
- Cost: metered **$0.00** of $5 this iteration; quota buckets only.
- Last 5 iterations all LANDED with an evaluator PASS (274: 80 · 273: 86 · 272: 88 · 271: 96 · 270: 94).

## Worth knowing (from this iteration)
- **11 `verify-*` targets are unwired from CI**, incl. `fmt-check-ail` (whose enumerator iteration 187
  already measured as pointing at a non-existent `stdlib/`, 46 files invisible) and
  `verify-stdlib-selftest` (the target iteration 273 had just fixed). Filed as `m-verify-targets-unwired`.
- `make verify-examples-trace` is currently **rc=1** (2/217 examples failing) and takes 135s — an
  orthogonal pre-existing failure nobody is watching, since no workflow invokes it.

## Parked on Mark (3 open decisions — see the ledger, none new)
- **D-30** — enforce the harness↔`ai-check` version coupling before the `not_applicable` split lands.
  Options: (a) versioned JSON schema, (b) `os.Executable()` same-binary bind, (c) accept + spot-check.
  *Blocks the headline cost-per-verified-success KPI.*
- **D-31** — split the designer rotation into authoring vs review lanes (or widen it). Two of its three
  entries cannot author at all; the usable rotation has ONE entry. 4 instances recorded.
- **D-32** — should an `inconclusive` verification obligation be exempted from the effective
  `cost_per_verified_success` arm, as `D-29` exempts `not_applicable`?

## Standing
- `D-34` discharged (iter-272): `v0.33.2` shipped `serveapi/protocol`; `#764` closed. Do not re-ask.
- Releases remain Mark's sole decision; the loop stops at ready-to-release.
