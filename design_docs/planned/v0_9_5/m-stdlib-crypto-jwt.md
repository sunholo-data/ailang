# M-STDLIB-CRYPTO-JWT: RSA Signature Verification & JWT Support

**Status**: Implemented
**Target**: v0.9.5
**Priority**: P1 — unblocks Firebase JWT verification in AILANG code
**Estimated**: 2-3 days
**Dependencies**: None (builds on existing `std/crypto`, `std/bytes`, `std/json`)
**Author**: Mark + Claude
**Created**: 2026-03-27
**Last updated**: 2026-03-27
**Triggered by**: Firebase auth flow needs JWT verification in AILANG — `std/crypto` has symmetric primitives but no RSA

---

## Axiom Compliance

| Axiom | Score | Justification |
|-------|-------|---------------|
| A1: Determinism | +1 | RSA verification is deterministic — same (message, sig, key) always gives same result |
| A2: Replayability | +1 | All new functions are pure transforms (no hidden state) |
| A3: Effect Legibility | +1 | Key fetching (Net) is the caller's responsibility, not the JWT module's |
| A4: Explicit Authority | +1 | No ambient capabilities — caller must provide keys explicitly |
| A5: Bounded Verification | 0 | No impact on type checking |
| A6: Safe Concurrency | 0 | No concurrency changes |
| A7: Machines First | +1 | AI agents can verify tokens without Go escape hatch |
| A8: Minimal Syntax | 0 | No new syntax — stdlib functions only |
| A9: Cost Visibility | 0 | No cost changes |
| A10: Composability | +1 | Layered design: crypto primitives compose into JWT verification |
| A11: Structured Failure | +1 | All errors returned as `Result[T, string]`, not panics |
| A12: System Boundary | 0 | No boundary changes |

**Net Score: +7** → **Decision: Move forward**

### Hard Violation Check

- [x] A1 (Determinism): No implicit nondeterminism introduced
- [x] A3 (Effects): No hidden side effects — key fetching is caller's job
- [x] A4 (Authority): No ambient access granted
- [x] A7 (Machines First): Directly enables machine-driven auth flows

---

## Problem Statement

AILANG services that need to verify Firebase ID tokens (or any RS256-signed JWT) cannot do so in AILANG code. The current `std/crypto` module provides only symmetric primitives:

- `sha256Hex` / `sha256Bytes` — hashing
- `hmacSha256` — HMAC (symmetric signing)
- `constantTimeEqual` — timing-safe comparison

**What's missing:**
1. **RSA signature verification** — needed to check JWT signatures against public keys
2. **Base64URL decoding** — JWTs use base64url (RFC 4648 §5) without padding, not standard base64
3. **JWT parsing** — splitting `header.payload.signature`, decoding each part
4. **JWT verification logic** — combining decode + signature check + claims validation

**Current workaround:** Verify JWTs in the Go server layer (`internal/server/auth/auth.go`) and pass claims into AILANG as records. This works for the server but not for:
- Standalone AILANG services deployed independently
- AILANG-native API middleware (e.g., `serve-api` handlers that need auth)
- Testing and development without the Go server

**Go has everything we need:**
- `crypto/rsa`, `crypto/x509`, `encoding/pem` — RSA key parsing and verification
- `encoding/base64.RawURLEncoding` — base64url without padding
- `github.com/golang-jwt/jwt/v4` is already an indirect dependency (via Firebase SDK)

---

## Goals

**Primary Goal:** Enable AILANG code to verify RS256-signed JWTs (including Firebase ID tokens) using only stdlib functions.

**Success Metrics:**
- `rsaVerifyPKCS1v15(message, signature, pemKey)` correctly verifies RSA-SHA256 signatures
- `decodeJWT(token)` parses any well-formed JWT into header + payload + signature
- `verifyRS256(token, pemKey)` decodes and verifies in one call
- Firebase ID token verification works end-to-end in AILANG
- All new functions are pure (no effects) — key fetching is the caller's responsibility
- All existing tests pass (no regressions)

---

## Non-Goals

