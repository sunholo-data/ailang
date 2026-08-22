# Mission Dashboard — V1

> Snapshot only; history lives in `v1-mission.md` + `v1-mission-log.md`. Written: **iter 252, 2026-08-22**.

## Where we are

- **Latest release** v0.33.1; the cons-cells programme targets v0.35.0 (LC-1 landed in v0.34.0's window).
- **dev CI** GREEN at `8e3928a08` — 16 checks, zero not-green (control `total_count`=16 fires).
- **Decision ledger** 23 rows, **ZERO OPEN**. Nothing parked on Mark.
- Main checkout **0 ahead / 0 behind** origin, holding since iter 249 — records write in place.

## In flight / next

1. **`m-list-accessor-api` (LC-2)** — **sprint plan LANDED** (iter 252): 6 milestones, ~1,660 Go
   LOC, **4.5 d** (+0.5 over the roadmap's 3–4 band, surfaced not compressed). **Next:
   sprint-executor on M1+M2**, which are independently committable.
   **⚠ Settle DEFECT-1 first**: `packages.Config.Tests` defaults to `false` while 380 of 903
   `.Elements` sites are in `_test.go` — the census denominator every later lane is measured
   against moves ~380 sites on an unstated flag.
2. **`m-array-show-diverges-run-vs-compile` M4** — deferred two iterations now (to the directive,
   then to LC-2's plan). CHANGELOG, doc move to `implemented/v0_34/`, VL-9 correction.
3. Then LC-3a/3b/3c (mechanical, parallelizable) → LC-4 (the swap, riskiest) → LC-5 (tuning).

## New this iteration

- **`go test -run` exits 0 on an EMPTY match set** — confirmed first-party (rc=0, 0 `=== RUN`
  lines, identical exit code to a real pass). V1 carries **54** such invocations across **16**
  files in `design_docs/planned`. Filed `[world-DEMAND]` for a repo-wide sweep; LC-2's plan is
  pre-emptively floored (all 7 ACs assert `N == <literal>` RUN lines, `N == 0` → instrument failure).
- **Two hypotheses refuted by measurement**: the codex planner lane is *not* silently bypassed
  (0/140 docs reach it because the allowlist is mission-infra by design), and LC-4's wasm exposure
  *does* get a PR-time signal (`internal/` and `cmd/` are both in `.github/docs-build-paths.txt`).

## Loop cadence + routing

Controller `claude:claude-opus-5` · planner **opus** this iteration (`derive-planner-lane.sh` →
`opus fail-closed:planner-lane-field-invalid`, used verbatim) · executor `codex:gpt-5.6-sol` ·
evaluator `sonnet`. Designer rotation still degraded: gemini is read-only (`CapRemoteSandbox`),
`codex` collides with quorum reviewer `gpt5-6-sol` — Fable is the only clean authoring lane.
Every wait bounded; worktrees are siblings of the repo, never `/tmp`.

## Cost posture · Parked on Mark

Iteration 252 metered **$0.00** of $5 — every lane a quota bucket (opus ×2). **Parked on Mark:
nothing.** One process gap named, not fixed: iter-251's quorum artifacts never persisted to
`.ailang/state/mission-quorum/`, so a later pick could misread absence as "never reviewed".
