# Mission Dashboard — the 30-second control context

> **Contract**: ≤40 lines, refreshed by mission-control Gate 4 every iteration (overwrite, never
> append — history lives in the charter/log). A fresh session reads THIS + MEMORY.md and has full
> steering context. Humans steer via comments on the bookkeeping issue, never a long-lived thread.

**Updated**: 2026-08-06 ~07:00 local (iteration 150)

## Now
- **Latest release**: v0.33.0 (2026-08-04) — recorded-stream S1 + #498 Lane B M1/M2
- **`m-net-effect-proxy-boundary` (D5 Option B) has a DESIGN DOC — and is PARKED on Mark.**
  662 lines, 19 verification rows, designer codex `gpt-5.6-sol`. Quorum BLOCKED ×2.
  R1 caught a genuine TOCTOU defect (target IP resolved in two places) — fixed. R2's surviving
  objection is a **scope call, not a defect**: how durable must the completeness gate be?
- **Next (if D-6 goes unanswered)**: **`#498` Lane B M3** — final Lane B milestone, then the
  release ask goes to Mark. Also newly queued: orphaned PR **`#545`** (cost provenance).
- **Loops**: v1 90min · world 4h · both armed
- **Routing**: controller opus-5 · executor codex `gpt-5.6-sol` · evaluator sonnet · planner via derive-lane
- **Designer rotation**: last-used `codex:gpt-5.6-sol` → next `claude:claude-fable-5`

## Parked on Mark
- **D-6 (one word)**: M4 completeness gate — **(A)** grep gate now, AST analyzer as a follow-up,
  sprint stays 3d; **(B)** `go/packages` AST analyzer in-sprint, 3d→4d. **Measured input: all five
  constructions the analyzer would catch are ZERO at HEAD**, so (A) is sufficient for *present*
  correctness; the argument for (B) is durability against future escapes.
- **Worth ratifying alongside**: D-1 trades target-IP SSRF pinning on **proxied** requests
  (preserved on direct/`NO_PROXY`; doc is explicit, never claims equivalence).
- Low-stakes tail: pure-prng split scope · persisted cost_status · pipe-operator · ?-op briefing

## Recently settled (don't re-ask)
- **D5 = Option A**; Option B is the doc above · **#588 closed** (M2 gated the *subtest*, not the fn)
- **#532 closed as SUPERSEDED** · **standing fast-forward** ratified (held a 14th iteration)

## Known-deferred (measured, not forgotten)
- **Bare `&http.Client{}` is ALREADY proxy-aware** — Go's `DefaultTransport` sets
  `Proxy: ProxyFromEnvironment` (`transport.go:46-48`, go1.26.5). Only hand-built nil-Proxy
  transports escape, which is *why* the residual is exactly 7 sites. Derivation, not assertion.
- **`go mod download all` writes to the TRACKED go.sum** — prefetch must precede binary building.
- **`-shellcheck='-e SC2086'` DISABLES shellcheck** (flag takes an executable path). **`#598`**
  pid-file race: 0/12 on a stress arm.

## Quota posture (week of 2026-08-03) · Bookkeeping
- **Metered**: iter-150 spent **$0.1785** of the $5/iteration ceiling (quorum ×2 only; codex is
  OAuth-subscription, so its designer run is a quota bucket, not dollars)
- Issue: `~/.ailang/state/mission-gh-issue` (rotates Mondays) — Mark's comments there = directives
- Full state: `v1-mission.md` (charter) · `v1-mission-log.md` (history) · this file = snapshot only
