package messaging

import (
	"crypto/rand"
	"database/sql"
	"fmt"
	"time"

	"github.com/sunholo-data/ailang/internal/statedir"
	"github.com/sunholo-data/ailang/internal/telemetry"
)

// messagingTracer is the OpenTelemetry tracer for messaging instrumentation.
// When no TracerProvider is configured, this returns a no-op tracer with ~2ns overhead.
var messagingTracer = telemetry.Tracer("ailang.messaging")

// Store provides CRUD operations for the collaboration hub database.
type Store struct {
	db *sql.DB
}

// NewStore creates a new store instance from an existing database connection.
func NewStore(db *sql.DB) *Store {
	return &Store{db: db}
}

// OpenStore opens or creates a SQLite database at the given path.
// If dbPath doesn't exist, creates a new database with schema.
// Also applies any pending schema migrations.
func OpenStore(dbPath string) (*Store, error) {
	if dbPath == "" {
		return nil, fmt.Errorf("messaging: no database path: set %s (or HOME)", statedir.EnvVar)
	}
	db, err := InitDB(dbPath)
	if err != nil {
		return nil, err
	}

	// Apply any pending migrations for existing databases
	if err := MigrateDB(db); err != nil {
		db.Close()
		return nil, fmt.Errorf("failed to migrate database: %w", err)
	}

	return &Store{db: db}, nil
}

// Close closes the database connection.
func (s *Store) Close() error {
	return s.db.Close()
}

// DB returns the underlying database connection for advanced operations.
func (s *Store) DB() *sql.DB {
	return s.db
}

// GetDefaultDatabasePath returns the default path for the collaboration
// database: collaboration.db under statedir.Dir(). When no state directory
// resolves it returns "", which OpenStore refuses — never a relative
// .ailang/state beside the process.
func GetDefaultDatabasePath() string {
	p, err := statedir.Path("collaboration.db")
	if err != nil {
		return ""
	}
	return p
}

// generateRandomID generates a random hex string of the given length using crypto/rand
func generateRandomID(length int) string {
	bytes := make([]byte, length/2)
	if _, err := rand.Read(bytes); err != nil {
		// Fallback to timestamp-based pseudo-random
		now := time.Now().UnixNano()
		for i := range bytes {
			bytes[i] = byte(now >> (i * 8))
		}
	}

	return fmt.Sprintf("%x", bytes)
}
