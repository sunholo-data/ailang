// Package observatory provides a unified observability platform for AILANG.
package observatory

import (
	"database/sql"
	"embed"
	"fmt"
)

//go:embed schema.sql schema_chains.sql
var schemaFS embed.FS

// Migrate runs database migrations to create or update the observatory schema.
// It is idempotent - safe to call multiple times.
func Migrate(db *sql.DB) error {
	// Run base schema
	schema, err := schemaFS.ReadFile("schema.sql")
	if err != nil {
		return fmt.Errorf("failed to read embedded schema: %w", err)
	}

	_, err = db.Exec(string(schema))
	if err != nil {
		return fmt.Errorf("failed to execute schema: %w", err)
	}

	// Run chains schema (v7+)
	chainsSchema, err := schemaFS.ReadFile("schema_chains.sql")
	if err != nil {
		return fmt.Errorf("failed to read chains schema: %w", err)
	}

	_, err = db.Exec(string(chainsSchema))
	if err != nil {
		return fmt.Errorf("failed to execute chains schema: %w", err)
	}

	// Create trace_summaries table (v12+, M-PERF-OBSERVATORY)
	_, err = db.Exec(`
		CREATE TABLE IF NOT EXISTS trace_summaries (
			trace_id TEXT PRIMARY KEY,
			root_span_name TEXT,
			root_span_status TEXT,
			span_count INTEGER DEFAULT 0,
			total_duration_ms INTEGER DEFAULT 0,
			start_time TIMESTAMP,
			end_time TIMESTAMP,
			task_id TEXT,
			service_name TEXT,
			workspace TEXT DEFAULT '',
			updated_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP
		)
	`)
	if err != nil {
		return fmt.Errorf("failed to create trace_summaries table: %w", err)
	}

	return nil
}