- **Full RSA key operations** (key generation, signing, encryption/decryption) — verify only
- **ECDSA/EdDSA support** — RS256 covers Firebase and most OAuth providers; add others when needed
- **JWE (encrypted JWTs)** — only JWS (signed) tokens
- **Built-in key fetching/caching** — caller uses `std/net` to fetch keys, passes them in
- **Firebase-specific builtin** — the API is generic; Firebase is an example consumer
- **HMAC-signed JWTs (HS256)** — can be built from existing `hmacSha256` + `constantTimeEqual`; document as example but don't add dedicated function
- **Automatic claim validation** (exp, nbf, iss, aud) — provide helpers, but caller decides policy

---

## Solution Design

### Architecture: Three Layers

```
┌──────────────────────────────────────────────┐
│  Layer 3: std/jwt (AILANG module)            │
│  decodeJWT, verifyRS256, verifyWithKid       │
│  Claims helpers: isExpired, checkIssuer       │
├──────────────────────────────────────────────┤
│  Layer 2: std/bytes addition (Go builtin)    │
│  fromBase64URL                               │
├──────────────────────────────────────────────┤
│  Layer 1: std/crypto addition (Go builtin)   │
│  rsaVerifyPKCS1v15                           │
└──────────────────────────────────────────────┘
```

Key insight: JWT parsing is simple string manipulation (split on `.`, base64url-decode, JSON-decode). It doesn't need a Go JWT library — we implement it in AILANG using existing `std/json.decode` and new `std/bytes.fromBase64URL`. Only the RSA signature verification needs a Go builtin.

---

### Layer 1: `std/crypto` Addition — RSA Verify

**New Go builtin:** `_crypto_rsa_verify_pkcs1v15`

```go
// internal/builtins/crypto_rsa.go

// rsaVerifyPKCS1v15: (bytes, bytes, string) -> Result[bool, string]
// Args: message (bytes to verify), signature (bytes), publicKeyPEM (string)
// The PEM can be either:
//   - PKCS#1/PKCS#8 RSA public key ("BEGIN PUBLIC KEY" / "BEGIN RSA PUBLIC KEY")
//   - X.509 certificate ("BEGIN CERTIFICATE") — extracts RSA public key from cert
// Returns Ok(true) if signature valid, Ok(false) if invalid, Err(msg) if key parsing fails.

func rsaVerifyImpl(_ *effects.EffContext, args []eval.Value) (eval.Value, error) {
    message := args[0].(*eval.BytesValue).Value   // message bytes
    sig := args[1].(*eval.BytesValue).Value        // signature bytes
    pemStr := args[2].(*eval.StringValue).Value    // PEM-encoded key or cert

    // 1. Decode PEM block
    block, _ := pem.Decode([]byte(pemStr))
    if block == nil {
        return wrapErr("invalid PEM: no PEM block found"), nil
    }

    // 2. Parse key based on PEM type
    var pubKey *rsa.PublicKey
    switch block.Type {
    case "CERTIFICATE":
        cert, err := x509.ParseCertificate(block.Bytes)
        if err != nil {
            return wrapErr("invalid certificate: " + err.Error()), nil
        }
        rsaKey, ok := cert.PublicKey.(*rsa.PublicKey)
        if !ok {
            return wrapErr("certificate does not contain RSA public key"), nil
        }
        pubKey = rsaKey
    case "PUBLIC KEY":
        parsed, err := x509.ParsePKIXPublicKey(block.Bytes)
        if err != nil {
            return wrapErr("invalid public key: " + err.Error()), nil
        }
        rsaKey, ok := parsed.(*rsa.PublicKey)
        if !ok {
            return wrapErr("key is not RSA"), nil
        }
        pubKey = rsaKey
    case "RSA PUBLIC KEY":
        parsed, err := x509.ParsePKCS1PublicKey(block.Bytes)
        if err != nil {
            return wrapErr("invalid PKCS#1 public key: " + err.Error()), nil
        }
        pubKey = parsed
    default:
        return wrapErr("unsupported PEM type: " + block.Type), nil
    }

    // 3. Hash the message with SHA-256
    hashed := sha256.Sum256(message)

    // 4. Verify signature
    err := rsa.VerifyPKCS1v15(pubKey, crypto.SHA256, hashed[:], sig)
    if err != nil {
        return wrapOk(&eval.BoolValue{Value: false}), nil  // invalid sig is Ok(false), not Err
    }
    return wrapOk(&eval.BoolValue{Value: true}), nil
}
```

