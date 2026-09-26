package embed

import (
	"os"
	"path/filepath"
	"strings"
	"testing"
)

// Root resolution used to be `New(".")` at every call site, and every one of
// them fell back to a Go copy when it resolved nothing — so the rule that
// answered depended on the caller's working directory and nothing said which.
// These pin the resolution itself, because a wrong root must now be an error
// rather than a quiet substitution.

const probeModule = "internal/dashboard_transforms/budget_checker"

func TestProjectRoot_WalksUpFromASubdirectory(t *testing.T) {
	root := t.TempDir()
	mustWriteModule(t, root, probeModule)

	deep := filepath.Join(root, "a", "b", "c")
	if err := os.MkdirAll(deep, 0o755); err != nil {
		t.Fatal(err)
	}
	chdir(t, deep)

	got, err := ProjectRoot(probeModule)
	if err != nil {
		t.Fatalf("ProjectRoot: %v", err)
	}
	// macOS /var is a symlink to /private/var, so compare resolved paths.
	if resolve(t, got) != resolve(t, root) {
		t.Errorf("root = %q, want %q", got, root)
	}
}

func TestProjectRoot_FailsRatherThanReturningAWrongRoot(t *testing.T) {
	empty := t.TempDir()
	chdir(t, empty)

	_, err := ProjectRoot(probeModule)
	if err == nil {
		t.Fatal("a tree with no module must be an error, not a root that silently resolves nothing")
	}
	if !strings.Contains(err.Error(), "AILANG_PROJECT_ROOT") {
		t.Errorf("the error must say how to fix it; got %q", err)
	}
}

func TestProjectRoot_EnvIsVerifiedNotTrusted(t *testing.T) {
	empty := t.TempDir()
	t.Setenv("AILANG_PROJECT_ROOT", empty)
	// Pointing the env at a tree without the module must fail HERE. Accepting it
	// would defer the failure to the first Call, which is exactly where the old
	// code substituted a Go answer.
	if _, err := ProjectRoot(probeModule); err == nil {
		t.Fatal("AILANG_PROJECT_ROOT without the module must be rejected")
	}

	good := t.TempDir()
	mustWriteModule(t, good, probeModule)
	t.Setenv("AILANG_PROJECT_ROOT", good)
	got, err := ProjectRoot(probeModule)
	if err != nil {
		t.Fatalf("ProjectRoot: %v", err)
	}
	if got != good {
		t.Errorf("root = %q, want %q", got, good)
	}
}

func TestProjectRoot_EnvBeatsTheWorkingDirectory(t *testing.T) {
	cwdRoot := t.TempDir()
	mustWriteModule(t, cwdRoot, probeModule)
	envRoot := t.TempDir()
	mustWriteModule(t, envRoot, probeModule)

	chdir(t, cwdRoot)
	t.Setenv("AILANG_PROJECT_ROOT", envRoot)

	got, err := ProjectRoot(probeModule)
	if err != nil {
		t.Fatalf("ProjectRoot: %v", err)
	}
	// The container case: an explicit root must win, or an image that ships its
	// modules at a known path can still be hijacked by whatever cwd it starts in.
	if got != envRoot {
		t.Errorf("root = %q, want the explicit %q", got, envRoot)
	}
}

func mustWriteModule(t *testing.T, root, modulePath string) {
	t.Helper()
	full := filepath.Join(root, filepath.FromSlash(modulePath)+".ail")
	if err := os.MkdirAll(filepath.Dir(full), 0o755); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(full, []byte("-- probe\n"), 0o600); err != nil {
		t.Fatal(err)
	}
}

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

func resolve(t *testing.T, p string) string {
	t.Helper()
	r, err := filepath.EvalSymlinks(p)
	if err != nil {
		return p
	}
	return r
}
