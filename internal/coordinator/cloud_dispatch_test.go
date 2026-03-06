package coordinator

import (
	"os"
	"testing"
)

func TestDeriveRepoURL(t *testing.T) {
	tests := []struct {
		name      string
		workspace string
		envURL    string // AILANG_REPO_URL env var
		want      string
	}{
		{
			name:      "github org/repo format",
			workspace: "sunholo-data/ailang",
			want:      "https://github.com/sunholo-data/ailang.git",
		},
		{
			name:      "another github repo",
			workspace: "myorg/myrepo",
			want:      "https://github.com/myorg/myrepo.git",
		},
		{
			name:      "local path falls back to env",
			workspace: "/Users/mark/dev/sunholo/ailang",
			envURL:    "https://github.com/sunholo-data/ailang.git",
			want:      "https://github.com/sunholo-data/ailang.git",
		},
		{
			name:      "empty workspace falls back to env",
			workspace: "",
			envURL:    "https://github.com/fallback/repo.git",
			want:      "https://github.com/fallback/repo.git",
		},
		{
			name:      "empty workspace no env returns empty",
			workspace: "",
			want:      "",
		},
		{
			name:      "nested path not github",
			workspace: "a/b/c",
			envURL:    "https://github.com/env/repo.git",
			want:      "https://github.com/env/repo.git",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if tt.envURL != "" {
				os.Setenv("AILANG_REPO_URL", tt.envURL)
				defer os.Unsetenv("AILANG_REPO_URL")
			} else {
				os.Unsetenv("AILANG_REPO_URL")
			}

			got := deriveRepoURL(tt.workspace)
			if got != tt.want {
				t.Errorf("deriveRepoURL(%q) = %q, want %q", tt.workspace, got, tt.want)
			}
		})
	}
}
