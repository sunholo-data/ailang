# M-OPENROUTER-BROADCAST-INGEST

**Status**: Planned
**Target**: v0.33.1
**Priority**: P0 — **Broadcast is LIVE on prod as of 2026-08-13 and every ingested trace ID is corrupted right now**
**Estimated**: ~12 hours (2 days)
**Dependencies**: None. Phase 1 and 2 are independently landable; Phase 3 is independent of both.

> **Live-state note (2026-08-13).** The OpenRouter Broadcast destination was pointed at prod
> mid-design. Data is arriving (`service.name = openrouter`, spans `LLM Generation` and
> `provider attempt N: …`) and **100% of it has a 48-char trace ID**. This resolves the one
> question the design could not answer from OpenRouter's docs: **they send OTLP/JSON**, so Phase 1
> is load-bearing, not precautionary. The corruption is **losslessly reversible** (see V12), so
> nothing is being permanently destroyed — but every ID is wrong until Phase 1 lands.

## Axiom Compliance

**Canonical reference:** [Design Axioms](/docs/references/axioms)

### Axiom Scoring

| Axiom | Score | Justification |
|-------|-------|---------------|
| A1: Determinism | +1 | A trace ID currently depends on which encoding the producer chose — protobuf gives the right ID, JSON a different one for the same input. The fix makes the identity of a span independent of its transport |
| A2: Replayability | +1 | Correct, stable trace IDs are the precondition for replaying or reconstructing a chain across systems |
| A3: Effect Legibility | 0 | No AILANG effect semantics change; this is Go infrastructure |
| A4: Explicit Authority | +1 | Converts an ambient, unauthenticated write into production state into an explicitly credentialed one |
| A5: Bounded Verification | +1 | The decode becomes locally checkable — a fixed-length hex ID either parses or is rejected, with no downstream discovery of corruption |
| A6: Safe Concurrency | 0 | No concurrency changes |
| A7: Machines First | +1 | Joinable identifiers are what make the telemetry machine-analyzable; today an agent querying `chains` gets fragmented singletons |
| A8: Minimal Syntax | 0 | No syntax changes |
| A9: Cost Visibility | +1 | Broadcast carries the provider's own per-request cost and token counts, giving a second independent instrument on spend |
| A10: Composability | 0 | Composes with the existing observatory, but adds no new composition primitive |
| A11: Structured Failure | +1 | Replaces `HTTP 200 {"partialSuccess":{}}` on corrupt input with a typed, non-200 rejection — a direct no-silent-fallback fix |
| A12: System Boundary | +1 | Makes the OpenRouter→observatory boundary explicit, authenticated, and validated rather than implicit and open |

**Net Score: +8** → **Decision: Move forward**

### Hard Violation Check

- [x] A1 (Determinism): No implicit nondeterminism introduced — removes an existing transport-dependent inconsistency
- [x] A3 (Effects): No hidden side effects
- [x] A4 (Authority): No ambient access granted — **removes** ambient write access
- [x] A7 (Machines First): Not optimizing for human convenience over machine analysis

No −1 on any axiom; net +8 clears the ≥ +2 threshold.

## Problem Statement

