package coordinator

import (
	"context"
	"path/filepath"
	"strings"
	"testing"
)

func TestTaskFilterOrderColumn(t *testing.T) {
	for _, tc := range []struct {
		orderBy, want string
		wantErr       bool
	}{
		{"", "created_at", false},
		{"priority", "priority", false},
		{"queued_at", "queued_at", false},
		{"id; DROP TABLE tasks", "", true},
		{"created_at DESC, (SELECT 1)", "", true},
	} {
		got, err := (&TaskFilter{OrderBy: tc.orderBy}).OrderColumn()
		if (err != nil) != tc.wantErr || got != tc.want {
			t.Errorf("OrderColumn(%q) = %q, %v; want %q, err=%t", tc.orderBy, got, err, tc.want, tc.wantErr)
		}
	}
}

// ListTasks writes the order column into SQL text, so an unlisted name must be
// refused before it reaches the query, loudly rather than as a default sort.
func TestSQLiteListTasksRejectsUnknownOrderColumn(t *testing.T) {
	store, err := NewSQLiteStore(filepath.Join(t.TempDir(), "test.db"))
	if err != nil {
		t.Fatalf("failed to create store: %v", err)
	}
	defer store.Close()

	ctx := context.Background()
	if _, err := store.ListTasks(ctx, &TaskFilter{OrderBy: "priority", OrderDesc: true}); err != nil {
		t.Fatalf("ListTasks(priority): %v", err)
	}
	_, err = store.ListTasks(ctx, &TaskFilter{OrderBy: "created_at; DELETE FROM tasks"})
	if err == nil || !strings.Contains(err.Error(), "cannot order by") {
		t.Fatalf("ListTasks(injected order) err = %v, want a cannot-order-by error", err)
	}
}
