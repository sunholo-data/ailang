//go:build !js

package effects

import (
	"os"
	"path/filepath"
	"strings"
	"testing"
	"time"

	"github.com/sunholo-data/ailang/internal/eval"
)

// M-EXECUTOR-POLICY-HARDENING M3 (AC7): the coarse Stream label must not
// authorize a process source. On the baseline ProcessContext.Authorize on a
// NIL context (Process never granted) fell through to PATH lookup, so a
// program holding only Stream spawned arbitrary commands.
func TestStreamAsyncExecProcess_RequiresProcessCapability(t *testing.T) {
	marker := filepath.Join(t.TempDir(), "spawned")
	ctx := NewEffContext(nil)
	ctx.Grant(NewCapability("Stream"))
	ctx.Stream = NewStreamContext()
	t.Cleanup(ctx.Stream.CloseAll)
	// ctx.Process deliberately nil: Process was never granted.

	res, err := Call(ctx, "Stream", "asyncExecProcess", []eval.Value{
		&eval.StringValue{Value: "touch"},
		&eval.ListValue{Elements: []eval.Value{&eval.StringValue{Value: marker}}},
		&eval.StringValue{Value: "src"},
		&eval.IntValue{Value: 1},
		&eval.IntValue{Value: 64},
	})
	if err == nil {
		t.Errorf("CAPABILITY FAILURE: asyncExecProcess without Process returned %v", res)
	} else if !strings.Contains(err.Error(), "Process") {
		t.Errorf("denial must name the Process capability, got %v", err)
	}
	// The command must not have run.
	deadline := time.Now().Add(500 * time.Millisecond)
	for time.Now().Before(deadline) {
		if _, statErr := os.Stat(marker); statErr == nil {
			t.Fatal("CAPABILITY FAILURE: the subprocess ran without the Process capability")
		}
		time.Sleep(20 * time.Millisecond)
	}
}

// Authorize on a nil context is a denial, never a PATH lookup.
func TestProcessAuthorize_NilContextDenies(t *testing.T) {
	var pc *ProcessContext
	if path, denial := pc.Authorize("echo", nil); denial == nil {
		t.Fatalf("nil ProcessContext authorized %q", path)
	}
}
