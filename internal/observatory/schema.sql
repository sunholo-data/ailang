-- Observatory Schema v1
-- Unified observability platform for AILANG
-- Supports: Workspaces → Tasks → Agents → Spans → Events + Messages

-- Workspaces (git repos, projects)
CREATE TABLE IF NOT EXISTS workspaces (
    id TEXT PRIMARY KEY,
    name TEXT NOT NULL,
    path TEXT NOT NULL,
    git_remote TEXT,
    created_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP,
    updated_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP
);

-- Tasks (root entity - from GitHub, messages, email, etc.)
CREATE TABLE IF NOT EXISTS tasks (
    id TEXT PRIMARY KEY,
    workspace_id TEXT NOT NULL REFERENCES workspaces(id) ON DELETE CASCADE,
    parent_task_id TEXT,        -- Links to parent task for handoff chains (design → sprint → execute)
    title TEXT NOT NULL,
    description TEXT,
    source_type TEXT NOT NULL,  -- 'github_issue', 'message', 'email', 'manual'
    source_ref TEXT,            -- e.g., 'sunholo-data/ailang#123'
    status TEXT NOT NULL DEFAULT 'pending',  -- pending, running, completed, failed
    priority TEXT DEFAULT 'medium',
    created_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP,
    started_at TIMESTAMP,
    completed_at TIMESTAMP,

    -- Aggregated metrics (updated on span completion)
    total_duration_ms INTEGER DEFAULT 0,
    total_tokens_in INTEGER DEFAULT 0,
    total_tokens_out INTEGER DEFAULT 0,
    total_cost_usd REAL DEFAULT 0.0,
    agent_count INTEGER DEFAULT 0,
    span_count INTEGER DEFAULT 0,
    error_count INTEGER DEFAULT 0
);

-- Agent assignments (coordinator → agent delegation)
CREATE TABLE IF NOT EXISTS agent_assignments (
    id TEXT PRIMARY KEY,
    task_id TEXT NOT NULL REFERENCES tasks(id) ON DELETE CASCADE,
    agent_id TEXT NOT NULL,      -- 'design-doc-creator', 'sprint-planner', etc.
    provider TEXT NOT NULL,      -- 'claude', 'gemini', 'ollama'
    status TEXT NOT NULL DEFAULT 'pending',
    assigned_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP,
    started_at TIMESTAMP,
    completed_at TIMESTAMP,
    parent_assignment_id TEXT REFERENCES agent_assignments(id),  -- for delegation chains

    -- Agent-level aggregates
    duration_ms INTEGER DEFAULT 0,
    tokens_in INTEGER DEFAULT 0,
    tokens_out INTEGER DEFAULT 0,
    cost_usd REAL DEFAULT 0.0,
    tool_calls INTEGER DEFAULT 0,
    turns INTEGER DEFAULT 0
);

-- Spans (full OTEL spans with attributes)
CREATE TABLE IF NOT EXISTS spans (
    id TEXT PRIMARY KEY,           -- span_id
    trace_id TEXT NOT NULL,
    parent_span_id TEXT,
    task_id TEXT REFERENCES tasks(id) ON DELETE SET NULL,
    agent_assignment_id TEXT REFERENCES agent_assignments(id) ON DELETE SET NULL,

    name TEXT NOT NULL,            -- 'compile.pipeline', 'executor.claude.execute', etc.
    kind TEXT NOT NULL DEFAULT 'internal',  -- 'internal', 'client', 'server', 'producer', 'consumer'
    status TEXT NOT NULL DEFAULT 'unset',   -- 'ok', 'error', 'unset'
    status_message TEXT,

    start_time TIMESTAMP NOT NULL,
    end_time TIMESTAMP,
    duration_ms INTEGER,

    -- Normalized attributes (common across providers)
    tokens_in INTEGER,
    tokens_out INTEGER,
    cache_read_tokens INTEGER DEFAULT 0,
    cache_creation_tokens INTEGER DEFAULT 0,
    cost_usd REAL,
    model TEXT,
    provider TEXT,

    -- Full attributes as JSON (for provider-specific data)
    attributes TEXT NOT NULL DEFAULT '{}',

    -- Resource attributes (service info)
    resource_attributes TEXT NOT NULL DEFAULT '{}',

    created_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP
);

-- NOTE: span_events table removed in migration v4 (M-DB-CLEANUP)
-- span_events: designed for OTEL events but never implemented (0 rows)