// MigrateWithVersion runs migrations and tracks schema version.
// Returns the current schema version after migration.
func MigrateWithVersion(db *sql.DB) (int, error) {
	// First ensure the schema_version table exists
	_, err := db.Exec(`
		CREATE TABLE IF NOT EXISTS schema_version (
			version INTEGER PRIMARY KEY,
			applied_at DATETIME DEFAULT CURRENT_TIMESTAMP
		)
	`)
	if err != nil {
		return 0, fmt.Errorf("failed to create schema_version table: %w", err)
	}

	// Get current version
	var currentVersion int
	err = db.QueryRow("SELECT COALESCE(MAX(version), 0) FROM schema_version").Scan(&currentVersion)
	if err != nil {
		return 0, fmt.Errorf("failed to get current version: %w", err)
	}

	// Run base migration if not applied
	if currentVersion < 1 {
		if err := Migrate(db); err != nil {
			return currentVersion, err
		}

		_, err = db.Exec("INSERT INTO schema_version (version) VALUES (1)")
		if err != nil {
			return currentVersion, fmt.Errorf("failed to record version: %w", err)
		}
		currentVersion = 1
	}

	// Migration v2: Add parent_task_id for task hierarchy tracking
	if currentVersion < 2 {
		// Check if column already exists (idempotent)
		var colExists int
		err := db.QueryRow(`
			SELECT COUNT(*) FROM pragma_table_info('tasks') WHERE name = 'parent_task_id'
		`).Scan(&colExists)
		if err != nil {
			return currentVersion, fmt.Errorf("failed to check parent_task_id column: %w", err)
		}

		if colExists == 0 {
			_, err = db.Exec("ALTER TABLE tasks ADD COLUMN parent_task_id TEXT")
			if err != nil {
				return currentVersion, fmt.Errorf("failed to add parent_task_id column: %w", err)
			}
		}

		// Create index (idempotent via IF NOT EXISTS)
		_, err = db.Exec("CREATE INDEX IF NOT EXISTS idx_tasks_parent ON tasks(parent_task_id)")
		if err != nil {
			return currentVersion, fmt.Errorf("failed to create parent_task_id index: %w", err)
		}

		_, err = db.Exec("INSERT INTO schema_version (version) VALUES (2)")
		if err != nil {
			return currentVersion, fmt.Errorf("failed to record version 2: %w", err)
		}
		currentVersion = 2
	}

	// Migration v3: Add sessions and session_tools tables for Claude Code hook integration
	if currentVersion < 3 {
		// Check if sessions table already exists (idempotent)
		var sessionsExists int
		err := db.QueryRow(`
			SELECT COUNT(*) FROM sqlite_master WHERE type='table' AND name='sessions'
		`).Scan(&sessionsExists)
		if err != nil {
			return currentVersion, fmt.Errorf("failed to check sessions table: %w", err)
		}

		if sessionsExists == 0 {
			_, err = db.Exec(`
				CREATE TABLE IF NOT EXISTS sessions (
					session_id TEXT PRIMARY KEY,
					workspace TEXT NOT NULL,
					claude_version TEXT,
					source TEXT DEFAULT 'hook',
					started_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP,
					ended_at TIMESTAMP,
					turn_count INTEGER DEFAULT 0
				)
			`)
			if err != nil {
				return currentVersion, fmt.Errorf("failed to create sessions table: %w", err)
			}

			_, err = db.Exec(`CREATE INDEX IF NOT EXISTS idx_sessions_workspace ON sessions(workspace)`)
			if err != nil {
				return currentVersion, fmt.Errorf("failed to create sessions workspace index: %w", err)
			}

			_, err = db.Exec(`CREATE INDEX IF NOT EXISTS idx_sessions_started ON sessions(started_at DESC)`)
			if err != nil {
				return currentVersion, fmt.Errorf("failed to create sessions started index: %w", err)
			}
		}

		// Check if session_tools table already exists
		var toolsExists int
		err = db.QueryRow(`
			SELECT COUNT(*) FROM sqlite_master WHERE type='table' AND name='session_tools'
		`).Scan(&toolsExists)
		if err != nil {
			return currentVersion, fmt.Errorf("failed to check session_tools table: %w", err)
		}

		if toolsExists == 0 {
			_, err = db.Exec(`
				CREATE TABLE IF NOT EXISTS session_tools (
					tool_use_id TEXT PRIMARY KEY,
					session_id TEXT NOT NULL,
					tool_name TEXT NOT NULL,
					tool_input TEXT,
					tool_response TEXT,
					start_time TIMESTAMP NOT NULL,
					end_time TIMESTAMP,
					success BOOLEAN,
					FOREIGN KEY (session_id) REFERENCES sessions(session_id) ON DELETE CASCADE
				)
			`)
			if err != nil {
				return currentVersion, fmt.Errorf("failed to create session_tools table: %w", err)
			}

			_, err = db.Exec(`CREATE INDEX IF NOT EXISTS idx_session_tools_session ON session_tools(session_id)`)
			if err != nil {
				return currentVersion, fmt.Errorf("failed to create session_tools session index: %w", err)
			}

			_, err = db.Exec(`CREATE INDEX IF NOT EXISTS idx_session_tools_name ON session_tools(tool_name)`)
			if err != nil {
				return currentVersion, fmt.Errorf("failed to create session_tools name index: %w", err)
			}

			_, err = db.Exec(`CREATE INDEX IF NOT EXISTS idx_session_tools_time ON session_tools(start_time DESC)`)
			if err != nil {
				return currentVersion, fmt.Errorf("failed to create session_tools time index: %w", err)
			}
		}

		_, err = db.Exec("INSERT INTO schema_version (version) VALUES (3)")
		if err != nil {
			return currentVersion, fmt.Errorf("failed to record version 3: %w", err)
		}
		currentVersion = 3
	}

	// Migration v4: Remove unused span_events table (M-DB-CLEANUP)
	// - span_events: designed for OTEL events but never implemented (0 rows)
	// NOTE: messages table kept - has API endpoints in observatory/api.go
	if currentVersion < 4 {
		// Drop unused span_events table (IF EXISTS for idempotency)
		_, err = db.Exec("DROP TABLE IF EXISTS span_events")
		if err != nil {
			return currentVersion, fmt.Errorf("failed to drop span_events table: %w", err)
		}

		// Drop associated indices
		_, _ = db.Exec("DROP INDEX IF EXISTS idx_span_events_span")
		_, _ = db.Exec("DROP INDEX IF EXISTS idx_span_events_type")
		_, _ = db.Exec("DROP INDEX IF EXISTS idx_span_events_time")

		_, err = db.Exec("INSERT INTO schema_version (version) VALUES (4)")
		if err != nil {
			return currentVersion, fmt.Errorf("failed to record version 4: %w", err)
		}
		currentVersion = 4
	}

	// Migration v5: Add metrics table and cache token columns to spans
	// Captures Claude Code telemetry: LOC, commits, PRs, active time, cache tokens
	if currentVersion < 5 {
		// Create metrics table for OTLP counter/gauge metrics
		_, err = db.Exec(`
			CREATE TABLE IF NOT EXISTS metrics (
				id INTEGER PRIMARY KEY AUTOINCREMENT,
				name TEXT NOT NULL,
				metric_type TEXT NOT NULL,
				session_id TEXT,
				workspace TEXT,
				provider TEXT,
				label_type TEXT,
				label_tool TEXT,
				label_decision TEXT,
				label_language TEXT,
				label_model TEXT,
				value_int INTEGER,
				value_float REAL,
				labels TEXT NOT NULL DEFAULT '{}',
				resource_attributes TEXT DEFAULT '{}',
				timestamp TIMESTAMP NOT NULL,
				created_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP
			)
		`)
		if err != nil {
			return currentVersion, fmt.Errorf("failed to create metrics table: %w", err)
		}

		// Create indices for metrics table
		_, err = db.Exec(`CREATE INDEX IF NOT EXISTS idx_metrics_name ON metrics(name)`)
		if err != nil {
			return currentVersion, fmt.Errorf("failed to create metrics name index: %w", err)
		}

		_, err = db.Exec(`CREATE INDEX IF NOT EXISTS idx_metrics_session ON metrics(session_id)`)
		if err != nil {
			return currentVersion, fmt.Errorf("failed to create metrics session index: %w", err)
		}

		_, err = db.Exec(`CREATE INDEX IF NOT EXISTS idx_metrics_timestamp ON metrics(timestamp DESC)`)
		if err != nil {
			return currentVersion, fmt.Errorf("failed to create metrics timestamp index: %w", err)
		}

		// Add cache token columns to spans table (idempotent via checking if column exists)
		var cacheReadExists int
		err = db.QueryRow(`
			SELECT COUNT(*) FROM pragma_table_info('spans') WHERE name = 'cache_read_tokens'
		`).Scan(&cacheReadExists)
		if err != nil {
			return currentVersion, fmt.Errorf("failed to check cache_read_tokens column: %w", err)
		}

		if cacheReadExists == 0 {
			_, err = db.Exec("ALTER TABLE spans ADD COLUMN cache_read_tokens INTEGER DEFAULT 0")
			if err != nil {
				return currentVersion, fmt.Errorf("failed to add cache_read_tokens column: %w", err)
			}
		}

		var cacheCreationExists int
		err = db.QueryRow(`
			SELECT COUNT(*) FROM pragma_table_info('spans') WHERE name = 'cache_creation_tokens'
		`).Scan(&cacheCreationExists)
		if err != nil {
			return currentVersion, fmt.Errorf("failed to check cache_creation_tokens column: %w", err)
		}

		if cacheCreationExists == 0 {
			_, err = db.Exec("ALTER TABLE spans ADD COLUMN cache_creation_tokens INTEGER DEFAULT 0")
			if err != nil {
				return currentVersion, fmt.Errorf("failed to add cache_creation_tokens column: %w", err)
			}
		}

		_, err = db.Exec("INSERT INTO schema_version (version) VALUES (5)")
		if err != nil {
			return currentVersion, fmt.Errorf("failed to record version 5: %w", err)
		}
		currentVersion = 5
	}

	// Migration v6: Add chat_messages table for storing Claude Code conversation history
	// Imports JSONL files from ~/.claude/projects/ for unified queries with spans
	if currentVersion < 6 {
		// Create chat_messages table
		_, err = db.Exec(`
			CREATE TABLE IF NOT EXISTS chat_messages (
				id TEXT PRIMARY KEY,
				session_id TEXT NOT NULL,
				turn_number INTEGER NOT NULL,
				role TEXT NOT NULL CHECK (role IN ('user', 'assistant')),
				content_text TEXT,
				content_thinking TEXT,
				content_json TEXT,
				tokens_in INTEGER DEFAULT 0,
				tokens_out INTEGER DEFAULT 0,
				model TEXT,
				request_id TEXT,
				timestamp TIMESTAMP NOT NULL,
				created_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP
			)
		`)
		if err != nil {
			return currentVersion, fmt.Errorf("failed to create chat_messages table: %w", err)
		}

		// Create indexes for chat_messages
		_, err = db.Exec(`CREATE INDEX IF NOT EXISTS idx_chat_messages_session ON chat_messages(session_id, turn_number)`)
		if err != nil {
			return currentVersion, fmt.Errorf("failed to create chat_messages session index: %w", err)
		}

		_, err = db.Exec(`CREATE INDEX IF NOT EXISTS idx_chat_messages_timestamp ON chat_messages(timestamp DESC)`)
		if err != nil {
			return currentVersion, fmt.Errorf("failed to create chat_messages timestamp index: %w", err)
		}

		_, err = db.Exec(`CREATE INDEX IF NOT EXISTS idx_chat_messages_request ON chat_messages(request_id)`)
		if err != nil {
			return currentVersion, fmt.Errorf("failed to create chat_messages request index: %w", err)
		}

		// Create chat_import_status table to track which JSONL files have been imported
		_, err = db.Exec(`
			CREATE TABLE IF NOT EXISTS chat_import_status (
				session_id TEXT PRIMARY KEY,
				file_path TEXT NOT NULL,
				file_mtime TIMESTAMP NOT NULL,
				message_count INTEGER NOT NULL,
				imported_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP
			)
		`)
		if err != nil {
			return currentVersion, fmt.Errorf("failed to create chat_import_status table: %w", err)
		}

		_, err = db.Exec("INSERT INTO schema_version (version) VALUES (6)")
		if err != nil {
			return currentVersion, fmt.Errorf("failed to record version 6: %w", err)
		}
		currentVersion = 6
	}

	// Migration v7: Add execution_chains and chain_stages tables (M-CHAINS-SIMPLIFY)
	// Unified hierarchy: Issue -> Message -> Task -> Session -> Turns -> Tools -> Traces
	// Replaces query-time correlation in hierarchy.go with write-time linking
	if currentVersion < 7 {
		// Read the chains schema file
		chainsSchema, err := schemaFS.ReadFile("schema_chains.sql")
		if err != nil {
			return currentVersion, fmt.Errorf("failed to read chains schema: %w", err)
		}

		// Execute the chains schema (creates tables, indexes, views)
		_, err = db.Exec(string(chainsSchema))
		if err != nil {
			return currentVersion, fmt.Errorf("failed to execute chains schema: %w", err)
		}

		// Add chain_id and stage_id columns to spans table
		// Check if chain_id column already exists (idempotent)
		var chainIDExists int
		err = db.QueryRow(`
			SELECT COUNT(*) FROM pragma_table_info('spans') WHERE name = 'chain_id'
		`).Scan(&chainIDExists)
		if err != nil {
			return currentVersion, fmt.Errorf("failed to check chain_id column: %w", err)
		}

		if chainIDExists == 0 {
			_, err = db.Exec("ALTER TABLE spans ADD COLUMN chain_id TEXT")
			if err != nil {
				return currentVersion, fmt.Errorf("failed to add chain_id column: %w", err)
			}
		}

		var stageIDExists int
		err = db.QueryRow(`
			SELECT COUNT(*) FROM pragma_table_info('spans') WHERE name = 'stage_id'
		`).Scan(&stageIDExists)
		if err != nil {
			return currentVersion, fmt.Errorf("failed to check stage_id column: %w", err)
		}

		if stageIDExists == 0 {
			_, err = db.Exec("ALTER TABLE spans ADD COLUMN stage_id TEXT")
			if err != nil {
				return currentVersion, fmt.Errorf("failed to add stage_id column: %w", err)
			}
		}

		// Create indexes for chain linking on spans
		_, err = db.Exec("CREATE INDEX IF NOT EXISTS idx_spans_chain ON spans(chain_id)")
		if err != nil {
			return currentVersion, fmt.Errorf("failed to create chain_id index: %w", err)
		}

		_, err = db.Exec("CREATE INDEX IF NOT EXISTS idx_spans_stage ON spans(stage_id)")
		if err != nil {
			return currentVersion, fmt.Errorf("failed to create stage_id index: %w", err)
		}

		_, err = db.Exec("INSERT INTO schema_version (version) VALUES (7)")
		if err != nil {
			return currentVersion, fmt.Errorf("failed to record version 7: %w", err)
		}
		currentVersion = 7
	}

	// Migration v8: Add correlation columns to sessions table (M-DETERMINISTIC-CHAT-LINKING)
	// Enables deterministic linking from Claude Code hooks via env vars
	if currentVersion < 8 {
		// Add task_id column
		var taskIDExists int
		err := db.QueryRow(`
			SELECT COUNT(*) FROM pragma_table_info('sessions') WHERE name = 'task_id'
		`).Scan(&taskIDExists)
		if err != nil {
			return currentVersion, fmt.Errorf("failed to check task_id column: %w", err)
		}
		if taskIDExists == 0 {
			_, err = db.Exec("ALTER TABLE sessions ADD COLUMN task_id TEXT")
			if err != nil {
				return currentVersion, fmt.Errorf("failed to add task_id column: %w", err)
			}
		}

		// Add chain_id column
		var chainIDExists int
		err = db.QueryRow(`
			SELECT COUNT(*) FROM pragma_table_info('sessions') WHERE name = 'chain_id'
		`).Scan(&chainIDExists)
		if err != nil {
			return currentVersion, fmt.Errorf("failed to check chain_id column: %w", err)
		}
		if chainIDExists == 0 {
			_, err = db.Exec("ALTER TABLE sessions ADD COLUMN chain_id TEXT")
			if err != nil {
				return currentVersion, fmt.Errorf("failed to add chain_id column: %w", err)
			}
		}

		// Add stage_id column
		var stageIDExists int
		err = db.QueryRow(`
			SELECT COUNT(*) FROM pragma_table_info('sessions') WHERE name = 'stage_id'
		`).Scan(&stageIDExists)
		if err != nil {
			return currentVersion, fmt.Errorf("failed to check stage_id column: %w", err)
		}
		if stageIDExists == 0 {
			_, err = db.Exec("ALTER TABLE sessions ADD COLUMN stage_id TEXT")
			if err != nil {
				return currentVersion, fmt.Errorf("failed to add stage_id column: %w", err)
			}
		}

		// Add message_id column
		var messageIDExists int
		err = db.QueryRow(`
			SELECT COUNT(*) FROM pragma_table_info('sessions') WHERE name = 'message_id'
		`).Scan(&messageIDExists)
		if err != nil {
			return currentVersion, fmt.Errorf("failed to check message_id column: %w", err)
		}
		if messageIDExists == 0 {
			_, err = db.Exec("ALTER TABLE sessions ADD COLUMN message_id TEXT")
			if err != nil {
				return currentVersion, fmt.Errorf("failed to add message_id column: %w", err)
			}
		}

		// Create indexes for correlation lookups
		_, err = db.Exec("CREATE INDEX IF NOT EXISTS idx_sessions_task ON sessions(task_id)")
		if err != nil {
			return currentVersion, fmt.Errorf("failed to create task_id index: %w", err)
		}

		_, err = db.Exec("CREATE INDEX IF NOT EXISTS idx_sessions_chain ON sessions(chain_id)")
		if err != nil {
			return currentVersion, fmt.Errorf("failed to create chain_id index: %w", err)
		}

		_, err = db.Exec("INSERT INTO schema_version (version) VALUES (8)")
		if err != nil {
			return currentVersion, fmt.Errorf("failed to record version 8: %w", err)
		}
		currentVersion = 8
	}

	// Migration v9: Add correlation columns to chat_messages table (M-DETERMINISTIC-CHAT-LINKING Phase 5)
	// Enables direct task->message queries without timestamp filtering
	if currentVersion < 9 {
		// Add task_id column
		var taskIDExists int
		err := db.QueryRow(`
			SELECT COUNT(*) FROM pragma_table_info('chat_messages') WHERE name = 'task_id'
		`).Scan(&taskIDExists)
		if err != nil {
			return currentVersion, fmt.Errorf("failed to check chat_messages task_id column: %w", err)
		}
		if taskIDExists == 0 {
			_, err = db.Exec("ALTER TABLE chat_messages ADD COLUMN task_id TEXT")
			if err != nil {
				return currentVersion, fmt.Errorf("failed to add task_id to chat_messages: %w", err)
			}
		}

		// Add chain_id column
		var chainIDExists int
		err = db.QueryRow(`
			SELECT COUNT(*) FROM pragma_table_info('chat_messages') WHERE name = 'chain_id'
		`).Scan(&chainIDExists)
		if err != nil {
			return currentVersion, fmt.Errorf("failed to check chat_messages chain_id column: %w", err)
		}
		if chainIDExists == 0 {
			_, err = db.Exec("ALTER TABLE chat_messages ADD COLUMN chain_id TEXT")
			if err != nil {
				return currentVersion, fmt.Errorf("failed to add chain_id to chat_messages: %w", err)
			}
		}

		// Add stage_id column
		var stageIDExists int
		err = db.QueryRow(`
			SELECT COUNT(*) FROM pragma_table_info('chat_messages') WHERE name = 'stage_id'
		`).Scan(&stageIDExists)
		if err != nil {
			return currentVersion, fmt.Errorf("failed to check chat_messages stage_id column: %w", err)
		}
		if stageIDExists == 0 {
			_, err = db.Exec("ALTER TABLE chat_messages ADD COLUMN stage_id TEXT")
			if err != nil {
				return currentVersion, fmt.Errorf("failed to add stage_id to chat_messages: %w", err)
			}
		}

		// Create index for task_id lookup (most common query)
		_, err = db.Exec("CREATE INDEX IF NOT EXISTS idx_chat_messages_task ON chat_messages(task_id)")
		if err != nil {
			return currentVersion, fmt.Errorf("failed to create chat_messages task_id index: %w", err)
		}

		_, err = db.Exec("INSERT INTO schema_version (version) VALUES (9)")
		if err != nil {
			return currentVersion, fmt.Errorf("failed to record version 9: %w", err)
		}
		currentVersion = 9
	}

	// Migration v10: Add eval_assessment column to chain_stages (M-EVAL-CHAINS)
	// Stores structured eval assessment data as JSON for agent benchmark evaluations
	if currentVersion < 10 {
		var evalAssessmentExists int
		err := db.QueryRow(`
			SELECT COUNT(*) FROM pragma_table_info('chain_stages') WHERE name = 'eval_assessment'
		`).Scan(&evalAssessmentExists)
		if err != nil {
			return currentVersion, fmt.Errorf("failed to check eval_assessment column: %w", err)
		}
		if evalAssessmentExists == 0 {
			_, err = db.Exec("ALTER TABLE chain_stages ADD COLUMN eval_assessment TEXT")
			if err != nil {
				return currentVersion, fmt.Errorf("failed to add eval_assessment column: %w", err)
			}
		}

		// Index for eval-specific queries (chains with eval data)
		_, err = db.Exec("CREATE INDEX IF NOT EXISTS idx_chain_stages_eval ON chain_stages(chain_id) WHERE eval_assessment IS NOT NULL")
		if err != nil {
			return currentVersion, fmt.Errorf("failed to create eval_assessment index: %w", err)
		}

		_, err = db.Exec("INSERT INTO schema_version (version) VALUES (10)")
		if err != nil {
			return currentVersion, fmt.Errorf("failed to record version 10: %w", err)
		}
		currentVersion = 10
	}

	// Migration v11: Add composite index on session_tools for timestamp range queries
	// The enrichExecHierarchy function queries session_tools by (start_time, end_time) range
	// which was causing full table scans and 400% CPU burn on ailang serve.
	if currentVersion < 11 {
		_, err = db.Exec(`CREATE INDEX IF NOT EXISTS idx_session_tools_time_range ON session_tools(start_time ASC, end_time ASC)`)
		if err != nil {
			return currentVersion, fmt.Errorf("failed to create session_tools time range index: %w", err)
		}

		_, err = db.Exec("INSERT INTO schema_version (version) VALUES (11)")
		if err != nil {
			return currentVersion, fmt.Errorf("failed to record version 11: %w", err)
		}
		currentVersion = 11
	}

	// Migration v12: Add trace_summaries table for fast trace listing (M-PERF-OBSERVATORY)
	// Replaces correlated subqueries in ListTraces (3 subqueries per trace × 213K traces).
	if currentVersion < 12 {
		// Create trace_summaries table
		_, err = db.Exec(`
			CREATE TABLE IF NOT EXISTS trace_summaries (
				trace_id TEXT PRIMARY KEY,
				root_span_name TEXT,
				root_span_status TEXT,
				span_count INTEGER DEFAULT 0,
				total_duration_ms INTEGER DEFAULT 0,
				start_time TIMESTAMP,
				end_time TIMESTAMP,
				task_id TEXT,
				service_name TEXT,
				updated_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP
			)
		`)
		if err != nil {
			return currentVersion, fmt.Errorf("failed to create trace_summaries table: %w", err)
		}

		_, err = db.Exec(`CREATE INDEX IF NOT EXISTS idx_trace_summaries_time ON trace_summaries(start_time DESC)`)
		if err != nil {
			return currentVersion, fmt.Errorf("failed to create trace_summaries time index: %w", err)
		}

		_, err = db.Exec(`CREATE INDEX IF NOT EXISTS idx_trace_summaries_task ON trace_summaries(task_id)`)
		if err != nil {
			return currentVersion, fmt.Errorf("failed to create trace_summaries task index: %w", err)
		}

		// Back-fill from existing spans (one-time aggregation)
		_, err = db.Exec(`
			INSERT OR REPLACE INTO trace_summaries (trace_id, root_span_name, root_span_status, span_count, total_duration_ms, start_time, task_id, service_name, updated_at)
			SELECT
				s.trace_id,
				(SELECT name FROM spans s2 WHERE s2.trace_id = s.trace_id AND s2.parent_span_id IS NULL LIMIT 1),
				(SELECT status FROM spans s3 WHERE s3.trace_id = s.trace_id AND s3.parent_span_id IS NULL LIMIT 1),
				COUNT(*),
				COALESCE(SUM(s.duration_ms), 0),
				MIN(s.start_time),
				MAX(s.task_id),
				NULL,
				CURRENT_TIMESTAMP
			FROM spans s
			GROUP BY s.trace_id
		`)
		if err != nil {
			// Back-fill is best-effort; don't fail migration
			fmt.Printf("Warning: trace_summaries back-fill failed (will populate incrementally): %v\n", err)
		}

		_, err = db.Exec("INSERT INTO schema_version (version) VALUES (12)")
		if err != nil {
			return currentVersion, fmt.Errorf("failed to record version 12: %w", err)
		}
		currentVersion = 12
	}

	// Migration v13: Add indexes for dashboard performance (M-PERF-OBSERVATORY)
	// Breakdown/heatmap/inbox handlers scan spans by provider, model, and cost_usd
	// without indexes, causing 5-15 second queries on 761K spans.
	if currentVersion < 13 {
		indexes := []struct {
			name string
			sql  string
		}{
			{"idx_spans_provider", "CREATE INDEX IF NOT EXISTS idx_spans_provider ON spans(provider)"},
			{"idx_spans_model", "CREATE INDEX IF NOT EXISTS idx_spans_model ON spans(model)"},
			{"idx_spans_cost", "CREATE INDEX IF NOT EXISTS idx_spans_cost ON spans(cost_usd)"},
			{"idx_spans_status", "CREATE INDEX IF NOT EXISTS idx_spans_status ON spans(status)"},
			{"idx_spans_kind", "CREATE INDEX IF NOT EXISTS idx_spans_kind ON spans(kind)"},
			// Composite index for common dashboard query pattern: time-range + parent filter
			{"idx_spans_time_parent", "CREATE INDEX IF NOT EXISTS idx_spans_time_parent ON spans(start_time DESC, parent_span_id)"},
		}

		for _, idx := range indexes {
			_, err = db.Exec(idx.sql)
			if err != nil {
				return currentVersion, fmt.Errorf("failed to create index %s: %w", idx.name, err)
			}
		}

		_, err = db.Exec("INSERT INTO schema_version (version) VALUES (13)")
		if err != nil {
			return currentVersion, fmt.Errorf("failed to record version 13: %w", err)
		}
		currentVersion = 13
	}

	// Migration v14: Add workspace column to trace_summaries (M-PERF-OBSERVATORY)
	// Eliminates json_extract(resource_attributes, '$."process.cwd"') at query time.
	// Workspace breakdown was 12s+ due to parsing 130K root spans' resource_attributes.
	if currentVersion < 14 {
		// Add workspace column
		_, err = db.Exec(`ALTER TABLE trace_summaries ADD COLUMN workspace TEXT DEFAULT ''`)
		if err != nil {
			// Column might already exist from a previous partial run
			if !isColumnAlreadyExists(err) {
				return currentVersion, fmt.Errorf("failed to add workspace column: %w", err)
			}
		}

		// Back-fill workspace from root spans' resource_attributes (one-time)
		_, err = db.Exec(`
			UPDATE trace_summaries
			SET workspace = COALESCE(
				(SELECT json_extract(s.resource_attributes, '$."process.cwd"')
				 FROM spans s
				 WHERE s.trace_id = trace_summaries.trace_id
				   AND (s.parent_span_id IS NULL OR s.parent_span_id = '')
				 LIMIT 1),
				''
			)
			WHERE workspace = '' OR workspace IS NULL
		`)
		if err != nil {
			// Best-effort — don't fail migration
			fmt.Printf("Warning: trace_summaries workspace back-fill failed (will populate incrementally): %v\n", err)
		}

		_, err = db.Exec("INSERT INTO schema_version (version) VALUES (14)")
		if err != nil {
			return currentVersion, fmt.Errorf("failed to record version 14: %w", err)
		}
		currentVersion = 14
	}

	return currentVersion, nil
}

