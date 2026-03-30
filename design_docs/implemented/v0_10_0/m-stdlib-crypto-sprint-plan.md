# Sprint Plan: M-STDLIB-CRYPTO

**Sprint ID**: M-STDLIB-CRYPTO
**Design Doc**: [m-stdlib-crypto.md](m-stdlib-crypto.md)
**Duration**: 1 day (~3 hours implementation)
**Risk Level**: Low — follows well-established builtin pattern
**Total LOC Estimate**: ~280 new

---

## Sprint Summary

Add `std/crypto` module with SHA-256, HMAC-SHA256, and constant-time comparison. Pure builtins backed by Go's `crypto/sha256`, `crypto/hmac`, and `crypto/subtle`. Follows the exact pattern of `internal/builtins/rand.go` and `internal/builtins/simhash.go`.

---

## Milestones

### M1: Core Crypto Builtins (~1.5 hours)

**Goal**: Implement Go builtins and AILANG stdlib module.

**Files to create:**
- `internal/builtins/crypto.go` (~130 LOC) — 4 builtins: `_crypto_sha256hex`, `_crypto_sha256bytes`, `_crypto_hmacsha256`, `_crypto_constanttimeequal`
- `std/crypto.ail` (~30 LOC) — stdlib module definition

**Pattern**: Copy `internal/builtins/simhash.go` structure (pure builtins with `IsPure: true`, `Effect: ""`).

**Acceptance criteria:**
- `sha256Hex("hello")` returns `"2cf24dba5fb0a30e26e83b2ac5b9e29e1b161e5c1fa7425e73043362938b9824"`
- `sha256Hex("")` returns `"e3b0c44298fc1c149afbf4c8996fb924..."` (SHA-256 of empty string)
- `hmacSha256("message", "secret")` matches Go `crypto/hmac` output
- `constantTimeEqual("abc", "abc")` returns true
- `constantTimeEqual("abc", "def")` returns false
- All functions registered as pure (no effect required)
- `make build` succeeds

### M2: Tests & Example (~1 hour)

**Goal**: Unit tests and runnable example file.

**Files to create:**
- `internal/builtins/crypto_test.go` (~80 LOC) — test all 4 builtins against known test vectors
- `examples/runnable/crypto_hashing.ail` (~20 LOC) — example showing all exports

**Files to modify:**
- `std/README.md` — add crypto to module list (~2 lines)

**Acceptance criteria:**
- `go test ./internal/builtins/ -run Crypto` passes
- SHA-256 test vectors from NIST match
- HMAC test vectors from RFC 4231 match
- `make test` passes (all 3400+ tests)
- `make verify-examples` passes
- Example file runs: `ailang run examples/runnable/crypto_hashing.ail`

### M3: Documentation & Notify (~30 min)

**Goal**: Update changelog, notify DocParse.

**Files to modify:**
- `changelogs/v0.9-current.md` — add std/crypto entry
- Send message to DocParse with usage example

**Acceptance criteria:**
- CHANGELOG entry documents new module
- DocParse notified with migration example (replace `foldl * 31` with `sha256Hex`)
- `make lint` passes

---

## Implementation Notes

- Crypto builtins are **pure** — no capability/effect needed (same as simhash)
- Use `RegisterEffectBuiltin` with `IsPure: true, Effect: ""` (not `RegisterBuiltin`)
- Go type builder: `T.Func(T.String()).Returns(T.String())` for sha256Hex
- `sha256Bytes` takes `bytes` type — check how `internal/builtins/bytes.go` handles the bytes type
- `constantTimeEqual` uses `crypto/subtle.ConstantTimeCompare` — returns `bool` not `int`

---

## Success Metrics

- [ ] All 4 crypto builtins work in AILANG code
- [ ] `make test` passes
- [ ] `make lint` passes
- [ ] `make verify-examples` passes
- [ ] DocParse unblocked for API key management

---

**Created**: 2026-03-19
