// Package sharedmem is the SQLite implementation of the language core's
// persistent SharedCache / BrainCache seam (internal/effects).
//
// The core defines the contract, the frame types and the option surface and
// never links cgo sqlite; this package implements them and the binary
// registers it once at startup via Register (cmd/ailang/platform_init.go).
// Part of M-BRAIN (Persistent Semantic Cache) and M-V1-SIMPLIFICATION-PROGRAM
// Phase 1.3 (the core is a leaf of the platform).
package sharedmem

import (
	"bytes"
	"database/sql"
	"fmt"
	"sort"
	"time"

	"github.com/sunholo-data/ailang/internal/effects"
	"github.com/sunholo-data/ailang/internal/embedprefix"
	"github.com/sunholo-data/ailang/internal/simhash"
	"github.com/sunholo-data/ailang/internal/sqliteopen"
)

// Compile-time check: the SQLite cache satisfies the core's seam.
var _ effects.BrainCache = (*SQLiteSharedCache)(nil)

// Register installs this package as the persistent SharedCache backend.
func Register() {
	effects.RegisterSharedCacheOpener(Open)
}

// Open is the effects.SharedCacheOpener for SQLite. A nil *SQLiteSharedCache
// is never returned as a non-nil interface.
func Open(dbPath string, opts ...effects.CacheOption) (effects.BrainCache, error) {
	c, err := NewSQLiteSharedCache(dbPath, opts...)
	if err != nil {
		return nil, err
	}
	return c, nil
}

// SQLiteSharedCache is a persistent implementation of SharedCache backed by SQLite.
//
// Features:
//   - Persistent: data survives process restarts
//   - Thread-safe: SQLite WAL mode + Go-level serialization
//   - SimHash search: find similar frames by hamming distance
//   - FTS5 keyword search: full-text search on content
//   - Embedding search: cosine similarity over float32 vectors
//   - TTL support: automatic expiration via GarbageCollect
//   - Namespace support: partition frames by namespace
//
// Opened through internal/sqliteopen (WAL, NORMAL synchronous, 5s busy
// timeout, foreign keys, one writer) plus a 64MB cache.
type SQLiteSharedCache struct {
	db       *sql.DB
	embedder effects.Embedder // optional, for auto-embedding on PutFrame
}

const brainSchemaVersion = "2.0.0"

// NewSQLiteSharedCache opens or creates a SQLite-backed SharedCache at the given path.
//
// The database is opened with the sqliteopen recipe.
// If the database doesn't exist, it is created with the brain_frames schema.
// Optional effects.CacheOption values can configure the cache (e.g., WithEmbedder).
func NewSQLiteSharedCache(dbPath string, opts ...effects.CacheOption) (*SQLiteSharedCache, error) {
	// The one recipe (WAL, NORMAL synchronous, 5s busy timeout, foreign
	// keys, one writer, directory creation) — this used to be a copy of
	// messaging's pragma list. The 64MB cache is the brain's own tuning.
	db, err := sqliteopen.Open(dbPath, sqliteopen.Options{CacheSizeKB: 64000})
	if err != nil {
		return nil, fmt.Errorf("brain: failed to open database: %w", err)
	}

	// Create schema
	if err := createBrainSchema(db); err != nil {
		db.Close()
		return nil, fmt.Errorf("brain: schema creation failed: %w", err)
	}

	o := effects.ResolveCacheOptions(opts...)
	return &SQLiteSharedCache{db: db, embedder: o.Embedder}, nil
}

func createBrainSchema(db *sql.DB) error {
	// Create base tables (v1 schema)
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

	// Migrate v1 → v2: add embedding columns (non-destructive)
	if err := migrateBrainV2(db); err != nil {
		return fmt.Errorf("v2 migration failed: %w", err)
	}

	// Set schema version
	_, err := db.Exec(
		`INSERT OR REPLACE INTO brain_meta(key, value) VALUES('schema_version', ?)`,
		brainSchemaVersion,
	)
	return err
}

