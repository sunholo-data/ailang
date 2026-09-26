package main

import (
	"os"
	"path/filepath"
	"runtime"
	"strings"
	"testing"
)

// `ailang sandbox-check` gives the RUNTIME's verdict (fileguard over
// os.Root), not a lexical imitation of it (M-EXECUTOR-POLICY-HARDENING M5).

func sandboxFixture(t *testing.T) (tmp, sandbox string) {
	t.Helper()
	if runtime.GOOS == "windows" {
		t.Skip("symlinks need a privilege on Windows")
	}
	tmp = t.TempDir()
	if real, err := filepath.EvalSymlinks(tmp); err == nil {
		tmp = real
	}
	sandbox = filepath.Join(tmp, "sandbox")
	must := func(err error) {
		t.Helper()
		if err != nil {
			t.Fatal(err)
		}
	}
	must(os.MkdirAll(filepath.Join(sandbox, "sub"), 0o755))
	must(os.WriteFile(filepath.Join(tmp, "marker.txt"), []byte("x"), 0o644))
	must(os.WriteFile(filepath.Join(sandbox, "config.json"), []byte("{}"), 0o644))
	must(os.Symlink("../marker.txt", filepath.Join(sandbox, "link.txt")))
	must(os.Symlink("sub", filepath.Join(sandbox, "oklink")))
	return tmp, sandbox
}

func TestSandboxVerdict_Allows(t *testing.T) {
	tmp, sandbox := sandboxFixture(t)
	for _, p := range []string{"config.json", filepath.Join(sandbox, "config.json"), sandbox, "sub/../config.json", "oklink", "new.txt", "sub/new.txt"} {
		got, err := sandboxVerdict(sandbox, p)
		if err != nil {
			t.Errorf("%s: %v", p, err)
			continue
		}
		if !strings.HasPrefix(got, sandbox) {
			t.Errorf("%s resolved to %q, outside %q", p, got, sandbox)
		}
	}
	_ = tmp
}

func TestSandboxVerdict_Rejects(t *testing.T) {
	tmp, sandbox := sandboxFixture(t)
	for _, p := range []string{"../marker.txt", "..", "link.txt", filepath.Join(tmp, "marker.txt"), "/etc/passwd", filepath.Join(sandbox, "..", "other"), "../new.txt", sandbox + "2/x"} {
		if got, err := sandboxVerdict(sandbox, p); err == nil {
			t.Errorf("%s must be REJECT, got ALLOW → %s", p, got)
		}
	}
}
