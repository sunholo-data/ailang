# M-CODEX-QUOTA-OBSERVATION

Status: attended implementation, 2026-09-07. Extends ratified M-QUOTA-RATIONING-ROUTING M3.
Authorization: Mark asked to make the guard account for Codex after observing rapid depletion;
the mission-runtime workstream and execution are already authorized in this session.

## Problem and evidence

`mission quota` reads only explicitly posted quota_tokens. Live ledger has two OpenCode stages
and no Codex, while local Codex provider receipts show weekly usage rising 4% to 17%.
Token capacity is unknown; deriving it from raw/cached tokens would invent a denominator.

## Contract

Read provider-reported `event_msg/token_count/rate_limits` from local Codex JSONL records,
without inference calls, credentials, prompts in output, or writes to the token journal.
Use the latest valid Codex snapshot, account-wide (interactive and mission usage share capacity).
Identify windows by reported duration, not primary/secondary position. Pace weekly usage at
10 percentage points/day from its reported reset/window; minimum first-day allowance 10%.
Track short-window exhaustion separately. Percentages are not tokens and are not summed.
Provider observations are a separate field in quota JSON, preserving the token ledger schema.
Deduplicate `--over` bucket output. Missing, stale (>15 minutes), malformed or expired Codex
observations explicitly block new Codex routing until refreshed; never silently report zero.
This conservative Codex-only admission rule replaces the old unknown-capacity fail-open behavior.
A fresh ordinary attended Codex call refreshes records; the quota command itself spends nothing.

Bound local scanning to eight UTC date directories, regular non-symlink rollout JSONL files,
128 most recently modified candidates and the last 2 MiB each. Ignore truncated boundary lines.
Accept a bounded scan only when the selected observation is newer than every omitted file's modification time; otherwise report unknown. Report scan errors explicitly. Preserve provenance
(timestamp, basename, observed percent, duration, reset), no prompt bodies or secret values.
The selected record is local account evidence, not cryptographic account identity. Switching
accounts without separate CODEX_HOME roots remains a limitation; invalidate ambiguous snapshots.

## Sprint / acceptance

M1: parser and percentage verdict (~200 LOC), fixtures proving reversed window order, legitimate
zero, invalid/missing values, expiry, staleness, duplicate snapshots and bounded file reads.
M2: wire CLI and help (~100 LOC), test bucket filtering and one Codex routing decision, replay
real local data with a scratch binary. No model calls for verification.
M2b: route the existing planner/executor Codex preflight through the same quota-gated probe;
offline extracted-driver test must prove zero inference and configured fallback for both roles.

M3: independent evaluation and rollout evidence. Keep running iterations intact; report exactly
which installed driver/binary consumes the fix before claiming live enforcement.

## Risks and limits

This supplies provider usage accounting and admission. It does not reconstruct per-mission
historical token attribution or stop an already-running role. Other providers retain existing
policy. Reservation/concurrency enforcement remains separate mission-runtime work.

## Verification evidence (2026-09-07)

- Focused quota regressions passed, including corrupted token-ledger admission and malformed-record recovery.
- Independent local engineering review found no remaining blockers in the provider observation/CLI scope after three rounds; this is not a cross-provider quorum.
- Live read-only replay: 19% weekly usage, 10% allowed, state `over`; `--over` prints `codex` without an inference call.
- Extracted real-driver admission test proves blocked probes make zero calls and planner/executor select configured fallbacks.
- Existing routing suite: 84 assertions pass; controller-chain suite passes. Both existing suites emit missing-stub warnings; do not represent them as clean stderr.
- Lint: zero issues. Initial full make test failed four subprocess-startup cases across Codex executor, Pi integration and SMT. Serial full-suite retry (`GOFLAGS=-p=1 make test`) passed. Focused race tests also passed.
- Installed binary and saved mission pins have not yet been changed for this increment.