-- Messages (observatory-specific messaging - note: overlaps with collaboration.db:inbox_messages)
-- TODO: Consider consolidating with collaboration.db in M-DB-UNIFY
CREATE TABLE IF NOT EXISTS messages (
    id TEXT PRIMARY KEY,
    task_id TEXT REFERENCES tasks(id) ON DELETE SET NULL,
    inbox TEXT NOT NULL,
    from_agent TEXT NOT NULL,
    title TEXT NOT NULL,
    content TEXT NOT NULL,
    message_type TEXT DEFAULT 'general',
    status TEXT NOT NULL DEFAULT 'unread',
    priority TEXT DEFAULT 'normal',

    -- GitHub sync
    github_issue_number INTEGER,
    github_repo TEXT,

    -- Correlation
    correlation_id TEXT,
    reply_to_id TEXT REFERENCES messages(id),

    created_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP,
    read_at TIMESTAMP,
    archived_at TIMESTAMP,

    -- Search
    content_hash TEXT,             -- for deduplication
    embedding BLOB                 -- for semantic search
);

-- Indexes for common queries
CREATE INDEX IF NOT EXISTS idx_tasks_workspace ON tasks(workspace_id);
CREATE INDEX IF NOT EXISTS idx_tasks_status ON tasks(status);
CREATE INDEX IF NOT EXISTS idx_tasks_created ON tasks(created_at DESC);
CREATE INDEX IF NOT EXISTS idx_tasks_source ON tasks(source_type, source_ref);
CREATE INDEX IF NOT EXISTS idx_tasks_parent ON tasks(parent_task_id);

CREATE INDEX IF NOT EXISTS idx_agent_assignments_task ON agent_assignments(task_id);
CREATE INDEX IF NOT EXISTS idx_agent_assignments_agent ON agent_assignments(agent_id);
CREATE INDEX IF NOT EXISTS idx_agent_assignments_status ON agent_assignments(status);
CREATE INDEX IF NOT EXISTS idx_agent_assignments_provider ON agent_assignments(provider);

CREATE INDEX IF NOT EXISTS idx_spans_trace ON spans(trace_id);
CREATE INDEX IF NOT EXISTS idx_spans_task ON spans(task_id);
CREATE INDEX IF NOT EXISTS idx_spans_agent ON spans(agent_assignment_id);
CREATE INDEX IF NOT EXISTS idx_spans_name ON spans(name);
CREATE INDEX IF NOT EXISTS idx_spans_time ON spans(start_time DESC);
CREATE INDEX IF NOT EXISTS idx_spans_parent ON spans(parent_span_id);

-- NOTE: idx_span_events_* indices removed in migration v4

CREATE INDEX IF NOT EXISTS idx_messages_task ON messages(task_id);
CREATE INDEX IF NOT EXISTS idx_messages_inbox ON messages(inbox);
CREATE INDEX IF NOT EXISTS idx_messages_status ON messages(status);
CREATE INDEX IF NOT EXISTS idx_messages_created ON messages(created_at DESC);
CREATE INDEX IF NOT EXISTS idx_messages_github ON messages(github_issue_number);

-- Claude Code session metadata (from hooks)
-- Links session.id to workspace for span enrichment
CREATE TABLE IF NOT EXISTS sessions (
    session_id TEXT PRIMARY KEY,
    workspace TEXT NOT NULL,          -- process.cwd from SessionStart hook
    claude_version TEXT,              -- Claude Code version
    source TEXT DEFAULT 'hook',       -- 'hook', 'otel', 'manual'
    started_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP,
    ended_at TIMESTAMP,
    turn_count INTEGER DEFAULT 0
);

CREATE INDEX IF NOT EXISTS idx_sessions_workspace ON sessions(workspace);
CREATE INDEX IF NOT EXISTS idx_sessions_started ON sessions(started_at DESC);

-- Tool call metadata (from PreToolUse/PostToolUse hooks)
-- Rich metadata for tool hierarchy
CREATE TABLE IF NOT EXISTS session_tools (
    tool_use_id TEXT PRIMARY KEY,
    session_id TEXT NOT NULL,
    tool_name TEXT NOT NULL,
    tool_input TEXT,                  -- JSON payload
    tool_response TEXT,               -- JSON response (truncated)
    start_time TIMESTAMP NOT NULL,
    end_time TIMESTAMP,
    success BOOLEAN,
    FOREIGN KEY (session_id) REFERENCES sessions(session_id) ON DELETE CASCADE
);

CREATE INDEX IF NOT EXISTS idx_session_tools_session ON session_tools(session_id);
CREATE INDEX IF NOT EXISTS idx_session_tools_name ON session_tools(tool_name);
CREATE INDEX IF NOT EXISTS idx_session_tools_time ON session_tools(start_time DESC);

