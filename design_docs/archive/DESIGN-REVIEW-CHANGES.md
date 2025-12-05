# Design Review Changes: M-CLAUDE-CODE-INTEGRATION-V2

**Date**: October 25, 2025
**Reviewer**: External Design Review
**Status**: Feedback Incorporated

## Summary

The original M-CLAUDE-CODE-INTEGRATION design was **validated as the right direction** but needed tightening and hardening before implementation. This document summarizes the key changes made based on the review feedback.

## Key Changes

### 1. Provider-Agnostic Bridge Layer ✅

**Original**: Claude Code hooks directly integrated into agent protocol
**Improved**: Introduced `InteractiveEvent` abstraction layer

**Impact**:
- Claude-specific code isolated to `scripts/hooks/*.sh` adapters
- All downstream logic speaks only `InteractiveEvent` + Agent Protocol
- Future providers (Gemini Code, VSCode) require only adapter changes
- Reduces coupling, improves testability

**Code**:
```go
// New abstraction in internal/agentprotocol/event.go
type InteractiveEvent struct {
    SessionID   string
    UserID      string
    Event       string
    Timestamp   time.Time
    Artifacts   []ArtifactRef  // Content-addressed
    Notes       string
}
```

### 2. Content-Addressed Artifacts ✅

**Original**: Large content embedded in message payloads
**Improved**: Hash-addressed artifact storage with references

**Impact**:
- No inline blobs in messages (solves inbox bloat)
- Deduplication (same file stored once)
- Verification contracts trivial (compare hashes)
- Artifact storage: `.ailang/state/artifacts/sha256/<hash>/`

**Example**:
```json
// Before: 5000 lines of markdown in payload
// After: Reference with hash
{
  "artifacts": [
    {
      "path": "design_docs/planned/M-FIX-123.md",
      "hash": "sha256:a1b2c3d4e5f6...",
      "mime_type": "text/markdown"
    }
  ]
}
```

### 3. At-Least-Once Delivery with Idempotency ✅

**Original**: No explicit delivery guarantees
**Improved**: Idempotent message processing with DLQ

**Impact**:
- `message_id` as idempotency key (persisted in SQLite)
- Re-processing same `message_id` is safe (duplicate detection)
- Dead Letter Queue for irrecoverable failures
- Added envelope fields: `ttl_seconds`, `deadline`, `attempt`

**Implementation**:
```sql
-- New SQLite constraint
CREATE TABLE processed_messages (
    message_id TEXT PRIMARY KEY,
    processed_at DATETIME DEFAULT CURRENT_TIMESTAMP
);
```

### 4. Correlation That Survives Restarts ✅

**Original**: Basic `correlation_id` only
**Improved**: Full distributed tracing support

**Impact**:
- `trace_id` - Workflow-level tracing (spans entire flow)
- `parent_message_id` - Message threading
- `protocol_version` / `schema_version` - Feature negotiation
- Enables debugging, observability, and future distributed tracing

**Added to envelope**:
```json
{
  "trace_id": "trace-abc123",
  "parent_message_id": "msg-xyz",
  "protocol_version": "v1",
  "schema_version": "v1"
}
```

### 5. HMAC Message Signing ✅

**Original**: No message authentication
**Improved**: HMAC signing with key ID

**Impact**:
- Prevents inbox spoofing (agents must have valid key)
- Agent allowlist enforcement
- Key rotation support via `kid` (key ID)

**Implementation**:
- `internal/agentprotocol/signing.go` (~150 LOC)
- Signing key: `AILANG_SIGNING_KEY` env var or auto-generated
- Envelope includes: `kid`, `hmac_alg`, `signature`

### 6. User Inbox with Read/Unread/Archive ✅

**Original**: Basic user inbox directory
**Improved**: Three-state inbox with lifecycle management

**Impact**:
- `_unread/` - New notifications
- `_read/` - Acknowledged messages
- `_archive/` - Old messages (kept for history)
- Prevents clutter, supports retention policies

**Structure**:
```
.ailang/state/messages/inbox/user/
├── _unread/   # New messages
├── _read/     # Displayed messages
└── _archive/  # Old messages (30 day retention)
```

### 7. Observability from Day One ✅

**Original**: No metrics or monitoring
**Improved**: SQLite metrics table + `ailang agent top` CLI

**Impact**:
- Counters: `handoff_latency_ms`, `messages_in/out`, `dlq_count`, `retry_count`
- `ailang agent top` - View queue sizes, last errors, metrics
- Foundation for future Prometheus/Grafana integration

**Metrics table**:
```sql
CREATE TABLE metrics (
    metric_name TEXT NOT NULL,
    value INTEGER NOT NULL,
    timestamp DATETIME DEFAULT CURRENT_TIMESTAMP
);
```

### 8. Platform-Safe FS Operations ✅

**Original**: Direct `os.Rename` (unsafe on Windows)
**Improved**: Abstract FS ops with platform-specific implementations

