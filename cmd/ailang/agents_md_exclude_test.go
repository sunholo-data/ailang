package main

import (
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"testing"
)

// The injected AGENTS.md must never reach a commit.
//
// It is the HARNESS's own instruction file ("AILANG Cloud Agent Instructions"),
// written into the workspace so the agent knows how to behave. It is not agent
// output. Until this fix it was untracked and unignored, so the commit step
// swept it up: four ailang-parse PRs (#26, #27, #28, #30) each carried an
// identical `AGENTS.md +60/-0`, and #27/#30 carried NOTHING ELSE. That made a
// run which produced no work look like a change, which is worse than an empty
// PR — it hides the real defect instead of surfacing it.
//
// The cascade path had already noticed ("AGENTS.md ... just clutters the cascade
// PR") and skipped injection when AILANG_CASCADE_ROOT_PACKAGE was set. That
// protected exactly one caller and left every other task committing the file —
// the symptom patched where it was noticed rather than fixed where it lives
// (CLAUDE.md Principle 3).

func gitInit(t *testing.T) string {
	t.Helper()
	dir := t.TempDir()
	for _, args := range [][]string{
		{"init", "-q"},
		{"config", "user.email", "t@example.com"},
		{"config", "user.name", "t"},
	} {
		cmd := exec.Command("git", append([]string{"-C", dir}, args...)...)
		if out, err := cmd.CombinedOutput(); err != nil {
			t.Fatalf("git %v: %v (%s)", args, err, out)
		}
	}
	return dir
}

func TestInjectedAgentsMD_IsExcludedFromGit(t *testing.T) {
	work := gitInit(t)
	plugin := t.TempDir()
	if err := os.WriteFile(filepath.Join(plugin, "AGENTS.md"), []byte("# harness instructions\n"), 0o644); err != nil {
		t.Fatal(err)
	}

	injectAgentsMD(plugin, work)

	if _, err := os.Stat(filepath.Join(work, "AGENTS.md")); err != nil {
		t.Fatalf("injection did not happen, so the exclusion assertion below is vacuous: %v", err)
	}

	// The property that matters is not "a line was written to a file" — it is
	// that git itself refuses to stage it. Ask git.
	out, err := exec.Command("git", "-C", work, "status", "--porcelain", "--untracked-files=all").Output()
	if err != nil {
		t.Fatalf("git status: %v", err)
	}
	if strings.Contains(string(out), "AGENTS.md") {
		t.Errorf("git still sees AGENTS.md as a change; it would be committed:\n%s", out)
	}

	// Control: a genuine edit MUST still show up, or we have excluded too much
	// and just hidden all the agent's work.
	if err := os.WriteFile(filepath.Join(work, "real_work.go"), []byte("package x\n"), 0o644); err != nil {
		t.Fatal(err)
	}
	out2, _ := exec.Command("git", "-C", work, "status", "--porcelain", "--untracked-files=all").Output()
	if !strings.Contains(string(out2), "real_work.go") {
		t.Errorf("real work is no longer visible to git — the exclusion is too broad:\n%s", out2)
	}
}

// A repo that ships its OWN AGENTS.md keeps it tracked and committable.
func TestExistingAgentsMD_IsNotExcluded(t *testing.T) {
	work := gitInit(t)
	plugin := t.TempDir()
	if err := os.WriteFile(filepath.Join(plugin, "AGENTS.md"), []byte("# harness\n"), 0o644); err != nil {
		t.Fatal(err)
	}
	// The repo already has one, tracked.
	if err := os.WriteFile(filepath.Join(work, "AGENTS.md"), []byte("# the repo's own\n"), 0o644); err != nil {
		t.Fatal(err)
	}

	injectAgentsMD(plugin, work)

	if got, _ := os.ReadFile(filepath.Join(work, "AGENTS.md")); string(got) != "# the repo's own\n" {
		t.Errorf("injection overwrote the repo's own AGENTS.md: %q", got)
	}
	out, _ := exec.Command("git", "-C", work, "status", "--porcelain", "--untracked-files=all").Output()
	if !strings.Contains(string(out), "AGENTS.md") {
		t.Errorf("the repo's OWN AGENTS.md was excluded; only the injected copy may be:\n%s", out)
	}
}

func TestExcludeFromGit_IsIdempotent(t *testing.T) {
	work := gitInit(t)
	for i := 0; i < 3; i++ {
		if err := excludeFromGit(work, "AGENTS.md"); err != nil {
			t.Fatalf("excludeFromGit: %v", err)
		}
	}
	raw, err := os.ReadFile(filepath.Join(work, ".git", "info", "exclude"))
	if err != nil {
		t.Fatal(err)
	}
	if n := strings.Count(string(raw), "\nAGENTS.md\n"); n != 1 {
		t.Errorf("AGENTS.md appears %d times in .git/info/exclude, want 1", n)
	}
}
