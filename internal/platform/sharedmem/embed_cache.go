package sharedmem

import (
	"database/sql"
	"errors"
	"fmt"
	"time"

	"github.com/sunholo-data/ailang/internal/effects"
	"github.com/sunholo-data/ailang/internal/sqliteopen"
)

// SQLiteEmbeddingCache uses a dedicated table, never SharedMem/brain keys.
type SQLiteEmbeddingCache struct{ db *sql.DB }

func OpenEmbeddingCache(path string) (*SQLiteEmbeddingCache, error) {
	db, err := sqliteopen.Open(path, sqliteopen.Options{})
	if err != nil {
		return nil, err
	}
	_, err = db.Exec(`CREATE TABLE IF NOT EXISTS embed_cache (model TEXT NOT NULL, text_hash TEXT NOT NULL, dim INTEGER NOT NULL, vector BLOB NOT NULL, created_at INTEGER NOT NULL, PRIMARY KEY(model,text_hash))`)
	if err != nil {
		db.Close()
		return nil, fmt.Errorf("embedding cache schema: %w", err)
	}
	return &SQLiteEmbeddingCache{db: db}, nil
}
func (c *SQLiteEmbeddingCache) Get(model, hash string, dimension int) ([]float32, bool, error) {
	var dim int
	var blob []byte
	err := c.db.QueryRow("SELECT dim,vector FROM embed_cache WHERE model=? AND text_hash=?", model, hash).Scan(&dim, &blob)
	if errors.Is(err, sql.ErrNoRows) {
		return nil, false, nil
	}
	if err != nil {
		return nil, false, fmt.Errorf("embedding cache read: %w", err)
	}
	if dim <= 0 || len(blob) != dim*4 || (dimension > 0 && dimension != dim) {
		return nil, false, fmt.Errorf("embedding cache corrupt or mismatched dimensions: stored %d, expected %d, bytes %d", dim, dimension, len(blob))
	}
	v := effects.DecodeEmbedding(blob)
	if err := effects.ValidateEmbedding(v, dim); err != nil {
		return nil, false, err
	}
	return v, true, nil
}
func (c *SQLiteEmbeddingCache) Put(model, hash string, vector []float32) error {
	if err := effects.ValidateEmbedding(vector, len(vector)); err != nil {
		return err
	}
	_, err := c.db.Exec("INSERT INTO embed_cache(model,text_hash,dim,vector,created_at) VALUES(?,?,?,?,?) ON CONFLICT(model,text_hash) DO UPDATE SET dim=excluded.dim,vector=excluded.vector", model, hash, len(vector), effects.EncodeEmbedding(vector), time.Now().UnixMilli())
	return err
}
func (c *SQLiteEmbeddingCache) Close() error { return c.db.Close() }