**Type signature:**
```go
func makeRSAVerifyType() types.Type {
    T := types.NewBuilder()
    return T.Func(T.Bytes(), T.Bytes(), T.String()).
        Returns(T.Result(T.Bool(), T.String())).Build()
}
```

**AILANG export in `std/crypto.ail`:**
```ailang
-- RSA PKCS#1 v1.5 signature verification with SHA-256
--
-- Verifies that `signature` is a valid RSA-SHA256 signature of `message`
-- using the given PEM-encoded public key or X.509 certificate.
--
-- Args:
--   message: bytes - The original message that was signed
--   signature: bytes - The RSA signature to verify
--   publicKeyPEM: string - PEM-encoded public key or X.509 certificate
--
-- Returns:
--   Result[bool, string] - Ok(true) if valid, Ok(false) if invalid, Err if key parsing fails
--
-- Example:
--   let result = rsaVerifyPKCS1v15(messageBytes, sigBytes, pemKey) in
--   match result {
--     Ok(true) => println("Signature valid"),
--     Ok(false) => println("Signature invalid"),
--     Err(e) => println("Key error: " ++ e)
--   }
export pure func rsaVerifyPKCS1v15(
  message: bytes,
  signature: bytes,
  publicKeyPEM: string
) -> Result[bool, string] =
  _crypto_rsa_verify_pkcs1v15(message, signature, publicKeyPEM)
```

**Design decisions:**
- **Pure function**: RSA verification is deterministic math — no effects needed
- **Ok(false) vs Err for bad signature**: Invalid signature is a normal outcome (Ok(false)), not an error. Err is reserved for key parsing failures. This follows the principle that "wrong signature" is data, "corrupt key" is an error.
- **Accepts X.509 certificates**: Firebase public keys are X.509 certs, not raw RSA keys. Supporting both avoids forcing callers to extract keys themselves.

---

### Layer 2: `std/bytes` Addition — Base64URL

**New Go builtin:** `_bytes_from_base64url`

```go
// internal/builtins/bytes.go (add to existing file)

// fromBase64URL: string -> Option[bytes]
// Decodes base64url (RFC 4648 §5) without padding.
// JWT tokens use this encoding for header and payload segments.

func base64URLDecodeImpl(_ *effects.EffContext, args []eval.Value) (eval.Value, error) {
    strVal := args[0].(*eval.StringValue)
    decoded, err := base64.RawURLEncoding.DecodeString(strVal.Value)
    if err != nil {
        return wrapNone(), nil  // Invalid base64url returns None
    }
    return wrapSome(&eval.BytesValue{Value: decoded}), nil
}
```

**AILANG export in `std/bytes.ail`:**
```ailang
-- Decode base64url (RFC 4648 §5) without padding to bytes
--
-- JWT tokens use base64url encoding (URL-safe alphabet, no padding).
-- Returns None if the input is not valid base64url.
--
-- Args:
--   s: string - Base64url-encoded string (no padding)
--
-- Returns:
--   Option[bytes] - Some(decoded) or None
--
-- Example:
--   fromBase64URL("SGVsbG8")  -- => Some(<bytes: Hello>)
--   fromBase64URL("!!!invalid!!!")  -- => None
export pure func fromBase64URL(s: string) -> Option[bytes] =
  _bytes_from_base64url(s)
```

---

### Layer 3: `std/jwt` — New AILANG Module

This layer is **pure AILANG code** composing the builtins from layers 1 and 2 with existing `std/json` and `std/bytes`.

