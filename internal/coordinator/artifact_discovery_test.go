package coordinator

import (
	"os"
	"os/exec"
	"path/filepath"
	"testing"
)

// TestArtifactDiscovery_BasicPatternMatching tests the glob pattern matching.
func TestArtifactDiscovery_BasicPatternMatching(t *testing.T) {
	tests := []struct {
		name    string
		pattern string
		file    string
		want    bool
	}{
		// Simple patterns
		{"exact match", "*.md", "README.md", true},
		{"no match extension", "*.md", "README.txt", false},
		{"nested file match", "*.md", "docs/README.md", true},

		// Recursive patterns
		{"recursive match", "design_docs/**/*.md", "design_docs/planned/v0_6_3/feature.md", true},
		{"recursive no match", "design_docs/**/*.md", "examples/test.md", false},
		{"recursive top level", "design_docs/**/*.md", "design_docs/index.md", true},

		// Directory prefix
		{"prefix match", "design_docs/**", "design_docs/planned/feature.md", true},
		{"prefix no match", "design_docs/**", "examples/test.md", false},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := matchGlob(tt.pattern, tt.file)
			if got != tt.want {
				t.Errorf("matchGlob(%q, %q) = %v, want %v", tt.pattern, tt.file, got, tt.want)
			}
		})
	}
}

// TestArtifactDiscovery_InGitWorktree tests artifact discovery in a real git worktree.
func TestArtifactDiscovery_InGitWorktree(t *testing.T) {
	// Create a temporary directory
	tmpDir, err := os.MkdirTemp("", "artifact-test-*")
	if err != nil {
		t.Fatalf("failed to create temp dir: %v", err)
	}
	defer os.RemoveAll(tmpDir)

	// Initialize git repo
	if err := runGit(tmpDir, "init"); err != nil {
		t.Fatalf("git init failed: %v", err)
	}

	// Create initial commit (required for diff to work)
	if err := runGit(tmpDir, "config", "user.email", "test@example.com"); err != nil {
		t.Fatalf("git config email failed: %v", err)
	}
	if err := runGit(tmpDir, "config", "user.name", "Test User"); err != nil {
		t.Fatalf("git config name failed: %v", err)
	}

	// Create .gitkeep for initial commit
	if err := os.WriteFile(filepath.Join(tmpDir, ".gitkeep"), []byte(""), 0644); err != nil {
		t.Fatalf("failed to write .gitkeep: %v", err)
	}
	if err := runGit(tmpDir, "add", "."); err != nil {
		t.Fatalf("git add failed: %v", err)
	}
	if err := runGit(tmpDir, "commit", "-m", "Initial commit"); err != nil {
		t.Fatalf("git commit failed: %v", err)
	}

	// Create a branch called 'dev' to use as base
	if err := runGit(tmpDir, "checkout", "-b", "dev"); err != nil {
		t.Fatalf("git checkout -b dev failed: %v", err)
	}
	if err := runGit(tmpDir, "checkout", "-b", "feature-branch"); err != nil {
		t.Fatalf("git checkout -b feature-branch failed: %v", err)
	}

	// Create design_docs directory structure
	designDir := filepath.Join(tmpDir, "design_docs", "planned", "v0_6_3")
	if err := os.MkdirAll(designDir, 0755); err != nil {
		t.Fatalf("failed to create design_docs dir: %v", err)
	}

	// Create a design doc (simulating what an agent would create)
	designDoc := filepath.Join(designDir, "m-test-feature.md")
	if err := os.WriteFile(designDoc, []byte("# Test Feature\n\nThis is a test design doc."), 0644); err != nil {
		t.Fatalf("failed to write design doc: %v", err)
	}

	// Stage and commit the design doc
	if err := runGit(tmpDir, "add", "."); err != nil {
		t.Fatalf("git add failed: %v", err)
	}
	if err := runGit(tmpDir, "commit", "-m", "Add design doc"); err != nil {
		t.Fatalf("git commit failed: %v", err)
	}

	// Now test artifact discovery
	ad := NewArtifactDiscovery(tmpDir, []string{"design_docs/**/*.md"})

	files, err := ad.DiscoverChangedFiles()
	if err != nil {
		t.Fatalf("DiscoverChangedFiles failed: %v", err)
	}

	t.Logf("Discovered files: %v", files)

	// Should find the design doc
	if len(files) == 0 {
		t.Error("expected to discover at least one file, got none")
	}

	found := false
	for _, f := range files {
		if f == "design_docs/planned/v0_6_3/m-test-feature.md" {
			found = true
			break
		}
	}
	if !found {
		t.Errorf("expected to find design_docs/planned/v0_6_3/m-test-feature.md in %v", files)
	}

	// Test reading artifact content
	content, err := ad.ReadArtifactContent("design_docs/planned/v0_6_3/m-test-feature.md")
	if err != nil {
		t.Fatalf("ReadArtifactContent failed: %v", err)
	}
	if content != "# Test Feature\n\nThis is a test design doc." {
		t.Errorf("unexpected content: %q", content)
	}
}

