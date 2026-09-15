package observatory

import (
	"database/sql"
	"path/filepath"
	"testing"
)

func openerPragmas(t *testing.T, db *sql.DB) map[string]string {
	t.Helper()
	out := map[string]string{}
	for _, p := range []string{"journal_mode", "busy_timeout", "foreign_keys", "synchronous"} {
		var v string
		if err := db.QueryRow("PRAGMA " + p).Scan(&v); err != nil {
			t.Fatalf("PRAGMA %s: %v", p, err)
		}
		out[p] = v
	}
	return out
}

// observatory.db used to be opened two ways with different pool caps and
// foreign-key settings (M-V1-SIMPLIFY-S3 M3). Both paths now go through one
// opener; this pins that they produce identical PRAGMA settings and pool
// limits, and that those are the recipe (WAL, 5s busy timeout, FKs on, one
// connection).
func TestObservatoryOpenersAgree(t *testing.T) {
	dir := t.TempDir()

	store, err := OpenStore(filepath.Join(dir, "a", "observatory.db"))
	if err != nil {
		t.Fatal(err)
	}
	defer store.Close()

	backend, err := NewSQLiteBackendFromPath(filepath.Join(dir, "b", "observatory.db"))
	if err != nil {
		t.Fatal(err)
	}
	defer backend.Close()

	viaStore := openerPragmas(t, store.DB())
	viaBackend := openerPragmas(t, backend.DB())
	for k, v := range viaStore {
		if viaBackend[k] != v {
			t.Errorf("PRAGMA %s: OpenStore=%q NewSQLiteBackendFromPath=%q", k, v, viaBackend[k])
		}
	}
	want := map[string]string{"journal_mode": "wal", "busy_timeout": "5000", "foreign_keys": "1", "synchronous": "1"}
	for k, v := range want {
		if viaStore[k] != v {
			t.Errorf("PRAGMA %s = %q, want %q", k, viaStore[k], v)
		}
	}
	if a, b := store.DB().Stats().MaxOpenConnections, backend.DB().Stats().MaxOpenConnections; a != 1 || b != 1 {
		t.Errorf("MaxOpenConnections: OpenStore=%d NewSQLiteBackendFromPath=%d, want 1 and 1", a, b)
	}

	// Both refuse an empty path the same way.
	if _, err := OpenStore(""); err != errNoDatabasePath {
		t.Errorf("OpenStore(\"\") = %v, want errNoDatabasePath", err)
	}
	if _, err := NewSQLiteBackendFromPath(""); err != errNoDatabasePath {
		t.Errorf("NewSQLiteBackendFromPath(\"\") = %v, want errNoDatabasePath", err)
	}
}
