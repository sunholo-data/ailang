# M-CLOUD-DUAL-AUTH: OAuth vs API Key Cloud Run Jobs

**Status**: Planned
**Target**: v0.9.2
**Priority**: P2 (Medium — enables external user workloads)
**Estimated**: 1 day
**Dependencies**: M-CLOUD-OAUTH (implemented), Terraform `agent-executor-apikey` job (done), KMS keyring (done)
**Source**: ailang-multivac messages 0bd05fb4, a2f552f8

## Axiom Compliance

**Canonical reference:** [Design Axioms](/docs/references/axioms)

### Axiom Scoring

| Axiom | Score | Justification |
|-------|-------|---------------|
| A1: Determinism | 0 | No semantic changes |
| A2: Replayability | 0 | Traces unchanged |
| A3: Effect Legibility | 0 | No effect system changes |
| A4: Explicit Authority | +2 | Explicit auth mode selection, no silent fallbacks |
| A5: Bounded Verification | 0 | Type checking unchanged |
| A6: Safe Concurrency | 0 | Thread-safe cache with sync.RWMutex |
| A7: Machines First | +1 | External agents can use own API keys |
| A8: Minimal Syntax | 0 | No syntax changes |
| A9: Cost Visibility | +1 | API key mode = user-visible pay-per-token costs |
| A10: Composability | 0 | No semantic changes |
| A11: Structured Failure | +1 | Explicit failure when key expires, never silent fallback |
| A12: System Boundary | +2 | Clear auth boundary: keys never touch Firestore, KMS encrypted in transit |

**Net Score: +8** → **Decision: Move forward**

## Problem Statement

The coordinator currently only supports OAuth authentication for Cloud Run Jobs (M-CLOUD-OAUTH pattern). External users who provide their own Anthropic API key cannot use cloud agent execution. A second Cloud Run Job template (`agent-executor-apikey`) already exists in Terraform but the Go dispatch layer has no way to select it or pass the key.

## Design

### Auth Flow

```
Client POST /api/messages {anthropic_api_key: "sk-..."}
  → Coordinator encrypts key with Cloud KMS (AILANG_KMS_KEY)
  → Encrypted key stored in memory cache (10min TTL, keyed by message_id)
  → Task created from message
  → dispatchTasksCloud() retrieves encrypted key from cache
  → Dispatcher selects job template: agent-executor-apikey
  → Injects ANTHROPIC_API_KEY=ENC:base64... + AILANG_AUTH_MODE as env overrides
  → Cloud Audit Logs see only ciphertext (not plaintext key)
  → Cloud Run Job starts with apikey auth
  → claude.go sees AILANG_AUTH_MODE=apikey, detects ENC: prefix
  → Decrypts with Cloud KMS (agent SA has decrypter role)
  → Sets decrypted ANTHROPIC_API_KEY in env
  → Claude Code reads ANTHROPIC_API_KEY natively
```

### Security Invariants

1. API keys NEVER touch Firestore — only `auth_mode` flag persisted in task metadata
2. API keys NEVER touch Secret Manager — runtime env overrides only
3. In-memory cache: 10min TTL, thread-safe, entries deleted after retrieval
4. Coordinator is single-instance — cache consistency guaranteed
5. Fail explicitly if key expires from cache — never fall back to OAuth
6. API keys encrypted with Cloud KMS before passing as env var overrides
7. Cloud Audit Logs see only ciphertext (`ENC:base64...`), never plaintext
8. Coordinator SA: `cloudkms.cryptoKeyEncrypter` (encrypt only, cannot read back)
9. Agent SA: `cloudkms.cryptoKeyDecrypter` (decrypt only, cannot forge)
10. KMS key auto-rotates every 90 days

## Changes

### 1. `internal/coordinator/cloud_dispatcher.go` — Add fields to DispatchParams

```go
AuthMode string // "oauth" (default) or "apikey"
APIKey   string // User-provided key, only when AuthMode == "apikey"
```

### 2. `internal/dispatch/cloudrun/dispatcher.go` — Job selection + key injection

- Select job suffix based on `params.AuthMode`
- Inject `ANTHROPIC_API_KEY` and `AILANG_AUTH_MODE` env overrides for apikey mode

### 3. `internal/coordinator/apikey_cache.go` — New file: thread-safe TTL cache

- `sync.RWMutex` + `map[string]cacheEntry`
- `Store(messageID, key)` — stores with timestamp
- `Retrieve(messageID)` — returns key and deletes entry (one-time use)
- Background cleanup goroutine for expired entries

### 4. `internal/coordinator/daemon_http.go` — Accept API key in REST body

- Add `AnthropicAPIKey string` to `postMessageRequest`
- Store in cache after message insertion (keyed by message ID)
- Set `auth_mode: apikey` in task metadata

### 5. `internal/coordinator/daemon.go` — Add cache to Daemon struct

### 6. `internal/coordinator/daemon_tasks_exec.go` — Retrieve key at dispatch time

- Check task metadata for `auth_mode == "apikey"`
- Retrieve from cache, fail explicitly if expired

### 7. `internal/executor/claude/claude.go` — Branch on AILANG_AUTH_MODE

- If `apikey`: skip `writeCredentialsFile()`, verify `ANTHROPIC_API_KEY` is set
- If `oauth` (default): existing behavior unchanged
- Don't strip `ANTHROPIC_API_KEY` when in apikey mode

### 8. `internal/coordinator/agent_registry.go` — Optional per-agent default

- Add `AuthMode string` field for agents that always use one mode

### 9. `internal/coordinator/kms.go` — KMS encrypt helper (new file)

- `EncryptAPIKey(ctx, plaintext) (string, error)` — encrypts with `AILANG_KMS_KEY`
- Returns `"ENC:" + base64(ciphertext)`
- No-op if `AILANG_KMS_KEY` not set (local dev: plaintext passthrough)
- Coordinator SA has `roles/cloudkms.cryptoKeyEncrypter`

### 10. `internal/executor/claude/kms.go` — KMS decrypt helper (new file)

- `DecryptAPIKey(ctx, encrypted) (string, error)` — decrypts `ENC:`-prefixed values
- Passes through values without `ENC:` prefix (backwards compat / local dev)
- Agent SA has `roles/cloudkms.cryptoKeyDecrypter`

### 11. New dependency: `cloud.google.com/go/kms`

## Testing

- Unit test: apikey_cache.go (store, retrieve, expiry, one-time use)
- Unit test: cloudrun dispatcher selects correct job name per auth mode
- Unit test: env var injection includes ANTHROPIC_API_KEY for apikey mode
- Unit test: claude executor skips credentials file in apikey mode
- Unit test: KMS encrypt returns `ENC:` prefixed base64 (mock client)
- Unit test: KMS decrypt strips `ENC:` prefix and decrypts (mock client)
- Unit test: plaintext passthrough when `AILANG_KMS_KEY` not set
