package storage

import (
	"context"
	"os"
	"path/filepath"
	"testing"
)

func TestGetMode(t *testing.T) {
	tests := []struct {
		envValue string
		expected Mode
	}{
		{"", ModeLocal},
		{"local", ModeLocal},
		{"gcp", ModeGCP},
		{"hybrid", ModeHybrid},
		{"unknown", Mode("unknown")}, // passes through, validated by NewBackends
	}

	for _, tt := range tests {
		t.Run(tt.envValue, func(t *testing.T) {
			os.Setenv("AILANG_STORAGE", tt.envValue)
			defer os.Unsetenv("AILANG_STORAGE")

			got := GetMode()
			if got != tt.expected {
				t.Errorf("GetMode() = %q, want %q", got, tt.expected)
			}
		})
	}
}

func TestNewSQLiteBackends(t *testing.T) {
	// Use a temporary directory for test databases
	tmpDir := t.TempDir()
	os.Setenv("AILANG_STATE_DIR", tmpDir)
	defer os.Unsetenv("AILANG_STATE_DIR")

	backends, err := NewSQLiteBackends()
	if err != nil {
		t.Fatalf("NewSQLiteBackends() error: %v", err)
	}
	defer backends.Close()

	if backends.Coordinator == nil {
		t.Error("Coordinator backend is nil")
	}
	if backends.Messaging == nil {
		t.Error("Messaging backend is nil")
	}
	if backends.Observatory == nil {
		t.Error("Observatory backend is nil")
	}

	// Verify database files were created
	for _, dbName := range []string{"coordinator.db", "collaboration.db", "observatory.db"} {
		path := filepath.Join(tmpDir, dbName)
		if _, err := os.Stat(path); os.IsNotExist(err) {
			t.Errorf("Database file %s was not created", dbName)
		}
	}
}

func TestNewBackendsLocal(t *testing.T) {
	tmpDir := t.TempDir()
	os.Setenv("AILANG_STATE_DIR", tmpDir)
	os.Setenv("AILANG_STORAGE", "local")
	defer os.Unsetenv("AILANG_STATE_DIR")
	defer os.Unsetenv("AILANG_STORAGE")

	ctx := context.Background()
	backends, err := NewBackends(ctx)
	if err != nil {
		t.Fatalf("NewBackends() error: %v", err)
	}
	defer backends.Close()

	if backends.Coordinator == nil || backends.Messaging == nil || backends.Observatory == nil {
		t.Error("One or more backends is nil")
	}
}

func TestNewBackendsUnknownMode(t *testing.T) {
	os.Setenv("AILANG_STORAGE", "dynamodb")
	defer os.Unsetenv("AILANG_STORAGE")

	ctx := context.Background()
	_, err := NewBackends(ctx)
	if err == nil {
		t.Error("Expected error for unknown storage mode")
	}
}

func TestNewGCPBackendsRequiresProject(t *testing.T) {
	os.Unsetenv("GOOGLE_CLOUD_PROJECT")

	ctx := context.Background()
	_, err := NewGCPBackends(ctx)
	if err == nil {
		t.Error("Expected error when GOOGLE_CLOUD_PROJECT is not set")
	}
}

func TestNewHybridBackendsRequiresProject(t *testing.T) {
	os.Unsetenv("GOOGLE_CLOUD_PROJECT")

	ctx := context.Background()
	_, err := NewHybridBackends(ctx)
	if err == nil {
		t.Error("Expected error when GOOGLE_CLOUD_PROJECT is not set")
	}
}

func TestBackendsClose(t *testing.T) {
	tmpDir := t.TempDir()
	os.Setenv("AILANG_STATE_DIR", tmpDir)
	defer os.Unsetenv("AILANG_STATE_DIR")

	backends, err := NewSQLiteBackends()
	if err != nil {
		t.Fatalf("NewSQLiteBackends() error: %v", err)
	}

	// Close should not error
	if err := backends.Close(); err != nil {
		t.Errorf("Close() error: %v", err)
	}
}
