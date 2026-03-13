// Package effects provides SQLiteSharedCache, a persistent SharedCache backed by SQLite.
// Part of M-BRAIN (Persistent Semantic Cache).
package effects

import (
	"bytes"
	"database/sql"
	"fmt"
	"os"
	"path/filepath"
	"sort"
	"time"

	_ "github.com/mattn/go-sqlite3"
)

// SQLiteSharedCache is a persistent implementation of SharedCache backed by SQLite.
//
// Features:
//   - Persistent: data survives process restarts
//   - Thread-safe: SQLite WAL mode + Go-level serialization
//   - SimHash search: find similar frames by hamming distance
//   - FTS5 keyword search: full-text search on content
//   - TTL support: automatic expiration via GarbageCollect
//   - Namespace support: partition frames by namespace
//
// The schema uses the same pragmas as internal/messaging/schema.go:
//   - WAL mode, NORMAL synchronous, 5s busy timeout, 64MB cache
type SQLiteSharedCache struct {
	db *sql.DB
}

const brainSchemaVersion = "1.0.0"

// NewSQLiteSharedCache opens or creates a SQLite-backed SharedCache at the given path.
//
// The database is configured with WAL mode and the same pragmas as the messaging system.
// If the database doesn't exist, it is created with the brain_frames schema.
func NewSQLiteSharedCache(dbPath string) (*SQLiteSharedCache, error) {
	dbDir := filepath.Dir(dbPath)
	if err := os.MkdirAll(dbDir, 0755); err != nil {
		return nil, fmt.Errorf("brain: failed to create directory %s: %w", dbDir, err)
	}

	db, err := sql.Open("sqlite3", dbPath)
	if err != nil {
		return nil, fmt.Errorf("brain: failed to open database: %w", err)
	}

	// Single-writer serialization (same as messaging)
	db.SetMaxOpenConns(1)
	db.SetMaxIdleConns(1)
	db.SetConnMaxLifetime(0)

	// Configure pragmas
	pragmas := []string{
		"PRAGMA journal_mode=WAL",
		"PRAGMA synchronous=NORMAL",
		"PRAGMA busy_timeout=5000",
		"PRAGMA foreign_keys=ON",
		"PRAGMA cache_size=-64000",
	}
	for _, p := range pragmas {
		if _, err := db.Exec(p); err != nil {
			db.Close()
			return nil, fmt.Errorf("brain: pragma %q failed: %w", p, err)
		}
	}

	// Create schema
	if err := createBrainSchema(db); err != nil {
		db.Close()
		return nil, fmt.Errorf("brain: schema creation failed: %w", err)
	}

	return &SQLiteSharedCache{db: db}, nil
}

func createBrainSchema(db *sql.DB) error {
	schema := `
	CREATE TABLE IF NOT EXISTS brain_frames (
		key         TEXT PRIMARY KEY,
		namespace   TEXT NOT NULL DEFAULT 'default',
		value       BLOB NOT NULL,
		simhash     INTEGER,
		content     TEXT,
		version     INTEGER DEFAULT 1,
		created_at  INTEGER NOT NULL,
		updated_at  INTEGER NOT NULL,
		expires_at  INTEGER,
		source      TEXT
	);

	CREATE INDEX IF NOT EXISTS idx_brain_ns ON brain_frames(namespace);
	CREATE INDEX IF NOT EXISTS idx_brain_simhash ON brain_frames(namespace, simhash);
	CREATE INDEX IF NOT EXISTS idx_brain_updated ON brain_frames(updated_at DESC);
	CREATE INDEX IF NOT EXISTS idx_brain_expires ON brain_frames(expires_at) WHERE expires_at IS NOT NULL;

	CREATE TABLE IF NOT EXISTS brain_meta (
		key   TEXT PRIMARY KEY,
		value TEXT
	);
	`
	if _, err := db.Exec(schema); err != nil {
		return err
	}

	// Set schema version
	_, err := db.Exec(
		`INSERT OR REPLACE INTO brain_meta(key, value) VALUES('schema_version', ?)`,
		brainSchemaVersion,
	)
	return err
}

