# M-STDLIB-CRYPTO: Cryptographic Hashing for AILANG

**Status**: Planned
**Target**: v0.9.4
**Priority**: P0 (High) — security gap in production use
**Estimated**: 1 day
**Dependencies**: None (stdlib addition, no language changes)
**Source**: DocParse agent message `64ecf19c`

## Axiom Compliance

| Axiom | Score | Justification |
|-------|-------|---------------|
| A1: Determinism | +1 | SHA-256 and HMAC are deterministic — same input always produces same output |
| A2: Replayability | +1 | Crypto functions are pure transforms (no hidden state) |
| A3: Effect Legibility | 0 | No effects needed — pure functions |
| A4: Explicit Authority | 0 | No capabilities required |
| A7: Machines First | 0 | Standard stdlib pattern |
| A8: Minimal Syntax | 0 | No syntax changes — builtin functions only |

**Net Score: +2** → **Decision: Move forward**

---

## Problem Statement

DocParse (and any AILANG API service) needs cryptographic hashing for secure API key management. The current workaround uses a non-cryptographic hash (`foldl * 31`) which is:

- **Insecure** — trivially reversible, collision-prone
- **Not suitable for key storage** — API key hashes stored in Firestore must use real crypto
- **Blocking production deployment** — every `serve-api` service will need API key management

**Current State:**
- No `std/crypto` module exists
- DocParse uses `foldl * 31` for hashing — NOT safe for API key storage
- Go's `crypto/sha256` and `crypto/hmac` are already used internally (see `internal/coordinator/kms.go`, `internal/apiserver/auth.go`)
- Pattern for stdlib builtins is well-established (see `internal/builtins/rand.go`)

**Impact:**
- Every AILANG API service needs secure key hashing
- DocParse is blocked from implementing proper API key management
- Security-sensitive code is being written with insecure primitives

---

## Goals

**Primary Goal:** Provide SHA-256 and HMAC-SHA256 as stdlib builtins for secure hashing.

**Success Metrics:**
- `sha256Hex(string) -> string` produces correct hex digest
- `hmacSha256(message, secret) -> string` produces correct HMAC
- DocParse can replace `foldl * 31` with `sha256Hex`
- All functions are pure (no effects required)

---

## Non-Goals

- Full crypto suite (AES, RSA, etc.) — defer to future need
- Key derivation functions (PBKDF2, scrypt, bcrypt) — defer
- Random byte generation — `std/rand` exists, extend separately if needed
- Certificate/TLS operations — out of scope

---

## Solution Design

### New Module: `std/crypto`

```ailang
-- Standard library: Cryptographic operations
-- Pure functions — no capabilities required

module std/crypto

-- SHA-256 hash of a string, returned as lowercase hex
-- Example: sha256Hex("hello") => "2cf24dba5fb0a30e..."
export pure func sha256Hex(input: string) -> string =
  _crypto_sha256hex(input)

-- SHA-256 hash of raw bytes (from std/bytes), returned as lowercase hex
export pure func sha256Bytes(input: bytes) -> string =
  _crypto_sha256bytes(input)

-- HMAC-SHA256: keyed hash for message authentication
-- Returns lowercase hex string
-- Example: hmacSha256("message", "secret") => "a]..."
export pure func hmacSha256(message: string, key: string) -> string =
  _crypto_hmacsha256(message, key)

-- Constant-time string comparison (prevents timing attacks)
-- Use for comparing hashes, tokens, API keys
export pure func constantTimeEqual(a: string, b: string) -> bool =
  _crypto_constanttimeequal(a, b)
```

### Go Builtin Implementation: `internal/builtins/crypto.go`

```go
package builtins

import (
    "crypto/hmac"
    "crypto/sha256"
    "crypto/subtle"
    "encoding/hex"

    "github.com/sunholo/ailang/internal/eval"
    "github.com/sunholo/ailang/internal/types"
)

func init() {
    registerSha256Hex()
    registerSha256Bytes()
    registerHmacSha256()
    registerConstantTimeEqual()
}

func registerSha256Hex() {
    err := RegisterBuiltin(BuiltinSpec{
        Module: "std/crypto",
        Name:   "_crypto_sha256hex",
        Params: []types.Type2{types.StringType2},
        Return: types.StringType2,
        Impl: func(args []eval.Value) (eval.Value, error) {
            input := args[0].(eval.StringValue).Value
            hash := sha256.Sum256([]byte(input))
            return eval.StringValue{Value: hex.EncodeToString(hash[:])}, nil
        },
    })
    if err != nil {
        panic("failed to register _crypto_sha256hex: " + err.Error())
    }
}
```

Follow the same pattern as `internal/builtins/rand.go` for the remaining three functions.

### Optional: `uuid4` in `std/rand`

```ailang
-- UUID v4 (random): standard format with hyphens
-- Example: uuid4() => "550e8400-e29b-41d4-a716-446655440000"
export func uuid4() -> string ! {Rand} = _rand_uuid4()
```

Lower priority — DocParse is currently generating UUIDs from `rand_int` which works.

---

## Implementation Plan

### Files to Create
- `std/crypto.ail` (~25 LOC) — stdlib module definition
- `internal/builtins/crypto.go` (~120 LOC) — Go builtin implementations
- `internal/builtins/crypto_test.go` (~80 LOC) — unit tests
- `examples/runnable/crypto_hashing.ail` (~15 LOC) — example

### Files to Modify
- `std/README.md` — add crypto module to list
- `std/rand.ail` — add `uuid4` (optional)
- `internal/builtins/rand.go` — add `uuid4` impl (optional)
- `CHANGELOG.md` — document new module

### Estimated LOC: ~240 new, ~10 modified

---

## Testing Strategy

**Unit tests (`internal/builtins/crypto_test.go`):**
- SHA-256 of known strings matches expected hex digests
- SHA-256 of empty string is correct (`e3b0c44298fc1c14...`)
- HMAC-SHA256 matches known test vectors (RFC 4231)
- `constantTimeEqual` returns true for equal, false for unequal
- `constantTimeEqual` handles empty strings, different lengths

**Integration test (`examples/runnable/crypto_hashing.ail`):**
- Import and use all exported functions
- Verify output format (64-char hex for SHA-256)

---

## Success Criteria

- [ ] `sha256Hex("hello")` returns `"2cf24dba5fb0a30e26e83b2ac5b9e29e1b161e5c1fa7425e73043362938b9824"`
- [ ] `hmacSha256("message", "secret")` returns correct HMAC
- [ ] `constantTimeEqual(hash1, hash1)` returns `true`
- [ ] All functions are pure (no effect annotation needed)
- [ ] `make test` passes
- [ ] `make verify-examples` passes
- [ ] DocParse can replace `foldl * 31` with `sha256Hex`

---

## Related Documents

- [std/rand.ail](../../../std/rand.ail) — Existing stdlib pattern for builtins with Go backing
- [internal/builtins/rand.go](../../../internal/builtins/rand.go) — Implementation pattern to follow
- [internal/coordinator/kms.go](../../../internal/coordinator/kms.go) — Existing Go crypto/sha256 usage in codebase
- [internal/apiserver/auth.go](../../../internal/apiserver/auth.go) — Existing constant-time comparison usage

---

**Document created**: 2026-03-19
**Last updated**: 2026-03-19
