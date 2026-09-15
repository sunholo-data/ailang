package repo_test

import (
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"testing"

	"github.com/sunholo-data/ailang/internal/repo"
)

func chdir(t *testing.T, dir string) {
	t.Helper()
	prev, err := os.Getwd()
	if err != nil {
		t.Fatal(err)
	}
	if err := os.Chdir(dir); err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() { _ = os.Chdir(prev) })
}

func TestFindRootWalksUpToTheNearestMarker(t *testing.T) {
	root := t.TempDir()
	// macOS gives /var/... which is a symlink to /private/var/...; compare
	// through EvalSymlinks so the assertion is about the same directory.
	root, _ = filepath.EvalSymlinks(root)
	nested := filepath.Join(root, "a", "b", "c")
	if err := os.MkdirAll(nested, 0o755); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(root, "go.mod"), []byte("module x\n"), 0o644); err != nil {
		t.Fatal(err)
	}
	// A nearer marker of a different kind wins over a farther one.
	if err := os.Mkdir(filepath.Join(root, "a", ".git"), 0o755); err != nil {
		t.Fatal(err)
	}
	chdir(t, nested)

	got, ok := repo.FindRoot("go.mod")
	if !ok || got != root {
		t.Errorf("FindRoot(go.mod) = %q, %v; want %q, true", got, ok, root)
	}
	got, ok = repo.FindRoot("go.mod", ".git")
	if !ok || got != filepath.Join(root, "a") {
		t.Errorf("FindRoot(go.mod, .git) = %q, %v; want %q (nearest), true", got, ok, filepath.Join(root, "a"))
	}
	got, ok = repo.FindRoot("does-not-exist.marker")
	if ok || got != nested {
		t.Errorf("no marker: FindRoot = %q, %v; want the working directory %q, false", got, ok, nested)
	}
}

// repo is a LEAF (stdlib only): core (module) and language roots (prompt)
// both import it.
func TestRepoIsALeaf(t *testing.T) {
	const module = "github.com/sunholo-data/ailang/"
	out, err := exec.Command("go", "list", "-deps", ".").Output()
	if err != nil {
		t.Fatalf("go list -deps: %v", err)
	}
	sawControl := false
	for _, d := range strings.Split(strings.TrimSpace(string(out)), "\n") {
		if d == "path/filepath" {
			sawControl = true
		}
		first, _, _ := strings.Cut(d, "/")
		if strings.Contains(first, ".") && d != module+"internal/repo" {
			t.Errorf("repo must be stdlib-only but depends on %s", d)
		}
	}
	if !sawControl {
		t.Fatal("instrument check failed: path/filepath absent from deps")
	}
}