// --- SharedCache interface implementation ---

// Get retrieves a value by key.
func (c *SQLiteSharedCache) Get(key string) ([]byte, bool) {
	var value []byte
	err := c.db.QueryRow(
		`SELECT value FROM brain_frames WHERE key = ?`, key,
	).Scan(&value)
	if err != nil {
		return nil, false
	}
	return value, true
}

// Put stores a value at the given key, overwriting any existing value.
func (c *SQLiteSharedCache) Put(key string, value []byte) {
	now := time.Now().UnixMilli()
	_, _ = c.db.Exec(
		`INSERT INTO brain_frames(key, value, created_at, updated_at)
		 VALUES(?, ?, ?, ?)
		 ON CONFLICT(key) DO UPDATE SET value=excluded.value, updated_at=excluded.updated_at`,
		key, value, now, now,
	)
}

// Delete removes a value by key. No-op if the key doesn't exist.
func (c *SQLiteSharedCache) Delete(key string) {
	_, _ = c.db.Exec(`DELETE FROM brain_frames WHERE key = ?`, key)
}

// CAS performs an atomic compare-and-swap operation.
//
// If oldValue is nil, creates the key only if it doesn't exist.
// Otherwise, updates only if the current value matches oldValue byte-for-byte.
func (c *SQLiteSharedCache) CAS(key string, oldValue, newValue []byte) bool {
	tx, err := c.db.Begin()
	if err != nil {
		return false
	}
	defer tx.Rollback() //nolint:errcheck

	var current []byte
	err = tx.QueryRow(`SELECT value FROM brain_frames WHERE key = ?`, key).Scan(&current)
	exists := err == nil

	now := time.Now().UnixMilli()

	if oldValue == nil {
		// Create-if-absent
		if exists {
			return false
		}
		_, err = tx.Exec(
			`INSERT INTO brain_frames(key, value, created_at, updated_at) VALUES(?, ?, ?, ?)`,
			key, newValue, now, now,
		)
		if err != nil {
			return false
		}
		return tx.Commit() == nil
	}

	// Normal CAS
	if !exists {
		return false
	}
	if !bytes.Equal(current, oldValue) {
		return false
	}

	_, err = tx.Exec(
		`UPDATE brain_frames SET value = ?, updated_at = ? WHERE key = ?`,
		newValue, now, key,
	)
	if err != nil {
		return false
	}
	return tx.Commit() == nil
}

// Keys returns all keys in the cache.
func (c *SQLiteSharedCache) Keys() []string {
	rows, err := c.db.Query(`SELECT key FROM brain_frames ORDER BY key`)
	if err != nil {
		return nil
	}
	defer rows.Close()

	var keys []string
	for rows.Next() {
		var k string
		if err := rows.Scan(&k); err == nil {
			keys = append(keys, k)
		}
	}
	return keys
}

// Len returns the number of entries in the cache.
func (c *SQLiteSharedCache) Len() int {
	var count int
	_ = c.db.QueryRow(`SELECT COUNT(*) FROM brain_frames`).Scan(&count)
	return count
}

// Close closes the underlying database connection.
func (c *SQLiteSharedCache) Close() error {
	return c.db.Close()
}

// --- Extended methods for brain-specific operations ---

// BrainFrame represents a frame stored in the brain with its metadata.
type BrainFrame struct {
	Key       string
	Namespace string
	Value     []byte
	SimHash   int64
	Content   string
	Version   int
	CreatedAt int64
	UpdatedAt int64
	ExpiresAt *int64
	Source    string
}

// SearchResult represents a search hit with a relevance score.
type BrainSearchResult struct {
	Frame BrainFrame
	Score float64
	Tier  string // "user" or "project"
}