[OpenRouter Broadcast](https://openrouter.ai/docs/guides/features/broadcast) pushes a trace for
every API request — input/output messages, prompt/completion/total tokens, cost, timing, model,
tool usage — from OpenRouter's servers to a configured destination. One of the supported
destinations is an OpenTelemetry Collector, and we already run a public OTLP receiver.

Adopting it would give us per-request cost and token data from the **provider's** side of the wire,
independent of our own accounting, for every OpenRouter call the rig makes. That is a genuine
second instrument on a number we currently only measure first-party.

Three things block it, and one of them is actively corrupting data today.

**Current State:**

1. **The OTLP/JSON decode path corrupts trace and span IDs, silently.**
   [`internal/observatory/otlp_receiver.go:166`](../../../internal/observatory/otlp_receiver.go)
   decodes JSON bodies with `protojson.Unmarshal`, which follows the proto3 JSON mapping and treats
   `bytes` fields as base64. The [OTLP/JSON spec](https://opentelemetry.io/docs/specs/otlp/#json-protobuf-encoding)
   overrides this: `traceId` and `spanId` are **hex**. Every OTLP/JSON ingest base64-decodes a hex
   string and stores the garbage, returning `HTTP 200 {"partialSuccess":{}}`. Measured 2026-08-13
   against prod:

   ```
   sent    5b8aa5a2d2c872e8321cf37308d69df2                  (32 hex chars = 16 bytes)
   stored  e5bf1a6b96b677673cef67bcdf6d5c7f7ef7d3c77af5d7f6  (48 hex chars = 24 bytes)
   base64decode("5b8aa5a2d2c872e8321cf37308d69df2").hex() == stored   →  True
   ```

   Same defect at `otlp_receiver.go:233` (logs) and in `otlp_receiver_metrics.go`. The protobuf
   path (`otlp_receiver.go:161`) is correct. This survived because **no test exercises the decode
   path at all** — `otlp_receiver_test.go` hands `convertSpan` raw `[]byte` trace IDs directly
   (`otlp_receiver_test.go:29`) and never calls the HTTP handler or `protojson`.

2. **The OTLP routes are public and unauthenticated.**
   [`internal/server/server.go:603-605`](../../../internal/server/server.go) registers `/v1/traces`,
   `/v1/logs`, `/v1/metrics` on the main mux; `server.go:631` wraps that mux in `corsMiddleware`
   **only**. Firebase auth is per-route and covers none of them. `ailang serve` is not local-only in
   production — `cmd/ailang/server.go:31-33` flips `bindAddr` to `0.0.0.0` whenever `PORT` is set.
   Anyone who knows the URL can write spans into the production `observatory.db`, and the read APIs
   are open too.

3. **Our OpenRouter requests carry no correlation fields.** Broadcast enriches traces from three
   optional body fields — `user` (≤128 chars), `session_id` (≤256 chars, also settable via the
   `x-session-id` header), and `trace` (arbitrary key-value object). None are set: `chatRequest`
   ([`internal/ai/openrouter/types.go:16-31`](../../../internal/ai/openrouter/types.go)) has no such
   fields, and a grep across the non-test sources finds no `session_id`/`trace` key. Without them,
   broadcast spans arrive as an undifferentiated stream that cannot be joined to eval runs, chains,
   or benchmarks — which is the entire reason to ingest them.

**Impact:**

- **Defect 1 affects anything already sending OTLP/JSON**, not just this feature. Trace IDs are the
  join key for `ailang chains`; corrupted IDs mean spans that should form one trace fragment into
  singletons. The `HTTP 200` makes it invisible. This is the same vacuous-pass class the mission has
  closed repeatedly: an exit code reporting success for work that did not happen.
- **Defect 2** is pre-existing, but Broadcast is the first thing that hands the URL to a third party.
- **Defect 3** is what makes the ingested data useful rather than merely present.

## Goals

**Primary Goal:** Make the observatory a correct and authenticated OTLP sink, and make our
OpenRouter calls carry the identifiers needed to join broadcast traces to eval runs.

**Success Metrics:**

- A trace ID sent as OTLP/JSON round-trips byte-identical: `POST /v1/traces` with
  `traceId="5b8aa5a2d2c872e8321cf37308d69df2"` reads back as 32 hex chars, not 48.
- A malformed ID is **rejected with a typed error**, not accepted with `partialSuccess:{}`.
- An unauthenticated `POST /v1/traces` against a shared-secret-configured server returns 401;
  the same request with the correct header returns 200.
- ≥95% of broadcast spans from a rig eval run carry a `session_id` that matches a real
  `chains` row, joinable with a single query.

## High-Impact Decisions

| Decision | Why High Impact | Chosen By | Deadline | Change Cost |
|----------|-----------------|-----------|----------|-------------|
| Reject vs. best-effort on a malformed trace ID | Determines whether bad data can enter `observatory.db` at all. Rejecting is the no-silent-fallback answer but will drop spans from any producer we get wrong | human | design | high |
| Auth mechanism: shared-secret header vs. Cloud Run IAM vs. signed ingest token | IAM would be stronger but OpenRouter can only send static custom headers, so IAM would exclude the very producer this doc is for | human | design | high |
| ~~Whether to backfill / quarantine existing corrupted rows~~ **RESOLVED — backfill** | Was scored high on the assumption the corruption was lossy. Measured (V12): it is a **lossless, reversible** mapping, and the affected population is **38 rows in prod**. `base64encode(unhexlify(stored))` recovers the original ID exactly. This is a one-line migration, not a judgement call | agent | design | **low** |
| What `session_id` maps to (chain ID vs. eval run ID vs. benchmark+model) | Determines what the join actually buys. Chain ID is the finest grain we already persist | agent | design | med |
| Whether Phase 3 sets `session_id` via body field or `x-session-id` header | Header is one place (the existing `setAttributionHeaders`); body field is three build sites | agent | compile | low |

### Design Freeze

Before implementation begins, these must be resolved:

- [ ] **Malformed-ID policy**: reject with typed error (proposed) vs. best-effort accept.
- [ ] **Auth mechanism**: shared-secret header via env var (proposed) vs. alternative.
      **Note the live constraint**: Broadcast is already sending to prod, so whatever is chosen must
      be expressible as a static custom header OpenRouter can attach, and the destination config
      must be updated in the same change or ingest breaks.
- [x] ~~**Existing corrupted rows**~~ — **RESOLVED: backfill.** Reversible (V12), 38 rows.

## Solution Design

### Overview

Three independent phases, ordered by what blocks what. **Phase 1 must land before Broadcast is
pointed at the endpoint** — enabling ingest first would pour un-joinable data into prod and we would
be measuring our own decode bug.

### Architecture

**Components:**

1. **OTLP/JSON ID codec** — a pre-decode normalization step that rewrites `traceId`/`spanId` string
   fields from hex to the base64 `protojson` expects, or a post-decode repair. The normalization
   approach is preferred: it keeps `protojson` as the single decoder and confines the spec deviation
   to one function. Applies to traces, logs, and metrics.

2. **OTLP ingest auth** — a middleware on the three OTLP routes only (not the whole mux; the
   dashboard UI and existing read APIs must keep working unchanged). Shared secret from an env var,
   compared with `constantTimeEqual`. **Absent env var = auth disabled**, preserving local
   `ailang serve` and rig workflows that post to `localhost:1957` with no credential.

3. **OpenRouter correlation fields** — `user`, `session_id`, `trace` plumbed from the eval harness
   through `ai.Request` into the three OpenRouter request-build sites.

### Implementation Plan

**Phase 1: Fix OTLP/JSON ID decoding** (~4 hours)
- [ ] Write the failing test FIRST: POST an OTLP/JSON body with a known hex `traceId` to
      `handleTraces` via `httptest`, assert the stored ID is the same 16 bytes. This test does not
      exist in any form today and is the reason the bug shipped.
- [ ] Add hex↔base64 normalization for `traceId`/`spanId` ahead of `protojson.Unmarshal`.
- [ ] Apply at all three sites: `otlp_receiver.go:166` (traces), `otlp_receiver.go:233` (logs), and
      the metrics equivalent in `otlp_receiver_metrics.go`.
- [ ] Reject malformed IDs with a typed error and a non-200 status (pending Design Freeze).
- [ ] Assert the protobuf path (`otlp_receiver.go:161`) is unchanged — it is already correct and
      must not regress.
- [ ] **Backfill the 38 existing prod rows**: `trace_id = base64encode(unhexlify(trace_id))` for
      every row whose ID is 48 chars, same for `span_id`. Lossless per V12. Guard the migration on
      the length so it is idempotent and cannot touch already-correct rows.

**Phase 2: Authenticate OTLP ingest** (~3 hours)
- [ ] Add `AILANG_OTLP_INGEST_TOKEN` env var; when unset, auth is disabled (documented explicitly).
- [ ] Middleware on the three OTLP routes only, using `constantTimeEqual` for the comparison.
- [ ] Tests: no header → 401; wrong header → 401; correct header → 200; unset env → 200 with no
      header (the local-dev path).
- [ ] Confirm the dashboard UI and `/api/observatory/*` reads are untouched.

**Phase 3: OpenRouter correlation fields** (~5 hours)
- [ ] Add `User`, `SessionID`, `Trace` to `chatRequest` (`types.go:16-31`) with `omitempty`, so a
      request with none set produces **byte-identical** wire output to today.
- [ ] Add the corresponding fields to `ai.Request` (`internal/ai/provider.go:32`).
- [ ] Wire all three build sites: `chat.go:44` (struct literal), `step.go:187` (reuse the existing
      `marshalStepBodyWithExtras` splice helper — do not add a second mechanism), `streamstep.go:118`.
- [ ] Populate from the eval harness at `internal/eval_harness/ai_provider.go:81`.
- [ ] Enforce the length caps (`user` ≤128, `session_id` ≤256) at construction, failing loudly
      rather than silently truncating.

### Files to Modify/Create

**New files:**
- `internal/observatory/migrate_v18.go` — idempotent, length-guarded backfill of the 38 corrupted rows (`trace_id`/`span_id` where length is 48), ~60 LOC

**Modified files:**
- `internal/observatory/otlp_receiver.go` — hex/base64 ID normalization at the two decode sites; typed rejection, ~80 LOC
- `internal/observatory/otlp_receiver_metrics.go` — same normalization for the metrics path, ~20 LOC
- `internal/observatory/otlp_receiver_test.go` — first tests to exercise the HTTP + `protojson` decode path, ~150 LOC
- `internal/server/server.go` — OTLP-route-scoped auth middleware near the registration at `:603-605`, ~40 LOC
- `internal/ai/openrouter/types.go` — three new `omitempty` fields on `chatRequest`, ~15 LOC
- `internal/ai/openrouter/chat.go` — populate at the struct literal (`:44`), ~15 LOC
- `internal/ai/openrouter/step.go` — populate via the existing `marshalStepBodyWithExtras` splice (`:186`), ~20 LOC
- `internal/ai/openrouter/streamstep.go` — populate at the stream build site (`:118`), ~20 LOC
- `internal/ai/provider.go` — `User`/`SessionID`/`Trace` on `ai.Request`, ~20 LOC
- `internal/eval_harness/ai_provider.go` — populate correlation fields from chain/run context, ~30 LOC
- `.claude/rules/cloud-endpoints.md` — remove the KNOWN DEFECT section once Phase 1 lands, ~10 LOC

## Examples

### Example 1: OTLP/JSON trace ID round-trip

**Before:**
```bash
$ curl -sX POST -H "Content-Type: application/json" \
    -d '{"resourceSpans":[{"scopeSpans":[{"spans":[
        {"traceId":"5b8aa5a2d2c872e8321cf37308d69df2","spanId":"051581bf3cb55c13",
         "name":"probe.span","kind":1,"startTimeUnixNano":"...","endTimeUnixNano":"..."}]}]}]}' \
    https://dashboard.ailang.sunholo.com/v1/traces
{"partialSuccess":{}}                      # HTTP 200 — looks fine

$ curl -s ".../api/observatory/spans?limit=1" | jq -r .[0].trace_id
e5bf1a6b96b677673cef67bcdf6d5c7f7ef7d3c77af5d7f6     # 48 chars — silently corrupted
```

**After:**
```bash
$ curl -s ".../api/observatory/spans?limit=1" | jq -r .[0].trace_id
5b8aa5a2d2c872e8321cf37308d69df2                     # 32 chars — byte-identical

# and a malformed ID no longer returns a cheerful 200:
$ curl -sX POST ... -d '{"...":"traceId":"not-hex"...}' .../v1/traces -w " HTTP %{http_code}"
{"error":"invalid traceId: expected 32 hex chars, got 7"} HTTP 400
```

### Example 2: Joining a broadcast trace to an eval run

**Before** — the broadcast span exists but nothing links it to the run that caused it:
```json
{"model": "anthropic/claude-sonnet-4.5", "messages": [...]}
```

**After:**
```json
{"model": "anthropic/claude-sonnet-4.5", "messages": [...],
 "session_id": "b1df1f0e-3cfe-4783-8712-9c5f73fe5a50",
 "trace": {"trace_name": "eval:fizzbuzz", "benchmark": "fizzbuzz", "tier": "core"}}
```
which makes OpenRouter's own cost/token numbers joinable against our first-party banked row for the
same chain — a second instrument on a figure we currently measure only once.

## Success Criteria

- [ ] A hex `traceId` posted as OTLP/JSON reads back byte-identical (test asserts 16 bytes, not 24)
- [ ] A malformed ID returns non-200 with a typed error, never `partialSuccess:{}`
- [ ] The protobuf ingest path is provably unchanged (existing tests green, plus a round-trip test)
- [ ] With `AILANG_OTLP_INGEST_TOKEN` set: no/wrong header → 401, correct header → 200
- [ ] With it unset: local `ailang serve` ingest works with no credential (no rig workflow breaks)
- [ ] A request with no correlation fields set produces byte-identical wire output to today
- [ ] ≥95% of broadcast spans from one rig eval run join to a real `chains` row by `session_id`
- [ ] All tests passing
- [ ] Documentation updated (`.claude/rules/cloud-endpoints.md` defect section removed)

## Testing Strategy

**Unit tests:**
- Hex↔base64 normalization: valid 16-byte trace ID, valid 8-byte span ID, wrong length, non-hex
  chars, empty, and — importantly — a value that is *valid base64 AND valid hex*, to pin the
  precedence.
- Length-cap enforcement on `user` (128) and `session_id` (256): at-cap passes, over-cap fails loudly.

**Integration tests:**
- `httptest` POST of a full OTLP/JSON payload through `handleTraces` into a temp DB, asserting the
  stored ID. **This is the coverage gap that let the bug ship** — the existing tests call
  `convertSpan` with raw `[]byte` (`otlp_receiver_test.go:29`) and never touch `protojson`.
- The same payload as protobuf, asserting identical stored output. JSON and protobuf must agree.
- Auth middleware across the four cases above.
- Golden-body test: an `ai.Request` with no correlation fields marshals byte-identically to the
  current output, for all three OpenRouter build sites.

**Manual testing:**
- ~~Enable Broadcast against dev first and check the trace ID length to determine JSON vs protobuf.~~
  **Already answered** — Broadcast went live on prod 2026-08-13 and 38/38 spans came back with
  48-char IDs (V11). OTLP/JSON, confirmed.
- After Phase 1 lands, read the next arriving broadcast span and assert a **32-char** trace ID.
  This is a live end-to-end check against a real third-party producer, which is stronger evidence
  than any fixture.
- Run the backfill migration on the 38 existing rows and spot-check that a recovered ID matches
  what OpenRouter's own dashboard shows for the same request.
- After Phase 2, confirm Broadcast still ingests once the shared-secret header is configured on the
  OpenRouter destination — **this is the step that can break live ingest**, so do it with the
  before/after span count in hand.

## Deferred Decisions

- **Which chain/run identifier becomes `session_id`** — agent may choose, guided by what
  `internal/eval_harness` already has in hand at the call site. Chain ID is the proposal.
- **What keys go in the `trace` object** — agent may choose; benchmark ID and tier are the obvious
  starting pair.
- **Sampling rate for the initial Broadcast rollout** — agent may choose, start ≤5%.
- **Whether to also set `user`** — agent may choose; it may be redundant with `session_id` for our
  single-tenant case.

## Non-Goals

- **Ingesting Broadcast into anything other than the observatory** — Langfuse, BigQuery and a
  dozen others are supported destinations and may well be better long-term homes for this data, but
  fan-out is a separate decision from making our own sink correct.
- **Reworking the observatory schema for LLM-specific span attributes** — the existing token/cost
  extraction (`otlp_receiver.go:334-359`) is assumed adequate until measured otherwise.
- **Authenticating the observatory read APIs** — real, and adjacent, but a wider blast radius (the
  dashboard UI depends on them) and should not ride along with an ingest fix.
- **Changing first-party cost accounting to trust OpenRouter's numbers** — this doc adds a second
  instrument; deciding which one is authoritative is a separate question with its own evidence bar.

## Timeline

**Day 1** (~7 hours):
- Phase 1: decode fix, failing-test-first, all three sites
- Phase 2: OTLP ingest auth

**Day 2** (~5 hours):
- Phase 3: correlation fields across the three build sites + harness plumbing
- Manual verification against dev, then prod

**Total: ~12 hours across 2 days**

## Risks & Mitigations

| Risk | Impact | Mitigation |
|------|--------|-----------|
| An existing OTLP/JSON producer depends on the current (broken) IDs | Med | Nothing *should* — corrupted IDs are useless by construction. Grep for internal JSON producers before landing; the rig uses the Go OTEL SDK, which sends protobuf |
| Rejecting malformed IDs drops spans from a producer we mis-diagnose | Med | Land the fix with rejection behind a log line first, measure the reject rate on dev for one eval rotation, then enforce |
| ~~Existing 48-char rows make the join key ambiguous forever~~ | ~~Med~~ **Resolved** | Reversible (V12), 38 rows (V13), no correct-ID rows to disambiguate against. Idempotent length-guarded migration |
| Auth env var unset in the Cloud Run deploy → ingest silently stays open | High | Deploy-time assertion, and the success criteria test the 401 path against the deployed service, not just locally |
| ~~OpenRouter's OTEL destination sends a shape we do not parse~~ | ~~Med~~ **Resolved** | Measured live: OTLP/JSON, parsed, spans land. Only the IDs are wrong (V11) |
| **Phase 2 breaks live ingest** — Broadcast is already flowing to prod, so adding auth without updating the OpenRouter destination config silently drops every span | **High** | Configure the header on the OpenRouter side in the same change window; verify with a before/after span count, not by assuming a 200 |
| Prod accumulates more corrupted rows while Phase 1 is unlanded | Low | Backfill is idempotent and length-guarded, so it covers whatever arrives before the fix. No urgency to disable Broadcast |
| Correlation fields change wire bytes for existing calls | High | `omitempty` + a golden-body test asserting byte-identical output when unset, at all three build sites |

## Related Documents

<!-- Auto-populated by Ollama neural search on "openrouter broadcast ingest" -->

**Implemented (may inform design):**
- [design_docs/implemented/v0_18_9/m-ai-openrouter-app-attribution.md](../../implemented/v0_18_9/m-ai-openrouter-app-attribution.md) (0.42) — established the header-precedence pattern (per-request → env → default) that Phase 3 should follow for correlation fields
- [design_docs/implemented/v0_7_0/observatory-architecture.md](../../implemented/v0_7_0/observatory-architecture.md) (0.40) — the sink this doc is fixing
- [design_docs/implemented/v0_18_9/m-ai-openrouter-reasoning-field.md](../../implemented/v0_18_9/m-ai-openrouter-reasoning-field.md) (0.39) — prior art for adding an optional field across all three OpenRouter build sites without changing flag-off wire bytes

**Planned (check for overlap):**
- [design_docs/planned/v0_29_0/m-eval-openrouter-baseline-rotation.md](../v0_29_0/m-eval-openrouter-baseline-rotation.md) (0.39) — distinct: that doc is about which models to rotate, this one is about ingesting telemetry from calls to them
- [design_docs/planned/m-eval-data-hosting-decouple.md](../m-eval-data-hosting-decouple.md) (0.34) — source of the `dashboard.ailang.sunholo.com` → prod-only mapping noted below
- [design_docs/planned/m-dynamic-data-runtime-plane.md](../m-dynamic-data-runtime-plane.md) (0.34) — distinct: benchmark data serving, not telemetry ingest

All below the 0.45 warn threshold; no duplicate-coverage rejection applies.

## Verification Log

Every claim below was checked against the code or the live service on **2026-08-13**, not inferred.
Negative-existence claims carry their own row per the design-doc-creator hard gate.

| # | Claim | How verified | Result |
|---|-------|--------------|--------|
| V1 | **No auth exists on the OTLP routes** | `grep -n "Authorization\|Bearer\|authMiddleware\|requireAuth" internal/observatory/otlp_receiver*.go` | Empty — **confirmed absent** |
| V2 | Only CORS wraps the mux | Read `internal/server/server.go:631` → `handler := s.corsMiddleware(mux)` | Confirmed |
| V3 | **No correlation fields are set on OpenRouter requests** | `grep -rn '"user"\|session_id\|"trace"' internal/ai/openrouter/*.go` (non-test) | Only `Role: "user"` at `chat.go:38` — **confirmed absent** |
| V4 | A body-extension helper already exists (reuse, don't add a second) | Read `internal/ai/openrouter/step.go:186` → `marshalStepBodyWithExtras` | **Exists** — Phase 3 reuses it |
| V5 | **No test exercises the OTLP HTTP/JSON decode path** | `grep -n "httptest\|handleTraces\|protojson\|application/json" internal/observatory/otlp_receiver_test.go` | Empty; tests pass raw `[]byte` at `otlp_receiver_test.go:29` — **confirmed absent**, and is why the bug shipped |
| V6 | OTLP/JSON base64-decodes hex IDs | Live POST to prod, then read back; `base64decode(sent).hex() == stored` → `True` | Confirmed, exact byte match |
| V7 | `/v1/traces` is publicly reachable | `POST` to prod and dev → **200**; control `POST /v1/bogus` → **404** | Confirmed with firing control |
| V8 | `bindAddr` is not localhost in prod | Read `cmd/ailang/server.go:26` (default `localhost`) and `:31-33` (`PORT` set → `0.0.0.0`) | Confirmed |
| V9 | `dashboard.ailang.sunholo.com` maps to **prod**, and dev has no DNS name | `curl` both; `dig +short` on three candidate dev names | Confirmed — all three dev names return no record |
| V10 | Broadcast is dashboard-only, no per-request opt-in | OpenRouter docs, `/docs/guides/features/broadcast` | Confirmed — account-level toggle, filter by key + sampling only |
| V11 | **Whether OpenRouter's OTEL destination sends JSON or protobuf** | Broadcast went live on prod mid-design; read 38 spans back — `service.name = openrouter`, **38/38 with 48-char trace IDs** | **RESOLVED: OTLP/JSON.** Phase 1 is required, not precautionary |
| V12 | **Whether the corruption is lossy** (decides backfill vs. quarantine) | `base64encode(unhexlify(stored))` on live rows → 32-char strings that are valid hex, e.g. `6bb6b86b…a6bd` → `a7a4a206034b2f4fa1c269a592b44aa9` | **Lossless and reversible.** base64 decoding is injective on fixed-length input, so the original ID is always recoverable. Backfill is a pure function |
| V13 | Size of the affected population in prod | `GET /api/observatory/spans?limit=2000` → 38 rows, length distribution `{48: 38}` | 38 rows total (37 broadcast + 1 probe). **No correct-ID spans exist in prod**, so there is no mixed-ID period to disambiguate |

## References

- [OpenRouter Broadcast](https://openrouter.ai/docs/guides/features/broadcast) — feature docs
- [OTLP/JSON encoding spec](https://opentelemetry.io/docs/specs/otlp/#json-protobuf-encoding) — the hex-vs-base64 override that Phase 1 implements
- [.claude/rules/cloud-endpoints.md](../../../.claude/rules/cloud-endpoints.md) — deployed-endpoint facts, committed `8773c8976`
- [Design Axioms](/docs/references/axioms) — the 12 non-negotiable principles

## Future Work

- **Fan out Broadcast to a second destination** (Langfuse or BigQuery) and compare against the
  observatory ingest — a cross-check on our own sink.
- **Reconcile OpenRouter-reported cost against first-party banked cost** per chain, now that they
  are joinable. A systematic divergence would be a finding in its own right.
- **Authenticate the observatory read APIs** — the adjacent half of Phase 2, deliberately out of
  scope here.

---

**Document created**: 2026-08-13
**Last updated**: 2026-08-13
