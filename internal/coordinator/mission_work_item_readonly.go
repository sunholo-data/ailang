package coordinator

import (
	"database/sql"
	"fmt"
	"os"
	"path/filepath"
)

// OpenMissionReadOnlyStore opens an existing runtime database without migrations,
// directory creation or writable SQL access. It reads committed WAL data normally;
// immutable=1 must not be used against a live runtime database.
func OpenMissionReadOnlyStore(path string) (*SQLiteStore, error) {
	if !filepath.IsAbs(path) {
		return nil, fmt.Errorf("runtime database path must be absolute")
	}
	info, err := os.Stat(path)
	if err != nil {
		return nil, fmt.Errorf("open existing runtime database: %w", err)
	}
	if !info.Mode().IsRegular() {
		return nil, fmt.Errorf("runtime database is not a regular file")
	}
	db, err := sql.Open("sqlite3", sqliteFileURI(path, "mode=ro&_query_only=1&_busy_timeout=5000"))
	if err != nil {
		return nil, err
	}
	db.SetMaxOpenConns(1)
	db.SetMaxIdleConns(1)
	if err = db.Ping(); err != nil {
		_ = db.Close()
		return nil, err
	}
	return &SQLiteStore{db: db}, nil
}
