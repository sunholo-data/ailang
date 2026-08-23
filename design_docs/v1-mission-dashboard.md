# Mission Dashboard — V1

*Snapshot, overwritten every iteration. History lives in `v1-mission.md` (STATUS) + `v1-mission-log.md`.*

**Iteration 260 · 2026-08-23 · dev green (16 checks, 0 not-green @ `a201237ca`)**

## In flight
- **`#764` protocol-only `serveapi` module** — design doc written, **two full quorum rounds
  (2/2 present both times), all four objections answered**. **PARKED on `D-35` alone.**
  Reproduced first-party: `serveapi` links **479** non-stdlib packages, `internal/apiserver`
  **478** — a 201-line facade adds one package over the whole server runtime. The four files it
  actually needs are stdlib-only (bar the MCP SDK), so the fix is a leaf package with a
  **stdlib-only closure**, CI-gated against regression.
  This is **Ailang World's SOLE remaining M4 blocker** — its value gate.

## Next
1. **`D-35` answered → straight to sprint-planner.** Nothing else gates it.
2. `m-verify-bounded-unrolling-false-counterexample` — routable now if `D-35` stays open.
3. `m-benchmark-ensures-coverage` — 4 of 5 candidates blocked behind (2).

## Parked on Mark (4 open — see `DECISIONS FOR MARK` on #745)
- **`D-35`** *(new, P1)* — module boundary for `serveapi/protocol`: **(a)** plain package in the
  main module, **(b)** nested module, **(c)** separate repo. **Recommend (a)** — your `D-34`
  already delivers via a v0.34.0 tag and World pins releases. One word unblocks World.
- **`D-30`** — harness↔`ai-check` PATH version coupling. Blocking TWO docs independently.
- **`D-31`** — designer rotation has ONE usable authoring lane (instance 7 this iteration).
- **`D-32`** — should `inconclusive` be exempted from the effective KPI arm, as `not_applicable`
  is under your `D-29` ruling? Only thing that could move `$0.7778`.
- *Bookkeeping:* your attended rulings were re-filed as **`D-33`** (cross-mission priority) and
  **`D-34`** (cut v0.34.0 when `#764` lands) — those labels collided with two already-open rows.
  No ruling changed; only the label moved.

## Loop
- Cadence: launchd, ~90 min. Controller opus · designer **rotation collapsed to fable** (`D-31`)
  · planner/executor `codex:gpt-5.6-sol` · evaluator sonnet.
- Metered **$0.2231** of $5 this iteration (quorum only). Quota: opus, fable ×2 (diet-compliant).
- Baseline KPI `cost_per_verified_success` = **$0.7778187072** (strict) / **$0.2121** (effective,
  per `D-29`). Unchanged this iteration — no code shipped.
