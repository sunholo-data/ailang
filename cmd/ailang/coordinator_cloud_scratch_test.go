package main

import (
	"context"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"testing"
)

func gitRepo(t *testing.T) string {
	t.Helper()
	dir := t.TempDir()
	for _, args := range [][]string{{"init", "-q"}, {"config", "user.email", "t@t"}, {"config", "user.name", "t"}} {
		if out, err := exec.Command("git", append([]string{"-C", dir}, args...)...).CombinedOutput(); err != nil {
			t.Fatalf("git %v: %v %s", args, err, out)
		}
	}
	if err := os.WriteFile(filepath.Join(dir, "a.txt"), []byte("a\n"), 0o644); err != nil {
		t.Fatal(err)
	}
	exec.Command("git", "-C", dir, "add", "-A").Run()
	exec.Command("git", "-C", dir, "commit", "-q", "-m", "init").Run()
	return dir
}

func staged(t *testing.T, dir string) []string {
	t.Helper()
	out, _ := exec.Command("git", "-C", dir, "diff", "--cached", "--name-only").Output()
	return strings.Fields(string(out))
}

func write(t *testing.T, dir, rel, body string) {
	t.Helper()
	p := filepath.Join(dir, rel)
	if err := os.MkdirAll(filepath.Dir(p), 0o755); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(p, []byte(body), 0o644); err != nil {
		t.Fatal(err)
	}
}

// The wrapper's commit never carries the scratch dir — at the root or nested —
// and a tree whose ONLY changes are scratch stages nothing (so no commit).
func TestStageForCommit_ExcludesScratch(t *testing.T) {
	ctx := context.Background()
	dir := gitRepo(t)

	ok, err := stageForCommit(ctx, dir)
	if err != nil || ok {
		t.Fatalf("clean tree: staged=%v err=%v", ok, err)
	}

	write(t, dir, ScratchDir+"/probe.ail", "module probe\n")
	write(t, dir, "pkg/"+ScratchDir+"/lock", "x")
	ok, err = stageForCommit(ctx, dir)
	if err != nil || ok {
		t.Fatalf("scratch-only tree must stage nothing: staged=%v err=%v files=%v", ok, err, staged(t, dir))
	}

	write(t, dir, "pkg/hello.ail", "module hello\n")
	write(t, dir, "a.txt", "changed\n")
	ok, err = stageForCommit(ctx, dir)
	if err != nil || !ok {
		t.Fatalf("real change must stage: staged=%v err=%v", ok, err)
	}
	got := strings.Join(staged(t, dir), ",")
	if got != "a.txt,pkg/hello.ail" {
		t.Fatalf("staged %q, want a.txt,pkg/hello.ail (no scratch)", got)
	}
}
