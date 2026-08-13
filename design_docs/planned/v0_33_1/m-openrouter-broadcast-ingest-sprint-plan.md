# Sprint Plan: M-OPENROUTER-BROADCAST-INGEST

**Design doc**: [m-openrouter-broadcast-ingest.md](m-openrouter-broadcast-ingest.md)
**Sprint ID**: `M-OPENROUTER-BROADCAST-INGEST`
**Created**: 2026-08-13
**Duration**: ~1.5 days (~12 hours)
**Risk**: Medium — one milestone touches a **live production ingest path** carrying real third-party data
**Total LOC estimate**: ~470 (implementation ~250, tests ~220)

## Summary

**Goal**: Make the observatory a correct OTLP sink before more corrupted data accumulates, then make
OpenRouter broadcast traces joinable to eval runs.

**Why now**: Broadcast went live on prod 2026-08-13 mid-design. 38/38 ingested spans carry a 48-char
trace ID instead of 32 — every one wrong, all of them returning `HTTP 200 {"partialSuccess":{}}`.
The corruption is lossless (V12) so nothing is being destroyed, but the join key is unusable until
M1 lands.

## Current Status Analysis

**Velocity (last 7 days)**: 126 Go files, +10,461/−671 LOC across three concurrent mission loops.
Recent comparable sprint (`M-Z3-HARD-TIMEOUT`) used `target_loc_per_day: 550` for a 330-LOC,
0.6-day sprint. This sprint is ~470 LOC → **~1.5 days at a conservative 400 LOC/day**, with the extra
half-day allocated to live-prod verification rather than code.

**Design Freeze**: all three items resolved (2026-08-13). **No pause point for the executor.**

**Dependencies**: none external. M1 → M2 (backfill needs the codec) and M1 → M4 (live verification
needs the fix deployed). M3 is fully independent and could run in parallel.

## Milestones

### M1_OTLP_JSON_ID_CODEC (~180 LOC, ~4h)

Fix the hex/base64 decode defect at all three sites, **test-first**.

**Tasks:**
- [ ] Write the failing integration test FIRST: `httptest` POST of an OTLP/JSON body with a known
      hex `traceId` through `handleTraces`, asserting the stored ID is the same 16 bytes. Capture
      the red transcript verbatim before touching the receiver.
- [ ] Add a `normalizeOTLPJSONIDs` pre-decode pass rewriting `traceId`/`spanId` string fields from
      hex to the base64 `protojson` expects. Keep `protojson` as the single decoder.
- [ ] Apply at `otlp_receiver.go:166` (traces), `otlp_receiver.go:233` (logs), and the metrics
      equivalent in `otlp_receiver_metrics.go`.
- [ ] Reject malformed IDs with a typed error and **HTTP 400** (Design Freeze: reject, no fallback).
- [ ] Verify the protobuf path (`otlp_receiver.go:161`) is untouched.

**Acceptance criteria:**
- `POST /v1/traces` with `traceId="5b8aa5a2d2c872e8321cf37308d69df2"` stores exactly those 16 bytes;
  test asserts `len(stored)==32`, not 24/48.
- **NON-VACUITY PROOF**: the red-first transcript (`--- FAIL`, showing the 48-char value) is recorded
  in the commit body, taken BEFORE the receiver is modified. Do not use `git stash`/`git checkout`
  to produce it (CLAUDE.md Principle 0).
- A payload with `traceId="not-hex"` returns **400** with a typed error, and **no row is written** —
  asserted by a span count before/after, not by the status code alone.
- JSON and protobuf ingest of the same logical payload produce **identical** stored IDs.
- Codec unit tests cover: valid 16-byte trace ID, valid 8-byte span ID, wrong length, non-hex chars,
  empty, and a value that is **both valid hex and valid base64** (pins the precedence).
- `grep -c "t.Skip" internal/observatory/otlp_receiver_test.go` == 0.
- `go test ./internal/observatory/...` green; existing `convertSpan` tests unchanged and passing.

### M2_BACKFILL_MIGRATION (~90 LOC, ~2h)

Repair the 38 existing corrupted rows.

