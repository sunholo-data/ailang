// Package sqliteopen is the ONE recipe for opening a SQLite database in
// AILANG (M-V1-SIMPLIFY-S3 M3, program §2.1).
//
// Before it there were nine openers. Three set WAL and a busy timeout in the
// DSN and capped the pool at one connection; two set the same pragmas by
// Exec after opening (which reaches only the connection that happened to
// run them); one enabled foreign keys and one did not on the SAME file
// (observatory.db); five in cmd/ailang used a bare sql.Open — no WAL, no
// busy timeout, a default pool — and returned nil on error, so a locked or
// missing database printed an empty report.
//
// Open applies, on EVERY pooled connection, through the DSN:
//
//   - journal_mode=WAL, so readers do not block the writer
//   - busy_timeout=5000, so a second process waits instead of failing
//   - foreign_keys=ON, so the schemas' ON DELETE clauses mean something
//   - synchronous=NORMAL, the WAL-safe durability/speed point
//
// and caps the pool at one open connection: SQLite is single-writer, and
// serialising at the pool is cheaper than contending on the file lock. It
// creates the parent directory unless told the file must already exist, and
// pings, so "cannot open" is an error at the call site rather than at the
// first query.
//
// It is a leaf — standard library, database/sql and the cgo driver — so any
// store may import it. Because it links the cgo driver it must never be
// imported by a language-core package (internal/diag's closure test polices
// that); the core's SharedCache seam is implemented in
// internal/platform/sharedmem, which is a platform package.
package sqliteopen

import (
	"database/sql"
	"fmt"
	"net/url"
	"os"
	"path/filepath"
	"runtime"
	"strings"

	_ "github.com/mattn/go-sqlite3" // the one place the driver is registered for stores
)

// Options tune one Open. The zero value is the store recipe.
type Options struct {
	// ReadOnly opens with mode=ro and query_only, without migrations,
	// directory creation or writable SQL. The file must exist.
	ReadOnly bool
	// MustExist refuses to create the file or its directory (mode=rw).
	MustExist bool
	// NoForeignKeys leaves foreign_keys off. Nothing in the repo needs it; it
	// exists so a caller that must read a file with dangling references can
	// say so explicitly rather than by picking a different opener.
	NoForeignKeys bool
	// MaxOpenConns caps the pool; 0 means 1.
	MaxOpenConns int
	// CacheSizeKB sets cache_size in KB (negative pragma form); 0 leaves the
	// SQLite default.
	CacheSizeKB int
}

// BusyTimeoutMS is the lock wait applied to every connection.
const BusyTimeoutMS = 5000

// Open opens the SQLite database at path with the recipe above.
func Open(path string, opts Options) (*sql.DB, error) {
	if path == "" {
		return nil, fmt.Errorf("sqliteopen: empty database path")
	}
	if isVirtual(path) {
		// :memory: (tests) or a caller-built file: URI — nothing on disk to
		// create or check.
	} else if opts.ReadOnly || opts.MustExist {
		info, err := os.Stat(path)
		if err != nil {
			return nil, fmt.Errorf("sqliteopen: open existing database: %w", err)
		}
		if !info.Mode().IsRegular() {
			return nil, fmt.Errorf("sqliteopen: %s is not a regular file", path)
		}
	} else if err := os.MkdirAll(filepath.Dir(path), 0o755); err != nil {
		return nil, fmt.Errorf("sqliteopen: create database directory: %w", err)
	}

	db, err := sql.Open("sqlite3", DSN(path, opts))
	if err != nil {
		return nil, fmt.Errorf("sqliteopen: %s: %w", path, err)
	}
	maxOpen := opts.MaxOpenConns
	if maxOpen <= 0 {
		maxOpen = 1
	}
	db.SetMaxOpenConns(maxOpen)
	db.SetMaxIdleConns(maxOpen)
	db.SetConnMaxLifetime(0)
	if err := db.Ping(); err != nil {
		_ = db.Close()
		return nil, fmt.Errorf("sqliteopen: %s: %w", path, err)
	}
	return db, nil
}

// DSN is the connection string Open uses, exposed so a test can assert on
// the recipe and a caller can see what it got.
func DSN(path string, opts Options) string {
	q := url.Values{}
	q.Set("_busy_timeout", fmt.Sprint(BusyTimeoutMS))
	if opts.ReadOnly {
		q.Set("mode", "ro")
		q.Set("_query_only", "1")
	} else {
		if opts.MustExist {
			q.Set("mode", "rw")
		}
		q.Set("_journal_mode", "WAL")
		q.Set("_synchronous", "NORMAL")
	}
	if !opts.NoForeignKeys {
		q.Set("_foreign_keys", "on")
	}
	if opts.CacheSizeKB > 0 {
		q.Set("_cache_size", fmt.Sprint(-opts.CacheSizeKB))
	}
	if isVirtual(path) {
		sep := "?"
		if strings.Contains(path, "?") {
			sep = "&"
		}
		return path + sep + q.Encode()
	}
	return fileURI(runtime.GOOS, path, q.Encode())
}

// isVirtual reports a DSN that is not an OS path: SQLite's in-memory
// database, or a file: URI the caller already spelled.
func isVirtual(path string) bool {
	return path == ":memory:" || strings.HasPrefix(path, "file:")
}

// fileURI spells path as a file: URI the driver accepts. The OS is a
// parameter so the Windows spelling is testable from any host: a Windows
// path is absolute but has no leading separator ("C:/x"), and the URI
// grammar needs one so the drive letter lands in the path and not the
// authority. The separator swap is gated on Windows because a backslash is a
// legal character in a unix filename.
func fileURI(goos, path, rawQuery string) string {
	p := path
	if goos == "windows" {
		p = strings.ReplaceAll(p, `\`, "/")
		if !strings.HasPrefix(p, "/") {
			p = "/" + p
		}
	} else if !strings.HasPrefix(p, "/") {
		// A relative path would be read as file:///<rel>, i.e. from the root.
		if abs, err := filepath.Abs(p); err == nil {
			p = abs
		} else {
			p = "/" + p
		}
	}
	u := url.URL{Scheme: "file", Path: p, RawQuery: rawQuery}
	return u.String()
}
