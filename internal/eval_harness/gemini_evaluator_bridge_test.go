package eval_harness

import (
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"testing"
)

// newTempGitRepo creates a temp git repo with one committed file so `git diff`
// has a HEAD to diff against, and returns the worktree path.
func newTempGitRepo(t *testing.T) string {
	t.Helper()
	dir := t.TempDir()
	run := func(args ...string) {
		t.Helper()
		cmd := exec.Command("git", args...)
		cmd.Dir = dir
		cmd.Env = append(os.Environ(),
			"GIT_AUTHOR_NAME=test", "GIT_AUTHOR_EMAIL=test@test",
			"GIT_COMMITTER_NAME=test", "GIT_COMMITTER_EMAIL=test@test",
			"GIT_CONFIG_NOSYSTEM=1",
		)
		if out, err := cmd.CombinedOutput(); err != nil {
			t.Fatalf("git %v failed: %v\n%s", args, err, out)
		}
	}
	run("init", "-q")
	// Seed a committed file so the repo has a HEAD.
	writeFile(t, dir, "seed.txt", "seed\n")
	run("add", "seed.txt")
	run("commit", "-q", "-m", "seed")
	return dir
}

func writeFile(t *testing.T, dir, rel, content string) {
	t.Helper()
	full := filepath.Join(dir, rel)
	if err := os.MkdirAll(filepath.Dir(full), 0o755); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(full, []byte(content), 0o644); err != nil {
		t.Fatal(err)
	}
}

func gitAdd(t *testing.T, dir string, paths ...string) {
	t.Helper()
	args := append([]string{"add"}, paths...)
	cmd := exec.Command("git", args...)
	cmd.Dir = dir
	cmd.Env = append(os.Environ(), "GIT_CONFIG_NOSYSTEM=1")
	if out, err := cmd.CombinedOutput(); err != nil {
		t.Fatalf("git %v failed: %v\n%s", args, err, out)
	}
}

func TestBuildDiffBundle_IncludesDiffAndFiles(t *testing.T) {
	dir := newTempGitRepo(t)
	// Two committed .go files, then modify both (tracked changes).
	writeFile(t, dir, "alpha.go", "package a\n\nvar Alpha = 1\n")
	writeFile(t, dir, "beta.go", "package b\n\nvar Beta = 2\n")
	gitAdd(t, dir, "alpha.go", "beta.go")
	// commit them so they're tracked
	commit(t, dir, "add go files")
	// modify both
	writeFile(t, dir, "alpha.go", "package a\n\nvar Alpha = 100\n")
	writeFile(t, dir, "beta.go", "package b\n\nvar Beta = 200\n")

	b, err := BuildDiffBundle(dir, BundleOptions{})
	if err != nil {
		t.Fatal(err)
	}
	// Unified diff present for both.
	if !strings.Contains(b.Text, "alpha.go") || !strings.Contains(b.Text, "beta.go") {
		t.Fatalf("bundle missing file names:\n%s", b.Text)
	}
	if !strings.Contains(b.Text, "diff --git") {
		t.Fatalf("bundle missing unified diff header:\n%s", b.Text)
	}
	// Full file contents present (the NEW values).
	if !strings.Contains(b.Text, "var Alpha = 100") {
		t.Errorf("bundle missing alpha full contents:\n%s", b.Text)
	}
	if !strings.Contains(b.Text, "var Beta = 200") {
		t.Errorf("bundle missing beta full contents:\n%s", b.Text)
	}
	if b.Truncated {
		t.Errorf("unexpected truncation: dropped=%v", b.DroppedFiles)
	}
}

func TestBuildDiffBundle_IncludesUntrackedNewFiles(t *testing.T) {
	dir := newTempGitRepo(t)
	// A brand-new untracked file — `git diff` alone would MISS this entirely.
	writeFile(t, dir, "newthing.go", "package n\n\nfunc Fresh() int { return 42 }\n")

	b, err := BuildDiffBundle(dir, BundleOptions{})
	if err != nil {
		t.Fatal(err)
	}
	if !strings.Contains(b.Text, "+++ NEW FILE: newthing.go") {
		t.Fatalf("bundle missing NEW FILE header (git diff would have missed it):\n%s", b.Text)
	}
	if !strings.Contains(b.Text, "func Fresh() int { return 42 }") {
		t.Fatalf("bundle missing untracked file contents:\n%s", b.Text)
	}
}

