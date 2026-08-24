# Mission Dashboard — V1

*Snapshot, overwritten every iteration. History lives in the charter STATUS block and the log.*
**Last iteration:** 271 · 2026-08-24 · **LANDED — `m-protocol-closure-arm2-floor` (`fffe2487b`)**

## Latest release
**v0.33.2** (2026-08-24 19:26Z) — already contains all four `#764` milestones plus iteration 270's
lint fix, so `D-34`'s premise ("v0.34.0 is the delivery to World") is **stale**. World told on `#764`.

## What just landed
The iteration-268 judge row named ONE hole in the closure gate's `./serveapi` arm; reproducing it
first-party found **two**, the unnamed one worse. Arm 2 runs a *second* enumeration — the
module-root query the allowlist check actually consumes — with no floor at all, its exit status
discarded as a pipeline head. Reducing it 10 roots → 0 left the gate green. `R10` + `R11` close
both. `sonnet` **PASS 96/100**, zero blocking.

## Next picks (queue head first)
1. `m-protocol-closure-goos-scope` — gate blind to GOOS/build-tag files (judge, iter-268)
2. `m-lint-tmpfile-collision` — `make lint` judges a fixed shared `/tmp/lint.out` (iter-270)
3. `m-gemini-verdict-score-threshold` — `ValidateVerdict` enforces half its invariant (iter-270)
4. `m-codex-streaming-test-flake` — dies under full-parallel `make test` (judge, iter-270)
## Loop cadence + routing
launchd `dev.ailang.mission-control`, ~90 min. Controller `opus`; designer ROTATION (pointer
`claude:claude-fable-5`, untouched — direct-fix iterations spawn none); planner/executor
`codex:gpt-5.6-sol`; evaluator `sonnet` (≠ executor, generator≠judge holds). Metered **$0.00** of
$5 — both quota buckets. Anthropic up; codex probe rc=0; billing tripwire CLEAN.

## Parked on Mark (`scripts/mission_decisions.sh --open`)
- **`D-30`** — enforce the harness↔`ai-check` version coupling: (a) versioned JSON schema,
  (b) bind to `os.Executable()`, (c) accept + spot-check.
- **`D-31`** — split the designer rotation into authoring vs review lanes, or widen it? Two of
  three entries cannot author for structural reasons no probe clears. 4+ instances.
- **`D-32`** — exempt an `inconclusive` obligation from `cost_per_verified_success`, as `D-29`
  exempts `not_applicable`?
- **NEW, one word** — `D-34` pre-authorised asking for **v0.34.0** as `#764`'s delivery; v0.33.2
  already delivers it. Ask **discharged**, or still want the minor bump to signal the new public
  `serveapi/protocol` surface?

**Rig note:** the MAIN checkout held a concurrent session's live uncommitted work at this fire;
left strictly alone, all writes went to worktrees.
