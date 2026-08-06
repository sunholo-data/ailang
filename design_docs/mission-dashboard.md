# Mission Dashboard — the 30-second control context

> **Contract**: ≤40 lines, refreshed by mission-control Gate 4 every iteration (overwrite, never
> append — history lives in the charter/log). A fresh session reads THIS + MEMORY.md and has full
> steering context. Humans steer via comments on the bookkeeping issue, never a long-lived thread.

**Updated**: 2026-08-06 ~23:15 local (iteration 155)

## Now
- **Latest release**: v0.33.0 (2026-08-04)
- **⚠ GitHub Actions MAJOR OUTAGE still live** (incident 15:22Z, `investigating`, 8h+). GitHub's own
  20:34Z update gives the mechanism: **webhooks throttled to ~15%, so pushes/PRs are not creating
  workflow runs at all.** dev CI red = `test`/`lint` `cancelled` at steps=0, **no step failed**.
  **No revert warranted.**
- **NEW LEVER: `workflow_dispatch` is webhook-INDEPENDENT and works during this outage.** Fired at
  `#608`: `total=0` → `total=1` (CI queued). Reaches 3 of 4 required contexts — `docs-gate` still
  needs a real `pull_request` context (its detector diffs against the PR base).
- **Two PRs open, BLOCKED ON CI ONLY** — both fully reviewed, neither LANDED:
  `#606` (evaluator 91/100) · `#608` (evaluator 88/100)
- **2 unpushed doc-only commits** (`945f36727`, `7c7e5e58a`) waiting on the outage.
- **Next picks**: merge `#606`+`#608` and push, then **`m-net-effect-proxy-boundary` M1** (now
  sprint-planned, 4 milestones / 12 ACs) · `#545` · Lane B1 · `#604` · `#612`
- **Loops**: v1 90min · world 4h · both armed
- **Routing**: controller opus-5 · executor `pi:…deepseek-v4-flash-0731` (3 datapoints) ·
  evaluator sonnet · planner via derive-lane (`opus fail-closed:env-pin` this iteration)
- **Designer rotation**: last-used `codex:gpt-5.6-sol` → next `claude:claude-fable-5` (not fired since iter-150)

## Parked on Mark
- **`D-1`** — the proxy-boundary sprint knowingly trades target-IP SSRF pinning on PROXIED routes
  (preserved on direct/`NO_PROXY`). Wants ratification before M1 ships. **The only live ask.**
- Low-stakes tail: pure-prng split scope · persisted cost_status · ?-op briefing

## Recently settled (don't re-ask)
- **D-6 = (A)**, **RELEASE = WAIT**, **D-7 = KEEP WEEKLY** (all Mark, attended 2026-08-06)
- Standing fast-forward authorization **YES**; the 08-03 dev reconcile has held **15 iterations**
- **codex quota dry until ~2026-08-08** — the pi lane covers executor work

## Open findings worth knowing
- **A tripwire can watch a local REPLICA of its subject.** `AC10(d)`'s `testEffectsProxyResidual`
  builds its own `&http.Transport{}` and imports **zero** ailang packages (`go list` TestImports =
  12, all stdlib; control: effects = 6 ailang). It **cannot** red when `internal/effects` changes —
  yet the charter called it "the acceptance signal already built and waiting" for 3 iterations.
  Not "a guard whose removal reds nothing" — a guard never *connected*.
- **`actions/runs?head_sha=` silently returns `total=0` for a TRUNCATED SHA.** Needs all 40 chars.
  Gate 3b recommends this endpoint for "maximum certainty"; validate it on a known-positive first.
- **Named-test `-run` gates are rc=0 with `[no tests to run]` at base** — assert a `=== RUN` count,
  never a bare exit code. `go build ./...` is **already rc=1 at base** (`cmd/wasm`, `gen/main`).
- Sprint JSON schema key is **`features`**, not `milestones` (cost me a false "malformed" reading).
