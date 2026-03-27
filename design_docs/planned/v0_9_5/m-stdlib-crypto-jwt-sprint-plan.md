# Sprint Plan: M-STDLIB-CRYPTO-JWT

## Summary

Add RSA signature verification and JWT parsing/verification to AILANG's standard library, enabling Firebase JWT auth and generic RS256 token verification in pure AILANG code.

**Duration:** 2 days (3 milestones)
**Dependencies:** None — builds on existing `std/crypto`, `std/bytes`, `std/json`
**Risk Level:** Low — well-understood crypto primitives, clear patterns from existing builtins
**Design Doc:** `design_docs/planned/v0_9_5/m-stdlib-crypto-jwt.md`

## Current Status Analysis

### Completed Recently
- M-STDLIB-CRYPTO: SHA-256, HMAC-SHA256, constant-time comparison (~210 LOC)
- M-TYPE-ALIAS: Cross-package record type alias unification
- M-TYPEENV-SUB: Design doc + regression tests (in progress)

### Velocity
- Recent 14 days: ~3280 LOC across 19 commits (~234 LOC/day)
- Estimated capacity for 2-day sprint: ~400-500 LOC

### Remaining from Design Doc
- Layer 1: RSA verify Go builtin (~120 LOC impl + ~150 LOC tests)
- Layer 2: Base64URL Go builtin (~40 LOC impl + ~30 LOC tests)
- Layer 3: `std/jwt` AILANG module (~180 LOC)
- Examples + integration tests (~190 LOC)
- **Total: ~710 LOC** (fits 2-day sprint with buffer)

---

## Proposed Milestones

### Milestone 1: Go Builtins — RSA Verify + Base64URL
**Goal:** Add the two Go-backed primitives that the JWT module depends on
**Estimated:** 160 LOC implementation + 180 LOC tests = 340 LOC
**Duration:** 1 day

**Tasks:**
1. Create `internal/builtins/crypto_rsa.go`:
   - Register `_crypto_rsa_verify_pkcs1v15` builtin
   - Accept PEM public keys AND X.509 certificates
   - Return `Result[bool, string]` using `eval.TaggedValue` pattern from `json_decode.go`
   - Type: `T.Func(T.Bytes(), T.Bytes(), T.String()).Returns(T.App("Result", T.Bool(), T.String())).Build()`
   - Go impl: `encoding/pem` → `x509.ParseCertificate`/`x509.ParsePKIXPublicKey` → `rsa.VerifyPKCS1v15` with SHA-256
2. Create `internal/builtins/crypto_rsa_test.go`:
   - Generate RSA keypair in test setup
   - Test valid signature → `Ok(true)`
   - Test tampered signature → `Ok(false)`
   - Test X.509 cert PEM, PKCS#8 PEM, PKCS#1 PEM
   - Test invalid PEM → `Err(...)`
   - Test non-RSA key → `Err(...)`
3. Add `fromBase64URL` to `internal/builtins/bytes.go`:
   - Register `_bytes_from_base64url` builtin
   - Return `Option[bytes]` using `eval.TaggedValue` pattern from existing `_bytes_from_base64`
   - Go impl: `base64.RawURLEncoding.DecodeString` (no padding)
4. Add base64url tests to `internal/builtins/bytes_test.go`
5. Update `std/crypto.ail` — add `rsaVerifyPKCS1v15` export
6. Update `std/bytes.ail` — add `fromBase64URL` export

**Acceptance Criteria:**
- [ ] `_crypto_rsa_verify_pkcs1v15` correctly verifies RSA-SHA256 signatures
- [ ] Accepts X.509 certificate PEM (Firebase key format)
- [ ] Invalid signature returns `Ok(false)`, not `Err`
- [ ] Invalid PEM/key returns `Err(message)`
- [ ] `_bytes_from_base64url` decodes without padding
- [ ] All Go unit tests pass
- [ ] `make test` passes (no regressions)
- [ ] `make lint` clean

**Risks:**
- `T.App("Result", T.Bool(), T.String())` — confirmed available via `builder.go:17`
- `wrapOk`/`wrapErr` are in `json_decode.go` (not exported) — either duplicate or extract to shared helper