```ailang
-- Standard library: JWT (JSON Web Token) parsing and verification
-- Pure functions — no capabilities required
-- Key fetching is the caller's responsibility (use std/net)

module std/jwt

import std/result (Result, Ok, Err, flatMap)
import std/option (Option, Some, None)
import std/json (Json, JObject, JString, decode, get, getString, asString)
import std/bytes (fromBase64URL, toString, fromString)
import std/crypto (rsaVerifyPKCS1v15)
import std/string (split, length)
import std/math (floatToInt)

-- ============================================================================
-- JWT Decoding (pure parsing, no verification)
-- ============================================================================

-- Decoded JWT with all three parts accessible
-- `signed` is the raw "header.payload" string used as signature input
export type DecodedJWT = {
  header: Json,
  payload: Json,
  signature: bytes,
  signed: string
}

-- Decode a JWT token into its three parts without verifying the signature.
--
-- Splits on '.', base64url-decodes header and payload, JSON-parses them.
-- Does NOT verify the signature — use verifyRS256 for that.
--
-- Args:
--   token: string - The raw JWT string (header.payload.signature)
--
-- Returns:
--   Result[DecodedJWT, string] - Decoded parts or error message
--
-- Example:
--   match decodeJWT(token) {
--     Ok(jwt) => println("Subject: " ++ show(getString(jwt.payload, "sub"))),
--     Err(e) => println("Invalid JWT: " ++ e)
--   }
export pure func decodeJWT(token: string) -> Result[DecodedJWT, string] = {
  let parts = split(token, ".") in
  match parts {
    [headerB64, payloadB64, sigB64] => {
      -- Decode header
      let headerResult = decodeSegment(headerB64, "header") in
      flatMap(\header.
        -- Decode payload
        let payloadResult = decodeSegment(payloadB64, "payload") in
        flatMap(\payload.
          -- Decode signature (raw bytes, not JSON)
          match fromBase64URL(sigB64) {
            Some(sigBytes) =>
              Ok({
                header: header,
                payload: payload,
                signature: sigBytes,
                signed: headerB64 ++ "." ++ payloadB64
              }),
            None => Err("invalid base64url in signature")
          }
        , payloadResult)
      , headerResult)
    },
    _ => Err("invalid JWT: expected 3 dot-separated segments, got " ++ show(length(parts)))
  }
}

-- Helper: base64url-decode a segment and JSON-parse it
pure func decodeSegment(b64: string, name: string) -> Result[Json, string] = {
  match fromBase64URL(b64) {
    Some(rawBytes) =>
      match decode(toString(rawBytes)) {
        Ok(json) => Ok(json),
        Err(e) => Err("invalid JSON in " ++ name ++ ": " ++ e)
      },
    None => Err("invalid base64url in " ++ name)
  }
}

-- ============================================================================
-- RS256 Verification
-- ============================================================================

-- Verify an RS256-signed JWT against a PEM-encoded public key or certificate.
--
-- Decodes the JWT, checks that the algorithm is RS256, verifies the RSA-SHA256
-- signature, and returns the payload claims on success.
--
-- Args:
--   token: string - The raw JWT string
--   publicKeyPEM: string - PEM-encoded RSA public key or X.509 certificate
--
-- Returns:
--   Result[Json, string] - Payload claims (as Json) or error message
--
-- Example:
--   match verifyRS256(token, pemKey) {
--     Ok(claims) => println("Verified! Sub: " ++ show(getString(claims, "sub"))),
--     Err(e) => println("Verification failed: " ++ e)
--   }
export pure func verifyRS256(token: string, publicKeyPEM: string) -> Result[Json, string] = {
  flatMap(\jwt.
    -- Check algorithm
    match getString(jwt.header, "alg") {
      Some("RS256") => {
        -- Verify signature: RSA-SHA256 over the signed portion
        let signedBytes = fromString(jwt.signed) in
        match rsaVerifyPKCS1v15(signedBytes, jwt.signature, publicKeyPEM) {
          Ok(true) => Ok(jwt.payload),
          Ok(false) => Err("invalid signature"),
          Err(e) => Err("key error: " ++ e)
        }
      },
      Some(alg) => Err("unsupported algorithm: " ++ alg ++ " (expected RS256)"),
      None => Err("missing 'alg' in JWT header")
    }
  , decodeJWT(token))
}

-- ============================================================================
-- Key Selection by Key ID (kid)
-- ============================================================================

-- Verify a JWT by selecting the signing key from a set of keys using the
-- 'kid' (Key ID) header claim.
--
-- This is the pattern used by Firebase, Google, and most OAuth providers:
-- they publish a set of public keys indexed by key ID, and each JWT's
-- header contains a 'kid' field indicating which key signed it.
--
-- Args:
--   token: string - The raw JWT string
--   keys: Json - JObject mapping kid strings to PEM-encoded keys/certs
--
-- Returns:
--   Result[Json, string] - Payload claims or error message
--
-- Example (Firebase):
--   -- Fetch keys: httpRequest("GET", firebaseKeysURL, [], "")
--   -- Parse response body as JSON to get {kid: pem, ...}
--   match verifyWithKid(token, keysJson) {
--     Ok(claims) => println("User: " ++ show(getString(claims, "sub"))),
--     Err(e) => println("Auth failed: " ++ e)
--   }
export pure func verifyWithKid(token: string, keys: Json) -> Result[Json, string] = {
  flatMap(\jwt.
    -- Extract kid from header
    match getString(jwt.header, "kid") {
      Some(kid) =>
        -- Look up key in the provided key set
        match get(keys, kid) {
          Some(keyJson) =>
            match asString(keyJson) {
              Some(pem) => {
                -- Check algorithm
                match getString(jwt.header, "alg") {
                  Some("RS256") => {
                    let signedBytes = fromString(jwt.signed) in
                    match rsaVerifyPKCS1v15(signedBytes, jwt.signature, pem) {
                      Ok(true) => Ok(jwt.payload),
                      Ok(false) => Err("invalid signature for kid: " ++ kid),
                      Err(e) => Err("key error for kid " ++ kid ++ ": " ++ e)
                    }
                  },
                  Some(alg) => Err("unsupported algorithm: " ++ alg),
                  None => Err("missing 'alg' in JWT header")
                }
              },
              None => Err("key for kid '" ++ kid ++ "' is not a string")
            },
          None => Err("unknown kid: " ++ kid ++ " (not in provided key set)")
        },
      None => Err("missing 'kid' in JWT header")
    }
  , decodeJWT(token))
}

-- ============================================================================
-- Claims Helpers
-- ============================================================================

-- Check if a JWT's exp claim has passed (token is expired).
-- Takes the current Unix timestamp (caller provides via std/time or Clock effect).
--
-- Args:
--   claims: Json - The JWT payload
--   nowUnix: int - Current Unix timestamp in seconds
--
-- Returns:
--   bool - true if expired (exp < now), false if still valid
export pure func isExpired(claims: Json, nowUnix: int) -> bool = {
  match getInt(claims, "exp") {
    Some(exp) => exp < nowUnix,
    None => true  -- No exp claim = treat as expired (safe default)
  }
}

-- Check that the issuer matches an expected value.
--
-- Args:
--   claims: Json - The JWT payload
--   expectedIss: string - Expected issuer string
--
-- Returns:
--   bool - true if iss matches
export pure func checkIssuer(claims: Json, expectedIss: string) -> bool = {
  match getString(claims, "iss") {
    Some(iss) => iss == expectedIss,
    None => false
  }
}

-- Check that the audience contains an expected value.
-- Handles both string and array aud claims (per RFC 7519 §4.1.3).
--
-- Args:
--   claims: Json - The JWT payload
--   expectedAud: string - Expected audience string
--
-- Returns:
--   bool - true if aud matches or contains the expected value
export pure func checkAudience(claims: Json, expectedAud: string) -> bool = {
  match get(claims, "aud") {
    Some(JString(aud)) => aud == expectedAud,
    Some(JArray(auds)) => containsString(auds, expectedAud),
    _ => false
  }
}

-- Helper: check if a Json array contains a specific string
pure func containsString(xs: [Json], target: string) -> bool = {
  match xs {
    [] => false,
    [JString(s), ...rest] => if s == target then true else containsString(rest, target),
    [_, ...rest] => containsString(rest, target)
  }
}

-- Helper: get int value from JSON (duplicated from std/json to avoid import complexity)
pure func getInt(obj: Json, key: string) -> Option[int] = {
  match get(obj, key) {
    Some(JNumber(n)) => Some(floatToInt(n)),
    _ => None
  }
}
```

