package messaging

import (
	"crypto/rand"
	"database/sql"
	"fmt"
	"os"
	"path/filepath"
	"time"
)

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
func OpenStore(dbPath string) (*Store, error) {
	db, err := InitDB(dbPath)
	if err != nil {
		return nil, err
	}
	return &Store{db: db}, nil
}

// Close closes the database connection.
func (s *Store) Close() error {
	return s.db.Close()
}

// DatabaseExists checks if the collaboration database exists at the given path.
func DatabaseExists(dbPath string) bool {
	if _, err := os.Stat(dbPath); os.IsNotExist(err) {
		return false
	}

	// Verify it's a valid SQLite database with our schema
	db, err := sql.Open("sqlite3", dbPath)
	if err != nil {
		return false
	}
	defer db.Close()

	// Check if schema_version table exists
	var version string
	err = db.QueryRow("SELECT version FROM schema_version LIMIT 1").Scan(&version)
	return err == nil
}

// GetDefaultDatabasePath returns the default path for the collaboration database.
func GetDefaultDatabasePath() string {
	stateDir := os.Getenv("AILANG_STATE_DIR")
	if stateDir == "" {
		homeDir, err := os.UserHomeDir()
		if err != nil {
			stateDir = ".ailang/state"
		} else {
			stateDir = filepath.Join(homeDir, ".ailang", "state")
		}
	}
	return filepath.Join(stateDir, "collaboration.db")
}

// EnsureDatabase creates the database if it doesn't exist.
// Returns true if database was created, false if it already exists.
func EnsureDatabase(dbPath string) (bool, error) {
	// If database already exists, nothing to do
	if DatabaseExists(dbPath) {
		return false, nil
	}

	// Create new database
	db, err := InitDB(dbPath)
	if err != nil {
		return false, fmt.Errorf("failed to initialize database: %w", err)
	}
	db.Close()
	return true, nil
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
