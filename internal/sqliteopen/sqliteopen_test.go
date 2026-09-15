package sqliteopen

import (
	"database/sql"
	"os"
	"path/filepath"
	"strings"
	"testing"
)

// Pragmas reads the settings the recipe promises, on a fresh connection.
// Exported-by-test so the observatory can prove its two openers agree.
func pragmas(t *testing.T, db *sql.DB) map[string]string {
	t.Helper()
	out := map[string]string{}
	for _, p := range []string{"journal_mode", "busy_timeout", "foreign_keys", "synchronous", "query_only"} {
		var v string
		if err := db.QueryRow("PRAGMA " + p).Scan(&v); err != nil {
			t.Fatalf("PRAGMA %s: %v", p, err)
		}
		out[p] = v
	}
	return out
}

func TestOpenAppliesTheRecipeAndCreatesTheDirectory(t *testing.T) {
	path := filepath.Join(t.TempDir(), "nested", "dir", "test.db")
	db, err := Open(path, Options{})
	if err != nil {
		t.Fatal(err)
	}
	defer db.Close()
	if _, err := os.Stat(path); err != nil {
		t.Fatalf("database file not created: %v", err)
	}
	got := pragmas(t, db)
	want := map[string]string{"journal_mode": "wal", "busy_timeout": "5000", "foreign_keys": "1", "synchronous": "1", "query_only": "0"}
	for k, v := range want {
		if got[k] != v {
			t.Errorf("PRAGMA %s = %q, want %q", k, got[k], v)
		}
	}
	if st := db.Stats(); st.MaxOpenConnections != 1 {
		t.Errorf("MaxOpenConnections = %d, want 1 (single-writer serialisation)", st.MaxOpenConnections)
	}
}

// Positive control for the pragma assertions: a bare sql.Open of a fresh
// file does NOT have WAL or foreign keys, so the test above is measuring the
// recipe and not SQLite's defaults. (busy_timeout=5000 happens to be the
// go-sqlite3 driver's own default, so it is not part of the control; the
// recipe still sets it explicitly so the value is a decision, not a driver
// version.)
func TestBareOpenLacksTheRecipe(t *testing.T) {
	path := filepath.Join(t.TempDir(), "bare.db")
	db, err := sql.Open("sqlite3", path)
	if err != nil {
		t.Fatal(err)
	}
	defer db.Close()
	got := pragmas(t, db)
	if got["journal_mode"] == "wal" || got["foreign_keys"] == "1" {
		t.Fatalf("control failed: a bare open already has the recipe (%v); the assertions in TestOpenAppliesTheRecipe would pass vacuously", got)
	}
}

func TestForeignKeysAreEnforcedOnEveryConnection(t *testing.T) {
	path := filepath.Join(t.TempDir(), "fk.db")
	db, err := Open(path, Options{MaxOpenConns: 4})
	if err != nil {
		t.Fatal(err)
	}
	defer db.Close()
	if _, err := db.Exec(`CREATE TABLE parent(id INTEGER PRIMARY KEY);
		CREATE TABLE child(id INTEGER PRIMARY KEY, parent_id INTEGER REFERENCES parent(id))`); err != nil {
		t.Fatal(err)
	}
	// Several connections in the pool: each must refuse the dangling insert,
	// which a post-open Exec("PRAGMA foreign_keys=ON") would not guarantee.
	for i := 0; i < 8; i++ {
		if _, err := db.Exec(`INSERT INTO child(parent_id) VALUES (999)`); err == nil {
			t.Fatalf("iteration %d: dangling insert succeeded; foreign keys are not on for this connection", i)
		}
	}
	if st := db.Stats(); st.MaxOpenConnections != 4 {
		t.Errorf("MaxOpenConnections = %d, want 4", st.MaxOpenConnections)
	}
}