// migrateBrainV2 adds embedding columns to brain_frames if they don't exist.
// Safe to run multiple times — uses ADD COLUMN which is a no-op if column exists.
func migrateBrainV2(db *sql.DB) error {
	// Check if columns already exist by querying table info
	var count int
	err := db.QueryRow(`SELECT COUNT(*) FROM pragma_table_info('brain_frames') WHERE name='embedding'`).Scan(&count)
	if err != nil {
		return err
	}
	if count > 0 {
		return nil // already migrated
	}

	migrations := []string{
		`ALTER TABLE brain_frames ADD COLUMN embedding BLOB`,
		`ALTER TABLE brain_frames ADD COLUMN embedding_dim INTEGER DEFAULT 0`,
		`ALTER TABLE brain_frames ADD COLUMN embed_model TEXT`,
	}
	for _, m := range migrations {
		if _, err := db.Exec(m); err != nil {
			return fmt.Errorf("migration %q: %w", m, err)
		}
	}
	return nil
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

// PutFrame stores a frame with full metadata.
// If an embedder is configured (via WithEmbedder) and the frame has content
// but no embedding, the embedding is computed automatically. effects.Embedder errors
// are silently ignored — the frame is stored with SimHash only.
func (c *SQLiteSharedCache) PutFrame(f effects.BrainFrame) error {
	now := time.Now().UnixMilli()
	if f.CreatedAt == 0 {
		f.CreatedAt = now
	}
	if f.UpdatedAt == 0 {
		f.UpdatedAt = now
	}

	// value column is NOT NULL; use content as canonical blob when value is empty.
	if len(f.Value) == 0 && f.Content != "" {
		f.Value = []byte(f.Content)
	}

	// Auto-embed: if we have an embedder, content, and no existing embedding.
	// Corpus content is a DOCUMENT — prefix it for the embedder's retrieval task
	// (EmbeddingGemma/nomic) so it shares a vector space with prefixed queries.
	if c.embedder != nil && f.Content != "" && len(f.Embedding) == 0 {
		if emb, err := embedprefix.EmbedWithRole(c.embedder, embedprefix.RoleDocument, f.Content); err == nil && len(emb) > 0 {
			f.Embedding = emb
			f.EmbeddingDim = len(emb)
			f.EmbedModel = c.embedder.ModelName()
		}
		// effects.Embedder errors silently ignored — falls back to SimHash only
	}

	var embBlob []byte
	if len(f.Embedding) > 0 {
		embBlob = effects.EncodeEmbedding(f.Embedding)
		f.EmbeddingDim = len(f.Embedding)
	}

	_, err := c.db.Exec(
		`INSERT INTO brain_frames(key, namespace, value, simhash, content, version, created_at, updated_at, expires_at, source, embedding, embedding_dim, embed_model)
		 VALUES(?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?)
		 ON CONFLICT(key) DO UPDATE SET
		   namespace=excluded.namespace, value=excluded.value, simhash=excluded.simhash,
		   content=excluded.content, version=excluded.version, updated_at=excluded.updated_at,
		   expires_at=excluded.expires_at, source=excluded.source,
		   embedding=excluded.embedding, embedding_dim=excluded.embedding_dim, embed_model=excluded.embed_model`,
		f.Key, f.Namespace, f.Value, f.SimHash, f.Content, f.Version,
		f.CreatedAt, f.UpdatedAt, f.ExpiresAt, f.Source,
		embBlob, f.EmbeddingDim, f.EmbedModel,
	)
	return err
}

// GetFrame reads one frame with full metadata. Returns (nil, false) when the
// key is absent or the row cannot be scanned.
func (c *SQLiteSharedCache) GetFrame(key string) (*effects.BrainFrame, bool) {
	rows, err := c.db.Query(
		`SELECT key, namespace, value, simhash, content, version, created_at, updated_at, expires_at, source, embedding, embedding_dim, embed_model
		 FROM brain_frames WHERE key = ?`, key,
	)
	if err != nil {
		return nil, false
	}
	defer rows.Close()
	if !rows.Next() {
		return nil, false
	}
	f := scanBrainFrame(rows)
	if f == nil {
		return nil, false
	}
	return f, true
}

// PutVector stores a frame with an embedding but no text content.
// Used for machine-to-machine vector communication.
func (c *SQLiteSharedCache) PutVector(key, namespace string, embedding []float32, model string, payload []byte) error {
	f := effects.BrainFrame{
		Key:          key,
		Namespace:    namespace,
		Value:        payload,
		Embedding:    embedding,
		EmbeddingDim: len(embedding),
		EmbedModel:   model,
		Source:       "vector",
	}
	return c.PutFrame(f)
}

// SearchBySimHash finds frames with similar SimHash values in a given namespace.
// Returns results sorted by score descending, key ascending (deterministic).
func (c *SQLiteSharedCache) SearchBySimHash(namespace string, queryHash int64, limit int) []effects.BrainSearchResult {
	var rows *sql.Rows
	var err error
	if namespace != "" {
		rows, err = c.db.Query(
			`SELECT key, namespace, value, simhash, content, version, created_at, updated_at, expires_at, source, embedding, embedding_dim, embed_model
			 FROM brain_frames WHERE namespace = ? AND simhash IS NOT NULL`,
			namespace,
		)
	} else {
		rows, err = c.db.Query(
			`SELECT key, namespace, value, simhash, content, version, created_at, updated_at, expires_at, source, embedding, embedding_dim, embed_model
			 FROM brain_frames WHERE simhash IS NOT NULL`,
		)
	}
	if err != nil {
		return nil
	}
	defer rows.Close()

	var results []effects.BrainSearchResult
	for rows.Next() {
		f := scanBrainFrame(rows)
		if f == nil {
			continue
		}
		score := simhash.Similarity(queryHash, f.SimHash)
		results = append(results, effects.BrainSearchResult{Frame: *f, Score: score})
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

// SearchByText performs keyword search across all namespaces (or a specific one).
// If namespace is empty, searches all namespaces.
func (c *SQLiteSharedCache) SearchByText(query string, namespace string, limit int) []effects.BrainSearchResult {
	var rows *sql.Rows
	var err error

	if limit <= 0 {
		limit = 1000 // no limit → reasonable max
	}

	if namespace != "" {
		rows, err = c.db.Query(
			`SELECT bf.key, bf.namespace, bf.value, bf.simhash, bf.content,
			        bf.version, bf.created_at, bf.updated_at, bf.expires_at, bf.source,
			        bf.embedding, bf.embedding_dim, bf.embed_model
			 FROM brain_frames bf
			 WHERE bf.namespace = ? AND (bf.content LIKE '%' || ? || '%' OR bf.key LIKE '%' || ? || '%')
			 ORDER BY bf.updated_at DESC
			 LIMIT ?`,
			namespace, query, query, limit,
		)
	} else {
		rows, err = c.db.Query(
			`SELECT bf.key, bf.namespace, bf.value, bf.simhash, bf.content,
			        bf.version, bf.created_at, bf.updated_at, bf.expires_at, bf.source,
			        bf.embedding, bf.embedding_dim, bf.embed_model
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

	var results []effects.BrainSearchResult
	for rows.Next() {
		f := scanBrainFrame(rows)
		if f == nil {
			continue
		}
		results = append(results, effects.BrainSearchResult{Frame: *f, Score: 1.0})
	}
	return results
}

// ListRecent returns the most recently updated frames.
func (c *SQLiteSharedCache) ListRecent(namespace string, limit int) []effects.BrainFrame {
	var rows *sql.Rows
	var err error

	if namespace != "" {
		rows, err = c.db.Query(
			`SELECT key, namespace, value, simhash, content, version, created_at, updated_at, expires_at, source, embedding, embedding_dim, embed_model
			 FROM brain_frames WHERE namespace = ? ORDER BY updated_at DESC LIMIT ?`,
			namespace, limit,
		)
	} else {
		rows, err = c.db.Query(
			`SELECT key, namespace, value, simhash, content, version, created_at, updated_at, expires_at, source, embedding, embedding_dim, embed_model
			 FROM brain_frames ORDER BY updated_at DESC LIMIT ?`,
			limit,
		)
	}
	if err != nil {
		return nil
	}
	defer rows.Close()

	var frames []effects.BrainFrame
	for rows.Next() {
		f := scanBrainFrame(rows)
		if f == nil {
			continue
		}
		frames = append(frames, *f)
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

// DeleteNamespace removes all frames in the given namespace.
// Used by the micro-rag indexer to support release-tied corpus reset.
// Returns the number of frames removed.
func (c *SQLiteSharedCache) DeleteNamespace(namespace string) (int64, error) {
	result, err := c.db.Exec(`DELETE FROM brain_frames WHERE namespace = ?`, namespace)
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

// Stats returns aggregate statistics about the brain.
func (c *SQLiteSharedCache) Stats() effects.BrainStats {
	var stats effects.BrainStats
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

// SearchByEmbedding performs brute-force cosine similarity scan over all frames with embeddings.
// Returns results sorted by cosine similarity descending, key ascending (deterministic).
func (c *SQLiteSharedCache) SearchByEmbedding(queryEmbedding []float32, namespace string, limit int) []effects.BrainSearchResult {
	var rows *sql.Rows
	var err error

	if namespace != "" {
		rows, err = c.db.Query(
			`SELECT key, namespace, value, simhash, content, version, created_at, updated_at, expires_at, source, embedding, embedding_dim, embed_model
			 FROM brain_frames WHERE embedding IS NOT NULL AND namespace = ?`,
			namespace,
		)
	} else {
		rows, err = c.db.Query(
			`SELECT key, namespace, value, simhash, content, version, created_at, updated_at, expires_at, source, embedding, embedding_dim, embed_model
			 FROM brain_frames WHERE embedding IS NOT NULL`,
		)
	}
	if err != nil {
		return nil
	}
	defer rows.Close()

	var results []effects.BrainSearchResult
	for rows.Next() {
		f := scanBrainFrame(rows)
		if f == nil || len(f.Embedding) == 0 {
			continue
		}
		// Cosine scores a dimension mismatch (an embedding from another model)
		// as 0, never a prefix comparison: there is no direction to compare.
		score := simhash.Cosine(queryEmbedding, f.Embedding)
		results = append(results, effects.BrainSearchResult{Frame: *f, Score: score})
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

// EmbeddingStats returns embedding coverage statistics.
func (c *SQLiteSharedCache) EmbeddingStats() (total, withEmbedding int, models map[string]int) {
	models = make(map[string]int)
	_ = c.db.QueryRow(`SELECT COUNT(*) FROM brain_frames`).Scan(&total)
	_ = c.db.QueryRow(`SELECT COUNT(*) FROM brain_frames WHERE embedding IS NOT NULL`).Scan(&withEmbedding)

	rows, err := c.db.Query(`SELECT embed_model, COUNT(*) FROM brain_frames WHERE embed_model IS NOT NULL AND embed_model != '' GROUP BY embed_model`)
	if err == nil {
		defer rows.Close()
		for rows.Next() {
			var model string
			var count int
			if rows.Scan(&model, &count) == nil {
				models[model] = count
			}
		}
	}
	return
}

// SetEmbedder configures the embedder for auto-embedding on PutFrame.
func (c *SQLiteSharedCache) SetEmbedder(e effects.Embedder) {
	c.embedder = e
}

// GetEmbedder returns the configured embedder (may be nil).
func (c *SQLiteSharedCache) GetEmbedder() effects.Embedder {
	return c.embedder
}

// BackfillEmbeddings computes embeddings for all frames that have content
// but no embedding. Returns (processed, errors) counts.
func (c *SQLiteSharedCache) BackfillEmbeddings(namespace string) (int, int) {
	if c.embedder == nil {
		return 0, 0
	}

	query := `SELECT key, content FROM brain_frames WHERE embedding IS NULL AND content IS NOT NULL AND content != ''`
	args := []interface{}{}
	if namespace != "" {
		query += ` AND namespace = ?`
		args = append(args, namespace)
	}

	rows, err := c.db.Query(query, args...)
	if err != nil {
		return 0, 0
	}

	// Collect all rows first to release the DB connection (MaxOpenConns=1)
	type pending struct {
		key, content string
	}
	var items []pending
	for rows.Next() {
		var p pending
		if rows.Scan(&p.key, &p.content) == nil {
			items = append(items, p)
		}
	}
	rows.Close()

	var processed, errCount int
	for _, p := range items {
		// Backfill is corpus content → DOCUMENT role (same as the upsert path).
		emb, err := embedprefix.EmbedWithRole(c.embedder, embedprefix.RoleDocument, p.content)
		if err != nil || len(emb) == 0 {
			errCount++
			continue
		}

		embBlob := effects.EncodeEmbedding(emb)
		_, err = c.db.Exec(
			`UPDATE brain_frames SET embedding = ?, embedding_dim = ?, embed_model = ?, updated_at = ? WHERE key = ?`,
			embBlob, len(emb), c.embedder.ModelName(), time.Now().UnixMilli(), p.key,
		)
		if err != nil {
			errCount++
			continue
		}
		processed++
	}
	return processed, errCount
}

// DB returns the underlying database connection for advanced operations.
func (c *SQLiteSharedCache) DB() *sql.DB {
	return c.db
}

// --- Embedding helpers ---

// scanBrainFrame scans a full row (13 columns) into a effects.BrainFrame.
func scanBrainFrame(rows *sql.Rows) *effects.BrainFrame {
	var f effects.BrainFrame
	var embBlob []byte
	var embedModel sql.NullString
	if err := rows.Scan(&f.Key, &f.Namespace, &f.Value, &f.SimHash, &f.Content,
		&f.Version, &f.CreatedAt, &f.UpdatedAt, &f.ExpiresAt, &f.Source,
		&embBlob, &f.EmbeddingDim, &embedModel); err != nil {
		return nil
	}
	if embedModel.Valid {
		f.EmbedModel = embedModel.String
	}
	if len(embBlob) > 0 {
		f.Embedding = effects.DecodeEmbedding(embBlob)
	}
	return &f
}
