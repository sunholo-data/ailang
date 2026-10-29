package coordinator

import (
	"path/filepath"
	"testing"
)

func TestAccessApprovalStoreFullSyncKeepsLegacyDefault(t *testing.T) {
	path := filepath.Join(t.TempDir(), "coordinator.db")
	access, err := NewAccessApprovalSQLiteStore(path)
	if err != nil {
		t.Fatal(err)
	}
	defer access.Close()
	legacy, err := NewSQLiteStore(path)
	if err != nil {
		t.Fatal(err)
	}
	defer legacy.Close()
	for _, tc := range []struct {
		name  string
		store *SQLiteStore
		want  int
	}{{"access", access, 2}, {"legacy", legacy, 1}} {
		var mode int
		if err := tc.store.db.QueryRow("PRAGMA synchronous").Scan(&mode); err != nil {
			t.Fatal(err)
		}
		if mode != tc.want {
			t.Fatalf("%s synchronous=%d want%d", tc.name, mode, tc.want)
		}
	}
}
