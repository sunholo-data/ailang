package coordinator

import (
	"fmt"
	"path/filepath"

	"github.com/sunholo-data/ailang/internal/sqliteopen"
)

// OpenMissionReadOnlyStore opens an existing runtime database without migrations,
// directory creation or writable SQL access. It reads committed WAL data normally;
// immutable=1 must not be used against a live runtime database.
func OpenMissionReadOnlyStore(path string) (*SQLiteStore, error) {
	if !filepath.IsAbs(path) {
		return nil, fmt.Errorf("runtime database path must be absolute")
	}
	db, err := sqliteopen.Open(path, sqliteopen.Options{ReadOnly: true, NoForeignKeys: coordinatorDBOptions.NoForeignKeys})
	if err != nil {
		return nil, fmt.Errorf("open existing runtime database: %w", err)
	}
	return &SQLiteStore{db: db}, nil
}