**Key files to reference:**
- `internal/builtins/crypto.go` — registration pattern
- `internal/builtins/bytes.go:230-254` — Option wrapping pattern for `fromBase64`
- `internal/builtins/json_decode.go:220-230` — `wrapOk`/`wrapErr` Result helpers
- `internal/types/builder.go:17` — `T.App("Result", ...)` usage

---

### Milestone 2: AILANG JWT Module
**Goal:** Create `std/jwt` with decode, verify, and claims helpers — all in pure AILANG
**Estimated:** 180 LOC implementation + 40 LOC example = 220 LOC
**Duration:** 0.5 day

**Tasks:**
1. Create `std/jwt.ail`:
   - `decodeJWT(token) -> Result[DecodedJWT, string]` — split on `.`, base64url decode, JSON parse
   - `verifyRS256(token, publicKeyPEM) -> Result[Json, string]` — decode + RSA verify
   - `verifyWithKid(token, keys) -> Result[Json, string]` — select key by kid header
   - `isExpired(claims, nowUnix) -> bool`
   - `checkIssuer(claims, expectedIss) -> bool`
   - `checkAudience(claims, expectedAud) -> bool` — handles string and array aud
2. Create `examples/runnable/jwt_verification.ail`:
   - Demonstrate `decodeJWT` with a test token (no signature verify — just parsing)
   - Show claims extraction with `getString`, `getInt`
3. Update `std/README.md` — add `std/jwt` to module list

**Acceptance Criteria:**
- [ ] `decodeJWT` parses valid JWTs into header + payload + signature
- [ ] `decodeJWT` returns `Err` for malformed tokens
- [ ] `verifyRS256` verifies RS256 tokens and rejects non-RS256
- [ ] `verifyWithKid` selects correct key from JObject by kid
- [ ] Claims helpers return correct results
- [ ] Example file compiles and runs
- [ ] `make verify-examples` passes

**Risks:**
- Record return type (`DecodedJWT`) may hit M-TYPEENV-SUB if the fix isn't landed — fallback: return `Json` or use a simpler return shape
- `std/string.split` exists (confirmed at `std/string.ail:40`) — no risk

---

### Milestone 3: Integration Tests + Docs
**Goal:** End-to-end verification and documentation
**Estimated:** 100 LOC tests + 50 LOC example = 150 LOC
**Duration:** 0.5 day

**Tasks:**
1. Create `internal/pipeline/jwt_test.go`:
   - Test JWT decode via AILANG pipeline (compile + run)
   - Test RS256 verification with Go-generated test JWT
   - Test invalid signature rejection
   - Test `verifyWithKid` key selection
   - Test claims helpers
2. Create `examples/runnable/firebase_auth.ail`:
   - Firebase auth example (type-checks but won't run without real token)
3. Update `CHANGELOG.md` with new features

**Acceptance Criteria:**
- [ ] Integration tests pass through full AILANG pipeline
- [ ] Firebase example compiles (type-checks)
- [ ] `make test` passes
- [ ] `make verify-examples` passes
- [ ] `make lint` clean
- [ ] CHANGELOG updated

---

## Success Metrics
- All Go unit tests passing
- All integration tests passing
- `make test` green
- `make verify-examples` green (all 152+ examples)
- `make lint` clean
- CHANGELOG.md updated
- No new Go dependencies (only stdlib `crypto/*`, `encoding/*`)

## Pause Points

- **After M1**: If RSA verify works in Go tests, safe to proceed. If `T.App("Result", ...)` doesn't work for the return type, stop and fix type builder.
- **After M2**: If `std/jwt.ail` compiles and the example runs, proceed to integration tests. If record types hit M-TYPEENV-SUB, switch to `Json` return type.

## Notes
- All new functions are pure — no effect annotations needed
- The `wrapOk`/`wrapErr` helpers in `json_decode.go` are unexported — M1 should either duplicate them in `crypto_rsa.go` or extract to a shared `adt_helpers.go`
- Firebase example uses `std/net.httpRequest` (Net effect) and a hypothetical `std/time.nowUnix` (Clock effect) — these are caller concerns, not part of this sprint
- The design doc is at `design_docs/planned/v0_9_5/m-stdlib-crypto-jwt.md`
