#!/usr/bin/env bash
# Diagnose Collaboration Hub and Observatory issues

set -euo pipefail

DB="${AILANG_OBSERVATORY_DB:-$HOME/.ailang/state/observatory.db}"
SERVER_URL="${AILANG_SERVER_URL:-http://localhost:1957}"

usage() {
    cat <<EOF
Usage: $0 <command> [args]

Commands:
  health          Check server and database health
  task <id>       Diagnose task by ID
  spans <task_id> Show spans for a task
  orphans         Find orphan spans (no task_id)
  costs           Show cost breakdown by model
  recent          Show recent activity

Examples:
  $0 health
  $0 task task-50b9518d
  $0 spans task-50b9518d
  $0 costs
EOF
    exit 1
}

check_db() {
    if [[ ! -f "$DB" ]]; then
        echo "✗ Database not found: $DB"
        exit 1
    fi
}

cmd_health() {
    echo "=== Server Health ==="
    if curl -sf "$SERVER_URL/health" | jq -r '.status' 2>/dev/null; then
        echo "✓ Server is running"
    else
        echo "✗ Server not responding at $SERVER_URL"
    fi

    echo ""
    echo "=== Database Health ==="
    check_db
    echo "✓ Database found: $DB"

    sqlite3 "$DB" "SELECT
        'workspaces: ' || (SELECT COUNT(*) FROM workspaces),
        'tasks: ' || (SELECT COUNT(*) FROM tasks),
        'spans: ' || (SELECT COUNT(*) FROM spans),
        'agents: ' || (SELECT COUNT(*) FROM agent_assignments)" | tr '|' '\n'

    echo ""
    echo "=== Recent Activity ==="
    sqlite3 "$DB" "SELECT provider, COUNT(*) as count,
        printf('$%.4f', COALESCE(SUM(cost_usd), 0)) as cost
        FROM spans WHERE provider IS NOT NULL
        GROUP BY provider ORDER BY count DESC LIMIT 5"
}

cmd_task() {
    local task_id="${1:-}"
    if [[ -z "$task_id" ]]; then
        echo "Error: Task ID required"
        usage
    fi

    check_db

    echo "=== Task: $task_id ==="
    sqlite3 -header "$DB" "SELECT id, title, status, source_type,
        span_count, printf('$%.4f', COALESCE(total_cost_usd, 0)) as cost
        FROM tasks WHERE id = '$task_id'"

    echo ""
    echo "=== Spans by task_id ==="
    sqlite3 "$DB" "SELECT name, trace_id FROM spans WHERE task_id = '$task_id' LIMIT 10"

    echo ""
    echo "=== Trace IDs ==="
    sqlite3 "$DB" "SELECT DISTINCT trace_id FROM spans WHERE task_id = '$task_id'"

    echo ""
    echo "=== API Check ==="
    curl -sf "$SERVER_URL/api/observatory/tasks/$task_id/hierarchy" | jq '{
        task: .task.id,
        agent_count: (.agents | length),
        total_spans: [.agents[].traces[].spans | length] | add
    }' 2>/dev/null || echo "Failed to fetch hierarchy"
}

cmd_spans() {
    local task_id="${1:-}"
    if [[ -z "$task_id" ]]; then
        echo "Error: Task ID required"
        usage
    fi

    check_db

    echo "=== All Spans in Task Traces ==="
    sqlite3 -header "$DB" "SELECT name, task_id, parent_span_id
        FROM spans WHERE trace_id IN (
            SELECT DISTINCT trace_id FROM spans WHERE task_id = '$task_id'
        ) ORDER BY start_time LIMIT 50"
}

cmd_orphans() {
    check_db

    echo "=== Orphan Spans (no task_id) ==="
    sqlite3 "$DB" "SELECT COUNT(*) as orphan_count FROM spans WHERE task_id IS NULL"

    echo ""
    echo "=== Sample Orphans ==="
    sqlite3 -header "$DB" "SELECT name, trace_id,
        datetime(start_time) as started
        FROM spans WHERE task_id IS NULL
        ORDER BY start_time DESC LIMIT 10"
}

cmd_costs() {
    check_db

    echo "=== Cost by Model ==="
    sqlite3 -header "$DB" "SELECT model,
        COUNT(*) as spans,
        SUM(tokens_in) as tokens_in,
        SUM(tokens_out) as tokens_out,
        printf('$%.4f', COALESCE(SUM(cost_usd), 0)) as cost
        FROM spans WHERE model IS NOT NULL
        GROUP BY model ORDER BY SUM(cost_usd) DESC"

    echo ""
    echo "=== Cost by Day (last 7 days) ==="
    sqlite3 -header "$DB" "SELECT DATE(start_time) as day,
        printf('$%.4f', COALESCE(SUM(cost_usd), 0)) as cost
        FROM spans
        GROUP BY DATE(start_time)
        ORDER BY day DESC LIMIT 7"
}

cmd_recent() {
    check_db

    echo "=== Recent Tasks ==="
    sqlite3 -header "$DB" "SELECT id, title, status,
        datetime(created_at) as created
        FROM tasks ORDER BY created_at DESC LIMIT 10"

    echo ""
    echo "=== Recent Spans ==="
    sqlite3 -header "$DB" "SELECT name, provider, model,
        printf('$%.4f', COALESCE(cost_usd, 0)) as cost
        FROM spans ORDER BY start_time DESC LIMIT 10"
}

# Main
case "${1:-}" in
    health)  cmd_health ;;
    task)    cmd_task "${2:-}" ;;
    spans)   cmd_spans "${2:-}" ;;
    orphans) cmd_orphans ;;
    costs)   cmd_costs ;;
    recent)  cmd_recent ;;
    *)       usage ;;
esac
