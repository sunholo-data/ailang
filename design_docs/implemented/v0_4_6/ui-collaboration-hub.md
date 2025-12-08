# UI Collaboration Hub - Human-AI Orchestration Interface

**Status**: Planned
**Target**: v0.4.4
**Priority**: P1 (Medium - High value but not blocking core language features)
**Estimated**: 3-4 weeks (120-160 hours)
**Dependencies**:
- SQLite message persistence (existing)
- AILANG runtime with effect system (v0.3.14+)
- Agent inbox infrastructure (v0.3.14+)

## AI-First Alignment Check

**Score this feature against AILANG's core principles:**

| Principle | Impact | Score | Notes |
|-----------|--------|-------|-------|
| Reduce Syntactic Noise | Neutral | 0 | UI/tooling feature, doesn't affect language syntax |
| Preserve Semantic Clarity | Helps | +1 | Makes instance collaboration explicit and traceable |
| Increase Determinism | Helps | +1 | SQLite message bus creates complete audit trail of all interactions |
| Lower Token Cost | Helps | +1 | Reduces wasted work through better human-AI coordination |
| **Net Score** | | **+3** | **Decision: Move forward** |

**Decision rule:** Net score > +1 → Move forward | ≤ 0 → Reject or redesign

**Reference:** See [AI-first DX philosophy](../v0_3_15/example-parity-vision-alignment.md#-design-principle-ai-first-dx)

---

## ⚠️ DESIGN REVISIONS (2025-11-08)

**This design was significantly improved based on architectural review. Key changes:**

### 1. **Extend, Don't Fork**
- ❌ ~~Create parallel message bus with new schema~~
- ✅ **Migrate existing file-based inbox to SQLite** (.ailang/state/messages/ → collaboration.db)
- ✅ **Single source of truth** for all CLIs, agents, and UI

### 2. **Ordering Guarantees**
- ❌ ~~Rely on timestamps alone (race conditions)~~
- ✅ **message_seq per thread** (monotonic, prevents gaps)
- ✅ **Cursor-based resume** (from_seq for at-least-once delivery)
- ✅ **Idempotent processing** (client deduplicates by seq)

### 3. **Effect-Gated Approvals**
- ❌ ~~Approvals as UI-only feature~~
- ✅ **Capability tokens** (HMAC-signed, tied to effect system)
- ✅ **Runtime verification** before allowing effects
- ✅ **Policy engine** (effect_delta: what agent wants to do)

### 4. **Deterministic Replay**
- ✅ **replay_snapshots table** (model, seed, temperature, prompt_slate, checksums)
- ✅ **Full context capture** for reproduction

### 5. **Security by Default**
- ✅ **Session auth + PATs** (not an afterthought)
- ✅ **CSRF, rate limiting, input validation**
- ✅ **Instance isolation** (can't read other instances' messages)

### 6. **Tight MVP Scope**
- ❌ ~~160 hours for full feature set~~
- ✅ **125 hours for MVP** (Message Center + Approvals + WebSocket + Security)
- ✅ **Defer to v0.4.5**: Kanban, Direction Panel, Timeline, Templates
- ✅ **3 weeks with 2 developers** (realistic)

### 7. **Scalability & Testing**
- ✅ **WAL mode, indices, attachments table** (handle 50k+ messages)
- ✅ **Test bots** (echo agent, flaky agent, human bot)
- ✅ **Load tests** (50 threads × 1k messages)

**Why This Matters:**
- **No parallel systems** → simpler, faster, less bugs
- **message_seq** → reliable ordering, no race conditions
- **Capability tokens** → approvals are *protocol*, not UI sugar
- **Tight MVP** → ship value fast, iterate based on usage

---

## Problem Statement

**Current State:**
- AILANG instances run in isolation with limited human oversight
- No structured way for humans to guide multiple AILANG instances
- Instance coordination happens through ad-hoc mechanisms
- No approval workflow for critical changes (schema modifications, breaking changes)
- Limited visibility into what instances are doing and why
- Existing agent inbox (`.ailang/state/messages/`) is CLI-only, no UI

**Impact:**
- **Developers**: Cannot efficiently orchestrate multiple AILANG instances
- **Team leads**: No visibility into AI agent activities for approval/oversight
- **AI researchers**: Missing data on human-AI collaboration patterns
- **Lost productivity**: Instances may pursue wrong approaches without early human guidance

## Goals

**Primary Goal:** Transform the UI from passive monitoring to an **active orchestration hub** where humans coordinate AILANG instances through structured messaging, approvals, and creative direction.

**Success Metrics:**
- Reduce time from directive to completion by 40% (through early guidance)
- Enable 5+ concurrent AILANG instances per developer with clear coordination
- Track 100% of human-AI interactions for deterministic replay
- Achieve <30 second response time for approval workflows
- Reduce wasted compute by 30% (fewer wrong paths taken)

## Solution Design

### Overview

The UI becomes the **human interface to the message bus**. Instead of just showing logs and metrics, it enables:

1. **Bidirectional Communication**: Humans send directives, instances ask questions, both coordinate via threaded messages
2. **Approval Workflows**: Instances request approval for critical changes before proceeding
3. **Creative Direction**: High-level goals and constraints visible to all instances
4. **Multi-Agent Orchestration**: Visual management of multiple instances working toward common goals
5. **Deterministic Audit Trail**: Every interaction stored in SQLite for replay and analysis

### Architecture

**Message Bus Integration:**

```
Human (via UI) ─┐
               ├─→ SQLite Message Queue ←→ AILANG Runtime
AILANG agents ─┘
```

**Core Components:**

1. **SQLite Message Schema** - Central persistence layer
2. **WebSocket Server** - Real-time bidirectional communication
3. **UI Components** - React-based orchestration interface
4. **AILANG Integration** - Runtime message polling and publishing
5. **Approval Engine** - Workflow logic for human approvals

### Database Schema (REVISED)

**Design Principle: Extend the existing message bus, don't fork it.**

The current agent inbox uses file-based messages in `.ailang/state/messages/`. This design **migrates** that to a single SQLite database that all CLIs, headless agents, and the UI share.

**Core schema with ordering guarantees:**

```sql
-- Main message table (extends existing agent inbox)
CREATE TABLE messages (
    id TEXT PRIMARY KEY,
    thread_id TEXT NOT NULL,
    message_seq INTEGER NOT NULL,      -- Monotonic per thread; guarantees order
    created_at INTEGER NOT NULL,       -- Unix timestamp (ms)

    -- Routing
    from_type TEXT NOT NULL,           -- 'human', 'ailang_instance'
    from_id TEXT NOT NULL,             -- user_id or instance_id
    to_type TEXT,                      -- 'human', 'ailang_instance', 'broadcast'
    to_id TEXT,                        -- target recipient (null for broadcast)

    -- Content
    kind TEXT NOT NULL,                -- 'directive', 'question', 'proposal', 'status', 'result'
    subject TEXT,
    content TEXT,                      -- Message body (keep small; use attachments for large data)
    metadata_json TEXT,                -- JSON: {model, temperature, seed, tool_outputs, etc.}

    -- State (separate transport from business)
    delivery_state TEXT NOT NULL DEFAULT 'pending',  -- 'pending', 'visible', 'acked'
    business_state TEXT DEFAULT 'open',              -- 'open', 'resolved', 'archived'

    -- Threading
    reply_to TEXT REFERENCES messages(id),

    -- Soft delete for auditability
    deleted_at INTEGER,

    UNIQUE (thread_id, message_seq)
);

-- Threads (replaces scattered inbox directories)
CREATE TABLE threads (
    id TEXT PRIMARY KEY,
    title TEXT NOT NULL,
    created_at INTEGER NOT NULL,
    created_by_type TEXT NOT NULL,
    created_by_id TEXT NOT NULL,
    status TEXT NOT NULL DEFAULT 'active',  -- 'active', 'paused', 'resolved', 'archived'
    context_json TEXT,                      -- JSON: {goals, constraints, directives}
    last_seq INTEGER NOT NULL DEFAULT 0,    -- Last allocated message_seq (for ordering)
    updated_at INTEGER NOT NULL
);

-- Subscriptions (which instances/humans watch which threads)
CREATE TABLE subscriptions (
    instance_id TEXT NOT NULL,
    thread_id TEXT NOT NULL,
    from_seq INTEGER NOT NULL DEFAULT 0,   -- Resume cursor for at-least-once delivery
    subscribed_at INTEGER NOT NULL,
    last_ack_seq INTEGER DEFAULT 0,        -- Last seq client acknowledged
    PRIMARY KEY (instance_id, thread_id)
);

-- Approvals (policy engine for effect-gated actions)
CREATE TABLE approvals (
    id TEXT PRIMARY KEY,
    thread_id TEXT NOT NULL,
    instance_id TEXT NOT NULL,
    created_at INTEGER NOT NULL,

    -- What the agent wants to do
    effect_delta_json TEXT NOT NULL,       -- JSON: {cap_type: 'FS', paths: [...], budget_delta: 0.50}
    proposal TEXT NOT NULL,                -- Human-readable description
    impact TEXT NOT NULL,                  -- 'low', 'medium', 'high'
    estimated_cost REAL,

    -- Approval state
    status TEXT NOT NULL DEFAULT 'pending', -- 'pending', 'approved', 'rejected', 'modified'
    reviewed_by TEXT,                       -- user_id
    reviewed_at INTEGER,
    review_notes TEXT,

    -- Capability token (signed JWT/JWS over thread+caps+expiry)
    -- Runtime verifies this before allowing action
    capability_token TEXT,
    token_expires_at INTEGER
);

-- Attachments (don't stuff large payloads into messages.content)
CREATE TABLE attachments (
    id TEXT PRIMARY KEY,
    message_id TEXT NOT NULL REFERENCES messages(id),
    kind TEXT NOT NULL,                    -- 'code', 'diff', 'test_output', 'artifact'
    content_type TEXT,                     -- 'text/plain', 'application/json', etc.
    path TEXT,                             -- File path if stored externally
    blob BLOB,                             -- Small attachments (<10KB) can be inline
    size_bytes INTEGER NOT NULL,
    created_at INTEGER NOT NULL
);

-- Replay metadata (enable deterministic replay)
CREATE TABLE replay_snapshots (
    id TEXT PRIMARY KEY,
    thread_id TEXT NOT NULL,
    message_id TEXT NOT NULL,              -- Snapshot taken at this message
    created_at INTEGER NOT NULL,

    -- Full context to reproduce
    model_id TEXT NOT NULL,
    model_version TEXT,
    temperature REAL,
    seed INTEGER,
    top_p REAL,
    tool_list_json TEXT,                   -- JSON array of available tools
    prompt_slate_json TEXT,                -- Full conversation up to this point
    prompt_checksum TEXT                   -- SHA256 of prompt_slate for quick comparison
);

-- Indices for performance (WAL mode required)
CREATE INDEX idx_messages_thread_seq ON messages(thread_id, message_seq);
CREATE INDEX idx_messages_to ON messages(to_type, to_id, delivery_state);
CREATE INDEX idx_messages_created ON messages(created_at, id);
CREATE INDEX idx_threads_status ON threads(status, updated_at);
CREATE INDEX idx_subscriptions_thread ON subscriptions(thread_id);
CREATE INDEX idx_approvals_status ON approvals(status, created_at);
CREATE INDEX idx_attachments_message ON attachments(message_id);
CREATE INDEX idx_replay_thread ON replay_snapshots(thread_id, created_at);

-- Enable WAL mode for write concurrency
PRAGMA journal_mode=WAL;
PRAGMA synchronous=NORMAL;
PRAGMA busy_timeout=5000;
```

**Migration from file-based agent inbox:**

```bash
# Existing structure:
.ailang/state/messages/
├── claude-code/_unread/msg_*.json
├── claude-code/_processed/msg_*.json
└── inbox/user/_unread/msg_*.json

# New structure:
.ailang/state/collaboration.db     # SQLite database with above schema
.ailang/state/messages/            # Deprecated, kept for rollback
```

**Backward compatibility:**
- `ailang agent inbox` reads from SQLite, falls back to file-based if DB doesn't exist
- `ailang agent ack` writes to SQLite, syncs to file-based during migration period
- UI is SQLite-only (simpler, faster)

**Key improvements over initial design:**
1. **Single source of truth**: No parallel buses or schema duplication
2. **Ordering guarantees**: `message_seq` prevents race conditions; `(thread_id, message_seq)` is resume cursor
3. **Deterministic replay**: `replay_snapshots` table stores full context for reproduction
4. **Effect-gated approvals**: `capability_token` ties to runtime effect system
5. **Scalability**: Indices, WAL mode, attachments table for large payloads
6. **Soft delete**: `deleted_at` preserves audit trail

### UI Components

#### 1. Message Center (Primary View)

**Layout**: Three-column design

```
┌─────────────┬──────────────────────┬──────────────┐
│  Threads    │   Conversation       │   Context    │
│  (left)     │   (center)           │   (right)    │
│             │                      │              │
│ ○ Feature X │ Human: "Add auth"   │ Instance: 3  │
│ ● Bug #123  │ AILANG: "Propose    │ Status: ⟳    │
│ ○ Refactor  │  OAuth2 or JWT?"    │ Tokens: 2.3k │
│             │ Human: [Approve]    │ Cost: $0.08  │
│ + New       │ AILANG: "Working... │              │
│             │  [progress bar]"     │ [View logs]  │
└─────────────┴──────────────────────┴──────────────┘
```

**Features**:
- Thread list with unread indicators (● = unread, ○ = read)
- Real-time message updates via WebSocket
- Rich message types: text, code blocks, diffs, diagrams
- Quick actions: approve, reject, provide input
- @mention AILANG instances to assign work
- Search/filter threads by status, instance, date

#### 2. Orchestration Board

**Purpose**: Visual Kanban-style management of AILANG instances

```
┌──────────────┬──────────────┬──────────────┬──────────────┐
│ Queued       │ In Progress  │ Review       │ Complete     │
├──────────────┼──────────────┼──────────────┼──────────────┤
│ Add tests    │ Refactor API │ Fix auth bug │ Update docs  │
│ @ailang-2    │ @ailang-1    │ @ailang-3    │ @ailang-1    │
│              │ 45% done     │ Needs review │ ✓            │
│              │              │              │              │
│ New endpoint │              │              │              │
│ Unassigned   │              │              │              │
└──────────────┴──────────────┴──────────────┴──────────────┘
```

**Interactions**:
- Drag cards between columns
- Click card to open message thread
- Assign work to specific instances
- Set priorities and dependencies
- Spawn new AILANG instance for a task

#### 3. Creative Direction Panel

**Purpose**: Provide high-level guidance visible to all instances

```
┌────────────────────────────────────────────────────┐
│ Current Focus:                                     │
│ "Migrate authentication to OAuth2. Prioritize     │
│  security over speed. Must maintain backward      │
│  compatibility with existing API."                 │
│                                                    │
│ Constraints:                                       │
│ • Budget: $50 remaining today                     │
│ • Deadline: End of week                           │
│ • Style: Follow existing patterns in auth/        │
└────────────────────────────────────────────────────┘

Active Directives:
☑ Prefer functional patterns over OOP
☑ Always add tests for new code
☑ Ask before making breaking changes
+ Add directive
```

**Template Library**:
- "Review my PR"
- "Optimize for performance"
- "Add comprehensive error handling"
- Custom templates user can save

#### 4. Instance Manager (Enhanced)

**Purpose**: Spawn and configure AILANG instances with specific roles

```
┌─────────────────────────────────────────────────┐
│ Create New Instance                             │
├─────────────────────────────────────────────────┤
│ Name: ailang-feature-x                          │
│ Role: [Developer / Reviewer / Researcher]       │
│ Specialization: [Backend / Frontend / Testing]  │
│                                                 │
│ Initial Prompt:                                 │
│ "Implement user profile endpoints. Follow      │
│  REST best practices. Use existing auth."       │
│                                                 │
│ Constraints:                                    │
│ ☑ Max 10k tokens per task                      │
│ ☑ Require approval for schema changes          │
│ ☐ Auto-commit on success                       │
│                                                 │
│ Subscribe to threads: [Feature X] [Bug #123]   │
│                                                 │
│                          [Cancel]  [Launch]    │
└─────────────────────────────────────────────────┘
```

#### 5. Approval Queue

**Purpose**: Review and approve AILANG proposals requiring human decision

```
┌─────────────────────────────────────────────────┐
│ ⚠ Requires Your Approval (3 pending)            │
├─────────────────────────────────────────────────┤
│ @ailang-1 proposes:                             │
│ "Change database schema to add user_roles table"│
│                                                 │
│ Impact: HIGH                                     │
│ • Migration required                            │
│ • 2,000 existing users affected                 │
│ • Estimated cost: $0.45                         │
│                                                 │
│ [View Details] [Approve] [Reject] [Modify]     │
└─────────────────────────────────────────────────┘
```

**Notification badge** on nav when items pending

#### 6. Collaboration Timeline

**Purpose**: Historical view of human-AI interaction

```
Now
 │
 ├─ You approved schema change (2m ago)
 │
 ├─ ailang-1 completed migration (5m ago)
 │
 ├─ You rejected OAuth approach (15m ago)
 │
 ├─ ailang-1 proposed OAuth (18m ago)
 │
 ├─ You: "Add authentication" (30m ago)
 │
Earlier...
```

### API Design

#### Message Operations

```
POST   /api/messages                # Send message (human or AI)
GET    /api/messages?thread_id=X    # Get thread messages
PATCH  /api/messages/:id/status     # Mark read/acknowledged
DELETE /api/messages/:id            # Delete message

POST   /api/threads                 # Create new thread
GET    /api/threads                 # List threads
PATCH  /api/threads/:id             # Update thread status/context

POST   /api/instances/:id/message   # Direct message to instance
POST   /api/broadcast                # Broadcast to all instances
```

#### Directive Management

```
GET    /api/directives              # Active directives
POST   /api/directives              # Add directive
DELETE /api/directives/:id          # Remove directive
GET    /api/directives/templates    # Template library
```

#### Approval Workflow

```
GET    /api/approvals/pending       # Items awaiting approval
POST   /api/approvals/:id/approve   # Approve proposal
POST   /api/approvals/:id/reject    # Reject proposal
POST   /api/approvals/:id/modify    # Request modification
```

#### Instance Management

```
GET    /api/instances               # List all instances
POST   /api/instances               # Spawn new instance
GET    /api/instances/:id           # Instance details
PATCH  /api/instances/:id           # Update configuration
DELETE /api/instances/:id           # Terminate instance
```

### WebSocket Protocol (REVISED)

**At-least-once delivery with sequence-based resume:**

**Client → Server (Subscribe):**
```json
{
  "type": "subscribe",
  "thread_id": "thread-123",
  "from_seq": 42  // Resume from message 43 onwards (0 = start from beginning)
}
```

**Server → Client (Message batch):**
```json
{
  "type": "messages",
  "thread_id": "thread-123",
  "messages": [
    {
      "id": "msg-456",
      "message_seq": 43,
      "created_at": 1699564832000,
      "from_type": "ailang_instance",
      "from_id": "ailang-1",
      "kind": "status",
      "content": "Refactoring complete. Ready for review."
    },
    {
      "id": "msg-457",
      "message_seq": 44,
      "created_at": 1699564840000,
      "from_type": "ailang_instance",
      "from_id": "ailang-1",
      "kind": "result",
      "content": "All tests passing (127/127)."
    }
  ],
  "last_in_thread_seq": 44  // Highest seq in this thread; client uses for catch-up decisions
}
```

**Client → Server (Acknowledgement):**
```json
{
  "type": "ack",
  "thread_id": "thread-123",
  "up_to_seq": 44  // Client processed messages 0-44
}
```

**Server → Client (Approval required):**
```json
{
  "type": "approval_required",
  "approval": {
    "id": "approval-789",
    "thread_id": "thread-123",
    "instance_id": "ailang-2",
    "proposal": "Change database schema to add user_roles table",
    "effect_delta": {
      "cap_type": "FS",
      "operation": "write",
      "paths": ["migrations/003_add_user_roles.sql"],
      "budget_delta": 0.45
    },
    "impact": "high",
    "estimated_cost": 0.45
  }
}
```

**Client → Server (Send message):**
```json
{
  "type": "send_message",
  "thread_id": "thread-123",
  "message": {
    "from_type": "human",
    "from_id": "user-1",
    "to_type": "ailang_instance",
    "to_id": "ailang-2",
    "kind": "directive",
    "subject": "Add validation",
    "content": "Add validation to user endpoints"
  }
}
```

**Reconnection flow:**
1. Client disconnects (network issue, browser refresh)
2. Client reconnects, sends `subscribe` with last acked `from_seq`
3. Server replays missed messages (at-least-once delivery)
4. Client deduplicates by `message_seq` (idempotent processing)

**Performance notes:**
- Batching: Server sends up to 50 messages per batch
- Throttling: Max 100 messages/sec per WebSocket connection
- Backpressure: If client falls >1000 messages behind, server pauses instance and notifies human

### Implementation Plan (REVISED - MVP First)

**MVP Scope (v0.4.4): Core collaboration value only**
- Message Center (threads + conversation)
- Approval Queue (effect-gated approvals)
- WebSocket with ordering guarantees
- Runtime integration (message polling, approval requests)
- Basic security (auth tokens, CSRF)

**Deferred to v0.4.5+:**
- Orchestration Board (Kanban drag-and-drop)
- Creative Direction Panel (start with simple directives in thread context)
- Instance Manager polish (role/specialization dropdowns)
- Timeline component (can derive from messages)
- Template library (start with 3 hard-coded templates)

---

**Phase 1: Database & Schema Migration** (~25 hours)
- [ ] Create SQLite schema (messages, threads, subscriptions, approvals, attachments, replay_snapshots)
- [ ] Enable WAL mode, add indices
- [ ] Migration tool: file-based inbox → SQLite (read existing msg_*.json files)
- [ ] Update `ailang agent inbox` to read from SQLite (fallback to files)
- [ ] Update `ailang agent ack` to write to SQLite (sync to files during migration)
- [ ] Unit tests for schema operations (90% coverage target)
- [ ] Benchmark: 1000 message writes, verify <20ms p95 latency

**Phase 2: Backend Messaging Core** (~30 hours)
- [ ] Message CRUD operations with `message_seq` allocation
- [ ] Thread management (create, subscribe, update context)
- [ ] Approval workflow with `capability_token` generation (HMAC-based)
- [ ] WebSocket server with connection pooling
  - [ ] Subscribe with `from_seq`, send batches
  - [ ] Client ack tracking (update `subscriptions.last_ack_seq`)
  - [ ] Reconnection handling
- [ ] Runtime integration:
  - [ ] Polling loop for AILANG instances (check for new messages every 2s)
  - [ ] Effect handler hook: emit approval request when cap exceeded
  - [ ] Capability token verification before allowing effect
- [ ] Unit tests for all operations (90% coverage)
- [ ] Integration test: Human message → instance receives → instance responds

**Phase 3: Core UI (Message Center + Approvals)** (~30 hours)
- [ ] Set up React + TypeScript frontend
- [ ] WebSocket client with `from_seq` resume logic
- [ ] Message Center:
  - [ ] Thread list (show unread count, last message preview)
  - [ ] Conversation view (render messages with markdown, code blocks)
  - [ ] Simple context panel (instance ID, status, token count)
- [ ] Approval Queue:
  - [ ] Show pending approvals
  - [ ] Approve/Reject/Modify actions
  - [ ] Display `effect_delta` (what the agent wants to do)
- [ ] Real-time updates via WebSocket
- [ ] UI tests with React Testing Library (80% coverage)

**Phase 4: Security & Auth** (~15 hours)
- [ ] Session-based auth for UI (cookie with 24h expiry)
- [ ] Personal Access Tokens (PATs) for CLIs/agents
  - [ ] Scope tokens to instance_id + thread_id
  - [ ] 7-day expiry
- [ ] CSRF protection (double-submit token)
- [ ] CORS: whitelist UI origin only
- [ ] Rate limiting (token bucket): 100 req/min per user, 20 req/min per IP
- [ ] Security tests: CSRF, unauthorized access, token expiry

**Phase 5: Polish & Testing** (~15 hours)
- [ ] Search threads by title, status, date
- [ ] Notification badges (approval queue count)
- [ ] Error handling (reconnection retries, toast notifications)
- [ ] Performance optimization:
  - [ ] Virtualized scrolling for 500+ message threads (react-window)
  - [ ] Lazy loading for attachments
- [ ] End-to-end tests with Playwright:
  - [ ] Human sends directive → instance receives → responds
  - [ ] Instance requests approval → human approves → instance proceeds
  - [ ] Reconnection after network drop
- [ ] Accessibility: ARIA labels, keyboard navigation

**Phase 6: Documentation** (~10 hours)
- [ ] User guide: "How to use the Message Center"
- [ ] API docs: REST endpoints, WebSocket events
- [ ] Configuration guide: polling intervals, token expiry, rate limits
- [ ] Migration guide: file-based inbox → SQLite
- [ ] Example workflows:
  - [ ] Assign work to an instance
  - [ ] Approve a schema change
  - [ ] Coordinate two instances on a task

**Total MVP: ~125 hours (3 weeks with 2 developers)**

**Post-MVP additions (v0.4.5+):**
- Orchestration Board (Kanban) - 20 hours
- Creative Direction Panel - 15 hours
- Instance Manager polish - 10 hours
- Timeline component - 8 hours
- Template library - 7 hours
- **Total v0.4.5: ~60 hours** (1.5 weeks)

### Security & Authentication

**Threat model:**
- Untrusted agents requesting excessive capabilities
- Unauthorized access to message threads
- CSRF attacks on approval endpoints
- Token theft/replay attacks
- Rate-based DoS

**Mitigations:**

**1. Authentication**
```go
// Session-based for UI
type Session struct {
    ID        string
    UserID    string
    CreatedAt time.Time
    ExpiresAt time.Time  // 24 hours
}

// Personal Access Tokens for CLIs/agents
type PAT struct {
    Token     string  // HMAC-SHA256 over (instance_id, thread_id, expiry, secret)
    InstanceID string
    ThreadID  string
    Scopes    []string  // ["messages:read", "messages:write", "approvals:request"]
    ExpiresAt time.Time  // 7 days
}
```

**2. Authorization**
- **Instance isolation**: Instance can only read messages where `to_id = instance_id` or `to_type = broadcast`
- **Thread access**: Instance can only subscribe to threads where `subscriptions.instance_id` exists
- **Approval scoping**: Approval requests must reference a thread the instance is subscribed to

**3. Capability Tokens (Effect-gated approvals)**
```go
// When human approves an effect request:
type CapabilityToken struct {
    ThreadID   string
    InstanceID string
    CapType    string  // "FS", "Net", "Exec"
    Operation  string  // "read", "write", "execute"
    Paths      []string
    BudgetDelta float64
    ExpiresAt  time.Time  // 1 hour

    Signature  string  // HMAC-SHA256(all_fields, secret)
}

// Runtime verifies before allowing effect:
func (r *Runtime) AllowEffect(token CapabilityToken, req EffectRequest) error {
    if !token.Verify(r.secret) {
        return ErrInvalidToken
    }
    if time.Now().After(token.ExpiresAt) {
        return ErrTokenExpired
    }
    if req.CapType != token.CapType || req.Operation != token.Operation {
        return ErrMismatch
    }
    for _, path := range req.Paths {
        if !contains(token.Paths, path) {
            return ErrUnauthorizedPath
        }
    }
    return nil  // Allowed
}
```

**4. CSRF Protection**
- Double-submit cookie: `CSRF-Token` header must match `csrf-token` cookie
- SameSite=Strict on session cookies
- Validate origin header on WebSocket upgrade

**5. Rate Limiting**
```go
type RateLimiter struct {
    buckets map[string]*TokenBucket
}

// Per-user limits
const (
    UserRequestsPerMin = 100
    IPRequestsPerMin   = 20
    WebSocketMsgsPerSec = 100
)
```

**6. Input Validation**
- Max message size: 10KB (use attachments for larger content)
- Max threads per user: 100 active
- Max subscriptions per instance: 50
- Sanitize all user input (XSS prevention)

### Files to Modify/Create

**New files:**
- `internal/messaging/schema.go` - SQLite schema definitions (~150 LOC)
- `internal/messaging/message.go` - Message CRUD operations (~300 LOC)
- `internal/messaging/thread.go` - Thread management (~200 LOC)
- `internal/messaging/approval.go` - Approval workflow logic (~250 LOC)
- `internal/messaging/directive.go` - Directive management (~150 LOC)
- `internal/websocket/server.go` - WebSocket server (~400 LOC)
- `internal/websocket/events.go` - Event types and handlers (~300 LOC)
- `ui/src/components/MessageCenter/` - React components (~800 LOC)
- `ui/src/components/OrchestrationBoard/` - Kanban board (~600 LOC)
- `ui/src/components/ApprovalQueue/` - Approval UI (~400 LOC)
- `ui/src/components/DirectionPanel/` - Creative direction (~300 LOC)
- `ui/src/components/InstanceManager/` - Instance config (~350 LOC)
- `ui/src/components/Timeline/` - Collaboration timeline (~250 LOC)
- `ui/src/hooks/useWebSocket.ts` - WebSocket React hook (~200 LOC)
- `ui/src/api/messages.ts` - API client for messaging (~300 LOC)

**Modified files:**
- `internal/runtime/runtime.go` - Add message polling and publishing (~100 LOC added)
- `internal/effects/context.go` - Add approval request mechanism (~50 LOC added)
- `cmd/ailang/main.go` - Add UI server command (~30 LOC added)
- `.ailang/state/messages/schema.sql` - Extend existing schema (~100 LOC added)

**Total estimated new code:** ~5,000-6,000 LOC (Go + TypeScript)

## Examples

### Example 1: Assigning New Work

**Workflow:**
1. Human opens UI, clicks "New Thread"
2. Enters directive: "Add user profile endpoints with validation"
3. @mentions `ailang-backend` or clicks "Auto-assign"
4. Sets priority: High, budget: $5
5. AILANG instance receives message via polling loop
6. Instance acknowledges: "Starting work on user profile endpoints"
7. Progress updates appear in conversation:
   - "Created GET /users/:id endpoint"
   - "Added validation middleware"
   - "Writing tests..."
8. Instance encounters design choice: "Should validation be strict (reject unknown fields) or permissive?"
9. Human responds: "Strict validation"
10. Instance completes, marks thread as "ready for review"

**UI Screenshot (conceptual):**
```
┌─────────────┬──────────────────────────────┬──────────────┐
│ ● Feature X │ You: "Add user profile       │ @ailang-1    │
│ ○ Bug #123  │      endpoints..."           │ Status: ⟳    │
│             │                              │ Progress:    │
│ + New       │ @ailang-1: "Starting work"   │ ████░░ 65%   │
│             │                              │              │
│             │ @ailang-1: "Created GET...   │ Cost: $1.23  │
│             │                              │              │
│             │ @ailang-1: "Should validation│ [View Code]  │
│             │  be strict or permissive?"   │ [View Tests] │
│             │                              │              │
│             │ You: "Strict validation"     │              │
│             │                              │              │
│             │ @ailang-1: "✓ Ready for     │              │
│             │  review"                     │              │
└─────────────┴──────────────────────────────┴──────────────┘
```

### Example 2: Multi-Agent Coordination

**Workflow:**
1. Human creates thread: "Refactor authentication system"
2. Spawns two instances:
   - `ailang-backend`: "Handle API changes"
   - `ailang-frontend`: "Update UI components"
3. Both subscribe to thread
4. Human posts context: "Must maintain backward compatibility"
5. Instances coordinate:
   - `ailang-backend`: "I'll expose new /v2/auth endpoints"
   - `ailang-frontend`: "I'll add feature flags for gradual rollout"
6. Human approves approach
7. Instances work in parallel
8. `ailang-backend` requests approval: "Modify users table schema?"
9. Human reviews impact, approves
10. Both instances complete, thread marked "resolved"

### Example 3: Handling Proposals

**Workflow:**
1. AILANG instance reaches critical decision: "Need to change database schema"
2. Creates approval request with details:
   - Change: Add `user_roles` table
   - Impact: High (migration required, 2,000 users)
   - Cost estimate: $0.45
   - Files: `migrations/003_add_user_roles.sql`, `internal/models/user.go`
3. Notification badge appears in UI
4. Human opens Approval Queue
5. Reviews proposal:
   - Views proposed SQL migration
   - Checks impact on existing users
   - Reviews cost estimate
6. Human chooses:
   - **Approve**: Instance proceeds immediately
   - **Reject**: Instance explores alternative approach
   - **Modify**: Human adds constraint: "Add rollback script too"

**Approval Queue UI (conceptual):**
```
┌─────────────────────────────────────────────────┐
│ ⚠ Requires Your Approval (1 pending)            │
├─────────────────────────────────────────────────┤
│ @ailang-backend proposes:                       │
│ "Add user_roles table to support role-based    │
│  access control"                                │
│                                                 │
│ Impact: HIGH                                     │
│ • Migration required (003_add_user_roles.sql)  │
│ • 2,000 existing users affected                 │
│ • Estimated cost: $0.45                         │
│ • Files: migrations/, internal/models/user.go  │
│                                                 │
│ [View Migration] [View Code] [View Tests]      │
│                                                 │
│ [Approve] [Reject] [Request Changes]           │
└─────────────────────────────────────────────────┘
```

## Success Criteria

- [ ] Humans can create threads and send directives to AILANG instances
- [ ] AILANG instances receive messages within 2 seconds (polling interval)
- [ ] Instances can send questions/status updates to humans in real-time
- [ ] Approval workflow supports approve/reject/modify actions
- [ ] Multiple instances can subscribe to same thread and coordinate
- [ ] Creative Direction panel shows active directives to all instances
- [ ] Orchestration Board provides drag-and-drop task management
- [ ] All human-AI interactions are stored in SQLite for deterministic replay
- [ ] WebSocket connections automatically reconnect on disconnect
- [ ] UI handles 100+ messages per thread without performance degradation
- [ ] All tests passing (backend 90%+ coverage, frontend 80%+ coverage)
- [ ] Documentation complete with user guide, API docs, and examples
- [ ] Integration with existing agent inbox infrastructure (`ailang agent inbox`)

## Testing Strategy (REVISED)

**Unit tests (90%+ coverage target):**
- Message CRUD with `message_seq` allocation
- Thread management (create, subscribe, update last_seq)
- Approval workflow (capability token generation/verification)
- WebSocket event serialization/deserialization
- Rate limiting (token bucket logic)
- React component rendering (Message Center, Approval Queue)

**Integration tests:**
- End-to-end message flow: human → SQLite → runtime polling → instance receives → responds
- WebSocket subscribe/ack/resume with `from_seq`
- Multi-client WebSocket broadcasting
- Approval request: instance asks → human approves → capability token issued → runtime verifies
- Migration: file-based inbox → SQLite (verify all messages preserved)

**Test bots (headless agents for soak testing):**

**1. Echo Agent**
```go
// Subscribes to thread, echoes every message back with "Received: <content>"
// Tests basic message flow and `message_seq` ordering
func NewEchoAgent(threadID string) *EchoAgent
```

**2. Slow/Flaky Agent**
```go
// Randomly disconnects, reconnects with old `from_seq`
// Tests WebSocket resume logic and duplicate message handling
// Simulates network flakiness
func NewFlakyAgent(threadID string, disconnectProbability float64) *FlakyAgent
```

**3. Human Bot (Auto-Approver)**
```go
// Auto-approves low-impact approval requests (<$1 budget, FS read-only)
// Auto-rejects high-impact (>$5 budget, DB writes)
// Tests approval workflow automation
func NewHumanBot(autoApproveThreshold float64) *HumanBot
```

**Load tests:**
- 50 threads × 1k messages each (50k total messages)
- Measure: DB write latency (target: <20ms p95), UI fetch latency, WebSocket throughput
- Verify: `message_seq` ordering, no duplicate messages, no dropped messages
- UI virtualized list handles 500+ messages without lag

**Security tests:**
- CSRF: Attempt POST without token (should fail 403)
- Unauthorized access: Instance A tries to read Instance B's messages (should fail 403)
- Token expiry: Use expired capability token (should fail 401)
- Rate limiting: Burst 200 requests in 1 minute (should throttle at 100)
- XSS: Inject `<script>alert(1)</script>` in message content (should be sanitized)

**Manual testing:**
- Human sends directive → instance receives within 2s → responds
- Instance requests approval → notification badge appears → human approves → instance proceeds
- Reconnect after network drop: messages resume from last ack_seq
- Multi-tab: message appears in all tabs simultaneously
- Search threads by title/status
- Performance with 100+ messages in conversation view

## Non-Goals

**Not in MVP (v0.4.4):**
- **Orchestration Board** - Defer to v0.4.5 (can derive from thread status)
- **Creative Direction Panel** - Defer to v0.4.5 (use simple directives in thread context_json)
- **Instance Manager polish** - Defer to v0.4.5 (spawn via API, configure via JSON)
- **Timeline component** - Defer to v0.4.5 (can query messages by created_at)
- **Template library** - Defer to v0.4.5 (start with 3 hard-coded directive templates)
- **Drag-and-drop UI** - Defer to v0.4.5
- **Advanced analytics** - Defer to v0.5.0 (track metrics, but don't visualize yet)
- **Mobile app** - Web UI only for v0.4.4
- **Video/audio chat** - Text-based collaboration only
- **External integrations** (Slack, Discord) - Defer to v0.5.0
- **Multi-tenancy** - Single user/team for v0.4.4
- **Fine-grained RBAC** - Basic human/instance roles only
- **Message encryption** - Assume trusted environment; HTTPS in transit is sufficient
- **AI-to-AI direct messaging** - All coordination visible to humans via message bus

## Timeline (REVISED - MVP Focus)

**Week 1** (~40 hours):
- Phase 1: Database & Schema Migration
  - Create SQLite schema with ordering guarantees
  - Migration tool (file-based → SQLite)
  - Update `ailang agent inbox/ack` commands
  - Benchmarks and tests
- Phase 2 (start): Backend Messaging Core
  - Message CRUD with `message_seq` allocation
  - Thread management

**Week 2** (~45 hours):
- Phase 2 (complete): Backend Messaging Core
  - Approval workflow with capability tokens
  - WebSocket server with `from_seq` resume
  - Runtime integration (polling, effect hooks)
  - Tests and integration tests
- Phase 3 (start): Core UI
  - React setup
  - WebSocket client

**Week 3** (~40 hours):
- Phase 3 (complete): Core UI
  - Message Center (threads + conversation)
  - Approval Queue
  - Real-time updates
  - UI tests
- Phase 4: Security & Auth
  - Session auth, PATs, CSRF, rate limiting
  - Security tests

**Weeks 4-5 (optional polish)** (~20 hours):
- Phase 5: Polish & Testing
  - Search, notifications, error handling
  - Performance optimization
  - End-to-end tests
- Phase 6: Documentation
  - User guide, API docs, migration guide

**Total MVP: ~125 hours (3 weeks core + 1 week polish)**

**With 2 developers:** 2 weeks core + 1 week polish = **3 weeks to MVP**

## Risks & Mitigations

| Risk | Impact | Mitigation |
|------|--------|-----------|
| **WebSocket scalability** with many instances | High | Implement connection pooling, message batching; benchmark with 50+ instances early |
| **SQLite write contention** under high message volume | Medium | Use WAL mode; consider message buffering; benchmark write throughput |
| **Complex UI state management** across components | Medium | Use Redux or Zustand for centralized state; clear data flow architecture |
| **Message ordering** issues in distributed system | High | Use timestamp + sequence numbers; implement vector clocks if needed |
| **Approval workflow deadlocks** (instance waiting indefinitely) | Medium | Implement timeout mechanism (24h default); auto-reject or escalate |
| **Integration breaks existing agent inbox** | High | Maintain backward compatibility with CLI commands; extensive integration tests |
| **UI performance** with 1000+ messages | Low | Implement virtualized scrolling (react-window); pagination for old messages |

## References

- [Existing Agent Inbox Infrastructure](../../../.ailang/state/messages/) - Current messaging system
- [SessionStart Hook](../../../scripts/hooks/session_start.sh) - Hook that checks inbox on session start
- [SQLite WAL Mode](https://www.sqlite.org/wal.html) - For write concurrency
- [WebSocket Protocol](https://datatracker.ietf.org/doc/html/rfc6455) - Real-time bidirectional communication
- [React Virtualized](https://github.com/bvaughn/react-virtualized) - For message list performance
- [Vector Clocks](https://en.wikipedia.org/wiki/Vector_clock) - For distributed message ordering if needed

## Future Work

**v0.5.0 - Collaboration Analytics:**
- Dashboard showing human-AI collaboration metrics
- Token cost attribution per thread/instance
- Success rate of different directive styles
- Time-to-completion analysis

**v0.5.0 - External Integrations:**
- Slack/Discord notifications for approval requests
- GitHub integration (link threads to PRs, issues)
- Export message threads to JSON/CSV for analysis

**v0.5.0 - Advanced Features:**
- Voice input for directives
- Visual diff viewer for code proposals
- A/B testing different directive formulations
- Recommendation engine for directive templates

**v0.6.0 - Multi-Tenancy:**
- Team workspaces with role-based access
- Shared directive libraries across teams
- Instance pools per team

---

**Document created**: 2025-11-08
**Last updated**: 2025-11-08
