# M-HTTP-HOOKS-CLOUD-TELEMETRY: Claude Code HTTP Hooks for Cloud-Native Telemetry

**Status**: Implemented
**Target**: v0.8.2
**Priority**: P1 (Medium-High)
**Estimated**: 1-2 weeks
**Dependencies**: [M-CLOUD-INFRA](m-cloud-infra.md) (GCP deployment target)
**Author**: Claude + Mark
**Created**: 2026-03-05

---

## Executive Summary

Replace AILANG's shell-script Claude Code hooks (`claude_telemetry.sh`) with Claude Code's native HTTP hooks (`type: "http"`). This eliminates the bash/jq/curl intermediary, enables cloud-native telemetry without requiring the `ailang` CLI on remote machines, and provides the foundation for M-CLOUD-INFRA by making hook endpoints deployable to Cloud Run.

**Scope**: Claude Code only. Gemini CLI has no hook mechanism (GCP-native telemetry only). OpenAI/Codex is API-only with no executor.

---

## Problem Statement

### Current Architecture: Shell Script Intermediary

```
Claude Code event (JSON on stdin)
    │
    ▼
~/.ailang/hooks/claude_telemetry.sh  (250 LOC bash)
    │  ├── jq: parse JSON from stdin
    │  ├── jq: extract fields
    │  ├── jq: build new payload
    │  ├── bash: read AILANG_* env vars
    │  └── curl: POST to localhost:1957
    ▼
AILANG Server (/api/observatory/hooks, /api/exec/events)
```

**Problems:**

1. **Shell dependency**: Requires `bash`, `jq`, `curl` on every machine running Claude Code
   - Cloud Run containers need extra packages
   - CI/CD environments may not have `jq`
   - Windows/WSL compatibility issues

2. **Two-hop latency**: Claude Code → bash script → curl → server (adds ~50-100ms per event)

3. **Information loss**: The bash script extracts and re-serializes JSON, losing fields not explicitly handled
   - New Claude Code hook fields (e.g., `agent_id`, `agent_type`, `permission_mode`) are silently dropped
   - Correlation IDs come from env vars, not the JSON payload

4. **Fragile parsing**: `jq` calls can fail silently, truncation of `tool_response` is approximate

5. **No auth story**: Shell script has no mechanism for bearer tokens or API keys for cloud endpoints

6. **Dual endpoint problem**: Same hook script POSTs to two different endpoints (`/api/observatory/hooks` AND `/api/exec/events`), duplicating data

### What Claude Code HTTP Hooks Provide Natively

Claude Code (as of 2026) supports `type: "http"` hooks that:

- POST the **full** hook JSON payload directly to a URL (zero information loss)
- Support `headers` with env var interpolation (`$VAR_NAME` syntax)
- Have `allowedEnvVars` for security (only listed vars are resolved)
- Are **non-blocking** on failure (connection errors, timeouts, non-2xx don't break the session)
- Can return JSON decisions (block tools, inject context, deny permissions)
- Run in parallel with other hooks for the same event

---

## Goals

### Primary Goal

Migrate from command hooks to HTTP hooks for all telemetry, gaining cloud-readiness with zero information loss.

### Success Metrics

| Metric | Current | Target |
|--------|---------|--------|
| Shell dependencies for telemetry | bash + jq + curl | None (HTTP native) |
| Hook payload fields captured | ~8 (manually extracted) | All (full JSON forwarded) |
| Cloud deployment complexity | Install jq/curl in container | Just set URL + token |
| Per-event overhead | ~80ms (fork + jq + curl) | ~5ms (HTTP POST) |
| Auth for cloud endpoints | None | Bearer token via env var |

### Non-Goals

- Gemini CLI hook support (Gemini has no hook mechanism)
- OpenAI/Codex hook support (API-only, no executor)
- Replacing the `session_start.sh` hook (that injects message context, not telemetry)
- Replacing the `agent_handoff.sh` Stop hook (that sends AILANG messages)
- Replacing the `format_go.sh` PostToolUse hook (local-only, needs filesystem)

---

## Solution Design

### Architecture Overview

```
BEFORE (command hooks):
┌─────────────┐     stdin      ┌────────────────────┐     curl      ┌───────────────┐
│ Claude Code  │───────────────►│ claude_telemetry.sh │──────────────►│ AILANG Server │
│ (hook event) │               │ (bash + jq + curl)  │              │ :1957          │
└─────────────┘               └────────────────────┘              └───────────────┘

AFTER (HTTP hooks):
┌─────────────┐     HTTP POST (full JSON)      ┌───────────────┐
│ Claude Code  │──────────────────────────────►│ AILANG Server │
│ (hook event) │                                │ :1957          │
└─────────────┘                                └───────────────┘

CLOUD:
┌─────────────┐     HTTP POST + Bearer token   ┌───────────────────────┐
│ Claude Code  │──────────────────────────────►│ Cloud Run             │
│ (any machine)│                                │ ailang-hub.run.app    │
└─────────────┘                                └───────────────────────┘
```

### Phase 1: Unified Hook Receiver Endpoint (Server-side)

Create a single endpoint that accepts the raw Claude Code hook JSON and routes internally.

**New endpoint:** `POST /api/hooks/claude`

This replaces the current split between `/api/observatory/hooks` and `/api/exec/events`. One POST, server-side fan-out.

```go
// internal/server/handlers_claude_hooks.go

// HandleClaudeHook receives raw Claude Code hook JSON and routes it.
// The JSON is the FULL payload from Claude Code — no field extraction needed.
func (s *Server) HandleClaudeHook(w http.ResponseWriter, r *http.Request) {
    var payload ClaudeHookPayload
    if err := json.NewDecoder(r.Body).Decode(&payload); err != nil {
        http.Error(w, "invalid JSON", http.StatusBadRequest)
        return
    }

    // Extract correlation IDs from headers (set by coordinator via allowedEnvVars)
    payload.TaskID = r.Header.Get("X-AILANG-Task-ID")
    payload.ChainID = r.Header.Get("X-AILANG-Chain-ID")
    payload.StageID = r.Header.Get("X-AILANG-Stage-ID")
    payload.MessageID = r.Header.Get("X-AILANG-Message-ID")

    switch payload.HookEventName {
    case "SessionStart":
        s.handleSessionStart(payload)
    case "PreToolUse":
        s.handlePreToolUse(payload)
    case "PostToolUse":
        s.handlePostToolUse(payload)
    case "Stop":
        s.handleStop(payload)
    // NEW events we couldn't capture before:
    case "SubagentStart", "SubagentStop":
        s.handleSubagentEvent(payload)
    case "TaskCompleted":
        s.handleTaskCompleted(payload)
    case "PostToolUseFailure":
        s.handleToolFailure(payload)
    }

    w.WriteHeader(http.StatusOK)
}

// ClaudeHookPayload matches Claude Code's hook JSON schema exactly.
// We store the FULL payload — no field subsetting.
type ClaudeHookPayload struct {
    // Common fields (always present)
    SessionID      string `json:"session_id"`
    TranscriptPath string `json:"transcript_path"`
    Cwd            string `json:"cwd"`
    PermissionMode string `json:"permission_mode"`
    HookEventName  string `json:"hook_event_name"`

    // Agent fields (present in --agent mode or subagents)
    AgentID   string `json:"agent_id,omitempty"`
    AgentType string `json:"agent_type,omitempty"`

    // Tool fields (PreToolUse, PostToolUse, PostToolUseFailure)
    ToolName     string          `json:"tool_name,omitempty"`
    ToolInput    json.RawMessage `json:"tool_input,omitempty"`
    ToolResponse json.RawMessage `json:"tool_response,omitempty"`
    ToolUseID    string          `json:"tool_use_id,omitempty"`

    // SessionStart fields
    Source string `json:"source,omitempty"`
    Model  string `json:"model,omitempty"`

    // AILANG correlation IDs (from headers, not JSON body)
    TaskID    string `json:"-"`
    ChainID   string `json:"-"`
    StageID   string `json:"-"`
    MessageID string `json:"-"`
}
```

### Phase 2: HTTP Hook Configuration

Replace command hooks with HTTP hooks in `.claude/settings.json`:

```json
{
  "hooks": {
    "SessionStart": [
      {
        "matcher": "",
        "hooks": [
          {
            "type": "command",
            "command": "\"$CLAUDE_PROJECT_DIR\"/scripts/hooks/session_start.sh"
          },
          {
            "type": "command",
            "command": "\"$CLAUDE_PROJECT_DIR\"/scripts/hooks/cloud_setup.sh"
          },
          {
            "type": "http",
            "url": "${AILANG_HUB_URL}/api/hooks/claude",
            "timeout": 5,
            "headers": {
              "X-AILANG-Task-ID": "$AILANG_TASK_ID",
              "X-AILANG-Chain-ID": "$AILANG_CHAIN_ID",
              "X-AILANG-Stage-ID": "$AILANG_STAGE_ID",
              "X-AILANG-Message-ID": "$AILANG_MESSAGE_ID",
              "Authorization": "Bearer $AILANG_HUB_TOKEN"
            },
            "allowedEnvVars": [
              "AILANG_HUB_URL",
              "AILANG_HUB_TOKEN",
              "AILANG_TASK_ID",
              "AILANG_CHAIN_ID",
              "AILANG_STAGE_ID",
              "AILANG_MESSAGE_ID"
            ]
          }
        ]
      }
    ],
    "PreToolUse": [
      {
        "matcher": "*",
        "hooks": [
          {
            "type": "http",
            "url": "${AILANG_HUB_URL}/api/hooks/claude",
            "timeout": 3,
            "headers": {
              "X-AILANG-Task-ID": "$AILANG_TASK_ID",
              "X-AILANG-Chain-ID": "$AILANG_CHAIN_ID",
              "X-AILANG-Stage-ID": "$AILANG_STAGE_ID",
              "Authorization": "Bearer $AILANG_HUB_TOKEN"
            },
            "allowedEnvVars": [
              "AILANG_HUB_URL",
              "AILANG_HUB_TOKEN",
              "AILANG_TASK_ID",
              "AILANG_CHAIN_ID",
              "AILANG_STAGE_ID"
            ]
          }
        ]
      }
    ],
    "PostToolUse": [
      {
        "matcher": "Edit|Write",
        "hooks": [
          {
            "type": "command",
            "command": "\"$CLAUDE_PROJECT_DIR\"/scripts/hooks/format_go.sh"
          }
        ]
      },
      {
        "matcher": "*",
        "hooks": [
          {
            "type": "http",
            "url": "${AILANG_HUB_URL}/api/hooks/claude",
            "timeout": 3,
            "headers": {
              "X-AILANG-Task-ID": "$AILANG_TASK_ID",
              "X-AILANG-Chain-ID": "$AILANG_CHAIN_ID",
              "X-AILANG-Stage-ID": "$AILANG_STAGE_ID",
              "Authorization": "Bearer $AILANG_HUB_TOKEN"
            },
            "allowedEnvVars": [
              "AILANG_HUB_URL",
              "AILANG_HUB_TOKEN",
              "AILANG_TASK_ID",
              "AILANG_CHAIN_ID",
              "AILANG_STAGE_ID"
            ]
          }
        ]
      }
    ],
    "PostToolUseFailure": [
      {
        "matcher": "*",
        "hooks": [
          {
            "type": "http",
            "url": "${AILANG_HUB_URL}/api/hooks/claude",
            "timeout": 3,
            "headers": {
              "Authorization": "Bearer $AILANG_HUB_TOKEN"
            },
            "allowedEnvVars": ["AILANG_HUB_URL", "AILANG_HUB_TOKEN"]
          }
        ]
      }
    ],
    "SubagentStart": [
      {
        "matcher": "",
        "hooks": [
          {
            "type": "http",
            "url": "${AILANG_HUB_URL}/api/hooks/claude",
            "timeout": 3,
            "headers": {
              "X-AILANG-Task-ID": "$AILANG_TASK_ID",
              "X-AILANG-Chain-ID": "$AILANG_CHAIN_ID",
              "Authorization": "Bearer $AILANG_HUB_TOKEN"
            },
            "allowedEnvVars": [
              "AILANG_HUB_URL",
              "AILANG_HUB_TOKEN",
              "AILANG_TASK_ID",
              "AILANG_CHAIN_ID"
            ]
          }
        ]
      }
    ],
    "SubagentStop": [
      {
        "matcher": "",
        "hooks": [
          {
            "type": "http",
            "url": "${AILANG_HUB_URL}/api/hooks/claude",
            "timeout": 3,
            "headers": {
              "X-AILANG-Task-ID": "$AILANG_TASK_ID",
              "X-AILANG-Chain-ID": "$AILANG_CHAIN_ID",
              "Authorization": "Bearer $AILANG_HUB_TOKEN"
            },
            "allowedEnvVars": [
              "AILANG_HUB_URL",
              "AILANG_HUB_TOKEN",
              "AILANG_TASK_ID",
              "AILANG_CHAIN_ID"
            ]
          }
        ]
      }
    ],
    "TaskCompleted": [
      {
        "hooks": [
          {
            "type": "http",
            "url": "${AILANG_HUB_URL}/api/hooks/claude",
            "timeout": 5,
            "headers": {
              "X-AILANG-Task-ID": "$AILANG_TASK_ID",
              "X-AILANG-Chain-ID": "$AILANG_CHAIN_ID",
              "Authorization": "Bearer $AILANG_HUB_TOKEN"
            },
            "allowedEnvVars": [
              "AILANG_HUB_URL",
              "AILANG_HUB_TOKEN",
              "AILANG_TASK_ID",
              "AILANG_CHAIN_ID"
            ]
          }
        ]
      }
    ],
    "Stop": [
      {
        "hooks": [
          {
            "type": "command",
            "command": "bash \"$CLAUDE_PROJECT_DIR\"/scripts/hooks/agent_handoff.sh",
            "timeout": 30
          },
          {
            "type": "http",
            "url": "${AILANG_HUB_URL}/api/hooks/claude",
            "timeout": 5,
            "headers": {
              "X-AILANG-Task-ID": "$AILANG_TASK_ID",
              "X-AILANG-Chain-ID": "$AILANG_CHAIN_ID",
              "Authorization": "Bearer $AILANG_HUB_TOKEN"
            },
            "allowedEnvVars": [
              "AILANG_HUB_URL",
              "AILANG_HUB_TOKEN",
              "AILANG_TASK_ID",
              "AILANG_CHAIN_ID"
            ]
          },
          {
            "type": "command",
            "command": "~/.claude/hooks/session_end_speak.sh",
            "timeout": 120
          }
        ]
      }
    ]
  }
}
```

**Key design decisions:**

1. **Single URL for all events** — The server routes by `hook_event_name` in the JSON body, not the URL path. One HTTP hook config pattern, copy-paste friendly.

2. **Correlation IDs via headers** — Claude Code's `allowedEnvVars` + `headers` feature lets us pass `AILANG_TASK_ID` etc. as HTTP headers. The server extracts them without modifying the JSON body.

3. **Command hooks retained for local-only work** — `format_go.sh` (needs filesystem), `session_start.sh` (injects message context into Claude), `agent_handoff.sh` (sends AILANG messages), `session_end_speak.sh` (TTS).

4. **New events captured** — `SubagentStart`, `SubagentStop`, `TaskCompleted`, `PostToolUseFailure` are events we couldn't easily capture before. They provide subagent hierarchy tracking and failure diagnostics.

### Phase 3: Coordinator Environment Injection

The coordinator already injects `AILANG_TASK_ID`, `AILANG_CHAIN_ID`, etc. as environment variables when spawning Claude Code. These flow automatically into HTTP hook headers via `allowedEnvVars`.

Update `internal/executor/environment.go` to also set `AILANG_HUB_URL`:

```go
func BuildEnvironment(task *Task, config *Config) []string {
    env := os.Environ()

    // Existing correlation IDs (already injected)
    env = append(env, "AILANG_TASK_ID="+task.ID)
    env = append(env, "AILANG_CHAIN_ID="+task.ChainID)
    env = append(env, "AILANG_STAGE_ID="+task.StageID)
    env = append(env, "AILANG_MESSAGE_ID="+task.MessageID)

    // NEW: Hub URL for HTTP hooks
    hubURL := config.HubURL // from ~/.ailang/config.yaml
    if hubURL == "" {
        hubURL = "http://127.0.0.1:1957" // local default
    }
    env = append(env, "AILANG_HUB_URL="+hubURL)

    // NEW: Hub auth token for cloud deployments
    if config.HubToken != "" {
        env = append(env, "AILANG_HUB_TOKEN="+config.HubToken)
    }

    return env
}
```

### Phase 4: Cloud Configuration

Add hub configuration to `~/.ailang/config.yaml`:

```yaml
# Hub configuration (for HTTP hooks)
hub:
  # Local development (default)
  url: http://127.0.0.1:1957

  # Cloud deployment (uncomment for Cloud Run)
  # url: https://ailang-hub-HASH.run.app
  # token: ${AILANG_HUB_TOKEN}  # or read from Secret Manager

  # Auth method for cloud
  # auth:
  #   type: bearer          # bearer | gcp-identity | none
  #   token_source: env     # env | secret-manager | file
  #   token_var: AILANG_HUB_TOKEN
```

### Phase 5: Auth Middleware (Cloud Only)

For cloud deployment, add auth middleware to the hub server:

```go
// internal/server/middleware_auth.go

func (s *Server) AuthMiddleware(next http.Handler) http.Handler {
    return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
        // Skip auth for health checks
        if r.URL.Path == "/health" {
            next.ServeHTTP(w, r)
            return
        }

        // Skip auth in local mode (no token configured)
        if s.config.HubToken == "" {
            next.ServeHTTP(w, r)
            return
        }

        // Validate bearer token
        auth := r.Header.Get("Authorization")
        if auth != "Bearer "+s.config.HubToken {
            http.Error(w, "unauthorized", http.StatusUnauthorized)
            return
        }

        next.ServeHTTP(w, r)
    })
}
```

---

## What Stays as Command Hooks

Not everything migrates to HTTP. These hooks remain `type: "command"`:

| Hook | Why it stays local |
|------|--------------------|
| `session_start.sh` | Injects AILANG inbox messages into Claude's context via stdout. HTTP hooks can return `additionalContext` but would need the server to query messages — adds latency to session start. **Migration candidate for Phase 2 of M-CLOUD-INFRA.** |
| `agent_handoff.sh` | Detects design docs created in session and sends `ailang messages send` to sprint-planner inbox. Requires `ailang` CLI. **Migration candidate: server-side webhook on Stop event.** |
| `format_go.sh` | Runs `gofmt` on edited Go files. Purely local filesystem operation. **No migration needed.** |
| `session_end_speak.sh` | Text-to-speech on session end. Local audio output. **No migration needed.** |
| `cloud_setup.sh` | Installs local tools (Go, make, gh) on cloud VMs. **No migration needed.** |

---

## New Events We Gain

HTTP hooks make it easy to capture events that were too expensive to capture via shell scripts:

| Event | What it tells us | Observatory value |
|-------|-----------------|-------------------|
| `SubagentStart` | Agent type, parent session | Build subagent hierarchy tree |
| `SubagentStop` | Completion, duration | Track subagent cost/performance |
| `TaskCompleted` | Task marked done | End-to-end task duration |
| `PostToolUseFailure` | Tool errors | Failure pattern analysis |
| `PreCompact` | Context about to be compacted | Audit context window usage |
| `ConfigChange` | Config modified mid-session | Track config drift |

These events replace the manual chain tracking that the coordinator currently does, and solve the TRACEPARENT propagation problem documented in CLAUDE.md — HTTP hooks send `session_id` and `agent_id` directly, no OTEL trace linking needed.

---

## Migration Plan

### Step 1: Server endpoint (no breaking changes)

- Add `POST /api/hooks/claude` endpoint
- Accepts full Claude Code hook JSON
- Routes to existing observatory + exec event storage
- **Old endpoints stay**: `/api/observatory/hooks` and `/api/exec/events` unchanged
- Tests: unit tests for payload parsing, integration test with sample hook JSON

### Step 2: Dual-mode hooks (transition period)

- Add HTTP hooks alongside existing command hooks in `.claude/settings.json`
- Both fire in parallel (Claude Code runs all matching hooks)
- Verify HTTP hook data matches command hook data in observatory
- Monitor `telemetry_hooks.log` for command hook calls → should still see them

### Step 3: Remove command telemetry hooks

- Remove `claude_telemetry.sh` from SessionStart, PreToolUse, PostToolUse, Stop
- Remove `~/.ailang/hooks/claude_telemetry.sh` reference from `~/.claude/settings.json`
- Keep `claude_telemetry.sh` in repo for Gemini executor reference (it's embedded anyway)
- Update CLAUDE.md hook documentation

### Step 4: Add new event hooks

- Add SubagentStart, SubagentStop, TaskCompleted, PostToolUseFailure hooks
- Add server-side handling for new events
- Update observatory schema if needed for subagent hierarchy

### Step 5: Cloud deployment config

- Add `hub` section to `~/.ailang/config.yaml`
- Add auth middleware to server
- Document cloud setup in M-CLOUD-INFRA
- Coordinator injects `AILANG_HUB_URL` and `AILANG_HUB_TOKEN` into Claude Code environment

---

## Interaction with M-CLOUD-INFRA

This design doc is a **prerequisite** for M-CLOUD-INFRA's agent telemetry story:

| M-CLOUD-INFRA requirement | How M-HTTP-HOOKS solves it |
|---------------------------|---------------------------|
| "Agents running on Cloud Run report telemetry" | HTTP hooks POST to Cloud Run URL |
| "No ailang CLI needed on agent containers" | HTTP hooks are built into Claude Code |
| "Auth for multi-tenant hub" | Bearer token via `allowedEnvVars` |
| "Real-time dashboard updates" | Server receives events, broadcasts via WebSocket (existing) |
| "Subagent hierarchy in observatory" | SubagentStart/SubagentStop events |
| "Cross-machine correlation" | AILANG_TASK_ID etc. in HTTP headers |

**Dependency direction**: M-HTTP-HOOKS (this doc) should be implemented first, tested locally, then M-CLOUD-INFRA deploys the hub server to Cloud Run and points `AILANG_HUB_URL` at it.

---

## Scope Exclusions

### Gemini CLI

Gemini CLI does **not** have a hooks mechanism. Its telemetry goes directly to GCP Cloud Logging via built-in OTEL exporters configured with environment variables:

```
GEMINI_TELEMETRY_ENABLED=true
GEMINI_TELEMETRY_TARGET=gcp
```

For cloud deployment, Gemini agent telemetry will be collected by querying Cloud Logging, not via hooks. This is handled by M-CLOUD-INFRA separately.

### OpenAI Codex

Codex is API-only in AILANG (`internal/ai/openai/`). There is no executor implementation and no hook mechanism. Codex telemetry comes from the `ai.Provider` interface, which already reports via OTEL spans.

### Prompt/Agent Hooks

Claude Code also supports `type: "prompt"` and `type: "agent"` hooks (LLM-based validation). These are interesting for future work (e.g., "is this code change safe?") but are out of scope for telemetry migration.

---

## Risks

| Risk | Mitigation |
|------|------------|
| HTTP hook errors are non-blocking (silent failures) | Server health check endpoint; `ailang trace status` validates connectivity |
| `allowedEnvVars` missing → empty headers | Validation in coordinator; log warning if correlation IDs are empty |
| Payload too large (tool_response can be huge) | Claude Code already truncates; server truncates at 10KB per field |
| Cloud Run cold starts add latency | Use min-instances=1; hooks have 3-5s timeout anyway |
| Token leakage in settings.json | Use env var interpolation, never literal tokens in config |

---

## Implementation Estimate

| Phase | Effort | Files |
|-------|--------|-------|
| Phase 1: Server endpoint | 1 day | `handlers_claude_hooks.go` (new, ~200 LOC), `server.go` (route registration) |
| Phase 2: Dual-mode config | 0.5 day | `.claude/settings.json` (add HTTP hooks alongside command hooks) |
| Phase 3: Coordinator env | 0.5 day | `environment.go` (~10 lines), `config.yaml` schema |
| Phase 4: Remove command hooks | 0.5 day | `.claude/settings.json`, verify no regressions |
| Phase 5: New events | 1 day | `handlers_claude_hooks.go` (extend), observatory schema |
| Phase 6: Auth middleware | 1 day | `middleware_auth.go` (new, ~50 LOC), config parsing |
| Testing | 1-2 days | Integration tests, local + simulated cloud |
| **Total** | **5-7 days** | |

---

## Files to Modify/Create

**New files:**
- `internal/server/handlers_claude_hooks.go` (~200 LOC) - Unified hook receiver endpoint
- `internal/server/handlers_claude_hooks_test.go` (~300 LOC) - Unit + integration tests
- `internal/server/middleware_auth.go` (~50 LOC) - Bearer token auth for cloud

**Modified files:**
- `internal/server/server.go` (+5 LOC) - Register `/api/hooks/claude` route
- `internal/executor/environment.go` (+10 LOC) - Inject `AILANG_HUB_URL`, `AILANG_HUB_TOKEN`
- `internal/coordinator/config.go` (+15 LOC) - Parse `hub` config section
- `.claude/settings.json` (+80/-30 LOC) - Replace command hooks with HTTP hooks

**Unchanged (retained as-is):**
- `internal/executor/claude_telemetry.sh` - Kept in repo, removed from hook config
- `scripts/hooks/session_start.sh` - Stays as command hook
- `scripts/hooks/agent_handoff.sh` - Stays as command hook
- `scripts/hooks/format_go.sh` - Stays as command hook

---

## Testing Strategy

**Unit tests:**
- `ClaudeHookPayload` JSON deserialization (all event types)
- Header extraction for correlation IDs
- Event routing (switch on `hook_event_name`)
- Auth middleware (valid token, invalid token, no token configured)

**Integration tests:**
- POST sample hook JSON to `/api/hooks/claude`, verify observatory rows created
- POST with correlation headers, verify task/chain linking
- POST with no server running, verify Claude Code continues (non-blocking)
- Dual-mode: fire both command and HTTP hooks, compare stored data

**Manual testing:**
- Run `ailang serve` + Claude Code session with HTTP hooks enabled
- Verify events appear in dashboard WebSocket stream
- Test with `AILANG_HUB_URL` pointing to localhost vs invalid URL
- Verify `telemetry_hooks.log` stops receiving entries after command hooks removed

---

## Success Criteria

- [ ] `POST /api/hooks/claude` accepts all 7 Claude Code event types
- [ ] Full JSON payload stored (no field subsetting)
- [ ] Correlation IDs flow from coordinator env → HTTP headers → observatory
- [ ] `claude_telemetry.sh` removed from all hook configs (no shell dependency)
- [ ] SubagentStart/SubagentStop events create hierarchy in observatory
- [ ] Auth middleware blocks unauthenticated requests when token configured
- [ ] All existing tests passing (`make test`)
- [ ] Dashboard WebSocket still receives live events
- [ ] Documentation updated (CLAUDE.md hooks section)

---

## Axiom Compliance

**Canonical reference:** [Design Axioms](/docs/references/axioms)

### Scoring

| Axiom | Score | Justification |
|-------|-------|---------------|
| A1: Determinism | 0 | No change to language semantics |
| A2: Replayability | +1 | Full hook payloads stored (no information loss), better trace reconstruction |
| A3: Effect Legibility | +1 | More events captured (SubagentStart/Stop, TaskCompleted), effects more visible |
| A4: Explicit Authority | +1 | Bearer token auth makes access control explicit; `allowedEnvVars` whitelisting |
| A5: Bounded Verification | 0 | No change to verification |
| A6: Safe Concurrency | 0 | No concurrency impact |
| A7: Machines First | +1 | Eliminates bash/jq intermediary; machine-native HTTP protocol |
| A8: Minimal Syntax | 0 | No new syntax |
| A9: Cost Visibility | +1 | Better telemetry enables cost tracking per agent/subagent |
| A10: Composability | +1 | HTTP hooks compose with any server (local, Cloud Run, etc.) |
| A11: Structured Failure | 0 | No error handling changes |
| A12: System Boundary | +1 | Cloud/local boundary made explicit via `AILANG_HUB_URL` config |

**Net Score: +7** ✅ Proceed to implementation

### Hard Violation Check

- [x] A1 (Determinism): No implicit nondeterminism introduced
- [x] A3 (Effects): No hidden side effects (HTTP hooks are explicit in config)
- [x] A4 (Authority): No ambient access granted (token required for cloud)
- [x] A7 (Machines First): Removes human-oriented shell scripting in favor of machine-native HTTP

---

## Related Documents

**Implemented (prior art):**
- [M-SESSION-WORKSPACE](../../implemented/v0_7_0/m-session-workspace-hooks.md) - Current hook telemetry system this replaces
- [M-TELEMETRY-HOOKS](../../implemented/v0_8_0/m-telemetry-hooks-handoff.md) - Worktree session enrichment
- [M-OTEL](../../implemented/v0_6_2/m-otel-integration.md) - OpenTelemetry integration foundation
- [M-CONTROL-PLANE-V4](../../implemented/v0_7_0/m-control-plane-v4-integration.md) - Unified telemetry for agent monitoring
- [M-OTEL-CROSS-PROCESS](../../implemented/v0_7_0/m-otel-cross-process-linking.md) - Cross-process trace linking (TRACEPARENT limitation documented here)

**Planned (coordinates with):**
- [M-CLOUD-INFRA](m-cloud-infra.md) - GCP deployment target (this doc is a prerequisite)
- [Global Collaboration Hub](../v1_0_0/global-collaboration-hub.md) - Long-term cloud vision

---

## Future Work

- **Session start via HTTP**: Migrate `session_start.sh` to HTTP hook returning `additionalContext` with inbox messages from server API
- **Agent handoff via HTTP**: Replace `agent_handoff.sh` with server-side Stop event handler that sends messages
- **Prompt hooks for safety**: Use `type: "prompt"` hooks for LLM-based code review before tool execution
- **Agent hooks for verification**: Use `type: "agent"` hooks to spawn verification subagents
- **GCP Identity Token auth**: Replace bearer tokens with GCP IAM identity tokens for zero-secret cloud auth

---

## Open Questions

1. **Should `session_start.sh` also migrate?** It returns `additionalContext` (inbox messages). HTTP hooks support this via JSON response body. But it currently runs `ailang messages list --unread` which needs the CLI. Could the server provide an endpoint that returns context for SessionStart? This would make Claude Code fully CLI-independent.

2. **Global vs project hooks?** Currently telemetry hooks are in project `.claude/settings.json`. For cloud deployment, they should be in `~/.claude/settings.json` (user-level) so they work across all projects. Or should the coordinator inject them per-agent?

3. **Should we deduplicate the two existing server endpoints?** `/api/observatory/hooks` and `/api/exec/events` store overlapping data. The new `/api/hooks/claude` could unify them, but we'd need to maintain backward compatibility during transition.

---

## Implementation Report

**Implemented**: 2026-03-05
**Actual Duration**: ~3 hours (single session)
**Total New Code**: ~350 LOC (implementation) + ~460 LOC (tests)

### Files Created
| File | LOC | Purpose |
|------|-----|---------|
| `internal/server/handlers_claude_hooks.go` | ~295 | Unified hook receiver with event routing |
| `internal/server/handlers_claude_hooks_test.go` | ~760 | 25 tests + 2 benchmarks |

### Files Modified
| File | Change |
|------|--------|
| `internal/server/server.go` | Added `hookToken` field, `WithHookToken` option, route registration |
| `.claude/settings.json` | Removed `claude_telemetry.sh`, added HTTP hooks for 7 events |
| `internal/executor/claude_settings.json` | HTTP-only hooks (no command hooks for coordinator sessions) |
| `scripts/hooks/claude_settings.json` | HTTP-only hooks |
| `cmd/ailang/server.go` | Wire `AILANG_HUB_TOKEN` env var to `WithHookToken` |

### Key Decisions
1. **URL validation constraint**: Claude Code requires `"format": "uri"` for HTTP hook URLs, so env var interpolation (`${AILANG_HUB_URL}`) doesn't work in the URL field. Used concrete `http://127.0.0.1:1957/api/hooks/claude`. Cloud deployments override via `settings.local.json`.
2. **Auth inline vs middleware**: Added bearer token check directly in `handleClaudeHooks` rather than a separate middleware wrapper, keeping the implementation simpler.
3. **SubagentStart/Stop as synthetic tools**: Stored in existing `session_tools` table with `Subagent:<agent_type>` tool name to avoid schema changes.
4. **Embedded settings HTTP-only**: Coordinator-spawned sessions use HTTP-only hooks (no `claude_telemetry.sh`) since the server is always available for coordinator tasks.
5. **Bash script kept in repo**: `claude_telemetry.sh` files kept in place per design doc guidance, just removed from hook configurations.

### Test Coverage
- 25 tests covering all event types, auth, edge cases
- 2 benchmarks (SessionStart, PreToolUse)
- All existing tests continue to pass