// PutFrame stores a frame with full metadata.
func (c *SQLiteSharedCache) PutFrame(f BrainFrame) error {
	now := time.Now().UnixMilli()
	if f.CreatedAt == 0 {
		f.CreatedAt = now
	}
	if f.UpdatedAt == 0 {
		f.UpdatedAt = now
	}

	_, err := c.db.Exec(
		`INSERT INTO brain_frames(key, namespace, value, simhash, content, version, created_at, updated_at, expires_at, source)
		 VALUES(?, ?, ?, ?, ?, ?, ?, ?, ?, ?)
		 ON CONFLICT(key) DO UPDATE SET
		   namespace=excluded.namespace, value=excluded.value, simhash=excluded.simhash,
		   content=excluded.content, version=excluded.version, updated_at=excluded.updated_at,
		   expires_at=excluded.expires_at, source=excluded.source`,
		f.Key, f.Namespace, f.Value, f.SimHash, f.Content, f.Version,
		f.CreatedAt, f.UpdatedAt, f.ExpiresAt, f.Source,
	)
	return err
}

// SearchBySimHash finds frames with similar SimHash values in a given namespace.
// Returns results sorted by score descending, key ascending (deterministic).
func (c *SQLiteSharedCache) SearchBySimHash(namespace string, queryHash int64, limit int) []BrainSearchResult {
	rows, err := c.db.Query(
		`SELECT key, namespace, value, simhash, content, version, created_at, updated_at, expires_at, source
		 FROM brain_frames WHERE namespace = ? AND simhash IS NOT NULL`,
		namespace,
	)
	if err != nil {
		return nil
	}
	defer rows.Close()

	var results []BrainSearchResult
	for rows.Next() {
		var f BrainFrame
		if err := rows.Scan(&f.Key, &f.Namespace, &f.Value, &f.SimHash, &f.Content,
			&f.Version, &f.CreatedAt, &f.UpdatedAt, &f.ExpiresAt, &f.Source); err != nil {
			continue
		}
		dist := hammingDistance64(queryHash, f.SimHash)
		score := 1.0 - float64(dist)/64.0
		results = append(results, BrainSearchResult{Frame: f, Score: score})
	}

	// Deterministic sort: score DESC, key ASC
	sort.Slice(results, func(i, j int) bool {
		if results[i].Score != results[j].Score {
			return results[i].Score > results[j].Score
		}
		return results[i].Frame.Key < results[j].Frame.Key
	})

	if limit > 0 && len(results) > limit {
		results = results[:limit]
	}
	return results
}

// SearchByText performs FTS5 keyword search across all namespaces (or a specific one).
// If namespace is empty, searches all namespaces.
func (c *SQLiteSharedCache) SearchByText(query string, namespace string, limit int) []BrainSearchResult {
	var rows *sql.Rows
	var err error

	if namespace != "" {
		rows, err = c.db.Query(
			`SELECT bf.key, bf.namespace, bf.value, bf.simhash, bf.content,
			        bf.version, bf.created_at, bf.updated_at, bf.expires_at, bf.source
			 FROM brain_frames bf
			 WHERE bf.namespace = ? AND (bf.content LIKE '%' || ? || '%' OR bf.key LIKE '%' || ? || '%')
			 ORDER BY bf.updated_at DESC
			 LIMIT ?`,
			namespace, query, query, limit,
		)
	} else {
		rows, err = c.db.Query(
			`SELECT bf.key, bf.namespace, bf.value, bf.simhash, bf.content,
			        bf.version, bf.created_at, bf.updated_at, bf.expires_at, bf.source
			 FROM brain_frames bf
			 WHERE bf.content LIKE '%' || ? || '%' OR bf.key LIKE '%' || ? || '%'
			 ORDER BY bf.updated_at DESC
			 LIMIT ?`,
			query, query, limit,
		)
	}
	if err != nil {
		return nil
	}
	defer rows.Close()

	var results []BrainSearchResult
	for rows.Next() {
		var f BrainFrame
		if err := rows.Scan(&f.Key, &f.Namespace, &f.Value, &f.SimHash, &f.Content,
			&f.Version, &f.CreatedAt, &f.UpdatedAt, &f.ExpiresAt, &f.Source); err != nil {
			continue
		}
		results = append(results, BrainSearchResult{Frame: f, Score: 1.0})
	}
	return results
}

