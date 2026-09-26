package gitexec

import (
	"context"
	"errors"
	"os"
	"os/exec"
	"path/filepath"
	"runtime"
	"testing"
)

// fakeGit creates a real, executable-looking git at a temp path and returns it.
//
// The name is platform-dependent on purpose. On Windows os/exec resolves an
// absolute path through lookExtensions, which consults PATHEXT and REFUSES a
// suffix-less file at exec.Command time — so a bare "git" makes Cmd.Err
// "executable file not found in %PATH%" and any later assertion about WHY the
// command failed is testing the fixture instead of the code. The file must also
// actually exist for the same reason.
func fakeGit(t *testing.T) string {
	t.Helper()
	name := "git"
	if runtime.GOOS == "windows" {
		name = "git.exe"
	}
	p := filepath.Join(t.TempDir(), name)
	if err := os.WriteFile(p, nil, 0o755); err != nil {
		t.Fatal(err)
	}
	return p
}

func resetForTest(t *testing.T, look func() (string, error)) {
	t.Helper()
	cacheMu.Lock()
	oldLook, oldCache := lookPath, cachedGit
	lookPath, cachedGit = look, ""
	cacheMu.Unlock()
	t.Cleanup(func() {
		cacheMu.Lock()
		lookPath, cachedGit = oldLook, oldCache
		cacheMu.Unlock()
	})
}

func TestResolveWith_LookError(t *testing.T) {
	abs := filepath.Join(t.TempDir(), "git")
	_, err := resolveWith(func() (string, error) { return abs, exec.ErrNotFound })
	if !errors.Is(err, ErrUnresolvable) {
		t.Fatalf("got %v", err)
	}
}

func TestResolveWith_RelativeRefused(t *testing.T) {
	for _, p := range []string{"git", "./git"} {
		_, err := resolveWith(func() (string, error) { return p, nil })
		if !errors.Is(err, ErrUnresolvable) {
			t.Fatalf("%q: got %v", p, err)
		}
	}
}

func TestCommand_DeferredError(t *testing.T) {
	want := errors.New("resolution failed")
	cmd := commandWith(context.Background(), func() (string, error) { return "", want }, "status")
	if err := cmd.Run(); !errors.Is(err, want) {
		t.Fatalf("Run error = %v", err)
	}
}

func TestCommand_UsesAbsolutePath(t *testing.T) {
	p := fakeGit(t)
	cmd := commandWith(context.Background(), func() (string, error) { return p, nil }, "status")
	if cmd.Path != p || cmd.Args[0] != p {
		t.Fatalf("Path/Args = %q/%q, want %q", cmd.Path, cmd.Args[0], p)
	}
	ctx, cancel := context.WithCancel(context.Background())
	cancel()
	cmd = commandWith(ctx, func() (string, error) { return p, nil })
	if err := cmd.Run(); !errors.Is(err, context.Canceled) {
		t.Fatalf("cancelled Run error = %v", err)
	}
}

func TestPath_CachesSuccess(t *testing.T) {
	p, calls := filepath.Join(t.TempDir(), "git"), 0
	resetForTest(t, func() (string, error) { calls++; return p, nil })
	if _, err := Path(); err != nil {
		t.Fatal(err)
	}
	if _, err := Path(); err != nil {
		t.Fatal(err)
	}
	if calls != 1 {
		t.Fatalf("lookup calls = %d, want 1", calls)
	}
}

func TestPath_DoesNotCacheFailure(t *testing.T) {
	p, calls := filepath.Join(t.TempDir(), "git"), 0
	resetForTest(t, func() (string, error) {
		calls++
		if calls == 1 {
			return "", exec.ErrNotFound
		}
		return p, nil
	})
	if _, err := Path(); !errors.Is(err, ErrUnresolvable) {
		t.Fatalf("first error = %v", err)
	}
	if got, err := Path(); err != nil || got != p {
		t.Fatalf("second = %q, %v", got, err)
	}
	if calls != 2 {
		t.Fatalf("lookup calls = %d, want 2", calls)
	}
}