// isColumnAlreadyExists checks if an ALTER TABLE error is due to the column already existing
func isColumnAlreadyExists(err error) bool {
	if err == nil {
		return false
	}
	msg := err.Error()
	// SQLite error: "duplicate column name: workspace"
	return findSubstring(msg, "duplicate column") >= 0 || findSubstring(msg, "already exists") >= 0
}

// ValidateSchema checks that all expected tables exist.
// Returns nil if schema is valid, error describing what's missing otherwise.
// NOTE: span_events table removed in v4 migration (M-DB-CLEANUP)
// NOTE: chat_messages and chat_import_status added in v6 migration (M-CHAT-HISTORY-DB)
// NOTE: execution_chains and chain_stages added in v7 migration (M-CHAINS-SIMPLIFY)
func ValidateSchema(db *sql.DB) error {
	expectedTables := []string{
		"workspaces",
		"tasks",
		"agent_assignments",
		"spans",
		"messages",
		"sessions",
		"session_tools",
		"metrics",
		"chat_messages",
		"chat_import_status",
		"execution_chains",
		"chain_stages",
		"trace_summaries",
	}

	for _, table := range expectedTables {
		var name string
		err := db.QueryRow(
			"SELECT name FROM sqlite_master WHERE type='table' AND name=?",
			table,
		).Scan(&name)
		if err == sql.ErrNoRows {
			return fmt.Errorf("missing table: %s", table)
		}
		if err != nil {
			return fmt.Errorf("error checking table %s: %w", table, err)
		}
	}

	return nil
}
