package main

import (
	"strings"
	"testing"

	"github.com/sunholo-data/ailang/internal/storage"
)

// TestMessagesTarget covers the scoped messaging selector. The bug this guards
// against: AILANG_STORAGE is a process-wide switch over coordinator + messaging +
// observatory, so before AILANG_MESSAGES_STORE existed the only way to put a
// machine's inbox on the shared cloud store was to move its eval banking and
// coordinator state too. Every case below is a configuration a real machine uses.
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
			name:        "AILANG_MESSAGES_STORE reaches cloud while AILANG_STORAGE stays local",
			env:         map[string]string{"AILANG_MESSAGES_STORE": "gcp", "AILANG_CLOUD_PROJECT": "ailang-multivac"},
			wantMode:    storage.ModeGCP,
			wantProject: "ailang-multivac",
		},
		{
			name: "AILANG_MESSAGES_STORE overrides AILANG_STORAGE",
			env: map[string]string{
				"AILANG_STORAGE":        "local",
				"AILANG_MESSAGES_STORE": "gcp",
				"AILANG_CLOUD_PROJECT":  "ailang-multivac",
			},
			wantMode:    storage.ModeGCP,
			wantProject: "ailang-multivac",
		},
		{
			name: "AILANG_MESSAGES_PROJECT overrides AILANG_CLOUD_PROJECT",
			env: map[string]string{
				"AILANG_MESSAGES_STORE":   "gcp",
				"AILANG_CLOUD_PROJECT":    "ailang-multivac-dev",
				"AILANG_MESSAGES_PROJECT": "ailang-multivac",
			},
			wantMode: storage.ModeGCP,
			// The laptop pins AILANG_CLOUD_PROJECT to -dev in ~/.zshenv; without this
			// override, opting into the cloud inbox would silently read the dev graveyard.
			wantProject: "ailang-multivac",
		},
		{
			name:        "AILANG_STORAGE=gcp still works (back-compat)",
			env:         map[string]string{"AILANG_STORAGE": "gcp", "AILANG_CLOUD_PROJECT": "ailang-multivac"},
			wantMode:    storage.ModeGCP,
			wantProject: "ailang-multivac",
		},
		{
			name:     "hybrid keeps messaging local, matching NewHybridBackends",
			env:      map[string]string{"AILANG_STORAGE": "hybrid", "AILANG_CLOUD_PROJECT": "p"},
			wantMode: storage.ModeHybrid, wantProject: "p",
		},
		{
			name:     "explicit local",
			env:      map[string]string{"AILANG_MESSAGES_STORE": "local"},
			wantMode: storage.ModeLocal,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			for _, k := range []string{"AILANG_STORAGE", "AILANG_MESSAGES_STORE", "AILANG_CLOUD_PROJECT", "AILANG_MESSAGES_PROJECT"} {
				t.Setenv(k, "")
			}
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

// TestOpenStoreRejectsUnknownMode asserts an unrecognised mode FAILS rather than
// silently falling back to local — reading the wrong store looks exactly like an
// empty inbox, which is how prod feedback stayed invisible for weeks.
func TestOpenStoreRejectsUnknownMode(t *testing.T) {
	t.Setenv("AILANG_MESSAGES_STORE", "gcs") // plausible typo for "gcp"
	t.Setenv("AILANG_CLOUD_PROJECT", "ailang-multivac")

	if _, err := openStore(); err == nil {
		t.Fatal("openStore() returned nil error for unknown mode 'gcs'; must refuse, not fall back to local")
	}
}

// TestOpenStoreGCPRequiresProject asserts gcp mode without a project is an error
// rather than a Firestore client pointed at nothing.
func TestOpenStoreGCPRequiresProject(t *testing.T) {
	t.Setenv("AILANG_MESSAGES_STORE", "gcp")
	t.Setenv("AILANG_CLOUD_PROJECT", "")
	t.Setenv("AILANG_MESSAGES_PROJECT", "")

	if _, err := openStore(); err == nil {
		t.Fatal("openStore() returned nil error for gcp mode with no project set")
	}
}

// TestDescribeMessageStore asserts the non-local store is always named in output.
func TestDescribeMessageStore(t *testing.T) {
	t.Setenv("AILANG_MESSAGES_STORE", "")
	t.Setenv("AILANG_STORAGE", "")
	if got := describeMessageStore(); got != "" {
		t.Errorf("local store should render no banner, got %q", got)
	}

	t.Setenv("AILANG_MESSAGES_STORE", "gcp")
	t.Setenv("AILANG_CLOUD_PROJECT", "ailang-multivac")
	got := describeMessageStore()
	if got == "" {
		t.Fatal("cloud store must be named in output; an empty cloud inbox and a wrong-project read are otherwise identical")
	}
	if !strings.Contains(got, "ailang-multivac") {
		t.Errorf("banner must name the project so a dev/prod mixup is visible, got %q", got)
	}
}
