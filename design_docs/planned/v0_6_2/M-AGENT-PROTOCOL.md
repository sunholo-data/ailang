# M-AGENT-PROTOCOL: Agent-to-Agent Communication Protocol

**Status:** 📋 Planned (Bootstrap Task) — Revised v2
**Priority:** P0 (Critical)
**Complexity:** M (3-4 days with hardening)
**Target Version:** v0.4.0
**Created:** 2025-10-23
**Revised:** 2025-10-23 (production-ready guardrails)
**Author:** ailang-dev-cycle meta-agent (self-assigned)

---

## Problem Statement

**Current limitation:** Agents and skills cannot communicate with each other directly.

**Pain points:**
1. **No shared state** — Each agent invocation is stateless, no memory across runs
2. **No skill-to-skill messaging** — Skills invoke via Claude, no direct communication
3. **No coordination primitives** — Cannot synchronize, broadcast, or negotiate
4. **No verification contracts** — Agents can't verify each other's work programmatically
5. **Pause points require humans** — Approval gates block automation

**Impact:**
- Meta-agent (ailang-dev-cycle) cannot run fully autonomous
- Multi-agent collaboration (VISION.md) requires human intermediary
- Cannot implement "AIs that reason, refactor, and verify" (VISION.md:3-4)

**Quote from VISION.md (lines 152-202):**
> "Once reflection and normalization are stable, AILANG will enable **multi-agent cooperation**... Agent B (Verifier) checks the refactoring... Both agents agree on equivalence via deterministic hashing and effect tracking."

**Current blocker:** No protocol for Agent A to send code to Agent B, or for Agent B to return verification results.

---

## Proposed Solution

**Create a production-ready agent coordination protocol using:**
1. **File-based message passing** (`.ailang/state/messages/`) — Observable transport
2. **SQLite control plane** (`.ailang/state/agents.db`) — Dedupe, leases, history, metrics
3. **At-least-once delivery + idempotency** — Crash-safe with message_id deduplication
4. **Test-based verification contracts** — Leverage AILANG's test suite
5. **Retry-with-backoff + DLQ** — Resilient error handling

**Design principles (strengthened):**
- ✅ **Simple** — Files + SQLite, no complex infrastructure
- ✅ **Observable** — Messages are plain JSON files (ls, cat, sqlite3)
- ✅ **Crash-safe** — Atomic handoff (temp → fsync → rename), lease-based recovery
- ✅ **Idempotent** — message_id deduplication, retries don't cause double-execution
- ✅ **Deterministic** — Message order guaranteed by timestamps
- ✅ **Effect-typed** — Messages declare their effects (IO, FS, Net, etc.)
- ✅ **Verifiable** — SHA256 hashes + test results provide proof
- ✅ **Evolvable** — Protocol version + schema version + capability negotiation
- ✅ **Auditable** — All transitions logged to agent_history
- ✅ **Self-improving** — Agents report DX friction, propose language improvements (KEY KPI!)

---

## Architecture

### Core Principle: Files = Transport, SQLite = Control Plane

**Files:** Observable queue, payload medium
**SQLite:** Dedupe, leases, history, metrics, verification records

**Clean separation ensures:**
- Observable debugging (ls .ailang/state/messages/)
- ACID guarantees (SQLite transactions)
- Recovery from crashes (leases + reaper)

---

### 1. Message Envelope (Enhanced)

**Location:** `.ailang/state/messages/`

**Message format:**
```json
{
  "protocol_version": "1.0.0",
  "schema_version": "1.0.0",
  "message_id": "msg_20251023_143022_abc123",
  "correlation_id": "cycle_20251023_001",
  "trace_id": "trace_xyz789",
  "parent_message_id": null,
  "timestamp": "2025-10-23T14:30:22Z",
  "ttl_seconds": 3600,
  "deadline": "2025-10-23T15:30:22Z",
  "from_agent": "ailang-dev-cycle",
  "to_agent": "eval-orchestrator",
  "message_type": "request",
  "retries": 0,
  "payload_schema": "https://ailang.dev/schemas/run_eval_baseline/v1.json",
  "payload": {
    "action": "run_eval_baseline",
    "params": {
      "version": "v0.3.15",
      "models": ["gpt5-mini", "claude-haiku-4-5", "gemini-2-5-flash"]
    }
  },
  "declared_effects": ["IO", "FS"],
  "signature_alg": "hmac-sha256",
  "kid": "agent-key-2025-10",
  "signature": "hmac:abc123..."
}
```

**Key additions:**
- **schema_version** — Evolution of envelope itself (separate from protocol_version)
- **trace_id / parent_message_id** — Distributed tracing across agents
- **ttl_seconds / deadline** — Prevent undead work
- **retries** — Sender-maintained counter for exponential backoff
- **payload_schema** — URI for independent schema evolution
- **kid / signature_alg** — Key rotation support
- **declared_effects** (renamed from effects_declared) — Consistency with AILANG conventions

---

### 2. Message Lifecycle (Crash-Safe)

**States:**
1. **pending** — Waiting for receiver
2. **processing** — Receiver is working on it (has lease)
3. **completed** — Successfully processed (archived)
4. **failed** — Exhausted retries or unrecoverable error (dead-letter)

**Handoff protocol (atomic):**
```bash
# Sender creates message
1. Write to /tmp/msg_123.tmp
2. fsync (flush to disk)
3. Atomic rename: mv msg_123.tmp .ailang/state/messages/to_agent/msg_123.pending.json

# Receiver acquires lease
4. INSERT INTO agent_locks (resource_id='msg_123', locked_by='receiver', expires_at=NOW()+60s)
5. Atomic rename: mv msg_123.pending.json msg_123.processing.json

# Receiver completes
6. Write response to from_agent/msg_123.response.json
7. DELETE FROM agent_locks WHERE resource_id='msg_123'
8. Atomic rename: mv msg_123.processing.json archive/msg_123.completed.json
9. INSERT INTO agent_history (event_type='message_completed', ...)
```

**Crash recovery (reaper process):**
```sql
-- Find orphaned .processing.json with expired leases
SELECT resource_id FROM agent_locks
WHERE expires_at < NOW() AND resource_id LIKE 'msg_%';

-- Move back to pending OR dead-letter if max_retries exceeded
-- Record in agent_history with reason='lease_expired'
```

**Directory structure:**
```
.ailang/state/messages/
├── ailang-dev-cycle/
│   ├── msg_001.pending.json       # Outgoing request
│   ├── msg_002.response.json      # Incoming response
│   ├── archive/
│   │   └── msg_001.completed.json # Completed conversation
│   └── dead_letter/
│       └── msg_003.failed.json    # Exhausted retries
├── eval-orchestrator/
│   ├── msg_001.processing.json    # Currently handling (has lease)
│   └── archive/
└── eval-fix-implementer/
    └── (similar structure)
```

---

### 3. SQLite Schema (Enhanced)

**Location:** `.ailang/state/agents.db`

