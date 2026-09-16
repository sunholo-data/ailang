package executor

import (
	"os"
	"path/filepath"
	"runtime"
	"testing"
)

func TestMaterializeAgentPolicy_ReadOnlyOutsideWorkspace(t *testing.T) {
	if runtime.GOOS == "windows" {
		t.Skip("posix permission bits")
	}
	home := t.TempDir()
	t.Setenv("HOME", home)
	// The 0555 dir is the property under test; give TempDir's cleanup its bits back.
	t.Cleanup(func() { _ = os.Chmod(filepath.Join(home, ".ailang", "agent-policy"), 0o755) })
	ws := filepath.Join(home, "work")
	p, err := MaterializeAgentPolicy("allowed_caps = [\"IO\"]\n", ws)
	if err != nil {
		t.Fatal(err)
	}
	if filepath.Dir(filepath.Dir(p)) == ws || p == "" {
		t.Fatalf("policy path %q must be outside the workspace", p)
	}
	if err := os.WriteFile(p, []byte("allowed_caps = [\"Net\"]\n"), 0o644); err == nil {
		t.Fatal("the materialised policy must not be writable")
	}
	// Re-materialising (a second task in the same container) must succeed.
	if _, err := MaterializeAgentPolicy("allowed_caps = []\n", ws); err != nil {
		t.Fatalf("re-materialise: %v", err)
	}
	if got, _ := MaterializeAgentPolicy("", ws); got != "" {
		t.Fatalf("no content must mean no path (default-deny), got %q", got)
	}
}