---

## Firebase JWT Verification — End-to-End Example

```ailang
-- firebase_auth.ail: Verify Firebase ID tokens in AILANG
module firebase_auth

import std/jwt (verifyWithKid, isExpired, checkIssuer, checkAudience)
import std/json (Json, decode, getString)
import std/result (Result, Ok, Err, flatMap)
import std/net (httpRequest)
import std/time (nowUnix)

-- Firebase public key endpoint
let firebaseKeysURL = "https://www.googleapis.com/robot/v1/metadata/x509/securetoken@system.gserviceaccount.com"

-- Verify a Firebase ID token
-- Requires Net effect (to fetch public keys) and Clock effect (for expiry check)
export func verifyFirebaseToken(
  token: string,
  projectId: string
) -> Result[Json, string] ! {Net, Clock} = {
  -- 1. Fetch Firebase public keys
  let keysResponse = httpRequest("GET", firebaseKeysURL, [], "") in
  match keysResponse {
    Ok(resp) =>
      if resp.ok then
        match decode(resp.body) {
          Ok(keys) =>
            -- 2. Verify JWT signature using kid header
            flatMap(\claims.
              -- 3. Validate standard claims
              let now = nowUnix() in
              if isExpired(claims, now) then
                Err("token expired")
              else if not(checkIssuer(claims, "https://securetoken.google.com/" ++ projectId)) then
                Err("invalid issuer")
              else if not(checkAudience(claims, projectId)) then
                Err("invalid audience")
              else
                Ok(claims)
            , verifyWithKid(token, keys)),
          Err(e) => Err("failed to parse keys: " ++ e)
        }
      else
        Err("failed to fetch keys: HTTP " ++ show(resp.status)),
    Err(e) => Err("network error fetching keys: " ++ show(e))
  }
}

export func main() -> () ! {IO, Net, Clock} = {
  let token = "eyJhbGciOiJSUzI1NiIsImtpZCI6Ii4uLiJ9.eyJzdWIiOiIxMjM0NTY3ODkwIn0.signature" in
  match verifyFirebaseToken(token, "my-project-id") {
    Ok(claims) => {
      println("Authenticated!")
      match getString(claims, "sub") {
        Some(uid) => println("Firebase UID: " ++ uid),
        None => println("No sub claim")
      }
    },
    Err(e) => println("Auth failed: " ++ e)
  }
}
```