// TestArtifactDiscovery_MultiplePatterns tests matching against multiple patterns.
func TestArtifactDiscovery_MultiplePatterns(t *testing.T) {
	ad := &ArtifactDiscovery{
		Patterns: []string{"design_docs/**/*.md", "**/*.go", "**/*.json"},
	}

	tests := []struct {
		file string
		want bool
	}{
		{"design_docs/planned/feature.md", true},
		{"internal/pkg/main.go", true},
		{"config.json", true},
		{"image.png", false},
		{"README.txt", false},
	}

	for _, tt := range tests {
		t.Run(tt.file, func(t *testing.T) {
			got := ad.matchesAnyPattern(tt.file)
			if got != tt.want {
				t.Errorf("matchesAnyPattern(%q) = %v, want %v", tt.file, got, tt.want)
			}
		})
	}
}

// TestArtifactDiscovery_DiscoverArtifacts tests the full artifact discovery with content.
func TestArtifactDiscovery_DiscoverArtifacts(t *testing.T) {
	// Create a temporary directory
	tmpDir, err := os.MkdirTemp("", "artifact-content-test-*")
	if err != nil {
		t.Fatalf("failed to create temp dir: %v", err)
	}
	defer os.RemoveAll(tmpDir)

	// Initialize git repo
	if err := runGit(tmpDir, "init"); err != nil {
		t.Fatalf("git init failed: %v", err)
	}
	if err := runGit(tmpDir, "config", "user.email", "test@example.com"); err != nil {
		t.Fatalf("git config email failed: %v", err)
	}
	if err := runGit(tmpDir, "config", "user.name", "Test User"); err != nil {
		t.Fatalf("git config name failed: %v", err)
	}

	// Create initial commit
	if err := os.WriteFile(filepath.Join(tmpDir, ".gitkeep"), []byte(""), 0644); err != nil {
		t.Fatalf("failed to write .gitkeep: %v", err)
	}
	if err := runGit(tmpDir, "add", "."); err != nil {
		t.Fatalf("git add failed: %v", err)
	}
	if err := runGit(tmpDir, "commit", "-m", "Initial commit"); err != nil {
		t.Fatalf("git commit failed: %v", err)
	}

	// Create dev branch and feature branch
	if err := runGit(tmpDir, "checkout", "-b", "dev"); err != nil {
		t.Fatalf("git checkout -b dev failed: %v", err)
	}
	if err := runGit(tmpDir, "checkout", "-b", "task-123"); err != nil {
		t.Fatalf("git checkout -b task-123 failed: %v", err)
	}

	// Create files of different sizes
	docsDir := filepath.Join(tmpDir, "docs")
	if err := os.MkdirAll(docsDir, 0755); err != nil {
		t.Fatalf("failed to create docs dir: %v", err)
	}

	// Small file (should be included)
	smallContent := "# Small Doc"
	if err := os.WriteFile(filepath.Join(docsDir, "small.md"), []byte(smallContent), 0644); err != nil {
		t.Fatalf("failed to write small.md: %v", err)
	}

	// Large file (should be excluded with maxSize)
	largeContent := make([]byte, 2000)
	for i := range largeContent {
		largeContent[i] = 'x'
	}
	if err := os.WriteFile(filepath.Join(docsDir, "large.md"), largeContent, 0644); err != nil {
		t.Fatalf("failed to write large.md: %v", err)
	}

	// Stage and commit
	if err := runGit(tmpDir, "add", "."); err != nil {
		t.Fatalf("git add failed: %v", err)
	}
	if err := runGit(tmpDir, "commit", "-m", "Add docs"); err != nil {
		t.Fatalf("git commit failed: %v", err)
	}

	// Test with size limit
	ad := NewArtifactDiscovery(tmpDir, []string{"**/*.md"})
	artifacts, err := ad.DiscoverArtifacts(1000) // 1KB limit
	if err != nil {
		t.Fatalf("DiscoverArtifacts failed: %v", err)
	}

	t.Logf("Discovered artifacts: %v", artifacts)

	// Should have small.md but not large.md
	if _, ok := artifacts["docs/small.md"]; !ok {
		t.Error("expected to find docs/small.md")
	}
	if _, ok := artifacts["docs/large.md"]; ok {
		t.Error("expected large.md to be excluded (over size limit)")
	}
}

