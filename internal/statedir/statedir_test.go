package statedir_test

import (
	"path/filepath"
	"testing"

	"github.com/sunholo-data/ailang/internal/statedir"
	"github.com/sunholo-data/ailang/internal/testutil"
)

func TestDirHonoursEnvVar(t *testing.T) {
	testutil.SetHomeDir(t, t.TempDir())
	want := filepath.Join(t.TempDir(), "x")
	t.Setenv(statedir.EnvVar, want)

	got, err := statedir.Dir()
	if err != nil {
		t.Fatalf("Dir: %v", err)
	}
	if got != want {
		t.Fatalf("Dir = %q, want %q", got, want)
	}

	p, err := statedir.Path("coordinator.db")
	if err != nil {
		t.Fatalf("Path: %v", err)
	}
	if p != filepath.Join(want, "coordinator.db") {
		t.Fatalf("Path = %q", p)
	}
}

func TestDirDefaultsUnderHome(t *testing.T) {
	home := t.TempDir()
	testutil.SetHomeDir(t, home)
	t.Setenv(statedir.EnvVar, "")

	got, err := statedir.Dir()
	if err != nil {
		t.Fatalf("Dir: %v", err)
	}
	if want := filepath.Join(home, ".ailang", "state"); got != want {
		t.Fatalf("Dir = %q, want %q", got, want)
	}
	if statedir.UnderHome(home) != got {
		t.Fatalf("UnderHome(%q) = %q, want %q", home, statedir.UnderHome(home), got)
	}
}

func TestDirErrorsWhenHomeUnresolvable(t *testing.T) {
	testutil.SetHomeDir(t, "")
	t.Setenv(statedir.EnvVar, "")

	got, err := statedir.Dir()
	if err == nil {
		t.Fatalf("Dir = %q, want error when HOME and %s are both unset", got, statedir.EnvVar)
	}
	if got != "" {
		t.Fatalf("Dir returned %q alongside an error; a relative fallback is the defect this package removes", got)
	}
	if _, err := statedir.Path("x"); err == nil {
		t.Fatal("Path must fail when Dir fails")
	}
}

func TestProjectIsRootRelativeAndIgnoresEnv(t *testing.T) {
	t.Setenv(statedir.EnvVar, filepath.Join(t.TempDir(), "elsewhere"))
	root := t.TempDir()
	got := statedir.Project(root, "brain.db")
	if want := filepath.Join(root, ".ailang", "state", "brain.db"); got != want {
		t.Fatalf("Project = %q, want %q", got, want)
	}
}