**Tasks:**
- [ ] `internal/observatory/migrate_v18.go` (v18 verified unallocated; v16/v17 are the current max).
- [ ] For every row whose `trace_id` is 48 chars: `trace_id = base64encode(unhexlify(trace_id))`.
      Same for `span_id`. Guard on length so it is idempotent.

**Acceptance criteria:**
- Running the migration twice produces an identical DB — idempotence asserted by a checksum, not
  by inspection.
- A row seeded with a known corrupted ID recovers the exact original hex.
- Rows with already-correct 32-char IDs are **provably untouched** (count of modified rows asserted).
- Migration runs on a copy of prod's 38 rows without error.

### M3_OPENROUTER_CORRELATION_FIELDS (~140 LOC, ~4h)

Add `user`/`session_id`/`trace` to OpenRouter requests. **Independent of M1/M2.**

**Tasks:**
- [ ] Add `User`, `SessionID`, `Trace` to `chatRequest` (`types.go:16-31`) with `omitempty`.
- [ ] Add the same to `ai.Request` (`internal/ai/provider.go:32`).
- [ ] Wire all three build sites: `chat.go:44`, `step.go:187` (**reuse** the existing
      `marshalStepBodyWithExtras` splice — do not add a second mechanism), `streamstep.go:118`.
- [ ] Populate from `internal/eval_harness/ai_provider.go:81` using the chain ID.
- [ ] Enforce caps (`user` ≤128, `session_id` ≤256) failing loudly, never truncating.

**Acceptance criteria:**
- **GOLDEN-BODY TEST**: an `ai.Request` with no correlation fields set marshals **byte-identical**
  to current output, asserted at **all three** build sites. This is the regression that matters —
  every OpenRouter call in the project flows through these.
- With fields set, the body carries `session_id` and a `trace` object with `trace_name`.
- An over-cap `session_id` (257 chars) returns a typed error before network dispatch; it is never
  silently truncated.
- `go test ./internal/ai/...` green.

### M4_LIVE_VERIFICATION (~60 LOC docs/tests, ~2h)

Prove it against the real third-party producer.

**Tasks:**
- [ ] Deploy, then read the next arriving broadcast span and assert a 32-char trace ID.
- [ ] Run the backfill on prod; spot-check a recovered ID.
- [ ] Remove the KNOWN DEFECT section from `.claude/rules/cloud-endpoints.md`.
- [ ] Update CHANGELOG.

**Acceptance criteria:**
- A **live** broadcast span, post-deploy, has a 32-char trace ID — stronger evidence than any fixture.
- Prod span count before/after backfill is unchanged (repair, not delete).
- `.claude/rules/cloud-endpoints.md` no longer claims a defect that is fixed.

## Out of Scope

**M-AUTH (Phase 2 of the design doc) is deliberately NOT in this sprint's executable milestones** as
a live-enable. Per Design Freeze, the middleware lands **disabled** (`AILANG_OTLP_INGEST_TOKEN`
unset = off) so deploying cannot break live ingest. It is folded into M1 as code + tests only; the
enable step is Mark's, and requires **both** the Cloud Run env var **and** the matching custom header
on the OpenRouter destination.

## Success Metrics

- [ ] Test coverage on `internal/observatory` OTLP paths increases from **0 decode-path tests** to
      full coverage of JSON, protobuf, and rejection
- [ ] All 38 prod rows carry valid 32-char IDs
- [ ] Zero wire-byte change for OpenRouter calls that set no correlation fields
- [ ] `make test` green, `make lint` clean

## Risks

| Risk | Impact | Mitigation |
|------|--------|-----------|
| Golden-body test proves insufficient and correlation fields shift wire bytes | High | Assert at all three build sites separately; `omitempty` on every new field |
| Rejecting malformed IDs drops a producer we mis-diagnosed | Med | Only the rig (protobuf) and OpenRouter (JSON, measured hex) produce today; both covered by tests |
| Backfill touches correct rows | Med | Length guard + assert modified-row count equals the corrupted-row count |
| Deploy breaks live ingest | Med | Auth ships disabled; verify with before/after span count, not a 200 |
