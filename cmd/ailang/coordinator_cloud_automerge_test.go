package main

import (
	"strings"
	"testing"

	"github.com/sunholo-data/ailang/internal/coordinator"
)

// The auto-merge scope guard. It runs in EVERY agent's repo, so it cannot carry
// AILANG-repo paths — scope comes from the agent's own declared artifact
// patterns, and a markdown floor sits under all of them.

func matchAll(t *testing.T, patterns []string, files []string) (bool, string) {
	t.Helper()
	for _, f := range files {
		if !strings.HasSuffix(f, ".md") {
			return false, f + " is not markdown"
		}
		if !coordinator.MatchesArtifactPattern(patterns, f) {
			return false, f + " is outside " + strings.Join(patterns, ",")
		}
	}
	return true, ""
}

func TestAutoMergeScope_PerAgentPatterns(t *testing.T) {
	tests := []struct {
		name     string
		patterns []string
		files    []string
		want     bool
		why      string
	}{
		{
			// daneel-writer, in sunholo-data/daneel-memory. The OLD hardcoded
			// design_docs/|changelogs/ rule refused this outright.
			name:     "daneel-writer documents in its own repo",
			patterns: []string{"documents/**/*.md", "sources/**/*.md"},
			files:    []string{"documents/DNL-7-note.md", "sources/DNL-7-1.md"},
			want:     true,
		},
		{
			name:     "design-doc-creator in the ailang repo",
			patterns: []string{"design_docs/**/*.md"},
			files:    []string{"design_docs/planned/m-thing.md"},
			want:     true,
		},
		{
			// Scope: the agent wandered outside what it is declared to produce.
			name:     "a file outside the declaration",
			patterns: []string{"documents/**/*.md"},
			files:    []string{"documents/ok.md", "runlog/secret.md"},
			want:     false,
			why:      "runlog/ is not declared",
		},
		{
			// The FLOOR. `**/*` is a real, legitimate declaration
			// (pkg-sunholo-ailang-parse) — as a sole gate it would merge anything.
			name:     "a wide-open pattern still cannot merge code",
			patterns: []string{"**/*"},
			files:    []string{"internal/coordinator/daemon.go"},
			want:     false,
			why:      "markdown floor holds regardless of scope",
		},
		{
			name:     "wide-open patterns DO allow documents",
			patterns: []string{"**/*"},
			files:    []string{"README.md"},
			want:     true,
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got, reason := matchAll(t, tt.patterns, tt.files)
			if got != tt.want {
				t.Errorf("mergeable = %v, want %v (%s) — %s", got, tt.want, tt.why, reason)
			}
		})
	}
}

func TestArtifactPatternsFromEnv(t *testing.T) {
	t.Setenv("AILANG_ARTIFACT_PATTERNS", "documents/**/*.md\nsources/**/*.md\n")
	got := artifactPatternsFromEnv()
	if len(got) != 2 || got[0] != "documents/**/*.md" || got[1] != "sources/**/*.md" {
		t.Errorf("parsed %v", got)
	}
	// Absent must yield nothing, which the guard treats as "nothing bounds this"
	// and refuses — never as "no restriction".
	t.Setenv("AILANG_ARTIFACT_PATTERNS", "")
	if got := artifactPatternsFromEnv(); len(got) != 0 {
		t.Errorf("empty env must yield no patterns, got %v", got)
	}
}
