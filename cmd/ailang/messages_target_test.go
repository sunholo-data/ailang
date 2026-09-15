package main

import (
	"errors"
	"strings"
	"testing"

	"github.com/sunholo-data/ailang/internal/config"
	"github.com/sunholo-data/ailang/internal/storage"
	"github.com/sunholo-data/ailang/internal/testutil"
)

// clearMessagesEnv isolates a test from the machine: the plane variables,
// the retired selectors, every project source and the config file.
func clearMessagesEnv(t *testing.T) {
	t.Helper()
	testutil.SetHomeDir(t, t.TempDir())
	t.Setenv("AILANG_NO_METADATA", "1")
	for _, k := range []string{config.EnvStorage, config.EnvStorageMessaging, config.EnvStorageCoordinator,
		config.EnvStorageObservatory, "AILANG_CLOUD_PROJECT", "GOOGLE_CLOUD_PROJECT", "AILANG_MESSAGES_PROJECT", "AILANG_CONFIG"} {
		t.Setenv(k, "")
	}
	for _, k := range config.RemovedEnvNames() {
		t.Setenv(k, "")
	}
}

// TestMessagesTarget covers the messaging store selection. The bug this guards
// against: AILANG_STORAGE is a process-wide switch over coordinator + messaging +
// observatory, so the only way to put a machine's inbox on the shared cloud store
// used to be to move its eval banking and coordinator state too. The per-store
// override AILANG_STORAGE_MESSAGING is the scoped form. Every case below is a
// configuration a real machine uses.
func TestMessagesTarget(t *testing.T) {
	tests := []struct {
		name        string
		env         map[string]string
		wantMode    storage.Mode
		wantProject string
	}{
		{
			name:     "unset defaults to local",
			env:      map[string]string{},
			wantMode: storage.ModeLocal,
		},
		{
			name:        "AILANG_STORAGE_MESSAGING reaches cloud while AILANG_STORAGE stays local",
			env:         map[string]string{"AILANG_STORAGE_MESSAGING": "gcp", "AILANG_CLOUD_PROJECT": "ailang-multivac"},
			wantMode:    storage.ModeGCP,
			wantProject: "ailang-multivac",
		},
		{
			name: "AILANG_STORAGE_MESSAGING overrides an explicit AILANG_STORAGE",
			env: map[string]string{
				"AILANG_STORAGE":           "local",
				"AILANG_STORAGE_MESSAGING": "gcp",
				"AILANG_CLOUD_PROJECT":     "ailang-multivac",
			},
			wantMode:    storage.ModeGCP,
			wantProject: "ailang-multivac",
		},
		{
			name: "AILANG_MESSAGES_PROJECT overrides AILANG_CLOUD_PROJECT",
			env: map[string]string{
				"AILANG_STORAGE_MESSAGING": "gcp",
				"AILANG_CLOUD_PROJECT":     "ailang-multivac-dev",
				"AILANG_MESSAGES_PROJECT":  "ailang-multivac",
			},
			wantMode: storage.ModeGCP,
			// The laptop pins AILANG_CLOUD_PROJECT to -dev in ~/.zshenv; without this
			// override, opting into the cloud inbox would silently read the dev graveyard.
			wantProject: "ailang-multivac",
		},
		{
			name:        "AILANG_STORAGE=gcp moves messaging with everything else",
			env:         map[string]string{"AILANG_STORAGE": "gcp", "AILANG_CLOUD_PROJECT": "ailang-multivac"},
			wantMode:    storage.ModeGCP,
			wantProject: "ailang-multivac",
		},
		{
			name:     "hybrid keeps messaging local, matching NewHybridBackends",
			env:      map[string]string{"AILANG_STORAGE": "hybrid", "AILANG_CLOUD_PROJECT": "p"},
			wantMode: storage.ModeHybrid,
		},
		{
			name:     "gcp plane with messaging pinned local",
			env:      map[string]string{"AILANG_STORAGE": "gcp", "AILANG_STORAGE_MESSAGING": "local", "AILANG_CLOUD_PROJECT": "p"},
			wantMode: storage.ModeLocal,
		},
		{
			name:     "explicit local",
			env:      map[string]string{"AILANG_STORAGE_MESSAGING": "local"},
			wantMode: storage.ModeLocal,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			clearMessagesEnv(t)
			for k, v := range tt.env {
				t.Setenv(k, v)
			}

			mode, project := messagesTarget()
			if mode != tt.wantMode {
				t.Errorf("mode = %q, want %q", mode, tt.wantMode)
			}
			if project != tt.wantProject {
				t.Errorf("project = %q, want %q", project, tt.wantProject)
			}
		})
	}
}