**Schema:**
```sql
-- Agents registry (discovery + capabilities)
CREATE TABLE agents (
    agent_id TEXT PRIMARY KEY,
    inbox_path TEXT NOT NULL,
    status TEXT CHECK(status IN ('idle', 'active', 'paused', 'error')) NOT NULL,
    protocol_caps TEXT NOT NULL,  -- JSON array: ["v1.0", "hmac", "streaming"]
    last_heartbeat TIMESTAMP NOT NULL,
    created_at TIMESTAMP NOT NULL DEFAULT CURRENT_TIMESTAMP
);

-- Agent state (persistent memory)
CREATE TABLE agent_state (
    agent_id TEXT PRIMARY KEY,
    state_version INTEGER NOT NULL,
    schema_version TEXT NOT NULL,  -- For state evolution
    last_active TIMESTAMP NOT NULL,
    current_task TEXT,
    status TEXT CHECK(status IN ('idle', 'active', 'paused', 'error')),
    state_json TEXT NOT NULL,  -- JSON blob of agent-specific state
    checksum TEXT NOT NULL,    -- SHA256 of state_json
    FOREIGN KEY(agent_id) REFERENCES agents(agent_id)
);

-- Message registry (deduplication + tracking)
CREATE TABLE messages (
    message_id TEXT PRIMARY KEY,
    correlation_id TEXT,
    trace_id TEXT,
    from_agent TEXT NOT NULL,
    to_agent TEXT NOT NULL,
    status TEXT CHECK(status IN ('pending', 'processing', 'completed', 'failed')) NOT NULL,
    created_at TIMESTAMP NOT NULL,
    processed_at TIMESTAMP,
    retries INTEGER NOT NULL DEFAULT 0,
    payload_hash TEXT NOT NULL,  -- SHA256 of payload for deduplication
    FOREIGN KEY(from_agent) REFERENCES agents(agent_id),
    FOREIGN KEY(to_agent) REFERENCES agents(agent_id)
);

CREATE INDEX idx_messages_correlation ON messages(correlation_id);
CREATE INDEX idx_messages_trace ON messages(trace_id);
CREATE INDEX idx_messages_status ON messages(status);

-- Agent history (audit log)
CREATE TABLE agent_history (
    id INTEGER PRIMARY KEY AUTOINCREMENT,
    agent_id TEXT NOT NULL,
    message_id TEXT,
    timestamp TIMESTAMP NOT NULL DEFAULT CURRENT_TIMESTAMP,
    event_type TEXT NOT NULL,
    event_data TEXT NOT NULL,  -- JSON blob
    FOREIGN KEY(agent_id) REFERENCES agents(agent_id),
    FOREIGN KEY(message_id) REFERENCES messages(message_id)
);

CREATE INDEX idx_history_agent_time ON agent_history(agent_id, timestamp);
CREATE INDEX idx_history_message ON agent_history(message_id);

-- Resource locks (leases for crash recovery)
CREATE TABLE agent_locks (
    resource_id TEXT PRIMARY KEY,
    locked_by TEXT NOT NULL,
    locked_at TIMESTAMP NOT NULL DEFAULT CURRENT_TIMESTAMP,
    expires_at TIMESTAMP NOT NULL,
    FOREIGN KEY(locked_by) REFERENCES agents(agent_id)
);

CREATE INDEX idx_locks_expires ON agent_locks(expires_at);

-- Verification results
CREATE TABLE verification_results (
    id INTEGER PRIMARY KEY AUTOINCREMENT,
    verifier_agent TEXT NOT NULL,
    target_agent TEXT NOT NULL,
    artifact_hash TEXT NOT NULL,  -- SHA256 of code/doc being verified
    artifact_path TEXT NOT NULL,  -- Content-addressed: /artifacts/sha256/<hash>
    status TEXT CHECK(status IN ('pending', 'verified', 'rejected')) NOT NULL,
    reason TEXT,
    toolchain_digest TEXT NOT NULL,  -- go_version + linter_version + test_flags
    timestamp TIMESTAMP NOT NULL DEFAULT CURRENT_TIMESTAMP,
    FOREIGN KEY(verifier_agent) REFERENCES agents(agent_id),
    FOREIGN KEY(target_agent) REFERENCES agents(agent_id)
);

CREATE INDEX idx_verification_status ON verification_results(status);
CREATE INDEX idx_verification_artifact ON verification_results(artifact_hash);

-- Metrics (observability)
CREATE TABLE agent_metrics (
    agent_id TEXT NOT NULL,
    metric_name TEXT NOT NULL,
    metric_value REAL NOT NULL,
    timestamp TIMESTAMP NOT NULL DEFAULT CURRENT_TIMESTAMP,
    FOREIGN KEY(agent_id) REFERENCES agents(agent_id)
);

CREATE INDEX idx_metrics_agent_time ON agent_metrics(agent_id, timestamp);
```

**SQLite configuration:**
```sql
-- Enable WAL mode for concurrent reads + single writer
PRAGMA journal_mode = WAL;

-- Store schema version for migrations
PRAGMA user_version = 1;

-- Foreign key enforcement
PRAGMA foreign_keys = ON;
```

**Migration strategy:**
```go
// Example migration from v1 → v2
func MigrateV1ToV2(db *sql.DB) error {
    version := getPragmaUserVersion(db)
    if version == 1 {
        db.Exec("ALTER TABLE agents ADD COLUMN protocol_caps TEXT DEFAULT '[]'")
        db.Exec("PRAGMA user_version = 2")
    }
    return nil
}
```

---

### 4. Delivery Semantics (At-Least-Once + Idempotency)

**Guarantee:** At-least-once delivery with idempotency ensures messages are never lost but may be retried.

**Idempotency key:** `message_id` is globally unique and used for deduplication.

**Sender side:**
```go
func SendMessage(to string, payload Payload) (*Response, error) {
    msg := Envelope{
        MessageID:   generateUUID(),  // Globally unique
        CorrelationID: currentCycleID,
        Payload:     payload,
        Retries:     0,
    }

    for attempt := 0; attempt < 3; attempt++ {
        // Check if already sent (dedupe on sender side too)
        if existsInDB(msg.MessageID) {
            return getCachedResponse(msg.MessageID), nil
        }

        // Atomic write + fsync
        writeTempFile(msg)
        fsync()
        atomicRename(tempPath, pendingPath)

        // Record in SQLite
        insertMessage(msg, status='pending')

        // Wait for response or timeout
        response, err := waitForResponse(msg.MessageID, timeout=60s)
        if err == TimeoutError {
            msg.Retries++
            backoff := time.Duration(2^attempt) * time.Second
            time.Sleep(backoff)
            continue
        }
        return response, err
    }

    return nil, ErrRetriesExhausted
}
```

**Receiver side:**
```go
func ReceiveMessage(agentID string) (*Envelope, error) {
    // Scan for .pending.json files
    files := scanDirectory(inbox(agentID))

    for _, file := range files {
        msg := parseJSON(file)

        // Deduplication check (critical!)
        if processed := checkDB(msg.MessageID); processed {
            // Already processed, return cached response
            sendCachedResponse(msg)
            archiveMessage(file)
            continue
        }

        // Acquire lease (prevents duplicate processing)
        lease := tryAcquireLease(msg.MessageID, ttl=60s)
        if !lease.Acquired {
            continue  // Another instance is processing
        }

        // Atomic rename
        atomicRename(file, processingPath)

        // Record start
        updateMessageStatus(msg.MessageID, 'processing')

        return msg, nil
    }
}

func AckMessage(msg *Envelope, response *Response) error {
    // Write response
    writeResponse(msg.FromAgent, msg.MessageID, response)

    // Release lease
    releaseLease(msg.MessageID)

    // Archive and record completion
    atomicRename(processingPath, completedPath)
    updateMessageStatus(msg.MessageID, 'completed')
    insertHistory(msg.MessageID, 'message_completed')

    return nil
}
```