---

## Implementation Plan

### Phase 1: Go Builtins (~1 day)

| Task | File | LOC |
|------|------|-----|
| RSA verify builtin | `internal/builtins/crypto_rsa.go` (new) | ~120 |
| RSA verify tests | `internal/builtins/crypto_rsa_test.go` (new) | ~150 |
| Base64URL builtin | `internal/builtins/bytes.go` (modify) | ~40 |
| Base64URL tests | `internal/builtins/bytes_test.go` (modify) | ~30 |
| Update `std/crypto.ail` | `std/crypto.ail` (modify) | ~20 |
| Update `std/bytes.ail` | `std/bytes.ail` (modify) | ~15 |

### Phase 2: AILANG JWT Module (~1 day)

| Task | File | LOC |
|------|------|-----|
| JWT module | `std/jwt.ail` (new) | ~180 |
| JWT example | `examples/runnable/jwt_verification.ail` (new) | ~40 |
| Firebase example | `examples/runnable/firebase_auth.ail` (new) | ~50 |
| Update std/README.md | `std/README.md` (modify) | ~5 |

### Phase 3: Testing & Docs (~0.5 day)

| Task | File | LOC |
|------|------|-----|
| Integration test | `internal/pipeline/jwt_test.go` (new) | ~100 |
| CHANGELOG.md | `CHANGELOG.md` (modify) | ~10 |
| Move design doc | `design_docs/implemented/v0_9_5/` | — |

### Files Summary

**New files (5):**
- `internal/builtins/crypto_rsa.go` — RSA verify builtin
- `internal/builtins/crypto_rsa_test.go` — RSA verify tests
- `std/jwt.ail` — JWT parsing and verification module
- `examples/runnable/jwt_verification.ail` — JWT example
- `examples/runnable/firebase_auth.ail` — Firebase auth example

**Modified files (5):**
- `internal/builtins/bytes.go` — add `fromBase64URL`
- `std/crypto.ail` — add `rsaVerifyPKCS1v15` export
- `std/bytes.ail` — add `fromBase64URL` export
- `std/README.md` — add jwt module to list
- `CHANGELOG.md` — document new features

---

## Testing Strategy

### Unit Tests (Go)

