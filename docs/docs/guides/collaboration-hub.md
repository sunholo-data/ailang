---
sidebar_position: 13
title: Collaboration Hub
description: Human-AI collaboration via the web UI and background agent
---

# Collaboration Hub Guide

The AILANG Collaboration Hub enables human-AI collaboration through a web UI and background agent.

## Installation

### Building from Source

```bash
# Build ailang-agent only
make build-agent

# Or build all binaries (ailang + ailang-agent)
make build-all

# Install to $GOPATH/bin (makes it available system-wide)
make install-agent

# Or install all binaries
make install-all
```

After installation, verify it works:

```bash
ailang-agent --version
# ailang-agent version 0.4.5-dev
```

## Quick Start

### 1. Start the Server

```bash
ailang serve
```

This opens [http://localhost:1956](http://localhost:1956) with:
- **Messages tab** - Send directives to agents, see results
- **Approvals tab** - Approve/reject capability requests

### 2. Start the Agent

In a separate terminal:

```bash
ailang-agent --instance-id my-agent --db ~/.ailang/state/collaboration.db
```

The agent:
- Polls for new messages every 2 seconds
- Executes directives using Claude Code
- Sends results back to the UI

### 3. Send a Directive

In the web UI:
1. Click **New Thread**
2. Type a directive: `Create a hello.go file that prints "Hello, World!"`
3. Click **Send**

Watch the agent execute and return results!

## Architecture

```
┌─────────────────────────────────────────────────────────────┐
│                   Browser (localhost:1956)                   │
│  ┌─────────────────┐  ┌─────────────────────────────────┐   │
│  │ Messages Tab    │  │ Approvals Tab                   │   │
│  │ - Send tasks    │  │ - Approve Net/FS capabilities   │   │
│  │ - See results   │  │ - Reject risky operations       │   │
│  └─────────────────┘  └─────────────────────────────────┘   │
└────────────────────────────┬────────────────────────────────┘
                             │ WebSocket + REST API
                             ▼
┌─────────────────────────────────────────────────────────────┐
│                    ailang serve                              │
│  - HTTP server on port 1956                                 │
│  - WebSocket for real-time updates                          │
│  - REST API: /api/threads, /api/messages, /api/approvals   │
└────────────────────────────┬────────────────────────────────┘
                             │
                             ▼
┌─────────────────────────────────────────────────────────────┐
│              ~/.ailang/state/collaboration.db               │
│  - threads: conversation threads                            │
│  - messages: directives and results                         │
│  - approvals: capability requests                           │
└────────────────────────────┬────────────────────────────────┘
                             │
                             ▼
┌─────────────────────────────────────────────────────────────┐
│                    ailang-agent                              │
│  - Polls for messages every 2 seconds                       │
│  - Detects required capabilities (IO, FS, Net)             │
│  - Requests approval for sensitive operations               │
│  - Executes via Claude Code headless mode                   │
│  - Reports results back to database                         │
└─────────────────────────────────────────────────────────────┘
```

## Approval Workflow

When a directive requires special capabilities (like network access), the agent:

1. **Detects capabilities** - Analyzes directive text for FS/Net/IO patterns
2. **Requests approval** - Creates an approval request in the database
3. **Waits for human** - The Approvals tab shows pending requests
4. **Proceeds or aborts** - After approval, executes; after rejection, cancels

Example:

```
User: "Fetch the API docs from https://api.example.com"

Agent: [Detected: Net capability required]
       [Created approval request]
       [Waiting 60s for approval...]

UI: Shows approval card with:
    - Requested capability: Net
    - Target URL: api.example.com
    - Estimated cost: $0.05
    - [Approve] [Reject] buttons

User: [Clicks Approve]

Agent: [Approval received]
       [Executing directive...]
       [Done! Results sent to UI]
```

## CLI Reference

### Server

```bash
# Start server (default port 1956)
ailang serve

# Custom port
ailang serve --port 8080

# Custom database
ailang serve --db /path/to/collab.db
```

### Agent

```bash
# Start agent
ailang-agent --instance-id my-agent --db ~/.ailang/state/collaboration.db

# Custom poll interval (seconds)
ailang-agent --instance-id my-agent --poll-interval 5

# Show version
ailang-agent --version
```

## Database Schema

The `collaboration.db` SQLite database contains:

```sql
-- Conversation threads
CREATE TABLE threads (
    id TEXT PRIMARY KEY,
    title TEXT,
    status TEXT,  -- 'active', 'completed', 'archived'
    created_by_type TEXT,
    created_by_id TEXT,
    created_at INTEGER,
    updated_at INTEGER
);

-- Messages (directives and results)
CREATE TABLE messages (
    id TEXT PRIMARY KEY,
    thread_id TEXT,
    message_seq INTEGER,
    from_type TEXT,   -- 'human', 'ailang_instance'
    from_id TEXT,
    to_type TEXT,
    to_id TEXT,
    kind TEXT,        -- 'directive', 'result', 'status', 'error'
    content TEXT,
    delivery_state TEXT,
    business_state TEXT,
    created_at INTEGER
);

-- Capability approval requests
CREATE TABLE approvals (
    id TEXT PRIMARY KEY,
    thread_id TEXT,
    status TEXT,      -- 'pending', 'approved', 'rejected'
    capability_token TEXT,
    proposal_text TEXT,
    impact_level TEXT,
    estimated_cost REAL,
    created_at INTEGER,
    reviewed_at INTEGER,
    reviewed_by TEXT,
    review_notes TEXT
);
```

## API Endpoints

### Threads

```bash
# List threads
curl http://localhost:1956/api/threads

# Create thread
curl -X POST http://localhost:1956/api/threads \
  -H "Content-Type: application/json" \
  -d '{"title": "Build a CLI tool"}'

# Get thread
curl http://localhost:1956/api/threads/{thread_id}
```

### Messages

```bash
# Get messages for thread
curl http://localhost:1956/api/messages?thread_id={thread_id}

# Send message
curl -X POST http://localhost:1956/api/messages \
  -H "Content-Type: application/json" \
  -d '{
    "thread_id": "...",
    "from_type": "human",
    "from_id": "user",
    "to_type": "ailang_instance",
    "to_id": "default",
    "kind": "directive",
    "content": "Create a hello.go file"
  }'
```

### Approvals

```bash
# List pending approvals
curl http://localhost:1956/api/approvals?status=pending

# Approve
curl -X POST http://localhost:1956/api/approvals/{id}/approve \
  -H "Content-Type: application/json" \
  -d '{"notes": "Looks safe"}'

# Reject
curl -X POST http://localhost:1956/api/approvals/{id}/reject \
  -H "Content-Type: application/json" \
  -d '{"notes": "Too risky"}'
```

### Health Check

```bash
curl http://localhost:1956/health
# {"status":"healthy","connections":1,"timestamp":1732888800}
```

## Troubleshooting

### Server won't start

```bash
# Check if port is in use
lsof -i :1956

# Use different port
ailang serve --port 8080
```

### Agent not receiving messages

```bash
# Verify database path matches
ailang serve --db ~/.ailang/state/collaboration.db
ailang-agent --db ~/.ailang/state/collaboration.db

# Check agent logs (runs in foreground)
ailang-agent --instance-id test
```

### Messages stuck as "pending"

The agent may have crashed. Restart it:

```bash
ailang-agent --instance-id my-agent --db ~/.ailang/state/collaboration.db
```

### Approval timeout

Default approval timeout is 60 seconds. If no human approves in time, the directive is cancelled.

## Current Limitations

:::warning Work in Progress
The Collaboration Hub is functional but has some limitations:
:::

1. **No background daemon** - Agent must run in foreground
2. **No project settings** - Agent executes in temp workspace, doesn't load `.claude/`
3. **No persistent config** - All settings via CLI flags
4. **No agent status UI** - Can't see if agent is running from the web UI

These are addressed in the planned [Background Agent Daemon](/docs/roadmap#v050) feature.

## See Also

- [Agent Messaging](/docs/guides/agent-messaging) - CLI-based messaging between projects
- [Evaluation Guide](/docs/guides/evaluation) - Running AI benchmarks