**Why at-least-once?**
- Simpler than exactly-once (no distributed transactions)
- Idempotency makes retries safe
- Fits file-based transport naturally

---

### 5. Agent Discovery & Capability Negotiation

**Registry:** `agents` table provides discovery

**Example:**
```go
func ResolveAgent(name string) (*AgentInfo, error) {
    row := db.QueryRow("SELECT inbox_path, protocol_caps, status FROM agents WHERE agent_id = ?", name)

    var info AgentInfo
    row.Scan(&info.InboxPath, &info.Caps, &info.Status)

    if info.Status != "active" {
        return nil, ErrAgentNotAvailable
    }

    // Check capability compatibility
    if !supportsVersion(info.Caps, "1.0") {
        return nil, ErrIncompatibleProtocol
    }

    return &info, nil
}
```

**Capability negotiation:**
```json
{
  "protocol_caps": ["v1.0", "v1.1", "hmac-sha256", "streaming", "backpressure"]
}
```

**Feature flags (future):**
- `streaming` — Supports chunked responses for large payloads
- `backpressure` — Can send throttled responses
- `hmac-sha256` — Supports message signing

---

### 6. Backpressure & Quotas

**Receiver-side backpressure:**
```go
func CheckBackpressure(agentID string) error {
    count := countPendingMessages(agentID)

    if count > 100 {
        return ErrorResponse{
            Status: "throttled",
            RetryAfter: 30,  // seconds
            Reason: "Receiver overloaded",
        }
    }

    return nil
}
```

**Rate limiting (sender):**
```go
func EnforceRateLimit(agentID string) error {
    recentCount := countRecentMessages(agentID, window=1m)

    if recentCount > 10 {
        return ErrRateLimitExceeded
    }

    return nil
}
```

**Disk quotas:**
```bash
# Enforce max 100MB for .ailang/state/
du -sh .ailang/state/ | awk '{if ($1 > 100) exit 1}'
```

**Metrics to track:**
- `messages_in` — Incoming messages per agent
- `messages_out` — Outgoing messages per agent
- `dlq_count` — Dead-letter queue size
- `avg_latency_ms` — Average processing time
- `retry_count` — Number of retries

---

### 7. Security (Hardened)

**Message signing (HMAC-SHA256):**
```go
func SignMessage(msg *Envelope, key []byte) string {
    payload := canonicalJSON(msg.Payload)
    mac := hmac.New(sha256.New, key)
    mac.Write([]byte(msg.MessageID + msg.Timestamp + payload))
    return base64.StdEncoding.EncodeToString(mac.Sum(nil))
}

func VerifySignature(msg *Envelope, key []byte) error {
    expected := SignMessage(msg, key)
    if !hmac.Equal([]byte(msg.Signature), []byte(expected)) {
        return ErrInvalidSignature
    }
    return nil
}
```

**Key management:**
```
.ailang/state/secrets/
├── agent-key-2025-10.key  # Current key (mode 0600)
├── agent-key-2025-09.key  # Previous key (for rotation)
└── keys.json              # Key metadata: {"kid": "agent-key-2025-10", "created": "..."}
```

**Key rotation:**
- Include `kid` (key ID) in envelope
- Store multiple keys for transition period
- Rotate every 90 days (automate with agent)

**Path validation:**
```go
func ValidatePath(path string) error {
    // Prevent directory traversal
    if strings.Contains(path, "..") {
        return ErrInvalidPath
    }

    // Ensure path is within .ailang/state/
    abs, _ := filepath.Abs(path)
    if !strings.HasPrefix(abs, ".ailang/state/") {
        return ErrInvalidPath
    }

    return nil
}
```

**Content-addressed artifacts:**
```
.ailang/state/artifacts/
└── sha256/
    └── abc123def456.ail  # Artifact referenced by hash
```

**Never inline large blobs:**
```json
// ❌ BAD
{
  "payload": {
    "code": "... 100KB of code ..."
  }
}

// ✅ GOOD
{
  "payload": {
    "artifact_hash": "sha256:abc123...",
    "artifact_path": "/artifacts/sha256/abc123.ail"
  }
}
```

**Agent allowlist:**
```sql
-- Only allow known agents
INSERT INTO agents (agent_id, inbox_path, status) VALUES
  ('ailang-dev-cycle', '.ailang/state/messages/ailang-dev-cycle/', 'active'),
  ('eval-orchestrator', '.ailang/state/messages/eval-orchestrator/', 'active'),
  ...;

-- Reject messages from unknown agents
SELECT COUNT(*) FROM agents WHERE agent_id = ? AND status = 'active';
```

---

### 8. Dead-Letter Queue (DLQ)