**RSA verification (`crypto_rsa_test.go`):**
- Generate RSA test keypair in test setup
- Sign a known message, verify returns `Ok(true)`
- Tamper with signature, verify returns `Ok(false)`
- Use X.509 certificate PEM, verify works
- Use PKCS#8 public key PEM, verify works
- Use PKCS#1 public key PEM, verify works
- Invalid PEM returns `Err(...)`
- Non-RSA key (e.g., EC) returns `Err(...)`
- Empty inputs handled gracefully

**Base64URL (`bytes_test.go`):**
- Standard test vectors: `""` → `""`, `"f"` → `"Zg"`, `"fo"` → `"Zm8"`
- URL-safe characters: `+/` → `-_` alphabet
- No padding required (raw encoding)
- Invalid input returns `None`

### Integration Tests (AILANG)

**JWT parsing (`jwt_test.go`):**
- Parse a known JWT (from jwt.io test vectors), verify header/payload fields
- Parse JWT with missing segment → error
- Parse JWT with invalid base64 → error
- `verifyRS256` with valid token + correct key → `Ok(claims)`
- `verifyRS256` with valid token + wrong key → `Err("invalid signature")`
- `verifyRS256` with HS256 token → `Err("unsupported algorithm...")`
- `verifyWithKid` selects correct key from set
- `verifyWithKid` with unknown kid → `Err("unknown kid...")`
- `isExpired` with future exp → `false`
- `isExpired` with past exp → `true`
- `checkIssuer` / `checkAudience` with matching and non-matching values

### Example Verification

- `make verify-examples` passes with new example files
- `make test` passes with no regressions

---

## Success Criteria

- [ ] `rsaVerifyPKCS1v15(msg, sig, pem)` verifies valid RSA-SHA256 signatures
- [ ] `rsaVerifyPKCS1v15` works with X.509 certificate PEM (Firebase format)
- [ ] `fromBase64URL` decodes JWT segments correctly (no padding)
- [ ] `decodeJWT(token)` parses well-formed JWTs into header + payload + signature
- [ ] `verifyRS256(token, pem)` combines decode + verify in one call
- [ ] `verifyWithKid(token, keys)` selects key by kid claim
- [ ] `isExpired`, `checkIssuer`, `checkAudience` helpers work correctly
- [ ] Firebase auth example compiles and type-checks
- [ ] All existing tests pass (`make test`)
- [ ] All examples valid (`make verify-examples`)
- [ ] No new Go dependencies added (uses only stdlib `crypto/*`, `encoding/*`)

---

## Risks & Mitigations

| Risk | Impact | Status | Mitigation |
|------|--------|--------|-----------|
| Type builder lacks `T.Result()` method | Medium | **Confirmed** | Construct manually: `T.TypeApp("Result", T.Bool(), T.String())` — see `builder_test.go:457` for pattern |
| Record type in `DecodedJWT` may hit M-TYPEENV-SUB bug | High | Open | The M-TYPEENV-SUB fix is in progress; if not landed, return `Json` instead of record type |
| `std/time.nowUnix` doesn't exist | Low | **Confirmed** | Already mitigated — `isExpired` takes `nowUnix: int` as argument; caller provides timestamp |
| ~~`std/string.split` doesn't exist~~ | — | **Resolved** | Exists: `std/string.split(s, delimiter)` at `std/string.ail:40` |
| ~~AILANG doesn't have `not()` builtin~~ | — | **Resolved** | Exists: logic builtin registered in `internal/builtins/math_logic.go` |

---

## Related Documents

- [M-STDLIB-CRYPTO: Cryptographic Hashing](../../planned/v0_9_4/m-stdlib-crypto.md) — Existing symmetric crypto (SHA-256, HMAC). This doc extends it.
- [M-STDLIB-CRYPTO Sprint Plan](../../planned/v0_9_4/m-stdlib-crypto-sprint-plan.md) — Implementation reference for crypto builtins
- [Firebase Auth (Go server)](../../../internal/server/auth/auth.go) — Existing JWT verification in Go, not exposed to AILANG
- [std/crypto.ail](../../../std/crypto.ail) — Current crypto module to extend
- [std/bytes.ail](../../../std/bytes.ail) — Current bytes module to extend
- [std/json.ail](../../../std/json.ail) — JSON ADT used for JWT claims
- [crypto_rsa.go pattern](../../../internal/builtins/crypto.go) — Builtin registration pattern to follow

---

**Document created**: 2026-03-27
**Last updated**: 2026-03-27
