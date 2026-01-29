-- Execution Chains Schema (Migration v7)
-- Unified hierarchy for: Issue -> Message -> Task -> Session -> Turns -> Tools -> Traces -> Approval -> Handoff
-- Replaces query-time correlation with write-time linking

-- Execution chains (top-level workflow from issue/message to completion)
CREATE TABLE IF NOT EXISTS execution_chains (
    id TEXT PRIMARY KEY,                    -- UUID

    -- Source (what triggered this chain)
    source_type TEXT NOT NULL,              -- 'github_issue', 'message', 'manual'
    source_ref TEXT,                        -- Issue number, message ID, etc.
    github_repo TEXT,                       -- e.g., 'sunholo-data/ailang'
    github_issue_number INTEGER,

    -- Current state
    status TEXT NOT NULL DEFAULT 'active',  -- 'active', 'pending_approval', 'completed', 'failed'
    current_stage INTEGER DEFAULT 0,        -- Which stage we're on

    -- Workspace context
    workspace_id TEXT,                      -- Links to workspaces table
    workspace_path TEXT,                    -- Git repo path for context

    -- Timestamps
    created_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP,
    updated_at TIMESTAMP,
    completed_at TIMESTAMP,

    -- Summary metrics (denormalized for quick queries)
    total_cost REAL DEFAULT 0,
    total_tokens INTEGER DEFAULT 0,
    total_turns INTEGER DEFAULT 0,
    stages_completed INTEGER DEFAULT 0
);

-- Chain stages (each agent execution within a chain)
CREATE TABLE IF NOT EXISTS chain_stages (
    id TEXT PRIMARY KEY,                    -- UUID
    chain_id TEXT NOT NULL REFERENCES execution_chains(id) ON DELETE CASCADE,
    stage_number INTEGER NOT NULL,

    -- Agent info
    agent_id TEXT NOT NULL,                 -- 'design-doc-creator', 'sprint-planner', etc.
    provider TEXT,                          -- 'claude', 'gemini', etc.

    -- Links to existing tables (foreign keys where possible, IDs for cross-DB)
    message_id TEXT,                        -- -> collaboration.db:inbox_messages.id (cross-DB)
    task_id TEXT,                           -- -> coordinator.db:tasks.id (cross-DB)
    session_id TEXT,                        -- -> sessions.session_id (same DB)

    -- State
    status TEXT NOT NULL DEFAULT 'pending', -- 'pending', 'running', 'awaiting_approval', 'completed', 'failed'
    approval_status TEXT,                   -- 'pending', 'approved', 'rejected', NULL
    approval_type TEXT,                     -- 'merge', 'handoff', 'merge_handoff'

    -- Handoff info
    handoff_to TEXT,                        -- Next agent ID (if any)
    iteration INTEGER DEFAULT 1,            -- Retry count (1 = first attempt)
    human_feedback TEXT,                    -- Feedback text if rejected

    -- Timestamps
    started_at TIMESTAMP,
    completed_at TIMESTAMP,

    -- Summary (denormalized from spans)
    cost REAL DEFAULT 0,
    tokens_in INTEGER DEFAULT 0,
    tokens_out INTEGER DEFAULT 0,
    turns INTEGER DEFAULT 0,
    tool_calls INTEGER DEFAULT 0,
    duration_ms INTEGER DEFAULT 0,

    -- Error tracking
    error_message TEXT,
    error_count INTEGER DEFAULT 0,

    UNIQUE(chain_id, stage_number)
);

-- Indexes for common queries
CREATE INDEX IF NOT EXISTS idx_execution_chains_status ON execution_chains(status);
CREATE INDEX IF NOT EXISTS idx_execution_chains_source ON execution_chains(source_type, source_ref);
CREATE INDEX IF NOT EXISTS idx_execution_chains_github ON execution_chains(github_repo, github_issue_number);
CREATE INDEX IF NOT EXISTS idx_execution_chains_workspace ON execution_chains(workspace_id);
CREATE INDEX IF NOT EXISTS idx_execution_chains_created ON execution_chains(created_at DESC);

CREATE INDEX IF NOT EXISTS idx_chain_stages_chain ON chain_stages(chain_id, stage_number);
CREATE INDEX IF NOT EXISTS idx_chain_stages_agent ON chain_stages(agent_id);
CREATE INDEX IF NOT EXISTS idx_chain_stages_task ON chain_stages(task_id);
CREATE INDEX IF NOT EXISTS idx_chain_stages_session ON chain_stages(session_id);
CREATE INDEX IF NOT EXISTS idx_chain_stages_status ON chain_stages(status);
CREATE INDEX IF NOT EXISTS idx_chain_stages_approval ON chain_stages(approval_status);

-- Add chain_id and stage_id columns to spans table (for write-time linking)
-- These are populated when AILANG_CHAIN_ID and AILANG_STAGE_ID env vars are present
-- ALTER TABLE spans ADD COLUMN chain_id TEXT REFERENCES execution_chains(id) ON DELETE SET NULL;
-- ALTER TABLE spans ADD COLUMN stage_id TEXT REFERENCES chain_stages(id) ON DELETE SET NULL;

-- View: Chain summary with all stages
CREATE VIEW IF NOT EXISTS chain_summary AS
SELECT
    c.id,
    c.source_type,
    c.source_ref,
    c.github_repo,
    c.github_issue_number,
    c.status,
    c.current_stage,
    c.total_cost,
    c.total_tokens,
    c.total_turns,
    c.stages_completed,
    c.created_at,
    c.completed_at,
    COUNT(s.id) as stage_count,
    MAX(s.stage_number) as max_stage,
    GROUP_CONCAT(s.agent_id, ' -> ') as agent_flow
FROM execution_chains c
LEFT JOIN chain_stages s ON s.chain_id = c.id
GROUP BY c.id
ORDER BY c.created_at DESC;

-- View: Active chains with pending approvals
CREATE VIEW IF NOT EXISTS pending_approvals_view AS
SELECT
    c.id as chain_id,
    c.source_type,
    c.source_ref,
    s.id as stage_id,
    s.stage_number,
    s.agent_id,
    s.approval_status,
    s.approval_type,
    s.task_id,
    s.session_id,
    s.cost,
    s.turns,
    s.created_at as stage_created
FROM execution_chains c
JOIN chain_stages s ON s.chain_id = c.id
WHERE s.approval_status = 'pending'
ORDER BY s.created_at DESC;
