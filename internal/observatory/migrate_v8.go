// Package observatory provides database migrations v8-v15 for the observatory schema.
package observatory

import (
	"database/sql"
	"fmt"
)

// migrateV8ToV15 runs migrations from v8 to v15.
// Called from MigrateWithVersion when currentVersion < 15.
func migrateV8ToV15(db *sql.DB, currentVersion int) (int, error) {
	var err error

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
	if currentVersion < 12 {
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
	if currentVersion < 14 {
		_, err = db.Exec(`ALTER TABLE trace_summaries ADD COLUMN workspace TEXT DEFAULT ''`)
		if err != nil {
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
			fmt.Printf("Warning: trace_summaries workspace back-fill failed (will populate incrementally): %v\n", err)
		}

		_, err = db.Exec("INSERT INTO schema_version (version) VALUES (14)")
		if err != nil {
			return currentVersion, fmt.Errorf("failed to record version 14: %w", err)
		}
		currentVersion = 14
	}

	// Migration v15: Create eval_baselines table for adaptive token-budget
	// baselines (M-EVAL-OS-LONGITUDINAL Phase 2).
	//
	// Bug fix: eval_baselines was originally added ONLY to the base Migrate()
	// schema. MigrateWithVersion runs Migrate() exactly once (currentVersion < 1),
	// so every observatory.db already past v1 — i.e. every existing DB, now at
	// v14 — never received the table. The eval-suite then logged "no such table:
	// eval_baselines" on every passing trial and silently recorded no baselines.
	// This versioned step backfills the table on existing DBs (mirrors how
	// trace_summaries got a v12 migration alongside its base-schema entry). The
	// CREATE is IF NOT EXISTS, so fresh DBs that already got it from Migrate()
	// are a harmless no-op.
	if currentVersion < 15 {
		_, err = db.Exec(`
			CREATE TABLE IF NOT EXISTS eval_baselines (
				model_id      TEXT NOT NULL,
				benchmark_id  TEXT NOT NULL,
				n_pass_trials INTEGER NOT NULL DEFAULT 0,
				mean_tokens   REAL    NOT NULL DEFAULT 0,
				stddev_tokens REAL    NOT NULL DEFAULT 0,
				m2_tokens     REAL    NOT NULL DEFAULT 0,
				last_updated  TIMESTAMP NOT NULL,
				PRIMARY KEY (model_id, benchmark_id)
			)
		`)
		if err != nil {
			return currentVersion, fmt.Errorf("failed to create eval_baselines table: %w", err)
		}

		_, err = db.Exec("INSERT INTO schema_version (version) VALUES (15)")
		if err != nil {
			return currentVersion, fmt.Errorf("failed to record version 15: %w", err)
		}
		currentVersion = 15
	}

	return currentVersion, nil
}
