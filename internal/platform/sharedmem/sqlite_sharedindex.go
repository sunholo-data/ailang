package sharedmem

import (
	"database/sql"
	"fmt"

	"github.com/sunholo-data/ailang/internal/effects"
	"github.com/sunholo-data/ailang/internal/sqliteopen"
)

// SQLiteSharedIndex stores namespace-scoped index metadata, independently of
// brain frames. All checked operations return operation-local storage errors.
type SQLiteSharedIndex struct{ db *sql.DB }

var _ effects.PersistentSharedIndex = (*SQLiteSharedIndex)(nil)
var _ effects.CheckedSharedIndex = (*SQLiteSharedIndex)(nil)

// OpenSharedIndex is the platform's registered index opener.
func OpenSharedIndex(path string) (effects.PersistentSharedIndex, error) {
	idx, err := NewSQLiteSharedIndex(path)
	if err != nil {
		return nil, err
	}
	return idx, nil
}

// NewSQLiteSharedIndex opens the shared WAL database and creates its own table.
func NewSQLiteSharedIndex(path string) (*SQLiteSharedIndex, error) {
	db, err := sqliteopen.Open(path, sqliteopen.Options{})
	if err != nil {
		return nil, fmt.Errorf("shared index open: %w", err)
	}
	_, err = db.Exec(`CREATE TABLE IF NOT EXISTS sharedindex_entries (
 namespace TEXT NOT NULL, key TEXT NOT NULL, simhash INTEGER NOT NULL,
 embedding BLOB, embedding_dim INTEGER NOT NULL DEFAULT 0,
 version INTEGER NOT NULL, timestamp INTEGER NOT NULL,
 PRIMARY KEY(namespace,key)); CREATE INDEX IF NOT EXISTS idx_sharedindex_ns ON sharedindex_entries(namespace)`)
	if err != nil {
		db.Close()
		return nil, fmt.Errorf("shared index schema: %w", err)
	}
	return &SQLiteSharedIndex{db: db}, nil
}

// Close releases this handle; other processes and handles remain independent.
func (i *SQLiteSharedIndex) Close() error { return i.db.Close() }

func (i *SQLiteSharedIndex) UpsertChecked(ns, key string, hash, version, ts int64) error {
	return i.UpsertWithEmbeddingChecked(ns, key, hash, nil, version, ts)
}

func (i *SQLiteSharedIndex) UpsertWithEmbeddingChecked(ns, key string, hash int64, embedding []float64, version, ts int64) error {
	v := make([]float32, len(embedding))
	for n, f := range embedding {
		v[n] = float32(f)
	}
	_, err := i.db.Exec(`INSERT INTO sharedindex_entries(namespace,key,simhash,embedding,embedding_dim,version,timestamp)
 VALUES(?,?,?,?,?,?,?) ON CONFLICT(namespace,key) DO UPDATE SET
 simhash=excluded.simhash,embedding=excluded.embedding,embedding_dim=excluded.embedding_dim,
 version=excluded.version,timestamp=excluded.timestamp`, ns, key, hash, effects.EncodeEmbedding(v), len(v), version, ts)
	if err != nil {
		return fmt.Errorf("shared index upsert: %w", err)
	}
	return nil
}

func (i *SQLiteSharedIndex) DeleteChecked(ns, key string) error {
	_, err := i.db.Exec("DELETE FROM sharedindex_entries WHERE namespace=? AND key=?", ns, key)
	if err != nil {
		return fmt.Errorf("shared index delete: %w", err)
	}
	return nil
}

func (i *SQLiteSharedIndex) EntryCountChecked(ns string) (int, error) {
	var n int
	err := i.db.QueryRow("SELECT COUNT(*) FROM sharedindex_entries WHERE namespace=?", ns).Scan(&n)
	if err != nil {
		return 0, fmt.Errorf("shared index count: %w", err)
	}
	return n, nil
}

func (i *SQLiteSharedIndex) NamespacesChecked() ([]string, error) {
	rows, err := i.db.Query("SELECT DISTINCT namespace FROM sharedindex_entries ORDER BY namespace")
	if err != nil {
		return nil, fmt.Errorf("shared index namespaces: %w", err)
	}
	defer rows.Close()
	names := []string{}
	for rows.Next() {
		var name string
		if err := rows.Scan(&name); err != nil {
			return nil, err
		}
		names = append(names, name)
	}
	return names, rows.Err()
}