**When to DLQ:**
- Retries exhausted (3 attempts)
- Invalid schema (can't parse)
- Timeout exceeded (past deadline)
- Unrecoverable error (e.g., missing artifact)

**Process:**
```go
func MoveToDLQ(msg *Envelope, reason string) {
    // Rename to dead_letter/
    atomicRename(processingPath, dlqPath)

    // Record in SQLite
    updateMessageStatus(msg.MessageID, 'failed')
    insertHistory(msg.MessageID, 'moved_to_dlq', reason)

    // Alert (optional)
    if critical(msg) {
        alertUser("Message failed: " + msg.MessageID)
    }
}
```

**DLQ inspection:**
```bash
# List failed messages
ls .ailang/state/messages/*/dead_letter/

# Analyze failure reasons
sqlite3 .ailang/state/agents.db \
  "SELECT message_id, event_data FROM agent_history WHERE event_type='moved_to_dlq'"
```

**Retry from DLQ (manual):**
```bash
# Resubmit failed message
mv dead_letter/msg_123.failed.json msg_123.pending.json

# Reset retries in SQLite
sqlite3 .ailang/state/agents.db \
  "UPDATE messages SET retries=0, status='pending' WHERE message_id='msg_123'"
```

---

### 9. Observability (First-Class)

**Metrics tracked:**
```sql
-- Per-agent counters
INSERT INTO agent_metrics (agent_id, metric_name, metric_value) VALUES
  ('eval-orchestrator', 'messages_in', 45),
  ('eval-orchestrator', 'messages_out', 42),
  ('eval-orchestrator', 'avg_latency_ms', 234.5),
  ('eval-orchestrator', 'retry_count', 3),
  ('eval-orchestrator', 'dlq_count', 1);
```

**CLI tool: `ailang agent top`**
```bash
$ ailang agent top

AGENT QUEUE STATUS (refreshes every 5s)

Agent                   Pending  Processing  DLQ  Avg Latency  Last Active
----------------------  -------  ----------  ---  -----------  -----------
ailang-dev-cycle             2           1    0      1.2s      2s ago
eval-orchestrator            0           0    0      0.8s      5s ago
sprint-executor              5           1    2      3.4s      1s ago
test-coverage-guardian       0           0    0      0.5s      10s ago

LAST ERRORS:
[sprint-executor] msg_456: Test failure (3 tests failed in specialize_test.go)
[sprint-executor] msg_789: Lint error (cyclomatic complexity 15 > 10)
```

**Distributed tracing:**
```bash
# Show message flow for a cycle
$ ailang agent trace cycle_20251023_001

TRACE: cycle_20251023_001
├─ msg_001: ailang-dev-cycle → design-doc-creator (2s)
│  └─ msg_002: design-doc-creator → design-spec-auditor (1s)
│     └─ msg_003: design-spec-auditor → ailang-dev-cycle (0.5s)
├─ msg_004: ailang-dev-cycle → sprint-planner (3s)
│  └─ msg_005: sprint-planner → ailang-dev-cycle (2s)
└─ msg_006: ailang-dev-cycle → sprint-executor (120s)
   └─ msg_007: sprint-executor → test-coverage-guardian (5s)
      └─ msg_008: test-coverage-guardian → sprint-executor (4s)

Total duration: 137.5s
Messages: 8
Retries: 0
Errors: 0
```

---

### 10. Verification Contracts (Enhanced)

**Goal:** Agents verify each other's work programmatically with reproducibility.

**Protocol:**
1. Producer creates artifact
2. Producer computes SHA256, stores in content-addressed path
3. Producer sends `verify_code` message with hash + toolchain digest
4. Verifier retrieves artifact by hash
5. Verifier runs tests with specified toolchain
6. Verifier records result (including toolchain used)
7. Verifier responds with verified/rejected

**Toolchain digest:**
```json
{
  "toolchain_digest": "go1.21.3_golangci-lint1.54.2_flags:-v,-race"
}
```

**Why record toolchain?**
- Makes verification **reproducible**
- Different linter versions = different results
- Enables "verify with same toolchain" guarantee

**Example: Code Verification**
```json
// Request from sprint-executor to test-coverage-guardian
{
  "message_type": "request",
  "action": "verify_code",
  "payload": {
    "artifact_hash": "sha256:abc123...",
    "artifact_path": "/artifacts/sha256/abc123.go",
    "git_commit": "def456",
    "verification_criteria": {
      "tests_must_pass": true,
      "coverage_threshold": 90.0,
      "lint_must_pass": true
    },
    "toolchain": {
      "go_version": "1.21.3",
      "linter": "golangci-lint",
      "linter_version": "1.54.2",
      "test_flags": ["-v", "-race"]
    }
  }
}

// Response from test-coverage-guardian
{
  "status": "success",
  "result": {
    "verified": true,
    "tests_passed": 12,
    "tests_failed": 0,
    "coverage": 94.2,
    "lint_issues": 0,
    "toolchain_used": "go1.21.3_golangci-lint1.54.2_flags:-v,-race"
  },
  "artifact_hash": "sha256:abc123...",  // Echo for traceability
  "verification_id": 42  // Row ID in verification_results table
}
```

**Artifact size limit:**
- Max inline: 64KB (for small snippets)
- Larger artifacts: Reference by hash only

---

### 11. DX Feedback Loop (Self-Improvement - KEY KPI!)

**Goal:** Agents analyze their own struggles with AILANG and propose language improvements to smooth rough edges.

**The Feedback Cycle:**
```
Agent struggles with AILANG
    ↓
Reports friction point
    ↓
Proposes DX improvement
    ↓
Design doc created
    ↓
Feature implemented
    ↓
Eval baseline measures improvement
    ↓
Friction reduced (measured!)
```

**Why this is critical:**
- **AI-first language** — AILANG evolves based on how AIs actually struggle, not human assumptions
- **Data-driven DX** — Friction is measured, not guessed
- **Continuous improvement** — Language gets smoother with every cycle
- **Key KPI**: Reduction in agent errors, retries, and development time per feature

---

#### 11.1 SQLite Schema Addition

```sql
-- DX friction reports (agents report struggles)
CREATE TABLE dx_friction_reports (
    id INTEGER PRIMARY KEY AUTOINCREMENT,
    agent_id TEXT NOT NULL,
    correlation_id TEXT,  -- Links to cycle that encountered friction
    timestamp TIMESTAMP NOT NULL DEFAULT CURRENT_TIMESTAMP,
    friction_type TEXT CHECK(friction_type IN (
        'syntax_error',
        'type_error',
        'missing_feature',
        'unclear_error_message',
        'verbose_boilerplate',
        'confusing_semantics',
        'tooling_gap'
    )) NOT NULL,
    ailang_version TEXT NOT NULL,  -- Version where friction occurred
    context TEXT NOT NULL,  -- JSON: {file, line, function, what_was_attempted}
    error_message TEXT,  -- Actual error from AILANG compiler/runtime
    workaround TEXT,  -- How agent worked around it (if any)
    impact TEXT CHECK(impact IN ('blocking', 'major', 'minor')) NOT NULL,
    proposed_fix TEXT,  -- Agent's suggestion for improvement
    FOREIGN KEY(agent_id) REFERENCES agents(agent_id)
);

CREATE INDEX idx_friction_type ON dx_friction_reports(friction_type);
CREATE INDEX idx_friction_impact ON dx_friction_reports(impact);
CREATE INDEX idx_friction_version ON dx_friction_reports(ailang_version);

-- DX improvements (tracking proposed vs implemented)
CREATE TABLE dx_improvements (
    id INTEGER PRIMARY KEY AUTOINCREMENT,
    friction_report_id INTEGER,  -- Links to original report
    design_doc_path TEXT,  -- e.g., "design_docs/planned/M-DX9-auto-imports.md"
    status TEXT CHECK(status IN ('proposed', 'planned', 'implemented', 'rejected')) NOT NULL,
    proposed_at TIMESTAMP NOT NULL DEFAULT CURRENT_TIMESTAMP,
    implemented_at TIMESTAMP,
    implemented_version TEXT,  -- Version where fix shipped
    FOREIGN KEY(friction_report_id) REFERENCES dx_friction_reports(id)
);

-- DX metrics (measuring impact of improvements)
CREATE TABLE dx_metrics (
    id INTEGER PRIMARY KEY AUTOINCREMENT,
    improvement_id INTEGER NOT NULL,
    metric_name TEXT NOT NULL,  -- 'compile_errors', 'avg_dev_time', 'retry_count', etc.
    before_value REAL NOT NULL,
    after_value REAL NOT NULL,
    improvement_pct REAL NOT NULL,  -- (before - after) / before * 100
    measured_at TIMESTAMP NOT NULL DEFAULT CURRENT_TIMESTAMP,
    FOREIGN KEY(improvement_id) REFERENCES dx_improvements(id)
);

CREATE INDEX idx_dx_metrics_improvement ON dx_metrics(improvement_id);
```

---

#### 11.2 Friction Reporting Protocol

**When to report friction:**
- Agent encounters compile error repeatedly (>2 retries on same issue)
- Agent finds workaround (signals missing feature)
- Agent spends >10 minutes on boilerplate
- Agent receives unclear error message
- Agent has to read source code to understand behavior

**Message format:**
```json
{
  "message_type": "notification",
  "notification_type": "dx_friction_report",
  "payload": {
    "friction_type": "missing_feature",
    "context": {
      "file": "internal/pipeline/specialize.go",
      "line": 127,
      "function": "specializeExpr",
      "what_was_attempted": "Trying to add auto-import of Prelude for entry modules",
      "code_snippet": "// Manual import injection - verbose boilerplate"
    },
    "error_message": null,
    "workaround": "Manually added 48 lines to inject imports in elaborate.go",
    "impact": "major",
    "proposed_fix": "Add compiler flag --auto-import-prelude for entry modules",
    "estimated_savings": "2 hours per similar feature, ~50 LOC reduction"
  }
}
```

**Friction types:**
- `syntax_error` — Confusing syntax, hard to remember
- `type_error` — Type system too restrictive or unclear
- `missing_feature` — Had to implement workaround
- `unclear_error_message` — Error didn't explain how to fix
- `verbose_boilerplate` — Repetitive code required
- `confusing_semantics` — Behavior didn't match expectations
- `tooling_gap` — Missing CLI tool or helper

---

#### 11.3 Automated Analysis

**After each dev cycle, run DX analysis:**
```bash
$ ailang dx analyze --since v0.3.14

DX FRICTION ANALYSIS (v0.3.14 → v0.3.15)

Top Friction Points:
1. missing_feature (5 reports, 3 blocking)
   - Auto-import prelude (3 reports)
   - Dict literal syntax (2 reports)

2. verbose_boilerplate (3 reports, 1 major)
   - Effect declarations (3 reports)

3. unclear_error_message (2 reports, 2 minor)
   - Type mismatch in pattern match (2 reports)

Proposed Fixes (generated design docs):
✅ design_docs/planned/M-DX9-auto-import-prelude.md (from friction #42)
⏳ design_docs/planned/M-DX10-dict-literal-syntax.md (from friction #43)
⏳ design_docs/planned/M-DX11-effect-inference.md (from friction #44)

Recommend prioritizing: M-DX9 (3 blocking reports, 6h savings per feature)
```

**Integration with meta-agent:**
```
Stage 1: Analyze Current State
    ↓
  [NEW] Run DX analysis
    ↓
  Prioritize friction reports by impact
    ↓
  Generate design docs for top issues
    ↓
Stage 2: Formalize Design (as usual)
```

---

#### 11.4 Friction-to-Design-Doc Pipeline

**Automated workflow:**
1. Agent reports friction (SQLite insert)
2. Meta-agent queries friction reports (Stage 1)
3. Group by similarity (NLP or rule-based)
4. Generate design doc for clustered issues
5. Include metrics: # reports, impact, estimated savings

**Example:**
```bash
# After 3 agents report "auto-import prelude" friction
$ ailang dx generate-design --friction-ids 42,43,45

Created: design_docs/planned/M-DX9-auto-import-prelude.md

Summary:
- Friction type: missing_feature
- Reports: 3 (2 blocking, 1 major)
- Estimated savings: 6 hours per similar feature
- Proposed fix: Compiler flag --auto-import-prelude
- LOC reduction: ~50 lines per feature
```

**Design doc includes:**
- **Friction Evidence**: List of reports, timestamps, agents
- **Current Workaround**: How agents currently handle it
- **Proposed Solution**: Compiler/language feature to eliminate friction
- **Success Metrics**: Expected reduction in errors, dev time, boilerplate

---

#### 11.5 Measuring Impact (KEY KPI!)

**Before implementing fix:**
```sql
-- Baseline: How often does this friction occur?
SELECT COUNT(*), AVG(impact)
FROM dx_friction_reports
WHERE friction_type = 'missing_feature'
  AND context LIKE '%auto-import%'
  AND ailang_version >= 'v0.3.14';
-- Result: 5 reports, avg impact: blocking
```

**After implementing fix (v0.3.16):**
```sql
-- After: Did friction reduce?
SELECT COUNT(*), AVG(impact)
FROM dx_friction_reports
WHERE friction_type = 'missing_feature'
  AND context LIKE '%auto-import%'
  AND ailang_version >= 'v0.3.16';
-- Result: 0 reports (friction eliminated!)

-- Insert improvement metric
INSERT INTO dx_metrics (improvement_id, metric_name, before_value, after_value, improvement_pct)
VALUES (9, 'friction_reports_auto_import', 5, 0, 100.0);
```

**Dashboard:**
```bash
$ ailang dx metrics --version v0.3.16

DX IMPROVEMENT METRICS (v0.3.16)

Friction Eliminated:
- Auto-import prelude: 5 → 0 reports (-100%) ✅
- Dict literal syntax: 3 → 0 reports (-100%) ✅

Friction Reduced:
- Effect declarations: 8 → 3 reports (-62.5%) ⚠️ (needs more work)

Development Time Savings:
- Avg time per feature: 7.5h → 5.2h (-30.7%) 🎉

Boilerplate Reduction:
- Avg LOC per feature: 250 → 180 (-28%) 🎉

Agent Success Rate:
- First-attempt compile: 45% → 68% (+51%) 🎉
```

---

#### 11.6 Example: Entry Prelude Feature (M-DX3)

**Friction reported:**
```json
{
  "friction_type": "verbose_boilerplate",
  "what_was_attempted": "Write simple AILANG program",
  "error_message": null,
  "workaround": "Manually added 'import std/prelude' to every file",
  "impact": "major",
  "proposed_fix": "Auto-import Prelude for entry modules"
}
```

**Design doc created:** `M-DX3-entry-prelude-auto-import.md`

**Implementation:** v0.3.14

**Impact measured:**
- Before: 12 friction reports about missing `print`
- After: 0 friction reports
- Dev time savings: 15 minutes per module setup
- LOC reduction: 4-8 lines per entry module

**This is the feedback loop in action!**

---

#### 11.7 Integration Points

**Meta-agent Stage 1 (enhanced):**
```go
func (m *MetaAgent) AnalyzeCurrentState() {
    // Existing: eval baselines, planned docs, roadmap
    evalBaselines := loadEvalBaselines()
    plannedDocs := loadPlannedDesignDocs()
    roadmap := loadRoadmap()

    // NEW: DX friction analysis
    frictionReports := queryFrictionReports(since=lastRelease)
    clusteredFriction := clusterSimilarReports(frictionReports)

    for cluster := range clusteredFriction {
        if cluster.Impact == "blocking" && cluster.Count >= 3 {
            // Auto-generate design doc
            designDoc := generateDesignDoc(cluster)
            saveToPlanned(designDoc)
        }
    }

    // Prioritize: friction reports influence roadmap
    prioritized := prioritizeByImpact(plannedDocs, frictionReports)
    return prioritized
}
```

**CLI commands:**
```bash
# Report friction (agents use this)
ailang dx report --type missing_feature --impact blocking \
  --context "auto-import prelude" --proposed-fix "..."

# Analyze friction (meta-agent uses this)
ailang dx analyze --since v0.3.14

# Generate design doc from friction
ailang dx generate-design --friction-ids 42,43,45

# Show metrics
ailang dx metrics --version v0.3.16

# Show friction trends
ailang dx trends --window 3months
```

---

#### 11.8 Success Criteria

**Must measure:**
- ✅ Friction reports per version (trending down)
- ✅ Time to fix friction (planned → implemented)
- ✅ Agent dev time per feature (trending down)
- ✅ Boilerplate LOC per feature (trending down)
- ✅ First-attempt compile rate (trending up)

**Target KPIs (v0.4.0 → v0.5.0):**
- Friction reports: -50%
- Dev time per feature: -30%
- Boilerplate LOC: -40%
- First-attempt compile: 70% → 85%

---

### 12. Cross-Platform Considerations

**File atomicity:**
```go
// POSIX (Linux, macOS)
func atomicRename(src, dst string) error {
    return os.Rename(src, dst)  // Atomic on POSIX
}

// Windows fallback
func atomicRenameWindows(src, dst string) error {
    // Use MoveFileEx with MOVEFILE_REPLACE_EXISTING
    // Fallback: copy + delete with temp marker
    if err := copyFile(src, dst); err != nil {
        return err
    }
    return os.Remove(src)
}
```

**File locks:**
- **Don't use flock** (platform-specific, unreliable on NFS)
- **Do use SQLite locks table** (works everywhere)

**Path separators:**
```go
import "path/filepath"

// ✅ GOOD (cross-platform)
path := filepath.Join(".claude", "state", "messages", agentID)

// ❌ BAD (POSIX-only)
path := ".ailang/state/messages/" + agentID
```

---

### 12. Minimal API Facade

**Go interface (ergonomics):**
```go
package protocol

type Protocol interface {
    // Send a message with retry
    Send(ctx context.Context, to string, msg *Envelope) (*Response, error)

    // Receive messages (returns channel, acknowledges via Ack)
    Receive(ctx context.Context, self string) (<-chan Envelope, error)

    // Acknowledge message (write response, archive)
    Ack(ctx context.Context, env *Envelope, resp *Response) error

    // Resolve agent (discovery)
    ResolveAgent(name string) (*AgentInfo, error)

    // Check backpressure
    CheckBackpressure(agentID string) error
}

type AgentInfo struct {
    AgentID    string
    InboxPath  string
    Caps       []string
    Status     string
}

// Mock for unit tests
type MockProtocol struct {
    messages map[string]*Response
}

func (m *MockProtocol) Send(ctx context.Context, to string, msg *Envelope) (*Response, error) {
    return m.messages[msg.MessageID], nil
}
```

**Example usage:**
```go
func (a *Agent) RunTask(task Task) error {
    p := protocol.New()

    // Send request
    msg := &Envelope{
        MessageID: generateUUID(),
        ToAgent:   "eval-orchestrator",
        Payload:   task,
    }
    resp, err := p.Send(context.TODO(), "eval-orchestrator", msg)
    if err != nil {
        return err
    }

    // Process response
    return a.handleResponse(resp)
}
```

---

## Implementation Plan (Revised)

**Total estimate:** 3-4 days (was 2-3)

### Phase 1: Core Infrastructure (2 days)

**Milestone 1: Message Passing + Idempotency (1 day)**
- Create `.ailang/state/messages/` directory structure
- Implement atomic write (temp → fsync → rename)
- Implement message reader with deduplication
- Create `messages` table in SQLite
- Test: Send message, dedupe duplicate sends
- Files:
  - `internal/agent_protocol/message.go` (300 LOC, was 200)
  - `internal/agent_protocol/message_test.go` (200 LOC, was 150)

**Milestone 2: SQLite State + Leases (1 day)**
- Create `.ailang/state/agents.db` with full schema
- Implement state save/load with integrity checks
- Implement lease acquisition/release
- Implement reaper for expired leases
- Test: Acquire lease, simulate crash, reaper recovers
- Files:
  - `internal/agent_protocol/state.go` (350 LOC, was 250)
  - `internal/agent_protocol/state_test.go` (250 LOC, was 180)
  - `internal/agent_protocol/reaper.go` (150 LOC, new)

### Phase 2: Hardening + Observability (1 day)

**Milestone 3: DLQ + Metrics (0.5 day)**
- Implement dead-letter queue logic
- Implement metrics tracking (agent_metrics table)
- Create `ailang agent top` CLI command
- Test: Exhaust retries, verify DLQ, inspect metrics
- Files:
  - `internal/agent_protocol/dlq.go` (150 LOC)
  - `internal/agent_protocol/metrics.go` (200 LOC)
  - `cmd/ailang/agent_top.go` (250 LOC)

**Milestone 4: Security + Discovery (0.5 day)**
- Implement HMAC signing with key rotation
- Implement agent discovery (ResolveAgent)
- Implement capability negotiation
- Implement path validation + content-addressed artifacts
- Test: Sign/verify, discover agent, negotiate caps
- Files:
  - `internal/agent_protocol/security.go` (300 LOC)
  - `internal/agent_protocol/discovery.go` (150 LOC)

### Phase 3: Verification + Integration (1 day)

**Milestone 5: Verification Contracts (0.5 day)**
- Implement SHA256 artifact hashing
- Implement content-addressed storage
- Implement toolchain digest recording
- Create `verification_results` table writer
- Test: Verify code artifact, check reproducibility
- Files:
  - `internal/agent_protocol/verification.go` (300 LOC, was 200)
  - `internal/agent_protocol/verification_test.go` (250 LOC, was 180)

**Milestone 6: API Facade + Integration Test (0.5 day)**
- Create `Protocol` interface
- Implement MockProtocol for testing
- End-to-end test: Two agents exchange messages with verification
- Test chaos scenarios (kill receiver, corrupt message, disk full)
- Files:
  - `internal/agent_protocol/protocol.go` (200 LOC)
  - `internal/agent_protocol/integration_test.go` (400 LOC, was 250)

### Phase 4: Documentation + Meta-Agent (0.5 day)

**Milestone 7: Documentation**
- Create protocol specification doc
- Add examples for common workflows
- Document message types + envelope fields
- Create troubleshooting guide
- Add chaos testing playbook
- Files:
  - `docs/guides/agent_protocol.md` (700 LOC, was 500)
  - `docs/guides/agent_protocol_examples.md` (400 LOC, was 300)
  - `docs/guides/agent_protocol_troubleshooting.md` (200 LOC, new)

**Milestone 8: DX Feedback Loop (0.5 day) — NEW!**
- Implement friction reporting tables (dx_friction_reports, dx_improvements, dx_metrics)
- Implement `ailang dx` CLI commands (report, analyze, generate-design, metrics)
- Integrate with meta-agent Stage 1 (analyze friction)
- Test: Report friction, cluster reports, generate design doc
- Files:
  - `internal/agent_protocol/dx_feedback.go` (300 LOC)
  - `cmd/ailang/dx_commands.go` (400 LOC)
  - `.claude/agents/ailang-dev-cycle.md` (updated with DX analysis)

**Milestone 9: Meta-Agent Integration (deferred to v0.4.1)**
- Update `ailang-dev-cycle.md` to use protocol
- Add message sending to each stage
- Add verification contracts
- Test full cycle with protocol
- Files:
  - `.claude/agents/ailang-dev-cycle.md` (updated)

---

## Testing Strategy (Enhanced)

### Unit Tests
- ✅ Message serialization/deserialization
- ✅ State save/load with integrity checks
- ✅ Idempotency (duplicate message_id rejected)
- ✅ Lease acquisition/release
- ✅ Reaper recovery from expired leases
- ✅ DLQ logic (retries exhausted)
- ✅ HMAC signing/verification with key rotation
- ✅ Path validation (prevent ../)
- ✅ Capability negotiation

### Integration Tests
- ✅ Two agents exchange messages (full round-trip)
- ✅ Three agents coordinate (A → B → C)
- ✅ Concurrent message handling (5 agents sending to 1)
- ✅ Verification contract (producer → verifier)
- ✅ Long-running task with progress updates
- ✅ Backpressure (receiver overloaded)
- ✅ DX friction reporting (agent reports friction → design doc generated)

### Chaos Tests (New!)
- ✅ Kill receiver mid-processing → reaper recovers
- ✅ Flip bit in message → checksum fails, move to DLQ
- ✅ Fill disk → sender retries with backoff
- ✅ Clock skew → deadline expires, move to DLQ
- ✅ Corrupt SQLite DB → detection triggers alert

### Migration Tests
- ✅ Bump protocol_version → old agents tolerate new fields
- ✅ Bump schema_version → state migration works
- ✅ Add new capability → older agents ignore gracefully

**Target:** 95% test coverage on protocol code

---

## Migration Plan

### Phase 1: Opt-In (v0.4.0-alpha)
- Protocol implemented but not required
- Meta-agent can use protocol OR pause for user
- User chooses: `--protocol` flag to enable
- Default: off (human-in-the-loop)

### Phase 2: Default (v0.4.0-beta)
- Protocol enabled by default
- User can disable: `--no-protocol` flag
- Pause points only for critical decisions (design approval)
- Metrics tracked, DLQ monitored

### Phase 3: Required (v0.4.0)
- Protocol is only option
- No `--no-protocol` flag
- Human approval via GitHub PR reviews, not interactive
- Full observability via `ailang agent top`

---

## Alternatives Considered

### Alternative 1: HTTP API
**Pros:** Standard protocol, widely supported
**Cons:** Requires server, complex deployment, overkill for local agents
**Rejected:** Too heavy for local-only coordination

### Alternative 2: Redis Pub/Sub
**Pros:** Fast, proven, handles concurrency
**Cons:** Requires Redis server, adds dependency
**Rejected:** Overkill, adds operational complexity

### Alternative 3: gRPC
**Pros:** Strongly typed, efficient
**Cons:** Requires proto definitions, complex setup
**Rejected:** Too complex for simple coordination

### Alternative 4: Unix Sockets
**Pros:** Fast IPC, no network overhead
**Cons:** Not portable to Windows, harder to debug
**Rejected:** Files are more observable and portable

### Alternative 5: Exactly-Once Delivery
**Pros:** No duplicate processing
**Cons:** Requires distributed transactions, complex
**Rejected:** At-least-once + idempotency is simpler and sufficient

**Chosen:** File-based + SQLite (simplest, most observable, crash-safe)

---

## Success Criteria

**Must have:**
- ✅ Meta-agent completes full cycle without human intervention
- ✅ Agents can verify each other's work programmatically
- ✅ Messages are observable (JSON files, easy to debug)
- ✅ State persists across agent restarts
- ✅ Crash-safe (reaper recovers from failures)
- ✅ Idempotent (retries don't cause double-execution)
- ✅ 95% test coverage on protocol code
- ✅ Observability (ailang agent top shows real-time status)

**Nice to have:**
- ⚠️ Protocol visualizer (web UI showing message flow)
- ⚠️ Performance benchmarks (message latency, throughput)
- ⚠️ Multi-machine support (agents on different computers via NFS/SSH)

**Future (v0.5+):**
- 🔮 AILANG-native protocol (agents implemented in AILANG)
- 🔮 Streaming responses (chunked payloads for large artifacts)
- 🔮 Socket transport option (same envelope, faster than files)
- 🔮 Cryptographic proofs (beyond SHA256, full ZK proofs)
- 🔮 Multi-agent swarms (10+ agents collaborating)

---

## Performance Considerations

**Expected load:**
- Agents: 5-10 active agents
- Messages: <100 messages per day
- State size: <10MB per agent
- SQLite file: <100MB total

**Benchmarks (targets):**
- Message send: <10ms (atomic write + fsync)
- Message receive: <10ms (scan + lease)
- State save: <50ms (SQLite insert + fsync)
- State load: <50ms (SQLite query + checksum)
- Verification: <5 seconds (depends on test suite)
- Reaper cycle: <100ms (scan expired leases)

**Optimization:**
- SQLite with WAL mode (concurrent reads)
- Message batching (send multiple in one file)
- State delta updates (only changed fields)
- Lease TTL tuning (60s is reasonable default)

---

## Security Considerations

### Threat Model

**Threat 1: Malicious Agent**
- **Attack:** Rogue agent sends fake verification results
- **Mitigation:** SHA256 hashes + Git commit verification + HMAC signatures
- **Detection:** Cross-verify with test-coverage-guardian

**Threat 2: State Tampering**
- **Attack:** Attacker modifies `.ailang/state/agents.db`
- **Mitigation:** Checksum validation on every state load
- **Detection:** State corruption triggers alert, move to error state

**Threat 3: Message Spoofing**
- **Attack:** Fake agent sends messages as another agent
- **Mitigation:** HMAC signatures with key rotation
- **Detection:** Signature verification on message receive

**Threat 4: Denial of Service**
- **Attack:** Agent sends 1000s of messages, fills disk
- **Mitigation:** Rate limiting (10 msg/sec per agent), disk quotas (100MB max)
- **Detection:** Monitor queue size, alert if >100 pending

**Threat 5: Directory Traversal**
- **Attack:** Malicious payload includes `../../../etc/passwd`
- **Mitigation:** Path validation (reject paths with `..`)
- **Detection:** Invalid path triggers error, move to DLQ

### Security Measures

1. **Message signing** (HMAC-SHA256 with key rotation)
2. **State checksums** (SHA256)
3. **Rate limiting** (10 msg/sec per agent)
4. **Disk quotas** (max 100MB for `.ailang/state/`)
5. **Audit logging** (all messages logged to `agent_history`)
6. **Agent allowlist** (only known agent_ids accepted)
7. **Path validation** (no `../`, must be under `.ailang/state/`)
8. **Content-addressed storage** (artifacts referenced by hash, not path)

---

## Observability CLI Commands

```bash
# Real-time agent status
ailang agent top

# Trace message flow for a cycle
ailang agent trace <correlation_id>

# Inspect agent state
ailang agent status <agent_id>

# Show agent metrics
ailang agent metrics <agent_id> [--window=1h]

# List dead-letter queue
ailang agent dlq list

# Retry message from DLQ
ailang agent dlq retry <message_id>

# Show agent history
ailang agent history <agent_id> [--limit=50]

# Show verification results
ailang agent verify list [--status=verified|rejected]
```

---

## Axiom Compliance

**Canonical reference:** [Design Axioms](/docs/references/axioms)

### Scoring

| Axiom | Score | Justification |
|-------|-------|---------------|
| A1: Determinism | +1 | Idempotent message delivery via message_id deduplication |
| A2: Replayability | +1 | Full audit log in agent_history enables replay |
| A3: Effect Legibility | +1 | declared_effects field makes agent effects explicit |
| A4: Explicit Authority | +1 | Agent allowlist enforces capability bounds |
| A5: Bounded Verification | +1 | SHA256 hashes enable local artifact verification |
| A6: Safe Concurrency | +1 | Lease-based coordination prevents race conditions |
| A7: Machines First | +1 | File-based messages are machine-readable JSON |
| A8: Minimal Syntax | 0 | No AILANG syntax changes |
| A9: Cost Visibility | +1 | agent_metrics tracks costs per operation |
| A10: Composability | +1 | Protocol composes with existing agent/skill architecture |
| A11: Structured Failure | +1 | DLQ with structured error reasons |
| A12: System Boundary | +1 | Clear agent boundaries via inbox directories |

**Net Score: +11** ✅ Proceed to implementation

### Hard Violation Check

- [x] A1 (Determinism): Message ordering guaranteed by timestamps
- [x] A3 (Effects): declared_effects field makes all effects explicit
- [x] A4 (Authority): Agent allowlist prevents unauthorized agents
- [x] A7 (Machines First): JSON files designed for machine consumption

---

## Roadmap Fit

| Version | Features | Status |
|---------|----------|--------|
| **v0.4.0-alpha** | Core protocol (files + SQLite + leases) | Planned |
| **v0.4.0-beta** | DLQ, metrics, observability CLI | Planned |
| **v0.4.0** | Security hardening, chaos testing | Planned |
| **v0.4.1** | Meta-agent integration, streaming responses | Future |
| **v0.5.x** | Socket transport option, multi-host support | Future |

---

## References

- [VISION.md](../../docs/VISION.md) — Multi-agent cooperation (lines 152-202)
- [ailang-dev-cycle.md](../../.claude/agents/ailang-dev-cycle.md) — Meta-agent specification
- [Skills README](../../.claude/skills/README.md) — Current agent/skill architecture
- [CLAUDE.md](../../CLAUDE.md) — Project instructions

---

## Appendix: Example Message Flow (Full Cycle)

**Scenario:** Meta-agent runs full development cycle (M-POLY-B) with protocol

```
1. Meta-agent → design-doc-creator
   Message ID: msg_001
   Correlation ID: cycle_20251023_001
   Action: create_design_doc { feature: "M-POLY-B" }
   Trace: [msg_001]

2. design-doc-creator → Meta-agent
   Message ID: msg_001_response
   Parent: msg_001
   Status: success
   Artifact: design_docs/planned/M-POLY-B.md (sha256:abc123...)
   Trace: [msg_001 → msg_001_response]

3. Meta-agent → design-spec-auditor
   Message ID: msg_002
   Parent: msg_001_response
   Action: verify_design_doc { artifact_hash: "sha256:abc123..." }
   Trace: [msg_001 → msg_001_response → msg_002]

4. design-spec-auditor → Meta-agent
   Message ID: msg_002_response
   Parent: msg_002
   Status: verified
   Verification ID: 1
   Trace: [msg_001 → msg_001_response → msg_002 → msg_002_response]

5. Meta-agent → sprint-planner
   Message ID: msg_003
   Parent: msg_002_response
   Action: plan_sprint { design: "M-POLY-B.md" }
   Trace: [... → msg_003]

6. sprint-planner → Meta-agent
   Message ID: msg_003_response
   Status: success
   Artifact: sprint_plan.md (sha256:def456...)
   Duration: 2 days

7. Meta-agent → sprint-executor
   Message ID: msg_004
   Action: execute_sprint { plan: "sprint_plan.md" }

8. sprint-executor → test-coverage-guardian (async)
   Message ID: msg_005
   Parent: msg_004
   Action: verify_code { artifact_hash: "sha256:789...", toolchain: "..." }

9. test-coverage-guardian → sprint-executor
   Message ID: msg_005_response
   Status: verified
   Tests: 12 passed, 0 failed
   Coverage: 94.2%
   Verification ID: 2

10. sprint-executor → Meta-agent
    Message ID: msg_004_response
    Status: success
    Milestones: 3 complete
    Duration: 1.8 days

11. Meta-agent → release-manager
    Message ID: msg_006
    Action: release_version { version: "v0.3.15" }

12. release-manager → Meta-agent
    Message ID: msg_006_response
    Status: success
    Tag: v0.3.15
    CI: passing

13. Meta-agent → post-release
    Message ID: msg_007
    Action: run_eval_baseline { version: "v0.3.15" }

14. post-release → eval-orchestrator
    Message ID: msg_008
    Parent: msg_007
    Action: run_eval_baseline { version: "v0.3.15" }

15. eval-orchestrator → post-release
    Message ID: msg_008_response
    Status: success
    Baseline: eval_results/baselines/v0.3.15/

16. post-release → Meta-agent
    Message ID: msg_007_response
    Status: success
    Dashboard: updated
    Success rate: 68.4%

✅ Cycle complete — 16 messages, 0 retries, 0 failures, 137.5s total
```

**Observability:**
```bash
$ ailang agent trace cycle_20251023_001

TRACE: cycle_20251023_001 (137.5s total, 16 messages, 0 retries, 0 failures)
├─ msg_001: ailang-dev-cycle → design-doc-creator (2s)
│  └─ msg_002: design-doc-creator → design-spec-auditor (1s)
├─ msg_003: ailang-dev-cycle → sprint-planner (3s)
├─ msg_004: ailang-dev-cycle → sprint-executor (120s)
│  └─ msg_005: sprint-executor → test-coverage-guardian (5s)
├─ msg_006: ailang-dev-cycle → release-manager (4s)
└─ msg_007: ailang-dev-cycle → post-release (5s)
   └─ msg_008: post-release → eval-orchestrator (3s)
```

---

## Changelog

**v3 (2025-10-23):** DX Feedback Loop (KEY KPI!)
- **Added DX friction reporting system** (dx_friction_reports, dx_improvements, dx_metrics tables)
- **Added friction-to-design-doc pipeline** (auto-generate design docs from clustered friction)
- **Added `ailang dx` CLI commands** (report, analyze, generate-design, metrics, trends)
- **Integrated with meta-agent Stage 1** (friction analysis drives roadmap prioritization)
- **Target KPIs**: -50% friction reports, -30% dev time, -40% boilerplate, +15pp compile rate
- Revised timeline: 3.5-4.5 days (added DX feedback milestone)

**v2 (2025-10-23):** Production-ready revision
- Added at-least-once delivery + idempotency
- Added crash-safe handoff (lease-based)
- Added DLQ for failed messages
- Added observability (metrics, CLI, tracing)
- Added security hardening (HMAC, path validation, content-addressed storage)
- Added backpressure + rate limiting
- Enhanced envelope (ttl, deadline, trace_id, parent_message_id, kid, toolchain_digest)
- Added agent discovery + capability negotiation
- Added chaos testing strategy
- Revised timeline: 3-4 days (was 2-3)

**v1 (2025-10-23):** Initial design
- File-based message passing
- SQLite state management
- Basic verification contracts

---

**Status:** 📋 This design doc was created by the meta-agent as its bootstrap task, then revised with production-ready guardrails. Next step: Move to sprint planning (Stage 3 of development cycle).

**Note:** This protocol enables the multi-agent future described in VISION.md. Once implemented, agents can truly "reason, refactor, and verify" each other's work without human intermediaries — with crash safety, observability, and security built in from day one.

🚀 **"When the coder is the model."**
