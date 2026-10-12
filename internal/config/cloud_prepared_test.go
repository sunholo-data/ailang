package config

import (
	"context"
	"errors"
	"os"
	"path/filepath"
	"testing"
)

func TestPrepareCloudProjectResolver_FrozenSources(t *testing.T) {
	t.Setenv(EnvCloudProject, "")
	t.Setenv(EnvGoogleCloudProject, "")
	t.Setenv(EnvNoMetadata, "1")
	file := filepath.Join(t.TempDir(), "config.yaml")
	t.Setenv(EnvConfigFile, file)
	if err := os.WriteFile(file, []byte("pubsub:\n  project_id: selected-file-project\n"), 0600); err != nil {
		t.Fatal(err)
	}
	prepared := PrepareCloudProjectResolver()
	if err := os.WriteFile(file, []byte("pubsub:\n  project_id: changed-file-project\n"), 0600); err != nil {
		t.Fatal(err)
	}
	t.Setenv(EnvCloudProject, "late-env-project")
	if project, err := prepared(context.Background()); err != nil || project != "selected-file-project" {
		t.Fatalf("frozen project=%s err=%v", project, err)
	}
	t.Setenv(EnvCloudProject, "")
	t.Setenv(EnvConfigFile, filepath.Join(t.TempDir(), "missing.yaml"))
	unconfigured := PrepareCloudProjectResolver()
	t.Setenv(EnvCloudProject, "late-default-must-not-win")
	t.Setenv(EnvNoMetadata, "")
	if _, err := unconfigured(context.Background()); !errors.Is(err, ErrNoCloudProject) {
		t.Fatalf("unconfigured selection changed: %v", err)
	}
}