-- OTLP metrics from Claude Code telemetry
-- Captures: lines_of_code.count, commit.count, pull_request.count, active_time.total, etc.
CREATE TABLE IF NOT EXISTS metrics (
    id INTEGER PRIMARY KEY AUTOINCREMENT,
    name TEXT NOT NULL,                -- e.g., 'claude_code.lines_of_code.count'
    metric_type TEXT NOT NULL,         -- 'counter', 'gauge'
    session_id TEXT,
    workspace TEXT,
    provider TEXT,
    -- Denormalized labels for efficient queries
    label_type TEXT,                   -- 'added'/'removed' for LOC
    label_tool TEXT,                   -- tool name for tool metrics
    label_decision TEXT,               -- 'approved'/'rejected' for code_edit_tool
    label_language TEXT,               -- language for LOC
    label_model TEXT,                  -- model name for model-specific metrics
    -- Values
    value_int INTEGER,
    value_float REAL,
    labels TEXT NOT NULL DEFAULT '{}',
    resource_attributes TEXT DEFAULT '{}',
    timestamp TIMESTAMP NOT NULL,
    created_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP
);

CREATE INDEX IF NOT EXISTS idx_metrics_name ON metrics(name);
CREATE INDEX IF NOT EXISTS idx_metrics_session ON metrics(session_id);
CREATE INDEX IF NOT EXISTS idx_metrics_timestamp ON metrics(timestamp DESC);

-- Aggregation views
CREATE VIEW IF NOT EXISTS workspace_stats AS
SELECT
    w.id,
    w.name,
    w.path,
    COUNT(DISTINCT t.id) as task_count,
    COALESCE(SUM(t.total_cost_usd), 0) as total_cost,
    COALESCE(SUM(t.total_tokens_in + t.total_tokens_out), 0) as total_tokens,
    CASE
        WHEN COUNT(DISTINCT t.id) > 0 THEN
            COUNT(DISTINCT CASE WHEN t.status = 'completed' THEN t.id END) * 100.0 / COUNT(DISTINCT t.id)
        ELSE 0
    END as success_rate,
    COUNT(DISTINCT aa.agent_id) as unique_agents,
    MAX(t.completed_at) as last_activity
FROM workspaces w
LEFT JOIN tasks t ON t.workspace_id = w.id
LEFT JOIN agent_assignments aa ON aa.task_id = t.id
GROUP BY w.id;

CREATE VIEW IF NOT EXISTS agent_stats AS
SELECT
    aa.agent_id,
    aa.provider,
    COUNT(*) as execution_count,
    COALESCE(SUM(aa.duration_ms), 0) as total_duration_ms,
    COALESCE(AVG(aa.duration_ms), 0) as avg_duration_ms,
    COALESCE(SUM(aa.tokens_in), 0) as total_tokens_in,
    COALESCE(SUM(aa.tokens_out), 0) as total_tokens_out,
    COALESCE(SUM(aa.cost_usd), 0) as total_cost,
    COALESCE(SUM(aa.tool_calls), 0) as total_tool_calls,
    CASE
        WHEN COUNT(*) > 0 THEN
            COUNT(CASE WHEN aa.status = 'completed' THEN 1 END) * 100.0 / COUNT(*)
        ELSE 0
    END as success_rate
FROM agent_assignments aa
GROUP BY aa.agent_id, aa.provider;

CREATE VIEW IF NOT EXISTS task_timeline AS
SELECT
    t.id as task_id,
    t.title,
    t.status,
    s.id as span_id,
    s.name as span_name,
    s.start_time,
    s.end_time,
    s.duration_ms,
    s.status as span_status,
    s.tokens_in,
    s.tokens_out,
    s.cost_usd,
    s.provider
FROM tasks t
LEFT JOIN spans s ON s.task_id = t.id
ORDER BY t.id, s.start_time;

CREATE VIEW IF NOT EXISTS provider_comparison AS
SELECT
    provider,
    COUNT(*) as total_executions,
    COALESCE(SUM(tokens_in), 0) as total_tokens_in,
    COALESCE(SUM(tokens_out), 0) as total_tokens_out,
    COALESCE(SUM(cost_usd), 0) as total_cost,
    COALESCE(AVG(duration_ms), 0) as avg_duration_ms,
    CASE
        WHEN COUNT(*) > 0 THEN
            COUNT(CASE WHEN status = 'completed' THEN 1 END) * 100.0 / COUNT(*)
        ELSE 0
    END as success_rate
FROM agent_assignments
WHERE provider IS NOT NULL
GROUP BY provider;
