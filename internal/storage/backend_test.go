package storage

import (
	"context"
	"errors"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/sunholo-data/ailang/internal/config"
	"github.com/sunholo-data/ailang/internal/testutil"
)

// clearPlaneEnv isolates a test from the machine's plane variables, the
// retired selectors included.
func clearPlaneEnv(t *testing.T) {
	t.Helper()
	for _, v := range []string{config.EnvStorage, config.EnvStorageMessaging, config.EnvStorageCoordinator, config.EnvStorageObservatory} {
		t.Setenv(v, "")
	}
	for _, v := range config.RemovedEnvNames() {
		t.Setenv(v, "")
	}
}

func TestGetMode(t *testing.T) {
	tests := []struct {
		envValue string
		expected Mode
	}{
		{"", ModeLocal},
		{"local", ModeLocal},
		{"gcp", ModeGCP},
		{"hybrid", ModeHybrid},
		{"unknown", ModeInvalid}, // never local: NewBackends returns the real error
	}

	for _, tt := range tests {
		t.Run(tt.envValue, func(t *testing.T) {
			clearPlaneEnv(t)
			t.Setenv(config.EnvStorage, tt.envValue)
			got := GetMode()
			if got != tt.expected {
				t.Errorf("GetMode() = %q, want %q", got, tt.expected)
			}
		})
	}
}

// A retired selector is a hard error from every constructor that reads the
// environment, and GetMode never reports it as local.
func TestRemovedSelectorIsAHardError(t *testing.T) {
	clearPlaneEnv(t)
	t.Setenv("AILANG_MESSAGES_STORE", "gcp")
	if got := GetMode(); got != ModeInvalid {
		t.Fatalf("GetMode() = %q with a retired selector set, want %q", got, ModeInvalid)
	}
	_, err := NewBackends(context.Background())
	if !errors.Is(err, config.ErrRemovedEnv) {
		t.Fatalf("NewBackends: err = %v, want config.ErrRemovedEnv", err)
	}
	if !strings.Contains(err.Error(), "AILANG_STORAGE_MESSAGING=gcp") {
		t.Fatalf("error must name the replacement: %v", err)
	}
}

// A per-store override moves one store: local plane + messaging=gcp opens
// coordinator and observatory in SQLite and needs a Firestore client only
// for messaging — which, with no project resolvable, is the error.
func TestPerStoreOverrideNeedsAProjectOnlyForThatStore(t *testing.T) {
	clearPlaneEnv(t)
	noCloudProject(t)
	t.Setenv("AILANG_STATE_DIR", t.TempDir())
	t.Setenv(config.EnvStorageMessaging, "gcp")

	_, err := NewBackends(context.Background())
	if !errors.Is(err, config.ErrNoCloudProject) {
		t.Fatalf("err = %v, want config.ErrNoCloudProject", err)
	}
	// The same override with a local value opens everything.
	t.Setenv(config.EnvStorageMessaging, "local")
	b, err := NewBackends(context.Background())
	if err != nil {
		t.Fatal(err)
	}
	defer b.Close()
	if b.Selection.Messaging.Source != config.Source(config.EnvStorageMessaging) || b.Selection.Coordinator.Source != config.SourceDefault {
		t.Fatalf("selection sources not carried: %+v", b.Selection)
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
	clearPlaneEnv(t)
	t.Setenv("AILANG_STATE_DIR", t.TempDir())
	t.Setenv(config.EnvStorage, "local")

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
	clearPlaneEnv(t)
	t.Setenv(config.EnvStorage, "dynamodb")

	ctx := context.Background()
	_, err := NewBackends(ctx)
	if err == nil {
		t.Error("Expected error for unknown storage mode")
	}
}

// noCloudProject isolates a test from every source config.CloudProject reads:
// both env vars, the user's config file (via a fixture home) and the metadata
// server.
func noCloudProject(t *testing.T) {
	t.Helper()
	testutil.SetHomeDir(t, t.TempDir())
	t.Setenv(config.EnvCloudProject, "")
	t.Setenv(config.EnvGoogleCloudProject, "")
	t.Setenv(config.EnvConfigFile, "")
	t.Setenv(config.EnvNoMetadata, "1")
}

func TestNewGCPBackendsRequiresProject(t *testing.T) {
	noCloudProject(t)

	_, err := NewGCPBackends(context.Background())
	if !errors.Is(err, config.ErrNoCloudProject) {
		t.Errorf("err = %v, want config.ErrNoCloudProject when no project resolves", err)
	}
	if _, err := NewGCPBackendsForProject(context.Background(), ""); !errors.Is(err, config.ErrNoCloudProject) {
		t.Errorf("explicit empty project: err = %v, want config.ErrNoCloudProject", err)
	}
}

func TestNewHybridBackendsRequiresProject(t *testing.T) {
	noCloudProject(t)

	_, err := NewHybridBackends(context.Background())
	if !errors.Is(err, config.ErrNoCloudProject) {
		t.Errorf("err = %v, want config.ErrNoCloudProject when no project resolves", err)
	}
}

func TestNewSQLiteBackendsFailsWithoutAnyStateDir(t *testing.T) {
	testutil.SetHomeDir(t, "")
	t.Setenv("AILANG_STATE_DIR", "")

	if _, err := NewSQLiteBackends(); err == nil {
		t.Fatal("expected an error, not a relative .ailang/state fallback, when neither AILANG_STATE_DIR nor HOME resolves")
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