func TestBuildDiffBundle_DropsBinaryAndGenerated(t *testing.T) {
	dir := newTempGitRepo(t)
	// Generated file (*.pb.go) with real text content.
	writeFile(t, dir, "api.pb.go", "package api\n\n// generated\nvar X = 1\n")
	// Binary file: contains a NUL byte in the first bytes.
	writeFile(t, dir, "blob.bin", "abc\x00def-binary-content\n")
	gitAdd(t, dir, "api.pb.go", "blob.bin")

	b, err := BuildDiffBundle(dir, BundleOptions{})
	if err != nil {
		t.Fatal(err)
	}
	if !b.Truncated {
		t.Errorf("expected Truncated=true when dropping binary/generated")
	}
	// Both listed in DroppedFiles.
	joined := strings.Join(b.DroppedFiles, "\n")
	if !strings.Contains(joined, "api.pb.go") || !strings.Contains(joined, "blob.bin") {
		t.Errorf("DroppedFiles missing entries: %v", b.DroppedFiles)
	}
	// LOUD marker present for each.
	if !strings.Contains(b.Text, "=== BUNDLE TRUNCATED: dropped api.pb.go") {
		t.Errorf("missing LOUD marker for generated file:\n%s", b.Text)
	}
	if !strings.Contains(b.Text, "=== BUNDLE TRUNCATED: dropped blob.bin") {
		t.Errorf("missing LOUD marker for binary file:\n%s", b.Text)
	}
	// Their CONTENT must NOT appear as an inlined full-file body.
	if strings.Contains(b.Text, "----- FULL FILE: api.pb.go") {
		t.Errorf("generated file was inlined but should be dropped:\n%s", b.Text)
	}
	if strings.Contains(b.Text, "binary-content") {
		t.Errorf("binary content leaked into bundle:\n%s", b.Text)
	}
}

func TestBuildDiffBundle_TruncatesOverCeiling(t *testing.T) {
	dir := newTempGitRepo(t)
	// One large untracked file that blows a tiny ceiling.
	large := "package big\n\nvar Blob = \"" + strings.Repeat("x", 20*1024) + "\"\n"
	writeFile(t, dir, "big.go", large)
	// A small tracked change too, to prove the diff stays.
	writeFile(t, dir, "seed.txt", "seed-modified\n")

	b, err := BuildDiffBundle(dir, BundleOptions{MaxBytes: 2 * 1024})
	if err != nil {
		t.Fatal(err)
	}
	if !b.Truncated {
		t.Fatalf("expected Truncated=true over ceiling")
	}
	joined := strings.Join(b.DroppedFiles, "\n")
	if !strings.Contains(joined, "big.go") {
		t.Errorf("big.go should be dropped: %v", b.DroppedFiles)
	}
	if !strings.Contains(b.Text, "=== BUNDLE TRUNCATED: dropped big.go") {
		t.Errorf("missing over-ceiling marker:\n%s", firstN(b.Text, 800))
	}
	// The unified diff of the tracked change is STILL present (never dropped).
	if !strings.Contains(b.Text, "seed-modified") {
		t.Errorf("tracked diff was dropped — it must never be:\n%s", firstN(b.Text, 800))
	}
	// The dropped new file KEEPS its NEW FILE header (never silently invisible).
	if !strings.Contains(b.Text, "+++ NEW FILE: big.go") {
		t.Errorf("dropped new file lost its NEW FILE header:\n%s", firstN(b.Text, 800))
	}
	// The huge body must NOT be inlined.
	if strings.Contains(b.Text, strings.Repeat("x", 20*1024)) {
		t.Errorf("over-ceiling body was inlined")
	}
}

func TestBuildDiffBundle_Deterministic(t *testing.T) {
	dir := newTempGitRepo(t)
	writeFile(t, dir, "zeta.go", "package z\nvar Z = 1\n")
	writeFile(t, dir, "alpha.go", "package a\nvar A = 1\n")
	writeFile(t, dir, "mid.go", "package m\nvar M = 1\n")
	// Mixed: one tracked-modified, others untracked.
	writeFile(t, dir, "seed.txt", "changed\n")

	b1, err := BuildDiffBundle(dir, BundleOptions{})
	if err != nil {
		t.Fatal(err)
	}
	b2, err := BuildDiffBundle(dir, BundleOptions{})
	if err != nil {
		t.Fatal(err)
	}
	if b1.Text != b2.Text {
		t.Errorf("BuildDiffBundle not deterministic:\n--- b1 ---\n%s\n--- b2 ---\n%s", b1.Text, b2.Text)
	}
}

// --- helpers ---

func commit(t *testing.T, dir, msg string) {
	t.Helper()
	cmd := exec.Command("git", "commit", "-q", "-m", msg)
	cmd.Dir = dir
	cmd.Env = append(os.Environ(),
		"GIT_AUTHOR_NAME=test", "GIT_AUTHOR_EMAIL=test@test",
		"GIT_COMMITTER_NAME=test", "GIT_COMMITTER_EMAIL=test@test",
		"GIT_CONFIG_NOSYSTEM=1",
	)
	if out, err := cmd.CombinedOutput(); err != nil {
		t.Fatalf("git commit failed: %v\n%s", err, out)
	}
}

func firstN(s string, n int) string {
	if len(s) <= n {
		return s
	}
	return s[:n]
}
