package motoko

import (
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func TestRequireProfileInRepo(t *testing.T) {
	repo := t.TempDir()
	if err := os.MkdirAll(filepath.Join(repo, ".motoko", "config", "cloud"), 0o755); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(repo, ".motoko", "config", "cloud", "config.json"), []byte("{}"), 0o644); err != nil {
		t.Fatal(err)
	}
	// A profile DIRECTORY without config.json is still missing as far as motoko is concerned.
	if err := os.MkdirAll(filepath.Join(repo, ".motoko", "config", "empty_dir"), 0o755); err != nil {
		t.Fatal(err)
	}

	tests := []struct {
		name    string
		repo    string
		profile string
		wantErr bool
	}{
		{name: "existing profile", repo: repo, profile: "cloud"},
		{name: "missing profile refused", repo: repo, profile: "ollama_fmt", wantErr: true},
		{name: "profile dir without config.json refused", repo: repo, profile: "empty_dir", wantErr: true},
		{name: "undiscovered repo is not judged", repo: "", profile: "ollama_fmt"},
		{name: "repo without a config tree is not judged", repo: t.TempDir(), profile: "ollama_fmt"},
		{name: "no profile is not judged", repo: repo, profile: ""},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := requireProfileInRepo(tt.repo, tt.profile)
			if (err != nil) != tt.wantErr {
				t.Fatalf("requireProfileInRepo(%q, %q) = %v, wantErr %v", tt.repo, tt.profile, err, tt.wantErr)
			}
			if err != nil && !strings.Contains(err.Error(), tt.profile) {
				t.Fatalf("error does not name the profile: %v", err)
			}
		})
	}
}
