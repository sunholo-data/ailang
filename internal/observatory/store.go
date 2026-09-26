// Package observatory provides a unified observability platform for AILANG.
package observatory

import (
	"database/sql"
	"fmt"
	"sync"
	"time"

	"github.com/sunholo-data/ailang/internal/sqliteopen"
)

// chainLink caches the chain_id/stage_id mapping for a task_id.
type chainLink struct {
	chainID string
	stageID string
}

// Store provides CRUD operations for the observatory platform.
type Store struct {
	db *sql.DB
	// dbPath is the on-disk file OpenStore opened, "" for a caller-supplied
	// connection. RunRetention stamps a sibling file so CheckHealth can tell
	// a recent pass from a stat, without opening the DB.
	dbPath string

	// Write-time chain linking cache (M-AUDIT-OBSERVATORY)
	chainLinkCache map[string]chainLink
	chainLinkMu    sync.RWMutex
}

// NewStore creates a new Store with the given database connection.
func NewStore(db *sql.DB) *Store {
	return &Store{
		db:             db,
		chainLinkCache: make(map[string]chainLink),
	}
}

// DB returns the underlying database connection.
func (s *Store) DB() *sql.DB {
	return s.db
}

// Close closes the underlying database connection.
func (s *Store) Close() error {
	return s.db.Close()
}

// OpenDefaultStore opens the observatory database at the default path,
// runs migrations, and returns a ready-to-use Store.
// This is the recommended way to access observatory.db from CLI tools.
func OpenDefaultStore() (*Store, error) {
	return OpenStore(DefaultDatabasePath())
}

// OpenStore opens the observatory database at the given path, runs
// migrations, and returns a ready-to-use Store. It is the ONE opener for
// observatory.db (NewSQLiteBackendFromPath wraps it).
//
// Pool and pragmas (M-V1-SIMPLIFY-S3 M3): one open connection and foreign
// keys ON — the settings NewSQLiteBackendFromPath had, not the default pool
// and no-FK this path had. One connection because the observatory is the
// busiest writer in the process (every span) and SQLite serialises writers
// anyway; foreign keys because session_tools declares ON DELETE CASCADE and
// an opener that ignored it would leave orphans a later reader trips over.
func OpenStore(dbPath string) (*Store, error) {
	if dbPath == "" {
		return nil, errNoDatabasePath
	}
	db, err := sqliteopen.Open(dbPath, sqliteopen.Options{})
	if err != nil {
		return nil, fmt.Errorf("failed to open observatory database: %w", err)
	}

	if _, err := MigrateWithVersion(db); err != nil {
		db.Close()
		return nil, fmt.Errorf("failed to run observatory migrations: %w", err)
	}

	// Checkpoint WAL on open to prevent unbounded WAL growth.
	// Without this, the WAL can grow to 40GB+ and cause memory pressure.
	db.Exec("PRAGMA wal_checkpoint(TRUNCATE)") //nolint:errcheck

	st := NewStore(db)
	st.dbPath = dbPath
	return st, nil
}

// ===== Workspace Operations =====

// CreateWorkspace inserts a new workspace.
func (s *Store) CreateWorkspace(w *Workspace) error {
	_, err := s.db.Exec(`
		INSERT INTO workspaces (id, name, path, git_remote, created_at, updated_at)
		VALUES (?, ?, ?, ?, ?, ?)
	`, w.ID, w.Name, w.Path, w.GitRemote, w.CreatedAt, w.UpdatedAt)
	return err
}

// GetWorkspace retrieves a workspace by ID.
func (s *Store) GetWorkspace(id string) (*Workspace, error) {
	w := &Workspace{}
	var gitRemote sql.NullString
	err := s.db.QueryRow(`
		SELECT id, name, path, git_remote, created_at, updated_at
		FROM workspaces WHERE id = ?
	`, id).Scan(&w.ID, &w.Name, &w.Path, &gitRemote, &w.CreatedAt, &w.UpdatedAt)
	if err != nil {
		return nil, err
	}
	if gitRemote.Valid {
		w.GitRemote = gitRemote.String
	}
	return w, nil
}

// ListWorkspaces returns all workspaces.
func (s *Store) ListWorkspaces() ([]*Workspace, error) {
	rows, err := s.db.Query(`
		SELECT id, name, path, git_remote, created_at, updated_at
		FROM workspaces ORDER BY updated_at DESC
	`)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var workspaces []*Workspace
	for rows.Next() {
		w := &Workspace{}
		var gitRemote sql.NullString
		if err := rows.Scan(&w.ID, &w.Name, &w.Path, &gitRemote, &w.CreatedAt, &w.UpdatedAt); err != nil {
			return nil, err
		}
		if gitRemote.Valid {
			w.GitRemote = gitRemote.String
		}
		workspaces = append(workspaces, w)
	}
	return workspaces, rows.Err()
}

// UpdateWorkspace updates an existing workspace.
func (s *Store) UpdateWorkspace(w *Workspace) error {
	w.UpdatedAt = time.Now()
	_, err := s.db.Exec(`
		UPDATE workspaces SET name = ?, path = ?, git_remote = ?, updated_at = ?
		WHERE id = ?
	`, w.Name, w.Path, w.GitRemote, w.UpdatedAt, w.ID)
	return err
}

// DeleteWorkspace removes a workspace by ID.
func (s *Store) DeleteWorkspace(id string) error {
	_, err := s.db.Exec("DELETE FROM workspaces WHERE id = ?", id)
	return err
}

// GetWorkspaceStats returns aggregated stats for a workspace.
func (s *Store) GetWorkspaceStats(workspaceID string) (*WorkspaceStats, error) {
	stats := &WorkspaceStats{}
	var lastActivity sql.NullTime
	err := s.db.QueryRow(`
		SELECT id, name, path, task_count, total_cost, total_tokens,
		       success_rate, unique_agents, last_activity
		FROM workspace_stats WHERE id = ?
	`, workspaceID).Scan(
		&stats.ID, &stats.Name, &stats.Path, &stats.TaskCount, &stats.TotalCost,
		&stats.TotalTokens, &stats.SuccessRate, &stats.UniqueAgents, &lastActivity,
	)
	if err != nil {
		return nil, err
	}
	if lastActivity.Valid {
		stats.LastActivity = lastActivity.Time
	}
	return stats, nil
}
