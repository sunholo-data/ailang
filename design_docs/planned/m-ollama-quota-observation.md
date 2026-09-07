# M-OLLAMA-QUOTA-OBSERVATION

Attended scope authorized 2026-09-07: add Ollama accounting and role admission, extending
M-QUOTA-RATIONING-ROUTING and M-CODEX-QUOTA-OBSERVATION. Use the same design/sprint/evaluation flow.

## Verified problem

GET https://ollama.com/api/usage succeeds with OLLAMA_API_KEY; the live legacy response contains
limits.session.usage=0.011 and limits.weekly.usage=0.356, without capacities or reset instants.
Do not reinterpret these as percentages. Existing code probes Pi role lanes without rationing.
Official 2026-08-31 pricing announcement distinguishes legacy session/weekly plans from new
monthly-credit plans: https://ollama.com/blog/transparent-pricing. No plan upgrade authorized.

## Design and milestones

M1: bounded usage-only HTTP reader (5s deadline, 1MiB response cap, no redirects, fixed HTTPS
Ollama endpoint, no keys/bodies in errors), strict finite nonnegative usage fields. Missing key,
HTTP failure, unsupported schema or missing account limits produces explicit blocked/unknown.
No model inference and no invented token denominator. Return raw usage in CLI JSON and text.

For legacy pacing, optional operator-verified account metadata lives at
~/.ailang/state/ollama-quota-limits.json: credential_sha256 (bind the gauge key), source
(nonempty provenance), session_capacity, weekly_capacity, session_resets_at, weekly_resets_at.
Capacities must use the SAME units as provider usage; times are provider/UI-reported future
reset instants bounded by 5h/7d. Never extrapolate a passed reset. The provider numerator plus
verified denominator yields percentages; weekly pacing reuses 10%/day, session exhaustion binds.
No metadata is written or guessed by the reader. Monthly credit schema remains unknown/blocked.
Await requested account UI metadata before configuring legacy capacities for this installation.

M2: quota CLI reports Ollama even with no ledger row; --over preserves provider blocks if token
ledger loading fails. Apply the cloud quota gate before all Pi Ollama-cloud role probes and
controller probes. Local Ollama model routes (no :cloud or -cloud suffix) remain outside the
cloud subscription guard. Existing fallback chains and metered budgets stay in force.

M3: offline HTTP/parser/CLI/extracted-driver regressions, full tests and lint, independent review,
real usage-only replay and reversible deployment to next-fire pins. No active job restart.

## Limits

Read-only provider fetch runs on quota reads (at most one per fire through the driver cache).
Missing launchd credentials blocks cloud admission; it does not prove exhaustion. This is new
admission, not cancellation or reservation for an in-flight role. UI metadata can be stale and
must be re-confirmed after reset/account change; never treat a stale gauge as available quota.

## Verification and deployment contract

Independent review PASS after closing controller-pin, file-pin, one-shot executor and quota-command
outage bypasses. Extracted-driver regression executes actual Pi role fallback; local-model and
OpenRouter positive controls prove the gate does not simply block everything. A blocked one-shot
is preserved for a later eligible fire, while the already-checked executor remains selected.

The quota CLI call itself is bounded to 15 seconds. Combined diagnostics are never interpreted
as bucket names: only exact known bucket lines enter admission. A command outage blocks Codex
and Ollama Cloud while retaining the pre-existing policy for other providers.

The API credential is not proof of the daemon's separate device-account identity. Operator
metadata must be verified against that same account before enabling admission. Unsuffixed
Ollama model names denote local models by the established fleet routing convention.

Do not populate real capacities/reset times until Mark supplies/verifies the requested account
page metadata. Deployment without that metadata intentionally blocks new Ollama Cloud admission
and advances configured fallbacks; it must not be reported as measured exhaustion or full pacing.

Verification completed: `GOFLAGS=-p=1 make test`, `make lint`, scratch build, focused quota
regressions and race tests passed. Existing routing suite passed 84 assertions; controller-chain
and Codex admission regressions passed. Full-suite correction: the live doctor test now requires
actual differing keys, rather than assuming any Docs drift must be the old planner-allowlist bug.
Independent review approved both the guard and that test correction. No inference was used in
verification; authenticated live reads targeted only Ollama's usage endpoint.

Deployment 2026-09-07 15:19 Copenhagen: installed binary and all four next-fire pins select
`8301960c9264d2774cce4ca2d3ffea3d5d6eb6ff`. Live usage-only replay: session 0.006, weekly 0.357,
state unknown because verified capacities/reset times are absent. No metadata fabricated.
No active iterations restarted; live next-fire fallback remains to be observed. Rollback hashes
and backups are in `design_docs/verification/mission-recovery-2026-09-07/ollama-quota-deployment.json`.
Account metadata remains an outstanding configuration dependency, not completed allowance pacing.