**Impact**:
- Atomic writes work on Windows/macOS/Linux
- Helper: "write temp → fsync → atomic replace (or copy+swap)"
- Avoids `flock` (uses SQLite lease table instead)

**Deferred**: Windows testing to v0.3.21 (test macOS/Linux first)

### 9. MVP Scope Reduction ✅

**Original**: 2 weeks, all hooks, headless mode, cron library
**Improved**: 2-4 days, Stop hook only, simple bash scripts

**Deferred to v0.3.21:**
- Streaming JSON handling
- Complex cron library
- Tool usage telemetry (PostToolUse hook)
- Multi-provider adapters (interface exists, only Claude implemented)
- Windows support
- Headless mode

**Impact**:
- ~60% scope reduction (2 weeks → 2-4 days)
- MVP proves core value (interactive → autonomous → user loop)
- Learn from real usage before adding complexity

### 10. Testable Headless Wrappers (Deferred) ⏳

**Original**: Shell scripts only
**Improved**: Go facade with fake for tests (when implemented in v0.3.21)

**Future design**:
```go
// Testable interface (v0.3.21)
type ClaudeRunner interface {
    RunHeadless(prompt, tools, outFormat string) (*Result, string, error)
}

// Default impl calls shell
type ShellClaudeRunner struct {}

// Test impl fakes execution
type FakeClaudeRunner struct {}
```

## Success Metrics (Tightened)

| Metric | Original | Hardened |
|--------|----------|----------|
| Handoff latency | <5s | <5s median in 100 test runs |
| Message delivery | No spec | **100%** in 1000-message soak test |
| Lost messages | No spec | **0 lost**, DLQ captures irrecoverables |
| E2E workflow | <30 min | <30 min with **artifact hash verification** |
| Artifacts | Inline blobs | **Hash-addressed** only |
| Security | No spec | **HMAC signing + agent allowlist** |
| Rate limiting | No spec | **10 notifications/hour to user** |

## Code Size Impact

| Category | Original Estimate | Hardened Estimate | Change |
|----------|-------------------|-------------------|--------|
| New files | ~500 LOC (5 scripts) | **~1,120 LOC (7 files)** | +124% |
| Modified files | ~200 LOC | **~500 LOC** | +150% |
| Tests | ~300 LOC | **Included in files** | Integrated |
| Documentation | ~1,500 LOC | **~550 LOC** | -63% (MVP focus) |
| **Total** | **~2,500 LOC** | **~1,620 LOC** | **-35%** (MVP reduction) |

**Note**: Total LOC reduced due to MVP scope cut (no headless, fewer hooks), but core implementation is more robust.

## Risk Mitigation Improvements

| Risk | Original Mitigation | Hardened Mitigation |
|------|---------------------|---------------------|
| Hook fragility | Native integration | **InteractiveEvent adapter (isolation)** |
| Message storms | No spec | **Per-agent rate caps + backpressure** |
| Inbox spoofing | No spec | **HMAC + agent allowlist** |
| Platform variance | No spec | **Abstract FS ops, test macOS/Linux** |
| Scope creep | No spec | **Strict MVP scope, defer everything not in Phase 1-4** |

## Testing Improvements

**Original**: Basic unit/integration tests
**Hardened**: Production-grade test strategy

**Added tests**:
1. **Idempotency tests** - Restart agent mid-processing, verify no duplicate work
2. **DLQ tests** - Send malformed messages, verify ends in `dead_letter/`
3. **Hash verification tests** - Corrupted artifact content, verify detection
4. **HMAC tests** - Forged messages, verify rejection
5. **Rate limiting tests** - >10 notifications/hour, verify throttling
6. **Soak tests** - 1000 messages, verify 100% delivery

## Future Work (Deferred)

**Deferred to v0.3.21+ with time estimates:**
- Headless mode integration (~2 days)
- Additional hooks (~1 day per hook)
- Multi-provider support (~2 days)
- Cross-machine transport (~1 week)
- Advanced observability (~2 days)

## Decision Summary

✅ **Proceed with MVP** (2-4 days)

**Why**:
- Core value proven (interactive → autonomous → user loop)
- Production-grade guarantees (100% delivery, idempotency, HMAC)
- Provider-agnostic design (future-proof)
- Reduced scope (learn before scaling)

**What's different**:
- Tighter scope (~60% reduction)
- Stronger guarantees (delivery, security, observability)
- Better architecture (adapters, content-addressed artifacts)
- Clear future path (v0.3.21+ roadmap)

---

**Next Steps**:
1. User review and approval of M-CLAUDE-CODE-INTEGRATION-V2.md
2. Phase 1 implementation (hook adapter + artifact storage)
3. Phase 2 implementation (user inbox)
4. Testing and validation
5. Documentation and release

**Comparison docs**:
- Original: [M-CLAUDE-CODE-INTEGRATION.md](../M-CLAUDE-CODE-INTEGRATION.md)
- Hardened: [m-claude-code-integration-v2.md](m-claude-code-integration-v2.md)
