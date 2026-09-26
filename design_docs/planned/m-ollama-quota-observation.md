# M-OLLAMA-QUOTA-OBSERVATION

> The correction at the end of this document supersedes the original mandatory-metadata design.

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


## Correction approved 2026-09-07: reuse Pi's measured gauge

The attended user approved using the existing Pi quota extension. V50 in
`design_docs/implemented/v0_34_0/m-ollama-cloud-provider.md` records session usage
0.613 → 0.792 → 0.972 → 1 followed by HTTP 429. This supersedes the earlier
numerator-only premise above. Pi already applies the same fractional interpretation to
weekly usage; weekly exhaustion was not independently measured in V50. Preserve that
existing interpretation explicitly as policy, rather than claiming a new measurement.

Plan: (1) reuse Pi's 80% warning / 95% admission cutoff for either legacy window;
(2) make account reset metadata optional for basic admission, retaining strict validation
when explicitly configured and applying pacing in addition to the cutoff; (3) reconcile
CLI/Pi documentation, test boundaries/failures, independently review and deploy next-fire pins.

Acceptance: valid low usage admits with no metadata; either window at 80% warns but admits,
95% or higher blocks; missing key, missing fields, negative values and HTTP errors block;
optional metadata cannot relax the cutoff; no invented resets or daily pacing claims when
metadata is absent. Existing local Ollama exemption and guarded role fallback remain intact.
Only usage HTTP reads are used for live verification. No model inference or active-run restart.


Correction verification: all three planned steps completed. Focused quota tests, cloud/local
admission and Codex fallback regressions, full `GOFLAGS=-p=1 make test`, `make lint`,
`make check-file-sizes`, scratch build and independent review all PASS. Usage-only live
observation at 13:32 UTC: state OK, session 0.019, weekly 0.359, no metadata needed.
Launchd inspection found the existing session gauge key absent from shared secrets;
deployment adds that same credential without logging it, preserving a private backup.


Correction deployed 2026-09-07 15:34 Copenhagen: binary and all four next-fire pins
select `48c4a6e4975632e1ac3c1452ebbb55e2de52c80f`. Shared mission secrets now export
the existing gauge credential. Installed verification sourced those secrets with the
session key removed: state OK, session 0.020, weekly 0.359. No active iteration restarted.
See `design_docs/verification/mission-recovery-2026-09-07/ollama-gauge-deployment.json`
for rollback backups and hashes. Reset-aware pacing remains optional and unconfigured.

## ADDENDUM 2026-09-08 — the bucket IS now rationed, by rate rather than by position

The sentence above stays true as written: `~/.ailang/state/ollama-quota-limits.json` still does
not exist, and reset-aware pacing is still unconfigured. What changed is that this no longer
means "unrationed", and reading only the paragraph above would now give the wrong answer.

**Why the limits file cannot simply be written.** Verified against the live endpoint on
2026-09-08: `/api/usage` returns `limits.session.usage` and `limits.weekly.usage` and *nothing
else* — no capacity, and no `resets_at`. Capacity is not the blocker (V50 already established
the gauge is a fraction of the limit, so it is 1.0 by construction); the reset timestamps are,
and inventing them is forbidden by the correction below.

**What was done instead.** `evaluateOllamaQuota`'s percentage pacing needs a window POSITION,
which needs a reset. A RATE does not. D-1 says "spend at most 10% of a bucket per day", and
with a fraction gauge that is directly measurable as percentage points consumed in a trailing
24 hours — needing neither capacity nor reset, the two things the provider withholds.
Implemented in `internal/mission/ollama_rate_ration.go`; readings are banked to
`ollama-quota-observations.jsonl`, and consumption sums POSITIVE deltas so a window rollover
contributes zero instead of reading as negative spend and licensing a fresh burst.

**Why it was needed, measured.** The weekly gauge went 36.1% → 43.1% → 69.4% between
2026-09-07 17:02 and 2026-09-08 09:06 — about 2.1pp/h against a 0.42pp/h ration, with nothing
to stop it before the 95% cutoff. Ollama was the fleet's only wholly unrationed bucket.

Thin history is UNPACED and loud rather than blocking (2+ readings spanning 1h+ are required),
and the ration can only ever make the verdict stricter — it cannot launder a critical gauge
into `ok`. If reset metadata is ever OBSERVED (not invented), the original reset-aware path
above still applies and is strictly better; this addendum does not retire it.
