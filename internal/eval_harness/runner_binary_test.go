package eval_harness

import (
	"context"
	"path/filepath"
	"testing"
)

func TestNewAILANGRunnerWithTask_DefaultBinaryUnderGoTest(t *testing.T) {
	r := NewAILANGRunnerWithTask(context.Background(), "", nil, "", nil)
	// Under `go test`, os.Executable() is `*.test`, so defaultAILANGBinary falls back to "ailang".
	if filepath.Base(r.ailangPath) != "ailang" {
		t.Errorf("expected ailang binary fallback under go test, got %q", r.ailangPath)
	}
	explicit := NewAILANGRunnerWithTask(context.Background(), "/custom/bin/ailang", nil, "", nil)
	if explicit.ailangPath != "/custom/bin/ailang" {
		t.Errorf("explicit ailangPath was overwritten: got %q", explicit.ailangPath)
	}
}
