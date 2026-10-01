package executor

import (
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/sunholo-data/ailang/internal/policy"
)

// isolatedHome points HOME at a temp dir and, on cleanup, makes the 0555
// policy directories writable again so TempDir can remove them.
func isolatedHome(t *testing.T) {
	t.Helper()
	home := t.TempDir()
	t.Setenv("HOME", home)
	t.Cleanup(func() {
		_ = filepath.Walk(home, func(p string, info os.FileInfo, err error) error {
			if err == nil && info.IsDir() {
				_ = os.Chmod(p, 0o755)
			}
			return nil
		})
	})
}

func writePolicyFile(t *testing.T, body string) string {
	t.Helper()
	p := filepath.Join(t.TempDir(), "lane.toml")
	if err := os.WriteFile(p, []byte(body), 0o644); err != nil {
		t.Fatal(err)
	}
	return p
}

func TestMaterializeRunPolicy(t *testing.T) {
	isolatedHome(t)
	src := writePolicyFile(t, "security_mode = \"restricted\"\nallowed_caps = [\"IO\",\"FS\"]\nfs_sandbox = \"${WORKSPACE}\"\nentry = \"main\"\n")
	wsA, wsB := t.TempDir(), t.TempDir()

	a, err := MaterializeRunPolicy(src, wsA)
	if err != nil {
		t.Fatal(err)
	}
	b, err := MaterializeRunPolicy(src, wsB)
	if err != nil {
		t.Fatal(err)
	}
	if a == b {
		t.Fatalf("two workspaces share one policy file %s — parallel runs would overwrite each other", a)
	}
	// Each resolves to a restricted policy rooted at ITS workspace.
	for _, tc := range []struct{ path, ws string }{{a, wsA}, {b, wsB}} {
		res, _, rerr := policy.LoadResolved(tc.path)
		if rerr != nil {
			t.Fatal(rerr)
		}
		want, _ := filepath.EvalSymlinks(tc.ws)
		if !res.Restricted() || res.Root != want {
			t.Fatalf("policy %s: restricted=%v root=%q, want restricted root %q", tc.path, res.Restricted(), res.Root, want)
		}
		if strings.HasPrefix(tc.path, want) {
			t.Fatalf("policy file %s lies inside the workspace", tc.path)
		}
		if st, _ := os.Stat(tc.path); st.Mode().Perm()&0o222 != 0 {
			t.Fatalf("policy file %s is writable (%v)", tc.path, st.Mode().Perm())
		}
	}
	// Re-materialising the same workspace works (the directory was left 0555).
	if _, err := MaterializeRunPolicy(src, wsA); err != nil {
		t.Fatalf("re-materialising: %v", err)
	}
}

func TestMaterializeRunPolicy_RefusesASharedSandbox(t *testing.T) {
	isolatedHome(t)
	shared := writePolicyFile(t, "security_mode = \"restricted\"\nallowed_caps = [\"IO\",\"FS\"]\nfs_sandbox = \"/tmp/ailang_eval\"\nentry = \"main\"\n")
	if _, err := MaterializeRunPolicy(shared, t.TempDir()); err == nil || !strings.Contains(err.Error(), "${WORKSPACE}") {
		t.Fatalf("a policy with a fixed, shared fs_sandbox was accepted: %v", err)
	}
}
