package feedbackgate

import (
	"os"
	"path/filepath"
	"strings"
	"testing"
)

// The embedded program must be the real one (module header, the package
// dependency pinned in the lock) — a stale or empty embed would fail only in
// prod, at first use.
func TestMaterializeShadowProgramWritesTheRealProgram(t *testing.T) {
	dir, err := MaterializeShadowProgram(filepath.Join(t.TempDir(), "sh"))
	if err != nil {
		t.Fatal(err)
	}
	prog, _ := os.ReadFile(filepath.Join(dir, "feedback_shadow.ail"))
	if !strings.Contains(string(prog), "module internal/feedbackgate/shadow/feedback_shadow") ||
		!strings.Contains(string(prog), "import pkg/sunholo/decisions/decide") {
		t.Fatalf("embedded program is not the shadow program: %.120s", prog)
	}
	lock, _ := os.ReadFile(filepath.Join(dir, "ailang.lock"))
	if !strings.Contains(string(lock), `"sunholo/decisions"`) {
		t.Fatalf("embedded lock does not pin sunholo/decisions: %.200s", lock)
	}
	// Idempotent: a second call overwrites, never errors.
	if _, err := MaterializeShadowProgram(dir); err != nil {
		t.Fatal(err)
	}
}