func TestReadOnlyAndMustExistRefuseToCreate(t *testing.T) {
	missing := filepath.Join(t.TempDir(), "missing", "x.db")
	if _, err := Open(missing, Options{ReadOnly: true}); err == nil {
		t.Fatal("ReadOnly must refuse a missing file")
	}
	if _, err := Open(missing, Options{MustExist: true}); err == nil {
		t.Fatal("MustExist must refuse a missing file")
	}
	if _, err := os.Stat(filepath.Dir(missing)); !os.IsNotExist(err) {
		t.Fatal("neither option may create the directory")
	}

	path := filepath.Join(t.TempDir(), "ro.db")
	w, err := Open(path, Options{})
	if err != nil {
		t.Fatal(err)
	}
	if _, err := w.Exec(`CREATE TABLE t(x)`); err != nil {
		t.Fatal(err)
	}
	w.Close()

	ro, err := Open(path, Options{ReadOnly: true})
	if err != nil {
		t.Fatal(err)
	}
	defer ro.Close()
	if got := pragmas(t, ro); got["query_only"] != "1" || got["busy_timeout"] != "5000" {
		t.Fatalf("read-only pragmas: %v", got)
	}
	if _, err := ro.Exec(`INSERT INTO t VALUES (1)`); err == nil {
		t.Fatal("a read-only open must refuse writes")
	}

	rw, err := Open(path, Options{MustExist: true})
	if err != nil {
		t.Fatal(err)
	}
	defer rw.Close()
	if _, err := rw.Exec(`INSERT INTO t VALUES (1)`); err != nil {
		t.Fatalf("MustExist on an existing file must be writable: %v", err)
	}
}

func TestOpenFailsLoudlyOnAnUnopenableFile(t *testing.T) {
	// A directory where a database file should be.
	dir := t.TempDir()
	if _, err := Open(dir, Options{}); err == nil {
		t.Fatal("opening a directory as a database must fail at Open, not at the first query")
	}
	if _, err := Open("", Options{}); err == nil {
		t.Fatal("an empty path must be an error")
	}
}

// :memory: must stay in memory. The first cut turned it into
// file:///<cwd>/:memory: and every observatory test shared one file in the
// package directory.
func TestMemoryDatabaseIsNotAFile(t *testing.T) {
	db, err := Open(":memory:", Options{})
	if err != nil {
		t.Fatal(err)
	}
	defer db.Close()
	if _, err := os.Stat(":memory:"); !os.IsNotExist(err) {
		t.Fatal("a file named :memory: was created in the working directory")
	}
	if got := pragmas(t, db); got["foreign_keys"] != "1" {
		t.Errorf("the recipe must still apply to an in-memory database: %v", got)
	}
	if dsn := DSN(":memory:", Options{}); !strings.HasPrefix(dsn, ":memory:?") {
		t.Errorf("DSN = %q", dsn)
	}
	if dsn := DSN("file:x.db?cache=shared", Options{}); !strings.HasPrefix(dsn, "file:x.db?cache=shared&") {
		t.Errorf("a caller-built URI must keep its own query: %q", dsn)
	}
}

func TestDSNSpellings(t *testing.T) {
	dsn := DSN("/tmp/a b.db", Options{})
	for _, want := range []string{"file:///tmp/a%20b.db?", "_busy_timeout=5000", "_foreign_keys=on", "_journal_mode=WAL", "_synchronous=NORMAL"} {
		if !strings.Contains(dsn, want) {
			t.Errorf("DSN %q lacks %q", dsn, want)
		}
	}
	ro := DSN("/tmp/a.db", Options{ReadOnly: true, NoForeignKeys: true})
	if !strings.Contains(ro, "mode=ro") || !strings.Contains(ro, "_query_only=1") || strings.Contains(ro, "_journal_mode") || strings.Contains(ro, "_foreign_keys") {
		t.Errorf("read-only DSN: %q", ro)
	}
	if got := fileURI("windows", `C:\Users\x\coordinator.db`, "mode=rw"); got != "file:///C:/Users/x/coordinator.db?mode=rw" {
		t.Errorf("windows URI = %q", got)
	}
	if got := fileURI("darwin", `/a\b/c.db`, ""); got != `file:///a%5Cb/c.db` {
		t.Errorf("a backslash in a unix path must survive: %q", got)
	}
	// The unix spelling is byte-identical to what url.URL produced before the
	// Windows fix moved here from internal/coordinator (2026-09-08): the fleet
	// runs on macOS and Linux, and a portability fix must not change the DSN
	// there.
	if got := fileURI("linux", "/tmp/canary.db", "mode=ro&_query_only=1"); got != "file:///tmp/canary.db?mode=ro&_query_only=1" {
		t.Errorf("unix spelling changed: %q", got)
	}
	// A UNC path keeps its scheme/authority shape rather than gaining a third slash.
	if got := fileURI("windows", `\\server\share\state.db`, ""); !strings.HasPrefix(got, "file://") {
		t.Errorf("UNC path lost its shape: %q", got)
	}
}