// runGit runs a git command in the specified directory.
func runGit(dir string, args ...string) error {
	cmd := exec.Command("git", args...)
	cmd.Dir = dir
	output, err := cmd.CombinedOutput()
	if err != nil {
		return &gitError{cmd: args, output: output, err: err}
	}
	return nil
}

type gitError struct {
	cmd    []string
	output []byte
	err    error
}

func (e *gitError) Error() string {
	return e.err.Error() + ": " + string(e.output)
}

// The approval card must describe the branch, and the branch never carries
// scratch: the wrapper's commit excludes ScratchDirName by pathspec while
// getChangedFiles reports untracked files too.
//
// Measured 2026-09-17, task-e0d87876: the card listed six files — two real ones
// and four `.ailang-scratch/*.ail` probes — against a PR carrying exactly the
// two. DecidePR then refuses that merge on evidence that was wrong.
func TestDropScratchPaths(t *testing.T) {
	in := []string{
		"Dockerfile",
		".ailang-scratch/gitstatus.ail",
		".github/workflows/ci.yml",
		"pkg/a/.ailang-scratch/probe.ail",
		// A real file whose NAME merely contains the marker must survive: this
		// is a segment test, not a substring test.
		"docs/notes.ailang-scratch.md",
	}
	got := dropScratchPaths(in)
	want := []string{"Dockerfile", ".github/workflows/ci.yml", "docs/notes.ailang-scratch.md"}
	if len(got) != len(want) {
		t.Fatalf("got %v, want %v", got, want)
	}
	for i := range want {
		if got[i] != want[i] {
			t.Errorf("got[%d] = %q, want %q", i, got[i], want[i])
		}
	}
	if isScratchPath("scratch/probe.ail") {
		t.Error("only .ailang-scratch is excluded; a repo's own scratch/ dir is the agent's problem, not a silent drop")
	}
}

// The same rule through the real path: an untracked scratch probe in a git
// worktree must not reach the changed-file list.
//
// Asserting on dropScratchPaths alone is not enough — deleting the call from
// DiscoverChangedFiles leaves that test green, which is how the card and the
// branch came apart in the first place.
func TestDiscoverChangedFiles_ExcludesUntrackedScratch(t *testing.T) {
	dir := t.TempDir()
	for _, args := range [][]string{
		{"init"}, {"config", "user.email", "t@example.com"}, {"config", "user.name", "T"},
	} {
		if err := runGit(dir, args...); err != nil {
			t.Fatalf("git %v: %v", args, err)
		}
	}
	if err := os.WriteFile(filepath.Join(dir, ".gitkeep"), []byte(""), 0o644); err != nil {
		t.Fatal(err)
	}
	if err := runGit(dir, "add", "."); err != nil {
		t.Fatal(err)
	}
	if err := runGit(dir, "commit", "-m", "init"); err != nil {
		t.Fatal(err)
	}

	// What the agent leaves behind: one real edit, one scratch probe, both
	// untracked — exactly the shape of task-e0d87876.
	if err := os.WriteFile(filepath.Join(dir, "Dockerfile"), []byte("FROM scratch\n"), 0o644); err != nil {
		t.Fatal(err)
	}
	if err := os.MkdirAll(filepath.Join(dir, ScratchDirName), 0o755); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(dir, ScratchDirName, "probe.ail"), []byte("module p\n"), 0o644); err != nil {
		t.Fatal(err)
	}

	files, err := NewArtifactDiscovery(dir, nil).DiscoverChangedFiles()
	if err != nil {
		t.Fatalf("discover: %v", err)
	}
	var sawDockerfile bool
	for _, f := range files {
		if isScratchPath(f) {
			t.Errorf("%q reached the changed-file list — the wrapper's commit excludes it, so the approval card would name a file the branch cannot carry", f)
		}
		if f == "Dockerfile" {
			sawDockerfile = true
		}
	}
	if !sawDockerfile {
		t.Errorf("the real change is missing from %v — the filter must drop scratch, not the work", files)
	}
}
