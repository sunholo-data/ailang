# Mission Dashboard — V1

*Snapshot, overwritten every iteration. History lives in the charter STATUS block and the log.*
**Last iteration:** 272 · 2026-08-25 · **LANDED — `m-protocol-closure-goos-scope`**

## Latest release
**v0.33.2** (2026-08-24 19:26Z). It already contains every `#764` milestone, so **no v0.34.0 is owed**
for that work — `D-34` is **discharged** (Mark, 2026-08-24 23:35Z) and `#764` is **closed**.

## What just landed
The closure gate ran every `go list` at the ambient GOOS, and CI invokes it only from the ubuntu
`test` job. The filed row demonstrated a `_linux.go` intruder — which is the one case CI *does*
catch. The real escape is `_darwin.go`/`_windows.go`, and it covered **both** arms, including the
`serveapi` facade the executor never probed. Both arms now run a GOOS matrix with their own
anti-vacuity floors (R12/R13). Self-test 5 → 9 arms. Evaluator PASS 88/100; its one blocking
finding (the `vacuous()` helper's platform attribution was unpinned) was reproduced and fixed
in-iteration.

## Next picks
1. `m-lint-tmpfile-collision` — `make lint` writes `/tmp/lint.raw`/`/tmp/lint.out` at fixed shared
   paths on a rig running three missions; one agent's findings can decide another's verdict.
2. `m-gemini-verdict-score-threshold` — `ValidateVerdict` enforces half its documented invariant.
3. `m-codex-streaming-test-flake` — judge-found; a bound that holds on one machine's load profile.

## Loop cadence + routing
Controller `opus` · executor `codex:gpt-5.6-sol` · evaluator `sonnet` (generator≠judge) ·
designer rotation seeded `claude:claude-fable-5`, **not spawned** for direct-fix iterations.
Metered spend this iteration **$0.00** of $5.

## Parked on Mark
- **D-30** — harness↔`ai-check` version coupling before the `not_applicable` split: (a) schema, (b) same-binary, (c) accept.
- **D-31** — designer rotation has ONE usable authoring lane: (a) split authoring/review lanes, (b) widen, (c) accept.
- **D-32** — should an `inconclusive` verification obligation be exempt from the effective KPI arm, as `D-29` exempts `not_applicable`?

## Quota posture
No Fable spend this iteration. Codex probe rc=0. Anthropic available.
