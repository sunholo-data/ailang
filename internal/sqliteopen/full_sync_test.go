package sqliteopen

import (
	"context"
	"path/filepath"
	"strings"
	"testing"
)

func TestFullSyncOptInOnEveryConnection(t *testing.T) {
	db, err := Open(filepath.Join(t.TempDir(), "durable.db"), Options{FullSync: true, MaxOpenConns: 3})
	if err != nil {
		t.Fatal(err)
	}
	defer db.Close()
	// Hold three separate pooled connections concurrently. A one-off post-open
	// PRAGMA would leave the second and third connection at NORMAL and fail.
	for i := 0; i < 3; i++ {
		conn, err := db.Conn(context.Background())
		if err != nil {
			t.Fatal(err)
		}
		defer conn.Close()
		var mode int
		if err := conn.QueryRowContext(context.Background(), "PRAGMA synchronous").Scan(&mode); err != nil {
			t.Fatal(err)
		}
		if mode != 2 {
			t.Fatalf("connection%d synchronous=%d, want FULL(2)", i, mode)
		}
	}
	if !strings.Contains(DSN("x.db", Options{FullSync: true}), "_synchronous=FULL") {
		t.Fatal("FULL sync not in DSN")
	}
	if !strings.Contains(DSN("x.db", Options{}), "_synchronous=NORMAL") {
		t.Fatal("legacy durability default changed")
	}
}
