# Docs Mission Dashboard (snapshot — history lives in the charter + log)

**Last updated**: 2026-08-31T08:07Z, iteration 2.

## Status
Recovered a died-mid-flight prior fire: `docs-9` ("intro.mdx is stale at v0.16.0") was verified
FALSE — the version tags are intentional historical ship-versions, not current-version claims —
and `[RULED OUT]`. Landed [PR #973](https://github.com/sunholo-data/ailang/pull/973) →
`ad7542ba5` (squash). `check_versions.sh`'s Check 3 (the false-positiving instrument) is deleted.
CI/Deploy-docs poll on the merge commit was still in flight when this dashboard was written; see
the log entry for the final read.

## Blocking on Mark
None. D-1 and D-2 (previous decisions) are both `RESOLVED`. No new ask this iteration.

## Queue (top = next)
1. `[LANDED]` docs-2 · clauses 1+3 · first `docs-sync` sweep.
2. `[RULED OUT]` docs-9 · intro.mdx version claim was a false positive of `check_versions.sh`.
3. `[NEXT]` docs-5 · clause 2 · fix the 9 genuinely-failing runnable examples.
4. `[NEXT]` docs-6 · clause 1 · fix `check_examples.sh`'s absolute-path bug.
5. `[NEXT]` docs-10 (new, this iteration) · `make verify-examples` never compares `expected.stdout`
   and has no anti-vacuity floor (#670, #654 — found by the weekly external-issue sweep).
6. `[PARKED]` docs-8 · 126 overdue planned design docs (aggregate triage).
7. `[NEXT]` docs-1 · clause 7 · build the inbox-routing trigger.
8. `[PARKED]` docs-3 / docs-4 · benchmark audit / taxonomy pass.

## Loop cadence + routing
launchd `dev.ailang.mission-docs`, every 6h, staggered against v1/world/motoko. Routing ladder:
subscription (`claude-sonnet-5`/`codex:gpt-5.6-luna`) → flat-rate (Ollama Cloud) → metered
OpenRouter twin. Evaluator vendor-disjoint from executor at every rung. Metered ceiling $1/iter.

## Cost this iteration
$0.00 of $1 — no new model-role spawns; this iteration verified and landed a prior fire's output
plus controller-session bookkeeping (Gate 0 weekly sweep, Gate 4/5).

## Quota posture
No fallback triggered. `launchd drivers (bash 3.2)` still red on `origin/dev`'s recent ancestry —
pre-existing, already flagged to V1 (repo owner); not re-flagged.