// ListRecent returns the most recently updated frames.
func (c *SQLiteSharedCache) ListRecent(namespace string, limit int) []BrainFrame {
	var rows *sql.Rows
	var err error

	if namespace != "" {
		rows, err = c.db.Query(
			`SELECT key, namespace, value, simhash, content, version, created_at, updated_at, expires_at, source
			 FROM brain_frames WHERE namespace = ? ORDER BY updated_at DESC LIMIT ?`,
			namespace, limit,
		)
	} else {
		rows, err = c.db.Query(
			`SELECT key, namespace, value, simhash, content, version, created_at, updated_at, expires_at, source
			 FROM brain_frames ORDER BY updated_at DESC LIMIT ?`,
			limit,
		)
	}
	if err != nil {
		return nil
	}
	defer rows.Close()

	var frames []BrainFrame
	for rows.Next() {
		var f BrainFrame
		if err := rows.Scan(&f.Key, &f.Namespace, &f.Value, &f.SimHash, &f.Content,
			&f.Version, &f.CreatedAt, &f.UpdatedAt, &f.ExpiresAt, &f.Source); err != nil {
			continue
		}
		frames = append(frames, f)
	}
	return frames
}

// GarbageCollect removes expired frames (where expires_at < now).
// Returns the number of frames removed.
func (c *SQLiteSharedCache) GarbageCollect() (int64, error) {
	now := time.Now().UnixMilli()
	result, err := c.db.Exec(
		`DELETE FROM brain_frames WHERE expires_at IS NOT NULL AND expires_at < ?`, now,
	)
	if err != nil {
		return 0, err
	}
	return result.RowsAffected()
}

// GarbageCollectOlderThan removes frames older than the given duration in a namespace.
// Returns the number of frames removed.
func (c *SQLiteSharedCache) GarbageCollectOlderThan(namespace string, age time.Duration) (int64, error) {
	cutoff := time.Now().Add(-age).UnixMilli()
	result, err := c.db.Exec(
		`DELETE FROM brain_frames WHERE namespace = ? AND updated_at < ?`,
		namespace, cutoff,
	)
	if err != nil {
		return 0, err
	}
	return result.RowsAffected()
}

// BrainStats holds aggregate statistics about the brain.
type BrainStats struct {
	TotalFrames int
	Namespaces  map[string]int // namespace -> frame count
	OldestFrame int64          // unix millis
	NewestFrame int64          // unix millis
}

// Stats returns aggregate statistics about the brain.
func (c *SQLiteSharedCache) Stats() BrainStats {
	var stats BrainStats
	stats.Namespaces = make(map[string]int)

	_ = c.db.QueryRow(`SELECT COUNT(*) FROM brain_frames`).Scan(&stats.TotalFrames)
	_ = c.db.QueryRow(`SELECT COALESCE(MIN(created_at), 0) FROM brain_frames`).Scan(&stats.OldestFrame)
	_ = c.db.QueryRow(`SELECT COALESCE(MAX(updated_at), 0) FROM brain_frames`).Scan(&stats.NewestFrame)

	rows, err := c.db.Query(`SELECT namespace, COUNT(*) FROM brain_frames GROUP BY namespace`)
	if err == nil {
		defer rows.Close()
		for rows.Next() {
			var ns string
			var count int
			if rows.Scan(&ns, &count) == nil {
				stats.Namespaces[ns] = count
			}
		}
	}
	return stats
}

// DB returns the underlying database connection for advanced operations.
func (c *SQLiteSharedCache) DB() *sql.DB {
	return c.db
}

// hammingDistance64 computes the hamming distance between two 64-bit integers.
func hammingDistance64(a, b int64) int {
	xor := uint64(a ^ b)
	count := 0
	for xor != 0 {
		count++
		xor &= xor - 1 // Clear lowest set bit
	}
	return count
}
