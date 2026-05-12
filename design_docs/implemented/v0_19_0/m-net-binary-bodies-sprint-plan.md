# M-NET-BINARY-BODIES — Sprint Plan

**Design doc**: [m-net-binary-bodies.md](m-net-binary-bodies.md)
**Sprint ID**: M-NET-BINARY-BODIES
**Target**: v0.19.0
**Estimated**: ~3–5 hours (one focused session)
**Risk**: Low–Medium
**Created**: 2026-05-12
**Source**: inbox message [12a77549](msg_20260512_101641_12a77549) from `demos/linkedin`

---

## Pre-flight checks (done)

- ✅ `eval.BytesValue` exists ([internal/eval/value.go:46](../../../internal/eval/value.go#L46))
- ✅ `types.TBytes` exists ([internal/types/types.go:443](../../../internal/types/types.go#L443))
- ✅ `T.Bytes()` builder helper exists (used by `_fs_writeFileBytes` at [internal/builtins/fs.go:128](../../../internal/builtins/fs.go#L128))
- ✅ Mirror builtin pattern exists: `_fs_writeFileBytes` already takes `Bytes` arg end-to-end
- ✅ `httpRequest` security path is well-isolated in `NetHTTPRequest` ([internal/effects/net.go:409](../../../internal/effects/net.go#L409))

**Conclusion**: All type-system + value plumbing is in place. No new infrastructure required.

---

## Velocity context

Recent comparable work:
- M-EXT-PORTABILITY-GATE follow-up F1–F5 — 5 small features in ~1 day
- M-PARSER-ROW-POLY-EFFECTS Phase 1 + 2 — 2 features, ~1 day each
- v0.18.11 release — same week

Pattern: small, additive features with focused tests are landing at ~1 feature per 1–2h. This sprint = 4 milestones of ~30–60 min each ⇒ 3–5h total is realistic.

---

## Milestones

### M1 — Refactor: extract `buildSecureRequest` helper (~30 min, ~60 LOC moved)

**Goal**: Steps 0–8 of `NetHTTPRequest` become a shared helper so the new bytes path can't drift on security checks.

**Files**:
- `internal/effects/net.go` — extract helper, refactor `NetHTTPRequest` to use it

**Tasks**:
1. Create `buildSecureRequest(ctx, method, url, headersList) (*http.Client, *http.Request, *url.URL, error)` (or similar) that does cap check, method validation, URL parse, protocol/domain/IP validation, header parsing, client construction, base `*http.Request` with User-Agent + Host.
2. Refactor `NetHTTPRequest` to call the helper, then add body + execute.
3. Run `make test` — all existing net tests must still pass with zero behavioral change.

**Acceptance**:
- [ ] `internal/effects/net_test.go` and `net_security_test.go` all pass
- [ ] `make verify-examples` still green
- [ ] Diff to `NetHTTPRequest` is purely deletion (no logic moved into the body section)

**Risk**: Medium. This touches the security-critical hot path. Mitigation: no logic changes, just extraction. Test suite is comprehensive.

---

### M2 — Add `bodyBytes` field to `HttpResponse` (~20 min, ~10 LOC)

**Goal**: Existing `httpRequest` callers can now access raw response bytes via `resp.bodyBytes` while `resp.body` (string view) still works.

**Files**:
- `internal/effects/net.go` — add `"bodyBytes"` to the response RecordValue
- `internal/builtins/net.go` — add `bodyBytes: bytes` to the response type in `makeHTTPRequestType`
- `std/net.ail` — add `bodyBytes: bytes` to `HttpResponse` record type

**Tasks**:
1. Add the field at all three layers (effect → builtin type → stdlib type).
2. Reuse the already-read `respBody` — no extra allocation, just `&BytesValue{Value: respBody}`.
3. Add a smoke test that `httpRequest` against a binary endpoint populates both `body` and `bodyBytes` correctly.

**Acceptance**:
- [ ] `httpRequest("GET", binary_url, [], "")` returns `Ok(resp)` with both `resp.body` (lossy string) and `resp.bodyBytes` (exact bytes)
- [ ] Binary-bytes test confirms `length(resp.bodyBytes) == content-length` even when string view is shorter (UTF-8 replacement)
- [ ] All existing tests still pass (record-type widening should be transparent)

**Risk**: Low. Field addition is backward-compatible — existing callers never reference the new field.

---

### M3 — Implement `_net_httpRequestBytes` builtin + effect handler (~90 min, ~80 LOC core + ~50 LOC type spec)

**Goal**: New builtin that mirrors `_net_httpRequest` but takes a `Bytes` body.

**Files**:
- `internal/effects/net.go` — `NetHTTPRequestBytes` (uses M1 helper)
- `internal/builtins/net.go` — `registerNetHTTPRequestBytes` + `makeHTTPRequestBytesType`
- `internal/builtins/registry.go` — registry entry for `_net_httpRequestBytes`
- `internal/builtins/registry_codegen.go` — codegen entry

**Tasks**:
1. Implement `NetHTTPRequestBytes`:
   - Call `buildSecureRequest` for steps 0–8
   - Unpack arg 3 as `*eval.BytesValue` (mirror pattern from `fsWriteFileBytes` at `internal/effects/fs.go:222`)
   - Use `bytes.NewReader(bodyVal.Value)` for the request body
   - Default `Content-Type: application/octet-stream` if user didn't set one
   - Set `req.ContentLength = int64(len(bodyVal.Value))` to suppress chunked encoding
2. Type signature: `(string, string, list[{name:string,value:string}], bytes) -> Result[HttpResponse, NetError] ! {Net}` (`HttpResponse` already includes `bodyBytes` from M2)
3. Register builtin via `RegisterEffectBuiltin`
4. Wire codegen entry mirroring `_net_httpRequest`

**Acceptance**:
- [ ] `_net_httpRequestBytes` callable from AILANG via the runtime
- [ ] Type-checks against `bytes` argument; rejects `string`
- [ ] Test: PUT a 1KB random byte payload to `httptest.Server` that responds with SHA256 of body — assert client SHA matches request SHA
- [ ] Test: caller-supplied `Content-Type` wins over default
- [ ] Test: empty `bytes` body sends `Content-Length: 0` with no chunked encoding
- [ ] Test: all M1 security checks fire (DNS rebinding, IP block, redirect — at least one repro per category against `httpRequestBytes`)

**Risk**: Low. Mechanical — same shape as `_net_httpRequest`, with `BytesValue` instead of `StringValue`.

---

### M4 — Stdlib `httpRequestBytes` + runnable example + verify (~60 min, ~30 LOC stdlib + ~50 LOC example + tests)

**Goal**: User-facing `httpRequestBytes` in `std/net`, plus an example demonstrating the LinkedIn-style upload (against a local httptest server).

**Files**:
- `std/net.ail` — `export func httpRequestBytes(...)` with full docstring
- `examples/runnable/http_put_bytes.ail` — new example
- `examples/runnable.test` (or wherever runnable example tests are wired) — register the new example

**Tasks**:
1. Add `httpRequestBytes` wrapper in `std/net.ail` (single-line `_net_httpRequestBytes(method, url, headers, body)` body, full docstring matching `httpRequest`'s style with binary-specific notes).
2. Update the `HttpResponse` record type in `std/net.ail` to include `bodyBytes: bytes`.
3. Write `examples/runnable/http_put_bytes.ail`:
   - Reads a small binary blob from disk (via `readFileBytes` + `fromBase64` to demonstrate the documented bridge), OR builds bytes via `bytes.fromInts` for full self-containment
   - PUTs to `http://localhost:<port>/upload` (httptest-style — needs investigation whether `make verify-examples` can spin up a server, or if this example just needs to be doc-only)
   - Asserts `resp.ok` and prints status
4. Run `make verify-examples` — must pass.
5. Update [docs/docs/reference/effects.md](../../../docs/docs/reference/effects.md) Net section with the new function.

**Acceptance**:
- [ ] `import std/net (httpRequestBytes, HttpResponse)` works
- [ ] Example file documented and either runnable in CI or marked as "manual-run" with clear instructions
- [ ] `make ci` green
- [ ] `make verify-examples` green

**Risk**: Medium-Low. The "runnable example needs a server" question may force this to be a doc-example rather than a CI-runnable one. Acceptable fallback: write the example, run it manually against `python3 -m http.server` style stub, document how to repro.

---

## Open question to resolve during M3

**Transparent gzip on binary responses** (flagged in design doc): Go's `http.Transport` may auto-decompress `Content-Encoding: gzip` responses. For binary downloads this is usually wanted, but for an exact-bytes round-trip it might not be.

**Test in M3**: serve a known-binary payload from `httptest.Server` with `Content-Encoding: identity` — confirm `bodyBytes` matches input byte-for-byte. If a follow-up gzip test surfaces unexpected inflation, document and defer (out of scope for this sprint).

---

## Day-by-day plan

This is a one-session sprint. Suggested order:

| Hour | Milestone | Deliverable |
|---|---|---|
| 0:00–0:30 | M1 | `buildSecureRequest` helper, all existing tests still green |
| 0:30–0:50 | M2 | `bodyBytes` field added at all 3 layers, smoke test green |
| 0:50–2:20 | M3 | `_net_httpRequestBytes` builtin + effect + 5 tests green |
| 2:20–3:20 | M4 | Stdlib wrapper, example, `make ci` green |
| 3:20–3:40 | Wrap | Reply to inbox message [12a77549](msg_20260512_101641_12a77549) with shipped-version snippet, ack inbox, move design doc to `implemented/v0_19_0/` |

Buffer: ~1h for the gzip investigation or unexpected test friction.

---

## Success metrics

- [ ] All 4 milestones complete
- [ ] `make ci` green
- [ ] `make verify-examples` green (or example documented as manual-run with clear justification)
- [ ] `examples/runnable/http_put_bytes.ail` exists and is verified working
- [ ] CHANGELOG.md updated with M-NET-BINARY-BODIES entry
- [ ] Design doc moved from `planned/v0_19_0/` to `implemented/v0_19_0/`
- [ ] Reply sent to inbox message 12a77549 with working code snippet

---

## Dependencies

None. All upstream pieces (`BytesValue`, `TBytes`, `T.Bytes()`, builtin registry pattern) already exist.

---

## Risks summary

| Risk | Likelihood | Impact | Mitigation |
|---|---|---|---|
| `buildSecureRequest` extraction breaks existing security tests | Low | High | No logic changes, comprehensive test suite catches drift |
| Record-type widening (`bodyBytes` field) breaks existing callers | Very Low | Medium | Records are open / row-polymorphic; existing callers don't reference new field |
| Transparent gzip on binary response | Medium | Low | Add identity-encoding test; defer gzip opt-out to follow-up if needed |
| Runnable example needs HTTP server | Medium | Low | Use `bytes.fromInts` + manual-run docs as fallback |