// TestMessagesTargetRemovedSelectorIsAHardError: AILANG_MESSAGES_STORE was
// retired by M-V1-SIMPLIFY-S3 M3. A binary that ignored it would read local
// SQLite and exit 0 — the exact failure the session-start hook's stale-binary
// probe exists for — so it is an error naming the replacement.
func TestMessagesTargetRemovedSelectorIsAHardError(t *testing.T) {
	clearMessagesEnv(t)
	t.Setenv("AILANG_MESSAGES_STORE", "gcp")
	t.Setenv("AILANG_CLOUD_PROJECT", "ailang-multivac")

	_, err := resolveMessagesTarget()
	if !errors.Is(err, config.ErrRemovedEnv) {
		t.Fatalf("err = %v, want config.ErrRemovedEnv", err)
	}
	if !strings.Contains(err.Error(), "AILANG_STORAGE_MESSAGING=gcp") {
		t.Fatalf("error must name the replacement: %v", err)
	}
	if _, err := openStore(); !errors.Is(err, config.ErrRemovedEnv) {
		t.Fatalf("openStore: err = %v, want config.ErrRemovedEnv", err)
	}
}

// TestOpenStoreRejectsUnknownMode asserts an unrecognised mode FAILS rather than
// silently falling back to local — reading the wrong store looks exactly like an
// empty inbox, which is how prod feedback stayed invisible for weeks.
func TestOpenStoreRejectsUnknownMode(t *testing.T) {
	clearMessagesEnv(t)
	t.Setenv("AILANG_STORAGE_MESSAGING", "gcs") // plausible typo for "gcp"
	t.Setenv("AILANG_CLOUD_PROJECT", "ailang-multivac")

	if _, err := openStore(); err == nil {
		t.Fatal("openStore() returned nil error for unknown mode 'gcs'; must refuse, not fall back to local")
	}
}

// TestOpenStoreGCPRequiresProject asserts gcp mode without a project is an error
// rather than a Firestore client pointed at nothing.
func TestOpenStoreGCPRequiresProject(t *testing.T) {
	clearMessagesEnv(t)
	noCloudIdentity(t)
	t.Setenv("AILANG_STORAGE_MESSAGING", "gcp")

	if _, err := openStore(); err == nil {
		t.Fatal("openStore() returned nil error for gcp mode with no project set")
	}
}

// TestDescribeMessageStore asserts the non-local store is always named in
// output, with the variable that selected it.
func TestDescribeMessageStore(t *testing.T) {
	clearMessagesEnv(t)
	if got := describeMessageStore(); got != "" {
		t.Errorf("local store should render no banner, got %q", got)
	}

	t.Setenv("AILANG_STORAGE_MESSAGING", "gcp")
	t.Setenv("AILANG_CLOUD_PROJECT", "ailang-multivac")
	got := describeMessageStore()
	if got == "" {
		t.Fatal("cloud store must be named in output; an empty cloud inbox and a wrong-project read are otherwise identical")
	}
	if !strings.Contains(got, "ailang-multivac") {
		t.Errorf("banner must name the project so a dev/prod mixup is visible, got %q", got)
	}
	if !strings.Contains(got, "AILANG_STORAGE_MESSAGING") {
		t.Errorf("banner must name the selector so a stray export is visible, got %q", got)
	}
}
