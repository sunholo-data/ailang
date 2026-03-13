// Package effects provides SQLiteSharedCache, a persistent SharedCache backed by SQLite.
// Part of M-BRAIN (Persistent Semantic Cache).
package effects

import (
	"bytes"
	"database/sql"
	"encoding/binary"
	"fmt"
	"math"
	"os"
	"path/filepath"
	"sort"
	"time"

	_ "github.com/mattn/go-sqlite3"
)

// Embedder is the interface for generating text embeddings.
// Matches messaging.Embedder but defined here to avoid circular imports.
type Embedder interface {
	Embed(text string) ([]float32, error)
	EmbedBatch(texts []string) ([][]float32, error)
	Dimension() int
	ModelName() string
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
// The schema uses the same pragmas as internal/messaging/schema.go:
//   - WAL mode, NORMAL synchronous, 5s busy timeout, 64MB cache
type SQLiteSharedCache struct {
	db       *sql.DB
	embedder Embedder // optional, for auto-embedding on PutFrame
}

// CacheOption configures a SQLiteSharedCache after creation.
type CacheOption func(*SQLiteSharedCache)

// WithEmbedder sets an embedder for auto-embedding on PutFrame.
// When set, PutFrame will automatically compute and store embeddings
// for frames that have content but no embedding.
func WithEmbedder(e Embedder) CacheOption {
	return func(c *SQLiteSharedCache) {
		c.embedder = e
	}
}

const brainSchemaVersion = "2.0.0"

// NewSQLiteSharedCache opens or creates a SQLite-backed SharedCache at the given path.
//
// The database is configured with WAL mode and the same pragmas as the messaging system.
// If the database doesn't exist, it is created with the brain_frames schema.
// Optional CacheOption values can configure the cache (e.g., WithEmbedder).
func NewSQLiteSharedCache(dbPath string, opts ...CacheOption) (*SQLiteSharedCache, error) {
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

	cache := &SQLiteSharedCache{db: db}
	for _, opt := range opts {
		opt(cache)
	}
	return cache, nil
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

// BrainFrame represents a frame stored in the brain with its metadata.
type BrainFrame struct {
	Key          string
	Namespace    string
	Value        []byte
	SimHash      int64
	Content      string
	Version      int
	CreatedAt    int64
	UpdatedAt    int64
	ExpiresAt    *int64
	Source       string
	Embedding    []float32 // optional embedding vector
	EmbeddingDim int       // dimension of embedding (0 if none)
	EmbedModel   string    // model that produced the embedding
}

// SearchResult represents a search hit with a relevance score.
type BrainSearchResult struct {
	Frame BrainFrame
	Score float64
	Tier  string // "user" or "project"
}

// PutFrame stores a frame with full metadata.
// If an embedder is configured (via WithEmbedder) and the frame has content
// but no embedding, the embedding is computed automatically. Embedder errors
// are silently ignored — the frame is stored with SimHash only.
func (c *SQLiteSharedCache) PutFrame(f BrainFrame) error {
	now := time.Now().UnixMilli()
	if f.CreatedAt == 0 {
		f.CreatedAt = now
	}
	if f.UpdatedAt == 0 {
		f.UpdatedAt = now
	}

	// Auto-embed: if we have an embedder, content, and no existing embedding
	if c.embedder != nil && f.Content != "" && len(f.Embedding) == 0 {
		if emb, err := c.embedder.Embed(f.Content); err == nil && len(emb) > 0 {
			f.Embedding = emb
			f.EmbeddingDim = len(emb)
			f.EmbedModel = c.embedder.ModelName()
		}
		// Embedder errors silently ignored — falls back to SimHash only
	}

	var embBlob []byte
	if len(f.Embedding) > 0 {
		embBlob = encodeEmbedding(f.Embedding)
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

// PutVector stores a frame with an embedding but no text content.
// Used for machine-to-machine vector communication.
func (c *SQLiteSharedCache) PutVector(key, namespace string, embedding []float32, model string, payload []byte) error {
	f := BrainFrame{
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
func (c *SQLiteSharedCache) SearchBySimHash(namespace string, queryHash int64, limit int) []BrainSearchResult {
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

	var results []BrainSearchResult
	for rows.Next() {
		f := scanBrainFrame(rows)
		if f == nil {
			continue
		}
		dist := hammingDistance64(queryHash, f.SimHash)
		score := 1.0 - float64(dist)/64.0
		results = append(results, BrainSearchResult{Frame: *f, Score: score})
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
func (c *SQLiteSharedCache) SearchByText(query string, namespace string, limit int) []BrainSearchResult {
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

	var results []BrainSearchResult
	for rows.Next() {
		f := scanBrainFrame(rows)
		if f == nil {
			continue
		}
		results = append(results, BrainSearchResult{Frame: *f, Score: 1.0})
	}
	return results
}

// ListRecent returns the most recently updated frames.
func (c *SQLiteSharedCache) ListRecent(namespace string, limit int) []BrainFrame {
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

	var frames []BrainFrame
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

// SearchByEmbedding performs brute-force cosine similarity scan over all frames with embeddings.
// Returns results sorted by cosine similarity descending, key ascending (deterministic).
func (c *SQLiteSharedCache) SearchByEmbedding(queryEmbedding []float32, namespace string, limit int) []BrainSearchResult {
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

	var results []BrainSearchResult
	for rows.Next() {
		f := scanBrainFrame(rows)
		if f == nil || len(f.Embedding) == 0 {
			continue
		}
		score := cosineSimilarityF32(queryEmbedding, f.Embedding)
		results = append(results, BrainSearchResult{Frame: *f, Score: score})
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
func (c *SQLiteSharedCache) SetEmbedder(e Embedder) {
	c.embedder = e
}

// GetEmbedder returns the configured embedder (may be nil).
func (c *SQLiteSharedCache) GetEmbedder() Embedder {
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
		emb, err := c.embedder.Embed(p.content)
		if err != nil || len(emb) == 0 {
			errCount++
			continue
		}

		embBlob := encodeEmbedding(emb)
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

// scanBrainFrame scans a full row (13 columns) into a BrainFrame.
func scanBrainFrame(rows *sql.Rows) *BrainFrame {
	var f BrainFrame
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
		f.Embedding = decodeEmbedding(embBlob)
	}
	return &f
}

// encodeEmbedding serializes a float32 slice to bytes using IEEE 754 little-endian encoding.
func encodeEmbedding(v []float32) []byte {
	buf := make([]byte, len(v)*4)
	for i, f := range v {
		binary.LittleEndian.PutUint32(buf[i*4:], math.Float32bits(f))
	}
	return buf
}

// decodeEmbedding deserializes bytes back to a float32 slice (IEEE 754 little-endian).
func decodeEmbedding(b []byte) []float32 {
	n := len(b) / 4
	v := make([]float32, n)
	for i := 0; i < n; i++ {
		v[i] = math.Float32frombits(binary.LittleEndian.Uint32(b[i*4:]))
	}
	return v
}

// cosineSimilarityF32 computes the cosine similarity between two float32 vectors.
// Returns 0.0 if either vector is zero-length or has zero magnitude.
func cosineSimilarityF32(a, b []float32) float64 {
	if len(a) == 0 || len(b) == 0 {
		return 0.0
	}
	// Use shorter length (handles dimension mismatch gracefully)
	n := len(a)
	if len(b) < n {
		n = len(b)
	}
	var dot, magA, magB float64
	for i := 0; i < n; i++ {
		ai, bi := float64(a[i]), float64(b[i])
		dot += ai * bi
		magA += ai * ai
		magB += bi * bi
	}
	if magA == 0 || magB == 0 {
		return 0.0
	}
	return dot / (math.Sqrt(magA) * math.Sqrt(magB))
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
